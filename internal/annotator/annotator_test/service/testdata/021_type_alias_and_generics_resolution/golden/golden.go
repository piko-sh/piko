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
					Line:   48,
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
						Line:   48,
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
							Line:   49,
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
								Line:   49,
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   49,
									Column: 12,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:0:0",
									RelativeLocation: ast_domain.Location{
										Line:   49,
										Column: 12,
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
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   49,
											Column: 12,
										},
										Literal: "Total Users: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   49,
											Column: 28,
										},
										RawExpression: "state.Users.TotalCount",
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
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 28,
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
													Name: "Users",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 7,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.PagedResult[models.User]"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Users",
															ReferenceLocation: ast_domain.Location{
																Line:   49,
																Column: 28,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
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
														TypeExpression:       typeExprFromString("models.PagedResult[models.User]"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Users",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "TotalCount",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 13,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("int"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "TotalCount",
														ReferenceLocation: ast_domain.Location{
															Line:   49,
															Column: 28,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   47,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/models.go"),
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
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "TotalCount",
													ReferenceLocation: ast_domain.Location{
														Line:   49,
														Column: 28,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   47,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/models.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("int"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "TotalCount",
												ReferenceLocation: ast_domain.Location{
													Line:   49,
													Column: 28,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   47,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/models.go"),
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
							Line:   50,
							Column: 9,
						},
						TagName: "ul",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("main_aaf9a2e0"),
							OriginalSourcePath:   new("main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
							RelativeLocation: ast_domain.Location{
								Line:   50,
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
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   51,
									Column: 13,
								},
								TagName: "li",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("main_aaf9a2e0"),
									OriginalSourcePath:   new("main.pk"),
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   51,
										Column: 24,
									},
									NameLocation: ast_domain.Location{
										Line:   51,
										Column: 17,
									},
									RawExpression: "user in state.Users.Items",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: nil,
										ItemVariable: &ast_domain.Identifier{
											Name: "user",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "user",
													ReferenceLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("user"),
												OriginalSourcePath: new("main.pk"),
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 9,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("main_aaf9a2e0.Response"),
															PackageAlias:         "main_aaf9a2e0",
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/dist/pages/main_aaf9a2e0",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 24,
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
													Name: "Users",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 15,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.PagedResult[models.User]"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Users",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 24,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   29,
																Column: 23,
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
													Column: 9,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.PagedResult[models.User]"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Users",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 24,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   29,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("dist/pages/main_aaf9a2e0/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Items",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 21,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Items",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 24,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   20,
															Column: 0,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/models.go"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 9,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Items",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 24,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   20,
														Column: 0,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/models.go"),
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]models.User"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 24,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   20,
													Column: 0,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/models.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]models.User"),
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Items",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 24,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   20,
													Column: 0,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/models.go"),
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]models.User"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Items",
											ReferenceLocation: ast_domain.Location{
												Line:   51,
												Column: 24,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   20,
												Column: 0,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/models.go"),
									},
								},
								DirKey: &ast_domain.Directive{
									Type: ast_domain.DirectiveKey,
									Location: ast_domain.Location{
										Line:   51,
										Column: 58,
									},
									NameLocation: ast_domain.Location{
										Line:   51,
										Column: 51,
									},
									RawExpression: "user.Name",
									Expression: &ast_domain.MemberExpression{
										Base: &ast_domain.Identifier{
											Name: "user",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("models.User"),
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "user",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 69,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("user"),
												OriginalSourcePath: new("main.pk"),
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
													PackageAlias:         "models",
													CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Name",
													ReferenceLocation: ast_domain.Location{
														Line:   51,
														Column: 58,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   42,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("user"),
												OriginalSourcePath:  new("main.pk"),
												GeneratedSourcePath: new("models/models.go"),
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
												PackageAlias:         "models",
												CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Name",
												ReferenceLocation: ast_domain.Location{
													Line:   51,
													Column: 69,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   42,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("user"),
											OriginalSourcePath:  new("main.pk"),
											GeneratedSourcePath: new("models/models.go"),
											Stringability:       1,
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("string"),
											PackageAlias:         "models",
											CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Name",
											ReferenceLocation: ast_domain.Location{
												Line:   51,
												Column: 58,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   42,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("user"),
										OriginalSourcePath:  new("main.pk"),
										GeneratedSourcePath: new("models/models.go"),
										Stringability:       1,
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   51,
												Column: 13,
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
											Literal: "r.0:1:0.",
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
												OriginalSourcePath: new("main.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "user",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 1,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.User"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "user",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 69,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   0,
																Column: 0,
															},
														},
														BaseCodeGenVarName: new("user"),
														OriginalSourcePath: new("main.pk"),
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
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   51,
																Column: 58,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   42,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("user"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/models.go"),
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
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   51,
															Column: 69,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   42,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("user"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/models.go"),
													Stringability:       1,
												},
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   51,
										Column: 13,
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
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   51,
											Column: 69,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("main_aaf9a2e0"),
											OriginalSourcePath:   new("main.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   51,
														Column: 69,
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
													Literal: "r.0:1:0.",
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
														OriginalSourcePath: new("main.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.MemberExpression{
														Base: &ast_domain.Identifier{
															Name: "user",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 1,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.User"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "user",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
																		Column: 69,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   0,
																		Column: 0,
																	},
																},
																BaseCodeGenVarName: new("user"),
																OriginalSourcePath: new("main.pk"),
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
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "Name",
																	ReferenceLocation: ast_domain.Location{
																		Line:   51,
																		Column: 58,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   42,
																		Column: 2,
																	},
																},
																BaseCodeGenVarName:  new("user"),
																OriginalSourcePath:  new("main.pk"),
																GeneratedSourcePath: new("models/models.go"),
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
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   51,
																	Column: 69,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("user"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/models.go"),
															Stringability:       1,
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   51,
														Column: 69,
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
													Literal: ":0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   51,
												Column: 69,
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
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   51,
													Column: 69,
												},
												Literal: "\n                ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("main.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   52,
													Column: 20,
												},
												RawExpression: "user.Name",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "user",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "user",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("user"),
															OriginalSourcePath: new("main.pk"),
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
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Name",
																ReferenceLocation: ast_domain.Location{
																	Line:   52,
																	Column: 20,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   42,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("user"),
															OriginalSourcePath:  new("main.pk"),
															GeneratedSourcePath: new("models/models.go"),
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
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Name",
															ReferenceLocation: ast_domain.Location{
																Line:   52,
																Column: 20,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   42,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("user"),
														OriginalSourcePath:  new("main.pk"),
														GeneratedSourcePath: new("models/models.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_21_type_alias_and_generics_resolution/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Name",
														ReferenceLocation: ast_domain.Location{
															Line:   52,
															Column: 20,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   42,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("user"),
													OriginalSourcePath:  new("main.pk"),
													GeneratedSourcePath: new("models/models.go"),
													Stringability:       1,
												},
											},
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   52,
													Column: 32,
												},
												Literal: "\n            ",
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
	}
}()
