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

	for range goroutines {
		wg.Go(func() {

			_ = m.CreateTask(ctx, task)
			_ = m.CreateTasks(ctx, []*Task{task})
			_ = m.UpdateTask(ctx, task)
			_, _ = m.GetWorkflowStatus(ctx, "wf")
			_, _ = m.PromoteScheduledTasks(ctx)
			_, _ = m.PendingTaskCount(ctx)
			_ = m.CreateTaskWithDedup(ctx, task)
			_ = m.CreateWorkflowReceipt(ctx, "id", "wf", "node")
			_, _ = m.ResolveWorkflowReceipts(ctx, "wf", "")
			_, _ = m.GetPendingReceiptsByNode(ctx, "node")
			_, _ = m.CleanupOldResolvedReceipts(ctx, now)
			_, _ = m.TimeoutStaleReceipts(ctx, now)
			_, _ = m.ListFailedTasks(ctx)
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), m.CreateTaskCallCount.Load())
	assert.Equal(t, int64(goroutines), m.CreateTasksCallCount.Load())
	assert.Equal(t, int64(goroutines), m.UpdateTaskCallCount.Load())
	assert.Equal(t, int64(goroutines), m.GetWorkflowStatusCallCount.Load())
	assert.Equal(t, int64(goroutines), m.PromoteScheduledTasksCallCount.Load())
	assert.Equal(t, int64(goroutines), m.PendingTaskCountCallCount.Load())
	assert.Equal(t, int64(goroutines), m.CreateTaskWithDedupCallCount.Load())
	assert.Equal(t, int64(goroutines), m.CreateWorkflowReceiptCallCount.Load())
	assert.Equal(t, int64(goroutines), m.ResolveWorkflowReceiptsCallCount.Load())
	assert.Equal(t, int64(goroutines), m.GetPendingReceiptsByNodeCallCount.Load())
	assert.Equal(t, int64(goroutines), m.CleanupOldResolvedReceiptsCallCount.Load())
	assert.Equal(t, int64(goroutines), m.TimeoutStaleReceiptsCallCount.Load())
	assert.Equal(t, int64(goroutines), m.ListFailedTasksCallCount.Load())
}

func TestMockTaskStore_CreateTask(t *testing.T) {
	t.Parallel()

	t.Run("nil CreateTaskFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.CreateTask(context.Background(), &Task{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), m.CreateTaskCallCount.Load())
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
		assert.Equal(t, int64(1), m.CreateTasksCallCount.Load())
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
		assert.Equal(t, int64(1), m.UpdateTaskCallCount.Load())
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

func TestMockTaskStore_GetWorkflowStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil GetWorkflowStatusFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		ok, err := m.GetWorkflowStatus(context.Background(), "wf-1")

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, int64(1), m.GetWorkflowStatusCallCount.Load())
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
		assert.Equal(t, int64(1), m.PromoteScheduledTasksCallCount.Load())
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
		assert.Equal(t, int64(1), m.PendingTaskCountCallCount.Load())
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
		assert.Equal(t, int64(1), m.CreateTaskWithDedupCallCount.Load())
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

func TestMockTaskStore_CreateWorkflowReceipt(t *testing.T) {
	t.Parallel()

	t.Run("nil CreateWorkflowReceiptFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		err := m.CreateWorkflowReceipt(context.Background(), "id", "wf", "node")

		require.NoError(t, err)
		assert.Equal(t, int64(1), m.CreateWorkflowReceiptCallCount.Load())
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
		assert.Equal(t, int64(1), m.ResolveWorkflowReceiptsCallCount.Load())
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
		assert.Equal(t, int64(1), m.GetPendingReceiptsByNodeCallCount.Load())
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
		assert.Equal(t, int64(1), m.CleanupOldResolvedReceiptsCallCount.Load())
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
		assert.Equal(t, int64(1), m.TimeoutStaleReceiptsCallCount.Load())
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
		assert.Equal(t, int64(1), m.ListFailedTasksCallCount.Load())
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

func TestMockTaskStore_GetTasksByID(t *testing.T) {
	t.Parallel()

	t.Run("nil GetTasksByIDFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockTaskStore{}
		tasks, err := m.GetTasksByID(context.Background(), []string{"a"})

		require.NoError(t, err)
		assert.Nil(t, tasks)
		assert.Equal(t, int64(1), m.GetTasksByIDCallCount.Load())
	})

	t.Run("delegates to GetTasksByIDFunc", func(t *testing.T) {
		t.Parallel()

		want := []*Task{{ID: "a"}}
		var capturedIDs []string
		m := &MockTaskStore{
			GetTasksByIDFunc: func(_ context.Context, ids []string) ([]*Task, error) {
				capturedIDs = ids
				return want, nil
			},
		}

		got, err := m.GetTasksByID(context.Background(), []string{"a", "b"})

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, []string{"a", "b"}, capturedIDs)
	})
}
