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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

func TestWithClock_DueQueriesUseInjectedClock(t *testing.T) {
	t.Parallel()

	now := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	store, err := NewOtterDAL(Config{Capacity: 100}, WithClock(clockpkg.NewMockClock(now)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	dal, ok := store.(*DAL)
	require.True(t, ok)

	for _, task := range []*orchestrator_domain.Task{
		{ID: "stale", WorkflowID: "wf", Executor: "test", Status: orchestrator_domain.StatusProcessing,
			Attempt: 1, ExecuteAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
		{ID: "fresh", WorkflowID: "wf", Executor: "test", Status: orchestrator_domain.StatusProcessing,
			Attempt: 1, ExecuteAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: "due", WorkflowID: "wf", Executor: "test", Status: orchestrator_domain.StatusPending,
			ExecuteAt: now.Add(-time.Second), CreatedAt: now, UpdatedAt: now},
		{ID: "later", WorkflowID: "wf", Executor: "test", Status: orchestrator_domain.StatusPending,
			ExecuteAt: now.Add(time.Minute), CreatedAt: now, UpdatedAt: now},
	} {
		require.NoError(t, store.CreateTask(t.Context(), task))
	}

	orphans, err := dal.ListOrphanedTasks(t.Context(), orchestrator_domain.OrphanQuery{
		Now: now, StaleBefore: now.Add(-time.Minute), Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, orphans, 1, "only the task idle past the cutoff is an orphan")
	assert.Equal(t, "stale", orphans[0].ID)

	recovered := orphans[0]
	recovered.Status = orchestrator_domain.StatusRetrying
	recovered.ExecuteAt, recovered.UpdatedAt = now, now
	_, err = dal.TransitionTask(t.Context(), orchestrator_domain.Transition{
		Task: recovered, From: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusProcessing},
	})
	require.NoError(t, err)

	page, err := dal.PendingDispatchPage(t.Context(), "", time.Time{}, 10)
	require.NoError(t, err)
	ids := make([]string, 0, len(page))
	for _, task := range page {
		ids = append(ids, task.ID)
	}
	assert.Contains(t, ids, "due")
	assert.Contains(t, ids, "stale", "the recovered task is due now by the injected clock")
	assert.NotContains(t, ids, "later", "tasks due after the injected clock's time are not dispatched")
}
