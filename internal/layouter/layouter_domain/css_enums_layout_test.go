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

func TestFlexDirectionType_String(t *testing.T) {
	testCases := []struct {
		value    FlexDirectionType
		expected string
	}{
		{FlexDirectionRow, "row"},
		{FlexDirectionRowReverse, "row-reverse"},
		{FlexDirectionColumn, "column"},
		{FlexDirectionColumnReverse, "column-reverse"},
		{FlexDirectionType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == FlexDirectionType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestFlexWrapType_String(t *testing.T) {
	testCases := []struct {
		value    FlexWrapType
		expected string
	}{
		{FlexWrapNowrap, "nowrap"},
		{FlexWrapWrap, "wrap"},
		{FlexWrapWrapReverse, "wrap-reverse"},
		{FlexWrapType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == FlexWrapType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestJustifyContentType_String(t *testing.T) {
	testCases := []struct {
		value    JustifyContentType
		expected string
	}{
		{JustifyFlexStart, "flex-start"},
		{JustifyFlexEnd, "flex-end"},
		{JustifyCentre, "centre"},
		{JustifySpaceBetween, "space-between"},
		{JustifySpaceAround, "space-around"},
		{JustifySpaceEvenly, "space-evenly"},
		{JustifyContentType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == JustifyContentType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestAlignItemsType_String(t *testing.T) {
	testCases := []struct {
		value    AlignItemsType
		expected string
	}{
		{AlignItemsStretch, "stretch"},
		{AlignItemsFlexStart, "flex-start"},
		{AlignItemsFlexEnd, "flex-end"},
		{AlignItemsCentre, "centre"},
		{AlignItemsBaseline, "baseline"},
		{AlignItemsType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == AlignItemsType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestAlignSelfType_String(t *testing.T) {
	testCases := []struct {
		value    AlignSelfType
		expected string
	}{
		{AlignSelfAuto, "auto"},
		{AlignSelfFlexStart, "flex-start"},
		{AlignSelfFlexEnd, "flex-end"},
		{AlignSelfCentre, "centre"},
		{AlignSelfBaseline, "baseline"},
		{AlignSelfStretch, "stretch"},
		{AlignSelfType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == AlignSelfType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestAlignContentType_String(t *testing.T) {
	testCases := []struct {
		value    AlignContentType
		expected string
	}{
		{AlignContentStretch, "stretch"},
		{AlignContentFlexStart, "flex-start"},
		{AlignContentFlexEnd, "flex-end"},
		{AlignContentCentre, "centre"},
		{AlignContentSpaceBetween, "space-between"},
		{AlignContentSpaceAround, "space-around"},
		{AlignContentType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == AlignContentType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestJustifyItemsType_String(t *testing.T) {
	testCases := []struct {
		value    JustifyItemsType
		expected string
	}{
		{JustifyItemsStretch, "stretch"},
		{JustifyItemsStart, "start"},
		{JustifyItemsEnd, "end"},
		{JustifyItemsCentre, "centre"},
		{JustifyItemsType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == JustifyItemsType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestJustifySelfType_String(t *testing.T) {
	testCases := []struct {
		value    JustifySelfType
		expected string
	}{
		{JustifySelfAuto, "auto"},
		{JustifySelfStretch, "stretch"},
		{JustifySelfStart, "start"},
		{JustifySelfEnd, "end"},
		{JustifySelfCentre, "centre"},
		{JustifySelfType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == JustifySelfType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestGridAutoFlowType_String(t *testing.T) {
	testCases := []struct {
		value    GridAutoFlowType
		expected string
	}{
		{GridAutoFlowRow, "row"},
		{GridAutoFlowColumn, "column"},
		{GridAutoFlowRowDense, "row dense"},
		{GridAutoFlowColumnDense, "column dense"},
		{GridAutoFlowType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == GridAutoFlowType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestPageBreakType_String(t *testing.T) {
	testCases := []struct {
		value    PageBreakType
		expected string
	}{
		{PageBreakAuto, "auto"},
		{PageBreakAlways, "always"},
		{PageBreakAvoid, "avoid"},
		{PageBreakLeft, "left"},
		{PageBreakRight, "right"},
		{PageBreakType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == PageBreakType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestWritingModeType_String(t *testing.T) {
	testCases := []struct {
		value    WritingModeType
		expected string
	}{
		{WritingModeHorizontalTB, "horizontal-tb"},
		{WritingModeVerticalRL, "vertical-rl"},
		{WritingModeVerticalLR, "vertical-lr"},
		{WritingModeType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == WritingModeType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestColumnSpanType_String(t *testing.T) {
	testCases := []struct {
		value    ColumnSpanType
		expected string
	}{
		{ColumnSpanNone, "none"},
		{ColumnSpanAll, "all"},
		{ColumnSpanType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == ColumnSpanType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestCaptionSideType_String(t *testing.T) {
	testCases := []struct {
		value    CaptionSideType
		expected string
	}{
		{CaptionSideTop, "top"},
		{CaptionSideBottom, "bottom"},
		{CaptionSideType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == CaptionSideType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestListStylePositionType_String(t *testing.T) {
	testCases := []struct {
		value    ListStylePositionType
		expected string
	}{
		{ListStylePositionOutside, "outside"},
		{ListStylePositionInside, "inside"},
		{ListStylePositionType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == ListStylePositionType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestVerticalAlignType_String(t *testing.T) {
	testCases := []struct {
		value    VerticalAlignType
		expected string
	}{
		{VerticalAlignBaseline, "baseline"},
		{VerticalAlignTop, "top"},
		{VerticalAlignMiddle, "middle"},
		{VerticalAlignBottom, "bottom"},
		{VerticalAlignSuper, "super"},
		{VerticalAlignSub, "sub"},
		{VerticalAlignTextTop, "text-top"},
		{VerticalAlignTextBottom, "text-bottom"},
		{VerticalAlignType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == VerticalAlignType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestTableLayoutType_String(t *testing.T) {
	testCases := []struct {
		value    TableLayoutType
		expected string
	}{
		{TableLayoutAuto, "auto"},
		{TableLayoutFixed, "fixed"},
		{TableLayoutType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == TableLayoutType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestBorderCollapseType_String(t *testing.T) {
	testCases := []struct {
		value    BorderCollapseType
		expected string
	}{
		{BorderCollapseSeparate, "separate"},
		{BorderCollapseCollapse, "collapse"},
		{BorderCollapseType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == BorderCollapseType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestListStyleType_String(t *testing.T) {
	testCases := []struct {
		value    ListStyleType
		expected string
	}{
		{ListStyleTypeDisc, "disc"},
		{ListStyleTypeCircle, "circle"},
		{ListStyleTypeSquare, "square"},
		{ListStyleTypeDecimal, "decimal"},
		{ListStyleTypeNone, "none"},
		{ListStyleType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == ListStyleType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestTextOverflowType_String(t *testing.T) {
	testCases := []struct {
		value    TextOverflowType
		expected string
	}{
		{TextOverflowClip, "clip"},
		{TextOverflowEllipsis, "ellipsis"},
		{TextOverflowType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == TextOverflowType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestUnicodeBidiType_String(t *testing.T) {
	testCases := []struct {
		value    UnicodeBidiType
		expected string
	}{
		{UnicodeBidiNormal, "normal"},
		{UnicodeBidiEmbed, "embed"},
		{UnicodeBidiIsolate, "isolate"},
		{UnicodeBidiBidiOverride, "bidi-override"},
		{UnicodeBidiIsolateOverride, "isolate-override"},
		{UnicodeBidiPlaintext, "plaintext"},
		{UnicodeBidiType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == UnicodeBidiType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}

func TestColumnFillType_String(t *testing.T) {
	testCases := []struct {
		value    ColumnFillType
		expected string
	}{
		{ColumnFillBalance, "balance"},
		{ColumnFillAuto, "auto"},
		{ColumnFillType(99), "unknown"},
	}
	for _, tc := range testCases {
		name := tc.expected
		if tc.value == ColumnFillType(99) {
			name = "out of range"
		}
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.value.String())
		})
	}
}
