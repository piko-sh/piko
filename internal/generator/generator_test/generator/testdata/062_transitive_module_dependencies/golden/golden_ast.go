package testcase_062_transitive_module_dependencies

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
								TextContent: "Transitive Module Dependency Test",
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
								TextContent: "This page imports ActionButton from composite-widgets, which internally imports Button from ui-components.",
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
								TextContent: "This tests the full dependency chain: page → composite-widgets → ui-components",
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
							Line:   28,
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
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   28,
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
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81"),
							OriginalSourcePath:   new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "action_btn_1239629d",
								PartialAlias:        "action_btn",
								PartialPackageName:  "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   30,
									Column: 5,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
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
								OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "action-button-wrapper",
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
								TagName: "label",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81"),
									OriginalSourcePath:   new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   23,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   23,
										Column: 12,
									},
									RawExpression: "state.LabelText != ``",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81.Response"),
														PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
														CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
													OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "LabelText",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
														CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "LabelText",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   45,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
													OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
													GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
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
													PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "LabelText",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   45,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
												OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
												Stringability:       1,
											},
										},
										Operator: "!=",
										Right: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
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
											OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
										Stringability:      1,
									},
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   23,
										Column: 49,
									},
									NameLocation: ast_domain.Location{
										Line:   23,
										Column: 41,
									},
									RawExpression: "state.LabelText",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81.Response"),
													PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 49,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
												OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "LabelText",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "LabelText",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 49,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   45,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
												OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
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
												PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "LabelText",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 49,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
											OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
											GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
											CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "LabelText",
											ReferenceLocation: ast_domain.Location{
												Line:   23,
												Column: 49,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   45,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
										OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
										GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
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
										OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
										Stringability:      1,
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
									OriginalPackageAlias: new("testdata_modules_ui_components_components_button_127f2b1d"),
									OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/Button.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "btn_action_btn_1239629d_04e13e48",
										PartialAlias:        "btn",
										PartialPackageName:  "testdata_modules_ui_components_components_button_127f2b1d",
										InvokerPackageAlias: "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
										Location: ast_domain.Location{
											Line:   24,
											Column: 5,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:1",
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
										OriginalSourcePath: new("../../../testdata-modules/ui-components/components/Button.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "btn btn-primary",
										Location: ast_domain.Location{
											Line:   22,
											Column: 18,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 11,
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
											OriginalPackageAlias: new("testdata_modules_ui_components_components_button_127f2b1d"),
											OriginalSourcePath:   new("../../../testdata-modules/ui-components/components/Button.pk"),
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
											RawExpression: "state.Label",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("testdata_modules_ui_components_components_button_127f2b1d.Response"),
															PackageAlias:         "testdata_modules_ui_components_components_button_127f2b1d",
															CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_ui_components_components_button_127f2b1d",
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
														BaseCodeGenVarName: new("testdata_modules_ui_components_components_button_127f2b1dData_btn_action_btn_1239629d_04e13e48"),
														OriginalSourcePath: new("../../../testdata-modules/ui-components/components/Button.pk"),
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
															PackageAlias:         "testdata_modules_ui_components_components_button_127f2b1d",
															CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_ui_components_components_button_127f2b1d",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 19,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 23,
															},
														},
														BaseCodeGenVarName:  new("testdata_modules_ui_components_components_button_127f2b1dData_btn_action_btn_1239629d_04e13e48"),
														OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/Button.pk"),
														GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_button_127f2b1d/generated.go"),
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
														PackageAlias:         "testdata_modules_ui_components_components_button_127f2b1d",
														CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_ui_components_components_button_127f2b1d",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("testdata_modules_ui_components_components_button_127f2b1dData_btn_action_btn_1239629d_04e13e48"),
													OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/Button.pk"),
													GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_button_127f2b1d/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "testdata_modules_ui_components_components_button_127f2b1d",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_ui_components_components_button_127f2b1d",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   32,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_ui_components_components_button_127f2b1dData_btn_action_btn_1239629d_04e13e48"),
												OriginalSourcePath:  new("../../../testdata-modules/ui-components/components/Button.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_ui_components_components_button_127f2b1d/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:1:0",
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
												OriginalSourcePath: new("../../../testdata-modules/ui-components/components/Button.pk"),
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
									OriginalPackageAlias: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81"),
									OriginalSourcePath:   new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   25,
										Column: 32,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 26,
									},
									RawExpression: "state.HelpText != ``",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81.Response"),
														PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
														CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
													OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "HelpText",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
														CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "HelpText",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
													OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
													GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
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
													PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "HelpText",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
												OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
												Stringability:       1,
											},
										},
										Operator: "!=",
										Right: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 19,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
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
											OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
										Stringability:      1,
									},
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   25,
										Column: 62,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 54,
									},
									RawExpression: "state.HelpText",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81.Response"),
													PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 62,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
												OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "HelpText",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
													CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "HelpText",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 62,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
												OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
												GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
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
												PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
												CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "HelpText",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 62,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
											OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
											GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
											CanonicalPackagePath: "testcase_062_transitive_module_dependencies/dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "HelpText",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 62,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   46,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81Data_action_btn_1239629d"),
										OriginalSourcePath:  new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
										GeneratedSourcePath: new("dist/partials/testdata_modules_composite_widgets_widgets_actionbutton_44cbbf81/generated.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:2",
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
										OriginalSourcePath: new("../../../testdata-modules/composite-widgets/widgets/ActionButton.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "help-text",
										Location: ast_domain.Location{
											Line:   25,
											Column: 15,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 8,
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
