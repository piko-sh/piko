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
	"context"
	"testing"

	goast "go/ast"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestAttributeEmitter_Emit_Orchestration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node                  *ast_domain.TemplateNode
		validateFunc          func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic)
		name                  string
		expectedMinStatements int
	}{
		{
			name: "static attributes only",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "id", Value: "main"},
					{Name: "class", Value: "container"},
				},
			},
			expectedMinStatements: 2,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)

				assert.GreaterOrEqual(t, len(statements), 2)
			},
		},
		{
			name: "dynamic attributes (p-bind)",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Binds: map[string]*ast_domain.Directive{
					"id": {
						Expression: &ast_domain.Identifier{
							Name: "elementId",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								BaseCodeGenVarName: new("elementId"),
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression: cachedIdent("string"),
								},
							},
						},
					},
				},
			},
			expectedMinStatements: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)

				assert.NotEmpty(t, statements)
			},
		},
		{
			name: "boolean attribute",
			node: &ast_domain.TemplateNode{
				TagName: "input",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name: "disabled",
						Expression: &ast_domain.Identifier{
							Name: "isDisabled",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								BaseCodeGenVarName: new("isDisabled"),
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression: cachedIdent("bool"),
								},
							},
						},
					},
				},
			},
			expectedMinStatements: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)

				hasIfStmt := false
				for _, statement := range statements {
					if _, ok := statement.(*goast.IfStmt); ok {
						hasIfStmt = true
						break
					}
				}
				assert.True(t, hasIfStmt, "Boolean attribute should generate if statement")
			},
		},
		{
			name: "p-class directive",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				DirClass: &ast_domain.Directive{
					Expression: &ast_domain.StringLiteral{
						Value: "btn btn-primary",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("string"),
							},
						},
					},
				},
			},
			expectedMinStatements: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)
				assert.NotEmpty(t, statements)
			},
		},
		{
			name: "p-style directive",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				DirStyle: &ast_domain.Directive{
					Expression: &ast_domain.ObjectLiteral{
						Pairs: map[string]ast_domain.Expression{
							"color": &ast_domain.StringLiteral{
								Value: "red",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression: cachedIdent("string"),
									},
								},
							},
							"fontSize": &ast_domain.StringLiteral{
								Value: "16px",
								GoAnnotations: &ast_domain.GoGeneratorAnnotation{
									ResolvedType: &ast_domain.ResolvedTypeInfo{
										TypeExpression: cachedIdent("string"),
									},
								},
							},
						},
					},
				},
			},
			expectedMinStatements: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)
				assert.NotEmpty(t, statements)
			},
		},
		{
			name: "p-key attribute",
			node: &ast_domain.TemplateNode{
				TagName: "div",
				Key: &ast_domain.StringLiteral{
					Value: "item-123",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
			},
			expectedMinStatements: 1,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)
				assert.NotEmpty(t, statements)
			},
		},
		{
			name: "mixed attributes (integration test)",
			node: &ast_domain.TemplateNode{
				TagName: "button",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "type", Value: "submit"},
				},
				Binds: map[string]*ast_domain.Directive{
					"disabled": {
						Expression: &ast_domain.Identifier{
							Name: "isDisabled",
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								BaseCodeGenVarName: new("isDisabled"),
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression: cachedIdent("bool"),
								},
							},
						},
					},
				},
				DirClass: &ast_domain.Directive{
					Expression: &ast_domain.StringLiteral{
						Value: "btn btn-primary",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("string"),
							},
						},
					},
				},
			},
			expectedMinStatements: 3,
			validateFunc: func(t *testing.T, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
				assert.Empty(t, diagnostics)

				assert.GreaterOrEqual(t, len(statements), 3)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.resetState(context.Background())
			em.ctx = NewEmitterContext()

			nodeVar := cachedIdent("node")

			nodeEmitter := requireNodeEmitter(t, em)
			attributeEmitter := requireAttributeEmitter(t, nodeEmitter)

			statements, diagnostics := attributeEmitter.emit(nodeVar, tc.node)

			tc.validateFunc(t, statements, diagnostics)
			assert.GreaterOrEqual(t, len(statements), tc.expectedMinStatements)
		})
	}
}
