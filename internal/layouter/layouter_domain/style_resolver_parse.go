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
	"strconv"
	"strings"
)

const (
	// cssKeywordNormal is the CSS "normal" keyword string.
	cssKeywordNormal = "normal"

	// cssKeywordCenter is the CSS keyword for centre alignment.
	cssKeywordCenter = "center"

	// fontWeightNormal is the numeric weight for normal text.
	fontWeightNormal = 400

	// fontWeightBold is the numeric weight for bold text.
	fontWeightBold = 700

	// fontWeightMin is the minimum valid CSS font-weight value.
	fontWeightMin = 100

	// fontWeightMax is the maximum valid CSS font-weight value.
	fontWeightMax = 900

	// fontSizeScaleXXSmall is the scale factor for xx-small (9/16 of medium).
	fontSizeScaleXXSmall = 0.5625

	// fontSizeScaleXSmall is the scale factor for x-small (10/16 of medium).
	fontSizeScaleXSmall = 0.625

	// fontSizeScaleSmall is the scale factor for small (13/16 of medium).
	fontSizeScaleSmall = 0.8125

	// fontSizeScaleLarge is the scale factor for large (18/16 of medium).
	fontSizeScaleLarge = 1.125

	// fontSizeScaleXLarge is the scale factor for x-large (24/16 of medium).
	fontSizeScaleXLarge = 1.5

	// fontSizeScaleXXLarge is the scale factor for xx-large (32/16 of medium).
	fontSizeScaleXXLarge = 2.0

	// fontSizeScaleSmaller is the scale factor for smaller.
	fontSizeScaleSmaller = 0.83

	// fontSizeScaleLarger is the scale factor for larger.
	fontSizeScaleLarger = 1.2
)

// parseDisplay maps a CSS display value string to a
// DisplayType enum.
//
// Takes value (string) which is the CSS display value.
//
// Returns DisplayType which is the corresponding enum.
func parseDisplay(value string) DisplayType {
	switch value {
	case "block":
		return DisplayBlock
	case "inline-block":
		return DisplayInlineBlock
	case "flex":
		return DisplayFlex
	case "inline-flex":
		return DisplayInlineFlex
	case "table":
		return DisplayTable
	case "table-row":
		return DisplayTableRow
	case "table-cell":
		return DisplayTableCell
	case "table-row-group":
		return DisplayTableRowGroup
	case "table-header-group":
		return DisplayTableHeaderGroup
	case "table-footer-group":
		return DisplayTableFooterGroup
	case "table-caption":
		return DisplayTableCaption
	case "list-item":
		return DisplayListItem
	case "grid":
		return DisplayGrid
	case "inline-grid":
		return DisplayInlineGrid
	case cssKeywordNone:
		return DisplayNone
	case "contents":
		return DisplayContents
	default:
		return DisplayInline
	}
}

// parsePosition maps a CSS position value string to a
// PositionType enum.
//
// Takes value (string) which is the CSS position value.
//
// Returns PositionType which is the corresponding enum.
func parsePosition(value string) PositionType {
	switch value {
	case "relative":
		return PositionRelative
	case "absolute":
		return PositionAbsolute
	case "fixed":
		return PositionFixed
	default:
		return PositionStatic
	}
}

// parseBoxSizing maps a CSS box-sizing value string to a
// BoxSizingType enum.
//
// Takes value (string) which is the CSS box-sizing value.
//
// Returns BoxSizingType which is the corresponding enum.
func parseBoxSizing(value string) BoxSizingType {
	if value == "border-box" {
		return BoxSizingBorderBox
	}
	return BoxSizingContentBox
}

// parseFloat maps a CSS float value string to a FloatType
// enum.
//
// Takes value (string) which is the CSS float value.
//
// Returns FloatType which is the corresponding enum.
func parseFloat(value string) FloatType {
	switch value {
	case cssKeywordLeft:
		return FloatLeft
	case cssKeywordRight:
		return FloatRight
	default:
		return FloatNone
	}
}

// parseClear maps a CSS clear value string to a ClearType
// enum.
//
// Takes value (string) which is the CSS clear value.
//
// Returns ClearType which is the corresponding enum.
func parseClear(value string) ClearType {
	switch value {
	case cssKeywordLeft:
		return ClearLeft
	case cssKeywordRight:
		return ClearRight
	case "both":
		return ClearBoth
	default:
		return ClearNone
	}
}

// parseVisibility maps a CSS visibility value string to a
// VisibilityType enum.
//
// Takes value (string) which is the CSS visibility
// value.
//
// Returns VisibilityType which is the corresponding
// enum.
func parseVisibility(value string) VisibilityType {
	switch value {
	case "hidden":
		return VisibilityHidden
	case "collapse":
		return VisibilityCollapse
	default:
		return VisibilityVisible
	}
}

// parseOverflow maps a CSS overflow value string to an
// OverflowType enum.
//
// Takes value (string) which is the CSS overflow value.
//
// Returns OverflowType which is the corresponding enum.
func parseOverflow(value string) OverflowType {
	switch value {
	case "hidden":
		return OverflowHidden
	case "scroll":
		return OverflowScroll
	case cssKeywordAuto:
		return OverflowAuto
	default:
		return OverflowVisible
	}
}

// parseBorderStyle maps a CSS border-style value string to
// a BorderStyleType enum.
//
// Takes value (string) which is the CSS border-style
// value.
//
// Returns BorderStyleType which is the corresponding
// enum.
func parseBorderStyle(value string) BorderStyleType {
	switch value {
	case "solid":
		return BorderStyleSolid
	case "dashed":
		return BorderStyleDashed
	case "dotted":
		return BorderStyleDotted
	case "double":
		return BorderStyleDouble
	case "groove":
		return BorderStyleGroove
	case "ridge":
		return BorderStyleRidge
	case "inset":
		return BorderStyleInset
	case "outset":
		return BorderStyleOutset
	default:
		return BorderStyleNone
	}
}

// parseTextAlign maps a CSS text-align value string to a
// TextAlignType enum.
//
// Takes value (string) which is the CSS text-align
// value.
//
// Returns TextAlignType which is the corresponding
// enum.
func parseTextAlign(value string) TextAlignType {
	switch value {
	case cssKeywordCenter, cssKeywordCentre:
		return TextAlignCentre
	case cssKeywordRight:
		return TextAlignRight
	case "justify":
		return TextAlignJustify
	case cssKeywordStart:
		return TextAlignStart
	case cssKeywordEnd:
		return TextAlignEnd
	default:
		return TextAlignLeft
	}
}

// parseTextDecoration parses a CSS text-decoration value
// into a bitmask of TextDecorationFlag values.
//
// Takes value (string) which is the CSS text-decoration
// value.
//
// Returns TextDecorationFlag which is the bitmask of
// active decorations.
func parseTextDecoration(value string) TextDecorationFlag {
	var result TextDecorationFlag
	for part := range strings.FieldsSeq(value) {
		switch part {
		case "underline":
			result |= TextDecorationUnderline
		case "overline":
			result |= TextDecorationOverline
		case "line-through":
			result |= TextDecorationLineThrough
		}
	}
	return result
}

// parseTextDecorationStyle maps a CSS text-decoration-style
// value string to a TextDecorationStyleType enum.
//
// Takes value (string) which is the CSS
// text-decoration-style value.
//
// Returns TextDecorationStyleType which is the
// corresponding enum.
func parseTextDecorationStyle(value string) TextDecorationStyleType {
	switch value {
	case "dashed":
		return TextDecorationStyleDashed
	case "dotted":
		return TextDecorationStyleDotted
	case "double":
		return TextDecorationStyleDouble
	case "wavy":
		return TextDecorationStyleWavy
	default:
		return TextDecorationStyleSolid
	}
}

// parseTextTransform maps a CSS text-transform value string
// to a TextTransformType enum.
//
// Takes value (string) which is the CSS text-transform
// value.
//
// Returns TextTransformType which is the corresponding
// enum.
func parseTextTransform(value string) TextTransformType {
	switch value {
	case "uppercase":
		return TextTransformUppercase
	case "lowercase":
		return TextTransformLowercase
	case "capitalize", "capitalise":
		return TextTransformCapitalise
	default:
		return TextTransformNone
	}
}

// parseWhiteSpace maps a CSS white-space value string to a
// WhiteSpaceType enum.
//
// Takes value (string) which is the CSS white-space
// value.
//
// Returns WhiteSpaceType which is the corresponding
// enum.
func parseWhiteSpace(value string) WhiteSpaceType {
	switch value {
	case "pre":
		return WhiteSpacePre
	case "nowrap":
		return WhiteSpaceNowrap
	case "pre-wrap":
		return WhiteSpacePreWrap
	case "pre-line":
		return WhiteSpacePreLine
	default:
		return WhiteSpaceNormal
	}
}

// parseWordBreak maps a CSS word-break value string to a
// WordBreakType enum.
//
// Takes value (string) which is the CSS word-break
// value.
//
// Returns WordBreakType which is the corresponding
// enum.
func parseWordBreak(value string) WordBreakType {
	switch value {
	case "break-all":
		return WordBreakBreakAll
	case "keep-all":
		return WordBreakKeepAll
	default:
		return WordBreakNormal
	}
}

// parseCounterOperations parses a CSS counter-reset or
// counter-increment value into a list of counter entries.
//
// Tokens alternate between counter names and optional integer
// values. When no value follows a name, defaultValue is used
// (0 for counter-reset, 1 for counter-increment).
//
// Takes value (string) which is the CSS property value.
// Takes defaultValue (int) which is the fallback integer.
//
// Returns []CounterEntry which is the parsed list.
func parseCounterOperations(value string, defaultValue int) []CounterEntry {
	if value == cssKeywordNone || value == "" {
		return nil
	}
	tokens := strings.Fields(value)
	var entries []CounterEntry
	for i := 0; i < len(tokens); i++ {
		name := tokens[i]
		val := defaultValue
		if i+1 < len(tokens) {
			if n, err := strconv.Atoi(tokens[i+1]); err == nil {
				val = n
				i++
			}
		}
		entries = append(entries, CounterEntry{Name: name, Value: val})
	}
	return entries
}

// parseObjectFit converts a CSS object-fit value string
// to an ObjectFitType enum.
//
// Takes value (string) which is the CSS object-fit value.
//
// Returns ObjectFitType which is the parsed enum value.
func parseObjectFit(value string) ObjectFitType {
	switch value {
	case "contain":
		return ObjectFitContain
	case "cover":
		return ObjectFitCover
	case cssKeywordNone:
		return ObjectFitNone
	case "scale-down":
		return ObjectFitScaleDown
	default:
		return ObjectFitFill
	}
}

// parseOverflowWrap converts a CSS overflow-wrap value
// string to an OverflowWrapType enum.
//
// Takes value (string) which is the CSS overflow-wrap
// value.
//
// Returns OverflowWrapType which is the parsed enum value.
func parseOverflowWrap(value string) OverflowWrapType {
	switch value {
	case "break-word":
		return OverflowWrapBreakWord
	case "anywhere":
		return OverflowWrapAnywhere
	default:
		return OverflowWrapNormal
	}
}

// parseFontFamily extracts the first font family name from
// a CSS font-family value, stripping quotes and commas.
//
// Takes value (string) which is the CSS font-family
// value.
//
// Returns string which is the extracted font family
// name.
func parseFontFamily(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') ||
		(value[0] == '\'' && value[len(value)-1] == '\'')) {
		return value[1 : len(value)-1]
	}
	if before, _, found := strings.Cut(value, commaDelimiter); found {
		return strings.TrimSpace(before)
	}
	return value
}

// parseFontWeight converts a CSS font-weight value to its
// numeric representation.
//
// Takes value (string) which is the CSS font-weight
// value.
//
// Returns int which is the numeric font weight.
func parseFontWeight(value string) int {
	switch value {
	case cssKeywordNormal:
		return fontWeightNormal
	case "bold":
		return fontWeightBold
	case "lighter":
		return fontWeightMin
	case "bolder":
		return fontWeightMax
	default:
		weight, err := strconv.Atoi(value)
		if err != nil {
			return fontWeightNormal
		}
		return weight
	}
}

// parseFontStyle maps a CSS font-style value string to a
// FontStyle enum.
//
// Takes value (string) which is the CSS font-style
// value.
//
// Returns FontStyle which is the corresponding enum.
func parseFontStyle(value string) FontStyle {
	switch value {
	case "italic", "oblique":
		return FontStyleItalic
	default:
		return FontStyleNormal
	}
}

// parseFlexDirection maps a CSS flex-direction value string
// to a FlexDirectionType enum.
//
// Takes value (string) which is the CSS flex-direction
// value.
//
// Returns FlexDirectionType which is the corresponding
// enum.
func parseFlexDirection(value string) FlexDirectionType {
	switch value {
	case "row-reverse":
		return FlexDirectionRowReverse
	case "column":
		return FlexDirectionColumn
	case "column-reverse":
		return FlexDirectionColumnReverse
	default:
		return FlexDirectionRow
	}
}

// parseFlexWrap maps a CSS flex-wrap value string to a
// FlexWrapType enum.
//
// Takes value (string) which is the CSS flex-wrap
// value.
//
// Returns FlexWrapType which is the corresponding enum.
func parseFlexWrap(value string) FlexWrapType {
	switch value {
	case "wrap":
		return FlexWrapWrap
	case "wrap-reverse":
		return FlexWrapWrapReverse
	default:
		return FlexWrapNowrap
	}
}

// parseJustifyContent maps a CSS justify-content value
// string to a JustifyContentType enum.
//
// Takes value (string) which is the CSS
// justify-content value.
//
// Returns JustifyContentType which is the corresponding
// enum.
func parseJustifyContent(value string) JustifyContentType {
	switch value {
	case cssKeywordFlexEnd:
		return JustifyFlexEnd
	case cssKeywordCenter, cssKeywordCentre:
		return JustifyCentre
	case "space-between":
		return JustifySpaceBetween
	case "space-around":
		return JustifySpaceAround
	case "space-evenly":
		return JustifySpaceEvenly
	default:
		return JustifyFlexStart
	}
}

// parseAlignItems maps a CSS align-items value string to an
// AlignItemsType enum.
//
// Takes value (string) which is the CSS align-items
// value.
//
// Returns AlignItemsType which is the corresponding
// enum.
func parseAlignItems(value string) AlignItemsType {
	switch value {
	case cssKeywordFlexStart:
		return AlignItemsFlexStart
	case cssKeywordFlexEnd:
		return AlignItemsFlexEnd
	case cssKeywordCenter, cssKeywordCentre:
		return AlignItemsCentre
	case "baseline":
		return AlignItemsBaseline
	default:
		return AlignItemsStretch
	}
}

// parseAlignSelf maps a CSS align-self value string to an
// AlignSelfType enum.
//
// Takes value (string) which is the CSS align-self
// value.
//
// Returns AlignSelfType which is the corresponding
// enum.
func parseAlignSelf(value string) AlignSelfType {
	switch value {
	case cssKeywordFlexStart, cssKeywordStart:
		return AlignSelfFlexStart
	case cssKeywordFlexEnd, cssKeywordEnd:
		return AlignSelfFlexEnd
	case cssKeywordCenter, cssKeywordCentre:
		return AlignSelfCentre
	case "baseline":
		return AlignSelfBaseline
	case "stretch":
		return AlignSelfStretch
	default:
		return AlignSelfAuto
	}
}

// parseAlignContent maps a CSS align-content value string
// to an AlignContentType enum.
//
// Takes value (string) which is the CSS align-content
// value.
//
// Returns AlignContentType which is the corresponding
// enum.
func parseAlignContent(value string) AlignContentType {
	switch value {
	case cssKeywordFlexStart:
		return AlignContentFlexStart
	case cssKeywordFlexEnd:
		return AlignContentFlexEnd
	case cssKeywordCenter, cssKeywordCentre:
		return AlignContentCentre
	case "space-between":
		return AlignContentSpaceBetween
	case "space-around":
		return AlignContentSpaceAround
	default:
		return AlignContentStretch
	}
}

// parseJustifyItems maps a CSS justify-items value string to
// a JustifyItemsType enum.
//
// Takes value (string) which is the CSS justify-items
// value.
//
// Returns JustifyItemsType which is the corresponding
// enum.
func parseJustifyItems(value string) JustifyItemsType {
	switch value {
	case cssKeywordStart:
		return JustifyItemsStart
	case cssKeywordEnd:
		return JustifyItemsEnd
	case cssKeywordCenter, cssKeywordCentre:
		return JustifyItemsCentre
	default:
		return JustifyItemsStretch
	}
}

// parseJustifySelf maps a CSS justify-self value string to
// a JustifySelfType enum.
//
// Takes value (string) which is the CSS justify-self
// value.
//
// Returns JustifySelfType which is the corresponding
// enum.
func parseJustifySelf(value string) JustifySelfType {
	switch value {
	case "stretch":
		return JustifySelfStretch
	case cssKeywordStart:
		return JustifySelfStart
	case cssKeywordEnd:
		return JustifySelfEnd
	case cssKeywordCenter, cssKeywordCentre:
		return JustifySelfCentre
	default:
		return JustifySelfAuto
	}
}

// parseTableLayout maps a CSS table-layout value string to
// a TableLayoutType enum.
//
// Takes value (string) which is the CSS table-layout
// value.
//
// Returns TableLayoutType which is the corresponding
// enum.
func parseTableLayout(value string) TableLayoutType {
	if value == "fixed" {
		return TableLayoutFixed
	}
	return TableLayoutAuto
}

// parseBorderCollapse maps a CSS border-collapse value
// string to a BorderCollapseType enum.
//
// Takes value (string) which is the CSS border-collapse
// value.
//
// Returns BorderCollapseType which is the corresponding
// enum.
func parseBorderCollapse(value string) BorderCollapseType {
	if value == "collapse" {
		return BorderCollapseCollapse
	}
	return BorderCollapseSeparate
}

// parseCaptionSide maps a CSS caption-side value string to
// a CaptionSideType enum.
//
// Takes value (string) which is the CSS caption-side
// value.
//
// Returns CaptionSideType which is the corresponding
// enum.
func parseCaptionSide(value string) CaptionSideType {
	if value == cssKeywordBottom {
		return CaptionSideBottom
	}
	return CaptionSideTop
}

// parseVerticalAlign maps a CSS vertical-align value string
// to a VerticalAlignType enum.
//
// Takes value (string) which is the CSS vertical-align
// value.
//
// Returns VerticalAlignType which is the corresponding
// enum.
func parseVerticalAlign(value string) VerticalAlignType {
	switch value {
	case cssKeywordTop:
		return VerticalAlignTop
	case "middle":
		return VerticalAlignMiddle
	case cssKeywordBottom:
		return VerticalAlignBottom
	case "super":
		return VerticalAlignSuper
	case "sub":
		return VerticalAlignSub
	case "text-top":
		return VerticalAlignTextTop
	case "text-bottom":
		return VerticalAlignTextBottom
	default:
		return VerticalAlignBaseline
	}
}

// parsePageBreak maps a CSS page-break value string to a
// PageBreakType enum.
//
// Takes value (string) which is the CSS page-break
// value.
//
// Returns PageBreakType which is the corresponding
// enum.
func parsePageBreak(value string) PageBreakType {
	switch value {
	case "always", "page":
		return PageBreakAlways
	case "avoid":
		return PageBreakAvoid
	case cssKeywordLeft:
		return PageBreakLeft
	case cssKeywordRight:
		return PageBreakRight
	default:
		return PageBreakAuto
	}
}

// parseWritingMode parses a CSS writing-mode value into a
// WritingModeType, defaulting to horizontal-tb.
//
// Takes value (string) which is the CSS writing-mode
// value.
//
// Returns WritingModeType which is the parsed writing
// mode.
func parseWritingMode(value string) WritingModeType {
	switch value {
	case "vertical-rl":
		return WritingModeVerticalRL
	case "vertical-lr":
		return WritingModeVerticalLR
	default:
		return WritingModeHorizontalTB
	}
}

// parseColour parses a CSS colour value string into a
// Colour, falling back to black on failure.
//
// Takes value (string) which is the CSS colour value.
//
// Returns Colour which is the parsed colour, or black
// on failure.
func parseColour(value string) Colour {
	colour, ok := ParseColour(value)
	if !ok {
		return ColourBlack
	}
	return colour
}

// parseFloatValue parses a string as a float64, returning
// zero on failure.
//
// Takes value (string) which is the string to parse.
//
// Returns float64 which is the parsed value, or zero
// on failure.
func parseFloatValue(value string) float64 {
	result, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return result
}

// parseIntValue parses a string as an int, returning zero
// on failure.
//
// Takes value (string) which is the string to parse.
//
// Returns int which is the parsed value, or zero on
// failure.
func parseIntValue(value string) int {
	result, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return result
}
