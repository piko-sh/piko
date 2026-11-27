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
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/json"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// httpErrorStatusThreshold is the minimum HTTP status code considered an error.
	httpErrorStatusThreshold = 400

	// httpMaxErrorStatus is the largest HTTP status code an action may answer with.
	httpMaxErrorStatus = 599

	// errorCodeInternal is the machine-readable code for a failure on the server side.
	errorCodeInternal = "INTERNAL_ERROR"

	// errorCodePayloadTooLarge is the machine-readable code for a request body that is
	// larger than the action accepts.
	errorCodePayloadTooLarge = "PAYLOAD_TOO_LARGE"

	// payloadTooLargeMessage is the message a client is shown for an oversized body.
	payloadTooLargeMessage = "Request body too large"

	// responseKeyStatus is the JSON key for the HTTP status in an error response.
	responseKeyStatus = "status"

	// responseKeyCode is the JSON key for the machine-readable code in an error response.
	responseKeyCode = "code"

	// responseKeyMessage is the JSON key for the message in an error response.
	responseKeyMessage = "message"

	// logFieldStatus is the log attribute key for the HTTP status a request is answered
	// with.
	logFieldStatus = "status"

	// logFieldCode is the log attribute key for the machine-readable error code.
	logFieldCode = "code"

	// logFieldInvalidStatus is the log attribute key for an action error status that was
	// outside 400 to 599 and so was replaced with 500.
	logFieldInvalidStatus = "invalid_status"

	// logFieldLimitBytes is the log attribute key for a body size limit, in bytes.
	logFieldLimitBytes = "limit_bytes"

	// maxErrorTreeDepth bounds how deep isRequestFault walks a tree of wrapped errors.
	maxErrorTreeDepth = 32

	// attributeKeyAction is the attribute key for action names in logs and metrics.
	attributeKeyAction = "action"

	// csrfEphemeralTokenKey is the key used for the ephemeral CSRF token in request
	// arguments (POST body) or query parameters (GET).
	csrfEphemeralTokenKey = "_csrf_ephemeral_token"

	// captchaTokenKey is the key used for the captcha token in request arguments. The token
	// is extracted and deleted before argument binding.
	captchaTokenKey = "_captcha_token"

	// headerCSRFActionToken is the HTTP header for the signed CSRF action token.
	headerCSRFActionToken = "X-CSRF-Action-Token"

	// defaultMaxMultipartBytes is the default maximum size (32 MiB) for multipart form data
	// when no explicit limit is configured.
	defaultMaxMultipartBytes = 32 << 20

	// maxBatchActions is the maximum number of actions allowed in a single batch request.
	// This bounds CPU and memory usage regardless of the body size limit.
	maxBatchActions = 100

	// defaultParallelBatchWorkers bounds how many entries of one parallel batch run at once
	// when an operator has set no bound. Without it a full batch on a busy server could
	// multiply the configured request concurrency by the batch size.
	defaultParallelBatchWorkers = 8

	// actionBatchComponent names the batch executor in panic recovery reports.
	actionBatchComponent = "daemon_adapters.actionBatch"

	// actionSingleComponent names the single action executor in panic recovery reports.
	actionSingleComponent = "daemon_adapters.actionSingle"

	// actionCacheKeyComponent names the application's cache key func in panic recovery
	// reports.
	actionCacheKeyComponent = "daemon_adapters.actionCacheKey"

	// batchActionName is the action attributed to a batch request's single analytics event.
	batchActionName = "_batch"

	// maxBatchActionsPropertyLen bounds the joined list of dispatched action names.
	maxBatchActionsPropertyLen = 1024

	// maxBatchAnalyticsProperties bounds the properties the batch attribution itself adds.
	maxBatchAnalyticsProperties = 3

	// defaultMaxJSONBodyDepth is the maximum nesting depth permitted when decoding action
	// request bodies. Pre-decoding the byte stream guards against stack-blow attacks before
	// the structural decoder allocates reflective storage.
	defaultMaxJSONBodyDepth = 32

	// maxQueryArgumentKeys bounds how many distinct query parameters are lifted into an
	// action's arguments.
	maxQueryArgumentKeys = 64

	// mediaTypeEventStream is the media type a client asks for to receive a stream.
	mediaTypeEventStream = "text/event-stream"

	// defaultSSEWriteTimeout bounds how long a single SSE write may block before the
	// underlying connection is forcibly closed. The deadline is reset before every write so
	// a healthy client can stream indefinitely while a slow-loris peer is dropped within
	// this window.
	defaultSSEWriteTimeout = 5 * time.Second
)

var (
	// errJSONDepthExceeded is returned when a request body's structural nesting exceeds the
	// configured depth limit. It is wrapped with context before being surfaced to callers.
	errJSONDepthExceeded = errors.New("JSON nesting depth exceeded")

	// actionErrorType lets isRequestFault tell an action error apart from an error that only
	// wraps one.
	actionErrorType = reflect.TypeFor[daemon_dto.ActionError]()
)

// ActionHandler is a generated-code friendly action handler that dispatches requests to
// actions using the generated registry.
//
// Actions are dispatched through generated wrapper functions that provide type-safe
// argument handling.
type ActionHandler struct {
	// csrfService validates CSRF tokens.
	csrfService security_domain.CSRFTokenService

	// responseCache stores cached action responses keyed by action name and request
	// characteristics. Nil when the cache hexagon is unavailable.
	responseCache cache_domain.Cache[string, []byte]

	// captchaService verifies captcha tokens. Nil when captcha is disabled.
	captchaService captcha_domain.CaptchaServicePort

	// spamdetectService analyses form content for spam. Nil when disabled.
	spamdetectService spamdetect_domain.SpamDetectServicePort

	// registry maps action names to their handler entries.
	registry map[string]ActionHandlerEntry

	// actionWarned records the misconfigurations already reported for an action, so each
	// warning is emitted once per action rather than once per request.
	actionWarned sync.Map

	// rateLimitMw applies per-action rate limiting. Nil when rate limiting is disabled
	// globally.
	rateLimitMw *rateLimitMiddleware

	// actionSlots holds one buffered channel per action that declares MaxConcurrent, used as
	// a semaphore.
	actionSlots sync.Map

	// maxBodyBytes is the maximum request body size in bytes.
	maxBodyBytes int64

	// defaultMaxSSEDuration is the default maximum lifetime for SSE connections. Zero means
	// unlimited; individual actions can override via ResourceLimits.
	defaultMaxSSEDuration time.Duration

	// requestTimeout bounds a single non-streaming action call when the action declares no
	// timeout of its own. Zero means unlimited.
	requestTimeout time.Duration

	// maxMultipartFormBytes is the maximum in-memory size for multipart form data.
	maxMultipartFormBytes int64

	// maxParallelBatchWorkers bounds how many entries of one parallel batch run at once; 0
	// uses defaultParallelBatchWorkers.
	maxParallelBatchWorkers int

	// compressResponses negotiates gzip or brotli for action responses.
	compressResponses bool

	// enforceSecFetchSite requires CSRF tokens on browser requests identified by the
	// Sec-Fetch-Site header.
	enforceSecFetchSite bool
}

// ActionHandlerEntry describes a registered action.
type ActionHandlerEntry struct {
	// Create returns a new action struct instance.
	Create func() any

	// Invoke calls the action with the given parsed arguments.
	Invoke func(ctx context.Context, action any, arguments map[string]any) (any, error)

	// Bind records the parsed arguments on the action without calling it. It is set only for
	// actions that stream, where Call never runs, and is nil for every other action.
	Bind func(ctx context.Context, action any, arguments map[string]any) error

	// Name is the action identifier in dot notation, used for routing and tracing.
	Name string

	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.).
	Method string

	// Middlewares contains handlers to apply to this action in sequence.
	Middlewares []func(http.Handler) http.Handler

	// HasSSE indicates if the action supports SSE streaming.
	HasSSE bool

	// SSEGetAlias registers a GET route alongside the action's declared method.
	//
	// A browser EventSource issues GET and can set no headers, so that route carries no CSRF
	// token and is exempt from the CSRF check. Declaring the alias is therefore an assertion
	// that the action is safe to reach cross-site, which a mutating action is not. It is off
	// unless the action opts in.
	SSEGetAlias bool
}

// NewActionHandler creates a new action handler.
//
// Takes csrfService (security_domain.CSRFTokenService) for CSRF validation.
// Takes maxBodyBytes (int64) which is the maximum request body size.
// Takes rateLimitService (security_domain.RateLimitService) for per-action rate limiting;
// may be nil when rate limiting is disabled.
// Takes rateLimitConfig (security_dto.RateLimitValues) which configures rate limit
// behaviour.
// Takes enforceSecFetchSite (bool) which requires CSRF tokens on browser requests
// identified by the Sec-Fetch-Site header.
// Takes responseCache (cache_domain.Cache[string, []byte]) which stores cached action
// responses; may be nil to disable action response caching.
// Takes captchaService (captcha_domain.CaptchaServicePort) for captcha verification; may
// be nil when captcha is disabled.
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
		registry:                make(map[string]ActionHandlerEntry),
		csrfService:             csrfService,
		maxBodyBytes:            maxBodyBytes,
		rateLimitMw:             rlMw,
		responseCache:           responseCache,
		captchaService:          captchaService,
		enforceSecFetchSite:     enforceSecFetchSite,
		spamdetectService:       nil,
		actionWarned:            sync.Map{},
		actionSlots:             sync.Map{},
		defaultMaxSSEDuration:   0,
		requestTimeout:          0,
		maxMultipartFormBytes:   0,
		maxParallelBatchWorkers: 0,
		compressResponses:       false,
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
// Takes basePath (string) which is the base path for actions (e.g., "/_piko/actions").
func (h *ActionHandler) Mount(r chi.Router, basePath string) {
	compress := actionCompressMiddleware(h.compressResponses)

	for _, entry := range h.registry {
		routePattern := fmt.Sprintf("%s/%s", basePath, entry.Name)

		handler := h.createHandler(entry)

		for _, middleware := range slices.Backward(entry.Middlewares) {
			handler = middleware(handler)
		}

		handler = compress(handler)

		r.Method(entry.Method, routePattern, handler)

		if entry.HasSSE && entry.SSEGetAlias && entry.Method != http.MethodGet {
			r.Method(http.MethodGet, routePattern, handler)
		}
	}

	r.Method(http.MethodPost, fmt.Sprintf("%s/_batch", basePath), compress(http.HandlerFunc(h.handleBatch)))
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

	ctx, l := logger_domain.From(ctx, log)

	if pctx := daemon_dto.PikoRequestCtxFromContext(request.Context()); pctx != nil {
		pctx.SetAnalyticsAction(entry.Name)
	}

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
// Returns bool which is true when the entry supports SSE and the request accepts
// text/event-stream.
func (*ActionHandler) shouldUseSSE(request *http.Request, entry ActionHandlerEntry) bool {
	if !entry.HasSSE {
		return false
	}
	return security_dto.AcceptsMediaType(request.Header.Get("Accept"), mediaTypeEventStream)
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
	prepared, release, ok := h.prepareAction(ctx, w, request, entry)
	if !ok {
		return
	}
	defer release()

	ctx, _ = logger_domain.From(prepared.Ctx, log)

	if h.handleCachedAction(ctx, w, prepared.Request, cachedActionParams{
		Action:          prepared.Action,
		Entry:           entry,
		Args:            prepared.Arguments,
		Span:            span,
		StartTime:       prepared.StartTime,
		SlowThreshold:   prepared.Limits.SlowThreshold,
		MaxResponseSize: prepared.Limits.MaxResponseSize,
	}) {
		return
	}

	result, err := goroutine.SafeCall1(ctx, actionSingleComponent, func() (any, error) {
		return entry.Invoke(ctx, prepared.Action, prepared.Arguments)
	})
	if err != nil {
		reportActionFailure(ctx, span, entry.Name, err, "Action execution failed")
		h.handleActionError(w, prepared.Request, prepared.Action, err)
		h.recordSlowAction(ctx, entry.Name, prepared.StartTime, prepared.Limits.SlowThreshold)

		return
	}

	h.applyResponseMetadata(w, prepared.Action)
	response := h.buildFullResponse(prepared.Action, result)
	if !h.writeActionResponse(ctx, w, entry.Name, response, prepared.Limits.MaxResponseSize) {
		h.recordSlowAction(ctx, entry.Name, prepared.StartTime, prepared.Limits.SlowThreshold)

		return
	}
	span.SetStatus(codes.Ok, "Action completed successfully")
	h.recordSlowAction(ctx, entry.Name, prepared.StartTime, prepared.Limits.SlowThreshold)
}

// isSSEGetAliasRequest reports whether a request arrived over an action's opt-in GET
// stream alias.
//
// Takes request (*http.Request) which supplies the method and Accept header.
// Takes entry (ActionHandlerEntry) which says whether the alias was opted into.
//
// Returns bool which is true when the request came over the alias as a stream.
func isSSEGetAliasRequest(request *http.Request, entry ActionHandlerEntry) bool {
	return entry.HasSSE &&
		entry.SSEGetAlias &&
		request.Method == http.MethodGet &&
		security_dto.AcceptsMediaType(request.Header.Get("Accept"), mediaTypeEventStream)
}

// runSecurityValidation performs CSRF, rate limit, and captcha checks for an incoming
// action request. Returns true when all checks pass; returns false when a check fails, in
// which case an error response has already been written.
//
// Takes w (http.ResponseWriter) which receives any error responses.
// Takes request (*http.Request) which contains headers for validation.
// Takes action (any) which may carry rate limit or captcha configuration.
// Takes arguments (map[string]any) which holds parsed request arguments.
// Takes entry (ActionHandlerEntry) which identifies the action for logging.
//
// Returns bool which is true when all security checks pass, or false when a check fails
// and an error response has already been written.
func (h *ActionHandler) runSecurityValidation(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	action any,
	arguments map[string]any,
	entry ActionHandlerEntry,
) bool {
	ctx, l := logger_domain.From(ctx, log)

	if !isSSEGetAliasRequest(request, entry) {
		if csrfErr := h.validateCSRF(request, arguments); csrfErr != nil {
			l.Warn("CSRF validation failed",
				logger_domain.String(attributeKeyAction, entry.Name),
				logger_domain.Error(csrfErr),
			)
			h.writeCSRFError(w, csrfErr)

			return false
		}
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

// writeActionResponse writes an action's response, refusing one that exceeds the action's
// declared size limit.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes actionName (string) which names the action in the log.
// Takes response (any) which is the value to encode.
// Takes maxResponseSize (int64) which is the limit, or zero for none.
//
// Returns bool which is false when the response was refused and an error written.
func (h *ActionHandler) writeActionResponse(
	ctx context.Context,
	w http.ResponseWriter,
	actionName string,
	response any,
	maxResponseSize int64,
) bool {
	if maxResponseSize <= 0 {
		h.writeJSON(w, http.StatusOK, response)

		return true
	}

	encoded, ok := h.encodeActionResponse(ctx, w, actionName, response, maxResponseSize)
	if !ok {
		return false
	}

	setActionJSONHeaders(w.Header())
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)

	return true
}

// encodeActionResponse encodes a response and refuses one larger than the action allows.
//
// Takes w (http.ResponseWriter) which receives the error response on failure.
// Takes actionName (string) which names the action in logs.
// Takes response (any) which is the value to encode.
// Takes maxResponseSize (int64) which bounds the encoded size, or zero for no bound.
//
// Returns []byte which is the encoded response.
// Returns bool which is false when an error response has already been written.
func (h *ActionHandler) encodeActionResponse(
	ctx context.Context,
	w http.ResponseWriter,
	actionName string,
	response any,
	maxResponseSize int64,
) ([]byte, bool) {
	ctx, l := logger_domain.From(ctx, log)

	encoded, err := json.ConfigDefault.Marshal(response)
	if err != nil {
		l.Error("Failed to encode the action response",
			logger_domain.String(attributeKeyAction, actionName),
			logger_domain.Error(err),
		)
		h.writeError(w, http.StatusInternalServerError, "Failed to encode response", err,
			isDevelopmentModeFromContext(ctx))

		return nil, false
	}

	if maxResponseSize > 0 && int64(len(encoded)) > maxResponseSize {
		l.Error("Action response exceeds its declared size limit",
			logger_domain.String(attributeKeyAction, actionName),
			logger_domain.Int("response_bytes", len(encoded)),
			logger_domain.Int64("max_response_bytes", maxResponseSize),
		)
		h.writeError(w, http.StatusInternalServerError, "Response too large", nil,
			isDevelopmentModeFromContext(ctx))

		return nil, false
	}

	return encoded, true
}

// acquireActionSlot claims one of an action's concurrent execution slots.
//
// Takes actionName (string) which names the action the slot belongs to.
// Takes maxConcurrent (int) which is the bound, or zero for no bound.
//
// Returns func() which releases the slot, and is never nil when admitted.
// Returns bool which is false when the action is already at its bound.
func (h *ActionHandler) acquireActionSlot(actionName string, maxConcurrent int) (func(), bool) {
	if maxConcurrent <= 0 {
		return func() {}, true
	}

	slots, found := h.actionSlots.Load(actionName)
	if !found {
		slots, _ = h.actionSlots.LoadOrStore(actionName, make(chan struct{}, maxConcurrent))
	}

	semaphore, ok := slots.(chan struct{})
	if !ok {
		return func() {}, true
	}

	select {
	case semaphore <- struct{}{}:
		return func() { <-semaphore }, true
	default:
		return nil, false
	}
}

// writeConcurrencyLimitError reports that an action is already running as many calls as
// it allows.
//
// Takes w (http.ResponseWriter) which receives the response.
func (h *ActionHandler) writeConcurrencyLimitError(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	h.writeJSON(w, http.StatusServiceUnavailable, map[string]any{
		"error":            "concurrency_limit",
		responseKeyMessage: "This action is handling as many requests as it allows. Retry shortly.",
	})
}

// preparedAction holds everything both transports need once a request has been admitted.
type preparedAction struct {
	// StartTime marks when the request began, for slow-action reporting.
	StartTime time.Time

	// Ctx is the request context after any limit has been applied. It replaces the caller's
	// context for the rest of the request.
	Ctx context.Context

	// Action is the instance the transport dispatches to.
	Action any

	// Request is the request re-bound to Ctx, with its body already capped.
	Request *http.Request

	// Arguments holds the parsed request arguments.
	Arguments map[string]any

	// Limits holds the resource limits resolved for this action.
	Limits actionLimits
}

// prepareAction runs every step both transports share, in the order both need.
//
// Takes w (http.ResponseWriter) which receives any rejection response.
// Takes request (*http.Request) which is the incoming request.
// Takes entry (ActionHandlerEntry) which creates the action and names it.
//
// Returns preparedAction which holds the admitted request, valid only when ok is true.
// Returns func() which releases the action's concurrency slot.
// Returns bool which is false when a response has already been written.
func (h *ActionHandler) prepareAction(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	entry ActionHandlerEntry,
) (preparedAction, func(), bool) {
	ctx, _ = logger_domain.From(ctx, log)

	action := entry.Create()
	prepared := preparedAction{
		Action:    action,
		StartTime: time.Now(),
		Ctx:       nil,
		Request:   nil,
		Arguments: nil,
		Limits:    actionLimits{},
	}
	prepared.Limits = h.resolveActionLimits(action)

	releaseSlot, admitted := h.acquireActionSlot(entry.Name, prepared.Limits.MaxConcurrent)
	if !admitted {
		h.writeConcurrencyLimitError(w)

		return preparedAction{}, nil, false
	}

	ctx, request, cancelDeadline := h.applyTransportDeadline(ctx, request, entry, prepared.Limits)
	release := func() {
		cancelDeadline()
		releaseSlot()
	}

	h.injectMetadata(request, action)
	request.Body = http.MaxBytesReader(baseResponseWriter(w), request.Body, prepared.Limits.BodyLimit)

	arguments, err := h.parseRequestBody(request)
	if err != nil {
		release()
		h.writeBodyError(ctx, w, err, "Invalid request body")

		return preparedAction{}, nil, false
	}

	if isSSEGetAliasRequest(request, entry) {
		arguments = queryArguments(request)
	}

	if !h.runSecurityValidation(ctx, w, request, action, arguments, entry) {
		release()

		return preparedAction{}, nil, false
	}

	prepared.Ctx = ctx
	prepared.Request = request
	prepared.Arguments = arguments

	return prepared, release, true
}

// writeBodyError answers a request whose body could not be read or decoded.
//
// A body larger than the limit answers 413 PAYLOAD_TOO_LARGE, and any other failure
// answers 400. Both are the client's fault, so they are logged at info and never
// reported. The 413 log names the limit but not the body.
//
// Takes w (http.ResponseWriter) which receives the error response.
// Takes err (error) which is the read or decode failure.
// Takes message (string) which is the message a 400 answer carries.
func (h *ActionHandler) writeBodyError(ctx context.Context, w http.ResponseWriter, err error, message string) {
	ctx, l := logger_domain.From(ctx, log)

	if maxBytesErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
		l.Info("Request body is larger than the limit",
			logger_domain.Int64(logFieldLimitBytes, maxBytesErr.Limit),
		)
		h.writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{
			responseKeyStatus:  http.StatusRequestEntityTooLarge,
			responseKeyCode:    errorCodePayloadTooLarge,
			responseKeyMessage: payloadTooLargeMessage,
		})
		return
	}

	l.Info("Request body could not be parsed", logger_domain.Error(err))
	h.writeError(w, http.StatusBadRequest, message, err, isDevelopmentModeFromContext(ctx))
}

// actionLimits holds the resource limits that apply to one action call.
type actionLimits struct {
	// BodyLimit is the largest request body the action accepts, in bytes.
	BodyLimit int64

	// Timeout bounds a single call, or zero when the action declares none.
	Timeout time.Duration

	// SlowThreshold is the duration past which a call is recorded as slow.
	SlowThreshold time.Duration

	// MaxSSEDuration bounds a stream from this action, or zero to take the handler default.
	MaxSSEDuration time.Duration

	// MaxResponseSize bounds the encoded response, or zero for no bound.
	MaxResponseSize int64

	// MaxConcurrent bounds how many calls to this action may run at once, or zero for no
	// per-action bound.
	MaxConcurrent int
}

// applyTransportDeadline bounds a request according to the transport it is answered over.
//
// A streaming request lives as long as the stream does, so it takes the SSE duration
// rather than the call timeout. Applying the call timeout would cut every stream at it
// and put any longer streaming limit out of reach. A JSON request takes the call timeout.
//
// Takes request (*http.Request) which is rebound to the returned context.
// Takes entry (ActionHandlerEntry) which says whether this call streams.
// Takes limits (actionLimits) which supplies the durations.
//
// Returns context.Context which carries the deadline, if any.
// Returns *http.Request which is the request bound to that context.
// Returns context.CancelFunc which releases the deadline, and is never nil.
func (h *ActionHandler) applyTransportDeadline(
	ctx context.Context,
	request *http.Request,
	entry ActionHandlerEntry,
	limits actionLimits,
) (context.Context, *http.Request, context.CancelFunc) {
	if h.shouldUseSSE(request, entry) {
		ctx, request, cancel := h.applySSEDurationLimit(ctx, request, limits)
		if cancel == nil {
			cancel = func() {}
		}

		return ctx, request, cancel
	}

	if limits.Timeout <= 0 {
		return ctx, request.WithContext(ctx), func() {}
	}

	ctx, cancel := context.WithTimeoutCause(ctx, limits.Timeout,
		fmt.Errorf("action execution exceeded %s timeout", limits.Timeout))

	return ctx, request.WithContext(ctx), cancel
}

// resolveActionLimits reads the resource limits an action declares.
//
// Takes action (any) which may implement daemon_domain.ResourceLimitable.
//
// Returns actionLimits which holds the handler defaults with any action override applied.
func (h *ActionHandler) resolveActionLimits(action any) actionLimits {
	limits := actionLimits{
		BodyLimit:       h.maxBodyBytes,
		Timeout:         h.requestTimeout,
		SlowThreshold:   0,
		MaxSSEDuration:  0,
		MaxResponseSize: 0,
		MaxConcurrent:   0,
	}

	limitable, ok := action.(daemon_domain.ResourceLimitable)
	if !ok {
		return limits
	}

	declared := limitable.ResourceLimits()
	if declared == nil {
		return limits
	}

	if declared.MaxRequestBodySize > 0 {
		limits.BodyLimit = declared.MaxRequestBodySize
	}
	limits.Timeout = cmp.Or(declared.Timeout, limits.Timeout)
	limits.SlowThreshold = declared.SlowThreshold
	limits.MaxSSEDuration = declared.MaxSSEDuration
	limits.MaxConcurrent = declared.MaxConcurrent
	limits.MaxResponseSize = declared.MaxResponseSize

	return limits
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

	// MaxResponseSize bounds the encoded response, or zero for no bound. It applies before
	// the response is cached, so an over-sized body is never stored and then served to every
	// later caller.
	MaxResponseSize int64
}

// handleCachedAction checks whether the action is cacheable and, if so, serves the
// response from cache or invokes the action and caches the result.
// Returns true when the response has been written (cache hit or miss with successful
// invocation).
//
// Takes w (http.ResponseWriter) which receives the response output.
// Takes request (*http.Request) which provides headers for cache key computation.
// Takes p (cachedActionParams) which groups the action, entry, arguments, span, start
// time, and slow threshold.
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

	ctx, _ = logger_domain.From(ctx, log)
	cacheKey := h.buildCacheKey(request, p.Action, p.Args, p.Entry, cc)

	if cached, found, _ := h.responseCache.GetIfPresent(ctx, cacheKey); found {
		actionCacheHitCount.Add(ctx, 1,
			metric.WithAttributes(attribute.String(attributeKeyAction, p.Entry.Name)))
		setActionJSONHeaders(w.Header())
		w.Header().Set("X-Action-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(cached)
		p.Span.SetStatus(codes.Ok, "Cache hit")
		h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)
		return true
	}

	actionCacheMissCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String(attributeKeyAction, p.Entry.Name)))

	result, invokeErr := goroutine.SafeCall1(ctx, actionSingleComponent, func() (any, error) {
		return p.Entry.Invoke(ctx, p.Action, p.Args)
	})
	if invokeErr != nil {
		reportActionFailure(ctx, p.Span, p.Entry.Name, invokeErr, "Action execution failed")
		h.handleActionError(w, request, p.Action, invokeErr)
		h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)
		return true
	}

	h.storeAndWriteCachedResponse(ctx, w, cacheKey, cc.TTL, result, p)

	return true
}

// storeAndWriteCachedResponse caches an action's response and sends it.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes cacheKey (string) which identifies the cache entry.
// Takes ttl (time.Duration) which is how long the entry lives.
// Takes result (any) which is the action's return value.
// Takes p (cachedActionParams) which carries the action and its limits.
func (h *ActionHandler) storeAndWriteCachedResponse(
	ctx context.Context,
	w http.ResponseWriter,
	cacheKey string,
	ttl time.Duration,
	result any,
	p cachedActionParams,
) {
	ctx, l := logger_domain.From(ctx, log)

	h.applyResponseMetadata(w, p.Action)

	response := h.buildFullResponse(p.Action, result)

	encoded, encodedOK := h.encodeActionResponse(ctx, w, p.Entry.Name, response, p.MaxResponseSize)
	if !encodedOK {
		h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)

		return
	}

	if err := h.responseCache.SetWithTTL(ctx, cacheKey, encoded, ttl); err != nil {
		l.Warn("Failed to cache the action response",
			logger_domain.String(attributeKeyAction, p.Entry.Name),
			logger_domain.Error(err),
		)
	}

	w.Header().Set("X-Action-Cache", "MISS")
	setActionJSONHeaders(w.Header())
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
	p.Span.SetStatus(codes.Ok, "Action completed, response cached")
	h.recordSlowAction(ctx, p.Entry.Name, p.StartTime, p.SlowThreshold)
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
	prepared, release, ok := h.prepareAction(ctx, w, request, entry)
	if !ok {
		return
	}
	defer release()

	ctx = prepared.Ctx

	if !h.bindSSEInput(ctx, w, prepared.Request, prepared.Action, prepared.Arguments, entry, span) {
		h.recordSlowAction(ctx, entry.Name, prepared.StartTime, prepared.Limits.SlowThreshold)

		return
	}

	h.applyResponseMetadata(w, prepared.Action)
	h.writeSSEHeaders(w)

	h.executeSSEStream(ctx, w, prepared.Request, prepared.Action, entry, span)
	h.recordSlowAction(ctx, entry.Name, prepared.StartTime, prepared.Limits.SlowThreshold)
}

// bindSSEInput records the request's arguments on a streaming action.
//
// Actions with no generated bind function are left alone.
//
// Takes w (http.ResponseWriter) which receives an error response when binding fails.
// Takes request (*http.Request) which is named in the error response.
// Takes action (any) which is the action instance the arguments are recorded on.
// Takes arguments (map[string]any) which the shared prologue already parsed.
// Takes entry (ActionHandlerEntry) which supplies the generated bind function.
// Takes span (trace.Span) which records the failure.
//
// Returns bool which is true when binding succeeded, or false when an error response has
// already been written.
func (h *ActionHandler) bindSSEInput(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	action any,
	arguments map[string]any,
	entry ActionHandlerEntry,
	span trace.Span,
) bool {
	if entry.Bind == nil {
		return true
	}

	ctx, _ = logger_domain.From(ctx, log)

	if bindErr := entry.Bind(ctx, action, arguments); bindErr != nil {
		reportActionFailure(ctx, span, entry.Name, bindErr, "SSE input binding failed")
		h.handleActionError(w, request, action, bindErr)

		return false
	}

	return true
}

// writeSSEHeaders sets the standard SSE response headers. The write deadline is
// intentionally left under the caller's control so each write can apply
// defaultSSEWriteTimeout, dropping slow-loris peers without truncating long-lived streams
// used by healthy clients.
//
// Takes w (http.ResponseWriter) which receives the SSE headers.
func (*ActionHandler) writeSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// applySSEDurationLimit bounds how long a stream may run.
//
// The caller must call the returned cancel function when it is non-nil.
//
// Takes request (*http.Request) which is rebound to the returned context.
// Takes limits (actionLimits) which supplies the action's streaming duration.
//
// Returns context.Context which may carry the duration limit.
// Returns *http.Request which is bound to that context.
// Returns context.CancelFunc which releases the limit, or nil when none applies.
func (h *ActionHandler) applySSEDurationLimit(
	ctx context.Context,
	request *http.Request,
	limits actionLimits,
) (context.Context, *http.Request, context.CancelFunc) {
	sseDuration := cmp.Or(limits.MaxSSEDuration, h.defaultMaxSSEDuration)
	if sseDuration <= 0 {
		return ctx, request, nil
	}

	ctx, cancel := context.WithTimeoutCause(ctx, sseDuration,
		fmt.Errorf("SSE stream exceeded %s duration limit", sseDuration))

	return ctx, request.WithContext(ctx), cancel
}

// executeSSEStream runs the SSE stream on the given action.
//
// Takes w (http.ResponseWriter) which receives the SSE events.
// Takes request (*http.Request) which provides the client context.
// Takes action (any) which must implement daemon_domain.SSECapable.
// Takes entry (ActionHandlerEntry) which identifies the action.
// Takes span (trace.Span) which records the operation status.
//
// Concurrency: a watcher goroutine derived from ctx via WithCancelCause closes the
// disconnect channel when the request context ends. The deferred cancel guarantees the
// watcher exits as soon as StreamProgress returns, even when the request context outlives
// the handler.
func (*ActionHandler) executeSSEStream(
	ctx context.Context,
	w http.ResponseWriter,
	request *http.Request,
	action any,
	entry ActionHandlerEntry,
	span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	watchCtx, cancelWatch := context.WithCancelCause(ctx)
	defer cancelWatch(errors.New("sse handler returned"))

	done := make(chan struct{})
	go func() {
		defer goroutine.RecoverPanic(watchCtx, "daemon.sseDisconnectWatcher")
		<-watchCtx.Done()
		close(done)
	}()

	sseCapable, ok := action.(daemon_domain.SSECapable)
	if !ok {
		l.Error("Action does not implement SSECapable", logger_domain.String(attributeKeyAction, entry.Name))
		return
	}

	deadlineWriter := newSSEDeadlineWriter(w, defaultSSEWriteTimeout)

	lastEventID := request.Header.Get("Last-Event-ID")
	stream := daemon_domain.NewSSEStream(deadlineWriter, done, lastEventID)
	if stream == nil {
		l.Error("Response writer does not support flushing for SSE", logger_domain.String(attributeKeyAction, entry.Name))
		return
	}
	stream.SetDevelopmentModeFromContext(ctx)

	if err := sseCapable.StreamProgress(stream); err != nil {
		reportActionFailure(ctx, span, entry.Name, err, "SSE streaming failed")

		if sendErr := stream.SendError(err); sendErr != nil {
			l.Warn("Failed to report the streaming failure to the client",
				logger_domain.String(attributeKeyAction, entry.Name),
				logger_domain.Error(sendErr),
			)
		}

		return
	}

	span.SetStatus(codes.Ok, "SSE streaming completed")
}

// newRequestMetadata builds the metadata an action reads about its request.
//
// Takes request (*http.Request) which provides the HTTP request details.
//
// Returns *daemon_dto.RequestMetadata which describes the request.
func newRequestMetadata(request *http.Request) *daemon_dto.RequestMetadata {
	return &daemon_dto.RequestMetadata{
		Method:          request.Method,
		Path:            request.URL.Path,
		Headers:         request.Header,
		QueryParams:     request.URL.Query(),
		RemoteAddr:      request.RemoteAddr,
		RawRequest:      request,
		Session:         nil,
		CSRFToken:       nil,
		CaptchaScore:    nil,
		SpamScore:       nil,
		SpamFieldScores: nil,
		SpamReasons:     nil,
	}
}

// injectMetadata injects request metadata into the action and gives it a fresh response
// writer.
//
// Takes request (*http.Request) which provides the HTTP request details.
// Takes action (any) which is the action to inject metadata into.
func (*ActionHandler) injectMetadata(request *http.Request, action any) {
	type metadataInjector interface {
		SetRequest(request *daemon_dto.RequestMetadata)
		SetResponse(response *daemon_dto.ResponseWriter)
	}

	if injector, ok := action.(metadataInjector); ok {
		injector.SetRequest(newRequestMetadata(request))
		injector.SetResponse(daemon_dto.NewResponseWriter())
	}
}

// applyResponseMetadata applies cookies and headers from the action's response writer.
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

// buildFullResponse builds the response, wrapping with helpers if present. If there are
// no helpers, returns the raw result to minimise response size.
//
// Takes action (any) which may provide a response with helpers.
// Takes result (any) which is the raw result to wrap or return.
//
// Returns any which is either the raw result or a wrapped response with helpers.
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

// isRequestFault reports whether err describes the request rather than the server.
//
// A request fault answers with a 4xx status. Every failure inside err must agree. Each
// leaf of the error tree is an action error with a 4xx status, or a cancellation because
// the client went away. An action that answers piko.NotFound after a database failure is
// still a server fault, so the database failure is reported. A deadline is a server fault
// too, since the server ran out of time.
//
// Reporting a request fault as a server error would attach a stack trace and mark the
// span failed, which lets any caller drive the error rate and the log volume by sending
// bad input.
//
// Takes err (error) which is the failure returned by an action invocation.
//
// Returns bool which is true when the request, not the server, is at fault.
func isRequestFault(ctx context.Context, err error) bool {
	if !isClientErrorStatus(extractErrorStatusCode(err)) {
		return false
	}
	return leavesAreRequestFaults(err, 0, requestCancellationCause(ctx))
}

// leavesAreRequestFaults reports whether every leaf of an error tree describes the
// request.
//
// An action error ends the walk, since it states the status it answers with. Below it, a
// validation error may wrap the validator's own error, which says nothing about who is at
// fault.
//
// Takes err (error) which is the node being checked.
// Takes depth (int) which is how deep the walk has gone, bounded by maxErrorTreeDepth.
// Takes cause (error) which is the request's own cancellation cause, or nil.
//
// Returns bool which is true when every leaf below err is a request fault.
func leavesAreRequestFaults(err error, depth int, cause error) bool {
	if err == nil || depth > maxErrorTreeDepth {
		return false
	}

	if reflect.TypeOf(err).Implements(actionErrorType) {
		actionErr, ok := errors.AsType[daemon_dto.ActionError](err)
		return ok && isClientErrorStatus(actionErr.StatusCode())
	}

	children := unwrapChildren(err)
	if len(children) == 0 {
		return errors.Is(err, context.Canceled)
	}
	return childrenAgree(children, depth, cause, leavesAreRequestFaults)
}

// isClientAbort reports whether a failure only says that the client went away.
//
// The request context must have been cancelled, err must not hold a deadline, and every
// leaf of err must be the cancellation or the request's own cancellation cause beside it.
// Any other error beside the cancellation is a real failure and is still reported. A
// deadline is not an abort, since the server ran out of time.
//
// Takes err (error) which is the failure returned by an action invocation.
//
// Returns bool which is true when the client, not the server, ended the request.
func isClientAbort(ctx context.Context, err error) bool {
	if !errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return leavesAreClientAborts(err, 0, requestCancellationCause(ctx))
}

// leavesAreClientAborts reports whether every leaf of an error tree describes a client
// that went away.
//
// Takes err (error) which is the node being checked.
// Takes depth (int) which is how deep the walk has gone, bounded by maxErrorTreeDepth.
// Takes cause (error) which is the request's own cancellation cause, or nil.
//
// Returns bool which is true when every leaf below err is part of the cancellation.
func leavesAreClientAborts(err error, depth int, cause error) bool {
	if err == nil || depth > maxErrorTreeDepth {
		return false
	}

	children := unwrapChildren(err)
	if len(children) == 0 {
		return errors.Is(err, context.Canceled)
	}
	return childrenAgree(children, depth, cause, leavesAreClientAborts)
}

// childrenAgree reports whether every child of one error node passes check.
//
// A node that wraps context.Canceled may also wrap the cause given to the cancellation,
// as in fmt.Errorf("%w: %w", ctx.Err(), context.Cause(ctx)). Only that exact cause is
// part of the cancellation and is skipped. Any other error beside it is checked, so a
// real failure that happens as the client leaves is still reported.
//
// Takes children ([]error) which are the errors one node wraps.
// Takes depth (int) which is how deep the node is.
// Takes cause (error) which is the request's own cancellation cause, or nil.
// Takes check (func(error, int, error) bool) which checks one child.
//
// Returns bool which is true when every child passes.
func childrenAgree(children []error, depth int, cause error, check func(error, int, error) bool) bool {
	besideCancellation := slices.ContainsFunc(children, isCancellationLeaf)
	for _, child := range children {
		if besideCancellation && isCancellationCause(child, cause) {
			continue
		}
		if !check(child, depth+1, cause) {
			return false
		}
	}
	return true
}

// requestCancellationCause returns the cause the request context was cancelled with.
//
// Returns error which is the custom cause, or nil when the context is live or was
// cancelled without one.
func requestCancellationCause(ctx context.Context) error {
	cause := context.Cause(ctx)
	if cause == nil || errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		return nil
	}
	return cause
}

// isCancellationLeaf reports whether err is a cancellation that wraps nothing.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true for context.Canceled itself.
func isCancellationLeaf(err error) bool {
	return len(unwrapChildren(err)) == 0 && errors.Is(err, context.Canceled)
}

// isCancellationCause reports whether err is the request's own cancellation cause.
//
// Takes err (error) which is the error to check.
// Takes cause (error) which is the request's cancellation cause, or nil.
//
// Returns bool which is true when err is a leaf matching cause.
func isCancellationCause(err error, cause error) bool {
	return cause != nil && len(unwrapChildren(err)) == 0 && errors.Is(err, cause)
}

// unwrapChildren returns the errors that err wraps directly, one level down.
//
// errors.As cannot be used here because it searches the whole tree, while the walk needs
// to know what this one node wraps.
//
// Takes err (error) which is the node to open.
//
// Returns []error which holds the wrapped errors, or nil when err wraps nothing.
func unwrapChildren(err error) []error {
	var node any = err
	switch wrapper := node.(type) {
	case interface{ Unwrap() []error }:
		return wrapper.Unwrap()
	case interface{ Unwrap() error }:
		if child := wrapper.Unwrap(); child != nil {
			return []error{child}
		}
	}
	return nil
}

// isClientErrorStatus reports whether status is a 4xx status.
//
// Takes status (int) which is the HTTP status to check.
//
// Returns bool which is true for 400 to 499.
func isClientErrorStatus(status int) bool {
	return status >= httpErrorStatusThreshold && status < http.StatusInternalServerError
}

// isSecurityRefusal reports whether status denotes refused credentials, access, or a rate
// limit.
//
// Takes status (int) which is the HTTP status to check.
//
// Returns bool which is true for 401, 403 and 429.
func isSecurityRefusal(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusTooManyRequests
}

// reportActionFailure logs a failed action once, at the level its fault deserves.
//
// A client that went away is logged at debug and nothing more. A panic was already logged
// with its stack where it was recovered, so it only marks the span failed. Any other
// server fault is reported as an error with its stack, names a status outside 400 to 599
// when the action chose one, and marks the span failed. A request fault is logged at
// info, or at warn for a security refusal, with the action, status and code only, so no
// value the client sent reaches the logs.
//
// Takes span (trace.Span) which is marked failed for a server fault.
// Takes actionName (string) which names the action in the log.
// Takes err (error) which is the failure.
// Takes message (string) which describes where the failure happened.
func reportActionFailure(ctx context.Context, span trace.Span, actionName string, err error, message string) {
	ctx, l := logger_domain.From(ctx, log)

	if isClientAbort(ctx, err) {
		l.Debug("Action stopped because the client went away", logger_domain.String(attributeKeyAction, actionName))
		return
	}

	if _, panicked := errors.AsType[*goroutine.PanicError](err); panicked {
		markSpanFailed(span, err, message)
		return
	}

	if !isRequestFault(ctx, err) {
		l.ReportError(span, err, message, serverFaultAttrs(actionName, err)...)
		return
	}

	status, code, _ := resolveActionErrorAnswer(err)
	attrs := []logger_domain.Attr{
		logger_domain.String(attributeKeyAction, actionName),
		logger_domain.Int(logFieldStatus, status),
		logger_domain.String(logFieldCode, code),
	}
	if isSecurityRefusal(status) {
		l.Warn("Action refused the request", attrs...)
		return
	}
	l.Info("Action refused the request", attrs...)
}

// serverFaultAttrs builds the attributes a reported server fault is logged with.
//
// An action error whose status is outside 400 to 599 is a bug in the action, so the
// status it chose is named.
//
// Takes actionName (string) which names the action.
// Takes err (error) which is the failure.
//
// Returns []logger_domain.Attr which holds the action and any invalid status.
func serverFaultAttrs(actionName string, err error) []logger_domain.Attr {
	attrs := []logger_domain.Attr{logger_domain.String(attributeKeyAction, actionName)}
	if _, _, valid := resolveActionErrorAnswer(err); !valid {
		attrs = append(attrs, logger_domain.Int(logFieldInvalidStatus, extractErrorStatusCode(err)))
	}
	return attrs
}

// markSpanFailed records err on span and marks it failed without logging.
//
// Takes span (trace.Span) which may be nil or not recording.
// Takes err (error) which is the failure.
// Takes message (string) which describes where the failure happened.
func markSpanFailed(span trace.Span, err error, message string) {
	if span == nil || !span.IsRecording() {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, message)
}

// resolveActionErrorAnswer chooses the status and code an action failure answers with.
//
// An action error whose status is outside 400 to 599 would tell the client the call
// worked, or send it somewhere else, so it answers 500 instead.
//
// Takes err (error) which is the failure.
//
// Returns int which is the HTTP status to answer with.
// Returns string which is the machine-readable code to answer with.
// Returns bool which is false when the action error's own status had to be replaced.
func resolveActionErrorAnswer(err error) (int, string, bool) {
	actionErr, ok := errors.AsType[daemon_dto.ActionError](err)
	if !ok {
		return http.StatusInternalServerError, errorCodeInternal, true
	}

	status := actionErr.StatusCode()
	if status < httpErrorStatusThreshold || status > httpMaxErrorStatus {
		return http.StatusInternalServerError, errorCodeInternal, false
	}
	return status, actionErr.ErrorCode(), true
}

// setActionJSONHeaders marks an action response as JSON and gives it the action default
// cache policy, no-store, unless the app, the action's middleware or the action itself
// has already set a Cache-Control.
//
// Takes h (http.Header) which is the response header map.
func setActionJSONHeaders(h http.Header) {
	h.Set(headerContentType, contentTypeJSON)
	setDefaultCacheControl(h, headerValCacheNoStore)
}
