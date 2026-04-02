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

func TestPositionedLayout_AbsoluteWithExplicitSize(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 500
	root.Style.Height = DimensionPt(500)

	child := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	child.Style.Position = PositionAbsolute
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(200)
	child.Style.Height = DimensionPt(100)
	child.Style.Top = DimensionPt(50)
	child.Style.Left = DimensionPt(30)
	child.ContainingBlock = root

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 200, child.ContentWidth, 0.001, "content width should be 200")
	assert.InDelta(t, 100, child.ContentHeight, 0.001, "content height should be 100")
	assert.InDelta(t, 30, child.ContentX, 0.001, "content X should equal left offset")
	assert.InDelta(t, 50, child.ContentY, 0.001, "content Y should equal top offset")
}

func TestPositionedLayout_AbsoluteWithOffsets(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 500
	root.Style.Height = DimensionPt(500)

	child := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	child.Style.Position = PositionAbsolute
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionAuto()
	child.Style.Height = DimensionAuto()
	child.Style.Top = DimensionPt(10)
	child.Style.Bottom = DimensionPt(10)
	child.Style.Left = DimensionPt(10)
	child.Style.Right = DimensionPt(10)
	child.ContainingBlock = root

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 480, child.ContentWidth, 0.001, "width derived from left+right offsets")
	assert.InDelta(t, 480, child.ContentHeight, 0.001, "height derived from top+bottom offsets")
	assert.InDelta(t, 10, child.ContentX, 0.001, "X positioned by left offset")
	assert.InDelta(t, 10, child.ContentY, 0.001, "Y positioned by top offset")
}

func TestPositionedLayout_AbsoluteRightBottom(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 500
	root.Style.Height = DimensionPt(500)

	child := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	child.Style.Position = PositionAbsolute
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(100)
	child.Style.Height = DimensionPt(80)
	child.Style.Right = DimensionPt(20)
	child.Style.Bottom = DimensionPt(30)
	child.ContainingBlock = root

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 100, child.ContentWidth, 0.001, "width should be explicit 100")
	assert.InDelta(t, 80, child.ContentHeight, 0.001, "height should be explicit 80")

	assert.InDelta(t, 380, child.ContentX, 0.001, "X positioned from right edge")

	assert.InDelta(t, 390, child.ContentY, 0.001, "Y positioned from bottom edge")
}

func TestPositionedLayout_FixedPosition(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 800
	root.Style.Height = DimensionPt(800)

	child := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	child.Style.Position = PositionFixed
	child.Style.Display = DisplayBlock
	child.Style.Width = DimensionPt(150)
	child.Style.Height = DimensionPt(60)
	child.Style.Top = DimensionPt(10)
	child.Style.Left = DimensionPt(20)

	root.Children = []*LayoutBox{child}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 150, child.ContentWidth, 0.001, "width should be 150")
	assert.InDelta(t, 60, child.ContentHeight, 0.001, "height should be 60")
	assert.InDelta(t, 20, child.ContentX, 0.001, "X should be left offset against root")
	assert.InDelta(t, 10, child.ContentY, 0.001, "Y should be top offset against root")
}

func TestPositionedLayout_AbsoluteWithBlockChildren(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 500
	root.Style.Height = DimensionPt(500)

	abs_box := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	abs_box.Style.Position = PositionAbsolute
	abs_box.Style.Display = DisplayBlock
	abs_box.Style.Width = DimensionPt(300)
	abs_box.Style.Top = DimensionPt(20)
	abs_box.Style.Left = DimensionPt(10)
	abs_box.ContainingBlock = root

	block_child := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: abs_box}
	block_child.Style.Display = DisplayBlock
	block_child.Style.Width = DimensionPt(200)
	block_child.Style.Height = DimensionPt(40)

	abs_box.Children = []*LayoutBox{block_child}
	root.Children = []*LayoutBox{abs_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 300, abs_box.ContentWidth, 0.001, "abs box width should be 300")
	assert.InDelta(t, 10, abs_box.ContentX, 0.001, "abs box X should be 10")
	assert.InDelta(t, 20, abs_box.ContentY, 0.001, "abs box Y should be 20")

	assert.InDelta(t, 200, block_child.ContentWidth, 0.001, "block child width should be 200")
	assert.InDelta(t, 40, block_child.ContentHeight, 0.001, "block child height should be 40")

	assert.True(t, abs_box.ContentHeight >= 40,
		"abs box height (%f) should be >= block child height (40)", abs_box.ContentHeight)
}

func TestPositionedLayout_AbsoluteWithInlineChildren(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 500
	root.Style.Height = DimensionPt(500)

	abs_box := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	abs_box.Style.Position = PositionAbsolute
	abs_box.Style.Display = DisplayBlock
	abs_box.Style.Width = DimensionPt(300)
	abs_box.Style.Top = DimensionPt(0)
	abs_box.Style.Left = DimensionPt(0)
	abs_box.ContainingBlock = root

	text_box := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: abs_box}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12
	text_box.Text = "Hello"

	abs_box.Children = []*LayoutBox{text_box}
	root.Children = []*LayoutBox{abs_box}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 300, abs_box.ContentWidth, 0.001, "abs box width should be 300")

	assert.True(t, abs_box.ContentHeight > 0,
		"abs box should have positive height from inline content")
	assert.True(t, text_box.ContentWidth > 0, "text run should have positive width")
}

func TestPositionedLayout_NestedAbsolute(t *testing.T) {
	root := makeRoot(500)
	root.ContentHeight = 500
	root.Style.Height = DimensionPt(500)

	outer := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	outer.Style.Position = PositionAbsolute
	outer.Style.Display = DisplayBlock
	outer.Style.Width = DimensionPt(400)
	outer.Style.Height = DimensionPt(400)
	outer.Style.Top = DimensionPt(50)
	outer.Style.Left = DimensionPt(50)
	outer.ContainingBlock = root

	inner := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: outer}
	inner.Style.Position = PositionAbsolute
	inner.Style.Display = DisplayBlock
	inner.Style.Width = DimensionPt(100)
	inner.Style.Height = DimensionPt(80)
	inner.Style.Top = DimensionPt(10)
	inner.Style.Left = DimensionPt(10)
	inner.ContainingBlock = outer

	outer.Children = []*LayoutBox{inner}
	root.Children = []*LayoutBox{outer}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 50, outer.ContentX, 0.001, "outer X")
	assert.InDelta(t, 50, outer.ContentY, 0.001, "outer Y")
	assert.InDelta(t, 400, outer.ContentWidth, 0.001, "outer width")
	assert.InDelta(t, 400, outer.ContentHeight, 0.001, "outer height")

	assert.InDelta(t, 60, inner.ContentX, 0.001, "inner X should be outer origin + left offset")
	assert.InDelta(t, 60, inner.ContentY, 0.001, "inner Y should be outer origin + top offset")
	assert.InDelta(t, 100, inner.ContentWidth, 0.001, "inner width")
	assert.InDelta(t, 80, inner.ContentHeight, 0.001, "inner height")
}

func TestMultiColumnLayout_TwoColumns(t *testing.T) {
	root := makeRoot(400)

	container := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	container.Style.Display = DisplayBlock
	container.Style.Width = DimensionPt(400)
	container.Style.ColumnCount = 2

	child1 := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: container}
	child1.Style.Display = DisplayBlock
	child1.Style.Height = DimensionPt(50)
	child1.Style.Width = DimensionAuto()

	child2 := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: container}
	child2.Style.Display = DisplayBlock
	child2.Style.Height = DimensionPt(50)
	child2.Style.Width = DimensionAuto()

	container.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{container}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child1.ContentWidth > 0, "child1 should have positive width")
	assert.True(t, child2.ContentWidth > 0, "child2 should have positive width")

	assert.True(t, child2.ContentX > child1.ContentX,
		"child2 X (%f) should be greater than child1 X (%f) when in separate columns",
		child2.ContentX, child1.ContentX)
}

func TestMultiColumnLayout_ColumnWidth(t *testing.T) {
	root := makeRoot(500)

	container := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	container.Style.Display = DisplayBlock
	container.Style.Width = DimensionPt(500)
	container.Style.ColumnWidth = DimensionPt(200)

	child1 := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: container}
	child1.Style.Display = DisplayBlock
	child1.Style.Height = DimensionPt(40)
	child1.Style.Width = DimensionAuto()

	child2 := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: container}
	child2.Style.Display = DisplayBlock
	child2.Style.Height = DimensionPt(40)
	child2.Style.Width = DimensionAuto()

	container.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{container}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child1.ContentWidth > 0, "child1 should have positive width")
	assert.True(t, child2.ContentWidth > 0, "child2 should have positive width")

	assert.True(t, child2.ContentX > child1.ContentX,
		"child2 X (%f) should be greater than child1 X (%f)",
		child2.ContentX, child1.ContentX)
}

func TestMultiColumnLayout_ColumnGap(t *testing.T) {
	root := makeRoot(420)

	container := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	container.Style.Display = DisplayBlock
	container.Style.Width = DimensionPt(420)
	container.Style.ColumnCount = 2
	container.Style.ColumnGap = 20

	child1 := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: container}
	child1.Style.Display = DisplayBlock
	child1.Style.Height = DimensionPt(50)
	child1.Style.Width = DimensionAuto()

	child2 := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: container}
	child2.Style.Display = DisplayBlock
	child2.Style.Height = DimensionPt(50)
	child2.Style.Width = DimensionAuto()

	container.Children = []*LayoutBox{child1, child2}
	root.Children = []*LayoutBox{container}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, child1.ContentWidth > 0, "child1 should have positive width")
	assert.True(t, child2.ContentWidth > 0, "child2 should have positive width")

	gap_between := child2.ContentX - child1.ContentX

	assert.InDelta(t, 220, gap_between, 0.001,
		"distance between column origins should be column_width + gap = 220")
}

func TestMultiColumnLayout_WithInlineChildren(t *testing.T) {
	root := makeRoot(400)

	container := &LayoutBox{Type: BoxBlock, Style: DefaultComputedStyle(), Parent: root}
	container.Style.Display = DisplayBlock
	container.Style.Width = DimensionPt(400)
	container.Style.ColumnCount = 2

	text_box := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: container}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12
	text_box.Text = "Hello World"

	container.Children = []*LayoutBox{text_box}
	root.Children = []*LayoutBox{container}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, container.ContentHeight > 0,
		"multi-column container with inline content should have positive height")
	assert.True(t, container.ContentWidth > 0,
		"multi-column container should have positive width")
}
