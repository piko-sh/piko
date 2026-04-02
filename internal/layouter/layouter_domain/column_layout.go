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

// layoutMultiColumnContainer performs multi-column layout
// for a container with column-count or column-width set.
//
// Takes box (*LayoutBox) which is the multi-column container.
// Takes input (layoutInput) which carries the layout constraints.
//
// Returns formattingContextResult which holds the positioned column fragments.
func layoutMultiColumnContainer(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	gap := box.Style.ColumnGap
	if gap <= 0 {
		gap = box.Style.FontSize
	}

	columnCount, columnWidth := resolveColumnDimensions(
		box.Style.ColumnCount,
		box.Style.ColumnWidth,
		gap,
		input.AvailableWidth,
	)

	singleColumnInput := layoutInput{
		AvailableWidth:     columnWidth,
		AvailableBlockSize: 0,
		FontMetrics:        input.FontMetrics,
		Cache:              input.Cache,
		SizingMode:         input.SizingMode,
		Edges:              input.Edges,
	}

	var singleColumnResult formattingContextResult
	if hasOnlyInlineChildren(box) {
		singleColumnResult = layoutInlineContent(ctx, box, singleColumnInput)
	} else {
		singleColumnResult = layoutBlockChildren(ctx, box, singleColumnInput)
	}

	totalContentHeight := singleColumnResult.ContentHeight
	columnHeight := resolveColumnHeight(
		totalContentHeight,
		columnCount,
		box.Style.ColumnFill,
		input.AvailableBlockSize,
		&box.Style,
	)

	columns := fragmentIntoColumns(singleColumnResult.Children, columnHeight)
	for len(columns) > columnCount && columnHeight > 0 {
		columnHeight++
		columns = fragmentIntoColumns(singleColumnResult.Children, columnHeight)
	}
	if len(columns) == 0 {
		return formattingContextResult{
			ContentHeight: 0,
			Margin:        singleColumnResult.Margin,
		}
	}

	resultFragments := positionColumns(columns, columnWidth, gap)

	return formattingContextResult{
		Children:      resultFragments,
		ContentHeight: columnHeight,
		Margin:        singleColumnResult.Margin,
	}
}

// resolveColumnDimensions computes the actual column count
// and column width from the CSS properties and available
// space, following the CSS Multi-column spec section 3.
//
// Takes columnCount (int) which is the declared column-count value.
// Takes columnWidth (Dimension) which is the declared column-width value.
// Takes gap (float64) which is the column gap in points.
// Takes availableWidth (float64) which is the available inline space.
//
// Returns the resolved column count and column width.
func resolveColumnDimensions(
	columnCount int,
	columnWidth Dimension,
	gap float64,
	availableWidth float64,
) (int, float64) {
	hasCount := columnCount > 0
	hasWidth := !columnWidth.IsAuto() && columnWidth.Value > 0

	switch {
	case hasCount && hasWidth:
		maxByWidth := int(math.Floor((availableWidth + gap) / (columnWidth.Value + gap)))
		maxByWidth = max(maxByWidth, 1)
		resolvedCount := min(columnCount, maxByWidth)
		resolvedWidth := (availableWidth - float64(resolvedCount-1)*gap) / float64(resolvedCount)
		return resolvedCount, resolvedWidth

	case hasCount:
		resolvedWidth := (availableWidth - float64(columnCount-1)*gap) / float64(columnCount)
		if resolvedWidth < 0 {
			resolvedWidth = 0
		}
		return columnCount, resolvedWidth

	case hasWidth:
		resolvedCount := max(int(math.Floor((availableWidth+gap)/(columnWidth.Value+gap))), 1)
		resolvedWidth := (availableWidth - float64(resolvedCount-1)*gap) / float64(resolvedCount)
		return resolvedCount, resolvedWidth

	default:
		return 1, availableWidth
	}
}

// resolveColumnHeight determines the column height.
//
// For balanced columns, the total content height is divided evenly.
// For auto-fill columns, an explicit container height is used.
//
// Takes totalContentHeight (float64) which is the total content height.
// Takes columnCount (int) which is the number of columns.
// Takes fill (ColumnFillType) which is the column-fill mode.
// Takes availableBlockSize (float64) which is the available block space.
// Takes style (*ComputedStyle) which is the container's style.
//
// Returns float64 which is the resolved column height in points.
func resolveColumnHeight(
	totalContentHeight float64,
	columnCount int,
	fill ColumnFillType,
	availableBlockSize float64,
	style *ComputedStyle,
) float64 {
	if !style.Height.IsAuto() {
		return adjustForBoxSizing(
			style.Height.Resolve(availableBlockSize, totalContentHeight),
			style, false,
		)
	}

	if fill == ColumnFillAuto && availableBlockSize > 0 {
		return availableBlockSize
	}

	balanced := totalContentHeight / float64(columnCount)
	if balanced < 1 {
		balanced = 0
	}
	return balanced
}

// fragmentIntoColumns splits a flat list of child fragments
// into column groups based on the target column height.
//
// Takes children ([]*Fragment) which is the flat list of child fragments.
// Takes columnHeight (float64) which is the target column height in points.
//
// Returns [][]*Fragment which is the list of column groups.
func fragmentIntoColumns(children []*Fragment, columnHeight float64) [][]*Fragment {
	if len(children) == 0 || columnHeight <= 0 {
		return nil
	}

	var columns [][]*Fragment
	var currentColumn []*Fragment
	currentColumnHeight := 0.0

	for _, child := range children {
		childHeight := child.MarginBoxHeight()
		heightWithoutTrailingMargin := childHeight - child.Margin.Bottom

		if currentColumnHeight+heightWithoutTrailingMargin > columnHeight && len(currentColumn) > 0 {
			columns = append(columns, currentColumn)
			currentColumn = nil
			currentColumnHeight = 0
		}

		adjustedChild := &Fragment{
			Box:           child.Box,
			Children:      child.Children,
			OffsetX:       0,
			OffsetY:       currentColumnHeight,
			ContentWidth:  child.ContentWidth,
			ContentHeight: child.ContentHeight,
			Padding:       child.Padding,
			Border:        child.Border,
			Margin:        child.Margin,
		}
		currentColumn = append(currentColumn, adjustedChild)
		currentColumnHeight += childHeight
	}

	if len(currentColumn) > 0 {
		columns = append(columns, currentColumn)
	}

	return columns
}

// positionColumns places column groups side by side with
// the given column width and gap, returning wrapper
// fragments for each column.
//
// Takes columns ([][]*Fragment) which is the column groups.
// Takes columnWidth (float64) which is the width of each column.
// Takes gap (float64) which is the gap between columns.
//
// Returns []*Fragment which is the positioned fragments.
func positionColumns(columns [][]*Fragment, columnWidth, gap float64) []*Fragment {
	result := make([]*Fragment, 0, len(columns))

	for index, column := range columns {
		offsetX := float64(index) * (columnWidth + gap)

		for _, child := range column {
			positioned := &Fragment{
				Box:           child.Box,
				Children:      child.Children,
				OffsetX:       offsetX + child.OffsetX,
				OffsetY:       child.OffsetY,
				ContentWidth:  child.ContentWidth,
				ContentHeight: child.ContentHeight,
				Padding:       child.Padding,
				Border:        child.Border,
				Margin:        child.Margin,
			}
			result = append(result, positioned)
		}
	}

	return result
}
