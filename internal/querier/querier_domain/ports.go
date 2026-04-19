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

package querier_domain

// Defines port interfaces for hexagonal architecture, establishing contracts
// between the domain and external adapters. Includes interfaces for SQL engine
// parsing, code emission, and the main querier service for dependency inversion.

import (
	"context"
	"os"

	"piko.sh/piko/internal/querier/querier_dto"
)

// QuerierServicePort defines the driving port for SQL analysis and code
// generation.
type QuerierServicePort interface {
	// GenerateDatabase analyses migration and query files for a named database
	// connection and generates Go code into the output directory.
	//
	// Takes name (string) which identifies the database connection (e.g. "main").
	// Takes config (*querier_dto.DatabaseConfig) which specifies the engine,
	// migration paths, query paths, and type overrides.
	// Takes outputDirectory (string) which is the target directory for generated
	// Go files (typically dist/databases/{name}/).
	//
	// Returns *querier_dto.GenerationResult which contains the generated files
	// and any diagnostics.
	// Returns error when generation fails fatally.
	GenerateDatabase(ctx context.Context, name string, config *querier_dto.DatabaseConfig, outputDirectory string) (*querier_dto.GenerationResult, error)
}

// EngineParserPort provides SQL parsing and DDL/DML analysis. Engine adapters
// implement this to parse dialect-specific SQL into the domain's neutral IR.
type EngineParserPort interface {
	// ParseStatements parses a SQL string into a sequence of parsed statements
	// using the engine's native parser.
	//
	// Takes sql (string) which is the raw SQL text to parse.
	//
	// Returns []querier_dto.ParsedStatement which contains the parsed AST nodes.
	// Returns error when the SQL contains syntax errors.
	ParseStatements(sql string) ([]querier_dto.ParsedStatement, error)

	// ApplyDDL interprets a parsed DDL statement and returns the mutation it
	// applies to the schema catalogue. Non-DDL statements (DML) are ignored
	// and return a nil mutation without error.
	//
	// Takes statement (querier_dto.ParsedStatement) which is a parsed SQL
	// statement from ParseStatements.
	//
	// Returns *querier_dto.CatalogueMutation which describes the schema change,
	// or nil if the statement is not DDL.
	// Returns error when the DDL is malformed or references non-existent objects.
	ApplyDDL(statement querier_dto.ParsedStatement) (*querier_dto.CatalogueMutation, error)

	// AnalyseQuery analyses a DML query against the catalogue, producing a raw
	// analysis result with output columns, parameter references, and scope
	// information. The domain layer performs further type resolution and
	// nullability propagation on this result.
	//
	// Takes catalogue (*querier_dto.Catalogue) which is the current schema state.
	// Takes statement (querier_dto.ParsedStatement) which is the query to analyse.
	//
	// Returns *querier_dto.RawQueryAnalysis which contains unresolved column and
	// parameter references for the domain to type-check.
	// Returns error when the query references non-existent tables or has
	// structural errors.
	AnalyseQuery(catalogue *querier_dto.Catalogue, statement querier_dto.ParsedStatement) (*querier_dto.RawQueryAnalysis, error)
}

// EngineTypeSystemPort provides type normalisation, promotion, and implicit
// casting rules. The type resolver and function resolver depend on this
// narrow interface rather than the full EnginePort.
type EngineTypeSystemPort interface {
	// NormaliseTypeName converts an engine-specific type name and optional
	// modifiers (precision, scale, length) into a structured SQLType.
	//
	// Takes name (string) which is the engine-native type name (e.g. "varchar",
	// "numeric", "int8").
	// Takes modifiers ([]int) which are optional type parameters such as
	// precision, scale, or length.
	//
	// Returns querier_dto.SQLType which is the structured type representation.
	NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType

	// PromoteType selects the wider type when both operands
	// share the same category but differ in precision or
	// width (e.g. int2 vs int8 promotes to int8).
	//
	// Takes left (querier_dto.SQLType) which is the first operand type.
	// Takes right (querier_dto.SQLType) which is the second operand type.
	//
	// Returns querier_dto.SQLType which is the promoted type.
	PromoteType(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType

	// CanImplicitCast reports whether a value of the source
	// category can be implicitly cast to the target category
	// according to the engine's casting rules.
	//
	// Takes from (querier_dto.SQLTypeCategory) which is the source category.
	// Takes to (querier_dto.SQLTypeCategory) which is the target category.
	//
	// Returns bool which is true if the implicit cast is allowed.
	CanImplicitCast(from querier_dto.SQLTypeCategory, to querier_dto.SQLTypeCategory) bool

	// SupportedExpressions returns the bitmask of SQL expression kinds this
	// engine adapter can emit in expression maps. The type resolver validates
	// expressions against these capabilities and emits diagnostics for
	// unsupported patterns.
	//
	// Returns querier_dto.SQLExpressionFeature which is the bitmask of
	// supported expression features.
	SupportedExpressions() querier_dto.SQLExpressionFeature
}

// EngineDirectivePort provides comment and directive syntax configuration.
// The directive parser depends on this narrow interface for parsing SQL
// comment directives.
type EngineDirectivePort interface {
	// CommentStyle returns the comment syntax used by this engine's query
	// files. SQL dialects typically return "--"; non-SQL targets (GraphQL,
	// Cypher) return "#" or "//".
	//
	// Returns querier_dto.CommentStyle which describes the line-comment prefix.
	CommentStyle() querier_dto.CommentStyle

	// SupportedDirectivePrefixes returns the set of parameter
	// reference sigils that may appear in directive comment
	// lines for this engine.
	//
	// Returns []querier_dto.DirectiveParameterPrefix which lists the valid
	// prefix styles.
	SupportedDirectivePrefixes() []querier_dto.DirectiveParameterPrefix
}

// EngineCataloguePort provides built-in catalogues and schema metadata.
// Components that need access to the engine's default schema, built-in functions,
// types, or table-valued functions depend on the port.
type EngineCataloguePort interface {
	// DefaultSchema returns the default schema name for this engine dialect.
	// PostgreSQL uses "public", SQLite uses "main", MySQL uses "" (empty).
	//
	// Returns string which is the default schema name.
	DefaultSchema() string

	// BuiltinFunctions returns the engine's catalogue of built-in functions
	// with their signatures, overloads, return types, and nullability behaviour.
	//
	// Returns *querier_dto.FunctionCatalogue which maps function names to their
	// overloaded signatures.
	BuiltinFunctions() *querier_dto.FunctionCatalogue

	// BuiltinTypes returns the engine's catalogue of built-in SQL types,
	// mapping engine-specific type names to structured SQLType values.
	//
	// Returns *querier_dto.TypeCatalogue which maps type names to SQLType.
	BuiltinTypes() *querier_dto.TypeCatalogue

	// TableValuedFunctionColumns returns the output column schema for a known
	// table-valued function (e.g. json_each, generate_series), or nil if the
	// function is not recognised.
	//
	// Returns []querier_dto.ScopedColumn which holds the output columns, or
	// nil for unknown functions.
	TableValuedFunctionColumns(functionName string) []querier_dto.ScopedColumn
}

// EngineMetadataPort provides engine identity and capability flags consumed
// by external adapters such as the code emitter.
type EngineMetadataPort interface {
	// ParameterStyle returns the parameter placeholder style for this engine.
	// PostgreSQL uses $1, $2; MySQL and SQLite use ?.
	//
	// Returns querier_dto.ParameterStyle which identifies the placeholder format.
	ParameterStyle() querier_dto.ParameterStyle

	// SupportsReturning reports whether the engine supports RETURNING clauses
	// on INSERT, UPDATE, and DELETE statements.
	//
	// Returns bool which is true if RETURNING is supported.
	SupportsReturning() bool

	// Dialect returns an opaque string identifying this
	// engine adapter (e.g. "postgres", "sqlite", "mysql").
	Dialect() string
}

// EnginePort defines the aggregate inbound adapter contract
// for SQL dialect parsers, embedding all engine
// sub-interfaces.
type EnginePort interface {
	EngineParserPort

	EngineTypeSystemPort

	EngineDirectivePort

	EngineCataloguePort

	EngineMetadataPort
}

// CatalogueFunctionResolverPort is an optional extension
// interface for resolving user-defined functions returning
// composite or set-of types from the catalogue.
type CatalogueFunctionResolverPort interface {
	// TableValuedFunctionColumnsFromCatalogue resolves the output columns of
	// a user-defined table-valued function by looking up its signature and
	// return type in the catalogue.
	//
	// Returns nil when the function is not found or its return type cannot
	// be expanded into columns.
	TableValuedFunctionColumnsFromCatalogue(
		catalogue *querier_dto.Catalogue,
		functionName string,
	) []querier_dto.ScopedColumn
}

// FunctionResolverPort is an optional extension interface that engine adapters
// can implement to handle context-dependent or polymorphic function resolution
// that the standard overload resolution cannot match. Examples include
// BigQuery's ANY_VALUE, Oracle's DECODE, and polymorphic functions like
// ARRAY_AGG where the return type depends on the argument types.
type FunctionResolverPort interface {
	// ResolveFunctionCall resolves a function call that the standard overload
	// resolution could not match. The engine can inspect the argument types
	// and catalogue context to compute a dynamic return type.
	//
	// Takes catalogue (*querier_dto.Catalogue) for schema context.
	// Takes name (string) which is the function name.
	// Takes schema (string) which is the function schema, if specified.
	// Takes argumentTypes ([]querier_dto.SQLType) which are the resolved
	// argument types.
	//
	// Returns *querier_dto.FunctionResolution with the resolved result.
	// Returns error when the function cannot be resolved.
	ResolveFunctionCall(
		catalogue *querier_dto.Catalogue,
		name string,
		schema string,
		argumentTypes []querier_dto.SQLType,
	) (*querier_dto.FunctionResolution, error)
}

// MultiStatementAnalyserPort is an optional extension interface that engine adapters
// can implement to handle multi-statement query blocks where earlier statements
// provide setup (temp tables, variable assignments) for the primary query. If an
// engine does not implement the extension, only the last statement in the block is
// analysed.
type MultiStatementAnalyserPort interface {
	// AnalyseMultiStatement analyses a sequence of statements as a single
	// logical query block, accumulating scope from setup statements into the
	// final analysis result.
	//
	// Takes catalogue (*querier_dto.Catalogue) which is the current schema.
	// Takes statements ([]querier_dto.ParsedStatement) which are all parsed
	// statements in the block.
	//
	// Returns *querier_dto.RawQueryAnalysis for the primary (last) statement.
	// Returns error when analysis fails.
	AnalyseMultiStatement(
		catalogue *querier_dto.Catalogue,
		statements []querier_dto.ParsedStatement,
	) (*querier_dto.RawQueryAnalysis, error)
}

// ExtensionLoaderPort is an optional extension interface
// for providing function definitions for SQL extensions
// (e.g. CREATE EXTENSION pgcrypto).
type ExtensionLoaderPort interface {
	// LoadExtensionFunctions returns function signatures provided by the named
	// extension. Returns nil if the extension is unknown.
	LoadExtensionFunctions(name string) []*querier_dto.FunctionSignature
}

// CatalogueProviderPort defines the contract for building a schema catalogue.
// The default implementation replays migration DDL files, but alternative
// implementations can introspect live databases, read declarative schemas,
// or use any other mechanism appropriate for the target engine.
type CatalogueProviderPort interface {
	// BuildCatalogue constructs the schema catalogue from whatever source
	// this provider uses (migration files, database introspection, etc.).
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *querier_dto.Catalogue which is the built schema state.
	// Returns []querier_dto.SourceError which contains any diagnostics.
	// Returns error when catalogue building fails fatally.
	BuildCatalogue(ctx context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error)
}

// CodeEmitterPort defines the outbound adapter contract for
// generating source code from analysed queries.
type CodeEmitterPort interface {
	// EmitModels generates source code for table model structs and enum types
	// derived from the schema catalogue.
	//
	// Takes packageName (string) which is the target package name for the
	// generated code.
	// Takes catalogue (*querier_dto.Catalogue) which is the schema state.
	// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go
	// type mappings.
	//
	// Returns []querier_dto.GeneratedFile which contains the model source files.
	// Returns error when code emission fails.
	EmitModels(packageName string, catalogue *querier_dto.Catalogue, mappings *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error)

	// EmitQueries generates source code for query methods, parameter structs,
	// and result structs from analysed queries.
	//
	// Takes packageName (string) which is the target package name for the
	// generated code.
	// Takes queries ([]*querier_dto.AnalysedQuery) which are the type-checked
	// queries.
	// Takes mappings (*querier_dto.TypeMappingTable) which defines SQL-to-Go
	// type mappings.
	//
	// Returns []querier_dto.GeneratedFile which contains the query source files.
	// Returns error when code emission fails.
	EmitQueries(packageName string, queries []*querier_dto.AnalysedQuery, mappings *querier_dto.TypeMappingTable) ([]querier_dto.GeneratedFile, error)

	// EmitQuerier generates the top-level Queries struct, constructor, WithTx
	// helper, and DBTX interface. The exact shape of the generated scaffold
	// depends on the emitter implementation (database/sql vs pgx vs custom).
	//
	// Takes packageName (string) which is the Go package name for the generated
	// code.
	// Takes capabilities (querier_dto.QueryCapabilities) which is reserved for
	// future use and may be ignored by emitter implementations.
	//
	// Returns querier_dto.GeneratedFile which contains the querier source file.
	// Returns error when code emission fails.
	EmitQuerier(packageName string, capabilities querier_dto.QueryCapabilities) (querier_dto.GeneratedFile, error)

	// EmitPrepared generates the PreparedDBTX wrapper with eager preparation
	// of static queries and lazy caching of dynamic query variants.
	//
	// Takes packageName (string) which is the target package name.
	// Takes queries ([]*querier_dto.AnalysedQuery) which are the analysed
	// queries used to determine which SQL constants to eagerly prepare.
	//
	// Returns querier_dto.GeneratedFile which contains the prepared.go source.
	// Returns error when code emission fails.
	EmitPrepared(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error)

	// EmitOTel generates the otel.go file containing the QueryNameResolver
	// function that maps SQL query constant text to human-readable operation
	// names for OpenTelemetry instrumentation.
	//
	// Takes packageName (string) which is the target package name.
	// Takes queries ([]*querier_dto.AnalysedQuery) which provide the query
	// names and their SQL constants.
	//
	// Returns querier_dto.GeneratedFile which contains the otel.go source.
	// Returns error when code emission fails.
	EmitOTel(packageName string, queries []*querier_dto.AnalysedQuery) (querier_dto.GeneratedFile, error)
}

// FileReaderPort abstracts filesystem access for reading migration and query
// SQL files. This allows the domain to remain independent of the filesystem
// and enables testing with in-memory file systems.
type FileReaderPort interface {
	// ReadFile reads the contents of a file at the given path.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes path (string) which is the file path to read.
	//
	// Returns []byte which contains the file contents.
	// Returns error when the file cannot be read.
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// ReadDir reads the directory entries, sorted by name.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes directory (string) which is the directory path to read.
	//
	// Returns []os.DirEntry which contains the directory entries sorted by name.
	// Returns error when the directory cannot be read.
	ReadDir(ctx context.Context, directory string) ([]os.DirEntry, error)
}

// MigrationServicePort defines the driving port for database migration
// operations. It provides methods for applying, rolling back, and inspecting
// migration state.
type MigrationServicePort interface {
	// Up applies all pending up migrations in version order. Returns the
	// number of migrations applied.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns int which is the number of migrations applied.
	// Returns error when a migration fails to apply or checksum validation
	// fails.
	Up(ctx context.Context) (int, error)

	// Down rolls back the last n applied migrations in reverse version
	// order. Returns the number of migrations rolled back.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes steps (int) which is the number of migrations to roll back.
	//
	// Returns int which is the number of migrations rolled back.
	// Returns error when a rollback fails or no down migration exists.
	Down(ctx context.Context, steps int) (int, error)

	// Status returns the list of all known migrations and their applied
	// state, combining on-disk files with the migration history table.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []querier_dto.MigrationStatus which lists all migrations with
	// their applied state, checksum match status, and down-migration
	// availability.
	// Returns error when the status cannot be determined.
	Status(ctx context.Context) ([]querier_dto.MigrationStatus, error)

	// UpTo applies pending up migrations up to and including the target
	// version. Returns the number of migrations applied.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes targetVersion (int64) which is the maximum version to apply.
	//
	// Returns int which is the number of migrations applied.
	// Returns error when a migration fails to apply or checksum validation
	// fails.
	UpTo(ctx context.Context, targetVersion int64) (int, error)

	// DownTo rolls back applied migrations down to (but not including) the
	// target version. Migrations with versions greater than targetVersion
	// are rolled back in reverse order.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes targetVersion (int64) which is the version to roll back to.
	//
	// Returns int which is the number of migrations rolled back.
	// Returns error when a rollback fails or no down migration exists.
	DownTo(ctx context.Context, targetVersion int64) (int, error)

	// Validates that all applied migration checksums match their
	// on-disk files without executing anything.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when any checksum mismatch is detected or files are
	// missing.
	Validate(ctx context.Context) error
}

// MigrationExecutorPort defines the database-specific operations needed by
// the migration service. Each engine adapter (SQLite, PostgreSQL) provides
// an implementation handling locking, transaction wrapping, and SQL execution
// against the target database.
type MigrationExecutorPort interface {
	// EnsureMigrationTable creates the migration history table if it does
	// not exist.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when the table cannot be created.
	EnsureMigrationTable(ctx context.Context) error

	// AcquireLock acquires an advisory lock to prevent
	// concurrent migration runs.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when the lock cannot be acquired.
	AcquireLock(ctx context.Context) error

	// TryAcquireLock attempts to acquire the advisory lock without blocking.
	// Returns querier_domain.ErrLockNotAcquired immediately if the lock is
	// already held by another process.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when the lock cannot be acquired or is already held.
	TryAcquireLock(ctx context.Context) error

	// ReleaseLock releases the advisory lock acquired by AcquireLock.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when the lock cannot be released.
	ReleaseLock(ctx context.Context) error

	// AppliedVersions returns all migration versions that have been applied,
	// ordered by version number ascending.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []querier_dto.AppliedMigration which lists all applied
	// migrations in version order.
	// Returns error when the history cannot be read.
	AppliedVersions(ctx context.Context) ([]querier_dto.AppliedMigration, error)

	// ExecuteMigration runs a single migration's SQL content
	// and updates the history table accordingly.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes migration (querier_dto.MigrationRecord) which holds the version,
	// name, SQL content, and checksum.
	// Takes direction (querier_dto.MigrationDirection) which indicates
	// whether this is an up or down migration.
	// Takes useTransaction (bool) which controls whether the migration runs
	// inside a database transaction.
	//
	// Returns error when the migration fails to execute.
	ExecuteMigration(
		ctx context.Context,
		migration querier_dto.MigrationRecord,
		direction querier_dto.MigrationDirection,
		useTransaction bool,
	) error
}

// SeedServicePort defines the driving port for database seed operations.
// Seeds are forward-only SQL files applied after migrations, tracked in a
// history table for idempotency.
type SeedServicePort interface {
	// Apply executes all pending seed files in version order, skipping those
	// already applied and warning on checksum mismatches.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns int which is the number of seeds applied.
	// Returns error when a seed fails to execute.
	Apply(ctx context.Context) (int, error)

	// Status returns the list of all known seeds and their applied state,
	// combining on-disk files with the seed history table.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []querier_dto.SeedStatus which lists all seeds with their
	// applied state and checksum match status.
	// Returns error when the status cannot be determined.
	Status(ctx context.Context) ([]querier_dto.SeedStatus, error)

	// Reseed clears the seed history table and re-applies all seed files.
	// This is useful for development resets where seed data needs to be
	// refreshed from scratch.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns int which is the number of seeds applied.
	// Returns error when clearing history or applying seeds fails.
	Reseed(ctx context.Context) (int, error)
}

// SeedExecutorPort defines the database-specific operations needed by the
// seed service. Unlike MigrationExecutorPort, seeds require no advisory
// locking, dirty state tracking, or rollback support.
type SeedExecutorPort interface {
	// EnsureSeedTable creates the piko_seeds history table if it does not
	// exist.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when the table cannot be created.
	EnsureSeedTable(ctx context.Context) error

	// AppliedSeeds returns all seeds that have been applied, ordered by
	// version ascending.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns []querier_dto.AppliedSeed which lists all applied seeds.
	// Returns error when the history cannot be read.
	AppliedSeeds(ctx context.Context) ([]querier_dto.AppliedSeed, error)

	// ExecuteSeed runs a single seed's SQL content in a transaction and
	// records it in the history table.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	// Takes seed (querier_dto.SeedRecord) which holds the version, name,
	// SQL content, and checksum.
	//
	// Returns error when the seed fails to execute.
	ExecuteSeed(ctx context.Context, seed querier_dto.SeedRecord) error

	// ClearSeedHistory removes all records from the piko_seeds table,
	// allowing seeds to be re-applied.
	//
	// Takes ctx (context.Context) for cancellation and timeout control.
	//
	// Returns error when the history cannot be cleared.
	ClearSeedHistory(ctx context.Context) error
}
