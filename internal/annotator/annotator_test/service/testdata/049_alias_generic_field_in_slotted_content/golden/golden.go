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
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_layout_ee037d9a"),
					OriginalSourcePath:   new("partials/layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "layout_1745aa65",
						PartialAlias:        "layout",
						PartialPackageName:  "partials_layout_ee037d9a",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   54,
							Column: 5,
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"page-title": "main_aaf9a2e0",
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
						OriginalSourcePath: new("partials/layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "layout",
						Location: ast_domain.Location{
							Line:   44,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   44,
							Column: 10,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "page-title",
						RawExpression: "'Search: ' + state.Query",
						Expression: &ast_domain.BinaryExpression{
							Left: &ast_domain.StringLiteral{
								Value: "Search: ",
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
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							Operator: "+",
							Right: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 14,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   54,
												Column: 44,
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
									Name: "Query",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 20,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Query",
											ReferenceLocation: ast_domain.Location{
												Line:   54,
												Column: 44,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   32,
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
									Column: 14,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Query",
										ReferenceLocation: ast_domain.Location{
											Line:   54,
											Column: 44,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   32,
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
								Column: 1,
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
						Location: ast_domain.Location{
							Line:   54,
							Column: 44,
						},
						NameLocation: ast_domain.Location{
							Line:   54,
							Column: 31,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   45,
							Column: 9,
						},
						TagName: "header",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("partials/layout.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   46,
									Column: 13,
								},
								TagName: "h1",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   46,
										Column: 13,
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
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   46,
											Column: 17,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_layout_ee037d9a"),
											OriginalSourcePath:   new("partials/layout.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   46,
												Column: 17,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   46,
													Column: 20,
												},
												RawExpression: "state.PageTitle",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_layout_ee037d9a.Response"),
																PackageAlias:         "partials_layout_ee037d9a",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   46,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_layout_ee037d9aData_layout_1745aa65"),
															OriginalSourcePath: new("partials/layout.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "PageTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_layout_ee037d9a",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "PageTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   46,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   29,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_1745aa65"),
															OriginalSourcePath:  new("partials/layout.pk"),
															GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
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
															PackageAlias:         "partials_layout_ee037d9a",
															CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "PageTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   46,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_1745aa65"),
														OriginalSourcePath:  new("partials/layout.pk"),
														GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "PageTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   46,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_1745aa65"),
													OriginalSourcePath:  new("partials/layout.pk"),
													GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
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
							Line:   48,
							Column: 9,
						},
						TagName: "main",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   48,
								Column: 9,
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
									Line:   55,
									Column: 9,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   55,
										Column: 21,
									},
									NameLocation: ast_domain.Location{
										Line:   55,
										Column: 14,
									},
									RawExpression: "result in state.Results",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: nil,
										ItemVariable: &ast_domain.Identifier{
											Name: "result",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
													PackageAlias:         "runtime",
													CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "result",
													ReferenceLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("result"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 11,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 21,
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
												Name: "Results",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]runtime.SearchResult[models.Doc]"),
														PackageAlias:         "piko",
														CanonicalPackagePath: "piko.sh/piko",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Results",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   33,
															Column: 2,
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
												Column: 11,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]runtime.SearchResult[models.Doc]"),
													PackageAlias:         "piko",
													CanonicalPackagePath: "piko.sh/piko",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Results",
													ReferenceLocation: ast_domain.Location{
														Line:   55,
														Column: 21,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   33,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]runtime.SearchResult[models.Doc]"),
												PackageAlias:         "piko",
												CanonicalPackagePath: "piko.sh/piko",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Results",
												ReferenceLocation: ast_domain.Location{
													Line:   55,
													Column: 21,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   33,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]runtime.SearchResult[models.Doc]"),
												PackageAlias:         "piko",
												CanonicalPackagePath: "piko.sh/piko",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Results",
												ReferenceLocation: ast_domain.Location{
													Line:   55,
													Column: 21,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   33,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]runtime.SearchResult[models.Doc]"),
											PackageAlias:         "piko",
											CanonicalPackagePath: "piko.sh/piko",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Results",
											ReferenceLocation: ast_domain.Location{
												Line:   55,
												Column: 21,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   33,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								DirKey: &ast_domain.Directive{
									Type: ast_domain.DirectiveKey,
									Location: ast_domain.Location{
										Line:   55,
										Column: 53,
									},
									NameLocation: ast_domain.Location{
										Line:   55,
										Column: 46,
									},
									RawExpression: "result.Item.URL",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "result",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
														PackageAlias:         "runtime",
														CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "result",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("result"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Item",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.Doc"),
														PackageAlias:         "models",
														CanonicalPackagePath: "models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Item",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   20,
															Column: 0,
														},
													},
													BaseCodeGenVarName:  new("result"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
													TypeExpression:       typeExprFromString("models.Doc"),
													PackageAlias:         "models",
													CanonicalPackagePath: "models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Item",
													ReferenceLocation: ast_domain.Location{
														Line:   57,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   20,
														Column: 0,
													},
												},
												BaseCodeGenVarName:  new("result"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "URL",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 13,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "URL",
													ReferenceLocation: ast_domain.Location{
														Line:   55,
														Column: 53,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("result"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/doc.go"),
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
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "URL",
												ReferenceLocation: ast_domain.Location{
													Line:   57,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("result"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/doc.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "URL",
											ReferenceLocation: ast_domain.Location{
												Line:   55,
												Column: 53,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   43,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("result"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/doc.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   55,
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
											Literal: "r.0:1:0.",
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: false,
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
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "result",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																PackageAlias:         "runtime",
																CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "result",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("result"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 8,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.Doc"),
																PackageAlias:         "models",
																CanonicalPackagePath: "models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Item",
																ReferenceLocation: ast_domain.Location{
																	Line:   55,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   20,
																	Column: 0,
																},
															},
															BaseCodeGenVarName:  new("result"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
															TypeExpression:       typeExprFromString("models.Doc"),
															PackageAlias:         "models",
															CanonicalPackagePath: "models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Item",
															ReferenceLocation: ast_domain.Location{
																Line:   57,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   20,
																Column: 0,
															},
														},
														BaseCodeGenVarName:  new("result"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "URL",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 13,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "URL",
															ReferenceLocation: ast_domain.Location{
																Line:   55,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("result"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/doc.go"),
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
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "URL",
														ReferenceLocation: ast_domain.Location{
															Line:   57,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("result"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/doc.go"),
													Stringability:       1,
												},
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   55,
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   56,
											Column: 13,
										},
										TagName: "h3",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   56,
														Column: 13,
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
													Literal: "r.0:1:0.",
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: false,
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
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "result",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																		PackageAlias:         "runtime",
																		CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "result",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("result"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 8,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.Doc"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   55,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   20,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
																	TypeExpression:       typeExprFromString("models.Doc"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   57,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   20,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName:  new("result"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "URL",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "URL",
																	ReferenceLocation: ast_domain.Location{
																		Line:   55,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   43,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("result"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/doc.go"),
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
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "URL",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("result"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/doc.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   56,
														Column: 13,
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
													Literal: ":0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   56,
												Column: 13,
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
													Line:   56,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   56,
																Column: 17,
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
															Literal: "r.0:1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("main.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "result",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																				PackageAlias:         "runtime",
																				CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "result",
																				ReferenceLocation: ast_domain.Location{
																					Line:   57,
																					Column: 16,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("result"),
																			OriginalSourcePath: new("main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Item",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 8,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("models.Doc"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   55,
																					Column: 53,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   20,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName:  new("result"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
																			TypeExpression:       typeExprFromString("models.Doc"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   57,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   20,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName:  new("result"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "URL",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "URL",
																			ReferenceLocation: ast_domain.Location{
																				Line:   55,
																				Column: 53,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   43,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("result"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("models/doc.go"),
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
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "URL",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   43,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/doc.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   56,
																Column: 17,
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
															Literal: ":0:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   56,
														Column: 17,
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
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   56,
															Column: 20,
														},
														RawExpression: "result.Item.Title",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "result",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																			PackageAlias:         "runtime",
																			CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "result",
																			ReferenceLocation: ast_domain.Location{
																				Line:   56,
																				Column: 20,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("result"),
																		OriginalSourcePath: new("main.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 8,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("models.Doc"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   56,
																				Column: 20,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   20,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName:  new("result"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
																		TypeExpression:       typeExprFromString("models.Doc"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   56,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   20,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Title",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 13,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   56,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   42,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/doc.go"),
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
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   56,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("result"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/doc.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   56,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("result"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/doc.go"),
															Stringability:       1,
														},
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   57,
											Column: 13,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   57,
														Column: 13,
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
													Literal: "r.0:1:0.",
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: false,
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
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "result",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																		PackageAlias:         "runtime",
																		CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "result",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("result"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 8,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.Doc"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   55,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   20,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
																	TypeExpression:       typeExprFromString("models.Doc"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   57,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   20,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName:  new("result"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "URL",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "URL",
																	ReferenceLocation: ast_domain.Location{
																		Line:   55,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   43,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("result"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/doc.go"),
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
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "URL",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("result"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/doc.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   57,
														Column: 13,
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
													Literal: ":1",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   57,
												Column: 13,
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
													Line:   57,
													Column: 16,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   57,
																Column: 16,
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
															Literal: "r.0:1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("main.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "result",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																				PackageAlias:         "runtime",
																				CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "result",
																				ReferenceLocation: ast_domain.Location{
																					Line:   57,
																					Column: 16,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("result"),
																			OriginalSourcePath: new("main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Item",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 8,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("models.Doc"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   55,
																					Column: 53,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   20,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName:  new("result"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
																			TypeExpression:       typeExprFromString("models.Doc"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   57,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   20,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName:  new("result"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "URL",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 13,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "URL",
																			ReferenceLocation: ast_domain.Location{
																				Line:   55,
																				Column: 53,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   43,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("result"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("models/doc.go"),
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
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "URL",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   43,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/doc.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   57,
																Column: 16,
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
															Literal: ":1:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   57,
														Column: 16,
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
															Line:   57,
															Column: 16,
														},
														Literal: "Score: ",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("main.pk"),
														},
													},
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   57,
															Column: 26,
														},
														RawExpression: "result.Score",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "result",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("runtime.SearchResult[models.Doc]"),
																		PackageAlias:         "runtime",
																		CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "result",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("result"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Score",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 8,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("float64"),
																		PackageAlias:         "runtime",
																		CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Score",
																		ReferenceLocation: ast_domain.Location{
																			Line:   57,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   751,
																			Column: 1,
																		},
																	},
																	BaseCodeGenVarName:  new("result"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
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
																	TypeExpression:       typeExprFromString("float64"),
																	PackageAlias:         "runtime",
																	CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Score",
																	ReferenceLocation: ast_domain.Location{
																		Line:   57,
																		Column: 26,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   751,
																		Column: 1,
																	},
																},
																BaseCodeGenVarName:  new("result"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("float64"),
																PackageAlias:         "runtime",
																CanonicalPackagePath: "piko.sh/piko/wdk/runtime",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Score",
																ReferenceLocation: ast_domain.Location{
																	Line:   57,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   751,
																	Column: 1,
																},
															},
															BaseCodeGenVarName:  new("result"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("../../../../../../../wdk/runtime/facade.go"),
															Stringability:       1,
														},
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
							Line:   51,
							Column: 9,
						},
						TagName: "footer",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   51,
								Column: 9,
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
									Line:   52,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   52,
										Column: 13,
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
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   52,
											Column: 16,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_layout_ee037d9a"),
											OriginalSourcePath:   new("partials/layout.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   52,
												Column: 16,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   52,
													Column: 16,
												},
												Literal: "Version: ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   52,
													Column: 28,
												},
												RawExpression: "state.Version",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_layout_ee037d9a.Response"),
																PackageAlias:         "partials_layout_ee037d9a",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 28,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_layout_ee037d9aData_layout_1745aa65"),
															OriginalSourcePath: new("partials/layout.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Version",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_layout_ee037d9a",
																CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Version",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 28,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   30,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_1745aa65"),
															OriginalSourcePath:  new("partials/layout.pk"),
															GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
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
															PackageAlias:         "partials_layout_ee037d9a",
															CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Version",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 28,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   30,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_1745aa65"),
														OriginalSourcePath:  new("partials/layout.pk"),
														GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_49_alias_generic_field_in_slotted_content/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Version",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_layout_1745aa65"),
													OriginalSourcePath:  new("partials/layout.pk"),
													GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
													Stringability:       1,
												},
											},
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
