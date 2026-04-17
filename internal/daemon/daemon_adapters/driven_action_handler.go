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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	// httpErrorStatusThreshold is the minimum HTTP status code considered an
	// error.
	httpErrorStatusThreshold = 400

	// attributeKeyAction is the attribute key for action names in logs and metrics.
	attributeKeyAction = "action"

	// csrfEphemeralTokenKey is the key used for the ephemeral CSRF token in
	// request arguments (POST body) or query parameters (GET).
	csrfEphemeralTokenKey = "_csrf_ephemeral_token"

	// captchaTokenKey is the key used for the captcha token in request
	// arguments. The token is extracted and deleted before argument binding.
	captchaTokenKey = "_captcha_token"

	// headerCSRFActionToken is the HTTP header for the signed CSRF action token.
	headerCSRFActionToken = "X-CSRF-Action-Token"

	// defaultMaxMultipartBytes is the default maximum size (32 MiB) for
	// multipart form data when no explicit limit is configured.
	defaultMaxMultipartBytes = 32 << 20

	// maxBatchActions is the maximum number of actions allowed in a single
	// batch request. This bounds CPU and memory usage regardless of the body
	// size limit.
	maxBatchActions = 100
)

// ActionHandler is a generated-code friendly action handler that dispatches
// requests to actions using the generated registry.
//
// Actions are dispatched through generated wrapper functions that provide
// type-safe argument handling.
type ActionHandler struct {
	// csrfService validates CSRF tokens.
	csrfService security_domain.CSRFTokenService

	// responseCache stores cached action responses keyed by action name and
	// request characteristics. Nil when the cache hexagon is unavailable.
	responseCache cache_domain.Cache[string, []byte]

	// captchaService verifies captcha tokens. Nil when captcha is disabled.
	captchaService captcha_domain.CaptchaServicePort

	// spamdetectService analyses form content for spam. Nil when disabled.
	spamdetectService spamdetect_domain.SpamDetectServicePort

	// registry maps action names to their handler entries.
	registry map[string]ActionHandlerEntry

	// rateLimitMw applies per-action rate limiting. Nil when rate limiting
	// is disabled globally.
	rateLimitMw *rateLimitMiddleware

	// maxBodyBytes is the maximum request body size in bytes.
	maxBodyBytes int64

	// defaultMaxSSEDuration is the default maximum lifetime for SSE connections.
	// Zero means unlimited; individual actions can override via ResourceLimits.
	defaultMaxSSEDuration time.Duration

	// maxMultipartFormBytes is the maximum in-memory size for multipart form data.
	maxMultipartFormBytes int64

	// enforceSecFetchSite requires CSRF tokens on browser requests identified
	// by the Sec-Fetch-Site header.
	enforceSecFetchSite bool
}

// ActionHandlerEntry describes a registered action.
type ActionHandlerEntry struct {
	// Create returns a new action struct instance.
	Create func() any

	// Invoke calls the action with the given parsed arguments.
	Invoke func(ctx context.Context, action any, arguments map[string]any) (any, error)

	// Name is the action identifier in dot notation, used for routing and tracing.
	Name string

	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.).
	Method string

	// Middlewares contains handlers to apply to this action in sequence.
	Middlewares []func(http.Handler) http.Handler

	// HasSSE indicates if the action supports SSE streaming.
	HasSSE bool
}

// NewActionHandler creates a new action handler.
//
// Takes csrfService (security_domain.CSRFTokenService) for CSRF validation.
// Takes maxBodyBytes (int64) which is the maximum request body size.
// Takes rateLimitService (security_domain.RateLimitService) for per-action rate
// limiting; may be nil when rate limiting is disabled.
// Takes rateLimitConfig (security_dto.RateLimitValues) which configures rate limit
// behaviour.
// Takes enforceSecFetchSite (bool) which requires CSRF tokens on browser requests
// identified by the Sec-Fetch-Site header.
// Takes responseCache (cache_domain.Cache[string, []byte]) which stores cached
// action responses; may be nil to disable action response caching.
// Takes captchaService (captcha_domain.CaptchaServicePort) for captcha
// verification; may be nil when captcha is disabled.
//
// Returns *ActionHandler which is ready to register actions.
func NewActionHandler(
	csrfService security_domain.CSRFTokenService,
	maxBodyBytes int64,
	rateLimitService security_domain.RateLimitService,
	rateLimitConfig security_dto.RateLimitValues,
	enforceSecFetchSite bool,
	responseCache cache_domain.Cache[string, []byte],
	captchaService captcha_domain.CaptchaServicePort,
) *ActionHandler {
	var rlMw *rateLimitMiddleware
	if rateLimitConfig.Enabled && rateLimitService != nil {
		rlMw = newRateLimitMiddleware(rateLimitConfig, rateLimitService)
	}

	return &ActionHandler{
		registry:            make(map[string]ActionHandlerEntry),
		csrfService:         csrfService,
		maxBodyBytes:        maxBodyBytes,
		rateLimitMw:         rlMw,
		responseCache:       responseCache,
		captchaService:      captchaService,
		enforceSecFetchSite: enforceSecFetchSite,
	}
}

// Register adds an action to the registry.
//
// Takes entry (ActionHandlerEntry) which describes the action.
func (h *ActionHandler) Register(entry ActionHandlerEntry) {
	h.registry[entry.Name] = entry
}

// RegisterAll adds multiple actions to the registry.
//
// Takes entries (map[string]ActionHandlerEntry) which maps names to handlers.
func (h *ActionHandler) RegisterAll(entries map[string]ActionHandlerEntry) {
	for name, entry := range entries {
		entry.Name = name
		h.registry[name] = entry
	}
}

// Mount registers all actions with the given router.
//
// Takes r (chi.Router) which receives the action routes.
// Takes basePath (string) which is the base path for actions (e.g.,
// "/_piko/actions").
func (h *ActionHandler) Mount(r chi.Router, basePath string) {
	for _, entry := range h.registry {
		routePattern := fmt.Sprintf("%s/%s", basePath, entry.Name)

		handler := h.createHandler(entry)

		for i := len(entry.Middlewares) - 1; i >= 0; i-- {
			handler = entry.Middlewares[i](handler)
		}

		r.Method(entry.Method, routePattern, handler)

		if entry.HasSSE && entry.Method != http.MethodGet {
			r.Method(http.MethodGet, routePattern, handler)
		}
	}

	r.Post(fmt.Sprintf("%s/_batch", basePath), h.handleBatch)
}

// createHandler creates an HTTP handler for an action entry.
//
// Takes entry (ActionHandlerEntry) which defines the action to handle.
//
// Returns http.Handler which wraps the entry in an HTTP handler function.
func (h *ActionHandler) createHandler(entry ActionHandlerEntry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		h.handleRequest(w, request, entry)
	})
}

// handleRequest processes an HTTP request for an action.
//
// Takes w (http.ResponseWriter) which receives the response output.
// Takes request (*http.Request) which provides the incoming request data.
// Takes entry (ActionHandlerEntry) which defines the action to execute.
func (h *ActionHandler) handleRequest(w http.ResponseWriter, request *http.Request, entry ActionHandlerEntry) {
	ctx := extractOTelContext(request)
	ctx, span := tracer.Start(ctx, "handleActionRequest")
	span.SetAttributes(
		attribute.String("action.name", entry.Name),
		attribute.String("http.method", request.Method),
		attribute.String("http.path", request.URL.Path),
	)
	defer span.End()

	l := log.WithSpanContext(ctx)

	actionRequestCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String(attributeKeyAction, entry.Name),
			attribute.String("method", request.Method),
		),
	)

	l.Trace("Handling action request",
		logger_domain.String(attributeKeyAction, entry.Name),
	)

	if h.shouldUseSSE(request, entry) {
		h.handleSSE(ctx, w, request, entry, span)
		return
	}

	h.handleHTTP(ctx, w, request, entry, span)
}

// shouldUseSSE checks if the request wants SSE transport.
//
// Takes request (*http.Request) which provides the incoming request headers.
// Takes entry (ActionHandlerEntry) which specifies if SSE is available.
//
// Returns bool which is true when the entry supports SSE and the request
// accepts text/event-stream.
func (*ActionHandler) shouldUseSSE(request *http.Request, entry ActionHandlerEntry) bool {
	if !entry.HasSSE {
		return false
	}
	return request.Header.Get("Accept") == "text/event-stream"
}

// handleHTTP processes a standard HTTP action request.
//
// Takes w (http.ResponseWriter) which receives the response output.
// Takes request (*http.Request) which contains the incoming request data.
// Takes entry (ActionHandlerEntry) which defines the action to invoke.
// Takes span (trace.Span) which records the request trace.
func (h *ActionHandler) handleHTTP(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	entry ActionHandlerEntry,
	span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	action := entry.Create()
	startTime := time.Now()

	ctx, cancel, bodyLimit, slowThreshold := h.applyResourceLimits(ctx, action)
	if cancel != nil {
		defer cancel()
	}

	request = request.WithContext(ctx)
	h.injectMetadata(request, action)
	request.Body = http.MaxBytesReader(w, request.Body, bodyLimit)

	arguments, err := h.parseRequestBody(request)
	if err != nil {
		l.ReportError(span, err, "Failed to parse request body")
		h.writeError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if !h.runSecurityValidation(ctx, w, request, action, arguments, entry) {
		return
	}

	if h.handleCachedAction(ctx, w, request, cachedActionParams{
		Action:        action,
		Entry:         entry,
		Args:          arguments,
		Span:          span,
		StartTime:     startTime,
		SlowThreshold: slowThreshold,
	}) {
		return
	}

	result, err := entry.Invoke(ctx, action, arguments)
	if err != nil {
		l.ReportError(span, err, "Action execution failed")
		h.handleActionError(w, request, action, err)
		h.recordSlowAction(ctx, entry.Name, startTime, slowThreshold)
		return
	}

	h.applyResponseMetadata(w, action)
	response := h.buildFullResponse(action, result)
	h.writeJSON(w, http.StatusOK, response)
	span.SetStatus(codes.Ok, "Action completed successfully")
	h.recordSlowAction(ctx, entry.Name, startTime, slowThreshold)
}

// runSecurityValidation performs CSRF, rate limit, and captcha checks for an
// incoming action request. Returns true when all checks pass; returns false
// when a check fails, in which case an error response has already been written.
//
// Takes w (http.ResponseWriter) which receives any error responses.
// Takes request (*http.Request) which contains headers for validation.
// Takes action (any) which may carry rate limit or captcha configuration.
// Takes arguments (map[string]any) which holds parsed request arguments.
// Takes entry (ActionHandlerEntry) which identifies the action for logging.
//
// Returns bool which is true when all security checks pass, or false when a
// check fails and an error response has already been written.
func (h *ActionHandler) runSecurityValidation(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	action any,
	arguments map[string]any,
	entry ActionHandlerEntry,
) bool {
	_, l := logger_domain.From(ctx, log)

	if csrfErr := h.validateCSRF(request, arguments); csrfErr != nil {
		l.Warn("CSRF validation failed",
			logger_domain.String(attributeKeyAction, entry.Name),
			logger_domain.Error(csrfErr),
		)
		h.writeCSRFError(w, csrfErr)
		return false
	}

	if !h.checkRateLimit(ctx, w, request, action, entry) {
		return false
	}

	if captchaErr := h.validateCaptcha(ctx, request, action, arguments, entry.Name); captchaErr != nil {
		l.Warn("Captcha validation failed",
			logger_domain.String(attributeKeyAction, entry.Name),
			logger_domain.Error(captchaErr),
		)
		if errors.Is(captchaErr, captcha_dto.ErrRateLimited) {
			h.writeCaptchaRateLimitError(w)
		} else {
			h.writeCaptchaError(w)
		}
		return false
	}

	if spamErr := h.validateSpamDetect(ctx, request, action, arguments, entry.Name); spamErr != nil {
		l.Warn("Spam detection rejected submission",
			logger_domain.String(attributeKeyAction, entry.Name),
			logger_domain.Error(spamErr),
		)
		h.writeSpamDetectError(ctx, w, spamErr)
		return false
	}

	return true
}

// applyResourceLimits reads resource limits from the action and returns the
// adjusted context, cancel function, body limit, and slow threshold.
//
// Takes action (any) which may implement daemon_domain.ResourceLimitable.
//
// Returns context.Context which may have a timeout applied.
// Returns context.CancelFunc which cancels the timeout, or nil.
// Returns int64 which is the maximum request body size.
// Returns time.Duration which is the slow action threshold.
func (h *ActionHandler) applyResourceLimits(ctx context.Context, action any) (context.Context, context.CancelFunc, int64, time.Duration) {
	bodyLimit := h.maxBodyBytes
	var slowThreshold time.Duration
	var cancel context.CancelFunc

	rl, ok := action.(daemon_domain.ResourceLimitable)
	if !ok {
		return ctx, nil, bodyLimit, slowThreshold
	}

	limits := rl.ResourceLimits()
	if limits == nil {
		return ctx, nil, bodyLimit, slowThreshold
	}

	if limits.MaxRequestBodySize > 0 {
		bodyLimit = limits.MaxRequestBodySize
	}
	if limits.Timeout > 0 {
		ctx, cancel = context.WithTimeoutCause(ctx, limits.Timeout,
			fmt.Errorf("action execution exceeded %s timeout", limits.Timeout))
	}
	slowThreshold = limits.SlowThreshold
	return ctx, cancel, bodyLimit, slowThreshold
}

// cachedActionParams groups the parameters for handleCachedAction.
type cachedActionParams struct {
	// StartTime records when the action request began for slow-action detection.
	StartTime time.Time

	// Action is the instantiated action struct for the current request.
	Action any

	// Span is the tracing span for the current action request.
	Span trace.Span

	// Args contains the parsed request arguments passed to the action.
	Args map[string]any

	// Entry describes the registered action being executed.
	Entry ActionHandlerEntry

	// SlowThreshold is the duration after which the action is considered slow.
	SlowThreshold time.Duration
}

// handleCachedAction checks whether the action is cacheable and, if so,
// serves the response from cache or invokes the action and caches the result.
// Returns true when the response has been written (cache hit or miss with
// successful invocation).
//
// Takes w (http.ResponseWriter) which receives the response output.
// Takes request (*http.Request) which provides headers for cache key computation.
// Takes p (cachedActionParams) which groups the action, entry, arguments, span,
// start time, and slow threshold.
//
// Returns bool which is true when the response has been written.
func (h *ActionHandler) handleCachedAction(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	p cachedActionParams,
) bool {
	if h.responseCache == nil {
		return false
	}

	cacheable, ok := p.Action.(daemon_domain.Cacheable)
	if !ok {
		return false
	}
	cc := cacheable.CacheConfig()
	if cc == nil || cc.TTL <= 0 {
		return false
	}

	ctx, l := logger_domain.From(ctx, log)
	cacheKey := h.buildCacheKey(request, p.Args, p.Entry.Name, cc)

	if cached, found, _ := h.responseCache.GetIfPresent(ctx, cacheKey); found {
		actionCacheHitCount.Add(ctx, 1,
			metric.WithAttributes(attribute.String(attributeKeyAction, p.Entry.Name)))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Action-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cached)
		p.Span.SetStatus(codes.Ok, "Cache hit")
		h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)
		return true
	}

	actionCacheMissCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String(attributeKeyAction, p.Entry.Name)))

	result, invokeErr := p.Entry.Invoke(ctx, p.Action, p.Args)
	if invokeErr != nil {
		l.ReportError(p.Span, invokeErr, "Action execution failed")
		h.handleActionError(w, request, p.Action, invokeErr)
		h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)
		return true
	}

	h.applyResponseMetadata(w, p.Action)
	response := h.buildFullResponse(p.Action, result)
	jsonBytes, _ := json.Marshal(response)
	_ = h.responseCache.SetWithTTL(ctx, cacheKey, jsonBytes, cc.TTL)

	w.Header().Set("X-Action-Cache", "MISS")
	h.writeJSON(w, http.StatusOK, response)
	p.Span.SetStatus(codes.Ok, "Action completed, response cached")
	h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)
	return true
}

// handleSSE processes an SSE action request.
//
// Takes w (http.ResponseWriter) which writes the SSE response stream.
// Takes request (*http.Request) which provides the incoming request data.
// Takes entry (ActionHandlerEntry) which creates the action instance.
// Takes span (trace.Span) which records the operation status.
func (h *ActionHandler) handleSSE(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	entry ActionHandlerEntry,
	span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	if request.Method != http.MethodGet {
		emptyArgs := make(map[string]any)
		if csrfErr := h.validateCSRF(request, emptyArgs); csrfErr != nil {
			l.Warn("CSRF validation failed for SSE request",
				logger_domain.String(attributeKeyAction, entry.Name),
				logger_domain.Error(csrfErr),
			)
			h.writeCSRFError(w, csrfErr)
			return
		}
	}

	h.writeSSEHeaders(w)

	action := entry.Create()
	h.injectMetadata(request, action)

	ctx, request, cancel := h.applySSEDurationLimit(ctx, request, action)
	if cancel != nil {
		defer cancel()
	}

	h.executeSSEStream(ctx, w, request, action, entry, span, l)
}

// writeSSEHeaders sets the standard SSE response headers and disables the
// write deadline.
//
// Takes w (http.ResponseWriter) which receives the SSE headers.
func (*ActionHandler) writeSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
}

// applySSEDurationLimit applies a timeout to the context if the action or
// handler specifies a maximum SSE duration. The caller must defer the returned
// cancel function when it is non-nil.
//
// Takes request (*http.Request) which provides the request to update.
// Takes action (any) which may implement daemon_domain.ResourceLimitable.
//
// Returns context.Context which may have a timeout applied.
// Returns *http.Request which is updated with the new context if needed.
// Returns context.CancelFunc which cancels the timeout, or nil.
func (h *ActionHandler) applySSEDurationLimit(ctx context.Context, request *http.Request, action any) (context.Context, *http.Request, context.CancelFunc) {
	sseDuration := h.defaultMaxSSEDuration
	if rl, ok := action.(daemon_domain.ResourceLimitable); ok {
		if limits := rl.ResourceLimits(); limits != nil && limits.MaxSSEDuration > 0 {
			sseDuration = limits.MaxSSEDuration
		}
	}

	if sseDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeoutCause(ctx, sseDuration,
			fmt.Errorf("SSE stream exceeded %s duration limit", sseDuration))
		request = request.WithContext(ctx)
		return ctx, request, cancel
	}

	return ctx, request, nil
}

// executeSSEStream runs the SSE stream on the given action.
//
// Takes w (http.ResponseWriter) which receives the SSE events.
// Takes request (*http.Request) which provides the client context.
// Takes action (any) which must implement daemon_domain.SSECapable.
// Takes entry (ActionHandlerEntry) which identifies the action.
// Takes span (trace.Span) which records the operation status.
// Takes l (logger_domain.Logger) which provides structured logging.
//
// Concurrent goroutine is spawned to detect client disconnection via
// the request context.
func (*ActionHandler) executeSSEStream(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	action any,
	entry ActionHandlerEntry,
	span trace.Span,
	l logger_domain.Logger,
) {
	done := make(chan struct{})
	go func() {
		<-request.Context().Done()
		close(done)
	}()

	sseCapable, ok := action.(daemon_domain.SSECapable)
	if !ok {
		l.Error("Action does not implement SSECapable", logger_domain.String(attributeKeyAction, entry.Name))
		return
	}

	lastEventID := request.Header.Get("Last-Event-ID")
	stream := daemon_domain.NewSSEStream(w, done, lastEventID)
	if stream == nil {
		l.Error("Response writer does not support flushing for SSE", logger_domain.String(attributeKeyAction, entry.Name))
		return
	}

	if err := sseCapable.StreamProgress(stream); err != nil {
		l.ReportError(span, err, "SSE streaming failed")
		return
	}

	_ = ctx
	span.SetStatus(codes.Ok, "SSE streaming completed")
}

// injectMetadata injects request metadata into the action.
//
// Takes request (*http.Request) which provides the HTTP request details.
// Takes action (any) which is the action to inject metadata into.
func (*ActionHandler) injectMetadata(request *http.Request, action any) {
	type metadataInjector interface {
		SetRequest(request *daemon_dto.RequestMetadata)
		SetResponse(response *daemon_dto.ResponseWriter)
	}

	if injector, ok := action.(metadataInjector); ok {
		injector.SetRequest(&daemon_dto.RequestMetadata{
			Method:      request.Method,
			Path:        request.URL.Path,
			Headers:     request.Header,
			QueryParams: request.URL.Query(),
			RemoteAddr:  request.RemoteAddr,
			RawRequest:  request,
		})
		injector.SetResponse(daemon_dto.NewResponseWriter())
	}
}

// applyResponseMetadata applies cookies and headers from the action's
// response writer.
//
// Takes w (http.ResponseWriter) which receives the metadata.
// Takes action (any) which may implement responseGetter to provide metadata.
func (*ActionHandler) applyResponseMetadata(w http.ResponseWriter, action any) {
	type responseGetter interface {
		Response() *daemon_dto.ResponseWriter
	}

	if getter, ok := action.(responseGetter); ok {
		if response := getter.Response(); response != nil {
			for _, cookie := range response.GetCookies() {
				http.SetCookie(w, cookie)
			}
			for key, values := range response.GetHeaders() {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
		}
	}
}

// buildFullResponse builds the response, wrapping with helpers if present.
// If there are no helpers, returns the raw result to minimise response size.
//
// Takes action (any) which may provide a response with helpers.
// Takes result (any) which is the raw result to wrap or return.
//
// Returns any which is either the raw result or a wrapped response with
// helpers.
func (*ActionHandler) buildFullResponse(action any, result any) any {
	type responseGetter interface {
		Response() *daemon_dto.ResponseWriter
	}

	if getter, ok := action.(responseGetter); ok {
		if response := getter.Response(); response != nil {
			helpers := response.GetHelpers()
			if len(helpers) > 0 {
				return daemon_dto.ActionFullResponse{
					Data:    result,
					Helpers: helpers,
				}
			}
		}
	}

	return result
}

// validateCSRF extracts CSRF tokens from the request and validates
// them. The ephemeral token is extracted from arguments (for
// POST/PUT/DELETE) or from query
// parameters (for GET), then stripped from arguments so it does
// not leak to business
// logic.
//
// When csrfService is nil, validation is skipped entirely. When both tokens are
// empty (e.g. server-to-server API calls without CSRF tokens), validation is
// also skipped.
//
// Takes request (*http.Request) which provides the HTTP request context and
// cookies.
// Takes arguments (map[string]any) which contains the parsed request body; the
// ephemeral token key is deleted from this map as a side effect.
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

// validateCSRFWithToken validates a CSRF token pair using the provided ephemeral
// token. Falls back to the query string if ephemeralToken is empty.
//
// Takes request (*http.Request) which provides headers and query parameters.
// Takes ephemeralToken (string) which is the ephemeral CSRF token from the request body.
//
// Returns error when CSRF validation fails or the token pair is invalid, nil
// on success or when validation is skipped.
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

// writeCSRFError writes a CSRF error response. It extracts the error code
// from a CSRFValidationError if available, otherwise uses a generic code.
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
// CaptchaProtected. The captcha token is extracted from arguments and deleted
// before binding, following the same pattern as CSRF token handling.
//
// If the action does not implement CaptchaProtected or returns a nil config,
// the token is cleaned up and the request is allowed through.
//
// If no captcha service is configured but the action requires captcha, the
// request is rejected with ErrCaptchaDisabled (fail-closed).
//
// Takes ctx (context.Context) which carries tracing and cancellation.
// Takes request (*http.Request) which provides the client IP.
// Takes action (any) which may implement daemon_domain.CaptchaProtected.
// Takes arguments (map[string]any) which contains the parsed request body;
// the captcha token key is deleted as a side effect.
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

// recordCaptchaScore stores the captcha score on the action's request metadata
// when both the response contains a score and the action exposes request
// metadata.
//
// Takes action (any) which is the action whose request metadata receives the
// score.
// Takes response (*captcha_dto.VerifyResponse) which contains the optional
// score returned by the captcha provider.
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
		score := *response.Score
		requestMeta.CaptchaScore = &score
	}
}

// writeCaptchaError writes a captcha validation error response. The error
// details are logged server-side but not exposed to the client to avoid
// leaking internal information.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
func (h *ActionHandler) writeCaptchaError(w http.ResponseWriter) {
	h.writeJSON(w, http.StatusForbidden, map[string]any{
		"status":  http.StatusForbidden,
		"code":    "CAPTCHA_FAILED",
		"message": "Captcha verification failed",
	})
}

// writeCaptchaRateLimitError writes a 429 response when captcha verification
// is rate limited.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
func (h *ActionHandler) writeCaptchaRateLimitError(w http.ResponseWriter) {
	h.writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"status":  http.StatusTooManyRequests,
		"code":    "RATE_LIMITED",
		"message": "Too many captcha attempts",
	})
}

// parseRequestBody parses the request body into a map of arguments.
// It detects the content type and uses the appropriate parser.
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

	var arguments map[string]any
	if err := json.ConfigDefault.NewDecoder(request.Body).Decode(&arguments); err != nil {
		return nil, fmt.Errorf("decoding request body: %w", err)
	}

	return arguments, nil
}

// parseMultipartBody parses a multipart form request into arguments. File uploads
// are added as *multipart.FileHeader for single files or
// []*multipart.FileHeader for multiple files.
//
// Takes request (*http.Request) which contains the multipart form data to parse.
//
// Returns map[string]any which contains the parsed form values and file
// handles.
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

// parseRawBody reads the entire request body and stores it as a RawBody.
// This is called for actions that have a RawBody parameter.
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

// handleBatch processes a batch action request (continue all, report failures).
//
// Takes w (http.ResponseWriter) which receives the JSON response.
// Takes request (*http.Request) which contains the batch action request body.
func (h *ActionHandler) handleBatch(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, h.maxBodyBytes)

	ctx := extractOTelContext(request)
	ctx, span := tracer.Start(ctx, "handleBatchActionRequest")
	defer span.End()

	l := log.WithSpanContext(ctx)
	h.trackBatchMetrics(ctx, request)
	l.Trace("Handling batch action request")

	var batchReq daemon_dto.BatchActionRequest
	if err := json.ConfigDefault.NewDecoder(request.Body).Decode(&batchReq); err != nil {
		l.ReportError(span, err, "Failed to parse batch request body")
		h.writeError(w, http.StatusBadRequest, "Invalid batch request body", err)
		return
	}

	if csrfErr := h.validateCSRFWithToken(request, batchReq.CSRFEphemeralToken); csrfErr != nil {
		l.Warn("CSRF validation failed for batch request",
			logger_domain.Error(csrfErr),
		)
		h.writeCSRFError(w, csrfErr)
		return
	}

	if len(batchReq.Actions) == 0 {
		h.writeJSON(w, http.StatusOK, daemon_dto.BatchActionResponse{Results: []daemon_dto.BatchActionResult{}, Success: true})
		return
	}

	if len(batchReq.Actions) > maxBatchActions {
		h.writeError(w, http.StatusBadRequest,
			fmt.Sprintf("batch exceeds maximum of %d actions", maxBatchActions), nil)
		return
	}

	span.SetAttributes(attribute.Int("batch.action_count", len(batchReq.Actions)))
	results, allSuccess := h.executeBatchActions(ctx, request, batchReq.Actions)

	l.Trace("Batch action request completed",
		logger_domain.Int("total_actions", len(batchReq.Actions)),
		logger_domain.Bool("all_success", allSuccess),
	)
	span.SetStatus(codes.Ok, "Batch request completed")
	h.writeJSON(w, http.StatusOK, daemon_dto.BatchActionResponse{Results: results, Success: allSuccess})
}

// trackBatchMetrics records metrics for a batch action request.
//
// Takes request (*http.Request) which provides the HTTP method for metric labels.
func (*ActionHandler) trackBatchMetrics(ctx context.Context, request *http.Request) {
	actionRequestCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String(attributeKeyAction, "_batch"), attribute.String("method", request.Method)),
	)
}

// executeBatchActions executes all actions in the batch and returns results.
//
// Takes request (*http.Request) which provides the original request context.
// Takes actions ([]daemon_dto.BatchActionItem) which contains the actions to
// execute.
//
// Returns []daemon_dto.BatchActionResult which contains the result for each
// action in the same order as the input.
// Returns bool which indicates whether all actions succeeded.
func (h *ActionHandler) executeBatchActions(ctx context.Context, request *http.Request, actions []daemon_dto.BatchActionItem) ([]daemon_dto.BatchActionResult, bool) {
	results := make([]daemon_dto.BatchActionResult, len(actions))
	allSuccess := true
	for i, item := range actions {
		results[i] = h.executeSingleAction(ctx, request, item)
		if results[i].Status >= httpErrorStatusThreshold {
			allSuccess = false
		}
	}
	return results, allSuccess
}

// executeSingleAction executes a single action within a batch request.
//
// Takes request (*http.Request) which provides the original HTTP request context.
// Takes item (daemon_dto.BatchActionItem) which specifies the action to run.
//
// Returns daemon_dto.BatchActionResult which contains the action outcome.
func (h *ActionHandler) executeSingleAction(
	ctx context.Context,
	request *http.Request,
	item daemon_dto.BatchActionItem,
) daemon_dto.BatchActionResult {
	entry, ok := h.registry[item.Name]
	if !ok {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusNotFound,
			Error:  fmt.Sprintf("action %q not found", item.Name),
			Code:   "NOT_FOUND",
		}
	}

	action := entry.Create()

	h.injectMetadata(request, action)

	if !h.checkBatchActionRateLimit(ctx, request, action, entry) {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusTooManyRequests,
			Error:  "Rate limit exceeded",
			Code:   "RATE_LIMITED",
		}
	}

	arguments := item.Args
	if arguments == nil {
		arguments = make(map[string]any)
	}

	if captchaErr := h.validateCaptcha(ctx, request, action, arguments, item.Name); captchaErr != nil {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusForbidden,
			Error:  "Captcha validation failed",
			Code:   "CAPTCHA_FAILED",
		}
	}

	if spamErr := h.validateSpamDetect(ctx, request, action, arguments, item.Name); spamErr != nil {
		return daemon_dto.BatchActionResult{
			Name:   item.Name,
			Status: http.StatusForbidden,
			Error:  "Submission flagged by spam filter",
			Code:   "SPAM_DETECTED",
		}
	}

	result, err := entry.Invoke(ctx, action, arguments)
	if err != nil {
		return h.buildBatchErrorResult(item.Name, err, isDevelopmentModeFromContext(request.Context()))
	}

	return daemon_dto.BatchActionResult{
		Name:   item.Name,
		Status: http.StatusOK,
		Data:   result,
	}
}

// buildBatchErrorResult creates a BatchActionResult from an error.
//
// Takes name (string) which identifies the action that failed.
// Takes err (error) which is the error to convert.
// Takes developmentMode (bool) which controls whether internal error details
// are exposed in the response message.
//
// Returns daemon_dto.BatchActionResult which contains the error details with
// appropriate status code and error code extracted from ActionError, or a
// generic internal error if the error is not an ActionError.
func (*ActionHandler) buildBatchErrorResult(name string, err error, developmentMode bool) daemon_dto.BatchActionResult {
	if actionErr, ok := errors.AsType[daemon_dto.ActionError](err); ok {
		return daemon_dto.BatchActionResult{
			Name:   name,
			Status: actionErr.StatusCode(),
			Error:  err.Error(),
			Code:   actionErr.ErrorCode(),
		}
	}

	return daemon_dto.BatchActionResult{
		Name:   name,
		Status: http.StatusInternalServerError,
		Error:  safeerror.ExtractSafeMessage(err, developmentMode),
		Code:   "INTERNAL_ERROR",
	}
}

// writeJSON writes a JSON response.
//
// Takes w (http.ResponseWriter) which receives the JSON output.
// Takes status (int) which sets the HTTP status code.
// Takes data (any) which is the value to encode as JSON, or nil to skip.
func (*ActionHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.ConfigDefault.NewEncoder(w).Encode(data)
	}
}

// writeError writes an error response.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
// Takes status (int) which specifies the HTTP status code.
// Takes message (string) which provides the error message for clients.
// Takes err (error) which contains the underlying error details, or nil.
func (h *ActionHandler) writeError(w http.ResponseWriter, status int, message string, err error) {
	response := map[string]any{"error": message}
	if err != nil {
		response["details"] = err.Error()
	}
	h.writeJSON(w, status, response)
}

// handleActionError processes an action error and writes the appropriate
// response.
//
// Takes w (http.ResponseWriter) which receives the JSON error response.
// Takes request (*http.Request) which provides the request context for
// development mode detection.
// Takes action (any) which provides response metadata and optional helpers.
// Takes err (error) which is the error to process.
//
// It discriminates between structured ActionError types and generic errors.
// If the action set helpers before returning an error, they are included
// in the response.
func (h *ActionHandler) handleActionError(w http.ResponseWriter, request *http.Request, action any, err error) {
	h.applyResponseMetadata(w, action)

	var helpers []daemon_dto.HelperCall
	if getter, ok := action.(interface {
		Response() *daemon_dto.ResponseWriter
	}); ok {
		if response := getter.Response(); response != nil {
			helpers = response.GetHelpers()
		}
	}

	if actionErr, ok := errors.AsType[daemon_dto.ActionError](err); ok {
		response := map[string]any{
			"status":  actionErr.StatusCode(),
			"code":    actionErr.ErrorCode(),
			"message": err.Error(),
		}

		if ve, ok := errors.AsType[*daemon_dto.ValidationError](err); ok {
			response["errors"] = ve.Fields
		}

		if len(helpers) > 0 {
			response["_helpers"] = helpers
		}

		h.writeJSON(w, actionErr.StatusCode(), response)
		return
	}

	developmentMode := isDevelopmentModeFromContext(request.Context())
	message := safeerror.ExtractSafeMessage(err, developmentMode)

	response := map[string]any{
		"status":  http.StatusInternalServerError,
		"code":    "INTERNAL_ERROR",
		"message": message,
	}
	if len(helpers) > 0 {
		response["_helpers"] = helpers
	}
	h.writeJSON(w, http.StatusInternalServerError, response)
}

// buildActionRateLimitOverride extracts rate limit configuration from an action
// that implements RateLimitable. Returns nil when the action has no rate limit
// or when rate limiting is disabled.
//
// Takes action (any) which may implement daemon_domain.RateLimitable.
// Takes entry (ActionHandlerEntry) which identifies the action for keying.
//
// Returns *security_dto.RateLimitOverride which contains the resolved rate
// limit settings, or nil when no limit applies.
func (h *ActionHandler) buildActionRateLimitOverride(
	_ *http.Request,
	action any,
	entry ActionHandlerEntry,
) *security_dto.RateLimitOverride {
	if h.rateLimitMw == nil {
		return nil
	}

	rateLimitable, ok := action.(daemon_domain.RateLimitable)
	if !ok {
		return nil
	}

	rl := rateLimitable.RateLimit()
	if rl == nil {
		return nil
	}

	keySuffix := entry.Name
	if rl.KeyFunc != nil {
		if getter, ok := action.(interface {
			Request() *daemon_dto.RequestMetadata
		}); ok {
			if reqMeta := getter.Request(); reqMeta != nil {
				keySuffix = entry.Name + ":" + rl.KeyFunc(reqMeta)
			}
		}
	}

	return &security_dto.RateLimitOverride{
		KeySuffix:         keySuffix,
		RequestsPerMinute: rl.RequestsPerMinute,
		BurstSize:         rl.BurstSize,
	}
}

// checkRateLimit enforces per-action rate limiting using the action's
// RateLimitable interface. Returns true if the request is allowed, false if
// rate limited (429 response already written).
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

// checkBatchActionRateLimit enforces per-action rate limiting within a batch
// request. Unlike checkRateLimit, this does not write headers or a response
// body since batch results are collected and returned as a single JSON
// response.
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

// buildCacheKey constructs a cache key for a cacheable action response.
//
// Takes request (*http.Request) which provides headers for VaryHeaders.
// Takes arguments (map[string]any) which holds the parsed request arguments.
// Takes actionName (string) which identifies the action.
// Takes cc (*daemon_domain.CacheConfig) which provides the cache configuration.
//
// Returns string which is the computed cache key.
func (*ActionHandler) buildCacheKey(
	request *http.Request,
	arguments map[string]any,
	actionName string,
	cc *daemon_domain.CacheConfig,
) string {
	argsJSON, _ := json.Marshal(arguments)

	var b strings.Builder
	b.WriteString(actionName)
	b.WriteByte(':')
	b.Write(argsJSON)

	for _, header := range cc.VaryHeaders {
		b.WriteByte(':')
		b.WriteString(header)
		b.WriteByte('=')
		b.WriteString(request.Header.Get(header))
	}
	return b.String()
}

// recordSlowAction logs a warning and increments the slow action metric when
// the action execution exceeds the configured slow threshold.
//
// Takes actionName (string) which identifies the action.
// Takes startTime (time.Time) which marks when execution began.
// Takes threshold (time.Duration) which is the slow threshold; zero disables.
func (*ActionHandler) recordSlowAction(
	ctx context.Context,
	actionName string,
	startTime time.Time,
	threshold time.Duration,
) {
	ctx, l := logger_domain.From(ctx, log)

	if threshold <= 0 {
		return
	}

	elapsed := time.Since(startTime)
	if elapsed <= threshold {
		return
	}

	actionSlowCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String(attributeKeyAction, actionName)))
	l.Warn("Slow action execution detected",
		logger_domain.String(attributeKeyAction, actionName),
		logger_domain.String("elapsed", elapsed.String()),
		logger_domain.String("threshold", threshold.String()),
	)
}
