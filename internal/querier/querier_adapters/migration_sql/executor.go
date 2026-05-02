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

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// insertPlaceholderVersion is the 1-based bind-parameter index for the version column.
	insertPlaceholderVersion = 1

	// insertPlaceholderName is the 1-based bind-parameter index for the name column.
	insertPlaceholderName = 2

	// insertPlaceholderChecksum is the 1-based bind-parameter index for the checksum column.
	insertPlaceholderChecksum = 3

	// insertPlaceholderAppliedAt is the 1-based bind-parameter index for the applied_at
	// column.
	insertPlaceholderAppliedAt = 4

	// insertPlaceholderDurationMs is the 1-based bind-parameter index for the duration_ms
	// column.
	insertPlaceholderDurationMs = 5

	// insertPlaceholderDownChecksum is the 1-based bind-parameter index for the
	// down_checksum column.
	insertPlaceholderDownChecksum = 6

	// insertPlaceholderLastStatement is the 1-based bind-parameter index for the
	// last_statement column.
	insertPlaceholderLastStatement = 7

	// insertPlaceholderDirty is the 1-based bind-parameter index for the dirty column.
	insertPlaceholderDirty = 8

	// clearDirtyPlaceholderDirty is the 1-based bind-parameter index for the dirty column in
	// the clearDirty UPDATE statement.
	clearDirtyPlaceholderDirty = 1

	// clearDirtyPlaceholderDurationMs is the 1-based bind-parameter index for the
	// duration_ms column in the clearDirty UPDATE statement.
	clearDirtyPlaceholderDurationMs = 2

	// clearDirtyPlaceholderVersion is the 1-based bind-parameter index for the version
	// column in the clearDirty UPDATE statement.
	clearDirtyPlaceholderVersion = 3

	// defaultStatementCapacity is the initial slice capacity used when collecting split
	// statements. It avoids the first few allocations for typical migration files without
	// over-allocating for empty inputs.
	defaultStatementCapacity = 8

	// defaultProgressUpdateBatchSize is the number of successful statements an executor
	// applies between writes to the last_statement column. A higher batch reduces the
	// per-statement UPDATE chatter but loses a finer-grained resume point on crash.
	defaultProgressUpdateBatchSize = 50
)

// queryRunner is the common interface satisfied by both *sql.DB and *sql.Conn, allowing
// the executor to route operations through a pinned connection when an advisory lock is
// held.
type queryRunner interface {
	// ExecContext executes a query without returning rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// QueryContext executes a query that returns rows.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRowContext executes a query expected to return at most one row.
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	// BeginTx starts a new transaction with the given options.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// execContextRunner is the minimal interface needed to execute a history-table write. It
// is satisfied by *sql.DB, *sql.Conn, and *sql.Tx, so the history helpers can run either
// inside a transaction (the SQL-standard engines) or directly on the pinned
// connection/pool (the non-transactional engines such as ClickHouse).
type execContextRunner interface {
	// ExecContext executes a query without returning rows.
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var (
	// errLockAlreadyHeld is returned by AcquireLock / TryAcquireLock when a lock connection
	// is already pinned. Acquiring again without releasing would orphan the previously
	// pinned connection, so the second acquire is refused rather than silently leaking it.
	errLockAlreadyHeld = errors.New("migration lock already held; release it before acquiring again")
)

// Executor implements MigrationExecutorPort using database/sql. It works for SQLite,
// PostgreSQL, MySQL, and ClickHouse via the DialectConfig strategy.
//
// Field order places the largest field (DialectConfig, which holds slices, maps, and
// function pointers) last so the preceding word-sized fields pack tightly under the
// fieldalignment govet check.
type Executor struct {
	// database holds the underlying database connection pool.
	database *sql.DB

	// clock is the time source used to stamp migration durations; never nil after
	// NewExecutor, which defaults it to clock.RealClock().
	clock clock.Clock

	// pinnedConnection holds a dedicated connection used when an advisory lock is held,
	// ensuring all operations run on the same session.
	pinnedConnection *sql.Conn

	// appliedVersionsSQL caches the rendered SELECT used by AppliedVersions. The history
	// table name comes from dialectConfig.HistoryTable() which has already passed
	// ValidateIdentifier upstream, so the SQL is safe to interpolate once at construction
	// time and reuse for every call.
	appliedVersionsSQL string

	// dialectConfig holds the dialect-specific SQL and locking behaviour.
	dialectConfig DialectConfig

	// progressUpdateBatchSize is the number of successful statements that may be applied
	// between writes to the last_statement column, set via WithProgressBatchSize. Zero or
	// below means use defaultProgressUpdateBatchSize.
	progressUpdateBatchSize int
}

var (
	_ querier_domain.MigrationExecutorPort = (*Executor)(nil)
)

// NewExecutor creates a new SQL-based migration executor.
//
// The executor caches a few dialect-derived SQL strings at construction time so the
// per-migration hot path does not pay for repeated fmt.Sprintf interpolation. The history
// table identifier is validated upstream by WithHistoryTable / ValidateIdentifier, which
// is what makes the cached form safe to retain across calls. NewExecutor re-validates the
// resolved HistoryTable() at construction time as defence in depth: an upstream
// regression that bypassed WithHistoryTable would otherwise let an unsafe identifier flow
// into the cached SELECT statement and remain there for the lifetime of the executor.
//
// Takes database (*sql.DB) which is the database connection.
// Takes dialectConfig (DialectConfig) which provides dialect-specific SQL and locking
// behaviour.
// Takes options (...ExecutorOption) which customise the executor, for example
// WithExecutorClock to inject a deterministic clock in tests.
//
// Returns *Executor which is ready to execute migrations.
//
// Panics when the resolved history table fails ValidateIdentifier, which is fail-fast at
// process start rather than at the first migration call.
func NewExecutor(database *sql.DB, dialectConfig DialectConfig, options ...ExecutorOption) *Executor {
	historyTable := dialectConfig.HistoryTable()
	if validationError := ValidateIdentifier(historyTable); validationError != nil {
		panic(fmt.Errorf("NewExecutor history table: %w", validationError))
	}
	if dialectConfig.PlaceholderFunc == nil {
		panic(errors.New("NewExecutor: dialect config has a nil PlaceholderFunc"))
	}
	executor := &Executor{
		database:                database,
		dialectConfig:           dialectConfig,
		progressUpdateBatchSize: defaultProgressUpdateBatchSize,
		clock:                   clock.RealClock(),
	}
	for _, option := range options {
		option(executor)
	}
	executor.appliedVersionsSQL = renderAppliedVersionsSQL(historyTable, dialectConfig.SelectHistoryFinal)
	return executor
}

// ExecutorOption customises an Executor at construction time.
type ExecutorOption func(*Executor)

// WithExecutorClock overrides the time source used to record migration durations.
//
// The default is clock.RealClock(); tests inject a mock clock to assert recorded
// durations deterministically. A nil clock is ignored so the default is preserved.
//
// Takes source (clock.Clock) which is the time source to use for duration recording.
//
// Returns ExecutorOption which applies the clock override at construction time.
func WithExecutorClock(source clock.Clock) ExecutorOption {
	return func(executor *Executor) {
		if source != nil {
			executor.clock = source
		}
	}
}

// WithProgressBatchSize overrides the number of successful statements an up migration
// applies between writes to the last_statement column. A higher batch reduces
// per-statement UPDATE chatter but loses a finer-grained resume point on crash.
//
// The default is defaultProgressUpdateBatchSize. A size of zero or below is ignored so
// the default is preserved, matching the zero-value fallback in progressBatchSize.
//
// Takes size (int) which is the number of statements between progress writes.
//
// Returns ExecutorOption which applies the batch-size override at construction time.
func WithProgressBatchSize(size int) ExecutorOption {
	return func(executor *Executor) {
		if size > 0 {
			executor.progressUpdateBatchSize = size
		}
	}
}

// renderAppliedVersionsSQL returns the cached SELECT statement used by AppliedVersions.
// The historyTable string has already passed ValidateIdentifier upstream so the
// interpolation is safe.
//
// Takes historyTable (string) which is the validated history table identifier.
// Takes final (bool) which, when true, appends a FINAL modifier so a ReplacingMergeTree
// history table (ClickHouse) collapses duplicate version rows at read time.
//
// Returns string which is the SELECT statement for the applied-migrations query.
func renderAppliedVersionsSQL(historyTable string, final bool) string {
	finalClause := ""
	if final {
		finalClause = " FINAL"
	}
	return fmt.Sprintf( //nolint:gosec // historyTable is validated upstream via ValidateIdentifier
		"SELECT version, name, checksum, applied_at, duration_ms, down_checksum, last_statement, dirty "+
			"FROM %s%s ORDER BY version",
		historyTable, finalClause,
	)
}

// EnsureMigrationTable creates the piko_migrations table if it does not exist and applies
// any pending AlterStatements idempotently.
//
// Returns error when the table cannot be created or altered.
func (executor *Executor) EnsureMigrationTable(ctx context.Context) error {
	_, createError := executor.queryExecutor().ExecContext(ctx, executor.dialectConfig.CreateTableSQL)
	if createError != nil {
		return fmt.Errorf("creating migration table: %w", createError)
	}

	for _, statement := range executor.dialectConfig.AlterStatements {
		_, alterError := executor.queryExecutor().ExecContext(ctx, statement)
		if alterError != nil && !isDuplicateColumnError(alterError) {
			return fmt.Errorf("altering migration table: %w", alterError)
		}
	}

	return nil
}

// AcquireLock acquires the database-specific advisory lock.
//
// For strategies that require connection pinning (e.g. PostgreSQL), a dedicated
// connection is held for the duration until ReleaseLock is called. After acquiring the
// lock, any configured PreMigrationStatements are executed.
//
// Returns error when the lock cannot be acquired or pre-migration statements fail.
func (executor *Executor) AcquireLock(ctx context.Context) error {
	if executor.pinnedConnection != nil {
		return errLockAlreadyHeld
	}
	connection, lockError := executor.dialectConfig.LockStrategy.Acquire(ctx, executor.database)
	if lockError != nil {
		return lockError
	}
	executor.pinnedConnection = connection

	if preMigrationError := executor.executePreMigrationStatements(ctx); preMigrationError != nil {
		releaseError := executor.ReleaseLock(ctx)
		return joinLockReleaseErrors("releasing lock after pre-migration failure", releaseError, preMigrationError)
	}

	return nil
}

// TryAcquireLock attempts to acquire the advisory lock without blocking.
//
// After acquiring the lock, any configured PreMigrationStatements are executed.
//
// Returns error when the lock cannot be acquired, including
// querier_domain.ErrLockNotAcquired if the lock is already held.
func (executor *Executor) TryAcquireLock(ctx context.Context) error {
	if executor.pinnedConnection != nil {
		return errLockAlreadyHeld
	}
	connection, lockError := executor.dialectConfig.LockStrategy.TryAcquire(ctx, executor.database)
	if lockError != nil {
		return lockError
	}
	executor.pinnedConnection = connection

	if preMigrationError := executor.executePreMigrationStatements(ctx); preMigrationError != nil {
		releaseError := executor.ReleaseLock(ctx)
		return joinLockReleaseErrors("releasing lock after pre-migration failure", releaseError, preMigrationError)
	}

	return nil
}

// ReleaseLock releases the database-specific advisory lock and returns any pinned
// connection to the pool.
//
// Returns error when the lock cannot be released.
func (executor *Executor) ReleaseLock(ctx context.Context) error {
	connection := executor.pinnedConnection
	executor.pinnedConnection = nil
	return executor.dialectConfig.LockStrategy.Release(ctx, connection)
}

// ExecuteMigration runs a single migration's SQL content.
//
// For up migrations it INSERTs a record; for down migrations it DELETEs the record. When
// useTransaction is true, both the SQL and history update happen atomically.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL and
// metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether this is an up
// or down migration.
// Takes useTransaction (bool) which controls whether the migration and history update are
// wrapped in a single transaction.
//
// Returns error when the migration SQL or history update fails.
//
// The migration content is always split into individual statements (see execStatements)
// and executed one at a time, regardless of the dialect's SplitStatements flag. That flag
// governs only the seed executor; migrations split unconditionally because per-statement
// progress tracking (the last_statement resume point advanced via SkipUpTo) and the
// non-transactional dirty-resume path both depend on executing statements individually.
func (executor *Executor) ExecuteMigration(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	useTransaction bool,
) error {
	start := executor.clock.Now()

	if executor.dialectConfig.DisableTransactions {
		useTransaction = false
	}

	if useTransaction {
		return executor.executeInTransaction(ctx, migration, direction, start)
	}
	return executor.executeWithoutTransaction(ctx, migration, direction, start)
}

// queryExecutor returns the pinned connection if one is held (i.e. under an advisory
// lock), or the connection pool otherwise.
//
// Returns queryRunner which is either the pinned connection or the database pool.
func (executor *Executor) queryExecutor() queryRunner {
	if executor.pinnedConnection != nil {
		return executor.pinnedConnection
	}
	return executor.database
}

// progressBatchSize returns the effective progress-update batch size, falling back to
// defaultProgressUpdateBatchSize when the executor was constructed without an explicit
// override. Centralising the fallback keeps the per-statement hot path in
// executeWithoutTransactionUp from branching on the zero value.
//
// Returns int which is the number of successful statements between progress writes.
func (executor *Executor) progressBatchSize() int {
	if executor.progressUpdateBatchSize <= 0 {
		return defaultProgressUpdateBatchSize
	}
	return executor.progressUpdateBatchSize
}

// executePreMigrationStatements runs all configured PreMigrationStatements on the current
// query executor.
//
// Returns error when any statement fails to execute.
func (executor *Executor) executePreMigrationStatements(ctx context.Context) error {
	for _, statement := range executor.dialectConfig.PreMigrationStatements {
		if _, execError := executor.queryExecutor().ExecContext(ctx, statement); execError != nil {
			return fmt.Errorf("executing pre-migration statement %q: %w", statement, execError)
		}
	}
	return nil
}

// executeInTransaction runs the migration SQL and history update within a single database
// transaction. Statements are split and executed individually for better error messages,
// but the transaction ensures atomicity so no dirty state tracking is needed.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL and
// metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether this is an up
// or down migration.
// Takes start (time.Time) which records when execution began for duration tracking.
//
// Returns error when the transaction, migration SQL, or history update fails.
func (executor *Executor) executeInTransaction(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	start time.Time,
) error {
	transaction, beginError := executor.queryExecutor().BeginTx(ctx, nil)
	if beginError != nil {
		return fmt.Errorf("beginning transaction: %w", beginError)
	}
	defer rollbackIfActive(ctx, transaction, "migration transaction")

	if _, execError := executor.execStatements(
		ctx, transaction, string(migration.Content), migration.Version, migration.SkipUpTo,
	); execError != nil {
		return fmt.Errorf("executing SQL: %w", execError)
	}

	durationMs := executor.clock.Now().Sub(start).Milliseconds()

	if historyError := executor.updateHistory(
		ctx, transaction, migration, direction, start, durationMs,
	); historyError != nil {
		return historyError
	}

	if commitError := transaction.Commit(); commitError != nil {
		return fmt.Errorf("committing transaction: %w", commitError)
	}

	return nil
}

// rollbackIfActive rolls the transaction back when the deferred cleanup fires. The
// sql.ErrTxDone branch is the harmless case where Commit succeeded earlier in the
// function; every other error is logged so operators can spot rollback failures rather
// than silently discarding them.
//
// Takes transaction (*sql.Tx) which is the transaction being released.
// Takes label (string) which prefixes log messages to identify the rollback site.
func rollbackIfActive(ctx context.Context, transaction *sql.Tx, label string) {
	rollbackError := transaction.Rollback()
	if rollbackError == nil || errors.Is(rollbackError, sql.ErrTxDone) {
		return
	}
	_, l := logger_domain.From(ctx, log)
	l.Warn("transaction rollback failed",
		logger_domain.String("site", label),
		logger_domain.Error(rollbackError),
	)
}

// executeWithoutTransaction runs the migration SQL outside a transaction with
// per-statement dirty state tracking, delegating to direction-specific helpers.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL and
// metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether this is an up
// or down migration.
// Takes start (time.Time) which records when execution began for duration tracking.
//
// Returns error when the migration SQL or history update fails.
func (executor *Executor) executeWithoutTransaction(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	start time.Time,
) error {
	if direction == querier_dto.MigrationDirectionUp {
		return executor.executeWithoutTransactionUp(ctx, migration, start)
	}

	return executor.executeWithoutTransactionDown(ctx, migration, start)
}

// executeWithoutTransactionUp handles non-transactional up migrations with per-statement
// dirty state tracking. On full success the record is finalised with dirty = FALSE.
//
// Progress writes to the last_statement column are batched: by default every
// defaultProgressUpdateBatchSize successful statements (configurable via
// progressUpdateBatchSize) we emit one UPDATE rather than one per statement. On failure
// the last completed statement index is always flushed so a subsequent retry can resume
// from the correct point. The final clearDirty UPDATE also implicitly persists the final
// position because it leaves last_statement unchanged after the last batched write.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL and
// metadata.
// Takes start (time.Time) which records when execution began for duration tracking.
//
// Returns error when the migration SQL or history update fails.
func (executor *Executor) executeWithoutTransactionUp(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	start time.Time,
) error {
	if executor.dialectConfig.AppendOnlyHistory {
		return executor.executeAppendOnlyUp(ctx, migration, start)
	}

	isRetry := migration.SkipUpTo >= 0

	statements, splitError := splitStatementsWithOptions(string(migration.Content), executor.dialectConfig.BackslashEscapes)
	if splitError != nil {
		return fmt.Errorf("splitting migration %d statements: %w", migration.Version, splitError)
	}

	if !isRetry {
		if preRecordError := executor.preRecordDirtyMigration(
			ctx, migration, start,
		); preRecordError != nil {
			return preRecordError
		}
	}

	skipUpTo := migration.SkipUpTo
	batchSize := executor.progressBatchSize()
	lastFlushedIndex := skipUpTo

	for i := range statements {
		if i <= skipUpTo {
			continue
		}
		if cancelErr := ctx.Err(); cancelErr != nil {
			return executor.handleCancellationCheckpoint(ctx, migration.Version, i, lastFlushedIndex, cancelErr)
		}
		if runError := executor.runMigrationStatement(ctx, migration.Version, statements, i); runError != nil {
			return runError
		}
		if shouldFlushProgress(i, lastFlushedIndex, batchSize, len(statements)) {
			executor.flushProgressCheckpoint(ctx, migration.Version, i)
			lastFlushedIndex = i
		}
	}

	return executor.clearDirty(ctx, migration.Version, start)
}

// handleCancellationCheckpoint flushes the resume checkpoint when the context is
// cancelled part-way through a non-transactional up migration, then returns the
// cancellation error.
//
// The loop guard fires before the statement at currentIndex runs, so the last completed
// statement is currentIndex-1. Progress writes are batched, so up to batchSize-1
// already-applied statements may sit between lastFlushedIndex and currentIndex-1 with no
// checkpoint. Without flushing, a resume would set SkipUpTo to lastFlushedIndex and
// re-execute those possibly non-idempotent statements (M8). We flush currentIndex-1 here,
// mirroring the per-statement failure path, but on a detached context because the parent
// ctx is already cancelled and would reject the UPDATE. Any flush error is joined onto
// the cancellation error rather than dropped silently.
//
// Takes version (int64) which identifies the migration record.
// Takes currentIndex (int) which is the 0-based index of the statement about to run.
// Takes lastFlushedIndex (int) which is the index of the most recent flushed checkpoint.
// Takes cancelErr (error) which is the context cancellation cause to report.
//
// Returns error which is the cancellation error, joined with any checkpoint-flush error.
func (executor *Executor) handleCancellationCheckpoint(
	ctx context.Context,
	version int64,
	currentIndex int,
	lastFlushedIndex int,
	cancelErr error,
) error {
	cancellationError := fmt.Errorf(
		"migration %d cancelled before statement %d: %w", version, currentIndex+1, cancelErr,
	)

	lastCompletedIndex := currentIndex - 1
	if lastCompletedIndex <= lastFlushedIndex {
		return cancellationError
	}

	detachedCtx := context.WithoutCancel(ctx)
	if progressError := executor.updateStatementProgress(detachedCtx, version, lastCompletedIndex); progressError != nil {
		return errors.Join(cancellationError, fmt.Errorf("flushing resume checkpoint: %w", progressError))
	}
	return cancellationError
}

// runMigrationStatement executes one statement of a non-transactional up migration. On
// failure it flushes the resume checkpoint to the previous statement index so a later
// retry resumes from the correct point, joining any checkpoint-flush error onto the
// statement error rather than dropping it silently.
//
// Takes version (int64) which identifies the migration for error messages.
// Takes statements ([]string) which holds the split migration statements.
// Takes index (int) which is the statement to execute.
//
// Returns error when the statement fails, wrapping the resume-checkpoint flush error too.
func (executor *Executor) runMigrationStatement(
	ctx context.Context,
	version int64,
	statements []string,
	index int,
) error {
	if _, execError := executor.queryExecutor().ExecContext(ctx, statements[index]); execError != nil {
		statementError := fmt.Errorf(
			"executing SQL: statement %d/%d of migration %d: %w",
			index+1, len(statements), version, execError,
		)
		if progressError := executor.updateStatementProgress(ctx, version, index-1); progressError != nil {
			return errors.Join(statementError, fmt.Errorf("flushing resume checkpoint: %w", progressError))
		}
		return statementError
	}
	return nil
}

// executeAppendOnlyUp handles non-transactional up migrations for engines that cannot
// UPDATE rows in place (ClickHouse).
//
// It executes every statement and then records a single completed history row (dirty =
// false). Because there is no in-place UPDATE, the dirty pre-record, per-statement
// progress, and resume machinery is skipped: a migration that fails part-way leaves no
// history row and is re-run from the start on the next invocation. ClickHouse DDL is
// typically idempotent (IF NOT EXISTS), and ClickHouse has no transactions to make
// partial application atomic regardless.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL and
// metadata.
// Takes start (time.Time) which records when execution began for duration tracking.
//
// Returns error when the migration SQL or history insert fails.
func (executor *Executor) executeAppendOnlyUp(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	start time.Time,
) error {
	if _, execError := executor.execStatements(
		ctx, executor.queryExecutor(), string(migration.Content), migration.Version, migration.SkipUpTo,
	); execError != nil {
		return fmt.Errorf("executing SQL: %w", execError)
	}

	durationMs := executor.clock.Now().Sub(start).Milliseconds()
	return executor.recordCompletedMigration(ctx, migration, start, durationMs)
}

// shouldFlushProgress reports whether the executor should write a progress checkpoint
// after completing the statement at currentIndex. We flush either when batchSize
// successful statements have accumulated since lastFlushedIndex or when currentIndex is
// the last statement in the migration.
//
// Takes currentIndex (int) which is the 0-based index of the statement just completed.
// Takes lastFlushedIndex (int) which is the index of the most recent flushed checkpoint.
// Takes batchSize (int) which is the configured number of statements per batch.
// Takes totalStatements (int) which is the migration's statement count.
//
// Returns bool which is true when a progress write should occur.
func shouldFlushProgress(currentIndex, lastFlushedIndex, batchSize, totalStatements int) bool {
	if currentIndex == totalStatements-1 {
		return true
	}
	return currentIndex-lastFlushedIndex >= batchSize
}

// executeWithoutTransactionDown handles non-transactional down migrations. Down
// migrations do not use dirty state tracking since they delete the history record on
// success.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration SQL and
// metadata.
// Takes start (time.Time) which records when execution began for duration tracking.
//
// Returns error when the migration SQL or history update fails.
func (executor *Executor) executeWithoutTransactionDown(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	start time.Time,
) error {
	if _, execError := executor.execStatements(
		ctx, executor.queryExecutor(), string(migration.Content), migration.Version, migration.SkipUpTo,
	); execError != nil {
		return fmt.Errorf("executing SQL: %w", execError)
	}

	durationMs := executor.clock.Now().Sub(start).Milliseconds()

	if executor.dialectConfig.DisableTransactions {
		return executor.updateHistory(
			ctx, executor.queryExecutor(), migration, querier_dto.MigrationDirectionDown, start, durationMs,
		)
	}

	transaction, beginError := executor.queryExecutor().BeginTx(ctx, nil)
	if beginError != nil {
		return fmt.Errorf("beginning history transaction: %w", beginError)
	}
	defer rollbackIfActive(ctx, transaction, "down-migration history transaction")

	if historyError := executor.updateHistory(
		ctx, transaction, migration, querier_dto.MigrationDirectionDown, start, durationMs,
	); historyError != nil {
		return historyError
	}

	if commitError := transaction.Commit(); commitError != nil {
		return fmt.Errorf("committing history: %w", commitError)
	}

	return nil
}

// preRecordDirtyMigration inserts a migration history record with dirty = TRUE and
// last_statement = -1 before any SQL statements are executed. This ensures the migration
// is recorded as in-progress even if the process crashes during execution.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration metadata.
// Takes start (time.Time) which is the timestamp to record.
//
// Returns error when the INSERT statement fails.
func (executor *Executor) preRecordDirtyMigration(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	start time.Time,
) error {
	exists, existsError := executor.historyRecordExists(ctx, migration.Version)
	if existsError != nil {
		return fmt.Errorf("checking existing dirty record for migration %d: %w", migration.Version, existsError)
	}
	if exists {
		return nil
	}

	_, insertError := executor.queryExecutor().ExecContext(ctx, executor.historyInsertSQL(),
		migration.Version, migration.Name, migration.Checksum, start.UTC(), int64(0),
		nullableDownChecksum(migration.DownChecksum), -1, true,
	)
	if insertError != nil {
		return fmt.Errorf("pre-recording dirty migration %d: %w", migration.Version, insertError)
	}

	return nil
}

// historyInsertSQL builds the eight-column INSERT statement for the migration history
// table using the dialect's placeholder syntax. Shared by the dirty pre-record path, the
// transactional history update, and the append-only completed-record path so the column
// order and placeholder layout stay in lockstep across all three.
//
// Returns string which is the complete INSERT statement.
func (executor *Executor) historyInsertSQL() string {
	placeholder := executor.dialectConfig.PlaceholderFunc
	return fmt.Sprintf( //nolint:gosec // history table name is a configured identifier under caller control
		"INSERT INTO %s (version, name, checksum, applied_at, duration_ms, down_checksum, last_statement, dirty) VALUES (%s, %s, %s, %s, %s, %s, %s, %s)",
		executor.dialectConfig.HistoryTable(),
		placeholder(insertPlaceholderVersion),
		placeholder(insertPlaceholderName),
		placeholder(insertPlaceholderChecksum),
		placeholder(insertPlaceholderAppliedAt),
		placeholder(insertPlaceholderDurationMs),
		placeholder(insertPlaceholderDownChecksum),
		placeholder(insertPlaceholderLastStatement),
		placeholder(insertPlaceholderDirty),
	)
}

// nullableDownChecksum returns the down-migration checksum as a bind value, or nil when
// the migration has no down checksum so the column is stored as NULL.
//
// Takes downChecksum (string) which is the down-migration checksum (may be empty).
//
// Returns any which is the checksum string or nil.
func nullableDownChecksum(downChecksum string) any {
	if downChecksum == "" {
		return nil
	}
	return downChecksum
}

// recordCompletedMigration inserts a finalised (dirty = false) migration history row for
// the append-only execution path. Unlike preRecordDirtyMigration it records the real
// duration and a clean dirty flag, and unlike clearDirty it does not rely on an in-place
// UPDATE, so it is safe on ClickHouse MergeTree.
//
// Takes migration (querier_dto.MigrationRecord) which holds the migration metadata.
// Takes appliedAt (time.Time) which is the timestamp to record.
// Takes durationMs (int64) which is the execution duration in milliseconds.
//
// Returns error when the INSERT statement fails.
func (executor *Executor) recordCompletedMigration(
	ctx context.Context,
	migration querier_dto.MigrationRecord,
	appliedAt time.Time,
	durationMs int64,
) error {
	_, insertError := executor.queryExecutor().ExecContext(ctx, executor.historyInsertSQL(),
		migration.Version, migration.Name, migration.Checksum, appliedAt.UTC(), durationMs,
		nullableDownChecksum(migration.DownChecksum), nil, false,
	)
	if insertError != nil {
		return fmt.Errorf("recording completed migration %d: %w", migration.Version, insertError)
	}
	return nil
}

// updateStatementProgress updates the last_statement column for a migration to reflect
// the most recently completed statement index. This is called after each successful
// statement in non-transactional execution, providing a resumption point if the process
// crashes.
//
// Takes version (int64) which identifies the migration record.
// Takes lastStatement (int) which is the 0-based index of the last successful statement.
//
// Returns error when the progress UPDATE fails. Callers on the success-checkpoint path
// treat the failure as best-effort (the next statement re-flushes), but the failure path
// must propagate it: a stale last_statement would let a crash-and-resume re-execute
// already-applied, possibly non-idempotent statements (M8).
func (executor *Executor) updateStatementProgress(
	ctx context.Context,
	version int64,
	lastStatement int,
) error {
	placeholder := executor.dialectConfig.PlaceholderFunc
	updateSQL := fmt.Sprintf( //nolint:gosec // history table name is a configured identifier under caller control
		"UPDATE %s SET last_statement = %s WHERE version = %s",
		executor.dialectConfig.HistoryTable(),
		placeholder(1),
		placeholder(2),
	)

	if _, execError := executor.queryExecutor().ExecContext(ctx, updateSQL, lastStatement, version); execError != nil {
		return fmt.Errorf("updating statement progress for migration %d: %w", version, execError)
	}
	return nil
}

// flushProgressCheckpoint writes a best-effort progress checkpoint after a successful
// statement batch. A failed checkpoint is logged but not propagated because a later
// statement (or the failure-path flush) re-writes last_statement, so a lost checkpoint
// only costs re-running idempotent statements already covered by the dirty record.
//
// Takes version (int64) which identifies the migration record.
// Takes lastStatement (int) which is the 0-based index of the last successful statement.
func (executor *Executor) flushProgressCheckpoint(
	ctx context.Context,
	version int64,
	lastStatement int,
) {
	if flushError := executor.updateStatementProgress(ctx, version, lastStatement); flushError != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("best-effort statement progress update failed",
			logger_domain.Error(flushError),
			logger_domain.Int64("version", version),
			logger_domain.Int("last_statement", lastStatement),
		)
	}
}

// historyRecordExists reports whether a history row already exists for the given
// migration version.
//
// Used by preRecordDirtyMigration to stay idempotent across retries: a SELECT is
// synchronous on every supported engine (including ClickHouse, whose DELETE/UPDATE are
// async mutations) so it is the safe way to avoid both a primary-key violation on the
// PK-enforcing engines and a duplicate history row on ClickHouse.
//
// Takes version (int64) which identifies the migration record.
//
// Returns bool which is true when a row for version already exists.
// Returns error when the existence query fails.
func (executor *Executor) historyRecordExists(ctx context.Context, version int64) (bool, error) {
	placeholder := executor.dialectConfig.PlaceholderFunc
	selectSQL := fmt.Sprintf( //nolint:gosec // history table name is a configured identifier under caller control
		"SELECT count(*) FROM %s WHERE version = %s",
		executor.dialectConfig.HistoryTable(),
		placeholder(1),
	)

	var count int64
	if scanError := executor.queryExecutor().QueryRowContext(ctx, selectSQL, version).Scan(&count); scanError != nil {
		return false, fmt.Errorf("querying history record for migration %d: %w", version, scanError)
	}
	return count > 0, nil
}

// isDuplicateColumnError reports whether the error indicates the column already exists.
// SQLite does not support IF NOT EXISTS for ADD COLUMN, so this suppresses the expected
// error when the column was already added.
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when the error message contains "duplicate column".
func isDuplicateColumnError(err error) bool {
	lower := strings.ToLower(err.Error())

	return strings.Contains(lower, "duplicate column") ||
		strings.Contains(lower, "already exists")
}
