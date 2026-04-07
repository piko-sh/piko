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

func TestTaskPool_GetReturnsTask(t *testing.T) {
	t.Parallel()

	pooledTask := TaskPool.Get()
	task, ok := pooledTask.(*Task)

	if !ok {
		t.Fatal("TaskPool.Get() did not return a *Task")
	}
	require.NotNil(t, task, "TaskPool.Get() returned nil")

	if task.Payload == nil {
		t.Error("Payload map should be initialised")
	}
	if task.Result == nil {
		t.Error("Result map should be initialised")
	}

	TaskPool.Put(task)
}

func TestTask_Reset(t *testing.T) {
	t.Parallel()

	task := &Task{
		ID:                 "test-id",
		WorkflowID:         "workflow-id",
		Executor:           "test-executor",
		Status:             StatusComplete,
		LastError:          "some error",
		Attempt:            5,
		ExecuteAt:          time.Now(),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		ScheduledExecuteAt: time.Now(),
		Payload:            map[string]any{"key": "value"},
		Result:             map[string]any{"result": "data"},
		Config: TaskConfig{
			Timeout:    10 * time.Minute,
			MaxRetries: 10,
			Priority:   PriorityHigh,
		},
	}

	task.Reset()

	if task.ID != "" {
		t.Error("ID should be empty after reset")
	}
	if task.WorkflowID != "" {
		t.Error("WorkflowID should be empty after reset")
	}
	if task.Executor != "" {
		t.Error("Executor should be empty after reset")
	}
	if task.Status != "" {
		t.Error("Status should be empty after reset")
	}
	if task.LastError != "" {
		t.Error("LastError should be empty after reset")
	}
	if task.Attempt != 0 {
		t.Error("Attempt should be 0 after reset")
	}
	if !task.ExecuteAt.IsZero() {
		t.Error("ExecuteAt should be zero after reset")
	}
	if !task.CreatedAt.IsZero() {
		t.Error("CreatedAt should be zero after reset")
	}
	if !task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be zero after reset")
	}
	if task.Config.Timeout != 0 {
		t.Error("Config.Timeout should be zero after reset")
	}
	if task.Config.MaxRetries != 0 {
		t.Error("Config.MaxRetries should be zero after reset")
	}
	if task.Config.Priority != 0 {
		t.Error("Config.Priority should be zero after reset")
	}

	if task.Payload == nil {
		t.Error("Payload should not be nil after reset")
	}
	if len(task.Payload) != 0 {
		t.Error("Payload should be empty after reset")
	}
	if task.Result == nil {
		t.Error("Result should not be nil after reset")
	}
	if len(task.Result) != 0 {
		t.Error("Result should be empty after reset")
	}
}

func TestTask_Reset_PreservesMapAllocation(t *testing.T) {
	t.Parallel()

	task := &Task{
		Payload: make(map[string]any, 10),
		Result:  make(map[string]any, 10),
	}
	task.Payload["key1"] = "value1"
	task.Payload["key2"] = "value2"

	payloadBefore := task.Payload
	resultBefore := task.Result

	task.Reset()

	if task.Payload == nil || task.Result == nil {
		t.Error("Maps should not be nil after reset")
	}
	if len(task.Payload) != 0 || len(task.Result) != 0 {
		t.Error("Maps should be cleared after reset")
	}

	task.Payload["new"] = "value"
	task.Result["new"] = "result"

	_ = payloadBefore
	_ = resultBefore
}

func TestNewTask_SetsDefaults(t *testing.T) {
	t.Parallel()

	executor := "test-executor"
	payload := map[string]any{"key": "value"}

	task := NewTask(executor, payload)
	defer TaskPool.Put(task)

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

	if task.ExecuteAt.IsZero() {
		t.Error("ExecuteAt should be set")
	}
	if task.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	if task.Config.Priority != PriorityNormal {
		t.Errorf("Config.Priority: expected %d, got %d", PriorityNormal, task.Config.Priority)
	}
	if task.Config.Timeout != 5*time.Minute {
		t.Errorf("Config.Timeout: expected %v, got %v", 5*time.Minute, task.Config.Timeout)
	}
	if task.Config.MaxRetries != 3 {
		t.Errorf("Config.MaxRetries: expected 3, got %d", task.Config.MaxRetries)
	}

	if task.Payload["key"] != "value" {
		t.Error("Payload should contain provided values")
	}
}

func TestNewTask_CopiesPayload(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"original": "value"}

	task := NewTask("executor", payload)
	defer TaskPool.Put(task)

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
	defer TaskPool.Put(task)

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
		TaskPool.Put(task)
	}
}

func TestNewTask_FromPool_ReusesObjects(t *testing.T) {
	t.Parallel()

	task1 := NewTask("executor", map[string]any{"key": "value"})
	task1ID := task1.ID
	TaskPool.Put(task1)

	task2 := NewTask("executor", map[string]any{"other": "data"})
	defer TaskPool.Put(task2)

	if task2.ID == task1ID {
		t.Error("New task should have a different ID")
	}
	if task2.Payload["key"] == "value" {
		t.Error("Reused task should have been reset (old payload data leaked)")
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

func TestTask_ScheduledExecuteAt_NotAffectedByReset(t *testing.T) {
	t.Parallel()

	task := &Task{
		ScheduledExecuteAt: time.Now(),
		Payload:            make(map[string]any),
		Result:             make(map[string]any),
	}

	task.Reset()
}
