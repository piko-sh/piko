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

//go:build !bench

package logger_domain_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

func TestIntegration_FullStack_LogToOTelToNotification(t *testing.T) {

	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	mockClock := clock.NewMockClock(time.Now())
	mockTransport := &recordingTransport{}
	baseHandler := NewRecordingHandler()
	lifecycle := logger_domain.NewLifecycleManager()

	notificationHandler := logger_domain.NewNotificationHandlerWithOptions(
		baseHandler,
		mockTransport,
		slog.LevelError,
		mockClock,
		lifecycle,
	)

	otelHandler := logger_domain.NewOTelSlogHandler(notificationHandler)

	slogLogger := slog.New(otelHandler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	ctx, span := tracer.Start(context.Background(), "integration-test-span")

	logger.WithContext(ctx).Error("test error 1",
		logger_domain.String("key1", "value1"),
	)

	logger.WithContext(ctx).Error("test error 2",
		logger_domain.String("key2", "value2"),
	)

	span.End()

	mockClock.Advance(15 * time.Second)

	records := baseHandler.GetRecords()
	assert.GreaterOrEqual(t, len(records), 2, "should have at least 2 error logs")

	for _, record := range records {
		attrs := baseHandler.GetRecordAttrs(record)
		assert.Contains(t, attrs, "trace_id", "log should have trace_id")
		assert.Contains(t, attrs, "span_id", "log should have span_id")
	}

	spans := tp.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].StatusCode)

	assert.NotEmpty(t, mockTransport.batches, "should have sent notification batches")
}

func TestIntegration_RunInSpanWithHooks(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	handler := NewRecordingHandler()
	slogLogger := slog.New(handler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	hook := &integrationTestHook{}
	logger.AddSpanLifecycleHook(hook)

	err := logger.RunInSpan(context.Background(), "test-operation", func(ctx context.Context, spanLog logger_domain.Logger) error {
		spanLog.Info("inside span operation")
		return nil
	}, logger_domain.String("operation", "test"))

	require.NoError(t, err)

	assert.Equal(t, 1, hook.onSpanStartCalled, "OnSpanStart should be called once")
	assert.Equal(t, 1, hook.finisherCalled, "finisher should be called once")

	spans := tp.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "test-operation", spans[0].Name)
	assert.Equal(t, codes.Ok, spans[0].StatusCode)
}

func TestIntegration_RunInSpanWithError(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	handler := NewRecordingHandler()
	slogLogger := slog.New(handler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	testErr := errors.New("operation failed")

	err := logger.RunInSpan(context.Background(), "failing-operation", func(ctx context.Context, spanLog logger_domain.Logger) error {
		return testErr
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, testErr)

	records := handler.GetRecords()
	require.NotEmpty(t, records)

	var foundTraceLog bool
	for _, record := range records {
		if record.Level == logger_domain.LevelTrace {
			foundTraceLog = true
			break
		}
	}
	assert.True(t, foundTraceLog, "should have logged at trace level")

	spans := tp.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].StatusCode)
}

func TestIntegration_NestedSpans(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	handler := NewRecordingHandler()
	slogLogger := slog.New(handler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	err := logger.RunInSpan(context.Background(), "outer-span", func(ctx context.Context, outerLog logger_domain.Logger) error {
		outerLog.Info("in outer span")

		return outerLog.RunInSpan(ctx, "inner-span", func(ctx context.Context, innerLog logger_domain.Logger) error {
			innerLog.Info("in inner span")
			return nil
		})
	})

	require.NoError(t, err)

	spans := tp.GetSpans()
	require.Len(t, spans, 2)

	var outerSpan, innerSpan *RecordedSpan
	for _, span := range spans {
		switch span.Name {
		case "outer-span":
			outerSpan = span
		case "inner-span":
			innerSpan = span
		}
	}

	require.NotNil(t, outerSpan)
	require.NotNil(t, innerSpan)

	assert.Equal(t, outerSpan.SpanContext.TraceID(), innerSpan.SpanContext.TraceID(),
		"inner span should have same trace ID as outer span")
	assert.Equal(t, outerSpan.SpanContext.SpanID(), innerSpan.Parent.SpanID(),
		"inner span's parent should be outer span")
}

func TestIntegration_ConcurrentLoggingWithTracing(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	handler := NewRecordingHandler()
	slogLogger := slog.New(handler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	const goroutines = 50
	const spansPerGoroutine = 5

	RunConcurrentTest(t, goroutines, func(id int) {
		for i := range spansPerGoroutine {
			_ = logger.RunInSpan(context.Background(), "concurrent-span", func(ctx context.Context, spanLog logger_domain.Logger) error {
				spanLog.Info("concurrent operation",
					logger_domain.Int("goroutine", id),
					logger_domain.Int("iteration", i),
				)
				return nil
			})
		}
	})

	spans := tp.GetSpans()
	expectedSpans := goroutines * spansPerGoroutine
	assert.Equal(t, expectedSpans, len(spans), "should create all expected spans")

	records := handler.GetRecords()
	assert.Equal(t, expectedSpans, len(records), "should record all expected logs")
}

func TestIntegration_LifecycleWithNotificationHandler(t *testing.T) {
	mockClock := clock.NewMockClock(time.Now())
	mockTransport := &recordingTransport{}
	baseHandler := NewRecordingHandler()
	lifecycle := logger_domain.NewLifecycleManager()

	notificationHandler := logger_domain.NewNotificationHandlerWithOptions(
		baseHandler,
		mockTransport,
		slog.LevelError,
		mockClock,
		lifecycle,
	)

	slogLogger := slog.New(notificationHandler)
	logger := logger_domain.NewLogger(slogLogger, nil)

	logger.Error("error 1")
	logger.Error("error 2")
	logger.Error("error 3")

	assert.True(t, notificationHandler.HasPendingBatch(), "should have pending batch")
	assert.Equal(t, 3, notificationHandler.GetPendingErrorCount(), "should have 3 pending errors")

	notificationHandler.Shutdown()

	assert.False(t, notificationHandler.HasPendingBatch(), "should not have pending batch after shutdown")
	assert.NotEmpty(t, mockTransport.batches, "should have sent batch during shutdown")
}

func TestIntegration_HookPropagation(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	handler := NewRecordingHandler()
	slogLogger := slog.New(handler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	hook := &integrationTestHook{}
	logger.AddSpanLifecycleHook(hook)

	derivedLogger := logger.With(logger_domain.String("derived", "true"))

	ctx, span, _ := derivedLogger.Span(context.Background(), "derived-span")
	span.End()

	assert.Equal(t, 1, hook.onSpanStartCalled, "hook should be called on derived logger")
	assert.Equal(t, 1, hook.finisherCalled, "finisher should be called")

	_ = ctx
}

func TestIntegration_AttributeConversionThroughStack(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("integration-test")

	handler := NewRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(handler)
	slogLogger := slog.New(otelHandler)
	logger := logger_domain.NewLogger(slogLogger, tracer)

	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	err := logger.RunInSpan(context.Background(), "attribute-test", func(ctx context.Context, spanLog logger_domain.Logger) error {
		spanLog.Info("test with attributes")
		return nil
	},
		logger_domain.String("string_attr", "value"),
		logger_domain.Int("int_attr", 42),
		logger_domain.Bool("bool_attr", true),
		logger_domain.Time("time_attr", testTime),
		logger_domain.Duration("duration_attr", 5*time.Second),
	)

	require.NoError(t, err)

	spans := tp.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	attributeMap := make(map[string]any)
	for _, attr := range attrs {
		attributeMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	assert.Contains(t, attributeMap, "string_attr")
	assert.Contains(t, attributeMap, "int_attr")
	assert.Contains(t, attributeMap, "bool_attr")
	assert.Contains(t, attributeMap, "time_attr")
	assert.Contains(t, attributeMap, "duration_attr")

	assert.Equal(t, "value", attributeMap["string_attr"])
	assert.Equal(t, int64(42), attributeMap["int_attr"])
	assert.Equal(t, true, attributeMap["bool_attr"])
}

type integrationTestHook struct {
	onSpanStartCalled   int
	onReportErrorCalled int
	finisherCalled      int
}

func (h *integrationTestHook) OnSpanStart(ctx context.Context, spanName string, attrs []logger_domain.Attr) (context.Context, func()) {
	h.onSpanStartCalled++
	return ctx, func() {
		h.finisherCalled++
	}
}

func (h *integrationTestHook) OnReportError(ctx context.Context, err error, message string, attrs []logger_domain.Attr) {
	h.onReportErrorCalled++
}
