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
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

func TestGridOccupancy_MarkAndQuery(t *testing.T) {
	t.Parallel()

	occupancy := newGridOccupancy(NewLimitTracker(layouter_dto.LayoutLimits{}))
	occupancy.mark(gridItemPlacement{column: 1, columnEnd: 3, row: 2, rowEnd: 4})

	tests := []struct {
		name   string
		row    int
		column int
		want   bool
	}{
		{name: "inside the area", row: 2, column: 1, want: true},
		{name: "last cell of the area", row: 3, column: 2, want: true},
		{name: "left of the area", row: 2, column: 0, want: false},
		{name: "row above the area", row: 1, column: 1, want: false},
		{name: "beyond the stored rows", row: 50, column: 1, want: false},
		{name: "beyond the stored columns", row: 2, column: 50, want: false},
		{name: "negative coordinates", row: -1, column: -1, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, occupancy.isOccupied(tt.row, tt.column))
		})
	}

	assert.True(t, occupancy.isAreaAvailable(0, 0, 2, 5))
	assert.False(t, occupancy.isAreaAvailable(1, 0, 2, 2))
}

func TestGridOccupancy_MarkIgnoresUnplacedItems(t *testing.T) {
	t.Parallel()

	occupancy := newGridOccupancy(nil)
	occupancy.mark(gridItemPlacement{column: -1, columnEnd: 1, row: 0, rowEnd: 1})

	assert.Empty(t, occupancy.rows)
}

func TestGridOccupancy_RowGrowthKeepsExistingCells(t *testing.T) {
	t.Parallel()

	occupancy := newGridOccupancy(nil)
	occupancy.mark(gridItemPlacement{column: 0, columnEnd: 1, row: 0, rowEnd: 1})
	occupancy.mark(gridItemPlacement{column: 7, columnEnd: 8, row: 0, rowEnd: 1})

	assert.True(t, occupancy.isOccupied(0, 0), "growing a row keeps the cells already marked")
	assert.True(t, occupancy.isOccupied(0, 7))
	assert.False(t, occupancy.isOccupied(0, 4))
}

func TestGridOccupancy_WithinTrackLimit(t *testing.T) {
	t.Parallel()

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxGridTracks: 10})
	occupancy := newGridOccupancy(tracker)

	assert.True(t, occupancy.withinTrackLimit(10, 10))
	require.NoError(t, tracker.Err())

	assert.False(t, occupancy.withinTrackLimit(11, 1))
	assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrTooManyGridTracks)
}

func TestGridOccupancy_CellBudget(t *testing.T) {
	t.Parallel()

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxGridCells: 20})
	occupancy := newGridOccupancy(tracker)

	occupancy.mark(gridItemPlacement{column: 0, columnEnd: 4, row: 0, rowEnd: 4})
	require.NoError(t, tracker.Err(), "sixteen cells fit the budget")

	assert.False(t, occupancy.isAreaAvailable(0, 0, 1, 1), "an occupied cell within the budget is reported")
	require.NoError(t, tracker.Err())

	occupancy.isAreaAvailable(10, 10, 3, 3)
	assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrGridTooLarge)
	assert.True(t, occupancy.isAreaAvailable(0, 0, 4, 4),
		"once the budget is exhausted every area reports available so searches end")
}

func TestGridOccupancy_MarkStopsWhenBudgetExhausted(t *testing.T) {
	t.Parallel()

	tracker := NewLimitTracker(layouter_dto.LayoutLimits{MaxGridCells: 10})
	occupancy := newGridOccupancy(tracker)

	occupancy.mark(gridItemPlacement{column: 0, columnEnd: 5, row: 0, rowEnd: 100})

	assert.ErrorIs(t, tracker.Err(), layouter_dto.ErrGridTooLarge)
	assert.LessOrEqual(t, len(occupancy.rows), 3, "marking stops once the budget is exhausted")
}
