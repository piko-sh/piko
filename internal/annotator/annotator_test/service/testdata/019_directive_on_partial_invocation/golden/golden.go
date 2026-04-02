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
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   45,
							Column: 9,
						},
						TagName: "h2",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   45,
									Column: 13,
								},
								TextContent: "Active Users:",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   45,
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
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   46,
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
								Line:   46,
								Column: 21,
							},
							NameLocation: ast_domain.Location{
								Line:   46,
								Column: 14,
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
											CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "user",
											ReferenceLocation: ast_domain.Location{
												Line:   30,
												Column: 25,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("user"),
										OriginalSourcePath: new("partials/user_badge.pk"),
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
												CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   46,
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
										Name: "Users",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 15,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]models.User"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Users",
												ReferenceLocation: ast_domain.Location{
													Line:   46,
													Column: 21,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 23,
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
											CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Users",
											ReferenceLocation: ast_domain.Location{
												Line:   46,
												Column: 21,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 23,
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
										CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   46,
											Column: 21,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
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
										CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Users",
										ReferenceLocation: ast_domain.Location{
											Line:   46,
											Column: 21,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
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
									CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Users",
									ReferenceLocation: ast_domain.Location{
										Line:   46,
										Column: 21,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   31,
										Column: 23,
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
										Line:   46,
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
									Literal: "r.0:1.",
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
												CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "user",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
													Column: 25,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("user"),
											OriginalSourcePath: new("partials/user_badge.pk"),
										},
									},
								},
							},
							RelativeLocation: ast_domain.Location{
								Line:   46,
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
									Line:   30,
									Column: 5,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_user_badge_2e143ad8"),
									OriginalSourcePath:   new("partials/user_badge.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "badge_name_user_name_0f53860a",
										PartialAlias:        "badge",
										PartialPackageName:  "partials_user_badge_2e143ad8",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   47,
											Column: 13,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"name": ast_domain.PropValue{
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
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "user",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 66,
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
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 66,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 66,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   43,
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
															CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 66,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 66,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
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
													Line:   47,
													Column: 66,
												},
												GoFieldName: "Name",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 66,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 66,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
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
										"name":  "main_aaf9a2e0",
										"title": "main_aaf9a2e0",
									},
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   47,
										Column: 44,
									},
									NameLocation: ast_domain.Location{
										Line:   47,
										Column: 38,
									},
									RawExpression: "user.IsActive",
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
													CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "user",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 44,
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
											Name: "IsActive",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 6,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IsActive",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   44,
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
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "IsActive",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 44,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   44,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("user"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/user.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "IsActive",
											ReferenceLocation: ast_domain.Location{
												Line:   47,
												Column: 44,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   44,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("user"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/user.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
												OriginalSourcePath: new("partials/user_badge.pk"),
												Stringability:      1,
											},
											Literal: "r.0:1.",
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
												OriginalSourcePath: new("partials/user_badge.pk"),
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
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "user",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 25,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("user"),
													OriginalSourcePath: new("partials/user_badge.pk"),
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
												OriginalSourcePath: new("partials/user_badge.pk"),
												Stringability:      1,
											},
											Literal: ":0",
										},
									},
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
										OriginalSourcePath: new("partials/user_badge.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "badge",
										Location: ast_domain.Location{
											Line:   30,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   30,
											Column: 11,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "name",
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
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "user",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 66,
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
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 66,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
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
													CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 66,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
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
											Line:   47,
											Column: 66,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 59,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 66,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("user"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/user.go"),
											Stringability:       1,
										},
									},
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
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "user",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 85,
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
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 85,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
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
													CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 85,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
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
											Line:   47,
											Column: 85,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 77,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 85,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
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
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   30,
											Column: 25,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_user_badge_2e143ad8"),
											OriginalSourcePath:   new("partials/user_badge.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 25,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/user_badge.pk"),
														Stringability:      1,
													},
													Literal: "r.0:1.",
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
														OriginalSourcePath: new("partials/user_badge.pk"),
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
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "user",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 25,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("user"),
															OriginalSourcePath: new("partials/user_badge.pk"),
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 25,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/user_badge.pk"),
														Stringability:      1,
													},
													Literal: ":0:0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   30,
												Column: 25,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/user_badge.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   30,
													Column: 25,
												},
												Literal: "\n        ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("partials/user_badge.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   31,
													Column: 12,
												},
												RawExpression: "props.Name",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_user_badge_2e143ad8.Props"),
																PackageAlias:         "partials_user_badge_2e143ad8",
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/dist/partials/partials_user_badge_2e143ad8",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_badge_name_user_name_0f53860a"),
															OriginalSourcePath: new("partials/user_badge.pk"),
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
																PackageAlias:         "partials_user_badge_2e143ad8",
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/dist/partials/partials_user_badge_2e143ad8",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 12,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   26,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   47,
																		Column: 66,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   43,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("user"),
															},
															BaseCodeGenVarName:  new("props_badge_name_user_name_0f53860a"),
															OriginalSourcePath:  new("partials/user_badge.pk"),
															GeneratedSourcePath: new("dist/partials/partials_user_badge_2e143ad8/generated.go"),
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
															PackageAlias:         "partials_user_badge_2e143ad8",
															CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/dist/partials/partials_user_badge_2e143ad8",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 12,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   26,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   47,
																	Column: 66,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("user"),
														},
														BaseCodeGenVarName:  new("props_badge_name_user_name_0f53860a"),
														OriginalSourcePath:  new("partials/user_badge.pk"),
														GeneratedSourcePath: new("dist/partials/partials_user_badge_2e143ad8/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_user_badge_2e143ad8",
														CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/dist/partials/partials_user_badge_2e143ad8",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 12,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   26,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_19_directive_on_partial_invocation/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   47,
																Column: 66,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("user"),
													},
													BaseCodeGenVarName:  new("props_badge_name_user_name_0f53860a"),
													OriginalSourcePath:  new("partials/user_badge.pk"),
													GeneratedSourcePath: new("dist/partials/partials_user_badge_2e143ad8/generated.go"),
													Stringability:       1,
												},
											},
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   31,
													Column: 25,
												},
												Literal: "\n    ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("partials/user_badge.pk"),
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
