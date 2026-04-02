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
								TextContent: "Items List",
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
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "items",
								Location: ast_domain.Location{
									Line:   24,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 10,
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
									OriginalPackageAlias: new("partials_item_card_e11b3960"),
									OriginalSourcePath:   new("partials/item_card.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "item_card_item_id_item_id_item_name_item_name_db0b718d",
										PartialAlias:        "item_card",
										PartialPackageName:  "partials_item_card_e11b3960",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   25,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"item_id": ast_domain.PropValue{
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   29,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
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
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   29,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   29,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("item"),
															},
															BaseCodeGenVarName:  new("item"),
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
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   29,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("item"),
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   29,
													Column: 19,
												},
												GoFieldName: "ItemID",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("item"),
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											"item_name": ast_domain.PropValue{
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
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
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   47,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   30,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   47,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("item"),
															},
															BaseCodeGenVarName:  new("item"),
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
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   47,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   47,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("item"),
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   30,
													Column: 21,
												},
												GoFieldName: "ItemName",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   47,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   47,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("item"),
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											IsLoopDependent: true,
											IsLoopDependent: true,
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"item_id":   "pages_main_594861c5",
										"item_name": "pages_main_594861c5",
									},
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   26,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   26,
										Column: 9,
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
													TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
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
												OriginalSourcePath: new("partials/item_card.pk"),
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
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
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
												Name: "Items",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 15,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.Item"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.Item"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   49,
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
												TypeExpression:       typeExprFromString("[]pages_main_594861c5.Item"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   49,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]pages_main_594861c5.Item"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   49,
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
											TypeExpression:       typeExprFromString("[]pages_main_594861c5.Item"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Items",
											ReferenceLocation: ast_domain.Location{
												Line:   26,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   49,
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
										Line:   27,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   27,
										Column: 9,
									},
									RawExpression: "item.ID",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "item",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "item",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 13,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("item"),
												OriginalSourcePath: new("partials/item_actions.pk"),
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
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
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
												CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ID",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 13,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("item"),
											OriginalSourcePath:  new("partials/item_actions.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ID",
											ReferenceLocation: ast_domain.Location{
												Line:   27,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   46,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("item"),
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
												Line:   22,
												Column: 3,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
											Literal: "r.0:1:0.",
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
												OriginalSourcePath: new("partials/item_card.pk"),
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 13,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("partials/item_actions.pk"),
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
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("partials/item_actions.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
										OriginalSourcePath: new("partials/item_card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "item-card",
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
										Name:          "item_id",
										RawExpression: "item.ID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "item",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "item",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("item"),
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
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
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
													CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   29,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   29,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   29,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ID",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("item"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									ast_domain.DynamicAttribute{
										Name:          "item_name",
										RawExpression: "item.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "item",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "item",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("item"),
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
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   47,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
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
													CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   30,
														Column: 21,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   47,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   30,
											Column: 21,
										},
										NameLocation: ast_domain.Location{
											Line:   30,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
													Column: 21,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   47,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("item"),
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
											Line:   23,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: "r.0:1:0.",
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
														OriginalSourcePath: new("partials/item_card.pk"),
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 13,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_actions.pk"),
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
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
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
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_actions.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
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
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-name",
												Location: ast_domain.Location{
													Line:   23,
													Column: 18,
												},
												NameLocation: ast_domain.Location{
													Line:   23,
													Column: 11,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   23,
													Column: 29,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 29,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:0.",
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
																OriginalSourcePath: new("partials/item_card.pk"),
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
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 13,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_actions.pk"),
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
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   27,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   46,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
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
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_actions.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 29,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":0:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 29,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   23,
															Column: 32,
														},
														RawExpression: "state.ItemName",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Response"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 32,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	OriginalSourcePath: new("partials/item_card.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "ItemName",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemName",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 32,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   46,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																	PackageAlias:         "partials_item_card_e11b3960",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ItemName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 32,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ItemName",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 32,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
											Line:   22,
											Column: 3,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_actions_7438833b"),
											OriginalSourcePath:   new("partials/item_actions.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54",
												PartialAlias:        "item_actions",
												PartialPackageName:  "partials_item_actions_7438833b",
												InvokerPackageAlias: "partials_item_card_e11b3960",
												Location: ast_domain.Location{
													Line:   24,
													Column: 5,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"item_id": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Response"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 17,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	OriginalSourcePath: new("partials/item_card.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "ItemID",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 17,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   45,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_item_card_e11b3960",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ItemID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   25,
																				Column: 17,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   45,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	},
																	BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																	PackageAlias:         "partials_item_card_e11b3960",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ItemID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 17,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   45,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 17,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   45,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																},
																BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   25,
															Column: 17,
														},
														GoFieldName: "ItemID",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ItemID",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   45,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_item_card_e11b3960",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ItemID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 17,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   45,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															},
															BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
															Stringability:       1,
														},
													},
													"item_name": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Response"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 19,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	OriginalSourcePath: new("partials/item_card.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "ItemName",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemName",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 19,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   46,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_item_card_e11b3960",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ItemName",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 19,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   46,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	},
																	BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																	PackageAlias:         "partials_item_card_e11b3960",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ItemName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemName",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 19,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   46,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																},
																BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   26,
															Column: 19,
														},
														GoFieldName: "ItemName",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ItemName",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_item_card_e11b3960",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ItemName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															},
															BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											DynamicAttributeOrigins: map[string]string{
												"item_id":   "partials_item_card_e11b3960",
												"item_name": "partials_item_card_e11b3960",
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
														OriginalSourcePath: new("partials/item_actions.pk"),
														Stringability:      1,
													},
													Literal: "r.0:1:0.",
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
														OriginalSourcePath: new("partials/item_actions.pk"),
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 13,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_actions.pk"),
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
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
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
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_actions.pk"),
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
														OriginalSourcePath: new("partials/item_actions.pk"),
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
												OriginalSourcePath: new("partials/item_actions.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-actions",
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
												Name:          "item_id",
												RawExpression: "state.ItemID",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Response"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath: new("partials/item_card.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "ItemID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ItemID",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   45,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
															PackageAlias:         "partials_item_card_e11b3960",
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ItemID",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   45,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
														OriginalSourcePath:  new("partials/item_card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   25,
													Column: 17,
												},
												NameLocation: ast_domain.Location{
													Line:   25,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_item_card_e11b3960",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ItemID",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   45,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
													OriginalSourcePath:  new("partials/item_card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
													Stringability:       1,
												},
											},
											ast_domain.DynamicAttribute{
												Name:          "item_name",
												RawExpression: "state.ItemName",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Response"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath: new("partials/item_card.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "ItemName",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_item_card_e11b3960",
																CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ItemName",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
															PackageAlias:         "partials_item_card_e11b3960",
															CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ItemName",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
														OriginalSourcePath:  new("partials/item_card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   26,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   26,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_item_card_e11b3960",
														CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_card_e11b3960",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ItemName",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_item_card_e11b3960Data_item_card_item_id_item_id_item_name_item_name_db0b718d"),
													OriginalSourcePath:  new("partials/item_card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
												TagName: "button",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_actions_7438833b"),
													OriginalSourcePath:   new("partials/item_actions.pk"),
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
																OriginalSourcePath: new("partials/item_actions.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:0.",
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
																OriginalSourcePath: new("partials/item_actions.pk"),
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
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 13,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_actions.pk"),
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
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   27,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   46,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
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
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_actions.pk"),
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
																OriginalSourcePath: new("partials/item_actions.pk"),
																Stringability:      1,
															},
															Literal: ":1:0",
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
														OriginalSourcePath: new("partials/item_actions.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   23,
															Column: 13,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_actions_7438833b"),
															OriginalSourcePath:   new("partials/item_actions.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   23,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_actions.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:0.",
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
																		OriginalSourcePath: new("partials/item_actions.pk"),
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
																					TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   24,
																						Column: 13,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_actions.pk"),
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
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   27,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   46,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
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
																				CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 13,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   38,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_actions.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   23,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_actions.pk"),
																		Stringability:      1,
																	},
																	Literal: ":1:0:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_actions.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   23,
																	Column: 13,
																},
																Literal: "Edit ",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/item_actions.pk"),
																},
															},
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   23,
																	Column: 21,
																},
																RawExpression: "state.ItemName",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_item_actions_7438833b.Response"),
																				PackageAlias:         "partials_item_actions_7438833b",
																				CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   23,
																					Column: 21,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																			OriginalSourcePath: new("partials/item_actions.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "ItemName",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_item_actions_7438833b",
																				CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ItemName",
																				ReferenceLocation: ast_domain.Location{
																					Line:   23,
																					Column: 21,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   38,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																			OriginalSourcePath:  new("partials/item_actions.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_item_actions_7438833b/generated.go"),
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
																			PackageAlias:         "partials_item_actions_7438833b",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ItemName",
																			ReferenceLocation: ast_domain.Location{
																				Line:   23,
																				Column: 21,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   38,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																		OriginalSourcePath:  new("partials/item_actions.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_item_actions_7438833b/generated.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_actions_7438833b",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemName",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 21,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																	OriginalSourcePath:  new("partials/item_actions.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_actions_7438833b/generated.go"),
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
													Line:   24,
													Column: 5,
												},
												TagName: "button",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_actions_7438833b"),
													OriginalSourcePath:   new("partials/item_actions.pk"),
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
																OriginalSourcePath: new("partials/item_actions.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:0.",
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
																OriginalSourcePath: new("partials/item_actions.pk"),
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
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 13,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_actions.pk"),
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
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   27,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   46,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
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
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_actions.pk"),
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
																OriginalSourcePath: new("partials/item_actions.pk"),
																Stringability:      1,
															},
															Literal: ":1:1",
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
														OriginalSourcePath: new("partials/item_actions.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   24,
															Column: 13,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_actions_7438833b"),
															OriginalSourcePath:   new("partials/item_actions.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   24,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_actions.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:0.",
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
																		OriginalSourcePath: new("partials/item_actions.pk"),
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
																					TypeExpression:       typeExprFromString("pages_main_594861c5.Item"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   24,
																						Column: 13,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_actions.pk"),
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
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   27,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   46,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
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
																				CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 13,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   38,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_actions.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   24,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_actions.pk"),
																		Stringability:      1,
																	},
																	Literal: ":1:1:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   24,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_actions.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   24,
																	Column: 13,
																},
																Literal: "Delete ",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/item_actions.pk"),
																},
															},
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																RawExpression: "state.ItemID",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_item_actions_7438833b.Response"),
																				PackageAlias:         "partials_item_actions_7438833b",
																				CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																			OriginalSourcePath: new("partials/item_actions.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "ItemID",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_item_actions_7438833b",
																				CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ItemID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   37,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																			OriginalSourcePath:  new("partials/item_actions.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_item_actions_7438833b/generated.go"),
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
																			PackageAlias:         "partials_item_actions_7438833b",
																			CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ItemID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 23,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   37,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																		OriginalSourcePath:  new("partials/item_actions.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_item_actions_7438833b/generated.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_item_actions_7438833b",
																		CanonicalPackagePath: "testcase_085_p_for_with_nested_partials/dist/partials/partials_item_actions_7438833b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ItemID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 23,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   37,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_item_actions_7438833bData_item_actions_item_card_item_id_item_id_item_name_item_name_db0b718d_item_id_state_itemid_item_name_state_itemname_88a9cc54"),
																	OriginalSourcePath:  new("partials/item_actions.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_actions_7438833b/generated.go"),
																	Stringability:       1,
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
		},
	}
}()
