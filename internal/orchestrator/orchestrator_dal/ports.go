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

package orchestrator_dal

import (
	"context"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// TaskDAL defines the interface for persisting and querying task state. It implements
// orchestrator_domain.TaskStore but is defined in the DAL layer to separate domain ports
// from data access implementation.
type TaskDAL interface {
	orchestrator_domain.TaskStoreV2

	// CreateTask saves a new task to the database.
	//
	// Takes task (*orchestrator_domain.Task) which is the task to save.
	//
	// Returns error when the task cannot be saved.
	CreateTask(ctx context.Context, task *orchestrator_domain.Task) error

	// CreateTasks inserts a batch of tasks into the database.
	//
	// Designed for bulk insertions; uses multi-row INSERT statements for better performance.
	//
	// Takes tasks ([]*orchestrator_domain.Task) which contains the tasks to store.
	//
	// Returns error when the database operation fails.
	CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error

	// UpdateTask updates an existing task in the database.
	//
	// Takes task (*orchestrator_domain.Task) which is the task to update.
	//
	// Returns error when the update fails.
	UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error

	// GetTasksByID reads the tasks with the given IDs without changing them. IDs that do not
	// exist are skipped.
	//
	// Takes ids ([]string) which lists the task IDs to read.
	//
	// Returns []*orchestrator_domain.Task which holds the tasks found.
	// Returns error when the read fails.
	GetTasksByID(ctx context.Context, ids []string) ([]*orchestrator_domain.Task, error)

	// GetWorkflowStatus checks whether all tasks in a workflow are complete.
	//
	// Takes workflowID (string) which identifies the workflow to check.
	//
	// Returns isComplete (bool) which is true when all tasks are done.
	// Returns err (error) when the workflow cannot be found or checked.
	GetWorkflowStatus(ctx context.Context, workflowID string) (isComplete bool, err error)

	// PromoteScheduledTasks moves scheduled tasks that are now due to pending status.
	//
	// Returns int which is the number of tasks promoted.
	// Returns error when the promotion fails.
	PromoteScheduledTasks(ctx context.Context) (int, error)

	// PendingTaskCount returns the number of tasks that are waiting to be run.
	//
	// Returns int64 which is the count of pending tasks.
	// Returns error when the count cannot be retrieved.
	PendingTaskCount(ctx context.Context) (int64, error)

	// CreateTaskWithDedup creates a task with deduplication based on its DeduplicationKey.
	// It returns ErrDuplicateTask if an active task with the same key exists, or behaves
	// like CreateTask if the key is empty.
	//
	// Takes task (*orchestrator_domain.Task) which is the task to create.
	//
	// Returns error when the task cannot be created or a duplicate exists.
	CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error

	// CreateWorkflowReceipt creates a new workflow receipt for tracking completion. Receipts
	// are used to notify the originating node when a workflow completes, even if the tasks
	// were processed by other nodes.
	//
	// Takes id (string) which is the unique identifier for the receipt.
	// Takes workflowID (string) which is the workflow being tracked.
	// Takes nodeID (string) which is the node that created the receipt.
	//
	// Returns error when the receipt cannot be created.
	CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error

	// ResolveWorkflowReceipts marks all pending receipts for a workflow as resolved. Called
	// when a workflow completes (all tasks finished).
	//
	// Takes workflowID (string) which identifies the completed workflow.
	// Takes errorMessage (string) which contains any error from workflow completion (empty
	// if success).
	//
	// Returns int which is the count of receipts resolved.
	// Returns error when the resolution fails.
	ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage string) (int, error)

	// GetPendingReceiptsByNode retrieves all pending receipts created by a node. Used during
	// startup recovery to check if workflows completed while the node was down.
	//
	// Takes nodeID (string) which identifies the node.
	//
	// Returns []orchestrator_domain.PendingReceipt which contains pending receipts for this
	// node.
	// Returns error when the query fails.
	GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error)

	// GetPendingReceiptsByWorkflow retrieves all pending receipts for a workflow. Used to
	// check which nodes are waiting for a workflow to complete.
	//
	// Takes workflowID (string) which identifies the workflow.
	//
	// Returns []orchestrator_domain.PendingReceipt which contains pending receipts for this
	// workflow.
	// Returns error when the query fails.
	GetPendingReceiptsByWorkflow(ctx context.Context, workflowID string) ([]orchestrator_domain.PendingReceipt, error)

	// CleanupOldResolvedReceipts deletes resolved receipts older than the specified time.
	// Used for periodic cleanup to prevent table bloat.
	//
	// Takes olderThan (time.Time) which is the cutoff for deletion.
	//
	// Returns int which is the count of receipts deleted.
	// Returns error when the cleanup fails.
	CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error)

	// TimeoutStaleReceipts marks very old pending receipts as timed out. Prevents receipts
	// from lingering indefinitely if workflows are abandoned.
	//
	// Takes olderThan (time.Time) which is the cutoff for timeout.
	//
	// Returns int which is the count of receipts timed out.
	// Returns error when the timeout operation fails.
	TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error)

	// ListFailedTasks returns all tasks that are in the FAILED state. Used for build error
	// reporting so users can see which tasks failed and why.
	//
	// Returns []*orchestrator_domain.Task which contains all failed tasks.
	// Returns error when the query fails.
	ListFailedTasks(ctx context.Context) ([]*orchestrator_domain.Task, error)
}

// OrchestratorDAL combines all data access interfaces for the orchestrator. It implements
// TaskStore and io.Closer.
type OrchestratorDAL interface {
	TaskDAL

	// HealthCheck checks that the database connection is working.
	//
	// Returns error when the connection is not healthy.
	HealthCheck(ctx context.Context) error

	// Close releases any resources held by the DAL.
	//
	// Returns error when the resources cannot be released.
	Close() error
}

// OrchestratorDALWithTx extends OrchestratorDAL with transaction support.
type OrchestratorDALWithTx interface {
	OrchestratorDAL

	// RunAtomic executes fn within a transaction.
	//
	// The provided TaskStore is scoped to the transaction; all reads and writes through it
	// are atomic. If fn returns an error (or panics), all mutations are rolled back.
	//
	// Takes fn (func(ctx context.Context, transactionStore orchestrator_domain.TaskStore)
	// error) which is the work to run inside the transaction.
	//
	// Returns error when fn fails or the transaction cannot be committed.
	RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error
}
