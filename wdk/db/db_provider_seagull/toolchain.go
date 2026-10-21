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

package db_provider_seagull

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"piko.sh/piko/internal/sqldb/sqldb_domain"
	"piko.sh/piko/internal/sqldb/sqldb_dto"

	"seagull.sh/seagull"
	"seagull.sh/seagull/sdk/dialect"
	"seagull.sh/seagull/sdk/emit"
	"seagull.sh/seagull/sdk/emit/emit_driver_clickhouse"
	"seagull.sh/seagull/sdk/emit/emit_driver_databasesql"
	"seagull.sh/seagull/sdk/emit/emit_driver_pgx"
	"seagull.sh/seagull/sdk/migrate"
	"seagull.sh/seagull/sdk/migrate/migrate_backend_clickhouse"
	"seagull.sh/seagull/sdk/migrate/migrate_backend_cockroachdb"
	"seagull.sh/seagull/sdk/migrate/migrate_backend_mysql"
	"seagull.sh/seagull/sdk/migrate/migrate_backend_postgres"
	"seagull.sh/seagull/sdk/migrate/migrate_backend_sqlite"
	"seagull.sh/seagull/sdk/migrate/migrate_backend_timescaledb"
)

const (
	// defaultPackageName is the package name of generated code when the request sets none.
	defaultPackageName = "generated"

	// timescaleDBDialect is the built-in dialect whose PostgreSQL-family migrations run on
	// Seagull's TimescaleDB backend.
	timescaleDBDialect = "timescaledb"
)

// postgresFamilyBackend names the Seagull backend that PostgreSQL-family migrations run
// on.
type postgresFamilyBackend uint8

const (
	// backendPostgres is Seagull's PostgreSQL backend, with an advisory lock.
	backendPostgres postgresFamilyBackend = iota

	// backendPostgresPgBouncer is Seagull's PostgreSQL backend with a table lock, for a
	// database behind PgBouncer.
	backendPostgresPgBouncer

	// backendTimescaleDB is Seagull's TimescaleDB backend, which runs a migration that
	// creates or refreshes a continuous aggregate outside a transaction.
	backendTimescaleDB
)

var (
	_ sqldb_domain.ToolchainPort = (*Toolchain)(nil)
)

// Toolchain serves one SQL dialect to Piko through Seagull.
type Toolchain struct {
	// dialect analyses the dialect's SQL.
	dialect dialect.Dialect

	// observer reports Seagull's stages as Piko's spans and metrics.
	observer *otelObserver

	// builtin is the name of the built-in dialect the toolchain serves, or empty for a
	// custom dialect. It selects the migration backend within a dialect family.
	builtin string
}

// NewToolchain returns a toolchain for a Seagull dialect.
//
// Takes chosen (dialect.Dialect) which analyses the dialect's SQL.
//
// Returns *Toolchain which generates code and builds migration and seed services in
// Piko's vocabulary.
func NewToolchain(chosen dialect.Dialect) *Toolchain {
	return &Toolchain{dialect: chosen, observer: newOTelObserver(), builtin: ""}
}

// newBuiltinToolchain returns a toolchain for one of Seagull's built-in dialects.
//
// Takes chosen (dialect.Dialect) which analyses the dialect's SQL.
// Takes builtin (string) which is the built-in dialect's name, such as "timescaledb".
//
// Returns *Toolchain which also picks the built-in dialect's own migration backend.
func newBuiltinToolchain(chosen dialect.Dialect, builtin string) *Toolchain {
	return &Toolchain{dialect: chosen, observer: newOTelObserver(), builtin: builtin}
}

// Generate analyses the request's migrations and queries and emits Go code.
//
// Takes request (sqldb_dto.GenerateRequest) which names the files and settings.
//
// Returns *sqldb_dto.GenerationResult which holds the files, diagnostics and query count.
// Returns error when the driver is unsupported or generation cannot run.
func (t *Toolchain) Generate(ctx context.Context, request sqldb_dto.GenerateRequest) (result *sqldb_dto.GenerationResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result, err = nil, panicError(ctx, "generating SQL access code", recovered)
		}
	}()

	emitter, err := emitterFor(request.Driver)
	if err != nil {
		return nil, err
	}

	facts := t.dialect.Info()
	generator, err := seagull.New(t.dialect,
		seagull.WithEmitter(emitter),
		seagull.WithPreset(pikoPreset()),
		seagull.WithTypeOverrides(mergeTypeOverrides(presetTypeOverrides(&facts), toSeagullTypeOverrides(request.TypeOverrides))...),
		seagull.WithCustomFunctions(toSeagullCustomFunctions(request.CustomFunctions)...),
		seagull.WithRuntimeTables(pikoRuntimeTables()...),
		seagull.WithObserver(t.observer),
	)
	if err != nil {
		return nil, translateError(err)
	}

	generated, err := generator.Generate(ctx, seagull.Input{
		Package:      cmp.Or(request.PackageName, defaultPackageName),
		Migrations:   request.Files,
		MigrationDir: request.MigrationDirectory,
		Queries:      request.Files,
		QueryDir:     request.QueryDirectory,
	})
	if err != nil {
		return nil, translateError(err)
	}

	return &sqldb_dto.GenerationResult{
		Files:       fromSeagullFiles(generated.Files),
		Diagnostics: fromSeagullDiagnostics(generated.Diagnostics),
		QueryCount:  generated.QueryCount,
	}, nil
}

// NewMigrationService builds a migration service over database.
//
// Takes database (*sql.DB) which is the connection pool migrations run on.
// Takes dialectConfig (sqldb_dto.DialectConfig) which selects the migration backend.
// Takes files (fs.FS) which holds the migration files.
// Takes directory (string) which is the migration directory within files.
// Takes config (sqldb_dto.MigrationServiceConfig) which holds hooks and locking.
//
// Returns sqldb_domain.MigrationService which is ready to use.
// Returns error when the dialect has no migration backend.
func (t *Toolchain) NewMigrationService(
	database *sql.DB,
	dialectConfig sqldb_dto.DialectConfig,
	files fs.FS,
	directory string,
	config sqldb_dto.MigrationServiceConfig,
) (sqldb_domain.MigrationService, error) {
	backend, err := migrationBackend(dialectConfig, t.builtin)
	if err != nil {
		return nil, err
	}

	migrator, err := migrate.NewMigrator(database, backend, files, directory, t.migrationOptions(config)...)
	if err != nil {
		return nil, translateError(err)
	}
	return &migrationService{inner: migrator}, nil
}

// NewSeedService builds a seed service over database.
//
// Takes database (*sql.DB) which is the connection pool seeds run on.
// Takes dialectConfig (sqldb_dto.DialectConfig) which selects the backend.
// Takes files (fs.FS) which holds the seed files.
// Takes directory (string) which is the seed directory within files.
//
// Returns sqldb_domain.SeedService which is ready to use.
// Returns error when the dialect has no seed backend.
func (t *Toolchain) NewSeedService(
	database *sql.DB,
	dialectConfig sqldb_dto.DialectConfig,
	files fs.FS,
	directory string,
) (sqldb_domain.SeedService, error) {
	backend, err := migrationBackend(dialectConfig, t.builtin)
	if err != nil {
		return nil, err
	}

	seeder, err := migrate.NewSeeder(database, backend, files, directory,
		migrate.WithDirectivePrefixes(pikoPrefix),
		migrate.WithObserver(t.observer),
	)
	if err != nil {
		return nil, translateError(err)
	}
	return &seedService{inner: seeder}, nil
}

// migrationOptions converts Piko's migration settings into Seagull options.
//
// Takes config (sqldb_dto.MigrationServiceConfig) which holds hooks and locking.
//
// Returns []migrate.Option which configures the migrator.
func (t *Toolchain) migrationOptions(config sqldb_dto.MigrationServiceConfig) []migrate.Option {
	options := make([]migrate.Option, 0,
		len(config.BeforeMigration)+len(config.AfterMigration)+len(config.BeforeRun)+len(config.AfterRun)+3)
	options = append(options, migrate.WithDirectivePrefixes(pikoPrefix), migrate.WithObserver(t.observer))
	if config.NonBlockingLock {
		options = append(options, migrate.WithNonBlockingLock())
	}
	for _, hook := range config.BeforeMigration {
		options = append(options, migrate.WithBeforeMigration(func(ctx context.Context, hookContext migrate.MigrationHookContext) error {
			return hook(ctx, fromSeagullHookContext(hookContext))
		}))
	}
	for _, hook := range config.AfterMigration {
		options = append(options, migrate.WithAfterMigration(func(ctx context.Context, hookContext migrate.MigrationHookContext) error {
			return hook(ctx, fromSeagullHookContext(hookContext))
		}))
	}
	for _, hook := range config.BeforeRun {
		options = append(options, migrate.WithBeforeRun(func(ctx context.Context, hookContext migrate.MigrationRunHookContext) error {
			return hook(ctx, fromSeagullRunHookContext(hookContext))
		}))
	}
	for _, hook := range config.AfterRun {
		options = append(options, migrate.WithAfterRun(func(ctx context.Context, hookContext migrate.MigrationRunHookContext, applied int) error {
			return hook(ctx, fromSeagullRunHookContext(hookContext), applied)
		}))
	}
	return options
}

// emitterFor returns the Seagull emitter for Piko's driver choice.
//
// Takes driver (sqldb_dto.GeneratedDriver) which names the driver API.
//
// Returns emit.EmitterPort which generates code for that driver.
// Returns error when the driver is unknown.
func emitterFor(driver sqldb_dto.GeneratedDriver) (emit.EmitterPort, error) {
	switch driver {
	case sqldb_dto.DriverDatabaseSQL:
		return emit_driver_databasesql.New(), nil
	case sqldb_dto.DriverPgx:
		return emit_driver_pgx.New(), nil
	case sqldb_dto.DriverClickHouseNative:
		return emit_driver_clickhouse.New(), nil
	default:
		return nil, fmt.Errorf("%w: driver %d", sqldb_domain.ErrUnsupportedDriver, driver)
	}
}

// migrationBackend selects a backend with Piko's table and lock names.
//
// MySQL uses server-wide lock keys and honours multiStatements for seeds. TimescaleDB
// uses its dedicated backend for nontransactional continuous-aggregate operations.
//
// Takes dialectConfig (sqldb_dto.DialectConfig) which names the backend.
// Takes builtin (string) which is the toolchain's built-in dialect name, or empty.
//
// Returns migrate.Backend which drives the executors.
// Returns error when the descriptor names no backend.
func migrationBackend(dialectConfig sqldb_dto.DialectConfig, builtin string) (migrate.Backend, error) {
	options := []migrate.BackendOption{migrate.WithNamespace(pikoNamespace)}
	if dialectConfig.HistoryTable != "" {
		options = append(options, migrate.WithHistoryTable(dialectConfig.HistoryTable))
	}

	switch dialectConfig.Family {
	case sqldb_dto.DialectFamilyPostgres:
		return postgresBackend(dialectConfig, builtin, options), nil
	case sqldb_dto.DialectFamilyCockroachDB:
		return migrate_backend_cockroachdb.New(options...), nil
	case sqldb_dto.DialectFamilyMySQL:
		return migrate_backend_mysql.New(append(options,
			migrate_backend_mysql.WithServerWideLockKey(),
			migrate.WithDriverMultiStatements(dsnEnablesMultiStatements(dialectConfig.DSN)),
		)...), nil
	case sqldb_dto.DialectFamilySQLite:
		return migrate_backend_sqlite.New(options...), nil
	case sqldb_dto.DialectFamilyClickHouse:
		return migrate_backend_clickhouse.New(options...), nil
	default:
		return migrate.Backend{}, fmt.Errorf("%w: %q", sqldb_domain.ErrNoMigrationBackend, dialectConfig.Family)
	}
}

// postgresBackend returns the PostgreSQL-family backend that postgresBackendFor chooses,
// with a table lock behind PgBouncer.
//
// Takes dialectConfig (sqldb_dto.DialectConfig) which says whether PgBouncer is in use.
// Takes builtin (string) which is the toolchain's built-in dialect name, or empty.
// Takes options ([]migrate.BackendOption) which set Piko's names.
//
// Returns migrate.Backend which drives the executors.
func postgresBackend(dialectConfig sqldb_dto.DialectConfig, builtin string, options []migrate.BackendOption) migrate.Backend {
	switch postgresBackendFor(dialectConfig, builtin) {
	case backendTimescaleDB:
		if dialectConfig.PgBouncer {
			options = append(options, migrate.WithLock(migrate.LockTable))
		}
		return migrate_backend_timescaledb.New(options...)
	case backendPostgresPgBouncer:
		return migrate_backend_postgres.NewPgBouncer(options...)
	default:
		return migrate_backend_postgres.New(options...)
	}
}

// postgresBackendFor selects TimescaleDB for that dialect, otherwise PostgreSQL with
// PgBouncer support when configured.
//
// Takes dialectConfig (sqldb_dto.DialectConfig) which says whether PgBouncer is in use.
// Takes builtin (string) which is the toolchain's built-in dialect name, or empty.
//
// Returns postgresFamilyBackend which names the chosen backend.
func postgresBackendFor(dialectConfig sqldb_dto.DialectConfig, builtin string) postgresFamilyBackend {
	if builtin == timescaleDBDialect {
		return backendTimescaleDB
	}
	if dialectConfig.PgBouncer {
		return backendPostgresPgBouncer
	}
	return backendPostgres
}
