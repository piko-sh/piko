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
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalconformance"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	orchestrator_querier_postgres "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_postgres"
	orchestrator_querier_sqlite "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

const (
	poolSize = 4
)

var (
	postgresSchemaCounter atomic.Int64
	sqlGaps               = map[string]string{}
	sqliteGaps            = withGaps(sqlGaps, nil)
	postgresGaps          = withGaps(sqlGaps, nil)
)

func TestConformance_SQLite(t *testing.T) {
	dalconformance.Run(t, dalconformance.Backend{
		Open: func(t *testing.T, clock clockpkg.Clock) dalconformance.Instance {
			path := filepath.Join(t.TempDir(), "orchestrator.db")
			database := openSQLite(t, path)
			runMigrations(t, database, db_engine_sqlite.SQLite(), orchestrator_querier_sqlite.Migrations)
			return dalconformance.Instance{
				Store: orchestrator_querier_sqlite.New(database, dalcore.WithClock(clock)),
				Reopen: func(t *testing.T) orchestrator_domain.TaskStore {
					_ = database.Close()
					return orchestrator_querier_sqlite.New(openSQLite(t, path), dalcore.WithClock(clock))
				},
			}
		},
		Gaps: sqliteGaps,
	})
}

func TestConformance_Postgres(t *testing.T) {
	if postgresConnectionString == "" {
		t.Skipf("Postgres unavailable: %s", postgresUnavailableReason)
	}
	dalconformance.Run(t, dalconformance.Backend{
		Open: func(t *testing.T, clock clockpkg.Clock) dalconformance.Instance {
			schema := fmt.Sprintf("orch_conf_%d_%d", os.Getpid(), postgresSchemaCounter.Add(1))
			admin, err := sql.Open("pgx", postgresConnectionString)
			require.NoError(t, err)
			_, err = admin.Exec("CREATE SCHEMA " + schema)
			require.NoError(t, err, "creating schema %q", schema)
			t.Cleanup(func() {
				_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
				_ = admin.Close()
			})

			database := openPostgres(t, schema)
			runMigrations(t, database, db_engine_postgres.Postgres(), orchestrator_querier_postgres.Migrations)
			return dalconformance.Instance{
				Store: orchestrator_querier_postgres.New(database, dalcore.WithClock(clock)),
				Reopen: func(t *testing.T) orchestrator_domain.TaskStore {
					_ = database.Close()
					return orchestrator_querier_postgres.New(openPostgres(t, schema), dalcore.WithClock(clock))
				},
			}
		},
		Gaps: postgresGaps,
	})
}

func withGaps(shared, specific map[string]string) map[string]string {
	gaps := make(map[string]string, len(shared)+len(specific))
	maps.Copy(gaps, shared)
	maps.Copy(gaps, specific)
	return gaps
}

func openSQLite(t *testing.T, path string) *sql.DB {
	t.Helper()
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	database, err := sql.Open("sqlite", dsn)
	require.NoError(t, err, "opening SQLite database")
	database.SetMaxOpenConns(poolSize)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func openPostgres(t *testing.T, schema string) *sql.DB {
	t.Helper()
	database, err := sql.Open("pgx", postgresDSNWithSearchPath(postgresConnectionString, schema))
	require.NoError(t, err, "opening Postgres database")
	database.SetMaxOpenConns(poolSize)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func postgresDSNWithSearchPath(dsn, schema string) string {
	optionsValue := "-csearch_path=" + schema
	if !strings.Contains(dsn, "://") {
		return dsn + " options='" + optionsValue + "'"
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "options=" + url.QueryEscape(optionsValue)
}

func runMigrations(t *testing.T, database *sql.DB, engine db.EngineConfig, migrations fs.FS) {
	t.Helper()
	service, err := db.NewMigrator(engine, database, migrations, "migrations")
	require.NoError(t, err, "creating orchestrator migration service")
	_, err = service.Up(t.Context())
	require.NoError(t, err, "running orchestrator migrations")
}
