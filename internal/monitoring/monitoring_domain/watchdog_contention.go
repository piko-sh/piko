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

package monitoring_domain

import (
	"context"
	"errors"
	"runtime"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

// RunContentionDiagnostic enables block + mutex profiling for the
// configured window, captures both profiles, then disables them.
//
// The call blocks for the diagnostic window plus capture overhead and
// serialises concurrent invocations via contentionMu (TryLock). On
// success two profile files are written under the "block-<ts>.pb.gz" and
// "mutex-<ts>.pb.gz" prefixes with paired sidecar metadata.
//
// Returns error when the diagnostic cannot start (already running, in
// cooldown, watchdog stopped, no profiling controller). Individual
// capture failures are logged but do not propagate.
func (w *Watchdog) RunContentionDiagnostic(ctx context.Context) error {
	if !w.contentionMu.TryLock() {
		watchdogContentionDiagnosticErrorCount.Add(ctx, 1)
		return ErrContentionDiagnosticInProgress
	}
	defer w.contentionMu.Unlock()

	if err := w.checkContentionPreconditions(); err != nil {
		watchdogContentionDiagnosticErrorCount.Add(ctx, 1)
		return err
	}

	ctx, l := logger_domain.From(ctx, log)
	l.Notice("Starting contention diagnostic",
		logger_domain.String("window", w.config.ContentionDiagnosticWindowDuration.String()),
	)
	w.sendNotification(ctx, NewContentionDiagnosticEvent("started", w.config.ContentionDiagnosticWindowDuration))

	restore := w.enableContentionProfiling()
	defer restore()

	if err := w.waitContentionWindow(ctx); err != nil {
		watchdogContentionDiagnosticErrorCount.Add(ctx, 1)
		return err
	}

	w.captureAndStoreProfile(ctx, "block", captureContext{Rule: "contention_diagnostic"})
	w.captureAndStoreProfile(ctx, "mutex", captureContext{Rule: "contention_diagnostic"})

	w.mu.Lock()
	w.lastContentionDiagnosticAt = w.clock.Now()
	w.mu.Unlock()

	watchdogContentionDiagnosticCount.Add(ctx, 1)
	w.sendNotification(ctx, NewContentionDiagnosticEvent("completed", w.config.ContentionDiagnosticWindowDuration))

	l.Notice("Contention diagnostic completed")
	return nil
}

// checkContentionPreconditions verifies the watchdog is in a state where a
// diagnostic can run: not stopped, profiling controller wired, and the
// cooldown elapsed since the previous diagnostic.
//
// Returns error when any precondition fails; the value is one of
// ErrWatchdogStopped, ErrProfilingControllerNil, or
// ErrContentionDiagnosticCooldown.
//
// Safe for concurrent use; acquires the watchdog mutex briefly.
func (w *Watchdog) checkContentionPreconditions() error {
	w.mu.Lock()
	stopped := w.stopped
	controller := w.profilingController
	last := w.lastContentionDiagnosticAt
	cooldown := w.config.ContentionDiagnosticCooldown
	w.mu.Unlock()

	if stopped {
		return ErrWatchdogStopped
	}
	if controller == nil {
		return ErrProfilingControllerNil
	}

	if cooldown > 0 && !last.IsZero() && w.clock.Now().Sub(last) < cooldown {
		return ErrContentionDiagnosticCooldown
	}
	return nil
}

// enableContentionProfiling sets the runtime block + mutex profile rates.
//
// runtime.SetBlockProfileRate has no return value so the previous rate
// cannot be recovered; restoring to zero on exit deliberately clears any
// previously active rate so heavyweight profiling is never left on by
// surprise.
//
// Returns func() which restores the previous rates when invoked; callers
// should defer it to ensure profiling does not stay enabled past the
// diagnostic window.
func (w *Watchdog) enableContentionProfiling() func() {
	runtime.SetBlockProfileRate(w.config.ContentionDiagnosticBlockProfileRate)
	originalMutexFraction := runtime.SetMutexProfileFraction(w.config.ContentionDiagnosticMutexProfileFraction)
	return func() {
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(originalMutexFraction)
	}
}

// waitContentionWindow blocks for the configured diagnostic window or
// returns early if the watchdog is being shut down or the context is
// cancelled.
//
// Returns nil when the timer fired normally, ctx.Cause when the context
// was cancelled, or ErrWatchdogStopped when Stop closed the stop channel.
func (w *Watchdog) waitContentionWindow(ctx context.Context) error {
	timer := w.clock.NewTimer(w.config.ContentionDiagnosticWindowDuration)
	defer timer.Stop()

	select {
	case <-timer.C():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-w.stopCh:
		return ErrWatchdogStopped
	}
}

// fireContentionDiagnosticOnce is the auto-fire path used by
// evaluateSchedulerLatency when ContentionDiagnosticAutoFire is enabled and
// the consecutive scheduler-latency events exceed the configured trigger.
// The diagnostic runs in a backgroundWG-tracked goroutine on a detached
// context so SIGTERM during the diagnostic does not truncate the captures.
func (w *Watchdog) fireContentionDiagnosticOnce(ctx context.Context) {
	detached := context.WithoutCancel(ctx)
	detached, cancel := context.WithTimeoutCause(detached,
		w.config.ContentionDiagnosticWindowDuration+contentionDiagnosticOverheadPad,
		errors.New("contention diagnostic exceeded window + overhead budget"))

	w.goSafely(&w.backgroundWG, func() {
		defer cancel()
		defer goroutine.RecoverPanic(detached, "monitoring.watchdogContentionDiagnostic")
		if err := w.RunContentionDiagnostic(detached); err != nil {
			_, l := logger_domain.From(detached, log)
			l.Warn("Contention diagnostic skipped", logger_domain.Error(err))
		}
	})
}
