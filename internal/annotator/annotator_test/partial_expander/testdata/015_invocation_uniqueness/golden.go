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
					Line:   33,
					Column: 5,
				},
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   33,
						Column: 5,
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
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_id_c1_1e2685fd",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   34,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "c1",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   34,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "c1",
								Location: ast_domain.Location{
									Line:   34,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   34,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_id_c2_c0cc5cb7",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   35,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "c2",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   35,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "c2",
								Location: ast_domain.Location{
									Line:   35,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   35,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_color_blue_id_s1_0dd4e0fc",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   37,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"color": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "blue",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   37,
											Column: 49,
										},
										GoFieldName: "",
									},
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "s1",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   37,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "color",
								Value: "blue",
								Location: ast_domain.Location{
									Line:   37,
									Column: 49,
								},
								NameLocation: ast_domain.Location{
									Line:   37,
									Column: 42,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "s1",
								Location: ast_domain.Location{
									Line:   37,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   37,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_color_red_id_s2_b63795e7",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   38,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"color": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "red",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   38,
											Column: 49,
										},
										GoFieldName: "",
									},
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "s2",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   38,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "color",
								Value: "red",
								Location: ast_domain.Location{
									Line:   38,
									Column: 49,
								},
								NameLocation: ast_domain.Location{
									Line:   38,
									Column: 42,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "s2",
								Location: ast_domain.Location{
									Line:   38,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   38,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_color_blue_id_s3_41b88ec4",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   39,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"color": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "blue",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   39,
											Column: 49,
										},
										GoFieldName: "",
									},
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "s3",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   39,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "color",
								Value: "blue",
								Location: ast_domain.Location{
									Line:   39,
									Column: 49,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 42,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "s3",
								Location: ast_domain.Location{
									Line:   39,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_count_state_primarycount_id_d1_b64c1729",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   41,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"count": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
											},
											Property: &ast_domain.Identifier{
												Name: "PrimaryCount",
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
											Line:   41,
											Column: 50,
										},
										GoFieldName: "",
									},
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "d1",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   41,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"count": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "d1",
								Location: ast_domain.Location{
									Line:   41,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 34,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "count",
								RawExpression: "state.PrimaryCount",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "PrimaryCount",
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
									Line:   41,
									Column: 50,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 42,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_count_state_secondarycount_id_d2_7c7abf84",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   42,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"count": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
											},
											Property: &ast_domain.Identifier{
												Name: "SecondaryCount",
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
											Line:   42,
											Column: 50,
										},
										GoFieldName: "",
									},
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "d2",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   42,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"count": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "d2",
								Location: ast_domain.Location{
									Line:   42,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   42,
									Column: 34,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "count",
								RawExpression: "state.SecondaryCount",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "SecondaryCount",
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
									Line:   42,
									Column: 50,
								},
								NameLocation: ast_domain.Location{
									Line:   42,
									Column: 42,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_count_state_primarycount_id_d3_a4ab395c",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   43,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"count": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
											},
											Property: &ast_domain.Identifier{
												Name: "PrimaryCount",
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
											Line:   43,
											Column: 50,
										},
										GoFieldName: "",
									},
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "d3",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   43,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"count": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:7",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "d3",
								Location: ast_domain.Location{
									Line:   43,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   43,
									Column: 34,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "count",
								RawExpression: "state.PrimaryCount",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "PrimaryCount",
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
									Line:   43,
									Column: 50,
								},
								NameLocation: ast_domain.Location{
									Line:   43,
									Column: 42,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_id_n1_label_state_label_2b8c5031",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   45,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "n1",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   45,
											Column: 38,
										},
										GoFieldName: "",
									},
									"label": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Label",
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
											Line:   45,
											Column: 50,
										},
										GoFieldName: "",
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"label": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:8",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "n1",
								Location: ast_domain.Location{
									Line:   45,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   45,
									Column: 34,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "label",
								RawExpression: "state.Label",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "Label",
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
									Line:   45,
									Column: 50,
								},
								NameLocation: ast_domain.Location{
									Line:   45,
									Column: 42,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:8:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_req_method_post_id_r1_88512b59",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   47,
									Column: 9,
								},
								RequestOverrides: map[string]ast_domain.PropValue{
									"method": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "POST",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   47,
											Column: 59,
										},
										GoFieldName: "",
									},
								},
								PassedProps: map[string]ast_domain.PropValue{
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "r1",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   47,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:9",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "r1",
								Location: ast_domain.Location{
									Line:   47,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   47,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:9:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 5,
						},
						TagName: "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_badge_63370d86"),
							OriginalSourcePath:   new("partials/badge.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "badge_id_x1_ef36f947",
								PartialAlias:        "badge",
								PartialPackageName:  "partials_badge_63370d86",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   49,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"id": ast_domain.PropValue{
										Expression: &ast_domain.StringLiteral{
											Value: "x1",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   49,
											Column: 38,
										},
										GoFieldName: "",
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:10",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "mt-4",
								Location: ast_domain.Location{
									Line:   49,
									Column: 49,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 42,
								},
							},
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "x1",
								Location: ast_domain.Location{
									Line:   49,
									Column: 38,
								},
								NameLocation: ast_domain.Location{
									Line:   49,
									Column: 34,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   22,
									Column: 11,
								},
								TextContent: "Badge",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:10:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 11,
									},
									GoAnnotations: nil,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   50,
							Column: 9,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						DirIf: &ast_domain.Directive{
							Type: ast_domain.DirectiveIf,
							Location: ast_domain.Location{
								Line:   50,
								Column: 20,
							},
							NameLocation: ast_domain.Location{
								Line:   50,
								Column: 14,
							},
							RawExpression: "true",
							Expression: &ast_domain.BooleanLiteral{
								Value: true,
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
								GoAnnotations: nil,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								OriginalSourcePath: new("main.pk"),
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:11",
							RelativeLocation: ast_domain.Location{
								Line:   50,
								Column: 9,
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
								TagName: "span",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_badge_63370d86"),
									OriginalSourcePath:   new("partials/badge.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "badge_id_x2_20c7e59d",
										PartialAlias:        "badge",
										PartialPackageName:  "partials_badge_63370d86",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   51,
											Column: 13,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"id": ast_domain.PropValue{
												Expression: &ast_domain.StringLiteral{
													Value: "x2",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   51,
													Column: 42,
												},
												GoFieldName: "",
											},
										},
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:11:0",
									RelativeLocation: ast_domain.Location{
										Line:   22,
										Column: 5,
									},
									GoAnnotations: nil,
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "x2",
										Location: ast_domain.Location{
											Line:   51,
											Column: 42,
										},
										NameLocation: ast_domain.Location{
											Line:   51,
											Column: 38,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   22,
											Column: 11,
										},
										TextContent: "Badge",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_badge_63370d86"),
											OriginalSourcePath:   new("partials/badge.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:11:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   22,
												Column: 11,
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
