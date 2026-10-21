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
	"errors"
	"fmt"
	"slices"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/safeconv"
)

const (
	// maxTransactionTimeout is the maximum duration a RunAtomic transaction may hold before
	// being cancelled.
	maxTransactionTimeout = 30 * time.Second

	// getTasksByIDChunkSize bounds the IDs bound into one GetTasksByID statement, keeping
	// every statement under the smallest engine bind-variable cap.
	getTasksByIDChunkSize = 500

	// maintenanceBatchSize caps the number of pending tasks read in one reconciliation page.
	maintenanceBatchSize = 100
)

var (
	// errDALNotInitialised is returned when a transaction is attempted but the core has not
	// been initialised with a sql.DB connection.
	errDALNotInitialised = errors.New("cannot create transaction: DAL not initialised with a sql.DB connection")

	// log is the package-level logger for the dalcore package.
	log = logger_domain.GetLogger("piko/internal/orchestrator/orchestrator_dal/dalcore")

	_ orchestrator_dal.OrchestratorDALWithTx = (*core)(nil)

	_ orchestrator_domain.TaskStore = (*core)(nil)

	_ orchestrator_domain.OrchestratorInspector = (*core)(nil)
)

// core is the dialect-agnostic orchestrator DAL. It satisfies OrchestratorDALWithTx,
// TaskStore, and OrchestratorInspector, delegating every generated query to a per-dialect
// Driver while owning JSON serialisation, timestamp conversion, task pooling, and
// transaction lifecycle.
type core struct {
	// sqlDB holds the database connection for health checks, transaction creation, and
	// closing; nil when not set up (for example when driven by a non-*sql.DB DBTX).
	sqlDB *sql.DB

	// driver performs the dialect-specific generated-query calls.
	driver Driver

	// clock supplies the wall-clock time used for domain timestamps and stale, due, and
	// threshold computations. It defaults to the real clock and can be replaced with a mock
	// for deterministic tests.
	clock clock.Clock

	// inTransaction is true when this core is a transaction-scoped clone created by
	// withTransaction. It prevents nested transactions.
	inTransaction bool
}

// Option configures a core during construction.
type Option func(*core)

// WithClock sets the clock the core uses for stored timestamps and for stale, due, and
// threshold computations.
//
// Use the same clock as the task dispatcher so stored times and due checks agree.
//
// Takes c (clock.Clock) which provides the time; nil keeps the real clock.
//
// Returns Option which configures the core's clock.
func WithClock(c clock.Clock) Option {
	return func(target *core) {
		if c != nil {
			target.clock = c
		}
	}
}

// New creates a core backed by the given database connection and dialect driver.
//
// Takes sqlDB (*sql.DB) which provides the database connection for transactions and
// health checks; it may be nil when the driver is not backed by a *sql.DB.
// Takes driver (Driver) which performs the dialect-specific generated-query calls.
// Takes opts (...Option) which customise the core, such as its clock.
//
// Returns orchestrator_dal.OrchestratorDALWithTx which is the configured DAL ready for
// use.
func New(sqlDB *sql.DB, driver Driver, opts ...Option) orchestrator_dal.OrchestratorDALWithTx {
	c := &core{
		sqlDB:         sqlDB,
		driver:        driver,
		clock:         clock.RealClock(),
		inTransaction: false,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// HealthCheck performs a health check on the database connection.
//
// Returns error when the database ping fails.
func (c *core) HealthCheck(ctx context.Context) error {
	if c.sqlDB != nil {
		return c.sqlDB.PingContext(ctx)
	}
	return nil
}

// Close is a no-op because the caller owns the database connection.
//
// Returns error which is always nil.
func (*core) Close() error {
	return nil
}

// RunAtomic executes fn within a database transaction at the driver's default isolation
// level.
//
// The provided TaskStore is scoped to the transaction, so all reads and writes through it
// are atomic. If fn returns an error (or panics), all mutations are rolled back. A
// transaction that fails with a serialisation failure, deadlock or busy database is
// rolled back and run again, fn included, a few times with jittered backoff.
//
// Takes fn (func(context.Context, orchestrator_domain.TaskStore) error) which receives a
// transactional TaskStore. The caller MUST use this transactional store for all
// operations that should be atomic.
//
// Returns error when fn returns an error, the transaction fails to begin, or the commit
// fails.
func (c *core) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	if c.inTransaction {
		return cache_domain.ErrNestedTransactionUnsupported
	}

	ctx, cancel := context.WithTimeoutCause(ctx, maxTransactionTimeout,
		fmt.Errorf("transaction exceeded maximum duration of %s", maxTransactionTimeout))
	defer cancel()

	return c.retryTransient(ctx, func() error {
		return c.withTransaction(ctx, func(ctx context.Context, transactionDAL orchestrator_dal.OrchestratorDAL) error {
			store, ok := transactionDAL.(orchestrator_domain.TaskStore)
			if !ok {
				return errors.New("transaction DAL does not implement TaskStore")
			}
			return fn(ctx, store)
		})
	})
}

// CreateTask saves a new task to the database.
//
// Takes task (*orchestrator_domain.Task) which is the task to save.
//
// Returns error when the database operation fails.
func (c *core) CreateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	params, err := buildCreateTaskParams(task, c.clock.Now().UTC())
	if err != nil {
		return fmt.Errorf("building create task params: %w", err)
	}

	return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		if err := driver.CreateTask(ctx, params); err != nil {
			return fmt.Errorf("executing create task query: %w", err)
		}
		return nil
	})
}

// CreateTasks inserts a batch of tasks into the database using the driver's batch method
// with automatic chunking.
//
// Takes tasks ([]*orchestrator_domain.Task) which is the batch of tasks to insert.
//
// Returns error when the transaction cannot start, a task cannot be serialised, or the
// batch insert fails.
func (c *core) CreateTasks(ctx context.Context, tasks []*orchestrator_domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	now := c.clock.Now().UTC()
	batchParams := make([]CreateTaskBatchParams, len(tasks))
	for i, task := range tasks {
		params, err := buildCreateTaskParams(task, now)
		if err != nil {
			return fmt.Errorf("building params for task %s: %w", task.ID, err)
		}
		batchParams[i] = params
	}

	return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		return driver.CreateTasksBatch(ctx, batchParams)
	})
}

// UpdateTask saves changes to an existing task in the database.
//
// Takes task (*orchestrator_domain.Task) which holds the updated task data.
//
// Returns error which is ErrTaskNotFound when the task does not exist, or another error
// when the database update fails.
func (c *core) UpdateTask(ctx context.Context, task *orchestrator_domain.Task) error {
	params, err := buildUpdateTaskParams(task, c.clock.Now().UTC())
	if err != nil {
		return fmt.Errorf("building update task params: %w", err)
	}

	return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		updated, err := driver.UpdateTask(ctx, params)
		if err != nil {
			return fmt.Errorf("executing update task query: %w", err)
		}
		if updated == 0 {
			return fmt.Errorf("%w: %s", orchestrator_domain.ErrTaskNotFound, task.ID)
		}
		return nil
	})
}

// CreateTaskWithDedup creates a task with deduplication support. If the task has a
// DeduplicationKey set, it returns ErrDuplicateTask when an active task with the same key
// exists; when DeduplicationKey is empty it behaves identically to CreateTask.
//
// Takes task (*orchestrator_domain.Task) which is the task to create.
//
// Returns error when the task cannot be created or a duplicate exists.
func (c *core) CreateTaskWithDedup(ctx context.Context, task *orchestrator_domain.Task) error {
	if task.DeduplicationKey == "" {
		return c.CreateTask(ctx, task)
	}

	params, err := buildCreateTaskParams(task, c.clock.Now().UTC())
	if err != nil {
		return fmt.Errorf("building create task params: %w", err)
	}

	return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		created, err := driver.CreateWithDedup(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to create task with dedup: %w", err)
		}
		if !created {
			return orchestrator_domain.ErrDuplicateTask
		}
		return nil
	})
}

// GetTasksByID returns the tasks with the given IDs, skipping IDs that do not exist.
//
// The IDs are read in chunks so no statement exceeds the engine's bind-variable cap.
//
// Takes ids ([]string) which lists the task IDs to read.
//
// Returns []*orchestrator_domain.Task which holds the tasks found.
// Returns error when a read fails or a row cannot be converted.
func (c *core) GetTasksByID(ctx context.Context, ids []string) ([]*orchestrator_domain.Task, error) {
	tasks := make([]*orchestrator_domain.Task, 0, len(ids))
	for chunk := range slices.Chunk(ids, getTasksByIDChunkSize) {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("reading tasks by ID: %w", err)
		}

		rows, err := c.driver.GetTasksByID(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("reading tasks by ID: %w", err)
		}

		converted, err := convertTaskRowsToDomain(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, converted...)
	}
	return tasks, nil
}

// convertTaskRowsToDomain converts driver rows to domain tasks.
//
// Takes rows ([]TaskRow) which contains the rows to convert.
//
// Returns []*orchestrator_domain.Task which contains the converted domain tasks.
// Returns error when a row cannot be converted.
func convertTaskRowsToDomain(rows []TaskRow) ([]*orchestrator_domain.Task, error) {
	domainTasks := make([]*orchestrator_domain.Task, len(rows))
	for i := range rows {
		domainTask, err := convertDBTaskToDomain(&rows[i])
		if err != nil {
			return nil, fmt.Errorf("failed to convert task '%s' from DB model: %w", rows[i].ID, err)
		}
		domainTasks[i] = domainTask
	}
	return domainTasks, nil
}

// GetWorkflowStatus checks if all tasks in a workflow are complete by querying whether
// any non-terminal tasks remain.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns bool which is true when all tasks in the workflow are complete.
// Returns error when the workflow status cannot be determined.
func (c *core) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	hasIncomplete, err := c.driver.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		return false, fmt.Errorf("failed to get workflow status: %w", err)
	}

	return !hasIncomplete, nil
}

// PromoteScheduledTasks moves scheduled tasks that are ready to run to pending status.
//
// Returns int which is the number of tasks promoted.
// Returns error when the database transaction fails.
func (c *core) PromoteScheduledTasks(ctx context.Context) (int, error) {
	nowSeconds := c.clock.Now().UTC().Unix()

	var rowsAffected int64
	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		var txErr error
		rowsAffected, txErr = driver.PromoteScheduledTasks(ctx, nowSeconds, nowSeconds)
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
func (c *core) PendingTaskCount(ctx context.Context) (int64, error) {
	count, err := c.driver.PendingTaskCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("querying pending task count: %w", err)
	}
	return count, nil
}

// CreateWorkflowReceipt creates a new workflow receipt for tracking completion.
//
// Takes id (string) which is the unique identifier for the receipt.
// Takes workflowID (string) which is the workflow being tracked.
// Takes nodeID (string) which is the node that created the receipt.
//
// Returns error when the receipt cannot be created.
func (c *core) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	nowSeconds := c.clock.Now().UTC().Unix()

	return c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		err := driver.CreateWorkflowReceipt(ctx, CreateWorkflowReceiptParams{
			ID:         id,
			WorkflowID: workflowID,
			NodeID:     nodeID,
			CreatedAt:  nowSeconds,
			UpdatedAt:  nowSeconds,
		})
		if err != nil {
			return fmt.Errorf("creating workflow receipt: %w", err)
		}
		return nil
	})
}

// ResolveWorkflowReceipts marks all pending receipts for a workflow as resolved.
//
// Takes workflowID (string) which identifies the completed workflow.
// Takes errorMessage (string) which contains any error from workflow completion.
//
// Returns int which is the count of receipts resolved.
// Returns error when the resolution fails.
func (c *core) ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage string) (int, error) {
	nowSeconds := c.clock.Now().UTC().Unix()
	var errorMessagePtr *string
	if errorMessage != "" {
		errorMessagePtr = &errorMessage
	}

	var rowsAffected int64
	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		var txErr error
		rowsAffected, txErr = driver.ResolveWorkflowReceipts(ctx, workflowID, errorMessagePtr, nowSeconds)
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
// Returns []orchestrator_domain.PendingReceipt which contains the pending receipts for
// the node.
// Returns error when the database query fails.
func (c *core) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]orchestrator_domain.PendingReceipt, error) {
	rows, err := c.driver.GetPendingReceiptsByNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("getting pending receipts by node: %w", err)
	}

	result := make([]orchestrator_domain.PendingReceipt, len(rows))
	for i, row := range rows {
		result[i] = orchestrator_domain.PendingReceipt{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			NodeID:     nodeID,
			CreatedAt:  row.CreatedAt,
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
func (c *core) GetPendingReceiptsByWorkflow(ctx context.Context, workflowID string) ([]orchestrator_domain.PendingReceipt, error) {
	rows, err := c.driver.GetPendingReceiptsByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("getting pending receipts by workflow: %w", err)
	}

	result := make([]orchestrator_domain.PendingReceipt, len(rows))
	for i, row := range rows {
		result[i] = orchestrator_domain.PendingReceipt{
			ID:         row.ID,
			WorkflowID: row.WorkflowID,
			NodeID:     row.NodeID,
			CreatedAt:  row.CreatedAt,
		}
	}
	return result, nil
}

// CleanupOldResolvedReceipts deletes resolved receipts older than the specified time.
//
// Takes olderThan (time.Time) which is the cutoff for deletion.
//
// Returns int which is the count of receipts deleted.
// Returns error when the cleanup fails.
func (c *core) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	olderThanSeconds := olderThan.Unix()

	var rowsAffected int64
	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		var txErr error
		rowsAffected, txErr = driver.CleanupOldResolvedReceipts(ctx, olderThanSeconds)
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
func (c *core) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	nowSeconds := c.clock.Now().UTC().Unix()
	olderThanSeconds := olderThan.Unix()

	var rowsAffected int64
	err := c.runInTransaction(ctx, func(ctx context.Context, driver Driver) error {
		var txErr error
		rowsAffected, txErr = driver.TimeoutStaleReceipts(ctx, nowSeconds, olderThanSeconds)
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
func (c *core) ListFailedTasks(ctx context.Context) ([]*orchestrator_domain.Task, error) {
	rows, err := c.driver.ListFailedTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying failed tasks: %w", err)
	}
	tasks, err := convertTaskRowsToDomain(rows)
	return tasks, err
}

// ListTaskSummary returns task counts grouped by status.
//
// Returns []orchestrator_domain.TaskSummary which contains one entry per status with its
// count.
// Returns error when the database query fails.
func (c *core) ListTaskSummary(ctx context.Context) ([]orchestrator_domain.TaskSummary, error) {
	rows, err := c.driver.ListTaskStatusCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing task status counts: %w", err)
	}

	results := make([]orchestrator_domain.TaskSummary, len(rows))
	for i, row := range rows {
		results[i] = orchestrator_domain.TaskSummary{
			Status: row.Status,
			Count:  row.TaskCount,
		}
	}

	return results, nil
}

// ListRecentTasks returns the most recently updated tasks.
//
// Takes limit (int) which specifies the maximum number of tasks to return.
//
// Returns []orchestrator_domain.TaskListItem which contains the tasks ordered by update
// time descending.
// Returns error when the database query fails.
func (c *core) ListRecentTasks(ctx context.Context, limit int) ([]orchestrator_domain.TaskListItem, error) {
	rows, err := c.driver.ListRecentTasks(ctx, limit)
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
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
	}

	return results, nil
}

// ListWorkflowSummary returns workflow-level aggregates ordered by most recently updated.
//
// Takes limit (int) which specifies the maximum number of workflows to return.
//
// Returns []orchestrator_domain.WorkflowSummary which contains one entry per workflow
// with task counts by status.
// Returns error when the database query fails.
func (c *core) ListWorkflowSummary(ctx context.Context, limit int) ([]orchestrator_domain.WorkflowSummary, error) {
	rows, err := c.driver.ListWorkflowSummary(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing workflow summary: %w", err)
	}

	results := make([]orchestrator_domain.WorkflowSummary, len(rows))
	for i, row := range rows {
		results[i] = orchestrator_domain.WorkflowSummary{
			WorkflowID:    row.WorkflowID,
			TaskCount:     row.TaskCount,
			CompleteCount: derefInt64(row.CompleteCount),
			FailedCount:   derefInt64(row.FailedCount),
			ActiveCount:   derefInt64(row.ActiveCount),
			CreatedAt:     derefInt64(row.CreatedAt),
			UpdatedAt:     derefInt64(row.UpdatedAt),
		}
	}

	return results, nil
}

// WorkflowError reads the authoritative workflow failure, including child tasks.
//
// Takes workflowID (string) which identifies the workflow to inspect.
//
// Returns outcome (error) which is a persisted task failure, or nil when none is found.
// Returns readErr (error) which reports a query failure or an unsupported driver
// operation.
func (c *core) WorkflowError(ctx context.Context, workflowID string) (outcome error, readErr error) {
	if reader, ok := c.driver.(interface {
		WorkflowError(context.Context, string) (error, error)
	}); ok {
		return reader.WorkflowError(ctx, workflowID)
	}
	return nil, errors.New("driver does not support workflow outcome lookup")
}

// PendingDispatchPage reads due tasks for reconciliation after publication loss or
// restart.
//
// Takes afterID (string) which identifies the preceding page's last task, or is empty to
// start at the beginning.
// Takes afterTime (time.Time) which is the execution time of the preceding page's last
// task.
// Takes limit (int) which caps the page size, subject to maintenanceBatchSize.
//
// Returns []*orchestrator_domain.Task which contains pending or retrying tasks ordered by
// execution time and ID.
// Returns error when storage cannot be read.
func (c *core) PendingDispatchPage(ctx context.Context, afterID string, afterTime time.Time, limit int) ([]*orchestrator_domain.Task, error) {
	query := orchestrator_domain.DispatchQuery{
		Now:             c.clock.Now(),
		AfterTime:       time.Time{},
		PublishedBefore: time.Time{},
		AfterID:         afterID,
		Limit:           min(limit, maintenanceBatchSize),
	}
	if afterID != "" {
		query.AfterTime = afterTime
	}
	return c.DispatchablePage(ctx, query)
}

// derefInt64 returns the value behind a nullable int64 pointer, defaulting to zero when
// nil.
//
// Takes value (*int64) which may be nil.
//
// Returns int64 which is the dereferenced value.
func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

// runInTransaction runs fn in a transaction and retries transient failures.
//
// Nested calls reuse the current transaction and leave retries to the outermost call.
//
// Takes fn (func(context.Context, Driver) error) which is the callback executed inside
// the transaction.
//
// Returns error when the transaction fails to begin, fn returns an error, or the commit
// fails.
func (c *core) runInTransaction(ctx context.Context, fn func(ctx context.Context, driver Driver) error) error {
	if c.inTransaction {
		return fn(ctx, c.driver)
	}
	return c.retryTransient(ctx, func() error {
		return c.runInTransactionOnce(ctx, fn)
	})
}

// runInTransactionOnce runs fn in one new transaction.
//
// Takes fn (func(context.Context, Driver) error) which is the callback executed inside
// the transaction.
//
// Returns error when the transaction fails to begin, fn returns an error, or the commit
// fails.
func (c *core) runInTransactionOnce(ctx context.Context, fn func(ctx context.Context, driver Driver) error) error {
	db, err := c.beginTxDB()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("transaction rollback failed",
				logger_domain.Error(rollbackErr))
		}
	}()

	if err := fn(ctx, c.driver.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// beginTxDB resolves the *sql.DB needed to start a transaction.
//
// Returns *sql.DB which is the underlying database connection.
// Returns error when the core has no suitable database connection.
func (c *core) beginTxDB() (*sql.DB, error) {
	if c.sqlDB != nil {
		return c.sqlDB, nil
	}
	return nil, errDALNotInitialised
}

// withTransaction runs a function through a transaction-scoped core and rolls back before
// propagating a panic.
//
// Takes transactionFunction (func(context.Context, orchestrator_dal.OrchestratorDAL)
// error) which is the function to execute within the transaction scope.
//
// Returns error when the core has no database connection, when beginning the transaction
// fails, when transactionFunction returns an error, or when commit fails.
func (c *core) withTransaction(ctx context.Context, transactionFunction func(ctx context.Context, dal orchestrator_dal.OrchestratorDAL) error) error {
	if c.sqlDB == nil {
		return errDALNotInitialised
	}

	tx, err := c.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			_, l := logger_domain.From(ctx, log)
			l.Warn("transaction rollback failed",
				logger_domain.Error(rollbackErr))
		}
	}()

	txCore := &core{
		sqlDB:         c.sqlDB,
		driver:        c.driver.WithTx(tx),
		clock:         c.clock,
		inTransaction: true,
	}

	if err := transactionFunction(ctx, txCore); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// buildCreateTaskParams marshals task fields and builds database creation parameters.
//
// Takes task (*orchestrator_domain.Task) which provides the task data to convert.
// Takes now (time.Time) which is the timestamp recorded as the task's UpdatedAt.
//
// Returns CreateTaskParams which contains the database-ready task parameters.
// Returns error when the payload or config cannot be marshalled to JSON.
func buildCreateTaskParams(task *orchestrator_domain.Task, now time.Time) (CreateTaskParams, error) {
	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		return CreateTaskParams{}, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return CreateTaskParams{}, fmt.Errorf("failed to marshal task config: %w", err)
	}

	var deduplicationKey, lastError, result *string
	if task.DeduplicationKey != "" {
		deduplicationKey = &task.DeduplicationKey
	}
	if task.LastError != "" {
		lastError = &task.LastError
	}
	if task.Result != nil {
		resultBytes, err := json.Marshal(task.Result)
		if err != nil {
			return CreateTaskParams{}, fmt.Errorf("failed to marshal task result: %w", err)
		}
		result = new(string(resultBytes))
	}

	return CreateTaskParams{
		DeduplicationKey:   deduplicationKey,
		LastError:          lastError,
		Result:             result,
		ID:                 task.ID,
		WorkflowID:         task.WorkflowID,
		Executor:           task.Executor,
		Payload:            string(payloadBytes),
		Config:             string(configBytes),
		Status:             string(task.Status),
		BuildTag:           task.BuildTag,
		OwnerInstance:      task.OwnerInstance,
		ExecuteAt:          task.ExecuteAt.Unix(),
		ScheduledExecuteAt: unixOrZero(task.ScheduledExecuteAt),
		CreatedAt:          task.CreatedAt.Unix(),
		UpdatedAt:          now.Unix(),
		Priority:           safeconv.IntToInt32(int(task.Config.Priority)),
		Attempt:            safeconv.IntToInt32(task.Attempt),
		IsFatal:            task.IsFatal,
		Background:         task.Background,
	}, nil
}

// buildUpdateTaskParams converts a task into database update parameters.
//
// Takes task (*orchestrator_domain.Task) which provides the task data to convert.
// Takes now (time.Time) which is the timestamp recorded as the task's UpdatedAt.
//
// Returns UpdateTaskParams which contains the serialised task fields ready for database
// update.
// Returns error when marshalling the payload, config, or result fails.
func buildUpdateTaskParams(task *orchestrator_domain.Task, now time.Time) (UpdateTaskParams, error) {
	payloadBytes, err := json.Marshal(task.Payload)
	if err != nil {
		return UpdateTaskParams{}, fmt.Errorf("failed to marshal task payload: %w", err)
	}

	configBytes, err := json.Marshal(task.Config)
	if err != nil {
		return UpdateTaskParams{}, fmt.Errorf("failed to marshal task config: %w", err)
	}

	resultBytes, err := json.Marshal(task.Result)
	if err != nil {
		return UpdateTaskParams{}, fmt.Errorf("failed to marshal task result: %w", err)
	}

	var lastErrorPtr *string
	if task.LastError != "" {
		lastErrorPtr = &task.LastError
	}

	var resultPtr *string
	if task.Result != nil {
		resultPtr = new(string(resultBytes))
	}

	return UpdateTaskParams{
		Status:             string(task.Status),
		Priority:           safeconv.IntToInt32(int(task.Config.Priority)),
		ExecuteAt:          task.ExecuteAt.Unix(),
		Attempt:            safeconv.IntToInt32(task.Attempt),
		LastError:          lastErrorPtr,
		Result:             resultPtr,
		Payload:            string(payloadBytes),
		Config:             string(configBytes),
		UpdatedAt:          now.Unix(),
		ID:                 task.ID,
		BuildTag:           task.BuildTag,
		OwnerInstance:      task.OwnerInstance,
		ScheduledExecuteAt: unixOrZero(task.ScheduledExecuteAt),
		IsFatal:            task.IsFatal,
		Background:         task.Background,
	}, nil
}

// convertDBTaskToDomain converts a driver task row to a domain task.
//
// Takes dbTask (*TaskRow) which is the row to convert.
//
// Returns *orchestrator_domain.Task which is the populated domain task, owned by the
// caller.
// Returns error when JSON unmarshalling fails for payload, config, or result fields.
func convertDBTaskToDomain(dbTask *TaskRow) (*orchestrator_domain.Task, error) {
	task := &orchestrator_domain.Task{}

	task.ID = dbTask.ID
	task.WorkflowID = dbTask.WorkflowID
	task.Executor = dbTask.Executor
	task.Status = orchestrator_domain.TaskStatus(dbTask.Status)
	task.ExecuteAt = time.Unix(dbTask.ExecuteAt, 0).UTC()
	task.Attempt = int(dbTask.Attempt)
	task.CreatedAt = time.Unix(dbTask.CreatedAt, 0).UTC()
	task.UpdatedAt = time.Unix(dbTask.UpdatedAt, 0).UTC()
	task.ScheduledExecuteAt = timeOrZero(dbTask.ScheduledExecuteAt)
	task.LastPublishedAt = timeOrZero(dbTask.LastPublishedAt)
	task.BuildTag = dbTask.BuildTag
	task.OwnerInstance = dbTask.OwnerInstance
	task.IsFatal = dbTask.IsFatal
	task.Background = dbTask.Background

	if dbTask.LastError != nil {
		task.LastError = *dbTask.LastError
	}

	if dbTask.DeduplicationKey != nil {
		task.DeduplicationKey = *dbTask.DeduplicationKey
	}

	if err := json.UnmarshalString(dbTask.Payload, &task.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload for task '%s': %w", dbTask.ID, err)
	}
	if err := json.UnmarshalString(dbTask.Config, &task.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config for task '%s': %w", dbTask.ID, err)
	}

	task.Config.Priority = orchestrator_domain.TaskPriority(dbTask.Priority)

	if dbTask.Result != nil {
		if err := json.UnmarshalString(*dbTask.Result, &task.Result); err != nil {
			return nil, fmt.Errorf("unmarshal result for task '%s': %w", dbTask.ID, err)
		}
	}

	if dbTask.RerunID != nil {
		rerun, err := decodeRerun(dbTask)
		if err != nil {
			return nil, err
		}
		task.Rerun = rerun
	}

	return task, nil
}
