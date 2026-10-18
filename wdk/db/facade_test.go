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

package db_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/db"
)

func TestDirectionConstantsAreDistinct(t *testing.T) {
	t.Parallel()

	require.NotEqual(t, db.DirectionUp, db.DirectionDown)
}

func TestSentinelsAreSet(t *testing.T) {
	t.Parallel()

	for _, err := range []error{
		db.ErrLockNotAcquired, db.ErrNoDownMigration, db.ErrInvalidIdentifier, db.ErrMigrationFileTooLarge,
		db.ErrMalformedSQLStatement, db.ErrNoMigrationBackend, db.ErrNoSQLToolchain, db.ErrUnsupportedDriver,
	} {
		require.Error(t, err)
	}
}

func TestDialectConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		dialect   db.DialectConfig
		name      string
		family    db.DialectFamily
		pgBouncer bool
	}{
		{name: "postgres", dialect: db.PostgresDialect(), family: db.DialectFamilyPostgres},
		{name: "postgres pgbouncer", dialect: db.PostgresPgBouncerDialect(), family: db.DialectFamilyPostgres, pgBouncer: true},
		{name: "cockroachdb", dialect: db.CockroachDBDialect(), family: db.DialectFamilyCockroachDB},
		{name: "mysql", dialect: db.MySQLDialect(), family: db.DialectFamilyMySQL},
		{name: "sqlite", dialect: db.SQLiteDialect(), family: db.DialectFamilySQLite},
		{name: "clickhouse", dialect: db.ClickHouseDialect(), family: db.DialectFamilyClickHouse},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.True(t, testCase.dialect.IsConfigured())
			require.Equal(t, testCase.family, testCase.dialect.Family)
			require.Equal(t, testCase.pgBouncer, testCase.dialect.PgBouncer)
		})
	}
}

func TestMySQLDialectWithDSNKeepsDSN(t *testing.T) {
	t.Parallel()

	dialect := db.MySQLDialectWithDSN("user:pass@tcp(localhost:3306)/app?multiStatements=true")

	require.Equal(t, db.DialectFamilyMySQL, dialect.Family)
	require.Equal(t, "user:pass@tcp(localhost:3306)/app?multiStatements=true", dialect.DSN)
}

func TestNewMigratorWithoutEngine(t *testing.T) {
	t.Parallel()

	_, err := db.NewMigrator(db.EngineConfig{}, nil, fstest.MapFS{}, ".")

	require.ErrorIs(t, err, db.ErrNoSQLToolchain)
}

func TestNewSeederWithoutEngine(t *testing.T) {
	t.Parallel()

	_, err := db.NewSeeder(db.EngineConfig{}, nil, fstest.MapFS{}, ".")

	require.ErrorIs(t, err, db.ErrNoSQLToolchain)
}

func TestGenerateWithoutEngine(t *testing.T) {
	t.Parallel()

	_, err := db.Generate(context.Background(), db.EngineConfig{}, db.GenerateRequest{})

	require.ErrorIs(t, err, db.ErrNoSQLToolchain)
}

func TestGetDatabaseConnectionWithoutBootstrap(t *testing.T) {
	_, err := db.GetDatabaseConnection("test")

	require.Error(t, err)
}

func TestGetMigrationServiceWithoutBootstrap(t *testing.T) {
	_, err := db.GetMigrationService("test")

	require.Error(t, err)
}

func TestGetSeedServiceWithoutBootstrap(t *testing.T) {
	_, err := db.GetSeedService("test")

	require.Error(t, err)
}

func TestGetDatabaseReaderWithoutBootstrap(t *testing.T) {
	_, err := db.GetDatabaseReader("test")

	require.Error(t, err)
}

func TestGetDatabaseWriterWithoutBootstrap(t *testing.T) {
	_, err := db.GetDatabaseWriter("test")

	require.Error(t, err)
}

func TestSeverityConstantsHaveExpectedOrdering(t *testing.T) {
	t.Parallel()

	assert.Less(t, db.SeverityError, db.SeverityWarning)
	assert.Less(t, db.SeverityWarning, db.SeverityHint)
}

func TestMigrationOptionsBuild(t *testing.T) {
	t.Parallel()

	options := []db.MigrationServiceOption{
		db.WithNonBlockingLock(),
		db.WithBeforeMigration(func(context.Context, db.MigrationHookContext) error { return nil }),
		db.WithAfterMigration(func(context.Context, db.MigrationHookContext) error { return nil }),
		db.WithBeforeRun(func(context.Context, db.MigrationRunHookContext) error { return nil }),
		db.WithAfterRun(func(context.Context, db.MigrationRunHookContext, int) error { return nil }),
	}

	require.Len(t, options, 5)
	for _, option := range options {
		require.NotNil(t, option)
	}
}
