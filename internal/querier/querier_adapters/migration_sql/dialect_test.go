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
