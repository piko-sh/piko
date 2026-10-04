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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safedisk"
)

func TestWatchdog_StartupHistoryDetectsUncleanPreviousExit(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	history := startupHistoryFile{
		Entries: []startupHistoryEntry{{
			StartedAt: startTime.Add(-30 * time.Second),
			PID:       1234,
			Hostname:  "previous-host",
			Version:   "v0.0.1",
		}},
	}
	encoded, err := json.MarshalIndent(history, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.CrashLoopWindow = 60 * time.Second
	config.CrashLoopThreshold = 5

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock

	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.Start(context.Background())
	t.Cleanup(watchdog.Stop)

	watchdog.backgroundWG.Wait()

	previousEvents := notifier.getEventsByType(WatchdogEventPreviousCrashClassified)
	require.Len(t, previousEvents, 1, "previous crash should have been classified")
	assert.Equal(t, "1234", previousEvents[0].Fields["prev_pid"])
	assert.Equal(t, "previous-host", previousEvents[0].Fields["prev_hostname"])
}

func TestWatchdog_StartupHistoryDetectsCrashLoop(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	history := startupHistoryFile{
		Entries: []startupHistoryEntry{
			{StartedAt: startTime.Add(-50 * time.Second), PID: 100},
			{StartedAt: startTime.Add(-30 * time.Second), PID: 200},
			{StartedAt: startTime.Add(-10 * time.Second), PID: 300},
		},
	}
	encoded, err := json.MarshalIndent(history, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0
	config.CrashLoopWindow = 60 * time.Second
	config.CrashLoopThreshold = 3

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock

	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier

	watchdog.Start(context.Background())
	t.Cleanup(watchdog.Stop)

	watchdog.backgroundWG.Wait()

	loopEvents := notifier.getEventsByType(WatchdogEventCrashLoopDetected)
	require.Len(t, loopEvents, 1, "three unclean entries within window should trigger crash loop alert")
	assert.Equal(t, WatchdogPriorityCritical, loopEvents[0].Priority)
	assert.Equal(t, "3", loopEvents[0].Fields["unclean_in_window"])
}

func TestWatchdog_StartupHistoryFirstRunNoEvents(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	notifier := &mockWatchdogNotifier{}
	watchdog.notifier = notifier
	watchdog.startedAt = startTime

	watchdog.processStartupHistory(context.Background())
	watchdog.backgroundWG.Wait()

	assert.Empty(t, notifier.getEventsByType(WatchdogEventPreviousCrashClassified))
	assert.Empty(t, notifier.getEventsByType(WatchdogEventCrashLoopDetected))

	data, err := watchdog.profileStore.sandbox.ReadFile(startupHistoryFilename)
	require.NoError(t, err)
	var file startupHistoryFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.Len(t, file.Entries, 1)
	assert.True(t, file.Entries[0].StoppedAt.IsZero())
}

func TestWatchdog_StartupHistoryStopMarksClean(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	watchdog.processStartupHistory(context.Background())
	watchdog.markStartupHistoryStopped(context.Background())

	data, err := watchdog.profileStore.sandbox.ReadFile(startupHistoryFilename)
	require.NoError(t, err)
	var file startupHistoryFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.Len(t, file.Entries, 1)
	assert.False(t, file.Entries[0].StoppedAt.IsZero())
	assert.Equal(t, "clean", file.Entries[0].Reason)
}

func TestWatchdog_StartupHistoryRingTrimsToTen(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	stopped := startTime.Add(-time.Hour)
	entries := make([]startupHistoryEntry, 10)
	for i := range entries {
		entries[i] = startupHistoryEntry{
			StartedAt: startTime.Add(-time.Duration(20-i) * time.Minute),
			StoppedAt: stopped,
			PID:       1000 + i,
			Reason:    "clean",
		}
	}
	encoded, err := json.MarshalIndent(startupHistoryFile{Entries: entries}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock
	watchdog.startedAt = startTime

	watchdog.processStartupHistory(context.Background())

	data, err := sandbox.ReadFile(startupHistoryFilename)
	require.NoError(t, err)
	var file startupHistoryFile
	require.NoError(t, json.Unmarshal(data, &file))
	assert.LessOrEqual(t, len(file.Entries), maxStartupHistoryEntries, "ring must not exceed maxStartupHistoryEntries")

	assert.True(t, file.Entries[len(file.Entries)-1].StoppedAt.IsZero())
}

func TestWatchdog_GetStartupHistoryReturnsEntries(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	pre := startupHistoryFile{
		Entries: []startupHistoryEntry{
			{StartedAt: startTime.Add(-2 * time.Hour), StoppedAt: startTime.Add(-time.Hour), PID: 100, Reason: "clean", Hostname: "alpha", Version: "v1"},
			{StartedAt: startTime.Add(-30 * time.Minute), PID: 200, Hostname: "alpha", Version: "v2"},
		},
	}
	encoded, err := json.MarshalIndent(pre, "", "  ")
	require.NoError(t, err)
	require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, encoded, 0o640))

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
	watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
	require.NoError(t, err)
	watchdog.profileStore.clock = mockClock

	entries, err := watchdog.GetStartupHistory(context.Background())
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "v1", entries[0].Version)
	assert.Equal(t, "clean", entries[0].Reason)
	assert.False(t, entries[0].StoppedAt.IsZero())
	assert.True(t, entries[1].StoppedAt.IsZero(), "second entry has no clean stop")
}

func TestWatchdog_GetStartupHistoryWithoutUsableHistory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		contents []byte
		write    bool
		wantErr  bool
	}{
		{name: "missing file", write: false},
		{name: "empty file", write: true, contents: []byte{}},
		{name: "corrupt file", write: true, contents: []byte("{not json"), wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockClock := clock.NewMockClock(time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC))
			sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
			require.NoError(t, err)
			if tc.write {
				require.NoError(t, sandbox.WriteFileAtomic(startupHistoryFilename, tc.contents, 0o640))
			}

			config := DefaultWatchdogConfig()
			config.WarmUpDuration = 0
			collector := NewSystemCollector(WithSystemCollectorClock(mockClock))
			watchdog, err := NewWatchdog(config, collector, WithWatchdogClock(mockClock), WithWatchdogSandbox(sandbox))
			require.NoError(t, err)

			entries, err := watchdog.GetStartupHistory(context.Background())

			assert.Nil(t, entries)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWatchdog_GetStartupHistoryWithoutProfileStore(t *testing.T) {
	t.Parallel()

	watchdog := &Watchdog{}

	entries, err := watchdog.GetStartupHistory(context.Background())

	require.NoError(t, err)
	assert.Nil(t, entries)
}
