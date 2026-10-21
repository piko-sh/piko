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

	"piko.sh/piko/internal/logger/logger_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/goroutine"
)

var (
	// errZeroScheduledTime is returned when a task without a due time is scheduled.
	errZeroScheduledTime = errors.New("task has zero ScheduledExecuteAt time")
)

// TaskDispatchFunc publishes a reference to a task the store already holds once it is
// due. It allows testing without needing the full task dispatcher.
type TaskDispatchFunc func(ctx context.Context, ref TaskRef) error

// DelayedTaskPublisher publishes task references when due without retaining the mutable
// task objects.
type DelayedTaskPublisher struct {
	// ctx controls the publisher lifecycle; cancelled when Stop is called.
	ctx context.Context

	// clock provides time operations for scheduling and timer creation.
	clock clockpkg.Clock

	// dispatchFunc publishes a reference that is due.
	dispatchFunc TaskDispatchFunc

	// wakeChan signals the loop to wake up and check for newly scheduled references.
	wakeChan chan struct{}

	// cancel stops the processing loop; set by Start.
	cancel context.CancelCauseFunc

	// loopDone is closed when the processing loop has exited; set by Start.
	loopDone chan struct{}

	// refs holds the scheduled references ordered by due time; mu guards it.
	refs scheduledRefHeap

	// foreground counts the references in refs that are not background work; mu guards it.
	foreground int

	// inTransit counts due references being handed to the dispatcher and is guarded by mu.
	inTransit int

	// foregroundInTransit counts the in-transit references that are not background work; mu
	// guards it.
	foregroundInTransit int

	// mu protects refs and the counters during scheduling and retrieval.
	mu sync.Mutex
}

var (
	_ DelayedPublisher = (*DelayedTaskPublisher)(nil)
)

// NewDelayedTaskPublisherForTesting creates a delayed task publisher with injected
// dependencies. Use this for unit testing the delayed publisher in isolation.
//
// Takes clock (Clock) which provides controllable time functions for testing.
// Takes dispatchFunc (TaskDispatchFunc) which is called when a reference is due.
//
// Returns *DelayedTaskPublisher which is the configured publisher for testing.
func NewDelayedTaskPublisherForTesting(clock clockpkg.Clock, dispatchFunc TaskDispatchFunc) *DelayedTaskPublisher {
	return &DelayedTaskPublisher{
		ctx:                 nil,
		clock:               clock,
		dispatchFunc:        dispatchFunc,
		wakeChan:            make(chan struct{}, 1),
		cancel:              nil,
		loopDone:            nil,
		refs:                nil,
		foreground:          0,
		inTransit:           0,
		foregroundInTransit: 0,
		mu:                  sync.Mutex{},
	}
}

// Start begins the delayed task processing loop.
//
// Takes parentCtx (context.Context) which controls the publisher lifecycle.
//
// Spawns a goroutine that runs until Stop is called or parentCtx is cancelled. Call Start
// and Stop from the goroutine that owns the publisher.
func (p *DelayedTaskPublisher) Start(parentCtx context.Context) {
	p.ctx, p.cancel = context.WithCancelCause(parentCtx)
	loopDone := make(chan struct{})
	p.loopDone = loopDone

	_, l := logger_domain.From(parentCtx, log)
	l.Internal("Starting delayed task publisher")
	go func() {
		defer close(loopDone)
		p.loop()
	}()
}

// Stop shuts down the publisher cleanly, waiting for the processing loop to exit so no
// due reference is published after Stop returns.
func (p *DelayedTaskPublisher) Stop() {
	if p.cancel != nil {
		p.cancel(errors.New("delayed publisher stopped"))
	}
	if p.loopDone != nil {
		<-p.loopDone
	}
}

// PendingCount counts references queued or being handed to the dispatcher.
//
// Returns int which is the count of references currently waiting or in transit.
//
// Safe for concurrent use.
func (p *DelayedTaskPublisher) PendingCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refs.Len() + p.inTransit
}

// ForegroundPendingCount counts pending foreground references, including references being
// handed to the dispatcher.
//
// Returns int which is the count of foreground references waiting or in transit.
//
// Safe for concurrent use.
func (p *DelayedTaskPublisher) ForegroundPendingCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.foreground + p.foregroundInTransit
}

// Schedule records the task's reference for publication at its ScheduledExecuteAt. Only
// the reference and the due time are kept; the task is not read again.
//
// Takes task (*Task) which is the stored task to publish, with its due time set.
//
// Returns error when the task has a zero ScheduledExecuteAt time.
//
// Safe for concurrent use. Wakes the background loop to recalculate its next sleep.
func (p *DelayedTaskPublisher) Schedule(_ context.Context, task *Task) error {
	if task.ScheduledExecuteAt.IsZero() {
		return errZeroScheduledTime
	}

	p.mu.Lock()
	p.pushLocked(scheduledRef{due: task.ScheduledExecuteAt, ref: RefOf(task)})
	p.mu.Unlock()

	select {
	case p.wakeChan <- struct{}{}:
	default:
	}
	return nil
}

// loop is the main event loop that sleeps until references are due.
//
// Concurrent goroutine runs this loop, started by Start. Exits when the publisher's
// context is cancelled.
func (p *DelayedTaskPublisher) loop() {
	defer goroutine.RecoverPanic(p.ctx, "orchestrator.delayedPublisherLoop")
	ctx := p.ctx

	for {
		p.mu.Lock()
		next, ok := p.refs.Peek()
		p.mu.Unlock()

		if !ok {
			select {
			case <-p.wakeChan:
				continue
			case <-ctx.Done():
				return
			}
		}

		waitDuration := next.due.Sub(p.clock.Now())
		if waitDuration <= 0 {
			p.dispatchDue()
			continue
		}

		timer := p.clock.NewTimer(waitDuration)

		if next.due.Sub(p.clock.Now()) <= 0 {
			timer.Stop()
			p.dispatchDue()
			continue
		}

		select {
		case <-timer.C():
			p.dispatchDue()

		case <-p.wakeChan:
			timer.Stop()

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// dispatchDue publishes the next due reference.
//
// Temporary failures are retried after dispatchFailureRetryDelay. Permanent failures
// leave the stored task for the dispatcher's sweep.
//
// Safe for concurrent use. Uses a mutex to protect heap access.
func (p *DelayedTaskPublisher) dispatchDue() {
	p.mu.Lock()
	entry, ok := p.refs.Pop()
	if !ok {
		p.mu.Unlock()
		return
	}
	p.takeLocked(entry.ref)
	p.mu.Unlock()
	defer p.landed(entry.ref)

	err := p.dispatchFunc(p.ctx, entry.ref)
	if err == nil {
		DelayedTaskPublishedCount.Add(p.ctx, 1)
		return
	}

	_, l := logger_domain.From(p.ctx, log)
	if isPermanentDispatchError(err) {
		l.Warn("Dropping due task that can never be dispatched",
			logger_domain.Error(err),
			logger_domain.String(attributeKeyTaskID, entry.ref.ID))
		DelayedTaskDroppedCount.Add(p.ctx, 1)
		return
	}

	l.Warn("Failed to dispatch due task",
		logger_domain.Error(err),
		logger_domain.String(attributeKeyTaskID, entry.ref.ID))
	entry.due = p.clock.Now().Add(dispatchFailureRetryDelay)
	p.mu.Lock()
	p.pushLocked(entry)
	p.mu.Unlock()
}

// pushLocked adds an entry to the heap and counts it. The caller must hold mu.
//
// Takes entry (scheduledRef) which is the reference and its due time.
func (p *DelayedTaskPublisher) pushLocked(entry scheduledRef) {
	p.refs.Push(entry)
	if !isBackgroundRef(entry.ref) {
		p.foreground++
	}
}

// takeLocked moves a reference popped from the heap into transit. The caller must hold
// mu.
//
// Takes ref (TaskRef) which is the due reference being published.
func (p *DelayedTaskPublisher) takeLocked(ref TaskRef) {
	p.inTransit++
	if !isBackgroundRef(ref) {
		p.foreground--
		p.foregroundInTransit++
	}
}

// landed ends a reference's transit once the dispatcher has counted it, dropped it, or it
// has been scheduled again.
//
// Takes ref (TaskRef) which is the reference whose publication has finished.
//
// Safe for concurrent use; guarded by mu.
func (p *DelayedTaskPublisher) landed(ref TaskRef) {
	p.mu.Lock()
	p.inTransit--
	if !isBackgroundRef(ref) {
		p.foregroundInTransit--
	}
	p.mu.Unlock()
}

// isBackgroundRef reports whether a reference is to background work no build waits for.
//
// Takes ref (TaskRef) which is the reference.
//
// Returns bool which is true for background work.
func isBackgroundRef(ref TaskRef) bool {
	return ref.Flags&TaskRefBackground != 0
}

// NewDelayedTaskPublisher creates a delayed task publisher with the given clock and
// dispatch function.
//
// Takes dispatcherDispatch (TaskDispatchFunc) which publishes a reference once it is due.
// Takes clock (Clock) which provides time functions for scheduling.
//
// Returns DelayedPublisher which is ready to schedule tasks for later dispatch.
func NewDelayedTaskPublisher(dispatcherDispatch TaskDispatchFunc, clock clockpkg.Clock) DelayedPublisher {
	return NewDelayedTaskPublisherForTesting(clock, dispatcherDispatch)
}

// isPermanentDispatchError reports whether a dispatch failure will repeat however often
// it is retried, so the delayed publisher drops the task instead of rescheduling it
// forever and keeping the dispatcher from ever becoming idle.
//
// Takes err (error) which is the dispatch failure.
//
// Returns bool which is true when retrying cannot succeed.
func isPermanentDispatchError(err error) bool {
	for _, permanent := range []error{
		ErrDuplicateTask,
		ErrDispatcherStopped,
		ErrTaskNotRequired,
		errTaskIDRequired,
		errTaskWorkflowIDRequired,
		errTaskExecutorRequired,
	} {
		if errors.Is(err, permanent) {
			return true
		}
	}
	return false
}
