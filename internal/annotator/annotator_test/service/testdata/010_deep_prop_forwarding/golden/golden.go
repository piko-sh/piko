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
					Line:   39,
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
						Line:   39,
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
							Line:   32,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_component_b_88e00e3c"),
							OriginalSourcePath:   new("partials/component_b.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "compb_data_from_a_state_topleveldata_99cc6b47",
								PartialAlias:        "compB",
								PartialPackageName:  "partials_component_b_88e00e3c",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   40,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"data-from-a": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   40,
															Column: 48,
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
												Name: "TopLevelData",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TopLevelData",
														ReferenceLocation: ast_domain.Location{
															Line:   40,
															Column: 48,
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
															CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TopLevelData",
															ReferenceLocation: ast_domain.Location{
																Line:   40,
																Column: 48,
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
													CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "TopLevelData",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 48,
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
														CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TopLevelData",
														ReferenceLocation: ast_domain.Location{
															Line:   40,
															Column: 48,
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
											Line:   40,
											Column: 48,
										},
										GoFieldName: "DataFromA",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "TopLevelData",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 48,
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
													CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "TopLevelData",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 48,
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
								"data-from-a": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   32,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/component_b.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "component-b",
								Location: ast_domain.Location{
									Line:   32,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   32,
									Column: 10,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "data-from-a",
								RawExpression: "state.TopLevelData",
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
												CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 48,
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
										Name: "TopLevelData",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "TopLevelData",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 48,
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
											CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "TopLevelData",
											ReferenceLocation: ast_domain.Location{
												Line:   40,
												Column: 48,
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
									Line:   40,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "TopLevelData",
										ReferenceLocation: ast_domain.Location{
											Line:   40,
											Column: 48,
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
									Line:   30,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_component_c_506c1d39"),
									OriginalSourcePath:   new("partials/component_c.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "compc_compb_data_from_a_state_topleveldata_99cc6b47_data_from_b_props_datafroma_813321b9",
										PartialAlias:        "compC",
										PartialPackageName:  "partials_component_c_506c1d39",
										InvokerPackageAlias: "partials_component_b_88e00e3c",
										Location: ast_domain.Location{
											Line:   33,
											Column: 9,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"data-from-b": ast_domain.PropValue{
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_component_b_88e00e3c.Props"),
																PackageAlias:         "partials_component_b_88e00e3c",
																CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
															OriginalSourcePath: new("partials/component_b.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "DataFromA",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_component_b_88e00e3c",
																CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DataFromA",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   31,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_component_b_88e00e3c",
																	CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DataFromA",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 48,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   31,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
															},
															BaseCodeGenVarName:  new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
															OriginalSourcePath:  new("partials/component_b.pk"),
															GeneratedSourcePath: new("dist/partials/partials_component_b_88e00e3c/generated.go"),
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
															PackageAlias:         "partials_component_b_88e00e3c",
															CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DataFromA",
															ReferenceLocation: ast_domain.Location{
																Line:   33,
																Column: 48,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   31,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_component_b_88e00e3c",
																CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DataFromA",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   31,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
														},
														BaseCodeGenVarName:  new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
														OriginalSourcePath:  new("partials/component_b.pk"),
														GeneratedSourcePath: new("dist/partials/partials_component_b_88e00e3c/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   33,
													Column: 48,
												},
												GoFieldName: "DataFromB",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_component_b_88e00e3c",
														CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DataFromA",
														ReferenceLocation: ast_domain.Location{
															Line:   33,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_component_b_88e00e3c",
															CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DataFromA",
															ReferenceLocation: ast_domain.Location{
																Line:   33,
																Column: 48,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   31,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
													},
													BaseCodeGenVarName:  new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
													OriginalSourcePath:  new("partials/component_b.pk"),
													GeneratedSourcePath: new("dist/partials/partials_component_b_88e00e3c/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"data-from-b": "partials_component_b_88e00e3c",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   30,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/component_c.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "component-c",
										Location: ast_domain.Location{
											Line:   30,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   30,
											Column: 10,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "data-from-b",
										RawExpression: "props.DataFromA",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_component_b_88e00e3c.Props"),
														PackageAlias:         "partials_component_b_88e00e3c",
														CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   33,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
													OriginalSourcePath: new("partials/component_b.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "DataFromA",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_component_b_88e00e3c",
														CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DataFromA",
														ReferenceLocation: ast_domain.Location{
															Line:   33,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
													OriginalSourcePath:  new("partials/component_b.pk"),
													GeneratedSourcePath: new("dist/partials/partials_component_b_88e00e3c/generated.go"),
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
													PackageAlias:         "partials_component_b_88e00e3c",
													CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DataFromA",
													ReferenceLocation: ast_domain.Location{
														Line:   33,
														Column: 48,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
												OriginalSourcePath:  new("partials/component_b.pk"),
												GeneratedSourcePath: new("dist/partials/partials_component_b_88e00e3c/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   33,
											Column: 48,
										},
										NameLocation: ast_domain.Location{
											Line:   33,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_component_b_88e00e3c",
												CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_b_88e00e3c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DataFromA",
												ReferenceLocation: ast_domain.Location{
													Line:   33,
													Column: 48,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_compb_data_from_a_state_topleveldata_99cc6b47"),
											OriginalSourcePath:  new("partials/component_b.pk"),
											GeneratedSourcePath: new("dist/partials/partials_component_b_88e00e3c/generated.go"),
											Stringability:       1,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   31,
											Column: 9,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_component_c_506c1d39"),
											OriginalSourcePath:   new("partials/component_c.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   31,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/component_c.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "id",
												Value: "final-output",
												Location: ast_domain.Location{
													Line:   31,
													Column: 16,
												},
												NameLocation: ast_domain.Location{
													Line:   31,
													Column: 12,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   31,
													Column: 30,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_component_c_506c1d39"),
													OriginalSourcePath:   new("partials/component_c.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   31,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/component_c.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   31,
															Column: 33,
														},
														RawExpression: "props.DataFromB",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_component_c_506c1d39.Props"),
																		PackageAlias:         "partials_component_c_506c1d39",
																		CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_c_506c1d39",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 33,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("props_compc_compb_data_from_a_state_topleveldata_99cc6b47_data_from_b_props_datafroma_813321b9"),
																	OriginalSourcePath: new("partials/component_c.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "DataFromB",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_component_c_506c1d39",
																		CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_c_506c1d39",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "DataFromB",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 33,
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
																			CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TopLevelData",
																			ReferenceLocation: ast_domain.Location{
																				Line:   40,
																				Column: 48,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName: new("pageData"),
																	},
																	BaseCodeGenVarName:  new("props_compc_compb_data_from_a_state_topleveldata_99cc6b47_data_from_b_props_datafroma_813321b9"),
																	OriginalSourcePath:  new("partials/component_c.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_component_c_506c1d39/generated.go"),
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
																	PackageAlias:         "partials_component_c_506c1d39",
																	CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_c_506c1d39",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DataFromB",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 33,
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
																		CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TopLevelData",
																		ReferenceLocation: ast_domain.Location{
																			Line:   40,
																			Column: 48,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																},
																BaseCodeGenVarName:  new("props_compc_compb_data_from_a_state_topleveldata_99cc6b47_data_from_b_props_datafroma_813321b9"),
																OriginalSourcePath:  new("partials/component_c.pk"),
																GeneratedSourcePath: new("dist/partials/partials_component_c_506c1d39/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_component_c_506c1d39",
																CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/partials/partials_component_c_506c1d39",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DataFromB",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 33,
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
																	CanonicalPackagePath: "testcase_10_deep_prop_forwarding/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "TopLevelData",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
																		Column: 48,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_compc_compb_data_from_a_state_topleveldata_99cc6b47_data_from_b_props_datafroma_813321b9"),
															OriginalSourcePath:  new("partials/component_c.pk"),
															GeneratedSourcePath: new("dist/partials/partials_component_c_506c1d39/generated.go"),
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
	}
}()
