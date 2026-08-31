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

package log_collector_grpcfb

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// maxFieldValueLen bounds a generic field value's length so a pathological attribute
	// cannot bloat a frame.
	maxFieldValueLen = 1024

	// maxStackLen bounds the dedicated ErrorEvent stack so an unbounded panic stack cannot
	// blow the frame budget.
	maxStackLen = 16384

	// maxErrorValueLen bounds the dedicated ErrorEvent error value so an unbounded error
	// string cannot blow the frame budget.
	maxErrorValueLen = 4096

	// fingerprintNormLen bounds the digit-normalised message prefix the fingerprint hashes.
	fingerprintNormLen = 80

	// initialStackFrameCap is the initial capacity hint for the parsed stack frame slice.
	initialStackFrameCap = 16

	// defaultLogRate bounds how many non-error records the handler forwards per second, so a
	// chatty Info/Warn source cannot dominate the shared lossy stream. Error+ records pass.
	defaultLogRate = 200

	// defaultLogBurst bounds the non-error burst allowance before the per-second rate gates
	// further records. Error+ records always pass.
	defaultLogBurst = 400
)

const (
	// releaseAttrKey is the OTel semantic-convention attribute name carrying the build
	// version piko's environment detection stamps on every record.
	releaseAttrKey = "service.version"

	// environmentAttrKey is the OTel semantic-convention attribute name carrying the
	// deployment name piko's environment detection stamps on every record.
	environmentAttrKey = "deployment.environment.name"
)

// Handler implements slog.Handler by translating each record into a
// telemetry_grpcfb.LogLine and enqueuing it on a (typically shared) telemetry client.
type Handler struct {
	// client is the telemetry transport that receives translated log lines and errors.
	client *telemetry_grpcfb.Client

	// limiter is the shared token bucket gating non-error forwards; nil allows everything.
	limiter *rateLimiter

	// clock supplies the time the limiter refills against; replaceable for testing.
	clock clock.Clock

	// breadcrumbs is the shared ring of recent forwarded lines, attached as the trail
	// leading up to each emitted error.
	breadcrumbs *breadcrumbRing

	// group is the dotted attribute namespace applied by WithGroup, empty for the root.
	group string

	// release is the explicit application release stamped on every ErrorEvent. Empty means
	// "use whatever the record's own attrs report".
	release string

	// environment is the explicit deployment environment stamped on every ErrorEvent. Empty
	// means "use whatever the record's own attrs report".
	environment string

	// attrs are the preset attributes prepended to every forwarded record.
	attrs []slog.Attr

	// minLevel is the lowest level forwarded; records below it are dropped.
	minLevel slog.Level

	// owns reports whether this handler created the client and must close it.
	owns bool

	// emitErrors reports whether Error+ records are also forwarded as ErrorEvents.
	emitErrors bool

	// emitUserID reports whether a user identifier found on a record may be lifted onto the
	// ErrorEvent.
	emitUserID bool

	// emitClientIP reports whether a client IP found on a record may reach the wire.
	emitClientIP bool
}

// extractedFields holds the un-prefixed attribute values lifted for the ErrorEvent
// (resolved from a.Key so they survive WithGroup, which prefixes the field key).
type extractedFields struct {
	// culprit is the function name attributed as the error's origin.
	culprit string

	// stack is the raw newline-separated stack trace for the ErrorEvent.
	stack string

	// errSuffix is the error value appended to the message to form the ErrorEvent value.
	errSuffix string

	// release is the application release the record was produced by.
	release string

	// environment is the deployment environment the record was produced in.
	environment string

	// userID is the affected user, lifted only when the handler opts in.
	userID string
}

var (
	_ slog.Handler = (*Handler)(nil)

	// reportingPanic is set while a recovered panic is being reported.
	reportingPanic atomic.Bool
)

// rateLimiter is a shared token bucket bounding the non-error log forward rate. It is
// shared (by pointer) across WithAttrs/WithGroup-derived handlers so the limit is global.
type rateLimiter struct {
	// clock supplies the current time for token refills, swappable in tests.
	clock clock.Clock

	// last is the time tokens were last refilled.
	last time.Time

	// rate is the tokens replenished per second.
	rate float64

	// burst is the maximum tokens the bucket holds.
	burst float64

	// tokens is the current number of available tokens.
	tokens float64

	// dropped counts records the bucket has shed.
	dropped int64

	// warnedOnce reports whether the first shed has been announced.
	warnedOnce bool

	// mu serialises concurrent allow calls.
	mu sync.Mutex
}

// newRateLimiter builds a token bucket starting full at burst, refilled at rate.
//
// Takes rate (float64) which is the tokens replenished per second.
// Takes burst (float64) which is the maximum tokens the bucket holds.
// Takes clk (clock.Clock) which supplies the current time for refills.
//
// Returns *rateLimiter which is the initialised limiter.
func newRateLimiter(rate, burst float64, clk clock.Clock) *rateLimiter {
	return &rateLimiter{
		clock: clk, last: time.Time{}, rate: rate, burst: burst, tokens: burst, dropped: 0,
	}
}

// allow reports whether a token is available, refilling by elapsed time. A nil limiter
// allows everything.
//
// Returns allowed (bool) which is true when a token was consumed and the record may
// forward.
// Returns firstShed (bool) which is true only on the first shed, so the caller can
// announce that records are being dropped without repeating it for every one.
//
// Concurrency: safe for concurrent callers; serialised by l.mu.
func (l *rateLimiter) allow() (allowed, firstShed bool) {
	if l == nil {
		return true, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock.Now()
	if l.last.IsZero() {
		l.last = now
	}
	l.tokens += now.Sub(l.last).Seconds() * l.rate
	l.last = now
	l.tokens = min(l.tokens, l.burst)
	if l.tokens >= 1 {
		l.tokens--

		return true, false
	}
	l.dropped++

	firstShed = !l.warnedOnce
	l.warnedOnce = true

	return false, firstShed
}

// recoverHandlePanic reports a panic raised while translating a record, and stops the
// report from re-entering the handler that raised it.
//
// Takes recovered (any) which is the value from recover, nil when nothing panicked.
func recoverHandlePanic(ctx context.Context, recovered any) {
	if recovered == nil {
		return
	}

	reportingPanic.Store(true)
	defer reportingPanic.Store(false)

	_, l := logger_domain.From(ctx, log)
	l.Warn("log_collector_grpcfb recovered from panic in Handle",
		logger_domain.String("panic", fmt.Sprint(recovered)),
		logger_domain.String("stack", string(debug.Stack())),
	)
}

// settings reports the configured rate and burst, zero for a nil limiter.
//
// Returns rate (float64) which is the sustained rate.
// Returns burst (float64) which is the bucket size.
//
// Concurrency: safe for concurrent callers; the paired read is serialised by l.mu.
func (l *rateLimiter) settings() (rate, burst float64) {
	if l == nil {
		return 0, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.rate, l.burst
}

// droppedCount reports how many records the bucket has shed since the handler was built.
//
// Returns int64 which is the shed count, zero for a nil limiter (which sheds nothing).
//
// Concurrency: safe for concurrent callers; the read of dropped is serialised by l.mu.
func (l *rateLimiter) droppedCount() int64 {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.dropped
}

// Enabled reports whether records at the given level are forwarded. Records below the
// configured minimum level (default Info) are dropped before translation.
//
// Takes lvl (slog.Level) which is the record level to test against the minimum.
//
// Returns bool which is true when records at lvl are forwarded.
func (h *Handler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.minLevel
}

// Handle translates a record into a LogLine and enqueues it (non-blocking, lossy by
// design). The trace_id, span_id and logger attributes are lifted into dedicated fields;
// all other attributes, including any preset via WithAttrs/WithGroup, become Fields.
//
// Takes r (slog.Record) which is the record to translate and forward.
//
// Returns error which is always nil; forwarding is best-effort and lossy.
//
//nolint:gocritic // hugeParam: by-value slog.Record is fixed by the slog.Handler interface.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if h.client == nil {
		return nil
	}

	if reportingPanic.Load() {
		return nil
	}

	defer func() { recoverHandlePanic(ctx, recover()) }()

	if r.Level < slog.LevelError {
		allowed, firstShed := h.limiter.allow()
		if !allowed {
			if firstShed {
				h.warnRateLimited(ctx)
			}

			return nil
		}
	}
	line := telemetry_grpcfb.LogLine{
		Level:       r.Level.String(),
		Message:     r.Message,
		TimestampMs: r.Time.UnixMilli(),
	}
	fields, extracted := h.collectFields(&r, &line)
	line.Fields = fields

	var crumbs string
	if h.emitErrors && r.Level >= slog.LevelError {
		crumbs = breadcrumbsJSON(h.breadcrumbs.recent(maxBreadcrumbs, line.TraceID))
	}
	h.breadcrumbs.add(breadcrumb{
		tsMs:    line.TimestampMs,
		level:   line.Level,
		logger:  line.Logger,
		message: line.Message,
		traceID: line.TraceID,
	})

	h.client.AddLog(ctx, line)

	if h.emitErrors && r.Level >= slog.LevelError {
		h.client.AddError(ctx, h.toError(&r, line, fields, extracted, crumbs))
	}
	return nil
}

// capField bounds a generic field value's length (UTF-8 safe) for the Context payload.
//
// Takes v (string) which is the raw field value to bound.
//
// Returns string which is v truncated to maxFieldValueLen on a rune boundary.
func capField(v string) string {
	out, _ := telemetry_grpcfb.TruncateUTF8(v, maxFieldValueLen)
	return out
}

// fingerprint is a stable hex hash grouping recurring errors: logger plus culprit plus a
// digit-normalised message prefix, so varying ids and counts collapse to one fingerprint.
//
// Takes logger (string) which is the originating logger or component name.
// Takes culprit (string) which is the function attributed as the error origin.
// Takes message (string) which is the log message, digit-normalised before hashing.
//
// Returns string which is the stable hex fingerprint.
func fingerprint(logger, culprit, message string) string {
	digitsToHash := func(r rune) rune {
		if r >= '0' && r <= '9' {
			return '#'
		}
		return r
	}
	norm, _ := telemetry_grpcfb.TruncateUTF8(strings.Map(digitsToHash, message), fingerprintNormLen)
	h := fnv.New64a()
	_, _ = h.Write([]byte(logger + "|" + culprit + "|" + norm))
	return strconv.FormatUint(h.Sum64(), 16)
}

// stackJSON wraps a raw newline-separated stack trace as a JSON array of frame strings,
// the shape the Issues detail reads, and "" when there is no stack. The stack is first
// length-bounded (UTF-8 safe) so an unbounded panic dump cannot blow the frame budget.
//
// Takes stack (string) which is the raw newline-separated stack trace.
//
// Returns string which is the JSON frame array, or "" when the stack is empty.
func stackJSON(stack string) string {
	stack, _ = telemetry_grpcfb.TruncateUTF8(stack, maxStackLen)
	if strings.TrimSpace(stack) == "" {
		return ""
	}
	frames := make([]string, 0, initialStackFrameCap)
	for ln := range strings.SplitSeq(stack, "\n") {
		if s := strings.TrimSpace(ln); s != "" {
			frames = append(frames, s)
		}
	}
	if len(frames) == 0 {
		return ""
	}
	b, err := json.Marshal(frames)
	if err != nil {
		return ""
	}
	return string(b)
}

// resolveStack resolves a stack_trace attribute to a newline-separated frame string.
//
// Takes v (slog.Value) which is the raw stack_trace attribute value.
// Takes fallback (string) which is returned when v is not a resolvable frame slice.
//
// Returns string which is the newline-joined frames, or fallback.
func resolveStack(v slog.Value, fallback string) string {
	rv := v.Resolve()
	if rv.Kind() == slog.KindAny {
		switch frames := rv.Any().(type) {
		case []string:
			return strings.Join(frames, "\n")
		case []any:
			parts := make([]string, 0, len(frames))
			for _, f := range frames {
				parts = append(parts, fmt.Sprint(f))
			}
			return strings.Join(parts, "\n")
		}
	}
	return fallback
}

// WithAttrs returns a handler that prepends attrs to every record it forwards.
//
// Takes attrs ([]slog.Attr) which are the attributes to preset on the derived handler.
//
// Returns slog.Handler which is the derived handler carrying the combined attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = slices.Concat(h.attrs, attrs)
	return &next
}

// WithGroup returns a handler that namespaces subsequent attributes under name.
//
// Takes name (string) which is the group prefix; an empty name returns the receiver.
//
// Returns slog.Handler which is the derived handler carrying the extended namespace.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	if h.group != "" {
		next.group = h.group + "." + name
	} else {
		next.group = name
	}
	return &next
}

// New wraps an existing, shared telemetry client whose lifecycle the caller owns. The
// default minimum level is Info and forwarding is non-blocking and lossy.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared transport to forward onto.
//
// Returns *Handler which forwards records onto the shared client without owning it.
func New(client *telemetry_grpcfb.Client) *Handler {
	realClock := clock.RealClock()
	return &Handler{
		client:      client,
		minLevel:    slog.LevelInfo,
		emitErrors:  true,
		limiter:     newRateLimiter(defaultLogRate, defaultLogBurst, realClock),
		clock:       realClock,
		breadcrumbs: newBreadcrumbRing(breadcrumbRingCap),
	}
}

// Close releases the underlying telemetry client when the handler owns it (built via
// Dial); a no-op for the shared-client case (New), where the caller owns the lifecycle.
//
// Returns error which is the client's close error, or nil when nothing is owned.
func (h *Handler) Close(ctx context.Context) error {
	if h != nil && h.owns && h.client != nil {
		return h.client.Close(ctx)
	}
	return nil
}

// WithErrorEvents controls whether Error+ records are also forwarded as ErrorEvents (on
// by default). Disable it when a separate error reporter owns that path.
//
// Takes on (bool) which enables forwarding Error+ records as ErrorEvents.
//
// Returns *Handler which is a copy carrying the updated error-event setting.
func (h *Handler) WithErrorEvents(on bool) *Handler {
	next := *h
	next.emitErrors = on
	return &next
}

// WithRelease returns a copy of the handler that stamps release on every ErrorEvent,
// overriding whatever the records themselves report.
//
// Takes release (string) which is the application release.
//
// Returns *Handler which is a copy carrying the explicit release.
func (h *Handler) WithRelease(release string) *Handler {
	next := *h
	next.release = release
	return &next
}

// WithEnvironment returns a copy of the handler that stamps environment on every
// ErrorEvent, overriding whatever the records themselves report.
//
// Takes environment (string) which is the deployment environment.
//
// Returns *Handler which is a copy carrying the explicit environment.
func (h *Handler) WithEnvironment(environment string) *Handler {
	next := *h
	next.environment = environment
	return &next
}

// WithUserID controls whether a user identifier present on a record may be lifted onto
// the ErrorEvent (off by default).
//
// Takes on (bool) which enables lifting the user identifier.
//
// Returns *Handler which is a copy carrying the updated setting.
func (h *Handler) WithUserID(on bool) *Handler {
	next := *h
	next.emitUserID = on
	return &next
}

// WithClientIP controls whether a client IP address present on a record may reach the
// wire (off by default).
//
// Takes on (bool) which enables forwarding the client IP.
//
// Returns *Handler which is a copy carrying the updated setting.
func (h *Handler) WithClientIP(on bool) *Handler {
	next := *h
	next.emitClientIP = on
	return &next
}

// WithRateLimit returns a copy of the handler with the non-error forward rate
// reconfigured.
//
// Takes perSecond (float64) which is the sustained forward rate; <= 0 disables limiting.
// Takes burst (float64) which is the bucket size; clamped up to perSecond when smaller.
//
// Returns *Handler which is a copy carrying the new limit.
func (h *Handler) WithRateLimit(perSecond, burst float64) *Handler {
	next := *h

	if math.IsNaN(perSecond) || math.IsNaN(burst) ||
		math.IsInf(perSecond, 0) || math.IsInf(burst, 0) || perSecond <= 0 {
		next.limiter = nil
		return &next
	}

	burst = max(burst, perSecond)
	next.limiter = newRateLimiter(perSecond, burst, next.clockOrReal())
	return &next
}

// WithClock sets the clock the rate limiter refills against.
//
// Takes clk (clock.Clock) which supplies the current time.
//
// Returns *Handler which is a copy using the given clock.
func (h *Handler) WithClock(clk clock.Clock) *Handler {
	next := *h
	if clk == nil {
		return &next
	}
	next.clock = clk
	if next.limiter != nil {
		next.limiter = newRateLimiter(next.limiter.rate, next.limiter.burst, clk)
	}
	return &next
}

// Dropped reports how many non-error records this handler's limiter has shed.
//
// Returns int64 which is the shed count.
func (h *Handler) Dropped() int64 {
	if h == nil {
		return 0
	}
	return h.limiter.droppedCount()
}

// WithLevel returns a copy of the handler that forwards records at or above lvl.
//
// Takes lvl (slog.Level) which is the new minimum level to forward.
//
// Returns *Handler which is a copy carrying the updated minimum level.
func (h *Handler) WithLevel(lvl slog.Level) *Handler {
	next := *h
	next.minLevel = lvl
	return &next
}

// liftAttr routes one attribute through the dedicated fields, returning the value that
// should still be recorded as ordinary context.
//
// Takes a (slog.Attr) which is the resolved, non-group attribute.
// Takes line (*telemetry_grpcfb.LogLine) which receives the lifted line fields.
// Takes extracted (*extractedFields) which receives the values lifted for the ErrorEvent.
//
// Returns val (string) which is the value to record as context.
// Returns withheld (bool) which is true when the attribute must not reach the wire at
// all, either because it was lifted onto a dedicated field or because it is personal data
// the handler has not been asked to forward.
func (h *Handler) liftAttr(
	a slog.Attr,
	line *telemetry_grpcfb.LogLine,
	extracted *extractedFields,
) (val string, withheld bool) {
	val = a.Value.String()

	switch a.Key {
	case "trace_id", "traceID", "traceId":
		line.TraceID = val
		return val, true
	case "span_id", "spanID", "spanId":
		line.SpanID = val

		return val, true
	case "logger", "component", "scope":
		if line.Logger == "" {
			line.Logger = val
		}

		return val, true
	case "functionName", "function", "func":
		setIfUnset(&extracted.culprit, val)
	case "error", "err":
		setIfUnset(&extracted.errSuffix, val)
	case "stack_trace", "stack", "stackTrace", "panic_info":
		val = resolveStack(a.Value, val)
		setIfUnset(&extracted.stack, val)
	case releaseAttrKey, "release":
		setIfUnset(&extracted.release, val)
	case environmentAttrKey, "environment", "env":
		setIfUnset(&extracted.environment, val)
	case "user_id", "userID", "userId", "user.id":
		if !h.emitUserID {
			return val, true
		}
		setIfUnset(&extracted.userID, val)
	case "client_ip", "clientIP", "clientIp", "ip", "remote_addr":
		if !h.emitClientIP {
			return val, true
		}
	}

	return val, false
}

// warnRateLimited announces, once, that the limiter has begun shedding records.
func (h *Handler) warnRateLimited(ctx context.Context) {
	if reportingPanic.Load() {
		return
	}

	rate, burst := h.limiter.settings()

	_, l := logger_domain.From(ctx, log)
	l.Warn("log_collector_grpcfb is shedding non-error records; the rate limit is saturated",
		logger_domain.Float64("records_per_second", rate),
		logger_domain.Float64("burst", burst),
	)
}

// addGroupAttrs flattens a group attribute and routes each leaf through addAttr, so a
// value nested in a group is judged by the same rules as one at the top level.
//
// Takes a (slog.Attr) which is the group attribute to flatten.
// Takes line (*telemetry_grpcfb.LogLine) which receives any lifted dedicated fields.
// Takes fields (*[]telemetry_grpcfb.KV) which collects the remaining attributes.
// Takes extracted (*extractedFields) which collects the values lifted for the ErrorEvent.
func (h *Handler) addGroupAttrs(
	a slog.Attr,
	line *telemetry_grpcfb.LogLine,
	fields *[]telemetry_grpcfb.KV,
	extracted *extractedFields,
) {
	for _, member := range a.Value.Group() {
		h.addAttr(
			slog.Attr{Key: a.Key + "." + member.Key, Value: member.Value},
			line, fields, extracted,
		)
	}
}

// toError projects a log record into an ErrorEvent.
//
// Takes r (*slog.Record) which supplies the level and timestamp.
// Takes line (telemetry_grpcfb.LogLine) which supplies the message, logger and base
// fields.
// Takes fields ([]telemetry_grpcfb.KV) which becomes the ErrorEvent Context.
// Takes extracted (extractedFields) which carries the lifted culprit, stack, error value
// and deployment identity.
// Takes breadcrumbs (string) which is the JSON trail of recent lines.
//
// Returns telemetry_grpcfb.ErrorEvent which is the projected, length-bounded event.
func (h *Handler) toError(r *slog.Record, line telemetry_grpcfb.LogLine, fields []telemetry_grpcfb.KV, extracted extractedFields, breadcrumbs string) telemetry_grpcfb.ErrorEvent {
	value := line.Message
	if extracted.errSuffix != "" {
		value = line.Message + ": " + extracted.errSuffix
	}
	value, _ = telemetry_grpcfb.TruncateUTF8(value, maxErrorValueLen)
	culprit, _ := telemetry_grpcfb.TruncateUTF8(extracted.culprit, maxFieldValueLen)
	errorType := cmp.Or(line.Logger, "error")

	release := cmp.Or(h.release, extracted.release)
	environment := cmp.Or(h.environment, extracted.environment)
	return telemetry_grpcfb.ErrorEvent{
		Fingerprint:     fingerprint(line.Logger, extracted.culprit, line.Message),
		Type:            errorType,
		Value:           value,
		Culprit:         culprit,
		Level:           r.Level.String(),
		StackJSON:       stackJSON(extracted.stack),
		BreadcrumbsJSON: breadcrumbs,
		Context:         fields,
		Release:         capField(release),
		Environment:     capField(environment),
		UserID:          capField(extracted.userID),
		TimestampMs:     r.Time.UnixMilli(),
		Handled:         true,
	}
}

// collectFields builds the LogLine fields from the handler's preset attrs and the
// record's own attrs, lifting trace_id, span_id and logger into dedicated LogLine fields
// and extracting the culprit, stack and error values for the ErrorEvent.
//
// Takes r (*slog.Record) which supplies the record's own attributes.
// Takes line (*telemetry_grpcfb.LogLine) which receives the lifted dedicated fields.
//
// Returns []telemetry_grpcfb.KV which is the collected field set.
// Returns extractedFields which carries the lifted culprit, stack and error value.
func (h *Handler) collectFields(r *slog.Record, line *telemetry_grpcfb.LogLine) ([]telemetry_grpcfb.KV, extractedFields) {
	fields := make([]telemetry_grpcfb.KV, 0, len(h.attrs)+r.NumAttrs())
	var extracted extractedFields
	for _, a := range h.attrs {
		h.addAttr(a, line, &fields, &extracted)
	}
	r.Attrs(func(a slog.Attr) bool {
		h.addAttr(a, line, &fields, &extracted)
		return true
	})
	return fields, extracted
}

// addAttr routes one attribute: trace, span and logger are lifted into line; culprit,
// stack and error are mirrored into extracted (from the un-prefixed key, so they survive
// WithGroup) while still appearing as a group-prefixed Field; everything else is a Field.
//
// Takes a (slog.Attr) which is the attribute to route.
// Takes line (*telemetry_grpcfb.LogLine) which receives the lifted dedicated fields.
// Takes fields (*[]telemetry_grpcfb.KV) which accumulates the group-prefixed fields.
// Takes extracted (*extractedFields) which receives the mirrored error values.
func (h *Handler) addAttr(a slog.Attr, line *telemetry_grpcfb.LogLine, fields *[]telemetry_grpcfb.KV, extracted *extractedFields) {
	a.Value = a.Value.Resolve()

	if a.Value.Kind() == slog.KindGroup {
		h.addGroupAttrs(a, line, fields, extracted)

		return
	}

	key := a.Key
	if h.group != "" {
		key = h.group + "." + key
	}
	val, withheld := h.liftAttr(a, line, extracted)
	if withheld {
		return
	}
	*fields = append(*fields, telemetry_grpcfb.KV{Key: key, Value: capField(val)})
}

// Dial creates a handler backed by its own telemetry client and connection.
//
// Takes target (string) which is the telemetry endpoint to dial.
// Takes config (telemetry_grpcfb.Config) which configures the new client.
// Takes dialOpts (...grpc.DialOption) which are extra gRPC dial options.
//
// Returns *Handler which owns the dialled client and must be closed.
// Returns error which is non-nil when the dial fails.
func Dial(target string, config telemetry_grpcfb.Config, dialOpts ...grpc.DialOption) (*Handler, error) {
	client, err := telemetry_grpcfb.Dial(target, config, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("log_collector_grpcfb: dial %q: %w", target, err)
	}
	realClock := clock.RealClock()
	return &Handler{
		client:      client,
		minLevel:    slog.LevelInfo,
		owns:        true,
		emitErrors:  true,
		limiter:     newRateLimiter(defaultLogRate, defaultLogBurst, realClock),
		clock:       realClock,
		breadcrumbs: newBreadcrumbRing(breadcrumbRingCap),
	}, nil
}

// clockOrReal returns the handler's clock, defaulting to the wall clock.
//
// Returns clock.Clock which the limiter refills against.
func (h *Handler) clockOrReal() clock.Clock {
	if h.clock == nil {
		return clock.RealClock()
	}
	return h.clock
}

// setIfUnset writes value into dst only when dst is still empty, so the first attribute
// that stated a value is the one kept.
//
// Takes dst (*string) which is the field to fill.
// Takes value (string) which is the candidate value.
func setIfUnset(dst *string, value string) {
	*dst = cmp.Or(*dst, value)
}
