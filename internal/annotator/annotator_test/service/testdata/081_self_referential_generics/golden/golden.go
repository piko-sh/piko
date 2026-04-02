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
					Line:   43,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   43,
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
						Value: "node-value",
						Location: ast_domain.Location{
							Line:   43,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   43,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   43,
							Column: 27,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   43,
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
									Line:   43,
									Column: 30,
								},
								RawExpression: "props.Data.CategoryTree.GetValue().FullName()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.CallExpression{
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
																	CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   43,
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
															Name: "Data",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.PageData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   43,
																		Column: 30,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   29,
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
																TypeExpression:       typeExprFromString("dto.PageData"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   29,
																	Column: 20,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "CategoryTree",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CategoryTree",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   64,
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
															TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CategoryTree",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   64,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/dto.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "GetValue",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 25,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 18,
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "GetValue",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 18,
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
													TypeExpression:       typeExprFromString("dto.Category"),
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
												Column: 36,
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
														Line:   43,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
														Column: 19,
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
													Line:   43,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
													Column: 19,
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
					Line:   45,
					Column: 5,
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
						Line:   45,
						Column: 39,
					},
					NameLocation: ast_domain.Location{
						Line:   45,
						Column: 33,
					},
					RawExpression: "props.Data.CategoryTree.HasChildren()",
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
												CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   45,
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
										Name: "Data",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("dto.PageData"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   45,
													Column: 39,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   29,
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
											TypeExpression:       typeExprFromString("dto.PageData"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Data",
											ReferenceLocation: ast_domain.Location{
												Line:   45,
												Column: 39,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   29,
												Column: 20,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "CategoryTree",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 12,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "CategoryTree",
											ReferenceLocation: ast_domain.Location{
												Line:   45,
												Column: 39,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   64,
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
										TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "CategoryTree",
										ReferenceLocation: ast_domain.Location{
											Line:   45,
											Column: 39,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   64,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("props"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dto/dto.go"),
								},
							},
							Property: &ast_domain.Identifier{
								Name: "HasChildren",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 25,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "HasChildren",
										ReferenceLocation: ast_domain.Location{
											Line:   45,
											Column: 39,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   57,
											Column: 18,
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
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "HasChildren",
									ReferenceLocation: ast_domain.Location{
										Line:   45,
										Column: 39,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   57,
										Column: 18,
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
								CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
							},
							BaseCodeGenVarName: new("props"),
							Stringability:      1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("bool"),
							PackageAlias:         "fields",
							CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
						},
						BaseCodeGenVarName: new("props"),
						Stringability:      1,
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.1",
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "node-haschildren",
						Location: ast_domain.Location{
							Line:   45,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   45,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   45,
							Column: 78,
						},
						TextContent: "Has children",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.1:0",
							RelativeLocation: ast_domain.Location{
								Line:   45,
								Column: 78,
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
					Line:   47,
					Column: 5,
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
						Column: 38,
					},
					NameLocation: ast_domain.Location{
						Line:   47,
						Column: 32,
					},
					RawExpression: "props.Data.CategoryTree.FirstChild() != nil",
					Expression: &ast_domain.BinaryExpression{
						Left: &ast_domain.CallExpression{
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
													CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
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
											Name: "Data",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.PageData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 38,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
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
												TypeExpression:       typeExprFromString("dto.PageData"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 38,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   29,
													Column: 20,
												},
											},
											BaseCodeGenVarName:  new("props"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "CategoryTree",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CategoryTree",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 38,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   64,
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
											TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "CategoryTree",
											ReferenceLocation: ast_domain.Location{
												Line:   47,
												Column: 38,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   64,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dto/dto.go"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "FirstChild",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 25,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("function"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "FirstChild",
											ReferenceLocation: ast_domain.Location{
												Line:   47,
												Column: 38,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   65,
												Column: 18,
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "FirstChild",
										ReferenceLocation: ast_domain.Location{
											Line:   47,
											Column: 38,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   65,
											Column: 18,
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
									TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
									PackageAlias:         "fields",
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								BaseCodeGenVarName: new("props"),
							},
						},
						Operator: "!=",
						Right: &ast_domain.NilLiteral{
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 41,
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
					Value: "r.2",
					RelativeLocation: ast_domain.Location{
						Line:   47,
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
						Value: "node-firstchild",
						Location: ast_domain.Location{
							Line:   47,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   47,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   47,
							Column: 83,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.2:0",
							RelativeLocation: ast_domain.Location{
								Line:   47,
								Column: 83,
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
									Column: 86,
								},
								RawExpression: "props.Data.CategoryTree.FirstChild().GetValue().FullName()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.CallExpression{
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
																			CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 86,
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
																	Name: "Data",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.PageData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 86,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   29,
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
																		TypeExpression:       typeExprFromString("dto.PageData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Data",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 86,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   29,
																			Column: 20,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "CategoryTree",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "CategoryTree",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 86,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   64,
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
																	TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "CategoryTree",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 86,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   64,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "FirstChild",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 25,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "FirstChild",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 86,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   65,
																		Column: 18,
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
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FirstChild",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 86,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   65,
																	Column: 18,
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
															TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														BaseCodeGenVarName: new("props"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "GetValue",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 38,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 86,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 18,
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "GetValue",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 86,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 18,
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
													TypeExpression:       typeExprFromString("dto.Category"),
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
												Column: 49,
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
														Line:   47,
														Column: 86,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
														Column: 19,
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
													Line:   47,
													Column: 86,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
													Column: 19,
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
					Line:   49,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.3",
					RelativeLocation: ast_domain.Location{
						Line:   49,
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
						Value: "node-children-value",
						Location: ast_domain.Location{
							Line:   49,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   49,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   49,
							Column: 36,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.3:0",
							RelativeLocation: ast_domain.Location{
								Line:   49,
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
									Line:   49,
									Column: 39,
								},
								RawExpression: "props.Data.CategoryTree.Children[0].GetValue().Name.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.CallExpression{
												Callee: &ast_domain.MemberExpression{
													Base: &ast_domain.IndexExpression{
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
																				CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "props",
																				ReferenceLocation: ast_domain.Location{
																					Line:   49,
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
																		Name: "Data",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.PageData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Data",
																				ReferenceLocation: ast_domain.Location{
																					Line:   49,
																					Column: 39,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   29,
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
																			TypeExpression:       typeExprFromString("dto.PageData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   49,
																				Column: 39,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   29,
																				Column: 20,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "CategoryTree",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "CategoryTree",
																			ReferenceLocation: ast_domain.Location{
																				Line:   49,
																				Column: 39,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   64,
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
																		TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "CategoryTree",
																		ReferenceLocation: ast_domain.Location{
																			Line:   49,
																			Column: 39,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   64,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dto/dto.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Children",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 25,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Children",
																		ReferenceLocation: ast_domain.Location{
																			Line:   49,
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
																	TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Children",
																	ReferenceLocation: ast_domain.Location{
																		Line:   49,
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
														Index: &ast_domain.IntegerLiteral{
															Value: 0,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 34,
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
																TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															BaseCodeGenVarName: new("props"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "GetValue",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 37,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetValue",
																ReferenceLocation: ast_domain.Location{
																	Line:   49,
																	Column: 39,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 18,
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
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 39,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 18,
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
														TypeExpression:       typeExprFromString("dto.Category"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "",
													},
													BaseCodeGenVarName: new("props"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 48,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
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
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
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
												Column: 53,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 39,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
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
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 39,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
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
					Column: 5,
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
						Line:   51,
						Column: 34,
					},
					NameLocation: ast_domain.Location{
						Line:   51,
						Column: 28,
					},
					RawExpression: "props.Data.CategoryTree.Children[0].HasParent()",
					Expression: &ast_domain.CallExpression{
						Callee: &ast_domain.MemberExpression{
							Base: &ast_domain.IndexExpression{
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
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
												Name: "Data",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.PageData"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 34,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
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
													TypeExpression:       typeExprFromString("dto.PageData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 34,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
														Column: 20,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "CategoryTree",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 12,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CategoryTree",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 34,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   64,
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
												TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CategoryTree",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 34,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   64,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/dto.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Children",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 25,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Children",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 34,
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
											TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Children",
											ReferenceLocation: ast_domain.Location{
												Line:   51,
												Column: 34,
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
								Index: &ast_domain.IntegerLiteral{
									Value: 0,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 34,
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
										TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									BaseCodeGenVarName: new("props"),
								},
							},
							Property: &ast_domain.Identifier{
								Name: "HasParent",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 37,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "HasParent",
										ReferenceLocation: ast_domain.Location{
											Line:   51,
											Column: 34,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   72,
											Column: 18,
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
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "HasParent",
									ReferenceLocation: ast_domain.Location{
										Line:   51,
										Column: 34,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   72,
										Column: 18,
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
								CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
							},
							BaseCodeGenVarName: new("props"),
							Stringability:      1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("bool"),
							PackageAlias:         "fields",
							CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
						},
						BaseCodeGenVarName: new("props"),
						Stringability:      1,
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.4",
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "node-parent",
						Location: ast_domain.Location{
							Line:   51,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   51,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   51,
							Column: 83,
						},
						TextContent: "Parent exists",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.4:0",
							RelativeLocation: ast_domain.Location{
								Line:   51,
								Column: 83,
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
					Line:   53,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.5",
					RelativeLocation: ast_domain.Location{
						Line:   53,
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
						Value: "list-value",
						Location: ast_domain.Location{
							Line:   53,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   53,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   53,
							Column: 27,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.5:0",
							RelativeLocation: ast_domain.Location{
								Line:   53,
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
									Line:   53,
									Column: 30,
								},
								RawExpression: "props.Data.TaskList.GetValue().Display()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.CallExpression{
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
																	CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
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
															Name: "Data",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.PageData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 30,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   29,
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
																TypeExpression:       typeExprFromString("dto.PageData"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   29,
																	Column: 20,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "TaskList",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TaskList",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   65,
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
															TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TaskList",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   65,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/dto.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "GetValue",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 21,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   81,
																Column: 24,
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "GetValue",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   81,
															Column: 24,
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
													TypeExpression:       typeExprFromString("dto.Task"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "",
												},
												BaseCodeGenVarName: new("props"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Display",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 32,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Display",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   59,
														Column: 15,
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
												Name: "Display",
												ReferenceLocation: ast_domain.Location{
													Line:   53,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   59,
													Column: 15,
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
					Line:   55,
					Column: 5,
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
						Line:   55,
						Column: 35,
					},
					NameLocation: ast_domain.Location{
						Line:   55,
						Column: 29,
					},
					RawExpression: "props.Data.TaskList.HasNext()",
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
												CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   55,
													Column: 35,
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
										Name: "Data",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("dto.PageData"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   55,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   29,
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
											TypeExpression:       typeExprFromString("dto.PageData"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Data",
											ReferenceLocation: ast_domain.Location{
												Line:   55,
												Column: 35,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   29,
												Column: 20,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "TaskList",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 12,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "TaskList",
											ReferenceLocation: ast_domain.Location{
												Line:   55,
												Column: 35,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   65,
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
										TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "TaskList",
										ReferenceLocation: ast_domain.Location{
											Line:   55,
											Column: 35,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   65,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("props"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dto/dto.go"),
								},
							},
							Property: &ast_domain.Identifier{
								Name: "HasNext",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 21,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "HasNext",
										ReferenceLocation: ast_domain.Location{
											Line:   55,
											Column: 35,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   85,
											Column: 24,
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
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "HasNext",
									ReferenceLocation: ast_domain.Location{
										Line:   55,
										Column: 35,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   85,
										Column: 24,
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
								CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
							},
							BaseCodeGenVarName: new("props"),
							Stringability:      1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("bool"),
							PackageAlias:         "fields",
							CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
						},
						BaseCodeGenVarName: new("props"),
						Stringability:      1,
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.6",
					RelativeLocation: ast_domain.Location{
						Line:   55,
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
						Value: "list-hasnext",
						Location: ast_domain.Location{
							Line:   55,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   55,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   55,
							Column: 66,
						},
						TextContent: "Has next",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.6:0",
							RelativeLocation: ast_domain.Location{
								Line:   55,
								Column: 66,
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
					Line:   57,
					Column: 5,
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
						Line:   57,
						Column: 32,
					},
					NameLocation: ast_domain.Location{
						Line:   57,
						Column: 26,
					},
					RawExpression: "props.Data.TaskList.GetNext() != nil",
					Expression: &ast_domain.BinaryExpression{
						Left: &ast_domain.CallExpression{
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
													CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
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
											Name: "Data",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.PageData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
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
												TypeExpression:       typeExprFromString("dto.PageData"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   57,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   29,
													Column: 20,
												},
											},
											BaseCodeGenVarName:  new("props"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "TaskList",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "TaskList",
												ReferenceLocation: ast_domain.Location{
													Line:   57,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   65,
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
											TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "TaskList",
											ReferenceLocation: ast_domain.Location{
												Line:   57,
												Column: 32,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   65,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dto/dto.go"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "GetNext",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 21,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("function"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "GetNext",
											ReferenceLocation: ast_domain.Location{
												Line:   57,
												Column: 32,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   89,
												Column: 24,
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "GetNext",
										ReferenceLocation: ast_domain.Location{
											Line:   57,
											Column: 32,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   89,
											Column: 24,
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
									TypeExpression:       typeExprFromString("*fields.LinkedList[dto.Task]"),
									PackageAlias:         "fields",
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								BaseCodeGenVarName: new("props"),
							},
						},
						Operator: "!=",
						Right: &ast_domain.NilLiteral{
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 34,
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
					Value: "r.7",
					RelativeLocation: ast_domain.Location{
						Line:   57,
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
						Value: "list-next",
						Location: ast_domain.Location{
							Line:   57,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   57,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   57,
							Column: 70,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.7:0",
							RelativeLocation: ast_domain.Location{
								Line:   57,
								Column: 70,
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
									Line:   57,
									Column: 73,
								},
								RawExpression: "props.Data.TaskList.GetNext().GetValue().Title.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.CallExpression{
												Callee: &ast_domain.MemberExpression{
													Base: &ast_domain.CallExpression{
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
																				CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "props",
																				ReferenceLocation: ast_domain.Location{
																					Line:   57,
																					Column: 73,
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
																		Name: "Data",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.PageData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Data",
																				ReferenceLocation: ast_domain.Location{
																					Line:   57,
																					Column: 73,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   29,
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
																			TypeExpression:       typeExprFromString("dto.PageData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   57,
																				Column: 73,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   29,
																				Column: 20,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "TaskList",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TaskList",
																			ReferenceLocation: ast_domain.Location{
																				Line:   57,
																				Column: 73,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   65,
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
																		TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TaskList",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 73,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   65,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dto/dto.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "GetNext",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 21,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("function"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "GetNext",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 73,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   89,
																			Column: 24,
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
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetNext",
																	ReferenceLocation: ast_domain.Location{
																		Line:   57,
																		Column: 73,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   89,
																		Column: 24,
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
																TypeExpression:       typeExprFromString("*fields.LinkedList[dto.Task]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															BaseCodeGenVarName: new("props"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "GetValue",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 31,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetValue",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 73,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   81,
																	Column: 24,
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
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   81,
																Column: 24,
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
														TypeExpression:       typeExprFromString("dto.Task"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "",
													},
													BaseCodeGenVarName: new("props"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Title",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 42,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 73,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   55,
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
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 73,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   55,
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
												Column: 48,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 73,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
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
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   57,
													Column: 73,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
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
					Line:   59,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.8",
					RelativeLocation: ast_domain.Location{
						Line:   59,
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
						Value: "list-chain",
						Location: ast_domain.Location{
							Line:   59,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   59,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   59,
							Column: 27,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.8:0",
							RelativeLocation: ast_domain.Location{
								Line:   59,
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
									Line:   59,
									Column: 30,
								},
								RawExpression: "props.Data.TaskList.GetNext().GetNext().GetValue().Display()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.CallExpression{
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
																					CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   59,
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
																			Name: "Data",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.PageData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   59,
																						Column: 30,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   29,
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
																				TypeExpression:       typeExprFromString("dto.PageData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Data",
																				ReferenceLocation: ast_domain.Location{
																					Line:   59,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   29,
																					Column: 20,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "TaskList",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 12,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TaskList",
																				ReferenceLocation: ast_domain.Location{
																					Line:   59,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   65,
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
																			TypeExpression:       typeExprFromString("fields.LinkedList[dto.Task]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TaskList",
																			ReferenceLocation: ast_domain.Location{
																				Line:   59,
																				Column: 30,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   65,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dto/dto.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "GetNext",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 21,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("function"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "GetNext",
																			ReferenceLocation: ast_domain.Location{
																				Line:   59,
																				Column: 30,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   89,
																				Column: 24,
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
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "GetNext",
																		ReferenceLocation: ast_domain.Location{
																			Line:   59,
																			Column: 30,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   89,
																			Column: 24,
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
																	TypeExpression:       typeExprFromString("*fields.LinkedList[dto.Task]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																BaseCodeGenVarName: new("props"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "GetNext",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 31,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetNext",
																	ReferenceLocation: ast_domain.Location{
																		Line:   59,
																		Column: 30,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   89,
																		Column: 24,
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
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetNext",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   89,
																	Column: 24,
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
															TypeExpression:       typeExprFromString("*fields.LinkedList[dto.Task]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														BaseCodeGenVarName: new("props"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "GetValue",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 41,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   81,
																Column: 24,
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "GetValue",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   81,
															Column: 24,
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
													TypeExpression:       typeExprFromString("dto.Task"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "",
												},
												BaseCodeGenVarName: new("props"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Display",
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
													Name: "Display",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   59,
														Column: 15,
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
												Name: "Display",
												ReferenceLocation: ast_domain.Location{
													Line:   59,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   59,
													Column: 15,
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
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
					Line:   61,
					Column: 5,
				},
				TagName: "ul",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.9",
					RelativeLocation: ast_domain.Location{
						Line:   61,
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
						Value: "children-loop",
						Location: ast_domain.Location{
							Line:   61,
							Column: 16,
						},
						NameLocation: ast_domain.Location{
							Line:   61,
							Column: 9,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   62,
							Column: 9,
						},
						TagName: "li",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   62,
								Column: 20,
							},
							NameLocation: ast_domain.Location{
								Line:   62,
								Column: 13,
							},
							RawExpression: "child in props.Data.CategoryTree.Children",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: nil,
								ItemVariable: &ast_domain.Identifier{
									Name: "child",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "child",
											ReferenceLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("child"),
										OriginalSourcePath: new("main.pk"),
									},
								},
								Collection: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 10,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_81_self_referential_generics/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   62,
															Column: 20,
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
												Name: "Data",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 16,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.PageData"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   62,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
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
												Column: 10,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.PageData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_81_self_referential_generics/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   62,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
														Column: 20,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "CategoryTree",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 21,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CategoryTree",
													ReferenceLocation: ast_domain.Location{
														Line:   62,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   64,
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
											Column: 10,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("fields.Node[dto.Category]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CategoryTree",
												ReferenceLocation: ast_domain.Location{
													Line:   62,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   64,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/dto.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Children",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 34,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Children",
												ReferenceLocation: ast_domain.Location{
													Line:   62,
													Column: 20,
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
										Column: 10,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Children",
											ReferenceLocation: ast_domain.Location{
												Line:   62,
												Column: 20,
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
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Children",
										ReferenceLocation: ast_domain.Location{
											Line:   62,
											Column: 20,
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
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Children",
										ReferenceLocation: ast_domain.Location{
											Line:   62,
											Column: 20,
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
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]*fields.Node[dto.Category]"),
									PackageAlias:         "fields",
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Children",
									ReferenceLocation: ast_domain.Location{
										Line:   62,
										Column: 20,
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
						DirKey: &ast_domain.Directive{
							Type: ast_domain.DirectiveKey,
							Location: ast_domain.Location{
								Line:   62,
								Column: 70,
							},
							NameLocation: ast_domain.Location{
								Line:   62,
								Column: 63,
							},
							RawExpression: "child.GetValue().Name.String()",
							Expression: &ast_domain.CallExpression{
								Callee: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
										Base: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "child",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "child",
															ReferenceLocation: ast_domain.Location{
																Line:   62,
																Column: 102,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("child"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "GetValue",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetValue",
															ReferenceLocation: ast_domain.Location{
																Line:   62,
																Column: 70,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 18,
															},
														},
														BaseCodeGenVarName:  new("child"),
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "GetValue",
														ReferenceLocation: ast_domain.Location{
															Line:   62,
															Column: 102,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 18,
														},
													},
													BaseCodeGenVarName:  new("child"),
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
													TypeExpression:       typeExprFromString("dto.Category"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "",
												},
												BaseCodeGenVarName: new("child"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Name",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 18,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("fields.Text"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   62,
														Column: 70,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("child"),
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
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   62,
													Column: 102,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("child"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dto/dto.go"),
											Stringability:       2,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "String",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   62,
													Column: 70,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 15,
												},
											},
											BaseCodeGenVarName:  new("child"),
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
											CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "String",
											ReferenceLocation: ast_domain.Location{
												Line:   62,
												Column: 102,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   43,
												Column: 15,
											},
										},
										BaseCodeGenVarName:  new("child"),
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
										CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
									},
									BaseCodeGenVarName: new("child"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "fields",
									CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
								},
								BaseCodeGenVarName: new("child"),
								Stringability:      1,
							},
						},
						Key: &ast_domain.TemplateLiteral{
							Parts: []ast_domain.TemplateLiteralPart{
								ast_domain.TemplateLiteralPart{
									IsLiteral: true,
									RelativeLocation: ast_domain.Location{
										Line:   62,
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
									Literal: "r.9:0.",
								},
								ast_domain.TemplateLiteralPart{
									IsLiteral: false,
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
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
									Expression: &ast_domain.CallExpression{
										Callee: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "child",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "child",
																	ReferenceLocation: ast_domain.Location{
																		Line:   62,
																		Column: 102,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("child"),
																OriginalSourcePath: new("main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "GetValue",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetValue",
																	ReferenceLocation: ast_domain.Location{
																		Line:   62,
																		Column: 70,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   53,
																		Column: 18,
																	},
																},
																BaseCodeGenVarName:  new("child"),
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
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetValue",
																ReferenceLocation: ast_domain.Location{
																	Line:   62,
																	Column: 102,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 18,
																},
															},
															BaseCodeGenVarName:  new("child"),
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
															TypeExpression:       typeExprFromString("dto.Category"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "",
														},
														BaseCodeGenVarName: new("child"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 18,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("fields.Text"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   62,
																Column: 70,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("child"),
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   62,
															Column: 102,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("child"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dto/dto.go"),
													Stringability:       2,
												},
											},
											Property: &ast_domain.Identifier{
												Name: "String",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   62,
															Column: 70,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 15,
														},
													},
													BaseCodeGenVarName:  new("child"),
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
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   62,
														Column: 102,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
														Column: 15,
													},
												},
												BaseCodeGenVarName:  new("child"),
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
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											BaseCodeGenVarName: new("child"),
											Stringability:      1,
										},
									},
								},
							},
							RelativeLocation: ast_domain.Location{
								Line:   62,
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   62,
									Column: 102,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   62,
												Column: 102,
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
											Literal: "r.9:0.",
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: false,
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
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.CallExpression{
												Callee: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.CallExpression{
															Callee: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "child",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "child",
																			ReferenceLocation: ast_domain.Location{
																				Line:   62,
																				Column: 102,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("child"),
																		OriginalSourcePath: new("main.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "GetValue",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("function"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "GetValue",
																			ReferenceLocation: ast_domain.Location{
																				Line:   62,
																				Column: 70,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   53,
																				Column: 18,
																			},
																		},
																		BaseCodeGenVarName:  new("child"),
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
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "GetValue",
																		ReferenceLocation: ast_domain.Location{
																			Line:   62,
																			Column: 102,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   53,
																			Column: 18,
																		},
																	},
																	BaseCodeGenVarName:  new("child"),
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
																	TypeExpression:       typeExprFromString("dto.Category"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "",
																},
																BaseCodeGenVarName: new("child"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Name",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 18,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("fields.Text"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   62,
																		Column: 70,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("child"),
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
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   62,
																	Column: 102,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("child"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dto/dto.go"),
															Stringability:       2,
														},
													},
													Property: &ast_domain.Identifier{
														Name: "String",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 23,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "String",
																ReferenceLocation: ast_domain.Location{
																	Line:   62,
																	Column: 70,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 15,
																},
															},
															BaseCodeGenVarName:  new("child"),
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
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   62,
																Column: 102,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 15,
															},
														},
														BaseCodeGenVarName:  new("child"),
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													BaseCodeGenVarName: new("child"),
													Stringability:      1,
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   62,
												Column: 102,
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
											Literal: ":0",
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   62,
										Column: 102,
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
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   62,
											Column: 102,
										},
										Literal: "\n            ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   63,
											Column: 16,
										},
										RawExpression: "child.GetValue().Name.String()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.CallExpression{
														Callee: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "child",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("*fields.Node[dto.Category]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "child",
																		ReferenceLocation: ast_domain.Location{
																			Line:   63,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("child"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "GetValue",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("function"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "GetValue",
																		ReferenceLocation: ast_domain.Location{
																			Line:   63,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   53,
																			Column: 18,
																		},
																	},
																	BaseCodeGenVarName:  new("child"),
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
																	CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetValue",
																	ReferenceLocation: ast_domain.Location{
																		Line:   63,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   53,
																		Column: 18,
																	},
																},
																BaseCodeGenVarName:  new("child"),
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
																TypeExpression:       typeExprFromString("dto.Category"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "",
															},
															BaseCodeGenVarName: new("child"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Name",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 18,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.Text"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   63,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("child"),
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
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   63,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("child"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dto/dto.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 23,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   63,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 15,
															},
														},
														BaseCodeGenVarName:  new("child"),
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
														CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   63,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 15,
														},
													},
													BaseCodeGenVarName:  new("child"),
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
													CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
												},
												BaseCodeGenVarName: new("child"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_81_self_referential_generics/fields",
											},
											BaseCodeGenVarName: new("child"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   63,
											Column: 49,
										},
										Literal: "\n        ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("main.pk"),
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
