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
						Value: "promoted-get",
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
							Column: 29,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   43,
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
									Line:   43,
									Column: 32,
								},
								RawExpression: "props.Data.WrappedMember.Get().FullName()",
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
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   43,
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
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   43,
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
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
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
														Name: "WrappedMember",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "WrappedMember",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 32,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   55,
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
															TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "WrappedMember",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 32,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   55,
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
														Column: 26,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 32,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
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
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Get",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
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
												Column: 32,
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
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
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
													Line:   43,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
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
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
					RawExpression: "props.Data.WrappedMember.HasItem()",
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
									Name: "WrappedMember",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 12,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "WrappedMember",
											ReferenceLocation: ast_domain.Location{
												Line:   45,
												Column: 39,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   55,
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
										TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "WrappedMember",
										ReferenceLocation: ast_domain.Location{
											Line:   45,
											Column: 39,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   55,
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
									Column: 26,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "HasItem",
										ReferenceLocation: ast_domain.Location{
											Line:   45,
											Column: 39,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   60,
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
									CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "HasItem",
									ReferenceLocation: ast_domain.Location{
										Line:   45,
										Column: 39,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   60,
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
								CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
							},
							BaseCodeGenVarName: new("props"),
							Stringability:      1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("bool"),
							PackageAlias:         "fields",
							CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
						Value: "promoted-hasitem",
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
							Column: 75,
						},
						TextContent: "Has item",
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
								Column: 75,
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
						Value: "promoted-getid",
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
							Column: 31,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.2:0",
							RelativeLocation: ast_domain.Location{
								Line:   47,
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
									Line:   47,
									Column: 34,
								},
								RawExpression: "props.Data.WrappedMember.GetID()",
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
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
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
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
												Name: "WrappedMember",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "WrappedMember",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 34,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   55,
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
													TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "WrappedMember",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 34,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   55,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/dto.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "GetID",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 26,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "GetID",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 34,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   64,
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "GetID",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 34,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   64,
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
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
						Value: "own-method",
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
							Column: 27,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.3:0",
							RelativeLocation: ast_domain.Location{
								Line:   49,
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
									Line:   49,
									Column: 30,
								},
								RawExpression: "props.Data.WrappedMember.GetLabel()",
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
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
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
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
												Name: "WrappedMember",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "WrappedMember",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   55,
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
													TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "WrappedMember",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   55,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/dto.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "GetLabel",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 26,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "GetLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 30,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   73,
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "GetLabel",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 30,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   73,
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
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
						Value: "promoted-field",
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
							Column: 31,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.4:0",
							RelativeLocation: ast_domain.Location{
								Line:   51,
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
									Line:   51,
									Column: 34,
								},
								RawExpression: "props.Data.WrappedMember.Get().FirstName.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
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
																		CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
															Name: "WrappedMember",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 12,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "WrappedMember",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   55,
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
																TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "WrappedMember",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   55,
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
															Column: 26,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   52,
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 34,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
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
													Column: 32,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FirstName",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 34,
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
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FirstName",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 34,
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
												Column: 42,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 34,
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 34,
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
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
						Value: "explicit-embedded",
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
							Column: 34,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.5:0",
							RelativeLocation: ast_domain.Location{
								Line:   53,
								Column: 34,
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
									Column: 37,
								},
								RawExpression: "props.Data.WrappedMember.Ref.Get().LastName.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.CallExpression{
												Callee: &ast_domain.MemberExpression{
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
																			CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   53,
																				Column: 37,
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
																			CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   53,
																				Column: 37,
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
																		CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Data",
																		ReferenceLocation: ast_domain.Location{
																			Line:   53,
																			Column: 37,
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
																Name: "WrappedMember",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "WrappedMember",
																		ReferenceLocation: ast_domain.Location{
																			Line:   53,
																			Column: 37,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   55,
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
																	TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "WrappedMember",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   55,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dto/dto.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Ref",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 26,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Ref",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   69,
																		Column: 2,
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
																TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Ref",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   69,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("fields/fields.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Get",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 30,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   52,
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
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
												Name: "LastName",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 36,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "LastName",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   47,
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
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "LastName",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 37,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   47,
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
												Column: 45,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 37,
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   53,
													Column: 37,
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
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
						Value: "named-field",
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
							Column: 28,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.6:0",
							RelativeLocation: ast_domain.Location{
								Line:   55,
								Column: 28,
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
									Line:   55,
									Column: 31,
								},
								RawExpression: "props.Data.NamedMember.TheRef.Get().FullName()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   55,
																			Column: 31,
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
																		CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Data",
																		ReferenceLocation: ast_domain.Location{
																			Line:   55,
																			Column: 31,
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
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   55,
																		Column: 31,
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
															Name: "NamedMember",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 12,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("fields.NamedRefHolder[dto.TeamMember]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "NamedMember",
																	ReferenceLocation: ast_domain.Location{
																		Line:   55,
																		Column: 31,
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
																TypeExpression:       typeExprFromString("fields.NamedRefHolder[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "NamedMember",
																ReferenceLocation: ast_domain.Location{
																	Line:   55,
																	Column: 31,
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
														Name: "TheRef",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 24,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TheRef",
																ReferenceLocation: ast_domain.Location{
																	Line:   55,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   78,
																	Column: 2,
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
															TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TheRef",
															ReferenceLocation: ast_domain.Location{
																Line:   55,
																Column: 31,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   78,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("fields/fields.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Get",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 31,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   55,
																Column: 31,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
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
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Get",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
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
												Column: 37,
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
														Line:   55,
														Column: 31,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
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
													Line:   55,
													Column: 31,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
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
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
					Line:   57,
					Column: 5,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
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
						Value: "combined",
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
							Column: 25,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.7:0",
							RelativeLocation: ast_domain.Location{
								Line:   57,
								Column: 25,
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
									Column: 28,
								},
								RawExpression: "props.Data.WrappedMember.GetLabel()",
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 28,
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
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 28,
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
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 28,
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
												Name: "WrappedMember",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "WrappedMember",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   55,
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
													TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "WrappedMember",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 28,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   55,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dto/dto.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "GetLabel",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 26,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "GetLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 28,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   73,
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
												CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "GetLabel",
												ReferenceLocation: ast_domain.Location{
													Line:   57,
													Column: 28,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   73,
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
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
									},
									BaseCodeGenVarName: new("props"),
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   57,
									Column: 66,
								},
								Literal: ": ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   57,
									Column: 71,
								},
								RawExpression: "props.Data.WrappedMember.Get().FullName()",
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
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   57,
																		Column: 71,
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
																	CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   57,
																		Column: 71,
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
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 71,
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
														Name: "WrappedMember",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "WrappedMember",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 71,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   55,
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
															TypeExpression:       typeExprFromString("fields.RefWrapper[dto.TeamMember]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "WrappedMember",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   55,
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
														Column: 26,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
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
														CanonicalPackagePath: "testcase_78_embedded_generic_promotion/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Get",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
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
												Column: 32,
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
														Line:   57,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
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
													Line:   57,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
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
											CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
										CanonicalPackagePath: "testcase_78_embedded_generic_promotion/dto",
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
	}
}()
