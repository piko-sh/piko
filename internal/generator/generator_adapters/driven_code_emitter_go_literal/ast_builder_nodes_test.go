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

package driven_code_emitter_go_literal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNodeContainsPikoContent(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "nil_node_returns_false",
			node: nil,
			want: false,
		},
		{
			name: "piko_content_tag_returns_true",
			node: &ast_domain.TemplateNode{
				TagName: "piko:content",
			},
			want: true,
		},
		{
			name: "regular_div_returns_false",
			node: &ast_domain.TemplateNode{
				TagName: "div",
			},
			want: false,
		},
		{
			name: "child_with_piko_content_returns_true",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{TagName: "span"},
					{TagName: "piko:content"},
				},
			},
			want: true,
		},
		{
			name: "deeply_nested_piko_content_returns_true",
			node: &ast_domain.TemplateNode{
				TagName: "section",
				Children: []*ast_domain.TemplateNode{
					{
						TagName: "article",
						Children: []*ast_domain.TemplateNode{
							{
								TagName: "div",
								Children: []*ast_domain.TemplateNode{
									{TagName: "piko:content"},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "no_piko_content_in_deep_tree_returns_false",
			node: &ast_domain.TemplateNode{
				TagName: "section",
				Children: []*ast_domain.TemplateNode{
					{
						TagName: "article",
						Children: []*ast_domain.TemplateNode{
							{TagName: "div"},
							{TagName: "p"},
						},
					},
					{TagName: "aside"},
				},
			},
			want: false,
		},
		{
			name: "empty_children_returns_false",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := builder.nodeContainsPikoContent(tc.node)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNodeContainsRichText(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "nil_node_returns_false",
			node: nil,
			want: false,
		},
		{
			name: "node_with_rich_text_returns_true",
			node: &ast_domain.TemplateNode{
				TagName: "span",
				RichText: []ast_domain.TextPart{
					{IsLiteral: true, Literal: "Hello "},
				},
			},
			want: true,
		},
		{
			name: "node_without_rich_text_returns_false",
			node: &ast_domain.TemplateNode{
				TagName:  "div",
				RichText: []ast_domain.TextPart{},
			},
			want: false,
		},
		{
			name: "child_with_rich_text_returns_true",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Children: []*ast_domain.TemplateNode{
					{
						TagName: "span",
						RichText: []ast_domain.TextPart{
							{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "x"}},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "deeply_nested_rich_text_returns_true",
			node: &ast_domain.TemplateNode{
				TagName: "section",
				Children: []*ast_domain.TemplateNode{
					{
						TagName: "article",
						Children: []*ast_domain.TemplateNode{
							{
								TagName: "p",
								RichText: []ast_domain.TextPart{
									{IsLiteral: true, Literal: "text"},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "no_rich_text_in_deep_tree_returns_false",
			node: &ast_domain.TemplateNode{
				TagName: "section",
				Children: []*ast_domain.TemplateNode{
					{
						TagName:  "article",
						RichText: []ast_domain.TextPart{},
						Children: []*ast_domain.TemplateNode{
							{TagName: "div", RichText: []ast_domain.TextPart{}},
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := builder.nodeContainsRichText(tc.node)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateNodeForEmission(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	testCases := []struct {
		originalNode *ast_domain.TemplateNode
		preparedNode *ast_domain.TemplateNode
		name         string
		wantDiag     bool
	}{
		{
			name: "same_node_returns_nil",
			originalNode: &ast_domain.TemplateNode{
				TagName: "div",
			},
			preparedNode: nil,
			wantDiag:     false,
		},
		{
			name: "different_nodes_with_valid_main_component_returns_nil",
			originalNode: &ast_domain.TemplateNode{
				TagName: "div",
			},
			preparedNode: &ast_domain.TemplateNode{
				TagName: "span",
			},
			wantDiag: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			preparedNode := tc.preparedNode
			if tc.name == "same_node_returns_nil" {
				preparedNode = tc.originalNode
			}

			diagnostic := builder.validateNodeForEmission(tc.originalNode, preparedNode)

			if tc.wantDiag {
				assert.NotNil(t, diagnostic)
			} else {
				assert.Nil(t, diagnostic)
			}
		})
	}
}

func TestIsElseClauseNode(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	testCases := []struct {
		original *ast_domain.TemplateNode
		prepared *ast_domain.TemplateNode
		name     string
		want     bool
	}{
		{
			name:     "neither_has_else_returns_false",
			original: &ast_domain.TemplateNode{TagName: "div"},
			prepared: &ast_domain.TemplateNode{TagName: "div"},
			want:     false,
		},
		{
			name:     "original_has_else_returns_true",
			original: &ast_domain.TemplateNode{TagName: "div", DirElse: &ast_domain.Directive{}},
			prepared: &ast_domain.TemplateNode{TagName: "div"},
			want:     true,
		},
		{
			name:     "prepared_has_else_returns_true",
			original: &ast_domain.TemplateNode{TagName: "div"},
			prepared: &ast_domain.TemplateNode{TagName: "div", DirElse: &ast_domain.Directive{}},
			want:     true,
		},
		{
			name:     "original_has_else_if_returns_true",
			original: &ast_domain.TemplateNode{TagName: "div", DirElseIf: &ast_domain.Directive{}},
			prepared: &ast_domain.TemplateNode{TagName: "div"},
			want:     true,
		},
		{
			name:     "prepared_has_else_if_returns_true",
			original: &ast_domain.TemplateNode{TagName: "div"},
			prepared: &ast_domain.TemplateNode{TagName: "div", DirElseIf: &ast_domain.Directive{}},
			want:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := builder.isElseClauseNode(tc.original, tc.prepared)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFragmentHasDynamicFeatures(t *testing.T) {
	t.Parallel()

	builder := createTestAstBuilder()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "plain_node_is_not_dynamic",
			node: &ast_domain.TemplateNode{
				TagName: "div",
			},
			want: false,
		},
		{
			name: "node_with_partial_info_is_dynamic",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{},
				},
			},
			want: true,
		},
		{
			name: "node_with_dynamic_attributes_is_dynamic",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "class"},
				},
			},
			want: true,
		},
		{
			name: "node_with_nil_go_annotations_is_not_dynamic",
			node: &ast_domain.TemplateNode{
				TagName:       "div",
				GoAnnotations: nil,
			},
			want: false,
		},
		{
			name: "node_with_empty_dynamic_attributes_is_not_dynamic",
			node: &ast_domain.TemplateNode{
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := builder.fragmentHasDynamicFeatures(tc.node)

			assert.Equal(t, tc.want, got)
		})
	}
}
