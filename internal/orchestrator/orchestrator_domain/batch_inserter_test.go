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

	"piko.sh/piko/wdk/clock"
)

type mockTaskStoreBatch struct {
	createTasksErr    error
	createdTasks      []*Task
	mu                sync.Mutex
	createTasksCalled atomic.Int32
}

func (m *mockTaskStoreBatch) CreateTask(_ context.Context, _ *Task) error { return nil }
func (m *mockTaskStoreBatch) CreateTasks(_ context.Context, tasks []*Task) error {
	m.createTasksCalled.Add(1)
	m.mu.Lock()
	m.createdTasks = append(m.createdTasks, tasks...)
	m.mu.Unlock()
	return m.createTasksErr
}
func (m *mockTaskStoreBatch) UpdateTask(_ context.Context, _ *Task) error { return nil }
func (m *mockTaskStoreBatch) FetchAndMarkDueTasks(_ context.Context, _ TaskPriority, _ int) ([]*Task, error) {
	return nil, nil
}
func (m *mockTaskStoreBatch) GetWorkflowStatus(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockTaskStoreBatch) PendingTaskCount(_ context.Context) (int64, error)    { return 0, nil }
func (m *mockTaskStoreBatch) PromoteScheduledTasks(_ context.Context) (int, error) { return 0, nil }
func (m *mockTaskStoreBatch) CreateTaskWithDedup(_ context.Context, task *Task) error {
	return m.CreateTask(context.Background(), task)
}
func (m *mockTaskStoreBatch) RecoverStaleTasks(_ context.Context, _ time.Duration, _ int, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) GetStaleProcessingTaskCount(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) UpdateTaskHeartbeat(_ context.Context, _ string) error { return nil }
func (m *mockTaskStoreBatch) ClaimStaleTasksForRecovery(_ context.Context, _ string, _ time.Duration, _ time.Duration, _ int) ([]RecoveryClaimedTask, error) {
	return nil, nil
}
func (m *mockTaskStoreBatch) RecoverClaimedTasks(_ context.Context, _ string, _ int, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) ReleaseRecoveryLeases(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) CreateWorkflowReceipt(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockTaskStoreBatch) ResolveWorkflowReceipts(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) GetPendingReceiptsByNode(_ context.Context, _ string) ([]PendingReceipt, error) {
	return nil, nil
}
func (m *mockTaskStoreBatch) CleanupOldResolvedReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) TimeoutStaleReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreBatch) ListFailedTasks(_ context.Context) ([]*Task, error) {
	return nil, nil
}

func (m *mockTaskStoreBatch) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	return fn(ctx, m)
}

type mockTaskDispatcherBatch struct {
	dispatchedTasks       []*Task
	mu                    sync.Mutex
	dispatchCalled        atomic.Int32
	dispatchDelayedCalled atomic.Int32
}

func (m *mockTaskDispatcherBatch) Dispatch(_ context.Context, task *Task) error {
	m.dispatchCalled.Add(1)
	m.mu.Lock()
	m.dispatchedTasks = append(m.dispatchedTasks, task)
	m.mu.Unlock()
	return nil
}
func (m *mockTaskDispatcherBatch) DispatchDelayed(_ context.Context, task *Task, _ time.Time) error {
	m.dispatchDelayedCalled.Add(1)
	m.mu.Lock()
	m.dispatchedTasks = append(m.dispatchedTasks, task)
	m.mu.Unlock()
	return nil
}
func (m *mockTaskDispatcherBatch) RegisterExecutor(_ context.Context, _ string, _ TaskExecutor) {}
func (m *mockTaskDispatcherBatch) Start(_ context.Context) error                                { return nil }
func (m *mockTaskDispatcherBatch) Stop()                                                        {}
func (m *mockTaskDispatcherBatch) Stats() DispatcherStats                                       { return DispatcherStats{} }
func (m *mockTaskDispatcherBatch) IsIdle() bool                                                 { return true }
func (m *mockTaskDispatcherBatch) FailedTasks(_ context.Context) ([]FailedTaskSummary, error) {
	return nil, nil
}
func (m *mockTaskDispatcherBatch) SetBuildTag(_ string) {}
func (m *mockTaskDispatcherBatch) BuildTag() string     { return "" }

func TestStopChannelTimerSafe(t *testing.T) {
	t.Parallel()
	realClock := clock.RealClock()

	t.Run("stops active timer", func(t *testing.T) {
		t.Parallel()

		timer := realClock.NewTimer(1 * time.Hour)
		stopChannelTimerSafe(timer)

	})

	t.Run("handles already stopped timer", func(t *testing.T) {
		t.Parallel()

		timer := realClock.NewTimer(1 * time.Hour)
		timer.Stop()
		stopChannelTimerSafe(timer)
	})

	t.Run("handles fired timer", func(t *testing.T) {
		t.Parallel()

		timer := realClock.NewTimer(1 * time.Nanosecond)
		time.Sleep(10 * time.Millisecond)
		stopChannelTimerSafe(timer)
	})
}

func TestFlushTaskBatch_EmptyBatch(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreBatch{}
	service := &orchestratorService{
		taskStore: store,
		receipts:  make(map[string][]*WorkflowReceipt),
	}

	batch := []*Task{}
	result := service.flushTaskBatch(batch)

	if len(result) != 0 {
		t.Error("empty batch should return empty slice")
	}
	if store.createTasksCalled.Load() != 0 {
		t.Error("CreateTasks should not be called for empty batch")
	}
}

func TestFlushTaskBatch_Success(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreBatch{}
	dispatcher := &mockTaskDispatcherBatch{}
	service := &orchestratorService{
		taskStore:      store,
		taskDispatcher: dispatcher,
		receipts:       make(map[string][]*WorkflowReceipt),
		receiptsMutex:  sync.Mutex{},
		runCtx:         context.Background(),
	}

	tasks := []*Task{
		{ID: "task-1", WorkflowID: "workflow-1", Status: StatusPending},
		{ID: "task-2", WorkflowID: "workflow-2", Status: StatusPending},
	}

	result := service.flushTaskBatch(tasks)

	if len(result) != 0 {
		t.Error("successful flush should return empty (reset) slice")
	}
	if store.createTasksCalled.Load() != 1 {
		t.Errorf("CreateTasks should be called once, got %d", store.createTasksCalled.Load())
	}
	if len(store.createdTasks) != 2 {
		t.Errorf("expected 2 created tasks, got %d", len(store.createdTasks))
	}
	if dispatcher.dispatchCalled.Load() != 2 {
		t.Errorf("expected 2 dispatch calls, got %d", dispatcher.dispatchCalled.Load())
	}
}

func TestFlushTaskBatch_StoreError(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreBatch{createTasksErr: errors.New("db error")}
	service := &orchestratorService{
		taskStore:     store,
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
		runCtx:        context.Background(),
	}

	receipt1 := newWorkflowReceipt("workflow-1")
	receipt2 := newWorkflowReceipt("workflow-2")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt1}
	service.receipts["workflow-2"] = []*WorkflowReceipt{receipt2}

	tasks := []*Task{
		{ID: "task-1", WorkflowID: "workflow-1"},
		{ID: "task-2", WorkflowID: "workflow-2"},
	}

	result := service.flushTaskBatch(tasks)

	if len(result) != 0 {
		t.Error("failed flush should return empty (reset) slice")
	}

	select {
	case err := <-receipt1.Done():
		if err == nil {
			t.Error("receipt1 should have error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt1 not resolved")
	}

	select {
	case err := <-receipt2.Done():
		if err == nil {
			t.Error("receipt2 should have error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("receipt2 not resolved")
	}
}

func TestDispatchSingleTask_PendingTask(t *testing.T) {
	t.Parallel()

	dispatcher := &mockTaskDispatcherBatch{}
	service := &orchestratorService{
		taskDispatcher: dispatcher,
	}

	task := &Task{ID: "task-1", Status: StatusPending}
	service.dispatchSingleTask(context.Background(), task)

	if dispatcher.dispatchCalled.Load() != 1 {
		t.Errorf("expected Dispatch to be called once, got %d", dispatcher.dispatchCalled.Load())
	}
	if dispatcher.dispatchDelayedCalled.Load() != 0 {
		t.Error("DispatchDelayed should not be called for pending task")
	}
}

func TestDispatchSingleTask_ScheduledTask(t *testing.T) {
	t.Parallel()

	dispatcher := &mockTaskDispatcherBatch{}
	service := &orchestratorService{
		taskDispatcher: dispatcher,
	}

	executeAt := time.Now().Add(1 * time.Hour)
	task := &Task{
		ID:        "task-1",
		Status:    StatusScheduled,
		ExecuteAt: executeAt,
	}
	service.dispatchSingleTask(context.Background(), task)

	if dispatcher.dispatchDelayedCalled.Load() != 1 {
		t.Errorf("expected DispatchDelayed to be called once, got %d", dispatcher.dispatchDelayedCalled.Load())
	}
	if dispatcher.dispatchCalled.Load() != 0 {
		t.Error("Dispatch should not be called for scheduled task")
	}
}

func TestDispatchPersistedTasks_NilDispatcher(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		taskDispatcher: nil,
	}

	tasks := []*Task{{ID: "task-1"}}
	service.dispatchPersistedTasks(context.Background(), tasks)
}

func TestDispatchPersistedTasks_MultipleTasks(t *testing.T) {
	t.Parallel()

	dispatcher := &mockTaskDispatcherBatch{}
	service := &orchestratorService{
		taskDispatcher: dispatcher,
	}

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending},
		{ID: "task-2", Status: StatusScheduled, ExecuteAt: time.Now().Add(1 * time.Hour)},
		{ID: "task-3", Status: StatusPending},
	}

	service.dispatchPersistedTasks(context.Background(), tasks)

	if dispatcher.dispatchCalled.Load() != 2 {
		t.Errorf("expected 2 Dispatch calls, got %d", dispatcher.dispatchCalled.Load())
	}
	if dispatcher.dispatchDelayedCalled.Load() != 1 {
		t.Errorf("expected 1 DispatchDelayed call, got %d", dispatcher.dispatchDelayedCalled.Load())
	}
}

func TestFailBatchReceipts(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	receipt1 := newWorkflowReceipt("workflow-1")
	receipt2 := newWorkflowReceipt("workflow-1")
	receipt3 := newWorkflowReceipt("workflow-2")
	service.receipts["workflow-1"] = []*WorkflowReceipt{receipt1, receipt2}
	service.receipts["workflow-2"] = []*WorkflowReceipt{receipt3}

	batchErr := errors.New("batch insert failed")
	tasks := []*Task{
		{ID: "task-1", WorkflowID: "workflow-1"},
		{ID: "task-2", WorkflowID: "workflow-2"},
	}

	service.failBatchReceipts(tasks, batchErr)

	for i, receipt := range []*WorkflowReceipt{receipt1, receipt2, receipt3} {
		select {
		case err := <-receipt.Done():
			if !errors.Is(err, batchErr) {
				t.Errorf("receipt %d: expected batch error, got %v", i, err)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("receipt %d: not resolved", i)
		}
	}

	if len(service.receipts) != 0 {
		t.Errorf("expected empty receipts map, got %d entries", len(service.receipts))
	}
}

func TestFailBatchReceipts_NoMatchingReceipts(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		receipts:      make(map[string][]*WorkflowReceipt),
		receiptsMutex: sync.Mutex{},
	}

	batchErr := errors.New("batch insert failed")
	tasks := []*Task{
		{ID: "task-1", WorkflowID: "workflow-1"},
	}

	service.failBatchReceipts(tasks, batchErr)
}

func TestDrainTaskChannel_EmptyChannel(t *testing.T) {
	t.Parallel()

	realClock := clock.RealClock()
	service := &orchestratorService{
		taskInsertChan: make(chan *Task, 10),
		batchSize:      100,
	}

	timer := realClock.NewTimer(1 * time.Hour)
	defer timer.Stop()

	batch := []*Task{{ID: "task-1"}}
	result := service.drainTaskChannel(batch, timer)

	if len(result) != 1 {
		t.Errorf("expected batch with 1 task, got %d", len(result))
	}
}

func TestDrainTaskChannel_ChannelWithTasks(t *testing.T) {
	t.Parallel()

	realClock := clock.RealClock()
	service := &orchestratorService{
		taskInsertChan: make(chan *Task, 10),
		batchSize:      100,
	}

	service.taskInsertChan <- &Task{ID: "task-2"}
	service.taskInsertChan <- &Task{ID: "task-3"}

	timer := realClock.NewTimer(1 * time.Hour)
	defer timer.Stop()

	batch := []*Task{{ID: "task-1"}}
	result := service.drainTaskChannel(batch, timer)

	if len(result) != 3 {
		t.Errorf("expected batch with 3 tasks, got %d", len(result))
	}
}

func TestDrainTaskChannel_StopsAtBatchSize(t *testing.T) {
	t.Parallel()

	realClock := clock.RealClock()
	service := &orchestratorService{
		taskInsertChan: make(chan *Task, 10),
		batchSize:      3,
	}

	for range 5 {
		service.taskInsertChan <- &Task{ID: "task"}
	}

	timer := realClock.NewTimer(1 * time.Hour)
	defer timer.Stop()

	batch := []*Task{{ID: "task-0"}}
	result := service.drainTaskChannel(batch, timer)

	if len(result) != 3 {
		t.Errorf("expected batch size of 3, got %d", len(result))
	}
}

func TestDrainTaskChannel_ClosedChannel(t *testing.T) {
	t.Parallel()

	realClock := clock.RealClock()
	service := &orchestratorService{
		taskInsertChan: make(chan *Task, 10),
		batchSize:      100,
	}

	service.taskInsertChan <- &Task{ID: "task-1"}
	close(service.taskInsertChan)

	timer := realClock.NewTimer(1 * time.Hour)
	defer timer.Stop()

	batch := []*Task{}
	result := service.drainTaskChannel(batch, timer)

	if len(result) != 1 {
		t.Errorf("expected 1 task, got %d", len(result))
	}
}
