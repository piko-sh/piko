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
	"piko.sh/piko/internal/querier/querier_domain"
)

type fakeLockStore struct {
	mu               sync.Mutex
	held             bool
	acquisitions     atomic.Int32
	releases         atomic.Int32
	failTableLockSQL bool
}

func (s *fakeLockStore) tryHold() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held {
		return false
	}
	s.held = true
	s.acquisitions.Add(1)
	return true
}

func (s *fakeLockStore) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held = false
	s.releases.Add(1)
}

type fakeLockConnector struct {
	store *fakeLockStore
}

func (c *fakeLockConnector) Connect(_ context.Context) (driver.Conn, error) {
	return &fakeLockConn{store: c.store}, nil
}

func (c *fakeLockConnector) Driver() driver.Driver {
	return &fakeLockDriver{store: c.store}
}

type fakeLockDriver struct {
	store *fakeLockStore
}

func (d *fakeLockDriver) Open(_ string) (driver.Conn, error) {
	return &fakeLockConn{store: d.store}, nil
}

type fakeLockConn struct {
	store *fakeLockStore
}

func (*fakeLockConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("Prepare not supported by fakeLockConn")
}

func (*fakeLockConn) Close() error { return nil }

func (c *fakeLockConn) Begin() (driver.Tx, error) { return &fakeLockTx{conn: c}, nil }

func (c *fakeLockConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return &fakeLockTx{conn: c}, nil
}

func (c *fakeLockConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	return c.store.exec(query)
}

func (c *fakeLockConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	return c.store.query(query)
}

type fakeLockTx struct {
	conn *fakeLockConn
}

func (*fakeLockTx) Commit() error   { return nil }
func (*fakeLockTx) Rollback() error { return nil }

type singleValueRows struct {
	value    driver.Value
	consumed bool
}

func (*singleValueRows) Columns() []string { return []string{"result"} }
func (*singleValueRows) Close() error      { return nil }
func (r *singleValueRows) Next(dest []driver.Value) error {
	if r.consumed {
		return io.EOF
	}
	r.consumed = true
	dest[0] = r.value
	return nil
}

func (s *fakeLockStore) exec(query string) (driver.Result, error) {
	switch {
	case strings.Contains(query, "pg_advisory_lock(hashtext('piko_migrations'))"):

		s.tryHold()
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "pg_advisory_unlock(hashtext('piko_migrations'))"):
		s.release()
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "RELEASE_LOCK('piko_migrations')"):
		s.release()
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "CREATE TABLE") && strings.Contains(query, "piko_migration_lock"):
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "INSERT INTO piko_migration_lock"):
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "FOR UPDATE NOWAIT") && strings.Contains(query, "piko_migration_lock"):
		if !s.tryHold() {
			return nil, errors.New("pq: could not obtain lock on row in relation \"piko_migration_lock\"")
		}
		return driver.RowsAffected(0), nil
	case strings.Contains(query, "FOR UPDATE") && strings.Contains(query, "piko_migration_lock"):
		if s.failTableLockSQL {
			return nil, errors.New("pq: deadlock detected")
		}
		s.tryHold()
		return driver.RowsAffected(0), nil
	default:
		return nil, fmt.Errorf("fakeLockStore.exec: unhandled SQL %q", query)
	}
}

func (s *fakeLockStore) query(query string) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "pg_try_advisory_lock(hashtext('piko_migrations'))"):
		return &singleValueRows{value: s.tryHold()}, nil
	case strings.Contains(query, "GET_LOCK('piko_migrations', -1)"):
		s.tryHold()
		return &singleValueRows{value: int64(1)}, nil
	case strings.Contains(query, "GET_LOCK('piko_migrations', 0)"):
		if s.tryHold() {
			return &singleValueRows{value: int64(1)}, nil
		}
		return &singleValueRows{value: int64(0)}, nil
	default:
		return nil, fmt.Errorf("fakeLockStore.query: unhandled SQL %q", query)
	}
}

func openFakeLockDB(store *fakeLockStore) *sql.DB {
	return sql.OpenDB(&fakeLockConnector{store: store})
}

func TestPostgresAdvisoryLock_AcquireAndRelease(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	lock := &migration_sql.PostgresAdvisoryLock{}

	connection, err := lock.Acquire(t.Context(), database)
	require.NoError(t, err)
	require.NotNil(t, connection, "PostgreSQL advisory lock must pin a connection")
	require.Equal(t, int32(1), store.acquisitions.Load())

	require.NoError(t, lock.Release(t.Context(), connection))
	require.Equal(t, int32(1), store.releases.Load())
}

func TestPostgresAdvisoryLock_TryAcquireSucceedsWhenFree(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	lock := &migration_sql.PostgresAdvisoryLock{}

	connection, err := lock.TryAcquire(t.Context(), database)
	require.NoError(t, err)
	require.NotNil(t, connection)
	require.NoError(t, lock.Release(t.Context(), connection))
}

func TestPostgresAdvisoryLock_TryAcquireReturnsLockNotAcquiredWhenHeld(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	first := &migration_sql.PostgresAdvisoryLock{}
	firstConn, err := first.Acquire(t.Context(), database)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Release(t.Context(), firstConn) })

	second := &migration_sql.PostgresAdvisoryLock{}
	_, secondErr := second.TryAcquire(t.Context(), database)
	require.ErrorIs(t, secondErr, querier_domain.ErrLockNotAcquired,
		"a non-blocking acquire against a held advisory lock must surface ErrLockNotAcquired")
}

func TestMySQLAdvisoryLock_AcquireAndRelease(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	lock := &migration_sql.MySQLAdvisoryLock{}

	connection, err := lock.Acquire(t.Context(), database)
	require.NoError(t, err)
	require.NotNil(t, connection, "MySQL advisory lock must pin a connection")
	require.Equal(t, int32(1), store.acquisitions.Load())

	require.NoError(t, lock.Release(t.Context(), connection))
	require.Equal(t, int32(1), store.releases.Load())
}

func TestMySQLAdvisoryLock_TryAcquireReturnsLockNotAcquiredWhenHeld(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	first := &migration_sql.MySQLAdvisoryLock{}
	firstConn, err := first.Acquire(t.Context(), database)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Release(t.Context(), firstConn) })

	second := &migration_sql.MySQLAdvisoryLock{}
	_, secondErr := second.TryAcquire(t.Context(), database)
	require.ErrorIs(t, secondErr, querier_domain.ErrLockNotAcquired,
		"GET_LOCK returning 0 must map to ErrLockNotAcquired")
}

func TestTableBasedLock_AcquireAndRelease(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	lock := &migration_sql.TableBasedLock{
		CreateLockTableSQL: "CREATE TABLE IF NOT EXISTS piko_migration_lock (lock_id INT PRIMARY KEY)",
	}

	connection, err := lock.Acquire(t.Context(), database)
	require.NoError(t, err)
	require.NotNil(t, connection, "table-based lock must pin a connection")

	require.NoError(t, lock.Release(t.Context(), connection))
}

func TestTableBasedLock_TryAcquireReturnsLockNotAcquiredWhenHeld(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	first := &migration_sql.TableBasedLock{
		CreateLockTableSQL: "CREATE TABLE IF NOT EXISTS piko_migration_lock (lock_id INT PRIMARY KEY)",
	}
	firstConn, err := first.Acquire(t.Context(), database)
	require.NoError(t, err)
	t.Cleanup(func() { _ = first.Release(t.Context(), firstConn) })

	second := &migration_sql.TableBasedLock{
		CreateLockTableSQL: "CREATE TABLE IF NOT EXISTS piko_migration_lock (lock_id INT PRIMARY KEY)",
	}
	_, secondErr := second.TryAcquire(t.Context(), database)
	require.ErrorIs(t, secondErr, querier_domain.ErrLockNotAcquired,
		"FOR UPDATE NOWAIT against a held row must map to ErrLockNotAcquired")
}

func TestTableBasedLock_AcquireRejectsDoubleAcquireWithoutRelease(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	lock := &migration_sql.TableBasedLock{
		CreateLockTableSQL: "CREATE TABLE IF NOT EXISTS piko_migration_lock (lock_id INT PRIMARY KEY)",
	}

	connection, err := lock.Acquire(t.Context(), database)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lock.Release(t.Context(), connection) })

	_, secondErr := lock.Acquire(t.Context(), database)
	require.Error(t, secondErr, "re-acquiring a held table lock must be refused to avoid leaking the transaction")
}

func TestTableBasedLock_AcquirePropagatesNonContentionError(t *testing.T) {
	t.Parallel()

	store := &fakeLockStore{failTableLockSQL: true}
	database := openFakeLockDB(store)
	t.Cleanup(func() { _ = database.Close() })

	lock := &migration_sql.TableBasedLock{
		CreateLockTableSQL: "CREATE TABLE IF NOT EXISTS piko_migration_lock (lock_id INT PRIMARY KEY)",
	}

	_, err := lock.Acquire(t.Context(), database)
	require.Error(t, err)
	require.NotErrorIs(t, err, querier_domain.ErrLockNotAcquired,
		"a non-contention lock failure must not be misreported as ErrLockNotAcquired")
}
