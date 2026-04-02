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

package generator_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestCollectAssetRefs(t *testing.T) {
	t.Parallel()

	t.Run("nil AST returns nil", func(t *testing.T) {
		t.Parallel()

		refs := CollectAssetRefs(nil, "")
		assert.Nil(t, refs)
	})

	t.Run("AST with no piko:svg elements returns empty", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		assert.Empty(t, refs)
	})

	t.Run("single piko:svg with src collected", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:svg",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "icons/star.svg"},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		require.Len(t, refs, 1)
		assert.Equal(t, "svg", refs[0].Kind)
		assert.Equal(t, "icons/star.svg", refs[0].Path)
	})

	t.Run("case-insensitive tag and attribute matching", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "PIKO:SVG",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "SRC", Value: "icons/arrow.svg"},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		require.Len(t, refs, 1)
		assert.Equal(t, "icons/arrow.svg", refs[0].Path)
	})

	t.Run("missing src attribute not collected", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:svg",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "class", Value: "icon"},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		assert.Empty(t, refs)
	})

	t.Run("empty src attribute not collected", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:svg",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: ""},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		assert.Empty(t, refs)
	})

	t.Run("duplicate refs deduplicated", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:svg",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "icons/star.svg"},
					},
				},
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:svg",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "icons/star.svg"},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		assert.Len(t, refs, 1)
	})

	t.Run("nested piko:svg found by Walk", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "piko:svg",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "src", Value: "icons/nested.svg"},
							},
						},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		require.Len(t, refs, 1)
		assert.Equal(t, "icons/nested.svg", refs[0].Path)
	})

	t.Run("text nodes ignored", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType:    ast_domain.NodeText,
					TextContent: "piko:svg",
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		assert.Empty(t, refs)
	})

	t.Run("path cleaning applied", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "piko:svg",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "src", Value: "./icons/../icons/star.svg"},
					},
				},
			},
		}
		refs := CollectAssetRefs(ast, "")
		require.Len(t, refs, 1)
		assert.Equal(t, "icons/star.svg", refs[0].Path)
	})
}
