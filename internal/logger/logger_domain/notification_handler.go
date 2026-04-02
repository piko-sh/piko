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
	"fmt"
	"hash"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultDebounceDuration is the default wait time before sending grouped
	// error notifications.
	defaultDebounceDuration = 10 * time.Second

	// defaultTimeoutDuration is the time limit for sending error notifications.
	defaultTimeoutDuration = 15 * time.Second
)

// hasherPool provides reusable xxhash instances for speed and to make it
// clear this is not for cryptographic purposes.
var hasherPool = sync.Pool{
	New: func() any {
		return xxhash.New()
	},
}

// notificationState tracks pending notifications and handles debouncing.
type notificationState struct {
	// notificationPort sends error notifications to an external service.
	notificationPort NotificationPort

	// debounceTimer delays sending grouped notifications; nil when not active.
	debounceTimer clock.Timer

	// clock provides time functions for scheduling and testing.
	clock clock.Clock

	// groupedErrors maps error keys to their grouped error data for batching.
	groupedErrors map[string]*GroupedError

	// minLevel is the lowest log level that triggers a notification.
	minLevel slog.Level

	// debounceDur is the delay before sending grouped messages; default is 10
	// seconds.
	debounceDur time.Duration

	// mu guards access to the notification state fields.
	mu sync.Mutex
}

// NotificationHandler is a slog.Handler that batches and sends log
// notifications to external services. It groups identical errors by their
// message and source location, debounces notifications to prevent flooding, and
// provides graceful shutdown to ensure all pending notifications are sent.
type NotificationHandler struct {
	slog.Handler

	// state holds shared mutable state that is copied across handler instances.
	state *notificationState
}

// NewNotificationHandler creates a notification handler that wraps an existing
// handler.
//
// It sends notifications via the provided notification port for all log entries
// at or above minLevel. The handler is automatically registered for graceful
// shutdown with the DefaultLifecycleManager. Uses the real system clock.
//
// Takes next (slog.Handler) which is the underlying handler to wrap.
// Takes notificationPort (NotificationPort) which sends the notifications.
// Takes minLevel (slog.Level) which sets the minimum level for notifications.
//
// Returns *NotificationHandler which is ready for use as an slog.Handler.
func NewNotificationHandler(next slog.Handler, notificationPort NotificationPort, minLevel slog.Level) *NotificationHandler {
	return newNotificationHandlerWithOptions(next, notificationPort, minLevel, clock.RealClock(), defaultLifecycleManager)
}

// Handle processes a log record and sends notifications for records at or
// above the set level. The record is always passed to the next handler in
// the chain.
//
// Takes r (slog.Record) which contains the log entry to process.
//
// Returns error when the underlying handler fails.
//
//nolint:gocritic // slog.Handler requires value receiver
func (h *NotificationHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= h.state.minLevel {
		h.groupAndScheduleSend(&r)
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs creates a new handler with additional attributes, preserving the
// notification state.
//
// Takes attrs ([]slog.Attr) which specifies the attributes to add.
//
// Returns slog.Handler which is the new handler with the added attributes.
func (h *NotificationHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &NotificationHandler{
		Handler: h.Handler.WithAttrs(attrs),
		state:   h.state,
	}
}

// WithGroup creates a new handler with a group name, preserving the
// notification state.
//
// Takes name (string) which specifies the group name for the new handler.
//
// Returns slog.Handler which is a new handler with the group applied.
func (h *NotificationHandler) WithGroup(name string) slog.Handler {
	return &NotificationHandler{
		Handler: h.Handler.WithGroup(name),
		state:   h.state,
	}
}

// Shutdown stops the debounce timer and flushes any pending notifications
// immediately. This is called automatically during application shutdown via
// the registered shutdown hook.
//
// Safe for concurrent use. Acquires the state mutex to stop the timer before
// flushing.
func (h *NotificationHandler) Shutdown() {
	h.state.mu.Lock()
	if h.state.debounceTimer != nil {
		h.state.debounceTimer.Stop()
		h.state.debounceTimer = nil
	}
	h.state.mu.Unlock()

	h.flushGroupedMessages()
}

// SetDebounceDuration sets how long to wait before sending grouped
// notifications. The default is 10 seconds.
//
// Takes d (time.Duration) which specifies the wait time.
func (h *NotificationHandler) SetDebounceDuration(d time.Duration) {
	h.state.debounceDur = d
}

// GetPendingErrorCount returns the number of unique grouped errors that are
// waiting to be sent. This method is mainly for test checks.
//
// Returns int which is the count of pending grouped errors.
//
// Safe for concurrent use.
func (h *NotificationHandler) GetPendingErrorCount() int {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	return len(h.state.groupedErrors)
}

// HasPendingBatch returns true if there are any grouped errors waiting to be
// sent. This method is primarily for test verification.
//
// Returns bool which indicates whether pending errors exist in the batch.
//
// Safe for concurrent use.
func (h *NotificationHandler) HasPendingBatch() bool {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	return len(h.state.groupedErrors) > 0
}

// GetPendingErrors returns a copy of the current grouped errors map.
// This method is primarily for test verification and debugging.
//
// Returns map[string]*GroupedError which is a shallow copy of the pending
// errors. Modifications to the returned map will not affect the handler's
// state.
//
// Safe for concurrent use.
func (h *NotificationHandler) GetPendingErrors() map[string]*GroupedError {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	result := make(map[string]*GroupedError, len(h.state.groupedErrors))
	maps.Copy(result, h.state.groupedErrors)
	return result
}

// GetDebounceDuration returns the current debounce duration.
// This method is primarily for test verification.
//
// Returns time.Duration which is the current debounce interval.
//
// Safe for concurrent use.
func (h *NotificationHandler) GetDebounceDuration() time.Duration {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	return h.state.debounceDur
}

// GetMinLevel returns the minimum log level for notifications.
// This method is primarily for test verification.
//
// Returns slog.Level which is the current minimum notification level.
//
// Safe for concurrent use.
func (h *NotificationHandler) GetMinLevel() slog.Level {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()
	return h.state.minLevel
}

// groupAndScheduleSend adds a log record to the grouped errors and sets up a
// delayed send.
//
// Takes r (*slog.Record) which is the log record to group and schedule.
//
// Safe for concurrent use. Uses mutex locking to protect shared state.
func (h *NotificationHandler) groupAndScheduleSend(r *slog.Record) {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	key := generateKey(r)
	now := h.state.clock.Now().UTC()

	if existing, ok := h.state.groupedErrors[key]; ok {
		existing.Count++
		existing.LastSeen = now
	} else {
		var file string
		var line int
		if r.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{r.PC})
			f, _ := fs.Next()
			file = filepath.Base(f.File)
			line = f.Line
		}

		h.state.groupedErrors[key] = &GroupedError{
			LogRecord:  *r,
			FirstSeen:  now,
			LastSeen:   now,
			SourceFile: file,
			SourceLine: line,
			Count:      1,
		}
	}

	if h.state.debounceTimer == nil {
		h.state.debounceTimer = h.state.clock.AfterFunc(h.state.debounceDur, h.flushGroupedMessages)
	}
}

// flushGroupedMessages sends all pending grouped errors as a batch.
//
// Safe for concurrent use; holds the state mutex during the operation.
func (h *NotificationHandler) flushGroupedMessages() {
	h.state.mu.Lock()
	if len(h.state.groupedErrors) == 0 {
		h.state.debounceTimer = nil
		h.state.mu.Unlock()
		return
	}

	errorsToSend := h.state.groupedErrors
	h.state.groupedErrors = make(map[string]*GroupedError)
	h.state.debounceTimer = nil
	h.state.mu.Unlock()

	h.sendBatch(errorsToSend)
}

// sendBatch sends a batch of grouped errors using the notification port.
//
// Takes batch (map[string]*GroupedError) which contains the errors to send.
func (h *NotificationHandler) sendBatch(batch map[string]*GroupedError) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), defaultTimeoutDuration,
		fmt.Errorf("notification batch send exceeded %s timeout", defaultTimeoutDuration))
	defer cancel()

	err := h.state.notificationPort.SendGroupedErrors(ctx, batch)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "ERROR(notification_handler): failed to send notifications: %v\n", err)
	}
}

// newNotificationHandlerWithClock creates a notification handler with a custom
// clock.
//
// This function is mainly for testing. It lets you pass in a mock clock to
// control time-based behaviour. The handler is added to the
// defaultLifecycleManager for shutdown.
//
// Takes next (slog.Handler) which is the handler to wrap.
// Takes notificationPort (NotificationPort) which sends notifications.
// Takes minLevel (slog.Level) which sets the minimum level for notifications.
// Takes clk (clock.Clock) which provides time and can be mocked in tests.
//
// Returns *NotificationHandler which is ready to use as a logging handler.
func newNotificationHandlerWithClock(next slog.Handler, notificationPort NotificationPort, minLevel slog.Level, clk clock.Clock) *NotificationHandler { //nolint:unused // exported via export_test.go
	return newNotificationHandlerWithOptions(next, notificationPort, minLevel, clk, defaultLifecycleManager)
}

// newNotificationHandlerWithOptions creates a notification handler with full
// control over all settings.
//
// This constructor allows a custom clock and lifecycle manager to be passed in
// for testing. If lifecycle is nil, no shutdown hook will be registered.
//
// Takes next (slog.Handler) which is the handler to wrap.
// Takes notificationPort (NotificationPort) which sends the notifications.
// Takes minLevel (slog.Level) which sets the lowest level for notifications.
// Takes clk (clock.Clock) which provides time functions.
// Takes lifecycle (*lifecycleManager) which manages shutdown hooks.
//
// Returns *NotificationHandler which is the configured handler ready for use.
func newNotificationHandlerWithOptions(next slog.Handler, notificationPort NotificationPort, minLevel slog.Level, clk clock.Clock, lifecycle *lifecycleManager) *NotificationHandler {
	handler := &NotificationHandler{
		Handler: next,
		state: &notificationState{
			notificationPort: notificationPort,
			minLevel:         minLevel,
			groupedErrors:    make(map[string]*GroupedError),
			debounceDur:      defaultDebounceDuration,
			debounceTimer:    nil,
			clock:            clk,
			mu:               sync.Mutex{},
		},
	}

	if lifecycle != nil {
		lifecycle.RegisterShutdownHook(handler.Shutdown)
	}

	return handler
}

// generateKey creates a unique hash key from a log record's message and source
// location.
//
// Takes r (*slog.Record) which provides the log message and program counter.
//
// Returns string which is a hex hash that identifies the record.
func generateKey(r *slog.Record) string {
	poolItem := hasherPool.Get()
	hasher, ok := poolItem.(hash.Hash)
	if !ok {
		hasher = xxhash.New()
	}
	defer func() {
		hasher.Reset()
		hasherPool.Put(hasher)
	}()

	_, _ = hasher.Write([]byte(r.Message))

	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		_, _ = hasher.Write([]byte(f.File))
		_, _ = hasher.Write([]byte(strconv.Itoa(f.Line)))
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
