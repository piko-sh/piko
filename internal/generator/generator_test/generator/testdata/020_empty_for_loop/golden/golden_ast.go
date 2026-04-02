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
								TextContent: "My Items",
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
							Line:   24,
							Column: 5,
						},
						TagName: "ul",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "item-list",
								Location: ast_domain.Location{
									Line:   24,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 9,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 7,
								},
								TagName: "li",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   25,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 11,
									},
									RawExpression: "item in state.Items",
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
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										ItemVariable: &ast_domain.Identifier{
											Name: "item",
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
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "item",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("item"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 9,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
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
												Name: "Items",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 15,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   35,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
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
													TypeExpression:       typeExprFromString("[]string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       5,
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   35,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       5,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   35,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       5,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Items",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 18,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   35,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										Stringability:       5,
									},
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   25,
										Column: 47,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 39,
									},
									RawExpression: "item",
									Expression: &ast_domain.Identifier{
										Name: "item",
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
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "item",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 47,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("item"),
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "item",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 47,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("item"),
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   25,
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.Identifier{
												Name: "item",
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
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "item",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("item"),
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   25,
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
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   27,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   27,
								Column: 14,
							},
							NameLocation: ast_domain.Location{
								Line:   27,
								Column: 8,
							},
							RawExpression: "len(state.Items) == 0",
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
													Line:   27,
													Column: 14,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("len"),
											OriginalSourcePath: new("pages/main.pk"),
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
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 14,
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
												Name: "Items",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 11,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 14,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   35,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
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
													TypeExpression:       typeExprFromString("[]string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_020_empty_for_loop/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 14,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   35,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       5,
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
								Operator: "==",
								Right: &ast_domain.IntegerLiteral{
									Value: 0,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 21,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("pages/main.pk"),
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
									OriginalSourcePath: new("pages/main.pk"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   27,
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
								Value: "empty-message",
								Location: ast_domain.Location{
									Line:   27,
									Column: 44,
								},
								NameLocation: ast_domain.Location{
									Line:   27,
									Column: 37,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   27,
									Column: 59,
								},
								TextContent: "Your list is empty.",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   27,
										Column: 59,
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
				},
			},
		},
	}
}()
