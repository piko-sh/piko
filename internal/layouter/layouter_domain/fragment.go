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

// Fragment represents the layout result for a single box. It stores
// positions as parent-relative offsets rather than absolute
// coordinates, enabling future caching and parallelism.
type Fragment struct {
	// Box is a back-reference to the input LayoutBox that
	// produced this fragment.
	Box *LayoutBox

	// Children are the child fragments in document order.
	Children []*Fragment

	// OffsetX is the X position of the content box relative
	// to the parent fragment's content box origin.
	OffsetX float64

	// OffsetY is the Y position of the content box relative
	// to the parent fragment's content box origin.
	OffsetY float64

	// ContentWidth is the width of the content area in
	// points.
	ContentWidth float64

	// ContentHeight is the height of the content area in
	// points.
	ContentHeight float64

	// Padding holds the resolved padding values in points.
	Padding BoxEdges

	// Border holds the resolved border widths in points.
	Border BoxEdges

	// Margin holds the resolved margin values in points.
	Margin BoxEdges
}

// BorderBoxWidth returns the total width including padding,
// borders, and content.
//
// Returns the border box width in points.
func (f *Fragment) BorderBoxWidth() float64 {
	return f.ContentWidth + f.Padding.Horizontal() + f.Border.Horizontal()
}

// BorderBoxHeight returns the total height including padding,
// borders, and content.
//
// Returns the border box height in points.
func (f *Fragment) BorderBoxHeight() float64 {
	return f.ContentHeight + f.Padding.Vertical() + f.Border.Vertical()
}

// MarginBoxWidth returns the total width including margins,
// padding, borders, and content.
//
// Returns the margin box width in points.
func (f *Fragment) MarginBoxWidth() float64 {
	return f.BorderBoxWidth() + f.Margin.Horizontal()
}

// MarginBoxHeight returns the total height including margins,
// padding, borders, and content.
//
// Returns the margin box height in points.
func (f *Fragment) MarginBoxHeight() float64 {
	return f.BorderBoxHeight() + f.Margin.Vertical()
}

// InlineSize returns the content size in the inline axis
// for the given writing mode.
//
// Takes writingMode (WritingModeType) which selects the
// writing direction.
//
// Returns float64 which is the inline-axis content size.
func (f *Fragment) InlineSize(writingMode WritingModeType) float64 {
	if writingMode == WritingModeHorizontalTB {
		return f.ContentWidth
	}
	return f.ContentHeight
}

// BlockSize returns the content size in the block axis
// for the given writing mode.
//
// Takes writingMode (WritingModeType) which selects the
// writing direction.
//
// Returns float64 which is the block-axis content size.
func (f *Fragment) BlockSize(writingMode WritingModeType) float64 {
	if writingMode == WritingModeHorizontalTB {
		return f.ContentHeight
	}
	return f.ContentWidth
}

// InlineOffset returns the inline-axis offset for the given
// writing mode.
//
// Takes writingMode (WritingModeType) which selects the
// writing direction.
//
// Returns float64 which is the inline-axis offset.
func (f *Fragment) InlineOffset(writingMode WritingModeType) float64 {
	if writingMode == WritingModeHorizontalTB {
		return f.OffsetX
	}
	return f.OffsetY
}

// BlockOffset returns the block-axis offset for the given
// writing mode.
//
// Takes writingMode (WritingModeType) which selects the
// writing direction.
//
// Returns float64 which is the block-axis offset.
func (f *Fragment) BlockOffset(writingMode WritingModeType) float64 {
	if writingMode == WritingModeHorizontalTB {
		return f.OffsetY
	}
	return f.OffsetX
}

// writeFragmentsToBoxTree recursively writes Fragment
// results back to the corresponding LayoutBox tree,
// converting parent-relative offsets to absolute
// coordinates.
//
// Takes fragment (*Fragment) which is the fragment tree
// to write back.
// Takes parentContentX (float64) which is the absolute X
// of the parent's content box origin.
// Takes parentContentY (float64) which is the absolute Y
// of the parent's content box origin.
func writeFragmentsToBoxTree(fragment *Fragment, parentContentX, parentContentY float64) {
	contentX := parentContentX + fragment.OffsetX
	contentY := parentContentY + fragment.OffsetY
	if fragment.Box != nil {
		box := fragment.Box
		box.ContentX = contentX
		box.ContentY = contentY
		box.ContentWidth = fragment.ContentWidth
		box.ContentHeight = fragment.ContentHeight
		box.Padding = fragment.Padding
		box.Border = fragment.Border
		box.Margin = fragment.Margin
	}
	for _, child := range fragment.Children {
		writeFragmentsToBoxTree(child, contentX, contentY)
	}
}
