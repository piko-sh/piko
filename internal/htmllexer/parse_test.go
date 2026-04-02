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

package htmllexer

import "testing"

func TestAttributes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "double quoted attribute",
			input: `<div class="foo">`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "class", attrVal: `"foo"`},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "single quoted attribute",
			input: `<div class='foo'>`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "class", attrVal: `'foo'`},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unquoted attribute",
			input: "<div class=foo>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "class", attrVal: "foo"},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "boolean attribute",
			input: "<input checked>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "input"},
				{tokenType: AttributeToken, text: "checked", attrValIsNil: true},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "multiple attributes",
			input: `<div id="main" class="wrapper" data-value="42">`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "id", attrVal: `"main"`},
				{tokenType: AttributeToken, text: "class", attrVal: `"wrapper"`},
				{tokenType: AttributeToken, text: "data-value", attrVal: `"42"`},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "attribute with spaces around equals",
			input: `<div class = "foo">`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "class", attrVal: `"foo"`},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty attribute value",
			input: `<div class="">`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "class", attrVal: `""`},
				{tokenType: StartTagCloseToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "attribute on self-closing tag",
			input: `<input type="text"/>`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "input"},
				{tokenType: AttributeToken, text: "type", attrVal: `"text"`},
				{tokenType: StartTagVoidToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "mixed boolean and valued attributes",
			input: `<input disabled type="text" required/>`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "input"},
				{tokenType: AttributeToken, text: "disabled", attrValIsNil: true},
				{tokenType: AttributeToken, text: "type", attrVal: `"text"`},
				{tokenType: AttributeToken, text: "required", attrValIsNil: true},
				{tokenType: StartTagVoidToken},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestComments(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "simple comment",
			input: "<!-- hello -->",
			expected: []expectedToken{
				{tokenType: CommentToken, text: " hello "},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty comment",
			input: "<!---->",
			expected: []expectedToken{
				{tokenType: CommentToken, text: ""},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "comment with bang close",
			input: "<!-- hello --!>",
			expected: []expectedToken{
				{tokenType: CommentToken, text: " hello "},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "comment between elements",
			input: "<p>a</p><!-- comment --><p>b</p>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "p"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "a"},
				{tokenType: EndTagToken, text: "p"},
				{tokenType: CommentToken, text: " comment "},
				{tokenType: StartTagToken, text: "p"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "b"},
				{tokenType: EndTagToken, text: "p"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "bogus comment with question mark",
			input: "<?xml version='1.0'?>",
			expected: []expectedToken{
				{tokenType: CommentToken},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestRawTextElements(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "script with content",
			input: "<script>var x = 1;</script>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "var x = 1;"},
				{tokenType: EndTagToken, text: "script"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "style with content",
			input: "<style>.foo { color: red; }</style>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "style"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: ".foo { color: red; }"},
				{tokenType: EndTagToken, text: "style"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "textarea with content",
			input: "<textarea>some text</textarea>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "textarea"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "some text"},
				{tokenType: EndTagToken, text: "textarea"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "script with angle brackets in content",
			input: "<script>if (a < b && c > d) {}</script>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "if (a < b && c > d) {}"},
				{tokenType: EndTagToken, text: "script"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "script with attributes",
			input: `<script type="module">import x from './y';</script>`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: AttributeToken, text: "type", attrVal: `"module"`},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "import x from './y';"},
				{tokenType: EndTagToken, text: "script"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "case-insensitive raw text closing tag",
			input: "<script>x</SCRIPT>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "x"},
				{tokenType: EndTagToken, text: "SCRIPT"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "title element",
			input: "<title>Page <Title></title>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "title"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "Page <Title>"},
				{tokenType: EndTagToken, text: "title"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty textarea must not emit text token",
			input: "<textarea></textarea>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "textarea"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "textarea"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "textarea with whitespace preserves content",
			input: "<textarea> </textarea>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "textarea"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: " "},
				{tokenType: EndTagToken, text: "textarea"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty style must not emit text token",
			input: "<style></style>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "style"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "style"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty script must not emit text token",
			input: "<script></script>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "script"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "empty textarea with attributes must not emit text token",
			input: `<textarea id="msg" placeholder="Type here"></textarea>`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "textarea"},
				{tokenType: AttributeToken, text: "id", attrVal: `"msg"`},
				{tokenType: AttributeToken, text: "placeholder", attrVal: `"Type here"`},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "textarea"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestScriptCommentNesting(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "script with html comment",
			input: "<script><!--comment--></script>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "<!--comment-->"},
				{tokenType: EndTagToken, text: "script"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "script with nested script in comment",
			input: "<script><!--<script></script>--></script>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "<!--<script></script>-->"},
				{tokenType: EndTagToken, text: "script"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestForeignContent(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "simple svg",
			input: `<svg><circle r="5"/></svg>`,
			expected: []expectedToken{
				{tokenType: SVGToken, text: `<svg><circle r="5"/></svg>`},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "simple math",
			input: "<math><mi>x</mi></math>",
			expected: []expectedToken{
				{tokenType: MathToken, text: "<math><mi>x</mi></math>"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "svg with closing tag in attribute",
			input: `<svg data-x="</svg>"></svg>`,
			expected: []expectedToken{
				{tokenType: SVGToken, text: `<svg data-x="</svg>"></svg>`},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "svg between other content",
			input: `<div><svg><rect/></svg></div>`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: StartTagCloseToken},
				{tokenType: SVGToken, text: `<svg><rect/></svg>`},
				{tokenType: EndTagToken, text: "div"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "case-insensitive svg closing",
			input: "<svg></SVG>",
			expected: []expectedToken{
				{tokenType: SVGToken, text: "<svg></SVG>"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "svg with single-quoted closing tag in attribute",
			input: "<svg data-x='</svg>'></svg>",
			expected: []expectedToken{
				{tokenType: SVGToken, text: "<svg data-x='</svg>'></svg>"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "math with single-quoted closing tag in attribute",
			input: "<math data-x='</math>'></math>",
			expected: []expectedToken{
				{tokenType: MathToken, text: "<math data-x='</math>'></math>"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "svg with mixed quotes in attributes",
			input: `<svg class="foo" data-x='</svg>'></svg>`,
			expected: []expectedToken{
				{tokenType: SVGToken, text: `<svg class="foo" data-x='</svg>'></svg>`},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestDoctype(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "html5 doctype is silently consumed",
			input: "<!DOCTYPE html><html></html>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "html"},
				{tokenType: StartTagCloseToken},
				{tokenType: EndTagToken, text: "html"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "lowercase doctype is silently consumed",
			input: "<!doctype html>",
			expected: []expectedToken{
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestUnclosedElements(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "unclosed comment at eof",
			input: "<!-- hello",
			expected: []expectedToken{
				{tokenType: CommentToken, text: " hello"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unclosed cdata at eof",
			input: "<![CDATA[some content",
			expected: []expectedToken{
				{tokenType: TextToken, text: "some content"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unclosed style at eof",
			input: "<style>.foo { color: red; }",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "style"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: ".foo { color: red; }"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unclosed script at eof",
			input: "<script>var x = 1;",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "var x = 1;"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unclosed script comment at eof",
			input: "<script><!-- comment",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "script"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "<!-- comment"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "attribute value truncated at eof after equals",
			input: `<div class=`,
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "div"},
				{tokenType: AttributeToken, text: "class"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "unclosed svg at eof",
			input: `<svg><rect/>`,
			expected: []expectedToken{
				{tokenType: SVGToken, text: `<svg><rect/>`},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestTruncatedMarkupDeclarations(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "truncated doctype keyword",
			input: "<!DOC>",
			expected: []expectedToken{
				{tokenType: CommentToken},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "markup declaration without match",
			input: "<!ENTITY foo>",
			expected: []expectedToken{
				{tokenType: CommentToken},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestAdditionalRawTextElements(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "xmp element",
			input: "<xmp><b>bold</b></xmp>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "xmp"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "<b>bold</b>"},
				{tokenType: EndTagToken, text: "xmp"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "iframe element",
			input: "<iframe>frame content</iframe>",
			expected: []expectedToken{
				{tokenType: StartTagToken, text: "iframe"},
				{tokenType: StartTagCloseToken},
				{tokenType: TextToken, text: "frame content"},
				{tokenType: EndTagToken, text: "iframe"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}

func TestCDATA(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []expectedToken
	}{
		{
			name:  "simple cdata",
			input: "<![CDATA[some content]]>",
			expected: []expectedToken{
				{tokenType: TextToken, text: "some content"},
				{tokenType: ErrorToken},
			},
		},
		{
			name:  "cdata with angle brackets",
			input: "<![CDATA[<div>not a tag</div>]]>",
			expected: []expectedToken{
				{tokenType: TextToken, text: "<div>not a tag</div>"},
				{tokenType: ErrorToken},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assertTokenSequence(t, tc.input, tc.expected)
		})
	}
}
