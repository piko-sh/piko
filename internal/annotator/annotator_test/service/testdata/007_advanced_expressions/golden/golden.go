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
					Line:   56,
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
						Line:   56,
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
						Value: "grade-check",
						Location: ast_domain.Location{
							Line:   56,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   56,
							Column: 10,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   56,
							Column: 27,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   56,
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
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   56,
									Column: 27,
								},
								Literal: "\n        Grade: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   57,
									Column: 19,
								},
								RawExpression: "state.Score > 50 ? (state.Score > 90 ? 'A' : 'B') : 'C'",
								Expression: &ast_domain.TernaryExpression{
									Condition: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 19,
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
												Name: "Score",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Score",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
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
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Score",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       1,
											},
										},
										Operator: ">",
										Right: &ast_domain.IntegerLiteral{
											Value: 50,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 15,
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
									Consequent: &ast_domain.TernaryExpression{
										Condition: &ast_domain.BinaryExpression{
											Left: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 21,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 19,
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
													Name: "Score",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 27,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Score",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
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
													Column: 21,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Score",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
											Operator: ">",
											Right: &ast_domain.IntegerLiteral{
												Value: 90,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 35,
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
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 21,
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
										Consequent: &ast_domain.StringLiteral{
											Value: "A",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 40,
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
										Alternate: &ast_domain.StringLiteral{
											Value: "B",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 46,
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
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 20,
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
									Alternate: &ast_domain.StringLiteral{
										Value: "C",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 53,
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
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
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
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   57,
									Column: 77,
								},
								Literal: "\n    ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("main.pk"),
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
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				DirIf: &ast_domain.Directive{
					Type: ast_domain.DirectiveIf,
					Location: ast_domain.Location{
						Line:   59,
						Column: 33,
					},
					NameLocation: ast_domain.Location{
						Line:   59,
						Column: 27,
					},
					RawExpression: "state.IsAdmin && IsAdmin(state.CurrentUser)",
					Expression: &ast_domain.BinaryExpression{
						Left: &ast_domain.MemberExpression{
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
										CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   59,
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
								Name: "IsAdmin",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsAdmin",
										ReferenceLocation: ast_domain.Location{
											Line:   59,
											Column: 33,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   30,
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
									CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "IsAdmin",
									ReferenceLocation: ast_domain.Location{
										Line:   59,
										Column: 33,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   30,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								Stringability:       1,
							},
						},
						Operator: "&&",
						Right: &ast_domain.CallExpression{
							Callee: &ast_domain.Identifier{
								Name: "IsAdmin",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 18,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:          typeExprFromString("function"),
										PackageAlias:            "main_aaf9a2e0",
										CanonicalPackagePath:    "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
										IsExportedPackageSymbol: true,
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsAdmin",
										ReferenceLocation: ast_domain.Location{
											Line:   59,
											Column: 33,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
									},
									BaseCodeGenVarName: new("IsAdmin"),
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							Args: []ast_domain.Expression{
								&ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 26,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   59,
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
										Name: "CurrentUser",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 32,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*models.User"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_07_advanced_expressions/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CurrentUser",
												ReferenceLocation: ast_domain.Location{
													Line:   59,
													Column: 33,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 2,
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
										Column: 26,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("*models.User"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_07_advanced_expressions/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "CurrentUser",
											ReferenceLocation: ast_domain.Location{
												Line:   59,
												Column: 33,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
							},
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 18,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
								},
								BaseCodeGenVarName: new("IsAdmin"),
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
					Value: "r.1",
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
						Name:  "id",
						Value: "admin-check",
						Location: ast_domain.Location{
							Line:   59,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   59,
							Column: 10,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   59,
							Column: 78,
						},
						TextContent: " Welcome, Admin! ",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.1:0",
							RelativeLocation: ast_domain.Location{
								Line:   59,
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
					Line:   62,
					Column: 5,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.2",
					RelativeLocation: ast_domain.Location{
						Line:   62,
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
						Value: "address-check",
						Location: ast_domain.Location{
							Line:   62,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   62,
							Column: 10,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   62,
							Column: 29,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.2:0",
							RelativeLocation: ast_domain.Location{
								Line:   62,
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
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   62,
									Column: 29,
								},
								Literal: "\n        Street: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("main.pk"),
								},
							},
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   63,
									Column: 20,
								},
								RawExpression: "state.CurrentUser?.Address?.Street",
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
														CanonicalPackagePath: "testcase_07_advanced_expressions/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   63,
															Column: 20,
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
												Name: "CurrentUser",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_07_advanced_expressions/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CurrentUser",
														ReferenceLocation: ast_domain.Location{
															Line:   63,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
															Column: 2,
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
													TypeExpression:       typeExprFromString("*models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_07_advanced_expressions/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CurrentUser",
													ReferenceLocation: ast_domain.Location{
														Line:   63,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Address",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*models.Address"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_07_advanced_expressions/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Address",
													ReferenceLocation: ast_domain.Location{
														Line:   63,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   49,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/models.go"),
											},
										},
										Optional: true,
										Computed: false,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*models.Address"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_07_advanced_expressions/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Address",
												ReferenceLocation: ast_domain.Location{
													Line:   63,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   49,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/models.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Street",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 29,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_07_advanced_expressions/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Street",
												ReferenceLocation: ast_domain.Location{
													Line:   63,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   42,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/models.go"),
											Stringability:       1,
										},
									},
									Optional: true,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_07_advanced_expressions/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Street",
											ReferenceLocation: ast_domain.Location{
												Line:   63,
												Column: 20,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   42,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/models.go"),
										Stringability:       1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_07_advanced_expressions/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Street",
										ReferenceLocation: ast_domain.Location{
											Line:   63,
											Column: 20,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   42,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("models/models.go"),
									Stringability:       1,
								},
							},
							ast_domain.TextPart{
								IsLiteral: true,
								Location: ast_domain.Location{
									Line:   63,
									Column: 57,
								},
								Literal: "\n    ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalSourcePath: new("main.pk"),
								},
							},
						},
					},
				},
			},
		},
	}
}()
