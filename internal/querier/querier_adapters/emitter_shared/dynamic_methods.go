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

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

// buildDynamicOneQueryStatements builds the var-row declaration, embed
// pre-alloc, err := queryRowCall.Scan(scanArguments...), and err-check
// statements that are common to every branch of BuildDynamicOneMethod.
//
// Takes rowTypeName (string) which holds the generated row struct name.
// Takes queryRowCall (*ast.CallExpr) which holds the db.QueryRow(...) call.
// Takes scanArguments ([]ast.Expr) which holds the Scan destination args.
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query.
//
// Returns []ast.Stmt which holds the common query-row statements.
func buildDynamicOneQueryStatements(
	rowTypeName string,
	queryRowCall *ast.CallExpr,
	scanArguments []ast.Expr,
	query *querier_dto.AnalysedQuery,
) []ast.Stmt {
	embedStmts := BuildEmbedPreAllocStatements(query)
	stmts := make([]ast.Stmt, 0, 3+len(embedStmts))
	stmts = append(stmts, goastutil.VarDecl(IdentRow, goastutil.CachedIdent(rowTypeName)))
	stmts = append(stmts, embedStmts...)
	stmts = append(stmts,
		goastutil.DefineStmt(IdentErr,
			goastutil.CallExpr(
				goastutil.SelectorExprFrom(queryRowCall, "Scan"),
				scanArguments...,
			),
		),
		BuildErrCheck(goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName))),
	)
	return stmts
}

// BuildDynamicOneMethod constructs a :one query method with a params struct.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// definition.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which holds the complete :one method AST node.
func BuildDynamicOneMethod(query *querier_dto.AnalysedQuery, strategy MethodStrategy) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	sortParameter := FindSortableParameter(query)
	scanArguments := BuildScanArgs(query)

	statements := BuildParamsInitStatements(query)

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query)...)
		if sortParameter != nil {
			statements = append(statements, BuildSortableQueryAppend(*sortParameter)...)
		}
		queryRowCall := &ast.CallExpr{
			Fun: goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			Args:     BuildSliceDBCallArgs(),
			Ellipsis: 1,
		}
		statements = append(statements, buildDynamicOneQueryStatements(rowTypeName, queryRowCall, scanArguments, query)...)
	} else if sortParameter != nil {
		statements = append(statements, BuildSortableQueryInit(query, *sortParameter)...)
		queryRowCall := goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			BuildSortableDynamicQueryArgs(query)...,
		)
		statements = append(statements, buildDynamicOneQueryStatements(rowTypeName, queryRowCall, scanArguments, query)...)
	} else {
		queryRowCall := goastutil.CallExpr(
			goastutil.SelectorExprFrom(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), strategy.ConnectionField(query)),
				strategy.QueryRowMethod(),
			),
			BuildDynamicQueryArgs(query)...,
		)
		statements = append(statements, buildDynamicOneQueryStatements(rowTypeName, queryRowCall, scanArguments, query)...)
	}

	statements = append(statements, BuildEmbedNilCheckStatements(query)...)
	statements = append(statements,
		goastutil.ReturnStmt(goastutil.CachedIdent(IdentRow), goastutil.CachedIdent(IdentNil)),
	)

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params:  BuildDynamicMethodParams(query),
			Results: goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(rowTypeName)), goastutil.Field("", goastutil.CachedIdent(IdentError))),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildDynamicManyMethod constructs a :many query method with a params struct.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// definition.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which holds the complete :many method AST node.
func BuildDynamicManyMethod(query *querier_dto.AnalysedQuery, strategy MethodStrategy) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	sortParameter := FindSortableParameter(query)
	scanArguments := BuildScanArgs(query)

	statements := BuildParamsInitStatements(query)

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query)...)
		if sortParameter != nil {
			statements = append(statements, BuildSortableQueryAppend(*sortParameter)...)
		}
		dbCall := SliceDBCall(strategy, query, strategy.QueryMethod())
		statements = append(statements, BuildRowsIterationBodyFromSliceCall(rowTypeName, dbCall, scanArguments, query)...)
	} else if sortParameter != nil {
		statements = append(statements, BuildSortableQueryInit(query, *sortParameter)...)
		queryArguments := BuildSortableDynamicQueryArgs(query)
		statements = append(statements, BuildRowsIterationBody(rowTypeName, queryArguments, scanArguments, query, strategy)...)
	} else {
		queryArguments := BuildDynamicQueryArgs(query)
		statements = append(statements, BuildRowsIterationBody(rowTypeName, queryArguments, scanArguments, query, strategy)...)
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildDynamicMethodParams(query),
			Results: goastutil.FieldList(
				goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(rowTypeName)}),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildDynamicExecMethod constructs a :exec query method with a params struct.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// definition.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which holds the complete :exec method AST node.
func BuildDynamicExecMethod(query *querier_dto.AnalysedQuery, strategy MethodStrategy) *ast.FuncDecl {
	statements := BuildParamsInitStatements(query)

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query)...)
		statements = append(statements,
			goastutil.DefineStmtMulti([]string{IdentBlank, IdentErr}, SliceDBCall(strategy, query, strategy.ExecMethod())),
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentErr)),
		)
	} else {
		queryArguments := BuildDynamicQueryArgs(query)
		statements = append(statements,
			goastutil.DefineStmtMulti([]string{IdentBlank, IdentErr}, strategy.DBCall(strategy.ConnectionField(query), strategy.ExecMethod(), queryArguments)),
			goastutil.ReturnStmt(goastutil.CachedIdent(IdentErr)),
		)
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params:  BuildDynamicMethodParams(query),
			Results: goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentError))),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildDynamicExecResultMethod constructs a :execresult method with a params
// struct.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// definition.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which holds the complete :execresult method AST node.
func BuildDynamicExecResultMethod(query *querier_dto.AnalysedQuery, strategy MethodStrategy) *ast.FuncDecl {
	statements := BuildParamsInitStatements(query)

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query)...)
		statements = append(statements, goastutil.ReturnStmt(SliceDBCall(strategy, query, strategy.ExecMethod())))
	} else {
		queryArguments := BuildDynamicQueryArgs(query)
		statements = append(statements, goastutil.ReturnStmt(strategy.DBCall(strategy.ConnectionField(query), strategy.ExecMethod(), queryArguments)))
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildDynamicMethodParams(query),
			Results: goastutil.FieldList(
				goastutil.Field("", strategy.ExecResultReturnType()),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}

// BuildDynamicExecRowsMethod constructs a :execrows method with a params
// struct.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// definition.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which holds the complete :execrows method AST node.
func BuildDynamicExecRowsMethod(query *querier_dto.AnalysedQuery, strategy MethodStrategy) *ast.FuncDecl {
	statements := BuildParamsInitStatements(query)

	if NeedsSliceExpansion(query, strategy) {
		statements = append(statements, BuildSliceExpansionPreamble(query)...)
		dbCall := SliceDBCall(strategy, query, strategy.ExecMethod())
		statements = append(statements,
			goastutil.DefineStmtMulti([]string{IdentResults, IdentErr}, dbCall),
			BuildErrCheck(goastutil.IntLit(0)),
			goastutil.ReturnStmt(
				goastutil.CallExpr(
					goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentResults), "RowsAffected"),
				),
			),
		)
	} else {
		queryArguments := BuildDynamicQueryArgs(query)
		field := strategy.ConnectionField(query)
		statements = append(statements, strategy.BuildExecRowsBody(queryArguments, field)...)
	}

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(query.Name),
		Type: &ast.FuncType{
			Params: BuildDynamicMethodParams(query),
			Results: goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent("int64")),
				goastutil.Field("", goastutil.CachedIdent(IdentError)),
			),
		},
		Body: goastutil.BlockStmt(statements...),
	}
}
