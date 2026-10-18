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

package querier_sqlite

import (
	"context"
	"errors"
	"strings"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
)

const (
	// sqlitePrimaryCodeMask selects the primary result code from an extended one.
	sqlitePrimaryCodeMask = 0xff

	// sqliteBusy identifies SQLITE_BUSY errors caused by a conflicting connection lock.
	sqliteBusy = 5

	// sqliteLocked identifies SQLITE_LOCKED errors caused by a conflicting lock within the
	// connection.
	sqliteLocked = 6
)

// ClaimTask moves a delivered attempt to processing in one fenced statement.
//
// Takes params (dalcore.ClaimTaskParams) which describes the claim.
//
// Returns dalcore.TaskRow which is the claimed row.
// Returns bool which is false when the row was not claimable.
// Returns error when the update fails.
func (d *driver) ClaimTask(ctx context.Context, params dalcore.ClaimTaskParams) (dalcore.TaskRow, bool, error) {
	row, ok, err := d.queries.ClaimTask(ctx, orchestrator_db.ClaimTaskParams{
		UpdatedAt:     params.UpdatedAt,
		OwnerInstance: params.Owner,
		ID:            params.ID,
		Attempt:       params.Attempt,
		ExecuteAt:     params.DueBy,
	})
	if err != nil || !ok {
		return dalcore.TaskRow{}, false, err
	}
	return toTaskRow(orchestrator_db.GetTasksByIDRow(row)), true, nil
}

// SettleTask writes a processing task's settled state, fenced by its claimed attempt.
//
// Takes params (dalcore.TaskWriteParams) which carries the new state.
//
// Returns dalcore.TaskRow which is the settled row.
// Returns bool which is false when the task is no longer processing that attempt.
// Returns error when the update fails.
func (d *driver) SettleTask(ctx context.Context, params dalcore.TaskWriteParams) (dalcore.TaskRow, bool, error) {
	row, ok, err := d.queries.SettleTask(ctx, orchestrator_db.SettleTaskParams{
		Status:             params.Status,
		ExecuteAt:          params.ExecuteAt,
		LastError:          params.LastError,
		Result:             params.Result,
		IsFatal:            params.IsFatal,
		ScheduledExecuteAt: params.ScheduledExecuteAt,
		UpdatedAt:          params.UpdatedAt,
		ID:                 params.ID,
		Attempt:            params.Attempt,
	})
	if err != nil || !ok {
		return dalcore.TaskRow{}, false, err
	}
	return toTaskRow(orchestrator_db.GetTasksByIDRow(row)), true, nil
}

// TransitionTask writes a task's new state, fenced by its attempt and status.
//
// Takes params (dalcore.TaskWriteParams) which carries the new state.
// Takes from ([]string) which lists the statuses the stored task may be in.
//
// Returns dalcore.TaskRow which is the moved row.
// Returns bool which is false when the task has moved on or does not exist.
// Returns error when the update fails.
func (d *driver) TransitionTask(ctx context.Context, params dalcore.TaskWriteParams, from []string) (dalcore.TaskRow, bool, error) {
	row, ok, err := d.queries.TransitionTask(ctx, orchestrator_db.TransitionTaskParams{
		Status:             params.Status,
		ExecuteAt:          params.ExecuteAt,
		LastError:          params.LastError,
		Result:             params.Result,
		IsFatal:            params.IsFatal,
		ScheduledExecuteAt: params.ScheduledExecuteAt,
		UpdatedAt:          params.UpdatedAt,
		OwnerInstance:      params.OwnerInstance,
		ID:                 params.ID,
		Attempt:            params.Attempt,
		FromStatuses:       from,
	})
	if err != nil || !ok {
		return dalcore.TaskRow{}, false, err
	}
	return toTaskRow(orchestrator_db.GetTasksByIDRow(row)), true, nil
}

// ClearRerun removes a task's rerun request.
//
// Takes id (string) which identifies the task.
//
// Returns error when the update fails.
func (d *driver) ClearRerun(ctx context.Context, id string) error {
	return d.queries.ClearRerun(ctx, id)
}

// RequestRerun records a rerun on the active task holding a deduplication key.
//
// Takes params (dalcore.RerunParams) which describes the rerun.
//
// Returns int64 which is the number of tasks updated.
// Returns error when the update fails.
func (d *driver) RequestRerun(ctx context.Context, params dalcore.RerunParams) (int64, error) {
	return d.queries.RequestRerun(ctx, orchestrator_db.RequestRerunParams{
		RerunID:          &params.ID,
		RerunPayload:     &params.Payload,
		RerunConfig:      &params.Config,
		DeduplicationKey: &params.DeduplicationKey,
	})
}

// CountUnsettledForBuild counts a build's foreground tasks that have not settled.
//
// Takes buildTag (string) which identifies the build.
//
// Returns int64 which is the count.
// Returns error when the query fails.
func (d *driver) CountUnsettledForBuild(ctx context.Context, buildTag string) (int64, error) {
	row, err := d.queries.CountUnsettledForBuild(ctx, buildTag)
	if err != nil {
		return 0, err
	}
	return row.Count, nil
}

// ListFailedForBuild returns a build's failed tasks.
//
// Takes buildTag (string) which identifies the build.
//
// Returns []dalcore.TaskRow which holds the failed tasks.
// Returns error when the query fails.
func (d *driver) ListFailedForBuild(ctx context.Context, buildTag string) ([]dalcore.TaskRow, error) {
	rows, err := d.queries.ListFailedForBuild(ctx, buildTag)
	if err != nil {
		return nil, err
	}
	return toTaskRows(rows), nil
}

// DispatchablePage reads a page of due pending and retrying tasks.
//
// Takes params (dalcore.DispatchPageParams) which selects the page.
//
// Returns []dalcore.TaskRow which holds the tasks in execution order.
// Returns error when the query fails.
func (d *driver) DispatchablePage(ctx context.Context, params dalcore.DispatchPageParams) ([]dalcore.TaskRow, error) {
	rows, err := d.queries.DispatchablePage(ctx, orchestrator_db.DispatchablePageParams{
		ExecuteAt:       params.Now,
		ExecuteAt2:      params.AfterTime,
		ID:              params.AfterID,
		LastPublishedAt: params.PublishedBefore,
		Limit:           params.Limit,
	})
	if err != nil {
		return nil, err
	}
	return toTaskRows(rows), nil
}

// MarkPublished records when tasks were last published.
//
// Takes publishedAt (int64) which is the publication time in Unix seconds.
// Takes ids ([]string) which identifies the tasks.
//
// Returns error when the update fails.
func (d *driver) MarkPublished(ctx context.Context, publishedAt int64, ids []string) error {
	return d.queries.MarkPublished(ctx, orchestrator_db.MarkPublishedParams{
		LastPublishedAt: publishedAt,
		IDs:             ids,
	})
}

// ListOrphanedTasks returns processing tasks whose owner holds no live lease.
//
// Takes staleBefore (int64) which is the write time, in Unix seconds, orphans predate.
// Takes now (int64) which is the time, in Unix seconds, leases are checked against.
// Takes limit (int) which caps the number of tasks returned.
//
// Returns []dalcore.TaskRow which holds the orphans.
// Returns error when the query fails.
func (d *driver) ListOrphanedTasks(ctx context.Context, staleBefore, now int64, limit int) ([]dalcore.TaskRow, error) {
	rows, err := d.queries.ListOrphanedTasks(ctx, orchestrator_db.ListOrphanedTasksParams{
		UpdatedAt: staleBefore,
		ExpiresAt: now,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	return toTaskRows(rows), nil
}

// PurgeTerminal deletes old completed and failed tasks whose workflows have no pending
// receipt.
//
// Takes before (int64) which is the cutoff in Unix seconds.
// Takes limit (int) which caps the number of tasks deleted.
//
// Returns int64 which is the number of tasks deleted.
// Returns error when the delete fails.
func (d *driver) PurgeTerminal(ctx context.Context, before int64, limit int) (int64, error) {
	return d.queries.PurgeTerminalTasks(ctx, orchestrator_db.PurgeTerminalTasksParams{
		UpdatedAt: before,
		Limit:     limit,
	})
}

// UpsertNode records a node lease, replacing any earlier one for the instance.
//
// Takes params (dalcore.NodeLeaseParams) which describes the lease.
//
// Returns error when the write fails.
func (d *driver) UpsertNode(ctx context.Context, params dalcore.NodeLeaseParams) error {
	return d.queries.UpsertNode(ctx, orchestrator_db.UpsertNodeParams{
		Instance:  params.Instance,
		NodeID:    params.NodeID,
		StartedAt: params.StartedAt,
		RenewedAt: params.RenewedAt,
		ExpiresAt: params.ExpiresAt,
	})
}

// RenewNode extends a node lease.
//
// Takes params (dalcore.NodeLeaseParams) which carries the renewal and expiry times.
//
// Returns int64 which is the number of leases updated.
// Returns error when the update fails.
func (d *driver) RenewNode(ctx context.Context, params dalcore.NodeLeaseParams) (int64, error) {
	return d.queries.RenewNode(ctx, orchestrator_db.RenewNodeParams{
		RenewedAt: params.RenewedAt,
		ExpiresAt: params.ExpiresAt,
		Instance:  params.Instance,
	})
}

// DeleteNode removes a node lease.
//
// Takes instance (string) which identifies the instance.
//
// Returns error when the delete fails.
func (d *driver) DeleteNode(ctx context.Context, instance string) error {
	return d.queries.DeleteNode(ctx, instance)
}

// ListExpiredNodes returns instances whose leases lapsed before now.
//
// Takes now (int64) which is the current time in Unix seconds.
// Takes limit (int) which caps the number of instances returned.
//
// Returns []string which holds the instances in name order.
// Returns error when the query fails.
func (d *driver) ListExpiredNodes(ctx context.Context, now int64, limit int) ([]string, error) {
	rows, err := d.queries.ListExpiredNodes(ctx, orchestrator_db.ListExpiredNodesParams{
		ExpiresAt: now,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	instances := make([]string, len(rows))
	for i, row := range rows {
		instances[i] = row.Instance
	}
	return instances, nil
}

// TouchOwned stamps the tasks an instance is processing as written at a time.
//
// Takes instance (string) which identifies the instance.
// Takes at (int64) which is the time in Unix seconds.
//
// Returns int64 which is the number of tasks touched.
// Returns error when the update fails.
func (d *driver) TouchOwned(ctx context.Context, instance string, at int64) (int64, error) {
	return d.queries.TouchOwnedTasks(ctx, orchestrator_db.TouchOwnedTasksParams{
		UpdatedAt:     at,
		OwnerInstance: instance,
	})
}

// IsTransient reports whether an error is SQLite's busy or locked error, after which the
// whole transaction can be retried.
//
// Takes err (error) which is the error to classify.
//
// Returns bool which is true for SQLITE_BUSY and SQLITE_LOCKED, including their extended
// codes.
func (*driver) IsTransient(err error) bool {
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		switch coded.Code() & sqlitePrimaryCodeMask {
		case sqliteBusy, sqliteLocked:
			return true
		}
		return false
	}
	return err != nil && strings.Contains(err.Error(), "database is locked")
}
