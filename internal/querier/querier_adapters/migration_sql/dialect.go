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
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const (
	// DefaultHistoryTableName is the table name used when DialectConfig HistoryTableName is
	// empty.
	//
	// Callers customise this via DialectConfig.WithHistoryTable to coexist multiple
	// migration timelines against the same database (for example one per hexagon).
	DefaultHistoryTableName = "piko_migrations"
)

// DialectConfig holds dialect-specific SQL and behaviour for the migration executor. Each
// supported database engine provides a pre-built config via PostgresDialect() or
// SQLiteDialect().
type DialectConfig struct {
	// HistoryTableName is the name of the migration history table.
	//
	// Defaults to DefaultHistoryTableName ("piko_migrations") when constructed via the
	// per-dialect helpers. Override via WithHistoryTable so multiple MigrationService
	// instances can coexist against the same database (for example one per hexagon) without
	// colliding on history rows or advisory locks.
	HistoryTableName string

	// LockStrategy provides database-specific advisory locking for migrations.
	LockStrategy LockStrategy

	// SeedLockStrategy provides database-specific advisory locking for seeds.
	//
	// It must use a distinct key from LockStrategy so seed and migration runs can serialise
	// independently. When nil the seed executor falls back to a no-op lock, which is correct
	// only for single-replica deployments.
	SeedLockStrategy LockStrategy

	// PlaceholderFunc converts a 1-based parameter index to the dialect's parameter-marker
	// syntax ("$1" for PostgreSQL, "?" for SQLite).
	PlaceholderFunc func(index int) string

	// InsertSeedSQLFunc builds the dialect-specific INSERT statement for recording an
	// applied seed.
	//
	// The returned SQL must be idempotent: a re-application of an already-recorded seed must
	// succeed without raising a primary-key violation. PostgreSQL/SQLite use "ON CONFLICT
	// (version) DO NOTHING"; MySQL uses "INSERT IGNORE".
	//
	// Takes versionPlaceholder (string) which is the parameter marker for the version
	// column, e.g. "$1" or "?".
	// Takes namePlaceholder (string) which is the parameter marker for the name column.
	// Takes checksumPlaceholder (string) which is the parameter marker for the checksum
	// column.
	// Takes durationPlaceholder (string) which is the parameter marker for the duration_ms
	// column.
	//
	// Returns string which is the complete INSERT statement.
	InsertSeedSQLFunc func(versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder string) string

	// DeleteHistorySQLFunc builds the dialect-specific statement that removes a single
	// migration history row by version (used on down-migration / rollback).
	//
	// When nil the executor falls back to a standard "DELETE FROM <table> WHERE version =
	// <placeholder>". ClickHouse MergeTree rejects plain DELETE, so it supplies an "ALTER
	// TABLE ... DELETE WHERE ..." mutation instead.
	//
	// Takes historyTable (string) which is the validated history table identifier.
	// Takes versionPlaceholder (string) which is the parameter marker for the version
	// column, e.g. "$1" or "?".
	//
	// Returns string which is the complete delete statement.
	DeleteHistorySQLFunc func(historyTable, versionPlaceholder string) string

	// CreateTableSQL is the DDL statement for creating the migration history table. The
	// constructors render it against HistoryTableName; WithHistoryTable rewrites it when the
	// table is renamed.
	CreateTableSQL string

	// CreateSeedTableSQL is the DDL statement for creating the piko_seeds history table.
	// Empty when seed tracking is not configured.
	CreateSeedTableSQL string

	// PreMigrationStatements holds SQL statements executed on the pinned connection after
	// lock acquisition and before any migrations run. Typical uses include SET ROLE, SET
	// search_path, or SET statement_timeout.
	PreMigrationStatements []string

	// AlterStatements holds SQL statements executed after CREATE TABLE to evolve the
	// migration table schema. Duplicate column errors are suppressed so the statements are
	// idempotent.
	AlterStatements []string

	// SplitStatements controls whether migration SQL is split on semicolons and executed as
	// individual statements. Required for MySQL which does not support multi-statement
	// execution by default.
	SplitStatements bool

	// DisableTransactions forces every migration onto the non-transactional execution path
	// regardless of the per-migration useTransaction request.
	//
	// ClickHouse has no DDL transactions, so it sets this true. The zero value (false)
	// preserves the historical behaviour for every other dialect, so existing constructors
	// need no change.
	DisableTransactions bool

	// AppendOnlyHistory selects an append-only history model for engines that cannot UPDATE
	// rows in place, such as ClickHouse MergeTree.
	//
	// When true, up-migrations skip the dirty pre-record, per-statement progress, and
	// clear-dirty machinery and instead record a single completed history row on success;
	// down-migrations remove the row via DeleteHistorySQLFunc. The zero value (false) keeps
	// the resumable dirty-tracking path used by every transactional or UPDATE-capable
	// engine.
	AppendOnlyHistory bool

	// SelectHistoryFinal appends a FINAL modifier to the applied-versions SELECT so a
	// ReplacingMergeTree history table collapses duplicate version rows at read time. Only
	// ClickHouse sets this; the zero value leaves the SELECT untouched.
	SelectHistoryFinal bool

	// BackslashEscapes tells the statement splitter that the dialect treats a backslash
	// inside a single-quoted string literal as an escape character (ClickHouse, MySQL).
	// Standard-SQL dialects (PostgreSQL, SQLite) leave this false so a backslash is an
	// ordinary character and only a doubled quote escapes.
	BackslashEscapes bool
}

// HistoryTable returns the configured history table name, falling back to
// DefaultHistoryTableName when the field is empty so older configs and zero-value structs
// keep working.
//
// Returns string which is the history table name to use in DDL/DML.
func (d DialectConfig) HistoryTable() string {
	if d.HistoryTableName == "" {
		return DefaultHistoryTableName
	}
	return d.HistoryTableName
}

var (
	// ErrInvalidIdentifier is returned by WithHistoryTable when the supplied name is not a
	// safe SQL identifier. Callers can branch on it via errors.Is to distinguish
	// input-validation failures from runtime errors.
	ErrInvalidIdentifier = errors.New("invalid SQL identifier")

	// safeIdentifierRegex matches the conservative subset of SQL identifier syntax Piko
	// accepts for the migration history table name and lock keys.
	//
	// The match requires an ASCII letter or underscore start, then ASCII letters, digits, or
	// underscores, bounded to 63 characters (Postgres NAMEDATALEN minus one byte for the
	// trailing terminator). Quoted identifiers are deliberately rejected so no
	// dialect-specific quoting needs to be performed downstream.
	safeIdentifierRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,62}$`)
)

// ValidateIdentifier reports whether name is a safe SQL identifier that Piko is willing
// to interpolate into DDL or advisory-lock keys. Returns nil for safe names and a wrapped
// ErrInvalidIdentifier otherwise.
//
// Takes name (string) which is the candidate identifier.
//
// Returns error when name fails the safe-identifier grammar.
func ValidateIdentifier(name string) error {
	if !safeIdentifierRegex.MatchString(name) {
		return fmt.Errorf("%w: %q", ErrInvalidIdentifier, name)
	}
	return nil
}

// WithHistoryTable returns a copy of d with the migration history table renamed to name.
//
// All places that reference the old name are rewritten so the rename is self-consistent:
// the CreateTableSQL DDL is updated and any advisory-lock keys keyed on the old table are
// repointed at the new name. Seed lock keys and the piko_seeds table are deliberately
// left alone because they have their own lifecycle and must not collide with a customised
// migration table name.
//
// When name is empty or matches the current HistoryTableName, d is returned unchanged.
// When name fails the safe-identifier grammar, WithHistoryTable returns d unchanged and a
// wrapped ErrInvalidIdentifier so callers can refuse to start the migration pipeline.
//
// Takes name (string) which is the new history table name.
//
// Returns DialectConfig which is a value-copy of d with the renamed table applied
// throughout.
// Returns error when name fails the safe-identifier grammar.
func (d DialectConfig) WithHistoryTable(name string) (DialectConfig, error) {
	if name == "" {
		return d, nil
	}

	previous := d.HistoryTable()
	if name == previous {
		return d, nil
	}

	if err := ValidateIdentifier(name); err != nil {
		return d, err
	}

	d.HistoryTableName = name
	d.CreateTableSQL = rewriteHistoryTableName(d.CreateTableSQL, previous, name)
	d.LockStrategy = relockStrategyForTable(d.LockStrategy, name)

	return d, nil
}

// rewriteHistoryTableName replaces previous with replacement only at SQL-identifier token
// boundaries so a previous name that happens to be a substring of another identifier or
// keyword cannot be corrupted. This stays correct when CreateTableSQL is extended to
// include columns whose names overlap the table name.
//
// Takes ddl (string) which is the original CREATE TABLE statement.
// Takes previous (string) which is the previously configured history table name to
// replace.
// Takes replacement (string) which is the new history table name.
//
// Returns string which is the rewritten DDL.
func rewriteHistoryTableName(ddl, previous, replacement string) string {
	if previous == "" || previous == replacement {
		return ddl
	}
	var builder strings.Builder
	builder.Grow(len(ddl))
	cursor := 0
	for cursor < len(ddl) {
		index := strings.Index(ddl[cursor:], previous)
		if index < 0 {
			builder.WriteString(ddl[cursor:])
			break
		}
		matchStart := cursor + index
		matchEnd := matchStart + len(previous)
		if isIdentifierBoundary(ddl, matchStart, matchEnd) {
			builder.WriteString(ddl[cursor:matchStart])
			builder.WriteString(replacement)
		} else {
			builder.WriteString(ddl[cursor:matchEnd])
		}
		cursor = matchEnd
	}
	return builder.String()
}

// isIdentifierBoundary reports whether the substring at ddl[start:end] is not flanked by
// identifier characters on either side.
//
// It is used to make sure rewriteHistoryTableName only rewrites whole-token matches.
//
// Takes ddl (string) which is the text being scanned.
// Takes start (int) which is the inclusive start index of the substring.
// Takes end (int) which is the exclusive end index of the substring.
//
// Returns bool which is true when neither flanking byte is an identifier character.
func isIdentifierBoundary(ddl string, start, end int) bool {
	if start > 0 && isIdentifierByte(ddl[start-1]) {
		return false
	}
	if end < len(ddl) && isIdentifierByte(ddl[end]) {
		return false
	}
	return true
}

// isIdentifierByte reports whether b is an ASCII letter, digit, or underscore.
//
// Takes b (byte) which is the byte to classify.
//
// Returns bool which is true when b can appear inside a SQL identifier.
func isIdentifierByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

// relockStrategyForTable returns a copy of strategy with its lock key repointed at name.
//
// Strategies that do not key on the table name (for example NoOpLock for SQLite, or
// TableBasedLock which keys on its own dedicated lock table) are returned unchanged. All
// variants are cloned via a value copy of the underlying struct before mutation so the
// original strategy stays untouched and any per-instance state (held transactions on
// TableBasedLock, or other fields on the advisory locks) is preserved by the clone before
// the LockKey field is overridden with name.
//
// Takes strategy (LockStrategy) which is the lock strategy to re-key. May be nil.
// Takes name (string) which is the new lock key (typically the migration history table
// name).
//
// Returns LockStrategy which is the re-keyed strategy.
func relockStrategyForTable(strategy LockStrategy, name string) LockStrategy {
	switch existing := strategy.(type) {
	case *PostgresAdvisoryLock:
		clone := *existing
		clone.LockKey = name
		return &clone
	case *MySQLAdvisoryLock:
		clone := *existing
		clone.LockKey = name
		return &clone
	case *TableBasedLock:
		return new(*existing)
	case nil:
		return nil
	default:
		return strategy
	}
}

// postgresOnConflictSeedInsert builds an idempotent INSERT statement using PostgreSQL's
// "ON CONFLICT (version) DO NOTHING" clause. SQLite (in modern versions) accepts the same
// syntax.
//
// Takes versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder
// (string) which are the dialect-specific parameter markers for the four bound columns.
//
// Returns string which is the complete INSERT statement.
func postgresOnConflictSeedInsert(
	versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder string,
) string {
	return fmt.Sprintf(
		"INSERT INTO piko_seeds (version, name, checksum, duration_ms) VALUES (%s, %s, %s, %s) ON CONFLICT (version) DO NOTHING",
		versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder,
	)
}

// sqliteOnConflictSeedInsert builds an idempotent INSERT statement using SQLite's "ON
// CONFLICT(version) DO NOTHING" clause (no space between ON CONFLICT and the column list,
// mirroring SQLite's grammar).
//
// Takes versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder
// (string) which are the dialect-specific parameter markers for the four bound columns.
//
// Returns string which is the complete INSERT statement.
func sqliteOnConflictSeedInsert(
	versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder string,
) string {
	return fmt.Sprintf(
		"INSERT INTO piko_seeds (version, name, checksum, duration_ms) VALUES (%s, %s, %s, %s) ON CONFLICT(version) DO NOTHING",
		versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder,
	)
}

// mysqlIgnoreSeedInsert builds an idempotent INSERT using MySQL's INSERT IGNORE.
//
// MySQL 5.7+ also supports "INSERT ... ON DUPLICATE KEY UPDATE", but INSERT IGNORE is the
// simplest no-op equivalent.
//
// Takes versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder
// (string) which are the dialect-specific parameter markers for the four bound columns.
//
// Returns string which is the complete INSERT statement.
func mysqlIgnoreSeedInsert(
	versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder string,
) string {
	return fmt.Sprintf(
		"INSERT IGNORE INTO piko_seeds (version, name, checksum, duration_ms) VALUES (%s, %s, %s, %s)",
		versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder,
	)
}

// clickHouseNotExistsSeedInsert builds an idempotent INSERT for ClickHouse engines.
//
// ClickHouse MergeTree tables do not enforce primary-key uniqueness and the engine has no
// equivalent of ON CONFLICT / INSERT IGNORE. The statement guards the INSERT by wrapping
// the values in a derived row source, then filtering the candidate row out when its
// version already exists in piko_seeds. A re-application of an already recorded seed
// inserts zero rows so the second migrator does not see a duplicate version row.
//
// Each placeholder appears exactly once in the rendered SQL so the call site can pass the
// same fixed positional argument tuple (version, name, checksum, duration_ms) that
// PostgreSQL, SQLite, and MySQL use; ClickHouse anonymous markers bind in order.
//
// Deployments that need exactly-once semantics at scan time should configure the
// piko_seeds table as ReplacingMergeTree(version) and read with FINAL; the NOT IN guard
// handles the write side but a race between two migrators between the SELECT and the
// INSERT can still emit two rows. ReplacingMergeTree + FINAL handles the read side after
// such a race. The ClickHouseDialect doc-comment captures this contract.
//
// Takes versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder
// (string) which are the dialect-specific parameter markers for the four bound columns.
//
// Returns string which is the complete INSERT statement.
func clickHouseNotExistsSeedInsert(
	versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder string,
) string {
	return fmt.Sprintf(
		"INSERT INTO piko_seeds (version, name, checksum, duration_ms) "+
			"SELECT candidate.version, candidate.name, candidate.checksum, candidate.duration_ms "+
			"FROM (SELECT %s AS version, %s AS name, %s AS checksum, %s AS duration_ms) AS candidate "+
			"WHERE candidate.version NOT IN (SELECT version FROM piko_seeds)",
		versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder,
	)
}

// clickHouseDeleteHistory builds the ClickHouse rollback statement for removing a single
// migration history row.
//
// ClickHouse MergeTree does not support a plain DELETE, so the removal is expressed as a
// lightweight ALTER TABLE ... DELETE mutation. The mutation is asynchronous, which is
// acceptable because rollbacks are rare and the ReplacingMergeTree history table reads
// with FINAL.
//
// Takes historyTable (string) which is the validated history table identifier.
// Takes versionPlaceholder (string) which is the parameter marker for the version column.
//
// Returns string which is the complete ALTER TABLE ... DELETE statement.
func clickHouseDeleteHistory(historyTable, versionPlaceholder string) string {
	return fmt.Sprintf( //nolint:gosec // historyTable is validated upstream via ValidateIdentifier
		"ALTER TABLE %s DELETE WHERE version = %s",
		historyTable, versionPlaceholder,
	)
}

// PostgresDialect returns a DialectConfig for PostgreSQL databases.
//
// Returns DialectConfig which is configured with PostgreSQL-specific SQL, advisory
// locking, and $N placeholder syntax.
func PostgresDialect() DialectConfig {
	return DialectConfig{
		HistoryTableName: DefaultHistoryTableName,
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
		LockStrategy:      &PostgresAdvisoryLock{LockKey: DefaultHistoryTableName},
		SeedLockStrategy:  &PostgresAdvisorySeedLock{},
		InsertSeedSQLFunc: postgresOnConflictSeedInsert,
		PlaceholderFunc: func(index int) string {
			return fmt.Sprintf("$%d", index)
		},
	}
}

// PostgresPgBouncerDialect returns a DialectConfig for PostgreSQL databases behind
// PgBouncer in transaction mode.
//
// Advisory locks are not available in this configuration, so a table-based lock via
// SELECT ... FOR UPDATE is used instead.
//
// Returns DialectConfig which is configured with PostgreSQL-specific SQL, table-based
// locking, and $N placeholder syntax.
func PostgresPgBouncerDialect() DialectConfig {
	return DialectConfig{
		HistoryTableName: DefaultHistoryTableName,
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
		SeedLockStrategy: &TableBasedSeedLock{
			CreateLockTableSQL: `CREATE TABLE IF NOT EXISTS piko_seed_lock (
    lock_id INTEGER NOT NULL PRIMARY KEY DEFAULT 1,
    CONSTRAINT piko_seed_lock_single_row CHECK (lock_id = 1)
)`,
		},
		InsertSeedSQLFunc: postgresOnConflictSeedInsert,
		PlaceholderFunc: func(index int) string {
			return fmt.Sprintf("$%d", index)
		},
	}
}

// MySQLDialect returns a DialectConfig for MySQL databases.
//
// Returns DialectConfig which is configured with MySQL-specific SQL, advisory locking,
// and ? placeholder syntax. SplitStatements is enabled since MySQL does not support
// multi-statement execution by default.
func MySQLDialect() DialectConfig {
	return DialectConfig{
		HistoryTableName: DefaultHistoryTableName,
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
		LockStrategy:      &MySQLAdvisoryLock{LockKey: DefaultHistoryTableName},
		SeedLockStrategy:  &MySQLAdvisorySeedLock{},
		InsertSeedSQLFunc: mysqlIgnoreSeedInsert,
		PlaceholderFunc:   func(_ int) string { return "?" },
		SplitStatements:   true,
		BackslashEscapes:  true,
	}
}

// MySQLDialectWithDSN returns a DialectConfig for MySQL databases, detecting whether the
// DSN already includes multiStatements=true. When the driver is configured to handle
// multi-statement execution natively, the framework disables its own statement splitting
// to avoid interfering with stored procedures or complex migration SQL.
//
// Takes dsn (string) which is the MySQL data source name.
//
// Returns DialectConfig which is configured for MySQL with automatic SplitStatements
// detection.
func MySQLDialectWithDSN(dsn string) DialectConfig {
	dialect := MySQLDialect()
	if strings.Contains(dsn, "multiStatements=true") {
		dialect.SplitStatements = false
	}
	return dialect
}

// ClickHouseDialect returns a DialectConfig for ClickHouse databases.
//
// ClickHouse has no advisory locks, no DDL transactions, and no in-place row UPDATE on
// MergeTree, so the dialect diverges from the SQL-standard engines in four ways.
//
// Locking is a no-op (NoOpLock) because ClickHouse offers no cheap session-scoped
// advisory lock. The earlier table-lock attempt used ON CONFLICT and SELECT ... FOR
// UPDATE, neither of which ClickHouse supports, so the very first migration aborted at
// lock acquisition. Concurrent migrators are therefore not serialised by Piko;
// deployments that run migrations from more than one place must coordinate through a CI
// lock or a deployment gate. The ReplacingMergeTree history table read with FINAL keeps
// the recorded state consistent after a race.
//
// Transactions are disabled (DisableTransactions: true), so every migration runs on the
// non-transactional path.
//
// History is append-only (AppendOnlyHistory: true). The history table is a
// ReplacingMergeTree(applied_at) ordered by version, so re-recording a version collapses
// to the latest row on merge or FINAL. Up-migrations write one completed row on success;
// rollbacks remove the row via an ALTER TABLE ... DELETE mutation (DeleteHistorySQLFunc).
//
// Applied-versions reads use FINAL (SelectHistoryFinal: true) so duplicate version rows
// that have not yet merged are deduplicated at read time.
//
// ClickHouse has no native boolean type; the `dirty` column is stored as UInt8 with 0 and
// 1 semantics that the framework translates transparently. The seed history table is a
// ReplacingMergeTree(version) and combines with the NOT IN guard inside InsertSeedSQLFunc
// so a re-application of a recorded seed is a no-op even across a migrator race.
//
// Returns DialectConfig which is configured with ClickHouse-specific SQL, no-op locking,
// ? placeholder syntax, per-statement execution, append-only history, and an idempotent
// seed insert helper.
func ClickHouseDialect() DialectConfig {
	return DialectConfig{
		HistoryTableName: DefaultHistoryTableName,
		CreateTableSQL: `CREATE TABLE IF NOT EXISTS piko_migrations (
    version        Int64       NOT NULL,
    name           String      NOT NULL,
    checksum       String      NOT NULL,
    applied_at     DateTime64(6) NOT NULL DEFAULT now64(),
    duration_ms    Int64       NOT NULL,
    down_checksum  Nullable(String),
    last_statement Nullable(Int32),
    dirty          UInt8       NOT NULL DEFAULT 0
) ENGINE = ReplacingMergeTree(applied_at) ORDER BY version`,
		CreateSeedTableSQL: `CREATE TABLE IF NOT EXISTS piko_seeds (
    version     Int64         NOT NULL,
    name        String        NOT NULL,
    checksum    String        NOT NULL,
    applied_at  DateTime64(6) NOT NULL DEFAULT now64(),
    duration_ms Int64         NOT NULL
) ENGINE = ReplacingMergeTree(applied_at) ORDER BY version`,
		LockStrategy:         &NoOpLock{},
		SeedLockStrategy:     &NoOpLock{},
		InsertSeedSQLFunc:    clickHouseNotExistsSeedInsert,
		DeleteHistorySQLFunc: clickHouseDeleteHistory,
		PlaceholderFunc:      func(_ int) string { return "?" },
		SplitStatements:      true,
		DisableTransactions:  true,
		AppendOnlyHistory:    true,
		SelectHistoryFinal:   true,
		BackslashEscapes:     true,
	}
}

// SQLiteDialect returns a DialectConfig for SQLite databases.
//
// Returns DialectConfig which is configured with SQLite-specific SQL, no-op locking, and
// ? placeholder syntax.
func SQLiteDialect() DialectConfig {
	return DialectConfig{
		HistoryTableName: DefaultHistoryTableName,
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
		LockStrategy:      &NoOpLock{},
		SeedLockStrategy:  &NoOpLock{},
		InsertSeedSQLFunc: sqliteOnConflictSeedInsert,
		PlaceholderFunc:   func(_ int) string { return "?" },
	}
}
