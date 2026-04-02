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
								TextContent: "P-If CSS Scoping Test",
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
							OriginalPackageAlias: new("partials_browser_4016e9df"),
							OriginalSourcePath:   new("partials/browser.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "browser_show_items_true_c57d7a47",
								PartialAlias:        "browser",
								PartialPackageName:  "partials_browser_4016e9df",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"show_items": ast_domain.PropValue{
										Expression: &ast_domain.BooleanLiteral{
											Value: true,
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 52,
										},
										GoFieldName: "ShowItems",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
											},
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
									},
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
								OriginalSourcePath: new("partials/browser.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "browser-container",
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
								TagName: "header",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_browser_4016e9df"),
									OriginalSourcePath:   new("partials/browser.pk"),
									IsStatic:             true,
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
										OriginalSourcePath: new("partials/browser.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "browser-header",
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   24,
											Column: 7,
										},
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_browser_4016e9df"),
											OriginalSourcePath:   new("partials/browser.pk"),
											IsStatic:             true,
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
												OriginalSourcePath: new("partials/browser.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 11,
												},
												TextContent: "Items Browser",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_browser_4016e9df"),
													OriginalSourcePath:   new("partials/browser.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 11,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/browser.pk"),
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
									Line:   27,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_browser_4016e9df"),
									OriginalSourcePath:   new("partials/browser.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   27,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   27,
										Column: 10,
									},
									RawExpression: "!props.ShowItems",
									Expression: &ast_domain.UnaryExpression{
										Operator: "!",
										Right: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 2,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_browser_4016e9df.Props"),
														PackageAlias:         "partials_browser_4016e9df",
														CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_browser_show_items_true_c57d7a47"),
													OriginalSourcePath: new("partials/browser.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "ShowItems",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "partials_browser_4016e9df",
														CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ShowItems",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_browser_show_items_true_c57d7a47"),
													OriginalSourcePath:  new("partials/browser.pk"),
													GeneratedSourcePath: new("dist/partials/partials_browser_4016e9df/generated.go"),
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
													PackageAlias:         "partials_browser_4016e9df",
													CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ShowItems",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												BaseCodeGenVarName:  new("props_browser_show_items_true_c57d7a47"),
												OriginalSourcePath:  new("partials/browser.pk"),
												GeneratedSourcePath: new("dist/partials/partials_browser_4016e9df/generated.go"),
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
											OriginalSourcePath: new("partials/browser.pk"),
											Stringability:      1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/browser.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/browser.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
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
										OriginalSourcePath: new("partials/browser.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "no-results",
										Location: ast_domain.Location{
											Line:   27,
											Column: 41,
										},
										NameLocation: ast_domain.Location{
											Line:   27,
											Column: 34,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   28,
											Column: 7,
										},
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_browser_4016e9df"),
											OriginalSourcePath:   new("partials/browser.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   28,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/browser.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   28,
													Column: 10,
												},
												TextContent: "No items found.",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_browser_4016e9df"),
													OriginalSourcePath:   new("partials/browser.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   28,
														Column: 10,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/browser.pk"),
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
									Line:   31,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_browser_4016e9df"),
									OriginalSourcePath:   new("partials/browser.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   31,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   31,
										Column: 10,
									},
									RawExpression: "props.ShowItems",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "props",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("partials_browser_4016e9df.Props"),
													PackageAlias:         "partials_browser_4016e9df",
													CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   31,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("props_browser_show_items_true_c57d7a47"),
												OriginalSourcePath: new("partials/browser.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "ShowItems",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_browser_4016e9df",
													CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ShowItems",
													ReferenceLocation: ast_domain.Location{
														Line:   31,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												BaseCodeGenVarName:  new("props_browser_show_items_true_c57d7a47"),
												OriginalSourcePath:  new("partials/browser.pk"),
												GeneratedSourcePath: new("dist/partials/partials_browser_4016e9df/generated.go"),
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
												PackageAlias:         "partials_browser_4016e9df",
												CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ShowItems",
												ReferenceLocation: ast_domain.Location{
													Line:   31,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
											},
											BaseCodeGenVarName:  new("props_browser_show_items_true_c57d7a47"),
											OriginalSourcePath:  new("partials/browser.pk"),
											GeneratedSourcePath: new("dist/partials/partials_browser_4016e9df/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "partials_browser_4016e9df",
											CanonicalPackagePath: "testcase_107_p_if_css_scoping_in_partial/dist/partials/partials_browser_4016e9df",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ShowItems",
											ReferenceLocation: ast_domain.Location{
												Line:   31,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   71,
												Column: 2,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
										},
										BaseCodeGenVarName:  new("props_browser_show_items_true_c57d7a47"),
										OriginalSourcePath:  new("partials/browser.pk"),
										GeneratedSourcePath: new("dist/partials/partials_browser_4016e9df/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:2",
									RelativeLocation: ast_domain.Location{
										Line:   31,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/browser.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "items-grid",
										Location: ast_domain.Location{
											Line:   31,
											Column: 40,
										},
										NameLocation: ast_domain.Location{
											Line:   31,
											Column: 33,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   32,
											Column: 7,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_browser_4016e9df"),
											OriginalSourcePath:   new("partials/browser.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   32,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/browser.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-card",
												Location: ast_domain.Location{
													Line:   32,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   32,
													Column: 12,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   32,
													Column: 30,
												},
												TextContent: "Item 1",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_browser_4016e9df"),
													OriginalSourcePath:   new("partials/browser.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:2:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   32,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/browser.pk"),
														Stringability:      1,
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   33,
											Column: 7,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_browser_4016e9df"),
											OriginalSourcePath:   new("partials/browser.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:2:1",
											RelativeLocation: ast_domain.Location{
												Line:   33,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/browser.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-card",
												Location: ast_domain.Location{
													Line:   33,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   33,
													Column: 12,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   33,
													Column: 30,
												},
												TextContent: "Item 2",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_browser_4016e9df"),
													OriginalSourcePath:   new("partials/browser.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:2:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   33,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/browser.pk"),
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
											Column: 7,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_browser_4016e9df"),
											OriginalSourcePath:   new("partials/browser.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:2:2",
											RelativeLocation: ast_domain.Location{
												Line:   34,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/browser.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-card",
												Location: ast_domain.Location{
													Line:   34,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   34,
													Column: 12,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   34,
													Column: 30,
												},
												TextContent: "Item 3",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_browser_4016e9df"),
													OriginalSourcePath:   new("partials/browser.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:2:2:0",
													RelativeLocation: ast_domain.Location{
														Line:   34,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/browser.pk"),
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
			},
		},
	}
}()
