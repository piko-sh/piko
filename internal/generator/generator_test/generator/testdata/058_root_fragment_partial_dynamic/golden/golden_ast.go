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
				NodeType: ast_domain.NodeFragment,
				Location: ast_domain.Location{
					Line:   0,
					Column: 0,
				},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "layout_1745aa65",
						PartialAlias:        "layout",
						PartialPackageName:  "partials_layout_ee037d9a",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"test": "pages_main_594861c5",
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
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
						OriginalSourcePath: new("partials/layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "main-content",
						Location: ast_domain.Location{
							Line:   22,
							Column: 55,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 48,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "test",
						RawExpression: "state.Test",
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
										CanonicalPackagePath: "testcase_058_root_fragment_partial_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 36,
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
								Name: "Test",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_058_root_fragment_partial_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Test",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 36,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   36,
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
									CanonicalPackagePath: "testcase_058_root_fragment_partial_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Test",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 36,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   36,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						Location: ast_domain.Location{
							Line:   22,
							Column: 36,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 29,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_058_root_fragment_partial_dynamic/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "Test",
								ReferenceLocation: ast_domain.Location{
									Line:   22,
									Column: 36,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   36,
									Column: 23,
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
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("partials/layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "simple-component",
								Location: ast_domain.Location{
									Line:   22,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 8,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "layout_test_state_test_048840a3",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "0",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
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
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										OriginalSourcePath: new("partials/layout.pk"),
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
										TagName: "h1",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
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
													Column: 9,
												},
												TextContent: "Main Page Title",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 9,
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
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   25,
							Column: 3,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   25,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "layout_test_state_test_048840a3",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "1",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   25,
									Column: 6,
								},
								TextContent: "Hello",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   25,
										Column: 6,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/layout.pk"),
										Stringability:      1,
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
