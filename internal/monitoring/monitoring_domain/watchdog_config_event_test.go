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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/clock"
)

func TestNewWatchdogConfigEvent_ReportsResolvedConfiguration(t *testing.T) {
	t.Parallel()

	status := &WatchdogStatusInfo{
		ProfileDirectory:             "/tmp/piko-watchdog",
		Hostname:                     "pod-7",
		Version:                      "v1.2.3",
		ContinuousProfilingTypes:     []string{"heap", "goroutine"},
		CheckInterval:                30 * time.Second,
		Cooldown:                     5 * time.Minute,
		WarmUpDuration:               time.Minute,
		CaptureWindow:                time.Hour,
		SchedulerLatencyP99Threshold: 50 * time.Millisecond,
		CrashLoopWindow:              10 * time.Minute,
		ContinuousProfilingInterval:  10 * time.Minute,
		ContentionDiagnosticWindow:   5 * time.Second,
		ContentionDiagnosticCooldown: time.Hour,
		GomemlimitBytes:              1 << 30,
		HeapThresholdBytes:           805306368,
		FDPressureThresholdPercent:   0.8,
		PID:                          4242,
		GoroutineThreshold:           10000,
		GoroutineSafetyCeiling:       50000,
		MaxProfilesPerType:           5,
		MaxCapturesPerWindow:         3,
		MaxWarningsPerWindow:         6,
		CrashLoopThreshold:           3,
		ContinuousProfilingRetention: 4,
		Enabled:                      true,
		ContinuousProfilingEnabled:   true,
		ContentionDiagnosticAutoFire: true,
		StartedAt:                    time.Time{},
		ContentionDiagnosticLastRun:  time.Time{},
		HeapHighWater:                0,
		CaptureWindowUsed:            0,
		WarningWindowUsed:            0,
		GoroutineBaseline:            0,
		Stopped:                      false,
	}

	event := NewWatchdogConfigEvent(status)

	assert.Equal(t, WatchdogEventConfig, event.EventType)
	assert.Equal(t, WatchdogPriorityNormal, event.Priority,
		"reporting configuration is informational, never a problem")

	assert.Equal(t, "805306368", event.Fields["heap_threshold_bytes"])
	assert.Equal(t, strconv.FormatInt(1<<30, 10), event.Fields["gomemlimit_bytes"])
	assert.Equal(t, "10000", event.Fields["goroutine_threshold"])
	assert.Equal(t, "0.8", event.Fields["fd_pressure_threshold_percent"])
	assert.Equal(t, "30s", event.Fields["check_interval"])
	assert.Equal(t, "50ms", event.Fields["scheduler_latency_p99_threshold"])
	assert.Equal(t, "heap,goroutine", event.Fields["continuous_profiling_types"])
	assert.Equal(t, "true", event.Fields["continuous_profiling_enabled"])
	assert.Equal(t, "pod-7", event.Fields["hostname"])
	assert.Equal(t, "v1.2.3", event.Fields["version"])
	assert.Equal(t, "4242", event.Fields["pid"])
	assert.Equal(t, "true", event.Fields["enabled"])
}

func TestNewWatchdogConfigEvent_NilStatus(t *testing.T) {
	t.Parallel()

	event := NewWatchdogConfigEvent(nil)

	assert.Equal(t, WatchdogEventConfig, event.EventType)
	assert.Nil(t, event.Fields, "an unavailable configuration reports nothing, not zeroes")
}

func TestWatchdogStart_EmitsConfigAfterThresholdResolution(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.ContinuousProfilingEnabled = false

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.Start(context.Background())
	watchdog.backgroundWG.Wait()

	var configEvents []WatchdogEvent
	for _, e := range notifier.getEvents() {
		if e.EventType == WatchdogEventConfig {
			configEvents = append(configEvents, e)
		}
	}
	require.Len(t, configEvents, 1, "the configuration is reported exactly once per start")

	fields := configEvents[0].Fields
	require.NotNil(t, fields)
	assert.Equal(t, strconv.FormatUint(watchdog.initialHeapThreshold, 10),
		fields["heap_threshold_bytes"], "the reported threshold is the resolved one")
	assert.Equal(t, "true", fields["enabled"])

	ring := watchdog.ListEvents(context.Background(), 0, time.Time{}, string(WatchdogEventConfig))
	require.Len(t, ring, 1)
	assert.Equal(t, fields, ring[0].Fields)
}

func TestWatchdogStart_DisabledEmitsNoConfig(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.Enabled = false

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.Start(context.Background())
	watchdog.backgroundWG.Wait()

	assert.Empty(t, notifier.getEvents(),
		"a disabled watchdog reports no configuration, because it enforces none")
}

func TestGetWatchdogStatus_CarriesProcessIdentity(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	watchdog := newTestWatchdog(t, DefaultWatchdogConfig(), mockClock)

	status := watchdog.GetWatchdogStatus(context.Background())

	assert.Positive(t, status.PID)
	assert.NotEmpty(t, status.Version)
	assert.Equal(t, watchdog.gomemlimit, status.GomemlimitBytes)
}

func TestWatchdogConfigEvent_ReportsEveryConfiguredField(t *testing.T) {
	t.Parallel()

	event := NewWatchdogConfigEvent(&WatchdogStatusInfo{})

	require.NotNil(t, event)
	assert.Len(t, event.Fields, configFieldCount,
		"configFieldCount no longer matches what the add*Fields helpers write")
}

func TestWatchdogConfigEvent_ReportsTheThresholdsBehindEveryRule(t *testing.T) {
	t.Parallel()

	event := NewWatchdogConfigEvent(&WatchdogStatusInfo{
		GCPressureThreshold: 0.25,
		RSSThresholdPercent: 90,
		TrendWindowSize:     12,
		TrendWarningHorizon: 30 * time.Minute,
	})

	require.NotNil(t, event)
	assert.Equal(t, "0.25", event.Fields["gc_pressure_threshold"])
	assert.Equal(t, "90", event.Fields["rss_threshold_percent"])
	assert.Equal(t, "12", event.Fields["trend_window_size"])
	assert.Equal(t, "30m0s", event.Fields["trend_warning_horizon"])
}
