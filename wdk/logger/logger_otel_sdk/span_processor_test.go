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

package logger_otel_sdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

func TestStatusToString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		code     codes.Code
	}{
		{
			name:     "unset status",
			code:     codes.Unset,
			expected: "UNSET",
		},
		{
			name:     "ok status",
			code:     codes.Ok,
			expected: "OK",
		},
		{
			name:     "error status",
			code:     codes.Error,
			expected: "ERROR",
		},
		{
			name:     "unknown status code",
			code:     codes.Code(99),
			expected: "UNKNOWN",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := statusToString(tc.code)
			if result != tc.expected {
				t.Errorf("statusToString(%v) = %q, want %q", tc.code, result, tc.expected)
			}
		})
	}
}

func TestNewSpanProcessor(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)
	require.NotNil(t, processor)
}

func TestSpanProcessor_OnStart(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	processor.OnStart(context.Background(), nil)
}

func TestSpanProcessor_Shutdown(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	err := processor.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestSpanProcessor_ForceFlush(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	err := processor.ForceFlush(context.Background())
	assert.NoError(t, err)
}

func TestSpanProcessor_OnEnd(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	otelResource, err := resource.New(
		context.Background(),
		resource.WithAttributes(semconv.ServiceName("test-service")),
	)
	require.NoError(t, err)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
		sdktrace.WithResource(otelResource),
	)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-operation",
		trace.WithAttributes(
			attribute.String("key1", "value1"),
			attribute.Int("key2", 42),
		),
	)
	span.SetStatus(codes.Ok, "")
	span.AddEvent("test-event",
		trace.WithAttributes(attribute.String("event-key", "event-value")),
	)
	span.End()

	spans := store.GetSpans(10, false)
	require.Len(t, spans, 1)

	recorded := spans[0]
	assert.Equal(t, "test-operation", recorded.Name)
	assert.Equal(t, "OK", recorded.Status)
	assert.Equal(t, "test-service", recorded.ServiceName)
	assert.Equal(t, "value1", recorded.Attributes["key1"])
	assert.Equal(t, "42", recorded.Attributes["key2"])
	assert.NotEmpty(t, recorded.TraceID)
	assert.NotEmpty(t, recorded.SpanID)
	assert.Greater(t, recorded.DurationNs, int64(0))

	require.Len(t, recorded.Events, 1)
	assert.Equal(t, "test-event", recorded.Events[0].Name)
	assert.Equal(t, "event-value", recorded.Events[0].Attributes["event-key"])
}

func TestSpanProcessor_OnEnd_WithParentSpan(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
	)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	ctx, parent := tracer.Start(context.Background(), "parent-op")
	_, child := tracer.Start(ctx, "child-op")
	child.End()
	parent.End()

	spans := store.GetSpans(10, false)
	require.Len(t, spans, 2)

	var childSpan monitoring_domain.SpanData
	for _, s := range spans {
		if s.Name == "child-op" {
			childSpan = s
			break
		}
	}

	assert.Equal(t, "child-op", childSpan.Name)
	assert.NotEmpty(t, childSpan.ParentSpanID, "child should have a parent span ID")
}

func TestSpanProcessor_OnEnd_ErrorStatus(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
	)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "failing-op")
	span.SetStatus(codes.Error, "something went wrong")
	span.End()

	spans := store.GetSpans(10, false)
	require.Len(t, spans, 1)
	assert.Equal(t, "ERROR", spans[0].Status)
	assert.Equal(t, "something went wrong", spans[0].StatusMessage)
}

func TestSpanProcessor_OnEnd_NoResource(t *testing.T) {
	t.Parallel()

	store := monitoring_domain.NewTelemetryStore()
	processor := NewSpanProcessor(store)

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
	)
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "no-resource-op")
	span.End()

	spans := store.GetSpans(10, false)
	require.Len(t, spans, 1)
	assert.Equal(t, "no-resource-op", spans[0].Name)
}
