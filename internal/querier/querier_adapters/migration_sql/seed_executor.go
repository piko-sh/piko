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
	// seedPlaceholderVersion holds the 1-based index for the version column.
	seedPlaceholderVersion = 1

	// seedPlaceholderName holds the 1-based index for the name column.
	seedPlaceholderName = 2

	// seedPlaceholderChecksum holds the 1-based index for the checksum column.
	seedPlaceholderChecksum = 3

	// seedPlaceholderDurationMs holds the 1-based index for the duration_ms column.
	seedPlaceholderDurationMs = 4
)

// SeedExecutor implements SeedExecutorPort using database/sql. It handles seed history
// tracking and SQL execution for all supported dialects.
type SeedExecutor struct {
	// database holds the underlying database connection pool.
	database *sql.DB

	// pinnedSeedLockConnection holds the dedicated connection that owns the seed advisory
	// lock when one has been acquired via AcquireSeedLock. Nil when no lock is currently
	// held.
	pinnedSeedLockConnection *sql.Conn

	// clock is the time source used to stamp seed durations; never nil after
	// NewSeedExecutor, which defaults it to clock.RealClock().
	clock clock.Clock

	// dialectConfig holds the dialect-specific SQL and behaviour.
	dialectConfig DialectConfig
}

var (
	_ querier_domain.SeedExecutorPort = (*SeedExecutor)(nil)

	// errSeedLockAlreadyHeld is returned by AcquireSeedLock when a seed-lock connection is
	// already pinned.
	//
	// Acquiring again without releasing would orphan the previously pinned connection and
	// its advisory lock, so the second acquire is refused rather than leaking it. This
	// mirrors errLockAlreadyHeld on the migration Executor.
	errSeedLockAlreadyHeld = errors.New("seed lock already held; release it before acquiring again")
)

// NewSeedExecutor creates a new SQL-based seed executor.
//
// Takes database (*sql.DB) which is the database connection.
// Takes dialectConfig (DialectConfig) which provides dialect-specific SQL.
// Takes options (...SeedExecutorOption) which customise the executor, for example
// WithSeedExecutorClock to inject a deterministic clock in tests.
//
// Returns *SeedExecutor which is ready to execute seeds.
func NewSeedExecutor(database *sql.DB, dialectConfig DialectConfig, options ...SeedExecutorOption) *SeedExecutor {
	executor := &SeedExecutor{
		database:      database,
		dialectConfig: dialectConfig,
		clock:         clock.RealClock(),
	}
	for _, option := range options {
		option(executor)
	}
	return executor
}

// SeedExecutorOption customises a SeedExecutor at construction time.
type SeedExecutorOption func(*SeedExecutor)

// WithSeedExecutorClock overrides the time source used to record seed durations.
//
// The default is clock.RealClock(), and tests inject a mock clock to assert recorded
// durations deterministically. A nil clock is ignored so the default is preserved.
//
// Takes source (clock.Clock) which is the time source to use, ignored when nil.
//
// Returns SeedExecutorOption which applies the clock during construction.
func WithSeedExecutorClock(source clock.Clock) SeedExecutorOption {
	return func(executor *SeedExecutor) {
		if source != nil {
			executor.clock = source
		}
	}
}

// EnsureSeedTable creates the piko_seeds table if it does not exist.
//
// Returns error when the table cannot be created.
func (e *SeedExecutor) EnsureSeedTable(ctx context.Context) error {
	if e.dialectConfig.CreateSeedTableSQL == "" {
		return errors.New("no seed table DDL configured for this dialect")
	}
	_, err := e.database.ExecContext(ctx, e.dialectConfig.CreateSeedTableSQL)
	if err != nil {
		return fmt.Errorf("creating piko_seeds table: %w", err)
	}
	return nil
}

// AppliedSeeds returns all seeds that have been applied, ordered by version ascending.
//
// Returns []querier_dto.AppliedSeed which lists all applied seeds.
// Returns error when the history cannot be read.
func (e *SeedExecutor) AppliedSeeds(ctx context.Context) ([]querier_dto.AppliedSeed, error) {
	rows, err := e.database.QueryContext(ctx,
		"SELECT version, name, checksum, applied_at, duration_ms FROM piko_seeds ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("querying applied seeds: %w", err)
	}
	defer rows.Close()

	var seeds []querier_dto.AppliedSeed
	for rows.Next() {
		var s querier_dto.AppliedSeed
		if scanErr := rows.Scan(&s.Version, &s.Name, &s.Checksum, &s.AppliedAt, &s.DurationMs); scanErr != nil {
			return nil, fmt.Errorf("scanning applied seed: %w", scanErr)
		}
		seeds = append(seeds, s)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterating applied seeds: %w", rowsErr)
	}
	return seeds, nil
}

// ExecuteSeed runs a single seed's SQL content and records it in the history table.
//
// The INSERT into piko_seeds is rendered through the dialect's InsertSeedSQLFunc, which
// yields an idempotent statement (e.g. "ON CONFLICT (version) DO NOTHING" on
// PostgreSQL/SQLite, "INSERT IGNORE" on MySQL). This keeps concurrent seed runs across
// multiple replicas safe even if both resolve the same seed as pending: the second
// writer's record insert becomes a no-op rather than a primary-key violation.
//
// On engines that do not support transactions (DialectConfig.DisableTransactions, such as
// ClickHouse), the seed SQL and the idempotent record insert run directly on the pool
// rather than inside a BeginTx that the engine would reject; the idempotent INSERT still
// provides the concurrency safety a transaction would otherwise give.
//
// Takes seed (querier_dto.SeedRecord) which holds the seed SQL and metadata.
//
// Returns error when the seed fails to execute.
func (e *SeedExecutor) ExecuteSeed(ctx context.Context, seed querier_dto.SeedRecord) error {
	start := e.clock.Now()

	if e.dialectConfig.DisableTransactions {
		return e.applySeed(ctx, e.database, seed, start)
	}

	tx, err := e.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning seed transaction: %w", err)
	}
	defer rollbackSeedTransactionIfActive(ctx, tx)

	if applyErr := e.applySeed(ctx, tx, seed, start); applyErr != nil {
		return applyErr
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("committing seed %d (%s): %w", seed.Version, seed.Name, commitErr)
	}
	return nil
}

// AcquireSeedLock acquires the dialect-specific advisory lock for seed runs. The lock
// uses a key distinct from the migration lock so seed and migration runs serialise
// independently rather than starving each other.
//
// When the dialect has no SeedLockStrategy configured, this falls back to a no-op lock.
// That is correct for single-replica deployments (and SQLite, where file-level locking
// suffices) but unsafe for multi-replica setups using a dialect that has not been
// updated; callers in such configurations should ensure SeedLockStrategy is set.
//
// Returns error when the lock cannot be acquired.
func (e *SeedExecutor) AcquireSeedLock(ctx context.Context) error {
	if e.pinnedSeedLockConnection != nil {
		return errSeedLockAlreadyHeld
	}
	strategy := e.dialectConfig.SeedLockStrategy
	if strategy == nil {
		strategy = &NoOpLock{}
	}
	connection, lockError := strategy.Acquire(ctx, e.database)
	if lockError != nil {
		return fmt.Errorf("acquiring seed lock: %w", lockError)
	}
	e.pinnedSeedLockConnection = connection
	return nil
}

// ReleaseSeedLock releases the dialect-specific advisory lock previously acquired by
// AcquireSeedLock. Safe to call when no lock is held.
//
// Returns error when the lock cannot be released.
func (e *SeedExecutor) ReleaseSeedLock(ctx context.Context) error {
	strategy := e.dialectConfig.SeedLockStrategy
	if strategy == nil {
		strategy = &NoOpLock{}
	}
	connection := e.pinnedSeedLockConnection
	e.pinnedSeedLockConnection = nil
	if releaseError := strategy.Release(ctx, connection); releaseError != nil {
		return fmt.Errorf("releasing seed lock: %w", releaseError)
	}
	return nil
}

// ClearSeedHistory removes all records from the piko_seeds table.
//
// Returns error when the history cannot be cleared.
func (e *SeedExecutor) ClearSeedHistory(ctx context.Context) error {
	_, err := e.database.ExecContext(ctx, "DELETE FROM piko_seeds")
	if err != nil {
		return fmt.Errorf("clearing seed history: %w", err)
	}
	return nil
}

// applySeed runs the seed SQL and records its history row on the given runner, which is
// the transaction on transactional engines or the pool on non-transactional ones.
//
// Takes runner (execContextRunner) which executes the SQL.
// Takes seed (querier_dto.SeedRecord) which holds the seed SQL and metadata.
// Takes start (time.Time) which marks when execution began, for the recorded duration.
//
// Returns error when the seed SQL or the history insert fails.
func (e *SeedExecutor) applySeed(ctx context.Context, runner execContextRunner, seed querier_dto.SeedRecord, start time.Time) error {
	if execErr := e.executeSeedSQL(ctx, runner, seed.Content); execErr != nil {
		return execErr
	}

	durationMs := e.clock.Now().Sub(start).Milliseconds()

	insertSQL, insertSQLErr := e.renderSeedInsertSQL()
	if insertSQLErr != nil {
		return insertSQLErr
	}

	if _, insertErr := runner.ExecContext(ctx, insertSQL,
		seed.Version, seed.Name, seed.Checksum, durationMs,
	); insertErr != nil {
		return fmt.Errorf("recording seed %d (%s): %w", seed.Version, seed.Name, insertErr)
	}

	return nil
}

// renderSeedInsertSQL builds the dialect-specific idempotent INSERT statement for the
// piko_seeds history table. Falls back to a plain INSERT only when the dialect has not
// configured InsertSeedSQLFunc, which is reserved for legacy callers; modern dialect
// builders always populate the field.
//
// Returns string which is the rendered INSERT statement.
// Returns error when the dialect lacks both the new function and the placeholder helper.
func (e *SeedExecutor) renderSeedInsertSQL() (string, error) {
	if e.dialectConfig.PlaceholderFunc == nil {
		return "", errors.New("seed executor missing PlaceholderFunc")
	}
	versionPlaceholder := e.dialectConfig.PlaceholderFunc(seedPlaceholderVersion)
	namePlaceholder := e.dialectConfig.PlaceholderFunc(seedPlaceholderName)
	checksumPlaceholder := e.dialectConfig.PlaceholderFunc(seedPlaceholderChecksum)
	durationPlaceholder := e.dialectConfig.PlaceholderFunc(seedPlaceholderDurationMs)
	if e.dialectConfig.InsertSeedSQLFunc != nil {
		return e.dialectConfig.InsertSeedSQLFunc(
			versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder,
		), nil
	}
	return fmt.Sprintf( //nolint:gosec // hardcoded table name
		"INSERT INTO piko_seeds (version, name, checksum, duration_ms) VALUES (%s, %s, %s, %s)",
		versionPlaceholder, namePlaceholder, checksumPlaceholder, durationPlaceholder,
	), nil
}

// rollbackSeedTransactionIfActive rolls the seed transaction back during deferred
// cleanup. The sql.ErrTxDone case is the harmless rollback-after-commit path; every other
// error is logged so operators can spot rollback failures.
//
// Takes tx (*sql.Tx) which is the seed transaction being released.
func rollbackSeedTransactionIfActive(ctx context.Context, tx *sql.Tx) {
	rollbackError := tx.Rollback()
	if rollbackError == nil || errors.Is(rollbackError, sql.ErrTxDone) {
		return
	}
	_, l := logger_domain.From(ctx, log)
	l.Warn("seed transaction rollback failed",
		logger_domain.Error(rollbackError),
	)
}

// executeSeedSQL executes the seed SQL content against the given runner.
//
// When the dialect requires statement splitting (MySQL), individual statements are
// executed separately.
//
// Takes runner (execContextRunner) which executes the SQL.
// Takes content ([]byte) which holds the raw seed SQL.
//
// Returns error when any statement fails to execute.
func (e *SeedExecutor) executeSeedSQL(ctx context.Context, runner execContextRunner, content []byte) error {
	if !e.dialectConfig.SplitStatements {
		if _, err := runner.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("executing seed SQL: %w", err)
		}
		return nil
	}

	statements, splitError := splitStatementsWithOptions(string(content), e.dialectConfig.BackslashEscapes)
	if splitError != nil {
		return fmt.Errorf("splitting seed statements: %w", splitError)
	}

	for index, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if cancelErr := ctx.Err(); cancelErr != nil {
			return fmt.Errorf("seed cancelled before statement %d: %w", index+1, cancelErr)
		}
		if _, err := runner.ExecContext(ctx, trimmed); err != nil {
			return fmt.Errorf("executing seed statement %d/%d: %w", index+1, len(statements), err)
		}
	}

	return nil
}
