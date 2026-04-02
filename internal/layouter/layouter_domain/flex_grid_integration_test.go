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

func TestFlexLayout_RowReverse(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRowReverse
	flex.Style.Width = DimensionPt(300)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(100)
	child1.Style.Height = DimensionPt(50)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(100)
	child2.Style.Height = DimensionPt(50)

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child1.ContentX > child2.ContentX,
		"row-reverse: child1 X (%f) should be > child2 X (%f)",
		child1.ContentX, child2.ContentX)
	assert.InDelta(t, 100, child1.ContentWidth, 0.001)
	assert.InDelta(t, 100, child2.ContentWidth, 0.001)
}

func TestFlexLayout_ColumnReverse(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionColumnReverse
	flex.Style.Width = DimensionPt(300)
	flex.Style.Height = DimensionPt(200)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(100)
	child1.Style.Height = DimensionPt(50)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(100)
	child2.Style.Height = DimensionPt(50)

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child1.ContentY > child2.ContentY,
		"column-reverse: child1 Y (%f) should be > child2 Y (%f)",
		child1.ContentY, child2.ContentY)
}

func TestFlexLayout_ColumnDirection(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionColumn
	flex.Style.Width = DimensionPt(300)
	flex.Style.Height = DimensionPt(200)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(200)
	child1.Style.Height = DimensionPt(60)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(200)
	child2.Style.Height = DimensionPt(40)

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 0, child1.ContentY, 0.001, "first column child Y should be 0")
	assert.True(t, child2.ContentY >= child1.ContentY+child1.ContentHeight,
		"child2 Y (%f) should be below child1 bottom (%f)",
		child2.ContentY, child1.ContentY+child1.ContentHeight)

	assert.InDelta(t, 200, flex.ContentHeight, 0.001,
		"column container height should match the explicit height")
}

func TestFlexLayout_FlexGrow(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(300)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Height = DimensionPt(50)
	child1.Style.FlexGrow = 1
	child1.Style.FlexBasis = DimensionPt(0)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Height = DimensionPt(50)
	child2.Style.FlexGrow = 2
	child2.Style.FlexBasis = DimensionPt(0)

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 100, child1.ContentWidth, 1.0,
		"flex-grow 1/3 of 300 should be ~100")
	assert.InDelta(t, 200, child2.ContentWidth, 1.0,
		"flex-grow 2/3 of 300 should be ~200")
}

func TestFlexLayout_FlexShrink(t *testing.T) {
	root := makeRoot(200)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(200)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(150)
	child1.Style.Height = DimensionPt(50)
	child1.Style.FlexShrink = 1

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(150)
	child2.Style.Height = DimensionPt(50)
	child2.Style.FlexShrink = 1

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	total_width := child1.ContentWidth + child2.ContentWidth
	assert.InDelta(t, 200, total_width, 1.0,
		"total width of shrunk items should be ~200")
	assert.True(t, child1.ContentWidth < 150,
		"child1 should have shrunk from 150")
	assert.True(t, child2.ContentWidth < 150,
		"child2 should have shrunk from 150")
}

func TestFlexLayout_FlexBasis(t *testing.T) {
	root := makeRoot(400)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(400)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(50)
	child1.Style.Height = DimensionPt(50)
	child1.Style.FlexBasis = DimensionPt(120)
	child1.Style.FlexGrow = 0

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(100)
	child2.Style.Height = DimensionPt(50)
	child2.Style.FlexGrow = 0

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 120, child1.ContentWidth, 1.0,
		"flex-basis should override width")
	assert.InDelta(t, 100, child2.ContentWidth, 1.0,
		"child2 should retain its explicit width")
}

func TestFlexLayout_AlignItemsStretch(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(300)
	flex.Style.AlignItems = AlignItemsStretch

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(100)
	child1.Style.Height = DimensionPt(100)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(100)
	child2.Style.Height = DimensionAuto()

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child2.ContentHeight >= 100,
		"stretched child height (%f) should be >= 100", child2.ContentHeight)
}

func TestFlexLayout_AlignItemsCentre(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(300)
	flex.Style.AlignItems = AlignItemsCentre

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(100)
	child1.Style.Height = DimensionPt(100)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(100)
	child2.Style.Height = DimensionPt(40)

	flex.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child2.ContentY > child1.ContentY,
		"centred child Y (%f) should be > first child Y (%f)",
		child2.ContentY, child1.ContentY)
}

func TestFlexLayout_FlexWrap(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.FlexWrap = FlexWrapWrap
	flex.Style.Width = DimensionPt(300)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child1.Style.Width = DimensionPt(150)
	child1.Style.Height = DimensionPt(50)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child2.Style.Width = DimensionPt(150)
	child2.Style.Height = DimensionPt(50)

	child3 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child3.Style.Width = DimensionPt(150)
	child3.Style.Height = DimensionPt(50)

	flex.Children = []*LayoutBox{child1, child2, child3}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, child1.ContentY, child2.ContentY, 0.001,
		"first two items should be on the same line")

	assert.True(t, child3.ContentY > child1.ContentY,
		"wrapped child Y (%f) should be > first line Y (%f)",
		child3.ContentY, child1.ContentY)
}

func TestFlexLayout_IntrinsicSizeItem(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(300)

	child := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child.Style.Width = DimensionAuto()
	child.Style.Height = DimensionAuto()

	text_box := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: child,
		Text:   "Hello",
	}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12

	child.Children = []*LayoutBox{text_box}
	flex.Children = []*LayoutBox{child}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child.ContentWidth > 0,
		"intrinsic flex item should have positive width")
	assert.True(t, child.ContentHeight > 0,
		"intrinsic flex item should have positive height")
}

func TestFlexLayout_ExplicitContainerHeight(t *testing.T) {
	root := makeRoot(300)

	flex := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root}
	flex.Style.Display = DisplayFlex
	flex.Style.FlexDirection = FlexDirectionRow
	flex.Style.Width = DimensionPt(300)
	flex.Style.Height = DimensionPt(200)

	child := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flex}
	child.Style.Width = DimensionPt(100)
	child.Style.Height = DimensionPt(50)

	flex.Children = []*LayoutBox{child}
	root.Children = []*LayoutBox{flex}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 200, flex.ContentHeight, 0.001,
		"explicit container height should be 200")
}

func TestGridLayout_ColumnMajorAutoFlow(t *testing.T) {
	root := makeRoot(300)

	grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
	grid.Style.Display = DisplayGrid
	grid.Style.Width = DimensionPt(300)
	grid.Style.GridAutoFlow = GridAutoFlowColumn
	grid.Style.GridTemplateRows = []GridTrack{
		{Value: 50, Unit: GridTrackPoints},
		{Value: 50, Unit: GridTrackPoints},
	}
	grid.Style.GridTemplateColumns = []GridTrack{
		{Value: 150, Unit: GridTrackPoints},
		{Value: 150, Unit: GridTrackPoints},
	}

	item1 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item1.Style.Height = DimensionPt(50)

	item2 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item2.Style.Height = DimensionPt(50)

	grid.Children = []*LayoutBox{item1, item2}
	root.Children = []*LayoutBox{grid}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, item1.ContentX, item2.ContentX, 0.001,
		"column-major items should share the same column X")

	assert.True(t, item2.ContentY > item1.ContentY,
		"item2 Y (%f) should be below item1 Y (%f) in column-major flow",
		item2.ContentY, item1.ContentY)

	assert.True(t, item1.ContentWidth > 0, "item1 should have positive width")
	assert.True(t, item2.ContentWidth > 0, "item2 should have positive width")
}

func TestGridLayout_FractionTracks(t *testing.T) {
	root := makeRoot(300)

	grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
	grid.Style.Display = DisplayGrid
	grid.Style.Width = DimensionPt(300)
	grid.Style.GridTemplateColumns = []GridTrack{
		{Value: 1, Unit: GridTrackFr},
		{Value: 2, Unit: GridTrackFr},
	}

	item1 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item1.Style.Height = DimensionPt(50)

	item2 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item2.Style.Height = DimensionPt(50)

	grid.Children = []*LayoutBox{item1, item2}
	root.Children = []*LayoutBox{grid}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 100, item1.ContentWidth, 1.0,
		"1fr of 300pt should be ~100")
	assert.InDelta(t, 200, item2.ContentWidth, 1.0,
		"2fr of 300pt should be ~200")
}

func TestGridLayout_AutoTracks(t *testing.T) {
	root := makeRoot(500)

	grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
	grid.Style.Display = DisplayGrid
	grid.Style.Width = DimensionPt(500)
	grid.Style.GridTemplateColumns = []GridTrack{
		{Unit: GridTrackAuto},
		{Unit: GridTrackAuto},
	}

	item1 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item1.Style.Height = DimensionPt(50)

	text1 := &LayoutBox{
		Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: item1, Text: "Short",
	}
	text1.Style.Display = DisplayInline
	text1.Style.FontSize = 12
	item1.Children = []*LayoutBox{text1}

	item2 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item2.Style.Height = DimensionPt(50)
	text2 := &LayoutBox{
		Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: item2, Text: "A much longer text",
	}
	text2.Style.Display = DisplayInline
	text2.Style.FontSize = 12
	item2.Children = []*LayoutBox{text2}

	grid.Children = []*LayoutBox{item1, item2}
	root.Children = []*LayoutBox{grid}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, item1.ContentWidth > 0,
		"auto-tracked item1 should have positive width")
	assert.True(t, item2.ContentWidth > 0,
		"auto-tracked item2 should have positive width")
}

func TestGridLayout_ExplicitPlacement(t *testing.T) {
	root := makeRoot(300)

	grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
	grid.Style.Display = DisplayGrid
	grid.Style.Width = DimensionPt(300)
	grid.Style.GridTemplateColumns = []GridTrack{
		{Value: 100, Unit: GridTrackPoints},
		{Value: 100, Unit: GridTrackPoints},
		{Value: 100, Unit: GridTrackPoints},
	}
	grid.Style.GridTemplateRows = []GridTrack{
		{Value: 50, Unit: GridTrackPoints},
		{Value: 50, Unit: GridTrackPoints},
	}

	item1 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item1.Style.Height = DimensionPt(50)
	item1.Style.GridColumnStart = GridLine{Line: 3, IsAuto: false}
	item1.Style.GridColumnEnd = DefaultGridLine()
	item1.Style.GridRowStart = GridLine{Line: 2, IsAuto: false}
	item1.Style.GridRowEnd = DefaultGridLine()

	item2 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item2.Style.Height = DimensionPt(50)
	item2.Style.GridColumnStart = GridLine{Line: 1, IsAuto: false}
	item2.Style.GridColumnEnd = DefaultGridLine()
	item2.Style.GridRowStart = GridLine{Line: 1, IsAuto: false}
	item2.Style.GridRowEnd = DefaultGridLine()

	grid.Children = []*LayoutBox{item1, item2}
	root.Children = []*LayoutBox{grid}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, item1.ContentX > item2.ContentX,
		"explicitly placed item1 X (%f) should be > item2 X (%f)",
		item1.ContentX, item2.ContentX)
	assert.True(t, item1.ContentY > item2.ContentY,
		"explicitly placed item1 Y (%f) should be > item2 Y (%f)",
		item1.ContentY, item2.ContentY)
}

func TestGridLayout_JustifyAlignItems(t *testing.T) {
	tests := []struct {
		name          string
		justify_items JustifyItemsType
		align_items   AlignItemsType
		expect_x_gt_0 bool
		expect_y_gt_0 bool
	}{
		{
			name:          "centre/centre",
			justify_items: JustifyItemsCentre,
			align_items:   AlignItemsCentre,
			expect_x_gt_0: true,
			expect_y_gt_0: true,
		},
		{
			name:          "end/flex-end",
			justify_items: JustifyItemsEnd,
			align_items:   AlignItemsFlexEnd,
			expect_x_gt_0: true,
			expect_y_gt_0: true,
		},
		{
			name:          "start/flex-start",
			justify_items: JustifyItemsStart,
			align_items:   AlignItemsFlexStart,
			expect_x_gt_0: false,
			expect_y_gt_0: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := makeRoot(300)

			grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
			grid.Style.Display = DisplayGrid
			grid.Style.Width = DimensionPt(300)
			grid.Style.JustifyItems = tt.justify_items
			grid.Style.AlignItems = tt.align_items
			grid.Style.GridTemplateColumns = []GridTrack{
				{Value: 300, Unit: GridTrackPoints},
			}
			grid.Style.GridTemplateRows = []GridTrack{
				{Value: 200, Unit: GridTrackPoints},
			}

			item := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
			item.Style.Width = DimensionPt(100)
			item.Style.Height = DimensionPt(50)

			grid.Children = []*LayoutBox{item}
			root.Children = []*LayoutBox{grid}

			require.NotPanics(t, func() { runLayout(root) })

			if tt.expect_x_gt_0 {
				assert.True(t, item.ContentX > 0,
					"item X (%f) should be > 0 for justify-items: %s",
					item.ContentX, tt.name)
			} else {
				assert.InDelta(t, 0, item.ContentX, 0.001,
					"item X should be 0 for justify-items: %s", tt.name)
			}

			if tt.expect_y_gt_0 {
				assert.True(t, item.ContentY > 0,
					"item Y (%f) should be > 0 for align-items: %s",
					item.ContentY, tt.name)
			} else {
				assert.InDelta(t, 0, item.ContentY, 0.001,
					"item Y should be 0 for align-items: %s", tt.name)
			}
		})
	}
}

func TestGridLayout_RowGapColumnGap(t *testing.T) {
	root := makeRoot(400)

	grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
	grid.Style.Display = DisplayGrid
	grid.Style.Width = DimensionPt(400)
	grid.Style.ColumnGap = 20
	grid.Style.RowGap = 10
	grid.Style.GridTemplateColumns = []GridTrack{
		{Value: 190, Unit: GridTrackPoints},
		{Value: 190, Unit: GridTrackPoints},
	}
	grid.Style.GridTemplateRows = []GridTrack{
		{Value: 50, Unit: GridTrackPoints},
		{Value: 50, Unit: GridTrackPoints},
	}

	item1 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item1.Style.Height = DimensionPt(50)

	item2 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item2.Style.Height = DimensionPt(50)

	item3 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item3.Style.Height = DimensionPt(50)

	item4 := &LayoutBox{Type: BoxGridItem, Style: DefaultComputedStyle(), Parent: grid}
	item4.Style.Height = DimensionPt(50)

	grid.Children = []*LayoutBox{item1, item2, item3, item4}
	root.Children = []*LayoutBox{grid}

	require.NotPanics(t, func() { runLayout(root) })

	column_gap_distance := item2.ContentX - item1.ContentX
	assert.True(t, column_gap_distance > 190,
		"column gap should push item2 X (%f) away from item1 X (%f) by more than 190",
		item2.ContentX, item1.ContentX)

	row_gap_distance := item3.ContentY - item1.ContentY
	assert.True(t, row_gap_distance > 50,
		"row gap should push item3 Y (%f) away from item1 Y (%f) by more than 50",
		item3.ContentY, item1.ContentY)
}
