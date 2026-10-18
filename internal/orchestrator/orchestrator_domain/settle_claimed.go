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
	"time"

	"go.opentelemetry.io/otel/codes"

	"piko.sh/piko/internal/logger/logger_domain"
)

// settleKind is what a settled attempt counts as once its state is written.
type settleKind uint8

const (
	// settleComplete counts the attempt as completed.
	settleComplete settleKind = iota

	// settleFailed counts the attempt as failed for good.
	settleFailed

	// settleRetry counts the attempt as failed with another attempt scheduled.
	settleRetry
)

// pendingSettle is an attempt whose settled state still has to be written and accounted
// for.
type pendingSettle struct {
	// startTime is when the attempt began executing, for its completion event.
	startTime time.Time

	// task holds the settled state.
	task *Task

	// next is a recurring task's successor, created with the settle, or nil.
	next *Task

	// cause is the execution error of a failed attempt, or nil.
	cause error

	// kind is what the attempt counts as.
	kind settleKind
}

// finishAttempt persists an attempt's settled state and updates its counters, metrics,
// events, and follow-on work.
//
// Failed writes remain unsettled until RetryPendingSettles succeeds. A lost claim
// releases its dispatch count because another attempt owns the task.
//
// Takes settle (pendingSettle) which describes the attempt.
//
// Returns bool which is true when the state was written and accounted for.
func (c *TaskProcessingCore) finishAttempt(ctx context.Context, settle pendingSettle) bool {
	outcome, err := c.writeSettlement(ctx, settle.task, settle.next)
	if err != nil && settle.next != nil && !errors.Is(err, ErrObsoleteDelivery) {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Settling without the recurring successor, which its re-seed recreates",
			logger_domain.String(attributeKeyTaskID, settle.task.ID),
			logger_domain.Error(err))
		settle.next = nil
		outcome, err = c.writeSettlement(ctx, settle.task, nil)
	}
	switch {
	case err == nil:
	case errors.Is(err, ErrObsoleteDelivery):
		c.TasksDispatched.Add(-1)
		StaleWriteCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Trace("Attempt was superseded before it settled",
			logger_domain.String(attributeKeyTaskID, settle.task.ID),
			logger_domain.Int(attributeKeyAttempt, settle.task.Attempt))
		return false
	default:
		c.keepPendingSettle(new(settle))
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to settle task; retrying on the next sweep",
			logger_domain.String(attributeKeyTaskID, settle.task.ID),
			logger_domain.Error(err))
		return false
	}

	switch settle.kind {
	case settleComplete:
		c.accountCompleted(ctx, &settle, outcome)
	case settleFailed:
		c.accountFailed(ctx, &settle, outcome)
	case settleRetry:
		c.accountRetry(ctx, &settle)
	}
	c.SignalSettled()
	return true
}

// accountCompleted counts a finished attempt and starts its successor, rerun, completion
// event, and callbacks.
//
// Takes settle (*pendingSettle) which describes the attempt.
// Takes outcome (SettleOutcome) which reports the tasks the settle created.
func (c *TaskProcessingCore) accountCompleted(ctx context.Context, settle *pendingSettle, outcome SettleOutcome) {
	c.TasksCompleted.Add(1)
	TaskSuccessCount.Add(ctx, 1)
	if successor := outcome.Successor; successor != nil && c.DelayedPublisher != nil {
		successor.persisted = true
		if err := c.DelayedPublisher.Schedule(context.WithoutCancel(ctx), successor); err != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Successor remains scheduled for recovery", logger_domain.Error(err))
		}
	}
	c.publishRerun(ctx, outcome.Rerun)
	c.PublishCompletionEvent(ctx, settle.task, nil, c.Clock.Now().Sub(settle.startTime))
	c.notifySettled(ctx, settle.task)
}

// accountFailed counts an attempt that failed for good and reports it.
//
// Takes settle (*pendingSettle) which describes the attempt.
// Takes outcome (SettleOutcome) which reports any rerun the settle created.
func (c *TaskProcessingCore) accountFailed(ctx context.Context, settle *pendingSettle, outcome SettleOutcome) {
	c.TasksFailed.Add(1)
	if settle.task.IsFatal {
		c.TasksFatalFailed.Add(1)
	}
	TaskFailureCount.Add(ctx, 1)
	c.publishRerun(ctx, outcome.Rerun)
	c.PublishCompletionEvent(ctx, settle.task, settle.cause, c.Clock.Now().Sub(settle.startTime))
	c.notifySettled(ctx, settle.task)
}

// accountRetry counts an attempt that failed with another attempt scheduled, and hands
// the task to the delayed publisher for its due time.
//
// Takes settle (*pendingSettle) which describes the attempt.
func (c *TaskProcessingCore) accountRetry(ctx context.Context, settle *pendingSettle) {
	c.TasksRetried.Add(1)
	TaskRetryCount.Add(ctx, 1)
	task := settle.task
	task.claimed = false
	if c.DelayedPublisher == nil {
		return
	}
	if err := c.DelayedPublisher.Schedule(context.WithoutCancel(ctx), task); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to schedule retry, task may be lost", logger_domain.Error(err))
	}
}

// publishRerun hands a rerun created by a settle to OnRerunCreated. Without a callback
// the rerun stays pending in the store for the next sweep to publish.
//
// Takes rerun (*Task) which is the rerun, or nil.
func (c *TaskProcessingCore) publishRerun(ctx context.Context, rerun *Task) {
	if rerun == nil || c.OnRerunCreated == nil {
		return
	}
	rerun.persisted = true
	c.OnRerunCreated(ctx, rerun)
}

// writeSettlement writes an attempt's settled state, bounded by taskPersistTimeout.
//
// A claimed attempt settles with one fenced write that also creates its successor and any
// requested rerun. An attempt that was never claimed, or a store without the
// compare-and-set API, writes the task directly.
//
// Takes task (*Task) which holds the settled state.
// Takes next (*Task) which is a successor to create, or nil.
//
// Returns SettleOutcome which reports the tasks created.
// Returns error which is ErrObsoleteDelivery when the claim was lost, or a storage error.
func (c *TaskProcessingCore) writeSettlement(ctx context.Context, task, next *Task) (SettleOutcome, error) {
	if c.TaskStore == nil {
		return SettleOutcome{Successor: next, Rerun: nil}, nil
	}
	ctx, cancel := c.storeContext(ctx, true, errTaskPersistTimeout)
	defer cancel()

	if task.claimed {
		outcome, err := c.TaskStore.SettleTask(ctx, Settlement{Task: task, Successor: next})
		if !errors.Is(err, ErrStoreUnsupported) {
			return outcome, err
		}
	}
	return c.writeSettlementInTransaction(ctx, task, next)
}

// writeSettlementInTransaction writes a settled state on a store without the
// compare-and-set API, fenced by the claimed attempt inside a transaction.
//
// Takes task (*Task) which holds the settled state.
// Takes next (*Task) which is a successor to create, or nil; one whose deduplication key
// is held is skipped.
//
// Returns SettleOutcome which reports the successor created.
// Returns error which is ErrObsoleteDelivery when the claim was lost, or a storage error.
func (c *TaskProcessingCore) writeSettlementInTransaction(ctx context.Context, task, next *Task) (SettleOutcome, error) {
	if next == nil && !task.claimed {
		return SettleOutcome{}, c.TaskStore.UpdateTask(ctx, task)
	}
	var outcome SettleOutcome
	err := c.TaskStore.RunAtomic(ctx, func(ctx context.Context, store TaskStore) error {
		outcome = SettleOutcome{}
		if err := checkClaimHeld(ctx, store, task); err != nil {
			return err
		}
		if err := store.UpdateTask(ctx, task); err != nil {
			return err
		}
		if next == nil {
			return nil
		}
		err := store.CreateTaskWithDedup(ctx, next)
		switch {
		case err == nil:
			outcome.Successor = next
		case !errors.Is(err, ErrDuplicateTask):
			return err
		}
		return nil
	})
	return outcome, err
}

// checkClaimHeld confirms, inside a settle's transaction, that the stored task is still
// processing the attempt a claimed task holds. A task that was never claimed passes.
//
// Takes store (TaskStore) which is the transaction's store.
// Takes task (*Task) which holds the settled state of the attempt.
//
// Returns error which is ErrObsoleteDelivery when the claim was lost, or a storage error.
func checkClaimHeld(ctx context.Context, store TaskStore, task *Task) error {
	if !task.claimed {
		return nil
	}
	rows, err := store.GetTasksByID(ctx, []string{task.ID})
	if err != nil {
		return err
	}
	if len(rows) != 1 || rows[0].Attempt != task.Attempt || rows[0].Status != StatusProcessing {
		return ErrObsoleteDelivery
	}
	return nil
}

// localStore is implemented by a task store whose writes never wait on a network or
// another process, so they need no timeout of their own.
type localStore interface {
	// LocalStore reports whether the store's writes complete in this process.
	//
	// Returns bool which is true when writes complete within this process.
	LocalStore() bool
}

// storeContext bounds a store write by taskPersistTimeout, detached from ctx's
// cancellation when detach is set. A local store's writes are neither bounded nor
// detached, which saves a timer and two contexts per write.
//
// Takes detach (bool) which is true when the write must outlive ctx.
// Takes cause (error) which is the timeout's cause.
//
// Returns context.Context which bounds the write.
// Returns context.CancelFunc which releases it.
func (c *TaskProcessingCore) storeContext(ctx context.Context, detach bool, cause error) (context.Context, context.CancelFunc) {
	if local, ok := c.TaskStore.(localStore); ok && local.LocalStore() {
		return ctx, func() {}
	}
	if detach {
		ctx = context.WithoutCancel(ctx)
	}
	return context.WithTimeoutCause(ctx, taskPersistTimeout, cause)
}

// keepPendingSettle keeps an attempt whose settled state could not be written.
//
// Takes settle (*pendingSettle) which describes the attempt.
//
// Safe for concurrent use; guarded by pendingSettlesMu.
func (c *TaskProcessingCore) keepPendingSettle(settle *pendingSettle) {
	c.pendingSettlesMu.Lock()
	c.pendingSettles = append(c.pendingSettles, settle)
	c.pendingSettlesMu.Unlock()
}

// PendingSettleCount counts finished attempts with unwritten settlement results.
//
// Returns int which is the count.
//
// Safe for concurrent use; guarded by pendingSettlesMu.
func (c *TaskProcessingCore) PendingSettleCount() int {
	c.pendingSettlesMu.Lock()
	defer c.pendingSettlesMu.Unlock()
	return len(c.pendingSettles)
}

// RetryPendingSettles writes the settled state of attempts whose earlier write failed,
// and accounts for each one written.
//
// Returns int which is the number of attempts written.
//
// Safe for concurrent use; an attempt that fails again is kept for the next call.
func (c *TaskProcessingCore) RetryPendingSettles(ctx context.Context) int {
	c.pendingSettlesMu.Lock()
	pending := c.pendingSettles
	c.pendingSettles = nil
	c.pendingSettlesMu.Unlock()

	written := 0
	for _, settle := range pending {
		if ctx.Err() != nil {
			c.keepPendingSettle(settle)
			continue
		}
		if c.finishAttempt(ctx, *settle) {
			written++
		}
	}
	return written
}

// beginSettle marks an attempt as settling, so it is settled at most once.
//
// Takes task (*Task) which is the attempt.
//
// Returns bool which is false when the attempt was already settled.
func beginSettle(task *Task) bool {
	if task.settled {
		return false
	}
	task.settled = true
	return true
}

// spanStatus is the part of a tracing span a settle reports to.
type spanStatus interface {
	// SetStatus records the span's outcome.
	//
	// Takes code (codes.Code) which is the span status.
	// Takes description (string) which explains the span status.
	SetStatus(code codes.Code, description string)
}
