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
	"compress/gzip"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

type mockProfilingController struct {
	captureCalls []string
	mu           sync.Mutex
}

func (m *mockProfilingController) Enable(_ context.Context, _ ProfilingEnableOpts) (*ProfilingStatus, error) {
	return &ProfilingStatus{}, nil
}

func (m *mockProfilingController) Disable(_ context.Context) (bool, error) {
	return true, nil
}

func (m *mockProfilingController) Close(_ context.Context) error {
	return nil
}

func (m *mockProfilingController) Status(_ context.Context) *ProfilingStatus {
	return &ProfilingStatus{}
}

func (m *mockProfilingController) SnapshotFlightRecorder(_ context.Context, _ io.Writer) error {

	return errors.New("rolling trace capture is not enabled")
}

func (m *mockProfilingController) CaptureProfile(_ context.Context, profileType string, _ int, writer io.Writer) (string, error) {
	m.mu.Lock()
	m.captureCalls = append(m.captureCalls, profileType)
	m.mu.Unlock()

	_, err := writer.Write([]byte("mock-profile-data-" + profileType))

	return "", err
}

func (m *mockProfilingController) getCaptureCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, len(m.captureCalls))
	copy(result, m.captureCalls)

	return result
}

func newTestWatchdog(t *testing.T, config WatchdogConfig, mockClock *clock.MockClock) *Watchdog {
	t.Helper()

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))

	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)

	watchdog.profileStore.clock = mockClock

	watchdog.resolveHeapThreshold(context.Background())

	t.Cleanup(func() {
		watchdog.Stop()
	})

	return watchdog
}

func TestDefaultWatchdogConfig(t *testing.T) {
	t.Parallel()

	config := DefaultWatchdogConfig()

	assert.InDelta(t, defaultHeapThresholdPercent, config.HeapThresholdPercent, 0.001)
	assert.Equal(t, uint64(defaultHeapThresholdBytes), config.HeapThresholdBytes)
	assert.Equal(t, defaultGoroutineThreshold, config.GoroutineThreshold)
	assert.Equal(t, defaultGoroutineSafetyCeiling, config.GoroutineSafetyCeiling)
	assert.InDelta(t, defaultGCPressureThreshold, config.GCPressureThreshold, 0.001)
	assert.Equal(t, defaultCooldown, config.Cooldown)
	assert.Equal(t, defaultMaxCapturesPerWindow, config.MaxCapturesPerWindow)
	assert.Equal(t, defaultCaptureWindow, config.CaptureWindow)
	assert.Equal(t, defaultHighWaterResetCooldown, config.HighWaterResetCooldown)
	assert.Equal(t, defaultWarmUpDuration, config.WarmUpDuration)
	assert.Equal(t, defaultMaxProfilesPerType, config.MaxProfilesPerType)
	assert.True(t, config.Enabled)
}

func TestNewWatchdog_HeapThresholdFromGomemlimit(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.HeapThresholdPercent = 0.80
	config.HeapThresholdBytes = 256 * 1024 * 1024

	watchdog := newTestWatchdog(t, config, mockClock)

	if watchdog.gomemlimit > 0 && watchdog.gomemlimit < (1<<63-1) {
		expected := uint64(float64(watchdog.gomemlimit) * 0.80)
		assert.Equal(t, expected, watchdog.initialHeapThreshold)
	} else {
		assert.Equal(t, uint64(256*1024*1024), watchdog.initialHeapThreshold)
	}

	assert.Equal(t, watchdog.initialHeapThreshold, watchdog.heapHighWater)
}

func TestNewWatchdog_HeapThresholdFromBytes(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.HeapThresholdBytes = 1024 * 1024 * 1024

	watchdog := newTestWatchdog(t, config, mockClock)

	if watchdog.gomemlimit <= 0 || watchdog.gomemlimit >= (1<<63-1) {
		assert.Equal(t, uint64(1024*1024*1024), watchdog.initialHeapThreshold)
	}

	assert.Equal(t, watchdog.initialHeapThreshold, watchdog.heapHighWater)
}

func TestWatchdog_WarmUpPeriod(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 30 * time.Second
	config.HeapThresholdBytes = 100

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	mockClock.Advance(5 * time.Second)

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 999_999_999},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)

	watchdog.captureWG.Wait()
	assert.Empty(t, controller.getCaptureCalls(), "no captures should occur during warm-up")
}

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
	_, gcPressureRecorded := watchdog.lastCaptureTime["gc_pressure"]
	watchdog.mu.Unlock()

	assert.True(t, gcPressureRecorded, "gc_pressure warning should have been recorded")
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

func TestWatchdog_NilProfilingController(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	assert.NotPanics(t, func() {
		watchdog.evaluate(context.Background(), &highHeapStats)
	}, "evaluate should not panic when the profiling controller is nil")
}

func TestWatchdog_StopIsIdempotent(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()

	watchdog := newTestWatchdog(t, config, mockClock)

	assert.NotPanics(t, func() {
		watchdog.Stop()
		watchdog.Stop()
		watchdog.Stop()
	}, "calling Stop multiple times should not panic")
}

func TestProfileStore_WriteAndRotate(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	maxProfiles := 3
	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: maxProfiles,
	}

	for i := range 5 {
		mockClock.Advance(time.Minute)

		data := []byte("profile-data-" + string(rune('A'+i)))
		err := store.write("heap", data)
		require.NoError(t, err, "write %d should succeed", i)
	}

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)

	heapFileCount := 0
	for _, entry := range entries {
		if entry.Name() != "" && len(entry.Name()) > 5 && entry.Name()[:5] == "heap-" {
			heapFileCount++
		}
	}

	assert.Equal(t, maxProfiles, heapFileCount, "rotation should keep only %d profiles", maxProfiles)
}

func TestProfileStore_GzipCompression(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	store := &profileStore{
		sandbox:            sandbox,
		clock:              mockClock,
		maxProfilesPerType: 5,
	}

	originalData := []byte("this is test profile data for gzip verification")

	err = store.write("goroutine", originalData)
	require.NoError(t, err)

	entries, err := sandbox.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	compressedData, err := sandbox.ReadFile(entries[0].Name())
	require.NoError(t, err)

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	require.NoError(t, err, "stored file must be valid gzip")

	decompressedData, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	require.NoError(t, gzipReader.Close())

	assert.Equal(t, originalData, decompressedData, "decompressed data should match the original input")
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

func TestWatchdog_StartDisabledIsNoop(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.Enabled = false

	watchdog := newTestWatchdog(t, config, mockClock)

	watchdog.Start(t.Context())

	assert.True(t, watchdog.startedAt.IsZero(), "startedAt should remain zero when watchdog is disabled")
}

type mockWatchdogNotifier struct {
	events []WatchdogEvent
	mu     sync.Mutex
}

func (m *mockWatchdogNotifier) Notify(_ context.Context, event WatchdogEvent) error {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()

	return nil
}

func (m *mockWatchdogNotifier) getEvents() []WatchdogEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]WatchdogEvent, len(m.events))
	copy(result, m.events)

	return result
}

func (m *mockWatchdogNotifier) getEventsByType(eventType WatchdogEventType) []WatchdogEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []WatchdogEvent
	for _, event := range m.events {
		if event.EventType == eventType {
			result = append(result, event)
		}
	}

	return result
}

type mockWatchdogProfileUploader struct {
	uploads []mockUploadRecord
	mu      sync.Mutex
}

type mockUploadRecord struct {
	profileType string
	metadata    map[string]string
	dataLength  int
}

func (m *mockWatchdogProfileUploader) Upload(_ context.Context, profileType string, data []byte, metadata map[string]string) error {
	m.mu.Lock()
	m.uploads = append(m.uploads, mockUploadRecord{
		profileType: profileType,
		metadata:    metadata,
		dataLength:  len(data),
	})
	m.mu.Unlock()

	return nil
}

func (m *mockWatchdogProfileUploader) getUploads() []mockUploadRecord {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]mockUploadRecord, len(m.uploads))
	copy(result, m.uploads)

	return result
}

func TestWatchdog_HeapThresholdNotifies(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	notifier := &mockWatchdogNotifier{}
	watchdog.SetProfilingController(controller)
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)
	watchdog.captureWG.Wait()
	watchdog.backgroundWG.Wait()

	assert.NotEmpty(t, notifier.getEvents(), "at least one notification should have been sent")
}

func TestWatchdog_GoroutineSafetyCeilingNotifies(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GoroutineThreshold = 100
	config.GoroutineSafetyCeiling = 50000
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime
	watchdog.goroutineBaseline.Store(10)

	ceilingStats := SystemStats{
		NumGoroutines: 60000,
	}

	watchdog.evaluate(context.Background(), &ceilingStats)
	watchdog.backgroundWG.Wait()

	ceilingEvents := notifier.getEventsByType(WatchdogEventGoroutineSafetyCeiling)
	assert.NotEmpty(t, ceilingEvents, "safety ceiling notification should have been sent")
	assert.Equal(t, WatchdogPriorityCritical, ceilingEvents[0].Priority)
}

func TestWatchdog_GCPressureNotifies(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.GCPressureThreshold = 0.5
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	gcStats := SystemStats{
		GC: GCInfo{GCCPUFraction: 0.6},
	}

	watchdog.evaluate(context.Background(), &gcStats)
	watchdog.backgroundWG.Wait()

	gcEvents := notifier.getEventsByType(WatchdogEventGCPressureWarning)
	assert.NotEmpty(t, gcEvents, "GC pressure notification should have been sent")
	assert.Equal(t, WatchdogPriorityNormal, gcEvents[0].Priority)
}

func TestWatchdog_GomemlimitWarningNotifies(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.HeapThresholdBytes = 100

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.resolveHeapThreshold(context.Background())

	watchdog.backgroundWG.Wait()

	if watchdog.gomemlimit <= 0 || watchdog.gomemlimit >= (1<<63-1) {
		assert.NotEmpty(t, notifier.getEventsByType(WatchdogEventGomemlimitNotConfigured),
			"GOMEMLIMIT warning notification should have been sent")
	}
}

func TestWatchdog_NilNotifierDoesNotPanic(t *testing.T) {
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

	assert.NotPanics(t, func() {
		watchdog.evaluate(context.Background(), &highHeapStats)
		watchdog.captureWG.Wait()
	}, "evaluate should not panic when the notifier is nil")
}

func TestWatchdog_ProfileUploadAfterCapture(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	uploader := &mockWatchdogProfileUploader{}
	watchdog.SetProfilingController(controller)
	watchdog.profileUploader = uploader
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)
	watchdog.captureWG.Wait()
	watchdog.backgroundWG.Wait()

	uploads := uploader.getUploads()
	assert.NotEmpty(t, uploads, "profile should have been uploaded after capture")

	if len(uploads) > 0 {
		assert.Equal(t, "heap", uploads[0].profileType)
		assert.Greater(t, uploads[0].dataLength, 0, "uploaded data should not be empty")
		assert.NotEmpty(t, uploads[0].metadata["hostname"])
		assert.NotEmpty(t, uploads[0].metadata["profile_type"])
	}
}

func TestWatchdog_NilUploaderDoesNotPanic(t *testing.T) {
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

	assert.NotPanics(t, func() {
		watchdog.evaluate(context.Background(), &highHeapStats)
		watchdog.captureWG.Wait()
	}, "evaluate should not panic when the uploader is nil")
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

func TestHeapTrendBuffer_Slope_LinearGrowth(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(10)

	for step := range 10 {
		buffer.add(uint64(100 + step*100))
	}

	slope := buffer.slope()

	assert.InDelta(t, 100.0, slope, 0.001, "slope should be 100 bytes per sample for linear growth")
}

func TestHeapTrendBuffer_Slope_StableHeap(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(10)

	for range 10 {
		buffer.add(500)
	}

	slope := buffer.slope()

	assert.InDelta(t, 0.0, slope, 0.001, "slope should be zero for a stable heap")
}

func TestHeapTrendBuffer_Slope_InsufficientSamples(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(10)
	buffer.add(100)

	slope := buffer.slope()

	assert.Equal(t, 0.0, slope, "slope should be zero with fewer than 2 samples")
}

func TestHeapTrendBuffer_RingBufferWraps(t *testing.T) {
	t.Parallel()

	buffer := newHeapTrendBuffer(5)

	for range 5 {
		buffer.add(0)
	}

	for step := range 5 {
		buffer.add(uint64(100 + step*100))
	}

	assert.True(t, buffer.isFull())

	slope := buffer.slope()

	assert.InDelta(t, 100.0, slope, 0.001, "slope should be 100 after ring buffer wraps")
}

func TestWatchdog_HeapTrendWarning(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.TrendWindowSize = 20
	config.TrendEvaluationInterval = 0
	config.TrendWarningHorizon = 5 * time.Minute
	config.CheckInterval = 500 * time.Millisecond
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	watchdog.heapTrendBuffer = newHeapTrendBuffer(config.TrendWindowSize)

	watchdog.gomemlimit = 1024 * 1024 * 1024

	baseMiB := uint64(800 * 1024 * 1024)
	stepMiB := uint64(10 * 1024 * 1024)

	for step := range 20 {
		heapAlloc := baseMiB + uint64(step)*stepMiB

		stats := SystemStats{
			Memory: MemoryInfo{HeapAlloc: heapAlloc},
		}

		mockClock.Advance(500 * time.Millisecond)
		watchdog.evaluate(context.Background(), &stats)
	}

	watchdog.backgroundWG.Wait()

	assert.NotEmpty(t, notifier.getEventsByType(WatchdogEventHeapTrendWarning),
		"heap trend warning notification should have been sent")
}

func TestWatchdog_HeapTrendNoWarningWhenStable(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.TrendWindowSize = 20
	config.TrendEvaluationInterval = 0
	config.TrendWarningHorizon = 5 * time.Minute
	config.CheckInterval = 500 * time.Millisecond

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime
	watchdog.heapTrendBuffer = newHeapTrendBuffer(config.TrendWindowSize)
	watchdog.gomemlimit = 1024 * 1024 * 1024

	for range 20 {
		stats := SystemStats{
			Memory: MemoryInfo{HeapAlloc: 200 * 1024 * 1024},
		}

		mockClock.Advance(500 * time.Millisecond)
		watchdog.evaluate(context.Background(), &stats)
	}

	trendEvents := notifier.getEventsByType(WatchdogEventHeapTrendWarning)
	assert.Empty(t, trendEvents, "no trend warning should be emitted for a stable heap")
}

func TestWatchdog_CapturePreDeathSnapshot(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	notifier := &mockWatchdogNotifier{}
	watchdog.SetProfilingController(controller)
	watchdog.notifier = notifier

	watchdog.CapturePreDeathSnapshot(context.Background())

	calls := controller.getCaptureCalls()

	heapCount := 0
	goroutineCount := 0
	for _, call := range calls {
		switch call {
		case "heap":
			heapCount++
		case "goroutine":
			goroutineCount++
		}
	}

	assert.Equal(t, 1, heapCount, "pre-death snapshot should capture one heap profile")
	assert.Equal(t, 1, goroutineCount, "pre-death snapshot should capture one goroutine profile")

	watchdog.backgroundWG.Wait()

	assert.NotEmpty(t, notifier.getEventsByType(WatchdogEventPreDeathSnapshot),
		"pre-death snapshot notification should have been sent")
}

func TestWatchdog_CapturePreDeathSnapshotRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	watchdog.CapturePreDeathSnapshot(cancelledCtx)

	calls := controller.getCaptureCalls()
	assert.Empty(t, calls, "no profiles should be captured when the context is already cancelled")
}

type errorProfilingController struct {
	mockProfilingController
	captureError error
}

func (e *errorProfilingController) CaptureProfile(_ context.Context, profileType string, _ int, _ io.Writer) (string, error) {
	e.mu.Lock()
	e.captureCalls = append(e.captureCalls, profileType)
	e.mu.Unlock()

	return "", e.captureError
}

type flightRecorderController struct {
	mockProfilingController
	traceData []byte
}

func (f *flightRecorderController) SnapshotFlightRecorder(_ context.Context, writer io.Writer) error {
	if len(f.traceData) == 0 {
		return errors.New("rolling trace capture is not enabled")
	}

	_, err := writer.Write(f.traceData)
	return err
}

func TestWatchdog_CaptureProfileControllerError(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &errorProfilingController{
		captureError: errors.New("simulated capture failure"),
	}
	notifier := &mockWatchdogNotifier{}
	watchdog.SetProfilingController(controller)
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)
	watchdog.captureWG.Wait()
	watchdog.backgroundWG.Wait()

	errorEvents := notifier.getEventsByType(WatchdogEventCaptureError)
	assert.NotEmpty(t, errorEvents, "capture error notification should have been sent")
}

func TestWatchdog_DeltaProfilingStoresBaseline(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second
	config.DeltaProfilingEnabled = true

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
	hasPrevious := watchdog.previousHeapProfile != nil
	watchdog.mu.Unlock()

	assert.True(t, hasPrevious, "previousHeapProfile should be set after the first capture")

	mockClock.Advance(2 * time.Second)

	watchdog.mu.Lock()
	watchdog.heapHighWater = 100
	watchdog.mu.Unlock()

	secondStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 300},
	}
	watchdog.evaluate(context.Background(), &secondStats)
	watchdog.captureWG.Wait()

	entries, err := watchdog.profileStore.sandbox.ReadDir(".")
	require.NoError(t, err)

	baselineCount := 0
	for _, entry := range entries {
		if len(entry.Name()) > 13 && entry.Name()[:13] == "heap-baseline" {
			baselineCount++
		}
	}

	assert.Greater(t, baselineCount, 0, "a heap-baseline profile should have been written on the second capture")
}

func TestWatchdog_FlightRecorderSnapshotCaptured(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &flightRecorderController{
		traceData: []byte("mock-trace-data"),
	}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)
	watchdog.captureWG.Wait()

	entries, err := watchdog.profileStore.sandbox.ReadDir(".")
	require.NoError(t, err)

	traceCount := 0
	for _, entry := range entries {
		if len(entry.Name()) > 6 && entry.Name()[:6] == "trace-" {
			traceCount++
		}
	}

	assert.Greater(t, traceCount, 0, "a flight recorder trace should have been captured alongside the heap profile")
}

func TestWatchdog_ConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(*WatchdogConfig)
	}{
		{
			name:   "zero CheckInterval",
			modify: func(c *WatchdogConfig) { c.CheckInterval = 0 },
		},
		{
			name:   "negative HeapThresholdPercent",
			modify: func(c *WatchdogConfig) { c.HeapThresholdPercent = -0.1 },
		},
		{
			name:   "HeapThresholdPercent above 1",
			modify: func(c *WatchdogConfig) { c.HeapThresholdPercent = 1.5 },
		},
		{
			name:   "zero MaxProfilesPerType",
			modify: func(c *WatchdogConfig) { c.MaxProfilesPerType = 0 },
		},
		{
			name:   "zero MaxCapturesPerWindow",
			modify: func(c *WatchdogConfig) { c.MaxCapturesPerWindow = 0 },
		},
		{
			name:   "zero GoroutineThreshold",
			modify: func(c *WatchdogConfig) { c.GoroutineThreshold = 0 },
		},
		{
			name:   "GoroutineSafetyCeiling equal to GoroutineThreshold",
			modify: func(c *WatchdogConfig) { c.GoroutineSafetyCeiling = c.GoroutineThreshold },
		},
		{
			name:   "negative TrendWindowSize",
			modify: func(c *WatchdogConfig) { c.TrendWindowSize = -1 },
		},
		{
			name:   "zero Cooldown",
			modify: func(c *WatchdogConfig) { c.Cooldown = 0 },
		},
		{
			name:   "zero MaxProfileSizeBytes",
			modify: func(c *WatchdogConfig) { c.MaxProfileSizeBytes = 0 },
		},
		{
			name:   "negative RSSThresholdPercent",
			modify: func(c *WatchdogConfig) { c.RSSThresholdPercent = -0.1 },
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := DefaultWatchdogConfig()
			testCase.modify(&config)

			collector := NewSystemCollector()
			_, err := NewWatchdog(config, collector)

			assert.ErrorIs(t, err, ErrInvalidWatchdogConfig,
				"NewWatchdog should return ErrInvalidWatchdogConfig for %s", testCase.name)
		})
	}
}

func TestWatchdog_ProfileSizeLimitEnforced(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.HeapThresholdBytes = 100
	config.Cooldown = time.Second
	config.MaxProfileSizeBytes = 10

	watchdog := newTestWatchdog(t, config, mockClock)
	controller := &mockProfilingController{}
	watchdog.SetProfilingController(controller)
	watchdog.startedAt = startTime

	highHeapStats := SystemStats{
		Memory: MemoryInfo{HeapAlloc: 200},
	}

	watchdog.evaluate(context.Background(), &highHeapStats)
	watchdog.captureWG.Wait()

	entries, err := watchdog.profileStore.sandbox.ReadDir(".")
	require.NoError(t, err)

	heapFileCount := 0
	for _, entry := range entries {
		if len(entry.Name()) > 5 && entry.Name()[:5] == "heap-" {
			heapFileCount++
		}
	}

	assert.Equal(t, 0, heapFileCount, "no heap profiles should be stored when they exceed the size limit")
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
