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
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	d "piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

func readTask(t *testing.T, store d.TaskStore, id string) *d.Task {
	t.Helper()
	rows, err := store.GetTasksByID(t.Context(), []string{id})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	return rows[0]
}

func claimedTask(t *testing.T, core *d.TaskProcessingCore, store d.TaskStore, key string) *d.Task {
	t.Helper()
	task := d.NewTask("test", nil)
	task.Status = d.StatusPending
	task.DeduplicationKey = key
	require.NoError(t, store.CreateTaskWithDedup(t.Context(), task))
	core.TasksDispatched.Add(1)
	require.NoError(t, core.ClaimDelivery(t.Context(), task))
	require.Equal(t, d.StatusProcessing, task.Status)
	return task
}

func TestPanicAfterSuccessIsCountedOnce(t *testing.T) {
	core, store := claimCore(t)
	task := claimedTask(t, core, store, "once")

	core.HandleTaskSuccess(t.Context(), task, time.Now())
	core.HandleTaskFailure(t.Context(), task, errors.New("panic in task executor"), time.Now())

	stats := core.Stats()
	assert.Equal(t, int64(1), stats.Completed)
	assert.Zero(t, stats.Failed)
	assert.Zero(t, stats.Retried)
	assert.Equal(t, d.StatusComplete, readTask(t, store, task.ID).Status)
}

func TestLostClaimReleasesTheDispatchInsteadOfSettling(t *testing.T) {
	clock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	core, store := claimCoreWithClock(t, clock)
	task := claimedTask(t, core, store, "lost")

	clock.Advance(core.Config.StaleTaskThreshold + time.Minute)
	peer := d.NewTaskProcessingCore(core.Config, nil, store, clock)
	retry, err := core.RecoverStaleTasks(t.Context())
	require.NoError(t, err)
	assert.Empty(t, retry, "an instance never recovers its own claims; its pool settles or abandons them")
	retry, err = peer.RecoverStaleTasks(t.Context())
	require.NoError(t, err)
	require.Len(t, retry, 1, "a peer recovers the orphan for another attempt")

	core.HandleTaskSuccess(t.Context(), task, clock.Now())
	stored := readTask(t, store, task.ID)
	assert.Equal(t, d.StatusRetrying, stored.Status, "the superseded attempt must not overwrite the recovery")
	assert.Equal(t, 1, stored.Attempt)
	stats := core.Stats()
	assert.Zero(t, stats.Completed, "a superseded attempt is not counted as completed")
	assert.Zero(t, stats.Dispatched, "a superseded attempt releases its dispatch; the recovered attempt is dispatched afresh")
}

type flakySettleStore struct {
	d.TaskStore
	failures atomic.Int32
}

func (s *flakySettleStore) SettleTask(ctx context.Context, settlement d.Settlement) (d.SettleOutcome, error) {
	if s.failures.Add(-1) >= 0 {
		return d.SettleOutcome{}, errors.New("database unavailable")
	}
	return s.TaskStore.SettleTask(ctx, settlement)
}

func TestFailedSettleStaysInFlightUntilRetried(t *testing.T) {
	_, inner := claimCore(t)
	store := &flakySettleStore{TaskStore: inner}
	store.failures.Store(1)
	cfg := d.DefaultDispatcherConfig()
	cfg.HeartbeatInterval = 0
	core := d.NewTaskProcessingCore(cfg, nil, store, nil)
	task := claimedTask(t, core, store, "flaky")

	core.HandleTaskSuccess(t.Context(), task, time.Now())
	assert.Zero(t, core.Stats().Completed, "nothing is counted before the state is written")
	assert.Equal(t, 1, core.PendingSettleCount(), "the attempt stays in flight while its settle is pending")
	assert.Equal(t, d.StatusProcessing, readTask(t, inner, task.ID).Status)

	assert.Equal(t, 1, core.RetryPendingSettles(t.Context()))
	assert.Equal(t, int64(1), core.Stats().Completed)
	assert.Zero(t, core.PendingSettleCount())
	assert.Equal(t, d.StatusComplete, readTask(t, inner, task.ID).Status)
}

func TestReleaseAndAbandonCannotOverwriteAClaimedTask(t *testing.T) {
	for _, name := range []string{"release", "abandon"} {
		t.Run(name, func(t *testing.T) {
			core, store := claimCore(t)
			task := d.NewTask("test", nil)
			task.Status = d.StatusPending
			task.DeduplicationKey = "deli-2-" + name
			require.NoError(t, store.CreateTaskWithDedup(t.Context(), task))
			stale := *task

			require.NoError(t, core.ClaimDelivery(t.Context(), task), "the sweep's message is claimed first")

			if name == "release" {
				err := core.ReleaseUnrequiredTask(t.Context(), &stale)
				require.ErrorIs(t, err, d.ErrObsoleteDelivery)
			} else {
				core.AbandonUnpublishedTask(t.Context(), &stale, errors.New("ambiguous publish"))
			}

			stored := readTask(t, store, task.ID)
			assert.Equal(t, d.StatusProcessing, stored.Status, "the running attempt keeps its claim")
			assert.Equal(t, 1, stored.Attempt)
			duplicate := d.NewTask("test", nil)
			duplicate.DeduplicationKey = task.DeduplicationKey
			assert.ErrorIs(t, store.CreateTaskWithDedup(t.Context(), duplicate), d.ErrDuplicateTask,
				"the key stays held by the running attempt")
		})
	}
}

func TestRecoveryKeepsTheAttemptAndHonoursPerTaskRetries(t *testing.T) {
	testCases := []struct {
		name       string
		attempt    int
		maxRetries int
		want       d.TaskStatus
	}{
		{name: "first attempt, default budget", attempt: 1, want: d.StatusRetrying},
		{name: "second attempt, default budget", attempt: 2, want: d.StatusRetrying},
		{name: "budget spent, default budget", attempt: 3, want: d.StatusFailed},
		{name: "second attempt, budget of two", attempt: 2, maxRetries: 2, want: d.StatusFailed},
		{name: "fourth attempt, budget of five", attempt: 4, maxRetries: 5, want: d.StatusRetrying},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
			store, err := otter.NewOtterDAL(otter.Config{Capacity: 100}, otter.WithClock(clock))
			require.NoError(t, err)
			t.Cleanup(func() { _ = store.Close() })
			cfg := d.DefaultDispatcherConfig()
			cfg.HeartbeatInterval = 0
			cfg.DefaultMaxRetries = 3
			core := d.NewTaskProcessingCore(cfg, nil, store, clock)

			task := d.NewTask("test", nil)
			task.Status = d.StatusProcessing
			task.Attempt = tc.attempt
			task.Config.MaxRetries = tc.maxRetries
			task.UpdatedAt = clock.Now()
			require.NoError(t, store.CreateTask(t.Context(), task))

			clock.Advance(cfg.StaleTaskThreshold + time.Minute)
			_, err = core.RecoverStaleTasks(t.Context())
			require.NoError(t, err)

			stored := readTask(t, store, task.ID)
			assert.Equal(t, tc.want, stored.Status)
			assert.Equal(t, tc.attempt, stored.Attempt, "recovery never charges an attempt; the next claim does")
		})
	}
}

func TestRetriedAttemptSettlesWhenClaimedAgain(t *testing.T) {
	clock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))
	core, store := claimCoreWithClock(t, clock)
	task := claimedTask(t, core, store, "retried")
	task.Config.MaxRetries = 3

	core.HandleTaskFailure(t.Context(), task, errors.New("temporary"), clock.Now())
	stored := readTask(t, store, task.ID)
	require.Equal(t, d.StatusRetrying, stored.Status)
	require.Equal(t, 1, stored.Attempt)

	clock.Advance(time.Minute)
	core.TasksDispatched.Add(1)
	require.NoError(t, core.ClaimDelivery(t.Context(), stored), "the retry's delivery claims the next attempt")
	core.HandleTaskSuccess(t.Context(), stored, clock.Now())

	assert.Equal(t, d.StatusComplete, readTask(t, store, task.ID).Status, "the retried attempt settles")
	stats := core.Stats()
	assert.Equal(t, int64(1), stats.Retried)
	assert.Equal(t, int64(1), stats.Completed)
}
