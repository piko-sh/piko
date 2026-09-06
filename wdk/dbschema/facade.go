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

package dbschema

import (
	"database/sql"
	"io/fs"

	"piko.sh/piko/internal/querier/querier_adapters/emitter_go_sql"
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// MigrationService defines the driving port for database migration operations. It
// provides methods for applying, rolling back, and inspecting migration state.
type MigrationService = querier_domain.MigrationServicePort

// MigrationExecutor defines the database-specific operations needed by the migration
// service.
type MigrationExecutor = querier_domain.MigrationExecutorPort

// EnginePort defines the aggregate adapter contract for SQL dialect parsers.
type EnginePort = querier_domain.EnginePort

// CatalogueProviderPort defines the contract for building a schema catalogue from a live
// database.
type CatalogueProviderPort = querier_domain.CatalogueProviderPort

// FileReaderPort abstracts filesystem access for reading migration and query SQL files.
type FileReaderPort = querier_domain.FileReaderPort

// QuerierServicePort is the SQL code-generation service returned by NewQuerierService.
type QuerierServicePort = querier_domain.QuerierServicePort

// QuerierService is the driving port for the querier code generator. It turns SQL
// migration + query files into Go source code.
type QuerierService = querier_domain.QuerierServicePort

// QuerierPorts groups the adapter ports a QuerierService needs at construction time. Pass
// it to NewQuerierService.
type QuerierPorts = querier_domain.QuerierPorts

// CodeEmitterPort is the outbound adapter that turns analysed queries into emitted source
// files.
type CodeEmitterPort = querier_domain.CodeEmitterPort

// GenerationResult holds the output of one QuerierService.GenerateDatabase call: the
// emitted files, plus any diagnostics produced along the way.
type GenerationResult = querier_dto.GenerationResult

// GeneratedFile is one file in GenerationResult.Files. Callers typically write Content to
// disk under their chosen output directory.
type GeneratedFile = querier_dto.GeneratedFile

// SourceError is one diagnostic emitted during catalogue building or query analysis.
// Diagnostics run from SeverityHint, which is worth knowing, through SeverityWarning to
// SeverityError, which stops generation.
type SourceError = querier_dto.SourceError

// Severity classifies a SourceError. Callers gate on SeverityError to decide whether to
// fail a code-generation run.
type Severity = querier_dto.ErrorSeverity

// CustomFunctionConfig describes a user-declared function signature that the analyser
// should recognise during query type resolution. Use this for SQLite extensions or for
// functions defined outside the migration files Piko parses.
type CustomFunctionConfig = querier_dto.CustomFunctionConfig

const (
	// SeverityHint flags a non-fatal suggestion that does not block generation.
	SeverityHint = querier_dto.SeverityHint

	// SeverityWarning flags a diagnostic that highlights a likely problem but does not block
	// generation.
	SeverityWarning = querier_dto.SeverityWarning

	// SeverityError flags a diagnostic that prevents successful generation; callers should
	// fail when one is reported.
	SeverityError = querier_dto.SeverityError

	// DirectionUp is a forward migration that applies schema changes.
	DirectionUp = querier_dto.MigrationDirectionUp

	// DirectionDown is a rollback migration that reverts schema changes.
	DirectionDown = querier_dto.MigrationDirectionDown
)

// NewSQLEmitter returns the database/sql code emitter. This is the default emitter for
// projects that use the standard library database abstraction; pgx-based projects can
// import a dedicated emitter from wdk/db/db_emitter_pgx instead.
//
// Returns CodeEmitterPort which is ready to inject into a QuerierPorts struct.
func NewSQLEmitter() CodeEmitterPort {
	return emitter_go_sql.NewSQLEmitter()
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
	return emitter_go_sql.NewSQLEmitterForDialect(dialect)
}

// NewQuerierService constructs a QuerierService from the supplied adapter ports.
//
// Takes ports (QuerierPorts) which provides the engine, emitter, file reader, and
// optional catalogue provider.
//
// Returns QuerierService which is ready to run GenerateDatabase.
// Returns error when any required port is missing.
func NewQuerierService(ports QuerierPorts) (QuerierService, error) {
	return querier_domain.NewQuerierService(ports)
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
	return querier_domain.NewMigrationCatalogueProvider(engine, fileReader, directory)
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
	return querier_domain.NewCompositeCatalogueProvider(providers)
}

// SeedService defines the driving port for database seed operations.
type SeedService = querier_domain.SeedServicePort

// SeedExecutorPort defines the database-specific operations needed by the seed service.
type SeedExecutorPort = querier_domain.SeedExecutorPort

// MigrationServiceOption configures optional behaviour of the migration service.
type MigrationServiceOption = querier_domain.MigrationServiceOption

// BeforeMigrationHook is called before each individual migration executes.
type BeforeMigrationHook = querier_domain.BeforeMigrationHook

// AfterMigrationHook is called after each individual migration executes successfully.
type AfterMigrationHook = querier_domain.AfterMigrationHook

// BeforeRunHook is called before the migration run begins.
type BeforeRunHook = querier_domain.BeforeRunHook

// AfterRunHook is called after the migration run completes successfully.
type AfterRunHook = querier_domain.AfterRunHook

// MigrationHookContext provides information about an individual migration being
// processed.
type MigrationHookContext = querier_domain.MigrationHookContext

// MigrationRunHookContext provides information about an entire migration run before it
// begins.
type MigrationRunHookContext = querier_domain.MigrationRunHookContext

// MigrationDirection indicates whether a migration is a forward (up) or rollback (down)
// migration.
type MigrationDirection = querier_dto.MigrationDirection

// MigrationStatus combines a migration file with its applied state.
type MigrationStatus = querier_dto.MigrationStatus

// MigrationFile represents a parsed migration file with version, direction, and content.
type MigrationFile = querier_dto.MigrationFile

// AppliedMigration represents a migration that has been applied to the database.
type AppliedMigration = querier_dto.AppliedMigration

// SeedStatus combines a seed file with its applied state.
type SeedStatus = querier_dto.SeedStatus

// AppliedSeed represents a seed that has been applied to the database.
type AppliedSeed = querier_dto.AppliedSeed

// DatabaseConfig is the configuration container for code generation.
type DatabaseConfig = querier_dto.DatabaseConfig

// TypeOverride maps a SQL type to a Go type for code generation.
type TypeOverride = querier_dto.TypeOverride

// CustomFunction defines a custom SQL function for code generation.
type CustomFunction = querier_dto.CustomFunctionConfig

// DialectConfig holds dialect-specific SQL and behaviour for the migration executor.
type DialectConfig = migration_sql.DialectConfig

// ChecksumMismatchError is returned when an applied migration's recorded checksum does
// not match the current file on disk.
type ChecksumMismatchError = querier_domain.ChecksumMismatchError

// DownChecksumMismatchError is returned when a down migration file's checksum does not
// match the checksum recorded when the up migration was applied.
type DownChecksumMismatchError = querier_domain.DownChecksumMismatchError

// MigrationExecutionError wraps an error from executing a migration's SQL content.
type MigrationExecutionError = querier_domain.MigrationExecutionError

// LockAcquisitionError wraps a failure to acquire the migration advisory lock.
type LockAcquisitionError = querier_domain.LockAcquisitionError

// MissingMigrationFileError is returned when the database records an applied migration
// but no corresponding file exists on disk.
type MissingMigrationFileError = querier_domain.MissingMigrationFileError

// NoDownMigrationError is returned when a rollback is requested for a version that has no
// .down.sql file.
type NoDownMigrationError = querier_domain.NoDownMigrationError

var (
	// ErrLockNotAcquired is returned when a non-blocking lock attempt fails because another
	// process already holds the migration lock.
	ErrLockNotAcquired = querier_domain.ErrLockNotAcquired
)

// PostgresDialect returns a DialectConfig for PostgreSQL databases.
//
// Returns DialectConfig which carries PostgreSQL SQL and locking behaviour.
func PostgresDialect() DialectConfig { return migration_sql.PostgresDialect() }

// PostgresPgBouncerDialect returns a DialectConfig for PostgreSQL via PgBouncer.
//
// Configured for PgBouncer transaction mode, using table-based locking instead of
// advisory locks.
//
// Returns DialectConfig which carries PgBouncer-compatible locking behaviour.
func PostgresPgBouncerDialect() DialectConfig { return migration_sql.PostgresPgBouncerDialect() }

// MySQLDialect returns a DialectConfig for MySQL databases.
//
// Returns DialectConfig which carries MySQL SQL and locking behaviour.
func MySQLDialect() DialectConfig { return migration_sql.MySQLDialect() }

// MySQLDialectWithDSN returns a DialectConfig for MySQL databases.
//
// Detects whether the DSN includes multiStatements=true. When the driver handles
// multi-statement execution natively, statement splitting is disabled.
//
// Takes dsn (string) which is the MySQL DSN used to probe driver capabilities.
//
// Returns DialectConfig which carries MySQL SQL and locking behaviour tuned to the DSN.
func MySQLDialectWithDSN(dsn string) DialectConfig {
	return migration_sql.MySQLDialectWithDSN(dsn)
}

// SQLiteDialect returns a DialectConfig for SQLite databases.
//
// Returns DialectConfig which carries SQLite SQL and locking behaviour.
func SQLiteDialect() DialectConfig { return migration_sql.SQLiteDialect() }

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
	return querier_domain.NewMigrationService(executor, fileReader, directory, opts...)
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
	return migration_sql.NewExecutor(database, dialect)
}

// NewFSFileReader creates a file reader backed by an fs.FS. This is typically used with
// embed.FS for embedding migration files into the binary, or with os.DirFS for reading
// from the local filesystem.
//
// Takes filesystem (fs.FS) which is the filesystem to read migration files from.
//
// Returns FileReaderPort which is ready to read files.
func NewFSFileReader(filesystem fs.FS) FileReaderPort {
	return migration_sql.NewFSFileReader(filesystem)
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
	return querier_domain.NewSeedService(executor, fileReader, directory)
}

// NewSeedExecutor creates a SQL-based seed executor from a database connection and
// dialect configuration.
//
// Takes database (*sql.DB) which is the database connection to execute seeds against.
// Takes dialect (DialectConfig) which provides dialect-specific SQL.
//
// Returns SeedExecutorPort which is ready to execute seeds.
func NewSeedExecutor(database *sql.DB, dialect DialectConfig) SeedExecutorPort {
	return migration_sql.NewSeedExecutor(database, dialect)
}

// WithNonBlockingLock configures non-blocking lock acquisition.
//
// If the lock is already held, operations return ErrLockNotAcquired immediately instead
// of waiting.
//
// Returns MigrationServiceOption which enables non-blocking lock behaviour.
func WithNonBlockingLock() MigrationServiceOption {
	return querier_domain.WithNonBlockingLock()
}

// WithBeforeMigration registers a hook that runs before each migration.
//
// Takes hook (BeforeMigrationHook) which observes the upcoming migration.
//
// Returns MigrationServiceOption which installs the before-migration hook.
func WithBeforeMigration(hook BeforeMigrationHook) MigrationServiceOption {
	return querier_domain.WithBeforeMigration(hook)
}

// WithAfterMigration registers a hook that runs after each migration succeeds.
//
// Takes hook (AfterMigrationHook) which observes the completed migration.
//
// Returns MigrationServiceOption which installs the after-migration hook.
func WithAfterMigration(hook AfterMigrationHook) MigrationServiceOption {
	return querier_domain.WithAfterMigration(hook)
}

// WithBeforeRun registers a hook that runs before the migration run begins.
//
// Takes hook (BeforeRunHook) which observes the migration run context.
//
// Returns MigrationServiceOption which installs the before-run hook.
func WithBeforeRun(hook BeforeRunHook) MigrationServiceOption {
	return querier_domain.WithBeforeRun(hook)
}

// WithAfterRun registers a hook that runs after the migration run completes.
//
// Takes hook (AfterRunHook) which observes the migration run context.
//
// Returns MigrationServiceOption which installs the after-run hook.
func WithAfterRun(hook AfterRunHook) MigrationServiceOption {
	return querier_domain.WithAfterRun(hook)
}
