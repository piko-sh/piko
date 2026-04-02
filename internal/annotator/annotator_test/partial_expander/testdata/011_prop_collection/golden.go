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
						InvocationKey:       "card_req_method_state_method_dynamic_prop_state_userobject_server_user_state_userobject_static_prop_hello_world_8bb75066",
						PartialAlias:        "card",
						PartialPackageName:  "partials_card_bfc4a3cf",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   33,
							Column: 5,
						},
						RequestOverrides: map[string]ast_domain.PropValue{
							"method": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Method",
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
									Line:   37,
									Column: 26,
								},
								GoFieldName: "",
							},
						},
						PassedProps: map[string]ast_domain.PropValue{
							"dynamic-prop": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "UserObject",
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
									Line:   35,
									Column: 24,
								},
								GoFieldName: "",
							},
							"server.user": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "UserObject",
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
									Line:   36,
									Column: 23,
								},
								GoFieldName: "",
							},
							"static-prop": ast_domain.PropValue{
								Expression: &ast_domain.StringLiteral{
									Value: "hello world",
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: nil,
								},
								Location: ast_domain.Location{
									Line:   34,
									Column: 22,
								},
								GoFieldName: "",
							},
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"dynamic-prop": "main_aaf9a2e0",
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
						Name:  "class",
						Value: "card",
						Location: ast_domain.Location{
							Line:   22,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   22,
							Column: 10,
						},
					},
					ast_domain.HTMLAttribute{
						Name:  "static-prop",
						Value: "hello world",
						Location: ast_domain.Location{
							Line:   34,
							Column: 22,
						},
						NameLocation: ast_domain.Location{
							Line:   34,
							Column: 9,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "dynamic-prop",
						RawExpression: "state.UserObject",
						Expression: &ast_domain.MemberExpression{
							Base: &ast_domain.Identifier{
								Name: "state",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
							},
							Property: &ast_domain.Identifier{
								Name: "UserObject",
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
							Line:   35,
							Column: 24,
						},
						NameLocation: ast_domain.Location{
							Line:   35,
							Column: 9,
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
								TextContent: "Check PartialInfo annotation on this node.",
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
