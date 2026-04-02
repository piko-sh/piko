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
								TextContent: "Items List",
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
						TagName: "div",
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
								Value: "items",
								Location: ast_domain.Location{
									Line:   24,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 10,
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
								TagName: "article",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_item_card_e11b3960"),
									OriginalSourcePath:   new("partials/item_card.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "item_card_item_item_6c0d4409",
										PartialAlias:        "item_card",
										PartialPackageName:  "partials_item_card_e11b3960",
										InvokerPackageAlias: "pages_main_594861c5",
										Location: ast_domain.Location{
											Line:   25,
											Column: 7,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"item": ast_domain.PropValue{
												Expression: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dto.ItemData"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.ItemData"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   29,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Location: ast_domain.Location{
													Line:   29,
													Column: 23,
												},
												GoFieldName: "Item",
												InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.ItemData"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "item",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dto.ItemData"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
													},
													BaseCodeGenVarName: new("item"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											IsLoopDependent: true,
										},
									},
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   27,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   27,
										Column: 9,
									},
									RawExpression: "item in state.Items",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: nil,
										ItemVariable: &ast_domain.Identifier{
											Name: "item",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.ItemData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "item",
													ReferenceLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("item"),
												OriginalSourcePath: new("partials/item_card.pk"),
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
														CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 16,
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
														TypeExpression:       typeExprFromString("[]dto.ItemData"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 16,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   45,
															Column: 23,
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
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.ItemData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   45,
														Column: 23,
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
												TypeExpression:       typeExprFromString("[]dto.ItemData"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.ItemData"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   27,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]dto.ItemData"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Items",
											ReferenceLocation: ast_domain.Location{
												Line:   27,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   45,
												Column: 23,
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
										Line:   28,
										Column: 16,
									},
									NameLocation: ast_domain.Location{
										Line:   28,
										Column: 9,
									},
									RawExpression: "item.ID",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "item",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.ItemData"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "item",
													ReferenceLocation: ast_domain.Location{
														Line:   38,
														Column: 27,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("item"),
												OriginalSourcePath: new("partials/item_card.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "ID",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 6,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   28,
														Column: 16,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   67,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/dto/dto.go"),
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
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "ID",
												ReferenceLocation: ast_domain.Location{
													Line:   38,
													Column: 27,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   74,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("item"),
											OriginalSourcePath:  new("partials/item_card.pk"),
											GeneratedSourcePath: new("pkg/dto/dto.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "ID",
											ReferenceLocation: ast_domain.Location{
												Line:   28,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   67,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("item"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("pkg/dto/dto.go"),
										Stringability:       1,
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
												OriginalSourcePath: new("partials/item_card.pk"),
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
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dto.ItemData"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 27,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("partials/item_card.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "ID",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   28,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   67,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/dto/dto.go"),
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
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   38,
															Column: 27,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   74,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("partials/item_card.pk"),
													GeneratedSourcePath: new("pkg/dto/dto.go"),
													Stringability:       1,
												},
											},
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
										OriginalSourcePath: new("partials/item_card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "item-card",
										Location: ast_domain.Location{
											Line:   22,
											Column: 19,
										},
										NameLocation: ast_domain.Location{
											Line:   22,
											Column: 12,
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
										TagName: "h2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   28,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: ":0",
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
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-title",
												Location: ast_domain.Location{
													Line:   23,
													Column: 16,
												},
												NameLocation: ast_domain.Location{
													Line:   23,
													Column: 9,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   23,
													Column: 28,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 28,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   23,
																Column: 28,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":0:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   23,
														Column: 28,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   23,
															Column: 31,
														},
														RawExpression: "props.Item.Title",
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
																			TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																			PackageAlias:         "partials_item_card_e11b3960",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   23,
																				Column: 31,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   23,
																				Column: 31,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   50,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   29,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("item"),
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																		TypeExpression:       typeExprFromString("dto.ItemData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 31,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   50,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   29,
																				Column: 23,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Title",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   23,
																			Column: 31,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   75,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   23,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   75,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   75,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
										TagName: "p",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   28,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: ":1",
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
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-description",
												Location: ast_domain.Location{
													Line:   24,
													Column: 15,
												},
												NameLocation: ast_domain.Location{
													Line:   24,
													Column: 8,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   24,
													Column: 33,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   24,
																Column: 33,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   24,
																Column: 33,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":1:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   24,
														Column: 33,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   24,
															Column: 36,
														},
														RawExpression: "props.Item.Description",
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
																			TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																			PackageAlias:         "partials_item_card_e11b3960",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 36,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 36,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   50,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   29,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("item"),
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																		TypeExpression:       typeExprFromString("dto.ItemData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 36,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   50,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   29,
																				Column: 23,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Description",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Description",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 36,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   76,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Description",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 36,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   76,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Description",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   76,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   28,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: ":2",
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
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-details",
												Location: ast_domain.Location{
													Line:   25,
													Column: 17,
												},
												NameLocation: ast_domain.Location{
													Line:   25,
													Column: 10,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   26,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":2:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   26,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "price",
														Location: ast_domain.Location{
															Line:   26,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   26,
															Column: 13,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   26,
															Column: 27,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_card_e11b3960"),
															OriginalSourcePath:   new("partials/item_card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   26,
																		Column: 27,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
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
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   38,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 6,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   67,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   74,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   26,
																		Column: 27,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":2:0:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 27,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   26,
																	Column: 30,
																},
																RawExpression: "props.Item.Price",
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
																					TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																					PackageAlias:         "partials_item_card_e11b3960",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   26,
																						Column: 30,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "Item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   26,
																						Column: 30,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   50,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("item"),
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   26,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   50,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   29,
																						Column: 23,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Price",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 12,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Price",
																				ReferenceLocation: ast_domain.Location{
																					Line:   26,
																					Column: 30,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   77,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																			TypeExpression:       typeExprFromString("int"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Price",
																			ReferenceLocation: ast_domain.Location{
																				Line:   26,
																				Column: 30,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   77,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Price",
																		ReferenceLocation: ast_domain.Location{
																			Line:   26,
																			Column: 30,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   77,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
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
													Line:   27,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":2:1",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   27,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "status",
														Location: ast_domain.Location{
															Line:   27,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   27,
															Column: 13,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   27,
															Column: 28,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_card_e11b3960"),
															OriginalSourcePath:   new("partials/item_card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 28,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
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
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   38,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 6,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   67,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   74,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 28,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":2:1:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 28,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   27,
																	Column: 31,
																},
																RawExpression: "props.Item.Status",
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
																					TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																					PackageAlias:         "partials_item_card_e11b3960",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   27,
																						Column: 31,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "Item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   27,
																						Column: 31,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   50,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("item"),
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   27,
																					Column: 31,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   50,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   29,
																						Column: 23,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Status",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 12,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Status",
																				ReferenceLocation: ast_domain.Location{
																					Line:   27,
																					Column: 31,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   78,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Status",
																			ReferenceLocation: ast_domain.Location{
																				Line:   27,
																				Column: 31,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   78,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Status",
																		ReferenceLocation: ast_domain.Location{
																			Line:   27,
																			Column: 31,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   78,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
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
													Line:   28,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":2:2",
														},
													},
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "category",
														Location: ast_domain.Location{
															Line:   28,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   28,
															Column: 13,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   28,
															Column: 30,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_card_e11b3960"),
															OriginalSourcePath:   new("partials/item_card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   28,
																		Column: 30,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
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
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   38,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 6,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   67,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   74,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   28,
																		Column: 30,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":2:2:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 30,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   28,
																	Column: 33,
																},
																RawExpression: "props.Item.Category",
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
																					TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																					PackageAlias:         "partials_item_card_e11b3960",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "props",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 33,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "Item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 33,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   50,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("item"),
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   28,
																					Column: 33,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   50,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   29,
																						Column: 23,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Category",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 12,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("string"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Category",
																				ReferenceLocation: ast_domain.Location{
																					Line:   28,
																					Column: 33,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   79,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Category",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 33,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   79,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Category",
																		ReferenceLocation: ast_domain.Location{
																			Line:   28,
																			Column: 33,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   79,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
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
											Line:   30,
											Column: 5,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   28,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: ":3",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   30,
												Column: 5,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "rooms",
												Location: ast_domain.Location{
													Line:   30,
													Column: 17,
												},
												NameLocation: ast_domain.Location{
													Line:   30,
													Column: 10,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   31,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":3:0",
														},
													},
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "bedrooms",
														Location: ast_domain.Location{
															Line:   31,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   31,
															Column: 13,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   31,
															Column: 30,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_card_e11b3960"),
															OriginalSourcePath:   new("partials/item_card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   31,
																		Column: 30,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
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
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   38,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 6,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   67,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   74,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   31,
																		Column: 30,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":3:0:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   31,
																Column: 30,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   31,
																	Column: 33,
																},
																RawExpression: "props.Item.RoomSetup.Bedrooms",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "props",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																						PackageAlias:         "partials_item_card_e11b3960",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "props",
																						ReferenceLocation: ast_domain.Location{
																							Line:   31,
																							Column: 33,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																					OriginalSourcePath: new("partials/item_card.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Item",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   31,
																							Column: 33,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   50,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("dto.ItemData"),
																							PackageAlias:         "dto",
																							CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "item",
																							ReferenceLocation: ast_domain.Location{
																								Line:   29,
																								Column: 23,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   0,
																								Column: 0,
																							},
																						},
																						BaseCodeGenVarName: new("item"),
																					},
																					BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																					OriginalSourcePath:  new("partials/item_card.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   31,
																						Column: 33,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   50,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("item"),
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "RoomSetup",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.RoomSetup"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "RoomSetup",
																					ReferenceLocation: ast_domain.Location{
																						Line:   31,
																						Column: 33,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   80,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				TypeExpression:       typeExprFromString("dto.RoomSetup"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "RoomSetup",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 33,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   80,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Bedrooms",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 22,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Bedrooms",
																				ReferenceLocation: ast_domain.Location{
																					Line:   31,
																					Column: 33,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   63,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																			TypeExpression:       typeExprFromString("int"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Bedrooms",
																			ReferenceLocation: ast_domain.Location{
																				Line:   31,
																				Column: 33,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   63,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Bedrooms",
																		ReferenceLocation: ast_domain.Location{
																			Line:   31,
																			Column: 33,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   63,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   31,
																	Column: 65,
																},
																Literal: " beds",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/item_card.pk"),
																},
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   32,
													Column: 7,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":3:1",
														},
													},
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "bathrooms",
														Location: ast_domain.Location{
															Line:   32,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   32,
															Column: 13,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   32,
															Column: 31,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_card_e11b3960"),
															OriginalSourcePath:   new("partials/item_card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   32,
																		Column: 31,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
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
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   38,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 6,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   67,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   74,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   32,
																		Column: 31,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":3:1:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   32,
																Column: 31,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   32,
																	Column: 34,
																},
																RawExpression: "props.Item.RoomSetup.Bathrooms",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "props",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																						PackageAlias:         "partials_item_card_e11b3960",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "props",
																						ReferenceLocation: ast_domain.Location{
																							Line:   32,
																							Column: 34,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																					OriginalSourcePath: new("partials/item_card.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Item",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   32,
																							Column: 34,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   50,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("dto.ItemData"),
																							PackageAlias:         "dto",
																							CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "item",
																							ReferenceLocation: ast_domain.Location{
																								Line:   29,
																								Column: 23,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   0,
																								Column: 0,
																							},
																						},
																						BaseCodeGenVarName: new("item"),
																					},
																					BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																					OriginalSourcePath:  new("partials/item_card.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   32,
																						Column: 34,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   50,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("item"),
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "RoomSetup",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.RoomSetup"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "RoomSetup",
																					ReferenceLocation: ast_domain.Location{
																						Line:   32,
																						Column: 34,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   80,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				TypeExpression:       typeExprFromString("dto.RoomSetup"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "RoomSetup",
																				ReferenceLocation: ast_domain.Location{
																					Line:   32,
																					Column: 34,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   80,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Bathrooms",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 22,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Bathrooms",
																				ReferenceLocation: ast_domain.Location{
																					Line:   32,
																					Column: 34,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   64,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																			TypeExpression:       typeExprFromString("int"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Bathrooms",
																			ReferenceLocation: ast_domain.Location{
																				Line:   32,
																				Column: 34,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   64,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Bathrooms",
																		ReferenceLocation: ast_domain.Location{
																			Line:   32,
																			Column: 34,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   64,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   32,
																	Column: 67,
																},
																Literal: " baths",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/item_card.pk"),
																},
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
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":3:2",
														},
													},
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "parking",
														Location: ast_domain.Location{
															Line:   33,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   33,
															Column: 13,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   33,
															Column: 29,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_item_card_e11b3960"),
															OriginalSourcePath:   new("partials/item_card.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   33,
																		Column: 29,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
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
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "item",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   38,
																						Column: 27,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																				OriginalSourcePath: new("partials/item_card.pk"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "ID",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 6,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("string"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "ID",
																					ReferenceLocation: ast_domain.Location{
																						Line:   28,
																						Column: 16,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   67,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("item"),
																				OriginalSourcePath:  new("pages/main.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   38,
																					Column: 27,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   74,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																			Stringability:       1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   33,
																		Column: 29,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/item_card.pk"),
																		Stringability:      1,
																	},
																	Literal: ":3:2:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   33,
																Column: 29,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   33,
																	Column: 32,
																},
																RawExpression: "props.Item.RoomSetup.Parking",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "props",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																						PackageAlias:         "partials_item_card_e11b3960",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "props",
																						ReferenceLocation: ast_domain.Location{
																							Line:   33,
																							Column: 32,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																					OriginalSourcePath: new("partials/item_card.pk"),
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "Item",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 7,
																				},
																				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "Item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   33,
																							Column: 32,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   50,
																							Column: 2,
																						},
																					},
																					PropDataSource: &ast_domain.PropDataSource{
																						ResolvedType: &ast_domain.ResolvedTypeInfo{
																							TypeExpression:       typeExprFromString("dto.ItemData"),
																							PackageAlias:         "dto",
																							CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																						},
																						Symbol: &ast_domain.ResolvedSymbol{
																							Name: "item",
																							ReferenceLocation: ast_domain.Location{
																								Line:   29,
																								Column: 23,
																							},
																							DeclarationLocation: ast_domain.Location{
																								Line:   0,
																								Column: 0,
																							},
																						},
																						BaseCodeGenVarName: new("item"),
																					},
																					BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																					OriginalSourcePath:  new("partials/item_card.pk"),
																					GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "Item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   33,
																						Column: 32,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   50,
																						Column: 2,
																					},
																				},
																				PropDataSource: &ast_domain.PropDataSource{
																					ResolvedType: &ast_domain.ResolvedTypeInfo{
																						TypeExpression:       typeExprFromString("dto.ItemData"),
																						PackageAlias:         "dto",
																						CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																					},
																					Symbol: &ast_domain.ResolvedSymbol{
																						Name: "item",
																						ReferenceLocation: ast_domain.Location{
																							Line:   29,
																							Column: 23,
																						},
																						DeclarationLocation: ast_domain.Location{
																							Line:   0,
																							Column: 0,
																						},
																					},
																					BaseCodeGenVarName: new("item"),
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "RoomSetup",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 12,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.RoomSetup"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "RoomSetup",
																					ReferenceLocation: ast_domain.Location{
																						Line:   33,
																						Column: 32,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   80,
																						Column: 2,
																					},
																				},
																				BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																				OriginalSourcePath:  new("partials/item_card.pk"),
																				GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																				TypeExpression:       typeExprFromString("dto.RoomSetup"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "RoomSetup",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 32,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   80,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Parking",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 22,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Parking",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 32,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   65,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																			TypeExpression:       typeExprFromString("int"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Parking",
																			ReferenceLocation: ast_domain.Location{
																				Line:   33,
																				Column: 32,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   65,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
																		Stringability:       1,
																	},
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Parking",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 32,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   65,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   33,
																	Column: 63,
																},
																Literal: " parking",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/item_card.pk"),
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
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
										},
										DirIf: &ast_domain.Directive{
											Type: ast_domain.DirectiveIf,
											Location: ast_domain.Location{
												Line:   35,
												Column: 16,
											},
											NameLocation: ast_domain.Location{
												Line:   35,
												Column: 10,
											},
											RawExpression: "props.Item.OpenViewingData.Date != ''",
											Expression: &ast_domain.BinaryExpression{
												Left: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																		PackageAlias:         "partials_item_card_e11b3960",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "props",
																		ReferenceLocation: ast_domain.Location{
																			Line:   35,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath: new("partials/item_card.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("dto.ItemData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   35,
																			Column: 16,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   50,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   29,
																				Column: 23,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   35,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   50,
																		Column: 2,
																	},
																},
																PropDataSource: &ast_domain.PropDataSource{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("dto.ItemData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   29,
																			Column: 23,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																},
																BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "OpenViewingData",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 12,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "OpenViewingData",
																	ReferenceLocation: ast_domain.Location{
																		Line:   35,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   81,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "OpenViewingData",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   81,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Date",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 28,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Date",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 16,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   69,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Date",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 16,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   69,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
														OriginalSourcePath:  new("partials/item_card.pk"),
														GeneratedSourcePath: new("pkg/dto/dto.go"),
														Stringability:       1,
													},
												},
												Operator: "!=",
												Right: &ast_domain.StringLiteral{
													Value: "",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 36,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
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
													OriginalSourcePath: new("partials/item_card.pk"),
													Stringability:      1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
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
														OriginalSourcePath: new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   28,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: ":4",
												},
											},
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
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "open-viewing",
												Location: ast_domain.Location{
													Line:   35,
													Column: 62,
												},
												NameLocation: ast_domain.Location{
													Line:   35,
													Column: 55,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   35,
													Column: 76,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   35,
																Column: 76,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   35,
																Column: 76,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":4:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   35,
														Column: 76,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: true,
														Location: ast_domain.Location{
															Line:   35,
															Column: 76,
														},
														Literal: "\n      Open viewing on ",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("partials/item_card.pk"),
														},
													},
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   36,
															Column: 26,
														},
														RawExpression: "props.Item.OpenViewingData.Date",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "props",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																				PackageAlias:         "partials_item_card_e11b3960",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "props",
																				ReferenceLocation: ast_domain.Location{
																					Line:   36,
																					Column: 26,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath: new("partials/item_card.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Item",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   36,
																					Column: 26,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   50,
																					Column: 2,
																				},
																			},
																			PropDataSource: &ast_domain.PropDataSource{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("dto.ItemData"),
																					PackageAlias:         "dto",
																					CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "item",
																					ReferenceLocation: ast_domain.Location{
																						Line:   29,
																						Column: 23,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   0,
																						Column: 0,
																					},
																				},
																				BaseCodeGenVarName: new("item"),
																			},
																			BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																			OriginalSourcePath:  new("partials/item_card.pk"),
																			GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   36,
																				Column: 26,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   50,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   29,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("item"),
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "OpenViewingData",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "OpenViewingData",
																			ReferenceLocation: ast_domain.Location{
																				Line:   36,
																				Column: 26,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   81,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		TypeExpression:       typeExprFromString("dto.OpenViewingData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "OpenViewingData",
																		ReferenceLocation: ast_domain.Location{
																			Line:   36,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   81,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Date",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 28,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Date",
																		ReferenceLocation: ast_domain.Location{
																			Line:   36,
																			Column: 26,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   69,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Date",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
																		Column: 26,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   69,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Date",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   69,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
															Stringability:       1,
														},
													},
													ast_domain.TextPart{
														IsLiteral: true,
														Location: ast_domain.Location{
															Line:   36,
															Column: 60,
														},
														Literal: "\n    ",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("partials/item_card.pk"),
														},
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   38,
											Column: 5,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_item_card_e11b3960"),
											OriginalSourcePath:   new("partials/item_card.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   38,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
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
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.ItemData"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 27,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("partials/item_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ID",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   28,
																		Column: 16,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   67,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 27,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   38,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
													Literal: ":5",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   38,
												Column: 5,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/item_card.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "item-id",
												Location: ast_domain.Location{
													Line:   38,
													Column: 18,
												},
												NameLocation: ast_domain.Location{
													Line:   38,
													Column: 11,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   38,
													Column: 27,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_item_card_e11b3960"),
													OriginalSourcePath:   new("partials/item_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   38,
																Column: 27,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
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
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 27,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "ID",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 6,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 16,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   67,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 27,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   38,
																Column: 27,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/item_card.pk"),
																Stringability:      1,
															},
															Literal: ":5:0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   38,
														Column: 27,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/item_card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: true,
														Location: ast_domain.Location{
															Line:   38,
															Column: 27,
														},
														Literal: "ID: ",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("partials/item_card.pk"),
														},
													},
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   38,
															Column: 34,
														},
														RawExpression: "props.Item.ID",
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
																			TypeExpression:       typeExprFromString("partials_item_card_e11b3960.Props"),
																			PackageAlias:         "partials_item_card_e11b3960",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/dist/partials/partials_item_card_e11b3960",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 34,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath: new("partials/item_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Item",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   38,
																				Column: 34,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   50,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("dto.ItemData"),
																				PackageAlias:         "dto",
																				CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   29,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("item"),
																		},
																		BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																		OriginalSourcePath:  new("partials/item_card.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
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
																		TypeExpression:       typeExprFromString("dto.ItemData"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 34,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   50,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("dto.ItemData"),
																			PackageAlias:         "dto",
																			CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "item",
																			ReferenceLocation: ast_domain.Location{
																				Line:   29,
																				Column: 23,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("item"),
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_item_card_e11b3960/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "ID",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 12,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "dto",
																		CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   38,
																			Column: 34,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   74,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																	OriginalSourcePath:  new("partials/item_card.pk"),
																	GeneratedSourcePath: new("pkg/dto/dto.go"),
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
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   38,
																		Column: 34,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   74,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
																OriginalSourcePath:  new("partials/item_card.pk"),
																GeneratedSourcePath: new("pkg/dto/dto.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_095_nested_struct_prop_in_for_loop/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   38,
																	Column: 34,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   74,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_item_card_item_item_6c0d4409"),
															OriginalSourcePath:  new("partials/item_card.pk"),
															GeneratedSourcePath: new("pkg/dto/dto.go"),
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
	}
}()
