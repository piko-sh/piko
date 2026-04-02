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
						TagName: "section",
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
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
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
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 11,
										},
										TextContent: "Explicit Keys",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
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
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   27,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   27,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   27,
												Column: 13,
											},
											RawExpression: "(i, item) in state.IntItems",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: &ast_domain.Identifier{
													Name: "i",
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
															Name: "i",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 2,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("i"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												ItemVariable: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 5,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 14,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
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
														Name: "IntItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "IntItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   150,
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
														Column: 14,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IntItems",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   150,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IntItems",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   150,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IntItems",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   150,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IntItems",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   150,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   27,
												Column: 73,
											},
											NameLocation: ast_domain.Location{
												Line:   27,
												Column: 65,
											},
											RawExpression: "item.Name",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   122,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 73,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   122,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 73,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   122,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   27,
												Column: 56,
											},
											NameLocation: ast_domain.Location{
												Line:   27,
												Column: 49,
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 56,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
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
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ID",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 56,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   121,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ID",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 56,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   121,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "ID",
													ReferenceLocation: ast_domain.Location{
														Line:   27,
														Column: 56,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   121,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
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
													Literal: "r.0:0:1:0.",
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
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 56,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("pages/main.pk"),
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
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 56,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   121,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
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
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ID",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 56,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   121,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   27,
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
									Line:   30,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:2",
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   31,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   31,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   31,
												Column: 13,
											},
											RawExpression: "item in state.StringItems",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.StringItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
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
														OriginalSourcePath: new("pages/main.pk"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
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
														Name: "StringItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.StringItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "StringItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   151,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.StringItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StringItems",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   151,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StringItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringItems",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   151,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StringItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringItems",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   151,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.StringItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StringItems",
													ReferenceLocation: ast_domain.Location{
														Line:   31,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   151,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   31,
												Column: 72,
											},
											NameLocation: ast_domain.Location{
												Line:   31,
												Column: 64,
											},
											RawExpression: "item.Value",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.StringItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 72,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Value",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Value",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 72,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   126,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 72,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   126,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Value",
													ReferenceLocation: ast_domain.Location{
														Line:   31,
														Column: 72,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   126,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   31,
												Column: 54,
											},
											NameLocation: ast_domain.Location{
												Line:   31,
												Column: 47,
											},
											RawExpression: "item.Key",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.StringItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Key",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Key",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   125,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Key",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   125,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Key",
													ReferenceLocation: ast_domain.Location{
														Line:   31,
														Column: 54,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   125,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
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
														Line:   31,
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
													Literal: "r.0:0:2:0.",
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
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.StringItem"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Key",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Key",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   125,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Key",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   125,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   31,
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
									Line:   34,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:3",
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   35,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   35,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   35,
												Column: 13,
											},
											RawExpression: "item in state.FloatItems",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.FloatItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
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
														OriginalSourcePath: new("pages/main.pk"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
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
														Name: "FloatItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.FloatItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "FloatItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   152,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.FloatItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FloatItems",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   152,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.FloatItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FloatItems",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   152,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.FloatItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FloatItems",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   152,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.FloatItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "FloatItems",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   152,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   35,
												Column: 73,
											},
											NameLocation: ast_domain.Location{
												Line:   35,
												Column: 65,
											},
											RawExpression: "item.Name",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.FloatItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   130,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 73,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   130,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 73,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   130,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   35,
												Column: 53,
											},
											NameLocation: ast_domain.Location{
												Line:   35,
												Column: 46,
											},
											RawExpression: "item.Score",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.FloatItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Score",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("float64"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Score",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   129,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														TypeExpression:       typeExprFromString("float64"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Score",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   129,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("float64"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Score",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 53,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   129,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
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
														Line:   35,
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
													Literal: "r.0:0:3:0.",
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
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.FloatItem"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   35,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Score",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("float64"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Score",
																	ReferenceLocation: ast_domain.Location{
																		Line:   35,
																		Column: 53,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   129,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
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
																TypeExpression:       typeExprFromString("float64"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Score",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   129,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   35,
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
									Line:   38,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:4",
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   39,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   39,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   39,
												Column: 13,
											},
											RawExpression: "item in state.BoolItems",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.BoolItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
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
														OriginalSourcePath: new("pages/main.pk"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   39,
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
														Name: "BoolItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.BoolItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "BoolItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   39,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   153,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.BoolItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "BoolItems",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   153,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.BoolItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "BoolItems",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   153,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.BoolItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "BoolItems",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   153,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.BoolItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "BoolItems",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   153,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   39,
												Column: 73,
											},
											NameLocation: ast_domain.Location{
												Line:   39,
												Column: 65,
											},
											RawExpression: "item.Label",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.BoolItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Label",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Label",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 73,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   134,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Label",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 73,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   134,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Label",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 73,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   134,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   39,
												Column: 52,
											},
											NameLocation: ast_domain.Location{
												Line:   39,
												Column: 45,
											},
											RawExpression: "item.Active",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.BoolItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 52,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Active",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Active",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 52,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   133,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Active",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 52,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   133,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("bool"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Active",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 52,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   133,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
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
														Line:   39,
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
													Literal: "r.0:0:4:0.",
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
															Name: "item",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("pages_main_594861c5.BoolItem"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "item",
																	ReferenceLocation: ast_domain.Location{
																		Line:   39,
																		Column: 52,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("item"),
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Active",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("bool"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Active",
																	ReferenceLocation: ast_domain.Location{
																		Line:   39,
																		Column: 52,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   133,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
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
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Active",
																ReferenceLocation: ast_domain.Location{
																	Line:   39,
																	Column: 52,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   133,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("item"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   39,
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
									Line:   42,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:5",
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   43,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   43,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   43,
												Column: 13,
											},
											RawExpression: "item in state.StructItems",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
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
														OriginalSourcePath: new("pages/main.pk"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
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
														Name: "StructItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "StructItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   154,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StructItems",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   154,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StructItems",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   154,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StructItems",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   154,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StructItems",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   154,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   43,
												Column: 68,
											},
											NameLocation: ast_domain.Location{
												Line:   43,
												Column: 60,
											},
											RawExpression: "item.Title",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 68,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Title",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   43,
																Column: 68,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   137,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 68,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   137,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 68,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   137,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   43,
												Column: 54,
											},
											NameLocation: ast_domain.Location{
												Line:   43,
												Column: 47,
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
														TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "item",
														ReferenceLocation: ast_domain.Location{
															Line:   43,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("item"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "item",
													ReferenceLocation: ast_domain.Location{
														Line:   43,
														Column: 54,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("item"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   43,
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
													Literal: "r.0:0:5:0.",
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   43,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   43,
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
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   47,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   47,
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
									Line:   48,
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
									Value: "r.0:1:0",
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
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   48,
											Column: 11,
										},
										TextContent: "Inferred Keys",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   48,
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
									Line:   50,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
									RelativeLocation: ast_domain.Location{
										Line:   50,
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
											Line:   51,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   51,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   51,
												Column: 13,
											},
											RawExpression: "(idx, val) in state.SimpleStrings",
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
																Line:   51,
																Column: 20,
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
												ItemVariable: &ast_domain.Identifier{
													Name: "val",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "val",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("val"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
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
														Name: "SimpleStrings",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 21,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "SimpleStrings",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   155,
																	Column: 2,
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
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "SimpleStrings",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   155,
																Column: 2,
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SimpleStrings",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   155,
															Column: 2,
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SimpleStrings",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   155,
															Column: 2,
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
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "SimpleStrings",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   155,
														Column: 2,
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
												Line:   51,
												Column: 63,
											},
											NameLocation: ast_domain.Location{
												Line:   51,
												Column: 55,
											},
											RawExpression: "val",
											Expression: &ast_domain.Identifier{
												Name: "val",
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
														Name: "val",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 63,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("val"),
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
													Name: "val",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 63,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("val"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   51,
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
													Literal: "r.0:1:1:0.",
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
																	Line:   51,
																	Column: 20,
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
												Line:   51,
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
									Line:   54,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:2",
									RelativeLocation: ast_domain.Location{
										Line:   54,
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
											Line:   55,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   55,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   55,
												Column: 13,
											},
											RawExpression: "val in state.SimpleStrings",
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
													Name: "val",
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
															Name: "val",
															ReferenceLocation: ast_domain.Location{
																Line:   55,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("val"),
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   55,
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
														Name: "SimpleStrings",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 14,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "SimpleStrings",
																ReferenceLocation: ast_domain.Location{
																	Line:   55,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   155,
																	Column: 2,
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
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "SimpleStrings",
															ReferenceLocation: ast_domain.Location{
																Line:   55,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   155,
																Column: 2,
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SimpleStrings",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   155,
															Column: 2,
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "SimpleStrings",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   155,
															Column: 2,
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
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "SimpleStrings",
													ReferenceLocation: ast_domain.Location{
														Line:   55,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   155,
														Column: 2,
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
												Line:   55,
												Column: 56,
											},
											NameLocation: ast_domain.Location{
												Line:   55,
												Column: 48,
											},
											RawExpression: "val",
											Expression: &ast_domain.Identifier{
												Name: "val",
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
														Name: "val",
														ReferenceLocation: ast_domain.Location{
															Line:   55,
															Column: 56,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("val"),
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
													Name: "val",
													ReferenceLocation: ast_domain.Location{
														Line:   55,
														Column: 56,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("val"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   55,
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
													Literal: "r.0:1:2:0.",
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
														Name: "val",
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
																Name: "val",
																ReferenceLocation: ast_domain.Location{
																	Line:   55,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("val"),
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   55,
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
									Line:   58,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:3",
									RelativeLocation: ast_domain.Location{
										Line:   58,
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
											Line:   59,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   59,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   59,
												Column: 13,
											},
											RawExpression: "item in state.StructItems",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
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
														Name: "StructItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "StructItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   154,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StructItems",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   154,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StructItems",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   154,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StructItems",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   154,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StructItems",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   154,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   59,
												Column: 55,
											},
											NameLocation: ast_domain.Location{
												Line:   59,
												Column: 47,
											},
											RawExpression: "item.Title",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 55,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Title",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 55,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   137,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 55,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   137,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 55,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   137,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
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
														Line:   59,
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
													Literal: "r.0:1:3:0.",
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
																TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   59,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   59,
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
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   63,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   63,
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
									Line:   64,
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
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   64,
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
											Line:   64,
											Column: 11,
										},
										TextContent: "Nested Loops",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   64,
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
									Line:   66,
									Column: 7,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   66,
										Column: 19,
									},
									NameLocation: ast_domain.Location{
										Line:   66,
										Column: 12,
									},
									RawExpression: "(sectionIdx, section) in state.Sections",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: &ast_domain.Identifier{
											Name: "sectionIdx",
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
													Name: "sectionIdx",
													ReferenceLocation: ast_domain.Location{
														Line:   69,
														Column: 52,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("sectionIdx"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										ItemVariable: &ast_domain.Identifier{
											Name: "section",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 14,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Section"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "section",
													ReferenceLocation: ast_domain.Location{
														Line:   1,
														Column: 14,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("section"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 26,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   66,
															Column: 19,
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
												Name: "Sections",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 32,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.Section"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Sections",
														ReferenceLocation: ast_domain.Location{
															Line:   66,
															Column: 19,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   156,
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
												Column: 26,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.Section"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Sections",
													ReferenceLocation: ast_domain.Location{
														Line:   66,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   156,
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
												TypeExpression:       typeExprFromString("[]pages_main_594861c5.Section"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Sections",
												ReferenceLocation: ast_domain.Location{
													Line:   66,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   156,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]pages_main_594861c5.Section"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Sections",
												ReferenceLocation: ast_domain.Location{
													Line:   66,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   156,
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
											TypeExpression:       typeExprFromString("[]pages_main_594861c5.Section"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Sections",
											ReferenceLocation: ast_domain.Location{
												Line:   66,
												Column: 19,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   156,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   66,
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
											Literal: "r.0:2:1.",
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
												Name: "sectionIdx",
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
														Name: "sectionIdx",
														ReferenceLocation: ast_domain.Location{
															Line:   69,
															Column: 52,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("sectionIdx"),
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   66,
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
											Line:   67,
											Column: 9,
										},
										TagName: "h3",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   67,
												Column: 21,
											},
											NameLocation: ast_domain.Location{
												Line:   67,
												Column: 13,
											},
											RawExpression: "section.Name",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "section",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Section"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "section",
															ReferenceLocation: ast_domain.Location{
																Line:   67,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("section"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   67,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   146,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("section"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   67,
															Column: 21,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   146,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("section"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   67,
														Column: 21,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   146,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("section"),
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
														Line:   67,
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
													Literal: "r.0:2:1.",
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
														Name: "sectionIdx",
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
																Name: "sectionIdx",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
																	Column: 52,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("sectionIdx"),
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   67,
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
													Literal: ":0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   67,
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
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   68,
											Column: 9,
										},
										TagName: "ul",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   68,
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
													Literal: "r.0:2:1.",
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
														Name: "sectionIdx",
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
																Name: "sectionIdx",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
																	Column: 52,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("sectionIdx"),
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   68,
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
													Literal: ":1",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   68,
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
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   69,
													Column: 11,
												},
												TagName: "li",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												DirFor: &ast_domain.Directive{
													Type: ast_domain.DirectiveFor,
													Location: ast_domain.Location{
														Line:   69,
														Column: 22,
													},
													NameLocation: ast_domain.Location{
														Line:   69,
														Column: 15,
													},
													RawExpression: "item in section.Items",
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
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
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
																OriginalSourcePath: new("pages/main.pk"),
															},
														},
														Collection: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "section",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 9,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("pages_main_594861c5.Section"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "section",
																		ReferenceLocation: ast_domain.Location{
																			Line:   69,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("section"),
																	OriginalSourcePath: new("pages/main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Items",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 17,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("[]string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Items",
																		ReferenceLocation: ast_domain.Location{
																			Line:   69,
																			Column: 22,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   147,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("section"),
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
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Items",
																	ReferenceLocation: ast_domain.Location{
																		Line:   69,
																		Column: 22,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   147,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("section"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Items",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
																	Column: 22,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   147,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("section"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       5,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Items",
																ReferenceLocation: ast_domain.Location{
																	Line:   69,
																	Column: 22,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   147,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("section"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															Stringability:       5,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Items",
															ReferenceLocation: ast_domain.Location{
																Line:   69,
																Column: 22,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   147,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("section"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       5,
													},
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   69,
														Column: 66,
													},
													NameLocation: ast_domain.Location{
														Line:   69,
														Column: 58,
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
																	Line:   69,
																	Column: 66,
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
																Line:   69,
																Column: 66,
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
												DirKey: &ast_domain.Directive{
													Type: ast_domain.DirectiveKey,
													Location: ast_domain.Location{
														Line:   69,
														Column: 52,
													},
													NameLocation: ast_domain.Location{
														Line:   69,
														Column: 45,
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
																	Line:   69,
																	Column: 52,
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
																Line:   69,
																Column: 52,
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
																Line:   69,
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
															Literal: "r.0:2:1.",
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
																Name: "sectionIdx",
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
																		Name: "sectionIdx",
																		ReferenceLocation: ast_domain.Location{
																			Line:   69,
																			Column: 52,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("sectionIdx"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   69,
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
															Literal: ":1:0.",
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
																			Line:   69,
																			Column: 52,
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
														Line:   69,
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
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   74,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   74,
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
									Line:   75,
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
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   75,
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
											Line:   75,
											Column: 11,
										},
										TextContent: "Static Keys",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   75,
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
									Line:   77,
									Column: 7,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								DirKey: &ast_domain.Directive{
									Type: ast_domain.DirectiveKey,
									Location: ast_domain.Location{
										Line:   77,
										Column: 19,
									},
									NameLocation: ast_domain.Location{
										Line:   77,
										Column: 12,
									},
									RawExpression: "'static-key-1'",
									Expression: &ast_domain.StringLiteral{
										Value: "static-key-1",
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:1.static-key-1",
									RelativeLocation: ast_domain.Location{
										Line:   77,
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
											Line:   77,
											Column: 35,
										},
										TextContent: "Static Content 1",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:1.static-key-1:0",
											RelativeLocation: ast_domain.Location{
												Line:   77,
												Column: 35,
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
									Line:   78,
									Column: 7,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								DirKey: &ast_domain.Directive{
									Type: ast_domain.DirectiveKey,
									Location: ast_domain.Location{
										Line:   78,
										Column: 19,
									},
									NameLocation: ast_domain.Location{
										Line:   78,
										Column: 12,
									},
									RawExpression: "'static-key-2'",
									Expression: &ast_domain.StringLiteral{
										Value: "static-key-2",
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:2.static-key-2",
									RelativeLocation: ast_domain.Location{
										Line:   78,
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
											Line:   78,
											Column: 35,
										},
										TextContent: "Static Content 2",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:3:2.static-key-2:0",
											RelativeLocation: ast_domain.Location{
												Line:   78,
												Column: 35,
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
							Line:   81,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   81,
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
									Line:   82,
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
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   82,
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
											Line:   82,
											Column: 11,
										},
										TextContent: "Expression Keys",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:4:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   82,
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
									Line:   84,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:1",
									RelativeLocation: ast_domain.Location{
										Line:   84,
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
											Line:   85,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   85,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   85,
												Column: 13,
											},
											RawExpression: "(i, item) in state.IntItems",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: &ast_domain.Identifier{
													Name: "i",
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
															Name: "i",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 2,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("i"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												ItemVariable: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 5,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 14,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   85,
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
														Name: "IntItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "IntItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   85,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   150,
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
														Column: 14,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IntItems",
															ReferenceLocation: ast_domain.Location{
																Line:   85,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   150,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IntItems",
														ReferenceLocation: ast_domain.Location{
															Line:   85,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   150,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IntItems",
														ReferenceLocation: ast_domain.Location{
															Line:   85,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   150,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.IntItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IntItems",
													ReferenceLocation: ast_domain.Location{
														Line:   85,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   150,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   85,
												Column: 97,
											},
											NameLocation: ast_domain.Location{
												Line:   85,
												Column: 89,
											},
											RawExpression: "item.Name",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   85,
																Column: 97,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Name",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   85,
																Column: 97,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   122,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   85,
															Column: 97,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   122,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   85,
														Column: 97,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   122,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   85,
												Column: 56,
											},
											NameLocation: ast_domain.Location{
												Line:   85,
												Column: 49,
											},
											RawExpression: "'item-' + strconv.Itoa(item.ID)",
											Expression: &ast_domain.BinaryExpression{
												Left: &ast_domain.StringLiteral{
													Value: "item-",
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
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Operator: "+",
												Right: &ast_domain.CallExpression{
													Callee: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "strconv",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 11,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       nil,
																	PackageAlias:         "strconv",
																	CanonicalPackagePath: "strconv",
																},
																BaseCodeGenVarName: new("strconv"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Itoa",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 19,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "strconv",
																	CanonicalPackagePath: "strconv",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Itoa",
																	ReferenceLocation: ast_domain.Location{
																		Line:   85,
																		Column: 56,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("strconv"),
															},
														},
														Optional: false,
														Computed: false,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 11,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "strconv",
																CanonicalPackagePath: "strconv",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Itoa",
																ReferenceLocation: ast_domain.Location{
																	Line:   85,
																	Column: 56,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("strconv"),
														},
													},
													Args: []ast_domain.Expression{
														&ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 24,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   85,
																			Column: 56,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																	OriginalSourcePath: new("pages/main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "ID",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 29,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "ID",
																		ReferenceLocation: ast_domain.Location{
																			Line:   85,
																			Column: 56,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   121,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	Stringability:       1,
																},
															},
															Optional: false,
															Computed: false,
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 24,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("int"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   85,
																		Column: 56,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   121,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																Stringability:       1,
															},
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 11,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
														Stringability:      1,
													},
												},
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   85,
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
													Literal: "r.0:4:1:0.",
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
													Expression: &ast_domain.BinaryExpression{
														Left: &ast_domain.StringLiteral{
															Value: "item-",
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
																OriginalSourcePath: new("pages/main.pk"),
																Stringability:      1,
															},
														},
														Operator: "+",
														Right: &ast_domain.CallExpression{
															Callee: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "strconv",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 11,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       nil,
																			PackageAlias:         "strconv",
																			CanonicalPackagePath: "strconv",
																		},
																		BaseCodeGenVarName: new("strconv"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Itoa",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 19,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("function"),
																			PackageAlias:         "strconv",
																			CanonicalPackagePath: "strconv",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Itoa",
																			ReferenceLocation: ast_domain.Location{
																				Line:   85,
																				Column: 56,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("strconv"),
																	},
																},
																Optional: false,
																Computed: false,
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 11,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("function"),
																		PackageAlias:         "strconv",
																		CanonicalPackagePath: "strconv",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Itoa",
																		ReferenceLocation: ast_domain.Location{
																			Line:   85,
																			Column: 56,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("strconv"),
																},
															},
															Args: []ast_domain.Expression{
																&ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "item",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 24,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("pages_main_594861c5.IntItem"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "item",
																				ReferenceLocation: ast_domain.Location{
																					Line:   85,
																					Column: 56,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("item"),
																			OriginalSourcePath: new("pages/main.pk"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "ID",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 29,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "ID",
																				ReferenceLocation: ast_domain.Location{
																					Line:   85,
																					Column: 56,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   121,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("item"),
																			OriginalSourcePath:  new("pages/main.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																			Stringability:       1,
																		},
																	},
																	Optional: false,
																	Computed: false,
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 24,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("int"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "ID",
																			ReferenceLocation: ast_domain.Location{
																				Line:   85,
																				Column: 56,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   121,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("item"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																		Stringability:       1,
																	},
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 11,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "strconv",
																	CanonicalPackagePath: "strconv",
																},
																BaseCodeGenVarName: new("strconv"),
																Stringability:      1,
															},
														},
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
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   85,
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
									Line:   88,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:2",
									RelativeLocation: ast_domain.Location{
										Line:   88,
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
											Line:   89,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   89,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   89,
												Column: 13,
											},
											RawExpression: "item in state.StructItems",
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
															TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
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
														OriginalSourcePath: new("pages/main.pk"),
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
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   89,
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
														Name: "StructItems",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "StructItems",
																ReferenceLocation: ast_domain.Location{
																	Line:   89,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   154,
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
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StructItems",
															ReferenceLocation: ast_domain.Location{
																Line:   89,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   154,
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
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StructItems",
														ReferenceLocation: ast_domain.Location{
															Line:   89,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   154,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StructItems",
														ReferenceLocation: ast_domain.Location{
															Line:   89,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   154,
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
													TypeExpression:       typeExprFromString("[]pages_main_594861c5.StructItem"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StructItems",
													ReferenceLocation: ast_domain.Location{
														Line:   89,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   154,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   89,
												Column: 77,
											},
											NameLocation: ast_domain.Location{
												Line:   89,
												Column: 69,
											},
											RawExpression: "item.Title",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "item",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "item",
															ReferenceLocation: ast_domain.Location{
																Line:   89,
																Column: 77,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("item"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Title",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 6,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   89,
																Column: 77,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   137,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("item"),
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
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   89,
															Column: 77,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   137,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("item"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   89,
														Column: 77,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   137,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("item"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       1,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   89,
												Column: 54,
											},
											NameLocation: ast_domain.Location{
												Line:   89,
												Column: 47,
											},
											RawExpression: "item.GetKey()",
											Expression: &ast_domain.CallExpression{
												Callee: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "item",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "item",
																ReferenceLocation: ast_domain.Location{
																	Line:   89,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("item"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "GetKey",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 6,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("function"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "GetKey",
																ReferenceLocation: ast_domain.Location{
																	Line:   89,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   141,
																	Column: 21,
																},
															},
															BaseCodeGenVarName:  new("item"),
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
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "GetKey",
															ReferenceLocation: ast_domain.Location{
																Line:   89,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   141,
																Column: 21,
															},
														},
														BaseCodeGenVarName:  new("item"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Args: []ast_domain.Expression{},
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													BaseCodeGenVarName: new("item"),
													Stringability:      1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												BaseCodeGenVarName: new("item"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   89,
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
													Literal: "r.0:4:2:0.",
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
													Expression: &ast_domain.CallExpression{
														Callee: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "item",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("pages_main_594861c5.StructItem"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "item",
																		ReferenceLocation: ast_domain.Location{
																			Line:   89,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("item"),
																	OriginalSourcePath: new("pages/main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "GetKey",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 6,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("function"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "GetKey",
																		ReferenceLocation: ast_domain.Location{
																			Line:   89,
																			Column: 54,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   141,
																			Column: 21,
																		},
																	},
																	BaseCodeGenVarName:  new("item"),
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
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "GetKey",
																	ReferenceLocation: ast_domain.Location{
																		Line:   89,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   141,
																		Column: 21,
																	},
																},
																BaseCodeGenVarName:  new("item"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															},
														},
														Args: []ast_domain.Expression{},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															BaseCodeGenVarName: new("item"),
															Stringability:      1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   89,
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
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   93,
							Column: 5,
						},
						TagName: "section",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   93,
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
									Line:   94,
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
										Line:   94,
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
											Line:   94,
											Column: 11,
										},
										TextContent: "Map Keys",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:5:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   94,
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
									Line:   96,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:1",
									RelativeLocation: ast_domain.Location{
										Line:   96,
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
											Line:   97,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   97,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   97,
												Column: 13,
											},
											RawExpression: "(key, val) in state.StringMap",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: &ast_domain.Identifier{
													Name: "key",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 2,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "key",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 2,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("key"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												ItemVariable: &ast_domain.Identifier{
													Name: "val",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "val",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("val"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   97,
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
														Name: "StringMap",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 21,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("map[string]string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "StringMap",
																ReferenceLocation: ast_domain.Location{
																	Line:   97,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   157,
																	Column: 2,
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
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("map[string]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StringMap",
															ReferenceLocation: ast_domain.Location{
																Line:   97,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   157,
																Column: 2,
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
														TypeExpression:       typeExprFromString("map[string]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringMap",
														ReferenceLocation: ast_domain.Location{
															Line:   97,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   157,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("map[string]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringMap",
														ReferenceLocation: ast_domain.Location{
															Line:   97,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   157,
															Column: 2,
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
													TypeExpression:       typeExprFromString("map[string]string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "StringMap",
													ReferenceLocation: ast_domain.Location{
														Line:   97,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   157,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       5,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   97,
												Column: 58,
											},
											NameLocation: ast_domain.Location{
												Line:   97,
												Column: 51,
											},
											RawExpression: "key",
											Expression: &ast_domain.Identifier{
												Name: "key",
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
														Name: "key",
														ReferenceLocation: ast_domain.Location{
															Line:   98,
															Column: 39,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("key"),
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
													Name: "key",
													ReferenceLocation: ast_domain.Location{
														Line:   97,
														Column: 58,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("key"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   97,
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
													Literal: "r.0:5:1:0.",
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
														Name: "key",
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
																Name: "key",
																ReferenceLocation: ast_domain.Location{
																	Line:   98,
																	Column: 39,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("key"),
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   97,
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
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   98,
													Column: 11,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   98,
														Column: 25,
													},
													NameLocation: ast_domain.Location{
														Line:   98,
														Column: 17,
													},
													RawExpression: "key",
													Expression: &ast_domain.Identifier{
														Name: "key",
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
																Name: "key",
																ReferenceLocation: ast_domain.Location{
																	Line:   98,
																	Column: 25,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("key"),
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
															Name: "key",
															ReferenceLocation: ast_domain.Location{
																Line:   98,
																Column: 25,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("key"),
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   98,
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
															Literal: "r.0:5:1:0.",
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
																Name: "key",
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
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   98,
																			Column: 39,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   98,
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
															Literal: ":0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   98,
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
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   98,
													Column: 37,
												},
												TextContent: ": ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   98,
																Column: 37,
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
															Literal: "r.0:5:1:0.",
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
																Name: "key",
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
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   98,
																			Column: 39,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   98,
																Column: 37,
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
														Line:   98,
														Column: 37,
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
													Line:   98,
													Column: 39,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   98,
														Column: 53,
													},
													NameLocation: ast_domain.Location{
														Line:   98,
														Column: 45,
													},
													RawExpression: "val",
													Expression: &ast_domain.Identifier{
														Name: "val",
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
																Name: "val",
																ReferenceLocation: ast_domain.Location{
																	Line:   98,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("val"),
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
															Name: "val",
															ReferenceLocation: ast_domain.Location{
																Line:   98,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("val"),
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   98,
																Column: 39,
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
															Literal: "r.0:5:1:0.",
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
																Name: "key",
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
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   98,
																			Column: 39,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   98,
																Column: 39,
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
															Literal: ":2",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   98,
														Column: 39,
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
									Line:   102,
									Column: 7,
								},
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:2",
									RelativeLocation: ast_domain.Location{
										Line:   102,
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
											Line:   103,
											Column: 9,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   103,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   103,
												Column: 13,
											},
											RawExpression: "(key, val) in state.IntMap",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: &ast_domain.Identifier{
													Name: "key",
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
															Name: "key",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 2,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("key"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												ItemVariable: &ast_domain.Identifier{
													Name: "val",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "val",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("val"),
														OriginalSourcePath: new("pages/main.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 15,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   103,
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
														Name: "IntMap",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 21,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("map[int]string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "IntMap",
																ReferenceLocation: ast_domain.Location{
																	Line:   103,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   158,
																	Column: 2,
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
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("map[int]string"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "IntMap",
															ReferenceLocation: ast_domain.Location{
																Line:   103,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   158,
																Column: 2,
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
														TypeExpression:       typeExprFromString("map[int]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IntMap",
														ReferenceLocation: ast_domain.Location{
															Line:   103,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   158,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													Stringability:       5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("map[int]string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "IntMap",
														ReferenceLocation: ast_domain.Location{
															Line:   103,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   158,
															Column: 2,
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
													TypeExpression:       typeExprFromString("map[int]string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_071_p_key_fnv_hashing/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "IntMap",
													ReferenceLocation: ast_domain.Location{
														Line:   103,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   158,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												Stringability:       5,
											},
										},
										DirKey: &ast_domain.Directive{
											Type: ast_domain.DirectiveKey,
											Location: ast_domain.Location{
												Line:   103,
												Column: 55,
											},
											NameLocation: ast_domain.Location{
												Line:   103,
												Column: 48,
											},
											RawExpression: "key",
											Expression: &ast_domain.Identifier{
												Name: "key",
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
														Name: "key",
														ReferenceLocation: ast_domain.Location{
															Line:   104,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("key"),
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
													Name: "key",
													ReferenceLocation: ast_domain.Location{
														Line:   103,
														Column: 55,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("key"),
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   103,
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
													Literal: "r.0:5:2:0.",
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
														Name: "key",
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
																Name: "key",
																ReferenceLocation: ast_domain.Location{
																	Line:   104,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("key"),
															OriginalSourcePath: new("pages/main.pk"),
															Stringability:      1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   103,
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
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   104,
													Column: 11,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   104,
														Column: 25,
													},
													NameLocation: ast_domain.Location{
														Line:   104,
														Column: 17,
													},
													RawExpression: "strconv.Itoa(key)",
													Expression: &ast_domain.CallExpression{
														Callee: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "strconv",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       nil,
																		PackageAlias:         "strconv",
																		CanonicalPackagePath: "strconv",
																	},
																	BaseCodeGenVarName: new("strconv"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Itoa",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 9,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("function"),
																		PackageAlias:         "strconv",
																		CanonicalPackagePath: "strconv",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Itoa",
																		ReferenceLocation: ast_domain.Location{
																			Line:   104,
																			Column: 25,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("strconv"),
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
																	TypeExpression:       typeExprFromString("function"),
																	PackageAlias:         "strconv",
																	CanonicalPackagePath: "strconv",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Itoa",
																	ReferenceLocation: ast_domain.Location{
																		Line:   104,
																		Column: 25,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("strconv"),
															},
														},
														Args: []ast_domain.Expression{
															&ast_domain.Identifier{
																Name: "key",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 14,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   104,
																			Column: 25,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "strconv",
																CanonicalPackagePath: "strconv",
															},
															BaseCodeGenVarName: new("strconv"),
															Stringability:      1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "strconv",
															CanonicalPackagePath: "strconv",
														},
														BaseCodeGenVarName: new("strconv"),
														Stringability:      1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   104,
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
															Literal: "r.0:5:2:0.",
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
																Name: "key",
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
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   104,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   104,
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
															Literal: ":0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   104,
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
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   104,
													Column: 51,
												},
												TextContent: ": ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   104,
																Column: 51,
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
															Literal: "r.0:5:2:0.",
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
																Name: "key",
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
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   104,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   104,
																Column: 51,
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
														Line:   104,
														Column: 51,
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
													Line:   104,
													Column: 53,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   104,
														Column: 67,
													},
													NameLocation: ast_domain.Location{
														Line:   104,
														Column: 59,
													},
													RawExpression: "val",
													Expression: &ast_domain.Identifier{
														Name: "val",
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
																Name: "val",
																ReferenceLocation: ast_domain.Location{
																	Line:   104,
																	Column: 67,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("val"),
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
															Name: "val",
															ReferenceLocation: ast_domain.Location{
																Line:   104,
																Column: 67,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("val"),
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   104,
																Column: 53,
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
															Literal: "r.0:5:2:0.",
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
																Name: "key",
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
																		Name: "key",
																		ReferenceLocation: ast_domain.Location{
																			Line:   104,
																			Column: 53,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("key"),
																	OriginalSourcePath: new("pages/main.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   104,
																Column: 53,
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
															Literal: ":2",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   104,
														Column: 53,
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
			},
		},
	}
}()
