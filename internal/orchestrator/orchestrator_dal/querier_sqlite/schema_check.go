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

package querier_sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
)

const (
	// schemaProbe checks for the columns and tables required by the newest migration.
	schemaProbe = `SELECT t.build_tag, t.is_fatal, t.scheduled_execute_at, t.background, t.owner_instance,
t.last_published_at, t.rerun_id, t.rerun_payload, t.rerun_config, n.instance, n.expires_at
FROM tasks t, nodes n WHERE 1 = 0`
)

// CheckSchema verifies that the orchestrator database has every migration applied.
//
// Takes database (*sql.DB) which is the orchestrator database.
//
// Returns error which wraps orchestrator_dal.ErrSchemaOutdated, naming the remedy, when a
// migration is missing.
func CheckSchema(ctx context.Context, database *sql.DB) error {
	rows, err := database.QueryContext(ctx, schemaProbe)
	if err != nil {
		return fmt.Errorf("%w: %w; apply the orchestrator migrations (wdk/db/db_schema_orchestrator_sqlite.Migrations) "+
			"to this database, or let piko run them by registering them with the database",
			orchestrator_dal.ErrSchemaOutdated, err)
	}
	return rows.Close()
}
