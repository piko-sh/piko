package testgolden

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
					Line:   31,
					Column: 5,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_comp_a_235c5bab"),
					OriginalSourcePath:   new("partials/comp_a.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "a_d24ec4f1",
						PartialAlias:        "a",
						PartialPackageName:  "partials_comp_a_235c5bab",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   31,
							Column: 5,
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   31,
						Column: 5,
					},
					GoAnnotations: nil,
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "comp-a",
						Location: ast_domain.Location{
							Line:   31,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   31,
							Column: 10,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   25,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_comp_b_81b49c90"),
							OriginalSourcePath:   new("partials/comp_b.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "b_78452aa1",
								PartialAlias:        "b",
								PartialPackageName:  "partials_comp_b_81b49c90",
								InvokerPackageAlias: "partials_comp_a_235c5bab",
								Location: ast_domain.Location{
									Line:   32,
									Column: 9,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   25,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "comp-b",
								Location: ast_domain.Location{
									Line:   25,
									Column: 15,
								},
								NameLocation: ast_domain.Location{
									Line:   25,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   25,
									Column: 23,
								},
								TextContent: "Component B",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_comp_b_81b49c90"),
									OriginalSourcePath:   new("partials/comp_b.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   25,
										Column: 23,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
				},
			},
		},
	}
}()
