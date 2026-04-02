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
	replicationLagDegradedSeconds = 10

	replicationLagUnhealthySeconds = 60

	initialDiagnosticsCapacity = 3
)

// CheckHealth returns MySQL-specific diagnostics: database size, threads
// connected, and replication lag. Each query handles its own errors
// independently so a single failing diagnostic does not prevent others.
func (*MySQLEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	diagnostics := make([]db.DatabaseHealthDiagnostic, 0, initialDiagnosticsCapacity)
	diagnostics = append(diagnostics, checkMySQLDatabaseSize(ctx, database)...)
	diagnostics = append(diagnostics, checkMySQLThreadsConnected(ctx, database)...)
	diagnostics = append(diagnostics, checkMySQLReplicationLag(ctx, database)...)
	return diagnostics
}

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

func checkMySQLReplicationLag(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	lag, found := queryReplicationLag(ctx, database, "SHOW REPLICA STATUS")
	if !found {
		lag, found = queryReplicationLag(ctx, database, "SHOW SLAVE STATUS")
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

func queryReplicationLag(ctx context.Context, database *sql.DB, query string) (int64, bool) {
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		return 0, false
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return 0, false
	}

	columns, err := rows.Columns()
	if err != nil {
		return 0, false
	}

	values := make([]sql.NullString, len(columns))
	scanArguments := make([]any, len(columns))
	for i := range values {
		scanArguments[i] = &values[i]
	}

	if err := rows.Scan(scanArguments...); err != nil {
		return 0, false
	}

	for i, column := range columns {
		if column == "Seconds_Behind_Source" || column == "Seconds_Behind_Master" {
			if !values[i].Valid {
				return 0, false
			}
			lag, parseError := strconv.ParseInt(values[i].String, 10, 64)
			if parseError != nil {
				return 0, false
			}
			return lag, true
		}
	}

	return 0, false
}

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
