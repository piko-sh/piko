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

// MegaComplexPageAST returns a complex template AST for testing purposes.
//
// Returns *ast_domain.TemplateAST which contains a multi-level page structure
// with header, navigation, main content, forms, and footer elements.
func MegaComplexPageAST() *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "mega-complex-root"},
					{Name: "data-testid", Value: "root-element"},
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "header",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "div",
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "class", Value: "logo-container"},
								},
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "piko:svg",
										Attributes: []ast_domain.HTMLAttribute{
											{Name: "src", Value: "logo.svg"},
											{Name: "class", Value: "logo-class-from-component"},
											{Name: "aria-label", Value: "Company Logo"},
										},
									},
								},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "nav",
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "piko:a",
										Attributes: []ast_domain.HTMLAttribute{
											{Name: "href", Value: "/about-us"},
											{Name: "class", Value: "nav-link"},
										},
										Children: []*ast_domain.TemplateNode{
											{NodeType: ast_domain.NodeText, TextContent: "About"},
										},
									},
								},
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "main",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "h1",
								Children: []*ast_domain.TemplateNode{
									{NodeType: ast_domain.NodeText, TextContent: "Product Page for <Widget & Co.>"},
								},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "my-card",
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "card-title", Value: "The Amazing Widget"},
								},
								Children: []*ast_domain.TemplateNode{
									{NodeType: ast_domain.NodeText, TextContent: "This card will be eagerly loaded."},
								},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "div",
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "style", Value: "border: 1px solid green; display: none !important;"},
								},
								Children: []*ast_domain.TemplateNode{
									{NodeType: ast_domain.NodeText, TextContent: "This content is hidden by p-show."},
								},
							},
							{
								NodeType:  ast_domain.NodeElement,
								TagName:   "div",
								InnerHTML: "<strong>This is raw, unescaped HTML.</strong>",
								Children:  []*ast_domain.TemplateNode{},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "form",
								RuntimeAnnotations: &ast_domain.RuntimeAnnotation{
									NeedsCSRF: true,
								},
								Attributes: []ast_domain.HTMLAttribute{
									{Name: "method", Value: "POST"},
								},
								Children: []*ast_domain.TemplateNode{
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "input",
										Attributes: []ast_domain.HTMLAttribute{
											{Name: "type", Value: "text"},
											{Name: "name", Value: "email"},
										},
									},
									{
										NodeType: ast_domain.NodeElement,
										TagName:  "button",
										OnEvents: map[string][]ast_domain.Directive{
											"click": {
												{Type: ast_domain.DirectiveOn, Modifier: "action", RawExpression: "submitFormAction"},
											},
										},
										Children: []*ast_domain.TemplateNode{
											{NodeType: ast_domain.NodeText, TextContent: "Submit"},
										},
									},
								},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "div",
								DirRef: &ast_domain.Directive{
									Type:          ast_domain.DirectiveRef,
									RawExpression: "myDiv",
									Expression:    &ast_domain.StringLiteral{Value: "myDiv"},
								},
								OnEvents: map[string][]ast_domain.Directive{
									"click": {
										{Type: ast_domain.DirectiveOn, Modifier: "helper", RawExpression: "showToast('Clicked!')"},
									},
								},
								Children: []*ast_domain.TemplateNode{
									{NodeType: ast_domain.NodeText, TextContent: "Click me to show a toast."},
								},
							},
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "another-component",
								Children: []*ast_domain.TemplateNode{
									{NodeType: ast_domain.NodeText, TextContent: "This one is lazy loaded."},
								},
							},
						},
					},
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "footer",
						Children: []*ast_domain.TemplateNode{
							{NodeType: ast_domain.NodeComment, TextContent: " End of complex page "},
						},
					},
				},
			},
		},
	}
}
