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
	"piko.sh/piko/wdk/db/db_engine_postgres"
)

// NewTimescaleDBEngine returns the TimescaleDB engine, a member of the PostgreSQL family
// that shares its parser, type system and migration backend.
//
// Takes options (...db_engine_postgres.Option) which tune the engine.
//
// Returns *db_engine_postgres.PostgresEngine which serves the TimescaleDB dialect.
func NewTimescaleDBEngine(options ...db_engine_postgres.Option) *db_engine_postgres.PostgresEngine {
	return db_engine_postgres.NewEngineForDialect("timescaledb", options...)
}
