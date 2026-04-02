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
							Line:   23,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "target-field",
								Location: ast_domain.Location{
									Line:   23,
									Column: 16,
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
									Column: 30,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 30,
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
											Line:   23,
											Column: 33,
										},
										RawExpression: "state.Root.Item.Ranges[0].To",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 33,
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
																Name: "Root",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("domain.RootModal"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Root",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 33,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   46,
																			Column: 23,
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
																	TypeExpression:       typeExprFromString("domain.RootModal"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Root",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 33,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 12,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemDto"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 33,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/modal.go"),
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
																TypeExpression:       typeExprFromString("dto.ItemDto"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Item",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 33,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/modal.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Ranges",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 17,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]dto.RangeItem"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Ranges",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 33,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   71,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dto/item.go"),
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
															TypeExpression:       typeExprFromString("[]dto.RangeItem"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Ranges",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 33,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/item.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 0,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 24,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
												},
												Optional: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.RangeItem"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "To",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 27,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "To",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 33,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   63,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dto/item.go"),
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
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "To",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 33,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   63,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/item.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "To",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 33,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/item.go"),
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
							Line:   24,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   24,
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
								Value: "target-method",
								Location: ast_domain.Location{
									Line:   24,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   24,
									Column: 31,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   24,
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
											Line:   24,
											Column: 34,
										},
										RawExpression: "state.Root.Item.Ranges[0].Charge.MustNumber()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.IndexExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
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
																				CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 34,
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
																		Name: "Root",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("domain.RootModal"),
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Root",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 34,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   46,
																					Column: 23,
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
																			TypeExpression:       typeExprFromString("domain.RootModal"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Root",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 34,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   46,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemDto"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 34,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   61,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("domain/modal.go"),
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
																		TypeExpression:       typeExprFromString("dto.ItemDto"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 34,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   61,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/modal.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Ranges",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 17,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]dto.RangeItem"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Ranges",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 34,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   71,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dto/item.go"),
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
																	TypeExpression:       typeExprFromString("[]dto.RangeItem"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Ranges",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   71,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/item.go"),
															},
														},
														Index: &ast_domain.IntegerLiteral{
															Value: 0,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 24,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int64"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("main.pk"),
																Stringability:      1,
															},
														},
														Optional: false,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.RangeItem"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
															},
															BaseCodeGenVarName: new("pageData"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Charge",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 27,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("maths.Money"),
																PackageAlias:         "maths",
																CanonicalPackagePath: "piko.sh/piko/wdk/maths",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Charge",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   64,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dto/item.go"),
															Stringability:       4,
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
															TypeExpression:       typeExprFromString("maths.Money"),
															PackageAlias:         "maths",
															CanonicalPackagePath: "piko.sh/piko/wdk/maths",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Charge",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   64,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/item.go"),
														Stringability:       4,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "MustNumber",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 34,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "maths",
															CanonicalPackagePath: "piko.sh/piko/wdk/maths",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "MustNumber",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   217,
																Column: 1,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
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
														PackageAlias:         "maths",
														CanonicalPackagePath: "piko.sh/piko/wdk/maths",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MustNumber",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 34,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   217,
															Column: 1,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "maths",
													CanonicalPackagePath: "piko.sh/piko/wdk/maths",
												},
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "maths",
												CanonicalPackagePath: "piko.sh/piko/wdk/maths",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
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
							Line:   26,
							Column: 9,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   26,
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
								Value: "target-subexpressions",
								Location: ast_domain.Location{
									Line:   26,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   26,
									Column: 14,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "c",
								RawExpression: "state.Root.Item.Ranges[0].Charge",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.IndexExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
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
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 17,
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
														Name: "Root",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("domain.RootModal"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Root",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 23,
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
															TypeExpression:       typeExprFromString("domain.RootModal"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Root",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dto.ItemDto"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Item",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   61,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/modal.go"),
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
														TypeExpression:       typeExprFromString("dto.ItemDto"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Item",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   61,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/modal.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Ranges",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.RangeItem"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Ranges",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dto/item.go"),
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
													TypeExpression:       typeExprFromString("[]dto.RangeItem"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Ranges",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/item.go"),
											},
										},
										Index: &ast_domain.IntegerLiteral{
											Value: 0,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int64"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										Optional: false,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("dto.RangeItem"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											BaseCodeGenVarName: new("pageData"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Charge",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 27,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("maths.Money"),
												PackageAlias:         "maths",
												CanonicalPackagePath: "piko.sh/piko/wdk/maths",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Charge",
												ReferenceLocation: ast_domain.Location{
													Line:   32,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   64,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/item.go"),
											Stringability:       4,
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
											TypeExpression:       typeExprFromString("maths.Money"),
											PackageAlias:         "maths",
											CanonicalPackagePath: "piko.sh/piko/wdk/maths",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Charge",
											ReferenceLocation: ast_domain.Location{
												Line:   32,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   64,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dto/item.go"),
										Stringability:       4,
									},
								},
								Location: ast_domain.Location{
									Line:   32,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   32,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("maths.Money"),
										PackageAlias:         "maths",
										CanonicalPackagePath: "piko.sh/piko/wdk/maths",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Charge",
										ReferenceLocation: ast_domain.Location{
											Line:   32,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   64,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dto/item.go"),
									Stringability:       4,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "e",
								RawExpression: "state.Root.Item.Ranges[0]",
								Expression: &ast_domain.IndexExpression{
									Base: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
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
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 17,
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
													Name: "Root",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.RootModal"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Root",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 23,
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
														TypeExpression:       typeExprFromString("domain.RootModal"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Root",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Item",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.ItemDto"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Item",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   61,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/modal.go"),
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
													TypeExpression:       typeExprFromString("dto.ItemDto"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Item",
													ReferenceLocation: ast_domain.Location{
														Line:   30,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   61,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/modal.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Ranges",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 17,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.RangeItem"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Ranges",
													ReferenceLocation: ast_domain.Location{
														Line:   30,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/item.go"),
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
												TypeExpression:       typeExprFromString("[]dto.RangeItem"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Ranges",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/item.go"),
										},
									},
									Index: &ast_domain.IntegerLiteral{
										Value: 0,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 24,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
									Optional: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("dto.RangeItem"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
										},
										BaseCodeGenVarName: new("pageData"),
									},
								},
								Location: ast_domain.Location{
									Line:   30,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   30,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("dto.RangeItem"),
										PackageAlias:         "dto",
										CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
									},
									BaseCodeGenVarName: new("pageData"),
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "f",
								RawExpression: "state.Root.Item.Ranges[0].To",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.IndexExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
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
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 17,
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
														Name: "Root",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("domain.RootModal"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Root",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 23,
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
															TypeExpression:       typeExprFromString("domain.RootModal"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Root",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dto.ItemDto"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Item",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   61,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/modal.go"),
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
														TypeExpression:       typeExprFromString("dto.ItemDto"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Item",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   61,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/modal.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Ranges",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.RangeItem"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Ranges",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dto/item.go"),
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
													TypeExpression:       typeExprFromString("[]dto.RangeItem"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Ranges",
													ReferenceLocation: ast_domain.Location{
														Line:   31,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/item.go"),
											},
										},
										Index: &ast_domain.IntegerLiteral{
											Value: 0,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int64"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										Optional: false,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("dto.RangeItem"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											BaseCodeGenVarName: new("pageData"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "To",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 27,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int64"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "To",
												ReferenceLocation: ast_domain.Location{
													Line:   31,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/item.go"),
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
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "To",
											ReferenceLocation: ast_domain.Location{
												Line:   31,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   63,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dto/item.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   31,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   31,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int64"),
										PackageAlias:         "dto",
										CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "To",
										ReferenceLocation: ast_domain.Location{
											Line:   31,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   63,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dto/item.go"),
									Stringability:       1,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "i",
								RawExpression: "state.Root.Item",
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
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   28,
														Column: 17,
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
											Name: "Root",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("domain.RootModal"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Root",
													ReferenceLocation: ast_domain.Location{
														Line:   28,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 23,
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
												TypeExpression:       typeExprFromString("domain.RootModal"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Root",
												ReferenceLocation: ast_domain.Location{
													Line:   28,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Item",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("dto.ItemDto"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Item",
												ReferenceLocation: ast_domain.Location{
													Line:   28,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   61,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/modal.go"),
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
											TypeExpression:       typeExprFromString("dto.ItemDto"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Item",
											ReferenceLocation: ast_domain.Location{
												Line:   28,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   61,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("domain/modal.go"),
									},
								},
								Location: ast_domain.Location{
									Line:   28,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   28,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("dto.ItemDto"),
										PackageAlias:         "dto",
										CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Item",
										ReferenceLocation: ast_domain.Location{
											Line:   28,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   61,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("domain/modal.go"),
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "m",
								RawExpression: "state.Root.Item.Ranges[0].Charge.MustNumber",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
										Base: &ast_domain.IndexExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
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
																	CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 17,
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
															Name: "Root",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("domain.RootModal"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Root",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 17,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 23,
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
																TypeExpression:       typeExprFromString("domain.RootModal"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Root",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.ItemDto"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Item",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 17,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/modal.go"),
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
															TypeExpression:       typeExprFromString("dto.ItemDto"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Item",
															ReferenceLocation: ast_domain.Location{
																Line:   33,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   61,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/modal.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Ranges",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 17,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]dto.RangeItem"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Ranges",
															ReferenceLocation: ast_domain.Location{
																Line:   33,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/item.go"),
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
														TypeExpression:       typeExprFromString("[]dto.RangeItem"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Ranges",
														ReferenceLocation: ast_domain.Location{
															Line:   33,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dto/item.go"),
												},
											},
											Index: &ast_domain.IntegerLiteral{
												Value: 0,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 24,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("main.pk"),
													Stringability:      1,
												},
											},
											Optional: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.RangeItem"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												BaseCodeGenVarName: new("pageData"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Charge",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 27,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("maths.Money"),
													PackageAlias:         "maths",
													CanonicalPackagePath: "piko.sh/piko/wdk/maths",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Charge",
													ReferenceLocation: ast_domain.Location{
														Line:   33,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   64,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/item.go"),
												Stringability:       4,
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
												TypeExpression:       typeExprFromString("maths.Money"),
												PackageAlias:         "maths",
												CanonicalPackagePath: "piko.sh/piko/wdk/maths",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Charge",
												ReferenceLocation: ast_domain.Location{
													Line:   33,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   64,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/item.go"),
											Stringability:       4,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "MustNumber",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "maths",
												CanonicalPackagePath: "piko.sh/piko/wdk/maths",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MustNumber",
												ReferenceLocation: ast_domain.Location{
													Line:   33,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   217,
													Column: 1,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
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
											PackageAlias:         "maths",
											CanonicalPackagePath: "piko.sh/piko/wdk/maths",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "MustNumber",
											ReferenceLocation: ast_domain.Location{
												Line:   33,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   217,
												Column: 1,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
									},
								},
								Location: ast_domain.Location{
									Line:   33,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   33,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "maths",
										CanonicalPackagePath: "piko.sh/piko/wdk/maths",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "MustNumber",
										ReferenceLocation: ast_domain.Location{
											Line:   33,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   217,
											Column: 1,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("../../../../../../../wdk/maths/money_convert.go"),
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "r",
								RawExpression: "state.Root",
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
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 17,
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
										Name: "Root",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("domain.RootModal"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Root",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 23,
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
											TypeExpression:       typeExprFromString("domain.RootModal"),
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Root",
											ReferenceLocation: ast_domain.Location{
												Line:   27,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   46,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								Location: ast_domain.Location{
									Line:   27,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   27,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("domain.RootModal"),
										PackageAlias:         "domain",
										CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Root",
										ReferenceLocation: ast_domain.Location{
											Line:   27,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   46,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "s",
								RawExpression: "state.Root.Item.Ranges",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 17,
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
												Name: "Root",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("domain.RootModal"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Root",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 23,
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
													TypeExpression:       typeExprFromString("domain.RootModal"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Root",
													ReferenceLocation: ast_domain.Location{
														Line:   29,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Item",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 12,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.ItemDto"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Item",
													ReferenceLocation: ast_domain.Location{
														Line:   29,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   61,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/modal.go"),
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
												TypeExpression:       typeExprFromString("dto.ItemDto"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Item",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   61,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/modal.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Ranges",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 17,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.RangeItem"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Ranges",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 17,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/item.go"),
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
											TypeExpression:       typeExprFromString("[]dto.RangeItem"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Ranges",
											ReferenceLocation: ast_domain.Location{
												Line:   29,
												Column: 17,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   71,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dto/item.go"),
									},
								},
								Location: ast_domain.Location{
									Line:   29,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   29,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]dto.RangeItem"),
										PackageAlias:         "dto",
										CanonicalPackagePath: "testcase_43_deep_field_resolution_on_slice_element_bug/dto",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Ranges",
										ReferenceLocation: ast_domain.Location{
											Line:   29,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   71,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dto/item.go"),
								},
							},
						},
					},
				},
			},
		},
	}
}()
