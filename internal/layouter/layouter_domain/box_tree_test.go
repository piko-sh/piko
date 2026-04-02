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
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestBoxType_String(t *testing.T) {
	tests := []struct {
		name     string
		boxType  BoxType
		expected string
	}{
		{"BoxBlock", BoxBlock, "Block"},
		{"BoxInline", BoxInline, "Inline"},
		{"BoxInlineBlock", BoxInlineBlock, "InlineBlock"},
		{"BoxFlex", BoxFlex, "Flex"},
		{"BoxFlexItem", BoxFlexItem, "FlexItem"},
		{"BoxTable", BoxTable, "Table"},
		{"BoxTableRowGroup", BoxTableRowGroup, "TableRowGroup"},
		{"BoxTableRow", BoxTableRow, "TableRow"},
		{"BoxTableCell", BoxTableCell, "TableCell"},
		{"BoxReplaced", BoxReplaced, "Replaced"},
		{"BoxTextRun", BoxTextRun, "TextRun"},
		{"BoxAnonymousBlock", BoxAnonymousBlock, "AnonymousBlock"},
		{"BoxAnonymousInline", BoxAnonymousInline, "AnonymousInline"},
		{"BoxListItem", BoxListItem, "ListItem"},
		{"BoxListMarker", BoxListMarker, "ListMarker"},
		{"BoxGrid", BoxGrid, "Grid"},
		{"BoxGridItem", BoxGridItem, "GridItem"},
		{"BoxNone", BoxNone, "None"},
		{"unknown value", BoxType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.boxType.String())
		})
	}
}

func TestBoxType_IsInlineLevel(t *testing.T) {
	tests := []struct {
		name     string
		boxType  BoxType
		expected bool
	}{
		{"BoxInline is inline-level", BoxInline, true},
		{"BoxInlineBlock is inline-level", BoxInlineBlock, true},
		{"BoxTextRun is inline-level", BoxTextRun, true},
		{"BoxAnonymousInline is inline-level", BoxAnonymousInline, true},
		{"BoxBlock is not inline-level", BoxBlock, false},
		{"BoxFlex is not inline-level", BoxFlex, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.boxType.IsInlineLevel())
		})
	}
}

func TestBoxType_IsBlockLevel(t *testing.T) {
	tests := []struct {
		name     string
		boxType  BoxType
		expected bool
	}{
		{"BoxBlock is block-level", BoxBlock, true},
		{"BoxFlex is block-level", BoxFlex, true},
		{"BoxTable is block-level", BoxTable, true},
		{"BoxAnonymousBlock is block-level", BoxAnonymousBlock, true},
		{"BoxFlexItem is block-level", BoxFlexItem, true},
		{"BoxListItem is block-level", BoxListItem, true},
		{"BoxGrid is block-level", BoxGrid, true},
		{"BoxGridItem is block-level", BoxGridItem, true},
		{"BoxInline is not block-level", BoxInline, false},
		{"BoxTextRun is not block-level", BoxTextRun, false},
		{"BoxReplaced is not block-level", BoxReplaced, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.boxType.IsBlockLevel())
		})
	}
}

func TestBoxEdges_Horizontal(t *testing.T) {
	tests := []struct {
		name     string
		edges    BoxEdges
		expected float64
	}{
		{
			"distinct values",
			BoxEdges{Top: 10, Right: 20, Bottom: 30, Left: 40},
			60,
		},
		{
			"zero edges",
			BoxEdges{},
			0,
		},
		{
			"all same value",
			BoxEdges{Top: 15, Right: 15, Bottom: 15, Left: 15},
			30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.edges.Horizontal(), 0.001)
		})
	}
}

func TestBoxEdges_Vertical(t *testing.T) {
	tests := []struct {
		name     string
		edges    BoxEdges
		expected float64
	}{
		{
			"distinct values",
			BoxEdges{Top: 10, Right: 20, Bottom: 30, Left: 40},
			40,
		},
		{
			"zero edges",
			BoxEdges{},
			0,
		},
		{
			"all same value",
			BoxEdges{Top: 15, Right: 15, Bottom: 15, Left: 15},
			30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, tt.edges.Vertical(), 0.001)
		})
	}
}

func TestLayoutBox_BorderBoxWidth(t *testing.T) {
	box := &LayoutBox{
		ContentWidth: 100,
		Padding:      BoxEdges{Left: 10, Right: 10},
		Border:       BoxEdges{Left: 5, Right: 5},
	}

	assert.InDelta(t, 130.0, box.BorderBoxWidth(), 0.001)
}

func TestLayoutBox_BorderBoxHeight(t *testing.T) {
	box := &LayoutBox{
		ContentHeight: 100,
		Padding:       BoxEdges{Top: 10, Bottom: 10},
		Border:        BoxEdges{Top: 5, Bottom: 5},
	}

	assert.InDelta(t, 130.0, box.BorderBoxHeight(), 0.001)
}

func TestLayoutBox_BorderBoxX(t *testing.T) {
	box := &LayoutBox{
		ContentX: 50,
		Padding:  BoxEdges{Left: 10},
		Border:   BoxEdges{Left: 5},
	}

	assert.InDelta(t, 35.0, box.BorderBoxX(), 0.001)
}

func TestLayoutBox_BorderBoxY(t *testing.T) {
	box := &LayoutBox{
		ContentY: 50,
		Padding:  BoxEdges{Top: 10},
		Border:   BoxEdges{Top: 5},
	}

	assert.InDelta(t, 35.0, box.BorderBoxY(), 0.001)
}

func TestLayoutBox_MarginBoxWidth(t *testing.T) {
	box := &LayoutBox{
		ContentWidth: 100,
		Padding:      BoxEdges{Left: 10, Right: 10},
		Border:       BoxEdges{Left: 5, Right: 5},
		Margin:       BoxEdges{Left: 8, Right: 12},
	}

	assert.InDelta(t, 150.0, box.MarginBoxWidth(), 0.001)
}

func TestLayoutBox_MarginBoxHeight(t *testing.T) {
	box := &LayoutBox{
		ContentHeight: 100,
		Padding:       BoxEdges{Top: 10, Bottom: 10},
		Border:        BoxEdges{Top: 5, Bottom: 5},
		Margin:        BoxEdges{Top: 8, Bottom: 12},
	}

	assert.InDelta(t, 150.0, box.MarginBoxHeight(), 0.001)
}

func TestLayoutBox_TagName(t *testing.T) {
	tests := []struct {
		name     string
		box      *LayoutBox
		expected string
	}{
		{
			"box with source node tag name",
			&LayoutBox{
				SourceNode: &ast_domain.TemplateNode{TagName: "div"},
				Type:       BoxBlock,
			},
			"div",
		},
		{
			"text run box without source node",
			&LayoutBox{
				Type: BoxTextRun,
			},
			"#text",
		},
		{
			"anonymous block without source node",
			&LayoutBox{
				Type: BoxAnonymousBlock,
			},
			"#anonymous-block",
		},
		{
			"anonymous inline without source node",
			&LayoutBox{
				Type: BoxAnonymousInline,
			},
			"#anonymous-inline",
		},
		{
			"block with nil source node falls back to root",
			&LayoutBox{
				Type: BoxBlock,
			},
			"#root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.box.TagName())
		})
	}
}
