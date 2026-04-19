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

package orchestrator_domain

import (
	"context"
	"time"
)

// TaskExecutor handles the running of tasks sent by the orchestrator.
// Implementations must be safe for use by multiple goroutines at once.
type TaskExecutor interface {
	// Execute runs the command with the given payload.
	//
	// Takes payload (map[string]any) which contains the input data for execution.
	//
	// Returns resultPayload (map[string]any) which contains the execution output.
	// Returns error when execution fails.
	Execute(ctx context.Context, payload map[string]any) (resultPayload map[string]any, err error)
}

// TaskStore provides storage and retrieval for task state.
// It handles crash recovery and workflow status queries.
type TaskStore interface {
	// CreateTask creates a new task in the system.
	//
	// Takes task (*Task) which specifies the task to create.
	//
	// Returns error when the task cannot be created.
	CreateTask(ctx context.Context, task *Task) error

	// CreateTasks stores multiple tasks in a single operation.
	//
	// Takes tasks ([]*Task) which contains the tasks to create.
	//
	// Returns error when the operation fails.
	CreateTasks(ctx context.Context, tasks []*Task) error

	// UpdateTask updates an existing task with new values.
	//
	// Takes task (*Task) which contains the updated task data.
	//
	// Returns error when the update fails.
	UpdateTask(ctx context.Context, task *Task) error

	// FetchAndMarkDueTasks retrieves tasks that are due and marks them as in
	// progress.
	//
	// Takes priority (TaskPriority) which filters tasks by their priority level.
	// Takes limit (int) which specifies the maximum number of tasks to fetch.
	//
	// Returns []*Task which contains the tasks that were fetched and marked.
	// Returns error when the fetch or mark operation fails.
	FetchAndMarkDueTasks(ctx context.Context, priority TaskPriority, limit int) ([]*Task, error)

	// GetWorkflowStatus retrieves the completion state of a workflow.
	//
	// Takes workflowID (string) which identifies the workflow to check.
	//
	// Returns isComplete (bool) which is true when the workflow has finished.
	// Returns err (error) when the workflow cannot be found or status check fails.
	GetWorkflowStatus(ctx context.Context, workflowID string) (isComplete bool, err error)

	// PromoteScheduledTasks moves tasks that are due from the scheduled queue to
	// the ready queue.
	//
	// Returns int which is the number of tasks promoted.
	// Returns error when the promotion fails.
	PromoteScheduledTasks(ctx context.Context) (int, error)

	// PendingTaskCount returns the number of tasks waiting to be processed.
	//
	// Returns int64 which is the count of pending tasks.
	// Returns error when the count cannot be retrieved.
	PendingTaskCount(ctx context.Context) (int64, error)

	// CreateTaskWithDedup creates a task with deduplication support, checking for
	// existing active tasks with the same DeduplicationKey and returning
	// ErrDuplicateTask if one exists. If DeduplicationKey is empty, it behaves
	// identically to CreateTask.
	//
	// Takes task (*Task) which is the task to create.
	//
	// Returns error when the task cannot be created or a duplicate exists.
	CreateTaskWithDedup(ctx context.Context, task *Task) error

	// RecoverStaleTasks resets PROCESSING tasks that have exceeded the
	// stale threshold. Tasks are marked as RETRYING if they have attempts
	// remaining, or FAILED if they have exceeded max retries.
	//
	// Takes staleThreshold (time.Duration) which defines how long a task can be in
	// PROCESSING before being considered stuck.
	// Takes maxRetries (int) which is the maximum retry count before a task
	// is marked FAILED.
	// Takes recoveryError (string) which is the error message to record on
	// recovered tasks.
	//
	// Returns int which is the count of tasks recovered.
	// Returns error when the recovery operation fails.
	RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error)

	// GetStaleProcessingTaskCount returns the count of tasks stuck in PROCESSING
	// longer than the threshold.
	//
	// Takes staleThreshold (time.Duration) which defines when a PROCESSING task is
	// considered stuck.
	//
	// Returns int64 which is the count of stale tasks.
	// Returns error when the count cannot be retrieved.
	GetStaleProcessingTaskCount(ctx context.Context, staleThreshold time.Duration) (int64, error)

	// UpdateTaskHeartbeat updates the updated_at timestamp for a task that is
	// currently in PROCESSING status. This prevents the task from being recovered
	// by the stale task recovery mechanism while it is still actively being worked
	// on.
	//
	// Takes taskID (string) which identifies the task to update.
	//
	// Returns error when the update fails (task not found or not in PROCESSING
	// status).
	UpdateTaskHeartbeat(ctx context.Context, taskID string) error

	// ClaimStaleTasksForRecovery atomically claims stale PROCESSING tasks for
	// recovery. It uses row-level locking to prevent multiple nodes from
	// recovering the same task.
	//
	// Takes nodeID (string) which identifies the node claiming the tasks.
	// Takes staleThreshold (time.Duration) which defines when a task is considered
	// stale.
	// Takes leaseTimeout (time.Duration) which sets how long the claim is valid.
	// Takes batchLimit (int) which limits the number of tasks to claim per call.
	//
	// Returns []RecoveryClaimedTask which contains the claimed tasks.
	// Returns error when the claim operation fails.
	ClaimStaleTasksForRecovery(ctx context.Context, nodeID string, staleThreshold time.Duration, leaseTimeout time.Duration, batchLimit int) ([]RecoveryClaimedTask, error)

	// RecoverClaimedTasks recovers tasks previously claimed by a node. Tasks are
	// set to RETRYING if attempts remain, or FAILED otherwise, and the lease is
	// cleared.
	//
	// Takes nodeID (string) which identifies the node that claimed the tasks.
	// Takes maxRetries (int) which is the max retries before marking FAILED.
	// Takes recoveryError (string) which is the error message to record.
	//
	// Returns int which is the count of tasks recovered.
	// Returns error when the recovery fails.
	RecoverClaimedTasks(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error)

	// ReleaseRecoveryLeases releases all recovery leases held by this node.
	// Called during graceful shutdown to allow other nodes to recover the tasks.
	//
	// Takes nodeID (string) which identifies the node releasing leases.
	//
	// Returns int which is the count of leases released.
	// Returns error when the release fails.
	ReleaseRecoveryLeases(ctx context.Context, nodeID string) (int, error)

	// CreateWorkflowReceipt creates a new workflow receipt for tracking
	// completion. Receipts are used to notify the originating node when a workflow
	// completes, even if the tasks were processed by other nodes.
	//
	// Takes id (string) which is the unique identifier for the receipt.
	// Takes workflowID (string) which is the workflow being tracked.
	// Takes nodeID (string) which is the node that created the receipt.
	//
	// Returns error when the receipt cannot be created.
	CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error

	// ResolveWorkflowReceipts marks all pending receipts for a workflow as
	// resolved. Called when a workflow completes (all tasks finished).
	//
	// Takes workflowID (string) which identifies the completed workflow.
	// Takes errorMessage (string) which contains any error from workflow
	// completion (empty if success).
	//
	// Returns int which is the count of receipts resolved.
	// Returns error when the resolution fails.
	ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage string) (int, error)

	// GetPendingReceiptsByNode retrieves all pending receipts created by a node.
	// Used during startup recovery to check if workflows completed while the node
	// was down.
	//
	// Takes nodeID (string) which identifies the node.
	//
	// Returns []PendingReceipt which contains pending receipts for this node.
	// Returns error when the query fails.
	GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]PendingReceipt, error)

	// CleanupOldResolvedReceipts deletes resolved receipts older than the cutoff.
	// Used for periodic cleanup to prevent table bloat.
	//
	// Takes olderThan (time.Time) which is the cutoff for deletion.
	//
	// Returns int which is the count of receipts deleted.
	// Returns error when the cleanup fails.
	CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error)

	// TimeoutStaleReceipts marks very old pending receipts as timed out.
	// Prevents receipts from lingering indefinitely if workflows are abandoned.
	//
	// Takes olderThan (time.Time) which is the cutoff for timeout.
	//
	// Returns int which is the count of receipts timed out.
	// Returns error when the timeout operation fails.
	TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error)

	// ListFailedTasks returns all tasks that are in the FAILED state.
	// Used for build error reporting so users can see which tasks failed and why.
	//
	// Returns []*Task which contains all failed tasks with their error details.
	// Returns error when the query fails.
	ListFailedTasks(ctx context.Context) ([]*Task, error)

	// RunAtomic executes fn within a transaction.
	//
	// The provided TaskStore is scoped to the
	// transaction; all reads and writes through it are
	// atomic. If fn returns an error (or panics), all
	// mutations are rolled back.
	//
	// Takes fn which receives a transactional TaskStore.
	// The caller MUST use this transactional store for
	// all operations that should be atomic.
	//
	// Returns error when fn returns an error or the
	// transaction fails to commit.
	RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error
}

// RecoveryClaimedTask represents a task that has been claimed for recovery.
// This is returned by ClaimStaleTasksForRecovery so the caller knows which
// tasks were claimed and their current attempt count.
type RecoveryClaimedTask struct {
	// ID is the unique identifier for the claimed task.
	ID string

	// WorkflowID is the unique identifier for the workflow.
	WorkflowID string

	// Attempt is the retry attempt number for this task.
	Attempt int32
}

// PendingReceipt represents a workflow receipt that is waiting to be resolved.
// Used for startup recovery and workflow status checking.
type PendingReceipt struct {
	// ID is the unique identifier for this pending receipt.
	ID string

	// WorkflowID is the unique identifier for the workflow.
	WorkflowID string

	// NodeID is the identifier of the node that processed the receipt.
	NodeID string

	// CreatedAt is the Unix timestamp when the receipt was created.
	CreatedAt int64
}

// OrchestratorService manages task lifecycle including dispatch, scheduling,
// and execution. It coordinates workers and tracks workflow completion.
type OrchestratorService interface {
	// RegisterExecutor adds a named task executor to the registry.
	//
	// Takes ctx (context.Context) which carries logging context.
	// Takes name (string) which identifies the executor.
	// Takes executor (TaskExecutor) which handles tasks of the named kind.
	//
	// Returns error when registration fails or the name is already taken.
	RegisterExecutor(ctx context.Context, name string, executor TaskExecutor) error

	// Dispatch sends a task for processing and returns a receipt.
	//
	// Takes task (*Task) which contains the work to be dispatched.
	//
	// Returns *WorkflowReceipt which confirms the task was accepted.
	// Returns error when the dispatch fails.
	Dispatch(ctx context.Context, task *Task) (*WorkflowReceipt, error)

	// Schedule queues a task for execution at a specified time.
	//
	// Takes task (*Task) which defines the work to be performed.
	// Takes executeAt (time.Time) which specifies when the task should run.
	//
	// Returns *WorkflowReceipt which confirms the task has been scheduled.
	// Returns error when scheduling fails.
	Schedule(ctx context.Context, task *Task, executeAt time.Time) (*WorkflowReceipt, error)

	// Run starts the orchestrator's background processes and blocks until the
	// context is cancelled.
	Run(ctx context.Context)

	// Stop halts the orchestrator and releases any associated resources.
	Stop()

	// ActiveTasks returns the number of tasks being processed at this moment.
	//
	// Returns int64 which is the count of active tasks.
	ActiveTasks(ctx context.Context) int64

	// PendingTasks returns the number of tasks waiting to be processed.
	//
	// Returns int64 which is the count of tasks that have not yet started.
	PendingTasks(ctx context.Context) int64

	// GetTaskDispatcher returns the TaskDispatcher for direct task dispatch.
	// The bridge uses this for competing-consumer task distribution.
	//
	// Returns TaskDispatcher which handles task dispatch operations.
	GetTaskDispatcher() TaskDispatcher

	// DispatchDirect saves and dispatches a task straight away, without using
	// the async batch queue. This is mainly for testing where you need tasks to
	// run in a set order.
	//
	// Takes task (*Task) which is the task to dispatch.
	//
	// Returns *WorkflowReceipt which confirms the task was dispatched.
	// Returns error when the task cannot be saved or dispatched.
	DispatchDirect(ctx context.Context, task *Task) (*WorkflowReceipt, error)
}

// EventBus provides pub/sub messaging with both channel-based and
// handler-based subscriptions. It is backed by WatermillEventBus which
// wraps Watermill's GoChannel provider.
//
// When using SubscribeWithHandler:
//   - Return nil -> Ack (message removed from queue)
//   - Return error -> Nack (message redelivered based on backend configuration)
//   - Handler is called concurrently for different messages
type EventBus interface {
	// Publish sends an event to the given topic.
	//
	// Takes topic (string) which identifies where the event is sent.
	// Takes event (Event) which contains the data to publish.
	//
	// Returns error when publishing fails.
	Publish(ctx context.Context, topic string, event Event) error

	// Subscribe returns a channel that receives events for the given topic.
	//
	// Takes topic (string) which specifies the event topic to subscribe to.
	// Supports wildcard patterns (e.g., "artefact.*").
	//
	// Returns <-chan Event which yields events as they are published.
	// Returns error when the subscription cannot be created.
	Subscribe(ctx context.Context, topic string) (<-chan Event, error)

	// SubscribeWithHandler subscribes to a topic and invokes the handler for each
	// received message.
	//
	// Takes topic (string) which specifies the topic to subscribe to.
	// Takes handler (EventHandler) which processes received messages.
	//
	// Returns error when the subscription fails.
	//
	// The handler is called concurrently for different messages. Wildcard
	// patterns are supported (e.g., "artefact.*").
	SubscribeWithHandler(ctx context.Context, topic string, handler EventHandler) error

	// Close releases all resources and closes active subscriptions.
	//
	// Takes ctx (context.Context) which carries logging context for the
	// shutdown operation.
	//
	// Returns error when the close operation fails.
	Close(ctx context.Context) error
}

// EventHandler defines a callback that processes subscription events, returning
// nil to acknowledge success or an error to trigger redelivery. Handlers must
// be idempotent as messages may be delivered multiple times.
type EventHandler func(ctx context.Context, event Event) error

// DelayedPublisher handles scheduling tasks for future execution.
// It keeps a min-heap of tasks ordered by their scheduled time and sends them
// when they become due.
type DelayedPublisher interface {
	// Schedule adds a task to run at its ScheduledExecuteAt time.
	//
	// Takes task (*Task) which is the task to schedule.
	//
	// Returns error when the task has no scheduled time set.
	Schedule(ctx context.Context, task *Task) error

	// Start begins the delayed task processing loop. Call this once before
	// scheduling tasks.
	Start(ctx context.Context)

	// Stop shuts down the publisher in a controlled way.
	Stop()

	// PendingCount returns the number of tasks waiting to be sent.
	// Used for idle detection; the dispatcher is not idle while tasks are pending.
	//
	// Returns int which is the count of tasks not yet dispatched.
	PendingCount() int
}
