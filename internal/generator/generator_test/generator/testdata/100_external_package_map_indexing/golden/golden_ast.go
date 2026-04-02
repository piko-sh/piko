package main_test

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
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
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
									Column: 9,
								},
								TextContent: "External Package Map Indexing Test",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   24,
								Column: 36,
							},
							NameLocation: ast_domain.Location{
								Line:   24,
								Column: 28,
							},
							RawExpression: "domain.ParishMap[props.Parish]",
							Expression: &ast_domain.IndexExpression{
								Base: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "domain",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       nil,
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
											},
											BaseCodeGenVarName: new("domain"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ParishMap",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("map[string]string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ParishMap",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   21,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("domain"),
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
											TypeExpression:       typeExprFromString("map[string]string"),
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ParishMap",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 36,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   21,
												Column: 5,
											},
										},
										BaseCodeGenVarName: new("domain"),
									},
								},
								Index: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 18,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("pages_main_594861c5.Props"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props"),
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Parish",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 24,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Parish",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   40,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props"),
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
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Parish",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 36,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   40,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Optional: false,
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
									BaseCodeGenVarName: new("domain"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								BaseCodeGenVarName: new("domain"),
								Stringability:      1,
							},
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "parish-name",
								Location: ast_domain.Location{
									Line:   24,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 8,
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
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   25,
								Column: 36,
							},
							NameLocation: ast_domain.Location{
								Line:   25,
								Column: 28,
							},
							RawExpression: "domain.StatusCodes[props.Index]",
							Expression: &ast_domain.IndexExpression{
								Base: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "domain",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       nil,
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
											},
											BaseCodeGenVarName: new("domain"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "StatusCodes",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]int"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "StatusCodes",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   36,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("domain"),
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
											TypeExpression:       typeExprFromString("[]int"),
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "StatusCodes",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 36,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   36,
												Column: 5,
											},
										},
										BaseCodeGenVarName: new("domain"),
									},
								},
								Index: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 20,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("pages_main_594861c5.Props"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props"),
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Index",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 26,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_100_external_package_map_indexing/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Index",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   41,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									Optional: false,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 20,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Index",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 36,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   41,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Optional: false,
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									BaseCodeGenVarName: new("domain"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("int"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								BaseCodeGenVarName: new("domain"),
								Stringability:      1,
							},
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "status-code",
								Location: ast_domain.Location{
									Line:   25,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 8,
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
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   26,
								Column: 33,
							},
							NameLocation: ast_domain.Location{
								Line:   26,
								Column: 25,
							},
							RawExpression: "domain.AppName",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "domain",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       nil,
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										BaseCodeGenVarName: new("domain"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "AppName",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 8,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "AppName",
											ReferenceLocation: ast_domain.Location{
												Line:   26,
												Column: 33,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   38,
												Column: 5,
											},
										},
										BaseCodeGenVarName: new("domain"),
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
										PackageAlias:         "domain",
										CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "AppName",
										ReferenceLocation: ast_domain.Location{
											Line:   26,
											Column: 33,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   38,
											Column: 5,
										},
									},
									BaseCodeGenVarName: new("domain"),
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "domain",
									CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "AppName",
									ReferenceLocation: ast_domain.Location{
										Line:   26,
										Column: 33,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   38,
										Column: 5,
									},
								},
								BaseCodeGenVarName: new("domain"),
							},
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "app-name",
								Location: ast_domain.Location{
									Line:   26,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   26,
									Column: 8,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   27,
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
								Line:   27,
								Column: 36,
							},
							NameLocation: ast_domain.Location{
								Line:   27,
								Column: 28,
							},
							RawExpression: "domain.MaxRetries",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "domain",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       nil,
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										BaseCodeGenVarName: new("domain"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "MaxRetries",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 8,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("domain.any /* failed to parse type string: untyped_int */"),
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "MaxRetries",
											ReferenceLocation: ast_domain.Location{
												Line:   27,
												Column: 36,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   40,
												Column: 7,
											},
										},
										BaseCodeGenVarName:   new("domain"),
										IsStatic:             true,
										IsStructurallyStatic: true,
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
										TypeExpression:       typeExprFromString("domain.any /* failed to parse type string: untyped_int */"),
										PackageAlias:         "domain",
										CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "MaxRetries",
										ReferenceLocation: ast_domain.Location{
											Line:   27,
											Column: 36,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   40,
											Column: 7,
										},
									},
									BaseCodeGenVarName:   new("domain"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("domain.any /* failed to parse type string: untyped_int */"),
									PackageAlias:         "domain",
									CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "MaxRetries",
									ReferenceLocation: ast_domain.Location{
										Line:   27,
										Column: 36,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   40,
										Column: 7,
									},
								},
								BaseCodeGenVarName:   new("domain"),
								IsStatic:             true,
								IsStructurallyStatic: true,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "max-retries",
								Location: ast_domain.Location{
									Line:   27,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   27,
									Column: 8,
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
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirText: &ast_domain.Directive{
							Type: ast_domain.DirectiveText,
							Location: ast_domain.Location{
								Line:   28,
								Column: 32,
							},
							NameLocation: ast_domain.Location{
								Line:   28,
								Column: 24,
							},
							RawExpression: "domain.DefaultTimeout",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "domain",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       nil,
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										BaseCodeGenVarName: new("domain"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "DefaultTimeout",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 8,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("domain.any /* failed to parse type string: untyped_string */"),
											PackageAlias:         "domain",
											CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "DefaultTimeout",
											ReferenceLocation: ast_domain.Location{
												Line:   28,
												Column: 32,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   42,
												Column: 7,
											},
										},
										BaseCodeGenVarName:   new("domain"),
										IsStatic:             true,
										IsStructurallyStatic: true,
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
										TypeExpression:       typeExprFromString("domain.any /* failed to parse type string: untyped_string */"),
										PackageAlias:         "domain",
										CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "DefaultTimeout",
										ReferenceLocation: ast_domain.Location{
											Line:   28,
											Column: 32,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   42,
											Column: 7,
										},
									},
									BaseCodeGenVarName:   new("domain"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("domain.any /* failed to parse type string: untyped_string */"),
									PackageAlias:         "domain",
									CanonicalPackagePath: "testcase_100_external_package_map_indexing/pkg/domain",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "DefaultTimeout",
									ReferenceLocation: ast_domain.Location{
										Line:   28,
										Column: 32,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   42,
										Column: 7,
									},
								},
								BaseCodeGenVarName:   new("domain"),
								IsStatic:             true,
								IsStructurallyStatic: true,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "timeout",
								Location: ast_domain.Location{
									Line:   28,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   28,
									Column: 8,
								},
							},
						},
					},
				},
			},
		},
	}
}()
