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

package orchestrator_domain

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// sweepPageSize is the number of tasks read per page while sweeping.
	sweepPageSize = 100
)

// PendingPageStore is implemented by task stores that can page through due pending and
// retrying tasks in execution order.
type PendingPageStore interface {
	// PendingDispatchPage reads due pending and retrying tasks after a cursor.
	//
	// Takes afterID (string) which identifies the previous page's last task, or is empty to
	// start at the beginning.
	// Takes afterTime (time.Time) which is the execution time of the previous page's last
	// task.
	// Takes limit (int) which caps the page size.
	//
	// Returns []*Task which holds the tasks ordered by execution time and ID.
	// Returns error when the store cannot be read.
	PendingDispatchPage(ctx context.Context, afterID string, afterTime time.Time, limit int) ([]*Task, error)
}

// PublishCapacity is implemented by event buses that hold a bounded number of messages,
// so a dispatcher waiting for room can tell when to try again.
type PublishCapacity interface {
	// CapacitySignal returns a channel that receives a value whenever the bus releases
	// capacity. Signals coalesce, so one value may stand for several releases.
	//
	// Returns <-chan struct{} which signals released capacity, or nil when the bus never
	// signals.
	CapacitySignal() <-chan struct{}

	// FreeCapacity reports how many more messages the bus can accept now.
	//
	// Returns int which is the number of free message slots.
	FreeCapacity() int
}

// NodeLocality identifies whether a bus or provider keeps messages within this process.
//
// Bus and store locality must match so workers can access the tasks they receive.
type NodeLocality interface {
	// NodeLocal reports whether messages are delivered only within this process.
	//
	// Returns bool which is true for an in-process bus.
	NodeLocal() bool
}

// SweepRequest describes one pass of SweepPending.
type SweepRequest struct {
	// Through limits tasks to a fixed due-time cutoff; zero uses the current time.
	Through time.Time

	// LastActiveBefore limits tasks by their last write time; zero includes every due task.
	LastActiveBefore time.Time

	// DeadBefore exempts earlier task writes from LastActiveBefore.
	//
	// Only process-local transports may set it. Zero exempts no tasks.
	DeadBefore time.Time

	// Redispatch publishes a swept task again. The task is marked as already persisted, so
	// dispatching it does not insert it twice.
	Redispatch func(ctx context.Context, task *Task) error

	// Exclude, when set, skips tasks the caller knows are already being published.
	Exclude func(id string) bool

	// After continues a sweep paged across passes from where the previous pass stopped; the
	// zero cursor starts at the beginning.
	After SweepCursor

	// Budget caps how many tasks the pass re-dispatches.
	Budget int
}

// SweepCursor marks how far a sweep has read, in execution order.
type SweepCursor struct {
	// ExecuteAt is the execution time of the last task read.
	ExecuteAt time.Time

	// ID identifies the last task read.
	ID string
}

// SweepResult reports one pass of SweepPending.
type SweepResult struct {
	// Next continues the sweep after the last task this pass read.
	Next SweepCursor

	// Swept is the number of tasks re-dispatched.
	Swept int
}

// SweepPending republishes due pending and retrying tasks from the requested cursor.
//
// Duplicate deliveries are safe because claims are fenced by attempt.
//
// Takes request (SweepRequest) which describes the pass.
//
// Returns SweepResult which reports how many tasks were re-dispatched and where the pass
// stopped.
// Returns error when the store cannot be read; re-dispatch failures are counted and the
// pass continues.
func (c *TaskProcessingCore) SweepPending(ctx context.Context, request SweepRequest) (SweepResult, error) {
	result := SweepResult{Next: request.After, Swept: 0}
	if request.Budget <= 0 {
		return result, nil
	}
	page := c.sweepPager(request.Through)
	if page == nil {
		return result, nil
	}

	for result.Swept < request.Budget {
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("sweeping pending tasks: %w", err)
		}
		tasks, err := page(ctx, result.Next)
		if err != nil {
			return result, fmt.Errorf("reading pending tasks: %w", err)
		}
		if redispatchSweepPage(ctx, &request, tasks, &result) || len(tasks) < sweepPageSize {
			return result, nil
		}
	}
	return result, nil
}

// redispatchSweepPage re-dispatches the eligible tasks of one page until the pass's
// budget is spent, moving the result's cursor past each task it reads.
//
// Takes request (*SweepRequest) which describes the pass.
// Takes tasks ([]*Task) which is the page read.
// Takes result (*SweepResult) which accumulates the cursor and the re-dispatch count.
//
// Returns bool which is true when the budget ran out before the page was finished.
func redispatchSweepPage(ctx context.Context, request *SweepRequest, tasks []*Task, result *SweepResult) bool {
	for _, task := range tasks {
		if result.Swept >= request.Budget {
			return true
		}
		result.Next = SweepCursor{ExecuteAt: task.ExecuteAt, ID: task.ID}
		if !request.eligible(task) {
			continue
		}
		task.persisted = true
		if err := request.Redispatch(ctx, task); err != nil {
			SweepRedispatchErrorCount.Add(ctx, 1)
			continue
		}
		SweepRedispatchCount.Add(ctx, 1)
		result.Swept++
	}
	return false
}

// eligible reports whether a pass may re-dispatch a task it read.
//
// Takes task (*Task) which is the task read.
//
// Returns bool which is false for a task written too recently or excluded by the caller.
func (r SweepRequest) eligible(task *Task) bool {
	if !r.LastActiveBefore.IsZero() && !task.UpdatedAt.Before(r.LastActiveBefore) &&
		(r.DeadBefore.IsZero() || !task.UpdatedAt.Before(r.DeadBefore)) {
		return false
	}
	return r.Exclude == nil || !r.Exclude(task.ID)
}

// sweepPager returns how the store reads one page of due tasks after a cursor.
//
// Takes through (time.Time) which bounds the tasks to those due by then; zero means due
// by now.
//
// Returns func(context.Context, SweepCursor) ([]*Task, error) which reads a page, or nil
// when the store cannot page through due tasks.
func (c *TaskProcessingCore) sweepPager(through time.Time) func(context.Context, SweepCursor) ([]*Task, error) {
	if store, ok := c.TaskStore.(TaskStoreV2); ok && !through.IsZero() {
		return func(ctx context.Context, after SweepCursor) ([]*Task, error) {
			tasks, err := store.DispatchablePage(ctx, DispatchQuery{
				Now:             through,
				AfterTime:       after.ExecuteAt,
				PublishedBefore: time.Time{},
				AfterID:         after.ID,
				Limit:           sweepPageSize,
			})
			if errors.Is(err, ErrStoreUnsupported) {
				return c.pendingPage(ctx, after)
			}
			return tasks, err
		}
	}
	if _, ok := c.TaskStore.(PendingPageStore); !ok {
		return nil
	}
	return c.pendingPage
}

// pendingPage reads one page of tasks due now after a cursor.
//
// Takes after (SweepCursor) which is where the previous page stopped.
//
// Returns []*Task which holds the page.
// Returns error when the store cannot be read.
func (c *TaskProcessingCore) pendingPage(ctx context.Context, after SweepCursor) ([]*Task, error) {
	pager, ok := c.TaskStore.(PendingPageStore)
	if !ok {
		return nil, nil
	}
	return pager.PendingDispatchPage(ctx, after.ID, after.ExecuteAt, sweepPageSize)
}
