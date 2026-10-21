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
		"AfterMigrationHook":             (*db.AfterMigrationHook)(nil),
		"AfterRunHook":                   (*db.AfterRunHook)(nil),
		"AppliedMigration":               (*db.AppliedMigration)(nil),
		"AppliedSeed":                    (*db.AppliedSeed)(nil),
		"BeforeMigrationHook":            (*db.BeforeMigrationHook)(nil),
		"BeforeRunHook":                  (*db.BeforeRunHook)(nil),
		"ChecksumMismatchError":          (*db.ChecksumMismatchError)(nil),
		"CustomFunction":                 (*db.CustomFunction)(nil),
		"DBTX":                           (*db.DBTX)(nil),
		"DatabaseHealthChecker":          (*db.DatabaseHealthChecker)(nil),
		"DatabaseHealthDiagnostic":       (*db.DatabaseHealthDiagnostic)(nil),
		"DatabaseRegistration":           (*db.DatabaseRegistration)(nil),
		"DialectConfig":                  (*db.DialectConfig)(nil),
		"DialectFamily":                  (*db.DialectFamily)(nil),
		"DirtyMigrationError":            (*db.DirtyMigrationError)(nil),
		"DownChecksumMismatchError":      (*db.DownChecksumMismatchError)(nil),
		"DuplicateMigrationVersionError": (*db.DuplicateMigrationVersionError)(nil),
		"DuplicateSeedVersionError":      (*db.DuplicateSeedVersionError)(nil),
		"EngineConfig":                   (*db.EngineConfig)(nil),
		"GenerateRequest":                (*db.GenerateRequest)(nil),
		"GeneratedDriver":                (*db.GeneratedDriver)(nil),
		"GeneratedFile":                  (*db.GeneratedFile)(nil),
		"GenerationResult":               (*db.GenerationResult)(nil),
		"LockAcquisitionError":           (*db.LockAcquisitionError)(nil),
		"MigrationDirection":             (*db.MigrationDirection)(nil),
		"MigrationExecutionError":        (*db.MigrationExecutionError)(nil),
		"MigrationHookContext":           (*db.MigrationHookContext)(nil),
		"MigrationRunHookContext":        (*db.MigrationRunHookContext)(nil),
		"MigrationService":               (*db.MigrationService)(nil),
		"MigrationServiceOption":         (*db.MigrationServiceOption)(nil),
		"MigrationStatus":                (*db.MigrationStatus)(nil),
		"MissingMigrationFileError":      (*db.MissingMigrationFileError)(nil),
		"NoDownMigrationError":           (*db.NoDownMigrationError)(nil),
		"Replica":                        (*db.Replica)(nil),
		"SeedExecutionError":             (*db.SeedExecutionError)(nil),
		"SeedService":                    (*db.SeedService)(nil),
		"SeedStatus":                     (*db.SeedStatus)(nil),
		"Severity":                       (*db.Severity)(nil),
		"SourceError":                    (*db.SourceError)(nil),
		"TypeOverride":                   (*db.TypeOverride)(nil),

		"DatabaseNameOrchestrator": db.DatabaseNameOrchestrator,
		"DatabaseNameRegistry":     db.DatabaseNameRegistry,
		"DialectFamilyClickHouse":  db.DialectFamilyClickHouse,
		"DialectFamilyMySQL":       db.DialectFamilyMySQL,
		"DialectFamilyPostgres":    db.DialectFamilyPostgres,
		"DialectFamilyCockroachDB": db.DialectFamilyCockroachDB,
		"DialectFamilySQLite":      db.DialectFamilySQLite,
		"DirectionDown":            db.DirectionDown,
		"DirectionUp":              db.DirectionUp,
		"DriverClickHouseNative":   db.DriverClickHouseNative,
		"DriverDatabaseSQL":        db.DriverDatabaseSQL,
		"DriverPgx":                db.DriverPgx,
		"ErrInvalidIdentifier":     db.ErrInvalidIdentifier,
		"ErrLockNotAcquired":       db.ErrLockNotAcquired,
		"ErrMalformedSQLStatement": db.ErrMalformedSQLStatement,
		"ErrMigrationFileTooLarge": db.ErrMigrationFileTooLarge,
		"ErrNoDownMigration":       db.ErrNoDownMigration,
		"ErrNoMigrationBackend":    db.ErrNoMigrationBackend,
		"ErrNoSQLToolchain":        db.ErrNoSQLToolchain,
		"ErrUnsupportedDriver":     db.ErrUnsupportedDriver,
		"SeverityError":            db.SeverityError,
		"SeverityHint":             db.SeverityHint,
		"SeverityWarning":          db.SeverityWarning,

		"ClickHouseDialect":        db.ClickHouseDialect,
		"Generate":                 db.Generate,
		"GetDatabaseConnection":    db.GetDatabaseConnection,
		"GetDatabaseReader":        db.GetDatabaseReader,
		"GetDatabaseWriter":        db.GetDatabaseWriter,
		"GetMigrationService":      db.GetMigrationService,
		"GetSeedService":           db.GetSeedService,
		"MySQLDialect":             db.MySQLDialect,
		"MySQLDialectWithDSN":      db.MySQLDialectWithDSN,
		"NewMigrator":              db.NewMigrator,
		"NewSeeder":                db.NewSeeder,
		"PostgresDialect":          db.PostgresDialect,
		"PostgresPgBouncerDialect": db.PostgresPgBouncerDialect,
		"CockroachDBDialect":       db.CockroachDBDialect,
		"SQLiteDialect":            db.SQLiteDialect,
		"WithAfterMigration":       db.WithAfterMigration,
		"WithAfterRun":             db.WithAfterRun,
		"WithBeforeMigration":      db.WithBeforeMigration,
		"WithBeforeRun":            db.WithBeforeRun,
		"WithDatabase":             db.WithDatabase,
		"WithNonBlockingLock":      db.WithNonBlockingLock,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
