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

package daemon_dto

import (
	"context"
	"net/http"
	"sync"

	"piko.sh/piko/wdk/maths"
)

// PikoRequestCtx is the single per-request carrier stored in the
// context via one context.WithValue call.
//
// Downstream middleware mutates the same pointer, eliminating
// additional context allocations. The struct is pooled to amortise
// allocation cost to zero after warmup.
type PikoRequestCtx struct {
	// CachedLogger stores a request-scoped logger (as logger_domain.Logger).
	// Set lazily by logger_domain.From on first call per request; subsequent
	// calls return the cached instance at zero cost.
	CachedLogger any

	// CachedAuth stores the resolved authentication context (as AuthContext).
	// Nil when no provider is registered or the request is unauthenticated.
	CachedAuth any

	// ResponseWriter holds the original http.ResponseWriter for the
	// current request. Nil when analytics is not enabled.
	ResponseWriter http.ResponseWriter

	// ErrorPage holds error page context on the error path. Nil for normal
	// requests.
	ErrorPage *ErrorPageContext

	// Locale is the current route locale (e.g., "en", "de"). Set by the route
	// handler closure.
	Locale string

	// CSPToken holds the per-request CSP nonce token. Set by the
	// SecurityHeaders middleware when CSP request tokens are enabled.
	CSPToken string

	// ForwardedRequestID holds a request ID forwarded from a trusted proxy via
	// the X-Request-Id header. Only set when RequestIDCounter is zero.
	ForwardedRequestID string

	// ClientIP is the real client IP address, extracted using trusted proxy
	// rules. Set by the RealIP middleware.
	ClientIP string

	// MatchedPattern is the route pattern that matched the request
	// (e.g., "/blog/{slug}"). Set by the route handler closure.
	MatchedPattern string

	// AnalyticsRevenue holds optional revenue data stashed by action
	// handlers during request processing. The analytics middleware
	// copies this into the automatic pageview event after the handler
	// returns. Nil when no revenue is associated with the request.
	AnalyticsRevenue *maths.Money

	// AnalyticsProperties holds key-value metadata stashed by action
	// handlers during request processing. The analytics middleware
	// merges these into the automatic pageview event. Nil when no
	// properties have been set; the map is allocated lazily on first
	// use to avoid overhead for requests that don't need it.
	AnalyticsProperties map[string]string

	// Hostname is the request host (e.g. "example.com"). Set by the
	// analytics middleware from r.Host. Used to enrich custom analytics
	// events fired from action handlers.
	Hostname string

	// AnalyticsEventName is an explicit event name stashed by action
	// handlers. When set, the analytics middleware changes the
	// automatic event type from EventPageView to EventCustom and uses
	// this as the EventName.
	AnalyticsEventName string

	// RequestIDCounter holds the raw counter for server-generated
	// request IDs. When non-zero, the formatted string is produced
	// lazily by FormatRequestID.
	//
	// Counter values start at 1 (via NextRequestIDCounter), so zero
	// is never a valid generated ID.
	RequestIDCounter uint64

	// ResponseStatusCode is the HTTP status code written by downstream
	// handlers. Set by WriteHeader when ResponseWriter is non-nil.
	// Zero means WriteHeader was not called explicitly.
	ResponseStatusCode int

	// FromTrustedProxy indicates whether the connection originated from a
	// trusted proxy CIDR range, allowing downstream code to trust forwarding
	// headers such as X-Request-Id.
	FromTrustedProxy bool

	// OtelExtracted marks that OpenTelemetry trace context has been extracted
	// from request headers. Prevents repeated header parsing when multiple
	// middleware call trace context extraction functions.
	OtelExtracted bool

	// DevelopmentMode indicates whether the daemon is running in development
	// mode (dev or dev-i). When true, internal error details are shown to
	// users instead of safe messages.
	DevelopmentMode bool
}

// RequestID returns the formatted request ID. For server-generated
// IDs the string is produced lazily from RequestIDCounter; for
// forwarded IDs the original string is returned.
//
// Returns string which is the formatted or forwarded request ID.
func (p *PikoRequestCtx) RequestID() string {
	if p.RequestIDCounter != 0 {
		return FormatRequestID(p.RequestIDCounter)
	}
	return p.ForwardedRequestID
}

// Header delegates to the underlying ResponseWriter.
//
// Returns http.Header which is the response header map.
func (p *PikoRequestCtx) Header() http.Header {
	return p.ResponseWriter.Header()
}

// Write delegates to the underlying ResponseWriter, defaulting the
// status to 200 if WriteHeader was not called.
//
// Takes b ([]byte) which is the data to write.
//
// Returns int which is the number of bytes written.
// Returns error when the underlying writer fails.
func (p *PikoRequestCtx) Write(b []byte) (int, error) {
	if p.ResponseStatusCode == 0 {
		p.ResponseStatusCode = http.StatusOK
	}
	return p.ResponseWriter.Write(b)
}

// WriteHeader captures the status code and delegates to the
// underlying ResponseWriter.
//
// Takes code (int) which is the HTTP status code.
func (p *PikoRequestCtx) WriteHeader(code int) {
	p.ResponseStatusCode = code
	p.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying ResponseWriter if it implements
// http.Flusher.
func (p *PikoRequestCtx) Flush() {
	if f, ok := p.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter so that middleware
// further down the chain can access optional interfaces (Hijacker,
// Pusher, etc.) via http.ResponseController.
//
// Returns http.ResponseWriter which is the wrapped response writer.
func (p *PikoRequestCtx) Unwrap() http.ResponseWriter {
	return p.ResponseWriter
}

// ctxKeyPikoRequestCtx is the context key for the per-request carrier.
type ctxKeyPikoRequestCtx struct{}

// pikoRequestCtxPool provides reusable PikoRequestCtx instances.
var pikoRequestCtxPool = sync.Pool{
	New: func() any { return &PikoRequestCtx{} },
}

// AcquirePikoRequestCtx returns a zeroed PikoRequestCtx from the
// pool.
//
// Returns *PikoRequestCtx which is a reset instance ready for use.
func AcquirePikoRequestCtx() *PikoRequestCtx {
	pctx, ok := pikoRequestCtxPool.Get().(*PikoRequestCtx)
	if !ok {
		pctx = &PikoRequestCtx{}
	}
	*pctx = PikoRequestCtx{}
	return pctx
}

// ReleasePikoRequestCtx returns a PikoRequestCtx to the pool. The
// caller must not use the struct after this call.
//
// Takes pctx (*PikoRequestCtx) which is the instance to return.
func ReleasePikoRequestCtx(pctx *PikoRequestCtx) {
	if pctx == nil {
		return
	}
	pctx.ErrorPage = nil
	pctx.CachedLogger = nil
	pctx.CachedAuth = nil
	pctx.ResponseWriter = nil
	pctx.AnalyticsRevenue = nil
	pctx.AnalyticsProperties = nil
	pctx.AnalyticsEventName = ""
	pctx.ResponseStatusCode = 0
	pctx.Hostname = ""
	pikoRequestCtxPool.Put(pctx)
}

// WithPikoRequestCtx returns a new context carrying the given
// PikoRequestCtx.
//
// Takes pctx (*PikoRequestCtx) which is the carrier to store.
//
// Returns context.Context which contains the PikoRequestCtx.
func WithPikoRequestCtx(ctx context.Context, pctx *PikoRequestCtx) context.Context {
	return context.WithValue(ctx, ctxKeyPikoRequestCtx{}, pctx)
}

// PikoRequestCtxFromContext retrieves the PikoRequestCtx from the
// context, or nil if not present.
//
// Returns *PikoRequestCtx which is the carrier, or nil if absent.
func PikoRequestCtxFromContext(ctx context.Context) *PikoRequestCtx {
	pctx, ok := ctx.Value(ctxKeyPikoRequestCtx{}).(*PikoRequestCtx)
	if !ok {
		return nil
	}
	return pctx
}
