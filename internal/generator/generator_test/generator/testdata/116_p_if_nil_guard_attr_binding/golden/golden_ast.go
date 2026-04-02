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
								TextContent: "p-if Nil Guard Attribute Binding Test",
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
								TextContent: " This test verifies that when a p-if directive guards against nil, the generated code should not emit redundant nil-check warnings for attribute bindings that access fields on the guarded pointer. ",
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
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
							OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "floor_plan_modal_662d983e",
								PartialAlias:        "floor_plan_modal",
								PartialPackageName:  "partials_floor_plan_modal_cecbd938",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   30,
									Column: 5,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
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
								OriginalSourcePath: new("partials/floor_plan_modal.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "modal-wrapper",
								Location: ast_domain.Location{
									Line:   22,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 8,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "floorPlanModal",
								Location: ast_domain.Location{
									Line:   22,
									Column: 34,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 30,
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
									OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
									OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
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
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "overlay",
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
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								TagName: "img",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
									OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   24,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   24,
										Column: 10,
									},
									RawExpression: "props.FloorPlan != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_floor_plan_modal_cecbd938.Props"),
														PackageAlias:         "partials_floor_plan_modal_cecbd938",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/dist/partials/partials_floor_plan_modal_cecbd938",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_floor_plan_modal_662d983e"),
													OriginalSourcePath: new("partials/floor_plan_modal.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "FloorPlan",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FloorPlan",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
													OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
													GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
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
													TypeExpression:       typeExprFromString("*pkg.Image"),
													PackageAlias:         "pkg",
													CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FloorPlan",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
												OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
												GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/floor_plan_modal.pk"),
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
											OriginalSourcePath: new("partials/floor_plan_modal.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:1",
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
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "floor-plan-img",
										Location: ast_domain.Location{
											Line:   24,
											Column: 47,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 40,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "src",
										RawExpression: "props.FloorPlan.URL",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_floor_plan_modal_cecbd938.Props"),
															PackageAlias:         "partials_floor_plan_modal_cecbd938",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/dist/partials/partials_floor_plan_modal_cecbd938",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props_floor_plan_modal_662d983e"),
														OriginalSourcePath: new("partials/floor_plan_modal.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FloorPlan",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pkg.Image"),
															PackageAlias:         "pkg",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FloorPlan",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
														OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
														GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
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
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FloorPlan",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
													OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
													GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "URL",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "URL",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   50,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
													OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
													GeneratedSourcePath: new("pkg/types.go"),
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
													PackageAlias:         "pkg",
													CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "URL",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 69,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
												OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
												GeneratedSourcePath: new("pkg/types.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 69,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 63,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pkg",
												CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "URL",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_floor_plan_modal_662d983e"),
											OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
											GeneratedSourcePath: new("pkg/types.go"),
											Stringability:       1,
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
								TagName: "img",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
									OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
									IsStructurallyStatic: true,
								},
								DirElse: &ast_domain.Directive{
									Type: ast_domain.DirectiveElse,
									Location: ast_domain.Location{
										Line:   25,
										Column: 10,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 10,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
									},
									ChainKey: &ast_domain.StringLiteral{
										Value: "r.0:2:1",
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
											OriginalSourcePath: new("partials/floor_plan_modal.pk"),
											Stringability:      1,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:2",
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
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "floor-plan-img",
										Location: ast_domain.Location{
											Line:   25,
											Column: 24,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 17,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "src",
										Value: "/placeholder.jpg",
										Location: ast_domain.Location{
											Line:   25,
											Column: 45,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 40,
										},
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
							OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
							OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "floor_plan_modal_floorplan_state_testimage_151d2dcf",
								PartialAlias:        "floor_plan_modal",
								PartialPackageName:  "partials_floor_plan_modal_cecbd938",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   32,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"floorplan": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.PageData"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 60,
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
												Name: "TestImage",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TestImage",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 60,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pkg.Image"),
															PackageAlias:         "pkg",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TestImage",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 60,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
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
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pkg.Image"),
													PackageAlias:         "pkg",
													CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "TestImage",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 60,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 23,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TestImage",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 60,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Location: ast_domain.Location{
											Line:   32,
											Column: 60,
										},
										GoFieldName: "FloorPlan",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*pkg.Image"),
												PackageAlias:         "pkg",
												CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "TestImage",
												ReferenceLocation: ast_domain.Location{
													Line:   32,
													Column: 60,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pkg.Image"),
													PackageAlias:         "pkg",
													CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "TestImage",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 60,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
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
								OriginalSourcePath: new("partials/floor_plan_modal.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "modal-wrapper",
								Location: ast_domain.Location{
									Line:   22,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 8,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "floorPlanModal",
								Location: ast_domain.Location{
									Line:   22,
									Column: 34,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 30,
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
									OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
									OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
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
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "overlay",
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
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								TagName: "img",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
									OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   24,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   24,
										Column: 10,
									},
									RawExpression: "props.FloorPlan != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_floor_plan_modal_cecbd938.Props"),
														PackageAlias:         "partials_floor_plan_modal_cecbd938",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/dist/partials/partials_floor_plan_modal_cecbd938",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
													OriginalSourcePath: new("partials/floor_plan_modal.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "FloorPlan",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FloorPlan",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pkg.Image"),
															PackageAlias:         "pkg",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TestImage",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 60,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
													OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
													GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
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
													TypeExpression:       typeExprFromString("*pkg.Image"),
													PackageAlias:         "pkg",
													CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FloorPlan",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TestImage",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 60,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
												OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
												GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/floor_plan_modal.pk"),
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
											OriginalSourcePath: new("partials/floor_plan_modal.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:1",
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
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "floor-plan-img",
										Location: ast_domain.Location{
											Line:   24,
											Column: 47,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 40,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "src",
										RawExpression: "props.FloorPlan.URL",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("partials_floor_plan_modal_cecbd938.Props"),
															PackageAlias:         "partials_floor_plan_modal_cecbd938",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/dist/partials/partials_floor_plan_modal_cecbd938",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
														OriginalSourcePath: new("partials/floor_plan_modal.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FloorPlan",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pkg.Image"),
															PackageAlias:         "pkg",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FloorPlan",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pkg.Image"),
																PackageAlias:         "pkg",
																CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "TestImage",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 60,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   46,
																	Column: 23,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
														OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
														GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
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
														TypeExpression:       typeExprFromString("*pkg.Image"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FloorPlan",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pkg.Image"),
															PackageAlias:         "pkg",
															CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "TestImage",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 60,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
													OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
													GeneratedSourcePath: new("dist/partials/partials_floor_plan_modal_cecbd938/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "URL",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pkg",
														CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "URL",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   50,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
													OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
													GeneratedSourcePath: new("pkg/types.go"),
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
													PackageAlias:         "pkg",
													CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "URL",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 69,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   50,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
												OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
												GeneratedSourcePath: new("pkg/types.go"),
												Stringability:       1,
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 69,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 63,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pkg",
												CanonicalPackagePath: "testcase_116_p_if_nil_guard_attr_binding/pkg",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "URL",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   50,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_floor_plan_modal_floorplan_state_testimage_151d2dcf"),
											OriginalSourcePath:  new("partials/floor_plan_modal.pk"),
											GeneratedSourcePath: new("pkg/types.go"),
											Stringability:       1,
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
								TagName: "img",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_floor_plan_modal_cecbd938"),
									OriginalSourcePath:   new("partials/floor_plan_modal.pk"),
									IsStructurallyStatic: true,
								},
								DirElse: &ast_domain.Directive{
									Type: ast_domain.DirectiveElse,
									Location: ast_domain.Location{
										Line:   25,
										Column: 10,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 10,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
									},
									ChainKey: &ast_domain.StringLiteral{
										Value: "r.0:3:1",
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
											OriginalSourcePath: new("partials/floor_plan_modal.pk"),
											Stringability:      1,
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:2",
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
										OriginalSourcePath: new("partials/floor_plan_modal.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "floor-plan-img",
										Location: ast_domain.Location{
											Line:   25,
											Column: 24,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 17,
										},
									},
									ast_domain.HTMLAttribute{
										Name:  "src",
										Value: "/placeholder.jpg",
										Location: ast_domain.Location{
											Line:   25,
											Column: 45,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 40,
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
