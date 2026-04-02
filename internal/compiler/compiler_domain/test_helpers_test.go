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

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/js_ast"
)

func mustParseExpr(t *testing.T, src string) ast_domain.Expression {
	t.Helper()
	ctx := context.Background()
	p := ast_domain.NewExpressionParser(ctx, src, "test")
	expression, diagnostics := p.ParseExpression(ctx)
	for _, d := range diagnostics {
		if d.Severity == ast_domain.Error {
			t.Fatalf("mustParseExpr(%q): parse error: %s", src, d.Message)
		}
	}
	require.NotNil(t, expression, "mustParseExpr(%q): got nil expression", src)
	return expression
}

func mustParseJS(t *testing.T, src string) (*js_ast.AST, *RegistryContext) {
	t.Helper()
	parser := NewTypeScriptParser()
	result, err := parser.ParseTypeScript(src, "test.ts")
	require.NoError(t, err, "mustParseJS: parse failed")
	require.NotNil(t, result, "mustParseJS: nil result")

	registry := NewRegistryContext()
	return result, registry
}
