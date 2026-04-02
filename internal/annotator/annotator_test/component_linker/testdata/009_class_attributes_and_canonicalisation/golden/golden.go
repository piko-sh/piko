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
						InvocationKey:       "card_title_same_title_d13b7192",
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
									Value: "Same Title",
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
								GoFieldName: "Title",
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
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "mt-4",
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
					},
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "card1",
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
					OriginalPackageAlias: new("partials_card_bfc4a3cf"),
					OriginalSourcePath:   new("partials/card.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_title_same_title_d13b7192",
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
									Value: "Same Title",
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
								GoFieldName: "Title",
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
				Directives: []ast_domain.Directive{
					ast_domain.Directive{
						Type: ast_domain.DirectiveClass,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						NameLocation: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						Expression: &ast_domain.ObjectLiteral{
							Pairs: map[string]ast_domain.Expression{},
							RelativeLocation: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
							GoAnnotations: nil,
						},
					},
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "card2",
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
