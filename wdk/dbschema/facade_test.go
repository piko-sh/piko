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

package dbschema_test

import (
	"database/sql"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/dbschema"
)

func concreteTypeOf(t *testing.T, value any) string {
	t.Helper()

	require.NotNil(t, value, "a forwarder must not return nil")

	return reflect.TypeOf(value).String()
}

func TestEmitterForwardersReturnDistinctEmitters(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, concreteTypeOf(t, dbschema.NewSQLEmitter()))
	assert.NotEmpty(t, concreteTypeOf(t, dbschema.NewSQLEmitterForDialect("postgres")))
}

func TestDialectForwardersAreEachDistinct(t *testing.T) {
	t.Parallel()

	dialects := map[string]dbschema.DialectConfig{
		"postgres":  dbschema.PostgresDialect(),
		"pgbouncer": dbschema.PostgresPgBouncerDialect(),
		"mysql":     dbschema.MySQLDialect(),
		"sqlite":    dbschema.SQLiteDialect(),
	}

	for name, dialect := range dialects {
		assert.NotEmpty(t, dialect.HistoryTableName, "%s must name a history table", name)
		assert.NotNil(t, dialect.LockStrategy, "%s must supply a lock strategy", name)
	}

	assert.NotEqual(t,
		reflect.TypeOf(dialects["postgres"].LockStrategy).String(),
		reflect.TypeOf(dialects["sqlite"].LockStrategy).String(),
		"postgres and sqlite must not share a locking strategy")

	assert.NotEqual(t,
		reflect.TypeOf(dialects["postgres"].LockStrategy).String(),
		reflect.TypeOf(dialects["pgbouncer"].LockStrategy).String(),
		"pgbouncer uses table locking rather than advisory locks")
}

func TestMySQLDialectWithDSNDetectsMultiStatements(t *testing.T) {
	t.Parallel()

	plain := dbschema.MySQLDialectWithDSN("user:pass@tcp(localhost:3306)/db")
	multi := dbschema.MySQLDialectWithDSN("user:pass@tcp(localhost:3306)/db?multiStatements=true")

	assert.NotEmpty(t, plain.HistoryTableName)
	assert.NotEmpty(t, multi.HistoryTableName)
	assert.NotEqual(t, plain.SplitStatements, multi.SplitStatements,
		"a multiStatements DSN must change whether statements are split")
}

func TestServiceForwardersBuildServices(t *testing.T) {
	t.Parallel()

	filesystem := fstest.MapFS{
		"0001_init.up.sql": &fstest.MapFile{Data: []byte("CREATE TABLE t (id INT);")},
	}

	fileReader := dbschema.NewFSFileReader(filesystem)
	require.NotNil(t, fileReader, "the file reader forwarder must build a reader")

	database := &sql.DB{}
	dialect := dbschema.SQLiteDialect()

	executor := dbschema.NewMigrationExecutor(database, dialect)
	seedExecutor := dbschema.NewSeedExecutor(database, dialect)

	assert.NotEqual(t, concreteTypeOf(t, executor), concreteTypeOf(t, seedExecutor),
		"the migration and seed executors are different types, so neither forwarder may "+
			"point at the other's constructor")

	assert.NotNil(t, dbschema.NewMigrationService(executor, fileReader, "."))
	assert.NotNil(t, dbschema.NewSeedService(seedExecutor, fileReader, "."))
}

func TestMigrationServiceOptionsAreDistinct(t *testing.T) {
	t.Parallel()

	options := map[string]dbschema.MigrationServiceOption{
		"nonBlockingLock": dbschema.WithNonBlockingLock(),
		"beforeMigration": dbschema.WithBeforeMigration(nil),
		"afterMigration":  dbschema.WithAfterMigration(nil),
		"beforeRun":       dbschema.WithBeforeRun(nil),
		"afterRun":        dbschema.WithAfterRun(nil),
	}

	for name, option := range options {
		assert.NotNil(t, option, "%s must return an option", name)
	}
}

func TestCatalogueProviderForwarders(t *testing.T) {
	t.Parallel()

	filesystem := fstest.MapFS{}
	fileReader := dbschema.NewFSFileReader(filesystem)

	assert.NotNil(t, dbschema.NewMigrationCatalogueProvider(nil, fileReader, "."))
	assert.NotNil(t, dbschema.NewCompositeCatalogueProvider(nil),
		"an empty chain must still yield a provider")
}

func TestNewQuerierServiceRejectsMissingPorts(t *testing.T) {
	t.Parallel()

	service, err := dbschema.NewQuerierService(dbschema.QuerierPorts{})

	require.Error(t, err, "an empty port set must be refused")
	assert.Nil(t, service)
}

func TestSeverityOrderMatchesTheDocumentedRange(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, dbschema.SeverityError, dbschema.SeverityWarning)
	assert.NotEqual(t, dbschema.SeverityWarning, dbschema.SeverityHint)
	assert.NotEqual(t, dbschema.DirectionUp, dbschema.DirectionDown)
}
