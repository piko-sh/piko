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
	"math"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

// AppliedVersions returns all applied migrations ordered by version ascending.
//
// Returns []querier_dto.AppliedMigration which holds the applied migration records.
// Returns error when the query or row scanning fails.
func (executor *Executor) AppliedVersions(
	ctx context.Context,
) ([]querier_dto.AppliedMigration, error) {
	rows, queryError := executor.queryExecutor().QueryContext(ctx, executor.appliedVersionsSQL)
	if queryError != nil {
		return nil, fmt.Errorf("querying applied versions: %w", queryError)
	}
	defer rows.Close()

	var applied []querier_dto.AppliedMigration
	for rows.Next() {
		var migration querier_dto.AppliedMigration
		var downChecksum sql.NullString
		var lastStatement sql.NullInt32
		var dirty sql.NullBool
		var appliedAtRaw any
		scanError := rows.Scan(
			&migration.Version,
			&migration.Name,
			&migration.Checksum,
			&appliedAtRaw,
			&migration.DurationMs,
			&downChecksum,
			&lastStatement,
			&dirty,
		)
		if scanError != nil {
			return nil, fmt.Errorf("scanning applied migration: %w", scanError)
		}
		migration.AppliedAt = parseAppliedAt(ctx, appliedAtRaw, migration.Version)
		migration.DownChecksum = downChecksum.String
		if lastStatement.Valid {
			migration.LastStatement = new(int(lastStatement.Int32))
		}
		migration.Dirty = dirty.Valid && dirty.Bool
		applied = append(applied, migration)
	}

	if rowsError := rows.Err(); rowsError != nil {
		return nil, fmt.Errorf("iterating applied migrations: %w", rowsError)
	}

	return applied, nil
}

// parseAppliedAt converts the raw applied_at value from the database into a time.Time,
// handling native time.Time (PostgreSQL), string formats (SQLite), and []byte values
// (MySQL frequently returns applied_at as a byte slice).
//
// On failure (an unparseable string or an unexpected driver type) the function emits a
// debug log via logger_domain.From(ctx, log) so operators can spot configuration drift at
// runtime; the zero time is still returned so the migration record stays usable for
// version/checksum equality checks. The version is included in the log payload so the
// emitted line points back at the offending row.
//
// Takes raw (any) which is the database driver's applied_at value.
// Takes version (int64) which identifies the migration row for diagnostics.
//
// Returns time.Time which is the parsed timestamp, or zero time if parsing fails.
func parseAppliedAt(ctx context.Context, raw any, version int64) time.Time {
	if raw == nil {
		return time.Time{}
	}

	switch v := raw.(type) {
	case time.Time:
		return v
	case []byte:

		return parseAppliedAt(ctx, string(v), version)
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return t
		}
		_, l := logger_domain.From(ctx, log)
		l.Debug("applied_at string did not match any known format",
			logger_domain.String("value", v),
			logger_domain.Int64("version", version),
		)
		return time.Time{}
	case int64:
		return time.Unix(v, 0).UTC()
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v >= math.MaxInt64 || v <= math.MinInt64 {
			_, l := logger_domain.From(ctx, log)
			l.Debug("applied_at float value is out of int64 range",
				logger_domain.String("value", fmt.Sprintf("%v", v)),
				logger_domain.Int64("version", version),
			)
			return time.Time{}
		}
		return time.Unix(int64(v), 0).UTC()
	default:
		_, l := logger_domain.From(ctx, log)
		l.Debug("applied_at value has unexpected driver type",
			logger_domain.String("type", fmt.Sprintf("%T", raw)),
			logger_domain.Int64("version", version),
		)
		return time.Time{}
	}
}

// updateHistory inserts or deletes a migration record in the piko_migrations table
// depending on the direction.
//
// Takes runner (execContextRunner) which executes the write. The transactional engines
// pass the active *sql.Tx; the non-transactional engines (ClickHouse) pass the pinned
// connection / pool directly.
// Takes migration (querier_dto.MigrationRecord) which holds the migration metadata.
// Takes direction (querier_dto.MigrationDirection) which specifies whether this is an up
// or down migration.
// Takes appliedAt (time.Time) which is the timestamp to record.
// Takes durationMs (int64) which is the execution duration in milliseconds.
//
// Returns error when the INSERT or DELETE statement fails.
func (executor *Executor) updateHistory(
	ctx context.Context,
	runner execContextRunner,
	migration querier_dto.MigrationRecord,
	direction querier_dto.MigrationDirection,
	appliedAt time.Time,
	durationMs int64,
) error {
	if direction == querier_dto.MigrationDirectionUp {
		_, insertError := runner.ExecContext(ctx, executor.historyInsertSQL(),
			migration.Version, migration.Name, migration.Checksum, appliedAt.UTC(), durationMs,
			nullableDownChecksum(migration.DownChecksum), nil, false,
		)
		if insertError != nil {
			return fmt.Errorf("inserting migration record: %w", insertError)
		}
		return nil
	}

	_, deleteError := runner.ExecContext(ctx, executor.deleteHistorySQL(), migration.Version)
	if deleteError != nil {
		return fmt.Errorf("deleting migration record: %w", deleteError)
	}
	return nil
}

// deleteHistorySQL builds the statement that removes a single migration history row by
// version, honouring the dialect's DeleteHistorySQLFunc when set (ClickHouse uses an
// ALTER TABLE ... DELETE mutation) and falling back to a standard DELETE otherwise.
//
// Returns string which is the complete delete statement.
func (executor *Executor) deleteHistorySQL() string {
	placeholder := executor.dialectConfig.PlaceholderFunc
	tableName := executor.dialectConfig.HistoryTable()
	if executor.dialectConfig.DeleteHistorySQLFunc != nil {
		return executor.dialectConfig.DeleteHistorySQLFunc(tableName, placeholder(1))
	}
	return fmt.Sprintf( //nolint:gosec // history table name is a configured identifier under caller control
		"DELETE FROM %s WHERE version = %s",
		tableName,
		placeholder(1),
	)
}

// clearDirty marks a non-transactional migration as successfully completed by setting
// dirty = FALSE and recording the final duration.
//
// Takes version (int64) which identifies the migration record.
// Takes start (time.Time) which is when execution began, used to compute the final
// duration.
//
// Returns error when the UPDATE statement fails.
func (executor *Executor) clearDirty(
	ctx context.Context,
	version int64,
	start time.Time,
) error {
	placeholder := executor.dialectConfig.PlaceholderFunc
	durationMs := executor.clock.Now().Sub(start).Milliseconds()
	updateSQL := fmt.Sprintf( //nolint:gosec // history table name is a configured identifier under caller control
		"UPDATE %s SET dirty = %s, duration_ms = %s WHERE version = %s",
		executor.dialectConfig.HistoryTable(),
		placeholder(clearDirtyPlaceholderDirty),
		placeholder(clearDirtyPlaceholderDurationMs),
		placeholder(clearDirtyPlaceholderVersion),
	)

	_, updateError := executor.queryExecutor().ExecContext(ctx, updateSQL, false, durationMs, version)
	if updateError != nil {
		return fmt.Errorf("clearing dirty flag for migration %d: %w", version, updateError)
	}

	return nil
}
