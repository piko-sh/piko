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

package orchestrator_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockTaskStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockTaskStore
	ctx := context.Background()
	task := &Task{ID: "zero"}
	now := time.Now()

	require.NoError(t, m.CreateTask(ctx, task))
	require.NoError(t, m.CreateTasks(ctx, []*Task{task}))
	require.NoError(t, m.UpdateTask(ctx, task))

	tasks, err := m.FetchAndMarkDueTasks(ctx, PriorityNormal, 10)
	require.NoError(t, err)
	assert.Nil(t, tasks)

	ok, err := m.GetWorkflowStatus(ctx, "wf")
	require.NoError(t, err)
	assert.False(t, ok)

	n, err := m.PromoteScheduledTasks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	count, err := m.PendingTaskCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	require.NoError(t, m.CreateTaskWithDedup(ctx, task))

	n, err = m.RecoverStaleTasks(ctx, time.Minute, 3, "err")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	count, err = m.GetStaleProcessingTaskCount(ctx, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	require.NoError(t, m.UpdateTaskHeartbeat(ctx, "tid"))

	claimed, err := m.ClaimStaleTasksForRecovery(ctx, "node", time.Minute, time.Minute, 10)
	require.NoError(t, err)
	assert.Nil(t, claimed)

	n, err = m.RecoverClaimedTasks(ctx, "node", 3, "err")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	n, err = m.ReleaseRecoveryLeases(ctx, "node")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	require.NoError(t, m.CreateWorkflowReceipt(ctx, "id", "wf", "node"))

	n, err = m.ResolveWorkflowReceipts(ctx, "wf", "")
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	receipts, err := m.GetPendingReceiptsByNode(ctx, "node")
	require.NoError(t, err)
	assert.Nil(t, receipts)

	n, err = m.CleanupOldResolvedReceipts(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	n, err = m.TimeoutStaleReceipts(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	failed, err := m.ListFailedTasks(ctx)
	require.NoError(t, err)
	assert.Nil(t, failed)
}

func TestMockTaskStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	m := &MockTaskStore{}
	ctx := context.Background()
	task := &Task{ID: "concurrent"}
	now := time.Now()

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = m.CreateTask(ctx, task)
			_ = m.CreateTasks(ctx, []*Task{task})
			_ = m.UpdateTask(ctx, task)
			_, _ = m.FetchAndMarkDueTasks(ctx, PriorityNormal, 10)
			_, _ = m.GetWorkflowStatus(ctx, "wf")
			_, _ = m.PromoteScheduledTasks(ctx)
			_, _ = m.PendingTaskCount(ctx)
			_ = m.CreateTaskWithDedup(ctx, task)
			_, _ = m.RecoverStaleTasks(ctx, time.Minute, 3, "err")
			_, _ = m.GetStaleProcessingTaskCount(ctx, time.Minute)
			_ = m.UpdateTaskHeartbeat(ctx, "tid")
			_, _ = m.ClaimStaleTasksForRecovery(ctx, "node", time.Minute, time.Minute, 10)
			_, _ = m.RecoverClaimedTasks(ctx, "node", 3, "err")
			_, _ = m.ReleaseRecoveryLeases(ctx, "node")
			_ = m.CreateWorkflowReceipt(ctx, "id", "wf", "node")
			_, _ = m.ResolveWorkflowReceipts(ctx, "wf", "")
			_, _ = m.GetPendingReceiptsByNode(ctx, "node")
			_, _ = m.CleanupOldResolvedReceipts(ctx, now)
			_, _ = m.TimeoutStaleReceipts(ctx, now)
			_, _ = m.ListFailedTasks(ctx)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CreateTaskCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CreateTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.UpdateTaskCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.FetchAndMarkDueTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetWorkflowStatusCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PromoteScheduledTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PendingTaskCountCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CreateTaskWithDedupCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RecoverStaleTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetStaleProcessingTaskCountCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.UpdateTaskHeartbeatCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ClaimStaleTasksForRecoveryCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RecoverClaimedTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ReleaseRecoveryLeasesCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CreateWorkflowReceiptCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ResolveWorkflowReceiptsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetPendingReceiptsByNodeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CleanupOldResolvedReceiptsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.TimeoutStaleReceiptsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ListFailedTasksCallCount))
}

func TestMockTaskStore_CreateTask(t *testing.T) {
	t.Parallel()

	t.Run("nil CreateTaskFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.CreateTask(context.Background(), &Task{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CreateTaskCallCount))
	})

	t.Run("delegates to CreateTaskFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTask *Task
		m := &MockTaskStore{
			CreateTaskFunc: func(_ context.Context, task *Task) error {
				capturedTask = task
				return nil
			},
		}

		task := &Task{ID: "ct1"}
		err := m.CreateTask(context.Background(), task)

		require.NoError(t, err)
		assert.Same(t, task, capturedTask)
	})

	t.Run("propagates error from CreateTaskFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("create failed")
		m := &MockTaskStore{
			CreateTaskFunc: func(_ context.Context, _ *Task) error {
				return expected
			},
		}

		err := m.CreateTask(context.Background(), &Task{})
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_CreateTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil CreateTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.CreateTasks(context.Background(), []*Task{{ID: "b1"}})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CreateTasksCallCount))
	})

	t.Run("delegates to CreateTasksFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTasks []*Task
		m := &MockTaskStore{
			CreateTasksFunc: func(_ context.Context, tasks []*Task) error {
				capturedTasks = tasks
				return nil
			},
		}

		tasks := []*Task{{ID: "b1"}, {ID: "b2"}}
		err := m.CreateTasks(context.Background(), tasks)

		require.NoError(t, err)
		assert.Len(t, capturedTasks, 2)
	})

	t.Run("propagates error from CreateTasksFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("batch create failed")
		m := &MockTaskStore{
			CreateTasksFunc: func(_ context.Context, _ []*Task) error {
				return expected
			},
		}

		err := m.CreateTasks(context.Background(), nil)
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_UpdateTask(t *testing.T) {
	t.Parallel()

	t.Run("nil UpdateTaskFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.UpdateTask(context.Background(), &Task{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpdateTaskCallCount))
	})

	t.Run("delegates to UpdateTaskFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTask *Task
		m := &MockTaskStore{
			UpdateTaskFunc: func(_ context.Context, task *Task) error {
				capturedTask = task
				return nil
			},
		}

		task := &Task{ID: "ut1"}
		err := m.UpdateTask(context.Background(), task)

		require.NoError(t, err)
		assert.Same(t, task, capturedTask)
	})

	t.Run("propagates error from UpdateTaskFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("update failed")
		m := &MockTaskStore{
			UpdateTaskFunc: func(_ context.Context, _ *Task) error {
				return expected
			},
		}

		err := m.UpdateTask(context.Background(), &Task{})
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_FetchAndMarkDueTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil FetchAndMarkDueTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		tasks, err := m.FetchAndMarkDueTasks(context.Background(), PriorityHigh, 5)

		require.NoError(t, err)
		assert.Nil(t, tasks)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.FetchAndMarkDueTasksCallCount))
	})

	t.Run("delegates to FetchAndMarkDueTasksFunc", func(t *testing.T) {
		t.Parallel()

		want := []*Task{{ID: "f1"}, {ID: "f2"}}
		var capturedPriority TaskPriority
		var capturedLimit int
		m := &MockTaskStore{
			FetchAndMarkDueTasksFunc: func(_ context.Context, p TaskPriority, limit int) ([]*Task, error) {
				capturedPriority = p
				capturedLimit = limit
				return want, nil
			},
		}

		got, err := m.FetchAndMarkDueTasks(context.Background(), PriorityHigh, 7)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, PriorityHigh, capturedPriority)
		assert.Equal(t, 7, capturedLimit)
	})

	t.Run("propagates error from FetchAndMarkDueTasksFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("fetch failed")
		m := &MockTaskStore{
			FetchAndMarkDueTasksFunc: func(_ context.Context, _ TaskPriority, _ int) ([]*Task, error) {
				return nil, expected
			},
		}

		_, err := m.FetchAndMarkDueTasks(context.Background(), PriorityNormal, 1)
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_GetWorkflowStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil GetWorkflowStatusFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		ok, err := m.GetWorkflowStatus(context.Background(), "wf-1")

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetWorkflowStatusCallCount))
	})

	t.Run("delegates to GetWorkflowStatusFunc", func(t *testing.T) {
		t.Parallel()

		var capturedID string
		m := &MockTaskStore{
			GetWorkflowStatusFunc: func(_ context.Context, id string) (bool, error) {
				capturedID = id
				return true, nil
			},
		}

		ok, err := m.GetWorkflowStatus(context.Background(), "wf-done")

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "wf-done", capturedID)
	})

	t.Run("propagates error from GetWorkflowStatusFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("status failed")
		m := &MockTaskStore{
			GetWorkflowStatusFunc: func(_ context.Context, _ string) (bool, error) {
				return false, expected
			},
		}

		_, err := m.GetWorkflowStatus(context.Background(), "wf")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_PromoteScheduledTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil PromoteScheduledTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.PromoteScheduledTasks(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PromoteScheduledTasksCallCount))
	})

	t.Run("delegates to PromoteScheduledTasksFunc", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{
			PromoteScheduledTasksFunc: func(_ context.Context) (int, error) {
				return 5, nil
			},
		}

		n, err := m.PromoteScheduledTasks(context.Background())

		require.NoError(t, err)
		assert.Equal(t, 5, n)
	})

	t.Run("propagates error from PromoteScheduledTasksFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("promote failed")
		m := &MockTaskStore{
			PromoteScheduledTasksFunc: func(_ context.Context) (int, error) {
				return 0, expected
			},
		}

		_, err := m.PromoteScheduledTasks(context.Background())
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_PendingTaskCount(t *testing.T) {
	t.Parallel()

	t.Run("nil PendingTaskCountFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		count, err := m.PendingTaskCount(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PendingTaskCountCallCount))
	})

	t.Run("delegates to PendingTaskCountFunc", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{
			PendingTaskCountFunc: func(_ context.Context) (int64, error) {
				return 123, nil
			},
		}

		count, err := m.PendingTaskCount(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(123), count)
	})

	t.Run("propagates error from PendingTaskCountFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("count failed")
		m := &MockTaskStore{
			PendingTaskCountFunc: func(_ context.Context) (int64, error) {
				return 0, expected
			},
		}

		_, err := m.PendingTaskCount(context.Background())
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_CreateTaskWithDedup(t *testing.T) {
	t.Parallel()

	t.Run("nil CreateTaskWithDedupFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.CreateTaskWithDedup(context.Background(), &Task{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CreateTaskWithDedupCallCount))
	})

	t.Run("delegates to CreateTaskWithDedupFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTask *Task
		m := &MockTaskStore{
			CreateTaskWithDedupFunc: func(_ context.Context, task *Task) error {
				capturedTask = task
				return nil
			},
		}

		task := &Task{ID: "dedup-1"}
		err := m.CreateTaskWithDedup(context.Background(), task)

		require.NoError(t, err)
		assert.Same(t, task, capturedTask)
	})

	t.Run("propagates error from CreateTaskWithDedupFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("dedup failed")
		m := &MockTaskStore{
			CreateTaskWithDedupFunc: func(_ context.Context, _ *Task) error {
				return expected
			},
		}

		err := m.CreateTaskWithDedup(context.Background(), &Task{})
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_RecoverStaleTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil RecoverStaleTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.RecoverStaleTasks(context.Background(), time.Minute, 3, "err")

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RecoverStaleTasksCallCount))
	})

	t.Run("delegates to RecoverStaleTasksFunc", func(t *testing.T) {
		t.Parallel()

		var capturedThreshold time.Duration
		var capturedRetries int
		var capturedError string
		m := &MockTaskStore{
			RecoverStaleTasksFunc: func(_ context.Context, threshold time.Duration, maxRetries int, recoveryError string) (int, error) {
				capturedThreshold = threshold
				capturedRetries = maxRetries
				capturedError = recoveryError
				return 7, nil
			},
		}

		n, err := m.RecoverStaleTasks(context.Background(), 5*time.Minute, 4, "stale recovery")

		require.NoError(t, err)
		assert.Equal(t, 7, n)
		assert.Equal(t, 5*time.Minute, capturedThreshold)
		assert.Equal(t, 4, capturedRetries)
		assert.Equal(t, "stale recovery", capturedError)
	})

	t.Run("propagates error from RecoverStaleTasksFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("recovery failed")
		m := &MockTaskStore{
			RecoverStaleTasksFunc: func(_ context.Context, _ time.Duration, _ int, _ string) (int, error) {
				return 0, expected
			},
		}

		_, err := m.RecoverStaleTasks(context.Background(), time.Minute, 3, "err")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_GetStaleProcessingTaskCount(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStaleProcessingTaskCountFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		count, err := m.GetStaleProcessingTaskCount(context.Background(), time.Minute)

		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStaleProcessingTaskCountCallCount))
	})

	t.Run("delegates to GetStaleProcessingTaskCountFunc", func(t *testing.T) {
		t.Parallel()

		var capturedThreshold time.Duration
		m := &MockTaskStore{
			GetStaleProcessingTaskCountFunc: func(_ context.Context, threshold time.Duration) (int64, error) {
				capturedThreshold = threshold
				return 42, nil
			},
		}

		count, err := m.GetStaleProcessingTaskCount(context.Background(), 10*time.Minute)

		require.NoError(t, err)
		assert.Equal(t, int64(42), count)
		assert.Equal(t, 10*time.Minute, capturedThreshold)
	})

	t.Run("propagates error from GetStaleProcessingTaskCountFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("stale count failed")
		m := &MockTaskStore{
			GetStaleProcessingTaskCountFunc: func(_ context.Context, _ time.Duration) (int64, error) {
				return 0, expected
			},
		}

		_, err := m.GetStaleProcessingTaskCount(context.Background(), time.Minute)
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_UpdateTaskHeartbeat(t *testing.T) {
	t.Parallel()

	t.Run("nil UpdateTaskHeartbeatFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.UpdateTaskHeartbeat(context.Background(), "task-1")

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.UpdateTaskHeartbeatCallCount))
	})

	t.Run("delegates to UpdateTaskHeartbeatFunc", func(t *testing.T) {
		t.Parallel()

		var capturedID string
		m := &MockTaskStore{
			UpdateTaskHeartbeatFunc: func(_ context.Context, taskID string) error {
				capturedID = taskID
				return nil
			},
		}

		err := m.UpdateTaskHeartbeat(context.Background(), "hb-1")

		require.NoError(t, err)
		assert.Equal(t, "hb-1", capturedID)
	})

	t.Run("propagates error from UpdateTaskHeartbeatFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("heartbeat failed")
		m := &MockTaskStore{
			UpdateTaskHeartbeatFunc: func(_ context.Context, _ string) error {
				return expected
			},
		}

		err := m.UpdateTaskHeartbeat(context.Background(), "tid")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_ClaimStaleTasksForRecovery(t *testing.T) {
	t.Parallel()

	t.Run("nil ClaimStaleTasksForRecoveryFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		claimed, err := m.ClaimStaleTasksForRecovery(context.Background(), "node", time.Minute, time.Minute, 10)

		require.NoError(t, err)
		assert.Nil(t, claimed)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ClaimStaleTasksForRecoveryCallCount))
	})

	t.Run("delegates to ClaimStaleTasksForRecoveryFunc", func(t *testing.T) {
		t.Parallel()

		want := []RecoveryClaimedTask{{ID: "rc1", WorkflowID: "wf1", Attempt: 2}}
		var capturedNodeID string
		var capturedStale time.Duration
		var capturedLease time.Duration
		var capturedLimit int
		m := &MockTaskStore{
			ClaimStaleTasksForRecoveryFunc: func(_ context.Context, nodeID string, stale, lease time.Duration, limit int) ([]RecoveryClaimedTask, error) {
				capturedNodeID = nodeID
				capturedStale = stale
				capturedLease = lease
				capturedLimit = limit
				return want, nil
			},
		}

		got, err := m.ClaimStaleTasksForRecovery(context.Background(), "node-a", 5*time.Minute, 10*time.Minute, 50)

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, "node-a", capturedNodeID)
		assert.Equal(t, 5*time.Minute, capturedStale)
		assert.Equal(t, 10*time.Minute, capturedLease)
		assert.Equal(t, 50, capturedLimit)
	})

	t.Run("propagates error from ClaimStaleTasksForRecoveryFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("claim failed")
		m := &MockTaskStore{
			ClaimStaleTasksForRecoveryFunc: func(_ context.Context, _ string, _, _ time.Duration, _ int) ([]RecoveryClaimedTask, error) {
				return nil, expected
			},
		}

		_, err := m.ClaimStaleTasksForRecovery(context.Background(), "n", time.Second, time.Second, 1)
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_RecoverClaimedTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil RecoverClaimedTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.RecoverClaimedTasks(context.Background(), "node", 3, "err")

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RecoverClaimedTasksCallCount))
	})

	t.Run("delegates to RecoverClaimedTasksFunc", func(t *testing.T) {
		t.Parallel()

		var capturedNodeID string
		var capturedRetries int
		var capturedError string
		m := &MockTaskStore{
			RecoverClaimedTasksFunc: func(_ context.Context, nodeID string, maxRetries int, recoveryError string) (int, error) {
				capturedNodeID = nodeID
				capturedRetries = maxRetries
				capturedError = recoveryError
				return 3, nil
			},
		}

		n, err := m.RecoverClaimedTasks(context.Background(), "node-b", 5, "recovered")

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, "node-b", capturedNodeID)
		assert.Equal(t, 5, capturedRetries)
		assert.Equal(t, "recovered", capturedError)
	})

	t.Run("propagates error from RecoverClaimedTasksFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("recover claimed failed")
		m := &MockTaskStore{
			RecoverClaimedTasksFunc: func(_ context.Context, _ string, _ int, _ string) (int, error) {
				return 0, expected
			},
		}

		_, err := m.RecoverClaimedTasks(context.Background(), "n", 1, "e")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_ReleaseRecoveryLeases(t *testing.T) {
	t.Parallel()

	t.Run("nil ReleaseRecoveryLeasesFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.ReleaseRecoveryLeases(context.Background(), "node")

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ReleaseRecoveryLeasesCallCount))
	})

	t.Run("delegates to ReleaseRecoveryLeasesFunc", func(t *testing.T) {
		t.Parallel()

		var capturedNodeID string
		m := &MockTaskStore{
			ReleaseRecoveryLeasesFunc: func(_ context.Context, nodeID string) (int, error) {
				capturedNodeID = nodeID
				return 2, nil
			},
		}

		n, err := m.ReleaseRecoveryLeases(context.Background(), "node-c")

		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.Equal(t, "node-c", capturedNodeID)
	})

	t.Run("propagates error from ReleaseRecoveryLeasesFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("release failed")
		m := &MockTaskStore{
			ReleaseRecoveryLeasesFunc: func(_ context.Context, _ string) (int, error) {
				return 0, expected
			},
		}

		_, err := m.ReleaseRecoveryLeases(context.Background(), "n")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_CreateWorkflowReceipt(t *testing.T) {
	t.Parallel()

	t.Run("nil CreateWorkflowReceiptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.CreateWorkflowReceipt(context.Background(), "id", "wf", "node")

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CreateWorkflowReceiptCallCount))
	})

	t.Run("delegates to CreateWorkflowReceiptFunc", func(t *testing.T) {
		t.Parallel()

		var capturedID, capturedWF, capturedNode string
		m := &MockTaskStore{
			CreateWorkflowReceiptFunc: func(_ context.Context, id, workflowID, nodeID string) error {
				capturedID = id
				capturedWF = workflowID
				capturedNode = nodeID
				return nil
			},
		}

		err := m.CreateWorkflowReceipt(context.Background(), "r1", "wf-1", "node-1")

		require.NoError(t, err)
		assert.Equal(t, "r1", capturedID)
		assert.Equal(t, "wf-1", capturedWF)
		assert.Equal(t, "node-1", capturedNode)
	})

	t.Run("propagates error from CreateWorkflowReceiptFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("receipt create failed")
		m := &MockTaskStore{
			CreateWorkflowReceiptFunc: func(_ context.Context, _, _, _ string) error {
				return expected
			},
		}

		err := m.CreateWorkflowReceipt(context.Background(), "id", "wf", "node")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_ResolveWorkflowReceipts(t *testing.T) {
	t.Parallel()

	t.Run("nil ResolveWorkflowReceiptsFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.ResolveWorkflowReceipts(context.Background(), "wf", "")

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ResolveWorkflowReceiptsCallCount))
	})

	t.Run("delegates to ResolveWorkflowReceiptsFunc", func(t *testing.T) {
		t.Parallel()

		var capturedWF, capturedErr string
		m := &MockTaskStore{
			ResolveWorkflowReceiptsFunc: func(_ context.Context, workflowID, errorMessage string) (int, error) {
				capturedWF = workflowID
				capturedErr = errorMessage
				return 4, nil
			},
		}

		n, err := m.ResolveWorkflowReceipts(context.Background(), "wf-2", "some error")

		require.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, "wf-2", capturedWF)
		assert.Equal(t, "some error", capturedErr)
	})

	t.Run("propagates error from ResolveWorkflowReceiptsFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("resolve failed")
		m := &MockTaskStore{
			ResolveWorkflowReceiptsFunc: func(_ context.Context, _, _ string) (int, error) {
				return 0, expected
			},
		}

		_, err := m.ResolveWorkflowReceipts(context.Background(), "wf", "")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_GetPendingReceiptsByNode(t *testing.T) {
	t.Parallel()

	t.Run("nil GetPendingReceiptsByNodeFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		receipts, err := m.GetPendingReceiptsByNode(context.Background(), "node")

		require.NoError(t, err)
		assert.Nil(t, receipts)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetPendingReceiptsByNodeCallCount))
	})

	t.Run("delegates to GetPendingReceiptsByNodeFunc", func(t *testing.T) {
		t.Parallel()

		want := []PendingReceipt{{ID: "pr1", WorkflowID: "wf1", NodeID: "n1", CreatedAt: 100}}
		var capturedNode string
		m := &MockTaskStore{
			GetPendingReceiptsByNodeFunc: func(_ context.Context, nodeID string) ([]PendingReceipt, error) {
				capturedNode = nodeID
				return want, nil
			},
		}

		got, err := m.GetPendingReceiptsByNode(context.Background(), "node-x")

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, "node-x", capturedNode)
	})

	t.Run("propagates error from GetPendingReceiptsByNodeFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("pending receipts failed")
		m := &MockTaskStore{
			GetPendingReceiptsByNodeFunc: func(_ context.Context, _ string) ([]PendingReceipt, error) {
				return nil, expected
			},
		}

		_, err := m.GetPendingReceiptsByNode(context.Background(), "n")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_CleanupOldResolvedReceipts(t *testing.T) {
	t.Parallel()

	t.Run("nil CleanupOldResolvedReceiptsFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.CleanupOldResolvedReceipts(context.Background(), time.Now())

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CleanupOldResolvedReceiptsCallCount))
	})

	t.Run("delegates to CleanupOldResolvedReceiptsFunc", func(t *testing.T) {
		t.Parallel()

		cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		var capturedTime time.Time
		m := &MockTaskStore{
			CleanupOldResolvedReceiptsFunc: func(_ context.Context, olderThan time.Time) (int, error) {
				capturedTime = olderThan
				return 10, nil
			},
		}

		n, err := m.CleanupOldResolvedReceipts(context.Background(), cutoff)

		require.NoError(t, err)
		assert.Equal(t, 10, n)
		assert.Equal(t, cutoff, capturedTime)
	})

	t.Run("propagates error from CleanupOldResolvedReceiptsFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("cleanup failed")
		m := &MockTaskStore{
			CleanupOldResolvedReceiptsFunc: func(_ context.Context, _ time.Time) (int, error) {
				return 0, expected
			},
		}

		_, err := m.CleanupOldResolvedReceipts(context.Background(), time.Now())
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_TimeoutStaleReceipts(t *testing.T) {
	t.Parallel()

	t.Run("nil TimeoutStaleReceiptsFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		n, err := m.TimeoutStaleReceipts(context.Background(), time.Now())

		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.TimeoutStaleReceiptsCallCount))
	})

	t.Run("delegates to TimeoutStaleReceiptsFunc", func(t *testing.T) {
		t.Parallel()

		cutoff := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
		var capturedTime time.Time
		m := &MockTaskStore{
			TimeoutStaleReceiptsFunc: func(_ context.Context, olderThan time.Time) (int, error) {
				capturedTime = olderThan
				return 3, nil
			},
		}

		n, err := m.TimeoutStaleReceipts(context.Background(), cutoff)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		assert.Equal(t, cutoff, capturedTime)
	})

	t.Run("propagates error from TimeoutStaleReceiptsFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("timeout failed")
		m := &MockTaskStore{
			TimeoutStaleReceiptsFunc: func(_ context.Context, _ time.Time) (int, error) {
				return 0, expected
			},
		}

		_, err := m.TimeoutStaleReceipts(context.Background(), time.Now())
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockTaskStore_ListFailedTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil ListFailedTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		tasks, err := m.ListFailedTasks(context.Background())

		require.NoError(t, err)
		assert.Nil(t, tasks)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ListFailedTasksCallCount))
	})

	t.Run("delegates to ListFailedTasksFunc", func(t *testing.T) {
		t.Parallel()

		want := []*Task{{ID: "fail-1"}, {ID: "fail-2"}}
		m := &MockTaskStore{
			ListFailedTasksFunc: func(_ context.Context) ([]*Task, error) {
				return want, nil
			},
		}

		got, err := m.ListFailedTasks(context.Background())

		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("propagates error from ListFailedTasksFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("list failed tasks failed")
		m := &MockTaskStore{
			ListFailedTasksFunc: func(_ context.Context) ([]*Task, error) {
				return nil, expected
			},
		}

		_, err := m.ListFailedTasks(context.Background())
		assert.ErrorIs(t, err, expected)
	})
}
