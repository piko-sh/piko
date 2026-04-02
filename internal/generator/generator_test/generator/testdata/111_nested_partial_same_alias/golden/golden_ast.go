package nested_partial_same_alias_test

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
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					IsStructurallyStatic: true,
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
						OriginalSourcePath: new("pages/main.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "main",
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
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   36,
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
									Line:   36,
									Column: 3,
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
											Line:   36,
											Column: 42,
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
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   36,
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
									Line:   36,
									Column: 14,
								},
								NameLocation: ast_domain.Location{
									Line:   36,
									Column: 7,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   37,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   37,
										Column: 3,
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
										Value: "price",
										Location: ast_domain.Location{
											Line:   37,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   37,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   37,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
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
													Line:   37,
													Column: 26,
												},
												RawExpression: "helper.FormatPrice(props.Price)",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "helper",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       nil,
																	PackageAlias:         "partials_price_helper_75efb1b0",
																	CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_price_helper_75efb1b0",
																},
																BaseCodeGenVarName: new("partials_price_helper_75efb1b0"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "FormatPrice",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 8,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "partials_price_helper_75efb1b0",
																	CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_price_helper_75efb1b0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "FormatPrice",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 26,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_price_helper_75efb1b0"),
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
																PackageAlias:         "partials_price_helper_75efb1b0",
																CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_price_helper_75efb1b0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FormatPrice",
																ReferenceLocation: ast_domain.Location{
																	Line:   37,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_price_helper_75efb1b0"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 20,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   37,
																			Column: 26,
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
																	Column: 26,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Price",
																		ReferenceLocation: ast_domain.Location{
																			Line:   37,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   31,
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
																Column: 20,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Price",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 26,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   31,
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
															PackageAlias:         "partials_price_helper_75efb1b0",
															CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_price_helper_75efb1b0",
														},
														BaseCodeGenVarName: new("partials_price_helper_75efb1b0"),
														OriginalSourcePath: new("partials/card.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_price_helper_75efb1b0",
														CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_price_helper_75efb1b0",
													},
													BaseCodeGenVarName: new("partials_price_helper_75efb1b0"),
													OriginalSourcePath: new("partials/card.pk"),
													Stringability:      1,
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
							Line:   36,
							Column: 2,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_level_7_b73711aa",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   37,
									Column: 3,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"level": ast_domain.PropValue{
										Expression: &ast_domain.IntegerLiteral{
											Value: 7,
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
											Line:   37,
											Column: 43,
										},
										GoFieldName: "Level",
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
							DynamicAttributeOrigins: map[string]string{
								"class": "partials_badge_63370d86",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   36,
								Column: 2,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/badge.pk"),
								Stringability:      1,
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "class",
								RawExpression: "helper.GetBadgeClass(props.Level)",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "helper",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       nil,
													PackageAlias:         "partials_style_helper_e0538422",
													CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_style_helper_e0538422",
												},
												BaseCodeGenVarName: new("partials_style_helper_e0538422"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "GetBadgeClass",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 8,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "partials_style_helper_e0538422",
													CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_style_helper_e0538422",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "GetBadgeClass",
													ReferenceLocation: ast_domain.Location{
														Line:   36,
														Column: 15,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_style_helper_e0538422"),
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
												PackageAlias:         "partials_style_helper_e0538422",
												CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_style_helper_e0538422",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "GetBadgeClass",
												ReferenceLocation: ast_domain.Location{
													Line:   36,
													Column: 15,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("partials_style_helper_e0538422"),
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 22,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_badge_63370d86.Props"),
														PackageAlias:         "partials_badge_63370d86",
														CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_badge_63370d86",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   36,
															Column: 15,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_badge_level_7_b73711aa"),
													OriginalSourcePath: new("partials/badge.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Level",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 28,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "partials_badge_63370d86",
														CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_badge_63370d86",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Level",
														ReferenceLocation: ast_domain.Location{
															Line:   36,
															Column: 15,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
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
													BaseCodeGenVarName:  new("props_badge_level_7_b73711aa"),
													OriginalSourcePath:  new("partials/badge.pk"),
													GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
													Stringability:       1,
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 22,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_badge_63370d86",
													CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_badge_63370d86",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Level",
													ReferenceLocation: ast_domain.Location{
														Line:   36,
														Column: 15,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
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
												BaseCodeGenVarName:  new("props_badge_level_7_b73711aa"),
												OriginalSourcePath:  new("partials/badge.pk"),
												GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
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
											PackageAlias:         "partials_style_helper_e0538422",
											CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_style_helper_e0538422",
										},
										BaseCodeGenVarName: new("partials_style_helper_e0538422"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   36,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   36,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_style_helper_e0538422",
										CanonicalPackagePath: "testcase_111_nested_partial_same_alias/dist/partials/partials_style_helper_e0538422",
									},
									BaseCodeGenVarName: new("partials_style_helper_e0538422"),
									Stringability:      1,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   37,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   37,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/badge.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   37,
											Column: 9,
										},
										TextContent: "Badge",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_badge_63370d86"),
											OriginalSourcePath:   new("partials/badge.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   37,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/badge.pk"),
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
