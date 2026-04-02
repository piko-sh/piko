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
	"sync/atomic"
	"time"
)

// MockTaskStore is a test double for TaskStore where nil function fields
// return zero values and call counts are tracked atomically.
type MockTaskStore struct {
	// CreateTaskFunc is the function called by
	// CreateTask.
	CreateTaskFunc func(ctx context.Context, task *Task) error

	// CreateTasksFunc is the function called by
	// CreateTasks.
	CreateTasksFunc func(ctx context.Context, tasks []*Task) error

	// UpdateTaskFunc is the function called by
	// UpdateTask.
	UpdateTaskFunc func(ctx context.Context, task *Task) error

	// FetchAndMarkDueTasksFunc is the function called
	// by FetchAndMarkDueTasks.
	FetchAndMarkDueTasksFunc func(ctx context.Context, priority TaskPriority, limit int) ([]*Task, error)

	// GetWorkflowStatusFunc is the function called by
	// GetWorkflowStatus.
	GetWorkflowStatusFunc func(ctx context.Context, workflowID string) (bool, error)

	// PromoteScheduledTasksFunc is the function called
	// by PromoteScheduledTasks.
	PromoteScheduledTasksFunc func(ctx context.Context) (int, error)

	// PendingTaskCountFunc is the function called by
	// PendingTaskCount.
	PendingTaskCountFunc func(ctx context.Context) (int64, error)

	// CreateTaskWithDedupFunc is the function called
	// by CreateTaskWithDedup.
	CreateTaskWithDedupFunc func(ctx context.Context, task *Task) error

	// RecoverStaleTasksFunc is the function called by
	// RecoverStaleTasks.
	RecoverStaleTasksFunc func(ctx context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error)

	// GetStaleProcessingTaskCountFunc is the function
	// called by GetStaleProcessingTaskCount.
	GetStaleProcessingTaskCountFunc func(ctx context.Context, staleThreshold time.Duration) (int64, error)

	// UpdateTaskHeartbeatFunc is the function called
	// by UpdateTaskHeartbeat.
	UpdateTaskHeartbeatFunc func(ctx context.Context, taskID string) error

	// ClaimStaleTasksForRecoveryFunc is the function
	// called by ClaimStaleTasksForRecovery.
	ClaimStaleTasksForRecoveryFunc func(ctx context.Context, nodeID string, staleThreshold time.Duration, leaseTimeout time.Duration, batchLimit int) ([]RecoveryClaimedTask, error)

	// RecoverClaimedTasksFunc is the function called
	// by RecoverClaimedTasks.
	RecoverClaimedTasksFunc func(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error)

	// ReleaseRecoveryLeasesFunc is the function called
	// by ReleaseRecoveryLeases.
	ReleaseRecoveryLeasesFunc func(ctx context.Context, nodeID string) (int, error)

	// CreateWorkflowReceiptFunc is the function called
	// by CreateWorkflowReceipt.
	CreateWorkflowReceiptFunc func(ctx context.Context, id, workflowID, nodeID string) error

	// ResolveWorkflowReceiptsFunc is the function
	// called by ResolveWorkflowReceipts.
	ResolveWorkflowReceiptsFunc func(ctx context.Context, workflowID string, errorMessage string) (int, error)

	// GetPendingReceiptsByNodeFunc is the function
	// called by GetPendingReceiptsByNode.
	GetPendingReceiptsByNodeFunc func(ctx context.Context, nodeID string) ([]PendingReceipt, error)

	// CleanupOldResolvedReceiptsFunc is the function
	// called by CleanupOldResolvedReceipts.
	CleanupOldResolvedReceiptsFunc func(ctx context.Context, olderThan time.Time) (int, error)

	// TimeoutStaleReceiptsFunc is the function called
	// by TimeoutStaleReceipts.
	TimeoutStaleReceiptsFunc func(ctx context.Context, olderThan time.Time) (int, error)

	// ListFailedTasksFunc is the function called by
	// ListFailedTasks.
	ListFailedTasksFunc func(ctx context.Context) ([]*Task, error)

	// RunAtomicFunc is the function called by RunAtomic.
	RunAtomicFunc func(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error

	// CreateTaskCallCount tracks how many times
	// CreateTask was called.
	CreateTaskCallCount int64

	// CreateTasksCallCount tracks how many times
	// CreateTasks was called.
	CreateTasksCallCount int64

	// UpdateTaskCallCount tracks how many times
	// UpdateTask was called.
	UpdateTaskCallCount int64

	// FetchAndMarkDueTasksCallCount tracks how many
	// times FetchAndMarkDueTasks was called.
	FetchAndMarkDueTasksCallCount int64

	// GetWorkflowStatusCallCount tracks how many times
	// GetWorkflowStatus was called.
	GetWorkflowStatusCallCount int64

	// PromoteScheduledTasksCallCount tracks how many
	// times PromoteScheduledTasks was called.
	PromoteScheduledTasksCallCount int64

	// PendingTaskCountCallCount tracks how many times
	// PendingTaskCount was called.
	PendingTaskCountCallCount int64

	// CreateTaskWithDedupCallCount tracks how many
	// times CreateTaskWithDedup was called.
	CreateTaskWithDedupCallCount int64

	// RecoverStaleTasksCallCount tracks how many times
	// RecoverStaleTasks was called.
	RecoverStaleTasksCallCount int64

	// GetStaleProcessingTaskCountCallCount tracks how
	// many times GetStaleProcessingTaskCount was
	// called.
	GetStaleProcessingTaskCountCallCount int64

	// UpdateTaskHeartbeatCallCount tracks how many
	// times UpdateTaskHeartbeat was called.
	UpdateTaskHeartbeatCallCount int64

	// ClaimStaleTasksForRecoveryCallCount tracks how
	// many times ClaimStaleTasksForRecovery was
	// called.
	ClaimStaleTasksForRecoveryCallCount int64

	// RecoverClaimedTasksCallCount tracks how many
	// times RecoverClaimedTasks was called.
	RecoverClaimedTasksCallCount int64

	// ReleaseRecoveryLeasesCallCount tracks how many
	// times ReleaseRecoveryLeases was called.
	ReleaseRecoveryLeasesCallCount int64

	// CreateWorkflowReceiptCallCount tracks how many
	// times CreateWorkflowReceipt was called.
	CreateWorkflowReceiptCallCount int64

	// ResolveWorkflowReceiptsCallCount tracks how many
	// times ResolveWorkflowReceipts was called.
	ResolveWorkflowReceiptsCallCount int64

	// GetPendingReceiptsByNodeCallCount tracks how
	// many times GetPendingReceiptsByNode was called.
	GetPendingReceiptsByNodeCallCount int64

	// CleanupOldResolvedReceiptsCallCount tracks how
	// many times CleanupOldResolvedReceipts was
	// called.
	CleanupOldResolvedReceiptsCallCount int64

	// TimeoutStaleReceiptsCallCount tracks how many
	// times TimeoutStaleReceipts was called.
	TimeoutStaleReceiptsCallCount int64

	// ListFailedTasksCallCount tracks how many times
	// ListFailedTasks was called.
	ListFailedTasksCallCount int64

	// RunAtomicCallCount tracks how many times
	// RunAtomic was called.
	RunAtomicCallCount int64
}

// CreateTask persists a new task.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to create.
//
// Returns error, or nil if CreateTaskFunc is nil.
func (m *MockTaskStore) CreateTask(ctx context.Context, task *Task) error {
	atomic.AddInt64(&m.CreateTaskCallCount, 1)
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, task)
	}
	return nil
}

// CreateTasks persists multiple tasks in a batch.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes tasks ([]*Task) which is the batch of tasks to create.
//
// Returns error, or nil if CreateTasksFunc is nil.
func (m *MockTaskStore) CreateTasks(ctx context.Context, tasks []*Task) error {
	atomic.AddInt64(&m.CreateTasksCallCount, 1)
	if m.CreateTasksFunc != nil {
		return m.CreateTasksFunc(ctx, tasks)
	}
	return nil
}

// UpdateTask updates an existing task.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to update.
//
// Returns error, or nil if UpdateTaskFunc is nil.
func (m *MockTaskStore) UpdateTask(ctx context.Context, task *Task) error {
	atomic.AddInt64(&m.UpdateTaskCallCount, 1)
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(ctx, task)
	}
	return nil
}

// FetchAndMarkDueTasks atomically fetches and marks tasks as processing.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes priority (TaskPriority) which filters tasks by priority level.
// Takes limit (int) which caps the number of tasks fetched.
//
// Returns ([]*Task, error), or (nil, nil) if FetchAndMarkDueTasksFunc is nil.
func (m *MockTaskStore) FetchAndMarkDueTasks(ctx context.Context, priority TaskPriority, limit int) ([]*Task, error) {
	atomic.AddInt64(&m.FetchAndMarkDueTasksCallCount, 1)
	if m.FetchAndMarkDueTasksFunc != nil {
		return m.FetchAndMarkDueTasksFunc(ctx, priority, limit)
	}
	return nil, nil
}

// GetWorkflowStatus checks whether a workflow has completed.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns (bool, error), or (false, nil) if GetWorkflowStatusFunc is nil.
func (m *MockTaskStore) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	atomic.AddInt64(&m.GetWorkflowStatusCallCount, 1)
	if m.GetWorkflowStatusFunc != nil {
		return m.GetWorkflowStatusFunc(ctx, workflowID)
	}
	return false, nil
}

// PromoteScheduledTasks moves scheduled tasks that are due to the pending
// queue.
//
// Returns (int, error), or (0, nil) if PromoteScheduledTasksFunc is nil.
func (m *MockTaskStore) PromoteScheduledTasks(ctx context.Context) (int, error) {
	atomic.AddInt64(&m.PromoteScheduledTasksCallCount, 1)
	if m.PromoteScheduledTasksFunc != nil {
		return m.PromoteScheduledTasksFunc(ctx)
	}
	return 0, nil
}

// PendingTaskCount returns the number of tasks awaiting processing.
//
// Returns (int64, error), or (0, nil) if PendingTaskCountFunc is nil.
func (m *MockTaskStore) PendingTaskCount(ctx context.Context) (int64, error) {
	atomic.AddInt64(&m.PendingTaskCountCallCount, 1)
	if m.PendingTaskCountFunc != nil {
		return m.PendingTaskCountFunc(ctx)
	}
	return 0, nil
}

// CreateTaskWithDedup creates a task with deduplication.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to create with deduplication.
//
// Returns error, or nil if CreateTaskWithDedupFunc is nil.
func (m *MockTaskStore) CreateTaskWithDedup(ctx context.Context, task *Task) error {
	atomic.AddInt64(&m.CreateTaskWithDedupCallCount, 1)
	if m.CreateTaskWithDedupFunc != nil {
		return m.CreateTaskWithDedupFunc(ctx, task)
	}
	return nil
}

// RecoverStaleTasks requeues tasks stuck in processing beyond the threshold.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes staleThreshold (time.Duration) which is the
// age after which a task is considered stale.
// Takes maxRetries (int) which caps the number of
// retry attempts.
// Takes recoveryError (string) which is the error
// message to record on recovery.
//
// Returns (int, error), or (0, nil) if
// RecoverStaleTasksFunc is nil.
func (m *MockTaskStore) RecoverStaleTasks(ctx context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	atomic.AddInt64(&m.RecoverStaleTasksCallCount, 1)
	if m.RecoverStaleTasksFunc != nil {
		return m.RecoverStaleTasksFunc(ctx, staleThreshold, maxRetries, recoveryError)
	}
	return 0, nil
}

// GetStaleProcessingTaskCount returns the count of stale processing tasks.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes staleThreshold (time.Duration) which is the
// age after which a task is considered stale.
//
// Returns (int64, error), or (0, nil) if
// GetStaleProcessingTaskCountFunc
// is nil.
func (m *MockTaskStore) GetStaleProcessingTaskCount(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	atomic.AddInt64(&m.GetStaleProcessingTaskCountCallCount, 1)
	if m.GetStaleProcessingTaskCountFunc != nil {
		return m.GetStaleProcessingTaskCountFunc(ctx, staleThreshold)
	}
	return 0, nil
}

// UpdateTaskHeartbeat refreshes the heartbeat timestamp for a task.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes taskID (string) which identifies the task to heartbeat.
//
// Returns error, or nil if UpdateTaskHeartbeatFunc is nil.
func (m *MockTaskStore) UpdateTaskHeartbeat(ctx context.Context, taskID string) error {
	atomic.AddInt64(&m.UpdateTaskHeartbeatCallCount, 1)
	if m.UpdateTaskHeartbeatFunc != nil {
		return m.UpdateTaskHeartbeatFunc(ctx, taskID)
	}
	return nil
}

// ClaimStaleTasksForRecovery claims stale tasks for recovery by a node.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes nodeID (string) which identifies the node claiming the tasks.
// Takes staleThreshold (time.Duration) which is the
// age after which a task is considered stale.
// Takes leaseTimeout (time.Duration) which is the
// duration of the recovery lease.
// Takes batchLimit (int) which caps the number of tasks claimed.
//
// Returns ([]RecoveryClaimedTask, error), or (nil, nil) if
// ClaimStaleTasksForRecoveryFunc is nil.
func (m *MockTaskStore) ClaimStaleTasksForRecovery(ctx context.Context, nodeID string, staleThreshold time.Duration, leaseTimeout time.Duration, batchLimit int) ([]RecoveryClaimedTask, error) {
	atomic.AddInt64(&m.ClaimStaleTasksForRecoveryCallCount, 1)
	if m.ClaimStaleTasksForRecoveryFunc != nil {
		return m.ClaimStaleTasksForRecoveryFunc(ctx, nodeID, staleThreshold, leaseTimeout, batchLimit)
	}
	return nil, nil
}

// RecoverClaimedTasks processes previously claimed recovery tasks.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes nodeID (string) which identifies the node that claimed the tasks.
// Takes maxRetries (int) which caps the number of retry attempts.
// Takes recoveryError (string) which is the error
// message to record on recovery.
//
// Returns (int, error), or (0, nil) if
// RecoverClaimedTasksFunc is nil.
func (m *MockTaskStore) RecoverClaimedTasks(ctx context.Context, nodeID string, maxRetries int, recoveryError string) (int, error) {
	atomic.AddInt64(&m.RecoverClaimedTasksCallCount, 1)
	if m.RecoverClaimedTasksFunc != nil {
		return m.RecoverClaimedTasksFunc(ctx, nodeID, maxRetries, recoveryError)
	}
	return 0, nil
}

// ReleaseRecoveryLeases releases all recovery leases held by a node.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes nodeID (string) which identifies the node whose leases to release.
//
// Returns (int, error), or (0, nil) if ReleaseRecoveryLeasesFunc is nil.
func (m *MockTaskStore) ReleaseRecoveryLeases(ctx context.Context, nodeID string) (int, error) {
	atomic.AddInt64(&m.ReleaseRecoveryLeasesCallCount, 1)
	if m.ReleaseRecoveryLeasesFunc != nil {
		return m.ReleaseRecoveryLeasesFunc(ctx, nodeID)
	}
	return 0, nil
}

// CreateWorkflowReceipt records a workflow receipt for tracking.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes id (string) which identifies the receipt.
// Takes workflowID (string) which identifies the workflow.
// Takes nodeID (string) which identifies the originating node.
//
// Returns error, or nil if CreateWorkflowReceiptFunc is nil.
func (m *MockTaskStore) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	atomic.AddInt64(&m.CreateWorkflowReceiptCallCount, 1)
	if m.CreateWorkflowReceiptFunc != nil {
		return m.CreateWorkflowReceiptFunc(ctx, id, workflowID, nodeID)
	}
	return nil
}

// ResolveWorkflowReceipts resolves all receipts for a workflow.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes workflowID (string) which identifies the workflow to resolve.
// Takes errorMessage (string) which is the error
// message to record, or empty on success.
//
// Returns (int, error), or (0, nil) if ResolveWorkflowReceiptsFunc is nil.
func (m *MockTaskStore) ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage string) (int, error) {
	atomic.AddInt64(&m.ResolveWorkflowReceiptsCallCount, 1)
	if m.ResolveWorkflowReceiptsFunc != nil {
		return m.ResolveWorkflowReceiptsFunc(ctx, workflowID, errorMessage)
	}
	return 0, nil
}

// GetPendingReceiptsByNode returns unresolved receipts for a node.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes nodeID (string) which identifies the node to query.
//
// Returns ([]PendingReceipt, error), or (nil, nil) if
// GetPendingReceiptsByNodeFunc is nil.
func (m *MockTaskStore) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]PendingReceipt, error) {
	atomic.AddInt64(&m.GetPendingReceiptsByNodeCallCount, 1)
	if m.GetPendingReceiptsByNodeFunc != nil {
		return m.GetPendingReceiptsByNodeFunc(ctx, nodeID)
	}
	return nil, nil
}

// CleanupOldResolvedReceipts removes resolved receipts older than the given
// time.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes olderThan (time.Time) which is the cutoff time for cleanup.
//
// Returns (int, error), or (0, nil) if CleanupOldResolvedReceiptsFunc is nil.
func (m *MockTaskStore) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	atomic.AddInt64(&m.CleanupOldResolvedReceiptsCallCount, 1)
	if m.CleanupOldResolvedReceiptsFunc != nil {
		return m.CleanupOldResolvedReceiptsFunc(ctx, olderThan)
	}
	return 0, nil
}

// TimeoutStaleReceipts marks stale receipts as timed out.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes olderThan (time.Time) which is the cutoff time for timeout.
//
// Returns (int, error), or (0, nil) if TimeoutStaleReceiptsFunc is nil.
func (m *MockTaskStore) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	atomic.AddInt64(&m.TimeoutStaleReceiptsCallCount, 1)
	if m.TimeoutStaleReceiptsFunc != nil {
		return m.TimeoutStaleReceiptsFunc(ctx, olderThan)
	}
	return 0, nil
}

// ListFailedTasks returns all tasks in a failed state.
//
// Returns ([]*Task, error), or (nil, nil) if ListFailedTasksFunc is nil.
func (m *MockTaskStore) ListFailedTasks(ctx context.Context) ([]*Task, error) {
	atomic.AddInt64(&m.ListFailedTasksCallCount, 1)
	if m.ListFailedTasksFunc != nil {
		return m.ListFailedTasksFunc(ctx)
	}
	return nil, nil
}

// RunAtomic executes fn within a transaction.
//
// Takes fn (func(...) error) which is the function to execute
// within the transaction.
//
// Returns error when fn returns an error, or nil on success.
func (m *MockTaskStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	atomic.AddInt64(&m.RunAtomicCallCount, 1)
	if m.RunAtomicFunc != nil {
		return m.RunAtomicFunc(ctx, fn)
	}
	return fn(ctx, m)
}
