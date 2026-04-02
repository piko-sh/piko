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
)

func TestNewHealthPanel(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "health" {
		t.Errorf("expected ID 'health', got %q", panel.ID())
	}
	if panel.Title() != "Health" {
		t.Errorf("expected Title 'Health', got %q", panel.Title())
	}
}

func TestNewHealthPanel_NilClock(t *testing.T) {
	panel := NewHealthPanel(nil, nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestHealthStateIndicator(t *testing.T) {
	testCases := []struct {
		name  string
		state HealthState
	}{
		{name: "healthy", state: HealthStateHealthy},
		{name: "degraded", state: HealthStateDegraded},
		{name: "unhealthy", state: HealthStateUnhealthy},
		{name: "unknown", state: HealthStateUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := healthStateIndicator(tc.state)
			if result == "" {
				t.Error("expected non-empty indicator")
			}
		})
	}
}

func TestHealthStateToValue(t *testing.T) {
	testCases := []struct {
		state    HealthState
		expected float64
	}{
		{state: HealthStateHealthy, expected: 1.0},
		{state: HealthStateDegraded, expected: 0.5},
		{state: HealthStateUnhealthy, expected: 0.0},
		{state: HealthStateUnknown, expected: 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.state.String(), func(t *testing.T) {
			result := healthStateToValue(tc.state)
			if result != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, result)
			}
		})
	}
}

func TestHealthPanel_HandleRefreshMessage(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())

	liveness := &HealthStatus{Name: "liveness", State: HealthStateHealthy}
	readiness := &HealthStatus{Name: "readiness", State: HealthStateHealthy}

	panel.handleRefreshMessage(HealthRefreshMessage{
		Liveness:  liveness,
		Readiness: readiness,
	})

	panel.stateMutex.RLock()
	storedLiveness := panel.liveness
	storedReadiness := panel.readiness
	panel.stateMutex.RUnlock()

	if storedLiveness == nil || storedReadiness == nil {
		t.Error("expected health data to be stored")
	}

	panel.handleRefreshMessage(HealthRefreshMessage{Err: ErrConnectionFailed})
	if !errors.Is(panel.err, ErrConnectionFailed) {
		t.Error("expected error to be set")
	}
}

func TestHealthPanel_HistoryTracking(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())

	for i := range 3 {
		state := HealthStateHealthy
		if i == 1 {
			state = HealthStateDegraded
		}
		panel.handleRefreshMessage(HealthRefreshMessage{
			Liveness:  &HealthStatus{State: state},
			Readiness: &HealthStatus{State: HealthStateHealthy},
		})
	}

	panel.stateMutex.RLock()
	livenessHistory := panel.livenessHistory.Values()
	panel.stateMutex.RUnlock()

	if len(livenessHistory) != 3 {
		t.Errorf("expected 3 history values, got %d", len(livenessHistory))
	}
}

func TestHealthPanel_View(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	view := panel.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "Waiting") {
		t.Error("expected waiting message")
	}
}

func TestHealthPanel_ViewWithData(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	panel.handleRefreshMessage(HealthRefreshMessage{
		Liveness: &HealthStatus{
			Name:     "liveness",
			State:    HealthStateHealthy,
			Duration: 10 * time.Millisecond,
		},
		Readiness: &HealthStatus{
			Name:     "readiness",
			State:    HealthStateHealthy,
			Duration: 5 * time.Millisecond,
		},
	})

	view := panel.View(80, 24)
	if !strings.Contains(view, "Liveness") {
		t.Error("expected 'Liveness' in view")
	}
	if !strings.Contains(view, "Readiness") {
		t.Error("expected 'Readiness' in view")
	}
}

func TestHealthRenderer_GetID(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())
	renderer := &healthRenderer{panel: panel}

	testCases := []struct {
		name     string
		expected string
		item     healthDisplayItem
	}{
		{
			name:     "probe row",
			item:     healthDisplayItem{probeKey: "liveness", isProbeRow: true},
			expected: "liveness",
		},
		{
			name:     "dependency row",
			item:     healthDisplayItem{probeKey: "liveness", dependencyIndex: 0, isProbeRow: false},
			expected: "liveness:0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.GetID(tc.item)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestHealthRenderer_MatchesFilter(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())
	renderer := &healthRenderer{panel: panel}

	testCases := []struct {
		name     string
		query    string
		item     healthDisplayItem
		expected bool
	}{
		{
			name:     "probe matches",
			item:     healthDisplayItem{probeKey: "liveness", isProbeRow: true},
			query:    "liveness",
			expected: true,
		},
		{
			name:     "dependency matches",
			item:     healthDisplayItem{dependency: &HealthStatus{Name: "database"}, isProbeRow: false},
			query:    "database",
			expected: true,
		},
		{
			name:     "no match",
			item:     healthDisplayItem{probeKey: "liveness", isProbeRow: true},
			query:    "xyz",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.MatchesFilter(tc.item, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestHealthRenderer_IsExpandable(t *testing.T) {
	panel := NewHealthPanel(nil, newTestClock())
	renderer := &healthRenderer{panel: panel}

	testCases := []struct {
		name     string
		item     healthDisplayItem
		expected bool
	}{
		{
			name: "probe with dependencies is expandable",
			item: healthDisplayItem{
				probeStatus: &HealthStatus{Dependencies: []*HealthStatus{{}}},
				isProbeRow:  true,
			},
			expected: true,
		},
		{
			name: "probe without dependencies not expandable",
			item: healthDisplayItem{
				probeStatus: &HealthStatus{Dependencies: nil},
				isProbeRow:  true,
			},
			expected: false,
		},
		{
			name: "dependency with message is expandable",
			item: healthDisplayItem{
				dependency: &HealthStatus{Message: "error", State: HealthStateUnhealthy},
				isProbeRow: false,
			},
			expected: true,
		},
		{
			name: "healthy dependency without message not expandable",
			item: healthDisplayItem{
				dependency: &HealthStatus{Message: "", State: HealthStateHealthy},
				isProbeRow: false,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.IsExpandable(tc.item)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSortHealthDependencies(t *testing.T) {
	deps := []*HealthStatus{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "beta"},
	}

	sorted := sortHealthDependencies(deps)

	if sorted[0].Name != "alpha" || sorted[1].Name != "beta" || sorted[2].Name != "zebra" {
		t.Error("expected sorted by name")
	}

	if deps[0].Name != "zebra" {
		t.Error("original slice should not be modified")
	}
}
