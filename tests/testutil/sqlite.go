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

package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"slices"
	"testing"
	"time"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/wdk/db/db_schema_orchestrator_sqlite"
	"piko.sh/piko/wdk/db/db_schema_registry_sqlite"
)

const (
	// testPoolSize is the number of workers in the pool during tests.
	testPoolSize = 1

	// testConnMaxIdleTime is the maximum idle time for test database connections.
	testConnMaxIdleTime = 5 * time.Minute

	// testConnMaxLifetime is the maximum lifetime for test database connections.
	testConnMaxLifetime = 1 * time.Hour
)

// OpenTestDB creates a new SQLite database connection for testing. It
// automatically detects whether to use CGO (sqlite3) or pure Go (sqlite)
// driver.
//
// Takes t (testing.TB) for error reporting.
// Takes dsn (string) which is the data source name for the database.
//
// Returns *sql.DB which is the configured database connection.
func OpenTestDB(t testing.TB, dsn string) *sql.DB {
	t.Helper()

	driverName := detectSQLiteDriver()

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("failed to open SQLite database: %v", err)
	}

	db.SetMaxOpenConns(testPoolSize)
	db.SetMaxIdleConns(testPoolSize)
	db.SetConnMaxIdleTime(testConnMaxIdleTime)
	db.SetConnMaxLifetime(testConnMaxLifetime)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("failed to ping SQLite database: %v", err)
	}

	return db
}

// RunRegistryMigrations runs the registry database migrations on the given
// path.
//
// Takes path (string) which specifies the file path to the database.
//
// Returns error when the migrations fail to apply.
func RunRegistryMigrations(path string) error {
	driverName := detectSQLiteDriver()
	database, err := sql.Open(driverName, path)
	if err != nil {
		return fmt.Errorf("failed to open registry database: %w", err)
	}
	defer func() { _ = database.Close() }()

	return runQuerierMigrations(database, db_schema_registry_sqlite.Migrations, "registry")
}

// RunOrchestratorMigrations runs orchestrator database migrations on the given
// path.
//
// Takes path (string) specifying the file path to the database.
//
// Returns error when migrations fail to apply.
func RunOrchestratorMigrations(path string) error {
	driverName := detectSQLiteDriver()
	database, err := sql.Open(driverName, path)
	if err != nil {
		return fmt.Errorf("failed to open orchestrator database: %w", err)
	}
	defer func() { _ = database.Close() }()

	return runQuerierMigrations(database, db_schema_orchestrator_sqlite.Migrations, "orchestrator")
}

// RunRegistryMigrationsOnDB runs registry migrations on an existing database
// connection. Use it for in-memory databases where the connection must stay
// open.
//
// Takes db (*sql.DB) which is the existing database connection.
//
// Returns error when migrations fail to apply.
func RunRegistryMigrationsOnDB(db *sql.DB) error {
	return runQuerierMigrations(db, db_schema_registry_sqlite.Migrations, "registry")
}

// RunOrchestratorMigrationsOnDB runs orchestrator migrations on an existing
// database connection. Use it for in-memory databases where the connection
// must stay open.
//
// Takes db (*sql.DB) which is the existing database connection.
//
// Returns error when migrations fail to apply.
func RunOrchestratorMigrationsOnDB(db *sql.DB) error {
	return runQuerierMigrations(db, db_schema_orchestrator_sqlite.Migrations, "orchestrator")
}

// detectSQLiteDriver returns the appropriate SQLite driver name based on
// which drivers are registered.
//
// Returns string which is the driver name to use for database connections.
func detectSQLiteDriver() string {
	drivers := sql.Drivers()
	if slices.Contains(drivers, "sqlite3") {
		return "sqlite3"
	}
	return "sqlite"
}

// runQuerierMigrations applies database migrations using the querier migration
// service.
func runQuerierMigrations(database *sql.DB, migrationFS fs.FS, name string) error {
	executor := migration_sql.NewExecutor(database, migration_sql.SQLiteDialect())
	fileReader := migration_sql.NewFSFileReader(migrationFS)
	service := querier_domain.NewMigrationService(executor, fileReader, "migrations")

	ctx := context.Background()
	if _, err := service.Up(ctx); err != nil {
		return fmt.Errorf("%s migration failed: %w", name, err)
	}

	return nil
}
