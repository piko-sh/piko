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
	"fmt"
)

// formattingContextResult holds the output of a formatting
// context layout pass. It separates the child fragments and
// intrinsic content height from the collapsed margin
// information, making the margin-collapsing protocol between
// block layout and layoutBox explicit rather than hidden
// inside Fragment.Margin.
type formattingContextResult struct {
	// Children are the child fragments in document order.
	Children []*Fragment

	// ContentHeight is the intrinsic content height
	// computed by the formatting context.
	ContentHeight float64

	// Margin holds the resolved margin values. For block
	// formatting contexts this carries the collapsed
	// parent-child margins; for other contexts it carries
	// the resolved vertical margins from the input edges.
	Margin BoxEdges

	// Border holds border values that override the box's own CSS border.
	//
	// Used by collapsed-border tables where the table's outer border
	// comes from adjacent cell borders rather than the table element's
	// own border property. Zero-valued means no override.
	Border BoxEdges

	// HasBorder indicates whether the Border field should be
	// used to override the box's CSS border.
	HasBorder bool
}

// FormattingContext encapsulates the layout algorithm for a
// particular formatting context (block, inline, flex, grid,
// table). The common steps - width resolution, height
// resolution, and min/max constraint application - are
// handled by the caller.
type FormattingContext interface {
	// Layout performs the formatting-context-specific layout
	// algorithm on the given box and returns a result with
	// child fragments, content height, and margin
	// information.
	Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult
}

// resolveFormattingContext selects the correct formatting
// context implementation for a box based on its type and
// children.
//
// Takes box (*LayoutBox) which is the box to inspect.
//
// Returns FormattingContext which is the layout algorithm
// for this box.
// Returns error when the box type has no known formatting
// context.
func resolveFormattingContext(box *LayoutBox) (FormattingContext, error) {
	switch {
	case box.Type == BoxFlex:
		return flexContext{}, nil
	case box.Type == BoxGrid:
		return gridContext{}, nil
	case box.Type == BoxTable:
		return tableContext{}, nil
	case isMultiColumnContainer(box):
		return multiColumnContext{}, nil
	case box.Style.Display == DisplayFlex || box.Style.Display == DisplayInlineFlex:
		return flexContext{}, nil
	case box.Style.Display == DisplayGrid || box.Style.Display == DisplayInlineGrid:
		return gridContext{}, nil
	case box.Style.Display == DisplayTable:
		return tableContext{}, nil
	case hasOnlyInlineChildren(box):
		return inlineContext{}, nil
	case box.Type.IsBlockLevel() || box.Type == BoxInlineBlock || box.Type == BoxReplaced:
		return blockContext{}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnhandledBoxType, box.Type)
	}
}

// isMultiColumnContainer reports whether a box establishes
// a multi-column formatting context.
//
// Takes box (*LayoutBox) which is the box to check.
//
// Returns bool which is true when the box establishes a
// multi-column formatting context.
func isMultiColumnContainer(box *LayoutBox) bool {
	return box.Style.ColumnCount > 1 ||
		(!box.Style.ColumnWidth.IsAuto() && box.Style.ColumnWidth.Value > 0)
}

// blockContext implements block formatting context layout.
type blockContext struct{}

// Layout lays out children using the block formatting
// context algorithm, returning child fragments with
// parent-relative offsets.
//
// Takes box (*LayoutBox) which is the block container to
// lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics.
//
// Returns formattingContextResult with the layout results.
func (blockContext) Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	return layoutBlockChildren(ctx, box, input)
}

// inlineContext implements inline formatting context layout.
type inlineContext struct{}

// Layout lays out children using the inline formatting
// context algorithm.
//
// Takes box (*LayoutBox) which is the inline container to
// lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics.
//
// Returns formattingContextResult with the layout results.
func (inlineContext) Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	return layoutInlineContent(ctx, box, input)
}

// flexContext implements flex formatting context layout.
type flexContext struct{}

// Layout lays out children using the flexbox algorithm.
//
// Takes box (*LayoutBox) which is the flex container to
// lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics.
//
// Returns formattingContextResult with the layout results.
func (flexContext) Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	return layoutFlexContainer(ctx, box, input)
}

// gridContext implements grid formatting context layout.
type gridContext struct{}

// Layout lays out children using the CSS grid algorithm.
//
// Takes box (*LayoutBox) which is the grid container to
// lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics.
//
// Returns formattingContextResult with the layout results.
func (gridContext) Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	return layoutGridContainer(ctx, box, input)
}

// multiColumnContext implements multi-column formatting
// context layout.
type multiColumnContext struct{}

// Layout lays out children using the multi-column algorithm.
//
// Takes box (*LayoutBox) which is the multi-column container
// to lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics.
//
// Returns formattingContextResult with the layout results.
func (multiColumnContext) Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	return layoutMultiColumnContainer(ctx, box, input)
}

// tableContext implements table formatting context layout.
type tableContext struct{}

// Layout lays out children using the table layout
// algorithm.
//
// Takes box (*LayoutBox) which is the table container to
// lay out.
// Takes input (layoutInput) which carries the available
// width and font metrics.
//
// Returns formattingContextResult with the layout results.
func (tableContext) Layout(ctx context.Context, box *LayoutBox, input layoutInput) formattingContextResult {
	return layoutTableContainer(ctx, box, input)
}
