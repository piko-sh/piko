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

	// ViewDefinition holds the parsed SELECT body of a CREATE VIEW statement. The catalogue
	// builder uses this to resolve the view's output column types.
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

	// VirtualModuleName is the module name from CREATE VIRTUAL TABLE ... USING module(...),
	// or empty for non-virtual tables.
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

	// AdditionalMutations carries follow-up mutations produced by a single DDL statement
	// that the engine could not collapse into the primary mutation.
	//
	// The catalogue builder applies these in order immediately after the primary, with the
	// same Origin. Used for multi-action ALTER TABLE statements where a single statement
	// adds a column and also adds a constraint, for example.
	AdditionalMutations []*CatalogueMutation

	// EngineSpecific carries free-form metadata an engine attaches to a mutation that the
	// rest of the pipeline does not interpret.
	//
	// The ClickHouse engine uses it to preserve table-engine declarations (e.g. ENGINE =
	// MergeTree(), PARTITION BY, ORDER BY, SETTINGS, TTL) that affect query planning but not
	// codegen. Other engines leave it nil.
	EngineSpecific map[string]string

	// InheritsTables lists parent tables from an INHERITS clause on CREATE TABLE. The
	// catalogue builder prepends each parent's columns to the child table.
	InheritsTables []TableReference

	// EnumValues holds enum values for CREATE TYPE or ADD VALUE.
	EnumValues []string

	// Origin records which migration file produced this mutation. Set by the catalogue
	// builder before applying each mutation.
	Origin MigrationOrigin

	// Kind identifies the type of mutation (create table, alter column, etc.).
	Kind MutationKind

	// IsWithoutRowID indicates the table was created with WITHOUT ROWID.
	IsWithoutRowID bool

	// IsVirtual indicates the table was created with CREATE VIRTUAL TABLE.
	IsVirtual bool
}

// CatalogueMutationOption sets an optional attribute of a mutation built by
// NewCatalogueMutation.
type CatalogueMutationOption func(*CatalogueMutation)

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

	// MutationAsyncDataUpdate records a data-mutating ALTER TABLE ... UPDATE statement
	// (ClickHouse).
	//
	// The mutation does not change the catalogue's schema view; it carries the WHERE/SET
	// text in EngineSpecific so the migration runner can execute it and codegen can ignore
	// it.
	MutationAsyncDataUpdate

	// MutationAsyncDataDelete records a data-mutating ALTER TABLE ... DELETE statement
	// (ClickHouse).
	//
	// See MutationAsyncDataUpdate for the storage convention.
	MutationAsyncDataDelete

	// MutationCreateDictionary creates a ClickHouse dictionary (key-value lookup table). The
	// dictionary is catalogue-visible as a view-like entity; its columns plus structured
	// engine metadata (PRIMARY KEY, SOURCE, LAYOUT, LIFETIME) are captured on the mutation.
	MutationCreateDictionary

	// MutationDropDictionary removes a ClickHouse dictionary.
	MutationDropDictionary

	// MutationExchangeTables atomically swaps two ClickHouse tables. The primary mutation
	// carries the left table; the right table name is stored under the EngineSpecific key
	// EXCHANGE_TARGET so downstream consumers can read both sides.
	MutationExchangeTables

	// MutationAlterTableAddProjection adds a projection to a table (ClickHouse). The
	// projection name is carried under EngineSpecific[PROJECTION_NAME] and the SELECT body
	// under EngineSpecific[PROJECTION_SELECT].
	MutationAlterTableAddProjection

	// MutationAlterTableDropProjection removes a projection from a table. The projection
	// name is carried under EngineSpecific[PROJECTION_NAME].
	MutationAlterTableDropProjection

	// MutationAlterTableMaterializeProjection rebuilds a projection's data on disk; pure
	// runtime operation captured for migration-runner replay.
	MutationAlterTableMaterializeProjection

	// MutationAlterTableAddSkippingIndex adds a data-skipping index to a table.
	// EngineSpecific carries INDEX_NAME, INDEX_EXPR, INDEX_TYPE, INDEX_GRANULARITY.
	MutationAlterTableAddSkippingIndex

	// MutationAlterTableDropSkippingIndex removes a data-skipping index. EngineSpecific
	// carries INDEX_NAME.
	MutationAlterTableDropSkippingIndex

	// MutationAlterTableMaterializeIndex rebuilds a data-skipping index; pure runtime
	// operation captured for migration-runner replay.
	MutationAlterTableMaterializeIndex

	// MutationAlterTableAddStatistics adds column statistics to a table. EngineSpecific
	// carries STATS_COLUMNS, STATS_TYPES.
	MutationAlterTableAddStatistics

	// MutationAlterTableDropStatistics removes column statistics from a table.
	// EngineSpecific carries STATS_COLUMNS.
	MutationAlterTableDropStatistics

	// MutationAlterTableMaterializeStatistics rebuilds column statistics; pure runtime
	// operation captured for migration-runner replay.
	MutationAlterTableMaterializeStatistics

	// MutationAlterTableModifyStatistics modifies column statistics. EngineSpecific carries
	// STATS_COLUMNS, STATS_TYPES.
	MutationAlterTableModifyStatistics

	// MutationAlterTableMaterializeColumn rebuilds a materialised column's stored data; pure
	// runtime operation captured for migration-runner replay.
	MutationAlterTableMaterializeColumn

	// MutationAlterTableModifyColumn captures sub-target column modifications (REMOVE
	// DEFAULT, MODIFY COMMENT, RESET SETTING) that do not change the column's type.
	// EngineSpecific carries COLUMN_REMOVE, COLUMN_MODIFY_COMMENT, or COLUMN_RESET_SETTING.
	MutationAlterTableModifyColumn

	// MutationAlterTableModifyQuery captures a refreshable materialised view's body
	// replacement (ALTER TABLE v MODIFY QUERY). EngineSpecific[NEW_QUERY] carries the new
	// SELECT text.
	MutationAlterTableModifyQuery

	// MutationAlterTableModifyRefresh captures changes to a refreshable materialised view's
	// refresh policy (ALTER TABLE v MODIFY REFRESH / SQL SECURITY / DEFINER).
	MutationAlterTableModifyRefresh

	// MutationAlterTablePartition captures partition operations on a MergeTree table
	// (ATTACH/DETACH/DROP/MOVE/REPLACE/FETCH/FREEZE/ UNFREEZE PARTITION). EngineSpecific
	// carries PARTITION_OP, PARTITION_TARGET, PARTITION_EXPR, and optional PARTITION_DEST,
	// PARTITION_BACKUP_NAME, PARTITION_FROM_TABLE, PARTITION_DETACHED.
	MutationAlterTablePartition

	// MutationCreateUser captures a CREATE USER statement (RBAC). EngineSpecific[USER_NAME]
	// carries the user name; the rest of the statement text is captured opaquely under
	// EngineSpecific[STATEMENT_BODY].
	MutationCreateUser

	// MutationAlterUser captures an ALTER USER statement (RBAC).
	MutationAlterUser

	// MutationDropUser captures a DROP USER statement (RBAC).
	MutationDropUser

	// MutationCreateRole captures a CREATE ROLE statement (RBAC).
	MutationCreateRole

	// MutationAlterRole captures an ALTER ROLE statement (RBAC).
	MutationAlterRole

	// MutationDropRole captures a DROP ROLE statement (RBAC).
	MutationDropRole

	// MutationCreatePolicy captures a CREATE ROW POLICY statement (RBAC).
	MutationCreatePolicy

	// MutationAlterPolicy captures an ALTER POLICY statement (RBAC).
	MutationAlterPolicy

	// MutationDropPolicy captures a DROP POLICY statement (RBAC).
	MutationDropPolicy

	// MutationCreateQuota captures a CREATE QUOTA statement (RBAC).
	MutationCreateQuota

	// MutationAlterQuota captures an ALTER QUOTA statement (RBAC).
	MutationAlterQuota

	// MutationDropQuota captures a DROP QUOTA statement (RBAC).
	MutationDropQuota

	// MutationCreateSettingsProfile captures a CREATE SETTINGS PROFILE statement (RBAC).
	MutationCreateSettingsProfile

	// MutationAlterSettingsProfile captures an ALTER SETTINGS PROFILE statement (RBAC).
	MutationAlterSettingsProfile

	// MutationDropSettingsProfile captures a DROP SETTINGS PROFILE statement (RBAC).
	MutationDropSettingsProfile

	// MutationGrantManagement captures a GRANT or REVOKE statement (RBAC).
	// EngineSpecific[RBAC_KIND] is GRANT or REVOKE; the body text is under
	// EngineSpecific[STATEMENT_BODY].
	MutationGrantManagement

	// MutationAttachTable captures an ATTACH TABLE statement, which re-registers an existing
	// on-disk table with the server.
	MutationAttachTable

	// MutationDetachTable captures a DETACH TABLE statement, which unregisters a table
	// without removing its data.
	MutationDetachTable

	// MutationBackup captures a BACKUP statement. The body text is under
	// EngineSpecific[STATEMENT_BODY].
	MutationBackup

	// MutationRestore captures a RESTORE statement. The body text is under
	// EngineSpecific[STATEMENT_BODY].
	MutationRestore

	// MutationKillQuery captures a KILL QUERY statement.
	MutationKillQuery

	// MutationKillMutation captures a KILL MUTATION statement.
	MutationKillMutation

	// MutationKindCount is a sentinel value for array dispatch table sizing.
	MutationKindCount
)

// GeneratedKind indicates how a generated column is stored.
type GeneratedKind uint8

const (
	// GeneratedKindNone indicates the column is not generated.
	GeneratedKindNone GeneratedKind = iota

	// GeneratedKindVirtual indicates a VIRTUAL generated column (computed on read, not
	// physically stored). This is the default in SQLite.
	GeneratedKindVirtual

	// GeneratedKindStored indicates a STORED generated column (computed on write and
	// physically stored).
	GeneratedKindStored
)

// Column represents a column definition within a table.
type Column struct {
	// Name is the column name.
	Name string

	// Comment is the column comment, if any.
	Comment string

	// SQLTypeOverride, when non-empty, carries the engine-specific SQL type name declared
	// via a migration-level piko.column(table.col, type: ...) directive. The query analyser
	// uses this to replace the engine-inferred SQLType on any output column that traces back
	// to this column via direct projection.
	SQLTypeOverride string

	// Origin records which migration introduced this column.
	Origin MigrationOrigin

	// GoTypeOverride, when non-nil, carries the import-path-qualified custom Go destination
	// type declared via a migration-level piko.column(table.col, go_type: ...) directive.
	// Propagates to output columns that project this column unchanged.
	GoTypeOverride *GoType

	// NullableOverride is non-nil when a migration-level piko.column directive declared an
	// explicit nullability for this column, overriding what the engine inferred from the
	// CREATE TABLE.
	NullableOverride *bool

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

	// GeneratedKind distinguishes VIRTUAL from STORED generated columns. Only meaningful
	// when IsGenerated is true.
	GeneratedKind GeneratedKind

	// IsArray indicates whether the column is an array type.
	IsArray bool
}

// FunctionSignature describes a SQL function's type signature.
type FunctionSignature struct {
	// BodyExpression is the parsed expression tree of a function whose body fits in a single
	// expression (ClickHouse lambdas, postgres SQL functions with a single RETURN
	// expression).
	//
	// Engines populate this during ApplyDDL when the body is structurally parsed;
	// catalogue_builder_function runs the shared analyser to fill in ReturnType when it is
	// left zero. Tagged json:"-" so persisted catalogue snapshots do not bloat with
	// expression trees.
	BodyExpression Expression `json:"-"`

	// Name is the function name.
	Name string

	// Schema is the schema the function belongs to.
	Schema string

	// Language is the declared function language (e.g. "sql", "plpgsql", "c"), or empty if
	// not declared or not applicable.
	Language string

	// BodySQL holds the raw SQL body text for LANGUAGE sql functions, used by the catalogue
	// builder to re-analyse the function body.
	//
	// Empty for procedural languages or when the body is not captured.
	BodySQL string

	// Arguments describes the function's parameters.
	Arguments []FunctionArgument

	// CalledFunctions records qualified function names called within the body. Populated
	// during body analysis; used for call graph construction.
	CalledFunctions []string

	// BodyParameters lists the parameter names in lexical order so the analyser can bind
	// them into a child scope before resolving BodyExpression.
	//
	// Mirrors LambdaExpression.Parameters but lives on the signature so the analyser does
	// not need to special-case the expression kind. Tagged json:"-" because it is only used
	// together with BodyExpression and shares its in-memory-only lifecycle.
	BodyParameters []string `json:"-"`

	// Origin records which migration introduced the function.
	Origin MigrationOrigin

	// ReturnType is the function's return type.
	ReturnType SQLType

	// MinArguments is the minimum number of arguments a caller must supply; arguments beyond
	// this index are optional (they carry defaults).
	//
	// When left zero on a non-variadic signature, the overload resolver derives the true
	// minimum from the arguments' IsOptional flags via MinimumArguments, so a genuinely-zero
	// minimum (a leading optional argument) is not conflated with an unpopulated field. Set
	// it explicitly only to override that derivation. On a variadic signature the value is
	// always authoritative, so zero means a call may pass no arguments at all.
	MinArguments int

	// ReturnsSet indicates whether a set of rows is returned.
	ReturnsSet bool

	// IsAggregate indicates whether the function is an aggregate.
	IsAggregate bool

	// IsStrict indicates that NULL is returned on any NULL argument. PostgreSQL: STRICT or
	// RETURNS NULL ON NULL INPUT.
	IsStrict bool

	// NullableBehaviour describes how the function handles NULL arguments.
	NullableBehaviour FunctionNullableBehaviour

	// DataAccess describes whether the function may modify data. Built-in functions are
	// DataAccessReadOnly; user-defined functions default to DataAccessUnknown (treated as
	// potentially modifying) unless the DDL declares otherwise.
	DataAccess FunctionDataAccess

	// IsVariadic indicates the last argument can repeat zero or more times. When true, the
	// resolver matches any arity >= MinArguments.
	IsVariadic bool
}

// FunctionSignatureOption sets an optional attribute of a signature built by
// NewFunctionSignature.
type FunctionSignatureOption func(*FunctionSignature)

// FunctionDataAccess describes whether a function may modify database state.
type FunctionDataAccess uint8

const (
	// DataAccessUnknown means the function's data access is not declared. Treated
	// conservatively as potentially modifying data.
	DataAccessUnknown FunctionDataAccess = iota

	// DataAccessReadOnly means the function does not modify data (PostgreSQL: IMMUTABLE or
	// STABLE; MySQL: NO SQL, READS SQL DATA, or DETERMINISTIC without MODIFIES SQL DATA).
	DataAccessReadOnly

	// DataAccessModifiesData means the function may modify data (PostgreSQL: VOLATILE
	// (default); MySQL: MODIFIES SQL DATA).
	DataAccessModifiesData
)

// FunctionArgument describes a single function parameter.
type FunctionArgument struct {
	// Name is the parameter name, if named.
	Name string

	// Type is the parameter type.
	Type SQLType

	// IsOptional indicates this argument has a default value and can be omitted. Optional
	// arguments must come after all required arguments.
	IsOptional bool
}

// FunctionNullableBehaviour describes how a function handles NULL inputs.
type FunctionNullableBehaviour uint8

const (
	// FunctionNullableCalledOnNull means the function is called even when arguments are
	// NULL. The result nullability depends on the function.
	FunctionNullableCalledOnNull FunctionNullableBehaviour = iota

	// FunctionNullableReturnsNullOnNull means NULL is returned when any argument is NULL
	// (SQL STRICT / RETURNS NULL ON NULL INPUT).
	FunctionNullableReturnsNullOnNull

	// FunctionNullableNeverNull means the function never returns NULL regardless of input
	// (e.g. COUNT, COALESCE).
	FunctionNullableNeverNull
)

// FunctionResolution is the result of engine-provided custom function resolution,
// returned by FunctionResolverPort.ResolveFunctionCall. This allows engines to handle
// context-dependent or polymorphic functions that the standard overload resolution cannot
// match.
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

// MigrationOrigin records which migration file introduced or last modified a catalogue
// object, enabling precise error messages that point back to the DDL source.
type MigrationOrigin struct {
	// Filename is the migration file that introduced this object (e.g.
	// "001_create_users.sql").
	Filename string

	// Index is the zero-based sequential position of the migration file after lexicographic
	// sorting.
	Index int
}

// Catalogue represents the full schema state of a database, built from replaying
// migration files.
type Catalogue struct {
	// Schemas maps schema names to their contents.
	Schemas map[string]*Schema

	// Extensions tracks loaded extensions.
	Extensions map[string]struct{}

	// DefaultSchema is the default schema name (e.g. "public" for PostgreSQL).
	DefaultSchema string
}

// Schema represents a single database schema containing tables, views, enums, functions,
// and types.
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

	// VirtualModuleName is the module name from CREATE VIRTUAL TABLE ... USING module(...),
	// or empty for non-virtual tables.
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

// MinimumArguments returns the minimum number of arguments a caller must supply for the
// given argument list: the count of leading non-optional arguments. Optional arguments
// (those carrying defaults) must trail the required ones, so the first optional argument
// marks the minimum; a list with no optional arguments requires all of them.
//
// Takes arguments ([]FunctionArgument) which are the declared arguments.
//
// Returns int which is the count of leading required arguments.
func MinimumArguments(arguments []FunctionArgument) int {
	for index := range arguments {
		if arguments[index].IsOptional {
			return index
		}
	}
	return len(arguments)
}

// NewColumn returns a column of the given name, type, and nullability, with every other
// attribute unset.
//
// Takes name (string) which is the column's name.
// Takes sqlType (SQLType) which is the column's type.
// Takes nullable (bool) which reports whether the column permits NULL values.
//
// Returns Column which carries only the name, type, and nullability.
func NewColumn(name string, sqlType SQLType, nullable bool) Column {
	column := Column{}
	column.Name = name
	column.SQLType = sqlType
	column.Nullable = nullable
	return column
}

// NewFunctionSignature returns a signature taking arguments and returning returnType,
// with every other attribute unset unless an option sets it.
//
// Takes arguments ([]FunctionArgument) which are the declared arguments, or nil for none.
// Takes returnType (SQLType) which specifies the result type.
// Takes nullableBehaviour (FunctionNullableBehaviour) which says when the result is NULL.
// Takes options (...FunctionSignatureOption) which set any further attributes.
//
// Returns *FunctionSignature which is the configured signature.
func NewFunctionSignature(
	arguments []FunctionArgument,
	returnType SQLType,
	nullableBehaviour FunctionNullableBehaviour,
	options ...FunctionSignatureOption,
) *FunctionSignature {
	signature := &FunctionSignature{}
	signature.Arguments = arguments
	signature.ReturnType = returnType
	signature.NullableBehaviour = nullableBehaviour
	for _, option := range options {
		option(signature)
	}
	return signature
}

// NewFunctionReference returns a signature that identifies a function by schema and name
// alone, with no arguments and a zero return type, for statements that name a function
// without declaring its full shape.
//
// Takes schemaName (string) which is the schema the function belongs to, or "" for the
// default.
// Takes functionName (string) which is the function's name.
//
// Returns *FunctionSignature which carries only the schema and the name.
func NewFunctionReference(schemaName string, functionName string) *FunctionSignature {
	return NewFunctionSignature(
		nil,
		SQLType{},
		FunctionNullableCalledOnNull,
		WithFunctionName(functionName),
		withFunctionSchema(schemaName),
	)
}

// WithFunctionName names the function a signature belongs to.
//
// Takes name (string) which is the function's name.
//
// Returns FunctionSignatureOption which sets the name.
func WithFunctionName(name string) FunctionSignatureOption {
	return func(signature *FunctionSignature) {
		signature.Name = name
	}
}

// withFunctionSchema sets the schema a signature's function belongs to.
//
// Takes schemaName (string) which is the schema's name.
//
// Returns FunctionSignatureOption which sets Schema.
func withFunctionSchema(schemaName string) FunctionSignatureOption {
	return func(signature *FunctionSignature) {
		signature.Schema = schemaName
	}
}

// WithAggregate marks a signature as an aggregate function.
//
// Returns FunctionSignatureOption which sets IsAggregate.
func WithAggregate() FunctionSignatureOption {
	return func(signature *FunctionSignature) {
		signature.IsAggregate = true
	}
}

// WithVariadic marks a signature as variadic, accepting at least minArguments arguments.
//
// Takes minArguments (int) which is the fewest arguments a call may pass.
//
// Returns FunctionSignatureOption which sets IsVariadic and MinArguments.
func WithVariadic(minArguments int) FunctionSignatureOption {
	return func(signature *FunctionSignature) {
		signature.IsVariadic = true
		signature.MinArguments = minArguments
	}
}

// WithMinArguments sets the fewest arguments a call to a non-variadic signature may pass,
// for functions whose trailing arguments are optional.
//
// Takes minArguments (int) which is the fewest arguments a call may pass.
//
// Returns FunctionSignatureOption which sets MinArguments.
func WithMinArguments(minArguments int) FunctionSignatureOption {
	return func(signature *FunctionSignature) {
		signature.MinArguments = minArguments
	}
}

// WithReturnsSet marks a signature as returning a set of rows.
//
// Returns FunctionSignatureOption which sets ReturnsSet.
func WithReturnsSet() FunctionSignatureOption {
	return func(signature *FunctionSignature) {
		signature.ReturnsSet = true
	}
}

// NewCatalogueMutation returns a mutation of the given kind against a schema and table,
// with every other attribute unset unless an option sets it.
//
// Takes kind (MutationKind) which is the operation the mutation applies.
// Takes schemaName (string) which is the schema it targets, or "" for the default.
// Takes tableName (string) which is the table it targets, or "" when it targets none.
// Takes options (...CatalogueMutationOption) which set any further attributes.
//
// Returns *CatalogueMutation which is the configured mutation.
func NewCatalogueMutation(
	kind MutationKind,
	schemaName string,
	tableName string,
	options ...CatalogueMutationOption,
) *CatalogueMutation {
	mutation := &CatalogueMutation{}
	mutation.Kind = kind
	mutation.SchemaName = schemaName
	mutation.TableName = tableName
	for _, option := range options {
		option(mutation)
	}
	return mutation
}

// WithEngineSpecific attaches engine-specific attributes to a mutation.
//
// Takes engineSpecific (map[string]string) which holds the attributes.
//
// Returns CatalogueMutationOption which sets EngineSpecific.
func WithEngineSpecific(engineSpecific map[string]string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.EngineSpecific = engineSpecific
	}
}

// WithNewName sets the name a renaming mutation gives its target.
//
// Takes newName (string) which is the new name.
//
// Returns CatalogueMutationOption which sets NewName.
func WithNewName(newName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.NewName = newName
	}
}

// WithColumnName sets the column a mutation targets.
//
// Takes columnName (string) which is the column's name.
//
// Returns CatalogueMutationOption which sets ColumnName.
func WithColumnName(columnName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.ColumnName = columnName
	}
}

// WithColumns sets the column definitions a mutation adds or replaces.
//
// Takes columns ([]Column) which are the column definitions.
//
// Returns CatalogueMutationOption which sets Columns.
func WithColumns(columns []Column) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.Columns = columns
	}
}

// WithEnum sets the enum type a mutation creates or alters, and its values.
//
// Takes enumName (string) which is the enum type's name.
// Takes enumValues ([]string) which are its values, in order.
//
// Returns CatalogueMutationOption which sets EnumName and EnumValues.
func WithEnum(enumName string, enumValues []string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.EnumName = enumName
		mutation.EnumValues = enumValues
	}
}

// WithFunction sets the function signature a mutation creates or drops.
//
// Takes signature (*FunctionSignature) which is the function's signature.
//
// Returns CatalogueMutationOption which sets FunctionSignature.
func WithFunction(signature *FunctionSignature) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.FunctionSignature = signature
	}
}

// WithTriggerName sets the trigger a mutation creates or drops.
//
// Takes triggerName (string) which is the trigger's name.
//
// Returns CatalogueMutationOption which sets TriggerName.
func WithTriggerName(triggerName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.TriggerName = triggerName
	}
}

// WithPrimaryKey sets the primary key columns of a table a mutation creates.
//
// Takes primaryKey ([]string) which are the primary key column names, in order.
//
// Returns CatalogueMutationOption which sets PrimaryKey.
func WithPrimaryKey(primaryKey []string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.PrimaryKey = primaryKey
	}
}

// WithConstraints sets the table constraints a mutation creates or adds.
//
// Takes constraints ([]Constraint) which are the table constraints.
//
// Returns CatalogueMutationOption which sets Constraints.
func WithConstraints(constraints []Constraint) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.Constraints = constraints
	}
}

// WithConstraintName sets the constraint a mutation drops.
//
// Takes constraintName (string) which is the constraint's name.
//
// Returns CatalogueMutationOption which sets ConstraintName.
func WithConstraintName(constraintName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.ConstraintName = constraintName
	}
}

// WithInheritsTables sets the parent tables a created table inherits columns from.
//
// Takes inheritsTables ([]TableReference) which are the parent tables, in order.
//
// Returns CatalogueMutationOption which sets InheritsTables.
func WithInheritsTables(inheritsTables []TableReference) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.InheritsTables = inheritsTables
	}
}

// WithWithoutRowID records whether a created table was declared WITHOUT ROWID.
//
// Takes isWithoutRowID (bool) which is true when the table has no implicit rowid.
//
// Returns CatalogueMutationOption which sets IsWithoutRowID.
func WithWithoutRowID(isWithoutRowID bool) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.IsWithoutRowID = isWithoutRowID
	}
}

// WithVirtualModule marks a created table as virtual, backed by the named module.
//
// Takes moduleName (string) which is the module from the USING clause.
//
// Returns CatalogueMutationOption which sets IsVirtual and VirtualModuleName.
func WithVirtualModule(moduleName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.IsVirtual = true
		mutation.VirtualModuleName = moduleName
	}
}

// WithSequence sets the sequence a mutation creates or drops.
//
// Takes sequenceName (string) which is the sequence's name.
//
// Returns CatalogueMutationOption which sets SequenceName.
func WithSequence(sequenceName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.SequenceName = sequenceName
	}
}

// WithSequenceOwner sets the table column that owns a sequence.
//
// Takes ownedByTable (string) which is the owning table's name.
// Takes ownedByColumn (string) which is the owning column's name.
//
// Returns CatalogueMutationOption which sets OwnedByTable and OwnedByColumn.
func WithSequenceOwner(ownedByTable string, ownedByColumn string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.OwnedByTable = ownedByTable
		mutation.OwnedByColumn = ownedByColumn
	}
}

// WithTypeName sets the user-defined type, such as an enum or composite type, a mutation
// creates, alters, or drops.
//
// Takes typeName (string) which is the type's name.
//
// Returns CatalogueMutationOption which sets EnumName.
func WithTypeName(typeName string) CatalogueMutationOption {
	return func(mutation *CatalogueMutation) {
		mutation.EnumName = typeName
	}
}
