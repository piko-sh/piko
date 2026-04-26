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
	"strconv"
	"strings"
)

const (
	// gaugeDefaultFill is the default filled-cell glyph drawn from the
	// Unicode block-drawing range.
	gaugeDefaultFill = '▰'

	// gaugeDefaultEmpty is the default empty-cell glyph drawn from the
	// Unicode block-drawing range.
	gaugeDefaultEmpty = '▱'

	// gaugeMinTextBarWidth is the minimum bar width before the trailing
	// "Used / Max P%" text is dropped to keep the meter readable.
	gaugeMinTextBarWidth = 4

	// gaugeMinReadableWidth is the minimum bar width below which the
	// label is also dropped, leaving only the meter glyphs.
	gaugeMinReadableWidth = 2

	// gaugeWarningThreshold is the percent above which severity becomes
	// warning.
	gaugeWarningThreshold = 0.6

	// gaugeCriticalThreshold is the percent above which severity becomes
	// critical.
	gaugeCriticalThreshold = 0.8

	// percentageScale converts a 0-1 fraction into a 0-100 integer
	// percentage for display.
	percentageScale = 100

	// emDashSymbol is the placeholder rendered when a gauge has no
	// configured maximum.
	emDashSymbol = "—"
)

// GaugeConfig configures a single horizontal gauge bar.
type GaugeConfig struct {
	// Theme provides the fill colour ramp. Required.
	Theme *Theme

	// Label is the optional text rendered alongside the bar. Empty
	// suppresses the leading label.
	Label string

	// Width is the total width of the bar in terminal cells. The bar
	// reserves chars for the label and value text and devotes the
	// remainder to the meter.
	Width int

	// Used is the consumed amount.
	Used float64

	// Max is the limit. When zero or negative the bar renders as
	// "--" without a meter.
	Max float64

	// Severity overrides the auto-derived severity band when set; pass
	// the zero value (SeverityHealthy) to use the auto-derived value.
	Severity Severity

	// FillChar is the character drawn for filled cells; defaults to a
	// solid block glyph.
	FillChar rune

	// EmptyChar is the character drawn for empty cells; defaults to a
	// shaded block glyph.
	EmptyChar rune

	// ShowText toggles the trailing text "Used / Max  P%" segment.
	ShowText bool
}

// Gauge renders the gauge bar according to config.
//
// Takes config (GaugeConfig) which configures the gauge.
//
// Returns string which is the rendered gauge sized to config.Width.
func Gauge(config GaugeConfig) string {
	if config.Width <= 0 {
		return ""
	}
	if config.FillChar == 0 {
		config.FillChar = gaugeDefaultFill
	}
	if config.EmptyChar == 0 {
		config.EmptyChar = gaugeDefaultEmpty
	}

	percent := gaugePercent(config.Used, config.Max)
	severity := config.Severity
	if severity == SeverityHealthy {
		severity = severityFromPercent(percent)
	}

	label := ""
	if config.Label != "" {
		label = config.Label + " "
	}
	labelWidth := TextWidth(label)

	text := ""
	if config.ShowText {
		text = " " + gaugeText(config.Used, config.Max, percent)
	}
	textWidth := TextWidth(text)

	barWidth := config.Width - labelWidth - textWidth
	if barWidth < gaugeMinTextBarWidth {
		text = ""
		textWidth = 0
		barWidth = config.Width - labelWidth
	}
	if barWidth < gaugeMinReadableWidth {
		return PadRightANSI(strings.Repeat(string(config.EmptyChar), config.Width), config.Width)
	}

	fillCells := max(0, min(barWidth, int(percent*float64(barWidth))))

	bar := strings.Repeat(string(config.FillChar), fillCells) + strings.Repeat(string(config.EmptyChar), barWidth-fillCells)
	if config.Theme != nil {
		bar = config.Theme.SeverityFor(severity).Render(bar)
	}

	body := label + bar + text
	return PadRightANSI(body, config.Width)
}

// gaugePercent returns the consumed fraction. Caps at 1.0 for rendering
// purposes; saturated values are still surfaced via Severity.
//
// Takes used (float64) and limit (float64) which describe the consumption.
//
// Returns float64 in [0, 1].
func gaugePercent(used, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	pct := used / limit
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	return pct
}

// severityFromPercent classifies a percentage into a Severity band.
// Mirrors UtilisationGauge.Severity so gauges built from raw used/max
// values get the same colour treatment as those computed by the provider.
//
// Takes percent (float64) which is the consumption fraction.
//
// Returns Severity which is the band the value falls into.
func severityFromPercent(percent float64) Severity {
	switch {
	case percent >= 1.0:
		return SeveritySaturated
	case percent >= gaugeCriticalThreshold:
		return SeverityCritical
	case percent >= gaugeWarningThreshold:
		return SeverityWarning
	default:
		return SeverityHealthy
	}
}

// gaugeText returns the trailing "Used / Max P%" string.
//
// Takes used (float64), limit (float64), percent (float64).
//
// Returns string which is the formatted text.
func gaugeText(used, limit, percent float64) string {
	if limit <= 0 {
		return emDashSymbol
	}
	return fmt.Sprintf("%s / %s  %d%%", trimFloat(used), trimFloat(limit), int(percent*percentageScale))
}

// trimFloat formats a float without unnecessary trailing zeros.
//
// Takes v (float64) which is the value to format.
//
// Returns string which is the trimmed representation.
func trimFloat(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return fmt.Sprintf("%.1f", v)
}
