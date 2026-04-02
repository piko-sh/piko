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
)

const (
	// gridSearchLimit is the maximum number of rows or columns
	// searched during grid auto-placement.
	gridSearchLimit = 1000

	// gridColumnOverflow is the extra columns searched beyond
	// the explicit grid when looking for available slots.
	gridColumnOverflow = 100
)

// gridItemPlacement holds the resolved grid position and layout
// fragment for a single grid item.
type gridItemPlacement struct {
	// item holds the layout box for this grid item.
	item *LayoutBox

	// fragment holds the laid-out fragment produced during layout.
	fragment *Fragment

	// column holds the zero-based start column index.
	column int

	// row holds the zero-based start row index.
	row int

	// columnEnd holds the exclusive end column index.
	columnEnd int

	// rowEnd holds the exclusive end row index.
	rowEnd int
}

// gridAreaBounds describes the rectangular region occupied
// by a named grid area within the grid-template-areas matrix.
type gridAreaBounds struct {
	// rowStart holds the zero-based inclusive start row.
	rowStart int

	// rowEnd holds the exclusive end row.
	rowEnd int

	// columnStart holds the zero-based inclusive start column.
	columnStart int

	// columnEnd holds the exclusive end column.
	columnEnd int
}

// buildGridAreaMap scans a grid-template-areas matrix and
// returns a mapping from area name to its bounding rectangle.
// The "." name (unnamed cells) is excluded.
//
// Takes areas ([][]string) which holds the grid-template-areas
// matrix.
//
// Returns map[string]gridAreaBounds which maps area names to
// their bounding rectangles, or nil when the matrix is empty.
func buildGridAreaMap(areas [][]string) map[string]gridAreaBounds {
	if len(areas) == 0 {
		return nil
	}
	result := make(map[string]gridAreaBounds)
	for row, cells := range areas {
		for col, name := range cells {
			if name != "." {
				updateAreaBounds(result, name, row, col)
			}
		}
	}
	return result
}

// updateAreaBounds inserts or expands the bounding rectangle for a
// named grid area at the given row and column position.
//
// Takes m (map[string]gridAreaBounds) which holds the area map
// to update.
// Takes name (string) which specifies the area name.
// Takes row (int) which specifies the row index of the cell.
// Takes col (int) which specifies the column index of the cell.
func updateAreaBounds(m map[string]gridAreaBounds, name string, row, col int) {
	bounds, exists := m[name]
	if !exists {
		m[name] = gridAreaBounds{
			rowStart:    row,
			rowEnd:      row + 1,
			columnStart: col,
			columnEnd:   col + 1,
		}
		return
	}
	if row+1 > bounds.rowEnd {
		bounds.rowEnd = row + 1
	}
	if col+1 > bounds.columnEnd {
		bounds.columnEnd = col + 1
	}
	m[name] = bounds
}

// layoutGridContainer lays out a CSS grid container,
// resolving track sizes, placing items, and positioning
// fragments.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes box (*LayoutBox) which is the grid container box.
//
// Returns formattingContextResult which holds the laid-out
// child fragments and content height.
func layoutGridContainer(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	containerWidth := input.AvailableWidth
	templateColumns := resolveAutoRepeatColumns(box, containerWidth)
	templateRows := resolveAutoRepeatRows(box)

	placements, columnCount, rowCount := placeGridItemsWithColumns(box, templateColumns)

	if box.Style.GridAutoRepeatColumns != nil &&
		box.Style.GridAutoRepeatColumns.Type == GridAutoRepeatFit {
		templateColumns = collapseEmptyAutoFitTracks(
			templateColumns, box.Style.GridAutoRepeatColumns, placements,
		)
		columnCount = len(templateColumns)
	}

	columnWidths := resolveGridTrackSizes(gridTrackSizingInput{
		templateTracks: templateColumns,
		autoTracks:     box.Style.GridAutoColumns,
		trackCount:     columnCount,
		containerSize:  containerWidth,
		gap:            box.Style.ColumnGap,
		placements:     placements,
		isColumn:       true,
		fontMetrics:    input.FontMetrics,
	})

	layoutGridItemFragments(ctx, placements, box, columnWidths, input)

	containerHeight := resolveGridContainerHeight(&box.Style, input.AvailableBlockSize)
	rowHeights := computeRowHeights(
		templateRows, box.Style.GridAutoRows,
		rowCount, placements, box.Style.RowGap, containerHeight,
	)

	relayoutStretchedGridItems(ctx, placements, box, columnWidths, rowHeights, input)

	childFragments := positionGridItems(box, placements, columnWidths, rowHeights, containerWidth)
	totalHeight := sumTrackHeights(rowHeights, box.Style.RowGap)

	return formattingContextResult{
		Children:      childFragments,
		ContentHeight: totalHeight,
		Margin: BoxEdges{
			Top:    input.Edges.MarginTop,
			Bottom: input.Edges.MarginBottom,
		},
	}
}

// resolveAutoRepeatColumns expands any auto-fill/auto-fit repeat
// on the column axis using the container width.
//
// Takes box (*LayoutBox) which is the grid container box.
// Takes containerWidth (float64) which specifies the available
// inline size.
//
// Returns []GridTrack which holds the expanded column track list.
func resolveAutoRepeatColumns(box *LayoutBox, containerWidth float64) []GridTrack {
	cols := box.Style.GridTemplateColumns
	if box.Style.GridAutoRepeatColumns != nil {
		cols = expandAutoRepeatTracks(
			cols, box.Style.GridAutoRepeatColumns,
			containerWidth, box.Style.ColumnGap,
		)
	}
	return cols
}

// resolveAutoRepeatRows expands any auto-fill/auto-fit repeat
// on the row axis. The available block size is typically
// indefinite so 0 is used.
//
// Takes box (*LayoutBox) which is the grid container box.
//
// Returns []GridTrack which holds the expanded row track list.
func resolveAutoRepeatRows(box *LayoutBox) []GridTrack {
	rows := box.Style.GridTemplateRows
	if box.Style.GridAutoRepeatRows != nil {
		rows = expandAutoRepeatTracks(
			rows, box.Style.GridAutoRepeatRows, 0, box.Style.RowGap,
		)
	}
	return rows
}

// sumTrackHeights returns the total height of all row tracks
// including gaps.
//
// Takes heights ([]float64) which holds the resolved row heights.
// Takes gap (float64) which specifies the row gap size.
//
// Returns float64 which holds the total height in points.
func sumTrackHeights(heights []float64, gap float64) float64 {
	total := 0.0
	for index, h := range heights {
		total += h
		if index > 0 {
			total += gap
		}
	}
	return total
}

// placeGridItemsWithColumns is like placeGridItems but uses the
// provided template columns instead of the style's columns.
// This is needed when auto-repeat has expanded the column list.
//
// Takes box (*LayoutBox) which is the grid container box.
// Takes templateColumns ([]GridTrack) which holds the expanded
// column tracks.
//
// Returns placements ([]gridItemPlacement) which holds each
// item's resolved position.
// Returns columnCount (int) which holds the total column count.
// Returns rowCount (int) which holds the total row count.
func placeGridItemsWithColumns(
	box *LayoutBox, templateColumns []GridTrack,
) (placements []gridItemPlacement, columnCount, rowCount int) {
	numColumns := len(templateColumns)
	if numColumns == 0 {
		numColumns = 1
	}

	areaMap := buildGridAreaMap(box.Style.GridTemplateAreas)
	occupied := make(map[[2]int]bool)
	placements = make([]gridItemPlacement, 0, len(box.Children))
	autoFlow := box.Style.GridAutoFlow
	cursorRow := 0
	cursorColumn := 0

	for _, child := range box.Children {
		placement := placeGridChild(
			child, numColumns, occupied,
			autoFlow, &cursorRow, &cursorColumn, areaMap,
		)
		markOccupied(occupied, placement)
		placements = append(placements, placement)
	}

	maxColumn, maxRow := computeGridBounds(placements, numColumns, len(box.Style.GridTemplateRows))
	return placements, maxColumn, maxRow
}

// expandAutoRepeatTracks expands an auto-fill or auto-fit repeat
// pattern into a concrete track list. The number of repetitions
// is the largest integer that does not overflow the container.
//
// Takes fixedTracks ([]GridTrack) which is the non-repeat tracks.
// Takes ar (*GridAutoRepeat) which is the auto-repeat to expand.
// Takes containerSize (float64) which is the available space.
// Takes gap (float64) which is the gap between tracks.
//
// Returns the full expanded track list.
func expandAutoRepeatTracks(
	fixedTracks []GridTrack, ar *GridAutoRepeat,
	containerSize, gap float64,
) []GridTrack {
	fixedSize := sumDefiniteTrackSizes(fixedTracks)
	fixedGaps := 0.0
	if len(fixedTracks) > 0 {
		fixedGaps = float64(len(fixedTracks)) * gap
	}

	patternSize := sumDefiniteTrackSizes(ar.Pattern)
	patternGaps := float64(len(ar.Pattern)-1) * gap

	oneRepetition := patternSize + patternGaps
	available := containerSize - fixedSize - fixedGaps
	if available < 0 {
		available = 0
	}

	count := 1
	if oneRepetition > 0 {
		count = 1
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
		if count < 1 {
			count = 1
		}
	}

	return spliceAutoRepeat(fixedTracks, ar, count)
}

// sumDefiniteTrackSizes returns the total size of all tracks
// with a definite (fixed) size. Flexible and intrinsic tracks
// contribute 0.
//
// Takes tracks ([]GridTrack) which holds the track list to sum.
//
// Returns float64 which holds the total definite size in points.
func sumDefiniteTrackSizes(tracks []GridTrack) float64 {
	total := 0.0
	for _, track := range tracks {
		switch track.Unit {
		case GridTrackPoints:
			total += track.Value
		case GridTrackPercentage:
		}
	}
	return total
}

// spliceAutoRepeat inserts count repetitions of the auto-repeat
// pattern into the fixed tracks at the recorded insert index.
//
// Takes fixedTracks ([]GridTrack) which holds the non-repeat
// tracks.
// Takes ar (*GridAutoRepeat) which holds the auto-repeat pattern
// and insertion index.
// Takes count (int) which specifies the number of repetitions.
//
// Returns []GridTrack which holds the spliced track list.
func spliceAutoRepeat(fixedTracks []GridTrack, ar *GridAutoRepeat, count int) []GridTrack {
	before := fixedTracks
	var after []GridTrack
	if ar.InsertIndex < len(fixedTracks) {
		before = fixedTracks[:ar.InsertIndex]
		after = fixedTracks[ar.InsertIndex:]
	}

	result := make([]GridTrack, 0, len(before)+count*len(ar.Pattern)+len(after))
	result = append(result, before...)
	for range count {
		result = append(result, ar.Pattern...)
	}
	result = append(result, after...)
	return result
}

// collapseEmptyAutoFitTracks replaces empty tracks in the
// auto-repeat region with zero-width fixed tracks. A track is
// empty when no grid item occupies it.
//
// Takes expandedTracks ([]GridTrack) which holds the full
// expanded track list.
// Takes ar (*GridAutoRepeat) which holds the auto-repeat metadata
// for locating the repeat region.
// Takes placements ([]gridItemPlacement) which holds the item
// placements used to determine occupancy.
//
// Returns []GridTrack which holds the track list with empty
// auto-fit tracks collapsed to zero width.
func collapseEmptyAutoFitTracks(
	expandedTracks []GridTrack, ar *GridAutoRepeat,
	placements []gridItemPlacement,
) []GridTrack {
	occupied := make(map[int]bool)
	for _, p := range placements {
		for col := p.column; col < p.columnEnd; col++ {
			occupied[col] = true
		}
	}

	repeatStart := ar.InsertIndex
	repeatEnd := len(expandedTracks) - ar.AfterCount

	result := make([]GridTrack, len(expandedTracks))
	copy(result, expandedTracks)

	for index := repeatStart; index < repeatEnd; index++ {
		if !occupied[index] {
			result[index] = GridTrack{Value: 0, Unit: GridTrackPoints}
		}
	}
	return result
}

// layoutGridItemFragments lays out each grid item's content using
// the resolved column widths, producing a fragment per placement.
//
// When a row track has a definite size (points or percentage),
// the resolved height is passed as AvailableBlockSize so that
// percentage-height children can resolve correctly.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes placements ([]gridItemPlacement) which holds the item
// placements to lay out.
// Takes box (*LayoutBox) which is the grid container box.
// Takes columnWidths ([]float64) which holds the resolved column
// sizes.
func layoutGridItemFragments(
	ctx context.Context,
	placements []gridItemPlacement,
	box *LayoutBox,
	columnWidths []float64,
	input layoutInput,
) {
	for index := range placements {
		placement := &placements[index]
		itemWidth := spanTrackSize(columnWidths, placement.column, placement.columnEnd, box.Style.ColumnGap)

		childEdges := resolveEdgesFromStyle(&placement.item.Style, itemWidth)
		childMarginLeft := placement.item.Style.MarginLeft.Resolve(itemWidth, 0)
		childMarginRight := placement.item.Style.MarginRight.Resolve(itemWidth, 0)

		childContentWidth := itemWidth -
			childEdges.Padding.Horizontal() -
			childEdges.Border.Horizontal() -
			childMarginLeft - childMarginRight
		if childContentWidth < 0 {
			childContentWidth = 0
		}

		isStretch := box.Style.JustifyItems == JustifyItemsStretch
		if placement.item.Style.JustifySelf != JustifySelfAuto {
			isStretch = placement.item.Style.JustifySelf == JustifySelfStretch
		}
		fixedSize := isStretch || placement.item.Style.Width.IsAuto()

		rowHeight := definiteRowHeight(
			placement.row, box.Style.GridTemplateRows,
			box.Style.GridAutoRows, input.AvailableBlockSize,
		)
		overrideHeight := rowHeight > 0 && placement.item.Style.Height.IsAuto()
		originalHeight := placement.item.Style.Height
		if overrideHeight {
			placement.item.Style.Height = DimensionPt(rowHeight)
		}

		placement.fragment = layoutBox(ctx, placement.item, layoutInput{
			AvailableWidth:     childContentWidth,
			AvailableBlockSize: rowHeight,
			FontMetrics:        input.FontMetrics,
			Cache:              input.Cache,
			IsFixedInlineSize:  fixedSize,
			Edges:              childEdges,
		})

		if overrideHeight {
			placement.item.Style.Height = originalHeight
		}
	}
}

// definiteRowHeight returns the definite block size for a
// grid row track, or 0 if the track size is indefinite.
//
// Takes rowIndex (int) which specifies the zero-based row index.
// Takes templateRows ([]GridTrack) which holds the explicit row
// tracks.
// Takes autoRows ([]GridTrack) which holds the implicit row
// tracks.
// Takes containerHeight (float64) which specifies the container
// block size for percentage resolution.
//
// Returns float64 which holds the definite row height in points,
// or 0 when indefinite.
func definiteRowHeight(
	rowIndex int,
	templateRows []GridTrack,
	autoRows []GridTrack,
	containerHeight float64,
) float64 {
	track := getTrack(templateRows, autoRows, rowIndex)
	switch track.Unit {
	case GridTrackPoints:
		return track.Value
	case GridTrackPercentage:
		if containerHeight > 0 {
			return track.Value / percentageDivisor * containerHeight
		}
	}
	return 0
}

// resolveGridContainerHeight returns the definite block size
// for the grid container, or 0 if the height is indefinite.
//
// Takes style (*ComputedStyle) which holds the container's
// computed style.
// Takes parentBlockSize (float64) which specifies the parent's
// block size for percentage resolution.
//
// Returns float64 which holds the definite container height in
// points, or 0 when indefinite.
func resolveGridContainerHeight(style *ComputedStyle, parentBlockSize float64) float64 {
	if !style.Height.IsAuto() && !style.Height.IsIntrinsic() {
		return adjustForBoxSizing(
			style.Height.Resolve(parentBlockSize, 0),
			style, false,
		)
	}
	return 0
}

// relayoutStretchedGridItems re-lays out grid items that need
// a second pass.
//
//  1. Stretching: the cell height exceeds the item's natural
//     height, so flex/grid items must be re-laid out with the
//     cell height as their main size.
//  2. Percentage resolution: the item is in a row whose height
//     was not known at initial layout (auto or fr tracks), so
//     children with percentage heights could not resolve. Now
//     that the row height is computed, a re-layout lets
//     percentage heights resolve correctly.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes placements ([]gridItemPlacement) which holds the item
// placements to check.
// Takes box (*LayoutBox) which is the grid container box.
// Takes columnWidths ([]float64) which holds the resolved column
// sizes.
// Takes rowHeights ([]float64) which holds the resolved row
// sizes.
func relayoutStretchedGridItems(
	ctx context.Context,
	placements []gridItemPlacement,
	box *LayoutBox,
	columnWidths []float64,
	rowHeights []float64,
	input layoutInput,
) {
	for index := range placements {
		placement := &placements[index]
		cellHeight := spanTrackSize(rowHeights, placement.row, placement.rowEnd, box.Style.RowGap)
		naturalHeight := placement.fragment.MarginBoxHeight()

		initialRowHeight := definiteRowHeight(
			placement.row, box.Style.GridTemplateRows,
			box.Style.GridAutoRows, input.AvailableBlockSize,
		)
		rowWasIndefinite := initialRowHeight == 0 && cellHeight > 0

		needsStretch := cellHeight > naturalHeight &&
			(placement.item.Style.Display == DisplayFlex ||
				placement.item.Style.Display == DisplayInlineFlex ||
				placement.item.Style.Display == DisplayGrid ||
				placement.item.Style.Display == DisplayInlineGrid)

		if !needsStretch && !rowWasIndefinite {
			continue
		}

		relayoutSingleGridItem(ctx, placement, box, columnWidths, cellHeight, input)
	}
}

// relayoutSingleGridItem performs a second layout pass for a single
// grid item that needs stretching or percentage height resolution.
//
// It temporarily sets an explicit height on the item so that
// resolveAvailableBlockSize passes through the definite size to
// children, enabling percentage height resolution and correct
// column flex sizing.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes placement (*gridItemPlacement) which holds the item
// placement to re-lay out.
// Takes box (*LayoutBox) which is the grid container box.
// Takes columnWidths ([]float64) which holds the resolved column
// sizes.
// Takes cellHeight (float64) which specifies the resolved cell
// height in points.
func relayoutSingleGridItem(
	ctx context.Context,
	placement *gridItemPlacement,
	box *LayoutBox,
	columnWidths []float64,
	cellHeight float64,
	input layoutInput,
) {
	itemWidth := spanTrackSize(columnWidths, placement.column, placement.columnEnd, box.Style.ColumnGap)
	childEdges := resolveEdgesFromStyle(&placement.item.Style, itemWidth)
	childMarginLeft := placement.item.Style.MarginLeft.Resolve(itemWidth, 0)
	childMarginRight := placement.item.Style.MarginRight.Resolve(itemWidth, 0)

	childContentWidth := itemWidth -
		childEdges.Padding.Horizontal() -
		childEdges.Border.Horizontal() -
		childMarginLeft - childMarginRight
	if childContentWidth < 0 {
		childContentWidth = 0
	}

	isStretch := box.Style.JustifyItems == JustifyItemsStretch
	if placement.item.Style.JustifySelf != JustifySelfAuto {
		isStretch = placement.item.Style.JustifySelf == JustifySelfStretch
	}
	fixedSize := isStretch || placement.item.Style.Width.IsAuto()

	stretchedHeight := cellHeight -
		childEdges.Padding.Vertical() -
		childEdges.Border.Vertical() -
		placement.item.Style.MarginTop.Resolve(0, 0) -
		placement.item.Style.MarginBottom.Resolve(0, 0)

	originalHeight := placement.item.Style.Height
	placement.item.Style.Height = DimensionPt(stretchedHeight)
	input.Cache.Invalidate(placement.item)

	placement.fragment = layoutBox(ctx, placement.item, layoutInput{
		AvailableWidth:    childContentWidth,
		FontMetrics:       input.FontMetrics,
		Cache:             input.Cache,
		IsFixedInlineSize: fixedSize,
		Edges:             childEdges,
	})

	placement.item.Style.Height = originalHeight
}

// placeGridChild resolves the grid position for a single child
// element, handling named areas, explicit placement, semi-auto
// placement (one axis specified), and fully automatic placement.
//
// Takes child (*LayoutBox) which is the grid item to place.
// Takes templateColumns (int) which specifies the explicit column
// count.
// Takes occupied (map[[2]int]bool) which tracks occupied cells.
// Takes autoFlow (GridAutoFlowType) which specifies the
// auto-placement direction.
// Takes cursorRow (*int) which holds the row-major cursor row.
// Takes cursorColumn (*int) which holds the row-major cursor
// column.
// Takes areaMap (map[string]gridAreaBounds) which maps named
// areas to their bounds.
//
// Returns gridItemPlacement which holds the resolved position.
func placeGridChild(
	child *LayoutBox,
	templateColumns int,
	occupied map[[2]int]bool,
	autoFlow GridAutoFlowType,
	cursorRow, cursorColumn *int,
	areaMap map[string]gridAreaBounds,
) gridItemPlacement {
	if child.Style.GridArea != "" && areaMap != nil {
		if bounds, ok := areaMap[child.Style.GridArea]; ok {
			return gridItemPlacement{
				item:      child,
				column:    bounds.columnStart,
				row:       bounds.rowStart,
				columnEnd: bounds.columnEnd,
				rowEnd:    bounds.rowEnd,
			}
		}
	}

	columnStart, columnSpan := resolveGridItemColumn(&child.Style, templateColumns)
	rowStart, rowSpan := resolveGridItemRow(&child.Style)

	switch {
	case columnStart >= 0 && rowStart >= 0:

	case columnStart >= 0:
		rowStart = findNextAvailableRow(occupied, columnStart, columnSpan, 0)
	case rowStart >= 0:
		columnStart = findNextAvailableColumn(occupied, rowStart, rowSpan, 0, templateColumns)
	default:
		columnStart, rowStart = autoPlaceItem(
			occupied, columnSpan, rowSpan, templateColumns,
			autoFlow, cursorRow, cursorColumn,
		)
	}

	return gridItemPlacement{
		item:      child,
		column:    columnStart,
		row:       rowStart,
		columnEnd: columnStart + columnSpan,
		rowEnd:    rowStart + rowSpan,
	}
}

// computeGridBounds returns the maximum column and row indices
// across all placements, clamped to at least the explicit grid
// dimensions.
//
// Takes placements ([]gridItemPlacement) which holds the item
// placements.
// Takes templateColumns (int) which specifies the explicit column
// count.
// Takes templateRows (int) which specifies the explicit row count.
//
// Returns maxColumn (int) which holds the total column count.
// Returns maxRow (int) which holds the total row count.
func computeGridBounds(placements []gridItemPlacement, templateColumns, templateRows int) (maxColumn, maxRow int) {
	maxColumn = templateColumns
	maxRow = templateRows
	for _, placement := range placements {
		if placement.columnEnd > maxColumn {
			maxColumn = placement.columnEnd
		}
		if placement.rowEnd > maxRow {
			maxRow = placement.rowEnd
		}
	}
	return maxColumn, maxRow
}

// resolveGridItemColumn resolves the column start and span
// for a grid item from its computed style properties.
//
// Takes style (*ComputedStyle) which holds the grid
// placement properties.
// Takes templateColumns (int) which is the explicit
// column count for negative line resolution.
//
// Returns start (int) which is the zero-based column
// start, or -1 for auto.
// Returns span (int) which is the column span count.
func resolveGridItemColumn(style *ComputedStyle, templateColumns int) (start, span int) {
	start = -1
	span = 1

	if !style.GridColumnStart.IsAuto {
		if style.GridColumnStart.Line > 0 {
			start = style.GridColumnStart.Line - 1
		} else if style.GridColumnStart.Line < 0 {
			start = max(templateColumns+style.GridColumnStart.Line, 0)
		}
	}

	if !style.GridColumnEnd.IsAuto {
		if style.GridColumnEnd.Span > 0 {
			span = style.GridColumnEnd.Span
		} else if style.GridColumnEnd.Line > 0 && start >= 0 {
			span = max(style.GridColumnEnd.Line-1-start, 1)
		}
	} else if style.GridColumnStart.Span > 0 {
		span = style.GridColumnStart.Span
		start = -1
	}

	return start, span
}

// resolveGridItemRow resolves the row start and span for a
// grid item from its computed style properties.
//
// Takes style (*ComputedStyle) which holds the grid
// placement properties.
//
// Returns start (int) which is the zero-based row start,
// or -1 for auto.
// Returns span (int) which is the row span count.
func resolveGridItemRow(style *ComputedStyle) (start, span int) {
	start = -1
	span = 1

	if !style.GridRowStart.IsAuto {
		if style.GridRowStart.Line > 0 {
			start = style.GridRowStart.Line - 1
		}
	}

	if !style.GridRowEnd.IsAuto {
		if style.GridRowEnd.Span > 0 {
			span = style.GridRowEnd.Span
		} else if style.GridRowEnd.Line > 0 && start >= 0 {
			span = max(style.GridRowEnd.Line-1-start, 1)
		}
	} else if style.GridRowStart.Span > 0 {
		span = style.GridRowStart.Span
		start = -1
	}

	return start, span
}

// markOccupied marks all cells covered by a placement as
// occupied in the occupancy map.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes placement (gridItemPlacement) which holds the item
// position and span.
func markOccupied(occupied map[[2]int]bool, placement gridItemPlacement) {
	for row := placement.row; row < placement.rowEnd; row++ {
		for column := placement.column; column < placement.columnEnd; column++ {
			occupied[[2]int{row, column}] = true
		}
	}
}

// findNextAvailableRow searches downward from startRow for
// the first row where the given column span is unoccupied.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes column (int) which is the fixed column index.
// Takes columnSpan (int) which is the required column span.
// Takes startRow (int) which is the row to begin searching
// from.
//
// Returns int which is the first available row index.
func findNextAvailableRow(occupied map[[2]int]bool, column, columnSpan, startRow int) int {
	for row := startRow; row < startRow+gridSearchLimit; row++ {
		if isAreaAvailable(occupied, row, column, 1, columnSpan) {
			return row
		}
	}
	return startRow
}

// findNextAvailableColumn searches rightward from
// startColumn for the first column where the given row
// span is unoccupied.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes row (int) which is the fixed row index.
// Takes rowSpan (int) which is the required row span.
// Takes startColumn (int) which is the column to begin
// searching from.
// Takes maxColumns (int) which is the column search limit.
//
// Returns int which is the first available column index.
func findNextAvailableColumn(occupied map[[2]int]bool, row, rowSpan, startColumn, maxColumns int) int {
	for column := startColumn; column < maxColumns+gridColumnOverflow; column++ {
		if isAreaAvailable(occupied, row, column, rowSpan, 1) {
			return column
		}
	}
	return 0
}

// autoPlaceItem places a grid item automatically using the
// specified auto-flow direction and density mode.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes columnSpan (int) which is the item column span.
// Takes rowSpan (int) which is the item row span.
// Takes maxColumns (int) which is the column limit.
// Takes autoFlow (GridAutoFlowType) which selects the
// placement algorithm.
// Takes cursorRow (*int) which holds the current row
// cursor position.
// Takes cursorColumn (*int) which holds the current
// column cursor position.
//
// Returns column (int) which is the placed column index.
// Returns row (int) which is the placed row index.
func autoPlaceItem(
	occupied map[[2]int]bool,
	columnSpan, rowSpan, maxColumns int,
	autoFlow GridAutoFlowType,
	cursorRow, cursorColumn *int,
) (column, row int) {
	isDense := autoFlow == GridAutoFlowRowDense || autoFlow == GridAutoFlowColumnDense
	isColumnFlow := autoFlow == GridAutoFlowColumn || autoFlow == GridAutoFlowColumnDense

	if isColumnFlow {
		return autoPlaceColumnMajor(
			occupied, columnSpan, rowSpan, maxColumns,
			isDense, cursorRow, cursorColumn,
		)
	}
	return autoPlaceRowMajor(
		occupied, columnSpan, rowSpan, maxColumns,
		isDense, cursorRow, cursorColumn,
	)
}

// autoPlaceRowMajor places a grid item using row-major
// auto-placement, scanning rows then columns for a free
// slot.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes columnSpan (int) which is the item column span.
// Takes rowSpan (int) which is the item row span.
// Takes maxColumns (int) which is the column limit.
// Takes isDense (bool) which enables dense packing mode.
// Takes cursorRow (*int) which holds the current row
// cursor.
// Takes cursorColumn (*int) which holds the current
// column cursor.
//
// Returns column (int) which is the placed column index.
// Returns row (int) which is the placed row index.
func autoPlaceRowMajor(
	occupied map[[2]int]bool,
	columnSpan, rowSpan, maxColumns int,
	isDense bool,
	cursorRow, cursorColumn *int,
) (column, row int) {
	startRow := *cursorRow
	startColumn := *cursorColumn
	if isDense {
		startRow = 0
		startColumn = 0
	}

	for row := startRow; row < gridSearchLimit; row++ {
		firstColumn := 0
		if row == startRow {
			firstColumn = startColumn
		}
		col, found := findAvailableColumnInRow(occupied, row, firstColumn, maxColumns, columnSpan, rowSpan)
		if found {
			advanceRowMajorCursor(cursorRow, cursorColumn, row, col+columnSpan, maxColumns-columnSpan)
			return col, row
		}
	}
	return 0, 0
}

// findAvailableColumnInRow scans a single row for an available area
// that can fit the given column and row span.
//
// Takes occupied (map[[2]int]bool) which tracks occupied cells.
// Takes row (int) which specifies the row to scan.
// Takes firstColumn (int) which specifies the starting column.
// Takes maxColumns (int) which specifies the column limit.
// Takes columnSpan (int) which specifies the required column span.
// Takes rowSpan (int) which specifies the required row span.
//
// Returns int which holds the column index of the slot.
// Returns bool which indicates whether a slot was found.
func findAvailableColumnInRow(
	occupied map[[2]int]bool,
	row, firstColumn, maxColumns, columnSpan, rowSpan int,
) (int, bool) {
	for column := firstColumn; column <= maxColumns-columnSpan; column++ {
		if isAreaAvailable(occupied, row, column, rowSpan, columnSpan) {
			return column, true
		}
	}
	return 0, false
}

// advanceRowMajorCursor advances the row-major auto-placement cursor
// past the newly placed item, wrapping to the next row when the
// cursor exceeds the column limit.
//
// Takes cursorRow (*int) which holds the cursor row to update.
// Takes cursorColumn (*int) which holds the cursor column to
// update.
// Takes row (int) which specifies the row of the placed item.
// Takes nextColumn (int) which specifies the column after the
// placed item.
// Takes columnLimit (int) which specifies the maximum column
// before wrapping.
func advanceRowMajorCursor(cursorRow, cursorColumn *int, row, nextColumn, columnLimit int) {
	*cursorRow = row
	*cursorColumn = nextColumn
	if *cursorColumn > columnLimit {
		*cursorRow++
		*cursorColumn = 0
	}
}

// autoPlaceColumnMajor places a grid item using
// column-major auto-placement, scanning columns then rows
// for a free slot.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes columnSpan (int) which is the item column span.
// Takes rowSpan (int) which is the item row span.
// Takes isDense (bool) which enables dense packing mode.
// Takes cursorRow (*int) which holds the current row
// cursor.
// Takes cursorColumn (*int) which holds the current
// column cursor.
//
// Returns column (int) which is the placed column index.
// Returns row (int) which is the placed row index.
func autoPlaceColumnMajor(
	occupied map[[2]int]bool,
	columnSpan, rowSpan, _ int,
	isDense bool,
	cursorRow, cursorColumn *int,
) (column, row int) {
	startColumn := *cursorColumn
	startRow := *cursorRow
	if isDense {
		startColumn = 0
		startRow = 0
	}

	for column := startColumn; column < gridSearchLimit; column++ {
		firstRow := 0
		if column == startColumn {
			firstRow = startRow
		}
		for row := firstRow; row < gridSearchLimit; row++ {
			if isAreaAvailable(occupied, row, column, rowSpan, columnSpan) {
				*cursorColumn = column
				*cursorRow = row + rowSpan
				return column, row
			}
		}
	}
	return 0, 0
}

// isAreaAvailable reports whether the rectangular region
// starting at (row, column) with the given spans is
// entirely unoccupied.
//
// Takes occupied (map[[2]int]bool) which tracks occupied
// cells.
// Takes row (int) which is the start row.
// Takes column (int) which is the start column.
// Takes rowSpan (int) which is the row extent.
// Takes columnSpan (int) which is the column extent.
//
// Returns bool which is true if every cell in the area is
// free.
func isAreaAvailable(occupied map[[2]int]bool, row, column, rowSpan, columnSpan int) bool {
	for checkRow := row; checkRow < row+rowSpan; checkRow++ {
		for checkColumn := column; checkColumn < column+columnSpan; checkColumn++ {
			if occupied[[2]int{checkRow, checkColumn}] {
				return false
			}
		}
	}
	return true
}

// gridTrackSizingInput groups the parameters for grid track
// sizing, reducing the argument count of resolveGridTrackSizes.
type gridTrackSizingInput struct {
	// fontMetrics holds the font metrics port for intrinsic sizing.
	fontMetrics FontMetricsPort

	// templateTracks holds the explicit template tracks.
	templateTracks []GridTrack

	// autoTracks holds the implicit auto tracks.
	autoTracks []GridTrack

	// placements holds the grid item placements for intrinsic
	// sizing.
	placements []gridItemPlacement

	// containerSize holds the available space for the axis in
	// points.
	containerSize float64

	// gap holds the gap size between tracks in points.
	gap float64

	// trackCount holds the total number of tracks on this axis.
	trackCount int

	// isColumn holds whether this sizing is for the column axis.
	isColumn bool
}

// resolveGridTrackSizes resolves all track sizes for a
// single axis, handling fixed, intrinsic, and fraction
// tracks.
//
// Takes input (gridTrackSizingInput) which holds the
// track definitions and container size.
//
// Returns []float64 which holds the resolved track sizes
// in points.
func resolveGridTrackSizes(input gridTrackSizingInput) []float64 {
	sizes := make([]float64, input.trackCount)
	totalGap := float64(input.trackCount-1) * input.gap
	availableForTracks := input.containerSize - totalGap
	if availableForTracks < 0 {
		availableForTracks = 0
	}

	fractionTotal, fixedTotal := resolveFixedTracks(sizes, input)

	if fractionTotal > 0 {
		resolveFractionTracks(sizes, input, fractionTotal, availableForTracks-fixedTotal)
	}

	return sizes
}

// resolveFixedTracks resolves all non-fraction tracks (points,
// percentages, min/max-content, auto) and returns the total
// fraction and fixed sizes.
//
// Takes sizes ([]float64) which holds the output slice to
// populate with resolved sizes.
// Takes input (gridTrackSizingInput) which holds the track
// sizing parameters.
//
// Returns fractionTotal (float64) which holds the sum of all
// fr values.
// Returns fixedTotal (float64) which holds the sum of all
// resolved non-fr sizes.
func resolveFixedTracks(sizes []float64, input gridTrackSizingInput) (fractionTotal, fixedTotal float64) {
	for index := range input.trackCount {
		track := getTrack(input.templateTracks, input.autoTracks, index)

		switch track.Unit {
		case GridTrackPoints:
			sizes[index] = track.Value
			fixedTotal += track.Value
		case GridTrackPercentage:
			resolved := track.Value / percentageDivisor * input.containerSize
			sizes[index] = resolved
			fixedTotal += resolved
		case GridTrackMinContent:
			minWidth := computeTrackIntrinsicSize(input.placements, index, input.isColumn, true, input.fontMetrics)
			sizes[index] = minWidth
			fixedTotal += minWidth
		case GridTrackMaxContent:
			maxWidth := computeTrackIntrinsicSize(input.placements, index, input.isColumn, false, input.fontMetrics)
			sizes[index] = maxWidth
			fixedTotal += maxWidth
		case GridTrackFitContent:
			minW := computeTrackIntrinsicSize(input.placements, index, input.isColumn, true, input.fontMetrics)
			maxW := computeTrackIntrinsicSize(input.placements, index, input.isColumn, false, input.fontMetrics)
			size := math.Min(maxW, math.Max(minW, track.Value))
			sizes[index] = size
			fixedTotal += size
		case GridTrackFitContentPct:
			minW := computeTrackIntrinsicSize(input.placements, index, input.isColumn, true, input.fontMetrics)
			maxW := computeTrackIntrinsicSize(input.placements, index, input.isColumn, false, input.fontMetrics)
			clamp := track.Value / percentageDivisor * input.containerSize
			size := math.Min(maxW, math.Max(minW, clamp))
			sizes[index] = size
			fixedTotal += size
		case GridTrackFr:
			fractionTotal += track.Value
		case GridTrackAuto:
			autoSize := computeTrackIntrinsicSize(input.placements, index, input.isColumn, false, input.fontMetrics)
			sizes[index] = autoSize
			fixedTotal += autoSize
		}
	}
	return fractionTotal, fixedTotal
}

// resolveFractionTracks distributes the remaining space among
// fraction-unit tracks proportionally to their fr values.
//
// Takes sizes ([]float64) which holds the output slice to update.
// Takes input (gridTrackSizingInput) which holds the track
// sizing parameters.
// Takes fractionTotal (float64) which specifies the sum of all
// fr values.
// Takes remainingSpace (float64) which specifies the space
// available for fr tracks.
func resolveFractionTracks(sizes []float64, input gridTrackSizingInput, fractionTotal, remainingSpace float64) {
	if remainingSpace < 0 {
		remainingSpace = 0
	}
	for index := range input.trackCount {
		track := getTrack(input.templateTracks, input.autoTracks, index)
		if track.Unit == GridTrackFr {
			sizes[index] = (track.Value / fractionTotal) * remainingSpace
		}
	}
}

// getTrack returns the grid track definition for the given
// index, falling back to auto tracks or GridTrackAuto when
// out of range.
//
// Takes templateTracks ([]GridTrack) which holds the
// explicit template tracks.
// Takes autoTracks ([]GridTrack) which holds the implicit
// auto tracks.
// Takes index (int) which is the track index to look up.
//
// Returns GridTrack which is the resolved track definition.
func getTrack(templateTracks []GridTrack, autoTracks []GridTrack, index int) GridTrack {
	if index < len(templateTracks) {
		return templateTracks[index]
	}
	if len(autoTracks) > 0 {
		return autoTracks[(index-len(templateTracks))%len(autoTracks)]
	}
	return GridTrack{Unit: GridTrackAuto}
}

// computeTrackIntrinsicSize computes the intrinsic size of
// a single track based on the content of items occupying
// it.
//
// Takes placements ([]gridItemPlacement) which holds the
// grid item placements.
// Takes trackIndex (int) which is the track to measure.
// Takes isColumn (bool) which selects column or row axis.
// Takes useMinContent (bool) which selects min-content or
// max-content sizing.
// Takes fontMetrics (FontMetricsPort) which provides font
// measurement.
//
// Returns float64 which is the intrinsic track size in
// points.
func computeTrackIntrinsicSize(
	placements []gridItemPlacement,
	trackIndex int,
	isColumn bool,
	useMinContent bool,
	fontMetrics FontMetricsPort,
) float64 {
	maxSize := 0.0
	for _, placement := range placements {
		var start, end int
		if isColumn {
			start = placement.column
			end = placement.columnEnd
		} else {
			start = placement.row
			end = placement.rowEnd
		}

		if start != trackIndex || end-start != 1 {
			continue
		}

		var itemSize float64
		if useMinContent {
			itemSize = measureMinContentWidth(placement.item, fontMetrics)
		} else {
			itemSize = measureMaxContentWidth(placement.item, fontMetrics)
		}
		if itemSize > maxSize {
			maxSize = itemSize
		}
	}
	return maxSize
}

// computeRowHeights resolves all row track heights,
// handling fixed tracks, item-driven sizing, fr
// distribution, multi-span items, and auto-row stretching.
//
// Takes templateRows ([]GridTrack) which holds the
// explicit row tracks.
// Takes autoRows ([]GridTrack) which holds the implicit
// row tracks.
// Takes rowCount (int) which is the total number of rows.
// Takes placements ([]gridItemPlacement) which holds the
// grid item placements.
// Takes gap (float64) which is the row gap in points.
// Takes containerHeight (float64) which is the container
// block size for percentage and fr resolution.
//
// Returns []float64 which holds the resolved row heights
// in points.
func computeRowHeights(
	templateRows []GridTrack,
	autoRows []GridTrack,
	rowCount int,
	placements []gridItemPlacement,
	gap float64,
	containerHeight float64,
) []float64 {
	heights := make([]float64, rowCount)

	frTotal, _ := resolveFixedRowHeights(heights, templateRows, autoRows, rowCount, containerHeight)
	applyItemDrivenRowHeights(heights, templateRows, autoRows, placements)

	if frTotal > 0 && containerHeight > 0 {
		distributeFrRowSpace(heights, templateRows, autoRows, rowCount, frTotal, gap, containerHeight)
	}
	if frTotal > 0 && containerHeight <= 0 {
		applyItemDrivenFrRowHeights(heights, placements)
	}

	resolveMultiSpanRowHeights(placements, heights, gap)
	stretchAutoRowsToFill(heights, templateRows, autoRows, rowCount, gap, containerHeight, frTotal)

	return heights
}

// resolveFixedRowHeights resolves all definite row track sizes
// (points and percentages) and accumulates the total fr value.
//
// Takes heights ([]float64) which holds the output slice to
// populate.
// Takes templateRows ([]GridTrack) which holds the explicit row
// tracks.
// Takes autoRows ([]GridTrack) which holds the implicit row
// tracks.
// Takes rowCount (int) which specifies the total number of rows.
// Takes containerHeight (float64) which specifies the container
// block size for percentage resolution.
//
// Returns frTotal (float64) which holds the sum of all fr values.
// Returns fixedTotal (float64) which holds the sum of all
// resolved fixed heights.
func resolveFixedRowHeights(
	heights []float64,
	templateRows, autoRows []GridTrack,
	rowCount int,
	containerHeight float64,
) (frTotal, fixedTotal float64) {
	for index := range rowCount {
		track := getTrack(templateRows, autoRows, index)

		switch track.Unit {
		case GridTrackPoints:
			heights[index] = track.Value
			fixedTotal += track.Value
		case GridTrackPercentage:
			resolved := track.Value / percentageDivisor * containerHeight
			heights[index] = resolved
			fixedTotal += resolved
		case GridTrackFr:
			frTotal += track.Value
		default:
			heights[index] = 0
		}
	}
	return frTotal, fixedTotal
}

// applyItemDrivenRowHeights updates row heights for non-fr tracks
// using the margin-box heights of single-row-span items. Each row
// is expanded to fit its tallest item.
//
// Takes heights ([]float64) which holds the row heights to update.
// Takes templateRows ([]GridTrack) which holds the explicit row
// tracks.
// Takes autoRows ([]GridTrack) which holds the implicit row
// tracks.
// Takes placements ([]gridItemPlacement) which holds the item
// placements.
func applyItemDrivenRowHeights(
	heights []float64,
	templateRows, autoRows []GridTrack,
	placements []gridItemPlacement,
) {
	for _, placement := range placements {
		if placement.rowEnd-placement.row != 1 {
			continue
		}

		track := getTrack(templateRows, autoRows, placement.row)
		if track.Unit == GridTrackFr {
			continue
		}

		itemHeight := placement.fragment.MarginBoxHeight()
		if itemHeight > heights[placement.row] {
			heights[placement.row] = itemHeight
		}
	}
}

// distributeFrRowSpace distributes the remaining container height
// among fr-unit row tracks proportionally to their fr values.
//
// Takes heights ([]float64) which holds the row heights to update.
// Takes templateRows ([]GridTrack) which holds the explicit row
// tracks.
// Takes autoRows ([]GridTrack) which holds the implicit row
// tracks.
// Takes rowCount (int) which specifies the total number of rows.
// Takes frTotal (float64) which specifies the sum of all fr
// values.
// Takes gap (float64) which specifies the row gap size.
// Takes containerHeight (float64) which specifies the container
// block size.
func distributeFrRowSpace(
	heights []float64,
	templateRows, autoRows []GridTrack,
	rowCount int,
	frTotal, gap, containerHeight float64,
) {
	fixedTotal := 0.0
	for index := range rowCount {
		track := getTrack(templateRows, autoRows, index)
		if track.Unit != GridTrackFr {
			fixedTotal += heights[index]
		}
	}
	totalGap := float64(rowCount-1) * gap
	availableForFr := containerHeight - fixedTotal - totalGap
	if availableForFr < 0 {
		availableForFr = 0
	}
	for index := range rowCount {
		track := getTrack(templateRows, autoRows, index)
		if track.Unit == GridTrackFr {
			heights[index] = (track.Value / frTotal) * availableForFr
		}
	}
}

// applyItemDrivenFrRowHeights sizes fr tracks using item content
// heights when the container has no definite height. Each fr row
// is expanded to fit its tallest single-row-span item.
//
// Takes heights ([]float64) which holds the row heights to update.
// Takes placements ([]gridItemPlacement) which holds the item
// placements.
func applyItemDrivenFrRowHeights(heights []float64, placements []gridItemPlacement) {
	for _, placement := range placements {
		if placement.rowEnd-placement.row != 1 {
			continue
		}
		itemHeight := placement.fragment.MarginBoxHeight()
		if itemHeight > heights[placement.row] {
			heights[placement.row] = itemHeight
		}
	}
}

// stretchAutoRowsToFill distributes remaining container height
// among auto-sized rows when there is a definite container height
// and no fr tracks consume the space. This implements the
// align-content: stretch default behaviour.
//
// Takes heights ([]float64) which holds the row heights to update.
// Takes templateRows ([]GridTrack) which holds the explicit row
// tracks.
// Takes autoRows ([]GridTrack) which holds the implicit row
// tracks.
// Takes rowCount (int) which specifies the total number of rows.
// Takes gap (float64) which specifies the row gap size.
// Takes containerHeight (float64) which specifies the container
// block size.
// Takes frTotal (float64) which specifies the sum of all fr
// values.
func stretchAutoRowsToFill(
	heights []float64,
	templateRows, autoRows []GridTrack,
	rowCount int,
	gap, containerHeight, frTotal float64,
) {
	if containerHeight <= 0 || frTotal != 0 {
		return
	}

	totalGap := float64(rowCount-1) * gap
	usedHeight := totalGap
	autoCount := 0
	for index := range rowCount {
		usedHeight += heights[index]
		track := getTrack(templateRows, autoRows, index)
		if track.Unit == GridTrackAuto {
			autoCount++
		}
	}
	remaining := containerHeight - usedHeight
	if remaining > 0 && autoCount > 0 {
		perAutoRow := remaining / float64(autoCount)
		for index := range rowCount {
			track := getTrack(templateRows, autoRows, index)
			if track.Unit == GridTrackAuto {
				heights[index] += perAutoRow
			}
		}
	}
}

// resolveMultiSpanRowHeights distributes extra height needed by
// multi-row-span items across the spanned rows.
//
// Takes placements ([]gridItemPlacement) which holds the item
// placements.
// Takes heights ([]float64) which holds the row heights to update.
// Takes gap (float64) which specifies the row gap size.
func resolveMultiSpanRowHeights(placements []gridItemPlacement, heights []float64, gap float64) {
	for _, placement := range placements {
		spanCount := placement.rowEnd - placement.row
		if spanCount <= 1 {
			continue
		}

		totalSpanHeight := spanTrackSize(heights, placement.row, placement.rowEnd, gap)
		itemHeight := placement.fragment.MarginBoxHeight()
		if itemHeight > totalSpanHeight {
			deficit := itemHeight - totalSpanHeight
			perRow := deficit / float64(spanCount)
			for row := placement.row; row < placement.rowEnd; row++ {
				heights[row] += perRow
			}
		}
	}
}

// spanTrackSize returns the total size of tracks from
// start (inclusive) to end (exclusive), including
// inter-track gaps.
//
// Takes sizes ([]float64) which holds the track sizes.
// Takes start (int) which is the first track index.
// Takes end (int) which is the exclusive end track index.
// Takes gap (float64) which is the inter-track gap size.
//
// Returns float64 which is the total spanned size in
// points.
func spanTrackSize(sizes []float64, start, end int, gap float64) float64 {
	total := 0.0
	for index := start; index < end && index < len(sizes); index++ {
		total += sizes[index]
		if index > start {
			total += gap
		}
	}
	return total
}

// positionGridItems positions all grid item fragments at
// their resolved grid coordinates, applying alignment and
// RTL mirroring.
//
// Takes box (*LayoutBox) which is the grid container.
// Takes placements ([]gridItemPlacement) which holds the
// item placements.
// Takes columnWidths ([]float64) which holds the resolved
// column widths.
// Takes rowHeights ([]float64) which holds the resolved
// row heights.
// Takes containerWidth (float64) which is the container
// width for RTL mirroring.
//
// Returns []*Fragment which holds the positioned child
// fragments.
func positionGridItems(
	box *LayoutBox,
	placements []gridItemPlacement,
	columnWidths []float64,
	rowHeights []float64,
	containerWidth float64,
) []*Fragment {
	contentOffsetX := 0.0
	contentOffsetY := 0.0
	columnOffsets := computeTrackOffsets(columnWidths, box.Style.ColumnGap)
	rowOffsets := computeTrackOffsets(rowHeights, box.Style.RowGap)

	isRTL := box.Style.Direction == DirectionRTL

	childFragments := make([]*Fragment, 0, len(placements))
	for _, placement := range placements {
		itemX := columnOffsets[placement.column]
		itemY := rowOffsets[placement.row]

		cellWidth := spanTrackSize(columnWidths, placement.column, placement.columnEnd, box.Style.ColumnGap)
		cellHeight := spanTrackSize(rowHeights, placement.row, placement.rowEnd, box.Style.RowGap)

		if isRTL {
			itemX = containerWidth - itemX - cellWidth
		}

		alignGridItem(placement.fragment, gridAlignmentInput{
			cellX: itemX, cellY: itemY,
			cellWidth: cellWidth, cellHeight: cellHeight,
			contentOffsetX: contentOffsetX, contentOffsetY: contentOffsetY,
		}, &box.Style)
		childFragments = append(childFragments, placement.fragment)
	}
	return childFragments
}

// computeTrackOffsets computes cumulative offsets for each
// track, producing a slice one longer than sizes where
// each entry is the start position of that track.
//
// Takes sizes ([]float64) which holds the track sizes.
// Takes gap (float64) which is the inter-track gap size.
//
// Returns []float64 which holds the cumulative offsets.
func computeTrackOffsets(sizes []float64, gap float64) []float64 {
	offsets := make([]float64, len(sizes)+1)
	for index, size := range sizes {
		offsets[index+1] = offsets[index] + size
		if index < len(sizes)-1 {
			offsets[index+1] += gap
		}
	}
	return offsets
}
