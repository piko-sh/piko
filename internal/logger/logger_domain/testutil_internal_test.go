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

package logger_domain_test

import (
	"context"
	"crypto/rand"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

type RecordedEvent struct {
	Name       string
	Attributes []attribute.KeyValue
	Time       time.Time
}

type RecordedSpan struct {
	Name          string
	SpanContext   trace.SpanContext
	Parent        trace.SpanContext
	Attributes    []attribute.KeyValue
	Events        []RecordedEvent
	StatusCode    codes.Code
	StatusMessage string
	StartTime     time.Time
	EndTime       time.Time
	ended         bool
}

type recordingSpan struct {
	noop.Span
	mu       sync.Mutex
	data     *RecordedSpan
	provider *RecordingTracerProvider
}

func (s *recordingSpan) End(_ ...trace.SpanEndOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.ended {
		return
	}
	s.data.EndTime = time.Now()
	s.data.ended = true
	s.provider.recordSpan(s.data)
}

func (s *recordingSpan) AddEvent(name string, opts ...trace.EventOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := trace.NewEventConfig(opts...)
	s.data.Events = append(s.data.Events, RecordedEvent{
		Name:       name,
		Attributes: cfg.Attributes(),
		Time:       cfg.Timestamp(),
	})
}

func (s *recordingSpan) IsRecording() bool {
	return true
}

func (s *recordingSpan) RecordError(err error, opts ...trace.EventOption) {
	if err == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := trace.NewEventConfig(opts...)

	attrs := make([]attribute.KeyValue, 0, 2+len(cfg.Attributes()))
	attrs = append(attrs,
		attribute.String("exception.type", "*errors.errorString"),
		attribute.String("exception.message", err.Error()),
	)
	attrs = append(attrs, cfg.Attributes()...)

	s.data.Events = append(s.data.Events, RecordedEvent{
		Name:       "exception",
		Attributes: attrs,
		Time:       cfg.Timestamp(),
	})
	s.data.StatusCode = codes.Error
	s.data.StatusMessage = err.Error()
}

func (s *recordingSpan) SpanContext() trace.SpanContext {
	return s.data.SpanContext
}

func (s *recordingSpan) SetStatus(code codes.Code, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.StatusCode = code
	s.data.StatusMessage = description
}

func (s *recordingSpan) SetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Name = name
}

func (s *recordingSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Attributes = append(s.data.Attributes, kv...)
}

func (s *recordingSpan) TracerProvider() trace.TracerProvider {
	return s.provider
}

type recordingTracer struct {
	embedded.Tracer
	name     string
	provider *RecordingTracerProvider
}

func (t *recordingTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)

	var parentCtx trace.SpanContext
	if parentSpan := trace.SpanFromContext(ctx); parentSpan != nil && parentSpan.SpanContext().IsValid() {
		parentCtx = parentSpan.SpanContext()
	}

	var traceID trace.TraceID
	if parentCtx.HasTraceID() {
		traceID = parentCtx.TraceID()
	} else {
		traceID = randomTraceID()
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     randomSpanID(),
		TraceFlags: trace.FlagsSampled,
	})

	data := &RecordedSpan{
		Name:        name,
		SpanContext: spanCtx,
		Parent:      parentCtx,
		Attributes:  slices.Clone(cfg.Attributes()),
		StartTime:   time.Now(),
	}

	span := &recordingSpan{
		data:     data,
		provider: t.provider,
	}

	return trace.ContextWithSpan(ctx, span), span
}

type RecordingTracerProvider struct {
	embedded.TracerProvider
	mu    sync.Mutex
	spans []*RecordedSpan
}

func (p *RecordingTracerProvider) Tracer(name string, _ ...trace.TracerOption) trace.Tracer {
	return &recordingTracer{name: name, provider: p}
}

func (p *RecordingTracerProvider) recordSpan(span *RecordedSpan) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spans = append(p.spans, span)
}

func (p *RecordingTracerProvider) GetSpans() []*RecordedSpan {
	p.mu.Lock()
	defer p.mu.Unlock()

	return slices.Clone(p.spans)
}

func (p *RecordingTracerProvider) GetSpanByName(name string) *RecordedSpan {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, span := range p.spans {
		if span.Name == name {
			return span
		}
	}
	return nil
}

func (p *RecordingTracerProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spans = nil
}

func (p *RecordingTracerProvider) Shutdown(_ context.Context) error {
	return nil
}

func randomTraceID() trace.TraceID {
	var id trace.TraceID
	_, _ = rand.Read(id[:])
	return id
}

func randomSpanID() trace.SpanID {
	var id trace.SpanID
	_, _ = rand.Read(id[:])
	return id
}

type SpanRecorder struct {
	provider *RecordingTracerProvider
	tracer   trace.Tracer
}

func NewSpanRecorder(tracerName string) *SpanRecorder {
	provider := &RecordingTracerProvider{}
	return &SpanRecorder{
		provider: provider,
		tracer:   provider.Tracer(tracerName),
	}
}

func (sr *SpanRecorder) GetTracer() trace.Tracer {
	return sr.tracer
}

func (sr *SpanRecorder) GetProvider() *RecordingTracerProvider {
	return sr.provider
}

func (sr *SpanRecorder) GetSpans() []*RecordedSpan {
	return sr.provider.GetSpans()
}

func (sr *SpanRecorder) GetSpanByName(name string) (*RecordedSpan, bool) {
	span := sr.provider.GetSpanByName(name)
	return span, span != nil
}

func (sr *SpanRecorder) Reset() {
	sr.provider.Reset()
}

func (sr *SpanRecorder) Shutdown(_ context.Context) error {
	return nil
}

type RecordingHandler struct {
	mu      *sync.Mutex
	records *[]slog.Record
	attrs   []slog.Attr
	groups  []string
	enabled bool
}

func NewRecordingHandler() *RecordingHandler {
	mu := &sync.Mutex{}
	return &RecordingHandler{
		mu:      mu,
		records: new(make([]slog.Record, 0)),
		enabled: true,
	}
}

func (h *RecordingHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return h.enabled
}

func (h *RecordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	clone := r.Clone()

	if len(h.attrs) > 0 {
		clone.AddAttrs(h.attrs...)
	}

	*h.records = append(*h.records, clone)
	return nil
}

func (h *RecordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &RecordingHandler{
		mu:      h.mu,
		records: h.records,
		enabled: h.enabled,
		attrs:   slices.Concat(h.attrs, attrs),
		groups:  slices.Clone(h.groups),
	}
}

func (h *RecordingHandler) WithGroup(name string) slog.Handler {
	return &RecordingHandler{
		mu:      h.mu,
		records: h.records,
		enabled: h.enabled,
		attrs:   slices.Clone(h.attrs),
		groups:  slices.Concat(h.groups, []string{name}),
	}
}

func (h *RecordingHandler) GetRecords() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	records := make([]slog.Record, len(*h.records))
	copy(records, *h.records)
	return records
}

func (h *RecordingHandler) GetLastRecord() (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(*h.records) == 0 {
		return slog.Record{}, false
	}
	return (*h.records)[len(*h.records)-1], true
}

func (h *RecordingHandler) GetRecordAttrs(r slog.Record) map[string]any {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	return attrs
}

func (h *RecordingHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.records = make([]slog.Record, 0)
}

func (h *RecordingHandler) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(*h.records)
}

func (h *RecordingHandler) SetEnabled(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = enabled
}

func (h *RecordingHandler) GetAttrs() []slog.Attr {
	return append([]slog.Attr{}, h.attrs...)
}

func (h *RecordingHandler) GetGroups() []string {
	return append([]string{}, h.groups...)
}

func RunConcurrentTest(t *testing.T, goroutines int, callback func(id int)) {
	t.Helper()

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			callback(id)
		}(i)
	}

	wg.Wait()
}

func AssertNoDataRaces(t *testing.T, callback func()) {
	t.Helper()

	callback()
}

func MakeTestRecord(level slog.Level, message string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(time.Now(), level, message, 0)
	for _, attr := range attrs {
		r.AddAttrs(attr)
	}
	return r
}
