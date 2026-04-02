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
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					IsStructurallyStatic: true,
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
						OriginalSourcePath: new("pages/main.pk"),
						Stringability:      1,
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   23,
							Column: 5,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   23,
									Column: 9,
								},
								TextContent: "Customer List",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 9,
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
							OriginalPackageAlias: new("partials_table_customers_3714da1b"),
							OriginalSourcePath:   new("partials/table_customers.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "customer_table_ccc87f7e",
								PartialAlias:        "customer_table",
								PartialPackageName:  "partials_table_customers_3714da1b",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
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
								OriginalSourcePath: new("partials/table_customers.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "table-wrapper",
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
								TagName: "table",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_table_customers_3714da1b"),
									OriginalSourcePath:   new("partials/table_customers.pk"),
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
										OriginalSourcePath: new("partials/table_customers.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 7,
										},
										TagName: "thead",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_table_customers_3714da1b"),
											OriginalSourcePath:   new("partials/table_customers.pk"),
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
												OriginalSourcePath: new("partials/table_customers.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   25,
													Column: 9,
												},
												TagName: "tr",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_table_customers_3714da1b"),
													OriginalSourcePath:   new("partials/table_customers.pk"),
												},
												DirIf: &ast_domain.Directive{
													Type: ast_domain.DirectiveIf,
													Location: ast_domain.Location{
														Line:   25,
														Column: 50,
													},
													NameLocation: ast_domain.Location{
														Line:   25,
														Column: 44,
													},
													RawExpression: "row.Header",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "row",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "row",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 50,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("row"),
																OriginalSourcePath: new("partials/table_customers.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Header",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 5,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("bool"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Header",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 50,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("row"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
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
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Header",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 50,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("row"),
															OriginalSourcePath:  new("partials/table_customers.pk"),
															GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "partials_table_customers_3714da1b",
															CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Header",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 50,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   61,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("row"),
														OriginalSourcePath:  new("partials/table_customers.pk"),
														GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
														Stringability:       1,
													},
												},
												DirFor: &ast_domain.Directive{
													Type: ast_domain.DirectiveFor,
													Location: ast_domain.Location{
														Line:   25,
														Column: 20,
													},
													NameLocation: ast_domain.Location{
														Line:   25,
														Column: 13,
													},
													RawExpression: "row in state.TableRows",
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
																OriginalSourcePath: new("partials/table_customers.pk"),
															},
														},
														ItemVariable: &ast_domain.Identifier{
															Name: "row",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "row",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("row"),
																OriginalSourcePath: new("partials/table_customers.pk"),
															},
														},
														Collection: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 8,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.Response"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
																	OriginalSourcePath: new("partials/table_customers.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "TableRows",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 14,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TableRows",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   63,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 8,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "TableRows",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   63,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TableRows",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
															OriginalSourcePath:  new("partials/table_customers.pk"),
															GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TableRows",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
															OriginalSourcePath:  new("partials/table_customers.pk"),
															GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
															PackageAlias:         "partials_table_customers_3714da1b",
															CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TableRows",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
														OriginalSourcePath:  new("partials/table_customers.pk"),
														GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   25,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/table_customers.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:0:0:0.",
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
																OriginalSourcePath: new("partials/table_customers.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "row",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "row",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("row"),
																	OriginalSourcePath: new("partials/table_customers.pk"),
																},
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/table_customers.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   26,
															Column: 11,
														},
														TagName: "th",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_table_customers_3714da1b"),
															OriginalSourcePath:   new("partials/table_customers.pk"),
														},
														DirFor: &ast_domain.Directive{
															Type: ast_domain.DirectiveFor,
															Location: ast_domain.Location{
																Line:   26,
																Column: 22,
															},
															NameLocation: ast_domain.Location{
																Line:   26,
																Column: 15,
															},
															RawExpression: "cell in row.TableCells",
															Expression: &ast_domain.ForInExpression{
																IndexVariable: &ast_domain.Identifier{
																	Name: "__pikoLoopIdx2",
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
																			Name: "__pikoLoopIdx2",
																			ReferenceLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("__pikoLoopIdx2"),
																		OriginalSourcePath: new("partials/table_customers.pk"),
																	},
																},
																ItemVariable: &ast_domain.Identifier{
																	Name: "cell",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableCell"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cell",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 22,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cell"),
																		OriginalSourcePath: new("partials/table_customers.pk"),
																	},
																},
																Collection: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "row",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 9,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "row",
																				ReferenceLocation: ast_domain.Location{
																					Line:   26,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("row"),
																			OriginalSourcePath: new("partials/table_customers.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "TableCells",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 13,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TableCells",
																				ReferenceLocation: ast_domain.Location{
																					Line:   26,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   60,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("row"),
																			OriginalSourcePath:  new("partials/table_customers.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
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
																			TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TableCells",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 22,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   60,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("row"),
																		OriginalSourcePath:  new("partials/table_customers.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																	},
																},
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TableCells",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   60,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TableCells",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   60,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "TableCells",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   60,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("row"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
															},
														},
														DirText: &ast_domain.Directive{
															Type: ast_domain.DirectiveText,
															Location: ast_domain.Location{
																Line:   26,
																Column: 54,
															},
															NameLocation: ast_domain.Location{
																Line:   26,
																Column: 46,
															},
															RawExpression: "cell.Text",
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "cell",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableCell"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cell",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 54,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cell"),
																		OriginalSourcePath: new("partials/table_customers.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Text",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Text",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 54,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   57,
																				Column: 24,
																			},
																		},
																		BaseCodeGenVarName:  new("cell"),
																		OriginalSourcePath:  new("partials/table_customers.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
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
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Text",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   57,
																			Column: 24,
																		},
																	},
																	BaseCodeGenVarName:  new("cell"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																	Stringability:       1,
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Text",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   57,
																		Column: 24,
																	},
																},
																BaseCodeGenVarName:  new("cell"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																Stringability:       1,
															},
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   26,
																		Column: 11,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:0:0:0.",
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
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "row",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "row",
																				ReferenceLocation: ast_domain.Location{
																					Line:   26,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("row"),
																			OriginalSourcePath: new("partials/table_customers.pk"),
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   26,
																		Column: 11,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Literal: ":0.",
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
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "cell",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableCell"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "cell",
																				ReferenceLocation: ast_domain.Location{
																					Line:   26,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("cell"),
																			OriginalSourcePath: new("partials/table_customers.pk"),
																		},
																	},
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 11,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/table_customers.pk"),
																Stringability:      1,
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
											Line:   29,
											Column: 7,
										},
										TagName: "tbody",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_table_customers_3714da1b"),
											OriginalSourcePath:   new("partials/table_customers.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   29,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/table_customers.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   30,
													Column: 9,
												},
												TagName: "tr",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_table_customers_3714da1b"),
													OriginalSourcePath:   new("partials/table_customers.pk"),
												},
												DirIf: &ast_domain.Directive{
													Type: ast_domain.DirectiveIf,
													Location: ast_domain.Location{
														Line:   30,
														Column: 50,
													},
													NameLocation: ast_domain.Location{
														Line:   30,
														Column: 44,
													},
													RawExpression: "!row.Header",
													Expression: &ast_domain.UnaryExpression{
														Operator: "!",
														Right: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "row",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 2,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "row",
																		ReferenceLocation: ast_domain.Location{
																			Line:   30,
																			Column: 50,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("row"),
																	OriginalSourcePath: new("partials/table_customers.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Header",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 6,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("bool"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Header",
																		ReferenceLocation: ast_domain.Location{
																			Line:   30,
																			Column: 50,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   61,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 2,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("bool"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Header",
																	ReferenceLocation: ast_domain.Location{
																		Line:   30,
																		Column: 50,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   61,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("row"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																Stringability:       1,
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("partials/table_customers.pk"),
															Stringability:      1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("partials/table_customers.pk"),
															Stringability:      1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/table_customers.pk"),
														Stringability:      1,
													},
												},
												DirFor: &ast_domain.Directive{
													Type: ast_domain.DirectiveFor,
													Location: ast_domain.Location{
														Line:   30,
														Column: 20,
													},
													NameLocation: ast_domain.Location{
														Line:   30,
														Column: 13,
													},
													RawExpression: "row in state.TableRows",
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
																OriginalSourcePath: new("partials/table_customers.pk"),
															},
														},
														ItemVariable: &ast_domain.Identifier{
															Name: "row",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "row",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("row"),
																OriginalSourcePath: new("partials/table_customers.pk"),
															},
														},
														Collection: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 8,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.Response"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   30,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
																	OriginalSourcePath: new("partials/table_customers.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "TableRows",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 14,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TableRows",
																		ReferenceLocation: ast_domain.Location{
																			Line:   30,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   63,
																			Column: 23,
																		},
																	},
																	BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 8,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "TableRows",
																	ReferenceLocation: ast_domain.Location{
																		Line:   30,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   63,
																		Column: 23,
																	},
																},
																BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TableRows",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
															OriginalSourcePath:  new("partials/table_customers.pk"),
															GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TableRows",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
															OriginalSourcePath:  new("partials/table_customers.pk"),
															GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
															PackageAlias:         "partials_table_customers_3714da1b",
															CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TableRows",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
														OriginalSourcePath:  new("partials/table_customers.pk"),
														GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   30,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/table_customers.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:0:1:0.",
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
																OriginalSourcePath: new("partials/table_customers.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "row",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "row",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("row"),
																	OriginalSourcePath: new("partials/table_customers.pk"),
																},
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/table_customers.pk"),
														Stringability:      1,
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   31,
															Column: 11,
														},
														TagName: "td",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_table_customers_3714da1b"),
															OriginalSourcePath:   new("partials/table_customers.pk"),
														},
														DirFor: &ast_domain.Directive{
															Type: ast_domain.DirectiveFor,
															Location: ast_domain.Location{
																Line:   31,
																Column: 22,
															},
															NameLocation: ast_domain.Location{
																Line:   31,
																Column: 15,
															},
															RawExpression: "cell in row.TableCells",
															Expression: &ast_domain.ForInExpression{
																IndexVariable: &ast_domain.Identifier{
																	Name: "__pikoLoopIdx2",
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
																			Name: "__pikoLoopIdx2",
																			ReferenceLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("__pikoLoopIdx2"),
																		OriginalSourcePath: new("partials/table_customers.pk"),
																	},
																},
																ItemVariable: &ast_domain.Identifier{
																	Name: "cell",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableCell"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cell",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 22,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cell"),
																		OriginalSourcePath: new("partials/table_customers.pk"),
																	},
																},
																Collection: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "row",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 9,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "row",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("row"),
																			OriginalSourcePath: new("partials/table_customers.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "TableCells",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 13,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "TableCells",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   60,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("row"),
																			OriginalSourcePath:  new("partials/table_customers.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
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
																			TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "TableCells",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 22,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   60,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("row"),
																		OriginalSourcePath:  new("partials/table_customers.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																	},
																},
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TableCells",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   60,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "TableCells",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   60,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableCell"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "TableCells",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   60,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("row"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
															},
														},
														DirText: &ast_domain.Directive{
															Type: ast_domain.DirectiveText,
															Location: ast_domain.Location{
																Line:   31,
																Column: 54,
															},
															NameLocation: ast_domain.Location{
																Line:   31,
																Column: 46,
															},
															RawExpression: "cell.Text",
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "cell",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableCell"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cell",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 54,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cell"),
																		OriginalSourcePath: new("partials/table_customers.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Text",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "partials_table_customers_3714da1b",
																			CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Text",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 54,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   57,
																				Column: 24,
																			},
																		},
																		BaseCodeGenVarName:  new("cell"),
																		OriginalSourcePath:  new("partials/table_customers.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
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
																		PackageAlias:         "partials_table_customers_3714da1b",
																		CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Text",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   57,
																			Column: 24,
																		},
																	},
																	BaseCodeGenVarName:  new("cell"),
																	OriginalSourcePath:  new("partials/table_customers.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																	Stringability:       1,
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "partials_table_customers_3714da1b",
																	CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Text",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   57,
																		Column: 24,
																	},
																},
																BaseCodeGenVarName:  new("cell"),
																OriginalSourcePath:  new("partials/table_customers.pk"),
																GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
																Stringability:       1,
															},
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   31,
																		Column: 11,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:0:1:0.",
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
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "row",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableRow"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "row",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("row"),
																			OriginalSourcePath: new("partials/table_customers.pk"),
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   31,
																		Column: 11,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Literal: ":0.",
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
																		OriginalSourcePath: new("partials/table_customers.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "cell",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.TableCell"),
																				PackageAlias:         "partials_table_customers_3714da1b",
																				CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "cell",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 22,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("cell"),
																			OriginalSourcePath: new("partials/table_customers.pk"),
																		},
																	},
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   31,
																Column: 11,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/table_customers.pk"),
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
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   35,
									Column: 5,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_table_customers_3714da1b"),
									OriginalSourcePath:   new("partials/table_customers.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   35,
										Column: 14,
									},
									NameLocation: ast_domain.Location{
										Line:   35,
										Column: 8,
									},
									RawExpression: "len(state.TableRows) <= 1",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.CallExpression{
											Callee: &ast_domain.Identifier{
												Name: "len",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("builtin_function"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "len",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 14,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("len"),
													OriginalSourcePath: new("partials/table_customers.pk"),
												},
											},
											Args: []ast_domain.Expression{
												&ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 5,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_table_customers_3714da1b.Response"),
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 14,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
															OriginalSourcePath: new("partials/table_customers.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "TableRows",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 11,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
																PackageAlias:         "partials_table_customers_3714da1b",
																CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TableRows",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 14,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
															OriginalSourcePath:  new("partials/table_customers.pk"),
															GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]partials_table_customers_3714da1b.TableRow"),
															PackageAlias:         "partials_table_customers_3714da1b",
															CanonicalPackagePath: "testcase_015_table_with_mock_dal/dist/partials/partials_table_customers_3714da1b",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TableRows",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 14,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("partials_table_customers_3714da1bData_customer_table_ccc87f7e"),
														OriginalSourcePath:  new("partials/table_customers.pk"),
														GeneratedSourcePath: new("dist/partials/partials_table_customers_3714da1b/generated.go"),
													},
												},
											},
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
												Stringability: 1,
											},
										},
										Operator: "<=",
										Right: &ast_domain.IntegerLiteral{
											Value: 1,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 25,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int64"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/table_customers.pk"),
												Stringability:      1,
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/table_customers.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/table_customers.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
									RelativeLocation: ast_domain.Location{
										Line:   35,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/table_customers.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   35,
											Column: 41,
										},
										TextContent: "No customers found.",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_table_customers_3714da1b"),
											OriginalSourcePath:   new("partials/table_customers.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   35,
												Column: 41,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/table_customers.pk"),
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
