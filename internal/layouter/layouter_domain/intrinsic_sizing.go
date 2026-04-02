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

// measureIntrinsicWidth returns the intrinsic width for the given
// box in the specified sizing mode. For min-content this is the
// narrowest width that avoids overflow; for max-content this is
// the width with no line breaks taken.
//
// Takes box (*LayoutBox) which is the box to measure.
// Takes mode (SizingMode) which selects min-content or
// max-content measurement.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement.
//
// Returns float64 which is the intrinsic width in points.
func measureIntrinsicWidth(box *LayoutBox, mode SizingMode, fontMetrics FontMetricsPort) float64 {
	switch box.Type {
	case BoxTextRun:
		return measureTextIntrinsicWidth(box, mode, fontMetrics)
	case BoxFlex:
		return measureFlexIntrinsicWidth(box, mode, fontMetrics)
	case BoxTable:
		return measureTableIntrinsicWidth(box, fontMetrics)
	default:

		switch box.Style.Display {
		case DisplayFlex, DisplayInlineFlex:
			return measureFlexIntrinsicWidth(box, mode, fontMetrics)
		case DisplayTable:
			return measureTableIntrinsicWidth(box, fontMetrics)
		}
		return measureBlockIntrinsicWidth(box, mode, fontMetrics)
	}
}

// measureMinContentWidth returns the min-content width for the
// given box. This is a convenience wrapper around
// measureIntrinsicWidth.
//
// Takes box (*LayoutBox) which is the box to measure.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the min-content width in points.
func measureMinContentWidth(box *LayoutBox, fontMetrics FontMetricsPort) float64 {
	return measureIntrinsicWidth(box, SizingModeMinContent, fontMetrics)
}

// measureMaxContentWidth returns the max-content width for the
// given box. This is a convenience wrapper around
// measureIntrinsicWidth.
//
// Takes box (*LayoutBox) which is the box to measure.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the max-content width in points.
func measureMaxContentWidth(box *LayoutBox, fontMetrics FontMetricsPort) float64 {
	return measureIntrinsicWidth(box, SizingModeMaxContent, fontMetrics)
}

// measureChildrenIntrinsicWidth returns the widest child's
// intrinsic width, bypassing the parent box's declared width.
//
// This is needed when computing fit-content for min-width/
// max-width constraints, because measureBlockIntrinsicWidth
// short-circuits to the declared width when one is set.
//
// Takes box (*LayoutBox) which is the parent box.
// Takes mode (SizingMode) which selects min-content or max-content.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the widest child's intrinsic width in points.
func measureChildrenIntrinsicWidth(box *LayoutBox, mode SizingMode, fontMetrics FontMetricsPort) float64 {
	maxChildWidth := 0.0
	for _, child := range box.Children {
		if child.Type == BoxListMarker {
			continue
		}
		childWidth := measureIntrinsicWidth(child, mode, fontMetrics)
		if childWidth > maxChildWidth {
			maxChildWidth = childWidth
		}
	}
	return maxChildWidth
}

// measureTextIntrinsicWidth computes the intrinsic width of a text run box.
//
// Takes box (*LayoutBox) which is the text run box.
// Takes mode (SizingMode) which selects min-content or max-content.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the intrinsic width in points.
func measureTextIntrinsicWidth(box *LayoutBox, mode SizingMode, fontMetrics FontMetricsPort) float64 {
	font := FontDescriptor{
		Family: box.Style.FontFamily,
		Weight: box.Style.FontWeight,
		Style:  box.Style.FontStyle,
	}

	text := applyTextTransform(box.Text, box.Style.TextTransform)

	if mode == SizingModeMinContent {
		words := splitIntoWords(text)
		widest := 0.0
		for _, word := range words {
			wordWidth := fontMetrics.MeasureText(font, box.Style.FontSize, word, box.Style.Direction)
			if box.Style.LetterSpacing != 0 {
				wordWidth += box.Style.LetterSpacing * float64(len([]rune(word)))
			}
			if wordWidth > widest {
				widest = wordWidth
			}
		}
		return widest
	}

	totalWidth := fontMetrics.MeasureText(font, box.Style.FontSize, text, box.Style.Direction)
	if box.Style.LetterSpacing != 0 {
		totalWidth += box.Style.LetterSpacing * float64(len([]rune(text)))
	}
	return totalWidth
}

// measureBlockIntrinsicWidth computes the intrinsic width of a block box.
//
// Takes box (*LayoutBox) which is the block box.
// Takes mode (SizingMode) which selects min-content or max-content.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the intrinsic width in points.
func measureBlockIntrinsicWidth(box *LayoutBox, mode SizingMode, fontMetrics FontMetricsPort) float64 {
	if !box.Style.Width.IsAuto() && !box.Style.Width.IsIntrinsic() {
		if box.Style.BoxSizing == BoxSizingBorderBox {
			return box.Style.Width.Resolve(0, 0)
		}
		return box.Style.Width.Resolve(0, 0) +
			box.Style.PaddingLeft + box.Style.PaddingRight +
			box.Style.BorderLeftWidth + box.Style.BorderRightWidth
	}

	maxChildWidth := 0.0
	for _, child := range box.Children {
		if child.Type == BoxListMarker {
			continue
		}
		childWidth := measureIntrinsicWidth(child, mode, fontMetrics)
		if childWidth > maxChildWidth {
			maxChildWidth = childWidth
		}
	}
	return maxChildWidth + box.Style.PaddingLeft + box.Style.PaddingRight +
		box.Style.BorderLeftWidth + box.Style.BorderRightWidth
}

// measureTableIntrinsicWidth computes the intrinsic width of a
// table by summing column preferred widths plus border-spacing
// and the table's own padding and border.
//
// This is needed because the generic block measurement takes the
// max of children (rows), but a table row's width is the sum of
// its cells, not the max.
//
// Takes box (*LayoutBox) which is the table box.
// Takes fontMetrics (FontMetricsPort) which provides text
// measurement.
//
// Returns float64 which is the table's intrinsic width in points.
func measureTableIntrinsicWidth(box *LayoutBox, fontMetrics FontMetricsPort) float64 {
	rows, columnCount := collectTableRows(box)
	if columnCount == 0 {
		return box.Style.PaddingLeft + box.Style.PaddingRight +
			box.Style.BorderLeftWidth + box.Style.BorderRightWidth
	}

	_, preferredWidths := measureColumnWidths(rows, columnCount, fontMetrics)

	placements, _, _ := buildTableGrid(rows)
	spacing := box.Style.BorderSpacing
	adjustColumnWidthsForColspan(placements, preferredWidths, columnCount, spacing, fontMetrics)

	total := 0.0
	for _, width := range preferredWidths {
		total += width
	}

	total += spacing * float64(columnCount+1)

	total += box.Style.PaddingLeft + box.Style.PaddingRight +
		box.Style.BorderLeftWidth + box.Style.BorderRightWidth

	return total
}

// measureFlexIntrinsicWidth computes the intrinsic width of a flex container.
//
// Takes box (*LayoutBox) which is the flex container box.
// Takes mode (SizingMode) which selects min-content or max-content.
// Takes fontMetrics (FontMetricsPort) which provides text measurement.
//
// Returns float64 which is the intrinsic width in points.
func measureFlexIntrinsicWidth(box *LayoutBox, mode SizingMode, fontMetrics FontMetricsPort) float64 {
	isRow := box.Style.FlexDirection == FlexDirectionRow ||
		box.Style.FlexDirection == FlexDirectionRowReverse

	total := 0.0
	maxItemWidth := 0.0
	for index, child := range box.Children {
		itemWidth := measureIntrinsicWidth(child, mode, fontMetrics)

		if isRow && mode == SizingModeMaxContent {
			if index > 0 {
				total += box.Style.ColumnGap
			}
			total += itemWidth
		} else {
			if itemWidth > maxItemWidth {
				maxItemWidth = itemWidth
			}
		}
	}

	outerPadding := box.Style.PaddingLeft + box.Style.PaddingRight +
		box.Style.BorderLeftWidth + box.Style.BorderRightWidth

	if isRow && mode == SizingModeMaxContent {
		return total + outerPadding
	}
	return maxItemWidth + outerPadding
}
