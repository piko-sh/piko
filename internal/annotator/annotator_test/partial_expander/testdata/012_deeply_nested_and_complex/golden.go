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
				TagName: "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalPackageAlias: new("partials_page_layout_d04fd246"),
					OriginalSourcePath:   new("partials/page_layout.pk"),
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey:       "layout_data_logged_in_state_isloggedin_3d8e08d3",
						PartialAlias:        "layout",
						PartialPackageName:  "partials_page_layout_d04fd246",
						InvokerPackageAlias: "main_aaf9a2e0",
						Location: ast_domain.Location{
							Line:   39,
							Column: 5,
						},
						PassedProps: map[string]ast_domain.PropValue{
							"data-logged-in": ast_domain.PropValue{
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
									},
									Property: &ast_domain.Identifier{
										Name: "IsLoggedIn",
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
									Line:   39,
									Column: 67,
								},
								GoFieldName: "",
							},
						},
					},
					DynamicAttributeOrigins: map[string]string{
						"data-logged-in": "main_aaf9a2e0",
					},
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   28,
						Column: 5,
					},
					GoAnnotations: nil,
				},
				Attributes: []ast_domain.HTMLAttribute{
					ast_domain.HTMLAttribute{
						Name:  "class",
						Value: "page-layout theme-dark",
						Location: ast_domain.Location{
							Line:   28,
							Column: 17,
						},
						NameLocation: ast_domain.Location{
							Line:   28,
							Column: 10,
						},
					},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					ast_domain.DynamicAttribute{
						Name:          "data-logged-in",
						RawExpression: "state.IsLoggedIn",
						Expression: &ast_domain.MemberExpression{
							Base: &ast_domain.Identifier{
								Name: "state",
								RelativeLocation: ast_domain.Location{
									Line:   1,
									Column: 1,
								},
							},
							Property: &ast_domain.Identifier{
								Name: "IsLoggedIn",
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
							Line:   39,
							Column: 67,
						},
						NameLocation: ast_domain.Location{
							Line:   39,
							Column: 50,
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
							Column: 9,
						},
						TagName: "main",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_page_layout_d04fd246"),
							OriginalSourcePath:   new("partials/page_layout.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   30,
								Column: 9,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "page-layout-main",
								Location: ast_domain.Location{
									Line:   30,
									Column: 22,
								},
								NameLocation: ast_domain.Location{
									Line:   30,
									Column: 15,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   42,
									Column: 13,
								},
								TagName: "h1",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   42,
										Column: 13,
									},
									GoAnnotations: nil,
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   42,
											Column: 17,
										},
										TextContent: "Welcome to the Dashboard",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   42,
												Column: 17,
											},
											GoAnnotations: nil,
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   43,
									Column: 13,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:1",
									RelativeLocation: ast_domain.Location{
										Line:   43,
										Column: 13,
									},
									GoAnnotations: nil,
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   43,
											Column: 16,
										},
										TextContent: "This is the main content area provided by main.pk.",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:0:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   43,
												Column: 16,
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
							Line:   36,
							Column: 9,
						},
						TagName: "aside",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_page_layout_d04fd246"),
							OriginalSourcePath:   new("partials/page_layout.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   36,
								Column: 9,
							},
							GoAnnotations: nil,
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "page-layout-sidebar",
								Location: ast_domain.Location{
									Line:   36,
									Column: 23,
								},
								NameLocation: ast_domain.Location{
									Line:   36,
									Column: 16,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   37,
									Column: 5,
								},
								TagName: "div",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_sidebar_959a74bf"),
									OriginalSourcePath:   new("partials/sidebar.pk"),
									PartialInfo: &ast_domain.PartialInvocationInfo{
										InvocationKey:       "sidebar_is_collapsible_true_bc6d3300",
										PartialAlias:        "sidebar",
										PartialPackageName:  "partials_sidebar_959a74bf",
										InvokerPackageAlias: "main_aaf9a2e0",
										Location: ast_domain.Location{
											Line:   47,
											Column: 13,
										},
										PassedProps: map[string]ast_domain.PropValue{
											"is-collapsible": ast_domain.PropValue{
												Expression: &ast_domain.BooleanLiteral{
													Value: true,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: nil,
												},
												Location: ast_domain.Location{
													Line:   47,
													Column: 57,
												},
												GoFieldName: "",
											},
										},
									},
									DynamicAttributeOrigins: map[string]string{
										"is-collapsible": "main_aaf9a2e0",
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   37,
										Column: 5,
									},
									GoAnnotations: nil,
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "sidebar",
										Location: ast_domain.Location{
											Line:   37,
											Column: 17,
										},
										NameLocation: ast_domain.Location{
											Line:   37,
											Column: 10,
										},
									},
								},
								DynamicAttributes: []ast_domain.DynamicAttribute{
									ast_domain.DynamicAttribute{
										Name:          "is-collapsible",
										RawExpression: "true",
										Expression: &ast_domain.BooleanLiteral{
											Value: true,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: nil,
										},
										Location: ast_domain.Location{
											Line:   47,
											Column: 57,
										},
										NameLocation: ast_domain.Location{
											Line:   47,
											Column: 40,
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
											Line:   38,
											Column: 9,
										},
										TagName: "header",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_sidebar_959a74bf"),
											OriginalSourcePath:   new("partials/sidebar.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   38,
												Column: 9,
											},
											GoAnnotations: nil,
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   50,
													Column: 21,
												},
												TagName: "h2",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:0:0",
													RelativeLocation: ast_domain.Location{
														Line:   50,
														Column: 21,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   50,
															Column: 25,
														},
														TextContent: "Main Navigation",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("main_aaf9a2e0"),
															OriginalSourcePath:   new("main.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:0:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   50,
																Column: 25,
															},
															GoAnnotations: nil,
														},
													},
												},
											},
										},
									},
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeFragment,
										Location: ast_domain.Location{
											Line:   0,
											Column: 0,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_sidebar_959a74bf"),
											OriginalSourcePath:   new("partials/sidebar.pk"),
											PartialInfo: &ast_domain.PartialInvocationInfo{
												InvocationKey:       "profile_notification_count_state_notifications_d2142e00",
												PartialAlias:        "profile",
												PartialPackageName:  "partials_user_profile_c73fd2a9",
												InvokerPackageAlias: "partials_sidebar_959a74bf",
												Location: ast_domain.Location{
													Line:   44,
													Column: 9,
												},
												PassedProps: map[string]ast_domain.PropValue{
													"notification-count": ast_domain.PropValue{
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Notifications",
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
															Line:   44,
															Column: 81,
														},
														GoFieldName: "",
													},
												},
											},
											DynamicAttributeOrigins: map[string]string{
												"notification-count": "partials_sidebar_959a74bf",
											},
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:1",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: nil,
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "profile-section",
												Location: ast_domain.Location{
													Line:   44,
													Column: 30,
												},
												NameLocation: ast_domain.Location{
													Line:   44,
													Column: 23,
												},
											},
										},
										DynamicAttributes: []ast_domain.DynamicAttribute{
											ast_domain.DynamicAttribute{
												Name:          "notification-count",
												RawExpression: "state.Notifications",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Notifications",
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
													Line:   44,
													Column: 81,
												},
												NameLocation: ast_domain.Location{
													Line:   44,
													Column: 60,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_sidebar_959a74bf"),
													OriginalSourcePath:   new("partials/sidebar.pk"),
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   35,
													Column: 5,
												},
												TagName: "div",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_avatar_c3a790d9"),
													OriginalSourcePath:   new("partials/avatar.pk"),
													PartialInfo: &ast_domain.PartialInvocationInfo{
														InvocationKey:       "avatar_initial_state_userinitial_0068e298",
														PartialAlias:        "avatar",
														PartialPackageName:  "partials_avatar_c3a790d9",
														InvokerPackageAlias: "partials_user_profile_c73fd2a9",
														Location: ast_domain.Location{
															Line:   38,
															Column: 5,
														},
														PassedProps: map[string]ast_domain.PropValue{
															"initial": ast_domain.PropValue{
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "UserInitial",
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
																	Line:   38,
																	Column: 41,
																},
																GoFieldName: "",
															},
														},
													},
													DynamicAttributeOrigins: map[string]string{
														"initial": "partials_user_profile_c73fd2a9",
														"title":   "partials_avatar_c3a790d9",
													},
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:0",
													RelativeLocation: ast_domain.Location{
														Line:   35,
														Column: 5,
													},
													GoAnnotations: nil,
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "avatar",
														Location: ast_domain.Location{
															Line:   35,
															Column: 17,
														},
														NameLocation: ast_domain.Location{
															Line:   35,
															Column: 10,
														},
													},
													ast_domain.HTMLAttribute{
														Name:  "p-fragment",
														Value: "profile_notification_count_state_notifications_d2142e00",
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
												DynamicAttributes: []ast_domain.DynamicAttribute{
													ast_domain.DynamicAttribute{
														Name:          "initial",
														RawExpression: "state.UserInitial",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
															},
															Property: &ast_domain.Identifier{
																Name: "UserInitial",
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
															Line:   38,
															Column: 41,
														},
														NameLocation: ast_domain.Location{
															Line:   38,
															Column: 31,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_user_profile_c73fd2a9"),
															OriginalSourcePath:   new("partials/user_profile.pk"),
														},
													},
													ast_domain.DynamicAttribute{
														Name:          "title",
														RawExpression: "props.initial",
														Expression: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "props",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
															},
															Property: &ast_domain.Identifier{
																Name: "initial",
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
															Column: 33,
														},
														NameLocation: ast_domain.Location{
															Line:   35,
															Column: 25,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalSourcePath: new("partials/avatar.pk"),
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   35,
															Column: 48,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_avatar_c3a790d9"),
															OriginalSourcePath:   new("partials/avatar.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   35,
																Column: 48,
															},
															GoAnnotations: nil,
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   35,
																	Column: 48,
																},
																Literal: "\n        ",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/avatar.pk"),
																},
															},
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   36,
																	Column: 12,
																},
																RawExpression: "props.initial",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "props",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "initial",
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
																	OriginalSourcePath: new("partials/avatar.pk"),
																},
															},
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   36,
																	Column: 28,
																},
																Literal: "\n    ",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("partials/avatar.pk"),
																},
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   40,
													Column: 5,
												},
												TagName: "div",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_user_profile_c73fd2a9"),
													OriginalSourcePath:   new("partials/user_profile.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:1:1",
													RelativeLocation: ast_domain.Location{
														Line:   40,
														Column: 5,
													},
													GoAnnotations: nil,
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "user-profile-info",
														Location: ast_domain.Location{
															Line:   40,
															Column: 17,
														},
														NameLocation: ast_domain.Location{
															Line:   40,
															Column: 10,
														},
													},
													ast_domain.HTMLAttribute{
														Name:  "p-fragment",
														Value: "profile_notification_count_state_notifications_d2142e00",
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
															Line:   41,
															Column: 9,
														},
														TagName: "p",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_user_profile_c73fd2a9"),
															OriginalSourcePath:   new("partials/user_profile.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:1:0",
															RelativeLocation: ast_domain.Location{
																Line:   41,
																Column: 9,
															},
															GoAnnotations: nil,
														},
														Children: []*ast_domain.TemplateNode{
															&ast_domain.TemplateNode{
																NodeType: ast_domain.NodeText,
																Location: ast_domain.Location{
																	Line:   41,
																	Column: 12,
																},
																TextContent: "User Info",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_user_profile_c73fd2a9"),
																	OriginalSourcePath:   new("partials/user_profile.pk"),
																},
																Key: &ast_domain.StringLiteral{
																	Value: "r.0:1:0:1:1:0:0",
																	RelativeLocation: ast_domain.Location{
																		Line:   41,
																		Column: 12,
																	},
																	GoAnnotations: nil,
																},
															},
														},
													},
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeElement,
														Location: ast_domain.Location{
															Line:   42,
															Column: 9,
														},
														TagName: "span",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_user_profile_c73fd2a9"),
															OriginalSourcePath:   new("partials/user_profile.pk"),
														},
														DirIf: &ast_domain.Directive{
															Type: ast_domain.DirectiveIf,
															Location: ast_domain.Location{
																Line:   42,
																Column: 56,
															},
															NameLocation: ast_domain.Location{
																Line:   42,
																Column: 50,
															},
															RawExpression: "props.notificationCount > 0",
															Expression: &ast_domain.BinaryExpression{
																Left: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "props",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "notificationCount",
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
																Operator: ">",
																Right: &ast_domain.IntegerLiteral{
																	Value: 0,
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 27,
																	},
																	GoAnnotations: nil,
																},
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																OriginalSourcePath: new("partials/user_profile.pk"),
															},
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:1:1:1",
															RelativeLocation: ast_domain.Location{
																Line:   42,
																Column: 9,
															},
															GoAnnotations: nil,
														},
														Attributes: []ast_domain.HTMLAttribute{
															ast_domain.HTMLAttribute{
																Name:  "class",
																Value: "user-profile-notifications",
																Location: ast_domain.Location{
																	Line:   42,
																	Column: 22,
																},
																NameLocation: ast_domain.Location{
																	Line:   42,
																	Column: 15,
																},
															},
														},
														Children: []*ast_domain.TemplateNode{
															&ast_domain.TemplateNode{
																NodeType: ast_domain.NodeText,
																Location: ast_domain.Location{
																	Line:   42,
																	Column: 85,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalPackageAlias: new("partials_user_profile_c73fd2a9"),
																	OriginalSourcePath:   new("partials/user_profile.pk"),
																},
																Key: &ast_domain.StringLiteral{
																	Value: "r.0:1:0:1:1:1:0",
																	RelativeLocation: ast_domain.Location{
																		Line:   42,
																		Column: 85,
																	},
																	GoAnnotations: nil,
																},
																RichText: []ast_domain.TextPart{
																	ast_domain.TextPart{
																		IsLiteral: true,
																		Location: ast_domain.Location{
																			Line:   42,
																			Column: 85,
																		},
																		Literal: "\n            ",
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			OriginalSourcePath: new("partials/user_profile.pk"),
																		},
																	},
																	ast_domain.TextPart{
																		IsLiteral: false,
																		Location: ast_domain.Location{
																			Line:   43,
																			Column: 16,
																		},
																		RawExpression: "props.notificationCount",
																		Expression: &ast_domain.MemberExpression{
																			Base: &ast_domain.Identifier{
																				Name: "props",
																				RelativeLocation: ast_domain.Location{
																					Line:   1,
																					Column: 1,
																				},
																			},
																			Property: &ast_domain.Identifier{
																				Name: "notificationCount",
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
																			OriginalSourcePath: new("partials/user_profile.pk"),
																		},
																	},
																	ast_domain.TextPart{
																		IsLiteral: true,
																		Location: ast_domain.Location{
																			Line:   43,
																			Column: 42,
																		},
																		Literal: " New\n        ",
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			OriginalSourcePath: new("partials/user_profile.pk"),
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
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   46,
											Column: 9,
										},
										TagName: "nav",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_sidebar_959a74bf"),
											OriginalSourcePath:   new("partials/sidebar.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:2",
											RelativeLocation: ast_domain.Location{
												Line:   46,
												Column: 9,
											},
											GoAnnotations: nil,
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   53,
													Column: 17,
												},
												TagName: "p",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("main_aaf9a2e0"),
													OriginalSourcePath:   new("main.pk"),
												},
												Key: &ast_domain.StringLiteral{
													Value: "r.0:1:0:2:0",
													RelativeLocation: ast_domain.Location{
														Line:   53,
														Column: 17,
													},
													GoAnnotations: nil,
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   53,
															Column: 20,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("main_aaf9a2e0"),
															OriginalSourcePath:   new("main.pk"),
														},
														Key: &ast_domain.StringLiteral{
															Value: "r.0:1:0:2:0:0",
															RelativeLocation: ast_domain.Location{
																Line:   53,
																Column: 20,
															},
															GoAnnotations: nil,
														},
														RichText: []ast_domain.TextPart{
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   53,
																	Column: 20,
																},
																Literal: "Hello, ",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("main.pk"),
																},
															},
															ast_domain.TextPart{
																IsLiteral: false,
																Location: ast_domain.Location{
																	Line:   53,
																	Column: 30,
																},
																RawExpression: "state.Username",
																Expression: &ast_domain.MemberExpression{
																	Base: &ast_domain.Identifier{
																		Name: "state",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 1,
																		},
																	},
																	Property: &ast_domain.Identifier{
																		Name: "Username",
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
															ast_domain.TextPart{
																IsLiteral: true,
																Location: ast_domain.Location{
																	Line:   53,
																	Column: 47,
																},
																Literal: "!",
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	OriginalSourcePath: new("main.pk"),
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
			},
		},
	}
}()
