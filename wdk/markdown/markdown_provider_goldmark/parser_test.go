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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/markdown/markdown_ast"
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
