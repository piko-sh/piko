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

package otter_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/worker/worker_dal/otter"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

func newStore(t *testing.T, clk clock.Clock) worker_domain.Store {
	t.Helper()
	store, err := otter.NewOtterDAL(otter.Config{}, otter.WithClock(clk))
	require.NoError(t, err)
	return store
}

func spec(id, queue string, priority int64, scheduledAt time.Time) worker_dto.EnqueueSpec {
	return worker_dto.EnqueueSpec{
		ID:             id,
		Kind:           "test:kind",
		Queue:          queue,
		Payload:        []byte("{}"),
		Priority:       priority,
		MaxAttempts:    3,
		TimeoutSeconds: 300,
		ScheduledAt:    scheduledAt,
	}
}

func TestOtter_EnqueueClaimComplete(t *testing.T) {
	clk := clock.NewMockClock(time.Unix(1000, 0).UTC())
	store := newStore(t, clk)
	ctx := context.Background()

	_, err := store.Enqueue(ctx, spec("j1", "default", 5, clk.Now()))
	require.NoError(t, err)

	state, err := store.GetJobState(ctx, "j1")
	require.NoError(t, err)
	require.Equal(t, string(worker_domain.StatusPending), state.Status)

	claimed, err := store.ClaimDue(ctx, "w1", 10)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.Equal(t, int64(1), claimed[0].Attempt, "attempt charged at claim")
	require.Equal(t, "w1", claimed[0].ClaimedByWorkerID)
	require.Equal(t, clk.Now(), claimed[0].ClaimedAt)

	require.NoError(t, store.MarkCompleted(ctx, "j1", []byte(`"ok"`)))
	state, _ = store.GetJobState(ctx, "j1")
	require.Equal(t, string(worker_domain.StatusCompleted), state.Status)

	versions, _ := store.ListJobVersions(ctx, "j1")
	require.Len(t, versions, 3)
	require.Equal(t, "enqueued", versions[0].Event)
	require.Equal(t, "claimed", versions[1].Event)
	require.Equal(t, "completed", versions[2].Event)
}

func TestOtter_ClaimOrderingPriorityThenScheduled(t *testing.T) {
	clk := clock.NewMockClock(time.Unix(1000, 0).UTC())
	store := newStore(t, clk)
	ctx := context.Background()
	base := clk.Now()

	_, _ = store.Enqueue(ctx, spec("low-early", "default", 1, base.Add(-2*time.Second)))
	_, _ = store.Enqueue(ctx, spec("high-late", "default", 9, base.Add(-1*time.Second)))

	claimed, err := store.ClaimDue(ctx, "w1", 10)
	require.NoError(t, err)
	require.Len(t, claimed, 2)
	require.Equal(t, "high-late", claimed[0].ID, "priority DESC wins over earlier scheduled_at")
	require.Equal(t, "low-early", claimed[1].ID)
}

func TestOtter_FutureJobNotClaimed(t *testing.T) {
	clk := clock.NewMockClock(time.Unix(1000, 0).UTC())
	store := newStore(t, clk)
	ctx := context.Background()

	_, _ = store.Enqueue(ctx, spec("future", "default", 5, clk.Now().Add(time.Hour)))
	claimed, err := store.ClaimDue(ctx, "w1", 10)
	require.NoError(t, err)
	require.Empty(t, claimed, "a job scheduled in the future is not due")
}

func TestOtter_ReclaimStale(t *testing.T) {
	clk := clock.NewMockClock(time.Unix(1000, 0).UTC())
	store := newStore(t, clk)
	ctx := context.Background()

	_, _ = store.Enqueue(ctx, spec("j1", "default", 5, clk.Now()))
	_, _ = store.ClaimDue(ctx, "w1", 10)

	n, err := store.ReclaimStale(ctx, time.Hour)
	require.NoError(t, err)
	require.Equal(t, 0, n)

	n, err = store.ReclaimStale(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	state, _ := store.GetJobState(ctx, "j1")
	require.Equal(t, string(worker_domain.StatusPending), state.Status)
}

func TestOtter_MarkRetryReschedules(t *testing.T) {
	clk := clock.NewMockClock(time.Unix(1000, 0).UTC())
	store := newStore(t, clk)
	ctx := context.Background()

	_, _ = store.Enqueue(ctx, spec("j1", "default", 5, clk.Now()))
	_, _ = store.ClaimDue(ctx, "w1", 10)

	runAt := clk.Now().Add(time.Minute)
	require.NoError(t, store.MarkRetry(ctx, "j1", 1, runAt, "boom"))

	state, _ := store.GetJobState(ctx, "j1")
	require.Equal(t, string(worker_domain.StatusPending), state.Status)
	require.Equal(t, "boom", state.LastError)
	require.Equal(t, runAt, state.ScheduledAt)
}

func TestOtter_Counts(t *testing.T) {
	clk := clock.NewMockClock(time.Unix(1000, 0).UTC())
	store := newStore(t, clk)
	ctx := context.Background()

	_, _ = store.Enqueue(ctx, spec("a", "q1", 5, clk.Now()))
	_, _ = store.Enqueue(ctx, spec("b", "q1", 5, clk.Now()))
	_, _ = store.Enqueue(ctx, spec("c", "q2", 5, clk.Now()))

	pending, _ := store.CountPendingJobs(ctx)
	require.Equal(t, int64(3), pending)

	claimable, _ := store.CountClaimableJobs(ctx)
	require.Equal(t, []worker_domain.ClaimableJobsDepth{
		{Queue: "q1", Count: 2},
		{Queue: "q2", Count: 1},
	}, claimable)

	nonTerminal, _ := store.CountNonTerminalJobs(ctx)
	require.Equal(t, int64(3), nonTerminal)

	_, _ = store.ClaimDue(ctx, "w1", 1)
	require.NoError(t, store.MarkCompleted(ctx, "a", nil))
	nonTerminal, _ = store.CountNonTerminalJobs(ctx)
	require.Equal(t, int64(2), nonTerminal)
}

func TestOtter_GetJobStateNotFound(t *testing.T) {
	store := newStore(t, clock.NewMockClock(time.Unix(1000, 0).UTC()))
	_, err := store.GetJobState(context.Background(), "nope")
	require.ErrorIs(t, err, worker_domain.ErrJobNotFound)
}

func TestOtter_EnqueueManyAllOrNothing(t *testing.T) {
	store := newStore(t, clock.NewMockClock(time.Unix(1000, 0).UTC()))
	ctx := context.Background()
	at := time.Unix(1000, 0).UTC()

	ids, err := store.EnqueueMany(ctx, []worker_dto.EnqueueSpec{
		spec("x", "default", 5, at),
		spec("y", "default", 5, at),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"x", "y"}, ids)

	_, err = store.EnqueueMany(ctx, []worker_dto.EnqueueSpec{
		spec("z", "default", 5, at),
		spec("z", "default", 5, at),
	})
	require.Error(t, err)
	_, err = store.GetJobState(ctx, "z")
	require.ErrorIs(t, err, worker_domain.ErrJobNotFound)
}

func TestOtter_MarkOnMissingJobReturnsNotFound(t *testing.T) {
	store := newStore(t, clock.NewMockClock(time.Unix(1000, 0).UTC()))
	ctx := context.Background()

	require.ErrorIs(t, store.MarkCompleted(ctx, "missing", nil), worker_domain.ErrJobNotFound)
	require.ErrorIs(t, store.MarkFailed(ctx, "missing", "boom"), worker_domain.ErrJobNotFound)
	require.ErrorIs(t, store.MarkRetry(ctx, "missing", 1, time.Unix(1000, 0).UTC(), "boom"), worker_domain.ErrJobNotFound)
}

func TestOtter_EnqueueManyBatchTooLarge(t *testing.T) {
	store := newStore(t, clock.NewMockClock(time.Unix(1000, 0).UTC()))
	at := time.Unix(1000, 0).UTC()

	specs := make([]worker_dto.EnqueueSpec, 10_001)
	for i := range specs {
		specs[i] = spec(fmt.Sprintf("job-%d", i), "default", 5, at)
	}

	_, err := store.EnqueueMany(context.Background(), specs)
	require.ErrorIs(t, err, worker_domain.ErrBatchTooLarge)
}

func TestOtter_ContextCancellationIsHonoured(t *testing.T) {
	store := newStore(t, clock.NewMockClock(time.Unix(1000, 0).UTC()))
	at := time.Unix(1000, 0).UTC()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, enqueueErr := store.Enqueue(ctx, spec("j1", "default", 5, at))
	require.ErrorIs(t, enqueueErr, context.Canceled)

	_, claimErr := store.ClaimDue(ctx, "w1", 10)
	require.ErrorIs(t, claimErr, context.Canceled)

	_, stateErr := store.GetJobState(ctx, "j1")
	require.ErrorIs(t, stateErr, context.Canceled)

	_, countErr := store.CountPendingJobs(ctx)
	require.ErrorIs(t, countErr, context.Canceled)

	_, reclaimErr := store.ReclaimStale(ctx, time.Hour)
	require.ErrorIs(t, reclaimErr, context.Canceled)
}
