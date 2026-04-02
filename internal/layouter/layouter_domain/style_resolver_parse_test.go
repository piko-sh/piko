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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDisplay(t *testing.T) {
	testCases := []struct {
		input    string
		expected DisplayType
	}{
		{"block", DisplayBlock},
		{"inline", DisplayInline},
		{"inline-block", DisplayInlineBlock},
		{"flex", DisplayFlex},
		{"inline-flex", DisplayInlineFlex},
		{"table", DisplayTable},
		{"table-row", DisplayTableRow},
		{"table-cell", DisplayTableCell},
		{"table-row-group", DisplayTableRowGroup},
		{"table-header-group", DisplayTableHeaderGroup},
		{"table-footer-group", DisplayTableFooterGroup},
		{"table-caption", DisplayTableCaption},
		{"list-item", DisplayListItem},
		{"grid", DisplayGrid},
		{"inline-grid", DisplayInlineGrid},
		{"none", DisplayNone},
		{"contents", DisplayContents},
		{"unknown", DisplayInline},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseDisplay(tc.input))
		})
	}
}

func TestParsePosition(t *testing.T) {
	testCases := []struct {
		input    string
		expected PositionType
	}{
		{"relative", PositionRelative},
		{"absolute", PositionAbsolute},
		{"fixed", PositionFixed},
		{"static", PositionStatic},
		{"unknown", PositionStatic},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parsePosition(tc.input))
		})
	}
}

func TestParseBoxSizing(t *testing.T) {
	testCases := []struct {
		input    string
		expected BoxSizingType
	}{
		{"border-box", BoxSizingBorderBox},
		{"content-box", BoxSizingContentBox},
		{"unknown", BoxSizingContentBox},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseBoxSizing(tc.input))
		})
	}
}

func TestParseFloat(t *testing.T) {
	testCases := []struct {
		input    string
		expected FloatType
	}{
		{"left", FloatLeft},
		{"right", FloatRight},
		{"none", FloatNone},
		{"unknown", FloatNone},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseFloat(tc.input))
		})
	}
}

func TestParseClear(t *testing.T) {
	testCases := []struct {
		input    string
		expected ClearType
	}{
		{"left", ClearLeft},
		{"right", ClearRight},
		{"both", ClearBoth},
		{"none", ClearNone},
		{"unknown", ClearNone},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseClear(tc.input))
		})
	}
}

func TestParseVisibility(t *testing.T) {
	testCases := []struct {
		input    string
		expected VisibilityType
	}{
		{"hidden", VisibilityHidden},
		{"collapse", VisibilityCollapse},
		{"visible", VisibilityVisible},
		{"unknown", VisibilityVisible},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseVisibility(tc.input))
		})
	}
}

func TestParseOverflow(t *testing.T) {
	testCases := []struct {
		input    string
		expected OverflowType
	}{
		{"hidden", OverflowHidden},
		{"scroll", OverflowScroll},
		{"auto", OverflowAuto},
		{"visible", OverflowVisible},
		{"unknown", OverflowVisible},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseOverflow(tc.input))
		})
	}
}

func TestParseBorderStyle(t *testing.T) {
	testCases := []struct {
		input    string
		expected BorderStyleType
	}{
		{"solid", BorderStyleSolid},
		{"dashed", BorderStyleDashed},
		{"dotted", BorderStyleDotted},
		{"double", BorderStyleDouble},
		{"groove", BorderStyleGroove},
		{"ridge", BorderStyleRidge},
		{"inset", BorderStyleInset},
		{"outset", BorderStyleOutset},
		{"none", BorderStyleNone},
		{"unknown", BorderStyleNone},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseBorderStyle(tc.input))
		})
	}
}

func TestParseTextAlign(t *testing.T) {
	testCases := []struct {
		input    string
		expected TextAlignType
	}{
		{"center", TextAlignCentre},
		{"centre", TextAlignCentre},
		{"right", TextAlignRight},
		{"justify", TextAlignJustify},
		{"left", TextAlignLeft},
		{"unknown", TextAlignLeft},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseTextAlign(tc.input))
		})
	}
}

func TestParseTextDecoration(t *testing.T) {
	testCases := []struct {
		input    string
		expected TextDecorationFlag
	}{
		{"underline", TextDecorationUnderline},
		{"overline", TextDecorationOverline},
		{"line-through", TextDecorationLineThrough},
		{"none", TextDecorationNone},
		{"unknown", TextDecorationNone},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseTextDecoration(tc.input))
		})
	}
}

func TestParseTextDecorationStyle(t *testing.T) {
	testCases := []struct {
		input    string
		expected TextDecorationStyleType
	}{
		{"dashed", TextDecorationStyleDashed},
		{"dotted", TextDecorationStyleDotted},
		{"double", TextDecorationStyleDouble},
		{"wavy", TextDecorationStyleWavy},
		{"solid", TextDecorationStyleSolid},
		{"unknown", TextDecorationStyleSolid},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseTextDecorationStyle(tc.input))
		})
	}
}

func TestParseTextTransform(t *testing.T) {
	testCases := []struct {
		input    string
		expected TextTransformType
	}{
		{"uppercase", TextTransformUppercase},
		{"lowercase", TextTransformLowercase},
		{"capitalize", TextTransformCapitalise},
		{"capitalise", TextTransformCapitalise},
		{"none", TextTransformNone},
		{"unknown", TextTransformNone},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseTextTransform(tc.input))
		})
	}
}

func TestParseWhiteSpace(t *testing.T) {
	testCases := []struct {
		input    string
		expected WhiteSpaceType
	}{
		{"pre", WhiteSpacePre},
		{"nowrap", WhiteSpaceNowrap},
		{"pre-wrap", WhiteSpacePreWrap},
		{"pre-line", WhiteSpacePreLine},
		{"normal", WhiteSpaceNormal},
		{"unknown", WhiteSpaceNormal},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseWhiteSpace(tc.input))
		})
	}
}

func TestParseWordBreak(t *testing.T) {
	testCases := []struct {
		input    string
		expected WordBreakType
	}{
		{"break-all", WordBreakBreakAll},
		{"keep-all", WordBreakKeepAll},
		{"normal", WordBreakNormal},
		{"unknown", WordBreakNormal},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseWordBreak(tc.input))
		})
	}
}

func TestParseOverflowWrap(t *testing.T) {
	testCases := []struct {
		input    string
		expected OverflowWrapType
	}{
		{"break-word", OverflowWrapBreakWord},
		{"anywhere", OverflowWrapAnywhere},
		{"normal", OverflowWrapNormal},
		{"unknown", OverflowWrapNormal},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseOverflowWrap(tc.input))
		})
	}
}

func TestParseFontFamily(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"double quoted", `"Helvetica"`, "Helvetica"},
		{"single quoted", `'Arial'`, "Arial"},
		{"comma separated takes first", `"Times New Roman", serif`, `"Times New Roman"`},
		{"unquoted generic", "sans-serif", "sans-serif"},
		{"empty string", "", ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseFontFamily(tc.input))
		})
	}
}

func TestParseFontWeight(t *testing.T) {
	testCases := []struct {
		input    string
		expected int
	}{
		{"normal", 400},
		{"bold", 700},
		{"lighter", 100},
		{"bolder", 900},
		{"500", 500},
		{"invalid", 400},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseFontWeight(tc.input))
		})
	}
}

func TestParseFontStyle(t *testing.T) {
	testCases := []struct {
		input    string
		expected FontStyle
	}{
		{"italic", FontStyleItalic},
		{"oblique", FontStyleItalic},
		{"normal", FontStyleNormal},
		{"unknown", FontStyleNormal},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseFontStyle(tc.input))
		})
	}
}

func TestParseFlexDirection(t *testing.T) {
	testCases := []struct {
		input    string
		expected FlexDirectionType
	}{
		{"row-reverse", FlexDirectionRowReverse},
		{"column", FlexDirectionColumn},
		{"column-reverse", FlexDirectionColumnReverse},
		{"row", FlexDirectionRow},
		{"unknown", FlexDirectionRow},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseFlexDirection(tc.input))
		})
	}
}

func TestParseFlexWrap(t *testing.T) {
	testCases := []struct {
		input    string
		expected FlexWrapType
	}{
		{"wrap", FlexWrapWrap},
		{"wrap-reverse", FlexWrapWrapReverse},
		{"nowrap", FlexWrapNowrap},
		{"unknown", FlexWrapNowrap},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseFlexWrap(tc.input))
		})
	}
}

func TestParseJustifyContent(t *testing.T) {
	testCases := []struct {
		input    string
		expected JustifyContentType
	}{
		{"flex-end", JustifyFlexEnd},
		{"center", JustifyCentre},
		{"centre", JustifyCentre},
		{"space-between", JustifySpaceBetween},
		{"space-around", JustifySpaceAround},
		{"space-evenly", JustifySpaceEvenly},
		{"flex-start", JustifyFlexStart},
		{"unknown", JustifyFlexStart},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseJustifyContent(tc.input))
		})
	}
}

func TestParseAlignItems(t *testing.T) {
	testCases := []struct {
		input    string
		expected AlignItemsType
	}{
		{"flex-start", AlignItemsFlexStart},
		{"flex-end", AlignItemsFlexEnd},
		{"center", AlignItemsCentre},
		{"centre", AlignItemsCentre},
		{"baseline", AlignItemsBaseline},
		{"stretch", AlignItemsStretch},
		{"unknown", AlignItemsStretch},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseAlignItems(tc.input))
		})
	}
}

func TestParseAlignSelf(t *testing.T) {
	testCases := []struct {
		input    string
		expected AlignSelfType
	}{
		{"flex-start", AlignSelfFlexStart},
		{"start", AlignSelfFlexStart},
		{"flex-end", AlignSelfFlexEnd},
		{"end", AlignSelfFlexEnd},
		{"center", AlignSelfCentre},
		{"centre", AlignSelfCentre},
		{"baseline", AlignSelfBaseline},
		{"stretch", AlignSelfStretch},
		{"auto", AlignSelfAuto},
		{"unknown", AlignSelfAuto},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseAlignSelf(tc.input))
		})
	}
}

func TestParseAlignContent(t *testing.T) {
	testCases := []struct {
		input    string
		expected AlignContentType
	}{
		{"flex-start", AlignContentFlexStart},
		{"flex-end", AlignContentFlexEnd},
		{"center", AlignContentCentre},
		{"centre", AlignContentCentre},
		{"space-between", AlignContentSpaceBetween},
		{"space-around", AlignContentSpaceAround},
		{"stretch", AlignContentStretch},
		{"unknown", AlignContentStretch},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseAlignContent(tc.input))
		})
	}
}

func TestParseJustifyItems(t *testing.T) {
	testCases := []struct {
		input    string
		expected JustifyItemsType
	}{
		{"start", JustifyItemsStart},
		{"end", JustifyItemsEnd},
		{"center", JustifyItemsCentre},
		{"centre", JustifyItemsCentre},
		{"stretch", JustifyItemsStretch},
		{"unknown", JustifyItemsStretch},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseJustifyItems(tc.input))
		})
	}
}

func TestParseJustifySelf(t *testing.T) {
	testCases := []struct {
		input    string
		expected JustifySelfType
	}{
		{"stretch", JustifySelfStretch},
		{"start", JustifySelfStart},
		{"end", JustifySelfEnd},
		{"center", JustifySelfCentre},
		{"centre", JustifySelfCentre},
		{"auto", JustifySelfAuto},
		{"unknown", JustifySelfAuto},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseJustifySelf(tc.input))
		})
	}
}

func TestParseTableLayout(t *testing.T) {
	testCases := []struct {
		input    string
		expected TableLayoutType
	}{
		{"fixed", TableLayoutFixed},
		{"auto", TableLayoutAuto},
		{"unknown", TableLayoutAuto},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseTableLayout(tc.input))
		})
	}
}

func TestParseBorderCollapse(t *testing.T) {
	testCases := []struct {
		input    string
		expected BorderCollapseType
	}{
		{"collapse", BorderCollapseCollapse},
		{"separate", BorderCollapseSeparate},
		{"unknown", BorderCollapseSeparate},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseBorderCollapse(tc.input))
		})
	}
}

func TestParseCaptionSide(t *testing.T) {
	testCases := []struct {
		input    string
		expected CaptionSideType
	}{
		{"bottom", CaptionSideBottom},
		{"top", CaptionSideTop},
		{"unknown", CaptionSideTop},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseCaptionSide(tc.input))
		})
	}
}

func TestParseVerticalAlign(t *testing.T) {
	testCases := []struct {
		input    string
		expected VerticalAlignType
	}{
		{"top", VerticalAlignTop},
		{"middle", VerticalAlignMiddle},
		{"bottom", VerticalAlignBottom},
		{"super", VerticalAlignSuper},
		{"sub", VerticalAlignSub},
		{"text-top", VerticalAlignTextTop},
		{"text-bottom", VerticalAlignTextBottom},
		{"baseline", VerticalAlignBaseline},
		{"unknown", VerticalAlignBaseline},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseVerticalAlign(tc.input))
		})
	}
}

func TestParsePageBreak(t *testing.T) {
	testCases := []struct {
		input    string
		expected PageBreakType
	}{
		{"always", PageBreakAlways},
		{"avoid", PageBreakAvoid},
		{"left", PageBreakLeft},
		{"right", PageBreakRight},
		{"auto", PageBreakAuto},
		{"unknown", PageBreakAuto},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parsePageBreak(tc.input))
		})
	}
}

func TestParseWritingMode(t *testing.T) {
	testCases := []struct {
		input    string
		expected WritingModeType
	}{
		{"vertical-rl", WritingModeVerticalRL},
		{"vertical-lr", WritingModeVerticalLR},
		{"horizontal-tb", WritingModeHorizontalTB},
		{"unknown", WritingModeHorizontalTB},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseWritingMode(tc.input))
		})
	}
}

func TestParseObjectFit(t *testing.T) {
	testCases := []struct {
		input    string
		expected ObjectFitType
	}{
		{"contain", ObjectFitContain},
		{"cover", ObjectFitCover},
		{"none", ObjectFitNone},
		{"scale-down", ObjectFitScaleDown},
		{"fill", ObjectFitFill},
		{"unknown", ObjectFitFill},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseObjectFit(tc.input))
		})
	}
}

func TestParseHyphens(t *testing.T) {
	testCases := []struct {
		input    string
		expected HyphensType
	}{
		{"none", HyphensNone},
		{"auto", HyphensAuto},
		{"manual", HyphensManual},
		{"unknown", HyphensManual},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseHyphens(tc.input))
		})
	}
}

func TestParseDirection(t *testing.T) {
	testCases := []struct {
		input    string
		expected DirectionType
	}{
		{"rtl", DirectionRTL},
		{"ltr", DirectionLTR},
		{"unknown", DirectionLTR},
	}
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseDirection(tc.input))
		})
	}
}

func TestParseCounterOperations(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		defaultValue int
		expected     []CounterEntry
	}{
		{"none returns nil", "none", 0, nil},
		{"empty returns nil", "", 0, nil},
		{"single counter with default value", "my-counter", 0, []CounterEntry{{Name: "my-counter", Value: 0}}},
		{"single counter with explicit value", "my-counter 5", 0, []CounterEntry{{Name: "my-counter", Value: 5}}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseCounterOperations(tc.input, tc.defaultValue))
		})
	}
}

func TestParseColour(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected Colour
	}{
		{"named colour red", "red", func() Colour { c, _ := ParseColour("red"); return c }()},
		{"hex colour green", "#00ff00", func() Colour { c, _ := ParseColour("#00ff00"); return c }()},
		{"invalid falls back to black", "invalidcolour", ColourBlack},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseColour(tc.input))
		})
	}
}

func TestParseFloatValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected float64
	}{
		{"valid float", "2.5", 2.5},
		{"zero", "0", 0.0},
		{"invalid returns zero", "abc", 0.0},
		{"whitespace trimmed", " 3.14 ", 3.14},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.expected, parseFloatValue(tc.input), 0.001)
		})
	}
}

func TestParseIntValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"valid int", "42", 42},
		{"zero", "0", 0},
		{"invalid returns zero", "abc", 0},
		{"whitespace trimmed", " 7 ", 7},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseIntValue(tc.input))
		})
	}
}
