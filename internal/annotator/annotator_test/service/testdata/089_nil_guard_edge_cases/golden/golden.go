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
						Line:   22,
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
							Line:   23,
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
								Line:   23,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   23,
								Column: 10,
							},
							RawExpression: "props.A != nil || props.B != nil",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.BinaryExpression{
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
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
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
											Name: "A",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "A",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   67,
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
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "A",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   67,
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
											Column: 12,
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
								Operator: "||",
								Right: &ast_domain.BinaryExpression{
									Left: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "props",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 19,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
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
											Name: "B",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 25,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "B",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   68,
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
											Column: 19,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "B",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   68,
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
											Column: 30,
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
										Column: 19,
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 7,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   24,
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
										Value: "or-not-guarded-a",
										Location: ast_domain.Location{
											Line:   24,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 13,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
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
													Line:   24,
													Column: 38,
												},
												RawExpression: "props.A.Value",
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
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
															Name: "A",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "A",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 38,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "A",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   67,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   58,
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 38,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   58,
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
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   58,
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
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 7,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   25,
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
										Value: "or-not-guarded-b",
										Location: ast_domain.Location{
											Line:   25,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 13,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   25,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   25,
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
													Line:   25,
													Column: 38,
												},
												RawExpression: "props.B.Value",
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
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
															Name: "B",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "B",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 38,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   68,
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "B",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   58,
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 38,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   58,
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
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   58,
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
							Line:   28,
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
								Line:   28,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   28,
								Column: 10,
							},
							RawExpression: "props.Outer != nil",
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
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   28,
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
										Name: "Outer",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Outer",
												ReferenceLocation: ast_domain.Location{
													Line:   28,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   69,
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
											TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Outer",
											ReferenceLocation: ast_domain.Location{
												Line:   28,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   69,
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
										Column: 16,
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
								Line:   28,
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
									Line:   29,
									Column: 7,
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
										Line:   29,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   29,
										Column: 12,
									},
									RawExpression: "props.Outer.Middle != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 18,
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
													Name: "Outer",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Outer",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 18,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   69,
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
														TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Outer",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   69,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Middle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 13,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Middle",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   64,
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
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Middle",
													ReferenceLocation: ast_domain.Location{
														Line:   29,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   64,
														Column: 20,
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
												Column: 23,
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
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   29,
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   30,
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
												Line:   30,
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
												Value: "triple-nested-partial",
												Location: ast_domain.Location{
													Line:   30,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   30,
													Column: 15,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   30,
													Column: 42,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 42,
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
															Line:   30,
															Column: 45,
														},
														RawExpression: "props.Outer.Middle.Value",
														Expression: &ast_domain.MemberExpression{
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
																				CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "props",
																				ReferenceLocation: ast_domain.Location{
																					Line:   30,
																					Column: 45,
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
																		Name: "Outer",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Outer",
																				ReferenceLocation: ast_domain.Location{
																					Line:   30,
																					Column: 45,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   69,
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
																			TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Outer",
																			ReferenceLocation: ast_domain.Location{
																				Line:   30,
																				Column: 45,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   69,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Middle",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Middle",
																			ReferenceLocation: ast_domain.Location{
																				Line:   30,
																				Column: 45,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   64,
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
																		TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Middle",
																		ReferenceLocation: ast_domain.Location{
																			Line:   30,
																			Column: 45,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   64,
																			Column: 20,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 20,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   30,
																			Column: 45,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   61,
																			Column: 2,
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   30,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 2,
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
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
																	Column: 2,
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
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   31,
											Column: 9,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   31,
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
												Value: "triple-nested-unguarded",
												Location: ast_domain.Location{
													Line:   31,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   31,
													Column: 15,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   31,
													Column: 44,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   31,
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
															Line:   31,
															Column: 47,
														},
														RawExpression: "props.Outer.Middle.Inner.Value",
														Expression: &ast_domain.MemberExpression{
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
																					CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   31,
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
																			Name: "Outer",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																					PackageAlias:         "main_aaf9a2e0",
																					CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Outer",
																					ReferenceLocation: ast_domain.Location{
																						Line:   31,
																						Column: 47,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   69,
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
																				TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Outer",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 47,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   69,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Middle",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 13,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Middle",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 47,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   64,
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
																			TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Middle",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   64,
																				Column: 20,
																			},
																		},
																		BaseCodeGenVarName:  new("props"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Inner",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 20,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*main_aaf9a2e0.Inner"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Inner",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 47,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   62,
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
																		TypeExpression:       typeExprFromString("*main_aaf9a2e0.Inner"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Inner",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   62,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 26,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 47,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   59,
																			Column: 20,
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   59,
																		Column: 20,
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
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   59,
																	Column: 20,
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
							Line:   35,
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
								Line:   35,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   35,
								Column: 10,
							},
							RawExpression: "props.Valid != nil && props.Enabled",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.BinaryExpression{
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
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
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
											Name: "Valid",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Valid"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Valid",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   70,
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
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Valid"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Valid",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   70,
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
											Column: 16,
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
								Operator: "&&",
								Right: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
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
										Name: "Enabled",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 29,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Enabled",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
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
										Column: 23,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Enabled",
											ReferenceLocation: ast_domain.Location{
												Line:   35,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   71,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
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
								Line:   35,
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
									Line:   36,
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
										Line:   36,
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
										Value: "and-with-unrelated",
										Location: ast_domain.Location{
											Line:   36,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   36,
											Column: 13,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   36,
											Column: 37,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   36,
												Column: 37,
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
													Line:   36,
													Column: 40,
												},
												RawExpression: "props.Valid.Name",
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
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
															Name: "Valid",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Valid"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Valid",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
																		Column: 40,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   70,
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Valid"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Valid",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 40,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   70,
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
															Column: 13,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 40,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   65,
																	Column: 20,
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   36,
																Column: 40,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   65,
																Column: 20,
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
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   36,
															Column: 40,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   65,
															Column: 20,
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
							Line:   39,
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
								Line:   39,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   39,
								Column: 10,
							},
							RawExpression: "props.A != nil && props.B != nil",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.BinaryExpression{
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
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
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
												BaseCodeGenVarName: new("props"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "A",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "A",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   67,
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
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "A",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   67,
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
											Column: 12,
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
								Operator: "&&",
								Right: &ast_domain.BinaryExpression{
									Left: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "props",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 19,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
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
												BaseCodeGenVarName: new("props"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "B",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 25,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "B",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   68,
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
											Column: 19,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "B",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   68,
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
											Column: 30,
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
										Column: 19,
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
							Value: "r.0:3",
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
								OriginalSourcePath: new("main.pk"),
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
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
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
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "both-guarded-a",
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
											Column: 33,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   40,
												Column: 33,
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
													Line:   40,
													Column: 36,
												},
												RawExpression: "props.A.Value",
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
																		Column: 36,
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
															Name: "A",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "A",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
																		Column: 36,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "A",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   67,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   58,
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   40,
																Column: 36,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   58,
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
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   40,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   58,
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
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   41,
									Column: 7,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:1",
									RelativeLocation: ast_domain.Location{
										Line:   41,
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
										Value: "both-guarded-b",
										Location: ast_domain.Location{
											Line:   41,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   41,
											Column: 13,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   41,
											Column: 33,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 33,
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
													Line:   41,
													Column: 36,
												},
												RawExpression: "props.B.Value",
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
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "props",
																	ReferenceLocation: ast_domain.Location{
																		Line:   41,
																		Column: 36,
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
															Name: "B",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "B",
																	ReferenceLocation: ast_domain.Location{
																		Line:   41,
																		Column: 36,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   68,
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Data"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "B",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   58,
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   41,
																Column: 36,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   58,
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
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   58,
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
							Line:   44,
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
								Line:   44,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   44,
								Column: 10,
							},
							RawExpression: "props.Outer != nil",
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
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   44,
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
										Name: "Outer",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Outer",
												ReferenceLocation: ast_domain.Location{
													Line:   44,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   69,
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
											TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Outer",
											ReferenceLocation: ast_domain.Location{
												Line:   44,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   69,
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
										Column: 16,
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
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   44,
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
									Line:   45,
									Column: 7,
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
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   45,
										Column: 12,
									},
									RawExpression: "props.Outer.Middle != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
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
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   45,
																Column: 18,
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
													Name: "Outer",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Outer",
															ReferenceLocation: ast_domain.Location{
																Line:   45,
																Column: 18,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   69,
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
														TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Outer",
														ReferenceLocation: ast_domain.Location{
															Line:   45,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   69,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Middle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 13,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Middle",
														ReferenceLocation: ast_domain.Location{
															Line:   45,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   64,
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
													TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Middle",
													ReferenceLocation: ast_domain.Location{
														Line:   45,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   64,
														Column: 20,
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
												Column: 23,
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
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   45,
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   46,
											Column: 9,
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
												Line:   46,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   46,
												Column: 14,
											},
											RawExpression: "props.Outer.Middle.Inner != nil",
											Expression: &ast_domain.BinaryExpression{
												Left: &ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   46,
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
																Name: "Outer",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Outer",
																		ReferenceLocation: ast_domain.Location{
																			Line:   46,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   69,
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
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Outer",
																	ReferenceLocation: ast_domain.Location{
																		Line:   46,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   69,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Middle",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Middle",
																	ReferenceLocation: ast_domain.Location{
																		Line:   46,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   64,
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
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Middle",
																ReferenceLocation: ast_domain.Location{
																	Line:   46,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   64,
																	Column: 20,
																},
															},
															BaseCodeGenVarName:  new("props"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Inner",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*main_aaf9a2e0.Inner"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Inner",
																ReferenceLocation: ast_domain.Location{
																	Line:   46,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   62,
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
															TypeExpression:       typeExprFromString("*main_aaf9a2e0.Inner"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Inner",
															ReferenceLocation: ast_domain.Location{
																Line:   46,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
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
														Column: 29,
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
											Value: "r.0:4:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   46,
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
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   47,
													Column: 11,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:4:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   47,
														Column: 11,
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
														Value: "fully-guarded-deep",
														Location: ast_domain.Location{
															Line:   47,
															Column: 21,
														},
														NameLocation: ast_domain.Location{
															Line:   47,
															Column: 17,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   47,
															Column: 41,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("main_aaf9a2e0"),
															OriginalSourcePath:   new("main.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:4:0:0:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   47,
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
																	Line:   47,
																	Column: 44,
																},
																RawExpression: "props.Outer.Middle.Inner.Value",
																Expression: &ast_domain.MemberExpression{
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
																							CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "props",
																							ReferenceLocation: ast_domain.Location{
																								Line:   47,
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
																					Name: "Outer",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 7,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																							PackageAlias:         "main_aaf9a2e0",
																							CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "Outer",
																							ReferenceLocation: ast_domain.Location{
																								Line:   47,
																								Column: 44,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   69,
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
																						TypeExpression:       typeExprFromString("*main_aaf9a2e0.Outer"),
																						PackageAlias:         "main_aaf9a2e0",
																						CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Outer",
																						ReferenceLocation: ast_domain.Location{
																							Line:   47,
																							Column: 44,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   69,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("props"),
																					OriginalSourcePath:  new("main.pk"),
																					GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Middle",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 13,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																						PackageAlias:         "main_aaf9a2e0",
																						CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Middle",
																						ReferenceLocation: ast_domain.Location{
																							Line:   47,
																							Column: 44,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   64,
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
																					TypeExpression:       typeExprFromString("*main_aaf9a2e0.Middle"),
																					PackageAlias:         "main_aaf9a2e0",
																					CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Middle",
																					ReferenceLocation: ast_domain.Location{
																						Line:   47,
																						Column: 44,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   64,
																						Column: 20,
																					},
																				},
																				BaseCodeGenVarName:  new("props"),
																				OriginalSourcePath:  new("main.pk"),
																				GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "Inner",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 20,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("*main_aaf9a2e0.Inner"),
																					PackageAlias:         "main_aaf9a2e0",
																					CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Inner",
																					ReferenceLocation: ast_domain.Location{
																						Line:   47,
																						Column: 44,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   62,
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
																				TypeExpression:       typeExprFromString("*main_aaf9a2e0.Inner"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Inner",
																				ReferenceLocation: ast_domain.Location{
																					Line:   47,
																					Column: 44,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   62,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Value",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 26,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "main_aaf9a2e0",
																				CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Value",
																				ReferenceLocation: ast_domain.Location{
																					Line:   47,
																					Column: 44,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   59,
																					Column: 20,
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
																			CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Value",
																			ReferenceLocation: ast_domain.Location{
																				Line:   47,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   59,
																				Column: 20,
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
																		CanonicalPackagePath: "testcase_89_nil_guard_edge_cases/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   47,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   59,
																			Column: 20,
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
						},
					},
				},
			},
		},
	}
}()
