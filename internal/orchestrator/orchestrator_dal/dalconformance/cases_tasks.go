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
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// taskTimeout is the timeout every conformance task is configured with.
	taskTimeout = 30 * time.Second

	// taskMaxRetries is the retry limit every conformance task is configured with.
	taskMaxRetries = 3

	// updatedResultBytes is the result value the update case writes.
	updatedResultBytes = 1024

	// waitingStatuses counts pending, scheduled, and retrying task statuses.
	waitingStatuses = 3
)

// newTask returns a pending task stamped at the harness clock's current time.
//
// Takes id (string) which is the task and workflow ID.
//
// Returns *orchestrator_domain.Task which is ready to create.
func (h *Harness) newTask(id string) *orchestrator_domain.Task {
	now := h.Clock.Now()
	return &orchestrator_domain.Task{
		ID:         id,
		WorkflowID: id,
		Executor:   "conformance.executor",
		Status:     orchestrator_domain.StatusPending,
		Payload:    map[string]any{"artefactID": "assets/" + id + ".svg", "profile": "minified"},
		Config: orchestrator_domain.TaskConfig{
			Timeout:    taskTimeout,
			MaxRetries: taskMaxRetries,
			Priority:   orchestrator_domain.PriorityHigh,
		},
		ExecuteAt:          now,
		CreatedAt:          now,
		UpdatedAt:          now,
		ScheduledExecuteAt: time.Time{},
		Result:             nil,
		LastError:          "",
		DeduplicationKey:   "",
		BuildTag:           "",
		Attempt:            0,
		IsFatal:            false,
		Background:         false,
		OwnerInstance:      "",
		LastPublishedAt:    time.Time{},
		Rerun:              nil,
	}
}

// get reads one task, failing the test when it is missing.
//
// Takes t (*testing.T) which reports test failures.
// Takes id (string) which identifies the task.
//
// Returns *orchestrator_domain.Task which is the stored task.
func (h *Harness) get(t *testing.T, id string) *orchestrator_domain.Task {
	t.Helper()
	rows, err := h.Store.GetTasksByID(t.Context(), []string{id})
	require.NoError(t, err)
	require.Len(t, rows, 1, "task %s should exist", id)
	return rows[0]
}

// create stores tasks, failing the test on error.
//
// Takes t (*testing.T) which reports test failures.
// Takes tasks (...*orchestrator_domain.Task) which are created in order.
func (h *Harness) create(t *testing.T, tasks ...*orchestrator_domain.Task) {
	t.Helper()
	for _, task := range tasks {
		require.NoError(t, h.Store.CreateTask(t.Context(), task))
	}
}

// createCases checks creating and updating tasks.
//
// Returns []testCase which lists the cases.
func createCases() []testCase {
	return []testCase{
		{name: "create/round-trips-core-fields", run: func(t *testing.T, h *Harness) {
			task := h.newTask("core")
			task.DeduplicationKey = "dedup:core"
			h.create(t, task)

			got := h.get(t, "core")
			assert.Equal(t, task.ID, got.ID)
			assert.Equal(t, task.WorkflowID, got.WorkflowID)
			assert.Equal(t, task.Executor, got.Executor)
			assert.Equal(t, orchestrator_domain.StatusPending, got.Status)
			assert.Equal(t, 0, got.Attempt)
			assert.Equal(t, task.Payload, got.Payload)
			assert.Equal(t, task.Config, got.Config)
			assert.True(t, got.ExecuteAt.Equal(Epoch), "execute_at %s", got.ExecuteAt)
			assert.True(t, got.CreatedAt.Equal(Epoch), "created_at %s", got.CreatedAt)
		}},
		{name: "create/round-trips-build-tag-fatal-schedule", run: func(t *testing.T, h *Harness) {
			task := h.newTask("tagged")
			task.BuildTag = "build-7"
			task.Status = orchestrator_domain.StatusScheduled
			task.ScheduledExecuteAt = Epoch.Add(time.Hour)
			task.ExecuteAt = task.ScheduledExecuteAt
			h.create(t, task)

			task.Status = orchestrator_domain.StatusFailed
			task.IsFatal = true
			task.LastError = "capability not found"
			require.NoError(t, h.Store.UpdateTask(t.Context(), task))

			got := h.get(t, "tagged")
			assert.Equal(t, "build-7", got.BuildTag)
			assert.True(t, got.IsFatal)
			assert.True(t, got.ScheduledExecuteAt.Equal(Epoch.Add(time.Hour)), "scheduled_execute_at %s", got.ScheduledExecuteAt)
		}},
		{name: "create/keeps-dedup-key", run: func(t *testing.T, h *Harness) {
			task := h.newTask("first")
			task.DeduplicationKey = "dedup:shared"
			h.create(t, task)

			assert.Equal(t, "dedup:shared", h.get(t, "first").DeduplicationKey)
			second := h.newTask("second")
			second.DeduplicationKey = "dedup:shared"
			assert.ErrorIs(t, h.Store.CreateTaskWithDedup(t.Context(), second), orchestrator_domain.ErrDuplicateTask,
				"a key created without the dedup check still holds")
		}},
		{name: "update/persists-changes", run: checkUpdatePersists},
		{name: "update/missing-row-is-not-found", run: func(t *testing.T, h *Harness) {
			err := h.Store.UpdateTask(t.Context(), h.newTask("ghost"))
			require.Error(t, err, "updating a task that does not exist must fail")
			rows, readErr := h.Store.GetTasksByID(t.Context(), []string{"ghost"})
			require.NoError(t, readErr)
			assert.Empty(t, rows, "a failed update must not create the task")
		}},
	}
}

// checkUpdatePersists checks that an update writes the status, attempt, error, result and
// settings of an existing task.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkUpdatePersists(t *testing.T, h *Harness) {
	task := h.newTask("update")
	h.create(t, task)

	task.Status = orchestrator_domain.StatusComplete
	task.Attempt = 2
	task.LastError = "earlier failure"
	task.Result = map[string]any{"bytes": float64(updatedResultBytes)}
	task.Config.Priority = orchestrator_domain.PriorityLow
	require.NoError(t, h.Store.UpdateTask(t.Context(), task))

	got := h.get(t, "update")
	assert.Equal(t, orchestrator_domain.StatusComplete, got.Status)
	assert.Equal(t, 2, got.Attempt)
	assert.Equal(t, "earlier failure", got.LastError)
	assert.Equal(t, map[string]any{"bytes": float64(updatedResultBytes)}, got.Result)
	assert.Equal(t, orchestrator_domain.PriorityLow, got.Config.Priority)
}

// dedupCases checks deduplication keys.
//
// Returns []testCase which lists the cases.
func dedupCases() []testCase {
	return []testCase{
		{name: "dedup/blocks-active-duplicate", run: func(t *testing.T, h *Harness) {
			for _, status := range []orchestrator_domain.TaskStatus{
				orchestrator_domain.StatusPending,
				orchestrator_domain.StatusScheduled,
				orchestrator_domain.StatusProcessing,
				orchestrator_domain.StatusRetrying,
			} {
				key := "dedup:" + string(status)
				holder := h.newTask("holder-" + string(status))
				holder.DeduplicationKey = key
				holder.Status = status
				require.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), holder))

				duplicate := h.newTask("duplicate-" + string(status))
				duplicate.DeduplicationKey = key
				assert.ErrorIs(t, h.Store.CreateTaskWithDedup(t.Context(), duplicate), orchestrator_domain.ErrDuplicateTask,
					"a %s task holds its key", status)
			}
		}},
		{name: "dedup/allows-after-terminal", run: func(t *testing.T, h *Harness) {
			for _, status := range []orchestrator_domain.TaskStatus{orchestrator_domain.StatusComplete, orchestrator_domain.StatusFailed} {
				key := "dedup:" + string(status)
				done := h.newTask("done-" + string(status))
				done.DeduplicationKey = key
				require.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), done))
				done.Status = status
				require.NoError(t, h.Store.UpdateTask(t.Context(), done))

				next := h.newTask("next-" + string(status))
				next.DeduplicationKey = key
				assert.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), next), "a %s task frees its key", status)
			}
		}},
		{name: "dedup/empty-key-never-blocks", run: func(t *testing.T, h *Harness) {
			require.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), h.newTask("a")))
			require.NoError(t, h.Store.CreateTaskWithDedup(t.Context(), h.newTask("b")))
			assert.Len(t, mustGet(t, h, "a", "b"), 2)
		}},
		{name: "dedup/atomic-under-concurrency", run: checkConcurrentDedup},
	}
}

// checkConcurrentDedup checks that exactly one of many concurrent creates sharing a
// deduplication key succeeds, and that every other is reported as a duplicate.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkConcurrentDedup(t *testing.T, h *Harness) {
	const contenders = 16
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		created    int
		duplicates int
		failures   []error
	)
	for i := range contenders {
		wg.Go(func() {
			task := h.newTask(fmt.Sprintf("contender-%02d", i))
			task.DeduplicationKey = "dedup:contended"
			err := h.Store.CreateTaskWithDedup(context.Background(), task)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				created++
			case errors.Is(err, orchestrator_domain.ErrDuplicateTask):
				duplicates++
			default:
				failures = append(failures, err)
			}
		})
	}
	wg.Wait()
	assert.Empty(t, failures, "contention must resolve as a duplicate, not an error")
	assert.Equal(t, 1, created, "exactly one contender holds the key")
	assert.Equal(t, contenders-1, duplicates)
}

// readCases checks reads, counts and workflow status.
//
// Returns []testCase which lists the cases.
func readCases() []testCase {
	return []testCase{
		{name: "read/get-by-id-skips-missing", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask("present"))
			rows, err := h.Store.GetTasksByID(t.Context(), []string{"absent", "present"})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, "present", rows[0].ID)
		}},
		{name: "read/get-by-id-returns-copies", run: func(t *testing.T, h *Harness) {
			h.create(t, h.newTask("copied"))
			first := h.get(t, "copied")
			first.Status = orchestrator_domain.StatusFailed
			first.Payload["artefactID"] = "mutated"
			got := h.get(t, "copied")
			assert.Equal(t, orchestrator_domain.StatusPending, got.Status)
			assert.Equal(t, "assets/copied.svg", got.Payload["artefactID"])
		}},
		{name: "workflow/incomplete-until-all-terminal", run: checkWorkflowCompletion},
		{name: "workflow/unknown-is-complete", run: func(t *testing.T, h *Harness) {
			complete, err := h.Store.GetWorkflowStatus(t.Context(), "never-dispatched")
			require.NoError(t, err)
			assert.True(t, complete, "a workflow with no tasks has nothing left to wait for")
		}},
		{name: "count/pending-covers-unsettled-statuses", run: func(t *testing.T, h *Harness) {
			for _, status := range []orchestrator_domain.TaskStatus{
				orchestrator_domain.StatusPending,
				orchestrator_domain.StatusScheduled,
				orchestrator_domain.StatusRetrying,
				orchestrator_domain.StatusProcessing,
				orchestrator_domain.StatusComplete,
				orchestrator_domain.StatusFailed,
			} {
				task := h.newTask("count-" + string(status))
				task.Status = status
				h.create(t, task)
			}
			count, err := h.Store.PendingTaskCount(t.Context())
			require.NoError(t, err)
			assert.Equal(t, int64(waitingStatuses), count, "pending, scheduled and retrying tasks are waiting to run")
		}},
		{name: "failed/list-includes-build-tag-and-error", run: func(t *testing.T, h *Harness) {
			task := h.newTask("broken")
			task.BuildTag = "build-9"
			h.create(t, task)
			task.Status = orchestrator_domain.StatusFailed
			task.LastError = "compile failed"
			task.IsFatal = true
			require.NoError(t, h.Store.UpdateTask(t.Context(), task))
			h.create(t, h.newTask("healthy"))

			failed, err := h.Store.ListFailedTasks(t.Context())
			require.NoError(t, err)
			require.Len(t, failed, 1)
			assert.Equal(t, "broken", failed[0].ID)
			assert.Equal(t, "compile failed", failed[0].LastError)
			assert.Equal(t, "build-9", failed[0].BuildTag)
			assert.True(t, failed[0].IsFatal)
		}},
	}
}

// checkWorkflowCompletion checks that a workflow is incomplete until every one of its
// tasks has reached a terminal status.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkWorkflowCompletion(t *testing.T, h *Harness) {
	first, second := h.newTask("wf-a"), h.newTask("wf-b")
	second.WorkflowID = first.WorkflowID
	h.create(t, first, second)

	complete, err := h.Store.GetWorkflowStatus(t.Context(), first.WorkflowID)
	require.NoError(t, err)
	assert.False(t, complete)

	first.Status = orchestrator_domain.StatusComplete
	require.NoError(t, h.Store.UpdateTask(t.Context(), first))
	complete, err = h.Store.GetWorkflowStatus(t.Context(), first.WorkflowID)
	require.NoError(t, err)
	assert.False(t, complete, "one task is still pending")

	second.Status = orchestrator_domain.StatusFailed
	require.NoError(t, h.Store.UpdateTask(t.Context(), second))
	complete, err = h.Store.GetWorkflowStatus(t.Context(), first.WorkflowID)
	require.NoError(t, err)
	assert.True(t, complete, "complete and failed are both terminal")
}

// atomicCases checks RunAtomic.
//
// Returns []testCase which lists the cases.
func atomicCases() []testCase {
	errAbort := errors.New("conformance: abort")
	return []testCase{
		{name: "atomic/commit-applies-all", run: func(t *testing.T, h *Harness) {
			err := h.Store.RunAtomic(t.Context(), func(ctx context.Context, store orchestrator_domain.TaskStore) error {
				if err := store.CreateTask(ctx, h.newTask("tx-a")); err != nil {
					return err
				}
				return store.CreateTask(ctx, h.newTask("tx-b"))
			})
			require.NoError(t, err)
			assert.Len(t, mustGet(t, h, "tx-a", "tx-b"), 2)
		}},
		{name: "atomic/error-rolls-back", run: func(t *testing.T, h *Harness) {
			existing := h.newTask("existing")
			h.create(t, existing)

			err := h.Store.RunAtomic(t.Context(), func(ctx context.Context, store orchestrator_domain.TaskStore) error {
				if err := store.CreateTask(ctx, h.newTask("rolled-back")); err != nil {
					return err
				}
				changed := h.newTask("existing")
				changed.Status = orchestrator_domain.StatusComplete
				if err := store.UpdateTask(ctx, changed); err != nil {
					return err
				}
				return errAbort
			})
			require.ErrorIs(t, err, errAbort)
			assert.Len(t, mustGet(t, h, "rolled-back"), 0, "a rolled-back create leaves nothing")
			assert.Equal(t, orchestrator_domain.StatusPending, h.get(t, "existing").Status, "a rolled-back update leaves the old state")
		}},
		{name: "atomic/reads-own-writes", run: checkAtomicReadsOwnWrites},
		{name: "atomic/nested-is-rejected", run: func(t *testing.T, h *Harness) {
			err := h.Store.RunAtomic(t.Context(), func(ctx context.Context, store orchestrator_domain.TaskStore) error {
				return store.RunAtomic(ctx, func(context.Context, orchestrator_domain.TaskStore) error { return nil })
			})
			assert.Error(t, err, "a nested transaction must be refused, not silently flattened")
		}},
	}
}

// checkAtomicReadsOwnWrites checks that a read inside RunAtomic sees the transaction's
// own uncommitted write.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
func checkAtomicReadsOwnWrites(t *testing.T, h *Harness) {
	err := h.Store.RunAtomic(t.Context(), func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		if err := store.CreateTask(ctx, h.newTask("visible")); err != nil {
			return err
		}
		rows, err := store.GetTasksByID(ctx, []string{"visible"})
		if err != nil {
			return err
		}
		if len(rows) != 1 {
			return fmt.Errorf("transaction read %d rows of its own write", len(rows))
		}
		return nil
	})
	assert.NoError(t, err)
}

// mustGet reads the tasks with the given IDs.
//
// Takes t (*testing.T) which reports test failures.
// Takes h (*Harness) which holds the store under test.
// Takes ids (...string) which identify the tasks.
//
// Returns []*orchestrator_domain.Task which holds the tasks found.
func mustGet(t *testing.T, h *Harness, ids ...string) []*orchestrator_domain.Task {
	t.Helper()
	rows, err := h.Store.GetTasksByID(t.Context(), ids)
	require.NoError(t, err)
	return rows
}
