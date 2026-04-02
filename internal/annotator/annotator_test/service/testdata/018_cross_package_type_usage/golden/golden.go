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
						Name:  "id",
						Value: "user-display",
						Location: ast_domain.Location{
							Line:   41,
							Column: 14,
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
						TagName: "p",
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   42,
									Column: 12,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   42,
										Column: 12,
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
											Line:   42,
											Column: 12,
										},
										Literal: "\n            User Details: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   43,
											Column: 30,
										},
										RawExpression: "util.FormatUser(state.CurrentUser)",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "util",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "util",
															CanonicalPackagePath: "testcase_18_cross_package_type_usage/util",
														},
														BaseCodeGenVarName: new("util"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormatUser",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "util",
															CanonicalPackagePath: "testcase_18_cross_package_type_usage/util",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FormatUser",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("util"),
													},
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
														PackageAlias:         "util",
														CanonicalPackagePath: "testcase_18_cross_package_type_usage/util",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FormatUser",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 30,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("util"),
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 17,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_18_cross_package_type_usage/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 30,
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
															Column: 23,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_18_cross_package_type_usage/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CurrentUser",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 30,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   29,
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
														Column: 17,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*models.User"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_18_cross_package_type_usage/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CurrentUser",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 30,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
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
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "util",
													CanonicalPackagePath: "testcase_18_cross_package_type_usage/util",
												},
												BaseCodeGenVarName: new("util"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "util",
												CanonicalPackagePath: "testcase_18_cross_package_type_usage/util",
											},
											BaseCodeGenVarName: new("util"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   43,
											Column: 67,
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
