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
					Line:   42,
					Column: 3,
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
						Line:   42,
						Column: 3,
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
							Line:   38,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_data_display_2ee457b2"),
							OriginalSourcePath:   new("partials/data_display.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "data_display_30d06343",
								PartialAlias:        "data_display",
								PartialPackageName:  "partials_data_display_2ee457b2",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   43,
									Column: 5,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   38,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/data_display.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "data-display",
								Location: ast_domain.Location{
									Line:   38,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   38,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   39,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_data_display_2ee457b2"),
									OriginalSourcePath:   new("partials/data_display.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   39,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   39,
										Column: 10,
									},
									RawExpression: "props.Data != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_data_display_2ee457b2.Props"),
														PackageAlias:         "partials_data_display_2ee457b2",
														CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_data_display_30d06343"),
													OriginalSourcePath: new("partials/data_display.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Data",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*partials_data_display_2ee457b2.Data"),
														PackageAlias:         "partials_data_display_2ee457b2",
														CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_data_display_30d06343"),
													OriginalSourcePath:  new("partials/data_display.pk"),
													GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
													TypeExpression:       typeExprFromString("*partials_data_display_2ee457b2.Data"),
													PackageAlias:         "partials_data_display_2ee457b2",
													CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   30,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_data_display_30d06343"),
												OriginalSourcePath:  new("partials/data_display.pk"),
												GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 15,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/data_display.pk"),
												Stringability:      1,
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/data_display.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/data_display.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										OriginalSourcePath: new("partials/data_display.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   40,
											Column: 7,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_data_display_2ee457b2"),
											OriginalSourcePath:   new("partials/data_display.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   40,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/data_display.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "id",
												Value: "partial-internal-guard",
												Location: ast_domain.Location{
													Line:   40,
													Column: 17,
												},
												NameLocation: ast_domain.Location{
													Line:   40,
													Column: 13,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   40,
													Column: 41,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_display_2ee457b2"),
													OriginalSourcePath:   new("partials/data_display.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   40,
														Column: 41,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/data_display.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   40,
															Column: 44,
														},
														RawExpression: "props.Data.Value",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "props",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_data_display_2ee457b2.Props"),
																			PackageAlias:         "partials_data_display_2ee457b2",
																			CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   40,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("props_data_display_30d06343"),
																		OriginalSourcePath: new("partials/data_display.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Data",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*partials_data_display_2ee457b2.Data"),
																			PackageAlias:         "partials_data_display_2ee457b2",
																			CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   40,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_data_display_30d06343"),
																		OriginalSourcePath:  new("partials/data_display.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
																		TypeExpression:       typeExprFromString("*partials_data_display_2ee457b2.Data"),
																		PackageAlias:         "partials_data_display_2ee457b2",
																		CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Data",
																		ReferenceLocation: ast_domain.Location{
																			Line:   40,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_data_display_30d06343"),
																	OriginalSourcePath:  new("partials/data_display.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_data_display_2ee457b2",
																		CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   40,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   28,
																			Column: 19,
																		},
																	},
																	BaseCodeGenVarName:  new("props_data_display_30d06343"),
																	OriginalSourcePath:  new("partials/data_display.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
																	PackageAlias:         "partials_data_display_2ee457b2",
																	CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   28,
																		Column: 19,
																	},
																},
																BaseCodeGenVarName:  new("props_data_display_30d06343"),
																OriginalSourcePath:  new("partials/data_display.pk"),
																GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_data_display_2ee457b2",
																CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   28,
																	Column: 19,
																},
															},
															BaseCodeGenVarName:  new("props_data_display_30d06343"),
															OriginalSourcePath:  new("partials/data_display.pk"),
															GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
									Line:   42,
									Column: 5,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_data_display_2ee457b2"),
									OriginalSourcePath:   new("partials/data_display.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   42,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/data_display.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "partial-unguarded",
										Location: ast_domain.Location{
											Line:   42,
											Column: 15,
										},
										NameLocation: ast_domain.Location{
											Line:   42,
											Column: 11,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   42,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_data_display_2ee457b2"),
											OriginalSourcePath:   new("partials/data_display.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   42,
												Column: 34,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/data_display.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   42,
													Column: 37,
												},
												RawExpression: "props.Data.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "props",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_data_display_2ee457b2.Props"),
																	PackageAlias:         "partials_data_display_2ee457b2",
																	CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   42,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("props_data_display_30d06343"),
																OriginalSourcePath: new("partials/data_display.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Data",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*partials_data_display_2ee457b2.Data"),
																	PackageAlias:         "partials_data_display_2ee457b2",
																	CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   42,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   30,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_data_display_30d06343"),
																OriginalSourcePath:  new("partials/data_display.pk"),
																GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
																TypeExpression:       typeExprFromString("*partials_data_display_2ee457b2.Data"),
																PackageAlias:         "partials_data_display_2ee457b2",
																CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_data_display_30d06343"),
															OriginalSourcePath:  new("partials/data_display.pk"),
															GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_data_display_2ee457b2",
																CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   28,
																	Column: 19,
																},
															},
															BaseCodeGenVarName:  new("props_data_display_30d06343"),
															OriginalSourcePath:  new("partials/data_display.pk"),
															GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
															PackageAlias:         "partials_data_display_2ee457b2",
															CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   28,
																Column: 19,
															},
														},
														BaseCodeGenVarName:  new("props_data_display_30d06343"),
														OriginalSourcePath:  new("partials/data_display.pk"),
														GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_data_display_2ee457b2",
														CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/partials/partials_data_display_2ee457b2",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   28,
															Column: 19,
														},
													},
													BaseCodeGenVarName:  new("props_data_display_30d06343"),
													OriginalSourcePath:  new("partials/data_display.pk"),
													GeneratedSourcePath: new("dist/partials/partials_data_display_2ee457b2/generated.go"),
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
							Line:   45,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   45,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   45,
								Column: 10,
							},
							RawExpression: "props.Item != nil",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   45,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Item",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Item",
												ReferenceLocation: ast_domain.Location{
													Line:   45,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props"),
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
											TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Item",
											ReferenceLocation: ast_domain.Location{
												Line:   45,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   34,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								Operator: "!=",
								Right: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 15,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   45,
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
									Line:   34,
									Column: 3,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_slot_wrapper_82652e43"),
									OriginalSourcePath:   new("partials/slot_wrapper.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "slot_wrapper_7aa7a776",
										PartialAlias:        "slot_wrapper",
										PartialPackageName:  "partials_slot_wrapper_82652e43",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   46,
											Column: 7,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   34,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/slot_wrapper.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "slot-wrapper",
										Location: ast_domain.Location{
											Line:   34,
											Column: 15,
										},
										NameLocation: ast_domain.Location{
											Line:   34,
											Column: 8,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   47,
											Column: 9,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   47,
												Column: 9,
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
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "id",
												Value: "slot-in-guarded-context",
												Location: ast_domain.Location{
													Line:   47,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   47,
													Column: 15,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   47,
													Column: 44,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   47,
														Column: 44,
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
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   47,
															Column: 47,
														},
														RawExpression: "props.Item.Name",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "props",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("props"),
																		OriginalSourcePath: new("main.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   34,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
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
																		TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   34,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Name",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   32,
																			Column: 19,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
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
																	CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   32,
																		Column: 19,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 19,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   51,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   51,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   51,
								Column: 10,
							},
							RawExpression: "props.Item != nil",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Item",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Item",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props"),
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
											TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Item",
											ReferenceLocation: ast_domain.Location{
												Line:   51,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   34,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								Operator: "!=",
								Right: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 15,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   51,
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
									Line:   52,
									Column: 7,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   52,
										Column: 7,
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
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "main-guarded",
										Location: ast_domain.Location{
											Line:   52,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   52,
											Column: 13,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   52,
											Column: 31,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   52,
												Column: 31,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   52,
													Column: 34,
												},
												RawExpression: "props.Item.Name",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "props",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   52,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("props"),
																OriginalSourcePath: new("main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   52,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   34,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Item",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   34,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Name",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 19,
																},
															},
															BaseCodeGenVarName:  new("props"),
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
															CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 19,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 34,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 19,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
							Line:   54,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   54,
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "main-unguarded",
								Location: ast_domain.Location{
									Line:   54,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   54,
									Column: 11,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   54,
									Column: 31,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   54,
										Column: 31,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   54,
											Column: 34,
										},
										RawExpression: "props.Item.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   54,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Item",
															ReferenceLocation: ast_domain.Location{
																Line:   54,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   34,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
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
														TypeExpression:       typeExprFromString("*main_aaf9a2e0.Item"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Item",
														ReferenceLocation: ast_domain.Location{
															Line:   54,
															Column: 34,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   54,
															Column: 34,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 19,
														},
													},
													BaseCodeGenVarName:  new("props"),
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
													CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   54,
														Column: 34,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
														Column: 19,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_88_nil_guard_partials_slots/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   54,
													Column: 34,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   32,
													Column: 19,
												},
											},
											BaseCodeGenVarName:  new("props"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
	}
}()
