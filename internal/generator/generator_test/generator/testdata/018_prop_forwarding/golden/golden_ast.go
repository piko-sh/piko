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
								TextContent: "Prop Forwarding Test",
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
							OriginalPackageAlias: new("partials_outer_796b96dd"),
							OriginalSourcePath:   new("partials/outer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "outer_wrapper_label_state_myformfieldlabel_bf2d489e",
								PartialAlias:        "outer_wrapper",
								PartialPackageName:  "partials_outer_796b96dd",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"label": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
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
												Name: "MyFormFieldLabel",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MyFormFieldLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "MyFormFieldLabel",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 46,
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
													CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MyFormFieldLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 46,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MyFormFieldLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
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
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 46,
										},
										GoFieldName: "Label",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MyFormFieldLabel",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MyFormFieldLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 46,
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
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"label": "pages_main_594861c5",
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
								OriginalSourcePath: new("partials/outer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "outer-wrapper",
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
								Name:          "label",
								RawExpression: "state.MyFormFieldLabel",
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
												CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
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
										Name: "MyFormFieldLabel",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MyFormFieldLabel",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
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
											CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "MyFormFieldLabel",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 46,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   24,
									Column: 46,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 38,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "MyFormFieldLabel",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 46,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   23,
									Column: 5,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_outer_796b96dd"),
									OriginalSourcePath:   new("partials/outer.pk"),
									IsStatic:             true,
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
										OriginalSourcePath: new("partials/outer.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 8,
										},
										TextContent: "This is a wrapper component.",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_outer_796b96dd"),
											OriginalSourcePath:   new("partials/outer.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 8,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/outer.pk"),
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
									OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
									OriginalSourcePath:   new("partials/inner.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "inner_component_outer_wrapper_label_state_myformfieldlabel_bf2d489e_label_props_label_5fd9b107",
										PartialAlias:        "inner_component",
										PartialPackageName:  "partials_inner_8c3dbfdd",
										InvokerPackageAlias: "partials_outer_796b96dd",
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"label": ast_domain.PropValue{
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_outer_796b96dd.Props"),
																PackageAlias:         "partials_outer_796b96dd",
																CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
															OriginalSourcePath: new("partials/outer.pk"),
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
																PackageAlias:         "partials_outer_796b96dd",
																CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_outer_796b96dd",
																	CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 48,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
															},
															BaseCodeGenVarName:  new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
															OriginalSourcePath:  new("partials/outer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
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
															PackageAlias:         "partials_outer_796b96dd",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 48,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_outer_796b96dd",
																CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 48,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
														},
														BaseCodeGenVarName:  new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
														OriginalSourcePath:  new("partials/outer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   24,
													Column: 48,
												},
												GoFieldName: "Label",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_outer_796b96dd",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_outer_796b96dd",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 48,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
													},
													BaseCodeGenVarName:  new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
													OriginalSourcePath:  new("partials/outer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"label": "partials_outer_796b96dd",
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
										OriginalSourcePath: new("partials/inner.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "inner-component",
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
										Name:          "label",
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
														TypeExpression:       typeExprFromString("partials_outer_796b96dd.Props"),
														PackageAlias:         "partials_outer_796b96dd",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
													OriginalSourcePath: new("partials/outer.pk"),
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
														PackageAlias:         "partials_outer_796b96dd",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 48,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "MyFormFieldLabel",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 46,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
													OriginalSourcePath:  new("partials/outer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
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
													PackageAlias:         "partials_outer_796b96dd",
													CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 48,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MyFormFieldLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
												OriginalSourcePath:  new("partials/outer.pk"),
												GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 48,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 40,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_outer_796b96dd",
												CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_outer_796b96dd",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Label",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 48,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MyFormFieldLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 46,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("props_outer_wrapper_label_state_myformfieldlabel_bf2d489e"),
											OriginalSourcePath:  new("partials/outer.pk"),
											GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
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
										TagName: "label",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
											OriginalSourcePath:   new("partials/inner.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 12,
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
															TypeExpression:       typeExprFromString("partials_inner_8c3dbfdd.Props"),
															PackageAlias:         "partials_inner_8c3dbfdd",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_inner_8c3dbfdd",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props_inner_component_outer_wrapper_label_state_myformfieldlabel_bf2d489e_label_props_label_5fd9b107"),
														OriginalSourcePath: new("partials/inner.pk"),
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
															PackageAlias:         "partials_inner_8c3dbfdd",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_inner_8c3dbfdd",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   33,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "MyFormFieldLabel",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 46,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_inner_component_outer_wrapper_label_state_myformfieldlabel_bf2d489e_label_props_label_5fd9b107"),
														OriginalSourcePath:  new("partials/inner.pk"),
														GeneratedSourcePath: new("dist/partials/partials_inner_8c3dbfdd/generated.go"),
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
														PackageAlias:         "partials_inner_8c3dbfdd",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_inner_8c3dbfdd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   33,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "MyFormFieldLabel",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 46,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_inner_component_outer_wrapper_label_state_myformfieldlabel_bf2d489e_label_props_label_5fd9b107"),
													OriginalSourcePath:  new("partials/inner.pk"),
													GeneratedSourcePath: new("dist/partials/partials_inner_8c3dbfdd/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_inner_8c3dbfdd",
													CanonicalPackagePath: "testcase_018_prop_forwarding/dist/partials/partials_inner_8c3dbfdd",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   33,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_018_prop_forwarding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MyFormFieldLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("props_inner_component_outer_wrapper_label_state_myformfieldlabel_bf2d489e_label_props_label_5fd9b107"),
												OriginalSourcePath:  new("partials/inner.pk"),
												GeneratedSourcePath: new("dist/partials/partials_inner_8c3dbfdd/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:0",
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
												OriginalSourcePath: new("partials/inner.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										TagName: "input",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
											OriginalSourcePath:   new("partials/inner.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:1",
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
												OriginalSourcePath: new("partials/inner.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "type",
												Value: "text",
												Location: ast_domain.Location{
													Line:   24,
													Column: 18,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 12,
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
