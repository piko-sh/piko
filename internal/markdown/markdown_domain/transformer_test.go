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

		children := tr.transformChildren(context.Background(), document)

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
