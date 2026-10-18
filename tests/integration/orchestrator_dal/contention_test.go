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

package orchestrator_dal_test

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orchestrator_querier_postgres "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_postgres"
	orchestrator_querier_sqlite "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

const contentionTasks = 60

func runContention(t *testing.T, store orchestrator_domain.TaskStore, workers int) {
	ctx := t.Context()
	ids := make([]string, contentionTasks)
	for i := range ids {
		task := orchestrator_domain.NewTask("contention", map[string]any{"n": float64(i)})
		task.Status = orchestrator_domain.StatusPending
		task.DeduplicationKey = "contention-" + strconv.Itoa(i)
		require.NoError(t, store.CreateTaskWithDedup(ctx, task))
		ids[i] = task.ID
	}

	var (
		executions sync.Map
		failures   []error
		failuresMu sync.Mutex
		done       atomic.Bool
		wg         sync.WaitGroup
	)
	fail := func(err error) {
		failuresMu.Lock()
		failures = append(failures, err)
		failuresMu.Unlock()
	}
	deadline := time.Now().Add(time.Minute)
	for worker := range workers {
		config := orchestrator_domain.DefaultDispatcherConfig()
		config.NodeID = "node-" + strconv.Itoa(worker)
		config.HeartbeatInterval = 0
		core := orchestrator_domain.NewTaskProcessingCore(config, nil, store, nil)
		wg.Go(func() {
			for !done.Load() && time.Now().Before(deadline) {
				page, err := store.DispatchablePage(ctx, orchestrator_domain.DispatchQuery{Now: time.Now(), Limit: 8})
				if err != nil {
					fail(fmt.Errorf("reading page: %w", err))
					return
				}
				if len(page) == 0 {
					done.Store(true)
					return
				}
				rand.Shuffle(len(page), func(i, j int) { page[i], page[j] = page[j], page[i] })
				for _, task := range page {
					err := core.ClaimDelivery(ctx, task)
					if errors.Is(err, orchestrator_domain.ErrObsoleteDelivery) {
						continue
					}
					if err != nil {
						fail(fmt.Errorf("claiming %s: %w", task.ID, err))
						continue
					}
					count, _ := executions.LoadOrStore(task.ID, new(atomic.Int32))
					count.(*atomic.Int32).Add(1)
					time.Sleep(time.Duration(rand.IntN(150)) * time.Millisecond)
					core.HandleTaskSuccess(ctx, task, time.Now())
				}
			}
		})
	}
	wg.Wait()

	require.Empty(t, failures, "losing a claim race is the only acceptable error")
	rows, err := store.GetTasksByID(ctx, ids)
	require.NoError(t, err)
	stranded := 0
	for _, row := range rows {
		if row.Status != orchestrator_domain.StatusComplete {
			stranded++
		}
	}
	assert.Zero(t, stranded, "every task settles")
	for _, id := range ids {
		count, ok := executions.Load(id)
		require.True(t, ok, "task %s never ran", id)
		assert.Equal(t, int32(1), count.(*atomic.Int32).Load(), "task %s ran more than once", id)
	}
}

func TestContention_SQLite(t *testing.T) {
	for _, workers := range []int{3, 10} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			database := openSQLite(t, t.TempDir()+"/orchestrator.db")
			runMigrations(t, database, db_engine_sqlite.SQLite(), orchestrator_querier_sqlite.Migrations)
			runContention(t, orchestrator_querier_sqlite.New(database), workers)
		})
	}
}

func TestContention_Postgres(t *testing.T) {
	if postgresConnectionString == "" {
		t.Skipf("Postgres unavailable: %s", postgresUnavailableReason)
	}
	for _, workers := range []int{3, 10} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			schema := fmt.Sprintf("orch_contention_%d_%d", os.Getpid(), postgresSchemaCounter.Add(1))
			admin, err := sql.Open("pgx", postgresConnectionString)
			require.NoError(t, err)
			_, err = admin.Exec("CREATE SCHEMA " + schema)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
				_ = admin.Close()
			})
			database := openPostgres(t, schema)
			runMigrations(t, database, db_engine_postgres.Postgres(), orchestrator_querier_postgres.Migrations)
			runContention(t, orchestrator_querier_postgres.New(database), workers)
		})
	}
}
