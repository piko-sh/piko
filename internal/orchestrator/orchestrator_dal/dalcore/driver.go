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
	// Returns int64 which counts updated rows and is zero for a missing task.
	// Returns error when the update fails.
	UpdateTask(ctx context.Context, params UpdateTaskParams) (int64, error)

	// CreateWithDedup creates a task unless an active task already holds its deduplication
	// key, as one atomic statement.
	//
	// Takes params (CreateTaskParams) which carries the fields of the task to insert,
	// including its deduplication key.
	//
	// Returns bool which is true when the task was created.
	// Returns error when the insert fails.
	CreateWithDedup(ctx context.Context, params CreateTaskParams) (created bool, err error)

	// GetTasksByID returns the tasks with the given IDs, skipping IDs that do not exist.
	//
	// Takes ids ([]string) which lists the task IDs to read.
	//
	// Returns []TaskRow which holds the tasks found.
	// Returns error when the query fails.
	GetTasksByID(ctx context.Context, ids []string) ([]TaskRow, error)

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
	// Takes olderThanUnix (int64) which is the cutoff time in Unix seconds.
	//
	// Returns int64 which is the number of receipts deleted.
	// Returns error when the delete fails.
	CleanupOldResolvedReceipts(ctx context.Context, olderThanUnix int64) (int64, error)

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
	// Returns []TaskRow which holds the failed tasks.
	// Returns error when the query fails.
	ListFailedTasks(ctx context.Context) ([]TaskRow, error)

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

	// ClaimTask moves a delivered attempt to processing in one fenced statement.
	//
	// Takes params (ClaimTaskParams) which describes the claim.
	//
	// Returns TaskRow which is the claimed row.
	// Returns bool which is false when the row was not claimable.
	// Returns error when the update fails.
	ClaimTask(ctx context.Context, params ClaimTaskParams) (TaskRow, bool, error)

	// SettleTask writes a processing task's settled state, fenced by its claimed attempt.
	//
	// Takes params (TaskWriteParams) which carries the new state.
	//
	// Returns TaskRow which is the settled row, still carrying any requested rerun.
	// Returns bool which is false when the task is no longer processing that attempt.
	// Returns error when the update fails.
	SettleTask(ctx context.Context, params TaskWriteParams) (TaskRow, bool, error)

	// TransitionTask writes a task's new state, fenced by its attempt and status.
	//
	// Takes params (TaskWriteParams) which carries the new state.
	// Takes from ([]string) which lists the statuses the stored task may be in.
	//
	// Returns TaskRow which is the moved row, still carrying any requested rerun.
	// Returns bool which is false when the task has moved on or does not exist.
	// Returns error when the update fails.
	TransitionTask(ctx context.Context, params TaskWriteParams, from []string) (TaskRow, bool, error)

	// ClearRerun removes a task's rerun request.
	//
	// Takes id (string) which identifies the task.
	//
	// Returns error when the update fails.
	ClearRerun(ctx context.Context, id string) error

	// RequestRerun records a rerun on the active task holding a deduplication key.
	//
	// Takes params (RerunParams) which describes the rerun.
	//
	// Returns int64 which is the number of tasks updated.
	// Returns error when the update fails.
	RequestRerun(ctx context.Context, params RerunParams) (int64, error)

	// CountUnsettledForBuild counts a build's foreground tasks that have not settled.
	//
	// Takes buildTag (string) which identifies the build.
	//
	// Returns int64 which is the count.
	// Returns error when the query fails.
	CountUnsettledForBuild(ctx context.Context, buildTag string) (int64, error)

	// ListFailedForBuild returns a build's failed tasks.
	//
	// Takes buildTag (string) which identifies the build.
	//
	// Returns []TaskRow which holds the failed tasks.
	// Returns error when the query fails.
	ListFailedForBuild(ctx context.Context, buildTag string) ([]TaskRow, error)

	// DispatchablePage reads a page of due pending and retrying tasks.
	//
	// Takes params (DispatchPageParams) which selects the page.
	//
	// Returns []TaskRow which holds the tasks in execution order.
	// Returns error when the query fails.
	DispatchablePage(ctx context.Context, params DispatchPageParams) ([]TaskRow, error)

	// MarkPublished records when tasks were last published.
	//
	// Takes publishedAt (int64) which is the publication time in Unix seconds.
	// Takes ids ([]string) which identifies the tasks.
	//
	// Returns error when the update fails.
	MarkPublished(ctx context.Context, publishedAt int64, ids []string) error

	// ListOrphanedTasks returns processing tasks whose owner holds no live lease.
	//
	// Takes staleBefore (int64) which is the write time, in Unix seconds, orphans predate.
	// Takes now (int64) which is the time, in Unix seconds, leases are checked against.
	// Takes limit (int) which caps the number of tasks returned.
	//
	// Returns []TaskRow which holds the orphans.
	// Returns error when the query fails.
	ListOrphanedTasks(ctx context.Context, staleBefore, now int64, limit int) ([]TaskRow, error)

	// PurgeTerminal deletes completed and failed tasks last written before a cutoff whose
	// workflows have no pending receipt.
	//
	// Takes before (int64) which is the cutoff in Unix seconds.
	// Takes limit (int) which caps the number of tasks deleted.
	//
	// Returns int64 which is the number of tasks deleted.
	// Returns error when the delete fails.
	PurgeTerminal(ctx context.Context, before int64, limit int) (int64, error)

	// UpsertNode records a node lease, replacing any earlier one for the instance.
	//
	// Takes params (NodeLeaseParams) which describes the lease.
	//
	// Returns error when the write fails.
	UpsertNode(ctx context.Context, params NodeLeaseParams) error

	// RenewNode extends a node lease.
	//
	// Takes params (NodeLeaseParams) which carries the renewal and expiry times.
	//
	// Returns int64 which counts renewed leases and is zero for a missing lease.
	// Returns error when the update fails.
	RenewNode(ctx context.Context, params NodeLeaseParams) (int64, error)

	// DeleteNode removes a node lease.
	//
	// Takes instance (string) which identifies the instance.
	//
	// Returns error when the delete fails.
	DeleteNode(ctx context.Context, instance string) error

	// ListExpiredNodes returns instances whose leases lapsed before now.
	//
	// Takes now (int64) which is the current time in Unix seconds.
	// Takes limit (int) which caps the number of instances returned.
	//
	// Returns []string which holds the instances in name order.
	// Returns error when the query fails.
	ListExpiredNodes(ctx context.Context, now int64, limit int) ([]string, error)

	// TouchOwned stamps the tasks an instance is processing as written at a time.
	//
	// Takes instance (string) which identifies the instance.
	// Takes at (int64) which is the time in Unix seconds.
	//
	// Returns int64 which is the number of tasks touched.
	// Returns error when the update fails.
	TouchOwned(ctx context.Context, instance string, at int64) (int64, error)

	// IsTransient reports whether an error is a serialisation failure, deadlock or busy
	// database, after which the whole transaction can be retried.
	//
	// Takes err (error) which is the error to classify.
	//
	// Returns bool which is true when retrying may succeed.
	IsTransient(err error) bool
}

// CreateTaskParams carries the serialised fields required to insert a task.
type CreateTaskParams struct {
	// DeduplicationKey is the optional deduplication key.
	DeduplicationKey *string

	// LastError is the last recorded error, if any.
	LastError *string

	// Result is the JSON-encoded task result, if any.
	Result *string

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

	// BuildTag identifies the build the task belongs to, or is empty.
	BuildTag string

	// OwnerInstance identifies the instance processing the task, or is empty.
	OwnerInstance string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// ScheduledExecuteAt is when a scheduled task is due, in Unix seconds, or zero.
	ScheduledExecuteAt int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32

	// IsFatal is true when the task failed in a way retries cannot fix.
	IsFatal bool

	// Background is true for tasks no build waits for.
	Background bool
}

// CreateTaskBatchParams carries the serialised fields required to insert a task as part
// of a batch.
type CreateTaskBatchParams = CreateTaskParams

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

	// BuildTag identifies the build the task belongs to, or is empty.
	BuildTag string

	// OwnerInstance identifies the instance processing the task, or is empty.
	OwnerInstance string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// ScheduledExecuteAt is when a scheduled task is due, in Unix seconds, or zero.
	ScheduledExecuteAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32

	// IsFatal is true when the task failed in a way retries cannot fix.
	IsFatal bool

	// Background is true for tasks no build waits for.
	Background bool
}

// TaskRow is a full task row.
type TaskRow struct {
	// Result is the JSON-encoded task result, if any.
	Result *string

	// DeduplicationKey is the deduplication key, if any.
	DeduplicationKey *string

	// LastError is the last recorded error, if any.
	LastError *string

	// RerunID identifies the rerun requested while the task was active, if any.
	RerunID *string

	// RerunPayload is the JSON-encoded payload of the requested rerun, if any.
	RerunPayload *string

	// RerunConfig is the JSON-encoded config of the requested rerun, if any.
	RerunConfig *string

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

	// BuildTag identifies the build the task belongs to, or is empty.
	BuildTag string

	// OwnerInstance identifies the instance processing the task, or is empty.
	OwnerInstance string

	// ExecuteAt is the scheduled execution time in Unix seconds.
	ExecuteAt int64

	// CreatedAt is the creation time in Unix seconds.
	CreatedAt int64

	// UpdatedAt is the last-update time in Unix seconds.
	UpdatedAt int64

	// ScheduledExecuteAt is when a scheduled task is due, in Unix seconds, or zero.
	ScheduledExecuteAt int64

	// LastPublishedAt is when the task was last published, in Unix seconds, or zero.
	LastPublishedAt int64

	// Priority is the task priority.
	Priority int32

	// Attempt is the current attempt count.
	Attempt int32

	// IsFatal is true when the task failed in a way retries cannot fix.
	IsFatal bool

	// Background is true for tasks no build waits for.
	Background bool
}

// ClaimTaskParams carries a fenced claim.
type ClaimTaskParams struct {
	// ID is the task identifier.
	ID string

	// Owner identifies the instance claiming the task.
	Owner string

	// UpdatedAt is the claim time in Unix seconds.
	UpdatedAt int64

	// DueBy is the latest due time, in Unix seconds, a retrying or scheduled task may have.
	DueBy int64

	// Attempt is the delivered attempt.
	Attempt int32
}

// TaskWriteParams carries the state a settle or transition writes. A task's payload,
// settings and priority are fixed once it is created, so they are not written.
type TaskWriteParams struct {
	// LastError is the last recorded error, if any.
	LastError *string

	// Result is the JSON-encoded task result, if any.
	Result *string

	// ID is the task identifier.
	ID string

	// Status is the new status.
	Status string

	// OwnerInstance is the owner a transition leaves on the task; settles always clear it.
	OwnerInstance string

	// ExecuteAt is the execution time in Unix seconds.
	ExecuteAt int64

	// ScheduledExecuteAt is when a scheduled task is due, in Unix seconds, or zero.
	ScheduledExecuteAt int64

	// UpdatedAt is the write time in Unix seconds.
	UpdatedAt int64

	// Attempt is the attempt the stored task must be at.
	Attempt int32

	// IsFatal is true when the task failed in a way retries cannot fix.
	IsFatal bool
}

// RerunParams carries a rerun request.
type RerunParams struct {
	// ID identifies the task the rerun becomes.
	ID string

	// Payload is the JSON-encoded rerun payload.
	Payload string

	// Config is the JSON-encoded rerun config.
	Config string

	// DeduplicationKey is the key of the active task that must run again.
	DeduplicationKey string
}

// DispatchPageParams selects a page of due tasks.
type DispatchPageParams struct {
	// AfterID is the previous page's last task ID.
	AfterID string

	// Now is the time, in Unix seconds, tasks must be due by.
	Now int64

	// AfterTime is the previous page's last execution time in Unix seconds; a page starts
	// after the task at (AfterTime, AfterID).
	AfterTime int64

	// PublishedBefore excludes tasks last published at or after this time, in Unix seconds.
	PublishedBefore int64

	// Limit caps the page size.
	Limit int
}

// NodeLeaseParams carries a node lease.
type NodeLeaseParams struct {
	// Instance identifies the running instance.
	Instance string

	// NodeID identifies the node the instance runs on.
	NodeID string

	// StartedAt is when the instance registered, in Unix seconds.
	StartedAt int64

	// RenewedAt is when the lease was last renewed, in Unix seconds.
	RenewedAt int64

	// ExpiresAt is when the lease lapses, in Unix seconds.
	ExpiresAt int64
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
