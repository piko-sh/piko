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
					Line:   29,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_card_bfc4a3cf"),
					OriginalSourcePath:   new("partials/card.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_formatted_price_formatprice_state_price_0e16e292",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   44,
							Column: 2,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"formatted_price": ast_domain.PropValue{
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
												PackageAlias:            "pages_main_594861c5",
												CanonicalPackagePath:    "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
												IsExportedPackageSymbol: true,
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FormatPrice",
												ReferenceLocation: ast_domain.Location{
													Line:   44,
													Column: 51,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("FormatPrice"),
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 13,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 51,
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
												Name: "Price",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 19,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Price",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 51,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
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
												Column: 13,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Price",
													ReferenceLocation: ast_domain.Location{
														Line:   44,
														Column: 51,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
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
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
											},
											BaseCodeGenVarName: new("FormatPrice"),
										},
										BaseCodeGenVarName: new("FormatPrice"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   44,
									Column: 51,
								},
								GoFieldName: "FormattedPrice",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
										},
										BaseCodeGenVarName: new("FormatPrice"),
									},
									BaseCodeGenVarName: new("FormatPrice"),
									Stringability:      1,
								},
							},
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   29,
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
							Line:   29,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   29,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   29,
							Column: 20,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   29,
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
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   29,
									Column: 20,
								},
								Literal: "Price: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("partials/card.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   29,
									Column: 30,
								},
								RawExpression: "props.FormattedPrice",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props_card_formatted_price_formatprice_state_price_0e16e292"),
											OriginalSourcePath: new("partials/card.pk"),
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
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FormattedPrice",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   26,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
												},
												BaseCodeGenVarName: new("FormatPrice"),
											},
											BaseCodeGenVarName:  new("props_card_formatted_price_formatprice_state_price_0e16e292"),
											OriginalSourcePath:  new("partials/card.pk"),
											GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
											PackageAlias:         "partials_card_bfc4a3cf",
											CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/partials/partials_card_bfc4a3cf",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "FormattedPrice",
											ReferenceLocation: ast_domain.Location{
												Line:   29,
												Column: 30,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   26,
												Column: 2,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
											},
											BaseCodeGenVarName: new("FormatPrice"),
										},
										BaseCodeGenVarName:  new("props_card_formatted_price_formatprice_state_price_0e16e292"),
										OriginalSourcePath:  new("partials/card.pk"),
										GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
										Stringability:       1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_card_bfc4a3cf",
										CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/partials/partials_card_bfc4a3cf",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "FormattedPrice",
										ReferenceLocation: ast_domain.Location{
											Line:   29,
											Column: 30,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   26,
											Column: 2,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_106_parent_function_in_prop_binding/dist/pages/pages_main_594861c5",
										},
										BaseCodeGenVarName: new("FormatPrice"),
									},
									BaseCodeGenVarName:  new("props_card_formatted_price_formatprice_state_price_0e16e292"),
									OriginalSourcePath:  new("partials/card.pk"),
									GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
									Stringability:       1,
								},
							},
						},
					},
				},
			},
		},
	}
}()
