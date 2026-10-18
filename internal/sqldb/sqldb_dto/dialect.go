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

package sqldb_dto

// DialectFamily names the migration backend a database uses.
type DialectFamily string

const (
	// DialectFamilyPostgres is PostgreSQL and its wire-compatible databases.
	DialectFamilyPostgres DialectFamily = "postgres"

	// DialectFamilyCockroachDB is CockroachDB, which speaks the PostgreSQL protocol but has
	// no advisory locks, so its migrations take a table lock.
	DialectFamilyCockroachDB DialectFamily = "cockroachdb"

	// DialectFamilyMySQL is MySQL and MariaDB.
	DialectFamilyMySQL DialectFamily = "mysql"

	// DialectFamilySQLite is SQLite and Cloudflare D1.
	DialectFamilySQLite DialectFamily = "sqlite"

	// DialectFamilyClickHouse is ClickHouse.
	DialectFamilyClickHouse DialectFamily = "clickhouse"
)

// DialectConfig selects and tunes the migration backend for one database. The zero value
// means the database has no migration backend.
type DialectConfig struct {
	// Family names the migration backend.
	Family DialectFamily

	// DSN is the connection string, which the MySQL backend inspects to decide whether the
	// driver accepts multiple statements per call.
	DSN string

	// HistoryTable overrides the migration history table name and its lock key. An empty
	// value keeps the default.
	HistoryTable string

	// PgBouncer selects table-based locking for PostgreSQL behind PgBouncer in transaction
	// mode, where advisory locks are unavailable.
	PgBouncer bool
}

// IsConfigured reports whether the descriptor names a migration backend.
//
// Returns bool which is true when Family is set.
func (d DialectConfig) IsConfigured() bool {
	return d.Family != ""
}
