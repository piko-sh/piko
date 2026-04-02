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

// BuildStreamMethod constructs a :stream query method that returns a
// range-over-func iterator: func(yield func(Row, error) bool).
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// metadata.
// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go type
// mappings.
// Takes tracker (*ImportTracker) which collects required import paths.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the stream method declaration.
func BuildStreamMethod(
	query *querier_dto.AnalysedQuery,
	mappings *querier_dto.TypeMappingTable,
	tracker *ImportTracker,
	strategy MethodStrategy,
) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	scanArguments := BuildScanArgs(query)

	var iteratorBody []ast.Stmt
	if NeedsSliceExpansion(query, strategy) {
		iteratorBody = append(iteratorBody, BuildSliceExpansionPreamble(query)...)
		dbCall := SliceDBCall(strategy, query, strategy.QueryMethod())
		iteratorBody = append(iteratorBody, buildStreamIteratorBodyFromCall(rowTypeName, dbCall, scanArguments, query)...)
	} else {
		queryArguments := BuildQueryArgs(query)
		iteratorBody = buildStreamIteratorBody(rowTypeName, queryArguments, scanArguments, query, strategy)
	}

	return buildStreamFuncDecl(
		query.Name,
		rowTypeName,
		BuildMethodParams(query, mappings, tracker),
		iteratorBody,
		strategy,
	)
}

// BuildDynamicStreamMethod constructs a :stream query method with a params
// struct for dynamic queries.
//
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// metadata.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns *ast.FuncDecl which is the dynamic stream method declaration.
func BuildDynamicStreamMethod(query *querier_dto.AnalysedQuery, strategy MethodStrategy) *ast.FuncDecl {
	rowTypeName := query.Name + "Row"
	sortParameter := FindSortableParameter(query)
	scanArguments := BuildScanArgs(query)

	preStatements := BuildParamsInitStatements(query)

	var iteratorBody []ast.Stmt
	if NeedsSliceExpansion(query, strategy) {
		preStatements = append(preStatements, BuildSliceExpansionPreamble(query)...)
		if sortParameter != nil {
			preStatements = append(preStatements, BuildSortableQueryAppend(*sortParameter)...)
		}
		dbCall := SliceDBCall(strategy, query, strategy.QueryMethod())
		iteratorBody = buildStreamIteratorBodyFromCall(rowTypeName, dbCall, scanArguments, query)
	} else if sortParameter != nil {
		preStatements = append(preStatements, BuildSortableQueryInit(query, *sortParameter)...)
		queryArguments := BuildSortableDynamicQueryArgs(query)
		iteratorBody = buildStreamIteratorBody(rowTypeName, queryArguments, scanArguments, query, strategy)
	} else {
		queryArguments := BuildDynamicQueryArgs(query)
		iteratorBody = buildStreamIteratorBody(rowTypeName, queryArguments, scanArguments, query, strategy)
	}

	if len(preStatements) > 0 {
		iteratorBody = append(preStatements, iteratorBody...)
	}

	return buildStreamFuncDecl(
		query.Name,
		rowTypeName,
		BuildDynamicMethodParams(query),
		iteratorBody,
		strategy,
	)
}

// buildStreamFuncDecl constructs the func declaration for a stream method.
//
// Takes queryName (string) which is the method name.
// Takes rowTypeName (string) which is the name of the row result type.
// Takes params (*ast.FieldList) which defines the method parameters.
// Takes bodyStatements ([]ast.Stmt) which are the statements in the iterator
// body.
// Takes strategy (MethodStrategy) which provides the receiver.
//
// Returns *ast.FuncDecl which is the method returning func(yield func(RowType,
// error) bool).
func buildStreamFuncDecl(
	queryName string,
	rowTypeName string,
	params *ast.FieldList,
	bodyStatements []ast.Stmt,
	strategy MethodStrategy,
) *ast.FuncDecl {
	yieldFuncType := goastutil.FuncType(
		goastutil.FieldList(
			goastutil.Field("yield", streamYieldType(rowTypeName)),
		),
		nil,
	)

	iteratorType := goastutil.FuncType(
		goastutil.FieldList(
			goastutil.Field("", streamYieldType(rowTypeName)),
		),
		nil,
	)

	return &ast.FuncDecl{
		Recv: strategy.QueriesReceiver(),
		Name: goastutil.CachedIdent(queryName),
		Type: &ast.FuncType{
			Params: params,
			Results: goastutil.FieldList(
				goastutil.Field("", iteratorType),
			),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.FuncLit(yieldFuncType, goastutil.BlockStmt(bodyStatements...)),
			),
		),
	}
}

// streamYieldType constructs the type func(RowType, error) bool.
//
// Takes rowTypeName (string) which is the name of the row result type.
//
// Returns *ast.FuncType which represents func(RowType, error) bool.
func streamYieldType(rowTypeName string) *ast.FuncType {
	return goastutil.FuncType(
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent(rowTypeName)),
			goastutil.Field("", goastutil.CachedIdent(IdentError)),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent("bool")),
		),
	)
}

// buildStreamIteratorBody constructs the body of the iterator closure:
// Query, error yield, defer Close, for rows.Next scan+yield loop, final
// rows.Err yield.
//
// Takes rowTypeName (string) which is the name of the row result type.
// Takes queryArguments ([]ast.Expr) which are the arguments for Query.
// Takes scanArguments ([]ast.Expr) which are the arguments for rows.Scan.
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// metadata.
// Takes strategy (MethodStrategy) which provides database-specific AST nodes.
//
// Returns []ast.Stmt which are the statements forming the iterator body.
func buildStreamIteratorBody(
	rowTypeName string,
	queryArguments []ast.Expr,
	scanArguments []ast.Expr,
	query *querier_dto.AnalysedQuery,
	strategy MethodStrategy,
) []ast.Stmt {
	dbCall := strategy.DBCall(strategy.ConnectionField(query), strategy.QueryMethod(), queryArguments)
	return buildStreamIteratorBodyFromCall(rowTypeName, dbCall, scanArguments, query)
}

func buildStreamIteratorBodyFromCall(
	rowTypeName string,
	dbCall *ast.CallExpr,
	scanArguments []ast.Expr,
	query *querier_dto.AnalysedQuery,
) []ast.Stmt {
	zeroRow := goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName))

	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{IdentRows, IdentErr},
			dbCall,
		),
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  goastutil.CachedIdent(IdentErr),
				Op: token.NEQ,
				Y:  goastutil.CachedIdent(IdentNil),
			},
			Body: goastutil.BlockStmt(
				&ast.ExprStmt{X: goastutil.CallExpr(
					goastutil.CachedIdent("yield"),
					zeroRow,
					goastutil.CachedIdent(IdentErr),
				)},
				goastutil.ReturnStmt(),
			),
		},
		&ast.DeferStmt{
			Call: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Close"),
			),
		},
		&ast.ForStmt{
			Cond: goastutil.CallExpr(
				goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentRows), "Next"),
			),
			Body: goastutil.BlockStmt(buildStreamScanLoop(rowTypeName, scanArguments, query)...),
		},
		buildStreamFinalErrCheck(rowTypeName),
	}
}

// buildStreamScanLoop constructs the inner for-loop body that declares a row,
// scans into it, and yields it.
//
// Takes rowTypeName (string) which is the name of the row result type.
// Takes scanArguments ([]ast.Expr) which are the arguments for rows.Scan.
// Takes query (*querier_dto.AnalysedQuery) which holds the analysed query
// metadata.
//
// Returns []ast.Stmt which are the statements forming the scan loop body.
func buildStreamScanLoop(rowTypeName string, scanArguments []ast.Expr, query *querier_dto.AnalysedQuery) []ast.Stmt {
	zeroRow := goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName))

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
				&ast.IfStmt{
					Cond: &ast.UnaryExpr{
						Op: token.NOT,
						X: goastutil.CallExpr(
							goastutil.CachedIdent("yield"),
							zeroRow,
							goastutil.CachedIdent(IdentErr),
						),
					},
					Body: goastutil.BlockStmt(goastutil.ReturnStmt()),
				},
				&ast.BranchStmt{Tok: token.CONTINUE},
			),
		),
	)
	if query != nil {
		statements = append(statements, BuildEmbedNilCheckStatements(query)...)
	}
	statements = append(statements,
		&ast.IfStmt{
			Cond: &ast.UnaryExpr{
				Op: token.NOT,
				X: goastutil.CallExpr(
					goastutil.CachedIdent("yield"),
					goastutil.CachedIdent(IdentRow),
					goastutil.CachedIdent(IdentNil),
				),
			},
			Body: goastutil.BlockStmt(goastutil.ReturnStmt()),
		},
	)
	return statements
}

// buildStreamFinalErrCheck constructs the final rows.Err() check after the
// iteration loop, yielding the error if present.
//
// Takes rowTypeName (string) which is the name of the row result type.
//
// Returns *ast.IfStmt which is the if-statement checking rows.Err().
func buildStreamFinalErrCheck(rowTypeName string) *ast.IfStmt {
	zeroRow := goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName))

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
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.CachedIdent("yield"),
				zeroRow,
				goastutil.CachedIdent(IdentErr),
			)},
		),
	)
}
