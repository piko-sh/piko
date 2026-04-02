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

package ast_adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEncodeDecodeAST_EmptyAST(t *testing.T) {
	t.Run("nil AST encodes to valid bytes with schema hash", func(t *testing.T) {
		encoded, err := EncodeAST(nil)
		require.NoError(t, err)

		assert.NotEmpty(t, encoded)
	})

	t.Run("empty AST with no root nodes", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{},
		}

		encoded, err := EncodeAST(original)
		require.NoError(t, err)
		require.NotEmpty(t, encoded)

		decoded, err := DecodeAST(context.Background(), encoded)
		require.NoError(t, err)
		require.NotNil(t, decoded)
		assert.Empty(t, decoded.RootNodes)
	})
}

func TestEncodeDecodeAST_BasicFields(t *testing.T) {
	t.Run("SourcePath is preserved", func(t *testing.T) {
		sourcePath := "/path/to/template.pk"
		original := &ast_domain.TemplateAST{
			SourcePath: &sourcePath,
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.SourcePath)
		assert.Equal(t, sourcePath, *decoded.SourcePath)
	})

	t.Run("Tidied flag is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			Tidied: true,
		}

		decoded := mustRoundTrip(t, original)
		assert.True(t, decoded.Tidied)
	})

	t.Run("ExpiresAtUnixNano is preserved", func(t *testing.T) {
		expiresAt := int64(1234567890123456789)
		original := &ast_domain.TemplateAST{
			ExpiresAtUnixNano: &expiresAt,
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.ExpiresAtUnixNano)
		assert.Equal(t, expiresAt, *decoded.ExpiresAtUnixNano)
	})

	t.Run("Metadata is preserved", func(t *testing.T) {
		metadata := "title: Test Page\nauthor: Test Author"
		original := &ast_domain.TemplateAST{
			Metadata: &metadata,
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.Metadata)
		assert.Equal(t, metadata, *decoded.Metadata)
	})

	t.Run("SourceSize is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			SourceSize: 12345,
		}

		decoded := mustRoundTrip(t, original)
		assert.Equal(t, int64(12345), decoded.SourceSize)
	})
}

func TestDecodeAST_EmptyData(t *testing.T) {
	t.Run("empty byte slice returns error", func(t *testing.T) {
		_, err := DecodeAST(context.Background(), []byte{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot decode empty byte slice")
	})

	t.Run("nil byte slice returns error", func(t *testing.T) {
		_, err := DecodeAST(context.Background(), nil)
		require.Error(t, err)
	})
}

func TestEncodeDecodeAST_NodeTypes(t *testing.T) {
	testCases := []struct {
		name     string
		tagName  string
		nodeType ast_domain.NodeType
	}{
		{name: "element node", nodeType: ast_domain.NodeElement, tagName: "div"},
		{name: "text node", nodeType: ast_domain.NodeText, tagName: ""},
		{name: "comment node", nodeType: ast_domain.NodeComment, tagName: ""},
		{name: "fragment node", nodeType: ast_domain.NodeFragment, tagName: ""},
		{name: "raw HTML node", nodeType: ast_domain.NodeRawHTML, tagName: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: tc.nodeType,
						TagName:  tc.tagName,
					},
				},
			}

			decoded := mustRoundTrip(t, original)

			require.Len(t, decoded.RootNodes, 1)
			assert.Equal(t, tc.nodeType, decoded.RootNodes[0].NodeType)
			assert.Equal(t, tc.tagName, decoded.RootNodes[0].TagName)
		})
	}
}

func TestEncodeDecodeAST_NestedChildren(t *testing.T) {
	t.Run("single level of children", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeElement, TagName: "span"},
						{NodeType: ast_domain.NodeText, TextContent: "Hello"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes, 1)
		require.Len(t, decoded.RootNodes[0].Children, 2)
		assert.Equal(t, "span", decoded.RootNodes[0].Children[0].TagName)
		assert.Equal(t, "Hello", decoded.RootNodes[0].Children[1].TextContent)
	})

	t.Run("deeply nested children", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "section",
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "article",
									Children: []*ast_domain.TemplateNode{
										{NodeType: ast_domain.NodeText, TextContent: "Deep content"},
									},
								},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes, 1)
		div := decoded.RootNodes[0]
		require.Len(t, div.Children, 1)

		section := div.Children[0]
		assert.Equal(t, "section", section.TagName)
		require.Len(t, section.Children, 1)

		article := section.Children[0]
		assert.Equal(t, "article", article.TagName)
		require.Len(t, article.Children, 1)

		text := article.Children[0]
		assert.Equal(t, "Deep content", text.TextContent)
	})
}

func TestEncodeDecodeAST_HTMLAttributes(t *testing.T) {
	t.Run("single attribute", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "id", Value: "my-div"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].Attributes, 1)
		attr := decoded.RootNodes[0].Attributes[0]
		assert.Equal(t, "id", attr.Name)
		assert.Equal(t, "my-div", attr.Value)
	})

	t.Run("multiple attributes", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "text"},
						{Name: "name", Value: "username"},
						{Name: "placeholder", Value: "Enter username"},
						{Name: "disabled", Value: ""},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].Attributes, 4)
		attrs := decoded.RootNodes[0].Attributes
		assert.Equal(t, "type", attrs[0].Name)
		assert.Equal(t, "text", attrs[0].Value)
		assert.Equal(t, "disabled", attrs[3].Name)
		assert.Equal(t, "", attrs[3].Value)
	})

	t.Run("attribute with location data", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{
							Name:  "class",
							Value: "container",
							Location: ast_domain.Location{
								Line:   5,
								Column: 10,
								Offset: 100,
							},
							NameLocation: ast_domain.Location{
								Line:   5,
								Column: 10,
								Offset: 100,
							},
							AttributeRange: ast_domain.Range{
								Start: ast_domain.Location{Line: 5, Column: 10, Offset: 100},
								End:   ast_domain.Location{Line: 5, Column: 27, Offset: 117},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		attr := decoded.RootNodes[0].Attributes[0]
		assert.Equal(t, 5, attr.Location.Line)
		assert.Equal(t, 10, attr.Location.Column)
		assert.Equal(t, 100, attr.AttributeRange.Start.Offset)
		assert.Equal(t, 117, attr.AttributeRange.End.Offset)
	})
}

func TestEncodeDecodeAST_DynamicAttributes(t *testing.T) {
	t.Run("dynamic attribute with expression", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DynamicAttributes: []ast_domain.DynamicAttribute{
						{
							Name:          "class",
							RawExpression: "isActive ? 'active' : 'inactive'",
							Expression: &ast_domain.TernaryExpression{
								Condition:  &ast_domain.Identifier{Name: "isActive"},
								Consequent: &ast_domain.StringLiteral{Value: "active"},
								Alternate:  &ast_domain.StringLiteral{Value: "inactive"},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].DynamicAttributes, 1)
		dynAttr := decoded.RootNodes[0].DynamicAttributes[0]
		assert.Equal(t, "class", dynAttr.Name)
		assert.Equal(t, "isActive ? 'active' : 'inactive'", dynAttr.RawExpression)
		require.NotNil(t, dynAttr.Expression)

		ternary, ok := dynAttr.Expression.(*ast_domain.TernaryExpression)
		require.True(t, ok, "expected TernaryExpr")
		assert.NotNil(t, ternary.Condition)
		assert.NotNil(t, ternary.Consequent)
		assert.NotNil(t, ternary.Alternate)
	})
}

func TestEncodeDecodeAST_ControlFlowDirectives(t *testing.T) {
	t.Run("p-if directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirIf: &ast_domain.Directive{
						Type:          ast_domain.DirectiveIf,
						RawExpression: "isVisible",
						Expression:    &ast_domain.Identifier{Name: "isVisible"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirIf)
		assert.Equal(t, ast_domain.DirectiveIf, decoded.RootNodes[0].DirIf.Type)
		assert.Equal(t, "isVisible", decoded.RootNodes[0].DirIf.RawExpression)
	})

	t.Run("p-for directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "li",
					DirFor: &ast_domain.Directive{
						Type:          ast_domain.DirectiveFor,
						RawExpression: "item in items",
						Arg:           "item",
						Expression:    &ast_domain.Identifier{Name: "items"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirFor)
		dirFor := decoded.RootNodes[0].DirFor
		assert.Equal(t, ast_domain.DirectiveFor, dirFor.Type)
		assert.Equal(t, "item", dirFor.Arg)
		assert.Equal(t, "item in items", dirFor.RawExpression)
	})

	t.Run("p-else-if directive with chain key", func(t *testing.T) {
		chainKey := &ast_domain.StringLiteral{Value: "chain-1"}
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirElseIf: &ast_domain.Directive{
						Type:          ast_domain.DirectiveElseIf,
						RawExpression: "count > 5",
						ChainKey:      chainKey,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirElseIf)
		require.NotNil(t, decoded.RootNodes[0].DirElseIf.ChainKey)
	})

	t.Run("p-else directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirElse: &ast_domain.Directive{
						Type: ast_domain.DirectiveElse,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirElse)
		assert.Equal(t, ast_domain.DirectiveElse, decoded.RootNodes[0].DirElse.Type)
	})
}

func TestEncodeDecodeAST_ContentDirectives(t *testing.T) {
	t.Run("p-text directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "span",
					DirText: &ast_domain.Directive{
						Type:          ast_domain.DirectiveText,
						RawExpression: "message",
						Expression:    &ast_domain.Identifier{Name: "message"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirText)
		assert.Equal(t, ast_domain.DirectiveText, decoded.RootNodes[0].DirText.Type)
	})

	t.Run("p-html directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirHTML: &ast_domain.Directive{
						Type:          ast_domain.DirectiveHTML,
						RawExpression: "rawContent",
						Expression:    &ast_domain.Identifier{Name: "rawContent"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirHTML)
		assert.Equal(t, ast_domain.DirectiveHTML, decoded.RootNodes[0].DirHTML.Type)
	})

	t.Run("p-class directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirClass: &ast_domain.Directive{
						Type:          ast_domain.DirectiveClass,
						RawExpression: "{ active: isActive }",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirClass)
		assert.Equal(t, ast_domain.DirectiveClass, decoded.RootNodes[0].DirClass.Type)
	})

	t.Run("p-style directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirStyle: &ast_domain.Directive{
						Type:          ast_domain.DirectiveStyle,
						RawExpression: "{ color: textColor }",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].DirStyle)
		assert.Equal(t, ast_domain.DirectiveStyle, decoded.RootNodes[0].DirStyle.Type)
	})
}

func TestEncodeDecodeAST_OtherDirectives(t *testing.T) {
	t.Run("p-show directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirShow: &ast_domain.Directive{
						Type:          ast_domain.DirectiveShow,
						RawExpression: "isVisible",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		require.NotNil(t, decoded.RootNodes[0].DirShow)
	})

	t.Run("p-model directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					DirModel: &ast_domain.Directive{
						Type:          ast_domain.DirectiveModel,
						RawExpression: "username",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		require.NotNil(t, decoded.RootNodes[0].DirModel)
	})

	t.Run("p-ref directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirRef: &ast_domain.Directive{
						Type: ast_domain.DirectiveRef,
						Arg:  "myDiv",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		require.NotNil(t, decoded.RootNodes[0].DirRef)
		assert.Equal(t, "myDiv", decoded.RootNodes[0].DirRef.Arg)
	})

	t.Run("p-slot directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "template",
					DirSlot: &ast_domain.Directive{
						Type: ast_domain.DirectiveSlot,
						Arg:  "header",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		require.NotNil(t, decoded.RootNodes[0].DirSlot)
		assert.Equal(t, "header", decoded.RootNodes[0].DirSlot.Arg)
	})

	t.Run("p-key directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "li",
					DirKey: &ast_domain.Directive{
						Type:          ast_domain.DirectiveKey,
						RawExpression: "item.id",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		require.NotNil(t, decoded.RootNodes[0].DirKey)
	})

	t.Run("p-context directive", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					DirContext: &ast_domain.Directive{
						Type:          ast_domain.DirectiveContext,
						RawExpression: "'my-context'",
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		require.NotNil(t, decoded.RootNodes[0].DirContext)
	})

}

func TestEncodeDecodeAST_TextContent(t *testing.T) {
	t.Run("simple text content", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "Hello, World!",
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.Equal(t, "Hello, World!", decoded.RootNodes[0].TextContent)
	})

	t.Run("text with special characters", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "<script>alert('XSS')</script> & \"quotes\" 'apostrophes'",
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.Equal(t, "<script>alert('XSS')</script> & \"quotes\" 'apostrophes'", decoded.RootNodes[0].TextContent)
	})

	t.Run("innerHTML is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:  ast_domain.NodeElement,
					TagName:   "div",
					InnerHTML: "<strong>Bold</strong> text",
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.Equal(t, "<strong>Bold</strong> text", decoded.RootNodes[0].InnerHTML)
	})
}

func TestEncodeDecodeAST_RichText(t *testing.T) {
	t.Run("rich text with literal part", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "p",
					RichText: []ast_domain.TextPart{
						{
							IsLiteral: true,
							Literal:   "Hello, ",
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].RichText, 1)
		assert.True(t, decoded.RootNodes[0].RichText[0].IsLiteral)
		assert.Equal(t, "Hello, ", decoded.RootNodes[0].RichText[0].Literal)
	})

	t.Run("rich text with expression part", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "p",
					RichText: []ast_domain.TextPart{
						{
							IsLiteral:     false,
							RawExpression: "name",
							Expression:    &ast_domain.Identifier{Name: "name"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].RichText, 1)
		part := decoded.RootNodes[0].RichText[0]
		assert.False(t, part.IsLiteral)
		assert.Equal(t, "name", part.RawExpression)
		require.NotNil(t, part.Expression)
	})

	t.Run("mixed rich text parts", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "p",
					RichText: []ast_domain.TextPart{
						{IsLiteral: true, Literal: "Hello, "},
						{IsLiteral: false, RawExpression: "name", Expression: &ast_domain.Identifier{Name: "name"}},
						{IsLiteral: true, Literal: "! You have "},
						{IsLiteral: false, RawExpression: "count", Expression: &ast_domain.Identifier{Name: "count"}},
						{IsLiteral: true, Literal: " messages."},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].RichText, 5)
		assert.True(t, decoded.RootNodes[0].RichText[0].IsLiteral)
		assert.False(t, decoded.RootNodes[0].RichText[1].IsLiteral)
		assert.True(t, decoded.RootNodes[0].RichText[2].IsLiteral)
		assert.False(t, decoded.RootNodes[0].RichText[3].IsLiteral)
		assert.True(t, decoded.RootNodes[0].RichText[4].IsLiteral)
	})
}

func TestEncodeDecodeAST_EventMaps(t *testing.T) {
	t.Run("OnEvents map", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "button",
					OnEvents: map[string][]ast_domain.Directive{
						"click": {
							{Type: ast_domain.DirectiveOn, Arg: "click", RawExpression: "handleClick"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].OnEvents)
		require.Contains(t, decoded.RootNodes[0].OnEvents, "click")
		require.Len(t, decoded.RootNodes[0].OnEvents["click"], 1)
		assert.Equal(t, "handleClick", decoded.RootNodes[0].OnEvents["click"][0].RawExpression)
	})

	t.Run("CustomEvents map", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "my-component",
					CustomEvents: map[string][]ast_domain.Directive{
						"update": {
							{Type: ast_domain.DirectiveEvent, Arg: "update", RawExpression: "onUpdate"},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].CustomEvents)
		require.Contains(t, decoded.RootNodes[0].CustomEvents, "update")
		require.Len(t, decoded.RootNodes[0].CustomEvents["update"], 1)
	})

	t.Run("Binds map", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Binds: map[string]*ast_domain.Directive{
						"title": {
							Type:          ast_domain.DirectiveBind,
							Arg:           "title",
							RawExpression: "tooltipText",
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.RootNodes[0].Binds)
		require.Contains(t, decoded.RootNodes[0].Binds, "title")
		assert.Equal(t, "tooltipText", decoded.RootNodes[0].Binds["title"].RawExpression)
	})
}

func TestEncodeDecodeAST_LocationData(t *testing.T) {
	t.Run("node location is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Location: ast_domain.Location{
						Line:   10,
						Column: 5,
						Offset: 150,
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		loc := decoded.RootNodes[0].Location
		assert.Equal(t, 10, loc.Line)
		assert.Equal(t, 5, loc.Column)
		assert.Equal(t, 150, loc.Offset)
	})

	t.Run("node range is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					NodeRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 10, Column: 1, Offset: 100},
						End:   ast_domain.Location{Line: 15, Column: 6, Offset: 200},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		nodeRange := decoded.RootNodes[0].NodeRange
		assert.Equal(t, 10, nodeRange.Start.Line)
		assert.Equal(t, 1, nodeRange.Start.Column)
		assert.Equal(t, 100, nodeRange.Start.Offset)
		assert.Equal(t, 15, nodeRange.End.Line)
		assert.Equal(t, 6, nodeRange.End.Column)
		assert.Equal(t, 200, nodeRange.End.Offset)
	})

	t.Run("opening and closing tag ranges", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					OpeningTagRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
						End:   ast_domain.Location{Line: 1, Column: 5, Offset: 4},
					},
					ClosingTagRange: ast_domain.Range{
						Start: ast_domain.Location{Line: 3, Column: 1, Offset: 50},
						End:   ast_domain.Location{Line: 3, Column: 6, Offset: 55},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		assert.Equal(t, 1, decoded.RootNodes[0].OpeningTagRange.Start.Line)
		assert.Equal(t, 3, decoded.RootNodes[0].ClosingTagRange.Start.Line)
	})
}

func TestEncodeDecodeAST_Diagnostics(t *testing.T) {
	t.Run("AST-level diagnostics", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			Diagnostics: []*ast_domain.Diagnostic{
				{
					Message:  "Missing closing tag",
					Severity: ast_domain.Error,
					Location: ast_domain.Location{Line: 10, Column: 1},
				},
				{
					Message:  "Deprecated directive",
					Severity: ast_domain.Warning,
					Location: ast_domain.Location{Line: 5, Column: 3},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.Diagnostics, 2)
		assert.Equal(t, "Missing closing tag", decoded.Diagnostics[0].Message)
		assert.Equal(t, ast_domain.Error, decoded.Diagnostics[0].Severity)
		assert.Equal(t, "Deprecated directive", decoded.Diagnostics[1].Message)
		assert.Equal(t, ast_domain.Warning, decoded.Diagnostics[1].Severity)
	})

	t.Run("node-level diagnostics", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Diagnostics: []*ast_domain.Diagnostic{
						{
							Message:  "Invalid attribute",
							Severity: ast_domain.Error,
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].Diagnostics, 1)
		assert.Equal(t, "Invalid attribute", decoded.RootNodes[0].Diagnostics[0].Message)
	})
}

func TestEncodeDecodeAST_OtherProperties(t *testing.T) {
	t.Run("IsContentEditable is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:          ast_domain.NodeElement,
					TagName:           "div",
					IsContentEditable: true,
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.True(t, decoded.RootNodes[0].IsContentEditable)
	})

	t.Run("PreferredFormat is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:        ast_domain.NodeElement,
					TagName:         "div",
					PreferredFormat: ast_domain.FormatInline,
				},
			},
		}

		decoded := mustRoundTrip(t, original)
		assert.Equal(t, ast_domain.FormatInline, decoded.RootNodes[0].PreferredFormat)
	})

	t.Run("Directives slice is preserved", func(t *testing.T) {
		original := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Directives: []ast_domain.Directive{
						{Type: ast_domain.DirectiveIf, RawExpression: "cond1"},
						{Type: ast_domain.DirectiveShow, RawExpression: "cond2"},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.Len(t, decoded.RootNodes[0].Directives, 2)
		assert.Equal(t, ast_domain.DirectiveIf, decoded.RootNodes[0].Directives[0].Type)
		assert.Equal(t, ast_domain.DirectiveShow, decoded.RootNodes[0].Directives[1].Type)
	})
}

func TestEncodeDecodeAST_ComplexTemplate(t *testing.T) {
	t.Run("complex nested template with multiple features", func(t *testing.T) {
		sourcePath := "/path/to/complex.pk"
		metadata := "title: Complex Page"

		original := &ast_domain.TemplateAST{
			SourcePath: &sourcePath,
			Metadata:   &metadata,
			Tidied:     true,
			SourceSize: 5000,
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "container"},
					},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "h1",
							DirText: &ast_domain.Directive{
								Type:          ast_domain.DirectiveText,
								RawExpression: "title",
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "ul",
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "li",
									DirFor: &ast_domain.Directive{
										Type:          ast_domain.DirectiveFor,
										Arg:           "item",
										RawExpression: "item in items",
									},
									DirKey: &ast_domain.Directive{
										Type:          ast_domain.DirectiveKey,
										RawExpression: "item.id",
									},
									RichText: []ast_domain.TextPart{
										{IsLiteral: false, RawExpression: "item.name"},
									},
								},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "button",
							OnEvents: map[string][]ast_domain.Directive{
								"click": {{Type: ast_domain.DirectiveOn, RawExpression: "handleClick"}},
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "Click me"},
							},
						},
					},
				},
			},
		}

		decoded := mustRoundTrip(t, original)

		require.NotNil(t, decoded.SourcePath)
		assert.Equal(t, sourcePath, *decoded.SourcePath)
		require.NotNil(t, decoded.Metadata)
		assert.Equal(t, metadata, *decoded.Metadata)
		assert.True(t, decoded.Tidied)
		assert.Equal(t, int64(5000), decoded.SourceSize)

		require.Len(t, decoded.RootNodes, 1)
		container := decoded.RootNodes[0]
		assert.Equal(t, "div", container.TagName)
		require.Len(t, container.Attributes, 1)
		assert.Equal(t, "container", container.Attributes[0].Value)

		require.Len(t, container.Children, 3)

		h1 := container.Children[0]
		assert.Equal(t, "h1", h1.TagName)
		require.NotNil(t, h1.DirText)

		ul := container.Children[1]
		assert.Equal(t, "ul", ul.TagName)
		require.Len(t, ul.Children, 1)
		li := ul.Children[0]
		assert.Equal(t, "li", li.TagName)
		require.NotNil(t, li.DirFor)
		require.NotNil(t, li.DirKey)

		button := container.Children[2]
		assert.Equal(t, "button", button.TagName)
		require.NotNil(t, button.OnEvents)
		require.Contains(t, button.OnEvents, "click")
	})
}

func mustRoundTrip(t *testing.T, original *ast_domain.TemplateAST) *ast_domain.TemplateAST {
	t.Helper()

	encoded, err := EncodeAST(original)
	require.NoError(t, err, "EncodeAST failed")
	require.NotEmpty(t, encoded, "EncodeAST returned empty bytes")

	decoded, err := DecodeAST(context.Background(), encoded)
	require.NoError(t, err, "DecodeAST failed")
	require.NotNil(t, decoded, "DecodeAST returned nil")

	return decoded
}
