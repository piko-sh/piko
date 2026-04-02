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

//go:build bench

package logger_domain_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/logger/logger_domain"
)

func setupBenchLogger(tracer trace.Tracer) logger_domain.Logger {
	discardHandler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	slogLogger := slog.New(discardHandler)
	return logger_domain.NewLoggerWithStackTraceProvider(
		slogLogger,
		tracer,
		logger_domain.NewRuntimeStackTraceProvider(),
	)
}

func BenchmarkSpan_Current(b *testing.B) {
	scenarios := []struct {
		name           string
		attributeCount int
	}{
		{name: "0_attrs", attributeCount: 0},
		{name: "1_attr", attributeCount: 1},
		{name: "3_attrs", attributeCount: 3},
		{name: "5_attrs", attributeCount: 5},
	}

	tracer := noop.NewTracerProvider().Tracer("bench")
	log := setupBenchLogger(tracer)
	ctx := context.Background()

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			attrs := makeSpanAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, span, _ := log.Span(ctx, "test-span", attrs...)
				span.End()
			}
		})
	}
}

func BenchmarkSpan_NestedSpans(b *testing.B) {
	scenarios := []struct {
		name  string
		depth int
	}{
		{name: "depth_1", depth: 1},
		{name: "depth_2", depth: 2},
		{name: "depth_3", depth: 3},
		{name: "depth_5", depth: 5},
	}

	tracer := noop.NewTracerProvider().Tracer("bench")
	log := setupBenchLogger(tracer)
	ctx := context.Background()

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				currentCtx := ctx
				currentLog := log
				spans := make([]trace.Span, sc.depth)

				for i := range sc.depth {
					var span trace.Span
					currentCtx, span, currentLog = currentLog.Span(currentCtx, "span")
					spans[i] = span
				}

				for i := sc.depth - 1; i >= 0; i-- {
					spans[i].End()
				}
			}
		})
	}
}

func BenchmarkSpan_WithRecordingTracer(b *testing.B) {
	scenarios := []struct {
		name           string
		attributeCount int
	}{
		{name: "0_attrs", attributeCount: 0},
		{name: "3_attrs", attributeCount: 3},
	}

	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("bench")
	log := setupBenchLogger(tracer)
	ctx := context.Background()

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			attrs := makeSpanAttrs(sc.attributeCount)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, span, _ := log.Span(ctx, "test-span", attrs...)
				span.End()
			}
		})
	}
}

func BenchmarkSpan_LogWithinSpan(b *testing.B) {
	scenarios := []struct {
		name     string
		logCalls int
	}{
		{name: "0_logs", logCalls: 0},
		{name: "1_log", logCalls: 1},
		{name: "3_logs", logCalls: 3},
	}

	tracer := noop.NewTracerProvider().Tracer("bench")
	log := setupBenchLogger(tracer)
	ctx := context.Background()

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, span, spanLog := log.Span(ctx, "test-span")
				for range sc.logCalls {
					spanLog.Debug("test message")
				}
				span.End()
			}
		})
	}
}

func makeSpanAttrs(count int) []logger_domain.Attr {
	if count == 0 {
		return nil
	}
	attrs := make([]logger_domain.Attr, count)
	for i := range count {
		attrs[i] = logger_domain.String("key", "value")
	}
	return attrs
}
