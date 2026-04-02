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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/logger/logger_domain"
)

type testContextKey string

type testJSONMarshaler struct {
	Value string
}

func (t testJSONMarshaler) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `{"custom":"%s"}`, t.Value), nil
}

type testTextMarshaler struct {
	Value string
}

func (t testTextMarshaler) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "text:%s", t.Value), nil
}

type testStringer struct {
	Value string
}

func (t testStringer) String() string {
	return fmt.Sprintf("stringer:%s", t.Value)
}

type testError struct {
	message string
}

func (e testError) Error() string {
	return e.message
}

type testFailingJSONMarshaler struct{}

func (t testFailingJSONMarshaler) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal error")
}

type testFailingTextMarshaler struct{}

func (t testFailingTextMarshaler) MarshalText() ([]byte, error) {
	return nil, errors.New("marshal error")
}

type testStruct struct {
	Name  string
	Count int
}

func TestNew(t *testing.T) {
	baseLogger := slog.Default()
	logger := logger_domain.New(baseLogger, "test-logger")

	require.NotNil(t, logger)
	assert.NotNil(t, logger.GetContext())
}

func TestNewLogger(t *testing.T) {
	testCases := []struct {
		name           string
		baseLogger     *slog.Logger
		tracerName     string
		ctx            []context.Context
		expectedCtxNil bool
	}{
		{
			name:           "with explicit context",
			baseLogger:     slog.Default(),
			tracerName:     "test-tracer",
			ctx:            []context.Context{context.WithValue(context.Background(), testContextKey("key"), "value")},
			expectedCtxNil: false,
		},
		{
			name:           "without context uses background",
			baseLogger:     slog.Default(),
			tracerName:     "test-tracer",
			ctx:            nil,
			expectedCtxNil: false,
		},
		{
			name:           "with nil context uses background",
			baseLogger:     slog.Default(),
			tracerName:     "test-tracer",
			ctx:            []context.Context{nil},
			expectedCtxNil: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tracer := otel.Tracer(tc.tracerName)
			logger := logger_domain.NewLogger(tc.baseLogger, tracer, tc.ctx...)

			require.NotNil(t, logger)
			assert.NotNil(t, logger.GetContext())
		})
	}
}

func TestNewLoggerWithStackTraceProvider(t *testing.T) {
	baseLogger := slog.Default()
	tracer := otel.Tracer("test")
	expectedFrames := []string{"frame1", "frame2"}
	mockProvider := logger_domain.NewMockStackTraceProvider(
		123,
		logger_domain.NewStackTraceFromFrames(expectedFrames),
	)

	logger := logger_domain.NewLoggerWithStackTraceProvider(baseLogger, tracer, mockProvider)

	require.NotNil(t, logger)

	handler := NewRecordingHandler()
	baseLogger = slog.New(handler)
	logger = logger_domain.NewLoggerWithStackTraceProvider(baseLogger, tracer, mockProvider)

	logger.Error("test error")

	records := handler.GetRecords()
	require.Len(t, records, 1)

	attrs := handler.GetRecordAttrs(records[0])
	stackTrace, hasStackTrace := attrs["stack_trace"]
	assert.True(t, hasStackTrace)
	st, ok := stackTrace.(logger_domain.StackTrace)
	require.True(t, ok)
	assert.Equal(t, expectedFrames, st.Frames())
}

func TestWith_AddsAttributes(t *testing.T) {
	testCases := []struct {
		name          string
		attrs         []logger_domain.Attr
		expectedCount int
	}{
		{
			name: "single attribute",
			attrs: []logger_domain.Attr{
				logger_domain.String("key1", "value1"),
			},
			expectedCount: 1,
		},
		{
			name: "multiple attributes",
			attrs: []logger_domain.Attr{
				logger_domain.String("key1", "value1"),
				logger_domain.Int("key2", 42),
				logger_domain.Bool("key3", true),
			},
			expectedCount: 3,
		},
		{
			name:          "empty attributes returns same logger",
			attrs:         []logger_domain.Attr{},
			expectedCount: 0,
		},
		{
			name: "complex types",
			attrs: []logger_domain.Attr{
				logger_domain.Time("timestamp", time.Now()),
				logger_domain.Duration("duration", time.Second),
				logger_domain.Field("struct", testStruct{Name: "test", Count: 1}),
			},
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

			derivedLogger := logger.With(tc.attrs...)
			require.NotNil(t, derivedLogger)

			derivedLogger.Info("test message")

			records := handler.GetRecords()
			require.Len(t, records, 1)

			attrs := handler.GetRecordAttrs(records[0])

			for _, expectedAttr := range tc.attrs {
				assert.Contains(t, attrs, expectedAttr.Key)
			}
		})
	}
}

func TestWithContext_UpdatesContext(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	originalCtx := logger.GetContext()

	newCtx := context.WithValue(context.Background(), testContextKey("test-key"), "test-value")
	derivedLogger := logger.WithContext(newCtx)

	derivedCtx := derivedLogger.GetContext()
	assert.NotEqual(t, originalCtx, derivedCtx)
	assert.Equal(t, "test-value", derivedCtx.Value(testContextKey("test-key")))
}

func TestLoggingMethods_AllLevels(t *testing.T) {
	testCases := []struct {
		logFunc     func(logger_domain.Logger, string, ...logger_domain.Attr)
		name        string
		message     string
		attrs       []logger_domain.Attr
		level       slog.Level
		expectStack bool
	}{
		{
			name:  "trace level",
			level: logger_domain.LevelTrace,
			logFunc: func(l logger_domain.Logger, message string, attrs ...logger_domain.Attr) {
				l.Trace(message, attrs...)
			},
			message:     "trace message",
			attrs:       []logger_domain.Attr{logger_domain.String("trace_key", "trace_value")},
			expectStack: false,
		},
		{
			name:  "debug level",
			level: slog.LevelDebug,
			logFunc: func(l logger_domain.Logger, message string, attrs ...logger_domain.Attr) {
				l.Debug(message, attrs...)
			},
			message:     "debug message",
			attrs:       []logger_domain.Attr{logger_domain.Int("debug_count", 1)},
			expectStack: false,
		},
		{
			name:  "info level",
			level: slog.LevelInfo,
			logFunc: func(l logger_domain.Logger, message string, attrs ...logger_domain.Attr) {
				l.Info(message, attrs...)
			},
			message:     "info message",
			attrs:       []logger_domain.Attr{logger_domain.Bool("info_flag", true)},
			expectStack: false,
		},
		{
			name:  "notice level",
			level: logger_domain.LevelNotice,
			logFunc: func(l logger_domain.Logger, message string, attrs ...logger_domain.Attr) {
				l.Notice(message, attrs...)
			},
			message:     "notice message",
			attrs:       []logger_domain.Attr{logger_domain.String("notice_type", "startup")},
			expectStack: false,
		},
		{
			name:  "warn level",
			level: slog.LevelWarn,
			logFunc: func(l logger_domain.Logger, message string, attrs ...logger_domain.Attr) {
				l.Warn(message, attrs...)
			},
			message:     "warn message",
			attrs:       []logger_domain.Attr{logger_domain.String("warn_code", "DEPRECATED")},
			expectStack: false,
		},
		{
			name:  "error level includes stack trace",
			level: slog.LevelError,
			logFunc: func(l logger_domain.Logger, message string, attrs ...logger_domain.Attr) {
				l.Error(message, attrs...)
			},
			message:     "error message",
			attrs:       []logger_domain.Attr{logger_domain.Error(errors.New("test error"))},
			expectStack: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

			tc.logFunc(logger, tc.message, tc.attrs...)

			records := handler.GetRecords()
			require.Len(t, records, 1)

			record := records[0]
			assert.Equal(t, tc.level, record.Level)
			assert.Equal(t, tc.message, record.Message)

			attrs := handler.GetRecordAttrs(record)
			for _, expectedAttr := range tc.attrs {
				assert.Contains(t, attrs, expectedAttr.Key)
			}

			_, hasStack := attrs["stack_trace"]
			if tc.expectStack {
				assert.True(t, hasStack, "error level should include stack trace")
			}
		})
	}
}

func TestPanic_CallsPanic(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	assert.Panics(t, func() {
		logger.Panic("panic message")
	})

	records := handler.GetRecords()
	require.Len(t, records, 1)
	assert.Equal(t, "panic message", records[0].Message)
}

func TestSpan_CreatesNewSpan(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	ctx, span, spanLogger := logger.Span(context.Background(), "test-span",
		logger_domain.String("span_attr", "span_value"),
	)

	require.NotNil(t, ctx)
	require.NotNil(t, span)
	require.NotNil(t, spanLogger)

	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "test-span", spans[0].Name)

	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "span_attr" && attr.Value.AsString() == "span_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "span should have the provided attribute")
}

func TestRunInSpan_Success(t *testing.T) {
	testCases := []struct {
		name         string
		spanFunction func(context.Context, logger_domain.Logger) error
		attrs        []logger_domain.Attr
		expectError  bool
		expectStatus codes.Code
	}{
		{
			name: "function returns nil",
			spanFunction: func(ctx context.Context, log logger_domain.Logger) error {
				log.Info("inside span")
				return nil
			},
			attrs:        []logger_domain.Attr{logger_domain.String("key", "value")},
			expectError:  false,
			expectStatus: codes.Ok,
		},
		{
			name: "function returns error",
			spanFunction: func(ctx context.Context, log logger_domain.Logger) error {
				return errors.New("test error")
			},
			attrs:        []logger_domain.Attr{},
			expectError:  true,
			expectStatus: codes.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := NewSpanRecorder("test-tracer")
			defer func() { _ = recorder.Shutdown(context.Background()) }()

			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

			err := logger.RunInSpan(context.Background(), "test-span", tc.spanFunction, tc.attrs...)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			spans := recorder.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, tc.expectStatus, spans[0].StatusCode)
		})
	}
}

func TestRunInSpan_Panic(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	assert.Panics(t, func() {
		_ = logger.RunInSpan(context.Background(), "test-span", func(ctx context.Context, log logger_domain.Logger) error {
			panic("test panic")
		})
	})

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)
}

func TestReportError_RecordsOnSpan(t *testing.T) {
	testCases := []struct {
		name            string
		err             error
		message         string
		attrs           []logger_domain.Attr
		expectLog       bool
		expectSpanError bool
	}{
		{
			name:            "with error",
			err:             errors.New("test error"),
			message:         "operation failed",
			attrs:           []logger_domain.Attr{logger_domain.String("context", "test")},
			expectLog:       true,
			expectSpanError: true,
		},
		{
			name:            "with nil error does nothing",
			err:             nil,
			message:         "operation failed",
			attrs:           []logger_domain.Attr{},
			expectLog:       false,
			expectSpanError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := NewSpanRecorder("test-tracer")
			defer func() { _ = recorder.Shutdown(context.Background()) }()

			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

			ctx, span := recorder.GetTracer().Start(context.Background(), "test-span")

			logger.ReportError(span, tc.err, tc.message, tc.attrs...)

			span.End()

			if tc.expectLog {
				records := handler.GetRecords()
				require.NotEmpty(t, records)
				assert.Equal(t, tc.message, records[0].Message)
			}

			if tc.expectSpanError {
				spans := recorder.GetSpans()
				require.Len(t, spans, 1)
				assert.Equal(t, codes.Error, spans[0].StatusCode)
				assert.Equal(t, tc.message, spans[0].StatusMessage)
			}

			_ = ctx
		})
	}
}

func TestAttributeConversion_AllTypes(t *testing.T) {
	testCases := []struct {
		attr          logger_domain.Attr
		validateValue func(*testing.T, attribute.KeyValue)
		name          string
		expectedKey   string
	}{
		{
			name:        "string type",
			attr:        logger_domain.String("str_key", "str_value"),
			expectedKey: "str_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, "str_value", kv.Value.AsString())
			},
		},
		{
			name:        "int type",
			attr:        logger_domain.Int("int_key", 42),
			expectedKey: "int_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, int64(42), kv.Value.AsInt64())
			},
		},
		{
			name:        "int64 type",
			attr:        logger_domain.Int64("int64_key", 12345),
			expectedKey: "int64_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, int64(12345), kv.Value.AsInt64())
			},
		},
		{
			name:        "uint64 type",
			attr:        logger_domain.Uint64("uint64_key", 67890),
			expectedKey: "uint64_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, int64(67890), kv.Value.AsInt64())
			},
		},
		{
			name:        "float64 type",
			attr:        logger_domain.Float64("float_key", 3.14),
			expectedKey: "float_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.InDelta(t, 3.14, kv.Value.AsFloat64(), 0.01)
			},
		},
		{
			name:        "bool type",
			attr:        logger_domain.Bool("bool_key", true),
			expectedKey: "bool_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.True(t, kv.Value.AsBool())
			},
		},
		{
			name:        "time type",
			attr:        logger_domain.Time("time_key", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			expectedKey: "time_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Contains(t, kv.Value.AsString(), "2025-01-01")
			},
		},
		{
			name:        "duration type",
			attr:        logger_domain.Duration("duration_key", 5*time.Second),
			expectedKey: "duration_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, "5s", kv.Value.AsString())
			},
		},
		{
			name:        "error type",
			attr:        logger_domain.Error(testError{message: "test error"}),
			expectedKey: "error",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, "test error", kv.Value.AsString())
			},
		},
		{
			name:        "nil error",
			attr:        logger_domain.Error(nil),
			expectedKey: "error",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, "<nil>", kv.Value.AsString())
			},
		},
		{
			name:        "string slice",
			attr:        logger_domain.Field("slice_key", []string{"a", "b", "c"}),
			expectedKey: "slice_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				slice := kv.Value.AsStringSlice()
				assert.Equal(t, []string{"a", "b", "c"}, slice)
			},
		},
		{
			name:        "json marshaler",
			attr:        logger_domain.Field("json_key", testJSONMarshaler{Value: "test"}),
			expectedKey: "json_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Contains(t, kv.Value.AsString(), `"custom":"test"`)
			},
		},
		{
			name:        "text marshaler",
			attr:        logger_domain.Field("text_key", testTextMarshaler{Value: "test"}),
			expectedKey: "text_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, "text:test", kv.Value.AsString())
			},
		},
		{
			name:        "stringer",
			attr:        logger_domain.Field("stringer_key", testStringer{Value: "test"}),
			expectedKey: "stringer_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				assert.Equal(t, "stringer:test", kv.Value.AsString())
			},
		},
		{
			name:        "struct fallback",
			attr:        logger_domain.Field("struct_key", testStruct{Name: "test", Count: 42}),
			expectedKey: "struct_key",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {
				jsonString := kv.Value.AsString()
				assert.Contains(t, jsonString, "test")
				assert.Contains(t, jsonString, "42")
			},
		},
		{
			name: "nested group",
			attr: slog.Group("group_key",
				slog.String("nested_key1", "nested_value1"),
				slog.Int("nested_key2", 99),
			),
			expectedKey: "group_key.nested_key1",
			validateValue: func(t *testing.T, kv attribute.KeyValue) {

				assert.Equal(t, "nested_value1", kv.Value.AsString())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := NewSpanRecorder("test-tracer")
			defer func() { _ = recorder.Shutdown(context.Background()) }()

			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

			ctx, span, _ := logger.Span(context.Background(), "test-span", tc.attr)
			span.End()

			spans := recorder.GetSpans()
			require.Len(t, spans, 1)

			attrs := spans[0].Attributes
			found := false
			for _, attr := range attrs {
				if string(attr.Key) == tc.expectedKey {
					found = true
					if tc.validateValue != nil {
						tc.validateValue(t, attr)
					}
					break
				}
			}
			assert.True(t, found, "expected attribute key %q not found", tc.expectedKey)

			_ = ctx
		})
	}
}

func TestAttributeConversion_FailingMarshalers(t *testing.T) {
	testCases := []struct {
		name string
		attr logger_domain.Attr
		key  string
	}{
		{
			name: "failing JSON marshaler falls back to fmt",
			attr: logger_domain.Field("failing_json", testFailingJSONMarshaler{}),
			key:  "failing_json",
		},
		{
			name: "failing text marshaler falls back to JSON",
			attr: logger_domain.Field("failing_text", testFailingTextMarshaler{}),
			key:  "failing_text",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := NewSpanRecorder("test-tracer")
			defer func() { _ = recorder.Shutdown(context.Background()) }()

			handler := NewRecordingHandler()
			baseLogger := slog.New(handler)
			logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

			ctx, span, _ := logger.Span(context.Background(), "test-span", tc.attr)
			span.End()

			spans := recorder.GetSpans()
			require.Len(t, spans, 1)

			attrs := spans[0].Attributes
			found := false
			for _, attr := range attrs {
				if string(attr.Key) == tc.key {
					found = true

					assert.NotEmpty(t, attr.Value.AsString())
					break
				}
			}
			assert.True(t, found, "attribute should exist even with failed marshalling")

			_ = ctx
		})
	}
}

func TestAttributeConversion_DeeplyNestedGroups(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	attr := slog.Group("level1",
		slog.Group("level2",
			slog.Group("level3",
				slog.String("deep_key", "deep_value"),
			),
		),
	)

	ctx, span, _ := logger.Span(context.Background(), "test-span", attr)
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "level1.level2.level3.deep_key" {
			found = true
			assert.Equal(t, "deep_value", attr.Value.AsString())
			break
		}
	}
	assert.True(t, found, "deeply nested group should create dot-separated key")

	_ = ctx
}

func TestAttributeConversion_EmptyGroup(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	attr := slog.Group("empty_group")

	ctx, span, _ := logger.Span(context.Background(), "test-span", attr)
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	_ = ctx
}

func TestAttributeConversion_ErrorType(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	testErr := testError{message: "test error message"}
	attr := logger_domain.Field("error_field", testErr)

	ctx, span, _ := logger.Span(context.Background(), "test-span", attr)
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	found := false
	for _, kvAttr := range spans[0].Attributes {
		if string(kvAttr.Key) == "error_field" {
			found = true
			assert.Equal(t, "test error message", kvAttr.Value.AsString())
			break
		}
	}
	assert.True(t, found, "error_field attribute should exist")

	_ = ctx
}

func TestAttributeConversion_UnmarshallableType(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	type unmarshallableStruct struct {
		Ch chan int
	}
	unmarshallable := unmarshallableStruct{Ch: make(chan int)}
	attr := logger_domain.Field("unmarshallable", unmarshallable)

	ctx, span, _ := logger.Span(context.Background(), "test-span", attr)
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	found := false
	for _, kvAttr := range spans[0].Attributes {
		if string(kvAttr.Key) == "unmarshallable" {
			found = true

			assert.NotEmpty(t, kvAttr.Value.AsString())

			assert.Contains(t, kvAttr.Value.AsString(), "Ch:")
			break
		}
	}
	assert.True(t, found, "unmarshallable attribute should exist with fallback representation")

	_ = ctx
}

func TestStrings_AttributeHelper(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	values := []string{"foo", "bar", "baz"}
	attr := logger_domain.Strings("items", values)

	ctx, span, _ := logger.Span(context.Background(), "test-span", attr)
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	found := false
	for _, kvAttr := range spans[0].Attributes {
		if string(kvAttr.Key) == "items" {
			found = true
			assert.Equal(t, "foo,bar,baz", kvAttr.Value.AsString())
			break
		}
	}
	assert.True(t, found, "items attribute should exist")

	_ = ctx
}

func TestLogWithStack_RespectsLogLevel(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, nil)

	handler.SetEnabled(false)

	logger.Trace("should not be logged")

	assert.Equal(t, 0, handler.Count(), "logs below minimum level should not be recorded")
}

func TestNewLoggerWithStackTraceProvider_WithContext(t *testing.T) {
	type contextKey string
	const testKey contextKey = "test_key"

	ctx := context.WithValue(context.Background(), testKey, "test_value")

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	provider := &logger_domain.RuntimeStackTraceProvider{}

	logger := logger_domain.NewLoggerWithStackTraceProvider(baseLogger, nil, provider, ctx)

	retrievedCtx := logger.GetContext()
	require.NotNil(t, retrievedCtx)

	value := retrievedCtx.Value(testKey)
	assert.Equal(t, "test_value", value)
}

func TestNewLoggerWithStackTraceProvider_WithoutContext(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	provider := &logger_domain.RuntimeStackTraceProvider{}

	logger := logger_domain.NewLoggerWithStackTraceProvider(baseLogger, nil, provider)

	ctx := logger.GetContext()
	require.NotNil(t, ctx)
	assert.Equal(t, context.Background(), ctx)
}

func TestWith_AttributesPropagatedToSpan(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	loggerWithAttr := logger.With(logger_domain.String("with_attr", "with_value"))
	ctx, span, _ := loggerWithAttr.Span(context.Background(), "test-span")
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "with_attr" && attr.Value.AsString() == "with_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "With() attribute should be propagated to span")

	_ = ctx
}

func TestWith_MultipleAttributesAccumulated(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	loggerWithAttrs := logger.
		With(logger_domain.String("attr1", "value1")).
		With(logger_domain.String("attr2", "value2")).
		With(logger_domain.Int("attr3", 42))

	ctx, span, _ := loggerWithAttrs.Span(context.Background(), "test-span")
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	attributeMap := make(map[string]attribute.Value)
	for _, attr := range attrs {
		attributeMap[string(attr.Key)] = attr.Value
	}

	assert.Equal(t, "value1", attributeMap["attr1"].AsString(), "attr1 should be present")
	assert.Equal(t, "value2", attributeMap["attr2"].AsString(), "attr2 should be present")
	assert.Equal(t, int64(42), attributeMap["attr3"].AsInt64(), "attr3 should be present")

	_ = ctx
}

func TestWith_AttributesMergedWithSpanAttrs(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	loggerWithAttr := logger.With(logger_domain.String("with_attr", "with_value"))

	ctx, span, _ := loggerWithAttr.Span(context.Background(), "test-span",
		logger_domain.String("span_attr", "span_value"),
	)
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	attributeMap := make(map[string]attribute.Value)
	for _, attr := range attrs {
		attributeMap[string(attr.Key)] = attr.Value
	}

	assert.Equal(t, "with_value", attributeMap["with_attr"].AsString(), "With() attr should be present")
	assert.Equal(t, "span_value", attributeMap["span_attr"].AsString(), "Span() attr should be present")

	_ = ctx
}

func TestWithContext_PreservesAccumulatedAttrs(t *testing.T) {
	type ctxKey string

	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	loggerWithAttr := logger.With(logger_domain.String("preserved_attr", "preserved_value"))
	newCtx := context.WithValue(context.Background(), ctxKey("test"), "value")
	loggerWithNewCtx := loggerWithAttr.WithContext(newCtx)

	ctx, span, _ := loggerWithNewCtx.Span(context.Background(), "test-span")
	span.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "preserved_attr" && attr.Value.AsString() == "preserved_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "With() attribute should be preserved through WithContext()")

	_ = ctx
}

func TestSpan_ReturnedLoggerHasAccumulatedAttrs(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	loggerWithAttr := logger.With(logger_domain.String("parent_attr", "parent_value"))

	_, span1, spanLogger := loggerWithAttr.Span(context.Background(), "span1",
		logger_domain.String("span1_attr", "span1_value"),
	)
	span1.End()

	_, span2, _ := spanLogger.Span(context.Background(), "span2")
	span2.End()

	spans := recorder.GetSpans()
	require.Len(t, spans, 2)

	var span2Stub *RecordedSpan
	for _, s := range spans {
		if s.Name == "span2" {
			span2Stub = s
			break
		}
	}
	require.NotNil(t, span2Stub, "span2 should exist")

	attributeMap := make(map[string]attribute.Value)
	for _, attr := range span2Stub.Attributes {
		attributeMap[string(attr.Key)] = attr.Value
	}

	assert.Equal(t, "parent_value", attributeMap["parent_attr"].AsString(),
		"parent_attr should be propagated to child span")
	assert.Equal(t, "span1_value", attributeMap["span1_attr"].AsString(),
		"span1_attr should be propagated to child span")
}

func TestRunInSpan_PropagatesWithAttrs(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)
	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer())

	loggerWithAttr := logger.With(logger_domain.String("run_in_span_attr", "run_in_span_value"))

	err := loggerWithAttr.RunInSpan(context.Background(), "test-span",
		func(ctx context.Context, l logger_domain.Logger) error {
			return nil
		},
	)
	require.NoError(t, err)

	spans := recorder.GetSpans()
	require.Len(t, spans, 1)

	attrs := spans[0].Attributes
	found := false
	for _, attr := range attrs {
		if string(attr.Key) == "run_in_span_attr" && attr.Value.AsString() == "run_in_span_value" {
			found = true
			break
		}
	}
	assert.True(t, found, "With() attribute should be propagated via RunInSpan()")
}

func TestCaller_ReturnsMethodAttribute(t *testing.T) {
	attr := logger_domain.Caller()

	assert.Equal(t, logger_domain.KeyMethod, attr.Key)

	assert.Contains(t, attr.Value.String(), "TestCaller_ReturnsMethodAttribute")
}

func TestCaller_CapturesActualCaller(t *testing.T) {

	attr := helperThatCallsCaller()

	assert.Equal(t, logger_domain.KeyMethod, attr.Key)

	assert.Contains(t, attr.Value.String(), "helperThatCallsCaller")
}

func helperThatCallsCaller() slog.Attr {
	return logger_domain.Caller()
}

func TestCaller_StripsPackagePath(t *testing.T) {
	attr := logger_domain.Caller()

	value := attr.Value.String()
	assert.NotContains(t, value, "piko.sh/piko")

	assert.Contains(t, value, "TestCaller_StripsPackagePath")
}

func TestAutoCaller_CapturesCallerInLog(t *testing.T) {
	var buffer bytes.Buffer
	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	baseLogger := slog.New(handler)
	log := logger_domain.New(baseLogger, "test")

	log.Info("test message")

	output := buffer.String()

	assert.Contains(t, output, `"mtd"`)
	assert.Contains(t, output, "TestAutoCaller_CapturesCallerInLog")

	assert.Contains(t, output, `"ctx"`)
	assert.Contains(t, output, "logger_domain_test")
}

func TestAutoCaller_ZeroOverheadWhenDisabled(t *testing.T) {

	var buffer bytes.Buffer
	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelInfo})
	baseLogger := slog.New(handler)
	log := logger_domain.New(baseLogger, "test")

	log.Debug("debug message")

	assert.Empty(t, buffer.String())
}

func TestAutoCaller_PreservedAfterWith(t *testing.T) {
	var buffer bytes.Buffer
	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	baseLogger := slog.New(handler)
	log := logger_domain.New(baseLogger, "test")

	derivedLog := log.With(slog.String("component", "test"))
	derivedLog.Info("test message")

	output := buffer.String()

	assert.Contains(t, output, `"mtd"`)
	assert.Contains(t, output, "TestAutoCaller_PreservedAfterWith")
	assert.Contains(t, output, `"ctx"`)
	assert.Contains(t, output, "logger_domain_test")

	assert.Contains(t, output, `"component"`)
}

func TestAutoCaller_PreservedAfterWithContext(t *testing.T) {
	var buffer bytes.Buffer
	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	baseLogger := slog.New(handler)
	log := logger_domain.New(baseLogger, "test")

	ctx := context.WithValue(context.Background(), testContextKey("key"), "value")
	derivedLog := log.WithContext(ctx)
	derivedLog.Info("test message")

	output := buffer.String()

	assert.Contains(t, output, `"mtd"`)
	assert.Contains(t, output, "TestAutoCaller_PreservedAfterWithContext")
	assert.Contains(t, output, `"ctx"`)
	assert.Contains(t, output, "logger_domain_test")
}

func TestWithoutAutoCaller_DisablesAutoCaller(t *testing.T) {
	var buffer bytes.Buffer
	handler := slog.NewJSONHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug})
	baseLogger := slog.New(handler)
	log := logger_domain.New(baseLogger, "test").WithoutAutoCaller()

	log.Info("test message")

	output := buffer.String()

	assert.NotContains(t, output, `"mtd"`)
	assert.NotContains(t, output, `"ctx"`)
}

func TestContextCause_WarnIncludesDescriptiveCause(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("graceful shutdown requested"))

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test")).WithContext(ctx)
	logger.Warn("operation interrupted")

	records := handler.GetRecords()
	require.Len(t, records, 1)
	attrs := handler.GetRecordAttrs(records[0])
	causeVal, hasCause := attrs["context.cause"]
	assert.True(t, hasCause, "Warn should include context.cause when descriptive cause is set")
	assert.Equal(t, "graceful shutdown requested", causeVal)
}

func TestContextCause_ErrorIncludesDescriptiveCause(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("database connection lost"))

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test")).WithContext(ctx)
	logger.Error("query failed")

	records := handler.GetRecords()
	require.Len(t, records, 1)
	attrs := handler.GetRecordAttrs(records[0])
	causeVal, hasCause := attrs["context.cause"]
	assert.True(t, hasCause, "Error should include context.cause when descriptive cause is set")
	assert.Equal(t, "database connection lost", causeVal)
}

func TestContextCause_BelowWarnDoesNotIncludeCause(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("shutdown requested"))

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test")).WithContext(ctx)

	logger.Trace("trace message")
	logger.Debug("debug message")
	logger.Info("info message")

	records := handler.GetRecords()
	require.Len(t, records, 3)
	for _, r := range records {
		attrs := handler.GetRecordAttrs(r)
		_, hasCause := attrs["context.cause"]
		assert.False(t, hasCause, "levels below Warn should not include context.cause (level=%s)", r.Level)
	}
}

func TestContextCause_SentinelOnlyNotIncluded(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(nil)

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test")).WithContext(ctx)
	logger.Warn("sentinel-only warning")

	records := handler.GetRecords()
	require.Len(t, records, 1)
	attrs := handler.GetRecordAttrs(records[0])
	_, hasCause := attrs["context.cause"]
	assert.False(t, hasCause, "context.cause should not appear when cause equals the sentinel")
}

func TestContextCause_NonCancelledContextNotIncluded(t *testing.T) {
	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))
	logger.Warn("no cancellation here")

	records := handler.GetRecords()
	require.Len(t, records, 1)
	attrs := handler.GetRecordAttrs(records[0])
	_, hasCause := attrs["context.cause"]
	assert.False(t, hasCause, "context.cause should not appear when context is not cancelled")
}

func TestContextCause_ReportErrorIncludesCause(t *testing.T) {
	recorder := NewSpanRecorder("test-tracer")
	defer func() { _ = recorder.Shutdown(context.Background()) }()

	handler := NewRecordingHandler()
	baseLogger := slog.New(handler)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("upstream service unavailable"))

	logger := logger_domain.NewLogger(baseLogger, recorder.GetTracer()).WithContext(ctx)

	_, span := recorder.GetTracer().Start(ctx, "test-span")
	logger.ReportError(span, errors.New("request failed"), "operation error")
	span.End()

	records := handler.GetRecords()
	require.NotEmpty(t, records)
	attrs := handler.GetRecordAttrs(records[0])
	causeVal, hasCause := attrs["context.cause"]
	assert.True(t, hasCause, "ReportError should include context.cause")
	assert.Equal(t, "upstream service unavailable", causeVal)
}
