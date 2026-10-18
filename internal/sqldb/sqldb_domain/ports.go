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

package sqldb_domain

import (
	"context"
	"database/sql"
	"io/fs"

	"piko.sh/piko/internal/sqldb/sqldb_dto"
)

// MigrationService applies, reverts and inspects schema migrations for one database.
type MigrationService interface {
	// Up applies every pending migration.
	//
	// Returns int which is the number of migrations applied.
	// Returns error when a migration fails or the lock cannot be taken.
	Up(ctx context.Context) (int, error)

	// Down reverts the most recent steps migrations.
	//
	// Takes steps (int) which is the number of migrations to revert.
	//
	// Returns int which is the number of migrations reverted.
	// Returns error when a rollback fails or a down migration is missing.
	Down(ctx context.Context, steps int) (int, error)

	// Status reports every migration file and whether it has been applied.
	//
	// Returns []sqldb_dto.MigrationStatus which is ordered by version.
	// Returns error when the history cannot be read.
	Status(ctx context.Context) ([]sqldb_dto.MigrationStatus, error)

	// UpTo applies pending migrations up to and including targetVersion.
	//
	// Takes targetVersion (int64) which is the highest migration version to apply.
	//
	// Returns int which is the number of migrations applied.
	// Returns error when a migration fails.
	UpTo(ctx context.Context, targetVersion int64) (int, error)

	// DownTo reverts migrations newer than targetVersion.
	//
	// Takes targetVersion (int64) which is the highest migration version to retain.
	//
	// Returns int which is the number of migrations reverted.
	// Returns error when a rollback fails.
	DownTo(ctx context.Context, targetVersion int64) (int, error)

	// Validate checks applied migrations against the files without changing anything.
	//
	// Returns error when a checksum differs or an applied migration has no file.
	Validate(ctx context.Context) error
}

// SeedService applies and inspects seed data for one database.
type SeedService interface {
	// Apply runs every seed that has not been applied.
	//
	// Returns int which is the number of seeds applied.
	// Returns error when a seed fails.
	Apply(ctx context.Context) (int, error)

	// Status reports every seed file and whether it has been applied.
	//
	// Returns []sqldb_dto.SeedStatus which is ordered by version.
	// Returns error when the history cannot be read.
	Status(ctx context.Context) ([]sqldb_dto.SeedStatus, error)

	// Reseed clears the seed history and applies every seed again.
	//
	// Returns int which is the number of seeds applied.
	// Returns error when a seed fails.
	Reseed(ctx context.Context) (int, error)
}

// ToolchainPort generates SQL access code and builds migration and seed services for one
// SQL dialect.
type ToolchainPort interface {
	// Generate analyses the request's migrations and queries and emits Go code.
	//
	// Error-severity diagnostics are reported in the result, not as an error.
	//
	// Takes request (sqldb_dto.GenerateRequest) which supplies migrations, queries, and
	// generation settings.
	//
	// Returns *sqldb_dto.GenerationResult which holds the files and diagnostics.
	// Returns error when generation cannot run at all.
	Generate(ctx context.Context, request sqldb_dto.GenerateRequest) (*sqldb_dto.GenerationResult, error)

	// NewMigrationService builds a migration service over database.
	//
	// Takes database (*sql.DB) which is the connection pool migrations run on.
	// Takes dialect (sqldb_dto.DialectConfig) which selects the migration backend.
	// Takes files (fs.FS) which holds the migration files.
	// Takes directory (string) which is the migration directory within files.
	// Takes config (sqldb_dto.MigrationServiceConfig) which holds hooks and locking.
	//
	// Returns MigrationService which is ready to use.
	// Returns error when the dialect has no migration backend.
	NewMigrationService(
		database *sql.DB,
		dialect sqldb_dto.DialectConfig,
		files fs.FS,
		directory string,
		config sqldb_dto.MigrationServiceConfig,
	) (MigrationService, error)

	// NewSeedService builds a seed service over database.
	//
	// Takes database (*sql.DB) which is the connection pool seeds run on.
	// Takes dialect (sqldb_dto.DialectConfig) which selects the backend.
	// Takes files (fs.FS) which holds the seed files.
	// Takes directory (string) which is the seed directory within files.
	//
	// Returns SeedService which is ready to use.
	// Returns error when the dialect has no seed backend.
	NewSeedService(database *sql.DB, dialect sqldb_dto.DialectConfig, files fs.FS, directory string) (SeedService, error)
}

// EngineToolchainPort describes a registered database engine's dialect and toolchain.
type EngineToolchainPort interface {
	// Dialect returns the engine's dialect name, such as "postgres" or "sqlite".
	//
	// Returns string which identifies the SQL dialect.
	Dialect() string

	// Toolchain returns the code generation and migration toolchain for the dialect.
	//
	// Returns ToolchainPort which provides generation, migration, and seed services.
	Toolchain() ToolchainPort
}
