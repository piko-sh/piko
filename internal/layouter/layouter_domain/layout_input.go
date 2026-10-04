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

// SizingMode controls how layout algorithms determine the box's width. Normal mode uses
// the available width from the parent; MinContent and MaxContent modes measure intrinsic
// sizes.
type SizingMode int

const (
	// SizingModeNormal resolves widths using the available width from the containing block.
	SizingModeNormal SizingMode = iota

	// SizingModeMinContent resolves to the narrowest width that avoids overflow.
	SizingModeMinContent

	// SizingModeMaxContent resolves to the width the content would take with no line breaks.
	SizingModeMaxContent
)

var (
	// sizingModeNames maps SizingMode values to their CSS keyword strings.
	sizingModeNames = [...]string{
		SizingModeNormal:     "normal",
		SizingModeMinContent: "min-content",
		SizingModeMaxContent: "max-content",
	}
)

// String returns the CSS keyword for this sizing mode.
//
// Returns string which is the CSS keyword.
func (s SizingMode) String() string {
	if int(s) < len(sizingModeNames) {
		return sizingModeNames[s]
	}
	return cssKeywordUnknown
}

// layoutInput carries the constraints and context passed from a parent formatting context
// to a child layout algorithm. Evolves into a full constraint space as layout algorithms
// are progressively enriched.
type layoutInput struct {
	// FontMetrics provides text measurement and font metric queries.
	FontMetrics FontMetricsPort

	// Cache stores previously computed layout results for reuse within a single
	// LayoutBoxTree call. Nil disables caching.
	Cache *layoutCache

	// Limits enforces the layout limits and records the first breach. Nil applies the
	// default limits without recording breaches.
	Limits *LimitTracker

	// Floats provides access to the parent block formatting context's float state, allowing
	// inline content to shorten line boxes around floats. Nil when no floats are active.
	Floats *floatContext

	// Edges carries the resolved padding, border, and vertical margin values for the current
	// box.
	Edges resolvedEdges

	// AvailableWidth is the inline-axis space available from the containing block, in
	// points.
	AvailableWidth float64

	// AvailableBlockSize is the block-axis space available from the containing block, in
	// points. Zero means indefinite (the default).
	AvailableBlockSize float64

	// PercentageResolution is the basis for resolving percentage widths and heights. Zero
	// means fall back to AvailableWidth.
	PercentageResolution float64

	// BFCOffset is the offset from the block formatting context root, used for accurate
	// float placement. Zero is the default.
	BFCOffset float64

	// MarginStrut is the pending collapsed margin carried from the parent, in points. Zero
	// is the default.
	MarginStrut float64

	// FragmentainerBlockSize is the block-axis size of the current fragmentainer (page or
	// column), in points. Zero means no fragmentainer is active.
	FragmentainerBlockSize float64

	// FragmentainerOffset is how far into the current fragmentainer layout has progressed,
	// in points. Used to determine remaining space before a break.
	FragmentainerOffset float64

	// FloatBFCOffsetY is the Y offset from the BFC root to this box's content top, used to
	// translate local Y coordinates into float coordinate space.
	FloatBFCOffsetY float64

	// FloatContainerX is the X coordinate of the BFC content area in float coordinate space.
	FloatContainerX float64

	// FloatContainerWidth is the width of the BFC content area in float coordinate space.
	FloatContainerWidth float64

	// SizingMode controls how width is determined. Zero value (SizingModeNormal) preserves
	// current behaviour.
	SizingMode SizingMode

	// ContainingBlockDirection is the direction property of the containing block, used to
	// determine which margin absorbs remaining space for over-constrained blocks.
	ContainingBlockDirection DirectionType

	// IsNewBFC indicates whether this box establishes a new block formatting context. False
	// is the default.
	IsNewBFC bool

	// IsFixedInlineSize indicates that the parent has already determined this box's inline
	// size. False is the default.
	IsFixedInlineSize bool
}

// newRootLayoutInput creates the layoutInput for the root of a layout, carrying the font
// metrics, layout cache and limit tracker that every descendant input inherits. All other
// constraints start at zero, giving no edges, floats, fragmentainer, or fixed inline
// size, and using normal sizing mode.
//
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
// Takes cache (*layoutCache) which stores layout results for reuse, or nil to disable
// caching.
// Takes limits (*LimitTracker) which enforces the layout limits, or nil for defaults.
// Takes availableWidth (float64) which is the inline-axis space in points.
// Takes availableBlockSize (float64) which is the block-axis space in points, or zero
// when indefinite.
//
// Returns layoutInput which is the constraint set for the root layout.
func newRootLayoutInput(
	fontMetrics FontMetricsPort,
	cache *layoutCache,
	limits *LimitTracker,
	availableWidth float64,
	availableBlockSize float64,
) layoutInput {
	return layoutInput{
		FontMetrics:              fontMetrics,
		Cache:                    cache,
		Limits:                   limits,
		Floats:                   nil,
		Edges:                    resolvedEdges{},
		AvailableWidth:           availableWidth,
		AvailableBlockSize:       availableBlockSize,
		PercentageResolution:     0,
		BFCOffset:                0,
		MarginStrut:              0,
		FragmentainerBlockSize:   0,
		FragmentainerOffset:      0,
		FloatBFCOffsetY:          0,
		FloatContainerX:          0,
		FloatContainerWidth:      0,
		SizingMode:               SizingModeNormal,
		ContainingBlockDirection: DirectionLTR,
		IsNewBFC:                 false,
		IsFixedInlineSize:        false,
	}
}

// newLayoutInput creates a layoutInput for a child layout, inheriting the parent's font
// metrics, layout cache and limit tracker with the given available sizes and resolved
// edges. All other constraints start at zero, giving no floats, fragmentainer, or fixed
// inline size, and using normal sizing mode.
//
// Takes parent (layoutInput) which supplies the inherited font metrics, cache and limit
// tracker.
// Takes availableWidth (float64) which is the inline-axis space in points.
// Takes availableBlockSize (float64) which is the block-axis space in points, or zero
// when indefinite.
// Takes edges (resolvedEdges) which holds the box's resolved padding, border and vertical
// margins.
//
// Returns layoutInput which is the constraint set for the child layout.
func newLayoutInput(
	parent layoutInput,
	availableWidth float64,
	availableBlockSize float64,
	edges resolvedEdges,
) layoutInput {
	input := newRootLayoutInput(parent.FontMetrics, parent.Cache, parent.Limits, availableWidth, availableBlockSize)
	input.Edges = edges
	return input
}

// newFixedInlineSizeInput creates a layoutInput for a child whose inline size its parent
// formatting context has already resolved, such as a flex item, grid item or table cell.
//
// Takes parent (layoutInput) which supplies the inherited font metrics, cache and limit
// tracker.
// Takes availableWidth (float64) which is the resolved inline size in points.
// Takes availableBlockSize (float64) which is the block-axis space in points, or zero
// when indefinite.
// Takes edges (resolvedEdges) which holds the child's resolved padding, border and
// vertical margins.
//
// Returns layoutInput which is the constraint set with IsFixedInlineSize set.
func newFixedInlineSizeInput(
	parent layoutInput,
	availableWidth float64,
	availableBlockSize float64,
	edges resolvedEdges,
) layoutInput {
	input := newLayoutInput(parent, availableWidth, availableBlockSize, edges)
	input.IsFixedInlineSize = true
	return input
}
