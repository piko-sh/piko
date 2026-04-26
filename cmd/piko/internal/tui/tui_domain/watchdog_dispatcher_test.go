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

package tui_domain

import (
	"context"
	"testing"
	"time"
)

const dispatcherTestTimeout = 2 * time.Second

func newTestDispatcher(t *testing.T) (*EventDispatcher, *MockWatchdogProvider, context.CancelFunc) {
	t.Helper()
	provider := NewMockWatchdogProvider()
	dispatcher := NewEventDispatcher(provider, nil)
	dispatcher.SetBackoff(10*time.Millisecond, 50*time.Millisecond)
	dispatcher.SetHistoryCap(64)

	ctx, cancel := context.WithCancel(context.Background())
	dispatcher.Start(ctx)

	t.Cleanup(func() {
		cancel()
		dispatcher.Stop()
	})

	return dispatcher, provider, cancel
}

func waitForState(t *testing.T, d *EventDispatcher, expected string) {
	t.Helper()
	deadline := time.Now().Add(dispatcherTestTimeout)
	for time.Now().Before(deadline) {
		if d.State() == expected {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timeout waiting for state %q (current: %q)", expected, d.State())
}

func waitForEvent(t *testing.T, ch <-chan WatchdogEvent) WatchdogEvent {
	t.Helper()
	select {
	case ev, ok := <-ch:
		if !ok {
			t.Fatalf("event channel closed unexpectedly")
		}
		return ev
	case <-time.After(dispatcherTestTimeout):
		t.Fatalf("timeout waiting for event")
		return WatchdogEvent{}
	}
}

func TestDispatcherFanOut(t *testing.T) {
	d, provider, _ := newTestDispatcher(t)
	waitForState(t, d, WatchdogStreamConnected)

	sub := d.Subscribe(EventFilter{}, time.Time{})
	defer sub.Cancel()

	now := time.Now()
	provider.EmitEvent(WatchdogEvent{EmittedAt: now, Message: "first", EventType: WatchdogEventHeapThresholdExceeded, Priority: WatchdogPriorityHigh})
	provider.EmitEvent(WatchdogEvent{EmittedAt: now.Add(time.Second), Message: "second", EventType: WatchdogEventGCPressureWarning, Priority: WatchdogPriorityNormal})

	first := waitForEvent(t, sub.Events)
	second := waitForEvent(t, sub.Events)

	if first.Message != "first" || second.Message != "second" {
		t.Errorf("event order/messages wrong: %q / %q", first.Message, second.Message)
	}
}

func TestDispatcherFilterPriority(t *testing.T) {
	d, provider, _ := newTestDispatcher(t)
	waitForState(t, d, WatchdogStreamConnected)

	sub := d.Subscribe(EventFilter{MinPriority: WatchdogPriorityHigh}, time.Time{})
	defer sub.Cancel()

	now := time.Now()
	provider.EmitEvent(WatchdogEvent{EmittedAt: now, Message: "info", Priority: WatchdogPriorityNormal})
	provider.EmitEvent(WatchdogEvent{EmittedAt: now, Message: "high", Priority: WatchdogPriorityHigh})

	ev := waitForEvent(t, sub.Events)
	if ev.Message != "high" {
		t.Errorf("filter let normal-priority through: %+v", ev)
	}

	select {
	case stray := <-sub.Events:
		t.Errorf("expected no further events, received %+v", stray)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDispatcherFilterTypes(t *testing.T) {
	d, provider, _ := newTestDispatcher(t)
	waitForState(t, d, WatchdogStreamConnected)

	filter := EventFilter{
		Types: map[WatchdogEventType]struct{}{
			WatchdogEventHeapThresholdExceeded: {},
		},
	}
	sub := d.Subscribe(filter, time.Time{})
	defer sub.Cancel()

	provider.EmitEvent(WatchdogEvent{EmittedAt: time.Now(), Message: "heap", EventType: WatchdogEventHeapThresholdExceeded})
	provider.EmitEvent(WatchdogEvent{EmittedAt: time.Now(), Message: "gc", EventType: WatchdogEventGCPressureWarning})

	ev := waitForEvent(t, sub.Events)
	if ev.EventType != WatchdogEventHeapThresholdExceeded {
		t.Errorf("type filter failed: %+v", ev)
	}

	select {
	case stray := <-sub.Events:
		t.Errorf("expected no further events, received %+v", stray)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDispatcherBackfill(t *testing.T) {
	d, provider, _ := newTestDispatcher(t)
	waitForState(t, d, WatchdogStreamConnected)

	provider.EmitEvent(WatchdogEvent{EmittedAt: time.Now(), Message: "old", EventType: WatchdogEventHeapThresholdExceeded})

	deadline := time.Now().Add(dispatcherTestTimeout)
	for time.Now().Before(deadline) {
		if len(d.HistorySnapshot()) >= 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	sub := d.Subscribe(EventFilter{}, time.Time{})
	defer sub.Cancel()

	ev := waitForEvent(t, sub.Events)
	if ev.Message != "old" {
		t.Errorf("backfill missing: got %+v", ev)
	}
}

func TestDispatcherStop(t *testing.T) {
	provider := NewMockWatchdogProvider()
	dispatcher := NewEventDispatcher(provider, nil)
	dispatcher.SetBackoff(10*time.Millisecond, 50*time.Millisecond)

	ctx := t.Context()
	dispatcher.Start(ctx)
	waitForState(t, dispatcher, WatchdogStreamConnected)

	sub := dispatcher.Subscribe(EventFilter{}, time.Time{})

	dispatcher.Stop()

	select {
	case _, ok := <-sub.Events:
		if ok {
			t.Errorf("expected closed channel after Stop, received event")
		}
	case <-time.After(dispatcherTestTimeout):
		t.Fatalf("timeout waiting for channel close")
	}
}

func TestDispatcherDropCounters(t *testing.T) {
	provider := NewMockWatchdogProvider()
	dispatcher := NewEventDispatcher(provider, nil)
	dispatcher.SetBackoff(10*time.Millisecond, 50*time.Millisecond)

	dispatcher.subscriberBuffer = 1

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); dispatcher.Stop() })
	dispatcher.Start(ctx)
	waitForState(t, dispatcher, WatchdogStreamConnected)

	sub := dispatcher.Subscribe(EventFilter{}, time.Time{})
	defer sub.Cancel()

	for range 200 {
		provider.EmitEvent(WatchdogEvent{EmittedAt: time.Now(), Message: "x"})
	}

	deadline := time.Now().Add(dispatcherTestTimeout)
	for time.Now().Before(deadline) {
		if sub.Dropped() > 0 && dispatcher.DroppedTotal() > 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected drops; sub=%d total=%d", sub.Dropped(), dispatcher.DroppedTotal())
}

func TestDispatcherStateTransitions(t *testing.T) {
	d, _, _ := newTestDispatcher(t)
	waitForState(t, d, WatchdogStreamConnected)
	if state := d.State(); state != WatchdogStreamConnected {
		t.Errorf("State = %q, want %q", state, WatchdogStreamConnected)
	}
}
