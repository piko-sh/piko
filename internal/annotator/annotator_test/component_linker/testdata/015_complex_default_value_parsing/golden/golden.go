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
					OriginalPackageAlias: new("partials_avatar_c3a790d9"),
					OriginalSourcePath:   new("partials/avatar.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "avatar_options_partials_avatar_c3a790d9_getdefaultoptions_ad40dd58",
						PartialAlias:        "avatar",
						PartialPackageName:  "partials_avatar_c3a790d9",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"options": ast_domain.PropValue{
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "partials_avatar_c3a790d9",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       nil,
													PackageAlias:         "partials_avatar_c3a790d9",
													CanonicalPackagePath: "testcase_15_complex_default_value_parsing/dist/partials/partials_avatar_c3a790d9",
												},
												BaseCodeGenVarName: new("partials_avatar_c3a790d9"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "GetDefaultOptions",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "partials_avatar_c3a790d9",
													CanonicalPackagePath: "testcase_15_complex_default_value_parsing/dist/partials/partials_avatar_c3a790d9",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "GetDefaultOptions",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("partials_avatar_c3a790d9"),
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
												PackageAlias:         "partials_avatar_c3a790d9",
												CanonicalPackagePath: "testcase_15_complex_default_value_parsing/dist/partials/partials_avatar_c3a790d9",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "GetDefaultOptions",
												ReferenceLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("partials_avatar_c3a790d9"),
										},
									},
									Args: []ast_domain.Expression{},
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("models.Options"),
											PackageAlias:         "models",
											CanonicalPackagePath: "",
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("models.Options"),
												PackageAlias:         "models",
												CanonicalPackagePath: "",
											},
											BaseCodeGenVarName: new("partials_avatar_c3a790d9"),
										},
										BaseCodeGenVarName: new("partials_avatar_c3a790d9"),
									},
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "Options",
							},
						},
					},
				},
			},
		},
	}
}()
