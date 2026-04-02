package test

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
					Column: 5,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   46,
						Column: 5,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("main.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "container",
						Location: ast_domain.Location{
							Line:   46,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   46,
							Column: 10,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   47,
							Column: 9,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   47,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   47,
									Column: 13,
								},
								TextContent: "External Package Variable Test",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   47,
										Column: 13,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   49,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   49,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "parish-name",
								Location: ast_domain.Location{
									Line:   49,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   49,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   49,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   49,
											Column: 35,
										},
										RawExpression: "domain.ParishMap[props.Parish]",
										Expression: &ast_domain.IndexExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "ParishMap",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("map[string]string"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ParishMap",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   24,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ParishMap",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   24,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Index: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 18,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Parish",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 24,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Parish",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   32,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 18,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Parish",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   32,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
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
												BaseCodeGenVarName: new("domain"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("domain"),
											OriginalSourcePath: new("main.pk"),
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
							Line:   50,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   50,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "status-code",
								Location: ast_domain.Location{
									Line:   50,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   50,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   50,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   50,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   50,
											Column: 35,
										},
										RawExpression: "domain.StatusCodes[props.Index]",
										Expression: &ast_domain.IndexExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "StatusCodes",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]int"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StatusCodes",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   40,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("[]int"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StatusCodes",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   40,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Index: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "props",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 20,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Props"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "props",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("props"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Index",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 26,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Index",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   33,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       1,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 20,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Index",
														ReferenceLocation: ast_domain.Location{
															Line:   50,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   33,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       1,
												},
											},
											Optional: false,
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
												BaseCodeGenVarName: new("domain"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("domain"),
											OriginalSourcePath: new("main.pk"),
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
							Line:   52,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   52,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "max-retries",
								Location: ast_domain.Location{
									Line:   52,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   52,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   52,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   52,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   52,
											Column: 35,
										},
										RawExpression: "domain.MaxRetries",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "domain",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "MaxRetries",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MaxRetries",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   43,
															Column: 7,
														},
													},
													BaseCodeGenVarName:   new("domain"),
													OriginalSourcePath:   new("main.pk"),
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
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MaxRetries",
													ReferenceLocation: ast_domain.Location{
														Line:   52,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   43,
														Column: 7,
													},
												},
												BaseCodeGenVarName:   new("domain"),
												OriginalSourcePath:   new("main.pk"),
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MaxRetries",
												ReferenceLocation: ast_domain.Location{
													Line:   52,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   43,
													Column: 7,
												},
											},
											BaseCodeGenVarName:   new("domain"),
											OriginalSourcePath:   new("main.pk"),
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
							Line:   53,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   53,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "app-version",
								Location: ast_domain.Location{
									Line:   53,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   53,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   53,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   53,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   53,
											Column: 35,
										},
										RawExpression: "domain.AppVersion",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "domain",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "AppVersion",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "AppVersion",
														ReferenceLocation: ast_domain.Location{
															Line:   53,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 7,
														},
													},
													BaseCodeGenVarName:   new("domain"),
													OriginalSourcePath:   new("main.pk"),
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
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "AppVersion",
													ReferenceLocation: ast_domain.Location{
														Line:   53,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 7,
													},
												},
												BaseCodeGenVarName:   new("domain"),
												OriginalSourcePath:   new("main.pk"),
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "AppVersion",
												ReferenceLocation: ast_domain.Location{
													Line:   53,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   46,
													Column: 7,
												},
											},
											BaseCodeGenVarName:   new("domain"),
											OriginalSourcePath:   new("main.pk"),
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
							Line:   54,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   54,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "app-name",
								Location: ast_domain.Location{
									Line:   54,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   54,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   54,
									Column: 29,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   54,
										Column: 29,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   54,
											Column: 32,
										},
										RawExpression: "domain.AppName",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "domain",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "AppName",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "AppName",
														ReferenceLocation: ast_domain.Location{
															Line:   54,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
													OriginalSourcePath: new("main.pk"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "AppName",
													ReferenceLocation: ast_domain.Location{
														Line:   54,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   49,
														Column: 5,
													},
												},
												BaseCodeGenVarName: new("domain"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "AppName",
												ReferenceLocation: ast_domain.Location{
													Line:   54,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   49,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("domain"),
											OriginalSourcePath: new("main.pk"),
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   56,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
							RelativeLocation: ast_domain.Location{
								Line:   56,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "timeout",
								Location: ast_domain.Location{
									Line:   56,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   56,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   56,
									Column: 28,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
									RelativeLocation: ast_domain.Location{
										Line:   56,
										Column: 28,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   56,
											Column: 31,
										},
										RawExpression: "domain.DefaultTimeout",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "domain",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "DefaultTimeout",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("time.Duration"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DefaultTimeout",
														ReferenceLocation: ast_domain.Location{
															Line:   56,
															Column: 31,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
													OriginalSourcePath: new("main.pk"),
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
													TypeExpression:       typeExprFromString("time.Duration"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DefaultTimeout",
													ReferenceLocation: ast_domain.Location{
														Line:   56,
														Column: 31,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   52,
														Column: 5,
													},
												},
												BaseCodeGenVarName: new("domain"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("time.Duration"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DefaultTimeout",
												ReferenceLocation: ast_domain.Location{
													Line:   56,
													Column: 31,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   52,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("domain"),
											OriginalSourcePath: new("main.pk"),
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   58,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:7",
							RelativeLocation: ast_domain.Location{
								Line:   58,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "config-name",
								Location: ast_domain.Location{
									Line:   58,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   58,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   58,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
									RelativeLocation: ast_domain.Location{
										Line:   58,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   58,
											Column: 35,
										},
										RawExpression: "domain.DefaultConfig.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "DefaultConfig",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Config"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DefaultConfig",
															ReferenceLocation: ast_domain.Location{
																Line:   58,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("domain.Config"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DefaultConfig",
														ReferenceLocation: ast_domain.Location{
															Line:   58,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   62,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 22,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   58,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   76,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   58,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   76,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   58,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   76,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   59,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:8",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "config-enabled",
								Location: ast_domain.Location{
									Line:   59,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   59,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   59,
									Column: 35,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:8:0",
									RelativeLocation: ast_domain.Location{
										Line:   59,
										Column: 35,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   59,
											Column: 38,
										},
										RawExpression: "domain.DefaultConfig.Enabled",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "DefaultConfig",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Config"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DefaultConfig",
															ReferenceLocation: ast_domain.Location{
																Line:   59,
																Column: 38,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("domain.Config"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DefaultConfig",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   62,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Enabled",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 22,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("bool"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Enabled",
														ReferenceLocation: ast_domain.Location{
															Line:   59,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   78,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Enabled",
													ReferenceLocation: ast_domain.Location{
														Line:   59,
														Column: 38,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   78,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Enabled",
												ReferenceLocation: ast_domain.Location{
													Line:   59,
													Column: 38,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   78,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   61,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:9",
							RelativeLocation: ast_domain.Location{
								Line:   61,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "result-value",
								Location: ast_domain.Location{
									Line:   61,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   61,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   61,
									Column: 33,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:9:0",
									RelativeLocation: ast_domain.Location{
										Line:   61,
										Column: 33,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   61,
											Column: 36,
										},
										RawExpression: "domain.StringResult.Value",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "StringResult",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Result[string]"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "StringResult",
															ReferenceLocation: ast_domain.Location{
																Line:   61,
																Column: 36,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   75,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("domain.Result[string]"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "StringResult",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   75,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Value",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 21,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Value",
														ReferenceLocation: ast_domain.Location{
															Line:   61,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   20,
															Column: 0,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Value",
													ReferenceLocation: ast_domain.Location{
														Line:   61,
														Column: 36,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   20,
														Column: 0,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Value",
												ReferenceLocation: ast_domain.Location{
													Line:   61,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   20,
													Column: 0,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   63,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   63,
								Column: 40,
							},
							NameLocation: ast_domain.Location{
								Line:   63,
								Column: 34,
							},
							RawExpression: "len(os.Args) > 0",
							Expression: &ast_domain.BinaryExpression{
								Left: &ast_domain.CallExpression{
									Callee: &ast_domain.Identifier{
										Name: "len",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("builtin_function"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "len",
												ReferenceLocation: ast_domain.Location{
													Line:   63,
													Column: 40,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("len"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "os",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 5,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "os",
														CanonicalPackagePath: "os",
													},
													BaseCodeGenVarName: new("os"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Args",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "os",
														CanonicalPackagePath: "os",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Args",
														ReferenceLocation: ast_domain.Location{
															Line:   63,
															Column: 40,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   16,
															Column: 1,
														},
													},
													BaseCodeGenVarName: new("os"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 5,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]string"),
													PackageAlias:         "os",
													CanonicalPackagePath: "os",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Args",
													ReferenceLocation: ast_domain.Location{
														Line:   63,
														Column: 40,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   16,
														Column: 1,
													},
												},
												BaseCodeGenVarName: new("os"),
											},
										},
									},
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
										Stringability: 1,
									},
								},
								Operator: ">",
								Right: &ast_domain.IntegerLiteral{
									Value: 0,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 16,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("int64"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
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
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("bool"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:10",
							RelativeLocation: ast_domain.Location{
								Line:   63,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "os-args-first",
								Location: ast_domain.Location{
									Line:   63,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   63,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   63,
									Column: 58,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:10:0",
									RelativeLocation: ast_domain.Location{
										Line:   63,
										Column: 58,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   63,
											Column: 61,
										},
										RawExpression: "os.Args[0]",
										Expression: &ast_domain.IndexExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "os",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "os",
															CanonicalPackagePath: "os",
														},
														BaseCodeGenVarName: new("os"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Args",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 4,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]string"),
															PackageAlias:         "os",
															CanonicalPackagePath: "os",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Args",
															ReferenceLocation: ast_domain.Location{
																Line:   63,
																Column: 61,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   16,
																Column: 1,
															},
														},
														BaseCodeGenVarName: new("os"),
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
														TypeExpression:       typeExprFromString("[]string"),
														PackageAlias:         "os",
														CanonicalPackagePath: "os",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Args",
														ReferenceLocation: ast_domain.Location{
															Line:   63,
															Column: 61,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   16,
															Column: 1,
														},
													},
													BaseCodeGenVarName: new("os"),
												},
											},
											Index: &ast_domain.IntegerLiteral{
												Value: 0,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 9,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int64"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													OriginalSourcePath: new("main.pk"),
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
												BaseCodeGenVarName: new("os"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("os"),
											OriginalSourcePath: new("main.pk"),
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
							Line:   65,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:11",
							RelativeLocation: ast_domain.Location{
								Line:   65,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "deep-country-name",
								Location: ast_domain.Location{
									Line:   65,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   65,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   65,
									Column: 38,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:11:0",
									RelativeLocation: ast_domain.Location{
										Line:   65,
										Column: 38,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   65,
											Column: 41,
										},
										RawExpression: "domain.DeepConfig.Primary.Country.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "domain",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       nil,
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																BaseCodeGenVarName: new("domain"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "DeepConfig",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 8,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DeepConfig",
																	ReferenceLocation: ast_domain.Location{
																		Line:   65,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   159,
																		Column: 5,
																	},
																},
																BaseCodeGenVarName: new("domain"),
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
																TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DeepConfig",
																ReferenceLocation: ast_domain.Location{
																	Line:   65,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   159,
																	Column: 5,
																},
															},
															BaseCodeGenVarName: new("domain"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Primary",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 19,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("domain.Address"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Primary",
																ReferenceLocation: ast_domain.Location{
																	Line:   65,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   126,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
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
															TypeExpression:       typeExprFromString("domain.Address"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Primary",
															ReferenceLocation: ast_domain.Location{
																Line:   65,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   126,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Country",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 27,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Country"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Country",
															ReferenceLocation: ast_domain.Location{
																Line:   65,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   101,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
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
														TypeExpression:       typeExprFromString("domain.Country"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Country",
														ReferenceLocation: ast_domain.Location{
															Line:   65,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   101,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 35,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   65,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   106,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   65,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   106,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   65,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   106,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   66,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:12",
							RelativeLocation: ast_domain.Location{
								Line:   66,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "deep-region-capital",
								Location: ast_domain.Location{
									Line:   66,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   66,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   66,
									Column: 40,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:12:0",
									RelativeLocation: ast_domain.Location{
										Line:   66,
										Column: 40,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   66,
											Column: 43,
										},
										RawExpression: "domain.DeepConfig.Primary.Country.Regions[0].Capital.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.IndexExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "domain",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       nil,
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			BaseCodeGenVarName: new("domain"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "DeepConfig",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 8,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "DeepConfig",
																				ReferenceLocation: ast_domain.Location{
																					Line:   66,
																					Column: 43,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   159,
																					Column: 5,
																				},
																			},
																			BaseCodeGenVarName: new("domain"),
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
																			TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "DeepConfig",
																			ReferenceLocation: ast_domain.Location{
																				Line:   66,
																				Column: 43,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   159,
																				Column: 5,
																			},
																		},
																		BaseCodeGenVarName: new("domain"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Primary",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 19,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("domain.Address"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Primary",
																			ReferenceLocation: ast_domain.Location{
																				Line:   66,
																				Column: 43,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   126,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("domain"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("domain/maps.go"),
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
																		TypeExpression:       typeExprFromString("domain.Address"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Primary",
																		ReferenceLocation: ast_domain.Location{
																			Line:   66,
																			Column: 43,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   126,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("domain"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/maps.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Country",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 27,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("domain.Country"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Country",
																		ReferenceLocation: ast_domain.Location{
																			Line:   66,
																			Column: 43,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   101,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("domain"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/maps.go"),
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
																	TypeExpression:       typeExprFromString("domain.Country"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Country",
																	ReferenceLocation: ast_domain.Location{
																		Line:   66,
																		Column: 43,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   101,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Regions",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 35,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]domain.Region"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Regions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   66,
																		Column: 43,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   108,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
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
																TypeExpression:       typeExprFromString("[]domain.Region"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Regions",
																ReferenceLocation: ast_domain.Location{
																	Line:   66,
																	Column: 43,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   108,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
														},
													},
													Index: &ast_domain.IntegerLiteral{
														Value: 0,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 43,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("main.pk"),
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
															TypeExpression:       typeExprFromString("domain.Region"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Capital",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 46,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.City"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Capital",
															ReferenceLocation: ast_domain.Location{
																Line:   66,
																Column: 43,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   115,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
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
														TypeExpression:       typeExprFromString("domain.City"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Capital",
														ReferenceLocation: ast_domain.Location{
															Line:   66,
															Column: 43,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   115,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 54,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   66,
															Column: 43,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   120,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   66,
														Column: 43,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   120,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   66,
													Column: 43,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   120,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   67,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:13",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "deep-region-postcode",
								Location: ast_domain.Location{
									Line:   67,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   67,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   67,
									Column: 41,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:13:0",
									RelativeLocation: ast_domain.Location{
										Line:   67,
										Column: 41,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   67,
											Column: 44,
										},
										RawExpression: "domain.DeepConfig.Primary.Country.Regions[0].Capital.PostCode",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.IndexExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "domain",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       nil,
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			BaseCodeGenVarName: new("domain"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "DeepConfig",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 8,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "DeepConfig",
																				ReferenceLocation: ast_domain.Location{
																					Line:   67,
																					Column: 44,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   159,
																					Column: 5,
																				},
																			},
																			BaseCodeGenVarName: new("domain"),
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
																			TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "DeepConfig",
																			ReferenceLocation: ast_domain.Location{
																				Line:   67,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   159,
																				Column: 5,
																			},
																		},
																		BaseCodeGenVarName: new("domain"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Primary",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 19,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("domain.Address"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Primary",
																			ReferenceLocation: ast_domain.Location{
																				Line:   67,
																				Column: 44,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   126,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("domain"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("domain/maps.go"),
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
																		TypeExpression:       typeExprFromString("domain.Address"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Primary",
																		ReferenceLocation: ast_domain.Location{
																			Line:   67,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   126,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("domain"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/maps.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Country",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 27,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("domain.Country"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Country",
																		ReferenceLocation: ast_domain.Location{
																			Line:   67,
																			Column: 44,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   101,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("domain"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/maps.go"),
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
																	TypeExpression:       typeExprFromString("domain.Country"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Country",
																	ReferenceLocation: ast_domain.Location{
																		Line:   67,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   101,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Regions",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 35,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]domain.Region"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Regions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   67,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   108,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
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
																TypeExpression:       typeExprFromString("[]domain.Region"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Regions",
																ReferenceLocation: ast_domain.Location{
																	Line:   67,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   108,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
														},
													},
													Index: &ast_domain.IntegerLiteral{
														Value: 0,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 43,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("main.pk"),
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
															TypeExpression:       typeExprFromString("domain.Region"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Capital",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 46,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.City"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Capital",
															ReferenceLocation: ast_domain.Location{
																Line:   67,
																Column: 44,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   115,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
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
														TypeExpression:       typeExprFromString("domain.City"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Capital",
														ReferenceLocation: ast_domain.Location{
															Line:   67,
															Column: 44,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   115,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "PostCode",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 54,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "PostCode",
														ReferenceLocation: ast_domain.Location{
															Line:   67,
															Column: 44,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   121,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "PostCode",
													ReferenceLocation: ast_domain.Location{
														Line:   67,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   121,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "PostCode",
												ReferenceLocation: ast_domain.Location{
													Line:   67,
													Column: 44,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   121,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   69,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:14",
							RelativeLocation: ast_domain.Location{
								Line:   69,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "pointer-user-name",
								Location: ast_domain.Location{
									Line:   69,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   69,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   69,
									Column: 38,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:14:0",
									RelativeLocation: ast_domain.Location{
										Line:   69,
										Column: 38,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   69,
											Column: 41,
										},
										RawExpression: "domain.DefaultUser.Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "DefaultUser",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("*domain.User"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DefaultUser",
															ReferenceLocation: ast_domain.Location{
																Line:   69,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   112,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("*domain.User"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DefaultUser",
														ReferenceLocation: ast_domain.Location{
															Line:   69,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   112,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 20,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   69,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   140,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   69,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   140,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   69,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   140,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   71,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:15",
							RelativeLocation: ast_domain.Location{
								Line:   71,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "embedded-created",
								Location: ast_domain.Location{
									Line:   71,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   71,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   71,
									Column: 37,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:15:0",
									RelativeLocation: ast_domain.Location{
										Line:   71,
										Column: 37,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   71,
											Column: 40,
										},
										RawExpression: "domain.DefaultArticle.CreatedAt",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "DefaultArticle",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Article"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DefaultArticle",
															ReferenceLocation: ast_domain.Location{
																Line:   71,
																Column: 40,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   137,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("domain.Article"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DefaultArticle",
														ReferenceLocation: ast_domain.Location{
															Line:   71,
															Column: 40,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   137,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "CreatedAt",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CreatedAt",
														ReferenceLocation: ast_domain.Location{
															Line:   71,
															Column: 40,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   145,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CreatedAt",
													ReferenceLocation: ast_domain.Location{
														Line:   71,
														Column: 40,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   145,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CreatedAt",
												ReferenceLocation: ast_domain.Location{
													Line:   71,
													Column: 40,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   145,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   72,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:16",
							RelativeLocation: ast_domain.Location{
								Line:   72,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "embedded-title",
								Location: ast_domain.Location{
									Line:   72,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   72,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   72,
									Column: 35,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:16:0",
									RelativeLocation: ast_domain.Location{
										Line:   72,
										Column: 35,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   72,
											Column: 38,
										},
										RawExpression: "domain.DefaultArticle.Title",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "domain",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       nil,
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "DefaultArticle",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 8,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Article"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DefaultArticle",
															ReferenceLocation: ast_domain.Location{
																Line:   72,
																Column: 38,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   137,
																Column: 5,
															},
														},
														BaseCodeGenVarName: new("domain"),
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
														TypeExpression:       typeExprFromString("domain.Article"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DefaultArticle",
														ReferenceLocation: ast_domain.Location{
															Line:   72,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   137,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Title",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   72,
															Column: 38,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   152,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   72,
														Column: 38,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   152,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Title",
												ReferenceLocation: ast_domain.Location{
													Line:   72,
													Column: 38,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   152,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   74,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:17",
							RelativeLocation: ast_domain.Location{
								Line:   74,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "alias-id",
								Location: ast_domain.Location{
									Line:   74,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   74,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   74,
									Column: 29,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:17:0",
									RelativeLocation: ast_domain.Location{
										Line:   74,
										Column: 29,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   74,
											Column: 32,
										},
										RawExpression: "domain.CurrentUserID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "domain",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "CurrentUserID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("domain.ID"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "CurrentUserID",
														ReferenceLocation: ast_domain.Location{
															Line:   74,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   153,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
													OriginalSourcePath: new("main.pk"),
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
													TypeExpression:       typeExprFromString("domain.ID"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CurrentUserID",
													ReferenceLocation: ast_domain.Location{
														Line:   74,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   153,
														Column: 5,
													},
												},
												BaseCodeGenVarName: new("domain"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("domain.ID"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CurrentUserID",
												ReferenceLocation: ast_domain.Location{
													Line:   74,
													Column: 32,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   153,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("domain"),
											OriginalSourcePath: new("main.pk"),
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   75,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:18",
							RelativeLocation: ast_domain.Location{
								Line:   75,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "named-type-id",
								Location: ast_domain.Location{
									Line:   75,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   75,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   75,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:18:0",
									RelativeLocation: ast_domain.Location{
										Line:   75,
										Column: 34,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   75,
											Column: 37,
										},
										RawExpression: "domain.OwnerID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "domain",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       nil,
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "OwnerID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 8,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("domain.UserID"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OwnerID",
														ReferenceLocation: ast_domain.Location{
															Line:   75,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   156,
															Column: 5,
														},
													},
													BaseCodeGenVarName: new("domain"),
													OriginalSourcePath: new("main.pk"),
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
													TypeExpression:       typeExprFromString("domain.UserID"),
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "OwnerID",
													ReferenceLocation: ast_domain.Location{
														Line:   75,
														Column: 37,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   156,
														Column: 5,
													},
												},
												BaseCodeGenVarName: new("domain"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("domain.UserID"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "OwnerID",
												ReferenceLocation: ast_domain.Location{
													Line:   75,
													Column: 37,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   156,
													Column: 5,
												},
											},
											BaseCodeGenVarName: new("domain"),
											OriginalSourcePath: new("main.pk"),
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   77,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:19",
							RelativeLocation: ast_domain.Location{
								Line:   77,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "map-nested-country",
								Location: ast_domain.Location{
									Line:   77,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   77,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   77,
									Column: 39,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:19:0",
									RelativeLocation: ast_domain.Location{
										Line:   77,
										Column: 39,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   77,
											Column: 42,
										},
										RawExpression: "domain.DeepConfig.Lookup[\"home\"].Country.Code",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.IndexExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "domain",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       nil,
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	BaseCodeGenVarName: new("domain"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "DeepConfig",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 8,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "DeepConfig",
																		ReferenceLocation: ast_domain.Location{
																			Line:   77,
																			Column: 42,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   159,
																			Column: 5,
																		},
																	},
																	BaseCodeGenVarName: new("domain"),
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
																	TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DeepConfig",
																	ReferenceLocation: ast_domain.Location{
																		Line:   77,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   159,
																		Column: 5,
																	},
																},
																BaseCodeGenVarName: new("domain"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Lookup",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 19,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("map[string]domain.Address"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Lookup",
																	ReferenceLocation: ast_domain.Location{
																		Line:   77,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   128,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
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
																TypeExpression:       typeExprFromString("map[string]domain.Address"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Lookup",
																ReferenceLocation: ast_domain.Location{
																	Line:   77,
																	Column: 42,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   128,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
														},
													},
													Index: &ast_domain.StringLiteral{
														Value: "home",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 26,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("main.pk"),
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
															TypeExpression:       typeExprFromString("domain.Address"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Country",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 34,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.Country"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Country",
															ReferenceLocation: ast_domain.Location{
																Line:   77,
																Column: 42,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   101,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
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
														TypeExpression:       typeExprFromString("domain.Country"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Country",
														ReferenceLocation: ast_domain.Location{
															Line:   77,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   101,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Code",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 42,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Code",
														ReferenceLocation: ast_domain.Location{
															Line:   77,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   107,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Code",
													ReferenceLocation: ast_domain.Location{
														Line:   77,
														Column: 42,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   107,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Code",
												ReferenceLocation: ast_domain.Location{
													Line:   77,
													Column: 42,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   107,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   78,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:20",
							RelativeLocation: ast_domain.Location{
								Line:   78,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "map-nested-region",
								Location: ast_domain.Location{
									Line:   78,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   78,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   78,
									Column: 38,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:20:0",
									RelativeLocation: ast_domain.Location{
										Line:   78,
										Column: 38,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   78,
											Column: 41,
										},
										RawExpression: "domain.DeepConfig.Lookup[\"home\"].Country.Regions[0].Name",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.IndexExpression{
															Base: &ast_domain.MemberExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "domain",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       nil,
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			BaseCodeGenVarName: new("domain"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "DeepConfig",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 8,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "DeepConfig",
																				ReferenceLocation: ast_domain.Location{
																					Line:   78,
																					Column: 41,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   159,
																					Column: 5,
																				},
																			},
																			BaseCodeGenVarName: new("domain"),
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
																			TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "DeepConfig",
																			ReferenceLocation: ast_domain.Location{
																				Line:   78,
																				Column: 41,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   159,
																				Column: 5,
																			},
																		},
																		BaseCodeGenVarName: new("domain"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Lookup",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 19,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("map[string]domain.Address"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Lookup",
																			ReferenceLocation: ast_domain.Location{
																				Line:   78,
																				Column: 41,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   128,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("domain"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("domain/maps.go"),
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
																		TypeExpression:       typeExprFromString("map[string]domain.Address"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Lookup",
																		ReferenceLocation: ast_domain.Location{
																			Line:   78,
																			Column: 41,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   128,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("domain"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/maps.go"),
																},
															},
															Index: &ast_domain.StringLiteral{
																Value: "home",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 26,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	OriginalSourcePath: new("main.pk"),
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
																	TypeExpression:       typeExprFromString("domain.Address"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																BaseCodeGenVarName: new("domain"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Country",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 34,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("domain.Country"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Country",
																	ReferenceLocation: ast_domain.Location{
																		Line:   78,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   101,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
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
																TypeExpression:       typeExprFromString("domain.Country"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Country",
																ReferenceLocation: ast_domain.Location{
																	Line:   78,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   101,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Regions",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 42,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]domain.Region"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Regions",
																ReferenceLocation: ast_domain.Location{
																	Line:   78,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   108,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
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
															TypeExpression:       typeExprFromString("[]domain.Region"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Regions",
															ReferenceLocation: ast_domain.Location{
																Line:   78,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   108,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 0,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 50,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("main.pk"),
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
														TypeExpression:       typeExprFromString("domain.Region"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													BaseCodeGenVarName: new("domain"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Name",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 53,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   78,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   113,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   78,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   113,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   78,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   113,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
							Line:   79,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:21",
							RelativeLocation: ast_domain.Location{
								Line:   79,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "map-nested-capital",
								Location: ast_domain.Location{
									Line:   79,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   79,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   79,
									Column: 39,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:21:0",
									RelativeLocation: ast_domain.Location{
										Line:   79,
										Column: 39,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   79,
											Column: 42,
										},
										RawExpression: "domain.DeepConfig.Lookup[\"home\"].Country.Regions[0].Capital.PostCode",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.IndexExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.IndexExpression{
																Base: &ast_domain.MemberExpression{
																	Base: &ast_domain.MemberExpression{
																		Base: &ast_domain.Identifier{
																			Name: "domain",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 1,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       nil,
																					PackageAlias:         "domain",
																					CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																				},
																				BaseCodeGenVarName: new("domain"),
																			},
																		},
																		Property: &ast_domain.Identifier{
																			Name: "DeepConfig",
																			RelativeLocation: ast_domain.Location{
																				Line:   1,
																				Column: 8,
																			},
																			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																				ResolvedType: &ast_domain.ResolvedTypeInfo{
																					TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																					PackageAlias:         "domain",
																					CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																				},
																				Symbol: &ast_domain.ResolvedSymbol{
																					Name: "DeepConfig",
																					ReferenceLocation: ast_domain.Location{
																						Line:   79,
																						Column: 42,
																					},
																					DeclarationLocation: ast_domain.Location{
																						Line:   159,
																						Column: 5,
																					},
																				},
																				BaseCodeGenVarName: new("domain"),
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
																				TypeExpression:       typeExprFromString("domain.DeepConfigType"),
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "DeepConfig",
																				ReferenceLocation: ast_domain.Location{
																					Line:   79,
																					Column: 42,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   159,
																					Column: 5,
																				},
																			},
																			BaseCodeGenVarName: new("domain"),
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Lookup",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 19,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("map[string]domain.Address"),
																				PackageAlias:         "domain",
																				CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "Lookup",
																				ReferenceLocation: ast_domain.Location{
																					Line:   79,
																					Column: 42,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   128,
																					Column: 2,
																				},
																			},
																			BaseCodeGenVarName:  new("domain"),
																			OriginalSourcePath:  new("main.pk"),
																			GeneratedSourcePath: new("domain/maps.go"),
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
																			TypeExpression:       typeExprFromString("map[string]domain.Address"),
																			PackageAlias:         "domain",
																			CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Lookup",
																			ReferenceLocation: ast_domain.Location{
																				Line:   79,
																				Column: 42,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   128,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("domain"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("domain/maps.go"),
																	},
																},
																Index: &ast_domain.StringLiteral{
																	Value: "home",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 26,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("main.pk"),
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
																		TypeExpression:       typeExprFromString("domain.Address"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	BaseCodeGenVarName: new("domain"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Country",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 34,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("domain.Country"),
																		PackageAlias:         "domain",
																		CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Country",
																		ReferenceLocation: ast_domain.Location{
																			Line:   79,
																			Column: 42,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   101,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("domain"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("domain/maps.go"),
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
																	TypeExpression:       typeExprFromString("domain.Country"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Country",
																	ReferenceLocation: ast_domain.Location{
																		Line:   79,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   101,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Regions",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 42,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("[]domain.Region"),
																	PackageAlias:         "domain",
																	CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Regions",
																	ReferenceLocation: ast_domain.Location{
																		Line:   79,
																		Column: 42,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   108,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("domain"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("domain/maps.go"),
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
																TypeExpression:       typeExprFromString("[]domain.Region"),
																PackageAlias:         "domain",
																CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Regions",
																ReferenceLocation: ast_domain.Location{
																	Line:   79,
																	Column: 42,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   108,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("domain"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("domain/maps.go"),
														},
													},
													Index: &ast_domain.IntegerLiteral{
														Value: 0,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 50,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int64"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("main.pk"),
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
															TypeExpression:       typeExprFromString("domain.Region"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														BaseCodeGenVarName: new("domain"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Capital",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 53,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("domain.City"),
															PackageAlias:         "domain",
															CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Capital",
															ReferenceLocation: ast_domain.Location{
																Line:   79,
																Column: 42,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   115,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("domain"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("domain/maps.go"),
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
														TypeExpression:       typeExprFromString("domain.City"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Capital",
														ReferenceLocation: ast_domain.Location{
															Line:   79,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   115,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "PostCode",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 61,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "domain",
														CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "PostCode",
														ReferenceLocation: ast_domain.Location{
															Line:   79,
															Column: 42,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   121,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("domain"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("domain/maps.go"),
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
													PackageAlias:         "domain",
													CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "PostCode",
													ReferenceLocation: ast_domain.Location{
														Line:   79,
														Column: 42,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   121,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("domain"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("domain/maps.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "domain",
												CanonicalPackagePath: "testcase_69_external_package_map_resolution/domain",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "PostCode",
												ReferenceLocation: ast_domain.Location{
													Line:   79,
													Column: 42,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   121,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("domain"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("domain/maps.go"),
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
