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

package emitter_go_sql

import (
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
)

const (
	// identDBTXWrapper is the generated interface a DBTX implements when it can produce an
	// equivalently wrapped view of a different DBTX.
	identDBTXWrapper = "dbtxWrapper"

	// identWrapMethod is the method name on that interface.
	identWrapMethod = "WrapDBTX"

	// identWrapper is the Queries field holding the resolved wrapper, and the local name the
	// constructors bind it to.
	identWrapper = "wrapper"

	// identWrapHelper is the unexported method that applies the wrapper, if any.
	identWrapHelper = "wrapDBTX"

	// identInner names the DBTX being wrapped, in both the interface and the helper.
	identInner = "inner"

	// identWrapped names the result of a wrap that still has to be type-asserted back.
	identWrapped = "wrapped"

	// identAny is the parameter and result type of WrapDBTX.
	identAny = "any"

	// identOK is the comma-ok variable name.
	identOK = "ok"
)

// buildDBTXWrapperInterface constructs the dbtxWrapper interface type declaration.
//
// Returns *ast.GenDecl which is the type dbtxWrapper interface { ... } declaration.
func buildDBTXWrapperInterface() *ast.GenDecl {
	return goastutil.GenDeclType(identDBTXWrapper, &ast.InterfaceType{
		Methods: &ast.FieldList{
			List: []*ast.Field{{
				Names: []*ast.Ident{goastutil.CachedIdent(identWrapMethod)},
				Type: goastutil.FuncType(
					goastutil.FieldList(
						goastutil.Field(identInner, goastutil.CachedIdent(identAny)),
					),
					goastutil.FieldList(
						goastutil.Field("", goastutil.CachedIdent(identAny)),
					),
				),
			}},
		},
	})
}

// buildWrapDBTXMethod constructs the wrapDBTX helper method on Queries, which applies the
// resolved wrapper to a connection and falls back to the connection itself.
//
// Returns *ast.FuncDecl which is the wrapDBTX method declaration.
func buildWrapDBTXMethod() *ast.FuncDecl {
	receiver := goastutil.CachedIdent(emitter_shared.IdentQueriesReceiver)
	wrapperField := goastutil.SelectorExprFrom(receiver, identWrapper)

	return &ast.FuncDecl{
		Recv: goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentQueriesReceiver,
				goastutil.StarExpr(goastutil.CachedIdent(emitter_shared.IdentQueries))),
		),
		Name: goastutil.CachedIdent(identWrapHelper),
		Type: goastutil.FuncType(
			goastutil.FieldList(goastutil.Field(emitter_shared.IdentDB, goastutil.CachedIdent(identDBTX))),
			goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identDBTX))),
		),
		Body: goastutil.BlockStmt(
			goastutil.IfStmt(nil,
				&ast.BinaryExpr{
					X:  wrapperField,
					Op: token.EQL,
					Y:  goastutil.NilIdent(),
				},
				goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentDB))),
			),
			goastutil.DefineStmtMulti(
				[]string{identWrapped, identOK},
				goastutil.TypeAssertExpr(
					goastutil.CallExpr(
						goastutil.SelectorExprFrom(wrapperField, identWrapMethod),
						goastutil.CachedIdent(emitter_shared.IdentDB),
					),
					goastutil.CachedIdent(identDBTX),
				),
			),
			goastutil.IfStmt(nil,
				&ast.UnaryExpr{Op: token.NOT, X: goastutil.CachedIdent(identOK)},
				goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentDB))),
			),
			goastutil.ReturnStmt(goastutil.CachedIdent(identWrapped)),
		),
	}
}

// resolveWrapperStmt constructs `wrapper, _ := <source>.(dbtxWrapper)`, the one statement
// each constructor needs to remember how its connection was wrapped.
//
// Takes source (string) which is the constructor parameter to inspect.
//
// Returns *ast.AssignStmt which binds the wrapper (nil when the connection is plain).
func resolveWrapperStmt(source string) *ast.AssignStmt {
	return goastutil.DefineStmtMulti(
		[]string{identWrapper, "_"},
		goastutil.TypeAssertExpr(
			goastutil.CachedIdent(source),
			goastutil.CachedIdent(identDBTXWrapper),
		),
	)
}
