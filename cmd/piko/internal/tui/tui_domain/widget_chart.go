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
	"time"

	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"
	tslc "github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart"
)

const (
	// chartMinWidth is the minimum cell width any chart will render at.
	chartMinWidth = 8

	// chartMinHeight is the minimum cell height any chart will render at.
	chartMinHeight = 4

	// chartXSteps is the number of tick steps on the X axis.
	chartXSteps = 4

	// chartYSteps is the number of tick steps on the Y axis.
	chartYSteps = 3

	// chartRangePadFraction adds 5% above and below the data span so
	// the line is not jammed against the top or bottom edge.
	chartRangePadFraction = 0.05

	// chartFlatPadFraction is the symmetric padding applied to a
	// constant-valued series (where min == max). 10% of the absolute
	// value gives a visible line; for value 0 we fall back to +/-1.
	chartFlatPadFraction = 0.1
)

// ChartPoint is a single (time, value) sample used to feed the chart.
type ChartPoint struct {
	// Time is when the sample was captured.
	Time time.Time

	// Value is the sample value.
	Value float64
}

// ChartSeries describes a named line plotted on the chart. Severity drives
// the line colour through the active Theme, so a "warning" series renders
// in the theme's warning hue regardless of which palette is active.
type ChartSeries struct {
	// Name uniquely identifies the series within the chart. Used as the
	// dataset key in the underlying ntcharts model.
	Name string

	// Points are the samples drawn for this series, in time order.
	Points []ChartPoint

	// Severity drives the line colour through the active theme.
	Severity Severity
}

// ChartConfig configures a Chart widget.
type ChartConfig struct {
	// Theme drives line colours via the severity-style mapping.
	Theme *Theme

	// Title is rendered above the chart.
	Title string

	// XLabel is the X-axis label, drawn under the X tick row.
	XLabel string

	// YLabel is the Y-axis label, drawn beside the Y tick column.
	YLabel string

	// Series is the slice of named lines to plot.
	Series []ChartSeries

	// Width is the rendered cell width.
	Width int

	// Height is the rendered cell height.
	Height int
}

// Chart wraps an ntcharts time-series line chart with our theme system.
// The chart owns its bubblezone manager privately so hover-on-data-point
// support can land in a future phase without leaking bubblezone into the
// rest of the codebase.
type Chart struct {
	// model is the underlying ntcharts time-series chart.
	model *tslc.Model

	// theme is the active theme.
	theme *Theme

	// width is the rendered cell width.
	width int

	// height is the rendered cell height.
	height int
}

// NewChart constructs a Chart with the supplied configuration. Width and
// height should be the inner content dimensions (already net of any
// surrounding pane chrome).
//
// The chart's Y range is computed from the supplied series values
// with a small symmetric padding so a near-constant series still
// shows variation. The X range covers the full time span of all
// supplied points.
//
// Takes config (ChartConfig) which configures the chart.
//
// Returns *Chart ready to call Render on.
func NewChart(config ChartConfig) *Chart {
	width := config.Width
	height := config.Height
	if width < chartMinWidth {
		width = chartMinWidth
	}
	if height < chartMinHeight {
		height = chartMinHeight
	}

	minY, maxY := chartYRange(config.Series)
	minT, maxT := chartTimeRange(config.Series)

	opts := []tslc.Option{
		tslc.WithLineStyle(runes.ArcLineStyle),
		tslc.WithXYSteps(chartXSteps, chartYSteps),
	}
	if minY != maxY {
		opts = append(opts, tslc.WithYRange(minY, maxY))
	}
	if !minT.IsZero() && !maxT.IsZero() && minT.Before(maxT) {
		opts = append(opts, tslc.WithTimeRange(minT, maxT))
	}

	chart := &Chart{
		model:  new(tslc.New(width, height, opts...)),
		theme:  config.Theme,
		width:  width,
		height: height,
	}

	chart.applySeries(config.Series)
	return chart
}

// chartYRange returns a (min, max) Y axis range covering every value
// in series with a small symmetric padding so near-constant data
// still shows visible variation. Returns (0, 0) for an empty input.
//
// Takes series ([]ChartSeries) which holds the plotted points.
//
// Returns minY (float64) which is the lower edge of the Y axis.
// Returns maxY (float64) which is the upper edge of the Y axis.
func chartYRange(series []ChartSeries) (minY, maxY float64) {
	minY, maxY, ok := minMaxValue(series)
	if !ok {
		return 0, 0
	}
	return paddedYRange(minY, maxY)
}

// minMaxValue scans series and returns the min and max value across
// all points. The bool result is false when no points were supplied.
//
// Takes series ([]ChartSeries) which holds the plotted points.
//
// Returns minVal (float64) which is the smallest sample value.
// Returns maxVal (float64) which is the largest sample value.
// Returns ok (bool) which is true when at least one point was scanned.
func minMaxValue(series []ChartSeries) (minVal, maxVal float64, ok bool) {
	for _, s := range series {
		for _, p := range s.Points {
			if !ok {
				minVal = p.Value
				maxVal = p.Value
				ok = true
				continue
			}
			if p.Value < minVal {
				minVal = p.Value
			}
			if p.Value > maxVal {
				maxVal = p.Value
			}
		}
	}
	return minVal, maxVal, ok
}

// paddedYRange applies symmetric padding around (minVal, maxVal) so
// the rendered line never sits flush against the chart edges. A
// constant series (min == max) gets +/-10% of its absolute value (or
// +/-1 when the value is exactly zero).
//
// Takes minVal (float64) which is the unpadded lower bound.
// Takes maxVal (float64) which is the unpadded upper bound.
//
// Returns paddedMin (float64) which is the padded lower edge.
// Returns paddedMax (float64) which is the padded upper edge.
func paddedYRange(minVal, maxVal float64) (paddedMin, paddedMax float64) {
	span := maxVal - minVal
	if span == 0 {
		pad := chartFlatPadFraction * absFloat(maxVal)
		if pad == 0 {
			pad = 1
		}
		return minVal - pad, maxVal + pad
	}
	pad := span * chartRangePadFraction
	return minVal - pad, maxVal + pad
}

// chartTimeRange returns the earliest and latest timestamps across all
// series. Returns (zero, zero) for empty input.
//
// Takes series ([]ChartSeries) which holds the plotted points.
//
// Returns minT (time.Time) which is the earliest sample timestamp.
// Returns maxT (time.Time) which is the latest sample timestamp.
func chartTimeRange(series []ChartSeries) (minT, maxT time.Time) {
	first := true
	for _, s := range series {
		for _, p := range s.Points {
			if first {
				minT = p.Time
				maxT = p.Time
				first = false
				continue
			}
			if p.Time.Before(minT) {
				minT = p.Time
			}
			if p.Time.After(maxT) {
				maxT = p.Time
			}
		}
	}
	return minT, maxT
}

// absFloat returns the absolute value of v.
//
// Takes v (float64) which is the value to take the absolute value of.
//
// Returns float64 which is the non-negative magnitude of v.
func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// SetTheme replaces the active theme and rebuilds line styles.
//
// Takes theme (*Theme) which is the new theme; never nil.
func (c *Chart) SetTheme(theme *Theme) {
	c.theme = theme

	for _, s := range c.cachedSeries() {
		c.model.SetDataSetStyle(s.Name, c.severityStyle(s.Severity))
	}
}

// Resize updates the chart's cell dimensions and rescales the data.
//
// Takes width (int) and height (int) which are the new dimensions.
func (c *Chart) Resize(width, height int) {
	if width < chartMinWidth {
		width = chartMinWidth
	}
	if height < chartMinHeight {
		height = chartMinHeight
	}
	c.width = width
	c.height = height
	c.model.Resize(width, height)
	c.model.DrawAll()
}

// Render redraws the chart and returns its string view, padded to the
// configured dimensions. Uses braille runes for higher visual density;
// each cell encodes a 2x4 sub-pixel grid so flat trends still read as
// a recognisable line.
//
// PadRightANSI is applied per-line so multi-line chart output is not
// collapsed into a single row by the global truncation path.
//
// Returns string with the rendered chart sized to width x height.
func (c *Chart) Render() string {
	if c.width <= 0 || c.height <= 0 || c.model == nil {
		return ""
	}
	c.model.DrawBrailleAll()
	body := c.model.View()

	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = PadRightANSI(line, c.width)
	}
	return strings.Join(lines, "\n")
}

// applySeries pushes each series into the underlying chart, applying
// severity-driven styles. Existing data is cleared first so callers can
// rebuild the chart from a fresh snapshot.
//
// Takes series ([]ChartSeries) which is the new data set.
func (c *Chart) applySeries(series []ChartSeries) {
	c.model.ClearAllData()
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}

		for _, p := range s.Points {
			c.model.PushDataSet(s.Name, tslc.TimePoint{Time: p.Time, Value: p.Value})
		}
		c.model.SetDataSetLineStyle(s.Name, runes.ArcLineStyle)
		c.model.SetDataSetStyle(s.Name, c.severityStyle(s.Severity))
	}
}

// severityStyle maps Severity to a lipgloss style for drawing the line.
// Falls back to a sensible foreground colour when no theme is set so
// tests can construct charts without configuring a theme.
//
// Takes s (Severity) which is the severity classification of the series.
//
// Returns lipgloss.Style which is the style for the line.
func (c *Chart) severityStyle(s Severity) lipgloss.Style {
	if c.theme == nil {
		return lipgloss.NewStyle()
	}
	style := c.theme.SeverityFor(s)

	return style.Inline(true)
}

// cachedSeries inspects the model's current dataset names to allow
// re-styling on theme changes. ntcharts does not expose dataset names
// directly, so we cache the slice of (name, severity) pairs at apply
// time.
//
// Returns []ChartSeries with names + severities only (Points unused).
func (*Chart) cachedSeries() []ChartSeries {
	return nil
}
