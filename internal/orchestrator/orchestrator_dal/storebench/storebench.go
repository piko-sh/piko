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

package storebench

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// chunkSize is the number of tasks prepared outside the timer before the store is
	// reopened.
	chunkSize = 2048

	// populationSize is the number of tasks in the store for the read operations.
	populationSize = 10_000

	// pendingShare is the fraction of the population left pending for the read operations;
	// the rest are settled.
	pendingShare = 10

	// pageLimit is the page size for op=pending-page.
	pageLimit = 100

	// benchExecutor is the executor name every benchmark task carries.
	benchExecutor = "storebench.noop"
)

// stage is how far a chunk's tasks are taken before the measured operation runs.
type stage int

const (
	// stageNone leaves tasks unsaved.
	stageNone stage = iota

	// stageAdmitted saves tasks as pending.
	stageAdmitted

	// stageClaimed saves tasks and claims them for execution.
	stageClaimed
)

var (
	// errRetryable is the executor failure used for op=retry; it is not fatal, so the task
	// is retried.
	errRetryable = errors.New("storebench: retryable failure")

	// payload is the payload every benchmark task carries.
	payload = map[string]any{
		"artefactID": "assets/images/storebench-artefact.svg",
		"profile":    "minified",
	}
)

// Opener creates an empty task store and returns it with a function that closes it. A
// benchmark opens a store for each chunk of operations and closes it before the next.
type Opener func(tb testing.TB) (orchestrator_domain.TaskStore, func())

// Run benchmarks task admission, claims, settlement, retries, completion counts, and
// dispatch paging.
//
// Takes b (*testing.B) which runs and measures the benchmark.
// Takes open (Opener) which creates an empty store.
func Run(b *testing.B, open Opener) {
	b.Run("op=admit", func(b *testing.B) { runWriteOp(b, open, stageNone, admit) })
	b.Run("op=claim", func(b *testing.B) { runWriteOp(b, open, stageAdmitted, claim) })
	b.Run("op=settle", func(b *testing.B) { runWriteOp(b, open, stageClaimed, settle) })
	b.Run("op=retry", func(b *testing.B) { runWriteOp(b, open, stageClaimed, retry) })
	b.Run("op=count-unsettled", func(b *testing.B) { runCountUnsettled(b, open) })
	b.Run("op=pending-page", func(b *testing.B) { runPendingPage(b, open) })
}

// fixture is one chunk's store, processing core and prepared tasks.
type fixture struct {
	// core runs the orchestrator's claim and settle code against store.
	core *orchestrator_domain.TaskProcessingCore

	// store is the chunk's task store.
	store orchestrator_domain.TaskStore

	// close closes store.
	close func()

	// tasks are the chunk's prepared tasks.
	tasks []*orchestrator_domain.Task
}

// operation runs one measured operation on one prepared task.
type operation func(ctx context.Context, f *fixture, task *orchestrator_domain.Task) error

// runWriteOp measures op over tasks prepared to the given stage, preparing them in chunks
// outside the timer.
//
// Takes b (*testing.B) which runs and measures the benchmark.
// Takes open (Opener) which creates each chunk's store.
// Takes prepared (stage) which is how far tasks are taken before op runs.
// Takes op (operation) which is the measured operation.
func runWriteOp(b *testing.B, open Opener, prepared stage, op operation) {
	ctx := context.Background()
	var f *fixture
	next := chunkSize
	b.Cleanup(func() {
		if f != nil {
			f.close()
		}
	})

	b.ReportAllocs()
	for b.Loop() {
		if next == chunkSize {
			b.StopTimer()
			if f != nil {
				f.close()
			}
			f = newFixture(b, open, chunkSize, prepared)
			next = 0
			b.StartTimer()
		}
		if err := op(ctx, f, f.tasks[next]); err != nil {
			b.Fatalf("operation failed: %v", err)
		}
		next++
	}
}

// runCountUnsettled measures counting the outstanding work in a populated store.
//
// Takes b (*testing.B) which runs and measures the benchmark.
// Takes open (Opener) which creates the store.
func runCountUnsettled(b *testing.B, open Opener) {
	ctx := context.Background()
	f := populate(b, open)
	b.Cleanup(f.close)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := f.store.PendingTaskCount(ctx); err != nil {
			b.Fatalf("counting pending tasks: %v", err)
		}
	}
}

// runPendingPage measures reading a page of due tasks from a populated store. It is
// skipped for stores that cannot page pending tasks.
//
// Takes b (*testing.B) which runs and measures the benchmark.
// Takes open (Opener) which creates the store.
func runPendingPage(b *testing.B, open Opener) {
	ctx := context.Background()
	f := populate(b, open)
	b.Cleanup(f.close)
	pager, ok := f.store.(interface {
		PendingDispatchPage(ctx context.Context, afterID string, afterTime time.Time, limit int) ([]*orchestrator_domain.Task, error)
	})
	if !ok {
		b.Skipf("%T cannot page pending tasks", f.store)
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := pager.PendingDispatchPage(ctx, "", time.Time{}, pageLimit); err != nil {
			b.Fatalf("reading pending page: %v", err)
		}
	}
}

// newFixture opens a store and prepares count tasks to the given stage.
//
// Takes b (*testing.B) which runs and measures the benchmark.
// Takes open (Opener) which creates the store.
// Takes count (int) which is the number of tasks to prepare.
// Takes prepared (stage) which is how far the tasks are taken.
//
// Returns *fixture which holds the store, core and tasks.
func newFixture(b *testing.B, open Opener, count int, prepared stage) *fixture {
	ctx := context.Background()
	store, closeStore := open(b)
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.NodeID = "storebench"
	core := orchestrator_domain.NewTaskProcessingCore(config, nil, store, nil)
	core.RegisterExecutor(ctx, benchExecutor, nopExecutor{})

	f := &fixture{core: core, store: store, close: closeStore, tasks: make([]*orchestrator_domain.Task, count)}
	for i := range f.tasks {
		task := orchestrator_domain.NewTask(benchExecutor, payload)
		task.DeduplicationKey = "storebench:" + task.ID
		core.ApplyDefaults(task)
		if prepared >= stageAdmitted {
			if err := admit(ctx, f, task); err != nil {
				b.Fatalf("admitting task: %v", err)
			}
		}
		if prepared >= stageClaimed {
			if err := claim(ctx, f, task); err != nil {
				b.Fatalf("claiming task: %v", err)
			}
		}
		f.tasks[i] = task
	}
	return f
}

// populate opens a store holding populationSize tasks, one in pendingShare pending and
// the rest complete.
//
// Takes b (*testing.B) which runs and measures the benchmark.
// Takes open (Opener) which creates the store.
//
// Returns *fixture which holds the populated store.
func populate(b *testing.B, open Opener) *fixture {
	ctx := context.Background()
	f := newFixture(b, open, populationSize, stageNone)
	for i, task := range f.tasks {
		if i%pendingShare != 0 {
			task.Status = orchestrator_domain.StatusComplete
		}
		if err := f.store.CreateTask(ctx, task); err != nil {
			b.Fatalf("populating store: %v", err)
		}
	}
	return f
}

// admit saves a new task with its deduplication key.
//
// Takes f (*fixture) which holds the benchmark store and processing core.
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the store rejects the task.
func admit(ctx context.Context, f *fixture, task *orchestrator_domain.Task) error {
	return f.core.PersistWithDedup(ctx, task)
}

// claim claims a saved task for execution.
//
// Takes f (*fixture) which holds the benchmark store and processing core.
// Takes task (*orchestrator_domain.Task) which is the task to claim.
//
// Returns error when the claim fails.
func claim(ctx context.Context, f *fixture, task *orchestrator_domain.Task) error {
	return f.core.ClaimDelivery(ctx, task)
}

// settle settles a claimed task as complete.
//
// Takes f (*fixture) which holds the benchmark store and processing core.
// Takes task (*orchestrator_domain.Task) which is the task to complete.
//
// Returns error when the task was not settled.
func settle(ctx context.Context, f *fixture, task *orchestrator_domain.Task) error {
	f.core.HandleTaskSuccess(ctx, task, time.Time{})
	if task.Status != orchestrator_domain.StatusComplete {
		return fmt.Errorf("task %s settled as %s", task.ID, task.Status)
	}
	return nil
}

// retry settles a claimed task for a retry after a retryable failure.
//
// Takes f (*fixture) which holds the benchmark store and processing core.
// Takes task (*orchestrator_domain.Task) which is the task to retry.
//
// Returns error when the task was not scheduled for a retry.
func retry(ctx context.Context, f *fixture, task *orchestrator_domain.Task) error {
	f.core.HandleTaskFailure(ctx, task, errRetryable, time.Time{})
	if task.Status != orchestrator_domain.StatusRetrying {
		return fmt.Errorf("task %s settled as %s", task.ID, task.Status)
	}
	return nil
}

// nopExecutor is a task executor that does nothing.
type nopExecutor struct{}

// Execute succeeds without a result.
//
// Takes arg2 (map[string]any) which is ignored.
//
// Returns map[string]any which is always nil.
// Returns error which is always nil.
func (nopExecutor) Execute(context.Context, map[string]any) (map[string]any, error) {
	return nil, nil
}
