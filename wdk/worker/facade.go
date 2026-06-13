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

// Package worker provides access to Piko's background job workers.
package worker

import (
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/json"

	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
)

type (
	// Kinded is the contract for a job's stable kind identity.
	Kinded = worker_domain.Kinded

	// Worker is the body of a fire-and-forget job parameterised by its argument type.
	Worker[T any] = worker_domain.Worker[T]

	// Job is the per-execution envelope handed to a Worker.
	Job[T any] = worker_domain.Job[T]

	// JobState is the current persisted snapshot of a job.
	JobState = worker_dto.JobState

	// Status is the lifecycle position of a job.
	Status = worker_domain.Status

	// Store is the durable backing for the jobs table.
	Store = worker_domain.Store

	// Service is the registration surface workers use to bind to a job kind.
	Service = worker_domain.Service

	// ServiceOption configures a Service at construction.
	ServiceOption = worker_domain.ServiceOption

	// Jitter spreads a base delay by a random amount to avoid synchronised retries.
	Jitter = worker_domain.Jitter

	// IDGenerator produces unique ids for job rows.
	IDGenerator = worker_domain.IDGenerator

	// Handle is a caller-side reference to an enqueued job used to await its outcome.
	Handle = worker_domain.Handle

	// EnqueueOption configures a job at enqueue time.
	EnqueueOption = worker_domain.EnqueueOption

	// Notifier wakes worker loops when a queue gains work.
	Notifier = worker_domain.Notifier

	// WorkersConfig carries the node-local timing knobs (poll floor, job timeout, recovery
	// cadence) that tune this worker node.
	WorkersConfig = worker_domain.WorkersConfig

	// UniqueScope selects which fields of an enqueue that we hash for identity dedupe.
	UniqueScope = worker_dto.UniqueScope
)

const (
	// StatusUnknown is the zero value; a job is never written in this state.
	StatusUnknown = string(worker_domain.StatusUnknown)

	// StatusPending means the job is waiting to be claimed.
	StatusPending = string(worker_domain.StatusPending)

	// StatusScheduled means the job is deferred until its scheduled time.
	StatusScheduled = string(worker_domain.StatusScheduled)

	// StatusRunning means a worker has claimed the job and is executing it.
	StatusRunning = string(worker_domain.StatusRunning)

	// StatusCompleted means the job finished successfully.
	StatusCompleted = string(worker_domain.StatusCompleted)

	// StatusFailed means the job exhausted its attempts or failed fatally.
	StatusFailed = string(worker_domain.StatusFailed)

	// StatusTimeout means the job exceeded its per-attempt budget.
	StatusTimeout = string(worker_domain.StatusTimeout)

	// StatusCancelled means the job was cancelled before it completed.
	StatusCancelled = string(worker_domain.StatusCancelled)

	// StatusRetryable means the job failed and is eligible for another attempt.
	StatusRetryable = string(worker_domain.StatusRetryable)

	// StatusDiscarded means the job was dropped without running to completion.
	StatusDiscarded = string(worker_domain.StatusDiscarded)

	// UniqueArgs dedupes on kind, queue and the json serialisation of the args.
	UniqueArgs = worker_dto.UniqueArgs

	// UniqueQueue dedupes on queue only.
	UniqueQueue = worker_dto.UniqueQueue

	// UniqueKind dedupes on queue only.
	UniqueKind = worker_dto.UniqueKind
)

// Fatal marks an error as non-retryable so the pool sends the row straight to failed.
//
// Takes err (error) which is the underlying failure, or nil for a bare fatal marker.
//
// Returns error which wraps err so the pool treats it as fatal.
func Fatal(err error) error {
	return worker_domain.Fatal(err)
}

// IsFatal reports whether err, or an error it wraps, was marked non-retryable by Fatal.
//
// Takes err (error) which is the error to inspect.
//
// Returns bool which is true when err is fatal.
func IsFatal(err error) bool {
	return worker_domain.IsFatal(err)
}

// WithClock sets the time source for the service's timers.
//
// Takes c (clock.Clock) which is the time source to use.
//
// Returns ServiceOption which records the clock.
func WithClock(c clock.Clock) ServiceOption {
	return worker_domain.WithClock(c)
}

// WithNotifier sets the notifier used to wake worker loops when a queue gains work.
//
// Takes n (Notifier) which is the notifier to use.
//
// Returns ServiceOption which records the notifier.
func WithNotifier(n Notifier) ServiceOption {
	return worker_domain.WithNotifier(n)
}

// WithQueues sets the queues the service worker loop claims from.
//
// Takes names (...string) which are the queue names to claim from.
//
// Returns ServiceOption which records the queues.
func WithQueues(names ...string) ServiceOption {
	return worker_domain.WithQueues(names...)
}

// WithJitter sets the jitter applied to retry delays.
//
// Takes jitter (Jitter) which spreads retry delays.
//
// Returns ServiceOption which records the jitter.
func WithJitter(jitter Jitter) ServiceOption {
	return worker_domain.WithJitter(jitter)
}

// WithWorkerID pins this node's worker identity instead of generating one.
//
// Takes workerID (string) which is the worker identity to use.
//
// Returns ServiceOption which records the worker id.
func WithWorkerID(workerID string) ServiceOption {
	return worker_domain.WithWorkerID(workerID)
}

// WithIDGenerator sets the generator used to allocate job row ids.
//
// Takes idGenerator (IDGenerator) which is the id source to use.
//
// Returns ServiceOption which records the generator.
func WithIDGenerator(idGenerator IDGenerator) ServiceOption {
	return worker_domain.WithIDGenerator(idGenerator)
}

// WithGlobalConcurrency caps the number of simultaneously in-flight jobs.
//
// Takes globalConcurrency (int) which is the maximum number of in-flight jobs.
//
// Returns ServiceOption which records the cap.
func WithGlobalConcurrency(globalConcurrency int) ServiceOption {
	return worker_domain.WithGlobalConcurrency(globalConcurrency)
}

// WithConfig threads the node-local WorkersConfig into the service, backfilling any unset
// field with its default.
//
// Takes cfg (WorkersConfig) which carries the node-local timing knobs.
//
// Returns ServiceOption which records the config.
func WithConfig(cfg WorkersConfig) ServiceOption {
	return worker_domain.WithConfig(cfg)
}

// WithMaxAttempts sets the cap on the total number of runs, retries included, for an
// enqueued job.
//
// Takes n (int) which is the maximum number of attempts.
//
// Returns EnqueueOption which records the cap.
func WithMaxAttempts(n int) EnqueueOption {
	return worker_domain.WithMaxAttempts(n)
}

// WithTimeout sets the per-attempt wall-clock budget for an enqueued job.
//
// Takes d (time.Duration) which is the per-attempt budget.
//
// Returns EnqueueOption which records the timeout.
func WithTimeout(d time.Duration) EnqueueOption {
	return worker_domain.WithTimeout(d)
}

// WithQueue routes an enqueued job to a named queue.
//
// Takes name (string) which is the target queue.
//
// Returns EnqueueOption which records the queue.
func WithQueue(name string) EnqueueOption {
	return worker_domain.WithQueue(name)
}

// WithDelay defers an enqueued job by a duration before it becomes eligible to run.
//
// Takes duration (time.Duration) which is the deferral before the first run.
//
// Returns EnqueueOption which records the delay.
func WithDelay(duration time.Duration) EnqueueOption {
	return worker_domain.WithDelay(duration)
}

// WithRunAt schedules an enqueued job to first become eligible at an absolute time.
//
// Takes runAt (time.Time) which is the absolute first-run time.
//
// Returns EnqueueOption which records the run-at time.
func WithRunAt(runAt time.Time) EnqueueOption {
	return worker_domain.WithRunAt(runAt)
}

// WithIdempotencyKey sets the caller-supplied key used to dedupe an enqueued job.
//
// Takes key (string) which is the caller-supplied dedupe key.
//
// Returns EnqueueOption which records the UniqueKey.
func WithIdempotencyKey(key string) EnqueueOption {
	return worker_domain.WithIdempotencyKey(key)
}

// WithIdempotencyBy dedupes an enqueued job by a unique scope over a time window.
//
// Takes scope (worker_dto.UniqueScope) which selects the fields the dedupe key hashes.
// Takes window (time.Duration) which buckets the key by time; 0 disables bucketing.
//
// Returns EnqueueOption which records the derived UniqueKey.
func WithIdempotencyBy(scope worker_dto.UniqueScope, window time.Duration) EnqueueOption {
	return worker_domain.WithIdempotencyBy(scope, window)
}

// WithCorrelationID tags an enqueued job with a caller-supplied trace token.
//
// Takes id (string) which is the caller-supplied trace token.
//
// Returns EnqueueOption which records the CorrelationID.
func WithCorrelationID(id string) EnqueueOption {
	return worker_domain.WithCorrelationID(id)
}

// WithPriority sets the claim-ordering weight for an enqueued job.
//
// Takes priority (int64) which is the claim-ordering weight.
//
// Returns EnqueueOption which records the priority.
func WithPriority(priority int64) EnqueueOption {
	return worker_domain.WithPriority(priority)
}

// RegisterOptions is the set of options accepted when registering a worker.
type RegisterOptions any

// NewService builds a worker Service backed by the given store.
//
// Takes store (Store) which is the durable backing for jobs.
//
// Returns Service which is the ready service.
func NewService(store Store, opts ...ServiceOption) Service {
	return worker_domain.NewService(store, opts...)
}

// Register binds a typed worker to its job kind on the service.
//
// The kind is taken from the zero value of T, so each worker type maps to exactly one
// kind. Any trailing RegisterOptions are accepted but have no effect on the registration.
//
// Takes workerService (Service) which is the service to register on.
// Takes worker (Worker[T]) which is the worker to run for the kind.
func Register[T Kinded](workerService Service, worker Worker[T], _ ...RegisterOptions) {
	var zero T
	workerService.RegisterHandler(zero.Kind(), newHandler(worker))
}

// Enqueue encodes the typed args and enqueues a job of their kind on the service,
// returning a Handle the caller can await.
//
// Takes workerService (Service) which is the service to enqueue on.
// Takes args (T) which are the typed job arguments; their Kind selects the worker.
// Takes opts (...EnqueueOption) which configure the enqueued job.
//
// Returns *Handle which references the enqueued job.
// Returns error when the args cannot be encoded or the enqueue fails.
func Enqueue[T Kinded](ctx context.Context, workerService Service, args T, opts ...EnqueueOption) (*Handle, error) {
	payload, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("encoding args for kind %q: %w", args.Kind(), err)
	}
	req := worker_dto.EnqueueRequest{
		Kind:    args.Kind(),
		Payload: payload,
	}
	for _, opt := range opts {
		opt(&req)
	}
	jobID, err := workerService.Enqueue(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("enqueue %q: %w", req.Kind, err)
	}

	return worker_domain.NewHandle(jobID, workerService), nil
}

// EnqueueMany marshals each item to a job payload and enqueues them as a batch on the worker
// service, returning a Handle per enqueued job.
//
// Takes workerService (Service) which the jobs are enqueued on.
// Takes items ([]T) which are the typed job payloads to enqueue.
// Takes opts (...EnqueueOption) which customise every request in the batch.
//
// Returns []*Handle which are the handles for the enqueued jobs, in item order.
// Returns error when marshalling or the batch enqueue fails.
func EnqueueMany[T Kinded](ctx context.Context, workerService Service, items []T, opts ...EnqueueOption) ([]*Handle, error) {
	if len(items) == 0 {
		return nil, nil
	}

	reqs := make([]worker_dto.EnqueueRequest, len(items))
	for i, item := range items {
		payload, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("encoding args for kind %q: %w", item.Kind(), err)
		}

		req := worker_dto.EnqueueRequest{
			Kind:    item.Kind(),
			Payload: payload,
		}
		for _, opt := range opts {
			opt(&req)
		}
		reqs[i] = req
	}

	jobIDs, err := workerService.EnqueueMany(ctx, reqs)
	if err != nil {
		return nil, fmt.Errorf("enqueue batch of %d: %w", len(items), err)
	}

	handles := make([]*Handle, len(jobIDs))
	for i, id := range jobIDs {
		handles[i] = worker_domain.NewHandle(id, workerService)
	}

	return handles, nil
}

// newHandler adapts a typed worker to the registry's type-erased handler, decoding each
// job's payload into T before invoking the worker.
//
// Takes worker (Worker[T]) which is the typed worker to adapt.
//
// Returns worker_domain.RegistryTypeErasedHandler which decodes the payload and runs the
// worker.
func newHandler[T Kinded](worker Worker[T]) worker_domain.RegistryTypeErasedHandler {
	return func(ctx context.Context, record worker_dto.JobRecord) error {
		var args T
		if err := json.Unmarshal(record.Payload, &args); err != nil {
			return worker_domain.Fatal(fmt.Errorf(
				"decoding payload for kind %q job %q: %w",
				record.Kind,
				record.ID,
				err,
			))
		}
		job := Job[T]{
			Args:       args,
			Attempt:    record.Attempt,
			MaxAttempt: record.MaxAttempts,
			ID:         record.ID,
			EnqueueAt:  record.EnqueueAt,
		}

		return worker.Work(ctx, job)
	}
}
