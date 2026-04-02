package test

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
					OriginalPackageAlias: new("main_aaf9a2e0"),
					OriginalSourcePath:   new("main.pk"),
					IsStructurallyStatic: true,
				},
				Key: &ast_domain.StringLiteral{
					Value: "r.0",
					RelativeLocation: ast_domain.Location{
						Line:   22,
						Column: 5,
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression:       typeExprFromString("string"),
							PackageAlias:         "",
							CanonicalPackagePath: "",
						},
						OriginalSourcePath: new("main.pk"),
						Stringability:      1,
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
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:0",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "direct-implicit",
								Location: ast_domain.Location{
									Line:   23,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   23,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   23,
									Column: 33,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   23,
										Column: 33,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   23,
											Column: 36,
										},
										RawExpression: "state.DirectUUID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "DirectUUID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DirectUUID",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   63,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DirectUUID",
													ReferenceLocation: ast_domain.Location{
														Line:   23,
														Column: 36,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   63,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       2,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DirectUUID",
												ReferenceLocation: ast_domain.Location{
													Line:   23,
													Column: 36,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       2,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   24,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   24,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "direct-explicit",
								Location: ast_domain.Location{
									Line:   24,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   24,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   24,
									Column: 33,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:1:0",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 33,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   24,
											Column: 36,
										},
										RawExpression: "state.DirectUUID.String()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "DirectUUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DirectUUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 36,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DirectUUID",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 36,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 18,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 36,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   291,
																Column: 1,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 36,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   291,
															Column: 1,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   26,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:2",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "model-implicit",
								Location: ast_domain.Location{
									Line:   26,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   26,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   26,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
									RelativeLocation: ast_domain.Location{
										Line:   26,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   26,
											Column: 35,
										},
										RawExpression: "state.ModelData.UserID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("pageData"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "ModelData",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.Data"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_38_third_party_types/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "ModelData",
															ReferenceLocation: ast_domain.Location{
																Line:   26,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   64,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
														TypeExpression:       typeExprFromString("models.Data"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_38_third_party_types/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "ModelData",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   64,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "UserID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserID",
														ReferenceLocation: ast_domain.Location{
															Line:   26,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/types.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserID",
													ReferenceLocation: ast_domain.Location{
														Line:   26,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/types.go"),
												Stringability:       2,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserID",
												ReferenceLocation: ast_domain.Location{
													Line:   26,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/types.go"),
											Stringability:       2,
										},
									},
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
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
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
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "model-explicit",
								Location: ast_domain.Location{
									Line:   27,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   27,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   27,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:3:0",
									RelativeLocation: ast_domain.Location{
										Line:   27,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   27,
											Column: 35,
										},
										RawExpression: "state.ModelData.UserID.String()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "state",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																	PackageAlias:         "main_aaf9a2e0",
																	CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 35,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
																OriginalSourcePath: new("main.pk"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "ModelData",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.Data"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_38_third_party_types/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "ModelData",
																	ReferenceLocation: ast_domain.Location{
																		Line:   27,
																		Column: 35,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   64,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
																TypeExpression:       typeExprFromString("models.Data"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_38_third_party_types/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "ModelData",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 35,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   64,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "UserID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 17,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UserID",
																ReferenceLocation: ast_domain.Location{
																	Line:   27,
																	Column: 35,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   71,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/types.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "UserID",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/types.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 24,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   27,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   291,
																Column: 1,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   27,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   291,
															Column: 1,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   29,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   29,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "aliased-implicit",
								Location: ast_domain.Location{
									Line:   29,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   29,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   29,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
									RelativeLocation: ast_domain.Location{
										Line:   29,
										Column: 34,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   29,
											Column: 37,
										},
										RawExpression: "state.AliasedUUID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "AliasedUUID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "AliasedUUID",
														ReferenceLocation: ast_domain.Location{
															Line:   29,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   67,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "AliasedUUID",
													ReferenceLocation: ast_domain.Location{
														Line:   29,
														Column: 37,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   67,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       2,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "AliasedUUID",
												ReferenceLocation: ast_domain.Location{
													Line:   29,
													Column: 37,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   67,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       2,
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
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   30,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "aliased-explicit",
								Location: ast_domain.Location{
									Line:   30,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   30,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   30,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   30,
										Column: 34,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   30,
											Column: 37,
										},
										RawExpression: "state.AliasedUUID.String()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "AliasedUUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "AliasedUUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   67,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "AliasedUUID",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   67,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 19,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   291,
																Column: 1,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   291,
															Column: 1,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   32,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
							RelativeLocation: ast_domain.Location{
								Line:   32,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "aliased-implicit",
								Location: ast_domain.Location{
									Line:   32,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   32,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   32,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
									RelativeLocation: ast_domain.Location{
										Line:   32,
										Column: 34,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   32,
											Column: 37,
										},
										RawExpression: "state.Deep.Data.UserID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Deep",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("deep.Deep"),
																PackageAlias:         "deep",
																CanonicalPackagePath: "testcase_38_third_party_types/deep",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Deep",
																ReferenceLocation: ast_domain.Location{
																	Line:   32,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   62,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
															TypeExpression:       typeExprFromString("deep.Deep"),
															PackageAlias:         "deep",
															CanonicalPackagePath: "testcase_38_third_party_types/deep",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Deep",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Data",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.Data"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_38_third_party_types/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   32,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("deep/deep.go"),
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
														TypeExpression:       typeExprFromString("models.Data"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_38_third_party_types/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("deep/deep.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "UserID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserID",
														ReferenceLocation: ast_domain.Location{
															Line:   32,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/types.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserID",
													ReferenceLocation: ast_domain.Location{
														Line:   32,
														Column: 37,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/types.go"),
												Stringability:       2,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserID",
												ReferenceLocation: ast_domain.Location{
													Line:   32,
													Column: 37,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/types.go"),
											Stringability:       2,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   33,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:7",
							RelativeLocation: ast_domain.Location{
								Line:   33,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "aliased-explicit",
								Location: ast_domain.Location{
									Line:   33,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   33,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   33,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
									RelativeLocation: ast_domain.Location{
										Line:   33,
										Column: 34,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   33,
											Column: 37,
										},
										RawExpression: "state.Deep.Data.UserID.String()",
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.MemberExpression{
														Base: &ast_domain.MemberExpression{
															Base: &ast_domain.Identifier{
																Name: "state",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 1,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																		PackageAlias:         "main_aaf9a2e0",
																		CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "state",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 37,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   0,
																			Column: 0,
																		},
																	},
																	BaseCodeGenVarName: new("pageData"),
																	OriginalSourcePath: new("main.pk"),
																},
															},
															Property: &ast_domain.Identifier{
																Name: "Deep",
																RelativeLocation: ast_domain.Location{
																	Line:   1,
																	Column: 7,
																},
																GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																	ResolvedType: &ast_domain.ResolvedTypeInfo{
																		TypeExpression:       typeExprFromString("deep.Deep"),
																		PackageAlias:         "deep",
																		CanonicalPackagePath: "testcase_38_third_party_types/deep",
																	},
																	Symbol: &ast_domain.ResolvedSymbol{
																		Name: "Deep",
																		ReferenceLocation: ast_domain.Location{
																			Line:   33,
																			Column: 37,
																		},
																		DeclarationLocation: ast_domain.Location{
																			Line:   62,
																			Column: 2,
																		},
																	},
																	BaseCodeGenVarName:  new("pageData"),
																	OriginalSourcePath:  new("main.pk"),
																	GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
																	TypeExpression:       typeExprFromString("deep.Deep"),
																	PackageAlias:         "deep",
																	CanonicalPackagePath: "testcase_38_third_party_types/deep",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Deep",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   62,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															},
														},
														Property: &ast_domain.Identifier{
															Name: "Data",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 12,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.Data"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_38_third_party_types/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Data",
																	ReferenceLocation: ast_domain.Location{
																		Line:   33,
																		Column: 37,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   71,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("pageData"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("deep/deep.go"),
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
																TypeExpression:       typeExprFromString("models.Data"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_38_third_party_types/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Data",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   71,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("deep/deep.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "UserID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 17,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UserID",
																ReferenceLocation: ast_domain.Location{
																	Line:   33,
																	Column: 37,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   71,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/types.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "UserID",
															ReferenceLocation: ast_domain.Location{
																Line:   33,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/types.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 24,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   33,
																Column: 37,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   291,
																Column: 1,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   33,
															Column: 37,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   291,
															Column: 1,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												BaseCodeGenVarName: new("pageData"),
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
											Stringability:      1,
										},
									},
								},
							},
						},
					},
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
						Key: &ast_domain.StringLiteral{
							Value: "r.0:8",
							RelativeLocation: ast_domain.Location{
								Line:   34,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "aliased-implicit",
								Location: ast_domain.Location{
									Line:   34,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   34,
									Column: 12,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "test",
								RawExpression: "state.Deep.Data.UserID",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   34,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Deep",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("deep.Deep"),
														PackageAlias:         "deep",
														CanonicalPackagePath: "testcase_38_third_party_types/deep",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Deep",
														ReferenceLocation: ast_domain.Location{
															Line:   34,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   62,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
													TypeExpression:       typeExprFromString("deep.Deep"),
													PackageAlias:         "deep",
													CanonicalPackagePath: "testcase_38_third_party_types/deep",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Deep",
													ReferenceLocation: ast_domain.Location{
														Line:   34,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   62,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											},
										},
										Property: &ast_domain.Identifier{
											Name: "Data",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 12,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.Data"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_38_third_party_types/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Data",
													ReferenceLocation: ast_domain.Location{
														Line:   34,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("deep/deep.go"),
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
												TypeExpression:       typeExprFromString("models.Data"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_38_third_party_types/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Data",
												ReferenceLocation: ast_domain.Location{
													Line:   34,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("deep/deep.go"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "UserID",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 17,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserID",
												ReferenceLocation: ast_domain.Location{
													Line:   34,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/types.go"),
											Stringability:       2,
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
											TypeExpression:       typeExprFromString("uuid.UUID"),
											PackageAlias:         "uuid",
											CanonicalPackagePath: "github.com/google/uuid",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "UserID",
											ReferenceLocation: ast_domain.Location{
												Line:   34,
												Column: 41,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   71,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/types.go"),
										Stringability:       2,
									},
								},
								Location: ast_domain.Location{
									Line:   34,
									Column: 41,
								},
								NameLocation: ast_domain.Location{
									Line:   34,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("uuid.UUID"),
										PackageAlias:         "uuid",
										CanonicalPackagePath: "github.com/google/uuid",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "UserID",
										ReferenceLocation: ast_domain.Location{
											Line:   34,
											Column: 41,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   71,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("models/types.go"),
									Stringability:       2,
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
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:9",
							RelativeLocation: ast_domain.Location{
								Line:   35,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "aliased-implicit",
								Location: ast_domain.Location{
									Line:   35,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   35,
									Column: 12,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "test",
								RawExpression: "state.Deep.Data.UserID.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Deep",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("deep.Deep"),
																PackageAlias:         "deep",
																CanonicalPackagePath: "testcase_38_third_party_types/deep",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Deep",
																ReferenceLocation: ast_domain.Location{
																	Line:   35,
																	Column: 41,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   62,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
															TypeExpression:       typeExprFromString("deep.Deep"),
															PackageAlias:         "deep",
															CanonicalPackagePath: "testcase_38_third_party_types/deep",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Deep",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   62,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "Data",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.Data"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_38_third_party_types/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Data",
															ReferenceLocation: ast_domain.Location{
																Line:   35,
																Column: 41,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("deep/deep.go"),
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
														TypeExpression:       typeExprFromString("models.Data"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_38_third_party_types/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Data",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("deep/deep.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "UserID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 17,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserID",
														ReferenceLocation: ast_domain.Location{
															Line:   35,
															Column: 41,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/types.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserID",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/types.go"),
												Stringability:       2,
											},
										},
										Property: &ast_domain.Identifier{
											Name: "String",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   35,
														Column: 41,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   291,
														Column: 1,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   35,
													Column: 41,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   291,
													Column: 1,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
										},
									},
									Args: []ast_domain.Expression{},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "uuid",
											CanonicalPackagePath: "github.com/google/uuid",
										},
										BaseCodeGenVarName: new("pageData"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   35,
									Column: 41,
								},
								NameLocation: ast_domain.Location{
									Line:   35,
									Column: 34,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "uuid",
										CanonicalPackagePath: "github.com/google/uuid",
									},
									BaseCodeGenVarName: new("pageData"),
									Stringability:      1,
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   37,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:10",
							RelativeLocation: ast_domain.Location{
								Line:   37,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "model-implicit",
								Location: ast_domain.Location{
									Line:   37,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   37,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   37,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:10:0",
									RelativeLocation: ast_domain.Location{
										Line:   37,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   37,
											Column: 35,
										},
										RawExpression: "state.MoreModelData.UserID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   37,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("pageData"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "MoreModelData",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.MoreData"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_38_third_party_types/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "MoreModelData",
															ReferenceLocation: ast_domain.Location{
																Line:   37,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   65,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
														TypeExpression:       typeExprFromString("models.MoreData"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_38_third_party_types/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "MoreModelData",
														ReferenceLocation: ast_domain.Location{
															Line:   37,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   65,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "UserID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 21,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "testcase_38_third_party_types/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserID",
														ReferenceLocation: ast_domain.Location{
															Line:   37,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/more_types.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "testcase_38_third_party_types/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserID",
													ReferenceLocation: ast_domain.Location{
														Line:   37,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/more_types.go"),
												Stringability:       2,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "testcase_38_third_party_types/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserID",
												ReferenceLocation: ast_domain.Location{
													Line:   37,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/more_types.go"),
											Stringability:       2,
										},
									},
								},
							},
						},
					},
					&ast_domain.TemplateNode{
						NodeType: ast_domain.NodeElement,
						Location: ast_domain.Location{
							Line:   38,
							Column: 9,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:11",
							RelativeLocation: ast_domain.Location{
								Line:   38,
								Column: 9,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("main.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "model-implicit",
								Location: ast_domain.Location{
									Line:   38,
									Column: 16,
								},
								NameLocation: ast_domain.Location{
									Line:   38,
									Column: 12,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   38,
									Column: 32,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:11:0",
									RelativeLocation: ast_domain.Location{
										Line:   38,
										Column: 32,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("main.pk"),
										Stringability:      1,
									},
								},
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   38,
											Column: 35,
										},
										RawExpression: "state.FinalModelData.UserID",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("pageData"),
														OriginalSourcePath: new("main.pk"),
													},
												},
												Property: &ast_domain.Identifier{
													Name: "FinalModelData",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.FinalData"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_38_third_party_types/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "FinalModelData",
															ReferenceLocation: ast_domain.Location{
																Line:   38,
																Column: 35,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   66,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
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
														TypeExpression:       typeExprFromString("models.FinalData"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_38_third_party_types/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "FinalModelData",
														ReferenceLocation: ast_domain.Location{
															Line:   38,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   66,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "UserID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 22,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[16]uint8"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "modernc.org/libc/uuid/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UserID",
														ReferenceLocation: ast_domain.Location{
															Line:   38,
															Column: 35,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/other_types.go"),
													Stringability:       5,
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
													TypeExpression:       typeExprFromString("[16]uint8"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "modernc.org/libc/uuid/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "UserID",
													ReferenceLocation: ast_domain.Location{
														Line:   38,
														Column: 35,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/other_types.go"),
												Stringability:       5,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[16]uint8"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "modernc.org/libc/uuid/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "UserID",
												ReferenceLocation: ast_domain.Location{
													Line:   38,
													Column: 35,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/other_types.go"),
											Stringability:       5,
										},
									},
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
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
							OriginalSourcePath:   new("partials/prop_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "consumer_title_state_directuuid_string_e857cbf2",
								PartialAlias:        "consumer",
								PartialPackageName:  "partials_prop_consumer_45fb2f06",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   40,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"title": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 71,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "DirectUUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DirectUUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   40,
																	Column: 71,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DirectUUID",
															ReferenceLocation: ast_domain.Location{
																Line:   40,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												Optional: false,
												Computed: false,
												RelativeLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											Args: []ast_domain.Expression{},
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
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "",
														CanonicalPackagePath: "",
													},
												},
												Stringability: 1,
											},
										},
										Location: ast_domain.Location{
											Line:   40,
											Column: 71,
										},
										GoFieldName: "Title",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
											},
											Stringability: 1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"title": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:12",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/prop_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "prop-implicit-string",
								Location: ast_domain.Location{
									Line:   40,
									Column: 41,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 37,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "title",
								RawExpression: "state.DirectUUID",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "DirectUUID",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DirectUUID",
												ReferenceLocation: ast_domain.Location{
													Line:   40,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       2,
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
											TypeExpression:       typeExprFromString("uuid.UUID"),
											PackageAlias:         "uuid",
											CanonicalPackagePath: "github.com/google/uuid",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "DirectUUID",
											ReferenceLocation: ast_domain.Location{
												Line:   40,
												Column: 71,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   63,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       2,
									},
								},
								Location: ast_domain.Location{
									Line:   40,
									Column: 71,
								},
								NameLocation: ast_domain.Location{
									Line:   40,
									Column: 63,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("uuid.UUID"),
										PackageAlias:         "uuid",
										CanonicalPackagePath: "github.com/google/uuid",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "DirectUUID",
										ReferenceLocation: ast_domain.Location{
											Line:   40,
											Column: 71,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   63,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       2,
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
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:12:0",
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
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "title",
										Location: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:12:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 23,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 26,
												},
												RawExpression: "props.Title",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Title",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "",
																	CanonicalPackagePath: "",
																},
															},
															BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath:  new("partials/prop_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
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
															PackageAlias:         "partials_prop_consumer_45fb2f06",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 26,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "",
																CanonicalPackagePath: "",
															},
														},
														BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
														OriginalSourcePath:  new("partials/prop_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 26,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
													},
													BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:  new("partials/prop_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:12:1",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "id",
										Location: ast_domain.Location{
											Line:   24,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 20,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:12:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   24,
													Column: 23,
												},
												RawExpression: "props.UUID",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "UUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath:  new("partials/prop_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "UUID",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
														OriginalSourcePath:  new("partials/prop_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:       2,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UUID",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:  new("partials/prop_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:       2,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   25,
										Column: 32,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 26,
									},
									RawExpression: "props.OptionalTitle != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath: new("partials/prop_consumer.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "OptionalTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OptionalTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:    new("partials/prop_consumer.pk"),
													GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:         1,
													IsPointerToStringable: true,
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
													TypeExpression:       typeExprFromString("*string"),
													PackageAlias:         "partials_prop_consumer_45fb2f06",
													CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "OptionalTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
												OriginalSourcePath:    new("partials/prop_consumer.pk"),
												GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
												Stringability:         1,
												IsPointerToStringable: true,
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/prop_consumer.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:12:2",
									RelativeLocation: ast_domain.Location{
										Line:   25,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "optional",
										Location: ast_domain.Location{
											Line:   25,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   25,
											Column: 60,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:12:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 60,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   25,
													Column: 63,
												},
												RawExpression: "props.OptionalTitle",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 63,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "OptionalTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*string"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "OptionalTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 63,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath:    new("partials/prop_consumer.pk"),
															GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
															Stringability:         1,
															IsPointerToStringable: true,
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
															TypeExpression:       typeExprFromString("*string"),
															PackageAlias:         "partials_prop_consumer_45fb2f06",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "OptionalTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 63,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
														OriginalSourcePath:    new("partials/prop_consumer.pk"),
														GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:         1,
														IsPointerToStringable: true,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OptionalTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 63,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:    new("partials/prop_consumer.pk"),
													GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:         1,
													IsPointerToStringable: true,
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
							Line:   22,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
							OriginalSourcePath:   new("partials/prop_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "consumer_uuid_state_directuuid_31ebf6ff",
								PartialAlias:        "consumer",
								PartialPackageName:  "partials_prop_consumer_45fb2f06",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   42,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"uuid": ast_domain.PropValue{
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 66,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "DirectUUID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DirectUUID",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 66,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   63,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DirectUUID",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 66,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DirectUUID",
													ReferenceLocation: ast_domain.Location{
														Line:   42,
														Column: 66,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   63,
														Column: 2,
													},
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DirectUUID",
														ReferenceLocation: ast_domain.Location{
															Line:   42,
															Column: 66,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   63,
															Column: 2,
														},
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       2,
											},
										},
										Location: ast_domain.Location{
											Line:   42,
											Column: 66,
										},
										GoFieldName: "UUID",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DirectUUID",
												ReferenceLocation: ast_domain.Location{
													Line:   42,
													Column: 66,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DirectUUID",
													ReferenceLocation: ast_domain.Location{
														Line:   42,
														Column: 66,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   63,
														Column: 2,
													},
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       2,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"uuid": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:13",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/prop_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "prop-exact-match",
								Location: ast_domain.Location{
									Line:   42,
									Column: 41,
								},
								NameLocation: ast_domain.Location{
									Line:   42,
									Column: 37,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "uuid",
								RawExpression: "state.DirectUUID",
								Expression: &ast_domain.MemberExpression{
									Base: &ast_domain.Identifier{
										Name: "state",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
												PackageAlias:         "main_aaf9a2e0",
												CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "state",
												ReferenceLocation: ast_domain.Location{
													Line:   42,
													Column: 66,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   0,
													Column: 0,
												},
											},
											BaseCodeGenVarName: new("pageData"),
											OriginalSourcePath: new("main.pk"),
										},
									},
									Property: &ast_domain.Identifier{
										Name: "DirectUUID",
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 7,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("uuid.UUID"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "DirectUUID",
												ReferenceLocation: ast_domain.Location{
													Line:   42,
													Column: 66,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   63,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
											Stringability:       2,
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
											TypeExpression:       typeExprFromString("uuid.UUID"),
											PackageAlias:         "uuid",
											CanonicalPackagePath: "github.com/google/uuid",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "DirectUUID",
											ReferenceLocation: ast_domain.Location{
												Line:   42,
												Column: 66,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   63,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
										Stringability:       2,
									},
								},
								Location: ast_domain.Location{
									Line:   42,
									Column: 66,
								},
								NameLocation: ast_domain.Location{
									Line:   42,
									Column: 59,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("uuid.UUID"),
										PackageAlias:         "uuid",
										CanonicalPackagePath: "github.com/google/uuid",
									},
									Symbol: &ast_domain.ResolvedSymbol{
										Name: "DirectUUID",
										ReferenceLocation: ast_domain.Location{
											Line:   42,
											Column: 66,
										},
										DeclarationLocation: ast_domain.Location{
											Line:   63,
											Column: 2,
										},
									},
									BaseCodeGenVarName:  new("pageData"),
									OriginalSourcePath:  new("main.pk"),
									GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
									Stringability:       2,
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
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:13:0",
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
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "title",
										Location: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:13:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 23,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 26,
												},
												RawExpression: "props.Title",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_uuid_state_directuuid_31ebf6ff"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Title",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_consumer_uuid_state_directuuid_31ebf6ff"),
															OriginalSourcePath:  new("partials/prop_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
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
															PackageAlias:         "partials_prop_consumer_45fb2f06",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 26,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_consumer_uuid_state_directuuid_31ebf6ff"),
														OriginalSourcePath:  new("partials/prop_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 26,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_consumer_uuid_state_directuuid_31ebf6ff"),
													OriginalSourcePath:  new("partials/prop_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:13:1",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "id",
										Location: ast_domain.Location{
											Line:   24,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 20,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:13:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   24,
													Column: 23,
												},
												RawExpression: "props.UUID",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_uuid_state_directuuid_31ebf6ff"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "UUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("uuid.UUID"),
																	PackageAlias:         "uuid",
																	CanonicalPackagePath: "github.com/google/uuid",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "DirectUUID",
																	ReferenceLocation: ast_domain.Location{
																		Line:   42,
																		Column: 66,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   63,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_consumer_uuid_state_directuuid_31ebf6ff"),
															OriginalSourcePath:  new("partials/prop_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "UUID",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DirectUUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   42,
																	Column: 66,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 2,
																},
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_consumer_uuid_state_directuuid_31ebf6ff"),
														OriginalSourcePath:  new("partials/prop_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:       2,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UUID",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DirectUUID",
															ReferenceLocation: ast_domain.Location{
																Line:   42,
																Column: 66,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 2,
															},
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_consumer_uuid_state_directuuid_31ebf6ff"),
													OriginalSourcePath:  new("partials/prop_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:       2,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   25,
										Column: 32,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 26,
									},
									RawExpression: "props.OptionalTitle != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_consumer_uuid_state_directuuid_31ebf6ff"),
													OriginalSourcePath: new("partials/prop_consumer.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "OptionalTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OptionalTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:    new("props_consumer_uuid_state_directuuid_31ebf6ff"),
													OriginalSourcePath:    new("partials/prop_consumer.pk"),
													GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:         1,
													IsPointerToStringable: true,
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
													TypeExpression:       typeExprFromString("*string"),
													PackageAlias:         "partials_prop_consumer_45fb2f06",
													CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "OptionalTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:    new("props_consumer_uuid_state_directuuid_31ebf6ff"),
												OriginalSourcePath:    new("partials/prop_consumer.pk"),
												GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
												Stringability:         1,
												IsPointerToStringable: true,
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/prop_consumer.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:13:2",
									RelativeLocation: ast_domain.Location{
										Line:   25,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "optional",
										Location: ast_domain.Location{
											Line:   25,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   25,
											Column: 60,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:13:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 60,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   25,
													Column: 63,
												},
												RawExpression: "props.OptionalTitle",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 63,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_uuid_state_directuuid_31ebf6ff"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "OptionalTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*string"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "OptionalTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 63,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:    new("props_consumer_uuid_state_directuuid_31ebf6ff"),
															OriginalSourcePath:    new("partials/prop_consumer.pk"),
															GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
															Stringability:         1,
															IsPointerToStringable: true,
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
															TypeExpression:       typeExprFromString("*string"),
															PackageAlias:         "partials_prop_consumer_45fb2f06",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "OptionalTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 63,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:    new("props_consumer_uuid_state_directuuid_31ebf6ff"),
														OriginalSourcePath:    new("partials/prop_consumer.pk"),
														GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:         1,
														IsPointerToStringable: true,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OptionalTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 63,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:    new("props_consumer_uuid_state_directuuid_31ebf6ff"),
													OriginalSourcePath:    new("partials/prop_consumer.pk"),
													GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:         1,
													IsPointerToStringable: true,
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
							Line:   22,
							Column: 5,
						},
						TagName: "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
							OriginalSourcePath:   new("partials/prop_consumer.pk"),
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey:       "consumer_title_state_directuuid_string_e857cbf2",
								PartialAlias:        "consumer",
								PartialPackageName:  "partials_prop_consumer_45fb2f06",
								InvokerPackageAlias: "main_aaf9a2e0",
								Location: ast_domain.Location{
									Line:   44,
									Column: 9,
								},
								PassedProps: map[string]ast_domain.PropValue{
									"title": ast_domain.PropValue{
										Expression: &ast_domain.CallExpression{
											Callee: &ast_domain.MemberExpression{
												Base: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "state",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
																PackageAlias:         "main_aaf9a2e0",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "state",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 71,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("pageData"),
															OriginalSourcePath: new("main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "DirectUUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "DirectUUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   44,
																	Column: 71,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   63,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "DirectUUID",
															ReferenceLocation: ast_domain.Location{
																Line:   44,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   63,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
														Stringability:       2,
													},
												},
												Property: &ast_domain.Identifier{
													Name: "String",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 18,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("function"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "String",
															ReferenceLocation: ast_domain.Location{
																Line:   44,
																Column: 71,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   291,
																Column: 1,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
														TypeExpression:       typeExprFromString("function"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "String",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   291,
															Column: 1,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
												},
											},
											Args: []ast_domain.Expression{},
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												PropDataSource: &ast_domain.PropDataSource{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													BaseCodeGenVarName: new("pageData"),
												},
												BaseCodeGenVarName: new("pageData"),
												Stringability:      1,
											},
										},
										Location: ast_domain.Location{
											Line:   44,
											Column: 71,
										},
										GoFieldName: "Title",
										InvokerAnnotation: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											PropDataSource: &ast_domain.PropDataSource{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												BaseCodeGenVarName: new("pageData"),
											},
											BaseCodeGenVarName: new("pageData"),
											Stringability:      1,
										},
									},
								},
							},
							DynamicAttributeOrigins: map[string]string{
								"title": "main_aaf9a2e0",
							},
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:14",
							RelativeLocation: ast_domain.Location{
								Line:   22,
								Column: 5,
							},
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression:       typeExprFromString("string"),
									PackageAlias:         "",
									CanonicalPackagePath: "",
								},
								OriginalSourcePath: new("partials/prop_consumer.pk"),
								Stringability:      1,
							},
						},
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "prop-explicit-string",
								Location: ast_domain.Location{
									Line:   44,
									Column: 41,
								},
								NameLocation: ast_domain.Location{
									Line:   44,
									Column: 37,
								},
							},
						},
						DynamicAttributes: []ast_domain.DynamicAttribute{
							ast_domain.DynamicAttribute{
								Name:          "title",
								RawExpression: "state.DirectUUID.String()",
								Expression: &ast_domain.CallExpression{
									Callee: &ast_domain.MemberExpression{
										Base: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "state",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
														PackageAlias:         "main_aaf9a2e0",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/pages/main_aaf9a2e0",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "state",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("pageData"),
													OriginalSourcePath: new("main.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "DirectUUID",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "DirectUUID",
														ReferenceLocation: ast_domain.Location{
															Line:   44,
															Column: 71,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   63,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
													Stringability:       2,
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
													TypeExpression:       typeExprFromString("uuid.UUID"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "DirectUUID",
													ReferenceLocation: ast_domain.Location{
														Line:   44,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   63,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												Stringability:       2,
											},
										},
										Property: &ast_domain.Identifier{
											Name: "String",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 18,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("function"),
													PackageAlias:         "uuid",
													CanonicalPackagePath: "github.com/google/uuid",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "String",
													ReferenceLocation: ast_domain.Location{
														Line:   44,
														Column: 71,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   291,
														Column: 1,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
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
												TypeExpression:       typeExprFromString("function"),
												PackageAlias:         "uuid",
												CanonicalPackagePath: "github.com/google/uuid",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "String",
												ReferenceLocation: ast_domain.Location{
													Line:   44,
													Column: 71,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   291,
													Column: 1,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("../../../../../../../../../../go/pkg/mod/github.com/google/uuid@v1.6.0/uuid.go"),
										},
									},
									Args: []ast_domain.Expression{},
									RelativeLocation: ast_domain.Location{
										Line:   1,
										Column: 1,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "uuid",
											CanonicalPackagePath: "github.com/google/uuid",
										},
										BaseCodeGenVarName: new("pageData"),
										Stringability:      1,
									},
								},
								Location: ast_domain.Location{
									Line:   44,
									Column: 71,
								},
								NameLocation: ast_domain.Location{
									Line:   44,
									Column: 63,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression:       typeExprFromString("string"),
										PackageAlias:         "uuid",
										CanonicalPackagePath: "github.com/google/uuid",
									},
									BaseCodeGenVarName: new("pageData"),
									Stringability:      1,
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
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:14:0",
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
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "title",
										Location: ast_domain.Location{
											Line:   23,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   23,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   23,
											Column: 23,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:14:0:0",
											RelativeLocation: ast_domain.Location{
												Line:   23,
												Column: 23,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   23,
													Column: 26,
												},
												RawExpression: "props.Title",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Title",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Title",
																ReferenceLocation: ast_domain.Location{
																	Line:   23,
																	Column: 26,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   37,
																	Column: 2,
																},
															},
															PropDataSource: &ast_domain.PropDataSource{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("string"),
																	PackageAlias:         "uuid",
																	CanonicalPackagePath: "github.com/google/uuid",
																},
																BaseCodeGenVarName: new("pageData"),
															},
															BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath:  new("partials/prop_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
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
															PackageAlias:         "partials_prop_consumer_45fb2f06",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Title",
															ReferenceLocation: ast_domain.Location{
																Line:   23,
																Column: 26,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   37,
																Column: 2,
															},
														},
														PropDataSource: &ast_domain.PropDataSource{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															BaseCodeGenVarName: new("pageData"),
														},
														BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
														OriginalSourcePath:  new("partials/prop_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Title",
														ReferenceLocation: ast_domain.Location{
															Line:   23,
															Column: 26,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   37,
															Column: 2,
														},
													},
													PropDataSource: &ast_domain.PropDataSource{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("string"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														BaseCodeGenVarName: new("pageData"),
													},
													BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:  new("partials/prop_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:       1,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   24,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:14:1",
									RelativeLocation: ast_domain.Location{
										Line:   24,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "id",
										Location: ast_domain.Location{
											Line:   24,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   24,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   24,
											Column: 20,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:14:1:0",
											RelativeLocation: ast_domain.Location{
												Line:   24,
												Column: 20,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   24,
													Column: 23,
												},
												RawExpression: "props.UUID",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "UUID",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("uuid.UUID"),
																PackageAlias:         "uuid",
																CanonicalPackagePath: "github.com/google/uuid",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "UUID",
																ReferenceLocation: ast_domain.Location{
																	Line:   24,
																	Column: 23,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   38,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath:  new("partials/prop_consumer.pk"),
															GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
															Stringability:       2,
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
															TypeExpression:       typeExprFromString("uuid.UUID"),
															PackageAlias:         "uuid",
															CanonicalPackagePath: "github.com/google/uuid",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "UUID",
															ReferenceLocation: ast_domain.Location{
																Line:   24,
																Column: 23,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   38,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
														OriginalSourcePath:  new("partials/prop_consumer.pk"),
														GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:       2,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("uuid.UUID"),
														PackageAlias:         "uuid",
														CanonicalPackagePath: "github.com/google/uuid",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "UUID",
														ReferenceLocation: ast_domain.Location{
															Line:   24,
															Column: 23,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   38,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:  new("partials/prop_consumer.pk"),
													GeneratedSourcePath: new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:       2,
												},
											},
										},
									},
								},
							},
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   25,
									Column: 9,
								},
								TagName: "p",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
									OriginalSourcePath:   new("partials/prop_consumer.pk"),
									IsStructurallyStatic: true,
								},
								DirIf: &ast_domain.Directive{
									Type: ast_domain.DirectiveIf,
									Location: ast_domain.Location{
										Line:   25,
										Column: 32,
									},
									NameLocation: ast_domain.Location{
										Line:   25,
										Column: 26,
									},
									RawExpression: "props.OptionalTitle != nil",
									Expression: &ast_domain.BinaryExpression{
										Left: &ast_domain.MemberExpression{
											Base: &ast_domain.Identifier{
												Name: "props",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "props",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath: new("partials/prop_consumer.pk"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "OptionalTitle",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 7,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OptionalTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 32,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:    new("partials/prop_consumer.pk"),
													GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:         1,
													IsPointerToStringable: true,
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
													TypeExpression:       typeExprFromString("*string"),
													PackageAlias:         "partials_prop_consumer_45fb2f06",
													CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "OptionalTitle",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 32,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   39,
														Column: 2,
													},
												},
												BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
												OriginalSourcePath:    new("partials/prop_consumer.pk"),
												GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
												Stringability:         1,
												IsPointerToStringable: true,
											},
										},
										Operator: "!=",
										Right: &ast_domain.NilLiteral{
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 24,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("nil"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("bool"),
												PackageAlias:         "",
												CanonicalPackagePath: "",
											},
											OriginalSourcePath: new("partials/prop_consumer.pk"),
											Stringability:      1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("bool"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:14:2",
									RelativeLocation: ast_domain.Location{
										Line:   25,
										Column: 9,
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "",
											CanonicalPackagePath: "",
										},
										OriginalSourcePath: new("partials/prop_consumer.pk"),
										Stringability:      1,
									},
								},
								Attributes: []ast_domain.HTMLAttribute{
									ast_domain.HTMLAttribute{
										Name:  "id",
										Value: "optional",
										Location: ast_domain.Location{
											Line:   25,
											Column: 16,
										},
										NameLocation: ast_domain.Location{
											Line:   25,
											Column: 12,
										},
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   25,
											Column: 60,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("partials_prop_consumer_45fb2f06"),
											OriginalSourcePath:   new("partials/prop_consumer.pk"),
										},
										Key: &ast_domain.StringLiteral{
											Value: "r.0:14:2:0",
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 60,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("string"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												OriginalSourcePath: new("partials/prop_consumer.pk"),
												Stringability:      1,
											},
										},
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   25,
													Column: 63,
												},
												RawExpression: "props.OptionalTitle",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "props",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("partials_prop_consumer_45fb2f06.Props"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "props",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 63,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath: new("partials/prop_consumer.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "OptionalTitle",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 7,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("*string"),
																PackageAlias:         "partials_prop_consumer_45fb2f06",
																CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "OptionalTitle",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 63,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   39,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
															OriginalSourcePath:    new("partials/prop_consumer.pk"),
															GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
															Stringability:         1,
															IsPointerToStringable: true,
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
															TypeExpression:       typeExprFromString("*string"),
															PackageAlias:         "partials_prop_consumer_45fb2f06",
															CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "OptionalTitle",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 63,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   39,
																Column: 2,
															},
														},
														BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
														OriginalSourcePath:    new("partials/prop_consumer.pk"),
														GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
														Stringability:         1,
														IsPointerToStringable: true,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("*string"),
														PackageAlias:         "partials_prop_consumer_45fb2f06",
														CanonicalPackagePath: "testcase_38_third_party_types/dist/partials/partials_prop_consumer_45fb2f06",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "OptionalTitle",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 63,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   39,
															Column: 2,
														},
													},
													BaseCodeGenVarName:    new("props_consumer_title_state_directuuid_string_e857cbf2"),
													OriginalSourcePath:    new("partials/prop_consumer.pk"),
													GeneratedSourcePath:   new("dist/partials/partials_prop_consumer_45fb2f06/generated.go"),
													Stringability:         1,
													IsPointerToStringable: true,
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
