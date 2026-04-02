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
						InvocationKey:       "card_title_hello_96ccd4bf",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"title": ast_domain.PropValue{
								Expression: &ast_domain.StringLiteral{
									Value: "Hello",
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
										PropDataSource: &ast_domain.PropDataSource{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
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
								GoFieldName: "PrimaryTitle",
								InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "",
										CanonicalPackagePath: "",
									},
									PropDataSource: &ast_domain.PropDataSource{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
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
			},
		},
	}
}()
