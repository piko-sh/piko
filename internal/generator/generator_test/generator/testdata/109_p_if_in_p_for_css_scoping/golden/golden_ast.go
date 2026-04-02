package default_test_pkg

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
					OriginalPackageAlias: new("pages_main_594861c5"),
					OriginalSourcePath:   new("pages/main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   22,
						Column: 3,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("pages/main.pk"),
						Stringability:      1,
					},
				},
				Children: []*ast_domain.TemplateNode{
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   23,
							Column: 5,
						},
						TagName: "h1",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStatic:             true,
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
							RelativeLocation: ast_domain.Location{
								Line:   23,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   23,
									Column: 9,
								},
								TextContent: "P-If in P-For CSS Scoping Test",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   22,
							Column: 3,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_task_list_07441b8b"),
							OriginalSourcePath:   new("partials/task_list.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "task_list_tasks_state_tasks_a948e044",
								PartialAlias:        "task_list",
								PartialPackageName:  "partials_task_list_07441b8b",
								InvokerPackageAlias: "pages_main_594861c5",
								Location: ast_domain.Location{
									Line:   24,
									Column: 5,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"tasks": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("pages_main_594861c5.PageData"),
														PackageAlias:         "pages_main_594861c5",
														CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/dist/pages/pages_main_594861c5",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 49,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Tasks",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.Task"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tasks",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 49,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 23,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]dto.Task"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Tasks",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 49,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 23,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.Task"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Tasks",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 49,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 23,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.Task"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tasks",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 49,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 23,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
											},
										},
										Location: ast_domain.Location{
											Line:   24,
											Column: 49,
										},
										GoFieldName: "Tasks",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.Task"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Tasks",
												ReferenceLocation: ast_domain.Location{
													Line:   24,
													Column: 49,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   38,
													Column: 23,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.Task"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Tasks",
													ReferenceLocation: ast_domain.Location{
														Line:   24,
														Column: 49,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   38,
														Column: 23,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
										},
									},
								},
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 3,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/task_list.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "class",
								Value: "task-container",
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
								TagName: "h2",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_task_list_07441b8b"),
									OriginalSourcePath:   new("partials/task_list.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/task_list.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "task-header",
										Location: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 29,
										},
										TextContent: "Task List",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_task_list_07441b8b"),
											OriginalSourcePath:   new("partials/task_list.pk"),
											IsStatic:             true,
											IsStructurallyStatic: true,
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:1:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 29,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/task_list.pk"),
												Stringability:      1,
											},
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
								TagName: "ul",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_task_list_07441b8b"),
									OriginalSourcePath:   new("partials/task_list.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:1",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 5,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/task_list.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "class",
										Value: "task-list",
										Location: ast_domain.Location{
											Line:   24,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 9,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeElement,
										Location: ast_domain.Location{
											Line:   25,
											Column: 7,
										},
										TagName: "li",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_task_list_07441b8b"),
											OriginalSourcePath:   new("partials/task_list.pk"),
										},
										DirFor: &ast_domain.Directive{
											Type: ast_domain.DirectiveFor,
											Location: ast_domain.Location{
												Line:   25,
												Column: 18,
											},
											NameLocation: ast_domain.Location{
												Line:   25,
												Column: 11,
											},
											RawExpression: "(idx, task) in props.Tasks",
											Expression: &ast_domain.ForInExpression{
												IndexVariable: &ast_domain.Identifier{
													Name: "idx",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 2,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "idx",
															ReferenceLocation: ast_domain.Location{
																Line:   28,
																Column: 43,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("idx"),
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
												},
												ItemVariable: &ast_domain.Identifier{
													Name: "task",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("dto.Task"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "task",
															ReferenceLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("task"),
														OriginalSourcePath: new("partials/task_list.pk"),
													},
												},
												Collection: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 16,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_task_list_07441b8b.Props"),
																PackageAlias:         "partials_task_list_07441b8b",
																CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/dist/partials/partials_task_list_07441b8b",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 18,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_task_list_tasks_state_tasks_a948e044"),
															OriginalSourcePath: new("partials/task_list.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Tasks",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 22,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]dto.Task"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Tasks",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 18,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   68,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_task_list_tasks_state_tasks_a948e044"),
															OriginalSourcePath:  new("partials/task_list.pk"),
															GeneratedSourcePath: new("dist/partials/partials_task_list_07441b8b/generated.go"),
														},
													},
													Optional: false,
													Computed: false,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 16,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("[]dto.Task"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Tasks",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 18,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   68,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_task_list_tasks_state_tasks_a948e044"),
														OriginalSourcePath:  new("partials/task_list.pk"),
														GeneratedSourcePath: new("dist/partials/partials_task_list_07441b8b/generated.go"),
													},
												},
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.Task"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tasks",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   68,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_task_list_tasks_state_tasks_a948e044"),
													OriginalSourcePath:  new("partials/task_list.pk"),
													GeneratedSourcePath: new("dist/partials/partials_task_list_07441b8b/generated.go"),
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.Task"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Tasks",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   68,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_task_list_tasks_state_tasks_a948e044"),
													OriginalSourcePath:  new("partials/task_list.pk"),
													GeneratedSourcePath: new("dist/partials/partials_task_list_07441b8b/generated.go"),
												},
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.Task"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Tasks",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   68,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("props_task_list_tasks_state_tasks_a948e044"),
												OriginalSourcePath:  new("partials/task_list.pk"),
												GeneratedSourcePath: new("dist/partials/partials_task_list_07441b8b/generated.go"),
											},
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
													Literal: "r.0:1:1:0.",
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: false,
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
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.Identifier{
														Name: "idx",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 2,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("int"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "idx",
																ReferenceLocation: ast_domain.Location{
																	Line:   28,
																	Column: 43,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("idx"),
															OriginalSourcePath: new("partials/task_list.pk"),
															Stringability:      1,
														},
													},
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 7,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/task_list.pk"),
												Stringability:      1,
											},
										},
										Attributes: []ast_domain.HTMLAttribute{
											ast_domain.HTMLAttribute{
												Name:  "class",
												Value: "task-item",
												Location: ast_domain.Location{
													Line:   25,
													Column: 66,
												},
												NameLocation: ast_domain.Location{
													Line:   25,
													Column: 59,
												},
											},
										},
										DynamicAttributes: []ast_domain.DynamicAttribute{
											ast_domain.DynamicAttribute{
												Name:          "p-key",
												RawExpression: "idx",
												Expression: &ast_domain.Identifier{
													Name: "idx",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "idx",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("idx"),
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
												},
												Location: ast_domain.Location{
													Line:   25,
													Column: 54,
												},
												NameLocation: ast_domain.Location{
													Line:   25,
													Column: 46,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "idx",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("idx"),
													OriginalSourcePath: new("partials/task_list.pk"),
													Stringability:      1,
												},
											},
										},
										Children: []*ast_domain.TemplateNode{
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   26,
													Column: 9,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_task_list_07441b8b"),
													OriginalSourcePath:   new("partials/task_list.pk"),
												},
												DirText: &ast_domain.Directive{
													Type: ast_domain.DirectiveText,
													Location: ast_domain.Location{
														Line:   26,
														Column: 41,
													},
													NameLocation: ast_domain.Location{
														Line:   26,
														Column: 33,
													},
													RawExpression: "task.Name",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "task",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.Task"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "task",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("task"),
																OriginalSourcePath: new("partials/task_list.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Name",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   26,
																		Column: 41,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   80,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("task"),
																OriginalSourcePath:  new("partials/task_list.pk"),
																GeneratedSourcePath: new("pkg/dto/task.go"),
																Stringability:       1,
															},
														},
														Optional: false,
														Computed: false,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   26,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   80,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("task"),
															OriginalSourcePath:  new("partials/task_list.pk"),
															GeneratedSourcePath: new("pkg/dto/task.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   80,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("task"),
														OriginalSourcePath:  new("partials/task_list.pk"),
														GeneratedSourcePath: new("pkg/dto/task.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "idx",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 2,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "idx",
																		ReferenceLocation: ast_domain.Location{
																			Line:   28,
																			Column: 43,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("idx"),
																	OriginalSourcePath: new("partials/task_list.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   26,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Literal: ":0",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   26,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "task-name",
														Location: ast_domain.Location{
															Line:   26,
															Column: 22,
														},
														NameLocation: ast_domain.Location{
															Line:   26,
															Column: 15,
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   27,
													Column: 9,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_task_list_07441b8b"),
													OriginalSourcePath:   new("partials/task_list.pk"),
												},
												DirIf: &ast_domain.Directive{
													Type: ast_domain.DirectiveIf,
													Location: ast_domain.Location{
														Line:   27,
														Column: 21,
													},
													NameLocation: ast_domain.Location{
														Line:   27,
														Column: 15,
													},
													RawExpression: "task.Done",
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "task",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("dto.Task"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "task",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("task"),
																OriginalSourcePath: new("partials/task_list.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Done",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 6,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("bool"),
																	PackageAlias:         "dto",
																	CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Done",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 21,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   81,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("task"),
																OriginalSourcePath:  new("partials/task_list.pk"),
																GeneratedSourcePath: new("pkg/dto/task.go"),
																Stringability:       1,
															},
														},
														Optional: false,
														Computed: false,
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("bool"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Done",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 21,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   81,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("task"),
															OriginalSourcePath:  new("partials/task_list.pk"),
															GeneratedSourcePath: new("pkg/dto/task.go"),
															Stringability:       1,
														},
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("bool"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_109_p_if_in_p_for_css_scoping/pkg/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Done",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 21,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   81,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("task"),
														OriginalSourcePath:  new("partials/task_list.pk"),
														GeneratedSourcePath: new("pkg/dto/task.go"),
														Stringability:       1,
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "idx",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 2,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "idx",
																		ReferenceLocation: ast_domain.Location{
																			Line:   28,
																			Column: 43,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("idx"),
																	OriginalSourcePath: new("partials/task_list.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Literal: ":1",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   27,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "task-done",
														Location: ast_domain.Location{
															Line:   27,
															Column: 39,
														},
														NameLocation: ast_domain.Location{
															Line:   27,
															Column: 32,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   27,
															Column: 50,
														},
														TextContent: "Done",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_task_list_07441b8b"),
															OriginalSourcePath:   new("partials/task_list.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 50,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:1:0.",
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: false,
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
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "idx",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 2,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "",
																				CanonicalPackagePath: "",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "idx",
																				ReferenceLocation: ast_domain.Location{
																					Line:   28,
																					Column: 43,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("idx"),
																			OriginalSourcePath: new("partials/task_list.pk"),
																			Stringability:      1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   27,
																		Column: 50,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																	Literal: ":1:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   27,
																Column: 50,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
														},
													},
												},
											},
											&ast_domain.TemplateNode{
												NodeType: ast_domain.NodeElement,
												Location: ast_domain.Location{
													Line:   28,
													Column: 9,
												},
												TagName: "span",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalPackageAlias: new("partials_task_list_07441b8b"),
													OriginalSourcePath:   new("partials/task_list.pk"),
												},
												DirElse: &ast_domain.Directive{
													Type: ast_domain.DirectiveElse,
													Location: ast_domain.Location{
														Line:   28,
														Column: 15,
													},
													NameLocation: ast_domain.Location{
														Line:   28,
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														OriginalSourcePath: new("partials/task_list.pk"),
													},
													ChainKey: &ast_domain.TemplateLiteral{
														Parts: []ast_domain.TemplateLiteralPart{
															ast_domain.TemplateLiteralPart{
																IsLiteral: true,
																RelativeLocation: ast_domain.Location{
																	Line:   27,
																	Column: 9,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	OriginalSourcePath: new("partials/task_list.pk"),
																	Stringability:      1,
																},
																Literal: "r.0:1:1:0.",
															},
															ast_domain.TemplateLiteralPart{
																IsLiteral: false,
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
																	OriginalSourcePath: new("partials/task_list.pk"),
																	Stringability:      1,
																},
																Expression: &ast_domain.Identifier{
																	Name: "idx",
																	RelativeLocation: ast_domain.Location{
																		Line:   1,
																		Column: 2,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("int"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		Symbol: &ast_domain.ResolvedSymbol{
																			Name: "idx",
																			ReferenceLocation: ast_domain.Location{
																				Line:   28,
																				Column: 43,
																			},
																			DeclarationLocation: ast_domain.Location{
																				Line:   0,
																				Column: 0,
																			},
																		},
																		BaseCodeGenVarName: new("idx"),
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																},
															},
															ast_domain.TemplateLiteralPart{
																IsLiteral: true,
																RelativeLocation: ast_domain.Location{
																	Line:   27,
																	Column: 9,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("string"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	OriginalSourcePath: new("partials/task_list.pk"),
																	Stringability:      1,
																},
																Literal: ":1",
															},
														},
														RelativeLocation: ast_domain.Location{
															Line:   27,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
															OriginalSourcePath: new("partials/task_list.pk"),
															Stringability:      1,
														},
													},
												},
												Key: &ast_domain.TemplateLiteral{
													Parts: []ast_domain.TemplateLiteralPart{
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Literal: "r.0:1:1:0.",
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: false,
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
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Expression: &ast_domain.Identifier{
																Name: "idx",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 2,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("int"),
																		PackageAlias:         "",
																		CanonicalPackagePath: "",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "idx",
																		ReferenceLocation: ast_domain.Location{
																			Line:   28,
																			Column: 43,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("idx"),
																	OriginalSourcePath: new("partials/task_list.pk"),
																	Stringability:      1,
																},
															},
														},
														ast_domain.TemplateLiteralPart{
															IsLiteral: true,
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 9,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
															},
															Literal: ":2",
														},
													},
													RelativeLocation: ast_domain.Location{
														Line:   28,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("partials/task_list.pk"),
														Stringability:      1,
													},
												},
												Attributes: []ast_domain.HTMLAttribute{
													ast_domain.HTMLAttribute{
														Name:  "class",
														Value: "task-pending",
														Location: ast_domain.Location{
															Line:   28,
															Column: 29,
														},
														NameLocation: ast_domain.Location{
															Line:   28,
															Column: 22,
														},
													},
												},
												Children: []*ast_domain.TemplateNode{
													&ast_domain.TemplateNode{
														NodeType: ast_domain.NodeText,
														Location: ast_domain.Location{
															Line:   28,
															Column: 43,
														},
														TextContent: "Pending",
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															OriginalPackageAlias: new("partials_task_list_07441b8b"),
															OriginalSourcePath:   new("partials/task_list.pk"),
														},
														Key: &ast_domain.TemplateLiteral{
															Parts: []ast_domain.TemplateLiteralPart{
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   28,
																		Column: 43,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																	Literal: "r.0:1:1:0.",
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: false,
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
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																	Expression: &ast_domain.Identifier{
																		Name: "idx",
																		RelativeLocation: ast_domain.Location{
																			Line:   1,
																			Column: 2,
																		},
																		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																			ResolvedType: &ast_domain.ResolvedTypeInfo{
																				TypeExpression:       typeExprFromString("int"),
																				PackageAlias:         "",
																				CanonicalPackagePath: "",
																			},
																			Symbol: &ast_domain.ResolvedSymbol{
																				Name: "idx",
																				ReferenceLocation: ast_domain.Location{
																					Line:   28,
																					Column: 43,
																				},
																				DeclarationLocation: ast_domain.Location{
																					Line:   0,
																					Column: 0,
																				},
																			},
																			BaseCodeGenVarName: new("idx"),
																			OriginalSourcePath: new("partials/task_list.pk"),
																			Stringability:      1,
																		},
																	},
																},
																ast_domain.TemplateLiteralPart{
																	IsLiteral: true,
																	RelativeLocation: ast_domain.Location{
																		Line:   28,
																		Column: 43,
																	},
																	GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																		ResolvedType: &ast_domain.ResolvedTypeInfo{
																			TypeExpression:       typeExprFromString("string"),
																			PackageAlias:         "",
																			CanonicalPackagePath: "",
																		},
																		OriginalSourcePath: new("partials/task_list.pk"),
																		Stringability:      1,
																	},
																	Literal: ":2:0",
																},
															},
															RelativeLocation: ast_domain.Location{
																Line:   28,
																Column: 43,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
																OriginalSourcePath: new("partials/task_list.pk"),
																Stringability:      1,
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
