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

package lifecycle_domain

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

type batchRecorder struct {
	release    chan struct{}
	started    chan *targetedRebuildBatch
	batches    []*targetedRebuildBatch
	running    atomic.Int32
	maxRunning atomic.Int32
	mu         sync.Mutex
}

func newBatchRecorder() *batchRecorder {
	return &batchRecorder{started: make(chan *targetedRebuildBatch, 16)}
}

func (r *batchRecorder) run(batch *targetedRebuildBatch) {
	current := r.running.Add(1)
	for {
		previous := r.maxRunning.Load()
		if current <= previous || r.maxRunning.CompareAndSwap(previous, current) {
			break
		}
	}
	r.started <- batch
	if r.release != nil {
		<-r.release
	}
	r.mu.Lock()
	r.batches = append(r.batches, batch)
	r.mu.Unlock()
	r.running.Add(-1)
}

func (r *batchRecorder) recorded() []*targetedRebuildBatch {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*targetedRebuildBatch(nil), r.batches...)
}

func newTestBatcher(mockClock clock.Clock, wg *sync.WaitGroup, run func(*targetedRebuildBatch)) *targetedRebuildBatcher {
	return newTargetedRebuildBatcher(mockClock, targetedRebuildBatchWindow,
		func() bool { wg.Add(1); return true }, wg.Done, run)
}

func TestTargetedRebuildBatcher_CoalescesChangesWithinTheWindow(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	recorder := newBatchRecorder()
	var wg sync.WaitGroup
	batcher := newTestBatcher(mockClock, &wg, recorder.run)
	ctx := context.Background()

	batcher.addChange(ctx, "pages/a.pk", "pages/a.pk", false)
	mockClock.Advance(targetedRebuildBatchWindow / 2)
	batcher.addChange(ctx, "pages/b.pk", "pages/b.pk", true)
	batcher.addChange(ctx, "pages/a.pk", "pages/a.pk", false)
	batcher.addStyleChange(ctx, "styles/site.css", []string{"pages/a.pk", "partials/card.pk"})
	require.Empty(t, recorder.recorded(), "nothing runs before the window ends")

	mockClock.Advance(targetedRebuildBatchWindow / 2)
	wg.Wait()

	batches := recorder.recorded()
	require.Len(t, batches, 1, "changes inside one window share one targeted rebuild")
	assert.Equal(t, []string{"pages/a.pk", "pages/b.pk"}, batches[0].changed)
	assert.Equal(t, []string{"pages/a.pk", "partials/card.pk"}, batches[0].styleImporters)
	assert.Equal(t, "pages/a.pk,pages/b.pk,styles/site.css", batches[0].label())
	assert.True(t, batches[0].reloadRoutes, "a page created anywhere in the batch reloads routes")
}

func TestTargetedRebuildBatcher_RunsBatchesOneAtATime(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	recorder := newBatchRecorder()
	recorder.release = make(chan struct{})
	var wg sync.WaitGroup
	batcher := newTestBatcher(mockClock, &wg, recorder.run)
	ctx := context.Background()

	batcher.addChange(ctx, "pages/a.pk", "pages/a.pk", false)
	advanced := make(chan struct{})
	go func() {
		mockClock.Advance(targetedRebuildBatchWindow)
		close(advanced)
	}()
	first := <-recorder.started

	batcher.addChange(ctx, "pages/b.pk", "pages/b.pk", false)
	batcher.addChange(ctx, "pages/c.pk", "pages/c.pk", false)
	recorder.release <- struct{}{}

	second := <-recorder.started
	recorder.release <- struct{}{}
	<-advanced
	wg.Wait()

	assert.Equal(t, []string{"pages/a.pk"}, first.changed)
	assert.Equal(t, []string{"pages/b.pk", "pages/c.pk"}, second.changed,
		"changes that arrive while a batch rebuilds form the next batch")
	assert.Equal(t, int32(1), recorder.maxRunning.Load(), "targeted rebuilds never overlap")
}

func TestTargetedRebuildBatcher_StopDropsTheOpenBatch(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	recorder := newBatchRecorder()
	var wg sync.WaitGroup
	batcher := newTestBatcher(mockClock, &wg, recorder.run)
	ctx := context.Background()

	batcher.addChange(ctx, "pages/a.pk", "pages/a.pk", false)
	batcher.stop()
	wg.Wait()

	batcher.addChange(ctx, "pages/b.pk", "pages/b.pk", false)
	mockClock.Advance(10 * targetedRebuildBatchWindow)
	wg.Wait()

	assert.Empty(t, recorder.recorded(), "an open batch is dropped on stop and later changes are ignored")
}
