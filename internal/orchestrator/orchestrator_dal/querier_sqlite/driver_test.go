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

package querier_sqlite

import (
	"context"
	"database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbpkg "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type recordingDBTX struct {
	queries []string
}

func (r *recordingDBTX) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	r.queries = append(r.queries, query)

	return nil, sql.ErrConnDone
}

func (r *recordingDBTX) QueryContext(_ context.Context, query string, _ ...any) (*sql.Rows, error) {
	r.queries = append(r.queries, query)

	return nil, sql.ErrConnDone
}

func (r *recordingDBTX) QueryRowContext(_ context.Context, query string, _ ...any) *sql.Row {
	r.queries = append(r.queries, query)

	return nil
}

func TestNewObserved_RunsStatementsThroughTheObservedHandle(t *testing.T) {
	t.Parallel()

	recorder := &recordingDBTX{}
	dal := NewObserved(new(sql.DB), recorder)

	require.NotNil(t, dal, "the DAL is constructed from the two handles")
	assert.Implements(t, (*any)(nil), dal)

	queries := dbpkg.New(recorder)
	assert.NotNil(t, queries)
}

type taskRowsConnector struct {
	rows      map[string][]sqldriver.Value
	queries   []string
	argCounts []int
	mu        sync.Mutex
}

func (c *taskRowsConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &taskRowsConn{connector: c}, nil
}

func (c *taskRowsConnector) Driver() sqldriver.Driver { return taskRowsDriver{connector: c} }

func (c *taskRowsConnector) recorded() ([]string, []int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.queries), slices.Clone(c.argCounts)
}

type taskRowsDriver struct {
	connector *taskRowsConnector
}

func (d taskRowsDriver) Open(string) (sqldriver.Conn, error) {
	return &taskRowsConn{connector: d.connector}, nil
}

type taskRowsConn struct {
	connector *taskRowsConnector
}

func (*taskRowsConn) Prepare(string) (sqldriver.Stmt, error) { return nil, errors.New("not supported") }
func (*taskRowsConn) Close() error                           { return nil }
func (*taskRowsConn) Begin() (sqldriver.Tx, error)           { return nil, errors.New("not supported") }

func (c *taskRowsConn) QueryContext(_ context.Context, query string, args []sqldriver.NamedValue) (sqldriver.Rows, error) {
	c.connector.mu.Lock()
	defer c.connector.mu.Unlock()
	c.connector.queries = append(c.connector.queries, query)
	c.connector.argCounts = append(c.connector.argCounts, len(args))

	var matched [][]sqldriver.Value
	for _, arg := range args {
		if id, ok := arg.Value.(string); ok {
			if row, found := c.connector.rows[id]; found {
				matched = append(matched, row)
			}
		}
	}
	return &taskRows{rows: matched}, nil
}

type taskRows struct {
	rows [][]sqldriver.Value
	next int
}

func (*taskRows) Columns() []string {
	return []string{
		"id", "workflow_id", "executor", "priority", "payload", "config", "result", "status",
		"execute_at", "attempt", "last_error", "created_at", "updated_at", "deduplication_key",
	}
}

func (*taskRows) Close() error { return nil }

func (r *taskRows) Next(dest []sqldriver.Value) error {
	if r.next >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.next])
	r.next++
	return nil
}

func taskRow(id, status string) []sqldriver.Value {
	return []sqldriver.Value{
		id, "wf-" + id, "test-executor", int64(1), `{"artefactID":"a"}`, `{}`, nil, status,
		int64(1700000000), int64(2), "worker crashed", int64(1700000000), int64(1700000100), "key-" + id,
	}
}

func TestDriver_GetTasksByID(t *testing.T) {
	t.Parallel()

	connector := &taskRowsConnector{rows: map[string][]sqldriver.Value{
		"t1": taskRow("t1", "RETRYING"),
		"t2": taskRow("t2", "FAILED"),
	}}
	database := sql.OpenDB(connector)
	t.Cleanup(func() { _ = database.Close() })

	tasks, err := New(database).GetTasksByID(t.Context(), []string{"t1", "missing", "t2"})
	require.NoError(t, err)
	require.Len(t, tasks, 2)

	assert.Equal(t, "t1", tasks[0].ID)
	assert.Equal(t, "wf-t1", tasks[0].WorkflowID)
	assert.Equal(t, "test-executor", tasks[0].Executor)
	assert.Equal(t, orchestrator_domain.StatusRetrying, tasks[0].Status)
	assert.Equal(t, 2, tasks[0].Attempt)
	assert.Equal(t, "worker crashed", tasks[0].LastError)
	assert.Equal(t, "key-t1", tasks[0].DeduplicationKey)
	assert.Equal(t, "a", tasks[0].Payload["artefactID"])
	assert.Equal(t, orchestrator_domain.StatusFailed, tasks[1].Status)

	queries, argCounts := connector.recorded()
	require.Len(t, queries, 1)
	assert.Contains(t, queries[0], "id IN (")
	assert.Equal(t, []int{3}, argCounts, "one bind variable per requested ID")
}
