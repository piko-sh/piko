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
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// collectFunctionCalls extracts the unique set of function names referenced in the output
// columns of an analysis.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which holds the raw query analysis with
// output column expressions.
//
// Returns []string which holds the unique function names, or nil if none are found.
func collectFunctionCalls(analysis *querier_dto.RawQueryAnalysis) []string {
	if analysis == nil {
		return nil
	}

	seen := make(map[string]struct{})
	collectFromAnalysis(analysis, seen, 0)

	if len(seen) == 0 {
		return nil
	}

	return slices.Sorted(maps.Keys(seen))
}

// collectFromAnalysis walks every expression and nested subquery reachable from a raw
// analysis, recording the function names each references.
//
// It descends into INSERT ... SELECT bodies, compound (UNION/INTERSECT/EXCEPT) branches,
// CTE bodies, FROM-clause derived subqueries, and predicate subqueries so a function that
// writes data transitively through a subquery is still discovered and the caller is
// correctly classified as data-modifying rather than routed to a read replica.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which is the analysis to walk.
// Takes seen (map[string]struct{}) which accumulates the discovered function names.
// Takes depth (int) which bounds recursion through nested subqueries; the ceiling matches
// the resolver's maxExpressionResolveDepth so a pathologically deep query cannot overflow
// the goroutine stack during catalogue build.
func collectFromAnalysis(analysis *querier_dto.RawQueryAnalysis, seen map[string]struct{}, depth int) {
	if analysis == nil || depth >= maxExpressionResolveDepth {
		return
	}

	for _, column := range analysis.OutputColumns {
		collectFromExpression(column.Expression, seen, depth+1)
	}
	collectFromAnalysis(analysis.InsertSelect, seen, depth+1)
	for _, branch := range analysis.CompoundBranches {
		collectFromAnalysis(branch.Query, seen, depth+1)
	}
	collectFromCTEDefinitions(analysis.CTEDefinitions, seen, depth+1)
	for _, subquery := range analysis.PredicateSubqueries {
		collectFromAnalysis(subquery, seen, depth+1)
	}
	for index := range analysis.RawDerivedTables {
		collectFromAnalysis(analysis.RawDerivedTables[index].InnerQuery, seen, depth+1)
	}
}

// collectFromCTEDefinitions records the function names referenced in each CTE body's
// output columns and compound branches.
//
// Takes definitions ([]querier_dto.RawCTEDefinition) which are the CTE bodies to walk.
// Takes seen (map[string]struct{}) which accumulates the discovered function names.
// Takes depth (int) which bounds the recursion at maxExpressionResolveDepth; the CTE hop
// advances the counter (depth+1) like every other descent in the walker so a pathological
// CTE nesting cannot bypass the ceiling.
func collectFromCTEDefinitions(definitions []querier_dto.RawCTEDefinition, seen map[string]struct{}, depth int) {
	for index := range definitions {
		definition := &definitions[index]
		for _, column := range definition.OutputColumns {
			collectFromExpression(column.Expression, seen, depth+1)
		}
		for _, branch := range definition.CompoundBranches {
			collectFromAnalysis(branch.Query, seen, depth+1)
		}
	}
}

// collectFromExpression recursively walks an expression tree and records all function
// call names in the seen map, descending into scalar/EXISTS subqueries and lambda bodies.
//
// Takes expression (querier_dto.Expression) which is the expression tree to walk.
// Takes seen (map[string]struct{}) which accumulates the discovered function names.
// Takes depth (int) which bounds recursion at maxExpressionResolveDepth so a deeply
// nested expression tree from a user-authored function body cannot overflow the goroutine
// stack.
//
//nolint:revive // expression dispatch
func collectFromExpression(expression querier_dto.Expression, seen map[string]struct{}, depth int) {
	if expression == nil || depth >= maxExpressionResolveDepth {
		return
	}

	switch typed := expression.(type) {
	case *querier_dto.FunctionCallExpression:
		name := strings.ToLower(typed.FunctionName)
		if typed.Schema != "" {
			name = strings.ToLower(typed.Schema) + "." + name
		}
		seen[name] = struct{}{}
		for _, argument := range typed.Arguments {
			collectFromExpression(argument, seen, depth+1)
		}
		collectFromExpression(typed.FilterExpression, seen, depth+1)
	case *querier_dto.BinaryOpExpression:
		collectFromExpression(typed.Left, seen, depth+1)
		collectFromExpression(typed.Right, seen, depth+1)
	case *querier_dto.ComparisonExpression:
		collectFromExpression(typed.Left, seen, depth+1)
		collectFromExpression(typed.Right, seen, depth+1)
	case *querier_dto.UnaryOpExpression:
		collectFromExpression(typed.Operand, seen, depth+1)
	case *querier_dto.CoalesceExpression:
		for _, argument := range typed.Arguments {
			collectFromExpression(argument, seen, depth+1)
		}
	case *querier_dto.CastExpression:
		collectFromExpression(typed.Inner, seen, depth+1)
	case *querier_dto.IsNullExpression:
		collectFromExpression(typed.Inner, seen, depth+1)
	case *querier_dto.InListExpression:
		collectFromExpression(typed.Inner, seen, depth+1)
		for _, value := range typed.Values {
			collectFromExpression(value, seen, depth+1)
		}
	case *querier_dto.BetweenExpression:
		collectFromExpression(typed.Inner, seen, depth+1)
		collectFromExpression(typed.Low, seen, depth+1)
		collectFromExpression(typed.High, seen, depth+1)
	case *querier_dto.LogicalOpExpression:
		for _, operand := range typed.Operands {
			collectFromExpression(operand, seen, depth+1)
		}
	case *querier_dto.CaseWhenExpression:
		collectFromExpression(typed.ElseResult, seen, depth+1)
		for _, branch := range typed.Branches {
			collectFromExpression(branch.Condition, seen, depth+1)
			collectFromExpression(branch.Result, seen, depth+1)
		}
	case *querier_dto.WindowFunctionExpression:
		if typed.Function != nil {
			collectFromExpression(typed.Function, seen, depth+1)
		}
	case *querier_dto.ArraySubscriptExpression:
		collectFromExpression(typed.Array, seen, depth+1)
		collectFromExpression(typed.Index, seen, depth+1)
	case *querier_dto.LambdaExpression:
		collectFromExpression(typed.Body, seen, depth+1)
	case *querier_dto.StructFieldAccessExpression:
		collectFromExpression(typed.Struct, seen, depth+1)
	case *querier_dto.ScalarSubqueryExpression:
		collectFromAnalysis(typed.InnerQuery, seen, depth+1)
	case *querier_dto.ExistsExpression:
		collectFromAnalysis(typed.InnerQuery, seen, depth+1)
	case *querier_dto.ColumnRefExpression, *querier_dto.LiteralExpression, *querier_dto.UnknownExpression:

	default:

	}
}
