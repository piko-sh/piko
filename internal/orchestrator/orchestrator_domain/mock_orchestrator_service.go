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

// MockOrchestratorService is a test double for OrchestratorService where
// nil function fields return zero values and call counts are tracked
// atomically.
type MockOrchestratorService struct {
	// RegisterExecutorFunc is the function called by
	// RegisterExecutor.
	RegisterExecutorFunc func(ctx context.Context, name string, executor TaskExecutor) error

	// DispatchFunc is the function called by Dispatch.
	DispatchFunc func(ctx context.Context, task *Task) (*WorkflowReceipt, error)

	// ScheduleFunc is the function called by Schedule.
	ScheduleFunc func(ctx context.Context, task *Task, executeAt time.Time) (*WorkflowReceipt, error)

	// RunFunc is the function called by Run.
	RunFunc func(ctx context.Context)

	// StopFunc is the function called by Stop.
	StopFunc func()

	// ActiveTasksFunc is the function called by
	// ActiveTasks.
	ActiveTasksFunc func(ctx context.Context) int64

	// PendingTasksFunc is the function called by
	// PendingTasks.
	PendingTasksFunc func(ctx context.Context) int64

	// GetTaskDispatcherFunc is the function called by
	// GetTaskDispatcher.
	GetTaskDispatcherFunc func() TaskDispatcher

	// DispatchDirectFunc is the function called by
	// DispatchDirect.
	DispatchDirectFunc func(ctx context.Context, task *Task) (*WorkflowReceipt, error)

	// RegisterExecutorCallCount tracks how many times
	// RegisterExecutor was called.
	RegisterExecutorCallCount int64

	// DispatchCallCount tracks how many times Dispatch
	// was called.
	DispatchCallCount int64

	// ScheduleCallCount tracks how many times Schedule
	// was called.
	ScheduleCallCount int64

	// RunCallCount tracks how many times Run was
	// called.
	RunCallCount int64

	// StopCallCount tracks how many times Stop was
	// called.
	StopCallCount int64

	// ActiveTasksCallCount tracks how many times
	// ActiveTasks was called.
	ActiveTasksCallCount int64

	// PendingTasksCallCount tracks how many times
	// PendingTasks was called.
	PendingTasksCallCount int64

	// GetTaskDispatcherCallCount tracks how many times
	// GetTaskDispatcher was called.
	GetTaskDispatcherCallCount int64

	// DispatchDirectCallCount tracks how many times
	// DispatchDirect was called.
	DispatchDirectCallCount int64
}

// RegisterExecutor registers a task executor under the given name.
//
// Takes ctx (context.Context) which carries logging context.
// Takes name (string) which identifies the executor.
// Takes executor (TaskExecutor) which is the executor to register.
//
// Returns error, or nil if RegisterExecutorFunc is nil.
func (m *MockOrchestratorService) RegisterExecutor(ctx context.Context, name string, executor TaskExecutor) error {
	atomic.AddInt64(&m.RegisterExecutorCallCount, 1)
	if m.RegisterExecutorFunc != nil {
		return m.RegisterExecutorFunc(ctx, name, executor)
	}
	return nil
}

// Dispatch submits a task for immediate execution.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to dispatch.
//
// Returns (*WorkflowReceipt, error), or (nil, nil) if DispatchFunc is nil.
func (m *MockOrchestratorService) Dispatch(ctx context.Context, task *Task) (*WorkflowReceipt, error) {
	atomic.AddInt64(&m.DispatchCallCount, 1)
	if m.DispatchFunc != nil {
		return m.DispatchFunc(ctx, task)
	}
	return nil, nil
}

// Schedule submits a task for execution at a future time.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to schedule.
// Takes executeAt (time.Time) which is the time to execute the task.
//
// Returns (*WorkflowReceipt, error), or (nil, nil) if ScheduleFunc is nil.
func (m *MockOrchestratorService) Schedule(ctx context.Context, task *Task, executeAt time.Time) (*WorkflowReceipt, error) {
	atomic.AddInt64(&m.ScheduleCallCount, 1)
	if m.ScheduleFunc != nil {
		return m.ScheduleFunc(ctx, task, executeAt)
	}
	return nil, nil
}

// Run starts the orchestrator's main processing loop.
func (m *MockOrchestratorService) Run(ctx context.Context) {
	atomic.AddInt64(&m.RunCallCount, 1)
	if m.RunFunc != nil {
		m.RunFunc(ctx)
	}
}

// Stop halts the orchestrator's processing loop.
func (m *MockOrchestratorService) Stop() {
	atomic.AddInt64(&m.StopCallCount, 1)
	if m.StopFunc != nil {
		m.StopFunc()
	}
}

// ActiveTasks returns the number of tasks currently being processed.
//
// Returns int64, or 0 if ActiveTasksFunc is nil.
func (m *MockOrchestratorService) ActiveTasks(ctx context.Context) int64 {
	atomic.AddInt64(&m.ActiveTasksCallCount, 1)
	if m.ActiveTasksFunc != nil {
		return m.ActiveTasksFunc(ctx)
	}
	return 0
}

// PendingTasks returns the number of tasks awaiting processing.
//
// Returns int64, or 0 if PendingTasksFunc is nil.
func (m *MockOrchestratorService) PendingTasks(ctx context.Context) int64 {
	atomic.AddInt64(&m.PendingTasksCallCount, 1)
	if m.PendingTasksFunc != nil {
		return m.PendingTasksFunc(ctx)
	}
	return 0
}

// GetTaskDispatcher returns the underlying task dispatcher.
//
// Returns TaskDispatcher, or nil if GetTaskDispatcherFunc is nil.
func (m *MockOrchestratorService) GetTaskDispatcher() TaskDispatcher {
	atomic.AddInt64(&m.GetTaskDispatcherCallCount, 1)
	if m.GetTaskDispatcherFunc != nil {
		return m.GetTaskDispatcherFunc()
	}
	return nil
}

// DispatchDirect submits a task bypassing the delayed publisher.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes task (*Task) which is the task to dispatch directly.
//
// Returns (*WorkflowReceipt, error), or (nil, nil) if DispatchDirectFunc
// is nil.
func (m *MockOrchestratorService) DispatchDirect(ctx context.Context, task *Task) (*WorkflowReceipt, error) {
	atomic.AddInt64(&m.DispatchDirectCallCount, 1)
	if m.DispatchDirectFunc != nil {
		return m.DispatchDirectFunc(ctx, task)
	}
	return nil, nil
}
