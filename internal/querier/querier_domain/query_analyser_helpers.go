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

// computeDynamicFlags determines whether a query is dynamic and collects sortable column
// names from the resolved parameters.
//
// A query is dynamic when it has a sortable input, an optional parameter (the predicate
// it appears in is dropped at runtime), or a LIMIT/OFFSET parameter that carries clamp
// configuration (a default or max), which needs the params-struct / clamp path. A
// LIMIT/OFFSET parameter without clamp config is a positional bind and does not make the
// query dynamic.
//
// Takes parameters ([]querier_dto.QueryParameter) which holds the resolved parameters.
//
// Returns bool which indicates whether any parameter makes the query dynamic.
//
// Returns []string which holds the column names declared on sortable inputs.
func computeDynamicFlags(parameters []querier_dto.QueryParameter) (bool, []string) {
	isDynamic := false
	var dynamicColumns []string
	for index := range parameters {
		parameter := &parameters[index]
		switch {
		case parameter.Kind == querier_dto.ParameterDirectiveSortable:
			isDynamic = true
			dynamicColumns = append(dynamicColumns, parameter.SortableColumns...)
		case parameter.IsOptional:
			isDynamic = true
		case parameter.IsPaginationBound() && (parameter.DefaultLimit != nil || parameter.MaxLimit != nil):
			isDynamic = true
		}
	}
	return isDynamic, dynamicColumns
}

// assembleQueryInput groups the inputs needed to construct an AnalysedQuery result.
type assembleQueryInput struct {
	// directives holds the parsed query-level directives such as group_by and nullable
	// overrides.
	directives *querier_dto.QueryDirectives

	// rawAnalysis holds the raw analysis result produced by the engine.
	rawAnalysis *querier_dto.RawQueryAnalysis

	// queryName holds the name declared by the piko.name directive.
	queryName string

	// filename holds the source file path for error reporting.
	filename string

	// outputColumns holds the resolved output columns for the query.
	outputColumns []querier_dto.OutputColumn

	// parameters holds the resolved query parameters.
	parameters []querier_dto.QueryParameter

	// block holds the raw query block containing SQL text and line information.
	block queryBlock

	// queryCommand holds the command type declared by the piko.command directive.
	queryCommand querier_dto.QueryCommand

	// isDynamic indicates whether the query requires dynamic SQL generation.
	isDynamic bool
}

// assembleQuery constructs an AnalysedQuery from the provided input fields.
//
// Takes input (assembleQueryInput) which holds all resolved query components.
//
// Returns *querier_dto.AnalysedQuery which holds the fully assembled query ready for code
// generation.
func (a *queryAnalyser) assembleQuery(input assembleQueryInput) *querier_dto.AnalysedQuery {
	readOnly := input.rawAnalysis.ReadOnly
	if input.directives.ReadOnlyOverride != nil {
		readOnly = *input.directives.ReadOnlyOverride
	}

	query := &querier_dto.AnalysedQuery{
		Name:                    input.queryName,
		Command:                 input.queryCommand,
		SQL:                     input.block.sql,
		Filename:                input.filename,
		Line:                    input.block.startLine,
		OutputColumns:           input.outputColumns,
		Parameters:              input.parameters,
		IsDynamic:               input.isDynamic,
		GroupByKey:              input.directives.GroupByKeys,
		Directives:              *input.directives,
		InsertTable:             input.rawAnalysis.InsertTable,
		InsertColumns:           input.rawAnalysis.InsertColumns,
		DynamicRuntime:          input.directives.DynamicRuntime,
		ReadOnly:                readOnly,
		BaseQueryHasWhereClause: input.rawAnalysis.HasWhereClause,
		Optional:                input.directives.Optional,
	}

	if input.directives.DynamicRuntime {
		query.AllowedColumns = extractAllowedColumns(input.outputColumns)

		countSQL, wrapped, rewriteErr := a.engine.RewriteSelectAsCount(input.block.sql, input.rawAnalysis)
		if rewriteErr == nil {
			query.CountSQL = countSQL
			query.CountSQLWrapped = wrapped
		}
	}

	return query
}

// extractAllowedColumns returns the columns a dynamic runtime builder may filter and
// order by, restricted to the query's SELECT projection.
//
// A dynamic query can only be filtered and ordered over the data it already returns. Each
// allowed column is keyed on its output (result-set) name, which is what a caller passes
// to Where and OrderBy, and carries the qualified source expression the builder emits in
// its place (for example output "email" -> "users.email"). Qualifying with the table
// reference keeps the emitted filter unambiguous across joins and makes it reference the
// real column rather than the caller's text. Output columns that are computed expressions
// or aggregates carry no source column and are excluded, since they cannot be referenced
// as a bare identifier in WHERE or ORDER BY. Scoping to the projection keeps the dynamic
// surface from exposing a column the query never selects, so a sensitive column that is
// present on a FROM or JOIN table but absent from the SELECT can never become a runtime
// filter or ordering oracle.
//
// Takes outputColumns ([]querier_dto.OutputColumn) which holds the query's projected
// columns.
//
// Returns []querier_dto.AllowedColumn which holds the deduplicated set of columns
// available for dynamic filtering and ordering, keyed on output name.
func extractAllowedColumns(outputColumns []querier_dto.OutputColumn) []querier_dto.AllowedColumn {
	var allowed []querier_dto.AllowedColumn
	seen := make(map[string]struct{})

	for index := range outputColumns {
		column := &outputColumns[index]
		if column.SourceColumn == "" {
			continue
		}

		outputName := column.Name
		if outputName == "" {
			outputName = column.SourceColumn
		}
		if _, exists := seen[outputName]; exists {
			continue
		}
		seen[outputName] = struct{}{}

		sourceExpression := column.SourceColumn
		if column.SourceQualifier != "" {
			sourceExpression = column.SourceQualifier + "." + column.SourceColumn
		}

		allowed = append(allowed, querier_dto.AllowedColumn{
			Name:             outputName,
			SourceExpression: sourceExpression,
			SQLType:          column.SQLType,
		})
	}

	return allowed
}

// findTable looks up a table in the catalogue by schema and name.
//
// Takes schema (string) which specifies the schema name, defaulting to the catalogue
// default if empty.
//
// Takes name (string) which specifies the table name to look up.
//
// Returns *querier_dto.Table which holds the matched table, or nil if not found.
func (a *queryAnalyser) findTable(schema string, name string) *querier_dto.Table {
	if schema == "" {
		schema = a.catalogue.DefaultSchema
	}
	schemaObject, exists := a.catalogue.Schemas[schema]
	if !exists {
		return nil
	}
	return schemaObject.Tables[name]
}

// buildScopeChain populates the scope chain from a raw query analysis result.
//
// FROM tables and JOIN tables are added with correct nullability adjustments. FROM
// entries that match previously resolved CTEs are registered as CTE aliases rather than
// resolved against the catalogue.
//
// Takes raw (*querier_dto.RawQueryAnalysis) which holds the parsed FROM, JOIN, and
// derived table references.
//
// Takes scope (*scopeChain) which holds the scope chain to populate.
//
// Returns []querier_dto.SourceError which holds any warnings produced when tables cannot
// be resolved.
func (a *queryAnalyser) buildScopeChain(
	raw *querier_dto.RawQueryAnalysis,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, tableReference := range raw.FromTables {
		if cte, exists := scope.ctes[strings.ToLower(tableReference.Name)]; exists {
			alias := tableReference.Alias
			if alias == "" {
				alias = tableReference.Name
			}
			if !strings.EqualFold(alias, tableReference.Name) {
				scope.AddCTE(alias, cte.columns)
			}
			scope.AddCTEAsTable(alias, cte.columns, querier_dto.JoinInner)
			continue
		}

		catalogueTable, resolveError := a.resolveTableReference(tableReference)
		if resolveError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  resolveError.Error(),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownTable,
			})
			continue
		}
		_ = scope.AddTable(tableReference, querier_dto.JoinInner, catalogueTable)
	}

	diagnostics = append(diagnostics, a.resolveJoinClauses(raw.JoinClauses, scope)...)

	for _, derivedTable := range raw.DerivedTables {
		scope.AddDerivedTable(derivedTable)
	}

	return diagnostics
}

// resolveJoinClauses adds each join clause to the scope chain, resolving CTE references
// or catalogue tables.
//
// Takes joinClauses ([]querier_dto.JoinClause) which holds the parsed JOIN clauses from
// the query.
//
// Takes scope (*scopeChain) which holds the scope chain to populate with join table
// entries.
//
// Returns []querier_dto.SourceError which holds any warnings produced when join tables
// cannot be resolved.
func (a *queryAnalyser) resolveJoinClauses(
	joinClauses []querier_dto.JoinClause,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, joinClause := range joinClauses {
		if cte, exists := scope.ctes[strings.ToLower(joinClause.Table.Name)]; exists {
			alias := joinClause.Table.Alias
			if alias == "" {
				alias = joinClause.Table.Name
			}
			scope.AddCTE(alias, cte.columns)
			scope.AddCTEAsTable(alias, cte.columns, joinClause.Kind)
			continue
		}

		catalogueTable, resolveError := a.resolveTableReference(joinClause.Table)
		if resolveError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  resolveError.Error(),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownTable,
			})
			continue
		}
		_ = scope.AddTable(joinClause.Table, joinClause.Kind, catalogueTable)
	}

	return diagnostics
}

// resolveTableReference looks up a table or view in the catalogue by its reference.
//
// Takes reference (querier_dto.TableReference) which specifies the schema, name, and
// alias of the table.
//
// Returns *querier_dto.Table which holds the matched catalogue table or view.
//
// Returns error when the schema or table name cannot be found in the catalogue.
func (a *queryAnalyser) resolveTableReference(
	reference querier_dto.TableReference,
) (*querier_dto.Table, error) {
	schemaName := reference.Schema
	if schemaName == "" {
		schemaName = a.catalogue.DefaultSchema
	}

	schema, exists := a.catalogue.Schemas[schemaName]
	if !exists {
		return nil, fmt.Errorf("unknown schema %q", schemaName)
	}

	if table, exists := schema.Tables[reference.Name]; exists {
		return table, nil
	}

	if view, exists := schema.Views[reference.Name]; exists {
		return &querier_dto.Table{
			Name:    view.Name,
			Schema:  view.Schema,
			Columns: view.Columns,
		}, nil
	}

	return nil, fmt.Errorf("unknown table or view %q in schema %q", reference.Name, schemaName)
}

// resolveCTEs resolves each CTE definition in order and registers the results in the
// scope chain.
//
// Takes cteDefinitions ([]querier_dto.RawCTEDefinition) which holds the parsed CTE
// definitions from the query.
//
// Takes scope (*scopeChain) which holds the scope chain where resolved CTEs are
// registered.
//
// Returns []querier_dto.SourceError which holds any diagnostics produced during CTE
// resolution.
func (a *queryAnalyser) resolveCTEs(
	ctx context.Context,
	cteDefinitions []querier_dto.RawCTEDefinition,
	scope *scopeChain,
) []querier_dto.SourceError {
	diagnostics := make([]querier_dto.SourceError, 0, len(cteDefinitions))

	for index := range cteDefinitions {
		if ctx.Err() != nil {
			return diagnostics
		}
		diagnostics = append(diagnostics, a.resolveSingleCTE(ctx, &cteDefinitions[index], scope)...)
	}

	return diagnostics
}

// resolveSingleCTE resolves one CTE definition by building a child scope, resolving its
// output columns, and registering it.
//
// Takes cteDefinition (querier_dto.RawCTEDefinition) which holds the parsed CTE
// definition to resolve.
//
// Takes scope (*scopeChain) which holds the parent scope chain where the CTE is
// registered.
//
// Returns []querier_dto.SourceError which holds any diagnostics produced during
// resolution.
func (a *queryAnalyser) resolveSingleCTE(
	ctx context.Context,
	cteDefinition *querier_dto.RawCTEDefinition,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	cteScope := scope.CreateChildScope(querier_dto.ScopeKindCTE)

	diagnostics = append(diagnostics, a.populateCTEScope(cteDefinition.FromTables, scope, cteScope)...)
	diagnostics = append(diagnostics, a.resolveJoinClauses(cteDefinition.JoinClauses, cteScope)...)

	cteColumns, _, cteDiagnostics := a.typeResolver.ResolveOutputColumns(ctx, cteDefinition.OutputColumns, cteScope)
	diagnostics = append(diagnostics, cteDiagnostics...)

	if cteDefinition.IsRecursive && len(cteDefinition.CompoundBranches) > 0 {
		scope.AddCTE(cteDefinition.Name, a.outputColumnsToScoped(cteColumns))
	}

	if len(cteDefinition.CompoundBranches) > 0 {
		branchDiagnostics := a.resolveCompoundBranches(ctx, cteDefinition.CompoundBranches, cteColumns, scope)
		diagnostics = append(diagnostics, branchDiagnostics...)
	}

	scope.AddCTE(cteDefinition.Name, a.outputColumnsToScoped(cteColumns))

	return diagnostics
}

// populateCTEScope adds FROM table references to a CTE's child scope, resolving against
// the parent scope's CTEs or catalogue.
//
// Takes fromTables ([]querier_dto.TableReference) which holds the FROM clause tables of
// the CTE body.
//
// Takes parentScope (*scopeChain) which holds the parent scope containing previously
// resolved CTEs.
//
// Takes cteScope (*scopeChain) which holds the child scope to populate.
//
// Returns []querier_dto.SourceError which holds any warnings when tables cannot be
// resolved.
func (a *queryAnalyser) populateCTEScope(
	fromTables []querier_dto.TableReference,
	parentScope *scopeChain,
	cteScope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, tableReference := range fromTables {
		if cte, exists := parentScope.ctes[strings.ToLower(tableReference.Name)]; exists {
			alias := tableReference.Alias
			if alias == "" {
				alias = tableReference.Name
			}
			cteScope.AddCTE(alias, cte.columns)
			continue
		}
		catalogueTable, resolveError := a.resolveTableReference(tableReference)
		if resolveError != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  resolveError.Error(),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownTable,
			})
			continue
		}
		_ = cteScope.AddTable(tableReference, querier_dto.JoinInner, catalogueTable)
	}

	return diagnostics
}

// outputColumnsToScoped converts output columns to scoped columns for CTE registration.
//
// Takes columns ([]querier_dto.OutputColumn) which holds the resolved output columns to
// convert.
//
// Returns []querier_dto.ScopedColumn which holds the converted scoped columns preserving
// name, type, and nullability.
func (*queryAnalyser) outputColumnsToScoped(columns []querier_dto.OutputColumn) []querier_dto.ScopedColumn {
	scoped := make([]querier_dto.ScopedColumn, len(columns))
	for i := range columns {
		scoped[i] = querier_dto.ScopedColumn{
			Name:     columns[i].Name,
			SQLType:  columns[i].SQLType,
			Nullable: columns[i].Nullable,
		}
	}
	return scoped
}

// resolveTableValuedFunctions resolves table-valued function references and adds them to
// the scope as derived tables.
//
// Takes tableValuedFunctions ([]querier_dto.RawTableValuedFunctionReference) which holds
// the parsed function references.
//
// Takes scope (*scopeChain) which holds the scope chain to populate with derived table
// entries.
//
// Returns []querier_dto.SourceError which holds any warnings for unresolvable function
// references.
func (a *queryAnalyser) resolveTableValuedFunctions(
	tableValuedFunctions []querier_dto.RawTableValuedFunctionReference,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	for _, tvf := range tableValuedFunctions {
		columns, columnDiagnostic := resolveTableValuedFunctionColumns(a.engine, a.catalogue, tvf)
		if columns == nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  fmt.Sprintf("%s: unknown table-valued function %q", querier_dto.CodeUnknownTable, tvf.FunctionName),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownTable,
			})
			continue
		}
		if columnDiagnostic != nil {
			diagnostics = append(diagnostics, *columnDiagnostic)
		}
		scope.AddDerivedTable(querier_dto.DerivedTableReference{
			Alias:    tvf.Alias,
			Columns:  columns,
			Source:   querier_dto.DerivedSourceTableFunction,
			JoinKind: tvf.JoinKind,
		})
	}
	return diagnostics
}

// resolveTableValuedFunctionColumns resolves the output columns of a table-valued
// function reference, independent of any analyser receiver so both the top-level query
// path and the inner-subquery scope builder share one implementation.
//
// Explicit column definitions win; otherwise the engine's built-in table-valued function
// schema is consulted, then (for engines that implement CatalogueFunctionResolverPort)
// the catalogue is searched for a user-defined table-valued function. The columns are nil
// when none of these resolve the function.
//
// Takes engine (EnginePort) which supplies the type normaliser and built-in table-valued
// function schema.
// Takes catalogue (*querier_dto.Catalogue) which holds user-defined functions for the
// optional catalogue fallback.
// Takes tvf (querier_dto.RawTableValuedFunctionReference) which holds the function
// reference with optional column definitions.
//
// Returns []querier_dto.ScopedColumn which holds the resolved columns, or nil if the
// function cannot be resolved.
// Returns *querier_dto.SourceError which is non-nil when the alias count exceeds the
// resolved column count.
func resolveTableValuedFunctionColumns(
	engine EnginePort,
	catalogue *querier_dto.Catalogue,
	tvf querier_dto.RawTableValuedFunctionReference,
) ([]querier_dto.ScopedColumn, *querier_dto.SourceError) {
	if len(tvf.ColumnDefinitions) > 0 && tvf.ColumnDefinitions[0].TypeName != "" {
		columns := make([]querier_dto.ScopedColumn, len(tvf.ColumnDefinitions))
		for i, definition := range tvf.ColumnDefinitions {
			columns[i] = querier_dto.ScopedColumn{
				Name:     definition.Name,
				SQLType:  engine.NormaliseTypeName(definition.TypeName),
				Nullable: true,
			}
		}
		return columns, nil
	}

	columns := engine.TableValuedFunctionColumns(tvf.FunctionName)
	if columns == nil {
		if resolver, ok := engine.(CatalogueFunctionResolverPort); ok {
			columns = resolver.TableValuedFunctionColumnsFromCatalogue(catalogue, tvf.FunctionName)
		}
	}
	if columns == nil {
		return nil, nil
	}

	return columns, applyTableValuedFunctionAliases(tvf, columns)
}

// applyTableValuedFunctionAliases renames the resolved columns of a table-valued function
// in place from the call's alias-only column definitions. Aliases beyond the resolved
// column count cannot be applied; in that case the in-range aliases are still applied and
// a warning diagnostic is returned describing the count mismatch.
//
// Takes tvf (querier_dto.RawTableValuedFunctionReference) which holds the alias
// definitions.
// Takes columns ([]querier_dto.ScopedColumn) which holds the resolved columns to rename.
//
// Returns *querier_dto.SourceError which is non-nil only when aliases outnumber columns.
func applyTableValuedFunctionAliases(
	tvf querier_dto.RawTableValuedFunctionReference,
	columns []querier_dto.ScopedColumn,
) *querier_dto.SourceError {
	if len(tvf.ColumnDefinitions) == 0 {
		return nil
	}
	applied := min(len(tvf.ColumnDefinitions), len(columns))
	for i := range applied {
		columns[i].Name = tvf.ColumnDefinitions[i].Name
	}
	if len(tvf.ColumnDefinitions) <= len(columns) {
		return nil
	}
	return &querier_dto.SourceError{
		Message: fmt.Sprintf(
			"%s: table-valued function %q exposes %d columns but %d column aliases were supplied; surplus aliases are ignored",
			querier_dto.CodeCompoundColumnCount, tvf.FunctionName, len(columns), len(tvf.ColumnDefinitions),
		),
		Severity: querier_dto.SeverityWarning,
		Code:     querier_dto.CodeCompoundColumnCount,
	}
}

// resolveRawDerivedTables resolves subquery-based derived tables by recursively analysing
// each inner query.
//
// Takes rawDerivedTables ([]querier_dto.RawDerivedTableReference) which holds the parsed
// derived table references.
//
// Takes scope (*scopeChain) which holds the scope chain to populate with resolved derived
// tables.
//
// Returns []querier_dto.SourceError which holds any diagnostics produced during
// resolution.
func (a *queryAnalyser) resolveRawDerivedTables(
	ctx context.Context,
	rawDerivedTables []querier_dto.RawDerivedTableReference,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, rawDerived := range rawDerivedTables {
		if ctx.Err() != nil {
			return diagnostics
		}
		if rawDerived.InnerQuery == nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  querier_dto.CodeInternalNilGuard + ": nil derived table query during type resolution",
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeInternalNilGuard,
			})
			continue
		}

		innerScope := newScopeChain(querier_dto.ScopeKindQuery, nil)

		cteDiagnostics := a.resolveCTEs(ctx, rawDerived.InnerQuery.CTEDefinitions, innerScope)
		diagnostics = append(diagnostics, cteDiagnostics...)

		scopeDiagnostics := a.buildScopeChain(rawDerived.InnerQuery, innerScope)
		diagnostics = append(diagnostics, scopeDiagnostics...)

		innerColumns, _, innerDiagnostics := a.typeResolver.ResolveOutputColumns(ctx, rawDerived.InnerQuery.OutputColumns, innerScope)
		diagnostics = append(diagnostics, innerDiagnostics...)

		scopedColumns := make([]querier_dto.ScopedColumn, len(innerColumns))
		for columnIndex := range innerColumns {
			scopedColumns[columnIndex] = querier_dto.ScopedColumn{
				Name:     innerColumns[columnIndex].Name,
				SQLType:  innerColumns[columnIndex].SQLType,
				Nullable: innerColumns[columnIndex].Nullable,
			}
		}

		scope.AddDerivedTable(querier_dto.DerivedTableReference{
			Alias:    rawDerived.Alias,
			Columns:  scopedColumns,
			JoinKind: rawDerived.JoinKind,
		})
	}

	return diagnostics
}

// resolveArrayJoinClauses registers each ClickHouse-style ARRAY JOIN alias as a
// single-column derived-table-style entry in the scope chain.
//
// The element type is looked up by walking the FROM tables in scope for a column matching
// the source name; the array element type is extracted and exposed under the alias. LEFT
// ARRAY JOIN entries promote the element type to Nullable because empty source arrays
// surface as NULL rows rather than collapsing.
//
// Takes clauses ([]querier_dto.RawArrayJoinClause) which are the unresolved ARRAY JOIN
// entries captured by the parser.
// Takes scope (*scopeChain) which already contains the FROM tables the array source must
// be looked up on.
//
// Returns []querier_dto.SourceError which holds Q001 diagnostics for any clause whose
// source column cannot be found in the FROM scope.
func (*queryAnalyser) resolveArrayJoinClauses(
	clauses []querier_dto.RawArrayJoinClause,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	for _, clause := range clauses {
		column, _, lookupErr := scope.ResolveColumn("", clause.SourceColumn)
		if lookupErr != nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  fmt.Sprintf("%s: array join source column %q", querier_dto.CodeUnknownColumn, clause.SourceColumn),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownColumn,
			})
			continue
		}
		element := column.SQLType
		if column.SQLType.Category == querier_dto.TypeCategoryArray && column.SQLType.ElementType != nil {
			element = *column.SQLType.ElementType
		}
		joinKind := querier_dto.JoinInner
		if clause.IsLeft {
			joinKind = querier_dto.JoinLeft
		}
		scope.AddDerivedTable(querier_dto.DerivedTableReference{
			Alias: clause.Alias,
			Columns: []querier_dto.ScopedColumn{{
				Name:     clause.Alias,
				SQLType:  element,
				Nullable: clause.IsLeft,
			}},
			JoinKind: joinKind,
		})
	}
	return diagnostics
}

// inheritOuterCTEs copies CTE definitions from an outer scope into a compound branch
// scope so that UNION/INTERSECT/EXCEPT branches can resolve table references to CTEs
// declared in the enclosing WITH clause.
//
// Branches do not get parent-traversal because they share peer status with the primary
// SELECT under a single compound expression, so we replicate the visible CTEs by value
// into the branch scope.
//
// Takes outerScope (*scopeChain) which holds the enclosing query scope. May be nil.
//
// Takes branchScope (*scopeChain) which holds the branch scope to receive the CTE
// entries.
func inheritOuterCTEs(outerScope *scopeChain, branchScope *scopeChain) {
	if outerScope == nil {
		return
	}
	names := slices.Sorted(maps.Keys(outerScope.ctes))
	for _, name := range names {
		cte := outerScope.ctes[name]
		branchScope.AddCTE(cte.name, cte.columns)
	}
}

// resolveCompoundBranches resolves UNION, INTERSECT, and EXCEPT branches and promotes
// types to match the primary SELECT.
//
// Takes branches ([]querier_dto.RawCompoundBranch) which holds the parsed compound query
// branches.
//
// Takes primaryColumns ([]querier_dto.OutputColumn) which holds the primary SELECT
// columns whose types are promoted in place.
//
// Takes outerScope (*scopeChain) which holds the scope chain of the enclosing query so
// each branch can see CTEs declared at the outer WITH clause. May be nil when no outer
// CTEs apply.
//
// Returns []querier_dto.SourceError which holds any diagnostics produced during branch
// resolution.
func (a *queryAnalyser) resolveCompoundBranches(
	ctx context.Context,
	branches []querier_dto.RawCompoundBranch,
	primaryColumns []querier_dto.OutputColumn,
	outerScope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, branch := range branches {
		if ctx.Err() != nil {
			return diagnostics
		}
		if branch.Query == nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  querier_dto.CodeInternalNilGuard + ": nil compound branch query during type resolution",
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeInternalNilGuard,
			})
			continue
		}

		branchScope := newScopeChain(querier_dto.ScopeKindQuery, nil)
		inheritOuterCTEs(outerScope, branchScope)

		cteDiagnostics := a.resolveCTEs(ctx, branch.Query.CTEDefinitions, branchScope)
		diagnostics = append(diagnostics, cteDiagnostics...)

		scopeDiagnostics := a.buildScopeChain(branch.Query, branchScope)
		diagnostics = append(diagnostics, scopeDiagnostics...)

		branchColumns, _, branchDiagnostics := a.typeResolver.ResolveOutputColumns(ctx, branch.Query.OutputColumns, branchScope)
		diagnostics = append(diagnostics, branchDiagnostics...)

		if len(branchColumns) != len(primaryColumns) {
			if len(branchDiagnostics) == 0 {
				diagnostics = append(diagnostics, querier_dto.SourceError{
					Message: fmt.Sprintf(
						"compound query branch has %d columns, expected %d to match primary SELECT",
						len(branchColumns), len(primaryColumns),
					),
					Severity: querier_dto.SeverityError,
					Code:     querier_dto.CodeCompoundColumnCount,
				})
			}
			continue
		}

		for columnIndex := range primaryColumns {
			primaryColumns[columnIndex].SQLType = a.engine.PromoteType(
				primaryColumns[columnIndex].SQLType,
				branchColumns[columnIndex].SQLType,
			)
			if branchColumns[columnIndex].Nullable {
				primaryColumns[columnIndex].Nullable = true
			}
		}
	}

	return diagnostics
}

// analyseStatements analyses parsed SQL statements, delegating to multi-statement
// analysis when available.
//
// Takes statements ([]querier_dto.ParsedStatement) which holds all parsed statements in
// the query block.
//
// Takes primaryStatement (querier_dto.ParsedStatement) which holds the last statement
// used for single-statement analysis.
//
// Returns *querier_dto.RawQueryAnalysis which holds the raw analysis result.
//
// Returns error when the engine fails to analyse the statements.
func (a *queryAnalyser) analyseStatements(
	statements []querier_dto.ParsedStatement,
	primaryStatement querier_dto.ParsedStatement,
) (*querier_dto.RawQueryAnalysis, error) {
	if multiAnalyser, ok := a.engine.(MultiStatementAnalyserPort); ok && len(statements) > 1 {
		return multiAnalyser.AnalyseMultiStatement(a.catalogue, statements)
	}
	return a.engine.AnalyseQuery(a.catalogue, primaryStatement)
}

// blockError constructs a SourceError positioned at the start of a query block.
//
// Takes filename (string) which specifies the source file path.
//
// Takes line (int) which specifies the line number within the file.
//
// Takes code (string) which specifies the diagnostic error code.
//
// Takes severity (querier_dto.ErrorSeverity) which specifies the error severity level.
//
// Takes message (string) which specifies the human-readable error message.
//
// Returns querier_dto.SourceError which holds the constructed source error.
func blockError(filename string, line int, code string, severity querier_dto.ErrorSeverity, message string) querier_dto.SourceError {
	return querier_dto.SourceError{
		Filename: filename,
		Line:     line,
		Column:   1,
		Message:  message,
		Severity: severity,
		Code:     code,
	}
}

// addFileLocation fills in missing file location fields on a slice of diagnostics.
//
// Takes diagnostics ([]querier_dto.SourceError) which holds the diagnostics to augment.
//
// Takes filename (string) which specifies the default filename for diagnostics that lack
// one.
//
// Takes startLine (int) which specifies the default line number for diagnostics that lack
// one.
//
// Returns []querier_dto.SourceError which holds a new slice with file locations filled
// in.
func addFileLocation(
	diagnostics []querier_dto.SourceError,
	filename string,
	startLine int,
) []querier_dto.SourceError {
	result := make([]querier_dto.SourceError, len(diagnostics))
	for i, diagnostic := range diagnostics {
		result[i] = diagnostic
		if result[i].Filename == "" {
			result[i].Filename = filename
		}
		if result[i].Line == 0 {
			result[i].Line = startLine
		}
		if result[i].Column == 0 {
			result[i].Column = 1
		}
	}
	return result
}

// resolveEmbedDirectives scans SQL for inline piko.embed comments and marks matching
// output columns as embedded.
//
// It sets IsEmbedded, EmbedTable, and EmbedIsOuter on each column whose SourceTable
// matches an embed directive table name.
//
// Takes directiveBlock (*querier_dto.DirectiveBlock) which holds the parsed embed
// declarations from the query header (the canonical source of embed metadata).
//
// Takes sql (string) which holds the raw SQL text. Kept for backward compatibility with
// any inline embed markers still present in legacy fixtures; the parsed directive block
// takes precedence when both forms are present.
//
// Takes outputColumns ([]querier_dto.OutputColumn) which holds the resolved output
// columns to annotate.
//
// Takes scope (*scopeChain) which holds the scope chain used to determine outer join
// status.
//
// Returns []querier_dto.OutputColumn which holds the updated output columns with embed
// annotations applied.
func resolveEmbedDirectives(
	directiveBlock *querier_dto.DirectiveBlock,
	sql string,
	outputColumns []querier_dto.OutputColumn,
	scope *scopeChain,
) []querier_dto.OutputColumn {
	embedTables := embedTablesFromDirectives(directiveBlock)
	if len(embedTables) == 0 {
		embedTables = extractEmbedTableNames(sql)
	}
	if len(embedTables) == 0 {
		return outputColumns
	}

	for columnIndex := range outputColumns {
		column := &outputColumns[columnIndex]
		if column.SourceTable == "" {
			continue
		}
		for _, embedTable := range embedTables {
			if !strings.EqualFold(column.SourceTable, embedTable) {
				continue
			}
			column.IsEmbedded = true
			column.EmbedTable = embedTable
			column.EmbedIsOuter = isOuterJoinTable(embedTable, scope)
			break
		}
	}

	return outputColumns
}

// applyQueryColumnOverrides applies each `-- piko.column(name, ...)` directive declared
// in the query header to the resolved output columns.
//
// For each override the Name is matched case-insensitively against the output column
// names, and a miss produces Q036 with a Levenshtein suggestion drawn from the actual
// column names. When `type:` was declared, the engine normalises the value into a
// structured SQLType that replaces the column's inferred type. When `nullable:` was
// declared, the column's Nullable flag is replaced. When `go_type:` was declared, the
// value is split on the last "." into Package and Name and stored on
// OutputColumn.GoTypeOverride for the emitter to consume.
//
// Takes outputColumns ([]querier_dto.OutputColumn) which holds the resolved output
// columns to override.
// Takes directiveBlock (*querier_dto.DirectiveBlock) which holds the parsed column
// overrides from the query header.
// Takes filename (string) which is the source file path for diagnostic locations.
// Takes diagnostics (*[]querier_dto.SourceError) which collects any unknown-column
// diagnostics produced.
//
// Returns []querier_dto.OutputColumn which is the updated output column slice.
func (a *queryAnalyser) applyQueryColumnOverrides(
	outputColumns []querier_dto.OutputColumn,
	directiveBlock *querier_dto.DirectiveBlock,
	filename string,
	diagnostics *[]querier_dto.SourceError,
) []querier_dto.OutputColumn {
	if directiveBlock == nil || len(directiveBlock.ColumnOverrides) == 0 {
		return outputColumns
	}

	columnNames := make([]string, 0, len(outputColumns))
	for index := range outputColumns {
		columnNames = append(columnNames, outputColumns[index].Name)
	}
	errorBuilder := querier_dto.NewErrorBuilder(filename)

	for _, override := range directiveBlock.ColumnOverrides {
		matched := findOutputColumnByName(outputColumns, override.Name)
		if matched == -1 {
			*diagnostics = append(*diagnostics, errorBuilder.UnknownOverrideColumn(override.NameSpan, override.Name, columnNames))
			continue
		}
		a.applyOverrideToColumn(&outputColumns[matched], override)
	}
	return outputColumns
}

// findOutputColumnByName scans outputColumns for the first entry whose Name matches
// target case-insensitively.
//
// Takes outputColumns ([]querier_dto.OutputColumn) which holds the columns to scan.
// Takes target (string) which is the column name to match.
//
// Returns int which is the matching index, or -1 when no match exists.
func findOutputColumnByName(outputColumns []querier_dto.OutputColumn, target string) int {
	for index := range outputColumns {
		if strings.EqualFold(outputColumns[index].Name, target) {
			return index
		}
	}
	return -1
}

// applyOverrideToColumn copies the SQL type, nullable, and Go-type override fields from a
// parsed query-level directive onto the matched output column. Empty values leave the
// underlying column field unchanged, matching the directive parser's optionality.
//
// Takes column (*querier_dto.OutputColumn) which receives the override fields.
// Takes override (*querier_dto.ColumnOverride) which holds the parsed directive values.
func (a *queryAnalyser) applyOverrideToColumn(column *querier_dto.OutputColumn, override *querier_dto.ColumnOverride) {
	if override.SQLType != "" {
		column.SQLType = a.engine.NormaliseTypeName(override.SQLType)
	}
	if override.Nullable != nil {
		column.Nullable = *override.Nullable
	}
	if override.GoType != "" {
		lastDot := strings.LastIndex(override.GoType, ".")
		if lastDot > 0 && lastDot < len(override.GoType)-1 {
			column.GoTypeOverride = &querier_dto.GoType{
				Package: override.GoType[:lastDot],
				Name:    override.GoType[lastDot+1:],
			}
		}
	}
}

// propagateCatalogueColumnOverrides walks the output columns and, for each one whose
// lineage traces back to a catalogue column via direct projection (SourceTable +
// SourceColumn both set), applies any migration-level override declared on the catalogue
// column.
//
// Casts, function calls, and computed expressions leave SourceTable empty, so the
// override drops on those. Query-level overrides already applied by
// applyQueryColumnOverrides take precedence: this pass only fills slots that are still
// unmodified by the directive block.
//
// Takes outputColumns ([]querier_dto.OutputColumn) which holds the resolved output
// columns.
// Takes directiveBlock (*querier_dto.DirectiveBlock) which holds the query-level
// overrides that take precedence over catalogue overrides.
//
// Returns []querier_dto.OutputColumn which is the updated output column slice.
func (a *queryAnalyser) propagateCatalogueColumnOverrides(
	outputColumns []querier_dto.OutputColumn,
	directiveBlock *querier_dto.DirectiveBlock,
) []querier_dto.OutputColumn {
	queryLevelNames := make(map[string]struct{}, len(directiveBlock.ColumnOverrides))
	for _, override := range directiveBlock.ColumnOverrides {
		queryLevelNames[strings.ToLower(override.Name)] = struct{}{}
	}

	for columnIndex := range outputColumns {
		a.propagateOverrideToOutputColumn(&outputColumns[columnIndex], queryLevelNames)
	}
	return outputColumns
}

// propagateOverrideToOutputColumn applies any migration-level overrides declared on the
// catalogue column to a single output column. Skips columns that lack a direct projection
// lineage or have already been overridden at the query level (queryLevelNames lookup).
//
// Takes column (*querier_dto.OutputColumn) which receives the catalogue override.
// Takes queryLevelNames (map[string]struct{}) which holds the lower-cased names already
// overridden at the query level.
func (a *queryAnalyser) propagateOverrideToOutputColumn(column *querier_dto.OutputColumn, queryLevelNames map[string]struct{}) {
	if column.SourceTable == "" || column.SourceColumn == "" {
		return
	}
	if _, queryOverridden := queryLevelNames[strings.ToLower(column.Name)]; queryOverridden {
		return
	}
	catalogueColumn := findCatalogueColumn(a.catalogue, column.SourceTable, column.SourceColumn)
	if catalogueColumn == nil {
		return
	}
	if catalogueColumn.SQLTypeOverride != "" {
		column.SQLType = a.engine.NormaliseTypeName(catalogueColumn.SQLTypeOverride)
	}
	if catalogueColumn.NullableOverride != nil {
		column.Nullable = *catalogueColumn.NullableOverride
	}
	if catalogueColumn.GoTypeOverride != nil && column.GoTypeOverride == nil {
		column.GoTypeOverride = new(*catalogueColumn.GoTypeOverride)
	}
}

// embedTablesFromDirectives returns the table names declared by `-- piko.embed(table,
// ...)` header directives on the parsed block.
//
// Takes directiveBlock (*querier_dto.DirectiveBlock) which holds the parsed embed
// declarations.
//
// Returns []string which holds the declared embed table names, or nil when no embeds are
// present.
func embedTablesFromDirectives(directiveBlock *querier_dto.DirectiveBlock) []string {
	if directiveBlock == nil || len(directiveBlock.Embeds) == 0 {
		return nil
	}
	tables := make([]string, 0, len(directiveBlock.Embeds))
	for _, embed := range directiveBlock.Embeds {
		if embed != nil && embed.Table != "" {
			tables = append(tables, embed.Table)
		}
	}
	return tables
}

// extractEmbedTableNames finds all table names referenced by inline piko.embed comments
// in the SQL text.
//
// Takes sql (string) which holds the raw SQL text to scan.
//
// Returns []string which holds the extracted table names in the order they appear.
func extractEmbedTableNames(sql string) []string {
	var tables []string
	marker := "/* piko.embed("
	searchPosition := 0

	for searchPosition < len(sql) {
		startIndex := strings.Index(sql[searchPosition:], marker)
		if startIndex == -1 {
			break
		}
		startIndex += searchPosition
		nameStart := startIndex + len(marker)
		closeIndex := strings.Index(sql[nameStart:], ")")
		if closeIndex == -1 {
			break
		}
		tableName := strings.TrimSpace(sql[nameStart : nameStart+closeIndex])
		if tableName != "" {
			tables = append(tables, tableName)
		}
		searchPosition = nameStart + closeIndex + 1
	}

	return tables
}

// isOuterJoinTable checks whether the given table was introduced via a LEFT, RIGHT, or
// FULL JOIN.
//
// Takes tableName (string) which specifies the table name or alias to look up.
//
// Takes scope (*scopeChain) which holds the scope chain containing resolved table
// entries.
//
// Returns bool which indicates whether the table's join kind implies nullable columns.
func isOuterJoinTable(tableName string, scope *scopeChain) bool {
	for _, table := range scope.tables {
		if !strings.EqualFold(table.Name, tableName) && !strings.EqualFold(table.Alias, tableName) {
			continue
		}
		switch table.JoinKind { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case querier_dto.JoinLeft, querier_dto.JoinRight, querier_dto.JoinFull, querier_dto.JoinPositional:
			return true
		}
		return false
	}
	return false
}
