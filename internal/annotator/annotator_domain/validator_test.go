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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestValidatePMLUsage(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for email component", func(t *testing.T) {
		t.Parallel()
		component := &annotator_dto.ParsedComponent{
			ComponentType: "email",
			Template: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "pml-button",
					},
				},
			},
		}

		result := validatePMLUsage(component)

		assert.Nil(t, result)
	})

	t.Run("returns nil for component with no template", func(t *testing.T) {
		t.Parallel()
		component := &annotator_dto.ParsedComponent{
			ComponentType: "page",
			Template:      nil,
		}

		result := validatePMLUsage(component)

		assert.Nil(t, result)
	})

	t.Run("returns diagnostic for pml tag in non-email component", func(t *testing.T) {
		t.Parallel()
		component := &annotator_dto.ParsedComponent{
			ComponentType: "page",
			SourcePath:    "/test/page.pk",
			Template: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-button",
								Location: ast_domain.Location{Line: 5, Column: 2, Offset: 50},
							},
						},
					},
				},
			},
		}

		result := validatePMLUsage(component)

		assert.Len(t, result, 1)
		assert.Equal(t, ast_domain.Warning, result[0].Severity)
		assert.Contains(t, result[0].Message, "pml-button")
		assert.Contains(t, result[0].Message, "not supported outside of email templates")
	})

	t.Run("returns multiple diagnostics for multiple pml tags", func(t *testing.T) {
		t.Parallel()
		component := &annotator_dto.ParsedComponent{
			ComponentType: "page",
			SourcePath:    "/test/page.pk",
			Template: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-button",
								Location: ast_domain.Location{Line: 5, Column: 2, Offset: 50},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "pml-text",
								Location: ast_domain.Location{Line: 6, Column: 2, Offset: 70},
							},
						},
					},
				},
			},
		}

		result := validatePMLUsage(component)

		assert.Len(t, result, 2)
		assert.Contains(t, result[0].Message, "pml-button")
		assert.Contains(t, result[1].Message, "pml-text")
	})

	t.Run("returns nil for non-pml tags", func(t *testing.T) {
		t.Parallel()
		component := &annotator_dto.ParsedComponent{
			ComponentType: "page",
			Template: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "button",
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "piko:Button",
							},
						},
					},
				},
			},
		}

		result := validatePMLUsage(component)

		assert.Nil(t, result)
	})

	t.Run("ignores non-element nodes", func(t *testing.T) {
		t.Parallel()
		component := &annotator_dto.ParsedComponent{
			ComponentType: "page",
			Template: &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeText,
								TagName:  "pml-button",
							},
						},
					},
				},
			},
		}

		result := validatePMLUsage(component)

		assert.Nil(t, result)
	})
}
