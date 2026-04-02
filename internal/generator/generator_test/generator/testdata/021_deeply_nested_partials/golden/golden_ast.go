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
					OriginalPackageAlias: new("partials_level1_layout_21bc9d4e"),
					OriginalSourcePath:   new("partials/level1_layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "page_layout_username_state_username_a8648d1a",
						PartialAlias:        "page_layout",
						PartialPackageName:  "partials_level1_layout_21bc9d4e",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"username": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Username",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Username",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   39,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Username",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Username",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 45,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   39,
												Column: 23,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Username",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   39,
													Column: 23,
												},
											},
											BaseCodeGenVarName: new("pageData"),
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 45,
								},
								GoFieldName: "Username",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Username",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 45,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   39,
											Column: 23,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Username",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 45,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   39,
												Column: 23,
											},
										},
										BaseCodeGenVarName: new("pageData"),
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"username": "pages_main_594861c5",
					},
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
						OriginalSourcePath: new("partials/level1_layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "layout",
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
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "username",
						RawExpression: "state.Username",
						Expression: &ast_domain.MemberExpression{
							Base: &ast_domain.Identifier{
								Name: "state",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 45,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
									},
									BaseCodeGenVarName: new("pageData"),
									OriginalSourcePath: new("pages/main.pk"),
								},
							},
							Property: &ast_domain.Identifier{
								Name: "Username",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Username",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 45,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   39,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Username",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 45,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   39,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						Location: ast_domain.Location{
							Line:   22,
							Column: 45,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 34,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "Username",
								ReferenceLocation: ast_domain.Location{
									Line:   22,
									Column: 45,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   39,
									Column: 23,
								},
							},
							BaseCodeGenVarName:  new("pageData"),
							OriginalSourcePath:  new("pages/main.pk"),
							GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
							Stringability:       1,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						TagName: "header",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_level2_header_beeda5cc"),
							OriginalSourcePath:   new("partials/level2_header.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066",
								PartialAlias:        "page_header",
								PartialPackageName:  "partials_level2_header_beeda5cc",
								InvokerPackageAlias: "partials_level1_layout_21bc9d4e",
								Location: ast_domain.Location{
									Line:   23,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"username": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_level1_layout_21bc9d4e.Props"),
														PackageAlias:         "partials_level1_layout_21bc9d4e",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_page_layout_username_state_username_a8648d1a"),
													OriginalSourcePath: new("partials/level1_layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Username",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level1_layout_21bc9d4e",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Username",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_level1_layout_21bc9d4e",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_page_layout_username_state_username_a8648d1a"),
													},
													BaseCodeGenVarName:  new("props_page_layout_username_state_username_a8648d1a"),
													OriginalSourcePath:  new("partials/level1_layout.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level1_layout_21bc9d4e/generated.go"),
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
													PackageAlias:         "partials_level1_layout_21bc9d4e",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Username",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level1_layout_21bc9d4e",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Username",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("props_page_layout_username_state_username_a8648d1a"),
												},
												BaseCodeGenVarName:  new("props_page_layout_username_state_username_a8648d1a"),
												OriginalSourcePath:  new("partials/level1_layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_level1_layout_21bc9d4e/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   23,
											Column: 47,
										},
										GoFieldName: "Username",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_level1_layout_21bc9d4e",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Username",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_level1_layout_21bc9d4e",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Username",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("props_page_layout_username_state_username_a8648d1a"),
											},
											BaseCodeGenVarName:  new("props_page_layout_username_state_username_a8648d1a"),
											OriginalSourcePath:  new("partials/level1_layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_level1_layout_21bc9d4e/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"username": "partials_level1_layout_21bc9d4e",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("partials/level2_header.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "site-header",
								Location: ast_domain.Location{
									Line:   22,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "username",
								RawExpression: "props.Username",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_level1_layout_21bc9d4e.Props"),
												PackageAlias:         "partials_level1_layout_21bc9d4e",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props_page_layout_username_state_username_a8648d1a"),
											OriginalSourcePath: new("partials/level1_layout.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Username",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_level1_layout_21bc9d4e",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Username",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Username",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("props_page_layout_username_state_username_a8648d1a"),
											OriginalSourcePath:  new("partials/level1_layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_level1_layout_21bc9d4e/generated.go"),
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
											PackageAlias:         "partials_level1_layout_21bc9d4e",
											CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Username",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 47,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   43,
												Column: 2,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Username",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   39,
													Column: 23,
												},
											},
											BaseCodeGenVarName: new("pageData"),
										},
										BaseCodeGenVarName:  new("props_page_layout_username_state_username_a8648d1a"),
										OriginalSourcePath:  new("partials/level1_layout.pk"),
										GeneratedSourcePath: new("dist/partials/partials_level1_layout_21bc9d4e/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   23,
									Column: 47,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 36,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_level1_layout_21bc9d4e",
										CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level1_layout_21bc9d4e",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Username",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 47,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   43,
											Column: 2,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Username",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 45,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   39,
												Column: 23,
											},
										},
										BaseCodeGenVarName: new("pageData"),
									},
									BaseCodeGenVarName:  new("props_page_layout_username_state_username_a8648d1a"),
									OriginalSourcePath:  new("partials/level1_layout.pk"),
									GeneratedSourcePath: new("dist/partials/partials_level1_layout_21bc9d4e/generated.go"),
									Stringability:       1,
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
									OriginalPackageAlias: new("partials_level2_header_beeda5cc"),
									OriginalSourcePath:   new("partials/level2_header.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   23,
										Column: 31,
									},
									NameLocation: ast_domain.Location{
										Line:   23,
										Column: 23,
									},
									RawExpression: "state.SiteName",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("partials_level2_header_beeda5cc.Response"),
													PackageAlias:         "partials_level2_header_beeda5cc",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 31,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_level2_header_beeda5ccData_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
												OriginalSourcePath: new("partials/level2_header.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "SiteName",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_level2_header_beeda5cc",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "SiteName",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 31,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_level2_header_beeda5ccData_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
												OriginalSourcePath:  new("partials/level2_header.pk"),
												GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
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
												PackageAlias:         "partials_level2_header_beeda5cc",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "SiteName",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 31,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   40,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_level2_header_beeda5ccData_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
											OriginalSourcePath:  new("partials/level2_header.pk"),
											GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_level2_header_beeda5cc",
											CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "SiteName",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 31,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   40,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("partials_level2_header_beeda5ccData_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
										OriginalSourcePath:  new("partials/level2_header.pk"),
										GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										OriginalSourcePath: new("partials/level2_header.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "logo",
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
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 3,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_level3_profile_9f247195"),
									OriginalSourcePath:   new("partials/level3_profile.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a",
										PartialAlias:        "user_profile",
										PartialPackageName:  "partials_level3_profile_9f247195",
										InvokerPackageAlias: "partials_level2_header_beeda5cc",
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"username": ast_domain.PropValue{
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_level2_header_beeda5cc.Props"),
																PackageAlias:         "partials_level2_header_beeda5cc",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
															OriginalSourcePath: new("partials/level2_header.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Username",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_level2_header_beeda5cc",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level2_header_beeda5cc",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 48,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
															},
															BaseCodeGenVarName:  new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
															OriginalSourcePath:  new("partials/level2_header.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
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
															PackageAlias:         "partials_level2_header_beeda5cc",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 48,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_level2_header_beeda5cc",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
														},
														BaseCodeGenVarName:  new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
														OriginalSourcePath:  new("partials/level2_header.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   24,
													Column: 48,
												},
												GoFieldName: "Username",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level2_header_beeda5cc",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Username",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_level2_header_beeda5cc",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 48,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
													},
													BaseCodeGenVarName:  new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
													OriginalSourcePath:  new("partials/level2_header.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"username": "partials_level2_header_beeda5cc",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
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
										OriginalSourcePath: new("partials/level3_profile.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "user-profile",
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
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "username",
										RawExpression: "props.Username",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_level2_header_beeda5cc.Props"),
														PackageAlias:         "partials_level2_header_beeda5cc",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
													OriginalSourcePath: new("partials/level2_header.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Username",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level2_header_beeda5cc",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Username",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
													OriginalSourcePath:  new("partials/level2_header.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
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
													PackageAlias:         "partials_level2_header_beeda5cc",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Username",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 48,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Username",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
												OriginalSourcePath:  new("partials/level2_header.pk"),
												GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 48,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 37,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_level2_header_beeda5cc",
												CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level2_header_beeda5cc",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Username",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 48,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Username",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("props_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066"),
											OriginalSourcePath:  new("partials/level2_header.pk"),
											GeneratedSourcePath: new("dist/partials/partials_level2_header_beeda5cc/generated.go"),
											Stringability:       1,
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
											OriginalPackageAlias: new("partials_level3_profile_9f247195"),
											OriginalSourcePath:   new("partials/level3_profile.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
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
												OriginalSourcePath: new("partials/level3_profile.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   23,
													Column: 11,
												},
												TextContent: "Welcome, ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_level3_profile_9f247195"),
													OriginalSourcePath:   new("partials/level3_profile.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 11,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/level3_profile.pk"),
														Stringability:      1,
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   23,
													Column: 20,
												},
												TagName: "b",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_level3_profile_9f247195"),
													OriginalSourcePath:   new("partials/level3_profile.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   23,
														Column: 31,
													},
													NameLocation: ast_domain.Location{
														Line:   23,
														Column: 23,
													},
													RawExpression: "props.Username",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "props",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_level3_profile_9f247195.Props"),
																	PackageAlias:         "partials_level3_profile_9f247195",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																OriginalSourcePath: new("partials/level3_profile.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Username",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level3_profile_9f247195",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Username",
																		ReferenceLocation: ast_domain.Location{
																			Line:   22,
																			Column: 45,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   39,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																},
																BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																OriginalSourcePath:  new("partials/level3_profile.pk"),
																GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
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
																PackageAlias:         "partials_level3_profile_9f247195",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   39,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
															OriginalSourcePath:  new("partials/level3_profile.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_level3_profile_9f247195",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 31,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
														OriginalSourcePath:  new("partials/level3_profile.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:1",
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 20,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/level3_profile.pk"),
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
											OriginalPackageAlias: new("partials_level4_avatar_dd5bb04f"),
											OriginalSourcePath:   new("partials/level4_avatar.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf",
												PartialAlias:        "user_avatar",
												PartialPackageName:  "partials_level4_avatar_dd5bb04f",
												InvokerPackageAlias: "partials_level3_profile_9f247195",
												Location: ast_domain.Location{
													Line:   24,
													Column: 5,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"username": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_level3_profile_9f247195.Props"),
																		PackageAlias:         "partials_level3_profile_9f247195",
																		CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																	OriginalSourcePath: new("partials/level3_profile.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Username",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_level3_profile_9f247195",
																		CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Username",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_level3_profile_9f247195",
																			CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Username",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   38,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																	},
																	BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																	OriginalSourcePath:  new("partials/level3_profile.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
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
																	PackageAlias:         "partials_level3_profile_9f247195",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_level3_profile_9f247195",
																		CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Username",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																},
																BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
																OriginalSourcePath:  new("partials/level3_profile.pk"),
																GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   24,
															Column: 47,
														},
														GoFieldName: "Username",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_level3_profile_9f247195",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level3_profile_9f247195",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
															},
															BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
															OriginalSourcePath:  new("partials/level3_profile.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											DynamicAttributeOrigins: map[string]string{
												"title":    "partials_level4_avatar_dd5bb04f",
												"username": "partials_level3_profile_9f247195",
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:1",
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
												OriginalSourcePath: new("partials/level4_avatar.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "avatar",
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
										DynamicAttributes: []ast_domain.DynamicAttribute{
											ast_domain.DynamicAttribute{
												Name:          "title",
												RawExpression: "props.Username + ' avatar'",
												Expression: &ast_domain.BinaryExpression{
													Left: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "props",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_level4_avatar_dd5bb04f.Props"),
																	PackageAlias:         "partials_level4_avatar_dd5bb04f",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
																OriginalSourcePath: new("partials/level4_avatar.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Username",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level4_avatar_dd5bb04f",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Username",
																		ReferenceLocation: ast_domain.Location{
																			Line:   22,
																			Column: 45,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   39,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																},
																BaseCodeGenVarName:  new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
																OriginalSourcePath:  new("partials/level4_avatar.pk"),
																GeneratedSourcePath: new("dist/partials/partials_level4_avatar_dd5bb04f/generated.go"),
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
																PackageAlias:         "partials_level4_avatar_dd5bb04f",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   39,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
															OriginalSourcePath:  new("partials/level4_avatar.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level4_avatar_dd5bb04f/generated.go"),
															Stringability:       1,
														},
													},
													Operator: "+",
													Right: &ast_domain.StringLiteral{
														Value: " avatar",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 18,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("partials/level4_avatar.pk"),
															Stringability:      1,
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/level4_avatar.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   22,
													Column: 31,
												},
												NameLocation: ast_domain.Location{
													Line:   22,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("partials/level4_avatar.pk"),
													Stringability:      1,
												},
											},
											ast_domain.DynamicAttribute{
												Name:          "username",
												RawExpression: "props.Username",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_level3_profile_9f247195.Props"),
																PackageAlias:         "partials_level3_profile_9f247195",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
															OriginalSourcePath: new("partials/level3_profile.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Username",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_level3_profile_9f247195",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   39,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
															OriginalSourcePath:  new("partials/level3_profile.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
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
															PackageAlias:         "partials_level3_profile_9f247195",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
														OriginalSourcePath:  new("partials/level3_profile.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   24,
													Column: 47,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 36,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level3_profile_9f247195",
														CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level3_profile_9f247195",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Username",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a"),
													OriginalSourcePath:  new("partials/level3_profile.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level3_profile_9f247195/generated.go"),
													Stringability:       1,
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
													OriginalPackageAlias: new("partials_level4_avatar_dd5bb04f"),
													OriginalSourcePath:   new("partials/level4_avatar.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													NameLocation: ast_domain.Location{
														Line:   23,
														Column: 11,
													},
													RawExpression: "props.Username",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "props",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_level4_avatar_dd5bb04f.Props"),
																	PackageAlias:         "partials_level4_avatar_dd5bb04f",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
																OriginalSourcePath: new("partials/level4_avatar.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Username",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level4_avatar_dd5bb04f",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Username",
																		ReferenceLocation: ast_domain.Location{
																			Line:   22,
																			Column: 45,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   39,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																},
																BaseCodeGenVarName:  new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
																OriginalSourcePath:  new("partials/level4_avatar.pk"),
																GeneratedSourcePath: new("dist/partials/partials_level4_avatar_dd5bb04f/generated.go"),
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
																PackageAlias:         "partials_level4_avatar_dd5bb04f",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Username",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   39,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
															OriginalSourcePath:  new("partials/level4_avatar.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level4_avatar_dd5bb04f/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_level4_avatar_dd5bb04f",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level4_avatar_dd5bb04f",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Username",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Username",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf"),
														OriginalSourcePath:  new("partials/level4_avatar.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level4_avatar_dd5bb04f/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:1:0",
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
														OriginalSourcePath: new("partials/level4_avatar.pk"),
														Stringability:      1,
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   22,
													Column: 3,
												},
												TagName: "sub",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_level5_subtext_a77f881b"),
													OriginalSourcePath:   new("partials/level5_subtext.pk"),
													PartialInfo: &ast_domain.PartialInvocationInfo{
														InvocationKey:       "subtext_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf_b7aff591",
														PartialAlias:        "subtext",
														PartialPackageName:  "partials_level5_subtext_a77f881b",
														InvokerPackageAlias: "partials_level4_avatar_dd5bb04f",
														Location: ast_domain.Location{
															Line:   24,
															Column: 5,
														},
													},
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   22,
														Column: 16,
													},
													NameLocation: ast_domain.Location{
														Line:   22,
														Column: 8,
													},
													RawExpression: "state.Info",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_level5_subtext_a77f881b.Response"),
																	PackageAlias:         "partials_level5_subtext_a77f881b",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level5_subtext_a77f881b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_level5_subtext_a77f881bData_subtext_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf_b7aff591"),
																OriginalSourcePath: new("partials/level5_subtext.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Info",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level5_subtext_a77f881b",
																	CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level5_subtext_a77f881b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Info",
																	ReferenceLocation: ast_domain.Location{
																		Line:   22,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   29,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("partials_level5_subtext_a77f881bData_subtext_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf_b7aff591"),
																OriginalSourcePath:  new("partials/level5_subtext.pk"),
																GeneratedSourcePath: new("dist/partials/partials_level5_subtext_a77f881b/generated.go"),
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
																PackageAlias:         "partials_level5_subtext_a77f881b",
																CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level5_subtext_a77f881b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Info",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   29,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_level5_subtext_a77f881bData_subtext_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf_b7aff591"),
															OriginalSourcePath:  new("partials/level5_subtext.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level5_subtext_a77f881b/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_level5_subtext_a77f881b",
															CanonicalPackagePath: "testcase_021_deeply_nested_partials/dist/partials/partials_level5_subtext_a77f881b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Info",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_level5_subtext_a77f881bData_subtext_user_avatar_user_profile_page_header_page_layout_username_state_username_a8648d1a_username_props_username_b62f9066_username_props_username_9e3f6b2a_username_props_username_e00bcdbf_b7aff591"),
														OriginalSourcePath:  new("partials/level5_subtext.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level5_subtext_a77f881b/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:1:1",
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
														OriginalSourcePath: new("partials/level5_subtext.pk"),
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   24,
							Column: 5,
						},
						TagName: "main",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_level1_layout_21bc9d4e"),
							OriginalSourcePath:   new("partials/level1_layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("partials/level1_layout.pk"),
								Stringability:      1,
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
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "page-specific-content",
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
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 11,
												},
												TextContent: "Page Title",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 11,
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
											Line:   25,
											Column: 7,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 7,
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
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   25,
													Column: 10,
												},
												TextContent: "This is the content unique to the main page.",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 10,
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
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   27,
							Column: 5,
						},
						TagName: "footer",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_level1_layout_21bc9d4e"),
							OriginalSourcePath:   new("partials/level1_layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   27,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/level1_layout.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   28,
									Column: 7,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_level1_layout_21bc9d4e"),
									OriginalSourcePath:   new("partials/level1_layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   28,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/level1_layout.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   28,
											Column: 10,
										},
										TextContent: "Site Footer",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_level1_layout_21bc9d4e"),
											OriginalSourcePath:   new("partials/level1_layout.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   28,
												Column: 10,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/level1_layout.pk"),
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
	}
}()
