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

package ast_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateNode_OwnText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node *TemplateNode
		name string
		want string
	}{
		{
			name: "nil node yields empty",
			node: nil,
			want: "",
		},
		{
			name: "no text at all",
			node: &TemplateNode{NodeType: NodeText},
			want: "",
		},
		{
			name: "plain static text",
			node: &TemplateNode{NodeType: NodeText, TextContent: "hello"},
			want: "hello",
		},
		{
			name: "comment body lives in TextContent",
			node: &TemplateNode{NodeType: NodeComment, TextContent: " note "},
			want: " note ",
		},
		{
			name: "rich text contributes literals only",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{literalPart("Count: "), expressionPart("state.n"), literalPart("!")},
			},
			want: "Count: !",
		},
		{
			name: "rich text of only expressions yields empty",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{expressionPart("state.n")},
			},
			want: "",
		},
		{
			name: "writer appended without escaping is returned verbatim",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("a & b") }),
			},
			want: "a & b",
		},
		{
			name: "writer appended with escaping stays escaped",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendEscapeString("a & b") }),
			},
			want: "a &amp; b",
		},
		{
			name: "writer holding one empty part wins over TextContent",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContent:       "shadowed",
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendEscapeString("") }),
			},
			want: "",
		},
		{
			name: "writer with no parts does not shadow TextContent",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContent:       "kept",
				TextContentWriter: &DirectWriter{},
			},
			want: "kept",
		},
		{
			name: "rich text wins over both other carriers",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContent:       "static",
				RichText:          []TextPart{literalPart("rich")},
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("written") }),
			},
			want: "rich",
		},
		{
			name: "writer wins over TextContent",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContent:       "static",
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("written") }),
			},
			want: "written",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.node.OwnText())
		})
	}
}

func TestTemplateNode_OwnRawText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node *TemplateNode
		name string
		want string
	}{
		{
			name: "nil node yields empty",
			node: nil,
			want: "",
		},
		{
			name: "plain static text",
			node: &TemplateNode{NodeType: NodeText, TextContent: "hello"},
			want: "hello",
		},
		{
			name: "interpolation is rebuilt with its braces",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{literalPart("Count: "), expressionPart("state.n"), literalPart("!")},
			},
			want: "Count: {{ state.n }}!",
		},
		{
			name: "empty expression still round-trips as braces",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{expressionPart("")},
			},
			want: "{{  }}",
		},
		{
			name: "evaluated text is never presented as authored source",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("Alice") }),
			},
			want: "",
		},
		{
			name: "writer stays invisible even alongside static text",
			node: &TemplateNode{
				NodeType:          NodeText,
				TextContent:       "authored",
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("Alice") }),
			},
			want: "authored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.node.OwnRawText())
		})
	}
}

func TestTemplateNode_IsWhitespaceOnlyText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node *TemplateNode
		name string
		want bool
	}{
		{
			name: "nil node is not whitespace text",
			node: nil,
			want: false,
		},
		{
			name: "element node is not whitespace text",
			node: &TemplateNode{NodeType: NodeElement, TagName: "div"},
			want: false,
		},
		{
			name: "comment node is not whitespace text",
			node: &TemplateNode{NodeType: NodeComment, TextContent: "   "},
			want: false,
		},
		{
			name: "blank static text",
			node: &TemplateNode{NodeType: NodeText, TextContent: " \n\t "},
			want: true,
		},
		{
			name: "non-blank static text",
			node: &TemplateNode{NodeType: NodeText, TextContent: " a "},
			want: false,
		},
		{
			name: "interpolation is never whitespace",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{expressionPart("state.value")},
			},
			want: false,
		},
		{
			name: "rich text of blank literals is whitespace",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{literalPart(" "), literalPart("\n")},
			},
			want: true,
		},
		{
			name: "rich text with a visible literal is not whitespace",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{literalPart(" "), literalPart("x")},
			},
			want: false,
		},
		{
			name: "a blank literal beside an expression is not whitespace",
			node: &TemplateNode{
				NodeType: NodeText,
				RichText: []TextPart{literalPart(" "), expressionPart("state.value")},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.node.IsWhitespaceOnlyText())
		})
	}
}

func TestTemplateNode_TextCarrier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		node *TemplateNode
		name string
		want textCarrier
	}{
		{name: "nil", node: nil, want: carrierNone},
		{name: "empty", node: &TemplateNode{}, want: carrierNone},
		{name: "static only", node: &TemplateNode{TextContent: "a"}, want: carrierStatic},
		{
			name: "writer only",
			node: &TemplateNode{TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("a") })},
			want: carrierWriter,
		},
		{name: "rich only", node: &TemplateNode{RichText: []TextPart{literalPart("a")}}, want: carrierRich},
		{
			name: "writer beats static",
			node: &TemplateNode{
				TextContent:       "a",
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("b") }),
			},
			want: carrierWriter,
		},
		{
			name: "rich beats writer",
			node: &TemplateNode{
				RichText:          []TextPart{literalPart("a")},
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("b") }),
			},
			want: carrierRich,
		},
		{
			name: "rich beats everything",
			node: &TemplateNode{
				TextContent:       "a",
				RichText:          []TextPart{literalPart("b")},
				TextContentWriter: writerWith(func(dw *DirectWriter) { dw.AppendString("c") }),
			},
			want: carrierRich,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.node.textCarrier())
		})
	}
}
