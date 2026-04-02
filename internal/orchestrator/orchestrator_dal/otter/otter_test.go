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

func TestRunAtomic_RollbackRestoresRecoveryLeases(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	task := makeTask("stale1", "wf1")
	task.Status = orchestrator_domain.StatusProcessing
	task.UpdatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, d.CreateTask(ctx, task))

	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		_, claimErr := store.ClaimStaleTasksForRecovery(ctx, "node1", time.Minute, time.Hour, 10)
		if claimErr != nil {
			return claimErr
		}
		return errors.New("simulated failure")
	})
	require.Error(t, err)

	d.mu.RLock()
	_, leaseExists := d.recoveryLeases["stale1"]
	d.mu.RUnlock()
	assert.False(t, leaseExists, "recovery lease should be rolled back")
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
