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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package db_engine_postgres implements the querier EnginePort
// for PostgreSQL and PostgreSQL-compatible databases.
//
// It uses a hand-written recursive-descent parser to convert
// DDL into catalogue mutations and analyse DML queries for
// type resolution. The PostgresDialect option pattern allows
// variants such as CockroachDB, YugabyteDB, and TimescaleDB
// to customise types, functions, and semantic rules without
// forking the parser. The package also provides health
// diagnostics covering database size, active connections,
// recovery state, and replication lag.
package db_engine_postgres
