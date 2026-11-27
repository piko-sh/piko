// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// shimStepMax bounds the form steps of one recipe.
	shimStepMax = 8

	// shimPageLimit bounds the form document in bytes.
	shimPageLimit = 256 << 10

	// shimQuotaLimit bounds a quota reply body in bytes.
	shimQuotaLimit = 4096

	// shimMismatchPrefix starts the subject of every form shim response mismatch.
	shimMismatchPrefix = "form-shim/"
)

// FormShimReceipt records the initial form page and each submission.
type FormShimReceipt struct {
	// Steps holds the ordered interaction steps.
	Steps []FormShimRead

	// Initial holds the initial rendered state.
	Initial casefile.FormShimState

	// PageStatus is the initial form page's HTTP status.
	PageStatus int

	// FormCount counts rendered forms.
	FormCount int
}

// FormShimRead records a submission response and its resulting state.
type FormShimRead struct {
	// Write holds the observed committed form write.
	Write *casefile.FormShimWrite

	// Name identifies the named operation.
	Name string

	// Code is the public error code.
	Code string

	// After records the state after the interaction.
	After casefile.FormShimState

	// Status is the HTTP response status.
	Status int

	// RetryAdvertised reports whether the reply advertises a retry.
	RetryAdvertised bool
}

// shimHTMLForm holds the fields read from a server-rendered form.
type shimHTMLForm struct {
	// name is the form's data-case identity.
	name string

	// action is the form action URL.
	action string

	// method is the upper-cased form method.
	method string

	// label is the rendered label input value.
	label string

	// ticket is the rendered captcha token value.
	ticket string
}

// shimFormClient holds a form client's transport and CSRF state.
type shimFormClient struct {
	// client is the HTTP transport used for requests.
	client *http.Client

	// base is the application base URL.
	base string

	// csrf is the action CSRF token.
	csrf string

	// ephemeral is the form's ephemeral CSRF token.
	ephemeral string
}

// shimStateWire is the decoded form fixture state, with absent fields left nil.
type shimStateWire struct {
	// Checks counts provider checks.
	Checks *int

	// Writes counts committed writes.
	Writes *int

	// Events holds the fixture event history.
	Events *[]string
}

// shimRefusal is the decoded public refusal envelope.
type shimRefusal struct {
	// Code is the public error code.
	Code string

	// Status is the status carried in the envelope.
	Status int
}

// post submits a form action with issued CSRF tokens.
//
// Takes path (string) which is the form action endpoint.
// Takes args (map[string]string) which supplies action fields in a fresh map. The
// function adds the CSRF ephemeral token to the map.
//
// Returns *http.Response which is the open form-action response whose body belongs to the
// caller.
// Returns error which reports request construction, HTTP transport or response-decoding
// failure, or nil on success.
func (f shimFormClient) post(ctx context.Context, path string, args map[string]string) (*http.Response, error) {
	args["_csrf_ephemeral_token"] = f.ephemeral
	body, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("form-shim/encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, f.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, f.csrf)
	return f.client.Do(request)
}

// read reads the form fixture's independent state endpoint.
//
// Returns casefile.FormShimState which records the independent fixture-state response.
// Returns error which reports an HTTP or state-decoding failure.
func (f shimFormClient) read(ctx context.Context) (state casefile.FormShimState, err error) {
	response, err := f.post(ctx, "/_piko/actions/intake.Read", map[string]string{})
	if err != nil {
		return state, err
	}
	if response.StatusCode != http.StatusOK {
		return state, errors.Join(responseMismatchf("form-shim/read", "read action returned HTTP %d", response.StatusCode), closeResponse(response))
	}

	var wire shimStateWire
	if err := decodeAction(response, &wire); err != nil {
		return state, err
	}
	if wire.Checks == nil || wire.Writes == nil || wire.Events == nil {
		return state, responseMismatchf("form-shim/read-shape", "read action omitted counters or history")
	}
	return casefile.FormShimState{Checks: *wire.Checks, Writes: *wire.Writes, Events: *wire.Events}, nil
}

// readShimForms extracts public form controls from the rendered page.
//
// Takes page (string) which holds the rendered page markup.
//
// Returns []shimHTMLForm which lists shim html form values in their recorded order.
// Returns error which reports invalid or incomplete rendered form controls, or nil on
// success.
func readShimForms(page string) ([]shimHTMLForm, error) {
	document, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return nil, err
	}
	var forms []shimHTMLForm
	for node := range document.Descendants() {
		if node.Type != html.ElementNode || node.Data != "form" || attr(node, "data-case") == "" {
			continue
		}
		form := shimHTMLForm{name: attr(node, "data-case"), action: attr(node, "action"), method: strings.ToUpper(attr(node, "method")), label: "", ticket: ""}
		for input := range node.Descendants() {
			if input.Type == html.ElementNode && input.Data == "input" {
				switch attr(input, "name") {
				case "label":
					form.label = attr(input, "value")
				case "_captcha_token":
					form.ticket = attr(input, "value")
				}
			}
		}
		forms = append(forms, form)
	}
	return forms, nil
}

// observeFormShim runs form submissions and reads their resulting state.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes primaryHTML (string) which holds the initial page markup.
// Takes model (*casefile.FormShimContract) which holds the independent expected
// interaction model.
//
// Returns *FormShimReceipt which records form submissions and their independent resulting
// state.
// Returns error which reports form setup, submission or independent state-read failure,
// or nil on success.
func observeFormShim(ctx context.Context, client *http.Client, base, primaryHTML string, model *casefile.FormShimContract) (*FormShimReceipt, error) {
	receipt := &FormShimReceipt{}
	if _, _, err := pageActionTokens(primaryHTML); err != nil {
		return receipt, err
	}
	if model.Page != "/intake" || len(model.Steps) == 0 || len(model.Steps) > shimStepMax {
		return receipt, responseMismatchf("form-shim/model", "unexpected route or step count")
	}
	page, err := readShimPage(ctx, client, base, model.Page, receipt)
	if err != nil {
		return receipt, err
	}
	token, ephemeral, err := pageActionTokens(page)
	if err != nil {
		return receipt, err
	}
	formClient := shimFormClient{client: client, base: base, csrf: token, ephemeral: ephemeral}
	forms, err := readShimForms(page)
	if err != nil {
		return receipt, err
	}
	receipt.FormCount = len(forms)
	if len(forms) != len(model.Steps) {
		return receipt, responseMismatchf("form-shim/forms", "found %d forms for %d steps", len(forms), len(model.Steps))
	}
	if receipt.Initial, err = formClient.read(ctx); err != nil {
		return receipt, err
	}
	for index, form := range forms {
		step, err := submitShimForm(ctx, formClient, form, &model.Steps[index])
		if err != nil {
			return receipt, err
		}
		receipt.Steps = append(receipt.Steps, step)
	}
	return receipt, nil
}

// readShimPage reads the bounded form document and records its status.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes path (string) which is the form page route.
// Takes receipt (*FormShimReceipt) which receives the page status.
//
// Returns string which holds the form document.
// Returns error which reports a request, read, size or status failure.
func readShimPage(ctx context.Context, client *http.Client, base, path string, receipt *FormShimReceipt) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return "", err
	}
	request.Host = requestHost
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	receipt.PageStatus = response.StatusCode
	page, err := io.ReadAll(io.LimitReader(response.Body, shimPageLimit+1))
	if err := errors.Join(err, closeResponse(response)); err != nil {
		return "", err
	}
	if len(page) > shimPageLimit {
		return "", responseMismatchf("form-shim/page", "form document exceeded 256 KiB")
	}
	if receipt.PageStatus != http.StatusOK {
		return "", responseMismatchf("form-shim/page", "got status %d", receipt.PageStatus)
	}
	return string(page), nil
}

// submitShimForm submits one rendered form and reads the resulting fixture state.
//
// Takes formClient (shimFormClient) which submits actions with the page's tokens.
// Takes form (shimHTMLForm) which is the rendered form.
// Takes expected (*casefile.FormShimStep) which holds the modelled step.
//
// Returns FormShimRead which records the reply and the state after it.
// Returns error which reports a form mismatch, submission or state-read failure.
func submitShimForm(ctx context.Context, formClient shimFormClient, form shimHTMLForm, expected *casefile.FormShimStep) (FormShimRead, error) {
	var step FormShimRead
	if form.name != expected.Name || form.action != "/_piko/actions/intake.Submit" || form.method != http.MethodPost || form.label != expected.Label || form.ticket != expected.Ticket {
		return step, responseMismatchf(shimMismatchPrefix+expected.Name+"/form", "actual form action, input, or ticket differs")
	}
	arguments := map[string]string{"label": form.label}
	if form.ticket != "" {
		arguments["_captcha_token"] = form.ticket
	}
	response, err := formClient.post(ctx, form.action, arguments)
	if err != nil {
		return step, err
	}
	step.Name, step.Status = form.name, response.StatusCode
	switch response.StatusCode {
	case http.StatusOK:
		var write casefile.FormShimWrite
		err = decodeAction(response, &write)
		step.Write = &write
	case http.StatusForbidden:
		var refusal shimRefusal
		err = decodeAction(response, &refusal)
		if err == nil && refusal.Status != response.StatusCode {
			err = responseMismatchf(shimMismatchPrefix+expected.Name+"/status-envelope", "HTTP %d with envelope %d", response.StatusCode, refusal.Status)
		}
		step.Code = refusal.Code
	default:
		step.RetryAdvertised, err = readQuotaResponse(response, expected.Name)
	}
	if err != nil {
		return step, err
	}
	step.After, err = formClient.read(ctx)
	return step, err
}

// readQuotaResponse reads and closes a non-success submission response, checking the
// quota reply's body and media type.
//
// Takes response (*http.Response) which is the submission response whose body is
// consumed.
// Takes name (string) which identifies the step in failure subjects.
//
// Returns bool which reports whether the reply advertises a positive retry delay.
// Returns error which reports a read failure or malformed quota reply, or nil on success.
func readQuotaResponse(response *http.Response, name string) (bool, error) {
	seconds, retryErr := strconv.Atoi(response.Header.Get("Retry-After"))
	advertised := retryErr == nil && seconds > 0
	raw, err := io.ReadAll(io.LimitReader(response.Body, shimQuotaLimit+1))
	if err := errors.Join(err, closeResponse(response)); err != nil {
		return advertised, err
	}
	if response.StatusCode != http.StatusTooManyRequests {
		return advertised, nil
	}
	media, _, mediaErr := mime.ParseMediaType(response.Header.Get(headerContentType))
	if mediaErr != nil || media != "text/plain" || len(raw) > shimQuotaLimit || !strings.Contains(string(raw), "Rate limit exceeded") {
		return advertised, responseMismatchf(shimMismatchPrefix+name+"/quota-response", "unexpected quota response body or media type")
	}
	return advertised, nil
}

// checkFormShim checks form replies and independent state against the model.
//
// Takes want (*casefile.FormShimContract) which holds the independently expected result.
// Takes got (*FormShimReceipt) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormShim(want *casefile.FormShimContract, got *FormShimReceipt) error {
	if got == nil || got.PageStatus != http.StatusOK || got.FormCount != len(want.Steps) || len(got.Steps) != len(want.Steps) {
		return responseMismatchf("form-shim/receipt", "missing page, forms, or complete steps")
	}
	if got.Initial.Checks != 0 || got.Initial.Writes != 0 || len(got.Initial.Events) != 0 {
		return responseMismatchf("form-shim/initial", "expected empty private host state")
	}
	for index, step := range want.Steps {
		if err := checkFormShimStep(step, got.Steps[index]); err != nil {
			return err
		}
	}
	return nil
}

// checkFormShimStep checks one form submission and its resulting state.
//
// Takes want (casefile.FormShimStep) which holds the saved form step.
// Takes got (FormShimRead) which holds the observed form submission.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormShimStep(want casefile.FormShimStep, got FormShimRead) error {
	field := ""
	switch {
	case got.Name != want.Name:
		field = "order"
	case got.Status != want.Status:
		field = "status"
	case got.Code != want.Code:
		field = "code"
	case got.RetryAdvertised != (want.Status == http.StatusTooManyRequests):
		field = "retry"
	case (got.Write == nil) != (want.Write == nil):
		field = "write-presence"
	case got.Write != nil && (got.Write.Label != want.Write.Label || got.Write.Revision != want.Write.Revision):
		field = "write-identity"
	case got.Write != nil && got.Write.ScorePercent != want.Write.ScorePercent:
		field = "score"
	case got.Write != nil && (got.Write.Method != want.Write.Method || got.Write.Path != want.Write.Path):
		field = "request-metadata"
	case got.After.Checks != want.After.Checks:
		field = "provider-count"
	case got.After.Writes != want.After.Writes:
		field = "write-count"
	case !reflect.DeepEqual(got.After.Events, want.After.Events):
		field = "history"
	}
	if field != "" {
		return responseMismatchf(shimMismatchPrefix+want.Name+"/"+field, "form step %s field %s differs: got %+v; expected %+v", want.Name, field, got, want)
	}
	return nil
}

// compareFormShim compares form-client observations across server modes.
//
// Takes a (*FormShimReceipt) which holds the first value to compare.
// Takes b (*FormShimReceipt) which holds the second value to compare.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareFormShim(a, b *FormShimReceipt) error {
	if !reflect.DeepEqual(a, b) {
		return responseMismatchf("form-shim/differential", "compiled and interpreted form observations differ")
	}
	return nil
}
