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

	"piko.sh/piko/internal/ast/ast_domain"
)

type mockImageResolver struct {
	width  float64
	height float64
}

func (m *mockImageResolver) GetImageDimensions(_ context.Context, _ string) (float64, float64, error) {
	return m.width, m.height, nil
}

func TestTableLayout_BorderCollapse(t *testing.T) {
	root := makeRoot(400)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)
	table.Style.BorderCollapse = BorderCollapseCollapse
	table.Style.BorderTopWidth = 2
	table.Style.BorderBottomWidth = 2
	table.Style.BorderLeftWidth = 2
	table.Style.BorderRightWidth = 2

	row := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row.Style.Display = DisplayTableRow

	cell1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell1.Style.Display = DisplayTableCell
	cell1.Style.BorderTopWidth = 4
	cell1.Style.BorderBottomWidth = 4
	cell1.Style.BorderLeftWidth = 4
	cell1.Style.BorderRightWidth = 4
	cell1.Style.Height = DimensionPt(50)

	cell2 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell2.Style.Display = DisplayTableCell
	cell2.Style.BorderTopWidth = 2
	cell2.Style.BorderBottomWidth = 2
	cell2.Style.BorderLeftWidth = 2
	cell2.Style.BorderRightWidth = 2
	cell2.Style.Height = DimensionPt(50)

	row.Children = []*LayoutBox{cell1, cell2}
	table.Children = []*LayoutBox{row}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.InDelta(t, 2, cell1.Style.BorderTopWidth, 0.001,
		"cell1 top border should be halved from 4 to 2")
	assert.InDelta(t, 1, cell2.Style.BorderTopWidth, 0.001,
		"cell2 top border should be halved from 2 to 1")

	assert.True(t, cell1.ContentWidth > 0, "cell1 should have positive width")
	assert.True(t, cell2.ContentWidth > 0, "cell2 should have positive width")
	assert.True(t, cell1.ContentHeight > 0, "cell1 should have positive height")
}

func TestTableLayout_RowGroup(t *testing.T) {
	root := makeRoot(400)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)

	group := &LayoutBox{Type: BoxTableRowGroup, Style: DefaultComputedStyle(), Parent: table}
	group.Style.Display = DisplayTableRowGroup

	row := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: group}
	row.Style.Display = DisplayTableRow

	cell1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell1.Style.Display = DisplayTableCell
	cell1.Style.Height = DimensionPt(40)

	cell2 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell2.Style.Display = DisplayTableCell
	cell2.Style.Height = DimensionPt(40)

	row.Children = []*LayoutBox{cell1, cell2}
	group.Children = []*LayoutBox{row}
	table.Children = []*LayoutBox{group}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, cell1.ContentWidth > 0, "cell1 inside row group should have positive width")
	assert.True(t, cell2.ContentWidth > 0, "cell2 inside row group should have positive width")
	assert.True(t, cell1.ContentHeight > 0, "cell1 inside row group should have positive height")
}

func TestTableLayout_Colspan(t *testing.T) {
	root := makeRoot(400)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)

	row1 := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row1.Style.Display = DisplayTableRow

	span_cell := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row1, Colspan: 2, Rowspan: 1}
	span_cell.Style.Display = DisplayTableCell
	span_cell.Style.Height = DimensionPt(30)

	row1.Children = []*LayoutBox{span_cell}

	row2 := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row2.Style.Display = DisplayTableRow

	cell_a := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row2, Colspan: 1, Rowspan: 1}
	cell_a.Style.Display = DisplayTableCell
	cell_a.Style.Height = DimensionPt(30)

	cell_b := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row2, Colspan: 1, Rowspan: 1}
	cell_b.Style.Display = DisplayTableCell
	cell_b.Style.Height = DimensionPt(30)

	row2.Children = []*LayoutBox{cell_a, cell_b}
	table.Children = []*LayoutBox{row1, row2}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, span_cell.ContentWidth > 0, "colspan cell should have positive width")
	assert.True(t, span_cell.ContentWidth >= cell_a.ContentWidth,
		"colspan cell width (%f) should be >= single cell width (%f)",
		span_cell.ContentWidth, cell_a.ContentWidth)
}

func TestTableLayout_Rowspan(t *testing.T) {
	root := makeRoot(400)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)

	row1 := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row1.Style.Display = DisplayTableRow

	tall_cell := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row1, Colspan: 1, Rowspan: 2}
	tall_cell.Style.Display = DisplayTableCell
	tall_cell.Style.Height = DimensionPt(100)

	cell_r1c1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row1, Colspan: 1, Rowspan: 1}
	cell_r1c1.Style.Display = DisplayTableCell
	cell_r1c1.Style.Height = DimensionPt(30)

	row1.Children = []*LayoutBox{tall_cell, cell_r1c1}

	row2 := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row2.Style.Display = DisplayTableRow

	cell_r2c1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row2, Colspan: 1, Rowspan: 1}
	cell_r2c1.Style.Display = DisplayTableCell
	cell_r2c1.Style.Height = DimensionPt(30)

	row2.Children = []*LayoutBox{cell_r2c1}

	table.Children = []*LayoutBox{row1, row2}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, tall_cell.ContentHeight > 0, "rowspan cell should have positive height")
	assert.True(t, cell_r1c1.ContentHeight > 0, "row 1 normal cell should have positive height")
	assert.True(t, cell_r2c1.ContentHeight > 0, "row 2 normal cell should have positive height")
}

func TestTableLayout_AutoLayout(t *testing.T) {
	root := makeRoot(500)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(400)
	table.Style.TableLayout = TableLayoutAuto

	row := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row.Style.Display = DisplayTableRow

	cell1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell1.Style.Display = DisplayTableCell

	text_long := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: cell1, Text: "Hello world, this is a long text"}
	text_long.Style.Display = DisplayInline
	text_long.Style.FontSize = 12
	cell1.Children = []*LayoutBox{text_long}

	cell2 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell2.Style.Display = DisplayTableCell

	text_short := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: cell2, Text: "Hi"}
	text_short.Style.Display = DisplayInline
	text_short.Style.FontSize = 12
	cell2.Children = []*LayoutBox{text_short}

	row.Children = []*LayoutBox{cell1, cell2}
	table.Children = []*LayoutBox{row}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, cell1.ContentWidth > 0, "cell1 should have positive width")
	assert.True(t, cell2.ContentWidth > 0, "cell2 should have positive width")
	assert.True(t, cell1.ContentWidth > cell2.ContentWidth,
		"cell1 with longer text (%f) should be wider than cell2 (%f)",
		cell1.ContentWidth, cell2.ContentWidth)
}

func TestTableLayout_MultipleRows(t *testing.T) {
	root := makeRoot(400)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)

	rows := make([]*LayoutBox, 0, 3)
	for row_index := range 3 {
		row := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
		row.Style.Display = DisplayTableRow

		cell := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
		cell.Style.Display = DisplayTableCell
		cell.Style.Height = DimensionPt(float64(20 + row_index*10))

		row.Children = []*LayoutBox{cell}
		rows = append(rows, row)
	}

	table.Children = rows
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	for i, row := range rows {
		cell := row.Children[0]
		assert.True(t, cell.ContentWidth > 0,
			"row %d cell should have positive width", i)
		assert.True(t, cell.ContentHeight > 0,
			"row %d cell should have positive height", i)
	}
}

func TestTableLayout_EmptyCells(t *testing.T) {
	root := makeRoot(400)

	table := &LayoutBox{Type: BoxTable, Style: DefaultComputedStyle(), Parent: root}
	table.Style.Display = DisplayTable
	table.Style.Width = DimensionPt(300)
	table.Style.TableLayout = TableLayoutFixed

	row := &LayoutBox{Type: BoxTableRow, Style: DefaultComputedStyle(), Parent: table}
	row.Style.Display = DisplayTableRow

	cell1 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell1.Style.Display = DisplayTableCell
	cell1.Style.Height = DimensionPt(40)

	text_box := &LayoutBox{Type: BoxTextRun, Style: DefaultComputedStyle(), Parent: cell1, Text: "Content"}
	text_box.Style.Display = DisplayInline
	text_box.Style.FontSize = 12
	cell1.Children = []*LayoutBox{text_box}

	cell2 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell2.Style.Display = DisplayTableCell
	cell2.Style.Height = DimensionPt(40)

	cell3 := &LayoutBox{Type: BoxTableCell, Style: DefaultComputedStyle(), Parent: row, Colspan: 1, Rowspan: 1}
	cell3.Style.Display = DisplayTableCell

	row.Children = []*LayoutBox{cell1, cell2, cell3}
	table.Children = []*LayoutBox{row}
	root.Children = []*LayoutBox{table}

	require.NotPanics(t, func() { runLayout(root) })

	assert.True(t, cell1.ContentWidth > 0, "cell1 with content should have positive width")
	assert.True(t, cell2.ContentWidth > 0, "empty cell2 should have positive width in fixed layout")
	assert.True(t, cell3.ContentWidth > 0, "empty cell3 should have positive width in fixed layout")
}

func TestBuildBoxTree_SimpleElement(t *testing.T) {
	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}
	style := DefaultComputedStyle()
	style.Display = DisplayBlock
	style_map := StyleMap{node: &style}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}},
		style_map, nil, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, BoxBlock, result.Type)

	require.True(t, len(result.Children) > 0, "root should have at least one child")
	assert.Equal(t, BoxBlock, result.Children[0].Type)
}

func TestBuildBoxTree_TextNode(t *testing.T) {
	text_node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Hello",
	}
	parent_node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "p",
		Children: []*ast_domain.TemplateNode{text_node},
	}
	style := DefaultComputedStyle()
	style.Display = DisplayBlock
	style_map := StyleMap{parent_node: &style}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{parent_node}},
		style_map, nil, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	require.True(t, len(result.Children) > 0, "root should have the p element")
	p_box := result.Children[0]
	require.True(t, len(p_box.Children) > 0, "p box should have a text run child")

	text_run := p_box.Children[0]
	assert.Equal(t, BoxTextRun, text_run.Type)
	assert.Equal(t, "Hello", text_run.Text)
}

func TestBuildBoxTree_DisplayNone(t *testing.T) {
	visible_node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}
	hidden_node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "span",
	}

	visible_style := DefaultComputedStyle()
	visible_style.Display = DisplayBlock

	hidden_style := DefaultComputedStyle()
	hidden_style.Display = DisplayNone

	style_map := StyleMap{
		visible_node: &visible_style,
		hidden_node:  &hidden_style,
	}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{visible_node, hidden_node}},
		style_map, nil, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, len(result.Children),
		"root should have exactly one child; the hidden node should be skipped")
}

func TestBuildBoxTree_DisplayContents(t *testing.T) {
	grandchild_node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Promoted",
	}
	contents_node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{grandchild_node},
	}

	contents_style := DefaultComputedStyle()
	contents_style.Display = DisplayContents

	style_map := StyleMap{contents_node: &contents_style}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{contents_node}},
		style_map, nil, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	require.True(t, len(result.Children) > 0,
		"root should have the promoted text child")
	assert.Equal(t, BoxTextRun, result.Children[0].Type)
	assert.Equal(t, "Promoted", result.Children[0].Text)
}

func TestBuildBoxTree_ReplacedElement(t *testing.T) {
	img_node := &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "img",
		Attributes: []ast_domain.HTMLAttribute{{Name: "src", Value: "test.png"}},
	}

	style := DefaultComputedStyle()
	style.Display = DisplayInline
	style_map := StyleMap{img_node: &style}

	resolver := &mockImageResolver{width: 100, height: 50}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{img_node}},
		style_map, nil, resolver, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, len(result.Children) > 0, "root should contain the img box")

	img_box := result.Children[0]
	assert.Equal(t, BoxReplaced, img_box.Type)
	assert.InDelta(t, 100, img_box.IntrinsicWidth, 0.001)
	assert.InDelta(t, 50, img_box.IntrinsicHeight, 0.001)
}

func TestBuildBoxTree_CommentNode(t *testing.T) {
	comment_node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeComment,
		TextContent: "This is a comment",
	}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{comment_node}},
		nil, nil, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, len(result.Children),
		"comment nodes should not produce any box children")
}

func TestBuildBoxTree_NestedElements(t *testing.T) {
	text_node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Nested text",
	}
	p_node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "p",
		Children: []*ast_domain.TemplateNode{text_node},
	}
	div_node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{p_node},
	}

	div_style := DefaultComputedStyle()
	div_style.Display = DisplayBlock

	p_style := DefaultComputedStyle()
	p_style.Display = DisplayBlock

	style_map := StyleMap{
		div_node: &div_style,
		p_node:   &p_style,
	}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{div_node}},
		style_map, nil, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)

	require.True(t, len(result.Children) > 0, "root should contain the div")
	div_box := result.Children[0]
	assert.Equal(t, BoxBlock, div_box.Type)

	require.True(t, len(div_box.Children) > 0, "div should contain the p")
	p_box := div_box.Children[0]
	assert.Equal(t, BoxBlock, p_box.Type)

	require.True(t, len(p_box.Children) > 0, "p should contain the text run")
	assert.Equal(t, BoxTextRun, p_box.Children[0].Type)
	assert.Equal(t, "Nested text", p_box.Children[0].Text)
}

func TestBuildBoxTree_PseudoElement(t *testing.T) {
	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}

	style := DefaultComputedStyle()
	style.Display = DisplayBlock
	style_map := StyleMap{node: &style}

	pseudo_style := DefaultComputedStyle()
	pseudo_style.Display = DisplayInline
	pseudo_style.Content = "Before: "

	pseudo_map := PseudoStyleMap{
		node: {PseudoBefore: &pseudo_style},
	}

	result, err := BuildBoxTree(
		context.Background(),
		&ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{node}},
		style_map, pseudo_map, nil, 595, 842,
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, len(result.Children) > 0, "root should contain the div")

	div_box := result.Children[0]
	require.True(t, len(div_box.Children) > 0,
		"div should contain the ::before pseudo-element text run")

	pseudo_box := div_box.Children[0]
	assert.Equal(t, BoxTextRun, pseudo_box.Type)
	assert.Equal(t, "Before: ", pseudo_box.Text)
}
