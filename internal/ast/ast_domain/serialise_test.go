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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerialiseASTToGoFileContent(t *testing.T) {
	t.Parallel()

	t.Run("nil tree returns placeholder", func(t *testing.T) {
		t.Parallel()

		result := SerialiseASTToGoFileContent(nil, "testpkg")
		assert.Contains(t, result, "package testpkg")
		assert.Contains(t, result, "AST was nil")
	})

	t.Run("empty tree serialises correctly", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{}
		result := SerialiseASTToGoFileContent(tree, "mypkg")

		assert.Contains(t, result, "package mypkg")
		assert.Contains(t, result, "GeneratedAST")
		assert.Contains(t, result, "ast_domain")
	})

	t.Run("simple tree serialises correctly", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "pages")

		assert.Contains(t, result, "package pages")
		assert.Contains(t, result, "TemplateAST")
		assert.Contains(t, result, "RootNodes")
		assert.Contains(t, result, `"div"`)
	})

	t.Run("tree with attributes", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "container"},
						{Name: "id", Value: "main"},
					},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "Attributes")
		assert.Contains(t, result, `"class"`)
		assert.Contains(t, result, `"container"`)
	})

	t.Run("tree with dynamic attributes", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DynamicAttributes: []DynamicAttribute{
						{
							Name:          "title",
							RawExpression: "pageTitle",
							Expression:    &Identifier{Name: "pageTitle"},
						},
					},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "DynamicAttributes")
		assert.Contains(t, result, `"title"`)
		assert.Contains(t, result, `"pageTitle"`)
	})

	t.Run("tree with directives", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirIf: &Directive{
						Type:          DirectiveIf,
						RawExpression: "isVisible",
						Expression:    &Identifier{Name: "isVisible"},
					},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "DirIf")
		assert.Contains(t, result, "Directive")
	})

	t.Run("tree with nested children", func(t *testing.T) {
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
							Children: []*TemplateNode{
								{
									NodeType:    NodeText,
									TextContent: "Hello",
								},
							},
						},
					},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "Children")
		assert.Contains(t, result, `"span"`)
		assert.Contains(t, result, `"Hello"`)
	})

	t.Run("tree with rich text", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeText,
					RichText: []TextPart{
						{IsLiteral: true, Literal: "Hello "},
						{IsLiteral: false, Expression: &Identifier{Name: "name"}},
					},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "RichText")
		assert.Contains(t, result, "TextPart")
		assert.Contains(t, result, "IsLiteral")
	})

	t.Run("tree with events", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "button",
					OnEvents: map[string][]Directive{
						"click": {
							{
								RawExpression: "handleClick()",
								Expression:    &CallExpression{Callee: &Identifier{Name: "handleClick"}},
							},
						},
					},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "OnEvents")
		assert.Contains(t, result, `"click"`)
	})

	t.Run("tree with diagnostics", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
			Diagnostics: []*Diagnostic{
				{
					Severity:   Warning,
					Message:    "Test warning",
					Expression: "expression",
					Location:   Location{Line: 1, Column: 1},
				},
			},
		}
		result := SerialiseASTToGoFileContent(tree, "test")

		assert.Contains(t, result, "Diagnostics")
	})
}

func TestSerialiseASTString(t *testing.T) {
	t.Parallel()

	t.Run("nil tree", func(t *testing.T) {
		t.Parallel()

		result := SerialiseASTString(nil)
		assert.Equal(t, "/* AST is nil */", result)
	})

	t.Run("empty tree", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{}
		result := SerialiseASTString(tree)
		assert.Contains(t, result, "TemplateAST")
	})

	t.Run("simple tree", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{NodeType: NodeElement, TagName: "div"},
			},
		}
		result := SerialiseASTString(tree)

		assert.Contains(t, result, "TemplateAST")
		assert.Contains(t, result, "RootNodes")
		assert.Contains(t, result, "TemplateNode")
	})
}

func TestSerialiseNodeString(t *testing.T) {
	t.Parallel()

	t.Run("nil node", func(t *testing.T) {
		t.Parallel()

		result := SerialiseNodeString(nil)
		assert.Equal(t, "/* Node is nil */", result)
	})

	t.Run("simple element node", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
		}
		result := SerialiseNodeString(node)

		assert.Contains(t, result, "TemplateNode")
		assert.Contains(t, result, `"div"`)
	})

	t.Run("text node", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			NodeType:    NodeText,
			TextContent: "Hello World",
		}
		result := SerialiseNodeString(node)

		assert.Contains(t, result, "TextContent")
		assert.Contains(t, result, "Hello World")
	})
}

func TestSerialiseAST(t *testing.T) {
	t.Parallel()

	t.Run("nil tree returns nil identifier", func(t *testing.T) {
		t.Parallel()

		result := SerialiseAST(nil)
		require.NotNil(t, result)
	})

	t.Run("tree returns call expression (IIFE)", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{}
		result := SerialiseAST(tree)
		require.NotNil(t, result)
	})
}

func TestSerialiseNode(t *testing.T) {
	t.Parallel()

	t.Run("nil node returns nil identifier", func(t *testing.T) {
		t.Parallel()

		result := SerialiseNode(nil)
		require.NotNil(t, result)
	})

	t.Run("node returns call expression (IIFE)", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{NodeType: NodeElement, TagName: "div"}
		result := SerialiseNode(node)
		require.NotNil(t, result)
	})
}

func TestSerialiseExpressionTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		expression Expression
		contains   []string
	}{
		{
			name:       "Identifier",
			expression: &Identifier{Name: "myVar"},
			contains:   []string{"Identifier", `"myVar"`},
		},
		{
			name:       "StringLiteral",
			expression: &StringLiteral{Value: "hello"},
			contains:   []string{"StringLiteral", `"hello"`},
		},
		{
			name:       "IntegerLiteral",
			expression: &IntegerLiteral{Value: 42},
			contains:   []string{"IntegerLiteral", "42"},
		},
		{
			name:       "FloatLiteral",
			expression: &FloatLiteral{Value: 3.14},
			contains:   []string{"FloatLiteral"},
		},
		{
			name:       "BooleanLiteral true",
			expression: &BooleanLiteral{Value: true},
			contains:   []string{"BooleanLiteral", "true"},
		},
		{
			name:       "BooleanLiteral false",
			expression: &BooleanLiteral{Value: false},
			contains:   []string{"BooleanLiteral", "false"},
		},
		{
			name:       "NilLiteral",
			expression: &NilLiteral{},
			contains:   []string{"NilLiteral"},
		},
		{
			name:       "DecimalLiteral",
			expression: &DecimalLiteral{Value: "123.456"},
			contains:   []string{"DecimalLiteral", `"123.456"`},
		},
		{
			name:       "BigIntLiteral",
			expression: &BigIntLiteral{Value: "12345678901234567890"},
			contains:   []string{"BigIntLiteral"},
		},
		{
			name:       "DateTimeLiteral",
			expression: &DateTimeLiteral{Value: "2024-01-15T10:30:00Z"},
			contains:   []string{"DateTimeLiteral"},
		},
		{
			name:       "DurationLiteral",
			expression: &DurationLiteral{Value: "1h30m"},
			contains:   []string{"DurationLiteral", `"1h30m"`},
		},
		{
			name:       "DateLiteral",
			expression: &DateLiteral{Value: "2024-01-15"},
			contains:   []string{"DateLiteral"},
		},
		{
			name:       "TimeLiteral",
			expression: &TimeLiteral{Value: "10:30:00"},
			contains:   []string{"TimeLiteral"},
		},
		{
			name:       "RuneLiteral",
			expression: &RuneLiteral{Value: 'A'},
			contains:   []string{"RuneLiteral"},
		},
		{
			name: "BinaryExpr",
			expression: &BinaryExpression{
				Left:     &IntegerLiteral{Value: 1},
				Operator: OpPlus,
				Right:    &IntegerLiteral{Value: 2},
			},
			contains: []string{"BinaryExpression", "Left", "Right", "Operator"},
		},
		{
			name: "UnaryExpr",
			expression: &UnaryExpression{
				Operator: OpNot,
				Right:    &BooleanLiteral{Value: true},
			},
			contains: []string{"UnaryExpression", "Operator", "Right"},
		},
		{
			name: "MemberExpr",
			expression: &MemberExpression{
				Base:     &Identifier{Name: "obj"},
				Property: &Identifier{Name: "prop"},
			},
			contains: []string{"MemberExpression", "Base", "Property"},
		},
		{
			name: "IndexExpr",
			expression: &IndexExpression{
				Base:  &Identifier{Name: "arr"},
				Index: &IntegerLiteral{Value: 0},
			},
			contains: []string{"IndexExpression", "Base", "Index"},
		},
		{
			name: "CallExpr",
			expression: &CallExpression{
				Callee: &Identifier{Name: "fn"},
				Args:   []Expression{&IntegerLiteral{Value: 1}},
			},
			contains: []string{"CallExpression", "Callee", "Args"},
		},
		{
			name: "TernaryExpr",
			expression: &TernaryExpression{
				Condition:  &BooleanLiteral{Value: true},
				Consequent: &StringLiteral{Value: "yes"},
				Alternate:  &StringLiteral{Value: "no"},
			},
			contains: []string{"TernaryExpression", "Condition", "Consequent", "Alternate"},
		},
		{
			name: "ForInExpr",
			expression: &ForInExpression{
				ItemVariable: &Identifier{Name: "item"},
				Collection:   &Identifier{Name: "items"},
			},
			contains: []string{"ForInExpression", "ItemVariable", "Collection"},
		},
		{
			name: "ArrayLiteral",
			expression: &ArrayLiteral{
				Elements: []Expression{
					&IntegerLiteral{Value: 1},
					&IntegerLiteral{Value: 2},
				},
			},
			contains: []string{"ArrayLiteral", "Elements"},
		},
		{
			name: "ObjectLiteral",
			expression: &ObjectLiteral{
				Pairs: map[string]Expression{
					"key": &StringLiteral{Value: "value"},
				},
			},
			contains: []string{"ObjectLiteral", "Pairs"},
		},
		{
			name: "TemplateLiteral",
			expression: &TemplateLiteral{
				Parts: []TemplateLiteralPart{
					{IsLiteral: true, Literal: "Hello "},
					{IsLiteral: false, Expression: &Identifier{Name: "name"}},
				},
			},
			contains: []string{"TemplateLiteral", "Parts"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tree := &TemplateAST{
				RootNodes: []*TemplateNode{
					{
						NodeType: NodeElement,
						TagName:  "div",
						DirText:  &Directive{Expression: tc.expression},
					},
				},
			}
			result := SerialiseASTString(tree)

			for _, expected := range tc.contains {
				assert.Contains(t, result, expected, "Expected result to contain %q", expected)
			}
		})
	}
}

func TestSerialiseGoAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("node with GoAnnotations", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					GoAnnotations: &GoGeneratorAnnotation{
						OriginalPackageAlias: new("main"),
						OriginalSourcePath:   new("pages/home.pkc"),
						IsStatic:             true,
						NeedsCSRF:            true,
					},
				},
			},
		}
		result := SerialiseASTString(tree)

		assert.Contains(t, result, "GoAnnotations")
		assert.Contains(t, result, "OriginalPackageAlias")
		assert.Contains(t, result, "IsStatic")
		assert.Contains(t, result, "NeedsCSRF")
	})

	t.Run("expression with GoAnnotations", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirText: &Directive{
						Expression: &Identifier{
							Name: "value",
							GoAnnotations: &GoGeneratorAnnotation{
								IsStatic:      true,
								Stringability: 2,
							},
						},
					},
				},
			},
		}
		result := SerialiseASTString(tree)

		assert.Contains(t, result, "GoAnnotations")
	})
}

func TestSerialiseComplexTree(t *testing.T) {
	t.Parallel()

	t.Run("complex tree with multiple node types", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "class", Value: "container"},
					},
					DirIf: &Directive{
						Expression: &Identifier{Name: "showContent"},
					},
					Children: []*TemplateNode{
						{
							NodeType: NodeElement,
							TagName:  "h1",
							DirText: &Directive{
								Expression: &Identifier{Name: "title"},
							},
						},
						{
							NodeType: NodeElement,
							TagName:  "ul",
							DirFor: &Directive{
								Expression: &ForInExpression{
									ItemVariable: &Identifier{Name: "item"},
									Collection:   &Identifier{Name: "items"},
								},
							},
							Children: []*TemplateNode{
								{
									NodeType: NodeElement,
									TagName:  "li",
									DirText: &Directive{
										Expression: &MemberExpression{
											Base:     &Identifier{Name: "item"},
											Property: &Identifier{Name: "name"},
										},
									},
								},
							},
						},
						{
							NodeType:    NodeText,
							TextContent: "Static text",
						},
						{
							NodeType:    NodeComment,
							TextContent: "This is a comment",
						},
					},
				},
				{
					NodeType: NodeFragment,
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "span"},
					},
				},
			},
		}

		result := SerialiseASTToGoFileContent(tree, "pages")

		assert.Contains(t, result, "package pages")
		assert.Contains(t, result, "GeneratedAST")

		assert.Contains(t, result, `"div"`)
		assert.Contains(t, result, `"h1"`)
		assert.Contains(t, result, `"ul"`)
		assert.Contains(t, result, `"li"`)
		assert.Contains(t, result, `"span"`)

		assert.Contains(t, result, "DirIf")
		assert.Contains(t, result, "DirFor")
		assert.Contains(t, result, "DirText")

		assert.Contains(t, result, "Static text")

		assert.True(t, strings.Contains(result, "func()"))
	})
}

func TestSerialiseRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("serialised tree preserves structure", func(t *testing.T) {
		t.Parallel()

		original := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "id", Value: "main"},
					},
					Children: []*TemplateNode{
						{
							NodeType: NodeElement,
							TagName:  "p",
							DirText: &Directive{
								Expression: &Identifier{Name: "message"},
							},
						},
					},
				},
			},
		}

		serialised := SerialiseASTString(original)

		assert.Contains(t, serialised, "RootNodes")
		assert.Contains(t, serialised, "Children")
		assert.Contains(t, serialised, `"div"`)
		assert.Contains(t, serialised, `"p"`)
		assert.Contains(t, serialised, `"main"`)
		assert.Contains(t, serialised, "Identifier")
		assert.Contains(t, serialised, `"message"`)
	})
}
