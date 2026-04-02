package partial_external_pkg_import_deep_test

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
						Value: "page",
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
								TextContent: "Deep Nesting Test",
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
							OriginalPackageAlias: new("partials_level1_d7f1f2b7"),
							OriginalSourcePath:   new("partials/level1.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "level1_content_hello_deep_world_c6871fc3",
								PartialAlias:        "level1",
								PartialPackageName:  "partials_level1_d7f1f2b7",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"content": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "hello deep world",
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
											Column: 40,
										},
										GoFieldName: "Content",
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
								OriginalSourcePath: new("partials/level1.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "level1",
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
								Name:  "content",
								Value: "hello deep world",
								Location: ast_domain.Location{
									Line:   24,
									Column: 40,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 31,
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
									OriginalPackageAlias: new("partials_level2_a0cb46b5"),
									OriginalSourcePath:   new("partials/level2.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf",
										PartialAlias:        "level2",
										PartialPackageName:  "partials_level2_a0cb46b5",
										InvokerPackageAlias: "partials_level1_d7f1f2b7",
										Location: ast_domain.Location{
											Line:   23,
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
																TypeExpression:       typeExprFromString("partials_level1_d7f1f2b7.Props"),
																PackageAlias:         "partials_level1_d7f1f2b7",
																CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_level1_content_hello_deep_world_c6871fc3"),
															OriginalSourcePath: new("partials/level1.pk"),
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
																PackageAlias:         "partials_level1_d7f1f2b7",
																CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Content",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level1_d7f1f2b7",
																	CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Content",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 38,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   37,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_level1_content_hello_deep_world_c6871fc3"),
															},
															BaseCodeGenVarName:  new("props_level1_content_hello_deep_world_c6871fc3"),
															OriginalSourcePath:  new("partials/level1.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level1_d7f1f2b7/generated.go"),
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
															PackageAlias:         "partials_level1_d7f1f2b7",
															CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Content",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 38,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_level1_d7f1f2b7",
																CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Content",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("props_level1_content_hello_deep_world_c6871fc3"),
														},
														BaseCodeGenVarName:  new("props_level1_content_hello_deep_world_c6871fc3"),
														OriginalSourcePath:  new("partials/level1.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level1_d7f1f2b7/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   23,
													Column: 38,
												},
												GoFieldName: "Text",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level1_d7f1f2b7",
														CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Content",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_level1_d7f1f2b7",
															CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Content",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 38,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_level1_content_hello_deep_world_c6871fc3"),
													},
													BaseCodeGenVarName:  new("props_level1_content_hello_deep_world_c6871fc3"),
													OriginalSourcePath:  new("partials/level1.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level1_d7f1f2b7/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"text": "partials_level1_d7f1f2b7",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
										OriginalSourcePath: new("partials/level2.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "level2",
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
										Name:          "text",
										RawExpression: "props.Content",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_level1_d7f1f2b7.Props"),
														PackageAlias:         "partials_level1_d7f1f2b7",
														CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_level1_content_hello_deep_world_c6871fc3"),
													OriginalSourcePath: new("partials/level1.pk"),
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
														PackageAlias:         "partials_level1_d7f1f2b7",
														CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Content",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_level1_content_hello_deep_world_c6871fc3"),
													OriginalSourcePath:  new("partials/level1.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level1_d7f1f2b7/generated.go"),
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
													PackageAlias:         "partials_level1_d7f1f2b7",
													CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Content",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 38,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_level1_content_hello_deep_world_c6871fc3"),
												OriginalSourcePath:  new("partials/level1.pk"),
												GeneratedSourcePath: new("dist/partials/partials_level1_d7f1f2b7/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   23,
											Column: 38,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 31,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_level1_d7f1f2b7",
												CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level1_d7f1f2b7",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Content",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 38,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_level1_content_hello_deep_world_c6871fc3"),
											OriginalSourcePath:  new("partials/level1.pk"),
											GeneratedSourcePath: new("dist/partials/partials_level1_d7f1f2b7/generated.go"),
											Stringability:       1,
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
											OriginalPackageAlias: new("partials_level3_b25210fb"),
											OriginalSourcePath:   new("partials/level3.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "level3_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf_message_props_text_ef3bd93e",
												PartialAlias:        "level3",
												PartialPackageName:  "partials_level3_b25210fb",
												InvokerPackageAlias: "partials_level2_a0cb46b5",
												Location: ast_domain.Location{
													Line:   23,
													Column: 5,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"message": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_level2_a0cb46b5.Props"),
																		PackageAlias:         "partials_level2_a0cb46b5",
																		CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 41,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
																	OriginalSourcePath: new("partials/level2.pk"),
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
																		PackageAlias:         "partials_level2_a0cb46b5",
																		CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Text",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 41,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_level2_a0cb46b5",
																			CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Text",
																			ReferenceLocation: ast_domain.Location{
																				Line:   23,
																				Column: 41,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   38,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
																	},
																	BaseCodeGenVarName:  new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
																	OriginalSourcePath:  new("partials/level2.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_level2_a0cb46b5/generated.go"),
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
																	PackageAlias:         "partials_level2_a0cb46b5",
																	CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Text",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_level2_a0cb46b5",
																		CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Text",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 41,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   38,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
																},
																BaseCodeGenVarName:  new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
																OriginalSourcePath:  new("partials/level2.pk"),
																GeneratedSourcePath: new("dist/partials/partials_level2_a0cb46b5/generated.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   23,
															Column: 41,
														},
														GoFieldName: "Message",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_level2_a0cb46b5",
																CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Text",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_level2_a0cb46b5",
																	CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Text",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   38,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
															},
															BaseCodeGenVarName:  new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
															OriginalSourcePath:  new("partials/level2.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level2_a0cb46b5/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											DynamicAttributeOrigins: map[string]string{
												"message": "partials_level2_a0cb46b5",
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
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
												OriginalSourcePath: new("partials/level3.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "level3",
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
												Name:          "message",
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
																TypeExpression:       typeExprFromString("partials_level2_a0cb46b5.Props"),
																PackageAlias:         "partials_level2_a0cb46b5",
																CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
															OriginalSourcePath: new("partials/level2.pk"),
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
																PackageAlias:         "partials_level2_a0cb46b5",
																CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Text",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
															OriginalSourcePath:  new("partials/level2.pk"),
															GeneratedSourcePath: new("dist/partials/partials_level2_a0cb46b5/generated.go"),
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
															PackageAlias:         "partials_level2_a0cb46b5",
															CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Text",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
														OriginalSourcePath:  new("partials/level2.pk"),
														GeneratedSourcePath: new("dist/partials/partials_level2_a0cb46b5/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   23,
													Column: 41,
												},
												NameLocation: ast_domain.Location{
													Line:   23,
													Column: 31,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_level2_a0cb46b5",
														CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level2_a0cb46b5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Text",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf"),
													OriginalSourcePath:  new("partials/level2.pk"),
													GeneratedSourcePath: new("dist/partials/partials_level2_a0cb46b5/generated.go"),
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
													OriginalPackageAlias: new("partials_level3_b25210fb"),
													OriginalSourcePath:   new("partials/level3.pk"),
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
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
														OriginalSourcePath: new("partials/level3.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   23,
															Column: 11,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_level3_b25210fb"),
															OriginalSourcePath:   new("partials/level3.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:0:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 11,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/level3.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   23,
																	Column: 14,
																},
																RawExpression: "formatter.FormatDeep(props.Message)",
																Expression: &ast_domain.CallExpression{
																	Callee: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "formatter",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       nil,
																					PackageAlias:         "formatter",
																					CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/pkg/formatter",
																				},
																				BaseCodeGenVarName: new("formatter"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "FormatDeep",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 11,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("function"),
																					PackageAlias:         "formatter",
																					CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/pkg/formatter",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "FormatDeep",
																					ReferenceLocation: ast_domain.Location{
																						Line:   23,
																						Column: 14,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("formatter"),
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
																				PackageAlias:         "formatter",
																				CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/pkg/formatter",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "FormatDeep",
																				ReferenceLocation: ast_domain.Location{
																					Line:   23,
																					Column: 14,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("formatter"),
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
																						TypeExpression:       typeExprFromString("partials_level3_b25210fb.Props"),
																						PackageAlias:         "partials_level3_b25210fb",
																						CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level3_b25210fb",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "props",
																						ReferenceLocation: ast_domain.Location{
																							Line:   23,
																							Column: 14,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("props_level3_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf_message_props_text_ef3bd93e"),
																					OriginalSourcePath: new("partials/level3.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Message",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 28,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "partials_level3_b25210fb",
																						CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level3_b25210fb",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Message",
																						ReferenceLocation: ast_domain.Location{
																							Line:   23,
																							Column: 14,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   35,
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
																					BaseCodeGenVarName:  new("props_level3_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf_message_props_text_ef3bd93e"),
																					OriginalSourcePath:  new("partials/level3.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_level3_b25210fb/generated.go"),
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
																					PackageAlias:         "partials_level3_b25210fb",
																					CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/dist/partials/partials_level3_b25210fb",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Message",
																					ReferenceLocation: ast_domain.Location{
																						Line:   23,
																						Column: 14,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   35,
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
																				BaseCodeGenVarName:  new("props_level3_level2_level1_content_hello_deep_world_c6871fc3_text_props_content_af16c4bf_message_props_text_ef3bd93e"),
																				OriginalSourcePath:  new("partials/level3.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_level3_b25210fb/generated.go"),
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
																			PackageAlias:         "formatter",
																			CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/pkg/formatter",
																		},
																		BaseCodeGenVarName: new("formatter"),
																		OriginalSourcePath: new("partials/level3.pk"),
																		Stringability:      1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "formatter",
																		CanonicalPackagePath: "testcase_099_partial_external_pkg_import_deep_nesting/pkg/formatter",
																	},
																	BaseCodeGenVarName: new("formatter"),
																	OriginalSourcePath: new("partials/level3.pk"),
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
						},
					},
				},
			},
		},
	}
}()
