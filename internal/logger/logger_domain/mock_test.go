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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/logger/logger_domain"
)

type mockContextKey string

func TestNewMockLogger(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	require.NotNil(t, logger)
	assert.NotNil(t, logger.GetContext())
}

func TestNewMockLoggerWithContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), mockContextKey("test-key"), "test-value")
	logger := logger_domain.NewMockLoggerWithContext(ctx)

	require.NotNil(t, logger)
	assert.Equal(t, "test-value", logger.GetContext().Value(mockContextKey("test-key")))
}

func TestMockLogger_AllMethodsNoOp(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	assert.NotPanics(t, func() {
		logger.Trace("trace message", logger_domain.String("key", "value"))
		logger.Debug("debug message", logger_domain.Int("count", 1))
		logger.Info("info message", logger_domain.Bool("flag", true))
		logger.Notice("notice message")
		logger.Warn("warn message")
		logger.Error("error message", logger_domain.Error(errors.New("test")))
		logger.Panic("panic message")
	})
}

func TestMockLogger_With_ReturnsLogger(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	derivedLogger := logger.With(
		logger_domain.String("key1", "value1"),
		logger_domain.Int("key2", 42),
	)

	require.NotNil(t, derivedLogger)

	assert.NotPanics(t, func() {
		derivedLogger.Info("test")
	})
}

func TestMockLogger_WithContext_UpdatesContext(t *testing.T) {
	logger := logger_domain.NewMockLogger()
	originalCtx := logger.GetContext()

	newCtx := context.WithValue(context.Background(), mockContextKey("new-key"), "new-value")
	derivedLogger := logger.WithContext(newCtx)

	require.NotNil(t, derivedLogger)
	assert.NotEqual(t, originalCtx, derivedLogger.GetContext())
	assert.Equal(t, "new-value", derivedLogger.GetContext().Value(mockContextKey("new-key")))
}

func TestMockLogger_Span_ReturnsContextAndLogger(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	ctx := context.WithValue(context.Background(), mockContextKey("span-key"), "span-value")
	returnedCtx, span, spanLogger := logger.Span(ctx, "test-span",
		logger_domain.String("attr", "value"),
	)

	assert.Equal(t, "span-value", returnedCtx.Value(mockContextKey("span-key")), "should preserve parent context values")
	assert.Nil(t, span, "mock logger returns nil span")
	assert.NotNil(t, spanLogger, "should return a logger")

	_, fromLogger := logger_domain.From(returnedCtx, nil)
	assert.Equal(t, spanLogger, fromLogger, "should store logger in returned context")

	assert.NotPanics(t, func() {
		spanLogger.Info("test")
	})
}

func TestMockLogger_RunInSpan_ExecutesFunction(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	executed := false
	err := logger.RunInSpan(context.Background(), "test-span", func(ctx context.Context, log logger_domain.Logger) error {
		executed = true
		assert.NotNil(t, ctx)
		assert.NotNil(t, log)
		return nil
	})

	assert.True(t, executed, "function should be executed")
	assert.NoError(t, err)
}

func TestMockLogger_RunInSpan_PropagatesError(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	testErr := errors.New("test error")
	err := logger.RunInSpan(context.Background(), "test-span", func(ctx context.Context, log logger_domain.Logger) error {
		return testErr
	})

	assert.ErrorIs(t, err, testErr, "should propagate function error")
}

func TestMockLogger_ReportError_IsNoOp(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	assert.NotPanics(t, func() {
		logger.ReportError(nil, errors.New("test error"), "operation failed",
			logger_domain.String("context", "test"),
		)
	})
}

func TestMockLogger_AddSpanLifecycleHook_StoresHook(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	hook := &mockHook{}

	logger.AddSpanLifecycleHook(hook)

	hooks := logger_domain.GetHooks(logger)
	require.Len(t, hooks, 1)
	assert.Equal(t, hook, hooks[0])
}

func TestMockLogger_GetHooks_ReturnsEmptyInitially(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	hooks := logger_domain.GetHooks(logger)
	assert.Empty(t, hooks, "new mock logger should have no hooks")
}

func TestMockLogger_GetHooks_ReturnsMultipleHooks(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	hook1 := &mockHook{}
	hook2 := &mockHook{}
	hook3 := &mockHook{}

	logger.AddSpanLifecycleHook(hook1)
	logger.AddSpanLifecycleHook(hook2)
	logger.AddSpanLifecycleHook(hook3)

	hooks := logger_domain.GetHooks(logger)
	require.Len(t, hooks, 3)
	assert.Equal(t, hook1, hooks[0])
	assert.Equal(t, hook2, hooks[1])
	assert.Equal(t, hook3, hooks[2])
}

func TestMockLogger_ThreadSafety(t *testing.T) {
	logger := logger_domain.NewMockLogger()

	RunConcurrentTest(t, 50, func(id int) {

		logger.Info("concurrent test", logger_domain.Int("goroutine", id))
		logger.Error("concurrent error", logger_domain.Int("goroutine", id))

		derivedLogger := logger.With(logger_domain.Int("id", id))
		derivedLogger.Debug("test")

		_ = logger.RunInSpan(context.Background(), "concurrent-span", func(ctx context.Context, log logger_domain.Logger) error {
			log.Info("inside span")
			return nil
		})
	})

}
