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
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"piko.sh/piko/internal/sqldb/sqldb_domain"
	"piko.sh/piko/internal/sqldb/sqldb_dto"
	"piko.sh/piko/wdk/db"

	"seagull.sh/seagull/sdk/dialect"
	"seagull.sh/seagull/sdk/dialect/dialect_registry"
)

var (
	_ sqldb_domain.EngineToolchainPort = (*Engine)(nil)

	_ sqldb_domain.ToolchainPort = (*failedToolchain)(nil)
)

// DialectOptions selects and tunes one of Seagull's built-in dialects.
type DialectOptions struct {
	// Returning overrides RETURNING support for the MySQL family. Nil keeps the default.
	Returning *bool

	// JoinUseNulls sets the assumed ClickHouse join_use_nulls setting.
	//
	// Nil defaults to off. Enable it when the server returns NULL for unmatched outer-join
	// rows.
	JoinUseNulls *bool

	// Dialect names the built-in SQL dialect.
	Dialect string

	// Name overrides the dialect name the engine reports. Empty keeps the default.
	Name string

	// MaxParseDepth limits expression nesting. Zero keeps the default.
	MaxParseDepth int

	// MaxTokensPerStatement limits the size of one statement. Zero keeps the default.
	MaxTokensPerStatement int

	// MaxTypeParseDepth limits type nesting for ClickHouse. Zero keeps the default.
	MaxTypeParseDepth int
}

// Engine adapts a Seagull dialect to Piko's engine port.
type Engine struct {
	// toolchain serves the dialect.
	toolchain sqldb_domain.ToolchainPort

	// name is the dialect name the engine reports.
	name string
}

// NewBuiltinEngine returns the engine for one of Seagull's built-in dialects.
//
// A dialect that cannot be built (an unknown name or an invalid option) still returns an
// engine, whose toolchain reports the error from every operation, so engine constructors
// keep their single-value signatures.
//
// Takes options (DialectOptions) which select and tune the dialect.
//
// Returns *Engine which serves the dialect to Piko.
func NewBuiltinEngine(options DialectOptions) *Engine {
	chosen, err := dialect_registry.New(options.Dialect, dialect_registry.Options{
		Returning:             options.Returning,
		JoinUseNulls:          options.JoinUseNulls,
		Name:                  options.Name,
		MaxParseDepth:         options.MaxParseDepth,
		MaxTokensPerStatement: options.MaxTokensPerStatement,
		MaxTypeParseDepth:     options.MaxTypeParseDepth,
	})
	name := options.Name
	if name == "" {
		name = options.Dialect
	}
	if err != nil {
		return &Engine{toolchain: &failedToolchain{err: fmt.Errorf("configuring the %s engine: %w", options.Dialect, err)}, name: name}
	}
	return &Engine{toolchain: newBuiltinToolchain(chosen, options.Dialect), name: chosen.Info().Name}
}

// NewEngine wraps a custom Seagull dialect as a Piko engine configuration.
//
// Takes custom (dialect.Dialect) which analyses the SQL.
// Takes driverName (string) which is the database/sql driver name, such as "pgx".
// Takes migrationDialect (db.DialectConfig) which selects the migration backend; the zero
// value marks a codegen-only engine.
//
// Returns db.EngineConfig which can be used in a database registration.
func NewEngine(custom dialect.Dialect, driverName string, migrationDialect db.DialectConfig) db.EngineConfig {
	return db.EngineConfig{
		DriverName:       driverName,
		Engine:           &Engine{toolchain: NewToolchain(custom), name: custom.Info().Name},
		MigrationDialect: migrationDialect,
	}
}

// Dialect returns the dialect name.
//
// Returns string which is the name the dialect reports, such as "postgres".
func (e *Engine) Dialect() string {
	return e.name
}

// Toolchain returns the dialect's code generation, migration and seed toolchain.
//
// Returns sqldb_domain.ToolchainPort which serves the dialect.
func (e *Engine) Toolchain() sqldb_domain.ToolchainPort {
	return e.toolchain
}

// failedToolchain reports a configuration error from every operation.
type failedToolchain struct {
	// err is the configuration error.
	err error
}

// Generate reports the configuration error.
//
// Takes _ (sqldb_dto.GenerateRequest) which is the request (unused).
//
// Returns *sqldb_dto.GenerationResult which is always nil.
// Returns error which is the configuration error.
func (f *failedToolchain) Generate(_ context.Context, _ sqldb_dto.GenerateRequest) (*sqldb_dto.GenerationResult, error) {
	return nil, f.err
}

// NewMigrationService reports the configuration error.
//
// Takes _ (*sql.DB) which is the connection pool (unused).
// Takes _ (sqldb_dto.DialectConfig) which is the migration backend (unused).
// Takes _ (fs.FS) which holds the migration files (unused).
// Takes _ (string) which is the migration directory (unused).
// Takes _ (sqldb_dto.MigrationServiceConfig) which holds hooks and locking (unused).
//
// Returns sqldb_domain.MigrationService which is always nil.
// Returns error which is the configuration error.
func (f *failedToolchain) NewMigrationService(
	_ *sql.DB,
	_ sqldb_dto.DialectConfig,
	_ fs.FS,
	_ string,
	_ sqldb_dto.MigrationServiceConfig,
) (sqldb_domain.MigrationService, error) {
	return nil, f.err
}

// NewSeedService reports the configuration error.
//
// Takes _ (*sql.DB) which is the connection pool (unused).
// Takes _ (sqldb_dto.DialectConfig) which is the migration backend (unused).
// Takes _ (fs.FS) which holds the seed files (unused).
// Takes _ (string) which is the seed directory (unused).
//
// Returns sqldb_domain.SeedService which is always nil.
// Returns error which is the configuration error.
func (f *failedToolchain) NewSeedService(
	_ *sql.DB,
	_ sqldb_dto.DialectConfig,
	_ fs.FS,
	_ string,
) (sqldb_domain.SeedService, error) {
	return nil, f.err
}
