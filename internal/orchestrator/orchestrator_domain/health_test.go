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
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type mockTaskStoreHealth struct {
	healthState healthprobe_dto.State
}

func (m *mockTaskStoreHealth) CreateTask(_ context.Context, _ *Task) error    { return nil }
func (m *mockTaskStoreHealth) CreateTasks(_ context.Context, _ []*Task) error { return nil }
func (m *mockTaskStoreHealth) UpdateTask(_ context.Context, _ *Task) error    { return nil }
func (m *mockTaskStoreHealth) FetchAndMarkDueTasks(_ context.Context, _ TaskPriority, _ int) ([]*Task, error) {
	return nil, nil
}
func (m *mockTaskStoreHealth) GetWorkflowStatus(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (m *mockTaskStoreHealth) PendingTaskCount(_ context.Context) (int64, error)    { return 0, nil }
func (m *mockTaskStoreHealth) PromoteScheduledTasks(_ context.Context) (int, error) { return 0, nil }
func (m *mockTaskStoreHealth) CreateTaskWithDedup(_ context.Context, _ *Task) error { return nil }
func (m *mockTaskStoreHealth) RecoverStaleTasks(_ context.Context, _ time.Duration, _ int, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) GetStaleProcessingTaskCount(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) UpdateTaskHeartbeat(_ context.Context, _ string) error { return nil }
func (m *mockTaskStoreHealth) ClaimStaleTasksForRecovery(_ context.Context, _ string, _ time.Duration, _ time.Duration, _ int) ([]RecoveryClaimedTask, error) {
	return nil, nil
}
func (m *mockTaskStoreHealth) RecoverClaimedTasks(_ context.Context, _ string, _ int, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) ReleaseRecoveryLeases(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) CreateWorkflowReceipt(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockTaskStoreHealth) ResolveWorkflowReceipts(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) GetPendingReceiptsByNode(_ context.Context, _ string) ([]PendingReceipt, error) {
	return nil, nil
}
func (m *mockTaskStoreHealth) CleanupOldResolvedReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) TimeoutStaleReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTaskStoreHealth) ListFailedTasks(_ context.Context) ([]*Task, error) {
	return nil, nil
}

func (m *mockTaskStoreHealth) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	return fn(ctx, m)
}

func (m *mockTaskStoreHealth) Name() string {
	return "MockTaskStore"
}

func (m *mockTaskStoreHealth) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:    m.Name(),
		State:   m.healthState,
		Message: "mock status",
	}
}

type mockTaskDispatcherHealth struct {
	activeWorkers int32
}

func (m *mockTaskDispatcherHealth) Dispatch(_ context.Context, _ *Task) error { return nil }
func (m *mockTaskDispatcherHealth) DispatchDelayed(_ context.Context, _ *Task, _ time.Time) error {
	return nil
}
func (m *mockTaskDispatcherHealth) RegisterExecutor(_ context.Context, _ string, _ TaskExecutor) {}
func (m *mockTaskDispatcherHealth) Start(_ context.Context) error                                { return nil }
func (m *mockTaskDispatcherHealth) Stop()                                                        {}
func (m *mockTaskDispatcherHealth) Stats() DispatcherStats {
	return DispatcherStats{ActiveWorkers: m.activeWorkers}
}
func (m *mockTaskDispatcherHealth) IsIdle() bool { return m.activeWorkers == 0 }
func (m *mockTaskDispatcherHealth) FailedTasks(_ context.Context) ([]FailedTaskSummary, error) {
	return nil, nil
}
func (m *mockTaskDispatcherHealth) SetBuildTag(_ string) {}
func (m *mockTaskDispatcherHealth) BuildTag() string     { return "" }

func newTestOrchestratorService(store TaskStore, dispatcher TaskDispatcher) *orchestratorService {
	return &orchestratorService{
		taskStore:      store,
		taskDispatcher: dispatcher,
		executors:      make(map[string]TaskExecutor),
		receipts:       make(map[string][]*WorkflowReceipt),
		executorsMutex: sync.RWMutex{},
		stopMutex:      sync.Mutex{},
		receiptsMutex:  sync.Mutex{},
		isStopped:      false,
	}
}

func TestOrchestratorService_Name(t *testing.T) {
	t.Parallel()

	service := newTestOrchestratorService(nil, nil)
	name := service.Name()

	if name != "OrchestratorService" {
		t.Errorf("expected 'OrchestratorService', got %s", name)
	}
}

func TestOrchestratorService_Check_Liveness_Running(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{}
	service := newTestOrchestratorService(store, nil)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.Name != "OrchestratorService" {
		t.Errorf("Name: expected 'OrchestratorService', got %s", status.Name)
	}
	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("State: expected Healthy, got %v", status.State)
	}
	if status.Message != "Orchestrator service is running" {
		t.Errorf("Message: expected 'Orchestrator service is running', got %s", status.Message)
	}
}

func TestOrchestratorService_Check_Liveness_Stopped(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{}
	service := newTestOrchestratorService(store, nil)
	service.isStopped = true

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("State: expected Unhealthy, got %v", status.State)
	}
	if status.Message != "Orchestrator service has been stopped" {
		t.Errorf("Message: expected 'Orchestrator service has been stopped', got %s", status.Message)
	}
}

func TestOrchestratorService_Check_Liveness_NilTaskStore(t *testing.T) {
	t.Parallel()

	service := newTestOrchestratorService(nil, nil)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("State: expected Unhealthy, got %v", status.State)
	}
	if status.Message != "Task store is not initialised" {
		t.Errorf("Message: expected 'Task store is not initialised', got %s", status.Message)
	}
}

func TestOrchestratorService_Check_Readiness_WithExecutors(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateHealthy}
	dispatcher := &mockTaskDispatcherHealth{activeWorkers: 5}
	service := newTestOrchestratorService(store, dispatcher)
	service.executors["test-executor"] = nil

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("State: expected Healthy, got %v", status.State)
	}
	if status.Message != "Orchestrator ready with 1 executor(s)" {
		t.Errorf("Message: expected 'Orchestrator ready with 1 executor(s)', got %s", status.Message)
	}
}

func TestOrchestratorService_Check_Readiness_NoExecutors(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateHealthy}
	dispatcher := &mockTaskDispatcherHealth{activeWorkers: 0}
	service := newTestOrchestratorService(store, dispatcher)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateDegraded {
		t.Errorf("State: expected Degraded, got %v", status.State)
	}
	if status.Message != "Orchestrator running but no executors registered" {
		t.Errorf("Message: expected 'Orchestrator running but no executors registered', got %s", status.Message)
	}
}

func TestOrchestratorService_Check_Readiness_Stopped(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateHealthy}
	service := newTestOrchestratorService(store, nil)
	service.executors["test-executor"] = nil
	service.isStopped = true

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("State: expected Unhealthy, got %v", status.State)
	}
	if status.Message != "Orchestrator service has been stopped" {
		t.Errorf("Message: expected 'Orchestrator service has been stopped', got %s", status.Message)
	}
}

func TestOrchestratorService_Check_Readiness_Dependencies(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateHealthy}
	dispatcher := &mockTaskDispatcherHealth{activeWorkers: 3}
	service := newTestOrchestratorService(store, dispatcher)
	service.executors["test-executor"] = nil

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if len(status.Dependencies) < 1 {
		t.Fatal("expected at least 1 dependency")
	}

	activeTasksDep := status.Dependencies[0]
	if activeTasksDep.Name != "Active Tasks" {
		t.Errorf("expected 'Active Tasks' dependency, got %s", activeTasksDep.Name)
	}
	if activeTasksDep.Message != "3 task(s) currently processing" {
		t.Errorf("unexpected active tasks message: %s", activeTasksDep.Message)
	}

	if len(status.Dependencies) >= 2 {
		storeDep := status.Dependencies[1]
		if storeDep.Name != "MockTaskStore" {
			t.Errorf("expected 'MockTaskStore' dependency, got %s", storeDep.Name)
		}
	}
}

func TestOrchestratorService_Check_Readiness_UnhealthyTaskStore(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateUnhealthy}
	dispatcher := &mockTaskDispatcherHealth{activeWorkers: 0}
	service := newTestOrchestratorService(store, dispatcher)
	service.executors["test-executor"] = nil

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("State: expected Unhealthy due to task store, got %v", status.State)
	}
	if status.Message != "Orchestrator unhealthy: task store unavailable" {
		t.Errorf("unexpected message: %s", status.Message)
	}
}

func TestOrchestratorService_Check_Readiness_DegradedTaskStore(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateDegraded}
	dispatcher := &mockTaskDispatcherHealth{activeWorkers: 0}
	service := newTestOrchestratorService(store, dispatcher)
	service.executors["test-executor"] = nil

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateDegraded {
		t.Errorf("State: expected Degraded due to task store, got %v", status.State)
	}
	if status.Message != "Orchestrator degraded: task store issues" {
		t.Errorf("unexpected message: %s", status.Message)
	}
}

func TestOrchestratorService_Check_Readiness_NoDispatcher(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{healthState: healthprobe_dto.StateHealthy}
	service := newTestOrchestratorService(store, nil)
	service.executors["test-executor"] = nil

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("State: expected Healthy, got %v", status.State)
	}

	if len(status.Dependencies) > 0 {
		activeTasksDep := status.Dependencies[0]
		if activeTasksDep.Message != "0 task(s) currently processing" {
			t.Errorf("expected 0 tasks, got: %s", activeTasksDep.Message)
		}
	}
}

func TestOrchestratorService_determineReadinessState(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{}

	testCases := []struct {
		name          string
		expectedState healthprobe_dto.State
		executorCount int
		stopped       bool
	}{
		{
			name:          "running with executors",
			stopped:       false,
			executorCount: 5,
			expectedState: healthprobe_dto.StateHealthy,
		},
		{
			name:          "running without executors",
			stopped:       false,
			executorCount: 0,
			expectedState: healthprobe_dto.StateDegraded,
		},
		{
			name:          "stopped",
			stopped:       true,
			executorCount: 5,
			expectedState: healthprobe_dto.StateUnhealthy,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state, _ := service.determineReadinessState(tc.stopped, tc.executorCount)

			if state != tc.expectedState {
				t.Errorf("expected state %v, got %v", tc.expectedState, state)
			}
		})
	}
}

func TestOrchestratorService_Check_HasTimestamp(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{}
	service := newTestOrchestratorService(store, nil)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestOrchestratorService_Check_HasDuration(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreHealth{}
	service := newTestOrchestratorService(store, nil)

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.Duration == "" {
		t.Error("Duration should be set")
	}
}
