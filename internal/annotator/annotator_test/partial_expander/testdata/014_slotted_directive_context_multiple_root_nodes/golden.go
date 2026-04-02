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
				NodeType: ast_domain.NodeFragment,
				Location: ast_domain.Location{
					Line:   0,
					Column: 0,
				},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "card_attribute_test_attribute2_state_shouldappear_5aa31378",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   33,
							Column: 5,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"attribute": ast_domain.PropValue{
								Expression: &ast_domain.StringLiteral{
									Value: "test",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: nil,
								},
								Location: ast_domain.Location{
									Line:   33,
									Column: 68,
								},
								GoFieldName: "",
							},
							"attribute2": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "ShouldAppear",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
									},
									Optional: false,
									Computed: false,
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
								},
								Location: ast_domain.Location{
									Line:   33,
									Column: 87,
								},
								GoFieldName: "",
							},
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"attribute2": "main_aaf9a2e0",
					},
				},
				DirShow: &ast_domain.Directive{
					Type: ast_domain.DirectiveShow,
					Location: ast_domain.Location{
						Line:   33,
						Column: 37,
					},
					NameLocation: ast_domain.Location{
						Line:   33,
						Column: 29,
					},
					RawExpression: "state.ShouldAppear",
					Expression: &ast_domain.MemberExpression{
						Base: &ast_domain.Identifier{
							Name: "state",
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 1,
							},
						},
						Property: &ast_domain.Identifier{
							Name: "ShouldAppear",
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 7,
							},
						},
						Optional: false,
						Computed: false,
						RelativeLocation: ast_domain.Location{
							Line:   1,
							Column: 1,
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						OriginalPackageAlias: new("main_aaf9a2e0"),
						OriginalSourcePath:   new("main.pk"),
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   0,
						Column: 0,
					},
					GoAnnotations: nil,
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "attribute",
						Value: "test",
						Location: ast_domain.Location{
							Line:   33,
							Column: 68,
						},
						NameLocation: ast_domain.Location{
							Line:   33,
							Column: 57,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "attribute2",
						RawExpression: "state.ShouldAppear",
						Expression: &ast_domain.MemberExpression{
							Base: &ast_domain.Identifier{
								Name: "state",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
							},
							Property: &ast_domain.Identifier{
								Name: "ShouldAppear",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 7,
								},
							},
							Optional: false,
							Computed: false,
							RelativeLocation: ast_domain.Location{
								Line:   1,
								Column: 1,
							},
						},
						Location: ast_domain.Location{
							Line:   33,
							Column: 87,
						},
						NameLocation: ast_domain.Location{
							Line:   33,
							Column: 74,
						},
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   30,
							Column: 5,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   30,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "card-title",
								Location: ast_domain.Location{
									Line:   30,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   30,
									Column: 9,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "card_attribute_test_attribute2_state_shouldappear_5aa31378",
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
									Line:   30,
									Column: 28,
								},
								TextContent: "Multi-Root Card",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_card_bfc4a3cf"),
									OriginalSourcePath:   new("partials/card.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   30,
										Column: 28,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   32,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_card_bfc4a3cf"),
							OriginalSourcePath:   new("partials/card.pk"),
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   32,
								Column: 34,
							},
							NameLocation: ast_domain.Location{
								Line:   32,
								Column: 28,
							},
							RawExpression: "state.IsActive",
							Expression: &ast_domain.MemberExpression{
								Base: &ast_domain.Identifier{
									Name: "state",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
								},
								Property: &ast_domain.Identifier{
									Name: "IsActive",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 7,
									},
								},
								Optional: false,
								Computed: false,
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								OriginalSourcePath: new("partials/card.pk"),
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   32,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "card-body",
								Location: ast_domain.Location{
									Line:   32,
									Column: 17,
								},
								NameLocation: ast_domain.Location{
									Line:   32,
									Column: 10,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "p-fragment",
								Value: "card_attribute_test_attribute2_state_shouldappear_5aa31378",
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
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   34,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   34,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   34,
										Column: 12,
									},
									RawExpression: "state.ShouldShow",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "state",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
										},
										Property: &ast_domain.Identifier{
											Name: "ShouldShow",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 7,
											},
										},
										Optional: false,
										Computed: false,
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										OriginalSourcePath: new("main.pk"),
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   34,
										Column: 9,
									},
									GoAnnotations: nil,
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   34,
											Column: 36,
										},
										TextContent: "Visible from Main",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   34,
												Column: 36,
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
