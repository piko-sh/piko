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

func TestBuildGridAreaMap(t *testing.T) {
	tests := []struct {
		name     string
		areas    [][]string
		expected map[string]gridAreaBounds
	}{
		{
			"2x2 grid with shared header area",
			[][]string{
				{"header", "header"},
				{"sidebar", "main"},
			},
			map[string]gridAreaBounds{
				"header":  {rowStart: 0, rowEnd: 1, columnStart: 0, columnEnd: 2},
				"sidebar": {rowStart: 1, rowEnd: 2, columnStart: 0, columnEnd: 1},
				"main":    {rowStart: 1, rowEnd: 2, columnStart: 1, columnEnd: 2},
			},
		},
		{
			"dot placeholders are skipped",
			[][]string{
				{"nav", "."},
				{".", "content"},
			},
			map[string]gridAreaBounds{
				"nav":     {rowStart: 0, rowEnd: 1, columnStart: 0, columnEnd: 1},
				"content": {rowStart: 1, rowEnd: 2, columnStart: 1, columnEnd: 2},
			},
		},
		{
			"empty input returns nil",
			[][]string{},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildGridAreaMap(tt.areas)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAreaAvailable(t *testing.T) {
	tests := []struct {
		name     string
		occupied map[[2]int]bool
		row      int
		column   int
		row_span int
		col_span int
		expected bool
	}{
		{
			"available cell in empty grid",
			make(map[[2]int]bool),
			0, 0, 1, 1,
			true,
		},
		{
			"occupied cell returns false",
			map[[2]int]bool{{0, 0}: true},
			0, 0, 1, 1,
			false,
		},
		{
			"multi-cell span fully available",
			make(map[[2]int]bool),
			0, 0, 2, 2,
			true,
		},
		{
			"multi-cell span partially occupied",
			map[[2]int]bool{{1, 1}: true},
			0, 0, 2, 2,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAreaAvailable(tt.occupied, tt.row, tt.column, tt.row_span, tt.col_span)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAutoPlaceItem(t *testing.T) {
	tests := []struct {
		name            string
		occupied        map[[2]int]bool
		col_span        int
		row_span        int
		max_columns     int
		cursor_row      int
		cursor_column   int
		expected_column int
		expected_row    int
	}{
		{
			"first item in empty grid placed at origin",
			make(map[[2]int]bool),
			1, 1, 2,
			0, 0,
			0, 0,
		},
		{
			"second item placed beside first",
			map[[2]int]bool{{0, 0}: true},
			1, 1, 2,
			0, 1,
			1, 0,
		},
		{
			"item at end of row wraps to next row",
			map[[2]int]bool{{0, 0}: true, {0, 1}: true},
			1, 1, 2,
			1, 0,
			0, 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, row := autoPlaceItem(
				tt.occupied,
				tt.col_span, tt.row_span, tt.max_columns,
				GridAutoFlowRow,
				new(tt.cursor_row), new(tt.cursor_column),
			)
			assert.Equal(t, tt.expected_column, col)
			assert.Equal(t, tt.expected_row, row)
		})
	}
}

func TestComputeTrackOffsets(t *testing.T) {
	tests := []struct {
		name     string
		sizes    []float64
		gap      float64
		expected []float64
	}{
		{
			"three tracks with gap produces cumulative offsets",
			[]float64{100, 200, 300},
			10,

			[]float64{0, 110, 320, 620},
		},
		{
			"empty sizes returns single zero offset",
			[]float64{},
			10,
			[]float64{0},
		},
		{
			"single track returns two offsets without gap",
			[]float64{100},
			10,
			[]float64{0, 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeTrackOffsets(tt.sizes, tt.gap)
			assert.Equal(t, len(tt.expected), len(result))
			for index := range tt.expected {
				assert.InDelta(t, tt.expected[index], result[index], 0.001)
			}
		})
	}
}

func TestSpanTrackSize(t *testing.T) {
	tests := []struct {
		name     string
		sizes    []float64
		start    int
		end      int
		gap      float64
		expected float64
	}{
		{
			"single track returns its size without gap",
			[]float64{100, 200},
			0, 1,
			10,
			100,
		},
		{
			"multi-track span includes intermediate gaps",
			[]float64{100, 200, 300},
			0, 2,
			10,

			310,
		},
		{
			"full span across all tracks",
			[]float64{100, 200, 300},
			0, 3,
			10,

			620,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := spanTrackSize(tt.sizes, tt.start, tt.end, tt.gap)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestComputeGridBounds(t *testing.T) {
	tests := []struct {
		name             string
		placements       []gridItemPlacement
		template_columns int
		template_rows    int
		expected_columns int
		expected_rows    int
	}{
		{
			"no placements returns template dimensions",
			nil,
			3, 2,
			3, 2,
		},
		{
			"placements within template bounds",
			[]gridItemPlacement{
				{column: 0, row: 0, columnEnd: 1, rowEnd: 1},
				{column: 1, row: 1, columnEnd: 2, rowEnd: 2},
			},
			3, 2,
			3, 2,
		},
		{
			"placements exceeding template bounds",
			[]gridItemPlacement{
				{column: 0, row: 0, columnEnd: 5, rowEnd: 1},
				{column: 1, row: 3, columnEnd: 2, rowEnd: 4},
			},
			2, 2,
			5, 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			max_column, max_row := computeGridBounds(tt.placements, tt.template_columns, tt.template_rows)
			assert.Equal(t, tt.expected_columns, max_column)
			assert.Equal(t, tt.expected_rows, max_row)
		})
	}
}

func TestExpandAutoRepeatTracks(t *testing.T) {
	tests := []struct {
		name          string
		fixed         []GridTrack
		ar            *GridAutoRepeat
		containerSize float64
		gap           float64
		expected      []GridTrack
	}{
		{
			"200pt tracks in 600pt container produces 3 repetitions",
			nil,
			&GridAutoRepeat{
				Type:    GridAutoRepeatFill,
				Pattern: []GridTrack{{Value: 200, Unit: GridTrackPoints}},
			},
			600, 0,
			[]GridTrack{
				{Value: 200, Unit: GridTrackPoints},
				{Value: 200, Unit: GridTrackPoints},
				{Value: 200, Unit: GridTrackPoints},
			},
		},
		{
			"200pt tracks in 600pt container with 10pt gap",
			nil,
			&GridAutoRepeat{
				Type:    GridAutoRepeatFill,
				Pattern: []GridTrack{{Value: 200, Unit: GridTrackPoints}},
			},

			600, 10,
			[]GridTrack{
				{Value: 200, Unit: GridTrackPoints},
				{Value: 200, Unit: GridTrackPoints},
			},
		},
		{
			"fixed tracks before and after reduce available space",
			[]GridTrack{
				{Value: 100, Unit: GridTrackPoints},
				{Value: 100, Unit: GridTrackPoints},
			},
			&GridAutoRepeat{
				Type:        GridAutoRepeatFill,
				Pattern:     []GridTrack{{Value: 150, Unit: GridTrackPoints}},
				InsertIndex: 1,
				AfterCount:  1,
			},
			600, 0,

			[]GridTrack{
				{Value: 100, Unit: GridTrackPoints},
				{Value: 150, Unit: GridTrackPoints},
				{Value: 150, Unit: GridTrackPoints},
				{Value: 100, Unit: GridTrackPoints},
			},
		},
		{
			"at least 1 repetition even if container is small",
			nil,
			&GridAutoRepeat{
				Type:    GridAutoRepeatFill,
				Pattern: []GridTrack{{Value: 500, Unit: GridTrackPoints}},
			},
			100, 0,
			[]GridTrack{{Value: 500, Unit: GridTrackPoints}},
		},
		{
			"multi-track pattern repeats as a unit",
			nil,
			&GridAutoRepeat{
				Type: GridAutoRepeatFill,
				Pattern: []GridTrack{
					{Value: 100, Unit: GridTrackPoints},
					{Value: 50, Unit: GridTrackPoints},
				},
			},

			450, 0,
			[]GridTrack{
				{Value: 100, Unit: GridTrackPoints},
				{Value: 50, Unit: GridTrackPoints},
				{Value: 100, Unit: GridTrackPoints},
				{Value: 50, Unit: GridTrackPoints},
				{Value: 100, Unit: GridTrackPoints},
				{Value: 50, Unit: GridTrackPoints},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandAutoRepeatTracks(tt.fixed, tt.ar, tt.containerSize, tt.gap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCollapseEmptyAutoFitTracks(t *testing.T) {
	tracks := []GridTrack{
		{Value: 100, Unit: GridTrackPoints},
		{Value: 200, Unit: GridTrackPoints},
		{Value: 200, Unit: GridTrackPoints},
		{Value: 200, Unit: GridTrackPoints},
		{Value: 100, Unit: GridTrackPoints},
	}
	ar := &GridAutoRepeat{
		Type:        GridAutoRepeatFit,
		Pattern:     []GridTrack{{Value: 200, Unit: GridTrackPoints}},
		InsertIndex: 1,
		AfterCount:  1,
	}

	placements := []gridItemPlacement{
		{column: 1, columnEnd: 2},
		{column: 3, columnEnd: 4},
	}

	result := collapseEmptyAutoFitTracks(tracks, ar, placements)

	assert.Equal(t, GridTrackPoints, result[0].Unit)
	assert.Equal(t, 100.0, result[0].Value)
	assert.Equal(t, 200.0, result[1].Value)
	assert.Equal(t, 0.0, result[2].Value)
	assert.Equal(t, 200.0, result[3].Value)
	assert.Equal(t, 100.0, result[4].Value)
}
