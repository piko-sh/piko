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

package otter

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// defaultCacheCapacity is the default number of items the cache can hold.
	defaultCacheCapacity = 100_000

	// maxTransactionTimeout is the maximum duration a RunAtomic transaction
	// may hold the mutex before the context is cancelled.
	maxTransactionTimeout = 30 * time.Second

	// receiptPending is the initial status for workflow receipts awaiting processing.
	receiptPending receiptStatus = "PENDING"

	// receiptResolved indicates the receipt has been processed and resolved.
	receiptResolved receiptStatus = "RESOLVED"

	// receiptTimedOut indicates the receipt expired before being processed.
	receiptTimedOut receiptStatus = "TIMED_OUT"
)

var (
	// log is the package-level logger for the otter package.
	log = logger_domain.GetLogger("piko/internal/orchestrator/orchestrator_dal/otter")

	_ orchestrator_dal.OrchestratorDALWithTx = (*DAL)(nil)

	_ orchestrator_domain.OrchestratorInspector = (*DAL)(nil)

	_ orchestrator_domain.TaskStore = (*otterTransactionDAL)(nil)
)

// receiptStatus represents the current state of a workflow receipt.
type receiptStatus string

// receipt represents a workflow completion receipt.
type receipt struct {
	// createdAt is when the receipt was created.
	createdAt time.Time

	// resolvedAt is when the receipt was resolved or timed out; zero means pending.
	resolvedAt time.Time

	// id is the unique identifier for this receipt.
	id string

	// workflowID is the identifier of the workflow this receipt belongs to.
	workflowID string

	// nodeID identifies the workflow node that generated this receipt.
	nodeID string

	// status tracks whether this receipt is pending or resolved.
	status receiptStatus

	// errorMessage contains the error text when the receipt is resolved with
	// a failure; empty when resolved successfully.
	errorMessage string
}

// recoveryLease tracks a claimed recovery for a task.
type recoveryLease struct {
	// claimedAt is when this lease was claimed.
	claimedAt time.Time

	// expiresAt is when the recovery lease expires.
	expiresAt time.Time

	// taskID string // taskID is the unique identifier of the task holding this lease.
	taskID string

	// nodeID identifies the node that holds this recovery lease.
	nodeID string
}

// DAL provides in-memory storage for orchestrator tasks using otter cache.
// It implements OrchestratorDALWithTx and OrchestratorInspector.
type DAL struct {
	// tasks is the main cache for tasks, keyed by task ID.
	// Uses the cache hexagon's ProviderPort for optional WAL persistence.
	tasks cache_domain.ProviderPort[string, *orchestrator_domain.Task]

	// scheduledIndex maps task IDs to their scheduled execution times.
	// Supports range queries to find tasks ready to run.
	scheduledIndex *provider_otter.SortedIndex[string]

	// executeIndex stores pending and retrying tasks sorted by their ExecuteAt
	// time. Used for efficient time-based range queries to find tasks ready to run.
	executeIndex *provider_otter.SortedIndex[string]

	// workflowIndex maps workflow IDs to their task IDs. Uses the cache
	// hexagon's TagIndex for set membership operations.
	workflowIndex *provider_otter.TagIndex[string]

	// dedupIndex maps deduplication keys to task IDs for detecting duplicate tasks.
	dedupIndex map[string]string

	// recoveryLeases maps task IDs to their claimed recovery leases.
	recoveryLeases map[string]*recoveryLease

	// receipts maps receipt IDs to workflow receipts.
	receipts map[string]*receipt

	// receiptsByWorkflow maps workflow IDs to their receipt IDs for quick lookup.
	receiptsByWorkflow *provider_otter.TagIndex[string]

	// receiptsByNode maps node IDs to their receipt IDs for quick lookup.
	receiptsByNode *provider_otter.TagIndex[string]

	// ownsCache indicates whether this DAL owns the cache and should close it.
	// When false, the cache was injected externally and the caller handles cleanup.
	ownsCache bool

	// mu guards concurrent access to indexes and maps.
	mu sync.RWMutex
}

// Config holds settings for the otter-based orchestrator DAL.
type Config struct {
	// Capacity is the maximum number of items to store.
	// Defaults to 100,000 if zero or negative.
	Capacity int64
}

// Option configures the DAL during construction.
type Option func(*DAL)

// HealthCheck verifies the DAL is operational.
//
// Returns error which is always nil for in-memory storage.
func (*DAL) HealthCheck(_ context.Context) error {
	return nil
}

// Close releases resources held by the DAL.
//
// Returns error when the cache cannot be closed cleanly.
func (d *DAL) Close() error {
	if d.ownsCache {
		return d.tasks.Close(context.Background())
	}
	return nil
}

// RunAtomic executes fn within a transaction.
//
// For in-memory storage, this acquires a write lock and
// provides a transaction-scoped TaskStore that delegates to
// locked method variants to avoid mutex re-entrancy
// deadlocks. The task cache is wrapped in a journal-based
// transaction for rollback, and non-cache state (recovery
// leases, receipts) is snapshotted at transaction start.
//
// If fn returns an error or panics, all mutations are
// rolled back. A maximum transaction timeout is applied via
// context.WithTimeoutCause to prevent unbounded lock
// holding.
//
// Takes fn (func) which receives a transactional TaskStore.
//
// Returns error when fn returns an error or the transaction
// fails to commit.
//
// Panics if fn panics. The transaction is rolled back before
// the panic is re-raised.
//
// Safe for concurrent use.
func (d *DAL) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	d.mu.Lock()
	defer d.mu.Unlock()

	snap := d.snapshotNonCacheState()
	realCache := d.tasks
	txCache := cache_domain.BeginTransaction(ctx, realCache)
	d.tasks = txCache
	defer func() { d.tasks = realCache }()

	rollbackCtx := context.WithoutCancel(ctx)

	rollback := func() {
		_ = txCache.Rollback(rollbackCtx)
		d.tasks = realCache
		d.restoreFromSnapshot(snap)
	}

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				rollback()
				panic(r)
			}
		}()
		err = fn(ctx, &otterTransactionDAL{parent: d})
	}()

	if err != nil {
		rollback()
		return err
	}

	if commitErr := txCache.Commit(ctx); commitErr != nil {
		rollback()
		return fmt.Errorf("committing transaction: %w", commitErr)
	}
	return nil
}

// CreateTask saves a new task to storage.
//
// Takes task (*orchestrator_domain.Task) which is the task to save.
//
// Returns error when the task cannot be saved.
//
// Safe for concurrent use.
func (d *DAL) CreateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.createTaskLocked(ctx, task)
}

// CreateTasks inserts a batch of tasks.
//
// Takes tasks ([]*orchestrator_domain.Task) which contains the tasks to store.
//
// Returns error when the database operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (d *DAL) CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.createTasksLocked(ctx, tasks)
}

// UpdateTask updates an existing task.
//
// Takes task (*orchestrator_domain.Task) which is the task to update.
//
// Returns error when the update fails.
//
// Safe for concurrent use. The method holds a mutex lock while updating.
func (d *DAL) UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.updateTaskLocked(ctx, task)
}

// FetchAndMarkDueTasks fetches tasks that are due and marks them as processing.
//
// Takes priority (TaskPriority) which filters tasks by their priority level.
// Takes limit (int) which sets the maximum number of tasks to fetch.
//
// Returns []*Task which contains the tasks now marked as processing.
// Returns error when the fetch or update fails.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (d *DAL) FetchAndMarkDueTasks(ctx context.Context, priority orchestrator_domain.TaskPriority, limit int) ([]*orchestrator_domain.Task, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.fetchAndMarkDueTasksLocked(ctx, priority, limit)
}

// GetWorkflowStatus checks whether all tasks in a workflow are complete.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns bool which is true when all tasks are done or failed.
// Returns error when the workflow cannot be found.
//
// Safe for concurrent use. Uses a read lock to protect access to workflow data.
func (d *DAL) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getWorkflowStatusLocked(ctx, workflowID)
}

// PromoteScheduledTasks moves scheduled tasks that are now due to pending status.
//
// Returns int which is the number of tasks promoted.
// Returns error when the promotion fails.
//
// Safe for concurrent use. Protected by mutex.
func (d *DAL) PromoteScheduledTasks(ctx context.Context) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.promoteScheduledTasksLocked(ctx)
}

// PendingTaskCount returns the number of tasks that are waiting to be run.
//
// Returns int64 which is the count of pending tasks.
// Returns error when the count cannot be retrieved.
//
// Safe for concurrent use.
func (d *DAL) PendingTaskCount(ctx context.Context) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.pendingTaskCountLocked(ctx)
}

// CreateTaskWithDedup creates a task with deduplication support.
//
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the task cannot be created or a duplicate exists.
//
// Safe for concurrent use. Uses a mutex to protect the dedup index and task
// store.
func (d *DAL) CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.createTaskWithDedupLocked(ctx, task)
}

// RecoverStaleTasks resets PROCESSING tasks that have exceeded the stale
// threshold.
//
// Takes staleThreshold (time.Duration) which defines how long a task can be in
// PROCESSING before being considered stuck.
// Takes maxRetries (int) which is the maximum retry attempts before marking
// FAILED.
// Takes recoveryError (string) which is the error message to record on
// recovered tasks.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery operation fails.
//
// Safe for concurrent use. The method holds the DAL mutex for its duration.
func (d *DAL) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.recoverStaleTasksLocked(ctx, staleThreshold, maxRetries, recoveryError)
}

// GetStaleProcessingTaskCount returns the count of tasks stuck in PROCESSING
// longer than the threshold.
//
// Takes staleThreshold (time.Duration) which defines when a PROCESSING task is
// considered stuck.
//
// Returns int64 which is the count of stale tasks.
// Returns error when the count cannot be retrieved.
//
// Safe for concurrent use; holds a read lock during the count operation.
func (d *DAL) GetStaleProcessingTaskCount(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getStaleProcessingTaskCountLocked(ctx, staleThreshold)
}

// UpdateTaskHeartbeat updates the updated_at timestamp for a task.
//
// Takes taskID (string) which identifies the task to update.
//
// Returns error when the task is not found or is not in PROCESSING status.
//
// Safe for concurrent use.
func (d *DAL) UpdateTaskHeartbeat(ctx context.Context, taskID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.updateTaskHeartbeatLocked(ctx, taskID)
}

// ClaimStaleTasksForRecovery atomically claims stale PROCESSING tasks for
// recovery.
//
// Takes nodeID (string) which identifies the node claiming the tasks.
// Takes staleThreshold (time.Duration) which defines when a task is considered
// stale.
// Takes leaseTimeout (time.Duration) which sets how long the claim is valid.
// Takes batchLimit (int) which limits the number of tasks to claim per call.
//
// Returns []orchestrator_domain.RecoveryClaimedTask which contains the claimed
// tasks.
// Returns error when the claim operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (d *DAL) ClaimStaleTasksForRecovery(ctx context.Context, nodeID string, staleThreshold, leaseTimeout time.Duration, batchLimit int) ([]orchestrator_domain.RecoveryClaimedTask, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.claimStaleTasksForRecoveryLocked(ctx, nodeID, staleThreshold, leaseTimeout, batchLimit)
}

// RecoverClaimedTasks recovers all tasks previously claimed by this node.
//
// Takes nodeID (string) which identifies the node that claimed the tasks.
// Takes maxRetries (int) which is the maximum retry attempts before marking
// FAILED.
// Takes recoveryError (string) which is the error message to record.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery fails.
//
// Safe for concurrent use; holds the DAL mutex for the duration of the
// operation.
func (d *DAL) RecoverClaimedTasks(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.recoverClaimedTasksLocked(ctx, nodeID, maxRetries, recoveryError)
}

// ReleaseRecoveryLeases releases all recovery leases held by this node.
//
// Takes nodeID (string) which identifies the node releasing leases.
//
// Returns int which is the count of leases released.
// Returns error when the release fails.
//
// Safe for concurrent use. Uses a mutex to protect access to the lease map.
func (d *DAL) ReleaseRecoveryLeases(ctx context.Context, nodeID string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.releaseRecoveryLeasesLocked(ctx, nodeID)
}

// CreateWorkflowReceipt creates a new workflow receipt for tracking completion.
//
// Takes id (string) which is the unique identifier for the receipt.
// Takes workflowID (string) which is the workflow being tracked.
// Takes nodeID (string) which is the node that created the receipt.
//
// Returns error when the receipt cannot be created.
//
// Safe for concurrent use.
func (d *DAL) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.createWorkflowReceiptLocked(ctx, id, workflowID, nodeID)
}

// ResolveWorkflowReceipts marks all pending receipts for a workflow as resolved.
//
// Takes workflowID (string) which identifies the completed workflow.
// Takes errorMessage (string) which contains any error from workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
//
// Safe for concurrent use. Protected by a mutex lock.
func (d *DAL) ResolveWorkflowReceipts(ctx context.Context, workflowID, errorMessage string) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.resolveWorkflowReceiptsLocked(ctx, workflowID, errorMessage)
}

// GetPendingReceiptsByNode retrieves all pending receipts created by a node.
//
// Takes nodeID (string) which identifies the node.
//
// Returns []orchestrator_domain.PendingReceipt which contains pending receipts.
// Returns error when the query fails.
//
// Safe for concurrent use; holds a read lock during retrieval.
func (d *DAL) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getPendingReceiptsByNodeLocked(ctx, nodeID)
}

// GetPendingReceiptsByWorkflow retrieves all pending receipts for a workflow.
//
// Takes workflowID (string) which identifies the workflow.
//
// Returns []orchestrator_domain.PendingReceipt which contains pending receipts.
// Returns error when the query fails.
//
// Safe for concurrent use. Uses a read lock to protect access to internal data.
func (d *DAL) GetPendingReceiptsByWorkflow(_ context.Context, workflowID string) ([]orchestrator_domain.PendingReceipt, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	receiptIDs := d.receiptsByWorkflow.Get(workflowID)
	return d.getPendingReceiptsFromIDs(receiptIDs), nil
}

// CleanupOldResolvedReceipts deletes resolved receipts older than the
// specified time.
//
// Takes olderThan (time.Time) which is the cutoff for deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
//
// Safe for concurrent use. The method holds a mutex lock for its duration.
func (d *DAL) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cleanupOldResolvedReceiptsLocked(ctx, olderThan)
}

// TimeoutStaleReceipts marks very old pending receipts as timed out.
//
// Takes olderThan (time.Time) which is the cutoff for timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
//
// Safe for concurrent use; protected by mutex.
func (d *DAL) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.timeoutStaleReceiptsLocked(ctx, olderThan)
}

// ListFailedTasks returns all tasks with a FAILED status.
//
// Returns []*orchestrator_domain.Task which contains the failed tasks.
// Returns error (always nil for the in-memory store).
//
// Safe for concurrent use.
func (d *DAL) ListFailedTasks(ctx context.Context) ([]*orchestrator_domain.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listFailedTasksLocked(ctx)
}

// ListTaskSummary returns task counts grouped by status.
//
// Returns []orchestrator_domain.TaskSummary which contains the count for each
// status.
// Returns error when the query fails.
//
// Safe for concurrent use.
func (d *DAL) ListTaskSummary(_ context.Context) ([]orchestrator_domain.TaskSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	statusCounts := make(map[string]int64)
	for _, task := range d.tasks.All() {
		statusCounts[string(task.Status)]++
	}

	results := make([]orchestrator_domain.TaskSummary, 0, len(statusCounts))
	for status, count := range statusCounts {
		results = append(results, orchestrator_domain.TaskSummary{
			Status: status,
			Count:  count,
		})
	}

	return results, nil
}

// ListRecentTasks returns the most recently updated tasks.
//
// Takes limit (int32) which specifies the maximum number of tasks to return.
//
// Returns []orchestrator_domain.TaskListItem which contains the task data
// for display.
// Returns error when the query fails.
//
// Safe for concurrent use; holds a read lock while accessing task data.
func (d *DAL) ListRecentTasks(_ context.Context, limit int32) ([]orchestrator_domain.TaskListItem, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	type sortableTask struct {
		task *orchestrator_domain.Task
	}
	items := make([]sortableTask, 0, d.tasks.EstimatedSize())
	for _, task := range d.tasks.All() {
		items = append(items, sortableTask{task: task})
	}

	for i := range len(items) - 1 {
		for j := i + 1; j < len(items); j++ {
			if items[j].task.UpdatedAt.After(items[i].task.UpdatedAt) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	count := min(int(limit), len(items))
	results := make([]orchestrator_domain.TaskListItem, count)
	for i := range count {
		task := items[i].task
		var lastError *string
		if task.LastError != "" {
			lastError = &task.LastError
		}

		results[i] = orchestrator_domain.TaskListItem{
			ID:         task.ID,
			WorkflowID: task.WorkflowID,
			Executor:   task.Executor,
			Status:     string(task.Status),
			Priority:   safeconv.IntToInt32(int(task.Config.Priority)),
			Attempt:    safeconv.IntToInt32(task.Attempt),
			CreatedAt:  task.CreatedAt.Unix(),
			UpdatedAt:  task.UpdatedAt.Unix(),
			LastError:  lastError,
		}
	}

	return results, nil
}

// ListWorkflowSummary returns workflow-level aggregates.
//
// Takes limit (int32) which specifies the maximum number of workflows to return.
//
// Returns []orchestrator_domain.WorkflowSummary which contains aggregated
// workflow data sorted by most recently updated.
// Returns error when the query fails.
//
// Safe for concurrent use; holds a read lock for the duration of the call.
func (d *DAL) ListWorkflowSummary(_ context.Context, limit int32) ([]orchestrator_domain.WorkflowSummary, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	workflows := d.aggregateWorkflowData()

	type sortableWorkflow struct {
		agg *workflowAgg
		id  string
	}
	sorted := make([]sortableWorkflow, 0, len(workflows))
	for id, agg := range workflows {
		sorted = append(sorted, sortableWorkflow{id: id, agg: agg})
	}
	slices.SortFunc(sorted, func(a, b sortableWorkflow) int {
		return cmp.Compare(b.agg.updatedAt, a.agg.updatedAt)
	})

	count := min(int(limit), len(sorted))
	results := make([]orchestrator_domain.WorkflowSummary, count)
	for i := range count {
		results[i] = orchestrator_domain.WorkflowSummary{
			WorkflowID:    sorted[i].id,
			TaskCount:     sorted[i].agg.taskCount,
			CompleteCount: sorted[i].agg.completeCount,
			FailedCount:   sorted[i].agg.failedCount,
			ActiveCount:   sorted[i].agg.activeCount,
			CreatedAt:     sorted[i].agg.createdAt,
			UpdatedAt:     sorted[i].agg.updatedAt,
		}
	}

	return results, nil
}

// RebuildIndexes rebuilds all secondary indexes from the primary cache data.
// Call this after WAL recovery to restore scheduledIndex, executeIndex,
// workflowIndex, and dedupIndex.
//
// Safe for concurrent use; holds the mutex for the entire operation.
//
// Note: recoveryLeases, receipts, and related receipt indexes are not
// recovered. These are ephemeral data that reset on restart. Recovery leases
// are tied to node instances and receipts track in-flight workflow completion
// notifications.
func (d *DAL) RebuildIndexes(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.scheduledIndex = provider_otter.NewSortedIndex[string]()
	d.executeIndex = provider_otter.NewSortedIndex[string]()
	d.workflowIndex = provider_otter.NewTagIndex[string]()
	d.dedupIndex = make(map[string]string)
	d.recoveryLeases = make(map[string]*recoveryLease)
	d.receipts = make(map[string]*receipt)
	d.receiptsByWorkflow = provider_otter.NewTagIndex[string]()
	d.receiptsByNode = provider_otter.NewTagIndex[string]()

	for _, task := range d.tasks.All() {
		d.indexTaskLocked(task)
	}

	_, rl := logger_domain.From(ctx, log)
	rl.Internal("Orchestrator indexes rebuilt from cache",
		logger_domain.Int("task_count", d.tasks.EstimatedSize()))
}

// transactionSnapshot holds deep copies of non-cache mutable state captured
// at the start of a RunAtomic transaction. On rollback, these are restored to
// undo mutations that are not tracked by the cache transaction journal.
type transactionSnapshot struct {
	// recoveryLeases holds deep copies of claimed recovery
	// leases keyed by task ID.
	recoveryLeases map[string]*recoveryLease

	// receipts holds deep copies of workflow receipts keyed
	// by receipt ID.
	receipts map[string]*receipt
}

// snapshotNonCacheState deep-copies the mutable maps that
// are not covered by the cache transaction journal. Caller
// must hold mu.
//
// Returns *transactionSnapshot which contains deep copies
// of recovery leases and receipts.
func (d *DAL) snapshotNonCacheState() *transactionSnapshot {
	leasesCopy := make(map[string]*recoveryLease, len(d.recoveryLeases))
	for k, v := range d.recoveryLeases {
		leasesCopy[k] = new(*v)
	}

	receiptsCopy := make(map[string]*receipt, len(d.receipts))
	for k, v := range d.receipts {
		receiptsCopy[k] = new(*v)
	}

	return &transactionSnapshot{
		recoveryLeases: leasesCopy,
		receipts:       receiptsCopy,
	}
}

// restoreFromSnapshot reverts non-cache state to the given
// snapshot and rebuilds all secondary indexes from the
// current cache contents. The index rebuild is O(n) over
// all cached tasks, acceptable because rollback is the
// error path and a full rebuild is simpler and safer than
// per-entry undo for volatile indexes.
//
// Takes snap (*transactionSnapshot) which contains the
// state to restore.
func (d *DAL) restoreFromSnapshot(snap *transactionSnapshot) {
	d.recoveryLeases = snap.recoveryLeases
	d.receipts = snap.receipts

	d.scheduledIndex = provider_otter.NewSortedIndex[string]()
	d.executeIndex = provider_otter.NewSortedIndex[string]()
	d.workflowIndex = provider_otter.NewTagIndex[string]()
	d.dedupIndex = make(map[string]string)
	for _, task := range d.tasks.All() {
		d.indexTaskLocked(task)
	}

	d.receiptsByWorkflow = provider_otter.NewTagIndex[string]()
	d.receiptsByNode = provider_otter.NewTagIndex[string]()
	for id, r := range d.receipts {
		d.receiptsByWorkflow.AddSingle(r.workflowID, id)
		d.receiptsByNode.AddSingle(r.nodeID, id)
	}
}

// createTasksLocked inserts a batch of tasks without
// acquiring the lock. Caller must hold mu.
//
// Takes tasks ([]*orchestrator_domain.Task) which contains
// the tasks to store.
//
// Returns error when any task in the batch cannot be saved.
func (d *DAL) createTasksLocked(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	for _, task := range tasks {
		if err := d.createTaskLocked(ctx, task); err != nil {
			return fmt.Errorf("creating task %q in batch: %w", task.ID, err)
		}
	}
	return nil
}

// updateTaskLocked updates an existing task without
// acquiring the lock. Caller must hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task
// to update.
//
// Returns error when the update fails.
func (d *DAL) updateTaskLocked(ctx context.Context, task *orchestrator_domain.Task) error {
	if old, found, _ := d.tasks.GetIfPresent(ctx, task.ID); found {
		d.unindexTaskLocked(old)
	}

	if err := d.tasks.Set(ctx, task.ID, task); err != nil {
		return fmt.Errorf("setting task %q: %w", task.ID, err)
	}

	d.indexTaskLocked(task)

	return nil
}

// fetchAndMarkDueTasksLocked fetches due tasks and marks
// them as processing without acquiring the lock. Caller
// must hold mu.
//
// Takes priority (TaskPriority) which filters tasks by
// their priority level.
// Takes limit (int) which sets the maximum number of tasks
// to fetch.
//
// Returns []*Task which contains the tasks now marked as
// processing.
// Returns error when the fetch or update fails.
func (d *DAL) fetchAndMarkDueTasksLocked(ctx context.Context, priority orchestrator_domain.TaskPriority, limit int) ([]*orchestrator_domain.Task, error) {
	now := time.Now()

	candidateIDs := d.executeIndex.KeysLessThanOrEqual(now, true)

	results := make([]*orchestrator_domain.Task, 0, min(len(candidateIDs), limit))
	for _, taskID := range candidateIDs {
		if len(results) >= limit {
			break
		}

		task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
		if !found {
			continue
		}

		if task.Status != orchestrator_domain.StatusPending && task.Status != orchestrator_domain.StatusRetrying {
			continue
		}

		if task.Config.Priority != priority {
			continue
		}

		d.unindexTaskLocked(task)
		task.Status = orchestrator_domain.StatusProcessing
		task.UpdatedAt = now
		d.indexTaskLocked(task)

		results = append(results, task)
	}

	return results, nil
}

// getWorkflowStatusLocked checks workflow completion
// without acquiring the lock. Caller must hold mu.
//
// Takes workflowID (string) which identifies the workflow
// to check.
//
// Returns bool which is true when all tasks are done or
// failed.
// Returns error when the workflow cannot be found.
func (d *DAL) getWorkflowStatusLocked(ctx context.Context, workflowID string) (bool, error) {
	taskIDs := d.workflowIndex.Get(workflowID)
	if len(taskIDs) == 0 {
		return false, fmt.Errorf("workflow %q not found", workflowID)
	}

	for taskID := range taskIDs {
		task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
		if !found {
			continue
		}

		if task.Status != orchestrator_domain.StatusComplete && task.Status != orchestrator_domain.StatusFailed {
			return false, nil
		}
	}

	return true, nil
}

// promoteScheduledTasksLocked promotes scheduled tasks
// without acquiring the lock. Caller must hold mu.
//
// Returns int which is the number of tasks promoted.
// Returns error when the promotion fails.
func (d *DAL) promoteScheduledTasksLocked(ctx context.Context) (int, error) {
	now := time.Now()
	candidateIDs := d.scheduledIndex.KeysLessThanOrEqual(now, true)

	count := 0
	for _, taskID := range candidateIDs {
		task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
		if !found {
			continue
		}

		if task.Status != orchestrator_domain.StatusScheduled {
			continue
		}

		d.unindexTaskLocked(task)
		task.Status = orchestrator_domain.StatusPending
		task.ExecuteAt = now
		task.UpdatedAt = now
		d.indexTaskLocked(task)

		count++
	}

	return count, nil
}

// pendingTaskCountLocked counts pending tasks without
// acquiring the lock. Caller must hold mu.
//
// Returns int64 which is the count of pending tasks.
// Returns error when the count cannot be retrieved.
func (d *DAL) pendingTaskCountLocked(_ context.Context) (int64, error) {
	var count int64
	for _, task := range d.tasks.All() {
		if task.Status == orchestrator_domain.StatusPending {
			count++
		}
	}
	return count, nil
}

// createTaskWithDedupLocked creates a task with
// deduplication without acquiring the lock. Caller must
// hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task
// to create.
//
// Returns error when the task cannot be created or a
// duplicate exists.
func (d *DAL) createTaskWithDedupLocked(ctx context.Context, task *orchestrator_domain.Task) error {
	if task.DeduplicationKey != "" {
		if existingID, ok := d.dedupIndex[task.DeduplicationKey]; ok {
			if existing, found, _ := d.tasks.GetIfPresent(ctx, existingID); found && isActiveStatus(existing.Status) {
				return orchestrator_domain.ErrDuplicateTask
			}
		}
	}

	return d.createTaskLocked(ctx, task)
}

// recoverStaleTasksLocked resets stale PROCESSING tasks
// without acquiring the lock. Caller must hold mu.
//
// Takes staleThreshold (time.Duration) which defines how
// long a task can be in PROCESSING before being considered
// stuck.
// Takes maxRetries (int) which is the maximum retry
// attempts before marking FAILED.
// Takes recoveryError (string) which is the error message
// to record on recovered tasks.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery operation fails.
func (d *DAL) recoverStaleTasksLocked(_ context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	now := time.Now()
	cutoff := now.Add(-staleThreshold)

	count := 0
	for _, task := range d.tasks.All() {
		if task.Status != orchestrator_domain.StatusProcessing {
			continue
		}

		if task.UpdatedAt.After(cutoff) {
			continue
		}

		if lease, ok := d.recoveryLeases[task.ID]; ok && lease.expiresAt.After(now) {
			continue
		}

		d.unindexTaskLocked(task)

		task.LastError = recoveryError
		task.UpdatedAt = now
		task.Attempt++

		if task.Attempt >= maxRetries {
			task.Status = orchestrator_domain.StatusFailed
		} else {
			task.Status = orchestrator_domain.StatusRetrying
			task.ExecuteAt = now
		}

		d.indexTaskLocked(task)
		count++
	}

	return count, nil
}

// getStaleProcessingTaskCountLocked counts stale processing
// tasks without acquiring the lock. Caller must hold mu.
//
// Takes staleThreshold (time.Duration) which defines when a
// PROCESSING task is considered stuck.
//
// Returns int64 which is the count of stale tasks.
// Returns error when the count cannot be retrieved.
func (d *DAL) getStaleProcessingTaskCountLocked(_ context.Context, staleThreshold time.Duration) (int64, error) {
	cutoff := time.Now().Add(-staleThreshold)

	var count int64
	for _, task := range d.tasks.All() {
		if task.Status == orchestrator_domain.StatusProcessing && task.UpdatedAt.Before(cutoff) {
			count++
		}
	}
	return count, nil
}

// updateTaskHeartbeatLocked updates the task heartbeat
// without acquiring the lock. Caller must hold mu.
//
// Takes taskID (string) which identifies the task to
// update.
//
// Returns error when the task is not found or is not in
// PROCESSING status.
func (d *DAL) updateTaskHeartbeatLocked(ctx context.Context, taskID string) error {
	task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
	if !found {
		return fmt.Errorf("task %q not found", taskID)
	}

	if task.Status != orchestrator_domain.StatusProcessing {
		return fmt.Errorf("task %q is not in PROCESSING status", taskID)
	}

	task.UpdatedAt = time.Now()
	return nil
}

// claimStaleTasksForRecoveryLocked claims stale tasks for
// recovery without acquiring the lock. Caller must hold mu.
//
// Takes nodeID (string) which identifies the node claiming
// the tasks.
// Takes staleThreshold (time.Duration) which defines when a
// task is considered stale.
// Takes leaseTimeout (time.Duration) which sets how long
// the claim is valid.
// Takes batchLimit (int) which limits the number of tasks
// to claim per call.
//
// Returns []RecoveryClaimedTask which contains the claimed
// tasks.
// Returns error when the claim operation fails.
func (d *DAL) claimStaleTasksForRecoveryLocked(_ context.Context, nodeID string, staleThreshold, leaseTimeout time.Duration, batchLimit int) ([]orchestrator_domain.RecoveryClaimedTask, error) {
	now := time.Now()
	cutoff := now.Add(-staleThreshold)

	results := make([]orchestrator_domain.RecoveryClaimedTask, 0, batchLimit)
	for _, task := range d.tasks.All() {
		if len(results) >= batchLimit {
			break
		}

		if task.Status != orchestrator_domain.StatusProcessing {
			continue
		}

		if task.UpdatedAt.After(cutoff) {
			continue
		}

		if lease, ok := d.recoveryLeases[task.ID]; ok && lease.expiresAt.After(now) {
			continue
		}

		d.recoveryLeases[task.ID] = &recoveryLease{
			taskID:    task.ID,
			nodeID:    nodeID,
			claimedAt: now,
			expiresAt: now.Add(leaseTimeout),
		}

		results = append(results, orchestrator_domain.RecoveryClaimedTask{
			ID:         task.ID,
			WorkflowID: task.WorkflowID,
			Attempt:    safeconv.IntToInt32(task.Attempt),
		})
	}

	return results, nil
}

// recoverClaimedTasksLocked recovers claimed tasks without
// acquiring the lock. Caller must hold mu.
//
// Takes nodeID (string) which identifies the node that
// claimed the tasks.
// Takes maxRetries (int) which is the maximum retry
// attempts before marking FAILED.
// Takes recoveryError (string) which is the error message
// to record.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery fails.
func (d *DAL) recoverClaimedTasksLocked(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error) {
	now := time.Now()
	count := 0

	for taskID, lease := range d.recoveryLeases {
		if lease.nodeID != nodeID {
			continue
		}

		task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
		if !found {
			delete(d.recoveryLeases, taskID)
			continue
		}

		d.unindexTaskLocked(task)

		task.LastError = recoveryError
		task.UpdatedAt = now
		task.Attempt++

		if task.Attempt >= maxRetries {
			task.Status = orchestrator_domain.StatusFailed
		} else {
			task.Status = orchestrator_domain.StatusRetrying
			task.ExecuteAt = now
		}

		d.indexTaskLocked(task)
		delete(d.recoveryLeases, taskID)
		count++
	}

	return count, nil
}

// releaseRecoveryLeasesLocked releases recovery leases
// without acquiring the lock. Caller must hold mu.
//
// Takes nodeID (string) which identifies the node releasing
// leases.
//
// Returns int which is the count of leases released.
// Returns error when the release fails.
func (d *DAL) releaseRecoveryLeasesLocked(_ context.Context, nodeID string) (int, error) {
	count := 0
	for taskID, lease := range d.recoveryLeases {
		if lease.nodeID == nodeID {
			delete(d.recoveryLeases, taskID)
			count++
		}
	}

	return count, nil
}

// createWorkflowReceiptLocked creates a workflow receipt
// without acquiring the lock. Caller must hold mu.
//
// Takes id (string) which is the unique identifier for the
// receipt.
// Takes workflowID (string) which is the workflow being
// tracked.
// Takes nodeID (string) which is the node that created the
// receipt.
//
// Returns error when the receipt cannot be created.
func (d *DAL) createWorkflowReceiptLocked(_ context.Context, id, workflowID, nodeID string) error {
	r := &receipt{
		id:         id,
		workflowID: workflowID,
		nodeID:     nodeID,
		status:     receiptPending,
		createdAt:  time.Now(),
	}

	d.receipts[id] = r
	d.receiptsByWorkflow.AddSingle(workflowID, id)
	d.receiptsByNode.AddSingle(nodeID, id)

	return nil
}

// resolveWorkflowReceiptsLocked resolves workflow receipts
// without acquiring the lock. Caller must hold mu.
//
// Takes workflowID (string) which identifies the completed
// workflow.
// Takes errorMessage (string) which contains any error from
// workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (d *DAL) resolveWorkflowReceiptsLocked(_ context.Context, workflowID, errorMessage string) (int, error) {
	receiptIDs := d.receiptsByWorkflow.Get(workflowID)
	now := time.Now()
	count := 0

	for receiptID := range receiptIDs {
		r, ok := d.receipts[receiptID]
		if !ok {
			continue
		}

		if r.status != receiptPending {
			continue
		}

		r.status = receiptResolved
		r.errorMessage = errorMessage
		r.resolvedAt = now
		count++
	}

	return count, nil
}

// getPendingReceiptsByNodeLocked retrieves pending receipts
// by node without acquiring the lock. Caller must hold mu.
//
// Takes nodeID (string) which identifies the node.
//
// Returns []PendingReceipt which contains pending receipts.
// Returns error when the query fails.
func (d *DAL) getPendingReceiptsByNodeLocked(_ context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	receiptIDs := d.receiptsByNode.Get(nodeID)
	return d.getPendingReceiptsFromIDs(receiptIDs), nil
}

// cleanupOldResolvedReceiptsLocked cleans up resolved
// receipts without acquiring the lock. Caller must hold mu.
//
// Takes olderThan (time.Time) which is the cutoff for
// deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (d *DAL) cleanupOldResolvedReceiptsLocked(_ context.Context, olderThan time.Time) (int, error) {
	count := 0
	for id, r := range d.receipts {
		if r.status != receiptResolved {
			continue
		}

		if r.resolvedAt.Before(olderThan) {
			d.receiptsByWorkflow.RemoveSingle(r.workflowID, id)
			d.receiptsByNode.RemoveSingle(r.nodeID, id)
			delete(d.receipts, id)
			count++
		}
	}

	return count, nil
}

// timeoutStaleReceiptsLocked times out stale receipts
// without acquiring the lock. Caller must hold mu.
//
// Takes olderThan (time.Time) which is the cutoff for
// timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
func (d *DAL) timeoutStaleReceiptsLocked(_ context.Context, olderThan time.Time) (int, error) {
	now := time.Now()
	count := 0

	for _, r := range d.receipts {
		if r.status != receiptPending {
			continue
		}

		if r.createdAt.Before(olderThan) {
			r.status = receiptTimedOut
			r.resolvedAt = now
			count++
		}
	}

	return count, nil
}

// listFailedTasksLocked returns failed tasks without
// acquiring the lock. Caller must hold mu.
//
// Returns []*Task which contains the failed tasks.
// Returns error (always nil for the in-memory store).
func (d *DAL) listFailedTasksLocked(_ context.Context) ([]*orchestrator_domain.Task, error) {
	var result []*orchestrator_domain.Task
	for _, task := range d.tasks.All() {
		if task.Status == orchestrator_domain.StatusFailed {
			result = append(result, new(*task))
		}
	}
	return result, nil
}

// createTaskLocked stores a task without acquiring the lock.
// Caller must hold mu.
//
// Takes ctx (context.Context) for cancellation and timeout.
// Takes task (*orchestrator_domain.Task) which is the task to store.
//
// Returns error when the storage operation fails.
func (d *DAL) createTaskLocked(ctx context.Context, task *orchestrator_domain.Task) error {
	if err := d.tasks.Set(ctx, task.ID, task); err != nil {
		return fmt.Errorf("setting task %q: %w", task.ID, err)
	}

	d.indexTaskLocked(task)

	return nil
}

// indexTaskLocked updates indexes for a task. Caller must hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task to index.
func (d *DAL) indexTaskLocked(task *orchestrator_domain.Task) {
	d.workflowIndex.AddSingle(task.WorkflowID, task.ID)

	if task.Status == orchestrator_domain.StatusScheduled {
		d.scheduledIndex.Add(task.ID, task.ScheduledExecuteAt)
	}

	if task.Status == orchestrator_domain.StatusPending || task.Status == orchestrator_domain.StatusRetrying {
		d.executeIndex.Add(task.ID, task.ExecuteAt)
	}

	if task.DeduplicationKey != "" && isActiveStatus(task.Status) {
		d.dedupIndex[task.DeduplicationKey] = task.ID
	}
}

// workflowAgg holds aggregate statistics for a single workflow.
type workflowAgg struct {
	// taskCount int64 // taskCount is the total number of tasks in the workflow.
	taskCount int64

	// completeCount is the number of tasks with status complete.
	completeCount int64

	// failedCount is the number of tasks with failed status.
	failedCount int64

	// activeCount is the number of workflows in pending or processing state.
	activeCount int64

	// createdAt is the earliest task creation time as a Unix timestamp.
	createdAt int64

	// updatedAt is the most recent update timestamp in Unix seconds.
	updatedAt int64
}

// unindexTaskLocked removes indexes for a task. Caller must hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task to remove from all
// indexes.
func (d *DAL) unindexTaskLocked(task *orchestrator_domain.Task) {
	d.workflowIndex.RemoveSingle(task.WorkflowID, task.ID)

	d.scheduledIndex.Remove(task.ID)
	d.executeIndex.Remove(task.ID)

	if task.DeduplicationKey != "" {
		if existing, ok := d.dedupIndex[task.DeduplicationKey]; ok && existing == task.ID {
			delete(d.dedupIndex, task.DeduplicationKey)
		}
	}
}

// getPendingReceiptsFromIDs retrieves pending receipts matching the given IDs.
//
// Takes receiptIDs (map[string]struct{}) which specifies the receipt IDs to
// look up.
//
// Returns []orchestrator_domain.PendingReceipt which contains the pending
// receipts found. Receipts that do not exist or are not pending are skipped.
func (d *DAL) getPendingReceiptsFromIDs(receiptIDs map[string]struct{}) []orchestrator_domain.PendingReceipt {
	results := make([]orchestrator_domain.PendingReceipt, 0, len(receiptIDs))

	for receiptID := range receiptIDs {
		r, ok := d.receipts[receiptID]
		if !ok || r.status != receiptPending {
			continue
		}

		results = append(results, orchestrator_domain.PendingReceipt{
			ID:         r.id,
			WorkflowID: r.workflowID,
			NodeID:     r.nodeID,
			CreatedAt:  r.createdAt.Unix(),
		})
	}

	return results
}

// aggregateWorkflowData groups all tasks by workflow ID and computes
// aggregate counts and timestamps per workflow.
//
// Returns map[string]*workflowAgg which maps workflow IDs to their aggregated
// task statistics.
func (d *DAL) aggregateWorkflowData() map[string]*workflowAgg {
	workflows := make(map[string]*workflowAgg)

	for _, task := range d.tasks.All() {
		agg, ok := workflows[task.WorkflowID]
		if !ok {
			agg = &workflowAgg{
				createdAt: task.CreatedAt.Unix(),
				updatedAt: task.UpdatedAt.Unix(),
			}
			workflows[task.WorkflowID] = agg
		}

		agg.taskCount++
		if task.CreatedAt.Unix() < agg.createdAt {
			agg.createdAt = task.CreatedAt.Unix()
		}
		if task.UpdatedAt.Unix() > agg.updatedAt {
			agg.updatedAt = task.UpdatedAt.Unix()
		}

		switch task.Status {
		case orchestrator_domain.StatusComplete:
			agg.completeCount++
		case orchestrator_domain.StatusFailed:
			agg.failedCount++
		case orchestrator_domain.StatusPending, orchestrator_domain.StatusProcessing, orchestrator_domain.StatusRetrying:
			agg.activeCount++
		}
	}

	return workflows
}

// WithCache injects an externally configured cache instance.
//
// This enables WAL persistence when the cache is created with
// PersistenceConfig. When provided, the DAL will not close the cache on
// shutdown - the caller is responsible for cache lifecycle management.
//
// Takes cache (cache_domain.ProviderPort) which is the cache instance to use.
//
// Returns Option which configures the DAL to use the provided cache.
func WithCache(cache cache_domain.ProviderPort[string, *orchestrator_domain.Task]) Option {
	return func(d *DAL) {
		d.tasks = cache
		d.ownsCache = false
	}
}

// NewOtterDAL creates a new in-memory orchestrator DAL using otter cache.
//
// Takes config (Config) which specifies cache settings.
// Takes opts (...Option) which configures optional features like cache injection.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is the configured DAL.
// Returns error when the cache cannot be created.
func NewOtterDAL(config Config, opts ...Option) (orchestrator_dal.OrchestratorDALWithTx, error) {
	dal := &DAL{
		scheduledIndex:     provider_otter.NewSortedIndex[string](),
		executeIndex:       provider_otter.NewSortedIndex[string](),
		workflowIndex:      provider_otter.NewTagIndex[string](),
		dedupIndex:         make(map[string]string),
		recoveryLeases:     make(map[string]*recoveryLease),
		receipts:           make(map[string]*receipt),
		receiptsByWorkflow: provider_otter.NewTagIndex[string](),
		receiptsByNode:     provider_otter.NewTagIndex[string](),
		ownsCache:          true,
		mu:                 sync.RWMutex{},
	}

	for _, opt := range opts {
		opt(dal)
	}

	if dal.tasks == nil {
		capacity := config.Capacity
		if capacity <= 0 {
			capacity = defaultCacheCapacity
		}

		cacheOpts := cache_dto.Options[string, *orchestrator_domain.Task]{
			MaximumSize: int(capacity),
		}

		cache, err := provider_otter.OtterProviderFactory(cacheOpts)
		if err != nil {
			return nil, fmt.Errorf("creating otter cache: %w", err)
		}
		dal.tasks = cache
	}

	return dal, nil
}

// otterTransactionDAL is a transaction-scoped TaskStore that delegates to
// the parent DAL's locked methods. The parent's mutex is already held by
// RunAtomic, so these methods skip lock acquisition.
type otterTransactionDAL struct {
	// parent is the owning DAL whose locked methods are
	// called by this transaction store.
	parent *DAL
}

// CreateTask saves a new task within the current
// transaction.
//
// Takes task (*orchestrator_domain.Task) which is the task
// to save.
//
// Returns error when the task cannot be saved.
func (tx *otterTransactionDAL) CreateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	return tx.parent.createTaskLocked(ctx, task)
}

// CreateTasks inserts a batch of tasks within the current
// transaction.
//
// Takes tasks ([]*orchestrator_domain.Task) which contains
// the tasks to store.
//
// Returns error when the database operation fails.
func (tx *otterTransactionDAL) CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	return tx.parent.createTasksLocked(ctx, tasks)
}

// UpdateTask updates an existing task within the current
// transaction.
//
// Takes task (*orchestrator_domain.Task) which is the task
// to update.
//
// Returns error when the update fails.
func (tx *otterTransactionDAL) UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	return tx.parent.updateTaskLocked(ctx, task)
}

// FetchAndMarkDueTasks fetches due tasks and marks them as
// processing within the current transaction.
//
// Takes priority (TaskPriority) which filters tasks by
// their priority level.
// Takes limit (int) which sets the maximum number of tasks
// to fetch.
//
// Returns []*Task which contains the tasks now marked as
// processing.
// Returns error when the fetch or update fails.
func (tx *otterTransactionDAL) FetchAndMarkDueTasks(ctx context.Context, priority orchestrator_domain.TaskPriority, limit int) ([]*orchestrator_domain.Task, error) {
	return tx.parent.fetchAndMarkDueTasksLocked(ctx, priority, limit)
}

// GetWorkflowStatus checks whether all tasks in a workflow
// are complete within the current transaction.
//
// Takes workflowID (string) which identifies the workflow
// to check.
//
// Returns bool which is true when all tasks are done or
// failed.
// Returns error when the workflow cannot be found.
func (tx *otterTransactionDAL) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	return tx.parent.getWorkflowStatusLocked(ctx, workflowID)
}

// PromoteScheduledTasks moves scheduled tasks that are now
// due to pending status within the current transaction.
//
// Returns int which is the number of tasks promoted.
// Returns error when the promotion fails.
func (tx *otterTransactionDAL) PromoteScheduledTasks(ctx context.Context) (int, error) {
	return tx.parent.promoteScheduledTasksLocked(ctx)
}

// PendingTaskCount returns the number of tasks waiting to
// be run within the current transaction.
//
// Returns int64 which is the count of pending tasks.
// Returns error when the count cannot be retrieved.
func (tx *otterTransactionDAL) PendingTaskCount(ctx context.Context) (int64, error) {
	return tx.parent.pendingTaskCountLocked(ctx)
}

// CreateTaskWithDedup creates a task with deduplication
// support within the current transaction.
//
// Takes task (*orchestrator_domain.Task) which is the task
// to create.
//
// Returns error when the task cannot be created or a
// duplicate exists.
func (tx *otterTransactionDAL) CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error {
	return tx.parent.createTaskWithDedupLocked(ctx, task)
}

// RecoverStaleTasks resets PROCESSING tasks that have
// exceeded the stale threshold within the current
// transaction.
//
// Takes staleThreshold (time.Duration) which defines how
// long a task can be in PROCESSING before being considered
// stuck.
// Takes maxRetries (int) which is the maximum retry
// attempts before marking FAILED.
// Takes recoveryError (string) which is the error message
// to record on recovered tasks.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery operation fails.
func (tx *otterTransactionDAL) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	return tx.parent.recoverStaleTasksLocked(ctx, staleThreshold, maxRetries, recoveryError)
}

// GetStaleProcessingTaskCount returns the count of tasks
// stuck in PROCESSING longer than the threshold within the
// current transaction.
//
// Takes staleThreshold (time.Duration) which defines when a
// PROCESSING task is considered stuck.
//
// Returns int64 which is the count of stale tasks.
// Returns error when the count cannot be retrieved.
func (tx *otterTransactionDAL) GetStaleProcessingTaskCount(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	return tx.parent.getStaleProcessingTaskCountLocked(ctx, staleThreshold)
}

// UpdateTaskHeartbeat updates the updated_at timestamp for
// a task within the current transaction.
//
// Takes taskID (string) which identifies the task to
// update.
//
// Returns error when the task is not found or is not in
// PROCESSING status.
func (tx *otterTransactionDAL) UpdateTaskHeartbeat(ctx context.Context, taskID string) error {
	return tx.parent.updateTaskHeartbeatLocked(ctx, taskID)
}

// ClaimStaleTasksForRecovery atomically claims stale
// PROCESSING tasks for recovery within the current
// transaction.
//
// Takes nodeID (string) which identifies the node claiming
// the tasks.
// Takes staleThreshold (time.Duration) which defines when a
// task is considered stale.
// Takes leaseTimeout (time.Duration) which sets how long
// the claim is valid.
// Takes batchLimit (int) which limits the number of tasks
// to claim per call.
//
// Returns []RecoveryClaimedTask which contains the claimed
// tasks.
// Returns error when the claim operation fails.
func (tx *otterTransactionDAL) ClaimStaleTasksForRecovery(
	ctx context.Context,
	nodeID string,
	staleThreshold, leaseTimeout time.Duration,
	batchLimit int,
) ([]orchestrator_domain.RecoveryClaimedTask, error) {
	return tx.parent.claimStaleTasksForRecoveryLocked(ctx, nodeID, staleThreshold, leaseTimeout, batchLimit)
}

// RecoverClaimedTasks recovers all tasks previously claimed
// by this node within the current transaction.
//
// Takes nodeID (string) which identifies the node that
// claimed the tasks.
// Takes maxRetries (int) which is the maximum retry
// attempts before marking FAILED.
// Takes recoveryError (string) which is the error message
// to record.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery fails.
func (tx *otterTransactionDAL) RecoverClaimedTasks(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error) {
	return tx.parent.recoverClaimedTasksLocked(ctx, nodeID, maxRetries, recoveryError)
}

// ReleaseRecoveryLeases releases all recovery leases held
// by this node within the current transaction.
//
// Takes nodeID (string) which identifies the node releasing
// leases.
//
// Returns int which is the count of leases released.
// Returns error when the release fails.
func (tx *otterTransactionDAL) ReleaseRecoveryLeases(ctx context.Context, nodeID string) (int, error) {
	return tx.parent.releaseRecoveryLeasesLocked(ctx, nodeID)
}

// CreateWorkflowReceipt creates a new workflow receipt for
// tracking completion within the current transaction.
//
// Takes id (string) which is the unique identifier for the
// receipt.
// Takes workflowID (string) which is the workflow being
// tracked.
// Takes nodeID (string) which is the node that created the
// receipt.
//
// Returns error when the receipt cannot be created.
func (tx *otterTransactionDAL) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	return tx.parent.createWorkflowReceiptLocked(ctx, id, workflowID, nodeID)
}

// ResolveWorkflowReceipts marks all pending receipts for a
// workflow as resolved within the current transaction.
//
// Takes workflowID (string) which identifies the completed
// workflow.
// Takes errorMessage (string) which contains any error from
// workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (tx *otterTransactionDAL) ResolveWorkflowReceipts(ctx context.Context, workflowID, errorMessage string) (int, error) {
	return tx.parent.resolveWorkflowReceiptsLocked(ctx, workflowID, errorMessage)
}

// GetPendingReceiptsByNode retrieves all pending receipts
// created by a node within the current transaction.
//
// Takes nodeID (string) which identifies the node.
//
// Returns []PendingReceipt which contains pending receipts.
// Returns error when the query fails.
func (tx *otterTransactionDAL) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	return tx.parent.getPendingReceiptsByNodeLocked(ctx, nodeID)
}

// CleanupOldResolvedReceipts deletes resolved receipts
// older than the specified time within the current
// transaction.
//
// Takes olderThan (time.Time) which is the cutoff for
// deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (tx *otterTransactionDAL) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	return tx.parent.cleanupOldResolvedReceiptsLocked(ctx, olderThan)
}

// TimeoutStaleReceipts marks very old pending receipts as
// timed out within the current transaction.
//
// Takes olderThan (time.Time) which is the cutoff for
// timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
func (tx *otterTransactionDAL) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	return tx.parent.timeoutStaleReceiptsLocked(ctx, olderThan)
}

// ListFailedTasks returns all tasks with a FAILED status
// within the current transaction.
//
// Returns []*Task which contains the failed tasks.
// Returns error (always nil for the in-memory store).
func (tx *otterTransactionDAL) ListFailedTasks(ctx context.Context) ([]*orchestrator_domain.Task, error) {
	return tx.parent.listFailedTasksLocked(ctx)
}

// RunAtomic rejects nested transactions.
//
// Returns error which is always
// ErrNestedTransactionUnsupported.
func (*otterTransactionDAL) RunAtomic(_ context.Context, _ func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	return cache_domain.ErrNestedTransactionUnsupported
}

// isActiveStatus returns true if the status represents an active task.
//
// Takes status (orchestrator_domain.TaskStatus) which is the task status to
// check.
//
// Returns bool which is true when the status is scheduled, pending,
// processing, or retrying.
func isActiveStatus(status orchestrator_domain.TaskStatus) bool {
	switch status {
	case orchestrator_domain.StatusScheduled,
		orchestrator_domain.StatusPending,
		orchestrator_domain.StatusProcessing,
		orchestrator_domain.StatusRetrying:
		return true
	}
	return false
}
