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

package worker_domain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/worker/worker_dto"
)

// Dispatch routes a claimed job to the worker registered for its kind and runs it.
//
// Takes record (worker_dto.JobRecord) which is the claimed job to execute.
//
// Returns error which is ErrWorkerNotRegistered when no worker handles the kind, or
// whatever the worker reports.
func (p *pool) Dispatch(ctx context.Context, record worker_dto.JobRecord) error {
	handler, ok := p.registry.Lookup(record.Kind)
	if !ok {
		return ErrWorkerNotRegistered
	}
	return handler(ctx, record)
}

// HasHandler reports whether a worker has been registered for this kind.
//
// Takes kind (string) which is the job kind to look up.
//
// Returns bool which is true when a worker is registered for the kind.
func (p *pool) HasHandler(kind string) bool {
	_, ok := p.registry.Lookup(kind)
	return ok
}

// run is the poll loop: it ticks on PollFloor and claims due jobs until the context is
// cancelled or the pool begins draining.
//
// Takes wake (<-chan Wake) which signals that a queue may have new work between ticks.
func (p *pool) run(ctx context.Context, wake <-chan Wake) {
	ctx, l := logger_domain.From(ctx, log)
	ticker := p.clk.NewTicker(p.config.pollFloor)
	defer ticker.Stop()
	defer goroutine.RecoverPanic(ctx, "worker.pool.run")

	l.Notice("Worker pool started",
		logger_domain.String("worker_id", p.config.workerID),
		logger_domain.Strings("queues", p.config.queues),
		logger_domain.Int("GlobalConcurrency", p.config.globalConcurrency),
	)

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.drainCha:
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case <-p.drainCha:
			return
		case <-wake:
		case <-ticker.C():
		}
		p.claimAndDispatch(ctx)
	}
}

// claimAndDispatch claims a batch of due jobs up to the free slot count and dispatches
// each claimed row.
func (p *pool) claimAndDispatch(ctx context.Context) {
	ctx, l := logger_domain.From(ctx, log)
	free := p.freeSlots()
	if free <= 0 {
		return
	}
	rows, err := p.store.ClaimDue(ctx, p.config.workerID, free)
	if err != nil {
		l.Warn("Claim batch failed", logger_domain.Error(err))
	}
	if len(rows) == 0 {
		return
	}
	p.recordClaimed(ctx, rows)
	for i := range rows {
		p.dispatchRow(ctx, rows[i])
	}
}

// recordClaimed increments the claimed-jobs metric, partitioned by kind.
//
// Takes rows ([]worker_dto.JobRecord) which are the rows just claimed.
func (*pool) recordClaimed(ctx context.Context, rows []worker_dto.JobRecord) {
	perKind := make(map[string]int, len(rows))
	for i := range rows {
		perKind[rows[i].Kind]++
	}
	for kind, kindClaimedCount := range perKind {
		jobsClaimed.Add(ctx, int64(kindClaimedCount), metric.WithAttributes(attribute.String(attrKind, kind)))
	}
}

// dispatchRow acquires the global and per-queue slots, records the job as in-flight and
// runs it on the pool's wait group. The slots are released when the job finishes.
//
// Takes row (worker_dto.JobRecord) which is the claimed job to run.
func (p *pool) dispatchRow(ctx context.Context, row worker_dto.JobRecord) {
	p.globalSem <- struct{}{}
	if !p.acquireQueueSlot(ctx, row.Queue) {
		p.releaseSlots(row)
		return
	}
	p.inFlight.Store(row.ID, &runningJob{Record: row, StartedAt: p.clk.Now()})
	p.wg.Go(func() {
		p.execute(ctx, row)
	})
}

// execute runs a single job inside a tracing span, releasing its slots and clearing its
// in-flight entry when done, and recovers from any panic the worker raises.
//
// Takes row (worker_dto.JobRecord) which is the claimed job to run.
func (p *pool) execute(ctx context.Context, row worker_dto.JobRecord) {
	ctx, l := logger_domain.From(ctx, log)
	defer p.releaseSlots(row)
	defer p.inFlight.Delete(row.ID)
	defer goroutine.RecoverPanic(ctx, "worker.pool.execute")
	err := l.RunInSpan(ctx, "worker.execute", func(_ context.Context, _ logger_domain.Logger) error {
		p.runAndRoute(ctx, row)
		return nil
	})
	if err != nil {
		l.Warn("Job execution failed", logger_domain.String(logJobID, row.ID), logger_domain.Error(err))
	}
}

// runAndRoute dispatches a job under its timeout budget and routes the outcome, marking
// the row completed on success or failed on error and recording the matching metric.
//
// Takes row (worker_dto.JobRecord) which is the claimed job to run.
func (p *pool) runAndRoute(ctx context.Context, row worker_dto.JobRecord) {
	ctx, l := logger_domain.From(ctx, log)
	if !p.HasHandler(row.Kind) {
		l.Warn("No worker registered for kind", logger_domain.String(logJobID, row.ID), logger_domain.String(attrKind, row.Kind))
		return
	}

	l.Trace("job started", logger_domain.String(logJobID, row.ID), logger_domain.String(attrKind, row.Kind))

	persistCtx := context.WithoutCancel(ctx)

	timeout := p.config.defaultTimeout
	if row.TimeoutSeconds != 0 {
		timeout = time.Duration(row.TimeoutSeconds) * time.Second
	}
	execCtx, cancel := context.WithCancelCause(ctx)
	cause := fmt.Errorf("job %s exceeded %s timeout: %w", row.ID, timeout, context.DeadlineExceeded)
	timer := p.clk.AfterFunc(timeout, func() {
		cancel(cause)
	})
	defer timer.Stop()
	defer cancel(nil)

	start := p.clk.Now()
	workErr := goroutine.SafeCall(execCtx, "worker.pool.work", func() error {
		return p.Dispatch(execCtx, row)
	})
	jobWorkDuration.Record(ctx, float64(p.clk.Now().Sub(start))/float64(time.Millisecond),
		metric.WithAttributes(attribute.String(attrKind, row.Kind), attribute.String(attrQueue, row.Queue)))
	workErr = resolveWorkErr(execCtx, workErr)
	if workErr != nil {
		p.routeWorkErr(ctx, persistCtx, row, workErr)
		return
	}

	if err := p.store.MarkCompleted(persistCtx, row.ID, nil); err != nil {
		l.Warn("failed to mark job completed", logger_domain.String(logJobID, row.ID))
	}
	jobsCompleted.Add(ctx, 1, metric.WithAttributes(
		attribute.String(attrKind, row.Kind),
		attribute.String(attrQueue, row.Queue),
	))

	l.Internal("job completed", logger_domain.String(logJobID, row.ID), logger_domain.String(attrKind, row.Kind))
}

// routeWorkErr records the outcome of a failed attempt.
//
// Takes row (worker_dto.JobRecord) which is the job whose failure is being routed.
// Takes workErr (error) which is the resolved error from the attempt.
func (p *pool) routeWorkErr(ctx, persistCtx context.Context, row worker_dto.JobRecord, workErr error) {
	ctx, l := logger_domain.From(ctx, log)

	if _, ok := errors.AsType[*goroutine.PanicError](workErr); ok {
		l.Warn("Job panicked", logger_domain.String(logJobID, row.ID), logger_domain.Error(workErr))
		if markErr := p.store.MarkFailed(persistCtx, row.ID, workErr.Error()); markErr != nil {
			l.Warn("failed to mark job failed", logger_domain.String(logJobID, row.ID), logger_domain.Error(markErr))
		}
		jobsFailed.Add(ctx, 1, metric.WithAttributes(attribute.String(attrKind, row.Kind), attribute.String(attrOutcome, outcomePanic)))
		return
	}

	if IsFatal(workErr) {
		l.Warn("Job failed fatally", logger_domain.String(logJobID, row.ID), logger_domain.Error(workErr))
		if markErr := p.store.MarkFailed(persistCtx, row.ID, workErr.Error()); markErr != nil {
			l.Warn("failed to mark job failed", logger_domain.String(logJobID, row.ID), logger_domain.Error(markErr))
		}
		jobsFailed.Add(ctx, 1, metric.WithAttributes(attribute.String(attrKind, row.Kind), attribute.String(attrOutcome, outcomeFatal)))
		return
	}

	if ctx.Err() != nil {
		l.Warn("Job left running for recovery after shutdown cut it short",
			logger_domain.String(logJobID, row.ID), logger_domain.Error(ctx.Err()))
		return
	}

	if row.Attempt >= row.MaxAttempts {
		l.Warn("Job exhausted attempts",
			logger_domain.String(logJobID, row.ID),
			logger_domain.Int("attempt", int(row.Attempt)),
			logger_domain.Int("max_attempts", int(row.MaxAttempts)),
			logger_domain.Error(workErr),
		)
		if markErr := p.store.MarkFailed(persistCtx, row.ID, workErr.Error()); markErr != nil {
			l.Warn("failed to mark job failed", logger_domain.String(logJobID, row.ID), logger_domain.Error(markErr))
		}
		jobsFailed.Add(ctx, 1, metric.WithAttributes(attribute.String(attrKind, row.Kind), attribute.String(attrOutcome, exhaustedOutcome(workErr))))
		return
	}

	l.Warn("Job retried", logger_domain.String(logJobID, row.ID), logger_domain.Error(workErr))

	delay := p.config.Backoff(int(row.Attempt))
	runAt := p.clk.Now().Add(delay)

	if markErr := p.store.MarkRetry(persistCtx, row.ID, int(row.Attempt), runAt, workErr.Error()); markErr != nil {
		l.Warn("failed to reschedule job", logger_domain.String(logJobID, row.ID), logger_domain.Error(markErr))
	}

	jobsRetried.Add(ctx, 1, metric.WithAttributes(attribute.String(attrKind, row.Kind)))
}

// resolveWorkErr folds a tripped timeout (or cancellation) cause into the worker's raw
// return, so a job whose execCtx the per-job timeout cancelled always reaches the store
// as the richer cause rather than being misrecorded as completed.
//
// Takes execCtx (context.Context) which is the per-job context the timeout cancels.
// Takes workErr (error) which is the worker's raw return, already panic-converted.
//
// Returns error which is the outcome to persist.
func resolveWorkErr(execCtx context.Context, workErr error) error {
	if execCtx.Err() == nil {
		return workErr
	}
	cause := context.Cause(execCtx)
	if workErr == nil {
		return cause
	}
	if errors.Is(workErr, context.Canceled) || errors.Is(workErr, context.DeadlineExceeded) {
		return cause
	}
	return workErr
}

// exhaustedOutcome labels why a job exhausted its attempts: a deadline-exceeded final
// failure is a timeout, anything else is a plain exhaustion.
//
// Takes workErr (error) which is the resolved error from the final attempt.
//
// Returns string which is the outcome label (outcomeTimeout or outcomeExhausted).
func exhaustedOutcome(workErr error) string {
	if errors.Is(workErr, context.DeadlineExceeded) {
		return outcomeTimeout
	}
	return outcomeExhausted
}
