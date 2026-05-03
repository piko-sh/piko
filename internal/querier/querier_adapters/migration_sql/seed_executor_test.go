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
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_dto"
)

type fakeSeedStore struct {
	applied           map[int64]struct{}
	lockChannel       chan struct{}
	mu                sync.Mutex
	seedSQLExecutions atomic.Int32
	insertConflicts   atomic.Int32
	lockHolder        atomic.Int32
	peakLockHolders   atomic.Int32
	lockAcquisitions  atomic.Int32
	lockReleases      atomic.Int32
}

func newFakeSeedStore() *fakeSeedStore {
	store := &fakeSeedStore{
		applied:     make(map[int64]struct{}),
		lockChannel: make(chan struct{}, 1),
	}
	store.lockChannel <- struct{}{}
	return store
}

type fakeSeedConnector struct {
	store *fakeSeedStore
}

func (c *fakeSeedConnector) Connect(_ context.Context) (driver.Conn, error) {
	return &fakeSeedConn{store: c.store}, nil
}

func (c *fakeSeedConnector) Driver() driver.Driver {
	return &fakeSeedDriver{store: c.store}
}

type fakeSeedDriver struct {
	store *fakeSeedStore
}

func (d *fakeSeedDriver) Open(_ string) (driver.Conn, error) {
	return &fakeSeedConn{store: d.store}, nil
}

type fakeSeedConn struct {
	store *fakeSeedStore
}

func (*fakeSeedConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("Prepare not supported by fakeSeedConn")
}

func (*fakeSeedConn) Close() error { return nil }

func (c *fakeSeedConn) Begin() (driver.Tx, error) {
	return &fakeSeedTx{conn: c}, nil
}

func (c *fakeSeedConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return &fakeSeedTx{conn: c}, nil
}

func (c *fakeSeedConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.store.exec(query, args)
}

func (c *fakeSeedConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	return &emptyRows{}, nil
}

type fakeSeedTx struct {
	conn *fakeSeedConn
}

func (*fakeSeedTx) Commit() error   { return nil }
func (*fakeSeedTx) Rollback() error { return nil }

type emptyRows struct{}

func (*emptyRows) Columns() []string {
	return []string{"version", "name", "checksum", "applied_at", "duration_ms"}
}
func (*emptyRows) Close() error                { return nil }
func (*emptyRows) Next(_ []driver.Value) error { return io.EOF }

func (s *fakeSeedStore) exec(query string, args []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "CREATE TABLE IF NOT EXISTS piko_seeds"):
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "pg_advisory_lock(hashtext('piko_seeds'))"):
		<-s.lockChannel
		holders := s.lockHolder.Add(1)
		if holders > s.peakLockHolders.Load() {
			s.peakLockHolders.Store(holders)
		}
		s.lockAcquisitions.Add(1)
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "pg_advisory_unlock(hashtext('piko_seeds'))"):
		s.lockHolder.Add(-1)
		s.lockReleases.Add(1)
		s.lockChannel <- struct{}{}
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "INSERT") && strings.Contains(query, "piko_seeds"):
		return s.recordSeed(query, args)
	case strings.Contains(query, "DELETE FROM piko_seeds"):
		s.mu.Lock()
		s.applied = make(map[int64]struct{})
		s.mu.Unlock()
		return driver.RowsAffected(0), nil
	case strings.HasPrefix(strings.TrimSpace(query), "/* seed body"):
		s.seedSQLExecutions.Add(1)
		return driver.RowsAffected(0), nil
	default:
		return nil, fmt.Errorf("fakeSeedStore: unhandled SQL %q", query)
	}
}

func (s *fakeSeedStore) recordSeed(query string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) == 0 {
		return nil, errors.New("fakeSeedStore: INSERT missing args")
	}
	version, ok := args[0].Value.(int64)
	if !ok {
		return nil, fmt.Errorf("fakeSeedStore: INSERT version arg expected int64, got %T", args[0].Value)
	}

	idempotent := strings.Contains(query, "ON CONFLICT") || strings.Contains(query, "INSERT IGNORE")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.applied[version]; exists {
		if idempotent {
			s.insertConflicts.Add(1)
			return driver.RowsAffected(0), nil
		}
		return nil, fmt.Errorf("fakeSeedStore: duplicate key value violates unique constraint (version=%d)", version)
	}
	s.applied[version] = struct{}{}
	return driver.RowsAffected(1), nil
}

func openFakeSeedDB(store *fakeSeedStore) *sql.DB {
	return sql.OpenDB(&fakeSeedConnector{store: store})
}

func TestSeedExecutor_ExecuteSeed_UsesIdempotentInsertForPostgresDialect(t *testing.T) {
	t.Parallel()

	store := newFakeSeedStore()
	store.applied[42] = struct{}{}

	database := openFakeSeedDB(store)
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewSeedExecutor(database, migration_sql.PostgresDialect())

	err := executor.ExecuteSeed(t.Context(), querier_dto.SeedRecord{
		Version:  42,
		Name:     "users",
		Checksum: "abc",
		Content:  []byte("/* seed body 42 */"),
	})

	require.NoError(t, err, "second writer must not surface a primary-key violation when the dialect uses ON CONFLICT")
	require.Equal(t, int32(1), store.insertConflicts.Load(), "the conflict path should have absorbed the duplicate INSERT")
}

func TestSeedExecutor_ExecuteSeed_UsesInsertIgnoreForMySQLDialect(t *testing.T) {
	t.Parallel()

	store := newFakeSeedStore()
	store.applied[7] = struct{}{}

	database := openFakeSeedDB(store)
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewSeedExecutor(database, migration_sql.MySQLDialect())

	err := executor.ExecuteSeed(t.Context(), querier_dto.SeedRecord{
		Version:  7,
		Name:     "products",
		Checksum: "xyz",
		Content:  []byte("/* seed body 7 */"),
	})

	require.NoError(t, err)
	require.Equal(t, int32(1), store.insertConflicts.Load(), "MySQL dialect should hit INSERT IGNORE on duplicates")
}

func TestSeedExecutor_ConcurrentApplyIsIdempotent(t *testing.T) {
	t.Parallel()

	store := newFakeSeedStore()

	database := openFakeSeedDB(store)
	t.Cleanup(func() { _ = database.Close() })

	const replicas = 4
	startBarrier := make(chan struct{})
	results := make(chan error, replicas)

	for range replicas {
		executor := migration_sql.NewSeedExecutor(database, migration_sql.PostgresDialect())
		go func() {
			<-startBarrier
			results <- executor.ExecuteSeed(t.Context(), querier_dto.SeedRecord{
				Version:  100,
				Name:     "concurrent",
				Checksum: "deadbeef",
				Content:  []byte("/* seed body 100 */"),
			})
		}()
	}

	close(startBarrier)
	for range replicas {
		require.NoError(t, <-results, "every concurrent writer must succeed without surfacing a duplicate-key violation")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	require.Len(t, store.applied, 1, "exactly one piko_seeds row must exist for the version")
}

func TestSeedExecutor_ImplementsLockingPort(t *testing.T) {
	t.Parallel()

	dialect := migration_sql.SQLiteDialect()
	executor := migration_sql.NewSeedExecutor(nil, dialect)

	err := executor.AcquireSeedLock(t.Context())
	require.NoError(t, err, "no-op SQLite lock must succeed without a database")

	err = executor.ReleaseSeedLock(t.Context())
	require.NoError(t, err, "no-op SQLite lock release must succeed")
}

func TestSeedExecutor_AdvisoryLockSerialisesConcurrentRuns(t *testing.T) {
	t.Parallel()

	store := newFakeSeedStore()

	database := openFakeSeedDB(store)
	t.Cleanup(func() { _ = database.Close() })

	const concurrentRuns = 8
	startBarrier := make(chan struct{})
	done := make(chan error, concurrentRuns)

	for runIndex := range concurrentRuns {
		executor := migration_sql.NewSeedExecutor(database, migration_sql.PostgresDialect())
		go func() {
			<-startBarrier
			if acquireErr := executor.AcquireSeedLock(t.Context()); acquireErr != nil {
				done <- acquireErr
				return
			}
			defer func() {
				_ = executor.ReleaseSeedLock(t.Context())
			}()
			done <- executor.ExecuteSeed(t.Context(), querier_dto.SeedRecord{
				Version:  int64(200 + runIndex),
				Name:     fmt.Sprintf("seed_%d", runIndex),
				Checksum: "chk",
				Content:  []byte("/* seed body N */"),
			})
		}()
	}

	close(startBarrier)
	for range concurrentRuns {
		require.NoError(t, <-done)
	}

	require.Equal(t, int32(1), store.peakLockHolders.Load(),
		"advisory lock must serialise replicas; peak concurrent holders should never exceed one")
	require.Equal(t, int32(concurrentRuns), store.lockAcquisitions.Load(),
		"every replica should have acquired the lock once")
	require.Equal(t, int32(concurrentRuns), store.lockReleases.Load(),
		"every replica should have released the lock once")
}

func TestPostgresAdvisorySeedLock_ImplementsLockStrategy(t *testing.T) {
	t.Parallel()

	var _ migration_sql.LockStrategy = (*migration_sql.PostgresAdvisorySeedLock)(nil)
}

func TestMySQLAdvisorySeedLock_ImplementsLockStrategy(t *testing.T) {
	t.Parallel()

	var _ migration_sql.LockStrategy = (*migration_sql.MySQLAdvisorySeedLock)(nil)
}

func TestTableBasedSeedLock_ImplementsLockStrategy(t *testing.T) {
	t.Parallel()

	var _ migration_sql.LockStrategy = (*migration_sql.TableBasedSeedLock)(nil)
}

func TestPostgresAdvisorySeedLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.PostgresAdvisorySeedLock{}

	require.NoError(t, lock.Release(context.Background(), nil))
}

func TestMySQLAdvisorySeedLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.MySQLAdvisorySeedLock{}

	require.NoError(t, lock.Release(context.Background(), nil))
}

func TestTableBasedSeedLock_ReleaseAcceptsNilConnection(t *testing.T) {
	t.Parallel()

	lock := &migration_sql.TableBasedSeedLock{
		CreateLockTableSQL: "CREATE TABLE",
	}

	require.NoError(t, lock.Release(context.Background(), nil))
}

func TestSeedExecutor_AdvisoryLockBlocksConcurrentRunsOnSQLite(t *testing.T) {
	t.Parallel()

	t.Skip("SQLite uses NoOpLock by design; advisory-lock semantics are exercised " +
		"against PostgreSQL/MySQL in tests/integration/querier_*.")
}
