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

// internal/render/render_domain/plaintext_walker_test.go

package render_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func textNode(content string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: content,
	}
}

func elementNode(tagName string, attrs []ast_domain.HTMLAttribute, children ...*ast_domain.TemplateNode) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    tagName,
		Attributes: attrs,
		Children:   children,
	}
}

func attr(name, value string) ast_domain.HTMLAttribute {
	return ast_domain.HTMLAttribute{Name: name, Value: value}
}

func walkNodes(t *testing.T, nodes ...*ast_domain.TemplateNode) string {
	t.Helper()
	walker := newPlainTextWalker()
	ast := &ast_domain.TemplateAST{RootNodes: nodes}
	result, err := walker.Walk(ast)
	require.NoError(t, err)
	return result
}

func TestPlainTextWalker_SimpleText(t *testing.T) {
	result := walkNodes(t, textNode("Hello, World!"))
	assert.Equal(t, "Hello, World!", result)
}

func TestPlainTextWalker_WhitespaceCollapsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multiple spaces",
			input:    "Hello     World",
			expected: "Hello World",
		},
		{
			name:     "tabs and newlines",
			input:    "Hello\t\n\nWorld",
			expected: "Hello World",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "   Hello World   ",
			expected: "Hello World",
		},
		{
			name:     "mixed whitespace",
			input:    "  Hello  \t\n  World  ",
			expected: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := walkNodes(t, textNode(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlainTextWalker_HTMLEntityDecoding(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hair space entity",
			input:    "Hello&#8202;World",
			expected: "Hello World",
		},
		{
			name:     "nbsp entity",
			input:    "Hello&nbsp;World",
			expected: "Hello World",
		},
		{
			name:     "common entities",
			input:    "&lt;tag&gt; &amp; &quot;text&quot;",
			expected: "<tag> & \"text\"",
		},
		{
			name:     "multiple entities",
			input:    "Price: &pound;100 &euro;90",
			expected: "Price: £100 €90",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := walkNodes(t, textNode(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlainTextWalker_Paragraphs(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		elementNode("p", nil, textNode("First paragraph.")),
		elementNode("p", nil, textNode("Second paragraph.")),
	}
	result := walkNodes(t, nodes...)
	expected := "First paragraph.\n\nSecond paragraph."
	assert.Equal(t, expected, result)
}

func TestPlainTextWalker_Divs(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		elementNode("div", nil, textNode("First div.")),
		elementNode("div", nil, textNode("Second div.")),
	}
	result := walkNodes(t, nodes...)
	expected := "First div.\n\nSecond div."
	assert.Equal(t, expected, result)
}

func TestPlainTextWalker_Blockquote(t *testing.T) {
	node := elementNode("blockquote", nil, textNode("This is a quote."))
	result := walkNodes(t, node)
	assert.Equal(t, "This is a quote.", result)
}

func TestPlainTextWalker_H1(t *testing.T) {
	node := elementNode("h1", nil, textNode("Main Title"))
	result := walkNodes(t, node)

	lines := strings.Split(result, "\n")
	require.Len(t, lines, 2, "H1 should produce 2 lines (text + underline)")
	assert.Equal(t, "Main Title", lines[0])

	assert.Regexp(t, "^=+$", lines[1], "H1 underline should be equals signs")
}

func TestPlainTextWalker_H2(t *testing.T) {
	node := elementNode("h2", nil, textNode("Subtitle"))
	result := walkNodes(t, node)

	lines := strings.Split(result, "\n")
	require.Len(t, lines, 2, "H2 should produce 2 lines (text + underline)")
	assert.Equal(t, "Subtitle", lines[0])
	assert.Regexp(t, "^-+$", lines[1], "H2 underline should be dashes")
}

func TestPlainTextWalker_H3ToH6(t *testing.T) {
	tests := []struct {
		tag      string
		expected string
	}{
		{tag: "h3", expected: "Heading 3"},
		{tag: "h4", expected: "Heading 4"},
		{tag: "h5", expected: "Heading 5"},
		{tag: "h6", expected: "Heading 6"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			node := elementNode(tt.tag, nil, textNode(tt.expected))
			result := walkNodes(t, node)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlainTextWalker_UnorderedList(t *testing.T) {
	ul := elementNode("ul", nil,
		elementNode("li", nil, textNode("First item")),
		elementNode("li", nil, textNode("Second item")),
		elementNode("li", nil, textNode("Third item")),
	)
	result := walkNodes(t, ul)
	expected := "* First item\n* Second item\n* Third item"
	assert.Equal(t, expected, result)
}

func TestPlainTextWalker_OrderedList(t *testing.T) {
	ol := elementNode("ol", nil,
		elementNode("li", nil, textNode("First")),
		elementNode("li", nil, textNode("Second")),
		elementNode("li", nil, textNode("Third")),
	)
	result := walkNodes(t, ol)
	expected := "1. First\n2. Second\n3. Third"
	assert.Equal(t, expected, result)
}

func TestPlainTextWalker_NestedLists(t *testing.T) {

	ul := elementNode("ul", nil,
		elementNode("li", nil, textNode("Item 1")),
		elementNode("li", nil,
			textNode("Item 2"),
			elementNode("ol", nil,
				elementNode("li", nil, textNode("Nested 1")),
				elementNode("li", nil, textNode("Nested 2")),
			),
		),
		elementNode("li", nil, textNode("Item 3")),
	)
	result := walkNodes(t, ul)

	assert.Contains(t, result, "* Item 1")
	assert.Contains(t, result, "* Item 2")
	assert.Contains(t, result, "  1. Nested 1")
	assert.Contains(t, result, "  2. Nested 2")
	assert.Contains(t, result, "* Item 3")
}

func TestPlainTextWalker_DeeplyNestedLists(t *testing.T) {

	ul := elementNode("ul", nil,
		elementNode("li", nil,
			textNode("Level 1"),
			elementNode("ul", nil,
				elementNode("li", nil,
					textNode("Level 2"),
					elementNode("ul", nil,
						elementNode("li", nil, textNode("Level 3")),
					),
				),
			),
		),
	)
	result := walkNodes(t, ul)

	assert.Contains(t, result, "* Level 1")
	assert.Contains(t, result, "  * Level 2")
	assert.Contains(t, result, "    * Level 3")
}

func TestPlainTextWalker_SimpleTable(t *testing.T) {
	table := elementNode("table", nil,
		elementNode("tr", nil,
			elementNode("th", nil, textNode("Name")),
			elementNode("th", nil, textNode("Age")),
		),
		elementNode("tr", nil,
			elementNode("td", nil, textNode("Alice")),
			elementNode("td", nil, textNode("30")),
		),
		elementNode("tr", nil,
			elementNode("td", nil, textNode("Bob")),
			elementNode("td", nil, textNode("25")),
		),
	)
	result := walkNodes(t, table)

	assert.Contains(t, result, "Name | Age")
	assert.Contains(t, result, "Alice | 30")
	assert.Contains(t, result, "Bob | 25")

	assert.NotContains(t, result, "| Name")
}

func TestPlainTextWalker_TableWithMultipleColumns(t *testing.T) {
	table := elementNode("table", nil,
		elementNode("tr", nil,
			elementNode("td", nil, textNode("Col1")),
			elementNode("td", nil, textNode("Col2")),
			elementNode("td", nil, textNode("Col3")),
			elementNode("td", nil, textNode("Col4")),
		),
	)
	result := walkNodes(t, table)

	expected := "Col1 | Col2 | Col3 | Col4"
	assert.Equal(t, expected, result)
}

func TestPlainTextWalker_SimpleLink(t *testing.T) {
	link := elementNode("a", []ast_domain.HTMLAttribute{attr("href", "https://example.com")},
		textNode("Click here"),
	)
	result := walkNodes(t, link)

	assert.Equal(t, "Click here (https://example.com)", result)
}

func TestPlainTextWalker_LinkWithoutHref(t *testing.T) {
	link := elementNode("a", nil, textNode("Not a real link"))
	result := walkNodes(t, link)

	assert.Equal(t, "Not a real link", result)
}

func TestPlainTextWalker_FragmentLink(t *testing.T) {
	link := elementNode("a", []ast_domain.HTMLAttribute{attr("href", "#section")},
		textNode("Go to section"),
	)
	result := walkNodes(t, link)

	assert.Equal(t, "Go to section", result)
	assert.NotContains(t, result, "#section")
}

func TestPlainTextWalker_NestedLinkWithImage(t *testing.T) {
	link := elementNode("a", []ast_domain.HTMLAttribute{attr("href", "https://example.com")},
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "Logo")}, nil),
	)
	result := walkNodes(t, link)

	assert.Equal(t, "Logo (https://example.com)", result)
}

func TestPlainTextWalker_Button(t *testing.T) {
	button := elementNode("pml-button", []ast_domain.HTMLAttribute{attr("href", "https://shop.com/buy")},
		textNode("Buy Now"),
	)
	result := walkNodes(t, button)

	assert.Contains(t, result, "Buy Now")
	assert.Contains(t, result, "https://shop.com/buy")

	assert.Contains(t, result, "[https://shop.com/buy]")
}

func TestPlainTextWalker_Bold(t *testing.T) {
	tests := []struct {
		tag      string
		text     string
		expected string
	}{
		{tag: "strong", text: "Important", expected: "**Important**"},
		{tag: "b", text: "Bold text", expected: "**Bold text**"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			node := elementNode(tt.tag, nil, textNode(tt.text))
			result := walkNodes(t, node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlainTextWalker_Italic(t *testing.T) {
	tests := []struct {
		tag      string
		text     string
		expected string
	}{
		{tag: "em", text: "Emphasis", expected: "*Emphasis*"},
		{tag: "i", text: "Italic text", expected: "*Italic text*"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			node := elementNode(tt.tag, nil, textNode(tt.text))
			result := walkNodes(t, node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlainTextWalker_CombinedEmphasis(t *testing.T) {

	node := elementNode("strong", nil,
		elementNode("em", nil, textNode("Very important")),
	)
	result := walkNodes(t, node)
	assert.Equal(t, "***Very important***", result)
}

func TestPlainTextWalker_ImageWithAlt(t *testing.T) {
	img := elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "Company Logo")}, nil)
	result := walkNodes(t, img)
	assert.Equal(t, "Company Logo", result)
}

func TestPlainTextWalker_ImageWithPlaintextAlt(t *testing.T) {

	img := elementNode("img", []ast_domain.HTMLAttribute{
		attr("alt", "Visual description"),
		attr("p-plaintext-alt", "Text-only description"),
	}, nil)
	result := walkNodes(t, img)
	assert.Equal(t, "Text-only description", result)
}

func TestPlainTextWalker_ImageWithoutAlt(t *testing.T) {
	img := elementNode("img", nil, nil)
	result := walkNodes(t, img)

	assert.Equal(t, "", result)
}

func TestPlainTextWalker_PMLImage(t *testing.T) {
	img := elementNode("pml-img", []ast_domain.HTMLAttribute{attr("alt", "Banner")}, nil)
	result := walkNodes(t, img)
	assert.Equal(t, "Banner", result)
}

func TestPlainTextWalker_LineBreak(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		textNode("First line"),
		elementNode("br", nil, nil),
		textNode("Second line"),
	}
	result := walkNodes(t, nodes...)
	assert.Equal(t, "First line\nSecond line", result)
}

func TestPlainTextWalker_PMLLineBreak(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		textNode("First line"),
		elementNode("pml-br", nil, nil),
		textNode("Second line"),
	}
	result := walkNodes(t, nodes...)
	assert.Equal(t, "First line\nSecond line", result)
}

func TestPlainTextWalker_HorizontalRule(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		textNode("Before"),
		elementNode("hr", nil, nil),
		textNode("After"),
	}
	result := walkNodes(t, nodes...)

	lines := strings.Split(result, "\n")
	var hrLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "---") {
			hrLine = line
			break
		}
	}
	require.NotEmpty(t, hrLine, "Should contain horizontal rule")

	assert.Regexp(t, "^-{20,40}$", hrLine)
}

func TestPlainTextWalker_PMLContainer(t *testing.T) {
	container := elementNode("pml-container", nil,
		elementNode("pml-row", nil,
			elementNode("pml-col", nil,
				elementNode("pml-p", nil, textNode("Content")),
			),
		),
	)
	result := walkNodes(t, container)
	assert.Equal(t, "Content", result)
}

func TestPlainTextWalker_PlaintextHideDirective(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		textNode("Visible"),
		elementNode("div", []ast_domain.HTMLAttribute{attr("p-plaintext-hide", "")},
			textNode("Hidden content"),
		),
		textNode("Also visible"),
	}
	result := walkNodes(t, nodes...)

	assert.Contains(t, result, "Visible")
	assert.Contains(t, result, "Also visible")
	assert.NotContains(t, result, "Hidden")
}

func TestPlainTextWalker_NestedPlaintextHide(t *testing.T) {

	div := elementNode("div", []ast_domain.HTMLAttribute{attr("p-plaintext-hide", "")},
		elementNode("p", nil, textNode("This is hidden")),
		elementNode("strong", nil, textNode("This too")),
	)
	result := walkNodes(t, div)
	assert.Equal(t, "", result)
}

func TestPlainTextWalker_Comments(t *testing.T) {
	comment := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeComment,
		TextContent: "This is a comment",
	}
	result := walkNodes(t, comment)

	assert.Equal(t, "", result)
}

func TestPlainTextWalker_Fragment(t *testing.T) {
	fragment := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeFragment,
		Children: []*ast_domain.TemplateNode{
			textNode("Fragment content 1"),
			textNode("Fragment content 2"),
		},
	}
	result := walkNodes(t, fragment)

	assert.Contains(t, result, "Fragment content 1")
	assert.Contains(t, result, "Fragment content 2")
}

func TestPlainTextWalker_ComplexEmail(t *testing.T) {

	email := []*ast_domain.TemplateNode{
		elementNode("h1", nil, textNode("Welcome to Our Newsletter")),
		elementNode("p", nil, textNode("Thank you for subscribing!")),
		elementNode("h2", nil, textNode("This Month's Highlights")),
		elementNode("ul", nil,
			elementNode("li", nil, textNode("Feature 1")),
			elementNode("li", nil, textNode("Feature 2")),
		),
		elementNode("hr", nil, nil),
		elementNode("p", nil,
			textNode("Visit our "),
			elementNode("a", []ast_domain.HTMLAttribute{attr("href", "https://example.com")},
				textNode("website"),
			),
			textNode(" for more."),
		),
	}

	result := walkNodes(t, email...)

	assert.Contains(t, result, "Welcome to Our Newsletter")
	assert.Contains(t, result, "Thank you for subscribing!")
	assert.Contains(t, result, "This Month's Highlights")
	assert.Contains(t, result, "* Feature 1")
	assert.Contains(t, result, "* Feature 2")
	assert.Contains(t, result, "website (https://example.com)")

	lines := strings.Split(result, "\n")
	assert.Greater(t, len(lines), 5, "Should have multiple lines with proper spacing")
}

func TestPlainTextWalker_NewsletterWithTable(t *testing.T) {
	newsletter := []*ast_domain.TemplateNode{
		elementNode("h2", nil, textNode("Product Comparison")),
		elementNode("table", nil,
			elementNode("tr", nil,
				elementNode("th", nil, textNode("Product")),
				elementNode("th", nil, textNode("Price")),
				elementNode("th", nil, textNode("Rating")),
			),
			elementNode("tr", nil,
				elementNode("td", nil, textNode("Widget A")),
				elementNode("td", nil, textNode("$99")),
				elementNode("td", nil, textNode("4.5")),
			),
		),
		elementNode("p", nil,
			elementNode("a", []ast_domain.HTMLAttribute{attr("href", "https://shop.example.com")},
				textNode("Shop Now"),
			),
		),
	}

	result := walkNodes(t, newsletter...)

	assert.Contains(t, result, "Product Comparison")
	assert.Contains(t, result, "Product | Price | Rating")
	assert.Contains(t, result, "Widget A | $99 | 4.5")
	assert.Contains(t, result, "Shop Now (https://shop.example.com)")
}

func TestPlainTextWalker_EmptyAST(t *testing.T) {
	walker := newPlainTextWalker()
	ast := &ast_domain.TemplateAST{RootNodes: []*ast_domain.TemplateNode{}}
	result, err := walker.Walk(ast)
	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestPlainTextWalker_NilNode(t *testing.T) {
	walker := newPlainTextWalker()

	walker.walkNode(nil)
	result := walker.builder.String()
	assert.Equal(t, "", result)
}

func TestPlainTextWalker_MixedInlineAndBlock(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		elementNode("p", nil,
			textNode("Inline text "),
			elementNode("strong", nil, textNode("bold")),
			textNode(" more inline"),
		),
		elementNode("p", nil, textNode("Block paragraph")),
		textNode("After block"),
	}
	result := walkNodes(t, nodes...)

	assert.Contains(t, result, "Inline text** bold** more inline")

	assert.Contains(t, result, "Block paragraph")
	assert.Contains(t, result, "After block")
}

func TestPlainTextWalker_ConsecutiveBlocks(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		elementNode("p", nil, textNode("First")),
		elementNode("p", nil, textNode("Second")),
		elementNode("p", nil, textNode("Third")),
	}
	result := walkNodes(t, nodes...)

	assert.Equal(t, "First\n\nSecond\n\nThird", result)
}

func TestPlainTextWalker_EmptyElements(t *testing.T) {
	nodes := []*ast_domain.TemplateNode{
		elementNode("p", nil, nil),
		elementNode("div", nil, textNode("Content")),
		elementNode("p", nil, nil),
	}
	result := walkNodes(t, nodes...)

	assert.Contains(t, result, "Content")
}

func TestNewPlainTextWalker_Initialisation(t *testing.T) {
	walker := newPlainTextWalker()

	assert.NotNil(t, walker)
	assert.True(t, walker.isNewLine, "Should start with isNewLine=true")
	assert.Empty(t, walker.linkHref, "Link stack should be empty")
	assert.Empty(t, walker.listCounters, "List counter stack should be empty")
	assert.Equal(t, 0, walker.listDepth, "List depth should be 0")
}

func BenchmarkPlainTextWalker_SimpleEmail(b *testing.B) {
	nodes := []*ast_domain.TemplateNode{
		elementNode("h1", nil, textNode("Welcome")),
		elementNode("p", nil, textNode("This is a test email.")),
		elementNode("p", nil,
			textNode("Click "),
			elementNode("a", []ast_domain.HTMLAttribute{attr("href", "https://example.com")},
				textNode("here"),
			),
		),
	}

	ast := &ast_domain.TemplateAST{RootNodes: nodes}

	b.ResetTimer()
	for b.Loop() {
		walker := newPlainTextWalker()
		_, _ = walker.Walk(ast)
	}
}

func BenchmarkPlainTextWalker_ComplexEmail(b *testing.B) {

	listItems := make([]*ast_domain.TemplateNode, 10)
	for i := range 10 {
		listItems[i] = elementNode("li", nil, textNode("List item"))
	}

	nodes := []*ast_domain.TemplateNode{
		elementNode("h1", nil, textNode("Newsletter")),
		elementNode("p", nil, textNode("Welcome to our newsletter!")),
		elementNode("h2", nil, textNode("Features")),
		elementNode("ul", nil, listItems...),
		elementNode("table", nil,
			elementNode("tr", nil,
				elementNode("th", nil, textNode("Name")),
				elementNode("th", nil, textNode("Value")),
			),
			elementNode("tr", nil,
				elementNode("td", nil, textNode("Feature A")),
				elementNode("td", nil, textNode("$99")),
			),
		),
	}

	ast := &ast_domain.TemplateAST{RootNodes: nodes}

	b.ResetTimer()
	for b.Loop() {
		walker := newPlainTextWalker()
		_, _ = walker.Walk(ast)
	}
}

func TestPlainTextWalker_HandleTextWithRichText(t *testing.T) {

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeText,
		RichText: []ast_domain.TextPart{
			{IsLiteral: true, Literal: "Hello "},
			{IsLiteral: false, Literal: "{{ .Name }}"},
			{IsLiteral: true, Literal: " World"},
		},
	}

	result := walkNodes(t, node)
	assert.Equal(t, "Hello World", result)
}

func TestPlainTextWalker_HandleTextWithEmptyRichText(t *testing.T) {

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeText,
		RichText: []ast_domain.TextPart{
			{IsLiteral: false, Literal: "{{ .Dynamic }}"},
		},
	}

	result := walkNodes(t, node)
	assert.Empty(t, result)
}

func TestPlainTextWalker_WriteListMarkerMisplacedLi(t *testing.T) {

	li := elementNode("li", nil, textNode("Orphan item"))

	div := elementNode("div", nil, li)
	result := walkNodes(t, div)

	assert.Contains(t, result, "* ")
	assert.Contains(t, result, "Orphan item")
}

func TestPlainTextWalker_CalculateUnderlineLengthShortText(t *testing.T) {

	h1 := elementNode("h1", nil, textNode("Hi"))
	result := walkNodes(t, h1)

	lines := strings.Split(result, "\n")
	assert.GreaterOrEqual(t, len(lines), 2)

	underline := strings.TrimSpace(lines[1])
	assert.GreaterOrEqual(t, len(underline), 10)
}

func TestPlainTextWalker_CalculateUnderlineLengthLongText(t *testing.T) {

	longText := strings.Repeat("A", 100)
	h1 := elementNode("h1", nil, textNode(longText))
	result := walkNodes(t, h1)

	lines := strings.Split(result, "\n")
	assert.GreaterOrEqual(t, len(lines), 2)

	underline := strings.TrimSpace(lines[1])
	assert.LessOrEqual(t, len(underline), 72)
}

func TestPlainTextWalker_HandleImageWithPlaintextAlt(t *testing.T) {

	img := elementNode("img", []ast_domain.HTMLAttribute{
		attr("alt", "Standard alt"),
		attr("p-plaintext-alt", "Plain text alt"),
	})

	result := walkNodes(t, img)
	assert.Equal(t, "Plain text alt", result)
	assert.NotContains(t, result, "Standard alt")
}

func TestPlainTextWalker_HandleImageWithOnlyAlt(t *testing.T) {
	img := elementNode("img", []ast_domain.HTMLAttribute{
		attr("alt", "Image description"),
	})

	result := walkNodes(t, img)
	assert.Equal(t, "Image description", result)
}

func TestPlainTextWalker_HandleImageWithEmptyAlt(t *testing.T) {
	img := elementNode("img", []ast_domain.HTMLAttribute{
		attr("alt", ""),
	})

	result := walkNodes(t, img)
	assert.Empty(t, result)
}

func TestPlainTextWalker_HandleImageWithNoAlt(t *testing.T) {
	img := elementNode("img", []ast_domain.HTMLAttribute{
		attr("src", "image.png"),
	})

	result := walkNodes(t, img)
	assert.Empty(t, result)
}

func TestPlainTextWalker_NestedOrderedLists(t *testing.T) {

	innerItems := []*ast_domain.TemplateNode{
		elementNode("li", nil, textNode("Inner 1")),
		elementNode("li", nil, textNode("Inner 2")),
	}
	innerList := elementNode("ol", nil, innerItems...)

	outerItems := []*ast_domain.TemplateNode{
		elementNode("li", nil, textNode("Outer 1")),
		elementNode("li", nil, innerList),
		elementNode("li", nil, textNode("Outer 2")),
	}
	outerList := elementNode("ol", nil, outerItems...)

	result := walkNodes(t, outerList)

	assert.Contains(t, result, "1. Outer 1")
	assert.Contains(t, result, "1. Inner 1")
	assert.Contains(t, result, "2. Inner 2")

	assert.Contains(t, result, "3. Outer 2")
}

func TestPlainTextWalker_DeepNestedLists(t *testing.T) {

	level3 := elementNode("ul", nil, elementNode("li", nil, textNode("Deep")))
	level2 := elementNode("ul", nil, elementNode("li", nil, level3))
	level1 := elementNode("ul", nil, elementNode("li", nil, level2))

	result := walkNodes(t, level1)

	assert.Contains(t, result, "* ")
	assert.Contains(t, result, "Deep")
}

func TestPlainTextWalker_HandleTextSpaceHandling(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		textNode("First"),
		textNode("Second"),
		textNode("Third"),
	}

	result := walkNodes(t, nodes...)

	assert.Equal(t, "First Second Third", result)
}

func TestPlainTextWalker_HandleTextNoDoubleSpaces(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		textNode("Word "),
		textNode(" Another"),
	}

	result := walkNodes(t, nodes...)

	assert.NotContains(t, result, "  ")
	assert.Equal(t, "Word Another", result)
}

func TestPlainTextWalker_HandleImageWithPrecedingInlineText(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		textNode("Check out this"),
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "cool image")}, nil),
		textNode("for more info"),
	}

	result := walkNodes(t, nodes...)

	assert.Equal(t, "Check out this cool image for more info", result)
	assert.NotContains(t, result, "thiscool")
}

func TestPlainTextWalker_HandleImageInlineWithTrailingSpace(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		textNode("See "),
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "the logo")}, nil),
	}

	result := walkNodes(t, nodes...)

	assert.Equal(t, "See the logo", result)
	assert.NotContains(t, result, "  ")
}

func TestPlainTextWalker_H1VeryLongText(t *testing.T) {

	longText := strings.Repeat("A", 100)
	h1 := elementNode("h1", nil, textNode(longText))
	result := walkNodes(t, h1)

	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	underline := strings.TrimSpace(lines[1])

	assert.True(t, len(underline) >= 10, "Underline should be at least min length")
	assert.Regexp(t, "^=+$", underline)
}

func TestPlainTextWalker_H2VeryLongText(t *testing.T) {

	longText := strings.Repeat("B", 100)
	h2 := elementNode("h2", nil, textNode(longText))
	result := walkNodes(t, h2)

	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	underline := strings.TrimSpace(lines[1])

	assert.True(t, len(underline) >= 10, "Underline should be at least min length")
	assert.Regexp(t, "^-+$", underline)
}

func TestPlainTextWalker_H1ShortText(t *testing.T) {

	h1 := elementNode("h1", nil, textNode("Hi"))
	result := walkNodes(t, h1)

	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	underline := strings.TrimSpace(lines[1])

	assert.Equal(t, 10, len(underline), "Underline should be exactly minUnderlineLength (10)")
}

func TestPlainTextWalker_H1MediumText(t *testing.T) {

	h1 := elementNode("h1", nil, textNode("Medium Title"))
	result := walkNodes(t, h1)

	lines := strings.Split(result, "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	underline := strings.TrimSpace(lines[1])

	assert.True(t, len(underline) >= 10, "Underline should be at least min length")
	assert.Regexp(t, "^=+$", underline)
}

func TestPlainTextWalker_WriteLinkSuffixEmptyStack(t *testing.T) {

	walker := newPlainTextWalker()
	walker.builder.WriteString("Some text")
	walker.writeLinkSuffix()

	assert.Equal(t, "Some text", walker.builder.String())
}

func TestPlainTextWalker_LinkWithEmptyHref(t *testing.T) {

	link := elementNode("a", []ast_domain.HTMLAttribute{attr("href", "")},
		textNode("Placeholder link"),
	)
	result := walkNodes(t, link)

	assert.Equal(t, "Placeholder link", result)
	assert.NotContains(t, result, "()")
}

func TestPlainTextWalker_ButtonWithFragmentHref(t *testing.T) {

	button := elementNode("pml-button", []ast_domain.HTMLAttribute{attr("href", "#top")},
		textNode("Back to Top"),
	)
	result := walkNodes(t, button)

	assert.Equal(t, "Back to Top", result)
	assert.NotContains(t, result, "#top")
}

func TestPlainTextWalker_NestedLinksWithImages(t *testing.T) {

	link := elementNode("a", []ast_domain.HTMLAttribute{attr("href", "https://example.com")},
		textNode("Click the"),
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "icon")}, nil),
		textNode("here"),
	)
	result := walkNodes(t, link)

	assert.Contains(t, result, "Click the")
	assert.Contains(t, result, "icon")
	assert.Contains(t, result, "here")
	assert.Contains(t, result, "(https://example.com)")
}

func TestPlainTextWalker_MultipleConsecutiveImages(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "First")}, nil),
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "Second")}, nil),
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "Third")}, nil),
	}

	result := walkNodes(t, nodes...)

	assert.Equal(t, "First Second Third", result)
}

func TestPlainTextWalker_ImageAfterNewLine(t *testing.T) {

	nodes := []*ast_domain.TemplateNode{
		elementNode("p", nil, textNode("Paragraph")),
		elementNode("img", []ast_domain.HTMLAttribute{attr("alt", "Image")}, nil),
	}

	result := walkNodes(t, nodes...)

	assert.Contains(t, result, "Paragraph")
	assert.Contains(t, result, "Image")
}

func TestPlainTextWalker_EnsureNewBlockEmptyBuilder(t *testing.T) {

	walker := newPlainTextWalker()
	walker.ensureNewBlock()

	assert.Equal(t, "", walker.builder.String())
	assert.True(t, walker.isNewLine)
}

func TestPlainTextWalker_EnsureNewLineAlreadyNewLine(t *testing.T) {

	walker := newPlainTextWalker()
	walker.isNewLine = true
	walker.ensureNewLine()

	assert.Equal(t, "", walker.builder.String())
}

func TestPlainTextWalker_TableCellAtStartOfRow(t *testing.T) {

	table := elementNode("table", nil,
		elementNode("tr", nil,
			elementNode("td", nil, textNode("First")),
		),
	)
	result := walkNodes(t, table)

	assert.Equal(t, "First", result)
	assert.NotContains(t, result, "|")
}

func TestPlainTextWalker_ListCounterIncrement(t *testing.T) {

	items := make([]*ast_domain.TemplateNode, 15)
	for i := range 15 {
		items[i] = elementNode("li", nil, textNode("Item"))
	}
	ol := elementNode("ol", nil, items...)

	result := walkNodes(t, ol)

	assert.Contains(t, result, "1. Item")
	assert.Contains(t, result, "10. Item")
	assert.Contains(t, result, "15. Item")
}
