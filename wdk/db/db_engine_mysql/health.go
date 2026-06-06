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

package db_engine_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"piko.sh/piko/wdk/db"
)

const (
	// replicationLagDegradedSeconds is the lag in seconds above which the replica is
	// reported as DEGRADED.
	replicationLagDegradedSeconds = 10

	// replicationLagUnhealthySeconds is the lag in seconds above which the replica is
	// reported as UNHEALTHY.
	replicationLagUnhealthySeconds = 60

	// initialDiagnosticsCapacity is the starting capacity for the diagnostics slice returned
	// by CheckHealth.
	initialDiagnosticsCapacity = 3
)

// CheckHealth returns MySQL-specific health diagnostics.
//
// Collects database size, threads connected, and replication lag. Each query handles its
// own errors independently so a single failing diagnostic does not prevent others.
//
// Takes database (*sql.DB) which is the MySQL connection pool to probe.
//
// Returns []db.DatabaseHealthDiagnostic which is the collected set of diagnostics.
func (*MySQLEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	diagnostics := make([]db.DatabaseHealthDiagnostic, 0, initialDiagnosticsCapacity)
	diagnostics = append(diagnostics, checkMySQLDatabaseSize(ctx, database)...)
	diagnostics = append(diagnostics, checkMySQLThreadsConnected(ctx, database)...)
	diagnostics = append(diagnostics, checkMySQLReplicationLag(ctx, database)...)
	return diagnostics
}

// checkMySQLDatabaseSize queries the size of the current database.
//
// Takes database (*sql.DB) which is the MySQL connection pool.
//
// Returns []db.DatabaseHealthDiagnostic which holds the database size diagnostic.
func checkMySQLDatabaseSize(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var sizeBytes sql.NullFloat64
	err := database.QueryRowContext(ctx,
		"SELECT SUM(data_length + index_length) FROM information_schema.tables WHERE table_schema = DATABASE()",
	).Scan(&sizeBytes)
	if err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "database_size", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	if !sizeBytes.Valid {
		return []db.DatabaseHealthDiagnostic{{
			Name: "database_size", Value: "0 B",
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "database_size", Value: formatBytes(int64(sizeBytes.Float64)),
	}}
}

// checkMySQLThreadsConnected reads the global Threads_connected status.
//
// Takes database (*sql.DB) which is the MySQL connection pool.
//
// Returns []db.DatabaseHealthDiagnostic which holds the threads connected diagnostic.
func checkMySQLThreadsConnected(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var variableName, value string
	err := database.QueryRowContext(ctx, "SHOW GLOBAL STATUS LIKE 'Threads_connected'").Scan(&variableName, &value)
	if err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "threads_connected", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "threads_connected", Value: value,
	}}
}

// checkMySQLReplicationLag probes replica status for seconds behind.
//
// Tries SHOW REPLICA STATUS first, then falls back to SHOW SLAVE STATUS for older MySQL
// versions.
//
// Takes database (*sql.DB) which is the MySQL connection pool.
//
// Returns []db.DatabaseHealthDiagnostic which is nil when no replication is configured, a
// single lag diagnostic when a replica is present, or an UNKNOWN diagnostic when the
// status query genuinely fails.
func checkMySQLReplicationLag(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	lag, found, err := queryReplicationLag(ctx, database, "SHOW REPLICA STATUS")
	if !found {
		lag, found, err = queryReplicationLag(ctx, database, "SHOW SLAVE STATUS")
	}
	if err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name:    "replication_lag",
			State:   "UNKNOWN",
			Message: fmt.Sprintf("replica status query failed: %v", err),
		}}
	}
	if !found {
		return nil
	}

	state := ""
	message := ""
	if lag >= replicationLagUnhealthySeconds {
		state = "UNHEALTHY"
		message = fmt.Sprintf("lag %ds exceeds %ds threshold", lag, replicationLagUnhealthySeconds)
	} else if lag >= replicationLagDegradedSeconds {
		state = "DEGRADED"
		message = fmt.Sprintf("lag %ds exceeds %ds threshold", lag, replicationLagDegradedSeconds)
	}

	return []db.DatabaseHealthDiagnostic{{
		Name:    "replication_lag",
		Value:   fmt.Sprintf("%ds", lag),
		State:   state,
		Message: message,
	}}
}

// queryReplicationLag scans a replica status query for Seconds_Behind.
//
// Tolerates schema differences between MySQL versions by inspecting column names and
// accepting either Seconds_Behind_Source or Seconds_Behind_Master. It distinguishes a
// server that is not a replica (no rows, or no lag column, or a NULL lag) from a genuine
// query, iteration, scan, or parse failure.
//
// Takes database (*sql.DB) which is the MySQL connection pool.
// Takes query (string) which is the SQL to execute.
//
// Returns int64 which is the lag in seconds when found.
// Returns bool which is true when a lag value was extracted.
// Returns error which is non-nil only on a genuine failure (not for a non-replica
// server).
func queryReplicationLag(ctx context.Context, database *sql.DB, query string) (int64, bool, error) {
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		return 0, false, fmt.Errorf("executing %q: %w", query, err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if rowsError := rows.Err(); rowsError != nil {
			return 0, false, fmt.Errorf("iterating %q: %w", query, rowsError)
		}
		return 0, false, nil
	}

	columns, err := rows.Columns()
	if err != nil {
		return 0, false, fmt.Errorf("reading columns for %q: %w", query, err)
	}

	values := make([]sql.NullString, len(columns))
	scanArguments := make([]any, len(columns))
	for i := range values {
		scanArguments[i] = &values[i]
	}

	if err := rows.Scan(scanArguments...); err != nil {
		return 0, false, fmt.Errorf("scanning %q: %w", query, err)
	}

	return scanReplicationLagColumns(columns, values)
}

// scanReplicationLagColumns extracts the Seconds_Behind value from a scanned replica
// status row.
//
// It accepts either Seconds_Behind_Source or Seconds_Behind_Master to tolerate schema
// differences between MySQL versions, and distinguishes a server that is not a replica
// (no lag column, or a NULL lag) from a genuine parse failure.
//
// Takes columns ([]string) which are the column names from the result set.
// Takes values ([]sql.NullString) which are the scanned values aligned with columns.
//
// Returns int64 which is the lag in seconds when found.
// Returns bool which is true when a lag value was extracted.
// Returns error which is non-nil only when the lag value fails to parse.
func scanReplicationLagColumns(columns []string, values []sql.NullString) (int64, bool, error) {
	for i, column := range columns {
		if column != "Seconds_Behind_Source" && column != "Seconds_Behind_Master" {
			continue
		}
		if !values[i].Valid {
			return 0, false, nil
		}
		lag, parseError := strconv.ParseInt(values[i].String, 10, 64)
		if parseError != nil {
			return 0, false, fmt.Errorf("parsing %s value %q: %w", column, values[i].String, parseError)
		}
		return lag, true, nil
	}

	return 0, false, nil
}

// formatBytes renders a byte count in human-readable binary units.
//
// Takes bytes (int64) which is the size to format.
//
// Returns string which is the formatted size with a binary unit suffix.
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
