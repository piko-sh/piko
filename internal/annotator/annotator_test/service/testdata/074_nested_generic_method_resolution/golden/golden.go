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
				TagName: "div",
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "property-details",
						Location: ast_domain.Location{
							Line:   41,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   41,
							Column: 10,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   42,
							Column: 9,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   42,
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
								Name:  "class",
								Value: "title",
								Location: ast_domain.Location{
									Line:   42,
									Column: 20,
								},
								NameLocation: ast_domain.Location{
									Line:   42,
									Column: 13,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   42,
									Column: 27,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   42,
										Column: 27,
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
											Line:   42,
											Column: 30,
										},
										RawExpression: "props.Property.Title.String()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
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
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   42,
																		Column: 30,
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
															Name: "Property",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.PropertyData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Property",
																	ReferenceLocation: ast_domain.Location{
																		Line:   42,
																		Column: 30,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   28,
																		Column: 20,
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
																TypeExpression:       typeExprFromString("dto.PropertyData"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Property",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   28,
																	Column: 20,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Title",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 16,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.Text"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   60,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dto/dto.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("fields.Text"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   60,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/dto.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 22,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   70,
																Column: 15,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("fields/fields.go"),
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
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   70,
															Column: 15,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("fields/fields.go"),
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
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
												},
												BaseCodeGenVarName: new("props"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
											},
											BaseCodeGenVarName: new("props"),
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
							Line:   44,
							Column: 9,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   44,
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
								Name:  "class",
								Value: "viewing-info",
								Location: ast_domain.Location{
									Line:   44,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   44,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   45,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   45,
										Column: 13,
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
										Name:  "class",
										Value: "date",
										Location: ast_domain.Location{
											Line:   45,
											Column: 23,
										},
										NameLocation: ast_domain.Location{
											Line:   45,
											Column: 16,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   45,
											Column: 29,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   45,
												Column: 29,
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
													Line:   45,
													Column: 32,
												},
												RawExpression: "props.Property.OpenViewing.Data.Date.String()",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
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
																					CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   45,
																						Column: 32,
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
																			Name: "Property",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.PropertyData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Property",
																					ReferenceLocation: ast_domain.Location{
																						Line:   45,
																						Column: 32,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   28,
																						Column: 20,
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
																				TypeExpression:       typeExprFromString("dto.PropertyData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Property",
																				ReferenceLocation: ast_domain.Location{
																					Line:   45,
																					Column: 32,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   28,
																					Column: 20,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "OpenViewing",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 16,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "OpenViewing",
																				ReferenceLocation: ast_domain.Location{
																					Line:   45,
																					Column: 32,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   61,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dto/dto.go"),
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
																			TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "OpenViewing",
																			ReferenceLocation: ast_domain.Location{
																				Line:   45,
																				Column: 32,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   61,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dto/dto.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Data",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 28,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   45,
																				Column: 32,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   20,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("fields/fields.go"),
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
																		TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Data",
																		ReferenceLocation: ast_domain.Location{
																			Line:   45,
																			Column: 32,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   20,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("fields/fields.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Date",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 33,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("fields.Text"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Date",
																		ReferenceLocation: ast_domain.Location{
																			Line:   45,
																			Column: 32,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   54,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dto/dto.go"),
																	Stringability:       2,
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
																	TypeExpression:       typeExprFromString("fields.Text"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Date",
																	ReferenceLocation: ast_domain.Location{
																		Line:   45,
																		Column: 32,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   54,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
																Stringability:       2,
															},
														},
														Property: &ast_domain.Identifier{
															Name: "String",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 38,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "String",
																	ReferenceLocation: ast_domain.Location{
																		Line:   45,
																		Column: 32,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   70,
																		Column: 15,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("fields/fields.go"),
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
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "String",
																ReferenceLocation: ast_domain.Location{
																	Line:   45,
																	Column: 32,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   70,
																	Column: 15,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("fields/fields.go"),
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
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
													},
													BaseCodeGenVarName: new("props"),
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
									Line:   47,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   47,
										Column: 40,
									},
									NameLocation: ast_domain.Location{
										Line:   47,
										Column: 34,
									},
									RawExpression: "props.Property.OpenViewing.Data.ViewingAgent.HasItem()",
									Expression: &ast_domain.CallExpression{
										Callee: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 40,
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
																Name: "Property",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("dto.PropertyData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Property",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 40,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   28,
																			Column: 20,
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
																	TypeExpression:       typeExprFromString("dto.PropertyData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Property",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 40,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   28,
																		Column: 20,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "OpenViewing",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 16,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "OpenViewing",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 40,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
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
																TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "OpenViewing",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 40,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dto/dto.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Data",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 28,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 40,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   20,
																	Column: 0,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("fields/fields.go"),
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
															TypeExpression:       typeExprFromString("dto.OpenViewingData"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 40,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   20,
																Column: 0,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("fields/fields.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "ViewingAgent",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 33,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ViewingAgent",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 40,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   56,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/dto.go"),
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
														TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ViewingAgent",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 40,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   56,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dto/dto.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "HasItem",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 46,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "HasItem",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 40,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   54,
															Column: 17,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("fields/fields.go"),
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
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "HasItem",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 40,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   54,
														Column: 17,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("fields/fields.go"),
											},
										},
										Args: []ast_domain.Expression{},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
											},
											BaseCodeGenVarName: new("props"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
										},
										BaseCodeGenVarName: new("props"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
									RelativeLocation: ast_domain.Location{
										Line:   47,
										Column: 13,
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
										Name:  "class",
										Value: "has-agent",
										Location: ast_domain.Location{
											Line:   47,
											Column: 23,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 16,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   47,
											Column: 96,
										},
										TextContent: "Has Agent",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   47,
												Column: 96,
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
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   49,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:2",
									RelativeLocation: ast_domain.Location{
										Line:   49,
										Column: 13,
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
										Name:  "class",
										Value: "agent-name",
										Location: ast_domain.Location{
											Line:   49,
											Column: 23,
										},
										NameLocation: ast_domain.Location{
											Line:   49,
											Column: 16,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   49,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   49,
												Column: 35,
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
													Line:   49,
													Column: 38,
												},
												RawExpression: "props.Property.OpenViewing.Data.ViewingAgent.Get().FullName()",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.CallExpression{
															Callee: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.MemberExpression{
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
																							CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dist/pages/main_aaf9a2e0",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "props",
																							ReferenceLocation: ast_domain.Location{
																								Line:   49,
																								Column: 38,
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
																					Name: "Property",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 7,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("dto.PropertyData"),
																							PackageAlias:         "dto",
																							CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "Property",
																							ReferenceLocation: ast_domain.Location{
																								Line:   49,
																								Column: 38,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   28,
																								Column: 20,
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
																						TypeExpression:       typeExprFromString("dto.PropertyData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Property",
																						ReferenceLocation: ast_domain.Location{
																							Line:   49,
																							Column: 38,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   28,
																							Column: 20,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "OpenViewing",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 16,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																						PackageAlias:         "fields",
																						CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "OpenViewing",
																						ReferenceLocation: ast_domain.Location{
																							Line:   49,
																							Column: 38,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   61,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("dto/dto.go"),
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
																					TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																					PackageAlias:         "fields",
																					CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "OpenViewing",
																					ReferenceLocation: ast_domain.Location{
																						Line:   49,
																						Column: 38,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   61,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("dto/dto.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "Data",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 28,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   49,
																						Column: 38,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   20,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("fields/fields.go"),
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
																				TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Data",
																				ReferenceLocation: ast_domain.Location{
																					Line:   49,
																					Column: 38,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   20,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("fields/fields.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "ViewingAgent",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 33,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ViewingAgent",
																				ReferenceLocation: ast_domain.Location{
																					Line:   49,
																					Column: 38,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   56,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dto/dto.go"),
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
																			TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ViewingAgent",
																			ReferenceLocation: ast_domain.Location{
																				Line:   49,
																				Column: 38,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   56,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dto/dto.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Get",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 46,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("function"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   49,
																				Column: 38,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   58,
																				Column: 17,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("fields/fields.go"),
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
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Get",
																		ReferenceLocation: ast_domain.Location{
																			Line:   49,
																			Column: 38,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   58,
																			Column: 17,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("fields/fields.go"),
																},
															},
															Args: []ast_domain.Expression{},
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.TeamMember"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "",
																},
																BaseCodeGenVarName: new("props"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "FullName",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 52,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "FullName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   49,
																		Column: 38,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   49,
																		Column: 21,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FullName",
																ReferenceLocation: ast_domain.Location{
																	Line:   49,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   49,
																	Column: 21,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dto/dto.go"),
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
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
													},
													BaseCodeGenVarName: new("props"),
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
									Line:   51,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:3",
									RelativeLocation: ast_domain.Location{
										Line:   51,
										Column: 13,
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
										Name:  "class",
										Value: "agent-first-name",
										Location: ast_domain.Location{
											Line:   51,
											Column: 23,
										},
										NameLocation: ast_domain.Location{
											Line:   51,
											Column: 16,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   51,
											Column: 41,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:3:0",
											RelativeLocation: ast_domain.Location{
												Line:   51,
												Column: 41,
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
													Line:   51,
													Column: 44,
												},
												RawExpression: "props.Property.OpenViewing.Data.ViewingAgent.Get().FirstName.String()",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.CallExpression{
																Callee: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.MemberExpression{
																			Base: &ast_domain.MemberExpression{
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
																								CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dist/pages/main_aaf9a2e0",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "props",
																								ReferenceLocation: ast_domain.Location{
																									Line:   51,
																									Column: 44,
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
																						Name: "Property",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 7,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("dto.PropertyData"),
																								PackageAlias:         "dto",
																								CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "Property",
																								ReferenceLocation: ast_domain.Location{
																									Line:   51,
																									Column: 44,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   28,
																									Column: 20,
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
																							TypeExpression:       typeExprFromString("dto.PropertyData"),
																							PackageAlias:         "dto",
																							CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "Property",
																							ReferenceLocation: ast_domain.Location{
																								Line:   51,
																								Column: 44,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   28,
																								Column: 20,
																							},
																						},
																						BaseCodeGenVarName:  new("props"),
																						OriginalSourcePath:  new("main.pk"),
																						GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																					},
																				},
																				Property: &ast_domain.Identifier{
																					Name: "OpenViewing",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 16,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																							PackageAlias:         "fields",
																							CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "OpenViewing",
																							ReferenceLocation: ast_domain.Location{
																								Line:   51,
																								Column: 44,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   61,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("props"),
																						OriginalSourcePath:  new("main.pk"),
																						GeneratedSourcePath: new("dto/dto.go"),
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
																						TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																						PackageAlias:         "fields",
																						CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "OpenViewing",
																						ReferenceLocation: ast_domain.Location{
																							Line:   51,
																							Column: 44,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   61,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("dto/dto.go"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Data",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 28,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Data",
																						ReferenceLocation: ast_domain.Location{
																							Line:   51,
																							Column: 44,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   20,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("fields/fields.go"),
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
																					TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   51,
																						Column: 44,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   20,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("fields/fields.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ViewingAgent",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 33,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																					PackageAlias:         "fields",
																					CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ViewingAgent",
																					ReferenceLocation: ast_domain.Location{
																						Line:   51,
																						Column: 44,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   56,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("dto/dto.go"),
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
																				TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ViewingAgent",
																				ReferenceLocation: ast_domain.Location{
																					Line:   51,
																					Column: 44,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   56,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dto/dto.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Get",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 46,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("function"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Get",
																				ReferenceLocation: ast_domain.Location{
																					Line:   51,
																					Column: 44,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   58,
																					Column: 17,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("fields/fields.go"),
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
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   51,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   58,
																				Column: 17,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("fields/fields.go"),
																	},
																},
																Args: []ast_domain.Expression{},
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("dto.TeamMember"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "",
																	},
																	BaseCodeGenVarName: new("props"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "FirstName",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 52,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("fields.Text"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "FirstName",
																		ReferenceLocation: ast_domain.Location{
																			Line:   51,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   44,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dto/dto.go"),
																	Stringability:       2,
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
																	TypeExpression:       typeExprFromString("fields.Text"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "FirstName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   44,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
																Stringability:       2,
															},
														},
														Property: &ast_domain.Identifier{
															Name: "String",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 62,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "String",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   70,
																		Column: 15,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("fields/fields.go"),
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
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "String",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   70,
																	Column: 15,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("fields/fields.go"),
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
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
													},
													BaseCodeGenVarName: new("props"),
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
									Line:   53,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:4",
									RelativeLocation: ast_domain.Location{
										Line:   53,
										Column: 13,
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
										Name:  "class",
										Value: "agent-phone",
										Location: ast_domain.Location{
											Line:   53,
											Column: 23,
										},
										NameLocation: ast_domain.Location{
											Line:   53,
											Column: 16,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   53,
											Column: 36,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:4:0",
											RelativeLocation: ast_domain.Location{
												Line:   53,
												Column: 36,
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
													Line:   53,
													Column: 39,
												},
												RawExpression: "props.Property.OpenViewing.Data.ViewingAgent.Get().PhoneNumber.String()",
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.CallExpression{
																Callee: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.MemberExpression{
																			Base: &ast_domain.MemberExpression{
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
																								CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dist/pages/main_aaf9a2e0",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "props",
																								ReferenceLocation: ast_domain.Location{
																									Line:   53,
																									Column: 39,
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
																						Name: "Property",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 7,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("dto.PropertyData"),
																								PackageAlias:         "dto",
																								CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "Property",
																								ReferenceLocation: ast_domain.Location{
																									Line:   53,
																									Column: 39,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   28,
																									Column: 20,
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
																							TypeExpression:       typeExprFromString("dto.PropertyData"),
																							PackageAlias:         "dto",
																							CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/dto",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "Property",
																							ReferenceLocation: ast_domain.Location{
																								Line:   53,
																								Column: 39,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   28,
																								Column: 20,
																							},
																						},
																						BaseCodeGenVarName:  new("props"),
																						OriginalSourcePath:  new("main.pk"),
																						GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																					},
																				},
																				Property: &ast_domain.Identifier{
																					Name: "OpenViewing",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 16,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																							PackageAlias:         "fields",
																							CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "OpenViewing",
																							ReferenceLocation: ast_domain.Location{
																								Line:   53,
																								Column: 39,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   61,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("props"),
																						OriginalSourcePath:  new("main.pk"),
																						GeneratedSourcePath: new("dto/dto.go"),
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
																						TypeExpression:       typeExprFromString("fields.Embedded[dto.OpenViewingData]"),
																						PackageAlias:         "fields",
																						CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "OpenViewing",
																						ReferenceLocation: ast_domain.Location{
																							Line:   53,
																							Column: 39,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   61,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("dto/dto.go"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Data",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 28,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Data",
																						ReferenceLocation: ast_domain.Location{
																							Line:   53,
																							Column: 39,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   20,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("fields/fields.go"),
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
																					TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   53,
																						Column: 39,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   20,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("fields/fields.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ViewingAgent",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 33,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																					PackageAlias:         "fields",
																					CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ViewingAgent",
																					ReferenceLocation: ast_domain.Location{
																						Line:   53,
																						Column: 39,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   56,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("dto/dto.go"),
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
																				TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ViewingAgent",
																				ReferenceLocation: ast_domain.Location{
																					Line:   53,
																					Column: 39,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   56,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dto/dto.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Get",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 46,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("function"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Get",
																				ReferenceLocation: ast_domain.Location{
																					Line:   53,
																					Column: 39,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   58,
																					Column: 17,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("fields/fields.go"),
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
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   53,
																				Column: 39,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   58,
																				Column: 17,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("fields/fields.go"),
																	},
																},
																Args: []ast_domain.Expression{},
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("dto.TeamMember"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "",
																	},
																	BaseCodeGenVarName: new("props"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "PhoneNumber",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 52,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("fields.Text"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "PhoneNumber",
																		ReferenceLocation: ast_domain.Location{
																			Line:   53,
																			Column: 39,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   46,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dto/dto.go"),
																	Stringability:       2,
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
																	TypeExpression:       typeExprFromString("fields.Text"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "PhoneNumber",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 39,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
																Stringability:       2,
															},
														},
														Property: &ast_domain.Identifier{
															Name: "String",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 64,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "String",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 39,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   70,
																		Column: 15,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("fields/fields.go"),
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
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "String",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 39,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   70,
																	Column: 15,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("fields/fields.go"),
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
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_74_nested_generic_method_resolution/fields",
													},
													BaseCodeGenVarName: new("props"),
													OriginalSourcePath: new("main.pk"),
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
