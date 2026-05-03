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

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

// batchInsertLoop gathers tasks from a channel and inserts them into the
// database in batches for better performance.
func (s *orchestratorService) batchInsertLoop() {
	defer s.wg.Done()
	defer goroutine.RecoverPanic(s.runCtx, "orchestrator.batchInsertLoop")

	batch := make([]*Task, 0, s.batchSize)
	timer := s.clock.NewTimer(s.batchTimeout)
	if !timer.Stop() {
		<-timer.C()
	}

	ctx, bl := logger_domain.From(s.runCtx, log)
	bl.Internal("Batch inserter started",
		logger_domain.Int("batchSize", s.batchSize),
		logger_domain.Duration("batchTimeout", s.batchTimeout))

	for {
		select {
		case <-ctx.Done():
			s.handleBatchShutdown(ctx, batch)
			return

		case task := <-s.taskInsertChan:
			batch = append(batch, task)
			timer.Reset(s.batchTimeout)
			batch = s.drainAndFlushBatch(batch, timer)
			stopChannelTimerSafe(timer)

		case <-timer.C():
		}
	}
}

// handleBatchShutdown drains remaining tasks from the channel and flushes them.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes batch ([]*Task) which contains any pending tasks to include in flush.
//
// The write-lock acquisition waits for any in-flight Dispatch/Schedule callers
// holding RLock to leave their critical section. Once the lock is held, the
// closed flag is set so future senders short-circuit, then the channel is
// closed and drained. This makes close-on-channel safe under concurrent sends.
//
// Concurrency: acquires taskInsertMutex.Lock to wait for all in-flight senders
// before closing taskInsertChan; senders observe taskInsertClosed via the read
// lock.
func (s *orchestratorService) handleBatchShutdown(ctx context.Context, batch []*Task) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Shutting down batch inserter, flushing remaining tasks...")

	s.taskInsertMutex.Lock()
	s.taskInsertClosed = true
	close(s.taskInsertChan)
	s.taskInsertMutex.Unlock()

	for task := range s.taskInsertChan {
		batch = append(batch, task)
	}
	s.flushTaskBatch(batch)
}

// drainAndFlushBatch drains available tasks up to batch size, then flushes.
//
// Takes batch ([]*Task) which is the current batch to append drained tasks to.
// Takes timer (clock.ChannelTimer) which controls the drain timeout.
//
// Returns []*Task which is the batch after draining and flushing.
func (s *orchestratorService) drainAndFlushBatch(batch []*Task, timer clock.ChannelTimer) []*Task {
	batch = s.drainTaskChannel(batch, timer)
	return s.flushTaskBatch(batch)
}

// drainTaskChannel collects tasks from the channel until the batch is full or
// time runs out.
//
// Takes batch ([]*Task) which is the current batch to add tasks to.
// Takes timer (clock.ChannelTimer) which signals when to stop waiting.
//
// Returns []*Task which is the batch with any newly collected tasks.
func (s *orchestratorService) drainTaskChannel(batch []*Task, timer clock.ChannelTimer) []*Task {
	for len(batch) < s.batchSize {
		select {
		case task, ok := <-s.taskInsertChan:
			if !ok {
				return batch
			}
			batch = append(batch, task)
		case <-timer.C():
			return batch
		default:
			return batch
		}
	}
	return batch
}

// flushTaskBatch saves a batch of tasks to the database in one operation.
// After saving, it sends the tasks to the dispatcher for processing.
//
// Takes batch ([]*Task) which contains the tasks to save and dispatch.
//
// Returns []*Task which is the batch slice reset to zero length for reuse.
func (s *orchestratorService) flushTaskBatch(batch []*Task) []*Task {
	if len(batch) == 0 {
		return batch
	}

	ctx := context.WithoutCancel(s.runCtx)
	ctx, fl := logger_domain.From(ctx, log)

	err := s.taskStore.CreateTasks(ctx, batch)
	if err != nil {
		fl.Error("Failed to insert task batch", logger_domain.Error(err), logger_domain.Int("batchSize", len(batch)))
		s.failBatchReceipts(batch, err)
		return batch[:0]
	}

	for _, task := range batch {
		task.persisted = true
	}

	fl.Trace("Successfully inserted task batch", logger_domain.Int("batchSize", len(batch)))
	s.dispatchPersistedTasks(ctx, batch)
	return batch[:0]
}

// failBatchReceipts marks all receipts for the given batch of tasks as failed
// with the provided error.
//
// Takes batch ([]*Task) which contains the tasks that failed to process.
// Takes err (error) which is the error to resolve all receipts with.
//
// Safe for concurrent use. Locks the receipts mutex while updating.
func (s *orchestratorService) failBatchReceipts(batch []*Task, err error) {
	s.receiptsMutex.Lock()
	defer s.receiptsMutex.Unlock()

	for _, task := range batch {
		waiters, ok := s.receipts[task.WorkflowID]
		if !ok {
			continue
		}
		for _, receipt := range waiters {
			receipt.resolve(err)
		}
		delete(s.receipts, task.WorkflowID)
	}
}

// dispatchPersistedTasks sends tasks to the task dispatcher after they have
// been saved.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes batch ([]*Task) which contains the tasks to send.
func (s *orchestratorService) dispatchPersistedTasks(ctx context.Context, batch []*Task) {
	if s.taskDispatcher == nil {
		return
	}

	for _, task := range batch {
		s.dispatchSingleTask(ctx, task)
	}
}

// dispatchSingleTask sends a task to the appropriate queue.
//
// Takes ctx (context.Context) which carries tracing spans and cancellation.
// Takes task (*Task) which is the task to dispatch.
func (s *orchestratorService) dispatchSingleTask(ctx context.Context, task *Task) {
	ctx, l := logger_domain.From(ctx, log)
	if task.Status == StatusScheduled {
		if err := s.taskDispatcher.DispatchDelayed(ctx, task, task.ExecuteAt); err != nil {
			l.Warn("Failed to dispatch scheduled task",
				logger_domain.Error(err),
				logger_domain.String(attributeKeyTaskID, task.ID))
		}
		return
	}

	if err := s.taskDispatcher.Dispatch(ctx, task); err != nil {
		l.Warn("Failed to dispatch task",
			logger_domain.Error(err),
			logger_domain.String(attributeKeyTaskID, task.ID))
	}
}

// stopChannelTimerSafe stops a timer and drains its channel if needed.
//
// Takes timer (clock.ChannelTimer) which is the timer to stop and drain.
func stopChannelTimerSafe(timer clock.ChannelTimer) {
	if !timer.Stop() {
		select {
		case <-timer.C():
		default:
		}
	}
}
