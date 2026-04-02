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
				NodeType: ast_domain.NodeFragment,
				Location: ast_domain.Location{
					Line:   0,
					Column: 0,
				},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "my_layout_place_world_question_state_question_dd7a1ef4",
						PartialAlias:        "my_layout",
						PartialPackageName:  "partials_layout_ee037d9a",
						InvokerPackageAlias: "pages_main_594861c5",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"place": ast_domain.PropValue{
								Expression: &ast_domain.StringLiteral{
									Value: "world",
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
									Line:   22,
									Column: 39,
								},
								GoFieldName: "Place",
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
							"question": ast_domain.PropValue{
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
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 64,
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
										Name: "Question",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Question",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 64,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Question",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 64,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 23,
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
											CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Question",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 64,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Question",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 64,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 23,
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
									Line:   22,
									Column: 64,
								},
								GoFieldName: "Question",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Question",
										ReferenceLocation: ast_domain.Location{
											Line:   22,
											Column: 64,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   37,
											Column: 23,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Question",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 64,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 23,
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
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
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
						OriginalSourcePath: new("partials/layout.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "place",
						Value: "world",
						Location: ast_domain.Location{
							Line:   22,
							Column: 39,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 32,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   22,
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
								Name:  "p-fragment",
								Value: "my_layout_place_world_server_question_state_question_050243c2",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "0",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 10,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 10,
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
											Line:   22,
											Column: 13,
										},
										RawExpression: "state.Hello",
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
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Hello",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Hello",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   36,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Hello",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 13,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   36,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
												OriginalSourcePath:  new("partials/layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_layout_ee037d9a",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Hello",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 13,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   36,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
											OriginalSourcePath:  new("partials/layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
											Stringability:       1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   22,
											Column: 27,
										},
										Literal: " ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("partials/layout.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   22,
											Column: 31,
										},
										RawExpression: "state.Place",
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
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Place",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Place",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Place",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 31,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
												OriginalSourcePath:  new("partials/layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_layout_ee037d9a",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Place",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 31,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
											OriginalSourcePath:  new("partials/layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
											Stringability:       1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   22,
											Column: 45,
										},
										Literal: ", ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("partials/layout.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   22,
											Column: 50,
										},
										RawExpression: "state.Question",
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
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 50,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Question",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Question",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 50,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Question",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 50,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
												OriginalSourcePath:  new("partials/layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_layout_ee037d9a",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Question",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 50,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   39,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
											OriginalSourcePath:  new("partials/layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
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
							Line:   23,
							Column: 5,
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
								Line:   23,
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
								Name:  "p-fragment",
								Value: "my_layout_place_world_server_question_state_question_050243c2",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "1",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
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
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "container",
										Location: ast_domain.Location{
											Line:   23,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 10,
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
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 10,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 10,
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
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: true,
														Location: ast_domain.Location{
															Line:   24,
															Column: 10,
														},
														Literal: "The question was: ",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   24,
															Column: 31,
														},
														RawExpression: "state.Question",
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
																		CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 31,
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
																Name: "Question",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Question",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 31,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   37,
																			Column: 23,
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
																	CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Question",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   37,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Question",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   24,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_layout_ee037d9a"),
							OriginalSourcePath:   new("partials/layout.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
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
								OriginalSourcePath: new("partials/layout.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "my_layout_place_world_server_question_state_question_050243c2",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment-id",
								Value: "2",
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								NameLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   24,
									Column: 10,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_layout_ee037d9a"),
									OriginalSourcePath:   new("partials/layout.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 10,
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
											Line:   24,
											Column: 13,
										},
										RawExpression: "state.Goodbye",
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
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Goodbye",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Goodbye",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Goodbye",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 13,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
												OriginalSourcePath:  new("partials/layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_layout_ee037d9a",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Goodbye",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 13,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
											OriginalSourcePath:  new("partials/layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
											Stringability:       1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   24,
											Column: 29,
										},
										Literal: " ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("partials/layout.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   24,
											Column: 33,
										},
										RawExpression: "state.Place",
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
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 33,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Place",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Place",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 33,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Place",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 33,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
												OriginalSourcePath:  new("partials/layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_layout_ee037d9a",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Place",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 33,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
											OriginalSourcePath:  new("partials/layout.pk"),
											GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
											Stringability:       1,
										},
									},
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   24,
											Column: 47,
										},
										Literal: ", ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("partials/layout.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   24,
											Column: 52,
										},
										RawExpression: "state.Question",
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
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 52,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
													OriginalSourcePath: new("partials/layout.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Question",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_layout_ee037d9a",
														CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Question",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 52,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
													CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Question",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 52,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
												OriginalSourcePath:  new("partials/layout.pk"),
												GeneratedSourcePath: new("dist/partials/partials_layout_ee037d9a/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_layout_ee037d9a",
												CanonicalPackagePath: "testcase_045_multiple_root_nodes/dist/partials/partials_layout_ee037d9a",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Question",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 52,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   39,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("partials_layout_ee037d9aData_my_layout_place_world_question_state_question_dd7a1ef4"),
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
	}
}()
