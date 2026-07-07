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

package persistence_sqlite_test

import (
	"context"
	"database/sql"
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	triggerMigrations embed.FS
)

func TestMigrationServiceUp_HandlesSQLiteTriggerBodies(t *testing.T) {
	t.Parallel()

	database := openTestDB(t, "trigger_migration.db")
	runTriggerMigrations(t, database)

	ctx := context.Background()

	_, err := database.ExecContext(ctx,
		`INSERT INTO accounts (id, email) VALUES (?, ?)`,
		"acc-1", "test@example.com",
	)
	require.NoError(t, err, "insert into accounts")

	var logCount int
	require.NoError(t,
		database.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM account_log WHERE account_id = ?`, "acc-1",
		).Scan(&logCount),
		"count account_log rows",
	)
	require.Equal(t, 2, logCount,
		"the AFTER INSERT trigger body has two INSERT statements; both must have fired")

	_, err = database.ExecContext(ctx,
		`UPDATE accounts SET email = ? WHERE id = ?`,
		"new@example.com", "acc-1",
	)
	require.Error(t, err, "UPDATE should be rejected by the trigger")
	require.Contains(t, err.Error(), "Cannot UPDATE on accounts",
		"trigger body's RAISE message must surface")
}

func TestMigrationServiceUp_TriggerMigrationIsIdempotent(t *testing.T) {
	t.Parallel()

	database := openTestDB(t, "trigger_idempotent.db")
	runTriggerMigrations(t, database)
	runTriggerMigrations(t, database)

	ctx := context.Background()
	_, err := database.ExecContext(ctx,
		`INSERT INTO accounts (id, email) VALUES (?, ?)`,
		"acc-2", "again@example.com",
	)
	require.NoError(t, err)
}

func runTriggerMigrations(t *testing.T, database *sql.DB) {
	t.Helper()

	runMigrationsWithDir(t, database, triggerMigrations, "testdata/trigger_migrations")
}
