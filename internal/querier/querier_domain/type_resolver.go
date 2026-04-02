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
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// errorCodeMinLength holds the minimum message length required to contain a Q-code prefix.
	errorCodeMinLength = 5

	// errorCodePrefixLength holds the number of characters in a Q-code prefix (e.g. "Q001").
	errorCodePrefixLength = 4
)

// typeResolver performs bottom-up expression type inference, resolving raw
// output columns and parameter references from the engine adapter into fully
// typed results. This is where column references are looked up in the scope
// chain and function calls go through overload resolution.
type typeResolver struct {
	// catalogue holds the database catalogue for table and type lookups.
	catalogue *querier_dto.Catalogue

	// functionResolver holds the function overload resolver for function call expressions.
	functionResolver *functionResolver

	// engine holds the engine adapter for type normalisation, promotion, and casting rules.
	engine EngineTypeSystemPort
}

// newTypeResolver creates a type resolver with the
// given catalogue, function resolver, and engine
// adapter.
//
// Takes catalogue (*querier_dto.Catalogue) which
// specifies the database catalogue for lookups.
// Takes functionResolver (*functionResolver) which
// specifies the function overload resolver.
// Takes engine (EngineTypeSystemPort) which specifies
// the engine adapter for type system rules.
//
// Returns *typeResolver which holds the initialised
// type resolver.
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

// ResolveOutputColumns resolves raw output columns from
// the engine adapter into fully typed output columns
// using the scope chain for column lookups and function
// resolver for expression types.
//
// Takes ctx (context.Context) which specifies the
// context for tracing.
// Takes rawColumns ([]querier_dto.RawOutputColumn)
// which specifies the unresolved output columns.
//
// Returns []querier_dto.OutputColumn which holds the
// fully typed output columns.
// Returns bool which indicates whether any expression
// modifies data.
// Returns []querier_dto.SourceError which holds any
// diagnostics from resolution failures.
func (r *typeResolver) ResolveOutputColumns(
	ctx context.Context,
	rawColumns []querier_dto.RawOutputColumn,
	scope *scopeChain,
) ([]querier_dto.OutputColumn, bool, []querier_dto.SourceError) {
	_, span, _ := log.Span(ctx, "TypeResolver.ResolveOutputColumns")
	defer span.End()

	var resolved []querier_dto.OutputColumn
	var diagnostics []querier_dto.SourceError
	var dataModifying bool

	for _, raw := range rawColumns {
		columns, diagnostic := r.resolveSingleOutputColumn(raw, scope, &dataModifying)
		if diagnostic != nil {
			diagnostics = append(diagnostics, *diagnostic)
			continue
		}
		resolved = append(resolved, columns...)
	}

	return resolved, dataModifying, diagnostics
}

// ResolveParameters resolves raw parameter references
// from the engine adapter into fully typed query
// parameters using the scope chain for context-based
// type inference and directive declarations for
// overrides.
//
// Takes ctx (context.Context) which specifies the
// context for tracing.
// Takes rawParameters
// ([]querier_dto.RawParameterReference) which specifies
// the unresolved parameter references.
// Takes parameterDirectives
// ([]*querier_dto.ParameterDirective) which specifies
// the directive overrides for parameters.
//
// Returns []querier_dto.QueryParameter which holds the
// fully typed parameters in ordinal order.
// Returns []querier_dto.SourceError which holds any
// diagnostics from resolution failures.
func (r *typeResolver) ResolveParameters(
	ctx context.Context,
	rawParameters []querier_dto.RawParameterReference,
	scope *scopeChain,
	parameterDirectives []*querier_dto.ParameterDirective,
) ([]querier_dto.QueryParameter, []querier_dto.SourceError) {
	_, span, _ := log.Span(ctx, "TypeResolver.ResolveParameters")
	defer span.End()

	directiveNumberMap := make(map[int]*querier_dto.ParameterDirective, len(parameterDirectives))
	directiveNameMap := make(map[string]*querier_dto.ParameterDirective, len(parameterDirectives))
	for _, directive := range parameterDirectives {
		directiveNumberMap[directive.Number] = directive
		if directive.DirectiveName != "" {
			directiveNameMap[directive.DirectiveName] = directive
		}
	}

	parameterTypes, diagnostics := r.mergeRawParameters(rawParameters, scope, directiveNumberMap, directiveNameMap)

	r.applyParameterDirectives(parameterTypes, parameterDirectives)

	return collectParameters(parameterTypes), diagnostics
}

// resolveSingleOutputColumn resolves a single raw
// output column into one or more typed output columns.
// Star expressions expand to multiple columns; column
// references and expressions each produce one.
//
// Takes raw (querier_dto.RawOutputColumn) which
// specifies the unresolved output column.
// Takes scope (*scopeChain) which specifies the scope
// chain for column lookups.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the expression modifies data.
//
// Returns []querier_dto.OutputColumn which holds the
// resolved output columns.
// Returns *querier_dto.SourceError which holds a
// diagnostic if resolution failed, or nil on success.
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

// resolveColumnRefOutput resolves a column reference
// output by looking up the column in the scope chain
// and building an output column with source metadata.
//
// Takes raw (querier_dto.RawOutputColumn) which
// specifies the raw column reference.
// Takes scope (*scopeChain) which specifies the scope
// chain for column lookups.
//
// Returns []querier_dto.OutputColumn which holds the
// resolved output column.
// Returns *querier_dto.SourceError which holds a
// diagnostic if resolution failed, or nil on success.
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
	if table != nil {
		sourceTable = table.Name
	}

	return []querier_dto.OutputColumn{{
		Name:         outputName,
		SQLType:      column.SQLType,
		Nullable:     column.Nullable,
		SourceTable:  sourceTable,
		SourceColumn: column.Name,
	}}, nil
}

// resolveExpressionOutput resolves an expression-based
// output column by inferring its type through the
// expression type resolver.
//
// Takes raw (querier_dto.RawOutputColumn) which
// specifies the raw expression column.
// Takes scope (*scopeChain) which specifies the scope
// chain for expression resolution.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the expression modifies data.
//
// Returns []querier_dto.OutputColumn which holds the
// resolved output column.
// Returns *querier_dto.SourceError which holds a
// diagnostic if resolution failed, or nil on success.
func (r *typeResolver) resolveExpressionOutput(
	raw querier_dto.RawOutputColumn,
	scope *scopeChain,
	dataModifying *bool,
) ([]querier_dto.OutputColumn, *querier_dto.SourceError) {
	sqlType, nullable, expressionError := r.resolveExpressionType(raw.Expression, scope, dataModifying)
	if expressionError != nil {
		return nil, &querier_dto.SourceError{
			Message:  expressionError.Error(),
			Severity: querier_dto.SeverityWarning,
			Code:     querier_dto.CodeExpressionTypeError,
		}
	}

	outputName := raw.Name
	if outputName == "" {
		outputName = "?column?"
	}

	return []querier_dto.OutputColumn{{
		Name:     outputName,
		SQLType:  sqlType,
		Nullable: nullable,
	}}, nil
}

// mergeRawParameters merges raw parameter references
// into a map of query parameters, combining multiple
// references to the same parameter number by promoting
// types.
//
// Takes rawParameters
// ([]querier_dto.RawParameterReference) which specifies
// the unresolved references.
// Takes scope (*scopeChain) which specifies the scope
// chain for type inference.
// Takes directiveNumberMap
// (map[int]*querier_dto.ParameterDirective) which
// specifies directives keyed by number.
// Takes directiveNameMap
// (map[string]*querier_dto.ParameterDirective) which
// specifies directives keyed by name.
//
// Returns map[int]*querier_dto.QueryParameter which
// holds the merged parameters keyed by number.
// Returns []querier_dto.SourceError which holds any
// diagnostics from resolution failures.
func (r *typeResolver) mergeRawParameters(
	rawParameters []querier_dto.RawParameterReference,
	scope *scopeChain,
	directiveNumberMap map[int]*querier_dto.ParameterDirective,
	directiveNameMap map[string]*querier_dto.ParameterDirective,
) (map[int]*querier_dto.QueryParameter, []querier_dto.SourceError) {
	parameterTypes := make(map[int]*querier_dto.QueryParameter)
	var diagnostics []querier_dto.SourceError

	for _, raw := range rawParameters {
		sqlType, nullable, resolveError := r.resolveParameterType(raw, scope)
		if resolveError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  resolveError.Error(),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeExpressionTypeError,
			})
		}

		if existing, exists := parameterTypes[raw.Number]; exists {
			r.mergeExistingParameterType(existing, sqlType, nullable, raw.CastType != nil)
			continue
		}

		name := resolveParameterName(raw, directiveNumberMap, directiveNameMap)
		parameterTypes[raw.Number] = &querier_dto.QueryParameter{
			Number:   raw.Number,
			Name:     name,
			SQLType:  sqlType,
			Nullable: nullable,
		}
	}

	return parameterTypes, diagnostics
}

// mergeExistingParameterType merges a newly resolved type into an existing parameter
// entry. Cast types take precedence; otherwise the engine's type promotion rules apply.
//
// Takes existing (*querier_dto.QueryParameter) which specifies the parameter to update.
// Takes sqlType (querier_dto.SQLType) which specifies the newly resolved type.
// Takes nullable (bool) which specifies whether the new reference context is nullable.
// Takes hasCastType (bool) which specifies whether the reference included an explicit cast.
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

// resolveParameterName determines the display name for
// a parameter from its raw reference and directive
// maps. Named parameters use their directive name if
// available; unnamed parameters fall back to the
// directive name or a generated "pN" name.
//
// Takes raw (querier_dto.RawParameterReference) which
// specifies the raw parameter reference.
// Takes directiveNumberMap
// (map[int]*querier_dto.ParameterDirective) which
// specifies directives keyed by number.
// Takes directiveNameMap
// (map[string]*querier_dto.ParameterDirective) which
// specifies directives keyed by name.
//
// Returns string which holds the resolved parameter
// name.
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
	return fmt.Sprintf("p%d", raw.Number)
}

// applyParameterDirectives applies directive overrides
// (type hints, nullability, kind) to the merged
// parameter map. Directives for parameters not yet in
// the map create new entries.
//
// Takes parameterTypes
// (map[int]*querier_dto.QueryParameter) which specifies
// the parameters to update.
// Takes parameterDirectives
// ([]*querier_dto.ParameterDirective) which specifies
// the directives to apply.
func (r *typeResolver) applyParameterDirectives(
	parameterTypes map[int]*querier_dto.QueryParameter,
	parameterDirectives []*querier_dto.ParameterDirective,
) {
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

		parameter.Kind = directive.Kind
		r.applyDirectiveKind(parameter, directive)
	}
}

// applyDirectiveKind applies kind-specific
// modifications to a parameter based on its directive
// kind (optional, slice, sortable, limit, offset).
//
// Takes parameter (*querier_dto.QueryParameter) which
// specifies the parameter to modify.
// Takes directive (*querier_dto.ParameterDirective)
// which specifies the directive with kind and
// constraints.
func (*typeResolver) applyDirectiveKind(
	parameter *querier_dto.QueryParameter,
	directive *querier_dto.ParameterDirective,
) {
	switch directive.Kind {
	case querier_dto.ParameterDirectiveOptional:
		parameter.IsOptional = true
		parameter.Nullable = true
	case querier_dto.ParameterDirectiveSlice:
		parameter.IsSlice = true
	case querier_dto.ParameterDirectiveSortable:
		parameter.SortableColumns = directive.Columns
		parameter.Nullable = false
	case querier_dto.ParameterDirectiveLimit:
		parameter.SQLType = querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryInteger,
			EngineName: querier_dto.CanonicalInt4,
		}
		parameter.Nullable = false
		parameter.DefaultLimit = directive.DefaultVal
		parameter.MaxLimit = directive.MaxVal
	case querier_dto.ParameterDirectiveOffset:
		parameter.SQLType = querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryInteger,
			EngineName: querier_dto.CanonicalInt4,
		}
		parameter.Nullable = false
	}
}

// collectParameters converts the parameter map into an
// ordered slice, sorted by parameter number from 1 to
// the maximum number. Missing numbers are skipped.
//
// Takes parameterTypes
// (map[int]*querier_dto.QueryParameter) which specifies
// the parameters keyed by number.
//
// Returns []querier_dto.QueryParameter which holds the
// parameters in ordinal order.
func collectParameters(parameterTypes map[int]*querier_dto.QueryParameter) []querier_dto.QueryParameter {
	maxNumber := 0
	for number := range parameterTypes {
		if number > maxNumber {
			maxNumber = number
		}
	}

	result := make([]querier_dto.QueryParameter, 0, maxNumber)
	for i := 1; i <= maxNumber; i++ {
		if parameter, exists := parameterTypes[i]; exists {
			result = append(result, *parameter)
		}
	}

	return result
}

// expandStar expands a SELECT * or table.* into fully typed output columns
// by reading all columns from the matching table or CTE in the scope.
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
					Name:         table.Columns[i].Name,
					SQLType:      table.Columns[i].SQLType,
					Nullable:     table.Columns[i].Nullable,
					SourceTable:  table.Name,
					SourceColumn: table.Columns[i].Name,
				}
			}
			return outputColumns, nil
		}
		if cte, exists := scope.ctes[strings.ToLower(tableAlias)]; exists {
			outputColumns := make([]querier_dto.OutputColumn, len(cte.columns))
			for i := range cte.columns {
				outputColumns[i] = querier_dto.OutputColumn{
					Name:         cte.columns[i].Name,
					SQLType:      cte.columns[i].SQLType,
					Nullable:     cte.columns[i].Nullable,
					SourceTable:  cte.name,
					SourceColumn: cte.columns[i].Name,
				}
			}
			return outputColumns, nil
		}
		return nil, fmt.Errorf("Q003: unknown table %q in SELECT *", tableAlias)
	}

	var outputColumns []querier_dto.OutputColumn
	for _, table := range scope.tables {
		for i := range table.Columns {
			outputColumns = append(outputColumns, querier_dto.OutputColumn{
				Name:         table.Columns[i].Name,
				SQLType:      table.Columns[i].SQLType,
				Nullable:     table.Columns[i].Nullable,
				SourceTable:  table.Name,
				SourceColumn: table.Columns[i].Name,
			})
		}
	}
	return outputColumns, nil
}

// resolveParameterType infers the SQL type and
// nullability of a single raw parameter reference from
// its cast type, column reference, or usage context.
//
// Takes raw (querier_dto.RawParameterReference) which
// specifies the raw parameter to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for column reference lookups.
//
// Returns querier_dto.SQLType which holds the inferred
// SQL type.
// Returns bool which indicates whether the parameter is
// nullable.
// Returns error which indicates a scope resolution
// failure.
func (r *typeResolver) resolveParameterType(
	raw querier_dto.RawParameterReference,
	scope *scopeChain,
) (querier_dto.SQLType, bool, error) {
	if raw.CastType != nil {
		resolved := r.resolveCustomType(*raw.CastType)
		return resolved, false, nil
	}

	if raw.ColumnReference != nil {
		column, _, err := scope.ResolveColumn(
			raw.ColumnReference.TableAlias,
			raw.ColumnReference.ColumnName,
		)
		if err != nil {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, false, err
		}
		if column == nil {
			return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, false, nil
		}
		return column.SQLType, column.Nullable, nil
	}

	switch raw.Context {
	case querier_dto.ParameterContextLimit, querier_dto.ParameterContextOffset:
		return querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryInteger,
			EngineName: querier_dto.CanonicalInt4,
		}, false, nil
	}

	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, false, nil
}
