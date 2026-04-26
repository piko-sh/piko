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
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"piko.sh/piko/cmd/piko/internal/inspector"
)

const (
	// detailKeyMinWidth is the minimum cell width allocated to the
	// key column in a key/value detail row.
	detailKeyMinWidth = 12

	// detailKeyMaxWidth is the maximum cell width the key column may
	// occupy before values start truncating.
	detailKeyMaxWidth = 24

	// detailRowGutter is the horizontal padding between the key and
	// value columns plus the leading indent.
	detailRowGutter = 4

	// detailMinRows is the smallest preallocated capacity for rendered
	// row slices.
	detailMinRows = 8

	// chartMinDetailHeight is the smallest allowed chart-area height in
	// a detail pane; below this the chart is suppressed and the body
	// uses the full pane.
	chartMinDetailHeight = 6

	// chartMaxDetailHeight caps the chart area so the body always has
	// room for the structured rows above it.
	chartMaxDetailHeight = 16

	// chartHeightDivisor controls how much of the pane is allocated to
	// the chart by default (1/3 of total height).
	chartHeightDivisor = 3

	// chartMinTotalHeight is the minimum pane height before the chart
	// is even considered.
	chartMinTotalHeight = 18

	// chartHeaderRows is the number of rows reserved between the body
	// and the chart for the title and a blank gap.
	chartHeaderRows = 2
)

// RenderDetailBody renders a inspector.DetailBody at (width, height). Long values
// are truncated to fit; the body is right-padded and clipped to the
// supplied dimensions so the composer's frame line up cleanly.
//
// Takes theme (*Theme) which provides title / dim / bold styles. May
// be nil during tests.
// Takes body (inspector.DetailBody) which is the structured content.
// Takes width (int) and height (int) which size the rendered body.
//
// Returns string with the rendered body sized to width x height.
func RenderDetailBody(theme *Theme, body inspector.DetailBody, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	rows := make([]string, 0, max(detailMinRows, height))

	if body.Title != "" {
		titleStyle := detailTitleStyle(theme)
		rows = append(rows, paddedLine(&titleStyle, body.Title, width))
	}
	if body.Subtitle != "" {
		dimStyle := detailDimStyle(theme)
		rows = append(rows, paddedLine(&dimStyle, body.Subtitle, width))
	}
	if body.Title != "" || body.Subtitle != "" {
		rows = append(rows, strings.Repeat(SingleSpace, width))
	}

	keyWidth := detailKeyWidth(body)
	first := true
	for _, section := range body.Sections {
		if !first {
			rows = append(rows, strings.Repeat(SingleSpace, width))
		}
		first = false
		if section.Heading != "" {
			dimStyle := detailDimStyle(theme)
			rows = append(rows, paddedLine(&dimStyle, strings.ToUpper(section.Heading), width))
		}
		for _, r := range section.Rows {
			rows = append(rows, renderDetailKeyValueWrapped(theme, r.Label, r.Value, keyWidth, width)...)
		}
	}

	return clipDetailBody(rows, width, height)
}

// renderDetailKeyValueWrapped renders a key/value row, wrapping the
// value across multiple continuation rows when it does not fit in the
// value column. The first row shows "  KEY        VALUE..."; each
// continuation row indents to align with the value column.
//
// Takes theme (*Theme) which provides dim and bold styles; may be nil.
// Takes key (string) which is the row label.
// Takes value (string) which is the wrapped value text.
// Takes keyWidth (int) which is the key column width.
// Takes width (int) which is the total row width.
//
// Returns []string, one entry per rendered row (always at least one).
func renderDetailKeyValueWrapped(theme *Theme, key, value string, keyWidth, width int) []string {
	maxValueWidth := max(1, width-keyWidth-detailRowGutter)
	indent := strings.Repeat(SingleSpace, keyWidth+detailRowGutter)

	chunks := wrapByWidth(value, maxValueWidth)
	if len(chunks) == 0 {
		chunks = []string{""}
	}

	keyText := PadRightANSI(detailDimStyle(theme).Render(key), keyWidth)

	out := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		styledValue := detailBoldStyle(theme).Render(chunk)
		var row string
		if i == 0 {
			row = "  " + keyText + " " + styledValue
		} else {
			row = indent + styledValue
		}
		out = append(out, PadRightANSI(row, width))
	}
	return out
}

// wrapByWidth splits s into chunks each at most maxWidth cells wide.
//
// Wraps at word boundaries when possible; falls back to mid-word
// breaks when a single token exceeds maxWidth so the column never
// overflows. Empty input returns an empty slice.
//
// Takes s (string) which is the unstyled value to wrap.
// Takes maxWidth (int) which is the per-line width budget.
//
// Returns []string with one chunk per output line.
func wrapByWidth(s string, maxWidth int) []string {
	if s == "" {
		return nil
	}
	if maxWidth <= 0 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}

	out := []string{}
	current := ""
	for _, word := range words {
		out, current = wrapAddWord(out, current, word, maxWidth)
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

// wrapAddWord folds word into the wrap accumulator, returning the new
// completed-lines slice and the in-progress current line. Long words
// that exceed maxWidth on their own are split mid-word so the column
// never overflows.
//
// Takes out ([]string), current (string), word (string), maxWidth (int).
//
// Returns ([]string, string) updated completed-lines and current
// in-progress line.
func wrapAddWord(out []string, current, word string, maxWidth int) ([]string, string) {
	if TextWidth(word) > maxWidth {
		if current != "" {
			out = append(out, current)
		}
		out, word = breakOversizeWord(out, word, maxWidth)
		return out, word
	}
	candidate := word
	if current != "" {
		candidate = current + " " + word
	}
	if TextWidth(candidate) > maxWidth {
		out = append(out, current)
		return out, word
	}
	return out, candidate
}

// breakOversizeWord splits word into rune-aligned chunks of at most maxWidth.
//
// Appends the leading chunks onto out and returns the trailing partial
// chunk. A single rune wider than maxWidth is emitted on its own line
// so the loop always makes progress even when one glyph overflows.
//
// Takes out ([]string), word (string), maxWidth (int).
//
// Returns ([]string, string) updated completed-lines slice and the
// trailing partial chunk.
func breakOversizeWord(out []string, word string, maxWidth int) ([]string, string) {
	for TextWidth(word) > maxWidth {
		head, tail := splitByCellWidth(word, maxWidth)
		if head == "" {
			_, size := utf8.DecodeRuneInString(word)
			if size == 0 {
				return out, ""
			}
			head, tail = word[:size], word[size:]
		}
		out = append(out, head)
		word = tail
	}
	return out, word
}

// splitByCellWidth returns the longest rune-aligned prefix of s whose
// cell width fits maxWidth, plus the remainder.
//
// Walks runes so multi-byte characters (East-Asian glyphs, emoji) are
// never cut mid-byte. Returns ("", s) when even the first rune is
// wider than maxWidth, leaving it to the caller to make progress.
//
// Takes s (string) which is the unstyled text to split.
// Takes maxWidth (int) which is the cell-width budget for the prefix.
//
// Returns prefix (string) which is the rune-aligned head whose cell
// width fits within maxWidth.
// Returns remainder (string) which is the unconsumed tail.
func splitByCellWidth(s string, maxWidth int) (prefix, remainder string) {
	if maxWidth <= 0 || s == "" {
		return "", s
	}
	width := 0
	cut := 0
	for i, r := range s {
		rw := lipgloss.Width(string(r))
		if width+rw > maxWidth {
			return s[:cut], s[i:]
		}
		width += rw
		cut = i + utf8.RuneLen(r)
	}
	return s, ""
}

// detailKeyWidth picks a key-column width based on the longest label
// across all sections, capped at detailKeyMaxWidth.
//
// Takes body (inspector.DetailBody) which is the structured detail body.
//
// Returns int which is the chosen key column width in cells.
func detailKeyWidth(body inspector.DetailBody) int {
	keyWidth := detailKeyMinWidth
	for _, section := range body.Sections {
		for _, r := range section.Rows {
			if w := TextWidth(r.Label); w > keyWidth {
				keyWidth = w
			}
		}
	}
	return min(keyWidth, detailKeyMaxWidth)
}

// paddedLine renders text in style and pads/clips to width.
//
// Accepts a pointer to lipgloss.Style because the underlying struct
// is large (~648 bytes); copying it on every detail-row render adds
// avoidable allocation pressure.
//
// Takes style (*lipgloss.Style) which styles the text; nil renders raw.
// Takes text (string) which is the content.
// Takes width (int) which is the cell width to pad/clip to.
//
// Returns string of the styled, padded text.
func paddedLine(style *lipgloss.Style, text string, width int) string {
	if style == nil {
		return PadRightANSI(text, width)
	}
	return PadRightANSI(style.Render(text), width)
}

// clipDetailBody pads or truncates rows to exactly height rows of
// width cells.
//
// Takes rows ([]string) which are the rendered rows.
// Takes width (int) which is the per-row cell width.
// Takes height (int) which is the target row count.
//
// Returns string of the joined, sized body.
func clipDetailBody(rows []string, width, height int) string {
	for len(rows) < height {
		rows = append(rows, strings.Repeat(SingleSpace, width))
	}
	if len(rows) > height {
		rows = rows[:height]
	}
	return strings.Join(rows, "\n")
}

// detailTitleStyle returns the active theme's panel-title style or a
// safe fallback for nil themes.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns lipgloss.Style which is the title style.
func detailTitleStyle(theme *Theme) lipgloss.Style {
	if theme != nil {
		return theme.PanelTitle
	}
	return panelTitleStyle
}

// detailDimStyle returns the active theme's dim/subtle style or a
// safe fallback for nil themes.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns lipgloss.Style which is the dim style.
func detailDimStyle(theme *Theme) lipgloss.Style {
	if theme != nil {
		return theme.Dim
	}
	return navItemStyle
}

// detailBoldStyle returns the active theme's bold style or a safe
// fallback for nil themes.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns lipgloss.Style which is the bold style.
func detailBoldStyle(theme *Theme) lipgloss.Style {
	if theme != nil {
		return theme.Bold
	}
	return navItemStyle
}

// RenderDetailBodyWithChart renders body in the top portion of (width,
// height) and a high-fidelity chart in the bottom portion. The chart
// receives at least chartMinDetailHeight rows when there is enough
// vertical room; smaller detail panes fall back to body-only.
//
// Takes theme (*Theme) for styling.
// Takes body (inspector.DetailBody) which is the structured content.
// Takes series ([]ChartSeries) which feeds the chart.
// Takes chartTitle (string) which is rendered above the chart.
// Takes width (int) and height (int) for the inner content.
//
// Returns string with body + chart joined vertically.
func RenderDetailBodyWithChart(theme *Theme, body inspector.DetailBody, series []ChartSeries, chartTitle string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	chartHeight := computeDetailChartHeight(height)
	if chartHeight <= 0 || len(series) == 0 {
		return RenderDetailBody(theme, body, width, height)
	}

	bodyHeight := height - chartHeight - chartHeaderRows
	bodyHeight = max(bodyHeight, detailMinRows)
	bodyView := RenderDetailBody(theme, body, width, bodyHeight)

	gap := strings.Repeat(SingleSpace, width)
	header := paddedLine(themeChartTitleStyle(theme), strings.ToUpper(chartTitle), width)

	chart := NewChart(ChartConfig{
		Theme:  theme,
		Series: series,
		Width:  width,
		Height: chartHeight,
	})
	chartView := chart.Render()

	return strings.Join([]string{bodyView, gap, header, chartView}, "\n")
}

// themeChartTitleStyle returns a non-nil pointer to the title style.
//
// Takes theme (*Theme) which is the active theme; may be nil.
//
// Returns *lipgloss.Style pointing at the chart title style.
func themeChartTitleStyle(theme *Theme) *lipgloss.Style {
	style := detailDimStyle(theme)
	return &style
}

// computeDetailChartHeight returns the rows allocated to the chart, or
// 0 when the pane is too short to show one usefully.
//
// Takes height (int) which is the total inner height.
//
// Returns int chart-area height; 0 means "no chart; body only".
func computeDetailChartHeight(height int) int {
	if height < chartMinTotalHeight {
		return 0
	}
	candidate := height / chartHeightDivisor
	if candidate < chartMinDetailHeight {
		return 0
	}
	return min(candidate, chartMaxDetailHeight)
}
