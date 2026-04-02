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
	"sync"
	"time"

	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

// TaskDispatchFunc is called when a delayed task is ready to run.
// It allows testing without needing the full task dispatcher.
type TaskDispatchFunc func(ctx context.Context, task *Task) error

// DelayedTaskPublisher handles scheduled task execution using a min-heap.
// Tasks are sorted by their scheduled execution time.
type DelayedTaskPublisher struct {
	// ctx controls the publisher lifecycle; cancelled when Stop is called.
	ctx context.Context

	// clock provides time operations for scheduling and timer creation.
	clock clockpkg.Clock

	// dispatchFunc sends a task that is due to its target queue.
	dispatchFunc TaskDispatchFunc

	// taskHeap stores tasks ordered by their scheduled execution time.
	taskHeap *scheduledTaskHeap

	// wakeChan signals the loop to wake up and check for newly scheduled tasks.
	wakeChan chan struct{}

	// cancel stops the processing loop; set by Start.
	cancel context.CancelCauseFunc

	// mu protects taskHeap during scheduling and retrieval.
	mu sync.Mutex
}

var _ DelayedPublisher = (*DelayedTaskPublisher)(nil)

// NewDelayedTaskPublisherForTesting creates a delayed task publisher with
// injected dependencies. Use this for unit testing the delayed publisher in
// isolation.
//
// Takes clock (Clock) which provides controllable time functions for
// testing.
// Takes dispatchFunc (TaskDispatchFunc) which is called when a task is
// due to run.
//
// Returns *DelayedTaskPublisher which is the configured publisher for
// testing.
func NewDelayedTaskPublisherForTesting(clock clockpkg.Clock, dispatchFunc TaskDispatchFunc) *DelayedTaskPublisher {
	return &DelayedTaskPublisher{
		ctx:          nil,
		clock:        clock,
		dispatchFunc: dispatchFunc,
		taskHeap:     newScheduledTaskHeap(),
		wakeChan:     make(chan struct{}, 1),
		cancel:       nil,
		mu:           sync.Mutex{},
	}
}

// Start begins the delayed task processing loop.
//
// Takes parentCtx (context.Context) which controls the publisher lifecycle.
//
// Safe for concurrent use. The spawned goroutine runs until Stop is called.
func (p *DelayedTaskPublisher) Start(parentCtx context.Context) {
	p.ctx, p.cancel = context.WithCancelCause(parentCtx)

	_, l := logger_domain.From(parentCtx, log)
	l.Internal("Starting delayed task publisher")
	go p.loop()
}

// Stop shuts down the publisher cleanly.
func (p *DelayedTaskPublisher) Stop() {
	if p.cancel != nil {
		p.cancel(errors.New("delayed publisher stopped"))
	}
}

// PendingCount returns the number of tasks waiting to be dispatched.
// Used for idle detection - the dispatcher is not idle while tasks are pending.
//
// Returns int which is the count of tasks currently in the pending queue.
//
// Safe for concurrent use.
func (p *DelayedTaskPublisher) PendingCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.taskHeap.Len()
}

// Schedule adds a task to be dispatched at the specified time.
//
// Takes task (*Task) which specifies the task to schedule with its execution
// time.
//
// Returns error when the task has a zero ScheduledExecuteAt time.
//
// Safe for concurrent use. Wakes the background dispatch loop to recalculate
// the next sleep duration.
func (p *DelayedTaskPublisher) Schedule(ctx context.Context, task *Task) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "DelayedTaskPublisher.Schedule",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.Time("executeAt", task.ScheduledExecuteAt),
	)
	defer span.End()

	if task.ScheduledExecuteAt.IsZero() {
		err := errors.New("task has zero ScheduledExecuteAt time")
		l.ReportError(span, err, "Cannot schedule task")
		return err
	}

	p.mu.Lock()
	p.taskHeap.Push(task)
	p.mu.Unlock()

	select {
	case p.wakeChan <- struct{}{}:
	default:
	}

	l.Trace("Task scheduled",
		logger_domain.Duration("delay", task.ScheduledExecuteAt.Sub(p.clock.Now())))
	span.SetStatus(codes.Ok, "Task scheduled")
	return nil
}

// loop is the main event loop that sleeps until tasks are due.
//
// Concurrent goroutine runs this loop, started by Start. Exits when the
// publisher's context is cancelled.
func (p *DelayedTaskPublisher) loop() {
	defer goroutine.RecoverPanic(p.ctx, "orchestrator.delayedPublisherLoop")
	ctx, ll := logger_domain.From(p.ctx, log)
	l := ll.With(logger_domain.String("component", "DelayedTaskPublisher"))
	l.Internal("Delayed task publisher loop started")

	for {
		p.mu.Lock()

		if p.taskHeap.Len() == 0 {
			p.mu.Unlock()

			select {
			case <-p.wakeChan:
				continue
			case <-ctx.Done():
				l.Trace("Delayed publisher shutting down")
				return
			}
		}

		nextTask := p.taskHeap.Peek()
		scheduledAt := nextTask.ScheduledExecuteAt
		p.mu.Unlock()

		waitDuration := scheduledAt.Sub(p.clock.Now())
		if waitDuration <= 0 {
			p.dispatchDueTask()
			continue
		}

		timer := p.clock.NewTimer(waitDuration)

		if scheduledAt.Sub(p.clock.Now()) <= 0 {
			timer.Stop()
			p.dispatchDueTask()
			continue
		}

		select {
		case <-timer.C():
			p.dispatchDueTask()

		case <-p.wakeChan:
			timer.Stop()

		case <-ctx.Done():
			timer.Stop()
			l.Trace("Delayed publisher shutting down during sleep")
			return
		}
	}
}

// dispatchDueTask pops and sends the next task that is due.
//
// Safe for concurrent use. Uses a mutex to protect heap access.
func (p *DelayedTaskPublisher) dispatchDueTask() {
	ctx, dl := logger_domain.From(p.ctx, log)
	ctx, span, l := dl.Span(ctx, "DelayedTaskPublisher.dispatchDueTask")
	defer span.End()

	p.mu.Lock()
	if p.taskHeap.Len() == 0 {
		p.mu.Unlock()
		return
	}

	task := p.taskHeap.Pop()
	p.mu.Unlock()

	if task == nil {
		return
	}

	l.Trace("Dispatching due task",
		logger_domain.String(attributeKeyTaskID, task.ID),
		logger_domain.Time("scheduledFor", task.ScheduledExecuteAt))

	task.ScheduledExecuteAt = time.Time{}
	task.Status = StatusPending

	if err := p.dispatchFunc(ctx, task); err != nil {
		l.Warn("Failed to dispatch due task",
			logger_domain.Error(err),
			logger_domain.String(attributeKeyTaskID, task.ID))

		task.ScheduledExecuteAt = p.clock.Now().Add(dispatchFailureRetryDelay)
		p.mu.Lock()
		p.taskHeap.Push(task)
		p.mu.Unlock()

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to dispatch due task")
		return
	}

	DelayedTaskPublishedCount.Add(ctx, 1)
	l.Trace("Delayed task dispatched successfully")
	span.SetStatus(codes.Ok, "Delayed task dispatched")
}

// NewDelayedTaskPublisher creates a delayed task publisher with the given
// clock and dispatch function.
//
// Takes dispatcherDispatch (TaskDispatchFunc) which is called when a task is
// due to run.
// Takes clock (Clock) which provides time functions for scheduling.
//
// Returns DelayedPublisher which is ready to schedule tasks for later dispatch.
func NewDelayedTaskPublisher(dispatcherDispatch TaskDispatchFunc, clock clockpkg.Clock) DelayedPublisher {
	return &DelayedTaskPublisher{
		ctx:          nil,
		clock:        clock,
		dispatchFunc: dispatcherDispatch,
		taskHeap:     newScheduledTaskHeap(),
		wakeChan:     make(chan struct{}, 1),
		cancel:       nil,
		mu:           sync.Mutex{},
	}
}
