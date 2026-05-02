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

package db_engine_timescaledb

import (
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/wdk/db"
)

// TimescaleDB returns an EngineConfig for TimescaleDB databases.
//
// The driver name is "postgres" because TimescaleDB is wire-compatible with postgres, and
// the migration dialect is reused unchanged from postgres because TimescaleDB inherits
// postgres transactions and advisory locks.
//
// Returns db.EngineConfig which wires the TimescaleDB engine, postgres driver, and
// postgres migration dialect together.
func TimescaleDB() db.EngineConfig {
	return db.EngineConfig{
		DriverName:       "postgres",
		Engine:           NewTimescaleDBEngine(),
		MigrationDialect: migration_sql.PostgresDialect(),
	}
}
