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

package db

import (
	"database/sql"
	"io/fs"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/wdk/dbschema"
)

// Schema types, aliased from dbschema.
type (
	// MigrationService defines the driving port for database migration operations. It
	// provides methods for applying, rolling back, and inspecting migration state.
	MigrationService = dbschema.MigrationService

	// MigrationExecutor defines the database-specific operations needed by the migration
	// service.
	MigrationExecutor = dbschema.MigrationExecutor

	// EnginePort defines the aggregate adapter contract for SQL dialect parsers.
	EnginePort = dbschema.EnginePort

	// CatalogueProviderPort defines the contract for building a schema catalogue from a live
	// database.
	CatalogueProviderPort = dbschema.CatalogueProviderPort

	// FileReaderPort abstracts filesystem access for reading migration and query SQL files.
	FileReaderPort = dbschema.FileReaderPort

	// QuerierServicePort is the SQL code-generation service returned by NewQuerierService.
	QuerierServicePort = dbschema.QuerierServicePort

	// QuerierService is the driving port for the querier code generator. It turns SQL
	// migration + query files into Go source code.
	QuerierService = dbschema.QuerierService

	// QuerierPorts groups the adapter ports a QuerierService needs at construction time.
	// Pass it to NewQuerierService.
	QuerierPorts = dbschema.QuerierPorts

	// CodeEmitterPort is the outbound adapter that turns analysed queries into emitted
	// source files.
	CodeEmitterPort = dbschema.CodeEmitterPort

	// GenerationResult holds the output of one QuerierService.GenerateDatabase call: the
	// emitted files, plus any diagnostics produced along the way.
	GenerationResult = dbschema.GenerationResult

	// GeneratedFile is one file in GenerationResult.Files. Callers typically write Content
	// to disk under their chosen output directory.
	GeneratedFile = dbschema.GeneratedFile

	// SourceError is one diagnostic emitted during catalogue building or query analysis.
	// Diagnostics run from SeverityHint, which is worth knowing, through SeverityWarning to
	// SeverityError, which stops generation.
	SourceError = dbschema.SourceError

	// Severity classifies a SourceError. Callers gate on SeverityError to decide whether to
	// fail a code-generation run.
	Severity = dbschema.Severity

	// CustomFunctionConfig describes a user-declared function signature that the analyser
	// should recognise during query type resolution. Use this for SQLite extensions or for
	// functions defined outside the migration files Piko parses.
	CustomFunctionConfig = dbschema.CustomFunctionConfig

	// SeedService defines the driving port for database seed operations.
	SeedService = dbschema.SeedService

	// SeedExecutorPort defines the database-specific operations needed by the seed service.
	SeedExecutorPort = dbschema.SeedExecutorPort

	// MigrationServiceOption configures optional behaviour of the migration service.
	MigrationServiceOption = dbschema.MigrationServiceOption

	// BeforeMigrationHook is called before each individual migration executes.
	BeforeMigrationHook = dbschema.BeforeMigrationHook

	// AfterMigrationHook is called after each individual migration executes successfully.
	AfterMigrationHook = dbschema.AfterMigrationHook

	// BeforeRunHook is called before the migration run begins.
	BeforeRunHook = dbschema.BeforeRunHook

	// AfterRunHook is called after the migration run completes successfully.
	AfterRunHook = dbschema.AfterRunHook

	// MigrationHookContext provides information about an individual migration being
	// processed.
	MigrationHookContext = dbschema.MigrationHookContext

	// MigrationRunHookContext provides information about an entire migration run before it
	// begins.
	MigrationRunHookContext = dbschema.MigrationRunHookContext

	// MigrationDirection indicates whether a migration is a forward (up) or rollback (down)
	// migration.
	MigrationDirection = dbschema.MigrationDirection

	// MigrationStatus combines a migration file with its applied state.
	MigrationStatus = dbschema.MigrationStatus

	// MigrationFile represents a parsed migration file with version, direction, and content.
	MigrationFile = dbschema.MigrationFile

	// AppliedMigration represents a migration that has been applied to the database.
	AppliedMigration = dbschema.AppliedMigration

	// SeedStatus combines a seed file with its applied state.
	SeedStatus = dbschema.SeedStatus

	// AppliedSeed represents a seed that has been applied to the database.
	AppliedSeed = dbschema.AppliedSeed

	// DatabaseConfig is the configuration container for code generation.
	DatabaseConfig = dbschema.DatabaseConfig

	// TypeOverride maps a SQL type to a Go type for code generation.
	TypeOverride = dbschema.TypeOverride

	// CustomFunction defines a custom SQL function for code generation.
	CustomFunction = dbschema.CustomFunction

	// DialectConfig holds dialect-specific SQL and behaviour for the migration executor.
	DialectConfig = dbschema.DialectConfig

	// ChecksumMismatchError is returned when an applied migration's recorded checksum does
	// not match the current file on disk.
	ChecksumMismatchError = dbschema.ChecksumMismatchError

	// DownChecksumMismatchError is returned when a down migration file's checksum does not
	// match the checksum recorded when the up migration was applied.
	DownChecksumMismatchError = dbschema.DownChecksumMismatchError

	// MigrationExecutionError wraps an error from executing a migration's SQL content.
	MigrationExecutionError = dbschema.MigrationExecutionError

	// LockAcquisitionError wraps a failure to acquire the migration advisory lock.
	LockAcquisitionError = dbschema.LockAcquisitionError

	// MissingMigrationFileError is returned when the database records an applied migration
	// but no corresponding file exists on disk.
	MissingMigrationFileError = dbschema.MissingMigrationFileError

	// NoDownMigrationError is returned when a rollback is requested for a version that has
	// no .down.sql file.
	NoDownMigrationError = dbschema.NoDownMigrationError
)

const (
	// SeverityHint flags a non-fatal suggestion that does not block generation.
	SeverityHint = dbschema.SeverityHint

	// SeverityWarning flags a diagnostic that highlights a likely problem but does not block
	// generation.
	SeverityWarning = dbschema.SeverityWarning

	// SeverityError flags a diagnostic that prevents successful generation; callers should
	// fail when one is reported.
	SeverityError = dbschema.SeverityError

	// DirectionUp is a forward migration that applies schema changes.
	DirectionUp = dbschema.DirectionUp

	// DirectionDown is a rollback migration that reverts schema changes.
	DirectionDown = dbschema.DirectionDown
)

const (
	// DatabaseNameRegistry is the reserved database name for piko's internal registry
	// subsystem. Register a database with this name to back the registry with a SQL database
	// instead of the default otter in-memory backend.
	DatabaseNameRegistry = bootstrap.DatabaseNameRegistry

	// DatabaseNameOrchestrator is the reserved database name for piko's internal
	// orchestrator subsystem. Register a database with this name to back the orchestrator
	// with a SQL database instead of the default otter in-memory backend.
	DatabaseNameOrchestrator = bootstrap.DatabaseNameOrchestrator
)

var (
	// ErrLockNotAcquired is returned when a non-blocking lock attempt fails because another
	// process already holds the migration lock.
	ErrLockNotAcquired = dbschema.ErrLockNotAcquired
)

// NewSQLEmitter returns the database/sql code emitter. This is the default emitter for
// projects that use the standard library database abstraction; pgx-based projects can
// import a dedicated emitter from wdk/db/db_emitter_pgx instead.
//
// Returns CodeEmitterPort which is ready to inject into a QuerierPorts struct.
func NewSQLEmitter() CodeEmitterPort {
	return dbschema.NewSQLEmitter()
}

// NewSQLEmitterForDialect returns the database/sql code emitter for an engine dialect.
//
// The emitter is configured so the generated placeholders and array decoding match the
// engine's driver. A project generating for more than one engine should pass each
// engine's Dialect() here rather than using NewSQLEmitter, otherwise the postgres family
// loses its `$N` placeholder form and its to_json array-column auto-decoding.
//
// Takes dialect (string) which is the engine's Dialect() name.
//
// Returns CodeEmitterPort which is ready to inject into a QuerierPorts struct.
func NewSQLEmitterForDialect(dialect string) CodeEmitterPort {
	return dbschema.NewSQLEmitterForDialect(dialect)
}

// NewQuerierService constructs a QuerierService from the supplied adapter ports.
//
// Takes ports (QuerierPorts) which provides the engine, emitter, file reader, and
// optional catalogue provider.
//
// Returns QuerierService which is ready to run GenerateDatabase.
// Returns error when any required port is missing.
func NewQuerierService(ports QuerierPorts) (QuerierService, error) {
	return dbschema.NewQuerierService(ports)
}

// NewCompositeCatalogueProvider wires a CatalogueProviderPort that union-merges the
// catalogues produced by each upstream provider in order. Use it when query files cross
// schemas or hexagons and need every upstream schema visible at analysis time.
//
// Takes providers ([]CatalogueProviderPort) which is the apply-order chain. Nil and empty
// slices return an empty catalogue.
//
// Returns CatalogueProviderPort which is ready to plug into a
// QuerierPorts.CatalogueProvider.
func NewCompositeCatalogueProvider(providers []CatalogueProviderPort) CatalogueProviderPort {
	return dbschema.NewCompositeCatalogueProvider(providers)
}

// PostgresDialect returns a DialectConfig for PostgreSQL databases.
//
// Returns DialectConfig which carries PostgreSQL SQL and locking behaviour.
func PostgresDialect() DialectConfig {
	return dbschema.PostgresDialect()
}

// PostgresPgBouncerDialect returns a DialectConfig for PostgreSQL via PgBouncer.
//
// Configured for PgBouncer transaction mode, using table-based locking instead of
// advisory locks.
//
// Returns DialectConfig which carries PgBouncer-compatible locking behaviour.
func PostgresPgBouncerDialect() DialectConfig {
	return dbschema.PostgresPgBouncerDialect()
}

// MySQLDialect returns a DialectConfig for MySQL databases.
//
// Returns DialectConfig which carries MySQL SQL and locking behaviour.
func MySQLDialect() DialectConfig {
	return dbschema.MySQLDialect()
}

// MySQLDialectWithDSN returns a DialectConfig for MySQL databases.
//
// Detects whether the DSN includes multiStatements=true. When the driver handles
// multi-statement execution natively, statement splitting is disabled.
//
// Takes dsn (string) which is the MySQL DSN used to probe driver capabilities.
//
// Returns DialectConfig which carries MySQL SQL and locking behaviour tuned to the DSN.
func MySQLDialectWithDSN(dsn string) DialectConfig {
	return dbschema.MySQLDialectWithDSN(dsn)
}

// SQLiteDialect returns a DialectConfig for SQLite databases.
//
// Returns DialectConfig which carries SQLite SQL and locking behaviour.
func SQLiteDialect() DialectConfig {
	return dbschema.SQLiteDialect()
}

// NewMigrationExecutor creates a SQL-based migration executor from a database connection
// and dialect configuration.
//
// Takes database (*sql.DB) which is the database connection to execute migrations
// against.
// Takes dialect (DialectConfig) which provides dialect-specific SQL and locking
// behaviour.
//
// Returns MigrationExecutor which is ready to execute migrations.
func NewMigrationExecutor(database *sql.DB, dialect DialectConfig) MigrationExecutor {
	return dbschema.NewMigrationExecutor(database, dialect)
}

// NewFSFileReader creates a file reader backed by an fs.FS. This is typically used with
// embed.FS for embedding migration files into the binary, or with os.DirFS for reading
// from the local filesystem.
//
// Takes filesystem (fs.FS) which is the filesystem to read migration files from.
//
// Returns FileReaderPort which is ready to read files.
func NewFSFileReader(filesystem fs.FS) FileReaderPort {
	return dbschema.NewFSFileReader(filesystem)
}

// NewSeedExecutor creates a SQL-based seed executor from a database connection and
// dialect configuration.
//
// Takes database (*sql.DB) which is the database connection to execute seeds against.
// Takes dialect (DialectConfig) which provides dialect-specific SQL.
//
// Returns SeedExecutorPort which is ready to execute seeds.
func NewSeedExecutor(database *sql.DB, dialect DialectConfig) SeedExecutorPort {
	return dbschema.NewSeedExecutor(database, dialect)
}

// WithNonBlockingLock configures non-blocking lock acquisition.
//
// If the lock is already held, operations return ErrLockNotAcquired immediately instead
// of waiting.
//
// Returns MigrationServiceOption which enables non-blocking lock behaviour.
func WithNonBlockingLock() MigrationServiceOption {
	return dbschema.WithNonBlockingLock()
}

// WithBeforeMigration registers a hook that runs before each migration.
//
// Takes hook (BeforeMigrationHook) which observes the upcoming migration.
//
// Returns MigrationServiceOption which installs the before-migration hook.
func WithBeforeMigration(hook BeforeMigrationHook) MigrationServiceOption {
	return dbschema.WithBeforeMigration(hook)
}

// WithAfterMigration registers a hook that runs after each migration succeeds.
//
// Takes hook (AfterMigrationHook) which observes the completed migration.
//
// Returns MigrationServiceOption which installs the after-migration hook.
func WithAfterMigration(hook AfterMigrationHook) MigrationServiceOption {
	return dbschema.WithAfterMigration(hook)
}

// WithBeforeRun registers a hook that runs before the migration run begins.
//
// Takes hook (BeforeRunHook) which observes the migration run context.
//
// Returns MigrationServiceOption which installs the before-run hook.
func WithBeforeRun(hook BeforeRunHook) MigrationServiceOption {
	return dbschema.WithBeforeRun(hook)
}

// WithAfterRun registers a hook that runs after the migration run completes.
//
// Takes hook (AfterRunHook) which observes the migration run context.
//
// Returns MigrationServiceOption which installs the after-run hook.
func WithAfterRun(hook AfterRunHook) MigrationServiceOption {
	return dbschema.WithAfterRun(hook)
}

// NewMigrationCatalogueProvider returns the default catalogue provider, which builds a
// schema catalogue by replaying migration DDL files.
//
// Takes engine (EnginePort) which parses and interprets DDL.
// Takes fileReader (FileReaderPort) which reads files from disk or an embed.FS.
// Takes directory (string) which is the migrations directory path.
//
// Returns CatalogueProviderPort which is ready to plug into a
// QuerierPorts.CatalogueProvider.
func NewMigrationCatalogueProvider(
	engine EnginePort,
	fileReader FileReaderPort,
	directory string,
) CatalogueProviderPort {
	return dbschema.NewMigrationCatalogueProvider(engine, fileReader, directory)
}

// NewMigrationService creates a migration service for executing database migrations. The
// service handles applying, rolling back, and inspecting migration state with advisory
// locking and checksum verification.
//
// Takes executor (MigrationExecutor) which provides database-specific migration
// operations.
// Takes fileReader (FileReaderPort) which provides filesystem access for reading
// migration SQL files.
// Takes directory (string) which is the path to the migration files within the
// filesystem.
// Takes opts (...MigrationServiceOption) which configure optional behaviour such as
// non-blocking lock acquisition and lifecycle hooks.
//
// Returns MigrationService which is ready to apply or roll back migrations.
func NewMigrationService(
	executor MigrationExecutor,
	fileReader FileReaderPort,
	directory string,
	opts ...MigrationServiceOption,
) MigrationService {
	return dbschema.NewMigrationService(executor, fileReader, directory, opts...)
}

// NewSeedService creates a seed service for applying database seed files. The service
// handles executing SQL seed files in version order with idempotency tracking via a
// history table.
//
// Takes executor (SeedExecutorPort) which provides database-specific seed operations.
// Takes fileReader (FileReaderPort) which provides filesystem access for reading seed SQL
// files.
// Takes directory (string) which is the path to the seed files within the filesystem.
//
// Returns SeedService which is ready to apply seeds.
func NewSeedService(
	executor SeedExecutorPort,
	fileReader FileReaderPort,
	directory string,
) SeedService {
	return dbschema.NewSeedService(executor, fileReader, directory)
}

// DBTX is the common database interface for read and write operations. It matches the
// interface generated by the querier code generator and is satisfied by *sql.DB and
// *sql.Tx.
type DBTX = bootstrap.DBTX

// Replica configures a single read replica connection.
type Replica = bootstrap.Replica

// DatabaseRegistration holds the configuration for a named database connection registered
// during bootstrap. See bootstrap.DatabaseRegistration for field documentation.
type DatabaseRegistration = bootstrap.DatabaseRegistration

// WithDatabase returns a bootstrap option that registers a named database connection.
// When the name is DatabaseNameRegistry or DatabaseNameOrchestrator, the framework uses
// the querier-based DAL adapters instead of the default otter in-memory backend for that
// subsystem.
//
// Takes name (string) which identifies the database.
// Takes registration (*DatabaseRegistration) which provides connection and migration
// configuration.
//
// Returns bootstrap.Option which registers the database with the container.
func WithDatabase(name string, registration *DatabaseRegistration) bootstrap.Option {
	return bootstrap.WithDatabase(name, registration)
}

// GetDatabaseConnection returns the *sql.DB for a named database.
//
// The database must have been registered during bootstrap. Safe for concurrent use from
// multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns *sql.DB which is the open database connection.
// Returns error when the framework is not initialised or the database is not registered.
func GetDatabaseConnection(name string) (*sql.DB, error) {
	return bootstrap.GetDatabaseConnection(name)
}

// GetDatabaseReader returns the DBTX for reading from a named database.
//
// The database must have been registered during bootstrap. When replicas are configured,
// returns a round-robin balancer across them; otherwise returns the primary connection.
// Safe for concurrent use from multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which can execute read queries.
// Returns error when the framework is not initialised or the database is not registered.
func GetDatabaseReader(name string) (DBTX, error) {
	return bootstrap.GetDatabaseReader(name)
}

// GetDatabaseWriter returns the DBTX for writing to a named database.
//
// The database must have been registered during bootstrap. When EnableOTel is set on the
// registration, the returned DBTX is wrapped with OpenTelemetry spans and metrics. Safe
// for concurrent use from multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which can execute write queries.
// Returns error when the framework is not initialised or the database is not registered.
func GetDatabaseWriter(name string) (DBTX, error) {
	return bootstrap.GetDatabaseWriter(name)
}

// GetMigrationService returns the migration service for a named database.
//
// The database must have been registered during bootstrap. Safe for concurrent use from
// multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns MigrationService which can apply and roll back migrations.
// Returns error when the framework is not initialised, the database is not registered, or
// no migration filesystem was configured.
func GetMigrationService(name string) (MigrationService, error) {
	return bootstrap.GetMigrationService(name)
}

// GetSeedService returns the seed service for a named database.
//
// The database must have been registered during bootstrap. Safe for concurrent use from
// multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns SeedService which can apply and inspect seeds.
// Returns error when the framework is not initialised, the database is not registered, or
// no seed filesystem was configured.
func GetSeedService(name string) (SeedService, error) {
	return bootstrap.GetSeedService(name)
}
