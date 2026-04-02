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

package orchestrator_adapters

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

type mockEventBus struct {
	handlers        map[string][]orchestrator_domain.EventHandler
	publishFunc     func(ctx context.Context, topic string, event orchestrator_domain.Event) error
	publishedEvents []mockPublishedEvent
	mu              sync.Mutex
}

type mockPublishedEvent struct {
	Topic string
	Event orchestrator_domain.Event
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{
		publishedEvents: make([]mockPublishedEvent, 0),
		handlers:        make(map[string][]orchestrator_domain.EventHandler),
	}
}

func (m *mockEventBus) Publish(ctx context.Context, topic string, event orchestrator_domain.Event) error {
	m.mu.Lock()
	m.publishedEvents = append(m.publishedEvents, mockPublishedEvent{
		Topic: topic,
		Event: event,
	})

	handlers := make([]orchestrator_domain.EventHandler, len(m.handlers[topic]))
	copy(handlers, m.handlers[topic])
	m.mu.Unlock()

	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, event)
	}

	for _, handler := range handlers {
		go func(h orchestrator_domain.EventHandler) {
			_ = h(ctx, event)
		}(handler)
	}

	return nil
}

func (m *mockEventBus) Subscribe(ctx context.Context, topic string) (<-chan orchestrator_domain.Event, error) {
	eventChannel := make(chan orchestrator_domain.Event, 10)
	go func() {
		<-ctx.Done()
		close(eventChannel)
	}()
	return eventChannel, nil
}

func (m *mockEventBus) Close(_ context.Context) error {
	return nil
}

func (m *mockEventBus) SubscribeWithHandler(ctx context.Context, topic string, handler orchestrator_domain.EventHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[topic] = append(m.handlers[topic], handler)
	return nil
}

func (m *mockEventBus) getPublishedEvents() []mockPublishedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]mockPublishedEvent, len(m.publishedEvents))
	copy(result, m.publishedEvents)
	return result
}

func (m *mockEventBus) getHandlerCount(topic string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.handlers[topic])
}

type mockExecutor struct {
	executeFunc func(ctx context.Context, payload map[string]any) (map[string]any, error)
	lastPayload map[string]any
	callCount   int
	mu          sync.Mutex
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{}
}

func (m *mockExecutor) Execute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	m.mu.Lock()
	m.callCount++
	m.lastPayload = payload
	execFunc := m.executeFunc
	m.mu.Unlock()

	if execFunc != nil {
		return execFunc(ctx, payload)
	}
	return map[string]any{"status": "success"}, nil
}

func (m *mockExecutor) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func newTrackingDelayedPublisher() *orchestrator_domain.MockDelayedPublisher {
	var count int64
	m := &orchestrator_domain.MockDelayedPublisher{}
	m.ScheduleFunc = func(_ context.Context, _ *orchestrator_domain.Task) error {
		atomic.AddInt64(&count, 1)
		return nil
	}
	m.PendingCountFunc = func() int {
		return int(atomic.LoadInt64(&count))
	}
	return m
}

func Test_watermillTaskDispatcher_Dispatch_RoutesToCorrectTopic(t *testing.T) {
	testCases := []struct {
		name          string
		expectedTopic string
		priority      orchestrator_domain.TaskPriority
	}{
		{
			name:          "high priority routes to high topic",
			priority:      orchestrator_domain.PriorityHigh,
			expectedTopic: orchestrator_domain.TopicTaskDispatchHigh,
		},
		{
			name:          "normal priority routes to normal topic",
			priority:      orchestrator_domain.PriorityNormal,
			expectedTopic: orchestrator_domain.TopicTaskDispatchNormal,
		},
		{
			name:          "low priority routes to low topic",
			priority:      orchestrator_domain.PriorityLow,
			expectedTopic: orchestrator_domain.TopicTaskDispatchLow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eventBus := newMockEventBus()
			config := orchestrator_domain.DefaultDispatcherConfig()

			dispatcher := newWatermillTaskDispatcher(config, eventBus, nil)

			task := &orchestrator_domain.Task{
				ID:         "task-1",
				WorkflowID: "workflow-1",
				Executor:   "test-executor",
				Config: orchestrator_domain.TaskConfig{
					Priority: tc.priority,
				},
			}

			ctx := t.Context()
			err := dispatcher.Dispatch(ctx, task)
			require.NoError(t, err)

			events := eventBus.getPublishedEvents()
			require.Len(t, events, 1)
			assert.Equal(t, tc.expectedTopic, events[0].Topic)
		})
	}
}

func Test_watermillTaskDispatcher_Dispatch_ValidationErrors(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil)
	ctx := t.Context()

	t.Run("nil task returns error", func(t *testing.T) {
		err := dispatcher.Dispatch(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty ID returns error", func(t *testing.T) {
		task := &orchestrator_domain.Task{
			WorkflowID: "workflow-1",
			Executor:   "test-executor",
		}
		err := dispatcher.Dispatch(ctx, task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ID")
	})

	t.Run("empty workflowID returns error", func(t *testing.T) {
		task := &orchestrator_domain.Task{
			ID:       "task-1",
			Executor: "test-executor",
		}
		err := dispatcher.Dispatch(ctx, task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "workflowID")
	})

	t.Run("empty executor returns error", func(t *testing.T) {
		task := &orchestrator_domain.Task{
			ID:         "task-1",
			WorkflowID: "workflow-1",
		}
		err := dispatcher.Dispatch(ctx, task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "executor")
	})
}

func Test_watermillTaskDispatcher_Dispatch_Deduplication(t *testing.T) {
	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true

	dispatcher := newWatermillTaskDispatcher(config, eventBus, store)
	ctx := t.Context()

	t.Run("persists task with deduplication key", func(t *testing.T) {
		task := &orchestrator_domain.Task{
			ID:               "task-1",
			WorkflowID:       "workflow-1",
			Executor:         "test-executor",
			DeduplicationKey: "dedup-key-1",
		}

		err := dispatcher.Dispatch(ctx, task)
		require.NoError(t, err)

		assert.Equal(t, int64(1), atomic.LoadInt64(&store.CreateTaskWithDedupCallCount))
	})

	t.Run("blocks duplicate task", func(t *testing.T) {
		store.CreateTaskWithDedupFunc = func(_ context.Context, _ *orchestrator_domain.Task) error {
			return orchestrator_domain.ErrDuplicateTask
		}

		task := &orchestrator_domain.Task{
			ID:               "task-2",
			WorkflowID:       "workflow-1",
			Executor:         "test-executor",
			DeduplicationKey: "dedup-key-1",
		}

		err := dispatcher.Dispatch(ctx, task)
		require.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask)
	})
}

func Test_watermillTaskDispatcher_Start_SubscribesHandlers(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.WatermillHighHandlers = 3
	config.WatermillNormalHandlers = 2
	config.WatermillLowHandlers = 1
	config.RecoveryInterval = 0

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	ctx, cancel := context.WithCancelCause(t.Context())

	var startErr error
	var startDone atomic.Bool
	go func() {
		startErr = dispatcher.Start(ctx)
		startDone.Store(true)
	}()

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, eventBus.getHandlerCount(orchestrator_domain.TopicTaskDispatchHigh))
	assert.Equal(t, 1, eventBus.getHandlerCount(orchestrator_domain.TopicTaskDispatchNormal))
	assert.Equal(t, 1, eventBus.getHandlerCount(orchestrator_domain.TopicTaskDispatchLow))

	cancel(fmt.Errorf("test: cleanup"))

	time.Sleep(50 * time.Millisecond)
	assert.True(t, startDone.Load())
	assert.NoError(t, startErr)
}

func Test_watermillTaskDispatcher_IsIdle(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	t.Run("idle when no tasks dispatched", func(t *testing.T) {
		assert.True(t, dispatcher.IsIdle())
	})

	t.Run("not idle with pending tasks", func(t *testing.T) {

		task := &orchestrator_domain.Task{
			ID:         "task-1",
			WorkflowID: "workflow-1",
			Executor:   "test-executor",
		}
		ctx := t.Context()
		err := dispatcher.Dispatch(ctx, task)
		require.NoError(t, err)

		assert.False(t, dispatcher.IsIdle())
	})
}

func Test_watermillTaskDispatcher_IsIdle_AfterTaskFailsWithRetries(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.DefaultMaxRetries = 3
	config.SyncPersistence = true

	delayedPub := newTrackingDelayedPublisher()
	store := &orchestrator_domain.MockTaskStore{}

	dispatcher := newWatermillTaskDispatcher(config, eventBus, store,
		withWatermillDelayedPublisher(delayedPub))

	atomic.StoreInt64(&dispatcher.TasksDispatched, 3)
	atomic.StoreInt64(&dispatcher.TasksCompleted, 0)
	atomic.StoreInt64(&dispatcher.TasksFailed, 1)
	atomic.StoreInt64(&dispatcher.TasksRetried, 2)
	atomic.StoreInt64(&dispatcher.pendingTasks, 0)

	assert.True(t, dispatcher.IsIdle(),
		"dispatcher should be idle when dispatched == completed + failed + retried (3 == 0+1+2)")
}

func Test_watermillTaskDispatcher_IsIdle_AfterRetryThenSuccess(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	atomic.StoreInt64(&dispatcher.TasksDispatched, 2)
	atomic.StoreInt64(&dispatcher.TasksCompleted, 1)
	atomic.StoreInt64(&dispatcher.TasksFailed, 0)
	atomic.StoreInt64(&dispatcher.TasksRetried, 1)
	atomic.StoreInt64(&dispatcher.pendingTasks, 0)

	assert.True(t, dispatcher.IsIdle(),
		"dispatcher should be idle when dispatched == completed + failed + retried (2 == 1+0+1)")
}

func Test_watermillTaskDispatcher_IsIdle_AfterMixedSuccessAndFailure(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.DefaultMaxRetries = 2
	config.SyncPersistence = true

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	atomic.StoreInt64(&dispatcher.TasksDispatched, 5)
	atomic.StoreInt64(&dispatcher.TasksCompleted, 2)
	atomic.StoreInt64(&dispatcher.TasksFailed, 1)
	atomic.StoreInt64(&dispatcher.TasksRetried, 2)
	atomic.StoreInt64(&dispatcher.pendingTasks, 0)

	assert.True(t, dispatcher.IsIdle(),
		"dispatcher should be idle when dispatched == completed + failed + retried (5 == 2+1+2)")
}

func Test_watermillTaskDispatcher_IsIdle_ZeroRetryFailsImmediately(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.DefaultMaxRetries = 0
	config.SyncPersistence = true

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	atomic.StoreInt64(&dispatcher.TasksDispatched, 1)
	atomic.StoreInt64(&dispatcher.TasksCompleted, 0)
	atomic.StoreInt64(&dispatcher.TasksFailed, 1)
	atomic.StoreInt64(&dispatcher.TasksRetried, 0)
	atomic.StoreInt64(&dispatcher.pendingTasks, 0)

	assert.True(t, dispatcher.IsIdle(),
		"dispatcher should be idle when dispatched == completed + failed + retried (1 == 0+1+0)")
}

func Test_watermillTaskDispatcher_IsIdle_NotIdleWithDelayedPending(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	atomic.StoreInt64(&dispatcher.TasksDispatched, 1)
	atomic.StoreInt64(&dispatcher.TasksCompleted, 0)
	atomic.StoreInt64(&dispatcher.TasksFailed, 0)
	atomic.StoreInt64(&dispatcher.TasksRetried, 1)
	atomic.StoreInt64(&dispatcher.pendingTasks, 0)

	_ = delayedPub.Schedule(context.Background(), &orchestrator_domain.Task{
		ID:                 "delayed-1",
		WorkflowID:         "wf-1",
		Executor:           "test",
		ScheduledExecuteAt: time.Now().Add(time.Hour),
	})

	assert.False(t, dispatcher.IsIdle(),
		"dispatcher should NOT be idle when delayed publisher has pending tasks")
}

func Test_watermillTaskDispatcher_IsIdle_NotIdleWithInFlightTasks(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	atomic.StoreInt64(&dispatcher.TasksDispatched, 1)
	atomic.StoreInt64(&dispatcher.TasksCompleted, 1)
	atomic.StoreInt64(&dispatcher.TasksFailed, 0)
	atomic.StoreInt64(&dispatcher.TasksRetried, 0)
	atomic.StoreInt64(&dispatcher.pendingTasks, 0)

	dispatcher.InFlightTasks.Store("task-inflight", &orchestrator_domain.Task{ID: "task-inflight"})

	assert.False(t, dispatcher.IsIdle(),
		"dispatcher should NOT be idle when in-flight tasks exist")

	dispatcher.InFlightTasks.Delete("task-inflight")
	assert.True(t, dispatcher.IsIdle(),
		"dispatcher should be idle after clearing in-flight task")
}

func Test_watermillTaskDispatcher_Stats(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.WatermillHighHandlers = 5
	config.WatermillNormalHandlers = 3
	config.WatermillLowHandlers = 1

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil)

	stats := dispatcher.Stats()

	assert.Equal(t, 0, stats.HighQueueLen)
	assert.Equal(t, 0, stats.NormalQueueLen)
	assert.Equal(t, 0, stats.LowQueueLen)

	assert.Equal(t, 9, stats.TotalWorkers)

	assert.Equal(t, int64(0), stats.TasksDispatched)
	assert.Equal(t, int64(0), stats.TasksCompleted)
	assert.Equal(t, int64(0), stats.TasksFailed)
}

func Test_watermillTaskDispatcher_DispatchDelayed(t *testing.T) {
	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, store,
		withWatermillDelayedPublisher(delayedPub))

	ctx := t.Context()
	executeAt := time.Now().Add(10 * time.Minute)

	task := &orchestrator_domain.Task{
		ID:               "task-1",
		WorkflowID:       "workflow-1",
		Executor:         "test-executor",
		DeduplicationKey: "dedup-key",
	}

	err := dispatcher.DispatchDelayed(ctx, task, executeAt)
	require.NoError(t, err)

	assert.Equal(t, orchestrator_domain.StatusScheduled, task.Status)
	assert.Equal(t, executeAt, task.ScheduledExecuteAt)

	assert.Equal(t, int64(1), atomic.LoadInt64(&store.CreateTaskWithDedupCallCount))

	assert.Equal(t, 1, delayedPub.PendingCount())
}

func Test_watermillTaskDispatcher_ProcessTask_Success(t *testing.T) {
	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true

	dispatcher := newWatermillTaskDispatcher(config, eventBus, store)

	executor := newMockExecutor()
	dispatcher.RegisterExecutor(context.Background(), "test-executor", executor)

	task := &orchestrator_domain.Task{
		ID:         "task-1",
		WorkflowID: "workflow-1",
		Executor:   "test-executor",
		Payload:    map[string]any{"input": "value"},
	}

	ctx := t.Context()
	dispatcher.processTask(ctx, task, 0)

	assert.Equal(t, 1, executor.getCallCount())
	assert.Equal(t, orchestrator_domain.StatusComplete, task.Status)

	events := eventBus.getPublishedEvents()
	require.Len(t, events, 1)
	assert.Equal(t, orchestrator_domain.TopicTaskCompleted, events[0].Topic)
}

func Test_watermillTaskDispatcher_ProcessTask_ExecutorNotFound(t *testing.T) {
	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.DefaultMaxRetries = 0

	dispatcher := newWatermillTaskDispatcher(config, eventBus, store)

	task := &orchestrator_domain.Task{
		ID:         "task-1",
		WorkflowID: "workflow-1",
		Executor:   "nonexistent-executor",
	}

	ctx := t.Context()
	dispatcher.processTask(ctx, task, 0)

	assert.Equal(t, orchestrator_domain.StatusFailed, task.Status)
	assert.Contains(t, task.LastError, "executor not found")
}

func TestCreateTaskDispatcher_SelectsCorrectImplementation(t *testing.T) {
	t.Run("creates Watermill dispatcher when EventBus supports handlers", func(t *testing.T) {
		config := orchestrator_domain.DefaultDispatcherConfig()

		eventBus := newMockEventBus()
		dispatcher := CreateTaskDispatcher(context.Background(), config, eventBus, nil)

		require.NotNil(t, dispatcher)
		_, isWatermill := dispatcher.(*watermillTaskDispatcher)
		assert.True(t, isWatermill)
	})

}

type simpleEventBus struct{}

func (s *simpleEventBus) Publish(ctx context.Context, topic string, event orchestrator_domain.Event) error {
	return nil
}

func (s *simpleEventBus) Subscribe(ctx context.Context, topic string) (<-chan orchestrator_domain.Event, error) {
	eventChannel := make(chan orchestrator_domain.Event)
	return eventChannel, nil
}

func (s *simpleEventBus) Close(_ context.Context) error {
	return nil
}

func (s *simpleEventBus) SubscribeWithHandler(_ context.Context, _ string, _ orchestrator_domain.EventHandler) error {
	return nil
}

func Test_watermillTaskDispatcher_WithClock(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	mockClock := clockpkg.NewMockClock(time.Now())

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillClock(mockClock))

	assert.Equal(t, mockClock, dispatcher.Clock)
}
