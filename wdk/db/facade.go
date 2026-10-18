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
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/sqldb/sqldb_domain"
	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

// Migration and seed services.
type (
	// MigrationService applies, reverts and inspects schema migrations for one database.
	MigrationService = sqldb_domain.MigrationService

	// SeedService applies and inspects seed data for one database.
	SeedService = sqldb_domain.SeedService

	// MigrationServiceOption configures a migration service.
	MigrationServiceOption = sqldb_domain.MigrationServiceOption

	// BeforeMigrationHook runs before each migration. Returning an error stops the run.
	BeforeMigrationHook = sqldb_dto.BeforeMigrationHook

	// AfterMigrationHook runs after each successful migration.
	AfterMigrationHook = sqldb_dto.AfterMigrationHook

	// BeforeRunHook runs once before a batch of migrations.
	BeforeRunHook = sqldb_dto.BeforeRunHook

	// AfterRunHook runs once after a batch of migrations, with the number applied.
	AfterRunHook = sqldb_dto.AfterRunHook

	// MigrationHookContext describes the migration a per-migration hook runs around.
	MigrationHookContext = sqldb_dto.MigrationHookContext

	// MigrationRunHookContext describes the batch a run hook runs around.
	MigrationRunHookContext = sqldb_dto.MigrationRunHookContext

	// MigrationDirection says whether a migration applies or reverts a change.
	MigrationDirection = sqldb_dto.MigrationDirection

	// MigrationStatus describes one migration file and whether it has been applied.
	MigrationStatus = sqldb_dto.MigrationStatus

	// AppliedMigration is one row of the migration history table.
	AppliedMigration = sqldb_dto.AppliedMigration

	// SeedStatus describes one seed file and whether it has been applied.
	SeedStatus = sqldb_dto.SeedStatus

	// AppliedSeed is one row of the seed history table.
	AppliedSeed = sqldb_dto.AppliedSeed

	// DialectConfig selects and tunes the migration backend for one database.
	DialectConfig = sqldb_dto.DialectConfig

	// DialectFamily names a migration backend.
	DialectFamily = sqldb_dto.DialectFamily
)

// Code generation.
type (
	// GenerateRequest describes one code generation run.
	GenerateRequest = sqldb_dto.GenerateRequest

	// GenerationResult is the outcome of generating code for one database.
	GenerationResult = sqldb_dto.GenerationResult

	// GeneratedFile is one Go source file produced by code generation.
	GeneratedFile = sqldb_dto.GeneratedFile

	// GeneratedDriver selects the database driver API generated code targets.
	GeneratedDriver = sqldb_dto.GeneratedDriver

	// SourceError is one diagnostic produced while analysing migrations and queries.
	SourceError = sqldb_dto.SourceError

	// Severity classifies a diagnostic.
	Severity = sqldb_dto.Severity

	// TypeOverride maps a SQL type to a Go type in generated code.
	TypeOverride = sqldb_dto.TypeOverride

	// CustomFunction declares a SQL function the analyser cannot discover.
	CustomFunction = sqldb_dto.CustomFunction
)

// Typed migration and seed errors. Match them with errors.As.
type (
	// ChecksumMismatchError reports an applied migration whose file has changed since.
	ChecksumMismatchError = sqldb_domain.ChecksumMismatchError

	// DownChecksumMismatchError reports a down migration that changed after its up migration
	// was applied.
	DownChecksumMismatchError = sqldb_domain.DownChecksumMismatchError

	// MigrationExecutionError reports a migration or rollback that failed to run.
	MigrationExecutionError = sqldb_domain.MigrationExecutionError

	// LockAcquisitionError reports a failure to take the migration lock.
	LockAcquisitionError = sqldb_domain.LockAcquisitionError

	// MissingMigrationFileError reports an applied migration whose file no longer exists.
	MissingMigrationFileError = sqldb_domain.MissingMigrationFileError

	// NoDownMigrationError reports a rollback that needs a missing down migration.
	NoDownMigrationError = sqldb_domain.NoDownMigrationError

	// DuplicateMigrationVersionError reports two migration files with the same version.
	DuplicateMigrationVersionError = sqldb_domain.DuplicateMigrationVersionError

	// DirtyMigrationError reports a migration that stopped part-way through.
	DirtyMigrationError = sqldb_domain.DirtyMigrationError

	// DuplicateSeedVersionError reports two seed files with the same version.
	DuplicateSeedVersionError = sqldb_domain.DuplicateSeedVersionError

	// SeedExecutionError reports a seed that failed to run.
	SeedExecutionError = sqldb_domain.SeedExecutionError
)

const (
	// DirectionUp is a forward migration.
	DirectionUp = sqldb_dto.DirectionUp

	// DirectionDown is a rollback migration.
	DirectionDown = sqldb_dto.DirectionDown

	// SeverityError indicates a problem that prevents code generation.
	SeverityError = sqldb_dto.SeverityError

	// SeverityWarning indicates a likely problem that does not block generation.
	SeverityWarning = sqldb_dto.SeverityWarning

	// SeverityHint indicates a suggestion for improvement.
	SeverityHint = sqldb_dto.SeverityHint

	// DriverDatabaseSQL generates code against database/sql.
	DriverDatabaseSQL = sqldb_dto.DriverDatabaseSQL

	// DriverPgx generates code against github.com/jackc/pgx/v5.
	DriverPgx = sqldb_dto.DriverPgx

	// DriverClickHouseNative generates code against the native ClickHouse driver.
	DriverClickHouseNative = sqldb_dto.DriverClickHouseNative

	// DialectFamilyPostgres is PostgreSQL and its wire-compatible databases.
	DialectFamilyPostgres = sqldb_dto.DialectFamilyPostgres

	// DialectFamilyCockroachDB is CockroachDB, whose migrations take a table lock.
	DialectFamilyCockroachDB = sqldb_dto.DialectFamilyCockroachDB

	// DialectFamilyMySQL is MySQL and MariaDB.
	DialectFamilyMySQL = sqldb_dto.DialectFamilyMySQL

	// DialectFamilySQLite is SQLite and Cloudflare D1.
	DialectFamilySQLite = sqldb_dto.DialectFamilySQLite

	// DialectFamilyClickHouse is ClickHouse.
	DialectFamilyClickHouse = sqldb_dto.DialectFamilyClickHouse
)

const (
	// DatabaseNameRegistry is the reserved database name for the registry subsystem.
	DatabaseNameRegistry = bootstrap.DatabaseNameRegistry

	// DatabaseNameOrchestrator is the reserved database name for the orchestrator subsystem.
	DatabaseNameOrchestrator = bootstrap.DatabaseNameOrchestrator
)

var (
	// ErrLockNotAcquired is returned when a non-blocking run finds the migration lock held.
	ErrLockNotAcquired = sqldb_domain.ErrLockNotAcquired

	// ErrNoDownMigration is returned when a rollback needs a missing down migration.
	ErrNoDownMigration = sqldb_domain.ErrNoDownMigration

	// ErrInvalidIdentifier is returned when a table or lock name is not a safe SQL
	// identifier.
	ErrInvalidIdentifier = sqldb_domain.ErrInvalidIdentifier

	// ErrMigrationFileTooLarge is returned when a migration or seed file exceeds the size
	// limit.
	ErrMigrationFileTooLarge = sqldb_domain.ErrMigrationFileTooLarge

	// ErrMalformedSQLStatement is returned when a migration cannot be split into statements.
	ErrMalformedSQLStatement = sqldb_domain.ErrMalformedSQLStatement

	// ErrNoMigrationBackend is returned when a dialect has no migration backend.
	ErrNoMigrationBackend = sqldb_domain.ErrNoMigrationBackend

	// ErrNoSQLToolchain is returned when a database has no engine.
	ErrNoSQLToolchain = sqldb_domain.ErrNoSQLToolchain

	// ErrUnsupportedDriver is returned when the toolchain cannot target the requested driver
	// API.
	ErrUnsupportedDriver = sqldb_domain.ErrUnsupportedDriver
)

// WithNonBlockingLock makes a run fail with ErrLockNotAcquired when another process holds
// the migration lock, instead of waiting for it.
//
// Returns MigrationServiceOption which enables non-blocking locking.
func WithNonBlockingLock() MigrationServiceOption {
	return sqldb_domain.WithNonBlockingLock()
}

// WithBeforeMigration adds a hook that runs before each migration.
//
// Takes hook (BeforeMigrationHook) which may stop the run by returning an error.
//
// Returns MigrationServiceOption which registers the hook.
func WithBeforeMigration(hook BeforeMigrationHook) MigrationServiceOption {
	return sqldb_domain.WithBeforeMigration(hook)
}

// WithAfterMigration adds a hook that runs after each successful migration.
//
// Takes hook (AfterMigrationHook) which may stop the run by returning an error.
//
// Returns MigrationServiceOption which registers the hook.
func WithAfterMigration(hook AfterMigrationHook) MigrationServiceOption {
	return sqldb_domain.WithAfterMigration(hook)
}

// WithBeforeRun adds a hook that runs once before a batch of migrations.
//
// Takes hook (BeforeRunHook) which may stop the run by returning an error.
//
// Returns MigrationServiceOption which registers the hook.
func WithBeforeRun(hook BeforeRunHook) MigrationServiceOption {
	return sqldb_domain.WithBeforeRun(hook)
}

// WithAfterRun adds a hook that runs once after a batch of migrations.
//
// Takes hook (AfterRunHook) which receives the number applied.
//
// Returns MigrationServiceOption which registers the hook.
func WithAfterRun(hook AfterRunHook) MigrationServiceOption {
	return sqldb_domain.WithAfterRun(hook)
}

// PostgresDialect selects the PostgreSQL migration backend with advisory locking.
//
// Returns DialectConfig which selects the backend.
func PostgresDialect() DialectConfig {
	return DialectConfig{Family: DialectFamilyPostgres, DSN: "", HistoryTable: "", PgBouncer: false}
}

// PostgresPgBouncerDialect selects the PostgreSQL migration backend with table-based
// locking, for PgBouncer in transaction mode where advisory locks are unavailable.
//
// Returns DialectConfig which selects the backend.
func PostgresPgBouncerDialect() DialectConfig {
	return DialectConfig{Family: DialectFamilyPostgres, DSN: "", HistoryTable: "", PgBouncer: true}
}

// CockroachDBDialect selects PostgreSQL migrations with table-based locking for
// CockroachDB.
//
// Returns DialectConfig which selects the backend.
func CockroachDBDialect() DialectConfig {
	return DialectConfig{Family: DialectFamilyCockroachDB, DSN: "", HistoryTable: "", PgBouncer: false}
}

// MySQLDialect selects the MySQL and MariaDB migration backend.
//
// Returns DialectConfig which selects the backend.
func MySQLDialect() DialectConfig {
	return DialectConfig{Family: DialectFamilyMySQL, DSN: "", HistoryTable: "", PgBouncer: false}
}

// MySQLDialectWithDSN selects the MySQL migration backend and lets it inspect dsn to
// decide whether the driver accepts multiple statements per call.
//
// Takes dsn (string) which is the MySQL connection string.
//
// Returns DialectConfig which selects the backend.
func MySQLDialectWithDSN(dsn string) DialectConfig {
	return DialectConfig{Family: DialectFamilyMySQL, DSN: dsn, HistoryTable: "", PgBouncer: false}
}

// SQLiteDialect selects the SQLite migration backend.
//
// Returns DialectConfig which selects the backend.
func SQLiteDialect() DialectConfig {
	return DialectConfig{Family: DialectFamilySQLite, DSN: "", HistoryTable: "", PgBouncer: false}
}

// ClickHouseDialect selects the ClickHouse migration backend.
//
// Returns DialectConfig which selects the backend.
func ClickHouseDialect() DialectConfig {
	return DialectConfig{Family: DialectFamilyClickHouse, DSN: "", HistoryTable: "", PgBouncer: false}
}

// NewMigrator builds a migration service for database using engine's toolchain.
//
// The engine's own migration dialect is used; pass a DialectConfig in the engine
// configuration to change it.
//
// Takes engine (EngineConfig) which supplies the toolchain and migration dialect.
// Takes database (*sql.DB) which is the connection pool migrations run on.
// Takes files (fs.FS) which holds the migration files.
// Takes directory (string) which is the migration directory within files.
// Takes options (...MigrationServiceOption) which add hooks and locking behaviour.
//
// Returns MigrationService which is ready to use.
// Returns error when the engine has no toolchain or no migration backend.
func NewMigrator(
	engine EngineConfig,
	database *sql.DB,
	files fs.FS,
	directory string,
	options ...MigrationServiceOption,
) (MigrationService, error) {
	if engine.Engine == nil {
		return nil, ErrNoSQLToolchain
	}
	migrator, err := engine.Engine.Toolchain().NewMigrationService(
		database, engine.MigrationDialect, files, directory, sqldb_domain.BuildMigrationServiceConfig(options...),
	)
	if err != nil {
		return nil, fmt.Errorf("creating migration service: %w", err)
	}
	return migrator, nil
}

// NewSeeder builds a seed service for database using engine's toolchain.
//
// Takes engine (EngineConfig) which supplies the toolchain and migration dialect.
// Takes database (*sql.DB) which is the connection pool seeds run on.
// Takes files (fs.FS) which holds the seed files.
// Takes directory (string) which is the seed directory within files.
//
// Returns SeedService which is ready to use.
// Returns error when the engine has no toolchain or no seed backend.
func NewSeeder(engine EngineConfig, database *sql.DB, files fs.FS, directory string) (SeedService, error) {
	if engine.Engine == nil {
		return nil, ErrNoSQLToolchain
	}
	seeder, err := engine.Engine.Toolchain().NewSeedService(database, engine.MigrationDialect, files, directory)
	if err != nil {
		return nil, fmt.Errorf("creating seed service: %w", err)
	}
	return seeder, nil
}

// Generate analyses migrations and queries and emits Go code using engine's toolchain.
//
// Error-severity diagnostics are reported in the result, not as an error.
//
// Takes engine (EngineConfig) which supplies the toolchain.
// Takes request (GenerateRequest) which names the files and settings.
//
// Returns *GenerationResult which holds the files, diagnostics and query count.
// Returns error when the engine has no toolchain or generation cannot run.
func Generate(ctx context.Context, engine EngineConfig, request GenerateRequest) (*GenerationResult, error) {
	if engine.Engine == nil {
		return nil, ErrNoSQLToolchain
	}
	result, err := engine.Engine.Toolchain().Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("generating SQL access code: %w", err)
	}
	return result, nil
}

// DBTX is the common interface for database operations, satisfied by *sql.DB and *sql.Tx.
type DBTX = bootstrap.DBTX

// Replica configures a read replica for a database registration.
type Replica = bootstrap.Replica

// DatabaseRegistration configures a named database connection with its engine,
// migrations, seeds and generated-code settings.
type DatabaseRegistration = bootstrap.DatabaseRegistration

// WithDatabase registers a named database with Piko.
//
// Takes name (string) which identifies the database.
// Takes registration (*DatabaseRegistration) which configures it.
//
// Returns bootstrap.Option which applies the registration.
func WithDatabase(name string, registration *DatabaseRegistration) bootstrap.Option {
	return bootstrap.WithDatabase(name, registration)
}

// GetDatabaseConnection returns the *sql.DB for a named database.
//
// Takes name (string) which identifies the database.
//
// Returns *sql.DB which is the connection pool.
// Returns error when the database is not registered or cannot be opened.
func GetDatabaseConnection(name string) (*sql.DB, error) {
	return bootstrap.GetDatabaseConnection(name)
}

// GetDatabaseReader returns the read-side connection for a named database, balanced
// across replicas when any are configured.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which routes reads.
// Returns error when the database is not registered or cannot be opened.
func GetDatabaseReader(name string) (DBTX, error) {
	return bootstrap.GetDatabaseReader(name)
}

// GetDatabaseWriter returns the write-side connection for a named database.
//
// Takes name (string) which identifies the database.
//
// Returns DBTX which routes writes to the primary.
// Returns error when the database is not registered or cannot be opened.
func GetDatabaseWriter(name string) (DBTX, error) {
	return bootstrap.GetDatabaseWriter(name)
}

// GetMigrationService returns the migration service for a named database.
//
// Takes name (string) which identifies the database.
//
// Returns MigrationService which can apply and roll back migrations.
// Returns error when the database is not registered or has no migration filesystem.
func GetMigrationService(name string) (MigrationService, error) {
	return bootstrap.GetMigrationService(name)
}

// GetSeedService returns the seed service for a named database.
//
// Takes name (string) which identifies the database.
//
// Returns SeedService which can apply and inspect seeds.
// Returns error when the database is not registered or has no seed filesystem.
func GetSeedService(name string) (SeedService, error) {
	return bootstrap.GetSeedService(name)
}
