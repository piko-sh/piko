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

func newTestConfigPanel() (*WatchdogConfigPanel, *MockWatchdogProvider, *clock.MockClock) {
	mc := clock.NewMockClock(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	provider := NewMockWatchdogProvider()
	return NewWatchdogConfigPanel(provider, mc), provider, mc
}

func TestConfigPanelEmpty(t *testing.T) {
	panel, _, _ := newTestConfigPanel()
	out := panel.View(120, 24)
	if !strings.Contains(out, "Status snapshot unavailable") {
		t.Errorf("expected unavailable hint: %q", out)
	}
}

func TestConfigPanelRendersSections(t *testing.T) {
	panel, provider, _ := newTestConfigPanel()
	provider.SetStatus(&WatchdogStatus{
		Enabled:                      true,
		CheckInterval:                500 * time.Millisecond,
		Cooldown:                     2 * time.Minute,
		HeapBudget:                   UtilisationGauge{Used: 100, Max: 512, Percent: 100.0 / 512.0},
		Goroutines:                   UtilisationGauge{Used: 200, Max: 5000, Percent: 0.04},
		ContentionDiagnosticWindow:   60 * time.Second,
		ContentionDiagnosticCooldown: 30 * time.Minute,
	})
	panel.Update(panel.Init()())

	out := panel.View(120, 40)
	headings := []string{"LIFECYCLE", "THRESHOLDS", "CRASH LOOP", "CONTINUOUS PROFILING", "CONTENTION DIAGNOSTIC", "CAPTURE LIMITS"}
	for _, h := range headings {
		if !strings.Contains(out, h) {
			t.Errorf("missing section heading %q in output", h)
		}
	}
}

func TestConfigPanelNavigation(t *testing.T) {
	panel, provider, _ := newTestConfigPanel()
	provider.SetStatus(&WatchdogStatus{Enabled: true})
	panel.Update(panel.Init()())

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
