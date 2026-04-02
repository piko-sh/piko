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

package compiler_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func TestInsertStaticCSS(t *testing.T) {
	ctx := context.Background()

	t.Run("empty CSS is a no-op", func(t *testing.T) {
		tree := &js_ast.AST{}
		err := InsertStaticCSS(ctx, tree, "", "MyElement")
		assert.NoError(t, err)
	})

	t.Run("inserts CSS into class", func(t *testing.T) {
		statement, err := parseSnippetAsStatement(`class TestElement extends PPElement {}`)
		require.NoError(t, err)
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}},
		}

		err = InsertStaticCSS(ctx, tree, ".foo { color: red; }", "TestElement")
		require.NoError(t, err)

		classDecl := findClassDeclarationByName(tree, "TestElement")
		require.NotNil(t, classDecl)
		assert.Greater(t, len(classDecl.Properties), 0, "expected CSS getter to be added")
	})

	t.Run("returns error when class not found", func(t *testing.T) {
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{}}},
		}

		err := InsertStaticCSS(ctx, tree, ".foo { color: red; }", "NonExistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("handles valid CSS with multiple rules", func(t *testing.T) {
		statement, err := parseSnippetAsStatement(`class MultiElement extends PPElement {}`)
		require.NoError(t, err)
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}},
		}

		err = InsertStaticCSS(ctx, tree, ".a { color: red; } .b { font-size: 14px; }", "MultiElement")
		require.NoError(t, err)
	})
}

func TestInsertGetterIntoClass(t *testing.T) {
	ctx := context.Background()

	t.Run("adds getter to existing class", func(t *testing.T) {
		statement, err := parseSnippetAsStatement(`class GetterTest extends PPElement {}`)
		require.NoError(t, err)
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{statement}}},
		}

		getter, err := createStaticGetterFunction("css", ".test{color:red}")
		require.NoError(t, err)
		require.NotNil(t, getter)

		_, span, _ := log.Span(ctx, "test")
		defer span.End()
		err = insertGetterIntoClass(ctx, span, tree, "GetterTest", getter)
		assert.NoError(t, err)
	})

	t.Run("returns error for missing class", func(t *testing.T) {
		tree := &js_ast.AST{
			Parts: []js_ast.Part{{Stmts: []js_ast.Stmt{}}},
		}

		getter, err := createStaticGetterFunction("css", ".test{color:red}")
		require.NoError(t, err)

		_, span, _ := log.Span(ctx, "test")
		defer span.End()
		err = insertGetterIntoClass(ctx, span, tree, "Missing", getter)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
