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

package daemon_adapters

import (
	"bytes"
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	// disabledWarningKey marks the warning about a declared but unenforced rate limit.
	disabledWarningKey = "disabled"

	// emptyIdentityWarningKey marks the warning about a key func returning no identity.
	emptyIdentityWarningKey = "empty-identity"
)

// validateCSRF extracts CSRF tokens from the request and validates them. The ephemeral
// token is extracted from arguments (for POST/PUT/DELETE) or from query parameters (for
// GET), then stripped from arguments so it does not leak to business logic.
//
// When csrfService is nil, validation is skipped entirely. When both tokens are empty
// (e.g. server-to-server API calls without CSRF tokens), validation is also skipped.
//
// Takes request (*http.Request) which provides the HTTP request context and cookies.
// Takes arguments (map[string]any) which contains the parsed request body; the ephemeral
// token key is deleted from this map as a side effect.
//
// Returns error when CSRF validation fails, nil on success or when skipped.
func (h *ActionHandler) validateCSRF(request *http.Request, arguments map[string]any) error {
	var ephemeralToken string
	if rawToken, ok := arguments[csrfEphemeralTokenKey].(string); ok {
		ephemeralToken = rawToken
	}
	delete(arguments, csrfEphemeralTokenKey)

	return h.validateCSRFWithToken(request, ephemeralToken)
}

// validateCSRFWithToken validates a CSRF token pair using the provided ephemeral token.
// Falls back to the query string if ephemeralToken is empty.
//
// Takes request (*http.Request) which provides headers and query parameters.
// Takes ephemeralToken (string) which is the ephemeral CSRF token from the request body.
//
// Returns error when CSRF validation fails or the token pair is invalid, nil on success
// or when validation is skipped.
func (h *ActionHandler) validateCSRFWithToken(request *http.Request, ephemeralToken string) error {
	if ephemeralToken == "" {
		ephemeralToken = request.URL.Query().Get(csrfEphemeralTokenKey)
	}

	if h.csrfService == nil {
		return nil
	}

	actionToken := request.Header.Get(headerCSRFActionToken)

	if actionToken == "" && ephemeralToken == "" {
		if h.enforceSecFetchSite && request.Header.Get("Sec-Fetch-Site") != "" {
			return &security_domain.CSRFValidationError{
				Code:    security_domain.CSRFErrorCodeMissing,
				Message: "CSRF tokens required for browser requests",
			}
		}
		return nil
	}

	valid, err := h.csrfService.ValidateCSRFPair(request, ephemeralToken, mem.Bytes(actionToken))
	if err != nil {
		return fmt.Errorf("validating CSRF token pair: %w", err)
	}
	if !valid {
		return &security_domain.CSRFValidationError{
			Code:    security_domain.CSRFErrorCodeInvalid,
			Message: "CSRF validation failed",
		}
	}

	return nil
}

// writeCSRFError writes a CSRF error response. It extracts the error code from a
// CSRFValidationError if available, otherwise uses a generic code.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
// Takes err (error) which is the CSRF validation error.
func (h *ActionHandler) writeCSRFError(w http.ResponseWriter, err error) {
	if csrfErr, ok := errors.AsType[*security_domain.CSRFValidationError](err); ok {
		h.writeJSON(w, http.StatusForbidden, map[string]any{
			"status":  http.StatusForbidden,
			"error":   csrfErr.Code,
			"message": csrfErr.Message,
		})
		return
	}

	h.writeJSON(w, http.StatusForbidden, map[string]any{
		"status":  http.StatusForbidden,
		"error":   "csrf_invalid",
		"message": "CSRF validation failed",
	})
}

// validateCaptcha checks captcha verification for actions that implement
// CaptchaProtected. The captcha token is extracted from arguments and deleted before
// binding, following the same pattern as CSRF token handling.
//
// If the action does not implement CaptchaProtected or returns a nil config, the token is
// cleaned up and the request is allowed through.
//
// If no captcha service is configured but the action requires captcha, the request is
// rejected with ErrCaptchaDisabled (fail-closed).
//
// Takes ctx (context.Context) which carries tracing and cancellation.
// Takes request (*http.Request) which provides the client IP.
// Takes action (any) which may implement daemon_domain.CaptchaProtected.
// Takes arguments (map[string]any) which contains the parsed request body; the captcha
// token key is deleted as a side effect.
// Takes actionName (string) which identifies the action for provider analytics.
//
// Returns error when captcha validation fails, nil on success or when skipped.
func (h *ActionHandler) validateCaptcha(ctx context.Context, request *http.Request, action any, arguments map[string]any, actionName string) error {
	captchaAction, ok := action.(daemon_domain.CaptchaProtected)
	if !ok {
		delete(arguments, captchaTokenKey)
		return nil
	}

	captchaConfig := captchaAction.CaptchaConfig()
	if captchaConfig == nil {
		delete(arguments, captchaTokenKey)
		return nil
	}

	var token string
	if rawToken, ok := arguments[captchaTokenKey].(string); ok {
		token = rawToken
	}
	delete(arguments, captchaTokenKey)

	if token == "" {
		return captcha_dto.ErrTokenMissing
	}

	if h.captchaService == nil || !h.captchaService.IsEnabled() {
		return fmt.Errorf("captcha required but service unavailable: %w", captcha_dto.ErrCaptchaDisabled)
	}

	providerAction := actionName
	if captchaConfig.Action != "" {
		providerAction = captchaConfig.Action
	}

	remoteIP := security_dto.ClientIPFromRequest(request)
	if remoteIP == "" {
		remoteIP = request.RemoteAddr
	}

	var response *captcha_dto.VerifyResponse
	var err error
	if captchaConfig.Provider != "" {
		response, err = h.captchaService.VerifyWithProvider(ctx, captchaConfig.Provider, token, remoteIP, providerAction, captchaConfig.ScoreThreshold)
	} else {
		response, err = h.captchaService.VerifyWithScore(ctx, token, remoteIP, providerAction, captchaConfig.ScoreThreshold)
	}
	if err != nil {
		return err
	}

	recordCaptchaScore(action, response)

	return nil
}

// recordCaptchaScore stores the captcha score on the action's request metadata when both
// the response contains a score and the action exposes request metadata.
//
// Takes action (any) which is the action whose request metadata receives the score.
// Takes response (*captcha_dto.VerifyResponse) which contains the optional score returned
// by the captcha provider.
func recordCaptchaScore(action any, response *captcha_dto.VerifyResponse) {
	if response == nil || response.Score == nil {
		return
	}

	type requestProvider interface {
		Request() *daemon_dto.RequestMetadata
	}

	provider, ok := action.(requestProvider)
	if !ok {
		return
	}

	if requestMeta := provider.Request(); requestMeta != nil {
		requestMeta.CaptchaScore = new(*response.Score)
	}
}

// writeCaptchaError writes a captcha validation error response. The error details are
// logged server-side but not exposed to the client to avoid leaking internal information.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
func (h *ActionHandler) writeCaptchaError(w http.ResponseWriter) {
	h.writeJSON(w, http.StatusForbidden, map[string]any{
		"status":  http.StatusForbidden,
		"code":    "CAPTCHA_FAILED",
		"message": "Captcha verification failed",
	})
}

// writeCaptchaRateLimitError writes a 429 response when captcha verification is rate
// limited.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
func (h *ActionHandler) writeCaptchaRateLimitError(w http.ResponseWriter) {
	h.writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"status":  http.StatusTooManyRequests,
		"code":    "RATE_LIMITED",
		"message": "Too many captcha attempts",
	})
}

// parseRequestBody parses the request body into a map of arguments. It detects the
// content type and uses the appropriate parser.
//
// Takes request (*http.Request) which provides the HTTP request to parse.
//
// Returns map[string]any which contains the parsed arguments.
// Returns error when decoding the request body fails.
func (h *ActionHandler) parseRequestBody(request *http.Request) (map[string]any, error) {
	if request.ContentLength == 0 || request.Method == http.MethodGet {
		return make(map[string]any), nil
	}

	contentType := request.Header.Get(headerContentType)

	if strings.HasPrefix(contentType, "multipart/form-data") {
		return h.parseMultipartBody(request)
	}

	body, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %w", err)
	}

	if err := validateJSONStructuralDepth(body, defaultMaxJSONBodyDepth); err != nil {
		return nil, fmt.Errorf("validating request body depth: %w", err)
	}

	decoder := stdjson.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var arguments map[string]any
	if err := decoder.Decode(&arguments); err != nil {
		return nil, fmt.Errorf("decoding request body: %w", err)
	}

	return arguments, nil
}

// queryArguments lifts a request's query string into the argument map an action binds
// from.
//
// Takes request (*http.Request) which supplies the query string.
//
// Returns map[string]any which holds at most maxQueryArgumentKeys parameters.
func queryArguments(request *http.Request) map[string]any {
	query := request.URL.Query()
	arguments := make(map[string]any, min(len(query), maxQueryArgumentKeys))

	for key, values := range query {
		if len(arguments) >= maxQueryArgumentKeys {
			break
		}
		if len(values) == 0 || key == csrfEphemeralTokenKey {
			continue
		}
		if len(values) == 1 {
			arguments[key] = values[0]

			continue
		}

		repeated := make([]any, len(values))
		for i, value := range values {
			repeated[i] = value
		}
		arguments[key] = repeated
	}

	return arguments
}

// validateJSONStructuralDepth scans a JSON byte stream and ensures the maximum
// brace/bracket nesting does not exceed maxDepth. The check is performed before any
// reflective decoding so attacker payloads that would otherwise blow the stack are
// rejected up front.
//
// Takes data ([]byte) which is the raw JSON payload to scan.
// Takes maxDepth (int) which is the inclusive maximum nesting depth.
//
// Returns error which wraps errJSONDepthExceeded when the payload nests beyond maxDepth.
// String literals (including escaped quotes) are skipped so brace characters within
// strings do not contribute to the depth.
func validateJSONStructuralDepth(data []byte, maxDepth int) error {
	if maxDepth <= 0 {
		return nil
	}

	depth := 0
	inString := false
	escape := false

	for i := range len(data) {
		c := data[i]
		if inString {
			inString, escape = advanceJSONStringScan(c, escape)
			continue
		}
		newDepth, entered := advanceJSONStructureScan(c, depth, &inString)
		depth = newDepth
		if entered && depth > maxDepth {
			return fmt.Errorf("nesting depth %d exceeds limit %d: %w", depth, maxDepth, errJSONDepthExceeded)
		}
	}

	return nil
}

// advanceJSONStringScan updates the in-string scan state for a single byte inside a JSON
// string literal.
//
// Takes c (byte) which is the byte being consumed inside the string literal.
// Takes escape (bool) which indicates whether the previous byte was a backslash so the
// current byte is treated as escaped.
//
// Returns inString (bool) which is true when the scanner remains inside the string
// literal after consuming c.
// Returns nextEscape (bool) which is true when the next byte should be treated as
// escaped.
func advanceJSONStringScan(c byte, escape bool) (inString, nextEscape bool) {
	if escape {
		return true, false
	}
	switch c {
	case '\\':
		return true, true
	case '"':
		return false, false
	}
	return true, false
}

// advanceJSONStructureScan updates the structural-scan state for a byte outside any
// string literal.
//
// Takes c (byte) which is the byte being consumed outside any string literal.
// Takes depth (int) which is the current structural nesting depth.
// Takes inString (*bool) which is set to true when c opens a string literal.
//
// Returns int which is the updated nesting depth after consuming c.
// Returns bool which is true when a brace or bracket was just opened so the caller can
// enforce the max depth.
func advanceJSONStructureScan(c byte, depth int, inString *bool) (int, bool) {
	switch c {
	case '"':
		*inString = true
	case '{', '[':
		return depth + 1, true
	case '}', ']':
		if depth > 0 {
			return depth - 1, false
		}
	}
	return depth, false
}

// parseMultipartBody parses a multipart form request into arguments. File uploads are
// added as *multipart.FileHeader for single files or []*multipart.FileHeader for multiple
// files.
//
// Takes request (*http.Request) which contains the multipart form data to parse.
//
// Returns map[string]any which contains the parsed form values and file handles.
// Returns error when parsing the multipart form fails.
func (h *ActionHandler) parseMultipartBody(request *http.Request) (map[string]any, error) {
	limit := h.maxMultipartFormBytes
	if limit <= 0 {
		limit = defaultMaxMultipartBytes
	}
	if err := request.ParseMultipartForm(limit); err != nil {
		return nil, fmt.Errorf("parsing multipart form: %w", err)
	}

	arguments := make(map[string]any)

	for key, values := range request.MultipartForm.Value {
		if len(values) == 1 {
			arguments[key] = values[0]
		} else {
			arguments[key] = values
		}
	}

	for key, files := range request.MultipartForm.File {
		if len(files) == 1 {
			arguments[key] = files[0]
		} else {
			arguments[key] = files
		}
	}

	return arguments, nil
}

// parseRawBody reads the entire request body and stores it as a RawBody. This is called
// for actions that have a RawBody parameter.
//
// Takes request (*http.Request) which provides the request body to read.
// Takes arguments (map[string]any) which receives the raw body under "_rawBody".
//
// Returns error when the request body cannot be read.
func (*ActionHandler) parseRawBody(request *http.Request, arguments map[string]any) error {
	contentType := request.Header.Get(headerContentType)

	data, err := io.ReadAll(request.Body)
	if err != nil {
		return fmt.Errorf("reading raw body: %w", err)
	}

	arguments["_rawBody"] = daemon_dto.NewRawBody(contentType, data)
	return nil
}

// buildActionRateLimitOverride extracts rate limit configuration from an action that
// implements RateLimitable. Returns nil when the action has no rate limit or when rate
// limiting is disabled.
//
// Takes action (any) which may implement daemon_domain.RateLimitable.
// Takes entry (ActionHandlerEntry) which identifies the action for keying.
//
// Returns *security_dto.RateLimitOverride which contains the resolved rate limit
// settings, or nil when no limit applies.
func (h *ActionHandler) buildActionRateLimitOverride(
	request *http.Request,
	action any,
	entry ActionHandlerEntry,
) *security_dto.RateLimitOverride {
	rateLimitable, ok := action.(daemon_domain.RateLimitable)
	if !ok {
		return nil
	}

	rl := rateLimitable.RateLimit()
	if rl == nil {
		return nil
	}

	if h.rateLimitMw == nil {
		h.warnRateLimitDisabled(request, entry)

		return nil
	}

	return &security_dto.RateLimitOverride{
		KeySuffix:         entry.Name,
		Identity:          h.resolveRateLimitIdentity(request, action, rl, entry),
		RequestsPerMinute: rl.RequestsPerMinute,
		BurstSize:         rl.BurstSize,
	}
}

// resolveRateLimitIdentity returns the identity an action is rate limited by.
//
// Takes request (*http.Request) which supplies the fallback identity.
// Takes action (any) which may expose its request metadata.
// Takes rl (*daemon_domain.RateLimit) which supplies the key func.
// Takes entry (ActionHandlerEntry) which names the action in logs.
//
// Returns string which is the sanitised identity, empty to key on the client address.
func (h *ActionHandler) resolveRateLimitIdentity(
	request *http.Request,
	action any,
	rl *daemon_domain.RateLimit,
	entry ActionHandlerEntry,
) string {
	if rl.KeyFunc == nil {
		return ""
	}

	getter, ok := action.(interface {
		Request() *daemon_dto.RequestMetadata
	})
	if !ok {
		return ""
	}

	reqMeta := getter.Request()
	if reqMeta == nil {
		return ""
	}

	identity := rl.KeyFunc(reqMeta)
	if strings.TrimSpace(identity) == "" {
		h.warnOncePerAction(request, entry, emptyIdentityWarningKey,
			"Rate limit key func returned no identity, falling back to the client address")

		return ""
	}

	return security_dto.SanitiseRateLimitKey(identity)
}

// checkRateLimit enforces per-action rate limiting using the action's RateLimitable
// interface. Returns true if the request is allowed, false if rate limited (429 response
// already written).
//
// Takes w (http.ResponseWriter) which receives rate limit headers and 429.
// Takes request (*http.Request) which provides client identity via proxy-aware IP.
// Takes action (any) which may implement daemon_domain.RateLimitable.
// Takes entry (ActionHandlerEntry) which identifies the action for keying.
//
// Returns bool which is true when the request is allowed.
func (h *ActionHandler) checkRateLimit(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	action any,
	entry ActionHandlerEntry,
) bool {
	override := h.buildActionRateLimitOverride(request, action, entry)
	if override == nil {
		return true
	}

	allowed := h.rateLimitMw.ActionHandler(w, request, override)
	if !allowed {
		actionRateLimitedCount.Add(ctx, 1,
			metric.WithAttributes(attribute.String(attributeKeyAction, entry.Name)))
	}
	return allowed
}

// checkBatchActionRateLimit enforces per-action rate limiting within a batch request.
// Unlike checkRateLimit, this does not write headers or a response body since batch
// results are collected and returned as a single JSON response.
//
// Takes request (*http.Request) which provides client identity via proxy-aware IP.
// Takes action (any) which may implement daemon_domain.RateLimitable.
// Takes entry (ActionHandlerEntry) which identifies the action for keying.
//
// Returns bool which is true when the request is allowed.
func (h *ActionHandler) checkBatchActionRateLimit(
	ctx context.Context,
	request *http.Request,
	action any,
	entry ActionHandlerEntry,
) bool {
	override := h.buildActionRateLimitOverride(request, action, entry)
	if override == nil {
		return true
	}

	allowed := h.rateLimitMw.CheckActionAllowed(request, override)
	if !allowed {
		actionRateLimitedCount.Add(ctx, 1,
			metric.WithAttributes(attribute.String(attributeKeyAction, entry.Name)))
	}
	return allowed
}

// warnRateLimitDisabled reports once that a declared rate limit is not enforced.
//
// Takes request (*http.Request) which carries the logger.
// Takes entry (ActionHandlerEntry) which names the action.
func (h *ActionHandler) warnRateLimitDisabled(request *http.Request, entry ActionHandlerEntry) {
	h.warnOncePerAction(request, entry, disabledWarningKey,
		"Action declares a rate limit but rate limiting is disabled, so it is not enforced")
}

// warnOncePerAction reports a rate limit misconfiguration once for a given action.
//
// Takes request (*http.Request) which carries the logger.
// Takes entry (ActionHandlerEntry) which names the action.
// Takes warningKey (string) which separates one warning from another for the same action.
// Takes message (string) which is the text to report.
func (h *ActionHandler) warnOncePerAction(
	request *http.Request,
	entry ActionHandlerEntry,
	warningKey string,
	message string,
) {
	warned, _ := h.rateLimitWarned.LoadOrStore(entry.Name+"\x00"+warningKey, &sync.Once{})

	once, ok := warned.(*sync.Once)
	if !ok {
		return
	}

	once.Do(func() {
		_, l := logger_domain.From(request.Context(), log)

		l.Warn(message, logger_domain.String(attributeKeyAction, entry.Name))
	})
}
