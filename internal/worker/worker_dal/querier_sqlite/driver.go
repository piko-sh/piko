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

// Package querier_sqlite is the SQLite driver for the worker DAL.
package querier_sqlite

import (
	"context"
	"database/sql"

	"piko.sh/piko/internal/worker/worker_dal/dalcore"
	"piko.sh/piko/internal/worker/worker_dal/querier_sqlite/db"
	"piko.sh/piko/internal/worker/worker_domain"
	"piko.sh/piko/wdk/clock"
)

// driver is the SQLite implementation of dalcore.Driver. It translates dalcore's
// dialect-neutral flat structs into the generated SQLite query calls.
type driver struct {
	// queries provides access to the generated SQLite query methods.
	queries *db.Queries
}

var (
	_ dalcore.Driver = (*driver)(nil)
)

// New creates a worker Store backed by the given SQLite database connection and clock.
//
// Takes database (*sql.DB) which is the open SQLite connection.
// Takes clk (clock.Clock) which is the time source; nil falls back to RealClock().
//
// Returns worker_domain.Store which is ready for use.
func New(database *sql.DB, clk clock.Clock) worker_domain.Store {
	return dalcore.New(database, &driver{queries: db.New(database)}, clk)
}

func (d *driver) HeartbeatJob(ctx context.Context, params dalcore.HeartbeatJobParams) error {
	return d.queries.HeartbeatJob(ctx, db.HeartbeatJobParams{
		ClaimedByWorkerID: params.ClaimedByWorkerID,
		ClaimedAt:         params.ClaimedAt,
		JobID:             params.JobID,
	})
}

func (d *driver) HeartbeatJobs(ctx context.Context, params dalcore.HeartbeatJobsParams) (int64, error) {
	return d.queries.HeartbeatJobs(ctx, db.HeartbeatJobsParams{
		ClaimedByWorkerID: params.ClaimedByWorkerID,
		ClaimedAt:         params.ClaimedAt,
		IDs:               params.JobIDs,
	})
}

// GetJobIDByUniqueKey looks up an existing job id by its unique idempotency key.
//
// Takes uniqueKey (string) which is the idempotency key to match.
//
// Returns string which is the matching job id, empty when no row holds the key.
// Returns error which is the raw sql.ErrNoRows when no row holds the key, or the
// query error on failure.
func (d *driver) GetJobIDByUniqueKey(ctx context.Context, uniqueKey string) (string, error) {
	row, err := d.queries.GetJobIDByUniqueKey(ctx, &uniqueKey)
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

// InsertJobRootWithUniqueKey inserts a new job root carrying a unique key.
//
// Takes params (dalcore.InsertJobRootParams) which describe the job root and its
// unique key.
//
// Returns string which is the inserted job id.
// Returns error when the insert fails, including when the unique key already
// exists and the conflicting row is skipped.
func (d *driver) InsertJobRootWithUniqueKey(ctx context.Context, params dalcore.InsertJobRootParams) (string, error) {
	row, err := d.queries.InsertJobRootWithUniqueKey(ctx, db.InsertJobRootWithUniqueKeyParams{
		ID:             params.ID,
		Kind:           params.Kind,
		Queue:          params.Queue,
		Payload:        params.Payload,
		UniqueKey:      params.UniqueKey,
		CorrelationID:  params.CorrelationID,
		MaxAttempts:    params.MaxAttempts,
		TimeoutSeconds: params.TimeoutSeconds,
		CreatedAt:      params.CreatedAt,
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

// PromoteCandidates reads the due scheduled and retryable rows to promote.
//
// Takes params (dalcore.PromoteCandidatesParams) which bound the candidate scan.
//
// Returns []dalcore.PromoteCandidateRow which are the due scheduled jobs.
// Returns error when the query fails.
func (d *driver) PromoteCandidates(ctx context.Context, params dalcore.PromoteCandidatesParams) ([]dalcore.PromoteCandidateRow, error) {
	rows, err := d.queries.PromoteCandidates(ctx, db.PromoteCandidatesParams{
		ScheduledAt: params.ScheduledAt,
		Limit:       params.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.PromoteCandidateRow, len(rows))
	for i := range rows {
		out[i] = dalcore.PromoteCandidateRow{
			JobID:       rows[i].JobID,
			ScheduledAt: rows[i].ScheduledAt,
			Attempt:     rows[i].Attempt,
			Priority:    rows[i].Priority,
		}
	}

	return out, nil
}

// WithTx returns a driver whose queries run inside tx.
//
// Takes tx (*sql.Tx) which is the transaction to scope queries to.
//
// Returns dalcore.Driver scoped to the transaction.
func (d *driver) WithTx(tx *sql.Tx) dalcore.Driver {
	return &driver{queries: d.queries.WithTx(tx)}
}

// InsertJobRoot inserts the immutable job root.
//
// Takes params (dalcore.InsertJobRootParams) which describe the job root to insert.
//
// Returns error when the insert fails.
func (d *driver) InsertJobRoot(ctx context.Context, params dalcore.InsertJobRootParams) error {
	return d.queries.InsertJobRoot(ctx, db.InsertJobRootParams{
		ID:             params.ID,
		Kind:           params.Kind,
		Queue:          params.Queue,
		Payload:        params.Payload,
		CorrelationID:  params.CorrelationID,
		MaxAttempts:    params.MaxAttempts,
		TimeoutSeconds: params.TimeoutSeconds,
		CreatedAt:      params.CreatedAt,
	})
}

// InsertJobRootBatch inserts a batch of immutable job roots.
//
// Takes params ([]dalcore.InsertJobRootParams) which describe the job roots to insert.
//
// Returns error when the insert fails.
func (d *driver) InsertJobRootBatch(ctx context.Context, params []dalcore.InsertJobRootParams) error {
	rows := make([]db.InsertJobRootBatchParams, len(params))
	for i := range params {
		rows[i] = db.InsertJobRootBatchParams{
			ID:             params[i].ID,
			Kind:           params[i].Kind,
			Queue:          params[i].Queue,
			Payload:        params[i].Payload,
			MaxAttempts:    params[i].MaxAttempts,
			TimeoutSeconds: params[i].TimeoutSeconds,
			CreatedAt:      params[i].CreatedAt,
		}
	}
	return d.queries.InsertJobRootBatch(ctx, rows)
}

// AppendJobVersion appends an immutable version to a job's append-only log.
//
// Takes params (dalcore.AppendJobVersionParams) which describe the version to append.
//
// Returns error when the append fails.
func (d *driver) AppendJobVersion(ctx context.Context, params dalcore.AppendJobVersionParams) error {
	return d.queries.AppendJobVersion(ctx, db.AppendJobVersionParams{
		JobID:             params.JobID,
		Event:             params.Event,
		Status:            params.Status,
		Priority:          params.Priority,
		ScheduledAt:       params.ScheduledAt,
		Attempt:           params.Attempt,
		LastError:         params.LastError,
		Result:            params.Result,
		ClaimedByWorkerID: params.ClaimedByWorkerID,
		ClaimedAt:         params.ClaimedAt,
		DeletedAt:         params.DeletedAt,
	})
}

// AppendJobVersionBatch appends a batch of immutable versions to append-only logs.
//
// Takes params ([]dalcore.AppendJobVersionParams) which describe the versions to append.
//
// Returns error when the append fails.
func (d *driver) AppendJobVersionBatch(ctx context.Context, params []dalcore.AppendJobVersionParams) error {
	rows := make([]db.AppendJobVersionBatchParams, len(params))
	for i := range params {
		rows[i] = db.AppendJobVersionBatchParams{
			JobID:             params[i].JobID,
			Event:             params[i].Event,
			Status:            params[i].Status,
			Priority:          params[i].Priority,
			ScheduledAt:       params[i].ScheduledAt,
			Attempt:           params[i].Attempt,
			LastError:         params[i].LastError,
			Result:            params[i].Result,
			ClaimedByWorkerID: params[i].ClaimedByWorkerID,
			ClaimedAt:         params[i].ClaimedAt,
			DeletedAt:         params[i].DeletedAt,
		}
	}
	return d.queries.AppendJobVersionBatch(ctx, rows)
}

// ClaimCandidates reads the due pending rows to claim.
//
// Takes params (dalcore.ClaimCandidatesParams) which bound the candidate scan.
//
// Returns []dalcore.ClaimCandidateRow which are the due pending jobs.
// Returns error when the query fails.
func (d *driver) ClaimCandidates(ctx context.Context, params dalcore.ClaimCandidatesParams) ([]dalcore.ClaimCandidateRow, error) {
	rows, err := d.queries.ClaimCandidates(ctx, db.ClaimCandidatesParams{
		ScheduledAt: params.ScheduledAt,
		Limit:       params.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.ClaimCandidateRow, len(rows))
	for i := range rows {
		out[i] = dalcore.ClaimCandidateRow{
			JobID:                  rows[i].JobID,
			CurrentVersionSequence: rows[i].CurrentVersionSequence,
			Priority:               rows[i].Priority,
			ScheduledAt:            rows[i].ScheduledAt,
			Attempt:                rows[i].Attempt,
			Kind:                   rows[i].Kind,
			Queue:                  rows[i].Queue,
			Payload:                rows[i].Payload,
			MaxAttempts:            rows[i].MaxAttempts,
			TimeoutSeconds:         rows[i].TimeoutSeconds,
			CreatedAt:              rows[i].CreatedAt,
		}
	}
	return out, nil
}

// StaleRunningJobs reads running rows whose claimed_at is at or before the cutoff.
//
// Takes cutoff (string) which is the inclusive upper bound on claimed_at.
//
// Returns []dalcore.StaleJobRow which are the stale running jobs.
// Returns error when the query fails.
func (d *driver) StaleRunningJobs(ctx context.Context, cutoff string) ([]dalcore.StaleJobRow, error) {
	rows, err := d.queries.StaleRunningJobs(ctx, &cutoff)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.StaleJobRow, len(rows))
	for i := range rows {
		out[i] = dalcore.StaleJobRow{
			JobID:                  rows[i].JobID,
			CurrentVersionSequence: rows[i].CurrentVersionSequence,
			Priority:               rows[i].Priority,
			ScheduledAt:            rows[i].ScheduledAt,
			Attempt:                rows[i].Attempt,
		}
	}
	return out, nil
}

// GetJob reads the joined current snapshot of a job.
//
// Takes id (string) which is the job identifier.
//
// Returns dalcore.JobRow which is the current snapshot.
// Returns error when the query fails.
func (d *driver) GetJob(ctx context.Context, id string) (dalcore.JobRow, error) {
	row, err := d.queries.GetJob(ctx, id)
	if err != nil {
		return dalcore.JobRow{}, err
	}
	return dalcore.JobRow{
		ID:          row.ID,
		Kind:        row.Kind,
		Queue:       row.Queue,
		MaxAttempts: row.MaxAttempts,
		CreatedAt:   row.CreatedAt,
		Status:      row.Status,
		Priority:    row.Priority,
		ScheduledAt: row.ScheduledAt,
		Attempt:     row.Attempt,
		UpdatedAt:   row.UpdatedAt,
		LastError:   row.LastError,
	}, nil
}

// GetJobIndex reads the current projection columns of a job.
//
// Takes jobID (string) which is the job identifier.
//
// Returns dalcore.JobIndexRow which is the current projection.
// Returns error when the query fails.
func (d *driver) GetJobIndex(ctx context.Context, jobID string) (dalcore.JobIndexRow, error) {
	row, err := d.queries.GetJobIndex(ctx, jobID)
	if err != nil {
		return dalcore.JobIndexRow{}, err
	}
	return dalcore.JobIndexRow{
		Priority:    row.Priority,
		ScheduledAt: row.ScheduledAt,
		Attempt:     row.Attempt,
	}, nil
}

// ListJobVersions reads a job's history oldest-first.
//
// Takes jobID (string) which is the job identifier.
//
// Returns []dalcore.JobVersionRow which is the job's history.
// Returns error when the query fails.
func (d *driver) ListJobVersions(ctx context.Context, jobID string) ([]dalcore.JobVersionRow, error) {
	rows, err := d.queries.ListJobVersions(ctx, jobID)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.JobVersionRow, len(rows))
	for i := range rows {
		out[i] = dalcore.JobVersionRow{
			VersionSequence:   rows[i].VersionSequence,
			JobID:             rows[i].JobID,
			Event:             rows[i].Event,
			Status:            rows[i].Status,
			Priority:          rows[i].Priority,
			ScheduledAt:       rows[i].ScheduledAt,
			Attempt:           rows[i].Attempt,
			LastError:         rows[i].LastError,
			Result:            rows[i].Result,
			ClaimedByWorkerID: rows[i].ClaimedByWorkerID,
			ClaimedAt:         rows[i].ClaimedAt,
			DeletedAt:         rows[i].DeletedAt,
			RecordedAt:        rows[i].RecordedAt,
		}
	}
	return out, nil
}

// CountPendingJobs returns the queue-depth count of pending jobs.
//
// Returns int64 which is the number of pending jobs.
// Returns error when the query fails.
func (d *driver) CountPendingJobs(ctx context.Context) (int64, error) {
	row, err := d.queries.CountPendingJobs(ctx)
	if err != nil {
		return 0, err
	}
	return row.JobCount, nil
}

// CountClaimableJobs returns the queue-depth counts of claimable jobs.
//
// Returns []dalcore.QueueDepthRow which holds a count per queue.
// Returns error when the query fails.
func (d *driver) CountClaimableJobs(ctx context.Context) ([]dalcore.QueueDepthRow, error) {
	rows, err := d.queries.CountClaimableJobs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.QueueDepthRow, len(rows))
	for i := range rows {
		out[i] = dalcore.QueueDepthRow{Queue: rows[i].Queue, Count: rows[i].JobCount}
	}
	return out, nil
}

// CountNonTerminalJobs returns the queue-depth count of non-terminal jobs.
//
// Returns int64 which is the number of non-terminal jobs.
// Returns error when the query fails.
func (d *driver) CountNonTerminalJobs(ctx context.Context) (int64, error) {
	row, err := d.queries.CountNonTerminalJobs(ctx)
	if err != nil {
		return 0, err
	}
	return row.JobCount, nil
}
