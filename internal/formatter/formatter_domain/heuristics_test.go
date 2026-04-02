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

func TestIsSingleShortTextChild(t *testing.T) {
	tests := []struct {
		name     string
		analysis contentAnalysis
		want     bool
	}{
		{
			name: "single text child with 30 chars",
			analysis: contentAnalysis{
				childCount:         1,
				hasTextChildren:    true,
				hasElementChildren: false,
				textLength:         30,
			},
			want: true,
		},
		{
			name: "single text child exactly at threshold (60 chars)",
			analysis: contentAnalysis{
				childCount:         1,
				hasTextChildren:    true,
				hasElementChildren: false,
				textLength:         maxInlineTextLength,
			},
			want: true,
		},
		{
			name: "single text child just over threshold (61 chars)",
			analysis: contentAnalysis{
				childCount:         1,
				hasTextChildren:    true,
				hasElementChildren: false,
				textLength:         maxInlineTextLength + 1,
			},
			want: false,
		},
		{
			name: "single text child with 100 chars",
			analysis: contentAnalysis{
				childCount:         1,
				hasTextChildren:    true,
				hasElementChildren: false,
				textLength:         100,
			},
			want: false,
		},
		{
			name: "multiple children",
			analysis: contentAnalysis{
				childCount:         2,
				hasTextChildren:    true,
				hasElementChildren: false,
				textLength:         30,
			},
			want: false,
		},
		{
			name: "no text children",
			analysis: contentAnalysis{
				childCount:         1,
				hasTextChildren:    false,
				hasElementChildren: true,
				textLength:         0,
			},
			want: false,
		},
		{
			name: "mixed text and element",
			analysis: contentAnalysis{
				childCount:         2,
				hasTextChildren:    true,
				hasElementChildren: true,
				textLength:         30,
			},
			want: false,
		},
		{
			name: "empty element",
			analysis: contentAnalysis{
				childCount:         0,
				hasTextChildren:    false,
				hasElementChildren: false,
				textLength:         0,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSingleShortTextChild(tt.analysis)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsSimpleListItem(t *testing.T) {
	tests := []struct {
		node     *ast_domain.TemplateNode
		name     string
		analysis contentAnalysis
		want     bool
	}{
		{
			name: "li with short content",
			node: &ast_domain.TemplateNode{TagName: "li"},
			analysis: contentAnalysis{
				totalContentLength: 50,
				hasBlockChildren:   false,
			},
			want: true,
		},
		{
			name: "li exactly at threshold (80 chars)",
			node: &ast_domain.TemplateNode{TagName: "li"},
			analysis: contentAnalysis{
				totalContentLength: maxListItemLength,
				hasBlockChildren:   false,
			},
			want: true,
		},
		{
			name: "li just over threshold (81 chars)",
			node: &ast_domain.TemplateNode{TagName: "li"},
			analysis: contentAnalysis{
				totalContentLength: maxListItemLength + 1,
				hasBlockChildren:   false,
			},
			want: false,
		},
		{
			name: "li with block children",
			node: &ast_domain.TemplateNode{TagName: "li"},
			analysis: contentAnalysis{
				totalContentLength: 50,
				hasBlockChildren:   true,
			},
			want: false,
		},
		{
			name: "dt with short content",
			node: &ast_domain.TemplateNode{TagName: "dt"},
			analysis: contentAnalysis{
				totalContentLength: 50,
				hasBlockChildren:   false,
			},
			want: true,
		},
		{
			name: "dd with short content",
			node: &ast_domain.TemplateNode{TagName: "dd"},
			analysis: contentAnalysis{
				totalContentLength: 50,
				hasBlockChildren:   false,
			},
			want: true,
		},
		{
			name: "non-list-item with short content",
			node: &ast_domain.TemplateNode{TagName: "div"},
			analysis: contentAnalysis{
				totalContentLength: 50,
				hasBlockChildren:   false,
			},
			want: false,
		},
		{
			name: "li with zero content",
			node: &ast_domain.TemplateNode{TagName: "li"},
			analysis: contentAnalysis{
				totalContentLength: 0,
				hasBlockChildren:   false,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSimpleListItem(tt.node, tt.analysis)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsShortMixedInlineContent(t *testing.T) {
	tests := []struct {
		name     string
		analysis contentAnalysis
		want     bool
	}{
		{
			name: "short mixed inline content (50 chars)",
			analysis: contentAnalysis{
				allChildrenInline:  true,
				hasBlockChildren:   false,
				totalContentLength: 50,
			},
			want: true,
		},
		{
			name: "exactly at threshold (80 chars)",
			analysis: contentAnalysis{
				allChildrenInline:  true,
				hasBlockChildren:   false,
				totalContentLength: maxMixedContentLength,
			},
			want: true,
		},
		{
			name: "just over threshold (81 chars)",
			analysis: contentAnalysis{
				allChildrenInline:  true,
				hasBlockChildren:   false,
				totalContentLength: maxMixedContentLength + 1,
			},
			want: false,
		},
		{
			name: "has block children",
			analysis: contentAnalysis{
				allChildrenInline:  false,
				hasBlockChildren:   true,
				totalContentLength: 50,
			},
			want: false,
		},
		{
			name: "not all children inline",
			analysis: contentAnalysis{
				allChildrenInline:  false,
				hasBlockChildren:   false,
				totalContentLength: 50,
			},
			want: false,
		},
		{
			name: "zero content length",
			analysis: contentAnalysis{
				allChildrenInline:  true,
				hasBlockChildren:   false,
				totalContentLength: 0,
			},
			want: true,
		},
		{
			name: "very long content (200 chars)",
			analysis: contentAnalysis{
				allChildrenInline:  true,
				hasBlockChildren:   false,
				totalContentLength: 200,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isShortMixedInlineContent(tt.analysis)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestShouldFormatAsBlockDueToMultipleChildren(t *testing.T) {
	tests := []struct {
		node     *ast_domain.TemplateNode
		name     string
		analysis contentAnalysis
		want     bool
	}{
		{
			name: "block element with 2 element children",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "p"},
					{NodeType: ast_domain.NodeElement, TagName: "span"},
				},
			},
			analysis: contentAnalysis{
				childCount:         2,
				hasElementChildren: true,
			},
			want: true,
		},
		{
			name: "block element with 1 element child",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "p"},
				},
			},
			analysis: contentAnalysis{
				childCount:         1,
				hasElementChildren: true,
			},
			want: false,
		},
		{
			name: "inline element with 2 element children",
			node: &ast_domain.TemplateNode{
				TagName: "span",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "strong"},
					{NodeType: ast_domain.NodeElement, TagName: "em"},
				},
			},
			analysis: contentAnalysis{
				childCount:         2,
				hasElementChildren: true,
			},
			want: false,
		},
		{
			name: "block element with text children only",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeText},
					{NodeType: ast_domain.NodeText},
				},
			},
			analysis: contentAnalysis{
				childCount:         2,
				hasElementChildren: false,
			},
			want: false,
		},
		{
			name: "block element with 3 element children",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "p"},
					{NodeType: ast_domain.NodeElement, TagName: "p"},
					{NodeType: ast_domain.NodeElement, TagName: "p"},
				},
			},
			analysis: contentAnalysis{
				childCount:         3,
				hasElementChildren: true,
			},
			want: true,
		},
		{
			name: "block element with mixed children (1 element, 1 text)",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{NodeType: ast_domain.NodeElement, TagName: "p"},
					{NodeType: ast_domain.NodeText},
				},
			},
			analysis: contentAnalysis{
				childCount:         2,
				hasElementChildren: true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldFormatAsBlockDueToMultipleChildren(tt.node, tt.analysis)
			assert.Equal(t, tt.want, result)
		})
	}
}
