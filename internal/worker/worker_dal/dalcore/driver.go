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

package dalcore

import (
	"context"
	"database/sql"
)

// Driver is the per-dialect seam between Core and a generated query package.
//
// Its implementations live in each querier_<dialect> package and perform the trivial
// translation between the dialect-neutral flat structs below and the generated database
// types. Read methods return the raw sql.ErrNoRows so Core can map it to the appropriate
// domain sentinel.
type Driver interface {
	// WithTx returns a driver whose queries run inside tx.
	WithTx(tx *sql.Tx) Driver

	// InsertJobRoot inserts the immutable job root.
	InsertJobRoot(ctx context.Context, params InsertJobRootParams) error

	// InsertJobRootBatch inserts many immutable job roots in one round-trip.
	InsertJobRootBatch(ctx context.Context, params []InsertJobRootParams) error

	// AppendJobVersion appends one immutable version to a job's log; the projection trigger
	// updates the current-state index.
	AppendJobVersion(ctx context.Context, params AppendJobVersionParams) error

	// AppendJobVersionBatch appends many versions in one round-trip.
	AppendJobVersionBatch(ctx context.Context, params []AppendJobVersionParams) error

	// ClaimCandidates reads the due, pending, not-deleted rows from the projection, ordered
	// priority DESC, scheduled_at ASC.
	ClaimCandidates(ctx context.Context, params ClaimCandidatesParams) ([]ClaimCandidateRow, error)

	// PromoteCandidates reads the due, scheduled and retryable, not-deleted rows eligible
	// for promotion to pending, ordered priority DESC, scheduled_at ASC, up to the limit.
	PromoteCandidates(ctx context.Context, params PromoteCandidatesParams) ([]PromoteCandidateRow, error)

	// StaleRunningJobs reads running rows whose claimed_at is at or before cutoff.
	StaleRunningJobs(ctx context.Context, cutoff string) ([]StaleJobRow, error)

	// GetJob reads the joined root + projection + latest-version snapshot; returns raw
	// sql.ErrNoRows when the job is unknown.
	GetJob(ctx context.Context, id string) (JobRow, error)

	// GetJobIndex reads the current projection columns a version append carries forward.
	GetJobIndex(ctx context.Context, jobID string) (JobIndexRow, error)

	// ListJobVersions reads a job's append-only history, oldest first.
	ListJobVersions(ctx context.Context, jobID string) ([]JobVersionRow, error)

	// CountPendingJobs counts pending, not-deleted rows.
	CountPendingJobs(ctx context.Context) (int64, error)

	// CountClaimableJobs counts claimable rows grouped by queue.
	CountClaimableJobs(ctx context.Context) ([]QueueDepthRow, error)

	// CountNonTerminalJobs counts non-terminal, not-deleted rows.
	CountNonTerminalJobs(ctx context.Context) (int64, error)

	// HeartbeatJob renews one running jobs lease.
	HeartbeatJob(ctx context.Context, params HeartbeatJobParams) error

	// HeartbeatJobs renews every running jobs lease that a particular worker owns.
	HeartbeatJobs(ctx context.Context, params HeartbeatJobsParams) (int64, error)

	// InsertJobRootWithUniqueKey inserts a unique-keyed root; it does nothing on a unique
	// key clash and returns the inserted id, or the raw sql.ErrNoRows when the key was
	// already taken.
	InsertJobRootWithUniqueKey(ctx context.Context, params InsertJobRootParams) (string, error)

	// GetJobIDByUniqueKey resolves the id of the job already holding the unique key;
	// it returns the raw sql.ErrNoRows sentinel when no row holds the key.
	GetJobIDByUniqueKey(ctx context.Context, uniqueKey string) (string, error)
}

type HeartbeatJobParams struct {
	ClaimedAt         *string
	ClaimedByWorkerID *string
	JobID             string
}

type HeartbeatJobsParams struct {
	ClaimedAt         *string
	ClaimedByWorkerID *string
	JobIDs            []string
}

// InsertJobRootParams is the dialect-neutral immutable-root insert.
type InsertJobRootParams struct {
	// ID is the caller-supplied unique job id.
	ID string

	// Kind is the job kind that selects its handler.
	Kind string

	// Queue is the queue the job belongs to.
	Queue string

	// Payload is the serialised job payload.
	Payload string

	// CreatedAt is the RFC3339Nano UTC time the job root was created.
	CreatedAt string

	// MaxAttempts is the maximum number of run attempts allowed.
	MaxAttempts int32

	// TimeoutSeconds is the per-attempt timeout in seconds.
	TimeoutSeconds int32

	// CorrelationID is the nullable trace correlation token stored on the job root.
	CorrelationID *string

	// UniqueKey is the nullable idempotency token that deduplicates enqueued job roots.
	UniqueKey *string
}

// AppendJobVersionParams is the dialect-neutral version append.
type AppendJobVersionParams struct {
	// LastError is the nullable last error message recorded by this version.
	LastError *string

	// Result is the nullable serialised job result recorded by this version.
	Result *string

	// ClaimedByWorkerID is the nullable id of the worker that claimed the job.
	ClaimedByWorkerID *string

	// ClaimedAt is the nullable RFC3339Nano UTC time the job was claimed.
	ClaimedAt *string

	// DeletedAt is the nullable RFC3339Nano UTC time the job was soft-deleted.
	DeletedAt *string

	// JobID is the id of the parent job this version belongs to.
	JobID string

	// Event is the event name that produced this version.
	Event string

	// Status is the job status after this event was applied.
	Status string

	// ScheduledAt is the RFC3339Nano UTC time the job becomes eligible to run.
	ScheduledAt string

	// Priority is the scheduling priority carried by this version.
	Priority int32

	// Attempt is the attempt number recorded by this version.
	Attempt int32
}

// ClaimCandidatesParams bounds a claim read.
type ClaimCandidatesParams struct {
	// ScheduledAt is the RFC3339Nano UTC cutoff; due rows are at or before it.
	ScheduledAt string

	// Limit is the maximum number of candidate rows to read.
	Limit int
}

// ClaimCandidateRow is a joined projection + root claim candidate.
type ClaimCandidateRow struct {
	// JobID is the id of the candidate job.
	JobID string

	// ScheduledAt is the RFC3339Nano UTC time the job became eligible to run.
	ScheduledAt string

	// Kind is the job kind that selects its handler.
	Kind string

	// Queue is the queue the job belongs to.
	Queue string

	// Payload is the serialised job payload.
	Payload string

	// CreatedAt is the RFC3339Nano UTC time the job root was created.
	CreatedAt string

	// CurrentVersionSequence is the sequence of the job's latest version.
	CurrentVersionSequence int32

	// Priority is the current scheduling priority from the projection.
	Priority int32

	// Attempt is the number of attempts made so far.
	Attempt int32

	// MaxAttempts is the maximum number of run attempts allowed.
	MaxAttempts int32

	// TimeoutSeconds is the per-attempt timeout in seconds.
	TimeoutSeconds int32
}

// PromoteCandidatesParams bounds a promote read.
type PromoteCandidatesParams struct {
	// ScheduledAt is the RFC3339Nano UTC cutoff; due rows are at or before it.
	ScheduledAt string

	// Limit is the maximum number of candidate rows to read.
	Limit int
}

// PromoteCandidateRow is a joined projection + root promote candidate.
type PromoteCandidateRow struct {
	// JobID is the id of the candidate job.
	JobID string

	// ScheduledAt is the RFC3339Nano UTC time the job became eligible to run.
	ScheduledAt string

	// Priority is the current scheduling priority from the projection.
	Priority int32

	// Attempt is the number of attempts made so far.
	Attempt int32
}

// JobRow is the joined current-snapshot row.
type JobRow struct {
	// LastError is the nullable last error message from the latest version.
	LastError *string

	// ID is the unique job id.
	ID string

	// Kind is the job kind that selects its handler.
	Kind string

	// Queue is the queue the job belongs to.
	Queue string

	// CreatedAt is the RFC3339Nano UTC time the job root was created.
	CreatedAt string

	// Status is the job's current status from the projection.
	Status string

	// ScheduledAt is the RFC3339Nano UTC time the job becomes eligible to run.
	ScheduledAt string

	// UpdatedAt is the RFC3339Nano UTC time the latest version was recorded.
	UpdatedAt string

	// MaxAttempts is the maximum number of run attempts allowed.
	MaxAttempts int32

	// Priority is the job's current scheduling priority.
	Priority int32

	// Attempt is the job's current attempt number.
	Attempt int32
}

// JobIndexRow is the current projection's forward-carried columns.
type JobIndexRow struct {
	// ScheduledAt is the RFC3339Nano UTC time the job becomes eligible to run.
	ScheduledAt string

	// Priority is the job's current scheduling priority.
	Priority int32

	// Attempt is the job's current attempt number.
	Attempt int32
}

// JobVersionRow is one row of the append-only history.
type JobVersionRow struct {
	// LastError is the nullable last error message recorded by this version.
	LastError *string

	// Result is the nullable serialised job result recorded by this version.
	Result *string

	// ClaimedByWorkerID is the nullable id of the worker that claimed the job.
	ClaimedByWorkerID *string

	// ClaimedAt is the nullable RFC3339Nano UTC time the job was claimed.
	ClaimedAt *string

	// DeletedAt is the nullable RFC3339Nano UTC time the job was soft-deleted.
	DeletedAt *string

	// JobID is the id of the parent job this version belongs to.
	JobID string

	// Event is the event name that produced this version.
	Event string

	// Status is the job status after this event was applied.
	Status string

	// ScheduledAt is the RFC3339Nano UTC time the job becomes eligible to run.
	ScheduledAt string

	// RecordedAt is the RFC3339Nano UTC time this version was recorded.
	RecordedAt string

	// VersionSequence is the monotonic sequence number of this version.
	VersionSequence int64

	// Priority is the scheduling priority recorded by this version.
	Priority int32

	// Attempt is the attempt number recorded by this version.
	Attempt int32
}

// StaleJobRow is a running row eligible for reclaim.
type StaleJobRow struct {
	// JobID is the id of the stale running job.
	JobID string

	// ScheduledAt is the RFC3339Nano UTC time the job became eligible to run.
	ScheduledAt string

	// CurrentVersionSequence is the sequence of the job's latest version.
	CurrentVersionSequence int32

	// Priority is the job's current scheduling priority.
	Priority int32

	// Attempt is the job's current attempt number.
	Attempt int32
}

// QueueDepthRow is one row of the grouped claimable-depth count.
type QueueDepthRow struct {
	// Queue is the queue the count applies to.
	Queue string

	// Count is the number of claimable jobs in the queue.
	Count int64
}
