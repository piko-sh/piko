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

// computeDynamicFlags determines whether a query is
// dynamic and collects sortable column names.
//
// Takes parameters ([]*querier_dto.ParameterDirective)
// which holds the parsed parameter directives for the
// query.
//
// Returns bool which indicates whether any parameter
// makes the query dynamic.
//
// Returns []string which holds the column names
// declared on sortable parameters.
func computeDynamicFlags(parameters []*querier_dto.ParameterDirective) (bool, []string) {
	isDynamic := false
	var dynamicColumns []string
	for _, parameter := range parameters {
		switch parameter.Kind {
		case querier_dto.ParameterDirectiveOptional,
			querier_dto.ParameterDirectiveSortable,
			querier_dto.ParameterDirectiveLimit,
			querier_dto.ParameterDirectiveOffset:
			isDynamic = true
		}
		if parameter.Kind == querier_dto.ParameterDirectiveSortable {
			dynamicColumns = append(dynamicColumns, parameter.Columns...)
		}
	}
	return isDynamic, dynamicColumns
}

// assembleQueryInput groups the inputs needed to construct an AnalysedQuery result.
type assembleQueryInput struct {
	// directives holds the parsed query-level directives
	// such as group_by and nullable overrides.
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

// assembleQuery constructs an AnalysedQuery from the
// provided input fields.
//
// Takes input (assembleQueryInput) which holds all
// resolved query components.
//
// Returns *querier_dto.AnalysedQuery which holds the
// fully assembled query ready for code generation.
func (a *queryAnalyser) assembleQuery(input assembleQueryInput) *querier_dto.AnalysedQuery {
	readOnly := input.rawAnalysis.ReadOnly
	if input.directives.ReadOnlyOverride != nil {
		readOnly = *input.directives.ReadOnlyOverride
	}

	query := &querier_dto.AnalysedQuery{
		Name:           input.queryName,
		Command:        input.queryCommand,
		SQL:            input.block.sql,
		Filename:       input.filename,
		Line:           input.block.startLine,
		OutputColumns:  input.outputColumns,
		Parameters:     input.parameters,
		IsDynamic:      input.isDynamic,
		GroupByKey:     input.directives.GroupByKeys,
		Directives:     *input.directives,
		InsertTable:    input.rawAnalysis.InsertTable,
		InsertColumns:  input.rawAnalysis.InsertColumns,
		DynamicRuntime: input.directives.DynamicRuntime,
		ReadOnly:       readOnly,
	}

	if input.directives.DynamicRuntime {
		query.AllowedColumns = a.extractAllowedColumns(input.rawAnalysis)
	}

	return query
}

// extractAllowedColumns collects the columns from all
// FROM and JOIN tables for dynamic runtime queries.
//
// Takes raw (*querier_dto.RawQueryAnalysis) which holds
// the parsed table references.
//
// Returns []querier_dto.AllowedColumn which holds the
// deduplicated set of columns available for dynamic
// ordering.
func (a *queryAnalyser) extractAllowedColumns(raw *querier_dto.RawQueryAnalysis) []querier_dto.AllowedColumn {
	var allowed []querier_dto.AllowedColumn
	seen := make(map[string]struct{})

	for _, tableReference := range raw.FromTables {
		table := a.findTable(tableReference.Schema, tableReference.Name)
		allowed = appendUniqueColumns(allowed, seen, table)
	}

	for _, joinClause := range raw.JoinClauses {
		table := a.findTable(joinClause.Table.Schema, joinClause.Table.Name)
		allowed = appendUniqueColumns(allowed, seen, table)
	}

	return allowed
}

// appendUniqueColumns appends columns from the given
// table that have not already been seen.
//
// Takes allowed ([]querier_dto.AllowedColumn) which
// holds the accumulated columns so far.
//
// Takes seen (map[string]struct{}) which tracks column
// names already added.
//
// Takes table (*querier_dto.Table) which holds the
// catalogue table whose columns are appended.
//
// Returns []querier_dto.AllowedColumn which holds the
// updated column list with any new entries appended.
func appendUniqueColumns(
	allowed []querier_dto.AllowedColumn,
	seen map[string]struct{},
	table *querier_dto.Table,
) []querier_dto.AllowedColumn {
	if table == nil {
		return allowed
	}
	for j := range table.Columns {
		if _, exists := seen[table.Columns[j].Name]; exists {
			continue
		}
		seen[table.Columns[j].Name] = struct{}{}
		allowed = append(allowed, querier_dto.AllowedColumn{
			Name:    table.Columns[j].Name,
			SQLType: table.Columns[j].SQLType,
		})
	}
	return allowed
}

// findTable looks up a table in the catalogue by schema
// and name.
//
// Takes schema (string) which specifies the schema
// name, defaulting to the catalogue default if empty.
//
// Takes name (string) which specifies the table name to
// look up.
//
// Returns *querier_dto.Table which holds the matched
// table, or nil if not found.
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

// buildScopeChain populates the scope chain from a raw
// query analysis result.
//
// FROM tables and JOIN tables are added with correct
// nullability adjustments. FROM entries that match
// previously resolved CTEs are registered as CTE
// aliases rather than resolved against the catalogue.
//
// Takes raw (*querier_dto.RawQueryAnalysis) which holds
// the parsed FROM, JOIN, and derived table references.
//
// Takes scope (*scopeChain) which holds the scope chain
// to populate.
//
// Returns []querier_dto.SourceError which holds any
// warnings produced when tables cannot be resolved.
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

// resolveJoinClauses adds each join clause to the scope
// chain, resolving CTE references or catalogue tables.
//
// Takes joinClauses ([]querier_dto.JoinClause) which
// holds the parsed JOIN clauses from the query.
//
// Takes scope (*scopeChain) which holds the scope chain
// to populate with join table entries.
//
// Returns []querier_dto.SourceError which holds any
// warnings produced when join tables cannot be
// resolved.
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

// resolveTableReference looks up a table or view in the
// catalogue by its reference.
//
// Takes reference (querier_dto.TableReference) which
// specifies the schema, name, and alias of the table.
//
// Returns *querier_dto.Table which holds the matched
// catalogue table or view.
//
// Returns error when the schema or table name cannot be
// found in the catalogue.
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

// resolveCTEs resolves each CTE definition in order and
// registers the results in the scope chain.
//
// Takes ctx (context.Context) which controls
// cancellation of the resolution.
//
// Takes cteDefinitions ([]querier_dto.RawCTEDefinition)
// which holds the parsed CTE definitions from the
// query.
//
// Takes scope (*scopeChain) which holds the scope chain
// where resolved CTEs are registered.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics produced during CTE resolution.
func (a *queryAnalyser) resolveCTEs(
	ctx context.Context,
	cteDefinitions []querier_dto.RawCTEDefinition,
	scope *scopeChain,
) []querier_dto.SourceError {
	diagnostics := make([]querier_dto.SourceError, 0, len(cteDefinitions))

	for _, cteDefinition := range cteDefinitions {
		diagnostics = append(diagnostics, a.resolveSingleCTE(ctx, cteDefinition, scope)...)
	}

	return diagnostics
}

// resolveSingleCTE resolves one CTE definition by
// building a child scope, resolving its output columns,
// and registering it.
//
// Takes ctx (context.Context) which controls
// cancellation.
//
// Takes cteDefinition (querier_dto.RawCTEDefinition)
// which holds the parsed CTE definition to resolve.
//
// Takes scope (*scopeChain) which holds the parent
// scope chain where the CTE is registered.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics produced during resolution.
func (a *queryAnalyser) resolveSingleCTE(
	ctx context.Context,
	cteDefinition querier_dto.RawCTEDefinition,
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
		branchDiagnostics := a.resolveCompoundBranches(ctx, cteDefinition.CompoundBranches, cteColumns)
		diagnostics = append(diagnostics, branchDiagnostics...)
	}

	scope.AddCTE(cteDefinition.Name, a.outputColumnsToScoped(cteColumns))

	return diagnostics
}

// populateCTEScope adds FROM table references to a
// CTE's child scope, resolving against the parent
// scope's CTEs or catalogue.
//
// Takes fromTables ([]querier_dto.TableReference) which
// holds the FROM clause tables of the CTE body.
//
// Takes parentScope (*scopeChain) which holds the
// parent scope containing previously resolved CTEs.
//
// Takes cteScope (*scopeChain) which holds the child
// scope to populate.
//
// Returns []querier_dto.SourceError which holds any
// warnings when tables cannot be resolved.
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

// outputColumnsToScoped converts output columns to
// scoped columns for CTE registration.
//
// Takes columns ([]querier_dto.OutputColumn) which
// holds the resolved output columns to convert.
//
// Returns []querier_dto.ScopedColumn which holds the
// converted scoped columns preserving name, type, and
// nullability.
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

// resolveTableValuedFunctions resolves table-valued
// function references and adds them to the scope as
// derived tables.
//
// Takes tableValuedFunctions
// ([]querier_dto.RawTableValuedFunctionReference) which
// holds the parsed function references.
//
// Takes scope (*scopeChain) which holds the scope chain
// to populate with derived table entries.
//
// Returns []querier_dto.SourceError which holds any
// warnings for unresolvable function references.
func (a *queryAnalyser) resolveTableValuedFunctions(
	tableValuedFunctions []querier_dto.RawTableValuedFunctionReference,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	for _, tvf := range tableValuedFunctions {
		columns := a.resolveColumnDefinitionsOrLookup(tvf)
		if columns == nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  fmt.Sprintf("%s: unknown table-valued function %q", querier_dto.CodeUnknownTable, tvf.FunctionName),
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeUnknownTable,
			})
			continue
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

// resolveColumnDefinitionsOrLookup resolves columns for
// a table-valued function, preferring explicit
// definitions over catalogue lookup.
//
// Takes tvf (querier_dto.RawTableValuedFunctionReference)
// which holds the function reference with optional
// column definitions.
//
// Returns []querier_dto.ScopedColumn which holds the
// resolved columns, or nil if the function cannot be
// resolved.
func (a *queryAnalyser) resolveColumnDefinitionsOrLookup(
	tvf querier_dto.RawTableValuedFunctionReference,
) []querier_dto.ScopedColumn {
	if len(tvf.ColumnDefinitions) > 0 && tvf.ColumnDefinitions[0].TypeName != "" {
		columns := make([]querier_dto.ScopedColumn, len(tvf.ColumnDefinitions))
		for i, definition := range tvf.ColumnDefinitions {
			sqlType := a.engine.NormaliseTypeName(definition.TypeName)
			columns[i] = querier_dto.ScopedColumn{
				Name:     definition.Name,
				SQLType:  sqlType,
				Nullable: true,
			}
		}
		return columns
	}

	columns := a.engine.TableValuedFunctionColumns(tvf.FunctionName)
	if columns == nil {
		if resolver, ok := a.engine.(CatalogueFunctionResolverPort); ok {
			columns = resolver.TableValuedFunctionColumnsFromCatalogue(a.catalogue, tvf.FunctionName)
		}
	}
	if columns == nil {
		return nil
	}

	if len(tvf.ColumnDefinitions) > 0 && len(tvf.ColumnDefinitions) <= len(columns) {
		for i, definition := range tvf.ColumnDefinitions {
			columns[i].Name = definition.Name
		}
	}

	return columns
}

// resolveRawDerivedTables resolves subquery-based
// derived tables by recursively analysing each inner
// query.
//
// Takes ctx (context.Context) which controls
// cancellation of the resolution.
//
// Takes rawDerivedTables
// ([]querier_dto.RawDerivedTableReference) which holds
// the parsed derived table references.
//
// Takes scope (*scopeChain) which holds the scope chain
// to populate with resolved derived tables.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics produced during resolution.
func (a *queryAnalyser) resolveRawDerivedTables(
	ctx context.Context,
	rawDerivedTables []querier_dto.RawDerivedTableReference,
	scope *scopeChain,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, rawDerived := range rawDerivedTables {
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

// resolveCompoundBranches resolves UNION, INTERSECT,
// and EXCEPT branches and promotes types to match the
// primary SELECT.
//
// Takes ctx (context.Context) which controls
// cancellation.
//
// Takes branches ([]querier_dto.RawCompoundBranch)
// which holds the parsed compound query branches.
//
// Takes primaryColumns ([]querier_dto.OutputColumn)
// which holds the primary SELECT columns whose types
// are promoted in place.
//
// Returns []querier_dto.SourceError which holds any
// diagnostics produced during branch resolution.
func (a *queryAnalyser) resolveCompoundBranches(
	ctx context.Context,
	branches []querier_dto.RawCompoundBranch,
	primaryColumns []querier_dto.OutputColumn,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	for _, branch := range branches {
		if branch.Query == nil {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message:  querier_dto.CodeInternalNilGuard + ": nil compound branch query during type resolution",
				Severity: querier_dto.SeverityWarning,
				Code:     querier_dto.CodeInternalNilGuard,
			})
			continue
		}

		branchScope := newScopeChain(querier_dto.ScopeKindQuery, nil)

		cteDiagnostics := a.resolveCTEs(ctx, branch.Query.CTEDefinitions, branchScope)
		diagnostics = append(diagnostics, cteDiagnostics...)

		scopeDiagnostics := a.buildScopeChain(branch.Query, branchScope)
		diagnostics = append(diagnostics, scopeDiagnostics...)

		branchColumns, _, branchDiagnostics := a.typeResolver.ResolveOutputColumns(ctx, branch.Query.OutputColumns, branchScope)
		diagnostics = append(diagnostics, branchDiagnostics...)

		if len(branchColumns) != len(primaryColumns) {
			diagnostics = append(diagnostics, querier_dto.SourceError{
				Message: fmt.Sprintf(
					"compound query branch has %d columns, expected %d to match primary SELECT",
					len(branchColumns), len(primaryColumns),
				),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeCompoundColumnCount,
			})
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

// analyseStatements analyses parsed SQL statements,
// delegating to multi-statement analysis when
// available.
//
// Takes statements ([]querier_dto.ParsedStatement)
// which holds all parsed statements in the query block.
//
// Takes primaryStatement (querier_dto.ParsedStatement)
// which holds the last statement used for
// single-statement analysis.
//
// Returns *querier_dto.RawQueryAnalysis which holds the
// raw analysis result.
//
// Returns error when the engine fails to analyse the
// statements.
func (a *queryAnalyser) analyseStatements(
	statements []querier_dto.ParsedStatement,
	primaryStatement querier_dto.ParsedStatement,
) (*querier_dto.RawQueryAnalysis, error) {
	if multiAnalyser, ok := a.engine.(MultiStatementAnalyserPort); ok && len(statements) > 1 {
		return multiAnalyser.AnalyseMultiStatement(a.catalogue, statements)
	}
	return a.engine.AnalyseQuery(a.catalogue, primaryStatement)
}

// blockError constructs a SourceError positioned at the
// start of a query block.
//
// Takes filename (string) which specifies the source
// file path.
//
// Takes line (int) which specifies the line number
// within the file.
//
// Takes code (string) which specifies the diagnostic
// error code.
//
// Takes severity (querier_dto.ErrorSeverity) which
// specifies the error severity level.
//
// Takes message (string) which specifies the
// human-readable error message.
//
// Returns querier_dto.SourceError which holds the
// constructed source error.
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

// addFileLocation fills in missing file location fields
// on a slice of diagnostics.
//
// Takes diagnostics ([]querier_dto.SourceError) which
// holds the diagnostics to augment.
//
// Takes filename (string) which specifies the default
// filename for diagnostics that lack one.
//
// Takes startLine (int) which specifies the default
// line number for diagnostics that lack one.
//
// Returns []querier_dto.SourceError which holds a new
// slice with file locations filled in.
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

// resolveEmbedDirectives scans SQL for inline
// piko.embed comments and marks matching output columns
// as embedded.
//
// It sets IsEmbedded, EmbedTable, and EmbedIsOuter on
// each column whose SourceTable matches an embed
// directive table name.
//
// Takes sql (string) which holds the raw SQL text to
// scan for embed comments.
//
// Takes outputColumns ([]querier_dto.OutputColumn)
// which holds the resolved output columns to annotate.
//
// Takes scope (*scopeChain) which holds the scope chain
// used to determine outer join status.
//
// Returns []querier_dto.OutputColumn which holds the
// updated output columns with embed annotations
// applied.
func resolveEmbedDirectives(
	sql string,
	outputColumns []querier_dto.OutputColumn,
	scope *scopeChain,
) []querier_dto.OutputColumn {
	embedTables := extractEmbedTableNames(sql)
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

// extractEmbedTableNames finds all table names
// referenced by inline piko.embed comments in the SQL
// text.
//
// Takes sql (string) which holds the raw SQL text to
// scan.
//
// Returns []string which holds the extracted table
// names in the order they appear.
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

// isOuterJoinTable checks whether the given table was
// introduced via a LEFT, RIGHT, or FULL JOIN.
//
// Takes tableName (string) which specifies the table
// name or alias to look up.
//
// Takes scope (*scopeChain) which holds the scope chain
// containing resolved table entries.
//
// Returns bool which indicates whether the table's join
// kind implies nullable columns.
func isOuterJoinTable(tableName string, scope *scopeChain) bool {
	for _, table := range scope.tables {
		if !strings.EqualFold(table.Name, tableName) && !strings.EqualFold(table.Alias, tableName) {
			continue
		}
		switch table.JoinKind {
		case querier_dto.JoinLeft, querier_dto.JoinRight, querier_dto.JoinFull, querier_dto.JoinPositional:
			return true
		}
		return false
	}
	return false
}
