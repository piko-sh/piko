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

func newTestProfilesPanel() (*WatchdogProfilesPanel, *MockWatchdogProvider, *clock.MockClock) {
	mc := clock.NewMockClock(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	provider := NewMockWatchdogProvider()
	panel := NewWatchdogProfilesPanel(provider, mc)
	return panel, provider, mc
}

func TestProfilesPanelEmpty(t *testing.T) {
	panel, _, _ := newTestProfilesPanel()
	out := panel.View(160, 14)
	if !strings.Contains(out, "No profiles") {
		t.Errorf("expected empty placeholder: %q", out)
	}
}

func TestProfilesPanelLoadsAndRenders(t *testing.T) {
	panel, provider, mc := newTestProfilesPanel()
	provider.SetProfiles([]WatchdogProfile{
		{Filename: "heap-2026.pb.gz", Type: "heap", SizeBytes: 1024 * 1024 * 5, Timestamp: mc.Now().Add(-time.Minute)},
		{Filename: "goroutine-2026.pb.gz", Type: "goroutine", SizeBytes: 1024 * 256, Timestamp: mc.Now().Add(-30 * time.Second), HasSidecar: true},
	})

	cmd := panel.Init()
	panel.Update(cmd())

	out := panel.View(160, 14)
	if !strings.Contains(out, "heap-2026") {
		t.Errorf("expected heap profile in output: %q", out)
	}
	if !strings.Contains(out, "goroutine-2026") {
		t.Errorf("expected goroutine profile in output: %q", out)
	}
}

func TestProfilesPanelSortCycle(t *testing.T) {
	panel, _, _ := newTestProfilesPanel()
	wantOrder := []profileSortMode{profileSortType, profileSortSizeDesc, profileSortFilename, profileSortAgeDesc}
	for _, want := range wantOrder {
		panel.Update(tea.KeyPressMsg{Code: 's'})
		if panel.sortMode != want {
			t.Errorf("after s sortMode = %d, want %d", panel.sortMode, want)
		}
	}
}

func TestProfilesPanelTypeFilter(t *testing.T) {
	panel, provider, _ := newTestProfilesPanel()
	provider.SetProfiles([]WatchdogProfile{
		{Filename: "heap-1.pb.gz", Type: "heap"},
		{Filename: "goroutine-1.pb.gz", Type: "goroutine"},
	})
	panel.Update(panel.Init()())

	panel.Update(tea.KeyPressMsg{Code: 't'})
	if got := panel.activeTypeFilterLabel(); got != "goroutine" {
		t.Errorf("type filter = %q, want %q", got, "goroutine")
	}

	panel.Update(tea.KeyPressMsg{Code: 't'})
	if got := panel.activeTypeFilterLabel(); got != "heap" {
		t.Errorf("type filter = %q, want %q", got, "heap")
	}

	panel.Update(tea.KeyPressMsg{Code: 't'})
	if got := panel.activeTypeFilterLabel(); got != "all" {
		t.Errorf("type filter = %q, want %q", got, "all")
	}
}

func TestProfilesPanelPruneRunsOnP(t *testing.T) {
	panel, provider, _ := newTestProfilesPanel()
	provider.SetProfiles([]WatchdogProfile{{Filename: "heap-1.pb.gz", Type: "heap"}})
	panel.Update(panel.Init()())

	_, cmd := panel.Update(tea.KeyPressMsg{Code: 'P'})
	if cmd == nil {
		t.Fatalf("P should produce a prune command")
	}
	msg := cmd()
	if _, ok := msg.(profilesPruneCompletedMsg); !ok {
		t.Errorf("prune cmd produced %T, want profilesPruneCompletedMsg", msg)
	}
	if provider.PruneCallCount() != 1 {
		t.Errorf("PruneProfiles call count = %d, want 1", provider.PruneCallCount())
	}
}

func TestProfilesPanelTickRefreshes(t *testing.T) {
	panel, provider, mc := newTestProfilesPanel()
	provider.SetProfiles([]WatchdogProfile{{Filename: "a.pb.gz", Type: "heap"}})
	panel.Update(panel.Init()())

	provider.SetProfiles([]WatchdogProfile{
		{Filename: "a.pb.gz", Type: "heap"},
		{Filename: "b.pb.gz", Type: "heap"},
	})

	_, cmd := panel.Update(TickMessage{Time: mc.Now()})
	if cmd == nil {
		t.Fatalf("Tick should produce a refresh cmd")
	}
	panel.Update(cmd())

	if got := len(panel.visibleProfiles()); got != 2 {
		t.Errorf("after tick visible profiles = %d, want 2", got)
	}
}
