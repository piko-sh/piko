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
								TextContent: "Nested Pointer Chain Nil-Guard Test",
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
								TextContent: " This test verifies that nested p-if guards properly protect nested pointer dereferences in generated code. Each level should generate its own if-block without redundant safety checks. ",
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
							Line:   30,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   30,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   30,
								Column: 10,
							},
							RawExpression: "state.User != nil",
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
												TypeExpression:       typeExprFromString("pages_main_594861c5.PageData"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
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
										Name: "User",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "User",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   65,
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
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "User",
											ReferenceLocation: ast_domain.Location{
												Line:   30,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   65,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									},
								},
								Operator: "!=",
								Right: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 15,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
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
								Line:   30,
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
									Line:   31,
									Column: 7,
								},
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
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
										Name:  "id",
										Value: "user-name",
										Location: ast_domain.Location{
											Line:   31,
											Column: 17,
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
											Column: 28,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   31,
												Column: 28,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   31,
													Column: 31,
												},
												RawExpression: "state.User.Name",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
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
																	CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
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
															Name: "User",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   31,
																		Column: 31,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   65,
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
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   65,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Name",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   31,
																	Column: 31,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   57,
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
															CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   31,
																Column: 31,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   57,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   31,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   57,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
									Line:   32,
									Column: 7,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   32,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   32,
										Column: 12,
									},
									RawExpression: "state.User.Profile != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
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
															CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
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
													Name: "User",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "User",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 18,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   65,
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
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "User",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   65,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Profile",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Profile",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   58,
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
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Profile",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   58,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 23,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
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
									Value: "r.0:2:1",
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
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   33,
											Column: 9,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:2:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   33,
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
												Name:  "id",
												Value: "profile-bio",
												Location: ast_domain.Location{
													Line:   33,
													Column: 19,
												},
												NameLocation: ast_domain.Location{
													Line:   33,
													Column: 15,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   33,
													Column: 32,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   33,
														Column: 32,
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
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   33,
															Column: 35,
														},
														RawExpression: "state.User.Profile.Bio",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
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
																				CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "state",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 35,
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
																		Name: "User",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 7,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "User",
																				ReferenceLocation: ast_domain.Location{
																					Line:   33,
																					Column: 35,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   65,
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
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "User",
																			ReferenceLocation: ast_domain.Location{
																				Line:   33,
																				Column: 35,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   65,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Profile",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 12,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Profile",
																			ReferenceLocation: ast_domain.Location{
																				Line:   33,
																				Column: 35,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   58,
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
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Profile",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 35,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   58,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Bio",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 20,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Bio",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 35,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   53,
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
																	CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Bio",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 35,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   53,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Bio",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 35,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   53,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
											Line:   34,
											Column: 9,
										},
										TagName: "div",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStructurallyStatic: true,
										},
										DirIf: &ast_domain.Directive{
											Type: ast_domain.DirectiveIf,
											Location: ast_domain.Location{
												Line:   34,
												Column: 20,
											},
											NameLocation: ast_domain.Location{
												Line:   34,
												Column: 14,
											},
											RawExpression: "state.User.Profile.Avatar != nil",
											Expression: &ast_domain.BinaryExpression{
												Left: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
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
																		CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   34,
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
																Name: "User",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "User",
																		ReferenceLocation: ast_domain.Location{
																			Line:   34,
																			Column: 20,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   65,
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
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   34,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   65,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("pages/main.pk"),
																GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Profile",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 12,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Profile",
																	ReferenceLocation: ast_domain.Location{
																		Line:   34,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   58,
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
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Profile",
																ReferenceLocation: ast_domain.Location{
																	Line:   34,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   58,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Avatar",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*pages_main_594861c5.Avatar"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Avatar",
																ReferenceLocation: ast_domain.Location{
																	Line:   34,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   54,
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
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*pages_main_594861c5.Avatar"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Avatar",
															ReferenceLocation: ast_domain.Location{
																Line:   34,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   54,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
													},
												},
												Operator: "!=",
												Right: &ast_domain.NilLiteral{
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 30,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("nil"),
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
											Value: "r.0:2:1:1",
											RelativeLocation: ast_domain.Location{
												Line:   34,
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
													Line:   35,
													Column: 11,
												},
												TagName: "img",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_main_594861c5"),
													OriginalSourcePath:   new("pages/main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:2:1:1:0",
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
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "id",
														Value: "avatar-url",
														Location: ast_domain.Location{
															Line:   35,
															Column: 20,
														},
														NameLocation: ast_domain.Location{
															Line:   35,
															Column: 16,
														},
													},
												},
												DynamicAttributes: []ast_domain.DynamicAttribute{
													ast_domain.DynamicAttribute{
														Name:          "src",
														RawExpression: "state.User.Profile.Avatar.URL",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
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
																					CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "state",
																					ReferenceLocation: ast_domain.Location{
																						Line:   35,
																						Column: 38,
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
																			Name: "User",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 7,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																					PackageAlias:         "pages_main_594861c5",
																					CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "User",
																					ReferenceLocation: ast_domain.Location{
																						Line:   35,
																						Column: 38,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   65,
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
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("*pages_main_594861c5.User"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "User",
																				ReferenceLocation: ast_domain.Location{
																					Line:   35,
																					Column: 38,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   65,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("pageData"),
																			OriginalSourcePath:  new("pages/main.pk"),
																			GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Profile",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 12,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
																				PackageAlias:         "pages_main_594861c5",
																				CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Profile",
																				ReferenceLocation: ast_domain.Location{
																					Line:   35,
																					Column: 38,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   58,
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
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*pages_main_594861c5.Profile"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Profile",
																			ReferenceLocation: ast_domain.Location{
																				Line:   35,
																				Column: 38,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   58,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("pageData"),
																		OriginalSourcePath:  new("pages/main.pk"),
																		GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Avatar",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 20,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("*pages_main_594861c5.Avatar"),
																			PackageAlias:         "pages_main_594861c5",
																			CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Avatar",
																			ReferenceLocation: ast_domain.Location{
																				Line:   35,
																				Column: 38,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   54,
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
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("*pages_main_594861c5.Avatar"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Avatar",
																		ReferenceLocation: ast_domain.Location{
																			Line:   35,
																			Column: 38,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   54,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("pages/main.pk"),
																	GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "URL",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 27,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "pages_main_594861c5",
																		CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "URL",
																		ReferenceLocation: ast_domain.Location{
																			Line:   35,
																			Column: 38,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   49,
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
																	CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "URL",
																	ReferenceLocation: ast_domain.Location{
																		Line:   35,
																		Column: 38,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   49,
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
															Line:   35,
															Column: 38,
														},
														NameLocation: ast_domain.Location{
															Line:   35,
															Column: 32,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "pages_main_594861c5",
																CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "URL",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 38,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   49,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
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
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   40,
							Column: 5,
						},
						TagName: "img",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   40,
								Column: 16,
							},
							NameLocation: ast_domain.Location{
								Line:   40,
								Column: 10,
							},
							RawExpression: "state.Image != nil",
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
												TypeExpression:       typeExprFromString("pages_main_594861c5.PageData"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
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
										Name: "Image",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*pages_main_594861c5.Image"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Image",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 16,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   66,
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
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("*pages_main_594861c5.Image"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Image",
											ReferenceLocation: ast_domain.Location{
												Line:   40,
												Column: 16,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   66,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									},
								},
								Operator: "!=",
								Right: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 16,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
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
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   40,
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
								Value: "image-guarded",
								Location: ast_domain.Location{
									Line:   40,
									Column: 40,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 36,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "src",
								RawExpression: "state.Image.URL",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 61,
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
											Name: "Image",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.Image"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Image",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 61,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   66,
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
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*pages_main_594861c5.Image"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Image",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 61,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   66,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "URL",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 13,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "URL",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 61,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   61,
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
											CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "URL",
											ReferenceLocation: ast_domain.Location{
												Line:   40,
												Column: 61,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   61,
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
									Line:   40,
									Column: 61,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 55,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "URL",
										ReferenceLocation: ast_domain.Location{
											Line:   40,
											Column: 61,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   61,
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
								Name:          "alt",
								RawExpression: "state.Image.Alt",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
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
													CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 84,
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
											Name: "Image",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*pages_main_594861c5.Image"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Image",
													ReferenceLocation: ast_domain.Location{
														Line:   40,
														Column: 84,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   66,
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
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("*pages_main_594861c5.Image"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Image",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 84,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   66,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Alt",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 13,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "pages_main_594861c5",
												CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Alt",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 84,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   62,
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
											CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Alt",
											ReferenceLocation: ast_domain.Location{
												Line:   40,
												Column: 84,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   62,
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
									Line:   40,
									Column: 84,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 78,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "pages_main_594861c5",
										CanonicalPackagePath: "testcase_117_nil_guard_nested_pointer_chain/dist/pages/pages_main_594861c5",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Alt",
										ReferenceLocation: ast_domain.Location{
											Line:   40,
											Column: 84,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   62,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("pages/main.pk"),
									GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
									Stringability:       1,
								},
							},
						},
					},
				},
			},
		},
	}
}()
