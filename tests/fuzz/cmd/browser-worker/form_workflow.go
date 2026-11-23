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

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
	"piko.sh/piko/wdk/browser"
)

const (
	// formOpenBudget bounds opening the workflow page and its first observation.
	formOpenBudget = 10 * time.Second

	// formReadyWait bounds the wait for the workflow page's controls.
	formReadyWait = 8 * time.Second

	// formCommandBudget bounds one browser form command.
	formCommandBudget = 5 * time.Second

	// formRevisionWait bounds the wait for a mutation's revision to render.
	formRevisionWait = 4 * time.Second

	// formObservationBudget covers the snapshot and network evidence reads of one step.
	formObservationBudget = 2 * time.Second

	// formStepBudget bounds one complete workflow step including its independent reads.
	formStepBudget = formCommandBudget + formRevisionWait + formObservationBudget + freshFormTimeout + formStoreTimeout

	// formRecipeLimit is the largest accepted encoded workflow recipe in bytes.
	formRecipeLimit = 8192

	// formReceiptLimit is the largest accepted workflow receipt in bytes.
	formReceiptLimit = 256 << 10

	// formRequestLimit is the largest number of retained workflow requests.
	formRequestLimit = 64

	// formResponseLimit is the largest accepted action response body in bytes.
	formResponseLimit = 16 << 10

	// formBodyLimit is the largest submitted body whose fields are recorded.
	formBodyLimit = 32 << 10

	// formValueLimit is the largest recorded multipart value in bytes.
	formValueLimit = 1024

	// formRenderWait bounds the wait for queued component rendering after a command.
	formRenderWait = 2 * time.Second

	// formReady waits for the workflow page's label control, status and partial.
	formReady = `document.querySelector('#workflow-label')?.shadowRoot?.querySelector('#native-label') && ` +
		`document.querySelector('#workflow-status') && document.querySelector('[partial_src]')`

	// formWatchInvalid counts invalid events delivered to the workflow label.
	formWatchInvalid = `(() => {window.__fuzzInvalid=0;document.getElementById('workflow-form').addEventListener('invalid', ` +
		`e=> {if(e.target.id==='workflow-label')window.__fuzzInvalid++},true)})()`

	// formPartialPath reads the path of the page's partial source.
	formPartialPath = "new URL(document.querySelector('[partial_src]').getAttribute('partial_src'),location.href).pathname"

	// formSnapshot reads public form and row state from the browser.
	formSnapshot = `(() => {
	const form=document.querySelector('#workflow-form'), host=document.querySelector('#workflow-label');
	const input=host.shadowRoot.querySelector('#native-label'), status=document.querySelector('#workflow-status');
	const rows=Array.from(document.querySelectorAll('[data-fuzz-task]'), el=>({
		ID:Number(el.dataset.fuzzTask),Label:el.querySelector('[data-task-label]').textContent, Done:el.dataset.done==='true'
	}));
	if(rows.length>16) throw Error('too many workflow rows');
	if (Array.from(document.querySelectorAll('[data-fuzz-task]')).some(el=>!['true','false'].includes(el.dataset.done))) throw Error('missing row completion state');
	const omitted=form.querySelectorAll('input[name="ignored"]');
	const disabledControlPresent=omitted.length===1 && omitted[0].form===form && omitted[0].disabled && omitted[0].value==='omit';
	const result=status.dataset.id ? {ID:Number(status.dataset.id), Label:status.dataset.label, Done:status.dataset.done==='true'} : null;
	if(result && !['true','false'].includes(status.dataset.done)) throw Error('missing action completion state');
	return {Input:input.value,DisabledControlPresent:disabledControlPresent, Valid:host.validity.valid,NativeValid:input.validity.valid,FormValid:form.matches(':valid'),
		Entries:Array.from(new FormData(form), ([Name,Value])=>({Name,Value})),
		Rows:rows,Revision:Number(status.dataset.revision),InvalidEvents:window.__fuzzInvalid,Result:result};
})()`
)

var (
	// errInvalidFormRecipe reports a malformed or oversized workflow recipe.
	errInvalidFormRecipe = errors.New("invalid form workflow recipe")

	// errFormRequestOverflow reports more workflow requests than the receipt retains.
	errFormRequestOverflow = fmt.Errorf("workflow network receipt exceeds %d requests", formRequestLimit)

	// errFormResponseTooLarge reports an action response above the retained limit.
	errFormResponseTooLarge = fmt.Errorf("action response exceeds %d bytes", formResponseLimit)
)

// requests collects the workflow's mutation requests.
type requests struct {
	// index maps network request IDs to receipt entries.
	index map[browser.RequestID]int

	// contentTypes tracks request media types by network ID.
	contentTypes map[browser.RequestID]string

	// origin is the application's origin.
	origin string

	// events holds mutation-request receipts.
	events []casefile.FormRequest

	// mu guards every other field against concurrent browser callbacks.
	mu sync.Mutex

	// overflow reports that the request bound was exceeded.
	overflow bool
}

// observe records the Chromium request and response event.
//
// Takes event (browser.NetworkEvent) which is the observed browser event.
//
// Concurrency: Safe for concurrent callbacks. The mutex guards the state.
func (calls *requests) observe(event browser.NetworkEvent) {
	calls.mu.Lock()
	defer calls.mu.Unlock()
	switch event.Kind {
	case browser.RequestStarted:
		location, err := url.Parse(event.URL)
		workflowPath := strings.HasPrefix(location.Path, "/_piko/actions/") || strings.HasPrefix(location.Path, "/_piko/partials/")
		if err != nil || location.Scheme+"://"+location.Host != calls.origin || !workflowPath {
			return
		}
		if len(calls.events) >= formRequestLimit {
			calls.overflow = true
			return
		}
		calls.index[event.ID] = len(calls.events)
		calls.contentTypes[event.ID] = event.ContentType
		calls.events = append(calls.events, casefile.FormRequest{Method: event.Method, Path: location.Path, Status: 0, InputFields: "", JSONResult: ""})
	case browser.ResponseReceived:
		if index, ok := calls.index[event.ID]; ok {
			calls.events[index].Status = event.Status
		}
	default:
	}
}

// pending moves the collected requests out of the recorder.
//
// Returns []casefile.FormRequest which lists the collected requests in observed order.
// Returns []browser.RequestID which lists each request's network ID at the same index.
// Returns map[browser.RequestID]string which holds each request's media type.
// Returns error which reports an exceeded request bound, or nil on success.
//
// Concurrency: Safe for concurrent use. The mutex protects the state swap.
func (calls *requests) pending() ([]casefile.FormRequest, []browser.RequestID, map[browser.RequestID]string, error) {
	calls.mu.Lock()
	defer calls.mu.Unlock()
	if calls.overflow {
		return nil, nil, nil, errFormRequestOverflow
	}
	out := slices.Clone(calls.events)
	order := make([]browser.RequestID, len(out))
	for id, index := range calls.index {
		order[index] = id
	}
	contentTypes := calls.contentTypes
	calls.contentTypes = make(map[browser.RequestID]string)
	calls.index = make(map[browser.RequestID]int)
	calls.events = nil
	return out, order, contentTypes, nil
}

// take collects completed request evidence and clears the pending workflow requests.
//
// Takes session (*browser.Session) which is the browser tab that made the requests.
//
// Returns []casefile.FormRequest which lists form request values in their recorded order.
// Returns error which reports an oversized request receipt or an unreadable action
// response, or nil on success.
func (calls *requests) take(ctx context.Context, session *browser.Session) ([]casefile.FormRequest, error) {
	out, order, contentTypes, err := calls.pending()
	if err != nil {
		return nil, err
	}
	for index, id := range order {
		if !strings.HasPrefix(out[index].Path, "/_piko/actions/") {
			continue
		}
		post, err := session.RequestPostData(ctx, id)
		if err != nil {
			out[index].InputFields = "unreadable request body: " + err.Error()
		} else {
			out[index].InputFields = workflowFields(contentTypes[id], post)
		}
		body, err := session.ResponseBody(ctx, id)
		if err == nil && len(body) > formResponseLimit {
			err = errFormResponseTooLarge
		}
		if err != nil {
			return out, fmt.Errorf("reading workflow action response: %w", err)
		}
		out[index].JSONResult = string(body)
	}
	return out, nil
}

// formProbe runs the form workflow and writes its observation.
//
// Takes address (string) which is the application URL to visit.
// Takes result (string) which is the receipt output path.
// Takes recipe (string) which is the saved browser recipe.
//
// Returns error which reports an invalid recipe, browser failure or receipt-write
// failure, or nil on success.
func formProbe(ctx context.Context, address, result, recipe string) error {
	workflow, err := decodeFormRecipe(recipe)
	if err != nil {
		return err
	}
	location, err := url.Parse(address)
	if err != nil {
		return err
	}
	coveragePath := os.Getenv(envJSCoverage)
	var record casefile.FormWorkflowObservation
	record.FailedStep = -1
	sessionCtx, session, closeBrowser, openErr := openBrowser(ctx, formWorkflowBudget(workflow, coveragePath != ""))
	if openErr != nil {
		record.DriverError = "opening workflow: " + openErr.Error()
		return writeWorkflowReceipt(result, &record)
	}
	defer closeBrowser()
	if coveragePath != "" {
		if err := startCoverage(sessionCtx, session); err != nil {
			return fmt.Errorf("starting %w: %w", errRuntimeCoverage, err)
		}
	}
	calls := new(requests)
	calls.origin = location.Scheme + "://" + location.Host
	calls.index = make(map[browser.RequestID]int)
	calls.contentTypes = make(map[browser.RequestID]string)
	if err := driveWorkflow(sessionCtx, session, address, workflow, &record, calls); err != nil {
		record.DriverError = sessionFailure(sessionCtx, "", err)
	}
	if err := writeWorkflowReceipt(result, &record); err != nil {
		return err
	}
	if coveragePath != "" {
		if err := captureCoverage(sessionCtx, session, address, coveragePath); err != nil {
			return fmt.Errorf("collecting %w: %w", errRuntimeCoverage, err)
		}
	}
	return nil
}

// decodeFormRecipe decodes and validates a saved form workflow recipe.
//
// Takes recipe (string) which is the saved browser recipe.
//
// Returns *casefile.FormWorkflow which is the decoded workflow.
// Returns error which reports an oversized, malformed or invalid recipe, or nil on
// success.
func decodeFormRecipe(recipe string) (*casefile.FormWorkflow, error) {
	if len(recipe) > formRecipeLimit {
		return nil, fmt.Errorf("%w: exceeds %d bytes", errInvalidFormRecipe, formRecipeLimit)
	}
	workflow := new(casefile.FormWorkflow)
	if err := json.Unmarshal([]byte(recipe), workflow); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidFormRecipe, err)
	}
	if _, err := workflow.Model(); err != nil {
		return nil, err
	}
	return workflow, nil
}

// writeWorkflowReceipt encodes and writes a bounded workflow receipt.
//
// Takes result (string) which is the receipt output path.
// Takes record (*casefile.FormWorkflowObservation) which is the workflow's observation.
//
// Returns error which reports an encoding, size or write failure, or nil on success.
func writeWorkflowReceipt(result string, record *casefile.FormWorkflowObservation) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encoding workflow receipt: %w", err)
	}
	if len(data) > formReceiptLimit {
		return fmt.Errorf("workflow result exceeds %d bytes", formReceiptLimit)
	}
	return writePrivateFile(result, data)
}

// driveWorkflow performs the saved form commands and independent state reads.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to visit.
// Takes workflow (*casefile.FormWorkflow) which holds the saved form workflow.
// Takes record (*casefile.FormWorkflowObservation) which receives the workflow's
// observations.
// Takes calls (*requests) which collects mutation-request evidence.
//
// Returns error which reports the failed browser command or independent state read, or
// nil on success.
func driveWorkflow(ctx context.Context, session *browser.Session, address string, workflow *casefile.FormWorkflow, record *casefile.FormWorkflowObservation, calls *requests) error {
	stop, err := session.ObserveNetwork(ctx, calls.observe)
	if err != nil {
		return fmt.Errorf("opening workflow: %w", err)
	}
	defer stop()
	if err := runBrowser(ctx, session, navigate(address), evaluate("location.pathname+location.search+location.hash", &record.BrowserPath),
		waitExpression(formReady, formReadyWait), evaluate(formWatchInvalid, nil), evaluate(formPartialPath, &record.PartialPath),
		evaluate(formSnapshot, &record.Initial)); err != nil {
		return fmt.Errorf("opening workflow: %w", err)
	}
	if _, err := calls.take(ctx, session); err != nil {
		return err
	}
	revision := 0
	for index, command := range workflow.Steps {
		record.FailedStep = index
		var step casefile.FormObservation
		step.Command = command
		driveErr, err := runWorkflowStep(ctx, session, address, calls, &revision, &step)
		if err != nil {
			return err
		}
		record.Steps = append(record.Steps, step)
		if driveErr != nil {
			return fmt.Errorf("%s: %w", command.Op, driveErr)
		}
	}
	record.FailedStep = len(workflow.Steps)
	record.FinalStore, err = readFormStore(ctx)
	record.FinalRead = err == nil
	return err
}

// runWorkflowStep applies one saved command and reads the browser, fresh SSR and direct
// SQLite state that follows it.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes address (string) which is the application URL to read freshly.
// Takes calls (*requests) which collects mutation-request evidence.
// Takes revision (*int) which holds the last awaited revision and receives the next.
// Takes step (*casefile.FormObservation) which holds the command and receives its
// observations.
//
// Returns error which is the command's own failure, recorded after the step's reads.
// Returns error which reports a failed snapshot or independent state read, or nil on
// success.
func runWorkflowStep(ctx context.Context, session *browser.Session, address string, calls *requests, revision *int, step *casefile.FormObservation) (driveErr, err error) {
	command := step.Command
	driveErr = applyFormCommand(ctx, session, command)
	if driveErr == nil && (command.Op == "create" || command.Op == "toggle" || command.Op == "delete") {
		*revision++
		driveErr = session.WaitForExpression(ctx, "Number(document.querySelector('#workflow-status').dataset.revision)>="+strconv.Itoa(*revision), formRevisionWait)
	}
	if err := session.Evaluate(ctx, formSnapshot, &step.State); err != nil {
		return driveErr, fmt.Errorf("snapshot after %s: %w", command.Op, err)
	}
	if step.Requests, err = calls.take(ctx, session); err != nil {
		return driveErr, err
	}
	if step.FreshRows, err = readFreshForm(ctx, address); err != nil {
		return driveErr, fmt.Errorf("fresh SSR after %s: %w", command.Op, err)
	}
	step.FreshRead = true
	if step.StoreRows, err = readFormStore(ctx); err != nil {
		return driveErr, fmt.Errorf("direct SQLite after %s: %w", command.Op, err)
	}
	step.StoreRead = true
	return driveErr, nil
}

// applyFormCommand applies one saved operation to the browser's form.
//
// Takes session (*browser.Session) which is the root browser session.
// Takes command (casefile.FormCommand) which selects the saved form operation.
//
// Returns error which reports a browser operation or DOM observation failure, or nil on
// success.
func applyFormCommand(ctx context.Context, session *browser.Session, command casefile.FormCommand) error {
	actions := browserActions{}
	if command.Op == "edit" || command.Op == "create" {
		actions = append(actions, evaluate(`(() => {const el=document.querySelector('#workflow-label').shadowRoot.querySelector('#native-label');el.focus();el.select()})()`, nil))
		if command.Value == "" {
			actions = append(actions, pressKey("Backspace"))
		} else {
			actions = append(actions, func(stepCtx context.Context, target *browser.Session) error {
				return target.InsertText(stepCtx, command.Value)
			})
		}
	}
	switch command.Op {
	case "reset":
		actions = append(actions, clickSelector("#workflow-reset"))
	case "create", "invalid":
		actions = append(actions, clickSelector("#workflow-create"))
	case "toggle", "delete":
		selector := fmt.Sprintf(`[data-fuzz-task="%d"] [data-%s]`, command.ID, command.Op)
		actions = append(actions, clickSelector(selector))
	}

	actions = append(actions, waitExpression("new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(()=>resolve(true))))", formRenderWait))
	stepCtx, cancel := stepContext(ctx, formCommandBudget)
	defer cancel()
	return actions.run(stepCtx, session)
}

// workflowFields records public workflow inputs without transport tokens.
//
// Takes contentType (string) which identifies the submitted body format.
// Takes body ([]byte) which holds the submitted bytes.
//
// Returns string which contains only the workflow's named public inputs, or a short
// description of why they could not be read.
func workflowFields(contentType string, body []byte) string {
	if len(body) > formBodyLimit {
		return "oversized"
	}
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "invalid content type"
	}
	fields := map[string]any{}
	switch mediaType {
	case "application/json":
		if problem := jsonFields(body, fields); problem != "" {
			return problem
		}
	case "multipart/form-data":
		values, problem := multipartFields(body, parameters["boundary"], publicWorkflowField)
		if problem != "" {
			return problem
		}
		for key, value := range values {
			fields[key] = value
		}
	default:
		return mediaType
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return "unencodable fields"
	}
	return string(data)
}

// publicWorkflowField reports whether a submitted field is a public workflow input.
//
// Takes key (string) which is the field name.
//
// Returns bool which is true for the workflow's named public inputs.
func publicWorkflowField(key string) bool {
	return key == "label" || key == "repeat" || key == "id" || key == "ignored"
}

// jsonFields copies the public workflow inputs of a JSON body.
//
// Takes body ([]byte) which holds the submitted JSON object.
// Takes fields (map[string]any) which receives the public inputs.
//
// Returns string which describes an unreadable body, or is empty on success.
func jsonFields(body []byte, fields map[string]any) string {
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return "invalid json"
	}
	for key, value := range object {
		if publicWorkflowField(key) {
			fields[key] = value
		}
	}
	return ""
}

// multipartFields reads the allowed fields of a multipart body.
//
// Takes body ([]byte) which holds the submitted bytes.
// Takes boundary (string) which separates the parts.
// Takes allowed (func(string) bool) which reports whether a field is public.
//
// Returns url.Values which holds the allowed values. A value above the limit is cut and
// marked as truncated.
// Returns string which describes an unreadable body, or is empty on success.
func multipartFields(body []byte, boundary string, allowed func(string) bool) (url.Values, string) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	values := url.Values{}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return values, ""
		}
		if err != nil {
			return nil, "invalid multipart"
		}
		if allowed(part.FormName()) {
			value, err := io.ReadAll(io.LimitReader(part, formValueLimit+1))
			if err != nil {
				return nil, "unreadable multipart"
			}
			values.Add(part.FormName(), textlimit.Truncate(string(value), formValueLimit))
		}
		if err := part.Close(); err != nil {
			return nil, "unreadable multipart"
		}
	}
}

// formWorkflowBudget sizes the session time a form workflow needs.
//
// Takes workflow (*casefile.FormWorkflow) which holds the saved form commands.
// Takes coverage (bool) which reports whether runtime coverage is collected.
//
// Returns time.Duration which is the opening budget, one step budget per command, the
// final store read, the coverage allowance when selected and the start-up margin.
func formWorkflowBudget(workflow *casefile.FormWorkflow, coverage bool) time.Duration {
	budget := sessionMargin + formOpenBudget + time.Duration(len(workflow.Steps))*formStepBudget + formStoreTimeout
	if coverage {
		budget += coverageBudget
	}
	return budget
}
