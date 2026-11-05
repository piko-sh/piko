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

package querier_postgres

import (
	"context"
	"database/sql"
	"fmt"

	"piko.sh/piko/internal/worker/worker_domain"
)

const (
	// schemaProbe checks for the columns and tables required by the newest migration.
	schemaProbe = `SELECT j.unique_key, j.correlation_id, j.timeout_seconds, v.claimed_by_worker_id, v.recorded_at,
i.deleted_at, i.claimed_at FROM worker_jobs j, worker_job_versions v, worker_job_index i WHERE 1 = 0`
)

// CheckSchema verifies that the worker database has every migration applied.
//
// Takes database (*sql.DB) which is the worker database.
//
// Returns error which wraps worker_domain.ErrSchemaOutdated, naming the remedy, when a
// migration is missing.
func CheckSchema(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, schemaProbe)
	if err != nil {
		return fmt.Errorf("%w: %w; apply the worker migrations (wdk/db/db_schema_worker_postgres.Migrations) "+
			"to this database, or let piko run them by registering them with the database",
			worker_domain.ErrSchemaOutdated, err)
	}
	return rows.Close()
}
