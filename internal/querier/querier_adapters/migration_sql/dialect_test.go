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

package migration_sql_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
)

func TestPostgresDialect_HasExpectedFields(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.PostgresDialect()

	require.Contains(t, dialect.CreateTableSQL, "piko_migrations")
	require.Contains(t, dialect.CreateTableSQL, "TIMESTAMPTZ")
	require.Contains(t, dialect.CreateSeedTableSQL, "piko_seeds")
	require.NotNil(t, dialect.LockStrategy)
	require.IsType(t, &migration_sql.PostgresAdvisoryLock{}, dialect.LockStrategy)
	require.Equal(t, "$1", dialect.PlaceholderFunc(1))
	require.Equal(t, "$5", dialect.PlaceholderFunc(5))
	require.False(t, dialect.SplitStatements)
}

func TestPostgresPgBouncerDialect_UsesTableBasedLock(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.PostgresPgBouncerDialect()

	require.NotNil(t, dialect.LockStrategy)
	require.IsType(t, &migration_sql.TableBasedLock{}, dialect.LockStrategy)
	require.Equal(t, "$1", dialect.PlaceholderFunc(1))
	require.Contains(t, dialect.CreateTableSQL, "piko_migrations")
}

func TestMySQLDialect_UsesQuestionPlaceholdersAndAdvisoryLock(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.MySQLDialect()

	require.IsType(t, &migration_sql.MySQLAdvisoryLock{}, dialect.LockStrategy)
	require.Equal(t, "?", dialect.PlaceholderFunc(1))
	require.Equal(t, "?", dialect.PlaceholderFunc(99))
	require.True(t, dialect.SplitStatements)
	require.Contains(t, dialect.CreateTableSQL, "ENGINE=InnoDB")
}

func TestMySQLDialectWithDSN_TogglesSplitStatementsBasedOnDSN(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		dsn       string
		wantSplit bool
	}{
		{"plain DSN keeps splitting", "user:pass@tcp(localhost)/db", true},
		{"DSN with multiStatements disables splitting", "user:pass@tcp(localhost)/db?multiStatements=true", false},
		{"unrelated query parameters keep splitting", "user:pass@tcp(localhost)/db?parseTime=true", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dialect := migration_sql.MySQLDialectWithDSN(tc.dsn)
			require.Equal(t, tc.wantSplit, dialect.SplitStatements)
		})
	}
}

func TestSQLiteDialect_UsesNoOpLockAndQuestionPlaceholders(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.SQLiteDialect()

	require.IsType(t, &migration_sql.NoOpLock{}, dialect.LockStrategy)
	require.Equal(t, "?", dialect.PlaceholderFunc(1))
	require.False(t, dialect.SplitStatements)
	require.Contains(t, dialect.CreateTableSQL, "piko_migrations")
	require.Contains(t, dialect.CreateSeedTableSQL, "piko_seeds")
}

func TestDialect_PlaceholderFunc_HandlesAllPlaceholders(t *testing.T) {
	t.Parallel()

	pg := migration_sql.PostgresDialect().PlaceholderFunc

	for index := 1; index <= 8; index++ {
		got := pg(index)
		require.True(t, strings.HasPrefix(got, "$"))
	}
}

func TestDialect_HistoryTable_FallsBackToDefault(t *testing.T) {
	t.Parallel()

	require.Equal(t, "piko_migrations", migration_sql.DialectConfig{}.HistoryTable())
	require.Equal(t, "piko_migrations", migration_sql.PostgresDialect().HistoryTable())
	require.Equal(t, "piko_migrations", migration_sql.SQLiteDialect().HistoryTable())
	require.Equal(t, "piko_migrations", migration_sql.MySQLDialect().HistoryTable())
	require.Equal(t, "piko_migrations", migration_sql.PostgresPgBouncerDialect().HistoryTable())
}

func TestDialect_WithHistoryTable_RewritesPostgresDDLAndLockKey(t *testing.T) {
	t.Parallel()

	dialect, err := migration_sql.PostgresDialect().WithHistoryTable("identity_piko_migrations")
	require.NoError(t, err)

	require.Equal(t, "identity_piko_migrations", dialect.HistoryTable())
	require.Contains(t, dialect.CreateTableSQL, "CREATE TABLE IF NOT EXISTS identity_piko_migrations")
	require.NotContains(t, dialect.CreateTableSQL, "EXISTS piko_migrations")
	require.Contains(t, dialect.CreateSeedTableSQL, "piko_seeds")

	lock, ok := dialect.LockStrategy.(*migration_sql.PostgresAdvisoryLock)
	require.True(t, ok, "expected PostgresAdvisoryLock")
	require.Equal(t, "identity_piko_migrations", lock.LockKey)
}

func TestDialect_WithHistoryTable_RewritesSQLiteDDL(t *testing.T) {
	t.Parallel()

	dialect, err := migration_sql.SQLiteDialect().WithHistoryTable("scheduler_piko_migrations")
	require.NoError(t, err)

	require.Equal(t, "scheduler_piko_migrations", dialect.HistoryTable())
	require.Contains(t, dialect.CreateTableSQL, "CREATE TABLE IF NOT EXISTS scheduler_piko_migrations")
	require.NotContains(t, dialect.CreateTableSQL, "EXISTS piko_migrations")
	require.IsType(t, &migration_sql.NoOpLock{}, dialect.LockStrategy)
}

func TestDialect_WithHistoryTable_RewritesMySQLLockKey(t *testing.T) {
	t.Parallel()

	dialect, err := migration_sql.MySQLDialect().WithHistoryTable("content_piko_migrations")
	require.NoError(t, err)

	require.Equal(t, "content_piko_migrations", dialect.HistoryTable())
	require.Contains(t, dialect.CreateTableSQL, "content_piko_migrations")

	lock, ok := dialect.LockStrategy.(*migration_sql.MySQLAdvisoryLock)
	require.True(t, ok, "expected MySQLAdvisoryLock")
	require.Equal(t, "content_piko_migrations", lock.LockKey)
}

func TestDialect_WithHistoryTable_EmptyNameIsNoop(t *testing.T) {
	t.Parallel()

	original := migration_sql.PostgresDialect()
	renamed, err := original.WithHistoryTable("")
	require.NoError(t, err)

	require.Equal(t, original.HistoryTable(), renamed.HistoryTable())
	require.Equal(t, original.CreateTableSQL, renamed.CreateTableSQL)
}

func TestDialect_WithHistoryTable_IsImmutable(t *testing.T) {
	t.Parallel()

	original := migration_sql.PostgresDialect()
	renamed, err := original.WithHistoryTable("tenancy_piko_migrations")
	require.NoError(t, err)

	require.NotEqual(t, original.HistoryTable(), renamed.HistoryTable(),
		"original config must not be mutated by WithHistoryTable")
	require.Contains(t, original.CreateTableSQL, "piko_migrations")
	require.NotContains(t, original.CreateTableSQL, "tenancy_piko_migrations")

	originalLock, ok := original.LockStrategy.(*migration_sql.PostgresAdvisoryLock)
	require.True(t, ok)
	require.NotEqual(t, "tenancy_piko_migrations", originalLock.LockKey)
}

func TestDialect_WithHistoryTable_ChainedRenamesArePreserved(t *testing.T) {
	t.Parallel()

	first, err := migration_sql.PostgresDialect().WithHistoryTable("first_name")
	require.NoError(t, err)
	dialect, err := first.WithHistoryTable("second_name")
	require.NoError(t, err)

	require.Equal(t, "second_name", dialect.HistoryTable())
	require.Contains(t, dialect.CreateTableSQL, "CREATE TABLE IF NOT EXISTS second_name")
	require.NotContains(t, dialect.CreateTableSQL, "first_name")
	require.NotContains(t, dialect.CreateTableSQL, "EXISTS piko_migrations")
}

func TestClickHouseDialect_UsesNoOpLockAndAppendOnlyHistory(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.ClickHouseDialect()

	require.IsType(t, &migration_sql.NoOpLock{}, dialect.LockStrategy)
	require.IsType(t, &migration_sql.NoOpLock{}, dialect.SeedLockStrategy)
	require.Equal(t, "?", dialect.PlaceholderFunc(1))
	require.Equal(t, "?", dialect.PlaceholderFunc(99))
	require.True(t, dialect.SplitStatements)

	require.True(t, dialect.DisableTransactions, "ClickHouse has no DDL transactions")
	require.True(t, dialect.AppendOnlyHistory, "ClickHouse history must be append-only")
	require.True(t, dialect.SelectHistoryFinal, "ClickHouse reads history with FINAL")
	require.True(t, dialect.BackslashEscapes, "ClickHouse string literals use backslash escapes")
	require.Contains(t, dialect.CreateTableSQL, "ENGINE = ReplacingMergeTree(applied_at)")
	require.Contains(t, dialect.CreateSeedTableSQL, "ENGINE = ReplacingMergeTree(applied_at)")

	require.NotNil(t, dialect.DeleteHistorySQLFunc)
	require.Equal(t,
		"ALTER TABLE piko_migrations DELETE WHERE version = ?",
		dialect.DeleteHistorySQLFunc("piko_migrations", "?"),
	)
}

func TestClickHouseDialect_InsertSeedSQLFuncIsIdempotent(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.ClickHouseDialect()

	require.NotNil(t, dialect.InsertSeedSQLFunc,
		"ClickHouseDialect must populate InsertSeedSQLFunc to keep seeds idempotent")

	sqlText := dialect.InsertSeedSQLFunc("?", "?", "?", "?")
	require.Contains(t, sqlText, "INSERT INTO piko_seeds")
	require.Contains(t, sqlText, "NOT IN (SELECT version FROM piko_seeds)",
		"the guard must filter the candidate row out when the version is already recorded")

	require.Equal(t, 4, strings.Count(sqlText, "?"),
		"the rendered SQL must take exactly four positional placeholders")
}

func TestDialect_WithHistoryTable_RejectsUnsafeIdentifiers(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"piko_migrations; DROP TABLE users",
		"name with space",
		"name'with'quote",
		"name\"with\"double",
		"1starts_with_digit",
		"--comment",
		"piko-migrations",
		string(make([]byte, 64)),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			original := migration_sql.PostgresDialect()
			renamed, err := original.WithHistoryTable(name)
			require.Error(t, err, "expected %q to be rejected", name)
			require.ErrorIs(t, err, migration_sql.ErrInvalidIdentifier)
			require.Equal(t, original.HistoryTable(), renamed.HistoryTable(),
				"failed rename must leave the original config unchanged")
		})
	}
}
