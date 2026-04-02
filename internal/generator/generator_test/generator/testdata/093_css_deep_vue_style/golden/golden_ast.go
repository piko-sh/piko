package default_test_pkg

import (
	goast "go/ast"
	"go/parser"

	"piko.sh/piko/internal/ast/ast_domain"
)

var GeneratedAST = func() *ast_domain.TemplateAST {
	typeExprFromString := func(s string) goast.Expr {
		expr, err := parser.ParseExpr(s)
		if err != nil {
			return nil
		}
		return expr
	}
	_ = typeExprFromString
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			&ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Location: ast_domain.Location{
					Line:   22,
					Column: 3,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   22,
						Column: 3,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("pages/main.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "page-wrapper",
						Location: ast_domain.Location{
							Line:   22,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   23,
							Column: 5,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   23,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "page-title",
								Location: ast_domain.Location{
									Line:   23,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 9,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   23,
									Column: 28,
								},
								TextContent: "Deep Selector Test Page",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 28,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_container_cba4efcb"),
							OriginalSourcePath:   new("partials/container.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "container_37e22e6d",
								PartialAlias:        "container",
								PartialPackageName:  "partials_container_cba4efcb",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/container.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "container",
								Location: ast_domain.Location{
									Line:   22,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   23,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_container_cba4efcb"),
									OriginalSourcePath:   new("partials/container.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/container.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "wrapper",
										Location: ast_domain.Location{
											Line:   23,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 10,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 7,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_container_cba4efcb"),
											OriginalSourcePath:   new("partials/container.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/container.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "local-item",
												Location: ast_domain.Location{
													Line:   24,
													Column: 20,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 13,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 32,
												},
												TextContent: "Local content",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_container_cba4efcb"),
													OriginalSourcePath:   new("partials/container.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 32,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/container.pk"),
														Stringability:      1,
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   22,
											Column: 3,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_nested_content_edda9701"),
											OriginalSourcePath:   new("partials/nested_content.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "nested_content_container_37e22e6d_e1077ca9",
												PartialAlias:        "nested_content",
												PartialPackageName:  "partials_nested_content_edda9701",
												InvokerPackageAlias: "partials_container_cba4efcb",
												Location: ast_domain.Location{
													Line:   25,
													Column: 7,
												},
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   22,
												Column: 3,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/nested_content.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "nested-wrapper",
												Location: ast_domain.Location{
													Line:   22,
													Column: 15,
												},
												NameLocation: ast_domain.Location{
													Line:   22,
													Column: 8,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   23,
													Column: 5,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_nested_content_edda9701"),
													OriginalSourcePath:   new("partials/nested_content.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/nested_content.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "nested-item",
														Location: ast_domain.Location{
															Line:   23,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   23,
															Column: 11,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   23,
															Column: 31,
														},
														TextContent: "Nested content from child partial",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_nested_content_edda9701"),
															OriginalSourcePath:   new("partials/nested_content.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 31,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/nested_content.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   24,
													Column: 5,
												},
												TagName: "div",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_nested_content_edda9701"),
													OriginalSourcePath:   new("partials/nested_content.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:1",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/nested_content.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "sub-item",
														Location: ast_domain.Location{
															Line:   24,
															Column: 17,
														},
														NameLocation: ast_domain.Location{
															Line:   24,
															Column: 10,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   24,
															Column: 27,
														},
														TextContent: "Sub item",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_nested_content_edda9701"),
															OriginalSourcePath:   new("partials/nested_content.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   24,
																Column: 27,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/nested_content.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   25,
													Column: 5,
												},
												TagName: "div",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_nested_content_edda9701"),
													OriginalSourcePath:   new("partials/nested_content.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:2",
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/nested_content.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "a",
														Location: ast_domain.Location{
															Line:   25,
															Column: 17,
														},
														NameLocation: ast_domain.Location{
															Line:   25,
															Column: 10,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   26,
															Column: 7,
														},
														TagName: "div",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_nested_content_edda9701"),
															OriginalSourcePath:   new("partials/nested_content.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:2:0",
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/nested_content.pk"),
																Stringability:      1,
															},
														},
														Attributes: []ast_domain.HTMLAttribute{
															ast_domain.HTMLAttribute{
																Name:  "class",
																Value: "b",
																Location: ast_domain.Location{
																	Line:   26,
																	Column: 19,
																},
																NameLocation: ast_domain.Location{
																	Line:   26,
																	Column: 12,
																},
															},
														},
														Children: []*ast_domain.TemplateNode{
															&ast_domain.TemplateNode{
																NodeType: ast_domain.NodeElement,
																Location: ast_domain.Location{
																	Line:   27,
																	Column: 9,
																},
																TagName: "div",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_nested_content_edda9701"),
																	OriginalSourcePath:   new("partials/nested_content.pk"),
																	IsStatic:             true,
																	IsStructurallyStatic: true,
																},
																Key: &ast_domain.StringLiteral{
																	Value: "r.0:1:0:1:2:0:0",
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/nested_content.pk"),
																		Stringability:      1,
																	},
																},
																Attributes: []ast_domain.HTMLAttribute{
																	ast_domain.HTMLAttribute{
																		Name:  "class",
																		Value: "c",
																		Location: ast_domain.Location{
																			Line:   27,
																			Column: 21,
																		},
																		NameLocation: ast_domain.Location{
																			Line:   27,
																			Column: 14,
																		},
																	},
																},
																Children: []*ast_domain.TemplateNode{
																	&ast_domain.TemplateNode{
																		NodeType: ast_domain.NodeText,
																		Location: ast_domain.Location{
																			Line:   27,
																			Column: 24,
																		},
																		TextContent: "Deeply nested",
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			OriginalPackageAlias: new("partials_nested_content_edda9701"),
																			OriginalSourcePath:   new("partials/nested_content.pk"),
																			IsStatic:             true,
																			IsStructurallyStatic: true,
																		},
																		Key: &ast_domain.StringLiteral{
																			Value: "r.0:1:0:1:2:0:0:0",
																			RelativeLocation: ast_domain.Location{
																				Line:   27,
																				Column: 24,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/nested_content.pk"),
																				Stringability:      1,
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
						},
					},
				},
			},
		},
	}
}()
