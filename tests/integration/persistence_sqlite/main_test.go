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

package persistence_sqlite_test

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/testutil/leakcheck"
	"piko.sh/piko/wdk/db/db_schema_orchestrator_sqlite"
	"piko.sh/piko/wdk/db/db_schema_registry_sqlite"
)

func TestMain(m *testing.M) {
	code := m.Run()

	if code == 0 {
		if err := leakcheck.FindLeaks(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(code)
}

func openTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()

	db_path := filepath.Join(t.TempDir(), name)
	database, err := sql.Open("sqlite", "file:"+db_path)
	require.NoError(t, err, "opening SQLite database")

	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)
	database.SetConnMaxIdleTime(5 * time.Minute)
	database.SetConnMaxLifetime(1 * time.Hour)

	_, err = database.Exec("PRAGMA journal_mode = WAL")
	require.NoError(t, err, "setting WAL journal mode")

	_, err = database.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err, "enabling foreign keys")

	_, err = database.Exec("PRAGMA busy_timeout = 10000")
	require.NoError(t, err, "setting busy timeout")

	t.Cleanup(func() {
		_ = database.Close()
	})

	return database
}

func runMigrations(t *testing.T, database *sql.DB, migration_fs embed.FS) {
	t.Helper()

	ctx := context.Background()

	executor := migration_sql.NewExecutor(database, migration_sql.SQLiteDialect())
	file_reader := migration_sql.NewFSFileReader(migration_fs)
	service := querier_domain.NewMigrationService(executor, file_reader, "migrations")

	_, err := service.Up(ctx)
	require.NoError(t, err, "running migrations")
}

func setupRegistryDB(t *testing.T) *sql.DB {
	t.Helper()

	database := openTestDB(t, "registry.db")
	runMigrations(t, database, db_schema_registry_sqlite.Migrations)
	return database
}

func setupOrchestratorDB(t *testing.T) *sql.DB {
	t.Helper()

	database := openTestDB(t, "orchestrator.db")
	runMigrations(t, database, db_schema_orchestrator_sqlite.Migrations)
	return database
}
