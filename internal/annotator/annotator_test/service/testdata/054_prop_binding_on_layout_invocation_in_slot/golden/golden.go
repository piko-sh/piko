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
					Line:   32,
					Column: 2,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_outer_layout_643254a4"),
					OriginalSourcePath:   new("partials/outer_layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "outer_layout_118242d2",
						PartialAlias:        "outer_layout",
						PartialPackageName:  "partials_outer_layout_643254a4",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   42,
							Column: 2,
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   32,
						Column: 2,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("partials/outer_layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "outer-layout",
						Location: ast_domain.Location{
							Line:   32,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   32,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   40,
							Column: 2,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_inner_layout_981558c4"),
							OriginalSourcePath:   new("partials/inner_layout.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702",
								PartialAlias:        "inner_layout",
								PartialPackageName:  "partials_inner_layout_981558c4",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   43,
									Column: 3,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"sidebar_visible": ast_domain.PropValue{
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
														CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 71,
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
												Name: "ShowSidebar",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ShowSidebar",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   35,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ShowSidebar",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   35,
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
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ShowSidebar",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ShowSidebar",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   35,
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
											Line:   43,
											Column: 71,
										},
										GoFieldName: "SidebarVisible",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ShowSidebar",
												ReferenceLocation: ast_domain.Location{
													Line:   43,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   35,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ShowSidebar",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
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
								"sidebar_visible": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   40,
								Column: 2,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/inner_layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "inner-layout",
								Location: ast_domain.Location{
									Line:   40,
									Column: 14,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 7,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "inner-layout",
								Location: ast_domain.Location{
									Line:   43,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   43,
									Column: 17,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "sidebar_visible",
								RawExpression: "state.ShowSidebar",
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
												CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   43,
													Column: 71,
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
										Name: "ShowSidebar",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ShowSidebar",
												ReferenceLocation: ast_domain.Location{
													Line:   43,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   35,
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
											CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ShowSidebar",
											ReferenceLocation: ast_domain.Location{
												Line:   43,
												Column: 71,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   35,
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
									Column: 71,
								},
								NameLocation: ast_domain.Location{
									Line:   43,
									Column: 53,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("bool"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ShowSidebar",
										ReferenceLocation: ast_domain.Location{
											Line:   43,
											Column: 71,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   35,
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
									Line:   41,
									Column: 3,
								},
								TagName: "aside",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_inner_layout_981558c4"),
									OriginalSourcePath:   new("partials/inner_layout.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   41,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   41,
										Column: 10,
									},
									RawExpression: "state.HasSidebar",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("partials_inner_layout_981558c4.Response"),
													PackageAlias:         "partials_inner_layout_981558c4",
													CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_inner_layout_981558c4",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   41,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_inner_layout_981558c4Data_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702"),
												OriginalSourcePath: new("partials/inner_layout.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "HasSidebar",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_inner_layout_981558c4",
													CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_inner_layout_981558c4",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "HasSidebar",
													ReferenceLocation: ast_domain.Location{
														Line:   41,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   28,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_inner_layout_981558c4Data_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702"),
												OriginalSourcePath:  new("partials/inner_layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_inner_layout_981558c4/generated.go"),
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
												PackageAlias:         "partials_inner_layout_981558c4",
												CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_inner_layout_981558c4",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "HasSidebar",
												ReferenceLocation: ast_domain.Location{
													Line:   41,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   28,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_inner_layout_981558c4Data_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702"),
											OriginalSourcePath:  new("partials/inner_layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_inner_layout_981558c4/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "partials_inner_layout_981558c4",
											CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_inner_layout_981558c4",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "HasSidebar",
											ReferenceLocation: ast_domain.Location{
												Line:   41,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   28,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("partials_inner_layout_981558c4Data_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702"),
										OriginalSourcePath:  new("partials/inner_layout.pk"),
										GeneratedSourcePath: new("dist/partials/partials_inner_layout_981558c4/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   41,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/inner_layout.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "sidebar",
										Location: ast_domain.Location{
											Line:   41,
											Column: 41,
										},
										NameLocation: ast_domain.Location{
											Line:   41,
											Column: 34,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   41,
											Column: 50,
										},
										TextContent: "Sidebar",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_inner_layout_981558c4"),
											OriginalSourcePath:   new("partials/inner_layout.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 50,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/inner_layout.pk"),
												Stringability:      1,
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   42,
									Column: 3,
								},
								TagName: "main",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_inner_layout_981558c4"),
									OriginalSourcePath:   new("partials/inner_layout.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   42,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/inner_layout.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "inner-content",
										Location: ast_domain.Location{
											Line:   42,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   42,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   40,
											Column: 2,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "child_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702_data_state_contentdata_71e74391",
												PartialAlias:        "child",
												PartialPackageName:  "partials_child_d247007e",
												InvokerPackageAlias: "main_aaf9a2e0",
												Location: ast_domain.Location{
													Line:   44,
													Column: 4,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"data": ast_domain.PropValue{
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
																		CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 54,
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
																Name: "ContentData",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ContentData",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   36,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "main_aaf9a2e0",
																			CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ContentData",
																			ReferenceLocation: ast_domain.Location{
																				Line:   44,
																				Column: 54,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   36,
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
																	CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ContentData",
																	ReferenceLocation: ast_domain.Location{
																		Line:   44,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   36,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ContentData",
																		ReferenceLocation: ast_domain.Location{
																			Line:   44,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   36,
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
															Line:   44,
															Column: 54,
														},
														GoFieldName: "Data",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ContentData",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   36,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ContentData",
																	ReferenceLocation: ast_domain.Location{
																		Line:   44,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   36,
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
												"data": "main_aaf9a2e0",
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   40,
												Column: 2,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/child.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "child",
												Location: ast_domain.Location{
													Line:   40,
													Column: 14,
												},
												NameLocation: ast_domain.Location{
													Line:   40,
													Column: 7,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "id",
												Value: "nested-child",
												Location: ast_domain.Location{
													Line:   44,
													Column: 22,
												},
												NameLocation: ast_domain.Location{
													Line:   44,
													Column: 18,
												},
											},
										},
										DynamicAttributes: []ast_domain.DynamicAttribute{
											ast_domain.DynamicAttribute{
												Name:          "data",
												RawExpression: "state.ContentData",
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
																CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 54,
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
														Name: "ContentData",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ContentData",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   36,
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
															CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ContentData",
															ReferenceLocation: ast_domain.Location{
																Line:   44,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   36,
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
													Line:   44,
													Column: 54,
												},
												NameLocation: ast_domain.Location{
													Line:   44,
													Column: 47,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ContentData",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   36,
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
													Line:   41,
													Column: 3,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_child_d247007e"),
													OriginalSourcePath:   new("partials/child.pk"),
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   41,
														Column: 3,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/child.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   41,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_child_d247007e"),
															OriginalSourcePath:   new("partials/child.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:1:0:0:0",
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
																OriginalSourcePath: new("partials/child.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   41,
																	Column: 12,
																},
																RawExpression: "state.Content",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_child_d247007e.Response"),
																				PackageAlias:         "partials_child_d247007e",
																				CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_child_d247007e",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   41,
																					Column: 12,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702_data_state_contentdata_71e74391"),
																			OriginalSourcePath: new("partials/child.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Content",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_child_d247007e",
																				CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_child_d247007e",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Content",
																				ReferenceLocation: ast_domain.Location{
																					Line:   41,
																					Column: 12,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   28,
																					Column: 23,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702_data_state_contentdata_71e74391"),
																			OriginalSourcePath:  new("partials/child.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
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
																			PackageAlias:         "partials_child_d247007e",
																			CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_child_d247007e",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Content",
																			ReferenceLocation: ast_domain.Location{
																				Line:   41,
																				Column: 12,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   28,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702_data_state_contentdata_71e74391"),
																		OriginalSourcePath:  new("partials/child.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_child_d247007e",
																		CanonicalPackagePath: "testcase_54_prop_binding_on_layout_invocation_in_slot/dist/partials/partials_child_d247007e",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Content",
																		ReferenceLocation: ast_domain.Location{
																			Line:   41,
																			Column: 12,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   28,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_child_d247007eData_child_inner_layout_outer_layout_118242d2_sidebar_visible_state_showsidebar_9bde2702_data_state_contentdata_71e74391"),
																	OriginalSourcePath:  new("partials/child.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
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
					},
				},
			},
		},
	}
}()
