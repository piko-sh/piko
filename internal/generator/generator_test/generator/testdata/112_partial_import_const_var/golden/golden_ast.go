package partial_import_const_var_test

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
					Line:   46,
					Column: 2,
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
						Line:   46,
						Column: 2,
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
						Value: "main",
						Location: ast_domain.Location{
							Line:   46,
							Column: 14,
						},
						NameLocation: ast_domain.Location{
							Line:   46,
							Column: 7,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   47,
							Column: 3,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   47,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   47,
									Column: 7,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   47,
											Column: 10,
										},
										RawExpression: "cfg.AppName",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "cfg",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													BaseCodeGenVarName: new("partials_config_76488b1c"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "AppName",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_config_76488b1c.any /* failed to parse type string: untyped_string */"),
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "AppName",
														ReferenceLocation: ast_domain.Location{
															Line:   47,
															Column: 10,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   7,
															Column: 7,
														},
													},
													BaseCodeGenVarName:   new("partials_config_76488b1c"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
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
													TypeExpression:       typeExprFromString("partials_config_76488b1c.any /* failed to parse type string: untyped_string */"),
													PackageAlias:         "partials_config_76488b1c",
													CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "AppName",
													ReferenceLocation: ast_domain.Location{
														Line:   47,
														Column: 10,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   7,
														Column: 7,
													},
												},
												BaseCodeGenVarName:   new("partials_config_76488b1c"),
												OriginalSourcePath:   new("pages/main.pk"),
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_config_76488b1c.any /* failed to parse type string: untyped_string */"),
												PackageAlias:         "partials_config_76488b1c",
												CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "AppName",
												ReferenceLocation: ast_domain.Location{
													Line:   47,
													Column: 10,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   7,
													Column: 7,
												},
											},
											BaseCodeGenVarName:   new("partials_config_76488b1c"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   48,
							Column: 3,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   48,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   48,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   48,
										Column: 6,
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
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   48,
											Column: 6,
										},
										Literal: "Version: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   48,
											Column: 18,
										},
										RawExpression: "cfg.Version",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "cfg",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													BaseCodeGenVarName: new("partials_config_76488b1c"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Version",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Version",
														ReferenceLocation: ast_domain.Location{
															Line:   48,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   10,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("partials_config_76488b1c"),
													OriginalSourcePath: new("pages/main.pk"),
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
													PackageAlias:         "partials_config_76488b1c",
													CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Version",
													ReferenceLocation: ast_domain.Location{
														Line:   48,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   10,
														Column: 5,
													},
												},
												BaseCodeGenVarName: new("partials_config_76488b1c"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_config_76488b1c",
												CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Version",
												ReferenceLocation: ast_domain.Location{
													Line:   48,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   10,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("partials_config_76488b1c"),
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   49,
							Column: 3,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   49,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   49,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   49,
										Column: 6,
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
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   49,
											Column: 6,
										},
										Literal: "Max Items: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   49,
											Column: 20,
										},
										RawExpression: "cfg.MaxItems",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "cfg",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													BaseCodeGenVarName: new("partials_config_76488b1c"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "MaxItems",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_config_76488b1c.any /* failed to parse type string: untyped_int */"),
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MaxItems",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   8,
															Column: 7,
														},
													},
													BaseCodeGenVarName:   new("partials_config_76488b1c"),
													OriginalSourcePath:   new("pages/main.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
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
													TypeExpression:       typeExprFromString("partials_config_76488b1c.any /* failed to parse type string: untyped_int */"),
													PackageAlias:         "partials_config_76488b1c",
													CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MaxItems",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   8,
														Column: 7,
													},
												},
												BaseCodeGenVarName:   new("partials_config_76488b1c"),
												OriginalSourcePath:   new("pages/main.pk"),
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_config_76488b1c.any /* failed to parse type string: untyped_int */"),
												PackageAlias:         "partials_config_76488b1c",
												CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MaxItems",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   8,
													Column: 7,
												},
											},
											BaseCodeGenVarName:   new("partials_config_76488b1c"),
											OriginalSourcePath:   new("pages/main.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   50,
							Column: 3,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   50,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   50,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   50,
										Column: 6,
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
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   50,
											Column: 6,
										},
										Literal: "Status: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   50,
											Column: 17,
										},
										RawExpression: "cfg.StatusLabels[\"pending\"]",
										Expression: &ast_domain.IndexExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "cfg",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "partials_config_76488b1c",
															CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
														},
														BaseCodeGenVarName: new("partials_config_76488b1c"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "StatusLabels",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 5,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("map[string]string"),
															PackageAlias:         "partials_config_76488b1c",
															CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StatusLabels",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 17,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   11,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("partials_config_76488b1c"),
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
														TypeExpression:       typeExprFromString("map[string]string"),
														PackageAlias:         "partials_config_76488b1c",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/partials/partials_config_76488b1c",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StatusLabels",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 17,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   11,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("partials_config_76488b1c"),
												},
											},
											Index: &ast_domain.StringLiteral{
												Value: "pending",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 18,
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
											Optional: false,
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
												BaseCodeGenVarName: new("partials_config_76488b1c"),
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
											BaseCodeGenVarName: new("partials_config_76488b1c"),
											OriginalSourcePath: new("pages/main.pk"),
											Stringability:      1,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   51,
							Column: 3,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   51,
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
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   51,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   51,
										Column: 6,
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
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   51,
											Column: 6,
										},
										Literal: "From Render: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   51,
											Column: 22,
										},
										RawExpression: "state.Status",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 22,
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
												Name: "Status",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Status",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 22,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   34,
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
													CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Status",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 22,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   34,
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
												CanonicalPackagePath: "testcase_112_partial_import_const_var/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Status",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 22,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   34,
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
	}
}()
