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
				TagName: "h1",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_wrapper_54f54229"),
					OriginalSourcePath:   new("partials/wrapper.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "",
						PartialAlias:        "",
						PartialPackageName:  "partials_wrapper_54f54229",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
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
							OriginalPackageAlias: new("partials_wrapper_54f54229"),
							OriginalSourcePath:   new("partials/wrapper.pk"),
						},
						RichText: []ast_domain.TextPart{
							ast_domain.TextPart{
								IsLiteral: false,
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								RawExpression: "state.WrapperTitle",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("partials_wrapper_54f54229.Response"),
												PackageAlias:         "partials_wrapper_54f54229",
												CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/partials/partials_wrapper_54f54229",
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
											BaseCodeGenVarName: new("partials_wrapper_54f54229Data_"),
											OriginalSourcePath: new("partials/wrapper.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "WrapperTitle",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "partials_wrapper_54f54229",
												CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/partials/partials_wrapper_54f54229",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "WrapperTitle",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   25,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("partials_wrapper_54f54229Data_"),
											OriginalSourcePath:  new("partials/wrapper.pk"),
											GeneratedSourcePath: new("dist/partials/partials_wrapper_54f54229/generated.go"),
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
											PackageAlias:         "partials_wrapper_54f54229",
											CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/partials/partials_wrapper_54f54229",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "WrapperTitle",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   25,
												Column: 23,
											},
										},
										BaseCodeGenVarName:  new("partials_wrapper_54f54229Data_"),
										OriginalSourcePath:  new("partials/wrapper.pk"),
										GeneratedSourcePath: new("dist/partials/partials_wrapper_54f54229/generated.go"),
										Stringability:       1,
									},
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "partials_wrapper_54f54229",
										CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/partials/partials_wrapper_54f54229",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "WrapperTitle",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   25,
											Column: 23,
										},
									},
									BaseCodeGenVarName:  new("partials_wrapper_54f54229Data_"),
									OriginalSourcePath:  new("partials/wrapper.pk"),
									GeneratedSourcePath: new("dist/partials/partials_wrapper_54f54229/generated.go"),
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
					Line:   0,
					Column: 0,
				},
				TagName: "main",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_wrapper_54f54229"),
					OriginalSourcePath:   new("partials/wrapper.pk"),
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										RawExpression: "state.MainMessage",
										Expression: &ast_domain.MemberExpression{
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
														CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/pages/main_aaf9a2e0",
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
												Name: "MainMessage",
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MainMessage",
														ReferenceLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   30,
															Column: 23,
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
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "MainMessage",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   30,
														Column: 23,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_03_slotted_content_context_switch/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "MainMessage",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   30,
													Column: 23,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
