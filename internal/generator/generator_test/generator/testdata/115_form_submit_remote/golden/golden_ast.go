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
								TextContent: "Form Submit Action Test",
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
								TextContent: " This page tests that p-on:submit with the action. prefix properly encodes the action payload as base64 JSON, not as a raw string. ",
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
						TagName: "form",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
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
								Name:  "id",
								Value: "static-form",
								Location: ast_domain.Location{
									Line:   29,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   29,
									Column: 11,
								},
							},
						},
						OnEvents: map[string][]ast_domain.Directive{
							"submit": []ast_domain.Directive{
								ast_domain.Directive{
									Type: ast_domain.DirectiveOn,
									Location: ast_domain.Location{
										Line:   29,
										Column: 41,
									},
									NameLocation: ast_domain.Location{
										Line:   29,
										Column: 28,
									},
									Arg:           "submit",
									Modifier:      "action",
									RawExpression: "action.static_action()",
									Expression: &ast_domain.CallExpression{
										Callee: &ast_domain.Identifier{
											Name: "static_action",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
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
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   30,
									Column: 7,
								},
								TagName: "input",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   30,
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
										Value: "text",
										Location: ast_domain.Location{
											Line:   30,
											Column: 20,
										},
										NameLocation: ast_domain.Location{
											Line:   30,
											Column: 14,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "name",
										Value: "email",
										Location: ast_domain.Location{
											Line:   30,
											Column: 32,
										},
										NameLocation: ast_domain.Location{
											Line:   30,
											Column: 26,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "placeholder",
										Value: "Enter email",
										Location: ast_domain.Location{
											Line:   30,
											Column: 52,
										},
										NameLocation: ast_domain.Location{
											Line:   30,
											Column: 39,
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   31,
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
									Value: "r.0:2:1",
									RelativeLocation: ast_domain.Location{
										Line:   31,
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
											Line:   31,
											Column: 21,
										},
										NameLocation: ast_domain.Location{
											Line:   31,
											Column: 15,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   31,
											Column: 29,
										},
										TextContent: "Submit Static",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   31,
												Column: 29,
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
						NodeType: ast_domain.NodeFragment,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "boxed_form_show_form_true_403a7678",
								PartialAlias:        "boxed_form",
								PartialPackageName:  "partials_boxed_form_f1ded90b",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   34,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"show_form": ast_domain.PropValue{
										Expression: &ast_domain.BooleanLiteral{
											Value: true,
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
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
											Line:   34,
											Column: 53,
										},
										GoFieldName: "ShowForm",
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
							Value: "r.0:3",
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
								OriginalSourcePath: new("partials/boxed_form.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 3,
								},
								TagName: "form",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_boxed_form_f1ded90b"),
									OriginalSourcePath:   new("partials/boxed_form.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   22,
										Column: 15,
									},
									NameLocation: ast_domain.Location{
										Line:   22,
										Column: 9,
									},
									RawExpression: "props.ShowForm",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "props",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("partials_boxed_form_f1ded90b.Props"),
													PackageAlias:         "partials_boxed_form_f1ded90b",
													CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "props",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 15,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("props_boxed_form_show_form_true_403a7678"),
												OriginalSourcePath: new("partials/boxed_form.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "ShowForm",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "partials_boxed_form_f1ded90b",
													CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ShowForm",
													ReferenceLocation: ast_domain.Location{
														Line:   22,
														Column: 15,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
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
												BaseCodeGenVarName:  new("props_boxed_form_show_form_true_403a7678"),
												OriginalSourcePath:  new("partials/boxed_form.pk"),
												GeneratedSourcePath: new("dist/partials/partials_boxed_form_f1ded90b/generated.go"),
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
												PackageAlias:         "partials_boxed_form_f1ded90b",
												CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ShowForm",
												ReferenceLocation: ast_domain.Location{
													Line:   22,
													Column: 15,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
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
											BaseCodeGenVarName:  new("props_boxed_form_show_form_true_403a7678"),
											OriginalSourcePath:  new("partials/boxed_form.pk"),
											GeneratedSourcePath: new("dist/partials/partials_boxed_form_f1ded90b/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "partials_boxed_form_f1ded90b",
											CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ShowForm",
											ReferenceLocation: ast_domain.Location{
												Line:   22,
												Column: 15,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   38,
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
										BaseCodeGenVarName:  new("props_boxed_form_show_form_true_403a7678"),
										OriginalSourcePath:  new("partials/boxed_form.pk"),
										GeneratedSourcePath: new("dist/partials/partials_boxed_form_f1ded90b/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
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
										OriginalSourcePath: new("partials/boxed_form.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "boxed-form",
										Location: ast_domain.Location{
											Line:   22,
											Column: 76,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 69,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "p-fragment",
										Value: "boxed_form_server_show_form_true_35b22e70",
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
								OnEvents: map[string][]ast_domain.Directive{
									"submit": []ast_domain.Directive{
										ast_domain.Directive{
											Type: ast_domain.DirectiveOn,
											Location: ast_domain.Location{
												Line:   22,
												Column: 44,
											},
											NameLocation: ast_domain.Location{
												Line:   22,
												Column: 31,
											},
											Arg:           "submit",
											Modifier:      "action",
											RawExpression: "action.dynamic_action()",
											Expression: &ast_domain.CallExpression{
												Callee: &ast_domain.Identifier{
													Name: "dynamic_action",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   35,
											Column: 7,
										},
										TagName: "input",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   35,
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
												Value: "text",
												Location: ast_domain.Location{
													Line:   35,
													Column: 20,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 14,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "name",
												Value: "name",
												Location: ast_domain.Location{
													Line:   35,
													Column: 32,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 26,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "placeholder",
												Value: "Enter name",
												Location: ast_domain.Location{
													Line:   35,
													Column: 51,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 38,
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
										TagName: "button",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_boxed_form_f1ded90b"),
											OriginalSourcePath:   new("partials/boxed_form.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:0:1",
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
												OriginalSourcePath: new("partials/boxed_form.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "type",
												Value: "submit",
												Location: ast_domain.Location{
													Line:   24,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 13,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 27,
												},
												TextContent: "Submit Dynamic",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_boxed_form_f1ded90b"),
													OriginalSourcePath:   new("partials/boxed_form.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:3:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 27,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/boxed_form.pk"),
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
									Column: 3,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_boxed_form_f1ded90b"),
									OriginalSourcePath:   new("partials/boxed_form.pk"),
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
									RawExpression: "!props.ShowForm",
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
														TypeExpression:       typeExprFromString("partials_boxed_form_f1ded90b.Props"),
														PackageAlias:         "partials_boxed_form_f1ded90b",
														CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 14,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_boxed_form_show_form_true_403a7678"),
													OriginalSourcePath: new("partials/boxed_form.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "ShowForm",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "partials_boxed_form_f1ded90b",
														CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ShowForm",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 14,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
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
													BaseCodeGenVarName:  new("props_boxed_form_show_form_true_403a7678"),
													OriginalSourcePath:  new("partials/boxed_form.pk"),
													GeneratedSourcePath: new("dist/partials/partials_boxed_form_f1ded90b/generated.go"),
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
													PackageAlias:         "partials_boxed_form_f1ded90b",
													CanonicalPackagePath: "testcase_115_form_submit_remote/dist/partials/partials_boxed_form_f1ded90b",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ShowForm",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 14,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
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
												BaseCodeGenVarName:  new("props_boxed_form_show_form_true_403a7678"),
												OriginalSourcePath:  new("partials/boxed_form.pk"),
												GeneratedSourcePath: new("dist/partials/partials_boxed_form_f1ded90b/generated.go"),
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
											OriginalSourcePath: new("partials/boxed_form.pk"),
											Stringability:      1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/boxed_form.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/boxed_form.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:1",
									RelativeLocation: ast_domain.Location{
										Line:   27,
										Column: 3,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/boxed_form.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "boxed-form-no-action",
										Location: ast_domain.Location{
											Line:   27,
											Column: 38,
										},
										NameLocation: ast_domain.Location{
											Line:   27,
											Column: 31,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "p-fragment",
										Value: "boxed_form_server_show_form_true_35b22e70",
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
											Line:   35,
											Column: 7,
										},
										TagName: "input",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   35,
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
												Value: "text",
												Location: ast_domain.Location{
													Line:   35,
													Column: 20,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 14,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "name",
												Value: "name",
												Location: ast_domain.Location{
													Line:   35,
													Column: 32,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 26,
												},
											},
											ast_domain.HTMLAttribute{
												Name:  "placeholder",
												Value: "Enter name",
												Location: ast_domain.Location{
													Line:   35,
													Column: 51,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 38,
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
										TagName: "button",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_boxed_form_f1ded90b"),
											OriginalSourcePath:   new("partials/boxed_form.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1:1",
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
												OriginalSourcePath: new("partials/boxed_form.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "type",
												Value: "submit",
												Location: ast_domain.Location{
													Line:   29,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   29,
													Column: 13,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   29,
													Column: 27,
												},
												TextContent: "No Action",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_boxed_form_f1ded90b"),
													OriginalSourcePath:   new("partials/boxed_form.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:3:1:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   29,
														Column: 27,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/boxed_form.pk"),
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
