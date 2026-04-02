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

package watermill

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	watermillLog "github.com/ThreeDotsLabs/watermill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/logger/logger_domain"
)

type logCall struct {
	method  string
	message string
	fields  []logger_domain.Attr
}

type recordingLogger struct {
	calls     []logCall
	withCalls [][]logger_domain.Attr
}

var _ logger_domain.Logger = (*recordingLogger)(nil)

func (r *recordingLogger) Error(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Error", message: message, fields: arguments})
}

func (r *recordingLogger) Warn(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Warn", message: message, fields: arguments})
}

func (r *recordingLogger) Notice(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Notice", message: message, fields: arguments})
}

func (r *recordingLogger) Info(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Info", message: message, fields: arguments})
}

func (r *recordingLogger) Debug(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Debug", message: message, fields: arguments})
}

func (r *recordingLogger) Internal(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Internal", message: message, fields: arguments})
}

func (r *recordingLogger) Trace(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Trace", message: message, fields: arguments})
}

func (r *recordingLogger) Panic(message string, arguments ...logger_domain.Attr) {
	r.calls = append(r.calls, logCall{method: "Panic", message: message, fields: arguments})
}

func (r *recordingLogger) With(arguments ...logger_domain.Attr) logger_domain.Logger {
	r.withCalls = append(r.withCalls, arguments)
	return &recordingLogger{}
}

func (*recordingLogger) Enabled(_ slog.Level) bool { return false }

func (r *recordingLogger) WithContext(_ context.Context) logger_domain.Logger {
	return r
}

func (*recordingLogger) GetContext() context.Context {
	return context.Background()
}

func (r *recordingLogger) Span(ctx context.Context, _ string, _ ...slog.Attr) (context.Context, trace.Span, logger_domain.Logger) {
	return ctx, nil, r
}

func (*recordingLogger) ReportError(_ trace.Span, _ error, _ string, _ ...logger_domain.Attr) {}

func (r *recordingLogger) RunInSpan(ctx context.Context, _ string, operation func(context.Context, logger_domain.Logger) error, _ ...logger_domain.Attr) error {
	return operation(ctx, r)
}

func (*recordingLogger) AddSpanLifecycleHook(_ logger_domain.SpanLifecycleHook) {}

func (r *recordingLogger) WithoutAutoCaller() logger_domain.Logger {
	return r
}

func (r *recordingLogger) WithSpanContext(_ context.Context) logger_domain.Logger {
	return r
}

func TestNewAdapter(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil adapter", func(t *testing.T) {
		t.Parallel()

		logger := &recordingLogger{}
		adapter := NewAdapter(logger)
		require.NotNil(t, adapter)
	})

	t.Run("implements watermill LoggerAdapter", func(t *testing.T) {
		t.Parallel()

		logger := &recordingLogger{}
		adapter := NewAdapter(logger)
		assert.Implements(t, (*watermillLog.LoggerAdapter)(nil), adapter)
	})
}

func TestConvertFields(t *testing.T) {
	t.Parallel()

	adapter := &Adapter{logger: &recordingLogger{}}

	t.Run("string field produces logger_domain String", func(t *testing.T) {
		t.Parallel()

		fields := watermillLog.LogFields{"name": "alice"}
		result := adapter.convertFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "name", result[0].Key)
		assert.Equal(t, "alice", result[0].Value.String())
	})

	t.Run("int field produces logger_domain Int", func(t *testing.T) {
		t.Parallel()

		fields := watermillLog.LogFields{"count": 42}
		result := adapter.convertFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "count", result[0].Key)
		assert.Equal(t, int64(42), result[0].Value.Int64())
	})

	t.Run("int64 field produces logger_domain Int64", func(t *testing.T) {
		t.Parallel()

		fields := watermillLog.LogFields{"offset": int64(9999)}
		result := adapter.convertFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "offset", result[0].Key)
		assert.Equal(t, int64(9999), result[0].Value.Int64())
	})

	t.Run("bool field produces logger_domain Bool", func(t *testing.T) {
		t.Parallel()

		fields := watermillLog.LogFields{"active": true}
		result := adapter.convertFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "active", result[0].Key)
		assert.Equal(t, true, result[0].Value.Bool())
	})

	t.Run("error field produces logger_domain Error", func(t *testing.T) {
		t.Parallel()

		testError := errors.New("connection refused")
		fields := watermillLog.LogFields{"err": testError}
		result := adapter.convertFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "error", result[0].Key)
		assert.Equal(t, "connection refused", result[0].Value.String())
	})

	t.Run("unknown type produces logger_domain Field with Any", func(t *testing.T) {
		t.Parallel()

		customValue := struct{ Name string }{Name: "test"}
		fields := watermillLog.LogFields{"custom": customValue}
		result := adapter.convertFields(fields)

		require.Len(t, result, 1)
		assert.Equal(t, "custom", result[0].Key)
		assert.Equal(t, customValue, result[0].Value.Any())
	})

	t.Run("extra fields are appended", func(t *testing.T) {
		t.Parallel()

		fields := watermillLog.LogFields{"key": "value"}
		extra := logger_domain.String("extra", "data")
		result := adapter.convertFields(fields, extra)

		require.Len(t, result, 2)

		found := false
		for _, attribute := range result {
			if attribute.Key == "extra" {
				found = true
				assert.Equal(t, "data", attribute.Value.String())
			}
		}
		assert.True(t, found)
	})
}

func TestAdapterError(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	adapter := &Adapter{logger: logger}
	testError := errors.New("something failed")

	adapter.Error("handler crashed", testError, watermillLog.LogFields{"topic": "events"})

	require.Len(t, logger.calls, 1)
	assert.Equal(t, "Error", logger.calls[0].method)
	assert.Equal(t, "handler crashed", logger.calls[0].message)

	hasErrorField := false
	hasTopicField := false
	for _, field := range logger.calls[0].fields {
		if field.Key == "error" {
			hasErrorField = true
		}
		if field.Key == "topic" {
			hasTopicField = true
		}
	}
	assert.True(t, hasErrorField)
	assert.True(t, hasTopicField)
}

func TestAdapterInfo(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	adapter := &Adapter{logger: logger}

	adapter.Info("subscriber started", watermillLog.LogFields{"topic": "orders"})

	require.Len(t, logger.calls, 1)
	assert.Equal(t, "Internal", logger.calls[0].method)
	assert.Equal(t, "subscriber started", logger.calls[0].message)
}

func TestAdapterDebug(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	adapter := &Adapter{logger: logger}

	adapter.Debug("processing message", watermillLog.LogFields{"id": "abc-123"})

	require.Len(t, logger.calls, 1)
	assert.Equal(t, "Internal", logger.calls[0].method)
	assert.Equal(t, "processing message", logger.calls[0].message)
}

func TestAdapterTrace(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	adapter := &Adapter{logger: logger}

	adapter.Trace("entering handler", watermillLog.LogFields{"handler": "process"})

	require.Len(t, logger.calls, 1)
	assert.Equal(t, "Trace", logger.calls[0].method)
	assert.Equal(t, "entering handler", logger.calls[0].message)
}

func TestAdapterWith(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	adapter := &Adapter{logger: logger}

	derived := adapter.With(watermillLog.LogFields{"component": "router"})

	require.NotNil(t, derived)
	assert.IsType(t, &Adapter{}, derived)
	require.Len(t, logger.withCalls, 1)

	hasComponentField := false
	for _, field := range logger.withCalls[0] {
		if field.Key == "component" && field.Value.String() == "router" {
			hasComponentField = true
		}
	}
	assert.True(t, hasComponentField)
}
