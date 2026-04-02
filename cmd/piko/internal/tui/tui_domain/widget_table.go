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

	"charm.land/lipgloss/v2"
)

const (
	// tableMinColumnWidth is the smallest width in characters for table columns.
	tableMinColumnWidth = 5
)

// Column defines a table column with its display style, title, width, and
// text alignment.
type Column struct {
	// Style sets the column styling; if empty, the base style is used.
	Style lipgloss.Style

	// Title is the column header text shown in the table.
	Title string

	// Width is the fixed column width in characters; 0 means flexible width.
	Width int

	// Align specifies the horizontal alignment of cell content.
	Align lipgloss.Position
}

// Row represents a single table row as a slice of string values.
type Row []string

// TableConfig holds the display settings for a table.
type TableConfig struct {
	// HeaderStyle sets the look of table header cells.
	HeaderStyle lipgloss.Style

	// RowStyle is the default style applied to all table rows.
	RowStyle lipgloss.Style

	// SelectedRowStyle is the style used for the currently selected row.
	SelectedRowStyle lipgloss.Style

	// BorderStyle sets the look of table borders.
	BorderStyle lipgloss.Style

	// AlternateRowStyle is the style applied to odd-numbered rows; nil disables
	// alternate row styling.
	AlternateRowStyle *lipgloss.Style

	// Columns defines the column headers and their display settings.
	Columns []Column

	// ShowHeader controls whether the table header row is shown.
	ShowHeader bool
}

// DefaultTableConfig returns a TableConfig with sensible default styling.
//
// Takes columns ([]Column) which defines the table column layout.
//
// Returns TableConfig which is ready for use with styled headers and rows.
func DefaultTableConfig(columns []Column) TableConfig {
	return TableConfig{
		HeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")),
		RowStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		SelectedRowStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("235")).
			Bold(true),
		BorderStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("238")),
		AlternateRowStyle: nil,
		Columns:           columns,
		ShowHeader:        true,
	}
}

// Table renders a data table with optional header and row selection.
//
// Takes rows ([]Row) which contains the data to display in the table.
// Takes selectedIndex (int) which specifies which row to highlight, or -1 for
// no selection.
// Takes config (*TableConfig) which defines columns, styles, and display
// options.
// Takes width (int) which sets the total available width for the table.
//
// Returns string which contains the rendered table ready for display.
func Table(rows []Row, selectedIndex int, config *TableConfig, width int) string {
	if len(config.Columns) == 0 {
		return ""
	}

	widths := calculateColumnWidths(config.Columns, width)

	var result strings.Builder

	if config.ShowHeader {
		header := renderTableRow(
			columnsToRow(config.Columns),
			widths,
			config.Columns,
			&config.HeaderStyle,
		)
		result.WriteString(header)
		result.WriteString("\n")
	}

	for i := range rows {
		style := config.RowStyle

		if config.AlternateRowStyle != nil && i%2 == 1 {
			style = *config.AlternateRowStyle
		}

		if i == selectedIndex {
			style = config.SelectedRowStyle
		}

		line := renderTableRow(rows[i], widths, config.Columns, &style)
		result.WriteString(line)
		if i < len(rows)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderTableRow builds a single table row with styled and aligned cells.
//
// Takes cells (Row) which holds the cell values for this row.
// Takes widths ([]int) which sets the width of each column.
// Takes columns ([]Column) which gives column settings for styling and
// alignment.
// Takes baseStyle (*lipgloss.Style) which is the default style to use.
//
// Returns string which is the formatted row with cells joined by spaces.
func renderTableRow(cells Row, widths []int, columns []Column, baseStyle *lipgloss.Style) string {
	parts := make([]string, 0, len(widths))

	for i, width := range widths {
		var cell string
		if i < len(cells) {
			cell = cells[i]
		}

		style := *baseStyle
		if i < len(columns) && columns[i].Style.Value() != "" {
			style = columns[i].Style.Inherit(*baseStyle)
		}

		cell = truncateOrPad(cell, width)

		if i < len(columns) {
			cell = align(cell, width, columns[i].Align)
		}

		parts = append(parts, style.Render(cell))
	}

	return strings.Join(parts, " ")
}

// calculateColumnWidths works out the width for each column.
//
// Takes columns ([]Column) which defines the column settings.
// Takes totalWidth (int) which sets the total space available.
//
// Returns []int which holds the worked out width for each column.
func calculateColumnWidths(columns []Column, totalWidth int) []int {
	widths := make([]int, len(columns))
	fixedWidth := 0
	flexCount := 0

	for i := range columns {
		if columns[i].Width > 0 {
			widths[i] = columns[i].Width
			fixedWidth += columns[i].Width
		} else {
			flexCount++
		}
	}

	separatorWidth := len(columns) - 1
	remaining := totalWidth - fixedWidth - separatorWidth

	if flexCount > 0 && remaining > 0 {
		flexWidth := remaining / flexCount
		for i := range columns {
			if columns[i].Width == 0 {
				widths[i] = max(tableMinColumnWidth, flexWidth)
			}
		}
	}

	return widths
}

// columnsToRow converts a slice of columns into a row of their titles.
//
// Takes columns ([]Column) which contains the columns to extract titles from.
//
// Returns Row which holds the title from each column.
func columnsToRow(columns []Column) Row {
	row := make(Row, len(columns))
	for i := range columns {
		row[i] = columns[i].Title
	}
	return row
}

// truncateOrPad adjusts a string to match the given width.
//
// Takes s (string) which is the input string to adjust.
// Takes width (int) which specifies the target width in characters.
//
// Returns string which is the adjusted string, cut short with an ellipsis if
// too long or padded with spaces if too short.
func truncateOrPad(s string, width int) string {
	if len(s) > width {
		if width <= ellipsisLength {
			return s[:width]
		}
		return s[:width-ellipsisLength] + ellipsis
	}
	return s + strings.Repeat(stringSpace, width-len(s))
}

// align pads text to fit a given width using the chosen alignment.
//
// Takes s (string) which is the text to align.
// Takes width (int) which is the total width to pad to.
// Takes position (lipgloss.Position) which sets left, right, or centre alignment.
//
// Returns string which is the text padded to the given width.
func align(s string, width int, position lipgloss.Position) string {
	padding := width - len(s)
	if padding <= 0 {
		return s
	}

	switch position {
	case lipgloss.Right:
		return strings.Repeat(stringSpace, padding) + s
	case lipgloss.Center:
		left := padding / 2
		right := padding - left
		return strings.Repeat(stringSpace, left) + s + strings.Repeat(stringSpace, right)
	default:
		return s + strings.Repeat(stringSpace, padding)
	}
}
