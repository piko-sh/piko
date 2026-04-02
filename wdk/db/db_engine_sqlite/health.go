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

package db_engine_sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"piko.sh/piko/wdk/db"
)

const initialDiagnosticsCapacity = 3

// CheckHealth returns SQLite-specific diagnostics: database file size,
// freelist page count, and journal mode. Each query handles its own errors
// independently so a single failing diagnostic does not prevent others.
func (*SQLiteEngine) CheckHealth(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	diagnostics := make([]db.DatabaseHealthDiagnostic, 0, initialDiagnosticsCapacity)
	diagnostics = append(diagnostics, checkSQLiteDatabaseSize(ctx, database)...)
	diagnostics = append(diagnostics, checkSQLiteFreelistPages(ctx, database)...)
	diagnostics = append(diagnostics, checkSQLiteJournalMode(ctx, database)...)
	return diagnostics
}

func checkSQLiteDatabaseSize(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var pageCount, pageSize int64
	if err := database.QueryRowContext(ctx, "SELECT page_count, page_size FROM pragma_page_count(), pragma_page_size()").Scan(&pageCount, &pageSize); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "database_size", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "database_size", Value: formatBytes(pageCount * pageSize),
	}}
}

func checkSQLiteFreelistPages(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var freelistCount int64
	if err := database.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&freelistCount); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "freelist_pages", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "freelist_pages", Value: strconv.FormatInt(freelistCount, 10),
	}}
}

func checkSQLiteJournalMode(ctx context.Context, database *sql.DB) []db.DatabaseHealthDiagnostic {
	var journalMode string
	if err := database.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return []db.DatabaseHealthDiagnostic{{
			Name: "journal_mode", State: "UNHEALTHY", Message: fmt.Sprintf("query failed: %v", err),
		}}
	}
	return []db.DatabaseHealthDiagnostic{{
		Name: "journal_mode", Value: journalMode,
	}}
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
