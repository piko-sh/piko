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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockTaskStoreTracker struct {
	workflowErr      error
	workflowComplete atomic.Bool
	checkCount       atomic.Int32
}

func (m *mockTaskStoreTracker) CreateTask(_ context.Context, _ *Task) error    { return nil }
func (m *mockTaskStoreTracker) CreateTasks(_ context.Context, _ []*Task) error { return nil }
func (m *mockTaskStoreTracker) UpdateTask(_ context.Context, _ *Task) error    { return nil }
func (m *mockTaskStoreTracker) FetchAndMarkDueTasks(_ context.Context, _ TaskPriority, _ int) ([]*Task, error) {
	return nil, nil
}
func (m *mockTaskStoreTracker) GetWorkflowStatus(_ context.Context, _ string) (bool, error) {
	m.checkCount.Add(1)
	return m.workflowComplete.Load(), m.workflowErr
}
func (m *mockTaskStoreTracker) PendingTaskCount(_ context.Context) (int64, error)    { return 0, nil }
func (m *mockTaskStoreTracker) PromoteScheduledTasks(_ context.Context) (int, error) { return 0, nil }
func (m *mockTaskStoreTracker) CreateTaskWithDedup(_ context.Context, _ *Task) error { return nil }
func (m *mockTaskStoreTracker) RecoverStaleTasks(_ context.Context, _ time.Duration, _ int, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) GetStaleProcessingTaskCount(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) UpdateTaskHeartbeat(_ context.Context, _ string) error { return nil }
func (m *mockTaskStoreTracker) ClaimStaleTasksForRecovery(_ context.Context, _ string, _ time.Duration, _ time.Duration, _ int) ([]RecoveryClaimedTask, error) {
	return nil, nil
}
func (m *mockTaskStoreTracker) RecoverClaimedTasks(_ context.Context, _ string, _ int, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) ReleaseRecoveryLeases(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) CreateWorkflowReceipt(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockTaskStoreTracker) ResolveWorkflowReceipts(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) GetPendingReceiptsByNode(_ context.Context, _ string) ([]PendingReceipt, error) {
	return nil, nil
}
func (m *mockTaskStoreTracker) CleanupOldResolvedReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) TimeoutStaleReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreTracker) ListFailedTasks(_ context.Context) ([]*Task, error) {
	return nil, nil
}

func (m *mockTaskStoreTracker) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	return fn(ctx, m)
}

func TestRegisterReceipt(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.registerReceipt(context.Background(), "receipt-1", receipt)

	if len(service.receipts["workflow-1"]) != 1 {
		t.Error("receipt not registered")
	}
}

func TestRegisterReceipt_MultipleForSameWorkflow(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt1 := newWorkflowReceipt("workflow-1")
	receipt2 := newWorkflowReceipt("workflow-1")
	service.registerReceipt(context.Background(), "receipt-1", receipt1)
	service.registerReceipt(context.Background(), "receipt-2", receipt2)

	if len(service.receipts["workflow-1"]) != 2 {
		t.Errorf("expected 2 receipts, got %d", len(service.receipts["workflow-1"]))
	}
}

func TestRemoveReceipt(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt1 := newWorkflowReceipt("workflow-1")
	receipt2 := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt1, receipt2}

	service.removeReceipt(receipt1)

	if len(service.receipts["workflow-1"]) != 1 {
		t.Errorf("expected 1 receipt after removal, got %d", len(service.receipts["workflow-1"]))
	}
	if service.receipts["workflow-1"][0] != receipt2 {
		t.Error("wrong receipt removed")
	}
}

func TestRemoveReceipt_LastReceipt(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt}

	service.removeReceipt(receipt)

	if _, exists := service.receipts["workflow-1"]; exists {
		t.Error("workflow entry should be deleted when last receipt is removed")
	}
}

func TestRemoveReceipt_NotFound(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.removeReceipt(receipt)
}

func TestResolveReceipts(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt1 := newWorkflowReceipt("workflow-1")
	receipt2 := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt1, receipt2}

	service.resolveReceipts(context.Background(), "workflow-1", nil)

	select {
	case <-receipt1.Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt1 not resolved")
	}

	select {
	case <-receipt2.Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt2 not resolved")
	}

	if _, exists := service.receipts["workflow-1"]; exists {
		t.Error("workflow entry should be deleted after resolving")
	}
}

func TestResolveReceipts_WithError(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt}

	expectedErr := errors.New("workflow failed")
	service.resolveReceipts(context.Background(), "workflow-1", expectedErr)

	select {
	case err := <-receipt.Done():
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt not resolved")
	}
}

func TestResolveReceipts_NoReceipts(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	service.resolveReceipts(context.Background(), "workflow-1", nil)
}

func TestHandleCompletionEvent_Success(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreTracker{}
	store.workflowComplete.Store(true)

	service := &orchestratorService{
		taskStore:     store,
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt}

	event := Event{
		Payload: map[string]any{
			"workflowId": "workflow-1",
			"status":     "success",
		},
	}

	service.handleCompletionEvent(context.Background(), event)

	time.Sleep(50 * time.Millisecond)

	select {
	case err := <-receipt.Done():
		if err != nil {
			t.Errorf("expected nil error for success, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt not resolved")
	}
}

func TestHandleCompletionEvent_Failure(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreTracker{}
	store.workflowComplete.Store(true)

	service := &orchestratorService{
		taskStore:     store,
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt}

	event := Event{
		Payload: map[string]any{
			"workflowId": "workflow-1",
			"status":     "failure",
			"error":      "task execution failed",
		},
	}

	service.handleCompletionEvent(context.Background(), event)

	time.Sleep(50 * time.Millisecond)

	select {
	case err := <-receipt.Done():
		if err == nil {
			t.Error("expected error for failure, got nil")
		}
		if err.Error() != "task execution failed" {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt not resolved")
	}
}

func TestHandleCompletionEvent_MalformedEvent(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	event := Event{
		Payload: map[string]any{
			"status": "success",
		},
	}

	service.handleCompletionEvent(context.Background(), event)
}

func TestHandleCompletionEvent_WorkflowNotComplete(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreTracker{}
	store.workflowComplete.Store(false)

	service := &orchestratorService{
		taskStore:     store,
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt}

	event := Event{
		Payload: map[string]any{
			"workflowId": "workflow-1",
			"status":     "success",
		},
	}

	service.handleCompletionEvent(context.Background(), event)

	time.Sleep(50 * time.Millisecond)

	select {
	case <-receipt.Done():
		t.Error("receipt should not be resolved when workflow is incomplete")
	default:
	}
}

func TestCheckAndResolveWorkflow_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreTracker{}
	store.workflowComplete.Store(true)

	service := &orchestratorService{
		taskStore:     store,
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt := newWorkflowReceipt("workflow-1")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt}

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			service.checkAndResolveWorkflow(context.Background(), "workflow-1", nil)
		})
	}
	wg.Wait()

	time.Sleep(50 * time.Millisecond)

	select {
	case <-receipt.Done():
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt not resolved after concurrent checks")
	}

	if store.checkCount.Load() == 0 {
		t.Error("expected at least one workflow check")
	}
}

func TestGetPayloadString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		payload  map[string]any
		key      string
		expected string
	}{
		{
			name:     "existing string value",
			payload:  map[string]any{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "missing key",
			payload:  map[string]any{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			payload:  map[string]any{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "nil payload",
			payload:  nil,
			key:      "key",
			expected: "",
		},
		{
			name:     "empty payload",
			payload:  map[string]any{},
			key:      "key",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := getPayloadString(tc.payload, tc.key)

			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}
