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

// NewTimescaleDBEngine creates a TimescaleDB engine adapter by configuring the postgres
// engine with TimescaleDB-specific extensions, type aliases, and function registrations.
//
// The returned engine is a fully-wired *PostgresEngine; downstream consumers treat it as
// any other postgres-flavoured engine.
//
// TimescaleDB-specific DDL parsing is performed by the registered StatementExtension for
// shapes the postgres classifier does not recognise (CREATE HYPERTABLE,
// continuous-aggregate materialised views, ALTER ... SET compression bodies, CALL
// refresh_continuous_aggregate, SELECT policy and hypertable-management calls). A
// PostParseHook covers the modern CREATE TABLE foo (...) WITH (tsdb.hypertable, ...) form
// because the host postgres parser already routes that statement through its built-in
// CREATE TABLE handler; the hook lifts the trailing reloption body without
// re-implementing column parsing.
//
// The migration dialect is not configured here. Callers that need a migration runner (for
// example via the TimescaleDB EngineConfig in config.go) reuse the postgres
// MigrationDialect unchanged because TimescaleDB inherits postgres transactions, advisory
// locks, and migration metadata tables.
//
// Returns *db_engine_postgres.PostgresEngine which is the wired TimescaleDB-flavoured
// postgres engine.
func NewTimescaleDBEngine() *db_engine_postgres.PostgresEngine {
	return db_engine_postgres.NewPostgresEngine(
		db_engine_postgres.WithDialectName("timescaledb"),
		db_engine_postgres.WithExtraTypes(timescaleDBTypes()),
		db_engine_postgres.WithExtraFunctions(registerTimescaleDBFunctions),
		db_engine_postgres.WithStatementExtensions(newTimescaleExtension()),
		db_engine_postgres.WithPostParseHook(timescaleCreateTableHypertableHook),
	)
}
