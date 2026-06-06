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

package migration_sql_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/clock"
)

type steppingClock struct {
	clock.Clock
	mu   sync.Mutex
	now  time.Time
	step time.Duration
}

func newSteppingClock(start time.Time, step time.Duration) *steppingClock {
	return &steppingClock{Clock: clock.RealClock(), now: start, step: step}
}

func (c *steppingClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	current := c.now
	c.now = c.now.Add(c.step)
	return current
}

type historyCapture struct {
	mu         sync.Mutex
	appliedAt  time.Time
	durationMs int64
	captured   bool
}

func (capture *historyCapture) recordInsert(query string, args []driver.NamedValue) {
	capture.mu.Lock()
	defer capture.mu.Unlock()
	switch {
	case strings.Contains(query, "INSERT INTO piko_migrations"):
		if appliedAt, ok := args[3].Value.(time.Time); ok {
			capture.appliedAt = appliedAt
		}
		if durationMs, ok := args[4].Value.(int64); ok {
			capture.durationMs = durationMs
		}
		capture.captured = true
	case strings.Contains(query, "INSERT INTO piko_seeds"):
		if durationMs, ok := args[3].Value.(int64); ok {
			capture.durationMs = durationMs
		}
		capture.captured = true
	}
}

type captureConnector struct{ capture *historyCapture }

func (c *captureConnector) Connect(context.Context) (driver.Conn, error) {
	return &captureConn{capture: c.capture}, nil
}
func (c *captureConnector) Driver() driver.Driver { return &captureDriver{capture: c.capture} }

type captureDriver struct{ capture *historyCapture }

func (d *captureDriver) Open(string) (driver.Conn, error) {
	return &captureConn{capture: d.capture}, nil
}

type captureConn struct{ capture *historyCapture }

func (*captureConn) Prepare(string) (driver.Stmt, error) { return nil, io.ErrUnexpectedEOF }
func (*captureConn) Close() error                        { return nil }
func (*captureConn) Begin() (driver.Tx, error)           { return captureTx{}, nil }
func (*captureConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return captureTx{}, nil
}

func (c *captureConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.capture.recordInsert(query, args)
	return driver.RowsAffected(1), nil
}

func (*captureConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &captureEmptyRows{}, nil
}

type captureTx struct{}

func (captureTx) Commit() error   { return nil }
func (captureTx) Rollback() error { return nil }

type captureEmptyRows struct{}

func (*captureEmptyRows) Columns() []string         { return []string{"unused"} }
func (*captureEmptyRows) Close() error              { return nil }
func (*captureEmptyRows) Next([]driver.Value) error { return io.EOF }

func TestExecuteMigrationRecordsClockDerivedTimings(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	const step = 1500 * time.Millisecond

	capture := &historyCapture{}
	database := sql.OpenDB(&captureConnector{capture: capture})
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewExecutor(
		database,
		migration_sql.SQLiteDialect(),
		migration_sql.WithExecutorClock(newSteppingClock(start, step)),
	)

	migration := querier_dto.MigrationRecord{
		Name:     "create_things",
		Checksum: "abc",
		Content:  []byte("CREATE TABLE things (id INTEGER);\n"),
		Version:  1,
	}

	require.NoError(t, executor.ExecuteMigration(t.Context(), migration, querier_dto.MigrationDirectionUp, true))

	require.True(t, capture.captured, "the migration history INSERT must have run")
	require.True(t, capture.appliedAt.Equal(start), "applied_at must be the clock's start time, got %s", capture.appliedAt)
	require.Equal(t, step.Milliseconds(), capture.durationMs, "duration_ms must be the gap between the two clock reads")
}

func TestExecuteSeedRecordsClockDerivedDuration(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	const step = 2500 * time.Millisecond

	capture := &historyCapture{}
	database := sql.OpenDB(&captureConnector{capture: capture})
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewSeedExecutor(
		database,
		migration_sql.SQLiteDialect(),
		migration_sql.WithSeedExecutorClock(newSteppingClock(start, step)),
	)

	seed := querier_dto.SeedRecord{
		Name:     "seed_things",
		Checksum: "def",
		Content:  []byte("INSERT INTO things (id) VALUES (1);\n"),
		Version:  1,
	}

	require.NoError(t, executor.ExecuteSeed(t.Context(), seed))

	require.True(t, capture.captured, "the seed history INSERT must have run")
	require.Equal(t, step.Milliseconds(), capture.durationMs, "duration_ms must be the gap between the two clock reads")
}

func TestWithExecutorClockIgnoresNil(t *testing.T) {
	t.Parallel()

	capture := &historyCapture{}
	database := sql.OpenDB(&captureConnector{capture: capture})
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewExecutor(
		database,
		migration_sql.SQLiteDialect(),
		migration_sql.WithExecutorClock(nil),
	)

	migration := querier_dto.MigrationRecord{
		Name:    "create_things",
		Content: []byte("CREATE TABLE things (id INTEGER);\n"),
		Version: 1,
	}

	require.NoError(t, executor.ExecuteMigration(t.Context(), migration, querier_dto.MigrationDirectionUp, true))
	require.True(t, capture.captured, "the default clock must still drive a working migration")
}
