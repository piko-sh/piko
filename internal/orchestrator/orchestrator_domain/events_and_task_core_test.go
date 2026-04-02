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
	clockpkg "piko.sh/piko/wdk/clock"
)

func TestEvent_Marshal_Success(t *testing.T) {
	t.Parallel()

	event := &Event{
		Type:    EventType("test.event"),
		Payload: map[string]any{"key": "value", "count": float64(42)},
	}

	data, err := event.Marshal()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"type":"test.event"`)
	assert.Contains(t, string(data), `"key":"value"`)
}

func TestEvent_Marshal_EmptyEvent(t *testing.T) {
	t.Parallel()

	event := &Event{}
	data, err := event.Marshal()
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestEvent_Marshal_NilPayload(t *testing.T) {
	t.Parallel()

	event := &Event{
		Type:    EventType("nil.payload"),
		Payload: nil,
	}
	data, err := event.Marshal()
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestEvent_Unmarshal_Success(t *testing.T) {
	t.Parallel()

	original := &Event{
		Type:    EventType("test.event"),
		Payload: map[string]any{"key": "value"},
	}

	data, err := original.Marshal()
	require.NoError(t, err)

	restored := &Event{}
	err = restored.Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, "value", restored.Payload["key"])
}

func TestEvent_Unmarshal_InvalidJSON(t *testing.T) {
	t.Parallel()

	event := &Event{}
	err := event.Unmarshal([]byte(`{invalid json`))
	require.Error(t, err)
}

func TestEvent_Unmarshal_EmptyBytes(t *testing.T) {
	t.Parallel()

	event := &Event{}
	err := event.Unmarshal([]byte{})
	require.Error(t, err)
}

func TestEvent_MarshalUnmarshal_Roundtrip(t *testing.T) {
	t.Parallel()

	original := &Event{
		Type: EventType("roundtrip.test"),
		Payload: map[string]any{
			"taskId":     "task-123",
			"workflowId": "workflow-456",
			"status":     "success",
			"durationMs": float64(1500),
		},
	}

	data, err := original.Marshal()
	require.NoError(t, err)

	restored := &Event{}
	err = restored.Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, original.Type, restored.Type)
	assert.Equal(t, original.Payload["taskId"], restored.Payload["taskId"])
	assert.Equal(t, original.Payload["workflowId"], restored.Payload["workflowId"])
	assert.Equal(t, original.Payload["status"], restored.Payload["status"])
}

func TestErrorSentinels(t *testing.T) {
	t.Parallel()

	t.Run("ErrExecutorNotFound", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, ErrExecutorNotFound)
		assert.Contains(t, ErrExecutorNotFound.Error(), "executor not found")
	})

	t.Run("ErrTaskFailedMaxRetries", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, ErrTaskFailedMaxRetries)
		assert.Contains(t, ErrTaskFailedMaxRetries.Error(), "max retries")
	})

	t.Run("ErrServiceClosed", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, ErrServiceClosed)
		assert.Contains(t, ErrServiceClosed.Error(), "service is closed")
	})

	t.Run("ErrDuplicateTask", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, ErrDuplicateTask)
		assert.Contains(t, ErrDuplicateTask.Error(), "duplicate")
	})

	t.Run("errors are distinct", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, ErrExecutorNotFound, ErrTaskFailedMaxRetries)
		assert.NotEqual(t, ErrServiceClosed, ErrDuplicateTask)
		assert.False(t, errors.Is(ErrExecutorNotFound, ErrDuplicateTask))
	})
}

func TestTopicConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "task.completed.v1", TopicTaskCompleted)
	assert.Equal(t, "task.dispatch.high.v1", TopicTaskDispatchHigh)
	assert.Equal(t, "task.dispatch.normal.v1", TopicTaskDispatchNormal)
	assert.Equal(t, "task.dispatch.low.v1", TopicTaskDispatchLow)

	topics := []string{TopicTaskCompleted, TopicTaskDispatchHigh, TopicTaskDispatchNormal, TopicTaskDispatchLow}
	seen := make(map[string]bool)
	for _, topic := range topics {
		assert.NotEmpty(t, topic)
		assert.False(t, seen[topic], "duplicate topic: %s", topic)
		seen[topic] = true
	}
}

func TestCompletionEvent_ZeroValue(t *testing.T) {
	t.Parallel()

	var ce CompletionEvent
	assert.Empty(t, ce.TaskID)
	assert.Empty(t, ce.WorkflowID)
	assert.Empty(t, ce.ArtefactID)
	assert.Empty(t, ce.Status)
	assert.Empty(t, ce.ErrorMessage)
	assert.Zero(t, ce.DurationMs)
	assert.True(t, ce.CompletedAt.IsZero())
}

func TestCompletionEvent_PopulatedFields(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	ce := CompletionEvent{
		TaskID:       "task-1",
		WorkflowID:   "wf-1",
		ArtefactID:   "art-1",
		Status:       "success",
		ErrorMessage: "",
		DurationMs:   1500,
		CompletedAt:  now,
	}

	assert.Equal(t, "task-1", ce.TaskID)
	assert.Equal(t, "wf-1", ce.WorkflowID)
	assert.Equal(t, "art-1", ce.ArtefactID)
	assert.Equal(t, "success", ce.Status)
	assert.Empty(t, ce.ErrorMessage)
	assert.Equal(t, int64(1500), ce.DurationMs)
	assert.Equal(t, now, ce.CompletedAt)
}

func TestDefaultServiceConfig(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()

	assert.Equal(t, 10*time.Second, config.SchedulerInterval)
	assert.Equal(t, 250, config.BatchSize)
	assert.Equal(t, 10*time.Millisecond, config.BatchTimeout)
	assert.Equal(t, 8192, config.InsertQueueSize)
	assert.Nil(t, config.DispatcherConfig)
	assert.Nil(t, config.TaskDispatcher)
	assert.Nil(t, config.Clock)
}

func TestWithSchedulerInterval(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	opt := WithSchedulerInterval(30 * time.Second)
	opt(&config)
	assert.Equal(t, 30*time.Second, config.SchedulerInterval)
}

func TestWithBatchConfig(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	opt := WithBatchConfig(500, 100*time.Millisecond)
	opt(&config)
	assert.Equal(t, 500, config.BatchSize)
	assert.Equal(t, 100*time.Millisecond, config.BatchTimeout)
}

func TestWithInsertQueueSize(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	opt := WithInsertQueueSize(16384)
	opt(&config)
	assert.Equal(t, 16384, config.InsertQueueSize)
}

func TestWithDispatcherConfig(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	dispConfig := DispatcherConfig{
		DefaultTimeout:    10 * time.Minute,
		DefaultMaxRetries: 5,
		NodeID:            "test-node",
	}
	opt := WithDispatcherConfig(dispConfig)
	opt(&config)

	require.NotNil(t, config.DispatcherConfig)
	assert.Equal(t, 10*time.Minute, config.DispatcherConfig.DefaultTimeout)
	assert.Equal(t, 5, config.DispatcherConfig.DefaultMaxRetries)
	assert.Equal(t, "test-node", config.DispatcherConfig.NodeID)
}

func TestWithTaskDispatcher(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	dispatcher := NewMockTaskDispatcher()
	opt := WithTaskDispatcher(dispatcher)
	opt(&config)
	assert.NotNil(t, config.TaskDispatcher)
}

func TestWithServiceClock(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	opt := WithServiceClock(mockClock)
	opt(&config)
	assert.NotNil(t, config.Clock)
}

func TestServiceOptions_Chained(t *testing.T) {
	t.Parallel()

	config := DefaultServiceConfig()
	opts := []ServiceOption{
		WithSchedulerInterval(1 * time.Minute),
		WithBatchConfig(100, 50*time.Millisecond),
		WithInsertQueueSize(2048),
	}
	for _, opt := range opts {
		opt(&config)
	}

	assert.Equal(t, 1*time.Minute, config.SchedulerInterval)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 50*time.Millisecond, config.BatchTimeout)
	assert.Equal(t, 2048, config.InsertQueueSize)
}

func TestNewService_WithDefaults(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	eventBus := &MockEventBus{}
	service := NewService(context.Background(), store, eventBus)
	require.NotNil(t, service)
}

func TestNewService_WithCustomClock(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	mockClock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	service := NewService(context.Background(), store, nil, WithServiceClock(mockClock))
	require.NotNil(t, service)
}

func TestNewService_WithNodeIDFromDispatcherConfig(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	service := NewService(context.Background(), store, nil, WithDispatcherConfig(DispatcherConfig{
		NodeID: "custom-node-id",
	}))
	impl, ok := service.(*orchestratorService)
	require.True(t, ok)
	assert.Equal(t, "custom-node-id", impl.nodeID)
}

func TestNewService_GeneratesNodeIDWhenNotProvided(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	service := NewService(context.Background(), store, nil)
	impl, ok := service.(*orchestratorService)
	require.True(t, ok)
	assert.NotEmpty(t, impl.nodeID)
}

func TestService_ActiveTasks_NoDispatcher(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{
		taskDispatcher: nil,
	}
	result := service.ActiveTasks(context.Background())
	assert.Equal(t, int64(0), result)
}

func TestService_ActiveTasks_WithDispatcher(t *testing.T) {
	t.Parallel()

	dispatcher := NewMockTaskDispatcher()
	dispatcher.StatsFunc = func() DispatcherStats {
		return DispatcherStats{ActiveWorkers: 7}
	}
	service := &orchestratorService{
		taskDispatcher: dispatcher,
	}
	result := service.ActiveTasks(context.Background())
	assert.Equal(t, int64(7), result)
}

func TestService_PendingTasks_Error(t *testing.T) {
	t.Parallel()

	store := &mockTaskStoreBatch{}
	service := &orchestratorService{
		taskStore: store,
	}

	result := service.PendingTasks(context.Background())
	assert.Equal(t, int64(0), result)
}

func TestService_GetTaskDispatcher_Nil(t *testing.T) {
	t.Parallel()

	service := &orchestratorService{taskDispatcher: nil}
	assert.Nil(t, service.GetTaskDispatcher())
}

func TestService_GetTaskDispatcher_NonNil(t *testing.T) {
	t.Parallel()

	dispatcher := NewMockTaskDispatcher()
	service := &orchestratorService{taskDispatcher: dispatcher}
	assert.NotNil(t, service.GetTaskDispatcher())
}

func TestNewTaskProcessingCore_Defaults(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)
	require.NotNil(t, core)

	assert.NotNil(t, core.Clock)
	assert.NotEmpty(t, core.NodeID())
	assert.Equal(t, config.DefaultTimeout, core.Config.DefaultTimeout)
	assert.Equal(t, config.DefaultMaxRetries, core.Config.DefaultMaxRetries)
}

func TestNewTaskProcessingCore_CustomClock(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, mockClock)
	assert.Equal(t, mockClock, core.Clock)
}

func TestNewTaskProcessingCore_CustomNodeID(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	config.NodeID = "my-node"
	core := NewTaskProcessingCore(config, nil, nil, nil)
	assert.Equal(t, "my-node", core.NodeID())
}

func TestNewTaskProcessingCore_GeneratedNodeID(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	config.NodeID = ""
	core := NewTaskProcessingCore(config, nil, nil, nil)
	assert.NotEmpty(t, core.NodeID())
}

func TestTaskProcessingCore_ValidateTask(t *testing.T) {
	t.Parallel()

	core := &TaskProcessingCore{}

	t.Run("valid task", func(t *testing.T) {
		t.Parallel()
		task := &Task{ID: "t-1", WorkflowID: "w-1", Executor: "exec"}
		assert.NoError(t, core.ValidateTask(task))
	})

	t.Run("missing ID", func(t *testing.T) {
		t.Parallel()
		task := &Task{ID: "", WorkflowID: "w-1", Executor: "exec"}
		err := core.ValidateTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "task ID is required")
	})

	t.Run("missing WorkflowID", func(t *testing.T) {
		t.Parallel()
		task := &Task{ID: "t-1", WorkflowID: "", Executor: "exec"}
		err := core.ValidateTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "workflowID is required")
	})

	t.Run("missing Executor", func(t *testing.T) {
		t.Parallel()
		task := &Task{ID: "t-1", WorkflowID: "w-1", Executor: ""}
		err := core.ValidateTask(task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor is required")
	})
}

func TestTaskProcessingCore_ApplyDefaults(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	t.Run("sets timeout when zero", func(t *testing.T) {
		t.Parallel()
		task := &Task{Config: TaskConfig{Timeout: 0, MaxRetries: 5}}
		core.ApplyDefaults(task)
		assert.Equal(t, config.DefaultTimeout, task.Config.Timeout)
	})

	t.Run("sets timeout when negative", func(t *testing.T) {
		t.Parallel()
		task := &Task{Config: TaskConfig{Timeout: -1 * time.Second, MaxRetries: 5}}
		core.ApplyDefaults(task)
		assert.Equal(t, config.DefaultTimeout, task.Config.Timeout)
	})

	t.Run("keeps custom timeout", func(t *testing.T) {
		t.Parallel()
		task := &Task{Config: TaskConfig{Timeout: 10 * time.Minute, MaxRetries: 5}}
		core.ApplyDefaults(task)
		assert.Equal(t, 10*time.Minute, task.Config.Timeout)
	})

	t.Run("sets max retries when zero", func(t *testing.T) {
		t.Parallel()
		task := &Task{Config: TaskConfig{Timeout: 5 * time.Minute, MaxRetries: 0}}
		core.ApplyDefaults(task)
		assert.Equal(t, config.DefaultMaxRetries, task.Config.MaxRetries)
	})

	t.Run("sets max retries when negative", func(t *testing.T) {
		t.Parallel()
		task := &Task{Config: TaskConfig{Timeout: 5 * time.Minute, MaxRetries: -1}}
		core.ApplyDefaults(task)
		assert.Equal(t, config.DefaultMaxRetries, task.Config.MaxRetries)
	})

	t.Run("keeps custom max retries", func(t *testing.T) {
		t.Parallel()
		task := &Task{Config: TaskConfig{Timeout: 5 * time.Minute, MaxRetries: 10}}
		core.ApplyDefaults(task)
		assert.Equal(t, 10, task.Config.MaxRetries)
	})

	t.Run("sets status to pending when empty", func(t *testing.T) {
		t.Parallel()
		task := &Task{Status: "", Config: TaskConfig{Timeout: 5 * time.Minute, MaxRetries: 3}}
		core.ApplyDefaults(task)
		assert.Equal(t, StatusPending, task.Status)
	})

	t.Run("keeps existing status", func(t *testing.T) {
		t.Parallel()
		task := &Task{Status: StatusScheduled, Config: TaskConfig{Timeout: 5 * time.Minute, MaxRetries: 3}}
		core.ApplyDefaults(task)
		assert.Equal(t, StatusScheduled, task.Status)
	})
}

func TestTaskProcessingCore_RegisterAndGetExecutor(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	executor := &RecordingExecutor{}
	core.RegisterExecutor(context.Background(), "test-exec", executor)

	assert.Equal(t, 1, core.ExecutorCount())

	retrieved, err := core.GetExecutor("test-exec")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
}

func TestTaskProcessingCore_GetExecutor_NotFound(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	_, err := core.GetExecutor("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "executor not found")
}

func TestTaskProcessingCore_ExecutorCount_Empty(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)
	assert.Equal(t, 0, core.ExecutorCount())
}

func TestTaskProcessingCore_RegisterExecutor_Multiple(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	core.RegisterExecutor(context.Background(), "exec-1", &RecordingExecutor{})
	core.RegisterExecutor(context.Background(), "exec-2", &RecordingExecutor{})
	core.RegisterExecutor(context.Background(), "exec-3", &RecordingExecutor{})

	assert.Equal(t, 3, core.ExecutorCount())
}

func TestTaskProcessingCore_RegisterExecutor_Overwrites(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	exec1 := &RecordingExecutor{}
	exec2 := &RecordingExecutor{}
	core.RegisterExecutor(context.Background(), "test", exec1)
	core.RegisterExecutor(context.Background(), "test", exec2)

	assert.Equal(t, 1, core.ExecutorCount())

	retrieved, err := core.GetExecutor("test")
	require.NoError(t, err)

	assert.Equal(t, exec2, retrieved)
}

func TestTaskProcessingCore_Stats(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	ps := core.Stats()
	assert.Equal(t, int64(0), ps.Dispatched)
	assert.Equal(t, int64(0), ps.Completed)
	assert.Equal(t, int64(0), ps.Failed)
	assert.Equal(t, int64(0), ps.FatalFailed)
	assert.Equal(t, int64(0), ps.Retried)

	atomic.AddInt64(&core.TasksDispatched, 10)
	atomic.AddInt64(&core.TasksCompleted, 7)
	atomic.AddInt64(&core.TasksFailed, 2)
	atomic.AddInt64(&core.TasksFatalFailed, 1)
	atomic.AddInt64(&core.TasksRetried, 1)

	ps = core.Stats()
	assert.Equal(t, int64(10), ps.Dispatched)
	assert.Equal(t, int64(7), ps.Completed)
	assert.Equal(t, int64(2), ps.Failed)
	assert.Equal(t, int64(1), ps.FatalFailed)
	assert.Equal(t, int64(1), ps.Retried)
}

func TestTaskProcessingCore_InFlightCount(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	assert.Equal(t, 0, core.InFlightCount())

	core.InFlightTasks.Store("task-1", &Task{ID: "task-1"})
	core.InFlightTasks.Store("task-2", &Task{ID: "task-2"})
	assert.Equal(t, 2, core.InFlightCount())

	core.InFlightTasks.Delete("task-1")
	assert.Equal(t, 1, core.InFlightCount())
}

func TestTaskProcessingCore_NodeID(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	config.NodeID = "explicit-node"
	core := NewTaskProcessingCore(config, nil, nil, nil)
	assert.Equal(t, "explicit-node", core.NodeID())
}

func TestTaskProcessingCore_recoveryParams(t *testing.T) {
	t.Parallel()

	t.Run("uses configured values", func(t *testing.T) {
		t.Parallel()
		config := DefaultDispatcherConfig()
		config.RecoveryLeaseTimeout = 10 * time.Minute
		config.RecoveryBatchLimit = 200
		core := NewTaskProcessingCore(config, nil, nil, nil)

		leaseTimeout, batchLimit := core.recoveryParams()
		assert.Equal(t, 10*time.Minute, leaseTimeout)
		assert.Equal(t, 200, batchLimit)
	})

	t.Run("uses defaults when zero", func(t *testing.T) {
		t.Parallel()
		config := DefaultDispatcherConfig()
		config.RecoveryLeaseTimeout = 0
		config.RecoveryBatchLimit = 0
		core := NewTaskProcessingCore(config, nil, nil, nil)

		leaseTimeout, batchLimit := core.recoveryParams()
		assert.Equal(t, defaultRecoveryLeaseTimeout, leaseTimeout)
		assert.Equal(t, defaultRecoveryBatchLimit, batchLimit)
	})

	t.Run("uses defaults when negative", func(t *testing.T) {
		t.Parallel()
		config := DefaultDispatcherConfig()
		config.RecoveryLeaseTimeout = -1 * time.Second
		config.RecoveryBatchLimit = -5
		core := NewTaskProcessingCore(config, nil, nil, nil)

		leaseTimeout, batchLimit := core.recoveryParams()
		assert.Equal(t, defaultRecoveryLeaseTimeout, leaseTimeout)
		assert.Equal(t, defaultRecoveryBatchLimit, batchLimit)
	})
}

func TestTaskProcessingCore_Shutdown_NoTasks(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 1*time.Second, fmt.Errorf("test: shutdown with no tasks"))
	defer cancel()

	err := core.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestTaskProcessingCore_Shutdown_Idempotent(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	ctx := context.Background()
	assert.NoError(t, core.Shutdown(ctx))
	assert.NoError(t, core.Shutdown(ctx))
	assert.NoError(t, core.Shutdown(ctx))
}

func TestTaskProcessingCore_PersistTaskUpdate_NilStore(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	core.PersistTaskUpdate(context.Background(), &Task{ID: "task-1"})
}

func TestTaskProcessingCore_PersistTaskUpdate_SyncMode(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	core := NewTaskProcessingCore(config, nil, store, nil)

	task := &Task{
		ID:         "task-sync",
		WorkflowID: "wf-1",
		Status:     StatusComplete,
	}
	core.PersistTaskUpdate(context.Background(), task)

	store.mu.Lock()
	_, exists := store.tasks["task-sync"]
	store.mu.Unlock()
	assert.True(t, exists)
}

func TestTaskProcessingCore_PublishCompletionEvent_NilEventBus(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	core.PublishCompletionEvent(context.Background(), &Task{ID: "t-1", WorkflowID: "w-1"}, nil, 1*time.Second)
}

func TestTaskProcessingCore_PublishCompletionEvent_Success(t *testing.T) {
	t.Parallel()

	var capturedTopic string
	var capturedEvent Event
	eventBus := &MockEventBus{
		PublishFunc: func(_ context.Context, topic string, event Event) error {
			capturedTopic = topic
			capturedEvent = event
			return nil
		},
	}
	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, eventBus, nil, nil)

	task := &Task{
		ID:         "task-1",
		WorkflowID: "workflow-1",
		Payload:    map[string]any{"artefactID": "art-1"},
	}

	core.PublishCompletionEvent(context.Background(), task, nil, 500*time.Millisecond)

	assert.Equal(t, 1, int(atomic.LoadInt64(&eventBus.PublishCallCount)))
	assert.Equal(t, TopicTaskCompleted, capturedTopic)
	assert.Equal(t, "success", capturedEvent.Payload["status"])
	assert.Equal(t, "task-1", capturedEvent.Payload["taskId"])
}

func TestTaskProcessingCore_PublishCompletionEvent_Failure(t *testing.T) {
	t.Parallel()

	var capturedEvent Event
	eventBus := &MockEventBus{
		PublishFunc: func(_ context.Context, _ string, event Event) error {
			capturedEvent = event
			return nil
		},
	}
	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, eventBus, nil, nil)

	task := &Task{ID: "task-fail", WorkflowID: "wf-fail", Payload: map[string]any{}}
	taskErr := errors.New("execution failed")

	core.PublishCompletionEvent(context.Background(), task, taskErr, 1*time.Second)

	assert.Equal(t, 1, int(atomic.LoadInt64(&eventBus.PublishCallCount)))
	assert.Equal(t, "failure", capturedEvent.Payload["status"])
	assert.Equal(t, "execution failed", capturedEvent.Payload["error"])
}

func TestTaskProcessingCore_HandleTaskSuccess(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	eventBus := &MockEventBus{}
	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, eventBus, store, mockClock)

	task := &Task{
		ID:         "task-success",
		WorkflowID: "wf-success",
		Status:     StatusProcessing,
		Attempt:    1,
		Payload:    map[string]any{},
		Result:     map[string]any{"data": "result"},
	}
	core.InFlightTasks.Store(task.ID, task)

	startTime := mockClock.Now()
	core.HandleTaskSuccess(context.Background(), task, startTime)

	assert.Equal(t, StatusComplete, task.Status)
	assert.Empty(t, task.LastError)

	_, loaded := core.InFlightTasks.Load(task.ID)
	assert.False(t, loaded, "task should be removed from in-flight")

	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksCompleted))
}

func TestTaskProcessingCore_HandleTaskFailure_NoRetriesLeft(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	eventBus := &MockEventBus{}
	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	config.DefaultMaxRetries = 3
	core := NewTaskProcessingCore(config, eventBus, store, mockClock)

	task := &Task{
		ID:         "task-fail-final",
		WorkflowID: "wf-fail-final",
		Status:     StatusProcessing,
		Attempt:    3,
		Config:     TaskConfig{MaxRetries: 3},
		Payload:    map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	startTime := mockClock.Now()
	execErr := errors.New("permanent failure")
	core.HandleTaskFailure(context.Background(), task, execErr, startTime)

	assert.Equal(t, StatusFailed, task.Status)
	assert.Equal(t, "permanent failure", task.LastError)
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksFailed))
}

func TestTaskProcessingCore_HandleTaskFailure_WithRetries(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	store := NewFakeTaskStore()
	delayedPub := &MockDelayedPublisher{}
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	config.DefaultMaxRetries = 3
	core := NewTaskProcessingCore(config, nil, store, mockClock)
	core.DelayedPublisher = delayedPub

	task := &Task{
		ID:         "task-retry",
		WorkflowID: "wf-retry",
		Status:     StatusProcessing,
		Attempt:    1,
		Config:     TaskConfig{MaxRetries: 3},
		Payload:    map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	startTime := mockClock.Now()
	execErr := errors.New("temporary failure")
	core.HandleTaskFailure(context.Background(), task, execErr, startTime)

	assert.Equal(t, StatusRetrying, task.Status)
	assert.Equal(t, "temporary failure", task.LastError)
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksRetried))

	_, loaded := core.InFlightTasks.Load(task.ID)
	assert.False(t, loaded, "task should be removed from in-flight")

	assert.Equal(t, 1, int(atomic.LoadInt64(&delayedPub.ScheduleCallCount)))
}

func TestTaskProcessingCore_HandleExecutionResult_Success(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, nil, NewFakeTaskStore(), mockClock)

	task := &Task{
		ID:         "result-ok",
		WorkflowID: "wf-result",
		Status:     StatusProcessing,
		Attempt:    1,
		Payload:    map[string]any{},
		Result:     map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	core.HandleExecutionResult(context.Background(), task, nil, mockClock.Now())
	assert.Equal(t, StatusComplete, task.Status)
}

func TestTaskProcessingCore_HandleExecutionResult_Failure(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	config.DefaultMaxRetries = 1
	core := NewTaskProcessingCore(config, nil, NewFakeTaskStore(), mockClock)

	task := &Task{
		ID:         "result-fail",
		WorkflowID: "wf-result-fail",
		Status:     StatusProcessing,
		Attempt:    1,
		Config:     TaskConfig{MaxRetries: 1},
		Payload:    map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	execErr := errors.New("exec error")
	core.HandleExecutionResult(context.Background(), task, execErr, mockClock.Now())
	assert.Equal(t, StatusFailed, task.Status)
}

func TestTaskProcessingCore_PrepareTaskExecution(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, nil, store, mockClock)

	task := &Task{
		ID:         "prep-task",
		WorkflowID: "wf-prep",
		Status:     StatusPending,
		Attempt:    0,
		Config:     TaskConfig{Timeout: 10 * time.Minute},
		Payload:    map[string]any{},
	}

	timeout := core.PrepareTaskExecution(context.Background(), task)

	assert.Equal(t, 1, task.Attempt)
	assert.Equal(t, StatusProcessing, task.Status)
	assert.Equal(t, 10*time.Minute, timeout)

	_, loaded := core.InFlightTasks.Load(task.ID)
	assert.True(t, loaded, "task should be tracked as in-flight")
}

func TestTaskProcessingCore_PrepareTaskExecution_DefaultTimeout(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, nil, NewFakeTaskStore(), mockClock)

	task := &Task{
		ID:         "prep-default-timeout",
		WorkflowID: "wf",
		Status:     StatusPending,
		Attempt:    0,
		Config:     TaskConfig{Timeout: 0},
		Payload:    map[string]any{},
	}

	timeout := core.PrepareTaskExecution(context.Background(), task)
	assert.Equal(t, config.DefaultTimeout, timeout)
}

func TestTaskProcessingCore_ExecuteTask_Success(t *testing.T) {
	t.Parallel()

	core := &TaskProcessingCore{}
	executor := &RecordingExecutor{}
	task := &Task{
		ID:      "exec-task",
		Payload: map[string]any{"taskID": "exec-task"},
		Result:  map[string]any{},
	}

	err := core.ExecuteTask(context.Background(), task, executor, 5*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, "ok", task.Result["status"])
}

func TestTaskProcessingCore_ExecuteTask_Error(t *testing.T) {
	t.Parallel()

	core := &TaskProcessingCore{}
	executor := &AlwaysFailExecutor{}
	task := &Task{
		ID:      "exec-fail-task",
		Payload: map[string]any{},
		Result:  map[string]any{},
	}

	err := core.ExecuteTask(context.Background(), task, executor, 5*time.Second)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "always fails")
}

func TestTaskProcessingCore_ExecuteTask_Timeout(t *testing.T) {
	t.Parallel()

	core := &TaskProcessingCore{}
	executor := &HangingExecutor{}
	task := &Task{
		ID:      "exec-timeout-task",
		Payload: map[string]any{},
		Result:  map[string]any{},
	}

	err := core.ExecuteTask(context.Background(), task, executor, 50*time.Millisecond)
	assert.Error(t, err)
}

func TestTaskProcessingCore_ReleaseInFlightTasks(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, nil, store, mockClock)

	task1 := &Task{ID: "flight-1", WorkflowID: "wf-1", Status: StatusProcessing, Payload: map[string]any{}}
	task2 := &Task{ID: "flight-2", WorkflowID: "wf-2", Status: StatusProcessing, Payload: map[string]any{}}
	core.InFlightTasks.Store(task1.ID, task1)
	core.InFlightTasks.Store(task2.ID, task2)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, fmt.Errorf("test: releasing in-flight tasks"))
	defer cancel()

	core.ReleaseInFlightTasks(ctx)

	assert.Equal(t, StatusPending, task1.Status)
	assert.Equal(t, StatusPending, task2.Status)
	assert.Equal(t, 0, core.InFlightCount())
}

func TestTaskProcessingCore_RecoverStaleTasks_NilStore(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	count, err := core.RecoverStaleTasks(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTaskProcessingCore_ReleaseRecoveryLeases_NilStore(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	count, err := core.ReleaseRecoveryLeases(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestTaskProcessingCore_PersistWithDedup_NilStore(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	err := core.PersistWithDedup(context.Background(), &Task{ID: "t-1"})
	assert.NoError(t, err)
}

func TestTaskProcessingCore_PersistWithDedup_SyncMode(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	core := NewTaskProcessingCore(config, nil, store, nil)

	task := &Task{
		ID:               "dedup-task",
		WorkflowID:       "wf-dedup",
		Status:           StatusPending,
		DeduplicationKey: "dedup-key",
		Payload:          map[string]any{},
	}

	err := core.PersistWithDedup(context.Background(), task)
	assert.NoError(t, err)

	store.mu.Lock()
	_, exists := store.tasks["dedup-task"]
	store.mu.Unlock()
	assert.True(t, exists)
}

func TestTaskProcessingCore_PersistWithDedup_DuplicateBlocked(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	core := NewTaskProcessingCore(config, nil, store, nil)

	existing := &Task{
		ID:               "existing",
		WorkflowID:       "wf-1",
		Status:           StatusPending,
		DeduplicationKey: "dedup-key",
		Payload:          map[string]any{},
	}
	err := store.CreateTask(context.Background(), existing)
	require.NoError(t, err)

	duplicate := &Task{
		ID:               "duplicate",
		WorkflowID:       "wf-2",
		Status:           StatusPending,
		DeduplicationKey: "dedup-key",
		Payload:          map[string]any{},
	}

	err = core.PersistWithDedup(context.Background(), duplicate)
	assert.ErrorIs(t, err, ErrDuplicateTask)
}

func TestTaskProcessingCore_StartHeartbeat_Disabled(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	config.HeartbeatInterval = 0
	core := NewTaskProcessingCore(config, nil, NewFakeTaskStore(), nil)

	core.StartHeartbeat(context.Background(), "task-1")

	core.stopHeartbeat("task-1")
}

func TestTaskProcessingCore_StartHeartbeat_NilStore(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	config.HeartbeatInterval = 30 * time.Second
	core := NewTaskProcessingCore(config, nil, nil, nil)

	core.StartHeartbeat(context.Background(), "task-1")
}

func TestCompletionEventData_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	ced := completionEventData{
		TaskID:      "task-1",
		WorkflowID:  "wf-1",
		ArtefactID:  "art-1",
		Status:      "success",
		Error:       "",
		DurationMs:  2500,
		CompletedAt: now,
	}

	assert.Equal(t, "task-1", ced.TaskID)
	assert.Equal(t, "wf-1", ced.WorkflowID)
	assert.Equal(t, "art-1", ced.ArtefactID)
	assert.Equal(t, "success", ced.Status)
	assert.Empty(t, ced.Error)
	assert.Equal(t, int64(2500), ced.DurationMs)
	assert.Equal(t, now, ced.CompletedAt)
}

func TestTaskSummary_ZeroValue(t *testing.T) {
	t.Parallel()

	var ts TaskSummary
	assert.Empty(t, ts.Status)
	assert.Zero(t, ts.Count)
}

func TestTaskListItem_ZeroValue(t *testing.T) {
	t.Parallel()

	var item TaskListItem
	assert.Empty(t, item.ID)
	assert.Empty(t, item.WorkflowID)
	assert.Empty(t, item.Executor)
	assert.Empty(t, item.Status)
	assert.Nil(t, item.LastError)
	assert.Zero(t, item.CreatedAt)
	assert.Zero(t, item.UpdatedAt)
	assert.Zero(t, item.Priority)
	assert.Zero(t, item.Attempt)
}

func TestTaskListItem_WithLastError(t *testing.T) {
	t.Parallel()

	item := TaskListItem{
		ID:        "task-1",
		LastError: new("some error"),
	}
	require.NotNil(t, item.LastError)
	assert.Equal(t, "some error", *item.LastError)
}

func TestWorkflowSummary_ZeroValue(t *testing.T) {
	t.Parallel()

	var ws WorkflowSummary
	assert.Empty(t, ws.WorkflowID)
	assert.Zero(t, ws.TaskCount)
	assert.Zero(t, ws.CompleteCount)
	assert.Zero(t, ws.FailedCount)
	assert.Zero(t, ws.ActiveCount)
	assert.Zero(t, ws.CreatedAt)
	assert.Zero(t, ws.UpdatedAt)
}

func TestWorkflowSummary_PopulatedFields(t *testing.T) {
	t.Parallel()

	ws := WorkflowSummary{
		WorkflowID:    "wf-1",
		TaskCount:     10,
		CompleteCount: 7,
		FailedCount:   2,
		ActiveCount:   1,
		CreatedAt:     1000,
		UpdatedAt:     2000,
	}

	assert.Equal(t, "wf-1", ws.WorkflowID)
	assert.Equal(t, int64(10), ws.TaskCount)
	assert.Equal(t, int64(7), ws.CompleteCount)
	assert.Equal(t, int64(2), ws.FailedCount)
	assert.Equal(t, int64(1), ws.ActiveCount)
}

func TestRecoveryClaimedTask_Fields(t *testing.T) {
	t.Parallel()

	rct := RecoveryClaimedTask{
		ID:         "task-1",
		WorkflowID: "wf-1",
		Attempt:    3,
	}

	assert.Equal(t, "task-1", rct.ID)
	assert.Equal(t, "wf-1", rct.WorkflowID)
	assert.Equal(t, int32(3), rct.Attempt)
}

func TestPendingReceipt_Fields(t *testing.T) {
	t.Parallel()

	pr := PendingReceipt{
		ID:         "receipt-1",
		WorkflowID: "wf-1",
		NodeID:     "node-1",
		CreatedAt:  1234567890,
	}

	assert.Equal(t, "receipt-1", pr.ID)
	assert.Equal(t, "wf-1", pr.WorkflowID)
	assert.Equal(t, "node-1", pr.NodeID)
	assert.Equal(t, int64(1234567890), pr.CreatedAt)
}

func TestDispatcherConfig_AllFields(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()

	assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 5*time.Minute, config.RecoveryLeaseTimeout)
	assert.Equal(t, 100, config.RecoveryBatchLimit)
	assert.Empty(t, config.NodeID)
}

func TestDispatcherConfig_CustomValues(t *testing.T) {
	t.Parallel()

	config := DispatcherConfig{
		NodeID:                  "custom-node",
		DefaultTimeout:          1 * time.Minute,
		DefaultMaxRetries:       10,
		RecoveryInterval:        1 * time.Minute,
		StaleTaskThreshold:      30 * time.Minute,
		HeartbeatInterval:       15 * time.Second,
		RecoveryLeaseTimeout:    10 * time.Minute,
		RecoveryBatchLimit:      50,
		WatermillHighHandlers:   20,
		WatermillNormalHandlers: 10,
		WatermillLowHandlers:    4,
		SyncPersistence:         true,
	}

	assert.Equal(t, "custom-node", config.NodeID)
	assert.Equal(t, 1*time.Minute, config.DefaultTimeout)
	assert.Equal(t, 10, config.DefaultMaxRetries)
	assert.Equal(t, 1*time.Minute, config.RecoveryInterval)
	assert.Equal(t, 30*time.Minute, config.StaleTaskThreshold)
	assert.Equal(t, 15*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 10*time.Minute, config.RecoveryLeaseTimeout)
	assert.Equal(t, 50, config.RecoveryBatchLimit)
	assert.Equal(t, 20, config.WatermillHighHandlers)
	assert.Equal(t, 10, config.WatermillNormalHandlers)
	assert.Equal(t, 4, config.WatermillLowHandlers)
	assert.True(t, config.SyncPersistence)
}

func TestDispatcherStats_PopulatedValues(t *testing.T) {
	t.Parallel()

	stats := DispatcherStats{
		HighQueueLen:    5,
		NormalQueueLen:  10,
		LowQueueLen:     2,
		ActiveWorkers:   3,
		TotalWorkers:    17,
		TasksDispatched: 100,
		TasksCompleted:  90,
		TasksFailed:     5,
		TasksRetried:    5,
	}

	assert.Equal(t, 5, stats.HighQueueLen)
	assert.Equal(t, 10, stats.NormalQueueLen)
	assert.Equal(t, 2, stats.LowQueueLen)
	assert.Equal(t, int32(3), stats.ActiveWorkers)
	assert.Equal(t, 17, stats.TotalWorkers)
	assert.Equal(t, int64(100), stats.TasksDispatched)
	assert.Equal(t, int64(90), stats.TasksCompleted)
	assert.Equal(t, int64(5), stats.TasksFailed)
	assert.Equal(t, int64(5), stats.TasksRetried)
}

func TestNewDelayedTaskPublisher(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	dispatchFunc := func(_ context.Context, _ *Task) error { return nil }

	publisher := NewDelayedTaskPublisher(dispatchFunc, mockClock)
	require.NotNil(t, publisher)

	var _ DelayedPublisher = publisher
}

func TestDelayedTaskPublisher_PendingCount_Empty(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(mockClock, nil)
	assert.Equal(t, 0, publisher.PendingCount())
}

func TestDelayedTaskPublisher_PendingCount_WithTasks(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(mockClock, nil)

	for i := range 5 {
		task := &Task{
			ID:                 "task",
			ScheduledExecuteAt: mockClock.Now().Add(time.Duration(i+1) * time.Hour),
		}
		err := publisher.Schedule(context.Background(), task)
		require.NoError(t, err)
	}

	assert.Equal(t, 5, publisher.PendingCount())
}

func TestDelayedTaskPublisher_Stop_NilCancel(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC))
	publisher := NewDelayedTaskPublisherForTesting(mockClock, nil)

	publisher.Stop()
}

func TestService_Schedule_SetsCorrectFields(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	service := NewService(context.Background(), store, nil,
		WithServiceClock(mockClock),
		WithInsertQueueSize(100),
		WithBatchConfig(testBatchSize, testBatchTimeout),
	)

	impl, ok := service.(*orchestratorService)
	if !ok {
		t.Fatal("expected *orchestratorService")
	}

	task := NewTask("test-exec", map[string]any{"key": "val"})
	executeAt := mockClock.Now().Add(2 * time.Hour)

	receipt, err := impl.Schedule(context.Background(), task, executeAt)
	require.NoError(t, err)
	require.NotNil(t, receipt)

	assert.Equal(t, StatusScheduled, task.Status)
	assert.Equal(t, executeAt, task.ExecuteAt)
}

func TestService_DispatchDirect_Success(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	dispatcher := NewMockTaskDispatcher()
	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))

	service := NewService(context.Background(), store, nil,
		WithServiceClock(mockClock),
		WithTaskDispatcher(dispatcher),
		WithInsertQueueSize(100),
	)
	impl, ok := service.(*orchestratorService)
	if !ok {
		t.Fatal("expected *orchestratorService")
	}

	task := NewTask("test-exec", map[string]any{"key": "val"})

	receipt, err := impl.DispatchDirect(context.Background(), task)
	require.NoError(t, err)
	require.NotNil(t, receipt)

	assert.Equal(t, StatusPending, task.Status)
	assert.Equal(t, 1, dispatcher.GetDispatchCallCount())
}

func TestService_DispatchDirect_StoreError(t *testing.T) {
	t.Parallel()

	store := &failingTaskStore{createErr: errors.New("store unavailable")}
	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))

	service := NewService(context.Background(), store, nil,
		WithServiceClock(mockClock),
		WithInsertQueueSize(100),
	)
	impl, ok := service.(*orchestratorService)
	if !ok {
		t.Fatal("expected *orchestratorService")
	}

	task := NewTask("test-exec", map[string]any{"key": "val"})

	receipt, err := impl.DispatchDirect(context.Background(), task)
	assert.Error(t, err)
	assert.Nil(t, receipt)
	assert.Contains(t, err.Error(), "persisting task")
}

func TestService_DispatchDirect_NilDispatcher(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))

	service := NewService(context.Background(), store, nil,
		WithServiceClock(mockClock),
		WithInsertQueueSize(100),
	)
	impl, ok := service.(*orchestratorService)
	if !ok {
		t.Fatal("expected *orchestratorService")
	}

	task := NewTask("test-exec", map[string]any{"key": "val"})

	receipt, err := impl.DispatchDirect(context.Background(), task)
	require.NoError(t, err)
	require.NotNil(t, receipt)
}

func TestTask_DeduplicationKey(t *testing.T) {
	t.Parallel()

	task := NewTask("exec", map[string]any{})
	defer TaskPool.Put(task)
	assert.Empty(t, task.DeduplicationKey)

	task.DeduplicationKey = "my-dedup-key"
	assert.Equal(t, "my-dedup-key", task.DeduplicationKey)

	task.Reset()
	assert.Empty(t, task.DeduplicationKey)
}

func TestTaskProcessingCore_ConcurrentExecutorAccess(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			name := "executor"
			if i%2 == 0 {
				core.RegisterExecutor(context.Background(), name, &RecordingExecutor{})
			} else {
				_, _ = core.GetExecutor(name)
			}
		})
	}
	wg.Wait()
}

func TestTaskProcessingCore_Shutdown_ContextTimeout(t *testing.T) {
	t.Parallel()

	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = false
	core := NewTaskProcessingCore(config, nil, store, nil)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 2*time.Second, fmt.Errorf("test: shutdown with async persistence"))
	defer cancel()

	err := core.Shutdown(ctx)
	assert.NoError(t, err)
}

type failingTaskStore struct {
	createErr error
}

func (f *failingTaskStore) CreateTask(_ context.Context, _ *Task) error { return f.createErr }
func (f *failingTaskStore) CreateTasks(_ context.Context, _ []*Task) error {
	return f.createErr
}
func (f *failingTaskStore) UpdateTask(_ context.Context, _ *Task) error { return nil }
func (f *failingTaskStore) FetchAndMarkDueTasks(_ context.Context, _ TaskPriority, _ int) ([]*Task, error) {
	return nil, nil
}
func (f *failingTaskStore) GetWorkflowStatus(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (f *failingTaskStore) PendingTaskCount(_ context.Context) (int64, error) { return 0, nil }
func (f *failingTaskStore) PromoteScheduledTasks(_ context.Context) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) CreateTaskWithDedup(_ context.Context, _ *Task) error {
	return f.createErr
}
func (f *failingTaskStore) RecoverStaleTasks(_ context.Context, _ time.Duration, _ int, _ string) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) GetStaleProcessingTaskCount(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}
func (f *failingTaskStore) UpdateTaskHeartbeat(_ context.Context, _ string) error { return nil }
func (f *failingTaskStore) ClaimStaleTasksForRecovery(_ context.Context, _ string, _ time.Duration, _ time.Duration, _ int) ([]RecoveryClaimedTask, error) {
	return nil, nil
}
func (f *failingTaskStore) RecoverClaimedTasks(_ context.Context, _ string, _ int, _ string) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) ReleaseRecoveryLeases(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) CreateWorkflowReceipt(_ context.Context, _, _, _ string) error {
	return nil
}
func (f *failingTaskStore) ResolveWorkflowReceipts(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) GetPendingReceiptsByNode(_ context.Context, _ string) ([]PendingReceipt, error) {
	return nil, nil
}
func (f *failingTaskStore) CleanupOldResolvedReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) TimeoutStaleReceipts(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (f *failingTaskStore) ListFailedTasks(_ context.Context) ([]*Task, error) {
	return nil, nil
}

func (f *failingTaskStore) RunAtomic(ctx context.Context, fn func(ctx context.Context, transactionStore TaskStore) error) error {
	return fn(ctx, f)
}

func TestTaskProcessingCore_HandleTaskFailure_FatalError(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	eventBus := &MockEventBus{}
	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	config.DefaultMaxRetries = 3
	core := NewTaskProcessingCore(config, eventBus, store, mockClock)

	task := &Task{
		ID:         "task-fatal",
		WorkflowID: "wf-fatal",
		Status:     StatusProcessing,
		Attempt:    1,
		Config:     TaskConfig{MaxRetries: 3},
		Payload:    map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	startTime := mockClock.Now()
	execErr := NewFatalError(errors.New("parse error: invalid syntax"))
	core.HandleTaskFailure(context.Background(), task, execErr, startTime)

	assert.Equal(t, StatusFailed, task.Status, "fatal error should mark task as failed, not retrying")
	assert.True(t, task.IsFatal, "task.IsFatal should be true for fatal errors")
	assert.Contains(t, task.LastError, "parse error: invalid syntax")
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksFailed))
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksFatalFailed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&core.TasksRetried), "fatal errors should not trigger retries")

	_, loaded := core.InFlightTasks.Load(task.ID)
	assert.False(t, loaded, "task should be removed from in-flight after fatal failure")
}

func TestTaskProcessingCore_HandleTaskFailure_FatalError_FirstAttempt(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	eventBus := &MockEventBus{}
	store := NewFakeTaskStore()
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	config.DefaultMaxRetries = 1
	core := NewTaskProcessingCore(config, eventBus, store, mockClock)

	task := &Task{
		ID:         "task-fatal-first",
		WorkflowID: "wf-fatal-first",
		Status:     StatusProcessing,
		Attempt:    1,
		Config:     TaskConfig{MaxRetries: 1},
		Payload:    map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	startTime := mockClock.Now()
	execErr := NewFatalError(errors.New("parse error: unexpected token"))
	core.HandleTaskFailure(context.Background(), task, execErr, startTime)

	assert.Equal(t, StatusFailed, task.Status)
	assert.True(t, task.IsFatal, "task.IsFatal should be true even when retries would be exhausted")
	assert.Contains(t, task.LastError, "parse error: unexpected token")
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksFailed))
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksFatalFailed))
	assert.Equal(t, int64(0), atomic.LoadInt64(&core.TasksRetried))

	_, loaded := core.InFlightTasks.Load(task.ID)
	assert.False(t, loaded, "task should be removed from in-flight")
}

func TestTaskProcessingCore_HandleTaskFailure_NonFatalError_StillRetries(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC))
	store := NewFakeTaskStore()
	delayedPub := &MockDelayedPublisher{}
	config := DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.HeartbeatInterval = 0
	config.DefaultMaxRetries = 3
	core := NewTaskProcessingCore(config, nil, store, mockClock)
	core.DelayedPublisher = delayedPub

	task := &Task{
		ID:         "task-nonfatal-retry",
		WorkflowID: "wf-nonfatal-retry",
		Status:     StatusProcessing,
		Attempt:    1,
		Config:     TaskConfig{MaxRetries: 3},
		Payload:    map[string]any{},
	}
	core.InFlightTasks.Store(task.ID, task)

	startTime := mockClock.Now()
	execErr := errors.New("transient failure")
	core.HandleTaskFailure(context.Background(), task, execErr, startTime)

	assert.False(t, task.IsFatal, "non-fatal error should not set IsFatal")
	assert.Equal(t, StatusRetrying, task.Status, "non-fatal error with retries remaining should set status to retrying")
	assert.Equal(t, int64(1), atomic.LoadInt64(&core.TasksRetried))
	assert.Equal(t, int64(0), atomic.LoadInt64(&core.TasksFatalFailed), "non-fatal errors should not increment TasksFatalFailed")

	_, loaded := core.InFlightTasks.Load(task.ID)
	assert.False(t, loaded, "task should be removed from in-flight after scheduling retry")

	assert.Equal(t, 1, int(atomic.LoadInt64(&delayedPub.ScheduleCallCount)))
}

func TestTaskProcessingCore_Stats_IncludesFatalFailed(t *testing.T) {
	t.Parallel()

	config := DefaultDispatcherConfig()
	core := NewTaskProcessingCore(config, nil, nil, nil)

	atomic.StoreInt64(&core.TasksFatalFailed, 2)

	ps := core.Stats()
	assert.Equal(t, int64(2), ps.FatalFailed, "Stats() should return TasksFatalFailed in FatalFailed field")
}
