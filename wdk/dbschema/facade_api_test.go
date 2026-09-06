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
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
	"piko.sh/piko/wdk/dbschema"
)

func TestDBSchemaFacadeAPI(t *testing.T) {
	surface := apitest.Surface{
		"AfterMigrationHook":        (*dbschema.AfterMigrationHook)(nil),
		"AfterRunHook":              (*dbschema.AfterRunHook)(nil),
		"AppliedMigration":          (*dbschema.AppliedMigration)(nil),
		"AppliedSeed":               (*dbschema.AppliedSeed)(nil),
		"BeforeMigrationHook":       (*dbschema.BeforeMigrationHook)(nil),
		"BeforeRunHook":             (*dbschema.BeforeRunHook)(nil),
		"CatalogueProviderPort":     (*dbschema.CatalogueProviderPort)(nil),
		"ChecksumMismatchError":     (*dbschema.ChecksumMismatchError)(nil),
		"CodeEmitterPort":           (*dbschema.CodeEmitterPort)(nil),
		"CustomFunction":            (*dbschema.CustomFunction)(nil),
		"CustomFunctionConfig":      (*dbschema.CustomFunctionConfig)(nil),
		"DatabaseConfig":            (*dbschema.DatabaseConfig)(nil),
		"DialectConfig":             (*dbschema.DialectConfig)(nil),
		"DownChecksumMismatchError": (*dbschema.DownChecksumMismatchError)(nil),
		"EnginePort":                (*dbschema.EnginePort)(nil),
		"FileReaderPort":            (*dbschema.FileReaderPort)(nil),
		"GeneratedFile":             (*dbschema.GeneratedFile)(nil),
		"GenerationResult":          (*dbschema.GenerationResult)(nil),
		"LockAcquisitionError":      (*dbschema.LockAcquisitionError)(nil),
		"MigrationDirection":        (*dbschema.MigrationDirection)(nil),
		"MigrationExecutionError":   (*dbschema.MigrationExecutionError)(nil),
		"MigrationExecutor":         (*dbschema.MigrationExecutor)(nil),
		"MigrationFile":             (*dbschema.MigrationFile)(nil),
		"MigrationHookContext":      (*dbschema.MigrationHookContext)(nil),
		"MigrationRunHookContext":   (*dbschema.MigrationRunHookContext)(nil),
		"MigrationService":          (*dbschema.MigrationService)(nil),
		"MigrationServiceOption":    (*dbschema.MigrationServiceOption)(nil),
		"MigrationStatus":           (*dbschema.MigrationStatus)(nil),
		"MissingMigrationFileError": (*dbschema.MissingMigrationFileError)(nil),
		"NoDownMigrationError":      (*dbschema.NoDownMigrationError)(nil),
		"QuerierPorts":              (*dbschema.QuerierPorts)(nil),
		"QuerierService":            (*dbschema.QuerierService)(nil),
		"QuerierServicePort":        (*dbschema.QuerierServicePort)(nil),
		"SeedExecutorPort":          (*dbschema.SeedExecutorPort)(nil),
		"SeedService":               (*dbschema.SeedService)(nil),
		"SeedStatus":                (*dbschema.SeedStatus)(nil),
		"Severity":                  (*dbschema.Severity)(nil),
		"SourceError":               (*dbschema.SourceError)(nil),
		"TypeOverride":              (*dbschema.TypeOverride)(nil),

		"DirectionDown":      dbschema.DirectionDown,
		"DirectionUp":        dbschema.DirectionUp,
		"SeverityError":      dbschema.SeverityError,
		"SeverityHint":       dbschema.SeverityHint,
		"SeverityWarning":    dbschema.SeverityWarning,
		"ErrLockNotAcquired": dbschema.ErrLockNotAcquired,

		"MySQLDialect":                  dbschema.MySQLDialect,
		"MySQLDialectWithDSN":           dbschema.MySQLDialectWithDSN,
		"NewCompositeCatalogueProvider": dbschema.NewCompositeCatalogueProvider,
		"NewFSFileReader":               dbschema.NewFSFileReader,
		"NewMigrationCatalogueProvider": dbschema.NewMigrationCatalogueProvider,
		"NewMigrationExecutor":          dbschema.NewMigrationExecutor,
		"NewMigrationService":           dbschema.NewMigrationService,
		"NewQuerierService":             dbschema.NewQuerierService,
		"NewSQLEmitter":                 dbschema.NewSQLEmitter,
		"NewSQLEmitterForDialect":       dbschema.NewSQLEmitterForDialect,
		"NewSeedExecutor":               dbschema.NewSeedExecutor,
		"NewSeedService":                dbschema.NewSeedService,
		"PostgresDialect":               dbschema.PostgresDialect,
		"PostgresPgBouncerDialect":      dbschema.PostgresPgBouncerDialect,
		"SQLiteDialect":                 dbschema.SQLiteDialect,
		"WithAfterMigration":            dbschema.WithAfterMigration,
		"WithAfterRun":                  dbschema.WithAfterRun,
		"WithBeforeMigration":           dbschema.WithBeforeMigration,
		"WithBeforeRun":                 dbschema.WithBeforeRun,
		"WithNonBlockingLock":           dbschema.WithNonBlockingLock,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
