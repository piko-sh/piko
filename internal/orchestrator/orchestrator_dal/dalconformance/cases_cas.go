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

package dalconformance

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// releasedID is the task the release case claims, releases and claims again.
	releasedID = "released"

	// testBuild is the build tag the rerun and build-completion cases give their tasks.
	testBuild = "build-1"

	// settledResultBytes is the result value the settle case writes.
	settledResultBytes = 512

	// unsettledInBuild is how many of the build-completion case's tagged tasks are unsettled
	// before one of them settles.
	unsettledInBuild = 4

	// pageLimit is the page size the dispatch, orphan and node lease cases read with.
	pageLimit = 10

	// purgeLimit caps how many tasks the purge case deletes.
	purgeLimit = 100
)

// claim claims a task's attempt for owner at the harness clock's time, failing the test
// on error.
//
// Takes t (*testing.T) which reports test failures.
// Takes id (string) which identifies the task.
// Takes attempt (int) which is the delivered attempt.
// Takes owner (string) which identifies the claiming instance.
//
// Returns *orchestrator_domain.Task which is the claimed task.
func (h *Harness) claim(t *testing.T, id string, attempt int, owner string) *orchestrator_domain.Task {
	t.Helper()
	claimed, err := h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{
		Now: h.Clock.Now(), ID: id, Owner: owner, Attempt: attempt, DueTolerance: 0, Into: nil,
	})
	require.NoError(t, err)
	return claimed
}

// settled returns the state a claimed task settles into at the harness clock's time.
//
// Takes claimed (*orchestrator_domain.Task) which is the claimed task.
// Takes status (orchestrator_domain.TaskStatus) which is the settled status.
//
// Returns *orchestrator_domain.Task which is the new state.
func (h *Harness) settled(claimed *orchestrator_domain.Task, status orchestrator_domain.TaskStatus) *orchestrator_domain.Task {
	next := *claimed
	next.Status = status
	next.UpdatedAt = h.Clock.Now()
	next.OwnerInstance = ""
	return &next
}

// casCases checks the compare-and-set store operations.
//
// Returns []testCase which lists the cases.
func casCases() []testCase {
	return append(append(append(claimCases(), settleCases()...), rerunCases()...), transitionCases()...)
}

// claimCases checks claiming delivered attempts.
//
// Returns []testCase which lists the cases.
func claimCases() []testCase {
	return []testCase{
		{name: "claim/pending-moves-to-processing", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask("claimed"))
			claimed := h.claim(t, "claimed", 0, nodeA)
			assert.Equal(t, orchestrator_domain.StatusProcessing, claimed.Status)
			assert.Equal(t, 1, claimed.Attempt, "a claim counts the attempt it starts")
			assert.Equal(t, nodeA, claimed.OwnerInstance)

			got := h.get(t, "claimed")
			assert.Equal(t, orchestrator_domain.StatusProcessing, got.Status)
			assert.Equal(t, 1, got.Attempt)
			assert.Equal(t, nodeA, got.OwnerInstance)
		}},
		{name: "claim/stale-or-repeated-delivery-is-obsolete", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask("fenced"))
			_, err := h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{Now: h.Clock.Now(), ID: "fenced", Owner: nodeA, Attempt: 3, DueTolerance: 0, Into: nil})
			require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "an attempt the task is not at")
			assert.Equal(t, orchestrator_domain.StatusPending, h.get(t, "fenced").Status, "a refused claim writes nothing")

			h.claim(t, "fenced", 0, nodeA)
			_, err = h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{Now: h.Clock.Now(), ID: "fenced", Owner: "node-b", Attempt: 0, DueTolerance: 0, Into: nil})
			require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "a second delivery of the same attempt")
			assert.Equal(t, nodeA, h.get(t, "fenced").OwnerInstance)

			_, err = h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{Now: h.Clock.Now(), ID: "ghost", Owner: nodeA, Attempt: 0, DueTolerance: 0, Into: nil})
			require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "a task that does not exist")
		}},
		{name: "claim/deferred-before-due-is-premature", run: func(t *testing.T, h *Harness) {
			task := h.newTask("deferred")
			task.Status = orchestrator_domain.StatusRetrying
			task.Attempt = 1
			task.ExecuteAt = h.Clock.Now().Add(time.Minute)
			h.create(t, task)

			early, err := h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{Now: h.Clock.Now(), ID: "deferred", Owner: nodeA, Attempt: 1, DueTolerance: 0, Into: nil})
			require.ErrorIs(t, err, orchestrator_domain.ErrPrematureDelivery)
			require.NotNil(t, early, "a premature claim reports the stored task so the delivery can be deferred")
			assert.True(t, early.ExecuteAt.Equal(task.ExecuteAt), "execute_at %s", early.ExecuteAt)
			assert.Equal(t, orchestrator_domain.StatusRetrying, h.get(t, "deferred").Status)

			claimed, err := h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{
				Now: h.Clock.Now(), ID: "deferred", Owner: nodeA, Attempt: 1, DueTolerance: 2 * time.Minute, Into: nil,
			})
			require.NoError(t, err, "the due tolerance absorbs clock skew between nodes")
			assert.Equal(t, 2, claimed.Attempt)
		}},
		{name: "claim/one-of-many-concurrent-claims-wins", run: checkConcurrentClaims},
	}
}

// checkConcurrentClaims checks that exactly one of many concurrent claims of the same
// attempt wins, and that every loser is told its delivery is obsolete.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkConcurrentClaims(t *testing.T, h *Harness) {
	h.create(t, h.newTask("contended"))
	const contenders = 16
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		won      int
		obsolete int
		failures []error
	)
	for range contenders {
		wg.Go(func() {
			_, err := h.Store.ClaimTask(t.Context(), orchestrator_domain.ClaimRequest{
				Now: h.Clock.Now(), ID: "contended", Owner: "node", Attempt: 0, DueTolerance: 0, Into: nil,
			})
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				won++
			case errors.Is(err, orchestrator_domain.ErrObsoleteDelivery):
				obsolete++
			default:
				failures = append(failures, err)
			}
		})
	}
	wg.Wait()
	require.Empty(t, failures, "losing a claim race must be reported as obsolete, not as a store failure")
	assert.Equal(t, 1, won, "exactly one executor may hold a task")
	assert.Equal(t, contenders-1, obsolete)
}

// settleCases checks settling claimed tasks.
//
// Returns []testCase which lists the cases.
func settleCases() []testCase {
	return []testCase{
		{name: "settle/writes-the-claimed-attempt", run: func(t *testing.T, h *Harness) {
			task := h.newTask("settled")
			task.DeduplicationKey = "dedup:settled"
			h.create(t, task)
			claimed := h.claim(t, "settled", 0, nodeA)

			next := h.settled(claimed, orchestrator_domain.StatusComplete)
			next.Result = map[string]any{"bytes": float64(settledResultBytes)}
			outcome, err := h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: next, Successor: nil})
			require.NoError(t, err)
			assert.Nil(t, outcome.Successor)
			assert.Nil(t, outcome.Rerun)

			got := h.get(t, "settled")
			assert.Equal(t, orchestrator_domain.StatusComplete, got.Status)
			assert.Equal(t, 1, got.Attempt)
			assert.Empty(t, got.OwnerInstance, "a settled task has no owner")
			assert.Equal(t, map[string]any{"bytes": float64(settledResultBytes)}, got.Result)

			again := h.newTask("again")
			again.DeduplicationKey = "dedup:settled"
			require.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), again), "settling frees the deduplication key")
		}},
		{name: "settle/lost-claim-is-obsolete", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask("lost"))
			claimed := h.claim(t, "lost", 0, nodeA)

			stale := h.settled(claimed, orchestrator_domain.StatusComplete)
			stale.Attempt = 0
			_, err := h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: stale, Successor: nil})
			require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "a settle for an attempt the task has moved past")

			recovered := h.settled(claimed, orchestrator_domain.StatusRetrying)
			_, err = h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
				Task: recovered, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusProcessing},
			})
			require.NoError(t, err)
			_, err = h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: h.settled(claimed, orchestrator_domain.StatusComplete), Successor: nil})
			require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "a settle after the task was recovered")
			assert.Equal(t, orchestrator_domain.StatusRetrying, h.get(t, "lost").Status, "a refused settle writes nothing")
		}},
		{name: "settle/creates-successor-unless-its-key-is-held", run: checkSettleSuccessor},
	}
}

// checkSettleSuccessor checks that a settle creates its successor, and skips it without
// failing when another active task holds the successor's deduplication key.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkSettleSuccessor(t *testing.T, h *Harness) {
	h.create(t, h.newTask("recurring"))
	claimed := h.claim(t, "recurring", 0, nodeA)
	successor := h.newTask("recurring-next")
	successor.DeduplicationKey = "dedup:recurring"
	successor.Status = orchestrator_domain.StatusScheduled
	successor.ExecuteAt = h.Clock.Now().Add(time.Hour)
	successor.ScheduledExecuteAt = successor.ExecuteAt
	outcome, err := h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{
		Task: h.settled(claimed, orchestrator_domain.StatusComplete), Successor: successor,
	})
	require.NoError(t, err)
	require.NotNil(t, outcome.Successor)
	assert.Equal(t, "recurring-next", outcome.Successor.ID)
	assert.Equal(t, orchestrator_domain.StatusScheduled, h.get(t, "recurring-next").Status)

	h.create(t, h.newTask("other"))
	other := h.claim(t, "other", 0, nodeA)
	duplicate := h.newTask("recurring-duplicate")
	duplicate.DeduplicationKey = "dedup:recurring"
	outcome, err = h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{
		Task: h.settled(other, orchestrator_domain.StatusComplete), Successor: duplicate,
	})
	require.NoError(t, err, "a successor whose key is held never fails the settle")
	assert.Nil(t, outcome.Successor)
	assert.Equal(t, orchestrator_domain.StatusComplete, h.get(t, "other").Status)
	rows, err := h.Store.GetTasksByID(t.Context(), []string{"recurring-duplicate"})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// rerunCases checks reruns requested while a task is active.
//
// Returns []testCase which lists the cases.
func rerunCases() []testCase {
	return []testCase{
		{name: "rerun/runs-once-the-task-settles", run: func(t *testing.T, h *Harness) {
			task := h.newTask("busy")
			task.DeduplicationKey = "dedup:busy"
			task.BuildTag = testBuild
			h.create(t, task)
			claimed := h.claim(t, "busy", 0, nodeA)

			for _, id := range []string{"busy-rerun-1", "busy-rerun-2"} {
				recorded, err := h.Store.RequestRerun(t.Context(), "dedup:busy", orchestrator_domain.RerunRequest{
					ID: id, Payload: map[string]any{"version": id}, Config: task.Config,
				})
				require.NoError(t, err)
				assert.True(t, recorded)
			}

			outcome, err := h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: h.settled(claimed, orchestrator_domain.StatusComplete), Successor: nil})
			require.NoError(t, err)
			require.NotNil(t, outcome.Rerun)
			assert.Equal(t, "busy-rerun-2", outcome.Rerun.ID, "only the latest request is kept")

			rerun := h.get(t, "busy-rerun-2")
			assert.Equal(t, orchestrator_domain.StatusPending, rerun.Status)
			assert.Equal(t, "dedup:busy", rerun.DeduplicationKey, "the rerun holds the key the settled task released")
			assert.Equal(t, testBuild, rerun.BuildTag)
			assert.Equal(t, map[string]any{"version": "busy-rerun-2"}, rerun.Payload)
			assert.Nil(t, h.get(t, "busy").Rerun, "the request is cleared once its rerun exists")
		}},
		{name: "rerun/no-active-holder-is-not-recorded", run: func(t *testing.T, h *Harness) {
			recorded, err := h.Store.RequestRerun(t.Context(), "dedup:nobody", orchestrator_domain.RerunRequest{ID: "orphan-rerun", Payload: nil, Config: orchestrator_domain.TaskConfig{}})
			require.NoError(t, err)
			assert.False(t, recorded, "the caller must dispatch the work itself")
		}},
		{name: "rerun/waits-out-a-retry", run: func(t *testing.T, h *Harness) {
			task := h.newTask("flaky")
			task.DeduplicationKey = "dedup:flaky"
			h.create(t, task)
			claimed := h.claim(t, "flaky", 0, nodeA)
			_, err := h.Store.RequestRerun(t.Context(), "dedup:flaky", orchestrator_domain.RerunRequest{ID: "flaky-rerun", Payload: nil, Config: task.Config})
			require.NoError(t, err)

			retry := h.settled(claimed, orchestrator_domain.StatusRetrying)
			outcome, err := h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: retry, Successor: nil})
			require.NoError(t, err)
			assert.Nil(t, outcome.Rerun, "a retrying task still holds its key, so its rerun waits")

			claimed = h.claim(t, "flaky", 1, nodeA)
			outcome, err = h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: h.settled(claimed, orchestrator_domain.StatusFailed), Successor: nil})
			require.NoError(t, err)
			require.NotNil(t, outcome.Rerun, "a failed task's rerun still runs: its inputs changed")
			assert.Equal(t, "flaky-rerun", outcome.Rerun.ID)
		}},
	}
}

// transitionCases checks status transitions outside a claim.
//
// Returns []testCase which lists the cases.
func transitionCases() []testCase {
	return []testCase{
		{name: "transition/fenced-by-attempt-and-status", run: checkTransitionFenced},
		{name: "transition/release-keeps-the-attempt-and-frees-the-owner", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask(releasedID))
			claimed := h.claim(t, releasedID, 0, nodeA)
			released := h.settled(claimed, orchestrator_domain.StatusPending)
			outcome, err := h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
				Task: released, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusProcessing},
			})
			require.NoError(t, err)
			assert.Nil(t, outcome.Rerun, "a move to an active status creates no rerun")
			got := h.get(t, releasedID)
			assert.Equal(t, orchestrator_domain.StatusPending, got.Status)
			assert.Equal(t, 1, got.Attempt)
			assert.Empty(t, got.OwnerInstance)
			h.claim(t, releasedID, 1, "node-b")
		}},
		{name: "transition/to-terminal-creates-pending-rerun", run: func(t *testing.T, h *Harness) {
			task := h.newTask("superseded")
			task.DeduplicationKey = "dedup:superseded"
			h.create(t, task)
			_, err := h.Store.RequestRerun(t.Context(), "dedup:superseded", orchestrator_domain.RerunRequest{ID: "superseded-rerun", Payload: nil, Config: task.Config})
			require.NoError(t, err)

			done := h.newTask("superseded")
			done.Status = orchestrator_domain.StatusComplete
			outcome, err := h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
				Task: done, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusPending},
			})
			require.NoError(t, err)
			assert.Equal(t, orchestrator_domain.StatusPending, h.get(t, "superseded-rerun").Status)
			require.NotNil(t, outcome.Rerun, "the rerun is reported so the caller can publish it")
			assert.Equal(t, "superseded-rerun", outcome.Rerun.ID)
			assert.Equal(t, "dedup:superseded", outcome.Rerun.DeduplicationKey)
			assert.Equal(t, orchestrator_domain.StatusPending, outcome.Rerun.Status)
		}},
	}
}

// checkTransitionFenced checks that a transition is refused when the stored task is not
// in an expected status or not at the expected attempt, and that a task the store does
// not hold is reported as not found.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkTransitionFenced(t *testing.T, h *Harness) {
	h.create(t, h.newTask("moved"))
	failed := h.newTask("moved")
	failed.Status = orchestrator_domain.StatusFailed
	failed.LastError = "could not publish"

	_, err := h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
		Task: failed, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusProcessing},
	})
	require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "the task is not in an expected status")

	stale := *failed
	stale.Attempt = 2
	_, err = h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
		Task: &stale, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusPending},
	})
	require.ErrorIs(t, err, orchestrator_domain.ErrObsoleteDelivery, "the task is not at the expected attempt")
	assert.Equal(t, orchestrator_domain.StatusPending, h.get(t, "moved").Status)

	_, err = h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
		Task: failed, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusPending},
	})
	require.NoError(t, err)
	got := h.get(t, "moved")
	assert.Equal(t, orchestrator_domain.StatusFailed, got.Status)
	assert.Equal(t, "could not publish", got.LastError)

	_, err = h.Store.TransitionTask(t.Context(), orchestrator_domain.Transition{
		Task: h.newTask("ghost"), From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusPending},
	})
	require.ErrorIs(t, err, orchestrator_domain.ErrTaskNotFound)
}

// completionCases checks the build-completion queries.
//
// Returns []testCase which lists the cases.
func completionCases() []testCase {
	return []testCase{
		{name: "build/count-unsettled-foreground-tasks", run: func(t *testing.T, h *Harness) {
			for i, status := range []orchestrator_domain.TaskStatus{
				orchestrator_domain.StatusPending, orchestrator_domain.StatusScheduled,
				orchestrator_domain.StatusProcessing, orchestrator_domain.StatusRetrying,
				orchestrator_domain.StatusComplete, orchestrator_domain.StatusFailed,
			} {
				task := h.newTask(string(status))
				task.Status = status
				task.BuildTag = testBuild
				task.Attempt = i % 2
				h.create(t, task)
			}
			background := h.newTask("background")
			background.BuildTag = testBuild
			background.Background = true
			other := h.newTask("other-build")
			other.BuildTag = "build-2"
			h.create(t, background, other, h.newTask("untagged"))

			count, err := h.Store.CountUnsettled(t.Context(), testBuild)
			require.NoError(t, err)
			assert.Equal(t, int64(unsettledInBuild), count, "scheduled, pending, processing and retrying tasks of the build, not background ones")
			count, err = h.Store.CountUnsettled(t.Context(), "")
			require.NoError(t, err)
			assert.Zero(t, count, "untagged tasks belong to no build")

			h.claim(t, string(orchestrator_domain.StatusPending), 0, nodeA)
			claimed := h.get(t, string(orchestrator_domain.StatusPending))
			_, err = h.Store.SettleTask(t.Context(), orchestrator_domain.Settlement{Task: h.settled(claimed, orchestrator_domain.StatusComplete), Successor: nil})
			require.NoError(t, err)
			count, err = h.Store.CountUnsettled(t.Context(), testBuild)
			require.NoError(t, err)
			assert.Equal(t, int64(unsettledInBuild-1), count)
		}},
		{name: "build/list-failed-for-build", run: func(t *testing.T, h *Harness) {
			for _, spec := range []struct{ id, tag string }{{"f1", testBuild}, {"f2", testBuild}, {"f3", "build-2"}} {
				task := h.newTask(spec.id)
				task.Status = orchestrator_domain.StatusFailed
				task.BuildTag = spec.tag
				task.IsFatal = spec.id == "f2"
				task.LastError = "failed " + spec.id
				h.create(t, task)
			}
			failed, err := h.Store.ListFailedForBuild(t.Context(), testBuild)
			require.NoError(t, err)
			ids := make(map[string]*orchestrator_domain.Task, len(failed))
			for _, task := range failed {
				ids[task.ID] = task
			}
			require.Len(t, ids, 2)
			require.Contains(t, ids, "f2")
			assert.True(t, ids["f2"].IsFatal)
			assert.Equal(t, "failed f1", ids["f1"].LastError)
		}},
	}
}

// maintenanceCases checks dispatch paging, orphans, node leases and retention.
//
// Returns []testCase which lists the cases.
func maintenanceCases() []testCase {
	return []testCase{
		{name: "dispatch/pages-due-tasks-in-order", run: checkDispatchPageOrder},
		{name: "dispatch/skips-recently-published", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask("p-sent"), h.newTask("p-new"))
			require.NoError(t, h.Store.MarkPublished(t.Context(), []string{"p-sent"}, h.Clock.Now()))

			page, err := h.Store.DispatchablePage(t.Context(), orchestrator_domain.DispatchQuery{
				Now: h.Clock.Now(), AfterTime: time.Time{}, PublishedBefore: h.Clock.Now(), AfterID: "", Limit: pageLimit,
			})
			require.NoError(t, err)
			require.Len(t, page, 1)
			assert.Equal(t, "p-new", page[0].ID)

			h.Clock.Advance(time.Minute)
			page, err = h.Store.DispatchablePage(t.Context(), orchestrator_domain.DispatchQuery{
				Now: h.Clock.Now(), AfterTime: time.Time{}, PublishedBefore: h.Clock.Now(), AfterID: "", Limit: pageLimit,
			})
			require.NoError(t, err)
			assert.Len(t, page, 2, "a task published before the cutoff may be published again")
		}},
		{name: "orphans/processing-without-a-live-lease", run: func(t *testing.T, h *Harness) {
			ctx := t.Context()
			require.NoError(t, h.Store.RegisterNode(ctx, orchestrator_domain.NodeLease{
				Instance: nodeA, NodeID: "host", StartedAt: h.Clock.Now(), RenewedAt: h.Clock.Now(),
				ExpiresAt: h.Clock.Now().Add(time.Minute),
			}))
			h.create(t, h.newTask("owned"))
			h.claim(t, "owned", 0, nodeA)
			h.newProcessing(t, "ownerless", 1)
			h.Clock.Advance(30 * time.Second)

			query := orchestrator_domain.OrphanQuery{Now: h.Clock.Now(), StaleBefore: h.Clock.Now(), Limit: pageLimit}
			orphans, err := h.Store.ListOrphanedTasks(ctx, query)
			require.NoError(t, err)
			require.Len(t, orphans, 1, "a task whose owner holds a live lease is not an orphan")
			assert.Equal(t, "ownerless", orphans[0].ID)

			h.Clock.Advance(time.Minute)
			query = orchestrator_domain.OrphanQuery{Now: h.Clock.Now(), StaleBefore: h.Clock.Now(), Limit: pageLimit}
			orphans, err = h.Store.ListOrphanedTasks(ctx, query)
			require.NoError(t, err)
			assert.Len(t, orphans, 2, "an expired lease orphans its owner's tasks")

			query.StaleBefore = Epoch
			orphans, err = h.Store.ListOrphanedTasks(ctx, query)
			require.NoError(t, err)
			assert.Empty(t, orphans, "only tasks written before the cutoff are orphans")
		}},
		{name: "nodes/lease-lifecycle", run: checkNodeLeaseLifecycle},
		{name: "purge/terminal-tasks-before-cutoff", run: checkPurgeTerminal},
	}
}

// checkDispatchPageOrder checks that dispatch pages hold only due pending and retrying
// tasks, ordered by execution time and then ID, and resume after a cursor.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkDispatchPageOrder(t *testing.T, h *Harness) {
	now := h.Clock.Now()
	for _, spec := range []struct {
		id     string
		status orchestrator_domain.TaskStatus
		at     time.Duration
	}{
		{"d-b", orchestrator_domain.StatusPending, -time.Minute},
		{"d-a", orchestrator_domain.StatusPending, -time.Minute},
		{"d-c", orchestrator_domain.StatusRetrying, -time.Second},
		{"d-later", orchestrator_domain.StatusPending, time.Hour},
		{"d-busy", orchestrator_domain.StatusProcessing, -time.Hour},
		{"d-done", orchestrator_domain.StatusComplete, -time.Hour},
	} {
		task := h.newTask(spec.id)
		task.Status = spec.status
		task.ExecuteAt = now.Add(spec.at)
		h.create(t, task)
	}

	first, err := h.Store.DispatchablePage(t.Context(), orchestrator_domain.DispatchQuery{Now: now, AfterTime: time.Time{}, PublishedBefore: time.Time{}, AfterID: "", Limit: 2})
	require.NoError(t, err)
	require.Len(t, first, 2)
	assert.Equal(t, []string{"d-a", "d-b"}, []string{first[0].ID, first[1].ID}, "ordered by execution time, then ID")

	last := first[len(first)-1]
	rest, err := h.Store.DispatchablePage(t.Context(), orchestrator_domain.DispatchQuery{
		Now: now, AfterTime: last.ExecuteAt, PublishedBefore: time.Time{}, AfterID: last.ID, Limit: pageLimit,
	})
	require.NoError(t, err)
	require.Len(t, rest, 1, "only due pending and retrying tasks are dispatchable")
	assert.Equal(t, "d-c", rest[0].ID)
}

// checkNodeLeaseLifecycle checks registering, renewing, expiring and deregistering a node
// lease, and touching the tasks its instance owns.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkNodeLeaseLifecycle(t *testing.T, h *Harness) {
	ctx := t.Context()
	lease := orchestrator_domain.NodeLease{
		Instance: nodeA, NodeID: "host", StartedAt: h.Clock.Now(), RenewedAt: h.Clock.Now(),
		ExpiresAt: h.Clock.Now().Add(time.Minute),
	}
	require.NoError(t, h.Store.RegisterNode(ctx, lease))
	h.create(t, h.newTask("n1"), h.newTask("n2"), h.newTask("n3"))
	h.claim(t, "n1", 0, nodeA)
	h.claim(t, "n2", 0, nodeA)
	h.claim(t, "n3", 0, "node-b")
	touched, err := h.Store.TouchOwned(ctx, nodeA, h.Clock.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), touched)

	h.Clock.Advance(50 * time.Second)
	lease.RenewedAt = h.Clock.Now()
	lease.ExpiresAt = h.Clock.Now().Add(time.Minute)
	require.NoError(t, h.Store.RenewNode(ctx, lease))
	h.Clock.Advance(30 * time.Second)
	expired, err := h.Store.ListExpiredNodes(ctx, h.Clock.Now(), pageLimit)
	require.NoError(t, err)
	assert.Empty(t, expired, "a renewed lease has not lapsed")

	h.Clock.Advance(time.Minute)
	expired, err = h.Store.ListExpiredNodes(ctx, h.Clock.Now(), pageLimit)
	require.NoError(t, err)
	assert.Equal(t, []string{nodeA}, expired)

	require.NoError(t, h.Store.DeregisterNode(ctx, nodeA))
	require.ErrorIs(t, h.Store.RenewNode(ctx, lease), orchestrator_domain.ErrLeaseLost)
}

// checkPurgeTerminal checks that a purge deletes only terminal tasks written before the
// cutoff, keeping those a pending receipt still waits on.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkPurgeTerminal(t *testing.T, h *Harness) {
	ctx := t.Context()
	for _, spec := range []struct {
		id     string
		status orchestrator_domain.TaskStatus
	}{
		{"old-done", orchestrator_domain.StatusComplete},
		{"old-failed", orchestrator_domain.StatusFailed},
		{"old-waited-on", orchestrator_domain.StatusComplete},
		{"old-pending", orchestrator_domain.StatusPending},
	} {
		task := h.newTask(spec.id)
		task.Status = spec.status
		h.create(t, task)
	}
	require.NoError(t, h.Store.CreateWorkflowReceipt(ctx, "receipt", "old-waited-on", nodeA))
	h.Clock.Advance(time.Hour)
	fresh := h.newTask("fresh-done")
	fresh.Status = orchestrator_domain.StatusComplete
	h.create(t, fresh)

	purged, err := h.Store.PurgeTerminal(ctx, h.Clock.Now().Add(-30*time.Minute), purgeLimit)
	require.NoError(t, err)
	assert.Equal(t, 2, purged)
	rows, err := h.Store.GetTasksByID(ctx, []string{"old-done", "old-failed", "old-waited-on", "old-pending", "fresh-done"})
	require.NoError(t, err)
	kept := make([]string, 0, len(rows))
	for _, row := range rows {
		kept = append(kept, row.ID)
	}
	assert.ElementsMatch(t, []string{"old-waited-on", "old-pending", "fresh-done"}, kept,
		"active tasks, recent tasks and tasks a receipt still waits on are kept")
}
