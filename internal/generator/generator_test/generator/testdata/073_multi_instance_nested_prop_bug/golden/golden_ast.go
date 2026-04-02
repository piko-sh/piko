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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "container",
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
								TextContent: "Multiple Card Instances",
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
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "card_label_first_card_c872cad8",
								PartialAlias:        "card",
								PartialPackageName:  "partials_card_bfc4a3cf",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"label": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "First Card",
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 36,
										},
										GoFieldName: "Label",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
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
							Value: "r.0:1",
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
								OriginalSourcePath: new("partials/card.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "card",
								Location: ast_domain.Location{
									Line:   22,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 8,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "label",
								Value: "First Card",
								Location: ast_domain.Location{
									Line:   24,
									Column: 36,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 29,
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
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
										OriginalSourcePath: new("partials/card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "card-title",
										Location: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 28,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 28,
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
													Line:   23,
													Column: 31,
												},
												RawExpression: "props.Label",
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
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_card_label_first_card_c872cad8"),
															OriginalSourcePath: new("partials/card.pk"),
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
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
															},
															BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
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
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 31,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
														OriginalSourcePath:  new("partials/card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "badge_card_label_first_card_c872cad8_text_props_label_3ba509ae",
										PartialAlias:        "badge",
										PartialPackageName:  "partials_badge_63370d86",
										InvokerPackageAlias: "partials_card_bfc4a3cf",
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"text": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_card_label_first_card_c872cad8"),
															OriginalSourcePath: new("partials/card.pk"),
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
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_card_label_first_card_c872cad8"),
															},
															BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
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
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("props_card_label_first_card_c872cad8"),
														},
														BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
														OriginalSourcePath:  new("partials/card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   24,
													Column: 37,
												},
												GoFieldName: "Text",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_card_bfc4a3cf",
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_card_label_first_card_c872cad8"),
													},
													BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"text": "partials_card_bfc4a3cf",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
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
										OriginalSourcePath: new("partials/badge.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "badge",
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 9,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "text",
										RawExpression: "props.Label",
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
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_card_label_first_card_c872cad8"),
													OriginalSourcePath: new("partials/card.pk"),
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
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
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
													CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 37,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
												OriginalSourcePath:  new("partials/card.pk"),
												GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 37,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 30,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Label",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 37,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_card_label_first_card_c872cad8"),
											OriginalSourcePath:  new("partials/card.pk"),
											GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
											Stringability:       1,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   22,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_badge_63370d86"),
											OriginalSourcePath:   new("partials/badge.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   22,
												Column: 23,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   22,
													Column: 26,
												},
												RawExpression: "props.Text",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_badge_63370d86.Props"),
																PackageAlias:         "partials_badge_63370d86",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_badge_card_label_first_card_c872cad8_text_props_label_3ba509ae"),
															OriginalSourcePath: new("partials/badge.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Text",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_badge_63370d86",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Text",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
															},
															BaseCodeGenVarName:  new("props_badge_card_label_first_card_c872cad8_text_props_label_3ba509ae"),
															OriginalSourcePath:  new("partials/badge.pk"),
															GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
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
															PackageAlias:         "partials_badge_63370d86",
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Text",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 26,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														BaseCodeGenVarName:  new("props_badge_card_label_first_card_c872cad8_text_props_label_3ba509ae"),
														OriginalSourcePath:  new("partials/badge.pk"),
														GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_badge_63370d86",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Text",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 26,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_badge_card_label_first_card_c872cad8_text_props_label_3ba509ae"),
													OriginalSourcePath:  new("partials/badge.pk"),
													GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
													Stringability:       1,
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
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "card_label_second_card_e0600ecf",
								PartialAlias:        "card",
								PartialPackageName:  "partials_card_bfc4a3cf",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   25,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"label": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "Second Card",
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   25,
											Column: 36,
										},
										GoFieldName: "Label",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
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
							Value: "r.0:2",
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
								OriginalSourcePath: new("partials/card.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "card",
								Location: ast_domain.Location{
									Line:   22,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 8,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "label",
								Value: "Second Card",
								Location: ast_domain.Location{
									Line:   25,
									Column: 36,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 29,
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
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
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
										OriginalSourcePath: new("partials/card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "card-title",
										Location: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 28,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 28,
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
													Line:   23,
													Column: 31,
												},
												RawExpression: "props.Label",
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
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_card_label_second_card_e0600ecf"),
															OriginalSourcePath: new("partials/card.pk"),
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
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
															},
															BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
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
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 31,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
														OriginalSourcePath:  new("partials/card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "badge_card_label_second_card_e0600ecf_text_props_label_18b1bb64",
										PartialAlias:        "badge",
										PartialPackageName:  "partials_badge_63370d86",
										InvokerPackageAlias: "partials_card_bfc4a3cf",
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"text": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_card_label_second_card_e0600ecf"),
															OriginalSourcePath: new("partials/card.pk"),
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
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_card_label_second_card_e0600ecf"),
															},
															BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
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
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("props_card_label_second_card_e0600ecf"),
														},
														BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
														OriginalSourcePath:  new("partials/card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   24,
													Column: 37,
												},
												GoFieldName: "Text",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_card_bfc4a3cf",
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_card_label_second_card_e0600ecf"),
													},
													BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"text": "partials_card_bfc4a3cf",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:1",
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
										OriginalSourcePath: new("partials/badge.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "badge",
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 9,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "text",
										RawExpression: "props.Label",
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
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_card_label_second_card_e0600ecf"),
													OriginalSourcePath: new("partials/card.pk"),
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
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
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
													CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 37,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
												OriginalSourcePath:  new("partials/card.pk"),
												GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 37,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 30,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Label",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 37,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_card_label_second_card_e0600ecf"),
											OriginalSourcePath:  new("partials/card.pk"),
											GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
											Stringability:       1,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   22,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_badge_63370d86"),
											OriginalSourcePath:   new("partials/badge.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   22,
												Column: 23,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   22,
													Column: 26,
												},
												RawExpression: "props.Text",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_badge_63370d86.Props"),
																PackageAlias:         "partials_badge_63370d86",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_badge_card_label_second_card_e0600ecf_text_props_label_18b1bb64"),
															OriginalSourcePath: new("partials/badge.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Text",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_badge_63370d86",
																CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Text",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
															},
															BaseCodeGenVarName:  new("props_badge_card_label_second_card_e0600ecf_text_props_label_18b1bb64"),
															OriginalSourcePath:  new("partials/badge.pk"),
															GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
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
															PackageAlias:         "partials_badge_63370d86",
															CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Text",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 26,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														BaseCodeGenVarName:  new("props_badge_card_label_second_card_e0600ecf_text_props_label_18b1bb64"),
														OriginalSourcePath:  new("partials/badge.pk"),
														GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_badge_63370d86",
														CanonicalPackagePath: "testcase_073_multi_instance_nested_prop_bug/dist/partials/partials_badge_63370d86",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Text",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 26,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_badge_card_label_second_card_e0600ecf_text_props_label_18b1bb64"),
													OriginalSourcePath:  new("partials/badge.pk"),
													GeneratedSourcePath: new("dist/partials/partials_badge_63370d86/generated.go"),
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
	}
}()
