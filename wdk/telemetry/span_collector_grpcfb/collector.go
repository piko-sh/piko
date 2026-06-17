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

package span_collector_grpcfb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// defaultMaxAttrValueLen bounds a single (non-redacted) span attribute value so one
	// pathological attribute cannot bloat a frame; the value is truncated UTF-8 safe.
	defaultMaxAttrValueLen = 2048

	// defaultMaxAttributes caps how many attributes are copied from an arbitrary OTel span
	// so a pathological span cannot bloat a frame; once reached, copying stops and a marker
	// attribute records that attributes were dropped.
	defaultMaxAttributes = 256

	// redactedPlaceholder replaces a sensitive attribute value before it leaves the box.
	redactedPlaceholder = "[redacted]"

	// serviceNameKey is the OTEL resource attribute key carrying the service name.
	serviceNameKey attribute.Key = "service.name"

	// attrsDroppedKey marks a span whose attributes hit the maxAttributes cap; its value is
	// the number of attributes that were not copied.
	attrsDroppedKey = "otel.attributes_dropped"

	// attrTruncatedKey marks a span that carried at least one attribute value clipped to the
	// maxAttrValueLen cap; its value is the number of values truncated.
	attrTruncatedKey = "otel.attributes_truncated"
)

// AttributeRedactor decides how a span attribute is emitted to the remote sink: it
// receives the attribute key and raw value and returns the value to ship and whether to
// keep the attribute at all. It is the seam for tailoring the privacy policy.
type AttributeRedactor func(key, value string) (string, bool)

var (
	_ sdktrace.SpanProcessor = (*Collector)(nil)

	// sensitiveAttrKeys are span attribute keys whose value is replaced wholesale: they
	// routinely carry secrets or PII (query strings with tokens, raw SQL, full URLs).
	sensitiveAttrKeys = map[string]struct{}{
		"url.query":         {},
		"url.full":          {},
		"http.url":          {},
		"http.target":       {},
		"db.statement":      {},
		"db.statement.text": {},
	}

	// sensitiveAttrSubstrings redact any key containing one of these tokens (e.g.
	// "http.request.header.authorization"), catching secret-bearing attributes by name.
	sensitiveAttrSubstrings = []string{"authorization", "cookie", "password", "secret", "token", "api_key", "apikey"}
)

// Collector is an OTEL sdktrace.SpanProcessor that forwards completed spans to a
// telemetry_grpcfb.Client. It is registered as an additional span processor on piko's
// tracer provider (alongside the monitoring service's own processor), so it observes
// every span the SDK ends without disturbing the in-process telemetry store.
type Collector struct {
	// client is the telemetry transport that streams spans off-box.
	client *telemetry_grpcfb.Client

	// redact decides how each span attribute is emitted to the remote sink.
	redact AttributeRedactor

	// maxAttributes caps how many attributes are copied from a single span.
	maxAttributes int

	// maxAttrValueLen bounds a single attribute value (UTF-8 safe) before it leaves the box.
	maxAttrValueLen int

	// ownsClient reports whether Shutdown should close client rather than no-op.
	ownsClient bool
}

// Option configures a Collector.
type Option func(*Collector)

// WithAttributeRedactor overrides the default privacy policy applied to every span
// attribute before it streams off-box. Pass a pass-through redactor (returning the value
// unchanged with keep=true) to disable redaction; the default redacts known secret or
// PII-bearing keys and length-bounds the rest.
//
// Takes fn (AttributeRedactor) which is the redactor to apply to every span attribute.
//
// Returns Option which configures the collector with the supplied redactor.
func WithAttributeRedactor(fn AttributeRedactor) Option {
	return func(c *Collector) {
		if fn != nil {
			c.redact = fn
		}
	}
}

// WithMaxAttributes caps how many attributes are copied from a single span before further
// attributes are dropped.
//
// A marker attribute records the dropped count so the clip is observable. The default is
// high (256). Non-positive values are ignored.
//
// Takes n (int) which is the maximum number of span attributes to copy.
//
// Returns Option which configures the collector's attribute-count cap.
func WithMaxAttributes(n int) Option {
	return func(c *Collector) {
		if n > 0 {
			c.maxAttributes = n
		}
	}
}

// WithMaxAttributeValueLen bounds a single attribute value, UTF-8 safe, before it streams
// off-box.
//
// A marker attribute records how many values were truncated so the clip is observable.
// The default is high (2048 bytes). Non-positive values are ignored.
//
// Takes n (int) which is the maximum byte length of a single attribute value.
//
// Returns Option which configures the collector's attribute-value length cap.
func WithMaxAttributeValueLen(n int) Option {
	return func(c *Collector) {
		if n > 0 {
			c.maxAttrValueLen = n
		}
	}
}

// defaultRedactAttribute replaces the value of known secret or PII-bearing attributes
// with a placeholder, so nothing sensitive leaves the box verbatim. Every value
// (including the one returned here) is additionally length-bounded in OnEnd, so the
// default keeps the non-sensitive value as-is.
//
// Takes key (string) which is the span attribute key being classified.
// Takes value (string) which is the raw attribute value being redacted.
//
// Returns string which is the value to ship (placeholder or the original).
// Returns bool which is always true; the default keeps every attribute.
func defaultRedactAttribute(key, value string) (string, bool) {
	lower := strings.ToLower(key)
	if _, ok := sensitiveAttrKeys[lower]; ok {
		return redactedPlaceholder, true
	}
	for _, needle := range sensitiveAttrSubstrings {
		if strings.Contains(lower, needle) {
			return redactedPlaceholder, true
		}
	}
	return value, true
}

// OnStart is a no-op; spans are forwarded once, on completion, in OnEnd.
//
// Takes span (sdktrace.ReadWriteSpan) which is the span just started, and is ignored.
func (*Collector) OnStart(context.Context, sdktrace.ReadWriteSpan) {}

// OnEnd translates a finished span into a telemetry_grpcfb.Span and enqueues it
// (non-blocking, lossy by design). It copies everything it needs; the ReadOnlySpan must
// not be retained.
//
// Takes s (sdktrace.ReadOnlySpan) which is the finished span to translate and enqueue.
func (c *Collector) OnEnd(s sdktrace.ReadOnlySpan) {
	if c == nil || c.client == nil || s == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			_, l := logger_domain.From(context.Background(), log)
			l.Error("span_collector_grpcfb recovered from panic in OnEnd",
				logger_domain.String("panic", fmt.Sprint(r)))
		}
	}()

	span := telemetry_grpcfb.Span{
		TraceID:    s.SpanContext().TraceID().String(),
		SpanID:     s.SpanContext().SpanID().String(),
		Service:    serviceName(s),
		Operation:  s.Name(),
		Kind:       s.SpanKind().String(),
		Status:     statusToString(s.Status().Code),
		StartMs:    s.StartTime().UnixMilli(),
		DurationUs: s.EndTime().Sub(s.StartTime()).Microseconds(),
	}
	if parent := s.Parent(); parent.HasSpanID() {
		span.ParentID = parent.SpanID().String()
	}
	span.Attributes = c.collectAttributes(s)

	c.client.AddSpan(context.Background(), span)
}

// Shutdown releases the processor. For a shared client it is a no-op (the client owner
// drives Start and Close); for an owned client it closes the connection.
//
// Returns error which wraps any failure from closing an owned client.
func (c *Collector) Shutdown(ctx context.Context) error {
	if c != nil && c.ownsClient && c.client != nil {
		if err := c.client.Close(ctx); err != nil {
			return fmt.Errorf("span_collector_grpcfb: shutdown: %w", err)
		}
	}
	return nil
}

// ForceFlush queues the current partial batch on the telemetry client.
//
// Returns error which wraps any failure from the underlying client flush.
func (c *Collector) ForceFlush(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Flush(ctx); err != nil {
		return fmt.Errorf("span_collector_grpcfb: force flush: %w", err)
	}
	return nil
}

// collectAttributes copies a finished span's attributes through the configured redactor,
// capping both the attribute count and each value length and recording marker attributes
// when either cap clips the span, so any loss is observable off-box.
//
// Takes s (sdktrace.ReadOnlySpan) which is the finished span whose attributes are copied.
//
// Returns []telemetry_grpcfb.KV which is the bounded, redacted attribute set to ship.
func (c *Collector) collectAttributes(s sdktrace.ReadOnlySpan) []telemetry_grpcfb.KV {
	redact := c.redact
	if redact == nil {
		redact = defaultRedactAttribute
	}
	maxAttributes := c.maxAttributes
	if maxAttributes <= 0 {
		maxAttributes = defaultMaxAttributes
	}
	maxValueLen := c.maxAttrValueLen
	if maxValueLen <= 0 {
		maxValueLen = defaultMaxAttrValueLen
	}

	attrs := s.Attributes()

	out := make([]telemetry_grpcfb.KV, 0, min(len(attrs), maxAttributes)+3)
	var dropped, truncated int
	for _, a := range attrs {
		if len(out) >= maxAttributes {
			dropped++
			continue
		}
		key := string(a.Key)
		value, keep := redact(key, a.Value.Emit())
		if !keep {
			continue
		}
		value, clipped := telemetry_grpcfb.TruncateUTF8(value, maxValueLen)
		if clipped {
			truncated++
		}
		out = append(out, telemetry_grpcfb.KV{Key: key, Value: value})
	}
	if dropped > 0 {
		out = append(out, telemetry_grpcfb.KV{Key: attrsDroppedKey, Value: strconv.Itoa(dropped)})
	}
	if truncated > 0 {
		out = append(out, telemetry_grpcfb.KV{Key: attrTruncatedKey, Value: strconv.Itoa(truncated)})
	}
	if msg := s.Status().Description; msg != "" {
		value, _ := telemetry_grpcfb.TruncateUTF8(msg, maxValueLen)
		out = append(out, telemetry_grpcfb.KV{Key: "otel.status_description", Value: value})
	}
	return out
}

// serviceName extracts service.name from the span's OTEL resource ("" when absent).
//
// Takes s (sdktrace.ReadOnlySpan) which is the span whose resource is inspected.
//
// Returns string which is the service.name attribute value, or "" when absent.
func serviceName(s sdktrace.ReadOnlySpan) string {
	res := s.Resource()
	if res == nil {
		return ""
	}
	if value, ok := res.Set().Value(serviceNameKey); ok {
		return value.AsString()
	}
	return ""
}

// statusToString maps an OTEL status code to the lowercase wire vocabulary.
//
// Takes code (codes.Code) which is the OTEL status code to map.
//
// Returns string which is the lowercase wire status ("unset", "ok", "error", "unknown").
func statusToString(code codes.Code) string {
	switch code {
	case codes.Unset:
		return "unset"
	case codes.Ok:
		return "ok"
	case codes.Error:
		return "error"
	default:
		return "unknown"
	}
}

// New wraps an existing, shared telemetry client.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared telemetry transport.
// Takes opts (...Option) which override the default attribute redactor.
//
// Returns *Collector which is the configured processor wrapping the shared client.
func New(client *telemetry_grpcfb.Client, opts ...Option) *Collector {
	c := &Collector{
		client:          client,
		redact:          defaultRedactAttribute,
		maxAttributes:   defaultMaxAttributes,
		maxAttrValueLen: defaultMaxAttrValueLen,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// Dial creates a span collector backed by its own telemetry client and connection.
//
// Its Shutdown drains and closes that client. The default attribute redactor is applied.
//
// Takes target (string) which is the telemetry sink address to dial.
// Takes config (telemetry_grpcfb.Config) which configures the new telemetry client.
// Takes dialOpts (...grpc.DialOption) which are passed to the underlying gRPC dial.
//
// Returns *Collector which owns the dialled telemetry client.
// Returns error which wraps any failure to dial the sink.
func Dial(target string, config telemetry_grpcfb.Config, dialOpts ...grpc.DialOption) (*Collector, error) {
	client, err := telemetry_grpcfb.Dial(target, config, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("span_collector_grpcfb: dial %q: %w", target, err)
	}
	return &Collector{
		client:          client,
		redact:          defaultRedactAttribute,
		maxAttributes:   defaultMaxAttributes,
		maxAttrValueLen: defaultMaxAttrValueLen,
		ownsClient:      true,
	}, nil
}
