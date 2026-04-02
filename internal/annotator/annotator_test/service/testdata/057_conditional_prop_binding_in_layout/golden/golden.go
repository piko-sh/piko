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
					Line:   32,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_layout_ee037d9a"),
					OriginalSourcePath:   new("partials/layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "layout_1745aa65",
						PartialAlias:        "layout",
						PartialPackageName:  "partials_layout_ee037d9a",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   47,
							Column: 2,
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   32,
						Column: 2,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("partials/layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "layout",
						Location: ast_domain.Location{
							Line:   32,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   32,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   40,
							Column: 2,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_child_d247007e"),
							OriginalSourcePath:   new("partials/child.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "child_layout_1745aa65_data_state_childdata_4f10e4c9",
								PartialAlias:        "child",
								PartialPackageName:  "partials_child_d247007e",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   48,
									Column: 3,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"data": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
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
												Name: "ChildData",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ChildData",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   36,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ChildData",
															ReferenceLocation: ast_domain.Location{
																Line:   48,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   36,
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
													CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ChildData",
													ReferenceLocation: ast_domain.Location{
														Line:   48,
														Column: 69,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   36,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ChildData",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   36,
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
											Line:   48,
											Column: 69,
										},
										GoFieldName: "Data",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ChildData",
												ReferenceLocation: ast_domain.Location{
													Line:   48,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   36,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ChildData",
													ReferenceLocation: ast_domain.Location{
														Line:   48,
														Column: 69,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   36,
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
								"data": "main_aaf9a2e0",
							},
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   48,
								Column: 45,
							},
							NameLocation: ast_domain.Location{
								Line:   48,
								Column: 39,
							},
							RawExpression: "state.ShowChild",
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
											CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   48,
												Column: 45,
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
									Name: "ShowChild",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ShowChild",
											ReferenceLocation: ast_domain.Location{
												Line:   48,
												Column: 45,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   35,
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
										CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ShowChild",
										ReferenceLocation: ast_domain.Location{
											Line:   48,
											Column: 45,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   35,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "ShowChild",
									ReferenceLocation: ast_domain.Location{
										Line:   48,
										Column: 45,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   35,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								Stringability:       1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("partials/child.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "child",
								Location: ast_domain.Location{
									Line:   40,
									Column: 14,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 7,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "child",
								Location: ast_domain.Location{
									Line:   48,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   48,
									Column: 17,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "data",
								RawExpression: "state.ChildData",
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
												CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   48,
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
										Name: "ChildData",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ChildData",
												ReferenceLocation: ast_domain.Location{
													Line:   48,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   36,
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
											CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ChildData",
											ReferenceLocation: ast_domain.Location{
												Line:   48,
												Column: 69,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   36,
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
									Line:   48,
									Column: 69,
								},
								NameLocation: ast_domain.Location{
									Line:   48,
									Column: 62,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ChildData",
										ReferenceLocation: ast_domain.Location{
											Line:   48,
											Column: 69,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   36,
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
									Line:   41,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_child_d247007e"),
									OriginalSourcePath:   new("partials/child.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   41,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/child.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   41,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/child.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   41,
													Column: 12,
												},
												RawExpression: "state.Content",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_child_d247007e.Response"),
																PackageAlias:         "partials_child_d247007e",
																CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_child_d247007e",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_child_d247007eData_child_layout_1745aa65_data_state_childdata_4f10e4c9"),
															OriginalSourcePath: new("partials/child.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Content",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_child_d247007e",
																CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_child_d247007e",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Content",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   28,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_data_state_childdata_4f10e4c9"),
															OriginalSourcePath:  new("partials/child.pk"),
															GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
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
															PackageAlias:         "partials_child_d247007e",
															CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_child_d247007e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Content",
															ReferenceLocation: ast_domain.Location{
																Line:   41,
																Column: 12,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   28,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_data_state_childdata_4f10e4c9"),
														OriginalSourcePath:  new("partials/child.pk"),
														GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_child_d247007e",
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_child_d247007e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Content",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 12,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   28,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_data_state_childdata_4f10e4c9"),
													OriginalSourcePath:  new("partials/child.pk"),
													GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
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
							Line:   40,
							Column: 2,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_fallback_6397e174"),
							OriginalSourcePath:   new("partials/fallback.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "fallback_layout_1745aa65_message_state_fallbackmessage_70c27b1f",
								PartialAlias:        "fallback",
								PartialPackageName:  "partials_fallback_6397e174",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   49,
									Column: 3,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"message": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 62,
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
												Name: "FallbackMessage",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FallbackMessage",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 62,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FallbackMessage",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 62,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
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
													CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FallbackMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 62,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FallbackMessage",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 62,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
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
											Line:   49,
											Column: 62,
										},
										GoFieldName: "Message",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FallbackMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 62,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FallbackMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 62,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
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
								"message": "main_aaf9a2e0",
							},
						},
						DirElse: &ast_domain.Directive{
							Type: ast_domain.DirectiveElse,
							Location: ast_domain.Location{
								Line:   49,
								Column: 45,
							},
							NameLocation: ast_domain.Location{
								Line:   49,
								Column: 45,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								OriginalPackageAlias: new("main_aaf9a2e0"),
								OriginalSourcePath:   new("main.pk"),
							},
							ChainKey: &ast_domain.StringLiteral{
								Value: "r.0:0",
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
									OriginalSourcePath: new("partials/child.pk"),
									Stringability:      1,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("partials/fallback.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "fallback",
								Location: ast_domain.Location{
									Line:   40,
									Column: 14,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 7,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "fallback",
								Location: ast_domain.Location{
									Line:   49,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 17,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "message",
								RawExpression: "state.FallbackMessage",
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
												CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 62,
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
										Name: "FallbackMessage",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FallbackMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 62,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
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
											CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "FallbackMessage",
											ReferenceLocation: ast_domain.Location{
												Line:   49,
												Column: 62,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
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
									Line:   49,
									Column: 62,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 52,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "FallbackMessage",
										ReferenceLocation: ast_domain.Location{
											Line:   49,
											Column: 62,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
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
									Line:   41,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_fallback_6397e174"),
									OriginalSourcePath:   new("partials/fallback.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   41,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/fallback.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   41,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_fallback_6397e174"),
											OriginalSourcePath:   new("partials/fallback.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/fallback.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   41,
													Column: 12,
												},
												RawExpression: "state.Text",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_fallback_6397e174.Response"),
																PackageAlias:         "partials_fallback_6397e174",
																CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_fallback_6397e174",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_fallback_6397e174Data_fallback_layout_1745aa65_message_state_fallbackmessage_70c27b1f"),
															OriginalSourcePath: new("partials/fallback.pk"),
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
																PackageAlias:         "partials_fallback_6397e174",
																CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_fallback_6397e174",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Text",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   28,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_fallback_6397e174Data_fallback_layout_1745aa65_message_state_fallbackmessage_70c27b1f"),
															OriginalSourcePath:  new("partials/fallback.pk"),
															GeneratedSourcePath: new("dist/partials/partials_fallback_6397e174/generated.go"),
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
															PackageAlias:         "partials_fallback_6397e174",
															CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_fallback_6397e174",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Text",
															ReferenceLocation: ast_domain.Location{
																Line:   41,
																Column: 12,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   28,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_fallback_6397e174Data_fallback_layout_1745aa65_message_state_fallbackmessage_70c27b1f"),
														OriginalSourcePath:  new("partials/fallback.pk"),
														GeneratedSourcePath: new("dist/partials/partials_fallback_6397e174/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_fallback_6397e174",
														CanonicalPackagePath: "testcase_57_conditional_prop_binding_in_layout/dist/partials/partials_fallback_6397e174",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Text",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 12,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   28,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_fallback_6397e174Data_fallback_layout_1745aa65_message_state_fallbackmessage_70c27b1f"),
													OriginalSourcePath:  new("partials/fallback.pk"),
													GeneratedSourcePath: new("dist/partials/partials_fallback_6397e174/generated.go"),
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
