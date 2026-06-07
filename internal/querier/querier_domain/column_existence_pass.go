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
	"errors"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// columnExistencePass surfaces unknown or ambiguous column references that are nested
// inside an output column's expression tree.
//
// The type resolver computes expression types leniently: a CASE, predicate, binary op,
// function argument or COALESCE still yields a type even when one of its operands
// references a column that does not exist, because each combinator discards the operand's
// resolution error so a usable type is always produced for codegen. A genuine typo such
// as a reference to a column the view does not expose therefore reaches runtime with no
// compile-time warning.
//
// This read-only pass closes that gap. It runs after type resolution (so the scope
// already carries every table and view column) and walks each output column's raw
// expression tree, validating every column reference against the same lenient gate the
// parameter resolver uses. It never modifies the typing path, so it can only add
// SeverityWarning diagnostics: codegen still succeeds.
type columnExistencePass struct {
	// catalogue backs the catalogue-wide fallback lookup that keeps the pass silent on
	// columns the engine adapter flat-scanned out of a subquery, matching the parameter
	// resolver's leniency.
	catalogue *querier_dto.Catalogue
}

// Analyse walks every output column's expression tree and reports each genuine unknown
// (Q001) or ambiguous (Q002) column reference as a warning.
//
// Takes context (*diagnosticContext) which carries the query, scope, raw analysis and the
// diagnostics already produced so duplicates are suppressed.
//
// Returns []querier_dto.SourceError which holds the unknown or ambiguous column warnings,
// or nil when the scope or analysis are absent or every reference resolves.
func (p *columnExistencePass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	if context == nil || context.Scope == nil || context.RawAnalysis == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(context.ExistingDiagnostics))
	for index := range context.ExistingDiagnostics {
		seen[context.ExistingDiagnostics[index].Message] = struct{}{}
	}

	var diagnostics []querier_dto.SourceError
	for _, column := range context.RawAnalysis.OutputColumns {
		if _, isDirectColumnRef := column.Expression.(*querier_dto.ColumnRefExpression); isDirectColumnRef {
			continue
		}
		p.walkExpressionColumns(column.Expression, context, seen, &diagnostics, 0)
	}
	return diagnostics
}

// walkExpressionColumns mirrors collectFromExpression's traversal but gates each
// ColumnRefExpression through gateUnknownColumn instead of recording a function name. It
// stops at the positions where a column reference does not denote a catalogue column.
//
// It deliberately does NOT gate:
//   - scalar or EXISTS subquery projections, whose references resolve against an inner
//     scope this pass does not hold, so descending would raise false positives on
//     correlated columns;
//   - lambda bodies, whose parameter occurrences the engine parser re-emits as bare
//     column references indistinguishable from real columns;
//   - function-call arguments, the position where the engine parsers collapse a
//     named-argument label (the := and => forms) into a bare or comparison-wrapped column
//     reference that is likewise indistinguishable from a genuine column.
//
// A FILTER predicate, a CASE, a predicate, a boolean operand, a COALESCE or an arithmetic
// operand is a genuine column position and is still gated, so a real typo there is still
// surfaced. Honouring the never-false-positive guarantee is worth missing the rare typo
// nested directly inside a lambda body or a function argument.
//
// Takes expression (querier_dto.Expression) which is the expression subtree to walk.
// Takes context (*diagnosticContext) which carries the scope and catalogue for gating.
// Takes seen (map[string]struct{}) which deduplicates already-reported diagnostics.
// Takes diagnostics (*[]querier_dto.SourceError) which accumulates the emitted warnings.
// Takes depth (int) which bounds recursion at maxExpressionResolveDepth.
//
//nolint:revive // expression dispatch mirrors collectFromExpression's type switch
func (p *columnExistencePass) walkExpressionColumns(
	expression querier_dto.Expression,
	context *diagnosticContext,
	seen map[string]struct{},
	diagnostics *[]querier_dto.SourceError,
	depth int,
) {
	if expression == nil || depth >= maxExpressionResolveDepth {
		return
	}

	switch typed := expression.(type) {
	case *querier_dto.ColumnRefExpression:
		if diagnostic, ok := p.gateUnknownColumn(context, typed, seen); ok {
			*diagnostics = append(*diagnostics, *diagnostic)
		}
	case *querier_dto.FunctionCallExpression:
		p.walkExpressionColumns(typed.FilterExpression, context, seen, diagnostics, depth+1)
	case *querier_dto.BinaryOpExpression:
		p.walkExpressionColumns(typed.Left, context, seen, diagnostics, depth+1)
		p.walkExpressionColumns(typed.Right, context, seen, diagnostics, depth+1)
	case *querier_dto.ComparisonExpression:
		p.walkExpressionColumns(typed.Left, context, seen, diagnostics, depth+1)
		p.walkExpressionColumns(typed.Right, context, seen, diagnostics, depth+1)
	case *querier_dto.UnaryOpExpression:
		p.walkExpressionColumns(typed.Operand, context, seen, diagnostics, depth+1)
	case *querier_dto.CoalesceExpression:
		for _, argument := range typed.Arguments {
			p.walkExpressionColumns(argument, context, seen, diagnostics, depth+1)
		}
	case *querier_dto.CastExpression:
		p.walkExpressionColumns(typed.Inner, context, seen, diagnostics, depth+1)
	case *querier_dto.IsNullExpression:
		p.walkExpressionColumns(typed.Inner, context, seen, diagnostics, depth+1)
	case *querier_dto.InListExpression:
		p.walkExpressionColumns(typed.Inner, context, seen, diagnostics, depth+1)
		for _, value := range typed.Values {
			p.walkExpressionColumns(value, context, seen, diagnostics, depth+1)
		}
	case *querier_dto.BetweenExpression:
		p.walkExpressionColumns(typed.Inner, context, seen, diagnostics, depth+1)
		p.walkExpressionColumns(typed.Low, context, seen, diagnostics, depth+1)
		p.walkExpressionColumns(typed.High, context, seen, diagnostics, depth+1)
	case *querier_dto.LogicalOpExpression:
		for _, operand := range typed.Operands {
			p.walkExpressionColumns(operand, context, seen, diagnostics, depth+1)
		}
	case *querier_dto.CaseWhenExpression:
		p.walkExpressionColumns(typed.ElseResult, context, seen, diagnostics, depth+1)
		for _, branch := range typed.Branches {
			p.walkExpressionColumns(branch.Condition, context, seen, diagnostics, depth+1)
			p.walkExpressionColumns(branch.Result, context, seen, diagnostics, depth+1)
		}
	case *querier_dto.WindowFunctionExpression:
		if typed.Function != nil {
			p.walkExpressionColumns(typed.Function, context, seen, diagnostics, depth+1)
		}
	case *querier_dto.ArraySubscriptExpression:
		p.walkExpressionColumns(typed.Array, context, seen, diagnostics, depth+1)
		p.walkExpressionColumns(typed.Index, context, seen, diagnostics, depth+1)
	case *querier_dto.StructFieldAccessExpression:
		p.walkExpressionColumns(typed.Struct, context, seen, diagnostics, depth+1)
	case *querier_dto.LambdaExpression, *querier_dto.ScalarSubqueryExpression,
		*querier_dto.ExistsExpression, *querier_dto.LiteralExpression,
		*querier_dto.UnknownExpression:

	default:
	}
}

// gateUnknownColumn resolves a single column reference with the same three-step leniency
// the parameter resolver applies in resolveColumnReferencedParameterType.
//
// A reference that resolves in scope, recovers via the catalogue-wide lookup, or recovers
// via the bare-column fallback is silent. A surviving scope-depth or internal-nil-guard
// error is also silent: those are not user-facing unknown columns, and because the depth
// sentinel carries no Q-code prefix extractErrorCode would otherwise mislabel it Q001.
// Anything else is surfaced once per (alias, column) as a warning, preserving the
// Q001/Q002 code already encoded in the scope error's message prefix.
//
// Takes context (*diagnosticContext) which provides the scope, catalogue and filename.
// Takes expression (*querier_dto.ColumnRefExpression) which is the column reference to
// gate.
// Takes seen (map[string]struct{}) which deduplicates already-reported diagnostics.
//
// Returns *querier_dto.SourceError which is the warning for a genuine unknown or
// ambiguous column, or nil when the reference resolves or is suppressed.
// Returns bool which is true when a warning was produced.
func (p *columnExistencePass) gateUnknownColumn(
	context *diagnosticContext,
	expression *querier_dto.ColumnRefExpression,
	seen map[string]struct{},
) (*querier_dto.SourceError, bool) {
	if expression == nil || expression.ColumnName == "" {
		return nil, false
	}

	_, _, err := context.Scope.ResolveColumn(expression.TableAlias, expression.ColumnName)
	if err == nil {
		return nil, false
	}

	reference := &querier_dto.ColumnReference{
		TableAlias: expression.TableAlias,
		ColumnName: expression.ColumnName,
	}
	if _, ok := findColumnInCatalogueFor(p.catalogue, reference); ok {
		return nil, false
	}
	if _, ok := resolveBareColumnFallback(context.Scope, reference); ok {
		return nil, false
	}
	if columnNameExistsInCatalogue(p.catalogue, expression.ColumnName) {
		return nil, false
	}

	if errors.Is(err, errScopeChainDepthExceeded) {
		return nil, false
	}
	if strings.HasPrefix(err.Error(), querier_dto.CodeInternalNilGuard) {
		return nil, false
	}

	message := err.Error()
	if _, exists := seen[message]; exists {
		return nil, false
	}
	seen[message] = struct{}{}

	line := 0
	if context.Query != nil {
		line = context.Query.Line
	}
	return &querier_dto.SourceError{
		Filename: context.Filename,
		Line:     line,
		Column:   1,
		Message:  message,
		Severity: querier_dto.SeverityWarning,
		Code:     extractErrorCode(err),
	}, true
}

// columnNameExistsInCatalogue reports whether columnName is the name of a column on any
// table or view in the catalogue, ignoring the qualifying alias.
//
// The pass uses it as a final suppression. A query may legitimately reference a column
// the active scope does not model: the output of a table-valued function, or a view whose
// body the catalogue could only partially resolve. If the bare name is known on some
// relation, the reference is almost certainly one of those cases rather than a typo, so
// the pass stays silent. A genuinely misspelled column - a name that appears on no
// relation at all, such as version_role where only version_role_id exists - is still
// surfaced.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state to search.
// Takes columnName (string) which is the bare column name to look for.
//
// Returns bool which is true when some table or view exposes a column of that name.
func columnNameExistsInCatalogue(catalogue *querier_dto.Catalogue, columnName string) bool {
	if catalogue == nil || columnName == "" {
		return false
	}
	for _, schema := range catalogue.Schemas {
		if schemaHasColumnNamed(schema, columnName) {
			return true
		}
	}
	return false
}

// schemaHasColumnNamed reports whether any table or view in the schema exposes a column
// named columnName, ignoring the qualifying alias.
//
// Takes schema (*querier_dto.Schema) which is the schema to scan.
// Takes columnName (string) which is the bare column name to look for.
//
// Returns bool which is true when a table or view in the schema has the column.
func schemaHasColumnNamed(schema *querier_dto.Schema, columnName string) bool {
	if schema == nil {
		return false
	}
	for _, table := range schema.Tables {
		if table == nil {
			continue
		}
		if _, ok := findColumnInTable(table, columnName); ok {
			return true
		}
	}
	for _, view := range schema.Views {
		if viewHasColumnNamed(view, columnName) {
			return true
		}
	}
	return false
}

// viewHasColumnNamed reports whether the view exposes a column named columnName.
//
// Takes view (*querier_dto.View) which is the view to scan.
// Takes columnName (string) which is the bare column name to look for.
//
// Returns bool which is true when the view has a column of that name.
func viewHasColumnNamed(view *querier_dto.View, columnName string) bool {
	if view == nil {
		return false
	}
	for index := range view.Columns {
		if strings.EqualFold(view.Columns[index].Name, columnName) {
			return true
		}
	}
	return false
}
