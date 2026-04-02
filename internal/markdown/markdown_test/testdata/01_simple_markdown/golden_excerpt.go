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
					Line:   2,
					Column: 13,
				},
				TagName: "p",
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   2,
							Column: 13,
						},
						TextContent: "This is a simple paragraph with ",
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   4,
							Column: 2,
						},
						TagName: "strong",
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   4,
									Column: 2,
								},
								TextContent: "bold text",
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   4,
							Column: 13,
						},
						TextContent: " and a ",
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   5,
							Column: 2,
						},
						TagName: "a",
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "href",
								Value: "https://piko.sh",
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   5,
									Column: 2,
								},
								TextContent: "link to the Piko project",
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeText,
						Location: ast_domain.Location{
							Line:   10,
							Column: 7,
						},
						TextContent: ".",
					},
				},
			},
		},
	}
}()
