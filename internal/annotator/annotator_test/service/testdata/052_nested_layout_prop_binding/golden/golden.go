package test

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
					Line:   32,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_outer_layout_643254a4"),
					OriginalSourcePath:   new("partials/outer_layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "outer_layout_118242d2",
						PartialAlias:        "outer_layout",
						PartialPackageName:  "partials_outer_layout_643254a4",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   42,
							Column: 2,
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   32,
						Column: 2,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("partials/outer_layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "outer-layout",
						Location: ast_domain.Location{
							Line:   32,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   32,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   33,
							Column: 3,
						},
						TagName: "header",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_outer_layout_643254a4"),
							OriginalSourcePath:   new("partials/outer_layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   33,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/outer_layout.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   33,
									Column: 11,
								},
								TextContent: "Outer Layout Header",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_outer_layout_643254a4"),
									OriginalSourcePath:   new("partials/outer_layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   33,
										Column: 11,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/outer_layout.pk"),
										Stringability:      1,
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   34,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_outer_layout_643254a4"),
							OriginalSourcePath:   new("partials/outer_layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   34,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/outer_layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "outer-content",
								Location: ast_domain.Location{
									Line:   34,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   34,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   32,
									Column: 2,
								},
								TagName: "section",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_inner_layout_981558c4"),
									OriginalSourcePath:   new("partials/inner_layout.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "inner_layout_outer_layout_118242d2_503bec27",
										PartialAlias:        "inner_layout",
										PartialPackageName:  "partials_inner_layout_981558c4",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   43,
											Column: 3,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   32,
										Column: 2,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/inner_layout.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "inner-layout",
										Location: ast_domain.Location{
											Line:   32,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   32,
											Column: 11,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   33,
											Column: 3,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_layout_981558c4"),
											OriginalSourcePath:   new("partials/inner_layout.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   33,
												Column: 3,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/inner_layout.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "inner-header",
												Location: ast_domain.Location{
													Line:   33,
													Column: 15,
												},
												NameLocation: ast_domain.Location{
													Line:   33,
													Column: 8,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   33,
													Column: 29,
												},
												TextContent: "Inner Layout",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_inner_layout_981558c4"),
													OriginalSourcePath:   new("partials/inner_layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   33,
														Column: 29,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/inner_layout.pk"),
														Stringability:      1,
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   34,
											Column: 3,
										},
										TagName: "main",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_layout_981558c4"),
											OriginalSourcePath:   new("partials/inner_layout.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   34,
												Column: 3,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/inner_layout.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "inner-content",
												Location: ast_domain.Location{
													Line:   34,
													Column: 16,
												},
												NameLocation: ast_domain.Location{
													Line:   34,
													Column: 9,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   41,
													Column: 2,
												},
												TagName: "div",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_child_d247007e"),
													OriginalSourcePath:   new("partials/child.pk"),
													PartialInfo: &ast_domain.PartialInvocationInfo{
														InvocationKey:       "child_inner_layout_outer_layout_118242d2_503bec27_theme_state_theme_user_id_state_userid_a03891c3",
														PartialAlias:        "child",
														PartialPackageName:  "partials_child_d247007e",
														InvokerPackageAlias: "main_aaf9a2e0",
														Location: ast_domain.Location{
															Line:   44,
															Column: 4,
														},
														PassedProps: map[string]ast_domain.PropValue{
															"theme": ast_domain.PropValue{
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   44,
																					Column: 79,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("pageData"),
																			OriginalSourcePath: new("main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Theme",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Theme",
																				ReferenceLocation: ast_domain.Location{
																					Line:   44,
																					Column: 79,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   35,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "main_aaf9a2e0",
																					CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Theme",
																					ReferenceLocation: ast_domain.Location{
																						Line:   44,
																						Column: 79,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   35,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName: new("pageData"),
																			},
																			BaseCodeGenVarName:  new("pageData"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																			Stringability:       1,
																		},
																	},
																	Optional: false,
																	Computed: false,
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Theme",
																			ReferenceLocation: ast_domain.Location{
																				Line:   44,
																				Column: 79,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   35,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Theme",
																				ReferenceLocation: ast_domain.Location{
																					Line:   44,
																					Column: 79,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   35,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName: new("pageData"),
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   44,
																	Column: 79,
																},
																GoFieldName: "Theme",
																InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Theme",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 79,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   35,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Theme",
																			ReferenceLocation: ast_domain.Location{
																				Line:   44,
																				Column: 79,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   35,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("pageData"),
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	Stringability:       1,
																},
															},
															"user_id": ast_domain.PropValue{
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   44,
																					Column: 57,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("pageData"),
																			OriginalSourcePath: new("main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "UserID",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "UserID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   44,
																					Column: 57,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   36,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "main_aaf9a2e0",
																					CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "UserID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   44,
																						Column: 57,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   36,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName: new("pageData"),
																			},
																			BaseCodeGenVarName:  new("pageData"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																			Stringability:       1,
																		},
																	},
																	Optional: false,
																	Computed: false,
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "UserID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   44,
																				Column: 57,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   36,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "UserID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   44,
																					Column: 57,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   36,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName: new("pageData"),
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   44,
																	Column: 57,
																},
																GoFieldName: "UserID",
																InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "UserID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 57,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   36,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "UserID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   44,
																				Column: 57,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   36,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("pageData"),
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	Stringability:       1,
																},
															},
														},
													},
													DynamicAttributeOrigins: map[string]string{
														"theme":   "main_aaf9a2e0",
														"user_id": "main_aaf9a2e0",
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   41,
														Column: 2,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/child.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "child-component",
														Location: ast_domain.Location{
															Line:   41,
															Column: 14,
														},
														NameLocation: ast_domain.Location{
															Line:   41,
															Column: 7,
														},
													},
													ast_domain.HTMLAttribute{
														Name:  "id",
														Value: "nested-child",
														Location: ast_domain.Location{
															Line:   44,
															Column: 22,
														},
														NameLocation: ast_domain.Location{
															Line:   44,
															Column: 18,
														},
													},
												},
												DynamicAttributes: []ast_domain.DynamicAttribute{
													ast_domain.DynamicAttribute{
														Name:          "theme",
														RawExpression: "state.Theme",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 79,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Theme",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Theme",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 79,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   35,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Theme",
																	ReferenceLocation: ast_domain.Location{
																		Line:   44,
																		Column: 79,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   35,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   44,
															Column: 79,
														},
														NameLocation: ast_domain.Location{
															Line:   44,
															Column: 71,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Theme",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 79,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   35,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       1,
														},
													},
													ast_domain.DynamicAttribute{
														Name:          "user_id",
														RawExpression: "state.UserID",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 57,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "UserID",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "UserID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 57,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   36,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "UserID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   44,
																		Column: 57,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   36,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   44,
															Column: 57,
														},
														NameLocation: ast_domain.Location{
															Line:   44,
															Column: 47,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UserID",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 57,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   36,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       1,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   42,
															Column: 3,
														},
														TagName: "span",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_child_d247007e"),
															OriginalSourcePath:   new("partials/child.pk"),
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   42,
																Column: 3,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/child.pk"),
																Stringability:      1,
															},
														},
														Children: []*ast_domain.TemplateNode{
															&ast_domain.TemplateNode{
																NodeType: ast_domain.NodeText,
																Location: ast_domain.Location{
																	Line:   42,
																	Column: 9,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_child_d247007e"),
																	OriginalSourcePath:   new("partials/child.pk"),
																},
																Key: &ast_domain.StringLiteral{
																	Value: "r.0:1:0:1:0:0:0",
																	RelativeLocation: ast_domain.Location{
																		Line:   42,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/child.pk"),
																		Stringability:      1,
																	},
																},
																RichText: []ast_domain.TextPart{
																	ast_domain.TextPart{
																		IsLiteral: false,
																		Location: ast_domain.Location{
																			Line:   42,
																			Column: 12,
																		},
																		RawExpression: "state.DisplayText",
																		Expression: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "state",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("partials_child_d247007e.Response"),
																						PackageAlias:         "partials_child_d247007e",
																						CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/partials/partials_child_d247007e",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "state",
																						ReferenceLocation: ast_domain.Location{
																							Line:   42,
																							Column: 12,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_503bec27_theme_state_theme_user_id_state_userid_a03891c3"),
																					OriginalSourcePath: new("partials/child.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "DisplayText",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "partials_child_d247007e",
																						CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/partials/partials_child_d247007e",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "DisplayText",
																						ReferenceLocation: ast_domain.Location{
																							Line:   42,
																							Column: 12,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																					},
																					BaseCodeGenVarName:  new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_503bec27_theme_state_theme_user_id_state_userid_a03891c3"),
																					OriginalSourcePath:  new("partials/child.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
																					Stringability:       1,
																				},
																			},
																			Optional: false,
																			Computed: false,
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "partials_child_d247007e",
																					CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/partials/partials_child_d247007e",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "DisplayText",
																					ReferenceLocation: ast_domain.Location{
																						Line:   42,
																						Column: 12,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   29,
																						Column: 23,
																					},
																				},
																				BaseCodeGenVarName:  new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_503bec27_theme_state_theme_user_id_state_userid_a03891c3"),
																				OriginalSourcePath:  new("partials/child.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
																				Stringability:       1,
																			},
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_child_d247007e",
																				CanonicalPackagePath: "testcase_52_nested_layout_prop_binding/dist/partials/partials_child_d247007e",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "DisplayText",
																				ReferenceLocation: ast_domain.Location{
																					Line:   42,
																					Column: 12,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   29,
																					Column: 23,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_503bec27_theme_state_theme_user_id_state_userid_a03891c3"),
																			OriginalSourcePath:  new("partials/child.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
																			Stringability:       1,
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   37,
							Column: 3,
						},
						TagName: "footer",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_outer_layout_643254a4"),
							OriginalSourcePath:   new("partials/outer_layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   37,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/outer_layout.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   37,
									Column: 11,
								},
								TextContent: "Outer Layout Footer",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_outer_layout_643254a4"),
									OriginalSourcePath:   new("partials/outer_layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   37,
										Column: 11,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/outer_layout.pk"),
										Stringability:      1,
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
