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
					Line:   22,
					Column: 5,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_card_bfc4a3cf"),
					OriginalSourcePath:   new("partials/card.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_data_main_id_new_id_b528641c",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   28,
							Column: 5,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"data-main": ast_domain.PropValue{
								Expression: &ast_domain.StringLiteral{
									Value: "",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: nil,
								},
								Location: ast_domain.Location{
									Line:   28,
									Column: 63,
								},
								GoFieldName: "",
							},
							"id": ast_domain.PropValue{
								Expression: &ast_domain.StringLiteral{
									Value: "new-id",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: nil,
								},
								Location: ast_domain.Location{
									Line:   28,
									Column: 33,
								},
								GoFieldName: "",
							},
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   22,
						Column: 5,
					},
					GoAnnotations: nil,
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "aria-label",
						Value: "Default label",
						Location: ast_domain.Location{
							Line:   22,
							Column: 53,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 41,
						},
					},
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "card-class invoker-class",
						Location: ast_domain.Location{
							Line:   22,
							Column: 29,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 22,
						},
					},
					ast_domain.HTMLAttribute{
						Name:  "data-main",
						Value: "",
						Location: ast_domain.Location{
							Line:   28,
							Column: 63,
						},
						NameLocation: ast_domain.Location{
							Line:   28,
							Column: 63,
						},
					},
					ast_domain.HTMLAttribute{
						Name:  "id",
						Value: "new-id",
						Location: ast_domain.Location{
							Line:   28,
							Column: 33,
						},
						NameLocation: ast_domain.Location{
							Line:   28,
							Column: 29,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   23,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   23,
								Column: 9,
							},
							GoAnnotations: nil,
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   23,
									Column: 12,
								},
								TextContent: "Attribute Test",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 12,
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
