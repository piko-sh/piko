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

// gridAlignmentInput groups the cell geometry and content offsets
// needed by alignGridItem, reducing its argument count.
type gridAlignmentInput struct {
	// cellX holds the horizontal position of the cell in points.
	cellX float64

	// cellY holds the vertical position of the cell in points.
	cellY float64

	// cellWidth holds the width of the grid cell in points.
	cellWidth float64

	// cellHeight holds the height of the grid cell in points.
	cellHeight float64

	// contentOffsetX holds the horizontal offset from the
	// grid container edge to the content area.
	contentOffsetX float64

	// contentOffsetY holds the vertical offset from the
	// grid container edge to the content area.
	contentOffsetY float64
}

// alignGridItem positions a grid item fragment within its cell
// by applying justify-items and align-items alignment.
//
// Takes fragment (*Fragment) which is the grid item fragment
// to position.
// Takes cell (gridAlignmentInput) which holds the cell
// geometry and content offsets.
// Takes containerStyle (*ComputedStyle) which provides the
// container's justify-items and align-items values.
func alignGridItem(
	fragment *Fragment,
	cell gridAlignmentInput,
	containerStyle *ComputedStyle,
) {
	justifyItems := resolveEffectiveJustify(fragment, containerStyle.JustifyItems)
	alignItems := resolveEffectiveAlign(fragment, containerStyle.AlignItems)

	itemMarginWidth := applyGridJustifyStretch(fragment, justifyItems, cell.cellWidth)
	offsetX := computeGridJustifyOffset(justifyItems, cell.cellWidth, itemMarginWidth)

	itemMarginHeight := applyGridAlignStretch(fragment, alignItems, cell.cellHeight)

	if fragment.Box.Style.AspectRatio > 0 {
		arWidth := fragment.ContentHeight * fragment.Box.Style.AspectRatio
		if arWidth < fragment.ContentWidth {
			fragment.ContentWidth = arWidth
		}
	}

	offsetY := computeGridAlignOffset(alignItems, cell.cellHeight, itemMarginHeight)

	fragment.OffsetX = cell.contentOffsetX + cell.cellX + offsetX + fragment.Margin.Left + fragment.Padding.Left + fragment.Border.Left
	fragment.OffsetY = cell.contentOffsetY + cell.cellY + offsetY + fragment.Margin.Top + fragment.Padding.Top + fragment.Border.Top
}

// applyGridJustifyStretch stretches a grid item's content width to
// fill the cell when justify-items is stretch.
//
// Takes fragment (*Fragment) which is the grid item to stretch.
// Takes justify (JustifyItemsType) which is the effective
// justify-items value.
// Takes cellWidth (float64) which is the width of the grid cell.
//
// Returns float64 which is the updated margin-box width.
func applyGridJustifyStretch(fragment *Fragment, justify JustifyItemsType, cellWidth float64) float64 {
	if justify == JustifyItemsStretch {
		stretchWidth := cellWidth -
			fragment.Padding.Horizontal() -
			fragment.Border.Horizontal() -
			fragment.Margin.Horizontal()
		stretchWidth = clampStretchDimension(stretchWidth, &fragment.Box.Style, true)
		if stretchWidth > fragment.ContentWidth {
			fragment.ContentWidth = stretchWidth
		}
	}
	return fragment.MarginBoxWidth()
}

// computeGridJustifyOffset returns the horizontal offset within a
// grid cell based on the effective justify-items value.
//
// Takes justify (JustifyItemsType) which is the effective
// justify-items value.
// Takes cellWidth (float64) which is the width of the grid cell.
// Takes itemMarginWidth (float64) which is the margin-box width
// of the grid item.
//
// Returns float64 which is the horizontal offset.
func computeGridJustifyOffset(justify JustifyItemsType, cellWidth, itemMarginWidth float64) float64 {
	switch justify {
	case JustifyItemsCentre:
		return (cellWidth - itemMarginWidth) / 2
	case JustifyItemsEnd:
		return cellWidth - itemMarginWidth
	default:
		return 0
	}
}

// applyGridAlignStretch stretches a grid item's content height to
// fill the cell when align-items is stretch.
//
// Takes fragment (*Fragment) which is the grid item to stretch.
// Takes align (AlignItemsType) which is the effective
// align-items value.
// Takes cellHeight (float64) which is the height of the grid
// cell.
//
// Returns float64 which is the updated margin-box height.
func applyGridAlignStretch(fragment *Fragment, align AlignItemsType, cellHeight float64) float64 {
	if align == AlignItemsStretch {
		stretchHeight := cellHeight -
			fragment.Padding.Vertical() -
			fragment.Border.Vertical() -
			fragment.Margin.Vertical()
		stretchHeight = clampStretchDimension(stretchHeight, &fragment.Box.Style, false)
		if stretchHeight > fragment.ContentHeight {
			fragment.ContentHeight = stretchHeight
		}
	}
	return fragment.MarginBoxHeight()
}

// computeGridAlignOffset returns the vertical offset within a grid
// cell based on the effective align-items value.
//
// Takes align (AlignItemsType) which is the effective
// align-items value.
// Takes cellHeight (float64) which is the height of the grid
// cell.
// Takes itemMarginHeight (float64) which is the margin-box
// height of the grid item.
//
// Returns float64 which is the vertical offset.
func computeGridAlignOffset(align AlignItemsType, cellHeight, itemMarginHeight float64) float64 {
	switch align {
	case AlignItemsCentre:
		return (cellHeight - itemMarginHeight) / 2
	case AlignItemsFlexEnd:
		return cellHeight - itemMarginHeight
	default:
		return 0
	}
}

// resolveEffectiveJustify resolves the effective justify-items value
// for a grid item, falling back to the container's justify-items
// when the item's justify-self is auto.
//
// Takes fragment (*Fragment) which is the grid item fragment.
// Takes containerJustify (JustifyItemsType) which is the
// container's justify-items value.
//
// Returns JustifyItemsType which is the effective value.
func resolveEffectiveJustify(fragment *Fragment, containerJustify JustifyItemsType) JustifyItemsType {
	if fragment.Box == nil || fragment.Box.Style.JustifySelf == JustifySelfAuto {
		return containerJustify
	}
	switch fragment.Box.Style.JustifySelf {
	case JustifySelfStart:
		return JustifyItemsStart
	case JustifySelfEnd:
		return JustifyItemsEnd
	case JustifySelfCentre:
		return JustifyItemsCentre
	case JustifySelfStretch:
		return JustifyItemsStretch
	default:
		return containerJustify
	}
}

// resolveEffectiveAlign resolves the effective align-items value
// for a grid item, falling back to the container's align-items
// when the item's align-self is auto.
//
// Takes fragment (*Fragment) which is the grid item fragment.
// Takes containerAlign (AlignItemsType) which is the
// container's align-items value.
//
// Returns AlignItemsType which is the effective value.
func resolveEffectiveAlign(fragment *Fragment, containerAlign AlignItemsType) AlignItemsType {
	if fragment.Box == nil || fragment.Box.Style.AlignSelf == AlignSelfAuto {
		return containerAlign
	}
	switch fragment.Box.Style.AlignSelf {
	case AlignSelfFlexStart:
		return AlignItemsFlexStart
	case AlignSelfFlexEnd:
		return AlignItemsFlexEnd
	case AlignSelfCentre:
		return AlignItemsCentre
	case AlignSelfStretch:
		return AlignItemsStretch
	default:
		return containerAlign
	}
}
