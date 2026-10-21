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

// Package db handles database registration, migrations, seeds and SQL code generation for
// Piko applications, with pluggable engine backends for different SQL dialects.
//
// Register a database with an engine from one of the db_engine_* sub-packages. The engine
// supplies the toolchain that analyses SQL, generates typed Go and runs migrations;
// Piko's build runs generation for every registered database that has query files.
//
// # Usage
//
// Running migrations outside the framework:
//
//	migrator, err := db.NewMigrator(db_engine_postgres.Postgres(), sqlDB, migrationsFS, ".")
//	if err != nil {
//	    return err
//	}
//	applied, err := migrator.Up(ctx)
//
// Using lifecycle hooks:
//
//	migrator, err := db.NewMigrator(engine, sqlDB, migrationsFS, ".",
//	    db.WithBeforeMigration(func(ctx context.Context, hook db.MigrationHookContext) error {
//	        log.Printf("applying %s (v%d)", hook.Name, hook.Version)
//	        return nil
//	    }),
//	)
//
// # Thread safety
//
// [MigrationService] uses locking to prevent concurrent migration runs. The service
// itself is safe for concurrent use.
package db
