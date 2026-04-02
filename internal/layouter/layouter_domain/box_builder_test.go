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

func TestDetermineBoxType(t *testing.T) {
	tests := []struct {
		name          string
		tagName       string
		display       DisplayType
		parentBoxType BoxType
		parentDisplay DisplayType
		expected      BoxType
	}{

		{"img element", "img", DisplayBlock, BoxBlock, DisplayBlock, BoxReplaced},
		{"svg element", "svg", DisplayInline, BoxBlock, DisplayBlock, BoxReplaced},
		{"video element", "video", DisplayFlex, BoxBlock, DisplayBlock, BoxReplaced},
		{"piko:img element", "piko:img", DisplayBlock, BoxBlock, DisplayBlock, BoxReplaced},
		{"piko:svg element", "piko:svg", DisplayInline, BoxBlock, DisplayBlock, BoxReplaced},

		{"display inline under block parent", "div", DisplayInline, BoxBlock, DisplayBlock, BoxInline},
		{"display block under block parent", "div", DisplayBlock, BoxBlock, DisplayBlock, BoxBlock},
		{"display inline-block under block parent", "div", DisplayInlineBlock, BoxBlock, DisplayBlock, BoxInlineBlock},
		{"display flex under block parent", "div", DisplayFlex, BoxBlock, DisplayBlock, BoxFlex},
		{"display grid under block parent", "div", DisplayGrid, BoxBlock, DisplayBlock, BoxGrid},
		{"display table under block parent", "div", DisplayTable, BoxBlock, DisplayBlock, BoxTable},
		{"display table-row under block parent", "div", DisplayTableRow, BoxBlock, DisplayBlock, BoxTableRow},
		{"display table-cell under block parent", "div", DisplayTableCell, BoxBlock, DisplayBlock, BoxTableCell},
		{"display list-item under block parent", "div", DisplayListItem, BoxBlock, DisplayBlock, BoxListItem},

		{"any element under flex parent", "span", DisplayInline, BoxFlex, DisplayFlex, BoxFlexItem},
		{"any element under grid parent", "span", DisplayBlock, BoxGrid, DisplayGrid, BoxGridItem},

		{"table-row under flex-item table", "tr", DisplayTableRow, BoxFlexItem, DisplayTable, BoxTableRow},
		{"table-cell under flex-item table", "td", DisplayTableCell, BoxFlexItem, DisplayTable, BoxTableCell},

		{"block under flex-item flex", "div", DisplayBlock, BoxFlexItem, DisplayFlex, BoxFlexItem},

		{"display table-row-group", "div", DisplayTableRowGroup, BoxBlock, DisplayBlock, BoxTableRowGroup},
		{"display table-header-group", "div", DisplayTableHeaderGroup, BoxBlock, DisplayBlock, BoxTableRowGroup},
		{"display table-footer-group", "div", DisplayTableFooterGroup, BoxBlock, DisplayBlock, BoxTableRowGroup},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  tt.tagName,
			}
			style := &ComputedStyle{Display: tt.display}
			result := determineBoxType(node, style, tt.parentBoxType, tt.parentDisplay)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsReplacedElement(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{"img is replaced", "img", true},
		{"svg is replaced", "svg", true},
		{"video is replaced", "video", true},
		{"canvas is replaced", "canvas", true},
		{"iframe is replaced", "iframe", true},
		{"piko:img is replaced", "piko:img", true},
		{"piko:svg is replaced", "piko:svg", true},
		{"piko:picture is replaced", "piko:picture", true},
		{"div is not replaced", "div", false},
		{"span is not replaced", "span", false},
		{"p is not replaced", "p", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  tt.tagName,
			}
			assert.Equal(t, tt.expected, isReplacedElement(node))
		})
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name     string
		node     *ast_domain.TemplateNode
		expected string
	}{
		{
			name: "returns TextContent when set",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "hello world",
			},
			expected: "hello world",
		},
		{
			name: "unescapes HTML entities in TextContent",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: `requires &#34;hello&#34; &amp; &lt;more&gt;`,
			},
			expected: `requires "hello" & <more>`,
		},
		{
			name: "concatenates literal rich text parts",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{
					{IsLiteral: true, Literal: "foo"},
					{IsLiteral: true, Literal: "bar"},
				},
			},
			expected: "foobar",
		},
		{
			name: "empty node returns empty string",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
			},
			expected: "",
		},
		{
			name: "skips non-literal rich text parts",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{
					{IsLiteral: true, Literal: "kept"},
					{IsLiteral: false, Literal: "skipped"},
					{IsLiteral: true, Literal: "also kept"},
				},
			},
			expected: "keptalso kept",
		},
		{
			name: "returns TextContentWriter content from DirectWriter",
			node: func() *ast_domain.TemplateNode {
				dw := ast_domain.GetDirectWriter()
				dw.AppendEscapeString("dynamic value")
				return &ast_domain.TemplateNode{
					NodeType:          ast_domain.NodeText,
					TextContentWriter: dw,
				}
			}(),
			expected: "dynamic value",
		},
		{
			name: "TextContentWriter returns raw text without HTML escaping",
			node: func() *ast_domain.TemplateNode {
				dw := ast_domain.GetDirectWriter()
				dw.AppendEscapeString(`value with "quotes" & <angles>`)
				return &ast_domain.TemplateNode{
					NodeType:          ast_domain.NodeText,
					TextContentWriter: dw,
				}
			}(),
			expected: `value with "quotes" & <angles>`,
		},
		{
			name: "TextContentWriter takes priority over TextContent",
			node: func() *ast_domain.TemplateNode {
				dw := ast_domain.GetDirectWriter()
				dw.AppendEscapeString("from writer")
				return &ast_domain.TemplateNode{
					NodeType:          ast_domain.NodeText,
					TextContent:       "from plain",
					TextContentWriter: dw,
				}
			}(),
			expected: "from writer",
		},
		{
			name: "empty TextContentWriter falls through to TextContent",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				TextContent:       "fallback",
				TextContentWriter: ast_domain.GetDirectWriter(),
			},
			expected: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractTextContent(tt.node))
		})
	}
}

func TestResolveMarkerText(t *testing.T) {
	tests := []struct {
		name     string
		style    *ComputedStyle
		parent   *LayoutBox
		expected string
	}{
		{
			name:     "disc produces bullet",
			style:    &ComputedStyle{ListStyleType: ListStyleTypeDisc},
			parent:   &LayoutBox{},
			expected: "\u2022 ",
		},
		{
			name:     "circle produces open bullet",
			style:    &ComputedStyle{ListStyleType: ListStyleTypeCircle},
			parent:   &LayoutBox{},
			expected: "\u25E6 ",
		},
		{
			name:     "square produces filled square",
			style:    &ComputedStyle{ListStyleType: ListStyleTypeSquare},
			parent:   &LayoutBox{},
			expected: "\u25AA ",
		},
		{
			name:     "decimal with no existing list item siblings",
			style:    &ComputedStyle{ListStyleType: ListStyleTypeDecimal},
			parent:   &LayoutBox{},
			expected: "0. ",
		},
		{
			name:  "decimal counts existing list item children",
			style: &ComputedStyle{ListStyleType: ListStyleTypeDecimal},
			parent: &LayoutBox{
				Children: []*LayoutBox{
					{Type: BoxListItem},
					{Type: BoxListItem},
				},
			},
			expected: "2. ",
		},
		{
			name:     "none produces empty string",
			style:    &ComputedStyle{ListStyleType: ListStyleTypeNone},
			parent:   &LayoutBox{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolveMarkerText(tt.style, tt.parent))
		})
	}
}

func TestParseIntAttributeOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		attrName     string
		attrs        []ast_domain.HTMLAttribute
		defaultValue int
		expected     int
	}{
		{
			name:         "valid attribute value",
			attrs:        []ast_domain.HTMLAttribute{{Name: "colspan", Value: "3"}},
			attrName:     "colspan",
			defaultValue: 1,
			expected:     3,
		},
		{
			name:         "attribute absent returns default",
			attrs:        []ast_domain.HTMLAttribute{},
			attrName:     "colspan",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "invalid value returns default",
			attrs:        []ast_domain.HTMLAttribute{{Name: "colspan", Value: "abc"}},
			attrName:     "colspan",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "zero value clamped to default",
			attrs:        []ast_domain.HTMLAttribute{{Name: "colspan", Value: "0"}},
			attrName:     "colspan",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "negative value returns default",
			attrs:        []ast_domain.HTMLAttribute{{Name: "rowspan", Value: "-1"}},
			attrName:     "rowspan",
			defaultValue: 1,
			expected:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{
				NodeType:   ast_domain.NodeElement,
				TagName:    "td",
				Attributes: tt.attrs,
			}
			assert.Equal(t, tt.expected, parseIntAttributeOrDefault(node, tt.attrName, tt.defaultValue))
		})
	}
}

func TestHasMixedChildren(t *testing.T) {
	tests := []struct {
		name     string
		children []*LayoutBox
		expected bool
	}{
		{
			name: "only inline children",
			children: []*LayoutBox{
				{Type: BoxTextRun},
				{Type: BoxInline},
			},
			expected: false,
		},
		{
			name: "only block children",
			children: []*LayoutBox{
				{Type: BoxBlock},
				{Type: BoxBlock},
			},
			expected: false,
		},
		{
			name: "mixed inline and block children",
			children: []*LayoutBox{
				{Type: BoxTextRun},
				{Type: BoxBlock},
			},
			expected: true,
		},
		{
			name:     "no children",
			children: nil,
			expected: false,
		},
		{
			name: "list markers are ignored",
			children: []*LayoutBox{
				{Type: BoxListMarker},
				{Type: BoxBlock},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			box := &LayoutBox{Children: tt.children}
			assert.Equal(t, tt.expected, hasMixedChildren(box))
		})
	}
}

func TestWrapInAnonymousBlock(t *testing.T) {
	t.Run("wraps children with correct type and inherited style", func(t *testing.T) {
		parent := &LayoutBox{
			Type:  BoxBlock,
			Style: DefaultComputedStyle(),
		}
		child1 := &LayoutBox{Type: BoxTextRun, Parent: parent}
		child2 := &LayoutBox{Type: BoxInline, Parent: parent}

		anon := wrapInAnonymousBlock([]*LayoutBox{child1, child2}, parent)

		assert.Equal(t, BoxAnonymousBlock, anon.Type)
		assert.Equal(t, DisplayBlock, anon.Style.Display)
		assert.Len(t, anon.Children, 2)
		assert.Equal(t, parent, anon.Parent)
	})

	t.Run("reparents children to anonymous block", func(t *testing.T) {
		parent := &LayoutBox{
			Type:  BoxBlock,
			Style: DefaultComputedStyle(),
		}
		child := &LayoutBox{Type: BoxTextRun, Parent: parent}

		anon := wrapInAnonymousBlock([]*LayoutBox{child}, parent)

		assert.Equal(t, anon, child.Parent)
	})
}

func TestFixAnonymousBoxes(t *testing.T) {
	t.Run("all inline children are not wrapped", func(t *testing.T) {
		box := &LayoutBox{
			Type: BoxBlock,
			Children: []*LayoutBox{
				{Type: BoxTextRun},
				{Type: BoxInline},
			},
		}
		fixAnonymousBoxes(box)

		assert.Len(t, box.Children, 2)
		assert.Equal(t, BoxTextRun, box.Children[0].Type)
		assert.Equal(t, BoxInline, box.Children[1].Type)
	})

	t.Run("all block children are not wrapped", func(t *testing.T) {
		box := &LayoutBox{
			Type: BoxBlock,
			Children: []*LayoutBox{
				{Type: BoxBlock},
				{Type: BoxBlock},
			},
		}
		fixAnonymousBoxes(box)
		assert.Len(t, box.Children, 2)
		assert.Equal(t, BoxBlock, box.Children[0].Type)
		assert.Equal(t, BoxBlock, box.Children[1].Type)
	})

	t.Run("mixed children wraps inline in anonymous blocks", func(t *testing.T) {
		parent := &LayoutBox{
			Type:  BoxBlock,
			Style: DefaultComputedStyle(),
		}
		inline1 := &LayoutBox{Type: BoxTextRun, Parent: parent}
		block := &LayoutBox{Type: BoxBlock, Parent: parent}
		inline2 := &LayoutBox{Type: BoxInline, Parent: parent}
		parent.Children = []*LayoutBox{inline1, block, inline2}

		fixAnonymousBoxes(parent)

		assert.Len(t, parent.Children, 3)
		assert.Equal(t, BoxAnonymousBlock, parent.Children[0].Type)
		assert.Equal(t, BoxBlock, parent.Children[1].Type)
		assert.Equal(t, BoxAnonymousBlock, parent.Children[2].Type)

		assert.Len(t, parent.Children[0].Children, 1)
		assert.Equal(t, inline1, parent.Children[0].Children[0])
		assert.Len(t, parent.Children[2].Children, 1)
		assert.Equal(t, inline2, parent.Children[2].Children[0])
	})
}

func TestGenerateListMarker(t *testing.T) {
	t.Run("outside position creates BoxListMarker", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.ListStylePosition = ListStylePositionOutside
		listItem := &LayoutBox{
			Type:  BoxListItem,
			Style: style,
		}

		generateListMarker(listItem, "1. ")

		assert.Len(t, listItem.Children, 1)
		marker := listItem.Children[0]
		assert.Equal(t, BoxListMarker, marker.Type)
		assert.Equal(t, "1. ", marker.Text)
		assert.Equal(t, listItem, marker.Parent)
	})

	t.Run("inside position creates BoxTextRun with IsListMarker", func(t *testing.T) {
		style := DefaultComputedStyle()
		style.ListStylePosition = ListStylePositionInside
		listItem := &LayoutBox{
			Type:  BoxListItem,
			Style: style,
		}

		generateListMarker(listItem, "\u2022 ")

		assert.Len(t, listItem.Children, 1)
		marker := listItem.Children[0]
		assert.Equal(t, BoxTextRun, marker.Type)
		assert.True(t, marker.IsListMarker)
		assert.Equal(t, "\u2022 ", marker.Text)
		assert.Equal(t, listItem, marker.Parent)
	})
}

func TestBoxTreeBuilder_CounterOperations(t *testing.T) {
	t.Run("processCounterReset pushes values onto stacks", func(t *testing.T) {
		b := &boxTreeBuilder{}
		style := &ComputedStyle{
			CounterReset: []CounterEntry{{Name: "section", Value: 0}},
		}

		resetCount := b.processCounterReset(style)

		assert.Equal(t, 1, resetCount)
		assert.Equal(t, []int{0}, b.counters["section"])
	})

	t.Run("processCounterIncrement increments top of stack", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{
				"section": {0},
			},
		}
		style := &ComputedStyle{
			CounterIncrement: []CounterEntry{{Name: "section", Value: 1}},
		}

		b.processCounterIncrement(style)

		assert.Equal(t, []int{1}, b.counters["section"])
	})

	t.Run("processCounterIncrement creates implicit reset when stack is empty", func(t *testing.T) {
		b := &boxTreeBuilder{}
		style := &ComputedStyle{
			CounterIncrement: []CounterEntry{{Name: "item", Value: 5}},
		}

		b.processCounterIncrement(style)

		assert.Equal(t, []int{5}, b.counters["item"])
	})

	t.Run("popCounterResets undoes resets", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{
				"section": {0, 3},
			},
		}
		style := &ComputedStyle{
			CounterReset: []CounterEntry{{Name: "section", Value: 3}},
		}

		b.popCounterResets(style, 1)

		assert.Equal(t, []int{0}, b.counters["section"])
	})

	t.Run("resolveCounterFunc returns current counter value", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{
				"section": {1, 4},
			},
		}

		result := b.resolveCounterFunc("section")
		assert.Equal(t, "4", result)
	})

	t.Run("resolveCounterFunc returns 0 for unknown counter", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{},
		}

		result := b.resolveCounterFunc("missing")
		assert.Equal(t, "0", result)
	})

	t.Run("resolveCountersFunc joins all stack values", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{
				"section": {1, 2, 3},
			},
		}

		result := b.resolveCountersFunc("section")
		assert.Equal(t, "1.2.3", result)
	})

	t.Run("resolveCountersFunc uses custom separator", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{
				"section": {1, 2, 3},
			},
		}

		result := b.resolveCountersFunc(`section, "-"`)
		assert.Equal(t, "1-2-3", result)
	})

	t.Run("resolveCountersFunc returns 0 for unknown counter", func(t *testing.T) {
		b := &boxTreeBuilder{
			counters: map[string][]int{},
		}

		result := b.resolveCountersFunc("missing")
		assert.Equal(t, "0", result)
	})
}
