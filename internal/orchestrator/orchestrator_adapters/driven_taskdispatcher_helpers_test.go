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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func Test_topicForPriority(t *testing.T) {
	t.Parallel()

	d := &watermillTaskDispatcher{}

	tests := []struct {
		name     string
		expected string
		priority orchestrator_domain.TaskPriority
	}{
		{name: "high priority", expected: orchestrator_domain.TopicTaskDispatchHigh, priority: orchestrator_domain.PriorityHigh},
		{name: "normal priority", expected: orchestrator_domain.TopicTaskDispatchNormal, priority: orchestrator_domain.PriorityNormal},
		{name: "low priority", expected: orchestrator_domain.TopicTaskDispatchLow, priority: orchestrator_domain.PriorityLow},
		{name: "unknown defaults to normal", expected: orchestrator_domain.TopicTaskDispatchNormal, priority: orchestrator_domain.TaskPriority(99)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, d.topicForPriority(tc.priority))
		})
	}
}

func Test_refPayload(t *testing.T) {
	t.Parallel()

	task := &orchestrator_domain.Task{
		ID:       "task-123",
		Executor: "artefact.compiler",
		Payload:  map[string]any{"key": "a large payload that stays in the store"},
		Config:   orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityHigh, MaxRetries: 3},
		Attempt:  2,
	}

	assert.Equal(t, map[string]any{
		"id":       "task-123",
		"attempt":  2,
		"priority": int(orchestrator_domain.PriorityHigh),
	}, refPayload(orchestrator_domain.RefOf(task)), "only what a claim needs travels on the bus")
}

func Test_taskFromPayload_ValidPayload(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	now := time.Now().UTC()

	payload := map[string]any{
		"id":                 "task-123",
		"workflowID":         "wf-456",
		"executor":           "test-executor",
		"payload":            map[string]any{"input": "data"},
		"result":             map[string]any{"output": "result"},
		"lastError":          "err message",
		"deduplicationKey":   "dedup-1",
		"status":             orchestrator_domain.StatusPending,
		"attempt":            3,
		"config":             orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityHigh, MaxRetries: 5},
		"executeAt":          now,
		"scheduledExecuteAt": now.Add(5 * time.Minute),
		"createdAt":          now.Add(-1 * time.Hour),
		"updatedAt":          now,
	}

	task, err := d.taskFromPayload(payload)
	require.NoError(t, err)

	assert.Equal(t, "task-123", task.ID)
	assert.Equal(t, "wf-456", task.WorkflowID)
	assert.Equal(t, "test-executor", task.Executor)
	assert.Equal(t, map[string]any{"input": "data"}, task.Payload)
	assert.Equal(t, map[string]any{"output": "result"}, task.Result)
	assert.Equal(t, "err message", task.LastError)
	assert.Equal(t, "dedup-1", task.DeduplicationKey)
	assert.Equal(t, orchestrator_domain.StatusPending, task.Status)
	assert.Equal(t, 3, task.Attempt)
	assert.Equal(t, orchestrator_domain.PriorityHigh, task.Config.Priority)
	assert.Equal(t, 5, task.Config.MaxRetries)
	assert.Equal(t, now, task.ExecuteAt)
	assert.Equal(t, now.Add(5*time.Minute), task.ScheduledExecuteAt)
	assert.Equal(t, now.Add(-1*time.Hour), task.CreatedAt)
	assert.Equal(t, now, task.UpdatedAt)
}

func Test_taskFromPayload_MissingID(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	payload := map[string]any{
		"workflowID": "wf-456",
		"executor":   "test-executor",
	}

	_, err := d.taskFromPayload(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid task ID")
}

func Test_taskFromPayload_InvalidIDType(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	payload := map[string]any{
		"id": 42,
	}

	_, err := d.taskFromPayload(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing or invalid task ID")
}

func Test_taskFromPayload_MinimalPayload(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	payload := map[string]any{
		"id": "task-only",
	}

	task, err := d.taskFromPayload(payload)
	require.NoError(t, err)

	assert.Equal(t, "task-only", task.ID)
	assert.Equal(t, "", task.WorkflowID)
	assert.Equal(t, "", task.Executor)
	assert.Nil(t, task.Payload)
	assert.Nil(t, task.Result)
	assert.Equal(t, "", task.LastError)
	assert.Equal(t, "", task.DeduplicationKey)
	assert.Equal(t, orchestrator_domain.TaskStatus(""), task.Status)
	assert.Equal(t, 0, task.Attempt)
	assert.True(t, task.ExecuteAt.IsZero())
}

func Test_payloadString(t *testing.T) {
	t.Parallel()

	t.Run("existing string key", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": "value"}
		assert.Equal(t, "value", payloadString(payload, "key"))
	})

	t.Run("missing key returns empty", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{}
		assert.Equal(t, "", payloadString(payload, "missing"))
	})

	t.Run("non-string value returns empty", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": 42}
		assert.Equal(t, "", payloadString(payload, "key"))
	})

	t.Run("nil value returns empty", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": nil}
		assert.Equal(t, "", payloadString(payload, "key"))
	})
}

func Test_payloadMap(t *testing.T) {
	t.Parallel()

	t.Run("existing map key", func(t *testing.T) {
		t.Parallel()
		inner := map[string]any{"nested": "data"}
		payload := map[string]any{"key": inner}
		assert.Equal(t, inner, payloadMap(payload, "key"))
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{}
		assert.Nil(t, payloadMap(payload, "missing"))
	})

	t.Run("non-map value returns nil", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": "not-a-map"}
		assert.Nil(t, payloadMap(payload, "key"))
	})

	t.Run("nil value returns nil", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": nil}
		assert.Nil(t, payloadMap(payload, "key"))
	})
}

func Test_payloadTime(t *testing.T) {
	t.Parallel()

	t.Run("existing time key", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		payload := map[string]any{"key": now}
		assert.Equal(t, now, payloadTime(payload, "key"))
	})

	t.Run("missing key returns zero time", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{}
		assert.True(t, payloadTime(payload, "missing").IsZero())
	})

	t.Run("non-time value returns zero time", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": "not-a-time"}
		assert.True(t, payloadTime(payload, "key").IsZero())
	})

	t.Run("nil value returns zero time", func(t *testing.T) {
		t.Parallel()
		payload := map[string]any{"key": nil}
		assert.True(t, payloadTime(payload, "key").IsZero())
	})
}

func Test_parseTaskStatusField(t *testing.T) {
	t.Parallel()

	t.Run("TaskStatus type", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"status": orchestrator_domain.StatusComplete}
		parseTaskStatusField(payload, task)
		assert.Equal(t, orchestrator_domain.StatusComplete, task.Status)
	})

	t.Run("string type", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"status": "FAILED"}
		parseTaskStatusField(payload, task)
		assert.Equal(t, orchestrator_domain.TaskStatus("FAILED"), task.Status)
	})

	t.Run("missing status leaves default", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{}
		parseTaskStatusField(payload, task)
		assert.Equal(t, orchestrator_domain.TaskStatus(""), task.Status)
	})

	t.Run("non-string type leaves default", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"status": 42}
		parseTaskStatusField(payload, task)
		assert.Equal(t, orchestrator_domain.TaskStatus(""), task.Status)
	})
}

func Test_parseTaskAttemptField(t *testing.T) {
	t.Parallel()

	t.Run("int type", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"attempt": 5}
		parseTaskAttemptField(payload, task)
		assert.Equal(t, 5, task.Attempt)
	})

	t.Run("float64 type", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"attempt": float64(3)}
		parseTaskAttemptField(payload, task)
		assert.Equal(t, 3, task.Attempt)
	})

	t.Run("missing attempt leaves zero", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{}
		parseTaskAttemptField(payload, task)
		assert.Equal(t, 0, task.Attempt)
	})

	t.Run("string type leaves zero", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"attempt": "five"}
		parseTaskAttemptField(payload, task)
		assert.Equal(t, 0, task.Attempt)
	})
}

func Test_parseTaskConfigField(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	t.Run("TaskConfig type", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		tc := orchestrator_domain.TaskConfig{
			Priority:   orchestrator_domain.PriorityHigh,
			MaxRetries: 5,
			Timeout:    30 * time.Second,
		}
		payload := map[string]any{"config": tc}
		d.parseTaskConfigField(payload, task)
		assert.Equal(t, tc, task.Config)
	})

	t.Run("map type with float64 fields", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{
			"config": map[string]any{
				"priority":    float64(orchestrator_domain.PriorityHigh),
				"timeout":     float64(30 * time.Second),
				"max_retries": float64(5),
			},
		}
		d.parseTaskConfigField(payload, task)
		assert.Equal(t, orchestrator_domain.PriorityHigh, task.Config.Priority)
		assert.Equal(t, time.Duration(int64(float64(30*time.Second))), task.Config.Timeout)
		assert.Equal(t, 5, task.Config.MaxRetries)
	})

	t.Run("missing config leaves default", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{}
		d.parseTaskConfigField(payload, task)
		assert.Equal(t, orchestrator_domain.TaskConfig{}, task.Config)
	})

	t.Run("invalid config type leaves default", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{}
		payload := map[string]any{"config": "not-a-config"}
		d.parseTaskConfigField(payload, task)
		assert.Equal(t, orchestrator_domain.TaskConfig{}, task.Config)
	})
}

func Test_parseTaskConfig(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	t.Run("all fields present", func(t *testing.T) {
		t.Parallel()
		configMap := map[string]any{
			"priority":    float64(2),
			"timeout":     float64(1000000000),
			"max_retries": float64(3),
		}
		result := d.parseTaskConfig(configMap)
		assert.Equal(t, orchestrator_domain.TaskPriority(2), result.Priority)
		assert.Equal(t, time.Duration(1000000000), result.Timeout)
		assert.Equal(t, 3, result.MaxRetries)
	})

	t.Run("empty map", func(t *testing.T) {
		t.Parallel()
		result := d.parseTaskConfig(map[string]any{})
		assert.Equal(t, orchestrator_domain.TaskConfig{}, result)
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		configMap := map[string]any{
			"priority": float64(1),
		}
		result := d.parseTaskConfig(configMap)
		assert.Equal(t, orchestrator_domain.TaskPriority(1), result.Priority)
		assert.Equal(t, time.Duration(0), result.Timeout)
		assert.Equal(t, 0, result.MaxRetries)
	})

	t.Run("wrong types ignored", func(t *testing.T) {
		t.Parallel()
		configMap := map[string]any{
			"priority":    "high",
			"timeout":     "1s",
			"max_retries": "3",
		}
		result := d.parseTaskConfig(configMap)
		assert.Equal(t, orchestrator_domain.TaskConfig{}, result)
	})
}

func Test_refPayload_RoundTrip(t *testing.T) {
	t.Parallel()

	d := newWatermillTaskDispatcher(orchestrator_domain.DefaultDispatcherConfig(), newMockEventBus(), nil)
	original := &orchestrator_domain.Task{
		ID:      "round-trip-task",
		Config:  orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityHigh},
		Attempt: 1,
	}

	restored, err := d.taskFromPayload(refPayload(orchestrator_domain.RefOf(original)))
	require.NoError(t, err)
	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.Attempt, restored.Attempt)
	assert.Equal(t, original.Config.Priority, restored.Config.Priority)

	decoded, err := d.taskFromPayload(map[string]any{"id": "json", "attempt": float64(3), "priority": float64(2)})
	require.NoError(t, err, "a reference that crossed a JSON bus decodes")
	assert.Equal(t, 3, decoded.Attempt)
	assert.Equal(t, orchestrator_domain.TaskPriority(2), decoded.Config.Priority)
}

func Test_handleTaskEvent_MalformedPayload(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("test"),
		Payload: map[string]any{"not-id": "value"},
	}

	ctx := t.Context()
	err := d.handleTaskEvent(ctx, event, 0)

	assert.NoError(t, err, "malformed tasks should be acknowledged to prevent infinite redelivery")
	assert.Equal(t, int64(1), d.Stats().TasksFailed,
		"a task message that can never run is counted as failed so idle detection does not wait for it")
}

func Test_persistOrUpdateTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		status        orchestrator_domain.TaskStatus
		wantStatus    orchestrator_domain.TaskStatus
		attempt       int
		wantCreateHit int64
		persistTwice  bool
		wantCreated   bool
	}{
		{
			name:          "retry task preserves its committed state without a new write",
			status:        orchestrator_domain.StatusRetrying,
			attempt:       1,
			wantCreated:   false,
			wantStatus:    orchestrator_domain.StatusRetrying,
			wantCreateHit: 0,
		},
		{
			name:          "new task creates its record",
			status:        orchestrator_domain.StatusPending,
			wantCreated:   true,
			wantStatus:    orchestrator_domain.StatusPending,
			wantCreateHit: 1,
		},
		{
			name:          "task whose record already exists is not created again",
			status:        orchestrator_domain.StatusPending,
			persistTwice:  true,
			wantCreated:   false,
			wantStatus:    orchestrator_domain.StatusPending,
			wantCreateHit: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &orchestrator_domain.MockTaskStore{}
			config := orchestrator_domain.DefaultDispatcherConfig()
			d := newWatermillTaskDispatcher(config, newMockEventBus(), store)

			task := &orchestrator_domain.Task{
				ID:               "task-1",
				WorkflowID:       "wf-1",
				Executor:         "test-executor",
				Status:           tc.status,
				Attempt:          tc.attempt,
				DeduplicationKey: "dedup-key",
			}

			if tc.persistTwice {
				_, err := d.persistOrUpdateTask(t.Context(), task)
				require.NoError(t, err)
			}

			created, err := d.persistOrUpdateTask(t.Context(), task)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCreated, created)
			assert.Equal(t, tc.wantStatus, task.Status)
			assert.Equal(t, tc.wantCreateHit, store.CreateTaskWithDedupCallCount.Load())
		})
	}

	t.Run("duplicate active task is reported", func(t *testing.T) {
		t.Parallel()

		store := &orchestrator_domain.MockTaskStore{
			CreateTaskWithDedupFunc: func(_ context.Context, _ *orchestrator_domain.Task) error {
				return orchestrator_domain.ErrDuplicateTask
			},
		}
		config := orchestrator_domain.DefaultDispatcherConfig()
		d := newWatermillTaskDispatcher(config, newMockEventBus(), store)

		task := &orchestrator_domain.Task{
			ID:               "new-task",
			WorkflowID:       "wf-2",
			Executor:         "test-executor",
			Status:           orchestrator_domain.StatusPending,
			DeduplicationKey: "dedup-key",
		}
		created, err := d.persistOrUpdateTask(t.Context(), task)
		require.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask)
		assert.False(t, created)
	})
}

func Test_watermillTaskDispatcher_RegisterExecutor(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	executor := newMockExecutor()
	d.RegisterExecutor(context.Background(), "test-exec", executor)

	found, err := d.GetExecutor("test-exec")
	require.NoError(t, err)
	assert.NotNil(t, found)

	_, err = d.GetExecutor("nonexistent")
	assert.Error(t, err)
}

func Test_watermillTaskDispatcher_StartAlreadyStarted(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.RecoveryInterval = 0

	delayedPub := newTrackingDelayedPublisher()
	d := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	ctx, cancel := context.WithCancelCause(t.Context())

	go func() {
		_ = d.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	err := d.Start(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	cancel(fmt.Errorf("test: cleanup"))
	time.Sleep(50 * time.Millisecond)
}

func Test_watermillTaskDispatcher_IsIdle_WithDelayedPublisher(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()

	delayedPub := newTrackingDelayedPublisher()
	d := newWatermillTaskDispatcher(config, eventBus, nil,
		withWatermillDelayedPublisher(delayedPub))

	assert.True(t, d.IsIdle())

	_ = delayedPub.Schedule(context.Background(), &orchestrator_domain.Task{ID: "t1"})

	assert.False(t, d.IsIdle())
}

func Test_watermillTaskDispatcher_DispatchDelayed_ValidationFails(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	delayedPub := newTrackingDelayedPublisher()

	d := newWatermillTaskDispatcher(config, eventBus, store,
		withWatermillDelayedPublisher(delayedPub))
	ctx := t.Context()

	task := &orchestrator_domain.Task{
		ID:       "",
		Executor: "test-executor",
	}

	err := d.DispatchDelayed(ctx, task, time.Now().Add(10*time.Minute))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validating")
}

func Test_watermillTaskDispatcher_DispatchDelayed_DedupBlocked(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{
		CreateTaskWithDedupFunc: func(_ context.Context, _ *orchestrator_domain.Task) error {
			return orchestrator_domain.ErrDuplicateTask
		},
	}
	config := orchestrator_domain.DefaultDispatcherConfig()
	delayedPub := newTrackingDelayedPublisher()

	d := newWatermillTaskDispatcher(config, eventBus, store,
		withWatermillDelayedPublisher(delayedPub))
	ctx := t.Context()

	task := &orchestrator_domain.Task{
		ID:               "task-1",
		WorkflowID:       "wf-1",
		Executor:         "test-executor",
		DeduplicationKey: "dedup-key",
	}

	err := d.DispatchDelayed(ctx, task, time.Now().Add(10*time.Minute))
	require.Error(t, err)
	assert.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask)
}

func Test_watermillTaskDispatcher_DispatchDelayed_NoDelayedPublisher(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()

	d := newWatermillTaskDispatcher(config, eventBus, store)
	ctx := t.Context()

	task := &orchestrator_domain.Task{
		ID:               "task-1",
		WorkflowID:       "wf-1",
		Executor:         "test-executor",
		DeduplicationKey: "dedup-key",
	}

	err := d.DispatchDelayed(ctx, task, time.Now().Add(10*time.Minute))
	require.NoError(t, err)
	assert.Equal(t, orchestrator_domain.StatusScheduled, task.Status)
}

func Test_watermillTaskDispatcher_Dispatch_TransientPublishFailureHoldsTask(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	var refuse atomic.Bool
	refuse.Store(true)
	eventBus.publishFunc = func(_ context.Context, _ string, _ orchestrator_domain.Event) error {
		if refuse.Load() {
			return assert.AnError
		}
		return nil
	}
	store := &orchestrator_domain.MockTaskStore{}
	d := newWatermillTaskDispatcher(orchestrator_domain.DefaultDispatcherConfig(), eventBus, store)
	startPublishing(t, d)

	task := &orchestrator_domain.Task{ID: "task-1", WorkflowID: "wf-1", Executor: "test-executor"}
	require.NoError(t, d.Dispatch(t.Context(), task), "a bus that cannot take the message yet is not a dispatch failure")
	assert.Equal(t, orchestrator_domain.StatusPending, task.Status, "a held task is not failed")
	assert.Zero(t, store.UpdateTaskCallCount.Load(), "a held task's record is left pending")

	stats := d.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched)
	assert.Equal(t, 1, stats.HeldTasks)
	assert.False(t, d.IsIdle())

	d.publishHeld(t.Context())
	assert.Equal(t, 1, d.Stats().HeldTasks, "the task stays held while the bus refuses it")

	refuse.Store(false)
	d.publishHeld(t.Context())
	assert.Zero(t, d.Stats().HeldTasks, "the task is published once the bus takes it")
	assert.Equal(t, int64(1), d.Stats().TasksDispatched, "publishing a held task does not count it twice")
}

func Test_watermillTaskDispatcher_Dispatch_PermanentPublishFailureAbandonsTask(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	eventBus.publishFunc = func(_ context.Context, topic string, _ orchestrator_domain.Event) error {
		if topic == orchestrator_domain.TopicTaskCompleted {
			return nil
		}
		return ErrEventPayloadTooLarge
	}
	store := &orchestrator_domain.MockTaskStore{}
	d := newWatermillTaskDispatcher(orchestrator_domain.DefaultDispatcherConfig(), eventBus, store)
	startPublishing(t, d)

	task := &orchestrator_domain.Task{ID: "task-1", WorkflowID: "wf-1", Executor: "test-executor"}
	err := d.Dispatch(t.Context(), task)
	require.ErrorIs(t, err, ErrEventPayloadTooLarge)
	assert.Equal(t, orchestrator_domain.StatusFailed, task.Status,
		"a task whose message can never be published is failed so its key is freed")
	assert.Equal(t, int64(1), store.UpdateTaskCallCount.Load())

	stats := d.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched, "the abandoned task is counted as dispatched")
	assert.Equal(t, int64(1), stats.TasksFailed, "and as failed, so completion checks settle it")
	assert.True(t, d.IsIdle())

	completions := 0
	for _, published := range eventBus.getPublishedEvents() {
		if published.Topic == orchestrator_domain.TopicTaskCompleted {
			completions++
		}
	}
	assert.Equal(t, 1, completions, "anyone waiting for the task hears that it failed")
}

func Test_watermillTaskDispatcher_HeldTasksWaitForCapacity(t *testing.T) {
	t.Parallel()

	bus := &capacityEventBus{mockEventBus: newMockEventBus(), signal: make(chan struct{}, 1)}
	d := newWatermillTaskDispatcher(orchestrator_domain.DefaultDispatcherConfig(), bus, &orchestrator_domain.MockTaskStore{})
	require.NotNil(t, d.capacity, "a bus that bounds its messages reports its capacity")
	require.True(t, d.holdForCapacity(orchestrator_domain.TaskRef{ID: "early"}) == false,
		"nothing is held for capacity before the dispatcher runs")
	startPublishing(t, d)

	for _, id := range []string{"a", "b", "c"} {
		require.True(t, d.holdForCapacity(orchestrator_domain.TaskRef{ID: id}))
	}

	bus.free.Store(2)
	d.publishHeld(t.Context())
	assert.Len(t, bus.getPublishedEvents(), 2, "only as many messages as the bus has room for are published")
	assert.Equal(t, 1, d.heldCount())

	bus.free.Store(5)
	d.publishHeld(t.Context())
	assert.Len(t, bus.getPublishedEvents(), 3)
	assert.Zero(t, d.heldCount())
}

func Test_watermillTaskDispatcher_ShutdownLeavesHeldTasksPending(t *testing.T) {
	t.Parallel()

	store := &orchestrator_domain.MockTaskStore{}
	d := newWatermillTaskDispatcher(orchestrator_domain.DefaultDispatcherConfig(), newMockEventBus(), store)
	task := &orchestrator_domain.Task{ID: "task-1", WorkflowID: "wf-1", Executor: "test-executor"}
	require.NoError(t, d.Dispatch(t.Context(), task), "a dispatch before Start is held")
	require.Equal(t, 1, d.Stats().HeldTasks)

	d.dropHeld(t.Context())
	assert.Equal(t, orchestrator_domain.StatusPending, task.Status, "a held task is left for the next sweep, not failed")
	assert.Zero(t, store.UpdateTaskCallCount.Load())
	assert.Zero(t, d.Stats().TasksDispatched)
	assert.ErrorIs(t, d.Dispatch(t.Context(), &orchestrator_domain.Task{ID: "late", WorkflowID: "late", Executor: "x"}),
		orchestrator_domain.ErrDispatcherStopped)
}

func Test_classifyPublishError(t *testing.T) {
	t.Parallel()

	assert.Equal(t, publishTransient, classifyPublishError(assert.AnError))
	assert.Equal(t, publishPermanent, classifyPublishError(ErrEventPayloadTooLarge))
	assert.Equal(t, publishPermanent, classifyPublishError(permanentError{}))
	assert.Equal(t, publishUnavailable, classifyPublishError(orchestrator_domain.ErrServiceClosed))
}

type permanentError struct{}

func (permanentError) Error() string   { return "permanent" }
func (permanentError) Permanent() bool { return true }

type capacityEventBus struct {
	*mockEventBus
	signal chan struct{}
	free   atomic.Int64
}

func (b *capacityEventBus) CapacitySignal() <-chan struct{} { return b.signal }

func (b *capacityEventBus) FreeCapacity() int { return int(b.free.Load()) }

func Test_handleTaskEvent_ValidTask(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := storedDispatcherTasks(t)
	config := orchestrator_domain.DefaultDispatcherConfig()

	d := newWatermillTaskDispatcher(config, eventBus, store)
	executor := newMockExecutor()
	d.RegisterExecutor(context.Background(), "test-executor", executor)

	task := &orchestrator_domain.Task{
		ID:         "task-valid",
		WorkflowID: "wf-1",
		Executor:   "test-executor",
		Payload:    map[string]any{"input": "data"},
	}
	task.Status = orchestrator_domain.StatusPending
	require.NoError(t, store.CreateTask(t.Context(), task))
	payload := refPayload(orchestrator_domain.RefOf(task))

	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("task.dispatch"),
		Payload: payload,
	}

	ctx := t.Context()
	err := d.handleTaskEvent(ctx, event, 0)
	assert.NoError(t, err)
	assert.Equal(t, 1, executor.getCallCount())
}

func Test_watermillTaskDispatcher_Dispatch_RetryPath(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()

	d := newWatermillTaskDispatcher(config, eventBus, store)
	startPublishing(t, d)
	ctx := t.Context()

	task := &orchestrator_domain.Task{
		ID:         "task-retry",
		WorkflowID: "wf-1",
		Executor:   "test-executor",
		Attempt:    2,
		Status:     orchestrator_domain.StatusRetrying,
	}

	err := d.Dispatch(ctx, task)
	require.NoError(t, err)

	assert.Equal(t, orchestrator_domain.StatusRetrying, task.Status)
	events := eventBus.getPublishedEvents()
	require.Len(t, events, 1)
}
