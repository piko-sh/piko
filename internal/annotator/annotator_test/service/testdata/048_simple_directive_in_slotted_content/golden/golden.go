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
							Line:   43,
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
											CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   43,
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
											CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Query",
											ReferenceLocation: ast_domain.Location{
												Line:   43,
												Column: 44,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
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
										CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Query",
										ReferenceLocation: ast_domain.Location{
											Line:   43,
											Column: 44,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
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
							Line:   43,
							Column: 44,
						},
						NameLocation: ast_domain.Location{
							Line:   43,
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
																CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
																CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
															CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
														CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
									Line:   44,
									Column: 9,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   44,
										Column: 22,
									},
									NameLocation: ast_domain.Location{
										Line:   44,
										Column: 14,
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
													TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   44,
														Column: 22,
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
											Name: "Message",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Message",
													ReferenceLocation: ast_domain.Location{
														Line:   44,
														Column: 22,
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
												CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Message",
												ReferenceLocation: ast_domain.Location{
													Line:   44,
													Column: 22,
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
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Message",
											ReferenceLocation: ast_domain.Location{
												Line:   44,
												Column: 22,
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
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   44,
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
										Name:  "class",
										Value: "message-box",
										Location: ast_domain.Location{
											Line:   44,
											Column: 44,
										},
										NameLocation: ast_domain.Location{
											Line:   44,
											Column: 37,
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
																CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
																CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
															CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
														CanonicalPackagePath: "testcase_48_simple_directive_in_slotted_content/dist/partials/partials_layout_ee037d9a",
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
