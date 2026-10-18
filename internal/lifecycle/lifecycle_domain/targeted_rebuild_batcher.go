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
	"slices"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/wdk/clock"
)

const (
	// targetedRebuildBatchWindow is the time allowed for further changes to join a batch.
	targetedRebuildBatchWindow = 100 * time.Millisecond
)

// targetedRebuildBatch collects the changes that one targeted rebuild covers.
type targetedRebuildBatch struct {
	// ctx is the context of the first change in the batch.
	ctx context.Context

	// changed lists the project-relative paths of changed components, in arrival order.
	changed []string

	// styleImporters lists components that import a changed stylesheet, in arrival order.
	styleImporters []string

	// labels names the changed files for logging and build causation.
	labels []string

	// reloadRoutes is true when any change in the batch created a page.
	reloadRoutes bool
}

// label returns the changed file names joined by commas for build logging.
//
// Returns string which is the label.
func (b *targetedRebuildBatch) label() string {
	return strings.Join(b.labels, ",")
}

// targetedRebuildBatcher groups file changes and runs targeted rebuilds sequentially.
//
// Changes received during a rebuild form the next batch. Outstanding-work tracking
// includes batches whose collection window is open.
type targetedRebuildBatcher struct {
	// clock schedules the end of each window.
	clock clock.Clock

	// run rebuilds one batch synchronously.
	run func(batch *targetedRebuildBatch)

	// acquire registers an open batch with the owner; false refuses the batch because the
	// owner is stopping.
	acquire func() bool

	// release ends the registration made by acquire.
	release func()

	// pending is the open batch, or nil.
	pending *targetedRebuildBatch

	// timer ends the open batch's window; nil when no window is running.
	timer clock.Timer

	// window is how long a batch stays open after its first change.
	window time.Duration

	// mu guards every field below it and pending and timer.
	mu sync.Mutex

	// running is true while a batch is being rebuilt.
	running bool

	// stopped is true once stop has been called; later changes are ignored.
	stopped bool
}

// newTargetedRebuildBatcher creates a batcher.
//
// Takes clk (clock.Clock) which schedules the end of each window.
// Takes window (time.Duration) which is how long a batch stays open.
// Takes acquire (func() bool) which registers an open batch with the owner and reports
// false when the owner is stopping.
// Takes release (func()) which ends a registration made by acquire.
// Takes run (func(*targetedRebuildBatch)) which rebuilds one batch synchronously.
//
// Returns *targetedRebuildBatcher which accepts changes until stopped.
func newTargetedRebuildBatcher(
	clk clock.Clock,
	window time.Duration,
	acquire func() bool,
	release func(),
	run func(batch *targetedRebuildBatch),
) *targetedRebuildBatcher {
	return &targetedRebuildBatcher{
		clock:   clk,
		run:     run,
		acquire: acquire,
		release: release,
		pending: nil,
		timer:   nil,
		window:  window,
		mu:      sync.Mutex{},
		running: false,
		stopped: false,
	}
}

// addChange adds a changed component to the open batch, opening one if needed.
//
// Takes label (string) which names the changed file.
// Takes changedPath (string) which is the project-relative path of the changed component.
// Takes reloadRoutes (bool) which is true when the change created a page.
func (b *targetedRebuildBatcher) addChange(ctx context.Context, label, changedPath string, reloadRoutes bool) {
	b.add(ctx, label, func(batch *targetedRebuildBatch) {
		batch.changed = appendUnique(batch.changed, changedPath)
		batch.reloadRoutes = batch.reloadRoutes || reloadRoutes
	})
}

// addStyleChange adds the importers of a changed stylesheet to the open batch, opening
// one if needed.
//
// Takes label (string) which names the changed stylesheet.
// Takes importers ([]string) which are the project-relative paths of components that
// import it.
func (b *targetedRebuildBatcher) addStyleChange(ctx context.Context, label string, importers []string) {
	b.add(ctx, label, func(batch *targetedRebuildBatch) {
		for _, importer := range importers {
			batch.styleImporters = appendUnique(batch.styleImporters, importer)
		}
	})
}

// add applies a change to the open batch, opening the batch and starting its window when
// none is open.
//
// Takes label (string) which names the changed file.
// Takes apply (func(*targetedRebuildBatch)) which records the change.
//
// Concurrent changes are serialised by mu.
func (b *targetedRebuildBatcher) add(ctx context.Context, label string, apply func(*targetedRebuildBatch)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped {
		return
	}

	if b.pending == nil {
		if !b.acquire() {
			return
		}
		b.pending = &targetedRebuildBatch{
			ctx:            ctx,
			changed:        nil,
			styleImporters: nil,
			labels:         nil,
			reloadRoutes:   false,
		}
		if !b.running {
			b.timer = b.clock.AfterFunc(b.window, b.flush)
		}
	}
	b.pending.labels = appendUnique(b.pending.labels, label)
	apply(b.pending)
}

// flush rebuilds the open batch when its window ends, then any batch that filled up while
// it ran.
//
// Concurrent access is coordinated through mu; rebuilds run after releasing the lock.
func (b *targetedRebuildBatcher) flush() {
	b.mu.Lock()
	b.timer = nil
	if b.running {
		b.mu.Unlock()
		return
	}

	for {
		batch := b.pending
		b.pending = nil
		if batch == nil {
			b.running = false
			b.mu.Unlock()
			return
		}
		if b.stopped {
			b.running = false
			b.mu.Unlock()
			b.release()
			return
		}
		b.running = true
		b.mu.Unlock()

		b.runAndRelease(batch)

		b.mu.Lock()
	}
}

// runAndRelease rebuilds one batch and releases its registration, even if the rebuild
// panics.
//
// Takes batch (*targetedRebuildBatch) which is the batch to rebuild.
func (b *targetedRebuildBatcher) runAndRelease(batch *targetedRebuildBatch) {
	defer b.release()
	b.run(batch)
}

// stop discards the open batch and ignores later changes. A batch already rebuilding
// finishes; one still in its window is dropped and released from the wait group.
//
// Concurrent additions and flushes are serialised by mu.
func (b *targetedRebuildBatcher) stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stopped = true

	if b.timer != nil && b.timer.Stop() {
		b.timer = nil
		if b.pending != nil {
			b.pending = nil
			b.release()
		}
	}
}

// appendUnique appends value to values unless it is already present.
//
// Takes values ([]string) which is the list to extend.
// Takes value (string) which is the value to add.
//
// Returns []string which is the list with value present once.
func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}
