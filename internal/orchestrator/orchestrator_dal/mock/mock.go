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

package mock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// activeStatuses are the task statuses considered "active" for deduplication.
var activeStatuses = map[orchestrator_domain.TaskStatus]bool{
	orchestrator_domain.StatusScheduled:  true,
	orchestrator_domain.StatusPending:    true,
	orchestrator_domain.StatusProcessing: true,
	orchestrator_domain.StatusRetrying:   true,
}

// Behaviour defines settings that control how mock methods act during tests.
type Behaviour struct {
	// Error is the error to return instead of the normal result.
	Error error

	// PanicMessage is the custom message to use when panicking; empty uses a
	// default.
	PanicMessage string

	// Delay specifies how long to wait before running; 0 means no delay.
	Delay time.Duration

	// ShouldPanic indicates whether to cause a panic during mock execution.
	ShouldPanic bool
}

// CallRecord tracks method calls for verification.
type CallRecord struct {
	// Timestamp is when the function call was recorded.
	Timestamp time.Time

	// Method is the name of the DAL method that was called.
	Method string

	// Args contains the arguments passed to the function call.
	Args []any
}

// OrchestratorDAL is a mock implementation of OrchestratorDALWithTx for
// testing.
type OrchestratorDAL struct {
	// tasks stores all tasks indexed by their ID.
	tasks map[string]*orchestrator_domain.Task

	// tasksByWorkflow maps workflow IDs to their associated task IDs.
	tasksByWorkflow map[string][]string

	// behaviours maps method names to their test behaviours.
	behaviours map[string]*Behaviour

	// calls stores the record of method calls for test checking.
	calls []CallRecord

	// mu guards access to the behaviours and calls maps.
	mu sync.RWMutex

	// inTransaction indicates whether a simulated transaction is active.
	inTransaction bool
}

// NewOrchestratorDAL creates a new mock DAL instance.
//
// Returns *OrchestratorDAL which is a ready-to-use mock with empty collections.
func NewOrchestratorDAL() *OrchestratorDAL {
	return &OrchestratorDAL{
		tasks:           make(map[string]*orchestrator_domain.Task),
		tasksByWorkflow: make(map[string][]string),
		behaviours:      make(map[string]*Behaviour),
		calls:           make([]CallRecord, 0),
	}
}

// SetBehaviour configures mock behaviour for a specific method.
//
// Takes method (string) which is the name of the method to configure.
// Takes behaviour (*Behaviour) which defines the mock response for that method.
//
// Safe for concurrent use; protects access with a mutex.
func (m *OrchestratorDAL) SetBehaviour(method string, behaviour *Behaviour) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.behaviours[method] = behaviour
}

// ClearBehaviours removes all configured behaviours.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) ClearBehaviours() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.behaviours = make(map[string]*Behaviour)
}

// GetCalls returns all recorded method calls.
//
// Returns []CallRecord which is a copy of all calls made to the mock.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) GetCalls() []CallRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]CallRecord, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// ClearCalls removes all recorded calls.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) ClearCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]CallRecord, 0)
}

// GetCallCount returns the number of times a method was called.
//
// Takes method (string) which is the name of the method to count calls for.
//
// Returns int which is the number of recorded calls to the specified method.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, call := range m.calls {
		if call.Method == method {
			count++
		}
	}
	return count
}

// ClearData removes all stored data.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) ClearData() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = make(map[string]*orchestrator_domain.Task)
	m.tasksByWorkflow = make(map[string][]string)
}

// HealthCheck implements orchestrator_dal.OrchestratorDAL.
//
// Returns error when the health check behaviour is set to fail.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) HealthCheck(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("HealthCheck")
	return m.executeBehaviour("HealthCheck")
}

// Close releases resources held by the mock DAL.
// Implements orchestrator_dal.OrchestratorDAL.
//
// Returns error when the configured behaviour returns an error.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("Close")
	return m.executeBehaviour("Close")
}

// RunAtomic executes fn within a transaction.
//
// Takes fn (func(...) error) which is the function to execute
// within the transaction.
//
// Returns error when fn returns an error, or nil on success.
func (m *OrchestratorDAL) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
	return m.withTransaction(ctx, func(ctx context.Context, transactionDAL orchestrator_dal.OrchestratorDAL) error {
		store, ok := transactionDAL.(orchestrator_domain.TaskStore)
		if !ok {
			return errors.New("transaction DAL does not implement TaskStore")
		}
		return fn(ctx, store)
	})
}

// IsInTransaction returns whether the mock is currently in a transaction.
//
// Returns bool which is true if a transaction is active, false otherwise.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) IsInTransaction() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inTransaction
}

// CreateTask implements orchestrator_dal.TaskDAL.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to create.
//
// Returns error when the configured behaviour returns an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) CreateTask(_ context.Context, task *orchestrator_domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("CreateTask", task)
	if err := m.executeBehaviour("CreateTask"); err != nil {
		return err
	}

	m.storeTaskInternal(task)
	return nil
}

// CreateTasks implements orchestrator_dal.TaskDAL.
//
// Takes tasks ([]*orchestrator_domain.Task) which specifies the tasks to
// create.
//
// Returns error when the configured behaviour produces an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) CreateTasks(_ context.Context, tasks []*orchestrator_domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("CreateTasks", tasks)
	if err := m.executeBehaviour("CreateTasks"); err != nil {
		return err
	}

	for _, task := range tasks {
		m.storeTaskInternal(task)
	}
	return nil
}

// UpdateTask implements orchestrator_dal.TaskDAL.
//
// Takes task (*orchestrator_domain.Task) which is the task to update.
//
// Returns error when the task is not found or the configured behaviour fails.
//
// Safe for concurrent use. Protected by a mutex.
func (m *OrchestratorDAL) UpdateTask(_ context.Context, task *orchestrator_domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("UpdateTask", task)
	if err := m.executeBehaviour("UpdateTask"); err != nil {
		return err
	}

	if _, exists := m.tasks[task.ID]; !exists {
		return orchestrator_dal.ErrTaskNotFound
	}

	m.tasks[task.ID] = new(*task)
	return nil
}

// FetchAndMarkDueTasks implements orchestrator_dal.TaskDAL.
//
// Takes priority (orchestrator_domain.TaskPriority) which filters tasks to
// fetch only those at or above this priority level.
// Takes limit (int) which specifies the maximum number of tasks to return.
//
// Returns []*orchestrator_domain.Task which contains the tasks that were
// marked as processing.
// Returns error when a configured behaviour produces an error.
//
// Safe for concurrent use; protects internal state with a mutex.
func (m *OrchestratorDAL) FetchAndMarkDueTasks(_ context.Context, priority orchestrator_domain.TaskPriority, limit int) ([]*orchestrator_domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("FetchAndMarkDueTasks", priority, limit)
	if err := m.executeBehaviour("FetchAndMarkDueTasks"); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	results := make([]*orchestrator_domain.Task, 0, limit)

	for _, task := range m.tasks {
		if len(results) >= limit {
			break
		}

		if (task.Status == orchestrator_domain.StatusPending || task.Status == orchestrator_domain.StatusRetrying) &&
			task.Config.Priority >= priority &&
			!task.ExecuteAt.After(now) {
			task.Status = orchestrator_domain.StatusProcessing
			task.UpdatedAt = now

			results = append(results, new(*task))
		}
	}

	return results, nil
}

// GetWorkflowStatus checks if all tasks in a workflow have reached a terminal
// state. Implements orchestrator_dal.TaskDAL.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns bool which is true when all tasks are complete or failed.
// Returns error when the workflow does not exist or has no tasks.
//
// Safe for concurrent use; protected by a read lock.
func (m *OrchestratorDAL) GetWorkflowStatus(_ context.Context, workflowID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("GetWorkflowStatus", workflowID)
	if err := m.executeBehaviour("GetWorkflowStatus"); err != nil {
		return false, err
	}

	taskIDs, exists := m.tasksByWorkflow[workflowID]
	if !exists || len(taskIDs) == 0 {
		return false, orchestrator_dal.ErrWorkflowNotFound
	}

	for _, taskID := range taskIDs {
		task, exists := m.tasks[taskID]
		if !exists {
			continue
		}

		if task.Status != orchestrator_domain.StatusComplete &&
			task.Status != orchestrator_domain.StatusFailed {
			return false, nil
		}
	}

	return true, nil
}

// PromoteScheduledTasks implements orchestrator_dal.TaskDAL.
//
// Returns int which is the number of tasks promoted from scheduled to pending.
// Returns error when the configured behaviour fails.
//
// Safe for concurrent use; protects internal state with a mutex.
func (m *OrchestratorDAL) PromoteScheduledTasks(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("PromoteScheduledTasks")
	if err := m.executeBehaviour("PromoteScheduledTasks"); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	promoted := 0

	for _, task := range m.tasks {
		if task.Status == orchestrator_domain.StatusScheduled && !task.ExecuteAt.After(now) {
			task.Status = orchestrator_domain.StatusPending
			task.UpdatedAt = now
			promoted++
		}
	}

	return promoted, nil
}

// PendingTaskCount returns the count of tasks with pending status.
// Implements orchestrator_dal.TaskDAL.
//
// Returns int64 which is the number of pending tasks.
// Returns error when a configured behaviour fails.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) PendingTaskCount(_ context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("PendingTaskCount")
	if err := m.executeBehaviour("PendingTaskCount"); err != nil {
		return 0, err
	}

	var count int64
	for _, task := range m.tasks {
		if task.Status == orchestrator_domain.StatusPending {
			count++
		}
	}

	return count, nil
}

// SetTask stores a task directly in the mock for testing.
//
// Takes task (*orchestrator_domain.Task) which is the task to store.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) SetTask(task *orchestrator_domain.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storeTaskInternal(task)
}

// GetTask is a test helper to retrieve a task from the mock.
//
// Takes taskID (string) which is the ID of the task to retrieve.
//
// Returns *orchestrator_domain.Task which is a copy of the task if found.
// Returns bool which indicates whether the task exists.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) GetTask(taskID string) (*orchestrator_domain.Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, exists := m.tasks[taskID]
	if !exists {
		return nil, false
	}
	return new(*task), true
}

// GetTaskCount returns the number of tasks in the mock.
//
// Returns int which is the current count of tasks.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) GetTaskCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tasks)
}

// GetTasksByWorkflow returns all tasks for a workflow.
//
// Takes workflowID (string) which identifies the workflow to retrieve tasks
// for.
//
// Returns []*orchestrator_domain.Task which contains copies of all tasks
// belonging to the workflow, or nil if the workflow does not exist.
//
// Safe for concurrent use; acquires a read lock on the internal data store.
func (m *OrchestratorDAL) GetTasksByWorkflow(workflowID string) []*orchestrator_domain.Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	taskIDs, exists := m.tasksByWorkflow[workflowID]
	if !exists {
		return nil
	}

	tasks := make([]*orchestrator_domain.Task, 0, len(taskIDs))
	for _, taskID := range taskIDs {
		if task, exists := m.tasks[taskID]; exists {
			tasks = append(tasks, new(*task))
		}
	}

	return tasks
}

// GetTasksByStatus returns all tasks with a specific status.
//
// Takes status (orchestrator_domain.TaskStatus) which filters tasks by their
// current status.
//
// Returns []*orchestrator_domain.Task which contains copies of all matching
// tasks.
//
// Safe for concurrent use. Uses a read lock to protect access to the task map.
func (m *OrchestratorDAL) GetTasksByStatus(status orchestrator_domain.TaskStatus) []*orchestrator_domain.Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*orchestrator_domain.Task, 0)
	for _, task := range m.tasks {
		if task.Status == status {
			tasks = append(tasks, new(*task))
		}
	}

	return tasks
}

// CreateTaskWithDedup implements orchestrator_dal.TaskDAL and creates a
// task with deduplication support. If the task has a DeduplicationKey set
// and an active task with the same key exists, returns ErrDuplicateTask.
//
// Takes task (*orchestrator_domain.Task) which specifies the task to create.
//
// Returns error when a duplicate exists or the configured behaviour returns
// an error.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) CreateTaskWithDedup(_ context.Context, task *orchestrator_domain.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("CreateTaskWithDedup", task)
	if err := m.executeBehaviour("CreateTaskWithDedup"); err != nil {
		return err
	}

	if task.DeduplicationKey != "" {
		if m.hasActiveDuplicate(task.DeduplicationKey) {
			return orchestrator_domain.ErrDuplicateTask
		}
	}

	m.storeTaskInternal(task)
	return nil
}

// RecoverStaleTasks implements orchestrator_dal.TaskDAL. It resets PROCESSING
// tasks that have exceeded the stale threshold.
//
// Takes staleThreshold (time.Duration) which defines how long a task can be in
// PROCESSING before being considered stuck.
// Takes maxRetries (int) which is the maximum retry attempts before marking
// FAILED.
// Takes recoveryError (string) which is the error message to record on
// recovered tasks.
//
// Returns int which is the count of tasks recovered.
// Returns error when the configured behaviour returns an error.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) RecoverStaleTasks(_ context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("RecoverStaleTasks", staleThreshold, maxRetries, recoveryError)
	if err := m.executeBehaviour("RecoverStaleTasks"); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	staleThresholdTime := now.Add(-staleThreshold)
	recovered := 0

	for _, task := range m.tasks {
		if task.Status == orchestrator_domain.StatusProcessing && task.UpdatedAt.Before(staleThresholdTime) {
			if task.Attempt >= maxRetries {
				task.Status = orchestrator_domain.StatusFailed
			} else {
				task.Status = orchestrator_domain.StatusRetrying
				task.Attempt++
			}
			task.LastError = recoveryError
			task.UpdatedAt = now
			task.ExecuteAt = now
			recovered++
		}
	}

	return recovered, nil
}

// GetStaleProcessingTaskCount implements orchestrator_dal.TaskDAL.
// It returns the count of tasks stuck in PROCESSING longer than the threshold.
//
// Takes staleThreshold (time.Duration) which defines when a PROCESSING task is
// considered stuck.
//
// Returns int64 which is the count of stale tasks.
// Returns error when the configured behaviour returns an error.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) GetStaleProcessingTaskCount(_ context.Context, staleThreshold time.Duration) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.recordCall("GetStaleProcessingTaskCount", staleThreshold)
	if err := m.executeBehaviour("GetStaleProcessingTaskCount"); err != nil {
		return 0, err
	}

	staleThresholdTime := time.Now().UTC().Add(-staleThreshold)
	var count int64

	for _, task := range m.tasks {
		if task.Status == orchestrator_domain.StatusProcessing && task.UpdatedAt.Before(staleThresholdTime) {
			count++
		}
	}

	return count, nil
}

// UpdateTaskHeartbeat updates the updated_at timestamp for a task in
// PROCESSING status.
//
// Takes taskID (string) which identifies the task to update.
//
// Returns error when a configured behaviour returns an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) UpdateTaskHeartbeat(_ context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("UpdateTaskHeartbeat", taskID)
	if err := m.executeBehaviour("UpdateTaskHeartbeat"); err != nil {
		return err
	}

	if task, exists := m.tasks[taskID]; exists && task.Status == orchestrator_domain.StatusProcessing {
		task.UpdatedAt = time.Now().UTC()
	}

	return nil
}

// ClaimStaleTasksForRecovery atomically claims stale PROCESSING tasks for
// recovery.
//
// Returns []orchestrator_domain.RecoveryClaimedTask which contains the claimed
// tasks, or nil if no stale tasks are found.
// Returns error when the configured mock behaviour returns an error.
//
// Safe for concurrent use; guards internal state with a mutex.
func (m *OrchestratorDAL) ClaimStaleTasksForRecovery(_ context.Context, _ string, _ time.Duration, _ time.Duration, _ int) ([]orchestrator_domain.RecoveryClaimedTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("ClaimStaleTasksForRecovery")
	if err := m.executeBehaviour("ClaimStaleTasksForRecovery"); err != nil {
		return nil, err
	}

	return nil, nil
}

// RecoverClaimedTasks recovers all tasks previously claimed by this node.
//
// Returns int which is the number of tasks recovered.
// Returns error when behaviour execution fails.
//
// Safe for concurrent use; method is protected by a mutex.
func (m *OrchestratorDAL) RecoverClaimedTasks(_ context.Context, _ string, _ int, _ string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("RecoverClaimedTasks")
	if err := m.executeBehaviour("RecoverClaimedTasks"); err != nil {
		return 0, err
	}

	return 0, nil
}

// ReleaseRecoveryLeases releases all recovery leases held by this node.
//
// Returns int which is the number of leases released.
// Returns error when the operation fails.
//
// Safe for concurrent use; protects internal state with a mutex.
func (m *OrchestratorDAL) ReleaseRecoveryLeases(_ context.Context, _ string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("ReleaseRecoveryLeases")
	if err := m.executeBehaviour("ReleaseRecoveryLeases"); err != nil {
		return 0, err
	}

	return 0, nil
}

// CreateWorkflowReceipt creates a new workflow receipt for tracking completion.
//
// Returns error when the configured behaviour fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) CreateWorkflowReceipt(_ context.Context, _, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("CreateWorkflowReceipt")
	return m.executeBehaviour("CreateWorkflowReceipt")
}

// ResolveWorkflowReceipts marks all pending receipts for a workflow as
// resolved.
//
// Returns int which is the number of receipts resolved.
// Returns error when the operation fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) ResolveWorkflowReceipts(_ context.Context, _ string, _ string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("ResolveWorkflowReceipts")
	if err := m.executeBehaviour("ResolveWorkflowReceipts"); err != nil {
		return 0, err
	}

	return 0, nil
}

// GetPendingReceiptsByNode retrieves all pending receipts created by a node.
//
// Returns []orchestrator_domain.PendingReceipt which contains the pending
// receipts for the specified node.
// Returns error when the configured behaviour produces an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) GetPendingReceiptsByNode(_ context.Context, _ string) ([]orchestrator_domain.PendingReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("GetPendingReceiptsByNode")
	if err := m.executeBehaviour("GetPendingReceiptsByNode"); err != nil {
		return nil, err
	}

	return nil, nil
}

// GetPendingReceiptsByWorkflow retrieves all pending receipts for a workflow.
//
// Returns []orchestrator_domain.PendingReceipt which contains the pending
// receipts for the workflow.
// Returns error when the configured behaviour fails.
//
// Safe for concurrent use; access is protected by a mutex.
func (m *OrchestratorDAL) GetPendingReceiptsByWorkflow(_ context.Context, _ string) ([]orchestrator_domain.PendingReceipt, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("GetPendingReceiptsByWorkflow")
	if err := m.executeBehaviour("GetPendingReceiptsByWorkflow"); err != nil {
		return nil, err
	}

	return nil, nil
}

// CleanupOldResolvedReceipts deletes resolved receipts older than the
// specified time.
//
// Returns int which is the number of receipts deleted.
// Returns error when the cleanup operation fails.
//
// Safe for concurrent use.
func (m *OrchestratorDAL) CleanupOldResolvedReceipts(_ context.Context, _ time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("CleanupOldResolvedReceipts")
	if err := m.executeBehaviour("CleanupOldResolvedReceipts"); err != nil {
		return 0, err
	}

	return 0, nil
}

// TimeoutStaleReceipts marks very old pending receipts as timed out.
//
// Returns int which is the number of receipts marked as timed out.
// Returns error when the behaviour execution fails.
//
// Safe for concurrent use; takes a mutex for the call duration.
func (m *OrchestratorDAL) TimeoutStaleReceipts(_ context.Context, _ time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("TimeoutStaleReceipts")
	if err := m.executeBehaviour("TimeoutStaleReceipts"); err != nil {
		return 0, err
	}

	return 0, nil
}

// ListFailedTasks returns all tasks with a FAILED status.
//
// Returns []*orchestrator_domain.Task which contains the failed tasks.
// Returns error when the configured behaviour returns an error.
//
// Safe for concurrent use; protected by a mutex.
func (m *OrchestratorDAL) ListFailedTasks(_ context.Context) ([]*orchestrator_domain.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordCall("ListFailedTasks")
	if err := m.executeBehaviour("ListFailedTasks"); err != nil {
		return nil, err
	}
	return m.GetTasksByStatus(orchestrator_domain.StatusFailed), nil
}

// withTransaction is an internal helper used by RunAtomic.
//
// Takes operation (func(...)) which is the function to execute within the
// transaction.
//
// Returns error when the transaction function fails or when a configured
// mock behaviour returns an error.
//
// Safe for concurrent use. The function executes within a transaction-isolated
// copy of the mock. Changes are committed only if operation
// succeeds; otherwise they are discarded.
func (m *OrchestratorDAL) withTransaction(ctx context.Context, operation func(ctx context.Context, dal orchestrator_dal.OrchestratorDAL) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.recordCall("withTransaction", operation)
	if err := m.executeBehaviour("withTransaction"); err != nil {
		return err
	}

	txMock := m.createTransactionCopy()

	err := operation(ctx, txMock)
	if err != nil {
		return err
	}

	m.commitTransaction(txMock)

	return nil
}

// recordCall logs a method call for verification.
//
// Takes method (string) which specifies the name of the method being called.
// Takes arguments (...any) which provides the arguments passed to the method.
func (m *OrchestratorDAL) recordCall(method string, arguments ...any) {
	m.calls = append(m.calls, CallRecord{
		Method:    method,
		Args:      arguments,
		Timestamp: time.Now(),
	})
}

// executeBehaviour checks for configured behaviour and executes it.
//
// Takes method (string) which identifies the method whose behaviour to execute.
//
// Returns error when the configured behaviour specifies an error to return.
//
// Panics if the configured behaviour has ShouldPanic set to true.
func (m *OrchestratorDAL) executeBehaviour(method string) error {
	if behaviour, exists := m.behaviours[method]; exists {
		if behaviour.Delay > 0 {
			time.Sleep(behaviour.Delay)
		}

		if behaviour.ShouldPanic {
			message := behaviour.PanicMessage
			if message == "" {
				message = fmt.Sprintf("Mock panic in %s", method)
			}
			panic(message)
		}

		if behaviour.Error != nil {
			return behaviour.Error
		}
	}
	return nil
}

// createTransactionCopy creates a deep copy of the mock for transaction
// isolation.
//
// Returns *OrchestratorDAL which is an isolated copy for use within a
// transaction.
func (m *OrchestratorDAL) createTransactionCopy() *OrchestratorDAL {
	txMock := &OrchestratorDAL{
		tasks:           make(map[string]*orchestrator_domain.Task),
		tasksByWorkflow: make(map[string][]string),
		behaviours:      make(map[string]*Behaviour),
		calls:           make([]CallRecord, 0),
		inTransaction:   true,
	}

	for k, v := range m.tasks {
		txMock.tasks[k] = new(*v)
	}

	for k, v := range m.tasksByWorkflow {
		txMock.tasksByWorkflow[k] = append([]string{}, v...)
	}

	for k, v := range m.behaviours {
		txMock.behaviours[k] = new(*v)
	}

	return txMock
}

// commitTransaction applies transaction changes back to the main mock.
//
// Takes txMock (*OrchestratorDAL) which contains the transaction state to
// merge.
func (m *OrchestratorDAL) commitTransaction(txMock *OrchestratorDAL) {
	m.tasks = txMock.tasks
	m.tasksByWorkflow = txMock.tasksByWorkflow

	m.calls = append(m.calls, txMock.calls...)
}

// storeTaskInternal stores a task without acquiring a lock.
//
// Takes task (*orchestrator_domain.Task) which is the task to store.
func (m *OrchestratorDAL) storeTaskInternal(task *orchestrator_domain.Task) {
	m.tasks[task.ID] = new(*task)

	m.tasksByWorkflow[task.WorkflowID] = append(m.tasksByWorkflow[task.WorkflowID], task.ID)
}

// hasActiveDuplicate checks if an active task with the given deduplication
// key exists. Must be called with lock held.
//
// Takes dedupKey (string) which identifies the task to check for duplicates.
//
// Returns bool which is true if an active task with the key exists.
func (m *OrchestratorDAL) hasActiveDuplicate(dedupKey string) bool {
	for _, existingTask := range m.tasks {
		if existingTask.DeduplicationKey == dedupKey && activeStatuses[existingTask.Status] {
			return true
		}
	}
	return false
}

var _ orchestrator_dal.OrchestratorDALWithTx = (*OrchestratorDAL)(nil)
