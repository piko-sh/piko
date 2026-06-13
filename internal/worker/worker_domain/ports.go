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
	"time"

	"piko.sh/piko/internal/worker/worker_dto"
)

// Notifier wakes worker loops when a queue gains work, decoupling producers from the poll
// interval so newly enqueued jobs start without waiting for the next tick.
type Notifier interface {
	// Notify wakes every subscriber interested in the given queue; empty broadcasts to all.
	Notify(ctx context.Context, queue string) error

	// Subscribe registers interest in a set of queues and returns the wake channel and an
	// unsubscribe function; an empty queue list subscribes to every queue.
	Subscribe(ctx context.Context, queues []string) (<-chan Wake, func(ctx context.Context), error)
}

// Dispatcher routes a claimed job to an appropriate registered worker for its kind.
type Dispatcher interface {
	// Dispatch routes a claimed job to the worker registered for its kind and runs it.
	Dispatch(ctx context.Context, record worker_dto.JobRecord) error

	// HasHandler reports whether a worker is registered for the given kind.
	HasHandler(kind string) bool
}

// JobStateWaiter blocks until a job reaches a terminal state and reports its final state.
type JobStateWaiter interface {
	// WaitForTerminal blocks until the job reaches a terminal state and returns it.
	WaitForTerminal(ctx context.Context, jobID string) (worker_dto.JobState, error)
}

// handlerRegistry resolves a job kind to its registered, type-erased worker.
type handlerRegistry interface {
	// Lookup returns the worker registered for a kind, and whether one exists.
	Lookup(kind string) (RegistryTypeErasedHandler, bool)
}

// Service is the registration surface workers use to bind themselves to a job kind.
type Service interface {
	// Start sweeps orphaned jobs and launches the pool and recovery loops.
	Start(ctx context.Context) error

	// Shutdown drains in-flight jobs within the context deadline and stops every loop.
	Shutdown(ctx context.Context) error

	// Enqueue persists a job from the request and wakes workers for its queue.
	Enqueue(ctx context.Context, req worker_dto.EnqueueRequest) (string, error)

	// EnqueueMany persists many jobs from one request set and wakes workers for their
	// queues.
	EnqueueMany(ctx context.Context, reqs []worker_dto.EnqueueRequest) ([]string, error)

	// WaitForTerminal blocks until the job reaches a terminal state and returns it.
	WaitForTerminal(ctx context.Context, jobID string) (worker_dto.JobState, error)

	// RegisterHandler binds a worker to a job kind, replacing any existing registration.
	RegisterHandler(kind string, handler RegistryTypeErasedHandler)

	// HasHandler reports whether a worker is registered for the given kind.
	HasHandler(kind string) bool
}

// Store is the durable backing for the jobs table.
//
// It is not generic: rows cross the boundary as the type-erased worker_dto types.
type Store interface {
	// Enqueue inserts one job into the store and returns the job's id.
	Enqueue(ctx context.Context, spec worker_dto.EnqueueSpec) (string, error)

	// EnqueueMany inserts many rows in one batch and returns their ids in spec order.
	EnqueueMany(ctx context.Context, spec []worker_dto.EnqueueSpec) ([]string, error)

	// ClaimDue atomically transitions up to limit due pending rows to running (stamping
	// workerID/claimed_at, incrementing attempt) and returns them.
	ClaimDue(ctx context.Context, workerID string, limit int) ([]worker_dto.JobRecord, error)

	// PromoteDue promotes all jobs, up to limit, which are currently scheduled to being pending.
	PromoteDue(ctx context.Context, limit int) (int, error)

	// ReclaimStale releases running rows whose claimed_at is older than now-olderThan. It
	// puts them back to pending by appending a 'recovered' version per stale job.
	ReclaimStale(ctx context.Context, olderThan time.Duration) (int, error)

	// GetJobState reads the current persisted snapshot of a job.
	GetJobState(ctx context.Context, id string) (worker_dto.JobState, error)

	// ListJobVersions returns the ordered event-sourced history of a job - every recorded
	// version from its insert to its latest transition, oldest first. Empty for an unknown
	// job.
	ListJobVersions(ctx context.Context, jobID string) ([]worker_dto.JobVersion, error)

	// MarkCompleted marks a job as being completed and provides their result.
	MarkCompleted(ctx context.Context, jobID string, result []byte) error

	// MarkRetry reschedules a job and marks the attempt with an error.
	MarkRetry(ctx context.Context, jobID string, attempt int, runAt time.Time, lastError string) error

	// MarkFailed records a terminal failure for a job, storing the last error.
	MarkFailed(ctx context.Context, jobID string, lastError string) error

	// Now returns the store clock's current instant. Pure reader: no ctx, no I/O, no error.
	Now() time.Time

	// CountPendingJobs returns the number of pending, not-deleted jobs.
	CountPendingJobs(ctx context.Context) (int64, error)

	// CountClaimableJobs returns the claimable job depth grouped by queue.
	CountClaimableJobs(ctx context.Context) ([]ClaimableJobsDepth, error)

	// CountNonTerminalJobs returns the number of non-terminal, not-deleted jobs.
	CountNonTerminalJobs(ctx context.Context) (int64, error)

	// Heartbeat renews claimed_at time on in-flight jobs to avoid being swept up
	// as a stale job. This is for long-running jobs, which run longer than the stale
	// window.
	Heartbeat(ctx context.Context, jobID, workerID string) error

	// HeartbeatMany renews claimed_at time on many in-flight jobs on the same worker
	// node, in order to avoid being swept up as stale jobs. This is for long-running jobs,
	// which run longer than the stale window.
	HeartbeatMany(ctx context.Context, jobIDs []string, workerID string) (int, error)
}
