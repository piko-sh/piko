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

// DerivedTableSource identifies the origin of a derived (virtual) table in the FROM
// clause.
type DerivedTableSource uint8

const (
	// DerivedSourceUnnest is a table produced by UNNEST(array_column).
	DerivedSourceUnnest DerivedTableSource = iota

	// DerivedSourceFlatten is a table produced by FLATTEN (BigQuery, Snowflake).
	DerivedSourceFlatten

	// DerivedSourceTableFunction is a table produced by a table-valued function.
	DerivedSourceTableFunction

	// DerivedSourceSubquery is a table produced by a subquery in the FROM clause.
	DerivedSourceSubquery
)

// DerivedTableReference describes a virtual table in the FROM clause that does not exist
// in the catalogue. Engine adapters emit these for UNNEST, FLATTEN, table-valued
// functions, and subqueries in FROM position.
type DerivedTableReference struct {
	// Alias is the table alias used in the query.
	Alias string

	// Columns holds the resolved column types of the derived table.
	Columns []ScopedColumn

	// Source identifies how the derived table was produced.
	Source DerivedTableSource

	// JoinKind is the join type that introduced this derived table.
	JoinKind JoinKind
}

// RawQueryAnalysis is the engine adapter's initial analysis of a query before the domain
// applies type resolution and nullability propagation.
type RawQueryAnalysis struct {
	// EngineSpecific carries free-form metadata an engine attaches to the analysis.
	//
	// ClickHouse uses keys such as LIMIT_FILL_FROM / LIMIT_FILL_TO / LIMIT_FILL_STEP to
	// record `LIMIT ... WITH FILL` boundary expressions. The map is allocated lazily by the
	// engine adapter; consumers must nil-check before reading.
	EngineSpecific map[string]string

	// InsertTable is the target table name for INSERT statements.
	InsertTable string

	// InsertSelect holds the analysis of an INSERT ... SELECT statement's SELECT body.
	//
	// It is populated when the inserted rows come from a query rather than a VALUES list.
	// The body carries its own FROM/JOIN relations and parameters; the analyser resolves
	// those against the body's own scope (not the INSERT target's), so a parameter or column
	// referencing the SELECT source resolves correctly. Nil for INSERT ... VALUES and
	// non-INSERT statements.
	InsertSelect *RawQueryAnalysis

	// CompoundBranches holds the branches of a compound query (UNION, UNION ALL, INTERSECT,
	// EXCEPT).
	CompoundBranches []RawCompoundBranch

	// RawTableValuedFunctions holds unresolved table-valued function calls in FROM clauses.
	// The domain layer resolves these into DerivedTableReference entries.
	RawTableValuedFunctions []RawTableValuedFunctionReference

	// CTEDefinitions holds any WITH clause CTE definitions.
	CTEDefinitions []RawCTEDefinition

	// DerivedTables holds virtual tables from UNNEST, FLATTEN, table-valued functions, or
	// subqueries in the FROM clause.
	DerivedTables []DerivedTableReference

	// OutputColumns holds the unresolved output column references.
	OutputColumns []RawOutputColumn

	// GroupByColumns holds the columns referenced in a GROUP BY clause.
	GroupByColumns []ColumnReference

	// FromTables holds the tables referenced in the FROM clause.
	FromTables []TableReference

	// JoinClauses holds the JOIN clauses with their types.
	JoinClauses []JoinClause

	// ParameterReferences holds the unresolved parameter references.
	ParameterReferences []RawParameterReference

	// InsertColumns holds the target column names for INSERT statements.
	InsertColumns []string

	// ArrayJoinClauses holds ClickHouse-style `[LEFT] ARRAY JOIN` entries.
	//
	// Each clause unfolds an array source column into rows and exposes the element under the
	// supplied alias. The domain layer resolves the element type by looking up the source on
	// the FROM tables and registering the alias in the scope chain.
	ArrayJoinClauses []RawArrayJoinClause

	// OrderByColumns captures the structured ORDER BY column list.
	//
	// Each entry records the textual expression plus the optional ASC/DESC direction, NULLS
	// FIRST/LAST nulls placement, and WITH FILL modifiers a ClickHouse query may attach.
	// Engines that surface only the textual form leave the slice nil; consumers should treat
	// nil as "no metadata available" rather than "no ORDER BY".
	OrderByColumns []OrderByColumn

	// RawDerivedTables holds unresolved subqueries in FROM clauses. The domain layer
	// resolves these and converts them to DerivedTableReference entries.
	RawDerivedTables []RawDerivedTableReference

	// PredicateSubqueries holds unresolved subqueries that appear in a token-scanned
	// predicate position such as a WHERE, HAVING, or JOIN ... ON clause.
	//
	// An example is the scalar subquery in `WHERE x.id = (SELECT MAX(y.id) FROM y WHERE y.id
	// < ?2)`. Such subqueries are not reached through the SELECT-list expression tree or the
	// FROM-clause derived tables, so the engine records each one here. The domain layer
	// resolves their parameters and columns in a scope built from the subquery's own
	// FROM/JOIN tables and chained to the parent, so a subquery-local alias resolves locally
	// and a correlated reference resolves through the parent chain. Engines that leave this
	// nil fall back to the flat pass for the predicate subquery's parameters.
	PredicateSubqueries []*RawQueryAnalysis

	// HasReturning indicates whether the statement has a RETURNING clause.
	HasReturning bool

	// ReadOnly indicates the query does not modify data. SELECT and VALUES statements are
	// read-only unless they contain FOR UPDATE/SHARE locking clauses or data-modifying CTEs
	// (INSERT/UPDATE/DELETE inside WITH).
	ReadOnly bool

	// HasWhereClause indicates whether the top-level SELECT/UPDATE/DELETE already carries a
	// WHERE clause.
	//
	// The runtime query builder reads this to decide whether to prefix appended runtime
	// predicates with " WHERE " (the base query has none) or " AND " (the base query already
	// filters), avoiding a duplicate-WHERE syntax error when the base query is filtered (for
	// example by a tenant id) and the runtime caller adds further predicates.
	HasWhereClause bool
}

// OrderDirection identifies the direction of an ORDER BY column.
type OrderDirection uint8

const (
	// OrderDirectionUnspecified indicates the column carried no explicit ASC / DESC keyword.
	// The engine's default ordering applies.
	OrderDirectionUnspecified OrderDirection = iota

	// OrderDirectionAsc indicates the column carried the ASC keyword.
	OrderDirectionAsc

	// OrderDirectionDesc indicates the column carried the DESC keyword.
	OrderDirectionDesc
)

// OrderNullsPlacement identifies the placement of NULL values in an ORDER BY column.
type OrderNullsPlacement uint8

const (
	// OrderNullsUnspecified indicates the column carried no explicit NULLS FIRST / NULLS
	// LAST clause.
	OrderNullsUnspecified OrderNullsPlacement = iota

	// OrderNullsFirst indicates the column carried NULLS FIRST.
	OrderNullsFirst

	// OrderNullsLast indicates the column carried NULLS LAST.
	OrderNullsLast
)

// OrderByColumn captures one column of an ORDER BY clause with its direction, nulls
// placement, and optional WITH FILL modifiers.
//
// Engines populate this for queries where downstream consumers such as codegen and the
// dynamic-runtime path need the per-column metadata; the textual expression body is
// preserved verbatim so callers can either re-emit the SQL or inspect the structured
// fields.
type OrderByColumn struct {
	// Expression is the literal text of the column expression as it appears in the source
	// SQL, with surrounding whitespace stripped but inner whitespace preserved.
	Expression string

	// FillFrom is the textual `WITH FILL FROM <expr>` value when present. Empty when the
	// column either lacks WITH FILL or carries it without a FROM bound.
	FillFrom string

	// FillTo is the textual `WITH FILL TO <expr>` value when present.
	FillTo string

	// FillStep is the textual `WITH FILL STEP <expr>` value when present.
	FillStep string

	// Direction captures the ASC / DESC modifier when present.
	Direction OrderDirection

	// Nulls captures the NULLS FIRST / NULLS LAST modifier when present.
	Nulls OrderNullsPlacement

	// HasFill indicates whether the column carried a WITH FILL modifier. Engines set this to
	// true even when no FROM / TO / STEP body was supplied so callers can distinguish "no
	// fill" from "fill with engine-default boundaries".
	HasFill bool
}

// RawArrayJoinClause is an unresolved ClickHouse ARRAY JOIN entry.
//
// Each entry refers to a source whose runtime value is an array, and exposes the element
// via the supplied alias. When IsLeft is true (the `LEFT ARRAY JOIN` form) empty source
// arrays still produce a row with the element value defaulted to NULL, so the resolved
// alias gains a Nullable element type.
//
// ClickHouse accepts both bare column references and arbitrary array-producing
// expressions as the source. SourceColumn carries the bare-column form; SourceExpression
// carries the textual body for the expression form such as literal arrays or
// arrayMap(...) results. Consumers should prefer SourceExpression when non-empty and fall
// back to SourceColumn for the bare-column shape.
type RawArrayJoinClause struct {
	// Alias is the name the unfolded element is exposed under. Required: the projection
	// references the alias by name.
	Alias string

	// SourceColumn is the bare column name (without any table qualification) whose array
	// element the alias represents. Populated only for the bare-column form; empty when the
	// source is an arbitrary expression.
	SourceColumn string

	// SourceExpression captures the textual body of an expression source such as `[1, 2, 3]`
	// or `arrayMap(f, src)` for the non-bare-column form, and is empty when the source is a
	// bare column.
	SourceExpression string

	// IsLeft indicates the `LEFT ARRAY JOIN` variant; the element type is promoted to
	// nullable so empty arrays surface as NULL rows rather than collapsing.
	IsLeft bool
}

// CompoundOperator identifies a compound query operator.
type CompoundOperator uint8

const (
	// CompoundUnion is the UNION operator (removes duplicates).
	CompoundUnion CompoundOperator = iota + 1

	// CompoundUnionAll is the UNION ALL operator (keeps duplicates).
	CompoundUnionAll

	// CompoundIntersect is the INTERSECT operator.
	CompoundIntersect

	// CompoundExcept is the EXCEPT operator.
	CompoundExcept
)

// RawCompoundBranch holds a single branch of a compound query with its operator and
// unresolved query analysis.
type RawCompoundBranch struct {
	// Query is the unresolved analysis of this branch's SELECT.
	Query *RawQueryAnalysis

	// Operator is the compound operator preceding this branch.
	Operator CompoundOperator
}

// RawDerivedTableReference holds an unresolved subquery in a FROM clause. The domain
// layer resolves the inner query's output columns and converts this into a
// DerivedTableReference for scope resolution.
type RawDerivedTableReference struct {
	// InnerQuery is the unresolved analysis of the subquery.
	InnerQuery *RawQueryAnalysis

	// Alias is the required alias for the derived table.
	Alias string

	// JoinKind indicates how this derived table is joined.
	JoinKind JoinKind
}

// TVFColumnDefinition holds a column name and optional type name from an AS alias(col1
// type1, col2 type2) clause on a table-valued function.
type TVFColumnDefinition struct {
	// Name is the column name.
	Name string

	// TypeName is the raw engine type name (e.g. "text", "integer", "int4[]"), or empty when
	// the column definition provides only a name without a type.
	TypeName string
}

// RawTableValuedFunctionReference holds an unresolved table-valued function call in a
// FROM clause (e.g. json_each, generate_series).
type RawTableValuedFunctionReference struct {
	// FunctionName is the table-valued function name.
	FunctionName string

	// Alias is the table alias used in the query.
	Alias string

	// ColumnDefinitions holds the column definitions from an AS alias(name type, ...)
	// clause. Empty when no column definitions are provided.
	ColumnDefinitions []TVFColumnDefinition

	// JoinKind indicates how this table-valued function is joined.
	JoinKind JoinKind
}

// RawOutputColumn is an unresolved output column from the engine adapter.
type RawOutputColumn struct {
	// Expression holds the typed expression for computed columns.
	Expression Expression

	// Name is the column alias or inferred name.
	Name string

	// TableAlias is the table alias or name this column references, if any.
	TableAlias string

	// ColumnName is the referenced column name, if this is a direct reference.
	ColumnName string

	// IsStar indicates this is a SELECT * expansion.
	IsStar bool
}

// RawParameterReference is an unresolved parameter reference from the engine.
type RawParameterReference struct {
	// ColumnReference is the column this parameter is compared with or assigned to, if
	// applicable.
	ColumnReference *ColumnReference

	// CastType is the explicit cast type, if the parameter appears in a CAST.
	CastType *SQLType

	// Name is the identifier for named parameters (:email, @user_id, $name). Empty for
	// positional/numbered parameters.
	Name string

	// EnclosingFunctionName is the name of the function or table-valued function whose
	// argument list this parameter sits in.
	//
	// It is recorded as the engine writes it (lower-cased, optionally schema-qualified, such
	// as "content.get_pages_with_latest_version" or "string_to_array") and is empty when the
	// parameter is not a function argument. The analyser pairs it with ArgumentOrdinal to
	// look up the matched function signature's declared argument type and back-propagate it
	// onto an otherwise untyped placeholder.
	EnclosingFunctionName string

	// Number is the positional parameter number ($1, ?1, etc.) or sequential ordinal for
	// anonymous question-mark style.
	Number int

	// ArgumentOrdinal is the zero-based position of this parameter among the enclosing
	// call's top-level arguments (the call-site argument slot, NOT the parameter number).
	// Only meaningful when Context is ParameterContextFunctionArgument and
	// EnclosingFunctionName is non-empty.
	ArgumentOrdinal int

	// Context describes where the parameter appears (comparison, function arg, assignment,
	// cast, etc.) for type inference.
	Context ParameterContext
}

// ParameterContext describes where a parameter appears in a query.
type ParameterContext uint8

const (
	// ParameterContextComparison indicates a parameter in a comparison expression.
	ParameterContextComparison ParameterContext = iota

	// ParameterContextAssignment indicates a parameter in a SET assignment.
	ParameterContextAssignment

	// ParameterContextFunctionArgument indicates a parameter as a function argument.
	ParameterContextFunctionArgument

	// ParameterContextCast indicates a parameter inside a CAST expression.
	ParameterContextCast

	// ParameterContextInList indicates a parameter in an IN list.
	ParameterContextInList

	// ParameterContextBetween indicates a parameter in a BETWEEN expression.
	ParameterContextBetween

	// ParameterContextLimit indicates a parameter in a LIMIT clause.
	ParameterContextLimit

	// ParameterContextOffset indicates a parameter in an OFFSET clause.
	ParameterContextOffset

	// ParameterContextLike indicates a parameter inside a LIKE / ILIKE / GLOB / REGEXP
	// pattern expression. The associated ColumnReference (when set) names the column on the
	// left of the pattern operator; the parameter itself always carries a string pattern, so
	// the analyser types it as text regardless of the column's type.
	ParameterContextLike

	// ParameterContextUnknown indicates a parameter in an unrecognised context.
	ParameterContextUnknown
)

// ColumnReference identifies a specific column in a specific table.
type ColumnReference struct {
	// TableAlias is the table alias or name.
	TableAlias string

	// ColumnName is the column name.
	ColumnName string
}

// TableReference identifies a table in the FROM clause.
type TableReference struct {
	// Schema is the table's schema.
	Schema string

	// Name is the table name.
	Name string

	// Alias is the table alias, if specified.
	Alias string
}

// JoinClause describes a JOIN in the query.
type JoinClause struct {
	// Table is the joined table.
	Table TableReference

	// Kind identifies the join type.
	Kind JoinKind
}

// JoinKind identifies the type of JOIN.
type JoinKind uint8

const (
	// JoinInner is an INNER JOIN.
	JoinInner JoinKind = iota

	// JoinLeft is a LEFT OUTER JOIN.
	JoinLeft

	// JoinRight is a RIGHT OUTER JOIN.
	JoinRight

	// JoinFull is a FULL OUTER JOIN.
	JoinFull

	// JoinCross is a CROSS JOIN.
	JoinCross

	// JoinPositional is a DuckDB POSITIONAL JOIN that joins tables by row position, padding
	// with NULLs when one side is shorter.
	JoinPositional

	// JoinAsof is a ClickHouse ASOF JOIN that joins the closest matching row from the right
	// side based on an inequality predicate (typically a timestamp comparison).
	JoinAsof

	// JoinSemi is a ClickHouse SEMI JOIN: returns rows from the left for which any matching
	// row exists on the right, without duplicating left rows. Equivalent to `WHERE EXISTS`.
	JoinSemi

	// JoinAnti is a ClickHouse ANTI JOIN: returns rows from the left for which no matching
	// row exists on the right. Equivalent to `WHERE NOT EXISTS`.
	JoinAnti

	// JoinLeftSemi is a left-biased SEMI JOIN: preserves left-side nullability semantics.
	// Equivalent to JoinSemi for ClickHouse but distinguished for engines that
	// differentiate.
	JoinLeftSemi

	// JoinLeftAnti is a left-biased ANTI JOIN; mirror of JoinLeftSemi.
	JoinLeftAnti

	// JoinRightSemi is a right-biased SEMI JOIN: returns rows from the right side that have
	// a match on the left.
	JoinRightSemi

	// JoinRightAnti is a right-biased ANTI JOIN: returns rows from the right side that have
	// no match on the left.
	JoinRightAnti

	// JoinAny is a ClickHouse ANY JOIN strictness modifier: for each left-side row, returns
	// one matching right-side row at most. The non-strict-multiplicity variant of an inner
	// join.
	JoinAny

	// JoinAll is a ClickHouse ALL JOIN strictness modifier: for each left-side row, returns
	// every matching right-side row. The strict-multiplicity variant of an inner join (and
	// the default).
	JoinAll

	// JoinGlobal is a ClickHouse distributed-join prefix: the right side is computed once on
	// the initiator and broadcast to every shard rather than re-evaluated per shard. The
	// strictness/side is otherwise inner-join semantics.
	JoinGlobal
)

// RawCTEDefinition is an unresolved CTE from the engine adapter.
type RawCTEDefinition struct {
	// EngineSpecific carries free-form metadata an engine attaches to a CTE definition.
	//
	// ClickHouse uses the key CTE_MATERIALIZED with the value "true" / "false" to reflect a
	// trailing MATERIALIZED / NOT MATERIALIZED qualifier. The map is allocated lazily by the
	// engine adapter; consumers must nil-check before reading.
	EngineSpecific map[string]string

	// Name is the CTE name.
	Name string

	// OutputColumns holds the CTE's output columns, if resolvable.
	OutputColumns []RawOutputColumn

	// FromTables holds the tables referenced in the CTE body's FROM clause.
	FromTables []TableReference

	// JoinClauses holds JOIN clauses from the CTE body.
	JoinClauses []JoinClause

	// CompoundBranches holds UNION/INTERSECT/EXCEPT branches from the CTE body.
	CompoundBranches []RawCompoundBranch

	// ParameterReferences holds the parameters that occur in the CTE body.
	//
	// Engines that flatten a CTE body's parameters into the top-level
	// RawQueryAnalysis.ParameterReferences (for parameter-count purposes) also record them
	// here so the shared analyser can resolve each one against the CTE body's own FROM/JOIN
	// scope rather than the outer query scope. A CTE-body column reference (for example an
	// unqualified id) is frequently ambiguous in the outer scope yet unambiguous inside the
	// CTE body; resolving in the body scope removes that false Q002/Q001. Engines that
	// number CTE-body parameters independently (and so do not flatten them) may leave this
	// empty; the analyser's identity guard makes population safe either way.
	ParameterReferences []RawParameterReference

	// IsRecursive indicates whether this is a recursive CTE.
	IsRecursive bool
}

// ScopeKind identifies the context that created a scope in the nested scope chain used
// for column resolution.
type ScopeKind uint8

const (
	// ScopeKindQuery is the root scope of a top-level query.
	ScopeKindQuery ScopeKind = iota

	// ScopeKindCTE is a scope created for a common table expression.
	ScopeKindCTE

	// ScopeKindSubquery is a scope created for a subquery.
	ScopeKindSubquery

	// ScopeKindLateral is a scope created for a LATERAL join that can reference columns from
	// preceding tables in the FROM clause.
	ScopeKindLateral

	// ScopeKindParameter is a scope created for a function body where parameter names
	// resolve as Unknown-typed bindings. The shared function-body analyser binds each
	// declared parameter into this scope so a body expression can reference parameters
	// through the existing column-resolution path.
	ScopeKindParameter
)

// ScopedTable is a table within a scope, carrying its columns with JOIN-adjusted
// nullability.
type ScopedTable struct {
	// Schema is the table's schema name.
	Schema string

	// Name is the table name.
	Name string

	// Alias is the table alias used in the query, or the table name if no alias was
	// specified.
	Alias string

	// Columns holds the columns with JOIN-adjusted nullability.
	Columns []ScopedColumn

	// JoinKind is the join type that introduced this table into the scope.
	JoinKind JoinKind

	// IsWithoutRowID indicates the table has no implicit rowid column.
	IsWithoutRowID bool
}

// ScopedColumn is a column within a scoped table, carrying its resolved type and
// JOIN-adjusted nullability.
type ScopedColumn struct {
	// Name is the column name.
	Name string

	// SQLType is the resolved SQL type from the catalogue.
	SQLType SQLType

	// Nullable indicates whether this column can be NULL in the current scope, accounting
	// for JOIN type adjustments.
	Nullable bool
}
