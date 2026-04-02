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
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireEmitter(t *testing.T) *emitter {
	t.Helper()
	em, ok := NewEmitter(context.Background()).(*emitter)
	require.True(t, ok, "NewEmitter should return *emitter")
	em.resetState(context.Background())
	return em
}

func requireNodeEmitter(t *testing.T, em *emitter) *nodeEmitter {
	t.Helper()
	ne, ok := em.astBuilder.nodeEmitter.(*nodeEmitter)
	require.True(t, ok, "astBuilder.nodeEmitter should be *nodeEmitter")
	return ne
}

func requireExpressionEmitter(t *testing.T, em *emitter) *expressionEmitter {
	t.Helper()
	ee, ok := em.astBuilder.expressionEmitter.(*expressionEmitter)
	require.True(t, ok, "astBuilder.expressionEmitter should be *expressionEmitter")
	return ee
}

func requireAttributeEmitter(t *testing.T, ne *nodeEmitter) *attributeEmitter {
	t.Helper()
	ae, ok := ne.attributeEmitter.(*attributeEmitter)
	require.True(t, ok, "nodeEmitter.attributeEmitter should be *attributeEmitter")
	return ae
}

func requireIdent(t *testing.T, expression goast.Expr, message string) *goast.Ident {
	t.Helper()
	identifier, ok := expression.(*goast.Ident)
	require.True(t, ok, "%s: expected *goast.Ident, got %T", message, expression)
	return identifier
}

func requireBasicLit(t *testing.T, expression goast.Expr, message string) *goast.BasicLit {
	t.Helper()
	lit, ok := expression.(*goast.BasicLit)
	require.True(t, ok, "%s: expected *goast.BasicLit, got %T", message, expression)
	return lit
}

func requireCallExpr(t *testing.T, expression goast.Expr, message string) *goast.CallExpr {
	t.Helper()
	call, ok := expression.(*goast.CallExpr)
	require.True(t, ok, "%s: expected *goast.CallExpr, got %T", message, expression)
	return call
}

func requireFuncLit(t *testing.T, expression goast.Expr, message string) *goast.FuncLit {
	t.Helper()
	functionLiteral, ok := expression.(*goast.FuncLit)
	require.True(t, ok, "%s: expected *goast.FuncLit, got %T", message, expression)
	return functionLiteral
}

func requireSelectorExpr(t *testing.T, expression goast.Expr, message string) *goast.SelectorExpr {
	t.Helper()
	selectorExpression, ok := expression.(*goast.SelectorExpr)
	require.True(t, ok, "%s: expected *goast.SelectorExpr, got %T", message, expression)
	return selectorExpression
}

func requireCompositeLit(t *testing.T, expression goast.Expr, message string) *goast.CompositeLit {
	t.Helper()
	comp, ok := expression.(*goast.CompositeLit)
	require.True(t, ok, "%s: expected *goast.CompositeLit, got %T", message, expression)
	return comp
}

func requireKeyValueExpr(t *testing.T, expression goast.Expr, message string) *goast.KeyValueExpr {
	t.Helper()
	kv, ok := expression.(*goast.KeyValueExpr)
	require.True(t, ok, "%s: expected *goast.KeyValueExpr, got %T", message, expression)
	return kv
}

func requireIfStmt(t *testing.T, statement goast.Stmt, message string) *goast.IfStmt {
	t.Helper()
	ifStmt, ok := statement.(*goast.IfStmt)
	require.True(t, ok, "%s: expected *goast.IfStmt, got %T", message, statement)
	return ifStmt
}

func requireReturnStmt(t *testing.T, statement goast.Stmt, message string) *goast.ReturnStmt {
	t.Helper()
	ret, ok := statement.(*goast.ReturnStmt)
	require.True(t, ok, "%s: expected *goast.ReturnStmt, got %T", message, statement)
	return ret
}

func requireBlockStmt(t *testing.T, statement goast.Stmt, message string) *goast.BlockStmt {
	t.Helper()
	block, ok := statement.(*goast.BlockStmt)
	require.True(t, ok, "%s: expected *goast.BlockStmt, got %T", message, statement)
	return block
}

func requireImportSpec(t *testing.T, spec goast.Spec, message string) *goast.ImportSpec {
	t.Helper()
	imp, ok := spec.(*goast.ImportSpec)
	require.True(t, ok, "%s: expected *goast.ImportSpec, got %T", message, spec)
	return imp
}

func requireFuncDecl(t *testing.T, declaration goast.Decl, message string) *goast.FuncDecl {
	t.Helper()
	functionDeclaration, ok := declaration.(*goast.FuncDecl)
	require.True(t, ok, "%s: expected *goast.FuncDecl, got %T", message, declaration)
	return functionDeclaration
}

func requireAstImportSpec(t *testing.T, spec goast.Spec, message string) *goast.ImportSpec {
	t.Helper()
	imp, ok := spec.(*goast.ImportSpec)
	require.True(t, ok, "%s: expected *goast.ImportSpec, got %T", message, spec)
	return imp
}
