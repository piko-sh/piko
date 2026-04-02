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
				},
				DirClass: &ast_domain.Directive{
					Type: ast_domain.DirectiveClass,
					Location: ast_domain.Location{
						Line:   22,
						Column: 83,
					},
					NameLocation: ast_domain.Location{
						Line:   22,
						Column: 74,
					},
					RawExpression: "{'card--active': state.IsActive, 'card--error': state.Status == 'error'}",
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"card--active": &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 18,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 83,
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
									Name: "IsActive",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 24,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IsActive",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 83,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   51,
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
									Column: 18,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsActive",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 83,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   51,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							"card--error": &ast_domain.BinaryExpression{
								Left: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 49,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 83,
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
										Name: "Status",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 55,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Status",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 83,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
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
										Column: 49,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Status",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 83,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   50,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Operator: "==",
								Right: &ast_domain.StringLiteral{
									Value: "error",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 65,
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
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 49,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									OriginalSourcePath: new("pages/main.pk"),
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
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "",
							},
							OriginalSourcePath: new("pages/main.pk"),
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("map[string]bool"),
							PackageAlias:         "pages_main_594861c5",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("pages/main.pk"),
					},
				},
				DirStyle: &ast_domain.Directive{
					Type: ast_domain.DirectiveStyle,
					Location: ast_domain.Location{
						Line:   22,
						Column: 166,
					},
					NameLocation: ast_domain.Location{
						Line:   22,
						Column: 157,
					},
					RawExpression: "{'background-color': state.BgColor,'font-size': '16px'}",
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"background-color": &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 22,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 166,
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
									Name: "BgColor",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 28,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "BgColor",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 166,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   53,
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
									Column: 22,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "BgColor",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 166,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   53,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							"font-size": &ast_domain.StringLiteral{
								Value: "16px",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 49,
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
						RelativeLocation: ast_domain.Location{
							Line:   1,
							Column: 1,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("map[string]string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "",
							},
							OriginalSourcePath: new("pages/main.pk"),
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("map[string]string"),
							PackageAlias:         "pages_main_594861c5",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("pages/main.pk"),
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
						OriginalSourcePath: new("pages/main.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "card",
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
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "id",
						RawExpression: "`item-${state.ID}`",
						Expression: &ast_domain.TemplateLiteral{
							Parts: []ast_domain.TemplateLiteralPart{
								ast_domain.TemplateLiteralPart{
									IsLiteral: true,
									RelativeLocation: ast_domain.Location{
										Line:   1,
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
									Literal: "item-",
								},
								ast_domain.TemplateLiteralPart{
									IsLiteral: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
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
													CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 26,
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
											Name: "ID",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 26,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
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
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ID",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 26,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Location: ast_domain.Location{
							Line:   22,
							Column: 26,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 21,
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
					ast_domain.DynamicAttribute{
						Name:          "data-status",
						RawExpression: "state.Status",
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
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 60,
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
								Name: "Status",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Status",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 60,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   50,
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
									CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Status",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 60,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   50,
										Column: 2,
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
							Column: 60,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 46,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "Status",
								ReferenceLocation: ast_domain.Location{
									Line:   22,
									Column: 60,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   50,
									Column: 2,
								},
							},
							BaseCodeGenVarName:  new("pageData"),
							OriginalSourcePath:  new("pages/main.pk"),
							GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
							Stringability:       1,
						},
					},
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
										TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 234,
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
								Name: "IsDisabled",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsDisabled",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 234,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   52,
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
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "IsDisabled",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 234,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   52,
										Column: 2,
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
							Column: 234,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 223,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("bool"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "IsDisabled",
								ReferenceLocation: ast_domain.Location{
									Line:   22,
									Column: 234,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   52,
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
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   23,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   23,
								Column: 8,
							},
							RawExpression: "state.Message",
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
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
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
									Name: "Message",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Message",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   49,
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
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Message",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   49,
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
									CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Message",
									ReferenceLocation: ast_domain.Location{
										Line:   23,
										Column: 16,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   49,
										Column: 2,
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
			&ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Location: ast_domain.Location{
					Line:   25,
					Column: 3,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
				},
				DirClass: &ast_domain.Directive{
					Type: ast_domain.DirectiveClass,
					Location: ast_domain.Location{
						Line:   29,
						Column: 14,
					},
					NameLocation: ast_domain.Location{
						Line:   29,
						Column: 5,
					},
					RawExpression: "{\n      'card--active': state.IsActive,\n      'card--error': state.Status == 'error'\n    }",
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"card--active": &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   2,
										Column: 23,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   29,
												Column: 14,
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
									Name: "IsActive",
									RelativeLocation: ast_domain.Location{
										Line:   2,
										Column: 29,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IsActive",
											ReferenceLocation: ast_domain.Location{
												Line:   29,
												Column: 14,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   51,
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
									Line:   2,
									Column: 23,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsActive",
										ReferenceLocation: ast_domain.Location{
											Line:   29,
											Column: 14,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   51,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							"card--error": &ast_domain.BinaryExpression{
								Left: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   3,
											Column: 22,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 14,
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
										Name: "Status",
										RelativeLocation: ast_domain.Location{
											Line:   3,
											Column: 28,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Status",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 14,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
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
										Line:   3,
										Column: 22,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Status",
											ReferenceLocation: ast_domain.Location{
												Line:   29,
												Column: 14,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   50,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Operator: "==",
								Right: &ast_domain.StringLiteral{
									Value: "error",
									RelativeLocation: ast_domain.Location{
										Line:   3,
										Column: 38,
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
								RelativeLocation: ast_domain.Location{
									Line:   3,
									Column: 22,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									OriginalSourcePath: new("pages/main.pk"),
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
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "",
							},
							OriginalSourcePath: new("pages/main.pk"),
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("map[string]bool"),
							PackageAlias:         "pages_main_594861c5",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("pages/main.pk"),
					},
				},
				DirStyle: &ast_domain.Directive{
					Type: ast_domain.DirectiveStyle,
					Location: ast_domain.Location{
						Line:   33,
						Column: 14,
					},
					NameLocation: ast_domain.Location{
						Line:   33,
						Column: 5,
					},
					RawExpression: "{\n      'background-color': state.BgColor,\n      'font-size': '16px'\n    }",
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"background-color": &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   2,
										Column: 27,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   33,
												Column: 14,
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
									Name: "BgColor",
									RelativeLocation: ast_domain.Location{
										Line:   2,
										Column: 33,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "BgColor",
											ReferenceLocation: ast_domain.Location{
												Line:   33,
												Column: 14,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   53,
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
									Line:   2,
									Column: 27,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "BgColor",
										ReferenceLocation: ast_domain.Location{
											Line:   33,
											Column: 14,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   53,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
							"font-size": &ast_domain.StringLiteral{
								Value: "16px",
								RelativeLocation: ast_domain.Location{
									Line:   3,
									Column: 20,
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
						RelativeLocation: ast_domain.Location{
							Line:   1,
							Column: 1,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("map[string]string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "",
							},
							OriginalSourcePath: new("pages/main.pk"),
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("map[string]string"),
							PackageAlias:         "pages_main_594861c5",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("pages/main.pk"),
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.1",
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
						OriginalSourcePath: new("pages/main.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "card",
						Location: ast_domain.Location{
							Line:   26,
							Column: 12,
						},
						NameLocation: ast_domain.Location{
							Line:   26,
							Column: 5,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "id",
						RawExpression: "`item-${state.ID}`",
						Expression: &ast_domain.TemplateLiteral{
							Parts: []ast_domain.TemplateLiteralPart{
								ast_domain.TemplateLiteralPart{
									IsLiteral: true,
									RelativeLocation: ast_domain.Location{
										Line:   1,
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
									Literal: "item-",
								},
								ast_domain.TemplateLiteralPart{
									IsLiteral: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
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
													CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 10,
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
											Name: "ID",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 10,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
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
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ID",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 10,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Location: ast_domain.Location{
							Line:   27,
							Column: 10,
						},
						NameLocation: ast_domain.Location{
							Line:   27,
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
					ast_domain.DynamicAttribute{
						Name:          "data-status",
						RawExpression: "state.Status",
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
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   28,
											Column: 19,
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
								Name: "Status",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Status",
										ReferenceLocation: ast_domain.Location{
											Line:   28,
											Column: 19,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   50,
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
									CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Status",
									ReferenceLocation: ast_domain.Location{
										Line:   28,
										Column: 19,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   50,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						Location: ast_domain.Location{
							Line:   28,
							Column: 19,
						},
						NameLocation: ast_domain.Location{
							Line:   28,
							Column: 5,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "Status",
								ReferenceLocation: ast_domain.Location{
									Line:   28,
									Column: 19,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   50,
									Column: 2,
								},
							},
							BaseCodeGenVarName:  new("pageData"),
							OriginalSourcePath:  new("pages/main.pk"),
							GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
							Stringability:       1,
						},
					},
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
										TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   37,
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
								Name: "IsDisabled",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "IsDisabled",
										ReferenceLocation: ast_domain.Location{
											Line:   37,
											Column: 16,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   52,
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
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "pages_main_594861c5",
									CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "IsDisabled",
									ReferenceLocation: ast_domain.Location{
										Line:   37,
										Column: 16,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   52,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						Location: ast_domain.Location{
							Line:   37,
							Column: 16,
						},
						NameLocation: ast_domain.Location{
							Line:   37,
							Column: 5,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("bool"),
								PackageAlias:         "pages_main_594861c5",
								CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "IsDisabled",
								ReferenceLocation: ast_domain.Location{
									Line:   37,
									Column: 16,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   52,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   39,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   39,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   39,
								Column: 8,
							},
							RawExpression: "state.Message",
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
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   39,
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
									Name: "Message",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Message",
											ReferenceLocation: ast_domain.Location{
												Line:   39,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   49,
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
										CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Message",
										ReferenceLocation: ast_domain.Location{
											Line:   39,
											Column: 16,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   49,
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
									CanonicalPackagePath: "testcase_009_attributes_static_and_dynamic/dist/pages/pages_main_594861c5",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Message",
									ReferenceLocation: ast_domain.Location{
										Line:   39,
										Column: 16,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   49,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								Stringability:       1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.1:0",
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
