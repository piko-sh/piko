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
						InvocationKey:       "parent_9f11ff87",
						PartialAlias:        "parent",
						PartialPackageName:  "partials_parent_f5d0595c",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
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
						Value: "parent-wrapper",
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
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_child_d247007e"),
							OriginalSourcePath:   new("partials/child.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28",
								PartialAlias:        "child",
								PartialPackageName:  "partials_child_d247007e",
								InvokerPackageAlias: "partials_parent_f5d0595c",
								Location: ast_domain.Location{
									Line:   23,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"forwarded-prop": ast_domain.PropValue{
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
															CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_parent_f5d0595c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
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
															TypeExpression:       typeExprFromString("data_a.ParentData"),
															PackageAlias:         "data_a",
															CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
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
														TypeExpression:       typeExprFromString("data_a.ParentData"),
														PackageAlias:         "data_a",
														CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
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
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "data_a",
														CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ValueForChild",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   48,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "data_a",
															CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ValueForChild",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 47,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   48,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
													},
													BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
													OriginalSourcePath:  new("partials/parent.pk"),
													GeneratedSourcePath: new("pkg/data_a/data.go"),
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
													PackageAlias:         "data_a",
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ValueForChild",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "data_a",
														CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ValueForChild",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 47,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   48,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
												},
												BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
												OriginalSourcePath:  new("partials/parent.pk"),
												GeneratedSourcePath: new("pkg/data_a/data.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   23,
											Column: 47,
										},
										GoFieldName: "ForwardedProp",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "data_a",
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForChild",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "data_a",
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ValueForChild",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("pkg/data_a/data.go"),
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"data-received":  "partials_child_d247007e",
								"forwarded-prop": "partials_parent_f5d0595c",
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
								Value: "child-component",
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
								Name:          "data-received",
								RawExpression: "props.ForwardedProp",
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
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_child_d247007e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 48,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
											OriginalSourcePath: new("partials/child.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ForwardedProp",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_child_d247007e",
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_child_d247007e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ForwardedProp",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 48,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   35,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "data_a",
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ValueForChild",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
											},
											BaseCodeGenVarName:  new("props_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
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
											CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_child_d247007e",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ForwardedProp",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 48,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   35,
												Column: 2,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "data_a",
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForChild",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
													Column: 2,
												},
											},
											BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
										},
										BaseCodeGenVarName:  new("props_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
										OriginalSourcePath:  new("partials/child.pk"),
										GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 48,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_child_d247007e",
										CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_child_d247007e",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ForwardedProp",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 48,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   35,
											Column: 2,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "data_a",
											CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ValueForChild",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 47,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   48,
												Column: 2,
											},
										},
										BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
									},
									BaseCodeGenVarName:  new("props_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
									OriginalSourcePath:  new("partials/child.pk"),
									GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
									Stringability:       1,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "forwarded-prop",
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
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_9f11ff87"),
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
													TypeExpression:       typeExprFromString("data_a.ParentData"),
													PackageAlias:         "data_a",
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 47,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
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
												TypeExpression:       typeExprFromString("data_a.ParentData"),
												PackageAlias:         "data_a",
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
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
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "data_a",
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ValueForChild",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("pkg/data_a/data.go"),
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
											PackageAlias:         "data_a",
											CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ValueForChild",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 47,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   48,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
										OriginalSourcePath:  new("partials/parent.pk"),
										GeneratedSourcePath: new("pkg/data_a/data.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   23,
									Column: 47,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 30,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "data_a",
										CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_a",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ValueForChild",
										ReferenceLocation: ast_domain.Location{
											Line:   23,
											Column: 47,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   48,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_9f11ff87"),
									OriginalSourcePath:  new("partials/parent.pk"),
									GeneratedSourcePath: new("pkg/data_a/data.go"),
									Stringability:       1,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 69,
								},
								TextContent: " Child's own state: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_child_d247007e"),
									OriginalSourcePath:   new("partials/child.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 69,
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
									Column: 24,
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
										Column: 38,
									},
									NameLocation: ast_domain.Location{
										Line:   23,
										Column: 30,
									},
									RawExpression: "state.OwnData.InternalValue",
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
														TypeExpression:       typeExprFromString("partials_child_d247007e.Response"),
														PackageAlias:         "partials_child_d247007e",
														CanonicalPackagePath: "testcase_026_cross_package_context_confusion/dist/partials/partials_child_d247007e",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_child_d247007eData_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
													OriginalSourcePath: new("partials/child.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "OwnData",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("data_b.ChildData"),
														PackageAlias:         "data_b",
														CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_b",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OwnData",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("partials_child_d247007eData_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
													OriginalSourcePath:  new("partials/child.pk"),
													GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
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
													TypeExpression:       typeExprFromString("data_b.ChildData"),
													PackageAlias:         "data_b",
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "OwnData",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 38,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_child_d247007eData_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
												OriginalSourcePath:  new("partials/child.pk"),
												GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "InternalValue",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 15,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "data_b",
													CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "InternalValue",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 38,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   48,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_child_d247007eData_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
												OriginalSourcePath:  new("partials/child.pk"),
												GeneratedSourcePath: new("pkg/data_b/data.go"),
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
												PackageAlias:         "data_b",
												CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_b",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "InternalValue",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 38,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   48,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_child_d247007eData_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
											OriginalSourcePath:  new("partials/child.pk"),
											GeneratedSourcePath: new("pkg/data_b/data.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int"),
											PackageAlias:         "data_b",
											CanonicalPackagePath: "testcase_026_cross_package_context_confusion/pkg/data_b",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "InternalValue",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 38,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   48,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("partials_child_d247007eData_child_parent_9f11ff87_forwarded_prop_state_data_valueforchild_c8a6ed28"),
										OriginalSourcePath:  new("partials/child.pk"),
										GeneratedSourcePath: new("pkg/data_b/data.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 24,
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
	}
}()
