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
					Line:   0,
					Column: 0,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				DirIf: &ast_domain.Directive{
					Type: ast_domain.DirectiveIf,
					Location: ast_domain.Location{
						Line:   0,
						Column: 0,
					},
					NameLocation: ast_domain.Location{
						Line:   0,
						Column: 0,
					},
					RawExpression: "state.Count > 'hello'",
					Expression: &ast_domain.BinaryExpression{
						Left: &ast_domain.MemberExpression{
							Base: &ast_domain.Identifier{
								Name: "state",
								RelativeLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_01_basic_resolution/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "state",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
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
								Name: "Count",
								RelativeLocation: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("int"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_01_basic_resolution/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Count",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   32,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       1,
								},
							},
							Optional: false,
							Computed: false,
							RelativeLocation: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("int"),
									PackageAlias:         "main_aaf9a2e0",
									CanonicalPackagePath: "testcase_01_basic_resolution/dist/pages/main_aaf9a2e0",
								},
								Symbol: &ast_domain.ResolvedSymbol{
									Name: "Count",
									ReferenceLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									DeclarationLocation: ast_domain.Location{
										Line:   32,
										Column: 2,
									},
								},
								BaseCodeGenVarName:  new("pageData"),
								OriginalSourcePath:  new("main.pk"),
								GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
								Stringability:       1,
							},
						},
						Operator: ">",
						Right: &ast_domain.StringLiteral{
							Value: "hello",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						RelativeLocation: ast_domain.Location{
							Line:   0,
							Column: 0,
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
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						TextContent: "Error",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
					},
				},
			},
			&ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Location: ast_domain.Location{
					Line:   0,
					Column: 0,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_card_bfc4a3cf"),
					OriginalSourcePath:   new("partials/card.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_title_state_title_somekey",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"title": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Title",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
									},
									Optional: false,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "",
							},
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
						},
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								RawExpression: "props.Title",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "props",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_card_bfc4a3cf.Props"),
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_01_basic_resolution/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "props",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("props_card_title_state_title_somekey"),
											OriginalSourcePath: new("partials/card.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Title",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_card_bfc4a3cf",
												CanonicalPackagePath: "testcase_01_basic_resolution/dist/partials/partials_card_bfc4a3cf",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Title",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   26,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("props_card_title_state_title_somekey"),
											OriginalSourcePath:  new("partials/card.pk"),
											GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
											Stringability:       1,
										},
									},
									Optional: false,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "partials_card_bfc4a3cf",
											CanonicalPackagePath: "testcase_01_basic_resolution/dist/partials/partials_card_bfc4a3cf",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Title",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   26,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("props_card_title_state_title_somekey"),
										OriginalSourcePath:  new("partials/card.pk"),
										GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
										Stringability:       1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_card_bfc4a3cf",
										CanonicalPackagePath: "testcase_01_basic_resolution/dist/partials/partials_card_bfc4a3cf",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "Title",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   26,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("props_card_title_state_title_somekey"),
									OriginalSourcePath:  new("partials/card.pk"),
									GeneratedSourcePath: new("dist/partials/partials_card_bfc4a3cf/generated.go"),
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
