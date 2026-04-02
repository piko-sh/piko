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
					Line:   0,
					Column: 0,
				},
				TagName: "ul",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "li",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
							NameLocation: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
							RawExpression: "(i, user) in state.Users",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: &ast_domain.Identifier{
									Name: "i",
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "i",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("i"),
										OriginalSourcePath: new("main.pk"),
									},
								},
								ItemVariable: &ast_domain.Identifier{
									Name: "user",
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("models.User"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_02_pfor_scoping/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "user",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("user"),
										OriginalSourcePath: new("main.pk"),
									},
								},
								Collection: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_02_pfor_scoping/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
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
										Name: "Users",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]models.User"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_02_pfor_scoping/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Users",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   28,
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
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]models.User"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_02_pfor_scoping/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Users",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   28,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]models.User"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_02_pfor_scoping/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   28,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]models.User"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_02_pfor_scoping/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   28,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]models.User"),
									PackageAlias:         "models",
									CanonicalPackagePath: "testcase_02_pfor_scoping/models",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Users",
									ReferenceLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   28,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										RawExpression: "i",
										Expression: &ast_domain.Identifier{
											Name: "i",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "i",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("i"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "i",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("i"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										Literal: ": ",
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										RawExpression: "user.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "user",
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_02_pfor_scoping/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "user",
														ReferenceLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("user"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_02_pfor_scoping/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   42,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("user"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/user.go"),
													Stringability:       1,
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_02_pfor_scoping/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   42,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("user"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/user.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_02_pfor_scoping/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   42,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("user"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/user.go"),
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
					Line:   0,
					Column: 0,
				},
				TagName: "p",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								RawExpression: "user",
								Expression: &ast_domain.Identifier{
									Name: "user",
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("any"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("any"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
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
