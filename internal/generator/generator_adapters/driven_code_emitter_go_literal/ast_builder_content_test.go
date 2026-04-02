// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package driven_code_emitter_go_literal

import (
	goast "go/ast"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
)

func TestExtractPropsTypeFromComponent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		component           *annotator_dto.VirtualComponent
		name                string
		expectNoPropsType   bool
		expectHasRenderFunc bool
	}{
		{
			name: "component with nil script AST",
			component: &annotator_dto.VirtualComponent{
				RewrittenScriptAST: nil,
			},
			expectNoPropsType: true,
		},
		{
			name: "component with no Render function",
			component: &annotator_dto.VirtualComponent{
				RewrittenScriptAST: &goast.File{
					Decls: []goast.Decl{
						&goast.FuncDecl{
							Name: cachedIdent("SomeOtherFunction"),
						},
					},
				},
			},
			expectNoPropsType: true,
		},
		{
			name: "component with Render function",
			component: &annotator_dto.VirtualComponent{
				RewrittenScriptAST: &goast.File{
					Decls: []goast.Decl{
						&goast.FuncDecl{
							Name: cachedIdent("Render"),
							Type: &goast.FuncType{
								Params: &goast.FieldList{
									List: []*goast.Field{
										{Type: cachedIdent("RenderContext")},
										{Type: cachedIdent("MyProps")},
									},
								},
							},
						},
					},
				},
			},
			expectHasRenderFunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propsTypeExpr, propsVarInit := extractPropsTypeFromComponent(tt.component)

			if propsTypeExpr == nil {
				t.Error("Expected non-nil propsTypeExpr")
			}

			if propsVarInit == nil {
				t.Error("Expected non-nil propsVarInit")
			}

			if tt.expectNoPropsType {

				if selectorExpression, ok := propsTypeExpr.(*goast.SelectorExpr); ok {
					if selectorExpression.Sel.Name != NoPropsTypeName {
						t.Errorf("Expected NoProps type, got: %s", selectorExpression.Sel.Name)
					}
				}
			}
		})
	}
}

func TestExtractPropsTypeFromRenderFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		renderFunc        *goast.FuncDecl
		name              string
		expectNoPropsType bool
	}{
		{
			name: "Render with nil params",
			renderFunc: &goast.FuncDecl{
				Name: cachedIdent("Render"),
				Type: &goast.FuncType{
					Params: nil,
				},
			},
			expectNoPropsType: true,
		},
		{
			name: "Render with only one param",
			renderFunc: &goast.FuncDecl{
				Name: cachedIdent("Render"),
				Type: &goast.FuncType{
					Params: &goast.FieldList{
						List: []*goast.Field{
							{Type: cachedIdent("RenderContext")},
						},
					},
				},
			},
			expectNoPropsType: true,
		},
		{
			name: "Render with custom props",
			renderFunc: &goast.FuncDecl{
				Name: cachedIdent("Render"),
				Type: &goast.FuncType{
					Params: &goast.FieldList{
						List: []*goast.Field{
							{Type: cachedIdent("RenderContext")},
							{Type: cachedIdent("CustomProps")},
						},
					},
				},
			},
			expectNoPropsType: false,
		},
		{
			name: "Render with NoProps type",
			renderFunc: &goast.FuncDecl{
				Name: cachedIdent("Render"),
				Type: &goast.FuncType{
					Params: &goast.FieldList{
						List: []*goast.Field{
							{Type: cachedIdent("RenderContext")},
							{Type: &goast.SelectorExpr{
								X:   cachedIdent(facadePackageName),
								Sel: cachedIdent(NoPropsTypeName),
							}},
						},
					},
				},
			},
			expectNoPropsType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			propsTypeExpr, propsVarInit := extractPropsTypeFromRenderFunction(tt.renderFunc)

			if propsTypeExpr == nil {
				t.Error("Expected non-nil propsTypeExpr")
			}

			if propsVarInit == nil {
				t.Error("Expected non-nil propsVarInit")
			}

			if _, ok := propsVarInit.(*goast.CompositeLit); !ok {
				t.Errorf("Expected *goast.CompositeLit, got %T", propsVarInit)
			}
		})
	}
}

func TestGenerateRuntimeContentFetch(t *testing.T) {
	t.Parallel()

	parentSliceExpr := &goast.SelectorExpr{
		X:   cachedIdent("tempVar1"),
		Sel: cachedIdent("Children"),
	}

	em := &emitter{}
	b := &astBuilder{emitter: em}

	statements := b.generateRuntimeContentFetch(parentSliceExpr)

	if len(statements) != 1 {
		t.Errorf("Expected 1 statement, got %d", len(statements))
		return
	}

	ifStmt, ok := statements[0].(*goast.IfStmt)
	if !ok {
		t.Errorf("Expected *goast.IfStmt, got %T", statements[0])
		return
	}

	if ifStmt.Init == nil {
		t.Error("Expected if statement to have Init")
		return
	}

	assignStmt, ok := ifStmt.Init.(*goast.AssignStmt)
	if !ok {
		t.Errorf("Expected Init to be *goast.AssignStmt, got %T", ifStmt.Init)
		return
	}

	if len(assignStmt.Lhs) != 1 {
		t.Error("Expected 1 LHS in assignment")
		return
	}
	identifier, ok := assignStmt.Lhs[0].(*goast.Ident)
	if !ok || identifier.Name != "contentAST" {
		t.Errorf("Expected LHS to be 'contentAST', got %v", assignStmt.Lhs[0])
	}

	if ifStmt.Body == nil || len(ifStmt.Body.List) == 0 {
		t.Error("Expected if body to have statements")
		return
	}

	forStmt, ok := ifStmt.Body.List[0].(*goast.RangeStmt)
	if !ok {
		t.Errorf("Expected for range statement in body, got %T", ifStmt.Body.List[0])
	} else {

		if forStmt.Value == nil {
			t.Error("Expected Value in range statement")
		} else if identifier, ok := forStmt.Value.(*goast.Ident); !ok || identifier.Name != "contentNode" {
			t.Errorf("Expected Value to be 'contentNode', got %v", forStmt.Value)
		}
	}
}

func TestBuildLocalStoreStatement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		result    *annotator_dto.AnnotationResult
		name      string
		request   generator_dto.GenerateRequest
		expectNil bool
	}{
		{
			name: "nil virtual module",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: nil,
			},
			request:   generator_dto.GenerateRequest{SourcePath: "/test/path.pk"},
			expectNil: true,
		},
		{
			name: "nil graph",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					Graph: nil,
				},
			},
			request:   generator_dto.GenerateRequest{SourcePath: "/test/path.pk"},
			expectNil: true,
		},
		{
			name: "missing component",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/test/path.pk": "test_hash",
						},
					},
				},
			},
			request:   generator_dto.GenerateRequest{SourcePath: "/test/path.pk"},
			expectNil: true,
		},
		{
			name: "nil local translations",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"test_hash": {
							Source: &annotator_dto.ParsedComponent{
								LocalTranslations: nil,
							},
						},
					},
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/test/path.pk": "test_hash",
						},
					},
				},
			},
			request:   generator_dto.GenerateRequest{SourcePath: "/test/path.pk"},
			expectNil: true,
		},
		{
			name: "empty local translations",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"test_hash": {
							Source: &annotator_dto.ParsedComponent{
								LocalTranslations: make(i18n_domain.Translations),
							},
						},
					},
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/test/path.pk": "test_hash",
						},
					},
				},
			},
			request:   generator_dto.GenerateRequest{SourcePath: "/test/path.pk"},
			expectNil: true,
		},
		{
			name: "valid local translations",
			result: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"test_hash": {
							Source: &annotator_dto.ParsedComponent{
								LocalTranslations: i18n_domain.Translations{
									"en": {
										"hello": "Hello",
										"world": "World",
									},
								},
							},
						},
					},
					Graph: &annotator_dto.ComponentGraph{
						PathToHashedName: map[string]string{
							"/test/path.pk": "test_hash",
						},
					},
				},
			},
			request:   generator_dto.GenerateRequest{SourcePath: "/test/path.pk"},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := &astBuilder{}

			statement := b.buildLocalStoreStatement(tt.request, tt.result)

			if tt.expectNil {
				if statement != nil {
					t.Error("Expected nil statement")
				}
			} else {
				if statement == nil {
					t.Error("Expected non-nil statement")
				}

				expressionStatement, ok := statement.(*goast.ExprStmt)
				if !ok {
					t.Fatalf("Expected *goast.ExprStmt, got %T", statement)
				}

				callExpr, ok := expressionStatement.X.(*goast.CallExpr)
				if !ok {
					t.Fatalf("Expected *goast.CallExpr, got %T", expressionStatement.X)
				}

				selExpr, ok := callExpr.Fun.(*goast.SelectorExpr)
				if !ok {
					t.Fatalf("Expected *goast.SelectorExpr, got %T", callExpr.Fun)
				}

				if selExpr.Sel.Name != "SetLocalStoreFromMap" {
					t.Errorf("Expected SetLocalStoreFromMap call, got: %s", selExpr.Sel.Name)
				}
			}
		})
	}
}
