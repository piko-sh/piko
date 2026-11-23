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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"slices"
	"sync"

	"github.com/go-chi/chi/v5"
	"piko.sh/json"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/security/security_adapters"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	// csrfBase is the fixture application's origin.
	csrfBase = "https://fixture.example"

	// csrfCookie names the binding cookie issued by Piko.
	csrfCookie = "_piko_csrf_token"

	// csrfField names the form's ephemeral token field.
	csrfField = "_csrf_ephemeral_token"

	// csrfReplyLimit bounds the decoded action response in bytes.
	csrfReplyLimit = 64 << 10

	// actionBodyLimit is the action handler's request body limit in bytes.
	actionBodyLimit = 4096

	// sameOrigin is the Sec-Fetch-Site value of a same-origin submission.
	sameOrigin = "same-origin"

	// csrfMissing is the refusal code for a submission without usable tokens.
	csrfMissing = "csrf_missing"

	// csrfHistoryLimit bounds the replayed choice bytes appended to the fixed lifecycle.
	csrfHistoryLimit = 12

	// csrfOperations is the number of step operations a choice byte selects between.
	csrfOperations = 8

	// labelShift selects the bits of a step that choose the submitted label.
	labelShift = 4
)

const (
	// forgeSwappedPair sends the other tab's ephemeral token with this tab's action token.
	forgeSwappedPair csrfForgery = iota

	// forgeTamperedAction alters one byte of the signed action token payload.
	forgeTamperedAction

	// forgeTamperedEphemeral alters the ephemeral token.
	forgeTamperedEphemeral

	// forgeCrossSiteWithoutTokens submits from another site without any token.
	forgeCrossSiteWithoutTokens

	// forgeryKinds is the number of forgery kinds.
	forgeryKinds
)

const (
	// forgeNone submits the tab's own token pair unchanged.
	forgeNone csrfForgery = 255
)

var (
	// csrfRuntime initialises the process-shared production token service once.
	csrfRuntime = sync.OnceValues(func() (security_domain.CSRFTokenService, error) {
		config := security_domain.SecurityConfig{
			HMACSecretKey:    []byte("public-local-fixture-key-for-form-contract"),
			CSRFTokenMaxAge:  0,
			CSRFCookieMaxAge: 0,
		}
		return security_domain.NewCSRFTokenService(config, security_adapters.NewNoOpBinderAdapter(), csrfSource)
	})

	// csrfSource provides the production cookie-based CSRF source.
	csrfSource = security_adapters.NewCookieCSRFSourceAdapter(security_adapters.NewSecureCookieWriter(
		security_adapters.CookieSecurityValues{DefaultSameSite: "", ForceHTTPOnly: true, ForceSecureOnHTTPS: true}, true), 0)

	// csrfLabels lists the submitted business labels.
	csrfLabels = [...]string{"weekly note", "récap", "two + three", "blue & green"}
)

// csrfForgery selects how a submission departs from the tab's own token pair.
type csrfForgery byte

// csrfTab owns a tab's issued token pair and cookie generation.
type csrfTab struct {
	// pair holds this tab's owned token pair.
	pair security_dto.CSRFPair

	// generation identifies the cookie generation used to issue tokens.
	generation int
}

// csrfScenario holds a two-tab form lifecycle and its independent model.
type csrfScenario struct {
	// service is the production CSRF token service.
	service security_domain.CSRFTokenService

	// handler is the mounted production action handler.
	handler http.Handler

	// jar holds the browser's shared cookies.
	jar *cookiejar.Jar

	// origin is the fixture application origin.
	origin *url.URL

	// label is the displayed fixture label.
	label string

	// tabs holds the two independently issued forms.
	tabs [2]csrfTab

	// generation identifies the cookie generation used to issue tokens.
	generation int

	// calls counts delivered action calls.
	calls int

	// hasCookie tracks whether the model has a binding cookie.
	hasCookie bool

	// argsOK reports whether the action received the expected arguments.
	argsOK bool
}

// csrfReply is the action endpoint's JSON response.
type csrfReply struct {
	// Error is the CSRF refusal code.
	Error string `json:"error"`

	// Saved is the label the action stored.
	Saved string `json:"saved"`

	// Status is the HTTP status echoed in the body.
	Status int `json:"status"`
}

// issue issues a form and gives the selected tab ownership of its token storage.
//
// Takes tab (int) which selects one of the two independently issued forms.
//
// Returns error which reports token generation, cookie handling or issuance-model
// mismatch, or nil on success.
func (s *csrfScenario) issue(ctx context.Context, tab int) error {
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, csrfBase+"/form", nil)
	for _, cookie := range s.jar.Cookies(s.origin) {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	pair, err := s.service.GenerateCSRFPair(recorder, request, new(bytes.Buffer))
	if err != nil {
		return fmt.Errorf("csrf-issue: form pair unavailable: %w", err)
	}
	if pair.RawEphemeralToken == "" || len(pair.ActionToken) == 0 {
		return errors.New("csrf-issue: form pair unavailable")
	}
	cookies, err := recordedCookies(recorder)
	if err != nil {
		return err
	}
	if !s.hasCookie {
		if !sessionCookieIssued(cookies) {
			return errors.New("csrf-issue-cookie: session cookie properties differ")
		}
	} else if len(cookies) != 0 {
		return errors.New("csrf-reuse-cookie: returning form reset the session cookie")
	}
	s.jar.SetCookies(s.origin, cookies)
	s.hasCookie = true
	s.tabs[tab] = csrfTab{pair: pair, generation: s.generation}
	return nil
}

// logout expires the shared binding cookie.
//
// Returns error which reports cookie invalidation or retention inconsistent with the
// model, or nil on success.
func (s *csrfScenario) logout() error {
	recorder := httptest.NewRecorder()
	csrfSource.InvalidateToken(recorder)
	cookies, err := recordedCookies(recorder)
	if err != nil {
		return err
	}
	if len(cookies) != 1 || cookies[0].Name != csrfCookie || cookies[0].Path != "/" || cookies[0].MaxAge >= 0 {
		return errors.New("csrf-invalidate: session cookie deletion missing")
	}
	s.jar.SetCookies(s.origin, cookies)
	if len(s.jar.Cookies(s.origin)) != 0 {
		return errors.New("csrf-invalidate: browser retained session cookie")
	}
	s.hasCookie = false
	s.generation++
	return nil
}

// submit submits the selected tab's form and checks the expected token outcome.
//
// Takes tab (int) which selects one of the two independently issued forms.
// Takes asForm (bool) which selects form encoding instead of JSON.
// Takes empty (bool) which omits the form's ephemeral token.
// Takes forgery (csrfForgery) which alters a fresh pair. Stale or missing pairs are
// submitted unchanged.
//
// Returns error which reports form encoding, response or expected token-outcome mismatch,
// or nil on success.
func (s *csrfScenario) submit(ctx context.Context, tab int, asForm, empty bool, forgery csrfForgery) error {
	ephemeral, actionToken, wantCode := s.tokens(tab, empty)
	fetchSite := sameOrigin
	if wantCode == "" && forgery != forgeNone {
		ephemeral, actionToken, fetchSite, wantCode = s.forge(tab, ephemeral, actionToken, forgery)
	}
	fields := map[string]string{"label": s.label}
	if ephemeral != "" {
		fields[csrfField] = ephemeral
	}
	body, contentType, err := encodeSubmission(fields, asForm)
	if err != nil {
		return err
	}
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, csrfBase+"/_piko/actions/forms.Save", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Sec-Fetch-Site", fetchSite)
	if actionToken != "" {
		request.Header.Set("X-CSRF-Action-Token", actionToken)
	}
	for _, cookie := range s.jar.Cookies(s.origin) {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	before := s.calls
	s.argsOK = false
	s.handler.ServeHTTP(recorder, request)
	return s.checkReply(recorder, wantCode, before)
}

// tokens selects the token pair a tab submits and the refusal code the model expects.
//
// Takes tab (int) which selects one of the two independently issued forms.
// Takes empty (bool) which omits the form's tokens.
//
// Returns ephemeral (string) which is the ephemeral token to send.
// Returns actionToken (string) which is the action token to send.
// Returns wantCode (string) which is the expected refusal code, or empty for acceptance.
func (s *csrfScenario) tokens(tab int, empty bool) (ephemeral, actionToken, wantCode string) {
	saved := s.tabs[tab]
	if empty || saved.generation == 0 {
		return "", "", csrfMissing
	}
	wantCode = "csrf_expired"
	if s.hasCookie && s.generation == saved.generation {
		wantCode = ""
	}
	return saved.pair.RawEphemeralToken, string(saved.pair.ActionToken), wantCode
}

// checkReply compares an action response with the model's expected outcome.
//
// Takes recorder (*httptest.ResponseRecorder) which holds the action response.
// Takes wantCode (string) which is the expected refusal code, or empty for acceptance.
// Takes before (int) which is the action call count before the submission.
//
// Returns error which reports an oversized, undecodable or unexpected response.
func (s *csrfScenario) checkReply(recorder *httptest.ResponseRecorder, wantCode string, before int) error {
	if recorder.Body.Len() > csrfReplyLimit {
		return fmt.Errorf("csrf-reply: action response exceeds %d bytes", csrfReplyLimit)
	}
	var reply csrfReply
	if err := json.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
		return fmt.Errorf("csrf-reply: action response was not JSON: %w", err)
	}
	if wantCode == "" {
		if recorder.Code != http.StatusOK || s.calls-before != 1 || !s.argsOK || reply.Saved != s.label || reply.Error != "" {
			return errors.New("csrf-accept: fresh form did not deliver its business fields exactly once")
		}
		return nil
	}
	if recorder.Code != http.StatusForbidden || reply.Status != http.StatusForbidden || reply.Error != wantCode || s.calls != before {
		return fmt.Errorf("csrf-refuse: expected HTTP 403 with %s and no action call", wantCode)
	}
	return nil
}

// forge alters a fresh token pair and returns the outcome the service must report.
//
// Takes tab (int) which selects the submitting tab.
// Takes ephemeral (string) which is the tab's own ephemeral token.
// Takes actionToken (string) which is the tab's own signed action token.
// Takes forgery (csrfForgery) which selects the alteration.
//
// Returns sentEphemeral (string) which is the ephemeral token to send.
// Returns sentAction (string) which is the action token to send.
// Returns fetchSite (string) which is the Sec-Fetch-Site value to send.
// Returns wantCode (string) which is the expected CSRF error code.
func (s *csrfScenario) forge(tab int, ephemeral, actionToken string, forgery csrfForgery) (sentEphemeral, sentAction, fetchSite, wantCode string) {
	const invalid = "csrf_invalid"
	switch forgery % forgeryKinds {
	case forgeSwappedPair:
		other := s.tabs[1-tab].pair.RawEphemeralToken
		if other == "" || other == ephemeral {
			other = "forged-ephemeral-token"
		}
		return other, actionToken, sameOrigin, invalid
	case forgeTamperedAction:
		forgedToken := []byte(actionToken)
		middle := len(forgedToken) / 2
		forgedToken[middle] ^= 1
		return ephemeral, string(forgedToken), sameOrigin, invalid
	case forgeTamperedEphemeral:
		return ephemeral + "x", actionToken, sameOrigin, invalid
	default:
		return "", "", "cross-site", csrfMissing
	}
}

// sessionCookieIssued reports whether a first form issued exactly one conforming session
// binding cookie.
//
// Takes cookies ([]*http.Cookie) which lists the cookies the response set.
//
// Returns bool which reports whether the cookie properties match the model.
func sessionCookieIssued(cookies []*http.Cookie) bool {
	if len(cookies) != 1 {
		return false
	}
	cookie := cookies[0]
	return cookie.Name == csrfCookie && cookie.Value != "" && cookie.Path == "/" && cookie.SameSite == http.SameSiteStrictMode &&
		cookie.HttpOnly && cookie.Secure && cookie.MaxAge == 0
}

// encodeSubmission encodes the submitted fields as multipart form data or JSON.
//
// Takes fields (map[string]string) which holds the submitted fields.
// Takes asForm (bool) which selects form encoding instead of JSON.
//
// Returns *bytes.Buffer which holds the encoded body.
// Returns string which is the body's content type.
// Returns error which reports an encoding failure.
func encodeSubmission(fields map[string]string, asForm bool) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	if !asForm {
		if err := json.NewEncoder(&body).Encode(fields); err != nil {
			return nil, "", fmt.Errorf("csrf-setup: cannot encode form fields: %w", err)
		}
		return &body, "application/json", nil
	}
	writer := multipart.NewWriter(&body)
	if err := writer.SetBoundary("PikoFixtureFormBoundary"); err != nil {
		return nil, "", fmt.Errorf("csrf-setup: cannot set form boundary: %w", err)
	}
	for _, key := range []string{"label", csrfField} {
		value, ok := fields[key]
		if !ok {
			continue
		}
		if err := writer.WriteField(key, value); err != nil {
			return nil, "", fmt.Errorf("csrf-setup: cannot write form field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("csrf-setup: cannot finalise form: %w", err)
	}
	return &body, writer.FormDataContentType(), nil
}

// newCSRFScenario creates a two-tab form lifecycle with production token handling.
//
// Returns *csrfScenario which owns the production token service, shared cookie jar and
// two form tabs.
// Returns error which reports production token-service or fixture setup failure, or nil
// on success.
func newCSRFScenario() (*csrfScenario, error) {
	service, err := csrfRuntime()
	if err != nil {
		return nil, fmt.Errorf("csrf-setup: service construction failed: %w", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("csrf-setup: cookie jar unavailable: %w", err)
	}
	origin, err := url.Parse(csrfBase)
	if err != nil {
		return nil, fmt.Errorf("csrf-setup: fixture origin invalid: %w", err)
	}
	scenario := &csrfScenario{
		service:    service,
		handler:    nil,
		jar:        jar,
		origin:     origin,
		label:      "",
		tabs:       [2]csrfTab{},
		generation: 1,
		calls:      0,
		hasCookie:  false,
		argsOK:     false,
	}
	handler := daemon_adapters.NewActionHandler(service, actionBodyLimit, nil, security_dto.RateLimitValues{}, true, nil, nil)
	var entry daemon_adapters.ActionHandlerEntry
	entry.Name = "forms.Save"
	entry.Method = http.MethodPost
	entry.Create = func() any { return struct{}{} }
	entry.Invoke = func(_ context.Context, _ any, arguments map[string]any) (any, error) {
		scenario.calls++
		label, ok := arguments["label"].(string)
		scenario.argsOK = ok && len(arguments) == 1
		return map[string]string{"saved": label}, nil
	}
	handler.Register(entry)
	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	scenario.handler = router
	return scenario, nil
}

// RunCSRF replays normal browser form lifecycles.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func RunCSRF(ctx context.Context, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	scenario, err := newCSRFScenario()
	if err != nil {
		return err
	}

	const (
		issueFirst    = 0
		issueSecond   = 1
		submitFirst   = 2
		submitSecond  = 3
		logout        = 4
		submitEmpty   = 5
		forgeFirst    = 6
		forgeSecond   = 7
		multipartBody = 8
		forgeShift    = 6
	)
	forged := func(tab byte, kind csrfForgery) byte { return tab | byte(kind)<<forgeShift }
	steps := slices.Concat([]byte{issueFirst, issueSecond, submitFirst, submitSecond | multipartBody, logout,
		submitFirst, issueSecond, submitFirst, submitSecond,
		forged(forgeSecond, forgeSwappedPair), forged(forgeSecond, forgeTamperedAction),
		forged(forgeSecond, forgeTamperedEphemeral), forged(forgeSecond, forgeCrossSiteWithoutTokens),
		submitSecond, submitEmpty}, data[:min(len(data), csrfHistoryLimit)])
	for index, step := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		scenario.label = csrfLabels[int(step>>labelShift)%len(csrfLabels)]
		asForm := step&multipartBody != 0
		switch step % csrfOperations {
		case issueFirst, issueSecond:
			err = scenario.issue(ctx, int(step%2))
		case submitFirst, submitSecond:
			err = scenario.submit(ctx, int(step%2), asForm, false, forgeNone)
		case forgeFirst, forgeSecond:
			err = scenario.submit(ctx, int(step%2), asForm, false, csrfForgery(step>>forgeShift))
		case logout:
			err = scenario.logout()
		case submitEmpty:
			err = scenario.submit(ctx, 0, asForm, true, forgeNone)
		}
		if err != nil {
			return fmt.Errorf("%w (step %d)", err, index)
		}
	}
	return nil
}

// recordedCookies reads the cookies set on a recorded response and releases its body.
//
// Takes recorder (*httptest.ResponseRecorder) which holds the recorded response.
//
// Returns []*http.Cookie which lists the cookies the response set.
// Returns error which reports a failure to drain or close the response body.
func recordedCookies(recorder *httptest.ResponseRecorder) ([]*http.Cookie, error) {
	response := recorder.Result()
	cookies := response.Cookies()
	_, drainErr := io.Copy(io.Discard, response.Body)
	closeErr := response.Body.Close()
	if err := errors.Join(drainErr, closeErr); err != nil {
		return nil, fmt.Errorf("csrf-setup: cannot release recorded response: %w", err)
	}
	return cookies, nil
}
