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
	"runtime/pprof"
	"strconv"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// highWaterResetNumerator is the numerator of the fraction of the initial
	// heap threshold below which heap usage must drop before the high-water
	// mark is reset (4/5 = 80%).
	highWaterResetNumerator = 4

	// highWaterResetDenominator is the denominator of the high-water reset
	// fraction (4/5 = 80%).
	highWaterResetDenominator = 5
)

// evaluateHeap checks whether the current heap allocation exceeds the
// high-water mark and triggers a heap profile capture when it does.
// It also handles escalation of the high-water mark and periodic reset
// back to the initial threshold when heap usage drops.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the current system metrics
// including heap allocation.
//
// Safe for concurrent use; acquires the watchdog's mutex for state updates.
func (w *Watchdog) evaluateHeap(ctx context.Context, now time.Time, stats *SystemStats) {
	heapAlloc := stats.Memory.HeapAlloc

	w.mu.Lock()
	reset := w.tryResetHeapHighWater(now, heapAlloc)
	belowHighWater := heapAlloc <= w.heapHighWater
	currentHighWater := w.heapHighWater
	w.mu.Unlock()

	if reset {
		_, l := logger_domain.From(ctx, log)
		l.Internal("Reset heap high-water mark to initial threshold",
			logger_domain.Uint64("current_heap_alloc", heapAlloc),
		)
		watchdogHeapHighWaterResetCount.Add(ctx, 1)
		watchdogHeapHighWaterBytes.Record(ctx, safeconv.Uint64ToInt64(w.initialHeapThreshold))
		return
	}

	if belowHighWater {
		return
	}

	if !w.checkCooldown(now, profileTypeHeap) {
		watchdogCooldownSkipCount.Add(ctx, 1)
		return
	}

	if !w.checkGlobalRateLimit(now) {
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.recordCapture(now, profileTypeHeap)

	_, l := logger_domain.From(ctx, log)
	l.Warn("Heap allocation exceeded high-water mark, capturing profile",
		logger_domain.Uint64("heap_alloc", heapAlloc),
		logger_domain.Uint64("high_water", currentHighWater),
	)

	watchdogHeapCaptureCount.Add(ctx, 1)

	w.mu.Lock()
	w.heapHighWater = heapAlloc
	w.heapHighWaterSetAt = now
	w.mu.Unlock()

	watchdogHeapHighWaterBytes.Record(ctx, safeconv.Uint64ToInt64(heapAlloc))

	w.sendNotification(ctx, NewHeapThresholdEvent(heapAlloc, currentHighWater))
	w.triggerCapture(ctx, profileTypeHeap)
}

// tryResetHeapHighWater resets the high-water mark to the initial threshold
// when heap usage has been below 80% of the initial threshold for longer than
// the configured reset cooldown. The caller must hold w.mu.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes heapAlloc (uint64) which is the current heap allocation in bytes.
//
// Returns bool which is true if the high-water mark was reset and the caller
// should skip further evaluation.
func (w *Watchdog) tryResetHeapHighWater(now time.Time, heapAlloc uint64) bool {
	if w.heapHighWater <= w.initialHeapThreshold ||
		now.Sub(w.heapHighWaterSetAt) <= w.config.HighWaterResetCooldown ||
		heapAlloc >= w.initialHeapThreshold*highWaterResetNumerator/highWaterResetDenominator {
		return false
	}

	w.heapHighWater = w.initialHeapThreshold
	w.heapHighWaterSetAt = now
	return true
}

// evaluateGoroutines checks whether the current goroutine count exceeds the
// configured threshold and triggers a goroutine profile capture when it does.
// Captures are suppressed if the count exceeds the safety ceiling, as the
// system may be too unstable for profiling.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the current system metrics
// including goroutine count.
func (w *Watchdog) evaluateGoroutines(ctx context.Context, now time.Time, stats *SystemStats) {
	goroutineCount := int(stats.NumGoroutines)

	if goroutineCount <= int(w.goroutineBaseline.Load()) {
		return
	}

	if w.config.GoroutineThreshold > goroutineCount {
		return
	}

	_, l := logger_domain.From(ctx, log)

	if goroutineCount >= w.config.GoroutineSafetyCeiling {
		l.Error("Goroutine count exceeds safety ceiling, skipping capture",
			logger_domain.Int("goroutine_count", goroutineCount),
			logger_domain.Int("safety_ceiling", w.config.GoroutineSafetyCeiling),
		)
		w.sendNotification(ctx, NewGoroutineSafetyCeilingEvent(goroutineCount, w.config.GoroutineSafetyCeiling))
		return
	}

	if !w.checkCooldown(now, profileTypeGoroutine) {
		watchdogCooldownSkipCount.Add(ctx, 1)
		return
	}

	if !w.checkGlobalRateLimit(now) {
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.recordCapture(now, profileTypeGoroutine)

	l.Warn("Goroutine count exceeded threshold, capturing profile",
		logger_domain.Int("goroutine_count", goroutineCount),
		logger_domain.Int("threshold", w.config.GoroutineThreshold),
	)

	watchdogGoroutineCaptureCount.Add(ctx, 1)

	w.sendNotification(ctx, NewGoroutineThresholdEvent(goroutineCount, w.config.GoroutineThreshold))
	w.triggerCapture(ctx, profileTypeGoroutine)
}

// evaluateGCPressure checks whether the GC CPU fraction exceeds the configured
// threshold and emits a warning log when it does. No profile capture is
// triggered because high GC pressure is best investigated with existing heap
// profiles.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the current system metrics
// including GC CPU fraction.
func (w *Watchdog) evaluateGCPressure(ctx context.Context, now time.Time, stats *SystemStats) {
	gcCPUFraction := stats.GC.GCCPUFraction

	if gcCPUFraction <= w.config.GCPressureThreshold {
		return
	}

	if !w.checkCooldown(now, "gc_pressure") {
		return
	}

	w.recordCapture(now, "gc_pressure")

	_, l := logger_domain.From(ctx, log)
	l.Warn("GC CPU fraction exceeded threshold",
		logger_domain.Float64("gc_cpu_fraction", gcCPUFraction),
		logger_domain.Float64("threshold", w.config.GCPressureThreshold),
	)

	watchdogGCPressureWarningCount.Add(ctx, 1)

	w.sendNotification(ctx, NewGCPressureEvent(gcCPUFraction, w.config.GCPressureThreshold))
}

// evaluateRSS checks whether the process RSS approaches the cgroup memory
// limit. This catches the failure mode where Go's heap metrics appear normal
// but the actual RSS (goroutine stacks, C allocations, fragmentation) is
// approaching the OOM killer threshold.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the current system metrics
// including RSS and cgroup memory limit.
func (w *Watchdog) evaluateRSS(ctx context.Context, now time.Time, stats *SystemStats) {
	cgroupLimit := stats.Process.CgroupMemoryLimit
	if cgroupLimit == 0 {
		return
	}

	rss := stats.Process.RSS
	threshold := uint64(float64(cgroupLimit) * w.config.RSSThresholdPercent)

	watchdogRSSBytes.Record(ctx, safeconv.Uint64ToInt64(rss))
	watchdogCgroupMemoryLimitBytes.Record(ctx, safeconv.Uint64ToInt64(cgroupLimit))

	if rss < threshold {
		return
	}

	if !w.checkCooldown(now, "rss") {
		watchdogCooldownSkipCount.Add(ctx, 1)
		return
	}

	if !w.checkGlobalRateLimit(now) {
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.recordCapture(now, "rss")

	_, l := logger_domain.From(ctx, log)
	l.Warn("RSS approaching cgroup memory limit, capturing heap profile",
		logger_domain.Uint64("rss_bytes", rss),
		logger_domain.Uint64("cgroup_limit_bytes", cgroupLimit),
		logger_domain.Uint64("threshold_bytes", threshold),
	)

	watchdogRSSCaptureCount.Add(ctx, 1)

	w.sendNotification(ctx, NewRSSThresholdEvent(rss, cgroupLimit, threshold))
	w.triggerCapture(ctx, profileTypeHeap)
}

// evaluateGoroutineLeaks checks the Go 1.26 goroutine leak profile for
// unreachable blocked goroutines. This runs on a slower cadence than other
// evaluators because it piggybacks on the GC reachability walk.
//
// Takes now (time.Time) which is the current evaluation timestamp.
func (w *Watchdog) evaluateGoroutineLeaks(ctx context.Context, now time.Time) {
	if !w.goroutineLeakAvailable {
		return
	}

	if now.Sub(w.lastGoroutineLeakCheck) < w.config.GoroutineLeakCheckInterval {
		return
	}

	w.lastGoroutineLeakCheck = now

	leakProfile := pprof.Lookup(profileTypeGoroutineLeak)
	if leakProfile == nil {
		return
	}

	leakCount := leakProfile.Count()
	if leakCount == 0 {
		return
	}

	if !w.checkCooldown(now, profileTypeGoroutineLeak) {
		return
	}

	if !w.checkGlobalRateLimit(now) {
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.recordCapture(now, profileTypeGoroutineLeak)

	_, l := logger_domain.From(ctx, log)
	l.Warn("Goroutine leak detected via Go 1.26 goroutine leak profile",
		logger_domain.Int("leaked_goroutine_count", leakCount),
	)

	watchdogGoroutineLeakDetectionCount.Add(ctx, 1)

	w.sendNotification(ctx, WatchdogEvent{
		EventType: WatchdogEventGoroutineLeakDetected,
		Priority:  WatchdogPriorityHigh,
		Message:   "Goroutine leak detected via the Go 1.26 goroutine leak profile",
		Fields: map[string]string{
			"leaked_goroutine_count": strconv.Itoa(leakCount),
		},
	})

	w.triggerCapture(ctx, profileTypeGoroutineLeak)
}

// checkCooldown checks whether enough time has elapsed since the last capture
// of the given profile type. It returns true if a new capture is permitted.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes profileType (string) which identifies the profile type to check.
//
// Returns bool which is true when the cooldown period has elapsed and a new
// capture is allowed.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) checkCooldown(now time.Time, profileType string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	lastCapture, exists := w.lastCaptureTime[profileType]
	if !exists {
		return true
	}

	return now.Sub(lastCapture) >= w.config.Cooldown
}

// checkGlobalRateLimit prunes expired timestamps from the sliding capture
// window and checks whether the number of recent captures is below the
// configured maximum.
//
// Takes now (time.Time) which is the current evaluation timestamp.
//
// Returns bool which is true when the number of captures in the current
// window is below MaxCapturesPerWindow.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) checkGlobalRateLimit(now time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	windowStart := now.Add(-w.config.CaptureWindow)

	pruned := w.captureTimestamps[:0]
	for _, timestamp := range w.captureTimestamps {
		if timestamp.After(windowStart) {
			pruned = append(pruned, timestamp)
		}
	}
	w.captureTimestamps = pruned

	return len(w.captureTimestamps) < w.config.MaxCapturesPerWindow
}

// recordCapture records a capture event for the given profile type and adds
// a timestamp to the global sliding window.
//
// Takes now (time.Time) which is the capture timestamp to record.
// Takes profileType (string) which identifies the profile type being captured.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) recordCapture(now time.Time, profileType string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.lastCaptureTime[profileType] = now
	w.captureTimestamps = append(w.captureTimestamps, now)
}
