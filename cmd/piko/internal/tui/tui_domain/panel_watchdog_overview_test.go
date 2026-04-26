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

func newTestOverviewPanel() (*WatchdogOverviewPanel, *MockWatchdogProvider, *clock.MockClock) {
	mc := clock.NewMockClock(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	provider := NewMockWatchdogProvider()
	panel := NewWatchdogOverviewPanel(provider, nil, mc)
	return panel, provider, mc
}

func TestOverviewPanelInitFetchesSnapshot(t *testing.T) {
	panel, provider, mc := newTestOverviewPanel()
	provider.SetStatus(&WatchdogStatus{
		Enabled:       true,
		StartedAt:     mc.Now().Add(-time.Hour),
		CaptureBudget: UtilisationGauge{Used: 1, Max: 5, Percent: 0.2},
		WarningBudget: UtilisationGauge{Used: 0, Max: 10, Percent: 0},
		HeapBudget:    UtilisationGauge{Used: 200, Max: 512, Percent: 200.0 / 512},
		Goroutines:    UtilisationGauge{Used: 100, Max: 5000, Percent: 0.02},
	})

	cmd := panel.Init()
	if cmd == nil {
		t.Fatalf("Init returned nil cmd")
	}
	msg := cmd()
	updated, _ := panel.Update(msg)
	if updated == nil {
		t.Fatalf("Update returned nil panel")
	}

	if got := panel.snapshot(); got == nil || !got.Enabled {
		t.Errorf("status not stored: %+v", got)
	}
}

func TestOverviewPanelViewRendersBody(t *testing.T) {
	panel, provider, mc := newTestOverviewPanel()
	provider.SetStatus(&WatchdogStatus{
		Enabled:       true,
		StartedAt:     mc.Now().Add(-2 * time.Hour),
		CaptureBudget: UtilisationGauge{Used: 3, Max: 5, Percent: 0.6},
		WarningBudget: UtilisationGauge{Used: 1, Max: 10, Percent: 0.1},
		HeapBudget:    UtilisationGauge{Used: 450, Max: 512, Percent: 0.88},
		Goroutines:    UtilisationGauge{Used: 800, Max: 5000, Percent: 0.16},
	})

	cmd := panel.Init()
	msg := cmd()
	panel.Update(msg)

	out := panel.View(120, 24)
	if !strings.Contains(out, "ENABLED") {
		t.Errorf("expected ENABLED tag in output: %q", out)
	}
	if !strings.Contains(out, "Capture window") {
		t.Errorf("expected Capture window gauge label: %q", out)
	}
}

func TestOverviewPanelNarrowDropsSectionNav(t *testing.T) {
	panel, provider, _ := newTestOverviewPanel()
	provider.SetStatus(&WatchdogStatus{Enabled: true})
	cmd := panel.Init()
	panel.Update(cmd())

	out := panel.View(60, 14)
	if strings.Contains(out, "SECTIONS") {
		t.Errorf("section nav should be suppressed at narrow widths: %q", out)
	}
}

func TestOverviewPanelSectionNavigation(t *testing.T) {
	panel, _, _ := newTestOverviewPanel()
	panel.SetSize(120, 24)

	if panel.Cursor() != 0 {
		t.Errorf("initial cursor = %d, want 0", panel.Cursor())
	}

	panel.Update(tea.KeyPressMsg{Code: 'j'})
	if panel.Cursor() != 1 {
		t.Errorf("after j cursor = %d, want 1", panel.Cursor())
	}

	panel.Update(tea.KeyPressMsg{Code: 'k'})
	if panel.Cursor() != 0 {
		t.Errorf("after k cursor = %d, want 0", panel.Cursor())
	}
}

func TestOverviewPanelAlertTapeFiltersToHighPriority(t *testing.T) {
	panel, _, mc := newTestOverviewPanel()

	panel.recordEvent(WatchdogEvent{EmittedAt: mc.Now(), Message: "info-msg", Priority: WatchdogPriorityNormal, EventType: WatchdogEventGCPressureWarning})
	panel.recordEvent(WatchdogEvent{EmittedAt: mc.Now(), Message: "high-msg", Priority: WatchdogPriorityHigh, EventType: WatchdogEventHeapThresholdExceeded})

	events := panel.recentHighPriorityEvents(5)
	if len(events) != 1 {
		t.Fatalf("expected 1 high event, got %d", len(events))
	}
	if events[0].Message != "high-msg" {
		t.Errorf("wrong event surfaced: %+v", events[0])
	}
}

func TestOverviewPanelAlertTapeRing(t *testing.T) {
	panel, _, mc := newTestOverviewPanel()

	for range overviewLocalEventCap + 5 {
		panel.recordEvent(WatchdogEvent{
			EmittedAt: mc.Now(),
			Message:   "x",
			Priority:  WatchdogPriorityHigh,
		})
	}

	panel.mu.RLock()
	defer panel.mu.RUnlock()
	if len(panel.events) != overviewLocalEventCap {
		t.Errorf("ring length = %d, want %d", len(panel.events), overviewLocalEventCap)
	}
}

func TestOverviewPanelHandlesNoStatus(t *testing.T) {
	panel, _, _ := newTestOverviewPanel()
	out := panel.View(120, 20)
	if !strings.Contains(out, "Awaiting first refresh") {
		t.Errorf("expected awaiting placeholder: %q", out)
	}
}

func TestOverviewPanelTickRefreshesSnapshot(t *testing.T) {
	panel, provider, mc := newTestOverviewPanel()
	provider.SetStatus(&WatchdogStatus{Enabled: true})
	panel.Update(panel.Init()())

	provider.SetStatus(&WatchdogStatus{Enabled: false, Stopped: true})

	_, cmd := panel.Update(TickMessage{Time: mc.Now()})
	if cmd == nil {
		t.Fatalf("Tick did not produce a refresh cmd")
	}
	panel.Update(cmd())

	if status := panel.snapshot(); status == nil || !status.Stopped {
		t.Errorf("expected snapshot to reflect stopped status: %+v", status)
	}
}
