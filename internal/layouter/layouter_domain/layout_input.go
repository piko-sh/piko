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

// SizingMode controls how layout algorithms determine
// the box's width. Normal mode uses the available width
// from the parent; MinContent and MaxContent modes
// measure intrinsic sizes.
type SizingMode int

const (
	// SizingModeNormal resolves widths using the
	// available width from the containing block.
	SizingModeNormal SizingMode = iota

	// SizingModeMinContent resolves to the narrowest
	// width that avoids overflow.
	SizingModeMinContent

	// SizingModeMaxContent resolves to the width the
	// content would take with no line breaks.
	SizingModeMaxContent
)

var sizingModeNames = [...]string{
	SizingModeNormal:     "normal",
	SizingModeMinContent: "min-content",
	SizingModeMaxContent: "max-content",
}

// String returns the CSS keyword for this sizing mode.
//
// Returns string which is the CSS keyword.
func (s SizingMode) String() string {
	if int(s) < len(sizingModeNames) {
		return sizingModeNames[s]
	}
	return cssKeywordUnknown
}

// layoutInput carries the constraints and context passed from
// a parent formatting context to a child layout algorithm.
// This struct evolves into a full constraint space as layout
// algorithms are progressively enriched.
type layoutInput struct {
	// FontMetrics provides text measurement and font
	// metric queries.
	FontMetrics FontMetricsPort

	// Cache stores previously computed layout results
	// for reuse within a single LayoutBoxTree call.
	// Nil disables caching.
	Cache *layoutCache

	// Floats provides access to the parent block formatting
	// context's float state, allowing inline content to
	// shorten line boxes around floats. Nil when no floats
	// are active.
	Floats *floatContext

	// Edges carries the resolved padding, border, and
	// vertical margin values for the current box.
	Edges resolvedEdges

	// AvailableWidth is the inline-axis space available
	// from the containing block, in points.
	AvailableWidth float64

	// AvailableBlockSize is the block-axis space available
	// from the containing block, in points. Zero means
	// indefinite (the default).
	AvailableBlockSize float64

	// PercentageResolution is the basis for resolving
	// percentage widths and heights. Zero means fall back
	// to AvailableWidth.
	PercentageResolution float64

	// BFCOffset is the offset from the block formatting
	// context root, used for accurate float placement.
	// Zero is the default.
	BFCOffset float64

	// MarginStrut is the pending collapsed margin carried
	// from the parent, in points. Zero is the default.
	MarginStrut float64

	// FragmentainerBlockSize is the block-axis size of
	// the current fragmentainer (page or column), in
	// points. Zero means no fragmentainer is active.
	FragmentainerBlockSize float64

	// FragmentainerOffset is how far into the current
	// fragmentainer layout has progressed, in points.
	// Used to determine remaining space before a break.
	FragmentainerOffset float64

	// FloatBFCOffsetY is the Y offset from the BFC root
	// to this box's content top, used to translate local
	// Y coordinates into float coordinate space.
	FloatBFCOffsetY float64

	// FloatContainerX is the X coordinate of the BFC
	// content area in float coordinate space.
	FloatContainerX float64

	// FloatContainerWidth is the width of the BFC content
	// area in float coordinate space.
	FloatContainerWidth float64

	// SizingMode controls how width is determined.
	// Zero value (SizingModeNormal) preserves current
	// behaviour.
	SizingMode SizingMode

	// ContainingBlockDirection is the direction property of
	// the containing block, used to determine which margin
	// absorbs remaining space for over-constrained blocks.
	ContainingBlockDirection DirectionType

	// IsNewBFC indicates whether this box establishes a
	// new block formatting context. False is the default.
	IsNewBFC bool

	// IsFixedInlineSize indicates that the parent has
	// already determined this box's inline size. False
	// is the default.
	IsFixedInlineSize bool
}
