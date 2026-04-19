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

// DirectiveRole identifies where a directive appears in a SQL file and what shape its
// grammar takes.
type DirectiveRole uint8

const (
	// DirectiveRoleTop is the top-level query header `-- piko.query(...)`.
	DirectiveRoleTop DirectiveRole = iota

	// DirectiveRoleParam is a parameter binding line such as `-- $1 as piko.optional(name)`.
	DirectiveRoleParam

	// DirectiveRoleHeader is a header-level directive such as `-- piko.embed(table, from:
	// o)` that is not a parameter binding.
	DirectiveRoleHeader

	// DirectiveRoleMigration is a directive recognised inside .up.sql migration files
	// preceding CREATE FUNCTION statements.
	DirectiveRoleMigration
)

// KeywordArgumentKind classifies the value shape a keyword argument accepts.
type KeywordArgumentKind uint8

const (
	// KeywordArgumentString accepts any bare word or quoted string literal.
	KeywordArgumentString KeywordArgumentKind = iota

	// KeywordArgumentBool accepts true or false.
	KeywordArgumentBool

	// KeywordArgumentInt accepts an integer literal.
	KeywordArgumentInt

	// KeywordArgumentIdent accepts a bare identifier; when AllowedValues is set the
	// identifier must match one of the listed values (case-insensitive).
	KeywordArgumentIdent

	// KeywordArgumentList accepts a bracketed list of bare identifiers (e.g. columns: [name,
	// price, created_at]).
	KeywordArgumentList

	// KeywordArgumentQualifiedIdent accepts a dotted identifier such as `orders.id`.
	KeywordArgumentQualifiedIdent
)

// PositionalSpec describes one positional argument a directive accepts. Each positional
// has a name and may be passed positionally or by name; the parser routes a keyword
// argument whose key matches a positional name into the corresponding slot.
type PositionalSpec struct {
	// Name is the human-readable name shown in completions, errors, and accepted as a
	// keyword argument key for keyword argument-as-positional usage.
	Name string

	// Description is the one-line summary shown in LSP hover.
	Description string

	// AllowedValues, when non-empty, restricts KeywordArgumentIdent values to a closed set
	// checked case-insensitively. Used by the validator and surfaced to the LSP as a
	// completion enum.
	AllowedValues []string

	// Required marks whether the positional argument must be present.
	Required bool

	// Kind classifies the positional's value shape, used by the validator to reject inputs
	// of the wrong shape.
	Kind KeywordArgumentKind
}

// KeywordArgumentSpec describes one accepted keyword argument on a directive.
type KeywordArgumentSpec struct {
	// Name is the keyword argument key as it appears on the directive line.
	Name string

	// Description is the one-line summary shown in LSP hover.
	Description string

	// AllowedValues, when non-empty, restricts KeywordArgumentIdent values to a closed set
	// checked case-insensitively. Used by the validator and surfaced to the LSP as a
	// completion enum.
	AllowedValues []string

	// Kind classifies the value shape.
	Kind KeywordArgumentKind

	// Required marks the keyword argument as mandatory.
	Required bool
}

// DirectiveSpec is the single source of truth for everything the parser, validator, LSP
// completion, LSP hover, and the generated reference documentation know about one
// directive shape.
type DirectiveSpec struct {
	// Name is the fully qualified directive name (e.g. "piko", "piko.embed",
	// "piko.optional").
	Name string

	// Summary is the one-line description shown in LSP hover.
	Summary string

	// Example is a complete one-line invocation shown in LSP hover.
	Example string

	// DocsURL is a stable link to documentation for this directive.
	DocsURL string

	// Positionals lists the positional arguments in declaration order.
	//
	// Each may also be passed by name. Nil or empty means the directive accepts no
	// positionals.
	Positionals []PositionalSpec

	// KeywordArguments lists the keyword argument-only arguments in display order. These
	// cannot be passed positionally.
	KeywordArguments []KeywordArgumentSpec

	// Role categorises where the directive may appear.
	Role DirectiveRole

	// ParamKind is set for Role==DirectiveRoleParam directives and maps to the existing
	// ParameterDirectiveKind enum so the AST shape downstream consumers depend on does not
	// change.
	ParamKind ParameterDirectiveKind
}

var (
	// commandEnumValues holds the literals accepted by the top-level `command:` keyword
	// argument.
	//
	// They are kept in declaration order so completion lists render predictably. The
	// asyncexec command marks fire-and-forget mutations whose completion the server reports
	// asynchronously (for example ClickHouse ALTER UPDATE and DELETE).
	commandEnumValues = []string{
		"one", "many", "exec", "execresult", "execrows",
		"batch", "stream", "copyfrom", "asyncexec",
	}

	// dynamicEnumValues are the accepted values for the `dynamic:` keyword argument.
	dynamicEnumValues = []string{"runtime"}

	// boolEnumValues are the accepted values for KeywordArgumentBool keyword arguments.
	boolEnumValues = []string{"true", "false"}

	// kindEnumValues are the accepted values for the piko.param `kind:` keyword argument
	// (the parameter's cardinality). The default when omitted is scalar.
	kindEnumValues = []string{"scalar", "slice"}

	// DirectiveSpecs is the registry of every directive the parser recognises. Lookups go
	// through LookupDirective; the slice form is preserved so completion lists render in
	// declaration order.
	DirectiveSpecs = []DirectiveSpec{
		{
			Name:    "piko.query",
			Role:    DirectiveRoleTop,
			Summary: "Top-level query declaration. Required on every query header.",
			Example: "-- piko.query(GetUser, one)",
			DocsURL: "https://docs.piko.sh/reference/querier#piko-query",
			Positionals: []PositionalSpec{
				{Name: "name", Required: true, Kind: KeywordArgumentIdent, Description: "Go method name for the generated query."},
				{
					Name:          "command",
					Required:      true,
					Kind:          KeywordArgumentIdent,
					AllowedValues: commandEnumValues,
					Description:   "Execution pattern (one/many/exec/execresult/execrows/batch/stream/copyfrom/asyncexec).",
				},
			},
			KeywordArguments: []KeywordArgumentSpec{
				{Name: "dynamic", Kind: KeywordArgumentIdent, AllowedValues: dynamicEnumValues, Description: "Emit a fluent runtime query builder instead of a static method."},
				{Name: "readonly", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "Override automatic read-only detection."},
				{Name: "nullable", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "Override automatic nullability propagation across the output columns."},
				{Name: "optional", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "For command:one, return (row, false, nil) instead of the no-rows sentinel."},
				{Name: "group_by", Kind: KeywordArgumentQualifiedIdent, Description: "Declare the grouping column for one-to-many embed joins."},
			},
		},
		{
			Name:      "piko.param",
			Role:      DirectiveRoleParam,
			ParamKind: ParameterDirectiveParam,
			Summary:   "Bind a positional or named placeholder to a Go parameter.",
			Example:   "-- $1 as piko.param(user_id, type: int8, nullable: false)",
			DocsURL:   "https://docs.piko.sh/reference/querier#piko-param",
			Positionals: []PositionalSpec{
				{Name: "name", Required: true, Kind: KeywordArgumentIdent, Description: "The Go identifier for the parameter."},
			},
			KeywordArguments: []KeywordArgumentSpec{
				{Name: "type", Kind: KeywordArgumentIdent, Description: "Explicit SQL type that overrides inference."},
				{Name: "nullable", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "Override the inferred nullability for this parameter."},
				{Name: "default", Kind: KeywordArgumentInt, Description: "Default integer for an omitted numeric (pagination) parameter, e.g. a LIMIT default. Excludes optional."},
				{Name: "optional", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "Drop the predicate this parameter appears in when the caller passes nil; excludes default."},
				{Name: "kind", Kind: KeywordArgumentIdent, AllowedValues: kindEnumValues, Description: "Cardinality: scalar (default) or slice (expands to IN (...) at call time)."},
				{Name: "max", Kind: KeywordArgumentInt, Description: "Inclusive upper bound enforced at call time for a numeric parameter (e.g. a LIMIT cap)."},
			},
		},
		{
			Name:      "piko.sortable",
			Role:      DirectiveRoleHeader,
			ParamKind: ParameterDirectiveSortable,
			Summary:   "Validated dynamic ORDER BY with a closed column allow-list. Standalone: it does not bind a placeholder, so it is not written with the $N as prefix.",
			Example:   "-- piko.sortable(order_by, [name, total, placed_at])",
			DocsURL:   "https://docs.piko.sh/reference/querier#piko-sortable",
			Positionals: []PositionalSpec{
				{Name: "name", Required: true, Kind: KeywordArgumentIdent, Description: "The Go identifier for the sortable input."},
				{Name: "columns", Required: true, Kind: KeywordArgumentList, Description: "Closed list of column names the caller may sort by."},
			},
		},
		{
			Name:    "piko.embed",
			Role:    DirectiveRoleHeader,
			Summary: "Group projection columns into a nested Go struct.",
			Example: "-- piko.embed(orders, from: o)",
			DocsURL: "https://docs.piko.sh/reference/querier#piko-embed",
			Positionals: []PositionalSpec{
				{Name: "table", Required: true, Kind: KeywordArgumentIdent, Description: "Source table or view whose columns are embedded."},
			},
			KeywordArguments: []KeywordArgumentSpec{
				{Name: "from", Kind: KeywordArgumentIdent, Description: "FROM-clause alias whose columns belong to this embed group."},
				{Name: "as", Kind: KeywordArgumentIdent, Description: "Override the Go field name for the embed."},
			},
		},
		{
			Name: "piko.column",
			Role: DirectiveRoleHeader,
			Summary: "Override the inferred SQL type, custom Go destination type, or nullability of one " +
				"column. Valid above a SELECT (where the positional is an output column name) or above a " +
				"CREATE TABLE in a migration file (where the positional is a qualified table.column name).",
			Example: "-- piko.column(email_lower, type: text)",
			DocsURL: "https://docs.piko.sh/reference/querier#piko-column",
			Positionals: []PositionalSpec{
				{
					Name:        "name",
					Required:    true,
					Kind:        KeywordArgumentQualifiedIdent,
					Description: "Output column name (in a query header) or qualified table.column name (in a migration file).",
				},
			},
			KeywordArguments: []KeywordArgumentSpec{
				{
					Name:        "type",
					Kind:        KeywordArgumentIdent,
					Description: "SQL type that overrides the analyser's inference. Mapped to Go through the engine's type registry. Mutually exclusive with go_type.",
				},
				{
					Name: "go_type",
					Kind: KeywordArgumentString,
					Description: "Explicit Go destination type with import path (e.g. github.com/google/uuid.UUID). " +
						"The package is imported automatically. Mutually exclusive with type.",
				},
				{
					Name:          "nullable",
					Kind:          KeywordArgumentBool,
					AllowedValues: boolEnumValues,
					Description:   "Override the inferred nullability of this column.",
				},
			},
		},
		{
			Name: "piko.migration",
			Role: DirectiveRoleMigration,
			Summary: "Per-block migration directive. Placed above the statement it configures: readonly: " +
				"above a CREATE FUNCTION marks the function read-only in the catalogue; no_transaction: " +
				"above any statement runs the migration outside a transaction.",
			Example: "-- piko.migration(readonly: true)",
			DocsURL: "https://docs.piko.sh/reference/querier#piko-migration",
			KeywordArguments: []KeywordArgumentSpec{
				{Name: "readonly", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "Mark the following CREATE FUNCTION read-only (true) or modifying (false) in the catalogue."},
				{Name: "no_transaction", Kind: KeywordArgumentBool, AllowedValues: boolEnumValues, Description: "Run the migration outside a transaction (e.g. CREATE INDEX CONCURRENTLY)."},
			},
		},
	}

	// directiveSpecsByName indexes DirectiveSpecs by name.
	//
	// It gives O(1) lookup, and LookupDirective resolves through it. It is built once at
	// package initialisation; Go's variable-initialisation order guarantees DirectiveSpecs
	// is populated first because this initialiser depends on it. Each value is a pointer
	// into the DirectiveSpecs backing array so callers observe the same spec instance the
	// slice form exposes.
	directiveSpecsByName = buildDirectiveSpecsByName()
)

// buildDirectiveSpecsByName indexes the DirectiveSpecs registry by directive name.
//
// Returns map[string]*DirectiveSpec which maps each directive's Name to its spec pointer.
func buildDirectiveSpecsByName() map[string]*DirectiveSpec {
	index := make(map[string]*DirectiveSpec, len(DirectiveSpecs))
	for position := range DirectiveSpecs {
		index[DirectiveSpecs[position].Name] = &DirectiveSpecs[position]
	}
	return index
}

// LookupDirective resolves a directive name (for example "piko" or "piko.embed") to its
// spec.
//
// Lookups are case-sensitive because the grammar only accepts the canonical lower-case
// form. Resolution is an O(1) map lookup over the directiveSpecsByName index.
//
// Takes name (string) which is the fully qualified directive name to resolve.
//
// Returns *DirectiveSpec which is the matching spec, or nil when none matches.
// Returns bool which is true when a spec was found.
func LookupDirective(name string) (*DirectiveSpec, bool) {
	spec, found := directiveSpecsByName[name]
	return spec, found
}

// LookupKeywordArgument resolves a keyword argument key against a directive spec.
//
// Takes spec (*DirectiveSpec) which holds the keyword arguments to search.
// Takes key (string) which is the keyword argument key to resolve.
//
// Returns *KeywordArgumentSpec which is the matching keyword argument, or nil when none
// matches.
// Returns bool which is true when a keyword argument was found.
func LookupKeywordArgument(spec *DirectiveSpec, key string) (*KeywordArgumentSpec, bool) {
	if spec == nil {
		return nil, false
	}
	for index := range spec.KeywordArguments {
		if spec.KeywordArguments[index].Name == key {
			return &spec.KeywordArguments[index], true
		}
	}
	return nil, false
}

// LookupPositional resolves a positional by name.
//
// It is used when a keyword argument key matches one of the directive's positional slots
// (keyword argument-as-positional).
//
// Takes spec (*DirectiveSpec) which holds the positionals to search.
// Takes name (string) which is the positional name to resolve.
//
// Returns *PositionalSpec which is the matching positional, or nil when none matches.
// Returns int which is the positional's index in the Positionals slice.
// Returns bool which is true when a positional was found.
func LookupPositional(spec *DirectiveSpec, name string) (*PositionalSpec, int, bool) {
	if spec == nil {
		return nil, 0, false
	}
	for index := range spec.Positionals {
		if spec.Positionals[index].Name == name {
			return &spec.Positionals[index], index, true
		}
	}
	return nil, 0, false
}

// PositionalNames returns the names of each positional in declaration order.
//
// It is used by the validator's "did you mean" suggestion helper for keyword
// argument-as-positional name typos.
//
// Takes spec (*DirectiveSpec) which holds the positionals to name.
//
// Returns []string which are the positional names in declaration order, or nil when spec
// is nil.
func PositionalNames(spec *DirectiveSpec) []string {
	if spec == nil {
		return nil
	}
	names := make([]string, len(spec.Positionals))
	for index := range spec.Positionals {
		names[index] = spec.Positionals[index].Name
	}
	return names
}

// DirectiveNames returns the canonical names of every directive in the registry.
//
// It is used by the validator's "did you mean" suggestion helper.
//
// Returns []string which are the canonical directive names in declaration order.
func DirectiveNames() []string {
	names := make([]string, len(DirectiveSpecs))
	for index := range DirectiveSpecs {
		names[index] = DirectiveSpecs[index].Name
	}
	return names
}

// KeywordArgumentNames returns the keyword argument keys accepted by the directive spec.
//
// It is used by the validator's "did you mean" suggestion helper.
//
// Takes spec (*DirectiveSpec) which holds the keyword arguments to name.
//
// Returns []string which are the keyword argument keys in display order, or nil when spec
// is nil.
func KeywordArgumentNames(spec *DirectiveSpec) []string {
	if spec == nil {
		return nil
	}
	names := make([]string, len(spec.KeywordArguments))
	for index := range spec.KeywordArguments {
		names[index] = spec.KeywordArguments[index].Name
	}
	return names
}
