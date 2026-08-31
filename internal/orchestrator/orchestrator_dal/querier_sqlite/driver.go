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
	"database/sql"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
)

// driver is the SQLite implementation of dalcore.Driver.
//
// It translates dalcore's dialect-neutral flat structs into the generated SQLite query
// calls. Deduplication uses a check-then-insert pair because SQLite does not generate a
// single dedup insert.
type driver struct {
	// queries provides access to the generated SQLite query methods.
	queries *orchestrator_db.Queries
}

var (
	_ dalcore.Driver = (*driver)(nil)
)

// New creates an orchestrator DAL backed by the given SQLite database connection or
// transaction.
//
// Takes db (orchestrator_db.DBTX) which provides the database connection to use for
// queries. When it is a *sql.DB it is also used for transactions and health checks.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is ready for use.
func New(db orchestrator_db.DBTX) orchestrator_dal.OrchestratorDALWithTx {
	var sqlDB *sql.DB
	if typedDB, ok := db.(*sql.DB); ok {
		sqlDB = typedDB
	}
	return dalcore.New(sqlDB, &driver{queries: orchestrator_db.New(db)})
}

// NewObserved builds an orchestrator DAL whose statements run through queryDB while
// transactions and health checks use database directly.
//
// Takes database (*sql.DB) which owns BeginTx and PingContext.
// Takes queryDB (orchestrator_db.DBTX) which executes non-transactional statements.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is ready for use.
func NewObserved(database *sql.DB, queryDB orchestrator_db.DBTX) orchestrator_dal.OrchestratorDALWithTx {
	return dalcore.New(database, &driver{queries: orchestrator_db.New(queryDB)})
}

// WithTx returns a driver whose queries run inside tx.
//
// Takes tx (*sql.Tx) which is the transaction to scope queries to.
//
// Returns dalcore.Driver scoped to the transaction.
func (d *driver) WithTx(tx *sql.Tx) dalcore.Driver {
	return &driver{queries: d.queries.WithTx(tx)}
}

// CreateTask inserts a single task.
//
// Takes params (dalcore.CreateTaskParams) which describes the task to insert.
//
// Returns error when the insert fails.
func (d *driver) CreateTask(ctx context.Context, params dalcore.CreateTaskParams) error {
	return d.queries.CreateTask(ctx, toCreateTaskParams(params))
}

// CreateTasksBatch inserts a batch of tasks with automatic chunking.
//
// Takes params ([]dalcore.CreateTaskBatchParams) which describes the tasks to insert.
//
// Returns error when the batch insert fails.
func (d *driver) CreateTasksBatch(ctx context.Context, params []dalcore.CreateTaskBatchParams) error {
	batch := make([]orchestrator_db.CreateTasksBatchParams, len(params))
	for i := range params {
		param := &params[i]
		batch[i] = orchestrator_db.CreateTasksBatchParams{
			ID:               param.ID,
			WorkflowID:       param.WorkflowID,
			Executor:         param.Executor,
			Priority:         param.Priority,
			Payload:          param.Payload,
			Config:           param.Config,
			Status:           param.Status,
			ExecuteAt:        param.ExecuteAt,
			Attempt:          param.Attempt,
			CreatedAt:        param.CreatedAt,
			UpdatedAt:        param.UpdatedAt,
			DeduplicationKey: param.DeduplicationKey,
		}
	}
	return d.queries.CreateTasksBatch(ctx, batch)
}

// UpdateTask updates an existing task.
//
// Takes params (dalcore.UpdateTaskParams) which describes the task changes to apply.
//
// Returns error when the update fails.
func (d *driver) UpdateTask(ctx context.Context, params dalcore.UpdateTaskParams) error {
	return d.queries.UpdateTask(ctx, orchestrator_db.UpdateTaskParams{
		Status:    params.Status,
		Priority:  params.Priority,
		ExecuteAt: params.ExecuteAt,
		Attempt:   params.Attempt,
		LastError: params.LastError,
		Result:    params.Result,
		Payload:   params.Payload,
		Config:    params.Config,
		UpdatedAt: params.UpdatedAt,
		ID:        params.ID,
	})
}

// CreateWithDedup creates a task honouring its deduplication key.
//
// The insert uses a check-then-insert pair inside the caller's transaction and persists
// the deduplication key so a later check can detect the row as an active duplicate.
//
// Takes params (dalcore.CreateTaskParams) which describes the task to insert.
// Takes deduplicationKey (string) which is the key that guards against duplicates.
//
// Returns bool which is true when a new task was inserted, or false when an active task
// with the same key already existed.
// Returns error when the check or insert fails.
func (d *driver) CreateWithDedup(ctx context.Context, params dalcore.CreateTaskParams, deduplicationKey string) (bool, error) {
	result, err := d.queries.CheckDuplicateActiveTask(ctx, &deduplicationKey)
	if err != nil {
		return false, err
	}
	if result.HasDuplicate {
		return false, nil
	}
	if err := d.queries.CreateTaskWithDedup(ctx, orchestrator_db.CreateTaskWithDedupParams{
		ID:               params.ID,
		WorkflowID:       params.WorkflowID,
		Executor:         params.Executor,
		Priority:         params.Priority,
		Payload:          params.Payload,
		Config:           params.Config,
		Status:           params.Status,
		ExecuteAt:        params.ExecuteAt,
		Attempt:          params.Attempt,
		CreatedAt:        params.CreatedAt,
		UpdatedAt:        params.UpdatedAt,
		DeduplicationKey: &deduplicationKey,
	}); err != nil {
		return false, err
	}
	return true, nil
}

// FetchDueTasks returns tasks that are due to run.
//
// Takes params (dalcore.FetchDueTasksParams) which describes the status, priority, and
// time filters to apply.
//
// Returns []dalcore.FetchDueTaskRow which holds the tasks that are due.
// Returns error when the query fails.
func (d *driver) FetchDueTasks(ctx context.Context, params dalcore.FetchDueTasksParams) ([]dalcore.FetchDueTaskRow, error) {
	rows, err := d.queries.FetchDueTasks(ctx, orchestrator_db.FetchDueTasksParams{
		Statuses:  params.Statuses,
		Priority:  params.Priority,
		ExecuteAt: params.ExecuteAt,
		Limit:     params.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.FetchDueTaskRow, len(rows))
	for i := range rows {
		out[i] = dalcore.FetchDueTaskRow{
			ID:               rows[i].ID,
			WorkflowID:       rows[i].WorkflowID,
			Executor:         rows[i].Executor,
			Priority:         rows[i].Priority,
			Payload:          rows[i].Payload,
			Config:           rows[i].Config,
			Result:           rows[i].Result,
			Status:           rows[i].Status,
			ExecuteAt:        rows[i].ExecuteAt,
			Attempt:          rows[i].Attempt,
			LastError:        rows[i].LastError,
			CreatedAt:        rows[i].CreatedAt,
			UpdatedAt:        rows[i].UpdatedAt,
			DeduplicationKey: rows[i].DeduplicationKey,
		}
	}
	return out, nil
}

// MarkTasksAsProcessing marks the given task IDs as processing.
//
// Takes updatedAt (int64) which is the Unix timestamp to record on the affected tasks.
// Takes ids ([]string) which are the task IDs to mark as processing.
//
// Returns error when the update fails.
func (d *driver) MarkTasksAsProcessing(ctx context.Context, updatedAt int64, ids []string) error {
	return d.queries.MarkTasksAsProcessing(ctx, orchestrator_db.MarkTasksAsProcessingParams{
		UpdatedAt: updatedAt,
		IDs:       ids,
	})
}

// GetWorkflowStatus reports whether the workflow still has incomplete tasks.
//
// Takes workflowID (string) which identifies the workflow to inspect.
//
// Returns bool which is true when the workflow still has incomplete tasks.
// Returns error when the query fails.
func (d *driver) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	row, err := d.queries.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		return false, err
	}
	return row.HasIncomplete, nil
}

// PromoteScheduledTasks promotes ready scheduled tasks to pending and returns the count.
//
// Takes updatedAt (int64) which is the Unix timestamp to record on promoted tasks.
// Takes executeAt (int64) which is the cutoff before which scheduled tasks are promoted.
//
// Returns int64 which is the number of tasks promoted.
// Returns error when the update fails.
func (d *driver) PromoteScheduledTasks(ctx context.Context, updatedAt, executeAt int64) (int64, error) {
	return d.queries.PromoteScheduledTasks(ctx, orchestrator_db.PromoteScheduledTasksParams{
		UpdatedAt: updatedAt,
		ExecuteAt: executeAt,
	})
}

// PendingTaskCount returns the number of pending tasks.
//
// Returns int64 which is the number of pending tasks.
// Returns error when the query fails.
func (d *driver) PendingTaskCount(ctx context.Context) (int64, error) {
	row, err := d.queries.PendingTaskCount(ctx)
	if err != nil {
		return 0, err
	}
	return row.Count, nil
}

// RecoverStale resets stale processing tasks to retrying or failed and returns the count.
//
// Takes maxRetries (int32) which is the attempt ceiling past which a task fails instead
// of retrying.
// Takes recoveryError (*string) which is the error message to record on recovered tasks.
// Takes nowUnix (int64) which is the current Unix timestamp.
// Takes staleThresholdUnix (int64) which is the Unix timestamp before which a processing
// task is considered stale.
//
// Returns int64 which is the number of tasks reset.
// Returns error when the update fails.
func (d *driver) RecoverStale(ctx context.Context, maxRetries int32, recoveryError *string, nowUnix, staleThresholdUnix int64) (int64, error) {
	return d.queries.RecoverStaleTasks(ctx, orchestrator_db.RecoverStaleTasksParams{
		Attempt:    maxRetries,
		Attempt2:   maxRetries,
		LastError:  recoveryError,
		UpdatedAt:  nowUnix,
		ExecuteAt:  nowUnix,
		UpdatedAt2: staleThresholdUnix,
	})
}

// GetStaleProcessingTaskCount returns the number of tasks stuck in processing past the
// threshold.
//
// Takes staleThresholdUnix (int64) which is the Unix timestamp before which a processing
// task is considered stale.
//
// Returns int64 which is the number of stale processing tasks.
// Returns error when the query fails.
func (d *driver) GetStaleProcessingTaskCount(ctx context.Context, staleThresholdUnix int64) (int64, error) {
	row, err := d.queries.GetStaleProcessingTaskCount(ctx, staleThresholdUnix)
	if err != nil {
		return 0, err
	}
	return row.Count, nil
}

// UpdateTaskHeartbeat updates the heartbeat timestamp for a processing task.
//
// Takes updatedAt (int64) which is the Unix timestamp to record as the latest heartbeat.
// Takes taskID (string) which identifies the task to update.
//
// Returns error when the update fails.
func (d *driver) UpdateTaskHeartbeat(ctx context.Context, updatedAt int64, taskID string) error {
	return d.queries.UpdateTaskHeartbeat(ctx, orchestrator_db.UpdateTaskHeartbeatParams{
		UpdatedAt: updatedAt,
		ID:        taskID,
	})
}

// GetStaleTasksForRecovery returns candidate stale tasks for recovery claiming.
//
// Takes staleThresholdUnix (int64) which is the Unix timestamp before which a processing
// task is considered stale.
// Takes recoveryExpiresAt (*int64) which is the Unix timestamp before which an existing
// recovery lease counts as expired.
// Takes limit (int) which caps the number of candidate tasks returned.
//
// Returns []dalcore.StaleTaskRow which holds the candidate stale tasks.
// Returns error when the query fails.
func (d *driver) GetStaleTasksForRecovery(ctx context.Context, staleThresholdUnix int64, recoveryExpiresAt *int64, limit int) ([]dalcore.StaleTaskRow, error) {
	rows, err := d.queries.GetStaleTasksForRecovery(ctx, orchestrator_db.GetStaleTasksForRecoveryParams{
		UpdatedAt:         staleThresholdUnix,
		RecoveryExpiresAt: recoveryExpiresAt,
		Limit:             limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.StaleTaskRow, len(rows))
	for i, row := range rows {
		out[i] = dalcore.StaleTaskRow{ID: row.ID, WorkflowID: row.WorkflowID, Attempt: row.Attempt}
	}
	return out, nil
}

// ClaimTaskForRecovery attempts to claim a single stale task and returns rows affected.
//
// Takes recoveryNodeID (*string) which identifies the node claiming the task.
// Takes leaseExpiresAt (*int64) which is the Unix timestamp at which the claim lease
// expires.
// Takes taskID (string) which identifies the task to claim.
// Takes nowUnix (*int64) which is the current Unix timestamp used to test the existing
// lease.
//
// Returns int64 which is the number of rows claimed, either zero or one.
// Returns error when the update fails.
func (d *driver) ClaimTaskForRecovery(ctx context.Context, recoveryNodeID *string, leaseExpiresAt *int64, taskID string, nowUnix *int64) (int64, error) {
	return d.queries.ClaimTaskForRecovery(ctx, orchestrator_db.ClaimTaskForRecoveryParams{
		RecoveryNodeID:     recoveryNodeID,
		RecoveryExpiresAt:  leaseExpiresAt,
		ID:                 taskID,
		RecoveryExpiresAt2: nowUnix,
	})
}

// RecoverClaimed recovers all tasks previously claimed by a node and returns the count.
//
// Takes maxRetries (int32) which is the attempt ceiling past which a task fails instead
// of retrying.
// Takes recoveryError (*string) which is the error message to record on recovered tasks.
// Takes nowUnix (int64) which is the current Unix timestamp.
// Takes nodeID (*string) which identifies the node whose claimed tasks are recovered.
//
// Returns int64 which is the number of tasks recovered.
// Returns error when the update fails.
func (d *driver) RecoverClaimed(ctx context.Context, maxRetries int32, recoveryError *string, nowUnix int64, nodeID *string) (int64, error) {
	return d.queries.RecoverClaimedTasks(ctx, orchestrator_db.RecoverClaimedTasksParams{
		Attempt:        maxRetries,
		Attempt2:       maxRetries,
		LastError:      recoveryError,
		UpdatedAt:      nowUnix,
		ExecuteAt:      nowUnix,
		RecoveryNodeID: nodeID,
	})
}

// ReleaseRecoveryLeases releases all recovery leases held by a node and returns the
// count.
//
// Takes nodeID (*string) which identifies the node whose leases are released.
//
// Returns int64 which is the number of leases released.
// Returns error when the update fails.
func (d *driver) ReleaseRecoveryLeases(ctx context.Context, nodeID *string) (int64, error) {
	return d.queries.ReleaseRecoveryLeases(ctx, nodeID)
}

// CreateWorkflowReceipt creates a workflow-completion receipt.
//
// Takes params (dalcore.CreateWorkflowReceiptParams) which describes the receipt to
// create.
//
// Returns error when the insert fails.
func (d *driver) CreateWorkflowReceipt(ctx context.Context, params dalcore.CreateWorkflowReceiptParams) error {
	return d.queries.CreateWorkflowReceipt(ctx, orchestrator_db.CreateWorkflowReceiptParams{
		ID:         params.ID,
		WorkflowID: params.WorkflowID,
		NodeID:     params.NodeID,
		CreatedAt:  params.CreatedAt,
		UpdatedAt:  params.UpdatedAt,
	})
}

// ResolveWorkflowReceipts marks a workflow's pending receipts as resolved and returns the
// count.
//
// Takes workflowID (string) which identifies the workflow whose receipts are resolved.
// Takes errorMessage (*string) which is the failure message to record, or nil on success.
// Takes nowUnix (int64) which is the current Unix timestamp to record as resolution time.
//
// Returns int64 which is the number of receipts resolved.
// Returns error when the update fails.
func (d *driver) ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage *string, nowUnix int64) (int64, error) {
	return d.queries.ResolveWorkflowReceipts(ctx, orchestrator_db.ResolveWorkflowReceiptsParams{
		ErrorMessage: errorMessage,
		UpdatedAt:    nowUnix,
		ResolvedAt:   &nowUnix,
		WorkflowID:   workflowID,
	})
}

// GetPendingReceiptsByNode returns the pending receipts created by a node.
//
// Takes nodeID (string) which identifies the node whose pending receipts are returned.
//
// Returns []dalcore.PendingReceiptRow which holds the node's pending receipts.
// Returns error when the query fails.
func (d *driver) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]dalcore.PendingReceiptRow, error) {
	rows, err := d.queries.GetPendingReceiptsByNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.PendingReceiptRow, len(rows))
	for i, row := range rows {
		out[i] = dalcore.PendingReceiptRow{ID: row.ID, WorkflowID: row.WorkflowID, CreatedAt: row.CreatedAt}
	}
	return out, nil
}

// GetPendingReceiptsByWorkflow returns the pending receipts for a workflow.
//
// Takes workflowID (string) which identifies the workflow whose pending receipts are
// returned.
//
// Returns []dalcore.PendingReceiptRow which holds the workflow's pending receipts.
// Returns error when the query fails.
func (d *driver) GetPendingReceiptsByWorkflow(ctx context.Context, workflowID string) ([]dalcore.PendingReceiptRow, error) {
	rows, err := d.queries.GetPendingReceiptsByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.PendingReceiptRow, len(rows))
	for i, row := range rows {
		out[i] = dalcore.PendingReceiptRow{ID: row.ID, WorkflowID: row.WorkflowID, NodeID: row.NodeID, CreatedAt: row.CreatedAt}
	}
	return out, nil
}

// CleanupOldResolvedReceipts deletes resolved receipts older than the cutoff and returns
// the count.
//
// Takes olderThanUnix (*int64) which is the Unix timestamp before which resolved receipts
// are deleted.
//
// Returns int64 which is the number of receipts deleted.
// Returns error when the delete fails.
func (d *driver) CleanupOldResolvedReceipts(ctx context.Context, olderThanUnix *int64) (int64, error) {
	return d.queries.CleanupOldResolvedReceipts(ctx, olderThanUnix)
}

// TimeoutStaleReceipts marks very old pending receipts as timed out and returns the
// count.
//
// Takes updatedAt (int64) which is the Unix timestamp to record on the affected receipts.
// Takes olderThanUnix (int64) which is the creation-time cutoff before which pending
// receipts time out.
//
// Returns int64 which is the number of receipts timed out.
// Returns error when the update fails.
func (d *driver) TimeoutStaleReceipts(ctx context.Context, updatedAt, olderThanUnix int64) (int64, error) {
	return d.queries.TimeoutStaleReceipts(ctx, orchestrator_db.TimeoutStaleReceiptsParams{
		UpdatedAt: updatedAt,
		CreatedAt: olderThanUnix,
	})
}

// ListFailedTasks returns all tasks in the failed state.
//
// Returns []dalcore.FailedTaskRow which holds the failed tasks.
// Returns error when the query fails.
func (d *driver) ListFailedTasks(ctx context.Context) ([]dalcore.FailedTaskRow, error) {
	rows, err := d.queries.ListFailedTasks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.FailedTaskRow, len(rows))
	for i := range rows {
		out[i] = dalcore.FailedTaskRow{
			ID:         rows[i].ID,
			WorkflowID: rows[i].WorkflowID,
			Executor:   rows[i].Executor,
			Attempt:    rows[i].Attempt,
			LastError:  rows[i].LastError,
		}
	}
	return out, nil
}

// ListTaskStatusCounts returns task counts grouped by status.
//
// Returns []dalcore.TaskStatusCountRow which holds the per-status task counts.
// Returns error when the query fails.
func (d *driver) ListTaskStatusCounts(ctx context.Context) ([]dalcore.TaskStatusCountRow, error) {
	rows, err := d.queries.ListTaskStatusCounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.TaskStatusCountRow, len(rows))
	for i, row := range rows {
		out[i] = dalcore.TaskStatusCountRow{Status: row.Status, TaskCount: row.TaskCount}
	}
	return out, nil
}

// ListRecentTasks returns the most recently updated tasks.
//
// Takes limit (int) which caps the number of tasks returned.
//
// Returns []dalcore.RecentTaskRow which holds the most recently updated tasks.
// Returns error when the query fails.
func (d *driver) ListRecentTasks(ctx context.Context, limit int) ([]dalcore.RecentTaskRow, error) {
	rows, err := d.queries.ListRecentTasks(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.RecentTaskRow, len(rows))
	for i := range rows {
		out[i] = dalcore.RecentTaskRow{
			ID:         rows[i].ID,
			WorkflowID: rows[i].WorkflowID,
			Executor:   rows[i].Executor,
			Status:     rows[i].Status,
			Priority:   rows[i].Priority,
			Attempt:    rows[i].Attempt,
			LastError:  rows[i].LastError,
			CreatedAt:  rows[i].CreatedAt,
			UpdatedAt:  rows[i].UpdatedAt,
		}
	}
	return out, nil
}

// ListWorkflowSummary returns per-workflow aggregates.
//
// Takes limit (int) which caps the number of workflows returned.
//
// Returns []dalcore.WorkflowSummaryRow which holds the per-workflow aggregates.
// Returns error when the query fails.
func (d *driver) ListWorkflowSummary(ctx context.Context, limit int) ([]dalcore.WorkflowSummaryRow, error) {
	rows, err := d.queries.ListWorkflowSummary(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dalcore.WorkflowSummaryRow, len(rows))
	for i := range rows {
		out[i] = dalcore.WorkflowSummaryRow{
			WorkflowID:    rows[i].WorkflowID,
			TaskCount:     rows[i].TaskCount,
			CompleteCount: rows[i].CompleteCount,
			FailedCount:   rows[i].FailedCount,
			ActiveCount:   rows[i].ActiveCount,
			CreatedAt:     rows[i].CreatedAt,
			UpdatedAt:     rows[i].UpdatedAt,
		}
	}
	return out, nil
}

// toCreateTaskParams maps a dalcore.CreateTaskParams to the generated SQLite params.
//
// Takes params (dalcore.CreateTaskParams) which is the dialect-neutral create-task input.
//
// Returns orchestrator_db.CreateTaskParams ready for the generated CreateTask query.
func toCreateTaskParams(params dalcore.CreateTaskParams) orchestrator_db.CreateTaskParams {
	return orchestrator_db.CreateTaskParams{
		ID:         params.ID,
		WorkflowID: params.WorkflowID,
		Executor:   params.Executor,
		Priority:   params.Priority,
		Payload:    params.Payload,
		Config:     params.Config,
		Status:     params.Status,
		ExecuteAt:  params.ExecuteAt,
		Attempt:    params.Attempt,
		CreatedAt:  params.CreatedAt,
		UpdatedAt:  params.UpdatedAt,
	}
}
