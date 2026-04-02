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

package logger_domain

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// MockLogger is a test double that implements the Logger interface,
// discarding all log output for testing code that needs a logger.
type MockLogger struct {
	// ctx holds the context for this logger instance.
	ctx context.Context

	// hooks stores span lifecycle hooks added during tests.
	hooks []SpanLifecycleHook
}

// NewMockLogger creates a new MockLogger instance for testing.
//
// Returns *MockLogger which is ready to use with a background context.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		ctx:   context.Background(),
		hooks: nil,
	}
}

// Enabled always returns false for the mock logger.
//
// Returns bool which is always false.
func (*MockLogger) Enabled(_ slog.Level) bool { return false }

// Error does nothing and discards all arguments.
func (*MockLogger) Error(_ string, _ ...Attr) {}

// Warn does nothing and discards all arguments.
func (*MockLogger) Warn(_ string, _ ...Attr) {}

// Notice does nothing, satisfying the logging interface.
func (*MockLogger) Notice(_ string, _ ...Attr) {}

// Info does nothing as this is a mock implementation.
func (*MockLogger) Info(_ string, _ ...Attr) {}

// Debug does nothing as a no-op method for testing.
func (*MockLogger) Debug(_ string, _ ...Attr) {}

// Internal does nothing as this is a stub for the logging interface.
func (*MockLogger) Internal(_ string, _ ...Attr) {}

// Trace does nothing as this is a mock implementation.
func (*MockLogger) Trace(_ string, _ ...Attr) {}

// Panic does nothing as a no-op for testing purposes.
func (*MockLogger) Panic(_ string, _ ...Attr) {}

// With returns this MockLogger unchanged, ignoring any given attributes.
//
// Returns Logger which is this same MockLogger instance.
func (m *MockLogger) With(_ ...Attr) Logger {
	return m
}

// WithContext creates a new MockLogger with the provided context.
//
// Returns Logger which is the new logger instance with the given context.
func (*MockLogger) WithContext(ctx context.Context) Logger {
	return &MockLogger{
		ctx:   ctx,
		hooks: nil,
	}
}

// GetContext returns the current context associated with this logger.
//
// Returns context.Context which is the context for this logger instance.
func (m *MockLogger) GetContext() context.Context {
	return m.ctx
}

// Span returns the provided context, a nil span, and the same MockLogger.
//
// Returns context.Context which is the same context that was passed in.
// Returns trace.Span which is always nil for the mock implementation.
// Returns Logger which is the same MockLogger instance.
func (m *MockLogger) Span(ctx context.Context, _ string, _ ...slog.Attr) (context.Context, trace.Span, Logger) {
	return WithLogger(ctx, m), nil, m
}

// ReportError does nothing as a mock for testing.
func (*MockLogger) ReportError(_ trace.Span, _ error, _ string, _ ...Attr) {}

// RunInSpan executes the provided function with the mock logger, ignoring span
// creation.
//
// Takes ctx (context.Context) which is passed through to the function.
// Takes operation (func(context.Context, Logger) error) which is the function
// to execute within the mock span.
//
// Returns error which is the error returned by operation.
func (m *MockLogger) RunInSpan(ctx context.Context, _ string, operation func(context.Context, Logger) error, _ ...Attr) error {
	return operation(WithLogger(ctx, m), m)
}

// AddSpanLifecycleHook is a no-op implementation that stores the hook for
// potential test verification.
//
// Takes hook (SpanLifecycleHook) which is stored for later inspection.
func (m *MockLogger) AddSpanLifecycleHook(hook SpanLifecycleHook) {
	m.hooks = append(m.hooks, hook)
}

// WithoutAutoCaller returns this MockLogger unchanged because
// MockLogger does not log anything and auto-caller has no effect.
//
// Returns Logger which is this same MockLogger instance.
func (m *MockLogger) WithoutAutoCaller() Logger {
	return m
}

// WithSpanContext returns this MockLogger unchanged because
// MockLogger does not track context.
//
// Returns Logger which is this same MockLogger instance.
func (m *MockLogger) WithSpanContext(_ context.Context) Logger {
	return m
}

// newMockLoggerWithContext creates a new MockLogger with the given context.
//
// Takes ctx (context.Context) which is the context for the new logger.
//
// Returns *MockLogger which is ready to use with the provided context.
func newMockLoggerWithContext(ctx context.Context) *MockLogger { //nolint:unused // exported via export_test.go
	return &MockLogger{
		ctx:   ctx,
		hooks: nil,
	}
}
