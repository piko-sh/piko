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
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/wdk/clock"
)

func newTestDiagnosticPanel() (*WatchdogDiagnosticPanel, *MockWatchdogProvider, *clock.MockClock) {
	mc := clock.NewMockClock(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC))
	provider := NewMockWatchdogProvider()
	return NewWatchdogDiagnosticPanel(provider, mc), provider, mc
}

func TestDiagnosticPanelEmpty(t *testing.T) {
	panel, _, _ := newTestDiagnosticPanel()
	out := panel.View(80, 14)
	if !strings.Contains(out, "Run contention diagnostic") {
		t.Errorf("missing run hero: %q", out)
	}
	if !strings.Contains(out, "status unavailable") {
		t.Errorf("missing status hint: %q", out)
	}
}

func TestDiagnosticPanelReadyToRun(t *testing.T) {
	panel, provider, _ := newTestDiagnosticPanel()
	provider.SetStatus(&WatchdogStatus{
		ContentionDiagnosticCooldown: 30 * time.Minute,
		ContentionDiagnosticWindow:   60 * time.Second,
	})
	panel.Update(panel.Init()())

	out := panel.View(80, 14)
	if !strings.Contains(out, "ready to run") {
		t.Errorf("expected ready indicator: %q", out)
	}
}

func TestDiagnosticPanelCooldownActive(t *testing.T) {
	panel, provider, mc := newTestDiagnosticPanel()
	provider.SetStatus(&WatchdogStatus{
		ContentionDiagnosticLastRun:  mc.Now().Add(-5 * time.Minute),
		ContentionDiagnosticCooldown: 30 * time.Minute,
		ContentionDiagnosticWindow:   60 * time.Second,
	})
	panel.Update(panel.Init()())

	out := panel.View(80, 14)
	if !strings.Contains(out, "cooldown active") {
		t.Errorf("expected cooldown indicator: %q", out)
	}
}

func TestDiagnosticPanelRunHappy(t *testing.T) {
	panel, provider, _ := newTestDiagnosticPanel()
	provider.SetStatus(&WatchdogStatus{ContentionDiagnosticCooldown: time.Minute})
	panel.Update(panel.Init()())

	_, cmd := panel.Update(tea.KeyPressMsg{Code: 13})
	if cmd == nil {
		t.Fatalf("Enter should produce a run command")
	}
	msg := cmd()
	if _, ok := msg.(diagnosticRunDoneMsg); !ok {
		t.Fatalf("unexpected msg %T", msg)
	}
	panel.Update(msg)

	if panel.phase != diagnosticCompleted {
		t.Errorf("expected completed phase, got %d", panel.phase)
	}
	if provider.ContentionDiagnosticRunCount() != 1 {
		t.Errorf("RunContentionDiagnostic count = %d, want 1", provider.ContentionDiagnosticRunCount())
	}
}

func TestDiagnosticPanelRunFails(t *testing.T) {
	panel, provider, _ := newTestDiagnosticPanel()
	provider.Errors.ContentionDiagRun = errors.New("server busy")
	provider.SetStatus(&WatchdogStatus{ContentionDiagnosticCooldown: time.Minute})
	panel.Update(panel.Init()())

	_, cmd := panel.Update(tea.KeyPressMsg{Code: 13})
	msg := cmd()
	panel.Update(msg)

	if panel.phase != diagnosticFailed {
		t.Errorf("expected failed phase, got %d", panel.phase)
	}
	if panel.lastErr == nil {
		t.Error("expected lastErr to be set")
	}

	out := panel.View(80, 14)
	if !strings.Contains(out, "failed") {
		t.Errorf("expected failure surfaced in view: %q", out)
	}
}

func TestDiagnosticPanelRefuseConcurrentRuns(t *testing.T) {
	panel, provider, _ := newTestDiagnosticPanel()
	provider.SetStatus(&WatchdogStatus{ContentionDiagnosticCooldown: time.Minute})
	panel.Update(panel.Init()())

	_, cmd := panel.Update(tea.KeyPressMsg{Code: 13})
	if cmd == nil {
		t.Fatalf("first Enter should produce a cmd")
	}

	_, cmd2 := panel.Update(tea.KeyPressMsg{Code: 13})
	if cmd2 != nil {
		t.Errorf("second Enter while running should be a no-op")
	}
}
