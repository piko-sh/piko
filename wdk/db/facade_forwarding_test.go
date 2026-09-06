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
	"database/sql"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/dbschema"
)

func TestForwardersMatchDBSchema(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		reflect.TypeOf(dbschema.NewSQLEmitter()).String(),
		reflect.TypeOf(db.NewSQLEmitter()).String())

	assert.Equal(t,
		reflect.TypeOf(dbschema.NewSQLEmitterForDialect("postgres")).String(),
		reflect.TypeOf(db.NewSQLEmitterForDialect("postgres")).String())

	dialects := map[string]struct {
		viaDB     db.DialectConfig
		viaSchema dbschema.DialectConfig
	}{
		"postgres":  {db.PostgresDialect(), dbschema.PostgresDialect()},
		"pgbouncer": {db.PostgresPgBouncerDialect(), dbschema.PostgresPgBouncerDialect()},
		"mysql":     {db.MySQLDialect(), dbschema.MySQLDialect()},
		"sqlite":    {db.SQLiteDialect(), dbschema.SQLiteDialect()},
	}

	for name, pair := range dialects {
		assert.Equal(t, pair.viaSchema.HistoryTableName, pair.viaDB.HistoryTableName,
			"%s must forward to its own dialect", name)
		assert.Equal(t,
			reflect.TypeOf(pair.viaSchema.LockStrategy).String(),
			reflect.TypeOf(pair.viaDB.LockStrategy).String(),
			"%s must forward to its own lock strategy", name)
	}

	withDSN := db.MySQLDialectWithDSN("user:pass@tcp(localhost:3306)/db?multiStatements=true")
	assert.Equal(t,
		dbschema.MySQLDialectWithDSN("user:pass@tcp(localhost:3306)/db?multiStatements=true").SplitStatements,
		withDSN.SplitStatements)
}

func TestConstructorForwardersBuildTheRightTypes(t *testing.T) {
	t.Parallel()

	fileReader := db.NewFSFileReader(fstest.MapFS{})
	require.NotNil(t, fileReader)

	database := &sql.DB{}
	dialect := db.SQLiteDialect()

	executor := db.NewMigrationExecutor(database, dialect)
	seedExecutor := db.NewSeedExecutor(database, dialect)
	require.NotNil(t, executor)
	require.NotNil(t, seedExecutor)

	assert.Equal(t,
		reflect.TypeOf(dbschema.NewMigrationExecutor(database, dialect)).String(),
		reflect.TypeOf(executor).String())
	assert.Equal(t,
		reflect.TypeOf(dbschema.NewSeedExecutor(database, dialect)).String(),
		reflect.TypeOf(seedExecutor).String())
	assert.NotEqual(t,
		reflect.TypeOf(executor).String(),
		reflect.TypeOf(seedExecutor).String(),
		"the migration and seed executors must not be the same type")

	assert.NotNil(t, db.NewMigrationService(executor, fileReader, "."))
	assert.NotNil(t, db.NewSeedService(seedExecutor, fileReader, "."))
	assert.NotNil(t, db.NewMigrationCatalogueProvider(nil, fileReader, "."))
	assert.NotNil(t, db.NewCompositeCatalogueProvider(nil))
}

func TestOptionForwardersAreEachDistinct(t *testing.T) {
	t.Parallel()

	options := map[string]db.MigrationServiceOption{
		"nonBlockingLock": db.WithNonBlockingLock(),
		"beforeMigration": db.WithBeforeMigration(nil),
		"afterMigration":  db.WithAfterMigration(nil),
		"beforeRun":       db.WithBeforeRun(nil),
		"afterRun":        db.WithAfterRun(nil),
	}

	for name, option := range options {
		assert.NotNil(t, option, "%s must return an option", name)
	}
}

func TestQuerierServiceForwarderSurfacesValidation(t *testing.T) {
	t.Parallel()

	service, err := db.NewQuerierService(db.QuerierPorts{})

	require.Error(t, err)
	assert.Nil(t, service)
}

func TestReservedDatabaseNamesAreDistinct(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, db.DatabaseNameRegistry)
	assert.NotEmpty(t, db.DatabaseNameOrchestrator)
	assert.NotEqual(t, db.DatabaseNameRegistry, db.DatabaseNameOrchestrator)
}
