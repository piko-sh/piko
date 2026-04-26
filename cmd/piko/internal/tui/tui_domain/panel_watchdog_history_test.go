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

func newTestHistoryPanel() (*WatchdogHistoryPanel, *MockWatchdogProvider, *clock.MockClock) {
	mc := clock.NewMockClock(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	provider := NewMockWatchdogProvider()
	panel := NewWatchdogHistoryPanel(provider, mc)
	return panel, provider, mc
}

func TestHistoryPanelEmpty(t *testing.T) {
	panel, _, _ := newTestHistoryPanel()
	out := panel.View(160, 14)
	if !strings.Contains(out, "No history entries") {
		t.Errorf("expected empty placeholder: %q", out)
	}
}

func TestHistoryPanelRenders(t *testing.T) {
	panel, provider, mc := newTestHistoryPanel()
	provider.SetStartupHistory([]WatchdogStartupEntry{
		{
			PID: 4821, StartedAt: mc.Now().Add(-time.Hour), StoppedAt: mc.Now().Add(-30 * time.Minute),
			Reason: "clean", Hostname: "host-a", Version: "v1.2.3",
		},
		{
			PID: 4798, StartedAt: mc.Now().Add(-90 * time.Minute), StoppedAt: time.Time{},
			Reason: "", Hostname: "host-a", Version: "v1.2.3",
		},
	})
	panel.Update(panel.Init()())

	out := panel.View(160, 14)
	if !strings.Contains(out, "4821") || !strings.Contains(out, "4798") {
		t.Errorf("expected PIDs in output: %q", out)
	}
	if !strings.Contains(out, "running") {
		t.Errorf("expected running label for live entry: %q", out)
	}
}

func TestHistoryPanelFilterCycle(t *testing.T) {
	panel, _, _ := newTestHistoryPanel()
	want := []historyFilter{historyFilterClean, historyFilterUnclean, historyFilterRunning, historyFilterAll}
	for _, expected := range want {
		panel.Update(tea.KeyPressMsg{Code: 'f'})
		if panel.filter != expected {
			t.Errorf("filter = %d, want %d", panel.filter, expected)
		}
	}
}

func TestHistoryPanelCrashLoopDetected(t *testing.T) {
	panel, provider, mc := newTestHistoryPanel()

	provider.SetStartupHistory([]WatchdogStartupEntry{
		{PID: 1, StartedAt: mc.Now().Add(-2 * time.Minute), Reason: "panic"},
		{PID: 2, StartedAt: mc.Now().Add(-90 * time.Second), Reason: "unclean"},
		{PID: 3, StartedAt: mc.Now().Add(-30 * time.Second), Reason: "panic"},
	})
	provider.SetStatus(&WatchdogStatus{
		CrashLoopWindow:    5 * time.Minute,
		CrashLoopThreshold: 3,
	})
	panel.Update(panel.Init()())

	out := panel.View(160, 14)
	if !strings.Contains(out, "crash loop detected") {
		t.Errorf("expected crash-loop banner: %q", out)
	}
}

func TestHistoryPanelCrashLoopAbsent(t *testing.T) {
	panel, provider, mc := newTestHistoryPanel()

	provider.SetStartupHistory([]WatchdogStartupEntry{
		{PID: 1, StartedAt: mc.Now().Add(-3 * time.Minute), Reason: "clean", StoppedAt: mc.Now().Add(-2 * time.Minute)},
	})
	provider.SetStatus(&WatchdogStatus{
		CrashLoopWindow:    5 * time.Minute,
		CrashLoopThreshold: 3,
	})
	panel.Update(panel.Init()())

	out := panel.View(160, 14)
	if !strings.Contains(out, "no crash loop") {
		t.Errorf("expected absence banner: %q", out)
	}
}

func TestHistoryPanelTickRefreshes(t *testing.T) {
	panel, provider, mc := newTestHistoryPanel()
	provider.SetStartupHistory([]WatchdogStartupEntry{
		{PID: 1, StartedAt: mc.Now().Add(-time.Minute)},
	})
	panel.Update(panel.Init()())

	provider.SetStartupHistory([]WatchdogStartupEntry{
		{PID: 1, StartedAt: mc.Now().Add(-time.Minute)},
		{PID: 2, StartedAt: mc.Now().Add(-30 * time.Second)},
	})
	_, cmd := panel.Update(TickMessage{Time: mc.Now()})
	if cmd == nil {
		t.Fatalf("Tick should produce a refresh cmd")
	}
	panel.Update(cmd())

	if got := len(panel.visibleEntries()); got != 2 {
		t.Errorf("entries after tick = %d, want 2", got)
	}
}
