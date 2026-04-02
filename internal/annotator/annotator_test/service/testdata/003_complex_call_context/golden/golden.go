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
					Line:   41,
					Column: 5,
				},
				TagName: "main",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   41,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   44,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "card_greeting_state_greeting_95d632c8",
								PartialAlias:        "card",
								PartialPackageName:  "partials_card_bfc4a3cf",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   42,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"greeting": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 44,
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
												Name: "Greeting",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Greeting",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 44,
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
															CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Greeting",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 44,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
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
													CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Greeting",
													ReferenceLocation: ast_domain.Location{
														Line:   42,
														Column: 44,
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
														CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Greeting",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 44,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   42,
											Column: 44,
										},
										GoFieldName: "Greeting",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Greeting",
												ReferenceLocation: ast_domain.Location{
													Line:   42,
													Column: 44,
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
													CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Greeting",
													ReferenceLocation: ast_domain.Location{
														Line:   42,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   30,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"greeting": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   44,
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
								Value: "card",
								Location: ast_domain.Location{
									Line:   44,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   44,
									Column: 10,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "greeting",
								RawExpression: "state.Greeting",
								Expression: &ast_domain.MemberExpression{
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
												CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   42,
													Column: 44,
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
										Name: "Greeting",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Greeting",
												ReferenceLocation: ast_domain.Location{
													Line:   42,
													Column: 44,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
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
											CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Greeting",
											ReferenceLocation: ast_domain.Location{
												Line:   42,
												Column: 44,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   30,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   42,
									Column: 44,
								},
								NameLocation: ast_domain.Location{
									Line:   42,
									Column: 33,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Greeting",
										ReferenceLocation: ast_domain.Location{
											Line:   42,
											Column: 44,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   30,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
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
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   45,
										Column: 9,
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   45,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   45,
												Column: 12,
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
													Line:   45,
													Column: 12,
												},
												Literal: "\n            ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("partials/card.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   46,
													Column: 16,
												},
												RawExpression: "Hello(state.Message, props.Greeting)",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "Hello",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:          typeExprFromString("function"),
																PackageAlias:            "partials_card_bfc4a3cf",
																CanonicalPackagePath:    "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																IsExportedPackageSymbol: true,
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Hello",
																ReferenceLocation: ast_domain.Location{
																	Line:   46,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("Hello"),
															OriginalSourcePath: new("partials/card.pk"),
															Stringability:      1,
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Response"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   46,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_greeting_state_greeting_95d632c8"),
																	OriginalSourcePath: new("partials/card.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Message",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 13,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Message",
																		ReferenceLocation: ast_domain.Location{
																			Line:   46,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   28,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_greeting_state_greeting_95d632c8"),
																	OriginalSourcePath:  new("partials/card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Message",
																	ReferenceLocation: ast_domain.Location{
																		Line:   46,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   28,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_greeting_state_greeting_95d632c8"),
																OriginalSourcePath:  new("partials/card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																Stringability:       1,
															},
														},
														&ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 22,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   46,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("props_card_greeting_state_greeting_95d632c8"),
																	OriginalSourcePath: new("partials/card.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Greeting",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 28,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Greeting",
																		ReferenceLocation: ast_domain.Location{
																			Line:   46,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   26,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Greeting",
																			ReferenceLocation: ast_domain.Location{
																				Line:   42,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName: new("pageData"),
																	},
																	BaseCodeGenVarName:  new("props_card_greeting_state_greeting_95d632c8"),
																	OriginalSourcePath:  new("partials/card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Greeting",
																	ReferenceLocation: ast_domain.Location{
																		Line:   46,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   26,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_03_complex_call_context/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Greeting",
																		ReferenceLocation: ast_domain.Location{
																			Line:   42,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																},
																BaseCodeGenVarName:  new("props_card_greeting_state_greeting_95d632c8"),
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
															CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
														},
														BaseCodeGenVarName: new("Hello"),
														OriginalSourcePath: new("partials/card.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_03_complex_call_context/dist/partials/partials_card_bfc4a3cf",
													},
													BaseCodeGenVarName: new("Hello"),
													OriginalSourcePath: new("partials/card.pk"),
													Stringability:      1,
												},
											},
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   46,
													Column: 55,
												},
												Literal: "\n        ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("partials/card.pk"),
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
