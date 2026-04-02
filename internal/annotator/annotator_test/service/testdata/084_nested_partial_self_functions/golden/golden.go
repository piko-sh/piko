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
					Line:   38,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_outer_796b96dd"),
					OriginalSourcePath:   new("partials/outer.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "outer_value_42_669718d0",
						PartialAlias:        "outer",
						PartialPackageName:  "partials_outer_796b96dd",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   34,
							Column: 2,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"value": ast_domain.PropValue{
								Expression: &ast_domain.IntegerLiteral{
									Value: 42,
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
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   34,
									Column: 42,
								},
								GoFieldName: "Value",
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
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   38,
						Column: 2,
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
						Value: "outer",
						Location: ast_domain.Location{
							Line:   38,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   38,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   39,
							Column: 3,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_outer_796b96dd"),
							OriginalSourcePath:   new("partials/outer.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   39,
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
								Value: "prefix",
								Location: ast_domain.Location{
									Line:   39,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 9,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   39,
									Column: 24,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_outer_796b96dd"),
									OriginalSourcePath:   new("partials/outer.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   39,
										Column: 24,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   39,
											Column: 27,
										},
										RawExpression: "OuterPrefix",
										Expression: &ast_domain.Identifier{
											Name: "OuterPrefix",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:          typeExprFromString("any"),
													PackageAlias:            "partials_outer_796b96dd",
													CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
													IsExportedPackageSymbol: true,
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "OuterPrefix",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 27,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("OuterPrefix"),
												OriginalSourcePath: new("partials/outer.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:          typeExprFromString("any"),
												PackageAlias:            "partials_outer_796b96dd",
												CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
												IsExportedPackageSymbol: true,
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "OuterPrefix",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 27,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("OuterPrefix"),
											OriginalSourcePath: new("partials/outer.pk"),
											Stringability:      1,
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
							Column: 3,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_outer_796b96dd"),
							OriginalSourcePath:   new("partials/outer.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   40,
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
								Name:  "id",
								Value: "outer-span",
								Location: ast_domain.Location{
									Line:   40,
									Column: 13,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 9,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   40,
									Column: 25,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_outer_796b96dd"),
									OriginalSourcePath:   new("partials/outer.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   40,
										Column: 25,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   40,
											Column: 28,
										},
										RawExpression: "FormatOuter(props.Value)",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.Identifier{
												Name: "FormatOuter",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:          typeExprFromString("function"),
														PackageAlias:            "partials_outer_796b96dd",
														CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
														IsExportedPackageSymbol: true,
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FormatOuter",
														ReferenceLocation: ast_domain.Location{
															Line:   40,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("FormatOuter"),
													OriginalSourcePath: new("partials/outer.pk"),
													Stringability:      1,
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 13,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_outer_796b96dd.Props"),
																PackageAlias:         "partials_outer_796b96dd",
																CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 28,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_outer_value_42_669718d0"),
															OriginalSourcePath: new("partials/outer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 19,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "partials_outer_796b96dd",
																CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 28,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   34,
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
															BaseCodeGenVarName:  new("props_outer_value_42_669718d0"),
															OriginalSourcePath:  new("partials/outer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
															Stringability:       1,
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 13,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_outer_796b96dd",
															CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   40,
																Column: 28,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   34,
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
														BaseCodeGenVarName:  new("props_outer_value_42_669718d0"),
														OriginalSourcePath:  new("partials/outer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
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
													PackageAlias:         "partials_outer_796b96dd",
													CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
												},
												BaseCodeGenVarName: new("FormatOuter"),
												OriginalSourcePath: new("partials/outer.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_outer_796b96dd",
												CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
											},
											BaseCodeGenVarName: new("FormatOuter"),
											OriginalSourcePath: new("partials/outer.pk"),
											Stringability:      1,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   39,
							Column: 2,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
							OriginalSourcePath:   new("partials/inner.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "inner_outer_value_42_669718d0_value_props_value_aefe5006",
								PartialAlias:        "inner",
								PartialPackageName:  "partials_inner_8c3dbfdd",
								InvokerPackageAlias: "partials_outer_796b96dd",
								Location: ast_domain.Location{
									Line:   41,
									Column: 3,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"value": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 43,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_outer_value_42_669718d0"),
													OriginalSourcePath: new("partials/outer.pk"),
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
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "partials_outer_796b96dd",
														CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 43,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_outer_796b96dd",
															CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   41,
																Column: 43,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   34,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_outer_value_42_669718d0"),
													},
													BaseCodeGenVarName:  new("props_outer_value_42_669718d0"),
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
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_outer_796b96dd",
													CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Value",
													ReferenceLocation: ast_domain.Location{
														Line:   41,
														Column: 43,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   34,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "partials_outer_796b96dd",
														CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 43,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("props_outer_value_42_669718d0"),
												},
												BaseCodeGenVarName:  new("props_outer_value_42_669718d0"),
												OriginalSourcePath:  new("partials/outer.pk"),
												GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   41,
											Column: 43,
										},
										GoFieldName: "Value",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "partials_outer_796b96dd",
												CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Value",
												ReferenceLocation: ast_domain.Location{
													Line:   41,
													Column: 43,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_outer_796b96dd",
													CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_outer_796b96dd",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Value",
													ReferenceLocation: ast_domain.Location{
														Line:   41,
														Column: 43,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   34,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("props_outer_value_42_669718d0"),
											},
											BaseCodeGenVarName:  new("props_outer_value_42_669718d0"),
											OriginalSourcePath:  new("partials/outer.pk"),
											GeneratedSourcePath: new("dist/partials/partials_outer_796b96dd/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   39,
								Column: 2,
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
								Value: "inner",
								Location: ast_domain.Location{
									Line:   39,
									Column: 14,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 7,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   40,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
									OriginalSourcePath:   new("partials/inner.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   40,
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
										Value: "prefix",
										Location: ast_domain.Location{
											Line:   40,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   40,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   40,
											Column: 24,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
											OriginalSourcePath:   new("partials/inner.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   40,
												Column: 24,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   40,
													Column: 27,
												},
												RawExpression: "InnerPrefix",
												Expression: &ast_domain.Identifier{
													Name: "InnerPrefix",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:          typeExprFromString("any"),
															PackageAlias:            "partials_inner_8c3dbfdd",
															CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
															IsExportedPackageSymbol: true,
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "InnerPrefix",
															ReferenceLocation: ast_domain.Location{
																Line:   40,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("InnerPrefix"),
														OriginalSourcePath: new("partials/inner.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:          typeExprFromString("any"),
														PackageAlias:            "partials_inner_8c3dbfdd",
														CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
														IsExportedPackageSymbol: true,
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "InnerPrefix",
														ReferenceLocation: ast_domain.Location{
															Line:   40,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("InnerPrefix"),
													OriginalSourcePath: new("partials/inner.pk"),
													Stringability:      1,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   41,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
									OriginalSourcePath:   new("partials/inner.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:1",
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
										OriginalSourcePath: new("partials/inner.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "value",
										Location: ast_domain.Location{
											Line:   41,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   41,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   41,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
											OriginalSourcePath:   new("partials/inner.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 23,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   41,
													Column: 26,
												},
												RawExpression: "FormatInner(props.Value)",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.Identifier{
														Name: "FormatInner",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:          typeExprFromString("function"),
																PackageAlias:            "partials_inner_8c3dbfdd",
																CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
																IsExportedPackageSymbol: true,
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FormatInner",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("FormatInner"),
															OriginalSourcePath: new("partials/inner.pk"),
															Stringability:      1,
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 13,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_inner_8c3dbfdd.Props"),
																		PackageAlias:         "partials_inner_8c3dbfdd",
																		CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   41,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("props_inner_outer_value_42_669718d0_value_props_value_aefe5006"),
																	OriginalSourcePath: new("partials/inner.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 19,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "partials_inner_8c3dbfdd",
																		CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   41,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   33,
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
																	BaseCodeGenVarName:  new("props_inner_outer_value_42_669718d0_value_props_value_aefe5006"),
																	OriginalSourcePath:  new("partials/inner.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_inner_8c3dbfdd/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "partials_inner_8c3dbfdd",
																	CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   41,
																		Column: 26,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   33,
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
																BaseCodeGenVarName:  new("props_inner_outer_value_42_669718d0_value_props_value_aefe5006"),
																OriginalSourcePath:  new("partials/inner.pk"),
																GeneratedSourcePath: new("dist/partials/partials_inner_8c3dbfdd/generated.go"),
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
															PackageAlias:         "partials_inner_8c3dbfdd",
															CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
														},
														BaseCodeGenVarName: new("FormatInner"),
														OriginalSourcePath: new("partials/inner.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_inner_8c3dbfdd",
														CanonicalPackagePath: "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
													},
													BaseCodeGenVarName: new("FormatInner"),
													OriginalSourcePath: new("partials/inner.pk"),
													Stringability:      1,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   42,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
									OriginalSourcePath:   new("partials/inner.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:2",
									RelativeLocation: ast_domain.Location{
										Line:   42,
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
										Value: "suffix",
										Location: ast_domain.Location{
											Line:   42,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   42,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   42,
											Column: 24,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_8c3dbfdd"),
											OriginalSourcePath:   new("partials/inner.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   42,
												Column: 24,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   42,
													Column: 27,
												},
												RawExpression: "InnerSuffix",
												Expression: &ast_domain.Identifier{
													Name: "InnerSuffix",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:          typeExprFromString("any"),
															PackageAlias:            "partials_inner_8c3dbfdd",
															CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
															IsExportedPackageSymbol: true,
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "InnerSuffix",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("InnerSuffix"),
														OriginalSourcePath: new("partials/inner.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:          typeExprFromString("any"),
														PackageAlias:            "partials_inner_8c3dbfdd",
														CanonicalPackagePath:    "testcase_84_nested_partial_self_functions/dist/partials/partials_inner_8c3dbfdd",
														IsExportedPackageSymbol: true,
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "InnerSuffix",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("InnerSuffix"),
													OriginalSourcePath: new("partials/inner.pk"),
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
	}
}()
