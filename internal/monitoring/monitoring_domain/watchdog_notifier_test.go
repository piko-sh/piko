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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/clock"
)

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

func TestNewContentionDiagnosticEventCarriesPhaseAndWindow(t *testing.T) {
	t.Parallel()

	event := NewContentionDiagnosticEvent("started", 90*time.Second)

	assert.Equal(t, WatchdogEventContentionDiagnostic, event.EventType)
	assert.Equal(t, WatchdogPriorityNormal, event.Priority)
	assert.Equal(t, "Contention diagnostic started", event.Message)
	assert.Equal(t, "started", event.Fields["phase"])
	assert.Equal(t, "1m30s", event.Fields["window"])
}

func TestNewRoutineProfileCapturedEventCarriesProfileType(t *testing.T) {
	t.Parallel()

	event := NewRoutineProfileCapturedEvent("heap")

	assert.Equal(t, WatchdogEventRoutineProfileCaptured, event.EventType)
	assert.Equal(t, WatchdogPriorityNormal, event.Priority)
	assert.Equal(t, "Routine profile captured", event.Message)
	assert.Equal(t, "heap", event.Fields["profile_type"])
}

func TestNewLoopPanickedEventCarriesPanicValue(t *testing.T) {
	t.Parallel()

	event := NewLoopPanickedEvent("nil pointer dereference")

	assert.Equal(t, WatchdogEventLoopPanicked, event.EventType)
	assert.Equal(t, WatchdogPriorityCritical, event.Priority)
	assert.Contains(t, event.Message, "panicked")
	assert.Equal(t, "nil pointer dereference", event.Fields["panic"])
}
