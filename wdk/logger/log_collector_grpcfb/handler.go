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
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"

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

// Handler implements slog.Handler by translating each record into a
// telemetry_grpcfb.LogLine and enqueuing it on a (typically shared) telemetry client.
type Handler struct {
	// client is the telemetry transport that receives translated log lines and errors.
	client *telemetry_grpcfb.Client

	// limiter is the shared token bucket gating non-error forwards; nil allows everything.
	limiter *rateLimiter

	// breadcrumbs is the shared ring of recent forwarded lines, attached as the trail
	// leading up to each emitted error.
	breadcrumbs *breadcrumbRing

	// group is the dotted attribute namespace applied by WithGroup, empty for the root.
	group string

	// attrs are the preset attributes prepended to every forwarded record.
	attrs []slog.Attr

	// minLevel is the lowest level forwarded; records below it are dropped.
	minLevel slog.Level

	// owns reports whether this handler created the client and must close it.
	owns bool

	// emitErrors reports whether Error+ records are also forwarded as ErrorEvents.
	emitErrors bool
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
}

var (
	_ slog.Handler = (*Handler)(nil)
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
	return &rateLimiter{rate: rate, burst: burst, tokens: burst, clock: clk}
}

// allow reports whether a token is available, refilling by elapsed time. A nil limiter
// allows everything.
//
// Returns bool which is true when a token was consumed and the record may forward.
//
// Concurrency: safe for concurrent callers; serialised by l.mu.
func (l *rateLimiter) allow() bool {
	if l == nil {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock.Now()
	if l.last.IsZero() {
		l.last = now
	}
	l.tokens += now.Sub(l.last).Seconds() * l.rate
	l.last = now
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
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

	defer func() {
		if rec := recover(); rec != nil {
			slog.Default().Error("log_collector_grpcfb: recovered from panic in Handle", "panic", rec)
		}
	}()

	if r.Level < slog.LevelError && !h.limiter.allow() {
		return nil
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
		h.client.AddError(ctx, toError(&r, line, fields, extracted, crumbs))
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

// toError projects a log record into an ErrorEvent.
//
// The culprit, stack and error values were lifted from the un-prefixed attr keys by the
// caller (so they resolve under WithGroup), and the full field set becomes Context. The
// dedicated stack and value are length-bounded so an unbounded panic stack or error
// string cannot blow the frame budget.
//
// Takes r (*slog.Record) which supplies the level and timestamp.
// Takes line (telemetry_grpcfb.LogLine) which supplies the message, logger and base
// fields.
// Takes fields ([]telemetry_grpcfb.KV) which becomes the ErrorEvent Context.
// Takes extracted (extractedFields) which carries the lifted culprit, stack and error
// value.
//
// Returns telemetry_grpcfb.ErrorEvent which is the projected, length-bounded event.
func toError(r *slog.Record, line telemetry_grpcfb.LogLine, fields []telemetry_grpcfb.KV, extracted extractedFields, breadcrumbs string) telemetry_grpcfb.ErrorEvent {
	value := line.Message
	if extracted.errSuffix != "" {
		value = line.Message + ": " + extracted.errSuffix
	}
	value, _ = telemetry_grpcfb.TruncateUTF8(value, maxErrorValueLen)
	culprit, _ := telemetry_grpcfb.TruncateUTF8(extracted.culprit, maxFieldValueLen)
	typ := line.Logger
	if typ == "" {
		typ = "error"
	}
	return telemetry_grpcfb.ErrorEvent{
		Fingerprint:     fingerprint(line.Logger, extracted.culprit, line.Message),
		Type:            typ,
		Value:           value,
		Culprit:         culprit,
		Level:           r.Level.String(),
		StackJSON:       stackJSON(extracted.stack),
		BreadcrumbsJSON: breadcrumbs,
		Context:         fields,
		TimestampMs:     r.Time.UnixMilli(),
		Handled:         true,
	}
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
	nh := *h
	nh.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &nh
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
	nh := *h
	if h.group != "" {
		nh.group = h.group + "." + name
	} else {
		nh.group = name
	}
	return &nh
}

// New wraps an existing, shared telemetry client whose lifecycle the caller owns. The
// default minimum level is Info and forwarding is non-blocking and lossy.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared transport to forward onto.
//
// Returns *Handler which forwards records onto the shared client without owning it.
func New(client *telemetry_grpcfb.Client) *Handler {
	return &Handler{
		client:      client,
		minLevel:    slog.LevelInfo,
		emitErrors:  true,
		limiter:     newRateLimiter(defaultLogRate, defaultLogBurst, clock.RealClock()),
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
	nh := *h
	nh.emitErrors = on
	return &nh
}

// WithLevel returns a copy of the handler that forwards records at or above lvl.
//
// Takes lvl (slog.Level) which is the new minimum level to forward.
//
// Returns *Handler which is a copy carrying the updated minimum level.
func (h *Handler) WithLevel(lvl slog.Level) *Handler {
	nh := *h
	nh.minLevel = lvl
	return &nh
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
	key := a.Key
	if h.group != "" {
		key = h.group + "." + key
	}
	val := a.Value.String()
	switch a.Key {
	case "trace_id", "traceID", "traceId":
		line.TraceID = val
		return
	case "span_id", "spanID", "spanId":
		line.SpanID = val
		return
	case "logger", "component", "scope":
		if line.Logger == "" {
			line.Logger = val
		}
		return
	case "functionName", "function", "func":
		if extracted.culprit == "" {
			extracted.culprit = val
		}
	case "error", "err":
		if extracted.errSuffix == "" {
			extracted.errSuffix = val
		}
	case "stack_trace", "stack", "stackTrace", "panic_info":

		val = resolveStack(a.Value, val)
		if extracted.stack == "" {
			extracted.stack = val
		}
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
	return &Handler{
		client:      client,
		minLevel:    slog.LevelInfo,
		owns:        true,
		emitErrors:  true,
		limiter:     newRateLimiter(defaultLogRate, defaultLogBurst, clock.RealClock()),
		breadcrumbs: newBreadcrumbRing(breadcrumbRingCap),
	}, nil
}
