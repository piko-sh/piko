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
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/internal/worker/worker_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// tsLayout is how the core writes every timestamp across the DB boundary.
	tsLayout = time.RFC3339Nano

	// dbDefaultLayout is the shape a DB-defaulted datetime('now') / now() column reads back
	// as, on both SQLite and Postgres.
	dbDefaultLayout = "2006-01-02 15:04:05"

	// claimBatchHardCap is the absolute ceiling a single claim round-trip is clamped to, so
	// an over-large caller limit cannot pin the single writer.
	claimBatchHardCap = 1000

	// enqueueBatchHardCap is the absolute ceiling on a single EnqueueMany call.
	enqueueBatchHardCap = 10000

	// errAppendEnqueuedVersion wraps a failure to append a job version.
	errAppendEnqueuedVersion = "failed to append enqueued version: %w"
)

var (
	_ worker_domain.Store = (*core)(nil)

	// errDALNotInitialised is returned when a transaction is requested but no *sql.DB is
	// bound to the core.
	errDALNotInitialised = errors.New("worker dal: no database connection")
)

// core is the dialect-agnostic worker Store: it owns the transaction lifecycle, the
// injected clock, timestamp (de)serialisation and the event-sourced domain mapping,
// delegating every generated query to a per-dialect Driver.
type core struct {
	// sqlDB is the open connection the transaction lifecycle runs against.
	sqlDB *sql.DB

	// driver is the per-dialect query seam.
	driver Driver

	// clock is the injected time source every row timestamp is stamped from.
	clock clock.Clock
}

// New builds a worker Store from an open *sql.DB, a dialect driver and a clock.
//
// Takes database (*sql.DB) which the transaction lifecycle runs against.
// Takes driver (Driver) which is the per-dialect query implementation.
// Takes clk (clock.Clock) which is the time source; nil falls back to RealClock().
//
// Returns worker_domain.Store which is the ready store.
func New(database *sql.DB, driver Driver, clk clock.Clock) worker_domain.Store {
	if clk == nil {
		clk = clock.RealClock()
	}
	return &core{sqlDB: database, driver: driver, clock: clk}
}

// PromoteDue promotes due scheduled and retryable jobs to pending, up to limit.
//
// It reads jobs whose status is scheduled or retryable and whose scheduled_at is at
// or before the store clock, then appends a pending 'promoted' version to each. The
// limit is clamped to at least one and at most claimBatchHardCap.
//
// Takes limit (int) which caps the jobs promoted, clamped to the batch ceiling.
//
// Returns int which is the number of jobs promoted to pending.
// Returns error when the context is cancelled or the promote transaction fails.
func (c *core) PromoteDue(ctx context.Context, limit int) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("promote due: %w", err)
	}
	batch := min(max(limit, 1), claimBatchHardCap)
	now := c.now()
	candidateCount := 0

	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		candidates, err := driver.PromoteCandidates(ctx, PromoteCandidatesParams{
			ScheduledAt: now,
			Limit:       batch,
		})
		if err != nil {
			return fmt.Errorf("failed to read promote candidates: %w", err)
		}
		candidateCount = len(candidates)

		for i := range candidates {
			candidate := candidates[i]
			if err := driver.AppendJobVersion(ctx, AppendJobVersionParams{
				JobID:       candidate.JobID,
				Event:       "promoted",
				Status:      string(worker_domain.StatusPending),
				Priority:    candidate.Priority,
				ScheduledAt: candidate.ScheduledAt,
				Attempt:     candidate.Attempt,
			}); err != nil {
				return fmt.Errorf("failed to append promoted version for job %q: %w", candidate.JobID, err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to promote jobs: %w", err)
	}
	return candidateCount, nil
}

// Enqueue writes the immutable root, then appends its enqueued pending version.
//
// Takes spec (worker_dto.EnqueueSpec) describing the job to enqueue.
//
// Returns string which is the enqueued job ID, echoed from spec.ID.
// Returns error when the context is cancelled or the insert transaction fails.
func (c *core) Enqueue(ctx context.Context, spec worker_dto.EnqueueSpec) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("enqueue: %w", err)
	}
	now := c.now()

	if spec.UniqueKey != "" {
		return c.enqueueWithDedupe(ctx, spec)
	}

	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		if err := driver.InsertJobRoot(ctx, InsertJobRootParams{
			ID:             spec.ID,
			Kind:           spec.Kind,
			Queue:          spec.Queue,
			Payload:        string(spec.Payload),
			CorrelationID:  optionalText(spec.CorrelationID),
			MaxAttempts:    safeconv.Int64ToInt32(spec.MaxAttempts),
			TimeoutSeconds: safeconv.Int64ToInt32(spec.TimeoutSeconds),
			CreatedAt:      now,
		}); err != nil {
			return fmt.Errorf("failed to insert job root: %w", err)
		}

		initialStatus := string(worker_domain.StatusPending)
		if spec.ScheduledAt.After(c.clock.Now()) {
			initialStatus = string(worker_domain.StatusScheduled)
		}

		if err := driver.AppendJobVersion(ctx, AppendJobVersionParams{
			JobID:       spec.ID,
			Event:       "enqueued",
			Status:      initialStatus,
			Priority:    safeconv.Int64ToInt32(spec.Priority),
			ScheduledAt: formatTime(spec.ScheduledAt),
			Attempt:     0,
		}); err != nil {
			return fmt.Errorf(errAppendEnqueuedVersion, err)
		}

		return nil
	})
	return spec.ID, err
}

// enqueueWithDedupe inserts a unique-keyed job root, deduplicating on the key.
//
// It inserts the root honouring the unique idempotency key; on a fresh insert it
// also appends the enqueued pending version (scheduled when the run time is in the
// future) and returns the inserted id. On a unique-key conflict it inserts nothing
// and instead resolves the id of the existing job already holding that key.
//
// Takes spec (worker_dto.EnqueueSpec) describing the job to enqueue, with a set
// UniqueKey.
//
// Returns string which is the resolved job id: the newly inserted id, or the
// existing job's id on a key conflict.
// Returns error when the context is cancelled, the insert fails, or the duplicate
// cannot be resolved (ErrJobNotFound when the conflicting row vanished).
func (c *core) enqueueWithDedupe(ctx context.Context, spec worker_dto.EnqueueSpec) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("enqueue with dedupe: %w", err)
	}
	now := c.now()

	resolvedID := spec.ID

	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		insertedId, insertedErr := driver.InsertJobRootWithUniqueKey(ctx, InsertJobRootParams{
			ID:             spec.ID,
			Kind:           spec.Kind,
			Queue:          spec.Queue,
			Payload:        string(spec.Payload),
			UniqueKey:      optionalText(spec.UniqueKey),
			CorrelationID:  optionalText(spec.CorrelationID),
			MaxAttempts:    safeconv.Int64ToInt32(spec.MaxAttempts),
			TimeoutSeconds: safeconv.Int64ToInt32(spec.TimeoutSeconds),
			CreatedAt:      now,
		})

		if insertedErr == nil {
			initialStatus := string(worker_domain.StatusPending)
			if spec.ScheduledAt.After(c.clock.Now()) {
				initialStatus = string(worker_domain.StatusScheduled)
			}

			if err := driver.AppendJobVersion(ctx, AppendJobVersionParams{
				JobID:       spec.ID,
				Event:       "enqueued",
				Status:      initialStatus,
				Priority:    safeconv.Int64ToInt32(spec.Priority),
				ScheduledAt: formatTime(spec.ScheduledAt),
				Attempt:     0,
			}); err != nil {
				return fmt.Errorf(errAppendEnqueuedVersion, err)
			}

			resolvedID = insertedId
			return nil
		}

		if !errors.Is(insertedErr, sql.ErrNoRows) {
			return fmt.Errorf("inserting unique job %q (kind %q, key %q): %w", spec.ID, spec.Kind, spec.UniqueKey, insertedErr)
		}

		existingID, lookupErr := driver.GetJobIDByUniqueKey(ctx, spec.UniqueKey)
		if lookupErr != nil {
			if errors.Is(lookupErr, sql.ErrNoRows) {
				return fmt.Errorf("resolving duplicate for key %q (kind %q): %w", spec.UniqueKey, spec.Kind, worker_domain.ErrJobNotFound)
			}
			return fmt.Errorf("resolving duplicate for key %q (kind %q): %w", spec.UniqueKey, spec.Kind, lookupErr)
		}

		resolvedID = existingID

		return nil
	})
	return resolvedID, err
}

// EnqueueMany inserts a batch of job roots and their enqueued versions.
//
// Takes specs ([]worker_dto.EnqueueSpec) which are the jobs to insert together.
//
// Returns []string which are the enqueued job IDs, in the order supplied.
// Returns error when the batch exceeds the cap, ctx is cancelled, or the insert fails.
func (c *core) EnqueueMany(ctx context.Context, specs []worker_dto.EnqueueSpec) ([]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("enqueue batch: %w", err)
	}
	if len(specs) > enqueueBatchHardCap {
		return nil, fmt.Errorf("enqueuing %d jobs exceeds the %d batch cap: %w",
			len(specs),
			enqueueBatchHardCap,
			worker_domain.ErrBatchTooLarge,
		)
	}

	roots := make([]InsertJobRootParams, len(specs))
	versions := make([]AppendJobVersionParams, len(specs))
	ids := make([]string, len(specs))
	now := c.now()

	for i := range specs {
		roots[i] = InsertJobRootParams{
			ID:             specs[i].ID,
			Kind:           specs[i].Kind,
			Queue:          specs[i].Queue,
			Payload:        string(specs[i].Payload),
			CorrelationID:  optionalText(specs[i].CorrelationID),
			MaxAttempts:    safeconv.Int64ToInt32(specs[i].MaxAttempts),
			TimeoutSeconds: safeconv.Int64ToInt32(specs[i].TimeoutSeconds),
			CreatedAt:      now,
		}

		initialStatus := string(worker_domain.StatusPending)
		if specs[i].ScheduledAt.After(c.clock.Now()) {
			initialStatus = string(worker_domain.StatusScheduled)
		}

		versions[i] = AppendJobVersionParams{
			JobID:       specs[i].ID,
			Event:       "enqueued",
			Status:      initialStatus,
			Priority:    safeconv.Int64ToInt32(specs[i].Priority),
			ScheduledAt: formatTime(specs[i].ScheduledAt),
			Attempt:     0,
		}
		ids[i] = specs[i].ID
	}

	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		if err := driver.InsertJobRootBatch(ctx, roots); err != nil {
			return fmt.Errorf("failed to insert job root: %w", err)
		}
		if err := driver.AppendJobVersionBatch(ctx, versions); err != nil {
			return fmt.Errorf(errAppendEnqueuedVersion, err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("inserting %d jobs in batch: %w", len(specs), err)
	}

	return ids, nil
}

// ClaimDue is the pending->running transition; there is no separate MarkRunning verb.
//
// Takes workerID (string) which stamps the claimed rows with this node's identity.
// Takes limit (int) which caps the number of jobs claimed, clamped to the batch ceiling.
//
// Returns []worker_dto.JobRecord which are the newly claimed, running jobs.
// Returns error when the claim transaction fails.
func (c *core) ClaimDue(ctx context.Context, workerID string, limit int) ([]worker_dto.JobRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("claim due: %w", err)
	}
	batch := min(max(limit, 1), claimBatchHardCap)
	now := c.now()
	var out []worker_dto.JobRecord

	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		candidates, err := driver.ClaimCandidates(ctx, ClaimCandidatesParams{
			ScheduledAt: now,
			Limit:       batch,
		})
		if err != nil {
			return fmt.Errorf("failed to read claim candidates: %w", err)
		}

		out = make([]worker_dto.JobRecord, 0, len(candidates))
		for i := range candidates {
			candidate := candidates[i]
			attempt := int64(candidate.Attempt) + 1
			if err := driver.AppendJobVersion(ctx, AppendJobVersionParams{
				JobID:             candidate.JobID,
				Event:             "claimed",
				Status:            string(worker_domain.StatusRunning),
				Priority:          candidate.Priority,
				ScheduledAt:       candidate.ScheduledAt,
				Attempt:           safeconv.Int64ToInt32(attempt),
				ClaimedByWorkerID: &workerID,
				ClaimedAt:         &now,
			}); err != nil {
				return fmt.Errorf("failed to append claimed version for job %q: %w", candidate.JobID, err)
			}

			record, err := claimCandidateToRecord(candidate, workerID, attempt, now)
			if err != nil {
				return fmt.Errorf("failed to map claimed job %q: %w", candidate.JobID, err)
			}
			out = append(out, record)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to claim jobs: %w", err)
	}
	return out, nil
}

// ReclaimStale releases running rows whose claimed_at is older than now-olderThan.
//
// Takes olderThan (time.Duration) which is the age past which a claim is stale.
//
// Returns int which is the count of stale running jobs returned to pending.
// Returns error when the context is cancelled or the reclaim transaction fails.
func (c *core) ReclaimStale(ctx context.Context, olderThan time.Duration) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("reclaim stale: %w", err)
	}
	cutoff := formatTime(c.clock.Now().Add(-olderThan))
	reclaimed := 0

	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		stale, staleErr := driver.StaleRunningJobs(ctx, cutoff)
		if staleErr != nil {
			return fmt.Errorf("reading stale running jobs: %w", staleErr)
		}
		for i := range stale {
			row := stale[i]
			if appendErr := driver.AppendJobVersion(ctx, AppendJobVersionParams{
				JobID:       row.JobID,
				Event:       "recovered",
				Status:      string(worker_domain.StatusPending),
				Priority:    row.Priority,
				ScheduledAt: row.ScheduledAt,
				Attempt:     row.Attempt,
			}); appendErr != nil {
				return fmt.Errorf("appending recovered version for job %q: %w", row.JobID, appendErr)
			}
			reclaimed++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("reclaiming jobs claimed before %s: %w", cutoff, err)
	}

	return reclaimed, nil
}

// GetJobState reads the current snapshot; a missing row maps to ErrJobNotFound.
//
// Takes id (string) which is the job ID to read the current snapshot for.
//
// Returns worker_dto.JobState which is the job's current projected state.
// Returns error which is ErrJobNotFound for a missing row, or a parse/read failure.
func (c *core) GetJobState(ctx context.Context, id string) (worker_dto.JobState, error) {
	if err := ctx.Err(); err != nil {
		return worker_dto.JobState{}, fmt.Errorf("get job state: %w", err)
	}
	job, err := c.driver.GetJob(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return worker_dto.JobState{}, fmt.Errorf("failed to get job %q: %w", id, worker_domain.ErrJobNotFound)
		}
		return worker_dto.JobState{}, fmt.Errorf("failed to get job %q: %w", id, err)
	}
	return getRowToState(job)
}

// ListJobVersions returns one job's append-only history, oldest first.
//
// Takes jobID (string) which is the job whose version history is read.
//
// Returns []worker_dto.JobVersion which are the job's versions, oldest first.
// Returns error when the context is cancelled, the read fails, or a row fails to map.
func (c *core) ListJobVersions(ctx context.Context, jobID string) ([]worker_dto.JobVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("list job versions: %w", err)
	}
	rows, err := c.driver.ListJobVersions(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for job %q: %w", jobID, err)
	}

	out := make([]worker_dto.JobVersion, len(rows))
	for i := range rows {
		version, err := versionRowToVersion(rows[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map version %d of job %q: %w", rows[i].VersionSequence, jobID, err)
		}
		out[i] = version
	}
	return out, nil
}

// MarkCompleted appends a terminal 'completed' version for a job, storing its result.
//
// Takes jobID (string) which is the job to mark completed.
// Takes result ([]byte) which is the JSON result stored on the terminal version.
//
// Returns error when the context is cancelled or the version cannot be appended.
func (c *core) MarkCompleted(ctx context.Context, jobID string, result []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	return c.appendFromCurrent(ctx, jobID, "completed", string(worker_domain.StatusCompleted), func(params *AppendJobVersionParams) {
		params.Result = optionalJSON(result)
	})
}

// MarkFailed appends a terminal 'failed' version for a job, storing the last error.
//
// Takes jobID (string) which is the job to mark failed.
// Takes lastError (string) which is the final error stored on the terminal version.
//
// Returns error when the context is cancelled or the version cannot be appended.
func (c *core) MarkFailed(ctx context.Context, jobID string, lastError string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return c.appendFromCurrent(ctx, jobID, "failed", string(worker_domain.StatusFailed), func(params *AppendJobVersionParams) {
		params.LastError = optionalText(lastError)
	})
}

// MarkRetry appends a pending 'retried' version, rescheduling the job for a later run.
//
// Takes jobID (string) which is the job to reschedule.
// Takes attempt (int) which is the attempt number stamped on the retried version.
// Takes runAt (time.Time) which is when the job next becomes claimable.
// Takes lastError (string) which is the error from the attempt that just failed.
//
// Returns error when the context is cancelled or the version cannot be appended.
func (c *core) MarkRetry(ctx context.Context, jobID string, attempt int, runAt time.Time, lastError string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("mark retry: %w", err)
	}
	return c.appendFromCurrent(ctx, jobID, "retried", string(worker_domain.StatusPending), func(params *AppendJobVersionParams) {
		params.ScheduledAt = formatTime(runAt)
		params.Attempt = safeconv.IntToInt32(attempt)
		params.LastError = optionalText(lastError)
	})
}

// CountPendingJobs returns the number of pending, not-deleted jobs.
//
// Returns int64 which is the number of pending, not-deleted jobs.
// Returns error when the count query fails.
func (c *core) CountPendingJobs(ctx context.Context) (int64, error) {
	count, err := c.driver.CountPendingJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("counting pending jobs: %w", err)
	}
	return count, nil
}

// CountClaimableJobs returns the claimable depth grouped by queue.
//
// Returns []worker_domain.ClaimableJobsDepth which is the claimable depth per queue.
// Returns error when the count query fails.
func (c *core) CountClaimableJobs(ctx context.Context) ([]worker_domain.ClaimableJobsDepth, error) {
	rows, err := c.driver.CountClaimableJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting claimable jobs: %w", err)
	}
	out := make([]worker_domain.ClaimableJobsDepth, len(rows))
	for i := range rows {
		out[i] = worker_domain.ClaimableJobsDepth{
			Queue: rows[i].Queue,
			Count: rows[i].Count,
		}
	}
	return out, nil
}

// CountNonTerminalJobs returns the number of non-terminal, not-deleted jobs.
//
// Returns int64 which is the number of non-terminal, not-deleted jobs.
// Returns error when the count query fails.
func (c *core) CountNonTerminalJobs(ctx context.Context) (int64, error) {
	count, err := c.driver.CountNonTerminalJobs(ctx)
	if err != nil {
		return 0, fmt.Errorf("counting non-terminal jobs: %w", err)
	}
	return count, nil
}

// Now exposes the store clock so a caller resolves scheduling anchors from the same
// source.
//
// Returns time.Time which is the current instant from the store clock.
func (c *core) Now() time.Time {
	return c.clock.Now()
}

// runInTransaction runs fn inside a single transaction, threading a tx-scoped driver. It
// is the atomicity boundary for the multi-statement writes (SQLite is serialised by its
// single writer; Postgres relies on this tx + FOR UPDATE SKIP LOCKED in the claim query).
//
// Takes fn (func(context.Context, Driver) error) which runs against the tx-scoped driver.
//
// Returns error when the transaction cannot begin, fn fails, or the commit fails.
func (c *core) runInTransaction(ctx context.Context, fn func(ctx context.Context, driver Driver) error) error {
	if c.sqlDB == nil {
		return errDALNotInitialised
	}
	tx, err := c.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(ctx, c.driver.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// Heartbeat renews the visibility lease on a single in-flight job. This prevents jobs from
// being reclaimed by the stalesweep.
//
// Takes jobID (string) which is the in-flight job to renew.
// Takes workerID (string) which must still own the lease for the renewal to work.
//
// Returns error when the context is cancelled or the append version fails.
func (c *core) Heartbeat(ctx context.Context, jobID, workerID string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}

	if err := c.driver.HeartbeatJob(ctx, HeartbeatJobParams{
		ClaimedAt:         new(c.now()),
		ClaimedByWorkerID: &workerID,
		JobID:             jobID,
	}); err != nil {
		return fmt.Errorf("renewing lease on job %q for worker %q: %w", jobID, workerID, err)
	}

	return nil
}

func (c *core) HeartbeatMany(ctx context.Context, jobIDs []string, workerID string) (int, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("heartbeat: %w", err)
	}

	renewedAmount, err := c.driver.HeartbeatJobs(ctx, HeartbeatJobsParams{
		ClaimedAt:         new(c.now()),
		ClaimedByWorkerID: &workerID,
		JobIDs:            jobIDs,
	})

	if err != nil {
		return 0, fmt.Errorf("renewing lease on jobs %v for worker %q: %w", jobIDs, workerID, err)
	}

	return int(renewedAmount), nil
}

// appendFromCurrent reads the job's current projection and appends a new version carrying
// priority/scheduled/attempt forward, letting apply set the event-specific fields.
//
// Takes jobID (string) which is the job whose current projection is carried forward.
// Takes event (string) which names the appended version's event.
// Takes status (string) which is the status the new version records.
// Takes apply (func(*AppendJobVersionParams)) which sets the event-specific fields.
//
// Returns error when the current index cannot be read or the append fails.
func (c *core) appendFromCurrent(ctx context.Context, jobID, event, status string, apply func(*AppendJobVersionParams)) error {
	current, err := c.driver.GetJobIndex(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job %q index: %w", jobID, err)
	}
	params := AppendJobVersionParams{
		JobID:       jobID,
		Event:       event,
		Status:      status,
		Priority:    current.Priority,
		ScheduledAt: current.ScheduledAt,
		Attempt:     current.Attempt,
	}
	apply(&params)
	if err := c.driver.AppendJobVersion(ctx, params); err != nil {
		return fmt.Errorf(errAppendEnqueuedVersion, err)
	}
	return nil
}

// now returns the current instant formatted for the database.
//
// Returns string which is the current instant formatted for the database.
func (c *core) now() string {
	return formatTime(c.clock.Now())
}

// getRowToState maps the joined current-snapshot row to a JobState.
//
// Takes row (JobRow) which is the joined current-snapshot row from the driver.
//
// Returns worker_dto.JobState which is the mapped current state.
// Returns error when a timestamp column fails to parse.
func getRowToState(row JobRow) (worker_dto.JobState, error) {
	scheduledAt, err := parseTime(row.ScheduledAt)
	if err != nil {
		return worker_dto.JobState{}, fmt.Errorf("failed to parse scheduled at: %w", err)
	}
	createdAt, err := parseTime(row.CreatedAt)
	if err != nil {
		return worker_dto.JobState{}, fmt.Errorf("failed to parse created at: %w", err)
	}
	updatedAt, err := parseTime(row.UpdatedAt)
	if err != nil {
		return worker_dto.JobState{}, fmt.Errorf("failed to parse updated at: %w", err)
	}

	return worker_dto.JobState{
		ID:          row.ID,
		Kind:        row.Kind,
		Queue:       row.Queue,
		LastError:   derefText(row.LastError),
		Status:      row.Status,
		Priority:    int64(row.Priority),
		Attempt:     int64(row.Attempt),
		MaxAttempts: int64(row.MaxAttempts),
		ScheduledAt: scheduledAt,
		UpdatedAt:   updatedAt,
		CreatedAt:   createdAt,
	}, nil
}

// claimCandidateToRecord builds a JobRecord from a claim candidate and the running state.
//
// Takes candidate (ClaimCandidateRow) which is the row selected for claiming.
// Takes workerID (string) which is the node that claimed the job.
// Takes attempt (int64) which is the incremented attempt number for this claim.
// Takes claimedAt (string) which is the formatted instant the claim was stamped.
//
// Returns worker_dto.JobRecord which is the running job handed to the caller.
// Returns error when a timestamp column fails to parse.
func claimCandidateToRecord(candidate ClaimCandidateRow, workerID string, attempt int64, claimedAt string) (worker_dto.JobRecord, error) {
	scheduledAt, err := parseTime(candidate.ScheduledAt)
	if err != nil {
		return worker_dto.JobRecord{}, fmt.Errorf("failed to parse scheduled at: %w", err)
	}
	enqueuedAt, err := parseTime(candidate.CreatedAt)
	if err != nil {
		return worker_dto.JobRecord{}, fmt.Errorf("failed to parse created at: %w", err)
	}
	claimed, err := parseTime(claimedAt)
	if err != nil {
		return worker_dto.JobRecord{}, fmt.Errorf("failed to parse claimed at: %w", err)
	}

	return worker_dto.JobRecord{
		ID:                candidate.JobID,
		Status:            string(worker_domain.StatusRunning),
		Kind:              candidate.Kind,
		Queue:             candidate.Queue,
		Payload:           []byte(candidate.Payload),
		ClaimedByWorkerID: workerID,
		Priority:          int64(candidate.Priority),
		Attempt:           attempt,
		MaxAttempts:       int64(candidate.MaxAttempts),
		TimeoutSeconds:    int64(candidate.TimeoutSeconds),
		ClaimedAt:         claimed,
		ScheduledAt:       scheduledAt,
		EnqueueAt:         enqueuedAt,
	}, nil
}

// versionRowToVersion maps a history row to a JobVersion.
//
// Takes row (JobVersionRow) which is one append-only history row from the driver.
//
// Returns worker_dto.JobVersion which is the mapped version.
// Returns error when a timestamp column fails to parse.
func versionRowToVersion(row JobVersionRow) (worker_dto.JobVersion, error) {
	scheduledAt, err := parseTime(row.ScheduledAt)
	if err != nil {
		return worker_dto.JobVersion{}, fmt.Errorf("failed to parse scheduled at: %w", err)
	}
	claimedAt, err := parseTimePtr(row.ClaimedAt)
	if err != nil {
		return worker_dto.JobVersion{}, fmt.Errorf("failed to parse claimed at: %w", err)
	}
	deletedAt, err := parseTimePtr(row.DeletedAt)
	if err != nil {
		return worker_dto.JobVersion{}, fmt.Errorf("failed to parse deleted at: %w", err)
	}
	recordedAt, err := parseTime(row.RecordedAt)
	if err != nil {
		return worker_dto.JobVersion{}, fmt.Errorf("failed to parse recorded at: %w", err)
	}

	return worker_dto.JobVersion{
		ScheduledAt:       scheduledAt,
		ClaimedAt:         claimedAt,
		DeletedAt:         deletedAt,
		RecordedAt:        recordedAt,
		Result:            textBytes(row.Result),
		JobID:             row.JobID,
		Event:             row.Event,
		Status:            row.Status,
		LastError:         derefText(row.LastError),
		ClaimedByWorkerID: derefText(row.ClaimedByWorkerID),
		Sequence:          row.VersionSequence,
		Priority:          int64(row.Priority),
		Attempt:           int64(row.Attempt),
	}, nil
}

// formatTime renders an instant as the UTC RFC3339Nano text written across the DB
// boundary.
//
// Takes t (time.Time) which is the instant to render.
//
// Returns string which is the UTC RFC3339Nano text written to the database.
func formatTime(t time.Time) string {
	return t.UTC().Format(tsLayout)
}

// parseTime reads a timestamp column written by either the core or a DB default.
//
// Takes value (string) which is the timestamp text read from a column.
//
// Returns time.Time which is the parsed UTC instant, or the zero time when empty.
// Returns error when the text matches neither the core nor the DB-default layout.
func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(tsLayout, value); err == nil {
		return parsed.UTC(), nil
	}
	parsed, err := time.Parse(dbDefaultLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

// parseTimePtr reads a nullable timestamp column.
//
// Takes value (*string) which is the nullable timestamp text read from a column.
//
// Returns time.Time which is the parsed instant, or the zero time when null or empty.
// Returns error when the non-empty text fails to parse.
func parseTimePtr(value *string) (time.Time, error) {
	if value == nil || *value == "" {
		return time.Time{}, nil
	}
	return parseTime(*value)
}

// derefText reads a nullable text column.
//
// Takes value (*string) which is the nullable text column to dereference.
//
// Returns string which is the pointed-to text, or "" when the pointer is nil.
func derefText(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// textBytes reads a nullable text column as bytes.
//
// Takes value (*string) which is the nullable text column to read as bytes.
//
// Returns []byte which is the text as bytes, or nil when the pointer is nil.
func textBytes(value *string) []byte {
	if value == nil {
		return nil
	}
	return []byte(*value)
}

// optionalJSON converts a byte array to a string pointer, preserving a nil reference on
// an empty array.
//
// Takes b ([]byte) which is the JSON result to store as a nullable column.
//
// Returns *string which is the text pointer, or nil when b is empty.
func optionalJSON(b []byte) *string {
	if len(b) == 0 {
		return nil
	}
	return new(string(b))
}

// optionalText converts a string to a string pointer, preserving a nil reference on an
// empty string.
//
// Takes s (string) which is the text to store as a nullable column.
//
// Returns *string which is the text pointer, or nil when s is empty.
func optionalText(s string) *string {
	if len(s) == 0 {
		return nil
	}
	return new(s)
}
