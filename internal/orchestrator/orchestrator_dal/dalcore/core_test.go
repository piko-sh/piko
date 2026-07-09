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

package dalcore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/clock"
)

type txDoneConnector struct {
	rollbackCalls atomic.Int32
}

func (c *txDoneConnector) Connect(_ context.Context) (driver.Conn, error) {
	return &txDoneConn{connector: c}, nil
}

func (*txDoneConnector) Driver() driver.Driver { return &txDoneDriver{} }

type txDoneDriver struct {
	connector *txDoneConnector
}

func (d *txDoneDriver) Open(_ string) (driver.Conn, error) {
	return &txDoneConn{connector: d.connector}, nil
}

type txDoneConn struct {
	connector *txDoneConnector
}

func (*txDoneConn) Prepare(_ string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (*txDoneConn) Close() error                          { return nil }
func (c *txDoneConn) Begin() (driver.Tx, error) {
	return &txDoneTx{conn: c, committed: false}, nil
}
func (c *txDoneConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return &txDoneTx{conn: c, committed: false}, nil
}

type txDoneTx struct {
	conn      *txDoneConn
	committed bool
}

func (t *txDoneTx) Commit() error {
	t.committed = true
	return nil
}
func (t *txDoneTx) Rollback() error {
	if t.conn != nil && t.conn.connector != nil {
		t.conn.connector.rollbackCalls.Add(1)
	}
	if t.committed {
		return sql.ErrTxDone
	}
	return nil
}

type stubDriver struct{}

func (s stubDriver) WithTx(_ *sql.Tx) Driver { return s }

func (stubDriver) CreateTask(context.Context, CreateTaskParams) error              { return nil }
func (stubDriver) CreateTasksBatch(context.Context, []CreateTaskBatchParams) error { return nil }
func (stubDriver) UpdateTask(context.Context, UpdateTaskParams) error              { return nil }
func (stubDriver) CreateWithDedup(context.Context, CreateTaskParams, string) (bool, error) {
	return false, nil
}
func (stubDriver) FetchDueTasks(context.Context, FetchDueTasksParams) ([]FetchDueTaskRow, error) {
	return nil, nil
}
func (stubDriver) MarkTasksAsProcessing(context.Context, int64, []string) error { return nil }
func (stubDriver) GetWorkflowStatus(context.Context, string) (bool, error)      { return false, nil }
func (stubDriver) PromoteScheduledTasks(context.Context, int64, int64) (int64, error) {
	return 0, nil
}
func (stubDriver) PendingTaskCount(context.Context) (int64, error) { return 0, nil }
func (stubDriver) RecoverStale(context.Context, int32, *string, int64, int64) (int64, error) {
	return 0, nil
}
func (stubDriver) GetStaleProcessingTaskCount(context.Context, int64) (int64, error) { return 0, nil }
func (stubDriver) UpdateTaskHeartbeat(context.Context, int64, string) error          { return nil }
func (stubDriver) GetStaleTasksForRecovery(context.Context, int64, *int64, int) ([]StaleTaskRow, error) {
	return nil, nil
}
func (stubDriver) ClaimTaskForRecovery(context.Context, *string, *int64, string, *int64) (int64, error) {
	return 0, nil
}
func (stubDriver) RecoverClaimed(context.Context, int32, *string, int64, *string) (int64, error) {
	return 0, nil
}
func (stubDriver) ReleaseRecoveryLeases(context.Context, *string) (int64, error) { return 0, nil }
func (stubDriver) CreateWorkflowReceipt(context.Context, CreateWorkflowReceiptParams) error {
	return nil
}
func (stubDriver) ResolveWorkflowReceipts(context.Context, string, *string, int64) (int64, error) {
	return 0, nil
}
func (stubDriver) GetPendingReceiptsByNode(context.Context, string) ([]PendingReceiptRow, error) {
	return nil, nil
}
func (stubDriver) GetPendingReceiptsByWorkflow(context.Context, string) ([]PendingReceiptRow, error) {
	return nil, nil
}
func (stubDriver) CleanupOldResolvedReceipts(context.Context, *int64) (int64, error) { return 0, nil }
func (stubDriver) TimeoutStaleReceipts(context.Context, int64, int64) (int64, error) { return 0, nil }
func (stubDriver) ListFailedTasks(context.Context) ([]FailedTaskRow, error)          { return nil, nil }
func (stubDriver) ListTaskStatusCounts(context.Context) ([]TaskStatusCountRow, error) {
	return nil, nil
}
func (stubDriver) ListRecentTasks(context.Context, int) ([]RecentTaskRow, error) { return nil, nil }
func (stubDriver) ListWorkflowSummary(context.Context, int) ([]WorkflowSummaryRow, error) {
	return nil, nil
}

var (
	_ Driver = stubDriver{}
)

func newStubCore(db *sql.DB) *core {
	return &core{sqlDB: db, driver: stubDriver{}, clock: clock.RealClock()}
}

func TestCore_RollbackGuardedByErrTxDone(t *testing.T) {
	t.Parallel()

	connector := &txDoneConnector{}
	db := sql.OpenDB(connector)
	defer db.Close()

	core := newStubCore(db)

	err := core.withTransaction(t.Context(), func(_ context.Context, _ orchestrator_dal.OrchestratorDAL) error {
		return nil
	})

	require.NoError(t, err, "withTransaction must not surface sql.ErrTxDone after a successful commit")
}

func TestCore_RollbackPropagatesNonErrTxDone(t *testing.T) {
	t.Parallel()

	connector := &txDoneConnector{}
	db := sql.OpenDB(connector)
	defer db.Close()

	core := newStubCore(db)

	sentinel := errors.New("user error")
	err := core.withTransaction(t.Context(), func(_ context.Context, _ orchestrator_dal.OrchestratorDAL) error {
		return sentinel
	})

	require.ErrorIs(t, err, sentinel, "user errors must surface")
	require.GreaterOrEqual(t, connector.rollbackCalls.Load(), int32(1),
		"deferred rollback must call the driver when commit was not reached")
}

type configurableStubDriver struct {
	createTaskFunc              func(context.Context, CreateTaskParams) error
	createTasksBatchFunc        func(context.Context, []CreateTaskBatchParams) error
	updateTaskFunc              func(context.Context, UpdateTaskParams) error
	createWithDedupFunc         func(context.Context, CreateTaskParams, string) (bool, error)
	fetchDueTasksFunc           func(context.Context, FetchDueTasksParams) ([]FetchDueTaskRow, error)
	markProcessingFunc          func(context.Context, int64, []string) error
	getWorkflowStatusFunc       func(context.Context, string) (bool, error)
	promoteFunc                 func(context.Context, int64, int64) (int64, error)
	pendingCountFunc            func(context.Context) (int64, error)
	recoverStaleFunc            func(context.Context, int32, *string, int64, int64) (int64, error)
	staleCountFunc              func(context.Context, int64) (int64, error)
	heartbeatFunc               func(context.Context, int64, string) error
	getStaleTasksFunc           func(context.Context, int64, *int64, int) ([]StaleTaskRow, error)
	claimTaskFunc               func(context.Context, *string, *int64, string, *int64) (int64, error)
	recoverClaimedFunc          func(context.Context, int32, *string, int64, *string) (int64, error)
	releaseLeasesFunc           func(context.Context, *string) (int64, error)
	createReceiptFunc           func(context.Context, CreateWorkflowReceiptParams) error
	resolveReceiptsFunc         func(context.Context, string, *string, int64) (int64, error)
	pendingByNodeFunc           func(context.Context, string) ([]PendingReceiptRow, error)
	pendingByWorkflowFunc       func(context.Context, string) ([]PendingReceiptRow, error)
	cleanupReceiptsFunc         func(context.Context, *int64) (int64, error)
	timeoutReceiptsFunc         func(context.Context, int64, int64) (int64, error)
	listFailedFunc              func(context.Context) ([]FailedTaskRow, error)
	listStatusCountsFunc        func(context.Context) ([]TaskStatusCountRow, error)
	listRecentFunc              func(context.Context, int) ([]RecentTaskRow, error)
	listWorkflowSummaryFunc     func(context.Context, int) ([]WorkflowSummaryRow, error)
	markProcessingCalledWithIDs []string
}

func (d *configurableStubDriver) WithTx(*sql.Tx) Driver { return d }

func (d *configurableStubDriver) CreateTask(ctx context.Context, params CreateTaskParams) error {
	if d.createTaskFunc != nil {
		return d.createTaskFunc(ctx, params)
	}
	return nil
}

func (d *configurableStubDriver) CreateTasksBatch(ctx context.Context, params []CreateTaskBatchParams) error {
	if d.createTasksBatchFunc != nil {
		return d.createTasksBatchFunc(ctx, params)
	}
	return nil
}

func (d *configurableStubDriver) UpdateTask(ctx context.Context, params UpdateTaskParams) error {
	if d.updateTaskFunc != nil {
		return d.updateTaskFunc(ctx, params)
	}
	return nil
}

func (d *configurableStubDriver) CreateWithDedup(ctx context.Context, params CreateTaskParams, key string) (bool, error) {
	if d.createWithDedupFunc != nil {
		return d.createWithDedupFunc(ctx, params, key)
	}
	return false, nil
}

func (d *configurableStubDriver) FetchDueTasks(ctx context.Context, params FetchDueTasksParams) ([]FetchDueTaskRow, error) {
	if d.fetchDueTasksFunc != nil {
		return d.fetchDueTasksFunc(ctx, params)
	}
	return nil, nil
}

func (d *configurableStubDriver) MarkTasksAsProcessing(ctx context.Context, updatedAt int64, ids []string) error {
	d.markProcessingCalledWithIDs = ids
	if d.markProcessingFunc != nil {
		return d.markProcessingFunc(ctx, updatedAt, ids)
	}
	return nil
}

func (d *configurableStubDriver) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	if d.getWorkflowStatusFunc != nil {
		return d.getWorkflowStatusFunc(ctx, workflowID)
	}
	return false, nil
}

func (d *configurableStubDriver) PromoteScheduledTasks(ctx context.Context, updatedAt, executeAt int64) (int64, error) {
	if d.promoteFunc != nil {
		return d.promoteFunc(ctx, updatedAt, executeAt)
	}
	return 0, nil
}

func (d *configurableStubDriver) PendingTaskCount(ctx context.Context) (int64, error) {
	if d.pendingCountFunc != nil {
		return d.pendingCountFunc(ctx)
	}
	return 0, nil
}

func (d *configurableStubDriver) RecoverStale(ctx context.Context, maxRetries int32, recoveryError *string, nowUnix, staleThresholdUnix int64) (int64, error) {
	if d.recoverStaleFunc != nil {
		return d.recoverStaleFunc(ctx, maxRetries, recoveryError, nowUnix, staleThresholdUnix)
	}
	return 0, nil
}

func (d *configurableStubDriver) GetStaleProcessingTaskCount(ctx context.Context, staleThresholdUnix int64) (int64, error) {
	if d.staleCountFunc != nil {
		return d.staleCountFunc(ctx, staleThresholdUnix)
	}
	return 0, nil
}

func (d *configurableStubDriver) UpdateTaskHeartbeat(ctx context.Context, updatedAt int64, taskID string) error {
	if d.heartbeatFunc != nil {
		return d.heartbeatFunc(ctx, updatedAt, taskID)
	}
	return nil
}

func (d *configurableStubDriver) GetStaleTasksForRecovery(ctx context.Context, staleThresholdUnix int64, recoveryExpiresAt *int64, limit int) ([]StaleTaskRow, error) {
	if d.getStaleTasksFunc != nil {
		return d.getStaleTasksFunc(ctx, staleThresholdUnix, recoveryExpiresAt, limit)
	}
	return nil, nil
}

func (d *configurableStubDriver) ClaimTaskForRecovery(ctx context.Context, nodeID *string, leaseExpiresAt *int64, taskID string, nowUnix *int64) (int64, error) {
	if d.claimTaskFunc != nil {
		return d.claimTaskFunc(ctx, nodeID, leaseExpiresAt, taskID, nowUnix)
	}
	return 0, nil
}

func (d *configurableStubDriver) RecoverClaimed(ctx context.Context, maxRetries int32, recoveryError *string, nowUnix int64, nodeID *string) (int64, error) {
	if d.recoverClaimedFunc != nil {
		return d.recoverClaimedFunc(ctx, maxRetries, recoveryError, nowUnix, nodeID)
	}
	return 0, nil
}

func (d *configurableStubDriver) ReleaseRecoveryLeases(ctx context.Context, nodeID *string) (int64, error) {
	if d.releaseLeasesFunc != nil {
		return d.releaseLeasesFunc(ctx, nodeID)
	}
	return 0, nil
}

func (d *configurableStubDriver) CreateWorkflowReceipt(ctx context.Context, params CreateWorkflowReceiptParams) error {
	if d.createReceiptFunc != nil {
		return d.createReceiptFunc(ctx, params)
	}
	return nil
}

func (d *configurableStubDriver) ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage *string, nowUnix int64) (int64, error) {
	if d.resolveReceiptsFunc != nil {
		return d.resolveReceiptsFunc(ctx, workflowID, errorMessage, nowUnix)
	}
	return 0, nil
}

func (d *configurableStubDriver) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]PendingReceiptRow, error) {
	if d.pendingByNodeFunc != nil {
		return d.pendingByNodeFunc(ctx, nodeID)
	}
	return nil, nil
}

func (d *configurableStubDriver) GetPendingReceiptsByWorkflow(ctx context.Context, workflowID string) ([]PendingReceiptRow, error) {
	if d.pendingByWorkflowFunc != nil {
		return d.pendingByWorkflowFunc(ctx, workflowID)
	}
	return nil, nil
}

func (d *configurableStubDriver) CleanupOldResolvedReceipts(ctx context.Context, olderThanUnix *int64) (int64, error) {
	if d.cleanupReceiptsFunc != nil {
		return d.cleanupReceiptsFunc(ctx, olderThanUnix)
	}
	return 0, nil
}

func (d *configurableStubDriver) TimeoutStaleReceipts(ctx context.Context, updatedAt, olderThanUnix int64) (int64, error) {
	if d.timeoutReceiptsFunc != nil {
		return d.timeoutReceiptsFunc(ctx, updatedAt, olderThanUnix)
	}
	return 0, nil
}

func (d *configurableStubDriver) ListFailedTasks(ctx context.Context) ([]FailedTaskRow, error) {
	if d.listFailedFunc != nil {
		return d.listFailedFunc(ctx)
	}
	return nil, nil
}

func (d *configurableStubDriver) ListTaskStatusCounts(ctx context.Context) ([]TaskStatusCountRow, error) {
	if d.listStatusCountsFunc != nil {
		return d.listStatusCountsFunc(ctx)
	}
	return nil, nil
}

func (d *configurableStubDriver) ListRecentTasks(ctx context.Context, limit int) ([]RecentTaskRow, error) {
	if d.listRecentFunc != nil {
		return d.listRecentFunc(ctx, limit)
	}
	return nil, nil
}

func (d *configurableStubDriver) ListWorkflowSummary(ctx context.Context, limit int) ([]WorkflowSummaryRow, error) {
	if d.listWorkflowSummaryFunc != nil {
		return d.listWorkflowSummaryFunc(ctx, limit)
	}
	return nil, nil
}

var (
	_ Driver = (*configurableStubDriver)(nil)
)

func newTxCore(t *testing.T, stub *configurableStubDriver) *core {
	t.Helper()
	db := sql.OpenDB(&txDoneConnector{})
	t.Cleanup(func() { _ = db.Close() })
	return &core{sqlDB: db, driver: stub, clock: clock.RealClock()}
}

func sampleTask() *orchestrator_domain.Task {
	return &orchestrator_domain.Task{
		ID:         "task-1",
		WorkflowID: "wf-1",
		Executor:   "greeter",
		Status:     orchestrator_domain.StatusPending,
		Payload:    map[string]any{"name": "ada"},
		Config:     orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityNormal, MaxRetries: 3},
		ExecuteAt:  time.Unix(1000, 0).UTC(),
		CreatedAt:  time.Unix(900, 0).UTC(),
		Attempt:    1,
	}
}

func TestNew_ReturnsUsableCore(t *testing.T) {
	t.Parallel()

	dal := New(nil, stubDriver{})
	require.NotNil(t, dal, "New must return a non-nil DAL")
	require.NoError(t, dal.HealthCheck(t.Context()), "HealthCheck with nil sqlDB must be a no-op")
	require.NoError(t, dal.Close(), "Close must be a no-op")
}

func TestCore_HealthCheckWithDB(t *testing.T) {
	t.Parallel()

	dal := newTxCore(t, &configurableStubDriver{})
	require.NoError(t, dal.HealthCheck(t.Context()), "HealthCheck should ping the fake DB without error")
}

func TestDerefInt64(t *testing.T) {
	t.Parallel()

	require.Equal(t, int64(0), derefInt64(nil), "nil pointer must default to zero")
	value := int64(42)
	require.Equal(t, int64(42), derefInt64(&value), "non-nil pointer must return its value")
}

func TestBuildCreateTaskParams(t *testing.T) {
	t.Parallel()

	task := sampleTask()
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	params, err := buildCreateTaskParams(task, now)
	require.NoError(t, err, "valid task must build params")
	require.Equal(t, "task-1", params.ID, "ID must be copied")
	require.Equal(t, "wf-1", params.WorkflowID, "WorkflowID must be copied")
	require.Equal(t, int64(1000), params.ExecuteAt, "ExecuteAt must be Unix seconds")
	require.Equal(t, int64(900), params.CreatedAt, "CreatedAt must be Unix seconds")
	require.Equal(t, int32(1), params.Attempt, "Attempt must be copied")
	require.JSONEq(t, `{"name":"ada"}`, params.Payload, "Payload must be JSON-marshalled")
	require.Equal(t, now.Unix(), params.UpdatedAt, "UpdatedAt must equal the injected now")
}

func TestBuildCreateTaskParams_MarshalErrors(t *testing.T) {
	t.Parallel()

	payloadFailure := sampleTask()
	payloadFailure.Payload = map[string]any{"bad": make(chan int)}
	_, err := buildCreateTaskParams(payloadFailure, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	require.Error(t, err, "unmarshalable payload must fail")
	require.Contains(t, err.Error(), "marshal task payload", "error must identify the payload stage")
}

func TestBuildUpdateTaskParams_NilResultAndError(t *testing.T) {
	t.Parallel()

	task := sampleTask()
	task.Status = orchestrator_domain.StatusPending
	task.Result = nil
	task.LastError = ""

	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	params, err := buildUpdateTaskParams(task, now)
	require.NoError(t, err, "valid task must build params")
	require.Nil(t, params.Result, "nil Result must produce a nil Result pointer")
	require.Nil(t, params.LastError, "empty LastError must produce a nil pointer")
	require.Equal(t, "task-1", params.ID, "ID must be copied")
	require.Equal(t, now.Unix(), params.UpdatedAt, "UpdatedAt must equal the injected now")
}

func TestBuildUpdateTaskParams_SetResultAndError(t *testing.T) {
	t.Parallel()

	task := sampleTask()
	task.Result = map[string]any{"ok": true}
	task.LastError = "boom"

	params, err := buildUpdateTaskParams(task, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	require.NoError(t, err, "valid task must build params")
	require.NotNil(t, params.Result, "non-nil Result must produce a non-nil Result pointer")
	require.JSONEq(t, `{"ok":true}`, *params.Result, "Result must be JSON-marshalled")
	require.NotNil(t, params.LastError, "non-empty LastError must produce a non-nil pointer")
	require.Equal(t, "boom", *params.LastError, "LastError must be copied through the pointer")
}

func TestBuildUpdateTaskParams_MarshalError(t *testing.T) {
	t.Parallel()

	task := sampleTask()
	task.Result = map[string]any{"bad": make(chan int)}
	_, err := buildUpdateTaskParams(task, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	require.Error(t, err, "unmarshalable result must fail")
	require.Contains(t, err.Error(), "marshal task result", "error must identify the result stage")
}

func TestConvertDBTaskToDomain_Success(t *testing.T) {
	t.Parallel()

	result := `{"ok":true}`
	lastError := "prior failure"
	dedup := "dedup-1"
	row := &FetchDueTaskRow{
		ID:               "task-1",
		WorkflowID:       "wf-1",
		Executor:         "greeter",
		Status:           string(orchestrator_domain.StatusRetrying),
		Payload:          `{"name":"ada"}`,
		Config:           `{"max_retries":2}`,
		Priority:         int32(orchestrator_domain.PriorityNormal),
		ExecuteAt:        1000,
		CreatedAt:        900,
		UpdatedAt:        950,
		Attempt:          2,
		Result:           &result,
		LastError:        &lastError,
		DeduplicationKey: &dedup,
	}

	task, err := convertDBTaskToDomain(row)
	require.NoError(t, err, "valid row must convert")
	defer orchestrator_domain.TaskPool.Put(task)

	require.Equal(t, "task-1", task.ID, "ID must be copied")
	require.Equal(t, orchestrator_domain.StatusRetrying, task.Status, "Status must be copied")
	require.Equal(t, "prior failure", task.LastError, "LastError must be dereferenced")
	require.Equal(t, "dedup-1", task.DeduplicationKey, "DeduplicationKey must be dereferenced")
	require.Equal(t, "ada", task.Payload["name"], "Payload must be unmarshalled")
	require.Equal(t, orchestrator_domain.PriorityNormal, task.Config.Priority, "Priority must override config")
	require.Equal(t, true, task.Result["ok"], "Result must be unmarshalled")
}

func TestConvertDBTaskToDomain_UnmarshalErrors(t *testing.T) {
	t.Parallel()

	badResult := `{invalid`
	testCases := []struct {
		name   string
		expect string
		row    FetchDueTaskRow
	}{
		{
			name:   "payload",
			row:    FetchDueTaskRow{ID: "task-1", Payload: `{invalid`, Config: `{}`},
			expect: "unmarshal payload",
		},
		{
			name:   "config",
			row:    FetchDueTaskRow{ID: "task-1", Payload: `{}`, Config: `{invalid`},
			expect: "unmarshal config",
		},
		{
			name:   "result",
			row:    FetchDueTaskRow{ID: "task-1", Payload: `{}`, Config: `{}`, Result: &badResult},
			expect: "unmarshal result",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			row := testCase.row
			task, err := convertDBTaskToDomain(&row)
			require.Error(t, err, "malformed JSON must fail")
			require.Nil(t, task, "failed conversion must return a nil task")
			require.Contains(t, err.Error(), testCase.expect, "error must identify the failing field")
		})
	}
}

func TestConvertFetchedRowsToDomain_ErrorFreesPool(t *testing.T) {
	t.Parallel()

	rows := []FetchDueTaskRow{
		{ID: "good", Payload: `{}`, Config: `{}`},
		{ID: "bad", Payload: `{invalid`, Config: `{}`},
	}

	ids, tasks, err := convertFetchedRowsToDomain(rows)
	require.Error(t, err, "a malformed row must fail the batch")
	require.Nil(t, ids, "failed batch must return nil ids")
	require.Nil(t, tasks, "failed batch must return nil tasks")
	require.Contains(t, err.Error(), "bad", "error must reference the offending task ID")
}

func TestCore_GetWorkflowStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		hasIncomplete bool
		expected      bool
	}{
		{name: "complete", hasIncomplete: false, expected: true},
		{name: "incomplete", hasIncomplete: true, expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			stub := &configurableStubDriver{getWorkflowStatusFunc: func(context.Context, string) (bool, error) {
				return testCase.hasIncomplete, nil
			}}
			dal := &core{driver: stub, clock: clock.RealClock()}
			complete, err := dal.GetWorkflowStatus(t.Context(), "wf-1")
			require.NoError(t, err, "status query must succeed")
			require.Equal(t, testCase.expected, complete, "completeness is the negation of hasIncomplete")
		})
	}
}

func TestCore_GetWorkflowStatus_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("db down")
	stub := &configurableStubDriver{getWorkflowStatusFunc: func(context.Context, string) (bool, error) {
		return false, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.GetWorkflowStatus(t.Context(), "wf-1")
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_PendingAndStaleCounts(t *testing.T) {
	t.Parallel()

	stub := &configurableStubDriver{
		pendingCountFunc: func(context.Context) (int64, error) { return 7, nil },
		staleCountFunc:   func(context.Context, int64) (int64, error) { return 3, nil },
	}
	dal := &core{driver: stub, clock: clock.RealClock()}

	pending, err := dal.PendingTaskCount(t.Context())
	require.NoError(t, err, "pending count must succeed")
	require.Equal(t, int64(7), pending, "pending count must be returned")

	stale, err := dal.GetStaleProcessingTaskCount(t.Context(), time.Minute)
	require.NoError(t, err, "stale count must succeed")
	require.Equal(t, int64(3), stale, "stale count must be returned")
}

func TestCore_CountErrorsWrapped(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("query failed")
	stub := &configurableStubDriver{
		pendingCountFunc: func(context.Context) (int64, error) { return 0, sentinel },
		staleCountFunc:   func(context.Context, int64) (int64, error) { return 0, sentinel },
	}
	dal := &core{driver: stub, clock: clock.RealClock()}

	_, pendingErr := dal.PendingTaskCount(t.Context())
	require.ErrorIs(t, pendingErr, sentinel, "pending count error must be wrapped")

	_, staleErr := dal.GetStaleProcessingTaskCount(t.Context(), time.Minute)
	require.ErrorIs(t, staleErr, sentinel, "stale count error must be wrapped")
}

func TestCore_ListFailedTasks(t *testing.T) {
	t.Parallel()

	lastError := "kaboom"
	stub := &configurableStubDriver{listFailedFunc: func(context.Context) ([]FailedTaskRow, error) {
		return []FailedTaskRow{
			{ID: "t1", WorkflowID: "wf", Executor: "e", Attempt: 4, LastError: &lastError},
			{ID: "t2", WorkflowID: "wf", Executor: "e", Attempt: 0, LastError: nil},
		}, nil
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}

	tasks, err := dal.ListFailedTasks(t.Context())
	require.NoError(t, err, "listing failed tasks must succeed")
	require.Len(t, tasks, 2, "both rows must be mapped")
	require.Equal(t, orchestrator_domain.StatusFailed, tasks[0].Status, "status must be forced to FAILED")
	require.Equal(t, "kaboom", tasks[0].LastError, "non-nil LastError must be dereferenced")
	require.Equal(t, 4, tasks[0].Attempt, "attempt must be copied")
	require.Empty(t, tasks[1].LastError, "nil LastError must map to an empty string")
}

func TestCore_ListFailedTasks_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	stub := &configurableStubDriver{listFailedFunc: func(context.Context) ([]FailedTaskRow, error) {
		return nil, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.ListFailedTasks(t.Context())
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_ListTaskSummary(t *testing.T) {
	t.Parallel()

	stub := &configurableStubDriver{listStatusCountsFunc: func(context.Context) ([]TaskStatusCountRow, error) {
		return []TaskStatusCountRow{{Status: "PENDING", TaskCount: 5}, {Status: "FAILED", TaskCount: 2}}, nil
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}

	summary, err := dal.ListTaskSummary(t.Context())
	require.NoError(t, err, "task summary must succeed")
	require.Len(t, summary, 2, "each status row must be mapped")
	require.Equal(t, "PENDING", summary[0].Status, "status must be copied")
	require.Equal(t, int64(5), summary[0].Count, "count must be copied")
}

func TestCore_ListTaskSummary_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	stub := &configurableStubDriver{listStatusCountsFunc: func(context.Context) ([]TaskStatusCountRow, error) {
		return nil, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.ListTaskSummary(t.Context())
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_ListRecentTasks(t *testing.T) {
	t.Parallel()

	lastError := "nope"
	stub := &configurableStubDriver{listRecentFunc: func(_ context.Context, limit int) ([]RecentTaskRow, error) {
		require.Equal(t, 10, limit, "limit must be forwarded")
		return []RecentTaskRow{{
			ID: "t1", WorkflowID: "wf", Executor: "e", Status: "PENDING",
			Priority: 2, Attempt: 1, LastError: &lastError, CreatedAt: 100, UpdatedAt: 200,
		}}, nil
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}

	items, err := dal.ListRecentTasks(t.Context(), 10)
	require.NoError(t, err, "recent tasks must succeed")
	require.Len(t, items, 1, "the single row must be mapped")
	require.Equal(t, "t1", items[0].ID, "ID must be copied")
	require.Equal(t, &lastError, items[0].LastError, "LastError pointer must be carried through")
	require.Equal(t, int64(200), items[0].UpdatedAt, "UpdatedAt must be copied")
}

func TestCore_ListRecentTasks_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	stub := &configurableStubDriver{listRecentFunc: func(context.Context, int) ([]RecentTaskRow, error) {
		return nil, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.ListRecentTasks(t.Context(), 5)
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_ListWorkflowSummary(t *testing.T) {
	t.Parallel()

	complete := int64(4)
	created := int64(111)
	stub := &configurableStubDriver{listWorkflowSummaryFunc: func(context.Context, int) ([]WorkflowSummaryRow, error) {
		return []WorkflowSummaryRow{{
			WorkflowID:    "wf",
			TaskCount:     10,
			CompleteCount: &complete,
			FailedCount:   nil,
			ActiveCount:   nil,
			CreatedAt:     &created,
			UpdatedAt:     nil,
		}}, nil
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}

	summary, err := dal.ListWorkflowSummary(t.Context(), 20)
	require.NoError(t, err, "workflow summary must succeed")
	require.Len(t, summary, 1, "the single row must be mapped")
	require.Equal(t, int64(10), summary[0].TaskCount, "TaskCount must be copied")
	require.Equal(t, int64(4), summary[0].CompleteCount, "non-nil CompleteCount must be dereferenced")
	require.Equal(t, int64(0), summary[0].FailedCount, "nil FailedCount must default to zero")
	require.Equal(t, int64(0), summary[0].ActiveCount, "nil ActiveCount must default to zero")
	require.Equal(t, int64(111), summary[0].CreatedAt, "non-nil CreatedAt must be dereferenced")
	require.Equal(t, int64(0), summary[0].UpdatedAt, "nil UpdatedAt must default to zero")
}

func TestCore_ListWorkflowSummary_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	stub := &configurableStubDriver{listWorkflowSummaryFunc: func(context.Context, int) ([]WorkflowSummaryRow, error) {
		return nil, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.ListWorkflowSummary(t.Context(), 5)
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_GetPendingReceiptsByNode(t *testing.T) {
	t.Parallel()

	stub := &configurableStubDriver{pendingByNodeFunc: func(_ context.Context, nodeID string) ([]PendingReceiptRow, error) {
		require.Equal(t, "node-a", nodeID, "node ID must be forwarded")
		return []PendingReceiptRow{{ID: "r1", WorkflowID: "wf", CreatedAt: 500}}, nil
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}

	receipts, err := dal.GetPendingReceiptsByNode(t.Context(), "node-a")
	require.NoError(t, err, "receipts by node must succeed")
	require.Len(t, receipts, 1, "the single row must be mapped")
	require.Equal(t, "node-a", receipts[0].NodeID, "NodeID must be filled from the query argument")
	require.Equal(t, int64(500), receipts[0].CreatedAt, "CreatedAt must be copied")
}

func TestCore_GetPendingReceiptsByNode_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	stub := &configurableStubDriver{pendingByNodeFunc: func(context.Context, string) ([]PendingReceiptRow, error) {
		return nil, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.GetPendingReceiptsByNode(t.Context(), "node-a")
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_GetPendingReceiptsByWorkflow(t *testing.T) {
	t.Parallel()

	stub := &configurableStubDriver{pendingByWorkflowFunc: func(context.Context, string) ([]PendingReceiptRow, error) {
		return []PendingReceiptRow{{ID: "r1", WorkflowID: "wf", NodeID: "node-b", CreatedAt: 600}}, nil
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}

	receipts, err := dal.GetPendingReceiptsByWorkflow(t.Context(), "wf")
	require.NoError(t, err, "receipts by workflow must succeed")
	require.Len(t, receipts, 1, "the single row must be mapped")
	require.Equal(t, "node-b", receipts[0].NodeID, "NodeID must be copied from the row")
}

func TestCore_GetPendingReceiptsByWorkflow_Error(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	stub := &configurableStubDriver{pendingByWorkflowFunc: func(context.Context, string) ([]PendingReceiptRow, error) {
		return nil, sentinel
	}}
	dal := &core{driver: stub, clock: clock.RealClock()}
	_, err := dal.GetPendingReceiptsByWorkflow(t.Context(), "wf")
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
}

func TestCore_CreateTasks(t *testing.T) {
	t.Parallel()

	t.Run("empty batch is a no-op", func(t *testing.T) {
		t.Parallel()
		dal := newTxCore(t, &configurableStubDriver{})
		require.NoError(t, dal.CreateTasks(t.Context(), nil), "an empty batch must succeed without a transaction")
	})

	t.Run("batch is built and forwarded", func(t *testing.T) {
		t.Parallel()
		var seen []CreateTaskBatchParams
		stub := &configurableStubDriver{createTasksBatchFunc: func(_ context.Context, params []CreateTaskBatchParams) error {
			seen = params
			return nil
		}}
		dal := newTxCore(t, stub)
		dedupTask := sampleTask()
		dedupTask.DeduplicationKey = "dk"
		require.NoError(t, dal.CreateTasks(t.Context(), []*orchestrator_domain.Task{sampleTask(), dedupTask}), "a batch must be inserted")
		require.Len(t, seen, 2, "both tasks must be in the batch")
		require.Nil(t, seen[0].DeduplicationKey, "an empty key must map to a nil pointer")
		require.NotNil(t, seen[1].DeduplicationKey, "a set key must map to a non-nil pointer")
		require.Equal(t, "dk", *seen[1].DeduplicationKey, "the deduplication key must be carried through")
	})

	t.Run("marshal error is wrapped", func(t *testing.T) {
		t.Parallel()
		dal := newTxCore(t, &configurableStubDriver{})
		task := sampleTask()
		task.Payload = map[string]any{"bad": make(chan int)}
		err := dal.CreateTasks(t.Context(), []*orchestrator_domain.Task{task})
		require.Error(t, err, "an unmarshalable payload must fail the batch")
		require.Contains(t, err.Error(), "marshal payload", "error must identify the payload stage")
	})
}

func TestCore_CreateTask(t *testing.T) {
	t.Parallel()

	var seen CreateTaskParams
	stub := &configurableStubDriver{createTaskFunc: func(_ context.Context, params CreateTaskParams) error {
		seen = params
		return nil
	}}
	dal := newTxCore(t, stub)

	require.NoError(t, dal.CreateTask(t.Context(), sampleTask()), "create task must succeed")
	require.Equal(t, "task-1", seen.ID, "driver must receive the built params")
}

func TestCore_CreateTask_DriverErrorWrapped(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("insert failed")
	stub := &configurableStubDriver{createTaskFunc: func(context.Context, CreateTaskParams) error {
		return sentinel
	}}
	dal := newTxCore(t, stub)

	err := dal.CreateTask(t.Context(), sampleTask())
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped with %w")
	require.Contains(t, err.Error(), "executing create task query", "error must describe the stage")
}

func TestCore_CreateTask_BuildParamsError(t *testing.T) {
	t.Parallel()

	dal := newTxCore(t, &configurableStubDriver{})
	task := sampleTask()
	task.Payload = map[string]any{"bad": make(chan int)}

	err := dal.CreateTask(t.Context(), task)
	require.Error(t, err, "unmarshalable payload must fail before the transaction")
	require.Contains(t, err.Error(), "building create task params", "error must describe the build stage")
}

func TestCore_UpdateTask_DriverErrorWrapped(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("update failed")
	stub := &configurableStubDriver{updateTaskFunc: func(context.Context, UpdateTaskParams) error {
		return sentinel
	}}
	dal := newTxCore(t, stub)

	err := dal.UpdateTask(t.Context(), sampleTask())
	require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
	require.Contains(t, err.Error(), "executing update task query", "error must describe the stage")
}

func TestCore_CreateTaskWithDedup(t *testing.T) {
	t.Parallel()

	t.Run("empty key delegates to CreateTask", func(t *testing.T) {
		t.Parallel()
		var createCalled bool
		stub := &configurableStubDriver{
			createTaskFunc: func(context.Context, CreateTaskParams) error {
				createCalled = true
				return nil
			},
			createWithDedupFunc: func(context.Context, CreateTaskParams, string) (bool, error) {
				t.Fatal("CreateWithDedup must not be called when DeduplicationKey is empty")
				return false, nil
			},
		}
		dal := newTxCore(t, stub)
		task := sampleTask()
		task.DeduplicationKey = ""
		require.NoError(t, dal.CreateTaskWithDedup(t.Context(), task), "empty key must delegate to CreateTask")
		require.True(t, createCalled, "CreateTask must be invoked for an empty key")
	})

	t.Run("created returns nil", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{createWithDedupFunc: func(context.Context, CreateTaskParams, string) (bool, error) {
			return true, nil
		}}
		dal := newTxCore(t, stub)
		task := sampleTask()
		task.DeduplicationKey = "key-1"
		require.NoError(t, dal.CreateTaskWithDedup(t.Context(), task), "a created task must return nil")
	})

	t.Run("not created returns ErrDuplicateTask", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{createWithDedupFunc: func(context.Context, CreateTaskParams, string) (bool, error) {
			return false, nil
		}}
		dal := newTxCore(t, stub)
		task := sampleTask()
		task.DeduplicationKey = "key-1"
		err := dal.CreateTaskWithDedup(t.Context(), task)
		require.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask, "an existing duplicate must return ErrDuplicateTask")
	})

	t.Run("driver error wrapped", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("dedup failed")
		stub := &configurableStubDriver{createWithDedupFunc: func(context.Context, CreateTaskParams, string) (bool, error) {
			return false, sentinel
		}}
		dal := newTxCore(t, stub)
		task := sampleTask()
		task.DeduplicationKey = "key-1"
		err := dal.CreateTaskWithDedup(t.Context(), task)
		require.ErrorIs(t, err, sentinel, "driver error must be wrapped")
	})
}

func TestCore_FetchAndMarkDueTasks(t *testing.T) {
	t.Parallel()

	t.Run("empty returns empty slice", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{fetchDueTasksFunc: func(context.Context, FetchDueTasksParams) ([]FetchDueTaskRow, error) {
			return nil, nil
		}}
		dal := newTxCore(t, stub)
		tasks, err := dal.FetchAndMarkDueTasks(t.Context(), orchestrator_domain.PriorityNormal, 10)
		require.NoError(t, err, "an empty fetch must succeed")
		require.Empty(t, tasks, "no rows must yield an empty slice")
	})

	t.Run("rows are converted and marked", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{fetchDueTasksFunc: func(context.Context, FetchDueTasksParams) ([]FetchDueTaskRow, error) {
			return []FetchDueTaskRow{
				{ID: "t1", WorkflowID: "wf", Executor: "e", Status: "PENDING", Payload: `{}`, Config: `{}`},
				{ID: "t2", WorkflowID: "wf", Executor: "e", Status: "PENDING", Payload: `{}`, Config: `{}`},
			}, nil
		}}
		dal := newTxCore(t, stub)
		tasks, err := dal.FetchAndMarkDueTasks(t.Context(), orchestrator_domain.PriorityNormal, 10)
		require.NoError(t, err, "conversion and marking must succeed")
		require.Len(t, tasks, 2, "both rows must be converted")
		require.Equal(t, []string{"t1", "t2"}, stub.markProcessingCalledWithIDs, "the fetched IDs must be marked as processing")
		for _, task := range tasks {
			orchestrator_domain.TaskPool.Put(task)
		}
	})

	t.Run("convert error returns error", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{fetchDueTasksFunc: func(context.Context, FetchDueTasksParams) ([]FetchDueTaskRow, error) {
			return []FetchDueTaskRow{
				{ID: "good", Payload: `{}`, Config: `{}`},
				{ID: "bad", Payload: `{invalid`, Config: `{}`},
			}, nil
		}}
		dal := newTxCore(t, stub)
		tasks, err := dal.FetchAndMarkDueTasks(t.Context(), orchestrator_domain.PriorityNormal, 10)
		require.Error(t, err, "a malformed row must fail the fetch")
		require.Nil(t, tasks, "a failed fetch must return no tasks")
		require.Nil(t, stub.markProcessingCalledWithIDs, "marking must not run when conversion fails")
	})

	t.Run("fetch error wrapped", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("fetch failed")
		stub := &configurableStubDriver{fetchDueTasksFunc: func(context.Context, FetchDueTasksParams) ([]FetchDueTaskRow, error) {
			return nil, sentinel
		}}
		dal := newTxCore(t, stub)
		_, err := dal.FetchAndMarkDueTasks(t.Context(), orchestrator_domain.PriorityNormal, 10)
		require.ErrorIs(t, err, sentinel, "fetch error must be wrapped")
	})
}

func TestCore_ClaimStaleTasksForRecovery(t *testing.T) {
	t.Parallel()

	t.Run("only claimed rows are returned", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{
			getStaleTasksFunc: func(context.Context, int64, *int64, int) ([]StaleTaskRow, error) {
				return []StaleTaskRow{
					{ID: "t1", WorkflowID: "wf", Attempt: 1},
					{ID: "t2", WorkflowID: "wf", Attempt: 2},
					{ID: "t3", WorkflowID: "wf", Attempt: 3},
				}, nil
			},
			claimTaskFunc: func(_ context.Context, _ *string, _ *int64, taskID string, _ *int64) (int64, error) {
				if taskID == "t2" {
					return 0, nil
				}
				return 1, nil
			},
		}
		dal := newTxCore(t, stub)
		claimed, err := dal.ClaimStaleTasksForRecovery(t.Context(), "node-a", time.Minute, time.Minute, 5)
		require.NoError(t, err, "claiming must succeed")
		require.Len(t, claimed, 2, "only rows whose claim affected rows must be returned")
		require.Equal(t, "t1", claimed[0].ID, "first claimed task must be t1")
		require.Equal(t, "t3", claimed[1].ID, "unclaimed t2 must be skipped")
	})

	t.Run("no candidates returns empty", func(t *testing.T) {
		t.Parallel()
		stub := &configurableStubDriver{getStaleTasksFunc: func(context.Context, int64, *int64, int) ([]StaleTaskRow, error) {
			return nil, nil
		}}
		dal := newTxCore(t, stub)
		claimed, err := dal.ClaimStaleTasksForRecovery(t.Context(), "node-a", time.Minute, time.Minute, 5)
		require.NoError(t, err, "no candidates must succeed")
		require.Empty(t, claimed, "no candidates must yield no claims")
	})

	t.Run("get candidates error wrapped", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("candidates failed")
		stub := &configurableStubDriver{getStaleTasksFunc: func(context.Context, int64, *int64, int) ([]StaleTaskRow, error) {
			return nil, sentinel
		}}
		dal := newTxCore(t, stub)
		_, err := dal.ClaimStaleTasksForRecovery(t.Context(), "node-a", time.Minute, time.Minute, 5)
		require.ErrorIs(t, err, sentinel, "candidate error must be wrapped")
	})

	t.Run("claim error wrapped", func(t *testing.T) {
		t.Parallel()
		sentinel := errors.New("claim failed")
		stub := &configurableStubDriver{
			getStaleTasksFunc: func(context.Context, int64, *int64, int) ([]StaleTaskRow, error) {
				return []StaleTaskRow{{ID: "t1"}}, nil
			},
			claimTaskFunc: func(context.Context, *string, *int64, string, *int64) (int64, error) {
				return 0, sentinel
			},
		}
		dal := newTxCore(t, stub)
		_, err := dal.ClaimStaleTasksForRecovery(t.Context(), "node-a", time.Minute, time.Minute, 5)
		require.ErrorIs(t, err, sentinel, "claim error must be wrapped")
	})
}

func TestCore_TransactionCountMethods(t *testing.T) {
	t.Parallel()

	stub := &configurableStubDriver{
		promoteFunc:         func(context.Context, int64, int64) (int64, error) { return 2, nil },
		recoverStaleFunc:    func(context.Context, int32, *string, int64, int64) (int64, error) { return 3, nil },
		recoverClaimedFunc:  func(context.Context, int32, *string, int64, *string) (int64, error) { return 4, nil },
		releaseLeasesFunc:   func(context.Context, *string) (int64, error) { return 5, nil },
		resolveReceiptsFunc: func(context.Context, string, *string, int64) (int64, error) { return 6, nil },
		cleanupReceiptsFunc: func(context.Context, *int64) (int64, error) { return 7, nil },
		timeoutReceiptsFunc: func(context.Context, int64, int64) (int64, error) { return 8, nil },
	}
	dal := newTxCore(t, stub)
	ctx := t.Context()

	promoted, err := dal.PromoteScheduledTasks(ctx)
	require.NoError(t, err, "promote must succeed")
	assert.Equal(t, 2, promoted, "promote count must be returned")

	recovered, err := dal.RecoverStaleTasks(ctx, time.Minute, 3, "recovered")
	require.NoError(t, err, "recover stale must succeed")
	assert.Equal(t, 3, recovered, "recover stale count must be returned")

	claimedRecovered, err := dal.RecoverClaimedTasks(ctx, "node-a", 3, "recovered")
	require.NoError(t, err, "recover claimed must succeed")
	assert.Equal(t, 4, claimedRecovered, "recover claimed count must be returned")

	released, err := dal.ReleaseRecoveryLeases(ctx, "node-a")
	require.NoError(t, err, "release leases must succeed")
	assert.Equal(t, 5, released, "release count must be returned")

	resolved, err := dal.ResolveWorkflowReceipts(ctx, "wf", "boom")
	require.NoError(t, err, "resolve receipts must succeed")
	assert.Equal(t, 6, resolved, "resolve count must be returned")

	cleaned, err := dal.CleanupOldResolvedReceipts(ctx, time.Unix(1000, 0))
	require.NoError(t, err, "cleanup must succeed")
	assert.Equal(t, 7, cleaned, "cleanup count must be returned")

	timedOut, err := dal.TimeoutStaleReceipts(ctx, time.Unix(1000, 0))
	require.NoError(t, err, "timeout must succeed")
	assert.Equal(t, 8, timedOut, "timeout count must be returned")
}

func TestCore_CreateWorkflowReceiptAndHeartbeat(t *testing.T) {
	t.Parallel()

	var receiptID string
	var heartbeatTaskID string
	stub := &configurableStubDriver{
		createReceiptFunc: func(_ context.Context, params CreateWorkflowReceiptParams) error {
			receiptID = params.ID
			return nil
		},
		heartbeatFunc: func(_ context.Context, _ int64, taskID string) error {
			heartbeatTaskID = taskID
			return nil
		},
	}
	dal := newTxCore(t, stub)

	require.NoError(t, dal.CreateWorkflowReceipt(t.Context(), "r1", "wf", "node-a"), "create receipt must succeed")
	require.Equal(t, "r1", receiptID, "receipt ID must be forwarded")

	require.NoError(t, dal.UpdateTaskHeartbeat(t.Context(), "task-1"), "heartbeat must succeed")
	require.Equal(t, "task-1", heartbeatTaskID, "task ID must be forwarded to the heartbeat query")
}

func TestCore_TransactionErrorWrapping(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("promote failed")
	stub := &configurableStubDriver{promoteFunc: func(context.Context, int64, int64) (int64, error) {
		return 0, sentinel
	}}
	dal := newTxCore(t, stub)

	_, err := dal.PromoteScheduledTasks(t.Context())
	require.ErrorIs(t, err, sentinel, "promote error must propagate through the transaction wrapper")
	require.Contains(t, err.Error(), "promoting scheduled tasks", "error must describe the operation")
}

func TestCore_RunAtomic(t *testing.T) {
	t.Parallel()

	t.Run("success runs fn with transactional store", func(t *testing.T) {
		t.Parallel()
		dal := newTxCore(t, &configurableStubDriver{})
		var invoked bool
		err := dal.RunAtomic(t.Context(), func(_ context.Context, store orchestrator_domain.TaskStore) error {
			invoked = true
			require.NotNil(t, store, "the transactional store must be provided")
			return nil
		})
		require.NoError(t, err, "a successful RunAtomic must commit")
		require.True(t, invoked, "fn must be invoked")
	})

	t.Run("nested transaction is rejected", func(t *testing.T) {
		t.Parallel()
		dal := &core{driver: &configurableStubDriver{}, clock: clock.RealClock(), inTransaction: true}
		err := dal.RunAtomic(t.Context(), func(context.Context, orchestrator_domain.TaskStore) error {
			return nil
		})
		require.Error(t, err, "a nested transaction must be rejected")
	})
}

func TestCore_RunInTransaction_NotInitialised(t *testing.T) {
	t.Parallel()

	dal := &core{driver: &configurableStubDriver{}, clock: clock.RealClock()}
	err := dal.runInTransaction(t.Context(), func(context.Context, Driver) error {
		return nil
	})
	require.ErrorIs(t, err, errDALNotInitialised, "a core without a sql.DB must report DAL not initialised")
}

func TestCore_RunInTransaction_ReusesDriverInsideTransaction(t *testing.T) {
	t.Parallel()

	var called bool
	stub := &configurableStubDriver{}
	dal := &core{driver: stub, clock: clock.RealClock(), inTransaction: true}
	err := dal.runInTransaction(t.Context(), func(_ context.Context, driver Driver) error {
		called = true
		require.Equal(t, Driver(stub), driver, "an in-transaction core must reuse its existing driver")
		return nil
	})
	require.NoError(t, err, "reusing the driver must succeed")
	require.True(t, called, "the callback must run")
}

func TestCore_MockClockDeterministicTimestamps(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	var receiptParams CreateWorkflowReceiptParams
	var heartbeatSeconds int64
	stub := &configurableStubDriver{
		createReceiptFunc: func(_ context.Context, params CreateWorkflowReceiptParams) error {
			receiptParams = params
			return nil
		},
		heartbeatFunc: func(_ context.Context, updatedAt int64, _ string) error {
			heartbeatSeconds = updatedAt
			return nil
		},
	}
	dal := newTxCore(t, stub)
	dal.clock = clock.NewMockClock(fixedTime)

	require.NoError(t, dal.CreateWorkflowReceipt(t.Context(), "r1", "wf", "node-a"), "create receipt must succeed")
	require.Equal(t, fixedTime.Unix(), receiptParams.CreatedAt, "CreatedAt must equal the injected mock clock time")
	require.Equal(t, fixedTime.Unix(), receiptParams.UpdatedAt, "UpdatedAt must equal the injected mock clock time")

	require.NoError(t, dal.UpdateTaskHeartbeat(t.Context(), "task-1"), "heartbeat must succeed")
	require.Equal(t, fixedTime.Unix(), heartbeatSeconds, "heartbeat timestamp must equal the injected mock clock time")
}
