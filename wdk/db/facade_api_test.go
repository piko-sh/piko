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
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
	"piko.sh/piko/wdk/db"
)

func TestDBFacadeAPI(t *testing.T) {
	surface := apitest.Surface{
		"AfterMigrationHook":        (*db.AfterMigrationHook)(nil),
		"AfterRunHook":              (*db.AfterRunHook)(nil),
		"AppliedMigration":          (*db.AppliedMigration)(nil),
		"AppliedSeed":               (*db.AppliedSeed)(nil),
		"BeforeMigrationHook":       (*db.BeforeMigrationHook)(nil),
		"BeforeRunHook":             (*db.BeforeRunHook)(nil),
		"CatalogueProviderPort":     (*db.CatalogueProviderPort)(nil),
		"ChecksumMismatchError":     (*db.ChecksumMismatchError)(nil),
		"CodeEmitterPort":           (*db.CodeEmitterPort)(nil),
		"CustomFunction":            (*db.CustomFunction)(nil),
		"CustomFunctionConfig":      (*db.CustomFunctionConfig)(nil),
		"DBTX":                      (*db.DBTX)(nil),
		"DatabaseConfig":            (*db.DatabaseConfig)(nil),
		"DatabaseRegistration":      (*db.DatabaseRegistration)(nil),
		"DialectConfig":             (*db.DialectConfig)(nil),
		"DownChecksumMismatchError": (*db.DownChecksumMismatchError)(nil),
		"EnginePort":                (*db.EnginePort)(nil),
		"FileReaderPort":            (*db.FileReaderPort)(nil),
		"GeneratedFile":             (*db.GeneratedFile)(nil),
		"GenerationResult":          (*db.GenerationResult)(nil),
		"LockAcquisitionError":      (*db.LockAcquisitionError)(nil),
		"MigrationDirection":        (*db.MigrationDirection)(nil),
		"MigrationExecutionError":   (*db.MigrationExecutionError)(nil),
		"MigrationExecutor":         (*db.MigrationExecutor)(nil),
		"MigrationFile":             (*db.MigrationFile)(nil),
		"MigrationHookContext":      (*db.MigrationHookContext)(nil),
		"MigrationRunHookContext":   (*db.MigrationRunHookContext)(nil),
		"MigrationService":          (*db.MigrationService)(nil),
		"MigrationServiceOption":    (*db.MigrationServiceOption)(nil),
		"MigrationStatus":           (*db.MigrationStatus)(nil),
		"MissingMigrationFileError": (*db.MissingMigrationFileError)(nil),
		"NoDownMigrationError":      (*db.NoDownMigrationError)(nil),
		"QuerierPorts":              (*db.QuerierPorts)(nil),
		"QuerierService":            (*db.QuerierService)(nil),
		"QuerierServicePort":        (*db.QuerierServicePort)(nil),
		"Replica":                   (*db.Replica)(nil),
		"SeedExecutorPort":          (*db.SeedExecutorPort)(nil),
		"SeedService":               (*db.SeedService)(nil),
		"SeedStatus":                (*db.SeedStatus)(nil),
		"Severity":                  (*db.Severity)(nil),
		"SourceError":               (*db.SourceError)(nil),
		"TypeOverride":              (*db.TypeOverride)(nil),

		"DatabaseNameOrchestrator": db.DatabaseNameOrchestrator,
		"DatabaseNameRegistry":     db.DatabaseNameRegistry,
		"DirectionDown":            db.DirectionDown,
		"DirectionUp":              db.DirectionUp,
		"ErrLockNotAcquired":       db.ErrLockNotAcquired,
		"SeverityError":            db.SeverityError,
		"SeverityHint":             db.SeverityHint,
		"SeverityWarning":          db.SeverityWarning,

		"GetDatabaseConnection":         db.GetDatabaseConnection,
		"GetDatabaseReader":             db.GetDatabaseReader,
		"GetDatabaseWriter":             db.GetDatabaseWriter,
		"GetMigrationService":           db.GetMigrationService,
		"GetSeedService":                db.GetSeedService,
		"MySQLDialect":                  db.MySQLDialect,
		"MySQLDialectWithDSN":           db.MySQLDialectWithDSN,
		"NewCompositeCatalogueProvider": db.NewCompositeCatalogueProvider,
		"NewFSFileReader":               db.NewFSFileReader,
		"NewMigrationCatalogueProvider": db.NewMigrationCatalogueProvider,
		"NewMigrationExecutor":          db.NewMigrationExecutor,
		"NewMigrationService":           db.NewMigrationService,
		"NewQuerierService":             db.NewQuerierService,
		"NewSQLEmitter":                 db.NewSQLEmitter,
		"NewSQLEmitterForDialect":       db.NewSQLEmitterForDialect,
		"NewSeedExecutor":               db.NewSeedExecutor,
		"NewSeedService":                db.NewSeedService,
		"PostgresDialect":               db.PostgresDialect,
		"PostgresPgBouncerDialect":      db.PostgresPgBouncerDialect,
		"SQLiteDialect":                 db.SQLiteDialect,
		"WithAfterMigration":            db.WithAfterMigration,
		"WithAfterRun":                  db.WithAfterRun,
		"WithBeforeMigration":           db.WithBeforeMigration,
		"WithBeforeRun":                 db.WithBeforeRun,
		"WithDatabase":                  db.WithDatabase,
		"WithNonBlockingLock":           db.WithNonBlockingLock,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
