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

	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	dbpkg "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// nullableSentinel is the value a write case binds when a nullable field is set.
	nullableSentinel = "nullable-sentinel"

	// nullableTimestamp is the value a read case returns for a non-NULL integer column.
	nullableTimestamp int64 = 1700000500
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
		"build_tag", "is_fatal", "scheduled_execute_at", "background", "owner_instance", "last_published_at",
		"rerun_id", "rerun_payload", "rerun_config",
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

// capturingConnector is a database/sql connector that records the arguments bound to
// each statement and answers every query with the same canned rows.
type capturingConnector struct {
	rows      [][]sqldriver.Value
	arguments [][]sqldriver.Value
	mu        sync.Mutex
}

// capturingConn is one connection opened from a capturingConnector.
type capturingConn struct {
	connector *capturingConnector
}

// capturingRows yields the canned rows of a capturingConnector.
type capturingRows struct {
	rows [][]sqldriver.Value
	next int
}

// capturingDriver opens connections that share one capturingConnector.
type capturingDriver struct {
	connector *capturingConnector
}

// nullableWriteCase describes one nullable field that a driver write binds.
type nullableWriteCase struct {
	write func(ctx context.Context, taskDriver *driver, value *string) error
	name  string
}

// nullableReadCase describes one nullable column that a driver read maps into a pointer.
type nullableReadCase[T comparable] struct {
	read   func(t *testing.T, taskDriver *driver) *T
	name   string
	column int
}

func (c *capturingConnector) Connect(context.Context) (sqldriver.Conn, error) {
	return &capturingConn{connector: c}, nil
}

func (c *capturingConnector) Driver() sqldriver.Driver { return capturingDriver{connector: c} }

func (c *capturingConnector) record(args []sqldriver.NamedValue) {
	values := make([]sqldriver.Value, len(args))
	for index := range args {
		values[index] = args[index].Value
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.arguments = append(c.arguments, values)
}

func (c *capturingConnector) recorded() [][]sqldriver.Value {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.arguments)
}

func (d capturingDriver) Open(string) (sqldriver.Conn, error) {
	return &capturingConn{connector: d.connector}, nil
}

func (*capturingConn) Prepare(string) (sqldriver.Stmt, error) {
	return nil, errors.New("not supported")
}

func (*capturingConn) Close() error { return nil }

func (*capturingConn) Begin() (sqldriver.Tx, error) { return nil, errors.New("not supported") }

func (c *capturingConn) ExecContext(_ context.Context, _ string, args []sqldriver.NamedValue) (sqldriver.Result, error) {
	c.connector.record(args)
	return sqldriver.RowsAffected(1), nil
}

func (c *capturingConn) QueryContext(_ context.Context, _ string, args []sqldriver.NamedValue) (sqldriver.Rows, error) {
	c.connector.record(args)
	return &capturingRows{rows: c.connector.rows, next: 0}, nil
}

func (r *capturingRows) Columns() []string {
	if len(r.rows) == 0 {
		return nil
	}
	columns := make([]string, len(r.rows[0]))
	for index := range columns {
		columns[index] = "column"
	}
	return columns
}

func (*capturingRows) Close() error { return nil }

func (r *capturingRows) Next(dest []sqldriver.Value) error {
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
		"build-1", false, int64(0), false, "", int64(0), nil, nil, nil,
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
	assert.Equal(t, "build-1", tasks[0].BuildTag)
	assert.Equal(t, "a", tasks[0].Payload["artefactID"])
	assert.Equal(t, orchestrator_domain.StatusFailed, tasks[1].Status)

	queries, argCounts := connector.recorded()
	require.Len(t, queries, 1)
	assert.Contains(t, queries[0], "id IN (")
	assert.Equal(t, []int{3}, argCounts, "one bind variable per requested ID")
}

// newCapturingDriver builds a SQLite driver over a fake database that answers every
// query with rows.
//
// Takes rows ([][]sqldriver.Value) which every query returns.
//
// Returns *driver which runs its statements against the fake.
// Returns *capturingConnector which records the bound arguments.
func newCapturingDriver(t *testing.T, rows ...[]sqldriver.Value) (*driver, *capturingConnector) {
	t.Helper()

	connector := &capturingConnector{rows: rows}
	database := sql.OpenDB(connector)
	t.Cleanup(func() { _ = database.Close() })

	return &driver{queries: dbpkg.New(database)}, connector
}

// boundArguments runs one write and returns the arguments it bound.
//
// Takes write (func) which performs the write under test.
// Takes value (*string) which is the nullable field value to write.
//
// Returns []sqldriver.Value which holds the arguments of the single statement run.
func boundArguments(t *testing.T, write func(context.Context, *driver, *string) error, value *string) []sqldriver.Value {
	t.Helper()

	taskDriver, connector := newCapturingDriver(t)
	require.NoError(t, write(t.Context(), taskDriver, value))

	statements := connector.recorded()
	require.Len(t, statements, 1, "the write runs a single statement")
	return statements[0]
}

// runNullableWriteCases checks that each case binds nil as NULL and a value as itself.
//
// Takes cases ([]nullableWriteCase) which list the writes to check.
func runNullableWriteCases(t *testing.T, cases []nullableWriteCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			present := boundArguments(t, testCase.write, new(nullableSentinel))
			index := slices.Index(present, sqldriver.Value(nullableSentinel))
			require.NotEqual(t, -1, index, "a set field is bound as its value")
			assert.Equal(t, -1, slices.Index(present[index+1:], sqldriver.Value(nullableSentinel)),
				"the value is bound to a single argument")

			absent := boundArguments(t, testCase.write, nil)
			require.Len(t, absent, len(present))
			assert.Nil(t, absent[index], "a nil field is bound as NULL")
		})
	}
}

// runNullableReadCases checks that each case reads NULL as nil and a value as a pointer
// to that value.
//
// Takes baseRow ([]sqldriver.Value) which is the row the query returns.
// Takes value (T) which is placed in the column under test for the non-NULL check.
// Takes cases ([]nullableReadCase[T]) which list the reads to check.
func runNullableReadCases[T comparable](t *testing.T, baseRow []sqldriver.Value, value T, cases []nullableReadCase[T]) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.name+" is nil when NULL", func(t *testing.T) {
			t.Parallel()

			row := slices.Clone(baseRow)
			row[testCase.column] = nil
			taskDriver, _ := newCapturingDriver(t, row)

			assert.Nil(t, testCase.read(t, taskDriver))
		})

		t.Run(testCase.name+" holds the stored value", func(t *testing.T) {
			t.Parallel()

			row := slices.Clone(baseRow)
			row[testCase.column] = value
			taskDriver, _ := newCapturingDriver(t, row)

			got := testCase.read(t, taskDriver)
			require.NotNil(t, got)
			assert.Equal(t, value, *got)
		})
	}
}

// nullTaskRow returns a whole task row whose nullable columns are all NULL.
//
// Returns []sqldriver.Value which follows the column order of the whole-task queries.
func nullTaskRow() []sqldriver.Value {
	return []sqldriver.Value{
		"task-1", "workflow-1", "test-executor", int64(1), `{}`, `{}`, nil, "PENDING",
		int64(1700000000), int64(0), nil, int64(1700000000), int64(1700000100), nil,
		"build-1", false, int64(0), false, "", int64(0), nil, nil, nil,
	}
}

// readTask reads the single task the fake returns through GetTasksByID.
//
// Takes taskDriver (*driver) which runs the read.
//
// Returns dalcore.TaskRow which is the mapped row.
func readTask(t *testing.T, taskDriver *driver) dalcore.TaskRow {
	t.Helper()

	rows, err := taskDriver.GetTasksByID(t.Context(), []string{"task-1"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

// readRecentTask reads the single row the fake returns through ListRecentTasks.
//
// Takes taskDriver (*driver) which runs the read.
//
// Returns dalcore.RecentTaskRow which is the mapped row.
func readRecentTask(t *testing.T, taskDriver *driver) dalcore.RecentTaskRow {
	t.Helper()

	rows, err := taskDriver.ListRecentTasks(t.Context(), 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

// readWorkflowSummary reads the single row the fake returns through ListWorkflowSummary.
//
// Takes taskDriver (*driver) which runs the read.
//
// Returns dalcore.WorkflowSummaryRow which is the mapped row.
func readWorkflowSummary(t *testing.T, taskDriver *driver) dalcore.WorkflowSummaryRow {
	t.Helper()

	rows, err := taskDriver.ListWorkflowSummary(t.Context(), 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

func TestDriver_TaskWritesBindNullableFields(t *testing.T) {
	t.Parallel()

	runNullableWriteCases(t, []nullableWriteCase{
		{name: "CreateTask deduplication key", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			return taskDriver.CreateTask(ctx, dalcore.CreateTaskParams{ID: "task-1", DeduplicationKey: value})
		}},
		{name: "CreateTask last error", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			return taskDriver.CreateTask(ctx, dalcore.CreateTaskParams{ID: "task-1", LastError: value})
		}},
		{name: "CreateTask result", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			return taskDriver.CreateTask(ctx, dalcore.CreateTaskParams{ID: "task-1", Result: value})
		}},
		{name: "CreateWithDedup deduplication key", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, err := taskDriver.CreateWithDedup(ctx, dalcore.CreateTaskParams{ID: "task-1", DeduplicationKey: value})
			return err
		}},
		{name: "UpdateTask last error", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, err := taskDriver.UpdateTask(ctx, dalcore.UpdateTaskParams{ID: "task-1", LastError: value})
			return err
		}},
		{name: "UpdateTask result", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, err := taskDriver.UpdateTask(ctx, dalcore.UpdateTaskParams{ID: "task-1", Result: value})
			return err
		}},
		{name: "ResolveWorkflowReceipts error message", write: func(ctx context.Context, taskDriver *driver, value *string) error {
			_, err := taskDriver.ResolveWorkflowReceipts(ctx, "workflow-1", value, nullableTimestamp)
			return err
		}},
	})
}

func TestDriver_ResolveWorkflowReceiptsAlwaysBindsResolvedAt(t *testing.T) {
	t.Parallel()

	taskDriver, connector := newCapturingDriver(t)
	_, err := taskDriver.ResolveWorkflowReceipts(t.Context(), "workflow-1", nil, nullableTimestamp)
	require.NoError(t, err)

	statements := connector.recorded()
	require.Len(t, statements, 1)
	assert.Equal(t, []sqldriver.Value{nil, nullableTimestamp, nullableTimestamp, "workflow-1"}, statements[0],
		"a success leaves the error message NULL and still records when it resolved")
}

func TestDriver_TaskReadsMapNullableColumns(t *testing.T) {
	t.Parallel()

	runNullableReadCases(t, nullTaskRow(), nullableSentinel, []nullableReadCase[string]{
		{name: "result", column: 6, read: func(t *testing.T, taskDriver *driver) *string {
			return readTask(t, taskDriver).Result
		}},
		{name: "last error", column: 10, read: func(t *testing.T, taskDriver *driver) *string {
			return readTask(t, taskDriver).LastError
		}},
		{name: "deduplication key", column: 13, read: func(t *testing.T, taskDriver *driver) *string {
			return readTask(t, taskDriver).DeduplicationKey
		}},
		{name: "rerun ID", column: 20, read: func(t *testing.T, taskDriver *driver) *string {
			return readTask(t, taskDriver).RerunID
		}},
		{name: "rerun payload", column: 21, read: func(t *testing.T, taskDriver *driver) *string {
			return readTask(t, taskDriver).RerunPayload
		}},
		{name: "rerun config", column: 22, read: func(t *testing.T, taskDriver *driver) *string {
			return readTask(t, taskDriver).RerunConfig
		}},
	})
}

func TestDriver_ListRecentTasksMapsNullableLastError(t *testing.T) {
	t.Parallel()

	recentRow := []sqldriver.Value{
		"task-1", "workflow-1", "test-executor", "FAILED", int64(1), int64(2),
		nil, int64(1700000000), int64(1700000100),
	}
	runNullableReadCases(t, recentRow, nullableSentinel, []nullableReadCase[string]{
		{name: "last error", column: 6, read: func(t *testing.T, taskDriver *driver) *string {
			return readRecentTask(t, taskDriver).LastError
		}},
	})
}

func TestDriver_ListWorkflowSummaryMapsNullableAggregates(t *testing.T) {
	t.Parallel()

	summaryRow := []sqldriver.Value{"workflow-1", int64(3), nil, nil, nil, nil, nil}
	runNullableReadCases(t, summaryRow, nullableTimestamp, []nullableReadCase[int64]{
		{name: "complete count", column: 2, read: func(t *testing.T, taskDriver *driver) *int64 {
			return readWorkflowSummary(t, taskDriver).CompleteCount
		}},
		{name: "failed count", column: 3, read: func(t *testing.T, taskDriver *driver) *int64 {
			return readWorkflowSummary(t, taskDriver).FailedCount
		}},
		{name: "active count", column: 4, read: func(t *testing.T, taskDriver *driver) *int64 {
			return readWorkflowSummary(t, taskDriver).ActiveCount
		}},
		{name: "created at", column: 5, read: func(t *testing.T, taskDriver *driver) *int64 {
			return readWorkflowSummary(t, taskDriver).CreatedAt
		}},
		{name: "updated at", column: 6, read: func(t *testing.T, taskDriver *driver) *int64 {
			return readWorkflowSummary(t, taskDriver).UpdatedAt
		}},
	})
}
