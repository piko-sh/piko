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
							Line:   48,
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
						Value: "layout",
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
							Line:   41,
							Column: 2,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_child_d247007e"),
							OriginalSourcePath:   new("partials/child.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "child_layout_1745aa65_count_state_itemcount_items_state_getactiveitems_4b4f7123",
								PartialAlias:        "child",
								PartialPackageName:  "partials_child_d247007e",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   49,
									Column: 3,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"count": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
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
															CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 79,
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
													Name: "ItemCount",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ItemCount",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 79,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 19,
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
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ItemCount",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 79,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 19,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName: new("pageData"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   49,
											Column: 79,
										},
										GoFieldName: "Count",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName: new("pageData"),
											Stringability:      1,
										},
									},
									"items": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
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
															CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 47,
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
													Name: "GetActiveItems",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetActiveItems",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   34,
																Column: 19,
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
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "GetActiveItems",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
															Column: 19,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName: new("pageData"),
												Stringability:      5,
											},
										},
										Location: ast_domain.Location{
											Line:   49,
											Column: 47,
										},
										GoFieldName: "Items",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName: new("pageData"),
											Stringability:      5,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"count": "main_aaf9a2e0",
								"items": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   41,
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
									Line:   41,
									Column: 14,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 7,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "child",
								Location: ast_domain.Location{
									Line:   49,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 17,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "count",
								RawExpression: "state.ItemCount()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 79,
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
											Name: "ItemCount",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ItemCount",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 79,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 19,
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
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ItemCount",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 79,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 19,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Args: []ast_domain.Expression{},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
										},
										BaseCodeGenVarName: new("pageData"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   49,
									Column: 79,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 71,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
									},
									BaseCodeGenVarName: new("pageData"),
									Stringability:      1,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "items",
								RawExpression: "state.GetActiveItems()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 47,
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
											Name: "GetActiveItems",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "GetActiveItems",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   34,
														Column: 19,
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
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "GetActiveItems",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
													Column: 19,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Args: []ast_domain.Expression{},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
										},
										BaseCodeGenVarName: new("pageData"),
										Stringability:      5,
									},
								},
								Location: ast_domain.Location{
									Line:   49,
									Column: 47,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 39,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/pages/main_aaf9a2e0",
									},
									BaseCodeGenVarName: new("pageData"),
									Stringability:      5,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   42,
									Column: 3,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_child_d247007e"),
									OriginalSourcePath:   new("partials/child.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										OriginalSourcePath: new("partials/child.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   42,
											Column: 9,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   42,
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
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   42,
													Column: 9,
												},
												Literal: "Total: ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("partials/child.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   42,
													Column: 19,
												},
												RawExpression: "state.Total",
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
																CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/partials/partials_child_d247007e",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_child_d247007eData_child_layout_1745aa65_count_state_itemcount_items_state_getactiveitems_4b4f7123"),
															OriginalSourcePath: new("partials/child.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Total",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "partials_child_d247007e",
																CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/partials/partials_child_d247007e",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Total",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 19,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   29,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_count_state_itemcount_items_state_getactiveitems_4b4f7123"),
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
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_child_d247007e",
															CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/partials/partials_child_d247007e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Total",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_count_state_itemcount_items_state_getactiveitems_4b4f7123"),
														OriginalSourcePath:  new("partials/child.pk"),
														GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "partials_child_d247007e",
														CanonicalPackagePath: "testcase_56_method_call_in_prop_binding_in_layout/dist/partials/partials_child_d247007e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Total",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_child_d247007eData_child_layout_1745aa65_count_state_itemcount_items_state_getactiveitems_4b4f7123"),
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
	}
}()
