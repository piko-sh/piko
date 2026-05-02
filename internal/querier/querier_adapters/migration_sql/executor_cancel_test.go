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
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_dto"
)

type cancelStore struct {
	mu                   sync.Mutex
	cancel               context.CancelFunc
	cancelAfterStatement string
	lastStatementWrites  []int64
	versions             map[int64]bool
}

func (store *cancelStore) exec(query string, args []driver.NamedValue) (driver.Result, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	switch {
	case strings.Contains(query, "INSERT INTO piko_migrations"):
		version := namedInt64(args, 0)
		if store.versions[version] {
			return nil, errors.New("UNIQUE constraint failed: piko_migrations.version")
		}
		store.versions[version] = true
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE piko_migrations") && strings.Contains(query, "last_statement"):
		store.lastStatementWrites = append(store.lastStatementWrites, namedInt64(args, 0))
		return driver.RowsAffected(1), nil
	default:
		if store.cancel != nil && store.cancelAfterStatement != "" &&
			strings.Contains(query, store.cancelAfterStatement) {
			store.cancel()
			store.cancel = nil
		}
		return driver.RowsAffected(0), nil
	}
}

func (store *cancelStore) query(query string, args []driver.NamedValue) (driver.Rows, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if strings.Contains(query, "count(*)") {
		count := int64(0)
		if store.versions[namedInt64(args, 0)] {
			count = 1
		}
		return &cancelCountRows{count: count}, nil
	}
	return &cancelEmptyRows{}, nil
}

func (store *cancelStore) writes() []int64 {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]int64(nil), store.lastStatementWrites...)
}

func namedInt64(args []driver.NamedValue, position int) int64 {
	if position >= len(args) {
		return 0
	}
	if value, ok := args[position].Value.(int64); ok {
		return value
	}
	return 0
}

type cancelConnector struct{ store *cancelStore }

func (connector *cancelConnector) Connect(context.Context) (driver.Conn, error) {
	return &cancelConn{store: connector.store}, nil
}
func (connector *cancelConnector) Driver() driver.Driver {
	return &cancelDriver{store: connector.store}
}

type cancelDriver struct{ store *cancelStore }

func (driverImpl *cancelDriver) Open(string) (driver.Conn, error) {
	return &cancelConn{store: driverImpl.store}, nil
}

type cancelConn struct{ store *cancelStore }

func (*cancelConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare unsupported")
}
func (*cancelConn) Close() error              { return nil }
func (*cancelConn) Begin() (driver.Tx, error) { return cancelTx{}, nil }
func (*cancelConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return cancelTx{}, nil
}

func (conn *cancelConn) ExecContext(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return conn.store.exec(query, args)
}

func (conn *cancelConn) QueryContext(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Rows, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return conn.store.query(query, args)
}

type cancelTx struct{}

func (cancelTx) Commit() error   { return nil }
func (cancelTx) Rollback() error { return nil }

type cancelCountRows struct {
	count int64
	done  bool
}

func (*cancelCountRows) Columns() []string { return []string{"count"} }
func (*cancelCountRows) Close() error      { return nil }
func (rows *cancelCountRows) Next(dest []driver.Value) error {
	if rows.done {
		return io.EOF
	}
	rows.done = true
	dest[0] = rows.count
	return nil
}

type cancelEmptyRows struct{}

func (*cancelEmptyRows) Columns() []string           { return []string{"unused"} }
func (*cancelEmptyRows) Close() error                { return nil }
func (*cancelEmptyRows) Next(_ []driver.Value) error { return io.EOF }

func TestExecutor_CancellationFlushesResumeCheckpoint(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	store := &cancelStore{
		versions:             map[int64]bool{},
		cancel:               cancel,
		cancelAfterStatement: "STMT_0 ",
	}
	database := sql.OpenDB(&cancelConnector{store: store})
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewExecutor(
		database,
		migration_sql.SQLiteDialect(),
		migration_sql.WithProgressBatchSize(5),
	)

	migration := querier_dto.MigrationRecord{
		Name:     "create_things",
		Checksum: "abc",
		Content:  buildStatements(8),
		Version:  1,
		SkipUpTo: -1,
	}

	err := executor.ExecuteMigration(ctx, migration, querier_dto.MigrationDirectionUp, false)

	require.Error(t, err, "a cancelled migration must surface the cancellation error")
	require.ErrorIs(t, err, context.Canceled)

	writes := store.writes()
	require.Equal(t, []int64{0}, writes,
		"the cancellation handler must flush exactly the last completed statement index (0) via a detached context")
}

func TestExecutor_CancellationBeforeFirstStatementDoesNotFlush(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	store := &cancelStore{versions: map[int64]bool{}}
	database := sql.OpenDB(&cancelConnector{store: store})
	t.Cleanup(func() { _ = database.Close() })

	executor := migration_sql.NewExecutor(database, migration_sql.SQLiteDialect())

	migration := querier_dto.MigrationRecord{
		Name:     "create_things",
		Checksum: "abc",
		Content:  buildStatements(4),
		Version:  1,
		SkipUpTo: 0,
	}

	err := executor.ExecuteMigration(ctx, migration, querier_dto.MigrationDirectionUp, false)

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, store.writes(),
		"no statement completed past the resume point, so nothing should be flushed")
}

func TestWithProgressBatchSize_ControlsCheckpointCadence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		batchSize  int
		statements int

		wantWrites []int64
	}{
		{
			name:       "batch of two over six statements",
			batchSize:  2,
			statements: 6,
			wantWrites: []int64{1, 3, 5},
		},
		{
			name:       "batch larger than statement count flushes only at the end",
			batchSize:  100,
			statements: 4,
			wantWrites: []int64{3},
		},
		{
			name:       "zero falls back to the default so only the final statement flushes",
			batchSize:  0,
			statements: 4,
			wantWrites: []int64{3},
		},
		{
			name:       "negative falls back to the default so only the final statement flushes",
			batchSize:  -7,
			statements: 4,
			wantWrites: []int64{3},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store := &cancelStore{versions: map[int64]bool{}}
			database := sql.OpenDB(&cancelConnector{store: store})
			t.Cleanup(func() { _ = database.Close() })

			executor := migration_sql.NewExecutor(
				database,
				migration_sql.SQLiteDialect(),
				migration_sql.WithProgressBatchSize(testCase.batchSize),
			)

			migration := querier_dto.MigrationRecord{
				Name:     "create_things",
				Checksum: "abc",
				Content:  buildStatements(testCase.statements),
				Version:  1,
				SkipUpTo: -1,
			}

			err := executor.ExecuteMigration(t.Context(), migration, querier_dto.MigrationDirectionUp, false)

			require.NoError(t, err)
			require.Equal(t, testCase.wantWrites, store.writes(),
				"the batch size must control which statement indices are flushed as progress checkpoints")
		})
	}
}

func buildStatements(count int) []byte {
	var builder strings.Builder
	for index := range count {
		builder.WriteString("STMT_")
		builder.WriteString(strconv.Itoa(index))
		builder.WriteString(" col")
		builder.WriteString(strconv.Itoa(index))
		builder.WriteString(";\n")
	}
	return []byte(builder.String())
}
