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

// MockTaskStore is a test double for TaskStore where nil function fields return zero
// values and call counts are tracked atomically.
type MockTaskStore struct {
	// CreateTaskFunc is the function called by CreateTask.
	CreateTaskFunc func(ctx context.Context, task *Task) error

	// CreateTasksFunc is the function called by CreateTasks.
	CreateTasksFunc func(ctx context.Context, tasks []*Task) error

	// UpdateTaskFunc is the function called by UpdateTask.
	UpdateTaskFunc func(ctx context.Context, task *Task) error

	// GetTasksByIDFunc is the function called by GetTasksByID.
	GetTasksByIDFunc func(ctx context.Context, ids []string) ([]*Task, error)

	// GetWorkflowStatusFunc is the function called by GetWorkflowStatus.
	GetWorkflowStatusFunc func(ctx context.Context, workflowID string) (bool, error)

	// PromoteScheduledTasksFunc is the function called by PromoteScheduledTasks.
	PromoteScheduledTasksFunc func(ctx context.Context) (int, error)

	// PendingTaskCountFunc is the function called by PendingTaskCount.
	PendingTaskCountFunc func(ctx context.Context) (int64, error)

	// CreateTaskWithDedupFunc is the function called by CreateTaskWithDedup.
	CreateTaskWithDedupFunc func(ctx context.Context, task *Task) error

	// CreateWorkflowReceiptFunc is the function called by CreateWorkflowReceipt.
	CreateWorkflowReceiptFunc func(ctx context.Context, id, workflowID, nodeID string) error

	// ResolveWorkflowReceiptsFunc is the function called by ResolveWorkflowReceipts.
	ResolveWorkflowReceiptsFunc func(ctx context.Context, workflowID string, errorMessage string) (int, error)

	// GetPendingReceiptsByNodeFunc is the function called by GetPendingReceiptsByNode.
	GetPendingReceiptsByNodeFunc func(ctx context.Context, nodeID string) ([]PendingReceipt, error)

	// CleanupOldResolvedReceiptsFunc is the function called by CleanupOldResolvedReceipts.
	CleanupOldResolvedReceiptsFunc func(ctx context.Context, olderThan time.Time) (int, error)

	// TimeoutStaleReceiptsFunc is the function called by TimeoutStaleReceipts.
	TimeoutStaleReceiptsFunc func(ctx context.Context, olderThan time.Time) (int, error)

	// ListFailedTasksFunc is the function called by ListFailedTasks.
	ListFailedTasksFunc func(ctx context.Context) ([]*Task, error)

	// RunAtomicFunc is the function called by RunAtomic.
	RunAtomicFunc func(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error

	// ClaimTaskFunc is the function called by ClaimTask.
	ClaimTaskFunc func(ctx context.Context, request ClaimRequest) (*Task, error)

	// SettleTaskFunc is the function called by SettleTask.
	SettleTaskFunc func(ctx context.Context, settlement Settlement) (SettleOutcome, error)

	// TransitionTaskFunc is the function called by TransitionTask.
	TransitionTaskFunc func(ctx context.Context, transition Transition) (TransitionOutcome, error)

	// RequestRerunFunc is the function called by RequestRerun.
	RequestRerunFunc func(ctx context.Context, key string, rerun RerunRequest) (bool, error)

	// CountUnsettledFunc is the function called by CountUnsettled.
	CountUnsettledFunc func(ctx context.Context, buildTag string) (int64, error)

	// ListFailedForBuildFunc is the function called by ListFailedForBuild.
	ListFailedForBuildFunc func(ctx context.Context, buildTag string) ([]*Task, error)

	// DispatchablePageFunc is the function called by DispatchablePage.
	DispatchablePageFunc func(ctx context.Context, query DispatchQuery) ([]*Task, error)

	// MarkPublishedFunc is the function called by MarkPublished.
	MarkPublishedFunc func(ctx context.Context, ids []string, at time.Time) error

	// ListOrphanedTasksFunc is the function called by ListOrphanedTasks.
	ListOrphanedTasksFunc func(ctx context.Context, query OrphanQuery) ([]*Task, error)

	// RegisterNodeFunc is the function called by RegisterNode.
	RegisterNodeFunc func(ctx context.Context, lease NodeLease) error

	// RenewNodeFunc is the function called by RenewNode.
	RenewNodeFunc func(ctx context.Context, lease NodeLease) error

	// DeregisterNodeFunc is the function called by DeregisterNode.
	DeregisterNodeFunc func(ctx context.Context, instance string) error

	// TouchOwnedFunc is the function called by TouchOwned.
	TouchOwnedFunc func(ctx context.Context, instance string, at time.Time) (int64, error)

	// ListExpiredNodesFunc is the function called by ListExpiredNodes.
	ListExpiredNodesFunc func(ctx context.Context, now time.Time, limit int) ([]string, error)

	// PurgeTerminalFunc is the function called by PurgeTerminal.
	PurgeTerminalFunc func(ctx context.Context, before time.Time, limit int) (int, error)

	// CreateTaskCallCount tracks how many times CreateTask was called.
	CreateTaskCallCount atomic.Int64

	// CreateTasksCallCount tracks how many times CreateTasks was called.
	CreateTasksCallCount atomic.Int64

	// UpdateTaskCallCount tracks how many times UpdateTask was called.
	UpdateTaskCallCount atomic.Int64

	// GetTasksByIDCallCount tracks how many times GetTasksByID was called.
	GetTasksByIDCallCount atomic.Int64

	// GetWorkflowStatusCallCount tracks how many times GetWorkflowStatus was called.
	GetWorkflowStatusCallCount atomic.Int64

	// PromoteScheduledTasksCallCount tracks how many times PromoteScheduledTasks was called.
	PromoteScheduledTasksCallCount atomic.Int64

	// PendingTaskCountCallCount tracks how many times PendingTaskCount was called.
	PendingTaskCountCallCount atomic.Int64

	// CreateTaskWithDedupCallCount tracks how many times CreateTaskWithDedup was called.
	CreateTaskWithDedupCallCount atomic.Int64

	// CreateWorkflowReceiptCallCount tracks how many times CreateWorkflowReceipt was called.
	CreateWorkflowReceiptCallCount atomic.Int64

	// ResolveWorkflowReceiptsCallCount tracks how many times ResolveWorkflowReceipts was
	// called.
	ResolveWorkflowReceiptsCallCount atomic.Int64

	// GetPendingReceiptsByNodeCallCount tracks how many times GetPendingReceiptsByNode was
	// called.
	GetPendingReceiptsByNodeCallCount atomic.Int64

	// CleanupOldResolvedReceiptsCallCount tracks how many times CleanupOldResolvedReceipts
	// was called.
	CleanupOldResolvedReceiptsCallCount atomic.Int64

	// TimeoutStaleReceiptsCallCount tracks how many times TimeoutStaleReceipts was called.
	TimeoutStaleReceiptsCallCount atomic.Int64

	// ListFailedTasksCallCount tracks how many times ListFailedTasks was called.
	ListFailedTasksCallCount atomic.Int64

	// RunAtomicCallCount tracks how many times RunAtomic was called.
	RunAtomicCallCount atomic.Int64

	// ClaimTaskCallCount tracks how many times ClaimTask was called.
	ClaimTaskCallCount atomic.Int64

	// SettleTaskCallCount tracks how many times SettleTask was called.
	SettleTaskCallCount atomic.Int64

	// TransitionTaskCallCount tracks how many times TransitionTask was called.
	TransitionTaskCallCount atomic.Int64

	// RequestRerunCallCount tracks how many times RequestRerun was called.
	RequestRerunCallCount atomic.Int64

	// CountUnsettledCallCount tracks how many times CountUnsettled was called.
	CountUnsettledCallCount atomic.Int64

	// ListFailedForBuildCallCount tracks how many times ListFailedForBuild was called.
	ListFailedForBuildCallCount atomic.Int64

	// DispatchablePageCallCount tracks how many times DispatchablePage was called.
	DispatchablePageCallCount atomic.Int64

	// MarkPublishedCallCount tracks how many times MarkPublished was called.
	MarkPublishedCallCount atomic.Int64

	// ListOrphanedTasksCallCount tracks how many times ListOrphanedTasks was called.
	ListOrphanedTasksCallCount atomic.Int64

	// RegisterNodeCallCount tracks how many times RegisterNode was called.
	RegisterNodeCallCount atomic.Int64

	// RenewNodeCallCount tracks how many times RenewNode was called.
	RenewNodeCallCount atomic.Int64

	// DeregisterNodeCallCount tracks how many times DeregisterNode was called.
	DeregisterNodeCallCount atomic.Int64

	// TouchOwnedCallCount tracks how many times TouchOwned was called.
	TouchOwnedCallCount atomic.Int64

	// ListExpiredNodesCallCount tracks how many times ListExpiredNodes was called.
	ListExpiredNodesCallCount atomic.Int64

	// PurgeTerminalCallCount tracks how many times PurgeTerminal was called.
	PurgeTerminalCallCount atomic.Int64
}

// CreateTask persists a new task.
//
// Takes task (*Task) which is the task to create.
//
// Returns error, or nil if CreateTaskFunc is nil.
func (m *MockTaskStore) CreateTask(ctx context.Context, task *Task) error {
	m.CreateTaskCallCount.Add(1)
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, task)
	}
	return nil
}

// CreateTasks persists multiple tasks in a batch.
//
// Takes tasks ([]*Task) which is the batch of tasks to create.
//
// Returns error, or nil if CreateTasksFunc is nil.
func (m *MockTaskStore) CreateTasks(ctx context.Context, tasks []*Task) error {
	m.CreateTasksCallCount.Add(1)
	if m.CreateTasksFunc != nil {
		return m.CreateTasksFunc(ctx, tasks)
	}
	return nil
}

// UpdateTask updates an existing task.
//
// Takes task (*Task) which is the task to update.
//
// Returns error, or nil if UpdateTaskFunc is nil.
func (m *MockTaskStore) UpdateTask(ctx context.Context, task *Task) error {
	m.UpdateTaskCallCount.Add(1)
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(ctx, task)
	}
	return nil
}

// GetTasksByID reads tasks by ID.
//
// Takes ids ([]string) which lists the task IDs to read.
//
// Returns ([]*Task, error), or ErrStoreUnsupported if GetTasksByIDFunc is nil.
func (m *MockTaskStore) GetTasksByID(ctx context.Context, ids []string) ([]*Task, error) {
	m.GetTasksByIDCallCount.Add(1)
	if m.GetTasksByIDFunc != nil {
		return m.GetTasksByIDFunc(ctx, ids)
	}
	return nil, nil
}

// GetWorkflowStatus checks whether a workflow has completed.
//
// Takes workflowID (string) which identifies the workflow to check.
//
// Returns (bool, error), or ErrStoreUnsupported if GetWorkflowStatusFunc is nil.
func (m *MockTaskStore) GetWorkflowStatus(ctx context.Context, workflowID string) (bool, error) {
	m.GetWorkflowStatusCallCount.Add(1)
	if m.GetWorkflowStatusFunc != nil {
		return m.GetWorkflowStatusFunc(ctx, workflowID)
	}
	return false, nil
}

// PromoteScheduledTasks moves scheduled tasks that are due to the pending queue.
//
// Returns (int, error), or ErrStoreUnsupported if PromoteScheduledTasksFunc is nil.
func (m *MockTaskStore) PromoteScheduledTasks(ctx context.Context) (int, error) {
	m.PromoteScheduledTasksCallCount.Add(1)
	if m.PromoteScheduledTasksFunc != nil {
		return m.PromoteScheduledTasksFunc(ctx)
	}
	return 0, nil
}

// PendingTaskCount returns the number of tasks queued for processing.
//
// Returns (int64, error), or ErrStoreUnsupported if PendingTaskCountFunc is nil.
func (m *MockTaskStore) PendingTaskCount(ctx context.Context) (int64, error) {
	m.PendingTaskCountCallCount.Add(1)
	if m.PendingTaskCountFunc != nil {
		return m.PendingTaskCountFunc(ctx)
	}
	return 0, nil
}

// CreateTaskWithDedup creates a task with deduplication.
//
// Takes task (*Task) which is the task to create with deduplication.
//
// Returns error, or nil if CreateTaskWithDedupFunc is nil.
func (m *MockTaskStore) CreateTaskWithDedup(ctx context.Context, task *Task) error {
	m.CreateTaskWithDedupCallCount.Add(1)
	if m.CreateTaskWithDedupFunc != nil {
		return m.CreateTaskWithDedupFunc(ctx, task)
	}
	return nil
}

// CreateWorkflowReceipt records a workflow receipt for tracking.
//
// Takes id (string) which identifies the receipt.
// Takes workflowID (string) which identifies the workflow.
// Takes nodeID (string) which identifies the originating node.
//
// Returns error, or nil if CreateWorkflowReceiptFunc is nil.
func (m *MockTaskStore) CreateWorkflowReceipt(ctx context.Context, id, workflowID, nodeID string) error {
	m.CreateWorkflowReceiptCallCount.Add(1)
	if m.CreateWorkflowReceiptFunc != nil {
		return m.CreateWorkflowReceiptFunc(ctx, id, workflowID, nodeID)
	}
	return nil
}

// ResolveWorkflowReceipts resolves all receipts for a workflow.
//
// Takes workflowID (string) which identifies the workflow to resolve.
// Takes errorMessage (string) which is the error message to record, or empty on success.
//
// Returns (int, error), or ErrStoreUnsupported if ResolveWorkflowReceiptsFunc is nil.
func (m *MockTaskStore) ResolveWorkflowReceipts(ctx context.Context, workflowID string, errorMessage string) (int, error) {
	m.ResolveWorkflowReceiptsCallCount.Add(1)
	if m.ResolveWorkflowReceiptsFunc != nil {
		return m.ResolveWorkflowReceiptsFunc(ctx, workflowID, errorMessage)
	}
	return 0, nil
}

// GetPendingReceiptsByNode returns unresolved receipts for a node.
//
// Takes nodeID (string) which identifies the node to query.
//
// Returns ([]PendingReceipt, error), or (nil, nil) if GetPendingReceiptsByNodeFunc is
// nil.
func (m *MockTaskStore) GetPendingReceiptsByNode(ctx context.Context, nodeID string) ([]PendingReceipt, error) {
	m.GetPendingReceiptsByNodeCallCount.Add(1)
	if m.GetPendingReceiptsByNodeFunc != nil {
		return m.GetPendingReceiptsByNodeFunc(ctx, nodeID)
	}
	return nil, nil
}

// CleanupOldResolvedReceipts removes resolved receipts older than the given time.
//
// Takes olderThan (time.Time) which is the cutoff time for cleanup.
//
// Returns (int, error), or ErrStoreUnsupported if CleanupOldResolvedReceiptsFunc is nil.
func (m *MockTaskStore) CleanupOldResolvedReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	m.CleanupOldResolvedReceiptsCallCount.Add(1)
	if m.CleanupOldResolvedReceiptsFunc != nil {
		return m.CleanupOldResolvedReceiptsFunc(ctx, olderThan)
	}
	return 0, nil
}

// TimeoutStaleReceipts marks stale receipts as timed out.
//
// Takes olderThan (time.Time) which is the cutoff time for timeout.
//
// Returns (int, error), or ErrStoreUnsupported if TimeoutStaleReceiptsFunc is nil.
func (m *MockTaskStore) TimeoutStaleReceipts(ctx context.Context, olderThan time.Time) (int, error) {
	m.TimeoutStaleReceiptsCallCount.Add(1)
	if m.TimeoutStaleReceiptsFunc != nil {
		return m.TimeoutStaleReceiptsFunc(ctx, olderThan)
	}
	return 0, nil
}

// ListFailedTasks returns all tasks in a failed state.
//
// Returns ([]*Task, error), or ErrStoreUnsupported if ListFailedTasksFunc is nil.
func (m *MockTaskStore) ListFailedTasks(ctx context.Context) ([]*Task, error) {
	m.ListFailedTasksCallCount.Add(1)
	if m.ListFailedTasksFunc != nil {
		return m.ListFailedTasksFunc(ctx)
	}
	return nil, nil
}

// RunAtomic executes fn within a transaction.
//
// Takes fn (func(...) error) which is the function to execute within the transaction.
//
// Returns error when fn returns an error, or nil on success.
func (m *MockTaskStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	m.RunAtomicCallCount.Add(1)
	if m.RunAtomicFunc != nil {
		return m.RunAtomicFunc(ctx, fn)
	}
	return fn(ctx, m)
}

// ClaimTask claims a delivered attempt.
//
// Takes request (ClaimRequest) which describes the task attempt to claim.
//
// Returns *Task which is the hook result, or its zero value when ClaimTaskFunc is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when ClaimTaskFunc is
// nil.
func (m *MockTaskStore) ClaimTask(ctx context.Context, request ClaimRequest) (*Task, error) {
	m.ClaimTaskCallCount.Add(1)
	if m.ClaimTaskFunc != nil {
		return m.ClaimTaskFunc(ctx, request)
	}
	return nil, ErrStoreUnsupported
}

// SettleTask settles a claimed task.
//
// Takes settlement (Settlement) which holds the task state and successor to persist.
//
// Returns SettleOutcome which is the hook result, or its zero value when SettleTaskFunc
// is nil.
// Returns error which is the hook error, or nil when SettleTaskFunc is nil.
func (m *MockTaskStore) SettleTask(ctx context.Context, settlement Settlement) (SettleOutcome, error) {
	m.SettleTaskCallCount.Add(1)
	if m.SettleTaskFunc != nil {
		return m.SettleTaskFunc(ctx, settlement)
	}
	return SettleOutcome{}, ErrStoreUnsupported
}

// TransitionTask moves a task between statuses.
//
// Takes transition (Transition) which specifies the expected and replacement task states.
//
// Returns TransitionOutcome which is the hook result, or its zero value when
// TransitionTaskFunc is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when TransitionTaskFunc
// is nil.
func (m *MockTaskStore) TransitionTask(ctx context.Context, transition Transition) (TransitionOutcome, error) {
	m.TransitionTaskCallCount.Add(1)
	if m.TransitionTaskFunc != nil {
		return m.TransitionTaskFunc(ctx, transition)
	}
	return TransitionOutcome{}, ErrStoreUnsupported
}

// RequestRerun records a rerun request.
//
// Takes key (string) which is the deduplication key.
// Takes rerun (RerunRequest) which describes the requested rerun.
//
// Returns bool which is the hook result, or its zero value when RequestRerunFunc is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when RequestRerunFunc is
// nil.
func (m *MockTaskStore) RequestRerun(ctx context.Context, key string, rerun RerunRequest) (bool, error) {
	m.RequestRerunCallCount.Add(1)
	if m.RequestRerunFunc != nil {
		return m.RequestRerunFunc(ctx, key, rerun)
	}
	return false, ErrStoreUnsupported
}

// CountUnsettled counts a build's unsettled tasks.
//
// Takes buildTag (string) which identifies the build.
//
// Returns int64 which is the hook result, or its zero value when CountUnsettledFunc is
// nil.
// Returns error which is the hook error, or ErrStoreUnsupported when CountUnsettledFunc
// is nil.
func (m *MockTaskStore) CountUnsettled(ctx context.Context, buildTag string) (int64, error) {
	m.CountUnsettledCallCount.Add(1)
	if m.CountUnsettledFunc != nil {
		return m.CountUnsettledFunc(ctx, buildTag)
	}
	return 0, ErrStoreUnsupported
}

// ListFailedForBuild lists a build's failed tasks.
//
// Takes buildTag (string) which identifies the build.
//
// Returns []*Task which is the hook result, or its zero value when ListFailedForBuildFunc
// is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when
// ListFailedForBuildFunc is nil.
func (m *MockTaskStore) ListFailedForBuild(ctx context.Context, buildTag string) ([]*Task, error) {
	m.ListFailedForBuildCallCount.Add(1)
	if m.ListFailedForBuildFunc != nil {
		return m.ListFailedForBuildFunc(ctx, buildTag)
	}
	return nil, ErrStoreUnsupported
}

// DispatchablePage reads a page of due tasks.
//
// Takes query (DispatchQuery) which selects due tasks and the page boundary.
//
// Returns []*Task which is the hook result, or its zero value when DispatchablePageFunc
// is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when DispatchablePageFunc
// is nil.
func (m *MockTaskStore) DispatchablePage(ctx context.Context, query DispatchQuery) ([]*Task, error) {
	m.DispatchablePageCallCount.Add(1)
	if m.DispatchablePageFunc != nil {
		return m.DispatchablePageFunc(ctx, query)
	}
	return nil, ErrStoreUnsupported
}

// MarkPublished records publication times.
//
// Takes ids ([]string) which identifies the tasks.
// Takes at (time.Time) which is the timestamp to record.
//
// Returns error which is the hook error, or nil when MarkPublishedFunc is nil.
func (m *MockTaskStore) MarkPublished(ctx context.Context, ids []string, at time.Time) error {
	m.MarkPublishedCallCount.Add(1)
	if m.MarkPublishedFunc != nil {
		return m.MarkPublishedFunc(ctx, ids, at)
	}
	return ErrStoreUnsupported
}

// ListOrphanedTasks lists orphaned processing tasks.
//
// Takes query (OrphanQuery) which selects orphaned tasks by age and result limit.
//
// Returns []*Task which is the hook result, or its zero value when ListOrphanedTasksFunc
// is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when
// ListOrphanedTasksFunc is nil.
func (m *MockTaskStore) ListOrphanedTasks(ctx context.Context, query OrphanQuery) ([]*Task, error) {
	m.ListOrphanedTasksCallCount.Add(1)
	if m.ListOrphanedTasksFunc != nil {
		return m.ListOrphanedTasksFunc(ctx, query)
	}
	return nil, ErrStoreUnsupported
}

// RegisterNode records a node lease.
//
// Takes lease (NodeLease) which describes the node instance and its expiry.
//
// Returns error which is the hook error, or nil when RegisterNodeFunc is nil.
func (m *MockTaskStore) RegisterNode(ctx context.Context, lease NodeLease) error {
	m.RegisterNodeCallCount.Add(1)
	if m.RegisterNodeFunc != nil {
		return m.RegisterNodeFunc(ctx, lease)
	}
	return ErrStoreUnsupported
}

// RenewNode extends a node lease.
//
// Takes lease (NodeLease) which describes the node instance and its expiry.
//
// Returns error which is the hook error, or nil when RenewNodeFunc is nil.
func (m *MockTaskStore) RenewNode(ctx context.Context, lease NodeLease) error {
	m.RenewNodeCallCount.Add(1)
	if m.RenewNodeFunc != nil {
		return m.RenewNodeFunc(ctx, lease)
	}
	return ErrStoreUnsupported
}

// DeregisterNode removes a node lease.
//
// Takes instance (string) which identifies the node instance.
//
// Returns error which is the hook error, or nil when DeregisterNodeFunc is nil.
func (m *MockTaskStore) DeregisterNode(ctx context.Context, instance string) error {
	m.DeregisterNodeCallCount.Add(1)
	if m.DeregisterNodeFunc != nil {
		return m.DeregisterNodeFunc(ctx, instance)
	}
	return ErrStoreUnsupported
}

// TouchOwned stamps an instance's processing tasks.
//
// Takes instance (string) which identifies the node instance.
// Takes at (time.Time) which is the timestamp to record.
//
// Returns int64 which is the hook result, or its zero value when TouchOwnedFunc is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when TouchOwnedFunc is
// nil.
func (m *MockTaskStore) TouchOwned(ctx context.Context, instance string, at time.Time) (int64, error) {
	m.TouchOwnedCallCount.Add(1)
	if m.TouchOwnedFunc != nil {
		return m.TouchOwnedFunc(ctx, instance, at)
	}
	return 0, ErrStoreUnsupported
}

// ListExpiredNodes lists lapsed node leases.
//
// Takes now (time.Time) which is the time used to check lease expiry.
// Takes limit (int) which caps the number of results or deletions.
//
// Returns []string which is the hook result, or its zero value when ListExpiredNodesFunc
// is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when ListExpiredNodesFunc
// is nil.
func (m *MockTaskStore) ListExpiredNodes(ctx context.Context, now time.Time, limit int) ([]string, error) {
	m.ListExpiredNodesCallCount.Add(1)
	if m.ListExpiredNodesFunc != nil {
		return m.ListExpiredNodesFunc(ctx, now, limit)
	}
	return nil, ErrStoreUnsupported
}

// PurgeTerminal deletes old terminal tasks.
//
// Takes before (time.Time) which is the update-time cutoff for deletion.
// Takes limit (int) which caps the number of results or deletions.
//
// Returns int which is the hook result, or its zero value when PurgeTerminalFunc is nil.
// Returns error which is the hook error, or ErrStoreUnsupported when PurgeTerminalFunc is
// nil.
func (m *MockTaskStore) PurgeTerminal(ctx context.Context, before time.Time, limit int) (int, error) {
	m.PurgeTerminalCallCount.Add(1)
	if m.PurgeTerminalFunc != nil {
		return m.PurgeTerminalFunc(ctx, before, limit)
	}
	return 0, ErrStoreUnsupported
}
