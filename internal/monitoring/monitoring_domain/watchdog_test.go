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
	"io"
	"strings"
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

func TestValidateWatchdogConfig_RejectsContinuousProfilingShortInterval(t *testing.T) {
	t.Parallel()

	config := DefaultWatchdogConfig()
	config.ContinuousProfilingEnabled = true
	config.ContinuousProfilingInterval = time.Second

	err := validateWatchdogConfig(&config)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidWatchdogConfig)
}

func TestValidateWatchdogConfig_RejectsContinuousProfilingDisallowedType(t *testing.T) {
	t.Parallel()

	config := DefaultWatchdogConfig()
	config.ContinuousProfilingEnabled = true
	config.ContinuousProfilingTypes = []string{"cpu"}

	err := validateWatchdogConfig(&config)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidWatchdogConfig)
}

func TestValidateWatchdogConfig_RejectsContentionDiagnosticWindowOutOfRange(t *testing.T) {
	t.Parallel()

	config := DefaultWatchdogConfig()
	config.ContentionDiagnosticWindowDuration = 10 * time.Minute

	err := validateWatchdogConfig(&config)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidWatchdogConfig)
}

func TestValidateWatchdogConfig_RejectsInvalidFDPressureThreshold(t *testing.T) {
	t.Parallel()

	config := DefaultWatchdogConfig()
	config.FDPressureThresholdPercent = 1.5

	err := validateWatchdogConfig(&config)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidWatchdogConfig)
}

func TestValidateWatchdogConfig_RejectsNegativeSchedulerThreshold(t *testing.T) {
	t.Parallel()

	config := DefaultWatchdogConfig()
	config.SchedulerLatencyP99Threshold = -time.Millisecond

	err := validateWatchdogConfig(&config)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidWatchdogConfig)
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
	metadata    map[string]string
	profileType string
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
	captureError error
	mockProfilingController
}

func (e *errorProfilingController) CaptureProfile(_ context.Context, profileType string, _ int, _ io.Writer) (string, error) {
	e.mu.Lock()
	e.captureCalls = append(e.captureCalls, profileType)
	e.mu.Unlock()

	return "", e.captureError
}

type flightRecorderController struct {
	traceData []byte
	mockProfilingController
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
		modify func(*WatchdogConfig)
		name   string
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

func TestWatchdog_WithWatchdogNotifierConfiguresNotifier(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	notifier := &mockWatchdogNotifier{}

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(DefaultWatchdogConfig(), collector,
		WithWatchdogClock(mockClock),
		WithWatchdogSandbox(sandbox),
		WithWatchdogNotifier(notifier),
	)
	require.NoError(t, err)
	t.Cleanup(watchdog.Stop)

	assert.Same(t, notifier, watchdog.notifier, "WithWatchdogNotifier should install the notifier")
}

func TestWatchdog_WithWatchdogProfileUploaderConfiguresUploader(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	uploader := &mockWatchdogProfileUploader{}

	tempDirectory := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDirectory, safedisk.ModeReadWrite)
	require.NoError(t, err)

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(DefaultWatchdogConfig(), collector,
		WithWatchdogClock(mockClock),
		WithWatchdogSandbox(sandbox),
		WithWatchdogProfileUploader(uploader),
	)
	require.NoError(t, err)
	t.Cleanup(watchdog.Stop)

	assert.Same(t, uploader, watchdog.profileUploader, "WithWatchdogProfileUploader should install the uploader")
}

func TestWatchdog_ListProfilesReturnsStoredEntries(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	watchdog := newTestWatchdog(t, config, mockClock)

	mockClock.Advance(time.Second)
	timestamp, err := watchdog.profileStore.write("heap", []byte("payload"))
	require.NoError(t, err)
	require.NoError(t, watchdog.profileStore.writeMetadata("heap", timestamp, captureMetadata{
		RuleFired: "heap_high_water", ProfileType: "heap",
	}))

	profiles, err := watchdog.ListProfiles(context.Background())
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	assert.Equal(t, "heap", profiles[0].Type)
	assert.True(t, profiles[0].HasSidecar, "sidecar pairing reported on the inspector surface")
}

func TestWatchdog_DownloadProfileWritesBytesToWriter(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	watchdog := newTestWatchdog(t, config, mockClock)

	mockClock.Advance(time.Second)
	timestamp, err := watchdog.profileStore.write("heap", []byte("compressed-bytes"))
	require.NoError(t, err)

	var buffer strings.Builder
	require.NoError(t, watchdog.DownloadProfile(context.Background(),
		"heap-"+timestamp+profileFileExtension, &buffer))
	assert.NotEmpty(t, buffer.String(), "downloaded profile should write bytes to the writer")
}

func TestWatchdog_DownloadProfileMissingReturnsError(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	watchdog := newTestWatchdog(t, config, mockClock)

	var buffer strings.Builder
	err := watchdog.DownloadProfile(context.Background(), "nonexistent.pb.gz", &buffer)
	require.Error(t, err, "missing profile should surface a read error")
}

func TestWatchdog_PruneProfilesByTypeAndAll(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	watchdog := newTestWatchdog(t, config, mockClock)

	mockClock.Advance(time.Second)
	_, err := watchdog.profileStore.write("heap", []byte("h"))
	require.NoError(t, err)
	mockClock.Advance(time.Second)
	_, err = watchdog.profileStore.write("heap", []byte("h2"))
	require.NoError(t, err)
	mockClock.Advance(time.Second)
	_, err = watchdog.profileStore.write("goroutine", []byte("g"))
	require.NoError(t, err)

	heapDeleted, err := watchdog.PruneProfiles(context.Background(), "heap")
	require.NoError(t, err)
	assert.Equal(t, 2, heapDeleted)

	allDeleted, err := watchdog.PruneProfiles(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, 1, allDeleted, "only the goroutine profile remained")
}

func TestWatchdog_GetWatchdogStatusReportsConfigAndState(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 30 * time.Second
	config.HeapThresholdBytes = 256 * 1024 * 1024
	config.GoroutineThreshold = 12345
	config.ContinuousProfilingTypes = []string{"heap", "goroutine"}
	config.ContentionDiagnosticAutoFire = true
	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	status := watchdog.GetWatchdogStatus(context.Background())
	require.NotNil(t, status)

	assert.True(t, status.Enabled)
	assert.False(t, status.Stopped)
	assert.Equal(t, 30*time.Second, status.WarmUpDuration)
	assert.Equal(t, 12345, status.GoroutineThreshold)
	assert.Equal(t, []string{"heap", "goroutine"}, status.ContinuousProfilingTypes)
	assert.True(t, status.ContentionDiagnosticAutoFire)
	assert.Equal(t, startTime, status.StartedAt)
}

func TestWatchdog_HandleLoopPanicEmitsCriticalEvent(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.handleLoopPanic(context.Background(), errors.New("simulated loop panic"))
	watchdog.backgroundWG.Wait()

	events := notifier.getEventsByType(WatchdogEventLoopPanicked)
	require.Len(t, events, 1)
	assert.Equal(t, WatchdogPriorityCritical, events[0].Priority)
	assert.Contains(t, events[0].Fields["panic"], "simulated loop panic")
}
