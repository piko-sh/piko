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
)

// layoutPositionedElements applies CSS absolute and fixed
// positioning as a post-layout pass over the tree rooted at root.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signal through recursive layout.
// Takes root (*LayoutBox) which is the root of the layout tree.
// Takes input (layoutInput) which carries font metrics and cache
// from the parent context.
func layoutPositionedElements(ctx context.Context, root *LayoutBox, input layoutInput) {
	layoutPositionedSubtree(ctx, root, root, input)
}

// layoutPositionedSubtree recursively resolves positioned children
// within the given subtree.
//
// Takes box (*LayoutBox) which is the current subtree root.
// Takes root (*LayoutBox) which is the overall layout root used
// as the containing block for fixed-position elements.
// Takes input (layoutInput) which carries font metrics and cache
// from the parent context.
func layoutPositionedSubtree(ctx context.Context, box, root *LayoutBox, input layoutInput) {
	for _, child := range box.Children {
		switch child.Style.Position {
		case PositionAbsolute:
			containingBlock := child.ContainingBlock
			if containingBlock == nil {
				containingBlock = root
			}
			resolvePositionedBox(ctx, child, containingBlock, input)
		case PositionFixed:
			containingBlock := root
			if child.TransformAncestor != nil {
				containingBlock = child.TransformAncestor
			}
			resolvePositionedBox(ctx, child, containingBlock, input)
		}

		layoutPositionedSubtree(ctx, child, root, input)
	}
}

// dimensionFromOffsetsInput groups the parameters for resolving a
// content dimension from explicit size or opposing offsets.
type dimensionFromOffsetsInput struct {
	// style holds the element's computed style for box-sizing adjustment.
	style *ComputedStyle

	// explicitSize holds the explicit width or height dimension.
	explicitSize Dimension

	// startOffset holds the start-side offset (left or top).
	startOffset Dimension

	// endOffset holds the end-side offset (right or bottom).
	endOffset Dimension

	// current holds the fallback content size when neither explicit size
	// nor opposing offsets are set.
	current float64

	// containingSize holds the containing block size along this axis.
	containingSize float64

	// edgesSum holds the total padding, border, and margin along this axis.
	edgesSum float64

	// horizontal indicates whether this is the horizontal axis.
	horizontal bool
}

// resolveDimensionFromOffsets computes the content size along a
// single axis from explicit size or opposing offsets.
//
// When an explicit size is set, it is resolved and adjusted for
// box-sizing. When both start and end offsets are set, the
// content size is derived from the containing size minus offsets
// and edges. Otherwise the fallback current value is returned.
//
// Takes input (dimensionFromOffsetsInput) which holds all
// parameters for the dimension resolution.
//
// Returns float64 which is the resolved content size.
func resolveDimensionFromOffsets(input dimensionFromOffsetsInput) float64 {
	if !input.explicitSize.IsAuto() {
		return adjustForBoxSizing(
			input.explicitSize.Resolve(input.containingSize, input.containingSize),
			input.style, input.horizontal,
		)
	}
	if !input.startOffset.IsAuto() && !input.endOffset.IsAuto() {
		resolved := input.containingSize -
			input.startOffset.Resolve(input.containingSize, 0) -
			input.endOffset.Resolve(input.containingSize, 0) -
			input.edgesSum
		if resolved < 0 {
			return 0
		}
		return resolved
	}
	return input.current
}

// resolvePositionFromOffset computes the content origin along a
// single axis from the start offset, end offset, or the default
// start-aligned position.
//
// Takes startOffset (Dimension) which is the start-side offset
// (left or top).
// Takes endOffset (Dimension) which is the end-side offset
// (right or bottom).
// Takes containingOrigin (float64) which is the containing
// block origin along this axis.
// Takes containingSize (float64) which is the containing
// block size along this axis.
// Takes contentSize (float64) which is the resolved content
// size of the element.
// Takes startEdges (float64) which is the total margin,
// padding, and border on the start side.
// Takes endEdges (float64) which is the total margin, padding,
// and border on the end side.
//
// Returns float64 which is the resolved content origin.
func resolvePositionFromOffset(
	startOffset Dimension,
	endOffset Dimension,
	containingOrigin float64,
	containingSize float64,
	contentSize float64,
	startEdges float64,
	endEdges float64,
) float64 {
	if !startOffset.IsAuto() {
		return containingOrigin + startOffset.Resolve(containingSize, 0) + startEdges
	}
	if !endOffset.IsAuto() {
		return containingOrigin + containingSize -
			endOffset.Resolve(containingSize, 0) - endEdges - contentSize
	}
	return containingOrigin + startEdges
}

// positionedBoxContext holds the containing block geometry and
// resolved edges needed during positioned box resolution.
type positionedBoxContext struct {
	// paddingBoxX holds the X origin of the containing block's padding box.
	paddingBoxX float64

	// paddingBoxY holds the Y origin of the containing block's padding box.
	paddingBoxY float64

	// containingWidth holds the containing block's padding-box width.
	containingWidth float64

	// containingHeight holds the containing block's padding-box height.
	containingHeight float64

	// edges holds the resolved padding and border edges for the positioned element.
	edges resolvedEdges

	// margin holds the resolved margin edges for the positioned element.
	margin BoxEdges
}

// resolvePositionedBox resolves the size and position of an
// absolutely or fixed-positioned element against its containing
// block.
//
// Takes box (*LayoutBox) which is the positioned element to
// resolve.
// Takes containingBlock (*LayoutBox) which is the reference
// block for offset resolution.
// Takes input (layoutInput) which carries font metrics and cache
// from the parent context.
func resolvePositionedBox(ctx context.Context, box *LayoutBox, containingBlock *LayoutBox, input layoutInput) {
	positionedContext := buildPositionedBoxContext(box, containingBlock)

	contentWidth := resolvePositionedHorizontal(box, &positionedContext)
	contentHeight := resolvePositionedVertical(box, &positionedContext)

	positionedInput := layoutInput{
		AvailableWidth: contentWidth,
		FontMetrics:    input.FontMetrics,
		Cache:          input.Cache,
		Edges:          positionedContext.edges,
	}

	childResult, childHeight := layoutPositionedChildren(ctx, box, positionedInput, &positionedContext)
	if childHeight >= 0 {
		contentHeight = childHeight
	}

	contentHeight = resolveDimensionFromOffsets(dimensionFromOffsetsInput{
		current:        contentHeight,
		explicitSize:   box.Style.Height,
		startOffset:    box.Style.Top,
		endOffset:      box.Style.Bottom,
		containingSize: positionedContext.containingHeight,
		edgesSum:       positionedContext.edges.Padding.Vertical() + positionedContext.edges.Border.Vertical() + positionedContext.margin.Top + positionedContext.margin.Bottom,
		style:          &box.Style,
		horizontal:     false,
	})

	buildPositionedFragment(box, containingBlock, &positionedContext, childResult, contentWidth, contentHeight)
}

// layoutPositionedChildren lays out the children of a positioned box
// using either inline or block formatting.
//
// Takes box (*LayoutBox) which is the positioned element.
// Takes input (layoutInput) which carries available width and font metrics.
// Takes positionedContext (*positionedBoxContext) which holds the
// containing block geometry.
//
// Returns *formattingContextResult which holds the child layout result,
// or nil when the box has no children.
// Returns float64 which is the content height from children, or -1
// when the box has no children.
func layoutPositionedChildren(
	ctx context.Context,
	box *LayoutBox,
	input layoutInput,
	positionedContext *positionedBoxContext,
) (*formattingContextResult, float64) {
	if hasOnlyInlineChildren(box) {
		result := layoutInlineContent(ctx, box, input)
		return &result, result.ContentHeight
	}
	if len(box.Children) > 0 {
		result := layoutBlockChildren(ctx, box, input)
		positionedContext.margin.Top = result.Margin.Top
		positionedContext.margin.Bottom = result.Margin.Bottom
		return &result, result.ContentHeight
	}
	return nil, -1
}

// buildPositionedFragment constructs the final fragment for a
// positioned box and writes it into the box tree.
//
// Takes box (*LayoutBox) which is the positioned element.
// Takes containingBlock (*LayoutBox) which is the reference block.
// Takes ctx (*positionedBoxContext) which holds the containing block geometry.
// Takes childResult (*formattingContextResult) which holds the child
// layout output.
// Takes contentWidth (float64) which is the resolved content width.
// Takes contentHeight (float64) which is the resolved content height.
func buildPositionedFragment(
	box *LayoutBox,
	containingBlock *LayoutBox,
	ctx *positionedBoxContext,
	childResult *formattingContextResult,
	contentWidth, contentHeight float64,
) {
	contentX := resolvePositionFromOffset(
		box.Style.Left, box.Style.Right,
		ctx.paddingBoxX, ctx.containingWidth, contentWidth,
		ctx.margin.Left+ctx.edges.Padding.Left+ctx.edges.Border.Left,
		ctx.margin.Right+ctx.edges.Padding.Right+ctx.edges.Border.Right,
	)

	contentY := resolvePositionFromOffset(
		box.Style.Top, box.Style.Bottom,
		ctx.paddingBoxY, ctx.containingHeight, contentHeight,
		ctx.margin.Top+ctx.edges.Padding.Top+ctx.edges.Border.Top,
		ctx.margin.Bottom+ctx.edges.Padding.Bottom+ctx.edges.Border.Bottom,
	)

	var children []*Fragment
	if childResult != nil {
		children = childResult.Children
	}

	fragment := &Fragment{
		Box:           box,
		Children:      children,
		OffsetX:       contentX - containingBlock.ContentX,
		OffsetY:       contentY - containingBlock.ContentY,
		ContentWidth:  contentWidth,
		ContentHeight: contentHeight,
		Padding:       ctx.edges.Padding,
		Border:        ctx.edges.Border,
		Margin:        ctx.margin,
	}
	writeFragmentsToBoxTree(fragment, containingBlock.ContentX, containingBlock.ContentY)
}

// buildPositionedBoxContext computes the containing block geometry,
// resolved edges, and margins for a positioned box.
//
// Takes box (*LayoutBox) which is the positioned element.
// Takes containingBlock (*LayoutBox) which is the reference block.
//
// Returns positionedBoxContext which holds the computed geometry and edges.
func buildPositionedBoxContext(box, containingBlock *LayoutBox) positionedBoxContext {
	containingWidth := containingBlock.ContentWidth + containingBlock.Padding.Horizontal()
	edges := resolveEdgesFromStyle(&box.Style, containingWidth)

	return positionedBoxContext{
		paddingBoxX:      containingBlock.ContentX - containingBlock.Padding.Left,
		paddingBoxY:      containingBlock.ContentY - containingBlock.Padding.Top,
		containingWidth:  containingWidth,
		containingHeight: containingBlock.ContentHeight + containingBlock.Padding.Vertical(),
		edges:            edges,
		margin: BoxEdges{
			Top:    edges.MarginTop,
			Right:  box.Style.MarginRight.Resolve(containingWidth, 0),
			Bottom: edges.MarginBottom,
			Left:   box.Style.MarginLeft.Resolve(containingWidth, 0),
		},
	}
}

// resolvePositionedHorizontal resolves the content width of a
// positioned box from explicit width or opposing left/right offsets.
//
// Takes box (*LayoutBox) which is the positioned element.
// Takes ctx (*positionedBoxContext) which holds the containing block geometry.
//
// Returns float64 which is the resolved content width.
func resolvePositionedHorizontal(box *LayoutBox, ctx *positionedBoxContext) float64 {
	return resolveDimensionFromOffsets(dimensionFromOffsetsInput{
		current:        box.ContentWidth,
		explicitSize:   box.Style.Width,
		startOffset:    box.Style.Left,
		endOffset:      box.Style.Right,
		containingSize: ctx.containingWidth,
		edgesSum:       ctx.edges.Padding.Horizontal() + ctx.edges.Border.Horizontal() + ctx.margin.Left + ctx.margin.Right,
		style:          &box.Style,
		horizontal:     true,
	})
}

// resolvePositionedVertical resolves the content height of a
// positioned box from explicit height or opposing top/bottom offsets.
//
// Takes box (*LayoutBox) which is the positioned element.
// Takes ctx (*positionedBoxContext) which holds the containing block geometry.
//
// Returns float64 which is the resolved content height.
func resolvePositionedVertical(box *LayoutBox, ctx *positionedBoxContext) float64 {
	return resolveDimensionFromOffsets(dimensionFromOffsetsInput{
		current:        box.ContentHeight,
		explicitSize:   box.Style.Height,
		startOffset:    box.Style.Top,
		endOffset:      box.Style.Bottom,
		containingSize: ctx.containingHeight,
		edgesSum:       ctx.edges.Padding.Vertical() + ctx.edges.Border.Vertical() + ctx.margin.Top + ctx.margin.Bottom,
		style:          &box.Style,
		horizontal:     false,
	})
}
