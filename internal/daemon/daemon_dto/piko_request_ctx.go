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
	"strings"
	"sync"
	"unicode/utf8"

	"piko.sh/piko/wdk/maths"
	"piko.sh/piko/wdk/useragent"
)

const (
	// maxAnalyticsNameLength bounds a caller-supplied analytics event or action name.
	maxAnalyticsNameLength = 256
)

// PikoRequestCtx is the single per-request carrier stored in the context via one
// context.WithValue call.
//
// Downstream middleware mutates the same pointer, eliminating additional context
// allocations. The struct is pooled to amortise allocation cost to zero after warmup.
type PikoRequestCtx struct {
	// CachedLogger stores a request-scoped logger (as logger_domain.Logger). Set lazily by
	// logger_domain.From on first call per request; subsequent calls return the cached
	// instance at zero cost.
	CachedLogger any

	// CachedAuth stores the resolved authentication context (as AuthContext). Nil when no
	// provider is registered or the request is unauthenticated.
	CachedAuth any

	// ResponseWriter holds the original http.ResponseWriter for the current request. Nil
	// when analytics is not enabled.
	ResponseWriter http.ResponseWriter

	// ErrorPage holds error page context on the error path. Nil for normal requests.
	ErrorPage *ErrorPageContext

	// Locale is the current route locale (e.g., "en", "de"). Set by the route handler
	// closure.
	Locale string

	// CSPToken holds the per-request CSP single-use token. Set by the SecurityHeaders
	// middleware when CSP request tokens are enabled.
	CSPToken string

	// ForwardedRequestID holds a request ID forwarded from a trusted proxy via the
	// X-Request-Id header. Only set when RequestIDCounter is zero.
	ForwardedRequestID string

	// ClientIP is the real client IP address, extracted using trusted proxy rules. Set by
	// the RealIP middleware.
	ClientIP string

	// MatchedPattern is the route pattern that matched the request (e.g., "/blog/{slug}").
	// Set by the route handler closure.
	MatchedPattern string

	// AnalyticsActionName is the server action that handled this request, stamped by the
	// action handler.
	AnalyticsActionName string

	// UserAgent is the raw User-Agent header for the current request.
	UserAgent string

	// AnalyticsRevenue holds optional revenue data stashed by action handlers during request
	// processing; nil when no revenue is associated with the request.
	AnalyticsRevenue *maths.Money

	// AnalyticsProperties holds key-value metadata stashed by action handlers during request
	// processing; nil when no properties have been set (the map is allocated lazily on first
	// use).
	AnalyticsProperties map[string]string

	// Hostname is the request host (e.g. "example.com"), set by the analytics middleware
	// from r.Host for enriching custom events.
	Hostname string

	// AnalyticsEventName is an explicit event name stashed by action handlers. When set, the
	// analytics middleware changes the automatic event type from EventPageView to
	// EventCustom and uses this as the EventName.
	AnalyticsEventName string

	// userAgentClass memoises the classification derived from UserAgent, guarded by mu.
	userAgentClass useragent.Classification

	// RequestIDCounter holds the raw counter for server-generated request IDs. When
	// non-zero, the formatted string is produced lazily by FormatRequestID.
	//
	// Counter values start at 1 (via NextRequestIDCounter), so zero is never a valid
	// generated ID.
	RequestIDCounter uint64

	// ResponseStatusCode is the HTTP status code written by downstream handlers, set by
	// WriteHeader when ResponseWriter is non-nil (zero means WriteHeader was not called
	// explicitly).
	ResponseStatusCode int

	// FromTrustedProxy indicates whether the connection originated from a trusted proxy CIDR
	// range, allowing downstream code to trust forwarding headers such as X-Request-Id.
	FromTrustedProxy bool

	// OtelExtracted marks that OpenTelemetry trace context has been extracted from request
	// headers. Prevents repeated header parsing when multiple middleware call trace context
	// extraction functions.
	OtelExtracted bool

	// DevelopmentMode indicates whether the daemon is running with the developer profile
	// (dev or dev-i). When true, internal error details are shown to users instead of safe
	// messages.
	DevelopmentMode bool

	// userAgentClassified records whether userAgentClass has been derived, so an agent that
	// classifies to all-empty is not derived again on every record.
	userAgentClassified bool

	// mu guards the analytics fields and the memoised User-Agent classification.
	mu sync.Mutex
}

// RequestID returns the formatted request ID. For server-generated IDs the string is
// produced lazily from RequestIDCounter; for forwarded IDs the original string is
// returned.
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

// Write delegates to the underlying ResponseWriter, defaulting the status to 200 if
// WriteHeader was not called.
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

// WriteHeader captures the status code and delegates to the underlying ResponseWriter.
//
// Takes code (int) which is the HTTP status code.
func (p *PikoRequestCtx) WriteHeader(code int) {
	p.ResponseStatusCode = code
	p.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying ResponseWriter if it implements http.Flusher.
func (p *PikoRequestCtx) Flush() {
	if f, ok := p.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter so that middleware further down the chain
// can access optional interfaces (Hijacker, Pusher, etc.) via http.ResponseController.
//
// Returns http.ResponseWriter which is the wrapped response writer.
func (p *PikoRequestCtx) Unwrap() http.ResponseWriter {
	return p.ResponseWriter
}

// ctxKeyPikoRequestCtx is the context key for the per-request carrier.
type ctxKeyPikoRequestCtx struct{}

var (
	// pikoRequestCtxPool provides reusable PikoRequestCtx instances.
	pikoRequestCtxPool = sync.Pool{
		New: func() any { return &PikoRequestCtx{} },
	}
)

// AcquirePikoRequestCtx returns a zeroed PikoRequestCtx from the pool.
//
// Returns *PikoRequestCtx which is a reset instance ready for use.
func AcquirePikoRequestCtx() *PikoRequestCtx {
	pctx, ok := pikoRequestCtxPool.Get().(*PikoRequestCtx)
	if !ok {
		return &PikoRequestCtx{}
	}
	pctx.reset()
	return pctx
}

// SetAnalyticsProperty records a custom analytics property, keeping the map within limit
// entries. An existing key may always be updated; only a new key is refused once the
// limit is reached, so the cap bounds distinct properties rather than writes.
//
// Takes key (string) which is the property name.
// Takes value (string) which is the property value.
// Takes limit (int) which is the most distinct properties to hold.
//
// Returns bool which is false when the property was refused because the limit was
// reached.
//
// Safe for concurrent use; mu is held for the whole body, so the lazy map allocation and
// existence test that enforces limit cannot interleave with another entry of the same
// batch request.
func (p *PikoRequestCtx) SetAnalyticsProperty(key, value string, limit int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.AnalyticsProperties == nil {
		p.AnalyticsProperties = make(map[string]string)
	}
	if _, exists := p.AnalyticsProperties[key]; !exists && len(p.AnalyticsProperties) >= limit {
		return false
	}
	p.AnalyticsProperties[key] = value

	return true
}

// SetAnalyticsEvent records an explicit analytics event name.
//
// Takes name (string) which is the event name, already clamped by the caller.
//
// Safe for concurrent use; mu guards the write to AnalyticsEventName. Two entries of one
// batch naming different events do not merge - the last write is the one the analytics
// middleware reports.
func (p *PikoRequestCtx) SetAnalyticsEvent(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.AnalyticsEventName = name
}

// SetAnalyticsAction records the action the request is attributed to.
//
// Takes name (string) which is the action name, already clamped by the caller.
//
// Safe for concurrent use; mu guards the write to AnalyticsActionName. A string header is
// two words wide, so an unsynchronised overwrite could otherwise be read back with one
// action's pointer and another's length.
func (p *PikoRequestCtx) SetAnalyticsAction(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.AnalyticsActionName = name
}

// SetAnalyticsRevenue records revenue against the request's analytics event.
//
// Takes revenue (*maths.Money) which is the amount to attach.
//
// Safe for concurrent use; mu guards the pointer write to AnalyticsRevenue. An amount
// replaces the one before it rather than adding to it, so a batch whose entries record
// revenue twice reports whichever write landed last, not the sum of the two.
func (p *PikoRequestCtx) SetAnalyticsRevenue(revenue *maths.Money) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.AnalyticsRevenue = revenue
}

// UserAgentClass returns the classification of this request's User-Agent, deriving it at
// most once.
//
// Returns useragent.Classification which describes the client.
//
// Safe for concurrent use; mu is held across the memo check and the derivation alike, so
// useragent.Classify runs at most once, however many error records ask for the client
// classification at once.
func (p *PikoRequestCtx) UserAgentClass() useragent.Classification {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.userAgentClassified {
		p.userAgentClass = useragent.Classify(p.UserAgent)
		p.userAgentClassified = true
	}

	return p.userAgentClass
}

// reset returns every field to its zero value, so a pooled carrier never shows one
// request anything belonging to another.
//
// Not safe for concurrent use with a live request; mu is held throughout, but the rest of
// the carrier is written by the request goroutine without it, so the clear is sound only
// under exclusive ownership.
func (p *PikoRequestCtx) reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.CachedLogger = nil
	p.CachedAuth = nil
	p.ResponseWriter = nil
	p.ErrorPage = nil
	p.Locale = ""
	p.CSPToken = ""
	p.ForwardedRequestID = ""
	p.ClientIP = ""
	p.MatchedPattern = ""
	p.AnalyticsActionName = ""
	p.UserAgent = ""
	p.AnalyticsRevenue = nil
	p.AnalyticsProperties = nil
	p.Hostname = ""
	p.AnalyticsEventName = ""
	p.RequestIDCounter = 0
	p.ResponseStatusCode = 0
	p.FromTrustedProxy = false
	p.OtelExtracted = false
	p.DevelopmentMode = false
	p.userAgentClass = useragent.Classification{}
	p.userAgentClassified = false
}

// ReleasePikoRequestCtx returns a PikoRequestCtx to the pool. The caller must not use the
// struct after this call.
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
	pctx.AnalyticsActionName = ""
	pctx.ResponseStatusCode = 0
	pctx.Hostname = ""
	pctx.UserAgent = ""
	pikoRequestCtxPool.Put(pctx)
}

// WithPikoRequestCtx returns a new context carrying the given PikoRequestCtx.
//
// Takes pctx (*PikoRequestCtx) which is the carrier to store.
//
// Returns context.Context which contains the PikoRequestCtx.
func WithPikoRequestCtx(ctx context.Context, pctx *PikoRequestCtx) context.Context {
	return context.WithValue(ctx, ctxKeyPikoRequestCtx{}, pctx)
}

// PikoRequestCtxFromContext retrieves the PikoRequestCtx from the context, or nil if not
// present.
//
// Returns *PikoRequestCtx which is the carrier, or nil if absent.
func PikoRequestCtxFromContext(ctx context.Context) *PikoRequestCtx {
	pctx, ok := ctx.Value(ctxKeyPikoRequestCtx{}).(*PikoRequestCtx)
	if !ok {
		return nil
	}
	return pctx
}

// ClampAnalyticsName shortens a caller-supplied analytics name to maxAnalyticsNameLength,
// keeping the result valid UTF-8.
//
// Takes name (string) which is the caller-supplied name.
//
// Returns string which is at most maxAnalyticsNameLength bytes.
// Returns bool which is true when the name had to be shortened, so a caller can report
// the loss rather than let two long names collapse into one series unnoticed.
func ClampAnalyticsName(name string) (string, bool) {
	if len(name) <= maxAnalyticsNameLength {
		return name, false
	}

	cut := maxAnalyticsNameLength
	for cut > 0 && !utf8.RuneStart(name[cut]) {
		cut--
	}

	if cut == 0 {
		cut = maxAnalyticsNameLength
	}
	return strings.Clone(name[:cut]), true
}
