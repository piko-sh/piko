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
					Line:   49,
					Column: 5,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   49,
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
							Line:   50,
							Column: 9,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   50,
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
								Value: "test-1",
								Location: ast_domain.Location{
									Line:   50,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   50,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   51,
									Column: 13,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   51,
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
											Line:   51,
											Column: 17,
										},
										TextContent: "Static Invocations",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   51,
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
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   36,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "card_title_state_firstcardtitle_c70ea489",
										PartialAlias:        "card",
										PartialPackageName:  "partials_card_bfc4a3cf",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   52,
											Column: 13,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"title": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 45,
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
														Name: "FirstCardTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FirstCardTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "FirstCardTitle",
																	ReferenceLocation: ast_domain.Location{
																		Line:   52,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   32,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
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
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FirstCardTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FirstCardTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   32,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   52,
													Column: 45,
												},
												GoFieldName: "Title",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FirstCardTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FirstCardTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"title": "main_aaf9a2e0",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   36,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "card",
										Location: ast_domain.Location{
											Line:   36,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   36,
											Column: 10,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "title",
										RawExpression: "state.FirstCardTitle",
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
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 45,
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
												Name: "FirstCardTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FirstCardTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 45,
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
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FirstCardTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   52,
														Column: 45,
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
										Location: ast_domain.Location{
											Line:   52,
											Column: 45,
										},
										NameLocation: ast_domain.Location{
											Line:   52,
											Column: 37,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FirstCardTitle",
												ReferenceLocation: ast_domain.Location{
													Line:   52,
													Column: 45,
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
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   37,
											Column: 9,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   37,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/card.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   37,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_card_bfc4a3cf"),
													OriginalSourcePath:   new("partials/card.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   37,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   37,
															Column: 15,
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
																		TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   37,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
																	OriginalSourcePath: new("partials/card.pk"),
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
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   37,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   26,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
																	OriginalSourcePath:  new("partials/card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 15,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   26,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
																OriginalSourcePath:  new("partials/card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   37,
																	Column: 15,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   26,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
															OriginalSourcePath:  new("partials/card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
											Line:   38,
											Column: 9,
										},
										TagName: "h3",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   38,
												Column: 21,
											},
											NameLocation: ast_domain.Location{
												Line:   38,
												Column: 13,
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
															TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
															PackageAlias:         "partials_card_bfc4a3cf",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
														OriginalSourcePath: new("partials/card.pk"),
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
															PackageAlias:         "partials_card_bfc4a3cf",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   26,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
														OriginalSourcePath:  new("partials/card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   38,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   26,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_card_bfc4a3cf",
													CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   38,
														Column: 21,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   26,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_firstcardtitle_c70ea489"),
												OriginalSourcePath:  new("partials/card.pk"),
												GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:1",
											RelativeLocation: ast_domain.Location{
												Line:   38,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/card.pk"),
												Stringability:      1,
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   36,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "card_title_state_secondcardtitle_13d929b8",
										PartialAlias:        "card",
										PartialPackageName:  "partials_card_bfc4a3cf",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   53,
											Column: 13,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"title": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 45,
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
														Name: "SecondCardTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "SecondCardTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   33,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "SecondCardTitle",
																	ReferenceLocation: ast_domain.Location{
																		Line:   53,
																		Column: 45,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   33,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
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
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "SecondCardTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   33,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "SecondCardTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   53,
																	Column: 45,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   33,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   53,
													Column: 45,
												},
												GoFieldName: "Title",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SecondCardTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   33,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "SecondCardTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   53,
																Column: 45,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   33,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"title": "main_aaf9a2e0",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:2",
									RelativeLocation: ast_domain.Location{
										Line:   36,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "card",
										Location: ast_domain.Location{
											Line:   36,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   36,
											Column: 10,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "title",
										RawExpression: "state.SecondCardTitle",
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
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 45,
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
												Name: "SecondCardTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SecondCardTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 45,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   33,
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
													CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "SecondCardTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 45,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   33,
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
											Line:   53,
											Column: 45,
										},
										NameLocation: ast_domain.Location{
											Line:   53,
											Column: 37,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "SecondCardTitle",
												ReferenceLocation: ast_domain.Location{
													Line:   53,
													Column: 45,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   33,
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   37,
											Column: 9,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   37,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/card.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   37,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_card_bfc4a3cf"),
													OriginalSourcePath:   new("partials/card.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:2:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   37,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   37,
															Column: 15,
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
																		TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   37,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
																	OriginalSourcePath: new("partials/card.pk"),
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
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   37,
																			Column: 15,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   26,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
																	OriginalSourcePath:  new("partials/card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   37,
																		Column: 15,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   26,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
																OriginalSourcePath:  new("partials/card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_card_bfc4a3cf",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   37,
																	Column: 15,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   26,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
															OriginalSourcePath:  new("partials/card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
											Line:   38,
											Column: 9,
										},
										TagName: "h3",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_card_bfc4a3cf"),
											OriginalSourcePath:   new("partials/card.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   38,
												Column: 21,
											},
											NameLocation: ast_domain.Location{
												Line:   38,
												Column: 13,
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
															TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
															PackageAlias:         "partials_card_bfc4a3cf",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
														OriginalSourcePath: new("partials/card.pk"),
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
															PackageAlias:         "partials_card_bfc4a3cf",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   26,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
														OriginalSourcePath:  new("partials/card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
														PackageAlias:         "partials_card_bfc4a3cf",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   38,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   26,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
													OriginalSourcePath:  new("partials/card.pk"),
													GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_card_bfc4a3cf",
													CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   38,
														Column: 21,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   26,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_state_secondcardtitle_13d929b8"),
												OriginalSourcePath:  new("partials/card.pk"),
												GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:2:1",
											RelativeLocation: ast_domain.Location{
												Line:   38,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/card.pk"),
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
							Line:   56,
							Column: 9,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   56,
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
								Value: "test-2",
								Location: ast_domain.Location{
									Line:   56,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   56,
									Column: 14,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   57,
									Column: 13,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
											Column: 17,
										},
										TextContent: "Looped Invocations",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   57,
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
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   58,
									Column: 13,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
									RelativeLocation: ast_domain.Location{
										Line:   58,
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   59,
											Column: 17,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   59,
												Column: 28,
											},
											NameLocation: ast_domain.Location{
												Line:   59,
												Column: 21,
											},
											RawExpression: "user in state.Users",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: &ast_domain.Identifier{
													Name: "__pikoLoopIdx",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "__pikoLoopIdx",
															ReferenceLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("__pikoLoopIdx"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												ItemVariable: &ast_domain.Identifier{
													Name: "user",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.User"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "user",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 9,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("user"),
														OriginalSourcePath: new("partials/card.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 28,
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
														Name: "Users",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Users",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 28,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   34,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]models.User"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Users",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 28,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   34,
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
														TypeExpression:       typeExprFromString("[]models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Users",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Users",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
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
													TypeExpression:       typeExprFromString("[]models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Users",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 28,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   34,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   59,
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
													Literal: "r.0:1:1:0.",
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
													Expression: &ast_domain.Identifier{
														Name: "user",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "user",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 9,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("user"),
															OriginalSourcePath: new("partials/card.pk"),
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   59,
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
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   36,
													Column: 5,
												},
												TagName: "div",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_card_bfc4a3cf"),
													OriginalSourcePath:   new("partials/card.pk"),
													PartialInfo: &ast_domain.PartialInvocationInfo{
														InvocationKey:       "card_title_user_name_1544ecfa",
														PartialAlias:        "card",
														PartialPackageName:  "partials_card_bfc4a3cf",
														InvokerPackageAlias: "main_aaf9a2e0",
														Location: ast_domain.Location{
															Line:   60,
															Column: 21,
														},
														PassedProps: map[string]ast_domain.PropValue{
															"title": ast_domain.PropValue{
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "user",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("models.User"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "user",
																				ReferenceLocation: ast_domain.Location{
																					Line:   60,
																					Column: 53,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("user"),
																			OriginalSourcePath: new("main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Name",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 6,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Name",
																				ReferenceLocation: ast_domain.Location{
																					Line:   60,
																					Column: 53,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   42,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "models",
																					CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Name",
																					ReferenceLocation: ast_domain.Location{
																						Line:   60,
																						Column: 53,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   42,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName: new("user"),
																			},
																			BaseCodeGenVarName:  new("user"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("models/user.go"),
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
																			CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Name",
																			ReferenceLocation: ast_domain.Location{
																				Line:   60,
																				Column: 53,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   42,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Name",
																				ReferenceLocation: ast_domain.Location{
																					Line:   60,
																					Column: 53,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   42,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName: new("user"),
																		},
																		BaseCodeGenVarName:  new("user"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("models/user.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   60,
																	Column: 53,
																},
																GoFieldName: "Title",
																InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   60,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   42,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Name",
																			ReferenceLocation: ast_domain.Location{
																				Line:   60,
																				Column: 53,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   42,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("user"),
																	},
																	BaseCodeGenVarName:  new("user"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/user.go"),
																	Stringability:       1,
																},
															},
															IsLoopDependent: true,
														},
													},
													DynamicAttributeOrigins: map[string]string{
														"title": "main_aaf9a2e0",
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   36,
																Column: 5,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/card.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:1:0.",
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
																OriginalSourcePath: new("partials/card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "user",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.User"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "user",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 9,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("user"),
																	OriginalSourcePath: new("partials/card.pk"),
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   36,
																Column: 5,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/card.pk"),
																Stringability:      1,
															},
															Literal: ":0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   36,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "card",
														Location: ast_domain.Location{
															Line:   36,
															Column: 17,
														},
														NameLocation: ast_domain.Location{
															Line:   36,
															Column: 10,
														},
													},
												},
												DynamicAttributes: []ast_domain.DynamicAttribute{
													ast_domain.DynamicAttribute{
														Name:          "title",
														RawExpression: "user.Name",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "user",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.User"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "user",
																		ReferenceLocation: ast_domain.Location{
																			Line:   60,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("user"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Name",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 6,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Name",
																		ReferenceLocation: ast_domain.Location{
																			Line:   60,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   42,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("user"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/user.go"),
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
																	CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   60,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("user"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/user.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   60,
															Column: 53,
														},
														NameLocation: ast_domain.Location{
															Line:   60,
															Column: 45,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   60,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("user"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/user.go"),
															Stringability:       1,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   37,
															Column: 9,
														},
														TagName: "p",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_card_bfc4a3cf"),
															OriginalSourcePath:   new("partials/card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   37,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:1:0.",
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
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "user",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("models.User"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "user",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 9,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("user"),
																			OriginalSourcePath: new("partials/card.pk"),
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   37,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":0:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   37,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/card.pk"),
																Stringability:      1,
															},
														},
														Children: []*ast_domain.TemplateNode{
															&ast_domain.TemplateNode{
																NodeType: ast_domain.NodeText,
																Location: ast_domain.Location{
																	Line:   37,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_card_bfc4a3cf"),
																	OriginalSourcePath:   new("partials/card.pk"),
																},
																Key: &ast_domain.TemplateLiteral{
																	Parts: []ast_domain.TemplateLiteralPart{
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
																			RelativeLocation: ast_domain.Location{
																				Line:   37,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/card.pk"),
																				Stringability:      1,
																			},
																			Literal: "r.0:1:1:0.",
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
																				OriginalSourcePath: new("partials/card.pk"),
																				Stringability:      1,
																			},
																			Expression: &ast_domain.Identifier{
																				Name: "user",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("models.User"),
																						PackageAlias:         "models",
																						CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "user",
																						ReferenceLocation: ast_domain.Location{
																							Line:   38,
																							Column: 9,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("user"),
																					OriginalSourcePath: new("partials/card.pk"),
																				},
																			},
																		},
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
																			RelativeLocation: ast_domain.Location{
																				Line:   37,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/card.pk"),
																				Stringability:      1,
																			},
																			Literal: ":0:0:0",
																		},
																	},
																	RelativeLocation: ast_domain.Location{
																		Line:   37,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																},
																RichText: []ast_domain.TextPart{
																	ast_domain.TextPart{
																		IsLiteral: false,
																		Location: ast_domain.Location{
																			Line:   37,
																			Column: 15,
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
																						TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
																						PackageAlias:         "partials_card_bfc4a3cf",
																						CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "state",
																						ReferenceLocation: ast_domain.Location{
																							Line:   37,
																							Column: 15,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																					OriginalSourcePath: new("partials/card.pk"),
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
																						PackageAlias:         "partials_card_bfc4a3cf",
																						CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Title",
																						ReferenceLocation: ast_domain.Location{
																							Line:   37,
																							Column: 15,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   26,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																					OriginalSourcePath:  new("partials/card.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
																					PackageAlias:         "partials_card_bfc4a3cf",
																					CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Title",
																					ReferenceLocation: ast_domain.Location{
																						Line:   37,
																						Column: 15,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   26,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																				OriginalSourcePath:  new("partials/card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																				Stringability:       1,
																			},
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_card_bfc4a3cf",
																				CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Title",
																				ReferenceLocation: ast_domain.Location{
																					Line:   37,
																					Column: 15,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   26,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																			OriginalSourcePath:  new("partials/card.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
															Line:   38,
															Column: 9,
														},
														TagName: "h3",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_card_bfc4a3cf"),
															OriginalSourcePath:   new("partials/card.pk"),
														},
														DirText: &ast_domain.Directive{
															Type: ast_domain.DirectiveText,
															Location: ast_domain.Location{
																Line:   38,
																Column: 21,
															},
															NameLocation: ast_domain.Location{
																Line:   38,
																Column: 13,
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
																			TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
																			PackageAlias:         "partials_card_bfc4a3cf",
																			CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "state",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 21,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																		OriginalSourcePath: new("partials/card.pk"),
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
																			PackageAlias:         "partials_card_bfc4a3cf",
																			CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Title",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 21,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   26,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																		OriginalSourcePath:  new("partials/card.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
																		PackageAlias:         "partials_card_bfc4a3cf",
																		CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 21,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   26,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																	OriginalSourcePath:  new("partials/card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																	Stringability:       1,
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_card_bfc4a3cf",
																	CanonicalPackagePath: "testcase_37_isolated_invocation_context/dist/partials/partials_card_bfc4a3cf",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   26,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("partials_card_bfc4a3cfData_card_title_user_name_1544ecfa"),
																OriginalSourcePath:  new("partials/card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
																Stringability:       1,
															},
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   38,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:1:0.",
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
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "user",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("models.User"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "testcase_37_isolated_invocation_context/models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "user",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 9,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("user"),
																			OriginalSourcePath: new("partials/card.pk"),
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   38,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":0:1",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   38,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/card.pk"),
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
				},
			},
		},
	}
}()
