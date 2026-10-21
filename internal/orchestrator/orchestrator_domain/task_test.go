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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPriorityOrder(t *testing.T) {
	t.Parallel()

	if PriorityLow >= PriorityNormal {
		t.Error("PriorityLow should be less than PriorityNormal")
	}
	if PriorityNormal >= PriorityHigh {
		t.Error("PriorityNormal should be less than PriorityHigh")
	}
}

func TestTaskStatusValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status   TaskStatus
		expected string
	}{
		{status: StatusScheduled, expected: "SCHEDULED"},
		{status: StatusPending, expected: "PENDING"},
		{status: StatusProcessing, expected: "PROCESSING"},
		{status: StatusRetrying, expected: "RETRYING"},
		{status: StatusFailed, expected: "FAILED"},
		{status: StatusComplete, expected: "COMPLETE"},
	}

	for _, tc := range testCases {
		if string(tc.status) != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.status)
		}
	}
}

func TestNewTask_SetsDefaults(t *testing.T) {
	t.Parallel()

	executor := "test-executor"
	payload := map[string]any{"key": "value"}

	task := NewTask(executor, payload)
	if task.ID == "" {
		t.Error("ID should be set")
	}

	if task.WorkflowID != task.ID {
		t.Errorf("WorkflowID should default to ID, got %s", task.WorkflowID)
	}

	if task.Executor != executor {
		t.Errorf("Executor: expected %s, got %s", executor, task.Executor)
	}

	if task.Status != StatusPending {
		t.Errorf("Status: expected %s, got %s", StatusPending, task.Status)
	}

	if task.Attempt != 0 {
		t.Errorf("Attempt: expected 0, got %d", task.Attempt)
	}

	if !task.ExecuteAt.IsZero() || !task.CreatedAt.IsZero() || !task.UpdatedAt.IsZero() {
		t.Errorf("times should be left for admission to stamp, got execute=%v created=%v updated=%v",
			task.ExecuteAt, task.CreatedAt, task.UpdatedAt)
	}

	if task.Config.Priority != PriorityNormal {
		t.Errorf("Config.Priority: expected %d, got %d", PriorityNormal, task.Config.Priority)
	}
	if task.Config.Timeout != 0 {
		t.Errorf("Config.Timeout: expected unset so the dispatcher default applies, got %v", task.Config.Timeout)
	}
	if task.Config.MaxRetries != 0 {
		t.Errorf("Config.MaxRetries: expected unset so the dispatcher default applies, got %d", task.Config.MaxRetries)
	}

	if task.Payload["key"] != "value" {
		t.Error("Payload should contain provided values")
	}
}

func TestNewTask_CopiesPayload(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"original": "value"}

	task := NewTask("executor", payload)
	payload["original"] = "modified"
	payload["new"] = "added"

	if task.Payload["original"] != "value" {
		t.Error("Task payload should be a copy, not a reference")
	}
	if _, exists := task.Payload["new"]; exists {
		t.Error("Task payload should not contain new keys added to original")
	}
}

func TestNewTask_NilPayload(t *testing.T) {
	t.Parallel()

	task := NewTask("executor", nil)
	if task.Payload == nil {
		t.Error("Payload should not be nil even when created with nil")
	}
	if len(task.Payload) != 0 {
		t.Error("Payload should be empty when created with nil")
	}
}

func TestNewTask_GeneratesUniqueIDs(t *testing.T) {
	t.Parallel()

	ids := make(map[string]bool)
	for range 100 {
		task := NewTask("executor", nil)
		if ids[task.ID] {
			t.Errorf("Duplicate ID generated: %s", task.ID)
		}
		ids[task.ID] = true
	}
}

func TestNewTaskID_IsTimeOrderedVersion7(t *testing.T) {
	t.Parallel()

	first, err := uuid.Parse(NewTaskID())
	require.NoError(t, err)
	assert.Equal(t, uuid.Version(7), first.Version())
	time.Sleep(2 * time.Millisecond)
	second := NewTaskID()
	assert.Less(t, first.String(), second, "a later ID sorts after an earlier one")
}

func TestNewTask_DoesNotShareState(t *testing.T) {
	t.Parallel()

	task1 := NewTask("executor", map[string]any{"key": "value"})
	task2 := NewTask("executor", map[string]any{"other": "data"})

	if task2.ID == task1.ID {
		t.Error("New task should have a different ID")
	}
	if task2.Payload["key"] == "value" {
		t.Error("A new task must not see another task's payload")
	}
}

func TestTaskConfig_ZeroValue(t *testing.T) {
	t.Parallel()

	var config TaskConfig

	if config.Timeout != 0 {
		t.Error("Zero value Timeout should be 0")
	}
	if config.MaxRetries != 0 {
		t.Error("Zero value MaxRetries should be 0")
	}
	if config.Priority != PriorityLow {
		t.Errorf("Zero value Priority should be PriorityLow (0), got %d", config.Priority)
	}
}

func TestTask_Persisted(t *testing.T) {
	t.Parallel()

	task := &Task{}
	assert.False(t, task.Persisted())

	task.persisted = true
	assert.True(t, task.Persisted())
}
