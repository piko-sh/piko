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
				NodeType: ast_domain.NodeFragment,
				Location: ast_domain.Location{
					Line:   0,
					Column: 0,
				},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "layout_page_title_state_title_3ee24b3e",
						PartialAlias:        "layout",
						PartialPackageName:  "partials_layout_ee037d9a",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"page_title": ast_domain.PropValue{
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
												CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 83,
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
										Name: "Title",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Title",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 83,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   61,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 83,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   61,
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
											CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Title",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 83,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   61,
												Column: 23,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Title",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 83,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   61,
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
									Column: 83,
								},
								GoFieldName: "PageTitle",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Title",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 83,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   61,
											Column: 23,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Title",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 83,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   61,
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
						"test2": "pages_main_594861c5",
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   0,
						Column: 0,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("partials/layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "test",
						Value: "hello",
						Location: ast_domain.Location{
							Line:   22,
							Column: 35,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 29,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "test2",
						RawExpression: "state.Title",
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
										CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 50,
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
								Name: "Title",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Title",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 50,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   61,
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
									CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Title",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 50,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   61,
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
							Column: 50,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 42,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "Title",
								ReferenceLocation: ast_domain.Location{
									Line:   22,
									Column: 50,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   61,
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
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
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
								OriginalSourcePath: new("partials/layout.pk"),
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
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "layout_server_page_title_state_title_test_hello_test2_state_title_4ae44b41",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "0",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
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
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
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
										OriginalSourcePath: new("partials/layout.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "header-container",
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
										TagName: "a",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_layout_ee037d9a"),
											OriginalSourcePath:   new("partials/layout.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
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
												OriginalSourcePath: new("partials/layout.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "href",
												Value: "/",
												Location: ast_domain.Location{
													Line:   24,
													Column: 16,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 10,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "brand",
												Location: ast_domain.Location{
													Line:   24,
													Column: 26,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 19,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 33,
												},
												TextContent: " 🚀 my-piko-app ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 33,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   27,
											Column: 7,
										},
										TagName: "nav",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_layout_ee037d9a"),
											OriginalSourcePath:   new("partials/layout.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   27,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/layout.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "main-nav",
												Location: ast_domain.Location{
													Line:   27,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   27,
													Column: 12,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   28,
													Column: 9,
												},
												TagName: "a",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   28,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "href",
														Value: "/",
														Location: ast_domain.Location{
															Line:   28,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   28,
															Column: 12,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   28,
															Column: 21,
														},
														TextContent: "Home",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:0:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 21,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   29,
													Column: 9,
												},
												TagName: "a",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:1:1",
													RelativeLocation: ast_domain.Location{
														Line:   29,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "href",
														Value: "#",
														Location: ast_domain.Location{
															Line:   29,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   29,
															Column: 12,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   29,
															Column: 21,
														},
														TextContent: "About",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:0:1:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   29,
																Column: 21,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   30,
													Column: 9,
												},
												TagName: "a",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:1:2",
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "href",
														Value: "#",
														Location: ast_domain.Location{
															Line:   30,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   30,
															Column: 12,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   30,
															Column: 21,
														},
														TextContent: "Contact",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:0:1:2:0",
															RelativeLocation: ast_domain.Location{
																Line:   30,
																Column: 21,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   35,
							Column: 3,
						},
						TagName: "main",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   35,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "layout_server_page_title_state_title_test_hello_test2_state_title_4ae44b41",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "1",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
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
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
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
										Value: "container",
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
										TagName: "h1",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
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
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
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
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   24,
															Column: 14,
														},
														RawExpression: "state.Title",
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
																		CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 14,
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
																Name: "Title",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 14,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   61,
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
																	CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 14,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_048_welcome_advanced/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 14,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
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
												TextContent: "Your Piko application is running!",
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
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   26,
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
											Value: "r.0:1:0:2",
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   26,
													Column: 10,
												},
												TextContent: "Edit ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:2:0",
													RelativeLocation: ast_domain.Location{
														Line:   26,
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
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   26,
													Column: 15,
												},
												TagName: "code",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:2:1",
													RelativeLocation: ast_domain.Location{
														Line:   26,
														Column: 15,
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
															Line:   0,
															Column: 0,
														},
														TextContent:        "pages/index.pk",
														PreserveWhitespace: true,
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("pages_main_594861c5"),
															OriginalSourcePath:   new("pages/main.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:2:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
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
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   26,
													Column: 42,
												},
												TextContent: " to get started.",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:2:2",
													RelativeLocation: ast_domain.Location{
														Line:   26,
														Column: 42,
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
							Line:   39,
							Column: 3,
						},
						TagName: "footer",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   39,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "site-footer",
								Location: ast_domain.Location{
									Line:   39,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 11,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "layout_server_page_title_state_title_test_hello_test2_state_title_4ae44b41",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "2",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   40,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   40,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/layout.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "footer-container",
										Location: ast_domain.Location{
											Line:   40,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   40,
											Column: 10,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   41,
											Column: 7,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_layout_ee037d9a"),
											OriginalSourcePath:   new("partials/layout.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/layout.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   41,
													Column: 10,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   41,
														Column: 10,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: true,
														Location: ast_domain.Location{
															Line:   41,
															Column: 10,
														},
														Literal: "© ",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("partials/layout.pk"),
														},
													},
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   41,
															Column: 15,
														},
														RawExpression: "state.CurrentYear",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_layout_ee037d9a.Response"),
																		PackageAlias:         "partials_layout_ee037d9a",
																		CanonicalPackagePath: "testcase_048_welcome_advanced/dist/partials/partials_layout_ee037d9a",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   41,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_layout_ee037d9aData_layout_page_title_state_title_3ee24b3e"),
																	OriginalSourcePath: new("partials/layout.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "CurrentYear",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "partials_layout_ee037d9a",
																		CanonicalPackagePath: "testcase_048_welcome_advanced/dist/partials/partials_layout_ee037d9a",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "CurrentYear",
																		ReferenceLocation: ast_domain.Location{
																			Line:   41,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   65,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_page_title_state_title_3ee24b3e"),
																	OriginalSourcePath:  new("partials/layout.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
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
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "partials_layout_ee037d9a",
																	CanonicalPackagePath: "testcase_048_welcome_advanced/dist/partials/partials_layout_ee037d9a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "CurrentYear",
																	ReferenceLocation: ast_domain.Location{
																		Line:   41,
																		Column: 15,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   65,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_page_title_state_title_3ee24b3e"),
																OriginalSourcePath:  new("partials/layout.pk"),
																GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "partials_layout_ee037d9a",
																CanonicalPackagePath: "testcase_048_welcome_advanced/dist/partials/partials_layout_ee037d9a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CurrentYear",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 15,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   65,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_page_title_state_title_3ee24b3e"),
															OriginalSourcePath:  new("partials/layout.pk"),
															GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
															Stringability:       1,
														},
													},
													ast_domain.TextPart{
														IsLiteral: true,
														Location: ast_domain.Location{
															Line:   41,
															Column: 35,
														},
														Literal: " my-piko-app. All Rights Reserved.",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("partials/layout.pk"),
														},
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   42,
											Column: 7,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_layout_ee037d9a"),
											OriginalSourcePath:   new("partials/layout.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   42,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/layout.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "social-links",
												Location: ast_domain.Location{
													Line:   42,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   42,
													Column: 12,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   43,
													Column: 9,
												},
												TagName: "a",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   43,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "href",
														Value: "#",
														Location: ast_domain.Location{
															Line:   43,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   43,
															Column: 12,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   43,
															Column: 21,
														},
														TextContent: "Twitter",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:2:0:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   43,
																Column: 21,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   44,
													Column: 9,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:0:1:1",
													RelativeLocation: ast_domain.Location{
														Line:   44,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   44,
															Column: 15,
														},
														TextContent: "·",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:2:0:1:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   44,
																Column: 15,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   45,
													Column: 9,
												},
												TagName: "a",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:0:1:2",
													RelativeLocation: ast_domain.Location{
														Line:   45,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "href",
														Value: "#",
														Location: ast_domain.Location{
															Line:   45,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   45,
															Column: 12,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   45,
															Column: 21,
														},
														TextContent: "GitHub",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:2:0:1:2:0",
															RelativeLocation: ast_domain.Location{
																Line:   45,
																Column: 21,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   46,
													Column: 9,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:0:1:3",
													RelativeLocation: ast_domain.Location{
														Line:   46,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   46,
															Column: 15,
														},
														TextContent: "·",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:2:0:1:3:0",
															RelativeLocation: ast_domain.Location{
																Line:   46,
																Column: 15,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   47,
													Column: 9,
												},
												TagName: "a",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_layout_ee037d9a"),
													OriginalSourcePath:   new("partials/layout.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:0:1:4",
													RelativeLocation: ast_domain.Location{
														Line:   47,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/layout.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "href",
														Value: "#",
														Location: ast_domain.Location{
															Line:   47,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   47,
															Column: 12,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   47,
															Column: 21,
														},
														TextContent: "LinkedIn",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_layout_ee037d9a"),
															OriginalSourcePath:   new("partials/layout.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:2:0:1:4:0",
															RelativeLocation: ast_domain.Location{
																Line:   47,
																Column: 21,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/layout.pk"),
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
	}
}()
