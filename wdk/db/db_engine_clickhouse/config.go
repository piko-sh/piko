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

package db_engine_clickhouse

import (
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/wdk/db"
)

// ClickHouse returns an EngineConfig for ClickHouse databases.
//
// The returned config supports both codegen against the native protocol driver
// `github.com/ClickHouse/clickhouse-go/v2` and runtime migration execution via the
// ClickHouse migration dialect.
//
// Returns db.EngineConfig which wires the ClickHouse engine, driver, and migration
// dialect.
func ClickHouse() db.EngineConfig {
	return db.EngineConfig{
		DriverName:       "clickhouse",
		Engine:           NewClickHouseEngine(),
		MigrationDialect: migration_sql.ClickHouseDialect(),
	}
}
