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

// DerivedTableSource identifies the origin of a derived (virtual) table in the
// FROM clause.
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

// DerivedTableReference describes a virtual table in the FROM clause that does
// not exist in the catalogue. Engine adapters emit these for UNNEST, FLATTEN,
// table-valued functions, and subqueries in FROM position.
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

// RawQueryAnalysis is the engine adapter's initial analysis of a query before
// the domain applies type resolution and nullability propagation.
type RawQueryAnalysis struct {
	// InsertTable is the target table name for INSERT statements.
	InsertTable string

	// RawDerivedTables holds unresolved subqueries in FROM clauses. The domain
	// layer resolves these and converts them to DerivedTableReference entries.
	RawDerivedTables []RawDerivedTableReference

	// FromTables holds the tables referenced in the FROM clause.
	FromTables []TableReference

	// JoinClauses holds the JOIN clauses with their types.
	JoinClauses []JoinClause

	// CTEDefinitions holds any WITH clause CTE definitions.
	CTEDefinitions []RawCTEDefinition

	// DerivedTables holds virtual tables from UNNEST, FLATTEN, table-valued
	// functions, or subqueries in the FROM clause.
	DerivedTables []DerivedTableReference

	// OutputColumns holds the unresolved output column references.
	OutputColumns []RawOutputColumn

	// GroupByColumns holds the columns referenced in a GROUP BY clause.
	GroupByColumns []ColumnReference

	// CompoundBranches holds the branches of a compound query
	// (UNION, UNION ALL, INTERSECT, EXCEPT).
	CompoundBranches []RawCompoundBranch

	// RawTableValuedFunctions holds unresolved table-valued function calls in
	// FROM clauses. The domain layer resolves these into DerivedTableReference
	// entries.
	RawTableValuedFunctions []RawTableValuedFunctionReference

	// ParameterReferences holds the unresolved parameter references.
	ParameterReferences []RawParameterReference

	// InsertColumns holds the target column names for INSERT statements.
	InsertColumns []string

	// HasReturning indicates whether the statement has a RETURNING clause.
	HasReturning bool

	// ReadOnly indicates the query does not modify data. SELECT and VALUES
	// statements are read-only unless they contain FOR UPDATE/SHARE locking
	// clauses or data-modifying CTEs (INSERT/UPDATE/DELETE inside WITH).
	ReadOnly bool
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

// RawCompoundBranch holds a single branch of a compound query with its
// operator and unresolved query analysis.
type RawCompoundBranch struct {
	// Query is the unresolved analysis of this branch's SELECT.
	Query *RawQueryAnalysis

	// Operator is the compound operator preceding this branch.
	Operator CompoundOperator
}

// RawDerivedTableReference holds an unresolved subquery in a FROM clause.
// The domain layer resolves the inner query's output columns and converts
// this into a DerivedTableReference for scope resolution.
type RawDerivedTableReference struct {
	// InnerQuery is the unresolved analysis of the subquery.
	InnerQuery *RawQueryAnalysis

	// Alias is the required alias for the derived table.
	Alias string

	// JoinKind indicates how this derived table is joined.
	JoinKind JoinKind
}

// TVFColumnDefinition holds a column name and optional type name from an
// AS alias(col1 type1, col2 type2) clause on a table-valued function.
type TVFColumnDefinition struct {
	// Name is the column name.
	Name string

	// TypeName is the raw engine type name (e.g. "text", "integer", "int4[]"),
	// or empty when the column definition provides only a name without a type.
	TypeName string
}

// RawTableValuedFunctionReference holds an unresolved table-valued function
// call in a FROM clause (e.g. json_each, generate_series).
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
	// ColumnReference is the column this parameter is compared with or assigned
	// to, if applicable.
	ColumnReference *ColumnReference

	// CastType is the explicit cast type, if the parameter appears in a CAST.
	CastType *SQLType

	// Name is the identifier for named parameters (:email, @user_id, $name).
	// Empty for positional/numbered parameters.
	Name string

	// Number is the positional parameter number ($1, ?1, etc.) or sequential
	// ordinal for anonymous question-mark style.
	Number int

	// Context describes where the parameter appears (comparison, function arg,
	// assignment, cast, etc.) for type inference.
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

	// ParameterContextLike indicates a parameter inside a LIKE / ILIKE /
	// GLOB / REGEXP pattern expression. The associated ColumnReference
	// (when set) names the column on the left of the pattern operator;
	// the parameter itself always carries a string pattern, so the
	// analyser types it as text regardless of the column's type.
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

	// JoinPositional is a DuckDB POSITIONAL JOIN that joins tables by row
	// position, padding with NULLs when one side is shorter.
	JoinPositional
)

// RawCTEDefinition is an unresolved CTE from the engine adapter.
type RawCTEDefinition struct {
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

	// IsRecursive indicates whether this is a recursive CTE.
	IsRecursive bool
}

// ScopeKind identifies the context that created a scope in the nested scope
// chain used for column resolution.
type ScopeKind uint8

const (
	// ScopeKindQuery is the root scope of a top-level query.
	ScopeKindQuery ScopeKind = iota

	// ScopeKindCTE is a scope created for a common table expression.
	ScopeKindCTE

	// ScopeKindSubquery is a scope created for a subquery.
	ScopeKindSubquery

	// ScopeKindLateral is a scope created for a LATERAL join that can
	// reference columns from preceding tables in the FROM clause.
	ScopeKindLateral
)

// ScopedTable is a table within a scope, carrying its columns with
// JOIN-adjusted nullability.
type ScopedTable struct {
	// Schema is the table's schema name.
	Schema string

	// Name is the table name.
	Name string

	// Alias is the table alias used in the query, or the table name if no
	// alias was specified.
	Alias string

	// Columns holds the columns with JOIN-adjusted nullability.
	Columns []ScopedColumn

	// JoinKind is the join type that introduced this table into the scope.
	JoinKind JoinKind

	// IsWithoutRowID indicates the table has no implicit rowid column.
	IsWithoutRowID bool
}

// ScopedColumn is a column within a scoped table, carrying its resolved type
// and JOIN-adjusted nullability.
type ScopedColumn struct {
	// Name is the column name.
	Name string

	// SQLType is the resolved SQL type from the catalogue.
	SQLType SQLType

	// Nullable indicates whether this column can be NULL in the current scope,
	// accounting for JOIN type adjustments.
	Nullable bool
}
