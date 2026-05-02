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

//go:build integration

package querier_clickhouse_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_dto"
)

func TestClickHouseMigrationExecutorLifecycle(t *testing.T) {
	databaseName := createIsolatedDatabase(t)
	defer dropDatabase(t, databaseName)

	db, err := sql.Open("clickhouse", dsnForDatabase(databaseName))
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	executor := migration_sql.NewExecutor(db, migration_sql.ClickHouseDialect())

	require.NoError(t, executor.EnsureMigrationTable(ctx),
		"ReplacingMergeTree history DDL must be accepted by ClickHouse")

	require.NoError(t, executor.AcquireLock(ctx))
	defer func() { require.NoError(t, executor.ReleaseLock(ctx)) }()

	up := querier_dto.MigrationRecord{
		Version:  1,
		Name:     "create_events",
		Checksum: "checksum-1",
		Content: []byte(
			"CREATE TABLE events (id Int64, label String) ENGINE = MergeTree() ORDER BY id;\n" +
				"INSERT INTO events (id, label) VALUES (1, 'first');",
		),
		SkipUpTo: -1,
	}

	require.NoError(t, executor.ExecuteMigration(ctx, up, querier_dto.MigrationDirectionUp, true))

	applied, err := executor.AppliedVersions(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, int64(1), applied[0].Version)
	assert.Equal(t, "create_events", applied[0].Name)
	assert.Equal(t, "checksum-1", applied[0].Checksum)
	assert.False(t, applied[0].Dirty, "a completed migration must be recorded with dirty=false")

	var rowCount uint64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT count() FROM events").Scan(&rowCount))
	assert.Equal(t, uint64(1), rowCount)

	require.NoError(t, executor.ExecuteMigration(ctx, querier_dto.MigrationRecord{
		Version: 1, Name: "create_events", Checksum: "checksum-1", SkipUpTo: -1,
		Content: []byte("SELECT 1;"),
	}, querier_dto.MigrationDirectionUp, true))

	applied, err = executor.AppliedVersions(ctx)
	require.NoError(t, err)
	require.Len(t, applied, 1, "ReplacingMergeTree + FINAL must dedupe duplicate version rows")

	down := querier_dto.MigrationRecord{
		Version: 1, Name: "create_events", SkipUpTo: -1,
		Content: []byte("DROP TABLE events;"),
	}
	require.NoError(t, executor.ExecuteMigration(ctx, down, querier_dto.MigrationDirectionDown, true))

	require.Eventually(t, func() bool {
		rows, versionsErr := executor.AppliedVersions(ctx)
		return versionsErr == nil && len(rows) == 0
	}, 30*time.Second, 250*time.Millisecond,
		"down-migration ALTER TABLE ... DELETE should eventually remove the history row")
}

func TestClickHouseMigrationConcurrentMigrators(t *testing.T) {
	databaseName := createIsolatedDatabase(t)
	defer dropDatabase(t, databaseName)

	db, err := sql.Open("clickhouse", dsnForDatabase(databaseName))
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	require.NoError(t, migration_sql.NewExecutor(db, migration_sql.ClickHouseDialect()).EnsureMigrationTable(ctx))

	const migrators = 2
	var waitGroup sync.WaitGroup
	errs := make([]error, migrators)
	for i := range migrators {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			executor := migration_sql.NewExecutor(db, migration_sql.ClickHouseDialect())
			errs[index] = executor.ExecuteMigration(ctx, querier_dto.MigrationRecord{
				Version: 7, Name: "concurrent", Checksum: "checksum-7", SkipUpTo: -1,
				Content: []byte("CREATE TABLE IF NOT EXISTS widgets (id Int64) ENGINE = MergeTree() ORDER BY id;"),
			}, querier_dto.MigrationDirectionUp, true)
		}(i)
	}
	waitGroup.Wait()

	for index, migratorErr := range errs {
		require.NoErrorf(t, migratorErr, "concurrent migrator %d", index)
	}

	require.Eventually(t, func() bool {
		applied, versionsErr := migration_sql.NewExecutor(db, migration_sql.ClickHouseDialect()).AppliedVersions(ctx)
		if versionsErr != nil {
			return false
		}
		count := 0
		for _, row := range applied {
			if row.Version == 7 {
				count++
			}
		}
		return count == 1
	}, 30*time.Second, 250*time.Millisecond,
		"FINAL read must converge to a single history row for the concurrently-applied version")
}
