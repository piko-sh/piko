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
	"errors"
	"fmt"
	"log/slog"
	"slices"
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
	subscribeErr    error
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
	if m.subscribeErr != nil {
		return m.subscribeErr
	}
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
	var count atomic.Int64
	m := &orchestrator_domain.MockDelayedPublisher{}
	m.ScheduleFunc = func(_ context.Context, _ *orchestrator_domain.Task) error {
		count.Add(1)
		return nil
	}
	m.PendingCountFunc = func() int {
		return int(count.Load())
	}
	return m
}

func startPublishing(t *testing.T, d *watermillTaskDispatcher) {
	t.Helper()
	d.publishBacklog(t.Context())
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
			startPublishing(t, dispatcher)

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

		assert.Equal(t, int64(1), store.CreateTaskWithDedupCallCount.Load())
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

	assert.Equal(t, 1, eventBus.getHandlerCount(orchestrator_domain.TopicTaskDispatchHigh),
		"high topic should register a single subscription handler regardless of WatermillHighHandlers")
	assert.Equal(t, 1, eventBus.getHandlerCount(orchestrator_domain.TopicTaskDispatchNormal),
		"normal topic should register a single subscription handler regardless of WatermillNormalHandlers")
	assert.Equal(t, 1, eventBus.getHandlerCount(orchestrator_domain.TopicTaskDispatchLow),
		"low topic should register a single subscription handler regardless of WatermillLowHandlers")

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

	dispatcher.TasksDispatched.Store(3)
	dispatcher.TasksCompleted.Store(0)
	dispatcher.TasksFailed.Store(1)
	dispatcher.TasksRetried.Store(2)
	dispatcher.pendingTasks.Store(0)

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

	dispatcher.TasksDispatched.Store(2)
	dispatcher.TasksCompleted.Store(1)
	dispatcher.TasksFailed.Store(0)
	dispatcher.TasksRetried.Store(1)
	dispatcher.pendingTasks.Store(0)

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

	dispatcher.TasksDispatched.Store(5)
	dispatcher.TasksCompleted.Store(2)
	dispatcher.TasksFailed.Store(1)
	dispatcher.TasksRetried.Store(2)
	dispatcher.pendingTasks.Store(0)

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

	dispatcher.TasksDispatched.Store(1)
	dispatcher.TasksCompleted.Store(0)
	dispatcher.TasksFailed.Store(1)
	dispatcher.TasksRetried.Store(0)
	dispatcher.pendingTasks.Store(0)

	assert.True(t, dispatcher.IsIdle(),
		"dispatcher should be idle when dispatched == completed + failed + retried (1 == 0+1+0)")
}

func Test_watermillTaskDispatcher_IsIdle_NotIdleWithDelayedPending(t *testing.T) {
	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	delayedPub := newTrackingDelayedPublisher()

	dispatcher := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	dispatcher.TasksDispatched.Store(1)
	dispatcher.TasksCompleted.Store(0)
	dispatcher.TasksFailed.Store(0)
	dispatcher.TasksRetried.Store(1)
	dispatcher.pendingTasks.Store(0)

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

	dispatcher.TasksDispatched.Store(1)
	dispatcher.TasksCompleted.Store(1)
	dispatcher.TasksFailed.Store(0)
	dispatcher.TasksRetried.Store(0)
	dispatcher.pendingTasks.Store(0)

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

	assert.Equal(t, int64(1), store.CreateTaskWithDedupCallCount.Load())

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
	config.DefaultMaxRetries = 3

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
	assert.True(t, task.IsFatal, "a missing executor cannot be fixed by retrying")
	assert.Equal(t, int64(0), dispatcher.Stats().TasksRetried)
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

type recordingSlogHandler struct {
	records []slog.Record
	mu      sync.Mutex
}

func (*recordingSlogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *recordingSlogHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *recordingSlogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *recordingSlogHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *recordingSlogHandler) findMessage(level slog.Level, substring string) (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Level == level && containsString(r.Message, substring) {
			return r, true
		}
	}
	return slog.Record{}, false
}

func containsString(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func withCapturingLogger(t *testing.T) *recordingSlogHandler {
	t.Helper()
	handler := &recordingSlogHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return handler
}

func Test_watermillTaskDispatcher_runRecoveryLoop_LogsWarningOnFailure(t *testing.T) {
	handler := withCapturingLogger(t)

	mockClock := clockpkg.NewMockClock(time.Now())

	config := orchestrator_domain.DefaultDispatcherConfig()
	config.RecoveryInterval = 100 * time.Millisecond

	recoveryError := errors.New("simulated stale recovery failure")
	store := &orchestrator_domain.MockTaskStore{}
	store.RunAtomicFunc = func(ctx context.Context, fn func(ctx context.Context, transactionStore orchestrator_domain.TaskStore) error) error {
		return fn(ctx, store)
	}
	store.ClaimStaleTasksForRecoveryFunc = func(_ context.Context, _ string, _, _ time.Duration, _ int) ([]orchestrator_domain.RecoveryClaimedTask, error) {
		return nil, recoveryError
	}

	eventBus := newMockEventBus()
	dispatcher := newWatermillTaskDispatcher(config, eventBus, store,
		withWatermillClock(mockClock))

	ctx, cancel := context.WithCancelCause(t.Context())
	dispatcher.runCtx = ctx
	dispatcher.cancel = cancel

	dispatcher.wg.Add(1)
	loopDone := make(chan struct{})
	go func() {
		dispatcher.runRecoveryLoop()
		close(loopDone)
	}()

	require.True(t, mockClock.AwaitTimerSetup(0, time.Second),
		"runRecoveryLoop must register its ticker with the mock clock")

	mockClock.Advance(config.RecoveryInterval)

	require.Eventually(t, func() bool {
		_, found := handler.findMessage(slog.LevelWarn, "stale task recovery failed")
		return found
	}, 2*time.Second, 10*time.Millisecond,
		"expected a warning log when stale task recovery fails")

	cancel(errors.New("test cleanup"))
	select {
	case <-loopDone:
	case <-time.After(2 * time.Second):
		require.FailNow(t, "recovery loop did not exit after context cancellation")
	}
}

type runningDispatcher struct {
	cancel context.CancelCauseFunc
	result chan error
	err    error
	once   sync.Once
}

func (r *runningDispatcher) stop() error {
	r.once.Do(func() {
		r.cancel(errors.New("test stopped the dispatcher"))
		r.err = <-r.result
	})
	return r.err
}

func runDispatcher(t *testing.T, d *watermillTaskDispatcher) *runningDispatcher {
	t.Helper()
	ctx, cancel := context.WithCancelCause(context.Background())
	r := &runningDispatcher{cancel: cancel, result: make(chan error, 1)}
	go func() { r.result <- d.Start(ctx) }()
	t.Cleanup(func() { _ = r.stop() })
	return r
}

func dispatcherPhaseOf(d *watermillTaskDispatcher) dispatcherPhase {
	d.phaseMutex.Lock()
	defer d.phaseMutex.Unlock()
	return d.phase
}

type statusRecordingStore struct {
	*orchestrator_domain.MockTaskStore
	statuses []orchestrator_domain.TaskStatus
	mu       sync.Mutex
}

func newStatusRecordingStore(updateErr error) *statusRecordingStore {
	store := &statusRecordingStore{MockTaskStore: &orchestrator_domain.MockTaskStore{}}
	store.UpdateTaskFunc = func(_ context.Context, task *orchestrator_domain.Task) error {
		store.mu.Lock()
		store.statuses = append(store.statuses, task.Status)
		store.mu.Unlock()
		return updateErr
	}
	return store
}

func (s *statusRecordingStore) recordedStatuses() []orchestrator_domain.TaskStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.statuses)
}

func newLifecycleTestConfig() orchestrator_domain.DispatcherConfig {
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true
	config.RecoveryInterval = 0
	config.HeartbeatInterval = 0
	return config
}

func newHighPriorityTask(id string) *orchestrator_domain.Task {
	return &orchestrator_domain.Task{
		ID:               id,
		WorkflowID:       "wf-" + id,
		Executor:         "test-executor",
		DeduplicationKey: "dedup-" + id,
		Config:           orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityHigh},
	}
}

func Test_watermillTaskDispatcher_HoldsTasksUntilSubscribed(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), eventBus, &orchestrator_domain.MockTaskStore{})
	executor := newMockExecutor()
	d.RegisterExecutor(t.Context(), "test-executor", executor)

	require.NoError(t, d.Dispatch(t.Context(), newHighPriorityTask("early")))

	assert.Empty(t, eventBus.getPublishedEvents(),
		"nothing is published while no handler is subscribed to receive it")
	assert.Equal(t, int64(1), d.Stats().TasksDispatched)
	assert.False(t, d.IsIdle(), "a held task is outstanding work")

	runDispatcher(t, d)

	require.Eventually(t, func() bool {
		return d.Stats().TasksCompleted == 1
	}, 5*time.Second, 5*time.Millisecond, "held task should be published and run once subscribed")

	events := eventBus.getPublishedEvents()
	require.NotEmpty(t, events)
	assert.Equal(t, orchestrator_domain.TopicTaskDispatchHigh, events[0].Topic)
	assert.Equal(t, 1, executor.getCallCount())
	require.Eventually(t, d.IsIdle, 5*time.Second, 5*time.Millisecond)
}

func Test_watermillTaskDispatcher_RefusesDispatchAfterStop(t *testing.T) {
	t.Parallel()

	store := &orchestrator_domain.MockTaskStore{}
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), store)

	running := runDispatcher(t, d)
	require.Eventually(t, func() bool {
		return dispatcherPhaseOf(d) == dispatcherPhaseRunning
	}, 5*time.Second, 5*time.Millisecond)
	require.NoError(t, running.stop())

	err := d.Dispatch(t.Context(), newHighPriorityTask("late"))
	require.ErrorIs(t, err, orchestrator_domain.ErrDispatcherStopped)
	assert.Equal(t, int64(0), store.CreateTaskWithDedupCallCount.Load(),
		"a task that can never be published is not persisted")
	assert.Equal(t, int64(0), d.Stats().TasksDispatched)
	assert.True(t, d.IsIdle())
}

func Test_watermillTaskDispatcher_BacklogLimit(t *testing.T) {
	t.Parallel()

	store := &orchestrator_domain.MockTaskStore{}
	config := newLifecycleTestConfig()
	config.DispatchBacklogLimit = 1
	d := newWatermillTaskDispatcher(config, newMockEventBus(), store)

	require.NoError(t, d.Dispatch(t.Context(), newHighPriorityTask("first")))

	err := d.Dispatch(t.Context(), newHighPriorityTask("second"))
	require.ErrorIs(t, err, orchestrator_domain.ErrDispatchBacklogFull)
	assert.Equal(t, int64(1), store.CreateTaskWithDedupCallCount.Load(),
		"a task refused by the backlog limit is not persisted")
	assert.Equal(t, int64(1), d.Stats().TasksDispatched)
}

func Test_watermillTaskDispatcher_holdForPublish(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr  error
		name     string
		phase    dispatcherPhase
		mode     publishMode
		held     int
		wantHeld bool
		wantWake bool
	}{
		{name: "holds while not started", phase: dispatcherPhaseNotStarted, mode: publishImmediately, wantHeld: true},
		{name: "refuses when the backlog is full", phase: dispatcherPhaseNotStarted, held: 1, wantErr: orchestrator_domain.ErrDispatchBacklogFull},
		{name: "publishes directly while running", phase: dispatcherPhaseRunning, mode: publishImmediately, wantHeld: false},
		{name: "holds a deferred publish while running and wakes the publisher", phase: dispatcherPhaseRunning, mode: publishDeferred, wantHeld: true, wantWake: true},
		{name: "refuses a deferred publish when the backlog is full", phase: dispatcherPhaseRunning, mode: publishDeferred, held: 1, wantErr: orchestrator_domain.ErrDispatchBacklogFull},
		{name: "refuses once stopped", phase: dispatcherPhaseStopped, wantErr: orchestrator_domain.ErrDispatcherStopped},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := newLifecycleTestConfig()
			config.DispatchBacklogLimit = 1
			d := newWatermillTaskDispatcher(config, newMockEventBus(), nil)
			d.phase = tc.phase
			for range tc.held {
				d.backlog = append(d.backlog, backloggedTask{task: newHighPriorityTask("held")})
			}

			held, err := d.holdForPublish(newHighPriorityTask("task"), orchestrator_domain.TopicTaskDispatchHigh, orchestrator_domain.Event{}, tc.mode)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantHeld, held)

			select {
			case <-d.heldWake:
				assert.True(t, tc.wantWake, "the held-task publisher was woken unexpectedly")
			default:
				assert.False(t, tc.wantWake, "the held-task publisher was not woken")
			}
		})
	}
}

func Test_watermillTaskDispatcher_Start_SubscribeFailureFailsHeldTasks(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	eventBus.subscribeErr = errors.New("broker unavailable")
	store := newStatusRecordingStore(nil)
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), eventBus, store)

	require.NoError(t, d.Dispatch(t.Context(), newHighPriorityTask("held")))

	err := d.Start(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker unavailable")

	stats := d.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "the held task is accounted for as failed")
	assert.True(t, d.IsIdle(), "no task is left waiting on a dispatcher that never ran")
	assert.Equal(t, []orchestrator_domain.TaskStatus{orchestrator_domain.StatusFailed}, store.recordedStatuses())

	err = d.Dispatch(t.Context(), newHighPriorityTask("after"))
	require.ErrorIs(t, err, orchestrator_domain.ErrDispatcherStopped)
}

func Test_watermillTaskDispatcher_Start_HeldTaskPublishFailureIsCounted(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	eventBus.publishFunc = func(context.Context, string, orchestrator_domain.Event) error {
		return errors.New("bus rejected message")
	}
	store := newStatusRecordingStore(nil)
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), eventBus, store)

	require.NoError(t, d.Dispatch(t.Context(), newHighPriorityTask("held")))

	runDispatcher(t, d)

	require.Eventually(t, func() bool {
		return d.Stats().TasksFailed == 1
	}, 5*time.Second, 5*time.Millisecond)
	assert.True(t, d.IsIdle())
	assert.Equal(t, []orchestrator_domain.TaskStatus{orchestrator_domain.StatusFailed}, store.recordedStatuses())
}

func Test_watermillTaskDispatcher_DispatchIfRequired(t *testing.T) {
	t.Parallel()

	errCheck := errors.New("registry unavailable")

	testCases := []struct {
		required       orchestrator_domain.DispatchRequirement
		dedupErr       error
		updateErr      error
		wantErr        error
		name           string
		wantStatuses   []orchestrator_domain.TaskStatus
		wantPublished  int
		wantDispatched int64
		wantCreates    int64
		wantPending    int
		wantChecked    bool
	}{
		{
			name:        "missing requirement is rejected before anything is persisted",
			required:    nil,
			wantErr:     errNilDispatchRequirement,
			wantCreates: 0,
		},
		{
			name:           "work still required is published",
			required:       func(context.Context) (bool, error) { return true, nil },
			wantPublished:  1,
			wantDispatched: 1,
			wantCreates:    1,
			wantChecked:    true,
		},
		{
			name:         "satisfied work is released without publishing",
			required:     func(context.Context) (bool, error) { return false, nil },
			wantErr:      orchestrator_domain.ErrTaskNotRequired,
			wantStatuses: []orchestrator_domain.TaskStatus{orchestrator_domain.StatusComplete},
			wantCreates:  1,
			wantChecked:  true,
		},
		{
			name:           "failed requirement check publishes anyway",
			required:       func(context.Context) (bool, error) { return false, errCheck },
			wantPublished:  1,
			wantDispatched: 1,
			wantCreates:    1,
			wantChecked:    true,
		},
		{
			name:           "failed release publishes anyway so completion frees the key",
			required:       func(context.Context) (bool, error) { return false, nil },
			updateErr:      errors.New("store unavailable"),
			wantStatuses:   []orchestrator_domain.TaskStatus{orchestrator_domain.StatusComplete},
			wantPublished:  1,
			wantDispatched: 1,
			wantCreates:    1,
			wantChecked:    true,
		},
		{
			name:        "duplicate claim becomes the pending rerun without checking the requirement",
			required:    func(context.Context) (bool, error) { return true, nil },
			dedupErr:    orchestrator_domain.ErrDuplicateTask,
			wantErr:     orchestrator_domain.ErrDuplicateTask,
			wantCreates: 2,
			wantPending: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			eventBus := newMockEventBus()
			store := newStatusRecordingStore(tc.updateErr)
			if tc.dedupErr != nil {
				store.CreateTaskWithDedupFunc = func(context.Context, *orchestrator_domain.Task) error {
					return tc.dedupErr
				}
			}
			d := newWatermillTaskDispatcher(newLifecycleTestConfig(), eventBus, store)
			startPublishing(t, d)

			var checked atomic.Bool
			var required orchestrator_domain.DispatchRequirement
			if tc.required != nil {
				required = func(ctx context.Context) (bool, error) {
					checked.Store(true)
					return tc.required(ctx)
				}
			}

			err := d.DispatchIfRequired(t.Context(), newHighPriorityTask("task"), required)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantChecked, checked.Load())
			assert.Len(t, eventBus.getPublishedEvents(), tc.wantPublished)
			assert.Equal(t, tc.wantDispatched, d.Stats().TasksDispatched)
			assert.Equal(t, tc.wantCreates, store.CreateTaskWithDedupCallCount.Load())
			assert.Equal(t, tc.wantStatuses, store.recordedStatuses())
			assert.Equal(t, tc.wantPending, d.reruns.size())
		})
	}
}

func Test_watermillTaskDispatcher_RecoveredTasksAreDispatchedAgain(t *testing.T) {
	t.Parallel()

	mockClock := clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	config := newLifecycleTestConfig()
	config.RecoveryInterval = 30 * time.Second
	config.DefaultMaxRetries = 3

	recovered := &orchestrator_domain.Task{
		ID:               "stale-task",
		WorkflowID:       "wf-stale",
		Executor:         "test-executor",
		Status:           orchestrator_domain.StatusRetrying,
		Attempt:          1,
		DeduplicationKey: "stale-key",
		Payload:          map[string]any{"input": "data"},
		Config:           orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityNormal, MaxRetries: 3},
	}

	var claimed atomic.Bool
	store := newStatusRecordingStore(nil)
	store.ClaimStaleTasksForRecoveryFunc = func(context.Context, string, time.Duration, time.Duration, int) ([]orchestrator_domain.RecoveryClaimedTask, error) {
		if claimed.Swap(true) {
			return nil, nil
		}
		return []orchestrator_domain.RecoveryClaimedTask{{ID: recovered.ID, WorkflowID: recovered.WorkflowID, Attempt: 1}}, nil
	}
	store.RecoverClaimedTasksFunc = func(context.Context, string, int, string) (int, error) {
		return 1, nil
	}
	store.GetTasksByIDFunc = func(context.Context, []string) ([]*orchestrator_domain.Task, error) {
		return []*orchestrator_domain.Task{recovered}, nil
	}

	d := newWatermillTaskDispatcher(config, newMockEventBus(), store, withWatermillClock(mockClock))
	executor := newMockExecutor()
	d.RegisterExecutor(t.Context(), "test-executor", executor)

	baseline := mockClock.TimerCount()
	runDispatcher(t, d)
	require.True(t, mockClock.AwaitTimerSetup(baseline, 5*time.Second), "recovery loop should arm its ticker")
	assert.Equal(t, 0, executor.getCallCount(), "nothing runs before a recovery sweep")

	mockClock.Advance(config.RecoveryInterval)

	require.Eventually(t, func() bool {
		return executor.getCallCount() == 1 && d.IsIdle()
	}, 5*time.Second, 5*time.Millisecond, "the recovered task must be dispatched and run again")

	stats := d.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched)
	assert.Equal(t, int64(1), stats.TasksCompleted)
	assert.Equal(t, int64(0), store.CreateTaskWithDedupCallCount.Load(),
		"a recovered task updates its existing record instead of being inserted again")
	assert.Contains(t, store.recordedStatuses(), orchestrator_domain.StatusComplete)
}
