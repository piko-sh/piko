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

package querier_adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/json"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// maxTransactionTimeout is the maximum duration a RunAtomic transaction
	// may hold before being cancelled.
	maxTransactionTimeout = 30 * time.Second
)

var (
	errDALNotInitialised = errors.New("cannot create transaction: DAL not initialised with a sql.DB connection")

	errTaskPoolAssertFailed = errors.New("failed to get task from pool: type assertion failed")

	_ orchestrator_dal.OrchestratorDALWithTx = (*Adapter)(nil)

	_ orchestrator_domain.TaskStore = (*Adapter)(nil)

	_ orchestrator_domain.OrchestratorInspector = (*Adapter)(nil)
)

// Adapter wraps the code-generated Queries struct to satisfy
// OrchestratorDALWithTx and TaskStore. It manages transactions, JSON
// serialisation, and timestamp conversion between the domain layer and
// the SQLite storage layer.
type Adapter struct {
	// db is the database connection for running queries.
	db orchestrator_db.DBTX

	// sqlDB holds the database connection for health checks, transaction
	// creation, and closing; nil when not set up.
	sqlDB *sql.DB

	// queries holds the code-generated database operations.
	queries *orchestrator_db.Queries

	// inTransaction is true when this Adapter is a transaction-scoped clone
	// created by withTransaction. It prevents nested transactions.
	inTransaction bool
}

// New creates a new Adapter wrapping the given database connection.
//
// If db is a *sql.DB, it is used directly for transactions and health checks.
// If db implements sqlDBProvider (e.g. PreparedDBTX), the provider is used
// for transaction creation.
//
// Takes db (orchestrator_db.DBTX) which provides the database connection or
// transaction to use for queries.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is the configured
// adapter ready for use.
func New(db orchestrator_db.DBTX) orchestrator_dal.OrchestratorDALWithTx {
	queries := orchestrator_db.New(db)

	var sqlDB *sql.DB
	if sdb, ok := db.(*sql.DB); ok {
		sqlDB = sdb
	}

	return &Adapter{
		db:      db,
		sqlDB:   sqlDB,
		queries: queries,
	}
}

// HealthCheck performs a health check on the database connection.
//
// Returns error when the database ping fails.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.sqlDB != nil {
		return a.sqlDB.PingContext(ctx)
	}
	return nil
}

// Close releases any resources held by the adapter.
//
// Returns error when the resources cannot be released.
func (*Adapter) Close() error {
	return nil
}

// RunAtomic executes fn within a serialisable transaction.
//
// The provided TaskStore is scoped to the transaction, so all reads and writes
// through it are atomic. If fn returns an error (or panics), all mutations are
// rolled back.
//
// Takes fn which receives a transactional TaskStore. The caller MUST use this
// transactional store for all operations that should be atomic.
//
// Returns error when fn returns an error, the transaction fails to begin, or
// the commit fails.
func (a *Adapter) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	if a.inTransaction {
		return cache_domain.ErrNestedTransactionUnsupported
	}

	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	return a.withTransaction(ctx, func(ctx context.Context, transactionDAL orchestrator_dal.OrchestratorDAL) error {
		store, ok := transactionDAL.(orchestrator_domain.TaskStore)
		if !ok {
			return errors.New("transaction DAL does not implement TaskStore")
		}
		return fn(ctx, store)
	})
}

// CreateTask saves a new task to the database.
//
// Takes task (*orchestrator_domain.Task) which is the task to save.
//
// Returns error when the database operation fails.
func (a *Adapter) CreateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	params, err := buildCreateTaskParams(task)
	if err != nil {
		return fmt.Errorf("building create task params: %w", err)
	}

	return a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		if err := qtx.CreateTask(ctx, params); err != nil {
			return fmt.Errorf("executing create task query: %w", err)
		}
		return nil
	})
}

// CreateTasks inserts a batch of tasks into the database using the generated
// CreateTasksBatch method with automatic chunking.
//
// Takes tasks ([]*orchestrator_domain.Task) which is the batch of tasks to
// insert.
//
// Returns error when the transaction cannot start, a task cannot be
// serialised, or the batch insert fails.
func (a *Adapter) CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	return a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		batchParams := make([]orchestrator_db.CreateTasksBatchParams, len(tasks))
		now := safeconv.Int64ToInt32(time.Now().UTC().Unix())

		for i, task := range tasks {
			payloadBytes, err := json.Marshal(task.Payload)
			if err != nil {
				return fmt.Errorf("failed to marshal payload for task %s: %w", task.ID, err)
			}
			configBytes, err := json.Marshal(task.Config)
			if err != nil {
				return fmt.Errorf("failed to marshal config for task %s: %w", task.ID, err)
			}

			var dedupKey *string
			if task.DeduplicationKey != "" {
				dedupKey = &task.DeduplicationKey
			}

			batchParams[i] = orchestrator_db.CreateTasksBatchParams{
				P1:  task.ID,
				P2:  task.WorkflowID,
				P3:  task.Executor,
				P4:  safeconv.IntToInt32(int(task.Config.Priority)),
				P5:  string(payloadBytes),
				P6:  string(configBytes),
				P7:  string(task.Status),
				P8:  safeconv.Int64ToInt32(task.ExecuteAt.Unix()),
				P9:  safeconv.IntToInt32(task.Attempt),
				P10: safeconv.Int64ToInt32(task.CreatedAt.Unix()),
				P11: now,
				P12: dedupKey,
			}
		}

		return qtx.CreateTasksBatch(ctx, batchParams)
	})
}

// UpdateTask saves changes to an existing task in the database.
//
// Takes task (*orchestrator_domain.Task) which holds the updated task data.
//
// Returns error when the database update fails.
func (a *Adapter) UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	params, err := buildUpdateTaskParams(task)
	if err != nil {
		return fmt.Errorf("building update task params: %w", err)
	}

	return a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		if err := qtx.UpdateTask(ctx, params); err != nil {
			return fmt.Errorf("executing update task query: %w", err)
		}
		return nil
	})
}

// CreateTaskWithDedup creates a task with deduplication support. If the task
// has a DeduplicationKey set, it checks for existing active tasks with the
// same key and returns ErrDuplicateTask if one exists; when DeduplicationKey
// is empty it behaves identically to CreateTask.
//
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the task cannot be created or a duplicate exists.
func (a *Adapter) CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error {
	if task.DeduplicationKey == "" {
		return a.CreateTask(ctx, task)
	}

	return a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		result, err := qtx.CheckDuplicateActiveTask(ctx, &task.DeduplicationKey)
		if err != nil {
			return fmt.Errorf("failed to check for duplicate task: %w", err)
		}

		if result.HasDuplicate {
			return orchestrator_domain.ErrDuplicateTask
		}

		params, err := buildCreateTaskParams(task)
		if err != nil {
			return fmt.Errorf("building create task params: %w", err)
		}

		if err := qtx.CreateTask(ctx, params); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
		return nil
	})
}

// FetchAndMarkDueTasks atomically fetches due tasks and marks them as
// processing. The fetch and mark happen within a single transaction to
// prevent multiple workers from picking up the same task.
//
// Takes priority (orchestrator_domain.TaskPriority) which filters tasks by
// their priority level.
// Takes limit (int) which specifies the maximum number of tasks to fetch.
//
// Returns []*orchestrator_domain.Task which contains the fetched tasks marked
// as processing.
// Returns error when the fetch or mark operation fails.
func (a *Adapter) FetchAndMarkDueTasks(ctx context.Context, priority orchestrator_domain.TaskPriority, limit int) ([]*orchestrator_domain.Task, error) {
	var domainTasks []*orchestrator_domain.Task
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())

	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		fetchedRows, err := qtx.FetchDueTasks(ctx, orchestrator_db.FetchDueTasksParams{
			Statuses: []string{
				string(orchestrator_domain.StatusPending),
				string(orchestrator_domain.StatusRetrying),
			},
			P2: safeconv.IntToInt32(int(priority)),
			P3: nowSeconds,
			P4: safeconv.IntToInt32(limit),
		})
		if err != nil {
			return fmt.Errorf("failed to fetch due tasks: %w", err)
		}

		if len(fetchedRows) == 0 {
			domainTasks = []*orchestrator_domain.Task{}
			return nil
		}

		taskIDs, converted, err := convertFetchedRowsToDomain(fetchedRows)
		if err != nil {
			return err
		}
		domainTasks = converted

		if err := qtx.MarkTasksAsProcessing(ctx, orchestrator_db.MarkTasksAsProcessingParams{
			P1:  safeconv.Int64ToInt32(time.Now().UTC().Unix()),
			IDs: taskIDs,
		}); err != nil {
			return fmt.Errorf("failed to mark tasks as processing: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return domainTasks, nil
}

// convertFetchedRowsToDomain converts database rows to domain tasks and
// collects their IDs for subsequent marking.
//
// Takes fetchedRows ([]orchestrator_db.FetchDueTasksRow) which contains the
// rows to convert.
//
// Returns []string which contains the task IDs.
// Returns []*orchestrator_domain.Task which contains the converted domain
// tasks.
// Returns error when a row cannot be converted.
func convertFetchedRowsToDomain(fetchedRows []orchestrator_db.FetchDueTasksRow) ([]string, []*orchestrator_domain.Task, error) {
	taskIDs := make([]string, len(fetchedRows))
	domainTasks := make([]*orchestrator_domain.Task, len(fetchedRows))
	for i := range fetchedRows {
		taskIDs[i] = fetchedRows[i].ID
		domainTask, err := convertDBTaskToDomain(&fetchedRows[i])
		if err != nil {
			for j := range i {
				orchestrator_domain.TaskPool.Put(domainTasks[j])
			}
			return nil, nil, fmt.Errorf("failed to convert task '%s' from DB model: %w", fetchedRows[i].ID, err)
		}
		domainTasks[i] = domainTask
	}
	return taskIDs, domainTasks, nil
}

// GetWorkflowStatus checks if all tasks in a workflow are complete by
// querying whether any non-terminal tasks remain.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns bool which is true when all tasks in the workflow are complete.
// Returns error when the workflow status cannot be determined.
func (a *Adapter) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	row, err := a.queries.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		return false, fmt.Errorf("failed to get workflow status: %w", err)
	}

	return !row.HasIncomplete, nil
}

// PromoteScheduledTasks moves scheduled tasks that are ready to run to pending
// status.
//
// Returns int which is the number of tasks promoted.
// Returns error when the database transaction fails.
func (a *Adapter) PromoteScheduledTasks(ctx context.Context) (int, error) {
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())

	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.PromoteScheduledTasks(ctx, orchestrator_db.PromoteScheduledTasksParams{
			P1: nowSeconds,
			P2: nowSeconds,
		})
		if txErr != nil {
			return fmt.Errorf("failed to promote scheduled tasks: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("promoting scheduled tasks: %w", err)
	}

	return int(rowsAffected), nil
}

// PendingTaskCount returns the number of tasks in pending status.
//
// Returns int64 which is the count of pending tasks.
// Returns error when the database query fails.
func (a *Adapter) PendingTaskCount(ctx context.Context) (int64, error) {
	result, err := a.queries.PendingTaskCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("querying pending task count: %w", err)
	}
	return int64(result.Count), nil
}

// RecoverStaleTasks resets PROCESSING tasks that have exceeded the stale
// threshold. Tasks are marked as RETRYING if they have attempts remaining,
// or FAILED if they have exceeded max retries.
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
func (a *Adapter) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	now := time.Now().UTC()
	nowSeconds := safeconv.Int64ToInt32(now.Unix())
	staleThresholdSeconds := safeconv.Int64ToInt32(now.Add(-staleThreshold).Unix())

	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.RecoverStaleTasks(ctx, orchestrator_db.RecoverStaleTasksParams{
			P1: maxRetries,
			P2: maxRetries,
			P3: &recoveryError,
			P4: nowSeconds,
			P5: nowSeconds,
			P6: staleThresholdSeconds,
		})
		if txErr != nil {
			return fmt.Errorf("executing stale task recovery: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("recovering stale tasks: %w", err)
	}

	return int(rowsAffected), nil
}

// GetStaleProcessingTaskCount returns the count of tasks stuck in PROCESSING
// longer than the threshold.
//
// Takes staleThreshold (time.Duration) which defines when a PROCESSING task is
// considered stuck.
//
// Returns int64 which is the count of stale tasks.
// Returns error when the count cannot be retrieved.
func (a *Adapter) GetStaleProcessingTaskCount(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	staleThresholdSeconds := safeconv.Int64ToInt32(time.Now().UTC().Add(-staleThreshold).Unix())

	result, err := a.queries.GetStaleProcessingTaskCount(ctx, staleThresholdSeconds)
	if err != nil {
		return 0, fmt.Errorf("querying stale processing task count: %w", err)
	}

	return int64(result.Count), nil
}

// UpdateTaskHeartbeat updates the updated_at timestamp for a task in
// PROCESSING status.
//
// Takes taskID (string) which identifies the task to update.
//
// Returns error when the update fails.
func (a *Adapter) UpdateTaskHeartbeat(ctx context.Context, taskID string) error {
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())
	return a.queries.UpdateTaskHeartbeat(ctx, orchestrator_db.UpdateTaskHeartbeatParams{
		P1: nowSeconds,
		P2: taskID,
	})
}

// ClaimStaleTasksForRecovery atomically claims stale PROCESSING tasks for
// recovery. SQLite uses transaction-based claiming since it does not support
// FOR UPDATE SKIP LOCKED, so the method fetches candidate stale tasks and
// attempts to claim each one individually, collecting only those where the
// claim succeeds.
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
func (a *Adapter) ClaimStaleTasksForRecovery(
	ctx context.Context, nodeID string, staleThreshold time.Duration, leaseTimeout time.Duration, batchLimit int,
) ([]orchestrator_domain.RecoveryClaimedTask, error) {
	now := time.Now().UTC()
	nowUnixSeconds := safeconv.Int64ToInt32(now.Unix())
	staleThresholdSeconds := safeconv.Int64ToInt32(now.Add(-staleThreshold).Unix())
	leaseExpiresAtSeconds := safeconv.Int64ToInt32(now.Add(leaseTimeout).Unix())

	var claimed []orchestrator_domain.RecoveryClaimedTask

	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		rows, err := qtx.GetStaleTasksForRecovery(ctx, orchestrator_db.GetStaleTasksForRecoveryParams{
			P1: staleThresholdSeconds,
			P2: &nowUnixSeconds,
			P3: safeconv.IntToInt32(batchLimit),
		})
		if err != nil {
			return fmt.Errorf("getting stale tasks for recovery: %w", err)
		}

		if len(rows) == 0 {
			return nil
		}

		claimed = make([]orchestrator_domain.RecoveryClaimedTask, 0, len(rows))
		for _, row := range rows {
			rowsAffected, err := qtx.ClaimTaskForRecovery(ctx, orchestrator_db.ClaimTaskForRecoveryParams{
				P1: &nodeID,
				P2: &leaseExpiresAtSeconds,
				P3: row.ID,
				P4: &nowUnixSeconds,
			})
			if err != nil {
				return fmt.Errorf("claiming task for recovery: %w", err)
			}
			if rowsAffected > 0 {
				claimed = append(claimed, orchestrator_domain.RecoveryClaimedTask{
					ID:         row.ID,
					WorkflowID: row.WorkflowID,
					Attempt:    row.Attempt,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return claimed, nil
}

// RecoverClaimedTasks recovers all tasks previously claimed by this node.
// Tasks are set to RETRYING if they have attempts remaining, or FAILED
// otherwise, and the lease is cleared.
//
// Takes nodeID (string) which identifies the node that claimed the tasks.
// Takes maxRetries (int) which sets the maximum retries before marking FAILED.
// Takes recoveryError (string) which is the error message to record.
//
// Returns int which is the count of tasks recovered.
// Returns error when the recovery fails.
func (a *Adapter) RecoverClaimedTasks(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error) {
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())

	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.RecoverClaimedTasks(ctx, orchestrator_db.RecoverClaimedTasksParams{
			P1: maxRetries,
			P2: maxRetries,
			P3: &recoveryError,
			P4: nowSeconds,
			P5: nowSeconds,
			P6: &nodeID,
		})
		if txErr != nil {
			return fmt.Errorf("recovering claimed tasks: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// ReleaseRecoveryLeases releases all recovery leases held by this node.
//
// Takes nodeID (string) which identifies the node releasing leases.
//
// Returns int which is the count of leases released.
// Returns error when the release fails.
func (a *Adapter) ReleaseRecoveryLeases(ctx context.Context, nodeID string) (int, error) {
	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.ReleaseRecoveryLeases(ctx, &nodeID)
		if txErr != nil {
			return fmt.Errorf("releasing recovery leases: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// CreateWorkflowReceipt creates a new workflow receipt for tracking completion.
//
// Takes id (string) which is the unique identifier for the receipt.
// Takes workflowID (string) which is the workflow being tracked.
// Takes nodeID (string) which is the node that created the receipt.
//
// Returns error when the receipt cannot be created.
func (a *Adapter) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())

	return a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		err := qtx.CreateWorkflowReceipt(ctx, orchestrator_db.CreateWorkflowReceiptParams{
			P1: id,
			P2: workflowID,
			P3: nodeID,
			P4: nowSeconds,
			P5: nowSeconds,
		})
		if err != nil {
			return fmt.Errorf("creating workflow receipt: %w", err)
		}
		return nil
	})
}

// ResolveWorkflowReceipts marks all pending receipts for a workflow as
// resolved.
//
// Takes workflowID (string) which identifies the completed workflow.
// Takes errorMessage (string) which contains any error from workflow
// completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (a *Adapter) ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage string) (int, error) {
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())
	var errorMessagePtr *string
	if errorMessage != "" {
		errorMessagePtr = &errorMessage
	}

	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.ResolveWorkflowReceipts(ctx, orchestrator_db.ResolveWorkflowReceiptsParams{
			P1: errorMessagePtr,
			P2: nowSeconds,
			P3: &nowSeconds,
			P4: workflowID,
		})
		if txErr != nil {
			return fmt.Errorf("resolving workflow receipts: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// GetPendingReceiptsByNode retrieves all pending receipts created by a node.
//
// Takes nodeID (string) which identifies the node to query.
//
// Returns []orchestrator_domain.PendingReceipt which contains the pending
// receipts for the node.
// Returns error when the database query fails.
func (a *Adapter) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	rows, err := a.queries.GetPendingReceiptsByNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("getting pending receipts by node: %w", err)
	}

	result := make([]orchestrator_domain.PendingReceipt, len(rows))
	for i, row := range rows {
		result[i] = orchestrator_domain.PendingReceipt{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			NodeID:     nodeID,
			CreatedAt:  int64(row.CreatedAt),
		}
	}
	return result, nil
}

// GetPendingReceiptsByWorkflow retrieves all pending receipts for a workflow.
//
// Takes workflowID (string) which identifies the workflow.
//
// Returns []orchestrator_domain.PendingReceipt which contains pending receipts.
// Returns error when the query fails.
func (a *Adapter) GetPendingReceiptsByWorkflow(ctx context.Context, workflowID string) ([]orchestrator_domain.PendingReceipt, error) {
	rows, err := a.queries.GetPendingReceiptsByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("getting pending receipts by workflow: %w", err)
	}

	result := make([]orchestrator_domain.PendingReceipt, len(rows))
	for i, row := range rows {
		result[i] = orchestrator_domain.PendingReceipt{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			NodeID:     row.NodeID,
			CreatedAt:  int64(row.CreatedAt),
		}
	}
	return result, nil
}

// CleanupOldResolvedReceipts deletes resolved receipts older than the
// specified time.
//
// Takes olderThan (time.Time) which is the cutoff for deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (a *Adapter) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	olderThanSeconds := safeconv.Int64ToInt32(olderThan.Unix())

	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.CleanupOldResolvedReceipts(ctx, &olderThanSeconds)
		if txErr != nil {
			return fmt.Errorf("cleaning up old resolved receipts: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// TimeoutStaleReceipts marks very old pending receipts as timed out.
//
// Takes olderThan (time.Time) which is the cutoff for timeout.
//
// Returns int which is the count of receipts timed out.
// Returns error when the timeout operation fails.
func (a *Adapter) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	nowSeconds := safeconv.Int64ToInt32(time.Now().UTC().Unix())
	olderThanSeconds := safeconv.Int64ToInt32(olderThan.Unix())

	var rowsAffected int64
	err := a.runInTransaction(ctx, func(ctx context.Context, qtx *orchestrator_db.Queries) error {
		var txErr error
		rowsAffected, txErr = qtx.TimeoutStaleReceipts(ctx, orchestrator_db.TimeoutStaleReceiptsParams{
			P1: nowSeconds,
			P2: olderThanSeconds,
		})
		if txErr != nil {
			return fmt.Errorf("timing out stale receipts: %w", txErr)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// ListFailedTasks returns all tasks with a FAILED status.
//
// Returns []*orchestrator_domain.Task which contains the failed tasks.
// Returns error when the query fails.
func (a *Adapter) ListFailedTasks(ctx context.Context) ([]*orchestrator_domain.Task, error) {
	dbRows, err := a.queries.ListFailedTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying failed tasks: %w", err)
	}

	tasks := make([]*orchestrator_domain.Task, len(dbRows))
	for i := range dbRows {
		row := &dbRows[i]
		tasks[i] = &orchestrator_domain.Task{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			Executor:   row.Executor,
			Status:     orchestrator_domain.StatusFailed,
			Attempt:    int(row.Attempt),
		}
		if row.LastError != nil {
			tasks[i].LastError = *row.LastError
		}
	}
	return tasks, nil
}

// ListTaskSummary returns task counts grouped by status.
//
// Returns []orchestrator_domain.TaskSummary which contains one entry per
// status with its count.
// Returns error when the database query fails.
func (a *Adapter) ListTaskSummary(ctx context.Context) ([]orchestrator_domain.TaskSummary, error) {
	rows, err := a.queries.ListTaskStatusCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing task status counts: %w", err)
	}

	results := make([]orchestrator_domain.TaskSummary, len(rows))
	for i, row := range rows {
		results[i] = orchestrator_domain.TaskSummary{
			Status: row.Status,
			Count:  int64(row.TaskCount),
		}
	}

	return results, nil
}

// ListRecentTasks returns the most recently updated tasks.
//
// Takes limit (int32) which specifies the maximum number of tasks to return.
//
// Returns []orchestrator_domain.TaskListItem which contains the tasks ordered
// by update time descending.
// Returns error when the database query fails.
func (a *Adapter) ListRecentTasks(ctx context.Context, limit int32) ([]orchestrator_domain.TaskListItem, error) {
	rows, err := a.queries.ListRecentTasks(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing recent tasks: %w", err)
	}

	results := make([]orchestrator_domain.TaskListItem, len(rows))
	for i, row := range rows {
		results[i] = orchestrator_domain.TaskListItem{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			Executor:   row.Executor,
			Status:     row.Status,
			Priority:   row.Priority,
			Attempt:    row.Attempt,
			LastError:  row.LastError,
			CreatedAt:  int64(row.CreatedAt),
			UpdatedAt:  int64(row.UpdatedAt),
		}
	}

	return results, nil
}

// ListWorkflowSummary returns workflow-level aggregates ordered by most
// recently updated.
//
// Takes limit (int32) which specifies the maximum number of workflows to
// return.
//
// Returns []orchestrator_domain.WorkflowSummary which contains one entry per
// workflow with task counts by status.
// Returns error when the database query fails.
func (a *Adapter) ListWorkflowSummary(ctx context.Context, limit int32) ([]orchestrator_domain.WorkflowSummary, error) {
	rows, err := a.queries.ListWorkflowSummary(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing workflow summary: %w", err)
	}

	results := make([]orchestrator_domain.WorkflowSummary, len(rows))
	for i, row := range rows {
		results[i] = orchestrator_domain.WorkflowSummary{
			WorkflowID:    row.WorkflowID,
			TaskCount:     int64(row.TaskCount),
			CompleteCount: derefInt32AsInt64(row.CompleteCount),
			FailedCount:   derefInt32AsInt64(row.FailedCount),
			ActiveCount:   derefInt32AsInt64(row.ActiveCount),
			CreatedAt:     derefInt32AsInt64(row.CreatedAt),
			UpdatedAt:     derefInt32AsInt64(row.UpdatedAt),
		}
	}

	return results, nil
}

// derefInt32AsInt64 returns the value behind a nullable int32 pointer as
// int64, defaulting to zero when nil.
//
// Takes value (*int32) which may be nil.
//
// Returns int64 which is the dereferenced and widened value.
func derefInt32AsInt64(value *int32) int64 {
	if value == nil {
		return 0
	}
	return int64(*value)
}

// runInTransaction executes fn within a transaction using the generated
// Queries struct.
//
// If the adapter is already inside a transaction (inTransaction == true), it
// reuses the existing queries to avoid deadlocking on SQLite's single-writer
// lock.
//
// Takes fn which is the callback executed inside the transaction.
//
// Returns error when the transaction fails to begin, fn returns an error, or
// the commit fails.
func (a *Adapter) runInTransaction(ctx context.Context, fn func(ctx context.Context, qtx *orchestrator_db.Queries) error) error {
	if a.inTransaction {
		return fn(ctx, a.queries)
	}

	db, err := a.beginTxDB()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(ctx, a.queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// beginTxDB resolves the *sql.DB needed to start a transaction.
//
// Returns *sql.DB which is the underlying database connection.
// Returns error when the adapter has no suitable database connection.
func (a *Adapter) beginTxDB() (*sql.DB, error) {
	if a.sqlDB != nil {
		return a.sqlDB, nil
	}
	return nil, errDALNotInitialised
}

// withTransaction is an internal helper that executes a function
// within a database transaction, providing a transaction-scoped
// Adapter clone.
//
// Takes transactionFunction which is the function to execute
// within the transaction scope.
//
// Returns error when the DAL has no database connection, when
// beginning the transaction fails, when transactionFunction
// returns an error, or when commit fails.
//
// Panics if transactionFunction panics; the transaction is rolled
// back and the panic is re-raised.
func (a *Adapter) withTransaction(ctx context.Context, transactionFunction func(ctx context.Context, dal orchestrator_dal.OrchestratorDAL) error) error {
	if a.sqlDB == nil {
		return errDALNotInitialised
	}

	tx, err := a.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	txQueries := orchestrator_db.New(tx)
	txAdapter := &Adapter{
		db:            tx,
		sqlDB:         a.sqlDB,
		queries:       txQueries,
		inTransaction: true,
	}

	if err := transactionFunction(ctx, txAdapter); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// buildCreateTaskParams marshals task fields and builds database creation
// parameters.
//
// Takes task (*orchestrator_domain.Task) which provides the task data to
// convert.
//
// Returns orchestrator_db.CreateTaskParams which contains the database-ready
// task parameters.
// Returns error when the payload or config cannot be marshalled to JSON.
func buildCreateTaskParams(task *orchestrator_domain.Task) (orchestrator_db.CreateTaskParams, error) {
	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		return orchestrator_db.CreateTaskParams{}, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return orchestrator_db.CreateTaskParams{}, fmt.Errorf("failed to marshal task config: %w", err)
	}

	now := time.Now().UTC()

	return orchestrator_db.CreateTaskParams{
		P1:  task.ID,
		P2:  task.WorkflowID,
		P3:  task.Executor,
		P4:  safeconv.IntToInt32(int(task.Config.Priority)),
		P5:  string(payloadBytes),
		P6:  string(configBytes),
		P7:  string(task.Status),
		P8:  safeconv.Int64ToInt32(task.ExecuteAt.Unix()),
		P9:  safeconv.IntToInt32(task.Attempt),
		P10: safeconv.Int64ToInt32(task.CreatedAt.Unix()),
		P11: safeconv.Int64ToInt32(now.Unix()),
	}, nil
}

// buildUpdateTaskParams converts a task into database update parameters.
//
// Takes task (*orchestrator_domain.Task) which provides the task data to
// convert.
//
// Returns orchestrator_db.UpdateTaskParams which contains the serialised task
// fields ready for database update.
// Returns error when marshalling the payload, config, or result fails.
func buildUpdateTaskParams(task *orchestrator_domain.Task) (orchestrator_db.UpdateTaskParams, error) {
	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		return orchestrator_db.UpdateTaskParams{}, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return orchestrator_db.UpdateTaskParams{}, fmt.Errorf("failed to marshal task config: %w", err)
	}

	resultBytes, err := json.Marshal(task.Result)
	if err != nil {
		return orchestrator_db.UpdateTaskParams{}, fmt.Errorf("failed to marshal task result: %w", err)
	}

	var lastErrorPtr *string
	if task.LastError != "" {
		lastErrorPtr = &task.LastError
	}

	var resultPtr *string
	if task.Result != nil {
		resultPtr = new(string(resultBytes))
	}

	return orchestrator_db.UpdateTaskParams{
		P1:  string(task.Status),
		P2:  safeconv.IntToInt32(int(task.Config.Priority)),
		P3:  safeconv.Int64ToInt32(task.ExecuteAt.Unix()),
		P4:  safeconv.IntToInt32(task.Attempt),
		P5:  lastErrorPtr,
		P6:  resultPtr,
		P7:  string(payloadBytes),
		P8:  string(configBytes),
		P9:  safeconv.Int64ToInt32(time.Now().UTC().Unix()),
		P10: task.ID,
	}, nil
}

// convertDBTaskToDomain converts a database task row to a domain task. It
// obtains a task from the pool and populates it with the database values.
//
// Takes dbTask (*orchestrator_db.FetchDueTasksRow) which is the database row
// to convert.
//
// Returns *orchestrator_domain.Task which is the populated domain task from
// the pool.
// Returns error when the pool returns an invalid type or JSON unmarshalling
// fails for payload, config, or result fields.
func convertDBTaskToDomain(dbTask *orchestrator_db.FetchDueTasksRow) (*orchestrator_domain.Task, error) {
	pooledTask, ok := orchestrator_domain.TaskPool.Get().(*orchestrator_domain.Task)
	if !ok {
		return nil, errTaskPoolAssertFailed
	}
	task := pooledTask
	task.Reset()

	task.ID = dbTask.ID
	task.WorkflowID = dbTask.WorkflowID
	task.Executor = dbTask.Executor
	task.Status = orchestrator_domain.TaskStatus(dbTask.Status)
	task.ExecuteAt = time.Unix(int64(dbTask.ExecuteAt), 0).UTC()
	task.Attempt = int(dbTask.Attempt)
	task.CreatedAt = time.Unix(int64(dbTask.CreatedAt), 0).UTC()
	task.UpdatedAt = time.Unix(int64(dbTask.UpdatedAt), 0).UTC()

	if dbTask.LastError != nil {
		task.LastError = *dbTask.LastError
	}

	if dbTask.DeduplicationKey != nil {
		task.DeduplicationKey = *dbTask.DeduplicationKey
	}

	if err := json.UnmarshalString(dbTask.Payload, &task.Payload); err != nil {
		orchestrator_domain.TaskPool.Put(task)
		return nil, fmt.Errorf("unmarshal payload for task '%s': %w", dbTask.ID, err)
	}
	if err := json.UnmarshalString(dbTask.Config, &task.Config); err != nil {
		orchestrator_domain.TaskPool.Put(task)
		return nil, fmt.Errorf("unmarshal config for task '%s': %w", dbTask.ID, err)
	}

	task.Config.Priority = orchestrator_domain.TaskPriority(dbTask.Priority)

	if dbTask.Result != nil {
		if err := json.UnmarshalString(*dbTask.Result, &task.Result); err != nil {
			orchestrator_domain.TaskPool.Put(task)
			return nil, fmt.Errorf("unmarshal result for task '%s': %w", dbTask.ID, err)
		}
	}

	return task, nil
}
