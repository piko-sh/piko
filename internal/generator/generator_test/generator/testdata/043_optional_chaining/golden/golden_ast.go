package default_test_pkg

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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   23,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
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
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   23,
											Column: 8,
										},
										Literal: "User street: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   23,
											Column: 24,
										},
										RawExpression: "state.User?.Address?.Street",
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 24,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "User",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 24,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "User",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 24,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Address",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 13,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Address",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 24,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   48,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
														TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Address",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 24,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   48,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Street",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 22,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Street",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 24,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   45,
															Column: 22,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Street",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 24,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   45,
														Column: 22,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Street",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 24,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 22,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   24,
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
									Line:   24,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   24,
											Column: 8,
										},
										Literal: "Guest street: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   24,
											Column: 25,
										},
										RawExpression: "state.Guest?.Address?.Street",
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 25,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Guest",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Guest",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 25,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   54,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Guest",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 25,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   54,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Address",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 14,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Address",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 25,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   48,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
														TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Address",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 25,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   48,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Street",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Street",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 25,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   45,
															Column: 22,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Street",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 25,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   45,
														Column: 22,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Street",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 25,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 22,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   25,
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
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "data-street",
								RawExpression: "state.User?.Address?.Street",
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
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "User",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "User",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "User",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   53,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Address",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 13,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Address",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
												TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Address",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Street",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 22,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Street",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 22,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Street",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 22,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   45,
												Column: 22,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   25,
									Column: 22,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Street",
										ReferenceLocation: ast_domain.Location{
											Line:   25,
											Column: 22,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   45,
											Column: 22,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   25,
									Column: 51,
								},
								TextContent: "User Attr",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   25,
										Column: 51,
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
							Line:   26,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   26,
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
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "data-street",
								RawExpression: "state.Guest?.Address?.Street",
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
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Guest",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Guest",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   54,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Guest",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   54,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Address",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 14,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Address",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
												TypeExpression:       typeExprFromString("*pages_main_594861c5.Address"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Address",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Street",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Street",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 22,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Street",
											ReferenceLocation: ast_domain.Location{
												Line:   26,
												Column: 22,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   45,
												Column: 22,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   26,
									Column: 22,
								},
								NameLocation: ast_domain.Location{
									Line:   26,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Street",
										ReferenceLocation: ast_domain.Location{
											Line:   26,
											Column: 22,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   45,
											Column: 22,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   26,
									Column: 52,
								},
								TextContent: "Guest Attr",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   26,
										Column: 52,
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
							Line:   28,
							Column: 5,
						},
						TagName: "hr",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   30,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   30,
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
									Line:   30,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   30,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   30,
											Column: 8,
										},
										Literal: "First Tag (User): ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   30,
											Column: 29,
										},
										RawExpression: "state.User?.Tags?.[0]",
										Expression: &ast_domain.IndexExpression{
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 29,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "User",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 29,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "User",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 29,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Tags",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 13,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Tags",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 29,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       5,
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
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tags",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 29,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
												},
											},
											Index: &ast_domain.IntegerLiteral{
												Value: 0,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 20,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											Optional: true,
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
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("pages/main.pk"),
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
							Line:   31,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
							RelativeLocation: ast_domain.Location{
								Line:   31,
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
									Line:   31,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
									RelativeLocation: ast_domain.Location{
										Line:   31,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   31,
											Column: 8,
										},
										Literal: "First Tag (Guest): ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   31,
											Column: 30,
										},
										RawExpression: "state.Guest?.Tags?.[0]",
										Expression: &ast_domain.IndexExpression{
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Guest",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Guest",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   54,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Guest",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   54,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Tags",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 14,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Tags",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       5,
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
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tags",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
												},
											},
											Index: &ast_domain.IntegerLiteral{
												Value: 0,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 21,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											Optional: true,
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
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("pages/main.pk"),
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
							Line:   32,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:7",
							RelativeLocation: ast_domain.Location{
								Line:   32,
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
									Line:   32,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
									RelativeLocation: ast_domain.Location{
										Line:   32,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   32,
											Column: 8,
										},
										Literal: "Out of Bounds Tag: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   32,
											Column: 30,
										},
										RawExpression: "state.User?.Tags?.[99]",
										Expression: &ast_domain.IndexExpression{
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "User",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "User",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Tags",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 13,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Tags",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       5,
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
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tags",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
												},
											},
											Index: &ast_domain.IntegerLiteral{
												Value: 99,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 20,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											Optional: true,
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
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("pages/main.pk"),
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
							Line:   34,
							Column: 5,
						},
						TagName: "hr",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:8",
							RelativeLocation: ast_domain.Location{
								Line:   34,
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
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   36,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:9",
							RelativeLocation: ast_domain.Location{
								Line:   36,
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
									Line:   36,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:9:0",
									RelativeLocation: ast_domain.Location{
										Line:   36,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   36,
											Column: 8,
										},
										Literal: "First Order Item Name (User): ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   36,
											Column: 41,
										},
										RawExpression: "state.User?.Orders?.[0]?.ItemName",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "User",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   53,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															},
														},
														Optional: false,
														Computed: false,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Orders",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 13,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]*pages_main_594861c5.Order"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Orders",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   50,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
															TypeExpression:       typeExprFromString("[]*pages_main_594861c5.Order"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Orders",
															ReferenceLocation: ast_domain.Location{
																Line:   36,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   50,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 0,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 22,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Optional: true,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pages_main_594861c5.Order"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "ItemName",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 26,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ItemName",
														ReferenceLocation: ast_domain.Location{
															Line:   36,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 20,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ItemName",
													ReferenceLocation: ast_domain.Location{
														Line:   36,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 20,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ItemName",
												ReferenceLocation: ast_domain.Location{
													Line:   36,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 20,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
							Line:   37,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:10",
							RelativeLocation: ast_domain.Location{
								Line:   37,
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
									Line:   37,
									Column: 8,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:10:0",
									RelativeLocation: ast_domain.Location{
										Line:   37,
										Column: 8,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   37,
											Column: 8,
										},
										Literal: "First Order Item Name (Guest): ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   37,
											Column: 42,
										},
										RawExpression: "state.Guest?.Orders?.[0]?.ItemName",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Guest",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Guest",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   54,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															},
														},
														Optional: false,
														Computed: false,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Guest",
																ReferenceLocation: ast_domain.Location{
																	Line:   37,
																	Column: 42,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   54,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Orders",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 14,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]*pages_main_594861c5.Order"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Orders",
																ReferenceLocation: ast_domain.Location{
																	Line:   37,
																	Column: 42,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   50,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
															TypeExpression:       typeExprFromString("[]*pages_main_594861c5.Order"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Orders",
															ReferenceLocation: ast_domain.Location{
																Line:   37,
																Column: 42,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   50,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 0,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 23,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Optional: true,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pages_main_594861c5.Order"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "ItemName",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 27,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ItemName",
														ReferenceLocation: ast_domain.Location{
															Line:   37,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 20,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ItemName",
													ReferenceLocation: ast_domain.Location{
														Line:   37,
														Column: 42,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 20,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_043_optional_chaining/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ItemName",
												ReferenceLocation: ast_domain.Location{
													Line:   37,
													Column: 42,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 20,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
