package main_test

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
								TextContent: "Addresses (from pkg/1/dto):",
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
							Line:   24,
							Column: 5,
						},
						TagName: "ul",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:1",
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
								OriginalSourcePath: new("pages/main.pk"),
								Stringability:      1,
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
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
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
									RawExpression: "address in state.User.Addresses",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: &ast_domain.Identifier{
											Name: "__pikoLoopIdx",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "__pikoLoopIdx",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("__pikoLoopIdx"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										ItemVariable: &ast_domain.Identifier{
											Name: "address",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.Address"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "address",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 51,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("address"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 18,
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
													Name: "User",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 18,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.User"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "User",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 18,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
																Column: 23,
															},
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
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "User",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Addresses",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.Address"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Addresses",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   71,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/models/user.go"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 12,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.Address"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Addresses",
													ReferenceLocation: ast_domain.Location{
														Line:   25,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   71,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/models/user.go"),
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.Address"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Addresses",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/models/user.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.Address"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Addresses",
												ReferenceLocation: ast_domain.Location{
													Line:   25,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   71,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/models/user.go"),
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]dto.Address"),
											PackageAlias:         "dto",
											CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Addresses",
											ReferenceLocation: ast_domain.Location{
												Line:   25,
												Column: 18,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   71,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("pkg/models/user.go"),
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
												OriginalSourcePath: new("pages/main.pk"),
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.Identifier{
												Name: "address",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.Address"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "address",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 51,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("address"),
													OriginalSourcePath: new("pages/main.pk"),
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
										OriginalSourcePath: new("pages/main.pk"),
										Stringability:      1,
									},
								},
								Children: []*ast_domain.TemplateNode{
									&ast_domain.TemplateNode{
										NodeType: ast_domain.NodeText,
										Location: ast_domain.Location{
											Line:   25,
											Column: 51,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 51,
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
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.Identifier{
														Name: "address",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.Address"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "address",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 51,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("address"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   25,
														Column: 51,
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
													Literal: ":0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   25,
												Column: 51,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   25,
													Column: 54,
												},
												RawExpression: "address.Street",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "address",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.Address"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "address",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("address"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Street",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Street",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   66,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("address"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/1/dto/address.go"),
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
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Street",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   66,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("address"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/1/dto/address.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Street",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   66,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("address"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/1/dto/address.go"),
													Stringability:       1,
												},
											},
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   25,
													Column: 71,
												},
												Literal: ", ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   25,
													Column: 76,
												},
												RawExpression: "address.City",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "address",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.Address"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "address",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 76,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("address"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "City",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "City",
																ReferenceLocation: ast_domain.Location{
																	Line:   25,
																	Column: 76,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   67,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("address"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/1/dto/address.go"),
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
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "City",
															ReferenceLocation: ast_domain.Location{
																Line:   25,
																Column: 76,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   67,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("address"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/1/dto/address.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "City",
														ReferenceLocation: ast_domain.Location{
															Line:   25,
															Column: 76,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   67,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("address"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/1/dto/address.go"),
													Stringability:       1,
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
							Line:   28,
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
							Value: "r.0:2",
							RelativeLocation: ast_domain.Location{
								Line:   28,
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
									Line:   28,
									Column: 9,
								},
								TextContent: "Contacts (from pkg/2/dto):",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:2:0",
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
							Line:   29,
							Column: 5,
						},
						TagName: "ul",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:3",
							RelativeLocation: ast_domain.Location{
								Line:   29,
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
								NodeType: ast_domain.NodeElement,
								Location: ast_domain.Location{
									Line:   30,
									Column: 7,
								},
								TagName: "li",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								DirFor: &ast_domain.Directive{
									Type: ast_domain.DirectiveFor,
									Location: ast_domain.Location{
										Line:   30,
										Column: 18,
									},
									NameLocation: ast_domain.Location{
										Line:   30,
										Column: 11,
									},
									RawExpression: "contact in state.User.Contacts",
									Expression: &ast_domain.ForInExpression{
										IndexVariable: &ast_domain.Identifier{
											Name: "__pikoLoopIdx",
											RelativeLocation: ast_domain.Location{
												Line:   0,
												Column: 0,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("int"),
													PackageAlias:         "",
													CanonicalPackagePath: "",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "__pikoLoopIdx",
													ReferenceLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("__pikoLoopIdx"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										ItemVariable: &ast_domain.Identifier{
											Name: "contact",
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 1,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("dto.Contact"),
													PackageAlias:         "dto",
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "contact",
													ReferenceLocation: ast_domain.Location{
														Line:   30,
														Column: 50,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   0,
														Column: 0,
													},
												},
												BaseCodeGenVarName: new("contact"),
												OriginalSourcePath: new("pages/main.pk"),
											},
										},
										Collection: &ast_domain.MemberExpression{
											Base: &ast_domain.MemberExpression{
												Base: &ast_domain.Identifier{
													Name: "state",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 12,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
															PackageAlias:         "pages_main_594861c5",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/dist/pages/pages_main_594861c5",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "state",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 18,
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
													Name: "User",
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 18,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("models.User"),
															PackageAlias:         "models",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "User",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 18,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   52,
																Column: 23,
															},
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
													Column: 12,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("models.User"),
														PackageAlias:         "models",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "User",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   52,
															Column: 23,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Contacts",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 23,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("[]dto.Contact"),
														PackageAlias:         "dto2",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Contacts",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 18,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   72,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/models/user.go"),
												},
											},
											Optional: false,
											Computed: false,
											RelativeLocation: ast_domain.Location{
												Line:   1,
												Column: 12,
											},
											GoAnnotations: &ast_domain.GoGeneratorAnnotation{
												ResolvedType: &ast_domain.ResolvedTypeInfo{
													TypeExpression:       typeExprFromString("[]dto.Contact"),
													PackageAlias:         "dto2",
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Contacts",
													ReferenceLocation: ast_domain.Location{
														Line:   30,
														Column: 18,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   72,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/models/user.go"),
											},
										},
										RelativeLocation: ast_domain.Location{
											Line:   1,
											Column: 1,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.Contact"),
												PackageAlias:         "dto2",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Contacts",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   72,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/models/user.go"),
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("[]dto.Contact"),
												PackageAlias:         "dto2",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Contacts",
												ReferenceLocation: ast_domain.Location{
													Line:   30,
													Column: 18,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   72,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/models/user.go"),
										},
									},
									GoAnnotations: &ast_domain.GoGeneratorAnnotation{
										ResolvedType: &ast_domain.ResolvedTypeInfo{
											TypeExpression:       typeExprFromString("[]dto.Contact"),
											PackageAlias:         "dto2",
											CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
										},
										Symbol: &ast_domain.ResolvedSymbol{
											Name: "Contacts",
											ReferenceLocation: ast_domain.Location{
												Line:   30,
												Column: 18,
											},
											DeclarationLocation: ast_domain.Location{
												Line:   72,
												Column: 2,
											},
										},
										BaseCodeGenVarName:  new("pageData"),
										OriginalSourcePath:  new("pages/main.pk"),
										GeneratedSourcePath: new("pkg/models/user.go"),
									},
								},
								Key: &ast_domain.TemplateLiteral{
									Parts: []ast_domain.TemplateLiteralPart{
										ast_domain.TemplateLiteralPart{
											IsLiteral: true,
											RelativeLocation: ast_domain.Location{
												Line:   30,
												Column: 7,
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
											Literal: "r.0:3:0.",
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
												OriginalSourcePath: new("pages/main.pk"),
												Stringability:      1,
											},
											Expression: &ast_domain.Identifier{
												Name: "contact",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.Contact"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "contact",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 50,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   0,
															Column: 0,
														},
													},
													BaseCodeGenVarName: new("contact"),
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
										},
									},
									RelativeLocation: ast_domain.Location{
										Line:   30,
										Column: 7,
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
											Line:   30,
											Column: 50,
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalPackageAlias: new("pages_main_594861c5"),
											OriginalSourcePath:   new("pages/main.pk"),
										},
										Key: &ast_domain.TemplateLiteral{
											Parts: []ast_domain.TemplateLiteralPart{
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 50,
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
													Literal: "r.0:3:0.",
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
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
													Expression: &ast_domain.Identifier{
														Name: "contact",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.Contact"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "contact",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 50,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("contact"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
												},
												ast_domain.TemplateLiteralPart{
													IsLiteral: true,
													RelativeLocation: ast_domain.Location{
														Line:   30,
														Column: 50,
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
													Literal: ":0",
												},
											},
											RelativeLocation: ast_domain.Location{
												Line:   30,
												Column: 50,
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
										RichText: []ast_domain.TextPart{
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   30,
													Column: 53,
												},
												RawExpression: "contact.Email",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "contact",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.Contact"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "contact",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("contact"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Email",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Email",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 53,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   66,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("contact"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/2/dto/contact.go"),
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
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Email",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 53,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   66,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("contact"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/2/dto/contact.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Email",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 53,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   66,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("contact"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/2/dto/contact.go"),
													Stringability:       1,
												},
											},
											ast_domain.TextPart{
												IsLiteral: true,
												Location: ast_domain.Location{
													Line:   30,
													Column: 69,
												},
												Literal: " - ",
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													OriginalSourcePath: new("pages/main.pk"),
												},
											},
											ast_domain.TextPart{
												IsLiteral: false,
												Location: ast_domain.Location{
													Line:   30,
													Column: 75,
												},
												RawExpression: "contact.Phone",
												Expression: &ast_domain.MemberExpression{
													Base: &ast_domain.Identifier{
														Name: "contact",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 1,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("dto.Contact"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "contact",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 75,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   0,
																	Column: 0,
																},
															},
															BaseCodeGenVarName: new("contact"),
															OriginalSourcePath: new("pages/main.pk"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Phone",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 9,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("string"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Phone",
																ReferenceLocation: ast_domain.Location{
																	Line:   30,
																	Column: 75,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   67,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("contact"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/2/dto/contact.go"),
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
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Phone",
															ReferenceLocation: ast_domain.Location{
																Line:   30,
																Column: 75,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   67,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("contact"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/2/dto/contact.go"),
														Stringability:       1,
													},
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Phone",
														ReferenceLocation: ast_domain.Location{
															Line:   30,
															Column: 75,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   67,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("contact"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/2/dto/contact.go"),
													Stringability:       1,
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
							Line:   33,
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
							Value: "r.0:4",
							RelativeLocation: ast_domain.Location{
								Line:   33,
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
									Line:   33,
									Column: 9,
								},
								TextContent: "Direct Access to Addresses:",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:4:0",
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
							Line:   34,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:5",
							RelativeLocation: ast_domain.Location{
								Line:   34,
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "safe-address",
								Location: ast_domain.Location{
									Line:   34,
									Column: 12,
								},
								NameLocation: ast_domain.Location{
									Line:   34,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   34,
									Column: 26,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:5:0",
									RelativeLocation: ast_domain.Location{
										Line:   34,
										Column: 26,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   34,
											Column: 26,
										},
										Literal: "First Address: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   34,
											Column: 44,
										},
										RawExpression: "state.User.Addresses[0].Street",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   34,
																		Column: 44,
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
															Name: "User",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.User"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   34,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   52,
																		Column: 23,
																	},
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
																TypeExpression:       typeExprFromString("models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   34,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   52,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Addresses",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]dto.Address"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Addresses",
																ReferenceLocation: ast_domain.Location{
																	Line:   34,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   71,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/models/user.go"),
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
															TypeExpression:       typeExprFromString("[]dto.Address"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Addresses",
															ReferenceLocation: ast_domain.Location{
																Line:   34,
																Column: 44,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/models/user.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 0,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 22,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Optional: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.Address"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Street",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 25,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Street",
														ReferenceLocation: ast_domain.Location{
															Line:   34,
															Column: 44,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   66,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/1/dto/address.go"),
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
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Street",
													ReferenceLocation: ast_domain.Location{
														Line:   34,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   66,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/1/dto/address.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Street",
												ReferenceLocation: ast_domain.Location{
													Line:   34,
													Column: 44,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   66,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/1/dto/address.go"),
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
							Line:   36,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:6",
							RelativeLocation: ast_domain.Location{
								Line:   36,
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "unsafe-address",
								Location: ast_domain.Location{
									Line:   36,
									Column: 12,
								},
								NameLocation: ast_domain.Location{
									Line:   36,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   36,
									Column: 28,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:6:0",
									RelativeLocation: ast_domain.Location{
										Line:   36,
										Column: 28,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   36,
											Column: 28,
										},
										Literal: "Out-of-Bounds Address: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   36,
											Column: 54,
										},
										RawExpression: "state.User.Addresses[99].City",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
																		Column: 54,
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
															Name: "User",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.User"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   36,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   52,
																		Column: 23,
																	},
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
																TypeExpression:       typeExprFromString("models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   52,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Addresses",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]dto.Address"),
																PackageAlias:         "dto",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Addresses",
																ReferenceLocation: ast_domain.Location{
																	Line:   36,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   71,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/models/user.go"),
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
															TypeExpression:       typeExprFromString("[]dto.Address"),
															PackageAlias:         "dto",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Addresses",
															ReferenceLocation: ast_domain.Location{
																Line:   36,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   71,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/models/user.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 99,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 22,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Optional: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.Address"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "City",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 26,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "City",
														ReferenceLocation: ast_domain.Location{
															Line:   36,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   67,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/1/dto/address.go"),
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
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "City",
													ReferenceLocation: ast_domain.Location{
														Line:   36,
														Column: 54,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   67,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/1/dto/address.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/1/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "City",
												ReferenceLocation: ast_domain.Location{
													Line:   36,
													Column: 54,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   67,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/1/dto/address.go"),
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
							Line:   38,
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
							Value: "r.0:7",
							RelativeLocation: ast_domain.Location{
								Line:   38,
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
									Line:   38,
									Column: 9,
								},
								TextContent: "Direct Access to Contacts:",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
									IsStatic:             true,
									IsStructurallyStatic: true,
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:7:0",
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
							Line:   39,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:8",
							RelativeLocation: ast_domain.Location{
								Line:   39,
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "safe-contact",
								Location: ast_domain.Location{
									Line:   39,
									Column: 12,
								},
								NameLocation: ast_domain.Location{
									Line:   39,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   39,
									Column: 26,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:8:0",
									RelativeLocation: ast_domain.Location{
										Line:   39,
										Column: 26,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   39,
											Column: 26,
										},
										Literal: "First Contact: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   39,
											Column: 44,
										},
										RawExpression: "state.User.Contacts[0].Email",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   39,
																		Column: 44,
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
															Name: "User",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.User"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   39,
																		Column: 44,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   52,
																		Column: 23,
																	},
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
																TypeExpression:       typeExprFromString("models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   39,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   52,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Contacts",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]dto.Contact"),
																PackageAlias:         "dto2",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Contacts",
																ReferenceLocation: ast_domain.Location{
																	Line:   39,
																	Column: 44,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   72,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/models/user.go"),
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
															TypeExpression:       typeExprFromString("[]dto.Contact"),
															PackageAlias:         "dto2",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Contacts",
															ReferenceLocation: ast_domain.Location{
																Line:   39,
																Column: 44,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   72,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/models/user.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 0,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 21,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Optional: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.Contact"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Email",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 24,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Email",
														ReferenceLocation: ast_domain.Location{
															Line:   39,
															Column: 44,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   66,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/2/dto/contact.go"),
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
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Email",
													ReferenceLocation: ast_domain.Location{
														Line:   39,
														Column: 44,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   66,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/2/dto/contact.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Email",
												ReferenceLocation: ast_domain.Location{
													Line:   39,
													Column: 44,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   66,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/2/dto/contact.go"),
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
							Line:   41,
							Column: 5,
						},
						TagName: "p",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalPackageAlias: new("pages_main_594861c5"),
							OriginalSourcePath:   new("pages/main.pk"),
							IsStructurallyStatic: true,
						},
						Key: &ast_domain.StringLiteral{
							Value: "r.0:9",
							RelativeLocation: ast_domain.Location{
								Line:   41,
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
						Attributes: []ast_domain.HTMLAttribute{
							ast_domain.HTMLAttribute{
								Name:  "id",
								Value: "unsafe-contact",
								Location: ast_domain.Location{
									Line:   41,
									Column: 12,
								},
								NameLocation: ast_domain.Location{
									Line:   41,
									Column: 8,
								},
							},
						},
						Children: []*ast_domain.TemplateNode{
							&ast_domain.TemplateNode{
								NodeType: ast_domain.NodeText,
								Location: ast_domain.Location{
									Line:   41,
									Column: 28,
								},
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									OriginalPackageAlias: new("pages_main_594861c5"),
									OriginalSourcePath:   new("pages/main.pk"),
								},
								Key: &ast_domain.StringLiteral{
									Value: "r.0:9:0",
									RelativeLocation: ast_domain.Location{
										Line:   41,
										Column: 28,
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
								RichText: []ast_domain.TextPart{
									ast_domain.TextPart{
										IsLiteral: true,
										Location: ast_domain.Location{
											Line:   41,
											Column: 28,
										},
										Literal: "Out-of-Bounds Contact: ",
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											OriginalSourcePath: new("pages/main.pk"),
										},
									},
									ast_domain.TextPart{
										IsLiteral: false,
										Location: ast_domain.Location{
											Line:   41,
											Column: 54,
										},
										RawExpression: "state.User.Contacts[99].Phone",
										Expression: &ast_domain.MemberExpression{
											Base: &ast_domain.IndexExpression{
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
																	TypeExpression:       typeExprFromString("pages_main_594861c5.Response"),
																	PackageAlias:         "pages_main_594861c5",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/dist/pages/pages_main_594861c5",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "state",
																	ReferenceLocation: ast_domain.Location{
																		Line:   41,
																		Column: 54,
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
															Name: "User",
															RelativeLocation: ast_domain.Location{
																Line:   1,
																Column: 7,
															},
															GoAnnotations: &ast_domain.GoGeneratorAnnotation{
																ResolvedType: &ast_domain.ResolvedTypeInfo{
																	TypeExpression:       typeExprFromString("models.User"),
																	PackageAlias:         "models",
																	CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
																},
																Symbol: &ast_domain.ResolvedSymbol{
																	Name: "User",
																	ReferenceLocation: ast_domain.Location{
																		Line:   41,
																		Column: 54,
																	},
																	DeclarationLocation: ast_domain.Location{
																		Line:   52,
																		Column: 23,
																	},
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
																TypeExpression:       typeExprFromString("models.User"),
																PackageAlias:         "models",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/models",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "User",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   52,
																	Column: 23,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("dist/pages/pages_main_594861c5/generated.go"),
														},
													},
													Property: &ast_domain.Identifier{
														Name: "Contacts",
														RelativeLocation: ast_domain.Location{
															Line:   1,
															Column: 12,
														},
														GoAnnotations: &ast_domain.GoGeneratorAnnotation{
															ResolvedType: &ast_domain.ResolvedTypeInfo{
																TypeExpression:       typeExprFromString("[]dto.Contact"),
																PackageAlias:         "dto2",
																CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
															},
															Symbol: &ast_domain.ResolvedSymbol{
																Name: "Contacts",
																ReferenceLocation: ast_domain.Location{
																	Line:   41,
																	Column: 54,
																},
																DeclarationLocation: ast_domain.Location{
																	Line:   72,
																	Column: 2,
																},
															},
															BaseCodeGenVarName:  new("pageData"),
															OriginalSourcePath:  new("pages/main.pk"),
															GeneratedSourcePath: new("pkg/models/user.go"),
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
															TypeExpression:       typeExprFromString("[]dto.Contact"),
															PackageAlias:         "dto2",
															CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
														},
														Symbol: &ast_domain.ResolvedSymbol{
															Name: "Contacts",
															ReferenceLocation: ast_domain.Location{
																Line:   41,
																Column: 54,
															},
															DeclarationLocation: ast_domain.Location{
																Line:   72,
																Column: 2,
															},
														},
														BaseCodeGenVarName:  new("pageData"),
														OriginalSourcePath:  new("pages/main.pk"),
														GeneratedSourcePath: new("pkg/models/user.go"),
													},
												},
												Index: &ast_domain.IntegerLiteral{
													Value: 99,
													RelativeLocation: ast_domain.Location{
														Line:   1,
														Column: 21,
													},
													GoAnnotations: &ast_domain.GoGeneratorAnnotation{
														ResolvedType: &ast_domain.ResolvedTypeInfo{
															TypeExpression:       typeExprFromString("int64"),
															PackageAlias:         "",
															CanonicalPackagePath: "",
														},
														OriginalSourcePath: new("pages/main.pk"),
														Stringability:      1,
													},
												},
												Optional: false,
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 1,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("dto.Contact"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													BaseCodeGenVarName: new("pageData"),
												},
											},
											Property: &ast_domain.Identifier{
												Name: "Phone",
												RelativeLocation: ast_domain.Location{
													Line:   1,
													Column: 25,
												},
												GoAnnotations: &ast_domain.GoGeneratorAnnotation{
													ResolvedType: &ast_domain.ResolvedTypeInfo{
														TypeExpression:       typeExprFromString("string"),
														PackageAlias:         "dto",
														CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
													},
													Symbol: &ast_domain.ResolvedSymbol{
														Name: "Phone",
														ReferenceLocation: ast_domain.Location{
															Line:   41,
															Column: 54,
														},
														DeclarationLocation: ast_domain.Location{
															Line:   67,
															Column: 2,
														},
													},
													BaseCodeGenVarName:  new("pageData"),
													OriginalSourcePath:  new("pages/main.pk"),
													GeneratedSourcePath: new("pkg/2/dto/contact.go"),
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
													CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
												},
												Symbol: &ast_domain.ResolvedSymbol{
													Name: "Phone",
													ReferenceLocation: ast_domain.Location{
														Line:   41,
														Column: 54,
													},
													DeclarationLocation: ast_domain.Location{
														Line:   67,
														Column: 2,
													},
												},
												BaseCodeGenVarName:  new("pageData"),
												OriginalSourcePath:  new("pages/main.pk"),
												GeneratedSourcePath: new("pkg/2/dto/contact.go"),
												Stringability:       1,
											},
										},
										GoAnnotations: &ast_domain.GoGeneratorAnnotation{
											ResolvedType: &ast_domain.ResolvedTypeInfo{
												TypeExpression:       typeExprFromString("string"),
												PackageAlias:         "dto",
												CanonicalPackagePath: "testcase_054_conflicting_dto_packages/pkg/2/dto",
											},
											Symbol: &ast_domain.ResolvedSymbol{
												Name: "Phone",
												ReferenceLocation: ast_domain.Location{
													Line:   41,
													Column: 54,
												},
												DeclarationLocation: ast_domain.Location{
													Line:   67,
													Column: 2,
												},
											},
											BaseCodeGenVarName:  new("pageData"),
											OriginalSourcePath:  new("pages/main.pk"),
											GeneratedSourcePath: new("pkg/2/dto/contact.go"),
											Stringability:       1,
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
