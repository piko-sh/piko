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

func TestCalculateColumnWidths_FixedOnly(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
		{Title: "Status", Width: 8},
		{Title: "Count", Width: 6},
	}

	widths := calculateColumnWidths(columns, 100)

	if widths[0] != 10 {
		t.Errorf("expected first column width 10, got %d", widths[0])
	}
	if widths[1] != 8 {
		t.Errorf("expected second column width 8, got %d", widths[1])
	}
	if widths[2] != 6 {
		t.Errorf("expected third column width 6, got %d", widths[2])
	}
}

func TestCalculateColumnWidths_FlexColumns(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
		{Title: "Description", Width: 0},
	}

	totalWidth := 50
	widths := calculateColumnWidths(columns, totalWidth)

	if widths[0] != 10 {
		t.Errorf("expected fixed column width 10, got %d", widths[0])
	}

	expectedFlex := totalWidth - 10 - 1
	if widths[1] != expectedFlex {
		t.Errorf("expected flex column width %d, got %d", expectedFlex, widths[1])
	}
}

func TestCalculateColumnWidths_MultipleFlexColumns(t *testing.T) {
	columns := []Column{
		{Title: "A", Width: 0},
		{Title: "B", Width: 0},
	}

	totalWidth := 41
	widths := calculateColumnWidths(columns, totalWidth)

	if widths[0] != widths[1] {
		t.Errorf("expected equal flex widths, got %d and %d", widths[0], widths[1])
	}
}

func TestCalculateColumnWidths_MinColumnWidth(t *testing.T) {
	columns := []Column{
		{Title: "A", Width: 0},
	}

	widths := calculateColumnWidths(columns, 2)

	if widths[0] < tableMinColumnWidth {
		t.Errorf("expected minimum width %d, got %d", tableMinColumnWidth, widths[0])
	}
}

func TestColumnsToRow(t *testing.T) {
	columns := []Column{
		{Title: "Name"},
		{Title: "Status"},
		{Title: "Count"},
	}

	row := columnsToRow(columns)

	if len(row) != 3 {
		t.Fatalf("expected 3 cells, got %d", len(row))
	}
	if row[0] != "Name" {
		t.Errorf("expected 'Name', got %q", row[0])
	}
	if row[1] != "Status" {
		t.Errorf("expected 'Status', got %q", row[1])
	}
	if row[2] != "Count" {
		t.Errorf("expected 'Count', got %q", row[2])
	}
}

func TestTruncateOrPad(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		width    int
	}{
		{
			name:     "exact length",
			input:    "hello",
			width:    5,
			expected: "hello",
		},
		{
			name:     "needs padding",
			input:    "hi",
			width:    5,
			expected: "hi   ",
		},
		{
			name:     "needs truncation with ellipsis",
			input:    "hello world",
			width:    8,
			expected: "hello...",
		},
		{
			name:     "very short width truncation",
			input:    "hello",
			width:    3,
			expected: "hel",
		},
		{
			name:     "empty string padded",
			input:    "",
			width:    3,
			expected: "   ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateOrPad(tc.input, tc.width)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestAlign(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		width    int
		position lipgloss.Position
	}{
		{
			name:     "left align",
			input:    "hi",
			width:    5,
			position: lipgloss.Left,
			expected: "hi   ",
		},
		{
			name:     "right align",
			input:    "hi",
			width:    5,
			position: lipgloss.Right,
			expected: "   hi",
		},
		{
			name:     "center align",
			input:    "hi",
			width:    6,
			position: lipgloss.Center,
			expected: "  hi  ",
		},
		{
			name:     "center align odd padding",
			input:    "hi",
			width:    5,
			position: lipgloss.Center,
			expected: " hi  ",
		},
		{
			name:     "no padding needed",
			input:    "hello",
			width:    5,
			position: lipgloss.Left,
			expected: "hello",
		},
		{
			name:     "width less than string",
			input:    "hello",
			width:    3,
			position: lipgloss.Right,
			expected: "hello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := align(tc.input, tc.width, tc.position)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTable_EmptyColumns(t *testing.T) {
	config := TableConfig{Columns: nil}
	result := Table(nil, 0, &config, 80)

	if result != "" {
		t.Errorf("expected empty result for no columns, got %q", result)
	}
}

func TestTable_WithHeader(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
		{Title: "Status", Width: 8},
	}
	rows := []Row{
		{"Alice", "Active"},
		{"Bob", "Inactive"},
	}

	result := Table(rows, 0, new(DefaultTableConfig(columns)), 80)

	if !strings.Contains(result, "Name") {
		t.Error("expected header to contain 'Name'")
	}
	if !strings.Contains(result, "Status") {
		t.Error("expected header to contain 'Status'")
	}
	if !strings.Contains(result, "Alice") {
		t.Error("expected table to contain 'Alice'")
	}
}

func TestTable_WithoutHeader(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
	}
	config := DefaultTableConfig(columns)
	config.ShowHeader = false
	rows := []Row{
		{"Alice"},
	}

	result := Table(rows, 0, &config, 80)

	lines := strings.Split(result, "\n")
	if len(lines) > 1 {

		t.Errorf("expected 1 line without header, got %d", len(lines))
	}
}

func TestTable_SelectedRow(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
	}
	rows := []Row{
		{"Alice"},
		{"Bob"},
		{"Charlie"},
	}

	result := Table(rows, 1, new(DefaultTableConfig(columns)), 80)

	if !strings.Contains(result, "Bob") {
		t.Error("expected table to contain selected row 'Bob'")
	}
}

func TestTable_AlternateRowStyle(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
	}
	config := DefaultTableConfig(columns)
	config.AlternateRowStyle = new(lipgloss.NewStyle().Foreground(lipgloss.Color("240")))
	rows := []Row{
		{"Alice"},
		{"Bob"},
		{"Charlie"},
	}

	result := Table(rows, -1, &config, 80)

	if result == "" {
		t.Error("expected non-empty table")
	}
}

func TestDefaultTableConfig(t *testing.T) {
	columns := []Column{
		{Title: "Test", Width: 10},
	}
	config := DefaultTableConfig(columns)

	if !config.ShowHeader {
		t.Error("expected ShowHeader to be true by default")
	}
	if len(config.Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(config.Columns))
	}
	if config.AlternateRowStyle != nil {
		t.Error("expected AlternateRowStyle to be nil by default")
	}
}

func TestTable_RowShorterThanColumns(t *testing.T) {
	columns := []Column{
		{Title: "A", Width: 5},
		{Title: "B", Width: 5},
		{Title: "C", Width: 5},
	}
	rows := []Row{
		{"X"},
	}

	result := Table(rows, 0, new(DefaultTableConfig(columns)), 80)

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestTable_EmptyRows(t *testing.T) {
	columns := []Column{
		{Title: "Name", Width: 10},
	}
	result := Table(nil, 0, new(DefaultTableConfig(columns)), 80)

	if !strings.Contains(result, "Name") {
		t.Error("expected header even with no rows")
	}
}
