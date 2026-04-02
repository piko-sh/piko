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

func TestDisplayType_String(t *testing.T) {
	testCases := []struct {
		value    DisplayType
		expected string
	}{
		{DisplayBlock, "block"},
		{DisplayInline, "inline"},
		{DisplayInlineBlock, "inline-block"},
		{DisplayFlex, "flex"},
		{DisplayInlineFlex, "inline-flex"},
		{DisplayTable, "table"},
		{DisplayTableRow, "table-row"},
		{DisplayTableCell, "table-cell"},
		{DisplayTableRowGroup, "table-row-group"},
		{DisplayTableHeaderGroup, "table-header-group"},
		{DisplayTableFooterGroup, "table-footer-group"},
		{DisplayTableCaption, "table-caption"},
		{DisplayListItem, "list-item"},
		{DisplayGrid, "grid"},
		{DisplayInlineGrid, "inline-grid"},
		{DisplayNone, "none"},
		{DisplayContents, "contents"},
		{DisplayType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == DisplayType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestPositionType_String(t *testing.T) {
	testCases := []struct {
		value    PositionType
		expected string
	}{
		{PositionStatic, "static"},
		{PositionRelative, "relative"},
		{PositionAbsolute, "absolute"},
		{PositionFixed, "fixed"},
		{PositionType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == PositionType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestFloatType_String(t *testing.T) {
	testCases := []struct {
		value    FloatType
		expected string
	}{
		{FloatNone, "none"},
		{FloatLeft, "left"},
		{FloatRight, "right"},
		{FloatType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == FloatType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestClearType_String(t *testing.T) {
	testCases := []struct {
		value    ClearType
		expected string
	}{
		{ClearNone, "none"},
		{ClearLeft, "left"},
		{ClearRight, "right"},
		{ClearBoth, "both"},
		{ClearType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == ClearType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestTextAlignType_String(t *testing.T) {
	testCases := []struct {
		value    TextAlignType
		expected string
	}{
		{TextAlignLeft, "left"},
		{TextAlignCentre, "centre"},
		{TextAlignRight, "right"},
		{TextAlignJustify, "justify"},
		{TextAlignType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == TextAlignType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestTextDecorationStyleType_String(t *testing.T) {
	testCases := []struct {
		value    TextDecorationStyleType
		expected string
	}{
		{TextDecorationStyleSolid, "solid"},
		{TextDecorationStyleDashed, "dashed"},
		{TextDecorationStyleDotted, "dotted"},
		{TextDecorationStyleDouble, "double"},
		{TextDecorationStyleWavy, "wavy"},
		{TextDecorationStyleType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == TextDecorationStyleType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestTextTransformType_String(t *testing.T) {
	testCases := []struct {
		value    TextTransformType
		expected string
	}{
		{TextTransformNone, "none"},
		{TextTransformUppercase, "uppercase"},
		{TextTransformLowercase, "lowercase"},
		{TextTransformCapitalise, "capitalise"},
		{TextTransformType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == TextTransformType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestWhiteSpaceType_String(t *testing.T) {
	testCases := []struct {
		value    WhiteSpaceType
		expected string
	}{
		{WhiteSpaceNormal, "normal"},
		{WhiteSpacePre, "pre"},
		{WhiteSpaceNowrap, "nowrap"},
		{WhiteSpacePreWrap, "pre-wrap"},
		{WhiteSpacePreLine, "pre-line"},
		{WhiteSpaceType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == WhiteSpaceType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestWordBreakType_String(t *testing.T) {
	testCases := []struct {
		value    WordBreakType
		expected string
	}{
		{WordBreakNormal, "normal"},
		{WordBreakBreakAll, "break-all"},
		{WordBreakKeepAll, "keep-all"},
		{WordBreakType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == WordBreakType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestOverflowWrapType_String(t *testing.T) {
	testCases := []struct {
		value    OverflowWrapType
		expected string
	}{
		{OverflowWrapNormal, "Normal"},
		{OverflowWrapBreakWord, "BreakWord"},
		{OverflowWrapAnywhere, "Anywhere"},
		{OverflowWrapType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == OverflowWrapType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestOverflowType_String(t *testing.T) {
	testCases := []struct {
		value    OverflowType
		expected string
	}{
		{OverflowVisible, "visible"},
		{OverflowHidden, "hidden"},
		{OverflowScroll, "scroll"},
		{OverflowAuto, "auto"},
		{OverflowType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == OverflowType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestVisibilityType_String(t *testing.T) {
	testCases := []struct {
		value    VisibilityType
		expected string
	}{
		{VisibilityVisible, "visible"},
		{VisibilityHidden, "hidden"},
		{VisibilityCollapse, "collapse"},
		{VisibilityType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == VisibilityType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestBorderStyleType_String(t *testing.T) {
	testCases := []struct {
		value    BorderStyleType
		expected string
	}{
		{BorderStyleNone, "none"},
		{BorderStyleSolid, "solid"},
		{BorderStyleDashed, "dashed"},
		{BorderStyleDotted, "dotted"},
		{BorderStyleDouble, "double"},
		{BorderStyleGroove, "groove"},
		{BorderStyleRidge, "ridge"},
		{BorderStyleInset, "inset"},
		{BorderStyleOutset, "outset"},
		{BorderStyleType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == BorderStyleType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestBoxSizingType_String(t *testing.T) {
	testCases := []struct {
		value    BoxSizingType
		expected string
	}{
		{BoxSizingContentBox, "content-box"},
		{BoxSizingBorderBox, "border-box"},
		{BoxSizingType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == BoxSizingType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestDirectionType_String(t *testing.T) {
	testCases := []struct {
		value    DirectionType
		expected string
	}{
		{DirectionLTR, "ltr"},
		{DirectionRTL, "rtl"},
		{DirectionType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == DirectionType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestHyphensType_String(t *testing.T) {
	testCases := []struct {
		value    HyphensType
		expected string
	}{
		{HyphensNone, "none"},
		{HyphensManual, "manual"},
		{HyphensAuto, "auto"},
		{HyphensType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == HyphensType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}
