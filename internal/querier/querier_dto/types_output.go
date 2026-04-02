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

// TypeMappingTable holds the complete set of SQL-to-Go type mappings,
// combining framework defaults with user overrides.
type TypeMappingTable struct {
	// Mappings holds the ordered list of type mappings. Later entries override
	// earlier ones for the same SQL type.
	Mappings []TypeMapping
}

// TypeMapping maps a SQL type to Go types for both nullable and non-nullable
// contexts.
type TypeMapping struct {
	// NotNull is the Go type to use when the column is NOT NULL.
	NotNull GoType

	// Nullable is the Go type to use when the column is nullable.
	Nullable GoType

	// SQLName is an optional engine-specific name for finer-grained matching
	// (e.g. "numeric" vs "decimal"), where empty matches any name in the category.
	SQLName string

	// SQLCategory is the type category this mapping applies to.
	SQLCategory SQLTypeCategory
}

// GoType identifies a Go type by its import path and name.
type GoType struct {
	// Package is the Go import path (e.g. "piko.sh/piko/wdk/maths"), or empty
	// for built-in types (string, int64, etc.).
	Package string

	// Name is the Go type name (e.g. "Decimal", "string", "*int64").
	Name string
}

// AnalysedQuery is the fully resolved result of query analysis, with all
// types and nullability propagated.
type AnalysedQuery struct {
	// InsertTable is the target table for :copyfrom queries.
	InsertTable string

	// Filename is the source SQL file path.
	Filename string

	// SQL is the original SQL text.
	SQL string

	// Name is the query name from the piko.name directive.
	Name string

	// Parameters holds the fully typed query parameters.
	Parameters []QueryParameter

	// OutputColumns holds the fully typed output columns.
	OutputColumns []OutputColumn

	// GroupByKey holds the column(s) used for one-to-many grouping, if any.
	GroupByKey []string

	// InsertColumns holds the target column names for :copyfrom queries.
	InsertColumns []string

	// AllowedColumns holds the columns that can be used in runtime WHERE
	// and ORDER BY clauses.
	AllowedColumns []AllowedColumn

	// Directives holds all parsed piko. directives for the query.
	Directives QueryDirectives

	// Line is the line number of the query in the source file.
	Line int

	// IsDynamic indicates the query has optional WHERE or ORDER BY clauses.
	IsDynamic bool

	// Command is the execution pattern from the piko.command directive.
	Command QueryCommand

	// DynamicRuntime indicates this query uses piko.dynamic: runtime to
	// generate a fluent runtime builder instead of a standard method.
	DynamicRuntime bool

	// ReadOnly indicates the query does not modify data. Downstream consumers
	// can use this to route read-only queries to read replicas.
	ReadOnly bool
}

// AllowedColumn represents a column available for runtime query building.
type AllowedColumn struct {
	// Name is the column name.
	Name string

	// SQLType is the column's resolved SQL type.
	SQLType SQLType
}

// OutputColumn is a fully typed output column.
type OutputColumn struct {
	// Name is the column name or alias.
	Name string

	// SourceTable is the source table, if this is a direct column reference.
	SourceTable string

	// SourceColumn is the source column name, if this is a direct reference.
	SourceColumn string

	// EmbedTable is the table name for embedded columns.
	EmbedTable string

	// SQLType is the resolved SQL type.
	SQLType SQLType

	// Nullable indicates whether the column can be NULL, accounting for JOIN
	// nullability, expression nullability, and aggregate behaviour.
	Nullable bool

	// IsEmbedded indicates this column is part of a piko.embed group.
	IsEmbedded bool

	// EmbedIsOuter indicates the embedded table was introduced via LEFT, RIGHT,
	// or FULL JOIN. When true, the emitter generates a pointer type for the
	// embedded struct (nil when no matching row exists).
	EmbedIsOuter bool
}

// QueryParameter is a fully typed query parameter.
type QueryParameter struct {
	// DefaultLimit holds the default value for a piko.limit parameter.
	DefaultLimit *int

	// MaxLimit holds the maximum allowed value for a piko.limit parameter.
	MaxLimit *int

	// Name is the parameter name from piko.param directive, or inferred from
	// the column name.
	Name string

	// SortableColumns holds the allowed ORDER BY column names from a
	// piko.sortable directive's columns: option.
	SortableColumns []string

	// SQLType is the resolved SQL type.
	SQLType SQLType

	// Number is the positional parameter number.
	Number int

	// Nullable indicates whether the parameter accepts NULL values.
	Nullable bool

	// IsSlice indicates the parameter expands to multiple values (piko.slice).
	IsSlice bool

	// IsOptional indicates the parameter is for a dynamic WHERE clause
	// (piko.optional).
	IsOptional bool

	// Kind identifies the directive kind that declared this parameter.
	Kind ParameterDirectiveKind
}

// QueryDirectives holds all parsed piko. directives for a query.
type QueryDirectives struct {
	// NullableOverride forces nullability on or off for the entire query result.
	NullableOverride *bool

	// ReadOnlyOverride forces the query's read-only flag on or off, overriding
	// the automatically detected value from statement analysis.
	ReadOnlyOverride *bool

	// EmbedTables holds table names from inline piko.embed directives.
	EmbedTables []string

	// GroupByKeys holds column references from piko.group_by directives.
	GroupByKeys []string

	// Slices holds parameter numbers declared with piko.slice.
	Slices []int

	// DynamicOrderByColumns holds allowed ORDER BY columns from
	// piko.sortable directives.
	DynamicOrderByColumns []string

	// ParamOverrides holds explicit type overrides from piko.param.
	ParamOverrides []ParamOverride

	// DynamicRuntime indicates a piko.dynamic: runtime directive was specified,
	// causing generation of a fluent runtime query builder.
	DynamicRuntime bool
}

// ParamOverride is an explicit parameter type override from a piko.param
// directive.
type ParamOverride struct {
	// Name is the parameter name.
	Name string

	// TypeName is the SQL type name.
	TypeName string

	// Nullable indicates whether the parameter is nullable.
	Nullable bool
}

// GenerationResult holds the output of the querier's code generation for a
// single named database connection.
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

// SourceError is a diagnostic error or warning mapped back to the source SQL
// file with line and column information.
type SourceError struct {
	// Filename is the source SQL file path.
	Filename string

	// Message describes the error.
	Message string

	// Code is a stable error code for documentation and suppression
	// (e.g. "Q001" for unknown column).
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

	// SeverityWarning indicates a potential problem that does not block
	// generation.
	SeverityWarning

	// SeverityHint indicates a suggestion for improvement.
	SeverityHint
)
