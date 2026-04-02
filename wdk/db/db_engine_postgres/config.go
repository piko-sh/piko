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

package db_engine_postgres

import (
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/wdk/db"
)

// Postgres returns an EngineConfig for PostgreSQL databases with advisory
// lock-based migration locking.
func Postgres() db.EngineConfig {
	return db.EngineConfig{
		DriverName:       "postgres",
		Engine:           NewPostgresEngine(),
		MigrationDialect: migration_sql.PostgresDialect(),
	}
}

// PostgresPgBouncer returns an EngineConfig for PostgreSQL databases behind
// PgBouncer in transaction mode, using table-based locking instead of
// advisory locks.
func PostgresPgBouncer() db.EngineConfig {
	return db.EngineConfig{
		DriverName:       "postgres",
		Engine:           NewPostgresEngine(),
		MigrationDialect: migration_sql.PostgresPgBouncerDialect(),
	}
}
