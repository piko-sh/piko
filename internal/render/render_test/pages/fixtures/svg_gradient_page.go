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
	t := ast_domain.TemplateAST{}
	t.RootNodes = []*ast_domain.TemplateNode{
		ast_domain.NewElementNode("div", []ast_domain.HTMLAttribute{
			{
				Name:           "class",
				Value:          "gallery",
				Location:       ast_domain.Location{},
				NameLocation:   ast_domain.Location{},
				AttributeRange: ast_domain.Range{},
			},
		}, []*ast_domain.TemplateNode{
			ast_domain.NewElementNode("piko:svg", []ast_domain.HTMLAttribute{
				{
					Name:           "src",
					Value:          "icons/star.svg",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
				{
					Name:           "class",
					Value:          "star",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
			}, nil),
			ast_domain.NewElementNode("piko:svg", []ast_domain.HTMLAttribute{
				{
					Name:           "src",
					Value:          "icons/star.svg",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
				{
					Name:           "class",
					Value:          "star star--large",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
			}, nil),
			ast_domain.NewElementNode("piko:svg", []ast_domain.HTMLAttribute{
				{
					Name:           "src",
					Value:          "icons/heart.svg",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
				{
					Name:           "class",
					Value:          "heart",
					Location:       ast_domain.Location{},
					NameLocation:   ast_domain.Location{},
					AttributeRange: ast_domain.Range{},
				},
			}, nil),
		}),
	}
	return &t
}
