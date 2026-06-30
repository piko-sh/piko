package json_attribute_test

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
								TextContent: "JSON Attribute Demo",
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
							OriginalPackageAlias: new("partials_jsonbox_06a49cad"),
							OriginalSourcePath:   new("partials/jsonbox.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "jsonbox_config_state_config_ff327353",
								PartialAlias:        "jsonbox",
								PartialPackageName:  "partials_jsonbox_06a49cad",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"config": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
														UnderlyingTypeString: "struct{Config map[string]string}",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 41,
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
												Name: "Config",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("map[string]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Config",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("map[string]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Config",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
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
													TypeExpression:       typeExprFromString("map[string]string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Config",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("map[string]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Config",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       5,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 41,
										},
										GoFieldName: "Config",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("map[string]string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Config",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("map[string]string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Config",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       5,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"config":      "pages_main_594861c5",
								"data-config": "partials_jsonbox_06a49cad",
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
								OriginalSourcePath: new("partials/jsonbox.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "json-box",
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
								Name:          "config",
								RawExpression: "state.Config",
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
												CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
												UnderlyingTypeString: "struct{Config map[string]string}",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 41,
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
										Name: "Config",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("map[string]string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Config",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       5,
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
											TypeExpression:       typeExprFromString("map[string]string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Config",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 41,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       5,
									},
								},
								Location: ast_domain.Location{
									Line:   24,
									Column: 41,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("map[string]string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_155_json_attribute_value/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Config",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 41,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       5,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "data-config",
								RawExpression: "state.Config",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_jsonbox_06a49cad.Props"),
												PackageAlias:         "partials_jsonbox_06a49cad",
												CanonicalPackagePath: "testcase_155_json_attribute_value/dist/partials/partials_jsonbox_06a49cad",
												UnderlyingTypeString: "struct{Config map[string]string}",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 39,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("partials_jsonbox_06a49cadData_jsonbox_config_state_config_ff327353"),
											OriginalSourcePath: new("partials/jsonbox.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Config",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("map[string]string"),
												PackageAlias:         "partials_jsonbox_06a49cad",
												CanonicalPackagePath: "testcase_155_json_attribute_value/dist/partials/partials_jsonbox_06a49cad",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Config",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 39,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_jsonbox_06a49cadData_jsonbox_config_state_config_ff327353"),
											OriginalSourcePath:  new("partials/jsonbox.pk"),
											GeneratedSourcePath: new("dist/partials/partials_jsonbox_06a49cad/generated.go"),
											Stringability:       5,
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
											TypeExpression:       typeExprFromString("map[string]string"),
											PackageAlias:         "partials_jsonbox_06a49cad",
											CanonicalPackagePath: "testcase_155_json_attribute_value/dist/partials/partials_jsonbox_06a49cad",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Config",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 39,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   30,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_jsonbox_06a49cadData_jsonbox_config_state_config_ff327353"),
										OriginalSourcePath:  new("partials/jsonbox.pk"),
										GeneratedSourcePath: new("dist/partials/partials_jsonbox_06a49cad/generated.go"),
										Stringability:       5,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 39,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 25,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("map[string]string"),
										PackageAlias:         "partials_jsonbox_06a49cad",
										CanonicalPackagePath: "testcase_155_json_attribute_value/dist/partials/partials_jsonbox_06a49cad",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Config",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 39,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   30,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("partials_jsonbox_06a49cadData_jsonbox_config_state_config_ff327353"),
									OriginalSourcePath:  new("partials/jsonbox.pk"),
									GeneratedSourcePath: new("dist/partials/partials_jsonbox_06a49cad/generated.go"),
									Stringability:       5,
								},
							},
						},
					},
				},
			},
		},
	}
}()
