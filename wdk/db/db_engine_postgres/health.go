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

package db_engine_postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"piko.sh/piko/wdk/db"
)

const (
	// replicationLagDegradedSeconds is the threshold above which replication lag is reported
	// as degraded.
	replicationLagDegradedSeconds = 10.0

	// replicationLagUnhealthySeconds is the threshold above which replication lag is
	// reported as unhealthy.
	replicationLagUnhealthySeconds = 60.0

	// healthStateUnhealthy is the diagnostic state string used when a probe fails or crosses
	// the unhealthy threshold.
	healthStateUnhealthy = "UNHEALTHY"

	// healthQueryFailedFormat is the format string used to describe a failed health probe
	// query.
	healthQueryFailedFormat = "query failed: %v"

	// initialDiagnosticsCapacity sizes the diagnostics slice for the common case where all
	// probes succeed.
	initialDiagnosticsCapacity = 4
)

// CheckHealth returns PostgreSQL-specific health diagnostics.
//
// Each probe handles its own errors so a single failing diagnostic does not prevent
// others. The set covers database size, active connections, recovery state, and
// replication lag.
//
// Takes ctx (context.Context) which bounds each probe query.
// Takes database (*sql.DB) which is the connection pool to probe.
//
// Returns []db.DatabaseHealthDiagnostic which lists every collected probe.
func (*PostgresEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	diagnostics := make([]db.DatabaseHealthDiagnostic, 0, initialDiagnosticsCapacity)
	diagnostics = append(diagnostics, checkPostgresDatabaseSize(ctx, database)...)
	diagnostics = append(diagnostics, checkPostgresActiveConnections(ctx, database)...)
	diagnostics = append(diagnostics, checkPostgresRecoveryState(ctx, database)...)
	diagnostics = append(diagnostics, checkPostgresReplicationLag(ctx, database)...)
	return diagnostics
}

// checkPostgresDatabaseSize probes the current database size in bytes.
//
// Takes ctx (context.Context) which bounds the probe query.
// Takes database (*sql.DB) which is the connection pool to probe.
//
// Returns []db.DatabaseHealthDiagnostic which is a single-entry slice reporting size or
// an unhealthy state.
func checkPostgresDatabaseSize(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var sizeBytes int64
	if err := database.QueryRowContext(ctx, "SELECT pg_database_size(current_database())").Scan(&sizeBytes); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "database_size", State: healthStateUnhealthy, Message: fmt.Sprintf(healthQueryFailedFormat, err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "database_size", Value: formatBytes(sizeBytes),
	}}
}

// checkPostgresActiveConnections probes the count of active sessions.
//
// Takes ctx (context.Context) which bounds the probe query.
// Takes database (*sql.DB) which is the connection pool to probe.
//
// Returns []db.DatabaseHealthDiagnostic which is a single-entry slice reporting the count
// or an unhealthy state.
func checkPostgresActiveConnections(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var count int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'").Scan(&count); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "active_connections", State: healthStateUnhealthy, Message: fmt.Sprintf(healthQueryFailedFormat, err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "active_connections", Value: strconv.Itoa(count),
	}}
}

// checkPostgresRecoveryState probes whether the server is in recovery mode.
//
// Takes ctx (context.Context) which bounds the probe query.
// Takes database (*sql.DB) which is the connection pool to probe.
//
// Returns []db.DatabaseHealthDiagnostic which is a single-entry slice reporting the
// boolean state or an unhealthy state.
func checkPostgresRecoveryState(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var inRecovery bool
	if err := database.QueryRowContext(ctx, "SELECT pg_is_in_recovery()").Scan(&inRecovery); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "is_in_recovery", State: healthStateUnhealthy, Message: fmt.Sprintf(healthQueryFailedFormat, err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "is_in_recovery", Value: strconv.FormatBool(inRecovery),
	}}
}

// checkPostgresReplicationLag probes replication lag when in recovery.
//
// Takes ctx (context.Context) which bounds the probe query.
// Takes database (*sql.DB) which is the connection pool to probe.
//
// Returns []db.DatabaseHealthDiagnostic which is a single-entry slice reporting lag and
// an optional degraded or unhealthy state, or nil when replication is not active.
func checkPostgresReplicationLag(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var lagSeconds sql.NullFloat64

	err := database.QueryRowContext(ctx,
		"SELECT CASE WHEN pg_is_in_recovery() "+
			"THEN extract(epoch FROM (now() - pg_last_xact_replay_timestamp())) "+
			"ELSE NULL END",
	).Scan(&lagSeconds)
	if err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "replication_lag", State: healthStateUnhealthy, Message: fmt.Sprintf(healthQueryFailedFormat, err),
		}}
	}

	if !lagSeconds.Valid {
		return nil
	}

	state := ""
	message := ""
	if lagSeconds.Float64 >= replicationLagUnhealthySeconds {
		state = healthStateUnhealthy
		message = fmt.Sprintf("lag %.1fs exceeds %.0fs threshold", lagSeconds.Float64, replicationLagUnhealthySeconds)
	} else if lagSeconds.Float64 >= replicationLagDegradedSeconds {
		state = "DEGRADED"
		message = fmt.Sprintf("lag %.1fs exceeds %.0fs threshold", lagSeconds.Float64, replicationLagDegradedSeconds)
	}

	return []db.DatabaseHealthDiagnostic{{
		Name:    "replication_lag",
		Value:   fmt.Sprintf("%.1fs", lagSeconds.Float64),
		State:   state,
		Message: message,
	}}
}

// formatBytes renders a byte count using IEC binary unit suffixes.
//
// Takes bytes (int64) which is the count of bytes to format.
//
// Returns string which is the human-readable size, such as "1.5 MiB".
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	divisor, exponent := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(divisor), "KMGTPE"[exponent])
}
