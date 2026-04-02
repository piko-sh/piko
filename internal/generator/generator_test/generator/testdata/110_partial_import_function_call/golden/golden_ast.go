package partial_import_function_call_test

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
					Line:   35,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_card_bfc4a3cf"),
					OriginalSourcePath:   new("partials/card.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_price_100_05e4dc0a",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   38,
							Column: 2,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"price": ast_domain.PropValue{
								Expression: &ast_domain.IntegerLiteral{
									Value: 100,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
										},
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   38,
									Column: 41,
								},
								GoFieldName: "Price",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int64"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
									},
									OriginalSourcePath: new("pages/main.pk"),
									Stringability:      1,
								},
							},
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   35,
						Column: 2,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("partials/card.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "card",
						Location: ast_domain.Location{
							Line:   35,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   35,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   35,
							Column: 20,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   35,
								Column: 20,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/card.pk"),
								Stringability:      1,
							},
						},
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   35,
									Column: 23,
								},
								RawExpression: "FormatPrice(props.Price)",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.Identifier{
										Name: "FormatPrice",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:          typeExprFromString("function"),
												PackageAlias:            "partials_card_bfc4a3cf",
												CanonicalPackagePath:    "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
												IsExportedPackageSymbol: true,
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FormatPrice",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
													Column: 23,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("FormatPrice"),
											OriginalSourcePath: new("partials/card.pk"),
											Stringability:      1,
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 13,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_card_price_100_05e4dc0a"),
													OriginalSourcePath: new("partials/card.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Price",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 19,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Price",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_card_price_100_05e4dc0a"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
													Stringability:       1,
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 13,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_card_bfc4a3cf",
													CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Price",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												BaseCodeGenVarName:  new("props_card_price_100_05e4dc0a"),
												OriginalSourcePath:  new("partials/card.pk"),
												GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
												Stringability:       1,
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_card_bfc4a3cf",
											CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
										},
										BaseCodeGenVarName: new("FormatPrice"),
										OriginalSourcePath: new("partials/card.pk"),
										Stringability:      1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_card_bfc4a3cf",
										CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
									},
									BaseCodeGenVarName: new("FormatPrice"),
									OriginalSourcePath: new("partials/card.pk"),
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
					Line:   39,
					Column: 2,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.1",
					RelativeLocation: ast_domain.Location{
						Line:   39,
						Column: 2,
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
							Line:   39,
							Column: 5,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.1:0",
							RelativeLocation: ast_domain.Location{
								Line:   39,
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
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   39,
									Column: 5,
								},
								Literal: "From Render: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("pages/main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   39,
									Column: 21,
								},
								RawExpression: "state.FormattedPrice",
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
												CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 21,
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
										Name: "FormattedPrice",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FormattedPrice",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 21,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
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
											CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "FormattedPrice",
											ReferenceLocation: ast_domain.Location{
												Line:   39,
												Column: 21,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   30,
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
										CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "FormattedPrice",
										ReferenceLocation: ast_domain.Location{
											Line:   39,
											Column: 21,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   30,
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
					Line:   40,
					Column: 2,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.2",
					RelativeLocation: ast_domain.Location{
						Line:   40,
						Column: 2,
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
							Line:   40,
							Column: 5,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.2:0",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   40,
									Column: 5,
								},
								Literal: "From Template: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("pages/main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   40,
									Column: 23,
								},
								RawExpression: "card.FormatPrice(200)",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "card",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       nil,
													PackageAlias:         "partials_card_bfc4a3cf",
													CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
												},
												BaseCodeGenVarName: new("partials_card_bfc4a3cf"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "FormatPrice",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 6,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "partials_card_bfc4a3cf",
													CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FormatPrice",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_card_bfc4a3cf"),
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
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FormatPrice",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 23,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("partials_card_bfc4a3cf"),
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.IntegerLiteral{
											Value: 200,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 18,
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
									},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_card_bfc4a3cf",
											CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
										},
										BaseCodeGenVarName: new("partials_card_bfc4a3cf"),
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_card_bfc4a3cf",
										CanonicalPackagePath: "testcase_110_partial_import_function_call/dist/partials/partials_card_bfc4a3cf",
									},
									BaseCodeGenVarName: new("partials_card_bfc4a3cf"),
									OriginalSourcePath: new("pages/main.pk"),
									Stringability:      1,
								},
							},
						},
					},
				},
			},
		},
	}
}()
