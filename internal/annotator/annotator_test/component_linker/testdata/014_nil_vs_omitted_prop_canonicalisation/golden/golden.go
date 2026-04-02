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
					OriginalPackageAlias: new("partials_profile_c5502d03"),
					OriginalSourcePath:   new("partials/profile.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "profile_user_nil_52ce3984",
						PartialAlias:        "profile",
						PartialPackageName:  "partials_profile_c5502d03",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"user": ast_domain.PropValue{
								Expression: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("nil"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "User",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("nil"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("nil"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
									},
									OriginalSourcePath: new("main.pk"),
									Stringability:      1,
								},
							},
						},
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "explicit-nil",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast_domain.Location{
							Line:   0,
							Column: 0,
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
					OriginalPackageAlias: new("partials_profile_c5502d03"),
					OriginalSourcePath:   new("partials/profile.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "profile_user_nil_52ce3984",
						PartialAlias:        "profile",
						PartialPackageName:  "partials_profile_c5502d03",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"user": ast_domain.PropValue{
								Expression: &ast_domain.NilLiteral{
									RelativeLocation: ast_domain.Location{
										Line:   0,
										Column: 0,
									},
									GoAnnotations: nil,
								},
								Location: ast_domain.Location{
									Line:   0,
									Column: 0,
								},
								GoFieldName: "User",
							},
						},
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "omitted",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
					},
				},
			},
		},
	}
}()
