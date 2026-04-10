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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   23,
									Column: 9,
								},
								TextContent: "Categories",
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
										Column: 9,
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
							Line:   24,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   24,
								Column: 17,
							},
							NameLocation: ast_domain.Location{
								Line:   24,
								Column: 10,
							},
							RawExpression: "cat in state.Categories",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: nil,
								ItemVariable: &ast_domain.Identifier{
									Name: "cat",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "cat",
											ReferenceLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("cat"),
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Collection: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 17,
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
										Name: "Categories",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 14,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]pages_main_594861c5.Category"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Categories",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   41,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Optional: false,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 8,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]pages_main_594861c5.Category"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Categories",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   41,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]pages_main_594861c5.Category"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Categories",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   41,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]pages_main_594861c5.Category"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Categories",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   41,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]pages_main_594861c5.Category"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Categories",
									ReferenceLocation: ast_domain.Location{
										Line:   24,
										Column: 17,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   41,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
							},
						},
						DirKey: &ast_domain.Directive{
							Type: ast_domain.DirectiveKey,
							Location: ast_domain.Location{
								Line:   24,
								Column: 49,
							},
							NameLocation: ast_domain.Location{
								Line:   24,
								Column: 42,
							},
							RawExpression: "cat.Name",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "cat",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "cat",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 5,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("cat"),
										OriginalSourcePath: new("partials/item_row.pk"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "Name",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Name",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 49,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   40,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("cat"),
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
										CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Name",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("cat"),
									OriginalSourcePath:  new("partials/item_row.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Name",
									ReferenceLocation: ast_domain.Location{
										Line:   24,
										Column: 49,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   40,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("cat"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						Key: &ast_domain.TemplateLiteral{
							Parts: []ast_domain.TemplateLiteralPart{
								ast_domain.TemplateLiteralPart{
									IsLiteral: true,
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
									Literal: "r.0:1.",
								},
								ast_domain.TemplateLiteralPart{
									IsLiteral: false,
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
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "cat",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "cat",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 5,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("cat"),
												OriginalSourcePath: new("partials/item_row.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Name",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 5,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 49,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("cat"),
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
												CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 5,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("cat"),
											OriginalSourcePath:  new("partials/item_row.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   25,
										Column: 19,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 11,
									},
									RawExpression: "cat.Name",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "cat",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "cat",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("cat"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Name",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 5,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("cat"),
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
												CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   40,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("cat"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Name",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 19,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   40,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("cat"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
											Literal: "r.0:1.",
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: false,
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
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "cat",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "cat",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 5,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("cat"),
														OriginalSourcePath: new("partials/item_row.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 49,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("cat"),
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
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 5,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("cat"),
													OriginalSourcePath:  new("partials/item_row.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
											Literal: ":0",
										},
									},
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
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 3,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_category_list_3eee8006"),
									OriginalSourcePath:   new("partials/category_list.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "category_list_category_name_cat_name_9264de01",
										PartialAlias:        "category_list",
										PartialPackageName:  "partials_category_list_3eee8006",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   26,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"category_name": ast_domain.PropValue{
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "cat",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "cat",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 56,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("cat"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Name",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 5,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 56,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   40,
																	Column: 23,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 56,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   40,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("cat"),
															},
															BaseCodeGenVarName:  new("cat"),
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
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 56,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 23,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 56,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   40,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("cat"),
														},
														BaseCodeGenVarName:  new("cat"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   26,
													Column: 56,
												},
												GoFieldName: "CategoryName",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 56,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 56,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("cat"),
													},
													BaseCodeGenVarName:  new("cat"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											IsLoopDependent: true,
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"category_name": "pages_main_594861c5",
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
												OriginalSourcePath: new("partials/category_list.pk"),
												Stringability:      1,
											},
											Literal: "r.0:1.",
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: false,
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
												OriginalSourcePath: new("partials/category_list.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "cat",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "cat",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 5,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("cat"),
														OriginalSourcePath: new("partials/item_row.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 49,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("cat"),
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
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 5,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("cat"),
													OriginalSourcePath:  new("partials/item_row.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
												OriginalSourcePath: new("partials/category_list.pk"),
												Stringability:      1,
											},
											Literal: ":1",
										},
									},
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
										OriginalSourcePath: new("partials/category_list.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "category-list",
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
										Name:          "category_name",
										RawExpression: "cat.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "cat",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "cat",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 56,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("cat"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 56,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("cat"),
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
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 56,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("cat"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   26,
											Column: 56,
										},
										NameLocation: ast_domain.Location{
											Line:   26,
											Column: 40,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 56,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   40,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("cat"),
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
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_row_8e7dec6a"),
											OriginalSourcePath:   new("partials/item_row.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1",
												PartialAlias:        "item_row",
												PartialPackageName:  "partials_item_row_8e7dec6a",
												InvokerPackageAlias: "partials_category_list_3eee8006",
												Location: ast_domain.Location{
													Line:   23,
													Column: 5,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"detail": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/category_list.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Extra",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*partials_category_list_3eee8006.Extra"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Extra",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   43,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("partials/category_list.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
																		TypeExpression:       typeExprFromString("*partials_category_list_3eee8006.Extra"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Extra",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   43,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/category_list.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Detail",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 13,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Detail",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   40,
																			Column: 20,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Detail",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   40,
																				Column: 20,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/category_list.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: true,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   40,
																		Column: 20,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Detail",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   40,
																			Column: 20,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("partials/category_list.pk"),
																GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   26,
															Column: 16,
														},
														GoFieldName: "Detail",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Detail",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   40,
																	Column: 20,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   40,
																		Column: 20,
																	},
																},
																BaseCodeGenVarName: new("item"),
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/category_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
															Stringability:       1,
														},
													},
													"label": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																	OriginalSourcePath: new("partials/category_list.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Label",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 6,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   42,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   25,
																				Column: 15,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   42,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/category_list.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 15,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   42,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("partials/category_list.pk"),
																GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   25,
															Column: 15,
														},
														GoFieldName: "Label",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 15,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 15,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("item"),
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/category_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
															Stringability:       1,
														},
													},
													IsLoopDependent: true,
													IsLoopDependent: true,
												},
											},
											DynamicAttributeOrigins: map[string]string{
												"detail": "partials_category_list_3eee8006",
												"label":  "partials_category_list_3eee8006",
											},
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   24,
												Column: 14,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 7,
											},
											RawExpression: "item in state.Items",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: nil,
												ItemVariable: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
															PackageAlias:         "partials_category_list_3eee8006",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("partials/item_row.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Response"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
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
															BaseCodeGenVarName: new("partials_category_list_3eee8006Data_category_list_category_name_cat_name_9264de01"),
															OriginalSourcePath: new("partials/category_list.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Items",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]partials_category_list_3eee8006.Item"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Items",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 14,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   48,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_category_list_3eee8006Data_category_list_category_name_cat_name_9264de01"),
															OriginalSourcePath:  new("partials/category_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]partials_category_list_3eee8006.Item"),
															PackageAlias:         "partials_category_list_3eee8006",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Items",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 14,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   48,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_category_list_3eee8006Data_category_list_category_name_cat_name_9264de01"),
														OriginalSourcePath:  new("partials/category_list.pk"),
														GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
													},
												},
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]partials_category_list_3eee8006.Item"),
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 14,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   48,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_category_list_3eee8006Data_category_list_category_name_cat_name_9264de01"),
													OriginalSourcePath:  new("partials/category_list.pk"),
													GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]partials_category_list_3eee8006.Item"),
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 14,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   48,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_category_list_3eee8006Data_category_list_category_name_cat_name_9264de01"),
													OriginalSourcePath:  new("partials/category_list.pk"),
													GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]partials_category_list_3eee8006.Item"),
													PackageAlias:         "partials_category_list_3eee8006",
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 14,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_category_list_3eee8006Data_category_list_category_name_cat_name_9264de01"),
												OriginalSourcePath:  new("partials/category_list.pk"),
												GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   24,
												Column: 42,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 35,
											},
											RawExpression: "item.Label",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
															PackageAlias:         "partials_category_list_3eee8006",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 5,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("partials/item_row.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Label",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_category_list_3eee8006",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 42,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   42,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("partials/category_list.pk"),
														GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 5,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("partials/item_row.pk"),
													GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_category_list_3eee8006",
													CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 42,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   42,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("partials/category_list.pk"),
												GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
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
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
													Literal: "r.0:1.",
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: false,
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
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "cat",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "cat",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 5,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("cat"),
																OriginalSourcePath: new("partials/item_row.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Name",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 5,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 49,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   40,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("cat"),
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
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 5,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("cat"),
															OriginalSourcePath:  new("partials/item_row.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
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
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
													Literal: ":1:0.",
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: false,
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
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 5,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_row.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Label",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("partials/category_list.pk"),
																GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 5,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_row.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
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
												OriginalSourcePath: new("partials/item_row.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-row",
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
												Name:          "detail",
												RawExpression: "item.Extra?.Detail",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/category_list.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Extra",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*partials_category_list_3eee8006.Extra"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Extra",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   43,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("partials/category_list.pk"),
																GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
																TypeExpression:       typeExprFromString("*partials_category_list_3eee8006.Extra"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Extra",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/category_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Detail",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 13,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Detail",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   40,
																	Column: 20,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/category_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
															Stringability:       1,
														},
													},
													Optional: true,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_category_list_3eee8006",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Detail",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 20,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("partials/category_list.pk"),
														GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   26,
													Column: 16,
												},
												NameLocation: ast_domain.Location{
													Line:   26,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Detail",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 20,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("partials/category_list.pk"),
													GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
													Stringability:       1,
												},
											},
											ast_domain.DynamicAttribute{
												Name:          "label",
												RawExpression: "item.Label",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 15,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
															OriginalSourcePath: new("partials/category_list.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Label",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 6,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 15,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/category_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
															PackageAlias:         "partials_category_list_3eee8006",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 15,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   42,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("partials/category_list.pk"),
														GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   25,
													Column: 15,
												},
												NameLocation: ast_domain.Location{
													Line:   25,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 15,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   42,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("partials/category_list.pk"),
													GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
													OriginalPackageAlias: new("partials_item_row_8e7dec6a"),
													OriginalSourcePath:   new("partials/item_row.pk"),
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
													RawExpression: "state.Label",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_item_row_8e7dec6a.Response"),
																	PackageAlias:         "partials_item_row_8e7dec6a",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																OriginalSourcePath: new("partials/item_row.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Label",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_item_row_8e7dec6a",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   37,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																OriginalSourcePath:  new("partials/item_row.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
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
																PackageAlias:         "partials_item_row_8e7dec6a",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
															OriginalSourcePath:  new("partials/item_row.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_item_row_8e7dec6a",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
														OriginalSourcePath:  new("partials/item_row.pk"),
														GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "cat",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cat",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 5,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cat"),
																		OriginalSourcePath: new("partials/item_row.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Name",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 5,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Name",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 49,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   40,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("cat"),
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
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 5,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   37,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("cat"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Literal: ":1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 5,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_row.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Label",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 42,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   42,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("partials/category_list.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 5,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   39,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Literal: ":0",
														},
													},
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
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   24,
													Column: 5,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_row_8e7dec6a"),
													OriginalSourcePath:   new("partials/item_row.pk"),
												},
												DirIf: &ast_domain.Directive{
													Type: ast_domain.DirectiveIf,
													Location: ast_domain.Location{
														Line:   24,
														Column: 54,
													},
													NameLocation: ast_domain.Location{
														Line:   24,
														Column: 48,
													},
													RawExpression: "state.Detail != ``",
													Expression: &ast_domain.BinaryExpression{
														Left: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_item_row_8e7dec6a.Response"),
																		PackageAlias:         "partials_item_row_8e7dec6a",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																	OriginalSourcePath: new("partials/item_row.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Detail",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_row_8e7dec6a",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Detail",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
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
																	PackageAlias:         "partials_item_row_8e7dec6a",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																OriginalSourcePath:  new("partials/item_row.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
																Stringability:       1,
															},
														},
														Operator: "!=",
														Right: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{},
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 17,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("partials/item_row.pk"),
															Stringability:      1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   24,
														Column: 34,
													},
													NameLocation: ast_domain.Location{
														Line:   24,
														Column: 26,
													},
													RawExpression: "state.Detail",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_item_row_8e7dec6a.Response"),
																	PackageAlias:         "partials_item_row_8e7dec6a",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																OriginalSourcePath: new("partials/item_row.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Detail",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_item_row_8e7dec6a",
																	CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
																OriginalSourcePath:  new("partials/item_row.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
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
																PackageAlias:         "partials_item_row_8e7dec6a",
																CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Detail",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
															OriginalSourcePath:  new("partials/item_row.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_item_row_8e7dec6a",
															CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Detail",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_item_row_8e7dec6aData_item_row_category_list_category_name_cat_name_9264de01_detail_item_extra_detail_label_item_label_99d11cd1"),
														OriginalSourcePath:  new("partials/item_row.pk"),
														GeneratedSourcePath: new("dist/partials/partials_item_row_8e7dec6a/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "cat",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Category"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cat",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 5,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cat"),
																		OriginalSourcePath: new("partials/item_row.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Name",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 5,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Name",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 49,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   40,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("cat"),
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
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 5,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   37,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("cat"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Literal: ":1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_category_list_3eee8006.Item"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 5,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_row.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Label",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 42,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   42,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("partials/category_list.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
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
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_142_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 5,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   39,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_category_list_3eee8006/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_row.pk"),
																Stringability:      1,
															},
															Literal: ":1",
														},
													},
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
														OriginalSourcePath: new("partials/item_row.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "detail",
														Location: ast_domain.Location{
															Line:   24,
															Column: 18,
														},
														NameLocation: ast_domain.Location{
															Line:   24,
															Column: 11,
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
