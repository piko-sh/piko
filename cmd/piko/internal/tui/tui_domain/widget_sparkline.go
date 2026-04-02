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
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// sparklineDefaultWidth is the default width in characters for sparkline widgets.
	sparklineDefaultWidth = 20

	// thresholdGiga is the threshold for formatting values with the "G" suffix.
	thresholdGiga = 1e9

	// thresholdMega is one million, used to format values with an "M" suffix.
	thresholdMega = 1e6

	// thresholdKilo is the value above which numbers are shown in thousands.
	thresholdKilo = 1e3

	// thresholdMilli is the threshold below which values are formatted in milli
	// notation.
	thresholdMilli = 0.001
)

// sparklineChars holds the characters from lowest to highest for sparklines.
// Uses Unicode block elements for smooth gradients.
var sparklineChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// SparklineConfig holds settings for rendering sparkline charts.
type SparklineConfig struct {
	// Style is the default style for sparkline bars.
	Style lipgloss.Style

	// HighStyle is the style used when a value is above the high threshold.
	HighStyle lipgloss.Style

	// LowStyle is the style used when the value falls below LowThreshold.
	LowStyle lipgloss.Style

	// HighThreshold is the value above which high-style formatting is applied;
	// nil means no high threshold check.
	HighThreshold *float64

	// LowThreshold is the value below which LowStyle is applied; nil disables it.
	LowThreshold *float64

	// Width is the number of characters for the sparkline display.
	Width int

	// Height specifies the vertical size in rows; 0 uses the default height.
	Height int

	// ShowMinMax enables display of minimum and maximum value labels.
	ShowMinMax bool

	// ShowCurrent appends the latest value after the sparkline when true.
	ShowCurrent bool
}

// DefaultSparklineConfig returns a SparklineConfig with sensible default
// values for rendering sparklines.
//
// Returns SparklineConfig which contains ready-to-use styles and settings.
func DefaultSparklineConfig() SparklineConfig {
	return SparklineConfig{
		Width:         sparklineDefaultWidth,
		Height:        1,
		ShowMinMax:    false,
		ShowCurrent:   true,
		Style:         lipgloss.NewStyle().Foreground(lipgloss.Color("39")),
		HighStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		LowStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		HighThreshold: nil,
		LowThreshold:  nil,
	}
}

// Sparkline renders a sparkline chart from the given data points.
//
// When values is empty, returns a horizontal line of the configured width.
//
// Takes values ([]float64) which contains the data points to display.
// Takes config (*SparklineConfig) which controls rendering options such as
// width, current value display, and min/max labels.
//
// Returns string which is the rendered sparkline with optional labels.
func Sparkline(values []float64, config *SparklineConfig) string {
	if len(values) == 0 {
		return strings.Repeat("─", config.Width)
	}

	sampled := resampleValues(values, config.Width)
	minVal, maxVal := findMinMax(sampled)
	valRange := maxVal - minVal
	if valRange == 0 {
		valRange = 1
	}

	spark := renderSparklineChars(sampled, minVal, valRange, config)
	result := spark

	if config.ShowCurrent && len(values) > 0 {
		result = fmt.Sprintf("%s %s", result, formatValue(values[len(values)-1]))
	}

	if config.ShowMinMax {
		result = addMinMaxLabels(result, minVal, maxVal)
	}

	return result
}

// MultilineSparkline renders a taller sparkline using multiple rows.
//
// Takes values ([]float64) which contains the data points to visualise.
// Takes width (int) which specifies the character width of the output.
// Takes height (int) which specifies the number of rows in the sparkline.
// Takes style (*lipgloss.Style) which defines the styling for filled blocks.
//
// Returns string which contains the rendered multiline sparkline joined by
// newlines. Returns a horizontal line when values is empty or height is less
// than one.
func MultilineSparkline(values []float64, width, height int, style *lipgloss.Style) string {
	if len(values) == 0 || height < 1 {
		return strings.Repeat("─", width)
	}

	sampled := resampleValues(values, width)

	minVal, maxVal := sampled[0], sampled[0]
	for _, v := range sampled {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	valRange := maxVal - minVal
	if valRange == 0 {
		valRange = 1
	}

	lines := make([]string, height)
	for row := range height {
		var line strings.Builder
		rowThreshold := float64(height-row) / float64(height)

		for _, v := range sampled {
			normalised := (v - minVal) / valRange
			if normalised >= rowThreshold {
				line.WriteString(style.Render("█"))
			} else if normalised >= rowThreshold-1.0/float64(height) {
				partialIndex := int((normalised - (rowThreshold - 1.0/float64(height))) * float64(height) * float64(len(sparklineChars)-1))
				partialIndex = max(0, min(partialIndex, len(sparklineChars)-1))
				line.WriteString(style.Render(string(sparklineChars[partialIndex])))
			} else {
				line.WriteString(" ")
			}
		}
		lines[row] = line.String()
	}

	return strings.Join(lines, "\n")
}

// findMinMax finds the smallest and largest values in a slice.
//
// Takes values ([]float64) which contains the numbers to check.
//
// Returns minVal (float64) which is the smallest value found.
// Returns maxVal (float64) which is the largest value found.
func findMinMax(values []float64) (minVal, maxVal float64) {
	minVal, maxVal = values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return minVal, maxVal
}

// renderSparklineChars builds the sparkline character string from sampled data.
//
// Takes sampled ([]float64) which contains the data points to render.
// Takes minVal (float64) which is the smallest value for scaling.
// Takes valRange (float64) which is the value range for scaling.
// Takes config (*SparklineConfig) which provides style settings.
//
// Returns string which is the styled sparkline character sequence.
func renderSparklineChars(sampled []float64, minVal, valRange float64, config *SparklineConfig) string {
	var spark strings.Builder
	for _, v := range sampled {
		normalised := (v - minVal) / valRange
		charIndex := int(normalised * float64(len(sparklineChars)-1))
		charIndex = max(0, min(charIndex, len(sparklineChars)-1))
		char := string(sparklineChars[charIndex])
		style := selectSparklineStyle(v, config)
		spark.WriteString(style.Render(char))
	}
	return spark.String()
}

// selectSparklineStyle picks the style based on value thresholds.
//
// Takes v (float64) which is the value to compare against thresholds.
// Takes config (*SparklineConfig) which holds the threshold values and styles.
//
// Returns lipgloss.Style which is the matching style for the value.
func selectSparklineStyle(v float64, config *SparklineConfig) lipgloss.Style {
	switch {
	case config.HighThreshold != nil && v > *config.HighThreshold:
		return config.HighStyle
	case config.LowThreshold != nil && v < *config.LowThreshold:
		return config.LowStyle
	default:
		return config.Style
	}
}

// addMinMaxLabels adds minimum and maximum value labels to a sparkline.
//
// Takes result (string) which is the sparkline content to label.
// Takes minVal (float64) which is the minimum value to show.
// Takes maxVal (float64) which is the maximum value to show.
//
// Returns string which is the sparkline with dimmed labels on each side.
func addMinMaxLabels(result string, minVal, maxVal float64) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	minString := dimStyle.Render(fmt.Sprintf("min:%.2f", minVal))
	maxString := dimStyle.Render(fmt.Sprintf("max:%.2f", maxVal))
	return fmt.Sprintf("%s %s %s", minString, result, maxString)
}

// resampleValues changes the size of a slice to match a target count.
//
// Takes values ([]float64) which holds the input data to resize.
// Takes targetCount (int) which sets the wanted output length.
//
// Returns []float64 which holds the resized values. When the input is shorter
// than the target, the first value fills the start. When longer, values are
// grouped and averaged.
func resampleValues(values []float64, targetCount int) []float64 {
	if len(values) <= targetCount {
		if len(values) == targetCount {
			return values
		}
		result := make([]float64, targetCount)
		padding := targetCount - len(values)
		for i := range padding {
			result[i] = values[0]
		}
		copy(result[padding:], values)
		return result
	}

	result := make([]float64, targetCount)
	ratio := float64(len(values)) / float64(targetCount)

	for i := range targetCount {
		start := int(float64(i) * ratio)
		end := min(len(values), int(float64(i+1)*ratio))

		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += values[j]
			count++
		}
		if count > 0 {
			result[i] = sum / float64(count)
		}
	}

	return result
}

// formatValue formats a float value for display with a unit suffix.
//
// Takes v (float64) which is the value to format.
//
// Returns string which contains the formatted value with a unit suffix (G, M,
// K) or in decimal or scientific notation based on size.
func formatValue(v float64) string {
	absV := math.Abs(v)
	switch {
	case absV >= thresholdGiga:
		return fmt.Sprintf("%.1fG", v/thresholdGiga)
	case absV >= thresholdMega:
		return fmt.Sprintf("%.1fM", v/thresholdMega)
	case absV >= thresholdKilo:
		return fmt.Sprintf("%.1fK", v/thresholdKilo)
	case absV >= 1:
		return fmt.Sprintf("%.1f", v)
	case absV >= thresholdMilli:
		return fmt.Sprintf("%.3f", v)
	default:
		return fmt.Sprintf("%.2e", v)
	}
}
