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

package logger_state

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	logger_adapters_handlers "piko.sh/piko/internal/logger/logger_adapters/driver_handlers"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// globalState holds the package-level shared state including logging handlers
	// and shutdown hooks. Access is protected by mux and the sync.Once fields for
	// thread-safe initialisation.
	globalState struct {
		destinationHandlers    []slog.Handler
		wrapperFactories       []func(slog.Handler) slog.Handler
		activeClosers          []io.Closer
		shutdownHooks          []func(context.Context) error
		sentryInitOnce         sync.Once
		mux                    sync.Mutex
		isDefaultHandlerActive bool
	}

	// getSharedHTTPClient returns a lazily initialised HTTP client configured
	// with optimised connection pooling and timeouts.
	getSharedHTTPClient = sync.OnceValue(func() *http.Client {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.MaxIdleConns = 100
		transport.MaxIdleConnsPerHost = 10
		transport.IdleConnTimeout = 90 * time.Second
		transport.TLSHandshakeTimeout = 10 * time.Second
		transport.ExpectContinueTimeout = 1 * time.Second
		transport.DialContext = (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext
		return &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		}
	})
)

// GetSharedHTTPClient returns a shared HTTP client for use across the logger
// package.
//
// Returns *http.Client which is a singleton client configured with optimised
// connection pooling and timeouts.
func GetSharedHTTPClient() *http.Client {
	return getSharedHTTPClient()
}

// ClearAllHandlers removes all registered log handlers and resets the logging
// state. It shuts down any active closers and clears all handler and wrapper
// registrations.
//
// Safe for concurrent use by multiple goroutines.
func ClearAllHandlers() {
	globalState.mux.Lock()
	defer globalState.mux.Unlock()

	if err := shutdownCurrentState(context.Background()); err != nil {
		log.Warn("Failed to shutdown current state during ClearAllHandlers", logger_domain.Error(err))
	}
	globalState.destinationHandlers = []slog.Handler{}
	globalState.activeClosers = []io.Closer{}
	globalState.wrapperFactories = []func(slog.Handler) slog.Handler{}
	globalState.isDefaultHandlerActive = false
	applyHandlerSet()
}

// AddHandler registers a new slog.Handler to receive log records. Adding a
// handler replaces any default handlers that were active.
//
// Takes handler (slog.Handler) which receives log records.
// Takes closer (io.Closer) which is closed during logger shutdown, or nil.
//
// Safe for concurrent use. Uses a mutex to protect global state.
func AddHandler(handler slog.Handler, closer io.Closer) {
	globalState.mux.Lock()
	defer globalState.mux.Unlock()

	if globalState.isDefaultHandlerActive {
		if err := shutdownCurrentState(context.Background()); err != nil {
			log.Warn("Failed to shutdown current state during AddHandler", logger_domain.Error(err))
		}
		globalState.destinationHandlers = []slog.Handler{}
		globalState.activeClosers = []io.Closer{}
		globalState.isDefaultHandlerActive = false
	}

	globalState.destinationHandlers = append(globalState.destinationHandlers, handler)
	if closer != nil {
		globalState.activeClosers = append(globalState.activeClosers, closer)
	}

	applyHandlerSet()
}

// AddWrapper registers a handler wrapper factory that will be applied to the
// composed handler. Wrappers are applied in the order they are registered,
// allowing for chained middleware such as notification handlers or filtering
// handlers.
//
// Takes wrapperFactory (func(slog.Handler) slog.Handler) which creates a
// wrapped handler from the given base handler.
//
// Safe for concurrent use by multiple goroutines.
func AddWrapper(wrapperFactory func(slog.Handler) slog.Handler) {
	globalState.mux.Lock()
	defer globalState.mux.Unlock()

	globalState.wrapperFactories = append(globalState.wrapperFactories, wrapperFactory)
	applyHandlerSet()
}

// GetSentryInitOnce returns the sync.Once that guards single initialisation
// of Sentry. This is used by the Sentry integration to prevent multiple
// initialisations.
//
// Returns *sync.Once which guards Sentry initialisation across all callers.
func GetSentryInitOnce() *sync.Once {
	return &globalState.sentryInitOnce
}

// AddShutdownHook registers a function to be called during logger shutdown.
// Hooks are executed in the order they were registered when GetShutdownFunc
// is called.
//
// Takes hook (func(context.Context) error) which is the function to call
// during shutdown.
//
// Safe for concurrent use by multiple goroutines.
func AddShutdownHook(hook func(context.Context) error) {
	globalState.mux.Lock()
	defer globalState.mux.Unlock()
	globalState.shutdownHooks = append(globalState.shutdownHooks, hook)
}

// GetShutdownFunc returns a function that performs graceful shutdown of the
// logging system. It closes all active closers and executes all registered
// shutdown hooks.
//
// Returns func(context.Context) error which performs the shutdown when called.
//
// Safe for concurrent use. The returned function acquires a mutex lock during
// shutdown.
func GetShutdownFunc() func(context.Context) error {
	return func(ctx context.Context) error {
		globalState.mux.Lock()
		defer globalState.mux.Unlock()
		return shutdownCurrentState(ctx)
	}
}

// ResetState resets the logging system to its default configuration.
// It shuts down the current state and reinitialises with a default pretty
// handler on stdout.
//
// Safe for concurrent use by multiple goroutines.
func ResetState() {
	globalState.mux.Lock()
	defer globalState.mux.Unlock()
	doResetState()
}

// HasExplicitHandlers reports whether handlers have been explicitly added
// via AddHandler or AddPrettyOutput, rather than just the default handler.
//
// Returns bool which is true if explicit handlers are configured.
//
// Safe for concurrent use.
func HasExplicitHandlers() bool {
	globalState.mux.Lock()
	defer globalState.mux.Unlock()
	return len(globalState.destinationHandlers) > 0 && !globalState.isDefaultHandlerActive
}

// EnableNotificationPort is a helper for notification facade packages
// (Discord, Slack, PagerDuty, etc.). It registers a notification port that
// wraps the current global logger.
//
// Takes name (string) which identifies this notification handler.
// Takes typeName (string) which specifies the notification service type.
// Takes notificationPort (logger_domain.NotificationPort) which sends
// notifications.
// Takes minLevel (slog.Level) which sets the minimum log level for
// notifications.
func EnableNotificationPort(name, typeName string, notificationPort logger_domain.NotificationPort, minLevel slog.Level) {
	log := logger_domain.GetLogger("logger")
	log.Info("Enabling notifications",
		slog.String("name", name),
		slog.String("type", typeName),
		slog.String("minLevel", minLevel.String()),
	)

	wrapperFactory := func(nextHandler slog.Handler) slog.Handler {
		return logger_domain.NewNotificationHandler(nextHandler, notificationPort, minLevel)
	}

	AddWrapper(wrapperFactory)

	log.Info("Notifications enabled",
		slog.String("type", typeName))
}

// applyHandlerSet configures and activates the composed handler chain.
func applyHandlerSet() {
	var composedHandler slog.Handler
	if len(globalState.destinationHandlers) == 0 {
		composedHandler = slog.NewJSONHandler(io.Discard, nil)
	} else if len(globalState.destinationHandlers) == 1 {
		composedHandler = globalState.destinationHandlers[0]
	} else {
		composedHandler = slog.NewMultiHandler(globalState.destinationHandlers...)
	}

	for _, wrapperFactory := range globalState.wrapperFactories {
		composedHandler = wrapperFactory(composedHandler)
	}

	composedHandler = logger_domain.NewRequestContextHandler(composedHandler)

	finalHandler := logger_domain.NewOTelSlogHandler(composedHandler)

	newLogger := slog.New(finalHandler)
	slog.SetDefault(newLogger)
	logger_domain.InitDefaultFactory(newLogger)
}

// shutdownCurrentState closes all active closers and runs shutdown hooks. The
// supplied ctx is honoured between iterations so callers can bound the total
// shutdown time; once ctx is cancelled the remaining closers and hooks are
// skipped and the cancellation cause is included in the returned error.
//
// Returns error when any closer or shutdown hook fails. Multiple errors are
// joined via errors.Join so callers can use errors.Is or errors.As.
func shutdownCurrentState(ctx context.Context) error {
	var allErrors []error
	for _, closer := range globalState.activeClosers {
		if err := ctx.Err(); err != nil {
			allErrors = append(allErrors, fmt.Errorf("logger shutdown cancelled before closing remaining resources: %w", err))
			return joinShutdownErrors(allErrors)
		}
		if err := closer.Close(); err != nil {
			allErrors = append(allErrors, err)
		}
	}
	for _, hook := range globalState.shutdownHooks {
		if err := ctx.Err(); err != nil {
			allErrors = append(allErrors, fmt.Errorf("logger shutdown cancelled before running remaining hooks: %w", err))
			return joinShutdownErrors(allErrors)
		}
		if err := hook(ctx); err != nil {
			allErrors = append(allErrors, err)
		}
	}
	return joinShutdownErrors(allErrors)
}

// joinShutdownErrors collapses a slice of errors into a single wrapped error
// suitable for callers using errors.Is or errors.As. Returns nil for an empty
// slice.
//
// Takes allErrors ([]error) which holds the errors collected during shutdown.
//
// Returns error which wraps the joined errors, or nil when allErrors is
// empty.
func joinShutdownErrors(allErrors []error) error {
	if len(allErrors) == 0 {
		return nil
	}
	return fmt.Errorf("errors during logger shutdown: %w", errors.Join(allErrors...))
}

// doResetState resets the global logger state to its default configuration.
func doResetState() {
	if err := shutdownCurrentState(context.Background()); err != nil {
		log.Warn("Failed to shutdown current state during doResetState", logger_domain.Error(err))
	}

	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.LevelInfo)

	handler := logger_adapters_handlers.NewPrettyHandler(os.Stdout, &logger_adapters_handlers.Options{
		Level: levelVar,
	})

	globalState.destinationHandlers = []slog.Handler{handler}
	globalState.wrapperFactories = []func(slog.Handler) slog.Handler{}
	globalState.activeClosers = []io.Closer{}
	globalState.shutdownHooks = []func(context.Context) error{}
	globalState.isDefaultHandlerActive = true
	globalState.sentryInitOnce = sync.Once{}

	applyHandlerSet()
}

func init() {
	if os.Getenv("PIKO_DISABLE_CONSOLE_LOG") == "true" {
		return
	}
	doResetState()
}
