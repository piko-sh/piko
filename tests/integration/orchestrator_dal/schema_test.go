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

//go:build integration

package orchestrator_dal_test

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	orchestrator_querier_postgres "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_postgres"
	orchestrator_querier_sqlite "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

func migrationsBefore(t *testing.T, migrations fs.FS, version string) fs.FS {
	t.Helper()
	entries, err := fs.ReadDir(migrations, "migrations")
	require.NoError(t, err)
	older := fstest.MapFS{}
	for _, entry := range entries {
		if entry.Name() >= version {
			continue
		}
		data, err := fs.ReadFile(migrations, "migrations/"+entry.Name())
		require.NoError(t, err)
		older["migrations/"+entry.Name()] = &fstest.MapFile{Data: data}
	}
	return older
}

type schemaUpgradeCase struct {
	open        func(t *testing.T) *sql.DB
	check       func(t *testing.T, database *sql.DB) error
	store       func(database *sql.DB) orchestrator_domain.TaskStore
	engine      db.EngineConfig
	migrations  fs.FS
	table       string
	placeholder func(n int) string
}

func runSchemaUpgrade(t *testing.T, c schemaUpgradeCase) {
	database := c.open(t)
	runMigrations(t, database, c.engine, migrationsBefore(t, c.migrations, "003"))
	require.ErrorIs(t, c.check(t, database), orchestrator_dal.ErrSchemaOutdated)

	insert := fmt.Sprintf(`INSERT INTO %s (id, workflow_id, executor, priority, payload, config, status, execute_at,
attempt, created_at, updated_at, deduplication_key) VALUES (%s)`, c.table, placeholders(c.placeholder, 12))
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	for _, row := range []struct{ id, status string }{{"old-scheduled", "SCHEDULED"}, {"old-pending", "PENDING"}} {
		_, err := database.Exec(insert, row.id, row.id, "legacy", 1, `{"k":"v"}`, `{}`, row.status, at+3600, 0, at, at, "dedup:"+row.id)
		require.NoError(t, err)
	}

	runMigrations(t, database, c.engine, c.migrations)
	require.NoError(t, c.check(t, database), "the probe passes once every migration is applied")

	store := c.store(database)
	rows, err := store.GetTasksByID(t.Context(), []string{"old-scheduled", "old-pending"})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	byID := map[string]*orchestrator_domain.Task{rows[0].ID: rows[0], rows[1].ID: rows[1]}
	assert.Empty(t, byID["old-pending"].BuildTag)
	assert.False(t, byID["old-pending"].Background)
	assert.True(t, byID["old-pending"].ScheduledExecuteAt.IsZero())
	assert.Equal(t, time.Unix(at+3600, 0).UTC(), byID["old-scheduled"].ScheduledExecuteAt.UTC(),
		"the upgrade backfills a scheduled row's due time")

	claimed, err := store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{
		Now: time.Unix(at, 0), ID: "old-pending", Owner: "node-a",
	})
	require.NoError(t, err, "a row written before the upgrade can be claimed")
	assert.Equal(t, 1, claimed.Attempt)
}

func placeholders(placeholder func(n int) string, n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = placeholder(i + 1)
	}
	return strings.Join(parts, ", ")
}

func TestSchemaUpgrade_SQLite(t *testing.T) {
	runSchemaUpgrade(t, schemaUpgradeCase{
		open: func(t *testing.T) *sql.DB {
			return openSQLite(t, t.TempDir()+"/orchestrator.db")
		},
		check: func(t *testing.T, database *sql.DB) error {
			return orchestrator_querier_sqlite.CheckSchema(t.Context(), database)
		},
		store: func(database *sql.DB) orchestrator_domain.TaskStore {
			return orchestrator_querier_sqlite.New(database)
		},
		engine:      db_engine_sqlite.SQLite(),
		migrations:  orchestrator_querier_sqlite.Migrations,
		table:       "tasks",
		placeholder: func(int) string { return "?" },
	})
}

func TestSchemaUpgrade_Postgres(t *testing.T) {
	if postgresConnectionString == "" {
		t.Skipf("Postgres unavailable: %s", postgresUnavailableReason)
	}
	runSchemaUpgrade(t, schemaUpgradeCase{
		open: func(t *testing.T) *sql.DB {
			schema := fmt.Sprintf("orch_upgrade_%d_%d", os.Getpid(), postgresSchemaCounter.Add(1))
			admin, err := sql.Open("pgx", postgresConnectionString)
			require.NoError(t, err)
			_, err = admin.Exec("CREATE SCHEMA " + schema)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
				_ = admin.Close()
			})
			return openPostgres(t, schema)
		},
		check: func(t *testing.T, database *sql.DB) error {
			return orchestrator_querier_postgres.CheckSchema(t.Context(), database)
		},
		store: func(database *sql.DB) orchestrator_domain.TaskStore {
			return orchestrator_querier_postgres.New(database, dalcore.WithClock(nil))
		},
		engine:      db_engine_postgres.Postgres(),
		migrations:  orchestrator_querier_postgres.Migrations,
		table:       "orchestrator_tasks",
		placeholder: func(n int) string { return fmt.Sprintf("$%d", n) },
	})
}
