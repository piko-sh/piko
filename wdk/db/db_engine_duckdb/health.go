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

package db_engine_duckdb

import (
	"context"
	"database/sql"
	"fmt"

	"piko.sh/piko/wdk/db"
)

const (
	// initialDiagnosticsCapacity sizes the pre-allocated diagnostics slice to the number of
	// probes CheckHealth runs.
	initialDiagnosticsCapacity = 3
)

// CheckHealth runs DuckDB-specific diagnostic probes.
//
// Each probe handles its own errors so a single failure does not block the others.
// Reported probes cover database size, memory limit, and thread count.
//
// Takes ctx (context.Context) which carries cancellation and deadlines for the underlying
// queries.
// Takes database (*sql.DB) which is the DuckDB connection to probe.
//
// Returns []db.DatabaseHealthDiagnostic which is one entry per probe, each marked
// UNHEALTHY when its query failed.
func (*DuckDBEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	diagnostics := make([]db.DatabaseHealthDiagnostic, 0, initialDiagnosticsCapacity)
	diagnostics = append(diagnostics, checkDuckDBDatabaseSize(ctx, database)...)
	diagnostics = append(diagnostics, checkDuckDBMemoryLimit(ctx, database)...)
	diagnostics = append(diagnostics, checkDuckDBThreads(ctx, database)...)
	return diagnostics
}

// checkDuckDBDatabaseSize probes the on-disk database size.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes database (*sql.DB) which is the DuckDB connection.
//
// Returns []db.DatabaseHealthDiagnostic which is a one-entry slice carrying the size, or
// an UNHEALTHY diagnostic on query failure.
func checkDuckDBDatabaseSize(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var databaseSize string
	err := database.QueryRowContext(ctx,
		"SELECT database_size FROM duckdb_databases() WHERE database_name = current_database()",
	).Scan(&databaseSize)
	if err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "database_size", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "database_size", Value: databaseSize,
	}}
}

// checkDuckDBMemoryLimit probes the configured memory limit setting.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes database (*sql.DB) which is the DuckDB connection.
//
// Returns []db.DatabaseHealthDiagnostic which is a one-entry slice carrying the limit, or
// an UNHEALTHY diagnostic on query failure.
func checkDuckDBMemoryLimit(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var memoryLimit string
	if err := database.QueryRowContext(ctx, "SELECT current_setting('memory_limit')").Scan(&memoryLimit); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "memory_limit", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "memory_limit", Value: memoryLimit,
	}}
}

// checkDuckDBThreads probes the configured thread count setting.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes database (*sql.DB) which is the DuckDB connection.
//
// Returns []db.DatabaseHealthDiagnostic which is a one-entry slice carrying the thread
// count, or an UNHEALTHY diagnostic on query failure.
func checkDuckDBThreads(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var threads string
	if err := database.QueryRowContext(ctx, "SELECT current_setting('threads')").Scan(&threads); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "threads", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "threads", Value: threads,
	}}
}
