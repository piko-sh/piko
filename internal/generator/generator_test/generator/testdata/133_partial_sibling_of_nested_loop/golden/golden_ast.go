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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "container",
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
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
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
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 7,
								},
								TagName: "tbody",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 9,
										},
										TagName: "tr",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
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
											RawExpression: "row in state.Rows",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: nil,
												ItemVariable: &ast_domain.Identifier{
													Name: "row",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "row",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("row"),
														OriginalSourcePath: new("pages/main.pk"),
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
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
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Rows",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 14,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.Row"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Rows",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   61,
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
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.Row"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Rows",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   61,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.Row"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Rows",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   61,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.Row"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Rows",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   61,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.Row"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Rows",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   61,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   25,
												Column: 46,
											},
											NameLocation: ast_domain.Location{
												Line:   25,
												Column: 39,
											},
											RawExpression: "row.ID",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "row",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "row",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 28,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("row"),
														OriginalSourcePath: new("partials/row_actions.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "ID",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 46,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   54,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("row"),
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
														CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("row"),
													OriginalSourcePath:  new("partials/row_actions.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 46,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   54,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("row"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
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
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
													Literal: "r.0:0:0:0.",
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
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "row",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "row",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 28,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("row"),
																OriginalSourcePath: new("partials/row_actions.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 5,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   25,
																		Column: 46,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   54,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("row"),
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
																CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 28,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   40,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("row"),
															OriginalSourcePath:  new("partials/row_actions.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       1,
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "row-wrapper",
												Location: ast_domain.Location{
													Line:   25,
													Column: 61,
												},
												NameLocation: ast_domain.Location{
													Line:   25,
													Column: 54,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   26,
													Column: 11,
												},
												TagName: "td",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
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
													RawExpression: "(idx, cell) in row.Cells",
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
																		Line:   1,
																		Column: 2,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("idx"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														ItemVariable: &ast_domain.Identifier{
															Name: "cell",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Cell"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "cell",
																	ReferenceLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("cell"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Collection: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "row",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 16,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
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
																	OriginalSourcePath: new("pages/main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Cells",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 20,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]pages_main_594861c5.Cell"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Cells",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   56,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
																	TypeExpression:       typeExprFromString("[]pages_main_594861c5.Cell"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Cells",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   56,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("row"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.Cell"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Cells",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 22,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   56,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("row"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.Cell"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Cells",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 22,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   56,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("row"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.Cell"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Cells",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 22,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   56,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("row"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												DirKey: &ast_domain.Directive{
													Type: ast_domain.DirectiveKey,
													Location: ast_domain.Location{
														Line:   26,
														Column: 55,
													},
													NameLocation: ast_domain.Location{
														Line:   26,
														Column: 48,
													},
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
																	Line:   27,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("idx"),
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
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
																Line:   26,
																Column: 55,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("idx"),
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
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
																OriginalSourcePath: new("pages/main.pk"),
																Stringability:      1,
															},
															Literal: "r.0:0:0:0.",
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
																OriginalSourcePath: new("pages/main.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "row",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "row",
																			ReferenceLocation: ast_domain.Location{
																				Line:   25,
																				Column: 28,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("row"),
																		OriginalSourcePath: new("partials/row_actions.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 5,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   25,
																				Column: 46,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   54,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("row"),
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
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 28,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   40,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/row_actions.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
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
																OriginalSourcePath: new("pages/main.pk"),
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
																OriginalSourcePath: new("pages/main.pk"),
																Stringability:      1,
															},
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
																			Line:   27,
																			Column: 13,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("idx"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
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
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "data-cell",
														Location: ast_domain.Location{
															Line:   26,
															Column: 67,
														},
														NameLocation: ast_domain.Location{
															Line:   26,
															Column: 60,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   27,
															Column: 13,
														},
														TagName: "span",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("pages_main_594861c5"),
															OriginalSourcePath:   new("pages/main.pk"),
														},
														DirText: &ast_domain.Directive{
															Type: ast_domain.DirectiveText,
															Location: ast_domain.Location{
																Line:   27,
																Column: 27,
															},
															NameLocation: ast_domain.Location{
																Line:   27,
																Column: 19,
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
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Cell"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "cell",
																			ReferenceLocation: ast_domain.Location{
																				Line:   27,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("cell"),
																		OriginalSourcePath: new("pages/main.pk"),
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
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Text",
																			ReferenceLocation: ast_domain.Location{
																				Line:   27,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   52,
																				Column: 19,
																			},
																		},
																		BaseCodeGenVarName:  new("cell"),
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
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Text",
																		ReferenceLocation: ast_domain.Location{
																			Line:   27,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   52,
																			Column: 19,
																		},
																	},
																	BaseCodeGenVarName:  new("cell"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Text",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   52,
																		Column: 19,
																	},
																},
																BaseCodeGenVarName:  new("cell"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																Stringability:       1,
															},
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 13,
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
																	Literal: "r.0:0:0:0.",
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
																		OriginalSourcePath: new("pages/main.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "row",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "row",
																					ReferenceLocation: ast_domain.Location{
																						Line:   25,
																						Column: 28,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("row"),
																				OriginalSourcePath: new("partials/row_actions.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 5,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   25,
																						Column: 46,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   54,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("row"),
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
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   25,
																					Column: 28,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   40,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("row"),
																			OriginalSourcePath:  new("partials/row_actions.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 13,
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
																		OriginalSourcePath: new("pages/main.pk"),
																		Stringability:      1,
																	},
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
																					Line:   27,
																					Column: 13,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("idx"),
																			OriginalSourcePath: new("pages/main.pk"),
																			Stringability:      1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 13,
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
																	Literal: ":0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 13,
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
													Line:   29,
													Column: 11,
												},
												TagName: "td",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   29,
																Column: 11,
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
															Literal: "r.0:0:0:0.",
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
																OriginalSourcePath: new("pages/main.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "row",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "row",
																			ReferenceLocation: ast_domain.Location{
																				Line:   25,
																				Column: 28,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("row"),
																		OriginalSourcePath: new("partials/row_actions.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 5,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   25,
																				Column: 46,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   54,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("row"),
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
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   25,
																			Column: 28,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   40,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("partials/row_actions.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   29,
																Column: 11,
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
															Literal: ":1",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   29,
														Column: 11,
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
														Value: "actions",
														Location: ast_domain.Location{
															Line:   29,
															Column: 22,
														},
														NameLocation: ast_domain.Location{
															Line:   29,
															Column: 15,
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
															OriginalPackageAlias: new("partials_row_actions_c2843091"),
															OriginalSourcePath:   new("partials/row_actions.pk"),
															PartialInfo: &ast_domain.PartialInvocationInfo{
																InvocationKey:       "row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af",
																PartialAlias:        "row_actions",
																PartialPackageName:  "partials_row_actions_c2843091",
																InvokerPackageAlias: "pages_main_594861c5",
																Location: ast_domain.Location{
																	Line:   30,
																	Column: 13,
																},
																PassedProps: map[string]ast_domain.PropValue{
																	"category_name": ast_domain.PropValue{
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
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "state",
																						ReferenceLocation: ast_domain.Location{
																							Line:   32,
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
																				Name: "CategoryName",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "CategoryName",
																						ReferenceLocation: ast_domain.Location{
																							Line:   32,
																							Column: 31,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   60,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "CategoryName",
																							ReferenceLocation: ast_domain.Location{
																								Line:   32,
																								Column: 31,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   60,
																								Column: 2,
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
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "CategoryName",
																					ReferenceLocation: ast_domain.Location{
																						Line:   32,
																						Column: 31,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   60,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "CategoryName",
																						ReferenceLocation: ast_domain.Location{
																							Line:   32,
																							Column: 31,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   60,
																							Column: 2,
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
																			Line:   32,
																			Column: 31,
																		},
																		GoFieldName: "CategoryName",
																		InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "CategoryName",
																				ReferenceLocation: ast_domain.Location{
																					Line:   32,
																					Column: 31,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   60,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "CategoryName",
																					ReferenceLocation: ast_domain.Location{
																						Line:   32,
																						Column: 31,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   60,
																						Column: 2,
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
																	"container_id": ast_domain.PropValue{
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
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "state",
																						ReferenceLocation: ast_domain.Location{
																							Line:   31,
																							Column: 30,
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
																				Name: "ContainerID",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ContainerID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   31,
																							Column: 30,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   59,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "ContainerID",
																							ReferenceLocation: ast_domain.Location{
																								Line:   31,
																								Column: 30,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   59,
																								Column: 2,
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
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ContainerID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   31,
																						Column: 30,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   59,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ContainerID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   31,
																							Column: 30,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   59,
																							Column: 2,
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
																			Line:   31,
																			Column: 30,
																		},
																		GoFieldName: "ContainerID",
																		InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ContainerID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   59,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ContainerID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   31,
																						Column: 30,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   59,
																						Column: 2,
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
																	"row_id": ast_domain.PropValue{
																		Expression: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "row",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "row",
																						ReferenceLocation: ast_domain.Location{
																							Line:   33,
																							Column: 24,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("row"),
																					OriginalSourcePath: new("pages/main.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "ID",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 5,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   33,
																							Column: 24,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   54,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "ID",
																							ReferenceLocation: ast_domain.Location{
																								Line:   33,
																								Column: 24,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   54,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName: new("row"),
																					},
																					BaseCodeGenVarName:  new("row"),
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
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   33,
																						Column: 24,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   54,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   33,
																							Column: 24,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   54,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName: new("row"),
																				},
																				BaseCodeGenVarName:  new("row"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																				Stringability:       1,
																			},
																		},
																		Location: ast_domain.Location{
																			Line:   33,
																			Column: 24,
																		},
																		GoFieldName: "RowID",
																		InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 24,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   54,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   33,
																						Column: 24,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   54,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName: new("row"),
																			},
																			BaseCodeGenVarName:  new("row"),
																			OriginalSourcePath:  new("pages/main.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																	"row_title": ast_domain.PropValue{
																		Expression: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "row",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "row",
																						ReferenceLocation: ast_domain.Location{
																							Line:   34,
																							Column: 27,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("row"),
																					OriginalSourcePath: new("pages/main.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Title",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 5,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Title",
																						ReferenceLocation: ast_domain.Location{
																							Line:   34,
																							Column: 27,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   55,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "Title",
																							ReferenceLocation: ast_domain.Location{
																								Line:   34,
																								Column: 27,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   55,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName: new("row"),
																					},
																					BaseCodeGenVarName:  new("row"),
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
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Title",
																					ReferenceLocation: ast_domain.Location{
																						Line:   34,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   55,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "pages_main_594861c5",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Title",
																						ReferenceLocation: ast_domain.Location{
																							Line:   34,
																							Column: 27,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   55,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName: new("row"),
																				},
																				BaseCodeGenVarName:  new("row"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																				Stringability:       1,
																			},
																		},
																		Location: ast_domain.Location{
																			Line:   34,
																			Column: 27,
																		},
																		GoFieldName: "RowTitle",
																		InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Title",
																				ReferenceLocation: ast_domain.Location{
																					Line:   34,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   55,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Title",
																					ReferenceLocation: ast_domain.Location{
																						Line:   34,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   55,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName: new("row"),
																			},
																			BaseCodeGenVarName:  new("row"),
																			OriginalSourcePath:  new("pages/main.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																	IsLoopDependent: true,
																	IsLoopDependent: true,
																},
															},
															DynamicAttributeOrigins: map[string]string{
																"category_name": "pages_main_594861c5",
																"container_id":  "pages_main_594861c5",
																"row_id":        "pages_main_594861c5",
																"row_title":     "pages_main_594861c5",
															},
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
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
																		OriginalSourcePath: new("partials/row_actions.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:0:0:0.",
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
																		OriginalSourcePath: new("partials/row_actions.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "row",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "row",
																					ReferenceLocation: ast_domain.Location{
																						Line:   25,
																						Column: 28,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("row"),
																				OriginalSourcePath: new("partials/row_actions.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 5,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   25,
																						Column: 46,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   54,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("row"),
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
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   25,
																					Column: 28,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   40,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("row"),
																			OriginalSourcePath:  new("partials/row_actions.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
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
																		OriginalSourcePath: new("partials/row_actions.pk"),
																		Stringability:      1,
																	},
																	Literal: ":1:0",
																},
															},
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
																OriginalSourcePath: new("partials/row_actions.pk"),
																Stringability:      1,
															},
														},
														Attributes: []ast_domain.HTMLAttribute{
															ast_domain.HTMLAttribute{
																Name:  "class",
																Value: "row-actions",
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
																Name:          "category_name",
																RawExpression: "state.CategoryName",
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
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   32,
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
																		Name: "CategoryName",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "CategoryName",
																				ReferenceLocation: ast_domain.Location{
																					Line:   32,
																					Column: 31,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   60,
																					Column: 2,
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
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "CategoryName",
																			ReferenceLocation: ast_domain.Location{
																				Line:   32,
																				Column: 31,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   60,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   32,
																	Column: 31,
																},
																NameLocation: ast_domain.Location{
																	Line:   32,
																	Column: 15,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "CategoryName",
																		ReferenceLocation: ast_domain.Location{
																			Line:   32,
																			Column: 31,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   60,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
															ast_domain.DynamicAttribute{
																Name:          "container_id",
																RawExpression: "state.ContainerID",
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
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 30,
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
																		Name: "ContainerID",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ContainerID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   59,
																					Column: 2,
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
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ContainerID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 30,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   59,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   31,
																	Column: 30,
																},
																NameLocation: ast_domain.Location{
																	Line:   31,
																	Column: 15,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ContainerID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 30,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   59,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
															ast_domain.DynamicAttribute{
																Name:          "row_id",
																RawExpression: "row.ID",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "row",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "row",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 24,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("row"),
																			OriginalSourcePath: new("pages/main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "ID",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 5,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 24,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   54,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("row"),
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
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   33,
																				Column: 24,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   54,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("row"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   33,
																	Column: 24,
																},
																NameLocation: ast_domain.Location{
																	Line:   33,
																	Column: 15,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 24,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   54,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
															ast_domain.DynamicAttribute{
																Name:          "row_title",
																RawExpression: "row.Title",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "row",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "row",
																				ReferenceLocation: ast_domain.Location{
																					Line:   34,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("row"),
																			OriginalSourcePath: new("pages/main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Title",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 5,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Title",
																				ReferenceLocation: ast_domain.Location{
																					Line:   34,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   55,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("row"),
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
																			CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Title",
																			ReferenceLocation: ast_domain.Location{
																				Line:   34,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   55,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("row"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																		Stringability:       1,
																	},
																},
																Location: ast_domain.Location{
																	Line:   34,
																	Column: 27,
																},
																NameLocation: ast_domain.Location{
																	Line:   34,
																	Column: 15,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   34,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   55,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("row"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
																TagName: "button",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_row_actions_c2843091"),
																	OriginalSourcePath:   new("partials/row_actions.pk"),
																},
																Key: &ast_domain.TemplateLiteral{
																	Parts: []ast_domain.TemplateLiteralPart{
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Literal: "r.0:0:0:0.",
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Expression: &ast_domain.MemberExpression{
																				Base: &ast_domain.Identifier{
																					Name: "row",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 1,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "row",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 28,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   0,
																								Column: 0,
																							},
																						},
																						BaseCodeGenVarName: new("row"),
																						OriginalSourcePath: new("partials/row_actions.pk"),
																					},
																				},
																				Property: &ast_domain.Identifier{
																					Name: "ID",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 5,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "ID",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 46,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   54,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("row"),
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
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   25,
																							Column: 28,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   40,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("row"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																					Stringability:       1,
																				},
																			},
																		},
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Literal: ":1:0:0",
																		},
																	},
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
																		OriginalSourcePath: new("partials/row_actions.pk"),
																		Stringability:      1,
																	},
																},
																Attributes: []ast_domain.HTMLAttribute{
																	ast_domain.HTMLAttribute{
																		Name:  "class",
																		Value: "edit-btn",
																		Location: ast_domain.Location{
																			Line:   23,
																			Column: 20,
																		},
																		NameLocation: ast_domain.Location{
																			Line:   23,
																			Column: 13,
																		},
																	},
																},
																Children: []*ast_domain.TemplateNode{
																	&ast_domain.TemplateNode{
																		NodeType: ast_domain.NodeText,
																		Location: ast_domain.Location{
																			Line:   23,
																			Column: 30,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			OriginalPackageAlias: new("partials_row_actions_c2843091"),
																			OriginalSourcePath:   new("partials/row_actions.pk"),
																		},
																		Key: &ast_domain.TemplateLiteral{
																			Parts: []ast_domain.TemplateLiteralPart{
																				ast_domain.TemplateLiteralPart{
																					IsLiteral: true,
																					RelativeLocation: ast_domain.Location{
																						Line:   23,
																						Column: 30,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "",
																							CanonicalPackagePath: "",
																						},
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Literal: "r.0:0:0:0.",
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
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Expression: &ast_domain.MemberExpression{
																						Base: &ast_domain.Identifier{
																							Name: "row",
																							RelativeLocation: ast_domain.Location{
																								Line:   1,
																								Column: 1,
																							},
																							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																								ResolvedType: &ast_domain.ResolvedTypeInfo{
																									TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																									PackageAlias:         "pages_main_594861c5",
																									CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																								},
																								Symbol: &ast_domain.ResolvedSymbol{
																									Name: "row",
																									ReferenceLocation: ast_domain.Location{
																										Line:   25,
																										Column: 28,
																									},
																									DeclarationLocation: ast_domain.Location{
																										Line:   0,
																										Column: 0,
																									},
																								},
																								BaseCodeGenVarName: new("row"),
																								OriginalSourcePath: new("partials/row_actions.pk"),
																							},
																						},
																						Property: &ast_domain.Identifier{
																							Name: "ID",
																							RelativeLocation: ast_domain.Location{
																								Line:   1,
																								Column: 5,
																							},
																							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																								ResolvedType: &ast_domain.ResolvedTypeInfo{
																									TypeExpression:       typeExprFromString("string"),
																									PackageAlias:         "pages_main_594861c5",
																									CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																								},
																								Symbol: &ast_domain.ResolvedSymbol{
																									Name: "ID",
																									ReferenceLocation: ast_domain.Location{
																										Line:   25,
																										Column: 46,
																									},
																									DeclarationLocation: ast_domain.Location{
																										Line:   54,
																										Column: 2,
																									},
																								},
																								BaseCodeGenVarName:  new("row"),
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
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "ID",
																								ReferenceLocation: ast_domain.Location{
																									Line:   25,
																									Column: 28,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   40,
																									Column: 2,
																								},
																							},
																							BaseCodeGenVarName:  new("row"),
																							OriginalSourcePath:  new("partials/row_actions.pk"),
																							GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																							Stringability:       1,
																						},
																					},
																				},
																				ast_domain.TemplateLiteralPart{
																					IsLiteral: true,
																					RelativeLocation: ast_domain.Location{
																						Line:   23,
																						Column: 30,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "",
																							CanonicalPackagePath: "",
																						},
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Literal: ":1:0:0:0",
																				},
																			},
																			RelativeLocation: ast_domain.Location{
																				Line:   23,
																				Column: 30,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																		},
																		RichText: []ast_domain.TextPart{
																			ast_domain.TextPart{
																				IsLiteral: true,
																				Location: ast_domain.Location{
																					Line:   23,
																					Column: 30,
																				},
																				Literal: "Edit ",
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					OriginalSourcePath: new("partials/row_actions.pk"),
																				},
																			},
																			ast_domain.TextPart{
																				IsLiteral: false,
																				Location: ast_domain.Location{
																					Line:   23,
																					Column: 38,
																				},
																				RawExpression: "state.DisplayTitle",
																				Expression: &ast_domain.MemberExpression{
																					Base: &ast_domain.Identifier{
																						Name: "state",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 1,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("partials_row_actions_c2843091.Response"),
																								PackageAlias:         "partials_row_actions_c2843091",
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
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
																							BaseCodeGenVarName: new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																							OriginalSourcePath: new("partials/row_actions.pk"),
																						},
																					},
																					Property: &ast_domain.Identifier{
																						Name: "DisplayTitle",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 7,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("string"),
																								PackageAlias:         "partials_row_actions_c2843091",
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "DisplayTitle",
																								ReferenceLocation: ast_domain.Location{
																									Line:   23,
																									Column: 38,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   44,
																									Column: 2,
																								},
																							},
																							BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																							OriginalSourcePath:  new("partials/row_actions.pk"),
																							GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
																							PackageAlias:         "partials_row_actions_c2843091",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "DisplayTitle",
																							ReferenceLocation: ast_domain.Location{
																								Line:   23,
																								Column: 38,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   44,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																						OriginalSourcePath:  new("partials/row_actions.pk"),
																						GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
																						Stringability:       1,
																					},
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "partials_row_actions_c2843091",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "DisplayTitle",
																						ReferenceLocation: ast_domain.Location{
																							Line:   23,
																							Column: 38,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   44,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
																	Line:   24,
																	Column: 5,
																},
																TagName: "a",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_row_actions_c2843091"),
																	OriginalSourcePath:   new("partials/row_actions.pk"),
																},
																Key: &ast_domain.TemplateLiteral{
																	Parts: []ast_domain.TemplateLiteralPart{
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Literal: "r.0:0:0:0.",
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Expression: &ast_domain.MemberExpression{
																				Base: &ast_domain.Identifier{
																					Name: "row",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 1,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "row",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 28,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   0,
																								Column: 0,
																							},
																						},
																						BaseCodeGenVarName: new("row"),
																						OriginalSourcePath: new("partials/row_actions.pk"),
																					},
																				},
																				Property: &ast_domain.Identifier{
																					Name: "ID",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 5,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "ID",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 46,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   54,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("row"),
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
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   25,
																							Column: 28,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   40,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("row"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																					Stringability:       1,
																				},
																			},
																		},
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Literal: ":1:0:1",
																		},
																	},
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
																		OriginalSourcePath: new("partials/row_actions.pk"),
																		Stringability:      1,
																	},
																},
																Attributes: []ast_domain.HTMLAttribute{
																	ast_domain.HTMLAttribute{
																		Name:  "class",
																		Value: "edit-link",
																		Location: ast_domain.Location{
																			Line:   24,
																			Column: 37,
																		},
																		NameLocation: ast_domain.Location{
																			Line:   24,
																			Column: 30,
																		},
																	},
																},
																DynamicAttributes: []ast_domain.DynamicAttribute{
																	ast_domain.DynamicAttribute{
																		Name:          "href",
																		RawExpression: "state.EditURL",
																		Expression: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "state",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("partials_row_actions_c2843091.Response"),
																						PackageAlias:         "partials_row_actions_c2843091",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "state",
																						ReferenceLocation: ast_domain.Location{
																							Line:   24,
																							Column: 15,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																					OriginalSourcePath: new("partials/row_actions.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "EditURL",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "partials_row_actions_c2843091",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "EditURL",
																						ReferenceLocation: ast_domain.Location{
																							Line:   24,
																							Column: 15,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   45,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
																					PackageAlias:         "partials_row_actions_c2843091",
																					CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "EditURL",
																					ReferenceLocation: ast_domain.Location{
																						Line:   24,
																						Column: 15,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   45,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																				OriginalSourcePath:  new("partials/row_actions.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
																				Stringability:       1,
																			},
																		},
																		Location: ast_domain.Location{
																			Line:   24,
																			Column: 15,
																		},
																		NameLocation: ast_domain.Location{
																			Line:   24,
																			Column: 8,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "partials_row_actions_c2843091",
																				CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "EditURL",
																				ReferenceLocation: ast_domain.Location{
																					Line:   24,
																					Column: 15,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   45,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																			OriginalSourcePath:  new("partials/row_actions.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
																			Stringability:       1,
																		},
																	},
																},
																Children: []*ast_domain.TemplateNode{
																	&ast_domain.TemplateNode{
																		NodeType: ast_domain.NodeText,
																		Location: ast_domain.Location{
																			Line:   24,
																			Column: 48,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			OriginalPackageAlias: new("partials_row_actions_c2843091"),
																			OriginalSourcePath:   new("partials/row_actions.pk"),
																		},
																		Key: &ast_domain.TemplateLiteral{
																			Parts: []ast_domain.TemplateLiteralPart{
																				ast_domain.TemplateLiteralPart{
																					IsLiteral: true,
																					RelativeLocation: ast_domain.Location{
																						Line:   24,
																						Column: 48,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "",
																							CanonicalPackagePath: "",
																						},
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Literal: "r.0:0:0:0.",
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
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Expression: &ast_domain.MemberExpression{
																						Base: &ast_domain.Identifier{
																							Name: "row",
																							RelativeLocation: ast_domain.Location{
																								Line:   1,
																								Column: 1,
																							},
																							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																								ResolvedType: &ast_domain.ResolvedTypeInfo{
																									TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																									PackageAlias:         "pages_main_594861c5",
																									CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																								},
																								Symbol: &ast_domain.ResolvedSymbol{
																									Name: "row",
																									ReferenceLocation: ast_domain.Location{
																										Line:   25,
																										Column: 28,
																									},
																									DeclarationLocation: ast_domain.Location{
																										Line:   0,
																										Column: 0,
																									},
																								},
																								BaseCodeGenVarName: new("row"),
																								OriginalSourcePath: new("partials/row_actions.pk"),
																							},
																						},
																						Property: &ast_domain.Identifier{
																							Name: "ID",
																							RelativeLocation: ast_domain.Location{
																								Line:   1,
																								Column: 5,
																							},
																							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																								ResolvedType: &ast_domain.ResolvedTypeInfo{
																									TypeExpression:       typeExprFromString("string"),
																									PackageAlias:         "pages_main_594861c5",
																									CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																								},
																								Symbol: &ast_domain.ResolvedSymbol{
																									Name: "ID",
																									ReferenceLocation: ast_domain.Location{
																										Line:   25,
																										Column: 46,
																									},
																									DeclarationLocation: ast_domain.Location{
																										Line:   54,
																										Column: 2,
																									},
																								},
																								BaseCodeGenVarName:  new("row"),
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
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "ID",
																								ReferenceLocation: ast_domain.Location{
																									Line:   25,
																									Column: 28,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   40,
																									Column: 2,
																								},
																							},
																							BaseCodeGenVarName:  new("row"),
																							OriginalSourcePath:  new("partials/row_actions.pk"),
																							GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																							Stringability:       1,
																						},
																					},
																				},
																				ast_domain.TemplateLiteralPart{
																					IsLiteral: true,
																					RelativeLocation: ast_domain.Location{
																						Line:   24,
																						Column: 48,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "",
																							CanonicalPackagePath: "",
																						},
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Literal: ":1:0:1:0",
																				},
																			},
																			RelativeLocation: ast_domain.Location{
																				Line:   24,
																				Column: 48,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																		},
																		RichText: []ast_domain.TextPart{
																			ast_domain.TextPart{
																				IsLiteral: true,
																				Location: ast_domain.Location{
																					Line:   24,
																					Column: 48,
																				},
																				Literal: "Go to ",
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					OriginalSourcePath: new("partials/row_actions.pk"),
																				},
																			},
																			ast_domain.TextPart{
																				IsLiteral: false,
																				Location: ast_domain.Location{
																					Line:   24,
																					Column: 57,
																				},
																				RawExpression: "state.DisplayTitle",
																				Expression: &ast_domain.MemberExpression{
																					Base: &ast_domain.Identifier{
																						Name: "state",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 1,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("partials_row_actions_c2843091.Response"),
																								PackageAlias:         "partials_row_actions_c2843091",
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "state",
																								ReferenceLocation: ast_domain.Location{
																									Line:   24,
																									Column: 57,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   0,
																									Column: 0,
																								},
																							},
																							BaseCodeGenVarName: new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																							OriginalSourcePath: new("partials/row_actions.pk"),
																						},
																					},
																					Property: &ast_domain.Identifier{
																						Name: "DisplayTitle",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 7,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("string"),
																								PackageAlias:         "partials_row_actions_c2843091",
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "DisplayTitle",
																								ReferenceLocation: ast_domain.Location{
																									Line:   24,
																									Column: 57,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   44,
																									Column: 2,
																								},
																							},
																							BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																							OriginalSourcePath:  new("partials/row_actions.pk"),
																							GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
																							PackageAlias:         "partials_row_actions_c2843091",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "DisplayTitle",
																							ReferenceLocation: ast_domain.Location{
																								Line:   24,
																								Column: 57,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   44,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																						OriginalSourcePath:  new("partials/row_actions.pk"),
																						GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
																						Stringability:       1,
																					},
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "partials_row_actions_c2843091",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "DisplayTitle",
																						ReferenceLocation: ast_domain.Location{
																							Line:   24,
																							Column: 57,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   44,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
																	Line:   25,
																	Column: 5,
																},
																TagName: "span",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_row_actions_c2843091"),
																	OriginalSourcePath:   new("partials/row_actions.pk"),
																},
																Key: &ast_domain.TemplateLiteral{
																	Parts: []ast_domain.TemplateLiteralPart{
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
																			RelativeLocation: ast_domain.Location{
																				Line:   25,
																				Column: 5,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Literal: "r.0:0:0:0.",
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
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Expression: &ast_domain.MemberExpression{
																				Base: &ast_domain.Identifier{
																					Name: "row",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 1,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "row",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 28,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   0,
																								Column: 0,
																							},
																						},
																						BaseCodeGenVarName: new("row"),
																						OriginalSourcePath: new("partials/row_actions.pk"),
																					},
																				},
																				Property: &ast_domain.Identifier{
																					Name: "ID",
																					RelativeLocation: ast_domain.Location{
																						Line:   1,
																						Column: 5,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "pages_main_594861c5",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "ID",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 46,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   54,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("row"),
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
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "ID",
																						ReferenceLocation: ast_domain.Location{
																							Line:   25,
																							Column: 28,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   40,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("row"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																					Stringability:       1,
																				},
																			},
																		},
																		ast_domain.TemplateLiteralPart{
																			IsLiteral: true,
																			RelativeLocation: ast_domain.Location{
																				Line:   25,
																				Column: 5,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																			Literal: ":1:0:2",
																		},
																	},
																	RelativeLocation: ast_domain.Location{
																		Line:   25,
																		Column: 5,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/row_actions.pk"),
																		Stringability:      1,
																	},
																},
																Attributes: []ast_domain.HTMLAttribute{
																	ast_domain.HTMLAttribute{
																		Name:  "class",
																		Value: "category",
																		Location: ast_domain.Location{
																			Line:   25,
																			Column: 18,
																		},
																		NameLocation: ast_domain.Location{
																			Line:   25,
																			Column: 11,
																		},
																	},
																},
																Children: []*ast_domain.TemplateNode{
																	&ast_domain.TemplateNode{
																		NodeType: ast_domain.NodeText,
																		Location: ast_domain.Location{
																			Line:   25,
																			Column: 28,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			OriginalPackageAlias: new("partials_row_actions_c2843091"),
																			OriginalSourcePath:   new("partials/row_actions.pk"),
																		},
																		Key: &ast_domain.TemplateLiteral{
																			Parts: []ast_domain.TemplateLiteralPart{
																				ast_domain.TemplateLiteralPart{
																					IsLiteral: true,
																					RelativeLocation: ast_domain.Location{
																						Line:   25,
																						Column: 28,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "",
																							CanonicalPackagePath: "",
																						},
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Literal: "r.0:0:0:0.",
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
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Expression: &ast_domain.MemberExpression{
																						Base: &ast_domain.Identifier{
																							Name: "row",
																							RelativeLocation: ast_domain.Location{
																								Line:   1,
																								Column: 1,
																							},
																							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																								ResolvedType: &ast_domain.ResolvedTypeInfo{
																									TypeExpression:       typeExprFromString("pages_main_594861c5.Row"),
																									PackageAlias:         "pages_main_594861c5",
																									CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																								},
																								Symbol: &ast_domain.ResolvedSymbol{
																									Name: "row",
																									ReferenceLocation: ast_domain.Location{
																										Line:   25,
																										Column: 28,
																									},
																									DeclarationLocation: ast_domain.Location{
																										Line:   0,
																										Column: 0,
																									},
																								},
																								BaseCodeGenVarName: new("row"),
																								OriginalSourcePath: new("partials/row_actions.pk"),
																							},
																						},
																						Property: &ast_domain.Identifier{
																							Name: "ID",
																							RelativeLocation: ast_domain.Location{
																								Line:   1,
																								Column: 5,
																							},
																							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																								ResolvedType: &ast_domain.ResolvedTypeInfo{
																									TypeExpression:       typeExprFromString("string"),
																									PackageAlias:         "pages_main_594861c5",
																									CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																								},
																								Symbol: &ast_domain.ResolvedSymbol{
																									Name: "ID",
																									ReferenceLocation: ast_domain.Location{
																										Line:   25,
																										Column: 46,
																									},
																									DeclarationLocation: ast_domain.Location{
																										Line:   54,
																										Column: 2,
																									},
																								},
																								BaseCodeGenVarName:  new("row"),
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
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/pages/pages_main_594861c5",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "ID",
																								ReferenceLocation: ast_domain.Location{
																									Line:   25,
																									Column: 28,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   40,
																									Column: 2,
																								},
																							},
																							BaseCodeGenVarName:  new("row"),
																							OriginalSourcePath:  new("partials/row_actions.pk"),
																							GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																							Stringability:       1,
																						},
																					},
																				},
																				ast_domain.TemplateLiteralPart{
																					IsLiteral: true,
																					RelativeLocation: ast_domain.Location{
																						Line:   25,
																						Column: 28,
																					},
																					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("string"),
																							PackageAlias:         "",
																							CanonicalPackagePath: "",
																						},
																						OriginalSourcePath: new("partials/row_actions.pk"),
																						Stringability:      1,
																					},
																					Literal: ":1:0:2:0",
																				},
																			},
																			RelativeLocation: ast_domain.Location{
																				Line:   25,
																				Column: 28,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "",
																					CanonicalPackagePath: "",
																				},
																				OriginalSourcePath: new("partials/row_actions.pk"),
																				Stringability:      1,
																			},
																		},
																		RichText: []ast_domain.TextPart{
																			ast_domain.TextPart{
																				IsLiteral: false,
																				Location: ast_domain.Location{
																					Line:   25,
																					Column: 31,
																				},
																				RawExpression: "state.CategoryLabel",
																				Expression: &ast_domain.MemberExpression{
																					Base: &ast_domain.Identifier{
																						Name: "state",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 1,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("partials_row_actions_c2843091.Response"),
																								PackageAlias:         "partials_row_actions_c2843091",
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "state",
																								ReferenceLocation: ast_domain.Location{
																									Line:   25,
																									Column: 31,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   0,
																									Column: 0,
																								},
																							},
																							BaseCodeGenVarName: new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																							OriginalSourcePath: new("partials/row_actions.pk"),
																						},
																					},
																					Property: &ast_domain.Identifier{
																						Name: "CategoryLabel",
																						RelativeLocation: ast_domain.Location{
																							Line:   1,
																							Column: 7,
																						},
																						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																							ResolvedType: &ast_domain.ResolvedTypeInfo{
																								TypeExpression:       typeExprFromString("string"),
																								PackageAlias:         "partials_row_actions_c2843091",
																								CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																							},
																							Symbol: &ast_domain.ResolvedSymbol{
																								Name: "CategoryLabel",
																								ReferenceLocation: ast_domain.Location{
																									Line:   25,
																									Column: 31,
																								},
																								DeclarationLocation: ast_domain.Location{
																									Line:   46,
																									Column: 2,
																								},
																							},
																							BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																							OriginalSourcePath:  new("partials/row_actions.pk"),
																							GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
																							PackageAlias:         "partials_row_actions_c2843091",
																							CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "CategoryLabel",
																							ReferenceLocation: ast_domain.Location{
																								Line:   25,
																								Column: 31,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   46,
																								Column: 2,
																							},
																						},
																						BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																						OriginalSourcePath:  new("partials/row_actions.pk"),
																						GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
																						Stringability:       1,
																					},
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("string"),
																						PackageAlias:         "partials_row_actions_c2843091",
																						CanonicalPackagePath: "testcase_133_partial_sibling_of_nested_loop/dist/partials/partials_row_actions_c2843091",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "CategoryLabel",
																						ReferenceLocation: ast_domain.Location{
																							Line:   25,
																							Column: 31,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   46,
																							Column: 2,
																						},
																					},
																					BaseCodeGenVarName:  new("partials_row_actions_c2843091Data_row_actions_category_name_state_categoryname_container_id_state_containerid_row_id_row_id_row_title_row_title_839351af"),
																					OriginalSourcePath:  new("partials/row_actions.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_row_actions_c2843091/generated.go"),
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
					},
				},
			},
		},
	}
}()
