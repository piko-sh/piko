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
	"fmt"
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitPrepared generates the PreparedDBTX wrapper with eager preparation of
// static queries and lazy caching of dynamic query variants.
//
// Takes packageName (string) which is the Go package name for the generated file.
// Takes queries ([]*querier_dto.AnalysedQuery) which are the queries to prepare.
//
// Returns querier_dto.GeneratedFile which contains the prepared statement source.
// Returns error when formatting fails.
func (*SQLEmitter) EmitPrepared(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error) {
	tracker := emitter_shared.NewImportTracker()
	tracker.AddImport("context")
	tracker.AddImport("database/sql")
	tracker.AddImport("fmt")
	tracker.AddImport("sync")

	var staticConstants []string
	for _, query := range queries {
		if isStaticQuery(query) {
			staticConstants = append(staticConstants, emitter_shared.SnakeToCamelCase(query.Name))
		}
	}

	declarations := []ast.Decl{
		buildPreparedDBTXStruct(),
		buildPrepareFunction(staticConstants),
		buildGetOrPrepareMethod(),
		buildPreparedExecContext(),
		buildPreparedQueryContext(),
		buildPreparedQueryRowContext(),
		buildPreparedBeginTx(),
		buildPreparedClose(),
	}

	content, err := emitter_shared.FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("formatting prepared file: %w", err)
	}

	return querier_dto.GeneratedFile{
		Name:    "prepared.go",
		Content: content,
	}, nil
}

// isStaticQuery reports whether a query has a fixed SQL string that does not
// change at runtime. Dynamic queries (sortable ORDER BY, runtime builder)
// produce different SQL strings and are lazily cached instead.
//
// Takes query (*querier_dto.AnalysedQuery) which is the query to check.
//
// Returns bool which is true when the query SQL is fixed.
func isStaticQuery(query *querier_dto.AnalysedQuery) bool {
	if query.IsDynamic || query.DynamicRuntime {
		return false
	}
	for i := range query.Parameters {
		if query.Parameters[i].IsSlice {
			return false
		}
	}
	return true
}

// preparedReceiver returns the standard receiver field list for *PreparedDBTX.
//
// Returns *ast.FieldList which is the receiver field list.
func preparedReceiver() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(identPrepared, goastutil.StarExpr(goastutil.CachedIdent(identPreparedDBTX))),
	)
}

// buildPreparedDBTXStruct generates the PreparedDBTX struct declaration with
// db, stmts, and mu fields.
//
// Returns *ast.GenDecl which is the struct type declaration.
func buildPreparedDBTXStruct() *ast.GenDecl {
	return goastutil.GenDeclType(identPreparedDBTX, goastutil.StructType(
		goastutil.Field(emitter_shared.IdentDB, goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "DB"))),
		goastutil.Field(identPreparedStmts, stmtMapType()),
		goastutil.Field(identPreparedMu, goastutil.SelectorExpr("sync", "RWMutex")),
	))
}

// buildPrepareFunction generates the Prepare(ctx, *sql.DB) (*PreparedDBTX, error)
// constructor that eagerly prepares all static SQL queries.
//
// Takes staticConstants ([]string) which are the constant names of static queries.
//
// Returns *ast.FuncDecl which is the Prepare function declaration.
func buildPrepareFunction(staticConstants []string) *ast.FuncDecl {
	return goastutil.FuncDecl(
		"Prepare",
		goastutil.FieldList(
			goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
			goastutil.Field(emitter_shared.IdentDB, goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "DB"))),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.StarExpr(goastutil.CachedIdent(identPreparedDBTX))),
			goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
		),
		goastutil.BlockStmt(buildPrepareFunctionBody(staticConstants)...),
	)
}

// buildPrepareFunctionBody constructs the statements for the Prepare function
// body, including static query collection, preparation loop, and return.
//
// Takes staticConstants ([]string) which are the constant names of static queries.
//
// Returns []ast.Stmt which contains the function body statements.
func buildPrepareFunctionBody(staticConstants []string) []ast.Stmt {
	queryElements := make([]ast.Expr, 0, len(staticConstants))
	for _, constant := range staticConstants {
		queryElements = append(queryElements, goastutil.CachedIdent(constant))
	}

	return []ast.Stmt{
		goastutil.DefineStmt(identStaticQueries, &ast.CompositeLit{
			Type: &ast.ArrayType{Elt: goastutil.CachedIdent(emitter_shared.IdentString)},
			Elts: queryElements,
		}),
		goastutil.DefineStmt(identPreparedStmts, goastutil.CallExpr(
			goastutil.CachedIdent("make"),
			stmtMapType(),
			goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(identStaticQueries)),
		)),
		buildPrepareRangeLoop(),
		goastutil.ReturnStmt(
			goastutil.AddressExpr(goastutil.CompositeLit(
				goastutil.CachedIdent(identPreparedDBTX),
				goastutil.KeyValueIdent(emitter_shared.IdentDB, goastutil.CachedIdent(emitter_shared.IdentDB)),
				goastutil.KeyValueIdent(identPreparedStmts, goastutil.CachedIdent(identPreparedStmts)),
			)),
			goastutil.CachedIdent(emitter_shared.IdentNil),
		),
	}
}

// stmtMapType returns the AST for map[string]*sql.Stmt.
//
// Returns *ast.MapType which represents the statement map type.
func stmtMapType() *ast.MapType {
	return goastutil.MapType(
		goastutil.CachedIdent(emitter_shared.IdentString),
		goastutil.StarExpr(goastutil.SelectorExpr(identSQL, identSQLStmt)),
	)
}

// buildPrepareRangeLoop constructs the range loop that prepares each static
// query and closes all statements on error.
//
// Returns *ast.RangeStmt which is the preparation loop.
func buildPrepareRangeLoop() *ast.RangeStmt {
	return &ast.RangeStmt{
		Key:   goastutil.CachedIdent(emitter_shared.IdentBlank),
		Value: goastutil.CachedIdent(emitter_shared.IdentQuery),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(identStaticQueries),
		Body: goastutil.BlockStmt(
			goastutil.DefineStmtMulti(
				[]string{identStatement, emitter_shared.IdentErr},
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(emitter_shared.IdentDB), "PrepareContext"),
					goastutil.CachedIdent(emitter_shared.IdentCtx),
					goastutil.CachedIdent(emitter_shared.IdentQuery),
				),
			),
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  goastutil.CachedIdent(emitter_shared.IdentErr),
					Op: token.NEQ,
					Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
				},
				Body: goastutil.BlockStmt(
					buildCloseStmtsRange(identPreparedStmts),
					goastutil.ReturnStmt(
						goastutil.CachedIdent(emitter_shared.IdentNil),
						goastutil.CallExpr(
							goastutil.SelectorExpr("fmt", "Errorf"),
							&ast.BasicLit{Kind: token.STRING, Value: `"preparing statement: %w"`},
							goastutil.CachedIdent(emitter_shared.IdentErr),
						),
					),
				),
			},
			goastutil.AssignStmt(
				&ast.IndexExpr{
					X:     goastutil.CachedIdent(identPreparedStmts),
					Index: goastutil.CachedIdent(emitter_shared.IdentQuery),
				},
				goastutil.CachedIdent(identStatement),
			),
		),
	}
}

// buildCloseStmtsRange generates a for-range that closes all statements in
// a map[string]*sql.Stmt.
//
// Takes mapIdent (string) which is the identifier name of the statements map.
//
// Returns *ast.RangeStmt which is the cleanup range loop.
func buildCloseStmtsRange(mapIdent string) *ast.RangeStmt {
	return &ast.RangeStmt{
		Key:   goastutil.CachedIdent(emitter_shared.IdentBlank),
		Value: goastutil.CachedIdent("s"),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(mapIdent),
		Body: goastutil.BlockStmt(
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent("s"), "Close"),
			)},
		),
	}
}

// buildGetOrPrepareMethod generates the getOrPrepare method that checks the
// cache with a read lock, then falls back to double-check locking with
// PrepareContext for cache misses.
//
// Returns *ast.FuncDecl which is the getOrPrepare method declaration.
func buildGetOrPrepareMethod() *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: preparedReceiver(),
		Name: goastutil.CachedIdent("getOrPrepare"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
				goastutil.Field(emitter_shared.IdentQuery, goastutil.CachedIdent(emitter_shared.IdentString)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, identSQLStmt))),
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildGetOrPrepareBody()...),
	}
}

// buildGetOrPrepareBody constructs the statement list for the getOrPrepare
// method body, implementing double-check locking for statement caching.
//
// Returns []ast.Stmt which contains the method body statements.
func buildGetOrPrepareBody() []ast.Stmt {
	return []ast.Stmt{
		muMethodCall("RLock"),
		buildStmtsMapLookup(),
		muMethodCall("RUnlock"),
		&ast.IfStmt{
			Cond: goastutil.CachedIdent("ok"),
			Body: goastutil.BlockStmt(returnStatementNil()),
		},
		muMethodCall("Lock"),
		&ast.DeferStmt{Call: goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedMu),
				"Unlock",
			),
		)},
		buildDoubleCheckLookup(),
		buildPrepareContextCall(),
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: goastutil.CachedIdent(emitter_shared.IdentErr), Op: token.NEQ, Y: goastutil.CachedIdent(emitter_shared.IdentNil),
			},
			Body: goastutil.BlockStmt(
				goastutil.ReturnStmt(goastutil.CachedIdent(emitter_shared.IdentNil), goastutil.CachedIdent(emitter_shared.IdentErr)),
			),
		},
		buildStmtsMapAssign(),
		returnStatementNil(),
	}
}

// muMethodCall constructs a prepared.mu.{method}() call expression statement.
//
// Takes method (string) which is the mutex method name (e.g. "RLock", "Lock").
//
// Returns *ast.ExprStmt which is the mutex method call statement.
func muMethodCall(method string) *ast.ExprStmt {
	return &ast.ExprStmt{X: goastutil.CallExpr(
		goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedMu),
			method,
		),
	)}
}

// buildStmtsMapLookup constructs the statement, ok := prepared.stmts[query]
// map lookup assignment.
//
// Returns *ast.AssignStmt which is the map lookup statement.
func buildStmtsMapLookup() *ast.AssignStmt {
	return goastutil.DefineStmtMulti(
		[]string{identStatement, "ok"},
		&ast.IndexExpr{
			X:     goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedStmts),
			Index: goastutil.CachedIdent(emitter_shared.IdentQuery),
		},
	)
}

// buildDoubleCheckLookup constructs the second map lookup under the write lock
// for the double-check locking pattern.
//
// Returns *ast.IfStmt which is the double-check if statement.
func buildDoubleCheckLookup() *ast.IfStmt {
	return goastutil.IfStmt(
		&ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(identStatement), goastutil.CachedIdent("ok")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.IndexExpr{
				X:     goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedStmts),
				Index: goastutil.CachedIdent(emitter_shared.IdentQuery),
			}},
		},
		goastutil.CachedIdent("ok"),
		goastutil.BlockStmt(returnStatementNil()),
	)
}

// buildPrepareContextCall constructs the prepared.db.PrepareContext(ctx, query)
// call assignment for cache misses.
//
// Returns *ast.AssignStmt which is the prepare call assignment.
func buildPrepareContextCall() *ast.AssignStmt {
	return goastutil.DefineStmtMulti(
		[]string{identStatement, emitter_shared.IdentErr},
		goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), emitter_shared.IdentDB),
				"PrepareContext",
			),
			goastutil.CachedIdent(emitter_shared.IdentCtx),
			goastutil.CachedIdent(emitter_shared.IdentQuery),
		),
	)
}

// buildStmtsMapAssign constructs the prepared.stmts[query] = statement
// assignment for caching a newly prepared statement.
//
// Returns *ast.AssignStmt which is the map assignment statement.
func buildStmtsMapAssign() *ast.AssignStmt {
	return goastutil.AssignStmt(
		&ast.IndexExpr{
			X:     goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedStmts),
			Index: goastutil.CachedIdent(emitter_shared.IdentQuery),
		},
		goastutil.CachedIdent(identStatement),
	)
}

// returnStatementNil constructs a return statement, nil return expression.
//
// Returns *ast.ReturnStmt which returns the cached statement and nil error.
func returnStatementNil() *ast.ReturnStmt {
	return goastutil.ReturnStmt(goastutil.CachedIdent(identStatement), goastutil.CachedIdent(emitter_shared.IdentNil))
}

// buildPreparedExecContext generates the ExecContext method on PreparedDBTX
// that routes through prepared statements when available.
//
// Returns *ast.FuncDecl which is the ExecContext method declaration.
func buildPreparedExecContext() *ast.FuncDecl {
	return buildPreparedDBTXMethod(
		"ExecContext",
		goastutil.FieldList(
			goastutil.Field("", goastutil.SelectorExpr(identSQL, "Result")),
			goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
		),
	)
}

// buildPreparedQueryContext generates the QueryContext method on PreparedDBTX
// that routes through prepared statements when available.
//
// Returns *ast.FuncDecl which is the QueryContext method declaration.
func buildPreparedQueryContext() *ast.FuncDecl {
	return buildPreparedDBTXMethod(
		"QueryContext",
		goastutil.FieldList(
			goastutil.Field("", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "Rows"))),
			goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
		),
	)
}

// buildPreparedQueryRowContext generates the QueryRowContext method on
// PreparedDBTX that routes through prepared statements when available.
//
// Returns *ast.FuncDecl which is the QueryRowContext method declaration.
func buildPreparedQueryRowContext() *ast.FuncDecl {
	return buildPreparedDBTXMethod(
		"QueryRowContext",
		goastutil.FieldList(
			goastutil.Field("", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "Row"))),
		),
	)
}

// buildPreparedDBTXMethod generates a DBTX method on PreparedDBTX that checks
// getOrPrepare first and falls back to the underlying db on error.
//
// Takes name (string) which is the method name (e.g. "ExecContext").
// Takes results (*ast.FieldList) which defines the return types.
//
// Returns *ast.FuncDecl which is the method declaration.
func buildPreparedDBTXMethod(name string, results *ast.FieldList) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: preparedReceiver(),
		Name: goastutil.CachedIdent(name),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
				goastutil.Field(emitter_shared.IdentQuery, goastutil.CachedIdent(emitter_shared.IdentString)),
				&ast.Field{
					Names: []*ast.Ident{goastutil.CachedIdent("arguments")},
					Type:  &ast.Ellipsis{Elt: goastutil.CachedIdent("any")},
				},
			),
			Results: results,
		},
		Body: goastutil.BlockStmt(buildPreparedDBTXMethodBody(name)...),
	}
}

// buildPreparedDBTXMethodBody constructs the body statements for a DBTX method,
// including the getOrPrepare call and fallback to direct db call on error.
//
// Takes name (string) which is the method name to call on statement or db.
//
// Returns []ast.Stmt which contains the method body statements.
func buildPreparedDBTXMethodBody(name string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{identStatement, emitter_shared.IdentErr},
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), "getOrPrepare"),
				goastutil.CachedIdent(emitter_shared.IdentCtx),
				goastutil.CachedIdent(emitter_shared.IdentQuery),
			),
		),
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: goastutil.CachedIdent(emitter_shared.IdentErr), Op: token.NEQ, Y: goastutil.CachedIdent(emitter_shared.IdentNil),
			},
			Body: goastutil.BlockStmt(
				goastutil.ReturnStmt(buildVariadicCall(
					goastutil.SelectorExprFrom(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), emitter_shared.IdentDB),
						name,
					),
					goastutil.CachedIdent(emitter_shared.IdentCtx), goastutil.CachedIdent(emitter_shared.IdentQuery), goastutil.CachedIdent("arguments"),
				)),
			),
		},
		goastutil.ReturnStmt(buildVariadicCall(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identStatement), name),
			goastutil.CachedIdent(emitter_shared.IdentCtx), goastutil.CachedIdent("arguments"),
		)),
	}
}

// buildVariadicCall constructs a function call expression with the variadic
// ellipsis marker set on the last argument.
//
// Takes fun (ast.Expr) which is the function to call.
// Takes arguments ([]ast.Expr) which are the call arguments.
//
// Returns *ast.CallExpr which is the variadic call expression.
func buildVariadicCall(fun ast.Expr, arguments ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: fun, Args: arguments, Ellipsis: 1}
}

// buildPreparedBeginTx generates the BeginTx method that delegates to the
// underlying *sql.DB.
//
// Returns *ast.FuncDecl which is the BeginTx method declaration.
func buildPreparedBeginTx() *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: preparedReceiver(),
		Name: goastutil.CachedIdent("BeginTx"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(emitter_shared.IdentCtx, goastutil.SelectorExpr(emitter_shared.IdentContext, emitter_shared.IdentContextType)),
				goastutil.Field("options", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "TxOptions"))),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.StarExpr(goastutil.SelectorExpr(identSQL, "Tx"))),
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), emitter_shared.IdentDB),
						"BeginTx",
					),
					goastutil.CachedIdent(emitter_shared.IdentCtx),
					goastutil.CachedIdent("options"),
				),
			),
		),
	}
}

// buildPreparedClose generates the Close method that closes all cached prepared
// statements without closing the underlying database.
//
// Returns *ast.FuncDecl which is the Close method declaration.
func buildPreparedClose() *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: preparedReceiver(),
		Name: goastutil.CachedIdent("Close"),
		Type: &ast.FuncType{
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(emitter_shared.IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildPreparedCloseBody()...),
	}
}

// buildPreparedCloseBody constructs the body statements for the Close method,
// including locking, iterating over cached statements, and resetting the map.
//
// Returns []ast.Stmt which contains the method body statements.
func buildPreparedCloseBody() []ast.Stmt {
	return []ast.Stmt{
		muMethodCall("Lock"),
		&ast.DeferStmt{Call: goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedMu),
				"Unlock",
			),
		)},
		&ast.DeclStmt{Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent("firstError")},
				Type:  goastutil.CachedIdent(emitter_shared.IdentError),
			}},
		}},
		buildCloseRangeLoop(),
		goastutil.AssignStmt(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedStmts),
			goastutil.CallExpr(
				goastutil.CachedIdent("make"),
				stmtMapType(),
			),
		),
		goastutil.ReturnStmt(goastutil.CachedIdent("firstError")),
	}
}

// buildCloseRangeLoop constructs the range loop that closes each cached
// statement and captures the first error encountered.
//
// Returns *ast.RangeStmt which is the close loop.
func buildCloseRangeLoop() *ast.RangeStmt {
	return &ast.RangeStmt{
		Key:   goastutil.CachedIdent(emitter_shared.IdentBlank),
		Value: goastutil.CachedIdent(identStatement),
		Tok:   token.DEFINE,
		X:     goastutil.SelectorExprFrom(goastutil.CachedIdent(identPrepared), identPreparedStmts),
		Body: goastutil.BlockStmt(
			goastutil.IfStmt(
				goastutil.DefineStmt(emitter_shared.IdentErr, goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(identStatement), "Close"),
				)),
				&ast.BinaryExpr{
					X: &ast.BinaryExpr{
						X:  goastutil.CachedIdent(emitter_shared.IdentErr),
						Op: token.NEQ,
						Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
					},
					Op: token.LAND,
					Y: &ast.BinaryExpr{
						X:  goastutil.CachedIdent("firstError"),
						Op: token.EQL,
						Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
					},
				},
				goastutil.BlockStmt(
					goastutil.AssignStmt(
						goastutil.CachedIdent("firstError"),
						goastutil.CachedIdent(emitter_shared.IdentErr),
					),
				),
			),
		),
	}
}
