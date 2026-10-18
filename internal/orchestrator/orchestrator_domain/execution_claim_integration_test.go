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

package orchestrator_domain_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

func claimCore(t *testing.T) (*d.TaskProcessingCore, d.TaskStore) {
	t.Helper()
	store, err := otter.NewOtterDAL(otter.Config{Capacity: 1000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	cfg := d.DefaultDispatcherConfig()
	cfg.HeartbeatInterval = 0
	return d.NewTaskProcessingCore(cfg, nil, store, nil), store
}
func TestClaimDeliveryExecutesOneAttempt(t *testing.T) {
	core, store := claimCore(t)
	task := d.NewTask("test", nil)
	task.Status = d.StatusPending
	require.NoError(t, store.CreateTask(t.Context(), task))
	var successes atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			delivery := *task
			err := core.ClaimDelivery(t.Context(), &delivery)
			if err == nil {
				successes.Add(1)
			} else {
				require.ErrorIs(t, err, d.ErrObsoleteDelivery)
			}
		})
	}
	wg.Wait()
	require.EqualValues(t, 1, successes.Load())
	rows, err := store.GetTasksByID(t.Context(), []string{task.ID})
	require.NoError(t, err)
	require.Equal(t, 1, rows[0].Attempt)
	require.Equal(t, d.StatusProcessing, rows[0].Status)
}
func TestStaleAttemptCannotOverwriteRecovery(t *testing.T) {
	core, store := claimCore(t)
	task := d.NewTask("test", nil)
	task.Status = d.StatusPending
	require.NoError(t, store.CreateTask(t.Context(), task))
	require.NoError(t, core.ClaimDelivery(t.Context(), task))
	current := *task
	current.UpdatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, store.UpdateTask(t.Context(), &current))
	peer := d.NewTaskProcessingCore(core.Config, nil, store, nil)
	retry, err := peer.RecoverStaleTasks(t.Context())
	require.NoError(t, err)
	require.Len(t, retry, 1, "a peer recovers the attempt whose owner holds no lease")
	core.HandleTaskSuccess(t.Context(), task, time.Now())
	rows, err := store.GetTasksByID(t.Context(), []string{task.ID})
	require.NoError(t, err)
	require.Equal(t, d.StatusRetrying, rows[0].Status)
	require.Equal(t, 1, rows[0].Attempt, "recovery keeps the attempt; the next claim counts the next one")
	require.NoError(t, core.ClaimDelivery(t.Context(), rows[0]))
	require.Equal(t, 2, rows[0].Attempt)
}

type successorExecutor struct {
	next, duplicate *d.Task
	due             time.Time
}

func (e successorExecutor) Execute(ctx context.Context, _ map[string]any) (map[string]any, error) {
	d.ScheduleSuccessor(ctx, e.next, e.due)
	d.ScheduleSuccessor(ctx, e.duplicate, e.due)
	return nil, nil
}
func TestRecurringSuccessorPersistsOnceWithCorrectDueTime(t *testing.T) {
	core, store := claimCore(t)
	current := d.NewTask("gc", nil)
	current.DeduplicationKey = "gc.hints"
	current.Status = d.StatusPending
	require.NoError(t, store.CreateTaskWithDedup(t.Context(), current))
	require.NoError(t, core.ClaimDelivery(t.Context(), current))
	next := d.NewTask("gc", nil)
	next.DeduplicationKey = current.DeduplicationKey
	duplicate := d.NewTask("gc", nil)
	duplicate.DeduplicationKey = current.DeduplicationKey
	due := time.Now().Add(time.Hour).Truncate(time.Second)
	core.RegisterExecutor(t.Context(), current.Executor, successorExecutor{next, duplicate, due})
	runClaimed(t, core, current)
	core.HandleTaskSuccess(t.Context(), current, time.Now())
	rows, err := store.GetTasksByID(t.Context(), []string{current.ID, next.ID, duplicate.ID})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, d.StatusComplete, rows[0].Status)
	require.Equal(t, d.StatusScheduled, rows[1].Status)
	require.True(t, rows[1].ScheduledExecuteAt.Equal(due))
	count, err := store.PromoteScheduledTasks(t.Context())
	require.NoError(t, err)
	require.Zero(t, count)
	require.ErrorIs(t, store.CreateTaskWithDedup(t.Context(), duplicate), d.ErrDuplicateTask)
}

func (successorExecutor) SchedulesSuccessor() bool { return true }

func runClaimed(t *testing.T, core *d.TaskProcessingCore, task *d.Task) {
	t.Helper()
	pool := d.NewExecutionPool(d.ExecutionPoolConfig{}, nil, core.RunClaimed, core.HandleAbandonedExecution)
	slot, err := pool.Acquire(t.Context(), task.Config.Priority)
	require.NoError(t, err)
	slot.Dispatch(t.Context(), task)
}

func claimCoreWithClock(t *testing.T, clk clockpkg.Clock) (*d.TaskProcessingCore, d.TaskStore) {
	t.Helper()
	store, err := otter.NewOtterDAL(otter.Config{Capacity: 1000}, otter.WithClock(clk))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	cfg := d.DefaultDispatcherConfig()
	cfg.HeartbeatInterval = 0
	return d.NewTaskProcessingCore(cfg, nil, store, clk), store
}

func storeClaimTask(t *testing.T, store d.TaskStore, status d.TaskStatus, attempt int, executeAt time.Time) *d.Task {
	t.Helper()
	task := d.NewTask("test", nil)
	task.Status = status
	task.Attempt = attempt
	task.ExecuteAt = executeAt
	task.ScheduledExecuteAt = executeAt
	task.CreatedAt = executeAt
	task.UpdatedAt = executeAt
	require.NoError(t, store.CreateTask(t.Context(), task))
	return task
}

func TestClaimDeliveryPendingIsAlwaysDue(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for name, executeAt := range map[string]time.Time{
		"due later by the shared clock": now.Add(time.Hour),
		"stamped by a clock far ahead":  time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			core, store := claimCoreWithClock(t, clockpkg.NewMockClock(now))
			task := storeClaimTask(t, store, d.StatusPending, 0, executeAt)

			delivery := *task
			require.NoError(t, core.ClaimDelivery(t.Context(), &delivery))

			rows, err := store.GetTasksByID(t.Context(), []string{task.ID})
			require.NoError(t, err)
			require.Equal(t, d.StatusProcessing, rows[0].Status)
			require.Equal(t, 1, rows[0].Attempt)
		})
	}
}

func TestClaimDeliveryDeferredStatuses(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, deferred := range []struct {
		status  d.TaskStatus
		attempt int
	}{
		{status: d.StatusRetrying, attempt: 1},
		{status: d.StatusScheduled, attempt: 0},
	} {
		t.Run(string(deferred.status)+" due in an hour is premature until due", func(t *testing.T) {
			clk := clockpkg.NewMockClock(now)
			core, store := claimCoreWithClock(t, clk)
			due := now.Add(time.Hour)
			task := storeClaimTask(t, store, deferred.status, deferred.attempt, due)

			delivery := *task
			err := core.ClaimDelivery(t.Context(), &delivery)
			require.ErrorIs(t, err, d.ErrPrematureDelivery)
			require.NotErrorIs(t, err, d.ErrObsoleteDelivery)
			require.True(t, delivery.Persisted(), "the stored state is re-armed without a new insert")
			require.Equal(t, deferred.status, delivery.Status)
			require.Equal(t, deferred.attempt, delivery.Attempt)
			require.True(t, delivery.ScheduledExecuteAt.Equal(due))

			rows, err := store.GetTasksByID(t.Context(), []string{task.ID})
			require.NoError(t, err)
			require.Equal(t, deferred.status, rows[0].Status, "a premature delivery writes nothing")
			require.Equal(t, deferred.attempt, rows[0].Attempt)

			clk.Advance(time.Hour)
			retry := *task
			require.NoError(t, core.ClaimDelivery(t.Context(), &retry))
			require.Equal(t, d.StatusProcessing, retry.Status)
			require.Equal(t, deferred.attempt+1, retry.Attempt)
		})

		for name, offset := range map[string]time.Duration{
			"within the skew tolerance": 500 * time.Millisecond,
			"already due":               -time.Second,
		} {
			t.Run(string(deferred.status)+" "+name+" is claimed", func(t *testing.T) {
				core, store := claimCoreWithClock(t, clockpkg.NewMockClock(now))
				task := storeClaimTask(t, store, deferred.status, deferred.attempt, now.Add(offset))

				delivery := *task
				require.NoError(t, core.ClaimDelivery(t.Context(), &delivery))
				require.Equal(t, d.StatusProcessing, delivery.Status)
			})
		}
	}
}

func TestClaimDeliveryObsolete(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name            string
		status          d.TaskStatus
		storedAttempt   int
		deliveryAttempt int
		executeAt       time.Time
	}{
		{name: "processing", status: d.StatusProcessing, storedAttempt: 1, deliveryAttempt: 1, executeAt: now},
		{name: "complete", status: d.StatusComplete, storedAttempt: 1, deliveryAttempt: 1, executeAt: now},
		{name: "failed", status: d.StatusFailed, storedAttempt: 1, deliveryAttempt: 1, executeAt: now},
		{name: "old attempt", status: d.StatusPending, storedAttempt: 1, deliveryAttempt: 0, executeAt: now},
		{name: "future retry from an old attempt", status: d.StatusRetrying, storedAttempt: 2, deliveryAttempt: 1, executeAt: now.Add(time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, store := claimCoreWithClock(t, clockpkg.NewMockClock(now))
			task := storeClaimTask(t, store, tc.status, tc.storedAttempt, tc.executeAt)

			delivery := *task
			delivery.Attempt = tc.deliveryAttempt
			err := core.ClaimDelivery(t.Context(), &delivery)
			require.ErrorIs(t, err, d.ErrObsoleteDelivery)
			require.NotErrorIs(t, err, d.ErrPrematureDelivery)
		})
	}

	t.Run("missing task", func(t *testing.T) {
		core, _ := claimCoreWithClock(t, clockpkg.NewMockClock(now))
		delivery := d.NewTask("test", nil)
		require.ErrorIs(t, core.ClaimDelivery(t.Context(), delivery), d.ErrObsoleteDelivery)
	})
}
