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
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// MigrationService defines the driving port for database migration operations.
// It provides methods for applying, rolling back, and inspecting migration
// state.
type MigrationService = querier_domain.MigrationServicePort

// MigrationExecutor defines the database-specific operations needed by the
// migration service.
type MigrationExecutor = querier_domain.MigrationExecutorPort

// EnginePort defines the aggregate adapter contract for SQL dialect parsers.
type EnginePort = querier_domain.EnginePort

// CatalogueProviderPort defines the contract for building a schema catalogue
// from a live database.
type CatalogueProviderPort = querier_domain.CatalogueProviderPort

// FileReaderPort abstracts filesystem access for reading migration and query
// SQL files.
type FileReaderPort = querier_domain.FileReaderPort

// SeedService defines the driving port for database seed operations.
type SeedService = querier_domain.SeedServicePort

// SeedExecutorPort defines the database-specific operations needed by the
// seed service.
type SeedExecutorPort = querier_domain.SeedExecutorPort

// MigrationServiceOption configures optional behaviour of the migration
// service.
type MigrationServiceOption = querier_domain.MigrationServiceOption

// BeforeMigrationHook is called before each individual migration executes.
type BeforeMigrationHook = querier_domain.BeforeMigrationHook

// AfterMigrationHook is called after each individual migration executes
// successfully.
type AfterMigrationHook = querier_domain.AfterMigrationHook

// BeforeRunHook is called before the migration run begins.
type BeforeRunHook = querier_domain.BeforeRunHook

// AfterRunHook is called after the migration run completes successfully.
type AfterRunHook = querier_domain.AfterRunHook

// MigrationHookContext provides information about an individual migration
// being processed.
type MigrationHookContext = querier_domain.MigrationHookContext

// MigrationRunHookContext provides information about an entire migration run
// before it begins.
type MigrationRunHookContext = querier_domain.MigrationRunHookContext

// MigrationDirection indicates whether a migration is a forward (up) or
// rollback (down) migration.
type MigrationDirection = querier_dto.MigrationDirection

// MigrationStatus combines a migration file with its applied state.
type MigrationStatus = querier_dto.MigrationStatus

// MigrationFile represents a parsed migration file with version, direction,
// and content.
type MigrationFile = querier_dto.MigrationFile

// AppliedMigration represents a migration that has been applied to the
// database.
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

// DialectConfig holds dialect-specific SQL and behaviour for the migration
// executor.
type DialectConfig = migration_sql.DialectConfig

// ChecksumMismatchError is returned when an applied migration's recorded
// checksum does not match the current file on disk.
type ChecksumMismatchError = querier_domain.ChecksumMismatchError

// DownChecksumMismatchError is returned when a down migration file's checksum
// does not match the checksum recorded when the up migration was applied.
type DownChecksumMismatchError = querier_domain.DownChecksumMismatchError

// MigrationExecutionError wraps an error from executing a migration's SQL
// content.
type MigrationExecutionError = querier_domain.MigrationExecutionError

// LockAcquisitionError wraps a failure to acquire the migration advisory lock.
type LockAcquisitionError = querier_domain.LockAcquisitionError

// MissingMigrationFileError is returned when the database records an applied
// migration but no corresponding file exists on disk.
type MissingMigrationFileError = querier_domain.MissingMigrationFileError

// NoDownMigrationError is returned when a rollback is requested for a version
// that has no .down.sql file.
type NoDownMigrationError = querier_domain.NoDownMigrationError

const (
	// DatabaseNameRegistry is the reserved database name for piko's internal
	// registry subsystem. Register a database with this name to back the
	// registry with a SQL database instead of the default otter in-memory
	// backend.
	DatabaseNameRegistry = bootstrap.DatabaseNameRegistry

	// DatabaseNameOrchestrator is the reserved database name for piko's
	// internal orchestrator subsystem. Register a database with this name to
	// back the orchestrator with a SQL database instead of the default otter
	// in-memory backend.
	DatabaseNameOrchestrator = bootstrap.DatabaseNameOrchestrator

	// DirectionUp is a forward migration that applies schema changes.
	DirectionUp = querier_dto.MigrationDirectionUp

	// DirectionDown is a rollback migration that reverts schema changes.
	DirectionDown = querier_dto.MigrationDirectionDown
)

// ErrLockNotAcquired is returned when a non-blocking lock attempt fails
// because another process already holds the migration lock.
var ErrLockNotAcquired = querier_domain.ErrLockNotAcquired

// PostgresDialect returns a DialectConfig for PostgreSQL databases.
func PostgresDialect() DialectConfig { return migration_sql.PostgresDialect() }

// PostgresPgBouncerDialect returns a DialectConfig for PostgreSQL databases
// behind PgBouncer in transaction mode, using table-based locking instead
// of advisory locks.
func PostgresPgBouncerDialect() DialectConfig { return migration_sql.PostgresPgBouncerDialect() }

// MySQLDialect returns a DialectConfig for MySQL databases.
func MySQLDialect() DialectConfig { return migration_sql.MySQLDialect() }

// MySQLDialectWithDSN returns a DialectConfig for MySQL databases, detecting
// whether the DSN includes multiStatements=true. When the driver handles
// multi-statement execution natively, statement splitting is disabled.
func MySQLDialectWithDSN(dsn string) DialectConfig {
	return migration_sql.MySQLDialectWithDSN(dsn)
}

// SQLiteDialect returns a DialectConfig for SQLite databases.
func SQLiteDialect() DialectConfig { return migration_sql.SQLiteDialect() }

// NewMigrationService creates a migration service for executing database
// migrations. The service handles applying, rolling back, and inspecting
// migration state with advisory locking and checksum verification.
//
// Takes executor (MigrationExecutor) which provides database-specific
// migration operations.
// Takes fileReader (FileReaderPort) which provides filesystem access for
// reading migration SQL files.
// Takes directory (string) which is the path to the migration files within
// the filesystem.
// Takes opts (...MigrationServiceOption) which configure optional behaviour
// such as non-blocking lock acquisition and lifecycle hooks.
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

// NewMigrationExecutor creates a SQL-based migration executor from a database
// connection and dialect configuration.
//
// Takes database (*sql.DB) which is the database connection to execute
// migrations against.
// Takes dialect (DialectConfig) which provides dialect-specific SQL and
// locking behaviour.
//
// Returns MigrationExecutor which is ready to execute migrations.
func NewMigrationExecutor(database *sql.DB, dialect DialectConfig) MigrationExecutor {
	return migration_sql.NewExecutor(database, dialect)
}

// NewFSFileReader creates a file reader backed by an fs.FS. This is typically
// used with embed.FS for embedding migration files into the binary, or with
// os.DirFS for reading from the local filesystem.
//
// Takes filesystem (fs.FS) which is the filesystem to read migration files
// from.
//
// Returns FileReaderPort which is ready to read files.
func NewFSFileReader(filesystem fs.FS) FileReaderPort {
	return migration_sql.NewFSFileReader(filesystem)
}

// NewSeedService creates a seed service for applying database seed files.
// The service handles executing SQL seed files in version order with
// idempotency tracking via a history table.
//
// Takes executor (SeedExecutorPort) which provides database-specific seed
// operations.
// Takes fileReader (FileReaderPort) which provides filesystem access for
// reading seed SQL files.
// Takes directory (string) which is the path to the seed files within the
// filesystem.
//
// Returns SeedService which is ready to apply seeds.
func NewSeedService(
	executor SeedExecutorPort,
	fileReader FileReaderPort,
	directory string,
) SeedService {
	return querier_domain.NewSeedService(executor, fileReader, directory)
}

// NewSeedExecutor creates a SQL-based seed executor from a database connection
// and dialect configuration.
//
// Takes database (*sql.DB) which is the database connection to execute seeds
// against.
// Takes dialect (DialectConfig) which provides dialect-specific SQL.
//
// Returns SeedExecutorPort which is ready to execute seeds.
func NewSeedExecutor(database *sql.DB, dialect DialectConfig) SeedExecutorPort {
	return migration_sql.NewSeedExecutor(database, dialect)
}

// WithNonBlockingLock configures the migration service to use a non-blocking
// lock acquisition. If the lock is already held, operations return
// ErrLockNotAcquired immediately instead of waiting.
func WithNonBlockingLock() MigrationServiceOption {
	return querier_domain.WithNonBlockingLock()
}

// WithBeforeMigration registers a hook that runs before each individual
// migration.
func WithBeforeMigration(hook BeforeMigrationHook) MigrationServiceOption {
	return querier_domain.WithBeforeMigration(hook)
}

// WithAfterMigration registers a hook that runs after each individual
// migration succeeds.
func WithAfterMigration(hook AfterMigrationHook) MigrationServiceOption {
	return querier_domain.WithAfterMigration(hook)
}

// WithBeforeRun registers a hook that runs before the migration run begins.
func WithBeforeRun(hook BeforeRunHook) MigrationServiceOption {
	return querier_domain.WithBeforeRun(hook)
}

// WithAfterRun registers a hook that runs after the migration run completes.
func WithAfterRun(hook AfterRunHook) MigrationServiceOption {
	return querier_domain.WithAfterRun(hook)
}

// DBTX is the common database interface for read and write operations. It
// matches the interface generated by the querier code generator and is
// satisfied by *sql.DB and *sql.Tx.
type DBTX = bootstrap.DBTX

// Replica configures a single read replica connection.
type Replica = bootstrap.Replica

// DatabaseRegistration holds the configuration for a named database connection
// registered during bootstrap. See bootstrap.DatabaseRegistration for field
// documentation.
type DatabaseRegistration = bootstrap.DatabaseRegistration

// WithDatabase returns a bootstrap option that registers a named database
// connection. When the name is DatabaseNameRegistry or
// DatabaseNameOrchestrator, the framework uses the querier-based DAL adapters
// instead of the default otter in-memory backend for that subsystem.
//
// Takes name (string) which identifies the database.
// Takes registration (*DatabaseRegistration) which provides connection and
// migration configuration.
//
// Returns bootstrap.Option which registers the database with the container.
func WithDatabase(name string, registration *DatabaseRegistration) bootstrap.Option {
	return bootstrap.WithDatabase(name, registration)
}

// GetDatabaseConnection returns the *sql.DB for a named database registered
// during bootstrap. This function is concurrency-safe and can be called from
// multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns *sql.DB which is the open database connection.
// Returns error when the framework is not initialised or the database is not
// registered.
func GetDatabaseConnection(name string) (*sql.DB, error) {
	return bootstrap.GetDatabaseConnection(name)
}

// GetDatabaseReader returns the DBTX for reading from a named database
// registered during bootstrap. When replicas are configured, this returns a
// round-robin balancer across them. When no replicas are configured, this
// returns the primary connection. This function is concurrency-safe and can be
// called from multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which can execute read queries.
// Returns error when the framework is not initialised or the database is not
// registered.
func GetDatabaseReader(name string) (DBTX, error) {
	return bootstrap.GetDatabaseReader(name)
}

// GetDatabaseWriter returns the DBTX for writing to a named database
// registered during bootstrap. When EnableOTel is set on the registration, the
// returned DBTX is wrapped with OpenTelemetry spans and metrics. This function
// is concurrency-safe and can be called from multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which can execute write queries.
// Returns error when the framework is not initialised or the database is not
// registered.
func GetDatabaseWriter(name string) (DBTX, error) {
	return bootstrap.GetDatabaseWriter(name)
}

// GetMigrationService returns the migration service for a named database
// registered during bootstrap. This function is concurrency-safe and can be
// called from multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns MigrationService which can apply and roll back migrations.
// Returns error when the framework is not initialised, the database is not
// registered, or no migration filesystem was configured.
func GetMigrationService(name string) (MigrationService, error) {
	return bootstrap.GetMigrationService(name)
}

// GetSeedService returns the seed service for a named database registered
// during bootstrap. This function is concurrency-safe and can be called from
// multiple goroutines.
//
// Takes name (string) which identifies the database.
//
// Returns SeedService which can apply and inspect seeds.
// Returns error when the framework is not initialised, the database is not
// registered, or no seed filesystem was configured.
func GetSeedService(name string) (SeedService, error) {
	return bootstrap.GetSeedService(name)
}
