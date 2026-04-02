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

package logger_test

import (
	"context"
	"crypto/rand"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

type recordedEvent struct {
	Name       string
	Attributes []attribute.KeyValue
	Time       time.Time
}

type recordedSpan struct {
	Name          string
	SpanContext   trace.SpanContext
	Parent        trace.SpanContext
	Attributes    []attribute.KeyValue
	Events        []recordedEvent
	StatusCode    codes.Code
	StatusMessage string
	StartTime     time.Time
	EndTime       time.Time
	ended         bool
}

type testRecordingSpan struct {
	noop.Span
	mu       sync.Mutex
	data     *recordedSpan
	provider *testRecordingTracerProvider
}

func (s *testRecordingSpan) End(_ ...trace.SpanEndOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.ended {
		return
	}
	s.data.EndTime = time.Now()
	s.data.ended = true
	s.provider.recordSpan(s.data)
}

func (s *testRecordingSpan) AddEvent(name string, opts ...trace.EventOption) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := trace.NewEventConfig(opts...)
	s.data.Events = append(s.data.Events, recordedEvent{
		Name:       name,
		Attributes: cfg.Attributes(),
		Time:       cfg.Timestamp(),
	})
}

func (s *testRecordingSpan) IsRecording() bool { return true }

func (s *testRecordingSpan) RecordError(err error, opts ...trace.EventOption) {
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

	s.data.Events = append(s.data.Events, recordedEvent{
		Name:       "exception",
		Attributes: attrs,
		Time:       cfg.Timestamp(),
	})
	s.data.StatusCode = codes.Error
	s.data.StatusMessage = err.Error()
}

func (s *testRecordingSpan) SpanContext() trace.SpanContext { return s.data.SpanContext }

func (s *testRecordingSpan) SetStatus(code codes.Code, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.StatusCode = code
	s.data.StatusMessage = description
}

func (s *testRecordingSpan) SetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Name = name
}

func (s *testRecordingSpan) SetAttributes(kv ...attribute.KeyValue) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Attributes = append(s.data.Attributes, kv...)
}

func (s *testRecordingSpan) TracerProvider() trace.TracerProvider { return s.provider }

type testRecordingTracer struct {
	embedded.Tracer
	name     string
	provider *testRecordingTracerProvider
}

func (t *testRecordingTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)

	var parentCtx trace.SpanContext
	if parentSpan := trace.SpanFromContext(ctx); parentSpan != nil && parentSpan.SpanContext().IsValid() {
		parentCtx = parentSpan.SpanContext()
	}

	var traceID trace.TraceID
	if parentCtx.HasTraceID() {
		traceID = parentCtx.TraceID()
	} else {
		traceID = testRandomTraceID()
	}

	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     testRandomSpanID(),
		TraceFlags: trace.FlagsSampled,
	})

	data := &recordedSpan{
		Name:        name,
		SpanContext: spanCtx,
		Parent:      parentCtx,
		Attributes:  append([]attribute.KeyValue{}, cfg.Attributes()...),
		StartTime:   time.Now(),
	}

	span := &testRecordingSpan{
		data:     data,
		provider: t.provider,
	}

	return trace.ContextWithSpan(ctx, span), span
}

type testRecordingTracerProvider struct {
	embedded.TracerProvider
	mu    sync.Mutex
	spans []*recordedSpan
}

func (p *testRecordingTracerProvider) Tracer(name string, _ ...trace.TracerOption) trace.Tracer {
	return &testRecordingTracer{name: name, provider: p}
}

func (p *testRecordingTracerProvider) recordSpan(span *recordedSpan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spans = append(p.spans, span)
}

func (p *testRecordingTracerProvider) getSpans() []*recordedSpan {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]*recordedSpan, len(p.spans))
	copy(result, p.spans)
	return result
}

func testRandomTraceID() trace.TraceID {
	var id trace.TraceID
	_, _ = rand.Read(id[:])
	return id
}

func testRandomSpanID() trace.SpanID {
	var id trace.SpanID
	_, _ = rand.Read(id[:])
	return id
}
