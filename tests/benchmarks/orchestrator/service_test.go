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

// This file contains benchmarks for the core orchestrator service logic on SQLite.
// It focuses on measuring the overhead of the service itself, such as task
// persistence, scheduling, and execution lifecycle, by using a trivial "no-op" executor.
package orchestrator_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	orchestrator_querier_sqlite "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/storebench"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/tests/testutil"
	"piko.sh/piko/wdk/events/events_provider_gochannel"
	"piko.sh/piko/wdk/json"
	"piko.sh/safeconv"

	_ "modernc.org/sqlite"
)

const (
	benchDirEnv = "PIKO_BENCH_WAL_DIR"
)

var (
	benchCounter atomic.Int64
)

func openTestDB(tb testing.TB, dsn string) *sql.DB {
	tb.Helper()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		tb.Fatalf("failed to open SQLite database: %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(1 * time.Hour)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		tb.Fatalf("failed to ping SQLite database: %v", err)
	}

	return db
}

func onDiskDSN(tb testing.TB, name string) (string, func()) {
	tb.Helper()
	dir, err := os.MkdirTemp(os.Getenv(benchDirEnv), "orchestrator-sqlite-*")
	if err != nil {
		tb.Fatalf("creating database directory: %v", err)
	}
	dsn := fmt.Sprintf(
		"file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=10000",
		filepath.Join(dir, name),
	)
	return dsn, func() { _ = os.RemoveAll(dir) }
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
	dsn, cleanup := onDiskDSN(b, fmt.Sprintf("bench_%d.db", benchCounter.Add(1)))
	b.Cleanup(cleanup)
	return setupServiceWithDSN(b, dsn)
}

func newEventBus(b *testing.B) orchestrator_domain.EventBus {
	b.Helper()
	provider, err := events_provider_gochannel.NewGoChannelProvider(events_provider_gochannel.DefaultConfig())
	if err != nil {
		b.Fatalf("failed to create events provider: %v", err)
	}
	if err := provider.Start(context.Background()); err != nil {
		b.Fatalf("failed to start events provider: %v", err)
	}
	bus := orchestrator_adapters.NewWatermillEventBus(provider.Publisher(), provider.Subscriber(), provider.Router())
	b.Cleanup(func() {
		_ = bus.Close(context.Background())
		_ = provider.Close()
	})
	return bus
}

func setupServiceWithDSN(b *testing.B, dsn string) (orchestrator.Service, *noopExecutor) {
	b.Helper()
	dbConn := openTestDB(b, dsn)
	b.Cleanup(func() { _ = dbConn.Close() })

	if err := testutil.RunOrchestratorMigrationsOnDB(dbConn); err != nil {
		b.Fatalf("failed to run migrations: %v", err)
	}

	eventBus := newEventBus(b)
	service, err := orchestrator.NewService(context.Background(), orchestrator.Config{
		TaskStore: orchestrator_querier_sqlite.New(dbConn),
		EventBus:  eventBus,
	})
	if err != nil {
		b.Fatalf("failed to create service: %v", err)
	}

	executor := &noopExecutor{calls: &atomic.Int64{}}
	if err := service.RegisterExecutor(context.Background(), "noop", executor); err != nil {
		b.Fatalf("failed to register executor: %v", err)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	running := make(chan struct{})
	go func() {
		defer close(running)
		service.Run(ctx)
	}()

	b.Cleanup(func() {
		cancel(errors.New("benchmark finished"))
		service.Stop()
		<-running
	})

	return service, executor
}

func dispatchAndWaitParallel(b *testing.B, service orchestrator.Service) {
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			task := orchestrator.NewTask("noop", map[string]any{"data": "payload"})
			receipt, err := service.Dispatch(ctx, task)
			if err != nil {
				b.Errorf("Dispatch failed: %v", err)
				return
			}
			if err := receipt.Wait(ctx); err != nil {
				b.Errorf("Receipt wait failed: %v", err)
				return
			}
		}
	})
}

func BenchmarkService_DispatchAndWait(b *testing.B) {
	service, _ := setupServiceBenchmark(b)
	dispatchAndWaitParallel(b, service)
}

func BenchmarkService_DispatchAndWait_OnDisk(b *testing.B) {
	service, _ := setupServiceBenchmark_OnDisk(b)
	dispatchAndWaitParallel(b, service)
}

func BenchmarkService_Throughput(b *testing.B) {
	service, _ := setupServiceBenchmark_OnDisk(b)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	var (
		lastReceipt *orchestrator.WorkflowReceipt
		rejected    int
	)
	for b.Loop() {
		task := orchestrator.NewTask("noop", nil)
		receipt, err := service.Dispatch(ctx, task)
		if err != nil {
			rejected++
			continue
		}
		lastReceipt = receipt
	}

	if lastReceipt != nil {
		if err := lastReceipt.Wait(ctx); err != nil {
			b.Fatalf("Final receipt wait failed: %v", err)
		}
	}
	b.ReportMetric(float64(rejected)/float64(b.N), "rejected/op")
}

func setupTaskStoreOnly_OnDisk(tb testing.TB) (orchestrator_domain.TaskStore, *sql.DB, func()) {
	tb.Helper()
	dsn, removeDir := onDiskDSN(tb, fmt.Sprintf("bench_store_%d.db", benchCounter.Add(1)))
	dbConn := openTestDB(tb, dsn)
	closeAll := func() {
		_ = dbConn.Close()
		removeDir()
	}

	if err := testutil.RunOrchestratorMigrationsOnDB(dbConn); err != nil {
		closeAll()
		tb.Fatalf("failed to run migrations: %v", err)
	}

	return orchestrator_querier_sqlite.New(dbConn), dbConn, closeAll
}

func BenchmarkStore(b *testing.B) {
	b.Run("store=sqlite", func(b *testing.B) {
		storebench.Run(b, func(tb testing.TB) (orchestrator_domain.TaskStore, func()) {
			store, _, closeAll := setupTaskStoreOnly_OnDisk(tb)
			return store, closeAll
		})
	})
}

func BenchmarkScheduler_PromoteTasks(b *testing.B) {
	taskStore, dbConn, closeAll := setupTaskStoreOnly_OnDisk(b)
	b.Cleanup(closeAll)
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
				ID:         task.ID,
				WorkflowID: task.WorkflowID,
				Executor:   task.Executor,
				Priority:   safeconv.IntToInt32(int(task.Config.Priority)),
				Payload:    string(payloadBytes),
				Config:     string(configBytes),
				Status:     string(task.Status),
				ExecuteAt:  task.ExecuteAt.Unix(),
				Attempt:    safeconv.IntToInt32(task.Attempt),
				CreatedAt:  task.CreatedAt.Unix(),
				UpdatedAt:  task.UpdatedAt.Unix(),
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
		b.StartTimer()
	}
}
