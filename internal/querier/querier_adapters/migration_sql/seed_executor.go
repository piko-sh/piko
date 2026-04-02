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
	"piko.sh/piko/internal/querier/querier_dto"
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

// SeedExecutor implements SeedExecutorPort using database/sql. It handles
// seed history tracking and SQL execution for all supported dialects.
type SeedExecutor struct {
	// database holds the underlying database connection pool.
	database *sql.DB

	// dialectConfig holds the dialect-specific SQL and behaviour.
	dialectConfig DialectConfig
}

var _ querier_domain.SeedExecutorPort = (*SeedExecutor)(nil)

// NewSeedExecutor creates a new SQL-based seed executor.
//
// Takes database (*sql.DB) which is the database connection.
// Takes dialectConfig (DialectConfig) which provides dialect-specific SQL.
//
// Returns *SeedExecutor which is ready to execute seeds.
func NewSeedExecutor(database *sql.DB, dialectConfig DialectConfig) *SeedExecutor {
	return &SeedExecutor{
		database:      database,
		dialectConfig: dialectConfig,
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

// AppliedSeeds returns all seeds that have been applied, ordered by version
// ascending.
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

	return seeds, rows.Err()
}

// ExecuteSeed runs a single seed's SQL content in a transaction and records it
// in the history table.
//
// Takes seed (querier_dto.SeedRecord) which holds the seed SQL and metadata.
//
// Returns error when the seed fails to execute.
func (e *SeedExecutor) ExecuteSeed(ctx context.Context, seed querier_dto.SeedRecord) error {
	start := time.Now()

	tx, err := e.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning seed transaction: %w", err)
	}
	defer tx.Rollback()

	if execErr := e.executeSeedSQL(ctx, tx, seed.Content); execErr != nil {
		return execErr
	}

	durationMs := time.Since(start).Milliseconds()

	insertSQL := fmt.Sprintf( //nolint:gosec // hardcoded table name
		"INSERT INTO piko_seeds (version, name, checksum, duration_ms) VALUES (%s, %s, %s, %s)",
		e.dialectConfig.PlaceholderFunc(seedPlaceholderVersion),
		e.dialectConfig.PlaceholderFunc(seedPlaceholderName),
		e.dialectConfig.PlaceholderFunc(seedPlaceholderChecksum),
		e.dialectConfig.PlaceholderFunc(seedPlaceholderDurationMs),
	)

	if _, insertErr := tx.ExecContext(ctx, insertSQL,
		seed.Version, seed.Name, seed.Checksum, durationMs,
	); insertErr != nil {
		return fmt.Errorf("recording seed %d (%s): %w", seed.Version, seed.Name, insertErr)
	}

	return tx.Commit()
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

// executeSeedSQL executes the seed SQL content against the transaction. When
// the dialect requires statement splitting (MySQL), individual statements are
// executed separately.
//
// Takes tx (*sql.Tx) which is the active database transaction.
// Takes content ([]byte) which holds the raw seed SQL.
//
// Returns error when any statement fails to execute.
func (e *SeedExecutor) executeSeedSQL(ctx context.Context, tx *sql.Tx, content []byte) error {
	if !e.dialectConfig.SplitStatements {
		_, err := tx.ExecContext(ctx, string(content))
		return err
	}

	for _, stmt := range splitStatements(string(content)) {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, trimmed); err != nil {
			return err
		}
	}

	return nil
}
