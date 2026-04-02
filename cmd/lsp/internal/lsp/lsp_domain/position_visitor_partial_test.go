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

package lsp_domain

import (
	"context"
	"testing"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestIsPartialInvocationNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "nil node",
			node:     nil,
			expected: false,
		},
		{
			name:     "node without annotations",
			node:     &ast_domain.TemplateNode{},
			expected: false,
		},
		{
			name: "node with annotations but no partial info",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
			},
			expected: false,
		},
		{
			name: "node with partial info",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialAlias: "status_badge",
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.node == nil {

				result := isPartialInvocationNode(&ast_domain.TemplateNode{})
				if result {
					t.Errorf("expected false for empty node, got true")
				}
				return
			}

			result := isPartialInvocationNode(tc.node)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIsPositionInAttributeRange(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		position       protocol.Position
		attributeRange ast_domain.Range
		expected       bool
	}{
		{
			name:     "position inside range",
			position: protocol.Position{Line: 0, Character: 5},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 3},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: true,
		},
		{
			name:     "position at range start",
			position: protocol.Position{Line: 0, Character: 2},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 3},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: true,
		},
		{
			name:     "position at range end",
			position: protocol.Position{Line: 0, Character: 19},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 3},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: true,
		},
		{
			name:     "position before range",
			position: protocol.Position{Line: 0, Character: 0},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 3},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: false,
		},
		{
			name:     "position after range",
			position: protocol.Position{Line: 0, Character: 25},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 1, Column: 3},
				End:   ast_domain.Location{Line: 1, Column: 20},
			},
			expected: false,
		},
		{
			name:     "synthetic range",
			position: protocol.Position{Line: 0, Character: 5},
			attributeRange: ast_domain.Range{
				Start: ast_domain.Location{Line: 0, Column: 0},
				End:   ast_domain.Location{Line: 0, Column: 0},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isPositionInAttributeRange(tc.position, &tc.attributeRange)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestHasAttributeAtPosition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		position protocol.Position
		expected bool
	}{
		{
			name: "position in static attribute",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{
					{
						Name:  "is",
						Value: "status_badge",
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 2, Column: 3},
							End:   ast_domain.Location{Line: 2, Column: 20},
						},
					},
				},
			},
			position: protocol.Position{Line: 1, Character: 10},
			expected: true,
		},
		{
			name: "position in dynamic attribute",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name: "server.badge_colour",
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 3, Column: 3},
							End:   ast_domain.Location{Line: 3, Column: 45},
						},
					},
				},
			},
			position: protocol.Position{Line: 2, Character: 20},
			expected: true,
		},
		{
			name: "position outside all attributes",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{
					{
						Name:  "is",
						Value: "status_badge",
						AttributeRange: ast_domain.Range{
							Start: ast_domain.Location{Line: 2, Column: 3},
							End:   ast_domain.Location{Line: 2, Column: 20},
						},
					},
				},
			},
			position: protocol.Position{Line: 5, Character: 10},
			expected: false,
		},
		{
			name:     "node with no attributes",
			node:     &ast_domain.TemplateNode{},
			position: protocol.Position{Line: 1, Character: 10},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := hasAttributeAtPosition(context.Background(), tc.node, tc.position)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestFindNodeAtPartialInvocationSite(t *testing.T) {
	t.Parallel()

	partialPath := "/path/to/partial.pk"

	testCases := []struct {
		tree          *ast_domain.TemplateAST
		name          string
		expectedAlias string
		position      protocol.Position
		expectFound   bool
	}{
		{
			name: "finds partial invocation by static attribute position",
			tree: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: &partialPath,
							PartialInfo: &ast_domain.PartialInvocationInfo{
								PartialAlias: "status_badge",
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							{
								Name:  "is",
								Value: "status_badge",
								AttributeRange: ast_domain.Range{
									Start: ast_domain.Location{Line: 2, Column: 3},
									End:   ast_domain.Location{Line: 2, Column: 20},
								},
							},
						},
					},
				},
			},
			position:      protocol.Position{Line: 1, Character: 10},
			expectFound:   true,
			expectedAlias: "status_badge",
		},
		{
			name: "finds partial invocation by dynamic attribute position",
			tree: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: &partialPath,
							PartialInfo: &ast_domain.PartialInvocationInfo{
								PartialAlias: "status_badge",
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name: "server.badge_colour",
								AttributeRange: ast_domain.Range{
									Start: ast_domain.Location{Line: 3, Column: 3},
									End:   ast_domain.Location{Line: 3, Column: 45},
								},
							},
						},
					},
				},
			},
			position:      protocol.Position{Line: 2, Character: 20},
			expectFound:   true,
			expectedAlias: "status_badge",
		},
		{
			name: "does not find non-partial node",
			tree: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						TagName:       "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{},
						Attributes: []ast_domain.HTMLAttribute{
							{
								Name:  "class",
								Value: "container",
								AttributeRange: ast_domain.Range{
									Start: ast_domain.Location{Line: 2, Column: 3},
									End:   ast_domain.Location{Line: 2, Column: 20},
								},
							},
						},
					},
				},
			},
			position:    protocol.Position{Line: 1, Character: 10},
			expectFound: false,
		},
		{
			name: "position outside all partial invocations",
			tree: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: &partialPath,
							PartialInfo: &ast_domain.PartialInvocationInfo{
								PartialAlias: "status_badge",
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							{
								Name:  "is",
								Value: "status_badge",
								AttributeRange: ast_domain.Range{
									Start: ast_domain.Location{Line: 2, Column: 3},
									End:   ast_domain.Location{Line: 2, Column: 20},
								},
							},
						},
					},
				},
			},
			position:    protocol.Position{Line: 10, Character: 5},
			expectFound: false,
		},
		{
			name: "finds correct partial among multiple",
			tree: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: &partialPath,
							PartialInfo: &ast_domain.PartialInvocationInfo{
								PartialAlias: "first_badge",
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							{
								Name:  "is",
								Value: "first_badge",
								AttributeRange: ast_domain.Range{
									Start: ast_domain.Location{Line: 2, Column: 3},
									End:   ast_domain.Location{Line: 2, Column: 20},
								},
							},
						},
					},
					{
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: &partialPath,
							PartialInfo: &ast_domain.PartialInvocationInfo{
								PartialAlias: "second_badge",
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							{
								Name:  "is",
								Value: "second_badge",
								AttributeRange: ast_domain.Range{
									Start: ast_domain.Location{Line: 5, Column: 3},
									End:   ast_domain.Location{Line: 5, Column: 22},
								},
							},
						},
					},
				},
			},
			position:      protocol.Position{Line: 4, Character: 10},
			expectFound:   true,
			expectedAlias: "second_badge",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := findNodeAtPartialInvocationSite(context.Background(), tc.tree, tc.position)

			if tc.expectFound {
				if result == nil {
					t.Fatalf("expected to find node, got nil")
				}
				if result.GoAnnotations == nil || result.GoAnnotations.PartialInfo == nil {
					t.Fatalf("expected node to have PartialInfo")
				}
				if result.GoAnnotations.PartialInfo.PartialAlias != tc.expectedAlias {
					t.Errorf("expected alias %q, got %q",
						tc.expectedAlias,
						result.GoAnnotations.PartialInfo.PartialAlias)
				}
			} else {
				if result != nil {
					t.Errorf("expected nil, got node with tag %q", result.TagName)
				}
			}
		})
	}
}
