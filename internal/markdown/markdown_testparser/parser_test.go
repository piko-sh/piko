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

package markdown_testparser

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/markdown/markdown_ast"
)

func TestParser_Parse_Blocks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty input yields an empty document", input: "", want: "doc"},
		{name: "blank lines are skipped", input: "\n\n   \n", want: "doc"},
		{name: "atx heading", input: "# Hello World", want: `doc(h1("Hello World"))`},
		{name: "closing hashes are stripped", input: "## Title ##", want: `doc(h2("Title"))`},
		{name: "heading made only of closing hashes is empty", input: "## ###", want: "doc(h2)"},
		{name: "bare hash is an empty heading", input: "#", want: "doc(h1)"},
		{name: "seven hashes are not a heading", input: "####### seven", want: `doc(p("####### seven"))`},
		{name: "hash without a space is not a heading", input: "#tag", want: `doc(p("#tag"))`},
		{name: "wrapped paragraph lines are joined with spaces", input: "Hello\n  world  \nagain", want: `doc(p("Hello world again"))`},
		{name: "horizontal rules are dropped", input: "one\n\n---\n\n* * *\n\n___\n\ntwo", want: `doc(p("one") p("two"))`},
		{name: "dashes mixed with text are not a rule", input: "--x", want: `doc(p("--x"))`},
		{
			name:  "fenced code block with language and info",
			input: "```go title=main.go\nfmt.Println()\n```\nafter",
			want:  `doc(code[go|go title=main.go|"fmt.Println()\n"] p("after"))`,
		},
		{
			name:  "tilde fence closed by a longer fence",
			input: "~~~\nx\n~~~~",
			want:  `doc(code[||"x\n"])`,
		},
		{
			name:  "unclosed fence runs to the end of the input",
			input: "```\nline one\nline two",
			want:  `doc(code[||"line one\nline two"])`,
		},
		{
			name:  "shorter closing fence does not close the block",
			input: "````\n```\n````",
			want:  "doc(code[||\"```\\n\"])",
		},
		{
			name:  "blockquote lines join into one paragraph",
			input: "> quoted\n> more",
			want:  `doc(quote(p("quoted more")))`,
		},
		{
			name:  "empty quote line separates paragraphs",
			input: "> first\n>\n> second",
			want:  `doc(quote(p("first") p("second")))`,
		},
		{
			name:  "lazy continuation stays in the quote",
			input: "> first\nlazy\n\nafter",
			want:  `doc(quote(p("first lazy")) p("after"))`,
		},
		{
			name:  "unordered list with continuation lines",
			input: "- one\n  continued\n- two\n\nafter",
			want:  `doc(ul(li(p("one continued")) li(p("two"))) p("after"))`,
		},
		{
			name:  "unordered list markers",
			input: "* star\n+ plus",
			want:  `doc(ul(li(p("star")) li(p("plus"))))`,
		},
		{
			name:  "ordered list with both delimiters",
			input: "1. first\n2) second",
			want:  `doc(ol(li(p("first")) li(p("second"))))`,
		},
		{
			name:  "list ends at a different list kind",
			input: "- bullet\n1. number",
			want:  `doc(ul(li(p("bullet"))) ol(li(p("number"))))`,
		},
		{
			name:  "paragraph ends at a block boundary",
			input: "text\n# Heading\ntext\n```\ncode\n```",
			want:  `doc(p("text") h1("Heading") p("text") code[||"code\n"])`,
		},
		{
			name:  "paragraph ends at quotes and lists",
			input: "a\n> q\n\nb\n- item\n\nc\n1. item",
			want:  `doc(p("a") quote(p("q")) p("b") ul(li(p("item"))) p("c") ol(li(p("item"))))`,
		},
		{
			name:  "paragraph ends at a horizontal rule",
			input: "a\n***\nb",
			want:  `doc(p("a") p("b"))`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, frontmatter, err := NewParser().Parse(context.Background(), []byte(tc.input))

			require.NoError(t, err)
			require.NotNil(t, doc)
			assert.Empty(t, frontmatter)
			assert.Equal(t, tc.want, describeNode(doc))
		})
	}
}

func TestParser_Parse_Inlines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "code span", input: "use `fmt` here", want: `doc(p("use " code("fmt") " here"))`},
		{name: "unclosed code span is text", input: "use `fmt", want: "doc(p(\"use `fmt\"))"},
		{name: "emphasis", input: "an *em* word", want: `doc(p("an " em1("em") " word"))`},
		{name: "strong with underscores", input: "__strong__", want: `doc(p(em2("strong")))`},
		{name: "triple delimiters are treated as strong", input: "***both***", want: `doc(p(em2("*both") "*"))`},
		{name: "unclosed emphasis is text", input: "an *open", want: `doc(p("an *open"))`},
		{name: "nested emphasis inside strong", input: "**a *b* c**", want: `doc(p(em2("a " em1("b") " c")))`},
		{
			name:  "link with a double-quoted title",
			input: `see [docs](https://example.test "The Docs")`,
			want:  `doc(p("see " a[https://example.test|The Docs]("docs")))`,
		},
		{
			name:  "link with a single-quoted title",
			input: `[docs](/path 'Title')`,
			want:  `doc(p(a[/path|Title]("docs")))`,
		},
		{name: "link without a title", input: "[docs](/path)", want: `doc(p(a[/path|]("docs")))`},
		{name: "link text with nested brackets", input: "[a [b] c](/x)", want: `doc(p(a[/x|]("a [b] c")))`},
		{name: "brackets without a destination are text", input: "[docs] here", want: `doc(p("[docs] here"))`},
		{name: "unclosed bracket is text", input: "[docs", want: `doc(p("[docs"))`},
		{name: "unclosed destination is text", input: "[docs](/path", want: `doc(p("[docs](/path"))`},
		{name: "image", input: "![alt text](/image.png)", want: `doc(p(img[/image.png|]("alt text")))`},
		{name: "image with title", input: `![alt](/i.png "Caption")`, want: `doc(p(img[/i.png|Caption]("alt")))`},
		{name: "image without a destination is text", input: "![alt] x", want: `doc(p("![alt] x"))`},
		{name: "unclosed image alt is text", input: "![alt", want: `doc(p("![alt"))`},
		{name: "unclosed image destination is text", input: "![alt](/i.png", want: `doc(p("![alt](/i.png"))`},
		{name: "trailing exclamation is text", input: "wow!", want: `doc(p("wow!"))`},
		{name: "inline constructs in headings", input: "# The *best* `code`", want: `doc(h1("The " em1("best") " " code("code")))`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, _, err := NewParser().Parse(context.Background(), []byte(tc.input))

			require.NoError(t, err)
			assert.Equal(t, tc.want, describeNode(doc))
		})
	}
}

func TestParser_Parse_HeadingIDs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		wantID string
	}{
		{name: "words become a kebab-case slug", input: "# Hello World", wantID: "hello-world"},
		{name: "punctuation is dropped", input: "## Hello, World! 123", wantID: "hello-world-123"},
		{name: "separators collapse into one dash", input: "# a _ b--c", wantID: "a-b-c"},
		{name: "letters outside ASCII are kept", input: "# Café Über", wantID: "café-über"},
		{name: "an empty heading falls back to a fixed id", input: "#", wantID: "heading"},
		{name: "a heading of only punctuation falls back to a fixed id", input: "# !!!", wantID: "heading"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, _, err := NewParser().Parse(context.Background(), []byte(tc.input))
			require.NoError(t, err)

			heading, ok := doc.FirstChild().(*markdown_ast.Heading)
			require.True(t, ok, "first block should be a heading")
			id, found := heading.AttributeString("id")
			require.True(t, found)
			assert.Equal(t, tc.wantID, id)
		})
	}
}

func TestParser_Parse_Frontmatter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		input           string
		wantFrontmatter map[string]any
		wantDocument    string
	}{
		{
			name:            "no frontmatter leaves the body untouched",
			input:           "Body text",
			wantFrontmatter: map[string]any{},
			wantDocument:    `doc(p("Body text"))`,
		},
		{
			name:            "frontmatter is split from the body",
			input:           "---\ntitle: Hello\ncount: 3\n---\nBody",
			wantFrontmatter: map[string]any{"title": "Hello", "count": 3},
			wantDocument:    `doc(p("Body"))`,
		},
		{
			name:  "dates are returned as strings, including nested ones",
			input: "---\ndate: 2024-01-02\nmeta:\n  updated: 2024-03-04\n---\nBody",
			wantFrontmatter: map[string]any{
				"date": "2024-01-02",
				"meta": map[string]any{"updated": "2024-03-04"},
			},
			wantDocument: `doc(p("Body"))`,
		},
		{
			name:            "windows line endings are accepted",
			input:           "---\r\ntitle: Hello\r\n---\r\nBody",
			wantFrontmatter: map[string]any{"title": "Hello"},
			wantDocument:    `doc(p("Body"))`,
		},
		{
			name:            "frontmatter that closes at the end of the input has no body",
			input:           "---\ntitle: Only\n---",
			wantFrontmatter: map[string]any{"title": "Only"},
			wantDocument:    "doc",
		},
		{
			name:            "unterminated frontmatter is treated as body",
			input:           "---\ntitle: open\nstill body",
			wantFrontmatter: map[string]any{},
			wantDocument:    `doc(p("title: open still body"))`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, frontmatter, err := NewParser().Parse(context.Background(), []byte(tc.input))

			require.NoError(t, err)
			assert.Equal(t, tc.wantFrontmatter, frontmatter)
			assert.Equal(t, tc.wantDocument, describeNode(doc))
		})
	}
}

func TestParser_Parse_SegmentsPointIntoTheSource(t *testing.T) {
	t.Parallel()

	source := []byte("# Title\n\nSome *bold* text")

	doc, _, err := NewParser().Parse(context.Background(), source)
	require.NoError(t, err)

	heading := doc.FirstChild()
	require.NotNil(t, heading)
	require.Equal(t, 1, heading.Lines().Len())
	assert.Equal(t, "Title", string(heading.Lines().At(0).Value(source)))

	paragraph := heading.NextSibling()
	require.NotNil(t, paragraph)
	require.Equal(t, 1, paragraph.Lines().Len())
	assert.Equal(t, "Some *bold* text", string(paragraph.Lines().At(0).Value(source)))

	var texts []string
	markdown_ast.Walk(paragraph, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
		if text, ok := node.(*markdown_ast.Text); ok && entering {
			texts = append(texts, string(text.Segment.Value(source)))
		}
		return markdown_ast.WalkContinue
	})
	assert.Equal(t, []string{"Some ", "bold", " text"}, texts)
}

func TestParser_Parse_SetsParentPointers(t *testing.T) {
	t.Parallel()

	doc, _, err := NewParser().Parse(context.Background(), []byte("- *item*"))
	require.NoError(t, err)

	markdown_ast.Walk(doc, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
		if !entering {
			return markdown_ast.WalkContinue
		}
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			assert.Same(t, node, child.Parent())
		}
		return markdown_ast.WalkContinue
	})
}

func describeNode(node markdown_ast.Node) string {
	var builder strings.Builder
	describeInto(&builder, node)
	return builder.String()
}

func describeInto(builder *strings.Builder, node markdown_ast.Node) {
	builder.WriteString(nodeLabel(node))
	if !node.HasChildren() {
		return
	}
	builder.WriteByte('(')
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child != node.FirstChild() {
			builder.WriteByte(' ')
		}
		describeInto(builder, child)
	}
	builder.WriteByte(')')
}

func nodeLabel(node markdown_ast.Node) string {
	switch typed := node.(type) {
	case *markdown_ast.Document:
		return "doc"
	case *markdown_ast.Paragraph:
		return "p"
	case *markdown_ast.Heading:
		return fmt.Sprintf("h%d", typed.Level)
	case *markdown_ast.Blockquote:
		return "quote"
	case *markdown_ast.List:
		if typed.IsOrdered {
			return "ol"
		}
		return "ul"
	case *markdown_ast.ListItem:
		return "li"
	case *markdown_ast.FencedCodeBlock:
		return fmt.Sprintf("code[%s|%s|%q]", typed.Language, typed.Info, bytes.Join(typed.Content, nil))
	case *markdown_ast.Text:
		return strconv.Quote(string(typed.Value))
	case *markdown_ast.Emphasis:
		return fmt.Sprintf("em%d", typed.Level)
	case *markdown_ast.Link:
		return fmt.Sprintf("a[%s|%s]", typed.Destination, typed.Title)
	case *markdown_ast.Image:
		return fmt.Sprintf("img[%s|%s]", typed.Destination, typed.Title)
	case *markdown_ast.CodeSpan:
		return "code"
	default:
		return fmt.Sprintf("unexpected(%d)", node.Kind())
	}
}

func TestParser_Parse_FencedCodeBlockInfoSegment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   string
	}{
		{name: "info after the fence", source: "```go title=main.go\nx\n```", want: "go title=main.go"},
		{name: "indented fence with padded info", source: "text\n\n  ~~~   piko card  \n~~~", want: "piko card"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, _, err := NewParser().Parse(context.Background(), []byte(tc.source))
			require.NoError(t, err)

			var codeBlock *markdown_ast.FencedCodeBlock
			markdown_ast.Walk(doc, func(node markdown_ast.Node, entering bool) markdown_ast.WalkStatus {
				if block, ok := node.(*markdown_ast.FencedCodeBlock); ok && entering {
					codeBlock = block
				}
				return markdown_ast.WalkContinue
			})
			require.NotNil(t, codeBlock)
			assert.Equal(t, tc.want, string(codeBlock.InfoSegment.Value([]byte(tc.source))))
		})
	}
}
