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
	"piko.sh/piko/internal/querier/querier_dto"
)

// BuildQueryMethod constructs the query method on *Queries for a single query,
// dispatching to the appropriate builder based on the command type. When a
// BatchCopyFromHandler is provided (non-nil), batch and copyfrom commands are also
// supported.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
// Takes batchHandler (BatchCopyFromHandler) which handles batch/copyfrom, or nil if
// unsupported.
//
// Returns ast.Decl which is the method declaration.
func BuildQueryMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
	batchHandler BatchCopyFromHandler,
) ast.Decl {
	if query.IsDynamic {
		return BuildDynamicQueryMethod(query, mappings, tracker, strategy, batchHandler)
	}

	switch query.Command {
	case querier_dto.QueryCommandOne:
		if query.Optional {
			strategy.NoRowsImport(tracker)
			tracker.AddImport("errors")
			return BuildOptionalOneMethod(query, mappings, tracker, strategy)
		}
		return BuildOneMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandMany:
		if HasGroupByKey(query) {
			return BuildGroupedManyMethod(query, mappings, tracker, strategy)
		}
		return BuildManyMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandStream:
		return BuildStreamMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandExecResult:
		strategy.ExecResultImport(tracker)
		return BuildExecResultMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandExecRows:
		return BuildExecRowsMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandBatch:
		if batchHandler != nil {
			tracker.AddImport(batchHandler.BatchImportPath())
			return batchHandler.BuildBatchMethod(query, mappings, tracker)
		}
		return BuildExecMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandCopyFrom:
		if batchHandler != nil {
			tracker.AddImport(batchHandler.CopyFromImportPath())
			return batchHandler.BuildCopyFromMethod(query, mappings, tracker)
		}
		return BuildExecMethod(query, mappings, tracker, strategy)
	case querier_dto.QueryCommandAsyncExec:
		return BuildAsyncExecMethod(query, mappings, tracker, strategy)
	default:
		return BuildExecMethod(query, mappings, tracker, strategy)
	}
}

// BuildDynamicQueryMethod dispatches to the appropriate dynamic method builder based on
// the query command type.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns ast.Decl which is the dynamic method declaration.
func BuildDynamicQueryMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
	_ BatchCopyFromHandler,
) ast.Decl {
	switch query.Command {
	case querier_dto.QueryCommandOne:
		return BuildDynamicOneMethod(query, strategy, mappings)
	case querier_dto.QueryCommandMany:
		if HasGroupByKey(query) {
			return BuildGroupedManyMethod(query, mappings, tracker, strategy)
		}
		return BuildDynamicManyMethod(query, strategy, mappings)
	case querier_dto.QueryCommandStream:
		return BuildDynamicStreamMethod(query, strategy, mappings)
	case querier_dto.QueryCommandExecResult:
		strategy.ExecResultImport(tracker)
		return BuildDynamicExecResultMethod(query, strategy)
	case querier_dto.QueryCommandExecRows:
		return BuildDynamicExecRowsMethod(query, strategy)
	default:
		return BuildDynamicExecMethod(query, strategy)
	}
}

// BuildOneMethod constructs a :one query method using QueryRow + Scan.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration.
func BuildOneMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	scanArguments := BuildScanArgs(query, strategy, mappings)

	statements := make([]ast.Stmt, 0, 4+len(query.OutputColumns))

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query, strategy, goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName)))...)
		queryRowCall := &ast.CallExpr{
			Fun: goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			Args:     BuildSliceDBCallArgs(),
			Ellipsis: 1,
		}
		statements = append(statements, buildOneMethodScanStatements(rowTypeName, queryRowCall, scanArguments, query)...)
	} else {
		queryArguments := BuildQueryArgs(query, strategy)
		queryRowCall := goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			queryArguments...,
		)
		statements = append(statements, buildOneMethodScanStatements(rowTypeName, queryRowCall, scanArguments, query)...)
	}

	statements = append(statements, BuildEmbedNilCheckStatements(query)...)
	statements = append(statements,
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentRow), goastutil.CachedIdent(IdentNil)),
	)

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildMethodParams(query, mappings, tracker),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(rowTypeName)),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// buildOneMethodScanStatements constructs the common VarDecl + embed pre-allocation +
// Scan + error-check statements shared by both the slice-expansion and normal branches of
// BuildOneMethod.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes queryRowCall (ast.Expr) which is the QueryRow call expression.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes query (*querier_dto.AnalysedQuery) for embed pre-allocation.
//
// Returns []ast.Stmt which contains the scan statements.
func buildOneMethodScanStatements(rowTypeName string, queryRowCall ast.Expr, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery) []ast.Stmt {
	embedStatements := BuildEmbedPreAllocStatements(query)
	statements := make([]ast.Stmt, 0, 3+len(embedStatements))
	statements = append(statements, goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)))
	statements = append(statements, embedStatements...)
	statements = append(statements,
		goastutil.DefineStmt(IdentErr,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(queryRowCall, "Scan"),
				scanArguments...,
			),
		),
		BuildErrCheck(goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName))),
	)
	return statements
}

// BuildOptionalOneMethod constructs an optional :one query method using QueryRow + Scan
// that returns (row, bool, error). A zero-row result yields (zeroRow, false, nil) rather
// than the driver no-rows sentinel, so a legitimately empty match is an ordinary outcome
// the caller distinguishes by the bool instead of an error it must special-case. It is
// emitted for a command:one query whose header sets optional: true.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration.
func BuildOptionalOneMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	scanArguments := BuildScanArgs(query, strategy, mappings)
	sentinel := strategy.NoRowsSentinel()

	statements := make([]ast.Stmt, 0, 5+len(query.OutputColumns))

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query, strategy,
			goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName)),
			goastutil.CachedIdent("false"))...)
		queryRowCall := &ast.CallExpr{
			Fun: goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			Args:     BuildSliceDBCallArgs(),
			Ellipsis: 1,
		}
		statements = append(statements, buildOptionalOneScanStatements(rowTypeName, queryRowCall, scanArguments, query, sentinel)...)
	} else {
		queryArguments := BuildQueryArgs(query, strategy)
		queryRowCall := goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			queryArguments...,
		)
		statements = append(statements, buildOptionalOneScanStatements(rowTypeName, queryRowCall, scanArguments, query, sentinel)...)
	}

	statements = append(statements, BuildEmbedNilCheckStatements(query)...)
	statements = append(statements,
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentRow), goastutil.CachedIdent("true"), goastutil.CachedIdent(IdentNil)),
	)

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildMethodParams(query, mappings, tracker),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(rowTypeName)),
				goastutil.Field("", goastutil.CachedIdent(IdentBool)),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// buildOptionalOneScanStatements mirrors buildOneMethodScanStatements but tells a
// zero-row scan apart from a real error: errors.Is(err, sentinel) returns (zeroRow,
// false, nil), any other error returns (zeroRow, false, err), and success falls through
// to the caller's (row, true, nil) return.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes queryRowCall (ast.Expr) which is the QueryRow call expression.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes query (*querier_dto.AnalysedQuery) for embed pre-allocation.
// Takes sentinel (ast.Expr) which is the driver no-rows sentinel tested with errors.Is.
//
// Returns []ast.Stmt which contains the scan statements.
func buildOptionalOneScanStatements(rowTypeName string, queryRowCall ast.Expr, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery, sentinel ast.Expr) []ast.Stmt {
	embedStatements := BuildEmbedPreAllocStatements(query)
	statements := make([]ast.Stmt, 0, 4+len(embedStatements))
	statements = append(statements, goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)))
	statements = append(statements, embedStatements...)
	statements = append(statements,
		goastutil.DefineStmt(IdentErr,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(queryRowCall, "Scan"),
				scanArguments...,
			),
		),
		goastutil.IfStmt(nil,
			goastutil.CallExpr(goastutil.SelectorExpr("errors", "Is"), goastutil.CachedIdent(IdentErr), sentinel),
			goastutil.BlockStmt(goastutil.ReturnStmt(
				goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName)),
				goastutil.CachedIdent("false"),
				goastutil.CachedIdent(IdentNil),
			)),
		),
		BuildErrCheck(
			goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName)),
			goastutil.CachedIdent("false"),
		),
	)
	return statements
}

// BuildManyMethod constructs a :many query method using Query + loop.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration.
func BuildManyMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	scanArguments := BuildScanArgs(query, strategy, mappings)

	var statements []ast.Stmt
	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query, strategy, goastutil.CachedIdent(IdentNil))...)
		dbCall := SliceDBCall(strategy, query, strategy.QueryMethod())
		statements = append(statements, BuildRowsIterationBodyFromSliceCall(rowTypeName, dbCall, scanArguments, query)...)
	} else {
		queryArguments := BuildQueryArgs(query, strategy)
		statements = BuildRowsIterationBody(rowTypeName, queryArguments, scanArguments, query, strategy)
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildMethodParams(query, mappings, tracker),
			Results: goastutil.FieldList(
				goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildRowsIterationBody constructs the full body of a :many method including Query call,
// error check, defer Close, var results, for loop with Scan, rows.Err check, and final
// return.
//
// When query is non-nil, embed handling is included in the scan loop.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes queryArguments ([]ast.Expr) which are the Query call arguments.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes query (*querier_dto.AnalysedQuery) which defines embed handling, or nil to skip
// embeds.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Stmt which contains the method body statements.
func BuildRowsIterationBody(rowTypeName string, queryArguments []ast.Expr, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Stmt {
	dbCall := strategy.DBCall(strategy.ConnectionField(query), strategy.QueryMethod(), queryArguments)
	return buildRowsIterationBodyFromCall(rowTypeName, dbCall, scanArguments, query)
}

// BuildRowsIterationBodyFromSliceCall constructs the body of a :many method from a
// pre-built call expression with ellipsis for slice-expanded queries.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes dbCall (*ast.CallExpr) which is the pre-built database call.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes query (*querier_dto.AnalysedQuery) which defines embed handling.
//
// Returns []ast.Stmt which contains the method body statements.
func BuildRowsIterationBodyFromSliceCall(rowTypeName string, dbCall *ast.CallExpr, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery) []ast.Stmt {
	return buildRowsIterationBodyFromCall(rowTypeName, dbCall, scanArguments, query)
}

// buildRowsIterationBodyFromCall constructs the full body of a :many method from a
// database call expression.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes dbCall (*ast.CallExpr) which is the database call expression.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes query (*querier_dto.AnalysedQuery) which defines embed handling.
//
// Returns []ast.Stmt which contains the method body statements.
func buildRowsIterationBodyFromCall(rowTypeName string, dbCall *ast.CallExpr, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{IdentRows, IdentErr},
			dbCall,
		),
		BuildErrCheck(goastutil.CachedIdent(IdentNil)),
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Close"),
			),
		},
		&ast.DeclStmt{
			Decl: &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{goastutil.CachedIdent(IdentResults)},
						Type:  &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)},
					},
				},
			},
		},
		&ast.ForStmt{
			Cond: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Next"),
			),
			Body: goastutil.BlockStmt(BuildRowsScanLoop(rowTypeName, scanArguments, query)...),
		},
		BuildRowsErrCheck(),
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentResults), goastutil.CachedIdent(IdentNil)),
	}
}

// BuildRowsScanLoop constructs the inner loop body that declares a row variable, scans
// into it, and appends to results.
//
// Takes rowTypeName (string) which is the row struct name.
// Takes scanArguments ([]ast.Expr) which are the Scan call arguments.
// Takes query (*querier_dto.AnalysedQuery) which defines embed handling, or nil to skip
// embeds.
//
// Returns []ast.Stmt which contains the loop body statements.
func BuildRowsScanLoop(rowTypeName string, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery) []ast.Stmt {
	statements := []ast.Stmt{
		goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)),
	}
	if query != nil {
		statements = append(statements, BuildEmbedPreAllocStatements(query)...)
	}
	statements = append(statements,
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
	)
	if query != nil {
		statements = append(statements, BuildEmbedNilCheckStatements(query)...)
	}
	statements = append(statements,
		goastutil.AssignStmt(
			goastutil.CachedIdent(IdentResults),
			goastutil.CallExpr(
				goastutil.CachedIdent("append"),
				goastutil.CachedIdent(IdentResults),
				goastutil.CachedIdent(IdentRow),
			),
		),
	)
	return statements
}

// BuildRowsErrCheck constructs the rows.Err() check after the iteration loop.
//
// Returns *ast.IfStmt which is the error check statement.
func BuildRowsErrCheck() *ast.IfStmt {
	return goastutil.IfStmt(
		goastutil.DefineStmt(IdentErr,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Err"),
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
	)
}

// BuildExecMethod constructs a :exec query method using Exec.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration.
func BuildExecMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	var statements []ast.Stmt

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query, strategy)...)
		execCall := SliceDBCall(strategy, query, strategy.ExecMethod())
		if strategy.ExecReturnsResult() {
			statements = append(statements,
				goastutil.DefineStmtMulti(
					[]string{IdentBlank, IdentErr},
					execCall,
				),
				goastutil.ReturnStmt(goastutil.CachedIdent(IdentErr)),
			)
		} else {
			statements = append(statements, goastutil.ReturnStmt(execCall))
		}
	} else {
		queryArguments := BuildQueryArgs(query, strategy)
		execCall := strategy.DBCall(strategy.ConnectionField(query), strategy.ExecMethod(), queryArguments)
		if strategy.ExecReturnsResult() {
			statements = []ast.Stmt{
				goastutil.DefineStmtMulti(
					[]string{IdentBlank, IdentErr},
					execCall,
				),
				goastutil.ReturnStmt(goastutil.CachedIdent(IdentErr)),
			}
		} else {
			statements = []ast.Stmt{
				goastutil.ReturnStmt(execCall),
			}
		}
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildMethodParams(query, mappings, tracker),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildAsyncExecMethod constructs an :asyncexec query method.
//
// The body is identical to BuildExecMethod (issue the statement via Exec and return only
// the error) because driver-level dispatch is the same. The distinction lives in the
// attached doc comment that documents the fire-and-forget semantics so downstream callers
// do not assume the mutation has finished applying when the call returns nil.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration with the async-semantics doc
// comment attached.
func BuildAsyncExecMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	decl := BuildExecMethod(query, mappings, tracker, strategy)
	decl.Doc = asyncExecDocComment(query.Name)
	return decl
}

// asyncExecDocComment builds the doc comment block attached to every async-exec method.
//
// The comment warns that the returned error reflects only the acceptance step and the
// server-side mutation may still be running, so callers must consult the engine's
// mutation log (system.mutations on ClickHouse) to confirm completion. The comment opens
// with the method name so go doc renders the binding cleanly.
//
// Takes name (string) which is the generated method name.
//
// Returns *ast.CommentGroup which carries the block ready to attach to a *ast.FuncDecl.
func asyncExecDocComment(name string) *ast.CommentGroup {
	return &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: "// " + name + " issues an asynchronous mutation. The server accepts the"},
			{Text: "// statement and returns when the mutation has been queued; the"},
			{Text: "// background execution proceeds independently. Inspect the"},
			{Text: "// engine's mutation log (system.mutations on ClickHouse) for"},
			{Text: "// completion status. The returned error reflects only the"},
			{Text: "// acceptance step; a nil error does NOT guarantee that the"},
			{Text: "// mutation has finished applying."},
		},
	}
}

// BuildExecResultMethod constructs a :execresult method returning the database-specific
// result type.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration.
func BuildExecResultMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	var statements []ast.Stmt

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query, strategy, goastutil.CachedIdent(IdentNil))...)
		statements = append(statements,
			goastutil.ReturnStmt(SliceDBCall(strategy, query, strategy.ExecMethod())),
		)
	} else {
		queryArguments := BuildQueryArgs(query, strategy)
		statements = []ast.Stmt{
			goastutil.ReturnStmt(strategy.DBCall(strategy.ConnectionField(query), strategy.ExecMethod(), queryArguments)),
		}
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildMethodParams(query, mappings, tracker),
			Results: goastutil.FieldList(
				goastutil.Field("", strategy.ExecResultReturnType()),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildExecRowsMethod constructs a :execrows method returning int64.
//
// Takes query (*querier_dto.AnalysedQuery) which defines the query to emit.
// Takes mappings (*querier_dto.TypeMappingTable) for type resolution.
// Takes tracker (*ImportTracker) for import collection.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the method declaration.
func BuildExecRowsMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	var bodyStatements []ast.Stmt

	if NeedsSliceExpansion(query, strategy) {
		bodyStatements = append(bodyStatements, BuildSliceExpansionPreamble(query, strategy, goastutil.IntLit(0))...)

		dbCall := SliceDBCall(strategy, query, strategy.ExecMethod())
		bodyStatements = append(bodyStatements,
			goastutil.DefineStmtMulti(
				[]string{IdentResults, IdentErr},
				dbCall,
			),
			BuildErrCheck(goastutil.IntLit(0)),
			goastutil.ReturnStmt(
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentResults), "RowsAffected"),
				),
			),
		)
	} else {
		queryArguments := BuildQueryArgs(query, strategy)
		bodyStatements = strategy.BuildExecRowsBody(queryArguments, strategy.ConnectionField(query))
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildMethodParams(query, mappings, tracker),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent("int64")),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(bodyStatements...),
	}
}
