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

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanWidth(t *testing.T) {
	tests := []struct {
		name          string
		column_widths []float64
		start_column  int
		colspan       int
		spacing       float64
		expected      float64
	}{
		{
			"single column returns its width",
			[]float64{100, 200, 300},
			0, 1, 0,
			100,
		},
		{
			"multi-column span includes intermediate spacing",
			[]float64{100, 200, 300},
			0, 2, 10,

			310,
		},
		{
			"all columns with spacing",
			[]float64{100, 200, 300},
			0, 3, 5,

			610,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := spanWidth(tt.column_widths, tt.start_column, tt.colspan, tt.spacing)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestSpanHeight(t *testing.T) {
	tests := []struct {
		name        string
		row_heights []float64
		start_row   int
		rowspan     int
		spacing     float64
		expected    float64
	}{
		{
			"single row returns its height",
			[]float64{50, 100, 150},
			0, 1, 0,
			50,
		},
		{
			"multi-row span includes intermediate spacing",
			[]float64{50, 100, 150},
			0, 2, 10,

			160,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := spanHeight(tt.row_heights, tt.start_row, tt.rowspan, tt.spacing)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestBuildTableGrid(t *testing.T) {
	tests := []struct {
		name             string
		rows             []tableRow
		expected_count   int
		expected_rows    int
		expected_columns int
	}{
		{
			"simple 2x2 table",
			[]tableRow{
				{cells: []*LayoutBox{
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
				}},
				{cells: []*LayoutBox{
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
				}},
			},
			4, 2, 2,
		},
		{
			"cell with colspan occupies multiple columns",
			[]tableRow{
				{cells: []*LayoutBox{
					{Type: BoxTableCell, Colspan: 2, Rowspan: 1},
				}},
				{cells: []*LayoutBox{
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
				}},
			},
			3, 2, 2,
		},
		{
			"cell with rowspan shifts subsequent row cells",
			[]tableRow{
				{cells: []*LayoutBox{
					{Type: BoxTableCell, Colspan: 1, Rowspan: 2},
					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
				}},
				{cells: []*LayoutBox{

					{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
				}},
			},
			3, 2, 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placements, row_count, column_count := buildTableGrid(tt.rows)
			assert.Equal(t, tt.expected_count, len(placements))
			assert.Equal(t, tt.expected_rows, row_count)
			assert.Equal(t, tt.expected_columns, column_count)
		})
	}

	t.Run("rowspan placement positions are correct", func(t *testing.T) {
		rows := []tableRow{
			{cells: []*LayoutBox{
				{Type: BoxTableCell, Colspan: 1, Rowspan: 2},
				{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
			}},
			{cells: []*LayoutBox{
				{Type: BoxTableCell, Colspan: 1, Rowspan: 1},
			}},
		}

		placements, _, _ := buildTableGrid(rows)

		assert.Equal(t, 0, placements[0].column)
		assert.Equal(t, 0, placements[0].row)
		assert.Equal(t, 1, placements[0].colspan)
		assert.Equal(t, 2, placements[0].rowspan)

		assert.Equal(t, 1, placements[1].column)
		assert.Equal(t, 0, placements[1].row)

		assert.Equal(t, 1, placements[2].column)
		assert.Equal(t, 1, placements[2].row)
	})
}

func TestComputeColumnXOffsets(t *testing.T) {
	tests := []struct {
		name           string
		column_widths  []float64
		content_startx float64
		spacing        float64
		expected       []float64
	}{
		{
			"three columns with spacing and start offset",
			[]float64{100, 200, 300},
			50, 10,

			[]float64{60, 170, 380},
		},
		{
			"single column with zero start",
			[]float64{100},
			0, 5,

			[]float64{5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeColumnXOffsets(tt.column_widths, tt.content_startx, tt.spacing)
			assert.Equal(t, len(tt.expected), len(result))
			for index := range tt.expected {
				assert.InDelta(t, tt.expected[index], result[index], 0.001)
			}
		})
	}
}

func TestComputeRowYOffsets(t *testing.T) {
	tests := []struct {
		name        string
		row_heights []float64
		start_y     float64
		spacing     float64
		expected    []float64
	}{
		{
			"three rows with spacing",
			[]float64{50, 100, 150},
			20, 5,

			[]float64{20, 75, 180},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeRowYOffsets(tt.row_heights, tt.start_y, tt.spacing)
			assert.Equal(t, len(tt.expected), len(result))
			for index := range tt.expected {
				assert.InDelta(t, tt.expected[index], result[index], 0.001)
			}
		})
	}
}

func TestDistributeColumnWidths(t *testing.T) {
	tests := []struct {
		name            string
		min_widths      []float64
		preferred       []float64
		column_count    int
		available_width float64
		expected        []float64
	}{
		{
			"preferred widths fit within available space and are scaled proportionally",
			[]float64{50, 50, 50},
			[]float64{100, 100, 100},
			3, 600,

			[]float64{200, 200, 200},
		},
		{
			"preferred widths exceed available width uses gap distribution",
			[]float64{50, 80, 70},
			[]float64{150, 200, 150},
			3, 400,

			[]float64{116.667, 160.0, 123.333},
		},
		{
			"zero preferred widths distributes equally",
			[]float64{0, 0, 0},
			[]float64{0, 0, 0},
			3, 300,
			[]float64{100, 100, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := distributeColumnWidths(tt.min_widths, tt.preferred, tt.column_count, tt.available_width)
			assert.Equal(t, len(tt.expected), len(result))
			for index := range tt.expected {
				assert.InDelta(t, tt.expected[index], result[index], 0.001)
			}
		})
	}
}

func TestComputeFixedColumnWidths(t *testing.T) {
	tests := []struct {
		name         string
		column_count int
		table_width  float64
		spacing      float64
		expected     float64
	}{
		{
			"three columns with spacing",
			3, 300, 10,

			86.667,
		},
		{
			"single column with zero spacing",
			1, 500, 0,

			500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeFixedColumnWidths(tt.column_count, tt.table_width, tt.spacing)
			assert.Equal(t, tt.column_count, len(result))
			for _, width := range result {
				assert.InDelta(t, tt.expected, width, 0.001)
			}
		})
	}
}

func TestApplyTableCellVerticalAlignFragment(t *testing.T) {
	tests := []struct {
		name             string
		vertical_align   VerticalAlignType
		content_height   float64
		available_height float64
		expected_offset  float64
	}{
		{
			"baseline alignment does not adjust offset",
			VerticalAlignBaseline,
			40, 100,
			0,
		},
		{
			"top alignment does not adjust offset",
			VerticalAlignTop,
			40, 100,
			0,
		},
		{
			"middle alignment centres the cell vertically",
			VerticalAlignMiddle,
			40, 100,

			30,
		},
		{
			"bottom alignment moves the cell to the bottom",
			VerticalAlignBottom,
			40, 100,

			60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := &LayoutBox{
				Type: BoxTableCell,
				Style: ComputedStyle{
					VerticalAlign: tt.vertical_align,
				},
			}
			fragment := &Fragment{
				Box:           box,
				ContentHeight: tt.content_height,
				OffsetY:       0,
			}

			applyTableCellVerticalAlignFragment(fragment, tt.available_height)
			assert.InDelta(t, tt.expected_offset, fragment.OffsetY, 0.001)
		})
	}
}
