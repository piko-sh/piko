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

	"charm.land/lipgloss/v2"
)

func TestSparkline_Empty(t *testing.T) {
	result := Sparkline(nil, new(DefaultSparklineConfig()))

	if !strings.Contains(result, "─") {
		t.Errorf("empty sparkline should contain dashes, got: %s", result)
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	config := DefaultSparklineConfig()
	config.Width = 10
	result := Sparkline([]float64{50.0}, &config)

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestSparkline_AscendingValues(t *testing.T) {
	config := DefaultSparklineConfig()
	config.Width = 5
	config.ShowCurrent = false
	result := Sparkline([]float64{1, 2, 3, 4, 5}, &config)

	if result == "" {
		t.Error("expected non-empty result")
	}

	if len(result) == 0 {
		t.Error("expected sparkline content")
	}
}

func TestSparkline_FlatValues(t *testing.T) {
	config := DefaultSparklineConfig()
	config.Width = 5
	config.ShowCurrent = false
	result := Sparkline([]float64{5, 5, 5, 5, 5}, &config)

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestSparkline_ShowCurrent(t *testing.T) {
	config := DefaultSparklineConfig()
	config.Width = 5
	config.ShowCurrent = true
	result := Sparkline([]float64{1, 2, 3, 4, 100}, &config)

	if !strings.Contains(result, "100") {
		t.Errorf("expected current value in output, got: %s", result)
	}
}

func TestSparkline_ShowMinMax(t *testing.T) {
	config := DefaultSparklineConfig()
	config.Width = 5
	config.ShowMinMax = true
	config.ShowCurrent = false
	result := Sparkline([]float64{10, 20, 30, 40, 50}, &config)

	if !strings.Contains(result, "min:") || !strings.Contains(result, "max:") {
		t.Errorf("expected min/max labels, got: %s", result)
	}
}

func TestFindMinMax(t *testing.T) {
	testCases := []struct {
		name        string
		values      []float64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "ascending",
			values:      []float64{1, 2, 3, 4, 5},
			expectedMin: 1,
			expectedMax: 5,
		},
		{
			name:        "descending",
			values:      []float64{5, 4, 3, 2, 1},
			expectedMin: 1,
			expectedMax: 5,
		},
		{
			name:        "single value",
			values:      []float64{42},
			expectedMin: 42,
			expectedMax: 42,
		},
		{
			name:        "with negatives",
			values:      []float64{-10, 0, 10},
			expectedMin: -10,
			expectedMax: 10,
		},
		{
			name:        "all same",
			values:      []float64{7, 7, 7},
			expectedMin: 7,
			expectedMax: 7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			minVal, maxVal := findMinMax(tc.values)
			if minVal != tc.expectedMin {
				t.Errorf("expected min %f, got %f", tc.expectedMin, minVal)
			}
			if maxVal != tc.expectedMax {
				t.Errorf("expected max %f, got %f", tc.expectedMax, maxVal)
			}
		})
	}
}

func TestResampleValues(t *testing.T) {
	testCases := []struct {
		name        string
		values      []float64
		targetCount int
		checkLen    int
	}{
		{
			name:        "exact match",
			values:      []float64{1, 2, 3, 4, 5},
			targetCount: 5,
			checkLen:    5,
		},
		{
			name:        "pad shorter",
			values:      []float64{1, 2, 3},
			targetCount: 5,
			checkLen:    5,
		},
		{
			name:        "downsample longer",
			values:      []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			targetCount: 5,
			checkLen:    5,
		},
		{
			name:        "single value padded",
			values:      []float64{42},
			targetCount: 5,
			checkLen:    5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resampleValues(tc.values, tc.targetCount)
			if len(result) != tc.checkLen {
				t.Errorf("expected length %d, got %d", tc.checkLen, len(result))
			}
		})
	}
}

func TestResampleValues_PaddingUsesFirstValue(t *testing.T) {
	values := []float64{10, 20, 30}
	result := resampleValues(values, 5)

	if result[0] != 10 {
		t.Errorf("expected first padding value to be 10, got %f", result[0])
	}
	if result[1] != 10 {
		t.Errorf("expected second padding value to be 10, got %f", result[1])
	}
}

func TestFormatValue(t *testing.T) {
	testCases := []struct {
		name     string
		contains string
		value    float64
	}{
		{
			name:     "giga",
			value:    2.5e9,
			contains: "G",
		},
		{
			name:     "mega",
			value:    3.5e6,
			contains: "M",
		},
		{
			name:     "kilo",
			value:    1500,
			contains: "K",
		},
		{
			name:     "normal",
			value:    42.5,
			contains: "42.5",
		},
		{
			name:     "milli",
			value:    0.005,
			contains: "0.005",
		},
		{
			name:     "scientific",
			value:    0.00000001,
			contains: "e",
		},
		{
			name:     "negative kilo",
			value:    -2500,
			contains: "K",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatValue(tc.value)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected %q to contain %q", result, tc.contains)
			}
		})
	}
}

func TestSelectSparklineStyle(t *testing.T) {
	config := DefaultSparklineConfig()
	config.HighThreshold = new(80.0)
	config.LowThreshold = new(20.0)

	testCases := []struct {
		name          string
		expectedStyle string
		value         float64
	}{
		{
			name:          "above high threshold",
			value:         90,
			expectedStyle: "high",
		},
		{
			name:          "below low threshold",
			value:         10,
			expectedStyle: "low",
		},
		{
			name:          "in normal range",
			value:         50,
			expectedStyle: "normal",
		},
		{
			name:          "at high threshold boundary",
			value:         80,
			expectedStyle: "normal",
		},
		{
			name:          "at low threshold boundary",
			value:         20,
			expectedStyle: "normal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			style := selectSparklineStyle(tc.value, &config)

			_ = style.Render("x")
		})
	}
}

func TestSelectSparklineStyle_NilThresholds(t *testing.T) {
	style := selectSparklineStyle(100, new(DefaultSparklineConfig()))
	result := style.Render("x")
	if result == "" {
		t.Error("expected styled output")
	}
}

func TestMultilineSparkline_Empty(t *testing.T) {
	result := MultilineSparkline(nil, 10, 3, nil)
	if !strings.Contains(result, "─") {
		t.Errorf("empty multiline sparkline should contain dashes, got: %s", result)
	}
}

func TestMultilineSparkline_ZeroHeight(t *testing.T) {
	result := MultilineSparkline([]float64{1, 2, 3}, 10, 0, nil)
	if !strings.Contains(result, "─") {
		t.Errorf("zero height should return dashes, got: %s", result)
	}
}

func TestMultilineSparkline_ValidInput(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	result := MultilineSparkline(values, 5, 3, new(lipgloss.NewStyle().Foreground(lipgloss.Color("39"))))

	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestDefaultSparklineConfig(t *testing.T) {
	config := DefaultSparklineConfig()

	if config.Width != sparklineDefaultWidth {
		t.Errorf("expected default width %d, got %d", sparklineDefaultWidth, config.Width)
	}
	if config.Height != 1 {
		t.Errorf("expected default height 1, got %d", config.Height)
	}
	if config.ShowMinMax != false {
		t.Error("expected ShowMinMax to be false by default")
	}
	if config.ShowCurrent != true {
		t.Error("expected ShowCurrent to be true by default")
	}
	if config.HighThreshold != nil {
		t.Error("expected HighThreshold to be nil by default")
	}
	if config.LowThreshold != nil {
		t.Error("expected LowThreshold to be nil by default")
	}
}
