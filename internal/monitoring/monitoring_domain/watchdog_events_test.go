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

func TestWatchdog_ListEventsFiltersBySinceAndType(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 9, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	watchdog.mu.Lock()
	watchdog.eventRing = nil
	watchdog.mu.Unlock()

	watchdog.sendNotification(context.Background(), WatchdogEvent{
		EventType: WatchdogEventGCPressureWarning,
		Priority:  WatchdogPriorityNormal,
		Message:   "first",
	})
	mockClock.Advance(time.Minute)
	watchdog.sendNotification(context.Background(), WatchdogEvent{
		EventType: WatchdogEventHeapThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
		Message:   "second",
	})
	mockClock.Advance(time.Minute)
	watchdog.sendNotification(context.Background(), WatchdogEvent{
		EventType: WatchdogEventGCPressureWarning,
		Priority:  WatchdogPriorityNormal,
		Message:   "third",
	})

	all := watchdog.ListEvents(context.Background(), 0, time.Time{}, "")
	require.Len(t, all, 3, "ring should hold all three emissions")

	afterFirst := watchdog.ListEvents(context.Background(), 0, startTime.Add(30*time.Second), "")
	assert.Len(t, afterFirst, 2, "since-filter should drop the first event")

	gcOnly := watchdog.ListEvents(context.Background(), 0, time.Time{}, string(WatchdogEventGCPressureWarning))
	assert.Len(t, gcOnly, 2, "type filter keeps only matching events")

	limited := watchdog.ListEvents(context.Background(), 1, time.Time{}, "")
	require.Len(t, limited, 1)
	assert.Equal(t, "third", limited[0].Message, "limit returns the most recent event")
}

func TestWatchdog_SubscribeEventsBackfillAndLive(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	watchdog.mu.Lock()
	watchdog.eventRing = nil
	watchdog.mu.Unlock()

	watchdog.sendNotification(context.Background(), WatchdogEvent{
		EventType: WatchdogEventHeapThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
		Message:   "before-subscribe",
	})

	subCtx, subCancel := context.WithCancel(t.Context())
	defer subCancel()

	ch, cancel := watchdog.SubscribeEvents(subCtx, startTime.Add(-time.Hour))
	defer cancel()

	select {
	case event := <-ch:
		assert.Equal(t, "before-subscribe", event.Message)
	case <-time.After(time.Second):
		t.Fatal("expected backfilled event")
	}

	mockClock.Advance(time.Second)
	watchdog.sendNotification(context.Background(), WatchdogEvent{
		EventType: WatchdogEventGoroutineThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
		Message:   "live",
	})

	select {
	case event := <-ch:
		assert.Equal(t, "live", event.Message)
	case <-time.After(time.Second):
		t.Fatal("expected live event")
	}

	cancel()
	select {
	case _, open := <-ch:
		assert.False(t, open, "channel should be closed after cancel")
	case <-time.After(time.Second):
		t.Fatal("channel did not close after cancel")
	}
}

func TestWatchdog_DownloadSidecarReturnsPresent(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 11, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)

	timestamp, err := watchdog.profileStore.write("heap", []byte("dummy-pprof"))
	require.NoError(t, err)
	require.NoError(t, watchdog.profileStore.writeMetadata("heap", timestamp, captureMetadata{
		RuleFired:   "heap_high_water",
		ProfileType: "heap",
	}))

	profileFilename := "heap-" + timestamp + profileFileExtension

	data, present, err := watchdog.DownloadSidecar(context.Background(), profileFilename)
	require.NoError(t, err)
	assert.True(t, present, "sidecar should be reported present")
	assert.NotEmpty(t, data, "sidecar bytes should be returned")
}

func TestWatchdog_DownloadSidecarMissing(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	timestamp, err := watchdog.profileStore.write("heap", []byte("dummy-pprof"))
	require.NoError(t, err)

	profileFilename := "heap-" + timestamp + profileFileExtension

	data, present, err := watchdog.DownloadSidecar(context.Background(), profileFilename)
	require.NoError(t, err)
	assert.False(t, present, "sidecar should be reported absent")
	assert.Nil(t, data, "no bytes when sidecar is absent")
}

func TestWatchdog_GetStartupHistoryReturnsEntries(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 13, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	tempDir := t.TempDir()
	sandbox, err := safedisk.NewNoOpSandbox(tempDir, safedisk.ModeReadWrite)
	require.NoError(t, err)

	stopped := startTime.Add(-time.Hour)
	pre := startupHistoryFile{
		Entries: []startupHistoryEntry{
			{StartedAt: startTime.Add(-2 * time.Hour), StoppedAt: &stopped, PID: 100, Reason: "clean", Hostname: "alpha", Version: "v1"},
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

func TestWatchdog_StopClosesActiveSubscribers(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 14, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	ch, _ := watchdog.SubscribeEvents(context.Background(), time.Time{})

	watchdog.Stop()

	select {
	case _, open := <-ch:
		assert.False(t, open, "subscriber channel should be closed once watchdog Stop runs")
	case <-time.After(time.Second):
		t.Fatal("channel did not close after Stop")
	}
}

func TestCloseSubscriberIsIdempotent(t *testing.T) {
	t.Parallel()

	sub := &watchdogEventSubscriber{
		ch:   make(chan WatchdogEventInfo, 1),
		done: make(chan struct{}),
	}

	assert.True(t, closeSubscriber(sub), "first close reports the close happened")
	assert.False(t, closeSubscriber(sub), "second close is a no-op when already closed")
}

func TestDeliverEventToSubscriberDropsOldestWhenFull(t *testing.T) {
	t.Parallel()

	sub := &watchdogEventSubscriber{
		ch:   make(chan WatchdogEventInfo, 2),
		done: make(chan struct{}),
	}

	deliverEventToSubscriber(sub, WatchdogEventInfo{Message: "first"})
	deliverEventToSubscriber(sub, WatchdogEventInfo{Message: "second"})

	dropped := deliverEventToSubscriber(sub, WatchdogEventInfo{Message: "third"})
	assert.True(t, dropped, "third event should report a drop because the buffer was full")

	assert.Len(t, sub.ch, 2, "buffer should still hold two events after drop-oldest")

	got1 := <-sub.ch
	got2 := <-sub.ch
	assert.Equal(t, "second", got1.Message, "oldest event was dropped")
	assert.Equal(t, "third", got2.Message)
}

func TestSubscribeEventsContextCancellationClosesChannel(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, 4, 25, 15, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(startTime)

	config := DefaultWatchdogConfig()
	config.WarmUpDuration = 0

	watchdog := newTestWatchdog(t, config, mockClock)
	watchdog.startedAt = startTime

	ctx, cancel := context.WithCancel(t.Context())
	ch, _ := watchdog.SubscribeEvents(ctx, time.Time{})

	cancel()

	select {
	case _, open := <-ch:
		assert.False(t, open, "channel should be closed after ctx cancel")
	case <-time.After(time.Second):
		t.Fatal("channel did not close after ctx cancel")
	}
}
