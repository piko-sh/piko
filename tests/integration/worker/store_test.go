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

//go:build integration

package worker_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

func TestStore_EnqueueThenGetState(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)

	spec := worker_dto.EnqueueSpec{
		ID:             "job_1",
		Kind:           "email:test_email",
		Queue:          "default",
		Payload:        []byte(`{"key": "value"}`),
		Priority:       5,
		MaxAttempts:    10,
		TimeoutSeconds: 300,
		ScheduledAt:    store.Now(),
	}
	id, err := store.Enqueue(ctx, spec)
	require.NoError(t, err)

	state, err := store.GetJobState(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "pending", state.Status)
	require.Equal(t, spec.Kind, state.Kind)
	require.Equal(t, spec.MaxAttempts, state.MaxAttempts)

	versions, err := store.ListJobVersions(ctx, id)
	require.NoError(t, err)
	require.Len(t, versions, 1, "history has the insert version only")
	require.Equal(t, "enqueued", versions[0].Event)
	require.Equal(t, "pending", versions[0].Status)
}

func TestStore_ClaimMovesPendingToRunning(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)
	_, err := store.Enqueue(ctx, pendingSpec(store, "job_2", "email:test_email"))
	require.NoError(t, err)

	rows, err := store.ClaimDue(ctx, "worker_a", 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "running", rows[0].Status)
	require.Equal(t, int64(1), rows[0].Attempt)
	require.Equal(t, "worker_a", rows[0].ClaimedByWorkerID)

	again, err := store.ClaimDue(ctx, "worker_a", 10)
	require.NoError(t, err)
	require.Empty(t, again, "a running row must not be re-claimable")

	versions, err := store.ListJobVersions(ctx, "job_2")
	require.NoError(t, err)
	require.Len(t, versions, 2, "history has insert + claim")
	claimVersion := versions[len(versions)-1]
	require.Equal(t, "claimed", claimVersion.Event)
	require.Equal(t, "running", claimVersion.Status)
	require.Equal(t, int64(1), claimVersion.Attempt)
}

func TestStore_MarkCompletedAndRetry(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)
	for _, id := range []string{"job_done", "job_retry"} {
		_, err := store.Enqueue(ctx, pendingSpec(store, id, "email:send_welcome"))
		require.NoErrorf(t, err, "enqueue %s", id)
	}
	_, err := store.ClaimDue(ctx, "worker_a", 10)
	require.NoError(t, err)

	require.NoError(t, store.MarkCompleted(ctx, "job_done", nil), "mark completed")
	done, err := store.GetJobState(ctx, "job_done")
	require.NoError(t, err)
	require.Equal(t, "completed", done.Status)

	future := store.Now().Add(time.Hour)
	require.NoError(t, store.MarkRetry(ctx, "job_retry", 1, future, "transient boom"), "mark retry")
	retry, err := store.GetJobState(ctx, "job_retry")
	require.NoError(t, err)
	require.Equal(t, "pending", retry.Status)
	require.Equal(t, "transient boom", retry.LastError)
	rows, err := store.ClaimDue(ctx, "worker_a", 10)
	require.NoError(t, err)
	require.Empty(t, rows, "a future-scheduled retry must not be claimed")
}

func TestStore_RecordsFullHistory(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)
	_, err := store.Enqueue(ctx, pendingSpec(store, "job_1", "email:send_welcome"))
	require.NoError(t, err, "Enqueue")
	_, err = store.ClaimDue(ctx, "worker_a", 10)
	require.NoError(t, err, "ClaimDue")
	require.NoError(t, store.MarkCompleted(ctx, "job_1", []byte(`{"ok":true}`)), "mark completed")

	versions, err := store.ListJobVersions(ctx, "job_1")
	require.NoError(t, err, "ListJobVersions")
	wantEvents := []string{"enqueued", "claimed", "completed"}
	wantStatuses := []string{"pending", "running", "completed"}
	require.Len(t, versions, len(wantEvents), "history has the insert version only")
	for index := range wantEvents {
		require.Equalf(t, wantEvents[index], versions[index].Event, "version %d event", index)
		require.Equalf(t, wantStatuses[index], versions[index].Status, "version %d status", index)
		if index > 0 {
			require.Greaterf(
				t,
				versions[index].Sequence,
				versions[index-1].Sequence,
				"version_sequence must increase incrementally: %d then %d",
				versions[index-1].Sequence,
				versions[index].Sequence,
			)
		}
	}

	_, err = store.GetJobState(ctx, "job_1")
	require.NoError(t, err, "GetJobState")
}

func TestStore_ReclaimStaleReleasesAgedClaim(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)
	_, err := store.Enqueue(ctx, pendingSpec(store, "job_1", "email:send_welcome"))
	require.NoError(t, err, "Enqueue")

	rows, err := store.ClaimDue(ctx, "worker-A", 10)
	require.NoError(t, err, "ClaimDue")
	require.Len(t, rows, 1, "expecting one running row to reclaim")

	fresh, err := store.ReclaimStale(ctx, time.Hour)
	require.NoError(t, err, "ReclaimStale with a cutoff before the claim")
	require.Zero(t, fresh, "a fresh claim is left alone")

	reclaimed, err := store.ReclaimStale(ctx, -time.Hour)
	require.NoError(t, err, "ReclaimStale with a cutoff after the claim")
	require.Equal(t, 1, reclaimed, "a claim older than the cutoff is reclaimed")

	state, err := store.GetJobState(ctx, "job_1")
	require.NoError(t, err, "GetJobState")
	require.Equal(t, "pending", state.Status, "a reclaimed row is back to pending")

	versions, err := store.ListJobVersions(ctx, "job_1")
	require.NoError(t, err, "ListJobVersions")
	require.Equal(t, "recovered", versions[len(versions)-1].Event, "the latest version records the recovery")
}

func TestStore_CountPending(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)

	for _, id := range []string{"a", "b", "c"} {
		_, err := store.Enqueue(ctx, pendingSpec(store, fmt.Sprintf("job_%s", id), "email:send_welcome"))
		require.NoError(t, err, "Enqueue: %s", id)
	}

	n, err := store.CountPendingJobs(ctx)
	require.NoError(t, err, "CountPending")
	require.Equal(t, int64(3), n)
}

func TestStore_ScheduledJobIsNotClaimableBeforeItsTime(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)

	spec := pendingSpec(store, "job_scheduled", "email:test_email")
	spec.ScheduledAt = store.Now().Add(time.Hour)
	_, err := store.Enqueue(ctx, spec)
	require.NoError(t, err, "Enqueue")

	state, err := store.GetJobState(ctx, "job_scheduled")
	require.NoError(t, err, "GetJobState")
	require.Equal(t, "scheduled", state.Status, "a future scheduled_at inserts as scheduled not pending")

	versions, err := store.ListJobVersions(ctx, "job_scheduled")
	require.NoError(t, err, "ListJobVersions")
	require.Len(t, versions, 1, "history has the insert version only")
	require.Equal(t, "enqueued", versions[0].Event)
	require.Equal(t, "scheduled", versions[0].Status)

	rows, err := store.ClaimDue(ctx, "worker_a", 10)
	require.NoError(t, err, "ClaimDue")
	require.Empty(t, rows, "a scheduled row is not claimable before its time")
}

func TestStore_ClaimReturnsHighPriorityFirst(t *testing.T) {
	ctx := t.Context()
	store := newStore(t)

	low := pendingSpec(store, "job_low_priority", "email:test_email")
	low.Priority = 1
	high := pendingSpec(store, "job_high_priority", "email:test_email")
	high.Priority = 10

	_, err := store.Enqueue(ctx, low)
	require.NoError(t, err, "Enqueue low")

	_, err = store.Enqueue(ctx, high)
	require.NoError(t, err, "Enqueue high")

	rows, err := store.ClaimDue(ctx, "worker-A", 1)
	require.NoError(t, err, "ClaimDue")
	require.Len(t, rows, 1, "expecting one running row to reclaim")
	require.Equal(t, "job_high_priority", rows[0].ID, "want high priority job first")
}

func TestStore_HeartbeatDefeatsStaleSweep(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, time.July, 11, 0, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(now)
	store := newStoreWithClock(t, clk)

	_, err := store.Enqueue(ctx, pendingSpec(store, "job_slow", "billing:charge_once"))
	require.NoError(t, err, "Enqueue job slow")

	_, err = store.ClaimDue(ctx, "worker-A", 1)
	require.NoError(t, err, "ClaimDue")

	const visibilityWindow = 30 * time.Second
	clk.Advance(visibilityWindow - 5*time.Second)

	require.NoError(t, store.Heartbeat(ctx, "job_slow", "worker-A"), "Heartbeat")

	clk.Advance(5 * time.Second)
	reclaimed, err := store.ReclaimStale(ctx, visibilityWindow)
	require.NoError(t, err, "ReclaimStale")
	require.Equal(t, 0, reclaimed, "the heartbeat reset the age clock")
	state, err := store.GetJobState(ctx, "job_slow")
	require.NoError(t, err, "GetJobState")
	require.Equal(t, "running", state.Status, "a heartbeated row is not reclaimed")
}

func TestStore_NoHeartbeatIsStaleSwept(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, time.July, 11, 0, 0, 0, 0, time.UTC)
	clk := clock.NewMockClock(now)
	store := newStoreWithClock(t, clk)

	_, err := store.Enqueue(ctx, pendingSpec(store, "job_dead", "billing:charge_once"))
	require.NoError(t, err, "Enqueue job dead")

	_, err = store.ClaimDue(ctx, "worker-A", 1)
	require.NoError(t, err, "ClaimDue")

	const visibilityWindow = 30 * time.Second
	clk.Advance(visibilityWindow - 5*time.Second)

	clk.Advance(5 * time.Second)
	reclaimed, err := store.ReclaimStale(ctx, visibilityWindow)
	require.NoError(t, err, "ReclaimStale")
	require.Equal(t, 1, reclaimed, "an unheartbeated stale row is reclaimed")
	state, err := store.GetJobState(ctx, "job_dead")
	require.NoError(t, err, "GetJobState")
	require.Equal(t, "pending", state.Status, "a heartbeated row is not reclaimed")
}
