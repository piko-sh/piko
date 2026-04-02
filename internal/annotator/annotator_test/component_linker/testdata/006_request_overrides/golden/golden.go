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
					OriginalPackageAlias: new("partials_pager_66521b00"),
					OriginalSourcePath:   new("partials/pager.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "pager_page_request_queryparam_p_5ad0d82f",
						PartialAlias:        "pager",
						PartialPackageName:  "partials_pager_66521b00",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						RequestOverrides: map[string]ast_domain.PropValue{
							"page": ast_domain.PropValue{
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "request",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*piko.RequestData"),
													PackageAlias:         "piko",
													CanonicalPackagePath: "piko.sh/piko",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "request",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("r"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "QueryParam",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "piko",
													CanonicalPackagePath: "piko.sh/piko",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "QueryParam",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   312,
														Column: 1,
													},
												},
												BaseCodeGenVarName:  new("r"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("../../../../../../templater/templater_dto/request.go"),
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
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "piko",
												CanonicalPackagePath: "piko.sh/piko",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "QueryParam",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   312,
													Column: 1,
												},
											},
											BaseCodeGenVarName:  new("r"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("../../../../../../templater/templater_dto/request.go"),
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.StringLiteral{
											Value: "p",
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
									},
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "piko",
											CanonicalPackagePath: "",
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "piko",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("r"),
										},
										BaseCodeGenVarName: new("r"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "",
							},
						},
						PassedProps: map[string]ast_domain.PropValue{
							"page": ast_domain.PropValue{
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "request",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("*piko.RequestData"),
													PackageAlias:         "piko",
													CanonicalPackagePath: "piko.sh/piko",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "request",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("r"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "QueryParam",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "piko",
													CanonicalPackagePath: "piko.sh/piko",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "QueryParam",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   312,
														Column: 1,
													},
												},
												BaseCodeGenVarName:  new("r"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("../../../../../../templater/templater_dto/request.go"),
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
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "piko",
												CanonicalPackagePath: "piko.sh/piko",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "QueryParam",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   312,
													Column: 1,
												},
											},
											BaseCodeGenVarName:  new("r"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("../../../../../../templater/templater_dto/request.go"),
										},
									},
									Args: []ast_domain.Expression{
										&ast_domain.StringLiteral{
											Value: "p",
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
									},
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "piko",
											CanonicalPackagePath: "",
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "piko",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("r"),
										},
										BaseCodeGenVarName: new("r"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "Page",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "piko",
										CanonicalPackagePath: "",
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "piko",
											CanonicalPackagePath: "",
										},
										BaseCodeGenVarName: new("r"),
									},
									BaseCodeGenVarName: new("r"),
									Stringability:      1,
								},
							},
						},
					},
				},
			},
		},
	}
}()
