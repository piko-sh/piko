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
	"fmt"

	"piko.sh/piko/internal/layouter/layouter_dto"
)

// gridOccupancy records which cells of a grid are covered by placed items.
type gridOccupancy struct {
	// limits enforces the grid track and cell limits and records the first breach.
	limits *LimitTracker

	// rows holds, for each row index, which column indices are occupied.
	rows [][]bool

	// maxTracks holds the maximum number of tracks on either axis.
	maxTracks int
}

// newGridOccupancy creates an empty occupancy grid bounded by the tracker's limits.
//
// Takes limits (*LimitTracker) which enforces the grid limits, or nil for the defaults.
//
// Returns *gridOccupancy which has no occupied cells.
func newGridOccupancy(limits *LimitTracker) *gridOccupancy {
	return &gridOccupancy{
		limits:    limits,
		rows:      nil,
		maxTracks: limits.Limits().MaxGridTracks,
	}
}

// withinTrackLimit reports whether an item ending at the given exclusive column and row
// fits within MaxGridTracks, recording a breach when it does not.
//
// Takes columnEnd (int) which is the exclusive end column of the item.
// Takes rowEnd (int) which is the exclusive end row of the item.
//
// Returns bool which is false when the item extends beyond the track limit.
func (o *gridOccupancy) withinTrackLimit(columnEnd, rowEnd int) bool {
	if columnEnd > o.maxTracks || rowEnd > o.maxTracks {
		o.limits.fail(fmt.Errorf("grid item extends beyond %d tracks: %w",
			o.maxTracks, layouter_dto.ErrTooManyGridTracks))
		return false
	}
	return true
}

// isOccupied reports whether the cell at row and column is covered by a placed item.
//
// Takes row (int) which is the cell's row index.
// Takes column (int) which is the cell's column index.
//
// Returns bool which is true when the cell is occupied.
func (o *gridOccupancy) isOccupied(row, column int) bool {
	if row < 0 || row >= len(o.rows) || column < 0 || column >= len(o.rows[row]) {
		return false
	}
	return o.rows[row][column]
}

// isAreaAvailable reports whether the rectangular region starting at (row, column) with
// the given spans is entirely unoccupied.
//
// Takes row (int) which is the start row.
// Takes column (int) which is the start column.
// Takes rowSpan (int) which is the row extent.
// Takes columnSpan (int) which is the column extent.
//
// Returns bool which is true if every cell in the area is free.
func (o *gridOccupancy) isAreaAvailable(row, column, rowSpan, columnSpan int) bool {
	if o.limits.failed() {
		return true
	}
	for checkRow := row; checkRow < row+rowSpan; checkRow++ {
		examined := 0
		for checkColumn := column; checkColumn < column+columnSpan; checkColumn++ {
			examined++
			if o.isOccupied(checkRow, checkColumn) {
				return !o.limits.addGridCells(examined)
			}
		}
		if !o.limits.addGridCells(examined) {
			return true
		}
	}
	return true
}

// mark records every cell covered by a placement as occupied, counting the cells against
// the grid cell budget and stopping once it is exhausted.
//
// Takes placement (gridItemPlacement) which holds the item position and span.
func (o *gridOccupancy) mark(placement gridItemPlacement) {
	if placement.column < 0 || placement.row < 0 {
		return
	}
	for row := placement.row; row < placement.rowEnd; row++ {
		if !o.limits.addGridCells(placement.columnEnd - placement.column) {
			return
		}
		cells := o.rowForWrite(row, placement.columnEnd)
		for column := placement.column; column < placement.columnEnd; column++ {
			cells[column] = true
		}
	}
}

// rowForWrite returns the occupancy slice for row, growing the grid so the row exists and
// holds at least columnCount cells.
//
// Takes row (int) which is the row index to return.
// Takes columnCount (int) which is the minimum number of cells the row must hold.
//
// Returns []bool which is the writable occupancy slice for the row.
func (o *gridOccupancy) rowForWrite(row, columnCount int) []bool {
	for len(o.rows) <= row {
		o.rows = append(o.rows, nil)
	}
	if len(o.rows[row]) < columnCount {
		grown := make([]bool, max(columnCount, min(2*len(o.rows[row]), o.maxTracks)))
		copy(grown, o.rows[row])
		o.rows[row] = grown
	}
	return o.rows[row]
}
