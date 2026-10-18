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

package otter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func newTestDAL(t *testing.T) *DAL {
	t.Helper()
	dal, err := NewOtterDAL(Config{Capacity: 1000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = dal.Close() })
	d, ok := dal.(*DAL)
	require.True(t, ok)
	return d
}

func makeTask(id, workflowID string) *orchestrator_domain.Task {
	now := time.Now()
	return &orchestrator_domain.Task{
		ID:         id,
		WorkflowID: workflowID,
		Executor:   "test-executor",
		Status:     orchestrator_domain.StatusPending,
		ExecuteAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
		Config: orchestrator_domain.TaskConfig{
			Priority: orchestrator_domain.PriorityNormal,
		},
	}
}

func TestRunAtomic_CommitPreservesMutations(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		return store.CreateTask(ctx, makeTask("t1", "wf1"))
	})
	require.NoError(t, err)

	task, found, _ := d.tasks.GetIfPresent(ctx, "t1")
	require.True(t, found)
	require.Equal(t, "t1", task.ID)
}

func TestRunAtomic_RollbackOnError(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, d.CreateTask(ctx, makeTask("existing", "wf1")))

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		if err := store.CreateTask(ctx, makeTask("new-task", "wf2")); err != nil {
			return err
		}
		return errors.New("simulated failure")
	})
	require.Error(t, err)

	_, found, _ := d.tasks.GetIfPresent(ctx, "new-task")
	assert.False(t, found, "new task should be rolled back")

	_, found, _ = d.tasks.GetIfPresent(ctx, "existing")
	assert.True(t, found, "existing task should survive rollback")
}

func TestRunAtomic_RollbackRestoresReceipts(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		if err := store.CreateWorkflowReceipt(ctx, "r1", "wf1", "node1"); err != nil {
			return err
		}
		return errors.New("simulated failure")
	})
	require.Error(t, err)

	d.mu.RLock()
	_, exists := d.receipts["r1"]
	d.mu.RUnlock()
	assert.False(t, exists, "receipt should be rolled back")
}

func TestRunAtomic_PanicRollsBack(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	assert.Panics(t, func() {
		_ = d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
			_ = store.CreateTask(ctx, makeTask("panic-task", "wf1"))
			panic("boom")
		})
	})

	_, found, _ := d.tasks.GetIfPresent(ctx, "panic-task")
	assert.False(t, found, "task created before panic should be rolled back")
}

func TestRunAtomic_NestedTransactionReturnsError(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		return store.RunAtomic(ctx, func(context.Context, orchestrator_domain.TaskStore) error {
			return nil
		})
	})
	require.ErrorIs(t, err, cache_domain.ErrNestedTransactionUnsupported)
}

func TestRunAtomic_IndexesRebuiltOnRollback(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	task := makeTask("t1", "wf1")
	task.DeduplicationKey = "dedup-key"
	require.NoError(t, d.CreateTask(ctx, task))

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		newTask := makeTask("t2", "wf2")
		newTask.DeduplicationKey = "dedup-key-2"
		if err := store.CreateTask(ctx, newTask); err != nil {
			return err
		}
		return errors.New("simulated failure")
	})
	require.Error(t, err)

	d.mu.RLock()
	_, dedupExists := d.dedupIndex["dedup-key-2"]
	origDedupID := d.dedupIndex["dedup-key"]
	wfTasks := d.workflowIndex.Get("wf1")
	d.mu.RUnlock()

	assert.False(t, dedupExists, "new dedup entry should be rolled back")
	assert.Equal(t, "t1", origDedupID, "original dedup entry should be preserved")
	assert.Contains(t, wfTasks, "t1", "workflow index should contain original task")
}

func TestRunAtomic_MultipleTasksRolledBack(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		for i := range 5 {
			task := makeTask("batch-"+string(rune('a'+i)), "wf1")
			if err := store.CreateTask(ctx, task); err != nil {
				return err
			}
		}
		return errors.New("batch failure")
	})
	require.Error(t, err)

	assert.Equal(t, 0, d.tasks.EstimatedSize(), "all batch tasks should be rolled back")
}

func TestDAL_GetTasksByID(t *testing.T) {
	t.Parallel()

	d := newTestDAL(t)
	ctx := t.Context()
	require.NoError(t, d.CreateTask(ctx, makeTask("t1", "wf1")))
	require.NoError(t, d.CreateTask(ctx, makeTask("t2", "wf1")))

	tasks, err := d.GetTasksByID(ctx, []string{"t1", "missing", "t2"})
	require.NoError(t, err)
	require.Len(t, tasks, 2, "missing IDs are skipped")
	assert.Equal(t, "t1", tasks[0].ID)
	assert.Equal(t, "t2", tasks[1].ID)

	tasks[0].Status = orchestrator_domain.StatusFailed
	stored, err := d.GetTasksByID(ctx, []string{"t1"})
	require.NoError(t, err)
	assert.Equal(t, orchestrator_domain.StatusPending, stored[0].Status, "the returned tasks are copies")

	err = d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		inTransaction, readErr := store.GetTasksByID(ctx, []string{"t2"})
		if readErr != nil {
			return readErr
		}
		if len(inTransaction) != 1 || inTransaction[0].ID != "t2" {
			return errors.New("transaction read returned the wrong tasks")
		}
		return nil
	})
	require.NoError(t, err)
}

func TestNewOtterDAL(t *testing.T) {
	t.Parallel()

	t.Run("creates with default capacity", func(t *testing.T) {
		t.Parallel()

		dal, err := NewOtterDAL(Config{})
		require.NoError(t, err)
		require.NotNil(t, dal)
		t.Cleanup(func() { _ = dal.Close() })
	})

	t.Run("creates with custom capacity", func(t *testing.T) {
		t.Parallel()

		dal, err := NewOtterDAL(Config{Capacity: 500})
		require.NoError(t, err)
		require.NotNil(t, dal)
		t.Cleanup(func() { _ = dal.Close() })
	})

	t.Run("creates with negative capacity uses default", func(t *testing.T) {
		t.Parallel()

		dal, err := NewOtterDAL(Config{Capacity: -1})
		require.NoError(t, err)
		require.NotNil(t, dal)
		t.Cleanup(func() { _ = dal.Close() })
	})
}

func TestDAL_HealthCheck(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	err := dal.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestDAL_Close(t *testing.T) {
	t.Parallel()

	t.Run("closes owned cache", func(t *testing.T) {
		t.Parallel()

		dal, err := NewOtterDAL(Config{Capacity: 100})
		require.NoError(t, err)
		assert.NoError(t, dal.Close())
	})
}

func TestDAL_CreateTask(t *testing.T) {
	t.Parallel()

	t.Run("stores task successfully", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("task-1", "wf-1")
		err := dal.CreateTask(ctx, task)
		require.NoError(t, err)

		stored, found, _ := dal.tasks.GetIfPresent(ctx, "task-1")
		assert.True(t, found)
		assert.Equal(t, "task-1", stored.ID)
	})

	t.Run("indexes task by workflow", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		require.NoError(t, dal.CreateTask(ctx, makeTask("t1", "wf-1")))
		require.NoError(t, dal.CreateTask(ctx, makeTask("t2", "wf-1")))
		require.NoError(t, dal.CreateTask(ctx, makeTask("t3", "wf-2")))

		dal.mu.RLock()
		workflowOneTasks := dal.workflowIndex.Get("wf-1")
		workflowTwoTasks := dal.workflowIndex.Get("wf-2")
		dal.mu.RUnlock()

		assert.Len(t, workflowOneTasks, 2)
		assert.Len(t, workflowTwoTasks, 1)
	})
}

func TestDAL_CreateTasks(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	tasks := []*orchestrator_domain.Task{
		makeTask("batch-1", "wf-1"),
		makeTask("batch-2", "wf-1"),
		makeTask("batch-3", "wf-2"),
	}

	err := dal.CreateTasks(ctx, tasks)
	require.NoError(t, err)

	assert.Equal(t, 3, dal.tasks.EstimatedSize())
}

func TestDAL_UpdateTask(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	task := makeTask("t1", "wf-1")
	require.NoError(t, dal.CreateTask(ctx, task))

	task.Status = orchestrator_domain.StatusComplete
	task.UpdatedAt = time.Now()
	err := dal.UpdateTask(ctx, task)
	require.NoError(t, err)

	stored, found, _ := dal.tasks.GetIfPresent(ctx, "t1")
	assert.True(t, found)
	assert.Equal(t, orchestrator_domain.StatusComplete, stored.Status)
}

func TestDAL_GetWorkflowStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns true when all tasks complete", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task1 := makeTask("t1", "wf-1")
		task1.Status = orchestrator_domain.StatusComplete
		task2 := makeTask("t2", "wf-1")
		task2.Status = orchestrator_domain.StatusFailed

		require.NoError(t, dal.CreateTask(ctx, task1))
		require.NoError(t, dal.CreateTask(ctx, task2))

		complete, err := dal.GetWorkflowStatus(ctx, "wf-1")
		require.NoError(t, err)
		assert.True(t, complete)
	})

	t.Run("returns false when tasks still pending", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task1 := makeTask("t1", "wf-1")
		task1.Status = orchestrator_domain.StatusComplete
		task2 := makeTask("t2", "wf-1")
		task2.Status = orchestrator_domain.StatusPending

		require.NoError(t, dal.CreateTask(ctx, task1))
		require.NoError(t, dal.CreateTask(ctx, task2))

		complete, err := dal.GetWorkflowStatus(ctx, "wf-1")
		require.NoError(t, err)
		assert.False(t, complete)
	})

	t.Run("reports an unknown workflow as complete", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		complete, err := dal.GetWorkflowStatus(context.Background(), "nonexistent")
		require.NoError(t, err)
		assert.True(t, complete, "a workflow with no tasks has nothing left to run")
	})
}

func TestDAL_PendingTaskCount(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	pending1 := makeTask("p1", "wf-1")
	pending1.Status = orchestrator_domain.StatusPending
	pending2 := makeTask("p2", "wf-1")
	pending2.Status = orchestrator_domain.StatusPending
	processing := makeTask("pr1", "wf-1")
	processing.Status = orchestrator_domain.StatusProcessing

	require.NoError(t, dal.CreateTask(ctx, pending1))
	require.NoError(t, dal.CreateTask(ctx, pending2))
	require.NoError(t, dal.CreateTask(ctx, processing))

	count, err := dal.PendingTaskCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestDAL_CreateTaskWithDedup(t *testing.T) {
	t.Parallel()

	t.Run("creates task with dedup key", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.DeduplicationKey = "dedup-1"
		err := dal.CreateTaskWithDedup(ctx, task)
		require.NoError(t, err)
	})

	t.Run("rejects duplicate active task", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task1 := makeTask("t1", "wf-1")
		task1.DeduplicationKey = "dedup-1"
		require.NoError(t, dal.CreateTaskWithDedup(ctx, task1))

		task2 := makeTask("t2", "wf-1")
		task2.DeduplicationKey = "dedup-1"
		err := dal.CreateTaskWithDedup(ctx, task2)
		require.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask)
	})

	t.Run("allows task with same key when previous is complete", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task1 := makeTask("t1", "wf-1")
		task1.DeduplicationKey = "dedup-1"
		task1.Status = orchestrator_domain.StatusComplete
		require.NoError(t, dal.CreateTask(ctx, task1))

		dal.mu.Lock()
		dal.dedupIndex["dedup-1"] = "t1"
		dal.mu.Unlock()

		task2 := makeTask("t2", "wf-1")
		task2.DeduplicationKey = "dedup-1"
		err := dal.CreateTaskWithDedup(ctx, task2)
		require.NoError(t, err)
	})

	t.Run("creates task without dedup key", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.DeduplicationKey = ""
		err := dal.CreateTaskWithDedup(ctx, task)
		require.NoError(t, err)
	})
}

func TestDAL_WorkflowReceipts(t *testing.T) {
	t.Parallel()

	t.Run("creates and retrieves receipt", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		err := dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1")
		require.NoError(t, err)

		dal.mu.RLock()
		receipt, exists := dal.receipts["r1"]
		dal.mu.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, receiptPending, receipt.status)
		assert.Equal(t, "wf-1", receipt.workflowID)
		assert.Equal(t, "node-1", receipt.nodeID)
	})

	t.Run("resolves receipts for workflow", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1"))
		require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r2", "wf-1", "node-2"))
		require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r3", "wf-2", "node-1"))

		count, err := dal.ResolveWorkflowReceipts(ctx, "wf-1", "")
		require.NoError(t, err)
		assert.Equal(t, 2, count)

		dal.mu.RLock()
		assert.Equal(t, receiptResolved, dal.receipts["r1"].status)
		assert.Equal(t, receiptResolved, dal.receipts["r2"].status)
		assert.Equal(t, receiptPending, dal.receipts["r3"].status)
		dal.mu.RUnlock()
	})

	t.Run("resolves receipts with error message", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1"))

		count, err := dal.ResolveWorkflowReceipts(ctx, "wf-1", "workflow failed")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		dal.mu.RLock()
		assert.Equal(t, "workflow failed", dal.receipts["r1"].errorMessage)
		dal.mu.RUnlock()
	})
}

func TestDAL_GetPendingReceiptsByNode(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1"))
	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r2", "wf-2", "node-1"))
	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r3", "wf-3", "node-2"))

	receipts, err := dal.GetPendingReceiptsByNode(ctx, "node-1")
	require.NoError(t, err)
	assert.Len(t, receipts, 2)
}

func TestDAL_GetPendingReceiptsByWorkflow(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1"))
	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r2", "wf-1", "node-2"))
	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r3", "wf-2", "node-1"))

	receipts, err := dal.GetPendingReceiptsByWorkflow(ctx, "wf-1")
	require.NoError(t, err)
	assert.Len(t, receipts, 2)
}

func TestDAL_CleanupOldResolvedReceipts(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1"))
	_, _ = dal.ResolveWorkflowReceipts(ctx, "wf-1", "")

	dal.mu.Lock()
	dal.receipts["r1"].resolvedAt = time.Now().Add(-2 * time.Hour)
	dal.resolvedIndex.Add("r1", dal.receipts["r1"].resolvedAt)
	dal.mu.Unlock()

	count, err := dal.CleanupOldResolvedReceipts(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	dal.mu.RLock()
	_, exists := dal.receipts["r1"]
	dal.mu.RUnlock()
	assert.False(t, exists)
}

func TestDAL_TimeoutStaleReceipts(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, dal.CreateWorkflowReceipt(ctx, "r1", "wf-1", "node-1"))

	dal.mu.Lock()
	dal.receipts["r1"].createdAt = time.Now().Add(-2 * time.Hour)
	dal.mu.Unlock()

	count, err := dal.TimeoutStaleReceipts(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	dal.mu.RLock()
	assert.Equal(t, receiptTimedOut, dal.receipts["r1"].status)
	dal.mu.RUnlock()
}

func TestDAL_ListFailedTasks(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	failedTask := makeTask("failed", "wf-1")
	failedTask.Status = orchestrator_domain.StatusFailed
	require.NoError(t, dal.CreateTask(ctx, failedTask))

	pendingTask := makeTask("pending", "wf-1")
	pendingTask.Status = orchestrator_domain.StatusPending
	require.NoError(t, dal.CreateTask(ctx, pendingTask))

	failed, err := dal.ListFailedTasks(ctx)
	require.NoError(t, err)
	assert.Len(t, failed, 1)
	assert.Equal(t, "failed", failed[0].ID)
}

func TestDAL_ListTaskSummary(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	pending := makeTask("p1", "wf-1")
	pending.Status = orchestrator_domain.StatusPending
	require.NoError(t, dal.CreateTask(ctx, pending))

	pending2 := makeTask("p2", "wf-1")
	pending2.Status = orchestrator_domain.StatusPending
	require.NoError(t, dal.CreateTask(ctx, pending2))

	failed := makeTask("f1", "wf-1")
	failed.Status = orchestrator_domain.StatusFailed
	require.NoError(t, dal.CreateTask(ctx, failed))

	summaries, err := dal.ListTaskSummary(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, summaries)

	statusCounts := make(map[string]int64)
	for _, summary := range summaries {
		statusCounts[summary.Status] = summary.Count
	}

	assert.Equal(t, int64(2), statusCounts[string(orchestrator_domain.StatusPending)])
	assert.Equal(t, int64(1), statusCounts[string(orchestrator_domain.StatusFailed)])
}

func TestDAL_ListRecentTasks(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	old := makeTask("old", "wf-1")
	old.UpdatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, dal.CreateTask(ctx, old))

	recent := makeTask("recent", "wf-1")
	recent.UpdatedAt = time.Now()
	require.NoError(t, dal.CreateTask(ctx, recent))

	items, err := dal.ListRecentTasks(ctx, 1)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "recent", items[0].ID)
}

func TestDAL_ListRecentTasks_LimitGreaterThanTasks(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, dal.CreateTask(ctx, makeTask("t1", "wf-1")))

	items, err := dal.ListRecentTasks(ctx, 100)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestDAL_ListWorkflowSummary(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	task1 := makeTask("t1", "wf-1")
	task1.Status = orchestrator_domain.StatusComplete
	require.NoError(t, dal.CreateTask(ctx, task1))

	task2 := makeTask("t2", "wf-1")
	task2.Status = orchestrator_domain.StatusPending
	require.NoError(t, dal.CreateTask(ctx, task2))

	task3 := makeTask("t3", "wf-2")
	task3.Status = orchestrator_domain.StatusFailed
	require.NoError(t, dal.CreateTask(ctx, task3))

	summaries, err := dal.ListWorkflowSummary(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, summaries, 2)
}

func TestDAL_PromoteScheduledTasks(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	dueTask := makeTask("due", "wf-1")
	dueTask.Status = orchestrator_domain.StatusScheduled
	dueTask.ScheduledExecuteAt = time.Now().Add(-time.Minute)
	require.NoError(t, dal.CreateTask(ctx, dueTask))

	futureTask := makeTask("future", "wf-1")
	futureTask.Status = orchestrator_domain.StatusScheduled
	futureTask.ScheduledExecuteAt = time.Now().Add(time.Hour)
	require.NoError(t, dal.CreateTask(ctx, futureTask))

	count, err := dal.PromoteScheduledTasks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	stored, _, _ := dal.tasks.GetIfPresent(ctx, "due")
	assert.Equal(t, orchestrator_domain.StatusPending, stored.Status)

	storedFuture, _, _ := dal.tasks.GetIfPresent(ctx, "future")
	assert.Equal(t, orchestrator_domain.StatusScheduled, storedFuture.Status)
}

func TestDAL_RebuildIndexes(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	task := makeTask("t1", "wf-1")
	task.DeduplicationKey = "dedup-key"
	require.NoError(t, dal.CreateTask(ctx, task))

	dal.RebuildIndexes(ctx)

	dal.mu.RLock()
	dedupID := dal.dedupIndex["dedup-key"]
	workflowTasks := dal.workflowIndex.Get("wf-1")
	dal.mu.RUnlock()

	assert.Equal(t, "t1", dedupID)
	assert.Contains(t, workflowTasks, "t1")
}
