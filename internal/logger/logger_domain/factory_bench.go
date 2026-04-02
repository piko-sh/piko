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

package logger_domain

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// noOpLogger is a logger that discards all log output.
type noOpLogger struct {
	// ctx is the context used to control cancellation and timeouts.
	ctx context.Context
}

var _ Logger = (*noOpLogger)(nil)

// Trace does nothing with the message and attributes.
func (*noOpLogger) Trace(_ string, _ ...Attr) {}

// Internal does nothing because this is a no-op logger.
func (*noOpLogger) Internal(_ string, _ ...Attr) {}

// Debug discards the debug message and attributes without logging.
func (*noOpLogger) Debug(_ string, _ ...Attr) {}

// Info discards the message without logging it.
func (*noOpLogger) Info(_ string, _ ...Attr) {}

// Notice discards the notice-level log message.
func (*noOpLogger) Notice(_ string, _ ...Attr) {}

// Warn discards a warning-level log message without any output.
func (*noOpLogger) Warn(_ string, _ ...Attr) {}

// Error discards the error message and attributes without logging.
func (*noOpLogger) Error(_ string, _ ...Attr) {}

// Enabled always returns false for the no-op logger.
//
// Returns bool which is always false.
func (*noOpLogger) Enabled(_ slog.Level) bool { return false }

// Panic does nothing as this is a no-op logger.
func (*noOpLogger) Panic(_ string, _ ...Attr) {}

// With returns the logger unchanged, ignoring any attributes.
//
// Returns Logger which is the same no-op logger instance.
func (l *noOpLogger) With(_ ...Attr) Logger {
	return l
}

// WithContext returns a new no-op logger with the given context.
//
// Returns Logger which is a no-op logger that discards all output.
func (*noOpLogger) WithContext(ctx context.Context) Logger {
	return &noOpLogger{ctx: ctx}
}

// WithSpanContext returns this no-op logger unchanged.
//
// Returns Logger which is the same no-op logger instance.
func (l *noOpLogger) WithSpanContext(_ context.Context) Logger {
	return l
}

// GetContext returns the context associated with the logger.
//
// Returns context.Context which is the stored context.
func (l *noOpLogger) GetContext() context.Context {
	return l.ctx
}

// Span returns the context, a no-op span, and this logger unchanged.
//
// Returns context.Context which is the input context passed through unchanged.
// Returns trace.Span which is a no-op span that performs no tracing.
// Returns Logger which is this logger instance.
func (l *noOpLogger) Span(ctx context.Context, _ string, _ ...slog.Attr) (context.Context, trace.Span, Logger) {
	return ctx, noop.Span{}, l
}

// ReportError does nothing and discards the error report.
func (*noOpLogger) ReportError(_ trace.Span, _ error, _ string, _ ...Attr) {}

// RunInSpan executes the given function without creating a trace span.
//
// Takes operation (func(context.Context, Logger) error) which is the
// function to run.
//
// Returns error when the provided function returns an error.
func (l *noOpLogger) RunInSpan(ctx context.Context, _ string, operation func(context.Context, Logger) error, _ ...Attr) error {
	return operation(ctx, l)
}

// AddSpanLifecycleHook does nothing as this is a no-op logger implementation.
func (*noOpLogger) AddSpanLifecycleHook(_ SpanLifecycleHook) {}

// WithoutAutoCaller returns the logger unchanged as this is a no-op
// implementation.
//
// Returns Logger which is the same no-op logger instance.
func (l *noOpLogger) WithoutAutoCaller() Logger {
	return l
}

// LogFactory provides a way to create log checkers.
type LogFactory struct{}

// DefaultFactory is the package-level log factory instance.
var DefaultFactory *LogFactory

// GetLoggerForPackage returns a logger for the named package.
//
// Returns Logger which is a no-op logger that discards all log output.
func (*LogFactory) GetLoggerForPackage(_ string) Logger {
	return &noOpLogger{ctx: context.Background()}
}

// GetLogger returns a logger for the given name.
//
// Returns Logger which is a no-op logger that discards all log output.
func GetLogger(_ string) Logger {
	return &noOpLogger{ctx: context.Background()}
}

// InitDefaultFactory sets up the default checker factory with the given logger.
func InitDefaultFactory(_ *slog.Logger) {}
