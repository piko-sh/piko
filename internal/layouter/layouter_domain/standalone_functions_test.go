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
	"github.com/stretchr/testify/require"
)

const standaloneEpsilon = 0.001

func TestLayoutTextRun_BasicText(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "Hello",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	frag := layoutTextRun(box, font_metrics)

	require.NotNil(t, frag)
	assert.InDelta(t, 30.0, frag.ContentWidth, standaloneEpsilon)
	assert.InDelta(t, 13.2, frag.ContentHeight, standaloneEpsilon)
	assert.Same(t, box, frag.Box)
}

func TestLayoutTextRun_EmptyText(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	frag := layoutTextRun(box, font_metrics)

	require.NotNil(t, frag)
	assert.InDelta(t, 0.0, frag.ContentWidth, standaloneEpsilon)

	assert.InDelta(t, 13.2, frag.ContentHeight, standaloneEpsilon)
}

func TestLayoutTextRun_ExplicitLineHeight(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "Hello",
		Style: ComputedStyle{
			FontSize:   12,
			LineHeight: 20,
		},
	}

	frag := layoutTextRun(box, font_metrics)

	require.NotNil(t, frag)
	assert.InDelta(t, 30.0, frag.ContentWidth, standaloneEpsilon)
	assert.InDelta(t, 20.0, frag.ContentHeight, standaloneEpsilon)
}

func TestLayoutTextRun_GlyphsSet(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "AB",
		Style: ComputedStyle{
			FontSize: 10,
		},
	}

	_ = layoutTextRun(box, font_metrics)

	require.NotEmpty(t, box.Glyphs)
	assert.Len(t, box.Glyphs, 2)

	assert.InDelta(t, 5.0, box.Glyphs[0].XAdvance, standaloneEpsilon)
	assert.InDelta(t, 5.0, box.Glyphs[1].XAdvance, standaloneEpsilon)
}

func TestMeasureIntrinsicWidth_TextRunMinContent(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "Hello world",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	result := measureIntrinsicWidth(box, SizingModeMinContent, font_metrics)
	assert.InDelta(t, 30.0, result, standaloneEpsilon)
}

func TestMeasureIntrinsicWidth_TextRunMaxContent(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxTextRun,
		Text: "Hello world",
		Style: ComputedStyle{
			FontSize: 12,
		},
	}

	result := measureIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 66.0, result, standaloneEpsilon)
}

func TestMeasureIntrinsicWidth_Block(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	child_a := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "short",
		Style: ComputedStyle{FontSize: 12},
	}
	child_b := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "longer text here",
		Style: ComputedStyle{FontSize: 12},
	}
	box := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{child_a, child_b},
		Style: ComputedStyle{
			Width:        DimensionAuto(),
			PaddingLeft:  4,
			PaddingRight: 4,
		},
	}

	result := measureIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 104.0, result, standaloneEpsilon)
}

func TestMeasureIntrinsicWidth_BlockWithExplicitWidth(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxBlock,
		Style: ComputedStyle{
			Width:            DimensionPt(200),
			BoxSizing:        BoxSizingContentBox,
			PaddingLeft:      10,
			PaddingRight:     10,
			BorderLeftWidth:  2,
			BorderRightWidth: 2,
		},
	}

	result := measureIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 224.0, result, standaloneEpsilon)
}

func TestMeasureMinContentWidth(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "Hello world",
		Style: ComputedStyle{FontSize: 12},
	}

	result := measureMinContentWidth(box, font_metrics)
	assert.InDelta(t, 30.0, result, standaloneEpsilon)
}

func TestMeasureMaxContentWidth(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "Hello world",
		Style: ComputedStyle{FontSize: 12},
	}

	result := measureMaxContentWidth(box, font_metrics)
	assert.InDelta(t, 66.0, result, standaloneEpsilon)
}

func TestMeasureBlockIntrinsicWidth_AutoWidthMultipleChildren(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	child_a := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abc",
		Style: ComputedStyle{FontSize: 10},
	}
	child_b := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abcdef",
		Style: ComputedStyle{FontSize: 10},
	}
	box := &LayoutBox{
		Type:     BoxBlock,
		Children: []*LayoutBox{child_a, child_b},
		Style: ComputedStyle{
			Width:            DimensionAuto(),
			PaddingLeft:      3,
			PaddingRight:     3,
			BorderLeftWidth:  1,
			BorderRightWidth: 1,
		},
	}

	result := measureBlockIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 38.0, result, standaloneEpsilon)
}

func TestMeasureBlockIntrinsicWidth_ExplicitWidthContentBox(t *testing.T) {
	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxBlock,
		Style: ComputedStyle{
			Width:            DimensionPt(100),
			BoxSizing:        BoxSizingContentBox,
			PaddingLeft:      5,
			PaddingRight:     5,
			BorderLeftWidth:  1,
			BorderRightWidth: 1,
		},
	}

	result := measureBlockIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 112.0, result, standaloneEpsilon)
}

func TestMeasureBlockIntrinsicWidth_ExplicitWidthBorderBox(t *testing.T) {
	font_metrics := &mockFontMetrics{}
	box := &LayoutBox{
		Type: BoxBlock,
		Style: ComputedStyle{
			Width:            DimensionPt(100),
			BoxSizing:        BoxSizingBorderBox,
			PaddingLeft:      5,
			PaddingRight:     5,
			BorderLeftWidth:  1,
			BorderRightWidth: 1,
		},
	}

	result := measureBlockIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 100.0, result, standaloneEpsilon)
}

func TestMeasureFlexIntrinsicWidth_RowMaxContent(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	child_a := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abc",
		Style: ComputedStyle{FontSize: 10},
	}
	child_b := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "de",
		Style: ComputedStyle{FontSize: 10},
	}
	child_c := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "f",
		Style: ComputedStyle{FontSize: 10},
	}
	box := &LayoutBox{
		Type:     BoxFlex,
		Children: []*LayoutBox{child_a, child_b, child_c},
		Style: ComputedStyle{
			FlexDirection: FlexDirectionRow,
			ColumnGap:     8,
			PaddingLeft:   2,
			PaddingRight:  2,
		},
	}

	result := measureFlexIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 50.0, result, standaloneEpsilon)
}

func TestMeasureFlexIntrinsicWidth_RowMinContent(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	child_a := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abc",
		Style: ComputedStyle{FontSize: 10},
	}
	child_b := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abcdef",
		Style: ComputedStyle{FontSize: 10},
	}
	box := &LayoutBox{
		Type:     BoxFlex,
		Children: []*LayoutBox{child_a, child_b},
		Style: ComputedStyle{
			FlexDirection: FlexDirectionRow,
			ColumnGap:     10,
			PaddingLeft:   0,
			PaddingRight:  0,
		},
	}

	result := measureFlexIntrinsicWidth(box, SizingModeMinContent, font_metrics)
	assert.InDelta(t, 30.0, result, standaloneEpsilon)
}

func TestMeasureFlexIntrinsicWidth_Column(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	child_a := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abc",
		Style: ComputedStyle{FontSize: 10},
	}
	child_b := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abcdef",
		Style: ComputedStyle{FontSize: 10},
	}
	box := &LayoutBox{
		Type:     BoxFlex,
		Children: []*LayoutBox{child_a, child_b},
		Style: ComputedStyle{
			FlexDirection: FlexDirectionColumn,
		},
	}

	result_max := measureFlexIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 30.0, result_max, standaloneEpsilon)

	result_min := measureFlexIntrinsicWidth(box, SizingModeMinContent, font_metrics)
	assert.InDelta(t, 30.0, result_min, standaloneEpsilon)
}

func TestMeasureFlexIntrinsicWidth_WithPaddingAndBorder(t *testing.T) {

	font_metrics := &mockFontMetrics{}
	child := &LayoutBox{
		Type:  BoxTextRun,
		Text:  "abc",
		Style: ComputedStyle{FontSize: 10},
	}
	box := &LayoutBox{
		Type:     BoxFlex,
		Children: []*LayoutBox{child},
		Style: ComputedStyle{
			FlexDirection:    FlexDirectionRow,
			PaddingLeft:      5,
			PaddingRight:     5,
			BorderLeftWidth:  2,
			BorderRightWidth: 2,
		},
	}

	result := measureFlexIntrinsicWidth(box, SizingModeMaxContent, font_metrics)
	assert.InDelta(t, 29.0, result, standaloneEpsilon)
}

func TestSplitIntoWords_Basic(t *testing.T) {
	result := splitIntoWords("hello world")
	assert.Equal(t, []string{"hello", "world"}, result)
}

func TestSplitIntoWords_MultipleSpaces(t *testing.T) {
	result := splitIntoWords("hello   world")
	assert.Equal(t, []string{"hello", "world"}, result)
}

func TestSplitIntoWords_EmptyString(t *testing.T) {
	result := splitIntoWords("")
	assert.Empty(t, result)
}

func TestSplitIntoWords_SingleWord(t *testing.T) {
	result := splitIntoWords("hello")
	assert.Equal(t, []string{"hello"}, result)
}

func TestExpandSoftHyphens_WithSoftHyphen(t *testing.T) {
	result := expandSoftHyphens([]string{"hel\u00ADlo"})
	assert.Equal(t, []string{"hel\u00AD", "lo"}, result)
}

func TestExpandSoftHyphens_WithoutSoftHyphen(t *testing.T) {
	result := expandSoftHyphens([]string{"hello"})
	assert.Equal(t, []string{"hello"}, result)
}

func TestExpandSoftHyphens_MultipleSoftHyphens(t *testing.T) {
	result := expandSoftHyphens([]string{"a\u00ADb\u00ADc"})
	assert.Equal(t, []string{"a\u00AD", "b\u00AD", "c"}, result)
}

func TestAllowsWrapping_Normal(t *testing.T) {
	assert.True(t, allowsWrapping(WhiteSpaceNormal))
}

func TestAllowsWrapping_Nowrap(t *testing.T) {
	assert.False(t, allowsWrapping(WhiteSpaceNowrap))
}

func TestAllowsWrapping_Pre(t *testing.T) {
	assert.False(t, allowsWrapping(WhiteSpacePre))
}

func TestAllowsWrapping_PreWrap(t *testing.T) {
	assert.True(t, allowsWrapping(WhiteSpacePreWrap))
}

func TestAllowsWrapping_PreLine(t *testing.T) {
	assert.True(t, allowsWrapping(WhiteSpacePreLine))
}

func TestPreservesWhitespace_Normal(t *testing.T) {
	assert.False(t, preservesWhitespace(WhiteSpaceNormal))
}

func TestPreservesWhitespace_Pre(t *testing.T) {
	assert.True(t, preservesWhitespace(WhiteSpacePre))
}

func TestPreservesWhitespace_PreWrap(t *testing.T) {
	assert.True(t, preservesWhitespace(WhiteSpacePreWrap))
}

func TestPreservesWhitespace_PreLine(t *testing.T) {
	assert.True(t, preservesWhitespace(WhiteSpacePreLine))
}

func TestPreservesWhitespace_Nowrap(t *testing.T) {
	assert.False(t, preservesWhitespace(WhiteSpaceNowrap))
}

func TestApplyJustifyToLine_ThreeItemsWithFreeSpace(t *testing.T) {

	items := []lineItem{
		{fragment: &Fragment{OffsetX: 0, ContentWidth: 30}, x: 0, width: 30},
		{fragment: &Fragment{OffsetX: 30, ContentWidth: 30}, x: 30, width: 30},
		{fragment: &Fragment{OffsetX: 60, ContentWidth: 30}, x: 60, width: 30},
	}
	line := lineBox{items: items, width: 90}

	applyJustifyToLine(line, 0, 3, 200, 0)

	assert.InDelta(t, 0.0, items[0].fragment.OffsetX, standaloneEpsilon)
	assert.InDelta(t, 85.0, items[0].fragment.ContentWidth, standaloneEpsilon)

	assert.InDelta(t, 85.0, items[1].fragment.OffsetX, standaloneEpsilon)
	assert.InDelta(t, 85.0, items[1].fragment.ContentWidth, standaloneEpsilon)

	assert.InDelta(t, 170.0, items[2].fragment.OffsetX, standaloneEpsilon)
	assert.InDelta(t, 30.0, items[2].fragment.ContentWidth, standaloneEpsilon)
}

func TestApplyJustifyToLine_LastLineSkipped(t *testing.T) {

	items := []lineItem{
		{fragment: &Fragment{OffsetX: 0, ContentWidth: 50}, x: 0, width: 50},
	}
	line := lineBox{items: items, width: 50}

	applyJustifyToLine(line, 2, 3, 200, 0)

	assert.InDelta(t, 0.0, items[0].fragment.OffsetX, standaloneEpsilon)
	assert.InDelta(t, 50.0, items[0].fragment.ContentWidth, standaloneEpsilon)
}

func TestApplyJustifyToLine_SingleItem(t *testing.T) {

	items := []lineItem{
		{fragment: &Fragment{OffsetX: 0, ContentWidth: 50}, x: 0, width: 50},
	}
	line := lineBox{items: items, width: 50}

	applyJustifyToLine(line, 0, 2, 200, 0)

	assert.InDelta(t, 200.0, items[0].fragment.ContentWidth, standaloneEpsilon)
}

func TestApplyOffsetToLine_Centre(t *testing.T) {
	items := []lineItem{
		{fragment: &Fragment{OffsetX: 0}, x: 0, width: 40},
		{fragment: &Fragment{OffsetX: 40}, x: 40, width: 60},
	}
	line := lineBox{items: items, width: 100}

	applyOffsetToLine(line, TextAlignCentre, 200, 0)

	assert.InDelta(t, 50.0, items[0].fragment.OffsetX, standaloneEpsilon)
	assert.InDelta(t, 90.0, items[1].fragment.OffsetX, standaloneEpsilon)
}

func TestApplyOffsetToLine_Right(t *testing.T) {
	items := []lineItem{
		{fragment: &Fragment{OffsetX: 0}, x: 0, width: 40},
		{fragment: &Fragment{OffsetX: 40}, x: 40, width: 60},
	}
	line := lineBox{items: items, width: 100}

	applyOffsetToLine(line, TextAlignRight, 200, 0)

	assert.InDelta(t, 100.0, items[0].fragment.OffsetX, standaloneEpsilon)
	assert.InDelta(t, 140.0, items[1].fragment.OffsetX, standaloneEpsilon)
}

func TestApplyInlineVerticalAlign_Baseline(t *testing.T) {
	items := []lineItem{
		{
			fragment: &Fragment{
				Box:     &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignBaseline}},
				OffsetY: 0,
				Margin:  BoxEdges{Top: 5},
			},
		},
		{
			fragment: &Fragment{
				Box:     &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignBaseline}},
				OffsetY: 0,
				Margin:  BoxEdges{Top: 10},
			},
		},
	}

	applyInlineVerticalAlign(items, 50)

	assert.InDelta(t, 5.0, items[0].fragment.OffsetY, standaloneEpsilon)
	assert.InDelta(t, 0.0, items[1].fragment.OffsetY, standaloneEpsilon)
}

func TestApplyInlineVerticalAlign_Top(t *testing.T) {
	items := []lineItem{
		{
			fragment: &Fragment{
				Box:     &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignTop}},
				OffsetY: 0,
			},
		},
	}

	applyInlineVerticalAlign(items, 50)

	assert.InDelta(t, 0.0, items[0].fragment.OffsetY, standaloneEpsilon)
}

func TestApplyInlineVerticalAlign_Middle(t *testing.T) {
	items := []lineItem{
		{
			fragment: &Fragment{
				Box:           &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignMiddle}},
				OffsetY:       0,
				ContentHeight: 20,
			},
		},
	}

	applyInlineVerticalAlign(items, 50)

	assert.InDelta(t, 15.0, items[0].fragment.OffsetY, standaloneEpsilon)
}

func TestApplyInlineVerticalAlign_Bottom(t *testing.T) {
	items := []lineItem{
		{
			fragment: &Fragment{
				Box:           &LayoutBox{Style: ComputedStyle{VerticalAlign: VerticalAlignBottom}},
				OffsetY:       0,
				ContentHeight: 20,
			},
		},
	}

	applyInlineVerticalAlign(items, 50)

	assert.InDelta(t, 30.0, items[0].fragment.OffsetY, standaloneEpsilon)
}

func TestFragmentIntoColumns_SplitsAcrossColumns(t *testing.T) {

	children := []*Fragment{
		{ContentHeight: 30},
		{ContentHeight: 30},
		{ContentHeight: 30},
	}

	columns := fragmentIntoColumns(children, 55)

	require.Len(t, columns, 3)
	assert.Len(t, columns[0], 1)
	assert.Len(t, columns[1], 1)
	assert.Len(t, columns[2], 1)
}

func TestFragmentIntoColumns_TwoFitPerColumn(t *testing.T) {

	children := []*Fragment{
		{ContentHeight: 30},
		{ContentHeight: 30},
		{ContentHeight: 30},
	}

	columns := fragmentIntoColumns(children, 65)

	require.Len(t, columns, 2)
	assert.Len(t, columns[0], 2)
	assert.Len(t, columns[1], 1)
}

func TestFragmentIntoColumns_EmptyChildren(t *testing.T) {
	columns := fragmentIntoColumns(nil, 100)
	assert.Nil(t, columns)
}

func TestFragmentIntoColumns_ZeroColumnHeight(t *testing.T) {
	children := []*Fragment{{ContentHeight: 30}}
	columns := fragmentIntoColumns(children, 0)
	assert.Nil(t, columns)
}

func TestFragmentIntoColumns_SingleFragmentFits(t *testing.T) {
	children := []*Fragment{{ContentHeight: 30}}
	columns := fragmentIntoColumns(children, 100)

	require.Len(t, columns, 1)
	assert.Len(t, columns[0], 1)
}

func TestFragmentIntoColumns_OffsetsAreAdjusted(t *testing.T) {

	children := []*Fragment{
		{ContentHeight: 20},
		{ContentHeight: 25},
		{ContentHeight: 15},
	}

	columns := fragmentIntoColumns(children, 100)

	require.Len(t, columns, 1)
	require.Len(t, columns[0], 3)
	assert.InDelta(t, 0.0, columns[0][0].OffsetY, standaloneEpsilon)
	assert.InDelta(t, 20.0, columns[0][1].OffsetY, standaloneEpsilon)
	assert.InDelta(t, 45.0, columns[0][2].OffsetY, standaloneEpsilon)
}

func TestPositionColumns_TwoColumnsWithGap(t *testing.T) {
	columns := [][]*Fragment{
		{{ContentWidth: 100, ContentHeight: 50}},
		{{ContentWidth: 100, ContentHeight: 50}},
	}

	result := positionColumns(columns, 100, 10)

	require.Len(t, result, 2)
	assert.InDelta(t, 0.0, result[0].OffsetX, standaloneEpsilon)
	assert.InDelta(t, 110.0, result[1].OffsetX, standaloneEpsilon)
}

func TestPositionColumns_ThreeColumnsNoGap(t *testing.T) {
	columns := [][]*Fragment{
		{{ContentWidth: 80, ContentHeight: 40}},
		{{ContentWidth: 80, ContentHeight: 40}},
		{{ContentWidth: 80, ContentHeight: 40}},
	}

	result := positionColumns(columns, 80, 0)

	require.Len(t, result, 3)
	assert.InDelta(t, 0.0, result[0].OffsetX, standaloneEpsilon)
	assert.InDelta(t, 80.0, result[1].OffsetX, standaloneEpsilon)
	assert.InDelta(t, 160.0, result[2].OffsetX, standaloneEpsilon)
}

func TestResolveColumnHeight_Balanced(t *testing.T) {
	style := &ComputedStyle{Height: DimensionAuto()}
	result := resolveColumnHeight(200, 2, ColumnFillBalance, 0, style)
	assert.InDelta(t, 100.0, result, standaloneEpsilon)
}

func TestResolveColumnHeight_ExplicitContainerHeight(t *testing.T) {
	style := &ComputedStyle{
		Height:    DimensionPt(150),
		BoxSizing: BoxSizingContentBox,
	}
	result := resolveColumnHeight(300, 3, ColumnFillBalance, 0, style)
	assert.InDelta(t, 150.0, result, standaloneEpsilon)
}

func TestResolveColumnHeight_AutoFillWithAvailableBlockSize(t *testing.T) {
	style := &ComputedStyle{Height: DimensionAuto()}
	result := resolveColumnHeight(300, 3, ColumnFillAuto, 250, style)
	assert.InDelta(t, 250.0, result, standaloneEpsilon)
}

func TestHalveTableCellBorders(t *testing.T) {
	cell_a := &LayoutBox{
		Type: BoxTableCell,
		Style: ComputedStyle{
			BorderTopWidth:    4,
			BorderBottomWidth: 4,
			BorderLeftWidth:   4,
			BorderRightWidth:  4,
		},
	}
	cell_b := &LayoutBox{
		Type: BoxTableCell,
		Style: ComputedStyle{
			BorderTopWidth:    6,
			BorderBottomWidth: 8,
			BorderLeftWidth:   10,
			BorderRightWidth:  12,
		},
	}

	placements := []tableCellPlacement{
		{cell: cell_a, row: 0, column: 0, colspan: 1, rowspan: 1},
		{cell: cell_b, row: 0, column: 1, colspan: 1, rowspan: 1},
	}

	halveTableCellBorders(placements)

	assert.InDelta(t, 2.0, cell_a.Style.BorderTopWidth, standaloneEpsilon)
	assert.InDelta(t, 2.0, cell_a.Style.BorderBottomWidth, standaloneEpsilon)
	assert.InDelta(t, 2.0, cell_a.Style.BorderLeftWidth, standaloneEpsilon)
	assert.InDelta(t, 2.0, cell_a.Style.BorderRightWidth, standaloneEpsilon)

	assert.InDelta(t, 3.0, cell_b.Style.BorderTopWidth, standaloneEpsilon)
	assert.InDelta(t, 4.0, cell_b.Style.BorderBottomWidth, standaloneEpsilon)
	assert.InDelta(t, 5.0, cell_b.Style.BorderLeftWidth, standaloneEpsilon)
	assert.InDelta(t, 6.0, cell_b.Style.BorderRightWidth, standaloneEpsilon)
}

func TestCollapsedTableOuterBorder_TableBordersWin(t *testing.T) {

	table_box := &LayoutBox{
		Type: BoxBlock,
		Style: ComputedStyle{
			BorderTopWidth:    5,
			BorderBottomWidth: 5,
			BorderLeftWidth:   5,
			BorderRightWidth:  5,
		},
	}
	cell := &LayoutBox{
		Type: BoxTableCell,
		Style: ComputedStyle{
			BorderTopWidth:    2,
			BorderBottomWidth: 2,
			BorderLeftWidth:   2,
			BorderRightWidth:  2,
		},
	}
	placements := []tableCellPlacement{
		{cell: cell, row: 0, column: 0, colspan: 1, rowspan: 1},
	}

	border := collapsedTableOuterBorder(table_box, placements, 1, 1)

	assert.InDelta(t, 5.0, border.Top, standaloneEpsilon)
	assert.InDelta(t, 5.0, border.Bottom, standaloneEpsilon)
	assert.InDelta(t, 5.0, border.Left, standaloneEpsilon)
	assert.InDelta(t, 5.0, border.Right, standaloneEpsilon)
}

func TestCollapsedTableOuterBorder_CellBordersWin(t *testing.T) {

	table_box := &LayoutBox{
		Type: BoxBlock,
		Style: ComputedStyle{
			BorderTopWidth:    1,
			BorderBottomWidth: 1,
			BorderLeftWidth:   1,
			BorderRightWidth:  1,
		},
	}
	cell := &LayoutBox{
		Type: BoxTableCell,
		Style: ComputedStyle{
			BorderTopWidth:    4,
			BorderBottomWidth: 4,
			BorderLeftWidth:   4,
			BorderRightWidth:  4,
		},
	}
	placements := []tableCellPlacement{
		{cell: cell, row: 0, column: 0, colspan: 1, rowspan: 1},
	}

	border := collapsedTableOuterBorder(table_box, placements, 1, 1)

	assert.InDelta(t, 4.0, border.Top, standaloneEpsilon)
	assert.InDelta(t, 4.0, border.Bottom, standaloneEpsilon)
	assert.InDelta(t, 4.0, border.Left, standaloneEpsilon)
	assert.InDelta(t, 4.0, border.Right, standaloneEpsilon)
}

func TestCollapsedTableOuterBorder_InteriorCellDoesNotAffectOuterBorder(t *testing.T) {

	table_box := &LayoutBox{
		Type: BoxBlock,
		Style: ComputedStyle{
			BorderTopWidth:    1,
			BorderBottomWidth: 1,
			BorderLeftWidth:   1,
			BorderRightWidth:  1,
		},
	}

	corner_cell := &LayoutBox{
		Type: BoxTableCell,
		Style: ComputedStyle{
			BorderTopWidth:    2,
			BorderBottomWidth: 2,
			BorderLeftWidth:   2,
			BorderRightWidth:  2,
		},
	}
	interior_cell := &LayoutBox{
		Type: BoxTableCell,
		Style: ComputedStyle{
			BorderTopWidth:    10,
			BorderBottomWidth: 10,
			BorderLeftWidth:   10,
			BorderRightWidth:  10,
		},
	}
	placements := []tableCellPlacement{

		{cell: corner_cell, row: 0, column: 0, colspan: 1, rowspan: 1},
		{cell: corner_cell, row: 0, column: 2, colspan: 1, rowspan: 1},
		{cell: corner_cell, row: 2, column: 0, colspan: 1, rowspan: 1},
		{cell: corner_cell, row: 2, column: 2, colspan: 1, rowspan: 1},

		{cell: interior_cell, row: 1, column: 1, colspan: 1, rowspan: 1},
	}

	border := collapsedTableOuterBorder(table_box, placements, 3, 3)

	assert.InDelta(t, 2.0, border.Top, standaloneEpsilon)
	assert.InDelta(t, 2.0, border.Bottom, standaloneEpsilon)
	assert.InDelta(t, 2.0, border.Left, standaloneEpsilon)
	assert.InDelta(t, 2.0, border.Right, standaloneEpsilon)
}
