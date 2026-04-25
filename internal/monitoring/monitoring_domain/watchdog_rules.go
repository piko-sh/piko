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
	ctx, l := logger_domain.From(ctx, log)
	heapAlloc := stats.Memory.HeapAlloc

	w.mu.Lock()
	reset := w.tryResetHeapHighWater(now, heapAlloc)
	belowHighWater := heapAlloc <= w.heapHighWater
	currentHighWater := w.heapHighWater
	w.mu.Unlock()

	if reset {
		w.recordHeapHighWaterReset(ctx, heapAlloc)
		return
	}
	if belowHighWater {
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	if !w.tryAdmitCapture(now, profileTypeHeap) {
		watchdogCooldownSkipCount.Add(ctx, 1)
		return
	}

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
	w.triggerCapture(ctx, profileTypeHeap, captureContext{
		Rule:      "heap_high_water",
		Observed:  heapAlloc,
		Threshold: currentHighWater,
	})
}

// recordHeapHighWaterReset logs and meters the high-water reset event so
// operators can correlate the gauge drop with the watchdog's decision.
//
// Takes heapAlloc (uint64) which is the current heap allocation logged
// alongside the reset, in bytes.
func (w *Watchdog) recordHeapHighWaterReset(ctx context.Context, heapAlloc uint64) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Reset heap high-water mark to initial threshold",
		logger_domain.Uint64("current_heap_alloc", heapAlloc),
	)
	watchdogHeapHighWaterResetCount.Add(ctx, 1)
	watchdogHeapHighWaterBytes.Record(ctx, safeconv.Uint64ToInt64(w.initialHeapThreshold))
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
	ctx, l := logger_domain.From(ctx, log)
	goroutineCount := int(stats.NumGoroutines)

	if goroutineCount <= int(w.goroutineBaseline.Load()) {
		return
	}

	if w.config.GoroutineThreshold > goroutineCount {
		return
	}

	if goroutineCount >= w.config.GoroutineSafetyCeiling {
		l.Error("Goroutine count exceeds safety ceiling, skipping capture",
			logger_domain.Int("goroutine_count", goroutineCount),
			logger_domain.Int("safety_ceiling", w.config.GoroutineSafetyCeiling),
		)
		w.sendNotification(ctx, NewGoroutineSafetyCeilingEvent(goroutineCount, w.config.GoroutineSafetyCeiling))
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	if !w.tryAdmitCapture(now, profileTypeGoroutine) {
		watchdogCooldownSkipCount.Add(ctx, 1)
		return
	}

	l.Warn("Goroutine count exceeded threshold, capturing profile",
		logger_domain.Int("goroutine_count", goroutineCount),
		logger_domain.Int("threshold", w.config.GoroutineThreshold),
	)

	watchdogGoroutineCaptureCount.Add(ctx, 1)

	w.sendNotification(ctx, NewGoroutineThresholdEvent(goroutineCount, w.config.GoroutineThreshold))
	w.triggerCapture(ctx, profileTypeGoroutine, captureContext{
		Rule:      "goroutine_threshold",
		Observed:  safeconv.IntToUint64(goroutineCount),
		Threshold: safeconv.IntToUint64(w.config.GoroutineThreshold),
	})
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
	ctx, l := logger_domain.From(ctx, log)
	gcCPUFraction := stats.GC.GCCPUFraction

	if gcCPUFraction <= w.config.GCPressureThreshold {
		return
	}

	if !w.tryAdmitWarning(now, "gc_pressure") {
		return
	}

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
	ctx, l := logger_domain.From(ctx, log)
	rss := stats.Process.RSS

	watchdogRSSBytes.Record(ctx, safeconv.Uint64ToInt64(rss))

	cgroupLimit := w.cgroupMemoryLimit
	if cgroupLimit == 0 {
		cgroupLimit = stats.Process.CgroupMemoryLimit
	}
	if cgroupLimit == 0 {
		return
	}

	threshold := uint64(float64(cgroupLimit) * w.config.RSSThresholdPercent)

	if rss < threshold {
		return
	}

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	if !w.tryAdmitCapture(now, "rss") {
		watchdogCooldownSkipCount.Add(ctx, 1)
		return
	}

	l.Warn("RSS approaching cgroup memory limit, capturing heap profile",
		logger_domain.Uint64("rss_bytes", rss),
		logger_domain.Uint64("cgroup_limit_bytes", cgroupLimit),
		logger_domain.Uint64("threshold_bytes", threshold),
	)

	watchdogRSSCaptureCount.Add(ctx, 1)

	w.sendNotification(ctx, NewRSSThresholdEvent(rss, cgroupLimit, threshold))
	w.triggerCapture(ctx, profileTypeHeap, captureContext{
		Rule:      "rss",
		Observed:  rss,
		Threshold: threshold,
	})
}

// evaluateGoroutineLeaks checks the Go 1.26 goroutine leak profile for
// unreachable blocked goroutines. This runs on a slower cadence than other
// evaluators because it piggybacks on the GC reachability walk.
//
// Takes now (time.Time) which is the current evaluation timestamp.
func (w *Watchdog) evaluateGoroutineLeaks(ctx context.Context, now time.Time) {
	ctx, l := logger_domain.From(ctx, log)
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

	w.mu.Lock()
	if w.profilingController == nil {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	if !w.tryAdmitCapture(now, profileTypeGoroutineLeak) {
		return
	}

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

	w.triggerCapture(ctx, profileTypeGoroutineLeak, captureContext{
		Rule:      "goroutineleak",
		Observed:  safeconv.IntToUint64(leakCount),
		Threshold: 0,
	})
}

// tryAdmitCapture atomically checks the per-type cooldown and the global
// capture window, then records the capture if both pass. Combining check
// + record under one lock removes the TOCTOU window where two concurrent
// rule evaluators could each see budget available, both fire captures,
// and over-spend the configured MaxCapturesPerWindow.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes profileType (string) which identifies the profile type being
// considered for capture.
//
// Returns bool which is true when the capture is admitted (and now
// recorded), false when cooldown or the global window denied it.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) tryAdmitCapture(now time.Time, profileType string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return admitInWindow(now, w.config.Cooldown, w.config.CaptureWindow, w.config.MaxCapturesPerWindow,
		w.lastCaptureTime, profileType, &w.captureTimestamps)
}

// tryAdmitWarning atomically checks the per-rule warning cooldown and the
// global warning-window budget, then records the warning if both pass.
// Mirror of tryAdmitCapture for warning-only rules (GC pressure, FD
// pressure, scheduler latency, heap trend).
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes ruleType (string) which identifies the warning rule being
// considered.
//
// Returns bool which is true when the warning is admitted (and now
// recorded), false when cooldown or the warning window denied it.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) tryAdmitWarning(now time.Time, ruleType string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return admitInWindow(now, w.config.Cooldown, w.config.CaptureWindow, w.config.MaxWarningsPerWindow,
		w.lastWarningTime, ruleType, &w.warningTimestamps)
}

// admitInWindow is the shared check-and-record kernel used by
// tryAdmitCapture and tryAdmitWarning. Callers must hold the watchdog
// mutex; the helper inspects the per-key last-time map and the supplied
// sliding-window timestamp slice, prunes expired entries, and records a
// new entry only when both cooldown and window budget allow it.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes cooldown (time.Duration) which is the per-key cooldown.
// Takes window (time.Duration) which is the rolling window for budgeting.
// Takes maxPerWindow (int) which is the budget cap.
// Takes lastByKey (map[string]time.Time) which records the per-key last
// admission time.
// Takes key (string) which identifies the per-key cooldown bucket.
// Takes timestamps (*[]time.Time) which is the sliding window slice; this
// helper prunes it in place.
//
// Returns bool which is true when the admission was recorded, false when
// the cooldown or window budget denied it.
func admitInWindow(
	now time.Time,
	cooldown time.Duration,
	window time.Duration,
	maxPerWindow int,
	lastByKey map[string]time.Time,
	key string,
	timestamps *[]time.Time,
) bool {
	if last, ok := lastByKey[key]; ok && now.Sub(last) < cooldown {
		return false
	}

	windowStart := now.Add(-window)
	pruned := (*timestamps)[:0]
	for _, ts := range *timestamps {
		if ts.After(windowStart) {
			pruned = append(pruned, ts)
		}
	}
	*timestamps = pruned

	if len(*timestamps) >= maxPerWindow {
		return false
	}

	lastByKey[key] = now
	*timestamps = append(*timestamps, now)
	return true
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

// evaluateFDPressure emits a warning when the open file descriptor count
// approaches the soft RLIMIT_NOFILE. The watchdog does not capture any
// profile because there is nothing useful to capture for FD exhaustion;
// alerting is the action.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the current FD count and limit.
func (w *Watchdog) evaluateFDPressure(ctx context.Context, now time.Time, stats *SystemStats) {
	ctx, l := logger_domain.From(ctx, log)
	fdCount := stats.Process.FDCount
	fdLimit := stats.Process.MaxOpenFilesSoft

	if w.config.FDPressureThresholdPercent <= 0 {
		return
	}

	if fdLimit > 0 {
		watchdogFDCount.Record(ctx, int64(fdCount))
	}

	if fdLimit <= 0 {
		return
	}

	threshold := int64(float64(fdLimit) * w.config.FDPressureThresholdPercent)
	if int64(fdCount) < threshold {
		return
	}

	if !w.tryAdmitWarning(now, "fd_pressure") {
		return
	}

	l.Error("Open file descriptor count is approaching the soft RLIMIT_NOFILE",
		logger_domain.Int("fd_count", int(fdCount)),
		logger_domain.Int64("fd_limit_soft", fdLimit),
		logger_domain.Int64("threshold", threshold),
	)

	watchdogFDPressureWarningCount.Add(ctx, 1)
	w.sendNotification(ctx, NewFDPressureEvent(fdCount, fdLimit, w.config.FDPressureThresholdPercent))
}

// evaluateSchedulerLatency emits a warning when the runtime/metrics
// scheduler latency p99 exceeds the configured threshold. High scheduler
// latency indicates that runnable goroutines are waiting too long for CPU
// -- symptoms include GC interference, lock contention, CPU starvation, or
// goroutine pile-ups.
//
// The rule also records each event timestamp in a small ring so the
// contention diagnostic (Phase 8) can detect repeated triggers and decide
// whether to escalate to mutex/block profiling.
//
// Takes now (time.Time) which is the current evaluation timestamp.
// Takes stats (*SystemStats) which contains the runtime/metrics-derived
// scheduler latency percentiles.
func (w *Watchdog) evaluateSchedulerLatency(ctx context.Context, now time.Time, stats *SystemStats) {
	ctx, l := logger_domain.From(ctx, log)
	threshold := w.config.SchedulerLatencyP99Threshold
	latency := stats.Schedule.LatencyP99

	if latency > 0 {
		watchdogSchedulerLatencyP99Nanos.Record(ctx, latency.Nanoseconds())
	}

	if threshold <= 0 {
		return
	}

	if latency <= threshold {
		return
	}

	if !w.tryAdmitWarning(now, "scheduler_latency") {
		return
	}

	consecutive := w.recordSchedulerLatencyEvent(now)

	l.Warn("Scheduler p99 latency exceeded threshold",
		logger_domain.Int64("latency_p99_ns", latency.Nanoseconds()),
		logger_domain.Int64("threshold_ns", threshold.Nanoseconds()),
		logger_domain.Int("consecutive_15m", consecutive),
	)

	watchdogSchedulerLatencyEventCount.Add(ctx, 1)
	w.sendNotification(ctx, NewSchedulerLatencyEvent(latency, threshold, consecutive))

	if w.config.ContentionDiagnosticAutoFire &&
		w.config.ContentionDiagnosticConsecutiveTrigger > 0 &&
		consecutive >= w.config.ContentionDiagnosticConsecutiveTrigger {
		w.fireContentionDiagnosticOnce(ctx)
	}
}

// recordSchedulerLatencyEvent appends a scheduler-latency event to the
// internal ring and returns the count of events within the last 15 minutes.
// Used by the contention diagnostic auto-fire path.
//
// Takes now (time.Time) which is the event timestamp to record.
//
// Returns int which is the count of recent events including this one.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) recordSchedulerLatencyEvent(now time.Time) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	trackingWindow := w.config.ContentionDiagnosticTriggerWindow
	if trackingWindow <= 0 {
		trackingWindow = defaultContentionDiagnosticTriggerWindow
	}

	cutoff := now.Add(-trackingWindow)
	pruned := w.schedulerLatencyEvents[:0]
	for _, t := range w.schedulerLatencyEvents {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	pruned = append(pruned, now)
	w.schedulerLatencyEvents = pruned
	return len(w.schedulerLatencyEvents)
}
