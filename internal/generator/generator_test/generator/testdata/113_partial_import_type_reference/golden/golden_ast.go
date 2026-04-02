package partial_import_type_reference_test

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
					Line:   58,
					Column: 2,
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
						Line:   58,
						Column: 2,
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "main",
						Location: ast_domain.Location{
							Line:   58,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   58,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   59,
							Column: 3,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   59,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   59,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   59,
										Column: 7,
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
											Line:   59,
											Column: 7,
										},
										Literal: "User: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   59,
											Column: 16,
										},
										RawExpression: "state.CurrentUser.Name",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 16,
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
													Name: "CurrentUser",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_models_86b482ba.User"),
															PackageAlias:         "partials_models_86b482ba",
															CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CurrentUser",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   31,
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
														TypeExpression:       typeExprFromString("partials_models_86b482ba.User"),
														PackageAlias:         "partials_models_86b482ba",
														CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CurrentUser",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 19,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_models_86b482ba",
														CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   28,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/partials/partials_models_86b482ba/generated.go"),
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
													PackageAlias:         "partials_models_86b482ba",
													CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   28,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/partials/partials_models_86b482ba/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_models_86b482ba",
												CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   59,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   28,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/partials/partials_models_86b482ba/generated.go"),
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
							Line:   60,
							Column: 3,
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
								Line:   60,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   60,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   60,
										Column: 6,
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
											Line:   60,
											Column: 6,
										},
										Literal: "Email: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   60,
											Column: 16,
										},
										RawExpression: "state.CurrentUser.Email",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   60,
																Column: 16,
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
													Name: "CurrentUser",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_models_86b482ba.User"),
															PackageAlias:         "partials_models_86b482ba",
															CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CurrentUser",
															ReferenceLocation: ast_domain.Location{
																Line:   60,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   31,
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
														TypeExpression:       typeExprFromString("partials_models_86b482ba.User"),
														PackageAlias:         "partials_models_86b482ba",
														CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CurrentUser",
														ReferenceLocation: ast_domain.Location{
															Line:   60,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   31,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Email",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 19,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_models_86b482ba",
														CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Email",
														ReferenceLocation: ast_domain.Location{
															Line:   60,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/partials/partials_models_86b482ba/generated.go"),
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
													PackageAlias:         "partials_models_86b482ba",
													CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Email",
													ReferenceLocation: ast_domain.Location{
														Line:   60,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   29,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/partials/partials_models_86b482ba/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_models_86b482ba",
												CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/partials/partials_models_86b482ba",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Email",
												ReferenceLocation: ast_domain.Location{
													Line:   60,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   29,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/partials/partials_models_86b482ba/generated.go"),
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
							Line:   61,
							Column: 3,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   61,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   61,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   61,
										Column: 6,
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
											Line:   61,
											Column: 6,
										},
										Literal: "Formatted: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   61,
											Column: 20,
										},
										RawExpression: "state.UserDisplay",
										Expression: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 20,
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
												Name: "UserDisplay",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserDisplay",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserDisplay",
													ReferenceLocation: ast_domain.Location{
														Line:   61,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
														Column: 2,
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
												CanonicalPackagePath: "testcase_113_partial_import_type_reference/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserDisplay",
												ReferenceLocation: ast_domain.Location{
													Line:   61,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   32,
													Column: 2,
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
