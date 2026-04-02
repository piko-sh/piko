package collection_static_test

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
								TextContent: "Blog Posts",
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
						TagName: "article",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   24,
								Column: 21,
							},
							NameLocation: ast_domain.Location{
								Line:   24,
								Column: 14,
							},
							RawExpression: "(_, post) in r.GetCollection[Post](\"blog\")",
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
									Name: "post",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("pages_main_594861c5.Post"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "post",
											ReferenceLocation: ast_domain.Location{
												Line:   1,
												Column: 5,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("post"),
										OriginalSourcePath: new("pages/main.pk"),
									},
								},
								Collection: &ast_domain.CallExpression{
									Callee: &ast_domain.IndexExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "r",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 14,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]Post"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
											},
											Property: &ast_domain.Identifier{
												Name: "GetCollection",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 16,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]Post"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													IsStatic:             true,
													IsStructurallyStatic: true,
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
													TypeExpression:       typeExprFromString("[]Post"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
										Index: &ast_domain.Identifier{
											Name: "Post",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 30,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]Post"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
										Optional: false,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 14,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]Post"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.StringLiteral{
											Value: "blog",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 36,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]Post"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												IsStatic:             true,
												IsStructurallyStatic: true,
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 14,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]Post"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										IsStatic:             true,
										IsStructurallyStatic: true,
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]Post"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]Post"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]Post"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								IsStatic:             true,
								IsStructurallyStatic: true,
							},
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
									Literal: "r.0:1.",
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
										Name: "_",
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
											BaseCodeGenVarName: new("_"),
											Stringability:      1,
										},
									},
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 7,
								},
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   25,
										Column: 19,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 11,
									},
									RawExpression: "post.Title",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "post",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Post"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "post",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("post"),
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
													CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   36,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("post"),
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
												CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Title",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   36,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("post"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Title",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 19,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   36,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("post"),
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
												Line:   25,
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
											Literal: "r.0:1.",
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
															Line:   25,
															Column: 7,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("__pikoLoopIdx"),
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
												},
											},
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   25,
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
											Literal: ":0",
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   25,
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
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   26,
									Column: 7,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   26,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   26,
										Column: 10,
									},
									RawExpression: "post.Slug",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "post",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_main_594861c5.Post"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "post",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("post"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Slug",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 6,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_main_594861c5",
													CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Slug",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   37,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("post"),
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
												CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Slug",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   37,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("post"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_main_594861c5",
											CanonicalPackagePath: "testcase_059_collection_static_getCollection/dist/pages/pages_main_594861c5",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Slug",
											ReferenceLocation: ast_domain.Location{
												Line:   26,
												Column: 18,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   37,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("post"),
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
											Literal: "r.0:1.",
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
															Line:   26,
															Column: 7,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("__pikoLoopIdx"),
													OriginalSourcePath: new("pages/main.pk"),
													Stringability:      1,
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
											Literal: ":1",
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
	}
}()
