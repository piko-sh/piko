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

package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	// logKeyFunctionName is the attribute key used for logging the name of cleanup
	// functions.
	logKeyFunctionName = "functionName"

	// logValueComponent is the component name for shutdown log entries.
	logValueComponent = "shutdown"

	// DefaultTimeout is the default timeout for shutdown cleanup tasks.
	// This gives enough time for apps with several services (database links,
	// caches, OTEL exporters) to shut down in a controlled way.
	DefaultTimeout = 30 * time.Second

	// MinFunctionTimeout is the shortest timeout given to each cleanup function.
	// Each function gets at least this much time, even when total time left is
	// less.
	MinFunctionTimeout = 500 * time.Millisecond
)

// CleanupFunc represents a function that runs during shutdown. It receives a
// context that may have a timeout and returns an error on failure.
type CleanupFunc func(ctx context.Context) error

// namedCleanup pairs a cleanup function with its name for ordered shutdown.
type namedCleanup struct {
	// cleanupFunction is the cleanup function to run.
	cleanupFunction CleanupFunc

	// name is the identifier used for logging and tracing.
	name string
}

// ManagerOption configures a shutdown Manager.
type ManagerOption func(*Manager)

// Manager coordinates shutdown cleanup functions. Use NewManager to create
// instances for testing, or use the package-level functions which use a
// shared default instance.
type Manager struct {
	// clock provides time operations for timeout calculations.
	clock clock.Clock

	// cleanupFuncs holds the functions to run during shutdown.
	cleanupFuncs []namedCleanup

	// funcsLock protects cleanupFuncs during registration and cleanup.
	// Uses RWMutex to allow reading at the same time while protecting writes.
	funcsLock sync.RWMutex

	// cleanupInProgress indicates whether cleanup is currently running. Uses
	// atomic.Bool for lock-free reads, preventing new registrations during
	// cleanup.
	cleanupInProgress atomic.Bool
}

// defaultManager is the global instance used by package-level functions
// for a simpler developer experience.
var defaultManager = NewManager()

// NewManager creates a new shutdown manager with isolated state. This is
// useful for testing where you need independent instances that do not
// interfere with each other.
//
// Takes opts (...ManagerOption) which configures optional behaviour.
//
// Returns *Manager which is the newly created shutdown manager.
func NewManager(opts ...ManagerOption) *Manager {
	m := &Manager{
		cleanupFuncs:      make([]namedCleanup, 0),
		funcsLock:         sync.RWMutex{},
		cleanupInProgress: atomic.Bool{},
		clock:             clock.RealClock(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Register adds a cleanup function to be executed during shutdown.
//
// Functions are executed in reverse registration order (LIFO) when shutdown is
// triggered. The name is used for logging and tracing purposes to identify
// the cleanup function.
//
// If cleanup is already in progress, this function logs a warning and returns
// without registering. This prevents potential deadlocks from cleanup functions
// attempting to register new cleanup handlers.
//
// Takes ctx (context.Context) which carries trace and logging context.
// Takes name (string) which identifies the cleanup function in logs and traces.
// Takes cleanupFunction (CleanupFunc) which is the function to call
// during shutdown.
//
// Safe for concurrent use. Uses a mutex to protect the cleanup
// function list.
func (m *Manager) Register(ctx context.Context, name string, cleanupFunction CleanupFunc) {
	ctx, span, l := log.Span(ctx, "shutdown.Register",
		logger_domain.String(logger_domain.FieldStrComponent, logValueComponent),
		logger_domain.String(logKeyFunctionName, name),
	)
	defer span.End()

	if m.cleanupInProgress.Load() {
		l.Warn("Cannot register cleanup function during shutdown",
			logger_domain.String("name", name))
		return
	}

	m.funcsLock.Lock()
	defer m.funcsLock.Unlock()

	if m.cleanupInProgress.Load() {
		l.Warn("Cannot register cleanup function during shutdown",
			logger_domain.String("name", name))
		return
	}

	m.cleanupFuncs = append(m.cleanupFuncs, namedCleanup{name: name, cleanupFunction: cleanupFunction})
	CleanupFunctionCount.Add(ctx, 1)

	l.Internal("Registered cleanup function", logger_domain.String("name", name))
}

// ListenAndShutdown waits for a shutdown signal (SIGINT or SIGTERM), then
// runs all registered cleanup functions within the given timeout. This method
// calls os.Exit(0) after cleanup finishes.
//
// Takes totalTimeout (time.Duration) which sets the maximum time for all
// cleanup tasks to complete.
func (m *Manager) ListenAndShutdown(totalTimeout time.Duration) {
	ctx := context.Background()
	ctx, span, l := log.Span(ctx, "shutdown.ListenAndShutdown",
		logger_domain.String(logger_domain.FieldStrComponent, logValueComponent),
		logger_domain.Duration("totalTimeout", totalTimeout),
	)

	l.Internal("Setting up shutdown signal handlers")

	signalCtx, cancelSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	<-signalCtx.Done()

	cause := context.Cause(signalCtx)
	causeMessage := "unknown"
	if cause != nil {
		causeMessage = cause.Error()
	}
	l.Notice("Shutdown signal received",
		logger_domain.String("cause", causeMessage),
	)
	ShutdownSignalCount.Add(ctx, 1)

	m.Cleanup(ctx, totalTimeout)

	l.Notice("Exiting application")
	span.End()

	_ = os.Stdout.Sync()
	_ = os.Stderr.Sync()

	cancelSignal()
	os.Exit(0)
}

// ListenAndShutdownWithSignal waits for a signal and then runs cleanup.
//
// This is a testable version of ListenAndShutdown. It accepts a signal channel
// so that tests can trigger shutdown without using OS signals. Unlike
// ListenAndShutdown, this method does not call os.Exit. The caller must handle
// the exit behaviour.
//
// Takes totalTimeout (time.Duration) which sets the maximum time for cleanup.
// Takes sigChan (<-chan os.Signal) which is the channel to listen for shutdown
// signals.
func (m *Manager) ListenAndShutdownWithSignal(ctx context.Context, totalTimeout time.Duration, sigChan <-chan os.Signal) {
	ctx, span, l := log.Span(ctx, "shutdown.ListenAndShutdownWithSignal",
		logger_domain.String(logger_domain.FieldStrComponent, logValueComponent),
		logger_domain.Duration("totalTimeout", totalTimeout),
	)
	defer span.End()

	l.Internal("Waiting for shutdown signal")
	<-sigChan

	l.Notice("Shutdown signal received")
	ShutdownSignalCount.Add(ctx, 1)

	m.Cleanup(ctx, totalTimeout)
	l.Internal("Cleanup completed")
}

// Cleanup executes all registered cleanup functions within the specified
// timeout.
//
// Functions are executed sequentially in reverse registration order (LIFO),
// similar to Go's defer semantics. This means resources registered early (such
// as the logger) are cleaned up last, allowing later-registered components to
// log during their cleanup.
//
// Each function receives a per-function timeout budget calculated by dividing
// the remaining time among remaining functions, with a minimum of
// MinFunctionTimeout. If a function panics, the panic is recovered and logged,
// and execution continues with the remaining functions.
//
// Takes totalTimeout (time.Duration) which specifies the maximum time allowed
// for all cleanup functions to complete.
//
// Safe for concurrent use. Acquires the internal lock briefly to copy the list
// of cleanup functions before executing them.
func (m *Manager) Cleanup(parentCtx context.Context, totalTimeout time.Duration) {
	ctx, span, l := log.Span(parentCtx, "shutdown.Cleanup",
		logger_domain.String(logger_domain.FieldStrComponent, logValueComponent),
		logger_domain.Duration("totalTimeout", totalTimeout),
	)
	defer span.End()

	m.cleanupInProgress.Store(true)
	defer m.cleanupInProgress.Store(false)

	m.funcsLock.Lock()
	functionCount := len(m.cleanupFuncs)
	funcs := make([]namedCleanup, functionCount)
	copy(funcs, m.cleanupFuncs)
	m.funcsLock.Unlock()

	l.Internal("Starting cleanup process", logger_domain.Int("functionCount", functionCount))

	startTime := m.clock.Now()
	defer func() {
		CleanupDuration.Record(ctx, float64(m.clock.Now().Sub(startTime).Milliseconds()))
		l.Internal("Cleanup process completed",
			logger_domain.Duration("duration", m.clock.Now().Sub(startTime)),
			logger_domain.Int("functionCount", functionCount))
	}()

	m.runCleanupFunctions(ctx, span, funcs, m.clock.Now().Add(totalTimeout))
}

// Reset clears all registered cleanup functions.
//
// This is mainly useful in tests to start each test case with a clean state.
//
// Safe for concurrent use.
func (m *Manager) Reset() {
	m.cleanupInProgress.Store(false)
	m.funcsLock.Lock()
	defer m.funcsLock.Unlock()
	m.cleanupFuncs = make([]namedCleanup, 0)
}

// Count returns the number of registered cleanup functions.
// Intended for testing and diagnostics.
//
// Returns int which is the current count of cleanup functions.
//
// Safe for concurrent use. Uses a read lock for better concurrency.
func (m *Manager) Count() int {
	m.funcsLock.RLock()
	defer m.funcsLock.RUnlock()
	return len(m.cleanupFuncs)
}

// IsCleanupInProgress returns whether cleanup is currently running.
// This can be used to check if new registrations will be rejected.
//
// Returns bool which is true if cleanup is in progress.
func (m *Manager) IsCleanupInProgress() bool {
	return m.cleanupInProgress.Load()
}

// runCleanupFunctions runs cleanup functions in LIFO order with deadline
// tracking.
//
// Takes span (trace.Span) which records tracing events for the cleanup process.
// Takes funcs ([]namedCleanup) which contains the cleanup functions to run.
// Takes deadline (time.Time) which specifies when cleanup must finish.
func (m *Manager) runCleanupFunctions(ctx context.Context, span trace.Span, funcs []namedCleanup, deadline time.Time) {
	ctx, l := logger_domain.From(ctx, log)
	for i := len(funcs) - 1; i >= 0; i-- {
		cleanup := funcs[i]
		remaining := i + 1

		if m.clock.Now().After(deadline) {
			l.Warn("Not enough time to run cleanup (deadline exceeded)",
				logger_domain.String(logKeyFunctionName, cleanup.name),
				logger_domain.Int("skippedCount", remaining))
			CleanupFunctionTimeoutCount.Add(ctx, int64(remaining))
			return
		}

		perFunctionTimeout := max(deadline.Sub(m.clock.Now())/time.Duration(remaining), MinFunctionTimeout)
		l.Trace("Running cleanup function",
			logger_domain.String(logKeyFunctionName, cleanup.name),
			logger_domain.Duration("timeout", perFunctionTimeout))

		err := m.executeCleanupFunction(ctx, cleanup, perFunctionTimeout)
		CleanupFunctionExecutedCount.Add(ctx, 1)

		if err != nil {
			l.ReportError(span, err, "Cleanup function failed",
				logger_domain.String(logKeyFunctionName, cleanup.name))
			CleanupFunctionErrorCount.Add(ctx, 1)
		} else {
			l.Trace("Cleanup function completed successfully",
				logger_domain.String(logKeyFunctionName, cleanup.name))
		}
	}

	l.Internal("All cleanup functions completed")
}

// executeCleanupFunction runs a single cleanup function with panic recovery.
// It wraps the function execution in a span and recovers from any panic,
// converting it to an error.
//
// Takes ctx (context.Context) which provides the parent context for
// the cleanup operation.
// Takes cleanup (namedCleanup) which is the cleanup function to execute.
// Takes timeout (time.Duration) which limits how long the function may
// run.
//
// Returns resultErr (error) when the cleanup function fails or panics.
func (*Manager) executeCleanupFunction(ctx context.Context, cleanup namedCleanup, timeout time.Duration) (resultErr error) {
	ctx, l := logger_domain.From(ctx, log)
	cleanupCtx, cancel := context.WithTimeoutCause(ctx, timeout,
		fmt.Errorf("cleanup function %q exceeded %s timeout", cleanup.name, timeout))
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			resultErr = fmt.Errorf("panic in cleanup function %q: %v", cleanup.name, r)
			CleanupFunctionPanicCount.Add(ctx, 1)
		}
	}()

	resultErr = l.RunInSpan(cleanupCtx, fmt.Sprintf("cleanup.%s", cleanup.name), func(spanCtx context.Context, _ logger_domain.Logger) error {
		return cleanup.cleanupFunction(spanCtx)
	}, logger_domain.String(logKeyFunctionName, cleanup.name))

	return resultErr
}

// WithClock sets the clock used for timeout calculations. If not provided,
// the real system clock is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns ManagerOption which configures the manager's clock.
func WithClock(c clock.Clock) ManagerOption {
	return func(m *Manager) {
		if c != nil {
			m.clock = c
		}
	}
}

// Register adds a cleanup function to the global default manager.
//
// This is a convenience function that delegates to defaultManager.Register.
// For testing, create a new Manager instance using NewManager instead.
//
// Takes ctx (context.Context) which carries trace and logging context.
// Takes name (string) which identifies the cleanup function for logging.
// Takes cleanupFunction (CleanupFunc) which is the function to call
// during shutdown.
func Register(ctx context.Context, name string, cleanupFunction CleanupFunc) {
	defaultManager.Register(ctx, name, cleanupFunction)
}

// ListenAndShutdown waits for a shutdown signal, then runs cleanup and calls
// os.Exit(0).
//
// This is a convenience function that calls defaultManager.ListenAndShutdown().
//
// Takes totalTimeout (time.Duration) which sets the maximum time allowed for
// cleanup before forcing exit.
func ListenAndShutdown(totalTimeout time.Duration) {
	defaultManager.ListenAndShutdown(totalTimeout)
}

// Cleanup executes all cleanup functions on the global default manager.
// This is a convenience function that delegates to defaultManager.Cleanup().
//
// Takes totalTimeout (time.Duration) which limits the total time for all
// cleanup operations.
func Cleanup(ctx context.Context, totalTimeout time.Duration) {
	defaultManager.Cleanup(ctx, totalTimeout)
}

// Reset clears all registered cleanup functions on the global default manager.
//
// This is a convenience function that delegates to defaultManager.Reset().
func Reset() {
	defaultManager.Reset()
}
