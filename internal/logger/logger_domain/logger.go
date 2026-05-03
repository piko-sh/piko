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
	"encoding"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/caller"
	"piko.sh/piko/internal/json"
)

const (
	// LevelTrace is the most detailed logging level, used to trace code paths.
	LevelTrace = slog.Level(-8)

	// LevelInternal is a log level for internal framework messages.
	LevelInternal = slog.Level(-6)

	// LevelNotice is a log level between Info and Warn for notable operational
	// events.
	LevelNotice = slog.Level(2)

	// FieldStrContext is the log field key for context attributes.
	FieldStrContext = "ctx"

	// FieldStrMethod is the logging key for recording method names.
	FieldStrMethod = "method"

	// FieldStrComponent is the logging key for the component name.
	FieldStrComponent = "component"

	// FieldStrAdapter is the key for the adapter name field in log entries.
	FieldStrAdapter = "adapter"

	// FieldStrService is the key for the service name field in structured logging.
	FieldStrService = "service"

	// FieldStrError is the key for error message fields in log entries.
	FieldStrError = "error"

	// FieldStrPath is the key for file path values in log messages.
	FieldStrPath = "path"

	// FieldStrFile is the key for the file name attribute in log spans.
	FieldStrFile = "file"

	// FieldStrDir is the legacy key for directory fields in log entries.
	FieldStrDir = "dir"

	// callerSkipFrames is the number of stack frames to skip when finding the
	// caller's source location for slog's AddSource feature.
	// Stack: runtime.Callers -> log -> Info/Error -> User code.
	callerSkipFrames = 3

	// maxStackTraceFrames is the maximum number of stack frames to capture.
	maxStackTraceFrames = 64

	// callerUnknown is the placeholder value when caller information cannot be
	// resolved.
	callerUnknown = "<unknown>"
)

var (
	// otelAttrPool reuses OpenTelemetry attribute slices to reduce allocation
	// pressure during span attribute propagation.
	otelAttrPool = sync.Pool{
		New: func() any {
			return new(make([]attribute.KeyValue, 0, 8))
		},
	}

	// callerCache maps program counters to their cached caller info.
	// For a given PC, the package and method names are fixed at compile time,
	// so we only need to compute them once per unique call site.
	callerCache sync.Map
)

// Logger is the primary logging interface for the Piko framework.
//
// It provides structured logging with multiple severity levels, context
// propagation, and deep integration with OpenTelemetry for distributed
// tracing. Implements logger.Logger and logger_domain.Logger.
type Logger interface {
	// Enabled reports whether the logger handles records at the given level.
	// Callers can use this to skip expensive argument evaluation when the
	// message would be discarded.
	//
	// Takes level (slog.Level) which is the level to check.
	//
	// Returns bool which is true when the level is enabled.
	Enabled(level slog.Level) bool

	// Trace logs a message at trace level with optional attributes.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...Attr) which provides optional structured attributes.
	Trace(message string, arguments ...Attr)

	// Internal logs a message at internal level for debugging.
	//
	// Takes message (string) which is the log message to record.
	// Takes arguments (...Attr) which provides optional structured attributes.
	Internal(message string, arguments ...Attr)

	// Debug logs a message at debug level with the given attributes.
	//
	// Takes message (string) which is the log message to record.
	// Takes arguments (...Attr) which provides optional key-value attributes.
	Debug(message string, arguments ...Attr)

	// Info logs a message at the info level with optional attributes.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...Attr) which are optional key-value pairs to include.
	Info(message string, arguments ...Attr)

	// Notice logs a message at notice level with optional attributes.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...Attr) which are optional key-value pairs for extra context.
	Notice(message string, arguments ...Attr)

	// Warn logs a message at the warning level.
	//
	// Takes message (string) which is the warning message to log.
	// Takes arguments (...Attr) which provides optional structured attributes.
	Warn(message string, arguments ...Attr)

	// Error logs a message at the error level with optional attributes.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...Attr) which are optional structured attributes.
	Error(message string, arguments ...Attr)

	// Panic logs a message at panic level with the given attributes.
	//
	// Takes message (string) which is the message to log.
	// Takes arguments (...Attr) which are optional attributes to include.
	Panic(message string, arguments ...Attr)

	// With returns a new Logger that includes the given attributes.
	//
	// Takes arguments (...Attr) which are the attributes to add to the logger.
	//
	// Returns Logger which is a new logger with the attributes included.
	With(arguments ...Attr) Logger

	// WithContext returns a new Logger with the given context attached.
	//
	// Takes ctx (context.Context) which is the context to use for logging.
	//
	// Returns Logger which is a new logger that uses the given context.
	WithContext(ctx context.Context) Logger

	// GetContext returns the context for this operation.
	//
	// Returns context.Context which provides cancellation and deadline control.
	GetContext() context.Context

	// Span creates a new tracing span with the given name and attributes.
	//
	// Takes spanName (string) which identifies the span.
	// Takes attrs (...slog.Attr) which provides additional span attributes.
	//
	// Returns context.Context which carries the new span.
	// Returns trace.Span which is the created span for manual control.
	// Returns Logger which is a child logger bound to the span context.
	Span(ctx context.Context, spanName string, attrs ...slog.Attr) (context.Context, trace.Span, Logger)

	// WithSpanContext returns a logger bound to the given span context without
	// calling WithAttrs on the underlying handler. Acts as a low-allocation
	// alternative to Span() for hot paths where the OTEL span is created
	// directly and only the logger is needed for correlation.
	//
	// Use when:
	//   - The span is created via otel.Tracer().Start() directly
	//   - Attributes are set on the span via span.SetAttributes()
	//   - A logger that correlates with the span is needed without
	//     repeating the attributes on every log line
	//
	// Takes ctx (context.Context) which contains the span from tracer.Start().
	//
	// Returns Logger which is bound to the span context for log correlation.
	WithSpanContext(ctx context.Context) Logger

	// ReportError records an error on the given span with a message and
	// attributes.
	//
	// Takes span (trace.Span) which is the span to record the error on.
	// Takes err (error) which is the error to report.
	// Takes message (string) which describes the error context.
	// Takes attrs (...Attr) which provides optional attributes for the error.
	ReportError(span trace.Span, err error, message string, attrs ...Attr)

	// RunInSpan executes the given function within a new
	// trace span.
	//
	// Takes spanName (string) which identifies the span.
	// Takes operation (func(context.Context, Logger) error)
	// which is the function to run.
	// Takes attrs (...Attr) which provides optional span
	// attributes.
	//
	// Returns error which propagates any failure raised by the operation.
	RunInSpan(ctx context.Context, spanName string, operation func(context.Context, Logger) error, attrs ...Attr) error

	// AddSpanLifecycleHook registers a hook for span lifecycle events.
	//
	// The hook is called during span creation and error reporting. Hooks are
	// passed on to derived loggers created via With() or WithContext().
	//
	// Takes hook (SpanLifecycleHook) which handles span lifecycle callbacks.
	AddSpanLifecycleHook(hook SpanLifecycleHook)

	// WithoutAutoCaller returns a logger that disables automatic caller capture,
	// which by default records the caller's package and method on every log call.
	//
	// Loggers created via GetLogger() automatically capture KeyContext (package)
	// and KeyMethod on every call. Use for performance-critical paths
	// where caller capture overhead is unacceptable.
	//
	// Returns Logger which does not capture caller information.
	WithoutAutoCaller() Logger
}

// Attr represents a key-value pair for structured logging. It is a type
// alias for slog.Attr.
type Attr = slog.Attr

// StackTrace represents a captured stack trace with pooled memory management.
// It provides special formatting for pretty-printing and does not implement
// String() to stop slog from converting it to a string too early.
//
// After use, Release() should be called to return the backing slice to the
// pool. This enables zero-allocation stack trace capture after warmup.
type StackTrace struct {
	// poolPtr points to the pooled slice; used to return it when released.
	poolPtr *[]string

	// frames holds the stack frame strings for this trace.
	frames []string
}

// Frames returns the stack frames for iteration.
//
// Returns []string which contains the stack frames.
func (st StackTrace) Frames() []string {
	return st.frames
}

// Release returns the backing slice to the pool and must be called after use.
// Safe to call multiple times or on a zero-value StackTrace.
func (st *StackTrace) Release() {
	if st.poolPtr != nil {
		*st.poolPtr = st.frames[:0]
		frameSlicePool.Put(st.poolPtr)
		st.poolPtr = nil
		st.frames = nil
	}
}

// LogValue implements slog.LogValuer to serialise the stack trace properly.
// This returns the frames slice so that slog serialises the actual frames
// rather than the struct's unexported fields.
//
// Returns slog.Value which contains the frames for proper serialisation.
func (st StackTrace) LogValue() slog.Value {
	return slog.AnyValue(st.frames)
}

// callerCacheEntry holds cached caller data for a program counter.
// It pre-builds slog.Attr values to avoid allocation on every log call.
type callerCacheEntry struct {
	// ctxAttr is a pre-built KeyContext attribute for the caller.
	ctxAttr Attr

	// methodAttr is the pre-built slog.Attr for the method name.
	methodAttr Attr
}

// slogLogger implements the Logger interface using Go's standard slog package.
// It provides structured logging with OpenTelemetry tracing support.
type slogLogger struct {
	// tracer is the OpenTelemetry tracer used to create spans for distributed
	// tracing.
	tracer trace.Tracer

	// ctx stores the context for hook calls and logging checks.
	ctx context.Context

	// stackTraceProvider captures call stack data when logging errors.
	stackTraceProvider stackTraceProvider

	// logger holds the structured logger used for log output.
	logger *slog.Logger

	// hooks stores callbacks that run during span lifecycle events.
	hooks []SpanLifecycleHook

	// attrs holds attributes from With() calls, added to spans for extra context.
	attrs []slog.Attr

	// hooksMutex guards the hooks slice for safe concurrent access.
	hooksMutex sync.RWMutex

	// useDynamicDefault when true makes getLogger return slog.Default()
	// instead of the stored logger. This keeps package-level loggers in sync
	// with runtime settings.
	useDynamicDefault bool

	// autoCaller enables automatic capture of the caller function name on each
	// log call. Only checked after Enabled(), so disabled levels have no overhead.
	autoCaller bool
}

// AddSpanLifecycleHook registers a hook to receive span lifecycle events.
// The hook is called for this logger and any derived loggers.
//
// Takes hook (SpanLifecycleHook) which handles span lifecycle events.
//
// Safe for concurrent use.
func (l *slogLogger) AddSpanLifecycleHook(hook SpanLifecycleHook) {
	l.hooksMutex.Lock()
	defer l.hooksMutex.Unlock()
	l.hooks = append(l.hooks, hook)
}

// Enabled reports whether the logger handles records at the given level.
//
// Takes level (slog.Level) which is the level to check.
//
// Returns bool which is true when the level is enabled.
func (l *slogLogger) Enabled(level slog.Level) bool {
	return l.getLogger().Handler().Enabled(l.ctx, level)
}

// Trace logs a message at TRACE level, the most granular diagnostic
// information.
//
// Used for loop iterations, per-node processing, variable state dumps, and
// high-frequency operations. This level is reserved for framework internals;
// webdevs should not use this level.
//
// Takes message (string) which specifies the log message.
// Takes arguments (...Attr) which provides optional structured attributes.
func (l *slogLogger) Trace(message string, arguments ...Attr) {
	l.log(LevelTrace, message, arguments...)
}

// Internal logs a message at INTERNAL level for framework surface
// operations.
//
// Used for service registration, cache operations, adapter lifecycle, and
// validation passes. This level is reserved for framework internals; webdevs
// should use Debug instead.
//
// Takes message (string) which is the log message to record.
// Takes arguments (...Attr) which provides optional structured attributes.
func (l *slogLogger) Internal(message string, arguments ...Attr) {
	l.log(LevelInternal, message, arguments...)
}

// Debug logs a message at DEBUG level for user application debugging.
//
// Used for request/response details, business logic checkpoints, and
// variable inspection. This level is owned by webdevs; framework code
// should use Internal or Trace instead.
//
// Takes message (string) which is the message to log.
// Takes arguments (...Attr) which provides structured key-value pairs.
func (l *slogLogger) Debug(message string, arguments ...Attr) {
	l.log(slog.LevelDebug, message, arguments...)
}

// Info logs a message at INFO level for normal operations affecting the
// application. Used for successful completions like page renders, route
// registrations, and request completions.
//
// Takes message (string) which is the message to log.
// Takes arguments (...Attr) which provides structured attributes
// for the log entry.
func (l *slogLogger) Info(message string, arguments ...Attr) {
	l.log(slog.LevelInfo, message, arguments...)
}

// Notice logs a message at NOTICE level for important lifecycle events.
// Used for service startup, shutdown, and major state changes.
//
// Takes message (string) which is the log message text.
// Takes arguments (...Attr) which provides optional structured attributes.
func (l *slogLogger) Notice(message string, arguments ...Attr) {
	l.log(LevelNotice, message, arguments...)
}

// Warn logs a message at WARN level for recoverable or unexpected events.
// Used for configuration defaults, rate limit warnings, and deprecated
// feature usage.
//
// Takes message (string) which is the log message to record.
// Takes arguments (...Attr) which provides optional structured attributes.
func (l *slogLogger) Warn(message string, arguments ...Attr) {
	l.log(slog.LevelWarn, message, arguments...)
}

// Error logs a message at ERROR level with a full stack trace for maximum
// context. Used for operation failures and actionable alerts requiring
// investigation.
//
// Takes message (string) which is the log message to record.
// Takes arguments (...Attr) which provides optional structured attributes.
func (l *slogLogger) Error(message string, arguments ...Attr) {
	l.logWithStack(slog.LevelError, message, arguments...)
}

// Panic logs a message at ERROR level with a stack trace, then panics.
// Use sparingly for unrecoverable errors that should crash the application.
//
// Takes message (string) which is the message to log and panic with.
// Takes arguments (...Attr) which are optional structured attributes to include.
func (l *slogLogger) Panic(message string, arguments ...Attr) {
	l.logWithStack(slog.LevelError, message, arguments...)
	panic(message)
}

// ReportError logs an error with a stack trace and records it on the provided
// span.
//
// If the error is nil, the call is a no-op. The span's status is set to Error when
// the span is recording.
//
// Takes span (trace.Span) which receives the error recording and status.
// Takes err (error) which is the error to log and record.
// Takes message (string) which describes the error context.
// Takes attrs (...Attr) which provides additional logging attributes.
func (l *slogLogger) ReportError(span trace.Span, err error, message string, attrs ...Attr) {
	if err == nil {
		return
	}
	allAttrs := make([]Attr, 0, len(attrs)+1)
	allAttrs = append(allAttrs, attrs...)
	allAttrs = append(allAttrs, Error(err))
	l.logWithStack(slog.LevelError, message, allAttrs...)

	if span != nil && span.IsRecording() {
		span.RecordError(err, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, message)
	}

	hooks := l.getHooks()
	for _, hook := range hooks {
		hook.OnReportError(l.ctx, err, message, attrs)
	}
}

// spanWithFinishers wraps a trace.Span to execute cleanup functions when the
// span ends.
type spanWithFinishers struct {
	trace.Span

	// finishers holds cleanup functions to run when the span ends, called in
	// reverse order.
	finishers []func()

	// endOnce guards single execution of End cleanup logic.
	endOnce sync.Once
}

// End finishes the span, executing all registered finishers in reverse order
// first.
//
// Takes options (...trace.SpanEndOption) which configures span end behaviour.
//
// Safe to call multiple times; subsequent calls are no-ops.
func (s *spanWithFinishers) End(options ...trace.SpanEndOption) {
	s.endOnce.Do(func() {
		for i := len(s.finishers) - 1; i >= 0; i-- {
			s.finishers[i]()
		}
		s.Span.End(options...)
	})
}

// Span starts a new OpenTelemetry span and returns the enriched context,
// span, and a logger bound to it.
//
// Takes spanName (string) which identifies the span in traces.
// Takes attrs (...Attr) which provides optional attributes to attach to the
// span.
//
// Returns context.Context which contains the new span context.
// Returns trace.Span which is the started span; the caller must call End when
// the operation completes.
// Returns Logger which is bound to the span context for correlated logging.
func (l *slogLogger) Span(ctx context.Context, spanName string, attrs ...Attr) (context.Context, trace.Span, Logger) {
	spanCtx, otelSpan := l.tracer.Start(ctx, spanName)

	hooksCopy := l.getHooks()

	if len(attrs) == 0 && len(l.attrs) == 0 && len(hooksCopy) == 0 {
		spanCtx = WithLogger(spanCtx, l)
		return spanCtx, otelSpan, l
	}

	allSpanAttrs := l.mergeSpanAttrs(attrs)
	l.setSpanOtelAttrs(otelSpan, allSpanAttrs)

	spanCtx, finishers := l.callHooksOnSpanStart(spanCtx, spanName, attrs, hooksCopy)

	wrappedSpan := &spanWithFinishers{
		Span:      otelSpan,
		finishers: finishers,
		endOnce:   sync.Once{},
	}

	finalLogger := l.createSpanLogger(spanCtx, attrs, allSpanAttrs, hooksCopy)
	spanCtx = WithLogger(spanCtx, finalLogger)
	return spanCtx, wrappedSpan, finalLogger
}

// With returns a new Logger with the given attributes added to every
// log entry. If no attributes are given, the original logger is
// returned.
//
// The attributes are also stored and will be set on any spans
// created via Span or RunInSpan.
//
// Takes attrs (...Attr) which are the attributes to add to each
// log entry.
//
// Returns Logger which is a new logger with the attributes applied.
//
// Safe for concurrent use. The internal hooks are copied under a
// read lock.
func (l *slogLogger) With(attrs ...Attr) Logger {
	if len(attrs) == 0 {
		return l
	}
	newHandler := l.getLogger().Handler().WithAttrs(attrs)

	var hooksCopy []SpanLifecycleHook
	if len(l.hooks) > 0 {
		l.hooksMutex.RLock()
		hooksCopy = make([]SpanLifecycleHook, len(l.hooks))
		copy(hooksCopy, l.hooks)
		l.hooksMutex.RUnlock()
	}

	combinedAttrs := make([]slog.Attr, len(l.attrs), len(l.attrs)+len(attrs))
	copy(combinedAttrs, l.attrs)
	combinedAttrs = append(combinedAttrs, attrs...)

	return &slogLogger{
		logger:             slog.New(newHandler),
		tracer:             l.tracer,
		ctx:                l.ctx,
		hooks:              hooksCopy,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: l.stackTraceProvider,
		useDynamicDefault:  false,
		attrs:              combinedAttrs,
		autoCaller:         l.autoCaller,
	}
}

// WithSpanContext returns a logger bound to the given span context without calling
// WithAttrs on the underlying handler. Acts as a low-allocation alternative to
// Span() for hot paths where the OTEL span is created directly and only the logger
// is needed for correlation.
//
// Saves ~400MB of allocations over 500K requests compared to Span() by avoiding the
// WithAttrs() call on the handler chain. The trade-off is that span attributes set
// via span.SetAttributes() won't appear on log lines, but logs will still be
// correlated with the span via trace context.
//
// Returns Logger which is bound to the span context for log correlation.
func (l *slogLogger) WithSpanContext(ctx context.Context) Logger {
	if l.ctx == ctx {
		return l
	}
	return l.cloneWithContext(ctx)
}

// WithContext returns a new Logger with the given context for trace linking.
// If the context is the same as the current one, the original logger is
// returned.
//
// Returns Logger which is either a new logger with the updated context or the
// original logger if the context has not changed.
func (l *slogLogger) WithContext(ctx context.Context) Logger {
	if l.ctx == ctx {
		return l
	}
	return l.cloneWithContext(ctx)
}

// GetContext returns the context associated with this logger instance.
//
// Returns context.Context which is the context bound to this logger.
func (l *slogLogger) GetContext() context.Context {
	return l.ctx
}

// WithoutAutoCaller returns a logger that disables automatic caller capture.
// By default, loggers from GetLogger capture the caller's package and method
// on every call; use this for performance-critical paths.
//
// Returns Logger which does not capture caller information.
//
// Safe for concurrent use.
func (l *slogLogger) WithoutAutoCaller() Logger {
	if !l.autoCaller {
		return l
	}

	var hooksCopy []SpanLifecycleHook
	if len(l.hooks) > 0 {
		l.hooksMutex.RLock()
		hooksCopy = make([]SpanLifecycleHook, len(l.hooks))
		copy(hooksCopy, l.hooks)
		l.hooksMutex.RUnlock()
	}

	return &slogLogger{
		logger:             l.logger,
		tracer:             l.tracer,
		ctx:                l.ctx,
		hooks:              hooksCopy,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: l.stackTraceProvider,
		useDynamicDefault:  l.useDynamicDefault,
		attrs:              l.attrs,
		autoCaller:         false,
	}
}

// RunInSpan executes the provided function within a new span, handling
// lifecycle and errors.
//
// Any error from the operation is reported to the span and the span status is
// set to Error. On success, the span status is set to Ok.
//
// Takes spanName (string) which identifies the span for tracing.
// Takes operation (func(...)) which is the function to execute within the span.
// Takes attrs (...Attr) which are optional attributes to attach to the span.
//
// Returns error which propagates any failure raised by the operation.
func (l *slogLogger) RunInSpan(ctx context.Context, spanName string, operation func(context.Context, Logger) error, attrs ...Attr) error {
	spanCtx, span, spanLog := l.Span(ctx, spanName, attrs...)
	defer span.End()
	err := operation(spanCtx, spanLog)
	if err != nil {
		spanLog.Trace("span finished with error", Error(err))
		if span != nil && span.IsRecording() {
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, "span finished with error")
		}
		return err
	}
	span.SetStatus(codes.Ok, "success")
	return nil
}

// mergeSpanAttrs combines logger-level attributes with span-level attributes.
// Reuses existing slices when possible to avoid memory allocation.
//
// Takes attrs ([]Attr) which contains the span-level attributes to merge.
//
// Returns []slog.Attr which contains the combined logger and span attributes.
func (l *slogLogger) mergeSpanAttrs(attrs []Attr) []slog.Attr {
	switch {
	case len(attrs) == 0:
		return l.attrs
	case len(l.attrs) == 0:
		return attrs
	default:
		allSpanAttrs := make([]slog.Attr, len(l.attrs), len(l.attrs)+len(attrs))
		copy(allSpanAttrs, l.attrs)
		return append(allSpanAttrs, attrs...)
	}
}

// setSpanOtelAttrs sets OpenTelemetry attributes on the span using a pooled
// slice for better performance and lower memory use.
//
// Takes otelSpan (trace.Span) which is the span to set attributes on.
// Takes allSpanAttrs ([]slog.Attr) which contains the attributes to convert
// and apply.
func (*slogLogger) setSpanOtelAttrs(otelSpan trace.Span, allSpanAttrs []slog.Attr) {
	if len(allSpanAttrs) == 0 {
		return
	}
	kvsPtr, ok := otelAttrPool.Get().(*[]attribute.KeyValue)
	if !ok {
		kvsPtr = new(make([]attribute.KeyValue, 0, len(allSpanAttrs)))
	}
	kvs := (*kvsPtr)[:0]
	for _, a := range allSpanAttrs {
		addAttrRecursive(&kvs, "", a)
	}
	otelSpan.SetAttributes(kvs...)
	*kvsPtr = kvs[:0]
	otelAttrPool.Put(kvsPtr)
}

// callHooksOnSpanStart invokes all hook OnSpanStart callbacks and collects
// finishers.
//
// Takes spanCtx (context.Context) which is the span context to pass to
// hooks.
// Takes spanName (string) which identifies the span.
// Takes attrs ([]Attr) which contains span attributes.
// Takes hooks ([]SpanLifecycleHook) which are the hooks to invoke.
//
// Returns context.Context which is the potentially modified span context.
// Returns []func() which contains finisher callbacks to run when the span ends.
func (*slogLogger) callHooksOnSpanStart(
	spanCtx context.Context, spanName string, attrs []Attr, hooks []SpanLifecycleHook,
) (context.Context, []func()) {
	var finishers []func()
	if len(hooks) > 0 {
		finishers = make([]func(), 0, len(hooks))
	}
	for _, hook := range hooks {
		var finisher func()
		spanCtx, finisher = hook.OnSpanStart(spanCtx, spanName, attrs)
		if finisher != nil {
			finishers = append(finishers, finisher)
		}
	}
	return spanCtx, finishers
}

// createSpanLogger creates a new logger bound to the span context.
// Reuses the existing slog.Logger when no attrs are provided.
//
// Takes attrs ([]Attr) which specifies attributes to add to the handler.
// Takes allSpanAttrs ([]Attr) which contains all attributes for the span.
// Takes hooksCopy ([]SpanLifecycleHook) which provides lifecycle hooks.
//
// Returns *slogLogger which is the new logger configured for the span.
func (l *slogLogger) createSpanLogger(
	spanCtx context.Context, attrs, allSpanAttrs []Attr, hooksCopy []SpanLifecycleHook,
) *slogLogger {
	var newLogger *slog.Logger
	if len(attrs) > 0 {
		newHandler := l.getLogger().Handler().WithAttrs(attrs)
		newLogger = slog.New(newHandler)
	} else {
		newLogger = l.getLogger()
	}

	return &slogLogger{
		logger:             newLogger,
		tracer:             l.tracer,
		ctx:                spanCtx,
		hooks:              hooksCopy,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: l.stackTraceProvider,
		useDynamicDefault:  false,
		attrs:              allSpanAttrs,
	}
}

// cloneWithContext creates a copy of the logger with a new context.
//
// Returns *slogLogger which is a shallow copy with the given context and a
// deep copy of the hooks slice.
//
// Safe for concurrent use. Acquires a read lock on hooks during the copy.
func (l *slogLogger) cloneWithContext(ctx context.Context) *slogLogger {
	var hooksCopy []SpanLifecycleHook
	if len(l.hooks) > 0 {
		l.hooksMutex.RLock()
		hooksCopy = make([]SpanLifecycleHook, len(l.hooks))
		copy(hooksCopy, l.hooks)
		l.hooksMutex.RUnlock()
	}

	return &slogLogger{
		logger:             l.logger,
		tracer:             l.tracer,
		ctx:                ctx,
		hooks:              hooksCopy,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: l.stackTraceProvider,
		useDynamicDefault:  l.useDynamicDefault,
		attrs:              l.attrs,
		autoCaller:         l.autoCaller,
	}
}

// getLogger returns the effective logger for this instance.
//
// If useDynamicDefault is true, it returns slog.Default() which may have been
// reconfigured after this logger was created (e.g., by AddPrettyOutput).
// This keeps package-level loggers in sync with runtime configuration.
//
// Returns *slog.Logger which is either the instance logger or the current
// default logger based on the useDynamicDefault setting.
func (l *slogLogger) getLogger() *slog.Logger {
	if l.useDynamicDefault {
		return slog.Default()
	}
	return l.logger
}

// getHooks returns a copy of the current hooks for safe iteration.
//
// Returns []SpanLifecycleHook which contains the instance hooks,
// or nil if no hooks are registered.
//
// Safe for concurrent use. Uses a read lock when accessing instance hooks.
func (l *slogLogger) getHooks() []SpanLifecycleHook {
	if len(l.hooks) == 0 {
		return nil
	}

	l.hooksMutex.RLock()
	defer l.hooksMutex.RUnlock()
	result := make([]SpanLifecycleHook, len(l.hooks))
	copy(result, l.hooks)
	return result
}

// addContextCauseIfPresent enriches a log record with the
// descriptive cancellation cause from the logger's context when
// available.
//
// Only called at WARN level and above to avoid overhead on hot
// paths. The cause is only added when it differs from the generic
// sentinel (context.Canceled / context.DeadlineExceeded), meaning
// someone set a descriptive cause via WithTimeoutCause or
// WithCancelCause.
//
// Takes r (*slog.Record) which is the record to enrich with the
// context cause.
func (l *slogLogger) addContextCauseIfPresent(r *slog.Record) {
	if l.ctx == nil {
		return
	}
	cause := context.Cause(l.ctx)
	if cause == nil {
		return
	}
	if !errors.Is(cause, l.ctx.Err()) {
		r.AddAttrs(slog.String("context.cause", cause.Error()))
	}
}

// log writes a log entry for non-error severity levels.
// It uses runtime.Callers directly instead of the full CaptureStackTrace
// to avoid the cost of frame iteration and string formatting.
//
// Takes level (slog.Level) which sets the severity of the log message.
// Takes message (string) which provides the log message text.
// Takes arguments (...Attr) which supplies optional structured attributes.
func (l *slogLogger) log(level slog.Level, message string, arguments ...Attr) {
	if !l.getLogger().Enabled(l.ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(callerSkipFrames, pcs[:])
	r := slog.NewRecord(time.Now(), level, message, pcs[0])

	if l.autoCaller {
		ctxAttr, methodAttr := callerAttrsAtSkip(2)
		r.AddAttrs(ctxAttr, methodAttr)
	}

	r.AddAttrs(arguments...)
	if level >= slog.LevelWarn {
		l.addContextCauseIfPresent(&r)
	}
	_ = l.getLogger().Handler().Handle(l.ctx, r)
}

// logWithStack logs a message at the given level and captures a stack trace.
//
// Takes level (slog.Level) which sets the log severity.
// Takes message (string) which provides the log message.
// Takes arguments (...Attr) which supplies optional structured attributes.
func (l *slogLogger) logWithStack(level slog.Level, message string, arguments ...Attr) {
	if !l.getLogger().Enabled(l.ctx, level) {
		return
	}

	var ctxAttr, methodAttr Attr
	if l.autoCaller {
		ctxAttr, methodAttr = callerAttrsAtSkip(2)
	}

	pc, stackTrace := l.stackTraceProvider.CaptureStackTrace(callerSkipFrames+1, maxStackTraceFrames)
	defer stackTrace.Release()

	if pc == 0 {
		var pcs [1]uintptr
		runtime.Callers(callerSkipFrames, pcs[:])
		r := slog.NewRecord(time.Now(), level, message, pcs[0])
		if l.autoCaller {
			r.AddAttrs(ctxAttr, methodAttr)
		}
		r.AddAttrs(arguments...)
		if level >= slog.LevelWarn {
			l.addContextCauseIfPresent(&r)
		}
		_ = l.getLogger().Handler().Handle(l.ctx, r)
		return
	}
	r := slog.NewRecord(time.Now(), level, message, pc)
	if l.autoCaller {
		r.AddAttrs(ctxAttr, methodAttr)
	}
	r.AddAttrs(arguments...)
	if len(stackTrace.Frames()) > 0 {
		r.AddAttrs(slog.Any("stack_trace", stackTrace))
	}
	if level >= slog.LevelWarn {
		l.addContextCauseIfPresent(&r)
	}
	_ = l.getLogger().Handler().Handle(l.ctx, r)
}

// LevelName returns the display name for a log level.
//
// Custom levels (TRACE, INTERNAL, NOTICE) return their proper names. Standard
// levels use the slog String method.
//
// Takes level (slog.Level) which specifies the log level to convert.
//
// Returns string which is the readable name for the level.
func LevelName(level slog.Level) string {
	switch level {
	case LevelTrace:
		return "TRACE"
	case LevelInternal:
		return "INTERNAL"
	case LevelNotice:
		return "NOTICE"
	default:
		return level.String()
	}
}

// ReplaceLevelAttr returns a ReplaceAttr function for slog handlers that
// properly formats custom log levels (TRACE, INTERNAL, NOTICE) instead of
// displaying them as offsets like "INFO+2" or "DEBUG-2".
//
// Takes a (slog.Attr) which is the attribute to inspect and possibly
// rewrite.
//
// Returns slog.Attr which is the attribute with the level value
// replaced by its human-readable name when applicable.
func ReplaceLevelAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.LevelKey {
		if level, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(LevelName(level))
		}
	}
	return a
}

// Field creates a log attribute with any value type.
//
// Takes key (string) which specifies the attribute name.
// Takes value (any) which provides the attribute value.
//
// Returns Attr which is the constructed log attribute.
func Field(key string, value any) Attr { return slog.Any(key, value) }

// String creates a structured log attribute with a string value.
//
// Takes key (string) which specifies the attribute name.
// Takes value (string) which specifies the attribute value.
//
// Returns Attr which is the structured log attribute.
func String(key string, value string) Attr { return slog.String(key, value) }

// Strings creates a log attribute from a slice of strings joined with commas.
//
// Takes key (string) which specifies the attribute name.
// Takes value ([]string) which provides the strings to join.
//
// Returns Attr which contains the joined string value.
func Strings(key string, value []string) Attr { return slog.String(key, strings.Join(value, ",")) }

// Int64 creates a structured log attribute with an int64 value.
//
// Takes key (string) which is the attribute name.
// Takes value (int64) which is the number to log.
//
// Returns Attr which is the structured logging attribute.
func Int64(key string, value int64) Attr { return slog.Int64(key, value) }

// Int creates a structured log attribute with an integer value.
//
// Takes key (string) which identifies the attribute name.
// Takes value (int) which specifies the integer to store.
//
// Returns Attr which contains the key-value pair for structured logging.
func Int(key string, value int) Attr { return slog.Int(key, value) }

// Uint64 creates a structured log attribute with a uint64 value.
//
// Takes key (string) which names the attribute in the log output.
// Takes value (uint64) which is the number to log.
//
// Returns Attr which is a structured logging attribute for use with slog.
func Uint64(key string, value uint64) Attr { return slog.Uint64(key, value) }

// Float64 creates a structured log attribute with a float64 value.
//
// Takes key (string) which names the attribute in log output.
// Takes value (float64) which is the number to log.
//
// Returns Attr which is the structured log attribute ready for use.
func Float64(key string, value float64) Attr { return slog.Float64(key, value) }

// Bool creates a structured log attribute with a boolean value.
//
// Takes key (string) which specifies the attribute name.
// Takes value (bool) which specifies the boolean value.
//
// Returns Attr which is the structured log attribute.
func Bool(key string, value bool) Attr { return slog.Bool(key, value) }

// Time creates a structured log attribute with a time value.
//
// Takes key (string) which names the attribute in log output.
// Takes value (time.Time) which provides the timestamp for this attribute.
//
// Returns Attr which is the structured log attribute ready for use.
func Time(key string, value time.Time) Attr { return slog.Time(key, value) }

// Duration creates a structured log attribute for a time duration.
//
// Takes key (string) which names the attribute in log output.
// Takes value (time.Duration) which is the duration to log.
//
// Returns Attr which is the structured log attribute ready for use.
func Duration(key string, value time.Duration) Attr { return slog.Duration(key, value) }

// Error creates a structured log attribute with an error value.
// If the error is nil, it logs the string "<nil>".
//
// Takes value (error) which is the error to include in the log output.
//
// Returns Attr which is the structured log attribute containing the
// error message.
func Error(value error) Attr {
	if value == nil {
		return slog.String("error", "<nil>")
	}
	return slog.String("error", value.Error())
}

// Caller creates a log attribute with the caller's function name.
//
// It captures which function called the log statement, so you do not need to
// set KeyMethod yourself.
//
// Returns Attr which contains KeyMethod with the full function name
// (e.g. "(*Container).StartMonitoringService").
//
// Cost: roughly 235ns for runtime.Caller() lookup.
//
// By default, loggers from GetLogger() capture caller information on their
// own. Use WithoutAutoCaller() to turn this off.
func Caller() Attr {
	_, method := callerInfoAtSkip(1)
	return slog.String(KeyMethod, method)
}

// ResetCallerCache clears the caller info cache. Intended for test isolation.
func ResetCallerCache() {
	callerCache.Range(func(key, _ any) bool {
		callerCache.Delete(key)
		return true
	})
}

// New creates a Logger instance that does not depend on any global factory.
//
// Takes baseLogger (*slog.Logger) which provides the underlying logging output.
// Takes name (string) which identifies the tracer for OpenTelemetry.
//
// Returns Logger which is the configured logger ready for use.
func New(baseLogger *slog.Logger, name string) Logger {
	return NewLogger(
		baseLogger,
		otel.Tracer(name),
		context.Background(),
	)
}

// NewLogger creates a new Logger instance with the provided slog.Logger and
// OpenTelemetry tracer. Auto-caller is enabled by default; use
// WithoutAutoCaller to disable.
//
// Takes logger (*slog.Logger) which handles the underlying log output.
// Takes tracer (trace.Tracer) which provides OpenTelemetry tracing support.
// Takes ctx (...context.Context) which sets the logger context; if omitted,
// context.Background is used.
//
// Returns Logger which is the configured logger ready for use.
func NewLogger(logger *slog.Logger, tracer trace.Tracer, ctx ...context.Context) Logger {
	logCtx := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		logCtx = ctx[0]
	}
	return &slogLogger{
		logger:             logger,
		tracer:             tracer,
		ctx:                logCtx,
		hooks:              nil,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: newRuntimeStackTraceProvider(),
		useDynamicDefault:  false,
		autoCaller:         true,
	}
}

// callerAttrsAtSkip returns slog.Attr values for the caller at the given stack
// depth. Uses the loc library for fast, allocation-free caller capture, with
// caching for the attribute building step.
//
// Takes skip (int) which specifies how many stack frames to skip.
//
// Returns ctxAttr (Attr) which is the context attribute (e.g. "bootstrap").
// Returns methodAttr (Attr) which is the method attribute
// (e.g. "(*Container).Start").
func callerAttrsAtSkip(skip int) (ctxAttr, methodAttr Attr) {
	pc := caller.Caller(skip + 1)
	if pc == 0 {
		return slog.String(KeyContext, callerUnknown), slog.String(KeyMethod, callerUnknown)
	}

	if cached, ok := callerCache.Load(pc); ok {
		if entry, assertOK := cached.(callerCacheEntry); assertOK {
			return entry.ctxAttr, entry.methodAttr
		}
	}

	name, _, _ := pc.NameFileLine()
	pkg, method := extractPackageAndMethod(name)
	ctxAttr = slog.String(KeyContext, pkg)
	methodAttr = slog.String(KeyMethod, method)
	callerCache.Store(pc, callerCacheEntry{ctxAttr: ctxAttr, methodAttr: methodAttr})
	return ctxAttr, methodAttr
}

// callerInfoAtSkip gets the package and method name at a given stack depth.
//
// Uses the loc library for fast caller capture without memory allocation.
// Results are cached to avoid repeated string parsing.
//
// Takes skip (int) which specifies how many stack frames to skip.
//
// Returns pkg (string) which is the package name from the full function path.
// Returns method (string) which is the method or function name.
func callerInfoAtSkip(skip int) (pkg, method string) {
	pc := caller.Caller(skip + 1)
	if pc == 0 {
		return callerUnknown, callerUnknown
	}

	if cached, ok := callerCache.Load(pc); ok {
		if entry, assertOK := cached.(callerCacheEntry); assertOK {
			return entry.ctxAttr.Value.String(), entry.methodAttr.Value.String()
		}
	}

	name, _, _ := pc.NameFileLine()
	pkg, method = extractPackageAndMethod(name)
	callerCache.Store(pc, callerCacheEntry{
		ctxAttr:    slog.String(KeyContext, pkg),
		methodAttr: slog.String(KeyMethod, method),
	})
	return pkg, method
}

// extractPackageAndMethod splits a fully qualified function name into its
// package and method parts. This is the slow path that only runs once per
// unique call site.
//
// Takes name (string) which is the fully qualified function name.
//
// Returns pkg (string) which is the package name.
// Returns method (string) which is the method or function name.
func extractPackageAndMethod(name string) (pkg, method string) {
	if name == "" {
		return "<unknown>", "<unknown>"
	}

	shortName := name
	if index := strings.LastIndex(name, "/"); index >= 0 {
		shortName = name[index+1:]
	}

	if pkg, method, found := strings.Cut(shortName, "."); found {
		return pkg, method
	}

	return shortName, shortName
}

// newLoggerWithStackTraceProvider creates a Logger with a custom stack trace
// provider. This is primarily for testing, allowing injection of mock stack
// trace providers.
//
// Takes logger (*slog.Logger) which provides the underlying structured logger.
// Takes tracer (trace.Tracer) which provides distributed tracing support.
// Takes provider (stackTraceProvider) which supplies stack trace information.
// Takes ctx (...context.Context) which optionally sets the logger context.
//
// Returns Logger which is the configured logger ready for use.
func newLoggerWithStackTraceProvider(logger *slog.Logger, tracer trace.Tracer, provider stackTraceProvider, ctx ...context.Context) Logger {
	logCtx := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		logCtx = ctx[0]
	}
	return &slogLogger{
		logger:             logger,
		tracer:             tracer,
		ctx:                logCtx,
		hooks:              nil,
		hooksMutex:         sync.RWMutex{},
		stackTraceProvider: provider,
		autoCaller:         true,
		useDynamicDefault:  false,
	}
}

// otelAttrsFromSlog converts slog attributes to OpenTelemetry key-value pairs.
//
// Takes attrs ([]slog.Attr) which contains the slog attributes to convert.
//
// Returns []attribute.KeyValue which contains the converted attributes, or
// nil when attrs is empty.
func otelAttrsFromSlog(attrs []slog.Attr) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	kvsPtr, ok := otelAttrPool.Get().(*[]attribute.KeyValue)
	if !ok {
		kvsPtr = new(make([]attribute.KeyValue, 0, len(attrs)))
	}
	kvs := (*kvsPtr)[:0]
	for _, a := range attrs {
		addAttrRecursive(&kvs, "", a)
	}
	result := make([]attribute.KeyValue, len(kvs))
	copy(result, kvs)
	*kvsPtr = kvs
	otelAttrPool.Put(kvsPtr)
	return result
}

// addAttrRecursive converts a slog attribute to OpenTelemetry key-value pairs.
//
// Takes kvs (*[]attribute.KeyValue) which collects the converted attributes.
// Takes prefix (string) which is added before the attribute key with a dot.
// Takes a (slog.Attr) which is the slog attribute to convert.
func addAttrRecursive(kvs *[]attribute.KeyValue, prefix string, a slog.Attr) {
	key := a.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	switch a.Value.Kind() {
	case slog.KindGroup:
		addGroupAttr(kvs, key, a)
	case slog.KindInt64:
		*kvs = append(*kvs, attribute.Int64(key, a.Value.Int64()))
	case slog.KindUint64:
		u := a.Value.Uint64()
		if u > math.MaxInt64 {
			*kvs = append(*kvs, attribute.String(key, fmt.Sprintf("%d", u)))
		} else {
			*kvs = append(*kvs, attribute.Int64(key, int64(u))) //nolint:gosec // bounds checked above
		}
	case slog.KindFloat64:
		*kvs = append(*kvs, attribute.Float64(key, a.Value.Float64()))
	case slog.KindBool:
		*kvs = append(*kvs, attribute.Bool(key, a.Value.Bool()))
	case slog.KindDuration:
		*kvs = append(*kvs, attribute.String(key, a.Value.Duration().String()))
	case slog.KindTime:
		*kvs = append(*kvs, attribute.String(key, a.Value.Time().Format(time.RFC3339Nano)))
	case slog.KindAny:
		addAnyAttr(kvs, key, a.Value.Any())
	default:
		*kvs = append(*kvs, attribute.String(key, a.Value.String()))
	}
}

// addGroupAttr processes a group attribute by adding each nested attribute
// to the output slice.
//
// Takes kvs (*[]attribute.KeyValue) which receives the converted attributes.
// Takes key (string) which specifies the group prefix for nested attributes.
// Takes a (slog.Attr) which contains the group of attributes to process.
func addGroupAttr(kvs *[]attribute.KeyValue, key string, a slog.Attr) {
	for _, groupAttr := range a.Value.Group() {
		addAttrRecursive(kvs, key, groupAttr)
	}
}

// addAnyAttr converts a value of any type to an OpenTelemetry attribute.
// It handles common types such as stdjson.Marshaler, encoding.TextMarshaler,
// error, fmt.Stringer, and string slices.
//
// Takes kvs (*[]attribute.KeyValue) which receives the converted attribute.
// Takes key (string) which specifies the attribute name.
// Takes anyVal (any) which is the value to convert based on its type.
func addAnyAttr(kvs *[]attribute.KeyValue, key string, anyVal any) {
	switch v := anyVal.(type) {
	case stdjson.Marshaler:
		addJSONMarshalerAttr(kvs, key, v)
	case encoding.TextMarshaler:
		addTextMarshalerAttr(kvs, key, v)
	case error:
		*kvs = append(*kvs, attribute.String(key, v.Error()))
	case fmt.Stringer:
		*kvs = append(*kvs, attribute.String(key, v.String()))
	case []string:
		*kvs = append(*kvs, attribute.StringSlice(key, v))
	default:
		addFallbackAttr(kvs, key, anyVal)
	}
}

// addJSONMarshalerAttr converts a stdjson.Marshaler value to JSON and adds it as
// a string attribute. If marshalling fails, it uses fmt.Sprintf instead.
//
// Takes kvs (*[]attribute.KeyValue) which receives the new attribute.
// Takes key (string) which specifies the attribute name.
// Takes v (stdjson.Marshaler) which provides the value to convert.
func addJSONMarshalerAttr(kvs *[]attribute.KeyValue, key string, v stdjson.Marshaler) {
	if b, err := v.MarshalJSON(); err == nil {
		*kvs = append(*kvs, attribute.String(key, string(b)))
	} else {
		*kvs = append(*kvs, attribute.String(key, fmt.Sprintf("%+v", v)))
	}
}

// addTextMarshalerAttr adds an attribute for encoding.TextMarshaler types.
// If text marshalling fails, it falls back to JSON marshalling, then to
// fmt.Sprintf.
//
// Takes kvs (*[]attribute.KeyValue) which receives the new attribute.
// Takes key (string) which specifies the attribute name.
// Takes v (encoding.TextMarshaler) which provides the value to marshal.
func addTextMarshalerAttr(kvs *[]attribute.KeyValue, key string, v encoding.TextMarshaler) {
	if b, err := v.MarshalText(); err == nil {
		*kvs = append(*kvs, attribute.String(key, string(b)))
	} else {
		if s, err := json.MarshalString(v); err == nil {
			*kvs = append(*kvs, attribute.String(key, s))
		} else {
			*kvs = append(*kvs, attribute.String(key, fmt.Sprintf("%+v", v)))
		}
	}
}

// addFallbackAttr converts a value to JSON and adds it as a string attribute.
// If JSON encoding fails, it uses fmt.Sprintf as a fallback.
//
// Takes kvs (*[]attribute.KeyValue) which receives the new attribute.
// Takes key (string) which is the attribute name.
// Takes anyVal (any) which is the value to convert.
func addFallbackAttr(kvs *[]attribute.KeyValue, key string, anyVal any) {
	if s, err := json.MarshalString(anyVal); err == nil {
		*kvs = append(*kvs, attribute.String(key, s))
	} else {
		*kvs = append(*kvs, attribute.String(key, fmt.Sprintf("%+v", anyVal)))
	}
}
