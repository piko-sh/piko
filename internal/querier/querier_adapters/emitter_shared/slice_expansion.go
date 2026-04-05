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
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	identArgs = "args"

	identPikoExpandSlicePlaceholders = "pikoExpandSlicePlaceholders"

	identPikoSliceExpansionSpec = "pikoSliceExpansionSpec"
)

// NeedsSliceExpansion reports whether the given query requires runtime SQL
// rewriting for slice parameters. This is determined by checking both the
// emitter strategy and the actual SQL text: expansion is only needed when the
// SQL contains parenthesised ?-style placeholders like (?1) that must be
// expanded to (?, ?, ...) at runtime.
//
// PostgreSQL queries use ANY($1) which accepts native array parameters and
// never need expansion, even when emitted through the database/sql emitter.
// SQLite and MySQL queries use IN (?1) which requires expansion.
func NeedsSliceExpansion(query *querier_dto.AnalysedQuery, strategy MethodStrategy) bool {
	if !strategy.NeedsSliceExpansion() || !HasSliceParameter(query) {
		return false
	}

	for i := range query.Parameters {
		if query.Parameters[i].IsSlice {
			placeholder := fmt.Sprintf("(?%d)", query.Parameters[i].Number)
			if strings.Contains(query.SQL, placeholder) {
				return true
			}
		}
	}

	return false
}

// BuildSliceExpansionPreamble constructs the AST statements that rewrite the
// SQL string with renumbered placeholders and flatten slice arguments at
// runtime.
//
// The generated code looks like:
//
//	query := pikoExpandSlicePlaceholders(fetchduetasks, []pikoSliceExpansionSpec{
//	    {1, len(params.Statuses)},
//	    {2, 1},
//	    {3, 1},
//	})
//	args := make([]any, 0, len(params.Statuses)+2)
//	for _, v := range params.Statuses {
//	    args = append(args, v)
//	}
//	args = append(args, params.P2, params.P3)
func BuildSliceExpansionPreamble(query *querier_dto.AnalysedQuery) []ast.Stmt {
	sqlConstName := SnakeToCamelCase(query.Name)

	specElements := make([]ast.Expr, len(query.Parameters))
	for i := range query.Parameters {
		var countExpr ast.Expr
		if query.Parameters[i].IsSlice {
			countExpr = goastutil.CallExpr(goastutil.CachedIdent("len"), paramAccessExpr(query, i))
		} else {
			countExpr = goastutil.IntLit(1)
		}
		specElements[i] = &ast.CompositeLit{
			Elts: []ast.Expr{
				goastutil.IntLit(query.Parameters[i].Number),
				countExpr,
			},
		}
	}

	specLiteral := &ast.CompositeLit{
		Type: &ast.ArrayType{Elt: goastutil.CachedIdent(identPikoSliceExpansionSpec)},
		Elts: specElements,
	}

	statements := []ast.Stmt{
		goastutil.DefineStmt(IdentQuery,
			goastutil.CallExpr(
				goastutil.CachedIdent(identPikoExpandSlicePlaceholders),
				goastutil.CachedIdent(sqlConstName),
				specLiteral,
			),
		),
	}

	statements = append(statements, goastutil.DefineStmt(identArgs, buildArgsMakeCall(query)))

	statements = append(statements, buildParameterFlatteningStatements(query)...)

	return statements
}

// buildParameterFlatteningStatements builds the AST statements that iterate
// over each query parameter and either range-flatten slices into args or append
// scalar values directly.
func buildParameterFlatteningStatements(query *querier_dto.AnalysedQuery) []ast.Stmt {
	var stmts []ast.Stmt

	for i := range query.Parameters {
		paramExpr := paramAccessExpr(query, i)

		if query.Parameters[i].IsSlice {
			stmts = append(stmts, &ast.RangeStmt{
				Key:   goastutil.CachedIdent(IdentBlank),
				Value: goastutil.CachedIdent("v"),
				Tok:   token.DEFINE,
				X:     paramExpr,
				Body: goastutil.BlockStmt(
					goastutil.AssignStmt(
						goastutil.CachedIdent(identArgs),
						goastutil.CallExpr(
							goastutil.CachedIdent("append"),
							goastutil.CachedIdent(identArgs),
							goastutil.CachedIdent("v"),
						),
					),
				),
			})
		} else {
			stmts = append(stmts,
				goastutil.AssignStmt(
					goastutil.CachedIdent(identArgs),
					goastutil.CallExpr(
						goastutil.CachedIdent("append"),
						goastutil.CachedIdent(identArgs),
						paramExpr,
					),
				),
			)
		}
	}

	return stmts
}

// BuildSliceDBCallArgs returns the DB call arguments [ctx, query, args...] for
// use in a method that has been rewritten with slice expansion. The returned
// CallExpr must have Ellipsis set on the call site.
func BuildSliceDBCallArgs() []ast.Expr {
	return []ast.Expr{
		goastutil.CachedIdent(IdentCtx),
		goastutil.CachedIdent(IdentQuery),
		goastutil.CachedIdent(identArgs),
	}
}

// SliceDBCall constructs a database call with ellipsis on the args parameter,
// for use with slice-expanded queries:
//
//	queries.{field}.{method}(ctx, query, args...)
func SliceDBCall(strategy MethodStrategy, query *querier_dto.AnalysedQuery, method string) *ast.CallExpr {
	field := strategy.ConnectionField(query)
	return &ast.CallExpr{
		Fun: goastutil.SelectorExprFrom(
			goastutil.SelectorExprFrom(goastutil.CachedIdent(IdentQueriesReceiver), field),
			method,
		),
		Args:     BuildSliceDBCallArgs(),
		Ellipsis: 1,
	}
}

// buildArgsMakeCall constructs make([]any, 0, <capacity>) where capacity is the
// sum of scalar parameter count plus len() calls for each slice parameter.
func buildArgsMakeCall(query *querier_dto.AnalysedQuery) *ast.CallExpr {
	var scalarCount int
	var sliceLens []ast.Expr

	for i := range query.Parameters {
		if query.Parameters[i].IsSlice {
			sliceLens = append(sliceLens, goastutil.CallExpr(
				goastutil.CachedIdent("len"),
				paramAccessExpr(query, i),
			))
		} else {
			scalarCount++
		}
	}

	var capExpr ast.Expr
	if len(sliceLens) > 0 {
		capExpr = sliceLens[0]
		for _, l := range sliceLens[1:] {
			capExpr = &ast.BinaryExpr{X: capExpr, Op: token.ADD, Y: l}
		}
		if scalarCount > 0 {
			capExpr = &ast.BinaryExpr{X: capExpr, Op: token.ADD, Y: goastutil.IntLit(scalarCount)}
		}
	} else {
		capExpr = goastutil.IntLit(scalarCount)
	}

	return goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		&ast.ArrayType{Elt: goastutil.CachedIdent("any")},
		goastutil.IntLit(0),
		capExpr,
	)
}

// paramAccessExpr returns the AST expression to access a query parameter. For
// queries with a single non-slice parameter, the access is via the local
// variable name; for multi-parameter queries, it is via params.FieldName.
func paramAccessExpr(query *querier_dto.AnalysedQuery, index int) ast.Expr {
	if len(query.Parameters) == 1 && !HasSliceParameter(query) {
		return goastutil.CachedIdent(SnakeToCamelCase(query.Parameters[0].Name))
	}
	return goastutil.SelectorExprFrom(
		goastutil.CachedIdent(IdentParams),
		SnakeToPascalCase(query.Parameters[index].Name),
	)
}
