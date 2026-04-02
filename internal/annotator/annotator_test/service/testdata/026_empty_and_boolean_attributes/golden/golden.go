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
					Line:   40,
					Column: 5,
				},
				TagName: "form",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   40,
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
							Line:   41,
							Column: 9,
						},
						TagName: "input",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   41,
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
								Name:  "disabled",
								Value: "",
								Location: ast_domain.Location{
									Line:   41,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 16,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "input1",
								Location: ast_domain.Location{
									Line:   41,
									Column: 29,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 25,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   43,
							Column: 9,
						},
						TagName: "input",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   43,
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
								Value: "input2",
								Location: ast_domain.Location{
									Line:   43,
									Column: 49,
								},
								NameLocation: ast_domain.Location{
									Line:   43,
									Column: 45,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "disabled",
								RawExpression: "state.IsDisabled",
								Expression: &ast_domain.MemberExpression{
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
												CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   43,
													Column: 27,
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
										Name: "IsDisabled",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "IsDisabled",
												ReferenceLocation: ast_domain.Location{
													Line:   43,
													Column: 27,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   26,
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
											CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IsDisabled",
											ReferenceLocation: ast_domain.Location{
												Line:   43,
												Column: 27,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   26,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   43,
									Column: 27,
								},
								NameLocation: ast_domain.Location{
									Line:   43,
									Column: 16,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsDisabled",
										ReferenceLocation: ast_domain.Location{
											Line:   43,
											Column: 27,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   26,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       1,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   45,
							Column: 9,
						},
						TagName: "input",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   45,
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
								Value: "input3",
								Location: ast_domain.Location{
									Line:   45,
									Column: 29,
								},
								NameLocation: ast_domain.Location{
									Line:   45,
									Column: 25,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "value",
								Value: "",
								Location: ast_domain.Location{
									Line:   45,
									Column: 23,
								},
								NameLocation: ast_domain.Location{
									Line:   45,
									Column: 16,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   47,
							Column: 9,
						},
						TagName: "input",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   47,
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
								Value: "input4",
								Location: ast_domain.Location{
									Line:   47,
									Column: 44,
								},
								NameLocation: ast_domain.Location{
									Line:   47,
									Column: 40,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "value",
								RawExpression: "state.EmptyVal",
								Expression: &ast_domain.MemberExpression{
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
												CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 24,
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
										Name: "EmptyVal",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "EmptyVal",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 24,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   27,
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
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "EmptyVal",
											ReferenceLocation: ast_domain.Location{
												Line:   47,
												Column: 24,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   27,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   47,
									Column: 24,
								},
								NameLocation: ast_domain.Location{
									Line:   47,
									Column: 16,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_26_empty_and_boolean_attributes/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "EmptyVal",
										ReferenceLocation: ast_domain.Location{
											Line:   47,
											Column: 24,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   27,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
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
	}
}()
