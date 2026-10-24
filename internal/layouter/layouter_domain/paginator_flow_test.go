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

func keep(y, height float64) *LayoutBox {
	return &LayoutBox{ContentY: y, ContentHeight: height, Style: ComputedStyle{PageBreakInside: PageBreakAvoid}}
}

func TestPaginate_PaddedPushLandsFlush(t *testing.T) {
	box := keep(100, 80)
	box.Padding = BoxEdges{Top: 10, Bottom: 10}
	box.Border = BoxEdges{Top: 2, Bottom: 2}
	box.ContentY = 100 + 12
	root := &LayoutBox{Children: []*LayoutBox{box}}

	mustPaginate(t, root, UniformPageGeometry(150))

	assert.Equal(t, 1, box.PageIndex)
	assert.InDelta(t, 150, box.BorderBoxY()+box.PageYOffset, 0.001)
}

func TestPaginate_ContainerSplitsBetweenChildren(t *testing.T) {
	first, second, third := keep(0, 100), keep(100, 100), keep(200, 100)
	container := &LayoutBox{ContentY: 0, ContentHeight: 300, Children: []*LayoutBox{first, second, third}}
	root := &LayoutBox{Children: []*LayoutBox{container}}

	maxPage := mustPaginate(t, root, UniformPageGeometry(250))

	assert.Equal(t, 1, maxPage)
	assert.Equal(t, []int{0, 0, 1}, []int{first.PageIndex, second.PageIndex, third.PageIndex})
	assert.Equal(t, 0, container.PageIndex)
	assert.Equal(t, 1, container.LastPageIndex)
	require.Len(t, container.Fragments, 2)
	assert.InDelta(t, 250, container.Fragments[0].Bottom, 0.001)
	assert.InDelta(t, 250, container.Fragments[1].Top, 0.001)
	assert.InDelta(t, 350, container.Fragments[1].Bottom, 0.001, "the third child moved down 50")
}

func TestPaginate_FirstPieceThatDoesNotFitMovesItsContainer(t *testing.T) {
	inner := keep(232, 60)
	container := &LayoutBox{ContentY: 232, ContentHeight: 60, Border: BoxEdges{Top: 2}, Children: []*LayoutBox{inner}}
	root := &LayoutBox{Children: []*LayoutBox{keep(0, 220), container}}

	mustPaginate(t, root, UniformPageGeometry(250))

	assert.Equal(t, 1, container.PageIndex)
	assert.Equal(t, 1, inner.PageIndex)
	assert.InDelta(t, 250, container.BorderBoxY()+container.PageYOffset, 0.001)
	assert.Nil(t, container.Fragments)
}

func TestPaginate_HeadingKeptWithNext(t *testing.T) {
	heading := keep(200, 30)
	heading.Style.PageBreakAfter = PageBreakAvoid
	body := keep(230, 60)
	root := &LayoutBox{Children: []*LayoutBox{keep(0, 200), heading, body}}

	mustPaginate(t, root, UniformPageGeometry(250))

	assert.Equal(t, 1, heading.PageIndex, "the heading moves with the box it introduces")
	assert.Equal(t, 1, body.PageIndex)
	assert.InDelta(t, 250, heading.BorderBoxY()+heading.PageYOffset, 0.001)
}

func TestPaginate_AvoidChainLongerThanAPageIsDropped(t *testing.T) {
	heading := keep(200, 30)
	heading.Style.PageBreakAfter = PageBreakAvoid
	body := &LayoutBox{ContentY: 230, ContentHeight: 400}
	root := &LayoutBox{Children: []*LayoutBox{keep(0, 200), heading, body}}

	mustPaginate(t, root, UniformPageGeometry(250))

	assert.Equal(t, 0, heading.PageIndex)
}

func TestPaginate_ForcedBreakAfterPropagatesFromTheLastChild(t *testing.T) {
	last := &LayoutBox{ContentY: 20, ContentHeight: 20, Style: ComputedStyle{PageBreakAfter: PageBreakAlways}}
	section := &LayoutBox{ContentY: 0, ContentHeight: 40, Children: []*LayoutBox{keep(0, 20), last}}
	next := &LayoutBox{ContentY: 40, ContentHeight: 20}
	root := &LayoutBox{Children: []*LayoutBox{section, next}}

	mustPaginate(t, root, UniformPageGeometry(250))

	assert.Equal(t, 1, next.PageIndex)
}

func TestPaginate_GridRowItemsBreakSideBySide(t *testing.T) {
	meta := keep(0, 40)
	meta.gridPlaced, meta.gridRowStart, meta.gridRowEnd = true, 0, 1
	bullets := []*LayoutBox{keep(0, 100), keep(100, 100), keep(200, 100)}
	body := &LayoutBox{ContentY: 0, ContentHeight: 300, Border: BoxEdges{Left: 1}, Children: bullets}
	body.gridPlaced, body.gridRowStart, body.gridRowEnd = true, 0, 1
	grid := &LayoutBox{Type: BoxGrid, ContentY: 0, ContentHeight: 300, Children: []*LayoutBox{meta, body}}
	after := keep(300, 20)
	root := &LayoutBox{Children: []*LayoutBox{grid, after}}

	mustPaginate(t, root, UniformPageGeometry(250))

	assert.Equal(t, 0, meta.PageIndex)
	assert.Equal(t, 0, body.PageIndex)
	assert.Equal(t, []int{0, 0, 1}, []int{bullets[0].PageIndex, bullets[1].PageIndex, bullets[2].PageIndex})
	require.Len(t, body.Fragments, 2, "the body column spans both pages")
	assert.InDelta(t, 350, body.Fragments[1].Bottom, 0.001)
	assert.InDelta(t, 350, after.BorderBoxY()+after.PageYOffset, 0.001, "the next row starts below the moved bullet")
}

func TestPaginate_MixedInlineParagraphBreaksBetweenLines(t *testing.T) {
	bold := &LayoutBox{Type: BoxInline, ContentY: 60, ContentHeight: 30}
	para := &LayoutBox{
		ContentY: 0, ContentHeight: 150,
		Style: ComputedStyle{Orphans: 2, Widows: 2},
		lines: []lineRecord{{0, 30}, {30, 30}, {60, 30}, {90, 30}, {120, 30}},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 0, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 30, ContentHeight: 30},
			bold,
			{Type: BoxTextRun, ContentY: 90, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 120, ContentHeight: 30},
		},
	}
	root := &LayoutBox{Children: []*LayoutBox{para}}

	mustPaginate(t, root, UniformPageGeometry(100))

	pages := make([]int, 0, len(para.Children))
	for _, child := range para.Children {
		pages = append(pages, child.PageIndex)
	}
	assert.Equal(t, []int{0, 0, 0, 1, 1}, pages)
	assert.Equal(t, 0, bold.PageIndex)
}

func TestPaginate_AvoidChainInATallParentStillMoves(t *testing.T) {
	caption := &LayoutBox{ContentY: 170, ContentHeight: 20, Style: ComputedStyle{PageBreakAfter: PageBreakAvoid}}
	block := keep(190, 40)
	body := &LayoutBox{ContentY: 0, ContentHeight: 800, Children: []*LayoutBox{keep(0, 170), caption, block}}
	root := &LayoutBox{Children: []*LayoutBox{body}}

	mustPaginate(t, root, UniformPageGeometry(200))

	assert.Equal(t, 1, caption.PageIndex)
	assert.Equal(t, 1, block.PageIndex)
	assert.InDelta(t, 200, caption.BorderBoxY()+caption.PageYOffset, 0.001)
}

func TestPaginate_LayoutRoleHeaderAndFooterPlaceContentBetweenThem(t *testing.T) {
	header := &LayoutBox{ContentY: 0, ContentHeight: 40, SourceNode: makeSourceNode("header")}
	blocks := []*LayoutBox{
		{ContentY: 40, ContentHeight: 60},
		{ContentY: 100, ContentHeight: 60},
		{ContentY: 160, ContentHeight: 60},
		{ContentY: 220, ContentHeight: 60},
	}
	footer := &LayoutBox{ContentY: 280, ContentHeight: 40, SourceNode: makeSourceNode("footer")}
	root := &LayoutBox{Children: append(append([]*LayoutBox{header}, blocks...), footer)}

	mustPaginate(t, root, UniformPageGeometry(200))

	pages := make([]int, 0, len(blocks))
	tops := make([]float64, 0, len(blocks))
	for _, block := range blocks {
		pages = append(pages, block.PageIndex)
		tops = append(tops, block.BorderBoxY()+block.PageYOffset-float64(block.PageIndex)*200)
	}
	assert.Equal(t, []int{0, 0, 1, 1}, pages)
	assert.InDeltaSlice(t, []float64{40, 100, 40, 100}, tops, 0.001)
}
