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

const initialDiagnosticsCapacity = 3

// CheckHealth returns DuckDB-specific diagnostics: database size, memory
// limit, and thread count. Each query handles its own errors independently
// so a single failing diagnostic does not prevent others.
func (*DuckDBEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	diagnostics := make([]db.DatabaseHealthDiagnostic, 0, initialDiagnosticsCapacity)
	diagnostics = append(diagnostics, checkDuckDBDatabaseSize(ctx, database)...)
	diagnostics = append(diagnostics, checkDuckDBMemoryLimit(ctx, database)...)
	diagnostics = append(diagnostics, checkDuckDBThreads(ctx, database)...)
	return diagnostics
}

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
