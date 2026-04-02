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
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirHTML: &ast_domain.Directive{
							Type: ast_domain.DirectiveHtml,
							Location: ast_domain.Location{
								Line:   23,
								Column: 18,
							},
							NameLocation: ast_domain.Location{
								Line:   23,
								Column: 10,
							},
							RawExpression: "state.SafeHTML",
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
											CanonicalPackagePath: "testcase_039_p_html_no_escaping/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 18,
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
									Name: "SafeHTML",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_039_p_html_no_escaping/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "SafeHTML",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 18,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 23,
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
										CanonicalPackagePath: "testcase_039_p_html_no_escaping/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "SafeHTML",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 18,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
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
									CanonicalPackagePath: "testcase_039_p_html_no_escaping/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "SafeHTML",
									ReferenceLocation: ast_domain.Location{
										Line:   23,
										Column: 18,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   31,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
					},
				},
			},
		},
	}
}()
