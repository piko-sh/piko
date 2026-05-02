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
	"fmt"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// errorCodeMinLength holds the minimum message length required to contain a Q-code
	// prefix.
	errorCodeMinLength = 5

	// errorCodePrefixLength holds the number of characters in a Q-code prefix (e.g. "Q001").
	errorCodePrefixLength = 4

	// maxInferExpressionNameDepth caps how deep CAST and COALESCE chains may recurse during
	// column-name inference. Hand-crafted SQL can never push past a handful of levels in
	// practice, but a maliciously deep parser output should not blow the Go stack here.
	maxInferExpressionNameDepth = 64

	// maxExpressionResolveDepth bounds recursion through resolveExpressionType. A deeply
	// nested expression tree (mirroring the engine parse-depth cap) would otherwise overflow
	// the stack.
	maxExpressionResolveDepth = 256
)

// typeResolver performs bottom-up expression type inference, resolving raw output columns
// and parameter references from the engine adapter into fully typed results. This is
// where column references are looked up in the scope chain and function calls go through
// overload resolution.
type typeResolver struct {
	// catalogue holds the database catalogue for table and type lookups.
	catalogue *querier_dto.Catalogue

	// functionResolver holds the function overload resolver for function call expressions.
	functionResolver *functionResolver

	// engine holds the engine adapter for type normalisation, promotion, and casting rules.
	engine EngineTypeSystemPort

	// expressionDepth bounds recursion through resolveExpressionType so a deeply nested
	// expression tree cannot overflow the goroutine stack (a fatal, non-recoverable error).
	// The increment/decrement is balanced per call, so reusing a resolver across queries is
	// safe.
	expressionDepth int
}

// catalogueColumnMatch holds the type information for a column found by a catalogue-wide
// lookup.
type catalogueColumnMatch struct {
	// sqlType is the resolved SQL type of the matched column.
	sqlType querier_dto.SQLType

	// nullable indicates whether the matched column accepts NULL values.
	nullable bool
}

// newTypeResolver creates a type resolver with the given catalogue, function resolver,
// and engine adapter.
//
// Takes catalogue (*querier_dto.Catalogue) which specifies the database catalogue for
// lookups.
// Takes functionResolver (*functionResolver) which specifies the function overload
// resolver.
// Takes engine (EngineTypeSystemPort) which specifies the engine adapter for type system
// rules.
//
// Returns *typeResolver which holds the initialised type resolver.
func newTypeResolver(
	catalogue *querier_dto.Catalogue,
	functionResolver *functionResolver,
	engine EngineTypeSystemPort,
) *typeResolver {
	return &typeResolver{
		catalogue:        catalogue,
		functionResolver: functionResolver,
		engine:           engine,
	}
}

// ResolveOutputColumns resolves raw output columns from the engine adapter into fully
// typed output columns using the scope chain for column lookups and function resolver for
// expression types.
//
// Takes rawColumns ([]querier_dto.RawOutputColumn) which specifies the unresolved output
// columns.
//
// Returns []querier_dto.OutputColumn which holds the fully typed output columns.
// Returns bool which indicates whether any expression modifies data.
// Returns []querier_dto.SourceError which holds any diagnostics from resolution failures.
func (r *typeResolver) ResolveOutputColumns(
	ctx context.Context,
	rawColumns []querier_dto.RawOutputColumn,
	scope *scopeChain,
) ([]querier_dto.OutputColumn, bool, []querier_dto.SourceError) {
	ctx, span, _ := log.Span(ctx, "TypeResolver.ResolveOutputColumns")
	defer span.End()

	var resolved []querier_dto.OutputColumn
	var diagnostics []querier_dto.SourceError
	var dataModifying bool

	for _, raw := range rawColumns {
		if err := ctx.Err(); err != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  fmt.Sprintf("output column resolution stopped before completion: %v", err),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeInternalNilGuard,
			})
			return resolved, dataModifying, diagnostics
		}
		columns, diagnostic := r.resolveSingleOutputColumn(raw, scope, &dataModifying)
		if diagnostic != nil {
			diagnostics = append(diagnostics, *diagnostic)
		}

		resolved = append(resolved, columns...)
	}

	return resolved, dataModifying, diagnostics
}

// ResolveParameters resolves raw parameter references from the engine adapter into fully
// typed query parameters using the scope chain for context-based type inference and
// directive declarations for overrides.
//
// Takes rawAnalysis (*querier_dto.RawQueryAnalysis) which holds the unresolved query: its
// flat parameter list plus the subqueries whose parameters need their own scope.
// Takes parameterDirectives ([]*querier_dto.ParameterDirective) which specifies the
// directive overrides for parameters.
//
// Returns []querier_dto.QueryParameter which holds the fully typed parameters in ordinal
// order.
// Returns []querier_dto.SourceError which holds any diagnostics from resolution failures.
func (r *typeResolver) ResolveParameters(
	ctx context.Context,
	rawAnalysis *querier_dto.RawQueryAnalysis,
	scope *scopeChain,
	parameterDirectives []*querier_dto.ParameterDirective,
) ([]querier_dto.QueryParameter, []querier_dto.SourceError) {
	ctx, span, _ := log.Span(ctx, "TypeResolver.ResolveParameters")
	defer span.End()

	directiveNumberMap := make(map[int]*querier_dto.ParameterDirective, len(parameterDirectives))
	directiveNameMap := make(map[string]*querier_dto.ParameterDirective, len(parameterDirectives))
	for _, directive := range parameterDirectives {
		directiveNumberMap[directive.Number] = directive
		if directive.DirectiveName != "" {
			directiveNameMap[directive.DirectiveName] = directive
		}
	}

	parameterTypes := make(map[int]*querier_dto.QueryParameter)

	var rawParameters []querier_dto.RawParameterReference
	if rawAnalysis != nil {
		rawParameters = rawAnalysis.ParameterReferences
	}

	subqueryParameters := make(map[int]bool)
	pass := &subqueryParameterPass{
		parameterTypes:     parameterTypes,
		directiveNumberMap: directiveNumberMap,
		directiveNameMap:   directiveNameMap,
		handled:            subqueryParameters,
		rootIdentities:     parameterIdentitySet(rawParameters),
	}
	diagnostics := r.resolveSubqueryParameters(ctx, rawAnalysis, scope, pass, 0)

	diagnostics = append(diagnostics, r.mergeRawParameters(rawParameters, scope, directiveNumberMap, directiveNameMap, parameterTypes, subqueryParameters)...)

	r.applyParameterDirectives(parameterTypes, parameterDirectives)

	parameters := collectParameters(parameterTypes)
	disambiguateParameterNames(parameters)
	return parameters, diagnostics
}

// resolveSingleOutputColumn resolves a single raw output column into one or more typed
// output columns. Star expressions expand to multiple columns; column references and
// expressions each produce one.
//
// Takes raw (querier_dto.RawOutputColumn) which specifies the unresolved output column.
// Takes scope (*scopeChain) which specifies the scope chain for column lookups.
// Takes dataModifying (*bool) which specifies a flag set to true if the expression
// modifies data.
//
// Returns []querier_dto.OutputColumn which holds the resolved output columns.
// Returns *querier_dto.SourceError which holds a diagnostic if resolution failed, or nil
// on success.
func (r *typeResolver) resolveSingleOutputColumn(
	raw querier_dto.RawOutputColumn,
	scope *scopeChain,
	dataModifying *bool,
) ([]querier_dto.OutputColumn, *querier_dto.SourceError) {
	if raw.IsStar {
		starColumns, starError := r.expandStar(raw.TableAlias, scope)
		if starError != nil {
			return nil, &querier_dto.SourceError{
				Message:  starError.Error(),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownTable,
			}
		}
		return starColumns, nil
	}

	if raw.ColumnName != "" {
		return r.resolveColumnRefOutput(raw, scope)
	}

	return r.resolveExpressionOutput(raw, scope, dataModifying)
}

// resolveColumnRefOutput resolves a column reference output by looking up the column in
// the scope chain and building an output column with source metadata.
//
// Takes raw (querier_dto.RawOutputColumn) which specifies the raw column reference.
// Takes scope (*scopeChain) which specifies the scope chain for column lookups.
//
// Returns []querier_dto.OutputColumn which holds the resolved output column.
// Returns *querier_dto.SourceError which holds a diagnostic if resolution failed, or nil
// on success.
func (*typeResolver) resolveColumnRefOutput(
	raw querier_dto.RawOutputColumn,
	scope *scopeChain,
) ([]querier_dto.OutputColumn, *querier_dto.SourceError) {
	column, table, resolveError := scope.ResolveColumn(raw.TableAlias, raw.ColumnName)
	if resolveError != nil {
		return nil, &querier_dto.SourceError{
			Message:  resolveError.Error(),
			Severity: querier_dto.SeverityWarning,
			Code:     extractErrorCode(resolveError),
		}
	}
	if column == nil {
		return nil, &querier_dto.SourceError{
			Message:  "Q030: nil column resolved for " + raw.ColumnName,
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeInternalNilGuard,
		}
	}

	outputName := raw.Name
	if outputName == "" {
		outputName = raw.ColumnName
	}

	sourceTable := ""
	sourceSchema := ""
	sourceQualifier := ""
	if table != nil {
		sourceTable = table.Name
		sourceSchema = table.Schema
		sourceQualifier = table.Alias
	}

	return []querier_dto.OutputColumn{{
		Name:            outputName,
		SQLType:         column.SQLType,
		Nullable:        column.Nullable,
		SourceTable:     sourceTable,
		SourceSchema:    sourceSchema,
		SourceColumn:    column.Name,
		SourceQualifier: sourceQualifier,
	}}, nil
}

// resolveExpressionOutput resolves an expression-based output column by inferring its
// type through the expression type resolver.
//
// Takes raw (querier_dto.RawOutputColumn) which specifies the raw expression column.
// Takes scope (*scopeChain) which specifies the scope chain for expression resolution.
// Takes dataModifying (*bool) which specifies a flag set to true if the expression
// modifies data.
//
// Returns []querier_dto.OutputColumn which holds the resolved output column.
// Returns *querier_dto.SourceError which holds a diagnostic if resolution failed, or nil
// on success.
func (r *typeResolver) resolveExpressionOutput(
	raw querier_dto.RawOutputColumn,
	scope *scopeChain,
	dataModifying *bool,
) ([]querier_dto.OutputColumn, *querier_dto.SourceError) {
	sqlType, nullable, expressionError := r.resolveExpressionType(raw.Expression, scope, dataModifying)

	var diagnostic *querier_dto.SourceError
	if expressionError != nil {
		sqlType = querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
		nullable = true
		diagnostic = &querier_dto.SourceError{
			Message:  expressionError.Error(),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeExpressionTypeError,
		}
	}

	outputName := raw.Name
	if outputName == "" {
		outputName = inferExpressionName(raw.Expression)
	}
	if outputName == "" {
		outputName = "?column?"
	}

	output := querier_dto.OutputColumn{
		Name:     outputName,
		SQLType:  sqlType,
		Nullable: nullable,
	}
	applyCastColumnSource(&output, raw.Expression, scope)

	return []querier_dto.OutputColumn{output}, diagnostic
}

// applyCastColumnSource records the underlying source column of an output expression that
// is a column reference wrapped in nothing but CASTs, so a cast-only projection such as
// page_id::content.uuid_v4 keeps the source metadata a bare column reference would carry.
//
// A CAST only relabels a column's type, so a cast over a column remains the same filterable
// and orderable column: it must stay in a dynamic runtime builder's allow-list and resolve
// the same nullability and array-wrap source as the bare column would. Any other expression
// shape, such as a function call, an arithmetic operand or a COALESCE, is not the column and
// is left without a source.
//
// Takes output (*querier_dto.OutputColumn) which receives the resolved source metadata.
// Takes expression (querier_dto.Expression) which is the output column's expression.
// Takes scope (*scopeChain) which resolves the underlying column.
func applyCastColumnSource(
	output *querier_dto.OutputColumn,
	expression querier_dto.Expression,
	scope *scopeChain,
) {
	reference := unwrapCastToColumnRef(expression)
	if reference == nil {
		return
	}
	column, table, resolveError := scope.ResolveColumn(reference.TableAlias, reference.ColumnName)
	if resolveError != nil || column == nil {
		return
	}
	output.SourceColumn = column.Name
	if table != nil {
		output.SourceTable = table.Name
		output.SourceSchema = table.Schema
		output.SourceQualifier = table.Alias
	}
}

// unwrapCastToColumnRef peels any chain of CAST expressions and returns the column
// reference they wrap, or nil when the expression is not a column reference behind zero or
// more casts.
//
// Takes expression (querier_dto.Expression) which is the expression to peel.
//
// Returns *querier_dto.ColumnRefExpression which is the wrapped column reference, or nil
// when the expression is not a column behind zero or more casts.
func unwrapCastToColumnRef(expression querier_dto.Expression) *querier_dto.ColumnRefExpression {
	for range maxExpressionResolveDepth {
		switch typed := expression.(type) {
		case *querier_dto.ColumnRefExpression:
			return typed
		case *querier_dto.CastExpression:
			expression = typed.Inner
		default:
			return nil
		}
	}
	return nil
}

// inferExpressionName produces a reasonable column name for an unaliased SELECT
// expression.
//
// SQL does not require expressions in the projection to have a name, and PostgreSQL falls
// back to the literal placeholder "?column?", which is not a valid Go identifier. To keep
// generated row structs usable, this helper walks common expression shapes (column refs,
// casts, COALESCE, function calls) and lifts the most descriptive underlying name.
// Callers fall back to "?column?" when this returns the empty string.
//
// Takes expression (querier_dto.Expression) which is the unaliased SELECT expression.
//
// Returns string which is the inferred column name, or empty when no sensible name can be
// derived.
func inferExpressionName(expression querier_dto.Expression) string {
	return inferExpressionNameDepth(expression, 0)
}

// inferExpressionNameDepth walks an expression to infer a column name, stopping once the
// recursion depth reaches maxInferExpressionNameDepth.
//
// Takes expression (querier_dto.Expression) which is the expression to inspect.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the inferred column name, or empty when none applies.
func inferExpressionNameDepth(expression querier_dto.Expression, depth int) string {
	if depth >= maxInferExpressionNameDepth {
		return ""
	}
	if expression == nil {
		return ""
	}
	switch expr := expression.(type) {
	case *querier_dto.ColumnRefExpression:
		return inferColumnRefName(expr)
	case *querier_dto.CastExpression:
		return inferCastName(expr, depth)
	case *querier_dto.CoalesceExpression:
		return inferCoalesceName(expr, depth)
	case *querier_dto.FunctionCallExpression:
		return inferFunctionCallName(expr)
	default:
		return ""
	}
}

// inferColumnRefName returns the column name carried by a column reference, or empty when
// the expression is nil.
//
// Takes expr (*querier_dto.ColumnRefExpression) which is the column reference.
//
// Returns string which is the column name, or empty when expr is nil.
func inferColumnRefName(expr *querier_dto.ColumnRefExpression) string {
	if expr == nil {
		return ""
	}
	return expr.ColumnName
}

// inferCastName recurses into the inner expression of a CAST so the resulting column
// inherits the inner reference's name where possible.
//
// Takes expr (*querier_dto.CastExpression) which is the cast expression.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the inner expression's name, or empty when none applies.
func inferCastName(expr *querier_dto.CastExpression, depth int) string {
	if expr == nil {
		return ""
	}
	return inferExpressionNameDepth(expr.Inner, depth+1)
}

// inferCoalesceName returns the first non-empty name found by recursing into a COALESCE
// argument list. Mirrors the convention that COALESCE inherits the name of its first
// defaulted column.
//
// Takes expr (*querier_dto.CoalesceExpression) which is the COALESCE expression.
// Takes depth (int) which is the current recursion depth.
//
// Returns string which is the first non-empty argument name, or empty when none applies.
func inferCoalesceName(expr *querier_dto.CoalesceExpression, depth int) string {
	if expr == nil {
		return ""
	}
	for _, argument := range expr.Arguments {
		if name := inferExpressionNameDepth(argument, depth+1); name != "" {
			return name
		}
	}
	return ""
}

// inferFunctionCallName returns the function name as the inferred column name. This
// matches the common Postgres convention where an unaliased function-call projection gets
// the function name.
//
// Takes expr (*querier_dto.FunctionCallExpression) which is the function call.
//
// Returns string which is the function name, or empty when expr is nil.
func inferFunctionCallName(expr *querier_dto.FunctionCallExpression) string {
	if expr == nil {
		return ""
	}
	return expr.FunctionName
}

// mergeRawParameters resolves a flat list of raw parameter references against scope and
// merges the results into parameterTypes, combining multiple references to the same
// parameter number by promoting types. Numbers present in skip are ignored: those belong
// to a subquery and were already typed against their own scope by
// resolveSubqueryParameters.
//
// Takes rawParameters ([]querier_dto.RawParameterReference) which specifies the
// unresolved references.
// Takes scope (*scopeChain) which specifies the scope chain for type inference.
// Takes directiveNumberMap (map[int]*querier_dto.ParameterDirective) which specifies
// directives keyed by number.
// Takes directiveNameMap (map[string]*querier_dto.ParameterDirective) which specifies
// directives keyed by name.
// Takes parameterTypes (map[int]*querier_dto.QueryParameter) which accumulates the merged
// parameters keyed by number.
// Takes skip (map[int]bool) which holds parameter numbers to leave to the subquery pass.
//
// Returns []querier_dto.SourceError which holds any diagnostics from resolution failures.
func (r *typeResolver) mergeRawParameters(
	rawParameters []querier_dto.RawParameterReference,
	scope *scopeChain,
	directiveNumberMap map[int]*querier_dto.ParameterDirective,
	directiveNameMap map[string]*querier_dto.ParameterDirective,
	parameterTypes map[int]*querier_dto.QueryParameter,
	skip map[int]bool,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, raw := range rawParameters {
		if skip[raw.Number] {
			continue
		}
		sqlType, nullable, resolveError := r.resolveParameterType(raw, scope)
		if resolveError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  resolveError.Error(),
				Severity: querier_dto.SeverityWarning,
				Code:     extractErrorCode(resolveError),
			})
		}
		r.upsertParameterType(parameterTypes, raw, sqlType, nullable, directiveNumberMap, directiveNameMap)
	}

	return diagnostics
}

// upsertParameterType records a resolved parameter in parameterTypes by number: it merges
// the type into an existing entry following cast and promotion rules, or inserts a new
// entry with the resolved display name. The flat outer pass and the subquery pass share
// it for identical merge semantics.
//
// Takes parameterTypes (map[int]*querier_dto.QueryParameter) which accumulates
// parameters.
// Takes raw (querier_dto.RawParameterReference) which is the parameter being recorded.
// Takes sqlType (querier_dto.SQLType) which is the resolved type.
// Takes nullable (bool) which indicates whether the reference context is nullable.
// Takes directiveNumberMap (map[int]*querier_dto.ParameterDirective) which specifies
// directives keyed by number.
// Takes directiveNameMap (map[string]*querier_dto.ParameterDirective) which specifies
// directives keyed by name.
func (r *typeResolver) upsertParameterType(
	parameterTypes map[int]*querier_dto.QueryParameter,
	raw querier_dto.RawParameterReference,
	sqlType querier_dto.SQLType,
	nullable bool,
	directiveNumberMap map[int]*querier_dto.ParameterDirective,
	directiveNameMap map[string]*querier_dto.ParameterDirective,
) {
	if existing, exists := parameterTypes[raw.Number]; exists {
		r.mergeExistingParameterType(existing, sqlType, nullable, raw.CastType != nil)

		if raw.Context == querier_dto.ParameterContextLimit || raw.Context == querier_dto.ParameterContextOffset {
			if existing.IsPaginationBound() || existing.SQLType.Category == querier_dto.TypeCategoryUnknown {
				existing.Context = raw.Context
			}
		}
		return
	}
	parameterTypes[raw.Number] = &querier_dto.QueryParameter{
		Number:   raw.Number,
		Name:     resolveParameterName(raw, directiveNumberMap, directiveNameMap),
		SQLType:  sqlType,
		Nullable: nullable,
		Context:  raw.Context,
	}
}

// mergeExistingParameterType merges a newly resolved type into an existing parameter
// entry. Cast types take precedence; otherwise the engine's type promotion rules apply.
//
// Takes existing (*querier_dto.QueryParameter) which specifies the parameter to update.
// Takes sqlType (querier_dto.SQLType) which specifies the newly resolved type.
// Takes nullable (bool) which specifies whether the new reference context is nullable.
// Takes hasCastType (bool) which specifies whether the reference included an explicit
// cast.
func (r *typeResolver) mergeExistingParameterType(
	existing *querier_dto.QueryParameter,
	sqlType querier_dto.SQLType,
	nullable bool,
	hasCastType bool,
) {
	if hasCastType || (existing.SQLType.Category == querier_dto.TypeCategoryUnknown && sqlType.Category != querier_dto.TypeCategoryUnknown) {
		existing.SQLType = sqlType
	} else if existing.SQLType.Category != querier_dto.TypeCategoryUnknown && sqlType.Category != querier_dto.TypeCategoryUnknown {
		existing.SQLType = r.engine.PromoteType(existing.SQLType, sqlType)
	}
	if nullable {
		existing.Nullable = true
	}
}

// resolveParameterName determines the display name for a parameter, trying in order: the
// parameter's own name (`:email`); a directive override; the associated column (with
// `_like` suffix for LIKE patterns); "limit"/"offset" for LIMIT/OFFSET; or a generated
// "pN" fallback. Disambiguating duplicates is the caller's job (see
// disambiguateParameterNames).
//
// Takes raw (querier_dto.RawParameterReference) which specifies the raw parameter
// reference.
// Takes directiveNumberMap (map[int]*querier_dto.ParameterDirective) which specifies
// directives keyed by number.
// Takes directiveNameMap (map[string]*querier_dto.ParameterDirective) which specifies
// directives keyed by name.
//
// Returns string which holds the resolved parameter name.
func resolveParameterName(
	raw querier_dto.RawParameterReference,
	directiveNumberMap map[int]*querier_dto.ParameterDirective,
	directiveNameMap map[string]*querier_dto.ParameterDirective,
) string {
	if raw.Name != "" {
		if directive, exists := directiveNameMap[raw.Name]; exists {
			return directive.Name
		}
		return raw.Name
	}
	if directive, exists := directiveNumberMap[raw.Number]; exists {
		return directive.Name
	}
	if raw.ColumnReference != nil && raw.ColumnReference.ColumnName != "" {
		if raw.Context == querier_dto.ParameterContextLike {
			return raw.ColumnReference.ColumnName + "_like"
		}
		return raw.ColumnReference.ColumnName
	}
	switch raw.Context { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case querier_dto.ParameterContextLimit:
		return "limit"
	case querier_dto.ParameterContextOffset:
		return "offset"
	}
	return fmt.Sprintf("p%d", raw.Number)
}

// disambiguateParameterNames suffixes duplicate parameter names with a 1-based ordinal so
// each name is unique within the query.
//
// The first occurrence keeps its bare name; subsequent collisions get suffixes (`_2`,
// `_3`, ...) in declaration order, so suffixes are stable across runs. Directive- and
// SQL-derived names participate equally to prevent accidental collisions.
//
// Takes parameters ([]querier_dto.QueryParameter) which holds the parameters to
// disambiguate in place; the caller must pass them in ordinal (Number-ascending) order.
func disambiguateParameterNames(parameters []querier_dto.QueryParameter) {
	if len(parameters) < 2 {
		return
	}

	counts := make(map[string]int, len(parameters))
	for index := range parameters {
		counts[parameters[index].Name]++
	}

	seen := make(map[string]int, len(parameters))
	for index := range parameters {
		name := parameters[index].Name
		if counts[name] < 2 {
			continue
		}
		seen[name]++
		if seen[name] == 1 {
			continue
		}
		parameters[index].Name = fmt.Sprintf("%s_%d", name, seen[name])
	}
}

// assignSortableNumbers gives each standalone sortable directive a parameter number after
// the highest bound placeholder (and any other directive) so sortable inputs appear last
// in the generated params struct and never collide with a bound placeholder. Sortable
// directives bind no placeholder, so they start with number zero until assigned here.
//
// Takes parameterTypes (map[int]*querier_dto.QueryParameter) whose keys are the bound
// placeholder numbers resolved so far.
// Takes parameterDirectives ([]*querier_dto.ParameterDirective) which are mutated in
// place.
func assignSortableNumbers(
	parameterTypes map[int]*querier_dto.QueryParameter,
	parameterDirectives []*querier_dto.ParameterDirective,
) {
	maxNumber := 0
	for number := range parameterTypes {
		if number > maxNumber {
			maxNumber = number
		}
	}
	for _, directive := range parameterDirectives {
		if directive.Kind != querier_dto.ParameterDirectiveSortable && directive.Number > maxNumber {
			maxNumber = directive.Number
		}
	}
	for _, directive := range parameterDirectives {
		if directive.Kind == querier_dto.ParameterDirectiveSortable {
			maxNumber++
			directive.Number = maxNumber
		}
	}
}

// applyParameterDirectives applies directive overrides (type hints, nullability, kind) to
// the merged parameter map. Directives for parameters not yet in the map create new
// entries.
//
// Takes parameterTypes (map[int]*querier_dto.QueryParameter) which specifies the
// parameters to update.
// Takes parameterDirectives ([]*querier_dto.ParameterDirective) which specifies the
// directives to apply.
func (r *typeResolver) applyParameterDirectives(
	parameterTypes map[int]*querier_dto.QueryParameter,
	parameterDirectives []*querier_dto.ParameterDirective,
) {
	assignSortableNumbers(parameterTypes, parameterDirectives)

	for _, directive := range parameterDirectives {
		if _, exists := parameterTypes[directive.Number]; !exists {
			parameterTypes[directive.Number] = &querier_dto.QueryParameter{
				Number:   directive.Number,
				Name:     directive.Name,
				SQLType:  querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown},
				Nullable: false,
			}
		}

		parameter := parameterTypes[directive.Number]
		parameter.Name = directive.Name

		if directive.TypeHint != nil {
			parameter.SQLType = r.engine.NormaliseTypeName(*directive.TypeHint)
		}
		if directive.Nullable != nil {
			parameter.Nullable = *directive.Nullable
		}

		parameter.IsSlice = directive.IsSlice
		parameter.IsOptional = directive.IsOptional
		if directive.IsOptional {
			parameter.Nullable = true
		}
		if directive.DefaultVal != nil {
			parameter.DefaultLimit = directive.DefaultVal
		}
		if directive.MaxVal != nil {
			parameter.MaxLimit = directive.MaxVal
		}

		parameter.Kind = directive.Kind
		r.applyDirectiveKind(parameter, directive)
	}
}

// applyDirectiveKind applies kind-specific modifications to a parameter based on its
// directive kind (optional, slice, sortable, limit, offset).
//
// Takes parameter (*querier_dto.QueryParameter) which specifies the parameter to modify.
// Takes directive (*querier_dto.ParameterDirective) which specifies the directive with
// kind and constraints.
func (*typeResolver) applyDirectiveKind(
	parameter *querier_dto.QueryParameter,
	directive *querier_dto.ParameterDirective,
) {
	if directive.Kind == querier_dto.ParameterDirectiveSortable {
		parameter.SortableColumns = directive.Columns
		parameter.Nullable = false
	}
}

// collectParameters converts the parameter map into an ordered slice, sorted by parameter
// number from 1 to the maximum number. Missing numbers are skipped.
//
// Takes parameterTypes (map[int]*querier_dto.QueryParameter) which specifies the
// parameters keyed by number.
//
// Returns []querier_dto.QueryParameter which holds the parameters in ordinal order.
func collectParameters(parameterTypes map[int]*querier_dto.QueryParameter) []querier_dto.QueryParameter {
	numbers := slices.Sorted(maps.Keys(parameterTypes))
	result := make([]querier_dto.QueryParameter, 0, len(numbers))
	for _, number := range numbers {
		result = append(result, *parameterTypes[number])
	}

	return result
}

// expandStar expands a SELECT * or table.* into fully typed output columns by reading all
// columns from the matching table or CTE in the scope.
//
// Takes tableAlias (string) which specifies the table to expand, or empty for all tables.
// Takes scope (*scopeChain) which specifies the scope chain for table lookups.
//
// Returns []querier_dto.OutputColumn which holds the expanded output columns.
// Returns error which indicates Q003 if the specified table alias is unknown.
func (*typeResolver) expandStar(
	tableAlias string,
	scope *scopeChain,
) ([]querier_dto.OutputColumn, error) {
	if tableAlias != "" {
		if table, exists := scope.tables[tableAlias]; exists {
			outputColumns := make([]querier_dto.OutputColumn, len(table.Columns))
			for i := range table.Columns {
				outputColumns[i] = querier_dto.OutputColumn{
					Name:            table.Columns[i].Name,
					SQLType:         table.Columns[i].SQLType,
					Nullable:        table.Columns[i].Nullable,
					SourceTable:     table.Name,
					SourceSchema:    table.Schema,
					SourceColumn:    table.Columns[i].Name,
					SourceQualifier: table.Alias,
				}
			}
			return outputColumns, nil
		}
		if cte, exists := scope.ctes[strings.ToLower(tableAlias)]; exists {
			outputColumns := make([]querier_dto.OutputColumn, len(cte.columns))
			for i := range cte.columns {
				outputColumns[i] = querier_dto.OutputColumn{
					Name:            cte.columns[i].Name,
					SQLType:         cte.columns[i].SQLType,
					Nullable:        cte.columns[i].Nullable,
					SourceTable:     cte.name,
					SourceColumn:    cte.columns[i].Name,
					SourceQualifier: tableAlias,
				}
			}
			return outputColumns, nil
		}
		return nil, fmt.Errorf("%s: unknown table %q in SELECT *", querier_dto.CodeUnknownTable, tableAlias)
	}

	tableAliases := make([]string, 0, len(scope.tables))
	for alias := range scope.tables {
		tableAliases = append(tableAliases, alias)
	}
	slices.Sort(tableAliases)

	var outputColumns []querier_dto.OutputColumn
	for _, alias := range tableAliases {
		table := scope.tables[alias]
		for i := range table.Columns {
			outputColumns = append(outputColumns, querier_dto.OutputColumn{
				Name:            table.Columns[i].Name,
				SQLType:         table.Columns[i].SQLType,
				Nullable:        table.Columns[i].Nullable,
				SourceTable:     table.Name,
				SourceSchema:    table.Schema,
				SourceColumn:    table.Columns[i].Name,
				SourceQualifier: table.Alias,
			})
		}
	}
	return outputColumns, nil
}

// resolveParameterType infers a parameter's SQL type and nullability.
//
// The type comes from the cast type, the referenced column, or the usage context, in that
// order. When the column reference cannot be resolved against the scope chain the
// resolver falls back to a catalogue-wide lookup so subquery parameters (whose inner
// scope the engine adapter does not preserve) still get a type; ambiguous fallbacks let
// the original Q001/Q002 diagnostic stand.
//
// Takes raw (querier_dto.RawParameterReference) which specifies the raw parameter to
// resolve.
// Takes scope (*scopeChain) which specifies the scope chain for column reference lookups.
//
// Returns querier_dto.SQLType which holds the inferred SQL type.
// Returns bool which indicates whether the parameter is nullable.
// Returns error which indicates a scope resolution failure that the catalogue fallback
// could not disambiguate.
func (r *typeResolver) resolveParameterType(
	raw querier_dto.RawParameterReference,
	scope *scopeChain,
) (querier_dto.SQLType, bool, error) {
	if raw.CastType != nil {
		return r.resolveCustomType(*raw.CastType), false, nil
	}
	if raw.Context == querier_dto.ParameterContextLike {
		return r.resolveLikeParameterType(raw, scope)
	}
	if raw.ColumnReference != nil {
		return r.resolveColumnReferencedParameterType(raw.ColumnReference, scope)
	}

	if raw.Context == querier_dto.ParameterContextFunctionArgument && raw.EnclosingFunctionName != "" {
		if argumentType, resolved := r.functionResolver.ArgumentType(raw.EnclosingFunctionName, raw.ArgumentOrdinal); resolved {
			return r.resolveCustomType(argumentType), false, nil
		}
	}
	return resolveContextOnlyParameterType(raw.Context), false, nil
}

// resolveLikeParameterType returns the type for a LIKE-pattern parameter, always text,
// while still surfacing a Q001-style error when the LHS column reference resolves neither
// in scope nor via catalogue fallback.
//
// Takes raw (querier_dto.RawParameterReference) which specifies the raw parameter under
// resolution.
// Takes scope (*scopeChain) which specifies the active scope chain.
//
// Returns querier_dto.SQLType which is always text for LIKE parameters.
// Returns bool which is always false (LIKE parameters are not nullable based on the
// column).
// Returns error which surfaces unresolved column references so the diagnostic pass can
// emit Q001.
func (r *typeResolver) resolveLikeParameterType(
	raw querier_dto.RawParameterReference,
	scope *scopeChain,
) (querier_dto.SQLType, bool, error) {
	likeType := querier_dto.SQLType{Category: querier_dto.TypeCategoryText}
	if raw.ColumnReference == nil || raw.ColumnReference.ColumnName == "" {
		return likeType, false, nil
	}
	_, _, err := scope.ResolveColumn(raw.ColumnReference.TableAlias, raw.ColumnReference.ColumnName)
	if err == nil {
		return likeType, false, nil
	}
	if _, ok := r.findColumnInCatalogue(raw.ColumnReference); ok {
		return likeType, false, nil
	}
	if _, ok := resolveBareColumnFallback(scope, raw.ColumnReference); ok {
		return likeType, false, nil
	}
	return likeType, false, err
}

// resolveColumnReferencedParameterType resolves a parameter whose ColumnReference
// identifies a target column, falling back to a catalogue-wide lookup when the active
// scope chain cannot find the column (which happens for parameters carried up from
// subqueries the engine adapter flat-scanned).
//
// Takes reference (*querier_dto.ColumnReference) which specifies the column the parameter
// is compared against or assigned to.
// Takes scope (*scopeChain) which specifies the active scope chain.
//
// Returns querier_dto.SQLType which holds the resolved column type or Unknown when
// neither the scope nor the catalogue could match.
// Returns bool which indicates whether the column is nullable.
// Returns error which surfaces unresolved column references so the diagnostic pass can
// emit Q001.
func (r *typeResolver) resolveColumnReferencedParameterType(
	reference *querier_dto.ColumnReference,
	scope *scopeChain,
) (querier_dto.SQLType, bool, error) {
	column, _, err := scope.ResolveColumn(reference.TableAlias, reference.ColumnName)
	if err != nil {
		if fallback, ok := r.findColumnInCatalogue(reference); ok {
			return fallback.sqlType, fallback.nullable, nil
		}
		if bare, ok := resolveBareColumnFallback(scope, reference); ok {
			return bare.SQLType, bare.Nullable, nil
		}
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, false, err
	}
	if column == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, false, nil
	}
	return column.SQLType, column.Nullable, nil
}

// resolveBareColumnFallback retries an unqualified column lookup when a qualified
// parameter reference fails to resolve.
//
// A parameter hoisted out of a subquery keeps that subquery's local table alias, which is
// absent from the parent scope, so the qualified lookup wrongly fails. If the bare column
// name then resolves unambiguously in the current scope, the parameter is typed from it
// and the spurious Q001/Q002 is suppressed. A reference that was already unqualified, or
// whose bare column is ambiguous or unknown, returns ok=false so the original diagnostic
// stands.
//
// Takes scope (*scopeChain) which specifies the active scope chain.
// Takes reference (*querier_dto.ColumnReference) which is the parameter's target column.
//
// Returns *querier_dto.ScopedColumn which holds the unambiguously resolved column.
// Returns bool which reports whether such a column was found.
func resolveBareColumnFallback(
	scope *scopeChain,
	reference *querier_dto.ColumnReference,
) (*querier_dto.ScopedColumn, bool) {
	if reference.TableAlias == "" {
		return nil, false
	}
	column, _, err := scope.ResolveColumn("", reference.ColumnName)
	if err != nil || column == nil {
		return nil, false
	}
	return column, true
}

// resolveContextOnlyParameterType returns the type that a parameter takes from its
// surrounding context alone, used when no cast or column reference is available.
//
// Takes parameterContext (querier_dto.ParameterContext) which specifies the SQL context
// the parameter appears in.
//
// Returns querier_dto.SQLType which is integer for LIMIT/OFFSET and Unknown otherwise.
func resolveContextOnlyParameterType(parameterContext querier_dto.ParameterContext) querier_dto.SQLType {
	switch parameterContext { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case querier_dto.ParameterContextLimit, querier_dto.ParameterContextOffset:
		return querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryInteger,
			EngineName: querier_dto.CanonicalInt4,
		}
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
}

// findColumnInCatalogue resolves a column via a catalogue-wide lookup.
//
// It searches every schema and returns the column's type only when exactly one table
// holds it. A non-empty TableAlias narrows the search to tables with that name (per the
// SQL convention of using bare table names as default aliases); an empty TableAlias scans
// all tables and refuses ambiguous results so the genuine Q001/Q002 diagnostic can fire.
// Comparison is case-insensitive. Views are skipped: their column types come from a
// SELECT body that this fallback does not re-resolve.
//
// Takes reference (*querier_dto.ColumnReference) which specifies the column to look up.
//
// Returns catalogueColumnMatch which holds the matched column's type and nullability when
// ok is true.
// Returns bool which is true when exactly one column matched.
func (r *typeResolver) findColumnInCatalogue(
	reference *querier_dto.ColumnReference,
) (catalogueColumnMatch, bool) {
	return findColumnInCatalogueFor(r.catalogue, reference)
}

// findColumnInCatalogueFor is the catalogue-wide column lookup shared by the type
// resolver's parameter path and the column-existence diagnostic pass. Holding a
// single body keeps the match, ambiguity-refusal and view-skip rules identical for
// both callers.
//
// Takes catalogue (*querier_dto.Catalogue) which holds the schema state to search.
// Takes reference (*querier_dto.ColumnReference) which specifies the column to look
// up.
//
// Returns catalogueColumnMatch which holds the matched column's type and nullability
// when ok is true.
// Returns bool which is true when exactly one column matched.
func findColumnInCatalogueFor(
	catalogue *querier_dto.Catalogue,
	reference *querier_dto.ColumnReference,
) (catalogueColumnMatch, bool) {
	if catalogue == nil || reference == nil || reference.ColumnName == "" {
		return catalogueColumnMatch{}, false
	}

	var found catalogueColumnMatch
	matchCount := 0
	for _, schema := range catalogue.Schemas {
		if schema == nil {
			continue
		}
		match, hits, ambiguous := matchColumnInSchema(schema, reference)
		if ambiguous {
			return catalogueColumnMatch{}, false
		}
		matchCount += hits
		if matchCount > 1 {
			return catalogueColumnMatch{}, false
		}
		if hits == 1 {
			found = match
		}
	}

	if matchCount != 1 {
		return catalogueColumnMatch{}, false
	}
	return found, true
}

// matchColumnInSchema scans a single schema for tables holding the reference's column,
// applying the optional TableAlias filter. It returns early when more than one match is
// seen so the caller can short-circuit before walking later schemas.
//
// Takes schema (*querier_dto.Schema) which specifies the schema to scan.
// Takes reference (*querier_dto.ColumnReference) which specifies the column to look up.
//
// Returns catalogueColumnMatch which holds the column type when a single match was found.
// Returns int which is the number of matches encountered (capped at 2 so callers can
// recognise ambiguity).
// Returns bool which is true when the schema alone contains two or more matches, allowing
// the caller to abort.
func matchColumnInSchema(
	schema *querier_dto.Schema,
	reference *querier_dto.ColumnReference,
) (catalogueColumnMatch, int, bool) {
	var found catalogueColumnMatch
	matches := 0
	for _, table := range schema.Tables {
		if table == nil {
			continue
		}
		if reference.TableAlias != "" && !tableMatchesAlias(table, reference.TableAlias) {
			continue
		}
		match, ok := findColumnInTable(table, reference.ColumnName)
		if !ok {
			continue
		}
		matches++
		if matches > 1 {
			return catalogueColumnMatch{}, matches, true
		}
		found = match
	}
	return found, matches, false
}

// findColumnInTable returns the type information for a column on a specific table.
// Comparison is case-insensitive.
//
// Takes table (*querier_dto.Table) which specifies the table to search.
// Takes columnName (string) which specifies the column to find.
//
// Returns catalogueColumnMatch which holds the matched column's type.
// Returns bool which is true when the column was found.
func findColumnInTable(table *querier_dto.Table, columnName string) (catalogueColumnMatch, bool) {
	for i := range table.Columns {
		if strings.EqualFold(table.Columns[i].Name, columnName) {
			return catalogueColumnMatch{
				sqlType:  arrayWrappedSQLType(&table.Columns[i]),
				nullable: table.Columns[i].Nullable,
			}, true
		}
	}
	return catalogueColumnMatch{}, false
}

// tableMatchesAlias reports whether the catalogue table can be referenced by the supplied
// alias, treating the table's bare name as a default alias. Query-local aliases (FROM
// users AS u) are not in the catalogue, so this only catches the unqualified-or-bare-name
// case.
//
// Takes table (*querier_dto.Table) which specifies the catalogue table.
// Takes alias (string) which specifies the qualifier from the parameter's column
// reference.
//
// Returns bool which is true when the alias matches the table's name.
func tableMatchesAlias(table *querier_dto.Table, alias string) bool {
	return strings.EqualFold(table.Name, alias)
}
