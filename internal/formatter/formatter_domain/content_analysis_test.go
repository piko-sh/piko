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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestAnalyseElementContent(t *testing.T) {
	tests := []struct {
		node            *ast_domain.TemplateNode
		name            string
		wantChildCount  int
		wantTextLength  int
		wantTotalLength int
		wantHasText     bool
		wantHasElement  bool
		wantHasBlock    bool
		wantAllInline   bool
	}{
		{
			name: "empty element",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{},
			},
			wantChildCount:  0,
			wantHasText:     false,
			wantHasElement:  false,
			wantHasBlock:    false,
			wantAllInline:   true,
			wantTextLength:  0,
			wantTotalLength: 0,
		},
		{
			name: "single text child",
			node: &ast_domain.TemplateNode{
				TagName: "p",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Hello, World!",
					},
				},
			},
			wantChildCount:  1,
			wantHasText:     true,
			wantHasElement:  false,
			wantHasBlock:    false,
			wantAllInline:   true,
			wantTextLength:  13,
			wantTotalLength: 13,
		},
		{
			name: "single inline element child",
			node: &ast_domain.TemplateNode{
				TagName: "p",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "strong",
					},
				},
			},
			wantChildCount:  1,
			wantHasText:     false,
			wantHasElement:  true,
			wantHasBlock:    false,
			wantAllInline:   true,
			wantTextLength:  0,
			wantTotalLength: 17,
		},
		{
			name: "single block element child",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
					},
				},
			},
			wantChildCount:  1,
			wantHasText:     false,
			wantHasElement:  true,
			wantHasBlock:    true,
			wantAllInline:   false,
			wantTextLength:  0,
			wantTotalLength: 7,
		},
		{
			name: "mixed text and inline element",
			node: &ast_domain.TemplateNode{
				TagName: "p",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Text with ",
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "strong",
					},
					{
						NodeType:    ast_domain.NodeText,
						TextContent: " more text",
					},
				},
			},
			wantChildCount:  3,
			wantHasText:     true,
			wantHasElement:  true,
			wantHasBlock:    false,
			wantAllInline:   true,
			wantTextLength:  20,
			wantTotalLength: 31,
		},
		{
			name: "text with interpolation",
			node: &ast_domain.TemplateNode{
				TagName: "p",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeText,
						RichText: []ast_domain.TextPart{
							{IsLiteral: true, Literal: "Hello "},
							{IsLiteral: false, RawExpression: "user.Name"},
							{IsLiteral: true, Literal: "!"},
						},
					},
				},
			},
			wantChildCount:  1,
			wantHasText:     true,
			wantHasElement:  false,
			wantHasBlock:    false,
			wantAllInline:   true,
			wantTextLength:  16,
			wantTotalLength: 22,
		},
		{
			name: "multiple block children",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "p"},
					{NodeType: ast_domain.NodeElement, TagName: "div"},
					{NodeType: ast_domain.NodeElement, TagName: "section"},
				},
			},
			wantChildCount:  3,
			wantHasText:     false,
			wantHasElement:  true,
			wantHasBlock:    true,
			wantAllInline:   false,
			wantTextLength:  0,
			wantTotalLength: 30,
		},
		{
			name: "comment child",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeComment,
						TextContent: "A comment",
					},
				},
			},
			wantChildCount:  1,
			wantHasText:     false,
			wantHasElement:  false,
			wantHasBlock:    false,
			wantAllInline:   true,
			wantTextLength:  0,
			wantTotalLength: 19,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyseElementContent(tt.node)

			assert.Equal(t, tt.wantChildCount, result.childCount, "childCount")
			assert.Equal(t, tt.wantHasText, result.hasTextChildren, "hasTextChildren")
			assert.Equal(t, tt.wantHasElement, result.hasElementChildren, "hasElementChildren")
			assert.Equal(t, tt.wantHasBlock, result.hasBlockChildren, "hasBlockChildren")
			assert.Equal(t, tt.wantAllInline, result.allChildrenInline, "allChildrenInline")

			assert.InDelta(t, tt.wantTextLength, result.textLength, 5, "textLength")
			assert.InDelta(t, tt.wantTotalLength, result.totalContentLength, 10,
				"totalContentLength should be approximately correct")
		})
	}
}

func TestAnalyseElementContent_EdgeCases(t *testing.T) {
	t.Run("deeply nested structure", func(t *testing.T) {
		innermost := &ast_domain.TemplateNode{
			NodeType:    ast_domain.NodeText,
			TextContent: "Deep",
		}
		nested := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "span",
			Children: []*ast_domain.TemplateNode{innermost},
		}
		parent := &ast_domain.TemplateNode{
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{nested},
		}

		result := analyseElementContent(parent)

		assert.Equal(t, 1, result.childCount)
		assert.True(t, result.hasElementChildren)
		assert.False(t, result.hasTextChildren)
		assert.True(t, result.allChildrenInline)
	})

	t.Run("nil node", func(t *testing.T) {
		result := analyseElementContent(nil)
		assert.Equal(t, 0, result.childCount)
	})

	t.Run("very long text", func(t *testing.T) {
		longText := strings.Repeat("A", 500)
		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: longText},
			},
		}

		result := analyseElementContent(node)

		assert.Equal(t, 500, result.textLength)
		assert.Equal(t, 500, result.totalContentLength)
	})

	t.Run("many interpolations", func(t *testing.T) {
		richText := []ast_domain.TextPart{
			{IsLiteral: true, Literal: "A"},
			{IsLiteral: false, RawExpression: "x"},
			{IsLiteral: true, Literal: "B"},
			{IsLiteral: false, RawExpression: "y"},
			{IsLiteral: true, Literal: "C"},
		}
		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, RichText: richText},
			},
		}

		result := analyseElementContent(node)

		assert.Equal(t, 17, result.textLength)
	})
}
