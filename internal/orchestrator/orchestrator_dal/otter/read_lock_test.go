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
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestGetWorkflowStatus_LeavesIndexesUnchangedUnderReadLock(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	complete := makeTask("done", "wf")
	complete.Status = orchestrator_domain.StatusComplete
	require.NoError(t, d.CreateTask(ctx, complete))
	require.NoError(t, d.CreateTask(ctx, makeTask("lost", "wf")))
	require.NoError(t, d.tasks.Invalidate(ctx, "lost"))

	isComplete, err := d.GetWorkflowStatus(ctx, "wf")
	require.NoError(t, err)
	assert.True(t, isComplete, "a task missing from the cache does not hold the workflow open")

	assert.Contains(t, d.workflowIndex.Get("wf"), "lost", "a reader must not prune the workflow index")
	assert.True(t, d.danglingPending.Load(), "the missing task is noted for the next write")

	require.NoError(t, d.CreateTask(ctx, makeTask("next", "other")))

	assert.NotContains(t, d.workflowIndex.Get("wf"), "lost", "the next write prunes the dangling task")
	assert.Contains(t, d.workflowIndex.Get("wf"), "done")
	assert.False(t, d.danglingPending.Load())
}

func TestPruneDangling_SkippedInsideTransaction(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, d.CreateTask(ctx, makeTask("lost", "wf")))
	require.NoError(t, d.tasks.Invalidate(ctx, "lost"))
	_, err := d.GetWorkflowStatus(ctx, "wf")
	require.NoError(t, err)

	require.NoError(t, d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		return store.CreateTask(ctx, makeTask("in-tx", "other"))
	}))
	assert.Contains(t, d.workflowIndex.Get("wf"), "lost", "pruning waits for a write outside a transaction")

	require.NoError(t, d.CreateTask(ctx, makeTask("after-tx", "other")))
	assert.NotContains(t, d.workflowIndex.Get("wf"), "lost")
}

func TestPruneDangling_KeepsTaskThatReappeared(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, d.CreateTask(ctx, makeTask("flaky", "wf")))
	require.NoError(t, d.tasks.Invalidate(ctx, "flaky"))
	_, err := d.GetWorkflowStatus(ctx, "wf")
	require.NoError(t, err)

	require.NoError(t, d.CreateTask(ctx, makeTask("flaky", "wf")))
	assert.Contains(t, d.workflowIndex.Get("wf"), "flaky", "a task written again stays indexed")
}

func TestGetWorkflowStatus_ConcurrentReadersAndWriters(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	const workflows = 8
	for w := range workflows {
		for i := range 4 {
			require.NoError(t, d.CreateTask(ctx, makeTask(fmt.Sprintf("wf%d-t%d", w, i), fmt.Sprintf("wf%d", w))))
		}
		require.NoError(t, d.tasks.Invalidate(ctx, fmt.Sprintf("wf%d-t0", w)))
	}

	var wg sync.WaitGroup
	for r := range 8 {
		wg.Go(func() {
			for i := range 200 {
				_, err := d.GetWorkflowStatus(ctx, fmt.Sprintf("wf%d", (r+i)%workflows))
				assert.NoError(t, err)
			}
		})
	}
	for w := range 4 {
		wg.Go(func() {
			for i := range 100 {
				task := makeTask(fmt.Sprintf("writer%d-%d", w, i), fmt.Sprintf("wf%d", i%workflows))
				assert.NoError(t, d.CreateTask(ctx, task))
				task.Status = orchestrator_domain.StatusComplete
				assert.NoError(t, d.UpdateTask(ctx, task))
			}
		})
	}
	wg.Wait()

	require.NoError(t, d.CreateTask(ctx, makeTask("final", "final")))
	for w := range workflows {
		assert.NotContains(t, d.workflowIndex.Get(fmt.Sprintf("wf%d", w)), fmt.Sprintf("wf%d-t0", w))
	}
}
