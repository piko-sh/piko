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

package querier_dto

// FunctionCatalogue holds built-in function signatures for an engine.
type FunctionCatalogue struct {
	// Functions maps function names to their overloaded signatures.
	Functions map[string][]*FunctionSignature
}

// TypeCatalogue holds built-in type definitions for an engine.
type TypeCatalogue struct {
	// Types maps normalised type names to their SQLType definitions.
	Types map[string]SQLType
}

// TypeMappingTable holds the complete set of SQL-to-Go type mappings, combining framework
// defaults with user overrides.
type TypeMappingTable struct {
	// Mappings holds the ordered list of type mappings. Later entries override earlier ones
	// for the same SQL type.
	Mappings []TypeMapping
}

// TypeMapping maps a SQL type to Go types for both nullable and non-nullable contexts.
type TypeMapping struct {
	// NotNull is the Go type to use when the column is NOT NULL.
	NotNull GoType

	// Nullable is the Go type to use when the column is nullable.
	Nullable GoType

	// SQLName is an optional engine-specific name for finer-grained matching (e.g. "numeric"
	// vs "decimal"), where empty matches any name in the category.
	SQLName string

	// SQLCategory is the type category this mapping applies to.
	SQLCategory SQLTypeCategory
}

// GoType identifies a Go type by its import path and name.
type GoType struct {
	// Package is the Go import path (e.g. "piko.sh/piko/wdk/maths"), or empty for built-in
	// types (string, int64, etc.).
	Package string

	// Name is the Go type name (e.g. "Decimal", "string", "*int64").
	Name string
}

// AnalysedQuery is the fully resolved result of query analysis, with all types and
// nullability propagated. Fields are ordered for natural Go alignment: strings first,
// then slices, then the Directives struct, then ints, then the QueryCommand enum, and
// finally booleans grouped at the end to minimise padding.
type AnalysedQuery struct {
	// InsertTable is the target table for :copyfrom queries.
	InsertTable string

	// Filename is the source SQL file path.
	Filename string

	// SQL is the original SQL text.
	SQL string

	// Name is the query name from the piko.name directive.
	Name string

	// CountSQL is the pre-derived `SELECT COUNT(*) ...` form of the query for use by the
	// runtime builder's .Count(ctx) terminal. Empty when the query is not piko.dynamic:
	// runtime or when count rewriting is not applicable.
	CountSQL string

	// Parameters holds the fully typed query parameters.
	Parameters []QueryParameter

	// OutputColumns holds the fully typed output columns.
	OutputColumns []OutputColumn

	// GroupByKey holds the column(s) used for one-to-many grouping, if any.
	GroupByKey []string

	// InsertColumns holds the target column names for :copyfrom queries.
	InsertColumns []string

	// AllowedColumns holds the columns that can be used in runtime WHERE and ORDER BY
	// clauses.
	AllowedColumns []AllowedColumn

	// Directives holds all parsed piko. directives for the query.
	Directives QueryDirectives

	// Line is the line number of the query in the source file.
	Line int

	// Command is the execution pattern from the piko.command directive.
	Command QueryCommand

	// IsDynamic indicates the query has optional WHERE or ORDER BY clauses.
	IsDynamic bool

	// DynamicRuntime indicates this query uses piko.dynamic: runtime to generate a fluent
	// runtime builder instead of a standard method.
	DynamicRuntime bool

	// ReadOnly indicates the query does not modify data. Downstream consumers can use this
	// to route read-only queries to read replicas.
	ReadOnly bool

	// BaseQueryHasWhereClause indicates that the SQL written in the .sql file already
	// carries a WHERE clause.
	//
	// The runtime query builder uses this to emit " AND " instead of " WHERE " when
	// appending runtime predicates, so a base query that filters by environment_id (or any
	// other static predicate) does not produce a duplicate-WHERE syntax error.
	BaseQueryHasWhereClause bool

	// CountSQLWrapped reports whether the count rewriter wrapped the original query in
	// `SELECT COUNT(*) FROM (<original>) sub` because of GROUP BY, DISTINCT, or
	// window-function semantics. Used by the diagnostic pass to emit Q023 so the webdev
	// understands the count is over outer rows.
	CountSQLWrapped bool
}

// AllowedColumn represents a column available for runtime query building.
type AllowedColumn struct {
	// Name is the column's output (result-set) name, which is what a caller passes to the
	// runtime builder's Where and OrderBy methods.
	Name string

	// SourceExpression is the qualified source reference the builder emits into the SQL in
	// place of the caller's text (for example "users.email"). It is unambiguous across joins
	// and references the real column behind an aliased projection.
	SourceExpression string

	// SQLType is the column's resolved SQL type.
	SQLType SQLType
}

// OutputColumn is a fully typed output column.
type OutputColumn struct {
	// GoTypeOverride, when non-nil, carries the custom Go destination type declared via a
	// piko.column(name, go_type: ...) directive (either in the query header for a per-query
	// override or propagated from the catalogue for a migration-level override). When set,
	// the emitter uses this directly instead of mapping SQLType through the engine type
	// registry.
	GoTypeOverride *GoType

	// Name is the column name or alias.
	Name string

	// SourceTable is the source table, if this is a direct column reference.
	SourceTable string

	// SourceSchema is the schema of SourceTable, when known, so a lookup against the
	// catalogue is unambiguous even when two schemas hold a table of the same name. Empty
	// for CTE/derived sources or when the schema could not be attributed.
	SourceSchema string

	// SourceColumn is the source column name, if this is a direct reference.
	SourceColumn string

	// SourceQualifier is the table reference that qualifies SourceColumn in the FROM clause.
	//
	// Holds the alias when the query aliased the table, otherwise the table or CTE name. The
	// dynamic runtime builder emits "<SourceQualifier>.<SourceColumn>" so a filter or
	// ordering on a projected column is unambiguous across joins and references the real
	// column rather than the caller's text. Empty when no qualifier is known.
	SourceQualifier string

	// EmbedTable is the table name for embedded columns.
	EmbedTable string

	// SQLType is the resolved SQL type.
	SQLType SQLType

	// Nullable indicates whether the column can be NULL, accounting for JOIN nullability,
	// expression nullability, and aggregate behaviour.
	Nullable bool

	// IsEmbedded indicates this column is part of a piko.embed group.
	IsEmbedded bool

	// EmbedIsOuter indicates the embedded table was introduced via LEFT, RIGHT, or FULL
	// JOIN. When true, the emitter generates a pointer type for the embedded struct (nil
	// when no matching row exists).
	EmbedIsOuter bool
}

// QueryParameter is a fully typed query parameter.
type QueryParameter struct {
	// DefaultLimit holds the default value applied when the caller omits a numeric parameter
	// (e.g. a LIMIT page size), from piko.param's default: quality.
	DefaultLimit *int

	// MaxLimit holds the inclusive maximum enforced at call time for a numeric parameter
	// (e.g. a LIMIT cap), from piko.param's max: quality.
	MaxLimit *int

	// Name is the parameter name from piko.param directive, or inferred from the column
	// name.
	Name string

	// SortableColumns holds the allowed ORDER BY column names from a piko.sortable
	// directive's columns: option.
	SortableColumns []string

	// SQLType is the resolved SQL type.
	SQLType SQLType

	// Number is the positional parameter number.
	Number int

	// Nullable indicates whether the parameter accepts NULL values.
	Nullable bool

	// IsSlice indicates the parameter expands to multiple values (piko.param kind: slice).
	IsSlice bool

	// IsOptional indicates the parameter is for a dynamic WHERE clause (piko.param optional:
	// true).
	IsOptional bool

	// Kind identifies the directive kind that declared this parameter.
	Kind ParameterDirectiveKind

	// Context records the SQL clause the parameter was found in. LIMIT/OFFSET contexts drive
	// integer typing, the non-negative clamp, and the generated `int` field type.
	Context ParameterContext
}

// QueryDirectives holds all parsed piko. directives for a query.
type QueryDirectives struct {
	// NullableOverride forces nullability on or off for the entire query result.
	NullableOverride *bool

	// ReadOnlyOverride forces the query's read-only flag on or off, overriding the
	// automatically detected value from statement analysis.
	ReadOnlyOverride *bool

	// EmbedTables holds table names from inline piko.embed directives.
	EmbedTables []string

	// GroupByKeys holds column references from piko.group_by directives.
	GroupByKeys []string

	// Slices holds parameter numbers declared with piko.slice.
	Slices []int

	// DynamicOrderByColumns holds allowed ORDER BY columns from piko.sortable directives.
	DynamicOrderByColumns []string

	// ParamOverrides holds explicit type overrides from piko.param.
	ParamOverrides []ParamOverride

	// DynamicRuntime indicates a piko.dynamic: runtime directive was specified, causing
	// generation of a fluent runtime query builder.
	DynamicRuntime bool
}

// ParamOverride is an explicit parameter type override from a piko.param directive.
type ParamOverride struct {
	// Name is the parameter name.
	Name string

	// TypeName is the SQL type name.
	TypeName string

	// Nullable indicates whether the parameter is nullable.
	Nullable bool
}

// GenerationResult holds the output of the querier's code generation for a single named
// database connection.
type GenerationResult struct {
	// Files holds the generated source files.
	Files []GeneratedFile

	// Diagnostics holds any warnings or errors encountered during analysis.
	Diagnostics []SourceError
}

// GeneratedFile represents a single generated source file.
type GeneratedFile struct {
	// Name is the filename (e.g. "models.go", "users.sql.go").
	Name string

	// Content is the formatted source code.
	Content []byte
}

// SourceError is a diagnostic error or warning mapped back to the source SQL file with
// line and column information.
type SourceError struct {
	// Filename is the source SQL file path.
	Filename string

	// Message describes the error.
	Message string

	// Code is a stable error code for documentation and suppression (e.g. "Q001" for unknown
	// column).
	Code string

	// Suggestion is an optional fix suggestion.
	Suggestion string

	// Line is the one-based line number.
	Line int

	// Column is the one-based column number.
	Column int

	// EndLine is the one-based end line number, if the error spans a range.
	EndLine int

	// EndColumn is the one-based end column number.
	EndColumn int

	// Severity indicates whether this is an error, warning, or hint.
	Severity ErrorSeverity
}

// ErrorSeverity classifies diagnostic severity.
type ErrorSeverity uint8

const (
	// SeverityError indicates a fatal error that prevents code generation.
	SeverityError ErrorSeverity = iota

	// SeverityWarning indicates a potential problem that does not block generation.
	SeverityWarning

	// SeverityHint indicates a suggestion for improvement.
	SeverityHint
)

// IsPaginationBound reports whether the parameter binds a LIMIT or OFFSET clause, which
// makes it a non-negative integer with a generated `int` field and optional default/max
// clamping.
//
// Returns bool which is true when the parameter binds a LIMIT or OFFSET clause.
func (p QueryParameter) IsPaginationBound() bool {
	return p.Context == ParameterContextLimit || p.Context == ParameterContextOffset
}
