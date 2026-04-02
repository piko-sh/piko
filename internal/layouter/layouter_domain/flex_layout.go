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
	"slices"
)

// flexItem holds the intermediate layout state for a
// single flex item during the flex layout algorithm.
type flexItem struct {
	// box is the layout box that this flex item wraps.
	box *LayoutBox

	// fragment is the layout result from layoutBox,
	// carrying child fragments with relative offsets.
	fragment *Fragment

	// flexBaseSize is the initial main size before flex
	// grow or shrink adjustments.
	flexBaseSize float64

	// mainSize is the resolved main axis size after
	// flexible length distribution.
	mainSize float64

	// crossSize is the resolved cross axis size of the
	// item.
	crossSize float64

	// flexGrow is the flex grow factor from the item
	// style.
	flexGrow float64

	// flexShrink is the flex shrink factor from the item
	// style.
	flexShrink float64

	// targetMainSize is the target main size computed
	// during flexible length resolution.
	targetMainSize float64

	// autoMarginStart is the resolved auto margin on the
	// start side of the main axis (left for row, top for
	// column). Zero when the margin is not auto.
	autoMarginStart float64

	// autoMarginEnd is the resolved auto margin on the
	// end side of the main axis (right for row, bottom
	// for column). Zero when the margin is not auto.
	autoMarginEnd float64
}

// flexLine holds a group of flex items that share a
// single line in the flex container.
type flexLine struct {
	// items is the slice of flex items on this line.
	items []*flexItem

	// mainSize is the total main axis size of this line.
	mainSize float64

	// crossSize is the resolved cross axis size of this
	// line.
	crossSize float64
}

// flexContainerParams holds the resolved direction flags,
// axis sizes, and gap values for a flex container.
type flexContainerParams struct {
	// containerMainSize holds the main axis size of the container.
	containerMainSize float64

	// containerCrossSize holds the cross axis size of the container.
	containerCrossSize float64

	// mainGap holds the gap between items along the main axis.
	mainGap float64

	// crossGap holds the gap between lines along the cross axis.
	crossGap float64

	// isRowDirection indicates whether the main axis is horizontal.
	isRowDirection bool

	// isReversed indicates whether the main axis direction is reversed.
	isReversed bool

	// isWrap indicates whether flex wrapping is enabled.
	isWrap bool

	// isWrapReverse indicates whether line ordering is reversed.
	isWrapReverse bool

	// mainSizeIndefinite indicates that no explicit main axis size was set.
	mainSizeIndefinite bool
}

// resolveFlexContainerParams resolves the direction flags,
// container axis sizes, and gap values from the box style
// and available layout space.
//
// Takes box (*LayoutBox) which is the flex container.
// Takes input (layoutInput) which carries available
// width and block size from the parent context.
//
// Returns the populated flexContainerParams.
func resolveFlexContainerParams(box *LayoutBox, input layoutInput) flexContainerParams {
	containerWidth := input.AvailableWidth

	isRow := box.Style.FlexDirection == FlexDirectionRow ||
		box.Style.FlexDirection == FlexDirectionRowReverse
	isReversed := box.Style.FlexDirection == FlexDirectionRowReverse ||
		box.Style.FlexDirection == FlexDirectionColumnReverse
	if isRow && box.Style.Direction == DirectionRTL {
		isReversed = !isReversed
	}

	params := flexContainerParams{
		containerMainSize: containerWidth,
		isRowDirection:    isRow,
		isReversed:        isReversed,
		isWrap:            box.Style.FlexWrap == FlexWrapWrap || box.Style.FlexWrap == FlexWrapWrapReverse,
		isWrapReverse:     box.Style.FlexWrap == FlexWrapWrapReverse,
		mainGap:           box.Style.ColumnGap,
		crossGap:          box.Style.RowGap,
	}

	if !isRow {
		resolveColumnFlexParams(&params, box, input, containerWidth)
	}

	return params
}

// resolveColumnFlexParams populates the container main size,
// cross size, gaps, and indefinite flag for a column-direction
// flex container.
//
// Takes params (*flexContainerParams) which is the params
// struct to populate.
// Takes box (*LayoutBox) which is the flex container.
// Takes input (layoutInput) which carries the available
// block size.
// Takes containerWidth (float64) which is the inline-axis
// available width.
func resolveColumnFlexParams(params *flexContainerParams, box *LayoutBox, input layoutInput, containerWidth float64) {
	if !box.Style.Height.IsAuto() && !box.Style.Height.IsIntrinsic() {
		params.containerMainSize = adjustForBoxSizing(box.Style.Height.Resolve(0, 0), &box.Style, false)
	} else if input.AvailableBlockSize > 0 {
		params.containerMainSize = input.AvailableBlockSize
	} else {
		params.mainSizeIndefinite = true
		params.containerMainSize = 0
	}
	params.containerCrossSize = containerWidth
	params.mainGap = box.Style.RowGap
	params.crossGap = box.Style.ColumnGap
}

// adjustIndefiniteMainSize replaces an indefinite container
// main size with the sum of item base sizes plus gaps so
// that flex-grow and flex-shrink distribute correctly.
//
// Takes containerMainSize (float64) which is the current
// main axis size (zero when indefinite).
// Takes items ([]*flexItem) which is the collected flex
// items.
// Takes mainGap (float64) which is the gap between items
// along the main axis.
// Takes indefinite (bool) which is true when the main
// axis size was not explicitly set.
//
// Returns the adjusted container main size.
func adjustIndefiniteMainSize(containerMainSize float64, items []*flexItem, mainGap float64, indefinite bool) float64 {
	if !indefinite || len(items) == 0 {
		return containerMainSize
	}
	total := 0.0
	for _, item := range items {
		total += item.flexBaseSize
	}
	return total + float64(len(items)-1)*mainGap
}

// preStretchColumnItems sets the cross size of column-direction
// flex items with auto width to the line cross size, so that
// children receive the correct available width before layout.
//
// Takes lines ([]*flexLine) which is the slice of flex
// lines to adjust.
func preStretchColumnItems(lines []*flexLine) {
	for _, line := range lines {
		for _, item := range line.items {
			if item.box.Style.Width.IsAuto() {
				item.crossSize = line.crossSize
			}
		}
	}
}

// layoutFlexContainer performs the full CSS flexbox layout
// algorithm on the given container box.
//
// Takes ctx (context.Context) which carries cancellation
// and deadline signals from the caller.
// Takes box (*LayoutBox) which is the flex container
// to lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics from the parent context.
//
// Returns formattingContextResult which holds the child
// fragments, content height, and margin edges.
func layoutFlexContainer(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	params := resolveFlexContainerParams(box, input)

	items := collectFlexItems(ctx, box, params.isRowDirection, params.containerMainSize, input)
	params.containerMainSize = adjustIndefiniteMainSize(
		params.containerMainSize, items, params.mainGap, params.mainSizeIndefinite,
	)

	if len(items) == 0 {
		return formattingContextResult{
			Margin: BoxEdges{
				Top:    input.Edges.MarginTop,
				Bottom: input.Edges.MarginBottom,
			},
		}
	}

	sortItemsByOrder(items)

	lines := collectFlexLines(items, params.containerMainSize, params.mainGap, params.isWrap)

	for _, line := range lines {
		resolveFlexibleLengths(line, params.containerMainSize, params.mainGap, params.isRowDirection)
	}

	acResult := resolveFlexLineCrossSizes(
		lines, box, params.isRowDirection, params.containerCrossSize, params.crossGap, params.isWrap,
	)

	if !params.isRowDirection {
		preStretchColumnItems(lines)
	}

	positionFlexItems(ctx, lines, params.isWrapReverse, acResult, new(buildFlexPositionContext(
		box, params.isRowDirection, params.isReversed, params.isWrap,
		params.mainGap, params.crossGap, input,
	)))

	containerHeight := computeFlexContainerHeight(box, lines, params.isRowDirection, params.crossGap)

	return formattingContextResult{
		Children:      collectFlexFragments(lines),
		ContentHeight: containerHeight,
		Margin: BoxEdges{
			Top:    input.Edges.MarginTop,
			Bottom: input.Edges.MarginBottom,
		},
	}
}

// collectFlexFragments gathers all item fragments from the
// resolved flex lines into a single slice.
//
// Takes lines ([]*flexLine) which holds the resolved flex lines.
//
// Returns []*Fragment which holds the collected item fragments.
func collectFlexFragments(lines []*flexLine) []*Fragment {
	var fragments []*Fragment
	for _, line := range lines {
		for _, item := range line.items {
			fragments = append(fragments, item.fragment)
		}
	}
	return fragments
}

// collectFlexItems builds flex items from the container
// children, skipping absolutely positioned elements.
//
// Takes ctx (context.Context) which carries cancellation
// and deadline signals from the caller.
// Takes container (*LayoutBox) which is the flex
// container whose children are collected.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes input (layoutInput) which carries font metrics
// and cache from the parent context.
//
// Returns the slice of flex items ready for layout.
func collectFlexItems(
	ctx context.Context,
	container *LayoutBox,
	isRowDirection bool,
	containerMainSize float64,
	input layoutInput,
) []*flexItem {
	items := make([]*flexItem, 0, len(container.Children))

	for _, child := range container.Children {
		if child.Style.Position == PositionAbsolute || child.Style.Position == PositionFixed {
			continue
		}

		baseSize := resolveFlexBaseSize(ctx, child, isRowDirection, containerMainSize, input)

		items = append(items, &flexItem{
			box:          child,
			flexBaseSize: baseSize,
			mainSize:     baseSize,
			flexGrow:     child.Style.FlexGrow,
			flexShrink:   child.Style.FlexShrink,
		})
	}

	return items
}

// resolveFlexBaseSize computes the flex base size for a
// child box using flex-basis, explicit size, or intrinsic
// content measurement.
//
// Takes ctx (context.Context) which carries cancellation
// and deadline signals from the caller.
// Takes child (*LayoutBox) which is the flex item box.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes input (layoutInput) which carries font metrics
// and cache from the parent context.
//
// Returns the resolved base size in points.
func resolveFlexBaseSize(
	ctx context.Context,
	child *LayoutBox,
	isRowDirection bool,
	containerMainSize float64,
	input layoutInput,
) float64 {
	if !child.Style.FlexBasis.IsAuto() {
		return resolveFlexDimension(child, child.Style.FlexBasis, containerMainSize, isRowDirection)
	}

	if size, ok := resolveExplicitMainSize(child, isRowDirection, containerMainSize); ok {
		return size
	}

	return resolveFlexIntrinsicSize(ctx, child, isRowDirection, containerMainSize, input)
}

// resolveFlexDimension resolves a dimension value for a flex item,
// adding edge widths when box-sizing is not border-box.
//
// Takes child (*LayoutBox) which is the flex item box.
// Takes dim (Dimension) which is the dimension to resolve.
// Takes containerMainSize (float64) which is the main axis
// size for percentage resolution.
// Takes isRowDirection (bool) which selects horizontal or
// vertical edges.
//
// Returns float64 which is the resolved border-box size in points.
func resolveFlexDimension(
	child *LayoutBox, dim Dimension, containerMainSize float64, isRowDirection bool,
) float64 {
	resolved := dim.Resolve(containerMainSize, 0)
	if child.Style.BoxSizing != BoxSizingBorderBox {
		edges := resolveEdgesFromStyle(&child.Style, containerMainSize)
		if isRowDirection {
			resolved += edges.Padding.Horizontal() + edges.Border.Horizontal()
		} else {
			resolved += edges.Padding.Vertical() + edges.Border.Vertical()
		}
	}
	return resolved
}

// resolveExplicitMainSize checks for an explicit width (row) or
// height (column) on the flex item and returns the resolved border-
// box size.
//
// Takes child (*LayoutBox) which is the flex item box.
// Takes isRowDirection (bool) which selects width or height.
// Takes containerMainSize (float64) which is the main axis
// size for percentage resolution.
//
// Returns float64 which is the resolved size.
// Returns bool which is false when the main-axis dimension
// is auto.
func resolveExplicitMainSize(child *LayoutBox, isRowDirection bool, containerMainSize float64) (float64, bool) {
	if isRowDirection && !child.Style.Width.IsAuto() && !child.Style.Width.IsFitContent() {
		return resolveFlexDimension(child, child.Style.Width, containerMainSize, true), true
	}
	if !isRowDirection && !child.Style.Height.IsAuto() && !child.Style.Height.IsFitContent() {
		return resolveFlexDimension(child, child.Style.Height, 0, false), true
	}
	return 0, false
}

// resolveFlexIntrinsicSize measures the intrinsic content of a
// flex item when neither flex-basis nor an explicit main-axis size
// is set. Uses max-content sizing so flex items size to their
// content rather than filling 100% of the container.
//
// Takes ctx (context.Context) which carries cancellation signals.
// Takes child (*LayoutBox) which is the flex item box.
// Takes isRowDirection (bool) which selects horizontal or
// vertical measurement.
// Takes containerMainSize (float64) which is the main axis
// size for column layout.
//
// Returns float64 which is the intrinsic content size in points.
func resolveFlexIntrinsicSize(
	ctx context.Context, child *LayoutBox, isRowDirection bool, containerMainSize float64, input layoutInput,
) float64 {
	if isRowDirection {
		return measureMaxContentWidth(child, input.FontMetrics)
	}

	fragment := layoutBox(ctx, child, layoutInput{AvailableWidth: containerMainSize, FontMetrics: input.FontMetrics, Cache: input.Cache})
	return fragment.ContentHeight + fragment.Padding.Vertical() + fragment.Border.Vertical()
}

// sortItemsByOrder sorts flex items in place by their CSS
// order property using a stable sort.
//
// Takes items ([]*flexItem) which is the slice of flex
// items to sort.
func sortItemsByOrder(items []*flexItem) {
	slices.SortStableFunc(items, func(a, b *flexItem) int {
		return a.box.Style.Order - b.box.Style.Order
	})
}

// collectFlexLines groups flex items into lines based on
// the container main size and wrap mode.
//
// Takes items ([]*flexItem) which is the slice of flex
// items to group into lines.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes mainGap (float64) which is the gap between
// items along the main axis.
// Takes isWrap (bool) which is true when flex wrapping
// is enabled.
//
// Returns the slice of flex lines.
func collectFlexLines(
	items []*flexItem,
	containerMainSize float64,
	mainGap float64,
	isWrap bool,
) []*flexLine {
	if !isWrap {
		totalMainSize := 0.0
		for index, item := range items {
			totalMainSize += item.flexBaseSize
			if index > 0 {
				totalMainSize += mainGap
			}
		}
		return []*flexLine{{items: items, mainSize: totalMainSize}}
	}

	var lines []*flexLine
	lineStart := 0
	lineMainSize := 0.0

	for index, item := range items {
		candidateSize := lineMainSize + item.flexBaseSize
		if index > lineStart {
			candidateSize += mainGap
		}

		if index > lineStart && candidateSize > containerMainSize {
			lines = append(lines, &flexLine{
				items:    items[lineStart:index],
				mainSize: lineMainSize,
			})
			lineStart = index
			lineMainSize = item.flexBaseSize
		} else {
			lineMainSize = candidateSize
		}
	}

	if lineStart < len(items) {
		lines = append(lines, &flexLine{
			items:    items[lineStart:],
			mainSize: lineMainSize,
		})
	}

	return lines
}

// resolveFlexibleLengths distributes free space among flex
// items on a line using flex-grow or flex-shrink factors.
//
// The algorithm follows CSS Flexbox spec section 9.7:
// determine whether the line is growing or shrinking,
// freeze inflexible items, then iteratively redistribute
// free space until no min/max violations remain.
//
// Takes line (*flexLine) which is the flex line whose
// items receive distributed space.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes mainGap (float64) which is the gap between
// items along the main axis.
// Takes isRowDirection (bool) which is true for row flex.
func resolveFlexibleLengths(line *flexLine, containerMainSize, mainGap float64, isRowDirection bool) {
	totalGaps := mainGap * float64(len(line.items)-1)

	growing := isFlexLineGrowing(line, totalGaps, isRowDirection, containerMainSize)
	frozen := freezeInflexibleItems(line, growing, isRowDirection, containerMainSize)

	flexFreezeLoop(line, frozen, growing, totalGaps, containerMainSize, isRowDirection)
}

// isFlexLineGrowing returns true when the sum of hypothetical
// main sizes (base sizes clamped to min/max) plus gaps does
// not exceed the container main size, meaning items should
// grow rather than shrink.
//
// Takes line (*flexLine) which is the flex line to check.
// Takes totalGaps (float64) which is the total gap space
// between items.
// Takes isRowDirection (bool) which selects horizontal or
// vertical clamping.
// Takes containerMainSize (float64) which is the main axis
// size of the container.
//
// Returns bool which is true when items should grow.
func isFlexLineGrowing(line *flexLine, totalGaps float64, isRowDirection bool, containerMainSize float64) bool {
	used := totalGaps
	for _, item := range line.items {
		used += clampFlexMainSize(item.flexBaseSize, item.box, isRowDirection, containerMainSize)
	}
	return used <= containerMainSize
}

// freezeInflexibleItems freezes items that have a zero
// flex factor for the current grow/shrink mode, setting
// their main size to the clamped base size.
//
// Takes line (*flexLine) which is the flex line to process.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes isRowDirection (bool) which selects horizontal or
// vertical clamping.
// Takes containerMainSize (float64) which is the main axis
// size for percentage resolution.
//
// Returns []bool which holds the frozen flag for each item.
func freezeInflexibleItems(line *flexLine, growing, isRowDirection bool, containerMainSize float64) []bool {
	frozen := make([]bool, len(line.items))
	for i, item := range line.items {
		if (growing && item.flexGrow == 0) || (!growing && item.flexShrink == 0) {
			item.mainSize = clampFlexMainSize(
				item.flexBaseSize, item.box, isRowDirection, containerMainSize,
			)
			item.targetMainSize = item.mainSize
			frozen[i] = true
		}
	}
	return frozen
}

// flexFreezeLoop performs the iterative freeze-and-redistribute
// loop per CSS Flexbox spec section 9.7, distributing free
// space to unfrozen items and freezing any that violate their
// min/max constraints until convergence.
//
// Takes line (*flexLine) which is the flex line to process.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes totalGaps (float64) which is the total gap space.
// Takes containerMainSize (float64) which is the main axis size.
// Takes isRowDirection (bool) which selects horizontal or
// vertical clamping.
func flexFreezeLoop(
	line *flexLine, frozen []bool, growing bool,
	totalGaps, containerMainSize float64, isRowDirection bool,
) {
	for {
		freeSpace, totalFactor := computeFlexFreeSpace(line, frozen, growing, totalGaps, containerMainSize)
		distributeFlexFreeSpace(line, frozen, growing, freeSpace, totalFactor)

		if !freezeFlexViolators(line, frozen, isRowDirection, containerMainSize) {
			applyUnfrozenTargets(line, frozen)
			break
		}
	}
}

// computeFlexFreeSpace calculates the remaining free space
// and the total flex factor among unfrozen items.
//
// Takes line (*flexLine) which is the flex line to measure.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes totalGaps (float64) which is the total gap space.
// Takes containerMainSize (float64) which is the main axis size.
//
// Returns freeSpace (float64) which is the remaining free
// space in points.
// Returns totalFactor (float64) which is the flex-grow sum
// when growing or weighted flex-shrink sum when shrinking.
func computeFlexFreeSpace(
	line *flexLine, frozen []bool, growing bool,
	totalGaps, containerMainSize float64,
) (freeSpace float64, totalFactor float64) {
	usedByFrozen := totalGaps
	for i, item := range line.items {
		if frozen[i] {
			usedByFrozen += item.mainSize
		} else {
			usedByFrozen += item.flexBaseSize
			if growing {
				totalFactor += item.flexGrow
			} else {
				totalFactor += item.flexShrink * item.flexBaseSize
			}
		}
	}
	return containerMainSize - usedByFrozen, totalFactor
}

// distributeFlexFreeSpace assigns target main sizes to
// unfrozen items by distributing the available free space
// according to their flex factors.
//
// Takes line (*flexLine) which is the flex line to update.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes growing (bool) which indicates the grow/shrink mode.
// Takes freeSpace (float64) which is the available free space.
// Takes totalFactor (float64) which is the total flex factor.
func distributeFlexFreeSpace(line *flexLine, frozen []bool, growing bool, freeSpace, totalFactor float64) {
	for i, item := range line.items {
		if frozen[i] {
			continue
		}
		if growing && totalFactor > 0 {
			item.targetMainSize = item.flexBaseSize +
				freeSpace*(item.flexGrow/totalFactor)
		} else if !growing && totalFactor > 0 {
			ratio := (item.flexShrink * item.flexBaseSize) / totalFactor
			item.targetMainSize = math.Max(0, item.flexBaseSize+freeSpace*ratio)
		} else {
			item.targetMainSize = item.flexBaseSize
		}
	}
}

// freezeFlexViolators clamps each unfrozen item's target
// main size to its min/max constraints and freezes any
// item that was clamped.
//
// Takes line (*flexLine) which is the flex line to check.
// Takes frozen ([]bool) which tracks which items are frozen.
// Takes isRowDirection (bool) which selects horizontal or
// vertical clamping.
// Takes containerMainSize (float64) which is the main axis
// size for percentage resolution.
//
// Returns bool which is true if at least one item was frozen.
func freezeFlexViolators(line *flexLine, frozen []bool, isRowDirection bool, containerMainSize float64) bool {
	anyFrozen := false
	for i, item := range line.items {
		if frozen[i] {
			continue
		}
		clamped := clampFlexMainSize(
			item.targetMainSize, item.box, isRowDirection, containerMainSize,
		)
		if clamped != item.targetMainSize {
			item.mainSize = clamped
			item.targetMainSize = clamped
			frozen[i] = true
			anyFrozen = true
		}
	}
	return anyFrozen
}

// applyUnfrozenTargets copies the target main size to the
// resolved main size for all items that were never frozen.
//
// Takes line (*flexLine) which is the flex line to update.
// Takes frozen ([]bool) which tracks which items are frozen.
func applyUnfrozenTargets(line *flexLine, frozen []bool) {
	for i, item := range line.items {
		if !frozen[i] {
			item.mainSize = item.targetMainSize
		}
	}
}

// clampFlexMainSize applies min/max main-axis constraints
// to a flex item's border-box size.
//
// Takes size (float64) which is the border-box main size.
// Takes box (*LayoutBox) which is the flex item.
// Takes isRowDirection (bool) which is true for row flex.
// Takes containerMainSize (float64) for percentage
// resolution.
//
// Returns the clamped border-box size.
func clampFlexMainSize(size float64, box *LayoutBox, isRowDirection bool, containerMainSize float64) float64 {
	if isRowDirection {
		if !box.Style.MinWidth.IsAuto() && !box.Style.MinWidth.IsFitContent() {
			minWidth := resolveFlexDimension(box, box.Style.MinWidth, containerMainSize, true)
			size = math.Max(size, minWidth)
		}
		if !box.Style.MaxWidth.IsAuto() && !box.Style.MaxWidth.IsFitContent() {
			maxWidth := resolveFlexDimension(box, box.Style.MaxWidth, containerMainSize, true)
			size = math.Min(size, maxWidth)
		}
	} else {
		if !box.Style.MinHeight.IsAuto() && !box.Style.MinHeight.IsFitContent() {
			minHeight := resolveFlexDimension(box, box.Style.MinHeight, 0, false)
			size = math.Max(size, minHeight)
		}
		if !box.Style.MaxHeight.IsAuto() && !box.Style.MaxHeight.IsFitContent() {
			maxHeight := resolveFlexDimension(box, box.Style.MaxHeight, 0, false)
			size = math.Min(size, maxHeight)
		}
	}
	return size
}

// resolveFlexLineCrossSizes computes the cross size for
// each flex line and adjusts for explicit container cross
// size when applicable.
//
// Takes lines ([]*flexLine) which is the slice of flex
// lines to resolve.
// Takes container (*LayoutBox) which is the flex
// container box.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
// Takes containerCrossSize (float64) which is the
// cross axis size of the container.
// Takes crossGap (float64) which is the gap between
// lines along the cross axis.
// Takes isWrap (bool) which is true when flex wrapping
// is enabled.
//
// Returns alignContentResult which holds the initial cross
// offset and per-line spacing from align-content distribution.
func resolveFlexLineCrossSizes(
	lines []*flexLine,
	container *LayoutBox,
	isRowDirection bool,
	containerCrossSize float64,
	crossGap float64,
	isWrap bool,
) alignContentResult {
	for _, line := range lines {
		maxCrossSize := 0.0
		for _, item := range line.items {
			itemCrossSize := resolveItemCrossSize(item, isRowDirection)
			item.crossSize = itemCrossSize
			maxCrossSize = math.Max(maxCrossSize, itemCrossSize)
		}
		line.crossSize = maxCrossSize
	}

	if !isRowDirection && containerCrossSize > 0 && len(lines) == 1 && !isWrap {
		lines[0].crossSize = containerCrossSize
	}

	if isRowDirection && !container.Style.Height.IsAuto() {
		explicitCrossSize := adjustForBoxSizing(container.Style.Height.Resolve(0, 0), &container.Style, false)
		totalCrossGaps := crossGap * float64(len(lines)-1)
		if len(lines) == 1 {
			lines[0].crossSize = math.Max(lines[0].crossSize, explicitCrossSize)
		} else if len(lines) > 1 {
			return distributeAlignContent(lines, explicitCrossSize, totalCrossGaps, container.Style.AlignContent)
		}
	}
	return alignContentResult{}
}

// resolveItemCrossSize computes the cross axis size for a
// single flex item, including padding and border.
//
// Takes item (*flexItem) which is the flex item to
// measure.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
//
// Returns the total cross size in points.
func resolveItemCrossSize(item *flexItem, isRowDirection bool) float64 {
	child := item.box
	edges := resolveEdgesFromStyle(&child.Style, 0)

	if isRowDirection {
		if !child.Style.Height.IsAuto() && !child.Style.Height.IsFitContent() {
			declared := child.Style.Height.Resolve(0, 0)
			if child.Style.BoxSizing == BoxSizingBorderBox {
				return declared
			}
			return declared + edges.Padding.Vertical() + edges.Border.Vertical()
		}
		return edges.Padding.Vertical() + edges.Border.Vertical()
	}

	if !child.Style.Width.IsAuto() && !child.Style.Width.IsFitContent() {
		declared := child.Style.Width.Resolve(0, 0)
		if child.Style.BoxSizing == BoxSizingBorderBox {
			return declared
		}
		return declared + edges.Padding.Horizontal() + edges.Border.Horizontal()
	}
	return edges.Padding.Horizontal() + edges.Border.Horizontal()
}

// alignContentResult holds the initial cross offset and
// per-line spacing computed by distributeAlignContent.
type alignContentResult struct {
	// initialOffset holds the cross axis offset before the first line.
	initialOffset float64

	// lineSpacing holds the extra spacing between lines.
	lineSpacing float64
}

// distributeAlignContent distributes remaining cross axis
// space among flex lines according to the align-content
// property.
//
// Takes lines ([]*flexLine) which is the slice of flex
// lines to adjust.
// Takes containerCrossSize (float64) which is the
// cross axis size of the container.
// Takes totalCrossGaps (float64) which is the total
// gap space between lines.
// Takes alignContent (AlignContentType) which is the
// align-content value from the container style.
//
// Returns alignContentResult which holds the computed
// initial offset and per-line spacing.
func distributeAlignContent(
	lines []*flexLine,
	containerCrossSize float64,
	totalCrossGaps float64,
	alignContent AlignContentType,
) alignContentResult {
	totalLineCross := totalCrossGaps
	for _, line := range lines {
		totalLineCross += line.crossSize
	}

	remainingSpace := containerCrossSize - totalLineCross
	if remainingSpace <= 0 {
		return alignContentResult{}
	}

	switch alignContent {
	case AlignContentStretch:
		extra := remainingSpace / float64(len(lines))
		for _, line := range lines {
			line.crossSize += extra
		}
	case AlignContentFlexEnd:
		return alignContentResult{initialOffset: remainingSpace}
	case AlignContentCentre:
		return alignContentResult{initialOffset: remainingSpace / 2}
	case AlignContentSpaceBetween:
		if len(lines) > 1 {
			return alignContentResult{lineSpacing: remainingSpace / float64(len(lines)-1)}
		}
	case AlignContentSpaceAround:
		if len(lines) > 0 {
			perLine := remainingSpace / float64(len(lines))
			return alignContentResult{initialOffset: perLine / 2, lineSpacing: perLine}
		}
	}
	return alignContentResult{}
}

// flexPositionContext holds the parameters needed to
// position flex items after their sizes are resolved.
type flexPositionContext struct {
	// input carries the layout constraints and cache
	// from the parent context.
	input layoutInput

	// contentOffsetX is the horizontal offset from
	// the container's ContentX to the content start
	// (padding + border).
	contentOffsetX float64

	// contentOffsetY is the vertical offset from
	// the container's ContentY to the content start
	// (padding + border).
	contentOffsetY float64

	// containerMainSize is the main axis size of the
	// container content area.
	containerMainSize float64

	// containerCrossSize is the explicit cross axis size
	// of the container, or 0 when the cross size is auto.
	containerCrossSize float64

	// mainGap is the gap between items along the main
	// axis.
	mainGap float64

	// crossGap is the gap between lines along the cross
	// axis.
	crossGap float64

	// alignItems is the container align-items value used
	// as the default cross axis alignment.
	alignItems AlignItemsType

	// justifyContent is the container justify-content
	// value for main axis distribution.
	justifyContent JustifyContentType

	// isRowDirection is true when the main axis is
	// horizontal.
	isRowDirection bool

	// isReversed is true when the main axis direction is
	// reversed.
	isReversed bool

	// isWrap is true when the container uses flex-wrap.
	isWrap bool
}

// buildFlexPositionContext constructs a position context
// from the container box and layout parameters.
//
// Takes container (*LayoutBox) which is the flex
// container box.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
// Takes isReversed (bool) which is true when the main
// axis direction is reversed.
// Takes mainGap (float64) which is the gap between
// items along the main axis.
// Takes crossGap (float64) which is the gap between
// lines along the cross axis.
// Takes input (layoutInput) which carries font metrics
// and cache from the parent context.
//
// Returns the populated flexPositionContext.
func buildFlexPositionContext(
	container *LayoutBox,
	isRowDirection bool,
	isReversed bool,
	isWrap bool,
	mainGap float64,
	crossGap float64,
	input layoutInput,
) flexPositionContext {
	containerMainSize := input.AvailableWidth
	if !isRowDirection {
		containerMainSize = adjustForBoxSizing(container.Style.Height.Resolve(0, 0), &container.Style, false)
	}

	containerCrossSize := 0.0
	if isRowDirection && !container.Style.Height.IsAuto() {
		containerCrossSize = adjustForBoxSizing(container.Style.Height.Resolve(0, 0), &container.Style, false)
	} else if !isRowDirection {
		containerCrossSize = input.AvailableWidth
	}

	return flexPositionContext{
		input:              input,
		contentOffsetX:     0,
		contentOffsetY:     0,
		containerMainSize:  containerMainSize,
		containerCrossSize: containerCrossSize,
		mainGap:            mainGap,
		crossGap:           crossGap,
		alignItems:         container.Style.AlignItems,
		justifyContent:     container.Style.JustifyContent,
		isRowDirection:     isRowDirection,
		isReversed:         isReversed,
		isWrap:             isWrap,
	}
}

// positionFlexItems positions all flex items across all
// lines, respecting wrap-reverse ordering.
//
// Takes ctx (context.Context) which carries cancellation
// and deadline signals from the caller.
// Takes lines ([]*flexLine) which is the slice of flex
// lines containing items to position.
// Takes isWrapReverse (bool) which is true when line
// ordering is reversed.
// Takes positionContext (*flexPositionContext) which holds the
// positioning parameters.
func positionFlexItems(ctx context.Context, lines []*flexLine, isWrapReverse bool, acResult alignContentResult, positionContext *flexPositionContext) {
	lineOrder := make([]int, len(lines))
	for index := range lineOrder {
		lineOrder[index] = index
	}
	if isWrapReverse {
		slices.Reverse(lineOrder)
	}

	crossOffset := acResult.initialOffset
	for iteration, lineIndex := range lineOrder {
		if iteration > 0 {
			crossOffset += positionContext.crossGap + acResult.lineSpacing
		}
		positionFlexLine(ctx, lines[lineIndex], crossOffset, positionContext)
		crossOffset += lines[lineIndex].crossSize
	}
}

// positionFlexLine positions all items within a single
// flex line using justify-content and item ordering.
//
// Takes ctx (context.Context) which carries cancellation
// and deadline signals from the caller.
// Takes line (*flexLine) which is the flex line to
// position.
// Takes crossOffset (float64) which is the cross axis
// offset for this line.
// Takes positionContext (*flexPositionContext) which holds the
// positioning parameters.
func positionFlexLine(ctx context.Context, line *flexLine, crossOffset float64, positionContext *flexPositionContext) {
	layoutFlexLineItems(ctx, line, positionContext)
	recomputeLineCrossSize(line)
	clampLineCrossToContainer(line, positionContext)
	stretchRelayoutLineItems(ctx, line, positionContext)

	justify := resolveLineJustifyContent(line, positionContext)
	positionFlexLineItems(line, crossOffset, justify, positionContext)
}

// layoutFlexLineItems performs the first layout pass for
// every item on the line, using auto cross sizes so that
// intrinsic heights (aspect-ratio, table content) are
// preserved.
//
// Takes ctx (context.Context) which carries cancellation signals.
// Takes line (*flexLine) which is the flex line to lay out.
func layoutFlexLineItems(ctx context.Context, line *flexLine, positionContext *flexPositionContext) {
	for _, item := range line.items {
		layoutFlexItem(
			ctx, item.box, item, positionContext.isRowDirection,
			positionContext.containerMainSize, positionContext.containerCrossSize,
			positionContext.input,
		)
	}
}

// clampLineCrossToContainer ensures that a single-line
// (non-wrap) row container's line cross size is at least
// the container cross size. Multi-line containers handle
// cross space distribution via align-content instead.
//
// Takes line (*flexLine) which is the flex line to clamp.
// Takes positionContext (*flexPositionContext) which holds
// the container cross size and wrap flag.
func clampLineCrossToContainer(line *flexLine, positionContext *flexPositionContext) {
	if positionContext.isRowDirection && positionContext.containerCrossSize > 0 && !positionContext.isWrap {
		line.crossSize = math.Max(line.crossSize, positionContext.containerCrossSize)
	}
}

// stretchRelayoutLineItems re-lays out row-direction items
// that have align-self: stretch and an auto cross dimension,
// using the line's definite cross size so children with
// percentage heights and nested flex containers resolve
// correctly.
//
// Takes ctx (context.Context) which carries cancellation signals.
// Takes line (*flexLine) which is the flex line to re-lay out.
func stretchRelayoutLineItems(ctx context.Context, line *flexLine, positionContext *flexPositionContext) {
	if !positionContext.isRowDirection || line.crossSize <= 0 {
		return
	}
	for _, item := range line.items {
		if !needsStretchRelayout(item, line.crossSize, positionContext.alignItems) {
			continue
		}
		relayoutFlexItemWithCrossSize(ctx, item, line.crossSize, positionContext)
	}
}

// needsStretchRelayout returns true when a row-direction
// flex item should be re-laid out with the line's definite
// cross size. The item must have auto height, no auto cross
// margins, align-self resolving to stretch, and a line
// cross size exceeding its current cross size.
//
// Takes item (*flexItem) which is the flex item to check.
// Takes lineCrossSize (float64) which is the line's cross size.
// Takes alignItems (AlignItemsType) which is the container's
// align-items fallback.
//
// Returns bool which is true when the item needs re-layout.
func needsStretchRelayout(item *flexItem, lineCrossSize float64, alignItems AlignItemsType) bool {
	if !item.box.Style.Height.IsAuto() {
		return false
	}
	if hasAutoCrossMargins(item, true) {
		return false
	}
	if resolveAlignSelf(item.box.Style.AlignSelf, alignItems) != AlignItemsStretch {
		return false
	}
	return lineCrossSize > item.crossSize
}

// resolveLineJustifyContent resolves the effective
// justify-content value for a flex line, falling back to
// flex-start when auto margins consume the free space.
//
// Takes line (*flexLine) which is the flex line to check.
// Takes positionContext (*flexPositionContext) which holds
// the justify-content value.
//
// Returns JustifyContentType which is the resolved
// justify-content value.
func resolveLineJustifyContent(line *flexLine, positionContext *flexPositionContext) JustifyContentType {
	if resolveFlexAutoMargins(line, positionContext) {
		return JustifyFlexStart
	}
	return positionContext.justifyContent
}

// positionFlexLineItems places each item on both axes
// using justify-content offsets and spacing, then mirrors
// positions when the main axis direction is reversed.
//
// Takes line (*flexLine) which is the flex line to position.
// Takes crossOffset (float64) which is the cross axis offset
// for this line.
// Takes justify (JustifyContentType) which is the resolved
// justify-content value.
// Takes positionContext (*flexPositionContext) which holds
// the positioning parameters.
func positionFlexLineItems(
	line *flexLine, crossOffset float64,
	justify JustifyContentType, positionContext *flexPositionContext,
) {
	placementOffset := computeJustifyOffset(
		justify, positionContext.containerMainSize, line, positionContext.mainGap,
	)
	itemSpacing := computeJustifySpacing(
		justify, positionContext.containerMainSize, line, positionContext.mainGap,
	)

	for iteration, item := range line.items {
		if iteration > 0 {
			placementOffset += positionContext.mainGap
		}
		autoBefore, autoAfter := itemAutoMainMargins(item, positionContext.isRowDirection)
		placementOffset += autoBefore
		placeFlexItemOnAxes(item, line.crossSize, crossOffset, placementOffset, positionContext)
		placementOffset += item.mainSize + autoAfter + itemSpacing
	}

	if positionContext.isReversed {
		for _, item := range line.items {
			mirrorFlexItemPosition(
				item.fragment, positionContext.containerMainSize,
				positionContext.isRowDirection, positionContext.contentOffsetX,
				positionContext.contentOffsetY,
			)
		}
	}
}

// relayoutFlexItemWithCrossSize re-lays out a row-direction
// flex item with a definite cross (height) size, enabling
// children with percentage heights and nested column flex
// containers to resolve against the stretched cross size.
//
// Takes ctx (context.Context) which carries cancellation signals.
// Takes item (*flexItem) which is the flex item to re-lay out.
// Takes lineCrossSize (float64) which is the definite cross size.
func relayoutFlexItemWithCrossSize(ctx context.Context, item *flexItem, lineCrossSize float64, positionContext *flexPositionContext) {
	child := item.box
	childEdges := resolveEdgesFromStyle(&child.Style, positionContext.containerMainSize)
	childMarginLeft := child.Style.MarginLeft.Resolve(positionContext.containerMainSize, 0)
	childMarginRight := child.Style.MarginRight.Resolve(positionContext.containerMainSize, 0)

	childContentWidth := item.mainSize -
		childEdges.Padding.Horizontal() - childEdges.Border.Horizontal() -
		childMarginLeft - childMarginRight
	childContentWidth = math.Max(0, childContentWidth)

	crossContentHeight := lineCrossSize -
		childEdges.Padding.Vertical() - childEdges.Border.Vertical() -
		childEdges.MarginTop - childEdges.MarginBottom
	if crossContentHeight < 0 {
		crossContentHeight = 0
	}

	originalHeight := child.Style.Height
	child.Style.Height = DimensionPt(crossContentHeight)
	positionContext.input.Cache.Invalidate(child)

	item.fragment = layoutBox(ctx, child, layoutInput{
		AvailableWidth:     childContentWidth,
		AvailableBlockSize: crossContentHeight,
		FontMetrics:        positionContext.input.FontMetrics,
		Cache:              positionContext.input.Cache,
		IsFixedInlineSize:  true,
		Edges:              childEdges,
	})

	child.Style.Height = originalHeight

	item.crossSize = item.fragment.ContentHeight +
		childEdges.Padding.Vertical() + childEdges.Border.Vertical() +
		childEdges.MarginTop + childEdges.MarginBottom
}

// mirrorFlexItemPosition mirrors a flex item's position along
// the main axis when flex-direction is reversed.
//
// Takes fragment (*Fragment) which is the item fragment to mirror.
// Takes containerMainSize (float64) which is the main axis size.
// Takes isRow (bool) which selects horizontal or vertical mirroring.
// Takes contentOffsetX (float64) which is the horizontal content offset.
// Takes contentOffsetY (float64) which is the vertical content offset.
func mirrorFlexItemPosition(fragment *Fragment, containerMainSize float64, isRow bool, contentOffsetX, contentOffsetY float64) {
	if isRow {
		marginBoxStart := fragment.OffsetX - contentOffsetX -
			fragment.Margin.Left - fragment.Padding.Left - fragment.Border.Left
		marginBoxWidth := fragment.Margin.Horizontal() +
			fragment.Padding.Horizontal() + fragment.Border.Horizontal() +
			fragment.ContentWidth
		mirroredStart := containerMainSize - marginBoxStart - marginBoxWidth
		fragment.OffsetX = contentOffsetX + mirroredStart +
			fragment.Margin.Left + fragment.Padding.Left + fragment.Border.Left
	} else {
		marginBoxStart := fragment.OffsetY - contentOffsetY -
			fragment.Margin.Top - fragment.Padding.Top - fragment.Border.Top
		marginBoxHeight := fragment.Margin.Vertical() +
			fragment.Padding.Vertical() + fragment.Border.Vertical() +
			fragment.ContentHeight
		mirroredStart := containerMainSize - marginBoxStart - marginBoxHeight
		fragment.OffsetY = contentOffsetY + mirroredStart +
			fragment.Margin.Top + fragment.Padding.Top + fragment.Border.Top
	}
}

// recomputeLineCrossSize updates the line cross size after
// all items have been laid out, using the actual computed
// cross sizes rather than the pre-layout estimates.
//
// Takes line (*flexLine) which is the flex line to update.
func recomputeLineCrossSize(line *flexLine) {
	maxCrossSize := line.crossSize
	for _, item := range line.items {
		if item.crossSize > maxCrossSize {
			maxCrossSize = item.crossSize
		}
	}
	line.crossSize = maxCrossSize
}

// placeFlexItemOnAxes sets the content position of a flex
// item by mapping main and cross offsets to x and y
// coordinates.
//
// Takes item (*flexItem) which is the flex item to
// place.
// Takes lineCrossSize (float64) which is the cross
// axis size of the item's line.
// Takes crossOffset (float64) which is the cross axis
// offset for the line.
// Takes mainOffset (float64) which is the main axis
// offset for this item.
// Takes positionContext (*flexPositionContext) which holds the
// positioning parameters.
func placeFlexItemOnAxes(
	item *flexItem,
	lineCrossSize float64,
	crossOffset float64,
	mainOffset float64,
	positionContext *flexPositionContext,
) {
	crossPosition := computeCrossPosition(
		item, lineCrossSize, crossOffset, positionContext.alignItems, positionContext.isRowDirection,
	)

	if positionContext.isRowDirection {
		item.fragment.OffsetX = positionContext.contentOffsetX + mainOffset +
			item.fragment.Margin.Left + item.fragment.Padding.Left + item.fragment.Border.Left
		item.fragment.OffsetY = positionContext.contentOffsetY + crossPosition +
			item.fragment.Margin.Top + item.fragment.Padding.Top + item.fragment.Border.Top
	} else {
		item.fragment.OffsetY = positionContext.contentOffsetY + mainOffset +
			item.fragment.Margin.Top + item.fragment.Padding.Top + item.fragment.Border.Top
		item.fragment.OffsetX = positionContext.contentOffsetX + crossPosition +
			item.fragment.Margin.Left + item.fragment.Padding.Left + item.fragment.Border.Left
	}
}

// layoutFlexItem resolves edges, margins, and content
// dimensions for a single flex item, then lays out its
// children.
//
// Takes ctx (context.Context) which carries cancellation
// and deadline signals from the caller.
// Takes child (*LayoutBox) which is the flex item box.
// Takes item (*flexItem) which is the flex item state.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes containerCrossSize (float64) which is the cross
// axis size of the container, or 0 when indefinite.
// Takes input (layoutInput) which carries font metrics
// and cache from the parent context.
func layoutFlexItem(
	ctx context.Context,
	child *LayoutBox,
	item *flexItem,
	isRowDirection bool,
	containerMainSize float64,
	containerCrossSize float64,
	input layoutInput,
) {
	childEdges := resolveEdgesFromStyle(&child.Style, containerMainSize)
	childMarginLeft := child.Style.MarginLeft.Resolve(containerMainSize, 0)
	childMarginRight := child.Style.MarginRight.Resolve(containerMainSize, 0)

	if isRowDirection {
		childContentWidth := item.mainSize -
			childEdges.Padding.Horizontal() - childEdges.Border.Horizontal() -
			childMarginLeft - childMarginRight
		childContentWidth = math.Max(0, childContentWidth)

		item.fragment = layoutBox(ctx, child, layoutInput{
			AvailableWidth:     childContentWidth,
			AvailableBlockSize: containerCrossSize,
			FontMetrics:        input.FontMetrics,
			Cache:              input.Cache,
			IsFixedInlineSize:  true,
			Edges:              childEdges,
		})

		item.crossSize = item.fragment.ContentHeight +
			childEdges.Padding.Vertical() + childEdges.Border.Vertical() +
			childEdges.MarginTop + childEdges.MarginBottom
	} else {
		childContentWidth := item.crossSize -
			childEdges.Padding.Horizontal() - childEdges.Border.Horizontal() -
			childMarginLeft - childMarginRight
		childContentWidth = math.Max(0, childContentWidth)
		item.fragment = layoutBox(ctx, child, layoutInput{
			AvailableWidth:     childContentWidth,
			AvailableBlockSize: containerMainSize,
			FontMetrics:        input.FontMetrics,
			Cache:              input.Cache,
			IsFixedInlineSize:  true,
			Edges:              childEdges,
		})

		mainContentHeight := item.mainSize -
			childEdges.Padding.Vertical() - childEdges.Border.Vertical()
		item.fragment.ContentHeight = math.Max(0, mainContentHeight)

		item.crossSize = item.fragment.ContentWidth +
			childEdges.Padding.Horizontal() + childEdges.Border.Horizontal() +
			childMarginLeft + childMarginRight
	}
}

// computeJustifyOffset calculates the initial main axis
// offset for the first item based on the justify-content
// mode.
//
// Takes justify (JustifyContentType) which is the
// justify-content value.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes line (*flexLine) which is the flex line being
// justified.
// Takes mainGap (float64) which is the gap between
// items along the main axis.
//
// Returns the offset in points.
func computeJustifyOffset(
	justify JustifyContentType,
	containerMainSize float64,
	line *flexLine,
	mainGap float64,
) float64 {
	totalItemSize := 0.0
	for _, item := range line.items {
		totalItemSize += item.mainSize
	}
	totalGaps := mainGap * float64(len(line.items)-1)
	freeSpace := containerMainSize - totalItemSize - totalGaps
	if freeSpace < 0 {
		freeSpace = 0
	}

	switch justify {
	case JustifyFlexEnd:
		return freeSpace
	case JustifyCentre:
		return freeSpace / 2
	case JustifySpaceAround:
		if len(line.items) == 0 {
			return 0
		}
		return freeSpace / float64(len(line.items)) / 2
	case JustifySpaceEvenly:
		return freeSpace / float64(len(line.items)+1)
	default:
		return 0
	}
}

// computeJustifySpacing calculates the extra spacing
// between items based on the justify-content mode.
//
// Takes justify (JustifyContentType) which is the
// justify-content value.
// Takes containerMainSize (float64) which is the main
// axis size of the container.
// Takes line (*flexLine) which is the flex line being
// spaced.
// Takes mainGap (float64) which is the gap between
// items along the main axis.
//
// Returns the per-item spacing in points.
func computeJustifySpacing(
	justify JustifyContentType,
	containerMainSize float64,
	line *flexLine,
	mainGap float64,
) float64 {
	totalItemSize := 0.0
	for _, item := range line.items {
		totalItemSize += item.mainSize
	}
	totalGaps := mainGap * float64(len(line.items)-1)
	freeSpace := containerMainSize - totalItemSize - totalGaps
	if freeSpace < 0 {
		freeSpace = 0
	}

	itemCount := len(line.items)

	switch justify {
	case JustifySpaceBetween:
		if itemCount <= 1 {
			return 0
		}
		return freeSpace / float64(itemCount-1)
	case JustifySpaceAround:
		if itemCount == 0 {
			return 0
		}
		return freeSpace / float64(itemCount)
	case JustifySpaceEvenly:
		return freeSpace / float64(itemCount+1)
	default:
		return 0
	}
}

// computeCrossPosition calculates the cross axis position
// of a flex item within its line, applying align-self or
// the container align-items fallback.
//
// Takes item (*flexItem) which is the flex item to
// position.
// Takes lineCrossSize (float64) which is the cross
// axis size of the item's line.
// Takes crossOffset (float64) which is the cross axis
// offset for the line.
// Takes containerAlignItems (AlignItemsType) which is
// the container's align-items fallback.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
//
// Returns the cross axis offset in points.
func computeCrossPosition(
	item *flexItem,
	lineCrossSize float64,
	crossOffset float64,
	containerAlignItems AlignItemsType,
	isRowDirection bool,
) float64 {
	if hasAutoCrossMargins(item, isRowDirection) {
		return resolveAutoCrossMargins(item, lineCrossSize, crossOffset, isRowDirection)
	}

	alignment := resolveAlignSelf(item.box.Style.AlignSelf, containerAlignItems)

	switch alignment {
	case AlignItemsFlexEnd:
		return crossOffset + lineCrossSize - item.crossSize
	case AlignItemsCentre:
		return crossOffset + (lineCrossSize-item.crossSize)/2
	case AlignItemsStretch:
		applyStretchCrossSize(item, lineCrossSize, isRowDirection)
		return crossOffset
	default:
		return crossOffset
	}
}

// hasAutoCrossMargins returns true if the flex item has any
// auto margins on the cross axis.
//
// Takes item (*flexItem) which is the flex item to check.
// Takes isRowDirection (bool) which selects the cross axis.
//
// Returns bool which is true when auto cross margins exist.
func hasAutoCrossMargins(item *flexItem, isRowDirection bool) bool {
	if isRowDirection {
		return item.box.Style.MarginTop.IsAuto() || item.box.Style.MarginBottom.IsAuto()
	}
	return item.box.Style.MarginLeft.IsAuto() || item.box.Style.MarginRight.IsAuto()
}

// resolveAutoCrossMargins distributes the cross-axis free
// space to auto margins on the cross axis, centering the
// item when both margins are auto.
//
// Takes item (*flexItem) which is the flex item to position.
// Takes lineCrossSize (float64) which is the line's cross size.
// Takes crossOffset (float64) which is the cross axis offset.
// Takes isRowDirection (bool) which selects the cross axis.
//
// Returns float64 which is the resolved cross axis position.
func resolveAutoCrossMargins(item *flexItem, lineCrossSize, crossOffset float64, isRowDirection bool) float64 {
	freeSpace := lineCrossSize - item.crossSize
	if freeSpace < 0 {
		freeSpace = 0
	}

	autoCount := 0
	if isRowDirection {
		if item.box.Style.MarginTop.IsAuto() {
			autoCount++
		}
		if item.box.Style.MarginBottom.IsAuto() {
			autoCount++
		}
	} else {
		if item.box.Style.MarginLeft.IsAuto() {
			autoCount++
		}
		if item.box.Style.MarginRight.IsAuto() {
			autoCount++
		}
	}

	if autoCount == 0 {
		return crossOffset
	}

	perMargin := freeSpace / float64(autoCount)

	startMargin := 0.0
	hasCrossStartAutoMargin := (isRowDirection && item.box.Style.MarginTop.IsAuto()) ||
		(!isRowDirection && item.box.Style.MarginLeft.IsAuto())
	if hasCrossStartAutoMargin {
		startMargin = perMargin
	}

	return crossOffset + startMargin
}

// applyStretchCrossSize stretches a flex item to fill the
// line cross size when the item has an auto cross
// dimension.
//
// Takes item (*flexItem) which is the flex item to
// stretch.
// Takes lineCrossSize (float64) which is the cross
// axis size of the item's line.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
func applyStretchCrossSize(item *flexItem, lineCrossSize float64, isRowDirection bool) {
	fragment := item.fragment
	style := &item.box.Style

	if isRowDirection && style.Height.IsAuto() {
		stretchedCross := lineCrossSize -
			fragment.Padding.Vertical() - fragment.Border.Vertical() -
			fragment.Margin.Vertical()
		stretchedCross = clampStretchDimension(stretchedCross, style, false)
		if stretchedCross > fragment.ContentHeight {
			fragment.ContentHeight = stretchedCross
		}
		item.crossSize = fragment.ContentHeight +
			fragment.Padding.Vertical() + fragment.Border.Vertical() +
			fragment.Margin.Vertical()
		return
	}

	if !isRowDirection && style.Width.IsAuto() {
		stretchedCross := lineCrossSize -
			fragment.Padding.Horizontal() - fragment.Border.Horizontal() -
			fragment.Margin.Horizontal()
		stretchedCross = clampStretchDimension(stretchedCross, style, true)
		if stretchedCross > fragment.ContentWidth {
			fragment.ContentWidth = stretchedCross
		}
		item.crossSize = fragment.ContentWidth +
			fragment.Padding.Horizontal() + fragment.Border.Horizontal() +
			fragment.Margin.Horizontal()
	}
}

// resolveAlignSelf maps an align-self value to the
// corresponding align-items value, falling back to the
// container align-items when align-self is auto.
//
// Takes alignSelf (AlignSelfType) which is the item's
// align-self value.
// Takes containerAlignItems (AlignItemsType) which is
// the container's align-items fallback.
//
// Returns the resolved AlignItemsType.
func resolveAlignSelf(alignSelf AlignSelfType, containerAlignItems AlignItemsType) AlignItemsType {
	switch alignSelf {
	case AlignSelfFlexStart:
		return AlignItemsFlexStart
	case AlignSelfFlexEnd:
		return AlignItemsFlexEnd
	case AlignSelfCentre:
		return AlignItemsCentre
	case AlignSelfStretch:
		return AlignItemsStretch
	case AlignSelfBaseline:
		return AlignItemsBaseline
	default:
		return containerAlignItems
	}
}

// computeFlexContainerHeight computes the container content
// height based on the resolved flex line sizes.
//
// Takes container (*LayoutBox) which is the flex
// container box used for style queries.
// Takes lines ([]*flexLine) which is the slice of
// resolved flex lines.
// Takes isRowDirection (bool) which is true when the
// main axis is horizontal.
// Takes crossGap (float64) which is the gap between
// lines along the cross axis.
//
// Returns float64 which is the resolved content height.
func computeFlexContainerHeight(
	container *LayoutBox,
	lines []*flexLine,
	isRowDirection bool,
	crossGap float64,
) float64 {
	if isRowDirection {
		totalCrossSize := 0.0
		for index, line := range lines {
			totalCrossSize += line.crossSize
			if index > 0 {
				totalCrossSize += crossGap
			}
		}
		if container.Style.Height.IsAuto() {
			return totalCrossSize
		}
	} else {
		maxMainSize := 0.0
		for _, line := range lines {
			lineMainSize := 0.0
			for _, item := range line.items {
				lineMainSize += item.mainSize
			}
			maxMainSize = math.Max(maxMainSize, lineMainSize)
		}
		if container.Style.Height.IsAuto() {
			return maxMainSize
		}
	}
	return adjustForBoxSizing(container.Style.Height.Resolve(0, 0), &container.Style, false)
}
