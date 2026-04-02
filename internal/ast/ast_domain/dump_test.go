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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumpAST(t *testing.T) {
	t.Parallel()

	t.Run("nil AST returns nil message", func(t *testing.T) {
		t.Parallel()

		result := DumpAST(context.Background(), nil)
		assert.Equal(t, "/* AST is nil */\n", result)
	})

	t.Run("empty AST", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "BEGIN AST DUMP")
		assert.Contains(t, result, "END AST DUMP")
	})

	t.Run("simple element node", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<div />")
	})

	t.Run("element with attributes", func(t *testing.T) {
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
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<div")
		assert.Contains(t, result, `class="container"`)
		assert.Contains(t, result, `id="main"`)
	})

	t.Run("element with children", func(t *testing.T) {
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
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<div>")
		assert.Contains(t, result, "<span />")
		assert.Contains(t, result, "</div>")
	})

	t.Run("text node with content", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeText,
					TextContent: "Hello World",
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "Hello World")
	})

	t.Run("text node with whitespace only is trimmed", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeText,
					TextContent: "   ",
				},
			},
		}
		result := DumpAST(context.Background(), tree)

		assert.NotContains(t, result, "   ")
	})

	t.Run("comment node", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeComment,
					TextContent: "This is a comment",
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<!-- This is a comment -->")
	})

	t.Run("fragment node empty", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeFragment,
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<Fragment />")
	})

	t.Run("fragment node with children", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeFragment,
					Children: []*TemplateNode{
						{NodeType: NodeElement, TagName: "p"},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<Fragment>")
		assert.Contains(t, result, "<p />")
		assert.Contains(t, result, "</Fragment>")
	})

	t.Run("fragment node with annotations", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeFragment,
					GoAnnotations: &GoGeneratorAnnotation{
						OriginalPackageAlias: new("partials_header"),
						PartialInfo: &PartialInvocationInfo{
							InvocationKey:       "inv-123",
							PartialPackageName:  "partials/header",
							InvokerPackageAlias: "main",
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "[ANNOTATIONS:")
		assert.Contains(t, result, "OriginPackage: partials_header")
		assert.Contains(t, result, "PARTIAL InvKey: inv-123")
	})

	t.Run("rich text with literal and expression", func(t *testing.T) {
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
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "<RichText>")
		assert.Contains(t, result, `"Hello "`)
		assert.Contains(t, result, "{{ name }}")
		assert.Contains(t, result, "</RichText>")
	})

	t.Run("dynamic attributes", func(t *testing.T) {
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
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, `:class="className"`)
		assert.Contains(t, result, "{P: className}")
	})

	t.Run("dynamic attributes with origin package", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DynamicAttributes: []DynamicAttribute{
						{
							Name:          "href",
							RawExpression: "url",
							Expression:    &Identifier{Name: "url"},
						},
					},
					GoAnnotations: &GoGeneratorAnnotation{
						DynamicAttributeOrigins: map[string]string{
							"href": "partials_nav",
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "OriginPackage: partials_nav")
	})

	t.Run("directives", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					DirIf:    &Directive{Expression: &Identifier{Name: "isVisible"}},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "[p-if: isVisible]")
	})

	t.Run("multiple directives", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "ul",
					DirFor:   &Directive{Expression: &ForInExpression{ItemVariable: &Identifier{Name: "item"}, Collection: &Identifier{Name: "items"}}},
					DirKey:   &Directive{Expression: &MemberExpression{Base: &Identifier{Name: "item"}, Property: &Identifier{Name: "id"}}},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "[p-for:")
		assert.Contains(t, result, "[p-key:")
	})

	t.Run("events", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "button",
					OnEvents: map[string][]Directive{
						"click": {
							{
								Modifier:      "",
								RawExpression: "handleClick()",
								Expression:    &CallExpression{Callee: &Identifier{Name: "handleClick"}},
							},
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "[Events:")
		assert.Contains(t, result, "p-on:click")
	})

	t.Run("custom events", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "my-component",
					CustomEvents: map[string][]Directive{
						"submit": {
							{
								RawExpression: "onSubmit()",
								Expression:    &CallExpression{Callee: &Identifier{Name: "onSubmit"}},
							},
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "[Events:")
		assert.Contains(t, result, "p-event:submit")
	})

	t.Run("annotations on element", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					GoAnnotations: &GoGeneratorAnnotation{
						OriginalPackageAlias: new("main"),
						OriginalSourcePath:   new("pages/home.pkc"),
						GeneratedSourcePath:  new("dist/pages/home.go"),
						ParentTypeName:       new("HomePageState"),
						NeedsCSRF:            true,
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "[ANNOTATIONS:")
		assert.Contains(t, result, "OriginPackage: main")
		assert.Contains(t, result, "OriginPath: pages/home.pkc")
		assert.Contains(t, result, "GenPath: dist/pages/home.go")
		assert.Contains(t, result, "ParentType: HomePageState")
		assert.Contains(t, result, "NeedsCSRF")
	})

	t.Run("annotations with resolved symbol", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "span",
					GoAnnotations: &GoGeneratorAnnotation{
						Symbol: &ResolvedSymbol{
							Name:                "Message",
							ReferenceLocation:   Location{Line: 15, Column: 10},
							DeclarationLocation: Location{Line: 5, Column: 2},
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, "Symbol: Message")
		assert.Contains(t, result, "definition @ L15:C10")
		assert.Contains(t, result, "gen @ L5:C2")
	})

	t.Run("escapes special characters", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType:    NodeText,
					TextContent: "Line1\nLine2",
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, `Line1\nLine2`)
	})

	t.Run("escapes quotes in attributes", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Attributes: []HTMLAttribute{
						{Name: "title", Value: `Say "Hello"`},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)
		assert.Contains(t, result, `title="Say \"Hello\""`)
	})

	t.Run("nested elements with indentation", func(t *testing.T) {
		t.Parallel()

		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Children: []*TemplateNode{
						{
							NodeType: NodeElement,
							TagName:  "ul",
							Children: []*TemplateNode{
								{
									NodeType: NodeElement,
									TagName:  "li",
								},
							},
						},
					},
				},
			},
		}
		result := DumpAST(context.Background(), tree)

		lines := strings.Split(result, "\n")
		var divLine, ulLine, liLine string
		for _, line := range lines {
			if strings.Contains(line, "<div>") {
				divLine = line
			}
			if strings.Contains(line, "<ul>") {
				ulLine = line
			}
			if strings.Contains(line, "<li") {
				liLine = line
			}
		}
		require.NotEmpty(t, divLine)
		require.NotEmpty(t, ulLine)
		require.NotEmpty(t, liLine)

		divIndent := len(divLine) - len(strings.TrimLeft(divLine, " "))
		ulIndent := len(ulLine) - len(strings.TrimLeft(ulLine, " "))
		liIndent := len(liLine) - len(strings.TrimLeft(liLine, " "))

		assert.Less(t, divIndent, ulIndent)
		assert.Less(t, ulIndent, liIndent)
	})

	t.Run("nil node in children is skipped", func(t *testing.T) {
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
		result := DumpAST(context.Background(), tree)

		assert.Contains(t, result, "<div>")
		assert.Contains(t, result, "</div>")
	})
}

func TestDumpNode(t *testing.T) {
	t.Parallel()

	t.Run("handles all node types", func(t *testing.T) {
		t.Parallel()

		nodeTypes := []struct {
			expected string
			nodeType NodeType
		}{
			{nodeType: NodeElement, expected: "<test"},
			{nodeType: NodeFragment, expected: "<Fragment"},
			{nodeType: NodeText, expected: ""},
			{nodeType: NodeComment, expected: "<!--"},
		}

		for _, tc := range nodeTypes {
			var builder strings.Builder
			node := &TemplateNode{
				NodeType:    tc.nodeType,
				TagName:     "test",
				TextContent: "content",
			}
			dumpNode(context.Background(), &builder, node, 0)
			result := builder.String()
			if tc.expected != "" {
				assert.Contains(t, result, tc.expected)
			}
		}
	})
}

func TestEscapeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no escaping needed", input: "hello", expected: "hello"},
		{name: "newline", input: "line1\nline2", expected: `line1\nline2`},
		{name: "quote", input: `say "hi"`, expected: `say \"hi\"`},
		{name: "both", input: "line1\n\"quoted\"", expected: `line1\n\"quoted\"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := escapeString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetNodePackageAlias(t *testing.T) {
	t.Parallel()

	t.Run("nil annotations returns empty", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{}
		assert.Equal(t, "", getNodePackageAlias(node))
	})

	t.Run("nil pkg alias returns empty", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			GoAnnotations: &GoGeneratorAnnotation{},
		}
		assert.Equal(t, "", getNodePackageAlias(node))
	})

	t.Run("returns pkg alias when set", func(t *testing.T) {
		t.Parallel()

		node := &TemplateNode{
			GoAnnotations: &GoGeneratorAnnotation{
				OriginalPackageAlias: new("my_package"),
			},
		}
		assert.Equal(t, "my_package", getNodePackageAlias(node))
	})
}

func TestExpressionToString(t *testing.T) {
	t.Parallel()

	t.Run("nil expression returns <nil>", func(t *testing.T) {
		t.Parallel()

		result := expressionToString(nil)
		assert.Equal(t, "<nil>", result)
	})
}

func TestDumpDirectives(t *testing.T) {
	t.Parallel()

	t.Run("all directive types", func(t *testing.T) {
		t.Parallel()

		directives := []struct {
			setDir func(*TemplateNode)
			name   string
		}{
			{name: "p-if", setDir: func(n *TemplateNode) { n.DirIf = &Directive{Expression: &BooleanLiteral{Value: true}} }},
			{name: "p-else-if", setDir: func(n *TemplateNode) { n.DirElseIf = &Directive{Expression: &BooleanLiteral{Value: true}} }},
			{name: "p-else", setDir: func(n *TemplateNode) { n.DirElse = &Directive{} }},
			{name: "p-for", setDir: func(n *TemplateNode) { n.DirFor = &Directive{Expression: &Identifier{Name: "items"}} }},
			{name: "p-show", setDir: func(n *TemplateNode) { n.DirShow = &Directive{Expression: &BooleanLiteral{Value: true}} }},
			{name: "p-text", setDir: func(n *TemplateNode) { n.DirText = &Directive{Expression: &Identifier{Name: "message"}} }},
			{name: "p-html", setDir: func(n *TemplateNode) { n.DirHTML = &Directive{Expression: &Identifier{Name: "html"}} }},
			{name: "p-class", setDir: func(n *TemplateNode) { n.DirClass = &Directive{Expression: &Identifier{Name: "cls"}} }},
			{name: "p-style", setDir: func(n *TemplateNode) { n.DirStyle = &Directive{Expression: &Identifier{Name: "style"}} }},
			{name: "p-model", setDir: func(n *TemplateNode) { n.DirModel = &Directive{Expression: &Identifier{Name: "value"}} }},
			{name: "p-ref", setDir: func(n *TemplateNode) { n.DirRef = &Directive{RawExpression: "myRef"} }},
			{name: "p-key", setDir: func(n *TemplateNode) { n.DirKey = &Directive{Expression: &Identifier{Name: "id"}} }},
			{name: "p-context", setDir: func(n *TemplateNode) { n.DirContext = &Directive{Expression: &Identifier{Name: "ctx"}} }},
			{name: "p-scaffold", setDir: func(n *TemplateNode) { n.DirScaffold = &Directive{} }},
		}

		for _, tc := range directives {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				var builder strings.Builder
				node := &TemplateNode{NodeType: NodeElement, TagName: "div"}
				tc.setDir(node)
				dumpDirectives(&builder, node, "")
				result := builder.String()
				assert.Contains(t, result, "["+tc.name)
			})
		}
	})

	t.Run("directive with origin package different from node", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		nodePackageAlias := "main"
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			DirIf: &Directive{
				Expression: &BooleanLiteral{Value: true},
				GoAnnotations: &GoGeneratorAnnotation{
					OriginalPackageAlias: new("partial"),
				},
			},
		}
		dumpDirectives(&builder, node, nodePackageAlias)
		result := builder.String()
		assert.Contains(t, result, "OriginPackage: partial")
	})
}

func TestDumpEvents(t *testing.T) {
	t.Parallel()

	t.Run("no events outputs nothing", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		node := &TemplateNode{NodeType: NodeElement, TagName: "div"}
		dumpEvents(&builder, node, "")
		assert.Equal(t, "", builder.String())
	})

	t.Run("on events sorted by name", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "button",
			OnEvents: map[string][]Directive{
				"click":     {{RawExpression: "handleClick"}},
				"mouseover": {{RawExpression: "handleMouseover"}},
			},
		}
		dumpEvents(&builder, node, "")
		result := builder.String()

		clickIndex := strings.Index(result, "click")
		mouseoverIndex := strings.Index(result, "mouseover")
		assert.Less(t, clickIndex, mouseoverIndex)
	})

	t.Run("event with modifier", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "button",
			OnEvents: map[string][]Directive{
				"click": {{Modifier: "prevent", RawExpression: "handleClick"}},
			},
		}
		dumpEvents(&builder, node, "")
		result := builder.String()
		assert.Contains(t, result, "p-on:click.prevent")
	})

	t.Run("event with origin package", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "button",
			OnEvents: map[string][]Directive{
				"click": {{
					RawExpression: "handleClick",
					Expression:    &Identifier{Name: "handleClick"},
					GoAnnotations: &GoGeneratorAnnotation{
						OriginalPackageAlias: new("partial_nav"),
					},
				}},
			},
		}
		dumpEvents(&builder, node, "main")
		result := builder.String()
		assert.Contains(t, result, "OriginPackage: partial_nav")
	})
}

func TestDumpAnnotations(t *testing.T) {
	t.Parallel()

	t.Run("nil annotations outputs nothing", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		node := &TemplateNode{NodeType: NodeElement, TagName: "div"}
		dumpAnnotations(&builder, node)
		assert.Equal(t, "", builder.String())
	})

	t.Run("field tag annotation", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		node := &TemplateNode{
			NodeType: NodeElement,
			TagName:  "div",
			GoAnnotations: &GoGeneratorAnnotation{
				FieldTag: new(`json:"field"`),
			},
		}
		dumpAnnotations(&builder, node)
		result := builder.String()
		assert.Contains(t, result, "Tag: json:\"field\"")
	})
}
