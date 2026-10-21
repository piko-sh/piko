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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal/otter"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	clockpkg "piko.sh/piko/wdk/clock"
)

func clockedDispatcherTasks(t *testing.T, clk clockpkg.Clock) orchestrator_domain.TaskStore {
	t.Helper()
	store, err := otter.NewOtterDAL(otter.Config{Capacity: 1000}, otter.WithClock(clk))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func Test_taskPayload_JSONWireRoundTrip(t *testing.T) {
	t.Parallel()

	d := newWatermillTaskDispatcher(orchestrator_domain.DefaultDispatcherConfig(), newMockEventBus(), nil)
	original := &orchestrator_domain.Task{
		ID:         "task-wire",
		WorkflowID: "wf-wire",
		Executor:   "test-executor",
		Status:     orchestrator_domain.StatusPending,
		Attempt:    2,
		Payload:    map[string]any{"input": "data"},
		Config: orchestrator_domain.TaskConfig{
			Priority:   orchestrator_domain.PriorityHigh,
			Timeout:    7 * time.Second,
			MaxRetries: 4,
		},
	}

	raw, err := json.Marshal(orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("task.dispatch"),
		Payload: refPayload(orchestrator_domain.RefOf(original)),
	})
	require.NoError(t, err)
	var decoded orchestrator_domain.Event
	require.NoError(t, json.Unmarshal(raw, &decoded))

	restored, err := d.taskFromPayload(decoded.Payload)
	require.NoError(t, err)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Attempt, restored.Attempt, "the attempt survives the JSON wire format")
	assert.Equal(t, original.Config.Priority, restored.Config.Priority, "the priority survives the JSON wire format")
	assert.Empty(t, restored.Payload, "the payload stays in the store; the claim reads it")
}

func Test_handleTaskEvent_PrematureDeliveryIsRearmed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := clockpkg.NewMockClock(now)
	store := clockedDispatcherTasks(t, clk)
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), store, withWatermillClock(clk))
	executor := newMockExecutor()
	d.RegisterExecutor(t.Context(), "test-executor", executor)

	task := &orchestrator_domain.Task{
		ID:         "task-premature",
		WorkflowID: "wf-premature",
		Executor:   "test-executor",
		Status:     orchestrator_domain.StatusRetrying,
		Attempt:    1,
		ExecuteAt:  now.Add(time.Hour),
		Payload:    map[string]any{},
	}
	require.NoError(t, store.CreateTask(t.Context(), task))

	d.TasksDispatched.Add(1)
	d.pendingTasks.Add(1)
	err := d.handleTaskEvent(t.Context(), orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("task.dispatch"),
		Payload: refPayload(orchestrator_domain.RefOf(task)),
	}, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, executor.getCallCount(), "a premature delivery does not run")
	assert.Equal(t, int64(0), d.TasksDispatched.Load(), "the re-dispatch counts the task again")
	assert.Equal(t, int64(0), d.pendingTasks.Load())
	assert.Equal(t, 1, d.DelayedPublisher.PendingCount(), "the task is re-armed for its due time")
	assert.False(t, d.IsIdle(), "a re-armed task keeps the dispatcher busy")

	rows, err := store.GetTasksByID(t.Context(), []string{task.ID})
	require.NoError(t, err)
	assert.Equal(t, orchestrator_domain.StatusRetrying, rows[0].Status)
	assert.Equal(t, 1, rows[0].Attempt)
}

func Test_handleTaskEvent_ObsoleteDeliveryIsUncounted(t *testing.T) {
	t.Parallel()

	clk := clockpkg.NewMockClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	store := clockedDispatcherTasks(t, clk)
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), store, withWatermillClock(clk))
	executor := newMockExecutor()
	d.RegisterExecutor(t.Context(), "test-executor", executor)

	task := &orchestrator_domain.Task{
		ID:         "task-obsolete",
		WorkflowID: "wf-obsolete",
		Executor:   "test-executor",
		Status:     orchestrator_domain.StatusProcessing,
		Attempt:    1,
		Payload:    map[string]any{},
	}
	require.NoError(t, store.CreateTask(t.Context(), task))

	d.TasksDispatched.Add(1)
	d.pendingTasks.Add(1)
	err := d.handleTaskEvent(t.Context(), orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("task.dispatch"),
		Payload: refPayload(orchestrator_domain.RefOf(task)),
	}, 0)

	require.NoError(t, err)
	assert.Equal(t, 0, executor.getCallCount())
	assert.Equal(t, int64(0), d.TasksDispatched.Load())
	assert.Equal(t, 0, d.DelayedPublisher.PendingCount(), "an obsolete delivery is not re-armed")
	assert.True(t, d.IsIdle())
}

func Test_watermillTaskDispatcher_NewTaskUsesDispatcherDefaults(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := clockpkg.NewMockClock(now)
	config := newLifecycleTestConfig()
	config.DefaultMaxRetries = 1
	config.DefaultTimeout = 7 * time.Second
	store := clockedDispatcherTasks(t, clk)
	d := newWatermillTaskDispatcher(config, newMockEventBus(), store, withWatermillClock(clk))
	executor := newMockExecutor()
	executor.executeFunc = func(context.Context, map[string]any) (map[string]any, error) {
		return nil, errors.New("always fails")
	}
	d.RegisterExecutor(t.Context(), "test-executor", executor)
	runDispatcher(t, d)

	task := orchestrator_domain.NewTask("test-executor", map[string]any{"input": "data"})
	task.DeduplicationKey = "defaults-key"
	require.NoError(t, d.Dispatch(t.Context(), task))

	require.Eventually(t, func() bool {
		return executor.getCallCount() == 1 && d.IsIdle()
	}, 5*time.Second, 5*time.Millisecond, "a task from NewTask runs under the mock clock and settles")

	stats := d.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one attempt exhausts DefaultMaxRetries=1")
	assert.Equal(t, int64(0), stats.TasksRetried)

	rows, err := store.GetTasksByID(t.Context(), []string{task.ID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, orchestrator_domain.StatusFailed, rows[0].Status)
	assert.Equal(t, 1, rows[0].Config.MaxRetries)
	assert.Equal(t, 7*time.Second, rows[0].Config.Timeout)
	assert.True(t, rows[0].CreatedAt.Equal(now), "admission times come from the dispatcher clock")

	next := orchestrator_domain.NewTask("test-executor", map[string]any{"input": "data"})
	next.DeduplicationKey = "defaults-key"
	require.NoError(t, d.Dispatch(t.Context(), next), "the failed task released its deduplication key")
}

func Test_watermillTaskDispatcher_PrematureRetryRunsOnceDue(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := clockpkg.NewMockClock(now)
	store := clockedDispatcherTasks(t, clk)
	d := newWatermillTaskDispatcher(newLifecycleTestConfig(), newMockEventBus(), store, withWatermillClock(clk))
	executor := newMockExecutor()
	d.RegisterExecutor(t.Context(), "test-executor", executor)
	runDispatcher(t, d)

	require.NoError(t, store.CreateTask(t.Context(), &orchestrator_domain.Task{
		ID:         "task-early",
		WorkflowID: "wf-early",
		Executor:   "test-executor",
		Status:     orchestrator_domain.StatusRetrying,
		Attempt:    1,
		ExecuteAt:  now.Add(time.Hour),
		Payload:    map[string]any{},
	}))
	rows, err := store.GetTasksByID(t.Context(), []string{"task-early"})
	require.NoError(t, err)

	baseline := clk.TimerCount()
	require.NoError(t, d.Dispatch(t.Context(), rows[0]))
	require.True(t, clk.AwaitTimerSetup(baseline, 5*time.Second), "the premature delivery is re-armed")
	assert.Equal(t, 0, executor.getCallCount(), "nothing runs before the due time")
	assert.False(t, d.IsIdle())

	clk.Advance(time.Hour)

	require.Eventually(t, func() bool {
		return executor.getCallCount() == 1 && d.IsIdle()
	}, 5*time.Second, 5*time.Millisecond, "the re-armed retry runs once it is due")
	stats := d.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched)
	assert.Equal(t, int64(1), stats.TasksCompleted)
}
