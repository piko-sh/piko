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

package migration_sql

import (
	"fmt"
	"strings"
)

// DialectConfig holds dialect-specific SQL and behaviour for the migration
// executor. Each supported database engine provides a pre-built config via
// PostgresDialect() or SQLiteDialect().
type DialectConfig struct {
	// LockStrategy provides database-specific advisory locking.
	LockStrategy LockStrategy

	// PlaceholderFunc converts a 1-based parameter index to the dialect's
	// placeholder syntax ("$1" for PostgreSQL, "?" for SQLite).
	PlaceholderFunc func(index int) string

	// CreateTableSQL is the DDL statement for creating the piko_migrations
	// history table. Includes all columns: version, name, checksum,
	// applied_at, duration_ms, down_checksum, last_statement, dirty.
	CreateTableSQL string

	// CreateSeedTableSQL is the DDL statement for creating the piko_seeds
	// history table. Empty when seed tracking is not configured.
	CreateSeedTableSQL string

	// PreMigrationStatements holds SQL statements executed on the pinned
	// connection after lock acquisition and before any migrations run.
	// Typical uses include SET ROLE, SET search_path, or
	// SET statement_timeout.
	PreMigrationStatements []string

	// AlterStatements holds SQL statements executed after CREATE TABLE to
	// evolve the migration table schema. Duplicate column errors are
	// suppressed so the statements are idempotent.
	AlterStatements []string

	// SplitStatements controls whether migration SQL is split on semicolons
	// and executed as individual statements. Required for MySQL which does
	// not support multi-statement execution by default.
	SplitStatements bool
}

// PostgresDialect returns a DialectConfig for PostgreSQL databases.
//
// Returns DialectConfig which is configured with PostgreSQL-specific SQL,
// advisory locking, and $N placeholder syntax.
func PostgresDialect() DialectConfig {
	return DialectConfig{
		CreateTableSQL: `CREATE TABLE IF NOT EXISTS piko_migrations (
    version        BIGINT      NOT NULL PRIMARY KEY,
    name           TEXT        NOT NULL,
    checksum       TEXT        NOT NULL,
    applied_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms    BIGINT      NOT NULL,
    down_checksum  TEXT,
    last_statement INTEGER,
    dirty          BOOLEAN     NOT NULL DEFAULT FALSE
)`,
		CreateSeedTableSQL: `CREATE TABLE IF NOT EXISTS piko_seeds (
    version     BIGINT      NOT NULL PRIMARY KEY,
    name        TEXT        NOT NULL,
    checksum    TEXT        NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms BIGINT      NOT NULL
)`,
		LockStrategy: &PostgresAdvisoryLock{},
		PlaceholderFunc: func(index int) string {
			return fmt.Sprintf("$%d", index)
		},
	}
}

// PostgresPgBouncerDialect returns a DialectConfig for PostgreSQL databases
// behind PgBouncer in transaction mode.
//
// Advisory locks are not available in this configuration, so a table-based lock
// via SELECT ... FOR UPDATE is used instead.
//
// Returns DialectConfig which is configured with PostgreSQL-specific SQL,
// table-based locking, and $N placeholder syntax.
func PostgresPgBouncerDialect() DialectConfig {
	return DialectConfig{
		CreateTableSQL: `CREATE TABLE IF NOT EXISTS piko_migrations (
    version        BIGINT      NOT NULL PRIMARY KEY,
    name           TEXT        NOT NULL,
    checksum       TEXT        NOT NULL,
    applied_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms    BIGINT      NOT NULL,
    down_checksum  TEXT,
    last_statement INTEGER,
    dirty          BOOLEAN     NOT NULL DEFAULT FALSE
)`,
		CreateSeedTableSQL: `CREATE TABLE IF NOT EXISTS piko_seeds (
    version     BIGINT      NOT NULL PRIMARY KEY,
    name        TEXT        NOT NULL,
    checksum    TEXT        NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms BIGINT      NOT NULL
)`,
		LockStrategy: &TableBasedLock{
			CreateLockTableSQL: `CREATE TABLE IF NOT EXISTS piko_migration_lock (
    lock_id INTEGER NOT NULL PRIMARY KEY DEFAULT 1,
    CONSTRAINT piko_migration_lock_single_row CHECK (lock_id = 1)
)`,
		},
		PlaceholderFunc: func(index int) string {
			return fmt.Sprintf("$%d", index)
		},
	}
}

// MySQLDialect returns a DialectConfig for MySQL databases.
//
// Returns DialectConfig which is configured with MySQL-specific SQL,
// advisory locking, and ? placeholder syntax. SplitStatements is enabled
// since MySQL does not support multi-statement execution by default.
func MySQLDialect() DialectConfig {
	return DialectConfig{
		CreateTableSQL: `CREATE TABLE IF NOT EXISTS piko_migrations (
    version        BIGINT       NOT NULL PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    checksum       VARCHAR(64)  NOT NULL,
    applied_at     TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    duration_ms    BIGINT       NOT NULL,
    down_checksum  VARCHAR(64),
    last_statement INTEGER,
    dirty          BOOLEAN      NOT NULL DEFAULT FALSE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		CreateSeedTableSQL: `CREATE TABLE IF NOT EXISTS piko_seeds (
    version     BIGINT       NOT NULL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    checksum    VARCHAR(64)  NOT NULL,
    applied_at  TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    duration_ms BIGINT       NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		LockStrategy:    &MySQLAdvisoryLock{},
		PlaceholderFunc: func(_ int) string { return "?" },
		SplitStatements: true,
	}
}

// MySQLDialectWithDSN returns a DialectConfig for MySQL databases, detecting
// whether the DSN already includes multiStatements=true. When the driver is
// configured to handle multi-statement execution natively, the framework
// disables its own statement splitting to avoid interfering with stored
// procedures or complex migration SQL.
//
// Takes dsn (string) which is the MySQL data source name.
//
// Returns DialectConfig which is configured for MySQL with automatic
// SplitStatements detection.
func MySQLDialectWithDSN(dsn string) DialectConfig {
	dialect := MySQLDialect()
	if strings.Contains(dsn, "multiStatements=true") {
		dialect.SplitStatements = false
	}
	return dialect
}

// SQLiteDialect returns a DialectConfig for SQLite databases.
//
// Returns DialectConfig which is configured with SQLite-specific SQL,
// no-op locking, and ? placeholder syntax.
func SQLiteDialect() DialectConfig {
	return DialectConfig{
		CreateTableSQL: `CREATE TABLE IF NOT EXISTS piko_migrations (
    version        INTEGER NOT NULL PRIMARY KEY,
    name           TEXT    NOT NULL,
    checksum       TEXT    NOT NULL,
    applied_at     TEXT    NOT NULL DEFAULT (datetime('now')),
    duration_ms    INTEGER NOT NULL,
    down_checksum  TEXT,
    last_statement INTEGER,
    dirty          INTEGER NOT NULL DEFAULT 0
)`,
		CreateSeedTableSQL: `CREATE TABLE IF NOT EXISTS piko_seeds (
    version     INTEGER NOT NULL PRIMARY KEY,
    name        TEXT    NOT NULL,
    checksum    TEXT    NOT NULL,
    applied_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    duration_ms INTEGER NOT NULL
)`,
		LockStrategy:    &NoOpLock{},
		PlaceholderFunc: func(_ int) string { return "?" },
	}
}
