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

// CatalogueMutation describes a single DDL change to the schema catalogue.
type CatalogueMutation struct {
	// FunctionSignature holds the function signature for CREATE FUNCTION.
	FunctionSignature *FunctionSignature

	// ViewDefinition holds the parsed SELECT body of a CREATE VIEW statement.
	// The catalogue builder uses this to resolve the view's output column types.
	ViewDefinition *RawQueryAnalysis

	// ConstraintName is the constraint affected by DROP CONSTRAINT.
	ConstraintName string

	// TriggerName is the trigger affected, if applicable.
	TriggerName string

	// ColumnName is the specific column affected by ALTER COLUMN or DROP COLUMN.
	ColumnName string

	// EnumName is the enum type affected, if applicable.
	EnumName string

	// SchemaName is the schema affected by the mutation.
	SchemaName string

	// TableName is the table affected, if applicable.
	TableName string

	// OwnedByColumn is the column that owns a sequence.
	OwnedByColumn string

	// VirtualModuleName is the module name from CREATE VIRTUAL TABLE ... USING
	// module(...), or empty for non-virtual tables.
	VirtualModuleName string

	// OwnedByTable is the table that owns a sequence (OWNED BY table.column).
	OwnedByTable string

	// SequenceName is the sequence affected, if applicable.
	SequenceName string

	// NewName holds the target name for RENAME operations.
	NewName string

	// PrimaryKey holds primary key column names for CREATE TABLE.
	PrimaryKey []string

	// Constraints holds table constraints for CREATE TABLE or ADD CONSTRAINT.
	Constraints []Constraint

	// Columns holds column definitions for CREATE TABLE or ADD COLUMN.
	Columns []Column

	// InheritsTables lists parent tables from an INHERITS clause on CREATE TABLE.
	// The catalogue builder prepends each parent's columns to the child table.
	InheritsTables []TableReference

	// EnumValues holds enum values for CREATE TYPE or ADD VALUE.
	EnumValues []string

	// Origin records which migration file produced this mutation. Set by the
	// catalogue builder before applying each mutation.
	Origin MigrationOrigin

	// Kind identifies the type of mutation (create table, alter column, etc.).
	Kind MutationKind

	// IsWithoutRowID indicates the table was created with WITHOUT ROWID.
	IsWithoutRowID bool

	// IsVirtual indicates the table was created with CREATE VIRTUAL TABLE.
	IsVirtual bool
}

// MutationKind identifies the type of DDL mutation.
type MutationKind uint8

const (
	// MutationCreateTable creates a new table.
	MutationCreateTable MutationKind = iota

	// MutationDropTable removes a table.
	MutationDropTable

	// MutationAlterTableAddColumn adds a column to a table.
	MutationAlterTableAddColumn

	// MutationAlterTableDropColumn removes a column from a table.
	MutationAlterTableDropColumn

	// MutationAlterTableAlterColumn modifies a column definition.
	MutationAlterTableAlterColumn

	// MutationAlterTableRenameColumn renames a column.
	MutationAlterTableRenameColumn

	// MutationAlterTableRenameTable renames a table.
	MutationAlterTableRenameTable

	// MutationAlterTableSetSchema moves a table to a different schema.
	MutationAlterTableSetSchema

	// MutationCreateEnum creates an enum type.
	MutationCreateEnum

	// MutationAlterEnumAddValue adds a value to an enum type.
	MutationAlterEnumAddValue

	// MutationAlterEnumRenameValue renames a value in an enum type.
	MutationAlterEnumRenameValue

	// MutationDropEnum removes an enum type.
	MutationDropEnum

	// MutationCreateCompositeType creates a composite type.
	MutationCreateCompositeType

	// MutationDropType removes a type.
	MutationDropType

	// MutationCreateFunction creates a function.
	MutationCreateFunction

	// MutationDropFunction removes a function.
	MutationDropFunction

	// MutationCreateSchema creates a schema.
	MutationCreateSchema

	// MutationDropSchema removes a schema.
	MutationDropSchema

	// MutationCreateView creates a view.
	MutationCreateView

	// MutationDropView removes a view.
	MutationDropView

	// MutationCreateIndex creates an index.
	MutationCreateIndex

	// MutationDropIndex removes an index.
	MutationDropIndex

	// MutationCreateExtension loads an extension.
	MutationCreateExtension

	// MutationComment sets a comment on a database object.
	MutationComment

	// MutationCreateTrigger creates a trigger.
	MutationCreateTrigger

	// MutationDropTrigger removes a trigger.
	MutationDropTrigger

	// MutationAlterTableAddConstraint adds a constraint to a table.
	MutationAlterTableAddConstraint

	// MutationAlterTableDropConstraint removes a constraint from a table.
	MutationAlterTableDropConstraint

	// MutationCreateSequence creates a sequence.
	MutationCreateSequence

	// MutationDropSequence removes a sequence.
	MutationDropSequence

	// MutationKindCount is a sentinel value for array dispatch table sizing.
	MutationKindCount
)

// GeneratedKind indicates how a generated column is stored.
type GeneratedKind uint8

const (
	// GeneratedKindNone indicates the column is not generated.
	GeneratedKindNone GeneratedKind = iota

	// GeneratedKindVirtual indicates a VIRTUAL generated column (computed on
	// read, not physically stored). This is the default in SQLite.
	GeneratedKindVirtual

	// GeneratedKindStored indicates a STORED generated column (computed on
	// write and physically stored).
	GeneratedKindStored
)

// Column represents a column definition within a table.
type Column struct {
	// Name is the column name.
	Name string

	// Comment is the column comment, if any.
	Comment string

	// Origin records which migration introduced this column.
	Origin MigrationOrigin

	// SQLType is the structured type of the column.
	SQLType SQLType

	// ArrayDimensions is the number of array dimensions.
	ArrayDimensions int

	// Nullable indicates whether the column permits NULL values.
	Nullable bool

	// HasDefault indicates whether the column has a DEFAULT expression.
	HasDefault bool

	// IsGenerated indicates whether the column is a generated column.
	IsGenerated bool

	// GeneratedKind distinguishes VIRTUAL from STORED generated columns.
	// Only meaningful when IsGenerated is true.
	GeneratedKind GeneratedKind

	// IsArray indicates whether the column is an array type.
	IsArray bool
}

// FunctionSignature describes a SQL function's type signature.
type FunctionSignature struct {
	// Name is the function name.
	Name string

	// Schema is the schema the function belongs to.
	Schema string

	// Language is the declared function language (e.g. "sql", "plpgsql", "c"),
	// or empty if not declared or not applicable.
	Language string

	// BodySQL holds the raw SQL body text for LANGUAGE sql functions, used by
	// the catalogue builder to re-analyse the function body.
	//
	// Empty for procedural languages or when the body is not captured.
	BodySQL string

	// Arguments describes the function's parameters.
	Arguments []FunctionArgument

	// CalledFunctions records qualified function names called within the body.
	// Populated during body analysis; used for call graph construction.
	CalledFunctions []string

	// Origin records which migration introduced the function.
	Origin MigrationOrigin

	// ReturnType is the function's return type.
	ReturnType SQLType

	// MinArguments is the minimum number of arguments required, where arguments
	// beyond this index have implicit defaults.
	//
	// When zero, defaults to len(Arguments) (all required).
	MinArguments int

	// ReturnsSet indicates whether a set of rows is returned.
	ReturnsSet bool

	// IsAggregate indicates whether the function is an aggregate.
	IsAggregate bool

	// IsStrict indicates that NULL is returned on any NULL argument.
	// PostgreSQL: STRICT or RETURNS NULL ON NULL INPUT.
	IsStrict bool

	// NullableBehaviour describes how the function handles NULL arguments.
	NullableBehaviour FunctionNullableBehaviour

	// DataAccess describes whether the function may modify data. Built-in
	// functions are DataAccessReadOnly; user-defined functions default to
	// DataAccessUnknown (treated as potentially modifying) unless the DDL
	// declares otherwise.
	DataAccess FunctionDataAccess

	// IsVariadic indicates the last argument can repeat zero or more times.
	// When true, the resolver matches any arity >= MinArguments.
	IsVariadic bool
}

// FunctionDataAccess describes whether a function may modify database state.
type FunctionDataAccess uint8

const (
	// DataAccessUnknown means the function's data access is not declared.
	// Treated conservatively as potentially modifying data.
	DataAccessUnknown FunctionDataAccess = iota

	// DataAccessReadOnly means the function does not modify data
	// (PostgreSQL: IMMUTABLE or STABLE; MySQL: NO SQL, READS SQL DATA, or
	// DETERMINISTIC without MODIFIES SQL DATA).
	DataAccessReadOnly

	// DataAccessModifiesData means the function may modify data
	// (PostgreSQL: VOLATILE (default); MySQL: MODIFIES SQL DATA).
	DataAccessModifiesData
)

// FunctionArgument describes a single function parameter.
type FunctionArgument struct {
	// Name is the parameter name, if named.
	Name string

	// Type is the parameter type.
	Type SQLType

	// IsOptional indicates this argument has a default value and can be
	// omitted. Optional arguments must come after all required arguments.
	IsOptional bool
}

// FunctionNullableBehaviour describes how a function handles NULL inputs.
type FunctionNullableBehaviour uint8

const (
	// FunctionNullableCalledOnNull means the function is called even when
	// arguments are NULL. The result nullability depends on the function.
	FunctionNullableCalledOnNull FunctionNullableBehaviour = iota

	// FunctionNullableReturnsNullOnNull means NULL is returned when any
	// argument is NULL (SQL STRICT / RETURNS NULL ON NULL INPUT).
	FunctionNullableReturnsNullOnNull

	// FunctionNullableNeverNull means the function never returns NULL
	// regardless of input (e.g. COUNT, COALESCE).
	FunctionNullableNeverNull
)

// FunctionResolution is the result of engine-provided custom function
// resolution, returned by FunctionResolverPort.ResolveFunctionCall. This
// allows engines to handle context-dependent or polymorphic functions that
// the standard overload resolution cannot match.
type FunctionResolution struct {
	// ReturnType is the resolved return type.
	ReturnType SQLType

	// NullableBehaviour describes how the function handles NULL arguments.
	NullableBehaviour FunctionNullableBehaviour

	// DataAccess describes whether the function may modify data.
	DataAccess FunctionDataAccess

	// IsAggregate indicates whether the function is an aggregate.
	IsAggregate bool

	// ReturnsSet indicates whether a set of rows is returned.
	ReturnsSet bool
}

// MigrationOrigin records which migration file introduced or last modified a
// catalogue object, enabling precise error messages that point back to the DDL
// source.
type MigrationOrigin struct {
	// Filename is the migration file that introduced this object
	// (e.g. "001_create_users.sql").
	Filename string

	// Index is the zero-based sequential position of the migration file
	// after lexicographic sorting.
	Index int
}

// Catalogue represents the full schema state of a database, built from
// replaying migration files.
type Catalogue struct {
	// Schemas maps schema names to their contents.
	Schemas map[string]*Schema

	// Extensions tracks loaded extensions.
	Extensions map[string]struct{}

	// DefaultSchema is the default schema name (e.g. "public" for PostgreSQL).
	DefaultSchema string
}

// Schema represents a single database schema containing tables, views,
// enums, functions, and types.
type Schema struct {
	// Tables maps table names to their definitions.
	Tables map[string]*Table

	// Views maps view names to their definitions.
	Views map[string]*View

	// Enums maps enum type names to their definitions.
	Enums map[string]*Enum

	// Functions maps function names to their overloaded signatures.
	Functions map[string][]*FunctionSignature

	// CompositeTypes maps composite type names to their definitions.
	CompositeTypes map[string]*CompositeType

	// Sequences maps sequence names to their definitions.
	Sequences map[string]*Sequence

	// Name is the schema name.
	Name string
}

// Table represents a database table.
type Table struct {
	// Name is the table name.
	Name string

	// Schema is the schema the table belongs to.
	Schema string

	// Comment is the table comment, if any.
	Comment string

	// VirtualModuleName is the module name from CREATE VIRTUAL TABLE ... USING
	// module(...), or empty for non-virtual tables.
	VirtualModuleName string

	// Columns holds the table's columns in declaration order.
	Columns []Column

	// PrimaryKey holds the primary key columns, if defined.
	PrimaryKey []string

	// Indexes holds the table's indexes.
	Indexes []Index

	// Constraints holds the table's constraints.
	Constraints []Constraint

	// Origin records which migration introduced this table.
	Origin MigrationOrigin

	// IsVirtual indicates the table was created with CREATE VIRTUAL TABLE.
	IsVirtual bool

	// IsWithoutRowID indicates the table was created with WITHOUT ROWID.
	IsWithoutRowID bool
}

// View represents a database view.
type View struct {
	// Name is the view name.
	Name string

	// Schema is the schema the view belongs to.
	Schema string

	// Columns holds the view's output columns.
	Columns []Column

	// Definition is the SQL SELECT statement defining the view.
	Definition string

	// Comment is the view comment, if any.
	Comment string

	// Origin records which migration introduced this view.
	Origin MigrationOrigin
}

// Enum represents a user-defined enum type.
type Enum struct {
	// Name is the enum type name.
	Name string

	// Schema is the schema the enum belongs to.
	Schema string

	// Values holds the enum values in declaration order.
	Values []string

	// Comment is the enum type comment, if any.
	Comment string

	// Origin records which migration introduced this enum.
	Origin MigrationOrigin
}

// CompositeType represents a user-defined composite type.
type CompositeType struct {
	// Name is the composite type name.
	Name string

	// Schema is the schema the type belongs to.
	Schema string

	// Fields holds the type's fields.
	Fields []Column

	// Origin records which migration introduced this composite type.
	Origin MigrationOrigin
}

// Sequence represents a database sequence.
type Sequence struct {
	// Name is the sequence name.
	Name string

	// Schema is the schema the sequence belongs to.
	Schema string

	// OwnedByTable is the table that owns this sequence (for serial columns).
	OwnedByTable string

	// OwnedByColumn is the column that owns this sequence.
	OwnedByColumn string

	// Origin records which migration introduced this sequence.
	Origin MigrationOrigin
}

// Index represents a database index.
type Index struct {
	// Name is the index name.
	Name string

	// Columns holds the indexed column names.
	Columns []string

	// Origin records which migration introduced this index.
	Origin MigrationOrigin

	// IsUnique indicates whether the index enforces uniqueness.
	IsUnique bool

	// IsPrimary indicates whether this is the primary key index.
	IsPrimary bool
}

// Constraint represents a database constraint.
type Constraint struct {
	// Name is the constraint name.
	Name string

	// ForeignTable is the referenced table for foreign key constraints.
	ForeignTable string

	// Columns holds the constrained column names.
	Columns []string

	// ForeignColumns holds the referenced columns for foreign key constraints.
	ForeignColumns []string

	// Origin records which migration introduced this constraint.
	Origin MigrationOrigin

	// Kind identifies the constraint type.
	Kind ConstraintKind
}

// ConstraintKind identifies the type of constraint.
type ConstraintKind uint8

const (
	// ConstraintPrimaryKey is a PRIMARY KEY constraint.
	ConstraintPrimaryKey ConstraintKind = iota

	// ConstraintForeignKey is a FOREIGN KEY constraint.
	ConstraintForeignKey

	// ConstraintUnique is a UNIQUE constraint.
	ConstraintUnique

	// ConstraintCheck is a CHECK constraint.
	ConstraintCheck

	// ConstraintNotNull is a NOT NULL constraint.
	ConstraintNotNull
)
