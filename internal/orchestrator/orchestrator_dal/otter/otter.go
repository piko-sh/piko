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
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
	"piko.sh/safeconv"
)

const (
	// defaultCacheCapacity is the default number of items the cache can hold.
	defaultCacheCapacity = 100_000

	// maintenanceBatchSize bounds each promotion, recovery, or cleanup batch.
	maintenanceBatchSize = 100

	// maxTransactionTimeout is the maximum duration a RunAtomic transaction may hold the
	// mutex before the context is cancelled.
	maxTransactionTimeout = 30 * time.Second

	// receiptPending is the initial status for workflow receipts queued for processing.
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

	_ orchestrator_domain.BuildCompletionStore = (*DAL)(nil)

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

	// errorMessage contains the error text when the receipt is resolved with a failure;
	// empty when resolved successfully.
	errorMessage string
}

// DAL provides in-memory storage for orchestrator tasks using otter cache. It implements
// OrchestratorDALWithTx and OrchestratorInspector.
type DAL struct {
	// clock supplies the time for promotion, recovery, heartbeats, receipts, and due-task
	// queries; it defaults to the real clock.
	clock clockpkg.Clock

	// processingIndex orders processing tasks by heartbeat update time for recovery.
	processingIndex *timeIndex

	// terminalIndex orders completed and failed tasks by update time for reclamation.
	terminalIndex *timeIndex

	// resolvedIndex orders resolved and timed-out receipts by resolution time for cleanup.
	resolvedIndex *timeIndex

	// journal records original states during a transaction; the DAL write lock guards
	// access.
	journal *mutationJournal

	// tasks is the main cache for tasks, keyed by task ID. Uses the cache hexagon's
	// ProviderPort for optional WAL persistence.
	tasks cache_domain.ProviderPort[string, *orchestrator_domain.Task]

	// scheduledIndex maps task IDs to their scheduled execution times. Supports range
	// queries to find tasks ready to run.
	scheduledIndex *timeIndex

	// executeIndex stores pending and retrying tasks sorted by their ExecuteAt time. Used
	// for efficient time-based range queries to find tasks ready to run.
	executeIndex *timeIndex

	// workflowIndex maps workflow IDs to their task IDs.
	workflowIndex *workflowIndex

	// dedupIndex maps deduplication keys to task IDs for detecting duplicate tasks.
	dedupIndex map[string]string

	// unsettledByTag counts unsettled foreground tasks by build tag, using the empty tag for
	// untagged tasks.
	unsettledByTag map[string]int64

	// degraded holds why the cache diverged from its log, once it has; writes then fail with
	// ErrStoreDegraded.
	degraded atomic.Pointer[error]

	// countedTag maps counted task IDs to their unsettledByTag keys.
	countedTag map[string]string

	// receipts maps receipt IDs to workflow receipts.
	receipts map[string]*receipt

	// receiptsByWorkflow maps workflow IDs to their receipt IDs for quick lookup.
	receiptsByWorkflow *provider_otter.TagIndex[string]

	// receiptsByNode maps node IDs to their receipt IDs for quick lookup.
	receiptsByNode *provider_otter.TagIndex[string]

	// nodes holds in-memory node leases protected by nodesMu.
	nodes map[string]orchestrator_domain.NodeLease

	// publishedAt holds in-memory task publication times protected by publishedMu.
	publishedAt map[string]time.Time

	// dangling holds missing task IDs for pruning by the next non-transactional write,
	// protected by danglingMu.
	dangling map[string]struct{}

	// pendingWaits holds log-write waits collected under mu and completed after unlockWrite
	// releases it.
	pendingWaits []func() error

	// capacity limits task storage and pending receipt admission.
	capacity int

	// ownsCache indicates whether this DAL owns the cache and should close it. When false,
	// the cache was injected externally and the caller handles cleanup.
	ownsCache bool

	// danglingPending reports whether dangling holds IDs, so writes check it without taking
	// danglingMu.
	danglingPending atomic.Bool

	// danglingMu guards dangling, which readers holding only the read lock add to.
	danglingMu sync.Mutex

	// nodesMu guards nodes.
	nodesMu sync.Mutex

	// publishedMu guards publishedAt.
	publishedMu sync.Mutex

	// mu guards concurrent access to indexes and maps.
	mu sync.RWMutex
}

// Config holds settings for the otter-based orchestrator DAL.
type Config struct {
	// Capacity is the maximum number of items to store. Defaults to 100,000 if zero or
	// negative.
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

// LocalStore reports that the store's writes complete in this process, so callers need
// not bound them with a timeout.
//
// Returns bool which is always true.
func (*DAL) LocalStore() bool {
	return true
}

// Close releases resources held by the DAL.
//
// Returns error when the cache cannot be closed cleanly.
//
// Safe for concurrent use; waits for any transaction in progress, which swaps the task
// cache while it runs.
func (d *DAL) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ownsCache {
		return d.tasks.Close(context.Background())
	}
	return nil
}

// RunAtomic executes fn in a transaction and rolls back its changes if it fails or
// panics.
//
// The DAL write lock serialises the transaction. The callback must finish within
// maxTransactionTimeout, respect its context, and avoid blocking I/O.
//
// Takes fn (func) which receives a transactional TaskStore.
//
// Returns error when fn returns an error or the transaction fails to commit.
//
// Panics if fn panics. The transaction is rolled back before the panic is re-raised.
func (d *DAL) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return err
	}

	journal := &mutationJournal{}
	d.journal = journal
	defer func() { d.journal = nil }()
	realCache := d.tasks
	txCache := cache_domain.BeginTransaction(ctx, realCache)
	d.tasks = txCache
	defer func() { d.tasks = realCache }()

	rollbackCtx := context.WithoutCancel(ctx)

	rollback := func() {
		if rollbackErr := txCache.Rollback(rollbackCtx); rollbackErr != nil {
			d.markDegraded(fmt.Errorf("undoing a failed transaction: %w", rollbackErr))
		}
		d.tasks = realCache
		d.restoreJournal(rollbackCtx, journal)
	}

	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				rollback()
				panic(r)
			}
		}()
		if err = ctx.Err(); err == nil {
			err = fn(ctx, &otterTransactionDAL{parent: d})
		}
		if err == nil {
			err = ctx.Err()
		}
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
// The DAL write lock serialises this operation.
//
// Takes task (*orchestrator_domain.Task) which is the task to save.
//
// Returns error when the task cannot be saved.
func (d *DAL) CreateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return err
	}

	return d.createTaskLocked(ctx, task)
}

// CreateTasks inserts a batch of tasks.
//
// The DAL write lock serialises this operation.
//
// Takes tasks ([]*orchestrator_domain.Task) which contains the tasks to store.
//
// Returns error when the database operation fails.
func (d *DAL) CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return err
	}
	return d.createTasksLocked(ctx, tasks)
}

// UpdateTask updates an existing task.
//
// The DAL write lock serialises this operation.
//
// Takes task (*orchestrator_domain.Task) which is the task to update.
//
// Returns error when the update fails.
func (d *DAL) UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return err
	}
	return d.updateTaskLocked(ctx, task)
}

// GetTasksByID reads the tasks with the given IDs without changing them.
//
// Takes ids ([]string) which lists the task IDs to read.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks found.
// Returns error which is always nil.
//
// Safe for concurrent use; holds a read lock during the lookup.
func (d *DAL) GetTasksByID(ctx context.Context, ids []string) ([]*orchestrator_domain.Task, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getTasksByIDLocked(ctx, ids)
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
// The DAL write lock serialises this operation.
//
// Returns int which is the number of tasks promoted.
// Returns error when the promotion fails.
func (d *DAL) PromoteScheduledTasks(ctx context.Context) (int, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return 0, err
	}
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
// The DAL write lock serialises this operation.
//
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the task cannot be created or a duplicate exists.
func (d *DAL) CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return err
	}
	return d.createTaskWithDedupLocked(ctx, task)
}

// CreateWorkflowReceipt creates a new workflow receipt for tracking completion.
//
// The DAL write lock serialises this operation.
//
// Takes id (string) which is the unique identifier for the receipt.
// Takes workflowID (string) which is the workflow being tracked.
// Takes nodeID (string) which is the node that created the receipt.
//
// Returns error when the receipt cannot be created.
func (d *DAL) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return err
	}
	return d.createWorkflowReceiptLocked(ctx, id, workflowID, nodeID)
}

// ResolveWorkflowReceipts marks all pending receipts for a workflow as resolved.
//
// The DAL write lock serialises this operation.
//
// Takes workflowID (string) which identifies the completed workflow.
// Takes errorMessage (string) which contains any error from workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (d *DAL) ResolveWorkflowReceipts(ctx context.Context, workflowID, errorMessage string) (int, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return 0, err
	}
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

// CleanupOldResolvedReceipts deletes resolved receipts older than the specified time.
//
// The DAL write lock serialises this operation.
//
// Takes olderThan (time.Time) which is the cutoff for deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (d *DAL) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return 0, err
	}
	return d.cleanupOldResolvedReceiptsLocked(ctx, olderThan)
}

// TimeoutStaleReceipts marks very old pending receipts as timed out.
//
// The DAL write lock serialises this operation.
//
// Takes olderThan (time.Time) which is the cutoff for timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
func (d *DAL) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	d.lockWrite()
	defer d.unlockWrite()

	if err := d.writable(); err != nil {
		return 0, err
	}
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
// Returns []orchestrator_domain.TaskSummary which contains the count for each status.
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
// Takes limit (int) which specifies the maximum number of tasks to return.
//
// Returns []orchestrator_domain.TaskListItem which contains the task data for display.
// Returns error when the query fails.
//
// Safe for concurrent use. Snapshots the task slice under a read lock, releases the lock,
// then sorts outside it so concurrent readers are not blocked by the O(n log n) sort.
func (d *DAL) ListRecentTasks(_ context.Context, limit int) ([]orchestrator_domain.TaskListItem, error) {
	d.mu.RLock()
	tasks := make([]*orchestrator_domain.Task, 0, d.tasks.EstimatedSize())
	for _, task := range d.tasks.All() {
		tasks = append(tasks, task)
	}
	d.mu.RUnlock()

	slices.SortFunc(tasks, func(a, b *orchestrator_domain.Task) int {
		return cmp.Compare(b.UpdatedAt.UnixNano(), a.UpdatedAt.UnixNano())
	})

	count := min(limit, len(tasks))
	results := make([]orchestrator_domain.TaskListItem, count)
	for i := range count {
		task := tasks[i]
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
// Takes limit (int) which specifies the maximum number of workflows to return.
//
// Returns []orchestrator_domain.WorkflowSummary which contains aggregated workflow data
// sorted by most recently updated.
// Returns error when the query fails.
//
// Safe for concurrent use; holds a read lock for the duration of the call.
func (d *DAL) ListWorkflowSummary(_ context.Context, limit int) ([]orchestrator_domain.WorkflowSummary, error) {
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

	count := min(limit, len(sorted))
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

// RebuildIndexes rebuilds all secondary indexes from the primary cache data. Call this
// after WAL recovery to restore scheduledIndex, executeIndex, workflowIndex, and
// dedupIndex.
//
// Safe for concurrent use; holds the mutex for the entire operation.
//
// Receipts and related receipt indexes are not recovered. They are ephemeral data that
// reset on restart, tracking in-flight workflow completion notifications.
func (d *DAL) RebuildIndexes(ctx context.Context) {
	d.lockWrite()
	defer d.unlockWrite()

	d.processingIndex = newTimeIndex()
	d.terminalIndex = newTimeIndex()
	d.resolvedIndex = newTimeIndex()
	d.scheduledIndex = newTimeIndex()
	d.executeIndex = newTimeIndex()
	d.workflowIndex = newWorkflowIndex()
	d.dedupIndex = make(map[string]string)
	d.unsettledByTag = make(map[string]int64)
	d.countedTag = make(map[string]string)
	d.nodesMu.Lock()
	d.nodes = make(map[string]orchestrator_domain.NodeLease)
	d.nodesMu.Unlock()
	d.publishedMu.Lock()
	d.publishedAt = make(map[string]time.Time)
	d.publishedMu.Unlock()
	d.receipts = make(map[string]*receipt)
	d.receiptsByWorkflow = provider_otter.NewTagIndex[string]()
	d.receiptsByNode = provider_otter.NewTagIndex[string]()

	_, repairLog := logger_domain.From(ctx, log)
	for id, task := range d.tasks.All() {
		if task.Status == orchestrator_domain.StatusScheduled && task.ScheduledExecuteAt.IsZero() {
			task = cloneTask(task)
			task.ScheduledExecuteAt = task.ExecuteAt.UTC()
			if err := d.tasks.Set(ctx, id, task); err != nil {
				repairLog.Warn("Failed to persist repaired schedule", logger_domain.Error(err), logger_domain.String("task_id", id))
			}
		}
		d.indexTaskLocked(task)
	}

	_, rl := logger_domain.From(ctx, log)
	rl.Internal("Orchestrator indexes rebuilt from cache",
		logger_domain.Int("task_count", d.tasks.EstimatedSize()))
}

// createTasksLocked inserts a batch of tasks without acquiring the lock. Caller must hold
// mu.
//
// Takes tasks ([]*orchestrator_domain.Task) which contains the tasks to store.
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

// updateTaskLocked stores an independent copy of an existing task while the caller holds
// mu.
//
// Takes task (*orchestrator_domain.Task) which is the task to update.
//
// Returns error which wraps ErrTaskNotFound when the task does not exist, or a cache
// failure.
func (d *DAL) updateTaskLocked(ctx context.Context, task *orchestrator_domain.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.pruneDanglingLocked(ctx)
	old, found, err := d.tasks.GetIfPresent(ctx, task.ID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("updating task %q: %w", task.ID, orchestrator_domain.ErrTaskNotFound)
	}
	return d.replaceTaskLocked(ctx, old, storedCopy(task))
}

// getTasksByIDLocked reads tasks by ID without acquiring the lock. Caller must hold mu.
//
// Takes ids ([]string) which lists the task IDs to read.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks found.
// Returns error which is always nil.
func (d *DAL) getTasksByIDLocked(ctx context.Context, ids []string) ([]*orchestrator_domain.Task, error) {
	results := make([]*orchestrator_domain.Task, 0, len(ids))
	for _, id := range ids {
		if task, found, _ := d.tasks.GetIfPresent(ctx, id); found {
			results = append(results, readCopy(task))
		}
	}
	return results, nil
}

// getWorkflowStatusLocked checks workflow completion without acquiring the lock. Caller
// must hold mu, for reading or writing.
//
// Indexed tasks missing from the cache are noted as dangling rather than removed, because
// a reader must not change the indexes.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns bool which is true when all tasks are done or failed, including when the
// workflow has no tasks.
// Returns error which is always nil.
func (d *DAL) getWorkflowStatusLocked(ctx context.Context, workflowID string) (bool, error) {
	taskIDs := d.workflowIndex.Get(workflowID)
	if len(taskIDs) == 0 {
		return true, nil
	}

	for taskID := range taskIDs {
		task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
		if !found {
			d.noteDangling(taskID)
			continue
		}

		if task.Status != orchestrator_domain.StatusComplete && task.Status != orchestrator_domain.StatusFailed {
			return false, nil
		}
	}

	return true, nil
}

// promoteScheduledTasksLocked promotes scheduled tasks without acquiring the lock. Caller
// must hold mu.
//
// Returns int which is the number of tasks promoted.
// Returns error when the promotion fails.
func (d *DAL) promoteScheduledTasksLocked(ctx context.Context) (int, error) {
	now := d.now()
	candidateIDs := d.scheduledIndex.KeysThrough(now, maintenanceBatchSize)

	count := 0
	for _, taskID := range candidateIDs {
		task, found, _ := d.tasks.GetIfPresent(ctx, taskID)
		if !found {
			d.scheduledIndex.Remove(taskID)
			d.executeIndex.Remove(taskID)
			d.workflowIndex.RemoveKey(taskID)
			continue
		}

		if task.Status != orchestrator_domain.StatusScheduled {
			continue
		}

		task = cloneTask(task)
		task.Status = orchestrator_domain.StatusPending
		task.ExecuteAt = now
		task.UpdatedAt = now
		if err := d.updateTaskLocked(ctx, task); err != nil {
			return count, err
		}

		count++
	}

	return count, nil
}

// pendingTaskCountLocked counts the tasks waiting to run, pending, retrying and
// scheduled, from the indexes that hold exactly those tasks. Caller must hold mu.
//
// Returns int64 which is the count of waiting tasks.
// Returns error when the count cannot be retrieved.
func (d *DAL) pendingTaskCountLocked(_ context.Context) (int64, error) {
	return int64(d.executeIndex.Size() + d.scheduledIndex.Size()), nil
}

// createTaskWithDedupLocked creates a task with deduplication without acquiring the lock.
// Caller must hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the task cannot be created or a duplicate exists.
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

// createWorkflowReceiptLocked creates a workflow receipt without acquiring the lock.
// Caller must hold mu.
//
// Takes id (string) which is the unique identifier for the receipt.
// Takes workflowID (string) which is the workflow being tracked.
// Takes nodeID (string) which is the node that created the receipt.
//
// Returns error when the receipt cannot be created.
func (d *DAL) createWorkflowReceiptLocked(ctx context.Context, id, workflowID, nodeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := d.receipts[id]; !exists && d.capacity > 0 && len(d.receipts) >= d.capacity {
		_, _ = d.cleanupOldResolvedReceiptsLocked(ctx, d.now().Add(time.Second))
		if len(d.receipts) >= d.capacity {
			return orchestrator_domain.ErrTaskCapacity
		}
	}
	r := &receipt{
		id:           id,
		workflowID:   workflowID,
		nodeID:       nodeID,
		status:       receiptPending,
		createdAt:    d.now(),
		resolvedAt:   time.Time{},
		errorMessage: "",
	}

	d.rememberReceipt(id)
	if old := d.receipts[id]; old != nil {
		d.receiptsByWorkflow.RemoveSingle(old.workflowID, id)
		d.receiptsByNode.RemoveSingle(old.nodeID, id)
		d.resolvedIndex.Remove(id)
	}
	d.receipts[id] = r
	d.receiptsByWorkflow.AddSingle(workflowID, id)
	d.receiptsByNode.AddSingle(nodeID, id)

	return nil
}

// resolveWorkflowReceiptsLocked resolves workflow receipts without acquiring the lock.
// Caller must hold mu.
//
// Takes workflowID (string) which identifies the completed workflow.
// Takes errorMessage (string) which contains any error from workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (d *DAL) resolveWorkflowReceiptsLocked(_ context.Context, workflowID, errorMessage string) (int, error) {
	receiptIDs := d.receiptsByWorkflow.Get(workflowID)
	now := d.now()
	count := 0

	for receiptID := range receiptIDs {
		r, ok := d.receipts[receiptID]
		if !ok {
			continue
		}

		if r.status != receiptPending {
			continue
		}

		d.rememberReceipt(receiptID)
		r.status = receiptResolved
		r.errorMessage = errorMessage
		r.resolvedAt = now
		d.resolvedIndex.Add(receiptID, now)
		count++
	}

	return count, nil
}

// getPendingReceiptsByNodeLocked retrieves pending receipts by node without acquiring the
// lock. Caller must hold mu.
//
// Takes nodeID (string) which identifies the node.
//
// Returns []PendingReceipt which contains pending receipts.
// Returns error when the query fails.
func (d *DAL) getPendingReceiptsByNodeLocked(_ context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	receiptIDs := d.receiptsByNode.Get(nodeID)
	return d.getPendingReceiptsFromIDs(receiptIDs), nil
}

// cleanupOldResolvedReceiptsLocked cleans up resolved receipts without acquiring the
// lock. Caller must hold mu.
//
// Takes olderThan (time.Time) which is the cutoff for deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (d *DAL) cleanupOldResolvedReceiptsLocked(_ context.Context, olderThan time.Time) (int, error) {
	count := 0
	for _, id := range d.resolvedIndex.KeysThrough(olderThan, maintenanceBatchSize) {
		r := d.receipts[id]
		if r == nil || r.status == receiptPending {
			continue
		}

		if r.resolvedAt.Before(olderThan) {
			d.rememberReceipt(id)
			d.receiptsByWorkflow.RemoveSingle(r.workflowID, id)
			d.receiptsByNode.RemoveSingle(r.nodeID, id)
			delete(d.receipts, id)
			d.resolvedIndex.Remove(id)
			count++
			if count >= maintenanceBatchSize {
				break
			}
		}
	}

	return count, nil
}

// timeoutStaleReceiptsLocked times out stale receipts without acquiring the lock. Caller
// must hold mu.
//
// Takes olderThan (time.Time) which is the cutoff for timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
func (d *DAL) timeoutStaleReceiptsLocked(_ context.Context, olderThan time.Time) (int, error) {
	now := d.now()
	count := 0

	for id, r := range d.receipts {
		if r.status != receiptPending {
			continue
		}

		if r.createdAt.Before(olderThan) {
			d.rememberReceipt(id)
			r.status = receiptTimedOut
			r.resolvedAt = now
			d.resolvedIndex.Add(id, now)
			count++
		}
	}

	return count, nil
}

// listFailedTasksLocked returns failed tasks without acquiring the lock. Caller must hold
// mu.
//
// Returns []*Task which contains the failed tasks.
// Returns error (always nil for the in-memory store).
func (d *DAL) listFailedTasksLocked(_ context.Context) ([]*orchestrator_domain.Task, error) {
	var result []*orchestrator_domain.Task
	for _, task := range d.tasks.All() {
		if task.Status == orchestrator_domain.StatusFailed {
			result = append(result, readCopy(task))
		}
	}
	return result, nil
}

// createTaskLocked stores a task without acquiring the lock. Caller must hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task to store.
//
// Returns error when the storage operation fails.
func (d *DAL) createTaskLocked(ctx context.Context, task *orchestrator_domain.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.pruneDanglingLocked(ctx)
	if _, found, _ := d.tasks.GetIfPresent(ctx, task.ID); !found && d.capacity > 0 && d.tasks.EstimatedSize() >= d.capacity {
		if err := d.reclaimTerminalLocked(ctx); err != nil {
			return err
		}
		if d.tasks.EstimatedSize() >= d.capacity {
			return orchestrator_domain.ErrTaskCapacity
		}
	}
	old, _, err := d.tasks.GetIfPresent(ctx, task.ID)
	if err != nil {
		return err
	}
	d.rememberTask(ctx, task.ID)
	stored := storedCopy(task)
	if err := d.setTaskLocked(ctx, task.ID, stored); err != nil {
		return fmt.Errorf("setting task %q: %w", task.ID, err)
	}

	if old != nil {
		d.reindexTaskLocked(old, stored)
	} else {
		d.indexTaskLocked(stored)
	}

	return nil
}

// noteDangling records a task ID that is indexed but missing from the cache, for the next
// write to prune.
//
// Takes taskID (string) which identifies the missing task.
//
// Safe for concurrent use by readers holding only the read lock; guarded by danglingMu.
func (d *DAL) noteDangling(taskID string) {
	d.danglingMu.Lock()
	if d.dangling == nil {
		d.dangling = make(map[string]struct{})
	}
	d.dangling[taskID] = struct{}{}
	d.danglingPending.Store(true)
	d.danglingMu.Unlock()
}

// pruneDanglingLocked removes missing tasks from the indexes outside transactions.
//
// Tasks that have reappeared remain indexed.
//
// Concurrent access requires the caller to hold the DAL write lock; danglingMu protects
// the pending IDs.
func (d *DAL) pruneDanglingLocked(ctx context.Context) {
	if d.journal != nil || !d.danglingPending.Load() {
		return
	}
	d.danglingMu.Lock()
	ids := d.dangling
	d.dangling = nil
	d.danglingPending.Store(false)
	d.danglingMu.Unlock()

	for id := range ids {
		if _, found, _ := d.tasks.GetIfPresent(ctx, id); found {
			continue
		}
		d.uncountUnsettledLocked(id)
		d.processingIndex.Remove(id)
		d.terminalIndex.Remove(id)
		d.scheduledIndex.Remove(id)
		d.executeIndex.Remove(id)
		d.workflowIndex.RemoveKey(id)
	}
}

// cloneTask returns a shallow copy of task.
//
// Takes task (*orchestrator_domain.Task) which is the source to clone.
//
// Returns *orchestrator_domain.Task which is a fresh shallow copy.
func cloneTask(task *orchestrator_domain.Task) *orchestrator_domain.Task {
	if task == nil {
		return nil
	}
	return new(*task)
}

// indexTaskLocked updates indexes for a task. Caller must hold mu.
//
// Takes task (*orchestrator_domain.Task) which is the task to index.
func (d *DAL) indexTaskLocked(task *orchestrator_domain.Task) {
	if task.Status == orchestrator_domain.StatusProcessing {
		d.processingIndex.Add(task.ID, task.UpdatedAt)
	}
	if !isActiveStatus(task.Status) {
		d.terminalIndex.Add(task.ID, task.UpdatedAt)
	}
	d.workflowIndex.AddSingle(task.WorkflowID, task.ID)

	if task.Status == orchestrator_domain.StatusScheduled {
		due := task.ScheduledExecuteAt
		if due.IsZero() {
			due = task.ExecuteAt
		}
		d.scheduledIndex.Add(task.ID, due)
	}

	if task.Status == orchestrator_domain.StatusPending || task.Status == orchestrator_domain.StatusRetrying {
		d.executeIndex.Add(task.ID, task.ExecuteAt)
	}

	if task.DeduplicationKey != "" && isActiveStatus(task.Status) {
		d.rememberDedup(task.DeduplicationKey)
		d.dedupIndex[task.DeduplicationKey] = task.ID
	}

	if !task.Background && isActiveStatus(task.Status) {
		d.unsettledByTag[task.BuildTag]++
		d.countedTag[task.ID] = task.BuildTag
	}
}

// uncountUnsettledLocked removes a task from its build tag's unsettled count. Caller must
// hold mu.
//
// Takes id (string) which identifies the task.
func (d *DAL) uncountUnsettledLocked(id string) {
	tag, counted := d.countedTag[id]
	if !counted {
		return
	}
	delete(d.countedTag, id)
	if remaining := d.unsettledByTag[tag] - 1; remaining > 0 {
		d.unsettledByTag[tag] = remaining
	} else {
		delete(d.unsettledByTag, tag)
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
// Takes task (*orchestrator_domain.Task) which is the task to remove from all indexes.
func (d *DAL) unindexTaskLocked(task *orchestrator_domain.Task) {
	d.uncountUnsettledLocked(task.ID)
	d.processingIndex.Remove(task.ID)
	d.terminalIndex.Remove(task.ID)
	d.workflowIndex.RemoveSingle(task.WorkflowID, task.ID)

	d.scheduledIndex.Remove(task.ID)
	d.executeIndex.Remove(task.ID)

	if task.DeduplicationKey != "" {
		if existing, ok := d.dedupIndex[task.DeduplicationKey]; ok && existing == task.ID {
			d.rememberDedup(task.DeduplicationKey)
			delete(d.dedupIndex, task.DeduplicationKey)
		}
	}
}

// getPendingReceiptsFromIDs retrieves pending receipts matching the given IDs.
//
// Takes receiptIDs (map[string]struct{}) which specifies the receipt IDs to look up.
//
// Returns []orchestrator_domain.PendingReceipt which contains the pending receipts found.
// Receipts that do not exist or are not pending are skipped.
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

// aggregateWorkflowData groups all tasks by workflow ID and computes aggregate counts and
// timestamps per workflow.
//
// Returns map[string]*workflowAgg which maps workflow IDs to their aggregated task
// statistics.
func (d *DAL) aggregateWorkflowData() map[string]*workflowAgg {
	workflows := make(map[string]*workflowAgg)

	for _, task := range d.tasks.All() {
		agg, ok := workflows[task.WorkflowID]
		if !ok {
			agg = &workflowAgg{
				createdAt:     task.CreatedAt.Unix(),
				updatedAt:     task.UpdatedAt.Unix(),
				taskCount:     0,
				completeCount: 0,
				failedCount:   0,
				activeCount:   0,
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

		switch task.Status { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
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
// This enables WAL persistence when the cache is created with PersistenceConfig. When
// provided, the DAL will not close the cache on shutdown - the caller is responsible for
// cache lifecycle management.
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

// WithClock sets the clock the DAL uses for its own time comparisons and stamps.
//
// Use the same clock as the task dispatcher so stored times and due checks agree.
//
// Takes c (clockpkg.Clock) which provides the time; nil keeps the real clock.
//
// Returns Option which configures the DAL's clock.
func WithClock(c clockpkg.Clock) Option {
	return func(d *DAL) {
		if c != nil {
			d.clock = c
		}
	}
}

// now returns the current time from the DAL's clock.
//
// Returns time.Time which is the configured clock's time, or the real time when unset.
func (d *DAL) now() time.Time {
	if d.clock == nil {
		return time.Now()
	}
	return d.clock.Now()
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
		clock:              clockpkg.RealClock(),
		processingIndex:    newTimeIndex(),
		terminalIndex:      newTimeIndex(),
		resolvedIndex:      newTimeIndex(),
		scheduledIndex:     newTimeIndex(),
		executeIndex:       newTimeIndex(),
		workflowIndex:      newWorkflowIndex(),
		dedupIndex:         make(map[string]string),
		unsettledByTag:     make(map[string]int64),
		countedTag:         make(map[string]string),
		nodes:              make(map[string]orchestrator_domain.NodeLease),
		publishedAt:        make(map[string]time.Time),
		receipts:           make(map[string]*receipt),
		receiptsByWorkflow: provider_otter.NewTagIndex[string](),
		receiptsByNode:     provider_otter.NewTagIndex[string](),
		ownsCache:          true,
		mu:                 sync.RWMutex{},
		tasks:              nil,
		capacity:           0,
		journal:            nil,
		pendingWaits:       nil,
		degraded:           atomic.Pointer[error]{},
		dangling:           nil,
		danglingPending:    atomic.Bool{},
		danglingMu:         sync.Mutex{},
		nodesMu:            sync.Mutex{},
		publishedMu:        sync.Mutex{},
	}

	for _, opt := range opts {
		opt(dal)
	}

	if dal.tasks == nil {
		capacity := config.Capacity
		if capacity <= 0 {
			capacity = defaultCacheCapacity
		}

		cacheOpts := cache_dto.Options[string, *orchestrator_domain.Task]{}
		dal.capacity = int(capacity)
		cacheOpts.MaximumEntries = 0

		cache, err := provider_otter.OtterProviderFactory(cacheOpts)
		if err != nil {
			return nil, fmt.Errorf("creating otter cache: %w", err)
		}
		dal.tasks = cache
	}

	if dal.capacity == 0 {
		dal.capacity = int(config.Capacity)
		if dal.capacity <= 0 {
			dal.capacity = defaultCacheCapacity
		}
	}
	return dal, nil
}

// otterTransactionDAL is a transaction-scoped TaskStore that delegates to the parent
// DAL's locked methods. The parent's mutex is already held by RunAtomic, so these methods
// skip lock acquisition.
type otterTransactionDAL struct {
	// parent is the owning DAL whose locked methods are called by this transaction store.
	parent *DAL
}

// CreateTask saves a new task within the current transaction.
//
// Takes task (*orchestrator_domain.Task) which is the task to save.
//
// Returns error when the task cannot be saved.
func (tx *otterTransactionDAL) CreateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	return tx.parent.createTaskLocked(ctx, task)
}

// CreateTasks inserts a batch of tasks within the current transaction.
//
// Takes tasks ([]*orchestrator_domain.Task) which contains the tasks to store.
//
// Returns error when the database operation fails.
func (tx *otterTransactionDAL) CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	return tx.parent.createTasksLocked(ctx, tasks)
}

// UpdateTask updates an existing task within the current transaction.
//
// Takes task (*orchestrator_domain.Task) which is the task to update.
//
// Returns error when the update fails.
func (tx *otterTransactionDAL) UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	return tx.parent.updateTaskLocked(ctx, task)
}

// GetTasksByID reads the tasks with the given IDs within the current transaction.
//
// Takes ids ([]string) which lists the task IDs to read.
//
// Returns []*orchestrator_domain.Task which holds copies of the tasks found.
// Returns error which is always nil.
func (tx *otterTransactionDAL) GetTasksByID(ctx context.Context, ids []string) ([]*orchestrator_domain.Task, error) {
	return tx.parent.getTasksByIDLocked(ctx, ids)
}

// GetWorkflowStatus checks whether all tasks in a workflow are complete within the
// current transaction.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns bool which is true when all tasks are done or failed.
// Returns error when the workflow cannot be found.
func (tx *otterTransactionDAL) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	return tx.parent.getWorkflowStatusLocked(ctx, workflowID)
}

// PromoteScheduledTasks moves scheduled tasks that are now due to pending status within
// the current transaction.
//
// Returns int which is the number of tasks promoted.
// Returns error when the promotion fails.
func (tx *otterTransactionDAL) PromoteScheduledTasks(ctx context.Context) (int, error) {
	return tx.parent.promoteScheduledTasksLocked(ctx)
}

// PendingTaskCount returns the number of tasks waiting to be run within the current
// transaction.
//
// Returns int64 which is the count of pending tasks.
// Returns error when the count cannot be retrieved.
func (tx *otterTransactionDAL) PendingTaskCount(ctx context.Context) (int64, error) {
	return tx.parent.pendingTaskCountLocked(ctx)
}

// CreateTaskWithDedup creates a task with deduplication support within the current
// transaction.
//
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the task cannot be created or a duplicate exists.
func (tx *otterTransactionDAL) CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error {
	return tx.parent.createTaskWithDedupLocked(ctx, task)
}

// CreateWorkflowReceipt creates a new workflow receipt for tracking completion within the
// current transaction.
//
// Takes id (string) which is the unique identifier for the receipt.
// Takes workflowID (string) which is the workflow being tracked.
// Takes nodeID (string) which is the node that created the receipt.
//
// Returns error when the receipt cannot be created.
func (tx *otterTransactionDAL) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	return tx.parent.createWorkflowReceiptLocked(ctx, id, workflowID, nodeID)
}

// ResolveWorkflowReceipts marks all pending receipts for a workflow as resolved within
// the current transaction.
//
// Takes workflowID (string) which identifies the completed workflow.
// Takes errorMessage (string) which contains any error from workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (tx *otterTransactionDAL) ResolveWorkflowReceipts(ctx context.Context, workflowID, errorMessage string) (int, error) {
	return tx.parent.resolveWorkflowReceiptsLocked(ctx, workflowID, errorMessage)
}

// GetPendingReceiptsByNode retrieves all pending receipts created by a node within the
// current transaction.
//
// Takes nodeID (string) which identifies the node.
//
// Returns []PendingReceipt which contains pending receipts.
// Returns error when the query fails.
func (tx *otterTransactionDAL) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	return tx.parent.getPendingReceiptsByNodeLocked(ctx, nodeID)
}

// CleanupOldResolvedReceipts deletes resolved receipts older than the specified time
// within the current transaction.
//
// Takes olderThan (time.Time) which is the cutoff for deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (tx *otterTransactionDAL) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	return tx.parent.cleanupOldResolvedReceiptsLocked(ctx, olderThan)
}

// TimeoutStaleReceipts marks very old pending receipts as timed out within the current
// transaction.
//
// Takes olderThan (time.Time) which is the cutoff for timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
func (tx *otterTransactionDAL) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	return tx.parent.timeoutStaleReceiptsLocked(ctx, olderThan)
}

// ListFailedTasks returns all tasks with a FAILED status within the current transaction.
//
// Returns []*Task which contains the failed tasks.
// Returns error (always nil for the in-memory store).
func (tx *otterTransactionDAL) ListFailedTasks(ctx context.Context) ([]*orchestrator_domain.Task, error) {
	return tx.parent.listFailedTasksLocked(ctx)
}

// RunAtomic rejects nested transactions.
//
// Returns error which is always ErrNestedTransactionUnsupported.
func (*otterTransactionDAL) RunAtomic(_ context.Context, _ func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	return cache_domain.ErrNestedTransactionUnsupported
}

// isActiveStatus returns true if the status represents an active task.
//
// Takes status (orchestrator_domain.TaskStatus) which is the task status to check.
//
// Returns bool which is true when the status is scheduled, pending, processing, or
// retrying.
func isActiveStatus(status orchestrator_domain.TaskStatus) bool {
	switch status { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
	case orchestrator_domain.StatusScheduled,
		orchestrator_domain.StatusPending,
		orchestrator_domain.StatusProcessing,
		orchestrator_domain.StatusRetrying:
		return true
	}
	return false
}
