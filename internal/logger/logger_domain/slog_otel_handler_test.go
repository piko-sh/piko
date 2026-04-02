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
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace/noop"
	"piko.sh/piko/internal/logger/logger_domain"
)

type recordingHandler struct {
	records *[]slog.Record
	attrs   []slog.Attr
	groups  []string
	enabled bool
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{
		records: new(make([]slog.Record, 0)),
		enabled: true,
	}
}

func (h *recordingHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return h.enabled
}

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {

	clone := r.Clone()

	if len(h.attrs) > 0 {
		clone.AddAttrs(h.attrs...)
	}

	*h.records = append(*h.records, clone)
	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &recordingHandler{
		records: h.records,
		enabled: h.enabled,
		attrs:   slices.Concat(h.attrs, attrs),
		groups:  slices.Clone(h.groups),
	}
	return newHandler
}

func (h *recordingHandler) WithGroup(name string) slog.Handler {
	newHandler := &recordingHandler{
		records: h.records,
		enabled: h.enabled,
		attrs:   slices.Clone(h.attrs),
		groups:  slices.Concat(h.groups, []string{name}),
	}
	return newHandler
}

func (h *recordingHandler) getRecordAttrs(r slog.Record) map[string]any {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	return attrs
}

func TestNewOTelSlogHandler(t *testing.T) {
	baseHandler := newRecordingHandler()
	handler := logger_domain.NewOTelSlogHandler(baseHandler)

	require.NotNil(t, handler)
	assert.IsType(t, &logger_domain.OTelSlogHandler{}, handler)
}

func TestOTelSlogHandler_Handle_WithTraceContext(t *testing.T) {
	testCases := []struct {
		name            string
		message         string
		attrs           []slog.Attr
		level           slog.Level
		expectTraceID   bool
		expectSpanID    bool
		expectSpanEvent bool
	}{
		{
			name:            "info level with trace context",
			level:           slog.LevelInfo,
			message:         "test info message",
			attrs:           []slog.Attr{slog.String("key1", "value1")},
			expectTraceID:   true,
			expectSpanID:    true,
			expectSpanEvent: true,
		},
		{
			name:            "debug level with trace context",
			level:           slog.LevelDebug,
			message:         "test debug message",
			attrs:           []slog.Attr{slog.Int("count", 42)},
			expectTraceID:   true,
			expectSpanID:    true,
			expectSpanEvent: true,
		},
		{
			name:            "warn level with trace context",
			level:           slog.LevelWarn,
			message:         "test warn message",
			attrs:           []slog.Attr{},
			expectTraceID:   true,
			expectSpanID:    true,
			expectSpanEvent: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			tp := &RecordingTracerProvider{}
			tracer := tp.Tracer("test")

			ctx, span := tracer.Start(context.Background(), "test-span")
			defer span.End()

			baseHandler := newRecordingHandler()
			otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

			r := slog.NewRecord(time.Now(), tc.level, tc.message, 0)
			for _, attr := range tc.attrs {
				r.AddAttrs(attr)
			}

			err := otelHandler.Handle(ctx, r)
			require.NoError(t, err)

			require.Len(t, *baseHandler.records, 1)
			recorded := (*baseHandler.records)[0]

			attrs := baseHandler.getRecordAttrs(recorded)

			if tc.expectTraceID {
				traceID, hasTraceID := attrs["trace_id"]
				require.True(t, hasTraceID, "trace_id attribute should be present")
				assert.NotEmpty(t, traceID, "trace_id should not be empty")
			}

			if tc.expectSpanID {
				spanID, hasSpanID := attrs["span_id"]
				require.True(t, hasSpanID, "span_id attribute should be present")
				assert.NotEmpty(t, spanID, "span_id should not be empty")
			}

			for _, expectedAttr := range tc.attrs {
				actualValue, exists := attrs[expectedAttr.Key]
				require.True(t, exists, "original attribute %q should be preserved", expectedAttr.Key)
				assert.Equal(t, expectedAttr.Value.Any(), actualValue)
			}

			span.End()
			if tc.expectSpanEvent {
				spans := tp.GetSpans()
				require.Len(t, spans, 1)

				events := spans[0].Events
				require.NotEmpty(t, events, "span should have at least one event")

				var foundEvent bool
				for _, event := range events {
					if event.Name == tc.message {
						foundEvent = true
						break
					}
				}
				assert.True(t, foundEvent, "span should contain event with message %q", tc.message)
			}
		})
	}
}

func TestOTelSlogHandler_Handle_WithoutSpan(t *testing.T) {
	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	ctx := context.Background()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)

	err := otelHandler.Handle(ctx, r)
	require.NoError(t, err)

	require.Len(t, *baseHandler.records, 1)

	attrs := baseHandler.getRecordAttrs((*baseHandler.records)[0])
	_, hasTraceID := attrs["trace_id"]
	_, hasSpanID := attrs["span_id"]

	assert.False(t, hasTraceID, "trace_id should not be added without active span")
	assert.False(t, hasSpanID, "span_id should not be added without active span")
}

func TestOTelSlogHandler_Handle_NonRecordingSpan(t *testing.T) {

	tracer := noop.NewTracerProvider().Tracer("test")
	ctx, span := tracer.Start(context.Background(), "noop-span")
	defer span.End()

	require.False(t, span.IsRecording(), "span should not be recording")

	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := otelHandler.Handle(ctx, r)
	require.NoError(t, err)

	require.Len(t, *baseHandler.records, 1)

	attrs := baseHandler.getRecordAttrs((*baseHandler.records)[0])
	_, hasTraceID := attrs["trace_id"]
	assert.False(t, hasTraceID, "trace_id should not be added for non-recording span")
}

func TestOTelSlogHandler_Handle_ErrorLevel_SetsSpanStatus(t *testing.T) {
	testCases := []struct {
		name               string
		message            string
		expectedStatusDesc string
		attrs              []slog.Attr
		level              slog.Level
		expectStatusError  bool
	}{
		{
			name:               "error level sets span status",
			level:              slog.LevelError,
			message:            "operation failed",
			attrs:              []slog.Attr{},
			expectStatusError:  true,
			expectedStatusDesc: "operation failed",
		},
		{
			name:               "error level with error attribute uses error message",
			level:              slog.LevelError,
			message:            "operation failed",
			attrs:              []slog.Attr{logger_domain.Error(assert.AnError)},
			expectStatusError:  true,
			expectedStatusDesc: assert.AnError.Error(),
		},
		{
			name:              "warn level does not set span status",
			level:             slog.LevelWarn,
			message:           "warning message",
			attrs:             []slog.Attr{},
			expectStatusError: false,
		},
		{
			name:              "info level does not set span status",
			level:             slog.LevelInfo,
			message:           "info message",
			attrs:             []slog.Attr{},
			expectStatusError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			tp := &RecordingTracerProvider{}
			tracer := tp.Tracer("test")

			ctx, span := tracer.Start(context.Background(), "test-span")

			baseHandler := newRecordingHandler()
			otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

			r := slog.NewRecord(time.Now(), tc.level, tc.message, 0)
			for _, attr := range tc.attrs {
				r.AddAttrs(attr)
			}

			err := otelHandler.Handle(ctx, r)
			require.NoError(t, err)

			span.End()

			spans := tp.GetSpans()
			require.Len(t, spans, 1)

			if tc.expectStatusError {
				assert.Equal(t, codes.Error, spans[0].StatusCode, "span status should be Error")
				assert.Equal(t, tc.expectedStatusDesc, spans[0].StatusMessage)
			} else {
				assert.NotEqual(t, codes.Error, spans[0].StatusCode, "span status should not be Error for non-error logs")
			}
		})
	}
}

func TestOTelSlogHandler_Enabled(t *testing.T) {
	testCases := []struct {
		name            string
		handlerEnabled  bool
		expectedEnabled bool
	}{
		{
			name:            "delegates to wrapped handler - enabled",
			handlerEnabled:  true,
			expectedEnabled: true,
		},
		{
			name:            "delegates to wrapped handler - disabled",
			handlerEnabled:  false,
			expectedEnabled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			baseHandler := newRecordingHandler()
			baseHandler.enabled = tc.handlerEnabled
			otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

			enabled := otelHandler.Enabled(context.Background(), slog.LevelInfo)
			assert.Equal(t, tc.expectedEnabled, enabled)
		})
	}
}

func TestOTelSlogHandler_WithAttrs(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	attrs := []slog.Attr{
		slog.String("service", "test-service"),
		slog.Int("version", 1),
	}
	newHandler := otelHandler.WithAttrs(attrs)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	err := newHandler.Handle(ctx, r)
	require.NoError(t, err)

	require.Len(t, *baseHandler.records, 1)
	recordedAttrs := baseHandler.getRecordAttrs((*baseHandler.records)[0])

	assert.Contains(t, recordedAttrs, "trace_id")
	assert.Contains(t, recordedAttrs, "span_id")
	assert.Equal(t, "test-service", recordedAttrs["service"])
	assert.Equal(t, int64(1), recordedAttrs["version"])
}

func TestOTelSlogHandler_WithGroup(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	newHandler := otelHandler.WithGroup("request")

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	r.AddAttrs(slog.String("method", "GET"))

	err := newHandler.Handle(ctx, r)
	require.NoError(t, err)

	require.Len(t, *baseHandler.records, 1)
}

func TestOTelSlogHandler_Chaining(t *testing.T) {

	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	r := slog.NewRecord(time.Now(), slog.LevelError, "chained error", 0)
	r.AddAttrs(slog.String("source", "test"))

	err := otelHandler.Handle(ctx, r)
	require.NoError(t, err)

	require.Len(t, *baseHandler.records, 1)
	attrs := baseHandler.getRecordAttrs((*baseHandler.records)[0])

	assert.Contains(t, attrs, "trace_id")
	assert.Contains(t, attrs, "span_id")
	assert.Equal(t, "test", attrs["source"])

	span.End()
	spans := tp.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Error, spans[0].StatusCode)
}

func TestOTelSlogHandler_Handle_SpanEvents_ContainAttributes(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")

	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "user action", 0)
	r.AddAttrs(
		slog.String("user_id", "usr_123"),
		slog.String("action", "login"),
		slog.Int("attempt", 1),
	)

	err := otelHandler.Handle(ctx, r)
	require.NoError(t, err)

	span.End()

	spans := tp.GetSpans()
	require.Len(t, spans, 1)

	events := spans[0].Events
	require.NotEmpty(t, events)

	var logEvent *RecordedEvent
	for i := range events {
		if events[i].Name == "user action" {
			logEvent = &events[i]
			break
		}
	}
	require.NotNil(t, logEvent, "should find log event in span")

	eventAttrs := make(map[string]any)
	for _, attr := range logEvent.Attributes {
		eventAttrs[string(attr.Key)] = attr.Value.AsInterface()
	}

	assert.Equal(t, "usr_123", eventAttrs["user_id"])
	assert.Equal(t, "login", eventAttrs["action"])
	assert.Equal(t, int64(1), eventAttrs["attempt"])
	assert.Contains(t, eventAttrs, "trace_id")
	assert.Contains(t, eventAttrs, "span_id")
}

func TestOTelSlogHandler_Handle_MultipleLogsCreateMultipleEvents(t *testing.T) {
	tp := &RecordingTracerProvider{}
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")

	baseHandler := newRecordingHandler()
	otelHandler := logger_domain.NewOTelSlogHandler(baseHandler)

	messages := []string{"first log", "second log", "third log"}
	for _, message := range messages {
		r := slog.NewRecord(time.Now(), slog.LevelInfo, message, 0)
		err := otelHandler.Handle(ctx, r)
		require.NoError(t, err)
	}

	span.End()

	spans := tp.GetSpans()
	require.Len(t, spans, 1)

	events := spans[0].Events
	require.Len(t, events, len(messages))

	for _, expectedMessage := range messages {
		found := false
		for _, event := range events {
			if event.Name == expectedMessage {
				found = true
				break
			}
		}
		assert.True(t, found, "should find event for message %q", expectedMessage)
	}
}
