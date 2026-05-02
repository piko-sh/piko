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
	"errors"
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/querier/querier_domain"
)

const (
	// lockReleaseTimeout bounds how long Release waits for the unlock SQL.
	//
	// Release is invoked from deferred cleanup after possibly long-running migrations. Even
	// when the caller's context is cancelled the advisory lock is still freed so subsequent
	// migration runs do not hang waiting for the session to be reaped.
	lockReleaseTimeout = 5 * time.Second

	// errorFormatPinningConnection holds the format string for connection pinning errors.
	errorFormatPinningConnection = "pinning database connection: %w"

	// postgresErrorCodeLockNotAvailable is the SQLSTATE string returned by PostgreSQL when a
	// SELECT ... FOR UPDATE NOWAIT or pg_try_advisory_lock fails because the resource is
	// already locked by another session.
	postgresErrorCodeLockNotAvailable = "55P03"

	// mysqlErrorNumberLockWaitTimeout is the MySQL error number reported as a string by
	// driver error messages when the lock wait exceeds innodb_lock_wait_timeout.
	mysqlErrorNumberLockWaitTimeout = "1205"

	// mysqlErrorNumberLockWaitTimeoutUint is the numeric form of MySQL error 1205 used by
	// the typed-driver-error check in isLockNotAvailableError.
	mysqlErrorNumberLockWaitTimeoutUint uint16 = 1205

	// seedLockKey is the hardcoded identifier used for both the PostgreSQL and MySQL
	// advisory seed locks. Centralised so the validation runs against the same constant
	// every Acquire / TryAcquire / Release path uses.
	seedLockKey = "piko_seeds"
)

var (
	// errLockReleaseTimeout is the cause attached to the detached release context's deadline
	// (see detachContextForRelease) so a caller inspecting context.Cause can tell a slow
	// unlock apart from an unrelated cancellation of the parent context.
	errLockReleaseTimeout = errors.New("migration lock release timed out")
)

// detachContextForRelease returns a fresh context that does not inherit cancellation from
// parent, with a short timeout so the unlock SQL has a fair chance to run even when the
// parent context is already cancelled. The returned cancel function MUST be invoked by
// the caller.
//
// Returns context.Context which is the detached release context with a short timeout.
// Returns context.CancelFunc which the caller must invoke to release the timer.
func detachContextForRelease(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeoutCause(context.WithoutCancel(parent), lockReleaseTimeout, errLockReleaseTimeout)
}

// validateLockKey checks that lockKey is safe to interpolate into advisory-lock SQL.
//
// Defence in depth: every Acquire / TryAcquire / Release path runs the validator even for
// hardcoded constants so accidental future edits to those constants cannot silently
// enable SQL injection.
//
// Takes lockKey (string) which is the textual lock identifier to validate.
// Takes site (string) which prefixes the error message so callers can identify the
// validation site (e.g. "PostgresAdvisorySeedLock").
//
// Returns error which wraps ErrInvalidIdentifier when lockKey fails the safe-identifier
// grammar.
func validateLockKey(lockKey, site string) error {
	if err := ValidateIdentifier(lockKey); err != nil {
		return fmt.Errorf("%s lock key: %w", site, err)
	}
	return nil
}

// joinLockReleaseErrors combines an unlock error and a close error into a single error so
// the caller can observe both when both fail.
//
// The unlock error is wrapped with a descriptive prefix so the caller can tell which step
// failed.
//
// Takes prefix (string) which describes the unlock step for error wrapping.
// Takes unlockError (error) which is the failure from running the unlock SQL.
// Takes closeError (error) which is the failure from closing the connection.
//
// Returns error which is nil when neither failed, the surviving error when only one
// failed, and a joined error otherwise.
func joinLockReleaseErrors(prefix string, unlockError, closeError error) error {
	var wrappedUnlock error
	if unlockError != nil {
		wrappedUnlock = fmt.Errorf("%s: %w", prefix, unlockError)
	}
	return errors.Join(wrappedUnlock, closeError)
}

// releaseLockSQL executes the dialect-specific unlock SQL against a detached release
// context, closes the pinned connection, and joins both errors under prefix.
//
// All four advisory-lock Release paths (PostgreSQL migration, MySQL migration, PostgreSQL
// seed, MySQL seed) share the same release pipeline: detach from the caller context so a
// cancelled parent does not strand a session-scoped lock, run the dialect-specific unlock
// SQL, close the connection back to the pool, and surface both failures together.
// Centralising the pipeline keeps the four call sites in lockstep so a future refactor
// (e.g. extending the release timeout) only edits one place.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
// Takes unlockSQL (string) which is the dialect-specific release statement.
// Takes prefix (string) which describes the release site for error wrapping.
//
// Returns error which wraps the unlock and close failures, or nil when both succeed.
func releaseLockSQL(ctx context.Context, connection *sql.Conn, unlockSQL, prefix string) error {
	releaseCtx, cancel := detachContextForRelease(ctx)
	defer cancel()

	_, unlockError := connection.ExecContext(releaseCtx, unlockSQL)
	closeError := connection.Close()
	return joinLockReleaseErrors(prefix, unlockError, closeError)
}

// LockStrategy abstracts database-specific advisory locking for migration concurrency
// control.
//
// Implementations that require connection pinning (e.g. PostgreSQL advisory locks, which
// are session-scoped) return a dedicated *sql.Conn from Acquire. The caller must pass
// this connection back to Release.
//
// A LockStrategy value is single-use per acquisition: a successful Acquire / TryAcquire
// must be paired with a Release before the next Acquire. The table-based strategies carry
// the open transaction in a struct field and refuse a second concurrent Acquire with
// errLockAlreadyHeld rather than silently overwriting (and leaking) the held transaction;
// they are therefore not safe to share across goroutines without external serialisation.
type LockStrategy interface {
	// Acquire acquires an advisory lock on the given database.
	//
	// For strategies that require connection pinning, this pins a dedicated connection from
	// the pool and returns it.
	//
	// Takes database (*sql.DB) which is the connection pool to acquire the lock from.
	//
	// Returns *sql.Conn which is the pinned connection, or nil when no connection pinning is
	// needed (e.g. NoOpLock for SQLite).
	// Returns error when the lock cannot be acquired.
	Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error)

	// TryAcquire attempts to acquire an advisory lock without blocking.
	//
	// Takes database (*sql.DB) which is the connection pool to acquire the lock from.
	//
	// Returns *sql.Conn which is the pinned connection, or nil when no connection pinning is
	// needed.
	// Returns error when the lock cannot be acquired, including
	// querier_domain.ErrLockNotAcquired if the lock is already held by another session.
	TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error)

	// Release releases the advisory lock.
	//
	// Takes connection (*sql.Conn) which is the pinned connection to unlock and close. If
	// nil, this is a no-op.
	//
	// Returns error when the lock cannot be released.
	Release(ctx context.Context, connection *sql.Conn) error
}

// PostgresAdvisoryLock implements LockStrategy using PostgreSQL's pg_advisory_lock
// function.
//
// The lock is session-scoped, so a dedicated connection is pinned from the pool to ensure
// all subsequent operations run on the same connection that holds the lock.
//
// LockKey, when non-empty, MUST be a safe SQL identifier per ValidateIdentifier. The key
// is interpolated into pg_advisory_lock SQL, so unsafe input would enable injection.
// Acquire / TryAcquire / Release validate the key on every call and return a wrapped
// ErrInvalidIdentifier when validation fails, so unsafe direct struct construction
// surfaces as an error rather than enabling injection.
type PostgresAdvisoryLock struct {
	// LockKey is the text passed to hashtext() when acquiring and releasing the advisory
	// lock.
	//
	// Distinct keys allow multiple MigrationService instances against the same database to
	// run concurrently without colliding. When empty, the key defaults to
	// DefaultHistoryTableName.
	LockKey string
}

// Acquire pins a connection from the pool and acquires a PostgreSQL advisory lock keyed
// on the migration table name.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (lock *PostgresAdvisoryLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	key, keyError := lock.key()
	if keyError != nil {
		return nil, keyError
	}

	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	//nolint:gosec // key passed ValidateIdentifier above
	lockSQL := fmt.Sprintf("SELECT pg_advisory_lock(hashtext('%s'))", key)
	_, lockError := connection.ExecContext(ctx, lockSQL)
	if lockError != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("acquiring PostgreSQL advisory lock: %w", lockError)
	}

	return connection, nil
}

// Release releases the PostgreSQL advisory lock and returns the pinned connection to the
// pool.
//
// The unlock SQL runs against a detached context with a short timeout so a cancelled
// caller context does not leak the session-scoped advisory lock for the duration of the
// server's idle-in-transaction timeout.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (lock *PostgresAdvisoryLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	key, keyError := lock.key()
	if keyError != nil {
		closeError := connection.Close()
		return joinLockReleaseErrors("validating PostgreSQL advisory lock key", keyError, closeError)
	}

	//nolint:gosec // key passed ValidateIdentifier above
	unlockSQL := fmt.Sprintf("SELECT pg_advisory_unlock(hashtext('%s'))", key)
	return releaseLockSQL(ctx, connection, unlockSQL, "releasing PostgreSQL advisory lock")
}

// TryAcquire attempts to acquire the advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (lock *PostgresAdvisoryLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	key, keyError := lock.key()
	if keyError != nil {
		return nil, keyError
	}
	return tryAcquirePostgresAdvisoryLock(ctx, database, key, "PostgreSQL")
}

// key returns the effective lock key, falling back to the default when the field is
// empty. Returns a wrapped ErrInvalidIdentifier when LockKey is set to a value that fails
// ValidateIdentifier, so callers can refuse to run the SQL rather than enable injection.
//
// Returns string which is the text the lock will hash.
// Returns error when LockKey fails ValidateIdentifier.
func (lock *PostgresAdvisoryLock) key() (string, error) {
	if lock.LockKey == "" {
		return DefaultHistoryTableName, nil
	}
	if err := ValidateIdentifier(lock.LockKey); err != nil {
		return "", fmt.Errorf("PostgresAdvisoryLock.LockKey: %w", err)
	}
	return lock.LockKey, nil
}

// TableBasedLock implements LockStrategy using a dedicated lock table with SELECT ... FOR
// UPDATE.
//
// This is compatible with PgBouncer in transaction mode where advisory locks are not
// available.
type TableBasedLock struct {
	// heldTransaction holds the open transaction that maintains the FOR UPDATE lock.
	heldTransaction *sql.Tx

	// CreateLockTableSQL is the DDL statement for creating the lock table.
	CreateLockTableSQL string
}

// Acquire pins a connection, creates the lock table if needed, inserts a lock row, and
// acquires a FOR UPDATE lock within a held transaction.
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

// Release commits the held transaction (releasing the FOR UPDATE lock) and closes the
// pinned connection.
//
// Commit is preferred over rollback because the FOR UPDATE row lock is released either
// way; commit additionally surfaces any commit error to the caller and matches the policy
// used by TableBasedSeedLock. The caller context is intentionally ignored: *sql.Tx Commit
// and Rollback do not honour a per-call context, so wrapping with detachContextForRelease
// would buy no extra safety here.
//
// Takes connection (*sql.Conn) which is the pinned connection to close.
//
// Returns error when the transaction commit or connection close fails.
func (lock *TableBasedLock) Release(ctx context.Context, connection *sql.Conn) error {
	_ = ctx
	if connection == nil {
		return nil
	}

	var commitError error
	if lock.heldTransaction != nil {
		commitError = lock.heldTransaction.Commit()
		lock.heldTransaction = nil
	}

	closeError := connection.Close()

	return joinLockReleaseErrors("committing table lock transaction", commitError, closeError)
}

// acquireWithMode pins a connection, creates the lock table, inserts a lock row, and
// acquires a row lock using the specified lock mode.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
// Takes lockMode (string) which specifies the row lock clause (e.g. "FOR UPDATE" or "FOR
// UPDATE NOWAIT").
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns error when any step of the acquisition fails.
func (lock *TableBasedLock) acquireWithMode(
	ctx context.Context,
	database *sql.DB,
	lockMode string,
) (*sql.Conn, error) {
	if lock.heldTransaction != nil {
		return nil, errLockAlreadyHeld
	}
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

// mysqlNumberedError matches the typed error shape exposed by the go-sql-driver/mysql
// driver and structurally equivalent forks such as MariaDB clients.
//
// Detecting the driver-typed error through this shape lets isLockNotAvailableError catch
// the MySQL lock-wait-timeout (error 1205) even when the surrounding code wraps the error
// chain through fmt.Errorf, because errors.As walks the wrapped chain. Keeping the
// declaration local avoids importing the driver as a hard dependency from this adapter.
type mysqlNumberedError interface {
	error

	// Number returns the driver-assigned MySQL error number.
	Number() uint16
}

// postgresSQLStateError matches the typed error shape exposed by PostgreSQL drivers that
// report a SQLSTATE code through a SQLState method (lib/pq and pgx/stdlib both qualify).
//
// Detecting the SQLSTATE through this shape lets isLockNotAvailableError catch the
// lock_not_available code (55P03) even when the error chain is wrapped through
// fmt.Errorf, because errors.As walks the wrapped chain. Keeping the declaration local
// avoids importing a postgres driver as a hard dependency from this adapter.
type postgresSQLStateError interface {
	error

	// SQLState returns the five-character SQLSTATE code reported by the server.
	SQLState() string
}

// isLockNotAvailableError reports whether err indicates the lock could not be acquired.
//
// The check prefers driver-typed sentinels when present so wrapped errors still surface
// the correct intent. PostgreSQL is matched through any driver error exposing a SQLState
// method that returns 55P03 (lock_not_available), and MySQL via any driver that exposes a
// Number() uint16 method (go-sql-driver/mysql and structurally compatible forks) is
// matched by error 1205 (lock wait timeout exceeded).
//
// For dialects whose driver type is not statically importable here, or for drivers that
// erase the typed shape entirely, the helper falls back to substring matching as a last
// resort. The substring fallback keeps third-party MySQL-compatible drivers working
// without forcing a hard dependency on a specific driver package.
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when the error matches a known lock-not-available pattern.
func isLockNotAvailableError(err error) bool {
	if err == nil {
		return false
	}
	if pgError, ok := errors.AsType[postgresSQLStateError](err); ok && pgError.SQLState() == postgresErrorCodeLockNotAvailable {
		return true
	}
	if mysqlError, ok := errors.AsType[mysqlNumberedError](err); ok && mysqlError.Number() == mysqlErrorNumberLockWaitTimeoutUint {
		return true
	}
	message := err.Error()
	return strings.Contains(message, postgresErrorCodeLockNotAvailable) ||
		strings.Contains(message, "could not obtain lock") ||
		strings.Contains(message, mysqlErrorNumberLockWaitTimeout) ||
		strings.Contains(message, "Lock wait timeout exceeded")
}

// MySQLAdvisoryLock implements LockStrategy using MySQL's GET_LOCK and RELEASE_LOCK
// functions. The lock is session-scoped, so a dedicated connection is pinned from the
// pool.
//
// LockKey, when non-empty, MUST be a safe SQL identifier per ValidateIdentifier. The key
// is interpolated into GET_LOCK / RELEASE_LOCK SQL, so unsafe input would enable
// injection. Acquire / TryAcquire / Release validate the key on every call and return a
// wrapped ErrInvalidIdentifier when validation fails.
type MySQLAdvisoryLock struct {
	// LockKey is the name passed to GET_LOCK/RELEASE_LOCK.
	//
	// Distinct keys allow multiple MigrationService instances against the same database to
	// run concurrently without colliding. When empty, the key defaults to
	// DefaultHistoryTableName.
	LockKey string
}

// Acquire pins a connection from the pool and acquires a MySQL advisory lock using
// GET_LOCK with an indefinite timeout.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (lock *MySQLAdvisoryLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	key, keyError := lock.key()
	if keyError != nil {
		return nil, keyError
	}
	//nolint:gosec // key passed ValidateIdentifier above
	query := fmt.Sprintf("SELECT GET_LOCK('%s', -1)", key)
	return mysqlGetLock(ctx, database, query)
}

// TryAcquire attempts to acquire the MySQL advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (lock *MySQLAdvisoryLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	key, keyError := lock.key()
	if keyError != nil {
		return nil, keyError
	}
	//nolint:gosec // key passed ValidateIdentifier above
	query := fmt.Sprintf("SELECT GET_LOCK('%s', 0)", key)
	return mysqlGetLock(ctx, database, query)
}

// mysqlGetLock pins a connection from the pool and executes the given GET_LOCK query.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
// Takes query (string) which is the GET_LOCK SQL statement to execute.
//
// Returns *sql.Conn which is the pinned connection holding the lock.
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

	if !acquired.Valid {
		_ = connection.Close()
		return nil, errors.New("MySQL GET_LOCK returned NULL, indicating a lock error (deadlock or killed session)")
	}
	if acquired.Int64 != 1 {
		_ = connection.Close()
		return nil, querier_domain.ErrLockNotAcquired
	}

	return connection, nil
}

// Release releases the MySQL advisory lock and returns the pinned connection to the pool.
//
// The unlock SQL runs against a detached context with a short timeout so a cancelled
// caller context does not strand the lock; MySQL would otherwise hold the session-scoped
// lock until the session was idle-timeout reaped (default 8h).
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (lock *MySQLAdvisoryLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	key, keyError := lock.key()
	if keyError != nil {
		closeError := connection.Close()
		return joinLockReleaseErrors("validating MySQL advisory lock key", keyError, closeError)
	}

	//nolint:gosec // key passed ValidateIdentifier above
	unlockSQL := fmt.Sprintf("SELECT RELEASE_LOCK('%s')", key)
	return releaseLockSQL(ctx, connection, unlockSQL, "releasing MySQL advisory lock")
}

// key returns the effective lock key, falling back to the default when the field is
// empty. Returns a wrapped ErrInvalidIdentifier when LockKey fails ValidateIdentifier;
// see PostgresAdvisoryLock.key for the rationale.
//
// Returns string which is the GET_LOCK identifier.
// Returns error when LockKey fails ValidateIdentifier.
func (lock *MySQLAdvisoryLock) key() (string, error) {
	if lock.LockKey == "" {
		return DefaultHistoryTableName, nil
	}
	if err := ValidateIdentifier(lock.LockKey); err != nil {
		return "", fmt.Errorf("MySQLAdvisoryLock.LockKey: %w", err)
	}
	return lock.LockKey, nil
}

// PostgresAdvisorySeedLock implements LockStrategy using PostgreSQL's pg_advisory_lock
// function, keyed on the literal "piko_seeds" so it never collides with the migration
// lock.
//
// The "piko_seeds" key is validated through validateLockKey on every Acquire / TryAcquire
// / Release path even though it is a hardcoded constant: this guarantees that any future
// edit to the constant cannot silently enable injection, and applies the same
// defence-in-depth policy used by PostgresAdvisoryLock.
type PostgresAdvisorySeedLock struct{}

// Acquire pins a connection from the pool and acquires a PostgreSQL advisory lock keyed
// on hashtext('piko_seeds').
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (*PostgresAdvisorySeedLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	if keyError := validateLockKey(seedLockKey, "PostgresAdvisorySeedLock"); keyError != nil {
		return nil, keyError
	}

	connection, connectionError := database.Conn(ctx)
	if connectionError != nil {
		return nil, fmt.Errorf(errorFormatPinningConnection, connectionError)
	}

	//nolint:gosec // seedLockKey is a validated constant
	lockSQL := fmt.Sprintf("SELECT pg_advisory_lock(hashtext('%s'))", seedLockKey)
	_, lockError := connection.ExecContext(ctx, lockSQL)
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
	if keyError := validateLockKey(seedLockKey, "PostgresAdvisorySeedLock"); keyError != nil {
		return nil, keyError
	}
	return tryAcquirePostgresAdvisoryLock(ctx, database, seedLockKey, "PostgreSQL seed")
}

// Release releases the PostgreSQL seed advisory lock and returns the pinned connection to
// the pool. The unlock SQL runs against a detached context with a short timeout so a
// cancelled caller context does not leak the session-scoped advisory lock.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (*PostgresAdvisorySeedLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	if keyError := validateLockKey(seedLockKey, "PostgresAdvisorySeedLock"); keyError != nil {
		closeError := connection.Close()
		return joinLockReleaseErrors("validating PostgreSQL seed advisory lock key", keyError, closeError)
	}

	//nolint:gosec // seedLockKey is a validated constant
	unlockSQL := fmt.Sprintf("SELECT pg_advisory_unlock(hashtext('%s'))", seedLockKey)
	return releaseLockSQL(ctx, connection, unlockSQL, "releasing PostgreSQL seed advisory lock")
}

// MySQLAdvisorySeedLock implements LockStrategy using MySQL's GET_LOCK and RELEASE_LOCK
// functions, keyed on the literal "piko_seeds" so it never collides with the migration
// lock.
//
// As with PostgresAdvisorySeedLock the key is validated through validateLockKey on every
// Acquire / TryAcquire / Release path for defence in depth, even though the value is a
// compile-time constant.
type MySQLAdvisorySeedLock struct{}

// Acquire pins a connection from the pool and acquires a MySQL advisory lock using
// GET_LOCK with an indefinite timeout.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the connection cannot be pinned or the lock fails.
func (*MySQLAdvisorySeedLock) Acquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	if keyError := validateLockKey(seedLockKey, "MySQLAdvisorySeedLock"); keyError != nil {
		return nil, keyError
	}
	//nolint:gosec // seedLockKey is a validated constant
	query := fmt.Sprintf("SELECT GET_LOCK('%s', -1)", seedLockKey)
	return mysqlGetLock(ctx, database, query)
}

// TryAcquire attempts to acquire the MySQL seed advisory lock without blocking.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
//
// Returns *sql.Conn which is the pinned connection holding the advisory lock.
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (*MySQLAdvisorySeedLock) TryAcquire(ctx context.Context, database *sql.DB) (*sql.Conn, error) {
	if keyError := validateLockKey(seedLockKey, "MySQLAdvisorySeedLock"); keyError != nil {
		return nil, keyError
	}
	//nolint:gosec // seedLockKey is a validated constant
	query := fmt.Sprintf("SELECT GET_LOCK('%s', 0)", seedLockKey)
	return mysqlGetLock(ctx, database, query)
}

// Release releases the MySQL seed advisory lock and returns the pinned connection to the
// pool. The unlock SQL runs against a detached context with a short timeout so a
// cancelled caller context does not strand the session-scoped lock.
//
// Takes connection (*sql.Conn) which is the pinned connection to unlock and close.
//
// Returns error when the lock cannot be released or the connection cannot be closed.
func (*MySQLAdvisorySeedLock) Release(ctx context.Context, connection *sql.Conn) error {
	if connection == nil {
		return nil
	}

	if keyError := validateLockKey(seedLockKey, "MySQLAdvisorySeedLock"); keyError != nil {
		closeError := connection.Close()
		return joinLockReleaseErrors("validating MySQL seed advisory lock key", keyError, closeError)
	}

	//nolint:gosec // seedLockKey is a validated constant
	unlockSQL := fmt.Sprintf("SELECT RELEASE_LOCK('%s')", seedLockKey)
	return releaseLockSQL(ctx, connection, unlockSQL, "releasing MySQL seed advisory lock")
}

// TableBasedSeedLock implements LockStrategy via a dedicated lock table.
//
// Uses piko_seed_lock with SELECT ... FOR UPDATE. Used in PgBouncer transaction mode
// where session-scoped advisory locks are not available.
type TableBasedSeedLock struct {
	// heldTransaction holds the open transaction that maintains the FOR UPDATE lock.
	heldTransaction *sql.Tx

	// CreateLockTableSQL is the DDL statement for creating the seed lock table.
	CreateLockTableSQL string
}

// Acquire pins a connection, creates the seed lock table if needed, inserts a lock row,
// and acquires a FOR UPDATE lock within a held transaction.
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

// Release commits the held transaction (releasing the FOR UPDATE lock) and closes the
// pinned connection.
//
// Both commit and rollback release the row-level lock; we pick commit unconditionally so
// seed lock release behaves identically to migration lock release. The caller context is
// intentionally ignored: *sql.Tx Commit and Rollback do not honour a per-call context,
// matching the rationale documented on TableBasedLock.Release.
//
// Takes connection (*sql.Conn) which is the pinned connection to close.
//
// Returns error when the transaction commit or connection close fails.
func (lock *TableBasedSeedLock) Release(ctx context.Context, connection *sql.Conn) error {
	_ = ctx
	if connection == nil {
		return nil
	}

	var commitError error
	if lock.heldTransaction != nil {
		commitError = lock.heldTransaction.Commit()
		lock.heldTransaction = nil
	}

	closeError := connection.Close()

	return joinLockReleaseErrors("committing seed table lock transaction", commitError, closeError)
}

// acquireWithMode pins a connection, creates the lock table, inserts a lock row, and
// acquires a row lock using the specified lock mode.
//
// Takes database (*sql.DB) which is the connection pool to pin a connection from.
// Takes lockMode (string) which specifies the row lock clause (e.g. "FOR UPDATE" or "FOR
// UPDATE NOWAIT").
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns error when any step of the acquisition fails.
func (lock *TableBasedSeedLock) acquireWithMode(
	ctx context.Context,
	database *sql.DB,
	lockMode string,
) (*sql.Conn, error) {
	if lock.heldTransaction != nil {
		return nil, errLockAlreadyHeld
	}
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

// NoOpLock implements LockStrategy as a no-op. This is used for SQLite where file-level
// locking provides sufficient concurrency control.
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

// tableBasedLockOptions captures the dialect-independent inputs to the table-based
// locking pipeline. Only the table name and an error-message prefix differ between the
// migration and seed lock paths.
type tableBasedLockOptions struct {
	// createTableSQL is the dialect-specific CREATE TABLE IF NOT EXISTS statement that
	// idempotently creates the underlying lock table.
	createTableSQL string

	// tableName is the bare table name used in INSERT and SELECT statements.
	//
	// e.g. "piko_migration_lock" or "piko_seed_lock".
	tableName string

	// errorContextLower is a short lower-case prefix interpolated into diagnostic error
	// messages so callers can distinguish migration and seed lock failures (e.g. "seed " or
	// empty for migrations).
	errorContextLower string
}

// acquireTableBasedLock pins a connection from the pool, ensures the lock table exists,
// inserts the singleton row if missing, opens a transaction, and acquires the configured
// row lock. On any failure all resources are released before the error is returned.
//
// The shared helper exists so the migration and seed table-based lock paths stay
// byte-equivalent in semantics and only their table name + error messages diverge.
//
// Takes database (*sql.DB) which is the connection pool to pin from.
// Takes lockMode (string) which is the row lock clause appended to the SELECT (e.g. "FOR
// UPDATE", "FOR UPDATE NOWAIT").
// Takes options (tableBasedLockOptions) which describe the table-specific SQL and
// error-message prefixes.
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns *sql.Tx which is the active transaction maintaining the lock.
// Returns error which wraps querier_domain.ErrLockNotAcquired when the lock is not
// available, or any underlying connection / SQL error.
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
// pg_try_advisory_lock for the supplied lock key (hashed via PostgreSQL's hashtext
// function). On any failure the pinned connection is returned to the pool before the
// error bubbles up.
//
// Takes database (*sql.DB) which is the connection pool to pin from.
// Takes lockKey (string) which is the textual identifier used as the lock key (passed
// through hashtext at SQL time).
// Takes errorContext (string) which prefixes diagnostic error messages (e.g.
// "PostgreSQL", "PostgreSQL seed").
//
// Returns *sql.Conn which is the pinned connection holding the lock.
// Returns error which wraps querier_domain.ErrLockNotAcquired when the lock is already
// held, or any underlying connection / query error.
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
