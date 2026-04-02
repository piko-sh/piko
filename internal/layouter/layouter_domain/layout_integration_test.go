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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRoot(width float64) *LayoutBox {
	root := &LayoutBox{
		Type:         BoxBlock,
		ContentWidth: width,
	}
	root.Style = DefaultComputedStyle()
	root.Style.Display = DisplayBlock
	root.Style.Width = DimensionPt(width)
	root.Style.Height = DimensionAuto()
	root.Style.OverflowX = OverflowHidden
	root.Style.OverflowY = OverflowHidden
	return root
}

func runLayout(root *LayoutBox) {
	fm := &mockFontMetrics{}
	LayoutBoxTree(context.Background(), root, fm)
}

func TestLayoutBoxTree_SimpleBlock(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(200)
	child.Style.Height = DimensionPt(100)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 200, child.ContentWidth, 0.001, "child width should be 200pt")
	assert.InDelta(t, 100, child.ContentHeight, 0.001, "child height should be 100pt")
	assert.InDelta(t, 0, child.ContentX, 0.001, "child X should be 0 (left-aligned)")
	assert.InDelta(t, 0, child.ContentY, 0.001, "child Y should be 0 (first child)")
}

func TestLayoutBoxTree_AutoWidth(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionAuto()
	child.Style.Height = DimensionPt(50)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 500, child.ContentWidth, 0.001, "auto width should fill parent")
}

func TestLayoutBoxTree_PercentageWidth(t *testing.T) {
	root := makeRoot(400)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPct(50)
	child.Style.Height = DimensionPt(50)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 200, child.ContentWidth, 0.001, "50%% of 400 should be 200")
}

func TestLayoutBoxTree_Padding(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionAuto()
	child.Style.Height = DimensionPt(50)
	child.Style.PaddingLeft = 10
	child.Style.PaddingRight = 10
	child.Style.PaddingTop = 10
	child.Style.PaddingBottom = 10

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 480, child.ContentWidth, 0.001, "auto width minus padding should be 480")
	assert.InDelta(t, 10, child.Padding.Left, 0.001, "left padding should be 10")
	assert.InDelta(t, 10, child.Padding.Right, 0.001, "right padding should be 10")
}

func TestLayoutBoxTree_MultipleBlockChildren(t *testing.T) {
	root := makeRoot(500)

	child1 := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child1.Style.Display = DisplayBlock
	child1.Style.Width = DimensionPt(500)
	child1.Style.Height = DimensionPt(100)

	child2 := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child2.Style.Display = DisplayBlock
	child2.Style.Width = DimensionPt(500)
	child2.Style.Height = DimensionPt(100)

	root.Children = []*LayoutBox{child1, child2}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 0, child1.ContentY, 0.001, "first child Y should be 0")
	assert.InDelta(t, 100, child2.ContentY, 0.001, "second child Y should be 100 (stacked below first)")
}

func TestLayoutBoxTree_MarginCollapsing(t *testing.T) {
	root := makeRoot(500)

	child1 := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child1.Style.Display = DisplayBlock
	child1.Style.Width = DimensionPt(500)
	child1.Style.Height = DimensionPt(50)
	child1.Style.MarginBottom = DimensionPt(20)

	child2 := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child2.Style.Display = DisplayBlock
	child2.Style.Width = DimensionPt(500)
	child2.Style.Height = DimensionPt(50)
	child2.Style.MarginTop = DimensionPt(30)

	root.Children = []*LayoutBox{child1, child2}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 0, child1.ContentY, 0.001, "first child Y should be 0")
	assert.InDelta(t, 80, child2.ContentY, 0.001,
		"second child Y should be 50 (child1 height) + 30 (collapsed margin)")
}

func TestLayoutBoxTree_InlineContent(t *testing.T) {
	root := makeRoot(500)
	root.Style.Height = DimensionAuto()

	textBox := &LayoutBox{
		Type:   BoxTextRun,
		Style:  DefaultComputedStyle(),
		Parent: root,
		Text:   "Hello",
	}
	textBox.Style.Display = DisplayInline
	textBox.Style.FontSize = 12

	root.Children = []*LayoutBox{textBox}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, textBox.ContentWidth > 0, "text run should have positive width")
	assert.True(t, textBox.ContentHeight > 0, "text run should have positive height")
}

func TestLayoutBoxTree_FlexLayout(t *testing.T) {
	root := makeRoot(300)

	flexRoot := &LayoutBox{Type: BoxFlex, Style: DefaultComputedStyle(), Parent: root, ContentWidth: 300}
	flexRoot.Style.Display = DisplayFlex
	flexRoot.Style.FlexDirection = FlexDirectionRow
	flexRoot.Style.Width = DimensionPt(300)

	child1 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flexRoot}
	child1.Style.Width = DimensionPt(100)
	child1.Style.Height = DimensionPt(50)

	child2 := &LayoutBox{Type: BoxFlexItem, Style: DefaultComputedStyle(), Parent: flexRoot}
	child2.Style.Width = DimensionPt(100)
	child2.Style.Height = DimensionPt(50)

	flexRoot.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{flexRoot}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child1.ContentWidth > 0, "flex child1 should have positive width")
	assert.True(t, child2.ContentWidth > 0, "flex child2 should have positive width")

	assert.True(t, child2.ContentX > child1.ContentX,
		"child2 X (%f) should be greater than child1 X (%f)", child2.ContentX, child1.ContentX)
}

func TestLayoutBoxTree_NestedBlocks(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(400)
	child.Style.Height = DimensionAuto()
	child.Style.PaddingTop = 20
	child.Style.PaddingLeft = 20

	grandchild := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: child,
	}
	grandchild.Style.Display = DisplayBlock
	grandchild.Style.Width = DimensionPt(200)
	grandchild.Style.Height = DimensionPt(50)

	child.Children = []*LayoutBox{grandchild}
	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 200, grandchild.ContentWidth, 0.001, "grandchild width should be 200")
	assert.True(t, grandchild.ContentX >= child.ContentX,
		"grandchild X should be >= child content X")
	assert.True(t, grandchild.ContentY >= child.ContentY,
		"grandchild Y should be >= child content Y")
}

func TestLayoutBoxTree_BorderBox(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.BoxSizing = BoxSizingBorderBox
	child.Style.Width = DimensionPt(200)
	child.Style.Height = DimensionPt(100)
	child.Style.PaddingLeft = 20
	child.Style.PaddingRight = 20

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 160, child.ContentWidth, 0.001, "border-box content width should be 160")
}

func TestLayoutBoxTree_BorderAndPadding(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionAuto()
	child.Style.Height = DimensionPt(50)
	child.Style.PaddingLeft = 5
	child.Style.PaddingRight = 5
	child.Style.BorderLeftWidth = 2
	child.Style.BorderRightWidth = 2

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 486, child.ContentWidth, 0.001,
		"auto width minus padding and border should be 486")
}

func TestLayoutBoxTree_RelativePositioning(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(200)
	child.Style.Height = DimensionPt(50)
	child.Style.Position = PositionRelative
	child.Style.Left = DimensionPt(10)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 10, child.ContentX, 0.001,
		"relative positioning should offset X by 10pt")
}

func TestLayoutBoxTree_MinMaxWidth(t *testing.T) {
	t.Run("min-width", func(t *testing.T) {
		root := makeRoot(500)

		child := &LayoutBox{
			Type:   BoxBlock,
			Style:  DefaultComputedStyle(),
			Parent: root,
		}
		child.Style.Display = DisplayBlock
		child.Style.Width = DimensionAuto()
		child.Style.Height = DimensionPt(50)
		child.Style.MinWidth = DimensionPt(200)

		root.Children = []*LayoutBox{child}

		require.NotPanics(t, func() { runLayout(root) })

		assert.True(t, child.ContentWidth >= 200,
			"content width (%f) should be >= min-width 200", child.ContentWidth)
	})

	t.Run("max-width", func(t *testing.T) {
		root := makeRoot(500)

		child := &LayoutBox{
			Type:   BoxBlock,
			Style:  DefaultComputedStyle(),
			Parent: root,
		}
		child.Style.Display = DisplayBlock
		child.Style.Width = DimensionAuto()
		child.Style.Height = DimensionPt(50)
		child.Style.MaxWidth = DimensionPt(100)

		root.Children = []*LayoutBox{child}

		require.NotPanics(t, func() { runLayout(root) })

		assert.True(t, child.ContentWidth <= 100,
			"content width (%f) should be <= max-width 100", child.ContentWidth)
	})
}

func TestLayoutBoxTree_FixedHeight(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 800
	root.Style.Height = DimensionPt(800)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(500)
	child.Style.Height = DimensionPt(300)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 800, root.ContentHeight, 0.001, "root height should be 800 (viewport height)")
	assert.InDelta(t, 300, child.ContentHeight, 0.001, "child height should be 300")
}

func TestLayoutBoxTree_EmptyRoot(t *testing.T) {
	root := makeRoot(500)

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, root.ContentHeight >= 0, "empty root height should be >= 0")
}

func TestLayoutBoxTree_TableLayout(t *testing.T) {
	root := makeRoot(500)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)

	row := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row.Style.Display = DisplayTableRow

	cell1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell1.Style.Display = DisplayTableCell
	cell1.Style.Width = DimensionPt(150)
	cell1.Style.Height = DimensionPt(50)

	cell2 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell2.Style.Display = DisplayTableCell
	cell2.Style.Width = DimensionPt(150)
	cell2.Style.Height = DimensionPt(50)

	row.Children = []*LayoutBox{cell1, cell2}
	table.Children = []*LayoutBox{row}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, cell1.ContentWidth > 0, "table cell1 should have positive width")
	assert.True(t, cell2.ContentWidth > 0, "table cell2 should have positive width")
	assert.True(t, cell1.ContentHeight > 0, "table cell1 should have positive height")
	assert.True(t, cell2.ContentHeight > 0, "table cell2 should have positive height")
}

func TestLayoutBoxTree_GridLayout(t *testing.T) {
	root := makeRoot(500)

	grid := &LayoutBox{Type: BoxGrid, Style: DefaultComputedStyle(), Parent: root}
	grid.Style.Display = DisplayGrid
	grid.Style.Width = DimensionPt(300)
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

	assert.True(t, item1.ContentWidth > 0, "grid item1 should have positive width")
	assert.True(t, item2.ContentWidth > 0, "grid item2 should have positive width")

	assert.True(t, item2.ContentX > item1.ContentX,
		"grid item2 X (%f) should be greater than item1 X (%f)", item2.ContentX, item1.ContentX)
}

func TestLayoutBoxTree_ListItem(t *testing.T) {
	root := makeRoot(500)

	list := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	list.Style.Display = DisplayBlock
	list.Style.Width = DimensionPt(400)

	item := &LayoutBox{Type: BoxListItem, Style: DefaultComputedStyle(), Parent: list}
	item.Style.Display = DisplayListItem
	item.Style.ListStyleType = ListStyleTypeDisc
	item.Style.ListStylePosition = ListStylePositionInside

	textBox := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: item, Text: "Item text"}
	textBox.Style.Display = DisplayInline
	textBox.Style.FontSize = 12
	item.Children = []*LayoutBox{textBox}

	list.Children = []*LayoutBox{item}
	root.Children = []*LayoutBox{list}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, item.ContentWidth > 0, "list item should have positive width")
	assert.True(t, item.ContentHeight > 0, "list item should have positive height")
}

func TestLayoutBoxTree_FloatedChild(t *testing.T) {
	root := makeRoot(500)
	root.Style.OverflowX = OverflowHidden

	child := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	child.Style.Display = DisplayBlock
	child.Style.Float = FloatLeft
	child.Style.Width = DimensionPt(100)
	child.Style.Height = DimensionPt(50)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 100, child.ContentWidth, 0.001, "floated child width should be 100")
	assert.InDelta(t, 50, child.ContentHeight, 0.001, "floated child height should be 50")
	assert.True(t, child.ContentX >= 0, "floated child X should be >= 0")
	assert.True(t, child.ContentY >= 0, "floated child Y should be >= 0")
}

func TestLayoutBoxTree_AspectRatio(t *testing.T) {
	root := makeRoot(500)

	child := &LayoutBox{
		Type:   BoxBlock,
		Style:  DefaultComputedStyle(),
		Parent: root,
	}
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(200)
	child.Style.AspectRatio = 2.0
	child.Style.AspectRatioAuto = false
	child.Style.Height = DimensionAuto()

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 200, child.ContentWidth, 0.001, "child width should be 200")
	assert.InDelta(t, 100, child.ContentHeight, 0.001,
		"child height should be 200 / 2 = 100 (aspect-ratio)")
}
