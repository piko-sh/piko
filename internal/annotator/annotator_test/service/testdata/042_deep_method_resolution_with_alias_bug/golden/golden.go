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
					Line:   22,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   22,
						Column: 5,
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "target",
						Location: ast_domain.Location{
							Line:   22,
							Column: 12,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   22,
							Column: 20,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 20,
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
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   22,
									Column: 23,
								},
								RawExpression: "state.ServiceResponse.Transaction.Amount.MustNumber()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
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
																CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 23,
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
														Name: "ServiceResponse",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("services.TransactionServiceResponse"),
																PackageAlias:         "services",
																CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/services",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ServiceResponse",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   35,
																	Column: 2,
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
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("services.TransactionServiceResponse"),
															PackageAlias:         "services",
															CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/services",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ServiceResponse",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   35,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Transaction",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 23,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dtos.TransactionDto"),
															PackageAlias:         "dtos",
															CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/dtos",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Transaction",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("services/transaction.go"),
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
														TypeExpression:       typeExprFromString("dtos.TransactionDto"),
														PackageAlias:         "dtos",
														CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/dtos",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Transaction",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("services/transaction.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Amount",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 35,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("maths.Money"),
														PackageAlias:         "maths",
														CanonicalPackagePath: "piko.sh/piko/wdk/maths",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Amount",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   50,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dtos/transaction.go"),
													Stringability:       4,
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
													TypeExpression:       typeExprFromString("maths.Money"),
													PackageAlias:         "maths",
													CanonicalPackagePath: "piko.sh/piko/wdk/maths",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Amount",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dtos/transaction.go"),
												Stringability:       4,
											},
										},
										Property: &ast_domain.Identifier{
											Name: "MustNumber",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 42,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "maths",
													CanonicalPackagePath: "piko.sh/piko/wdk/maths",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MustNumber",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   205,
														Column: 1,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
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
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "maths",
												CanonicalPackagePath: "piko.sh/piko/wdk/maths",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MustNumber",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 23,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   205,
													Column: 1,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
										},
									},
									Args: []ast_domain.Expression{},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "maths",
											CanonicalPackagePath: "piko.sh/piko/wdk/maths",
										},
										BaseCodeGenVarName: new("pageData"),
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "maths",
										CanonicalPackagePath: "piko.sh/piko/wdk/maths",
									},
									BaseCodeGenVarName: new("pageData"),
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
						},
					},
				},
			},
			&ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Location: ast_domain.Location{
					Line:   23,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.1",
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
						OriginalSourcePath: new("main.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "target-2",
						Location: ast_domain.Location{
							Line:   23,
							Column: 12,
						},
						NameLocation: ast_domain.Location{
							Line:   23,
							Column: 8,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "hello",
						RawExpression: "state.ServiceResponse.Transaction.Amount.MustNumber()",
						Expression: &ast_domain.CallExpression{
							Callee: &ast_domain.MemberExpression{
								Base: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 30,
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
												Name: "ServiceResponse",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("services.TransactionServiceResponse"),
														PackageAlias:         "services",
														CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/services",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ServiceResponse",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   35,
															Column: 2,
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
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("services.TransactionServiceResponse"),
													PackageAlias:         "services",
													CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/services",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ServiceResponse",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Transaction",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 23,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dtos.TransactionDto"),
													PackageAlias:         "dtos",
													CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/dtos",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Transaction",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   49,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("services/transaction.go"),
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
												TypeExpression:       typeExprFromString("dtos.TransactionDto"),
												PackageAlias:         "dtos",
												CanonicalPackagePath: "testcase_42_deep_method_resolution_with_alias_bug/dtos",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Transaction",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   49,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("services/transaction.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Amount",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("maths.Money"),
												PackageAlias:         "maths",
												CanonicalPackagePath: "piko.sh/piko/wdk/maths",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Amount",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dtos/transaction.go"),
											Stringability:       4,
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
											TypeExpression:       typeExprFromString("maths.Money"),
											PackageAlias:         "maths",
											CanonicalPackagePath: "piko.sh/piko/wdk/maths",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Amount",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 30,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   50,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dtos/transaction.go"),
										Stringability:       4,
									},
								},
								Property: &ast_domain.Identifier{
									Name: "MustNumber",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 42,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("function"),
											PackageAlias:         "maths",
											CanonicalPackagePath: "piko.sh/piko/wdk/maths",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "MustNumber",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 30,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   205,
												Column: 1,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
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
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "maths",
										CanonicalPackagePath: "piko.sh/piko/wdk/maths",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "MustNumber",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 30,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   205,
											Column: 1,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
								},
							},
							Args: []ast_domain.Expression{},
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 1,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "maths",
									CanonicalPackagePath: "piko.sh/piko/wdk/maths",
								},
								BaseCodeGenVarName: new("pageData"),
								Stringability:      1,
							},
						},
						Location: ast_domain.Location{
							Line:   23,
							Column: 30,
						},
						NameLocation: ast_domain.Location{
							Line:   23,
							Column: 22,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "maths",
								CanonicalPackagePath: "piko.sh/piko/wdk/maths",
							},
							BaseCodeGenVarName: new("pageData"),
							Stringability:      1,
						},
					},
				},
			},
		},
	}
}()
