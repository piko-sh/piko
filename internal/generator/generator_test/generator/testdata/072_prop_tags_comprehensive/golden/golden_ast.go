package testcase_072_prop_tags_comprehensive

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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "prop-tags-demo",
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
								TextContent: "Prop Tags Comprehensive Test",
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
							Line:   25,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								Value: "test-basic",
								Location: ast_domain.Location{
									Line:   25,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   26,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   26,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   26,
											Column: 11,
										},
										TextContent: "1. Basic Prop Tag (prop:\"name\")",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   26,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_basic_ae555b0a"),
									OriginalSourcePath:   new("partials/basic.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0",
										PartialAlias:        "basic",
										PartialPackageName:  "partials_basic_ae555b0a",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   27,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"card-description": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 82,
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
														Name: "BasicDescription",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BasicDescription",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 82,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   99,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "BasicDescription",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 82,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   99,
																		Column: 2,
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
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "BasicDescription",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 82,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   99,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BasicDescription",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 82,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   99,
																	Column: 2,
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
													Line:   27,
													Column: 82,
												},
												GoFieldName: "Description",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "BasicDescription",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 82,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   99,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "BasicDescription",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 82,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   99,
																Column: 2,
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
											"card-title": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
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
														Name: "BasicTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BasicTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   98,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "BasicTitle",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   98,
																		Column: 2,
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
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "BasicTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   98,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BasicTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   98,
																	Column: 2,
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
													Line:   27,
													Column: 45,
												},
												GoFieldName: "Title",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "BasicTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   98,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "BasicTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   98,
																Column: 2,
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
										"card-description": "pages_main_594861c5",
										"card-title":       "pages_main_594861c5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
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
										OriginalSourcePath: new("partials/basic.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "basic-card",
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
										Name:          "card-description",
										RawExpression: "state.BasicDescription",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 82,
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
												Name: "BasicDescription",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "BasicDescription",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 82,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   99,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "BasicDescription",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 82,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   99,
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
											Line:   27,
											Column: 82,
										},
										NameLocation: ast_domain.Location{
											Line:   27,
											Column: 63,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "BasicDescription",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 82,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   99,
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
										Name:          "card-title",
										RawExpression: "state.BasicTitle",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
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
												Name: "BasicTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "BasicTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   98,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "BasicTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   98,
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
											Line:   27,
											Column: 45,
										},
										NameLocation: ast_domain.Location{
											Line:   27,
											Column: 32,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "BasicTitle",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   98,
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
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_basic_ae555b0a"),
											OriginalSourcePath:   new("partials/basic.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 9,
											},
											RawExpression: "state.CardTitle",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_basic_ae555b0a.Response"),
															PackageAlias:         "partials_basic_ae555b0a",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
														OriginalSourcePath: new("partials/basic.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "CardTitle",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_basic_ae555b0a",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CardTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
														OriginalSourcePath:  new("partials/basic.pk"),
														GeneratedSourcePath: new("dist/partials/partials_basic_ae555b0a/generated.go"),
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
														PackageAlias:         "partials_basic_ae555b0a",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CardTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
													OriginalSourcePath:  new("partials/basic.pk"),
													GeneratedSourcePath: new("dist/partials/partials_basic_ae555b0a/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_basic_ae555b0a",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CardTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
												OriginalSourcePath:  new("partials/basic.pk"),
												GeneratedSourcePath: new("dist/partials/partials_basic_ae555b0a/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:0",
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
												OriginalSourcePath: new("partials/basic.pk"),
												Stringability:      1,
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
											OriginalPackageAlias: new("partials_basic_ae555b0a"),
											OriginalSourcePath:   new("partials/basic.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 16,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 8,
											},
											RawExpression: "state.CardDescription",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_basic_ae555b0a.Response"),
															PackageAlias:         "partials_basic_ae555b0a",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
														OriginalSourcePath: new("partials/basic.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "CardDescription",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_basic_ae555b0a",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CardDescription",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
														OriginalSourcePath:  new("partials/basic.pk"),
														GeneratedSourcePath: new("dist/partials/partials_basic_ae555b0a/generated.go"),
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
														PackageAlias:         "partials_basic_ae555b0a",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CardDescription",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
													OriginalSourcePath:  new("partials/basic.pk"),
													GeneratedSourcePath: new("dist/partials/partials_basic_ae555b0a/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_basic_ae555b0a",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_basic_ae555b0a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CardDescription",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_basic_ae555b0aData_basic_card_description_state_basicdescription_card_title_state_basictitle_1e89f3b0"),
												OriginalSourcePath:  new("partials/basic.pk"),
												GeneratedSourcePath: new("dist/partials/partials_basic_ae555b0a/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:1",
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
												OriginalSourcePath: new("partials/basic.pk"),
												Stringability:      1,
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
							Line:   30,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   30,
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
								Value: "test-required",
								Location: ast_domain.Location{
									Line:   30,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   30,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   31,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   31,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   31,
											Column: 11,
										},
										TextContent: "2. Required Props (validate:\"required\")",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   31,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_required_e49b403b"),
									OriginalSourcePath:   new("partials/required.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "required_id_state_requiredid_name_state_requiredname_723c96ce",
										PartialAlias:        "required",
										PartialPackageName:  "partials_required_e49b403b",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   32,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"id": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 67,
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
														Name: "RequiredID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "RequiredID",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 67,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   101,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "RequiredID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   32,
																		Column: 67,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   101,
																		Column: 2,
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
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "RequiredID",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 67,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   101,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "RequiredID",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 67,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   101,
																	Column: 2,
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
													Line:   32,
													Column: 67,
												},
												GoFieldName: "ID",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "RequiredID",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 67,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   101,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "RequiredID",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 67,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   101,
																Column: 2,
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
											"name": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 42,
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
														Name: "RequiredName",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "RequiredName",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 42,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   100,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "RequiredName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   32,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   100,
																		Column: 2,
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
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "RequiredName",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 42,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   100,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "RequiredName",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 42,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   100,
																	Column: 2,
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
													Line:   32,
													Column: 42,
												},
												GoFieldName: "Name",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "RequiredName",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   100,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "RequiredName",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 42,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   100,
																Column: 2,
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
										"id":   "pages_main_594861c5",
										"name": "pages_main_594861c5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:1",
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
										OriginalSourcePath: new("partials/required.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "required-card",
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
										RawExpression: "state.RequiredID",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 67,
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
												Name: "RequiredID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "RequiredID",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 67,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   101,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "RequiredID",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 67,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   101,
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
											Line:   32,
											Column: 67,
										},
										NameLocation: ast_domain.Location{
											Line:   32,
											Column: 62,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "RequiredID",
												ReferenceLocation: ast_domain.Location{
													Line:   32,
													Column: 67,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   101,
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
										Name:          "name",
										RawExpression: "state.RequiredName",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 42,
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
												Name: "RequiredName",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "RequiredName",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   100,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "RequiredName",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 42,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   100,
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
											Line:   32,
											Column: 42,
										},
										NameLocation: ast_domain.Location{
											Line:   32,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "RequiredName",
												ReferenceLocation: ast_domain.Location{
													Line:   32,
													Column: 42,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   100,
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
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_required_e49b403b"),
											OriginalSourcePath:   new("partials/required.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 9,
											},
											RawExpression: "state.Name",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_required_e49b403b.Response"),
															PackageAlias:         "partials_required_e49b403b",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
														OriginalSourcePath: new("partials/required.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_required_e49b403b",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
														OriginalSourcePath:  new("partials/required.pk"),
														GeneratedSourcePath: new("dist/partials/partials_required_e49b403b/generated.go"),
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
														PackageAlias:         "partials_required_e49b403b",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
													OriginalSourcePath:  new("partials/required.pk"),
													GeneratedSourcePath: new("dist/partials/partials_required_e49b403b/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_required_e49b403b",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
												OriginalSourcePath:  new("partials/required.pk"),
												GeneratedSourcePath: new("dist/partials/partials_required_e49b403b/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:1:0",
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
												OriginalSourcePath: new("partials/required.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_required_e49b403b"),
											OriginalSourcePath:   new("partials/required.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 11,
											},
											RawExpression: "state.ID",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_required_e49b403b.Response"),
															PackageAlias:         "partials_required_e49b403b",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
														OriginalSourcePath: new("partials/required.pk"),
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
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_required_e49b403b",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
														OriginalSourcePath:  new("partials/required.pk"),
														GeneratedSourcePath: new("dist/partials/partials_required_e49b403b/generated.go"),
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
														PackageAlias:         "partials_required_e49b403b",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
													OriginalSourcePath:  new("partials/required.pk"),
													GeneratedSourcePath: new("dist/partials/partials_required_e49b403b/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_required_e49b403b",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_required_e49b403b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_required_e49b403bData_required_id_state_requiredid_name_state_requiredname_723c96ce"),
												OriginalSourcePath:  new("partials/required.pk"),
												GeneratedSourcePath: new("dist/partials/partials_required_e49b403b/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:1:1",
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
												OriginalSourcePath: new("partials/required.pk"),
												Stringability:      1,
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
							Line:   35,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   35,
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
								Value: "test-defaults",
								Location: ast_domain.Location{
									Line:   35,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   35,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   36,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   36,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   36,
											Column: 11,
										},
										TextContent: "3. Default Values (default:\"value\")",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   36,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_default_value_b97ed726"),
									OriginalSourcePath:   new("partials/default-value.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "defaultval_label_click_me_size_medium_theme_dark_0c5f3c87",
										PartialAlias:        "defaultval",
										PartialPackageName:  "partials_default_value_b97ed726",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   37,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"label": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "Click me",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Label",
											},
											"size": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "medium",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Size",
											},
											"theme": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "dark",
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
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   37,
													Column: 44,
												},
												GoFieldName: "Theme",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"class": "partials_default_value_b97ed726",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:1",
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
										OriginalSourcePath: new("partials/default-value.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "theme",
										Value: "dark",
										Location: ast_domain.Location{
											Line:   37,
											Column: 44,
										},
										NameLocation: ast_domain.Location{
											Line:   37,
											Column: 37,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "class",
										RawExpression: "'themed-card theme-' + state.Theme",
										Expression: &ast_domain.BinaryExpression{
											Left: &ast_domain.StringLiteral{
												Value: "themed-card theme-",
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
													OriginalSourcePath: new("partials/default-value.pk"),
													Stringability:      1,
												},
											},
											Operator: "+",
											Right: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 24,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_default_value_b97ed726.Response"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
														OriginalSourcePath: new("partials/default-value.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Theme",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Theme",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
														OriginalSourcePath:  new("partials/default-value.pk"),
														GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
														Stringability:       1,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 24,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_default_value_b97ed726",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Theme",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
													OriginalSourcePath:  new("partials/default-value.pk"),
													GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
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
												OriginalSourcePath: new("partials/default-value.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/default-value.pk"),
											Stringability:      1,
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
											OriginalPackageAlias: new("partials_default_value_b97ed726"),
											OriginalSourcePath:   new("partials/default-value.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.Size",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_default_value_b97ed726.Response"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
														OriginalSourcePath: new("partials/default-value.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Size",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Size",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
														OriginalSourcePath:  new("partials/default-value.pk"),
														GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
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
														PackageAlias:         "partials_default_value_b97ed726",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Size",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
													OriginalSourcePath:  new("partials/default-value.pk"),
													GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_default_value_b97ed726",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Size",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
												OriginalSourcePath:  new("partials/default-value.pk"),
												GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1:0",
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
												OriginalSourcePath: new("partials/default-value.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_default_value_b97ed726"),
											OriginalSourcePath:   new("partials/default-value.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 11,
											},
											RawExpression: "state.Label",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_default_value_b97ed726.Response"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
														OriginalSourcePath: new("partials/default-value.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Label",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
														OriginalSourcePath:  new("partials/default-value.pk"),
														GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
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
														PackageAlias:         "partials_default_value_b97ed726",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
													OriginalSourcePath:  new("partials/default-value.pk"),
													GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_default_value_b97ed726",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_click_me_size_medium_theme_dark_0c5f3c87"),
												OriginalSourcePath:  new("partials/default-value.pk"),
												GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1:1",
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
												OriginalSourcePath: new("partials/default-value.pk"),
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_default_value_b97ed726"),
									OriginalSourcePath:   new("partials/default-value.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "defaultval_label_custom_label_size_large_theme_custom_a00c63c5",
										PartialAlias:        "defaultval",
										PartialPackageName:  "partials_default_value_b97ed726",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   38,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"label": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "Custom Label",
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
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   38,
													Column: 72,
												},
												GoFieldName: "Label",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"size": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "large",
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
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   38,
													Column: 58,
												},
												GoFieldName: "Size",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"theme": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "custom",
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
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   38,
													Column: 44,
												},
												GoFieldName: "Theme",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"class": "partials_default_value_b97ed726",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:2",
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
										OriginalSourcePath: new("partials/default-value.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "label",
										Value: "Custom Label",
										Location: ast_domain.Location{
											Line:   38,
											Column: 72,
										},
										NameLocation: ast_domain.Location{
											Line:   38,
											Column: 65,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "size",
										Value: "large",
										Location: ast_domain.Location{
											Line:   38,
											Column: 58,
										},
										NameLocation: ast_domain.Location{
											Line:   38,
											Column: 52,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "theme",
										Value: "custom",
										Location: ast_domain.Location{
											Line:   38,
											Column: 44,
										},
										NameLocation: ast_domain.Location{
											Line:   38,
											Column: 37,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "class",
										RawExpression: "'themed-card theme-' + state.Theme",
										Expression: &ast_domain.BinaryExpression{
											Left: &ast_domain.StringLiteral{
												Value: "themed-card theme-",
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
													OriginalSourcePath: new("partials/default-value.pk"),
													Stringability:      1,
												},
											},
											Operator: "+",
											Right: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 24,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_default_value_b97ed726.Response"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
														OriginalSourcePath: new("partials/default-value.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Theme",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Theme",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
														OriginalSourcePath:  new("partials/default-value.pk"),
														GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
														Stringability:       1,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 24,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_default_value_b97ed726",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Theme",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
													OriginalSourcePath:  new("partials/default-value.pk"),
													GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
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
												OriginalSourcePath: new("partials/default-value.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/default-value.pk"),
											Stringability:      1,
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
											OriginalPackageAlias: new("partials_default_value_b97ed726"),
											OriginalSourcePath:   new("partials/default-value.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.Size",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_default_value_b97ed726.Response"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
														OriginalSourcePath: new("partials/default-value.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Size",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Size",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
														OriginalSourcePath:  new("partials/default-value.pk"),
														GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
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
														PackageAlias:         "partials_default_value_b97ed726",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Size",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
													OriginalSourcePath:  new("partials/default-value.pk"),
													GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_default_value_b97ed726",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Size",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
												OriginalSourcePath:  new("partials/default-value.pk"),
												GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:2:0",
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
												OriginalSourcePath: new("partials/default-value.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_default_value_b97ed726"),
											OriginalSourcePath:   new("partials/default-value.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 11,
											},
											RawExpression: "state.Label",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_default_value_b97ed726.Response"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
														OriginalSourcePath: new("partials/default-value.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Label",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_default_value_b97ed726",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
														OriginalSourcePath:  new("partials/default-value.pk"),
														GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
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
														PackageAlias:         "partials_default_value_b97ed726",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
													OriginalSourcePath:  new("partials/default-value.pk"),
													GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_default_value_b97ed726",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_default_value_b97ed726",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_default_value_b97ed726Data_defaultval_label_custom_label_size_large_theme_custom_a00c63c5"),
												OriginalSourcePath:  new("partials/default-value.pk"),
												GeneratedSourcePath: new("dist/partials/partials_default_value_b97ed726/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:2:1",
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
												OriginalSourcePath: new("partials/default-value.pk"),
												Stringability:      1,
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
							Line:   41,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   41,
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
								Value: "test-factory",
								Location: ast_domain.Location{
									Line:   41,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   42,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   42,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   42,
											Column: 11,
										},
										TextContent: "4. Factory Function Default (factory:\"FuncName\")",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   42,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_factory_default_937e675c"),
									OriginalSourcePath:   new("partials/factory-default.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d",
										PartialAlias:        "factorydefault",
										PartialPackageName:  "partials_factory_default_937e675c",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   43,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"options": ast_domain.PropValue{
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "partials_factory_default_937e675c",
															RelativeLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       nil,
																	PackageAlias:         "partials_factory_default_937e675c",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
																},
																BaseCodeGenVarName: new("partials_factory_default_937e675c"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "GetDefaultOptions",
															RelativeLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "partials_factory_default_937e675c",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetDefaultOptions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   43,
																		Column: 7,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_factory_default_937e675c"),
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
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "partials_factory_default_937e675c",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetDefaultOptions",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 7,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_factory_default_937e675c"),
														},
													},
													Args: []ast_domain.Expression{},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															BaseCodeGenVarName: new("partials_factory_default_937e675c"),
														},
														BaseCodeGenVarName: new("partials_factory_default_937e675c"),
													},
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Options",
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"class": "partials_factory_default_937e675c",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:1",
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
										OriginalSourcePath: new("partials/factory-default.pk"),
										Stringability:      1,
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "class",
										RawExpression: "'avatar avatar-' + state.Options.Shape",
										Expression: &ast_domain.BinaryExpression{
											Left: &ast_domain.StringLiteral{
												Value: "avatar avatar-",
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
													OriginalSourcePath: new("partials/factory-default.pk"),
													Stringability:      1,
												},
											},
											Operator: "+",
											Right: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_factory_default_937e675c.Response"),
																PackageAlias:         "partials_factory_default_937e675c",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
															OriginalSourcePath: new("partials/factory-default.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Options",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 26,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Options",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
															OriginalSourcePath:  new("partials/factory-default.pk"),
															GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
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
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Options",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
														OriginalSourcePath:  new("partials/factory-default.pk"),
														GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Shape",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 34,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Shape",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
														OriginalSourcePath:  new("partials/factory-default.pk"),
														GeneratedSourcePath: new("pkg/models/models.go"),
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
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Shape",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
													OriginalSourcePath:  new("partials/factory-default.pk"),
													GeneratedSourcePath: new("pkg/models/models.go"),
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
												OriginalSourcePath: new("partials/factory-default.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/factory-default.pk"),
											Stringability:      1,
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
											OriginalPackageAlias: new("partials_factory_default_937e675c"),
											OriginalSourcePath:   new("partials/factory-default.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.SizeDisplay",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_factory_default_937e675c.Response"),
															PackageAlias:         "partials_factory_default_937e675c",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
														OriginalSourcePath: new("partials/factory-default.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "SizeDisplay",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_factory_default_937e675c",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "SizeDisplay",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   44,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
														OriginalSourcePath:  new("partials/factory-default.pk"),
														GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
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
														PackageAlias:         "partials_factory_default_937e675c",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SizeDisplay",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   44,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
													OriginalSourcePath:  new("partials/factory-default.pk"),
													GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_factory_default_937e675c",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "SizeDisplay",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   44,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_partials_factory_default_937e675c_getdefaultoptions_36c30c3d"),
												OriginalSourcePath:  new("partials/factory-default.pk"),
												GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:1:0",
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
												OriginalSourcePath: new("partials/factory-default.pk"),
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_factory_default_937e675c"),
									OriginalSourcePath:   new("partials/factory-default.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "factorydefault_options_state_customoptions_20f3f550",
										PartialAlias:        "factorydefault",
										PartialPackageName:  "partials_factory_default_937e675c",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   44,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"options": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 51,
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
														Name: "CustomOptions",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CustomOptions",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 51,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   102,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.AvatarOptions"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "CustomOptions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   44,
																		Column: 51,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   102,
																		Column: 2,
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
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CustomOptions",
															ReferenceLocation: ast_domain.Location{
																Line:   44,
																Column: 51,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   102,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CustomOptions",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 51,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   102,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Location: ast_domain.Location{
													Line:   44,
													Column: 51,
												},
												GoFieldName: "Options",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.AvatarOptions"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CustomOptions",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 51,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   102,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CustomOptions",
															ReferenceLocation: ast_domain.Location{
																Line:   44,
																Column: 51,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   102,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"class":   "partials_factory_default_937e675c",
										"options": "pages_main_594861c5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:2",
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
										OriginalSourcePath: new("partials/factory-default.pk"),
										Stringability:      1,
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "class",
										RawExpression: "'avatar avatar-' + state.Options.Shape",
										Expression: &ast_domain.BinaryExpression{
											Left: &ast_domain.StringLiteral{
												Value: "avatar avatar-",
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
													OriginalSourcePath: new("partials/factory-default.pk"),
													Stringability:      1,
												},
											},
											Operator: "+",
											Right: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_factory_default_937e675c.Response"),
																PackageAlias:         "partials_factory_default_937e675c",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
															OriginalSourcePath: new("partials/factory-default.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Options",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 26,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Options",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
															OriginalSourcePath:  new("partials/factory-default.pk"),
															GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
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
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Options",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
														OriginalSourcePath:  new("partials/factory-default.pk"),
														GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Shape",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 34,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Shape",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
														OriginalSourcePath:  new("partials/factory-default.pk"),
														GeneratedSourcePath: new("pkg/models/models.go"),
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
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Shape",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
													OriginalSourcePath:  new("partials/factory-default.pk"),
													GeneratedSourcePath: new("pkg/models/models.go"),
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
												OriginalSourcePath: new("partials/factory-default.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/factory-default.pk"),
											Stringability:      1,
										},
									},
									ast_domain.DynamicAttribute{
										Name:          "options",
										RawExpression: "state.CustomOptions",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 51,
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
												Name: "CustomOptions",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.AvatarOptions"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CustomOptions",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 51,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   102,
															Column: 2,
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
													TypeExpression:       typeExprFromString("models.AvatarOptions"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CustomOptions",
													ReferenceLocation: ast_domain.Location{
														Line:   44,
														Column: 51,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   102,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Location: ast_domain.Location{
											Line:   44,
											Column: 51,
										},
										NameLocation: ast_domain.Location{
											Line:   44,
											Column: 41,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("models.AvatarOptions"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CustomOptions",
												ReferenceLocation: ast_domain.Location{
													Line:   44,
													Column: 51,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   102,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
											OriginalPackageAlias: new("partials_factory_default_937e675c"),
											OriginalSourcePath:   new("partials/factory-default.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.SizeDisplay",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_factory_default_937e675c.Response"),
															PackageAlias:         "partials_factory_default_937e675c",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
														OriginalSourcePath: new("partials/factory-default.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "SizeDisplay",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_factory_default_937e675c",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "SizeDisplay",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   44,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
														OriginalSourcePath:  new("partials/factory-default.pk"),
														GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
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
														PackageAlias:         "partials_factory_default_937e675c",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SizeDisplay",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   44,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
													OriginalSourcePath:  new("partials/factory-default.pk"),
													GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_factory_default_937e675c",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_factory_default_937e675c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "SizeDisplay",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   44,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_factory_default_937e675cData_factorydefault_options_state_customoptions_20f3f550"),
												OriginalSourcePath:  new("partials/factory-default.pk"),
												GeneratedSourcePath: new("dist/partials/partials_factory_default_937e675c/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:2:0",
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
												OriginalSourcePath: new("partials/factory-default.pk"),
												Stringability:      1,
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
							Line:   47,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   47,
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
								Value: "test-coercion",
								Location: ast_domain.Location{
									Line:   47,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   47,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   48,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   48,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   48,
											Column: 11,
										},
										TextContent: "5. Type Coercion (coerce:\"true\")",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   48,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_coercion_97bb15e1"),
									OriginalSourcePath:   new("partials/coercion.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "coercion_count_42_is_active_true_price_19_99_7ee9fd01",
										PartialAlias:        "coercion",
										PartialPackageName:  "partials_coercion_97bb15e1",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   49,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"count": ast_domain.PropValue{
												Expression: &ast_domain.IntegerLiteral{
													Value: 42,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   49,
													Column: 59,
												},
												GoFieldName: "Count",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"is-active": ast_domain.PropValue{
												Expression: &ast_domain.BooleanLiteral{
													Value: true,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   49,
													Column: 46,
												},
												GoFieldName: "IsActive",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"price": ast_domain.PropValue{
												Expression: &ast_domain.FloatLiteral{
													Value: 19.99,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("float64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   49,
													Column: 70,
												},
												GoFieldName: "Price",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("float64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:1",
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
										OriginalSourcePath: new("partials/coercion.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-display",
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
										Name:  "count",
										Value: "42",
										Location: ast_domain.Location{
											Line:   49,
											Column: 59,
										},
										NameLocation: ast_domain.Location{
											Line:   49,
											Column: 52,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "is-active",
										Value: "true",
										Location: ast_domain.Location{
											Line:   49,
											Column: 46,
										},
										NameLocation: ast_domain.Location{
											Line:   49,
											Column: 35,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "price",
										Value: "19.99",
										Location: ast_domain.Location{
											Line:   49,
											Column: 70,
										},
										NameLocation: ast_domain.Location{
											Line:   49,
											Column: 63,
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
											OriginalPackageAlias: new("partials_coercion_97bb15e1"),
											OriginalSourcePath:   new("partials/coercion.pk"),
											IsStructurallyStatic: true,
										},
										DirIf: &ast_domain.Directive{
											Type: ast_domain.DirectiveIf,
											Location: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.IsActive",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_coercion_97bb15e1.Response"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
														OriginalSourcePath: new("partials/coercion.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "IsActive",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IsActive",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
														OriginalSourcePath:  new("partials/coercion.pk"),
														GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
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
														PackageAlias:         "partials_coercion_97bb15e1",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IsActive",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
													OriginalSourcePath:  new("partials/coercion.pk"),
													GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_coercion_97bb15e1",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IsActive",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
												OriginalSourcePath:  new("partials/coercion.pk"),
												GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:1:0",
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
												OriginalSourcePath: new("partials/coercion.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   23,
													Column: 33,
												},
												TextContent: "Active",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_coercion_97bb15e1"),
													OriginalSourcePath:   new("partials/coercion.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:5:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 33,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/coercion.pk"),
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_coercion_97bb15e1"),
											OriginalSourcePath:   new("partials/coercion.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 11,
											},
											RawExpression: "state.Count",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_coercion_97bb15e1.Response"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
														OriginalSourcePath: new("partials/coercion.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Count",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Count",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
														OriginalSourcePath:  new("partials/coercion.pk"),
														GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
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
														PackageAlias:         "partials_coercion_97bb15e1",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Count",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
													OriginalSourcePath:  new("partials/coercion.pk"),
													GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_coercion_97bb15e1",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Count",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
												OriginalSourcePath:  new("partials/coercion.pk"),
												GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:1:1",
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
												OriginalSourcePath: new("partials/coercion.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_coercion_97bb15e1"),
											OriginalSourcePath:   new("partials/coercion.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   25,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   25,
												Column: 11,
											},
											RawExpression: "state.Price",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_coercion_97bb15e1.Response"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
														OriginalSourcePath: new("partials/coercion.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Price",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Price",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   41,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
														OriginalSourcePath:  new("partials/coercion.pk"),
														GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
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
														PackageAlias:         "partials_coercion_97bb15e1",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Price",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   41,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
													OriginalSourcePath:  new("partials/coercion.pk"),
													GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("float64"),
													PackageAlias:         "partials_coercion_97bb15e1",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Price",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   41,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_42_is_active_true_price_19_99_7ee9fd01"),
												OriginalSourcePath:  new("partials/coercion.pk"),
												GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:1:2",
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
												OriginalSourcePath: new("partials/coercion.pk"),
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_coercion_97bb15e1"),
									OriginalSourcePath:   new("partials/coercion.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07",
										PartialAlias:        "coercion",
										PartialPackageName:  "partials_coercion_97bb15e1",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   50,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"count": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 74,
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
														Name: "DynamicInt",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DynamicInt",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 74,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   104,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DynamicInt",
																	ReferenceLocation: ast_domain.Location{
																		Line:   50,
																		Column: 74,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   104,
																		Column: 2,
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
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DynamicInt",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 74,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   104,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DynamicInt",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 74,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   104,
																	Column: 2,
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
													Line:   50,
													Column: 74,
												},
												GoFieldName: "Count",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DynamicInt",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 74,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   104,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DynamicInt",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 74,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   104,
																Column: 2,
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
											"is-active": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 47,
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
														Name: "DynamicBool",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DynamicBool",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   103,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("bool"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DynamicBool",
																	ReferenceLocation: ast_domain.Location{
																		Line:   50,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   103,
																		Column: 2,
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
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DynamicBool",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   103,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DynamicBool",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   103,
																	Column: 2,
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
													Line:   50,
													Column: 47,
												},
												GoFieldName: "IsActive",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DynamicBool",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   103,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DynamicBool",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   103,
																Column: 2,
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
											"price": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 100,
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
														Name: "DynamicFloat",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("float64"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DynamicFloat",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 100,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   105,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("float64"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DynamicFloat",
																	ReferenceLocation: ast_domain.Location{
																		Line:   50,
																		Column: 100,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   105,
																		Column: 2,
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
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DynamicFloat",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 100,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   105,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("float64"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DynamicFloat",
																ReferenceLocation: ast_domain.Location{
																	Line:   50,
																	Column: 100,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   105,
																	Column: 2,
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
													Line:   50,
													Column: 100,
												},
												GoFieldName: "Price",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("float64"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DynamicFloat",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 100,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   105,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DynamicFloat",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 100,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   105,
																Column: 2,
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
										"count":     "pages_main_594861c5",
										"is-active": "pages_main_594861c5",
										"price":     "pages_main_594861c5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:2",
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
										OriginalSourcePath: new("partials/coercion.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "coerced-display",
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
										Name:          "count",
										RawExpression: "state.DynamicInt",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 74,
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
												Name: "DynamicInt",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DynamicInt",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 74,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   104,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DynamicInt",
													ReferenceLocation: ast_domain.Location{
														Line:   50,
														Column: 74,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   104,
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
											Line:   50,
											Column: 74,
										},
										NameLocation: ast_domain.Location{
											Line:   50,
											Column: 66,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DynamicInt",
												ReferenceLocation: ast_domain.Location{
													Line:   50,
													Column: 74,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   104,
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
										Name:          "is-active",
										RawExpression: "state.DynamicBool",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 47,
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
												Name: "DynamicBool",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DynamicBool",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   103,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DynamicBool",
													ReferenceLocation: ast_domain.Location{
														Line:   50,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   103,
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
											Line:   50,
											Column: 47,
										},
										NameLocation: ast_domain.Location{
											Line:   50,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DynamicBool",
												ReferenceLocation: ast_domain.Location{
													Line:   50,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   103,
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
										Name:          "price",
										RawExpression: "state.DynamicFloat",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 100,
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
												Name: "DynamicFloat",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("float64"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DynamicFloat",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 100,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   105,
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
													TypeExpression:       typeExprFromString("float64"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DynamicFloat",
													ReferenceLocation: ast_domain.Location{
														Line:   50,
														Column: 100,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   105,
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
											Line:   50,
											Column: 100,
										},
										NameLocation: ast_domain.Location{
											Line:   50,
											Column: 92,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("float64"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DynamicFloat",
												ReferenceLocation: ast_domain.Location{
													Line:   50,
													Column: 100,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   105,
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_coercion_97bb15e1"),
											OriginalSourcePath:   new("partials/coercion.pk"),
											IsStructurallyStatic: true,
										},
										DirIf: &ast_domain.Directive{
											Type: ast_domain.DirectiveIf,
											Location: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.IsActive",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_coercion_97bb15e1.Response"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
														OriginalSourcePath: new("partials/coercion.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "IsActive",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IsActive",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
														OriginalSourcePath:  new("partials/coercion.pk"),
														GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
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
														PackageAlias:         "partials_coercion_97bb15e1",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IsActive",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
													OriginalSourcePath:  new("partials/coercion.pk"),
													GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_coercion_97bb15e1",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IsActive",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
												OriginalSourcePath:  new("partials/coercion.pk"),
												GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:2:0",
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
												OriginalSourcePath: new("partials/coercion.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   23,
													Column: 33,
												},
												TextContent: "Active",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_coercion_97bb15e1"),
													OriginalSourcePath:   new("partials/coercion.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:5:2:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 33,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/coercion.pk"),
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_coercion_97bb15e1"),
											OriginalSourcePath:   new("partials/coercion.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 11,
											},
											RawExpression: "state.Count",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_coercion_97bb15e1.Response"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
														OriginalSourcePath: new("partials/coercion.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Count",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Count",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
														OriginalSourcePath:  new("partials/coercion.pk"),
														GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
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
														PackageAlias:         "partials_coercion_97bb15e1",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Count",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
													OriginalSourcePath:  new("partials/coercion.pk"),
													GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_coercion_97bb15e1",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Count",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   40,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
												OriginalSourcePath:  new("partials/coercion.pk"),
												GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:2:1",
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
												OriginalSourcePath: new("partials/coercion.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_coercion_97bb15e1"),
											OriginalSourcePath:   new("partials/coercion.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   25,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   25,
												Column: 11,
											},
											RawExpression: "state.Price",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_coercion_97bb15e1.Response"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
														OriginalSourcePath: new("partials/coercion.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Price",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "partials_coercion_97bb15e1",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Price",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   41,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
														OriginalSourcePath:  new("partials/coercion.pk"),
														GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
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
														PackageAlias:         "partials_coercion_97bb15e1",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Price",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   41,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
													OriginalSourcePath:  new("partials/coercion.pk"),
													GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("float64"),
													PackageAlias:         "partials_coercion_97bb15e1",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_coercion_97bb15e1",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Price",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   41,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_coercion_97bb15e1Data_coercion_count_state_dynamicint_is_active_state_dynamicbool_price_state_dynamicfloat_61565a07"),
												OriginalSourcePath:  new("partials/coercion.pk"),
												GeneratedSourcePath: new("dist/partials/partials_coercion_97bb15e1/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:2:2",
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
												OriginalSourcePath: new("partials/coercion.pk"),
												Stringability:      1,
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
							Line:   53,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
							RelativeLocation: ast_domain.Location{
								Line:   53,
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
								Value: "test-optional",
								Location: ast_domain.Location{
									Line:   53,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   53,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   54,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
									RelativeLocation: ast_domain.Location{
										Line:   54,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   54,
											Column: 11,
										},
										TextContent: "6. Optional Pointer Props (*Type)",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:6:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   54,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_optional_68cf5cc2"),
									OriginalSourcePath:   new("partials/optional.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "optional_58fda695",
										PartialAlias:        "optional",
										PartialPackageName:  "partials_optional_68cf5cc2",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   55,
											Column: 7,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:1",
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
										OriginalSourcePath: new("partials/optional.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "user-card",
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   23,
											Column: 5,
										},
										TagName: "p-if",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_optional_68cf5cc2"),
											OriginalSourcePath:   new("partials/optional.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:6:1:0",
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
												OriginalSourcePath: new("partials/optional.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "condition",
												Value: "state.HasProfile",
												Location: ast_domain.Location{
													Line:   23,
													Column: 22,
												},
												NameLocation: ast_domain.Location{
													Line:   23,
													Column: 11,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   24,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_optional_68cf5cc2"),
													OriginalSourcePath:   new("partials/optional.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   24,
														Column: 21,
													},
													NameLocation: ast_domain.Location{
														Line:   24,
														Column: 13,
													},
													RawExpression: "state.ProfileName",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_optional_68cf5cc2.Response"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_optional_68cf5cc2Data_optional_58fda695"),
																OriginalSourcePath: new("partials/optional.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ProfileName",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ProfileName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_58fda695"),
																OriginalSourcePath:  new("partials/optional.pk"),
																GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
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
																PackageAlias:         "partials_optional_68cf5cc2",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ProfileName",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_58fda695"),
															OriginalSourcePath:  new("partials/optional.pk"),
															GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_optional_68cf5cc2",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ProfileName",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_58fda695"),
														OriginalSourcePath:  new("partials/optional.pk"),
														GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:6:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/optional.pk"),
														Stringability:      1,
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   25,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_optional_68cf5cc2"),
													OriginalSourcePath:   new("partials/optional.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   25,
														Column: 21,
													},
													NameLocation: ast_domain.Location{
														Line:   25,
														Column: 13,
													},
													RawExpression: "state.ProfileEmail",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_optional_68cf5cc2.Response"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_optional_68cf5cc2Data_optional_58fda695"),
																OriginalSourcePath: new("partials/optional.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ProfileEmail",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ProfileEmail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   47,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_58fda695"),
																OriginalSourcePath:  new("partials/optional.pk"),
																GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
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
																PackageAlias:         "partials_optional_68cf5cc2",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ProfileEmail",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   47,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_58fda695"),
															OriginalSourcePath:  new("partials/optional.pk"),
															GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_optional_68cf5cc2",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ProfileEmail",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   47,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_58fda695"),
														OriginalSourcePath:  new("partials/optional.pk"),
														GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:6:1:0:1",
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/optional.pk"),
														Stringability:      1,
													},
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
										TagName: "p-else",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_optional_68cf5cc2"),
											OriginalSourcePath:   new("partials/optional.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:6:1:1",
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
												OriginalSourcePath: new("partials/optional.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   28,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_optional_68cf5cc2"),
													OriginalSourcePath:   new("partials/optional.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:6:1:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   28,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/optional.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   28,
															Column: 13,
														},
														TextContent: "No profile provided",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_optional_68cf5cc2"),
															OriginalSourcePath:   new("partials/optional.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:6:1:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/optional.pk"),
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
									Line:   22,
									Column: 3,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_optional_68cf5cc2"),
									OriginalSourcePath:   new("partials/optional.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "optional_profile_state_userprofile_b0f1fda6",
										PartialAlias:        "optional",
										PartialPackageName:  "partials_optional_68cf5cc2",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   56,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"profile": ast_domain.PropValue{
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
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   56,
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
															Name: "UserProfile",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.UserProfile"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "UserProfile",
																	ReferenceLocation: ast_domain.Location{
																		Line:   56,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   106,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.UserProfile"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "UserProfile",
																		ReferenceLocation: ast_domain.Location{
																			Line:   56,
																			Column: 45,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   106,
																			Column: 2,
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
																TypeExpression:       typeExprFromString("models.UserProfile"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UserProfile",
																ReferenceLocation: ast_domain.Location{
																	Line:   56,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   106,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.UserProfile"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "UserProfile",
																	ReferenceLocation: ast_domain.Location{
																		Line:   56,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   106,
																		Column: 2,
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
													Line:   56,
													Column: 45,
												},
												GoFieldName: "Profile",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*models.UserProfile"),
														PackageAlias:         "partials_optional_68cf5cc2",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserProfile",
														ReferenceLocation: ast_domain.Location{
															Line:   56,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   106,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.UserProfile"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "UserProfile",
															ReferenceLocation: ast_domain.Location{
																Line:   56,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   106,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"profile": "pages_main_594861c5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:2",
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
										OriginalSourcePath: new("partials/optional.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "user-card",
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
										Name:          "profile",
										RawExpression: "state.UserProfile",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   56,
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
												Name: "UserProfile",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.UserProfile"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserProfile",
														ReferenceLocation: ast_domain.Location{
															Line:   56,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   106,
															Column: 2,
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
													TypeExpression:       typeExprFromString("models.UserProfile"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserProfile",
													ReferenceLocation: ast_domain.Location{
														Line:   56,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   106,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Location: ast_domain.Location{
											Line:   56,
											Column: 45,
										},
										NameLocation: ast_domain.Location{
											Line:   56,
											Column: 35,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("models.UserProfile"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserProfile",
												ReferenceLocation: ast_domain.Location{
													Line:   56,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   106,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
										TagName: "p-if",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_optional_68cf5cc2"),
											OriginalSourcePath:   new("partials/optional.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:6:2:0",
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
												OriginalSourcePath: new("partials/optional.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "condition",
												Value: "state.HasProfile",
												Location: ast_domain.Location{
													Line:   23,
													Column: 22,
												},
												NameLocation: ast_domain.Location{
													Line:   23,
													Column: 11,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   24,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_optional_68cf5cc2"),
													OriginalSourcePath:   new("partials/optional.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   24,
														Column: 21,
													},
													NameLocation: ast_domain.Location{
														Line:   24,
														Column: 13,
													},
													RawExpression: "state.ProfileName",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_optional_68cf5cc2.Response"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
																OriginalSourcePath: new("partials/optional.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ProfileName",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ProfileName",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   46,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
																OriginalSourcePath:  new("partials/optional.pk"),
																GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
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
																PackageAlias:         "partials_optional_68cf5cc2",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ProfileName",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
															OriginalSourcePath:  new("partials/optional.pk"),
															GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_optional_68cf5cc2",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ProfileName",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
														OriginalSourcePath:  new("partials/optional.pk"),
														GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:6:2:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/optional.pk"),
														Stringability:      1,
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   25,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_optional_68cf5cc2"),
													OriginalSourcePath:   new("partials/optional.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   25,
														Column: 21,
													},
													NameLocation: ast_domain.Location{
														Line:   25,
														Column: 13,
													},
													RawExpression: "state.ProfileEmail",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_optional_68cf5cc2.Response"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
																OriginalSourcePath: new("partials/optional.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ProfileEmail",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_optional_68cf5cc2",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ProfileEmail",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   47,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
																OriginalSourcePath:  new("partials/optional.pk"),
																GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
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
																PackageAlias:         "partials_optional_68cf5cc2",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ProfileEmail",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   47,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
															OriginalSourcePath:  new("partials/optional.pk"),
															GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_optional_68cf5cc2",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_optional_68cf5cc2",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ProfileEmail",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   47,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_optional_68cf5cc2Data_optional_profile_state_userprofile_b0f1fda6"),
														OriginalSourcePath:  new("partials/optional.pk"),
														GeneratedSourcePath: new("dist/partials/partials_optional_68cf5cc2/generated.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:6:2:0:1",
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/optional.pk"),
														Stringability:      1,
													},
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
										TagName: "p-else",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_optional_68cf5cc2"),
											OriginalSourcePath:   new("partials/optional.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:6:2:1",
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
												OriginalSourcePath: new("partials/optional.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   28,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_optional_68cf5cc2"),
													OriginalSourcePath:   new("partials/optional.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:6:2:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   28,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/optional.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   28,
															Column: 13,
														},
														TextContent: "No profile provided",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_optional_68cf5cc2"),
															OriginalSourcePath:   new("partials/optional.pk"),
															IsStatic:             true,
															IsStructurallyStatic: true,
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:6:2:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 13,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/optional.pk"),
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
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   59,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:7",
							RelativeLocation: ast_domain.Location{
								Line:   59,
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
								Value: "test-comprehensive",
								Location: ast_domain.Location{
									Line:   59,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   59,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   60,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
									RelativeLocation: ast_domain.Location{
										Line:   60,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   60,
											Column: 11,
										},
										TextContent: "7. Comprehensive (Multiple Tags)",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   60,
												Column: 11,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
									OriginalSourcePath:   new("partials/comprehensive.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c",
										PartialAlias:        "comprehensive",
										PartialPackageName:  "partials_comprehensive_3b6160d5",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   61,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"card-theme": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "default",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Theme",
											},
											"card-title": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 53,
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
														Name: "CompTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CompTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   107,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "CompTitle",
																	ReferenceLocation: ast_domain.Location{
																		Line:   61,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   107,
																		Column: 2,
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
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CompTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   61,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   107,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CompTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   107,
																	Column: 2,
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
													Line:   61,
													Column: 53,
												},
												GoFieldName: "Title",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CompTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   107,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CompTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   61,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   107,
																Column: 2,
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
											"description": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "No description provided",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Description",
											},
											"options": ast_domain.PropValue{
												Expression: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "partials_comprehensive_3b6160d5",
															RelativeLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       nil,
																	PackageAlias:         "partials_comprehensive_3b6160d5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
																},
																BaseCodeGenVarName: new("partials_comprehensive_3b6160d5"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "GetDefaultCardOptions",
															RelativeLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "partials_comprehensive_3b6160d5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetDefaultCardOptions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   61,
																		Column: 7,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("partials_comprehensive_3b6160d5"),
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
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "partials_comprehensive_3b6160d5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetDefaultCardOptions",
																ReferenceLocation: ast_domain.Location{
																	Line:   61,
																	Column: 7,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_comprehensive_3b6160d5"),
														},
													},
													Args: []ast_domain.Expression{},
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															BaseCodeGenVarName: new("partials_comprehensive_3b6160d5"),
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5"),
													},
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Options",
											},
											"priority": ast_domain.PropValue{
												Expression: &ast_domain.IntegerLiteral{
													Value: 1,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoFieldName: "Priority",
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"card-title": "pages_main_594861c5",
										"class":      "partials_comprehensive_3b6160d5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:1",
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
										OriginalSourcePath: new("partials/comprehensive.pk"),
										Stringability:      1,
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "card-title",
										RawExpression: "state.CompTitle",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 53,
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
												Name: "CompTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CompTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   107,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CompTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   61,
														Column: 53,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   107,
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
											Line:   61,
											Column: 53,
										},
										NameLocation: ast_domain.Location{
											Line:   61,
											Column: 40,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CompTitle",
												ReferenceLocation: ast_domain.Location{
													Line:   61,
													Column: 53,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   107,
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
										Name:          "class",
										RawExpression: "'comprehensive-card theme-' + state.Theme",
										Expression: &ast_domain.BinaryExpression{
											Left: &ast_domain.StringLiteral{
												Value: "comprehensive-card theme-",
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
													OriginalSourcePath: new("partials/comprehensive.pk"),
													Stringability:      1,
												},
											},
											Operator: "+",
											Right: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 31,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Theme",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 37,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Theme",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
														Stringability:       1,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 31,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Theme",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/comprehensive.pk"),
											Stringability:      1,
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
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 9,
											},
											RawExpression: "state.Title",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Title",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   51,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   51,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   51,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:1:0",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
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
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 16,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 8,
											},
											RawExpression: "state.Description",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Description",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Description",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Description",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Description",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   52,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:1:1",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
											IsStructurallyStatic: true,
										},
										DirIf: &ast_domain.Directive{
											Type: ast_domain.DirectiveIf,
											Location: ast_domain.Location{
												Line:   25,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   25,
												Column: 11,
											},
											RawExpression: "state.IsHighlighted",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "IsHighlighted",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IsHighlighted",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   54,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IsHighlighted",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   54,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IsHighlighted",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   54,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:1:2",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   25,
													Column: 38,
												},
												TextContent: "Highlighted!",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
													OriginalSourcePath:   new("partials/comprehensive.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:7:1:2:0",
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 38,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/comprehensive.pk"),
														Stringability:      1,
													},
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   26,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   26,
												Column: 11,
											},
											RawExpression: "state.Priority",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Priority",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Priority",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   55,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Priority",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   55,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Priority",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   55,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:1:3",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   27,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   27,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   27,
												Column: 11,
											},
											RawExpression: "state.Options.Shape",
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
																TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
																PackageAlias:         "partials_comprehensive_3b6160d5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
															OriginalSourcePath: new("partials/comprehensive.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Options",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Options",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   56,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
															OriginalSourcePath:  new("partials/comprehensive.pk"),
															GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Options",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   56,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Shape",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Shape",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("pkg/models/models.go"),
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Shape",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("pkg/models/models.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Shape",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   53,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_default_card_title_state_comptitle_description_no_description_provided_options_partials_comprehensive_3b6160d5_getdefaultcardoptions_priority_1_7a85f91c"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("pkg/models/models.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:1:4",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
									OriginalSourcePath:   new("partials/comprehensive.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01",
										PartialAlias:        "comprehensive",
										PartialPackageName:  "partials_comprehensive_3b6160d5",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   62,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"card-theme": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "dark",
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
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   66,
													Column: 21,
												},
												GoFieldName: "Theme",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"card-title": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   64,
																	Column: 22,
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
														Name: "CompTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CompTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   64,
																	Column: 22,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   107,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "CompTitle",
																	ReferenceLocation: ast_domain.Location{
																		Line:   64,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   107,
																		Column: 2,
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
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CompTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   64,
																Column: 22,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   107,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CompTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   64,
																	Column: 22,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   107,
																	Column: 2,
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
													Line:   64,
													Column: 22,
												},
												GoFieldName: "Title",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CompTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   64,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   107,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CompTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   64,
																Column: 22,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   107,
																Column: 2,
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
											"description": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "Custom description",
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
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   65,
													Column: 22,
												},
												GoFieldName: "Description",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"highlighted": ast_domain.PropValue{
												Expression: &ast_domain.BooleanLiteral{
													Value: true,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   67,
													Column: 22,
												},
												GoFieldName: "IsHighlighted",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
											"options": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
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
														Name: "CustomOptions",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CustomOptions",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   102,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.AvatarOptions"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "CustomOptions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   69,
																		Column: 19,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   102,
																		Column: 2,
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
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CustomOptions",
															ReferenceLocation: ast_domain.Location{
																Line:   69,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   102,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "CustomOptions",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   102,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Location: ast_domain.Location{
													Line:   69,
													Column: 19,
												},
												GoFieldName: "Options",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.AvatarOptions"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CustomOptions",
														ReferenceLocation: ast_domain.Location{
															Line:   69,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   102,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "CustomOptions",
															ReferenceLocation: ast_domain.Location{
																Line:   69,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   102,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											"priority": ast_domain.PropValue{
												Expression: &ast_domain.IntegerLiteral{
													Value: 5,
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   68,
													Column: 19,
												},
												GoFieldName: "Priority",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"card-title": "pages_main_594861c5",
										"class":      "partials_comprehensive_3b6160d5",
										"options":    "pages_main_594861c5",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:2",
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
										OriginalSourcePath: new("partials/comprehensive.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "card-theme",
										Value: "dark",
										Location: ast_domain.Location{
											Line:   66,
											Column: 21,
										},
										NameLocation: ast_domain.Location{
											Line:   66,
											Column: 9,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "description",
										Value: "Custom description",
										Location: ast_domain.Location{
											Line:   65,
											Column: 22,
										},
										NameLocation: ast_domain.Location{
											Line:   65,
											Column: 9,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "highlighted",
										Value: "true",
										Location: ast_domain.Location{
											Line:   67,
											Column: 22,
										},
										NameLocation: ast_domain.Location{
											Line:   67,
											Column: 9,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "priority",
										Value: "5",
										Location: ast_domain.Location{
											Line:   68,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   68,
											Column: 9,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "card-title",
										RawExpression: "state.CompTitle",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   64,
															Column: 22,
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
												Name: "CompTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CompTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   64,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   107,
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
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CompTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   64,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   107,
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
											Line:   64,
											Column: 22,
										},
										NameLocation: ast_domain.Location{
											Line:   64,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CompTitle",
												ReferenceLocation: ast_domain.Location{
													Line:   64,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   107,
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
										Name:          "class",
										RawExpression: "'comprehensive-card theme-' + state.Theme",
										Expression: &ast_domain.BinaryExpression{
											Left: &ast_domain.StringLiteral{
												Value: "comprehensive-card theme-",
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
													OriginalSourcePath: new("partials/comprehensive.pk"),
													Stringability:      1,
												},
											},
											Operator: "+",
											Right: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 31,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Theme",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 37,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Theme",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
														Stringability:       1,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 31,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Theme",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   22,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 8,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/comprehensive.pk"),
											Stringability:      1,
										},
									},
									ast_domain.DynamicAttribute{
										Name:          "options",
										RawExpression: "state.CustomOptions",
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   69,
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
												Name: "CustomOptions",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.AvatarOptions"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CustomOptions",
														ReferenceLocation: ast_domain.Location{
															Line:   69,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   102,
															Column: 2,
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
													TypeExpression:       typeExprFromString("models.AvatarOptions"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CustomOptions",
													ReferenceLocation: ast_domain.Location{
														Line:   69,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   102,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Location: ast_domain.Location{
											Line:   69,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   69,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("models.AvatarOptions"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CustomOptions",
												ReferenceLocation: ast_domain.Location{
													Line:   69,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   102,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 9,
											},
											RawExpression: "state.Title",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Title",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   51,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   51,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   51,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:2:0",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
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
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 16,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 8,
											},
											RawExpression: "state.Description",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Description",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Description",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Description",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Description",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   52,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:2:1",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
											IsStructurallyStatic: true,
										},
										DirIf: &ast_domain.Directive{
											Type: ast_domain.DirectiveIf,
											Location: ast_domain.Location{
												Line:   25,
												Column: 17,
											},
											NameLocation: ast_domain.Location{
												Line:   25,
												Column: 11,
											},
											RawExpression: "state.IsHighlighted",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "IsHighlighted",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IsHighlighted",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   54,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IsHighlighted",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   54,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IsHighlighted",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 17,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   54,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:2:2",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   25,
													Column: 38,
												},
												TextContent: "Highlighted!",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
													OriginalSourcePath:   new("partials/comprehensive.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:7:2:2:0",
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 38,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/comprehensive.pk"),
														Stringability:      1,
													},
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   26,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   26,
												Column: 11,
											},
											RawExpression: "state.Priority",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath: new("partials/comprehensive.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Priority",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_comprehensive_3b6160d5",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Priority",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   55,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
														PackageAlias:         "partials_comprehensive_3b6160d5",
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Priority",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   55,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_comprehensive_3b6160d5",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Priority",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   55,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:2:3",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
												Stringability:      1,
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   27,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_comprehensive_3b6160d5"),
											OriginalSourcePath:   new("partials/comprehensive.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   27,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   27,
												Column: 11,
											},
											RawExpression: "state.Options.Shape",
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
																TypeExpression:       typeExprFromString("partials_comprehensive_3b6160d5.Response"),
																PackageAlias:         "partials_comprehensive_3b6160d5",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/dist/partials/partials_comprehensive_3b6160d5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
															OriginalSourcePath: new("partials/comprehensive.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Options",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.AvatarOptions"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Options",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   56,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
															OriginalSourcePath:  new("partials/comprehensive.pk"),
															GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
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
															TypeExpression:       typeExprFromString("models.AvatarOptions"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Options",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   56,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("dist/partials/partials_comprehensive_3b6160d5/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Shape",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Shape",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   53,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
														OriginalSourcePath:  new("partials/comprehensive.pk"),
														GeneratedSourcePath: new("pkg/models/models.go"),
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
														CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Shape",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   53,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
													OriginalSourcePath:  new("partials/comprehensive.pk"),
													GeneratedSourcePath: new("pkg/models/models.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_072_prop_tags_comprehensive/pkg/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Shape",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   53,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_comprehensive_3b6160d5Data_comprehensive_card_theme_dark_card_title_state_comptitle_description_custom_description_highlighted_true_options_state_customoptions_priority_5_98c8be01"),
												OriginalSourcePath:  new("partials/comprehensive.pk"),
												GeneratedSourcePath: new("pkg/models/models.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:7:2:4",
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
												OriginalSourcePath: new("partials/comprehensive.pk"),
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
		},
	}
}()
