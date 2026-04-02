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
						InvocationKey:       "parent_partial_29dfea8a",
						PartialAlias:        "parent_partial",
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
							OriginalPackageAlias: new("partials_parent_f5d0595c"),
							OriginalSourcePath:   new("partials/parent.pk"),
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
								OriginalSourcePath: new("partials/parent.pk"),
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
								TextContent: "Parent says: ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_parent_f5d0595c"),
									OriginalSourcePath:   new("partials/parent.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										OriginalSourcePath: new("partials/parent.pk"),
										Stringability:      1,
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   23,
									Column: 21,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_parent_f5d0595c"),
									OriginalSourcePath:   new("partials/parent.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   23,
										Column: 35,
									},
									NameLocation: ast_domain.Location{
										Line:   23,
										Column: 27,
									},
									RawExpression: "state.ParentMessage",
									Expression: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
												OriginalSourcePath: new("partials/parent.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "ParentMessage",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_parent_f5d0595c",
													CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ParentMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
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
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ParentMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_parent_f5d0595c",
											CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ParentMessage",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 35,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
										OriginalSourcePath:  new("partials/parent.pk"),
										GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 21,
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
							OriginalPackageAlias: new("partials_child_d247007e"),
							OriginalSourcePath:   new("partials/child.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d",
								PartialAlias:        "child_partial",
								PartialPackageName:  "partials_child_d247007e",
								InvokerPackageAlias: "partials_parent_f5d0595c",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"label": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
													OriginalSourcePath: new("partials/parent.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "ParentMessage",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_parent_f5d0595c",
														CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ParentMessage",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "partials_parent_f5d0595c",
															CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ParentMessage",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 46,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
													},
													BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
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
													CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ParentMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 46,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_parent_f5d0595c",
														CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ParentMessage",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 46,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
												},
												BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
												OriginalSourcePath:  new("partials/parent.pk"),
												GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 46,
										},
										GoFieldName: "Label",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_parent_f5d0595c",
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ParentMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_parent_f5d0595c",
													CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ParentMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 46,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
											OriginalSourcePath:  new("partials/parent.pk"),
											GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
											Stringability:       1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"data-child-message":     "partials_child_d247007e",
								"data-label-from-parent": "partials_child_d247007e",
								"label":                  "partials_parent_f5d0595c",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								Name:          "data-child-message",
								RawExpression: "state.ChildMessage",
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
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 43,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("partials_child_d247007eData_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
											OriginalSourcePath: new("partials/child.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ChildMessage",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_child_d247007e",
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ChildMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 43,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_child_d247007eData_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
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
											CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ChildMessage",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 43,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   34,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("partials_child_d247007eData_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
										OriginalSourcePath:  new("partials/child.pk"),
										GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 43,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 22,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_child_d247007e",
										CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ChildMessage",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 43,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   34,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("partials_child_d247007eData_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
									OriginalSourcePath:  new("partials/child.pk"),
									GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
									Stringability:       1,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "data-label-from-parent",
								RawExpression: "props.Label",
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
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 88,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
											OriginalSourcePath: new("partials/child.pk"),
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
												PackageAlias:         "partials_child_d247007e",
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Label",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 88,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   32,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "partials_parent_f5d0595c",
													CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ParentMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 46,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
											},
											BaseCodeGenVarName:  new("props_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
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
											CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Label",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 88,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   32,
												Column: 2,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_parent_f5d0595c",
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ParentMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
										},
										BaseCodeGenVarName:  new("props_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
										OriginalSourcePath:  new("partials/child.pk"),
										GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 88,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 63,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_child_d247007e",
										CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_child_d247007e",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Label",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 88,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   32,
											Column: 2,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_parent_f5d0595c",
											CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ParentMessage",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 46,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
											},
										},
										BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
									},
									BaseCodeGenVarName:  new("props_child_partial_parent_partial_29dfea8a_label_state_parentmessage_490ff31d"),
									OriginalSourcePath:  new("partials/child.pk"),
									GeneratedSourcePath: new("dist/partials/partials_child_d247007e/generated.go"),
									Stringability:       1,
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "label",
								RawExpression: "state.ParentMessage",
								Expression: &ast_domain.MemberExpression{
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
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
											OriginalSourcePath: new("partials/parent.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ParentMessage",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_parent_f5d0595c",
												CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ParentMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 46,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
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
											CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ParentMessage",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 46,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
										OriginalSourcePath:  new("partials/parent.pk"),
										GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   24,
									Column: 46,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 38,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_parent_f5d0595c",
										CanonicalPackagePath: "testcase_024_partial_with_forwarded_and_self_referential_attrs/dist/partials/partials_parent_f5d0595c",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "ParentMessage",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 46,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("partials_parent_f5d0595cData_parent_partial_29dfea8a"),
									OriginalSourcePath:  new("partials/parent.pk"),
									GeneratedSourcePath: new("dist/partials/partials_parent_f5d0595c/generated.go"),
									Stringability:       1,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 101,
								},
								TextContent: " Child content here. ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_child_d247007e"),
									OriginalSourcePath:   new("partials/child.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 101,
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
