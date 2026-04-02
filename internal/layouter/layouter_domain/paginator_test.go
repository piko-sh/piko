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
	if maxPage != 0 {
		t.Errorf("expected maxPage 0, got %d", maxPage)
	}

	for _, child := range root.Children {
		if child.PageIndex != 0 {
			t.Errorf("expected PageIndex 0 for child at Y=%v, got %d", child.ContentY, child.PageIndex)
		}
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
	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	if root.Children[0].PageIndex != 0 {
		t.Errorf("expected child 0 on page 0, got %d", root.Children[0].PageIndex)
	}
	if root.Children[1].PageIndex != 0 {
		t.Errorf("expected child 1 on page 0, got %d", root.Children[1].PageIndex)
	}
	if root.Children[2].PageIndex != 1 {
		t.Errorf("expected child 2 on page 1, got %d", root.Children[2].PageIndex)
	}
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
	if maxPage != 3 {
		t.Errorf("expected maxPage 3, got %d", maxPage)
	}

	expected := []int{0, 1, 2, 3}
	for i, child := range root.Children {
		if child.PageIndex != expected[i] {
			t.Errorf("child %d: expected page %d, got %d", i, expected[i], child.PageIndex)
		}
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
	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	if root.Children[0].PageIndex != 0 {
		t.Errorf("expected parent on page 0, got %d", root.Children[0].PageIndex)
	}
	if root.Children[0].Children[0].PageIndex != 0 {
		t.Errorf("expected nested child 0 on page 0, got %d", root.Children[0].Children[0].PageIndex)
	}
	if root.Children[0].Children[1].PageIndex != 1 {
		t.Errorf("expected nested child 1 on page 1, got %d", root.Children[0].Children[1].PageIndex)
	}
}

func TestPaginate_ZeroPageHeight(t *testing.T) {
	root := &LayoutBox{ContentY: 100}
	maxPage := Paginate(context.Background(), root, UniformPageGeometry(0))
	if maxPage != 0 {
		t.Errorf("expected maxPage 0 for zero page height, got %d", maxPage)
	}
}

func TestPaginate_NegativePageHeight(t *testing.T) {
	root := &LayoutBox{ContentY: 100}
	maxPage := Paginate(context.Background(), root, UniformPageGeometry(-100))
	if maxPage != 0 {
		t.Errorf("expected maxPage 0 for negative page height, got %d", maxPage)
	}
}

func TestPageGeometry_Methods(t *testing.T) {

	g := UniformPageGeometry(200)
	if g.heightForPage(0) != 200 {
		t.Errorf("uniform heightForPage(0) = %v, want 200", g.heightForPage(0))
	}
	if g.PageStart(0) != 0 {
		t.Errorf("PageStart(0) = %v, want 0", g.PageStart(0))
	}
	if g.PageStart(1) != 200 {
		t.Errorf("PageStart(1) = %v, want 200", g.PageStart(1))
	}
	if g.PageStart(3) != 600 {
		t.Errorf("PageStart(3) = %v, want 600", g.PageStart(3))
	}
	if g.pageEnd(0) != 200 {
		t.Errorf("pageEnd(0) = %v, want 200", g.pageEnd(0))
	}
	if g.pageForY(0) != 0 {
		t.Errorf("pageForY(0) = %v, want 0", g.pageForY(0))
	}
	if g.pageForY(199) != 0 {
		t.Errorf("pageForY(199) = %v, want 0", g.pageForY(199))
	}
	if g.pageForY(200) != 1 {
		t.Errorf("pageForY(200) = %v, want 1", g.pageForY(200))
	}
	if g.pageForY(500) != 2 {
		t.Errorf("pageForY(500) = %v, want 2", g.pageForY(500))
	}

	g2 := PageGeometry{DefaultHeight: 200, FirstPageHeight: 100}
	if g2.heightForPage(0) != 100 {
		t.Errorf("heightForPage(0) = %v, want 100", g2.heightForPage(0))
	}
	if g2.heightForPage(1) != 200 {
		t.Errorf("heightForPage(1) = %v, want 200", g2.heightForPage(1))
	}
	if g2.PageStart(0) != 0 {
		t.Errorf("PageStart(0) = %v, want 0", g2.PageStart(0))
	}
	if g2.PageStart(1) != 100 {
		t.Errorf("PageStart(1) = %v, want 100", g2.PageStart(1))
	}
	if g2.PageStart(2) != 300 {
		t.Errorf("PageStart(2) = %v, want 300", g2.PageStart(2))
	}
	if g2.pageEnd(0) != 100 {
		t.Errorf("pageEnd(0) = %v, want 100", g2.pageEnd(0))
	}
	if g2.pageEnd(1) != 300 {
		t.Errorf("pageEnd(1) = %v, want 300", g2.pageEnd(1))
	}
	if g2.pageForY(50) != 0 {
		t.Errorf("pageForY(50) = %v, want 0", g2.pageForY(50))
	}
	if g2.pageForY(100) != 1 {
		t.Errorf("pageForY(100) = %v, want 1", g2.pageForY(100))
	}
	if g2.pageForY(299) != 1 {
		t.Errorf("pageForY(299) = %v, want 1", g2.pageForY(299))
	}
	if g2.pageForY(300) != 2 {
		t.Errorf("pageForY(300) = %v, want 2", g2.pageForY(300))
	}
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

	if maxPage != 2 {
		t.Errorf("expected maxPage 2, got %d", maxPage)
	}

	expectedPages := []int{0, 0, 1, 2}
	for i, child := range root.Children {
		if child.PageIndex != expectedPages[i] {
			t.Errorf("child %d: expected page %d, got %d", i, expectedPages[i], child.PageIndex)
		}
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
	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}
	if root.Children[1].PageIndex != 1 {
		t.Errorf("expected box on page 1 (moved), got %d", root.Children[1].PageIndex)
	}
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

	if container.PageIndex != 0 {
		t.Errorf("expected container on page 0, got %d", container.PageIndex)
	}

	expectedPages := []int{0, 1, 2, 3}
	for i, child := range container.Children {
		if child.PageIndex != expectedPages[i] {
			t.Errorf("child %d: expected page %d, got %d", i, expectedPages[i], child.PageIndex)
		}
	}

	if maxPage != 3 {
		t.Errorf("expected maxPage 3, got %d", maxPage)
	}
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
	if box.PageIndex != 2 {
		t.Errorf("expected box on page 2 (right), got %d", box.PageIndex)
	}
	if box.PageIndex%2 != 0 {
		t.Error("expected box on an even (right) page")
	}
	if maxPage != 2 {
		t.Errorf("expected maxPage 2, got %d", maxPage)
	}
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
	if box.PageIndex != 1 {
		t.Errorf("expected box on page 1 (left), got %d", box.PageIndex)
	}
	if box.PageIndex%2 != 1 {
		t.Error("expected box on an odd (left) page")
	}
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

	if box.PageIndex != 2 {
		t.Errorf("expected box on page 2 (next right), got %d", box.PageIndex)
	}
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
	if next.PageIndex%2 != 1 {
		t.Errorf("expected next sibling on odd (left) page, got page %d", next.PageIndex)
	}
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
		if line.PageIndex != 0 {
			t.Errorf("line %d: expected page 0, got %d", i, line.PageIndex)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	expectedPages := []int{0, 0, 0, 0, 1, 1}
	for i, line := range para.Children {
		if line.PageIndex != expectedPages[i] {
			t.Errorf("line %d: expected page %d, got %d", i, expectedPages[i], line.PageIndex)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	expectedPages := []int{0, 0, 0, 1, 1}
	for i, line := range para.Children {
		if line.PageIndex != expectedPages[i] {
			t.Errorf("line %d: expected page %d, got %d", i, expectedPages[i], line.PageIndex)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	for i, line := range para.Children {
		if line.PageIndex != 1 {
			t.Errorf("line %d: expected page 1, got %d", i, line.PageIndex)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	for i, line := range para.Children {
		if line.PageIndex != 1 {
			t.Errorf("line %d: expected page 1, got %d", i, line.PageIndex)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	expectedPages := []int{0, 0, 0, 0, 1, 1, 1}
	for i, line := range para.Children {
		if line.PageIndex != expectedPages[i] {
			t.Errorf("line %d: expected page %d, got %d", i, expectedPages[i], line.PageIndex)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}
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
		if row.PageIndex != expectedPages[i] {
			t.Errorf("row %d: expected page %d, got %d", i, expectedPages[i], row.PageIndex)
		}
		relYPt := row.ContentY + row.PageYOffset - geo.PageStart(row.PageIndex)
		relYPx := relYPt / ppp
		if relYPx != expectedRelYPx[i] {
			t.Errorf("row %d: expected page-relative Y=%.0fpx, got %.1fpx", i, expectedRelYPx[i], relYPx)
		}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	if thead.PageIndex != 0 {
		t.Errorf("expected thead on page 0, got %d", thead.PageIndex)
	}

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1}
	expectedRelativeY := []float64{40, 100, 40, 100}

	for i, row := range rows {
		if row.PageIndex != expectedPages[i] {
			t.Errorf("row %d: expected page %d, got %d", i, expectedPages[i], row.PageIndex)
		}
		relY := row.ContentY + row.PageYOffset - geo.PageStart(row.PageIndex)
		if relY != expectedRelativeY[i] {
			t.Errorf("row %d: expected page-relative Y=%.0f, got %.0f", i, expectedRelativeY[i], relY)
		}
	}

	originalChildCount := 2
	clonedCount := len(table.Children) - originalChildCount
	if clonedCount != 1 {
		t.Errorf("expected 1 cloned header, got %d", clonedCount)
	}

	if clonedCount > 0 {
		cloned := table.Children[originalChildCount]
		if cloned.SourceNode != nil {
			t.Error("cloned header should have nil SourceNode")
		}
		if cloned.PageIndex != 1 {
			t.Errorf("cloned header expected on page 1, got %d", cloned.PageIndex)
		}
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

	if maxPage != 2 {
		t.Errorf("expected maxPage 2, got %d", maxPage)
	}

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1, 2, 2}
	expectedRelativeY := []float64{40, 100, 40, 100, 40, 100}

	for i, row := range rows {
		if row.PageIndex != expectedPages[i] {
			t.Errorf("row %d: expected page %d, got %d", i, expectedPages[i], row.PageIndex)
		}
		relY := row.ContentY + row.PageYOffset - geo.PageStart(row.PageIndex)
		if relY != expectedRelativeY[i] {
			t.Errorf("row %d: expected page-relative Y=%.0f, got %.0f", i, expectedRelativeY[i], relY)
		}
	}

	originalChildCount := 2
	clonedCount := len(table.Children) - originalChildCount
	if clonedCount != 2 {
		t.Errorf("expected 2 cloned headers, got %d", clonedCount)
	}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	rows := tbody.Children
	expectedPages := []int{0, 0, 1, 1}
	for i, row := range rows {
		if row.PageIndex != expectedPages[i] {
			t.Errorf("row %d: expected page %d, got %d", i, expectedPages[i], row.PageIndex)
		}
	}

	if tfoot.PageIndex != 1 {
		t.Errorf("expected original tfoot on page 1, got %d", tfoot.PageIndex)
	}

	tfootRelY := tfoot.ContentY + tfoot.PageYOffset - geo.PageStart(tfoot.PageIndex)
	if tfootRelY != 160 {
		t.Errorf("expected tfoot page-relative Y=160, got %.0f", tfootRelY)
	}

	extraCount := len(table.Children) - 3
	if extraCount != 2 {
		t.Errorf("expected 2 cloned elements (1 header + 1 footer), got %d", extraCount)
	}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	rows := tbody.Children

	expectedPages := []int{0, 0, 1}
	for i, row := range rows {
		if row.PageIndex != expectedPages[i] {
			t.Errorf("row %d: expected page %d, got %d", i, expectedPages[i], row.PageIndex)
		}
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

	if maxPage != 2 {
		t.Errorf("expected maxPage 2, got %d", maxPage)
	}

	rows := tbody.Children

	expectedPages := []int{0, 0, 1, 1, 2}
	for i, row := range rows {
		if row.PageIndex != expectedPages[i] {
			t.Errorf("row %d: expected page %d, got %d", i, expectedPages[i], row.PageIndex)
		}
	}

	if tfoot.PageIndex != 2 {
		t.Errorf("expected original tfoot on page 2, got %d", tfoot.PageIndex)
	}

	extraCount := len(table.Children) - 3
	if extraCount != 4 {
		t.Errorf("expected 4 cloned elements (2 headers + 2 footers), got %d", extraCount)
	}
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

	if maxPage != 2 {
		t.Errorf("expected maxPage 2, got %d", maxPage)
	}

	if fixed.PageIndex != 0 {
		t.Errorf("expected original fixed on page 0, got %d", fixed.PageIndex)
	}

	cloneCount := 0
	for _, child := range root.Children {
		if child != fixed && child.Style.Position == PositionFixed && child.SourceNode == nil {
			cloneCount++

			origRelY := fixed.ContentY + fixed.PageYOffset - geo.PageStart(fixed.PageIndex)
			cloneRelY := child.ContentY + child.PageYOffset - geo.PageStart(child.PageIndex)
			if origRelY != cloneRelY {
				t.Errorf("clone on page %d: expected relY=%.0f, got %.0f", child.PageIndex, origRelY, cloneRelY)
			}
		}
	}
	if cloneCount != 2 {
		t.Errorf("expected 2 fixed clones, got %d", cloneCount)
	}
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

	if len(root.Children) != initialChildCount {
		t.Errorf("expected no clones (TransformAncestor), got %d extra children", len(root.Children)-initialChildCount)
	}
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
	if cloneCount != 2 {
		t.Errorf("expected 2 fixed clones (1 per element), got %d", cloneCount)
	}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	if header.PageIndex != 0 {
		t.Errorf("expected header on page 0, got %d", header.PageIndex)
	}

	contentBoxes := root.Children[:3]

	cloneCount := 0
	for _, child := range root.Children {
		if child != header && child.SourceNode == nil {
			cloneCount++
		}
	}
	_ = contentBoxes
	if cloneCount < 1 {
		t.Errorf("expected at least 1 cloned header for page 1, got %d", cloneCount)
	}
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

	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	footerRelY := footer.ContentY + footer.PageYOffset - geo.PageStart(footer.PageIndex)
	if footerRelY != 170 {
		t.Errorf("expected footer page-relative Y=170, got %.0f", footerRelY)
	}

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

	if maxPage < 1 {
		t.Errorf("expected maxPage >= 1, got %d", maxPage)
	}

	if header.PageIndex != 0 {
		t.Errorf("expected header on page 0, got %d", header.PageIndex)
	}

	footerRelY := footer.ContentY + footer.PageYOffset - geo.PageStart(footer.PageIndex)
	if footerRelY != 170 {
		t.Errorf("expected footer page-relative Y=170, got %.0f", footerRelY)
	}
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

	if len(root.Children) != initialCount {
		t.Errorf("expected no extra children, got %d (was %d)", len(root.Children), initialCount)
	}
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
	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	if root.Children[0].PageIndex != 0 {
		t.Errorf("expected child 0 on page 0, got %d", root.Children[0].PageIndex)
	}
	if root.Children[1].PageIndex != 0 {
		t.Errorf("expected child 1 on page 0, got %d", root.Children[1].PageIndex)
	}
	if root.Children[2].PageIndex != 1 {
		t.Errorf("expected child 2 on page 1, got %d", root.Children[2].PageIndex)
	}
}

func TestPaginate_ChildOverflow_TallerThanPage(t *testing.T) {

	root := &LayoutBox{
		ContentY: 0,
		Children: []*LayoutBox{
			{ContentY: 0, ContentHeight: 300},
		},
	}

	maxPage := Paginate(context.Background(), root, UniformPageGeometry(200))
	if maxPage != 0 {
		t.Errorf("expected maxPage 0, got %d", maxPage)
	}
	if root.Children[0].PageIndex != 0 {
		t.Errorf("expected child on page 0, got %d", root.Children[0].PageIndex)
	}
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
	if maxPage != 0 {
		t.Errorf("expected maxPage 0, got %d", maxPage)
	}
	if root.Children[1].PageIndex != 0 {
		t.Errorf("expected child 1 on page 0, got %d", root.Children[1].PageIndex)
	}
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
	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}

	expected := []int{0, 0, 0, 1}
	for i, child := range root.Children {
		if child.PageIndex != expected[i] {
			t.Errorf("child %d: expected page %d, got %d", i, expected[i], child.PageIndex)
		}
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
	if maxPage != 1 {
		t.Errorf("expected maxPage 1, got %d", maxPage)
	}
	if root.Children[1].PageIndex != 1 {
		t.Errorf("expected child 1 on page 1, got %d", root.Children[1].PageIndex)
	}
}
