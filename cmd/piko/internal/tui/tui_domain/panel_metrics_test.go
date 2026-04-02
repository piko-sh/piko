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
)

func TestNewMetricsPanel(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.ID() != "metrics" {
		t.Errorf("expected ID 'metrics', got %q", panel.ID())
	}
	if panel.Title() != "Metrics" {
		t.Errorf("expected Title 'Metrics', got %q", panel.Title())
	}
}

func TestNewMetricsPanel_NilClock(t *testing.T) {
	panel := NewMetricsPanel(nil, nil)

	if panel == nil {
		t.Fatal("expected non-nil panel")
	}
	if panel.clock == nil {
		t.Error("expected non-nil clock even when nil passed")
	}
}

func TestCalculateMetricStats(t *testing.T) {
	testCases := []struct {
		name            string
		values          []float64
		expectedMin     float64
		expectedMax     float64
		expectedAverage float64
	}{
		{
			name:            "empty values",
			values:          nil,
			expectedMin:     0,
			expectedMax:     0,
			expectedAverage: 0,
		},
		{
			name:            "single value",
			values:          []float64{42.0},
			expectedMin:     42.0,
			expectedMax:     42.0,
			expectedAverage: 42.0,
		},
		{
			name:            "multiple values",
			values:          []float64{10.0, 20.0, 30.0},
			expectedMin:     10.0,
			expectedMax:     30.0,
			expectedAverage: 20.0,
		},
		{
			name:            "with negative values",
			values:          []float64{-10.0, 0.0, 10.0},
			expectedMin:     -10.0,
			expectedMax:     10.0,
			expectedAverage: 0.0,
		},
		{
			name:            "all same values",
			values:          []float64{5.0, 5.0, 5.0, 5.0},
			expectedMin:     5.0,
			expectedMax:     5.0,
			expectedAverage: 5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			minVal, maxVal, avg := calculateMetricStats(tc.values)

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

func TestFormatMetricValue(t *testing.T) {
	testCases := []struct {
		name     string
		unit     string
		contains string
		value    float64
	}{
		{name: "giga with unit", value: 2.5e9, unit: "bytes", contains: "G"},
		{name: "mega", value: 3.5e6, unit: "", contains: "M"},
		{name: "kilo", value: 1500, unit: "ops", contains: "K"},
		{name: "normal", value: 42.5, unit: "", contains: "42.5"},
		{name: "milli", value: 0.005, unit: "", contains: "0.00"},
		{name: "zero", value: 0, unit: "", contains: "0"},
		{name: "scientific", value: 0.00000001, unit: "", contains: "e"},
		{name: "negative", value: -2500, unit: "", contains: "K"},
		{name: "with unit suffix", value: 100, unit: "ms", contains: "ms"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatMetricValue(tc.value, tc.unit)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected %q to contain %q", result, tc.contains)
			}
		})
	}
}

func TestMetricsRenderer_GetID(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())
	renderer := &metricsRenderer{panel: panel}

	metric := metricDisplay{name: "test.metric"}
	id := renderer.GetID(metric)

	if id != "test.metric" {
		t.Errorf("expected 'test.metric', got %q", id)
	}
}

func TestMetricsRenderer_MatchesFilter(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())
	renderer := &metricsRenderer{panel: panel}

	metric := metricDisplay{
		name:        "http_requests_total",
		description: "Total HTTP requests",
	}

	testCases := []struct {
		query    string
		expected bool
	}{
		{query: "http", expected: true},
		{query: "requests", expected: true},
		{query: "total", expected: true},
		{query: "HTTP", expected: false},
		{query: "xyz", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			result := renderer.MatchesFilter(metric, tc.query)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestMetricsRenderer_IsExpandable(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())
	renderer := &metricsRenderer{panel: panel}

	testCases := []struct {
		name     string
		metric   metricDisplay
		expected bool
	}{
		{
			name:     "with description",
			metric:   metricDisplay{description: "A description"},
			expected: true,
		},
		{
			name:     "with values",
			metric:   metricDisplay{values: []float64{1, 2, 3}},
			expected: true,
		},
		{
			name:     "with both",
			metric:   metricDisplay{description: "Desc", values: []float64{1}},
			expected: true,
		},
		{
			name:     "empty metric",
			metric:   metricDisplay{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.IsExpandable(tc.metric)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestMetricsRenderer_ExpandedLineCount(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())
	renderer := &metricsRenderer{panel: panel}

	testCases := []struct {
		name     string
		metric   metricDisplay
		expected int
	}{
		{
			name:     "no content",
			metric:   metricDisplay{},
			expected: 0,
		},
		{
			name:     "description only",
			metric:   metricDisplay{description: "Desc"},
			expected: 1,
		},
		{
			name:     "values only",
			metric:   metricDisplay{values: []float64{1}},
			expected: 1,
		},
		{
			name:     "both description and values",
			metric:   metricDisplay{description: "Desc", values: []float64{1}},
			expected: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := renderer.ExpandedLineCount(tc.metric)
			if result != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, result)
			}
		})
	}
}

func TestMetricsRenderer_RenderExpanded(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())
	renderer := &metricsRenderer{panel: panel}

	metric := metricDisplay{
		name:        "test",
		description: "Test metric description",
		unit:        "ms",
		values:      []float64{10, 20, 30},
	}

	lines := renderer.RenderExpanded(metric, 80)

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "Test metric description") {
		t.Error("expected description in first line")
	}

	if !strings.Contains(lines[1], "min") || !strings.Contains(lines[1], "max") {
		t.Error("expected stats in second line")
	}
}

func TestMetricsPanel_HandleRefreshMessage(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())

	metrics := []metricDisplay{
		{name: "metric1", values: []float64{1, 2, 3}, current: 3},
		{name: "metric2", values: []float64{10, 20}, current: 20},
	}
	panel.handleRefreshMessage(MetricsRefreshMessage{Metrics: metrics})

	items := panel.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	panel.handleRefreshMessage(MetricsRefreshMessage{Err: ErrConnectionFailed})
	if !errors.Is(panel.err, ErrConnectionFailed) {
		t.Error("expected error to be set")
	}
}

func TestMetricsPanel_View(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())
	panel.SetSize(80, 24)
	panel.SetFocused(true)

	view := panel.View(80, 24)
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestMetricsPanel_HistoryAccumulation(t *testing.T) {
	panel := NewMetricsPanel(nil, newTestClock())

	panel.handleRefreshMessage(MetricsRefreshMessage{
		Metrics: []metricDisplay{
			{name: "metric1", values: []float64{10}, current: 10},
		},
	})

	panel.handleRefreshMessage(MetricsRefreshMessage{
		Metrics: []metricDisplay{
			{name: "metric1", values: []float64{20}, current: 20},
		},
	})

	items := panel.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if len(items[0].values) != 2 {
		t.Errorf("expected 2 values in history, got %d", len(items[0].values))
	}
}
