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

package querier_postgres

import (
	"context"
	"database/sql"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/dalcore"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_postgres/db"
)

// driver is the PostgreSQL implementation of dalcore.Driver.
//
// It translates dalcore's dialect-neutral flat structs into the generated PostgreSQL
// query calls. Deduplication uses a single INSERT with ON CONFLICT DO NOTHING, and
// recovery omits the columns Postgres does not rewrite.
type driver struct {
	// queries provides access to the generated PostgreSQL query methods.
	queries *orchestrator_db.Queries
}

var (
	_ dalcore.Driver = (*driver)(nil)
)

// New creates an orchestrator DAL backed by the given PostgreSQL database connection or
// transaction.
//
// Takes db (orchestrator_db.DBTX) which provides the database connection to use for
// queries. When it is a *sql.DB it is also used for transactions and health checks.
//
// Takes opts (...dalcore.Option) which customise the DAL, such as its clock.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is ready for use.
func New(db orchestrator_db.DBTX, opts ...dalcore.Option) orchestrator_dal.OrchestratorDALWithTx {
	var sqlDB *sql.DB
	if typedDB, ok := db.(*sql.DB); ok {
		sqlDB = typedDB
	}
	return dalcore.New(sqlDB, &driver{queries: orchestrator_db.New(db)}, opts...)
}

// NewObserved builds an orchestrator DAL whose statements run through queryDB while
// transactions and health checks use database directly.
//
// Takes database (*sql.DB) which owns BeginTx and PingContext.
// Takes queryDB (orchestrator_db.DBTX) which executes non-transactional statements.
//
// Takes opts (...dalcore.Option) which customise the DAL, such as its clock.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is ready for use.
func NewObserved(database *sql.DB, queryDB orchestrator_db.DBTX, opts ...dalcore.Option) orchestrator_dal.OrchestratorDALWithTx {
	return dalcore.New(database, &driver{queries: orchestrator_db.New(queryDB)}, opts...)
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
		batch[i] = orchestrator_db.CreateTasksBatchParams(toCreateTaskParams(params[i]))
	}
	return d.queries.CreateTasksBatch(ctx, batch)
}

// UpdateTask updates an existing task.
//
// Takes params (dalcore.UpdateTaskParams) which describes the task changes to apply.
//
// Returns int64 which is the number of rows updated.
// Returns error when the update fails.
func (d *driver) UpdateTask(ctx context.Context, params dalcore.UpdateTaskParams) (int64, error) {
	return d.queries.UpdateTask(ctx, orchestrator_db.UpdateTaskParams{
		Status:             params.Status,
		Priority:           params.Priority,
		ExecuteAt:          params.ExecuteAt,
		Attempt:            params.Attempt,
		LastError:          params.LastError,
		Result:             params.Result,
		Payload:            params.Payload,
		Config:             params.Config,
		UpdatedAt:          params.UpdatedAt,
		BuildTag:           params.BuildTag,
		IsFatal:            params.IsFatal,
		ScheduledExecuteAt: params.ScheduledExecuteAt,
		Background:         params.Background,
		OwnerInstance:      params.OwnerInstance,
		ID:                 params.ID,
	})
}

// CreateWithDedup creates a task unless an active task already holds its deduplication
// key.
//
// The insert is a single INSERT with ON CONFLICT DO NOTHING against the unique index of
// active keys.
//
// Takes params (dalcore.CreateTaskParams) which describes the task to insert.
//
// Returns bool which is true when a new task was inserted, or false when an active task
// with the same key already existed.
// Returns error when the insert fails.
func (d *driver) CreateWithDedup(ctx context.Context, params dalcore.CreateTaskParams) (bool, error) {
	rowsAffected, err := d.queries.CreateTaskWithDedup(ctx, orchestrator_db.CreateTaskWithDedupParams(toCreateTaskParams(params)))
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

// GetTasksByID returns the tasks with the given IDs, skipping IDs that do not exist.
//
// Takes ids ([]string) which lists the task IDs to read.
//
// Returns []dalcore.TaskRow which holds the tasks found.
// Returns error when the query fails.
func (d *driver) GetTasksByID(ctx context.Context, ids []string) ([]dalcore.TaskRow, error) {
	rows, err := d.queries.GetTasksByID(ctx, orchestrator_db.GetTasksByIDParams{IDs: ids})
	if err != nil {
		return nil, err
	}
	return toTaskRows(rows), nil
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
		UpdatedAt:    &nowUnix,
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
		out[i] = dalcore.PendingReceiptRow{ID: row.ID, WorkflowID: row.WorkflowID, CreatedAt: row.CreatedAt, NodeID: ""}
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
// Takes olderThanUnix (int64) which is the Unix timestamp before which resolved receipts
// are deleted.
//
// Returns int64 which is the number of receipts deleted.
// Returns error when the delete fails.
func (d *driver) CleanupOldResolvedReceipts(ctx context.Context, olderThanUnix int64) (int64, error) {
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
// Returns []dalcore.TaskRow which holds the failed tasks.
// Returns error when the query fails.
func (d *driver) ListFailedTasks(ctx context.Context) ([]dalcore.TaskRow, error) {
	rows, err := d.queries.ListFailedTasks(ctx)
	if err != nil {
		return nil, err
	}
	return toTaskRows(rows), nil
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

// toCreateTaskParams maps a dalcore.CreateTaskParams to the generated PostgreSQL params.
//
// Takes params (dalcore.CreateTaskParams) which is the dialect-neutral create-task input.
//
// Returns orchestrator_db.CreateTaskParams ready for the generated CreateTask query.
func toCreateTaskParams(params dalcore.CreateTaskParams) orchestrator_db.CreateTaskParams {
	return orchestrator_db.CreateTaskParams{
		ID:                 params.ID,
		WorkflowID:         params.WorkflowID,
		Executor:           params.Executor,
		Priority:           params.Priority,
		Payload:            params.Payload,
		Config:             params.Config,
		Status:             params.Status,
		ExecuteAt:          params.ExecuteAt,
		Attempt:            params.Attempt,
		CreatedAt:          params.CreatedAt,
		UpdatedAt:          params.UpdatedAt,
		DeduplicationKey:   params.DeduplicationKey,
		BuildTag:           params.BuildTag,
		IsFatal:            params.IsFatal,
		ScheduledExecuteAt: params.ScheduledExecuteAt,
		Background:         params.Background,
		OwnerInstance:      params.OwnerInstance,
		LastError:          params.LastError,
		Result:             params.Result,
	}
}

// wholeTaskRow is satisfied by every generated row type that selects a whole task.
type wholeTaskRow interface {
	orchestrator_db.GetTasksByIDRow | orchestrator_db.ListFailedTasksRow |
		orchestrator_db.ListFailedForBuildRow | orchestrator_db.DispatchablePageRow |
		orchestrator_db.ListOrphanedTasksRow | orchestrator_db.ClaimTaskRow |
		orchestrator_db.SettleTaskRow | orchestrator_db.TransitionTaskRow
}

// toTaskRows converts generated whole-task rows into the driver-neutral row shape.
//
// Takes rows ([]R) which are the generated rows.
//
// Returns []dalcore.TaskRow which carries the same columns.
func toTaskRows[R wholeTaskRow](rows []R) []dalcore.TaskRow {
	out := make([]dalcore.TaskRow, len(rows))
	for i := range rows {
		out[i] = toTaskRow(orchestrator_db.GetTasksByIDRow(rows[i]))
	}
	return out
}

// toTaskRow converts a generated task row into the driver-neutral row shape.
//
// Takes row (orchestrator_db.GetTasksByIDRow) which is the generated row.
//
// Returns dalcore.TaskRow which carries the same columns.
func toTaskRow(row orchestrator_db.GetTasksByIDRow) dalcore.TaskRow {
	return dalcore.TaskRow{
		Result:             row.Result,
		DeduplicationKey:   row.DeduplicationKey,
		LastError:          row.LastError,
		RerunID:            row.RerunID,
		RerunPayload:       row.RerunPayload,
		RerunConfig:        row.RerunConfig,
		Payload:            row.Payload,
		Config:             row.Config,
		Status:             row.Status,
		ID:                 row.ID,
		Executor:           row.Executor,
		WorkflowID:         row.WorkflowID,
		BuildTag:           row.BuildTag,
		OwnerInstance:      row.OwnerInstance,
		ExecuteAt:          row.ExecuteAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
		ScheduledExecuteAt: row.ScheduledExecuteAt,
		LastPublishedAt:    row.LastPublishedAt,
		Priority:           row.Priority,
		Attempt:            row.Attempt,
		IsFatal:            row.IsFatal,
		Background:         row.Background,
	}
}

// WorkflowError reports a persisted workflow failure for receipt reconciliation.
//
// Takes workflowID (string) which identifies the workflow to inspect.
//
// Returns outcome (error) which is a persisted task failure, or nil when none is found.
// Returns readErr (error) which reports a query failure.
func (d *driver) WorkflowError(ctx context.Context, workflowID string) (outcome error, readErr error) {
	return d.queries.WorkflowError(ctx, workflowID)
}
