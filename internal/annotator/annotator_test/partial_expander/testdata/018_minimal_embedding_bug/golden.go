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
					Column: 3,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_data_table_a736e31a"),
					OriginalSourcePath:   new("partials/data_table.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "data_table_3e7317fb",
						PartialAlias:        "data_table",
						PartialPackageName:  "partials_data_table_a736e31a",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   22,
						Column: 3,
					},
					GoAnnotations: nil,
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "table-wrapper",
						Location: ast_domain.Location{
							Line:   22,
							Column: 15,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 8,
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   23,
							Column: 5,
						},
						TagName: "table",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_data_table_a736e31a"),
							OriginalSourcePath:   new("partials/data_table.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   23,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "data-table",
								Location: ast_domain.Location{
									Line:   23,
									Column: 19,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 7,
								},
								TagName: "thead",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_data_table_a736e31a"),
									OriginalSourcePath:   new("partials/data_table.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 7,
									},
									GoAnnotations: nil,
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 9,
										},
										TagName: "tr",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_data_table_a736e31a"),
											OriginalSourcePath:   new("partials/data_table.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 9,
											},
											GoAnnotations: nil,
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   26,
													Column: 11,
												},
												TagName: "th",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_table_a736e31a"),
													OriginalSourcePath:   new("partials/data_table.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   26,
														Column: 11,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   26,
															Column: 15,
														},
														TextContent: "Name",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_data_table_a736e31a"),
															OriginalSourcePath:   new("partials/data_table.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:0:0:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 15,
															},
															GoAnnotations: nil,
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   27,
													Column: 11,
												},
												TagName: "th",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_table_a736e31a"),
													OriginalSourcePath:   new("partials/data_table.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:0:0:1",
													RelativeLocation: ast_domain.Location{
														Line:   27,
														Column: 11,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   27,
															Column: 15,
														},
														TextContent: "Status",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_data_table_a736e31a"),
															OriginalSourcePath:   new("partials/data_table.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:0:0:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 15,
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
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   30,
									Column: 7,
								},
								TagName: "tbody",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_data_table_a736e31a"),
									OriginalSourcePath:   new("partials/data_table.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   30,
										Column: 7,
									},
									GoAnnotations: nil,
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   31,
											Column: 9,
										},
										TagName: "tr",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_data_table_a736e31a"),
											OriginalSourcePath:   new("partials/data_table.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   31,
												Column: 9,
											},
											GoAnnotations: nil,
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   32,
													Column: 11,
												},
												TagName: "td",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_table_a736e31a"),
													OriginalSourcePath:   new("partials/data_table.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   32,
														Column: 11,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   32,
															Column: 15,
														},
														TextContent: "Item 1",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_data_table_a736e31a"),
															OriginalSourcePath:   new("partials/data_table.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:1:0:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   32,
																Column: 15,
															},
															GoAnnotations: nil,
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   33,
													Column: 11,
												},
												TagName: "td",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_table_a736e31a"),
													OriginalSourcePath:   new("partials/data_table.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:0:1",
													RelativeLocation: ast_domain.Location{
														Line:   33,
														Column: 11,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   33,
															Column: 15,
														},
														TextContent: "Active",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_data_table_a736e31a"),
															OriginalSourcePath:   new("partials/data_table.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:1:0:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   33,
																Column: 15,
															},
															GoAnnotations: nil,
														},
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   35,
											Column: 9,
										},
										TagName: "tr",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_data_table_a736e31a"),
											OriginalSourcePath:   new("partials/data_table.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:1",
											RelativeLocation: ast_domain.Location{
												Line:   35,
												Column: 9,
											},
											GoAnnotations: nil,
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   36,
													Column: 11,
												},
												TagName: "td",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_table_a736e31a"),
													OriginalSourcePath:   new("partials/data_table.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   36,
														Column: 11,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   36,
															Column: 15,
														},
														TextContent: "Item 2",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_data_table_a736e31a"),
															OriginalSourcePath:   new("partials/data_table.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:1:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   36,
																Column: 15,
															},
															GoAnnotations: nil,
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   37,
													Column: 11,
												},
												TagName: "td",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_data_table_a736e31a"),
													OriginalSourcePath:   new("partials/data_table.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:0:1:1:1",
													RelativeLocation: ast_domain.Location{
														Line:   37,
														Column: 11,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   37,
															Column: 15,
														},
														TextContent: "Inactive",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_data_table_a736e31a"),
															OriginalSourcePath:   new("partials/data_table.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:0:1:1:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   37,
																Column: 15,
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
					},
				},
			},
		},
	}
}()
