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

package worker_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
	"piko.sh/piko/wdk/worker"

	"piko.sh/piko/internal/worker/worker_dal/querier_sqlite"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	migrationsDir = "../../../internal/worker/worker_dal/querier_sqlite/migrations"
)

func openDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "sqlite.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err, "open sqlite")
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})
	require.NoError(t, db.Ping(), "ping sqlite")
	return db
}

func applyMigrations(t *testing.T, db *sql.DB, migrationName string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(migrationsDir, migrationName))
	require.NoError(t, err, "read migration file")
	_, err = db.Exec(string(body))
	require.NoError(t, err, "execute migration")
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	require.NoError(t, err, "query sqlite_master")
	return strings.EqualFold(name, table)
}

func newStore(t *testing.T) worker_domain.Store {
	t.Helper()
	database := openDatabase(t)
	applyMigrations(t, database, "001_worker.up.sql")
	return querier_sqlite.New(database, clock.RealClock())
}

func newStoreWithClock(t *testing.T, clk clock.Clock) worker_domain.Store {
	t.Helper()
	database := openDatabase(t)
	applyMigrations(t, database, "001_worker.up.sql")
	return querier_sqlite.New(database, clk)
}

type runner struct {
	service  worker.Service
	database *sql.DB
	store    worker_domain.Store
	clk      clock.Clock
}

type runnerConfig struct {
	clk            clock.Clock
	workersConfig  worker_domain.WorkersConfig
	serviceOptions []worker.ServiceOption
}

type runnerOption func(config *runnerConfig)

func withClock(clk clock.Clock) runnerOption {
	return func(config *runnerConfig) {
		config.clk = clk
	}
}

func withVisibilityTimeout(d time.Duration) runnerOption {
	return func(config *runnerConfig) {
		config.workersConfig.VisibilityTimeout = d
	}
}

func withRecoveryInterval(d time.Duration) runnerOption {
	return func(config *runnerConfig) {
		config.workersConfig.RecoveryInterval = d
	}
}

func withHeartbeatInterval(d time.Duration) runnerOption {
	return func(config *runnerConfig) {
		config.workersConfig.HeartbeatInterval = d
	}
}

func withPromoteInterval(d time.Duration) runnerOption {
	return func(config *runnerConfig) {
		config.workersConfig.PromoteInterval = d
	}
}

func newRunner(t *testing.T, opts ...runnerOption) *runner {
	t.Helper()
	config := runnerConfig{
		clk: clock.RealClock(),
	}
	for _, opt := range opts {
		opt(&config)
	}
	database := openDatabase(t)
	applyMigrations(t, database, "001_worker.up.sql")
	store := querier_sqlite.New(database, clock.RealClock())
	serviceOpts := append([]worker.ServiceOption{
		worker.WithClock(config.clk),
		worker.WithConfig(config.workersConfig),
	}, config.serviceOptions...)
	service := worker.NewService(store, serviceOpts...)
	t.Cleanup(func() {
		_ = service.Shutdown(context.Background())
	})
	return &runner{
		service:  service,
		store:    store,
		database: database,
		clk:      config.clk,
	}
}

func (r *runner) seedOrphanedJob(t *testing.T, args welcomeArgs, claimedByWorkerID string, claimedAt time.Time) *worker.Handle {
	t.Helper()
	payload, err := json.Marshal(args)
	require.NoError(t, err, "marshal orphan args")
	id := "orphan_" + args.UserID
	now := r.clk.Now().UTC().Format(time.RFC3339Nano)
	claimedAtFormatted := claimedAt.UTC().Format(time.RFC3339Nano)

	_, err = r.database.Exec(`INSERT INTO jobs (
    id, kind, queue, payload, max_attempts, timeout_seconds, created_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)`, id, args.Kind(), "default", string(payload), 3, 300, now)
	require.NoError(t, err, "seed orphaned job root")

	_, err = r.database.Exec(`INSERT INTO job_versions (
    job_id, event, status, priority, scheduled_at, attempt,
    claimed_by_worker_id, claimed_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)`, id, "claimed", "running", 5, now, 1, claimedByWorkerID, claimedAtFormatted)
	require.NoError(t, err, "seed orphaned claimed version")

	return worker_domain.NewHandle(id, r.service)
}

func (r *runner) Service() worker.Service {
	return r.service
}

func (r *runner) Start(ctx context.Context) error {
	return r.service.Start(ctx)
}

func pendingSpec(store worker_domain.Store, id, kind string) worker_dto.EnqueueSpec {
	return worker_dto.EnqueueSpec{
		ID:             id,
		Kind:           kind,
		Queue:          "default",
		Payload:        []byte(`{}`),
		Priority:       5,
		MaxAttempts:    3,
		TimeoutSeconds: 300,
		ScheduledAt:    store.Now(),
	}
}
