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

const columnEpsilon = 0.001

func TestResolveColumnDimensions(t *testing.T) {
	tests := []struct {
		name            string
		column_count    int
		column_width    Dimension
		gap             float64
		available_width float64
		expected_count  int
		expected_width  float64
	}{
		{
			name:            "count only divides available width minus gaps",
			column_count:    3,
			column_width:    DimensionAuto(),
			gap:             10,
			available_width: 500,
			expected_count:  3,

			expected_width: 160,
		},
		{
			name:            "width only computes count from available space",
			column_count:    0,
			column_width:    DimensionPt(150),
			gap:             10,
			available_width: 500,

			expected_count: 3,
			expected_width: 160,
		},
		{
			name:            "both set uses minimum of count and max fitting by width",
			column_count:    2,
			column_width:    DimensionPt(200),
			gap:             10,
			available_width: 500,

			expected_count: 2,
			expected_width: 245,
		},
		{
			name:            "both set with count lower than max by width uses count",
			column_count:    2,
			column_width:    DimensionPt(100),
			gap:             10,
			available_width: 500,

			expected_count: 2,
			expected_width: 245,
		},
		{
			name:            "neither set defaults to single column at full width",
			column_count:    0,
			column_width:    DimensionAuto(),
			gap:             10,
			available_width: 500,
			expected_count:  1,
			expected_width:  500,
		},
		{
			name:            "count of 1 returns full width with no gaps",
			column_count:    1,
			column_width:    DimensionAuto(),
			gap:             10,
			available_width: 500,

			expected_count: 1,
			expected_width: 500,
		},
		{
			name:            "zero gap with count distributes width evenly",
			column_count:    4,
			column_width:    DimensionAuto(),
			gap:             0,
			available_width: 400,
			expected_count:  4,
			expected_width:  100,
		},
		{
			name:            "width larger than available space yields single column",
			column_count:    0,
			column_width:    DimensionPt(600),
			gap:             10,
			available_width: 500,

			expected_count: 1,
			expected_width: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result_count, result_width := resolveColumnDimensions(
				tt.column_count, tt.column_width, tt.gap, tt.available_width,
			)
			assert.Equal(t, tt.expected_count, result_count, "column count")
			assert.InDelta(t, tt.expected_width, result_width, columnEpsilon, "column width")
		})
	}
}

func TestResolveColumnHeight(t *testing.T) {
	tests := []struct {
		style                *ComputedStyle
		name                 string
		total_content_height float64
		column_count         int
		fill                 ColumnFillType
		available_block_size float64
		expected             float64
	}{
		{
			name:                 "balanced mode divides content evenly across columns",
			total_content_height: 300,
			column_count:         3,
			fill:                 ColumnFillBalance,
			available_block_size: 0,
			style:                &ComputedStyle{Height: DimensionAuto()},

			expected: 100,
		},
		{
			name:                 "auto fill with available block size uses that height",
			total_content_height: 300,
			column_count:         3,
			fill:                 ColumnFillAuto,
			available_block_size: 250,
			style:                &ComputedStyle{Height: DimensionAuto()},
			expected:             250,
		},
		{
			name:                 "auto fill without available block size falls back to balanced",
			total_content_height: 300,
			column_count:         3,
			fill:                 ColumnFillAuto,
			available_block_size: 0,
			style:                &ComputedStyle{Height: DimensionAuto()},

			expected: 100,
		},
		{
			name:                 "balanced mode with very small content clamps to zero",
			total_content_height: 1.5,
			column_count:         3,
			fill:                 ColumnFillBalance,
			available_block_size: 0,
			style:                &ComputedStyle{Height: DimensionAuto()},

			expected: 0,
		},
		{
			name:                 "explicit height on style overrides balancing",
			total_content_height: 300,
			column_count:         3,
			fill:                 ColumnFillBalance,
			available_block_size: 0,
			style: &ComputedStyle{
				Height:    DimensionPt(150),
				BoxSizing: BoxSizingContentBox,
			},
			expected: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveColumnHeight(
				tt.total_content_height, tt.column_count,
				tt.fill, tt.available_block_size, tt.style,
			)
			assert.InDelta(t, tt.expected, result, columnEpsilon)
		})
	}
}

func TestPositionColumns(t *testing.T) {
	tests := []struct {
		name             string
		columns          [][]*Fragment
		expected_offsets []float64
		column_width     float64
		gap              float64
	}{
		{
			name: "three columns positioned with gap",
			columns: [][]*Fragment{
				{{ContentWidth: 100, ContentHeight: 50, OffsetX: 0, OffsetY: 0}},
				{{ContentWidth: 100, ContentHeight: 50, OffsetX: 0, OffsetY: 0}},
				{{ContentWidth: 100, ContentHeight: 50, OffsetX: 0, OffsetY: 0}},
			},
			column_width: 100,
			gap:          10,

			expected_offsets: []float64{0, 110, 220},
		},
		{
			name: "single column is positioned at origin",
			columns: [][]*Fragment{
				{{ContentWidth: 200, ContentHeight: 50, OffsetX: 0, OffsetY: 0}},
			},
			column_width:     200,
			gap:              10,
			expected_offsets: []float64{0},
		},
		{
			name: "zero gap places columns flush",
			columns: [][]*Fragment{
				{{ContentWidth: 100, ContentHeight: 50, OffsetX: 0, OffsetY: 0}},
				{{ContentWidth: 100, ContentHeight: 50, OffsetX: 0, OffsetY: 0}},
			},
			column_width:     100,
			gap:              0,
			expected_offsets: []float64{0, 100},
		},
		{
			name: "multiple items per column all share the column offset",
			columns: [][]*Fragment{
				{
					{ContentWidth: 100, ContentHeight: 25, OffsetX: 0, OffsetY: 0},
					{ContentWidth: 100, ContentHeight: 25, OffsetX: 0, OffsetY: 25},
				},
				{
					{ContentWidth: 100, ContentHeight: 50, OffsetX: 0, OffsetY: 0},
				},
			},
			column_width: 100,
			gap:          10,

			expected_offsets: []float64{0, 0, 110},
		},
		{
			name:             "empty columns returns empty result",
			columns:          nil,
			column_width:     100,
			gap:              10,
			expected_offsets: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := positionColumns(tt.columns, tt.column_width, tt.gap)

			if tt.expected_offsets == nil {
				assert.Empty(t, result)
				return
			}

			assert.Equal(t, len(tt.expected_offsets), len(result), "fragment count")
			for i, fragment := range result {
				assert.InDelta(t, tt.expected_offsets[i], fragment.OffsetX, columnEpsilon, "fragment %d OffsetX", i)
			}
		})
	}
}
