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
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestElementClassification(t *testing.T) {
	tests := []struct {
		tagName         string
		description     string
		wantBlock       bool
		wantInline      bool
		wantWhitespace  bool
		wantSelfClosing bool
	}{
		{tagName: "div", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "div is block"},
		{tagName: "section", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "section is block"},
		{tagName: "article", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "article is block"},
		{tagName: "header", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "header is block"},
		{tagName: "footer", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "footer is block"},
		{tagName: "nav", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "nav is block"},
		{tagName: "main", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "main is block"},
		{tagName: "aside", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "aside is block"},
		{tagName: "form", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "form is block"},
		{tagName: "table", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "table is block"},
		{tagName: "ul", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "ul is block"},
		{tagName: "ol", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "ol is block"},
		{tagName: "li", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "li is block (but can format inline)"},
		{tagName: "dl", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "dl is block"},
		{tagName: "dt", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "dt is block (but can format inline)"},
		{tagName: "dd", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "dd is block (but can format inline)"},
		{tagName: "h1", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "h1 is block"},
		{tagName: "h2", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "h2 is block"},
		{tagName: "h3", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "h3 is block"},
		{tagName: "h4", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "h4 is block"},
		{tagName: "h5", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "h5 is block"},
		{tagName: "h6", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "h6 is block"},
		{tagName: "p", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "p is block"},
		{tagName: "blockquote", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "blockquote is block"},
		{tagName: "address", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "address is block"},
		{tagName: "span", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "span is inline"},
		{tagName: "a", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "a is inline"},
		{tagName: "strong", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "strong is inline"},
		{tagName: "em", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "em is inline"},
		{tagName: "b", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "b is inline"},
		{tagName: "i", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "i is inline"},
		{tagName: "u", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "u is inline"},
		{tagName: "s", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "s is inline"},
		{tagName: "abbr", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "abbr is inline"},
		{tagName: "cite", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "cite is inline"},
		{tagName: "kbd", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "kbd is inline"},
		{tagName: "mark", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "mark is inline"},
		{tagName: "q", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "q is inline"},
		{tagName: "samp", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "samp is inline"},
		{tagName: "small", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "small is inline"},
		{tagName: "sub", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "sub is inline"},
		{tagName: "sup", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "sup is inline"},
		{tagName: "time", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "time is inline"},
		{tagName: "var", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "var is inline"},
		{tagName: "pre", wantBlock: true, wantInline: false, wantWhitespace: true, wantSelfClosing: false, description: "pre is block and whitespace-sensitive"},
		{tagName: "code", wantBlock: false, wantInline: true, wantWhitespace: true, wantSelfClosing: false, description: "code is inline and whitespace-sensitive"},
		{tagName: "textarea", wantBlock: false, wantInline: false, wantWhitespace: true, wantSelfClosing: false, description: "textarea is whitespace-sensitive"},
		{tagName: "img", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "img is self-closing"},
		{tagName: "br", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: true, description: "br is inline and self-closing"},
		{tagName: "hr", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "hr is self-closing"},
		{tagName: "input", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "input is self-closing"},
		{tagName: "link", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "link is self-closing"},
		{tagName: "meta", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "meta is self-closing"},
		{tagName: "area", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "area is self-closing"},
		{tagName: "base", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "base is self-closing"},
		{tagName: "col", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "col is self-closing"},
		{tagName: "embed", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "embed is self-closing"},
		{tagName: "param", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "param is self-closing"},
		{tagName: "source", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "source is self-closing"},
		{tagName: "track", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "track is self-closing"},
		{tagName: "wbr", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: true, description: "wbr is inline and self-closing"},
		{tagName: "piko:slot", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "piko:slot is block"},
		{tagName: "slot", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "slot is block"},

		{tagName: "figure", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "figure is block"},
		{tagName: "figcaption", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "figcaption is block"},
		{tagName: "details", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "details is block"},
		{tagName: "dialog", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "dialog is block"},
		{tagName: "hgroup", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "hgroup is block"},
		{tagName: "search", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "search is block"},
		{tagName: "audio", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "audio is block"},
		{tagName: "video", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "video is block"},
		{tagName: "picture", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "picture is block"},
		{tagName: "canvas", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "canvas is block"},
		{tagName: "noscript", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "noscript is block"},
		{tagName: "style", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "style is block"},

		{tagName: "button", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "button is inline"},
		{tagName: "label", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "label is inline"},
		{tagName: "legend", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "legend is inline"},
		{tagName: "output", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "output is inline"},
		{tagName: "progress", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "progress is inline"},
		{tagName: "meter", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "meter is inline"},
		{tagName: "summary", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "summary is inline"},
		{tagName: "ruby", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "ruby is inline"},
		{tagName: "rt", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "rt is inline"},
		{tagName: "rp", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "rp is inline"},
		{tagName: "del", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "del is inline"},
		{tagName: "ins", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "ins is inline"},

		{tagName: "DIV", wantBlock: true, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "DIV uppercase is block"},
		{tagName: "SPAN", wantBlock: false, wantInline: true, wantWhitespace: false, wantSelfClosing: false, description: "SPAN uppercase is inline"},
		{tagName: "PRE", wantBlock: true, wantInline: false, wantWhitespace: true, wantSelfClosing: false, description: "PRE uppercase is whitespace-sensitive"},
		{tagName: "IMG", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: true, description: "IMG uppercase is self-closing"},
		{tagName: "custom-element", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "unknown tag defaults to none"},
		{tagName: "my-component", wantBlock: false, wantInline: false, wantWhitespace: false, wantSelfClosing: false, description: "custom component defaults to none"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			assert.Equal(t, tt.wantBlock, isBlockElement(tt.tagName),
				"isBlockElement(%s) should be %v", tt.tagName, tt.wantBlock)
			assert.Equal(t, tt.wantInline, isInlineElement(tt.tagName),
				"isInlineElement(%s) should be %v", tt.tagName, tt.wantInline)
			assert.Equal(t, tt.wantWhitespace, isWhitespaceSensitive(tt.tagName),
				"isWhitespaceSensitive(%s) should be %v", tt.tagName, tt.wantWhitespace)

			node := &ast_domain.TemplateNode{TagName: tt.tagName}
			assert.Equal(t, tt.wantSelfClosing, isSelfClosing(node),
				"isSelfClosing(%s) should be %v", tt.tagName, tt.wantSelfClosing)
		})
	}
}
