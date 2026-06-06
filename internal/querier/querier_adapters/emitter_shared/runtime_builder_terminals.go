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

package emitter_shared

import (
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
)

// buildBuilderBuildQueryMethod constructs the buildQuery() method that assembles the
// final SQL string and returns the positional argument snapshot for the call.
//
// The baseHasWhere flag is baked in at codegen time so a query whose static SQL already
// contains a WHERE clause (e.g. a tenant-scoping `WHERE environment_id = $1`) appends
// runtime predicates with " AND " instead of a duplicate " WHERE ".
//
// buildQuery does not mutate the builder. The whereArgs field, the parameterCount
// counter, and the limit/offset values stay intact so the same builder can be reused
// (notably, .All(ctx) followed by .Count(ctx) must not double-bind LIMIT/OFFSET to the
// count query). LIMIT and OFFSET values are appended to a local `args` slice (seeded from
// builder.whereArgs); the placeholder index uses a local `parameterCount` counter seeded
// from builder.parameterCount.
//
// buildQuery returns the builder.pendingError before assembling anything when a prior
// chainable call recorded one (an oversized IN / NOT IN list), so the All / One terminals
// surface the wrapped errPikoTooManyBindVariables sentinel rather than issuing an
// under-bound statement.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes baseHasWhere (bool) which is true when the .sql file's SELECT already includes a
// WHERE clause.
//
// Returns *ast.FuncDecl which is the buildQuery method declaration.
func buildBuilderBuildQueryMethod(builderTypeName string, baseHasWhere bool, strategy MethodStrategy) *ast.FuncDecl {
	useNumbered := strategy == nil || strategy.RuntimeBuilderUsesNumberedPlaceholders()
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("buildQuery"),
		Type: &ast.FuncType{
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(IdentString)),
				goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)}),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(
			buildPendingErrorGuard(
				goastutil.StrLit(""),
				&ast.CompositeLit{Type: &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)}},
			),
			goastutil.DefineStmt(IdentQuery,
				builderField("baseSQL"),
			),
			goastutil.DefineStmt(IdentArgs,
				&ast.CallExpr{
					Fun: goastutil.CachedIdent("append"),
					Args: []ast.Expr{
						&ast.CompositeLit{
							Type: &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
						},
						builderField(IdentWhereArgs),
					},
					Ellipsis: 1,
				},
			),
			goastutil.DefineStmt(IdentParameterCount,
				builderField(IdentParameterCount),
			),
			buildWhereClauseBlock(baseHasWhere),
			buildOrderByClauseBlock(),
			buildQueryParameterAppendBlock("limitValue", " LIMIT ", useNumbered),
			buildQueryParameterAppendBlock("offsetValue", " OFFSET ", useNumbered),
			goastutil.ReturnStmt(
				goastutil.CachedIdent(IdentQuery),
				goastutil.CachedIdent(IdentArgs),
				goastutil.CachedIdent(IdentNil),
			),
		),
	}
}

// buildPendingErrorGuard constructs the leading guard shared by buildQuery and
// buildCountQuery that returns builder.pendingError (set by a prior chainable call such
// as an oversized IN / NOT IN list) before any SQL assembly happens. The zeroValues are
// returned alongside the error so the guard matches the enclosing method's result
// signature.
//
// Takes zeroValues ([]ast.Expr) which are the zero values returned ahead of the pending
// error.
//
// Returns *ast.IfStmt which is the pending-error guard statement.
func buildPendingErrorGuard(zeroValues ...ast.Expr) *ast.IfStmt {
	returnValues := make([]ast.Expr, 0, len(zeroValues)+1)
	returnValues = append(returnValues, zeroValues...)
	returnValues = append(returnValues, builderField(identPendingError))

	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  builderField(identPendingError),
			Op: token.NEQ,
			Y:  goastutil.CachedIdent(IdentNil),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(returnValues...),
		),
	}
}

// buildWhereClauseBlock constructs the if-block that appends joined WHERE clauses. When
// baseHasWhere is true the appended fragment is prefixed with " AND " instead of " WHERE
// " so the merged SQL stays syntactically valid when the base SELECT already filters.
//
// Takes baseHasWhere (bool) which is true when the base SELECT already includes a WHERE
// clause.
//
// Returns *ast.IfStmt which is the WHERE clause if-block.
func buildWhereClauseBlock(baseHasWhere bool) *ast.IfStmt {
	prefix := " WHERE "
	if baseHasWhere {
		prefix = " AND "
	}
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: goastutil.CallExpr(
				goastutil.CachedIdent("len"),
				builderField(IdentWhereClauses),
			),
			Op: token.GTR,
			Y:  goastutil.IntLit(0),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentQuery)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{
					&ast.BinaryExpr{
						X:  goastutil.StrLit(prefix),
						Op: token.ADD,
						Y: goastutil.CallExpr(
							goastutil.SelectorExpr("strings", "Join"),
							builderField(IdentWhereClauses),
							goastutil.StrLit(" AND "),
						),
					},
				},
			},
		),
	}
}

// buildOrderByClauseBlock constructs the if-block that joins every accumulated ORDER BY
// fragment with ", " and appends the result to the query. Mirrors the WHERE-clause join
// pattern below so single-key and multi-key callers share one emitted code path.
//
// Returns *ast.IfStmt which is the ORDER BY clause if-block.
func buildOrderByClauseBlock() *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: goastutil.CallExpr(
				goastutil.CachedIdent("len"),
				builderField(IdentOrderByClauses),
			),
			Op: token.GTR,
			Y:  goastutil.IntLit(0),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentQuery)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{
					&ast.BinaryExpr{
						X:  goastutil.StrLit(" ORDER BY "),
						Op: token.ADD,
						Y: goastutil.CallExpr(
							goastutil.SelectorExpr("strings", "Join"),
							builderField(IdentOrderByClauses),
							goastutil.StrLit(", "),
						),
					},
				},
			},
		),
	}
}

// buildQueryParameterAppendBlock constructs the if-block that appends a LIMIT or OFFSET
// clause with a bind placeholder. The bumped counter and the appended value land on the
// local `parameterCount` and `args` declared at the top of buildQuery so the builder's
// persisted state stays untouched across repeated calls.
//
// Engines whose driver expects numbered placeholders concatenate the counter after the
// SQL fragment (`LIMIT $1`); engines that bind positionally emit a bare marker (`LIMIT
// ?`) without consulting the counter.
//
// Takes fieldName (string) which is the builder field holding the value.
// Takes sqlClause (string) which is the SQL fragment prefix (e.g. " LIMIT ").
// Takes useNumberedPlaceholders (bool) which selects between `$N` (true) and bare `?`
// (false) for the appended placeholder.
//
// Returns *ast.IfStmt which is the parameter append if-block.
func buildQueryParameterAppendBlock(fieldName string, sqlClause string, useNumberedPlaceholders bool) *ast.IfStmt {
	var clauseExpression ast.Expr
	if useNumberedPlaceholders {
		clauseExpression = &ast.BinaryExpr{
			X:  goastutil.StrLit(sqlClause + "$"),
			Op: token.ADD,
			Y: goastutil.CallExpr(
				goastutil.SelectorExpr("strconv", "Itoa"),
				goastutil.CachedIdent(IdentParameterCount),
			),
		}
	} else {
		clauseExpression = goastutil.StrLit(sqlClause + "?")
	}
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  builderField(fieldName),
			Op: token.GTR,
			Y:  goastutil.IntLit(0),
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentParameterCount)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{goastutil.IntLit(1)},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentQuery)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{clauseExpression},
			},
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentArgs)},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					goastutil.CallExpr(
						goastutil.CachedIdent("append"),
						goastutil.CachedIdent(IdentArgs),
						builderField(fieldName),
					),
				},
			},
		),
	}
}

// buildBuilderAllMethod constructs the All(ctx) terminal method.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the All method declaration.
func buildBuilderAllMethod(
	builderTypeName string,
	rowTypeName string,
	scanArguments []ast.Expr,
	strategy MethodStrategy,
) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("All"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(IdentCtx, goastutil.SelectorExpr(IdentContext, IdentContextType)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildBuilderAllBody(rowTypeName, scanArguments, strategy)...),
	}
}

// buildBuilderAllBody constructs the statement list for the All method body.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Stmt which contains the All method body statements.
func buildBuilderAllBody(rowTypeName string, scanArguments []ast.Expr, strategy MethodStrategy) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{IdentQuery, IdentArgs, IdentErr},
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), "buildQuery"),
			),
		),
		BuildErrCheck(goastutil.CachedIdent(IdentNil)),
		goastutil.DefineStmtMulti(
			[]string{IdentRows, IdentErr},
			strategy.BuilderQueryCall(goastutil.CachedIdent(IdentArgs)),
		),
		BuildErrCheck(goastutil.CachedIdent(IdentNil)),
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Close"),
			),
		},
		goastutil.VarDecl(IdentItems, &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
		buildBuilderScanLoop(rowTypeName, scanArguments),
		BuildRowsErrCheck(),
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentItems), goastutil.CachedIdent(IdentNil)),
	}
}

// buildBuilderScanLoop constructs the for-loop that iterates over rows, scans each into a
// row struct, and appends to the items slice.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
//
// Returns *ast.ForStmt which is the scan loop statement.
func buildBuilderScanLoop(rowTypeName string, scanArguments []ast.Expr) *ast.ForStmt {
	return &ast.ForStmt{
		Cond: goastutil.CallExpr(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Next"),
		),
		Body: goastutil.BlockStmt(
			goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)),
			goastutil.IfStmt(
				goastutil.DefineStmt(IdentErr,
					goastutil.CallExpr(
						goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Scan"),
						scanArguments...,
					),
				),
				&ast.BinaryExpr{
					X:  goastutil.CachedIdent(IdentErr),
					Op: token.NEQ,
					Y:  goastutil.CachedIdent(IdentNil),
				},
				goastutil.BlockStmt(
					goastutil.ReturnStmt(goastutil.CachedIdent(IdentNil), goastutil.CachedIdent(IdentErr)),
				),
			),
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(IdentItems)},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					goastutil.CallExpr(
						goastutil.CachedIdent("append"),
						goastutil.CachedIdent(IdentItems),
						goastutil.CachedIdent(IdentRow),
					),
				},
			},
		),
	}
}

// buildBuilderOneMethod constructs the One(ctx) terminal method.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the One method declaration.
func buildBuilderOneMethod(
	builderTypeName string,
	rowTypeName string,
	scanArguments []ast.Expr,
	strategy MethodStrategy,
) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("One"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(IdentCtx, goastutil.SelectorExpr(IdentContext, IdentContextType)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(rowTypeName)),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(buildBuilderOneBody(rowTypeName, scanArguments, strategy)...),
	}
}

// buildBuilderOneBody constructs the statement list for the One method body.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Stmt which contains the One method body statements.
func buildBuilderOneBody(rowTypeName string, scanArguments []ast.Expr, strategy MethodStrategy) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{IdentQuery, IdentArgs, IdentErr},
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), "buildQuery"),
			),
		),
		BuildErrCheck(goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName))),
		goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)),
		goastutil.AssignStmt(
			goastutil.CachedIdent(IdentErr),
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(
					strategy.BuilderQueryRowCall(goastutil.CachedIdent(IdentArgs)),
					"Scan",
				),
				scanArguments...,
			),
		),
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentRow), goastutil.CachedIdent(IdentErr)),
	}
}

// buildBuilderBuildCountQueryMethod constructs the private buildCountQuery() helper that
// assembles the COUNT(*) form of the query at runtime.
//
// It starts from the pre-derived `<query>CountSQL` package const and appends accumulated
// WHERE fragments using the same WHERE/AND switch as buildQuery(). ORDER BY, LIMIT, and
// OFFSET are omitted because they cannot affect the row count.
//
// buildCountQuery returns the builder.pendingError before assembling the count SQL when a
// prior chainable call recorded one (an oversized IN / NOT IN list) so the Count terminal
// surfaces the wrapped errPikoTooManyBindVariables sentinel rather than issuing an
// under-bound statement.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes countSQLConstName (string) which is the package const carrying the pre-derived
// count SQL.
// Takes baseHasWhere (bool) which is true when the .sql file's SELECT already includes a
// WHERE clause (so we use AND instead of WHERE).
//
// Returns *ast.FuncDecl which is the buildCountQuery method declaration.
func buildBuilderBuildCountQueryMethod(builderTypeName string, countSQLConstName string, baseHasWhere bool) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("buildCountQuery"),
		Type: &ast.FuncType{
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(IdentString)),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(
			buildPendingErrorGuard(goastutil.StrLit("")),
			goastutil.DefineStmt(IdentQuery,
				goastutil.CachedIdent(countSQLConstName),
			),
			buildWhereClauseBlock(baseHasWhere),
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentQuery), goastutil.CachedIdent(IdentNil)),
		),
	}
}

// buildBuilderCountMethod constructs the Count(ctx) terminal method that executes the
// buildCountQuery() form against the read connection and scans a single int64 result. It
// composes with the same .Where() accumulator the All/One terminals use; ORDER BY / LIMIT
// / OFFSET state is ignored by construction because buildCountQuery does not consult
// them.
//
// Takes builderTypeName (string) which is the name of the builder struct.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the Count method declaration.
func buildBuilderCountMethod(builderTypeName string, strategy MethodStrategy) *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: builderReceiver(builderTypeName),
		Name: goastutil.CachedIdent("Count"),
		Type: &ast.FuncType{
			Params: goastutil.FieldList(
				goastutil.Field(IdentCtx, goastutil.SelectorExpr(IdentContext, IdentContextType)),
			),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent("int64")),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(
			goastutil.DefineStmtMulti(
				[]string{IdentQuery, IdentErr},
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentBuilder), "buildCountQuery"),
				),
			),
			BuildErrCheck(goastutil.IntLit(0)),
			goastutil.VarDecl("count", goastutil.CachedIdent("int64")),
			goastutil.AssignStmt(
				goastutil.CachedIdent(IdentErr),
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(
						strategy.BuilderQueryRowCall(
							goastutil.SelectorExprFrom(
								goastutil.CachedIdent(IdentBuilder),
								IdentWhereArgs,
							),
						),
						"Scan",
					),
					&ast.UnaryExpr{
						Op: token.AND,
						X:  goastutil.CachedIdent("count"),
					},
				),
			),
			goastutil.ReturnStmt(goastutil.CachedIdent("count"), goastutil.CachedIdent(IdentErr)),
		),
	}
}
