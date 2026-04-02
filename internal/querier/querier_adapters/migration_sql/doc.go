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

// Package migration_sql implements MigrationExecutorPort using
// database/sql. It handles migration history tracking, advisory
// locking, and per-migration transaction wrapping for both SQLite and
// PostgreSQL via dialect-specific configuration.
//
// The executor assumes the database/sql driver supports multi-statement
// execution in a single ExecContext call. Known compatible drivers
// include lib/pq and pgx/stdlib for PostgreSQL, and modernc.org/sqlite
// and mattn/go-sqlite3 for SQLite. If a driver does not support this,
// individual migration files must contain a single statement each.
//
// Migrations using the -- piko:no-transaction directive bypass
// transaction wrapping. If the process crashes after the SQL executes
// but before the history record is committed, the migration will not
// be recorded and will be treated as pending on the next run. For this
// reason, no-transaction migrations must be idempotent.
package migration_sql
