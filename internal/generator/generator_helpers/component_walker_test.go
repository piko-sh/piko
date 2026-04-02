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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestCollectComponentComponents(t *testing.T) {
	t.Parallel()

	t.Run("nil AST returns nil", func(t *testing.T) {
		t.Parallel()

		result := CollectComponentComponents(nil, "")
		assert.Nil(t, result)
	})

	t.Run("no custom elements returns empty", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeElement, TagName: "span"},
			},
		}
		result := CollectComponentComponents(ast, "")
		assert.Empty(t, result)
	})

	t.Run("hyphenated tag collected", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "my-component"},
			},
		}
		result := CollectComponentComponents(ast, "")
		require.Len(t, result, 1)
		assert.Equal(t, "my-component", result[0])
	})

	t.Run("piko: prefixed non-meta tag collected", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "piko:header"},
			},
		}
		result := CollectComponentComponents(ast, "")
		require.Len(t, result, 1)
		assert.Equal(t, "piko:header", result[0])
	})

	t.Run("meta tags excluded", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "piko:svg"},
				{NodeType: ast_domain.NodeElement, TagName: "piko:a"},
				{NodeType: ast_domain.NodeElement, TagName: "piko:video"},
			},
		}
		result := CollectComponentComponents(ast, "")
		assert.Empty(t, result)
	})

	t.Run("duplicates deduplicated", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "my-component"},
				{NodeType: ast_domain.NodeElement, TagName: "my-component"},
				{NodeType: ast_domain.NodeElement, TagName: "my-component"},
			},
		}
		result := CollectComponentComponents(ast, "")
		assert.Len(t, result, 1)
	})

	t.Run("non-element nodes skipped", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TagName: "my-component"},
			},
		}
		result := CollectComponentComponents(ast, "")
		assert.Empty(t, result)
	})

	t.Run("mixed standard custom and meta", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeElement, TagName: "my-card"},
				{NodeType: ast_domain.NodeElement, TagName: "piko:svg"},
				{NodeType: ast_domain.NodeElement, TagName: "user-avatar"},
				{NodeType: ast_domain.NodeElement, TagName: "span"},
				{NodeType: ast_domain.NodeElement, TagName: "piko:nav"},
			},
		}
		result := CollectComponentComponents(ast, "")
		sort.Strings(result)
		require.Len(t, result, 3)
		assert.Equal(t, []string{"my-card", "piko:nav", "user-avatar"}, result)
	})

	t.Run("nested custom elements found", func(t *testing.T) {
		t.Parallel()

		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeElement, TagName: "inner-component"},
					},
				},
			},
		}
		result := CollectComponentComponents(ast, "")
		require.Len(t, result, 1)
		assert.Equal(t, "inner-component", result[0])
	})
}
