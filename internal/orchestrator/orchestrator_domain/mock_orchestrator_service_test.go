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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockOrchestratorService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockOrchestratorService
	ctx := context.Background()
	task := &Task{ID: "zero-val"}

	require.NoError(t, m.RegisterExecutor(context.Background(), "exec", nil))
	receipt, err := m.Dispatch(ctx, task)
	require.NoError(t, err)
	assert.Nil(t, receipt)

	receipt, err = m.Schedule(ctx, task, time.Now())
	require.NoError(t, err)
	assert.Nil(t, receipt)

	m.Run(ctx)
	m.Stop()

	assert.Equal(t, int64(0), m.ActiveTasks(ctx))
	assert.Equal(t, int64(0), m.PendingTasks(ctx))
	assert.Nil(t, m.GetTaskDispatcher())

	receipt, err = m.DispatchDirect(ctx, task)
	require.NoError(t, err)
	assert.Nil(t, receipt)
}

func TestMockOrchestratorService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	m := &MockOrchestratorService{}
	ctx := context.Background()
	task := &Task{ID: "concurrent"}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = m.RegisterExecutor(context.Background(), "e", nil)
			_, _ = m.Dispatch(ctx, task)
			_, _ = m.Schedule(ctx, task, time.Now())
			m.Run(ctx)
			m.Stop()
			_ = m.ActiveTasks(ctx)
			_ = m.PendingTasks(ctx)
			_ = m.GetTaskDispatcher()
			_, _ = m.DispatchDirect(ctx, task)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RegisterExecutorCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DispatchCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ScheduleCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RunCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.StopCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ActiveTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PendingTasksCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetTaskDispatcherCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.DispatchDirectCallCount))
}

func TestMockOrchestratorService_RegisterExecutor(t *testing.T) {
	t.Parallel()

	t.Run("nil RegisterExecutorFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		err := m.RegisterExecutor(context.Background(), "exec", nil)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RegisterExecutorCallCount))
	})

	t.Run("delegates to RegisterExecutorFunc", func(t *testing.T) {
		t.Parallel()

		var capturedName string
		m := &MockOrchestratorService{
			RegisterExecutorFunc: func(_ context.Context, name string, _ TaskExecutor) error {
				capturedName = name
				return nil
			},
		}

		err := m.RegisterExecutor(context.Background(), "my-executor", nil)
		require.NoError(t, err)
		assert.Equal(t, "my-executor", capturedName)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RegisterExecutorCallCount))
	})

	t.Run("propagates error from RegisterExecutorFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("registration failed")
		m := &MockOrchestratorService{
			RegisterExecutorFunc: func(_ context.Context, _ string, _ TaskExecutor) error {
				return expected
			},
		}

		err := m.RegisterExecutor(context.Background(), "exec", nil)
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockOrchestratorService_Dispatch(t *testing.T) {
	t.Parallel()

	t.Run("nil DispatchFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		receipt, err := m.Dispatch(context.Background(), &Task{ID: "t1"})

		require.NoError(t, err)
		assert.Nil(t, receipt)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DispatchCallCount))
	})

	t.Run("delegates to DispatchFunc", func(t *testing.T) {
		t.Parallel()

		want := &WorkflowReceipt{WorkflowID: "wf-1"}
		var capturedTask *Task
		m := &MockOrchestratorService{
			DispatchFunc: func(_ context.Context, task *Task) (*WorkflowReceipt, error) {
				capturedTask = task
				return want, nil
			},
		}

		task := &Task{ID: "d1"}
		got, err := m.Dispatch(context.Background(), task)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Same(t, task, capturedTask)
	})

	t.Run("propagates error from DispatchFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("dispatch failed")
		m := &MockOrchestratorService{
			DispatchFunc: func(_ context.Context, _ *Task) (*WorkflowReceipt, error) {
				return nil, expected
			},
		}

		_, err := m.Dispatch(context.Background(), &Task{})
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockOrchestratorService_Schedule(t *testing.T) {
	t.Parallel()

	t.Run("nil ScheduleFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		receipt, err := m.Schedule(context.Background(), &Task{}, time.Now())

		require.NoError(t, err)
		assert.Nil(t, receipt)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ScheduleCallCount))
	})

	t.Run("delegates to ScheduleFunc", func(t *testing.T) {
		t.Parallel()

		want := &WorkflowReceipt{WorkflowID: "wf-sched"}
		execAt := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
		var capturedTime time.Time
		m := &MockOrchestratorService{
			ScheduleFunc: func(_ context.Context, _ *Task, at time.Time) (*WorkflowReceipt, error) {
				capturedTime = at
				return want, nil
			},
		}

		got, err := m.Schedule(context.Background(), &Task{}, execAt)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Equal(t, execAt, capturedTime)
	})

	t.Run("propagates error from ScheduleFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("schedule failed")
		m := &MockOrchestratorService{
			ScheduleFunc: func(_ context.Context, _ *Task, _ time.Time) (*WorkflowReceipt, error) {
				return nil, expected
			},
		}

		_, err := m.Schedule(context.Background(), &Task{}, time.Now())
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockOrchestratorService_Run(t *testing.T) {
	t.Parallel()

	t.Run("nil RunFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		m.Run(context.Background())

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunCallCount))
	})

	t.Run("delegates to RunFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockOrchestratorService{
			RunFunc: func(_ context.Context) {
				called = true
			},
		}

		m.Run(context.Background())
		assert.True(t, called)
	})
}

func TestMockOrchestratorService_Stop(t *testing.T) {
	t.Parallel()

	t.Run("nil StopFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		m.Stop()

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.StopCallCount))
	})

	t.Run("delegates to StopFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockOrchestratorService{
			StopFunc: func() {
				called = true
			},
		}

		m.Stop()
		assert.True(t, called)
	})
}

func TestMockOrchestratorService_ActiveTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil ActiveTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		got := m.ActiveTasks(context.Background())

		assert.Equal(t, int64(0), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ActiveTasksCallCount))
	})

	t.Run("delegates to ActiveTasksFunc", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{
			ActiveTasksFunc: func(_ context.Context) int64 {
				return 42
			},
		}

		got := m.ActiveTasks(context.Background())
		assert.Equal(t, int64(42), got)
	})
}

func TestMockOrchestratorService_PendingTasks(t *testing.T) {
	t.Parallel()

	t.Run("nil PendingTasksFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		got := m.PendingTasks(context.Background())

		assert.Equal(t, int64(0), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PendingTasksCallCount))
	})

	t.Run("delegates to PendingTasksFunc", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{
			PendingTasksFunc: func(_ context.Context) int64 {
				return 99
			},
		}

		got := m.PendingTasks(context.Background())
		assert.Equal(t, int64(99), got)
	})
}

func TestMockOrchestratorService_GetTaskDispatcher(t *testing.T) {
	t.Parallel()

	t.Run("nil GetTaskDispatcherFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		got := m.GetTaskDispatcher()

		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetTaskDispatcherCallCount))
	})

	t.Run("delegates to GetTaskDispatcherFunc", func(t *testing.T) {
		t.Parallel()

		want := NewMockTaskDispatcher()
		m := &MockOrchestratorService{
			GetTaskDispatcherFunc: func() TaskDispatcher {
				return want
			},
		}

		got := m.GetTaskDispatcher()
		assert.Same(t, want, got)
	})
}

func TestMockOrchestratorService_DispatchDirect(t *testing.T) {
	t.Parallel()

	t.Run("nil DispatchDirectFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockOrchestratorService{}
		receipt, err := m.DispatchDirect(context.Background(), &Task{})

		require.NoError(t, err)
		assert.Nil(t, receipt)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.DispatchDirectCallCount))
	})

	t.Run("delegates to DispatchDirectFunc", func(t *testing.T) {
		t.Parallel()

		want := &WorkflowReceipt{WorkflowID: "wf-direct"}
		var capturedTask *Task
		m := &MockOrchestratorService{
			DispatchDirectFunc: func(_ context.Context, task *Task) (*WorkflowReceipt, error) {
				capturedTask = task
				return want, nil
			},
		}

		task := &Task{ID: "dd1"}
		got, err := m.DispatchDirect(context.Background(), task)

		require.NoError(t, err)
		assert.Same(t, want, got)
		assert.Same(t, task, capturedTask)
	})

	t.Run("propagates error from DispatchDirectFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("direct dispatch failed")
		m := &MockOrchestratorService{
			DispatchDirectFunc: func(_ context.Context, _ *Task) (*WorkflowReceipt, error) {
				return nil, expected
			},
		}

		_, err := m.DispatchDirect(context.Background(), &Task{})
		assert.ErrorIs(t, err, expected)
	})
}
