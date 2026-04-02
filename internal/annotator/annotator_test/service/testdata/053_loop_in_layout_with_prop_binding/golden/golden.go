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
					OriginalPackageAlias: new("partials_layout_ee037d9a"),
					OriginalSourcePath:   new("partials/layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "layout_1745aa65",
						PartialAlias:        "layout",
						PartialPackageName:  "partials_layout_ee037d9a",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   46,
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
						OriginalSourcePath: new("partials/layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "layout-wrapper",
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
							Line:   33,
							Column: 3,
						},
						TagName: "header",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   33,
								Column: 3,
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
									Line:   33,
									Column: 11,
								},
								TextContent: "Layout Header",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   33,
										Column: 11,
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
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   34,
							Column: 3,
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
								Line:   34,
								Column: 3,
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
								Value: "content",
								Location: ast_domain.Location{
									Line:   34,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   34,
									Column: 9,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   47,
									Column: 3,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   47,
										Column: 35,
									},
									NameLocation: ast_domain.Location{
										Line:   47,
										Column: 28,
									},
									RawExpression: "(idx, item) in state.Items",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: &ast_domain.Identifier{
											Name: "idx",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 2,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "idx",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 9,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("idx"),
												OriginalSourcePath: new("partials/child.pk"),
												Stringability:      1,
											},
										},
										ItemVariable: &ast_domain.Identifier{
											Name: "item",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.Item"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "item",
													ReferenceLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("item"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 16,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 35,
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
												Name: "Items",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 22,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]models.Item"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   33,
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
												Column: 16,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]models.Item"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   33,
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
												TypeExpression:       typeExprFromString("[]models.Item"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   33,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]models.Item"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   33,
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
											TypeExpression:       typeExprFromString("[]models.Item"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Items",
											ReferenceLocation: ast_domain.Location{
												Line:   47,
												Column: 35,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   33,
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
												Line:   47,
												Column: 3,
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
											Expression: &ast_domain.Identifier{
												Name: "idx",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 2,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "idx",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 9,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("idx"),
													OriginalSourcePath: new("partials/child.pk"),
													Stringability:      1,
												},
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   47,
										Column: 3,
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
										Value: "loop-container",
										Location: ast_domain.Location{
											Line:   47,
											Column: 12,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 8,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "p-key",
										RawExpression: "item.Value",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "item",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.Item"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "item",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("item"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Value",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 6,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   42,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/item.go"),
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
													CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Value",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   42,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/item.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   47,
											Column: 71,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 63,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Value",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   42,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("item"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/item.go"),
											Stringability:       1,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   42,
											Column: 2,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "child_layout_1745aa65_data_item_value_index_idx_label_item_label_dc9a1179",
												PartialAlias:        "child",
												PartialPackageName:  "partials_child_d247007e",
												InvokerPackageAlias: "main_aaf9a2e0",
												Location: ast_domain.Location{
													Line:   48,
													Column: 4,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"data": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.Item"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 52,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Value",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 6,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 52,
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
																			CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Value",
																			ReferenceLocation: ast_domain.Location{
																				Line:   48,
																				Column: 52,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   42,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/item.go"),
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
																	CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   48,
																		Column: 52,
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
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Value",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 52,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   42,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/item.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   48,
															Column: 52,
														},
														GoFieldName: "Data",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 52,
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
																	CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Value",
																	ReferenceLocation: ast_domain.Location{
																		Line:   48,
																		Column: 52,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("item"),
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/item.go"),
															Stringability:       1,
														},
													},
													"index": ast_domain.PropValue{
														Expression: &ast_domain.Identifier{
															Name: "idx",
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
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "idx",
																	ReferenceLocation: ast_domain.Location{
																		Line:   48,
																		Column: 92,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "idx",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 92,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("idx"),
																},
																BaseCodeGenVarName: new("idx"),
																OriginalSourcePath: new("main.pk"),
																Stringability:      1,
															},
														},
														Location: ast_domain.Location{
															Line:   48,
															Column: 92,
														},
														GoFieldName: "Index",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "idx",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 92,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "idx",
																	ReferenceLocation: ast_domain.Location{
																		Line:   48,
																		Column: 92,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("idx"),
															},
															BaseCodeGenVarName: new("idx"),
															OriginalSourcePath: new("main.pk"),
															Stringability:      1,
														},
													},
													"label": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("models.Item"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 72,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Label",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 6,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 72,
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
																			CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Label",
																			ReferenceLocation: ast_domain.Location{
																				Line:   48,
																				Column: 72,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   43,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("models/item.go"),
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
																	CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   48,
																		Column: 72,
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
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Label",
																		ReferenceLocation: ast_domain.Location{
																			Line:   48,
																			Column: 72,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   43,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/item.go"),
																Stringability:       1,
															},
														},
														Location: ast_domain.Location{
															Line:   48,
															Column: 72,
														},
														GoFieldName: "Label",
														InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 72,
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
																	CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Label",
																	ReferenceLocation: ast_domain.Location{
																		Line:   48,
																		Column: 72,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   43,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("item"),
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/item.go"),
															Stringability:       1,
														},
													},
													IsLoopDependent: true,
													IsLoopDependent: true,
													IsLoopDependent: true,
												},
											},
											DynamicAttributeOrigins: map[string]string{
												"data":  "main_aaf9a2e0",
												"index": "main_aaf9a2e0",
												"label": "main_aaf9a2e0",
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   42,
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
														OriginalSourcePath: new("partials/child.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.Identifier{
														Name: "idx",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 2,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "idx",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 9,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("idx"),
															OriginalSourcePath: new("partials/child.pk"),
															Stringability:      1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   42,
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
													Literal: ":0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   42,
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
												Value: "child-item",
												Location: ast_domain.Location{
													Line:   42,
													Column: 14,
												},
												NameLocation: ast_domain.Location{
													Line:   42,
													Column: 7,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "id",
												Value: "loop-child",
												Location: ast_domain.Location{
													Line:   48,
													Column: 22,
												},
												NameLocation: ast_domain.Location{
													Line:   48,
													Column: 18,
												},
											},
										},
										DynamicAttributes: []ast_domain.DynamicAttribute{
											ast_domain.DynamicAttribute{
												Name:          "data",
												RawExpression: "item.Value",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.Item"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 52,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Value",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 6,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Value",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 52,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/item.go"),
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
															CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   48,
																Column: 52,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   42,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/item.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   48,
													Column: 52,
												},
												NameLocation: ast_domain.Location{
													Line:   48,
													Column: 45,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
															Column: 52,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   42,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/item.go"),
													Stringability:       1,
												},
											},
											ast_domain.DynamicAttribute{
												Name:          "index",
												RawExpression: "idx",
												Expression: &ast_domain.Identifier{
													Name: "idx",
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
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "idx",
															ReferenceLocation: ast_domain.Location{
																Line:   48,
																Column: 92,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("idx"),
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   48,
													Column: 92,
												},
												NameLocation: ast_domain.Location{
													Line:   48,
													Column: 84,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "idx",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
															Column: 92,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("idx"),
													OriginalSourcePath: new("main.pk"),
													Stringability:      1,
												},
											},
											ast_domain.DynamicAttribute{
												Name:          "label",
												RawExpression: "item.Label",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.Item"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 72,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Label",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 6,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Label",
																ReferenceLocation: ast_domain.Location{
																	Line:   48,
																	Column: 72,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   43,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/item.go"),
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
															CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   48,
																Column: 72,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/item.go"),
														Stringability:       1,
													},
												},
												Location: ast_domain.Location{
													Line:   48,
													Column: 72,
												},
												NameLocation: ast_domain.Location{
													Line:   48,
													Column: 64,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
															Column: 72,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/item.go"),
													Stringability:       1,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   43,
													Column: 3,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_child_d247007e"),
													OriginalSourcePath:   new("partials/child.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   43,
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
																OriginalSourcePath: new("partials/child.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "idx",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 2,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "idx",
																		ReferenceLocation: ast_domain.Location{
																			Line:   43,
																			Column: 9,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("idx"),
																	OriginalSourcePath: new("partials/child.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   43,
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
															Literal: ":0:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   43,
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
															Line:   43,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_child_d247007e"),
															OriginalSourcePath:   new("partials/child.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   43,
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
																		OriginalSourcePath: new("partials/child.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "idx",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 2,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "",
																				CanonicalPackagePath: "",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "idx",
																				ReferenceLocation: ast_domain.Location{
																					Line:   43,
																					Column: 9,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("idx"),
																			OriginalSourcePath: new("partials/child.pk"),
																			Stringability:      1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   43,
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
																	Literal: ":0:0:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   43,
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
																	Line:   43,
																	Column: 12,
																},
																RawExpression: "state.DisplayText",
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
																				CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/dist/partials/partials_child_d247007e",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   43,
																					Column: 12,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("partials_child_d247007eData_child_layout_1745aa65_data_item_value_index_idx_label_item_label_dc9a1179"),
																			OriginalSourcePath: new("partials/child.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "DisplayText",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_child_d247007e",
																				CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/dist/partials/partials_child_d247007e",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "DisplayText",
																				ReferenceLocation: ast_domain.Location{
																					Line:   43,
																					Column: 12,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   30,
																					Column: 23,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_data_item_value_index_idx_label_item_label_dc9a1179"),
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
																			CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/dist/partials/partials_child_d247007e",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "DisplayText",
																			ReferenceLocation: ast_domain.Location{
																				Line:   43,
																				Column: 12,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   30,
																				Column: 23,
																			},
																		},
																		BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_data_item_value_index_idx_label_item_label_dc9a1179"),
																		OriginalSourcePath:  new("partials/child.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "partials_child_d247007e",
																		CanonicalPackagePath: "testcase_53_loop_in_layout_with_prop_binding/dist/partials/partials_child_d247007e",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "DisplayText",
																		ReferenceLocation: ast_domain.Location{
																			Line:   43,
																			Column: 12,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   30,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_data_item_value_index_idx_label_item_label_dc9a1179"),
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
