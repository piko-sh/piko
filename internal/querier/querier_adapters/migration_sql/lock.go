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
	"context"
	"database/sql"
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_domain"
)

// errorFormatPinningConnection holds the format string for connection pinning errors.
const errorFormatPinningConnection = "pinning database connection: %w"

// LockStrategy abstracts database-specific advisory locking for migration
// concurrency control.
//
// Implementations that require connection pinning (e.g. PostgreSQL advisory
// locks, which are session-scoped) return a dedicated *sql.Conn from Acquire.
// The caller must pass this connection back to Release.
type LockStrategy interface {
	// Acquire acquires an advisory lock on the given database.
	//
	// For strategies that require connection pinning, this pins a dedicated
	// connection from the pool and returns it.
	//
	// Takes database (*sql.DB) which is the connection pool to acquire the lock from.
	//
	// Returns *sql.Conn which is the pinned connection, or nil when no
	// connection pinning is needed (e.g. NoOpLock for SQLite).
	// Returns error when the lock cannot be acquired.
	Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error)

	// TryAcquire attempts to acquire an advisory lock without blocking.
	//
	// Takes database (*sql.DB) which is the connection pool to acquire the lock from.
	//
	// Returns *sql.Conn which is the pinned connection, or nil when no
	// connection pinning is needed.
	// Returns error when the lock cannot be acquired, including
	// querier_domain.ErrLockNotAcquired if the lock is already held by
	// another session.
	TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error)

	// Release releases the advisory lock.
	//
	// Takes connection (*sql.Conn) which is the pinned connection to unlock
	// and close. If nil, this is a no-op.
	//
	// Returns error when the lock cannot be released.
	Release(ctx context.Context, connection *sql.Conn) error
}

// PostgresAdvisoryLock implements LockStrategy using PostgreSQL's
// pg_advisory_lock function.
//
// The lock is session-scoped, so a dedicated connection is pinned from the pool
// to ensure all subsequent operations run on the same connection that holds the
// lock.
type PostgresAdvisoryLock struct{}

// Acquire pins a connection from the pool and acquires a PostgreSQL advisory
// lock keyed on the migration table name.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (*PostgresAdvisoryLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	_, lockError := connection.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext('piko_migrations'))")
	if lockError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("acquiring PostgreSQL advisory lock: %w", lockError)
	}

	return connection, nil
}

// Release releases the PostgreSQL advisory lock and returns the pinned
// connection to the pool.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (*PostgresAdvisoryLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	_, unlockError := connection.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext('piko_migrations'))")

	closeError := connection.Close()

	if unlockError != nil {
		return fmt.Errorf("releasing PostgreSQL advisory lock: %w", unlockError)
	}
	return closeError
}

// TryAcquire attempts to acquire the advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (*PostgresAdvisoryLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return tryAcquirePostgresAdvisoryLock(ctx, database, "piko_migrations", "PostgreSQL")
}

// TableBasedLock implements LockStrategy using a dedicated lock table with
// SELECT ... FOR UPDATE.
//
// This is compatible with PgBouncer in transaction mode where advisory locks
// are not available.
type TableBasedLock struct {
	// heldTransaction holds the open transaction that maintains the FOR UPDATE lock.
	heldTransaction *sql.Tx

	// CreateLockTableSQL is the DDL statement for creating the lock table.
	CreateLockTableSQL string
}

// Acquire pins a connection, creates the lock table if needed, inserts a lock
// row, and acquires a FOR UPDATE lock within a held transaction.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the table lock.
// Returns error when any step of the lock acquisition fails.
func (lock *TableBasedLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return lock.acquireWithMode(ctx, database, "FOR UPDATE")
}

// TryAcquire is like Acquire but uses FOR UPDATE NOWAIT.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the table lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (lock *TableBasedLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return lock.acquireWithMode(ctx, database, "FOR UPDATE NOWAIT")
}

// Release commits the held transaction (releasing the FOR UPDATE lock) and
// closes the pinned connection.
//
// Takes connection (*sql.Conn) which is the pinned connection to close.
//
// Returns error when the transaction commit or connection close fails.
func (lock *TableBasedLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	var commitError error
	if lock.heldTransaction != nil {
		if ctx.Err() != nil {
			_ = lock.heldTransaction.Rollback()
		} else {
			commitError = lock.heldTransaction.Commit()
		}
		lock.heldTransaction = nil
	}

	closeError := connection.Close()

	if commitError != nil {
		return fmt.Errorf("committing table lock transaction: %w", commitError)
	}
	return closeError
}

// acquireWithMode pins a connection, creates the lock
// table, inserts a lock row, and acquires a row lock using
// the specified lock mode.
//
// Takes database (*sql.DB) which is the connection pool to
// pin a connection from.
// Takes lockMode (string) which specifies the row lock
// clause (e.g. "FOR UPDATE" or "FOR UPDATE NOWAIT").
//
// Returns *sql.Conn which is the pinned connection holding
// the lock.
// Returns error when any step of the acquisition fails.
func (lock *TableBasedLock) acquireWithMode(
	ctx context.Context,
	database *sql.DB,
	lockMode string,
) (*sql.Conn, error) {
	connection, transaction, err := acquireTableBasedLock(ctx, database, lockMode, tableBasedLockOptions{
		createTableSQL:    lock.CreateLockTableSQL,
		tableName:         "piko_migration_lock",
		errorContextLower: "",
	})
	if err != nil {
		return nil, err
	}
	lock.heldTransaction = transaction
	return connection, nil
}

// isLockNotAvailableError checks whether the error indicates the lock could
// not be acquired (PostgreSQL error code 55P03: lock_not_available).
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when the error matches a known lock-not-available pattern.
func isLockNotAvailableError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "55P03") ||
		strings.Contains(message, "could not obtain lock") ||
		strings.Contains(message, "1205") ||
		strings.Contains(message, "Lock wait timeout exceeded")
}

// MySQLAdvisoryLock implements LockStrategy using MySQL's GET_LOCK and
// RELEASE_LOCK functions. The lock is session-scoped, so a dedicated
// connection is pinned from the pool.
type MySQLAdvisoryLock struct{}

// Acquire pins a connection from the pool and acquires a MySQL advisory lock
// using GET_LOCK with an indefinite timeout.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (*MySQLAdvisoryLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return mysqlGetLock(ctx, database, "SELECT GET_LOCK('piko_migrations', -1)")
}

// TryAcquire attempts to acquire the MySQL advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (*MySQLAdvisoryLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return mysqlGetLock(ctx, database, "SELECT GET_LOCK('piko_migrations', 0)")
}

// mysqlGetLock pins a connection from the pool and executes
// the given GET_LOCK query.
//
// Takes database (*sql.DB) which is the connection pool to
// pin a connection from.
// Takes query (string) which is the GET_LOCK SQL statement
// to execute.
//
// Returns *sql.Conn which is the pinned connection holding
// the lock.
// Returns error when the lock cannot be acquired.
func mysqlGetLock(ctx context.Context, database *sql.DB, query string) (*sql.Conn, error) {
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	var acquired sql.NullInt64
	scanError := connection.QueryRowContext(ctx, query).Scan(&acquired)
	if scanError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("scanning MySQL lock result: %w", scanError)
	}

	if !acquired.Valid || acquired.Int64 != 1 {
		_ = connection.Close()
		return nil, querier_domain.ErrLockNotAcquired
	}

	return connection, nil
}

// Release releases the MySQL advisory lock and returns the pinned connection
// to the pool.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (*MySQLAdvisoryLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	_, unlockError := connection.ExecContext(ctx, "SELECT RELEASE_LOCK('piko_migrations')")

	closeError := connection.Close()

	if unlockError != nil {
		return fmt.Errorf("releasing MySQL advisory lock: %w", unlockError)
	}
	return closeError
}

// PostgresAdvisorySeedLock implements LockStrategy using PostgreSQL's
// pg_advisory_lock function, keyed on the literal "piko_seeds" so it never
// collides with the migration lock.
type PostgresAdvisorySeedLock struct{}

// Acquire pins a connection from the pool and acquires a PostgreSQL advisory
// lock keyed on hashtext('piko_seeds').
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (*PostgresAdvisorySeedLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	_, lockError := connection.ExecContext(ctx, "SELECT pg_advisory_lock(hashtext('piko_seeds'))")
	if lockError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("acquiring PostgreSQL seed advisory lock: %w", lockError)
	}

	return connection, nil
}

// TryAcquire attempts to acquire the seed advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (*PostgresAdvisorySeedLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return tryAcquirePostgresAdvisoryLock(ctx, database, "piko_seeds", "PostgreSQL seed")
}

// Release releases the PostgreSQL seed advisory lock and returns the pinned
// connection to the pool.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (*PostgresAdvisorySeedLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	_, unlockError := connection.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext('piko_seeds'))")

	closeError := connection.Close()

	if unlockError != nil {
		return fmt.Errorf("releasing PostgreSQL seed advisory lock: %w", unlockError)
	}
	return closeError
}

// MySQLAdvisorySeedLock implements LockStrategy using MySQL's GET_LOCK and
// RELEASE_LOCK functions, keyed on the literal "piko_seeds" so it never
// collides with the migration lock.
type MySQLAdvisorySeedLock struct{}

// Acquire pins a connection from the pool and acquires a MySQL advisory lock
// using GET_LOCK with an indefinite timeout.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (*MySQLAdvisorySeedLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return mysqlGetLock(ctx, database, "SELECT GET_LOCK('piko_seeds', -1)")
}

// TryAcquire attempts to acquire the MySQL seed advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (*MySQLAdvisorySeedLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return mysqlGetLock(ctx, database, "SELECT GET_LOCK('piko_seeds', 0)")
}

// Release releases the MySQL seed advisory lock and returns the pinned
// connection to the pool.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (*MySQLAdvisorySeedLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	_, unlockError := connection.ExecContext(ctx, "SELECT RELEASE_LOCK('piko_seeds')")

	closeError := connection.Close()

	if unlockError != nil {
		return fmt.Errorf("releasing MySQL seed advisory lock: %w", unlockError)
	}
	return closeError
}

// TableBasedSeedLock implements LockStrategy via a dedicated lock table.
//
// Uses piko_seed_lock with SELECT ... FOR UPDATE. Used in PgBouncer
// transaction mode where session-scoped advisory locks are not available.
type TableBasedSeedLock struct {
	// heldTransaction holds the open transaction that maintains the FOR UPDATE lock.
	heldTransaction *sql.Tx

	// CreateLockTableSQL is the DDL statement for creating the seed lock table.
	CreateLockTableSQL string
}

// Acquire pins a connection, creates the seed lock table if needed, inserts a
// lock row, and acquires a FOR UPDATE lock within a held transaction.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the table lock.
// Returns error when any step of the lock acquisition fails.
func (lock *TableBasedSeedLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return lock.acquireWithMode(ctx, database, "FOR UPDATE")
}

// TryAcquire is like Acquire but uses FOR UPDATE NOWAIT.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the table lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (lock *TableBasedSeedLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	return lock.acquireWithMode(ctx, database, "FOR UPDATE NOWAIT")
}

// Release commits the held transaction (releasing the FOR UPDATE lock) and
// closes the pinned connection.
//
// Takes connection (*sql.Conn) which is the pinned connection to close.
//
// Returns error when the transaction commit or connection close fails.
func (lock *TableBasedSeedLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	var commitError error
	if lock.heldTransaction != nil {
		if ctx.Err() != nil {
			_ = lock.heldTransaction.Rollback()
		} else {
			commitError = lock.heldTransaction.Commit()
		}
		lock.heldTransaction = nil
	}

	closeError := connection.Close()

	if commitError != nil {
		return fmt.Errorf("committing seed table lock transaction: %w", commitError)
	}
	return closeError
}

// acquireWithMode pins a connection, creates the lock table, inserts a lock
// row, and acquires a row lock using the specified lock mode.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection
// from.
// Takes lockMode (string) which specifies the row lock clause (e.g.
// "FOR UPDATE" or "FOR UPDATE NOWAIT").
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns error when any step of the acquisition fails.
func (lock *TableBasedSeedLock) acquireWithMode(
	ctx context.Context,
	database *sql.DB,
	lockMode string,
) (*sql.Conn, error) {
	connection, transaction, err := acquireTableBasedLock(ctx, database, lockMode, tableBasedLockOptions{
		createTableSQL:    lock.CreateLockTableSQL,
		tableName:         "piko_seed_lock",
		errorContextLower: "seed ",
	})
	if err != nil {
		return nil, err
	}
	lock.heldTransaction = transaction
	return connection, nil
}

// NoOpLock implements LockStrategy as a no-op. This is used for SQLite where
// file-level locking provides sufficient concurrency control.
type NoOpLock struct{}

// Acquire is a no-op for SQLite.
//
// Returns nil *sql.Conn and nil error since no locking is needed.
func (*NoOpLock) Acquire(_ context.Context, _ *sql.DB) (*sql.Conn, error) { return nil, nil }

// TryAcquire is a no-op for SQLite.
//
// Returns nil *sql.Conn and nil error since no locking is needed.
func (*NoOpLock) TryAcquire(_ context.Context, _ *sql.DB) (*sql.Conn, error) { return nil, nil }

// Release is a no-op for SQLite.
//
// Returns nil error since no lock was held.
func (*NoOpLock) Release(_ context.Context, _ *sql.Conn) error { return nil }

// tableBasedLockOptions captures the dialect-independent inputs to the
// table-based locking pipeline. Only the table name and an error-message
// prefix differ between the migration and seed lock paths.
type tableBasedLockOptions struct {
	// createTableSQL is the dialect-specific CREATE TABLE IF NOT EXISTS
	// statement that idempotently creates the underlying lock table.
	createTableSQL string

	// tableName is the bare table name used in INSERT and SELECT statements.
	//
	// e.g. "piko_migration_lock" or "piko_seed_lock".
	tableName string

	// errorContextLower is a short lower-case prefix interpolated into
	// diagnostic error messages so callers can distinguish migration and
	// seed lock failures (e.g. "seed " or empty for migrations).
	errorContextLower string
}

// acquireTableBasedLock pins a connection from the pool, ensures the lock
// table exists, inserts the singleton row if missing, opens a transaction,
// and acquires the configured row lock. On any failure all resources are
// released before the error is returned.
//
// The shared helper exists so the migration and seed table-based lock paths
// stay byte-equivalent in semantics and only their table name + error
// messages diverge.
//
// Takes ctx (context.Context) for cancellation and timeouts.
// Takes database (*sql.DB) which is the connection pool to pin from.
// Takes lockMode (string) which is the row lock clause appended to the
// SELECT (e.g. "FOR UPDATE", "FOR UPDATE NOWAIT").
// Takes options (tableBasedLockOptions) which describe the table-specific
// SQL and error-message prefixes.
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns *sql.Tx which is the active transaction maintaining the lock.
// Returns error which wraps querier_domain.ErrLockNotAcquired when the lock
// is not available, or any underlying connection / SQL error.
func acquireTableBasedLock(
	ctx context.Context,
	database *sql.DB,
	lockMode string,
	options tableBasedLockOptions,
) (*sql.Conn, *sql.Tx, error) {
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	if _, createError := connection.ExecContext(ctx, options.createTableSQL); createError != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf("creating %slock table: %w", options.errorContextLower, createError)
	}

	insertSQL := fmt.Sprintf( //nolint:gosec // tableName is a hardcoded internal constant
		"INSERT INTO %s (lock_id) VALUES (1) ON CONFLICT DO NOTHING",
		options.tableName,
	)
	if _, insertError := connection.ExecContext(ctx, insertSQL); insertError != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf("inserting %slock row: %w", options.errorContextLower, insertError)
	}

	transaction, beginError := connection.BeginTx(ctx, nil)
	if beginError != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf("beginning %slock transaction: %w", options.errorContextLower, beginError)
	}

	selectSQL := fmt.Sprintf( //nolint:gosec // tableName is a hardcoded internal constant
		"SELECT lock_id FROM %s %s", options.tableName, lockMode,
	)
	if _, lockError := transaction.ExecContext(ctx, selectSQL); lockError != nil {
		_ = transaction.Rollback()
		_ = connection.Close()
		if isLockNotAvailableError(lockError) {
			return nil, nil, querier_domain.ErrLockNotAcquired
		}
		return nil, nil, fmt.Errorf("acquiring %stable lock: %w", options.errorContextLower, lockError)
	}

	return connection, transaction, nil
}

// tryAcquirePostgresAdvisoryLock pins a connection from the pool and runs
// pg_try_advisory_lock for the supplied lock key (hashed via PostgreSQL's
// hashtext function). On any failure the pinned connection is returned to
// the pool before the error bubbles up.
//
// Takes database (*sql.DB) which is the connection pool to pin from.
// Takes lockKey (string) which is the textual identifier used as the lock key
// (passed through hashtext at SQL time).
// Takes errorContext (string) which prefixes diagnostic error messages
// (e.g. "PostgreSQL", "PostgreSQL seed").
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns error which wraps querier_domain.ErrLockNotAcquired when the lock
// is already held, or any underlying connection / query error.
func tryAcquirePostgresAdvisoryLock(
	ctx context.Context,
	database *sql.DB,
	lockKey, errorContext string,
) (*sql.Conn, error) {
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	var acquired bool
	//nolint:gosec // lockKey is a hardcoded internal constant ("piko_migrations" or "piko_seeds")
	tryQuery := fmt.Sprintf("SELECT pg_try_advisory_lock(hashtext('%s'))", lockKey)
	scanError := connection.QueryRowContext(ctx, tryQuery).Scan(&acquired)
	if scanError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("scanning %s try-lock result: %w", errorContext, scanError)
	}

	if !acquired {
		_ = connection.Close()
		return nil, querier_domain.ErrLockNotAcquired
	}

	return connection, nil
}
