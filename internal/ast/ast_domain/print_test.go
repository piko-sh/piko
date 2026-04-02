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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_treeToString(t *testing.T) {
	t.Parallel()

	t.Run("nil tree returns empty message", func(t *testing.T) {
		t.Parallel()

		result := treeToString(nil)
		assert.Equal(t, "<empty tree>", result)
	})

	t.Run("empty tree returns empty message", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{}
		result := treeToString(tree)
		assert.Equal(t, "<empty tree>", result)
	})

	t.Run("simple element", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Location: Location{Line: 1, Column: 1},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "AST Tree")
		assert.Contains(t, result, "Element: <div>")
		assert.Contains(t, result, "L1:C1")
	})

	t.Run("text node with content", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeText,
					TextContent: "Hello World",
					Location:    Location{Line: 1, Column: 1},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Text (L1:C1)")
		assert.Contains(t, result, `Content: "Hello World"`)
	})

	t.Run("text node with whitespace only", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeText,
					TextContent: "   ",
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Text (whitespace only)")
	})

	t.Run("comment node", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeComment,
					TextContent: "This is a comment",
					Location:    Location{Line: 5, Column: 3},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Comment (L5:C3)")
	})

	t.Run("unknown node type", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeType(99),
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Unknown Node Type")
	})

	t.Run("node with attributes", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "container", Location: Location{Line: 1, Column: 5}},
						{Name: "id", Value: "main", Location: Location{Line: 1, Column: 20}},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Attributes:")
		assert.Contains(t, result, `class="container"`)
		assert.Contains(t, result, `id="main"`)
	})

	t.Run("node with dynamic attributes", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DynamicAttributes: []DynamicAttribute{
						{
							Name:          "class",
							RawExpression: "className",
							Expression:    &Identifier{Name: "className"},
							Location:      Location{Line: 1, Column: 5},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Dynamic Attributes:")
		assert.Contains(t, result, `:class="className"`)
		assert.Contains(t, result, "-> className")
	})

	t.Run("node with dynamic attribute nil expression", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DynamicAttributes: []DynamicAttribute{
						{
							Name:          "class",
							RawExpression: "className",
							Expression:    nil,
							Location:      Location{Line: 1, Column: 5},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "-> nil")
	})

	t.Run("nested children", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{
							NodeType: NodeElement,
							TagName:  "span",
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Children:")
		assert.Contains(t, result, "<span>")
	})

	t.Run("nil child is skipped", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{nil},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "<div>")
	})

	t.Run("rich text parts", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeText,
					RichText: []TextPart{
						{IsLiteral: true, Literal: "Hello "},
						{IsLiteral: false, Expression: &Identifier{Name: "name"}},
						{IsLiteral: false, Expression: nil},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "RichText Parts: (3)")
		assert.Contains(t, result, "[0] Literal: \"Hello \"")
		assert.Contains(t, result, "[1] Expression: name")
		assert.Contains(t, result, "[2] Expression: nil")
	})

	t.Run("directives p-if", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirIf: &Directive{
						Expression: &Identifier{Name: "isVisible"},
						Location:   Location{Line: 1, Column: 5},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "Directives:")
		assert.Contains(t, result, "p-if: isVisible")
	})

	t.Run("directives p-for", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "ul",
					DirFor: &Directive{
						Expression: &ForInExpression{
							ItemVariable: &Identifier{Name: "item"},
							Collection:   &Identifier{Name: "items"},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-for:")
	})

	t.Run("directives p-show", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirShow:  &Directive{Expression: &BooleanLiteral{Value: true}},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-show: true")
	})

	t.Run("directives p-model", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "input",
					DirModel: &Directive{Expression: &Identifier{Name: "value"}},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-model: value")
	})

	t.Run("directives p-text", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "span",
					DirText:  &Directive{Expression: &Identifier{Name: "message"}},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-text: message")
	})

	t.Run("directives p-html", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirHTML:  &Directive{Expression: &Identifier{Name: "htmlContent"}},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-html: htmlContent")
	})

	t.Run("directives p-class", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirClass: &Directive{Expression: &Identifier{Name: "className"}},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-class: className")
	})

	t.Run("directives p-style", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirStyle: &Directive{Expression: &Identifier{Name: "styles"}},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-style: styles")
	})

	t.Run("directives p-ref", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "input",
					DirRef:   &Directive{RawExpression: "inputRef"},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-ref: inputRef")
	})

	t.Run("p-bind directives", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "input",
					Binds: map[string]*Directive{
						"value": {Expression: &Identifier{Name: "inputValue"}},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-bind:value:")
	})

	t.Run("p-on event directives", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "button",
					OnEvents: map[string][]Directive{
						"click": {
							{
								Expression: &CallExpression{Callee: &Identifier{Name: "handleClick"}},
							},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-on:click:")
	})

	t.Run("p-on event with modifier", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "button",
					OnEvents: map[string][]Directive{
						"click": {
							{
								Expression: &CallExpression{Callee: &Identifier{Name: "handleClick"}},
								Modifier:   "prevent",
							},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-on:click.prevent:")
	})

	t.Run("p-event custom event directives", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "my-component",
					CustomEvents: map[string][]Directive{
						"submit": {
							{
								Expression: &CallExpression{Callee: &Identifier{Name: "onSubmit"}},
							},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-event:submit:")
	})

	t.Run("p-event with modifier", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "my-component",
					CustomEvents: map[string][]Directive{
						"submit": {
							{
								Expression: &CallExpression{Callee: &Identifier{Name: "onSubmit"}},
								Modifier:   "once",
							},
						},
					},
				},
			},
		}
		result := treeToString(tree)

		assert.Contains(t, result, "p-event:submit.once:")
	})
}

func TestBuildModifierString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		modifier string
		expected string
	}{
		{name: "empty modifier", modifier: "", expected: ""},
		{name: "prevent modifier", modifier: "prevent", expected: ".prevent"},
		{name: "stop modifier", modifier: "stop", expected: ".stop"},
		{name: "once modifier", modifier: "once", expected: ".once"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := buildModifierString(tc.modifier)
			assert.Equal(t, tc.expected, result)
		})
	}
}
