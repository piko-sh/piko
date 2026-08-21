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
	"piko.sh/piko/internal/esbuild/helpers"
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

type mockCSSPreProcessor struct {
	err         error
	result      string
	gotCSS      string
	gotSource   string
	gotLocation ast_domain.Location
	called      bool
}

func (m *mockCSSPreProcessor) InlineImports(_ context.Context, cssContent string, sourcePath string,
	startLocation ast_domain.Location,
) (string, error) {
	m.called = true
	m.gotCSS = cssContent
	m.gotSource = sourcePath
	m.gotLocation = startLocation
	return m.result, m.err
}

func richLiteral(text string) ast_domain.TextPart {
	return ast_domain.TextPart{
		Expression:    nil,
		GoAnnotations: nil,
		Literal:       text,
		RawExpression: "",
		Location:      ast_domain.Location{},
		IsLiteral:     true,
	}
}

func richExpression(raw string) ast_domain.TextPart {
	return ast_domain.TextPart{
		Expression:    &ast_domain.Identifier{Name: raw},
		GoAnnotations: nil,
		Literal:       "",
		RawExpression: raw,
		Location:      ast_domain.Location{},
		IsLiteral:     false,
	}
}

func makeRichTextNode(parts []ast_domain.TextPart, keyVal string) *ast_domain.TemplateNode {
	return &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeText,
		RichText: parts,
		Key:      &ast_domain.StringLiteral{Value: keyVal},
	}
}

func domCallName(t *testing.T, expr js_ast.Expr) string {
	t.Helper()
	call, isCall := expr.Data.(*js_ast.ECall)
	if !isCall {
		return ""
	}
	dot, isDot := call.Target.Data.(*js_ast.EDot)
	if !isDot {
		return ""
	}
	return dot.Name
}

func domCallStringArg(t *testing.T, expr js_ast.Expr) (string, bool) {
	t.Helper()
	call, isCall := expr.Data.(*js_ast.ECall)
	if !isCall || len(call.Args) == 0 {
		return "", false
	}
	str, isStr := call.Args[0].Data.(*js_ast.EString)
	if !isStr {
		return "", false
	}
	return helpers.UTF16ToString(str.Value), true
}

func domCallArgCount(t *testing.T, expr js_ast.Expr) int {
	t.Helper()

	call, isCall := expr.Data.(*js_ast.ECall)
	require.True(t, isCall, "expected a dom call")
	return len(call.Args)
}

func writerHoldingText(text string) *ast_domain.DirectWriter {
	writer := &ast_domain.DirectWriter{}
	writer.AppendString(text)
	return writer
}
