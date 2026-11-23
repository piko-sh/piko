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

package shimcontracts

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"

	"piko.sh/json"
	"piko.sh/piko/wdk/captcha"
	hcaptcha "piko.sh/piko/wdk/captcha/captcha_provider_hcaptcha"
	recaptcha "piko.sh/piko/wdk/captcha/captcha_provider_recaptcha_v3"
	turnstile "piko.sh/piko/wdk/captcha/captcha_provider_turnstile"
)

const (
	// hostedCaptchaLimit bounds hosted-provider fixture response bytes.
	hostedCaptchaLimit = 512

	// acceptScoreBase selects the base score variant of the opening accepted step.
	acceptScoreBase = 2

	// acceptScoreStride keeps the opening accepted step at the base score.
	acceptScoreStride = 6

	// acceptVariants bounds the opening accepted step's drawn variants.
	acceptVariants = 8

	// extraStepStart is the first choice byte that selects an extra step.
	extraStepStart = 3

	// extraStepEnd bounds the choice bytes that select extra steps.
	extraStepEnd = 9

	// outcomeStride divides a choice byte before selecting an extra step's outcome.
	outcomeStride = 3

	// outcomeCount is the number of selectable hosted outcomes.
	outcomeCount = 4

	// omitActionBit marks a request without an expected action.
	omitActionBit = 8

	// omitWireActionBit marks a provider reply without an action.
	omitWireActionBit = 64
)

const (
	// hostedCaptchaAccept selects a successful verification response.
	hostedCaptchaAccept hostedCaptchaOutcome = iota

	// hostedCaptchaDecline selects a negative verification response.
	hostedCaptchaDecline

	// hostedCaptchaUnavailable selects a provider HTTP outage.
	hostedCaptchaUnavailable

	// hostedCaptchaOfflineCall selects a provider transport outage.
	hostedCaptchaOfflineCall
)

var (
	// errHostedCaptchaOffline marks the controlled provider transport outage.
	errHostedCaptchaOffline = errors.New("local fixture unavailable")

	// errHostedCaptchaRequest reports a provider request outside the fixture contract.
	errHostedCaptchaRequest = errors.New("unexpected local fixture request")

	// hostedCaptchaTokens lists the fixture response tokens.
	hostedCaptchaTokens = []string{"fixture token A", "fixture+token/B", "fixture-token-3"}

	// hostedCaptchaAddresses lists the fixture remote addresses, including none.
	hostedCaptchaAddresses = []string{"", "192.0.2.21", "2001:db8::21"}

	// hostedCaptchaScores pairs literal provider scores with independent normalised scores.
	// A successful reCAPTCHA v3 response without a score has no normalised score because
	// verification fails.
	hostedCaptchaScores = []hostedCaptchaScore{
		{present: false, raw: 0, hcaptcha: 1, recaptcha: 0},
		{present: true, raw: 0, hcaptcha: 1, recaptcha: 0},
		{present: true, raw: .25, hcaptcha: .75, recaptcha: .25},
		{present: true, raw: .5, hcaptcha: .5, recaptcha: .5},
		{present: true, raw: .75, hcaptcha: .25, recaptcha: .75},
		{present: true, raw: 1, hcaptcha: 0, recaptcha: 1},
	}
)

// hostedCaptchaAdapter holds one real provider and its expected configuration.
type hostedCaptchaAdapter struct {
	// provider is the real public provider implementation.
	provider captcha.Provider

	// kind selects the provider kind.
	kind captcha.ProviderType

	// endpoint is the provider's verification URL.
	endpoint string

	// script is the provider's expected widget script URL.
	script string

	// site is the provider's public site key.
	site string

	// secret is the provider's fixture secret key.
	secret string
}

// hostedCaptchaOutcome selects a recorded HTTP or transport outcome.
type hostedCaptchaOutcome byte

// hostedCaptchaBody records whether a provider closed its response body.
type hostedCaptchaBody struct {
	// Reader supplies the response bytes.
	io.Reader

	// closed reports whether the response body was closed.
	closed bool
}

// hostedCaptchaContextKey marks hosted-provider context values.
type hostedCaptchaContextKey struct{}

// hostedCaptchaRequest specifies the expected verification request and reply.
type hostedCaptchaRequest struct {
	// fields maps wire field names to encoded values.
	fields url.Values

	// endpoint is the provider's verification URL.
	endpoint string

	// body is the controlled verification response body.
	body string

	// outcome selects the provider's response or transport failure.
	outcome hostedCaptchaOutcome
}

// hostedCaptchaTransport records provider requests without sending network traffic.
type hostedCaptchaTransport struct {
	// expected holds the expected provider request.
	expected *hostedCaptchaRequest

	// sent tracks the response body given to the provider.
	sent *hostedCaptchaBody

	// mismatch records a request-contract mismatch.
	mismatch string

	// calls counts observed calls.
	calls int
}

// hostedCaptchaScore pairs a literal provider score with its normalised forms.
type hostedCaptchaScore struct {
	// raw is the score the provider reports on the wire.
	raw float64

	// hcaptcha is the normalised hCaptcha confidence.
	hcaptcha float64

	// recaptcha is the normalised reCAPTCHA confidence.
	recaptcha float64

	// present reports whether the provider reports a score at all.
	present bool
}

// hostedCaptchaStep selects one provider outcome and score variant.
type hostedCaptchaStep struct {
	// outcome selects the provider's response or transport failure.
	outcome hostedCaptchaOutcome

	// choice selects the provider score and reply variant.
	choice byte
}

// Close records that the provider closed its response body.
//
// Returns error which is always nil.
func (b *hostedCaptchaBody) Close() error {
	b.closed = true
	return nil
}

// RoundTrip records a provider request and supplies the controlled reply.
//
// Takes request (*http.Request) which is the incoming verification request.
//
// Returns *http.Response which is the controlled verification response whose body must be
// closed by the provider.
// Returns error which reports a request-contract mismatch or the controlled transport
// outage, or nil on success.
func (t *hostedCaptchaTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.calls++
	if request.Body != nil {
		defer func() { _ = request.Body.Close() }()
	}
	want := t.expected
	if want == nil || request.Context().Value(hostedCaptchaContextKey{}) != want {
		t.mismatch = "context"
		return nil, fmt.Errorf("%w: outside the active local fixture", errHostedCaptchaRequest)
	}
	if request.URL.String() != want.endpoint || request.Method != http.MethodPost {
		t.mismatch = "endpoint"
		return nil, fmt.Errorf("%w: endpoint", errHostedCaptchaRequest)
	}
	if request.Body == nil || request.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.mismatch = "encoding"
		return nil, fmt.Errorf("%w: content type", errHostedCaptchaRequest)
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, hostedCaptchaLimit+1))
	if err != nil || len(body) > hostedCaptchaLimit {
		t.mismatch = "body"
		return nil, fmt.Errorf("%w: body", errHostedCaptchaRequest)
	}
	form, err := url.ParseQuery(string(body))
	if err != nil || !reflect.DeepEqual(form, want.fields) {
		t.mismatch = "fields"
		return nil, fmt.Errorf("%w: fields", errHostedCaptchaRequest)
	}
	if want.outcome == hostedCaptchaOfflineCall {
		return nil, errHostedCaptchaOffline
	}
	status := http.StatusOK
	if want.outcome == hostedCaptchaUnavailable {
		status = http.StatusServiceUnavailable
	}
	t.sent = &hostedCaptchaBody{Reader: strings.NewReader(want.body), closed: false}
	var response http.Response
	response.Request = request
	response.StatusCode = status
	response.Header = http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}
	response.Body = t.sent
	return &response, nil
}

// newHostedCaptchaAdapters constructs the three real hosted-provider adapters.
//
// Takes transport (http.RoundTripper) which carries every provider request.
//
// Returns []hostedCaptchaAdapter which lists hCaptcha, Turnstile and reCAPTCHA provider
// fixtures in that order.
// Returns error which reports provider construction or public metadata mismatch, or nil
// on success.
func newHostedCaptchaAdapters(transport http.RoundTripper) ([]hostedCaptchaAdapter, error) {
	adapters := []hostedCaptchaAdapter{
		hostedAdapter(captcha.ProviderTypeHCaptcha, "https://api.hcaptcha.com/siteverify", "https://js.hcaptcha.com/1/api.js"),
		hostedAdapter(captcha.ProviderTypeTurnstile, "https://challenges.cloudflare.com/turnstile/v0/siteverify",
			"https://challenges.cloudflare.com/turnstile/v0/api.js"),
		hostedAdapter(captcha.ProviderTypeRecaptchaV3, "https://www.google.com/recaptcha/api/siteverify",
			"https://www.google.com/recaptcha/api.js?render=fixture-site-recaptcha_v3"),
	}
	for index := range adapters {
		adapter := &adapters[index]
		provider, err := newHostedProvider(adapter, transport)
		if err != nil {
			return nil, fmt.Errorf("hosted-captcha-construction: provider did not accept its local fixture settings: %w", err)
		}
		adapter.provider = provider
		if provider.Type() != adapter.kind || provider.SiteKey() != adapter.site || provider.ScriptURL() != adapter.script {
			return nil, fmt.Errorf("hosted-captcha-metadata: %s widget metadata differs", adapter.kind)
		}
	}
	return adapters, nil
}

// hostedAdapter describes a hosted provider with its fixture keys.
//
// Takes kind (captcha.ProviderType) which selects the provider kind.
// Takes endpoint (string) which is the provider's verification URL.
// Takes script (string) which is the provider's expected widget script URL.
//
// Returns hostedCaptchaAdapter which describes the provider before construction.
func hostedAdapter(kind captcha.ProviderType, endpoint, script string) hostedCaptchaAdapter {
	return hostedCaptchaAdapter{
		provider: nil,
		kind:     kind,
		endpoint: endpoint,
		script:   script,
		site:     "fixture-site-" + string(kind),
		secret:   "local-example-key-" + string(kind),
	}
}

// newHostedProvider constructs the real provider an adapter describes.
//
// Takes adapter (*hostedCaptchaAdapter) which describes the provider and its keys.
// Takes transport (http.RoundTripper) which carries every provider request.
//
// Returns captcha.Provider which is the constructed provider.
// Returns error which reports a rejected configuration or an unsupported kind.
func newHostedProvider(adapter *hostedCaptchaAdapter, transport http.RoundTripper) (captcha.Provider, error) {
	switch adapter.kind {
	case captcha.ProviderTypeHCaptcha:
		return hcaptcha.NewProvider(hcaptcha.Config{SiteKey: adapter.site, SecretKey: adapter.secret},
			hcaptcha.WithMaxResponseBytes(hostedCaptchaLimit), hcaptcha.WithTransport(transport))
	case captcha.ProviderTypeTurnstile:
		return turnstile.NewProvider(turnstile.Config{SiteKey: adapter.site, SecretKey: adapter.secret},
			turnstile.WithMaxResponseBytes(hostedCaptchaLimit), turnstile.WithTransport(transport))
	case captcha.ProviderTypeRecaptchaV3:
		return recaptcha.NewProvider(recaptcha.Config{SiteKey: adapter.site, SecretKey: adapter.secret, TestKeyScore: nil},
			recaptcha.WithMaxResponseBytes(hostedCaptchaLimit), recaptcha.WithTransport(transport))
	default:
		return nil, fmt.Errorf("%w: unsupported provider kind %s", errHostedCaptchaRequest, adapter.kind)
	}
}

// RunCaptchaAdapters runs up to 24 verification calls across three real public providers
// in a dedicated worker. Each call uses private providers and a private transport,
// independent of other calls and of the process default transport.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns error which reports a model mismatch or failed production operation.
func RunCaptchaAdapters(ctx context.Context, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	transport := new(hostedCaptchaTransport)
	adapters, err := newHostedCaptchaAdapters(transport)
	if err != nil {
		return err
	}
	for index := range adapters {
		var choice byte
		if index < len(data) {
			choice = data[index]
		}
		if err := runHostedBridge(ctx, &adapters[index], transport, choice); err != nil {
			return err
		}
	}
	if len(data) <= extraStepStart {
		return nil
	}
	for _, choice := range data[extraStepStart:min(len(data), extraStepEnd)] {
		adapter := &adapters[int(choice)%len(adapters)]
		step := hostedCaptchaStep{outcome: hostedCaptchaOutcome(choice / outcomeStride % outcomeCount), choice: choice}
		if err := exerciseHostedCaptcha(ctx, adapter, transport, step); err != nil {
			return err
		}
	}
	return nil
}

// runHostedBridge runs the fixed outcome sequence against one provider.
//
// Takes adapter (*hostedCaptchaAdapter) which is the hosted-provider fixture under test.
// Takes transport (*hostedCaptchaTransport) which records the provider's requests.
// Takes choice (byte) which varies the sequence's requests and replies.
//
// Returns error which reports a model mismatch.
func runHostedBridge(ctx context.Context, adapter *hostedCaptchaAdapter, transport *hostedCaptchaTransport, choice byte) error {
	bridge := []hostedCaptchaStep{
		{outcome: hostedCaptchaAccept, choice: acceptScoreBase + acceptScoreStride*(choice%acceptVariants)},
		{outcome: hostedCaptchaDecline, choice: choice},
		{outcome: hostedCaptchaUnavailable, choice: choice},
		{outcome: hostedCaptchaOfflineCall, choice: choice},
		{outcome: hostedCaptchaAccept, choice: choice},
		{outcome: hostedCaptchaAccept, choice: 0},
	}
	for _, step := range bridge {
		if err := exerciseHostedCaptcha(ctx, adapter, transport, step); err != nil {
			return err
		}
	}
	return nil
}

// exerciseHostedCaptcha checks one real provider's request and controlled outcome.
//
// Takes adapter (*hostedCaptchaAdapter) which is the hosted-provider fixture under test.
// Takes transport (*hostedCaptchaTransport) which records the real provider's outbound
// request.
// Takes step (hostedCaptchaStep) which selects the controlled provider outcome.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func exerciseHostedCaptcha(ctx context.Context, adapter *hostedCaptchaAdapter, transport *hostedCaptchaTransport, step hostedCaptchaStep) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	request, fields, reportedAction := hostedRequest(adapter, step.choice)
	score := hostedCaptchaScores[int(step.choice)%len(hostedCaptchaScores)]
	expected, wire := hostedReply(adapter, step, &score, reportedAction)
	payload, err := json.Marshal(wire)
	if err != nil {
		return fmt.Errorf("hosted-captcha-fixture: response encoding failed: %w", err)
	}
	if len(payload) > hostedCaptchaLimit {
		return errors.New("hosted-captcha-fixture: response encoding exceeds fixture bounds")
	}
	want := &hostedCaptchaRequest{fields: fields, endpoint: adapter.endpoint, body: string(payload), outcome: step.outcome}
	*transport = hostedCaptchaTransport{expected: want, sent: nil, mismatch: "", calls: 0}
	result, err := adapter.provider.Verify(context.WithValue(ctx, hostedCaptchaContextKey{}, want), &request)
	if err := checkHostedTransport(adapter, transport, step); err != nil {
		return err
	}
	if failure := hostedFailure(adapter, step, &expected, &score); failure != nil {
		if result != nil || !errors.Is(err, failure) {
			return fmt.Errorf("hosted-captcha-unavailable: %s outage or refusal result differs", adapter.kind)
		}
		return nil
	}
	if err != nil || result == nil {
		return fmt.Errorf("hosted-captcha-result: %s did not produce a verdict", adapter.kind)
	}
	return checkHostedCaptchaVerdict(adapter.kind, result, &expected)
}

// hostedRequest builds a step's verification request and the form fields the provider
// must send.
//
// Takes adapter (*hostedCaptchaAdapter) which is the hosted-provider fixture under test.
// Takes choice (byte) which selects the token, address and action.
//
// Returns captcha.VerifyRequest which is the verification input.
// Returns url.Values which holds the expected wire form fields.
// Returns string which is the action the provider reports.
func hostedRequest(adapter *hostedCaptchaAdapter, choice byte) (captcha.VerifyRequest, url.Values, string) {
	reportedAction := []string{"notes_save", "notes_preview"}[choice>>2&1]
	request := captcha.VerifyRequest{
		Token:    hostedCaptchaTokens[int(choice>>3)%len(hostedCaptchaTokens)],
		Action:   reportedAction,
		RemoteIP: hostedCaptchaAddresses[int(choice>>4)%len(hostedCaptchaAddresses)],
	}
	if choice&omitActionBit != 0 {
		request.Action = ""
	}
	fields := url.Values{"secret": {adapter.secret}, "response": {request.Token}}
	if request.RemoteIP != "" {
		fields.Set("remoteip", request.RemoteIP)
	}
	if adapter.kind == captcha.ProviderTypeHCaptcha {
		fields.Set("sitekey", adapter.site)
	}
	return request, fields, reportedAction
}

// hostedReply builds the provider's wire reply and the verdict it must produce.
//
// Takes adapter (*hostedCaptchaAdapter) which is the hosted-provider fixture under test.
// Takes step (hostedCaptchaStep) which selects the controlled provider outcome.
// Takes score (*hostedCaptchaScore) which selects the reported score.
// Takes reportedAction (string) which is the action the provider reports.
//
// Returns captcha.VerifyResponse which is the independently expected verdict.
// Returns map[string]any which is the provider's wire reply.
func hostedReply(adapter *hostedCaptchaAdapter, step hostedCaptchaStep, score *hostedCaptchaScore, reportedAction string) (captcha.VerifyResponse, map[string]any) {
	choice := step.choice
	expected := captcha.VerifyResponse{
		Timestamp:  time.Time{},
		Score:      nil,
		Action:     "",
		Hostname:   "",
		ErrorCodes: nil,
		Success:    step.outcome == hostedCaptchaAccept,
	}
	wire := map[string]any{"success": expected.Success}
	normalised := 0.0
	if expected.Success {
		normalised = 1
		expected.Hostname = []string{"forms.example", "preview.example"}[choice>>1&1]
		expected.Timestamp = time.Date(2024, 8, 6, 9, 30, 0, 0, time.UTC)
		wire["hostname"] = expected.Hostname
		wire["challenge_ts"] = "2024-08-06T11:30:00+02:00"
		if adapter.kind != captcha.ProviderTypeHCaptcha && choice&omitWireActionBit == 0 {
			wire["action"] = reportedAction
			expected.Action = reportedAction
		}
		switch adapter.kind {
		case captcha.ProviderTypeHCaptcha:
			normalised = score.hcaptcha
		case captcha.ProviderTypeRecaptchaV3:
			normalised = score.recaptcha
		default:
		}
		if adapter.kind != captcha.ProviderTypeTurnstile && score.present {
			wire["score"] = score.raw
		}
	} else if step.outcome == hostedCaptchaDecline {
		expected.ErrorCodes = []string{"invalid-input-response"}
		wire["error-codes"] = expected.ErrorCodes
	}
	expected.Score = &normalised
	return expected, wire
}

// checkHostedTransport checks that the provider made one conforming request and closed
// the reply.
//
// Takes adapter (*hostedCaptchaAdapter) which is the hosted-provider fixture under test.
// Takes transport (*hostedCaptchaTransport) which recorded the provider's request.
// Takes step (hostedCaptchaStep) which selects the controlled provider outcome.
//
// Returns error which reports a nonconforming request, a wrong call count or an unclosed
// reply.
func checkHostedTransport(adapter *hostedCaptchaAdapter, transport *hostedCaptchaTransport, step hostedCaptchaStep) error {
	if transport.mismatch != "" {
		return fmt.Errorf("hosted-captcha-request-%s: %s verification request differs", transport.mismatch, adapter.kind)
	}
	if transport.calls != 1 {
		return fmt.Errorf("hosted-captcha-call-count: %s issued %d HTTP calls", adapter.kind, transport.calls)
	}
	if step.outcome != hostedCaptchaOfflineCall && (transport.sent == nil || !transport.sent.closed) {
		return fmt.Errorf("hosted-captcha-response-close: %s response body was not closed", adapter.kind)
	}
	return nil
}

// hostedFailure selects the error a step must produce instead of a verdict.
//
// Takes adapter (*hostedCaptchaAdapter) which is the hosted-provider fixture under test.
// Takes step (hostedCaptchaStep) which selects the controlled provider outcome.
// Takes expected (*captcha.VerifyResponse) which is the independently expected verdict.
// Takes score (*hostedCaptchaScore) which is the reported score.
//
// Returns error which is the expected failure, or nil when a verdict is expected.
func hostedFailure(adapter *hostedCaptchaAdapter, step hostedCaptchaStep, expected *captcha.VerifyResponse, score *hostedCaptchaScore) error {
	switch {
	case step.outcome == hostedCaptchaUnavailable:
		return captcha.ErrProviderUnavailable
	case step.outcome == hostedCaptchaOfflineCall:
		return errHostedCaptchaOffline
	case adapter.kind == captcha.ProviderTypeRecaptchaV3 && expected.Success && !score.present:
		return recaptcha.ErrScoreMissing
	default:
		return nil
	}
}

// checkHostedCaptchaVerdict compares the provider's public verification fields.
//
// Takes kind (captcha.ProviderType) which is the hosted provider kind.
// Takes got (*captcha.VerifyResponse) which holds the observed result.
// Takes want (*captcha.VerifyResponse) which holds the independently expected result.
//
// Returns error which reports a verification-field mismatch, or nil on success.
func checkHostedCaptchaVerdict(kind captcha.ProviderType, got, want *captcha.VerifyResponse) error {
	if got.Success != want.Success || !slices.Equal(got.ErrorCodes, want.ErrorCodes) {
		return fmt.Errorf("hosted-captcha-verdict: %s success or error codes differ", kind)
	}
	if got.Score == nil || *got.Score != *want.Score {
		return fmt.Errorf("hosted-captcha-score: %s normalised confidence differs", kind)
	}
	if got.Hostname != want.Hostname || got.Action != want.Action || !got.Timestamp.Equal(want.Timestamp) {
		return fmt.Errorf("hosted-captcha-attribution: %s hostname, action or timestamp differs", kind)
	}
	return nil
}
