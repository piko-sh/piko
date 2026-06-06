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
	// identArgs is the Go identifier for the flattened arguments variable.
	identArgs = "args"

	// identPikoExpandSlicePlaceholders is the name of the generated helper function that
	// rewrites SQL placeholders for slice expansion.
	identPikoExpandSlicePlaceholders = "pikoExpandSlicePlaceholders"

	// identPikoSliceExpansionSpec is the name of the generated struct type that describes
	// how each placeholder expands at runtime.
	identPikoSliceExpansionSpec = "pikoSliceExpansionSpec"

	// identExpansionError is the local error variable the slice-expansion preamble binds
	// from pikoExpandSlicePlaceholders. A dedicated name (rather than the shared err) avoids
	// a redeclaration conflict with the later `err :=` produced by the scan and exec
	// branches that follow the preamble in the generated method body.
	identExpansionError = "expansionError"
)

// NeedsSliceExpansion reports whether the query needs runtime slice expansion. This is
// determined by checking both the emitter strategy and the actual SQL text: expansion is
// only needed when the SQL contains a parenthesised placeholder like (?1) or ($1) that
// must be expanded to (?, ?, ...) at runtime.
//
// The marker rune comes from the strategy, so postgres-family IN ($1) queries are
// detected with the '$' marker while sqlite and mysql IN (?1) queries use '?'. A postgres
// native array query written as ANY($1) on the native pgx path never reaches here because
// the pgx strategy reports NeedsSliceExpansion false.
//
// Takes query (*querier_dto.AnalysedQuery) which is the analysed query to inspect for
// slice parameters.
// Takes strategy (MethodStrategy) which provides the emitter behaviour that determines
// whether expansion is supported.
//
// Returns bool which is true when the query requires runtime slice expansion.
func NeedsSliceExpansion(query *querier_dto.AnalysedQuery, strategy MethodStrategy) bool {
	if !strategy.NeedsSliceExpansion() || !HasSliceParameter(query) {
		return false
	}

	marker := strategy.PlaceholderMarker()
	for i := range query.Parameters {
		if query.Parameters[i].IsSlice {
			placeholder := fmt.Sprintf("(%c%d)", marker, query.Parameters[i].Number)
			if strings.Contains(query.SQL, placeholder) {
				return true
			}
		}
	}

	return false
}

// BuildSliceExpansionPreamble constructs the AST statements that rewrite the SQL string
// with renumbered placeholders and flatten slice arguments at runtime.
//
// The generated code looks like:
//
//	query, expansionError := pikoExpandSlicePlaceholders(fetchduetasks, []pikoSliceExpansionSpec{
//	    {Placeholder: 1, Count: len(params.Statuses)},
//	    {Placeholder: 2, Count: 1},
//	    {Placeholder: 3, Count: 1},
//	})
//	if expansionError != nil {
//	    return nil, expansionError
//	}
//	args := make([]any, 0, len(params.Statuses)+2)
//	for _, v := range params.Statuses {
//	    args = append(args, v)
//	}
//	args = append(args, params.P2, params.P3)
//
// pikoExpandSlicePlaceholders returns errPikoTooManyBindVariables (wrapped) when the
// renumbered placeholder count would exceed pikoMaxBindVariables, so an oversized slice
// parameter surfaces as a returned error rather than a statement the driver would reject.
// The zeroValues are returned alongside the error so the guard matches the enclosing
// method's result signature.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter metadata used to
// build the preamble statements.
// Takes zeroValues ([]ast.Expr) which are the zero values returned ahead of the expansion
// error (none for an error-only :exec method, the row struct for a :one method, and so
// on).
//
// Returns []ast.Stmt which contains the variable definitions, the error guard, and
// flattening loops.
func BuildSliceExpansionPreamble(query *querier_dto.AnalysedQuery, strategy MethodStrategy, zeroValues ...ast.Expr) []ast.Stmt {
	guard := buildExpansionErrorReturnGuard(zeroValues)
	return assembleSliceExpansionPreamble(query, strategy, guard)
}

// BuildSliceExpansionStreamPreamble constructs the slice-expansion preamble for a :stream
// method. The iterator closure reports errors by yielding them rather than returning a
// value tuple, so an oversized bind list yields (zeroRow, expansionError) and then
// returns from the closure instead of returning the error to the caller directly.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter metadata.
// Takes rowTypeName (string) which is the row struct name used for the zero row passed to
// yield.
//
// Returns []ast.Stmt which contains the variable definitions, the yield guard, and
// flattening loops.
func BuildSliceExpansionStreamPreamble(query *querier_dto.AnalysedQuery, strategy MethodStrategy, rowTypeName string) []ast.Stmt {
	guard := buildExpansionErrorYieldGuard(rowTypeName)
	return assembleSliceExpansionPreamble(query, strategy, guard)
}

// assembleSliceExpansionPreamble emits the shared slice-expansion statements: the
// pikoExpandSlicePlaceholders call binding query and expansionError, the supplied error
// guard, the args make() call, and the per-parameter flattening loops. The two public
// entry points differ only in how they surface expansionError, which the caller passes as
// the guard statement.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter metadata.
// Takes guard (ast.Stmt) which surfaces a non-nil expansionError.
//
// Returns []ast.Stmt which contains the preamble statements.
func assembleSliceExpansionPreamble(query *querier_dto.AnalysedQuery, strategy MethodStrategy, guard ast.Stmt) []ast.Stmt {
	sqlConstName := SnakeToCamelCase(query.Name)

	var specElements []ast.Expr
	for i := range query.Parameters {
		if query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}
		var countExpr ast.Expr
		if query.Parameters[i].IsSlice {
			countExpr = goastutil.CallExpr(goastutil.CachedIdent("len"), paramAccessExpr(query, i))
		} else {
			countExpr = goastutil.IntLit(1)
		}
		specElements = append(specElements, &ast.CompositeLit{
			Elts: []ast.Expr{
				goastutil.KeyValueIdent("Placeholder", goastutil.IntLit(query.Parameters[i].Number)),
				goastutil.KeyValueIdent("Count", countExpr),
			},
		})
	}

	specLiteral := &ast.CompositeLit{
		Type: &ast.ArrayType{Elt: goastutil.CachedIdent(identPikoSliceExpansionSpec)},
		Elts: specElements,
	}

	flatteningStatements := buildParameterFlatteningStatements(query, strategy)
	statements := make([]ast.Stmt, 0, 3+len(flatteningStatements))
	statements = append(statements,
		goastutil.DefineStmtMulti(
			[]string{IdentQuery, identExpansionError},
			goastutil.CallExpr(
				goastutil.CachedIdent(identPikoExpandSlicePlaceholders),
				goastutil.CachedIdent(sqlConstName),
				specLiteral,
			),
		),
		guard,
		goastutil.DefineStmt(identArgs, buildArgsMakeCall(query)),
	)

	statements = append(statements, flatteningStatements...)

	return statements
}

// buildExpansionErrorReturnGuard constructs the if-block that returns the zero values and
// the expansion error when pikoExpandSlicePlaceholders reports an oversized bind list.
// Used by the return-style query methods.
//
// Takes zeroValues ([]ast.Expr) which are the zero values returned ahead of the expansion
// error.
//
// Returns *ast.IfStmt which is the expansion-error guard statement.
func buildExpansionErrorReturnGuard(zeroValues []ast.Expr) *ast.IfStmt {
	returnValues := make([]ast.Expr, 0, len(zeroValues)+1)
	returnValues = append(returnValues, zeroValues...)
	returnValues = append(returnValues, goastutil.CachedIdent(identExpansionError))

	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(identExpansionError),
			Op: token.NEQ,
			Y:  goastutil.CachedIdent(IdentNil),
		},
		Body: goastutil.BlockStmt(
			goastutil.ReturnStmt(returnValues...),
		),
	}
}

// buildExpansionErrorYieldGuard constructs the if-block that yields (zeroRow,
// expansionError) and returns from the iterator closure when pikoExpandSlicePlaceholders
// reports an oversized bind list. Used by the :stream methods, whose closure surfaces
// errors through yield rather than a return tuple.
//
// Takes rowTypeName (string) which is the row struct name used for the zero row.
//
// Returns *ast.IfStmt which is the expansion-error yield guard statement.
func buildExpansionErrorYieldGuard(rowTypeName string) *ast.IfStmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent(identExpansionError),
			Op: token.NEQ,
			Y:  goastutil.CachedIdent(IdentNil),
		},
		Body: goastutil.BlockStmt(
			&ast.ExprStmt{X: goastutil.CallExpr(
				goastutil.CachedIdent("yield"),
				goastutil.CompositeLit(goastutil.CachedIdent(rowTypeName)),
				goastutil.CachedIdent(identExpansionError),
			)},
			goastutil.ReturnStmt(),
		),
	}
}

// buildParameterFlatteningStatements builds the AST statements that iterate over each
// query parameter and either range-flatten slices into args or append scalar values
// directly.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter list to generate
// flattening code for.
//
// Returns []ast.Stmt which contains the range loops and append statements for all
// parameters.
func buildParameterFlatteningStatements(query *querier_dto.AnalysedQuery, strategy MethodStrategy) []ast.Stmt {
	if strategy != nil && !strategy.PreservesPlaceholderIndices() {
		if ordered := buildOccurrenceOrderedFlattening(query); ordered != nil {
			return ordered
		}
	}

	var stmts []ast.Stmt
	for i := range query.Parameters {
		if query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}
		stmts = append(stmts, flattenParameterStatement(query, i))
	}

	return stmts
}

// buildOccurrenceOrderedFlattening builds the args-flattening statements in SQL
// placeholder-occurrence order (one statement per occurrence), used by engines that do
// not preserve placeholder indices. Returns nil when no placeholder order can be derived
// so the caller falls back to declaration order.
//
// Every placeholder number in the analysed SQL is expected to resolve to a declared
// parameter, so parameterIndexByNumber returning -1 signals an analyser invariant break
// that would mismatch the flattened bind count. The occurrence is skipped to keep codegen
// panic-free; the mismatch surfaces as a driver argument-count error at the call site if
// such an invalid SQL ever reaches this path.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the SQL and parameter metadata.
//
// Returns []ast.Stmt in occurrence order, or nil.
func buildOccurrenceOrderedFlattening(query *querier_dto.AnalysedQuery) []ast.Stmt {
	ordering := PlaceholderOccurrenceOrder(query.SQL)
	if len(ordering) == 0 {
		return nil
	}
	var stmts []ast.Stmt
	for _, number := range ordering {
		index := parameterIndexByNumber(query, number)
		if index < 0 || query.Parameters[index].Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}
		stmts = append(stmts, flattenParameterStatement(query, index))
	}
	return stmts
}

// flattenParameterStatement returns the single flattening statement for the parameter at
// index: a range-append loop for a slice parameter, or a direct append for a scalar.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter list.
// Takes index (int) which is the parameter slice index.
//
// Returns ast.Stmt which appends the parameter's value(s) to args.
func flattenParameterStatement(query *querier_dto.AnalysedQuery, index int) ast.Stmt {
	paramExpr := paramAccessExpr(query, index)
	if query.Parameters[index].IsSlice {
		return &ast.RangeStmt{
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
		}
	}
	return goastutil.AssignStmt(
		goastutil.CachedIdent(identArgs),
		goastutil.CallExpr(
			goastutil.CachedIdent("append"),
			goastutil.CachedIdent(identArgs),
			paramExpr,
		),
	)
}

// BuildSliceDBCallArgs returns the DB call arguments [ctx, query, args...] for use in a
// method that has been rewritten with slice expansion. The returned CallExpr must have
// Ellipsis set on the call site.
//
// Returns []ast.Expr which contains the three argument expressions.
func BuildSliceDBCallArgs() []ast.Expr {
	return []ast.Expr{
		goastutil.CachedIdent(IdentCtx),
		goastutil.CachedIdent(IdentQuery),
		goastutil.CachedIdent(identArgs),
	}
}

// SliceDBCall constructs a database call with ellipsis on the args parameter for use with
// slice-expanded queries.
//
// Takes strategy (MethodStrategy) which provides the connection field accessor.
// Takes query (*querier_dto.AnalysedQuery) which identifies the target query.
// Takes method (string) which is the database method name to call.
//
// Returns *ast.CallExpr which is the constructed call expression.
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

// buildArgsMakeCall constructs make([]any, 0, <capacity>) where capacity is the sum of
// scalar parameter count plus len() calls for each slice parameter.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter metadata used to
// compute the capacity expression.
//
// Returns *ast.CallExpr which is the make call expression.
func buildArgsMakeCall(query *querier_dto.AnalysedQuery) *ast.CallExpr {
	var scalarCount int
	var sliceLens []ast.Expr

	for i := range query.Parameters {
		if query.Parameters[i].Kind == querier_dto.ParameterDirectiveSortable {
			continue
		}
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
		&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
		goastutil.IntLit(0),
		capExpr,
	)
}

// paramAccessExpr returns the AST expression to access a query parameter. For queries
// with a single non-slice parameter, the access is via the local variable name; for
// multi-parameter queries, it is via params.FieldName.
//
// Takes query (*querier_dto.AnalysedQuery) which provides the parameter list and slice
// metadata.
// Takes index (int) which is the zero-based position of the parameter to access.
//
// Returns ast.Expr which is the field or identifier access expression.
func paramAccessExpr(query *querier_dto.AnalysedQuery, index int) ast.Expr {
	if len(query.Parameters) == 1 && !HasSliceParameter(query) {
		return goastutil.CachedIdent(SnakeToCamelCase(query.Parameters[0].Name))
	}
	return goastutil.SelectorExprFrom(
		goastutil.CachedIdent(IdentParams),
		SnakeToPascalCase(query.Parameters[index].Name),
	)
}
