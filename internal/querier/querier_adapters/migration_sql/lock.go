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
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	var acquired bool
	scanError := connection.QueryRowContext(
		ctx, "SELECT pg_try_advisory_lock(hashtext('piko_migrations'))",
	).Scan(&acquired)
	if scanError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("scanning PostgreSQL try-lock result: %w", scanError)
	}

	if !acquired {
		_ = connection.Close()
		return nil, querier_domain.ErrLockNotAcquired
	}

	return connection, nil
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
	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	if _, createError := connection.ExecContext(ctx, lock.CreateLockTableSQL); createError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("creating lock table: %w", createError)
	}

	_, insertError := connection.ExecContext(ctx,
		"INSERT INTO piko_migration_lock (lock_id) VALUES (1) ON CONFLICT DO NOTHING",
	)
	if insertError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("inserting lock row: %w", insertError)
	}

	transaction, beginError := connection.BeginTx(ctx, nil)
	if beginError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("beginning lock transaction: %w", beginError)
	}

	_, lockError := transaction.ExecContext(ctx,
		"SELECT lock_id FROM piko_migration_lock "+lockMode, //nolint:gosec // internal constant
	)
	if lockError != nil {
		_ = transaction.Rollback()
		_ = connection.Close()
		if isLockNotAvailableError(lockError) {
			return nil, querier_domain.ErrLockNotAcquired
		}
		return nil, fmt.Errorf("acquiring table lock: %w", lockError)
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
