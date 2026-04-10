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
					Line:   48,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   48,
						Column: 2,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("main.pk"),
						Stringability:      1,
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   49,
							Column: 3,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   49,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   49,
									Column: 7,
								},
								TextContent: "Categories",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   49,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   50,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   50,
								Column: 15,
							},
							NameLocation: ast_domain.Location{
								Line:   50,
								Column: 8,
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
											TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
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
										OriginalSourcePath: new("main.pk"),
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
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   50,
													Column: 15,
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
										Name: "Categories",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 14,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]main_aaf9a2e0.Category"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Categories",
												ReferenceLocation: ast_domain.Location{
													Line:   50,
													Column: 15,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
											TypeExpression:       typeExprFromString("[]main_aaf9a2e0.Category"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Categories",
											ReferenceLocation: ast_domain.Location{
												Line:   50,
												Column: 15,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]main_aaf9a2e0.Category"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Categories",
										ReferenceLocation: ast_domain.Location{
											Line:   50,
											Column: 15,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]main_aaf9a2e0.Category"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Categories",
										ReferenceLocation: ast_domain.Location{
											Line:   50,
											Column: 15,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]main_aaf9a2e0.Category"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Categories",
									ReferenceLocation: ast_domain.Location{
										Line:   50,
										Column: 15,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   31,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
							},
						},
						DirKey: &ast_domain.Directive{
							Type: ast_domain.DirectiveKey,
							Location: ast_domain.Location{
								Line:   50,
								Column: 47,
							},
							NameLocation: ast_domain.Location{
								Line:   50,
								Column: 40,
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
											TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "cat",
											ReferenceLocation: ast_domain.Location{
												Line:   47,
												Column: 3,
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
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Name",
											ReferenceLocation: ast_domain.Location{
												Line:   50,
												Column: 47,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   30,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("cat"),
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
										CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Name",
										ReferenceLocation: ast_domain.Location{
											Line:   47,
											Column: 3,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   30,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("cat"),
									OriginalSourcePath:  new("partials/item_row.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Name",
									ReferenceLocation: ast_domain.Location{
										Line:   50,
										Column: 47,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   30,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("cat"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								Stringability:       1,
							},
						},
						Key: &ast_domain.TemplateLiteral{
							Parts: []ast_domain.TemplateLiteralPart{
								ast_domain.TemplateLiteralPart{
									IsLiteral: true,
									RelativeLocation: ast_domain.Location{
										Line:   50,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
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
										OriginalSourcePath: new("main.pk"),
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
													TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "cat",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 3,
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
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   50,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   30,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("cat"),
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
												CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 3,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("cat"),
											OriginalSourcePath:  new("partials/item_row.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
							RelativeLocation: ast_domain.Location{
								Line:   50,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   51,
									Column: 4,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   51,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   51,
										Column: 8,
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
													TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "cat",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("cat"),
												OriginalSourcePath: new("main.pk"),
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
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   30,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("cat"),
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
												CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("cat"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Name",
											ReferenceLocation: ast_domain.Location{
												Line:   51,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   30,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("cat"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   51,
												Column: 4,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("main.pk"),
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
												OriginalSourcePath: new("main.pk"),
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
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "cat",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 3,
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
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("cat"),
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
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 3,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("cat"),
													OriginalSourcePath:  new("partials/item_row.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   51,
												Column: 4,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
											Literal: ":0",
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   51,
										Column: 4,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   57,
									Column: 2,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_category_list_3eee8006"),
									OriginalSourcePath:   new("partials/category_list.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "category_list_category_name_cat_name_9264de01",
										PartialAlias:        "category_list",
										PartialPackageName:  "partials_category_list_3eee8006",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   52,
											Column: 4,
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
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "cat",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("cat"),
															OriginalSourcePath: new("main.pk"),
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
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 23,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   52,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("cat"),
															},
															BaseCodeGenVarName:  new("cat"),
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 23,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("cat"),
														},
														BaseCodeGenVarName:  new("cat"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   52,
													Column: 53,
												},
												GoFieldName: "CategoryName",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("cat"),
													},
													BaseCodeGenVarName:  new("cat"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
											IsLoopDependent: true,
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"category_name": "main_aaf9a2e0",
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   57,
												Column: 2,
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
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "cat",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 3,
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
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("cat"),
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
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 3,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("cat"),
													OriginalSourcePath:  new("partials/item_row.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   57,
												Column: 2,
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
										Line:   57,
										Column: 2,
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
											Line:   57,
											Column: 14,
										},
										NameLocation: ast_domain.Location{
											Line:   57,
											Column: 7,
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
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "cat",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("cat"),
													OriginalSourcePath: new("main.pk"),
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
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("cat"),
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
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   52,
														Column: 53,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   30,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("cat"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   52,
											Column: 53,
										},
										NameLocation: ast_domain.Location{
											Line:   52,
											Column: 37,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   52,
													Column: 53,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("cat"),
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
											Line:   45,
											Column: 2,
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
													Line:   58,
													Column: 3,
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
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   61,
																				Column: 13,
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
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Extra",
																			ReferenceLocation: ast_domain.Location{
																				Line:   61,
																				Column: 13,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   33,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Extra",
																		ReferenceLocation: ast_domain.Location{
																			Line:   61,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   33,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Detail",
																		ReferenceLocation: ast_domain.Location{
																			Line:   61,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 20,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Detail",
																			ReferenceLocation: ast_domain.Location{
																				Line:   61,
																				Column: 13,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   61,
																		Column: 13,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
																		Column: 20,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Detail",
																		ReferenceLocation: ast_domain.Location{
																			Line:   61,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
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
															Line:   61,
															Column: 13,
														},
														GoFieldName: "Detail",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Detail",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 20,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   61,
																		Column: 13,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   60,
																			Column: 12,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   60,
																			Column: 12,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   32,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_category_list_3eee8006",
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   60,
																				Column: 12,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   32,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   60,
																		Column: 12,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   32,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_category_list_3eee8006",
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   60,
																			Column: 12,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   32,
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
															Line:   60,
															Column: 12,
														},
														GoFieldName: "Label",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_category_list_3eee8006",
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   60,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_category_list_3eee8006",
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   60,
																		Column: 12,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   32,
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
												Line:   59,
												Column: 11,
											},
											NameLocation: ast_domain.Location{
												Line:   59,
												Column: 4,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 11,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Items",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 11,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Items",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 11,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
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
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 11,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
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
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 11,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
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
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 11,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
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
												Line:   59,
												Column: 39,
											},
											NameLocation: ast_domain.Location{
												Line:   59,
												Column: 32,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 3,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 39,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 3,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 39,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
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
														Line:   45,
														Column: 2,
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
																	TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "cat",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 3,
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
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   50,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("cat"),
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 3,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("cat"),
															OriginalSourcePath:  new("partials/item_row.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   45,
														Column: 2,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 3,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   59,
																		Column: 39,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   32,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 3,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
												Line:   45,
												Column: 2,
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
													Line:   45,
													Column: 14,
												},
												NameLocation: ast_domain.Location{
													Line:   45,
													Column: 7,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   61,
																		Column: 13,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Extra",
																	ReferenceLocation: ast_domain.Location{
																		Line:   61,
																		Column: 13,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   33,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Extra",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   33,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Detail",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Detail",
															ReferenceLocation: ast_domain.Location{
																Line:   61,
																Column: 13,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
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
													Line:   61,
													Column: 13,
												},
												NameLocation: ast_domain.Location{
													Line:   61,
													Column: 4,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Detail",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   60,
																	Column: 12,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   60,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   60,
																Column: 12,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
													Line:   60,
													Column: 12,
												},
												NameLocation: ast_domain.Location{
													Line:   60,
													Column: 4,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_category_list_3eee8006",
														CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   60,
															Column: 12,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													Line:   46,
													Column: 3,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_row_8e7dec6a"),
													OriginalSourcePath:   new("partials/item_row.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   46,
														Column: 17,
													},
													NameLocation: ast_domain.Location{
														Line:   46,
														Column: 9,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   46,
																		Column: 17,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   46,
																		Column: 17,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   46,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   46,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
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
																Line:   46,
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
																			TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cat",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 3,
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
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Name",
																			ReferenceLocation: ast_domain.Location{
																				Line:   50,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("cat"),
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 3,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("cat"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   46,
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
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 3,
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
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   59,
																				Column: 39,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   32,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 3,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   32,
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
																Line:   46,
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
															Literal: ":0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   46,
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
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   47,
													Column: 3,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_row_8e7dec6a"),
													OriginalSourcePath:   new("partials/item_row.pk"),
												},
												DirIf: &ast_domain.Directive{
													Type: ast_domain.DirectiveIf,
													Location: ast_domain.Location{
														Line:   47,
														Column: 52,
													},
													NameLocation: ast_domain.Location{
														Line:   47,
														Column: 46,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 52,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Detail",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 52,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   31,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 52,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   31,
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
														Line:   47,
														Column: 32,
													},
													NameLocation: ast_domain.Location{
														Line:   47,
														Column: 24,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 32,
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
																	CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Detail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 32,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   31,
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
																CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Detail",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 32,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   31,
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
															CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_item_row_8e7dec6a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Detail",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 32,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   31,
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
																Line:   47,
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
																			TypeExpression:       typeExprFromString("main_aaf9a2e0.Category"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cat",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 3,
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
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Name",
																			ReferenceLocation: ast_domain.Location{
																				Line:   50,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("cat"),
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 3,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("cat"),
																	OriginalSourcePath:  new("partials/item_row.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   47,
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
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 3,
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
																			CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   59,
																				Column: 39,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   32,
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
																		CanonicalPackagePath: "testcase_101_nested_partial_p_for_loop_var_scope/dist/partials/partials_category_list_3eee8006",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 3,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   32,
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
																Line:   47,
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
															Literal: ":1",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   47,
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
														Value: "detail",
														Location: ast_domain.Location{
															Line:   47,
															Column: 16,
														},
														NameLocation: ast_domain.Location{
															Line:   47,
															Column: 9,
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
