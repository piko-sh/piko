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

// DirectiveBlock is the root AST node for all directives in a query
// header. It holds the fully parsed representation of the -- piko.*
// comment lines that precede a SQL statement.
type DirectiveBlock struct {
	// Name holds the parsed piko.name directive, or nil if absent.
	Name *NameDirective

	// Command holds the parsed piko.command directive, or nil if absent.
	Command *CommandDirective

	// Parameters holds the list of parsed parameter directives.
	Parameters []*ParameterDirective

	// Metadata holds the list of parsed metadata directives.
	Metadata []*MetadataDirective

	// Span holds the source range covering all directives in this block.
	Span TextSpan
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

// ParameterDirectiveKind identifies the role of a parameter declared
// with the $N as piko.X(name) header syntax.
type ParameterDirectiveKind uint8

const (
	// ParameterDirectiveParam defines a standard query parameter.
	ParameterDirectiveParam ParameterDirectiveKind = iota

	// ParameterDirectiveOptional defines an optional query parameter.
	ParameterDirectiveOptional

	// ParameterDirectiveSlice defines a parameter that accepts a slice of values.
	ParameterDirectiveSlice

	// ParameterDirectiveSortable defines a parameter that controls sort order.
	ParameterDirectiveSortable

	// ParameterDirectiveLimit defines a parameter that specifies a row limit.
	ParameterDirectiveLimit

	// ParameterDirectiveOffset defines a parameter that specifies a row offset.
	ParameterDirectiveOffset
)

// ParameterDirective represents a parameter directive such as
// `-- ?1 as piko.limit(page_size)` or `-- :email as piko.param`.
type ParameterDirective struct {
	// TypeHint holds the optional SQL type hint for this parameter, or nil if unspecified.
	TypeHint *string

	// Nullable holds the optional nullability override, or nil if unspecified.
	Nullable *bool

	// DefaultVal holds the optional default value for
	// limit/offset parameters, or nil if unspecified.
	DefaultVal *int

	// MaxVal holds the optional maximum value for
	// limit/offset parameters, or nil if unspecified.
	MaxVal *int

	// Name holds the declared parameter name.
	Name string

	// DirectiveName holds the raw directive kind string (e.g. "param", "limit").
	DirectiveName string

	// Columns holds the list of column names for sortable parameters.
	Columns []string

	// Options holds the list of key:value options attached to this parameter.
	Options []*DirectiveOption

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

	// IsNamed indicates whether this parameter uses named
	// (:name) rather than positional (?N) syntax.
	IsNamed bool
}

// DirectiveOption represents a key:value option in a parameter directive.
type DirectiveOption struct {
	// Key holds the option key string.
	Key string

	// Value holds the option value string.
	Value string

	// Span holds the source range of the entire key:value option.
	Span TextSpan

	// KeySpan holds the source range of the option key.
	KeySpan TextSpan

	// ValueSpan holds the source range of the option value.
	ValueSpan TextSpan
}

// MetadataDirective represents a -- piko.directive: value directive
// such as piko.group_by or piko.nullable.
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
