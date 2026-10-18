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
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sqldb/sqldb_dto"

	"seagull.sh/seagull/sdk/migrate"
)

func TestMigrationBackendKeepsPikoPersistedNames(t *testing.T) {
	t.Parallel()

	pikoNames := migrate.Names{
		History:                 "piko_migrations",
		Seeds:                   "piko_seeds",
		MigrationLock:           "piko_migration_lock",
		SeedLock:                "piko_seed_lock",
		MigrationLockConstraint: "piko_migration_lock_single_row",
		SeedLockConstraint:      "piko_seed_lock_single_row",
		LockKey:                 "piko_migrations",
		SeedLockKey:             "piko_seeds",
	}
	renamedNames := pikoNames
	renamedNames.History = "app_schema_history"
	renamedNames.LockKey = "app_schema_history"

	testCases := []struct {
		name     string
		builtin  string
		config   sqldb_dto.DialectConfig
		expected migrate.Names
	}{
		{name: "postgres", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyPostgres}, expected: pikoNames},
		{name: "postgres pgbouncer", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyPostgres, PgBouncer: true}, expected: pikoNames},
		{name: "timescaledb", builtin: "timescaledb", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyPostgres}, expected: pikoNames},
		{name: "timescaledb pgbouncer", builtin: "timescaledb", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyPostgres, PgBouncer: true}, expected: pikoNames},
		{name: "cockroachdb", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyCockroachDB}, expected: pikoNames},
		{name: "mysql", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyMySQL}, expected: pikoNames},
		{name: "mysql with dsn", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyMySQL, DSN: "user:pass@tcp(localhost:3306)/app"}, expected: pikoNames},
		{name: "mysql with multi-statement dsn", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyMySQL, DSN: "user:pass@tcp(localhost:3306)/app?multiStatements=true"}, expected: pikoNames},
		{name: "mariadb", builtin: "mariadb", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyMySQL}, expected: pikoNames},
		{name: "sqlite", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilySQLite}, expected: pikoNames},
		{name: "clickhouse", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyClickHouse}, expected: pikoNames},
		{name: "renamed history", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyPostgres, HistoryTable: "app_schema_history"}, expected: renamedNames},
		{name: "renamed history on mysql", config: sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamilyMySQL, HistoryTable: "app_schema_history"}, expected: renamedNames},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			backend, err := migrationBackend(testCase.config, testCase.builtin)

			require.NoError(t, err)
			require.Equal(t, testCase.expected, backend.Names())
		})
	}
}

func TestMigrationBackendRejectsUnknownFamily(t *testing.T) {
	t.Parallel()

	_, err := migrationBackend(sqldb_dto.DialectConfig{Family: sqldb_dto.DialectFamily("oracle")}, "")

	require.Error(t, err)
}
