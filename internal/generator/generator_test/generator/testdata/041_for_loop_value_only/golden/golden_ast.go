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
				TagName: "ul",
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
						TagName: "li",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   23,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   23,
								Column: 9,
							},
							RawExpression: "user in state.Users",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: nil,
								ItemVariable: &ast_domain.Identifier{
									Name: "user",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.User"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "user",
											ReferenceLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("user"),
										OriginalSourcePath: new("pages/main.pk"),
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
												TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 16,
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
										Name: "Users",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 15,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]pages_main_594861c5.User"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Users",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
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
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]pages_main_594861c5.User"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Users",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   34,
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
										TypeExpression:       typeExprFromString("[]pages_main_594861c5.User"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   34,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]pages_main_594861c5.User"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   34,
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
									TypeExpression:       typeExprFromString("[]pages_main_594861c5.User"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Users",
									ReferenceLocation: ast_domain.Location{
										Line:   23,
										Column: 16,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   34,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
							},
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   23,
								Column: 61,
							},
							NameLocation: ast_domain.Location{
								Line:   23,
								Column: 53,
							},
							RawExpression: "user.Name",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "user",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.User"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "user",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 61,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("user"),
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "Name",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 6,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Name",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 61,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   32,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("user"),
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
										CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Name",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 61,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   32,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("user"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Name",
									ReferenceLocation: ast_domain.Location{
										Line:   23,
										Column: 61,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   32,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("user"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						DirKey: &ast_domain.Directive{
							Type: ast_domain.DirectiveKey,
							Location: ast_domain.Location{
								Line:   23,
								Column: 44,
							},
							NameLocation: ast_domain.Location{
								Line:   23,
								Column: 37,
							},
							RawExpression: "user.ID",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "user",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.User"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "user",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 44,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("user"),
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "ID",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 6,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ID",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 44,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("user"),
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
										TypeExpression:       typeExprFromString("int"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ID",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 44,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("user"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("int"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "ID",
									ReferenceLocation: ast_domain.Location{
										Line:   23,
										Column: 44,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   31,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("user"),
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
									Literal: "r.0:0.",
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
											Name: "user",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.User"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "user",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("user"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "ID",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 6,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("user"),
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
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_041_for_loop_value_only/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ID",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 44,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("user"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
					},
				},
			},
		},
	}
}()
