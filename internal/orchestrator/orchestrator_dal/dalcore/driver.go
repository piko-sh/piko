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
// Its implementations live in each querier_<dialect> package and translate between
// dalcore's dialect-neutral flat structs and the generated database types. The
// deduplication, recovery, and receipt-resolution methods are semantic: each dialect maps
// the given arguments onto its own generated SQL, hiding the per-dialect parameter
// shapes.
type Driver interface {
	// WithTx returns a driver whose queries run inside tx.
	//
	// Takes tx (*sql.Tx) which is the open transaction the queries join.
	//
	// Returns Driver which runs every query inside that transaction.
	WithTx(tx *sql.Tx) Driver

	// CreateTask inserts a single task.
	//
	// Takes params (CreateTaskParams) which carries the fields of the task to insert.
	//
	// Returns error when the insert fails.
	CreateTask(ctx context.Context, params CreateTaskParams) error

	// CreateTasksBatch inserts a batch of tasks with automatic chunking.
	//
	// Takes params ([]CreateTaskBatchParams) which carries the fields of each task.
	//
	// Returns error when the insert fails.
	CreateTasksBatch(ctx context.Context, params []CreateTaskBatchParams) error

	// UpdateTask updates an existing task.
	//
	// Takes params (UpdateTaskParams) which carries the new field values for the task.
	//
	// Returns error when the update fails.
	UpdateTask(ctx context.Context, params UpdateTaskParams) error

	// CreateWithDedup creates a task honouring its deduplication key.
	//
	// It returns created == false when an active task with the same key already exists.
	//
	// Takes params (CreateTaskParams) which carries the fields of the task to insert.
	// Takes deduplicationKey (string) which identifies duplicate work.
	//
	// Returns bool which is true when the task was created.
	// Returns error when the insert fails.
	CreateWithDedup(ctx context.Context, params CreateTaskParams, deduplicationKey string) (created bool, err error)

	// FetchDueTasks returns tasks that are due to run.
	//
	// Takes params (FetchDueTasksParams) which carries the filter for the fetch.
	//
	// Returns []FetchDueTaskRow which holds the tasks that are due.
	// Returns error when the query fails.
	FetchDueTasks(ctx context.Context, params FetchDueTasksParams) ([]FetchDueTaskRow, error)

	// MarkTasksAsProcessing marks the given task IDs as processing.
	//
	// Takes updatedAt (int64) which is the update time in Unix seconds.
	// Takes ids ([]string) which are the tasks to mark.
	//
	// Returns error when the update fails.
	MarkTasksAsProcessing(ctx context.Context, updatedAt int64, ids []string) error

	// GetWorkflowStatus reports whether the workflow still has incomplete tasks.
	//
	// Takes workflowID (string) which identifies the workflow.
	//
	// Returns bool which is true when the workflow still has incomplete tasks.
	// Returns error when the query fails.
	GetWorkflowStatus(ctx context.Context, workflowID string) (hasIncomplete bool, err error)

	// PromoteScheduledTasks promotes ready scheduled tasks to pending and returns the count.
	//
	// Takes updatedAt (int64) which is the update time in Unix seconds.
	// Takes executeAt (int64) which is the upper bound of scheduled execution time in Unix
	// seconds.
	//
	// Returns int64 which is the number of tasks promoted.
	// Returns error when the update fails.
	PromoteScheduledTasks(ctx context.Context, updatedAt, executeAt int64) (int64, error)

	// PendingTaskCount returns the number of pending tasks.
	//
	// Returns int64 which is the number of pending tasks.
	// Returns error when the query fails.
	PendingTaskCount(ctx context.Context) (int64, error)

	// RecoverStale resets stale processing tasks to retrying or failed and returns the
	// count.
	//
	// Takes maxRetries (int32) which is the attempt limit before a task is failed.
	// Takes recoveryError (*string) which is the error message recorded on the task, if any.
	// Takes nowUnix (int64) which is the current time in Unix seconds.
	// Takes staleThresholdUnix (int64) which is the time before which a task counts as
	// stale.
	//
	// Returns int64 which is the number of tasks recovered.
	// Returns error when the update fails.
	RecoverStale(ctx context.Context, maxRetries int32, recoveryError *string, nowUnix, staleThresholdUnix int64) (int64, error)

	// GetStaleProcessingTaskCount returns the number of tasks stuck in processing past the
	// threshold.
	//
	// Takes staleThresholdUnix (int64) which is the time before which a task counts as
	// stale.
	//
	// Returns int64 which is the number of stale processing tasks.
	// Returns error when the query fails.
	GetStaleProcessingTaskCount(ctx context.Context, staleThresholdUnix int64) (int64, error)

	// UpdateTaskHeartbeat updates the heartbeat timestamp for a processing task.
	//
	// Takes updatedAt (int64) which is the heartbeat time in Unix seconds.
	// Takes taskID (string) which identifies the task.
	//
	// Returns error when the update fails.
	UpdateTaskHeartbeat(ctx context.Context, updatedAt int64, taskID string) error

	// GetStaleTasksForRecovery returns candidate stale tasks for recovery claiming.
	//
	// Takes staleThresholdUnix (int64) which is the time before which a task counts as
	// stale.
	// Takes recoveryExpiresAt (*int64) which is the time an existing recovery lease expires,
	// if any.
	// Takes limit (int) which caps the number of candidates returned.
	//
	// Returns []StaleTaskRow which holds the candidate tasks.
	// Returns error when the query fails.
	GetStaleTasksForRecovery(ctx context.Context, staleThresholdUnix int64, recoveryExpiresAt *int64, limit int) ([]StaleTaskRow, error)

	// ClaimTaskForRecovery attempts to claim a single stale task and returns rows affected.
	//
	// Takes recoveryNodeID (*string) which identifies the node making the claim, if any.
	// Takes leaseExpiresAt (*int64) which is the lease expiry in Unix seconds, if any.
	// Takes taskID (string) which identifies the task to claim.
	// Takes nowUnix (*int64) which is the current time in Unix seconds, if any.
	//
	// Returns int64 which is the number of rows the claim affected.
	// Returns error when the update fails.
	ClaimTaskForRecovery(ctx context.Context, recoveryNodeID *string, leaseExpiresAt *int64, taskID string, nowUnix *int64) (int64, error)

	// RecoverClaimed recovers all tasks previously claimed by a node and returns the count.
	//
	// Takes maxRetries (int32) which is the attempt limit before a task is failed.
	// Takes recoveryError (*string) which is the error message recorded on the task, if any.
	// Takes nowUnix (int64) which is the current time in Unix seconds.
	// Takes nodeID (*string) which identifies the node holding the claims, if any.
	//
	// Returns int64 which is the number of tasks recovered.
	// Returns error when the update fails.
	RecoverClaimed(ctx context.Context, maxRetries int32, recoveryError *string, nowUnix int64, nodeID *string) (int64, error)

	// ReleaseRecoveryLeases releases all recovery leases held by a node and returns the
	// count.
	//
	// Takes nodeID (*string) which identifies the node holding the leases, if any.
	//
	// Returns int64 which is the number of leases released.
	// Returns error when the update fails.
	ReleaseRecoveryLeases(ctx context.Context, nodeID *string) (int64, error)

	// CreateWorkflowReceipt creates a workflow-completion receipt.
	//
	// Takes params (CreateWorkflowReceiptParams) which carries the fields of the receipt.
	//
	// Returns error when the insert fails.
	CreateWorkflowReceipt(ctx context.Context, params CreateWorkflowReceiptParams) error

	// ResolveWorkflowReceipts marks a workflow's pending receipts as resolved and returns
	// the count.
	//
	// Takes workflowID (string) which identifies the workflow.
	// Takes errorMessage (*string) which is the failure message stored on the receipts, if
	// any.
	// Takes nowUnix (int64) which is the resolution time in Unix seconds.
	//
	// Returns int64 which is the number of receipts resolved.
	// Returns error when the update fails.
	ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage *string, nowUnix int64) (int64, error)

	// GetPendingReceiptsByNode returns the pending receipts created by a node.
	//
	// Takes nodeID (string) which identifies the node.
	//
	// Returns []PendingReceiptRow which holds the pending receipts.
	// Returns error when the query fails.
	GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]PendingReceiptRow, error)

	// GetPendingReceiptsByWorkflow returns the pending receipts for a workflow.
	//
	// Takes workflowID (string) which identifies the workflow.
	//
	// Returns []PendingReceiptRow which holds the pending receipts.
	// Returns error when the query fails.
	GetPendingReceiptsByWorkflow(ctx context.Context, workflowID string) ([]PendingReceiptRow, error)

	// CleanupOldResolvedReceipts deletes resolved receipts older than the cutoff and returns
	// the count.
	//
	// Takes olderThanUnix (*int64) which is the cutoff time in Unix seconds, if any.
	//
	// Returns int64 which is the number of receipts deleted.
	// Returns error when the delete fails.
	CleanupOldResolvedReceipts(ctx context.Context, olderThanUnix *int64) (int64, error)

	// TimeoutStaleReceipts marks very old pending receipts as timed out and returns the
	// count.
	//
	// Takes updatedAt (int64) which is the update time in Unix seconds.
	// Takes olderThanUnix (int64) which is the cutoff time in Unix seconds.
	//
	// Returns int64 which is the number of receipts timed out.
	// Returns error when the update fails.
	TimeoutStaleReceipts(ctx context.Context, updatedAt, olderThanUnix int64) (int64, error)

	// ListFailedTasks returns all tasks in the failed state.
	//
	// Returns []FailedTaskRow which holds the failed tasks.
	// Returns error when the query fails.
	ListFailedTasks(ctx context.Context) ([]FailedTaskRow, error)

	// ListTaskStatusCounts returns task counts grouped by status.
	//
	// Returns []TaskStatusCountRow which pairs each status with its task count.
	// Returns error when the query fails.
	ListTaskStatusCounts(ctx context.Context) ([]TaskStatusCountRow, error)

	// ListRecentTasks returns the most recently updated tasks.
	//
	// Takes limit (int) which caps the number of tasks returned.
	//
	// Returns []RecentTaskRow which holds the recently updated tasks.
	// Returns error when the query fails.
	ListRecentTasks(ctx context.Context, limit int) ([]RecentTaskRow, error)

	// ListWorkflowSummary returns per-workflow aggregates.
	//
	// Takes limit (int) which caps the number of workflows returned.
	//
	// Returns []WorkflowSummaryRow which holds the per-workflow aggregates.
	// Returns error when the query fails.
	ListWorkflowSummary(ctx context.Context, limit int) ([]WorkflowSummaryRow, error)
}

// CreateTaskParams carries the serialised fields required to insert a task.
type CreateTaskParams struct {
	// ID is the task identifier.
	ID string

	// WorkflowID is the owning workflow.
	WorkflowID string

	// Executor is the executor name.
	Executor string

	// Payload is the JSON-encoded task payload.
	Payload string

	// Config is the JSON-encoded task config.
	Config string

	// Status is the task status.
	Status string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32
}

// CreateTaskBatchParams carries the serialised fields required to insert a task as part
// of a batch, including the optional deduplication key.
type CreateTaskBatchParams struct {
	// DeduplicationKey is the optional deduplication key.
	DeduplicationKey *string

	// ID is the task identifier.
	ID string

	// WorkflowID is the owning workflow.
	WorkflowID string

	// Executor is the executor name.
	Executor string

	// Payload is the JSON-encoded task payload.
	Payload string

	// Config is the JSON-encoded task config.
	Config string

	// Status is the task status.
	Status string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32
}

// UpdateTaskParams carries the serialised fields required to update a task.
type UpdateTaskParams struct {
	// LastError is the last recorded error, if any.
	LastError *string

	// Result is the JSON-encoded task result, if any.
	Result *string

	// Status is the task status.
	Status string

	// Payload is the JSON-encoded task payload.
	Payload string

	// Config is the JSON-encoded task config.
	Config string

	// ID is the task identifier.
	ID string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32
}

// FetchDueTasksParams carries the filter for fetching due tasks.
type FetchDueTasksParams struct {
	// Statuses are the task statuses eligible to be fetched.
	Statuses []string

	// Priority is the priority level to fetch.
	Priority int32

	// ExecuteAt is the upper bound (inclusive) of scheduled execution time in Unix seconds.
	ExecuteAt int64

	// Limit is the maximum number of tasks to fetch.
	Limit int
}

// FetchDueTaskRow is a due-task row.
type FetchDueTaskRow struct {
	// Result is the JSON-encoded task result, if any.
	Result *string

	// DeduplicationKey is the deduplication key, if any.
	DeduplicationKey *string

	// LastError is the last recorded error, if any.
	LastError *string

	// Payload is the JSON-encoded task payload.
	Payload string

	// Config is the JSON-encoded task config.
	Config string

	// Status is the task status.
	Status string

	// ID is the task identifier.
	ID string

	// Executor is the executor name.
	Executor string

	// WorkflowID is the owning workflow.
	WorkflowID string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32
}

// CreateWorkflowReceiptParams carries the fields required to create a workflow receipt.
type CreateWorkflowReceiptParams struct {
	// ID is the receipt identifier.
	ID string

	// WorkflowID is the tracked workflow.
	WorkflowID string

	// NodeID is the node that created the receipt.
	NodeID string

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64
}

// StaleTaskRow is a stale-task candidate.
type StaleTaskRow struct {
	// ID is the task identifier.
	ID string

	// WorkflowID is the owning workflow.
	WorkflowID string

	// Attempt is the current attempt count.
	Attempt int32
}

// FailedTaskRow is a failed-task row.
type FailedTaskRow struct {
	// LastError is the last recorded error, if any.
	LastError *string

	// ID is the task identifier.
	ID string

	// WorkflowID is the owning workflow.
	WorkflowID string

	// Executor is the executor name.
	Executor string

	// Attempt is the current attempt count.
	Attempt int32
}

// TaskStatusCountRow pairs a task status with the number of tasks in that status.
type TaskStatusCountRow struct {
	// Status is the task status.
	Status string

	// TaskCount is the number of tasks in the status.
	TaskCount int64
}

// RecentTaskRow is a recently-updated task row.
type RecentTaskRow struct {
	// LastError is the last recorded error, if any.
	LastError *string

	// ID is the task identifier.
	ID string

	// WorkflowID is the owning workflow.
	WorkflowID string

	// Executor is the executor name.
	Executor string

	// Status is the task status.
	Status string

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32
}

// WorkflowSummaryRow is a per-workflow aggregate.
type WorkflowSummaryRow struct {
	// CompleteCount is the number of complete tasks, if known.
	CompleteCount *int64

	// FailedCount is the number of failed tasks, if known.
	FailedCount *int64

	// ActiveCount is the number of active tasks, if known.
	ActiveCount *int64

	// CreatedAt is the earliest task creation time in Unix seconds, if known.
	CreatedAt *int64

	// UpdatedAt is the latest task update time in Unix seconds, if known.
	UpdatedAt *int64

	// WorkflowID identifies the workflow.
	WorkflowID string

	// TaskCount is the total number of tasks in the workflow.
	TaskCount int64
}

// PendingReceiptRow is a pending-receipt row returned by the receipt queries.
type PendingReceiptRow struct {
	// ID is the receipt identifier.
	ID string

	// WorkflowID is the tracked workflow.
	WorkflowID string

	// NodeID is the node that created the receipt. It is empty for by-node queries where the
	// caller already knows the node.
	NodeID string

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64
}
