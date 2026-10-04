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
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_dto"
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
			result := occupancyFromCells(t, tt.occupied).isAreaAvailable(tt.row, tt.column, tt.row_span, tt.col_span)
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
				occupancyFromCells(t, tt.occupied),
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
			result := expandAutoRepeatTracks(tt.fixed, tt.ar, tt.containerSize, tt.gap, nil)
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

func occupancyFromCells(t *testing.T, cells map[[2]int]bool) *gridOccupancy {
	t.Helper()
	occupancy := newGridOccupancy(nil)
	for cell, occupied := range cells {
		if occupied {
			occupancy.rowForWrite(cell[0], cell[1]+1)[cell[1]] = true
		}
	}
	return occupancy
}

func TestExpandAutoRepeatTracks_HostileInput(t *testing.T) {
	t.Parallel()

	pattern := []GridTrack{{Value: 100, Unit: GridTrackPoints}}
	tests := []struct {
		wantErr       error
		name          string
		pattern       []GridTrack
		limits        layouter_dto.LayoutLimits
		containerSize float64
		gap           float64
		wantTracks    int
	}{
		{
			name:          "negative gap is treated as zero and terminates",
			pattern:       pattern,
			containerSize: 400,
			gap:           -100,
			wantTracks:    4,
		},
		{
			name:          "infinite container repeats once",
			pattern:       pattern,
			containerSize: math.Inf(1),
			wantTracks:    1,
		},
		{
			name:          "NaN container repeats once",
			pattern:       pattern,
			containerSize: math.NaN(),
			wantTracks:    1,
		},
		{
			name:          "empty pattern adds no tracks",
			pattern:       nil,
			containerSize: 400,
			wantTracks:    0,
		},
		{
			name:          "tiny track breaches the track limit",
			pattern:       []GridTrack{{Value: 0.00001, Unit: GridTrackPoints}},
			containerSize: 400,
			limits:        layouter_dto.LayoutLimits{MaxGridTracks: 50},
			wantTracks:    50,
			wantErr:       layouter_dto.ErrTooManyGridTracks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tracker := NewLimitTracker(tt.limits)
			ar := &GridAutoRepeat{Type: GridAutoRepeatFill, Pattern: tt.pattern}
			tracks := expandAutoRepeatTracks(nil, ar, tt.containerSize, tt.gap, tracker)

			assert.Len(t, tracks, tt.wantTracks)
			if tt.wantErr != nil {
				assert.ErrorIs(t, tracker.Err(), tt.wantErr)
				return
			}
			assert.NoError(t, tracker.Err())
		})
	}
}

func TestAutoRepeatCount_MatchesIncrementalSearch(t *testing.T) {
	t.Parallel()

	incremental := func(oneRepetition, gap, available float64) int {
		count := 1
		if oneRepetition > 0 {
			for {
				next := float64(count) * oneRepetition
				if count > 1 {
					next += float64(count-1) * gap
				}
				if next > available {
					count--
					break
				}
				count++
			}
			count = max(count, 1)
		}
		return count
	}

	for _, oneRepetition := range []float64{0, 0.1, 1, 7.5, 33.3, 100, 150, 1000} {
		for _, gap := range []float64{0, 0.1, 2, 10, 33.3} {
			for _, available := range []float64{0, 1, 99.9, 100, 300, 333.3, 600, 1234.5} {
				assert.Equal(t, incremental(oneRepetition, gap, available), autoRepeatCount(oneRepetition, gap, available),
					"repetition %v gap %v available %v", oneRepetition, gap, available)
			}
		}
	}
}

func TestAutoRepeatCount_SaturatesForAbsurdRatios(t *testing.T) {
	t.Parallel()

	assert.Equal(t, maxPageIndex, autoRepeatCount(1e-300, 0, 1e300))
	assert.Equal(t, 1, autoRepeatCount(math.Inf(1), 0, 100))
	assert.Equal(t, 1, autoRepeatCount(math.NaN(), 0, 100))
}

func TestGridLayout_HostileTemplatesTerminate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		properties map[string]string
		wantErr    error
		name       string
	}{
		{
			name: "negative column gap with auto-fill",
			properties: map[string]string{
				"display":               "grid",
				"grid-template-columns": "repeat(auto-fill, 100px)",
				"column-gap":            "-100px",
			},
		},
		{
			name: "tiny auto-fill track",
			properties: map[string]string{
				"display":               "grid",
				"grid-template-columns": "repeat(auto-fill, 0.00001px)",
			},
			wantErr: layouter_dto.ErrTooManyGridTracks,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := makeRoot(400)
			root.Children = []*LayoutBox{newGridContainerForTest(root, tt.properties)}

			_, err := LayoutBoxTree(context.Background(), root, &mockFontMetrics{}, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestPlaceGridItems_EnforcesLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		configure func(item *LayoutBox)
		wantErr   error
		name      string
		limits    layouter_dto.LayoutLimits
		items     int
	}{
		{
			name:   "column span beyond the track limit",
			limits: layouter_dto.LayoutLimits{MaxGridTracks: 100},
			items:  1,
			configure: func(item *LayoutBox) {
				item.Style.GridColumnStart = GridLine{Span: 500}
			},
			wantErr: layouter_dto.ErrTooManyGridTracks,
		},
		{
			name:   "explicit row line beyond the track limit",
			limits: layouter_dto.LayoutLimits{MaxGridTracks: 100},
			items:  1,
			configure: func(item *LayoutBox) {
				item.Style.GridRowStart = GridLine{Line: 1000}
				item.Style.GridColumnStart = GridLine{Line: 1}
			},
			wantErr: layouter_dto.ErrTooManyGridTracks,
		},
		{
			name:   "overlapping large items exhaust the cell budget",
			limits: layouter_dto.LayoutLimits{MaxGridCells: 10_000},
			items:  10,
			configure: func(item *LayoutBox) {
				item.Style.GridColumnStart = GridLine{Line: 1}
				item.Style.GridColumnEnd = GridLine{Span: 50}
				item.Style.GridRowStart = GridLine{Line: 1}
				item.Style.GridRowEnd = GridLine{Span: 50}
			},
			wantErr: layouter_dto.ErrGridTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			grid := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle()}
			grid.Style.Display = DisplayGrid
			for range tt.items {
				item := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid}
				tt.configure(item)
				grid.Children = append(grid.Children, item)
			}

			tracker := NewLimitTracker(tt.limits)
			placements, columns, rows := placeGridItemsWithColumns(grid, []GridTrack{{}}, tracker)

			assert.ErrorIs(t, tracker.Err(), tt.wantErr)
			assert.Nil(t, placements)
			assert.Zero(t, columns)
			assert.Zero(t, rows)
		})
	}
}

func TestPlaceGridItems_AutoPlacementBeyondSearchWindow(t *testing.T) {
	t.Parallel()

	const itemCount = gridSearchLimit + 200
	grid := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle()}
	grid.Style.Display = DisplayGrid
	for range itemCount {
		grid.Children = append(grid.Children, &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid})
	}

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{})
	placements, columns, rows := placeGridItemsWithColumns(grid, []GridTrack{{}}, tracker)

	require.NoError(t, tracker.Err())
	require.Len(t, placements, itemCount)
	assert.Equal(t, 1, columns)
	assert.Equal(t, itemCount, rows)
	for index, placement := range placements {
		assert.Equal(t, index, placement.row, "item %d should occupy its own row", index)
	}
}

func TestSingleSpanItemsByTrack(t *testing.T) {
	t.Parallel()

	first := &LayoutBox{}
	second := &LayoutBox{}
	spanning := &LayoutBox{}
	placements := []gridItemPlacement{
		{item: first, column: 0, columnEnd: 1, row: 0, rowEnd: 1},
		{item: second, column: 0, columnEnd: 1, row: 1, rowEnd: 2},
		{item: spanning, column: 1, columnEnd: 3, row: 0, rowEnd: 1},
		{item: &LayoutBox{}, column: 9, columnEnd: 10, row: 9, rowEnd: 10},
	}

	columns := singleSpanItemsByTrack(placements, 3, true)
	assert.Equal(t, []*LayoutBox{first, second}, columns[0])
	assert.Empty(t, columns[1], "a spanning item belongs to no single track")
	assert.Empty(t, columns[2])

	rows := singleSpanItemsByTrack(placements, 2, false)
	assert.Equal(t, []*LayoutBox{first, spanning}, rows[0])
	assert.Equal(t, []*LayoutBox{second}, rows[1])
}

func TestPlaceGridItems_SemiAutomaticAndNamedPlacement(t *testing.T) {
	t.Parallel()

	grid := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle()}
	grid.Style.Display = DisplayGrid
	grid.Style.GridTemplateAreas = [][]string{{"head", "head"}, {"side", "main"}}

	named := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid}
	named.Style.GridArea = "main"
	columnOnly := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid}
	columnOnly.Style.GridColumnStart = GridLine{Line: 1}
	rowOnly := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid}
	rowOnly.Style.GridRowStart = GridLine{Line: 3}
	grid.Children = []*LayoutBox{named, columnOnly, rowOnly}

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{})
	placements, columns, rows := placeGridItemsWithColumns(grid, []GridTrack{{}, {}}, tracker)

	require.NoError(t, tracker.Err())
	require.Len(t, placements, 3)
	assert.Equal(t, gridItemPlacement{item: named, column: 1, columnEnd: 2, row: 1, rowEnd: 2}, placements[0])
	assert.Equal(t, 0, placements[1].column, "a column-only item keeps its column")
	assert.Equal(t, 0, placements[1].row, "and takes the first free row in that column")
	assert.Equal(t, 2, placements[2].row, "a row-only item keeps its row")
	assert.Equal(t, 0, placements[2].column, "and takes the first free column in that row")
	assert.Equal(t, 2, columns)
	assert.Equal(t, 3, rows)
}

func TestPlaceGridItems_NamedAreaBeyondTrackLimit(t *testing.T) {
	t.Parallel()

	grid := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle()}
	grid.Style.Display = DisplayGrid
	grid.Style.GridTemplateAreas = [][]string{{"a", "a", "a", "a"}}
	item := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: grid}
	item.Style.GridArea = "a"
	grid.Children = []*LayoutBox{item}

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxGridTracks: 3})
	placements, _, _ := placeGridItemsWithColumns(grid, []GridTrack{{}}, tracker)

	assert.Nil(t, placements)
	assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrTooManyGridTracks)
}

func TestResolveAutoRepeatRows(t *testing.T) {
	t.Parallel()

	box := &LayoutBox{Style: DefaultComputedStyle()}
	box.Style.GridTemplateRows = []GridTrack{{Value: 10, Unit: GridTrackPoints}}
	assert.Equal(t, box.Style.GridTemplateRows, resolveAutoRepeatRows(box, nil))

	box.Style.GridAutoRepeatRows = &GridAutoRepeat{Type: GridAutoRepeatFill, Pattern: []GridTrack{{Value: 20, Unit: GridTrackPoints}}, InsertIndex: 1}
	rows := resolveAutoRepeatRows(box, nil)
	assert.Len(t, rows, 2, "an indefinite block size repeats the pattern once")
}
