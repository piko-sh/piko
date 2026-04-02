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

func Test_taskToPayload(t *testing.T) {
	t.Parallel()

	d := &watermillTaskDispatcher{}
	now := time.Now().UTC()

	task := &orchestrator_domain.Task{
		ID:                 "task-123",
		WorkflowID:         "wf-456",
		Executor:           "artefact.compiler",
		Payload:            map[string]any{"key": "value"},
		Config:             orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityHigh, MaxRetries: 3},
		Status:             orchestrator_domain.StatusPending,
		Attempt:            2,
		Result:             map[string]any{"status": "ok"},
		LastError:          "some error",
		DeduplicationKey:   "dedup-key",
		ExecuteAt:          now,
		ScheduledExecuteAt: now.Add(10 * time.Minute),
		CreatedAt:          now.Add(-1 * time.Hour),
		UpdatedAt:          now,
	}

	payload := d.taskToPayload(task)

	assert.Equal(t, "task-123", payload["id"])
	assert.Equal(t, "wf-456", payload["workflowID"])
	assert.Equal(t, "artefact.compiler", payload["executor"])
	assert.Equal(t, map[string]any{"key": "value"}, payload["payload"])
	assert.Equal(t, task.Config, payload["config"])
	assert.Equal(t, orchestrator_domain.StatusPending, payload["status"])
	assert.Equal(t, 2, payload["attempt"])
	assert.Equal(t, map[string]any{"status": "ok"}, payload["result"])
	assert.Equal(t, "some error", payload["lastError"])
	assert.Equal(t, "dedup-key", payload["deduplicationKey"])
	assert.Equal(t, now, payload["executeAt"])
	assert.Equal(t, now.Add(10*time.Minute), payload["scheduledExecuteAt"])
	assert.Equal(t, now.Add(-1*time.Hour), payload["createdAt"])
	assert.Equal(t, now, payload["updatedAt"])
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
				"Priority":   float64(orchestrator_domain.PriorityHigh),
				"Timeout":    float64(30 * time.Second),
				"MaxRetries": float64(5),
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
			"Priority":   float64(2),
			"Timeout":    float64(1000000000),
			"MaxRetries": float64(3),
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
			"Priority": float64(1),
		}
		result := d.parseTaskConfig(configMap)
		assert.Equal(t, orchestrator_domain.TaskPriority(1), result.Priority)
		assert.Equal(t, time.Duration(0), result.Timeout)
		assert.Equal(t, 0, result.MaxRetries)
	})

	t.Run("wrong types ignored", func(t *testing.T) {
		t.Parallel()
		configMap := map[string]any{
			"Priority":   "high",
			"Timeout":    "1s",
			"MaxRetries": "3",
		}
		result := d.parseTaskConfig(configMap)
		assert.Equal(t, orchestrator_domain.TaskConfig{}, result)
	})
}

func Test_taskToPayload_RoundTrip(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	config := orchestrator_domain.DefaultDispatcherConfig()
	d := newWatermillTaskDispatcher(config, eventBus, nil)

	now := time.Now().UTC()
	original := &orchestrator_domain.Task{
		ID:                 "round-trip-task",
		WorkflowID:         "wf-rt",
		Executor:           "test-executor",
		Payload:            map[string]any{"data": "value"},
		Config:             orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityHigh, MaxRetries: 3, Timeout: 5 * time.Second},
		Status:             orchestrator_domain.StatusPending,
		Attempt:            1,
		Result:             map[string]any{"result": "ok"},
		LastError:          "last err",
		DeduplicationKey:   "dedup-rt",
		ExecuteAt:          now,
		ScheduledExecuteAt: now.Add(5 * time.Minute),
		CreatedAt:          now.Add(-1 * time.Hour),
		UpdatedAt:          now,
	}

	payload := d.taskToPayload(original)
	restored, err := d.taskFromPayload(payload)
	require.NoError(t, err)

	assert.Equal(t, original.ID, restored.ID)
	assert.Equal(t, original.WorkflowID, restored.WorkflowID)
	assert.Equal(t, original.Executor, restored.Executor)
	assert.Equal(t, original.Payload, restored.Payload)
	assert.Equal(t, original.Status, restored.Status)
	assert.Equal(t, original.Attempt, restored.Attempt)
	assert.Equal(t, original.Result, restored.Result)
	assert.Equal(t, original.LastError, restored.LastError)
	assert.Equal(t, original.DeduplicationKey, restored.DeduplicationKey)
	assert.Equal(t, original.ExecuteAt, restored.ExecuteAt)
	assert.Equal(t, original.ScheduledExecuteAt, restored.ScheduledExecuteAt)
	assert.Equal(t, original.CreatedAt, restored.CreatedAt)
	assert.Equal(t, original.UpdatedAt, restored.UpdatedAt)
	assert.Equal(t, original.Config, restored.Config)
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
}

func Test_persistOrUpdateTask_Retry(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true

	d := newWatermillTaskDispatcher(config, eventBus, store)

	ctx := t.Context()

	t.Run("retry task updates status to pending", func(t *testing.T) {
		t.Parallel()
		task := &orchestrator_domain.Task{
			ID:         "retry-task",
			WorkflowID: "wf-1",
			Executor:   "test-executor",
			Status:     orchestrator_domain.StatusRetrying,
			Attempt:    1,
		}
		err := d.persistOrUpdateTask(ctx, task)
		require.NoError(t, err)
		assert.Equal(t, orchestrator_domain.StatusPending, task.Status)
	})

	t.Run("new task with dedup error", func(t *testing.T) {
		t.Parallel()
		localStore := &orchestrator_domain.MockTaskStore{
			CreateTaskWithDedupFunc: func(_ context.Context, _ *orchestrator_domain.Task) error {
				return orchestrator_domain.ErrDuplicateTask
			},
		}
		localD := newWatermillTaskDispatcher(config, eventBus, localStore)

		task := &orchestrator_domain.Task{
			ID:               "new-task",
			WorkflowID:       "wf-2",
			Executor:         "test-executor",
			Status:           orchestrator_domain.StatusPending,
			Attempt:          0,
			DeduplicationKey: "dedup-key",
		}
		err := localD.persistOrUpdateTask(ctx, task)
		require.Error(t, err)
		assert.ErrorIs(t, err, orchestrator_domain.ErrDuplicateTask)
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
	config.SyncPersistence = true
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
	config.SyncPersistence = true

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

func Test_watermillTaskDispatcher_Dispatch_PublishFails(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	eventBus.publishFunc = func(_ context.Context, _ string, _ orchestrator_domain.Event) error {
		return assert.AnError
	}
	config := orchestrator_domain.DefaultDispatcherConfig()

	d := newWatermillTaskDispatcher(config, eventBus, nil)
	ctx := t.Context()

	task := &orchestrator_domain.Task{
		ID:         "task-1",
		WorkflowID: "wf-1",
		Executor:   "test-executor",
	}

	err := d.Dispatch(ctx, task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "publishing task")
}

func Test_handleTaskEvent_ValidTask(t *testing.T) {
	t.Parallel()

	eventBus := newMockEventBus()
	store := &orchestrator_domain.MockTaskStore{}
	config := orchestrator_domain.DefaultDispatcherConfig()
	config.SyncPersistence = true

	d := newWatermillTaskDispatcher(config, eventBus, store)
	executor := newMockExecutor()
	d.RegisterExecutor(context.Background(), "test-executor", executor)

	task := &orchestrator_domain.Task{
		ID:         "task-valid",
		WorkflowID: "wf-1",
		Executor:   "test-executor",
		Payload:    map[string]any{"input": "data"},
	}
	payload := d.taskToPayload(task)

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
	config.SyncPersistence = true

	d := newWatermillTaskDispatcher(config, eventBus, store)
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

	assert.Equal(t, orchestrator_domain.StatusPending, task.Status)
	events := eventBus.getPublishedEvents()
	require.Len(t, events, 1)
}
