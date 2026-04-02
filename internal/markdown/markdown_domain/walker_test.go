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

package markdown_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

func Test_newMarkdownWalker(t *testing.T) {
	t.Run("CreatesWalkerWithDependencies", func(t *testing.T) {
		source := []byte("# Test")
		transformer := &mockNodeTransformer{}
		diagnostics := make([]*ast_domain.Diagnostic, 0)

		walker := newMarkdownWalker(transformer, source, diagnostics)

		assert.NotNil(t, walker)
		assert.Equal(t, source, walker.source)
		assert.Equal(t, transformer, walker.transformer)
		assert.Equal(t, diagnostics, walker.diagnostics)
		assert.NotNil(t, walker.blocks)
		assert.Equal(t, 0, walker.wordCount)
	})

	t.Run("AcceptsInterfaceForTransformer", func(t *testing.T) {
		source := []byte("# Test")
		mockTransformer := &mockNodeTransformer{}
		diagnostics := make([]*ast_domain.Diagnostic, 0)

		walker := newMarkdownWalker(mockTransformer, source, diagnostics)

		assert.NotNil(t, walker)
		assert.Equal(t, mockTransformer, walker.transformer)
	})
}

func Test_markdownWalker_Transform(t *testing.T) {
	t.Run("EmptyDocument", func(t *testing.T) {
		source := []byte("")
		document := markdown_ast.NewDocument()
		walker := NewWalkerTestBuilder().
			WithSource(source).
			Build()

		result, err := walker.Transform(context.Background(), document)

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.PageAST)
		assert.Nil(t, result.ExcerptAST, "Empty document should not have excerpt")
		assert.NotNil(t, result.Metadata)
		assert.Empty(t, result.Metadata.Sections)
		assert.Empty(t, result.Metadata.Images)
		assert.Empty(t, result.Metadata.Links)
		assert.Equal(t, 0, result.Metadata.WordCount)
	})
}

func Test_markdownWalker_BuildExcerptNodes(t *testing.T) {
	t.Run("NoExcerptWhenNoParagraph", func(t *testing.T) {
		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "h1",
				},
			},
		}

		result := walker.buildExcerptNodes()

		assert.Nil(t, result, "Should not have excerpt when there's no paragraph")
	})

	t.Run("ReturnsFirstParagraph", func(t *testing.T) {
		paragraph := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
		}

		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{
				paragraph,
			},
		}

		result := walker.buildExcerptNodes()

		require.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, "p", result[0].TagName)
	})

	t.Run("ExcerptSeparatorSplitsContent", func(t *testing.T) {
		para1 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
			Children: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "First paragraph",
				},
			},
		}

		separator := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "<!--more-->",
		}

		para2 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
			Children: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "Second paragraph",
				},
			},
		}

		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{
				para1,
				separator,
				para2,
			},
		}

		result := walker.buildExcerptNodes()

		require.NotNil(t, result)
		assert.Len(t, result, 1, "Excerpt should only contain content before separator")
		assert.Equal(t, "p", result[0].TagName)
		assert.Equal(t, "First paragraph", result[0].Children[0].TextContent)
	})

	t.Run("ExcerptSeparatorWithMultipleNodesBeforeSeparator", func(t *testing.T) {
		heading := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "h1",
		}

		para1 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
		}

		para2 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
		}

		separator := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "<!--more-->",
		}

		para3 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
		}

		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{
				heading,
				para1,
				para2,
				separator,
				para3,
			},
		}

		result := walker.buildExcerptNodes()

		require.NotNil(t, result)
		assert.Len(t, result, 3, "Excerpt should contain all nodes before separator")
		assert.Equal(t, "h1", result[0].TagName)
		assert.Equal(t, "p", result[1].TagName)
		assert.Equal(t, "p", result[2].TagName)
	})
}

func Test_markdownWalker_CollectMetadata(t *testing.T) {
	t.Run("CollectsLinksFromNode", func(t *testing.T) {
		linkChild := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "link text",
		}

		linkNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "a",
			Attributes: []ast_domain.HTMLAttribute{
				{
					Name:  "href",
					Value: "https://example.com",
				},
			},
			Children: []*ast_domain.TemplateNode{linkChild},
		}

		walker := &markdownWalker{
			links: []markdown_dto.LinkMeta{},
		}

		walker.collectMetadata(linkNode)

		assert.Len(t, walker.links, 1)
		assert.Equal(t, "https://example.com", walker.links[0].Href)
		assert.Equal(t, "link text", walker.links[0].Text)
	})

	t.Run("CollectsImagesFromNode", func(t *testing.T) {
		imgNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "img",
			Attributes: []ast_domain.HTMLAttribute{
				{
					Name:  "src",
					Value: "/image.png",
				},
				{
					Name:  "alt",
					Value: "alt text",
				},
			},
		}

		walker := &markdownWalker{
			images: []markdown_dto.ImageMeta{},
		}

		walker.collectMetadata(imgNode)

		assert.Len(t, walker.images, 1)
		assert.Equal(t, "/image.png", walker.images[0].Src)
		assert.Equal(t, "alt text", walker.images[0].Alt)
	})
}

func Test_markdownWalker_BuildSectionsData(t *testing.T) {
	t.Run("ExtractsSectionsFromHeadings", func(t *testing.T) {
		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "h2",
					Attributes: []ast_domain.HTMLAttribute{
						{
							Name:  "id",
							Value: "heading-text",
						},
						{
							Name:  "title",
							Value: "Heading Text",
						},
					},
				},
			},
		}

		sections := walker.buildSectionsData()

		assert.Len(t, sections, 1)
		assert.Equal(t, "Heading Text", sections[0].Title)
		assert.Equal(t, "heading-text", sections[0].Slug)
		assert.Equal(t, 2, sections[0].Level)
	})

	t.Run("HandlesMultipleHeadingLevels", func(t *testing.T) {
		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "h1",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "id", Value: "main"},
						{Name: "title", Value: "Main Title"},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "h3",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "id", Value: "sub"},
						{Name: "title", Value: "Subsection"},
					},
				},
			},
		}

		sections := walker.buildSectionsData()

		assert.Len(t, sections, 2)
		assert.Equal(t, 1, sections[0].Level)
		assert.Equal(t, 3, sections[1].Level)
	})
}

func Test_markdownWalker_AppendNode(t *testing.T) {
	t.Run("AppendsRegularNodeToContent", func(t *testing.T) {
		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{},
		}

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
		}

		walker.appendNode(node)

		assert.Len(t, walker.pikoContent, 1)
		assert.Equal(t, "p", walker.pikoContent[0].TagName)
	})

	t.Run("FlattensFragmentChildren", func(t *testing.T) {
		walker := &markdownWalker{
			pikoContent: []*ast_domain.TemplateNode{},
		}

		child1 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
		}

		child2 := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "h1",
		}

		fragment := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeFragment,
			Children: []*ast_domain.TemplateNode{child1, child2},
		}

		walker.appendNode(fragment)

		assert.Len(t, walker.pikoContent, 2, "Fragment children should be flattened")
		assert.Equal(t, "p", walker.pikoContent[0].TagName)
		assert.Equal(t, "h1", walker.pikoContent[1].TagName)
	})

	t.Run("AppendsToNamedBlock", func(t *testing.T) {
		walker := &markdownWalker{
			pikoContent:      []*ast_domain.TemplateNode{},
			blocks:           make(map[string][]*ast_domain.TemplateNode),
			currentBlockName: "sidebar",
		}

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		walker.appendNode(node)

		assert.Empty(t, walker.pikoContent, "Should not append to main content when in named block")
		assert.Len(t, walker.blocks["sidebar"], 1, "Should append to named block")
		assert.Equal(t, "div", walker.blocks["sidebar"][0].TagName)
	})
}

func Test_countWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected int
	}{
		{
			name:     "NilNode",
			node:     nil,
			expected: 0,
		},
		{
			name: "SingleTextNode",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "hello world foo bar",
			},
			expected: 4,
		},
		{
			name: "EmptyTextContent",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "",
			},
			expected: 0,
		},
		{
			name: "NonTextNodeWithoutChildren",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
			},
			expected: 0,
		},
		{
			name: "NestedChildren",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "p",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "hello world",
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "strong",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "bold text",
							},
						},
					},
				},
			},
			expected: 4,
		},
		{
			name: "DeeplyNested",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "em",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType:    ast_domain.NodeText,
										TextContent: "deeply nested words here",
									},
								},
							},
						},
					},
				},
			},
			expected: 4,
		},
		{
			name: "MixedNodeTypes",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeFragment,
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "first",
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "br",
					},
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "second third",
					},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, countWords(tt.node))
		})
	}
}

func Test_markdownWalker_HandleNodeEnter_WordCount(t *testing.T) {
	t.Parallel()

	t.Run("CountsWordsFromTransformedNodes", func(t *testing.T) {
		t.Parallel()

		textNode := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "hello world foo",
		}

		mockTransformer := &mockNodeTransformer{
			TransformNodeFunc: func(_ context.Context, node markdown_ast.Node) *ast_domain.TemplateNode {
				return textNode
			},
		}

		walker := newMarkdownWalker(mockTransformer, []byte("hello world foo"), nil)

		paraNode := markdown_ast.NewParagraph()

		status := walker.handleNodeEnter(paraNode)

		assert.Equal(t, markdown_ast.WalkSkipChildren, status)
		assert.Equal(t, 3, walker.wordCount)
	})

	t.Run("AccumulatesWordCount", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		textNodes := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "hello world"},
			{NodeType: ast_domain.NodeText, TextContent: "foo bar baz"},
		}

		mockTransformer := &mockNodeTransformer{
			TransformNodeFunc: func(_ context.Context, node markdown_ast.Node) *ast_domain.TemplateNode {
				index := callCount
				callCount++
				if index < len(textNodes) {
					return textNodes[index]
				}
				return nil
			},
		}

		walker := newMarkdownWalker(mockTransformer, []byte("test"), nil)

		_ = walker.handleNodeEnter(markdown_ast.NewParagraph())
		_ = walker.handleNodeEnter(markdown_ast.NewParagraph())

		assert.Equal(t, 5, walker.wordCount)
	})

	t.Run("ZeroWordsWhenTransformReturnsNil", func(t *testing.T) {
		t.Parallel()

		mockTransformer := &mockNodeTransformer{
			TransformNodeFunc: func(_ context.Context, node markdown_ast.Node) *ast_domain.TemplateNode {
				return nil
			},
		}

		walker := newMarkdownWalker(mockTransformer, []byte("test"), nil)

		status := walker.handleNodeEnter(markdown_ast.NewParagraph())

		assert.Equal(t, markdown_ast.WalkContinue, status)
		assert.Equal(t, 0, walker.wordCount)
	})

	t.Run("CountsWordsInNestedPikoNodes", func(t *testing.T) {
		t.Parallel()

		paragraphNode := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "p",
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "hello "},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "strong",
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "bold world"},
					},
				},
			},
		}

		mockTransformer := &mockNodeTransformer{
			TransformNodeFunc: func(_ context.Context, node markdown_ast.Node) *ast_domain.TemplateNode {
				return paragraphNode
			},
		}

		walker := newMarkdownWalker(mockTransformer, []byte("test"), nil)

		_ = walker.handleNodeEnter(markdown_ast.NewParagraph())

		assert.Equal(t, 3, walker.wordCount)
	})
}

func Test_markdownWalker_DiagnosticsSharing(t *testing.T) {
	t.Run("SharesDiagnosticsWithTransformer", func(t *testing.T) {
		source := []byte("Test")
		diagnostics := make([]*ast_domain.Diagnostic, 0)

		walker := newMarkdownWalker(&mockNodeTransformer{}, source, diagnostics)

		assert.Equal(t, diagnostics, walker.diagnostics, "Walker should reference the same diagnostics slice")
	})
}
