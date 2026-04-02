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

// LayoutBoxTree performs layout on the entire box tree.
//
// Takes ctx (context.Context) which carries the
// cancellation signal through recursive layout.
// Takes root (*LayoutBox) which is the root box of the
// tree to lay out.
// Takes fontMetrics (FontMetricsPort) which provides font
// measurement capabilities for text layout.
//
// Returns *Fragment which holds the layout results for the
// entire box tree.
func LayoutBoxTree(ctx context.Context, root *LayoutBox, fontMetrics FontMetricsPort) *Fragment {
	viewportHeight := root.ContentHeight
	cache := newLayoutCache()
	input := layoutInput{
		AvailableWidth:     root.ContentWidth,
		AvailableBlockSize: root.ContentHeight,
		FontMetrics:        fontMetrics,
		Cache:              cache,
	}
	fragment := layoutBox(ctx, root, input)
	writeFragmentsToBoxTree(fragment, 0, 0)
	root.ContentHeight = viewportHeight
	layoutListMarkers(root, fontMetrics)
	applyAllRelativeOffsets(root)
	layoutPositionedElements(ctx, root, input)
	return fragment
}

// layoutListMarkers walks the box tree and positions
// outside list markers now that all list items have
// their final coordinates.
//
// Takes box (*LayoutBox) which is the root of the tree
// to walk.
// Takes fontMetrics (FontMetricsPort) which provides
// text measurement for marker sizing.
func layoutListMarkers(box *LayoutBox, fontMetrics FontMetricsPort) {
	if box.Type == BoxListItem {
		layoutOutsideListMarker(box, fontMetrics)
	}
	for _, child := range box.Children {
		layoutListMarkers(child, fontMetrics)
	}
}

// boxDimensions holds the resolved edges and content width
// for a box before formatting context layout.
type boxDimensions struct {
	// edges holds the resolved padding, border, and vertical margin values.
	edges resolvedEdges

	// contentWidth holds the resolved content area width in points.
	contentWidth float64

	// marginLeft holds the resolved left margin in points.
	marginLeft float64

	// marginRight holds the resolved right margin in points.
	marginRight float64
}

// layoutBox dispatches layout for a single box based on its
// type and children. Returns a Fragment capturing the layout
// results with parent-relative offsets.
//
// Takes box (*LayoutBox) which is the box to lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics from the parent context.
//
// Returns *Fragment which captures the layout results for
// this box and its descendants.
func layoutBox(ctx context.Context, box *LayoutBox, input layoutInput) *Fragment {
	if ctx.Err() != nil {
		return &Fragment{Box: box}
	}

	if cached := input.Cache.Lookup(box, input); cached != nil {
		return cached
	}

	if box.Type == BoxTextRun {
		fragment := layoutTextRun(box, input.FontMetrics)
		input.Cache.Store(box, input, fragment)
		return fragment
	}

	dims := resolveBoxDimensions(box, input)

	fcResult, ok := runFormattingContext(ctx, box, dims, input)
	if !ok {
		return &Fragment{Box: box}
	}

	contentHeight := resolveBoxContentHeight(box, fcResult.ContentHeight, &box.Style, dims.contentWidth, input.AvailableBlockSize)
	contentWidth, contentHeight := clampDimensions(dims.contentWidth, contentHeight, &box.Style, input.AvailableWidth, input.AvailableBlockSize, box, input.FontMetrics)

	borderEdges := dims.edges.Border
	if fcResult.HasBorder {
		borderEdges = fcResult.Border
	}

	fragment := &Fragment{
		Box:           box,
		Children:      fcResult.Children,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
		Padding:       dims.edges.Padding,
		Border:        borderEdges,
		Margin: BoxEdges{
			Top:    fcResult.Margin.Top,
			Bottom: fcResult.Margin.Bottom,
			Left:   dims.marginLeft,
			Right:  dims.marginRight,
		},
	}
	input.Cache.Store(box, input, fragment)
	return fragment
}

// runFormattingContext resolves which formatting context applies
// to the box, builds the layout input for it, and runs the
// layout pass.
//
// Takes box (*LayoutBox) which is the box to lay out.
// Takes dims (boxDimensions) which holds the resolved edges
// and content width.
// Takes input (layoutInput) which carries the available width
// and font metrics from the parent context.
//
// Returns formattingContextResult with the layout results
// and bool which is false when the formatting context cannot
// be resolved.
func runFormattingContext(
	ctx context.Context,
	box *LayoutBox,
	dims boxDimensions,
	input layoutInput,
) (formattingContextResult, bool) {
	formattingContext, formattingContextError := resolveFormattingContext(box)
	if formattingContextError != nil {
		return formattingContextResult{}, false
	}
	fcInput := layoutInput{
		AvailableWidth:      dims.contentWidth,
		AvailableBlockSize:  resolveAvailableBlockSize(&box.Style, input.AvailableBlockSize),
		FontMetrics:         input.FontMetrics,
		Cache:               input.Cache,
		SizingMode:          input.SizingMode,
		Edges:               dims.edges,
		Floats:              input.Floats,
		FloatBFCOffsetY:     input.FloatBFCOffsetY,
		FloatContainerX:     input.FloatContainerX,
		FloatContainerWidth: input.FloatContainerWidth,
	}
	return formattingContext.Layout(ctx, box, fcInput), true
}

// resolveBoxContentHeight applies replaced-element fallback,
// explicit height resolution, and aspect ratio to produce the
// final content height for the box.
//
// Takes box (*LayoutBox) which is the box being resolved.
// Takes fcHeight (float64) which is the height from the
// formatting context.
// Takes style (*ComputedStyle) which carries the height
// property.
// Takes contentWidth (float64) which is the resolved content
// width for aspect ratio computation.
// Takes availableBlockSize (float64) which is the block size
// used for percentage resolution.
//
// Returns float64 which is the final content height.
func resolveBoxContentHeight(
	box *LayoutBox,
	fcHeight float64,
	style *ComputedStyle,
	contentWidth float64,
	availableBlockSize float64,
) float64 {
	if box.Type == BoxReplaced && fcHeight == 0 && box.IntrinsicHeight > 0 && style.Height.IsAuto() {
		fcHeight = box.IntrinsicHeight
	}
	contentHeight := resolveContentHeight(fcHeight, style, availableBlockSize)
	return applyAspectRatio(contentWidth, contentHeight, style, box)
}

// resolveBoxDimensions resolves the edges, content width, and
// horizontal margins for a box from its style and layout input.
//
// Takes box (*LayoutBox) which is the box to resolve.
// Takes input (layoutInput) which carries the available width
// and font metrics from the parent context.
//
// Returns boxDimensions with the resolved values.
func resolveBoxDimensions(box *LayoutBox, input layoutInput) boxDimensions {
	if input.IsFixedInlineSize {
		return boxDimensions{
			edges:        input.Edges,
			contentWidth: input.AvailableWidth,
			marginLeft:   box.Style.MarginLeft.Resolve(input.AvailableWidth, 0),
			marginRight:  box.Style.MarginRight.Resolve(input.AvailableWidth, 0),
		}
	}

	edges := resolveEdgesFromStyle(&box.Style, input.AvailableWidth)
	widthResult := resolveWidthFromStyle(&box.Style, edges, input.AvailableWidth, box, input.FontMetrics, input.ContainingBlockDirection)
	return boxDimensions{
		edges:        edges,
		contentWidth: widthResult.ContentWidth,
		marginLeft:   widthResult.MarginLeft,
		marginRight:  widthResult.MarginRight,
	}
}

// resolvedWidth holds the content width and horizontal
// margin values computed by resolveWidthFromStyle.
type resolvedWidth struct {
	// ContentWidth is the resolved content area width.
	ContentWidth float64

	// MarginLeft is the resolved left margin.
	MarginLeft float64

	// MarginRight is the resolved right margin.
	MarginRight float64
}

// resolveWidthFromStyle computes content width and
// horizontal margins from a computed style without
// mutating any box. The box parameter is only used for
// intrinsic sizing measurement (min-content, max-content).
//
// Takes style (*ComputedStyle) which is the style to
// resolve.
// Takes edges (resolvedEdges) which are the resolved
// padding and border values.
// Takes availableWidth (float64) which is the width
// available from the containing block.
// Takes box (*LayoutBox) which is the box used for
// intrinsic sizing measurement only.
// Takes fontMetrics (FontMetricsPort) which provides
// text measurement for intrinsic sizing.
//
// Returns resolvedWidth with the resolved values.
func resolveWidthFromStyle(
	style *ComputedStyle,
	edges resolvedEdges,
	availableWidth float64,
	box *LayoutBox,
	fontMetrics FontMetricsPort,
	containingBlockDirection DirectionType,
) resolvedWidth {
	horizontalEdges := edges.Padding.Horizontal() + edges.Border.Horizontal()

	if style.Width.IsMinContent() {
		intrinsicWidth := measureMinContentWidth(box, fontMetrics) -
			style.PaddingLeft - style.PaddingRight -
			style.BorderLeftWidth - style.BorderRightWidth
		if intrinsicWidth < 0 {
			intrinsicWidth = 0
		}
		return resolvedWidth{
			ContentWidth: intrinsicWidth,
			MarginLeft:   style.MarginLeft.Resolve(availableWidth, 0),
			MarginRight:  style.MarginRight.Resolve(availableWidth, 0),
		}
	}

	if style.Width.IsMaxContent() {
		intrinsicWidth := measureMaxContentWidth(box, fontMetrics) -
			style.PaddingLeft - style.PaddingRight -
			style.BorderLeftWidth - style.BorderRightWidth
		if intrinsicWidth < 0 {
			intrinsicWidth = 0
		}
		return resolvedWidth{
			ContentWidth: intrinsicWidth,
			MarginLeft:   style.MarginLeft.Resolve(availableWidth, 0),
			MarginRight:  style.MarginRight.Resolve(availableWidth, 0),
		}
	}

	if style.Width.IsFitContent() {
		return resolveFitContentWidth(style, availableWidth, horizontalEdges, box, fontMetrics)
	}

	if style.Width.IsAuto() {
		if box != nil && box.Type == BoxReplaced && box.IntrinsicWidth > 0 {
			return resolvedWidth{
				ContentWidth: box.IntrinsicWidth,
				MarginLeft:   style.MarginLeft.Resolve(availableWidth, 0),
				MarginRight:  style.MarginRight.Resolve(availableWidth, 0),
			}
		}
		return resolveAutoWidthFromStyle(style, availableWidth, horizontalEdges)
	}
	return resolveExplicitWidthFromStyle(style, availableWidth, horizontalEdges, containingBlockDirection)
}

// resolveFitContentWidth resolves width for a box with
// width: fit-content or fit-content(<arg>). The result is
// min(max-content, max(min-content, clamp)) where clamp
// is the argument or the available width for bare fit-content.
//
// Takes style (*ComputedStyle) which is the style to resolve.
// Takes availableWidth (float64) which is the width available
// from the containing block.
// Takes horizontalEdges (float64) which is the sum of
// horizontal padding and border.
// Takes box (*LayoutBox) which is the box used for intrinsic
// sizing measurement.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement for intrinsic sizing.
//
// Returns resolvedWidth with the resolved values.
func resolveFitContentWidth(
	style *ComputedStyle, availableWidth, horizontalEdges float64,
	box *LayoutBox, fontMetrics FontMetricsPort,
) resolvedWidth {
	edgeSum := style.PaddingLeft + style.PaddingRight +
		style.BorderLeftWidth + style.BorderRightWidth
	minW := math.Max(0, measureMinContentWidth(box, fontMetrics)-edgeSum)
	maxW := math.Max(0, measureMaxContentWidth(box, fontMetrics)-edgeSum)

	clamp := availableWidth - horizontalEdges
	if style.Width.Unit == DimensionUnitFitContent {
		clamp = style.Width.Value
	}
	clamp = math.Max(0, clamp)

	return resolvedWidth{
		ContentWidth: math.Min(maxW, math.Max(minW, clamp)),
		MarginLeft:   style.MarginLeft.Resolve(availableWidth, 0),
		MarginRight:  style.MarginRight.Resolve(availableWidth, 0),
	}
}

// resolveAutoWidthFromStyle resolves width for a box
// with width: auto by filling the available space.
//
// Takes style (*ComputedStyle) which is the style to
// resolve.
// Takes availableWidth (float64) which is the width
// available from the containing block.
// Takes horizontalEdges (float64) which is the sum of
// horizontal padding and border.
//
// Returns resolvedWidth with the resolved values.
func resolveAutoWidthFromStyle(style *ComputedStyle, availableWidth, horizontalEdges float64) resolvedWidth {
	marginLeft := style.MarginLeft.Resolve(availableWidth, 0)
	marginRight := style.MarginRight.Resolve(availableWidth, 0)
	contentWidth := availableWidth - horizontalEdges - marginLeft - marginRight
	if contentWidth < 0 {
		contentWidth = 0
	}
	return resolvedWidth{
		ContentWidth: contentWidth,
		MarginLeft:   marginLeft,
		MarginRight:  marginRight,
	}
}

// resolveExplicitWidthFromStyle resolves width for a box
// with an explicit width value and distributes remaining
// space to auto margins.
//
// Takes style (*ComputedStyle) which is the style to
// resolve.
// Takes availableWidth (float64) which is the width
// available from the containing block.
// Takes horizontalEdges (float64) which is the sum of
// horizontal padding and border.
//
// Returns resolvedWidth with the resolved values.
func resolveExplicitWidthFromStyle(style *ComputedStyle, availableWidth, horizontalEdges float64, containingBlockDirection DirectionType) resolvedWidth {
	contentWidth := adjustForBoxSizing(
		style.Width.Resolve(availableWidth, availableWidth),
		style, true,
	)

	var marginLeft, marginRight float64
	switch {
	case style.MarginLeft.IsAuto() && style.MarginRight.IsAuto():
		remaining := availableWidth - contentWidth - horizontalEdges
		if remaining > 0 {
			marginLeft = remaining / 2
			marginRight = remaining / 2
		}
	case style.MarginLeft.IsAuto():
		marginRight = style.MarginRight.Resolve(availableWidth, 0)
		marginLeft = math.Max(0, availableWidth-contentWidth-horizontalEdges-marginRight)
	case style.MarginRight.IsAuto():
		marginLeft = style.MarginLeft.Resolve(availableWidth, 0)
		marginRight = math.Max(0, availableWidth-contentWidth-horizontalEdges-marginLeft)
	default:
		marginLeft = style.MarginLeft.Resolve(availableWidth, 0)
		marginRight = style.MarginRight.Resolve(availableWidth, 0)

		if containingBlockDirection == DirectionRTL {
			remaining := availableWidth - contentWidth - horizontalEdges - marginLeft - marginRight
			if remaining > 0 {
				marginLeft += remaining
			}
		}
	}

	return resolvedWidth{
		ContentWidth: contentWidth,
		MarginLeft:   marginLeft,
		MarginRight:  marginRight,
	}
}

// adjustForBoxSizing converts a resolved declared dimension to a content
// dimension by subtracting padding and border along the relevant axis when
// box-sizing is border-box. When box-sizing is content-box, the value is
// returned unchanged.
//
// For border-box, the declared value represents the border-box size, so the
// content size is declared minus padding minus border. The result is floored
// at zero to prevent negative content dimensions.
//
// Takes declared (float64) which is the resolved dimension value.
// Takes style (*ComputedStyle) which provides box-sizing and edge values.
// Takes horizontal (bool) which selects the axis: true for width
// (left+right edges), false for height (top+bottom edges).
//
// Returns float64 which is the content dimension.
func adjustForBoxSizing(declared float64, style *ComputedStyle, horizontal bool) float64 {
	if style.BoxSizing != BoxSizingBorderBox {
		return declared
	}
	var edges float64
	if horizontal {
		edges = style.PaddingLeft + style.PaddingRight +
			style.BorderLeftWidth + style.BorderRightWidth
	} else {
		edges = style.PaddingTop + style.PaddingBottom +
			style.BorderTopWidth + style.BorderBottomWidth
	}
	if declared-edges < 0 {
		return 0
	}
	return declared - edges
}

// resolvedEdges holds the padding, border, and vertical
// margin values resolved from a ComputedStyle. This is
// the return type of resolveEdgesFromStyle.
type resolvedEdges struct {
	// Padding holds the resolved padding values in points.
	Padding BoxEdges

	// Border holds the resolved border widths in points.
	Border BoxEdges

	// MarginTop is the resolved top margin in points.
	MarginTop float64

	// MarginBottom is the resolved bottom margin in points.
	MarginBottom float64
}

// resolveEdgesFromStyle computes padding, border, and
// vertical margin values from a ComputedStyle without
// mutating any box. Horizontal margins are not resolved
// here because they depend on the width resolution
// algorithm.
//
// Takes style (*ComputedStyle) which is the style to
// read properties from.
// Takes marginPercentageBasis (float64) which is the
// reference size for resolving percentage margins.
//
// Returns resolvedEdges with the resolved values.
func resolveEdgesFromStyle(style *ComputedStyle, marginPercentageBasis float64) resolvedEdges {
	return resolvedEdges{
		Padding: BoxEdges{
			Top:    style.PaddingTop,
			Right:  style.PaddingRight,
			Bottom: style.PaddingBottom,
			Left:   style.PaddingLeft,
		},
		Border: BoxEdges{
			Top:    style.BorderTopWidth,
			Right:  style.BorderRightWidth,
			Bottom: style.BorderBottomWidth,
			Left:   style.BorderLeftWidth,
		},
		MarginTop:    style.MarginTop.Resolve(marginPercentageBasis, 0),
		MarginBottom: style.MarginBottom.Resolve(marginPercentageBasis, 0),
	}
}

// blockLayoutContext holds mutable state while laying out
// the block-level children of a single box.
type blockLayoutContext struct {
	// box is the parent box being laid out.
	box *LayoutBox

	// floats tracks active left and right floats.
	floats floatContext

	// input carries the available width and font metrics
	// from the parent context.
	input layoutInput

	// cursorY is the current vertical position for the
	// next in-flow child, relative to the parent's
	// ContentY.
	cursorY float64

	// previousMarginBottom is the bottom margin of the
	// most recently laid-out in-flow child.
	previousMarginBottom float64

	// childAvailableWidth is the width available to
	// children within the content area.
	childAvailableWidth float64

	// parentMarginTop tracks the parent's top margin,
	// updated by parent-child margin collapsing.
	parentMarginTop float64

	// parentMarginBottom tracks the parent's bottom
	// margin, updated by parent-child margin collapsing.
	parentMarginBottom float64

	// firstInFlowIndex is the index of the first in-flow
	// child, or -1 if none has been laid out yet.
	firstInFlowIndex int
}

// layoutBlockChildren lays out children of a block container
// using the block formatting context algorithm, returning
// child fragments with parent-relative offsets, the intrinsic
// content height, and collapsed margin information.
//
// Takes box (*LayoutBox) which is the parent block
// container.
// Takes input (layoutInput) which carries the available
// width and font metrics from the parent context.
//
// Returns formattingContextResult with the layout results
// for this container and its children.
func layoutBlockChildren(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	layoutContext := blockLayoutContext{
		box:                 box,
		input:               input,
		cursorY:             0,
		childAvailableWidth: input.AvailableWidth,
		parentMarginTop:     input.Edges.MarginTop,
		parentMarginBottom:  input.Edges.MarginBottom,
		firstInFlowIndex:    -1,
	}

	var childFragments []*Fragment
	for index, child := range box.Children {
		if child.Type == BoxListMarker {
			continue
		}
		if child.Style.Position == PositionAbsolute || child.Style.Position == PositionFixed {
			continue
		}
		if child.Style.Float != FloatNone {
			childFragment := layoutContext.layoutFloatChild(ctx, child)
			childFragments = append(childFragments, childFragment)
			continue
		}
		childFragment := layoutContext.layoutInFlowChild(ctx, child, index)
		childFragments = append(childFragments, childFragment)
	}

	contentHeight := layoutContext.finaliseBlockHeight()

	return formattingContextResult{
		Children:      childFragments,
		ContentHeight: contentHeight,
		Margin: BoxEdges{
			Top:    layoutContext.parentMarginTop,
			Bottom: layoutContext.parentMarginBottom,
		},
	}
}

// layoutFloatChild lays out a floated child and positions it
// using the float placement algorithm. Returns a Fragment
// with parent-relative offsets.
//
// Takes child (*LayoutBox) which is the floated child
// box to lay out and position.
//
// Returns *Fragment with the float's layout results.
func (layoutContext *blockLayoutContext) layoutFloatChild(ctx context.Context, child *LayoutBox) *Fragment {
	childInput := layoutInput{
		AvailableWidth:           layoutContext.childAvailableWidth,
		AvailableBlockSize:       layoutContext.input.AvailableBlockSize,
		FontMetrics:              layoutContext.input.FontMetrics,
		Cache:                    layoutContext.input.Cache,
		ContainingBlockDirection: layoutContext.box.Style.Direction,
	}
	childFragment := layoutBox(ctx, child, childInput)

	floatWidth := childFragment.MarginBoxWidth()
	floatHeight := childFragment.MarginBoxHeight()

	contentStartX := 0.0
	var x, y float64
	if child.Style.Float == FloatLeft {
		x, y = layoutContext.floats.placeLeftFloat(
			layoutContext.cursorY, floatWidth, floatHeight, contentStartX, layoutContext.childAvailableWidth,
		)
	} else {
		x, y = layoutContext.floats.placeRightFloat(
			layoutContext.cursorY, floatWidth, floatHeight, contentStartX, layoutContext.childAvailableWidth,
		)
	}

	childFragment.OffsetX = x + childFragment.Margin.Left + childFragment.Padding.Left + childFragment.Border.Left
	childFragment.OffsetY = y + childFragment.Margin.Top + childFragment.Padding.Top + childFragment.Border.Top
	return childFragment
}

// layoutInFlowChild lays out a normal-flow child, applying
// float clearing and margin collapsing. Returns a Fragment
// with parent-relative offsets.
//
// Takes child (*LayoutBox) which is the in-flow child
// box to lay out.
// Takes index (int) which is the child's position in
// the parent's children slice.
//
// Returns *Fragment with the child's layout results.
func (layoutContext *blockLayoutContext) layoutInFlowChild(ctx context.Context, child *LayoutBox, index int) *Fragment {
	if child.Style.Clear != ClearNone {
		clearY := layoutContext.floats.clearY(child.Style.Clear)
		if clearY > layoutContext.cursorY {
			layoutContext.cursorY = clearY
		}
	}

	contentStartX := 0.0
	effectiveWidth := layoutContext.childAvailableWidth
	leftOffset := 0.0

	if child.Type != BoxBlock && child.Type != BoxAnonymousBlock {
		effectiveWidth = layoutContext.floats.availableWidthAtY(
			layoutContext.cursorY, 0, contentStartX, layoutContext.childAvailableWidth,
		)
		leftOffset = layoutContext.floats.leftOffsetAtY(layoutContext.cursorY, 0, contentStartX)
	}

	childInput := layoutInput{
		AvailableWidth:           effectiveWidth,
		AvailableBlockSize:       layoutContext.input.AvailableBlockSize,
		FontMetrics:              layoutContext.input.FontMetrics,
		Cache:                    layoutContext.input.Cache,
		Floats:                   &layoutContext.floats,
		FloatBFCOffsetY:          layoutContext.cursorY,
		FloatContainerX:          contentStartX,
		FloatContainerWidth:      layoutContext.childAvailableWidth,
		ContainingBlockDirection: layoutContext.box.Style.Direction,
	}
	childFragment := layoutBox(ctx, child, childInput)
	layoutContext.collapseChildMarginTop(childFragment, index)

	childFragment.OffsetX = leftOffset +
		childFragment.Margin.Left + childFragment.Padding.Left + childFragment.Border.Left
	childFragment.OffsetY = layoutContext.cursorY + childFragment.Padding.Top + childFragment.Border.Top

	layoutContext.cursorY += childFragment.MarginBoxHeight()
	layoutContext.previousMarginBottom = childFragment.Margin.Bottom
	return childFragment
}

// collapseChildMarginTop collapses the top margin of a child
// fragment with the previous sibling's bottom margin or the
// parent's top margin.
//
// Takes childFragment (*Fragment) which is the child
// whose top margin may be collapsed.
// Takes index (int) which is the child's position in
// the parent's children slice.
func (layoutContext *blockLayoutContext) collapseChildMarginTop(childFragment *Fragment, index int) {
	childMarginTop := childFragment.Margin.Top

	if layoutContext.firstInFlowIndex == -1 {
		layoutContext.firstInFlowIndex = index
		if canCollapseParentChildTop(layoutContext.box, layoutContext.input.Edges) {
			layoutContext.parentMarginTop = collapseMargins(layoutContext.parentMarginTop, childMarginTop)
			childMarginTop = 0
		}
		layoutContext.cursorY += childMarginTop
		childFragment.Margin.Top = 0
		return
	}

	collapsedMargin := collapseMargins(layoutContext.previousMarginBottom, childMarginTop)
	layoutContext.cursorY -= layoutContext.previousMarginBottom
	layoutContext.cursorY += collapsedMargin
	childFragment.Margin.Top = 0
}

// finaliseBlockHeight computes the final content height of
// the parent box after all children have been laid out.
//
// Returns float64 which is the computed content height.
func (layoutContext *blockLayoutContext) finaliseBlockHeight() float64 {
	if layoutContext.firstInFlowIndex >= 0 && canCollapseParentChildBottom(layoutContext.box, layoutContext.input.Edges) {
		layoutContext.parentMarginBottom = collapseMargins(
			layoutContext.parentMarginBottom, layoutContext.previousMarginBottom,
		)
		layoutContext.cursorY -= layoutContext.previousMarginBottom
		layoutContext.previousMarginBottom = 0
	}

	if establishesBlockFormattingContext(layoutContext.box) || layoutContext.box.Parent == nil {
		floatBottomY := layoutContext.floats.clearBothY()
		if floatBottomY > layoutContext.cursorY {
			layoutContext.cursorY = floatBottomY
		}
	}

	contentBottom := layoutContext.cursorY
	if layoutContext.firstInFlowIndex >= 0 {
		contentBottom += layoutContext.previousMarginBottom
	}
	if contentBottom < 0 {
		contentBottom = 0
	}
	return contentBottom
}

// canCollapseParentChildTop reports whether the parent's top
// margin can collapse with its first child's top margin.
//
// Takes box (*LayoutBox) which is the parent box to
// check.
// Takes edges (resolvedEdges) which are the resolved
// padding and border values for the parent.
//
// Returns bool which is true when the parent's top
// margin can collapse with the first child.
func canCollapseParentChildTop(box *LayoutBox, edges resolvedEdges) bool {
	if edges.Padding.Top != 0 || edges.Border.Top != 0 {
		return false
	}
	return !establishesBlockFormattingContext(box)
}

// canCollapseParentChildBottom reports whether the parent's
// bottom margin can collapse with its last child's bottom
// margin.
//
// Takes box (*LayoutBox) which is the parent box to
// check.
// Takes edges (resolvedEdges) which are the resolved
// padding and border values for the parent.
//
// Returns bool which is true when the parent's bottom
// margin can collapse with the last child.
func canCollapseParentChildBottom(box *LayoutBox, edges resolvedEdges) bool {
	if edges.Padding.Bottom != 0 || edges.Border.Bottom != 0 {
		return false
	}
	if !box.Style.Height.IsAuto() {
		return false
	}
	if !box.Style.MinHeight.IsAuto() && box.Style.MinHeight.Resolve(0, 0) > 0 {
		return false
	}
	return !establishesBlockFormattingContext(box)
}

// establishesBlockFormattingContext reports whether a box
// establishes a new block formatting context.
//
// Takes box (*LayoutBox) which is the box to check.
//
// Returns bool which is true when the box establishes
// a new block formatting context.
func establishesBlockFormattingContext(box *LayoutBox) bool {
	if box.Style.OverflowX != OverflowVisible || box.Style.OverflowY != OverflowVisible {
		return true
	}
	if box.Style.Float != FloatNone {
		return true
	}
	if box.Style.Position == PositionAbsolute || box.Style.Position == PositionFixed {
		return true
	}
	if box.Style.Display == DisplayInlineBlock || box.Style.Display == DisplayFlex ||
		box.Style.Display == DisplayInlineFlex ||
		box.Style.Display == DisplayGrid || box.Style.Display == DisplayInlineGrid {
		return true
	}

	if box.Type == BoxGridItem || box.Type == BoxFlexItem {
		return true
	}
	return false
}

// resolveAvailableBlockSize returns the definite block
// size for this box if it has an explicit height, or 0
// (indefinite) when height is auto.
//
// Takes style (*ComputedStyle) which carries the height
// property.
// Takes parentBlockSize (float64) which is the parent
// block size used for percentage resolution.
//
// Returns float64 which is the resolved block size.
func resolveAvailableBlockSize(style *ComputedStyle, parentBlockSize float64) float64 {
	if !style.Height.IsAuto() && !style.Height.IsIntrinsic() {
		return adjustForBoxSizing(
			style.Height.Resolve(parentBlockSize, 0),
			style, false,
		)
	}
	return 0
}

// resolveContentHeight returns the effective content
// height by applying an explicit height from the style
// if one is specified, otherwise falls back to the
// intrinsic height from the formatting context.
//
// Takes intrinsicHeight (float64) which is the height
// computed by the formatting context.
// Takes style (*ComputedStyle) which carries the height
// property.
// Takes availableBlockSize (float64) which is the block
// size used for percentage resolution.
//
// Returns float64 which is the resolved content height.
func resolveContentHeight(intrinsicHeight float64, style *ComputedStyle, availableBlockSize float64) float64 {
	if style.Height.IsAuto() || style.Height.IsFitContent() {
		return intrinsicHeight
	}
	return adjustForBoxSizing(
		style.Height.Resolve(availableBlockSize, intrinsicHeight),
		style, false,
	)
}

// applyAspectRatio computes height from width and the CSS
// aspect-ratio when height is auto. For replaced elements
// with AspectRatioAuto, intrinsic dimensions take
// precedence.
//
// Takes contentWidth (float64) which is the resolved content
// width.
// Takes contentHeight (float64) which is the current content
// height.
// Takes style (*ComputedStyle) which carries the aspect-ratio
// property.
// Takes box (*LayoutBox) which is the box being resolved.
//
// Returns float64 which is the adjusted content height.
func applyAspectRatio(contentWidth, contentHeight float64, style *ComputedStyle, box *LayoutBox) float64 {
	if style.AspectRatio <= 0 {
		return contentHeight
	}
	if !style.Height.IsAuto() {
		return contentHeight
	}
	if style.AspectRatioAuto && box.Type == BoxReplaced && box.IntrinsicHeight > 0 {
		return contentHeight
	}
	return contentWidth / style.AspectRatio
}

// clampDimensions applies min/max width and height
// constraints from the computed style and returns the
// clamped values.
//
// Takes contentWidth (float64) which is the resolved
// content width.
// Takes contentHeight (float64) which is the resolved
// content height.
// Takes style (*ComputedStyle) which carries the min/max
// properties.
// Takes box (*LayoutBox) and fontMetrics (FontMetricsPort)
// for intrinsic measurement when min/max uses fit-content.
//
// Returns the clamped width and height.
func clampDimensions(
	contentWidth, contentHeight float64,
	style *ComputedStyle, containingBlockWidth, availableBlockSize float64,
	box *LayoutBox, fontMetrics FontMetricsPort,
) (clampedWidth, clampedHeight float64) {
	contentWidth = clampMaxWidth(contentWidth, style, containingBlockWidth, box, fontMetrics)
	contentWidth = clampMinWidth(contentWidth, style, containingBlockWidth, box, fontMetrics)
	if !style.MaxHeight.IsAuto() && !style.MaxHeight.IsFitContent() {
		if style.MaxHeight.Unit != DimensionUnitPercentage || availableBlockSize > 0 {
			maxHeight := adjustForBoxSizing(style.MaxHeight.Resolve(availableBlockSize, contentHeight), style, false)
			contentHeight = math.Min(contentHeight, maxHeight)
		}
	}
	if !style.MinHeight.IsAuto() && !style.MinHeight.IsFitContent() {
		minHeight := adjustForBoxSizing(style.MinHeight.Resolve(availableBlockSize, 0), style, false)
		contentHeight = math.Max(contentHeight, minHeight)
	}
	return contentWidth, contentHeight
}

// clampMinWidth applies the min-width constraint from the
// computed style to the given content width.
//
// Takes contentWidth (float64) which is the current content
// width.
// Takes style (*ComputedStyle) which carries the min-width
// property.
// Takes containingBlockWidth (float64) which is the containing
// block width for percentage resolution.
// Takes box (*LayoutBox) which is used for intrinsic sizing
// measurement.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement for intrinsic sizing.
//
// Returns float64 which is the clamped content width.
func clampMinWidth(contentWidth float64, style *ComputedStyle, containingBlockWidth float64, box *LayoutBox, fontMetrics FontMetricsPort) float64 {
	if style.MinWidth.IsAuto() {
		return contentWidth
	}
	if style.MinWidth.IsFitContent() {
		fitContent := resolveFitContentValue(style.MinWidth, box, fontMetrics, containingBlockWidth, style)
		return math.Max(contentWidth, fitContent)
	}
	minWidth := adjustForBoxSizing(style.MinWidth.Resolve(containingBlockWidth, 0), style, true)
	return math.Max(contentWidth, minWidth)
}

// clampMaxWidth applies the max-width constraint from the
// computed style to the given content width.
//
// Takes contentWidth (float64) which is the current content
// width.
// Takes style (*ComputedStyle) which carries the max-width
// property.
// Takes containingBlockWidth (float64) which is the containing
// block width for percentage resolution.
// Takes box (*LayoutBox) which is used for intrinsic sizing
// measurement.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement for intrinsic sizing.
//
// Returns float64 which is the clamped content width.
func clampMaxWidth(contentWidth float64, style *ComputedStyle, containingBlockWidth float64, box *LayoutBox, fontMetrics FontMetricsPort) float64 {
	if style.MaxWidth.IsAuto() {
		return contentWidth
	}
	if style.MaxWidth.IsFitContent() {
		fitContent := resolveFitContentValue(style.MaxWidth, box, fontMetrics, containingBlockWidth, style)
		return math.Min(contentWidth, fitContent)
	}
	maxWidth := adjustForBoxSizing(style.MaxWidth.Resolve(containingBlockWidth, contentWidth), style, true)
	return math.Min(contentWidth, maxWidth)
}

// resolveFitContentValue computes the fit-content width for
// use in min-width/max-width constraints. The formula is
// min(max-content, max(min-content, clamp)) where clamp is
// the explicit argument or available width for bare fit-content.
//
// Children are measured directly rather than through the box,
// because measureBlockIntrinsicWidth short-circuits to the
// declared width when the box has an explicit width property.
//
// Takes dim (Dimension) which is the fit-content dimension
// to resolve.
// Takes box (*LayoutBox) which is the box whose children
// are measured.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement for intrinsic sizing.
// Takes availableWidth (float64) which is the width available
// from the containing block.
// Takes style (*ComputedStyle) which provides edge values for
// subtracting padding and border.
//
// Returns float64 which is the resolved fit-content width.
func resolveFitContentValue(dim Dimension, box *LayoutBox, fontMetrics FontMetricsPort, availableWidth float64, style *ComputedStyle) float64 {
	edgeSum := style.PaddingLeft + style.PaddingRight +
		style.BorderLeftWidth + style.BorderRightWidth
	minW := measureChildrenIntrinsicWidth(box, SizingModeMinContent, fontMetrics)
	maxW := measureChildrenIntrinsicWidth(box, SizingModeMaxContent, fontMetrics)
	clamp := availableWidth - edgeSum
	if dim.Unit == DimensionUnitFitContent {
		clamp = dim.Value
	}
	return math.Min(maxW, math.Max(minW, math.Max(0, clamp)))
}

// collapseMargins computes the collapsed margin from two
// adjacent margins per CSS margin collapsing rules.
//
// Takes marginA (float64) which is the first adjacent
// margin value.
// Takes marginB (float64) which is the second adjacent
// margin value.
//
// Returns float64 which is the collapsed margin.
func collapseMargins(marginA, marginB float64) float64 {
	if marginA >= 0 && marginB >= 0 {
		return math.Max(marginA, marginB)
	}
	if marginA < 0 && marginB < 0 {
		return math.Min(marginA, marginB)
	}
	return marginA + marginB
}

// computeRelativeOffset stores the relative positioning
// offset on the box without applying it. The offset is
// applied later by applyAllRelativeOffsets.
//
// Takes box (*LayoutBox) which is the box whose
// relative offset is computed.
func computeRelativeOffset(box *LayoutBox) {
	if box.Style.Position != PositionRelative {
		return
	}

	if !box.Style.Left.IsAuto() {
		box.OffsetX = box.Style.Left.Resolve(box.ContentWidth, 0)
	} else if !box.Style.Right.IsAuto() {
		box.OffsetX = -box.Style.Right.Resolve(box.ContentWidth, 0)
	}

	if !box.Style.Top.IsAuto() {
		box.OffsetY = box.Style.Top.Resolve(box.ContentHeight, 0)
	} else if !box.Style.Bottom.IsAuto() {
		box.OffsetY = -box.Style.Bottom.Resolve(box.ContentHeight, 0)
	}
}

// applyAllRelativeOffsets walks the tree, computes relative
// offsets for each positioned box, and applies them by
// shifting the box and all its descendants.
//
// Takes box (*LayoutBox) which is the root of the tree
// to walk.
func applyAllRelativeOffsets(box *LayoutBox) {
	computeRelativeOffset(box)
	if box.OffsetX != 0 || box.OffsetY != 0 {
		applyOffsetRecursive(box, box.OffsetX, box.OffsetY)
	}
	for _, child := range box.Children {
		applyAllRelativeOffsets(child)
	}
}

// applyOffsetRecursive translates a box and all its children
// by the given x and y offsets.
//
// Takes box (*LayoutBox) which is the box to translate.
// Takes offsetX (float64) which is the horizontal
// translation in points.
// Takes offsetY (float64) which is the vertical
// translation in points.
func applyOffsetRecursive(box *LayoutBox, offsetX, offsetY float64) {
	box.ContentX += offsetX
	box.ContentY += offsetY
	for _, child := range box.Children {
		applyOffsetRecursive(child, offsetX, offsetY)
	}
}

// layoutOutsideListMarker finds and positions an outside
// list marker to the left of the list item content area.
//
// Takes listItem (*LayoutBox) which is the list item box.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement for sizing the marker.
func layoutOutsideListMarker(listItem *LayoutBox, fontMetrics FontMetricsPort) {
	for _, child := range listItem.Children {
		if child.Type != BoxListMarker {
			continue
		}

		font := FontDescriptor{
			Family: child.Style.FontFamily,
			Weight: child.Style.FontWeight,
			Style:  child.Style.FontStyle,
		}
		fontSize := child.Style.FontSize
		metrics := fontMetrics.GetMetrics(font, fontSize)

		markerWidth := fontMetrics.MeasureText(font, fontSize, child.Text, child.Style.Direction)
		child.Glyphs = fontMetrics.ShapeText(font, fontSize, child.Text, child.Style.Direction)
		fontLineHeight := metrics.Ascent + metrics.Descent + metrics.LineGap
		markerHeight := child.Style.LineHeight
		if markerHeight < fontLineHeight {
			markerHeight = fontLineHeight
		}

		child.ContentWidth = markerWidth
		child.ContentHeight = markerHeight
		child.ContentX = listItem.ContentX + listItem.Padding.Left + listItem.Border.Left - markerWidth
		child.ContentY = listItem.ContentY + listItem.Padding.Top + listItem.Border.Top
		return
	}
}

// hasOnlyInlineChildren reports whether all children of the
// box are inline-level.
//
// Takes box (*LayoutBox) which is the box to inspect.
//
// Returns bool which is true when every child of the
// box is inline-level.
func hasOnlyInlineChildren(box *LayoutBox) bool {
	if len(box.Children) == 0 {
		return false
	}
	for _, child := range box.Children {
		if child.Type == BoxListMarker {
			continue
		}
		if child.Type.IsInlineLevel() {
			continue
		}

		if child.Type == BoxReplaced && child.Style.Display == DisplayInline {
			continue
		}
		return false
	}
	return true
}
