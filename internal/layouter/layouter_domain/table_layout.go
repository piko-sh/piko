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

// tableCellPlacement tracks the grid position and span of a
// placed table cell.
type tableCellPlacement struct {
	// cell holds the layout box for the table cell element.
	cell *LayoutBox

	// fragment holds the laid-out fragment for the cell.
	fragment *Fragment

	// column holds the zero-based grid column index of the cell.
	column int

	// row holds the zero-based grid row index of the cell.
	row int

	// colspan holds the number of columns the cell spans.
	colspan int

	// rowspan holds the number of rows the cell spans.
	rowspan int
}

// layoutTableContainer performs CSS table layout on a table box,
// returning a Fragment with parent-relative child offsets.
//
// Takes box (*LayoutBox) which is the table container to lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics from the parent context.
//
// Returns formattingContextResult which holds the table layout output.
func layoutTableContainer(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	tableWidth := input.AvailableWidth

	rows, columnCount := collectTableRows(box)
	if len(rows) == 0 {
		return formattingContextResult{
			Margin: BoxEdges{
				Top:    input.Edges.MarginTop,
				Bottom: input.Edges.MarginBottom,
			},
		}
	}

	placements, rowCount, gridColumnCount := buildTableGrid(rows)
	if gridColumnCount > columnCount {
		columnCount = gridColumnCount
	}

	spacing := box.Style.BorderSpacing
	isCollapsed := box.Style.BorderCollapse == BorderCollapseCollapse
	if isCollapsed {
		spacing = 0
		halveTableCellBorders(placements)
	}
	columnWidths := computeColumnWidths(box, rows, placements, columnCount, tableWidth, spacing, input.FontMetrics)

	rowHeights := sizeCellsAndComputeRowHeights(ctx, placements, columnWidths, rowCount, spacing, input)

	childFragments := buildTableChildFragments(tableChildInput{
		placements:   placements,
		rows:         rows,
		columnWidths: columnWidths,
		rowHeights:   rowHeights,
		rowCount:     rowCount,
		spacing:      spacing,
		tableWidth:   tableWidth,
	})

	totalHeight := computeTableTotalHeight(rowHeights, rowCount, spacing)

	result := formattingContextResult{
		Children:      childFragments,
		ContentHeight: totalHeight,
		Margin: BoxEdges{
			Top:    input.Edges.MarginTop,
			Bottom: input.Edges.MarginBottom,
		},
	}

	if isCollapsed {
		result.Border = collapsedTableOuterBorder(box, placements, rowCount, columnCount)
		result.HasBorder = true
	}

	return result
}

// sizeCellsAndComputeRowHeights lays out each table cell and
// determines the height of each row from the tallest single-span
// cell, then distributes extra height for rowspan cells.
//
// Takes placements ([]tableCellPlacement) which are the grid placements.
// Takes columnWidths ([]float64) which holds the resolved column widths.
// Takes rowCount (int) which is the number of grid rows.
// Takes spacing (float64) which is the border-spacing value.
// Takes input (layoutInput) which carries font metrics and cache.
//
// Returns []float64 which holds the computed height for each row.
func sizeCellsAndComputeRowHeights(
	ctx context.Context,
	placements []tableCellPlacement,
	columnWidths []float64,
	rowCount int,
	spacing float64,
	input layoutInput,
) []float64 {
	rowHeights := make([]float64, rowCount)
	for index := range placements {
		placement := &placements[index]
		cellWidth := spanWidth(columnWidths, placement.column, placement.colspan, spacing)
		fragment, totalHeight := sizeTableCell(ctx, placement.cell, cellWidth, input)

		placement.fragment = fragment

		if placement.rowspan == 1 && totalHeight > rowHeights[placement.row] {
			rowHeights[placement.row] = totalHeight
		}
	}
	distributeRowspanHeights(placements, rowHeights, spacing)
	return rowHeights
}

// tableChildInput groups the parameters for building table
// child fragments, reducing the argument count of
// buildTableChildFragments.
type tableChildInput struct {
	// placements holds the grid placements for all table cells.
	placements []tableCellPlacement

	// rows holds the collected table rows.
	rows []tableRow

	// columnWidths holds the resolved width for each column.
	columnWidths []float64

	// rowHeights holds the resolved height for each row.
	rowHeights []float64

	// rowCount holds the total number of grid rows.
	rowCount int

	// spacing holds the border-spacing value.
	spacing float64

	// tableWidth holds the available table width.
	tableWidth float64
}

// buildTableChildFragments positions all cell and row fragments
// within the table grid.
//
// Takes input (tableChildInput) which holds the placements, widths,
// heights, and spacing needed for positioning.
//
// Returns []*Fragment which holds the positioned cell and row fragments.
func buildTableChildFragments(input tableChildInput) []*Fragment {
	columnXOffsets := computeColumnXOffsets(input.columnWidths, 0, input.spacing)
	rowYOffsets := computeRowYOffsets(input.rowHeights, input.spacing, input.spacing)

	childFragments := make([]*Fragment, 0, len(input.placements)+len(input.rows))
	layoutTableCells(input.placements, input.rowHeights, columnXOffsets, rowYOffsets, input.spacing, &childFragments)
	appendTableRowFragments(input.rows, input.rowCount, rowYOffsets, input.rowHeights, input.spacing, input.tableWidth, &childFragments)
	return childFragments
}

// computeTableTotalHeight sums row heights and inter-row spacing
// to produce the overall table content height.
//
// Takes rowHeights ([]float64) which holds the height of each row.
// Takes rowCount (int) which is the number of rows.
// Takes spacing (float64) which is the border-spacing value.
//
// Returns float64 which is the total table content height.
func computeTableTotalHeight(rowHeights []float64, rowCount int, spacing float64) float64 {
	totalHeight := float64(rowCount+1) * spacing
	for _, rowHeight := range rowHeights {
		totalHeight += rowHeight
	}
	return totalHeight
}

// layoutTableCells positions each cell fragment within the table
// grid using column X offsets and row Y offsets.
//
// Takes placements ([]tableCellPlacement) which are the grid placements.
// Takes rowHeights ([]float64) which holds the row heights.
// Takes columnXOffsets ([]float64) which holds the X origin per column.
// Takes rowYOffsets ([]float64) which holds the Y origin per row.
// Takes spacing (float64) which is the border-spacing value.
// Takes childFragments (*[]*Fragment) which receives the positioned fragments.
func layoutTableCells(
	placements []tableCellPlacement,
	rowHeights []float64,
	columnXOffsets, rowYOffsets []float64,
	spacing float64,
	childFragments *[]*Fragment,
) {
	for _, placement := range placements {
		cellHeight := spanHeight(rowHeights, placement.row, placement.rowspan, spacing)

		cellContentHeight := cellHeight -
			placement.fragment.Padding.Vertical() - placement.fragment.Border.Vertical()
		if cellContentHeight > placement.fragment.ContentHeight {
			placement.fragment.ContentHeight = cellContentHeight
		}

		placement.fragment.OffsetX = columnXOffsets[placement.column] +
			placement.fragment.Padding.Left + placement.fragment.Border.Left
		placement.fragment.OffsetY = rowYOffsets[placement.row] +
			placement.fragment.Padding.Top + placement.fragment.Border.Top

		applyTableCellVerticalAlignFragment(placement.fragment, cellHeight)
		*childFragments = append(*childFragments, placement.fragment)
	}
}

// appendTableRowFragments constructs row fragments for table rows
// that have an associated row box and appends them to the output
// slice.
//
// Takes rows ([]tableRow) which are the collected table rows.
// Takes rowCount (int) which is the number of grid rows.
// Takes rowYOffsets ([]float64) which holds the Y origin per row.
// Takes rowHeights ([]float64) which holds the height per row.
// Takes spacing (float64) which is the border-spacing value.
// Takes tableWidth (float64) which is the total table width.
// Takes childFragments (*[]*Fragment) which receives the row fragments.
func appendTableRowFragments(
	rows []tableRow, rowCount int,
	rowYOffsets, rowHeights []float64,
	spacing, tableWidth float64,
	childFragments *[]*Fragment,
) {
	for rowIndex, row := range rows {
		if rowIndex >= rowCount {
			break
		}
		if row.box != nil {
			rowFragment := &Fragment{
				Box:           row.box,
				OffsetX:       spacing,
				OffsetY:       rowYOffsets[rowIndex],
				ContentWidth:  tableWidth - 2*spacing,
				ContentHeight: rowHeights[rowIndex],
			}
			*childFragments = append(*childFragments, rowFragment)
		}
	}
}

// buildTableGrid places cells into a 2D grid, respecting
// colspan and rowspan attributes. Returns the cell placements,
// total row count, and total column count.
//
// Takes rows ([]tableRow) which are the collected table rows.
//
// Returns cells ([]tableCellPlacement) listing all placed cells.
// Returns columnCount (int) which is the maximum grid column count.
// Returns rowCount (int) which is the total number of grid rows.
func buildTableGrid(rows []tableRow) (cells []tableCellPlacement, columnCount, rowCount int) {
	occupied := make(map[[2]int]bool)
	var placements []tableCellPlacement
	maxColumn := 0

	for rowIndex, row := range rows {
		columnCursor := 0
		for _, cell := range row.cells {
			for occupied[[2]int{rowIndex, columnCursor}] {
				columnCursor++
			}

			placement, endColumn := placeSingleCell(cell, rowIndex, columnCursor, occupied)
			placements = append(placements, placement)

			if endColumn > maxColumn {
				maxColumn = endColumn
			}
			columnCursor = endColumn
		}
	}

	return placements, len(rows), maxColumn
}

// placeSingleCell creates a placement for one cell at the given grid
// position and marks the occupied cells in the grid map.
//
// Takes cell (*LayoutBox) which is the cell to place.
// Takes rowIndex (int) which is the grid row for the cell.
// Takes columnCursor (int) which is the starting grid column.
// Takes occupied (map[[2]int]bool) which tracks occupied grid positions.
//
// Returns tableCellPlacement which is the cell's grid placement.
// Returns int which is the column index after the cell.
func placeSingleCell(
	cell *LayoutBox, rowIndex, columnCursor int,
	occupied map[[2]int]bool,
) (tableCellPlacement, int) {
	colspan := max(cell.Colspan, 1)
	rowspan := max(cell.Rowspan, 1)

	placement := tableCellPlacement{
		cell:    cell,
		column:  columnCursor,
		row:     rowIndex,
		colspan: colspan,
		rowspan: rowspan,
	}

	for spanRow := range rowspan {
		for spanCol := range colspan {
			occupied[[2]int{rowIndex + spanRow, columnCursor + spanCol}] = true
		}
	}

	return placement, columnCursor + colspan
}

// adjustColumnWidthsForColspan ensures that spanning cells'
// content widths are satisfied by growing spanned columns when
// their combined width falls short.
//
// Takes placements ([]tableCellPlacement) which are the grid
// placements.
// Takes columnWidths ([]float64) which are the column widths
// to adjust.
// Takes columnCount (int) which is the number of columns.
// Takes spacing (float64) which is the border-spacing value.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement.
func adjustColumnWidthsForColspan(
	placements []tableCellPlacement,
	columnWidths []float64,
	columnCount int,
	spacing float64,
	fontMetrics FontMetricsPort,
) {
	for _, placement := range placements {
		if placement.colspan <= 1 {
			continue
		}

		cellContentWidth := colspanCellBorderBoxWidth(placement, fontMetrics)
		currentSpanWidth := spanWidth(columnWidths, placement.column, placement.colspan, spacing)
		deficit := cellContentWidth - currentSpanWidth
		if deficit <= 0 {
			continue
		}

		perColumn := deficit / float64(placement.colspan)
		for columnIndex := placement.column; columnIndex < placement.column+placement.colspan && columnIndex < columnCount; columnIndex++ {
			columnWidths[columnIndex] += perColumn
		}
	}
}

// colspanCellBorderBoxWidth computes the border-box width needed for
// a colspan cell. It takes the larger of the measured content width
// and any explicit CSS width, both including padding and border.
//
// Takes placement (tableCellPlacement) which is the cell placement.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the border-box width.
func colspanCellBorderBoxWidth(placement tableCellPlacement, fontMetrics FontMetricsPort) float64 {
	style := &placement.cell.Style
	horizontalExtra := style.PaddingLeft + style.PaddingRight +
		style.BorderLeftWidth + style.BorderRightWidth

	cellContentWidth := measureCellContentWidth(placement.cell, fontMetrics) + horizontalExtra

	if !style.Width.IsAuto() && !style.Width.IsIntrinsic() {
		explicitBorderBox := resolveExplicitBorderBox(style)
		if explicitBorderBox > cellContentWidth {
			cellContentWidth = explicitBorderBox
		}
	}

	return cellContentWidth
}

// resolveExplicitBorderBox resolves the explicit CSS width of a cell
// to a border-box value, adding padding and border widths when the
// box-sizing model is content-box.
//
// Takes style (*ComputedStyle) which is the cell's computed style.
//
// Returns float64 which is the resolved border-box width.
func resolveExplicitBorderBox(style *ComputedStyle) float64 {
	if style.BoxSizing == BoxSizingBorderBox {
		return style.Width.Resolve(0, 0)
	}
	return style.Width.Resolve(0, 0) +
		style.PaddingLeft + style.PaddingRight +
		style.BorderLeftWidth + style.BorderRightWidth
}

// spanWidth returns the total width allocated to a span of
// columns including intermediate spacing.
//
// Takes columnWidths ([]float64) which holds column widths.
// Takes startColumn (int) which is the first column index.
// Takes colspan (int) which is the number of columns spanned.
// Takes spacing (float64) which is the border-spacing value.
//
// Returns float64 which is the total span width.
func spanWidth(columnWidths []float64, startColumn, colspan int, spacing float64) float64 {
	total := 0.0
	endColumn := min(startColumn+colspan, len(columnWidths))
	for columnIndex := startColumn; columnIndex < endColumn; columnIndex++ {
		total += columnWidths[columnIndex]
	}
	if colspan > 1 {
		total += spacing * float64(colspan-1)
	}
	return total
}

// spanHeight returns the total height allocated to a span of
// rows including intermediate spacing.
//
// Takes rowHeights ([]float64) which holds row heights.
// Takes startRow (int) which is the first row index.
// Takes rowspan (int) which is the number of rows spanned.
// Takes spacing (float64) which is the border-spacing value.
//
// Returns float64 which is the total span height.
func spanHeight(rowHeights []float64, startRow, rowspan int, spacing float64) float64 {
	total := 0.0
	endRow := min(startRow+rowspan, len(rowHeights))
	for rowIndex := startRow; rowIndex < endRow; rowIndex++ {
		total += rowHeights[rowIndex]
	}
	if rowspan > 1 {
		total += spacing * float64(rowspan-1)
	}
	return total
}

// distributeRowspanHeights distributes extra height needed by
// rowspan cells across the spanned rows.
//
// Takes placements ([]tableCellPlacement) which are the grid
// placements.
// Takes rowHeights ([]float64) which are the row heights to
// adjust.
// Takes spacing (float64) which is the border-spacing value.
func distributeRowspanHeights(placements []tableCellPlacement, rowHeights []float64, spacing float64) {
	for _, placement := range placements {
		if placement.rowspan <= 1 {
			continue
		}

		cellHeight := placement.fragment.BorderBoxHeight()
		currentSpanHeight := spanHeight(rowHeights, placement.row, placement.rowspan, spacing)

		deficit := cellHeight - currentSpanHeight
		if deficit <= 0 {
			continue
		}

		perRow := deficit / float64(placement.rowspan)
		endRow := min(placement.row+placement.rowspan, len(rowHeights))
		for rowIndex := placement.row; rowIndex < endRow; rowIndex++ {
			rowHeights[rowIndex] += perRow
		}
	}
}

// computeColumnXOffsets computes the X origin for each column.
//
// Takes columnWidths ([]float64) which holds column widths.
// Takes contentStartX (float64) which is the table content
// origin X.
// Takes spacing (float64) which is the border-spacing value.
//
// Returns []float64 which holds the X offset per column.
func computeColumnXOffsets(columnWidths []float64, contentStartX, spacing float64) []float64 {
	offsets := make([]float64, len(columnWidths))
	cursor := contentStartX + spacing
	for columnIndex := range columnWidths {
		offsets[columnIndex] = cursor
		cursor += columnWidths[columnIndex] + spacing
	}
	return offsets
}

// computeRowYOffsets computes the Y origin for each row.
//
// Takes rowHeights ([]float64) which holds row heights.
// Takes startY (float64) which is the table content origin Y.
// Takes spacing (float64) which is the border-spacing value.
//
// Returns []float64 which holds the Y offset per row.
func computeRowYOffsets(rowHeights []float64, startY, spacing float64) []float64 {
	offsets := make([]float64, len(rowHeights))
	cursor := startY
	for rowIndex := range rowHeights {
		offsets[rowIndex] = cursor
		cursor += rowHeights[rowIndex] + spacing
	}
	return offsets
}

// applyTableCellVerticalAlignFragment adjusts a cell
// fragment's Y offset based on the vertical-align property.
//
// Takes fragment (*Fragment) which is the cell fragment to
// align.
// Takes availableHeight (float64) which is the total height
// available to the cell.
func applyTableCellVerticalAlignFragment(fragment *Fragment, availableHeight float64) {
	cellHeight := fragment.BorderBoxHeight()

	switch fragment.Box.Style.VerticalAlign {
	case VerticalAlignBaseline, VerticalAlignTop:
		return
	case VerticalAlignMiddle:
		fragment.OffsetY += (availableHeight - cellHeight) / 2
	case VerticalAlignBottom:
		fragment.OffsetY += availableHeight - cellHeight
	}
}

// sizeTableCell resolves edges, sizes, and lays out a single table
// cell's content, returning a Fragment with children positioned
// relative to the cell and the cell's border-box height.
//
// Takes cell (*LayoutBox) which is the cell to size.
// Takes columnWidth (float64) which is the allocated column width.
// Takes input (layoutInput) which carries the available width,
// font metrics, and cache from the parent context.
//
// Returns *Fragment which holds the cell's layout results.
// Returns float64 which is the total cell height including padding
// and borders.
func sizeTableCell(ctx context.Context, cell *LayoutBox, columnWidth float64, input layoutInput) (*Fragment, float64) {
	edges := resolveEdgesFromStyle(&cell.Style, columnWidth)

	cellContentWidth := columnWidth - edges.Padding.Horizontal() - edges.Border.Horizontal()
	if cellContentWidth < 0 {
		cellContentWidth = 0
	}

	fragment := layoutBox(ctx, cell, layoutInput{
		AvailableWidth:    cellContentWidth,
		FontMetrics:       input.FontMetrics,
		Cache:             input.Cache,
		IsFixedInlineSize: true,
		Edges:             edges,
	})

	if !cell.Style.Height.IsAuto() {
		explicitContentHeight := adjustForBoxSizing(
			cell.Style.Height.Resolve(0, fragment.ContentHeight),
			&cell.Style, false,
		)
		if explicitContentHeight > fragment.ContentHeight {
			fragment.ContentHeight = explicitContentHeight
		}
	}

	borderBoxHeight := fragment.BorderBoxHeight()
	return fragment, borderBoxHeight
}

// tableRow groups a row box with its child cell boxes.
type tableRow struct {
	// box is the LayoutBox for the row element, or nil when the
	// row is synthesised from a bare cell.
	box *LayoutBox

	// cells holds the cell boxes belonging to this row.
	cells []*LayoutBox
}

// collectTableRows gathers all rows from a table box, handling
// direct row children, row groups, and bare cells.
//
// Takes table (*LayoutBox) which is the table container to scan.
//
// Returns []tableRow which lists the collected rows.
// Returns int which is the maximum column count across all rows.
func collectTableRows(table *LayoutBox) ([]tableRow, int) {
	var rows []tableRow
	maxColumns := 0

	for _, child := range table.Children {
		switch child.Type {
		case BoxTableRow:
			row := collectCellsFromRow(child)
			rows = append(rows, row)
			maxColumns = maxColumnCount(maxColumns, countRowColumns(row))
		case BoxTableRowGroup:
			groupRows := collectRowsFromGroup(child)
			for _, row := range groupRows {
				rows = append(rows, row)
				maxColumns = maxColumnCount(maxColumns, countRowColumns(row))
			}
		case BoxTableCell:
			rows = append(rows, tableRow{cells: []*LayoutBox{child}})
			maxColumns = maxColumnCount(maxColumns, max(child.Colspan, 1))
		}
	}

	return rows, maxColumns
}

// countRowColumns returns the total number of grid columns
// consumed by the cells in a row, accounting for colspan.
//
// Takes row (tableRow) which is the row to count.
//
// Returns int which is the total column count.
func countRowColumns(row tableRow) int {
	total := 0
	for _, cell := range row.cells {
		total += max(cell.Colspan, 1)
	}
	return total
}

// collectRowsFromGroup extracts table rows from a row group
// element such as thead, tbody, or tfoot.
//
// Takes group (*LayoutBox) which is the row group to scan.
//
// Returns []tableRow which lists the rows found in the group.
func collectRowsFromGroup(group *LayoutBox) []tableRow {
	var rows []tableRow
	for _, groupChild := range group.Children {
		if groupChild.Type == BoxTableRow {
			rows = append(rows, collectCellsFromRow(groupChild))
		}
	}
	return rows
}

// maxColumnCount returns the larger of two column counts.
//
// Takes current (int) which is the existing maximum.
// Takes candidate (int) which is the new value to compare.
//
// Returns int which is the larger of the two values.
func maxColumnCount(current, candidate int) int {
	if candidate > current {
		return candidate
	}
	return current
}

// collectCellsFromRow builds a tableRow by collecting all cell
// children from a row box.
//
// Takes rowBox (*LayoutBox) which is the row to extract cells from.
//
// Returns tableRow which pairs the row box with its cells.
func collectCellsFromRow(rowBox *LayoutBox) tableRow {
	row := tableRow{box: rowBox}
	for _, child := range rowBox.Children {
		if child.Type == BoxTableCell {
			row.cells = append(row.cells, child)
		}
	}
	return row
}

// computeColumnWidths selects the fixed or auto column width
// algorithm based on the table-layout property.
//
// Takes table (*LayoutBox) which provides the table-layout style.
// Takes rows ([]tableRow) which are the table rows for auto
// sizing.
// Takes columnCount (int) which is the number of columns.
// Takes tableWidth (float64) which is the available table width.
// Takes spacing (float64) which is the border-spacing value.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement for auto sizing.
//
// Returns []float64 which holds the computed width for each column.
func computeColumnWidths(
	table *LayoutBox,
	rows []tableRow,
	placements []tableCellPlacement,
	columnCount int,
	tableWidth, spacing float64,
	fontMetrics FontMetricsPort,
) []float64 {
	if columnCount == 0 {
		return nil
	}

	if table.Style.TableLayout == TableLayoutFixed {
		return computeFixedColumnWidths(columnCount, tableWidth, spacing)
	}

	return computeAutoColumnWidths(rows, placements, columnCount, tableWidth, spacing, fontMetrics)
}

// computeFixedColumnWidths divides the available width equally
// among all columns for fixed table layout.
//
// Takes columnCount (int) which is the number of columns.
// Takes tableWidth (float64) which is the total table width.
// Takes spacing (float64) which is the border-spacing value.
//
// Returns []float64 which holds equal widths for each column.
func computeFixedColumnWidths(columnCount int, tableWidth, spacing float64) []float64 {
	availableWidth := tableWidth - spacing*float64(columnCount+1)
	if availableWidth < 0 {
		availableWidth = 0
	}

	columnWidth := availableWidth / float64(columnCount)
	widths := make([]float64, columnCount)
	for index := range widths {
		widths[index] = columnWidth
	}
	return widths
}

// computeAutoColumnWidths measures cell content and distributes
// available width proportionally for auto table layout.
//
// Takes rows ([]tableRow) which are the table rows to measure.
// Takes columnCount (int) which is the number of columns.
// Takes tableWidth (float64) which is the total table width.
// Takes spacing (float64) which is the border-spacing value.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement.
//
// Returns []float64 which holds the computed width for each column.
func computeAutoColumnWidths(
	rows []tableRow,
	placements []tableCellPlacement,
	columnCount int,
	tableWidth, spacing float64,
	fontMetrics FontMetricsPort,
) []float64 {
	minWidths, preferredWidths := measureColumnWidths(rows, columnCount, fontMetrics)

	adjustColumnWidthsForColspan(placements, preferredWidths, columnCount, spacing, fontMetrics)

	availableWidth := tableWidth - spacing*float64(columnCount+1)
	if availableWidth < 0 {
		availableWidth = 0
	}

	return distributeColumnWidths(minWidths, preferredWidths, columnCount, availableWidth)
}

// measureColumnWidths computes the minimum content width and
// preferred width for each column across all rows.
//
// The minimum width is the border-box width needed to fit the
// cell's content (text + padding + border). The preferred width
// is the maximum of the minimum and any explicit CSS width.
//
// Takes rows ([]tableRow) which are the rows to measure.
// Takes columnCount (int) which is the number of columns.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement.
//
// Returns []float64 holding the minimum content width per column.
// Returns []float64 holding the preferred width per column.
func measureColumnWidths(rows []tableRow, columnCount int, fontMetrics FontMetricsPort) (minWidths, preferredWidths []float64) {
	minWidths = make([]float64, columnCount)
	preferredWidths = make([]float64, columnCount)

	occupied := make(map[[2]int]bool)
	for rowIndex, row := range rows {
		measureRowCellWidths(row, rowIndex, columnCount, occupied, fontMetrics, minWidths, preferredWidths)
	}

	return minWidths, preferredWidths
}

// measureRowCellWidths walks the cells of a single row, marks their
// grid positions as occupied, and measures single-column-span cells
// to update the minimum and preferred width arrays.
//
// Takes row (tableRow) which is the row to measure.
// Takes rowIndex (int) which is the row's grid index.
// Takes columnCount (int) which is the number of columns.
// Takes occupied (map[[2]int]bool) which tracks occupied grid positions.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
// Takes minWidths ([]float64) which receives the minimum widths.
// Takes preferredWidths ([]float64) which receives the preferred widths.
func measureRowCellWidths(
	row tableRow, rowIndex, columnCount int,
	occupied map[[2]int]bool,
	fontMetrics FontMetricsPort,
	minWidths, preferredWidths []float64,
) {
	columnCursor := 0
	for _, cell := range row.cells {
		for occupied[[2]int{rowIndex, columnCursor}] {
			columnCursor++
		}
		if columnCursor >= columnCount {
			break
		}

		colspan := max(cell.Colspan, 1)
		rowspan := max(cell.Rowspan, 1)
		markTableOccupied(occupied, rowIndex, columnCursor, rowspan, colspan)

		if colspan == 1 {
			measureSingleColumnCell(cell, columnCursor, fontMetrics, minWidths, preferredWidths)
		}

		columnCursor += colspan
	}
}

// markTableOccupied marks all grid cells covered by a cell's
// rowspan and colspan as occupied in the grid map.
//
// Takes occupied (map[[2]int]bool) which is the grid occupancy map.
// Takes rowIndex (int) which is the cell's starting row.
// Takes columnCursor (int) which is the cell's starting column.
// Takes rowspan (int) which is the number of rows spanned.
// Takes colspan (int) which is the number of columns spanned.
func markTableOccupied(occupied map[[2]int]bool, rowIndex, columnCursor, rowspan, colspan int) {
	for spanRow := range rowspan {
		for spanCol := range colspan {
			occupied[[2]int{rowIndex + spanRow, columnCursor + spanCol}] = true
		}
	}
}

// measureSingleColumnCell measures the content and preferred widths
// of a single-column-span cell and updates the column width arrays.
//
// Takes cell (*LayoutBox) which is the cell to measure.
// Takes column (int) which is the column index for the cell.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
// Takes minWidths ([]float64) which receives the minimum widths.
// Takes preferredWidths ([]float64) which receives the preferred widths.
func measureSingleColumnCell(
	cell *LayoutBox, column int,
	fontMetrics FontMetricsPort,
	minWidths, preferredWidths []float64,
) {
	contentWidth := measureCellContentWidth(cell, fontMetrics) +
		cell.Style.PaddingLeft + cell.Style.PaddingRight +
		cell.Style.BorderLeftWidth + cell.Style.BorderRightWidth

	cellWidth := contentWidth
	if !cell.Style.Width.IsAuto() && !cell.Style.Width.IsIntrinsic() {
		var explicitBorderBox float64
		if cell.Style.BoxSizing == BoxSizingBorderBox {
			explicitBorderBox = cell.Style.Width.Resolve(0, 0)
		} else {
			explicitBorderBox = cell.Style.Width.Resolve(0, 0) +
				cell.Style.PaddingLeft + cell.Style.PaddingRight +
				cell.Style.BorderLeftWidth + cell.Style.BorderRightWidth
		}
		if explicitBorderBox > cellWidth {
			cellWidth = explicitBorderBox
		}
	}

	if contentWidth > minWidths[column] {
		minWidths[column] = contentWidth
	}
	if cellWidth > preferredWidths[column] {
		preferredWidths[column] = cellWidth
	}
}

// distributeColumnWidths allocates available width to columns
// using a two-phase approach matching browser behaviour.
//
// First, minimum content widths are allocated. Then remaining
// space is distributed proportionally to the gap between each
// column's preferred and minimum width.
//
// Takes minWidths ([]float64) holding the minimum content width
// per column.
// Takes preferredWidths ([]float64) holding the preferred width
// per column.
// Takes columnCount (int) which is the number of columns.
// Takes availableWidth (float64) which is the space to distribute.
//
// Returns []float64 which holds the final width for each column.
func distributeColumnWidths(minWidths, preferredWidths []float64, columnCount int, availableWidth float64) []float64 {
	widths := make([]float64, columnCount)

	totalPreferred := 0.0
	totalMin := 0.0
	for index := range widths {
		totalPreferred += preferredWidths[index]
		totalMin += minWidths[index]
	}

	if totalPreferred <= 0 {
		equalWidth := availableWidth / float64(columnCount)
		for index := range widths {
			widths[index] = equalWidth
		}
		return widths
	}

	if totalPreferred <= availableWidth {
		for index := range widths {
			widths[index] = preferredWidths[index] / totalPreferred * availableWidth
		}
		return widths
	}

	if totalMin >= availableWidth {
		for index := range widths {
			widths[index] = math.Max(0, preferredWidths[index]/totalPreferred*availableWidth)
		}
		return widths
	}

	remaining := availableWidth - totalMin
	totalGap := totalPreferred - totalMin
	for index := range widths {
		widths[index] = minWidths[index]
		if totalGap > 0 {
			gap := preferredWidths[index] - minWidths[index]
			widths[index] += gap / totalGap * remaining
		}
	}
	return widths
}

// measureCellContentWidth measures the intrinsic content width
// of a table cell by examining all children, including text
// runs, nested tables, and block-level elements.
//
// Takes cell (*LayoutBox) which is the cell to measure.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement.
//
// Returns float64 which is the intrinsic content width.
func measureCellContentWidth(cell *LayoutBox, fontMetrics FontMetricsPort) float64 {
	maxWidth := 0.0
	for _, child := range cell.Children {
		var childWidth float64
		switch {
		case child.Type == BoxTextRun:
			font := FontDescriptor{
				Family: child.Style.FontFamily,
				Weight: child.Style.FontWeight,
				Style:  child.Style.FontStyle,
			}
			text := applyTextTransform(child.Text, child.Style.TextTransform)
			childWidth = fontMetrics.MeasureText(font, child.Style.FontSize, text, child.Style.Direction)
		case !child.Style.Width.IsAuto() && !child.Style.Width.IsIntrinsic():
			if child.Style.BoxSizing == BoxSizingBorderBox {
				childWidth = child.Style.Width.Resolve(0, 0)
			} else {
				childWidth = child.Style.Width.Resolve(0, 0) +
					child.Style.PaddingLeft + child.Style.PaddingRight +
					child.Style.BorderLeftWidth + child.Style.BorderRightWidth
			}
		}
		if childWidth > maxWidth {
			maxWidth = childWidth
		}
	}
	return maxWidth
}

// halveTableCellBorders halves all border widths on each
// cell's computed style for collapsed border model. In the
// collapsed model, adjacent cells share a single border so
// each cell owns half.
//
// Takes placements ([]tableCellPlacement) which are the
// grid placements whose cell styles will be modified.
func halveTableCellBorders(placements []tableCellPlacement) {
	for _, placement := range placements {
		placement.cell.Style.BorderTopWidth /= 2
		placement.cell.Style.BorderBottomWidth /= 2
		placement.cell.Style.BorderLeftWidth /= 2
		placement.cell.Style.BorderRightWidth /= 2
	}
}

// collapsedTableOuterBorder computes the table's outer
// border for the collapsed border model. The outer border
// on each edge is the maximum of the table's own border
// and the halved borders of the outermost cells on that
// edge.
//
// Takes box (*LayoutBox) which is the table box providing
// the table's own border widths.
// Takes placements ([]tableCellPlacement) which are the
// grid placements (with already-halved cell borders).
// Takes rowCount (int) which is the total number of rows.
// Takes columnCount (int) which is the total number of
// columns.
//
// Returns BoxEdges with the collapsed outer border widths.
func collapsedTableOuterBorder(
	box *LayoutBox,
	placements []tableCellPlacement,
	rowCount, columnCount int,
) BoxEdges {
	border := BoxEdges{
		Top:    box.Style.BorderTopWidth,
		Bottom: box.Style.BorderBottomWidth,
		Left:   box.Style.BorderLeftWidth,
		Right:  box.Style.BorderRightWidth,
	}

	for _, placement := range placements {
		cell := placement.cell
		if placement.row == 0 && cell.Style.BorderTopWidth > border.Top {
			border.Top = cell.Style.BorderTopWidth
		}
		if placement.row+placement.rowspan >= rowCount && cell.Style.BorderBottomWidth > border.Bottom {
			border.Bottom = cell.Style.BorderBottomWidth
		}
		if placement.column == 0 && cell.Style.BorderLeftWidth > border.Left {
			border.Left = cell.Style.BorderLeftWidth
		}
		if placement.column+placement.colspan >= columnCount && cell.Style.BorderRightWidth > border.Right {
			border.Right = cell.Style.BorderRightWidth
		}
	}

	return border
}
