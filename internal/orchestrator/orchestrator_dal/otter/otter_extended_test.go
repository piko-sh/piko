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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestNewOtterDAL(t *testing.T) {
	t.Parallel()

	t.Run("creates with default capacity", func(t *testing.T) {
		t.Parallel()

		dal, err := NewOtterDAL(Config{Capacity: 0})
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

	t.Run("returns error for unknown workflow", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		_, err := dal.GetWorkflowStatus(context.Background(), "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
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

func TestDAL_UpdateTaskHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("updates heartbeat for processing task", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusProcessing
		task.UpdatedAt = time.Now().Add(-time.Minute)
		require.NoError(t, dal.CreateTask(ctx, task))

		beforeUpdate := time.Now()
		err := dal.UpdateTaskHeartbeat(ctx, "t1")
		require.NoError(t, err)

		stored, _, _ := dal.tasks.GetIfPresent(ctx, "t1")
		assert.True(t, stored.UpdatedAt.After(beforeUpdate) || stored.UpdatedAt.Equal(beforeUpdate))
	})

	t.Run("returns error for non-existent task", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		err := dal.UpdateTaskHeartbeat(context.Background(), "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error for non-processing task", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusPending
		require.NoError(t, dal.CreateTask(ctx, task))

		err := dal.UpdateTaskHeartbeat(ctx, "t1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in PROCESSING status")
	})
}

func TestDAL_FetchAndMarkDueTasks(t *testing.T) {
	t.Parallel()

	t.Run("marks due pending tasks as processing", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusPending
		task.ExecuteAt = time.Now().Add(-time.Minute)
		require.NoError(t, dal.CreateTask(ctx, task))

		results, err := dal.FetchAndMarkDueTasks(ctx, orchestrator_domain.PriorityNormal, 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, orchestrator_domain.StatusProcessing, results[0].Status)
	})

	t.Run("respects priority filter", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		normalTask := makeTask("t-normal", "wf-1")
		normalTask.Status = orchestrator_domain.StatusPending
		normalTask.ExecuteAt = time.Now().Add(-time.Minute)
		normalTask.Config.Priority = orchestrator_domain.PriorityNormal
		require.NoError(t, dal.CreateTask(ctx, normalTask))

		highTask := makeTask("t-high", "wf-1")
		highTask.Status = orchestrator_domain.StatusPending
		highTask.ExecuteAt = time.Now().Add(-time.Minute)
		highTask.Config.Priority = orchestrator_domain.PriorityHigh
		require.NoError(t, dal.CreateTask(ctx, highTask))

		results, err := dal.FetchAndMarkDueTasks(ctx, orchestrator_domain.PriorityNormal, 10)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "t-normal", results[0].ID)
	})

	t.Run("respects limit", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		for i := range 5 {
			task := makeTask("t-"+string(rune('a'+i)), "wf-1")
			task.Status = orchestrator_domain.StatusPending
			task.ExecuteAt = time.Now().Add(-time.Minute)
			require.NoError(t, dal.CreateTask(ctx, task))
		}

		results, err := dal.FetchAndMarkDueTasks(ctx, orchestrator_domain.PriorityNormal, 2)
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("does not fetch future tasks", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t-future", "wf-1")
		task.Status = orchestrator_domain.StatusPending
		task.ExecuteAt = time.Now().Add(time.Hour)
		require.NoError(t, dal.CreateTask(ctx, task))

		results, err := dal.FetchAndMarkDueTasks(ctx, orchestrator_domain.PriorityNormal, 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestDAL_RecoverStaleTasks(t *testing.T) {
	t.Parallel()

	t.Run("recovers stale processing tasks", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusProcessing
		task.UpdatedAt = time.Now().Add(-2 * time.Hour)
		require.NoError(t, dal.CreateTask(ctx, task))

		count, err := dal.RecoverStaleTasks(ctx, time.Hour, 3, "stale task recovery")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		stored, _, _ := dal.tasks.GetIfPresent(ctx, "t1")
		assert.Equal(t, orchestrator_domain.StatusRetrying, stored.Status)
		assert.Equal(t, "stale task recovery", stored.LastError)
	})

	t.Run("marks as failed when max retries exceeded", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusProcessing
		task.UpdatedAt = time.Now().Add(-2 * time.Hour)
		task.Attempt = 2
		require.NoError(t, dal.CreateTask(ctx, task))

		count, err := dal.RecoverStaleTasks(ctx, time.Hour, 3, "exceeded retries")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		stored, _, _ := dal.tasks.GetIfPresent(ctx, "t1")
		assert.Equal(t, orchestrator_domain.StatusFailed, stored.Status)
	})

	t.Run("skips tasks with active recovery leases", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusProcessing
		task.UpdatedAt = time.Now().Add(-2 * time.Hour)
		require.NoError(t, dal.CreateTask(ctx, task))

		dal.mu.Lock()
		dal.recoveryLeases["t1"] = &recoveryLease{
			taskID:    "t1",
			nodeID:    "node-1",
			expiresAt: time.Now().Add(time.Hour),
		}
		dal.mu.Unlock()

		count, err := dal.RecoverStaleTasks(ctx, time.Hour, 3, "recovery")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestDAL_GetStaleProcessingTaskCount(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	staleTask := makeTask("stale", "wf-1")
	staleTask.Status = orchestrator_domain.StatusProcessing
	staleTask.UpdatedAt = time.Now().Add(-2 * time.Hour)
	require.NoError(t, dal.CreateTask(ctx, staleTask))

	freshTask := makeTask("fresh", "wf-1")
	freshTask.Status = orchestrator_domain.StatusProcessing
	freshTask.UpdatedAt = time.Now()
	require.NoError(t, dal.CreateTask(ctx, freshTask))

	count, err := dal.GetStaleProcessingTaskCount(ctx, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestDAL_ClaimStaleTasksForRecovery(t *testing.T) {
	t.Parallel()

	t.Run("claims stale tasks", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		task := makeTask("t1", "wf-1")
		task.Status = orchestrator_domain.StatusProcessing
		task.UpdatedAt = time.Now().Add(-2 * time.Hour)
		require.NoError(t, dal.CreateTask(ctx, task))

		claimed, err := dal.ClaimStaleTasksForRecovery(ctx, "node-1", time.Hour, time.Hour, 10)
		require.NoError(t, err)
		assert.Len(t, claimed, 1)
		assert.Equal(t, "t1", claimed[0].ID)

		dal.mu.RLock()
		lease, exists := dal.recoveryLeases["t1"]
		dal.mu.RUnlock()
		assert.True(t, exists)
		assert.Equal(t, "node-1", lease.nodeID)
	})

	t.Run("respects batch limit", func(t *testing.T) {
		t.Parallel()

		dal := newTestDAL(t)
		ctx := context.Background()

		for i := range 5 {
			task := makeTask("t-"+string(rune('a'+i)), "wf-1")
			task.Status = orchestrator_domain.StatusProcessing
			task.UpdatedAt = time.Now().Add(-2 * time.Hour)
			require.NoError(t, dal.CreateTask(ctx, task))
		}

		claimed, err := dal.ClaimStaleTasksForRecovery(ctx, "node-1", time.Hour, time.Hour, 2)
		require.NoError(t, err)
		assert.Len(t, claimed, 2)
	})
}

func TestDAL_RecoverClaimedTasks(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	task := makeTask("t1", "wf-1")
	task.Status = orchestrator_domain.StatusProcessing
	task.UpdatedAt = time.Now().Add(-2 * time.Hour)
	require.NoError(t, dal.CreateTask(ctx, task))

	dal.mu.Lock()
	dal.recoveryLeases["t1"] = &recoveryLease{
		taskID: "t1",
		nodeID: "node-1",
	}
	dal.mu.Unlock()

	count, err := dal.RecoverClaimedTasks(ctx, "node-1", 3, "recovered")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	stored, _, _ := dal.tasks.GetIfPresent(ctx, "t1")
	assert.Equal(t, orchestrator_domain.StatusRetrying, stored.Status)
	assert.Equal(t, "recovered", stored.LastError)

	dal.mu.RLock()
	_, leaseExists := dal.recoveryLeases["t1"]
	dal.mu.RUnlock()
	assert.False(t, leaseExists)
}

func TestDAL_ReleaseRecoveryLeases(t *testing.T) {
	t.Parallel()

	dal := newTestDAL(t)
	ctx := context.Background()

	dal.mu.Lock()
	dal.recoveryLeases["t1"] = &recoveryLease{taskID: "t1", nodeID: "node-1"}
	dal.recoveryLeases["t2"] = &recoveryLease{taskID: "t2", nodeID: "node-1"}
	dal.recoveryLeases["t3"] = &recoveryLease{taskID: "t3", nodeID: "node-2"}
	dal.mu.Unlock()

	count, err := dal.ReleaseRecoveryLeases(ctx, "node-1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	dal.mu.RLock()
	assert.Len(t, dal.recoveryLeases, 1)
	_, nodeOneExists := dal.recoveryLeases["t1"]
	_, nodeTwoExists := dal.recoveryLeases["t3"]
	dal.mu.RUnlock()
	assert.False(t, nodeOneExists)
	assert.True(t, nodeTwoExists)
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
