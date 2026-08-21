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

package compiler_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestNewVDOMBuilder(t *testing.T) {
	t.Run("creates VDOMBuilder", func(t *testing.T) {
		builder := NewVDOMBuilder()
		require.NotNil(t, builder)
		var _ = builder
	})
}

func TestVDOMBuilder_BuildRenderVDOM(t *testing.T) {
	ctx := context.Background()

	t.Run("nil template AST returns comment node", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		result, err := builder.BuildRenderVDOM(ctx, nil, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.Fn.Body.Block.Stmts)
	})

	t.Run("empty root nodes returns comment node", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("single text node", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "Hello World",
					Key:         &ast_domain.StringLiteral{Value: "0"},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Fn.Body.Block.Stmts, 1)
		returnStmt, ok := result.Fn.Body.Block.Stmts[0].Data.(*js_ast.SReturn)
		require.True(t, ok)
		require.NotNil(t, returnStmt.ValueOrNil.Data)
	})

	t.Run("single element node", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Key:      &ast_domain.StringLiteral{Value: "0"},
					Children: []*ast_domain.TemplateNode{},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("element with text child", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Key:      &ast_domain.StringLiteral{Value: "0"},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Hello",
							Key:         &ast_domain.StringLiteral{Value: "0_0"},
						},
					},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("multiple root nodes creates fragment", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Key:      &ast_domain.StringLiteral{Value: "0"},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "span",
					Key:      &ast_domain.StringLiteral{Value: "1"},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Fn.Body.Block.Stmts, 1)
		returnStmt, ok := result.Fn.Body.Block.Stmts[0].Data.(*js_ast.SReturn)
		require.True(t, ok)
		call, ok := returnStmt.ValueOrNil.Data.(*js_ast.ECall)
		require.True(t, ok)
		require.NotNil(t, call)
	})

	t.Run("comment node", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeComment,
					TextContent: "This is a comment",
					Key:         &ast_domain.StringLiteral{Value: "0"},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("fragment node", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeFragment,
					Key:      &ast_domain.StringLiteral{Value: "0"},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Inside fragment",
							Key:         &ast_domain.StringLiteral{Value: "0_0"},
						},
					},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("element with attributes", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Key:      &ast_domain.StringLiteral{Value: "0"},
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "container"},
						{Name: "id", Value: "main"},
					},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("element with boolean prop", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "button",
					Key:      &ast_domain.StringLiteral{Value: "0"},
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "disabled", Value: ""},
					},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{"disabled"}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("nested elements", func(t *testing.T) {
		builder := NewVDOMBuilder()
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		tmplAST := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Key:      &ast_domain.StringLiteral{Value: "0"},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "ul",
							Key:      &ast_domain.StringLiteral{Value: "0_0"},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "li",
									Key:      &ast_domain.StringLiteral{Value: "0_0_0"},
									Children: []*ast_domain.TemplateNode{
										{
											NodeType:    ast_domain.NodeText,
											TextContent: "Item 1",
											Key:         &ast_domain.StringLiteral{Value: "0_0_0_0"},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result, err := builder.BuildRenderVDOM(ctx, tmplAST, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestBuildFinalExpr(t *testing.T) {
	t.Run("empty slice returns null", func(t *testing.T) {
		result := buildFinalExpr([]js_ast.Expr{})

		_, ok := result.Data.(*js_ast.ENull)
		assert.True(t, ok)
	})

	t.Run("single expression returns it directly", func(t *testing.T) {
		input := newStringLiteral("test")
		result := buildFinalExpr([]js_ast.Expr{input})

		strExpr, ok := result.Data.(*js_ast.EString)
		assert.True(t, ok)
		assert.NotNil(t, strExpr)
	})

	t.Run("multiple expressions creates fragment", func(t *testing.T) {
		inputs := []js_ast.Expr{
			newStringLiteral("one"),
			newStringLiteral("two"),
		}
		result := buildFinalExpr(inputs)

		call, ok := result.Data.(*js_ast.ECall)
		assert.True(t, ok)
		assert.NotNil(t, call)
	})
}

func TestBuildNodeAST(t *testing.T) {
	ctx := context.Background()

	t.Run("text node", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "Hello World",
			Key:         &ast_domain.StringLiteral{Value: "0"},
		}

		result, err := buildNodeAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
		_, ok := result.Data.(*js_ast.ECall)
		assert.True(t, ok)
	})

	t.Run("whitespace-only text node", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "   \n\t   ",
			Key:         &ast_domain.StringLiteral{Value: "0"},
		}

		result, err := buildNodeAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("comment node", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeComment,
			TextContent: "A comment",
			Key:         &ast_domain.StringLiteral{Value: "0"},
		}

		result, err := buildNodeAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("element node", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Key:      &ast_domain.StringLiteral{Value: "0"},
		}

		result, err := buildNodeAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("fragment node", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeFragment,
			Key:      &ast_domain.StringLiteral{Value: "0"},
			Children: []*ast_domain.TemplateNode{},
		}

		result, err := buildNodeAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("unknown node type returns null", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: 999,
			Key:      &ast_domain.StringLiteral{Value: "0"},
		}

		result, err := buildNodeAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
		_, ok := result.Data.(*js_ast.ENull)
		assert.True(t, ok)
	})
}

func TestBuildForLoopAST(t *testing.T) {
	ctx := context.Background()

	t.Run("basic for loop", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  &ast_domain.Identifier{Name: "item"},
					IndexVariable: &ast_domain.Identifier{Name: "i"},
					Collection:    &ast_domain.Identifier{Name: "items"},
				},
			},
		}

		result, err := buildForLoopAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
		_, ok := result.Data.(*js_ast.ECall)
		assert.True(t, ok)
	})

	t.Run("for loop with existing loop vars", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		outerVars := map[string]bool{
			"outerItem": true,
		}

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{
					ItemVariable:  &ast_domain.Identifier{Name: "item"},
					IndexVariable: &ast_domain.Identifier{Name: "i"},
					Collection:    &ast_domain.Identifier{Name: "items"},
				},
			},
		}

		result, err := buildForLoopAST(ctx, node, &nodeBuildContext{events: events, loopVars: outerVars, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("for loop with default variable names", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			DirFor: &ast_domain.Directive{
				Type: ast_domain.DirectiveFor,
				Expression: &ast_domain.ForInExpression{

					Collection: &ast_domain.Identifier{Name: "items"},
				},
			},
		}

		result, err := buildForLoopAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		require.NoError(t, err)
		require.NotNil(t, result.Data)
	})

	t.Run("invalid for expression type returns error", func(t *testing.T) {
		registry := NewRegistryContext()
		events := newEventBindingCollection(registry)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "li",
			Key:      &ast_domain.StringLiteral{Value: "0"},
			DirFor: &ast_domain.Directive{
				Type:       ast_domain.DirectiveFor,
				Expression: &ast_domain.StringLiteral{Value: "not a for expr"},
			},
		}

		_, err := buildForLoopAST(ctx, node, &nodeBuildContext{events: events, booleanProps: []string{}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ForInExpr")
	})
}

func TestPickTagName(t *testing.T) {
	t.Run("regular tag", func(t *testing.T) {
		node := &ast_domain.TemplateNode{TagName: "div"}
		result := pickTagName(node, false)
		assert.Equal(t, "div", result)
	})

	t.Run("piko:a becomes a", func(t *testing.T) {
		node := &ast_domain.TemplateNode{TagName: "piko:a"}
		result := pickTagName(node, true)
		assert.Equal(t, "a", result)
	})

	t.Run("custom element unchanged", func(t *testing.T) {
		node := &ast_domain.TemplateNode{TagName: "my-component"}
		result := pickTagName(node, false)
		assert.Equal(t, "my-component", result)
	})
}

func TestBuildDOMCall(t *testing.T) {
	t.Run("creates dom method call", func(t *testing.T) {
		result := buildDOMCall("txt", newStringLiteral("hello"))

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok)
		assert.Equal(t, "txt", dot.Name)
		assert.Len(t, call.Args, 1)
	})

	t.Run("handles multiple arguments", func(t *testing.T) {
		result := buildDOMCall("el",
			newStringLiteral("div"),
			newStringLiteral("key"),
			newNullLiteral(),
		)

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		assert.Len(t, call.Args, 3)
	})

	t.Run("handles no arguments", func(t *testing.T) {
		result := buildDOMCall("method")

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)
		assert.Empty(t, call.Args)
	})
}

func TestBuildMethodCallOnExpr(t *testing.T) {
	t.Run("creates method call on target", func(t *testing.T) {
		target := newIdentifier("myObj")
		result := buildMethodCallOnExpr(target, "doSomething", newStringLiteral("argument"))

		call, ok := result.Data.(*js_ast.ECall)
		require.True(t, ok)

		dot, ok := call.Target.Data.(*js_ast.EDot)
		require.True(t, ok)
		assert.Equal(t, "doSomething", dot.Name)
		assert.Len(t, call.Args, 1)
	})
}

func TestVdomBuilder_SingleLiteralRichTextKeepsItsText(t *testing.T) {
	t.Parallel()

	registry := NewRegistryContext()
	key := newStringLiteral("k")

	tests := []struct {
		name         string
		parts        []ast_domain.TextPart
		wantCall     string
		wantText     string
		wantTextSeen bool
	}{
		{
			name:         "literal after an empty interpolation survives",
			parts:        []ast_domain.TextPart{richLiteral("foo")},
			wantCall:     "txt",
			wantText:     "foo",
			wantTextSeen: true,
		},
		{
			name:         "literal before an empty interpolation survives",
			parts:        []ast_domain.TextPart{richLiteral("Hello ")},
			wantCall:     "txt",
			wantText:     "Hello ",
			wantTextSeen: true,
		},
		{
			name:         "whitespace-only literal still collapses to a placeholder",
			parts:        []ast_domain.TextPart{richLiteral("  ")},
			wantCall:     "ws",
			wantTextSeen: false,
		},
		{
			name:         "internal whitespace is squashed as it is for plain text",
			parts:        []ast_domain.TextPart{richLiteral("a\n  b")},
			wantCall:     "txt",
			wantText:     "a b",
			wantTextSeen: true,
		},
		{
			name:         "a real interpolation still takes the general path",
			parts:        []ast_domain.TextPart{richExpression("state.value")},
			wantCall:     "txt",
			wantTextSeen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			node := makeRichTextNode(tt.parts, "k")
			result, err := buildRichTextNodeAST(node, key, registry)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCall, domCallName(t, result))

			text, isLiteral := domCallStringArg(t, result)
			if !tt.wantTextSeen {
				return
			}
			require.True(t, isLiteral, "expected a string literal argument")
			assert.Equal(t, tt.wantText, text)
		})
	}
}

func TestVdomBuilder_TextNodeReadsAllCarriers(t *testing.T) {
	t.Parallel()

	key := newStringLiteral("k")

	tests := []struct {
		node         *ast_domain.TemplateNode
		name         string
		wantCall     string
		wantText     string
		wantTextSeen bool
	}{
		{
			name:         "plain text",
			node:         makeTextNode("hello", "k"),
			wantCall:     "txt",
			wantText:     "hello",
			wantTextSeen: true,
		},
		{
			name:         "text held in rich text parts",
			node:         makeRichTextNode([]ast_domain.TextPart{richLiteral("from rich")}, "k"),
			wantCall:     "txt",
			wantText:     "from rich",
			wantTextSeen: true,
		},
		{
			name: "text held in a writer",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				TextContentWriter: writerHoldingText("from writer"),
				Key:               &ast_domain.StringLiteral{Value: "k"},
			},
			wantCall:     "txt",
			wantText:     "from writer",
			wantTextSeen: true,
		},
		{
			name:     "whitespace-only text becomes a placeholder",
			node:     makeTextNode("   ", "k"),
			wantCall: "ws",
		},
		{
			name: "preserved whitespace is emitted verbatim",
			node: &ast_domain.TemplateNode{
				NodeType:           ast_domain.NodeText,
				TextContent:        "\n  ",
				PreserveWhitespace: true,
				Key:                &ast_domain.StringLiteral{Value: "k"},
			},
			wantCall:     "txt",
			wantText:     "\n  ",
			wantTextSeen: true,
		},
		{
			name: "preserved but genuinely empty stays a placeholder",
			node: &ast_domain.TemplateNode{
				NodeType:           ast_domain.NodeText,
				TextContent:        "",
				PreserveWhitespace: true,
				Key:                &ast_domain.StringLiteral{Value: "k"},
			},
			wantCall: "ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := buildTextNodeAST(tt.node, key)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCall, domCallName(t, result))

			if !tt.wantTextSeen {
				assert.Equal(t, 1, domCallArgCount(t, result),
					"a placeholder call carries only the key, never text")
				return
			}

			text, isLiteral := domCallStringArg(t, result)
			require.True(t, isLiteral)
			assert.Equal(t, tt.wantText, text)
		})
	}
}

func TestCompileSFC_PHTMLWritesToTheElementItself(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sfcContent  string
		wantKept    []string
		wantDropped []string
	}{
		{
			name: "p-html on a plain element carries the markup as the element value",
			sfcContent: `<template name="html-plain"><div p-html="state.markup"></div></template>
<script lang="ts">
const state = { markup: "<em>hi</em>" };
</script>`,
			wantKept:    []string{`dom.el("div"`, `[], String(this.$$ctx.state.markup)`},
			wantDropped: []string{"dom.html("},
		},
		{
			name: "p-html inside svg keeps the svg element as the target",
			sfcContent: `<template name="html-svg"><svg viewBox="0 0 10 10" p-html="state.shapes"></svg></template>
<script lang="ts">
const state = { shapes: "<circle r='4'></circle>" };
</script>`,
			wantKept:    []string{`dom.el("svg"`, `String(this.$$ctx.state.shapes)`},
			wantDropped: []string{"dom.html("},
		},
		{
			name: "p-html on a static piko:element target",
			sfcContent: `<template name="html-piko-static"><piko:element is="section" p-html="state.markup"></piko:element></template>
<script lang="ts">
const state = { markup: "<b>x</b>" };
</script>`,
			wantKept:    []string{`dom.el("section"`, `String(this.$$ctx.state.markup)`},
			wantDropped: []string{"dom.html("},
		},
		{
			name: "p-html on a dynamic piko:element target keeps the module name argument",
			sfcContent: `<template name="html-piko-dynamic"><piko:element :is="state.tag" p-html="state.markup"></piko:element></template>
<script lang="ts">
const state = { tag: "section", markup: "<b>x</b>" };
</script>`,
			wantKept:    []string{"dom.pikoEl(", `String(this.$$ctx.state.markup)`},
			wantDropped: []string{"dom.html("},
		},
		{
			name: "p-text is unaffected and still builds a text child",
			sfcContent: `<template name="text-plain"><div p-text="state.label"></div></template>
<script lang="ts">
const state = { label: "hi" };
</script>`,
			wantKept:    []string{"dom.txt("},
			wantDropped: []string{"dom.html("},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orchestrator := NewCompilerOrchestrator(nil, nil, WithOrchestratorModuleName("example.com/proj"))
			artefact, err := orchestrator.CompileSFCBytes(context.Background(), "phtml.pkc", []byte(tt.sfcContent))
			require.NoError(t, err)
			require.NotNil(t, artefact)

			js := artefact.Files[artefact.BaseJSPath]
			for _, kept := range tt.wantKept {
				assert.Contains(t, js, kept, "expected %q in the emitted JS", kept)
			}
			for _, dropped := range tt.wantDropped {
				assert.NotContains(t, js, dropped, "expected %q to be gone from the emitted JS", dropped)
			}
		})
	}
}
