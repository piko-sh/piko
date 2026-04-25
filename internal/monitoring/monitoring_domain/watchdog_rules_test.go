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
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestWatchdog_HeapThresholdTriggersCapture(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)

	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	assert.Contains(t, calls, "heap", "heap capture should have been triggered")
}

func TestWatchdog_HeapCaptureWritesSidecarMetadata(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	stats := SystemStats{
		Memory:        MemoryInfo{HeapAlloc: 200},
		NumGoroutines: 17,
		Process:       ProcessInfo{RSS: 555, CgroupMemoryLimit: 1024, FDCount: 9, MaxOpenFilesSoft: 1024},
		GC:            GCInfo{GCCPUFraction: 0.05},
	}

	watchdog.evaluate(context.Background(), &stats)
	watchdog.captureWG.Wait()

	require.Contains(t, controller.getCaptureCalls(), "heap", "heap capture should have run")

	entries, err := watchdog.profileStore.sandbox.ReadDir(".")
	require.NoError(t, err)

	var sidecarName string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), profileSidecarExtension) && strings.HasPrefix(e.Name(), "heap-") {
			sidecarName = e.Name()
			break
		}
	}
	require.NotEmpty(t, sidecarName, "expected heap sidecar JSON next to profile")

	data, err := watchdog.profileStore.sandbox.ReadFile(sidecarName)
	require.NoError(t, err)

	var meta captureMetadata
	require.NoError(t, json.Unmarshal(data, &meta))
	assert.Equal(t, "heap_high_water", meta.RuleFired)
	assert.Equal(t, "heap", meta.ProfileType)
	assert.Equal(t, uint64(200), meta.ObservedValue)
	assert.NotZero(t, meta.PID, "PID should be populated")
	assert.NotEmpty(t, meta.RuntimeMetricsSnapshot, "runtime metrics snapshot should be embedded")
}

func TestWatchdog_FDPressureFiresAtThreshold(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.FDPressureThresholdPercent = 0.50
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	stats := SystemStats{
		Process: ProcessInfo{FDCount: 600, MaxOpenFilesSoft: 1000},
	}
	watchdog.evaluate(context.Background(), &stats)
	watchdog.backgroundWG.Wait()

	events := notifier.getEventsByType(WatchdogEventFDPressureExceeded)
	require.Len(t, events, 1)
	assert.Equal(t, WatchdogPriorityCritical, events[0].Priority)
	assert.Equal(t, "600", events[0].Fields["fd_count"])
	assert.Equal(t, "1000", events[0].Fields["fd_limit_soft"])
}

func TestWatchdog_FDPressureSilentWhenLimitUnknown(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.FDPressureThresholdPercent = 0.50
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	stats := SystemStats{
		Process: ProcessInfo{FDCount: 99999, MaxOpenFilesSoft: 0},
	}
	watchdog.evaluate(context.Background(), &stats)
	watchdog.backgroundWG.Wait()

	assert.Empty(t, notifier.getEventsByType(WatchdogEventFDPressureExceeded))
}

func TestWatchdog_FDPressureDisabledWhenThresholdZero(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.FDPressureThresholdPercent = 0
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	stats := SystemStats{
		Process: ProcessInfo{FDCount: 1000, MaxOpenFilesSoft: 1000},
	}
	watchdog.evaluate(context.Background(), &stats)
	watchdog.backgroundWG.Wait()

	assert.Empty(t, notifier.getEventsByType(WatchdogEventFDPressureExceeded))
}

func TestWatchdog_SchedulerLatencyFiresAndCountsConsecutive(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.SchedulerLatencyP99Threshold = 5 * time.Millisecond
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	for range 3 {
		stats := SystemStats{
			Schedule: SchedulerInfo{LatencyP99: 12 * time.Millisecond},
		}
		watchdog.evaluate(context.Background(), &stats)
		mockClock.Advance(2 * time.Second)
	}
	watchdog.backgroundWG.Wait()

	events := notifier.getEventsByType(WatchdogEventSchedulerLatencyHigh)
	require.Len(t, events, 3, "three observations above threshold should produce three events")

	seen := map[string]bool{}
	for _, ev := range events {
		seen[ev.Fields["consecutive_15m"]] = true
	}
	assert.True(t, seen["1"], "expected an event with consecutive_15m=1")
	assert.True(t, seen["2"], "expected an event with consecutive_15m=2")
	assert.True(t, seen["3"], "expected an event with consecutive_15m=3")
}

func TestWatchdog_SchedulerLatencyDisabledWhenThresholdZero(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.SchedulerLatencyP99Threshold = 0
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	stats := SystemStats{
		Schedule: SchedulerInfo{LatencyP99: 100 * time.Millisecond},
	}
	watchdog.evaluate(context.Background(), &stats)
	watchdog.backgroundWG.Wait()

	assert.Empty(t, notifier.getEventsByType(WatchdogEventSchedulerLatencyHigh))
}

func TestWatchdog_HeapHighWaterEscalation(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	firstStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}
	watchdog.evaluate(context.Background(), &firstStats)
	watchdog.captureWG.Wait()

	watchdog.mu.Lock()
	escalatedHighWater := watchdog.heapHighWater
	watchdog.mu.Unlock()

	assert.Equal(t, uint64(200), escalatedHighWater, "high-water mark should escalate to current heap")

	mockClock.Advance(2 * time.Second)

	secondStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}
	watchdog.evaluate(context.Background(), &secondStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	heapCaptureCount := 0
	for _, call := range calls {
		if call == "heap" {
			heapCaptureCount++
		}
	}

	assert.Equal(t, 1, heapCaptureCount, "no second capture should occur at the same heap level")
}

func TestWatchdog_HeapHighWaterReset(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 1000
	config.HighWaterResetCooldown = 10 * time.Minute
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	watchdog.mu.Lock()
	watchdog.heapHighWater = 5000
	watchdog.heapHighWaterSetAt = startTime
	watchdog.mu.Unlock()

	mockClock.Advance(11 * time.Minute)

	lowStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 100},
	}
	watchdog.evaluate(context.Background(), &lowStats)

	watchdog.mu.Lock()
	resetHighWater := watchdog.heapHighWater
	watchdog.mu.Unlock()

	assert.Equal(t, uint64(1000), resetHighWater, "high-water mark should reset to the initial threshold")
}

func TestWatchdog_GoroutineThreshold(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GoroutineThreshold = 100
	config.GoroutineSafetyCeiling = 100000
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime
	watchdog.goroutineBaseline.Store(10)

	goroutineStats := SystemStats{
		NumGoroutines: 200,
	}

	watchdog.evaluate(context.Background(), &goroutineStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	assert.Contains(t, calls, "goroutine", "goroutine capture should have been triggered")
}

func TestWatchdog_GoroutineSafetyCeiling(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GoroutineThreshold = 100
	config.GoroutineSafetyCeiling = 50000
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime
	watchdog.goroutineBaseline.Store(10)

	ceilingStats := SystemStats{
		NumGoroutines: 60000,
	}

	watchdog.evaluate(context.Background(), &ceilingStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	goroutineCaptureCount := 0
	for _, call := range calls {
		if call == "goroutine" {
			goroutineCaptureCount++
		}
	}

	assert.Equal(t, 0, goroutineCaptureCount, "no capture should occur when goroutine count exceeds the safety ceiling")
}

func TestWatchdog_GCPressureWarning(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GCPressureThreshold = 0.5
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	gcStats := SystemStats{
		GC: GCInfo{GCCPUFraction: 0.6},
	}

	watchdog.evaluate(context.Background(), &gcStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	assert.Empty(t, calls, "GC pressure should not trigger a profile capture")

	watchdog.mu.Lock()
	_, gcPressureCaptured := watchdog.lastCaptureTime["gc_pressure"]
	_, gcPressureWarned := watchdog.lastWarningTime["gc_pressure"]
	captureWindowEntries := len(watchdog.captureTimestamps)
	warningWindowEntries := len(watchdog.warningTimestamps)
	watchdog.mu.Unlock()

	assert.False(t, gcPressureCaptured, "gc_pressure must not consume the capture budget")
	assert.True(t, gcPressureWarned, "gc_pressure warning should have been recorded under the warning budget")
	assert.Empty(t, captureWindowEntries, "gc_pressure must not appear in the capture sliding window")
	assert.Equal(t, 1, warningWindowEntries, "gc_pressure should appear in the warning sliding window")
}

func TestWatchdog_CooldownPreventsRapidCaptures(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = 2 * time.Minute

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	firstStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}
	watchdog.evaluate(context.Background(), &firstStats)
	watchdog.captureWG.Wait()

	mockClock.Advance(10 * time.Second)

	watchdog.mu.Lock()
	watchdog.heapHighWater = 100
	watchdog.mu.Unlock()

	secondStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 300},
	}
	watchdog.evaluate(context.Background(), &secondStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	heapCaptureCount := 0
	for _, call := range calls {
		if call == "heap" {
			heapCaptureCount++
		}
	}

	assert.Equal(t, 1, heapCaptureCount, "cooldown should prevent a second capture")
}

func TestWatchdog_GlobalRateLimit(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second
	config.MaxCapturesPerWindow = 2
	config.CaptureWindow = 15 * time.Minute

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	watchdog.recordCapture(startTime, "heap")
	watchdog.recordCapture(startTime.Add(time.Second), "goroutine")

	mockClock.Advance(5 * time.Second)

	watchdog.mu.Lock()
	watchdog.heapHighWater = 100
	watchdog.mu.Unlock()

	exceededStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}
	watchdog.evaluate(context.Background(), &exceededStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	assert.Empty(t, calls, "no capture should occur when the global rate limit is exhausted")
}

func TestWatchdog_CheckCooldown(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.Cooldown = 2 * time.Minute

	watchdog := newTestWatchdog(t, config, mockClock)

	assert.True(t, watchdog.checkCooldown(mockClock.Now(), "heap"),
		"first cooldown check should pass with no prior captures")

	watchdog.recordCapture(mockClock.Now(), "heap")

	assert.False(t, watchdog.checkCooldown(mockClock.Now(), "heap"),
		"cooldown check should fail immediately after a capture")

	mockClock.Advance(3 * time.Minute)

	assert.True(t, watchdog.checkCooldown(mockClock.Now(), "heap"),
		"cooldown check should pass after the cooldown duration has elapsed")
}

func TestWatchdog_CheckGlobalRateLimit(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.MaxCapturesPerWindow = 3
	config.CaptureWindow = 10 * time.Minute

	watchdog := newTestWatchdog(t, config, mockClock)

	for i := range 3 {
		watchdog.recordCapture(startTime.Add(time.Duration(i)*time.Second), "heap")
	}

	assert.False(t, watchdog.checkGlobalRateLimit(mockClock.Now()),
		"rate limit should be enforced when window is full")

	mockClock.Advance(11 * time.Minute)

	assert.True(t, watchdog.checkGlobalRateLimit(mockClock.Now()),
		"rate limit should pass after all timestamps have expired")
}

func TestWatchdog_RecordCapture(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()

	watchdog := newTestWatchdog(t, config, mockClock)

	captureTime := startTime.Add(5 * time.Minute)

	watchdog.recordCapture(captureTime, "heap")

	watchdog.mu.Lock()
	lastHeapCapture := watchdog.lastCaptureTime["heap"]
	timestampCount := len(watchdog.captureTimestamps)
	watchdog.mu.Unlock()

	assert.Equal(t, captureTime, lastHeapCapture, "last capture time should match")
	assert.Equal(t, 1, timestampCount, "one timestamp should be recorded in the sliding window")
}

func TestWatchdog_EvaluateGoroutinesIgnoresBelowBaseline(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GoroutineThreshold = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime
	watchdog.goroutineBaseline.Store(50)

	belowBaselineStats := SystemStats{
		NumGoroutines: 40,
	}
	watchdog.evaluateGoroutines(context.Background(), mockClock.Now(), &belowBaselineStats)
	watchdog.captureWG.Wait()

	assert.Empty(t, controller.getCaptureCalls(), "no capture should occur when goroutine count is below baseline")
}

func TestWatchdog_EvaluateGoroutinesBelowThresholdNoCapture(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GoroutineThreshold = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime
	watchdog.goroutineBaseline.Store(10)

	belowThresholdStats := SystemStats{
		NumGoroutines: 50,
	}
	watchdog.evaluateGoroutines(context.Background(), mockClock.Now(), &belowThresholdStats)
	watchdog.captureWG.Wait()

	assert.Empty(t, controller.getCaptureCalls(), "no capture should occur when goroutine count is below the threshold")
}

func TestWatchdog_RSSThresholdTriggersCapture(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.RSSThresholdPercent = 0.85
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	notifier := &mockWatchdogNotifier{}
	watchdog.SetProfilingController(controller)
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	rssStats := SystemStats{
		Process: ProcessInfo{
			RSS:               900 * 1024 * 1024,
			CgroupMemoryLimit: 1024 * 1024 * 1024,
		},
	}

	watchdog.evaluate(context.Background(), &rssStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	assert.Contains(t, calls, "heap", "heap capture should have been triggered by RSS threshold")

	watchdog.backgroundWG.Wait()

	assert.NotEmpty(t, notifier.getEventsByType(WatchdogEventRSSThresholdExceeded),
		"RSS threshold notification should have been sent")
}

func TestWatchdog_RSSBelowThresholdNoCapture(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.RSSThresholdPercent = 0.85
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	lowRSSStats := SystemStats{
		Process: ProcessInfo{
			RSS:               500 * 1024 * 1024,
			CgroupMemoryLimit: 1024 * 1024 * 1024,
		},
	}

	watchdog.evaluate(context.Background(), &lowRSSStats)
	watchdog.captureWG.Wait()

	calls := controller.getCaptureCalls()
	rssCaptureCount := 0
	for _, call := range calls {
		if call == "heap" {
			rssCaptureCount++
		}
	}

	assert.Equal(t, 0, rssCaptureCount, "no capture should occur when RSS is below the threshold")
}

func TestWatchdog_RSSNoCgroupLimitSkips(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	noCgroupStats := SystemStats{
		Process: ProcessInfo{
			RSS:               900 * 1024 * 1024,
			CgroupMemoryLimit: 0,
		},
	}

	watchdog.evaluate(context.Background(), &noCgroupStats)
	watchdog.captureWG.Wait()

	assert.Empty(t, controller.getCaptureCalls(), "no capture should occur when cgroup memory limit is unknown")
}

func TestWatchdog_EvaluateGoroutineLeaksDisabled(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	assert.NotPanics(t, func() {
		watchdog.evaluateGoroutineLeaks(context.Background(), mockClock.Now())
	})
}
