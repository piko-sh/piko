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
				DirIf: &ast_domain.Directive{
					Type: ast_domain.DirectiveIf,
					Location: ast_domain.Location{
						Line:   43,
						Column: 29,
					},
					NameLocation: ast_domain.Location{
						Line:   43,
						Column: 23,
					},
					RawExpression: "props.Data.TripleNested.Get().Data.HasItem()",
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 29,
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 29,
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
														CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 29,
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
												Name: "TripleNested",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TripleNested",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 29,
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
													TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "TripleNested",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 29,
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
												Column: 25,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Get",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 29,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   51,
														Column: 21,
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
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Get",
												ReferenceLocation: ast_domain.Location{
													Line:   43,
													Column: 29,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   51,
													Column: 21,
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
											TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
										},
										BaseCodeGenVarName: new("props"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "Data",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 31,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
											PackageAlias:         "fields",
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Data",
											ReferenceLocation: ast_domain.Location{
												Line:   43,
												Column: 29,
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
										TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Data",
										ReferenceLocation: ast_domain.Location{
											Line:   43,
											Column: 29,
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
								Name: "HasItem",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 36,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "HasItem",
										ReferenceLocation: ast_domain.Location{
											Line:   43,
											Column: 29,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   76,
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
									CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "HasItem",
									ReferenceLocation: ast_domain.Location{
										Line:   43,
										Column: 29,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   76,
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
								CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
							},
							BaseCodeGenVarName: new("props"),
							Stringability:      1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("bool"),
							PackageAlias:         "fields",
							CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
						},
						BaseCodeGenVarName: new("props"),
						Stringability:      1,
					},
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
						Value: "level1",
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
							Column: 75,
						},
						TextContent: "Level 1 OK",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   43,
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
						Column: 29,
					},
					NameLocation: ast_domain.Location{
						Line:   45,
						Column: 23,
					},
					RawExpression: "props.Data.TripleNested.Get().Get().HasItem()",
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
																CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   45,
																	Column: 29,
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
																CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   45,
																	Column: 29,
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   45,
																Column: 29,
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
													Name: "TripleNested",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TripleNested",
															ReferenceLocation: ast_domain.Location{
																Line:   45,
																Column: 29,
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
														TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TripleNested",
														ReferenceLocation: ast_domain.Location{
															Line:   45,
															Column: 29,
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
													Column: 25,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Get",
														ReferenceLocation: ast_domain.Location{
															Line:   45,
															Column: 29,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   51,
															Column: 21,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Get",
													ReferenceLocation: ast_domain.Location{
														Line:   45,
														Column: 29,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   51,
														Column: 21,
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
												TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
												PackageAlias:         "fields",
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											BaseCodeGenVarName: new("props"),
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
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Get",
												ReferenceLocation: ast_domain.Location{
													Line:   45,
													Column: 29,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   59,
													Column: 23,
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
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Get",
											ReferenceLocation: ast_domain.Location{
												Line:   45,
												Column: 29,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   59,
												Column: 23,
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
										TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
									},
									BaseCodeGenVarName: new("props"),
								},
							},
							Property: &ast_domain.Identifier{
								Name: "HasItem",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 37,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("function"),
										PackageAlias:         "fields",
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "HasItem",
										ReferenceLocation: ast_domain.Location{
											Line:   45,
											Column: 29,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   76,
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
									CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "HasItem",
									ReferenceLocation: ast_domain.Location{
										Line:   45,
										Column: 29,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   76,
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
								CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
							},
							BaseCodeGenVarName: new("props"),
							Stringability:      1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("bool"),
							PackageAlias:         "fields",
							CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
						Value: "level2",
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
							Column: 76,
						},
						TextContent: "Level 2 OK",
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
								Column: 76,
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
						Value: "level3",
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
							Column: 23,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.2:0",
							RelativeLocation: ast_domain.Location{
								Line:   47,
								Column: 23,
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
									Column: 26,
								},
								RawExpression: "props.Data.TripleNested.Get().Get().Get().FullName()",
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
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   47,
																						Column: 26,
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
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   47,
																						Column: 26,
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
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Data",
																				ReferenceLocation: ast_domain.Location{
																					Line:   47,
																					Column: 26,
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
																		Name: "TripleNested",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 12,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TripleNested",
																				ReferenceLocation: ast_domain.Location{
																					Line:   47,
																					Column: 26,
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
																			TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TripleNested",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 26,
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
																		Column: 25,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("function"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 26,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   51,
																				Column: 21,
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
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Get",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   51,
																			Column: 21,
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
																	TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																BaseCodeGenVarName: new("props"),
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
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Get",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 26,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   59,
																		Column: 23,
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
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   59,
																	Column: 23,
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
															TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														BaseCodeGenVarName: new("props"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Get",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 37,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "fields",
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 26,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   68,
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
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Get",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 26,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   68,
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
												Column: 43,
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
														Column: 26,
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
													Line:   47,
													Column: 26,
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
											CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
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
										CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
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
						Value: "field-access",
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
							Column: 29,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.3:0",
							RelativeLocation: ast_domain.Location{
								Line:   49,
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
									Line:   49,
									Column: 32,
								},
								RawExpression: "props.Data.TripleNested.Get().Get().Get().FirstName.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
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
																						CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "props",
																						ReferenceLocation: ast_domain.Location{
																							Line:   49,
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
																						CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Data",
																						ReferenceLocation: ast_domain.Location{
																							Line:   49,
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
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   49,
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
																			Name: "TripleNested",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																					PackageAlias:         "fields",
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "TripleNested",
																					ReferenceLocation: ast_domain.Location{
																						Line:   49,
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
																				TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TripleNested",
																				ReferenceLocation: ast_domain.Location{
																					Line:   49,
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
																			Column: 25,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("function"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Get",
																				ReferenceLocation: ast_domain.Location{
																					Line:   49,
																					Column: 32,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   51,
																					Column: 21,
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
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   49,
																				Column: 32,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   51,
																				Column: 21,
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
																		TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	BaseCodeGenVarName: new("props"),
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
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Get",
																		ReferenceLocation: ast_domain.Location{
																			Line:   49,
																			Column: 32,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   59,
																			Column: 23,
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
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Get",
																	ReferenceLocation: ast_domain.Location{
																		Line:   49,
																		Column: 32,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   59,
																		Column: 23,
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
																TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															BaseCodeGenVarName: new("props"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Get",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 37,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   49,
																	Column: 32,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 32,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   68,
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
													Column: 43,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FirstName",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 32,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FirstName",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 32,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 32,
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
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 32,
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
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
						Value: "mixed-access",
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
							Column: 29,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.4:0",
							RelativeLocation: ast_domain.Location{
								Line:   51,
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
									Line:   51,
									Column: 32,
								},
								RawExpression: "props.Data.TripleNested.Inner.Data.Get().LastName.String()",
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
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "props",
																				ReferenceLocation: ast_domain.Location{
																					Line:   51,
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
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Data",
																				ReferenceLocation: ast_domain.Location{
																					Line:   51,
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
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Data",
																			ReferenceLocation: ast_domain.Location{
																				Line:   51,
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
																	Name: "TripleNested",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																			PackageAlias:         "fields",
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TripleNested",
																			ReferenceLocation: ast_domain.Location{
																				Line:   51,
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
																		TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TripleNested",
																		ReferenceLocation: ast_domain.Location{
																			Line:   51,
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
																Name: "Inner",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 25,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Inner",
																		ReferenceLocation: ast_domain.Location{
																			Line:   51,
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
																	TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Inner",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
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
															Name: "Data",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 31,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																	PackageAlias:         "fields",
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
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
																TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
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
														Name: "Get",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 36,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 32,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 32,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   68,
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
													Column: 42,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "LastName",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 32,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "LastName",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 32,
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
												Column: 51,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 32,
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
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 32,
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
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
						Value: "full-chain",
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
								RawExpression: "props.Data.TripleNested.Get().Get().Get().FirstName.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
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
																						CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
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
																						CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
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
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
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
																			Name: "TripleNested",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																					PackageAlias:         "fields",
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "TripleNested",
																					ReferenceLocation: ast_domain.Location{
																						Line:   53,
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
																				TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TripleNested",
																				ReferenceLocation: ast_domain.Location{
																					Line:   53,
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
																		Name: "Get",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 25,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("function"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Get",
																				ReferenceLocation: ast_domain.Location{
																					Line:   53,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   51,
																					Column: 21,
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
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   53,
																				Column: 30,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   51,
																				Column: 21,
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
																		TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	BaseCodeGenVarName: new("props"),
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
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Get",
																		ReferenceLocation: ast_domain.Location{
																			Line:   53,
																			Column: 30,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   59,
																			Column: 23,
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
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Get",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 30,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   59,
																		Column: 23,
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
																TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															BaseCodeGenVarName: new("props"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Get",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 37,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   68,
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
													Column: 43,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FirstName",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 30,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FirstName",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 30,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 30,
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
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   53,
													Column: 30,
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
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
									},
									BaseCodeGenVarName: new("props"),
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   53,
									Column: 93,
								},
								Literal: " ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   53,
									Column: 97,
								},
								RawExpression: "props.Data.TripleNested.Get().Get().Get().LastName.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
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
																						CanonicalPackagePath: "testcase_77_triple_nested_generics/dist/pages/main_aaf9a2e0",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "props",
																						ReferenceLocation: ast_domain.Location{
																							Line:   53,
																							Column: 97,
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
																						CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Data",
																						ReferenceLocation: ast_domain.Location{
																							Line:   53,
																							Column: 97,
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
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Data",
																					ReferenceLocation: ast_domain.Location{
																						Line:   53,
																						Column: 97,
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
																			Name: "TripleNested",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																					PackageAlias:         "fields",
																					CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "TripleNested",
																					ReferenceLocation: ast_domain.Location{
																						Line:   53,
																						Column: 97,
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
																				TypeExpression:       typeExprFromString("fields.Wrapper[fields.Container[fields.Ref[dto.TeamMember]]]"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TripleNested",
																				ReferenceLocation: ast_domain.Location{
																					Line:   53,
																					Column: 97,
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
																			Column: 25,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("function"),
																				PackageAlias:         "fields",
																				CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Get",
																				ReferenceLocation: ast_domain.Location{
																					Line:   53,
																					Column: 97,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   51,
																					Column: 21,
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
																			CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Get",
																			ReferenceLocation: ast_domain.Location{
																				Line:   53,
																				Column: 97,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   51,
																				Column: 21,
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
																		TypeExpression:       typeExprFromString("fields.Container[fields.Ref[dto.TeamMember]]"),
																		PackageAlias:         "fields",
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	BaseCodeGenVarName: new("props"),
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
																		CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Get",
																		ReferenceLocation: ast_domain.Location{
																			Line:   53,
																			Column: 97,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   59,
																			Column: 23,
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
																	CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Get",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 97,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   59,
																		Column: 23,
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
																TypeExpression:       typeExprFromString("fields.Ref[dto.TeamMember]"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															BaseCodeGenVarName: new("props"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Get",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 37,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "fields",
																CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Get",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 97,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
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
															CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Get",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 97,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   68,
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
													Column: 43,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("fields.Text"),
														PackageAlias:         "fields",
														CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "LastName",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 97,
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
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "LastName",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 97,
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
												Column: 52,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "fields",
													CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 97,
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
												CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   53,
													Column: 97,
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
											CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
										CanonicalPackagePath: "testcase_77_triple_nested_generics/fields",
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
