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
					Line:   45,
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
						Line:   45,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   46,
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
								Line:   46,
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
									Line:   46,
									Column: 13,
								},
								TextContent: "Articles",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   46,
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
							Line:   22,
							Column: 3,
						},
						TagName: "a",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_article_card_48687e83"),
							OriginalSourcePath:   new("partials/article_card.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "article_card_article_article_2d70a8c0",
								PartialAlias:        "article_card",
								PartialPackageName:  "partials_article_card_48687e83",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   47,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"article": ast_domain.PropValue{
										Expression: &ast_domain.Identifier{
											Name: "article",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.Article"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "article",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.Article"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "article",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("article"),
												},
												BaseCodeGenVarName: new("article"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Location: ast_domain.Location{
											Line:   51,
											Column: 23,
										},
										GoFieldName: "Article",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("models.Article"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "article",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 23,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.Article"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "article",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 23,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("article"),
											},
											BaseCodeGenVarName: new("article"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									IsLoopDependent: true,
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"article": "main_aaf9a2e0",
								"href":    "partials_article_card_48687e83",
							},
						},
						DirFor: &ast_domain.Directive{
							Type: ast_domain.DirectiveFor,
							Location: ast_domain.Location{
								Line:   49,
								Column: 20,
							},
							NameLocation: ast_domain.Location{
								Line:   49,
								Column: 13,
							},
							RawExpression: "article in state.Articles",
							Expression: &ast_domain.ForInExpression{
								IndexVariable: nil,
								ItemVariable: &ast_domain.Identifier{
									Name: "article",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("models.Article"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "article",
											ReferenceLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("article"),
										OriginalSourcePath: new("partials/article_card.pk"),
									},
								},
								Collection: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 12,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Articles",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 18,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]models.Article"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Articles",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 20,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										},
									},
									Optional: false,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 12,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]models.Article"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Articles",
											ReferenceLocation: ast_domain.Location{
												Line:   49,
												Column: 20,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									},
								},
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]models.Article"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Articles",
										ReferenceLocation: ast_domain.Location{
											Line:   49,
											Column: 20,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("[]models.Article"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Articles",
										ReferenceLocation: ast_domain.Location{
											Line:   49,
											Column: 20,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("[]models.Article"),
									PackageAlias:         "models",
									CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Articles",
									ReferenceLocation: ast_domain.Location{
										Line:   49,
										Column: 20,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   31,
										Column: 23,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
							},
						},
						DirKey: &ast_domain.Directive{
							Type: ast_domain.DirectiveKey,
							Location: ast_domain.Location{
								Line:   50,
								Column: 20,
							},
							NameLocation: ast_domain.Location{
								Line:   50,
								Column: 13,
							},
							RawExpression: "article.Slug",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "article",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("models.Article"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "article",
											ReferenceLocation: ast_domain.Location{
												Line:   24,
												Column: 11,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("article"),
										OriginalSourcePath: new("partials/article_card.pk"),
									},
								},
								Property: &ast_domain.Identifier{
									Name: "Slug",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Slug",
											ReferenceLocation: ast_domain.Location{
												Line:   50,
												Column: 20,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   41,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("article"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/article.go"),
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
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Slug",
										ReferenceLocation: ast_domain.Location{
											Line:   24,
											Column: 11,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   49,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("article"),
									OriginalSourcePath:  new("partials/article_card.pk"),
									GeneratedSourcePath: new("models/article.go"),
									Stringability:       1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "models",
									CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Slug",
									ReferenceLocation: ast_domain.Location{
										Line:   50,
										Column: 20,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   41,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("article"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("models/article.go"),
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
										OriginalSourcePath: new("partials/article_card.pk"),
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
										OriginalSourcePath: new("partials/article_card.pk"),
										Stringability:      1,
									},
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "article",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.Article"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "article",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 11,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("article"),
												OriginalSourcePath: new("partials/article_card.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Slug",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Slug",
													ReferenceLocation: ast_domain.Location{
														Line:   50,
														Column: 20,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   41,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("article"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/article.go"),
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
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Slug",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 11,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   49,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("article"),
											OriginalSourcePath:  new("partials/article_card.pk"),
											GeneratedSourcePath: new("models/article.go"),
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
								OriginalSourcePath: new("partials/article_card.pk"),
								Stringability:      1,
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "article",
								RawExpression: "article",
								Expression: &ast_domain.Identifier{
									Name: "article",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("models.Article"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "article",
											ReferenceLocation: ast_domain.Location{
												Line:   51,
												Column: 23,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
										},
										BaseCodeGenVarName: new("article"),
										OriginalSourcePath: new("main.pk"),
									},
								},
								Location: ast_domain.Location{
									Line:   51,
									Column: 23,
								},
								NameLocation: ast_domain.Location{
									Line:   51,
									Column: 13,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("models.Article"),
										PackageAlias:         "models",
										CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "article",
										ReferenceLocation: ast_domain.Location{
											Line:   51,
											Column: 23,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
									},
									BaseCodeGenVarName: new("article"),
									OriginalSourcePath: new("main.pk"),
								},
							},
							ast_domain.DynamicAttribute{
								Name:          "href",
								RawExpression: "`/articles/${props.Article.Slug}`",
								Expression: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
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
												OriginalSourcePath: new("partials/article_card.pk"),
												Stringability:      1,
											},
											Literal: "/articles/",
										},
										ast_domain.TemplateLiteralPart{
											IsLiteral: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 12,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/article_card.pk"),
												Stringability:      1,
											},
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
																TypeExpression:       typeExprFromString("partials_article_card_48687e83.Props"),
																PackageAlias:         "partials_article_card_48687e83",
																CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/dist/partials/partials_article_card_48687e83",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_article_card_article_article_2d70a8c0"),
															OriginalSourcePath: new("partials/article_card.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Article",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.Article"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Article",
																ReferenceLocation: ast_domain.Location{
																	Line:   22,
																	Column: 13,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.Article"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "article",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
																		Column: 23,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("article"),
															},
															BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
															OriginalSourcePath:  new("partials/article_card.pk"),
															GeneratedSourcePath: new("dist/partials/partials_article_card_48687e83/generated.go"),
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
															TypeExpression:       typeExprFromString("models.Article"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Article",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 13,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.Article"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "article",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("article"),
														},
														BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
														OriginalSourcePath:  new("partials/article_card.pk"),
														GeneratedSourcePath: new("dist/partials/partials_article_card_48687e83/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Slug",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Slug",
															ReferenceLocation: ast_domain.Location{
																Line:   22,
																Column: 13,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   49,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
														OriginalSourcePath:  new("partials/article_card.pk"),
														GeneratedSourcePath: new("models/article.go"),
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
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Slug",
														ReferenceLocation: ast_domain.Location{
															Line:   22,
															Column: 13,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
													OriginalSourcePath:  new("partials/article_card.pk"),
													GeneratedSourcePath: new("models/article.go"),
													Stringability:       1,
												},
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
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/article_card.pk"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   22,
									Column: 13,
								},
								NameLocation: ast_domain.Location{
									Line:   22,
									Column: 6,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									OriginalSourcePath: new("partials/article_card.pk"),
									Stringability:      1,
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
								TagName: "article",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_article_card_48687e83"),
									OriginalSourcePath:   new("partials/article_card.pk"),
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
												OriginalSourcePath: new("partials/article_card.pk"),
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
												OriginalSourcePath: new("partials/article_card.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "article",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.Article"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "article",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 11,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("article"),
														OriginalSourcePath: new("partials/article_card.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Slug",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Slug",
															ReferenceLocation: ast_domain.Location{
																Line:   50,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   41,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("article"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/article.go"),
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
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Slug",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 11,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   49,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("article"),
													OriginalSourcePath:  new("partials/article_card.pk"),
													GeneratedSourcePath: new("models/article.go"),
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
												OriginalSourcePath: new("partials/article_card.pk"),
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
										OriginalSourcePath: new("partials/article_card.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "article-card",
										Location: ast_domain.Location{
											Line:   23,
											Column: 21,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 14,
										},
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
											OriginalPackageAlias: new("partials_article_card_48687e83"),
											OriginalSourcePath:   new("partials/article_card.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
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
														OriginalSourcePath: new("partials/article_card.pk"),
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
														OriginalSourcePath: new("partials/article_card.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "article",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.Article"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "article",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 11,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("article"),
																OriginalSourcePath: new("partials/article_card.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Slug",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Slug",
																	ReferenceLocation: ast_domain.Location{
																		Line:   50,
																		Column: 20,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   41,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("article"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/article.go"),
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
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Slug",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 11,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   49,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("article"),
															OriginalSourcePath:  new("partials/article_card.pk"),
															GeneratedSourcePath: new("models/article.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
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
														OriginalSourcePath: new("partials/article_card.pk"),
														Stringability:      1,
													},
													Literal: ":0:0",
												},
											},
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
												OriginalSourcePath: new("partials/article_card.pk"),
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
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_article_card_48687e83"),
													OriginalSourcePath:   new("partials/article_card.pk"),
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/article_card.pk"),
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
																OriginalSourcePath: new("partials/article_card.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.MemberExpression{
																Base: &ast_domain.Identifier{
																	Name: "article",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 1,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("models.Article"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "article",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 11,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("article"),
																		OriginalSourcePath: new("partials/article_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Slug",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 9,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Slug",
																			ReferenceLocation: ast_domain.Location{
																				Line:   50,
																				Column: 20,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   41,
																				Column: 2,
																			},
																		},
																		BaseCodeGenVarName:  new("article"),
																		OriginalSourcePath:  new("main.pk"),
																		GeneratedSourcePath: new("models/article.go"),
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
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Slug",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 11,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   49,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("article"),
																	OriginalSourcePath:  new("partials/article_card.pk"),
																	GeneratedSourcePath: new("models/article.go"),
																	Stringability:       1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
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
																OriginalSourcePath: new("partials/article_card.pk"),
																Stringability:      1,
															},
															Literal: ":0:0:0",
														},
													},
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
														OriginalSourcePath: new("partials/article_card.pk"),
														Stringability:      1,
													},
												},
												RichText: []ast_domain.TextPart{
													ast_domain.TextPart{
														IsLiteral: false,
														Location: ast_domain.Location{
															Line:   24,
															Column: 14,
														},
														RawExpression: "props.Article.Title",
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
																			TypeExpression:       typeExprFromString("partials_article_card_48687e83.Props"),
																			PackageAlias:         "partials_article_card_48687e83",
																			CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/dist/partials/partials_article_card_48687e83",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "props",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 14,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("props_article_card_article_article_2d70a8c0"),
																		OriginalSourcePath: new("partials/article_card.pk"),
																	},
																},
																Property: &ast_domain.Identifier{
																	Name: "Article",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 7,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("models.Article"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "Article",
																			ReferenceLocation: ast_domain.Location{
																				Line:   24,
																				Column: 14,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   37,
																				Column: 2,
																			},
																		},
																		PropDataSource: &ast_domain.PropDataSource{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("models.Article"),
																				PackageAlias:         "models",
																				CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "article",
																				ReferenceLocation: ast_domain.Location{
																					Line:   51,
																					Column: 23,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("article"),
																		},
																		BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
																		OriginalSourcePath:  new("partials/article_card.pk"),
																		GeneratedSourcePath: new("dist/partials/partials_article_card_48687e83/generated.go"),
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
																		TypeExpression:       typeExprFromString("models.Article"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Article",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 14,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   37,
																			Column: 2,
																		},
																	},
																	PropDataSource: &ast_domain.PropDataSource{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("models.Article"),
																			PackageAlias:         "models",
																			CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "article",
																			ReferenceLocation: ast_domain.Location{
																				Line:   51,
																				Column: 23,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("article"),
																	},
																	BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
																	OriginalSourcePath:  new("partials/article_card.pk"),
																	GeneratedSourcePath: new("dist/partials/partials_article_card_48687e83/generated.go"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Title",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 15,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "models",
																		CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Title",
																		ReferenceLocation: ast_domain.Location{
																			Line:   24,
																			Column: 14,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   50,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
																	OriginalSourcePath:  new("partials/article_card.pk"),
																	GeneratedSourcePath: new("models/article.go"),
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
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Title",
																	ReferenceLocation: ast_domain.Location{
																		Line:   24,
																		Column: 14,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   50,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
																OriginalSourcePath:  new("partials/article_card.pk"),
																GeneratedSourcePath: new("models/article.go"),
																Stringability:       1,
															},
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_61_p_for_on_partial_with_dynamic_attr_root/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 14,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   50,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_article_card_article_article_2d70a8c0"),
															OriginalSourcePath:  new("partials/article_card.pk"),
															GeneratedSourcePath: new("models/article.go"),
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
