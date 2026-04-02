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
					Line:   0,
					Column: 0,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				DirFor: &ast_domain.Directive{
					Type: ast_domain.DirectiveFor,
					Location: ast_domain.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast_domain.Location{
						Line:   0,
						Column: 0,
					},
					Expression: &ast_domain.ForInExpression{
						IndexVariable: &ast_domain.Identifier{
							Name: "i",
							RelativeLocation: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
						},
						ItemVariable: &ast_domain.Identifier{
							Name: "user",
							RelativeLocation: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
						},
						Collection: &ast_domain.MemberExpression{
							Base: &ast_domain.Identifier{
								Name: "state",
								RelativeLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_16_interaction_with_pfor/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
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
								Name: "Users",
								RelativeLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]models.User"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_16_interaction_with_pfor/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
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
								Line:   0,
								Column: 0,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]models.User"),
									PackageAlias:         "models",
									CanonicalPackagePath: "testcase_16_interaction_with_pfor/models",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Users",
									ReferenceLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
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
							Line:   0,
							Column: 0,
						},
						GoAnnotations: nil,
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "tr",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_user_row_7ab08aa2"),
							OriginalSourcePath:   new("partials/user-row.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "user_row_index_i_user_user_c0ffd7f2",
								PartialAlias:        "user-row",
								PartialPackageName:  "partials_user_row_7ab08aa2",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"index": ast_domain.PropValue{
										Expression: &ast_domain.Identifier{
											Name: "i",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "i",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "i",
														ReferenceLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("i"),
												},
												BaseCodeGenVarName: new("i"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoFieldName: "Index",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "i",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "i",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("i"),
											},
											BaseCodeGenVarName: new("i"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
									"user": ast_domain.PropValue{
										Expression: &ast_domain.Identifier{
											Name: "user",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_16_interaction_with_pfor/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "user",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_16_interaction_with_pfor/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "user",
														ReferenceLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("user"),
												},
												BaseCodeGenVarName: new("user"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoFieldName: "User",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("models.User"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_16_interaction_with_pfor/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "user",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_16_interaction_with_pfor/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "user",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("user"),
											},
											BaseCodeGenVarName: new("user"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									IsLoopDependent: true,
									IsLoopDependent: true,
								},
							},
						},
					},
				},
			},
		},
	}
}()
