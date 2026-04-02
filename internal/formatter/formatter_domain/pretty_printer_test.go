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

package formatter_domain

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func Test_newPrettyPrinter(t *testing.T) {
	t.Run("with nil options uses defaults", func(t *testing.T) {
		printer := newPrettyPrinter(nil)

		require.NotNil(t, printer)
		assert.NotNil(t, printer.options)
		assert.Equal(t, 2, printer.options.IndentSize)
		assert.False(t, printer.options.PreserveEmptyLines)
		assert.True(t, printer.options.SortAttributes)
	})

	t.Run("with custom options", func(t *testing.T) {
		opts := &FormatOptions{
			IndentSize:         4,
			PreserveEmptyLines: true,
			SortAttributes:     false,
		}
		printer := newPrettyPrinter(opts)

		require.NotNil(t, printer)
		assert.Equal(t, opts, printer.options)
		assert.Equal(t, 4, printer.options.IndentSize)
		assert.True(t, printer.options.PreserveEmptyLines)
		assert.False(t, printer.options.SortAttributes)
	})
}

func TestPrettyPrinter_String(t *testing.T) {
	printer := newPrettyPrinter(nil)
	printer.write("Hello, World!")

	result := printer.String()
	assert.Equal(t, "Hello, World!", result)
}

func TestPrettyPrinter_isSelfClosing(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{name: "img is self-closing", tagName: "img", expected: true},
		{name: "br is self-closing", tagName: "br", expected: true},
		{name: "hr is self-closing", tagName: "hr", expected: true},
		{name: "input is self-closing", tagName: "input", expected: true},
		{name: "link is self-closing", tagName: "link", expected: true},
		{name: "meta is self-closing", tagName: "meta", expected: true},
		{name: "area is self-closing", tagName: "area", expected: true},
		{name: "base is self-closing", tagName: "base", expected: true},
		{name: "col is self-closing", tagName: "col", expected: true},
		{name: "embed is self-closing", tagName: "embed", expected: true},
		{name: "param is self-closing", tagName: "param", expected: true},
		{name: "source is self-closing", tagName: "source", expected: true},
		{name: "track is self-closing", tagName: "track", expected: true},
		{name: "wbr is self-closing", tagName: "wbr", expected: true},
		{name: "div is not self-closing", tagName: "div", expected: false},
		{name: "span is not self-closing", tagName: "span", expected: false},
		{name: "p is not self-closing", tagName: "p", expected: false},
		{name: "IMG uppercase is self-closing", tagName: "IMG", expected: true},
		{name: "DIV uppercase is not self-closing", tagName: "DIV", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ast_domain.TemplateNode{TagName: tt.tagName}
			result := isSelfClosing(node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrettyPrinter_isBlockElement(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{name: "div is block", tagName: "div", expected: true},
		{name: "section is block", tagName: "section", expected: true},
		{name: "article is block", tagName: "article", expected: true},
		{name: "header is block", tagName: "header", expected: true},
		{name: "footer is block", tagName: "footer", expected: true},
		{name: "nav is block", tagName: "nav", expected: true},
		{name: "main is block", tagName: "main", expected: true},
		{name: "aside is block", tagName: "aside", expected: true},
		{name: "p is block", tagName: "p", expected: true},
		{name: "h1 is block", tagName: "h1", expected: true},
		{name: "h2 is block", tagName: "h2", expected: true},
		{name: "ul is block", tagName: "ul", expected: true},
		{name: "ol is block", tagName: "ol", expected: true},
		{name: "li is block", tagName: "li", expected: true},
		{name: "table is block", tagName: "table", expected: true},
		{name: "form is block", tagName: "form", expected: true},
		{name: "span is not block", tagName: "span", expected: false},
		{name: "a is not block", tagName: "a", expected: false},
		{name: "strong is not block", tagName: "strong", expected: false},
		{name: "DIV uppercase is block", tagName: "DIV", expected: true},
		{name: "SPAN uppercase is not block", tagName: "SPAN", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBlockElement(tt.tagName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrettyPrinter_formatText(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple text",
			content:  "Hello, World!",
			expected: "Hello, World!\n",
		},
		{
			name:     "text with leading/trailing spaces",
			content:  "  Hello  ",
			expected: "Hello\n",
		},
		{
			name:     "empty text",
			content:  "",
			expected: "",
		},
		{
			name:     "whitespace only",
			content:  "   \n  \t  ",
			expected: "",
		},
		{
			name:     "text with internal spaces",
			content:  "Hello    World",
			expected: "Hello World\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := newPrettyPrinter(nil)
			node := &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: tt.content,
			}

			printer.formatText(node)
			result := printer.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrettyPrinter_formatComment(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple comment",
			content:  "This is a comment",
			expected: "<!-- This is a comment -->\n",
		},
		{
			name:     "comment with extra spaces",
			content:  "  Comment with spaces  ",
			expected: "<!-- Comment with spaces -->\n",
		},
		{
			name:     "empty comment",
			content:  "",
			expected: "<!--  -->\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			printer := newPrettyPrinter(nil)
			node := &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeComment,
				TextContent: tt.content,
			}

			printer.formatComment(node)
			result := printer.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrettyPrinter_collectAttributes(t *testing.T) {
	t.Run("static attributes only", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
		}

		attrs := collectAttributesInOrder(node)
		assert.Len(t, attrs, 2)
		assert.Contains(t, attrs, `class="container"`)
		assert.Contains(t, attrs, `id="main"`)
	})

	t.Run("boolean attributes", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "disabled", Value: ""},
				{Name: "readonly", Value: ""},
			},
		}

		attrs := collectAttributesSorted(node)
		assert.Len(t, attrs, 2)
		assert.Contains(t, attrs, "disabled")
		assert.Contains(t, attrs, "readonly")
	})

	t.Run("dynamic attributes", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", RawExpression: "state.Title"},
				{Name: "href", RawExpression: "state.URL"},
			},
		}

		attrs := collectAttributesSorted(node)
		assert.Len(t, attrs, 2)
		assert.Contains(t, attrs, `:title="state.Title"`)
		assert.Contains(t, attrs, `:href="state.URL"`)
	})

	t.Run("attributes are sorted when enabled", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "z-attr", Value: "last"},
				{Name: "a-attr", Value: "first"},
				{Name: "m-attr", Value: "middle"},
			},
		}

		attrs := collectAttributesSorted(node)
		assert.Len(t, attrs, 3)

		assert.Equal(t, `a-attr="first"`, attrs[0])
		assert.Equal(t, `m-attr="middle"`, attrs[1])
		assert.Equal(t, `z-attr="last"`, attrs[2])
	})

	t.Run("attributes not sorted when disabled", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "z-attr", Value: "last"},
				{Name: "a-attr", Value: "first"},
			},
		}

		attrs := collectAttributesInOrder(node)
		assert.Len(t, attrs, 2)

		assert.Equal(t, `z-attr="last"`, attrs[0])
		assert.Equal(t, `a-attr="first"`, attrs[1])
	})
}

func TestPrettyPrinter_formatDirectives(t *testing.T) {
	t.Run("structural directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirIf: &ast_domain.Directive{
				RawExpression: "state.IsVisible",
			},
			DirFor: &ast_domain.Directive{
				RawExpression: "item in items",
			},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, `p-if="state.IsVisible"`)
		assert.Contains(t, directives, `p-for="item in items"`)
	})

	t.Run("p-else directive", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirElse: &ast_domain.Directive{},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, "p-else")
	})

	t.Run("content directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirText: &ast_domain.Directive{
				RawExpression: "state.Message",
			},
			DirHTML: &ast_domain.Directive{
				RawExpression: "state.HTML",
			},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, `p-text="state.Message"`)
		assert.Contains(t, directives, `p-html="state.HTML"`)
	})

	t.Run("style and class bindings", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirClass: &ast_domain.Directive{
				RawExpression: "{'active': state.IsActive}",
			},
			DirStyle: &ast_domain.Directive{
				RawExpression: "{'color': state.Color}",
			},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, `p-class="{'active': state.IsActive}"`)
		assert.Contains(t, directives, `p-style="{'color': state.Color}"`)
	})

	t.Run("binds directive", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"disabled": {RawExpression: "state.IsDisabled"},
				"value":    {RawExpression: "state.Value"},
			},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, `p-bind:disabled="state.IsDisabled"`)
		assert.Contains(t, directives, `p-bind:value="state.Value"`)
	})

	t.Run("event handlers", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			OnEvents: map[string][]ast_domain.Directive{
				"click": {
					{RawExpression: "handleClick()"},
				},
				"submit": {
					{RawExpression: "handleSubmit()"},
				},
			},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, `p-on:click="handleClick()"`)
		assert.Contains(t, directives, `p-on:submit="handleSubmit()"`)
	})

	t.Run("other directives", func(t *testing.T) {
		node := &ast_domain.TemplateNode{
			DirModel: &ast_domain.Directive{
				RawExpression: "state.Username",
			},
			DirRef: &ast_domain.Directive{
				RawExpression: "myRef",
			},
			DirKey: &ast_domain.Directive{
				RawExpression: "item.id",
			},
		}

		directives := formatDirectives(node)
		assert.Contains(t, directives, `p-model="state.Username"`)
		assert.Contains(t, directives, `p-ref="myRef"`)
		assert.Contains(t, directives, `p-key="item.id"`)
	})
}

func TestPrettyPrinter_formatElement(t *testing.T) {
	t.Run("simple element with no children", func(t *testing.T) {
		printer := newPrettyPrinter(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: nil,
		}

		printer.formatElement(node)
		result := printer.String()
		assert.Equal(t, "<div></div>\n", result)
	})

	t.Run("self-closing element", func(t *testing.T) {
		printer := newPrettyPrinter(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "img",
			Children: nil,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "src", Value: "/image.png"},
			},
		}

		printer.formatElement(node)
		result := printer.String()
		assert.Equal(t, `<img src="/image.png" />`+"\n", result)
	})

	t.Run("element with attributes", func(t *testing.T) {
		printer := newPrettyPrinter(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: nil,
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
		}

		printer.formatElement(node)
		result := printer.String()
		assert.Contains(t, result, "<div")
		assert.Contains(t, result, `class="container"`)
		assert.Contains(t, result, `id="main"`)
	})

	t.Run("element with children increments indentation", func(t *testing.T) {
		printer := newPrettyPrinter(nil)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "p"},
				{NodeType: ast_domain.NodeElement, TagName: "span"},
			},
		}

		initialLevel := printer.indentationLevel
		printer.formatElement(node)

		assert.Equal(t, initialLevel+1, printer.indentationLevel)
	})
}

func TestPrettyPrinter_Enter_Exit(t *testing.T) {
	t.Run("Enter returns self for continuation", func(t *testing.T) {
		printer := newPrettyPrinter(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
		}

		visitor, err := printer.Enter(context.Background(), node)
		assert.NoError(t, err)
		assert.Equal(t, printer, visitor)
	})

	t.Run("Enter with nil node", func(t *testing.T) {
		printer := newPrettyPrinter(nil)

		visitor, err := printer.Enter(context.Background(), nil)
		assert.NoError(t, err)
		assert.Equal(t, printer, visitor)
	})

	t.Run("Exit with nil node", func(t *testing.T) {
		printer := newPrettyPrinter(nil)

		err := printer.Exit(context.Background(), nil)
		assert.NoError(t, err)
	})

	t.Run("Enter/Exit for fragment is transparent", func(t *testing.T) {
		printer := newPrettyPrinter(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeFragment,
		}

		visitor, err := printer.Enter(context.Background(), node)
		assert.NoError(t, err)
		assert.Equal(t, printer, visitor)

		err = printer.Exit(context.Background(), node)
		assert.NoError(t, err)

		assert.Equal(t, "", printer.String())
	})

	t.Run("Exit writes closing tag for elements with children", func(t *testing.T) {
		printer := newPrettyPrinter(nil)
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText},
			},
		}

		_, _ = printer.Enter(context.Background(), node)

		_ = printer.Exit(context.Background(), node)

		result := printer.String()
		assert.Contains(t, result, "</div>")
	})

	t.Run("Exit decrements indentation before closing tag", func(t *testing.T) {
		printer := newPrettyPrinter(nil)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "p"},
				{NodeType: ast_domain.NodeElement, TagName: "span"},
			},
		}

		_, _ = printer.Enter(context.Background(), node)
		levelAfterEnter := printer.indentationLevel

		_ = printer.Exit(context.Background(), node)
		assert.Equal(t, levelAfterEnter-1, printer.indentationLevel)
	})
}

func TestPrettyPrinter_SiblingIndentationWithWrappedAttributes(t *testing.T) {

	t.Run("siblings with wrapped attributes should have consistent indentation", func(t *testing.T) {
		printer := newPrettyPrinter(&FormatOptions{
			IndentSize:          2,
			MaxLineLength:       40,
			AttributeWrapIndent: 1,
			SortAttributes:      true,
		})

		root := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "wrapper",
			Children: []*ast_domain.TemplateNode{
				{
					NodeType:        ast_domain.NodeElement,
					TagName:         "custom-element",
					PreferredFormat: ast_domain.FormatInline,
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "class", Value: "first-child-class-name"},
								{Name: "id", Value: "first-child"},
								{Name: "data-long-attribute", Value: "some-value"},
							},

							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "h3",
									Children: []*ast_domain.TemplateNode{
										{
											NodeType:    ast_domain.NodeText,
											TextContent: "Title",
										},
									},
								},
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "p",
									Children: []*ast_domain.TemplateNode{
										{
											NodeType:    ast_domain.NodeText,
											TextContent: "Content paragraph",
										},
									},
								},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "class", Value: "second-child-class-name"},
								{Name: "id", Value: "second-child"},
								{Name: "data-long-attribute", Value: "another-value"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType:    ast_domain.NodeText,
									TextContent: "Second",
								},
							},
						},
					},
				},
			},
		}

		err := ast_domain.WalkWithVisitor(context.Background(), printer, root)
		require.NoError(t, err)

		result := printer.String()

		lines := strings.Split(result, "\n")

		var closingDivLine, secondOpeningDivLine int
		closingDivLine = -1
		secondOpeningDivLine = -1

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "</div>") && closingDivLine == -1 {
				closingDivLine = i
			}

			if trimmed == "<div" || strings.HasPrefix(trimmed, "<div ") {

				if closingDivLine != -1 && secondOpeningDivLine == -1 {
					secondOpeningDivLine = i
				}
			}
		}

		require.NotEqual(t, -1, closingDivLine, "Should find closing </div> line")
		require.NotEqual(t, -1, secondOpeningDivLine, "Should find second <div line after closing tag")

		getLeadingWhitespace := func(line string) string {
			for i, character := range line {
				if character != ' ' && character != '\t' {
					return line[:i]
				}
			}
			return line
		}

		closingDivWhitespace := getLeadingWhitespace(lines[closingDivLine])
		secondDivWhitespace := getLeadingWhitespace(lines[secondOpeningDivLine])

		nextLineIndex := secondOpeningDivLine + 1
		var attributeLineWhitespace string
		if nextLineIndex < len(lines) {
			attributeLineWhitespace = getLeadingWhitespace(lines[nextLineIndex])
		}

		if closingDivWhitespace != "" && secondDivWhitespace == "" && attributeLineWhitespace != "" {
			t.Errorf("Sibling indentation bug: second sibling <div has no indent but should have.\n"+
				"Closing </div> indent: %q\n"+
				"Second <div indent: %q (should not be empty)\n"+
				"Attribute indent: %q\n"+
				"Full output:\n%s",
				closingDivWhitespace, secondDivWhitespace, attributeLineWhitespace, result)
		}

		assert.Equal(t, closingDivWhitespace, secondDivWhitespace,
			"Second sibling should have same indentation as previous closing tag")
	})
}

func TestPrettyPrinter_SiblingSelfClosingElements(t *testing.T) {

	t.Run("sibling self-closing elements should have consistent indentation", func(t *testing.T) {
		printer := newPrettyPrinter(&FormatOptions{
			IndentSize:          2,
			MaxLineLength:       40,
			AttributeWrapIndent: 1,
			SortAttributes:      true,
		})

		root := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "item-row"},
			},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "hidden"},
						{Name: "name", Value: "items[0].id"},
						{Name: "value", Value: "uuid-123"},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "text"},
						{Name: "name", Value: "items[0].name"},
						{Name: "placeholder", Value: "Item name"},
						{Name: "class", Value: "item-name"},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "input",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "text"},
						{Name: "name", Value: "items[0].value"},
						{Name: "placeholder", Value: "Item value"},
						{Name: "class", Value: "item-value"},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "button",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "type", Value: "button"},
						{Name: "class", Value: "remove-btn"},
					},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Remove",
						},
					},
				},
			},
		}

		err := ast_domain.WalkWithVisitor(context.Background(), printer, root)
		require.NoError(t, err)

		result := printer.String()
		t.Logf("Formatted output:\n%s", result)

		lines := strings.Split(result, "\n")
		var inputLines []int
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "<input") {
				inputLines = append(inputLines, i)
			}
		}

		getLeadingWhitespace := func(line string) string {
			for i, character := range line {
				if character != ' ' && character != '\t' {
					return line[:i]
				}
			}
			return line
		}

		if len(inputLines) >= 2 {
			for i := 1; i < len(inputLines); i++ {
				lineIndex := inputLines[i]
				whitespace := getLeadingWhitespace(lines[lineIndex])

				if lineIndex > 0 {
					previousLine := lines[lineIndex-1]

					trimmedPrevious := strings.TrimSpace(previousLine)
					if strings.HasSuffix(trimmedPrevious, "/>") || strings.HasSuffix(trimmedPrevious, ">") {

						if whitespace == "" {
							t.Errorf("Input element on line %d has no indentation but should have.\n"+
								"Previous line: %q\n"+
								"Current line: %q\n"+
								"Full output:\n%s",
								lineIndex, previousLine, lines[lineIndex], result)
						}
					}
				}
			}
		}
	})
}

func TestPrettyPrinter_SiblingAfterWhitespaceSensitive(t *testing.T) {

	t.Run("sibling after textarea should have correct indentation", func(t *testing.T) {

		printer := newPrettyPrinter(DefaultFormatOptions())

		root := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "form",
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "fieldset",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "legend",
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "Personal Info"},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "label",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "for", Value: "fname"},
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "First name:"},
							},
						},
						{
							NodeType:   ast_domain.NodeElement,
							TagName:    "input",
							Attributes: []ast_domain.HTMLAttribute{{Name: "type", Value: "text"}, {Name: "id", Value: "fname"}},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "select",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "id", Value: "country"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType:   ast_domain.NodeElement,
									TagName:    "option",
									Attributes: []ast_domain.HTMLAttribute{{Name: "value", Value: "uk"}},
									Children:   []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText, TextContent: "UK"}},
								},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "textarea",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "id", Value: "bio"},
								{Name: "rows", Value: "4"},
								{Name: "cols", Value: "50"},
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "Biography"},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "button",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "type", Value: "submit"},
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "Submit"},
							},
						},
					},
				},
			},
		}

		err := ast_domain.WalkWithVisitor(context.Background(), printer, root)
		require.NoError(t, err)

		result := printer.String()
		t.Logf("Formatted output:\n%s", result)

		lines := strings.Split(result, "\n")
		var textareaIndent, buttonIndent string

		for _, line := range lines {
			if strings.Contains(line, "<textarea") {
				for i, character := range line {
					if character != ' ' {
						textareaIndent = line[:i]
						break
					}
				}
			}
			if strings.Contains(line, "<button") {
				for i, character := range line {
					if character != ' ' {
						buttonIndent = line[:i]
						break
					}
				}
			}
		}

		assert.Equal(t, textareaIndent, buttonIndent,
			"Button should have same indentation as textarea.\n"+
				"Textarea indent: %q (%d spaces)\n"+
				"Button indent: %q (%d spaces)\n"+
				"Full output:\n%s",
			textareaIndent, len(textareaIndent),
			buttonIndent, len(buttonIndent),
			result)
	})
}

func TestIsWhitespaceSensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tagName string
		want    bool
	}{
		{name: "pre is sensitive", tagName: "pre", want: true},
		{name: "code is sensitive", tagName: "code", want: true},
		{name: "textarea is sensitive", tagName: "textarea", want: true},
		{name: "div is not sensitive", tagName: "div", want: false},
		{name: "span is not sensitive", tagName: "span", want: false},
		{name: "PRE uppercase is sensitive", tagName: "PRE", want: true},
		{name: "empty tag is not sensitive", tagName: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isWhitespaceSensitive(tc.tagName)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestShouldAddEmptyLineBefore(t *testing.T) {
	t.Parallel()

	t.Run("nil node returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
		p.lastWasBlock = true
		assert.False(t, p.shouldAddEmptyLineBefore(nil))
	})

	t.Run("non-element node returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
		p.lastWasBlock = true
		node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeText}
		assert.False(t, p.shouldAddEmptyLineBefore(node))
	})

	t.Run("PreserveEmptyLines disabled returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: false})
		p.lastWasBlock = true
		node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "header"}
		assert.False(t, p.shouldAddEmptyLineBefore(node))
	})

	t.Run("lastWasBlock false returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
		p.lastWasBlock = false
		node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "header"}
		assert.False(t, p.shouldAddEmptyLineBefore(node))
	})

	t.Run("directive node with p-if returns true", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
		p.lastWasBlock = true
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirIf:    &ast_domain.Directive{RawExpression: "x"},
		}
		assert.True(t, p.shouldAddEmptyLineBefore(node))
	})

	t.Run("directive node with p-for returns true", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
		p.lastWasBlock = true
		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DirFor:   &ast_domain.Directive{RawExpression: "x in y"},
		}
		assert.True(t, p.shouldAddEmptyLineBefore(node))
	})

	t.Run("major block elements return true", func(t *testing.T) {
		t.Parallel()
		majorBlocks := []string{"header", "main", "footer", "section", "article", "form"}
		for _, tag := range majorBlocks {
			p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
			p.lastWasBlock = true
			node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: tag}
			assert.True(t, p.shouldAddEmptyLineBefore(node), "expected true for %s", tag)
		}
	})

	t.Run("non-major element returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(&FormatOptions{PreserveEmptyLines: true})
		p.lastWasBlock = true
		node := &ast_domain.TemplateNode{NodeType: ast_domain.NodeElement, TagName: "div"}
		assert.False(t, p.shouldAddEmptyLineBefore(node))
	})
}

func TestContainsShadowRootChild(t *testing.T) {
	t.Parallel()

	t.Run("no children returns false", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{Children: nil}
		assert.False(t, containsShadowRootChild(node))
	})

	t.Run("regular children returns false", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeText, TextContent: "hello"},
			},
		}
		assert.False(t, containsShadowRootChild(node))
	})

	t.Run("shadow root child returns true", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "template",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "shadowrootmode", Value: "open"},
					},
				},
			},
		}
		assert.True(t, containsShadowRootChild(node))
	})
}

func TestShouldWrapAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tagName       string
		attrs         []string
		maxLineLength int
		currentIndent int
		want          bool
	}{
		{
			name:          "no max line length disables wrapping",
			tagName:       "div",
			attrs:         []string{`class="container"`, `id="main"`},
			maxLineLength: 0,
			currentIndent: 0,
			want:          false,
		},
		{
			name:          "no attributes disables wrapping",
			tagName:       "div",
			attrs:         []string{},
			maxLineLength: 80,
			currentIndent: 0,
			want:          false,
		},
		{
			name:          "short line does not wrap",
			tagName:       "div",
			attrs:         []string{`id="x"`},
			maxLineLength: 80,
			currentIndent: 0,
			want:          false,
		},
		{
			name:          "long line wraps",
			tagName:       "div",
			attrs:         []string{`class="very-long-class-name"`, `id="main-container"`, `data-value="something"`},
			maxLineLength: 40,
			currentIndent: 10,
			want:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldWrapAttributes(tc.tagName, tc.attrs, tc.maxLineLength, tc.currentIndent)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsParentAlsoInline(t *testing.T) {
	t.Parallel()

	t.Run("empty stack returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		assert.False(t, p.isParentAlsoInline())
	})

	t.Run("single entry returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingModeStack = append(p.formattingModeStack, true)
		assert.False(t, p.isParentAlsoInline())
	})

	t.Run("parent inline returns true", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingModeStack = append(p.formattingModeStack, true, false)
		assert.True(t, p.isParentAlsoInline())
	})

	t.Run("parent block returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingModeStack = append(p.formattingModeStack, false, true)
		assert.False(t, p.isParentAlsoInline())
	})
}

func TestPopFormattingMode(t *testing.T) {
	t.Parallel()

	t.Run("empty stack returns false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		assert.False(t, p.popFormattingMode())
	})

	t.Run("pops inline mode", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingModeStack = append(p.formattingModeStack, true)
		assert.True(t, p.popFormattingMode())
		assert.Empty(t, p.formattingModeStack)
	})

	t.Run("pops block mode", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingModeStack = append(p.formattingModeStack, false)
		assert.False(t, p.popFormattingMode())
		assert.Empty(t, p.formattingModeStack)
	})
}

func TestRestoreParentFormattingContext(t *testing.T) {
	t.Parallel()

	t.Run("empty stack sets formattingInline to false", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingInline = true
		p.restoreParentFormattingContext()
		assert.False(t, p.formattingInline)
	})

	t.Run("restores from stack", func(t *testing.T) {
		t.Parallel()
		p := newPrettyPrinter(nil)
		p.formattingModeStack = append(p.formattingModeStack, true)
		p.restoreParentFormattingContext()
		assert.True(t, p.formattingInline)
	})
}

func TestPrettyPrinter_Integration(t *testing.T) {
	t.Run("complete element with nested children", func(t *testing.T) {
		printer := newPrettyPrinter(&FormatOptions{
			IndentSize:     2,
			SortAttributes: true,
		})

		root := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			Children: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "p",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType:    ast_domain.NodeText,
							TextContent: "Hello",
						},
					},
				},
			},
		}

		err := ast_domain.WalkWithVisitor(context.Background(), printer, root)
		require.NoError(t, err)

		result := printer.String()
		assert.Contains(t, result, `<div class="container">`)
		assert.Contains(t, result, "<p>")
		assert.Contains(t, result, "Hello")
		assert.Contains(t, result, "</p>")
		assert.Contains(t, result, "</div>")
	})
}
