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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
)

func TestNewTransformer(t *testing.T) {
	t.Run("CreatesTransformerWithAllDependencies", func(t *testing.T) {
		source := []byte("# Test")
		mapper := newLocationMapper(source)
		diagnostics := make([]*ast_domain.Diagnostic, 0)

		tr := newTransformer("test.md", source, mapper, &diagnostics, nil)

		assert.NotNil(t, tr)
		assert.Equal(t, "test.md", tr.sourcePath)
		assert.Equal(t, source, tr.source)
		assert.Equal(t, mapper, tr.locationMapper)
		assert.Equal(t, &diagnostics, tr.diagnostics)
	})

	t.Run("AcceptsInterfaceForLocationMapper", func(t *testing.T) {
		source := []byte("# Test")
		mockMapper := &mockPositionMapper{}
		tr := newTransformer("test.md", source, mockMapper, new([]*ast_domain.Diagnostic), nil)

		assert.NotNil(t, tr)
		assert.Equal(t, mockMapper, tr.locationMapper)
	})
}

func TestTransformer_TransformNode_Paragraph(t *testing.T) {
	t.Run("SimpleParagraph", func(t *testing.T) {
		source := []byte("Hello world")

		para := markdown_ast.NewParagraph()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		para.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), para)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "p", result.TagName)
		assert.NotEmpty(t, result.Children, "Paragraph should have children")
	})

	t.Run("EmptyParagraph", func(t *testing.T) {
		source := []byte("")
		para := markdown_ast.NewParagraph()

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), para)

		require.NotNil(t, result)
		assert.Equal(t, "p", result.TagName)
	})
}

func TestTransformer_TransformNode_Heading(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		headingID     string
		expectedTag   string
		expectedTitle string
		level         int
	}{
		{
			name:          "H1",
			level:         1,
			text:          "Hello World",
			headingID:     "hello-world",
			expectedTag:   "h1",
			expectedTitle: "Hello World",
		},
		{
			name:          "H2",
			level:         2,
			text:          "Introduction",
			headingID:     "introduction",
			expectedTag:   "h2",
			expectedTitle: "Introduction",
		},
		{
			name:          "H3 with ID",
			level:         3,
			text:          "Hello, World! 123",
			headingID:     "custom-id",
			expectedTag:   "h3",
			expectedTitle: "Hello, World! 123",
		},
		{
			name:          "H6",
			level:         6,
			text:          "Deep Heading",
			headingID:     "deep",
			expectedTag:   "h6",
			expectedTitle: "Deep Heading",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := []byte(tt.text)

			heading := markdown_ast.NewHeading(tt.level)
			textNode := markdown_ast.NewText(source)
			textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
			heading.AppendChild(textNode)
			heading.SetAttributeString("id", tt.headingID)

			tr := NewTransformerTestBuilder().
				WithSource(source).
				Build()

			result := tr.TransformNode(context.Background(), heading)

			require.NotNil(t, result)
			assert.Equal(t, ast_domain.NodeElement, result.NodeType)
			assert.Equal(t, tt.expectedTag, result.TagName)

			slug, hasSlug := result.GetAttribute("id")
			assert.True(t, hasSlug, "Heading should have id attribute")
			assert.Equal(t, tt.headingID, slug)

			title, hasTitle := result.GetAttribute("title")
			assert.True(t, hasTitle, "Heading should have title attribute")
			assert.Equal(t, tt.expectedTitle, title)
		})
	}
}

func TestTransformer_TransformNode_Text(t *testing.T) {
	t.Run("SimpleText", func(t *testing.T) {
		source := []byte("Hello, World!")
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), textNode)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeText, result.NodeType)
		assert.Equal(t, "Hello, World!", result.TextContent)
	})

	t.Run("TextWithUnicode", func(t *testing.T) {
		source := []byte("Hello 世界")
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), textNode)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeText, result.NodeType)
		assert.Equal(t, "Hello 世界", result.TextContent)
	})
}

func TestTransformer_TransformNode_EmphasisAndStrong(t *testing.T) {
	t.Run("Emphasis", func(t *testing.T) {
		source := []byte("italic")

		emph := markdown_ast.NewEmphasis(1)
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		emph.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), emph)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "em", result.TagName)
		assert.NotEmpty(t, result.Children)
	})

	t.Run("Strong", func(t *testing.T) {
		source := []byte("bold")

		strong := markdown_ast.NewEmphasis(2)
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		strong.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), strong)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "strong", result.TagName)
		assert.NotEmpty(t, result.Children)
	})
}

func TestTransformer_TransformNode_Link(t *testing.T) {
	t.Run("LinkWithHref", func(t *testing.T) {
		source := []byte("Click here")

		link := markdown_ast.NewLink([]byte("https://example.com"), nil)
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		link.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), link)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "a", result.TagName)

		href, hasHref := result.GetAttribute("href")
		assert.True(t, hasHref)
		assert.Equal(t, "https://example.com", href)
	})

	t.Run("LinkWithTitle", func(t *testing.T) {
		source := []byte("Click here")

		link := markdown_ast.NewLink([]byte("https://example.com"), []byte("Example Site"))
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		link.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), link)

		require.NotNil(t, result)
		title, hasTitle := result.GetAttribute("title")
		assert.True(t, hasTitle)
		assert.Equal(t, "Example Site", title)
	})
}

func TestTransformer_TransformNode_Image(t *testing.T) {
	t.Run("ImageWithSrcAndAlt", func(t *testing.T) {
		source := []byte("Alt text")

		img := markdown_ast.NewImage([]byte("/path/to/image.png"), nil)
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		img.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), img)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "img", result.TagName)

		src, hasSrc := result.GetAttribute("src")
		assert.True(t, hasSrc)
		assert.Equal(t, "/path/to/image.png", src)

		alt, hasAlt := result.GetAttribute("alt")
		assert.True(t, hasAlt)
		assert.Equal(t, "Alt text", alt)
	})

	t.Run("ImageWithTitle", func(t *testing.T) {
		source := []byte("Alt text")

		img := markdown_ast.NewImage([]byte("/image.png"), []byte("Hover title"))
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		img.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), img)

		require.NotNil(t, result)
		title, hasTitle := result.GetAttribute("title")
		assert.True(t, hasTitle)
		assert.Equal(t, "Hover title", title)
	})
}

func TestTransformer_TransformNode_List(t *testing.T) {
	t.Run("UnorderedList", func(t *testing.T) {
		source := []byte("Item 1")

		list := markdown_ast.NewList(false)
		listItem := markdown_ast.NewListItem()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		listItem.AppendChild(textNode)
		list.AppendChild(listItem)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), list)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "ul", result.TagName)
		assert.NotEmpty(t, result.Children)
	})

	t.Run("OrderedList", func(t *testing.T) {
		source := []byte("Item 1")

		list := markdown_ast.NewList(true)
		listItem := markdown_ast.NewListItem()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		listItem.AppendChild(textNode)
		list.AppendChild(listItem)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), list)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "ol", result.TagName)
	})

	t.Run("ListItem", func(t *testing.T) {
		source := []byte("Item content")

		listItem := markdown_ast.NewListItem()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		listItem.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), listItem)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "li", result.TagName)
	})
}

func TestTransformer_TransformNode_CodeBlock(t *testing.T) {
	t.Run("FencedCodeBlock", func(t *testing.T) {
		codeBlock := markdown_ast.NewFencedCodeBlock()
		codeBlock.Language = "js"
		codeBlock.Info = "js"
		codeBlock.Content = [][]byte{[]byte("console.log('hello');")}

		tr := NewTransformerTestBuilder().
			WithSource([]byte("js\nconsole.log('hello');")).
			Build()

		result := tr.TransformNode(context.Background(), codeBlock)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "pre", result.TagName)
		assert.NotEmpty(t, result.Children, "pre should contain code element")
	})

	t.Run("FencedCodeBlockWithHTMLContent_ShouldEscape", func(t *testing.T) {
		codeBlock := markdown_ast.NewFencedCodeBlock()
		codeBlock.Language = "html"
		codeBlock.Info = "html"
		codeBlock.Content = [][]byte{[]byte("<script>alert('xss')</script>")}

		tr := NewTransformerTestBuilder().
			WithSource([]byte("html\n<script>alert('xss')</script>")).
			Build()

		result := tr.TransformNode(context.Background(), codeBlock)

		require.NotNil(t, result)
		assert.Equal(t, "pre", result.TagName)
		require.Len(t, result.Children, 1, "pre should contain code element")

		codeElement := result.Children[0]
		assert.Equal(t, "code", codeElement.TagName)
		require.Len(t, codeElement.Children, 1, "code element should have text child")

		textNode := codeElement.Children[0]
		assert.Equal(t, ast_domain.NodeText, textNode.NodeType)
		assert.NotNil(t, textNode.TextContentWriter, "Code block text should use DirectWriter for escaping")
	})

	t.Run("FencedCodeBlockWithNoLanguage", func(t *testing.T) {
		codeBlock := markdown_ast.NewFencedCodeBlock()
		codeBlock.Content = [][]byte{[]byte("plain text code")}

		tr := NewTransformerTestBuilder().
			WithSource([]byte("plain text code")).
			Build()

		result := tr.TransformNode(context.Background(), codeBlock)

		require.NotNil(t, result)
		assert.Equal(t, "pre", result.TagName)
		require.Len(t, result.Children, 1)

		codeElement := result.Children[0]
		assert.Equal(t, "code", codeElement.TagName)
		_, hasClass := codeElement.GetAttribute("class")
		assert.False(t, hasClass, "Code element without language should have no class")
	})
}

func TestTransformer_TransformNode_Blockquote(t *testing.T) {
	t.Run("SimpleBlockquote", func(t *testing.T) {
		source := []byte("Quote text")

		blockquote := markdown_ast.NewBlockquote()
		para := markdown_ast.NewParagraph()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		para.AppendChild(textNode)
		blockquote.AppendChild(para)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), blockquote)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "blockquote", result.TagName)
		assert.NotEmpty(t, result.Children)
	})
}

func TestTransformer_TransformNode_StructuralNodes(t *testing.T) {
	t.Run("DocumentNode", func(t *testing.T) {
		document := markdown_ast.NewDocument()

		tr := NewTransformerTestBuilder().Build()

		result := tr.TransformNode(context.Background(), document)

		require.NotNil(t, result, "Document node should return a fragment")
		assert.Equal(t, ast_domain.NodeFragment, result.NodeType, "Document should be transformed to Fragment")
	})

	t.Run("TextBlock", func(t *testing.T) {
		textBlock := markdown_ast.NewTextBlock()

		tr := NewTransformerTestBuilder().Build()

		result := tr.TransformNode(context.Background(), textBlock)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeFragment, result.NodeType)
	})
}

func TestTransformer_TransformNode_HTMLBlock(t *testing.T) {
	t.Run("HTMLBlockWithExcerptSeparator", func(t *testing.T) {
		htmlBlock := markdown_ast.NewHTMLBlock()
		htmlBlock.Content = [][]byte{[]byte("<!--more-->")}
		htmlBlock.SetLines(markdown_ast.NewSegments(markdown_ast.Segment{Start: 0, Stop: 11}))

		tr := NewTransformerTestBuilder().
			WithSource([]byte("<!--more-->")).
			Build()

		result := tr.TransformNode(context.Background(), htmlBlock)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)
		assert.Equal(t, "<!--more-->", result.TextContent, "Should preserve excerpt separator")
	})

	t.Run("HTMLBlockWithRegularHTML", func(t *testing.T) {
		htmlBlock := markdown_ast.NewHTMLBlock()
		htmlBlock.Content = [][]byte{[]byte("<div>Hello</div>")}
		htmlBlock.SetLines(markdown_ast.NewSegments(markdown_ast.Segment{Start: 0, Stop: 16}))

		tr := NewTransformerTestBuilder().
			WithSource([]byte("<div>Hello</div>")).
			Build()

		result := tr.TransformNode(context.Background(), htmlBlock)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)
		assert.Equal(t, "<div>Hello</div>", result.TextContent)
	})
}

func TestTransformer_TransformNode_RawHTML(t *testing.T) {
	t.Run("RawHTMLInline", func(t *testing.T) {
		rawHTML := markdown_ast.NewRawHTML()
		rawHTML.Content = [][]byte{[]byte("<span>inline</span>")}
		rawHTML.SourceSegments = markdown_ast.NewSegments(markdown_ast.Segment{Start: 0, Stop: 19})

		tr := NewTransformerTestBuilder().
			WithSource([]byte("<span>inline</span>")).
			Build()

		result := tr.TransformNode(context.Background(), rawHTML)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeRawHTML, result.NodeType)
		assert.Equal(t, "<span>inline</span>", result.TextContent)
	})
}

func TestTransformer_TransformNode_CodeSpan(t *testing.T) {
	t.Run("InlineCodeSpan", func(t *testing.T) {
		source := []byte("inline code")

		codeSpan := markdown_ast.NewCodeSpan()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		codeSpan.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), codeSpan)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "code", result.TagName)
		assert.NotEmpty(t, result.Children, "Code span should have text children")
	})

	t.Run("CodeSpanWithHTMLTags_ShouldEscape", func(t *testing.T) {
		source := []byte("<script>alert('xss')</script>")

		codeSpan := markdown_ast.NewCodeSpan()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		codeSpan.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), codeSpan)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "code", result.TagName)
		require.Len(t, result.Children, 1, "Code span should have exactly one text child")

		child := result.Children[0]
		assert.Equal(t, ast_domain.NodeText, child.NodeType)
		assert.NotNil(t, child.TextContentWriter, "Code span text should use DirectWriter for escaping")
	})

	t.Run("CodeSpanWithAngledBrackets_ShouldEscape", func(t *testing.T) {
		source := []byte("<tag>")

		codeSpan := markdown_ast.NewCodeSpan()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		codeSpan.AppendChild(textNode)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		result := tr.TransformNode(context.Background(), codeSpan)

		require.NotNil(t, result)
		require.Len(t, result.Children, 1)
		child := result.Children[0]
		assert.NotNil(t, child.TextContentWriter, "Angle brackets in code spans should be escaped via DirectWriter")
	})
}

func TestTransformer_TransformNode_FragmentFlattening(t *testing.T) {
	t.Run("FragmentChildrenAreFlattened", func(t *testing.T) {
		source := []byte("Text in document")

		document := markdown_ast.NewDocument()

		para := markdown_ast.NewParagraph()
		textNode := markdown_ast.NewText(source)
		textNode.Segment = markdown_ast.Segment{Start: 0, Stop: len(source)}
		para.AppendChild(textNode)
		document.AppendChild(para)

		tr := NewTransformerTestBuilder().
			WithSource(source).
			Build()

		fragmentNode := tr.TransformNode(context.Background(), document)
		require.NotNil(t, fragmentNode)
		assert.Equal(t, ast_domain.NodeFragment, fragmentNode.NodeType)

		children := tr.transformChildren(context.Background(), document, 0)

		require.NotEmpty(t, children)
		assert.Equal(t, "p", children[0].TagName)
	})
}

func TestTransformer_DiagnosticsCollection(t *testing.T) {
	t.Run("TransformerSharesDiagnosticsSlice", func(t *testing.T) {
		source := []byte("Test")
		diagnostics := make([]*ast_domain.Diagnostic, 0)

		tr := newTransformer("test.md", source, newLocationMapper(source), &diagnostics, nil)

		assert.Equal(t, &diagnostics, tr.diagnostics, "Transformer should reference the same diagnostics slice")
	})
}

func TestTransformer_TransformNode_TableCell(t *testing.T) {
	t.Run("TableDataCell", func(t *testing.T) {
		tableCell := markdown_ast.NewTableCell(false)

		tr := NewTransformerTestBuilder().
			WithSource([]byte("cell content")).
			Build()

		result := tr.TransformNode(context.Background(), tableCell)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "td", result.TagName)
	})

	t.Run("TableHeaderCell", func(t *testing.T) {
		tableHeader := markdown_ast.NewTableHeader()
		tableCell := markdown_ast.NewTableCell(true)
		tableHeader.AppendChild(tableCell)

		tr := NewTransformerTestBuilder().
			WithSource([]byte("header content")).
			Build()

		result := tr.TransformNode(context.Background(), tableCell)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "th", result.TagName)
	})
}

func TestTransformer_TransformNode_TaskCheckBox(t *testing.T) {
	t.Run("UncheckedCheckbox", func(t *testing.T) {
		checkbox := markdown_ast.NewTaskCheckBox(false)

		tr := NewTransformerTestBuilder().
			WithSource([]byte("[ ] task")).
			Build()

		result := tr.TransformNode(context.Background(), checkbox)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "input", result.TagName)

		typeAttr, hasType := result.GetAttribute("type")
		assert.True(t, hasType)
		assert.Equal(t, "checkbox", typeAttr)

		_, hasDisabled := result.GetAttribute("disabled")
		assert.True(t, hasDisabled)

		_, hasChecked := result.GetAttribute("checked")
		assert.False(t, hasChecked, "Unchecked checkbox should not have checked attribute")
	})

	t.Run("CheckedCheckbox", func(t *testing.T) {
		checkbox := markdown_ast.NewTaskCheckBox(true)

		tr := NewTransformerTestBuilder().
			WithSource([]byte("[x] task")).
			Build()

		result := tr.TransformNode(context.Background(), checkbox)

		require.NotNil(t, result)
		assert.Equal(t, "input", result.TagName)

		_, hasChecked := result.GetAttribute("checked")
		assert.True(t, hasChecked, "Checked checkbox should have checked attribute")
	})
}

func TestTransformer_TransformNode_Table(t *testing.T) {
	t.Run("TableElement", func(t *testing.T) {
		table := markdown_ast.NewTable()

		tr := NewTransformerTestBuilder().
			WithSource([]byte("| a | b |")).
			Build()

		result := tr.TransformNode(context.Background(), table)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "table", result.TagName)
	})

	t.Run("TableHeaderElement", func(t *testing.T) {
		tableHeader := markdown_ast.NewTableHeader()

		tr := NewTransformerTestBuilder().
			WithSource([]byte("| a | b |")).
			Build()

		result := tr.TransformNode(context.Background(), tableHeader)

		require.NotNil(t, result)
		assert.Equal(t, "thead", result.TagName)
	})

	t.Run("TableRowElement", func(t *testing.T) {
		tableRow := markdown_ast.NewTableRow()

		tr := NewTransformerTestBuilder().
			WithSource([]byte("| a | b |")).
			Build()

		result := tr.TransformNode(context.Background(), tableRow)

		require.NotNil(t, result)
		assert.Equal(t, "tr", result.TagName)
	})
}

func TestTransformer_TransformNode_Strikethrough(t *testing.T) {
	t.Run("StrikethroughText", func(t *testing.T) {
		strikethrough := markdown_ast.NewStrikethrough()

		tr := NewTransformerTestBuilder().
			WithSource([]byte("~~deleted~~")).
			Build()

		result := tr.TransformNode(context.Background(), strikethrough)

		require.NotNil(t, result)
		assert.Equal(t, ast_domain.NodeElement, result.NodeType)
		assert.Equal(t, "del", result.TagName)
	})
}

func TestTransformer_TransformNode_LineBreak(t *testing.T) {
	t.Parallel()

	paragraph := markdown_ast.NewParagraph()
	paragraph.AppendChild(markdown_ast.NewText([]byte("first line")))
	paragraph.AppendChild(markdown_ast.NewLineBreak())
	paragraph.AppendChild(markdown_ast.NewText([]byte("second line")))

	tr := NewTransformerTestBuilder().
		WithSource([]byte("first line  \nsecond line")).
		Build()

	result := tr.TransformNode(context.Background(), paragraph)

	require.NotNil(t, result)
	require.Len(t, result.Children, 3)
	assert.Equal(t, ast_domain.NodeText, result.Children[0].NodeType)
	assert.Equal(t, ast_domain.NodeElement, result.Children[1].NodeType)
	assert.Equal(t, "br", result.Children[1].TagName)
	assert.Empty(t, result.Children[1].Children)
	assert.Equal(t, "second line", result.Children[2].TextContent)
}

func TestTransformer_ExtractNodeText_LineBreak(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		build func() markdown_ast.Node
		want  string
	}{
		{
			name: "a hard line break separates the words either side with a newline",
			build: func() markdown_ast.Node {
				heading := markdown_ast.NewHeading(2)
				heading.AppendChild(markdown_ast.NewText([]byte("Part")))
				heading.AppendChild(markdown_ast.NewLineBreak())
				heading.AppendChild(markdown_ast.NewText([]byte("two")))
				return heading
			},
			want: "Part\ntwo",
		},
		{
			name: "a hard line break nested in emphasis is kept",
			build: func() markdown_ast.Node {
				heading := markdown_ast.NewHeading(2)
				emphasis := markdown_ast.NewEmphasis(1)
				emphasis.AppendChild(markdown_ast.NewText([]byte("a")))
				emphasis.AppendChild(markdown_ast.NewLineBreak())
				emphasis.AppendChild(markdown_ast.NewText([]byte("b")))
				heading.AppendChild(emphasis)
				return heading
			},
			want: "a\nb",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tr := NewTransformerTestBuilder().Build()

			assert.Equal(t, tc.want, tr.extractNodeText(tc.build()))
		})
	}
}

func TestTransformer_TransformChildren_StopsWhenCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("render abandoned"))

	paragraph := markdown_ast.NewParagraph()
	paragraph.AppendChild(markdown_ast.NewText([]byte("one")))
	paragraph.AppendChild(markdown_ast.NewText([]byte("two")))

	tr := NewTransformerTestBuilder().Build()

	result := tr.TransformNode(ctx, paragraph)

	require.NotNil(t, result)
	assert.Empty(t, result.Children)
}

func TestTransformer_DepthLimit(t *testing.T) {
	t.Parallel()

	const deepNesting = 10_000

	testCases := []struct {
		name           string
		levels         int
		wantDiagnostic bool
	}{
		{name: "nesting within the limit is fully rendered", levels: 20, wantDiagnostic: false},
		{name: "nesting at the limit is dropped and reported once", levels: markdown_ast.MaxMarkdownDepth, wantDiagnostic: true},
		{name: "pathological nesting is bounded and reported once", levels: deepNesting, wantDiagnostic: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			document := markdown_ast.NewDocument()
			innermost := nestBlockquotes(document, tc.levels)
			innermost.AppendChild(markdown_ast.NewText([]byte("deep")))
			diagnostics := new([]*ast_domain.Diagnostic)
			tr := NewTransformerTestBuilder().WithDiagnostics(diagnostics).Build()

			result := tr.TransformNode(context.Background(), document)

			require.NotNil(t, result)
			assert.LessOrEqual(t, templateDepth(result), markdown_ast.MaxMarkdownDepth)
			if !tc.wantDiagnostic {
				assert.Empty(t, *diagnostics)
				assert.Equal(t, tc.levels+2, templateDepth(result), "document, every quote and the text")
				return
			}
			require.Len(t, *diagnostics, 1)
			assert.Equal(t, ast_domain.Error, (*diagnostics)[0].Severity)
			assert.Contains(t, (*diagnostics)[0].Message, "deeper than 256 levels")
		})
	}
}

func TestTransformer_ExtractNodeText_DepthLimit(t *testing.T) {
	t.Parallel()

	heading := markdown_ast.NewHeading(1)
	heading.AppendChild(markdown_ast.NewText([]byte("shallow ")))
	var parent markdown_ast.Node = heading
	for range 10_000 {
		emphasis := markdown_ast.NewEmphasis(1)
		parent.AppendChild(emphasis)
		parent = emphasis
	}
	parent.AppendChild(markdown_ast.NewText([]byte("deep")))
	diagnostics := new([]*ast_domain.Diagnostic)
	tr := NewTransformerTestBuilder().WithDiagnostics(diagnostics).Build()

	text := tr.extractNodeText(heading)

	assert.Equal(t, "shallow ", text)
	require.Len(t, *diagnostics, 1)
}

func TestTransformer_GetNodeLocation_DeepInlineNesting(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		levels     int
		wantOffset int
	}{
		{name: "a shallow inline container takes its first text position", levels: 3, wantOffset: 7},
		{name: "a position deeper than the limit is not searched for", levels: 10_000, wantOffset: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			outer := markdown_ast.NewEmphasis(1)
			var parent markdown_ast.Node = outer
			for range tc.levels - 1 {
				emphasis := markdown_ast.NewEmphasis(1)
				parent.AppendChild(emphasis)
				parent = emphasis
			}
			text := markdown_ast.NewText([]byte("x"))
			text.Segment = markdown_ast.Segment{Start: 7, Stop: 8}
			parent.AppendChild(text)
			tr := NewTransformerTestBuilder().Build()

			location := tr.getNodeLocation(outer)

			assert.Equal(t, tc.wantOffset, location.Offset)
		})
	}
}

func Test_transformMarkdownAST_DeepNestingReportsDiagnostic(t *testing.T) {
	t.Parallel()

	document := markdown_ast.NewDocument()
	innermost := nestBlockquotes(document, 10_000)
	innermost.AppendChild(markdown_ast.NewText([]byte("deep")))

	result, err := transformMarkdownAST(context.Background(), document, []byte("deep"), "deep.md", nil)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, "deep.md", result.Diagnostics[0].SourcePath)
}

func nestBlockquotes(root markdown_ast.Node, levels int) markdown_ast.Node {
	parent := root
	for range levels {
		quote := markdown_ast.NewBlockquote()
		parent.AppendChild(quote)
		parent = quote
	}
	return parent
}

func templateDepth(node *ast_domain.TemplateNode) int {
	depth := 1
	for current := node; len(current.Children) > 0; current = current.Children[0] {
		depth++
	}
	return depth
}

func TestTransformer_TransformNode_PikoShortcode(t *testing.T) {
	t.Parallel()

	codeBlock := markdown_ast.NewFencedCodeBlock()
	codeBlock.Info = `piko my-card title="Hello"`
	slotParagraph := markdown_ast.NewParagraph()
	slotParagraph.AppendChild(markdown_ast.NewText([]byte("slot content")))
	codeBlock.AppendChild(slotParagraph)
	diagnostics := new([]*ast_domain.Diagnostic)
	tr := NewTransformerTestBuilder().WithDiagnostics(diagnostics).Build()

	result := tr.TransformNode(context.Background(), codeBlock)

	require.NotNil(t, result)
	assert.Empty(t, *diagnostics)
	assert.Equal(t, "my-card", result.TagName)
	title, found := result.GetAttribute("title")
	require.True(t, found)
	assert.Equal(t, "Hello", title)
	require.Len(t, result.Children, 1)
	assert.Equal(t, "p", result.Children[0].TagName)
}

func TestTransformer_PikoShortcode_DiagnosticLocations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		info       string
		wantAt     string
		wantColumn int
	}{
		{name: "an incomplete binary expression points at the operator", info: `piko my-card :title="1 +"`, wantColumn: 27, wantAt: `+"`},
		{name: "an unclosed group points at the end of the value", info: `piko my-card p-if="(" x="1"`, wantColumn: 24, wantAt: `" x="1"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := "Intro\n\n```" + tc.info + "\n```\n"
			infoStart := strings.Index(source, "piko")
			codeBlock := markdown_ast.NewFencedCodeBlock()
			codeBlock.Info = tc.info
			codeBlock.InfoSegment = markdown_ast.Segment{Start: infoStart, Stop: infoStart + len(tc.info)}
			diagnostics := new([]*ast_domain.Diagnostic)
			tr := NewTransformerTestBuilder().
				WithSource([]byte(source)).
				WithLocationMapper(newLocationMapper([]byte(source))).
				WithDiagnostics(diagnostics).
				Build()

			result := tr.TransformNode(context.Background(), codeBlock)

			assert.Nil(t, result)
			require.Len(t, *diagnostics, 1)
			location := (*diagnostics)[0].Location
			assert.Equal(t, 3, location.Line)
			assert.Equal(t, tc.wantColumn, location.Column)
			fenceLine := strings.Split(source, "\n")[2]
			assert.Equal(t, tc.wantAt, fenceLine[location.Column-1:])
		})
	}
}

func TestTransformer_PikoShortcode_InvalidSyntaxPointsAtFenceLine(t *testing.T) {
	t.Parallel()

	source := "Intro\n\n```piko\n```\n"
	infoStart := strings.Index(source, "piko")
	codeBlock := markdown_ast.NewFencedCodeBlock()
	codeBlock.Info = "piko"
	codeBlock.InfoSegment = markdown_ast.Segment{Start: infoStart, Stop: infoStart + len("piko")}
	diagnostics := new([]*ast_domain.Diagnostic)
	tr := NewTransformerTestBuilder().
		WithSource([]byte(source)).
		WithLocationMapper(newLocationMapper([]byte(source))).
		WithDiagnostics(diagnostics).
		Build()

	location := tr.infoStringLocation(codeBlock, 0)
	_, problems := tr.transformPikoShortcode(context.Background(), "piko", codeBlock, 0)

	assert.Equal(t, ast_domain.Location{Line: 3, Column: 4, Offset: infoStart}, location)
	require.Len(t, problems, 1)
	assert.Equal(t, location, problems[0].Location)
}

func TestTransformer_InfoStringLocation_FallsBackToBlock(t *testing.T) {
	t.Parallel()

	codeBlock := markdown_ast.NewFencedCodeBlock()
	codeBlock.SetLines(markdown_ast.NewSegments(markdown_ast.Segment{Start: 5, Stop: 9}))
	tr := NewTransformerTestBuilder().Build()

	location := tr.infoStringLocation(codeBlock, 3)

	assert.Equal(t, 5, location.Offset)
}

func TestClampToShortcode(t *testing.T) {
	t.Parallel()

	shortcode := ast_domain.Location{Line: 4, Column: 10, Offset: 30}
	inside := &ast_domain.Diagnostic{Location: ast_domain.Location{Line: 4, Column: 15}}
	before := &ast_domain.Diagnostic{Location: ast_domain.Location{Line: 4, Column: 2}}
	otherLine := &ast_domain.Diagnostic{Location: ast_domain.Location{Line: 0, Column: 0}}

	clamped := clampToShortcode([]*ast_domain.Diagnostic{inside, before, otherLine, nil}, shortcode)

	require.Len(t, clamped, 4)
	assert.Equal(t, 15, inside.Location.Column)
	assert.Equal(t, shortcode, before.Location)
	assert.Equal(t, shortcode, otherLine.Location)
}
