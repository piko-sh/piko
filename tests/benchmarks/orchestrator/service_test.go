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

// This file contains benchmarks for the core orchestrator service logic.
// It focuses on measuring the overhead of the service itself, such as task
// persistence, scheduling, and execution lifecycle, by using a trivial "no-op" executor.
package orchestrator_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator"
	orchestrator_querier_adapter "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_adapter"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/tests/testutil"
	"piko.sh/piko/wdk/json"
	"piko.sh/piko/wdk/safeconv"

	_ "modernc.org/sqlite"
)

var benchCounter atomic.Int64

func openTestDB(b *testing.B, dsn string) *sql.DB {
	b.Helper()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		b.Fatalf("failed to open SQLite database: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(1 * time.Hour)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		b.Fatalf("failed to ping SQLite database: %v", err)
	}

	return db
}

type noopExecutor struct {
	calls *atomic.Int64
}

func (e *noopExecutor) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	e.calls.Add(1)
	return map[string]any{"status": "ok"}, nil
}

func setupServiceBenchmark(b *testing.B) (orchestrator.Service, *noopExecutor) {
	b.Helper()
	dbID := benchCounter.Add(1)
	dsn := fmt.Sprintf(
		"file:memdb_svc_%d?mode=memory&cache=shared&_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=10000",
		dbID,
	)
	return setupServiceWithDSN(b, dsn)
}

func setupServiceBenchmark_OnDisk(b *testing.B) (orchestrator.Service, *noopExecutor) {
	b.Helper()
	dbID := benchCounter.Add(1)
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, fmt.Sprintf("bench_%d.db", dbID))
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=10000",
		dbPath,
	)
	return setupServiceWithDSN(b, dsn)
}

func setupServiceWithDSN(b *testing.B, dsn string) (orchestrator.Service, *noopExecutor) {
	b.Helper()
	dbConn := openTestDB(b, dsn)

	if err := testutil.RunOrchestratorMigrationsOnDB(dbConn); err != nil {
		b.Fatalf("failed to run migrations: %v", err)
	}

	taskStore := orchestrator_querier_adapter.New(dbConn)

	orchestratorConfig := orchestrator.Config{
		TaskStore:          taskStore,
		WorkerCount:        8,
		SchedulerInterval:  1 * time.Minute,
		DispatcherInterval: 20 * time.Millisecond,
	}
	service, err := orchestrator.NewService(context.Background(), orchestratorConfig)
	if err != nil {
		b.Fatalf("failed to create service: %v", err)
	}

	executor := &noopExecutor{calls: &atomic.Int64{}}
	if err := service.RegisterExecutor(context.Background(), "noop", executor); err != nil {
		b.Fatalf("failed to register executor: %v", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	go service.Run(ctx)

	b.Cleanup(func() { _ = dbConn.Close() })
	b.Cleanup(func() { cancel(fmt.Errorf("test: cleanup")) })
	b.Cleanup(func() { service.Stop() })

	return service, executor
}

func BenchmarkService_DispatchAndWait(b *testing.B) {
	service, _ := setupServiceBenchmark(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := orchestrator.NewTask("noop", map[string]any{"data": "payload"})
			receipt, err := service.Dispatch(ctx, task)
			if err != nil {
				b.Fatalf("Dispatch failed: %v", err)
			}
			if err := receipt.Wait(ctx); err != nil {
				b.Fatalf("Receipt wait failed: %v", err)
			}
		}
	})
}

func BenchmarkService_DispatchAndWait_OnDisk(b *testing.B) {
	service, _ := setupServiceBenchmark_OnDisk(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := orchestrator.NewTask("noop", map[string]any{"data": "payload"})
			receipt, err := service.Dispatch(ctx, task)
			if err != nil {
				b.Fatalf("Dispatch failed: %v", err)
			}
			if err := receipt.Wait(ctx); err != nil {
				b.Fatalf("Receipt wait failed: %v", err)
			}
		}
	})
}

func BenchmarkService_Throughput(b *testing.B) {
	service, _ := setupServiceBenchmark_OnDisk(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	var lastReceipt *orchestrator.WorkflowReceipt
	var err error

	for b.Loop() {
		task := orchestrator.NewTask("noop", nil)
		lastReceipt, err = service.Dispatch(ctx, task)
		if err != nil {
			b.Fatalf("Dispatch during loop failed: %v", err)
		}
	}

	if lastReceipt != nil {
		if err := lastReceipt.Wait(ctx); err != nil {
			b.Fatalf("Final receipt wait failed: %v", err)
		}
	}
}

func setupTaskStoreOnly_OnDisk(b *testing.B) (orchestrator_domain.TaskStore, *sql.DB) {
	b.Helper()
	dbID := benchCounter.Add(1)
	tempDir := b.TempDir()
	dbPath := filepath.Join(tempDir, fmt.Sprintf("bench_store_%d.db", dbID))
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=10000",
		dbPath,
	)

	dbConn := openTestDB(b, dsn)
	b.Cleanup(func() { _ = dbConn.Close() })

	if err := testutil.RunOrchestratorMigrationsOnDB(dbConn); err != nil {
		b.Fatalf("failed to run migrations: %v", err)
	}

	taskStore := orchestrator_querier_adapter.New(dbConn)
	return taskStore, dbConn
}

func BenchmarkScheduler_PromoteTasks(b *testing.B) {
	taskStore, dbConn := setupTaskStoreOnly_OnDisk(b)
	ctx := context.Background()
	querier := orchestrator_db.New(dbConn)

	const tasksToPromote = 100

	tasks := make([]*orchestrator_domain.Task, tasksToPromote)
	for i := range tasksToPromote {
		task := orchestrator.NewTask("noop", nil)
		task.Status = "SCHEDULED"
		task.ExecuteAt = time.Now().Add(-1 * time.Minute)
		tasks[i] = task
	}

	b.ResetTimer()
	for b.Loop() {

		b.StopTimer()

		tx, err := dbConn.BeginTx(ctx, nil)
		if err != nil {
			b.Fatalf("Failed to begin tx: %v", err)
		}
		qtx := querier.WithTx(tx)

		for _, task := range tasks {
			payloadBytes, _ := json.Marshal(task.Payload)
			configBytes, _ := json.Marshal(task.Config)
			params := orchestrator_db.CreateTaskParams{
				P1:  task.ID,
				P2:  task.WorkflowID,
				P3:  task.Executor,
				P4:  safeconv.IntToInt32(int(task.Config.Priority)),
				P5:  string(payloadBytes),
				P6:  string(configBytes),
				P7:  string(task.Status),
				P8:  safeconv.Int64ToInt32(task.ExecuteAt.Unix()),
				P9:  safeconv.IntToInt32(task.Attempt),
				P10: safeconv.Int64ToInt32(task.CreatedAt.Unix()),
				P11: safeconv.Int64ToInt32(task.UpdatedAt.Unix()),
			}
			if err := qtx.CreateTask(ctx, params); err != nil {
				_ = tx.Rollback()
				b.Fatalf("Failed to create task in tx: %v", err)
			}
		}

		if err := tx.Commit(); err != nil {
			b.Fatalf("Failed to commit tx: %v", err)
		}

		b.StartTimer()

		promoted, err := taskStore.PromoteScheduledTasks(ctx)
		if err != nil {
			b.Fatalf("PromoteScheduledTasks failed: %v", err)
		}
		if promoted < tasksToPromote {
			b.Logf("Warning: expected at least %d promoted tasks, but got %d", tasksToPromote, promoted)
		}

		b.StopTimer()

		if _, err := dbConn.ExecContext(ctx, "DELETE FROM tasks"); err != nil {
			b.Fatalf("Failed to clean up tasks table: %v", err)
		}
	}
}
