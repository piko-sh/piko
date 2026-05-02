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
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_dto"
)

type c2Store struct {
	mu                  sync.Mutex
	versions            map[int64]bool
	failingStmtFailures int
}

func (s *c2Store) version(args []driver.NamedValue) int64 {
	if len(args) == 0 {
		return 0
	}
	if value, ok := args[0].Value.(int64); ok {
		return value
	}
	return 0
}

func (s *c2Store) exec(query string, args []driver.NamedValue) (driver.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case strings.Contains(query, "INSERT INTO piko_migrations"):
		version := s.version(args)
		if s.versions[version] {
			return nil, errors.New("UNIQUE constraint failed: piko_migrations.version")
		}
		s.versions[version] = true
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "FAIL_ONCE"):
		if s.failingStmtFailures > 0 {
			s.failingStmtFailures--
			return nil, errors.New("simulated statement 0 failure")
		}
		return driver.RowsAffected(0), nil
	default:

		return driver.RowsAffected(0), nil
	}
}

func (s *c2Store) query(query string, args []driver.NamedValue) (driver.Rows, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.Contains(query, "count(*)") {
		count := int64(0)
		if s.versions[s.version(args)] {
			count = 1
		}
		return &c2CountRows{count: count}, nil
	}
	return &c2EmptyRows{}, nil
}

type c2Connector struct{ store *c2Store }

func (c *c2Connector) Connect(context.Context) (driver.Conn, error) {
	return &c2Conn{store: c.store}, nil
}
func (c *c2Connector) Driver() driver.Driver { return &c2Driver{store: c.store} }

type c2Driver struct{ store *c2Store }

func (d *c2Driver) Open(string) (driver.Conn, error) { return &c2Conn{store: d.store}, nil }

type c2Conn struct{ store *c2Store }

func (*c2Conn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("prepare unsupported") }
func (*c2Conn) Close() error                        { return nil }
func (*c2Conn) Begin() (driver.Tx, error)           { return c2Tx{}, nil }
func (*c2Conn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c2Tx{}, nil
}

func (c *c2Conn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.store.exec(query, args)
}

func (c *c2Conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.store.query(query, args)
}

type c2Tx struct{}

func (c2Tx) Commit() error   { return nil }
func (c2Tx) Rollback() error { return nil }

type c2CountRows struct {
	count int64
	done  bool
}

func (*c2CountRows) Columns() []string { return []string{"count"} }
func (*c2CountRows) Close() error      { return nil }
func (r *c2CountRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	dest[0] = r.count
	return nil
}

type c2EmptyRows struct{}

func (*c2EmptyRows) Columns() []string           { return []string{"unused"} }
func (*c2EmptyRows) Close() error                { return nil }
func (*c2EmptyRows) Next(_ []driver.Value) error { return io.EOF }

func TestExecutor_FirstStatementFailureThenRetryResumes(t *testing.T) {
	t.Parallel()

	store := &c2Store{versions: map[int64]bool{}, failingStmtFailures: 1}
	database := sql.OpenDB(&c2Connector{store: store})
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewExecutor(database, migration_sql.SQLiteDialect())

	migration := querier_dto.MigrationRecord{
		Name:     "create_things",
		Checksum: "abc",
		Content:  []byte("FAIL_ONCE;\nOK2;\n"),
		Version:  1,
		SkipUpTo: -1,
	}

	firstErr := executor.ExecuteMigration(t.Context(), migration, querier_dto.MigrationDirectionUp, false)
	require.Error(t, firstErr)

	retryErr := executor.ExecuteMigration(t.Context(), migration, querier_dto.MigrationDirectionUp, false)
	require.NoError(t, retryErr, "retry after first-statement failure must resume, not hit a duplicate-key error")
}
