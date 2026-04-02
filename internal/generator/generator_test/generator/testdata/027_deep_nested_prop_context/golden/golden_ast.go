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
					OriginalPackageAlias: new("partials_parent_f5d0595c"),
					OriginalSourcePath:   new("partials/parent.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "parent_from_grandparent_state_data_valueforparent_e958c52a",
						PartialAlias:        "parent",
						PartialPackageName:  "partials_parent_f5d0595c",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"from-grandparent": ast_domain.PropValue{
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
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 62,
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
											Name: "Data",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("grandparent_types.GrandparentState"),
													PackageAlias:         "grandparent_types",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 62,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   36,
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
												TypeExpression:       typeExprFromString("grandparent_types.GrandparentState"),
												PackageAlias:         "grandparent_types",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 62,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   36,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ValueForParent",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "grandparent_types",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForParent",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 62,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "grandparent_types",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ValueForParent",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 62,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/grandparent_types/data.go"),
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
											PackageAlias:         "grandparent_types",
											CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ValueForParent",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 62,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   46,
												Column: 2,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "grandparent_types",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForParent",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 62,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 2,
												},
											},
											BaseCodeGenVarName: new("pageData"),
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("pkg/grandparent_types/data.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 62,
								},
								GoFieldName: "FromGrandparent",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "grandparent_types",
										CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ValueForParent",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 62,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   46,
											Column: 2,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "grandparent_types",
											CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ValueForParent",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 62,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   46,
												Column: 2,
											},
										},
										BaseCodeGenVarName: new("pageData"),
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("pkg/grandparent_types/data.go"),
									Stringability:       1,
								},
							},
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"from-grandparent": "pages_main_594861c5",
					},
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
						OriginalSourcePath: new("partials/parent.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "parent",
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
						Name:  "hello",
						Value: "world",
						Location: ast_domain.Location{
							Line:   22,
							Column: 36,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 29,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "from-grandparent",
						RawExpression: "state.Data.ValueForParent",
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
											CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "state",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 62,
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
									Name: "Data",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("grandparent_types.GrandparentState"),
											PackageAlias:         "grandparent_types",
											CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Data",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 62,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   36,
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
										TypeExpression:       typeExprFromString("grandparent_types.GrandparentState"),
										PackageAlias:         "grandparent_types",
										CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Data",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 62,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   36,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
								},
							},
							Property: &ast_domain.Identifier{
								Name: "ValueForParent",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 12,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "grandparent_types",
										CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ValueForParent",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 62,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   46,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("pkg/grandparent_types/data.go"),
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
									PackageAlias:         "grandparent_types",
									CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "ValueForParent",
									ReferenceLocation: ast_domain.Location{
										Line:   22,
										Column: 62,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   46,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("pages/main.pk"),
								GeneratedSourcePath: new("pkg/grandparent_types/data.go"),
								Stringability:       1,
							},
						},
						Location: ast_domain.Location{
							Line:   22,
							Column: 62,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 43,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression:       typeExprFromString("string"),
								PackageAlias:         "grandparent_types",
								CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
							},
							Symbol: &ast_domain.ResolvedSymbol{
								Name: "ValueForParent",
								ReferenceLocation: ast_domain.Location{
									Line:   22,
									Column: 62,
								},
								DeclarationLocation: ast_domain.Location{
									Line:   46,
									Column: 2,
								},
							},
							BaseCodeGenVarName:  new("pageData"),
							OriginalSourcePath:  new("pages/main.pk"),
							GeneratedSourcePath: new("pkg/grandparent_types/data.go"),
							Stringability:       1,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_child_d247007e"),
							OriginalSourcePath:   new("partials/child.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235",
								PartialAlias:        "child",
								PartialPackageName:  "partials_child_d247007e",
								InvokerPackageAlias: "partials_parent_f5d0595c",
								Location: ast_domain.Location{
									Line:   23,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"from-grandparent-state": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_parent_f5d0595c.Props"),
														PackageAlias:         "partials_parent_f5d0595c",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													OriginalSourcePath: new("partials/parent.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "FromGrandparent",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_parent_f5d0595c",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FromGrandparent",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   41,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_parent_f5d0595c",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FromGrandparent",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 32,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   41,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													},
													BaseCodeGenVarName:  new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													OriginalSourcePath:  new("partials/parent.pk"),
													GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
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
													PackageAlias:         "partials_parent_f5d0595c",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FromGrandparent",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   41,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_parent_f5d0595c",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FromGrandparent",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   41,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												},
												BaseCodeGenVarName:  new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												OriginalSourcePath:  new("partials/parent.pk"),
												GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   25,
											Column: 32,
										},
										GoFieldName: "FromGrandparentState",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_parent_f5d0595c",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FromGrandparent",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   41,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_parent_f5d0595c",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FromGrandparent",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   41,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											},
											BaseCodeGenVarName:  new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
											Stringability:       1,
										},
									},
									"from-parent-state": ast_domain.PropValue{
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
															TypeExpression:       typeExprFromString("partials_parent_f5d0595c.Response"),
															PackageAlias:         "partials_parent_f5d0595c",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
														OriginalSourcePath: new("partials/parent.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Data",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("parent_types.ParentState"),
															PackageAlias:         "parent_types",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   43,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
														OriginalSourcePath:  new("partials/parent.pk"),
														GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
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
														TypeExpression:       typeExprFromString("parent_types.ParentState"),
														PackageAlias:         "parent_types",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													OriginalSourcePath:  new("partials/parent.pk"),
													GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "ValueForChild",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "parent_types",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ValueForChild",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   51,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "parent_types",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ValueForChild",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   51,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													},
													BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													OriginalSourcePath:  new("partials/parent.pk"),
													GeneratedSourcePath: new("pkg/parent_types/data.go"),
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
													PackageAlias:         "parent_types",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ValueForChild",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 27,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   51,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "parent_types",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ValueForChild",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   51,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												},
												BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												OriginalSourcePath:  new("partials/parent.pk"),
												GeneratedSourcePath: new("pkg/parent_types/data.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 27,
										},
										GoFieldName: "FromParentState",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "parent_types",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForChild",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 27,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   51,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "parent_types",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ValueForChild",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 27,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   51,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("pkg/parent_types/data.go"),
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"from-grandparent-state": "partials_parent_f5d0595c",
								"from-parent-state":      "partials_parent_f5d0595c",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("partials/child.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "child",
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
								Name:          "from-grandparent-state",
								RawExpression: "props.FromGrandparent",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_parent_f5d0595c.Props"),
												PackageAlias:         "partials_parent_f5d0595c",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											OriginalSourcePath: new("partials/parent.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "FromGrandparent",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_parent_f5d0595c",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "FromGrandparent",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   41,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
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
											PackageAlias:         "partials_parent_f5d0595c",
											CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "FromGrandparent",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 32,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   41,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
										OriginalSourcePath:  new("partials/parent.pk"),
										GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   25,
									Column: 32,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_parent_f5d0595c",
										CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "FromGrandparent",
										ReferenceLocation: ast_domain.Location{
											Line:   25,
											Column: 32,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   41,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("props_parent_from_grandparent_state_data_valueforparent_e958c52a"),
									OriginalSourcePath:  new("partials/parent.pk"),
									GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
									Stringability:       1,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "from-parent-state",
								RawExpression: "state.Data.ValueForChild",
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
													TypeExpression:       typeExprFromString("partials_parent_f5d0595c.Response"),
													PackageAlias:         "partials_parent_f5d0595c",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 27,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												OriginalSourcePath: new("partials/parent.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Data",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("parent_types.ParentState"),
													PackageAlias:         "parent_types",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 27,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												OriginalSourcePath:  new("partials/parent.pk"),
												GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
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
												TypeExpression:       typeExprFromString("parent_types.ParentState"),
												PackageAlias:         "parent_types",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 27,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ValueForChild",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "parent_types",
												CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForChild",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 27,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   51,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("pkg/parent_types/data.go"),
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
											PackageAlias:         "parent_types",
											CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ValueForChild",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 27,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   51,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
										OriginalSourcePath:  new("partials/parent.pk"),
										GeneratedSourcePath: new("pkg/parent_types/data.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   24,
									Column: 27,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int"),
										PackageAlias:         "parent_types",
										CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ValueForChild",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 27,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   51,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
									OriginalSourcePath:  new("partials/parent.pk"),
									GeneratedSourcePath: new("pkg/parent_types/data.go"),
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
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_child_d247007e"),
									OriginalSourcePath:   new("partials/child.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										OriginalSourcePath: new("partials/child.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 8,
										},
										TextContent: "Received from parent state: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 8,
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
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   23,
											Column: 36,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 50,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 42,
											},
											RawExpression: "props.FromParentState",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_child_d247007e.Props"),
															PackageAlias:         "partials_child_d247007e",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 50,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
														OriginalSourcePath: new("partials/child.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FromParentState",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "partials_child_d247007e",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FromParentState",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 50,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   36,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "parent_types",
																CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ValueForChild",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   51,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
														},
														BaseCodeGenVarName:  new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
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
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FromParentState",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 50,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   36,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "parent_types",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ValueForChild",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   51,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
													},
													BaseCodeGenVarName:  new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
													OriginalSourcePath:  new("partials/child.pk"),
													GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "partials_child_d247007e",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FromParentState",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 50,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   36,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "parent_types",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/parent_types",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ValueForChild",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   51,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_from_grandparent_state_data_valueforparent_e958c52a"),
												},
												BaseCodeGenVarName:  new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
												OriginalSourcePath:  new("partials/child.pk"),
												GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 36,
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
									OriginalPackageAlias: new("partials_child_d247007e"),
									OriginalSourcePath:   new("partials/child.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
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
										OriginalSourcePath: new("partials/child.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 8,
										},
										TextContent: "Forwarded from grandparent state: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 8,
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
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 42,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_child_d247007e"),
											OriginalSourcePath:   new("partials/child.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   24,
												Column: 56,
											},
											NameLocation: ast_domain.Location{
												Line:   24,
												Column: 48,
											},
											RawExpression: "props.FromGrandparentState",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_child_d247007e.Props"),
															PackageAlias:         "partials_child_d247007e",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 56,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
														OriginalSourcePath: new("partials/child.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FromGrandparentState",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_child_d247007e",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FromGrandparentState",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 56,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "grandparent_types",
																CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ValueForParent",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 62,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
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
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FromGrandparentState",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 56,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "grandparent_types",
															CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ValueForParent",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 62,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
													OriginalSourcePath:  new("partials/child.pk"),
													GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_child_d247007e",
													CanonicalPackagePath: "testcase_027_deep_nested_prop_context/dist/partials/partials_child_d247007e",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FromGrandparentState",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 56,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "grandparent_types",
														CanonicalPackagePath: "testcase_027_deep_nested_prop_context/pkg/grandparent_types",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ValueForParent",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 62,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("props_child_parent_from_grandparent_state_data_valueforparent_e958c52a_from_grandparent_state_props_fromgrandparent_from_parent_state_state_data_valueforchild_35e9e235"),
												OriginalSourcePath:  new("partials/child.pk"),
												GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:1",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 42,
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
