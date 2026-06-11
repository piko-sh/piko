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
								TextContent: "Dashboard",
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
							Line:   22,
							Column: 3,
						},
						TagName: "header",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_appheader_4fd14528"),
							OriginalSourcePath:   new("partials/appheader.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "appheader_icon_state_header_icon_e54b7d0f",
								PartialAlias:        "appheader",
								PartialPackageName:  "partials_appheader_4fd14528",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"icon": ast_domain.PropValue{
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
															CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
															UnderlyingTypeString: "struct{Header *pages_main_594861c5.HeaderData}",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 68,
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
													Name: "Header",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.HeaderData"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
															UnderlyingTypeString: "struct{Icon string}",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Header",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 68,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 23,
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
														TypeExpression:       typeExprFromString("*pages_main_594861c5.HeaderData"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
														UnderlyingTypeString: "struct{Icon string}",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Header",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 68,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Icon",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 14,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Icon",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 68,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 25,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Icon",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 68,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 25,
															},
														},
														BaseCodeGenVarName: new("pageData"),
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
													CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Icon",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 68,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 25,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Icon",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 68,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 25,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 68,
										},
										GoFieldName: "Icon",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Icon",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 68,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 25,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Icon",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 68,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 25,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"icon": "pages_main_594861c5",
							},
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   24,
								Column: 25,
							},
							NameLocation: ast_domain.Location{
								Line:   24,
								Column: 19,
							},
							RawExpression: "state.Header != nil",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.MemberExpression{
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
												CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
												UnderlyingTypeString: "struct{Header *pages_main_594861c5.HeaderData}",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 25,
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
										Name: "Header",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*pages_main_594861c5.HeaderData"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
												UnderlyingTypeString: "struct{Icon string}",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Header",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 25,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 23,
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
											TypeExpression:       typeExprFromString("*pages_main_594861c5.HeaderData"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
											UnderlyingTypeString: "struct{Icon string}",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Header",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 25,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   38,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									},
								},
								Operator: "!=",
								Right: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 17,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
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
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("partials/appheader.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "app-header",
								Location: ast_domain.Location{
									Line:   22,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "icon",
								RawExpression: "state.Header.Icon",
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
													CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
													UnderlyingTypeString: "struct{Header *pages_main_594861c5.HeaderData}",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 68,
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
											Name: "Header",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.HeaderData"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
													UnderlyingTypeString: "struct{Icon string}",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Header",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 68,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 23,
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
												TypeExpression:       typeExprFromString("*pages_main_594861c5.HeaderData"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
												UnderlyingTypeString: "struct{Icon string}",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Header",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 68,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Icon",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 14,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Icon",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 68,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 25,
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
											CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Icon",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 68,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 25,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   24,
									Column: 68,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 61,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Icon",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 68,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
											Column: 25,
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
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_appheader_4fd14528"),
									OriginalSourcePath:   new("partials/appheader.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   23,
										Column: 32,
									},
									NameLocation: ast_domain.Location{
										Line:   23,
										Column: 24,
									},
									RawExpression: "state.Icon",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("partials_appheader_4fd14528.Props"),
													PackageAlias:         "partials_appheader_4fd14528",
													CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/partials/partials_appheader_4fd14528",
													UnderlyingTypeString: "struct{Icon string}",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_appheader_4fd14528Data_appheader_icon_state_header_icon_e54b7d0f"),
												OriginalSourcePath: new("partials/appheader.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Icon",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_appheader_4fd14528",
													CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/partials/partials_appheader_4fd14528",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Icon",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_appheader_4fd14528Data_appheader_icon_state_header_icon_e54b7d0f"),
												OriginalSourcePath:  new("partials/appheader.pk"),
												GeneratedSourcePath: new("dist/partials/partials_appheader_4fd14528/generated.go"),
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
												PackageAlias:         "partials_appheader_4fd14528",
												CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/partials/partials_appheader_4fd14528",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Icon",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   32,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_appheader_4fd14528Data_appheader_icon_state_header_icon_e54b7d0f"),
											OriginalSourcePath:  new("partials/appheader.pk"),
											GeneratedSourcePath: new("dist/partials/partials_appheader_4fd14528/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_appheader_4fd14528",
											CanonicalPackagePath: "testcase_144_partial_in_p_if/dist/partials/partials_appheader_4fd14528",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Icon",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 32,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   32,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_appheader_4fd14528Data_appheader_icon_state_header_icon_e54b7d0f"),
										OriginalSourcePath:  new("partials/appheader.pk"),
										GeneratedSourcePath: new("dist/partials/partials_appheader_4fd14528/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
										OriginalSourcePath: new("partials/appheader.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "icon",
										Location: ast_domain.Location{
											Line:   23,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 11,
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
