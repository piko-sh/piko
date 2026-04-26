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
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/wdk/clock"
)

func newTestEventsPanel() (*WatchdogEventsPanel, *clock.MockClock) {
	mc := clock.NewMockClock(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	panel := NewWatchdogEventsPanel(nil, mc)
	return panel, mc
}

func TestEventsPanelEmptyState(t *testing.T) {
	panel, _ := newTestEventsPanel()
	out := panel.View(120, 14)
	if !strings.Contains(out, "No events received yet") {
		t.Errorf("expected empty-state placeholder: %q", out)
	}
}

func TestEventsPanelRecordEvent(t *testing.T) {
	panel, mc := newTestEventsPanel()
	panel.recordEvent(WatchdogEvent{
		EmittedAt: mc.Now(),
		Message:   "heap exceeded",
		EventType: WatchdogEventHeapThresholdExceeded,
		Priority:  WatchdogPriorityHigh,
	})

	out := panel.View(120, 14)
	if !strings.Contains(out, "heap_threshold_exceeded") {
		t.Errorf("expected event type in output: %q", out)
	}
	if !strings.Contains(out, "heap exceeded") {
		t.Errorf("expected event message in output: %q", out)
	}
}

func TestEventsPanelPauseResume(t *testing.T) {
	panel, _ := newTestEventsPanel()

	panel.Update(tea.KeyPressMsg{Code: ' '})
	if !panel.paused {
		t.Errorf("space should pause")
	}
	panel.Update(tea.KeyPressMsg{Code: ' '})
	if panel.paused {
		t.Errorf("space should resume")
	}
}

func TestEventsPanelPriorityFilter(t *testing.T) {
	panel, mc := newTestEventsPanel()
	panel.recordEvent(WatchdogEvent{EmittedAt: mc.Now(), Message: "info", Priority: WatchdogPriorityNormal})
	panel.recordEvent(WatchdogEvent{EmittedAt: mc.Now(), Message: "warn", Priority: WatchdogPriorityHigh})
	panel.recordEvent(WatchdogEvent{EmittedAt: mc.Now(), Message: "boom", Priority: WatchdogPriorityCritical})

	if got := len(panel.visibleEvents()); got != 3 {
		t.Errorf("default filter visible = %d, want 3", got)
	}

	panel.Update(tea.KeyPressMsg{Code: 'f'})
	if got := len(panel.visibleEvents()); got != 2 {
		t.Errorf("high+ filter visible = %d, want 2", got)
	}

	panel.Update(tea.KeyPressMsg{Code: 'f'})
	if got := len(panel.visibleEvents()); got != 1 {
		t.Errorf("critical filter visible = %d, want 1", got)
	}

	panel.Update(tea.KeyPressMsg{Code: 'f'})
	if got := len(panel.visibleEvents()); got != 3 {
		t.Errorf("filter cycle back to all = %d, want 3", got)
	}
}

func TestEventsPanelLocalRingEvicts(t *testing.T) {
	panel, mc := newTestEventsPanel()
	for range eventsLocalCap + 5 {
		panel.recordEvent(WatchdogEvent{EmittedAt: mc.Now(), Message: "x", Priority: WatchdogPriorityHigh})
	}

	panel.mu.RLock()
	defer panel.mu.RUnlock()
	if got := len(panel.events); got != eventsLocalCap {
		t.Errorf("ring length = %d, want %d", got, eventsLocalCap)
	}
}

func TestEventsPanelExpandToggle(t *testing.T) {
	panel, mc := newTestEventsPanel()
	panel.recordEvent(WatchdogEvent{
		EmittedAt: mc.Now(),
		Message:   "boom",
		EventType: WatchdogEventCrashLoopDetected,
		Priority:  WatchdogPriorityCritical,
		Fields:    map[string]string{"unclean": "3", "window": "60s"},
	})

	panel.SetSize(120, 14)
	panel.Update(tea.KeyPressMsg{Code: 13})
	if !panel.expanded {
		t.Errorf("Enter should expand")
	}

	out := panel.View(120, 14)
	if !strings.Contains(out, "unclean") {
		t.Errorf("expected expanded fields in output: %q", out)
	}
}

func TestEventsPanelHeaderShowsState(t *testing.T) {
	panel, _ := newTestEventsPanel()
	out := panel.View(120, 14)
	if !strings.Contains(out, "live") && !strings.Contains(out, "no source") {
		t.Errorf("expected state label in header: %q", out)
	}
}
