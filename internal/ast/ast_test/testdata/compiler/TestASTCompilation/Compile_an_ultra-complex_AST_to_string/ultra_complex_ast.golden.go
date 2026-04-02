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

package testgolden

import (
	goast "go/ast"
	"go/parser"
)

var GeneratedAST = func() *ast.TemplateAST {
	stringPtr := func(s string) *string {
		return &s
	}
	typeExprFromString := func(s string) goast.Expr {
		expr, err := parser.ParseExpr(s)
		if err != nil {
			return nil
		}
		return expr
	}
	_ = stringPtr
	_ = typeExprFromString
	return &ast.TemplateAST{
		RootNodes: []*ast.TemplateNode{
			&ast.TemplateNode{
				NodeType: ast.NodeElement,
				Location: ast.Location{
					Line:   2,
					Column: 1,
				},
				TagName: "div",
				GoAnnotations: &ast.GoGeneratorAnnotation{
					ResolvedType: &ast.ResolvedTypeInfo{
						TypeExpression: typeExprFromString("map[string][]pkg.MyType"),
						PackageAlias:   "models",
					},
					Symbol: &ast.ResolvedSymbol{
						Name: "KitchenSinkComponent",
						DefinitionLocation: ast.Location{
							Line:   100,
							Column: 1,
						},
					},
					OriginalPackageAlias: stringPtr("main"),
					OriginalSourcePath:   stringPtr("/path/to/ultra_complex.pkc"),
					PartialInfo: &ast.PartialInvocationInfo{
						InvocationKey:       "partial-1",
						PartialAlias:        "widgets",
						PartialPackageName:  "github.com/user/widgets",
						InvokerPackageAlias: "main",
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						RequestOverrides: map[string]ast.PropValue{
							"override_prop": ast.PropValue{
								Expression: &ast.BooleanLiteral{
									Value: true,
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
								},
								Location: ast.Location{
									Line:   1,
									Column: 1,
								},
								GoFieldName: "",
							},
						},
						PassedProps: map[string]ast.PropValue{
							"passed_prop": ast.PropValue{
								Expression: &ast.StringLiteral{
									Value: "value",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
								},
								Location: ast.Location{
									Line:   1,
									Column: 1,
								},
								GoFieldName: "",
							},
						},
					},
					NeedsCSRF: true,
					DynamicAttributeOrigins: map[string]string{
						"aria-label": "local",
						"data-state": "parent-component",
					},
				},
				DirIf: &ast.Directive{
					Type: ast.DirectiveIf,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.UnaryExpr{
						Operator: "!",
						Right: &ast.Identifier{
							Name: "isLoading",
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: nil,
					},
				},
				DirShow: &ast.Directive{
					Type: ast.DirectiveShow,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.Identifier{
						Name: "shouldDisplay",
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				DirClass: &ast.Directive{
					Type: ast.DirectiveClass,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.ObjectLiteral{
						Pairs: map[string]ast.Expression{
							"has-warning": &ast.TernaryExpr{
								Condition: &ast.Identifier{
									Name: "errorCount",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Consequent: &ast.BooleanLiteral{
									Value: true,
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
								},
								Alternate: &ast.BooleanLiteral{
									Value: false,
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
								},
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
							"is-active": &ast.MemberExpr{
								Base: &ast.Identifier{
									Name: "user",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Property: &ast.Identifier{
									Name: "isActive",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Optional: false,
								Computed: false,
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: nil,
					},
				},
				DirStyle: &ast.Directive{
					Type: ast.DirectiveStyle,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.ObjectLiteral{
						Pairs: map[string]ast.Expression{
							"background": &ast.StringLiteral{
								Value: "blue",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
							"color": &ast.Identifier{
								Name: "fontColor",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							"fontSize": &ast.StringLiteral{
								Value: "16px",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
						},
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: nil,
					},
				},
				Attributes: []ast.HTMLAttribute{
					ast.HTMLAttribute{
						Name:  "id",
						Value: "kitchen-sink",
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast.HTMLAttribute{
						Name:  "class",
						Value: "container theme-dark",
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast.HTMLAttribute{
						Name:  "disabled",
						Value: "",
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast.HTMLAttribute{
						Name:  "data-ação",
						Value: "teste",
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				DynamicAttributes: []ast.DynamicAttribute{
					ast.DynamicAttribute{
						Name:          "aria-label",
						RawExpression: "",
						Expression: &ast.StringLiteral{
							Value: "Main container",
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							GoAnnotations: nil,
						},
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast.DynamicAttribute{
						Name:          "data-state",
						RawExpression: "",
						Expression: &ast.TernaryExpr{
							Condition: &ast.MemberExpr{
								Base: &ast.Identifier{
									Name: "user",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Property: &ast.Identifier{
									Name: "isActive",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Optional: false,
								Computed: false,
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Consequent: &ast.StringLiteral{
								Value: "active",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
							Alternate: &ast.StringLiteral{
								Value: "inactive",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							GoAnnotations: nil,
						},
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast.DynamicAttribute{
						Name:          "data-dynamic-string",
						RawExpression: "",
						Expression: &ast.TemplateLiteral{
							Parts: []ast.TemplateLiteralPart{
								ast.TemplateLiteralPart{
									IsLiteral: true,
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
									Literal:       "Outer: ",
								},
								ast.TemplateLiteralPart{
									IsLiteral: false,
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
									Expression: &ast.TemplateLiteral{
										Parts: []ast.TemplateLiteralPart{
											ast.TemplateLiteralPart{
												IsLiteral: true,
												RelativeLocation: ast.Location{
													Line:   0,
													Column: 0,
												},
												GoAnnotations: nil,
												Literal:       "Inner: ",
											},
											ast.TemplateLiteralPart{
												IsLiteral: false,
												RelativeLocation: ast.Location{
													Line:   0,
													Column: 0,
												},
												GoAnnotations: nil,
												Expression: &ast.Identifier{
													Name: "user",
													RelativeLocation: ast.Location{
														Line:   0,
														Column: 0,
													},
												},
											},
										},
										RelativeLocation: ast.Location{
											Line:   0,
											Column: 0,
										},
									},
								},
							},
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				Binds: map[string]*ast.Directive{
					"data-user-id": &ast.Directive{
						Type: ast.DirectiveBind,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
						Arg: "data-user-id",
						Expression: &ast.MemberExpr{
							Base: &ast.Identifier{
								Name: "user",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Property: &ast.Identifier{
								Name: "id",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Optional: false,
							Computed: false,
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
					},
					"title": &ast.Directive{
						Type: ast.DirectiveBind,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
						Arg: "title",
						Expression: &ast.MemberExpr{
							Base: &ast.Identifier{
								Name: "page",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Property: &ast.Identifier{
								Name: "title",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Optional: false,
							Computed: false,
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
					},
				},
				OnEvents: map[string][]ast.Directive{
					"click": []ast.Directive{
						ast.Directive{
							Type: ast.DirectiveOn,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Arg: "click",
							Expression: &ast.Identifier{
								Name: "handleClick",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						ast.Directive{
							Type: ast.DirectiveOn,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Arg: "click",
							Expression: &ast.Identifier{
								Name: "trackClick",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
					"submit": []ast.Directive{
						ast.Directive{
							Type: ast.DirectiveOn,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Arg:      "submit",
							Modifier: "prevent",
							Expression: &ast.Identifier{
								Name: "handleSubmit",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
				},
				CustomEvents: map[string][]ast.Directive{
					"custom-update": []ast.Directive{
						ast.Directive{
							Type: ast.DirectiveEvent,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Arg: "custom-update",
							Expression: &ast.Identifier{
								Name: "onCustomUpdate",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
				},
				Children: []*ast.TemplateNode{
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "div",
						Children: []*ast.TemplateNode{
							&ast.TemplateNode{
								NodeType: ast.NodeElement,
								Location: ast.Location{
									Line:   0,
									Column: 0,
								},
								TagName: "div",
								Children: []*ast.TemplateNode{
									&ast.TemplateNode{
										NodeType: ast.NodeElement,
										Location: ast.Location{
											Line:   0,
											Column: 0,
										},
										TagName: "p",
										DirIf: &ast.Directive{
											Type: ast.DirectiveIf,
											Location: ast.Location{
												Line:   0,
												Column: 0,
											},
											NameLocation: ast.Location{
												Line:   0,
												Column: 0,
											},
											Expression: &ast.Identifier{
												Name: "showParagraph",
												RelativeLocation: ast.Location{
													Line:   0,
													Column: 0,
												},
											},
										},
										Children: []*ast.TemplateNode{
											&ast.TemplateNode{
												NodeType: ast.NodeElement,
												Location: ast.Location{
													Line:   0,
													Column: 0,
												},
												TagName: "span",
												Children: []*ast.TemplateNode{
													&ast.TemplateNode{
														NodeType: ast.NodeText,
														Location: ast.Location{
															Line:   0,
															Column: 0,
														},
														RichText: []ast.TextPart{
															ast.TextPart{
																IsLiteral: true,
																Location: ast.Location{
																	Line:   0,
																	Column: 0,
																},
																Literal: "Link: ",
															},
															ast.TextPart{
																IsLiteral: false,
																Location: ast.Location{
																	Line:   0,
																	Column: 0,
																},
																RawExpression: "",
																Expression: &ast.TemplateLiteral{
																	Parts: []ast.TemplateLiteralPart{
																		ast.TemplateLiteralPart{
																			IsLiteral: true,
																			RelativeLocation: ast.Location{
																				Line:   0,
																				Column: 0,
																			},
																			GoAnnotations: nil,
																			Literal:       "/users/",
																		},
																		ast.TemplateLiteralPart{
																			IsLiteral: false,
																			RelativeLocation: ast.Location{
																				Line:   0,
																				Column: 0,
																			},
																			GoAnnotations: nil,
																			Expression: &ast.Identifier{
																				Name: "userId",
																				RelativeLocation: ast.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																		},
																	},
																	RelativeLocation: ast.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					&ast.TemplateNode{
						NodeType: ast.NodeComment,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TextContent: " this is a comment ",
					},
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "section",
						Directives: []ast.Directive{
							ast.Directive{
								Type: ast.DirectiveIf,
								Location: ast.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
				},
			},
			&ast.TemplateNode{
				NodeType: ast.NodeFragment,
				Location: ast.Location{
					Line:   20,
					Column: 1,
				},
				Children: []*ast.TemplateNode{
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "template",
						DirFor: &ast.Directive{
							Type: ast.DirectiveFor,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.ForInExpr{
								IndexVariable: nil,
								ItemVariable: &ast.Identifier{
									Name: "item",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Collection: &ast.Identifier{
									Name: "items",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
						},
						Children: []*ast.TemplateNode{
							&ast.TemplateNode{
								NodeType: ast.NodeElement,
								Location: ast.Location{
									Line:   0,
									Column: 0,
								},
								TagName: "h2",
								Children: []*ast.TemplateNode{
									&ast.TemplateNode{
										NodeType: ast.NodeText,
										Location: ast.Location{
											Line:   0,
											Column: 0,
										},
										TextContent: "Title",
									},
								},
							},
							&ast.TemplateNode{
								NodeType: ast.NodeElement,
								Location: ast.Location{
									Line:   0,
									Column: 0,
								},
								TagName: "p",
								Children: []*ast.TemplateNode{
									&ast.TemplateNode{
										NodeType: ast.NodeText,
										Location: ast.Location{
											Line:   0,
											Column: 0,
										},
										TextContent: "Content",
									},
								},
							},
						},
					},
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "template",
						DirIf: &ast.Directive{
							Type: ast.DirectiveIf,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.Identifier{
								Name: "condition",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "div",
						DirElse: &ast.Directive{
							Type: ast.DirectiveElse,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
					},
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "div",
						DirIf: &ast.Directive{
							Type: ast.DirectiveIf,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.Identifier{
								Name: "c1",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
					&ast.TemplateNode{
						NodeType: ast.NodeElement,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "div",
						DirElseIf: &ast.Directive{
							Type: ast.DirectiveElseIf,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.Identifier{
								Name: "c2",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						DirFor: &ast.Directive{
							Type: ast.DirectiveFor,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.ForInExpr{
								IndexVariable: nil,
								ItemVariable: &ast.Identifier{
									Name: "i",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Collection: &ast.Identifier{
									Name: "items",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
						},
						DirKey: &ast.Directive{
							Type: ast.DirectiveKey,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.Identifier{
								Name: "i",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						DirContext: &ast.Directive{
							Type: ast.DirectiveContext,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Expression: &ast.StringLiteral{
								Value: "loop",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: nil,
							},
						},
					},
				},
			},
			&ast.TemplateNode{
				NodeType: ast.NodeElement,
				Location: ast.Location{
					Line:   30,
					Column: 1,
				},
				TagName: "input",
				GoAnnotations: &ast.GoGeneratorAnnotation{
					ResolvedType: &ast.ResolvedTypeInfo{
						TypeExpression: typeExprFromString("int"),
						PackageAlias:   "",
					},
					Symbol: &ast.ResolvedSymbol{
						Name: "formInput",
						DefinitionLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				DirModel: &ast.Directive{
					Type: ast.DirectiveModel,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.Identifier{
						Name: "form.value",
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				DirKey: &ast.Directive{
					Type: ast.DirectiveKey,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.MemberExpr{
						Base: &ast.Identifier{
							Name: "form",
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						Property: &ast.Identifier{
							Name: "id",
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						Optional: false,
						Computed: false,
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				DynamicAttributes: []ast.DynamicAttribute{
					ast.DynamicAttribute{
						Name:          "data-calc",
						RawExpression: "",
						Expression: &ast.BinaryExpr{
							Left: &ast.BinaryExpr{
								Left: &ast.Identifier{
									Name: "a",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								Operator: "-",
								Right: &ast.Identifier{
									Name: "b",
									RelativeLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Operator: "-",
							Right: &ast.Identifier{
								Name: "c",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast.DynamicAttribute{
						Name:          "data-for",
						RawExpression: "",
						Expression: &ast.MemberExpr{
							Base: &ast.Identifier{
								Name: "data",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Property: &ast.Identifier{
								Name: "for",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							Optional: false,
							Computed: false,
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
			},
			&ast.TemplateNode{
				NodeType: ast.NodeComment,
				Location: ast.Location{
					Line:   0,
					Column: 0,
				},
				TextContent: " Final Section ",
			},
			&ast.TemplateNode{
				NodeType: ast.NodeText,
				Location: ast.Location{
					Line:   0,
					Column: 0,
				},
				RichText: []ast.TextPart{
					ast.TextPart{
						IsLiteral: false,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						RawExpression: "  finalMessage  ",
						Expression: &ast.Identifier{
							Name: "finalMessage",
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
					},
				},
			},
			&ast.TemplateNode{
				NodeType: ast.NodeElement,
				Location: ast.Location{
					Line:   0,
					Column: 0,
				},
				TagName: "textarea",
				Children: []*ast.TemplateNode{
					&ast.TemplateNode{
						NodeType: ast.NodeText,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						TextContent: "  Line 1\n  <span>not a tag</span>",
					},
				},
			},
			&ast.TemplateNode{
				NodeType: ast.NodeElement,
				Location: ast.Location{
					Line:   40,
					Column: 1,
				},
				TagName: "p",
				GoAnnotations: &ast.GoGeneratorAnnotation{
					ResolvedType: &ast.ResolvedTypeInfo{
						TypeExpression: typeExprFromString("*string"),
						PackageAlias:   "",
					},
					Symbol: &ast.ResolvedSymbol{
						Name: "optionalMessage",
						DefinitionLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
				DirText: &ast.Directive{
					Type: ast.DirectiveText,
					Location: ast.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast.Identifier{
						Name: "optionalMessage",
						RelativeLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
			},
			&ast.TemplateNode{
				NodeType: ast.NodeElement,
				Location: ast.Location{
					Line:   41,
					Column: 1,
				},
				TagName: "span",
				Directives: []ast.Directive{
					ast.Directive{
						Type: ast.DirectiveOn,
						Location: ast.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast.Location{
							Line:   0,
							Column: 0,
						},
						Arg: "load",
						Expression: &ast.Identifier{
							Name: "loadData",
							RelativeLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
						},
						GoAnnotations: &ast.GoGeneratorAnnotation{
							ResolvedType: &ast.ResolvedTypeInfo{
								TypeExpression: typeExprFromString("io.Reader"),
								PackageAlias:   "io",
							},
							Symbol: &ast.ResolvedSymbol{
								Name: "dataSource",
								DefinitionLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							PartialInfo: &ast.PartialInvocationInfo{
								InvocationKey:       "",
								PartialAlias:        "",
								PartialPackageName:  "",
								InvokerPackageAlias: "",
								Location: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
					},
				},
				OnEvents: map[string][]ast.Directive{
					"load": []ast.Directive{
						ast.Directive{
							Type: ast.DirectiveOn,
							Location: ast.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast.Location{
								Line:   0,
								Column: 0,
							},
							Arg: "load",
							Expression: &ast.Identifier{
								Name: "loadData",
								RelativeLocation: ast.Location{
									Line:   0,
									Column: 0,
								},
							},
							GoAnnotations: &ast.GoGeneratorAnnotation{
								ResolvedType: &ast.ResolvedTypeInfo{
									TypeExpression: typeExprFromString("io.Reader"),
									PackageAlias:   "io",
								},
								Symbol: &ast.ResolvedSymbol{
									Name: "dataSource",
									DefinitionLocation: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
								PartialInfo: &ast.PartialInvocationInfo{
									InvocationKey:       "",
									PartialAlias:        "",
									PartialPackageName:  "",
									InvokerPackageAlias: "",
									Location: ast.Location{
										Line:   0,
										Column: 0,
									},
								},
							},
						},
					},
				},
			},
		},
		Diagnostics: []*ast.Diagnostic{
			&ast.Diagnostic{
				Message:  "Root-level diagnostic message.",
				Severity: ast.Info,
				Location: ast.Location{
					Line:   1,
					Column: 1,
				},
				Expression: "",
				SourcePath: "",
			},
		},
	}
}()
