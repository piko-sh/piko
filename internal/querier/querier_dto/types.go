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

// Dialect is an opaque string that identifies a SQL engine dialect
// (e.g. "postgres", "mysql", "sqlite", "duckdb").
//
// Each engine adapter defines its own dialect name. The domain layer never
// switches on dialect values; all engine-specific behaviour is expressed
// through the EnginePort interface.
type Dialect = string

// ParameterStyle identifies how parameter placeholders are written.
type ParameterStyle uint8

const (
	// ParameterStyleDollar uses $1, $2, $3 (PostgreSQL).
	ParameterStyleDollar ParameterStyle = iota

	// ParameterStyleQuestion uses ? (MySQL, SQLite).
	ParameterStyleQuestion

	// ParameterStyleNamed uses @param_name (BigQuery).
	ParameterStyleNamed

	// ParameterStyleColon uses :1, :2 (Oracle).
	ParameterStyleColon

	// ParameterStyleAt uses @p1, @p2 (SQL Server).
	ParameterStyleAt
)

// DirectiveParameterPrefix describes a sigil that may introduce a parameter
// reference in a directive comment line (e.g. `-- ?1 as piko.limit(page_size)`
// or `-- :email as piko.param`).
//
// Each engine declares the set of valid prefixes via
// SupportedDirectivePrefixes.
type DirectiveParameterPrefix struct {
	// Prefix is the leading byte ('$', '?', ':', '@').
	Prefix byte

	// IsNamed indicates whether the prefix expects an
	// identifier rather than digits.
	IsNamed bool
}

// CommentStyle describes the comment syntax used by an engine's query files.
// The default for SQL dialects is "--" line comments.
type CommentStyle struct {
	// LinePrefix is the single-line comment prefix (e.g. "--", "#", "//").
	LinePrefix string
}

// SQLTypeCategory classifies SQL types into broad categories for structured
// type resolution, replacing stringly-typed switch statements.
type SQLTypeCategory uint8

const (
	// TypeCategoryInteger covers smallint, integer, bigint, and serial variants.
	TypeCategoryInteger SQLTypeCategory = iota

	// TypeCategoryFloat covers real and double precision.
	TypeCategoryFloat

	// TypeCategoryDecimal covers numeric and decimal with precision/scale.
	TypeCategoryDecimal

	// TypeCategoryBoolean covers boolean.
	TypeCategoryBoolean

	// TypeCategoryText covers char, varchar, text, and similar string types.
	TypeCategoryText

	// TypeCategoryBytea covers bytea, blob, and binary types.
	TypeCategoryBytea

	// TypeCategoryTemporal covers timestamp, timestamptz, date, time, interval.
	TypeCategoryTemporal

	// TypeCategoryJSON covers json and jsonb.
	TypeCategoryJSON

	// TypeCategoryUUID covers uuid.
	TypeCategoryUUID

	// TypeCategoryNetwork covers inet, cidr, macaddr.
	TypeCategoryNetwork

	// TypeCategoryGeometric covers point, line, polygon, and similar types.
	TypeCategoryGeometric

	// TypeCategoryEnum covers user-defined enum types.
	TypeCategoryEnum

	// TypeCategoryComposite covers user-defined composite types.
	TypeCategoryComposite

	// TypeCategoryArray covers array types wrapping another element type.
	TypeCategoryArray

	// TypeCategoryRange covers range types (PostgreSQL).
	TypeCategoryRange

	// TypeCategoryStruct covers DuckDB STRUCT types - named fields with typed values.
	TypeCategoryStruct

	// TypeCategoryMap covers DuckDB MAP types - key/value pairs.
	TypeCategoryMap

	// TypeCategoryUnion covers DuckDB UNION types - tagged variant types.
	TypeCategoryUnion

	// TypeCategoryUnknown is the fallback for unrecognised types.
	TypeCategoryUnknown
)

// SQLType is a structured representation of a SQL type, carrying category,
// engine-specific name, and optional parameters. This replaces the
// stringly-typed approach where type resolution switches on raw name strings.
type SQLType struct {
	// Precision is the numeric precision for decimal types, or nil if
	// unspecified.
	Precision *int

	// Scale is the numeric scale for decimal types, or nil if unspecified.
	Scale *int

	// Length is the character length for text types, or nil if unspecified.
	Length *int

	// ElementType is the element type for array and range types.
	ElementType *SQLType

	// EngineName is the engine-native type name (e.g. "numeric", "varchar",
	// "int8"), used for display and engine-specific code paths.
	EngineName string

	// Schema is the schema for user-defined types (enums, composites).
	// Empty for built-in types.
	Schema string

	// EnumValues holds the valid values for enum types.
	EnumValues []string

	// StructFields holds the named fields for STRUCT types (DuckDB).
	StructFields []StructField

	// KeyType is the key type for MAP types (DuckDB). The value type is stored
	// in ElementType.
	KeyType *SQLType

	// UnionMembers holds the tagged members for UNION types (DuckDB).
	UnionMembers []UnionMember

	// Category classifies the type for structured resolution.
	Category SQLTypeCategory
}

// StructField describes a single named field within a STRUCT type.
type StructField struct {
	// Name is the field name.
	Name string

	// SQLType is the field's type.
	SQLType SQLType
}

// UnionMember describes a single tagged member within a UNION type.
type UnionMember struct {
	// Tag is the member's tag name.
	Tag string

	// SQLType is the member's type.
	SQLType SQLType
}

// QueryCommand specifies the execution pattern for a query.
type QueryCommand uint8

const (
	// QueryCommandOne returns a single row (error if zero or more than one).
	QueryCommandOne QueryCommand = iota

	// QueryCommandMany returns a slice of rows.
	QueryCommandMany

	// QueryCommandExec executes without returning rows.
	QueryCommandExec

	// QueryCommandExecResult returns sql.Result.
	QueryCommandExecResult

	// QueryCommandExecRows returns the affected row count.
	QueryCommandExecRows

	// QueryCommandBatch executes as a batched operation (pgx).
	QueryCommandBatch

	// QueryCommandStream returns an iterator over rows.
	QueryCommandStream

	// QueryCommandCopyFrom uses bulk insert (PostgreSQL COPY).
	QueryCommandCopyFrom
)

// QueryCapabilities is a bitmask indicating which pgx-specific features are
// used by the query set, controlling conditional generation of PgxDBTX.
type QueryCapabilities uint8

const (
	// CapabilityBatch indicates at least one :batch query exists.
	CapabilityBatch QueryCapabilities = 1 << iota

	// CapabilityCopyFrom indicates at least one :copyfrom query exists.
	CapabilityCopyFrom
)

// DatabaseConfig holds the configuration for a single named database
// connection during code generation.
type DatabaseConfig struct {
	// MigrationDirectory is the path to the directory containing ordered
	// migration SQL files.
	MigrationDirectory string

	// QueryDirectory is the path to the directory containing query SQL files.
	QueryDirectory string

	// TypeOverrides provides custom SQL-to-Go type mappings that supplement
	// the framework defaults.
	TypeOverrides []TypeOverride

	// CustomFunctions declares additional function signatures from SQLite
	// extensions that the querier should recognise during query analysis.
	CustomFunctions []CustomFunctionConfig
}

// TypeOverride maps a specific SQL type to a custom Go type, overriding the
// framework default.
type TypeOverride struct {
	// SQLTypeName is the SQL type to override (e.g. "public.custom_type").
	SQLTypeName string

	// GoPackage is the Go import path for the target type.
	GoPackage string

	// GoName is the Go type name (e.g. "CustomType").
	GoName string
}

// CustomFunctionConfig describes a user-declared function signature for a
// SQLite extension that the querier does not have built-in knowledge of.
type CustomFunctionConfig struct {
	// Name is the SQL function name.
	Name string

	// ReturnType is the SQL type name of the return value.
	ReturnType string

	// Nullable specifies the null-handling behaviour. Valid values are
	// "null_on_null" (default), "never_null", and "called_on_null".
	Nullable string

	// Arguments lists the SQL type names for each parameter (e.g. "integer",
	// "real", "text", "blob", "any").
	Arguments []string

	// MinArguments is the minimum argument count when IsVariadic is true
	// or when arguments have defaults. Defaults to len(Arguments).
	MinArguments int

	// IsAggregate indicates whether the function is an aggregate.
	IsAggregate bool

	// IsVariadic indicates the last argument can repeat.
	IsVariadic bool
}

// ParsedStatementRaw is a marker interface for engine-native AST nodes stored
// in ParsedStatement.Raw.
//
// Each engine adapter defines its own concrete type implementing this
// interface. The domain never inspects the contents; it passes statements
// back to the same engine for DDL application or query analysis.
type ParsedStatementRaw interface {
	// IsParsedStatement is a marker method that distinguishes parsed statement
	// types from other interfaces.
	IsParsedStatement()
}

// ParsedStatement is an opaque wrapper around an engine-native AST node.
// The domain never inspects the contents - it passes statements back to the
// engine adapter for DDL application or query analysis.
type ParsedStatement struct {
	// Raw holds the engine-native AST node, whose concrete type depends on the
	// engine adapter (e.g. *parsedStatement for SQLite).
	Raw ParsedStatementRaw

	// Location is the byte offset in the source SQL.
	Location int

	// Length is the byte length of the statement in the source SQL.
	Length int
}

// DefaultSQLCommentStyle returns the standard SQL comment style using "--".
//
// Returns CommentStyle which uses "--" as the line
// prefix.
func DefaultSQLCommentStyle() CommentStyle {
	return CommentStyle{LinePrefix: "--"}
}
