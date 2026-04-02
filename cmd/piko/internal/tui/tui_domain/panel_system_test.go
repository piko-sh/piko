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

func TestNewSystemPanel(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "system" {
		t.Errorf("expected ID 'system', got %q", panel.ID())
	}
	if panel.Title() != "System" {
		t.Errorf("expected Title 'System', got %q", panel.Title())
	}
}

func TestNewSystemPanel_NilClock(t *testing.T) {
	panel := NewSystemPanel(nil, nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestFormatMillicores(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		m        float64
	}{
		{name: "small value", m: 500, expected: "500m"},
		{name: "zero", m: 0, expected: "0m"},
		{name: "one core", m: 1000, expected: "1.00"},
		{name: "multi core", m: 2500, expected: "2.50"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatMillicores(tc.m)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name     string
		contains string
		b        uint64
	}{
		{name: "bytes", b: 500, contains: "500 B"},
		{name: "kilobytes", b: 1024, contains: "KiB"},
		{name: "megabytes", b: 1024 * 1024, contains: "MiB"},
		{name: "gigabytes", b: 1024 * 1024 * 1024, contains: "GiB"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatBytes(tc.b)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected result to contain %q, got %q", tc.contains, result)
			}
		})
	}
}

func TestFormatUptime(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		d        time.Duration
	}{
		{name: "seconds only", d: 30 * time.Second, expected: "30s"},
		{name: "minutes and seconds", d: 5*time.Minute + 30*time.Second, expected: "5m30s"},
		{name: "hours minutes seconds", d: 2*time.Hour + 5*time.Minute + 30*time.Second, expected: "2h5m30s"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatUptime(tc.d)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestMinMaxAverage(t *testing.T) {
	testCases := []struct {
		name            string
		values          []float64
		expectedMin     float64
		expectedMax     float64
		expectedAverage float64
	}{
		{
			name:            "empty",
			values:          nil,
			expectedMin:     0,
			expectedMax:     0,
			expectedAverage: 0,
		},
		{
			name:            "single value",
			values:          []float64{10},
			expectedMin:     10,
			expectedMax:     10,
			expectedAverage: 10,
		},
		{
			name:            "multiple values",
			values:          []float64{1, 2, 3, 4, 5},
			expectedMin:     1,
			expectedMax:     5,
			expectedAverage: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			minVal, maxVal, avg := minMaxAverage(tc.values)
			if minVal != tc.expectedMin {
				t.Errorf("expected min %f, got %f", tc.expectedMin, minVal)
			}
			if maxVal != tc.expectedMax {
				t.Errorf("expected max %f, got %f", tc.expectedMax, maxVal)
			}
			if avg != tc.expectedAverage {
				t.Errorf("expected avg %f, got %f", tc.expectedAverage, avg)
			}
		})
	}
}

func TestSectionLabel(t *testing.T) {
	testCases := []struct {
		key      string
		expected string
	}{
		{key: "cpu", expected: "CPU"},
		{key: "memory", expected: "Memory"},
		{key: "goroutines", expected: "Goroutines"},
		{key: "gc", expected: "GC Pause"},
		{key: "build", expected: "Build"},
		{key: "process", expected: "Process"},
		{key: "runtime", expected: "Runtime"},
		{key: "unknown", expected: "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			result := sectionLabel(tc.key)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestSystemPanel_HandleRefreshMessage(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())

	stats := &SystemStats{
		NumGoroutines: 100,
		NumCPU:        4,
		GOMAXPROCS:    4,
		CPUMillicores: 500,
		Uptime:        time.Hour,
		Memory: SystemMemoryStats{
			Alloc:     1024 * 1024,
			HeapAlloc: 512 * 1024,
		},
		GC: SystemGCStats{
			NumGC:       10,
			LastPauseNs: 1000000,
		},
	}

	panel.handleRefreshMessage(SystemRefreshMessage{Stats: stats})

	panel.stateMutex.RLock()
	storedStats := panel.stats
	panel.stateMutex.RUnlock()

	if storedStats == nil {
		t.Fatal("expected stats to be stored")
	}
	if storedStats.NumGoroutines != 100 {
		t.Errorf("expected 100 goroutines, got %d", storedStats.NumGoroutines)
	}

	panel.handleRefreshMessage(SystemRefreshMessage{Err: ErrConnectionFailed})
	if !errors.Is(panel.err, ErrConnectionFailed) {
		t.Error("expected error to be set")
	}
}

func TestSystemPanel_HistoryTracking(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())

	for i := range 3 {
		panel.handleRefreshMessage(SystemRefreshMessage{
			Stats: &SystemStats{
				CPUMillicores: float64(100 * (i + 1)),
				NumGoroutines: 50 + i,
				Memory:        SystemMemoryStats{Alloc: uint64(1024 * (i + 1))},
			},
		})
	}

	panel.stateMutex.RLock()
	cpuHistory := panel.cpuHistory.Values()
	panel.stateMutex.RUnlock()

	if len(cpuHistory) != 3 {
		t.Errorf("expected 3 history values, got %d", len(cpuHistory))
	}
}

func TestSystemPanel_View(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	view := panel.View(100, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}

	if !strings.Contains(view, "System") {
		t.Error("expected 'System' title in view")
	}
}

func TestSystemPanel_ViewWithData(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())
	panel.SetSize(100, 24)
	panel.SetFocused(true)

	panel.handleRefreshMessage(SystemRefreshMessage{
		Stats: &SystemStats{
			NumGoroutines: 100,
			NumCPU:        4,
			GOMAXPROCS:    4,
			CPUMillicores: 500,
			Uptime:        time.Hour,
			Memory:        SystemMemoryStats{Alloc: 1024 * 1024},
			Build: SystemBuildInfo{
				Version:   "1.0.0",
				GoVersion: "go1.21",
			},
			Process: SystemProcessInfo{
				PID: 1234,
				RSS: 1024 * 1024 * 10,
			},
			Runtime: SystemRuntimeConfig{
				GOGC:       "100",
				GOMEMLIMIT: "off",
			},
			Cache: SystemCacheStats{
				ComponentCacheSize: 42,
				SVGCacheSize:       7,
			},
		},
	})

	view := panel.View(100, 24)
	if !strings.Contains(view, "Uptime") {
		t.Error("expected 'Uptime' in view")
	}
	if !strings.Contains(view, "CPU") {
		t.Error("expected 'CPU' in view")
	}
	if !strings.Contains(view, "Cache") {
		t.Error("expected 'Cache' in view")
	}
}

func TestSystemRenderer_GetID(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())
	renderer := &systemRenderer{panel: panel}

	section := systemSection{key: "cpu"}
	id := renderer.GetID(section)

	if id != "cpu" {
		t.Errorf("expected 'cpu', got %q", id)
	}
}

func TestSystemRenderer_MatchesFilter(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())
	renderer := &systemRenderer{panel: panel}

	testCases := []struct {
		key      string
		query    string
		expected bool
	}{
		{key: "cpu", query: "cpu", expected: true},
		{key: "cpu", query: "CPU", expected: true},
		{key: "memory", query: "mem", expected: true},
		{key: "gc", query: "pause", expected: true},
		{key: "cache", query: "cache", expected: true},
		{key: "cache", query: "Cache", expected: true},
		{key: "cpu", query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.key+"_"+tc.query, func(t *testing.T) {
			section := systemSection{key: tc.key}
			result := renderer.MatchesFilter(section, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestSystemRenderer_IsExpandable(t *testing.T) {
	panel := NewSystemPanel(nil, newTestClock())
	renderer := &systemRenderer{panel: panel}

	section := systemSection{key: "cpu"}
	if !renderer.IsExpandable(section) {
		t.Error("all system sections should be expandable")
	}
}
