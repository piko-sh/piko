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

package persistence_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orchestrator_sqlite "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func makeTask(id string, executor string) *orchestrator_domain.Task {
	now := time.Now().UTC().Truncate(time.Second)
	return &orchestrator_domain.Task{
		ID:         id,
		WorkflowID: "wf-" + id,
		Executor:   executor,
		Status:     orchestrator_domain.StatusPending,
		ExecuteAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
		Payload:    map[string]any{"key": "value"},
		Result:     map[string]any{},
		Config: orchestrator_domain.TaskConfig{
			Priority:   orchestrator_domain.PriorityNormal,
			Timeout:    5 * time.Minute,
			MaxRetries: 3,
		},
		Attempt: 0,
	}
}

func createTask(t *testing.T, adapter orchestrator_domain.TaskStore, ctx context.Context, task *orchestrator_domain.Task) {
	t.Helper()
	err := adapter.CreateTasks(ctx, []*orchestrator_domain.Task{task})
	require.NoError(t, err, "creating task via batch path")
}

func dueNow(t *testing.T, adapter orchestrator_domain.TaskStore, limit int) []*orchestrator_domain.Task {
	t.Helper()
	page, err := adapter.DispatchablePage(context.Background(), orchestrator_domain.DispatchQuery{
		Now:   time.Now().UTC(),
		Limit: limit,
	})
	require.NoError(t, err, "reading dispatchable tasks")
	return page
}

func claim(t *testing.T, adapter orchestrator_domain.TaskStore, task *orchestrator_domain.Task) *orchestrator_domain.Task {
	t.Helper()
	claimed, err := adapter.ClaimTask(context.Background(), orchestrator_domain.ClaimRequest{
		Now:     time.Now().UTC(),
		ID:      task.ID,
		Owner:   "test-instance",
		Attempt: task.Attempt,
	})
	require.NoError(t, err, "claiming the task")
	return claimed
}

func TestOrchestratorCreateAndGetTask(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	task := makeTask("task-create-001", "test-executor")
	createTask(t, adapter, ctx, task)

	fetched, err := adapter.GetTasksByID(ctx, []string{task.ID})
	require.NoError(t, err, "reading the task")
	require.Len(t, fetched, 1)

	assert.Equal(t, task.ID, fetched[0].ID)
	assert.Equal(t, task.WorkflowID, fetched[0].WorkflowID)
	assert.Equal(t, task.Executor, fetched[0].Executor)

	require.Len(t, dueNow(t, adapter, 10), 1, "the new task is due")
	claimed := claim(t, adapter, fetched[0])
	assert.Equal(t, orchestrator_domain.StatusProcessing, claimed.Status)
	assert.Empty(t, dueNow(t, adapter, 10), "a claimed task is no longer dispatchable")
}

func TestOrchestratorUpdateTask(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	task := makeTask("task-update-001", "test-executor")
	createTask(t, adapter, ctx, task)

	claimed := claim(t, adapter, task)
	claimed.Status = orchestrator_domain.StatusComplete
	claimed.Result = map[string]any{"output": "done"}
	claimed.UpdatedAt = time.Now().UTC().Truncate(time.Second)

	err := adapter.UpdateTask(ctx, claimed)
	require.NoError(t, err, "updating task")

	is_complete, err := adapter.GetWorkflowStatus(ctx, task.WorkflowID)
	require.NoError(t, err, "getting workflow status")
	assert.True(t, is_complete, "workflow should be complete")
}

func TestOrchestratorDispatchablePagesDueTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	tasks := make([]*orchestrator_domain.Task, 3)
	for i := range 3 {
		task := makeTask("task-fetch-"+[]string{"a", "b", "c"}[i], "test-executor")
		tasks[i] = task
	}
	err := adapter.CreateTasks(ctx, tasks)
	require.NoError(t, err, "creating tasks")

	first := dueNow(t, adapter, 2)
	require.Len(t, first, 2, "a page holds at most its limit")

	last := first[len(first)-1]
	remaining, err := adapter.DispatchablePage(ctx, orchestrator_domain.DispatchQuery{
		Now:       time.Now().UTC(),
		AfterTime: last.ExecuteAt,
		AfterID:   last.ID,
		Limit:     10,
	})
	require.NoError(t, err, "reading the next page")
	assert.Len(t, remaining, 1, "the next page continues after the cursor")
}

func TestOrchestratorPromoteScheduledTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	task := makeTask("task-promote-001", "test-executor")
	task.Status = orchestrator_domain.StatusScheduled
	task.ExecuteAt = time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)
	createTask(t, adapter, ctx, task)

	assert.Empty(t, dueNow(t, adapter, 10), "scheduled task should not be dispatchable before promotion")

	promoted_count, err := adapter.PromoteScheduledTasks(ctx)
	require.NoError(t, err, "promoting scheduled tasks")
	assert.Equal(t, 1, promoted_count)

	fetched := dueNow(t, adapter, 10)
	require.Len(t, fetched, 1)
	assert.Equal(t, task.ID, fetched[0].ID)
}

func TestOrchestratorPendingTaskCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	count, err := adapter.PendingTaskCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	tasks := make([]*orchestrator_domain.Task, 3)
	for i := range 3 {
		tasks[i] = makeTask("task-count-"+[]string{"a", "b", "c"}[i], "test-executor")
	}
	err = adapter.CreateTasks(ctx, tasks)
	require.NoError(t, err, "creating tasks")

	count, err = adapter.PendingTaskCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestOrchestratorCreateTaskWithDedup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	now_seconds := time.Now().UTC().Unix()
	_, err := database.ExecContext(ctx,
		`INSERT INTO tasks (id, workflow_id, executor, priority, payload, config, status, execute_at, attempt, created_at, updated_at, deduplication_key)
		 VALUES (?, ?, ?, ?, '{}', '{}', ?, ?, 0, ?, ?, ?)`,
		"task-dedup-001", "wf-dedup-001", "test-executor", int32(orchestrator_domain.PriorityNormal),
		string(orchestrator_domain.StatusPending), int32(now_seconds), int32(now_seconds), int32(now_seconds),
		"dedup-key-001",
	)
	require.NoError(t, err, "inserting first task with dedup key")

	task2 := makeTask("task-dedup-002", "test-executor")
	task2.DeduplicationKey = "dedup-key-001"

	err = adapter.CreateTaskWithDedup(ctx, task2)
	assert.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask)
}

func TestOrchestratorWorkflowReceipts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	workflow_id := "wf-receipt-001"
	node_id := "node-001"

	err := adapter.CreateWorkflowReceipt(ctx, "receipt-001", workflow_id, node_id)
	require.NoError(t, err, "creating workflow receipt")

	pending, err := adapter.GetPendingReceiptsByNode(ctx, node_id)
	require.NoError(t, err, "getting pending receipts")
	require.Len(t, pending, 1)
	assert.Equal(t, workflow_id, pending[0].WorkflowID)
	assert.Equal(t, node_id, pending[0].NodeID)

	resolved_count, err := adapter.ResolveWorkflowReceipts(ctx, workflow_id, "")
	require.NoError(t, err, "resolving workflow receipts")
	assert.Equal(t, 1, resolved_count)

	pending, err = adapter.GetPendingReceiptsByNode(ctx, node_id)
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestOrchestratorRunAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupOrchestratorDB(t)
	adapter := orchestrator_sqlite.New(database)
	ctx := context.Background()

	deliberate_error := errors.New("deliberate rollback")

	err := adapter.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		task := makeTask("task-atomic-rollback", "test-executor")
		create_err := store.CreateTasks(ctx, []*orchestrator_domain.Task{task})
		if create_err != nil {
			return create_err
		}

		return deliberate_error
	})
	assert.ErrorIs(t, err, deliberate_error)

	count, err := adapter.PendingTaskCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "no tasks should exist after rollback")
}
