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
					OriginalPackageAlias: new("partials_card_bfc4a3cf"),
					OriginalSourcePath:   new("partials/card.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_id_state_currentid_b6e2e4ef",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"id": ast_domain.PropValue{
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
												CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
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
										Name: "CurrentID",
										RelativeLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CurrentID",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "main_aaf9a2e0",
													CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "CurrentID",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   31,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
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
											CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "CurrentID",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 23,
											},
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "CurrentID",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   31,
													Column: 23,
												},
											},
											BaseCodeGenVarName: new("pageData"),
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       1,
									},
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "ID",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "main_aaf9a2e0",
										CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "CurrentID",
										ReferenceLocation: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   31,
											Column: 23,
										},
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "main_aaf9a2e0",
											CanonicalPackagePath: "testcase_08_prop_type_is_local_alias/dist/pages/main_aaf9a2e0",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "CurrentID",
											ReferenceLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   31,
												Column: 23,
											},
										},
										BaseCodeGenVarName: new("pageData"),
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
	}
}()
