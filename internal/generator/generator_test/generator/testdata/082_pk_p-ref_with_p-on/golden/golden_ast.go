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
					IsStatic:             true,
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
								TextContent: "p-ref with p-on:click Interaction Test",
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
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   24,
									Column: 8,
								},
								TextContent: " This page tests that p-ref and p-on:click work together. The event handler should be able to access refs to manipulate DOM. ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
							Column: 5,
						},
						TagName: "input",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						DirRef: &ast_domain.Directive{
							Type: ast_domain.DirectiveRef,
							Location: ast_domain.Location{
								Line:   31,
								Column: 14,
							},
							NameLocation: ast_domain.Location{
								Line:   31,
								Column: 7,
							},
							RawExpression: "searchInput",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								OriginalSourcePath: new("pages/main.pk"),
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   29,
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
								Name:  "type",
								Value: "text",
								Location: ast_domain.Location{
									Line:   30,
									Column: 13,
								},
								NameLocation: ast_domain.Location{
									Line:   30,
									Column: 7,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "placeholder",
								Value: "Search...",
								Location: ast_domain.Location{
									Line:   32,
									Column: 20,
								},
								NameLocation: ast_domain.Location{
									Line:   32,
									Column: 7,
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
						TagName: "button",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						OnEvents: map[string][]ast_domain.Directive{
							"click": []ast_domain.Directive{
								ast_domain.Directive{
									Type: ast_domain.DirectiveOn,
									Location: ast_domain.Location{
										Line:   35,
										Column: 25,
									},
									NameLocation: ast_domain.Location{
										Line:   35,
										Column: 13,
									},
									Arg:           "click",
									RawExpression: "clearSearch()",
									Expression: &ast_domain.CallExpression{
										Callee: &ast_domain.Identifier{
											Name: "clearSearch",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
										},
										Args: []ast_domain.Expression{},
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
									Line:   35,
									Column: 40,
								},
								TextContent: " Clear Search ",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   35,
										Column: 40,
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
							Line:   39,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   40,
									Column: 7,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								DirRef: &ast_domain.Directive{
									Type: ast_domain.DirectiveRef,
									Location: ast_domain.Location{
										Line:   40,
										Column: 20,
									},
									NameLocation: ast_domain.Location{
										Line:   40,
										Column: 13,
									},
									RawExpression: "counterDisplay",
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   40,
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
											Line:   40,
											Column: 36,
										},
										TextContent: "0",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   40,
												Column: 36,
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
									Line:   41,
									Column: 7,
								},
								TagName: "button",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:1",
									RelativeLocation: ast_domain.Location{
										Line:   41,
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
								OnEvents: map[string][]ast_domain.Directive{
									"click": []ast_domain.Directive{
										ast_domain.Directive{
											Type: ast_domain.DirectiveOn,
											Location: ast_domain.Location{
												Line:   41,
												Column: 27,
											},
											NameLocation: ast_domain.Location{
												Line:   41,
												Column: 15,
											},
											Arg:           "click",
											RawExpression: "incrementCounter()",
											Expression: &ast_domain.CallExpression{
												Callee: &ast_domain.Identifier{
													Name: "incrementCounter",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
												},
												Args: []ast_domain.Expression{},
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
											Line:   41,
											Column: 47,
										},
										TextContent: "+",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   41,
												Column: 47,
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
									Line:   42,
									Column: 7,
								},
								TagName: "button",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:2",
									RelativeLocation: ast_domain.Location{
										Line:   42,
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
								OnEvents: map[string][]ast_domain.Directive{
									"click": []ast_domain.Directive{
										ast_domain.Directive{
											Type: ast_domain.DirectiveOn,
											Location: ast_domain.Location{
												Line:   42,
												Column: 27,
											},
											NameLocation: ast_domain.Location{
												Line:   42,
												Column: 15,
											},
											Arg:           "click",
											RawExpression: "decrementCounter()",
											Expression: &ast_domain.CallExpression{
												Callee: &ast_domain.Identifier{
													Name: "decrementCounter",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
												},
												Args: []ast_domain.Expression{},
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
											Line:   42,
											Column: 47,
										},
										TextContent: "-",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   42,
												Column: 47,
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   45,
							Column: 5,
						},
						TagName: "form",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						DirRef: &ast_domain.Directive{
							Type: ast_domain.DirectiveRef,
							Location: ast_domain.Location{
								Line:   45,
								Column: 18,
							},
							NameLocation: ast_domain.Location{
								Line:   45,
								Column: 11,
							},
							RawExpression: "loginForm",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								OriginalSourcePath: new("pages/main.pk"),
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   45,
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
						OnEvents: map[string][]ast_domain.Directive{
							"submit": []ast_domain.Directive{
								ast_domain.Directive{
									Type: ast_domain.DirectiveOn,
									Location: ast_domain.Location{
										Line:   45,
										Column: 42,
									},
									NameLocation: ast_domain.Location{
										Line:   45,
										Column: 29,
									},
									Arg:           "submit",
									RawExpression: "handleLogin($event)",
									Expression: &ast_domain.CallExpression{
										Callee: &ast_domain.Identifier{
											Name: "handleLogin",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
										},
										Args: []ast_domain.Expression{
											&ast_domain.Identifier{
												Name: "$event",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 13,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("js.Event"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "$event",
														ReferenceLocation: ast_domain.Location{
															Line:   45,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("$event"),
													OriginalSourcePath: new("pages/main.pk"),
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
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   46,
									Column: 7,
								},
								TagName: "input",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								DirRef: &ast_domain.Directive{
									Type: ast_domain.DirectiveRef,
									Location: ast_domain.Location{
										Line:   46,
										Column: 34,
									},
									NameLocation: ast_domain.Location{
										Line:   46,
										Column: 27,
									},
									RawExpression: "emailField",
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   46,
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
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "type",
										Value: "email",
										Location: ast_domain.Location{
											Line:   46,
											Column: 20,
										},
										NameLocation: ast_domain.Location{
											Line:   46,
											Column: 14,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "name",
										Value: "email",
										Location: ast_domain.Location{
											Line:   46,
											Column: 52,
										},
										NameLocation: ast_domain.Location{
											Line:   46,
											Column: 46,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "placeholder",
										Value: "Email",
										Location: ast_domain.Location{
											Line:   46,
											Column: 72,
										},
										NameLocation: ast_domain.Location{
											Line:   46,
											Column: 59,
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   47,
									Column: 7,
								},
								TagName: "input",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								DirRef: &ast_domain.Directive{
									Type: ast_domain.DirectiveRef,
									Location: ast_domain.Location{
										Line:   47,
										Column: 37,
									},
									NameLocation: ast_domain.Location{
										Line:   47,
										Column: 30,
									},
									RawExpression: "passwordField",
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:1",
									RelativeLocation: ast_domain.Location{
										Line:   47,
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
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "type",
										Value: "password",
										Location: ast_domain.Location{
											Line:   47,
											Column: 20,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 14,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "name",
										Value: "password",
										Location: ast_domain.Location{
											Line:   47,
											Column: 58,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 52,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "placeholder",
										Value: "Password",
										Location: ast_domain.Location{
											Line:   47,
											Column: 81,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 68,
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   48,
									Column: 7,
								},
								TagName: "button",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								DirRef: &ast_domain.Directive{
									Type: ast_domain.DirectiveRef,
									Location: ast_domain.Location{
										Line:   48,
										Column: 36,
									},
									NameLocation: ast_domain.Location{
										Line:   48,
										Column: 29,
									},
									RawExpression: "submitButton",
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:2",
									RelativeLocation: ast_domain.Location{
										Line:   48,
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
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "type",
										Value: "submit",
										Location: ast_domain.Location{
											Line:   48,
											Column: 21,
										},
										NameLocation: ast_domain.Location{
											Line:   48,
											Column: 15,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   48,
											Column: 50,
										},
										TextContent: "Login",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   48,
												Column: 50,
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
			},
		},
	}
}()
