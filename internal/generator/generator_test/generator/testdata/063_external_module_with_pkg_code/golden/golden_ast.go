package testcase_063_external_module_with_pkg_code

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
								TextContent: "External Module pkg/ Code Test",
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
							Line:   25,
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
								Line:   25,
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
									Line:   25,
									Column: 8,
								},
								TextContent: "This page uses ButtonWithUtils component from ui-components module.",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   25,
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
							Line:   26,
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
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   26,
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
									Line:   26,
									Column: 8,
								},
								TextContent: "That component imports helper functions from github.com/example/ui-components/pkg/utils.",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   26,
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
							Line:   27,
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
							Value: "r.0:3",
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   27,
									Column: 8,
								},
								TextContent: "This tests that external components can reference their own module's Go packages.",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   27,
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
						TagName: "hr",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
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
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   31,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "button-showcase",
								Location: ast_domain.Location{
									Line:   31,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   31,
									Column: 10,
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
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   32,
											Column: 11,
										},
										TextContent: "Primary Button",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   32,
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
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 3,
								},
								TagName: "button",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0c"),
									OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "btn_0d50974e",
										PartialAlias:        "btn",
										PartialPackageName:  "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   33,
											Column: 7,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:1",
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
										OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "btn",
										Location: ast_domain.Location{
											Line:   22,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 11,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "p-attr:class",
										Value: "state.CssClass",
										Location: ast_domain.Location{
											Line:   22,
											Column: 37,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 23,
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0c"),
											OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.FormattedLabel",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("testdata_modules_ui_components_components_buttonwithutils_9d255c0c.Response"),
															PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
															CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_0d50974e"),
														OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormattedLabel",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
															CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FormattedLabel",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_0d50974e"),
														OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
														GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
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
														PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FormattedLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_0d50974e"),
													OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
													GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
													CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FormattedLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_0d50974e"),
												OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:1:0",
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
												OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
												Stringability:      1,
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   35,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:2",
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   35,
											Column: 11,
										},
										TextContent: "Secondary Button",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   35,
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
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 3,
								},
								TagName: "button",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0c"),
									OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "btn_secondary_412732ba",
										PartialAlias:        "btn_secondary",
										PartialPackageName:  "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   36,
											Column: 7,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:3",
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
										OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "btn",
										Location: ast_domain.Location{
											Line:   22,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 11,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "p-attr:class",
										Value: "state.CssClass",
										Location: ast_domain.Location{
											Line:   22,
											Column: 37,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 23,
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0c"),
											OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.FormattedLabel",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("testdata_modules_ui_components_components_buttonwithutils_9d255c0c.Response"),
															PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
															CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_secondary_412732ba"),
														OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormattedLabel",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
															CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FormattedLabel",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_secondary_412732ba"),
														OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
														GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
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
														PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FormattedLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_secondary_412732ba"),
													OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
													GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
													CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FormattedLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_secondary_412732ba"),
												OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:3:0",
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
												OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
												Stringability:      1,
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   38,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:4",
									RelativeLocation: ast_domain.Location{
										Line:   38,
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
											Line:   38,
											Column: 11,
										},
										TextContent: "Danger Button",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:4:0",
											RelativeLocation: ast_domain.Location{
												Line:   38,
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
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 3,
								},
								TagName: "button",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0c"),
									OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "btn_danger_3a4b2617",
										PartialAlias:        "btn_danger",
										PartialPackageName:  "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   39,
											Column: 7,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:5",
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
										OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "btn",
										Location: ast_domain.Location{
											Line:   22,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 11,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "p-attr:class",
										Value: "state.CssClass",
										Location: ast_domain.Location{
											Line:   22,
											Column: 37,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 23,
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
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0c"),
											OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   23,
												Column: 19,
											},
											NameLocation: ast_domain.Location{
												Line:   23,
												Column: 11,
											},
											RawExpression: "state.FormattedLabel",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("testdata_modules_ui_components_components_buttonwithutils_9d255c0c.Response"),
															PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
															CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_danger_3a4b2617"),
														OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FormattedLabel",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
															CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FormattedLabel",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_danger_3a4b2617"),
														OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
														GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
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
														PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
														CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FormattedLabel",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_danger_3a4b2617"),
													OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
													GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
													CanonicalPackagePath: "testcase_063_external_module_with_pkg_code/dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FormattedLabel",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_ui_components_components_buttonwithutils_9d255c0cData_btn_danger_3a4b2617"),
												OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_buttonwithutils_9d255c0c/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:5:0",
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
												OriginalSourcePath: new("../../../testdata-modules/ui-components/components/ButtonWithUtils.pk"),
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
