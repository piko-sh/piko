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
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "data-status",
						RawExpression: "state.Code == 200 ? 'ok' : (state.Code < 500 ? 'client-error' : 'server-error')",
						Expression: &ast_domain.TernaryExpression{
							Condition: &ast_domain.BinaryExpression{
								Left: &ast_domain.MemberExpression{
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
												CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 22,
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
										Name: "Code",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Code",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 2,
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
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Code",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 22,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Operator: "==",
								Right: &ast_domain.IntegerLiteral{
									Value: 200,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 15,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("pages/main.pk"),
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
									OriginalSourcePath: new("pages/main.pk"),
									Stringability:      1,
								},
							},
							Consequent: &ast_domain.StringLiteral{
								Value: "ok",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 21,
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
							Alternate: &ast_domain.TernaryExpression{
								Condition: &ast_domain.BinaryExpression{
									Left: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 29,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 22,
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
											Name: "Code",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 35,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Code",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
														Column: 2,
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
											Column: 29,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Code",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									Operator: "<",
									Right: &ast_domain.IntegerLiteral{
										Value: 500,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 42,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 29,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Consequent: &ast_domain.StringLiteral{
									Value: "client-error",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 48,
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
								Alternate: &ast_domain.StringLiteral{
									Value: "server-error",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 65,
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
								RelativeLocation: ast_domain.Location{
									Line:   1,
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Location: ast_domain.Location{
							Line:   22,
							Column: 22,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 8,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   22,
							Column: 103,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 103,
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
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   22,
									Column: 103,
								},
								Literal: "\n    Message: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("pages/main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   23,
									Column: 17,
								},
								RawExpression: "~state.Message ? state.Message : 'No message'",
								Expression: &ast_domain.TernaryExpression{
									Condition: &ast_domain.UnaryExpression{
										Operator: "~",
										Right: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 2,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
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
												Name: "Message",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Message",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 2,
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
												Column: 2,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Message",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
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
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
									},
									Consequent: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 18,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
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
											Name: "Message",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Message",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
														Column: 2,
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
											Column: 18,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Message",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   32,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									Alternate: &ast_domain.StringLiteral{
										Value: "No message",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 34,
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
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Message",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   32,
												Column: 2,
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
										CanonicalPackagePath: "testcase_040_ternary_nested/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Message",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   32,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   23,
									Column: 65,
								},
								Literal: "\n  ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("pages/main.pk"),
								},
							},
						},
					},
				},
			},
		},
	}
}()
