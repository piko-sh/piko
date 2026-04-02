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
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"piko.sh/piko/internal/logger/logger_domain"
)

type hooksTestContextKey string

type mockHook struct {
	lastError         error
	lastSpanName      string
	lastErrorMessage  string
	spanStartContexts []context.Context
	spanStartCalls    int
	reportErrorCalls  int
}

func (m *mockHook) OnSpanStart(ctx context.Context, spanName string, attrs []slog.Attr) (context.Context, func()) {
	m.spanStartCalls++
	m.lastSpanName = spanName
	m.spanStartContexts = append(m.spanStartContexts, ctx)
	finisher := func() {

	}
	return ctx, finisher
}

func (m *mockHook) OnReportError(ctx context.Context, err error, message string, attrs []slog.Attr) {
	m.reportErrorCalls++
	m.lastError = err
	m.lastErrorMessage = message
}

func TestLogger_AddSpanLifecycleHook_InstanceBased(t *testing.T) {
	baseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	hook := &mockHook{}
	logger.AddSpanLifecycleHook(hook)

	ctx, span, _ := logger.Span(context.Background(), "test-span")
	defer span.End()

	assert.Equal(t, 1, hook.spanStartCalls, "hook should be called once")
	assert.Equal(t, "test-span", hook.lastSpanName, "span name should be captured")
	assert.NotNil(t, ctx, "context should be returned")
}

func TestLogger_AddSpanLifecycleHook_MultipleHooks(t *testing.T) {
	baseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	hook1 := &mockHook{}
	hook2 := &mockHook{}

	logger.AddSpanLifecycleHook(hook1)
	logger.AddSpanLifecycleHook(hook2)

	_, span, _ := logger.Span(context.Background(), "test-span")
	defer span.End()

	assert.Equal(t, 1, hook1.spanStartCalls, "hook1 should be called once")
	assert.Equal(t, 1, hook2.spanStartCalls, "hook2 should be called once")
}

func TestLogger_AddSpanLifecycleHook_PropagatedToWithLogger(t *testing.T) {
	baseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	hook := &mockHook{}
	logger.AddSpanLifecycleHook(hook)

	derivedLogger := logger.With(logger_domain.String("key", "value"))

	_, span, _ := derivedLogger.Span(context.Background(), "derived-span")
	defer span.End()

	assert.Equal(t, 1, hook.spanStartCalls, "hook should be called on derived logger")
	assert.Equal(t, "derived-span", hook.lastSpanName)
}

func TestLogger_AddSpanLifecycleHook_PropagatedToWithContext(t *testing.T) {
	baseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	hook := &mockHook{}
	logger.AddSpanLifecycleHook(hook)

	ctx := context.WithValue(context.Background(), hooksTestContextKey("test-key"), "test-value")
	derivedLogger := logger.WithContext(ctx)

	_, span, _ := derivedLogger.Span(context.Background(), "context-span")
	defer span.End()

	assert.Equal(t, 1, hook.spanStartCalls, "hook should be called on context logger")
	assert.Equal(t, "context-span", hook.lastSpanName)
}

func TestLogger_ReportError_CallsHook(t *testing.T) {
	baseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	logger := logger_domain.NewLogger(baseLogger, otel.Tracer("test"))

	hook := &mockHook{}
	logger.AddSpanLifecycleHook(hook)

	ctx, span, spanLogger := logger.Span(context.Background(), "error-span")
	defer span.End()

	testErr := assert.AnError
	spanLogger.ReportError(span, testErr, "test error message")

	assert.Equal(t, 1, hook.reportErrorCalls, "ReportError hook should be called")
	assert.ErrorIs(t, hook.lastError, testErr, "error should be captured")
	assert.Equal(t, "test error message", hook.lastErrorMessage, "error message should be captured")
	assert.NotNil(t, ctx)
}

func TestLogger_HookIsolation_BetweenInstances(t *testing.T) {
	baseLogger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	logger1 := logger_domain.NewLogger(baseLogger, otel.Tracer("test1"))
	logger2 := logger_domain.NewLogger(baseLogger, otel.Tracer("test2"))

	hook1 := &mockHook{}
	hook2 := &mockHook{}

	logger1.AddSpanLifecycleHook(hook1)
	logger2.AddSpanLifecycleHook(hook2)

	_, span1, _ := logger1.Span(context.Background(), "span1")
	defer span1.End()

	_, span2, _ := logger2.Span(context.Background(), "span2")
	defer span2.End()

	assert.Equal(t, 1, hook1.spanStartCalls, "hook1 should only be called for logger1")
	assert.Equal(t, "span1", hook1.lastSpanName)

	assert.Equal(t, 1, hook2.spanStartCalls, "hook2 should only be called for logger2")
	assert.Equal(t, "span2", hook2.lastSpanName)
}

func TestMockLogger_AddSpanLifecycleHook(t *testing.T) {
	mockLogger := logger_domain.NewMockLogger()

	hook := &mockHook{}
	mockLogger.AddSpanLifecycleHook(hook)

	hooks := logger_domain.GetHooks(mockLogger)
	require.Len(t, hooks, 1, "mock logger should store the hook")
}
