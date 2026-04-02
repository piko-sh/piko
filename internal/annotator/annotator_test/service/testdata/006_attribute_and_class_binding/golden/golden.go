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
					Line:   44,
					Column: 5,
				},
				TagName: "a",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				DirClass: &ast_domain.Directive{
					Type: ast_domain.DirectiveClass,
					Location: ast_domain.Location{
						Line:   48,
						Column: 18,
					},
					NameLocation: ast_domain.Location{
						Line:   48,
						Column: 9,
					},
					RawExpression: "{ 'active': state.IsActive, 'inactive': !state.IsActive }",
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"active": &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 13,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   48,
												Column: 18,
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
									Name: "IsActive",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 19,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IsActive",
											ReferenceLocation: ast_domain.Location{
												Line:   48,
												Column: 18,
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
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsActive",
										ReferenceLocation: ast_domain.Location{
											Line:   48,
											Column: 18,
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
							"inactive": &ast_domain.UnaryExpression{
								Operator: "!",
								Right: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 42,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   48,
													Column: 18,
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
										Name: "IsActive",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 48,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "IsActive",
												ReferenceLocation: ast_domain.Location{
													Line:   48,
													Column: 18,
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
										Column: 42,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IsActive",
											ReferenceLocation: ast_domain.Location{
												Line:   48,
												Column: 18,
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
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 41,
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
						},
						RelativeLocation: ast_domain.Location{
							Line:   1,
							Column: 1,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("map[string]bool"),
								PackageAlias:         "main_aaf9a2e0",
								CanonicalPackagePath: "",
							},
							OriginalSourcePath: new("main.pk"),
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("map[string]bool"),
							PackageAlias:         "main_aaf9a2e0",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("main.pk"),
					},
				},
				DirStyle: &ast_domain.Directive{
					Type: ast_domain.DirectiveStyle,
					Location: ast_domain.Location{
						Line:   49,
						Column: 18,
					},
					NameLocation: ast_domain.Location{
						Line:   49,
						Column: 9,
					},
					RawExpression: "{ color: state.Color, 'font-weight': 'bold' }",
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"color": &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 10,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   49,
												Column: 18,
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
									Name: "Color",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 16,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Color",
											ReferenceLocation: ast_domain.Location{
												Line:   49,
												Column: 18,
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
									Column: 10,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Color",
										ReferenceLocation: ast_domain.Location{
											Line:   49,
											Column: 18,
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
							"font-weight": &ast_domain.StringLiteral{
								Value: "bold",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 38,
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
						RelativeLocation: ast_domain.Location{
							Line:   1,
							Column: 1,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("map[string]string"),
								PackageAlias:         "main_aaf9a2e0",
								CanonicalPackagePath: "",
							},
							OriginalSourcePath: new("main.pk"),
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("map[string]string"),
							PackageAlias:         "main_aaf9a2e0",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("main.pk"),
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "my-link",
						Location: ast_domain.Location{
							Line:   45,
							Column: 13,
						},
						NameLocation: ast_domain.Location{
							Line:   45,
							Column: 9,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "class",
						RawExpression: "state.ExtraClasses",
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
										CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   47,
											Column: 17,
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
								Name: "ExtraClasses",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ExtraClasses",
										ReferenceLocation: ast_domain.Location{
											Line:   47,
											Column: 17,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   28,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       5,
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
									TypeExpression:       typeExprFromString("[]string"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "ExtraClasses",
									ReferenceLocation: ast_domain.Location{
										Line:   47,
										Column: 17,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   28,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								Stringability:       5,
							},
						},
						Location: ast_domain.Location{
							Line:   47,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   47,
							Column: 9,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("[]string"),
								PackageAlias:         "main_aaf9a2e0",
								CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "ExtraClasses",
								ReferenceLocation: ast_domain.Location{
									Line:   47,
									Column: 17,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   28,
									Column: 2,
								},
							},
							BaseCodeGenVarName:  new("pageData"),
							OriginalSourcePath:  new("main.pk"),
							GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
							Stringability:       5,
						},
					},
					ast_domain.DynamicAttribute{
						Name:          "href",
						RawExpression: "state.URL",
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
										CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   46,
											Column: 16,
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
								Name: "URL",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "URL",
										ReferenceLocation: ast_domain.Location{
											Line:   46,
											Column: 16,
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
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "URL",
									ReferenceLocation: ast_domain.Location{
										Line:   46,
										Column: 16,
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
							Line:   46,
							Column: 16,
						},
						NameLocation: ast_domain.Location{
							Line:   46,
							Column: 9,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "main_aaf9a2e0",
								CanonicalPackagePath: "testcase_06_attribute_and_class_binding/dist/pages/main_aaf9a2e0",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "URL",
								ReferenceLocation: ast_domain.Location{
									Line:   46,
									Column: 16,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   49,
							Column: 65,
						},
						TextContent: " Click Me ",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   49,
								Column: 65,
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
		},
	}
}()
