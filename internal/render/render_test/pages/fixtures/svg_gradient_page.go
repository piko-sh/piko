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

package fixtures

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// SvgGradientPageAST returns a page that uses gradient-bearing SVGs. The star icon is
// used twice to exercise symbol and definition deduplication, and the heart icon ships a
// gradient with the same Figma-style identifier as the star to exercise per-asset
// namespacing.
//
// Returns *ast_domain.TemplateAST which contains three piko:svg usages.
func SvgGradientPageAST() *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "gallery"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:svg",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "src", Value: "icons/star.svg"},
							{Name: "class", Value: "star"},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:svg",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "src", Value: "icons/star.svg"},
							{Name: "class", Value: "star star--large"},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:svg",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "src", Value: "icons/heart.svg"},
							{Name: "class", Value: "heart"},
						},
					},
				},
			},
		},
	}
}
