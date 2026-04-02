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

// SvgComponentNode creates a template node for an SVG component.
//
// Returns *ast_domain.TemplateNode which represents a piko:svg element with
// source and class attributes configured for testing.
func SvgComponentNode() *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "piko:svg",
		Attributes: []ast_domain.HTMLAttribute{
			{Name: "src", Value: "testmodule/lib/icon.svg"},
			{Name: "class", Value: "icon"},
		},
	}
}

// ComplexPageAST returns a test fixture representing a complex page template.
//
// Returns *ast_domain.TemplateAST which contains a page structure with nested
// components, a form with CSRF annotation, and text requiring HTML escaping.
func ComplexPageAST() *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "main",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "content"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "my-card",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "title", Value: "My Awesome Card"},
						},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "Card content here.",
							},
						},
					},
					SvgComponentNode(),
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "piko:a",
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "href", Value: "/about-us"},
						},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "Learn More",
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "p",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "This text contains characters that need escaping: < > & \" '",
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "form",
						RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
							NeedsCSRF: true,
						},
						Attributes: []ast_domain.HTMLAttribute{
							{Name: "action", Value: "/submit"},
							{Name: "method", Value: "POST"},
						},
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "input",
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "type", Value: "text"},
									{Name: "name", Value: "username"},
								},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "button",
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "type", Value: "submit"},
									{Name: "p-on:click.prevent", Value: "submitAction"},
								},
								Children: []*ast_domain.TemplateNode{
									{
										NodeType:    ast_domain.NodeText,
										TextContent: "Submit",
									},
								},
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "another-component",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType:    ast_domain.NodeText,
								TextContent: "This one is lazy.",
							},
						},
					},
				},
			},
		},
	}
}
