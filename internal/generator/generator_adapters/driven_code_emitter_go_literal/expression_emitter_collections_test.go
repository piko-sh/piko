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
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestExpressionEmitter_CollectionCall_Static(t *testing.T) {
	em := requireEmitter(t)
	em.resetState(context.Background())

	staticLiteral := &goast.CompositeLit{
		Type: &goast.ArrayType{Elt: cachedIdent("string")},
		Elts: []goast.Expr{
			&goast.BasicLit{Value: `"item1"`},
			&goast.BasicLit{Value: `"item2"`},
			&goast.BasicLit{Value: `"item3"`},
		},
	}

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "getStaticItems"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			IsCollectionCall:        true,
			StaticCollectionLiteral: staticLiteral,
		},
	}

	expressionEmitter := requireExpressionEmitter(t, em)
	goExpr, statements, diagnostics, handled := expressionEmitter.tryEmitCollectionCall(expression)

	assert.True(t, handled, "Should handle collection call")
	assert.Empty(t, diagnostics, "Should not generate diagnostics")
	assert.Empty(t, statements, "Static literal needs no prerequisite statements")
	assert.Same(t, staticLiteral, goExpr, "Should return the exact static literal")
}

func TestExpressionEmitter_CollectionCall_Dynamic(t *testing.T) {
	em := requireEmitter(t)
	em.resetState(context.Background())
	em.ctx = NewEmitterContext()

	fetcherFunc := &goast.FuncDecl{
		Name: cachedIdent("FetchUsers"),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{cachedIdent("ctx")},
						Type:  cachedIdent("context.Context"),
					},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: &goast.ArrayType{Elt: cachedIdent("User")}},
					{Type: cachedIdent("error")},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.ReturnStmt{
					Results: []goast.Expr{
						cachedIdent("nil"),
						cachedIdent("nil"),
					},
				},
			},
		},
	}

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "getUsers"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			IsCollectionCall: true,
			DynamicCollectionInfo: &collection_dto.DynamicCollectionInfo{
				FetcherCode: &collection_dto.RuntimeFetcherCode{
					FetcherFunc: fetcherFunc,
					RequiredImports: map[string]string{
						"context":      "",
						"database/sql": "sql",
					},
				},
			},
		},
	}

	expressionEmitter := requireExpressionEmitter(t, em)
	goExpr, statements, diagnostics, handled := expressionEmitter.tryEmitCollectionCall(expression)

	assert.True(t, handled, "Should handle collection call")
	assert.Empty(t, diagnostics, "Should not generate diagnostics")
	assert.Empty(t, statements, "Fetcher is added to file-level declarations, not prereq statements")

	callExpr := requireCallExpr(t, goExpr, "collection call result")

	require.Len(t, callExpr.Args, 1, "Should have one argument")
	ctxArg := requireIdent(t, callExpr.Args[0], "collection call argument")
	assert.Equal(t, "ctx", ctxArg.Name, "Should pass ctx as argument")

	assert.NotEmpty(t, em.ctx.fetcherDecls, "Fetcher should be added to file declarations")
	assert.Len(t, em.ctx.fetcherDecls, 1, "Should have exactly one fetcher")

	_, hasContext := em.ctx.requiredImports["context"]
	assert.True(t, hasContext, "Should register context import")

	sqlAlias, hasSQL := em.ctx.requiredImports["database/sql"]
	assert.True(t, hasSQL, "Should register database/sql import")
	assert.Equal(t, "sql", sqlAlias, "Should register import with correct alias")
}

func TestExpressionEmitter_CollectionCall_MissingData(t *testing.T) {
	em := requireEmitter(t)
	em.resetState(context.Background())

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "getItems"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			IsCollectionCall:        true,
			StaticCollectionLiteral: nil,
			DynamicCollectionInfo:   nil,
		},
		RelativeLocation: ast_domain.Location{Line: 42, Column: 10},
	}

	expressionEmitter := requireExpressionEmitter(t, em)
	goExpr, statements, diagnostics, handled := expressionEmitter.tryEmitCollectionCall(expression)

	assert.True(t, handled, "Should handle (even though it's an error)")
	require.Len(t, diagnostics, 1, "Should generate error diagnostic")
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "missing both StaticCollectionLiteral and DynamicCollectionInfo")

	nilIdent := requireIdent(t, goExpr, "error result")
	assert.Equal(t, "nil", nilIdent.Name, "Should return nil identifier")
	assert.Empty(t, statements, "Should not generate statements")
}

func TestExpressionEmitter_CollectionCall_NotACollection(t *testing.T) {
	em := requireEmitter(t)
	em.resetState(context.Background())

	expression := &ast_domain.CallExpression{
		Callee: &ast_domain.Identifier{Name: "regularFunction"},
		GoAnnotations: &ast_domain.GoGeneratorAnnotation{
			IsCollectionCall: false,
		},
	}

	expressionEmitter := requireExpressionEmitter(t, em)
	_, _, _, handled := expressionEmitter.tryEmitCollectionCall(expression)

	assert.False(t, handled, "Should not handle non-collection calls")
}

func TestCloneFuncDecl(t *testing.T) {
	t.Run("clones function declaration", func(t *testing.T) {
		original := &goast.FuncDecl{
			Name: cachedIdent("MyFunc"),
			Type: &goast.FuncType{
				Params:  &goast.FieldList{},
				Results: &goast.FieldList{},
			},
			Body: &goast.BlockStmt{
				List: []goast.Stmt{
					&goast.ReturnStmt{},
				},
			},
		}

		cloned := cloneFuncDecl(original)

		assert.NotSame(t, original, cloned, "Should create a new instance")
		assert.Equal(t, "MyFunc", cloned.Name.Name, "Should preserve name")
		assert.NotNil(t, cloned.Type, "Should preserve type")
		assert.NotNil(t, cloned.Body, "Should preserve body")
	})

	t.Run("returns nil for nil input", func(t *testing.T) {
		cloned := cloneFuncDecl(nil)
		assert.Nil(t, cloned, "Should return nil for nil input")
	})
}

func TestValidateHybridCollectionInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ann          *ast_domain.GoGeneratorAnnotation
		name         string
		diagContains string
		expectInfo   bool
		expectDiag   bool
	}{
		{
			name: "nil DynamicCollectionInfo returns diagnostic",
			ann: &ast_domain.GoGeneratorAnnotation{
				DynamicCollectionInfo: nil,
			},
			expectInfo: false,
			expectDiag: true,
		},
		{
			name: "wrong type returns diagnostic with wrong type message",
			ann: &ast_domain.GoGeneratorAnnotation{
				DynamicCollectionInfo: "not the right type",
			},
			expectInfo:   false,
			expectDiag:   true,
			diagContains: "wrong type",
		},
		{
			name: "valid DynamicCollectionInfo returns info without diagnostic",
			ann: &ast_domain.GoGeneratorAnnotation{
				DynamicCollectionInfo: &collection_dto.DynamicCollectionInfo{
					ProviderName:   "testProvider",
					CollectionName: "testCollection",
				},
			},
			expectInfo: true,
			expectDiag: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			info, diagnostic := validateHybridCollectionInfo(tc.ann)

			if tc.expectInfo {
				assert.NotNil(t, info, "Expected non-nil info")
			} else {
				assert.Nil(t, info, "Expected nil info")
			}

			if tc.expectDiag {
				require.NotNil(t, diagnostic, "Expected non-nil diagnostic")
				assert.Equal(t, ast_domain.Error, diagnostic.Severity)
				if tc.diagContains != "" {
					assert.Contains(t, diagnostic.Message, tc.diagContains)
				}
			} else {
				assert.Nil(t, diagnostic, "Expected nil diagnostic")
			}
		})
	}
}

func TestHandleNonHybridMode(t *testing.T) {
	t.Parallel()

	staticLiteral := &goast.CompositeLit{
		Type: &goast.ArrayType{Elt: cachedIdent("string")},
		Elts: []goast.Expr{
			&goast.BasicLit{Value: `"a"`},
		},
	}

	testCases := []struct {
		ann           *ast_domain.GoGeneratorAnnotation
		name          string
		expectLiteral bool
	}{
		{
			name: "with StaticCollectionLiteral returns literal and warning",
			ann: &ast_domain.GoGeneratorAnnotation{
				StaticCollectionLiteral: staticLiteral,
			},
			expectLiteral: true,
		},
		{
			name: "without StaticCollectionLiteral returns nil ident and warning",
			ann: &ast_domain.GoGeneratorAnnotation{
				StaticCollectionLiteral: nil,
			},
			expectLiteral: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression, statements, diagnostics := handleNonHybridMode(tc.ann)

			assert.Nil(t, statements, "Should not produce any statements")
			require.Len(t, diagnostics, 1, "Should produce exactly one diagnostic")
			assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
			assert.Contains(t, diagnostics[0].Message, "HybridMode is false")

			if tc.expectLiteral {
				assert.Same(t, staticLiteral, expression, "Should return the static literal")
			} else {
				identifier := requireIdent(t, expression, "nil fallback")
				assert.Equal(t, "nil", identifier.Name, "Should return nil identifier")
			}
		})
	}
}

func TestHybridMissingLiteralDiagnostic(t *testing.T) {
	t.Parallel()

	info := &collection_dto.DynamicCollectionInfo{
		ProviderName:   "myProvider",
		CollectionName: "myCollection",
	}

	diagnostic := hybridMissingLiteralDiagnostic(info)

	require.NotNil(t, diagnostic, "Should return a diagnostic")
	assert.Equal(t, ast_domain.Error, diagnostic.Severity)
	assert.Contains(t, diagnostic.Message, "myProvider")
	assert.Contains(t, diagnostic.Message, "myCollection")
}

func TestEmitHybridCollectionFetcher(t *testing.T) {
	t.Parallel()

	staticLiteral := &goast.CompositeLit{
		Type: &goast.ArrayType{Elt: cachedIdent("int")},
		Elts: []goast.Expr{
			&goast.BasicLit{Value: "1"},
		},
	}

	targetType := cachedIdent("Post")

	testCases := []struct {
		ann            *ast_domain.GoGeneratorAnnotation
		name           string
		diagSeverity   ast_domain.Severity
		expectNilExpr  bool
		expectLiteral  bool
		expectCallExpr bool
		expectDiag     bool
	}{
		{
			name: "nil DynamicCollectionInfo returns nil expr and error diagnostic",
			ann: &ast_domain.GoGeneratorAnnotation{
				IsHybridCollection:    true,
				DynamicCollectionInfo: nil,
			},
			expectNilExpr: true,
			expectDiag:    true,
			diagSeverity:  ast_domain.Error,
		},
		{
			name: "non-hybrid mode with literal returns literal and warning",
			ann: &ast_domain.GoGeneratorAnnotation{
				IsHybridCollection: true,
				DynamicCollectionInfo: &collection_dto.DynamicCollectionInfo{
					HybridMode: false,
				},
				StaticCollectionLiteral: staticLiteral,
			},
			expectLiteral: true,
			expectDiag:    true,
			diagSeverity:  ast_domain.Warning,
		},
		{
			name: "non-hybrid mode without literal returns nil and warning",
			ann: &ast_domain.GoGeneratorAnnotation{
				IsHybridCollection: true,
				DynamicCollectionInfo: &collection_dto.DynamicCollectionInfo{
					HybridMode: false,
				},
				StaticCollectionLiteral: nil,
			},
			expectNilExpr: true,
			expectDiag:    true,
			diagSeverity:  ast_domain.Warning,
		},
		{
			name: "hybrid mode generates getter call expression",
			ann: &ast_domain.GoGeneratorAnnotation{
				IsHybridCollection: true,
				DynamicCollectionInfo: &collection_dto.DynamicCollectionInfo{
					HybridMode:     true,
					ProviderName:   "markdown",
					CollectionName: "blog",
					TargetType:     targetType,
				},
				StaticCollectionLiteral: staticLiteral,
			},
			expectCallExpr: true,
			expectDiag:     false,
		},
		{
			name: "hybrid mode without target type returns nil and error",
			ann: &ast_domain.GoGeneratorAnnotation{
				IsHybridCollection: true,
				DynamicCollectionInfo: &collection_dto.DynamicCollectionInfo{
					HybridMode:     true,
					ProviderName:   "testProvider",
					CollectionName: "testCollection",
					TargetType:     nil,
				},
				StaticCollectionLiteral: nil,
			},
			expectNilExpr: true,
			expectDiag:    true,
			diagSeverity:  ast_domain.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.ctx = NewEmitterContext()
			expressionEmitter := requireExpressionEmitter(t, em)

			expression, statements, diagnostics := expressionEmitter.emitHybridCollectionFetcher(tc.ann)

			assert.Nil(t, statements, "Should not produce any statements")

			if tc.expectDiag {
				require.Len(t, diagnostics, 1, "Should produce exactly one diagnostic")
				assert.Equal(t, tc.diagSeverity, diagnostics[0].Severity)
			} else {
				assert.Empty(t, diagnostics, "Should not produce diagnostics")
			}

			if tc.expectCallExpr {
				callExpr, ok := expression.(*goast.CallExpr)
				require.True(t, ok, "Should return a call expression, got %T", expression)
				assert.NotNil(t, callExpr.Fun, "Call expression should have a function")
				assert.Len(t, callExpr.Args, 1, "Should pass ctx as argument")
			} else if tc.expectLiteral {
				assert.Same(t, staticLiteral, expression, "Should return the static literal")
			} else if tc.expectNilExpr {
				identifier := requireIdent(t, expression, "nil fallback")
				assert.Equal(t, "nil", identifier.Name, "Should return nil identifier")
			}
		})
	}
}
