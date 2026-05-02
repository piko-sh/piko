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

package querier_domain

import (
	"context"

	"piko.sh/piko/internal/querier/querier_dto"
)

// parameterIdentity identifies a raw parameter reference by its display name, the column
// it compares against, and its number.
//
// It is the key used by the identity guard so a subquery parameter is only taken over by
// the subquery pass when the same parameter, not just the same number, also appears in
// the top-level flat list.
type parameterIdentity struct {
	// name is the parameter's display name.
	name string

	// tableAlias is the alias of the table the parameter compares against.
	tableAlias string

	// columnName is the column the parameter compares against.
	columnName string

	// number is the parameter's positional number.
	number int
}

// identityOf derives the parameterIdentity of a raw parameter reference.
//
// Takes reference (querier_dto.RawParameterReference) which is the parameter to key.
//
// Returns parameterIdentity which uniquely identifies the reference.
func identityOf(reference querier_dto.RawParameterReference) parameterIdentity {
	identity := parameterIdentity{number: reference.Number, name: reference.Name}
	if reference.ColumnReference != nil {
		identity.tableAlias = reference.ColumnReference.TableAlias
		identity.columnName = reference.ColumnReference.ColumnName
	}
	return identity
}

// parameterIdentitySet builds the set of identities present in the top-level flat
// parameter list.
//
// The subquery pass consults it to decide whether a nested parameter was flattened into
// this list by the engine (and so should be typed in its own scope here) or belongs to an
// engine that numbers nested parameters independently (and so must be left to the flat
// pass).
//
// Takes references ([]querier_dto.RawParameterReference) which is the top-level flat
// list.
//
// Returns map[parameterIdentity]bool keyed by every identity in the list.
func parameterIdentitySet(references []querier_dto.RawParameterReference) map[parameterIdentity]bool {
	set := make(map[parameterIdentity]bool, len(references))
	for _, reference := range references {
		set[identityOf(reference)] = true
	}
	return set
}

// subqueryParameterPass carries the accumulators and lookup sets threaded unchanged
// through the recursive subquery-parameter walk, keeping the recursion signature small.
type subqueryParameterPass struct {
	// parameterTypes accumulates resolved parameters keyed by number.
	parameterTypes map[int]*querier_dto.QueryParameter

	// directiveNumberMap carries directive overrides keyed by parameter number, used for
	// naming.
	directiveNumberMap map[int]*querier_dto.ParameterDirective

	// directiveNameMap carries directive overrides keyed by parameter name, used for naming.
	directiveNameMap map[string]*querier_dto.ParameterDirective

	// handled records the parameter numbers the subquery pass took over so the outer flat
	// pass skips them.
	handled map[int]bool

	// rootIdentities is the identity guard set: the identities present in the top-level flat
	// parameter list.
	rootIdentities map[parameterIdentity]bool
}

// resolveSubqueryParameters types every parameter that occurs inside a subquery of raw
// against a scope built from that subquery's own FROM tables.
//
// Each subquery scope is chained to scope so a correlated parameter that compares against
// an outer column still resolves (the parser leaves such a parameter unqualified; the
// inner scope is consulted first, then the outer chain). Each parameter is typed in the
// innermost subquery that owns it.
//
// The subqueries visited are the EXISTS and scalar subqueries reachable through the typed
// SELECT-list expression tree (at any nesting), the FROM-clause derived tables, the
// SELECT body of an INSERT ... SELECT, the CTE bodies that carry their own parameters,
// and the WHERE/HAVING/JOIN-ON predicate subqueries the engine records on
// PredicateSubqueries. A nested parameter is taken over here only when it passes the
// identity guard: its identity (number, name, and column reference) must be present in
// pass.rootIdentities, the top-level flat list. SQL engines flatten a subquery's
// parameters into that list unchanged, so the identity matches and the parameter is typed
// in its own scope. An engine that parses a derived table with an independent nested
// parser (ClickHouse) numbers its parameters from one again, so their identity is absent
// from the flat list, the guard declines, and an unrelated outer parameter sharing the
// same number is never suppressed.
//
// Every number taken over here is recorded in pass.handled so the outer flat pass in
// mergeRawParameters skips it, which both prevents a stale unknown-column diagnostic and
// stops the outer pass overwriting the correctly scoped type.
//
// Takes raw (*querier_dto.RawQueryAnalysis) which is the query whose subqueries are
// walked.
// Takes scope (*scopeChain) which is the enclosing scope (each subquery scope's parent).
// Takes pass (*subqueryParameterPass) which carries the accumulators and the identity
// guard.
// Takes depth (int) which bounds the subquery recursion.
//
// Returns []querier_dto.SourceError which holds diagnostics for genuinely unresolved
// subquery parameters (a column missing from the subquery's own scope).
func (r *typeResolver) resolveSubqueryParameters(
	ctx context.Context,
	raw *querier_dto.RawQueryAnalysis,
	scope *scopeChain,
	pass *subqueryParameterPass,
	depth int,
) []querier_dto.SourceError {
	if raw == nil || depth >= maxExpressionResolveDepth {
		return nil
	}

	var diagnostics []querier_dto.SourceError
	for _, subquery := range collectDirectSubqueries(raw) {
		if ctx.Err() != nil {
			return diagnostics
		}
		diagnostics = append(diagnostics, r.resolveOneSubqueryParameters(ctx, subquery, scope, pass, depth)...)
	}
	return diagnostics
}

// resolveOneSubqueryParameters types the parameters owned by a single direct subquery
// against a scope chained to the enclosing scope, then recurses into its own descendants.
//
// A parameter is taken over only when it is not owned by a deeper subquery and its
// identity is present in pass.rootIdentities (the identity guard on
// resolveSubqueryParameters). Every number taken over is recorded in pass.handled so the
// outer flat pass skips it.
//
// Takes subquery (*querier_dto.RawQueryAnalysis) which is the direct subquery to type.
// Takes scope (*scopeChain) which is the enclosing scope (the subquery scope's parent).
// Takes pass (*subqueryParameterPass) which carries the accumulators and the identity
// guard.
// Takes depth (int) which is the current recursion depth of the enclosing query.
//
// Returns []querier_dto.SourceError which holds diagnostics for genuinely unresolved
// subquery parameters.
func (r *typeResolver) resolveOneSubqueryParameters(
	ctx context.Context,
	subquery *querier_dto.RawQueryAnalysis,
	scope *scopeChain,
	pass *subqueryParameterPass,
	depth int,
) []querier_dto.SourceError {
	subqueryScope := newScopeChain(querier_dto.ScopeKindSubquery, scope)
	r.addRawTablesToScope(subquery, subqueryScope, depth)

	deeper := directSubqueryParameterNumbers(subquery)
	for _, rawParameter := range subquery.ParameterReferences {
		if deeper[rawParameter.Number] {
			continue
		}
		if !pass.rootIdentities[identityOf(rawParameter)] {
			continue
		}
		sqlType, nullable, resolveError := r.resolveParameterType(rawParameter, subqueryScope)

		if resolveError != nil || rawParameter.ColumnReference == nil {
			continue
		}
		pass.handled[rawParameter.Number] = true
		r.upsertParameterType(
			pass.parameterTypes, rawParameter, sqlType, nullable, pass.directiveNumberMap, pass.directiveNameMap,
		)
	}

	return r.resolveSubqueryParameters(ctx, subquery, subqueryScope, pass, depth+1)
}

// collectDirectSubqueries returns the subquery analyses one level inside raw.
//
// These are the EXISTS and scalar subqueries found in the SELECT-list expressions, plus
// the FROM-clause derived-table subqueries. Subqueries nested deeper are reached by
// recursing into the returned analyses. Resolving a derived table's parameters here is
// made safe by the identity guard in resolveSubqueryParameters, which declines any nested
// parameter not present in the top-level flat list (so an engine that numbers
// derived-table parameters independently is unaffected).
//
// Takes raw (*querier_dto.RawQueryAnalysis) which is the query to inspect.
//
// Returns []*querier_dto.RawQueryAnalysis which holds the immediate subquery analyses.
func collectDirectSubqueries(raw *querier_dto.RawQueryAnalysis) []*querier_dto.RawQueryAnalysis {
	if raw == nil {
		return nil
	}
	var subqueries []*querier_dto.RawQueryAnalysis
	for index := range raw.OutputColumns {
		appendExpressionSubqueries(raw.OutputColumns[index].Expression, &subqueries, 0)
	}
	for index := range raw.RawDerivedTables {
		if raw.RawDerivedTables[index].InnerQuery != nil {
			subqueries = append(subqueries, raw.RawDerivedTables[index].InnerQuery)
		}
	}

	for _, predicateSubquery := range raw.PredicateSubqueries {
		if predicateSubquery != nil {
			subqueries = append(subqueries, predicateSubquery)
		}
	}

	if raw.InsertSelect != nil {
		subqueries = append(subqueries, raw.InsertSelect)
	}

	for index := range raw.CTEDefinitions {
		definition := &raw.CTEDefinitions[index]
		if len(definition.ParameterReferences) == 0 {
			continue
		}
		subqueries = append(subqueries, &querier_dto.RawQueryAnalysis{
			FromTables:          definition.FromTables,
			JoinClauses:         definition.JoinClauses,
			ParameterReferences: definition.ParameterReferences,
		})
	}
	return subqueries
}

// directSubqueryParameterNumbers returns the set of parameter numbers that belong to any
// subquery one level inside raw. Because each engine parser flattens a subquery's own
// parameters together with its descendants' parameters into that subquery's
// ParameterReferences, this set covers every parameter nested below raw.
//
// Takes raw (*querier_dto.RawQueryAnalysis) which is the query to inspect.
//
// Returns map[int]bool keyed by the nested parameter numbers.
func directSubqueryParameterNumbers(raw *querier_dto.RawQueryAnalysis) map[int]bool {
	numbers := make(map[int]bool)
	for _, subquery := range collectDirectSubqueries(raw) {
		for _, parameter := range subquery.ParameterReferences {
			numbers[parameter.Number] = true
		}
	}
	return numbers
}

// appendExpressionSubqueries walks an expression tree and appends the EXISTS and scalar
// subquery analyses it contains to subqueries.
//
// It descends through every expression form that can carry an operand so a subquery
// nested inside (for example) a NOT, a CASE branch, or a function argument is still
// found. The subquery's own inner expression tree is not descended here; the caller
// recurses into the collected analyses instead.
//
// Takes expression (querier_dto.Expression) which is the expression to walk (may be nil).
// Takes subqueries (*[]*querier_dto.RawQueryAnalysis) which accumulates the found
// analyses.
// Takes depth (int) which bounds the walk at maxExpressionResolveDepth so a
// pathologically deep expression tree cannot overflow the goroutine stack during
// analysis.
func appendExpressionSubqueries(expression querier_dto.Expression, subqueries *[]*querier_dto.RawQueryAnalysis, depth int) {
	if depth >= maxExpressionResolveDepth {
		return
	}
	switch typed := expression.(type) {
	case *querier_dto.ExistsExpression:
		appendInnerQuery(typed.InnerQuery, subqueries)
	case *querier_dto.ScalarSubqueryExpression:
		appendInnerQuery(typed.InnerQuery, subqueries)
	case *querier_dto.CaseWhenExpression:
		appendCaseWhenSubqueries(typed, subqueries, depth+1)
	default:
		appendExpressionListSubqueries(subqueryBearingOperands(expression), subqueries, depth+1)
	}
}

// appendInnerQuery appends a subquery analysis when it is present.
//
// Takes inner (*querier_dto.RawQueryAnalysis) which is the subquery analysis (may be
// nil).
// Takes subqueries (*[]*querier_dto.RawQueryAnalysis) which accumulates the found
// analyses.
func appendInnerQuery(inner *querier_dto.RawQueryAnalysis, subqueries *[]*querier_dto.RawQueryAnalysis) {
	if inner != nil {
		*subqueries = append(*subqueries, inner)
	}
}

// subqueryBearingOperands returns the operand sub-expressions of an expression that can
// structurally hold a nested subquery, so the caller can recurse into them.
//
// EXISTS, scalar subqueries, and CASE are handled by appendExpressionSubqueries directly
// and so are absent here. The multi-operand and single-operand forms are split across two
// helpers to keep each switch small.
//
// Takes expression (querier_dto.Expression) which is the expression to inspect.
//
// Returns []querier_dto.Expression which holds the operands to walk (nil for leaf forms).
func subqueryBearingOperands(expression querier_dto.Expression) []querier_dto.Expression {
	if operands, matched := multiOperandSubexpressions(expression); matched {
		return operands
	}
	return singleOperandSubexpressions(expression)
}

// multiOperandSubexpressions returns the operands of expression forms that carry several
// sub-expressions.
//
// Takes expression (querier_dto.Expression) which is the expression to inspect.
//
// Returns []querier_dto.Expression which holds the operands.
// Returns bool which reports whether the expression matched a multi-operand form.
func multiOperandSubexpressions(expression querier_dto.Expression) ([]querier_dto.Expression, bool) {
	switch typed := expression.(type) {
	case *querier_dto.FunctionCallExpression:
		return append([]querier_dto.Expression{typed.FilterExpression}, typed.Arguments...), true
	case *querier_dto.CoalesceExpression:
		return typed.Arguments, true
	case *querier_dto.BinaryOpExpression:
		return []querier_dto.Expression{typed.Left, typed.Right}, true
	case *querier_dto.ComparisonExpression:
		return []querier_dto.Expression{typed.Left, typed.Right}, true
	case *querier_dto.InListExpression:
		return append([]querier_dto.Expression{typed.Inner}, typed.Values...), true
	case *querier_dto.BetweenExpression:
		return []querier_dto.Expression{typed.Inner, typed.Low, typed.High}, true
	case *querier_dto.LogicalOpExpression:
		return typed.Operands, true
	case *querier_dto.ArraySubscriptExpression:
		return []querier_dto.Expression{typed.Array, typed.Index}, true
	default:
		return nil, false
	}
}

// singleOperandSubexpressions returns the operand of expression forms that carry exactly
// one sub-expression. A window function with a nil underlying call yields no operands.
//
// Takes expression (querier_dto.Expression) which is the expression to inspect.
//
// Returns []querier_dto.Expression which holds the operand (nil for leaf forms).
func singleOperandSubexpressions(expression querier_dto.Expression) []querier_dto.Expression {
	switch typed := expression.(type) {
	case *querier_dto.CastExpression:
		return []querier_dto.Expression{typed.Inner}
	case *querier_dto.IsNullExpression:
		return []querier_dto.Expression{typed.Inner}
	case *querier_dto.UnaryOpExpression:
		return []querier_dto.Expression{typed.Operand}
	case *querier_dto.LambdaExpression:
		return []querier_dto.Expression{typed.Body}
	case *querier_dto.StructFieldAccessExpression:
		return []querier_dto.Expression{typed.Struct}
	case *querier_dto.WindowFunctionExpression:
		if typed.Function == nil {
			return nil
		}
		return []querier_dto.Expression{typed.Function}
	default:
		return nil
	}
}

// appendExpressionListSubqueries applies appendExpressionSubqueries to each expression in
// a slice. It keeps the per-form cases in appendExpressionSubqueries small.
//
// Takes expressions ([]querier_dto.Expression) which is the slice to walk.
// Takes subqueries (*[]*querier_dto.RawQueryAnalysis) which accumulates the found
// analyses.
// Takes depth (int) which bounds the recursion at maxExpressionResolveDepth.
func appendExpressionListSubqueries(expressions []querier_dto.Expression, subqueries *[]*querier_dto.RawQueryAnalysis, depth int) {
	for _, expression := range expressions {
		appendExpressionSubqueries(expression, subqueries, depth)
	}
}

// appendCaseWhenSubqueries walks the conditions, results, and ELSE of a CASE expression.
//
// Takes expression (*querier_dto.CaseWhenExpression) which is the CASE to walk.
// Takes subqueries (*[]*querier_dto.RawQueryAnalysis) which accumulates the found
// analyses.
// Takes depth (int) which bounds the recursion at maxExpressionResolveDepth.
func appendCaseWhenSubqueries(expression *querier_dto.CaseWhenExpression, subqueries *[]*querier_dto.RawQueryAnalysis, depth int) {
	for index := range expression.Branches {
		appendExpressionSubqueries(expression.Branches[index].Condition, subqueries, depth)
		appendExpressionSubqueries(expression.Branches[index].Result, subqueries, depth)
	}
	appendExpressionSubqueries(expression.ElseResult, subqueries, depth)
}
