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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustCastToOrchestratorService(t *testing.T, service OrchestratorService) *orchestratorService {
	t.Helper()
	impl, ok := service.(*orchestratorService)
	if !ok {
		t.Fatalf("expected *orchestratorService, got %T", service)
	}
	return impl
}

type FakeTaskStore struct {
	tasks map[string]*Task
	mu    sync.Mutex
}

func NewFakeTaskStore() *FakeTaskStore {
	return &FakeTaskStore{
		tasks: make(map[string]*Task),
	}
}

func (f *FakeTaskStore) CreateTask(_ context.Context, task *Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks[task.ID] = new(*task)
	return nil
}

func (f *FakeTaskStore) CreateTasks(ctx context.Context, tasks []*Task) error {
	for _, task := range tasks {
		if err := f.CreateTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (f *FakeTaskStore) UpdateTask(_ context.Context, task *Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks[task.ID] = new(*task)
	return nil
}

func (f *FakeTaskStore) GetWorkflowStatus(_ context.Context, workflowID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, task := range f.tasks {
		if task.WorkflowID == workflowID {
			if task.Status != StatusComplete && task.Status != StatusFailed {
				return false, nil
			}
		}
	}
	return true, nil
}

func (f *FakeTaskStore) FetchAndMarkDueTasks(_ context.Context, priority TaskPriority, limit int) ([]*Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var dueTasks []*Task
	for _, task := range f.tasks {
		if (task.Status == StatusPending || task.Status == StatusRetrying) &&
			task.Config.Priority == priority &&
			len(dueTasks) < limit {
			task.Status = StatusProcessing
			dueTasks = append(dueTasks, task)
		}
	}
	tasksCopy := make([]*Task, len(dueTasks))
	for i, task := range dueTasks {
		tasksCopy[i] = new(*task)
	}
	return tasksCopy, nil
}

func (f *FakeTaskStore) PromoteScheduledTasks(context.Context) (int, error) { return 0, nil }
func (f *FakeTaskStore) PendingTaskCount(context.Context) (int64, error)    { return 0, nil }

func (f *FakeTaskStore) CreateTaskWithDedup(_ context.Context, task *Task) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if task.DeduplicationKey != "" {
		for _, existing := range f.tasks {
			if existing.DeduplicationKey == task.DeduplicationKey &&
				(existing.Status == StatusScheduled ||
					existing.Status == StatusPending ||
					existing.Status == StatusProcessing ||
					existing.Status == StatusRetrying) {
				return ErrDuplicateTask
			}
		}
	}

	f.tasks[task.ID] = new(*task)
	return nil
}

func (f *FakeTaskStore) RecoverStaleTasks(_ context.Context, staleThreshold time.Duration, maxRetries int, recoveryError string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	staleTime := now.Add(-staleThreshold)
	recovered := 0

	for _, task := range f.tasks {
		if task.Status == StatusProcessing && task.UpdatedAt.Before(staleTime) {
			if task.Attempt >= maxRetries {
				task.Status = StatusFailed
			} else {
				task.Status = StatusRetrying
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

func (f *FakeTaskStore) GetStaleProcessingTaskCount(_ context.Context, staleThreshold time.Duration) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	staleTime := time.Now().Add(-staleThreshold)
	var count int64

	for _, task := range f.tasks {
		if task.Status == StatusProcessing && task.UpdatedAt.Before(staleTime) {
			count++
		}
	}

	return count, nil
}

func (f *FakeTaskStore) UpdateTaskHeartbeat(_ context.Context, _ string) error { return nil }
func (f *FakeTaskStore) ClaimStaleTasksForRecovery(_ context.Context, _ string, _ time.Duration, _ time.Duration, _ int) ([]RecoveryClaimedTask, error) {
	return nil, nil
}
func (f *FakeTaskStore) RecoverClaimedTasks(_ context.Context, _ string, _ int, _ string) (int, error) {
	return 0, nil
}
func (f *FakeTaskStore) ReleaseRecoveryLeases(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (f *FakeTaskStore) CreateWorkflowReceipt(_ context.Context, _, _, _ string) error {
	return nil
}
func (f *FakeTaskStore) ResolveWorkflowReceipts(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (f *FakeTaskStore) GetPendingReceiptsByNode(_ context.Context, _ string) ([]PendingReceipt, error) {
	return nil, nil
}
func (f *FakeTaskStore) CleanupOldResolvedReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (f *FakeTaskStore) TimeoutStaleReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (f *FakeTaskStore) ListFailedTasks(_ context.Context) ([]*Task, error) {
	return nil, nil
}

func (f *FakeTaskStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	return fn(ctx, f)
}

func TestFakeTaskStore_CreateTaskWithDedup(t *testing.T) {
	t.Parallel()

	t.Run("creates task when no dedup key", func(t *testing.T) {
		t.Parallel()

		store := NewFakeTaskStore()
		task := &Task{
			ID:         "task-1",
			WorkflowID: "workflow-1",
			Status:     StatusPending,
		}

		err := store.CreateTaskWithDedup(context.Background(), task)
		require.NoError(t, err)

		store.mu.Lock()
		_, exists := store.tasks["task-1"]
		store.mu.Unlock()
		assert.True(t, exists)
	})

	t.Run("creates task when dedup key exists but task is completed", func(t *testing.T) {
		t.Parallel()

		store := NewFakeTaskStore()

		existingTask := &Task{
			ID:               "existing-task",
			WorkflowID:       "workflow-1",
			Status:           StatusComplete,
			DeduplicationKey: "dedup-key-1",
		}
		err := store.CreateTask(context.Background(), existingTask)
		require.NoError(t, err)

		newTask := &Task{
			ID:               "new-task",
			WorkflowID:       "workflow-2",
			Status:           StatusPending,
			DeduplicationKey: "dedup-key-1",
		}

		err = store.CreateTaskWithDedup(context.Background(), newTask)
		require.NoError(t, err)
	})

	t.Run("blocks duplicate when active task exists", func(t *testing.T) {
		t.Parallel()

		store := NewFakeTaskStore()

		existingTask := &Task{
			ID:               "existing-task",
			WorkflowID:       "workflow-1",
			Status:           StatusPending,
			DeduplicationKey: "dedup-key-2",
		}
		err := store.CreateTask(context.Background(), existingTask)
		require.NoError(t, err)

		newTask := &Task{
			ID:               "new-task",
			WorkflowID:       "workflow-2",
			Status:           StatusPending,
			DeduplicationKey: "dedup-key-2",
		}

		err = store.CreateTaskWithDedup(context.Background(), newTask)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDuplicateTask)
	})

	t.Run("blocks duplicate for all active statuses", func(t *testing.T) {
		t.Parallel()

		activeStatuses := []TaskStatus{StatusScheduled, StatusPending, StatusProcessing, StatusRetrying}

		for _, status := range activeStatuses {
			t.Run(string(status), func(t *testing.T) {
				t.Parallel()

				store := NewFakeTaskStore()

				existingTask := &Task{
					ID:               "existing-" + string(status),
					WorkflowID:       "workflow-1",
					Status:           status,
					DeduplicationKey: "dedup-" + string(status),
				}
				err := store.CreateTask(context.Background(), existingTask)
				require.NoError(t, err)

				newTask := &Task{
					ID:               "new-" + string(status),
					WorkflowID:       "workflow-2",
					Status:           StatusPending,
					DeduplicationKey: "dedup-" + string(status),
				}

				err = store.CreateTaskWithDedup(context.Background(), newTask)
				require.ErrorIs(t, err, ErrDuplicateTask)
			})
		}
	})
}

func TestFakeTaskStore_RecoverStaleTasks(t *testing.T) {
	t.Parallel()

	t.Run("recovers stale processing tasks", func(t *testing.T) {
		t.Parallel()

		store := NewFakeTaskStore()

		staleTask := &Task{
			ID:         "stale-task",
			WorkflowID: "workflow-1",
			Status:     StatusProcessing,
			UpdatedAt:  time.Now().Add(-20 * time.Minute),
			Attempt:    1,
		}
		err := store.CreateTask(context.Background(), staleTask)
		require.NoError(t, err)

		freshTask := &Task{
			ID:         "fresh-task",
			WorkflowID: "workflow-2",
			Status:     StatusProcessing,
			UpdatedAt:  time.Now(),
			Attempt:    1,
		}
		err = store.CreateTask(context.Background(), freshTask)
		require.NoError(t, err)

		count, err := store.RecoverStaleTasks(context.Background(), 10*time.Minute, 3, "recovered by test")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		store.mu.Lock()
		recovered := store.tasks["stale-task"]
		notRecovered := store.tasks["fresh-task"]
		store.mu.Unlock()

		assert.Equal(t, StatusRetrying, recovered.Status)
		assert.Equal(t, 2, recovered.Attempt)
		assert.Equal(t, "recovered by test", recovered.LastError)

		assert.Equal(t, StatusProcessing, notRecovered.Status)
	})

	t.Run("marks task as failed when max retries exceeded", func(t *testing.T) {
		t.Parallel()

		store := NewFakeTaskStore()

		staleTask := &Task{
			ID:         "stale-task-max-retries",
			WorkflowID: "workflow-1",
			Status:     StatusProcessing,
			UpdatedAt:  time.Now().Add(-20 * time.Minute),
			Attempt:    3,
		}
		err := store.CreateTask(context.Background(), staleTask)
		require.NoError(t, err)

		count, err := store.RecoverStaleTasks(context.Background(), 10*time.Minute, 3, "max retries exceeded")
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		store.mu.Lock()
		recovered := store.tasks["stale-task-max-retries"]
		store.mu.Unlock()

		assert.Equal(t, StatusFailed, recovered.Status)
		assert.Equal(t, 3, recovered.Attempt)
	})
}

func TestFakeTaskStore_GetStaleProcessingTaskCount(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()

	tasks := []*Task{
		{ID: "stale-1", Status: StatusProcessing, UpdatedAt: time.Now().Add(-20 * time.Minute)},
		{ID: "stale-2", Status: StatusProcessing, UpdatedAt: time.Now().Add(-15 * time.Minute)},
		{ID: "fresh", Status: StatusProcessing, UpdatedAt: time.Now()},
		{ID: "pending", Status: StatusPending, UpdatedAt: time.Now().Add(-20 * time.Minute)},
	}

	for _, task := range tasks {
		err := store.CreateTask(context.Background(), task)
		require.NoError(t, err)
	}

	count, err := store.GetStaleProcessingTaskCount(context.Background(), 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

type RecordingExecutor struct {
	err            error
	ExecutionOrder []string
	wg             sync.WaitGroup
	mu             sync.Mutex
	expectCalls    int32
	callCount      int32
}

func (e *RecordingExecutor) ExpectCalls(n int) {
	atomic.StoreInt32(&e.expectCalls, int32(n))
	e.wg.Add(n)
}

func (e *RecordingExecutor) Execute(_ context.Context, payload map[string]any) (map[string]any, error) {
	e.mu.Lock()
	taskID, ok := payload["taskID"].(string)
	if !ok {
		panic("test: payload missing taskID string")
	}
	e.ExecutionOrder = append(e.ExecutionOrder, taskID)
	e.mu.Unlock()

	if atomic.AddInt32(&e.callCount, 1) <= atomic.LoadInt32(&e.expectCalls) {
		e.wg.Done()
	}
	return map[string]any{"status": "ok"}, e.err
}

func setupFullServiceTest(t *testing.T) (*orchestratorService, *RecordingExecutor, *FakeTaskStore) {
	t.Helper()

	fakeStore := NewFakeTaskStore()
	executor := &RecordingExecutor{}

	mockDispatcher := NewMockTaskDispatcher()
	mockDispatcher.RegisterExecutor(context.Background(), "recorder", executor)

	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(10*time.Second),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
		WithTaskDispatcher(mockDispatcher),
	))

	err := service.RegisterExecutor(context.Background(), "recorder", executor)
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	go service.Run(ctx)

	t.Cleanup(func() {
		cancel(fmt.Errorf("test: cleanup"))
		service.Stop()
	})

	return service, executor, fakeStore
}

func TestService_DispatchAndCompletion(t *testing.T) {
	t.Parallel()

	service, recorder, _ := setupFullServiceTest(t)
	ctx := context.Background()

	taskID := "task-completion-001"
	task := NewTask("recorder", map[string]any{"taskID": taskID})
	task.ID = taskID
	task.WorkflowID = taskID

	recorder.ExpectCalls(1)

	_, err := service.Dispatch(ctx, task)
	require.NoError(t, err)

	waitChan := make(chan struct{})
	go func() {
		recorder.wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task to be processed")
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	require.Contains(t, recorder.ExecutionOrder, taskID)
}

func TestService_BatchInsertion(t *testing.T) {
	t.Parallel()

	service, recorder, fakeStore := setupFullServiceTest(t)
	ctx := context.Background()

	taskID1 := "batch-task-1"
	task1 := NewTask("recorder", map[string]any{"taskID": taskID1})
	task1.ID = taskID1
	task1.WorkflowID = taskID1

	taskID2 := "batch-task-2"
	task2 := NewTask("recorder", map[string]any{"taskID": taskID2})
	task2.ID = taskID2
	task2.WorkflowID = taskID2

	recorder.ExpectCalls(2)

	_, err1 := service.Dispatch(ctx, task1)
	_, err2 := service.Dispatch(ctx, task2)
	require.NoError(t, err1)
	require.NoError(t, err2)

	waitChan := make(chan struct{})
	go func() {
		recorder.wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for tasks to be processed")
	}

	fakeStore.mu.Lock()
	defer fakeStore.mu.Unlock()
	assert.Contains(t, fakeStore.tasks, taskID1)
	assert.Contains(t, fakeStore.tasks, taskID2)
}

func TestService_WorkflowFailure(t *testing.T) {
	t.Parallel()

	service, recorder, _ := setupFullServiceTest(t)
	ctx := context.Background()

	expectedErr := errors.New("execution failed")
	recorder.err = expectedErr

	taskID := "task-fail-001"
	task := NewTask("recorder", map[string]any{"taskID": taskID})
	task.ID = taskID
	task.WorkflowID = taskID
	task.Config.MaxRetries = 1
	recorder.ExpectCalls(1)

	_, err := service.Dispatch(ctx, task)
	require.NoError(t, err)

	waitChan := make(chan struct{})
	go func() {
		recorder.wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for task to be processed")
	}

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	require.Contains(t, recorder.ExecutionOrder, taskID)
}

func TestService_BatchInsertSize(t *testing.T) {
	t.Parallel()

	fakeStore := NewFakeTaskStore()
	executor := &RecordingExecutor{}

	batchSize := 5
	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(1*time.Minute),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
	))
	service.batchSize = batchSize
	service.batchTimeout = 500 * time.Millisecond

	err := service.RegisterExecutor(context.Background(), "recorder", executor)
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	go service.Run(ctx)
	t.Cleanup(func() {
		cancel(fmt.Errorf("test: cleanup"))
		service.Stop()
	})

	for i := range batchSize {
		task := NewTask("recorder", map[string]any{"taskID": fmt.Sprintf("task-%d", i)})
		task.ID = fmt.Sprintf("batch-size-task-%d", i)
		task.WorkflowID = fmt.Sprintf("workflow-%d", i)
		_, err := service.Dispatch(context.Background(), task)
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		fakeStore.mu.Lock()
		count := len(fakeStore.tasks)
		fakeStore.mu.Unlock()
		return count == batchSize
	}, 2*time.Second, 5*time.Millisecond, "All tasks should be inserted when batch size is reached")
}

func TestService_MultipleConcurrentDispatches(t *testing.T) {
	t.Parallel()

	service, executor, _ := setupFullServiceTest(t)
	ctx := context.Background()

	const numTasks = 50
	executor.ExpectCalls(numTasks)

	var dispatchWg sync.WaitGroup
	for i := range numTasks {
		dispatchWg.Add(1)
		go func(index int) {
			defer dispatchWg.Done()
			task := NewTask("recorder", map[string]any{"taskID": fmt.Sprintf("concurrent-task-%d", index)})
			task.ID = fmt.Sprintf("concurrent-task-%d", index)
			task.WorkflowID = fmt.Sprintf("workflow-%d", index)
			_, err := service.Dispatch(ctx, task)
			assert.NoError(t, err)
		}(i)
	}

	dispatchWg.Wait()

	waitChan := make(chan struct{})
	go func() {
		executor.wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for concurrent tasks to be processed")
	}

	executor.mu.Lock()
	defer executor.mu.Unlock()
	assert.Equal(t, numTasks, len(executor.ExecutionOrder))
}

func TestService_MultipleWorkersConcurrency(t *testing.T) {
	t.Parallel()

	fakeStore := NewFakeTaskStore()
	executor := &RecordingExecutor{}

	mockDispatcher := NewMockTaskDispatcher()
	mockDispatcher.RegisterExecutor(context.Background(), "recorder", executor)

	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(10*time.Second),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
		WithTaskDispatcher(mockDispatcher),
	))

	err := service.RegisterExecutor(context.Background(), "recorder", executor)
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(fmt.Errorf("test: cleanup"))

	go service.Run(ctx)
	t.Cleanup(func() {
		cancel(fmt.Errorf("test: cleanup"))
		service.Stop()
	})

	const numTasks = 20
	executor.ExpectCalls(numTasks)

	for i := range numTasks {
		task := NewTask("recorder", map[string]any{"taskID": fmt.Sprintf("task-%d", i)})
		task.ID = fmt.Sprintf("multi-worker-task-%d", i)
		task.WorkflowID = fmt.Sprintf("workflow-%d", i)
		_, err := service.Dispatch(context.Background(), task)
		require.NoError(t, err)
	}

	waitChan := make(chan struct{})
	go func() {
		executor.wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for multi-worker tasks")
	}

	executor.mu.Lock()
	defer executor.mu.Unlock()
	assert.Equal(t, numTasks, len(executor.ExecutionOrder))
}

func TestService_GracefulShutdown(t *testing.T) {
	t.Parallel()

	fakeStore := NewFakeTaskStore()
	executor := &RecordingExecutor{}

	mockDispatcher := NewMockTaskDispatcher()
	mockDispatcher.RegisterExecutor(context.Background(), "recorder", executor)

	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(10*time.Second),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
		WithTaskDispatcher(mockDispatcher),
	))
	err := service.RegisterExecutor(context.Background(), "recorder", executor)
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())

	go service.Run(ctx)

	task := NewTask("recorder", map[string]any{"taskID": "shutdown-task"})
	task.ID = "shutdown-task"
	task.WorkflowID = "shutdown-workflow"
	executor.ExpectCalls(1)

	_, err = service.Dispatch(context.Background(), task)
	require.NoError(t, err)

	executor.wg.Wait()

	cancel(fmt.Errorf("test: cleanup"))
	service.Stop()

	service.Stop()
	service.Stop()

	executor.mu.Lock()
	defer executor.mu.Unlock()
	assert.Contains(t, executor.ExecutionOrder, "shutdown-task")
}

func TestService_StopWithPendingTasks(t *testing.T) {
	t.Parallel()

	fakeStore := NewFakeTaskStore()
	slowExecutor := &SlowExecutor{delay: 50 * time.Millisecond}

	mockDispatcher := NewMockTaskDispatcher()
	mockDispatcher.RegisterExecutor(context.Background(), "slow", slowExecutor)

	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(10*time.Second),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
		WithTaskDispatcher(mockDispatcher),
	))
	err := service.RegisterExecutor(context.Background(), "slow", slowExecutor)
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())

	go service.Run(ctx)

	task := NewTask("slow", map[string]any{"id": "pending"})
	task.ID = "pending-task"
	task.WorkflowID = "pending-workflow"
	slowExecutor.ExpectCalls(1)

	_, err = service.Dispatch(context.Background(), task)
	require.NoError(t, err)

	slowExecutor.wg.Wait()

	cancel(fmt.Errorf("test: cleanup"))

	stopDone := make(chan struct{})
	go func() {
		service.Stop()
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop took too long")
	}

	slowExecutor.mu.Lock()
	completed := slowExecutor.completedCount
	slowExecutor.mu.Unlock()
	assert.Equal(t, 1, completed)
}

func TestService_DuplicateExecutorRegistration(t *testing.T) {
	t.Parallel()

	fakeStore := NewFakeTaskStore()
	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(10*time.Second),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
	))

	executor1 := &RecordingExecutor{}
	executor2 := &RecordingExecutor{}

	err := service.RegisterExecutor(context.Background(), "test-executor", executor1)
	require.NoError(t, err)

	err = service.RegisterExecutor(context.Background(), "test-executor", executor2)
	require.Error(t, err, "Should not allow duplicate executor registration")
	assert.Contains(t, err.Error(), "already registered")
}

func TestService_DispatchBackpressure(t *testing.T) {
	t.Parallel()

	fakeStore := NewFakeTaskStore()
	service := mustCastToOrchestratorService(t, NewService(context.Background(), fakeStore, nil,
		WithSchedulerInterval(10*time.Second),
		WithBatchConfig(testBatchSize, testBatchTimeout),
		WithInsertQueueSize(testInsertQueueSize),
	))

	service.taskInsertChan = make(chan *Task, 2)

	executor := &RecordingExecutor{}
	err := service.RegisterExecutor(context.Background(), "recorder", executor)
	require.NoError(t, err)

	ctx := context.Background()

	task1 := NewTask("recorder", map[string]any{"id": "1"})
	task1.ID = "backpressure-1"
	task1.WorkflowID = "bp-1"

	task2 := NewTask("recorder", map[string]any{"id": "2"})
	task2.ID = "backpressure-2"
	task2.WorkflowID = "bp-2"

	_, err = service.Dispatch(ctx, task1)
	require.NoError(t, err)

	_, err = service.Dispatch(ctx, task2)
	require.NoError(t, err)

	task3 := NewTask("recorder", map[string]any{"id": "3"})
	task3.ID = "backpressure-3"
	task3.WorkflowID = "bp-3"

	_, err = service.Dispatch(ctx, task3)
	assert.Error(t, err, "Should fail when queue is full")
	assert.Contains(t, err.Error(), "overloaded")
}

type SlowExecutor struct {
	wg             sync.WaitGroup
	delay          time.Duration
	mu             sync.Mutex
	completedCount int
	expectCalls    int32
	callCount      int32
}

func (e *SlowExecutor) ExpectCalls(n int) {
	atomic.StoreInt32(&e.expectCalls, int32(n))
	e.wg.Add(n)
}

func (e *SlowExecutor) Execute(_ context.Context, _ map[string]any) (map[string]any, error) {
	time.Sleep(e.delay)
	e.mu.Lock()
	e.completedCount++
	e.mu.Unlock()

	if atomic.AddInt32(&e.callCount, 1) <= atomic.LoadInt32(&e.expectCalls) {
		e.wg.Done()
	}
	return map[string]any{"status": "ok"}, nil
}

type HangingExecutor struct{}

func (e *HangingExecutor) Execute(ctx context.Context, _ map[string]any) (map[string]any, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type AlwaysFailExecutor struct {
	wg          sync.WaitGroup
	expectCalls int32
	callCount   int32
}

func (e *AlwaysFailExecutor) ExpectCalls(n int) {
	atomic.StoreInt32(&e.expectCalls, int32(n))
	e.wg.Add(n)
}

func (e *AlwaysFailExecutor) Execute(_ context.Context, _ map[string]any) (map[string]any, error) {
	if atomic.AddInt32(&e.callCount, 1) <= atomic.LoadInt32(&e.expectCalls) {
		defer e.wg.Done()
	}
	return nil, errors.New("always fails")
}
