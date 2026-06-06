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

// TextSpan represents a range of text in a source file.
//
// All positions are 1-based. EndColumn is exclusive (one past the last character).
type TextSpan struct {
	// Line holds the 1-based starting line number.
	Line int

	// Column holds the 1-based starting column number.
	Column int

	// EndLine holds the 1-based ending line number.
	EndLine int

	// EndColumn holds the 1-based exclusive ending column number.
	EndColumn int
}

// DirectiveBlock is the root AST node for all directives in a query header. It holds the
// fully parsed representation of the -- piko.query(...) comment lines that precede a SQL
// statement.
type DirectiveBlock struct {
	// Name holds the parsed name keyword argument from -- piko.query(name: ...), or nil when
	// absent.
	Name *NameDirective

	// Command holds the parsed command keyword argument from -- piko.query(command: ...), or
	// nil when absent.
	Command *CommandDirective

	// Parameters holds the list of parsed parameter directives, each produced by a -- $N as
	// piko.X(...) header line.
	Parameters []*ParameterDirective

	// Metadata holds the list of parsed metadata directives produced from query-level
	// keyword arguments (group_by, dynamic, readonly, nullable).
	Metadata []*MetadataDirective

	// Embeds holds the parsed -- piko.embed(table, from: alias) header directives.
	Embeds []*EmbedDirective

	// ColumnOverrides holds the parsed -- piko.column(name, ...) header directives. Each
	// entry overrides the inferred SQL type, custom Go destination type, or nullability of
	// one output column.
	ColumnOverrides []*ColumnOverride

	// Span holds the source range covering all directives in this block.
	Span TextSpan
}

// ColumnOverride represents one -- piko.column(name, type:, go_type:, nullable:)
// directive. Populated by applyHeaderCall in the directive parser; consumed by the query
// analyser after output-column inference and by the catalogue builder when the directive
// is parsed from a migration file.
//
// Name is either an output column name (when the directive appears in a query header) or
// a qualified table.column name (when it appears in a migration file above a CREATE
// TABLE). The consumer distinguishes by the directive's position relative to surrounding
// SQL.
type ColumnOverride struct {
	// Nullable is non-nil when the `nullable:` keyword argument was declared.
	Nullable *bool

	// Name is the output column or table.column reference targeted by the override.
	Name string

	// SQLType carries the `type:` keyword argument value when set. When empty, no SQL-type
	// override was declared.
	SQLType string

	// GoType carries the `go_type:` keyword argument value when set.
	//
	// The string is "<import-path>.<TypeName>"; the emitter splits it on the last '.' into
	// Package and Name. When empty, no Go-type override was declared.
	GoType string

	// Span holds the source range of the entire piko.column(...) call.
	Span TextSpan

	// NameSpan holds the source range of the positional name token.
	NameSpan TextSpan

	// TypeSpan holds the source range of the `type:` keyword-argument value when present.
	TypeSpan TextSpan

	// GoTypeSpan holds the source range of the `go_type:` keyword- argument value when
	// present.
	GoTypeSpan TextSpan
}

// EmbedDirective represents a header-line -- piko.embed(table[, from: alias][, as:
// field]) declaration. The codegen groups every column whose source-table alias matches
// the From value into a nested Go struct named after As (or the table name when As is
// empty).
type EmbedDirective struct {
	// Table is the source table or view whose columns are embedded.
	Table string

	// From is the FROM-clause alias whose columns belong to this embed group. Empty when the
	// embed targets the table directly.
	From string

	// As overrides the Go field name; empty defaults to the table.
	As string

	// Span holds the source range of the entire embed directive line.
	Span TextSpan

	// TableSpan holds the source range of the positional table token.
	TableSpan TextSpan
}

// NameDirective represents a -- piko.name: Value directive.
type NameDirective struct {
	// Value holds the name string specified in the directive.
	Value string

	// Span holds the source range of the entire directive line.
	Span TextSpan

	// KeySpan holds the source range of the "piko.name" key portion.
	KeySpan TextSpan

	// ValueSpan holds the source range of the value portion after the colon.
	ValueSpan TextSpan
}

// CommandDirective represents a -- piko.command: Value directive.
type CommandDirective struct {
	// Value holds the raw command string specified in the directive.
	Value string

	// Span holds the source range of the entire directive line.
	Span TextSpan

	// KeySpan holds the source range of the "piko.command" key portion.
	KeySpan TextSpan

	// ValueSpan holds the source range of the value portion after the colon.
	ValueSpan TextSpan

	// Command holds the parsed query command enumeration value.
	Command QueryCommand
}

// ParameterDirectiveKind identifies the role of a parameter declared with the $N as
// piko.param(name) header syntax. Cardinality (slice) and presence (optional) are now
// qualities on a parameter (IsSlice/IsOptional), not distinct kinds, and limit/offset are
// inferred from clause position, so only the standard and sortable kinds remain.
type ParameterDirectiveKind uint8

const (
	// ParameterDirectiveParam defines a standard query parameter.
	ParameterDirectiveParam ParameterDirectiveKind = iota

	// ParameterDirectiveSortable defines a sortable input (dynamic ORDER BY) declared with
	// the standalone piko.sortable directive; it does not bind a placeholder.
	ParameterDirectiveSortable
)

// ParameterDirective represents a parameter directive such as `-- $1 as
// piko.param(page_size)` or `-- :email as piko.param`.
type ParameterDirective struct {
	// TypeHint holds the optional SQL type hint for this parameter, or nil if unspecified.
	TypeHint *string

	// Nullable holds the optional nullability override, or nil if unspecified.
	Nullable *bool

	// DefaultVal holds the optional default value applied when the caller omits a numeric
	// parameter (e.g. a LIMIT page size), or nil if unspecified.
	DefaultVal *int

	// MaxVal holds the optional inclusive maximum enforced at call time for a numeric
	// parameter (e.g. a LIMIT cap), or nil if unspecified.
	MaxVal *int

	// Name holds the declared parameter name.
	Name string

	// DirectiveName holds the raw directive kind string (e.g. "param", "limit").
	DirectiveName string

	// Columns holds the list of column names for sortable parameters.
	Columns []string

	// Number holds the positional parameter number (e.g. 1 for ?1).
	Number int

	// Span holds the source range of the entire parameter directive.
	Span TextSpan

	// NumberSpan holds the source range of the parameter number token.
	NumberSpan TextSpan

	// KindSpan holds the source range of the directive kind token (e.g. "piko.limit").
	KindSpan TextSpan

	// NameSpan holds the source range of the parameter name token.
	NameSpan TextSpan

	// Kind holds the parsed parameter directive kind enumeration value.
	Kind ParameterDirectiveKind

	// IsNamed indicates whether this parameter uses named (:name) rather than positional
	// (?N) syntax.
	IsNamed bool

	// IsOptional indicates the parameter may be omitted (the `optional: true` quality); the
	// predicate it appears in is dropped when the caller supplies nil.
	IsOptional bool

	// IsSlice indicates the parameter expands to multiple placeholders at call time (the
	// `kind: slice` quality).
	IsSlice bool
}

// MetadataDirective represents a -- piko.directive: value directive such as piko.group_by
// or piko.nullable.
type MetadataDirective struct {
	// Directive holds the directive name string (e.g. "group_by", "nullable").
	Directive string

	// Value holds the directive value string.
	Value string

	// Span holds the source range of the entire metadata directive line.
	Span TextSpan

	// KeySpan holds the source range of the directive key portion.
	KeySpan TextSpan

	// ValueSpan holds the source range of the directive value portion.
	ValueSpan TextSpan
}
