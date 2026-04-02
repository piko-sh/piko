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
					Line:   39,
					Column: 5,
				},
				TagName: "form",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   39,
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
							Line:   40,
							Column: 9,
						},
						TagName: "button",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							NeedsCSRF:            true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   40,
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
								Name:  "data-pk-action-method",
								Value: "POST",
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
								Name:  "id",
								Value: "btn-mismatch",
								Location: ast_domain.Location{
									Line:   40,
									Column: 21,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 17,
								},
							},
						},
						OnEvents: map[string][]ast_domain.Directive{
							"click": []ast_domain.Directive{
								ast_domain.Directive{
									Type: ast_domain.DirectiveOn,
									Location: ast_domain.Location{
										Line:   40,
										Column: 47,
									},
									NameLocation: ast_domain.Location{
										Line:   40,
										Column: 35,
									},
									Arg:           "click",
									Modifier:      "action",
									RawExpression: "action.actions.UpdateUser({name: state.Name, age: state.BadAge})",
									Expression: &ast_domain.CallExpression{
										Callee: &ast_domain.Identifier{
											Name: "actions.UpdateUser",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										Args: []ast_domain.Expression{
											&ast_domain.ObjectLiteral{
												Pairs: map[string]ast_domain.Expression{
													"age": &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 51,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("main_aaf9a2e0.State"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_095_action_object_type_mismatch/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
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
															Name: "BadAge",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 57,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_095_action_object_type_mismatch/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "BadAge",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   27,
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
															Column: 51,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_095_action_object_type_mismatch/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BadAge",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   27,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       1,
														},
													},
													"name": &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 34,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("main_aaf9a2e0.State"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_095_action_object_type_mismatch/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
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
															Name: "Name",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 40,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_095_action_object_type_mismatch/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   40,
																		Column: 47,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   26,
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
															Column: 34,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_095_action_object_type_mismatch/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 47,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   26,
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
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 27,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("map[string]string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("main.pk"),
												},
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("any"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
									},
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   40,
									Column: 113,
								},
								TextContent: "Update",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   40,
										Column: 113,
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
				},
			},
		},
	}
}()
