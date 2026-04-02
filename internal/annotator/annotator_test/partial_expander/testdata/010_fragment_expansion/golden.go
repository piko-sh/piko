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
					Line:   28,
					Column: 5,
				},
				TagName: "section",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   28,
						Column: 5,
					},
					GoAnnotations: nil,
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeFragment,
						Location: ast_domain.Location{
							Line:   0,
							Column: 0,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "multi_8df50fb8",
								PartialAlias:        "multi",
								PartialPackageName:  "partials_multi_root_bfb66450",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   29,
									Column: 9,
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   0,
								Column: 0,
							},
							GoAnnotations: nil,
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   22,
									Column: 5,
								},
								TagName: "h1",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_multi_root_bfb66450"),
									OriginalSourcePath:   new("partials/multi_root.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 5,
									},
									GoAnnotations: nil,
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "p-fragment",
										Value: "multi_8df50fb8",
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
										Name:  "p-fragment-id",
										Value: "0",
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
											Line:   22,
											Column: 9,
										},
										TextContent: "Title",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_multi_root_bfb66450"),
											OriginalSourcePath:   new("partials/multi_root.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   22,
												Column: 9,
											},
											GoAnnotations: nil,
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   23,
									Column: 5,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_multi_root_bfb66450"),
									OriginalSourcePath:   new("partials/multi_root.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 5,
									},
									GoAnnotations: nil,
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "p-fragment",
										Value: "multi_8df50fb8",
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
										Name:  "p-fragment-id",
										Value: "1",
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
											Line:   23,
											Column: 8,
										},
										TextContent: "Paragraph one.",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_multi_root_bfb66450"),
											OriginalSourcePath:   new("partials/multi_root.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 8,
											},
											GoAnnotations: nil,
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_multi_root_bfb66450"),
									OriginalSourcePath:   new("partials/multi_root.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:2",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 5,
									},
									GoAnnotations: nil,
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "p-fragment",
										Value: "multi_8df50fb8",
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
										Name:  "p-fragment-id",
										Value: "2",
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
											Line:   24,
											Column: 8,
										},
										TextContent: "Paragraph two.",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_multi_root_bfb66450"),
											OriginalSourcePath:   new("partials/multi_root.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 8,
											},
											GoAnnotations: nil,
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
