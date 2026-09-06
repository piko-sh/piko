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

package markdown_provider_goldmark

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gmast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/markdown/markdown_ast"
	"piko.sh/piko/internal/markdown/markdown_domain"
)

func TestParser_Parse_BasicDocument(t *testing.T) {
	p := NewParser()

	t.Run("EmptyDocument", func(t *testing.T) {
		doc, fm, err := p.Parse(context.Background(), []byte(""))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.Equal(t, markdown_ast.KindDocument, doc.Kind())
		assert.NotNil(t, fm)
	})

	t.Run("SimpleHeading", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("# Hello World"))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.True(t, doc.HasChildren(), "Document should have children")

		first := doc.FirstChild()
		require.NotNil(t, first)
		assert.Equal(t, markdown_ast.KindHeading, first.Kind())

		heading, ok := first.(*markdown_ast.Heading)
		require.True(t, ok)
		assert.Equal(t, 1, heading.Level)
	})

	t.Run("ParagraphWithText", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("Hello world"))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.True(t, doc.HasChildren())

		first := doc.FirstChild()
		require.NotNil(t, first)
		assert.Equal(t, markdown_ast.KindParagraph, first.Kind())

		textChild := first.FirstChild()
		require.NotNil(t, textChild)
		assert.Equal(t, markdown_ast.KindText, textChild.Kind())

		textNode, ok := textChild.(*markdown_ast.Text)
		require.True(t, ok)
		assert.Contains(t, string(textNode.Value), "Hello")
	})

	t.Run("WrappedParagraphKeepsLineBreaks", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("Hello\nworld,\nagain"))

		require.NoError(t, err)
		paragraph := doc.FirstChild()
		require.NotNil(t, paragraph)

		var text []byte
		for child := paragraph.FirstChild(); child != nil; child = child.NextSibling() {
			textNode, ok := child.(*markdown_ast.Text)
			require.True(t, ok)
			text = append(text, textNode.Value...)
		}
		assert.Equal(t, "Hello\nworld,\nagain", string(text))
	})

	t.Run("FencedCodeBlock", func(t *testing.T) {
		input := "```go\nfunc main() {}\n```"
		doc, _, err := p.Parse(context.Background(), []byte(input))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.True(t, doc.HasChildren())

		first := doc.FirstChild()
		require.NotNil(t, first)
		assert.Equal(t, markdown_ast.KindFencedCodeBlock, first.Kind())

		fcb, ok := first.(*markdown_ast.FencedCodeBlock)
		require.True(t, ok)
		assert.Equal(t, "go", fcb.Language)
		assert.NotEmpty(t, fcb.Content)
	})

	t.Run("HTMLBlock", func(t *testing.T) {
		input := "<div>Hello</div>\n"
		doc, _, err := p.Parse(context.Background(), []byte(input))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.True(t, doc.HasChildren())

		first := doc.FirstChild()
		require.NotNil(t, first)
		assert.Equal(t, markdown_ast.KindHTMLBlock, first.Kind())

		hb, ok := first.(*markdown_ast.HTMLBlock)
		require.True(t, ok)
		assert.NotEmpty(t, hb.Content)
	})
}

func TestParser_Parse_InlineElements(t *testing.T) {
	p := NewParser()

	t.Run("Emphasis", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("*italic*"))

		require.NoError(t, err)
		para := doc.FirstChild()
		require.NotNil(t, para)

		em := para.FirstChild()
		require.NotNil(t, em)
		assert.Equal(t, markdown_ast.KindEmphasis, em.Kind())
		assert.Equal(t, 1, em.(*markdown_ast.Emphasis).Level)
	})

	t.Run("Strong", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("**bold**"))

		require.NoError(t, err)
		para := doc.FirstChild()
		require.NotNil(t, para)

		strong := para.FirstChild()
		require.NotNil(t, strong)
		assert.Equal(t, markdown_ast.KindEmphasis, strong.Kind())
		assert.Equal(t, 2, strong.(*markdown_ast.Emphasis).Level)
	})

	t.Run("Link", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("[text](https://example.com)"))

		require.NoError(t, err)
		para := doc.FirstChild()
		require.NotNil(t, para)

		link := para.FirstChild()
		require.NotNil(t, link)
		assert.Equal(t, markdown_ast.KindLink, link.Kind())
		assert.Equal(t, "https://example.com", string(link.(*markdown_ast.Link).Destination))
	})

	t.Run("Image", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("![alt](image.png)"))

		require.NoError(t, err)
		para := doc.FirstChild()
		require.NotNil(t, para)

		img := para.FirstChild()
		require.NotNil(t, img)
		assert.Equal(t, markdown_ast.KindImage, img.Kind())
		assert.Equal(t, "image.png", string(img.(*markdown_ast.Image).Destination))
	})

	t.Run("CodeSpan", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("`code`"))

		require.NoError(t, err)
		para := doc.FirstChild()
		require.NotNil(t, para)

		cs := para.FirstChild()
		require.NotNil(t, cs)
		assert.Equal(t, markdown_ast.KindCodeSpan, cs.Kind())
	})
}

func TestParser_Parse_GFMExtensions(t *testing.T) {
	p := NewParser()

	t.Run("Table", func(t *testing.T) {
		input := "| A | B |\n| - | - |\n| 1 | 2 |\n"
		doc, _, err := p.Parse(context.Background(), []byte(input))

		require.NoError(t, err)
		require.NotNil(t, doc)
		assert.True(t, doc.HasChildren())

		table := doc.FirstChild()
		require.NotNil(t, table)
		assert.Equal(t, markdown_ast.KindTable, table.Kind())
	})

	t.Run("Strikethrough", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("~~deleted~~"))

		require.NoError(t, err)
		para := doc.FirstChild()
		require.NotNil(t, para)

		strike := para.FirstChild()
		require.NotNil(t, strike)
		assert.Equal(t, markdown_ast.KindStrikethrough, strike.Kind())
	})

	t.Run("TaskCheckBox", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("- [x] Done\n- [ ] Not done"))

		require.NoError(t, err)
		list := doc.FirstChild()
		require.NotNil(t, list)
		assert.Equal(t, markdown_ast.KindList, list.Kind())
	})
}

func TestParser_Parse_Frontmatter(t *testing.T) {
	p := NewParser()

	t.Run("ExtractsFrontmatter", func(t *testing.T) {
		input := "---\ntitle: Test Post\nauthor: Jane\n---\n\n# Hello"
		doc, fm, err := p.Parse(context.Background(), []byte(input))

		require.NoError(t, err)
		require.NotNil(t, doc)
		require.NotNil(t, fm)
		assert.Equal(t, "Test Post", fm["title"])
		assert.Equal(t, "Jane", fm["author"])
	})

	t.Run("EmptyFrontmatterReturnsEmptyMap", func(t *testing.T) {
		doc, fm, err := p.Parse(context.Background(), []byte("# Hello"))

		require.NoError(t, err)
		require.NotNil(t, doc)
		require.NotNil(t, fm)
		assert.Empty(t, fm)
	})
}

func TestParser_Parse_ParentPointers(t *testing.T) {
	p := NewParser()

	t.Run("ChildrenHaveCorrectParent", func(t *testing.T) {
		doc, _, err := p.Parse(context.Background(), []byte("# Hello\n\nworld"))

		require.NoError(t, err)
		heading := doc.FirstChild()
		require.NotNil(t, heading)
		assert.Equal(t, doc, heading.Parent(), "Heading parent should be document")

		textChild := heading.FirstChild()
		if textChild != nil {
			assert.Equal(t, heading, textChild.Parent(), "Text parent should be heading")
		}
	})

	t.Run("TableCellParentIsHeader", func(t *testing.T) {
		input := "| A | B |\n| - | - |\n| 1 | 2 |\n"
		doc, _, err := p.Parse(context.Background(), []byte(input))

		require.NoError(t, err)
		table := doc.FirstChild()
		require.NotNil(t, table)

		header := table.FirstChild()
		require.NotNil(t, header)
		assert.Equal(t, markdown_ast.KindTableHeader, header.Kind())

		cell := header.FirstChild()
		if cell != nil {
			assert.Equal(t, header, cell.Parent(), "Table cell parent should be table header")
			assert.Equal(t, markdown_ast.KindTableHeader, cell.Parent().Kind())
		}
	})
}

func TestParser_Parse_LineBreaks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantPath []int
		want     []string
	}{
		{
			name:  "two trailing spaces produce a hard line break",
			input: "Hello  \nworld",
			want:  []string{"text:Hello", "break", "text:world"},
		},
		{
			name:  "a trailing backslash produces a hard line break",
			input: "Hello\\\nworld",
			want:  []string{"text:Hello", "break", "text:world"},
		},
		{
			name:  "a plain line ending stays a soft break inside the text",
			input: "Hello\nworld",
			want:  []string{"text:Hello\n", "text:world"},
		},
		{
			name:     "a hard line break inside emphasis stays inside the emphasis",
			input:    "*Hello  \nworld*",
			wantPath: []int{0},
			want:     []string{"text:Hello", "break", "text:world"},
		},
		{
			name:  "trailing spaces at the end of a paragraph are not a break",
			input: "Hello  ",
			want:  []string{"text:Hello"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, _, err := NewParser().Parse(context.Background(), []byte(tc.input))
			require.NoError(t, err)

			container := doc.FirstChild()
			require.NotNil(t, container)
			for _, index := range tc.wantPath {
				container = nthChild(t, container, index)
			}

			assert.Equal(t, tc.want, describeInlineChildren(container))
		})
	}
}

func TestParser_Parse_HardLineBreakRendersAsBreakElement(t *testing.T) {
	t.Parallel()

	service := markdown_domain.NewMarkdownService(NewParser(), nil)

	processed, err := service.Process(context.Background(), []byte("---\ntitle: Page\n---\nHello  \nworld"), "page.md")

	require.NoError(t, err)
	require.Len(t, processed.PageAST.RootNodes, 1)
	paragraph := processed.PageAST.RootNodes[0]
	require.Len(t, paragraph.Children, 3)
	assert.Equal(t, "Hello", paragraph.Children[0].TextContent)
	assert.Equal(t, ast_domain.NodeElement, paragraph.Children[1].NodeType)
	assert.Equal(t, "br", paragraph.Children[1].TagName)
	assert.Equal(t, "world", paragraph.Children[2].TextContent)
	assert.Equal(t, 2, processed.Metadata.WordCount)
}

func nthChild(t *testing.T, parent markdown_ast.Node, index int) markdown_ast.Node {
	t.Helper()

	child := parent.FirstChild()
	for range index {
		require.NotNil(t, child)
		child = child.NextSibling()
	}
	require.NotNil(t, child)
	return child
}

func describeInlineChildren(parent markdown_ast.Node) []string {
	var described []string
	for child := parent.FirstChild(); child != nil; child = child.NextSibling() {
		switch typed := child.(type) {
		case *markdown_ast.Text:
			described = append(described, "text:"+string(typed.Value))
		case *markdown_ast.LineBreak:
			described = append(described, "break")
		default:
			described = append(described, "other")
		}
	}
	return described
}

func TestParser_ConvertNode_DepthLimit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		levels    int
		wantDepth int
	}{
		{name: "nesting within the limit is converted in full", levels: 20, wantDepth: 22},
		{name: "pathological nesting stops one level past the limit", levels: 10_000, wantDepth: markdown_ast.MaxMarkdownDepth + 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := []byte("deep")
			root := gmast.NewDocument()
			var parent gmast.Node = root
			for range tc.levels {
				quote := gmast.NewBlockquote()
				parent.AppendChild(parent, quote)
				parent = quote
			}
			parent.AppendChild(parent, gmast.NewTextSegment(text.NewSegment(0, len(source))))

			converted := NewParser().convertNode(root, source, 0)

			assert.Equal(t, tc.wantDepth, chainDepth(converted))
		})
	}
}

func TestParser_Parse_DeepNestingIsReportedByTheMarkdownService(t *testing.T) {
	t.Parallel()

	input := "---\ntitle: Deep\n---\n" + strings.Repeat(">", markdown_ast.MaxMarkdownDepth+20) + " deep"
	service := markdown_domain.NewMarkdownService(NewParser(), nil)

	processed, err := service.Process(context.Background(), []byte(input), "deep.md")

	require.NoError(t, err)
	require.Len(t, processed.Diagnostics, 1)
	assert.Contains(t, processed.Diagnostics[0].Message, "deeper than")
}

func chainDepth(node markdown_ast.Node) int {
	depth := 0
	for current := node; current != nil; current = current.FirstChild() {
		depth++
	}
	return depth
}

func TestParser_Parse_FallbackAndRawHTMLNodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		wantPath []int
		wantKind markdown_ast.NodeKind
	}{
		{name: "an unhandled block becomes a text block", input: "a\n\n***\n\nb", wantPath: []int{1}, wantKind: markdown_ast.KindTextBlock},
		{name: "an unhandled inline becomes a code span", input: "see <https://example.test>", wantPath: []int{0, 1}, wantKind: markdown_ast.KindCodeSpan},
		{name: "inline HTML keeps its source", input: "a <span>b</span>", wantPath: []int{0, 1}, wantKind: markdown_ast.KindRawHTML},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, _, err := NewParser().Parse(context.Background(), []byte(tc.input))
			require.NoError(t, err)

			var node markdown_ast.Node = doc
			for _, index := range tc.wantPath {
				node = nthChild(t, node, index)
			}

			assert.Equal(t, tc.wantKind, node.Kind())
			if rawHTML, ok := node.(*markdown_ast.RawHTML); ok {
				assert.Equal(t, "<span>", string(bytes.Join(rawHTML.Content, nil)))
			}
		})
	}
}

func TestParser_Parse_ShortcodeDiagnosticsUseSourcePositions(t *testing.T) {
	t.Parallel()

	input := "---\ntitle: Page\n---\n# Heading\n\n```piko my-card :title=\"1 +\"\n```\n"
	service := markdown_domain.NewMarkdownService(NewParser(), nil)

	processed, err := service.Process(context.Background(), []byte(input), "page.md")

	require.NoError(t, err)
	require.Len(t, processed.Diagnostics, 1)
	assert.Equal(t, 6, processed.Diagnostics[0].Location.Line)
	assert.Equal(t, 27, processed.Diagnostics[0].Location.Column)
	assert.Equal(t, "page.md", processed.Diagnostics[0].SourcePath)
}

func TestParser_Parse_FencedCodeBlockInfoSegment(t *testing.T) {
	t.Parallel()

	source := []byte("text\n\n```go title=main.go\nx\n```\n")

	doc, _, err := NewParser().Parse(context.Background(), source)
	require.NoError(t, err)

	codeBlock, ok := nthChild(t, doc, 1).(*markdown_ast.FencedCodeBlock)
	require.True(t, ok)
	assert.Equal(t, "go title=main.go", string(codeBlock.InfoSegment.Value(source)))
}
