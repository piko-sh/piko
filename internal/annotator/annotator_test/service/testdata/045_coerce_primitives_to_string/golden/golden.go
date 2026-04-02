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
				TagName: "div",
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatint_int64_state_int64value_10_1f398346",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   23,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatInt",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "int64",
														RelativeLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("int64"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 70,
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
																Name: "Int64Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int64"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Int64Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 70,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   56,
																			Column: 2,
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
																	TypeExpression:       typeExprFromString("int64"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Int64Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 70,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   56,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												&ast_domain.IntegerLiteral{
													Value: 10,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   23,
											Column: 70,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "int64-coerce",
								Location: ast_domain.Location{
									Line:   23,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.Int64Value",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 70,
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
										Name: "Int64Value",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Int64Value",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 70,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   56,
													Column: 2,
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
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Int64Value",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 70,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   56,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   23,
									Column: 70,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 62,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int64"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Int64Value",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 70,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   56,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatint_int64_state_int64value_10_1f398346"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_int64value_10_1f398346"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_int64value_10_1f398346"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_int64value_10_1f398346"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatint_int64_state_intvalue_10_303239c0",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   25,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatInt",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "int64",
														RelativeLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("int64"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 68,
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
																Name: "IntValue",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "IntValue",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 68,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   57,
																			Column: 2,
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
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "IntValue",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 68,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   57,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												&ast_domain.IntegerLiteral{
													Value: 10,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   25,
											Column: 68,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "int-coerce",
								Location: ast_domain.Location{
									Line:   25,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.IntValue",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 68,
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
										Name: "IntValue",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "IntValue",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 68,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   57,
													Column: 2,
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
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IntValue",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 68,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   57,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   25,
									Column: 68,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 60,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IntValue",
										ReferenceLocation: ast_domain.Location{
											Line:   25,
											Column: 68,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   57,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatint_int64_state_intvalue_10_303239c0"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_intvalue_10_303239c0"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_intvalue_10_303239c0"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_intvalue_10_303239c0"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatfloat_float64_state_float64value_r_f_1_64_85b70bc2",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   27,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatFloat",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "float64",
														RelativeLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("float64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("float64"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   27,
																			Column: 72,
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
																Name: "Float64Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("float64"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Float64Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   27,
																			Column: 72,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   58,
																			Column: 2,
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
																	TypeExpression:       typeExprFromString("float64"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Float64Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 72,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   58,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												&ast_domain.RuneLiteral{
													Value: 'f',
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												&ast_domain.IntegerLiteral{
													Value: -1,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												&ast_domain.IntegerLiteral{
													Value: 64,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   27,
											Column: 72,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "float64-coerce",
								Location: ast_domain.Location{
									Line:   27,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   27,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.Float64Value",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 72,
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
										Name: "Float64Value",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("float64"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Float64Value",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 72,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   58,
													Column: 2,
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
											TypeExpression:       typeExprFromString("float64"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Float64Value",
											ReferenceLocation: ast_domain.Location{
												Line:   27,
												Column: 72,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   58,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   27,
									Column: 72,
								},
								NameLocation: ast_domain.Location{
									Line:   27,
									Column: 64,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("float64"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Float64Value",
										ReferenceLocation: ast_domain.Location{
											Line:   27,
											Column: 72,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   58,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatfloat_float64_state_float64value_r_f_1_64_85b70bc2"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatfloat_float64_state_float64value_r_f_1_64_85b70bc2"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatfloat_float64_state_float64value_r_f_1_64_85b70bc2"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatfloat_float64_state_float64value_r_f_1_64_85b70bc2"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatbool_state_boolvalue_4be359a9",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   29,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatBool",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.MemberExpression{
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
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   29,
																	Column: 69,
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
														Name: "BoolValue",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BoolValue",
																ReferenceLocation: ast_domain.Location{
																	Line:   29,
																	Column: 69,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   59,
																	Column: 2,
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
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "BoolValue",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   59,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   29,
											Column: 69,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "bool-coerce",
								Location: ast_domain.Location{
									Line:   29,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   29,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.BoolValue",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 69,
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
										Name: "BoolValue",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "BoolValue",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   59,
													Column: 2,
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
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "BoolValue",
											ReferenceLocation: ast_domain.Location{
												Line:   29,
												Column: 69,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   59,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   29,
									Column: 69,
								},
								NameLocation: ast_domain.Location{
									Line:   29,
									Column: 61,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "BoolValue",
										ReferenceLocation: ast_domain.Location{
											Line:   29,
											Column: 69,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   59,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatbool_state_boolvalue_4be359a9"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatbool_state_boolvalue_4be359a9"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatbool_state_boolvalue_4be359a9"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatbool_state_boolvalue_4be359a9"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatuint_uint64_state_uintvalue_10_c1adc86b",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   31,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatUint",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "uint64",
														RelativeLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uint64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("uint64"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 69,
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
																Name: "UintValue",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("uint"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "UintValue",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 69,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   60,
																			Column: 2,
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
																	TypeExpression:       typeExprFromString("uint"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "UintValue",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 69,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   60,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												&ast_domain.IntegerLiteral{
													Value: 10,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   31,
											Column: 69,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "uint-coerce",
								Location: ast_domain.Location{
									Line:   31,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   31,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.UintValue",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   31,
													Column: 69,
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
										Name: "UintValue",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uint"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UintValue",
												ReferenceLocation: ast_domain.Location{
													Line:   31,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   60,
													Column: 2,
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
											TypeExpression:       typeExprFromString("uint"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "UintValue",
											ReferenceLocation: ast_domain.Location{
												Line:   31,
												Column: 69,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   60,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   31,
									Column: 69,
								},
								NameLocation: ast_domain.Location{
									Line:   31,
									Column: 61,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("uint"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "UintValue",
										ReferenceLocation: ast_domain.Location{
											Line:   31,
											Column: 69,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   60,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatuint_uint64_state_uintvalue_10_c1adc86b"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatuint_uint64_state_uintvalue_10_c1adc86b"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatuint_uint64_state_uintvalue_10_c1adc86b"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatuint_uint64_state_uintvalue_10_c1adc86b"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatuint_uint64_state_uint64value_10_d0e8f0aa",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   33,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatUint",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "uint64",
														RelativeLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uint64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("uint64"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 71,
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
																Name: "Uint64Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("uint64"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Uint64Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 71,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   61,
																			Column: 2,
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
																	TypeExpression:       typeExprFromString("uint64"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Uint64Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 71,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												&ast_domain.IntegerLiteral{
													Value: 10,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   33,
											Column: 71,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "uint64-coerce",
								Location: ast_domain.Location{
									Line:   33,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   33,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.Uint64Value",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   33,
													Column: 71,
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
										Name: "Uint64Value",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uint64"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Uint64Value",
												ReferenceLocation: ast_domain.Location{
													Line:   33,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   61,
													Column: 2,
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
											TypeExpression:       typeExprFromString("uint64"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Uint64Value",
											ReferenceLocation: ast_domain.Location{
												Line:   33,
												Column: 71,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   61,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   33,
									Column: 71,
								},
								NameLocation: ast_domain.Location{
									Line:   33,
									Column: 63,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("uint64"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Uint64Value",
										ReferenceLocation: ast_domain.Location{
											Line:   33,
											Column: 71,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   61,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatuint_uint64_state_uint64value_10_d0e8f0aa"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatuint_uint64_state_uint64value_10_d0e8f0aa"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatuint_uint64_state_uint64value_10_d0e8f0aa"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatuint_uint64_state_uint64value_10_d0e8f0aa"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_strconv_formatint_int64_state_nesteddata_count_10_e3082717",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   35,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "strconv",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("strconv.strconv"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatInt",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "int64",
														RelativeLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("int64"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
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
																			CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "state",
																			ReferenceLocation: ast_domain.Location{
																				Line:   35,
																				Column: 71,
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
																	Name: "NestedData",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("main_aaf9a2e0.NestedData"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "NestedData",
																			ReferenceLocation: ast_domain.Location{
																				Line:   35,
																				Column: 71,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   63,
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
																		TypeExpression:       typeExprFromString("main_aaf9a2e0.NestedData"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "NestedData",
																		ReferenceLocation: ast_domain.Location{
																			Line:   35,
																			Column: 71,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   63,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Count",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 18,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int64"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Count",
																		ReferenceLocation: ast_domain.Location{
																			Line:   35,
																			Column: 71,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   54,
																			Column: 25,
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
																	TypeExpression:       typeExprFromString("int64"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Count",
																	ReferenceLocation: ast_domain.Location{
																		Line:   35,
																		Column: 71,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   54,
																		Column: 25,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												&ast_domain.IntegerLiteral{
													Value: 10,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
											},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   35,
											Column: 71,
										},
										GoFieldName: "Value",
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
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "nested-coerce",
								Location: ast_domain.Location{
									Line:   35,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   35,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.NestedData.Count",
								Expression: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 71,
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
											Name: "NestedData",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("main_aaf9a2e0.NestedData"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "NestedData",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   63,
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
												TypeExpression:       typeExprFromString("main_aaf9a2e0.NestedData"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "NestedData",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Count",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 18,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Count",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   54,
													Column: 25,
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
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Count",
											ReferenceLocation: ast_domain.Location{
												Line:   35,
												Column: 71,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   54,
												Column: 25,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   35,
									Column: 71,
								},
								NameLocation: ast_domain.Location{
									Line:   35,
									Column: 63,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int64"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Count",
										ReferenceLocation: ast_domain.Location{
											Line:   35,
											Column: 71,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   54,
											Column: 25,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:6:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_strconv_formatint_int64_state_nesteddata_count_10_e3082717"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
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
															BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_nesteddata_count_10_e3082717"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
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
														BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_nesteddata_count_10_e3082717"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
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
													BaseCodeGenVarName:  new("props_string_consumer_value_strconv_formatint_int64_state_nesteddata_count_10_e3082717"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
							OriginalSourcePath:   new("partials/string_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "string_consumer_value_state_stringvalue_2507f4d5",
								PartialAlias:        "string_consumer",
								PartialPackageName:  "partials_string_consumer_6cdbb289",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   37,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   37,
															Column: 71,
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
												Name: "StringValue",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringValue",
														ReferenceLocation: ast_domain.Location{
															Line:   37,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   62,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StringValue",
															ReferenceLocation: ast_domain.Location{
																Line:   37,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
																Column: 2,
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
													CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StringValue",
													ReferenceLocation: ast_domain.Location{
														Line:   37,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   62,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringValue",
														ReferenceLocation: ast_domain.Location{
															Line:   37,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   62,
															Column: 2,
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
											Line:   37,
											Column: 71,
										},
										GoFieldName: "Value",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "StringValue",
												ReferenceLocation: ast_domain.Location{
													Line:   37,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   62,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StringValue",
													ReferenceLocation: ast_domain.Location{
														Line:   37,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   62,
														Column: 2,
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
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:7",
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
								OriginalSourcePath: new("partials/string_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "string-direct",
								Location: ast_domain.Location{
									Line:   37,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   37,
									Column: 44,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.StringValue",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   37,
													Column: 71,
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
										Name: "StringValue",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "StringValue",
												ReferenceLocation: ast_domain.Location{
													Line:   37,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   62,
													Column: 2,
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
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "StringValue",
											ReferenceLocation: ast_domain.Location{
												Line:   37,
												Column: 71,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   62,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   37,
									Column: 71,
								},
								NameLocation: ast_domain.Location{
									Line:   37,
									Column: 63,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "StringValue",
										ReferenceLocation: ast_domain.Location{
											Line:   37,
											Column: 71,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   62,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
									OriginalSourcePath:   new("partials/string_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
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
										OriginalSourcePath: new("partials/string_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_string_consumer_6cdbb289"),
											OriginalSourcePath:   new("partials/string_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/string_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 37,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_string_consumer_6cdbb289.Props"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_string_consumer_value_state_stringvalue_2507f4d5"),
															OriginalSourcePath: new("partials/string_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_string_consumer_6cdbb289",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "StringValue",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 71,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   62,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_string_consumer_value_state_stringvalue_2507f4d5"),
															OriginalSourcePath:  new("partials/string_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
															PackageAlias:         "partials_string_consumer_6cdbb289",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "StringValue",
																ReferenceLocation: ast_domain.Location{
																	Line:   37,
																	Column: 71,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   62,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_string_consumer_value_state_stringvalue_2507f4d5"),
														OriginalSourcePath:  new("partials/string_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_string_consumer_6cdbb289",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_string_consumer_6cdbb289",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StringValue",
															ReferenceLocation: ast_domain.Location{
																Line:   37,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_string_consumer_value_state_stringvalue_2507f4d5"),
													OriginalSourcePath:  new("partials/string_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_string_consumer_6cdbb289/generated.go"),
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
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_no_coerce_consumer_4a0aebc8"),
							OriginalSourcePath:   new("partials/no_coerce_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "no_coerce_consumer_value_state_int64value_9f0dacf6",
								PartialAlias:        "no_coerce_consumer",
								PartialPackageName:  "partials_no_coerce_consumer_4a0aebc8",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   39,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 72,
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
												Name: "Int64Value",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Int64Value",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 72,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   56,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Int64Value",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 72,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   56,
																Column: 2,
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
													TypeExpression:       typeExprFromString("int64"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Int64Value",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 72,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   56,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Int64Value",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 72,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   56,
															Column: 2,
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
											Line:   39,
											Column: 72,
										},
										GoFieldName: "Value",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Int64Value",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 72,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   56,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int64"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Int64Value",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 72,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   56,
														Column: 2,
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
								"value": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:8",
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
								OriginalSourcePath: new("partials/no_coerce_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "exact-match",
								Location: ast_domain.Location{
									Line:   39,
									Column: 51,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 47,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.Int64Value",
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
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 72,
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
										Name: "Int64Value",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Int64Value",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 72,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   56,
													Column: 2,
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
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Int64Value",
											ReferenceLocation: ast_domain.Location{
												Line:   39,
												Column: 72,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   56,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   39,
									Column: 72,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 64,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int64"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Int64Value",
										ReferenceLocation: ast_domain.Location{
											Line:   39,
											Column: 72,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   56,
											Column: 2,
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
									Line:   23,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_no_coerce_consumer_4a0aebc8"),
									OriginalSourcePath:   new("partials/no_coerce_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:8:0",
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
										OriginalSourcePath: new("partials/no_coerce_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "exact-value",
										Location: ast_domain.Location{
											Line:   23,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 32,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_no_coerce_consumer_4a0aebc8"),
											OriginalSourcePath:   new("partials/no_coerce_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:8:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 32,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/no_coerce_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 35,
												},
												RawExpression: "props.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_no_coerce_consumer_4a0aebc8.Props"),
																PackageAlias:         "partials_no_coerce_consumer_4a0aebc8",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_no_coerce_consumer_4a0aebc8",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 35,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_no_coerce_consumer_value_state_int64value_9f0dacf6"),
															OriginalSourcePath: new("partials/no_coerce_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "partials_no_coerce_consumer_4a0aebc8",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_no_coerce_consumer_4a0aebc8",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 35,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int64"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Int64Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   39,
																		Column: 72,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   56,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_no_coerce_consumer_value_state_int64value_9f0dacf6"),
															OriginalSourcePath:  new("partials/no_coerce_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_no_coerce_consumer_4a0aebc8/generated.go"),
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
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "partials_no_coerce_consumer_4a0aebc8",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_no_coerce_consumer_4a0aebc8",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Int64Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   39,
																	Column: 72,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   56,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_no_coerce_consumer_value_state_int64value_9f0dacf6"),
														OriginalSourcePath:  new("partials/no_coerce_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_no_coerce_consumer_4a0aebc8/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "partials_no_coerce_consumer_4a0aebc8",
														CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/partials/partials_no_coerce_consumer_4a0aebc8",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_45_coerce_primitives_to_string/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Int64Value",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 72,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   56,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_no_coerce_consumer_value_state_int64value_9f0dacf6"),
													OriginalSourcePath:  new("partials/no_coerce_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_no_coerce_consumer_4a0aebc8/generated.go"),
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
