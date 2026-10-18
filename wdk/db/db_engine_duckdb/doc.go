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

// Package db_engine_duckdb is Piko's engine for DuckDB.
//
// Register it with a database to generate typed Go from SQL, run migrations and seeds,
// and report engine-specific health diagnostics:
//
//	piko.WithDatabase("primary", &db.DatabaseRegistration{
//	    DB:           database,
//	    EngineConfig: db_engine_duckdb.DuckDB(),
//	})
//
// The SQL analysis and code generation are provided by Seagull through
// wdk/db/db_provider_seagull; db_engine_duckdb selects and tunes the dialect.
package db_engine_duckdb
