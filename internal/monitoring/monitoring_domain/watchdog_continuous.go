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
	"bytes"
	"context"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
)

// continuousProfilingLoop periodically captures routine profile snapshots
// at the configured interval. The loop is opt-in (gated on
// ContinuousProfilingEnabled) and lives alongside the main evaluation loop --
// routine captures complement, not replace, threshold-triggered captures.
//
// Routine captures are written under a distinct prefix so they rotate
// independently from threshold-triggered ones; both sets coexist in the
// profile directory.
//
// Routine captures are skipped when the system is already in an unstable
// state (goroutine count above the safety ceiling) -- capturing pprof in
// that situation would worsen the failure.
func (w *Watchdog) continuousProfilingLoop(ctx context.Context) {
	defer goroutine.RecoverPanic(ctx, "monitoring.watchdogContinuousProfiling")

	ticker := w.clock.NewTicker(w.config.ContinuousProfilingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C():
			w.captureRoutineProfiles(ctx)
		}
	}
}

// captureRoutineProfiles iterates through the configured profile types and
// triggers a routine capture for each. Captures are dispatched to the same
// captureWG used by threshold-triggered captures so Stop() drains them
// cleanly.
func (w *Watchdog) captureRoutineProfiles(ctx context.Context) {
	stats := w.systemCollector.GetStats()
	if int(stats.NumGoroutines) >= w.config.GoroutineSafetyCeiling {
		_, l := logger_domain.From(ctx, log)
		l.Internal("Skipping routine profile capture: goroutine count exceeds safety ceiling",
			logger_domain.Int("goroutine_count", int(stats.NumGoroutines)),
			logger_domain.Int("safety_ceiling", w.config.GoroutineSafetyCeiling),
		)
		return
	}

	for _, profileType := range w.config.ContinuousProfilingTypes {
		if w.heapProfilingDisabled && requiresMemProfileRate(profileType) {
			continue
		}
		w.goSafely(&w.captureWG, func() {
			defer goroutine.RecoverPanic(ctx, "monitoring.watchdogRoutineCapture."+profileType)
			w.captureAndStoreRoutineProfile(ctx, profileType)
		})
	}
}

// requiresMemProfileRate reports whether a profile type is empty when the
// runtime allocation sampler is disabled.
//
// Takes profileType (string) which identifies the pprof profile type.
//
// Returns bool which is true when the profile depends on
// runtime.MemProfileRate sampling (heap and allocs).
func requiresMemProfileRate(profileType string) bool {
	return profileType == profileTypeHeap || profileType == "allocs"
}

// captureAndStoreRoutineProfile is the routine-mode counterpart to
// captureAndStoreProfile. It uses the routine-prefixed type name so storage
// rotation segregates routine files from threshold-triggered files, and
// uses the dedicated retention budget rather than the per-type cap.
//
// Errors are logged + counted but never propagate.
//
// Takes profileType (string) which is the pprof profile type to capture
// (heap, goroutine, allocs).
//
// Safe for concurrent use; reads the profilingController under the
// watchdog mutex and dispatches I/O outside the lock.
func (w *Watchdog) captureAndStoreRoutineProfile(ctx context.Context, profileType string) {
	if ctx.Err() != nil {
		return
	}
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	controller := w.profilingController
	w.mu.Unlock()

	if controller == nil {
		return
	}

	_, l := logger_domain.From(ctx, log)

	var buffer bytes.Buffer
	if _, err := controller.CaptureProfile(ctx, profileType, 0, &buffer); err != nil {
		l.Internal("Routine profile capture failed",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		return
	}

	profileData := buffer.Bytes()
	if int64(len(profileData)) > w.config.MaxProfileSizeBytes {
		watchdogCaptureErrorCount.Add(ctx, 1)
		return
	}

	prefixed := routineProfilePrefix + profileType
	timestamp, err := w.profileStore.writeWithRetention(prefixed, profileData, w.config.ContinuousProfilingRetention)
	if err != nil {
		l.Internal("Routine profile store write failed",
			String(logFieldProfileType, profileType),
			logger_domain.Error(err),
		)
		watchdogCaptureErrorCount.Add(ctx, 1)
		return
	}

	w.writeSidecarMetadata(ctx, prefixed, timestamp, captureContext{Rule: "routine"})
	watchdogRoutineProfileCaptureCount.Add(ctx, 1)

	if w.config.ContinuousProfilingNotify {
		w.sendNotification(ctx, NewRoutineProfileCapturedEvent(profileType))
	}
}
