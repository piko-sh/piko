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
					OriginalPackageAlias: new("partials_partial_a840a93c"),
					OriginalSourcePath:   new("partials/partial.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "partial_state_state_state_da5e9caa",
						PartialAlias:        "partial",
						PartialPackageName:  "partials_partial_a840a93c",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"state": ast_domain.PropValue{
								Expression: &ast_domain.UnaryExpression{
									Operator: "&",
									Right: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_028_optional_prop/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 45,
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
											Name: "State",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("data.State"),
													PackageAlias:         "data",
													CanonicalPackagePath: "testcase_028_optional_prop/pkg/data",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "State",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
														Column: 23,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("data.State"),
														PackageAlias:         "data",
														CanonicalPackagePath: "testcase_028_optional_prop/pkg/data",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "State",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   35,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
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
												TypeExpression:       typeExprFromString("data.State"),
												PackageAlias:         "data",
												CanonicalPackagePath: "testcase_028_optional_prop/pkg/data",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "State",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   35,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("data.State"),
													PackageAlias:         "data",
													CanonicalPackagePath: "testcase_028_optional_prop/pkg/data",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "State",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 45,
								},
								GoFieldName: "State",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("*data.State"),
										PackageAlias:         "partials_partial_a840a93c",
										CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "State",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 45,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   35,
											Column: 23,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("data.State"),
											PackageAlias:         "data",
											CanonicalPackagePath: "testcase_028_optional_prop/pkg/data",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "State",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 45,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   35,
												Column: 23,
											},
										},
										BaseCodeGenVarName: new("pageData"),
									},
									BaseCodeGenVarName: new("pageData"),
								},
							},
						},
					},
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
						OriginalSourcePath: new("partials/partial.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "partial",
						Location: ast_domain.Location{
							Line:   22,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   22,
							Column: 24,
						},
						TextContent: "Hello All: ",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_partial_a840a93c"),
							OriginalSourcePath:   new("partials/partial.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 24,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/partial.pk"),
								Stringability:      1,
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 35,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_partial_a840a93c"),
							OriginalSourcePath:   new("partials/partial.pk"),
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   22,
								Column: 49,
							},
							NameLocation: ast_domain.Location{
								Line:   22,
								Column: 41,
							},
							RawExpression: "state.Text",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("partials_partial_a840a93c.Response"),
											PackageAlias:         "partials_partial_a840a93c",
											CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 49,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
										OriginalSourcePath: new("partials/partial.pk"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "Text",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_partial_a840a93c",
											CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Text",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 49,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
										OriginalSourcePath:  new("partials/partial.pk"),
										GeneratedSourcePath: new("dist/partials/partials_partial_a840a93c/generated.go"),
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
										PackageAlias:         "partials_partial_a840a93c",
										CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Text",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 49,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
									OriginalSourcePath:  new("partials/partial.pk"),
									GeneratedSourcePath: new("dist/partials/partials_partial_a840a93c/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "partials_partial_a840a93c",
									CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Text",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 49,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   37,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
								OriginalSourcePath:  new("partials/partial.pk"),
								GeneratedSourcePath: new("dist/partials/partials_partial_a840a93c/generated.go"),
								Stringability:       1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 35,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/partial.pk"),
								Stringability:      1,
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   22,
							Column: 68,
						},
						TextContent: " & ",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_partial_a840a93c"),
							OriginalSourcePath:   new("partials/partial.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 68,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/partial.pk"),
								Stringability:      1,
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 71,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_partial_a840a93c"),
							OriginalSourcePath:   new("partials/partial.pk"),
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   22,
								Column: 85,
							},
							NameLocation: ast_domain.Location{
								Line:   22,
								Column: 77,
							},
							RawExpression: "state.OtherText",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("partials_partial_a840a93c.Response"),
											PackageAlias:         "partials_partial_a840a93c",
											CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 85,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
										OriginalSourcePath: new("partials/partial.pk"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "OtherText",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_partial_a840a93c",
											CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "OtherText",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 85,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   38,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
										OriginalSourcePath:  new("partials/partial.pk"),
										GeneratedSourcePath: new("dist/partials/partials_partial_a840a93c/generated.go"),
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
										PackageAlias:         "partials_partial_a840a93c",
										CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "OtherText",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 85,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   38,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
									OriginalSourcePath:  new("partials/partial.pk"),
									GeneratedSourcePath: new("dist/partials/partials_partial_a840a93c/generated.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "partials_partial_a840a93c",
									CanonicalPackagePath: "testcase_028_optional_prop/dist/partials/partials_partial_a840a93c",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "OtherText",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 85,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   38,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("partials_partial_a840a93cData_partial_state_state_state_da5e9caa"),
								OriginalSourcePath:  new("partials/partial.pk"),
								GeneratedSourcePath: new("dist/partials/partials_partial_a840a93c/generated.go"),
								Stringability:       1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 71,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/partial.pk"),
								Stringability:      1,
							},
						},
					},
				},
			},
		},
	}
}()
