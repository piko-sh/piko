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

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestPaginate_SinglePage(t *testing.T) {
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 10},
			{ContentY: 100},
			{ContentY: 400},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(841.89))
	assert.Equal(t, 0, maxPage)

	for _, child := range root.Children {
		assert.Equal(t, 0, child.PageIndex)
	}
}

func TestPaginate_TwoPages(t *testing.T) {
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 100},
			{ContentY: 500},
			{ContentY: 900},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(841.89))
	assert.Equal(t, 1, maxPage)

	assert.Equal(t, 0, root.Children[0].PageIndex)
	assert.Equal(t, 0, root.Children[1].PageIndex)
	assert.Equal(t, 1, root.Children[2].PageIndex)
}

func TestPaginate_ManyPages(t *testing.T) {
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0},
			{ContentY: 1000},
			{ContentY: 2000},
			{ContentY: 3000},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(841.89))
	assert.Equal(t, 3, maxPage)

	expected := []int{0, 1, 2, 3}
	for i, child := range root.Children {
		assert.Equal(t, expected[i], child.PageIndex, "child %d", i)
	}
}

func TestPaginate_NestedChildren(t *testing.T) {
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{
				ContentY: 50,
				Children: []*LayoutBox{
					{ContentY: 60},
					{ContentY: 900},
				},
			},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(841.89))
	assert.Equal(t, 1, maxPage)

	assert.Equal(t, 0, root.Children[0].PageIndex)
	assert.Equal(t, 0, root.Children[0].Children[0].PageIndex)
	assert.Equal(t, 1, root.Children[0].Children[1].PageIndex)
}

func TestPageForY_ExactBoundaryFloatingPoint(t *testing.T) {

	for _, h := range []float64{768.2, 841.89, 595.28, 1000.0 / 3.0} {
		g := UniformPageGeometry(h)
		for page := 0; page <= 8; page++ {
			y := float64(page) * h
			assert.Equal(t, page, g.pageForY(y),
				"pageForY(%.6f) with height %.4f", y, h)
		}
	}
}

func TestPaginate_ZeroPageHeight(t *testing.T) {
	root := &LayoutBox{ContentY: 100}
	maxPage := Paginate(context.Background(), root, UniformPageGeometry(0))
	assert.Equal(t, 0, maxPage, "expected maxPage 0 for zero page height")
}

func TestPaginate_NegativePageHeight(t *testing.T) {
	root := &LayoutBox{ContentY: 100}
	maxPage := Paginate(context.Background(), root, UniformPageGeometry(-100))
	assert.Equal(t, 0, maxPage, "expected maxPage 0 for negative page height")
}

func TestPageGeometry_Methods(t *testing.T) {

	g := UniformPageGeometry(200)
	assert.Equal(t, 200.0, g.heightForPage(0))
	assert.Equal(t, 0.0, g.PageStart(0))
	assert.Equal(t, 200.0, g.PageStart(1))
	assert.Equal(t, 600.0, g.PageStart(3))
	assert.Equal(t, 200.0, g.pageEnd(0))
	assert.Equal(t, 0, g.pageForY(0))
	assert.Equal(t, 0, g.pageForY(199))
	assert.Equal(t, 1, g.pageForY(200))
	assert.Equal(t, 2, g.pageForY(500))

	g2 := PageGeometry{DefaultHeight: 200, FirstPageHeight: 100}
	assert.Equal(t, 100.0, g2.heightForPage(0))
	assert.Equal(t, 200.0, g2.heightForPage(1))
	assert.Equal(t, 0.0, g2.PageStart(0))
	assert.Equal(t, 100.0, g2.PageStart(1))
	assert.Equal(t, 300.0, g2.PageStart(2))
	assert.Equal(t, 100.0, g2.pageEnd(0))
	assert.Equal(t, 300.0, g2.pageEnd(1))
	assert.Equal(t, 0, g2.pageForY(50))
	assert.Equal(t, 1, g2.pageForY(100))
	assert.Equal(t, 1, g2.pageForY(299))
	assert.Equal(t, 2, g2.pageForY(300))
}

func TestPaginate_FirstPageDifferentHeight(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 10},
			{ContentY: 80, ContentHeight: 10},
			{ContentY: 150, ContentHeight: 10},
			{ContentY: 350, ContentHeight: 10},
		},
	}

	geo := PageGeometry{DefaultHeight: 200, FirstPageHeight: 100}
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 2, maxPage)

	expectedPages := []int{0, 0, 1, 2}
	for i, child := range root.Children {
		assert.Equal(t, expectedPages[i], child.PageIndex, "child %d", i)
	}
}

func TestPaginate_BreakInsideAvoid_FitsOnPage(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 150},
			{
				ContentY:      150,
				ContentHeight: 100,
				Style:         ComputedStyle{PageBreakInside: PageBreakAvoid},
			},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))
	assert.Equal(t, 1, maxPage)
	assert.Equal(t, 1, root.Children[1].PageIndex)
}

func TestPaginate_BreakInsideAvoid_TallerThanPage(t *testing.T) {

	container := &LayoutBox{
		ContentY:      0,
		ContentHeight: 400,
		Style:         ComputedStyle{PageBreakInside: PageBreakAvoid},
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 100, ContentHeight: 100},
			{ContentY: 200, ContentHeight: 100},
			{ContentY: 300, ContentHeight: 100},
		},
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{container},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(150))

	assert.Equal(t, 0, container.PageIndex)

	expectedPages := []int{0, 1, 2, 3}
	for i, child := range container.Children {
		assert.Equal(t, expectedPages[i], child.PageIndex, "child %d", i)
	}

	assert.Equal(t, 3, maxPage)
}

func TestPaginate_BreakBeforeRight_OnLeftPage(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 200},
			{
				ContentY:      250,
				ContentHeight: 10,
				Style:         ComputedStyle{PageBreakBefore: PageBreakRight},
			},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))

	box := root.Children[1]
	assert.Equal(t, 2, box.PageIndex)
	assert.Equal(t, 0, box.PageIndex%2, "expected box on an even (right) page")
	assert.Equal(t, 2, maxPage)
}

func TestPaginate_BreakBeforeLeft_OnRightPage(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 10},
			{
				ContentY:      50,
				ContentHeight: 10,
				Style:         ComputedStyle{PageBreakBefore: PageBreakLeft},
			},
		},
	}

	Paginate(context.Background(), root, UniformPageGeometry(200))

	box := root.Children[1]
	assert.Equal(t, 1, box.PageIndex)
	assert.Equal(t, 1, box.PageIndex%2, "expected box on an odd (left) page")
}

func TestPaginate_BreakBeforeRight_AlreadyOnRight(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 10},
			{
				ContentY:      50,
				ContentHeight: 10,
				Style:         ComputedStyle{PageBreakBefore: PageBreakRight},
			},
		},
	}

	Paginate(context.Background(), root, UniformPageGeometry(200))

	box := root.Children[1]

	assert.Equal(t, 2, box.PageIndex)
}

func TestPaginate_BreakAfterLeft(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{
				ContentY:      0,
				ContentHeight: 50,
				Style:         ComputedStyle{PageBreakAfter: PageBreakLeft},
			},
			{ContentY: 50, ContentHeight: 10},
		},
	}

	Paginate(context.Background(), root, UniformPageGeometry(200))

	next := root.Children[1]
	assert.Equal(t, 1, next.PageIndex%2)
}

func TestPaginate_OrphansWidows_FitsOnPage(t *testing.T) {

	para := &LayoutBox{
		Type:          BoxBlock,
		ContentY:      0,
		ContentHeight: 90,
		Style:         ComputedStyle{Orphans: 2, Widows: 2},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 0, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 30, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 60, ContentHeight: 30},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{para}}

	Paginate(context.Background(), root, UniformPageGeometry(200))

	for i, line := range para.Children {
		assert.Equal(t, 0, line.PageIndex, "line %d", i)
	}
}

func TestPaginate_OrphansWidows_DefaultSatisfied(t *testing.T) {

	para := &LayoutBox{
		Type:          BoxBlock,
		ContentY:      0,
		ContentHeight: 180,
		Style:         ComputedStyle{Orphans: 2, Widows: 2},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 0, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 30, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 60, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 90, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 120, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 150, ContentHeight: 30},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{para}}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(130))

	assert.Equal(t, 1, maxPage)

	expectedPages := []int{0, 0, 0, 0, 1, 1}
	for i, line := range para.Children {
		assert.Equal(t, expectedPages[i], line.PageIndex, "line %d", i)
	}
}

func TestPaginate_OrphansWidows_WidowsViolated(t *testing.T) {

	para := &LayoutBox{
		Type:          BoxBlock,
		ContentY:      0,
		ContentHeight: 150,
		Style:         ComputedStyle{Orphans: 2, Widows: 2},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 0, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 30, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 60, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 90, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 120, ContentHeight: 30},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{para}}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(130))

	assert.Equal(t, 1, maxPage)

	expectedPages := []int{0, 0, 0, 1, 1}
	for i, line := range para.Children {
		assert.Equal(t, expectedPages[i], line.PageIndex, "line %d", i)
	}
}

func TestPaginate_OrphansWidows_OrphansViolated(t *testing.T) {

	para := &LayoutBox{
		Type:          BoxBlock,
		ContentY:      180,
		ContentHeight: 120,
		Style:         ComputedStyle{Orphans: 2, Widows: 2},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 180, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 210, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 240, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 270, ContentHeight: 30},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{para}}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))

	assert.Equal(t, 1, maxPage)

	for i, line := range para.Children {
		assert.Equal(t, 1, line.PageIndex, "line %d", i)
	}
}

func TestPaginate_OrphansWidows_BothUnsatisfiable(t *testing.T) {

	para := &LayoutBox{
		Type:          BoxBlock,
		ContentY:      170,
		ContentHeight: 90,
		Style:         ComputedStyle{Orphans: 2, Widows: 2},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 170, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 200, ContentHeight: 30},
			{Type: BoxTextRun, ContentY: 230, ContentHeight: 30},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{para}}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))

	assert.Equal(t, 1, maxPage)

	for i, line := range para.Children {
		assert.Equal(t, 1, line.PageIndex, "line %d", i)
	}
}

func TestPaginate_OrphansWidows_CustomValues(t *testing.T) {

	para := &LayoutBox{
		Type:          BoxBlock,
		ContentY:      0,
		ContentHeight: 140,
		Style:         ComputedStyle{Orphans: 3, Widows: 3},
		Children: []*LayoutBox{
			{Type: BoxTextRun, ContentY: 0, ContentHeight: 20},
			{Type: BoxTextRun, ContentY: 20, ContentHeight: 20},
			{Type: BoxTextRun, ContentY: 40, ContentHeight: 20},
			{Type: BoxTextRun, ContentY: 60, ContentHeight: 20},
			{Type: BoxTextRun, ContentY: 80, ContentHeight: 20},
			{Type: BoxTextRun, ContentY: 100, ContentHeight: 20},
			{Type: BoxTextRun, ContentY: 120, ContentHeight: 20},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{para}}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(100))

	assert.Equal(t, 1, maxPage)

	expectedPages := []int{0, 0, 0, 0, 1, 1, 1}
	for i, line := range para.Children {
		assert.Equal(t, expectedPages[i], line.PageIndex, "line %d", i)
	}
}

func TestPaginate_TableHeader_NoThead(t *testing.T) {

	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 300,
		Children: []*LayoutBox{
			{
				Type:          BoxTableRowGroup,
				ContentY:      0,
				ContentHeight: 300,
				Style:         ComputedStyle{Display: DisplayTableRowGroup},
				Children: []*LayoutBox{
					{Type: BoxTableRow, ContentY: 0, ContentHeight: 100},
					{Type: BoxTableRow, ContentY: 100, ContentHeight: 100},
					{Type: BoxTableRow, ContentY: 200, ContentHeight: 100},
				},
			},
		},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))

	assert.Equal(t, 1, maxPage)
}

func TestPaginate_TableHeader_TwoPages_RealisticRowGroup(t *testing.T) {

	pageHeight := 150.0
	geo := UniformPageGeometry(pageHeight)
	ppp := 0.75

	thead := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      0,
		ContentHeight: 0,
		Style:         ComputedStyle{Display: DisplayTableHeaderGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 0, ContentHeight: 30},
		},
	}
	tbody := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      30,
		ContentHeight: 0,
		Style:         ComputedStyle{Display: DisplayTableRowGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 30, ContentHeight: 45},
			{Type: BoxTableRow, ContentY: 75, ContentHeight: 45},
			{Type: BoxTableRow, ContentY: 120, ContentHeight: 45},
			{Type: BoxTableRow, ContentY: 165, ContentHeight: 45},
		},
	}
	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 210,
		Children:      []*LayoutBox{thead, tbody},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	Paginate(context.Background(), root, geo)

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1}
	expectedRelYPx := []float64{40, 100, 40, 100}
	for i, row := range rows {
		assert.Equal(t, expectedPages[i], row.PageIndex, "row %d", i)
		relYPt := row.ContentY + row.PageYOffset - geo.PageStart(row.PageIndex)
		relYPx := relYPt / ppp
		assert.Equal(t, expectedRelYPx[i], relYPx)
	}
}

func TestPaginate_TableHeader_TwoPages(t *testing.T) {

	thead := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      0,
		ContentHeight: 40,
		Style:         ComputedStyle{Display: DisplayTableHeaderGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 0, ContentHeight: 40},
		},
	}
	tbody := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      40,
		ContentHeight: 240,
		Style:         ComputedStyle{Display: DisplayTableRowGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 40, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 100, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 160, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 220, ContentHeight: 60},
		},
	}
	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 280,
		Children:      []*LayoutBox{thead, tbody},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	pageHeight := 200.0
	geo := UniformPageGeometry(pageHeight)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 1, maxPage)

	assert.Equal(t, 0, thead.PageIndex)

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1}
	expectedRelativeY := []float64{40, 100, 40, 100}

	for i, row := range rows {
		assert.Equal(t, expectedPages[i], row.PageIndex, "row %d", i)
		relY := row.ContentY + row.PageYOffset - geo.PageStart(row.PageIndex)
		assert.Equal(t, expectedRelativeY[i], relY)
	}

	originalChildCount := 2
	clonedCount := len(table.Children) - originalChildCount
	assert.Equal(t, 1, clonedCount)

	if clonedCount > 0 {
		cloned := table.Children[originalChildCount]
		assert.Nil(t, cloned.SourceNode, "cloned header should have nil SourceNode")
		assert.Equal(t, 1, cloned.PageIndex)
	}
}

func TestPaginate_TableHeader_ThreePages(t *testing.T) {

	thead := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      0,
		ContentHeight: 40,
		Style:         ComputedStyle{Display: DisplayTableHeaderGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 0, ContentHeight: 40},
		},
	}
	tbody := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      40,
		ContentHeight: 360,
		Style:         ComputedStyle{Display: DisplayTableRowGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 40, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 100, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 160, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 220, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 280, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 340, ContentHeight: 60},
		},
	}
	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 400,
		Children:      []*LayoutBox{thead, tbody},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	pageHeight := 200.0
	geo := UniformPageGeometry(pageHeight)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 2, maxPage)

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1, 2, 2}
	expectedRelativeY := []float64{40, 100, 40, 100, 40, 100}

	for i, row := range rows {
		assert.Equal(t, expectedPages[i], row.PageIndex, "row %d", i)
		relY := row.ContentY + row.PageYOffset - geo.PageStart(row.PageIndex)
		assert.Equal(t, expectedRelativeY[i], relY)
	}

	originalChildCount := 2
	clonedCount := len(table.Children) - originalChildCount
	assert.Equal(t, 2, clonedCount)
}

func TestPaginate_TableFooter_TwoPages(t *testing.T) {

	thead := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      0,
		ContentHeight: 40,
		Style:         ComputedStyle{Display: DisplayTableHeaderGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 0, ContentHeight: 40},
		},
	}
	tfoot := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      280,
		ContentHeight: 40,
		Style:         ComputedStyle{Display: DisplayTableFooterGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 280, ContentHeight: 40},
		},
	}
	tbody := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      40,
		ContentHeight: 240,
		Style:         ComputedStyle{Display: DisplayTableRowGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 40, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 100, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 160, ContentHeight: 60},
			{Type: BoxTableRow, ContentY: 220, ContentHeight: 60},
		},
	}
	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 320,
		Children:      []*LayoutBox{thead, tbody, tfoot},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 1, maxPage)

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1}
	for i, row := range rows {
		assert.Equal(t, expectedPages[i], row.PageIndex, "row %d", i)
	}

	assert.Equal(t, 1, tfoot.PageIndex)

	tfootRelY := tfoot.ContentY + tfoot.PageYOffset - geo.PageStart(tfoot.PageIndex)
	assert.Equal(t, 160.0, tfootRelY)

	extraCount := len(table.Children) - 3
	assert.Equal(t, 2, extraCount)
}

func TestPaginate_TableFooter_NoThead(t *testing.T) {

	tfoot := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      200,
		ContentHeight: 40,
		Style:         ComputedStyle{Display: DisplayTableFooterGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 200, ContentHeight: 40},
		},
	}
	tbody := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      0,
		ContentHeight: 200,
		Style:         ComputedStyle{Display: DisplayTableRowGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 0, ContentHeight: 80},
			{Type: BoxTableRow, ContentY: 80, ContentHeight: 80},
			{Type: BoxTableRow, ContentY: 160, ContentHeight: 80},
		},
	}
	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 240,
		Children:      []*LayoutBox{tbody, tfoot},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 1, maxPage)

	rows := tbody.Children

	expectedPages := []int{0, 0, 1}
	for i, row := range rows {
		assert.Equal(t, expectedPages[i], row.PageIndex, "row %d", i)
	}
}

func TestPaginate_TableFooter_ThreePages(t *testing.T) {

	thead := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      0,
		ContentHeight: 30,
		Style:         ComputedStyle{Display: DisplayTableHeaderGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 0, ContentHeight: 30},
		},
	}
	tfoot := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      280,
		ContentHeight: 30,
		Style:         ComputedStyle{Display: DisplayTableFooterGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 280, ContentHeight: 30},
		},
	}
	tbody := &LayoutBox{
		Type:          BoxTableRowGroup,
		ContentY:      30,
		ContentHeight: 250,
		Style:         ComputedStyle{Display: DisplayTableRowGroup},
		Children: []*LayoutBox{
			{Type: BoxTableRow, ContentY: 30, ContentHeight: 50},
			{Type: BoxTableRow, ContentY: 80, ContentHeight: 50},
			{Type: BoxTableRow, ContentY: 130, ContentHeight: 50},
			{Type: BoxTableRow, ContentY: 180, ContentHeight: 50},
			{Type: BoxTableRow, ContentY: 230, ContentHeight: 50},
		},
	}
	table := &LayoutBox{
		Type:          BoxTable,
		ContentY:      0,
		ContentHeight: 310,
		Children:      []*LayoutBox{thead, tbody, tfoot},
	}
	root := &LayoutBox{ContentY: 0, Children: []*LayoutBox{table}}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 2, maxPage)

	rows := tbody.Children

	expectedPages := []int{0, 0, 1, 1, 2}
	for i, row := range rows {
		assert.Equal(t, expectedPages[i], row.PageIndex, "row %d", i)
	}

	assert.Equal(t, 2, tfoot.PageIndex)

	extraCount := len(table.Children) - 3
	assert.Equal(t, 4, extraCount)
}

func TestPaginate_FixedPosition_ClonedToAllPages(t *testing.T) {

	fixed := &LayoutBox{
		ContentY:      10,
		ContentHeight: 20,
		Style:         ComputedStyle{Position: PositionFixed},
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			fixed,
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 300, ContentHeight: 100},
			{ContentY: 500, ContentHeight: 100},
		},
	}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 2, maxPage)

	assert.Equal(t, 0, fixed.PageIndex)

	cloneCount := 0
	for _, child := range root.Children {
		if child != fixed && child.Style.Position == PositionFixed && child.SourceNode == nil {
			cloneCount++

			origRelY := fixed.ContentY + fixed.PageYOffset - geo.PageStart(fixed.PageIndex)
			cloneRelY := child.ContentY + child.PageYOffset - geo.PageStart(child.PageIndex)
			assert.Equal(t, cloneRelY, origRelY)
		}
	}
	assert.Equal(t, 2, cloneCount)
}

func TestPaginate_FixedPosition_WithTransformAncestor(t *testing.T) {

	ancestor := &LayoutBox{ContentY: 0, ContentHeight: 50}
	fixed := &LayoutBox{
		ContentY:          10,
		ContentHeight:     20,
		Style:             ComputedStyle{Position: PositionFixed},
		TransformAncestor: ancestor,
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			fixed,
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 300, ContentHeight: 100},
		},
	}

	initialChildCount := len(root.Children)
	Paginate(context.Background(), root, UniformPageGeometry(200))

	assert.Equal(t, initialChildCount, len(root.Children))
}

func TestPaginate_FixedPosition_MultipleFixedElements(t *testing.T) {

	fixed1 := &LayoutBox{
		ContentY:      0,
		ContentHeight: 20,
		Style:         ComputedStyle{Position: PositionFixed},
	}
	fixed2 := &LayoutBox{
		ContentY:      50,
		ContentHeight: 20,
		Style:         ComputedStyle{Position: PositionFixed},
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			fixed1, fixed2,
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 300, ContentHeight: 100},
		},
	}

	initialChildCount := len(root.Children)
	Paginate(context.Background(), root, UniformPageGeometry(200))

	cloneCount := len(root.Children) - initialChildCount
	assert.Equal(t, 2, cloneCount)
}

func makeSourceNode(role string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "data-layout-role", Value: role},
		},
	}
}

func TestPaginate_LayoutRoleHeader_TwoPages(t *testing.T) {

	header := &LayoutBox{
		ContentY:      0,
		ContentHeight: 30,
		SourceNode:    makeSourceNode("header"),
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			header,
			{ContentY: 30, ContentHeight: 80},
			{ContentY: 110, ContentHeight: 80},
			{ContentY: 190, ContentHeight: 80},
		},
	}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 1, maxPage)

	assert.Equal(t, 0, header.PageIndex)

	contentBoxes := root.Children[:3]

	cloneCount := 0
	for _, child := range root.Children {
		if child != header && child.SourceNode == nil {
			cloneCount++
		}
	}
	_ = contentBoxes
	assert.GreaterOrEqual(t, cloneCount, 1, "expected at least 1 cloned header for page 1")
}

func TestPaginate_LayoutRoleFooter_TwoPages(t *testing.T) {

	footer := &LayoutBox{
		ContentY:      240,
		ContentHeight: 30,
		SourceNode:    makeSourceNode("footer"),
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 80},
			{ContentY: 80, ContentHeight: 80},
			{ContentY: 160, ContentHeight: 80},
			footer,
		},
	}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.Equal(t, 1, maxPage)

	footerRelY := footer.ContentY + footer.PageYOffset - geo.PageStart(footer.PageIndex)
	assert.Equal(t, 170.0, footerRelY)

}

func TestPaginate_LayoutRoleHeaderAndFooter(t *testing.T) {

	header := &LayoutBox{
		ContentY:      0,
		ContentHeight: 30,
		SourceNode:    makeSourceNode("header"),
	}
	footer := &LayoutBox{
		ContentY:      330,
		ContentHeight: 30,
		SourceNode:    makeSourceNode("footer"),
	}
	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			header,
			{ContentY: 30, ContentHeight: 60},
			{ContentY: 90, ContentHeight: 60},
			{ContentY: 150, ContentHeight: 60},
			{ContentY: 210, ContentHeight: 60},
			{ContentY: 270, ContentHeight: 60},
			footer,
		},
	}

	geo := UniformPageGeometry(200)
	maxPage := Paginate(context.Background(), root, geo)

	assert.GreaterOrEqual(t, maxPage, 1)

	assert.Equal(t, 0, header.PageIndex)

	footerRelY := footer.ContentY + footer.PageYOffset - geo.PageStart(footer.PageIndex)
	assert.Equal(t, 170.0, footerRelY)
}

func TestPaginate_LayoutRole_NoAttribute(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 100, ContentHeight: 100},
			{ContentY: 200, ContentHeight: 100},
		},
	}

	initialCount := len(root.Children)
	Paginate(context.Background(), root, UniformPageGeometry(200))

	assert.Equal(t, initialCount, len(root.Children))
}

func TestPaginate_ChildOverflow_PushToNextPage(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 100, ContentHeight: 100},
			{ContentY: 200, ContentHeight: 100},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(250))
	assert.Equal(t, 1, maxPage)

	assert.Equal(t, 0, root.Children[0].PageIndex)
	assert.Equal(t, 0, root.Children[1].PageIndex)
	assert.Equal(t, 1, root.Children[2].PageIndex)
}

func TestPaginate_ChildOverflow_TallerThanPage(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 300},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))
	assert.Equal(t, 0, maxPage)
	assert.Equal(t, 0, root.Children[0].PageIndex)
}

func TestPaginate_ChildOverflow_FitsExactly(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 100},
			{ContentY: 100, ContentHeight: 150},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(250))
	assert.Equal(t, 0, maxPage)
	assert.Equal(t, 0, root.Children[1].PageIndex)
}

func TestPaginate_ChildOverflow_ChainedPush(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 80},
			{ContentY: 80, ContentHeight: 80},
			{ContentY: 160, ContentHeight: 80},
			{ContentY: 240, ContentHeight: 80},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(250))
	assert.Equal(t, 1, maxPage)

	expected := []int{0, 0, 0, 1}
	for i, child := range root.Children {
		assert.Equal(t, expected[i], child.PageIndex, "child %d", i)
	}
}

func TestPaginate_ChildOverflow_WithBreakAfter(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{
				ContentY:      0,
				ContentHeight: 50,
				Style: ComputedStyle{
					PageBreakAfter: PageBreakAlways,
				},
			},
			{ContentY: 50, ContentHeight: 50},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(250))
	assert.Equal(t, 1, maxPage)
	assert.Equal(t, 1, root.Children[1].PageIndex)
}
