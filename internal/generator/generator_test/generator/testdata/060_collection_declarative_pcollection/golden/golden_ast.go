package collection_declarative_test

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
				TagName: "article",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
					OriginalSourcePath:   new("pages/blog/{slug}.pk"),
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
						OriginalSourcePath: new("pages/blog/{slug}.pk"),
						Stringability:      1,
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "blog-post",
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
						TagName: "header",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
							OriginalSourcePath:   new("pages/blog/{slug}.pk"),
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
								OriginalSourcePath: new("pages/blog/{slug}.pk"),
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
								TagName: "h1",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
									OriginalSourcePath:   new("pages/blog/{slug}.pk"),
								},
								DirText: &ast_domain.Directive{
									Type: ast_domain.DirectiveText,
									Location: ast_domain.Location{
										Line:   24,
										Column: 19,
									},
									NameLocation: ast_domain.Location{
										Line:   24,
										Column: 11,
									},
									RawExpression: "state.Title",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("pages_blog_slug_989a4cf3.Response"),
													PackageAlias:         "pages_blog_slug_989a4cf3",
													CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "state",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("pages/blog/{slug}.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Title",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_blog_slug_989a4cf3",
													CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Title",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 19,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   45,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/blog/{slug}.pk"),
												GeneratedSourcePath: new("dist/pages/pages_blog_slug_989a4cf3/generated.go"),
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
												PackageAlias:         "pages_blog_slug_989a4cf3",
												CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Title",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 19,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   45,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/blog/{slug}.pk"),
											GeneratedSourcePath: new("dist/pages/pages_blog_slug_989a4cf3/generated.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "pages_blog_slug_989a4cf3",
											CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Title",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 19,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   45,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/blog/{slug}.pk"),
										GeneratedSourcePath: new("dist/pages/pages_blog_slug_989a4cf3/generated.go"),
										Stringability:       1,
									},
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
										OriginalSourcePath: new("pages/blog/{slug}.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 32,
										},
										TextContent: "Blog Post Title",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
											OriginalSourcePath:   new("pages/blog/{slug}.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 32,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("pages/blog/{slug}.pk"),
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
									Column: 7,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
									OriginalSourcePath:   new("pages/blog/{slug}.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
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
										OriginalSourcePath: new("pages/blog/{slug}.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "meta",
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
											Column: 9,
										},
										TagName: "span",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
											OriginalSourcePath:   new("pages/blog/{slug}.pk"),
										},
										DirText: &ast_domain.Directive{
											Type: ast_domain.DirectiveText,
											Location: ast_domain.Location{
												Line:   26,
												Column: 23,
											},
											NameLocation: ast_domain.Location{
												Line:   26,
												Column: 15,
											},
											RawExpression: "state.Slug",
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_blog_slug_989a4cf3.Response"),
															PackageAlias:         "pages_blog_slug_989a4cf3",
															CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("pageData"),
														OriginalSourcePath: new("pages/blog/{slug}.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Slug",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "pages_blog_slug_989a4cf3",
															CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Slug",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   46,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/blog/{slug}.pk"),
														GeneratedSourcePath: new("dist/pages/pages_blog_slug_989a4cf3/generated.go"),
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
														PackageAlias:         "pages_blog_slug_989a4cf3",
														CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Slug",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   46,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/blog/{slug}.pk"),
													GeneratedSourcePath: new("dist/pages/pages_blog_slug_989a4cf3/generated.go"),
													Stringability:       1,
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "pages_blog_slug_989a4cf3",
													CanonicalPackagePath: "testcase_060_collection_declarative_pcollection/dist/pages/pages_blog_slug_989a4cf3",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Slug",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   46,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/blog/{slug}.pk"),
												GeneratedSourcePath: new("dist/pages/pages_blog_slug_989a4cf3/generated.go"),
												Stringability:       1,
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   26,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("pages/blog/{slug}.pk"),
												Stringability:      1,
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeText,
												Location: ast_domain.Location{
													Line:   26,
													Column: 35,
												},
												TextContent: "slug",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
													OriginalSourcePath:   new("pages/blog/{slug}.pk"),
													IsStatic:             true,
													IsStructurallyStatic: true,
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   26,
														Column: 35,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/blog/{slug}.pk"),
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
							Line:   29,
							Column: 5,
						},
						TagName: "main",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
							OriginalSourcePath:   new("pages/blog/{slug}.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("pages/blog/{slug}.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "content",
								Location: ast_domain.Location{
									Line:   29,
									Column: 18,
								},
								NameLocation: ast_domain.Location{
									Line:   29,
									Column: 11,
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
								TagName: "piko:content",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_blog_slug_989a4cf3"),
									OriginalSourcePath:   new("pages/blog/{slug}.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
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
										OriginalSourcePath: new("pages/blog/{slug}.pk"),
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
