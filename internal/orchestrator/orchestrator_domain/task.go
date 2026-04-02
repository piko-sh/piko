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
	"maps"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskPriority defines how important a task is compared to other tasks.
// Tasks with higher priority are processed before those with lower priority.
type TaskPriority int

const (
	// PriorityLow is for tasks that can wait while higher priority work runs first.
	PriorityLow TaskPriority = iota

	// PriorityNormal is the default priority level for tasks.
	PriorityNormal

	// PriorityHigh is for tasks that should be processed before others.
	PriorityHigh
)

// TaskStatus represents the current state of a task in its lifecycle.
type TaskStatus string

const (
	// StatusScheduled means the task will run at a set time in the future.
	StatusScheduled TaskStatus = "SCHEDULED"

	// StatusPending means the task is waiting to be processed.
	StatusPending TaskStatus = "PENDING"

	// StatusProcessing means a worker is running the task.
	StatusProcessing TaskStatus = "PROCESSING"

	// StatusRetrying indicates the task failed but is scheduled for another attempt.
	StatusRetrying TaskStatus = "RETRYING"

	// StatusFailed indicates the task has failed after exhausting all retries.
	StatusFailed TaskStatus = "FAILED"

	// StatusComplete indicates the task has finished successfully.
	StatusComplete TaskStatus = "COMPLETE"

	// defaultNewTaskTimeout is the default time limit for newly created tasks.
	defaultNewTaskTimeout = 5 * time.Minute

	// defaultNewTaskMaxRetries is the most times a new task will retry if it fails.
	defaultNewTaskMaxRetries = 3
)

// TaskConfig holds the settings for running a task.
type TaskConfig struct {
	// Timeout is the maximum duration for task execution. A zero or negative
	// value means the system default timeout is used.
	Timeout time.Duration `json:"timeout"`

	// MaxRetries is the maximum number of retry attempts after failure.
	// A value of 0 or less uses the default from TaskProcessingConfig.
	MaxRetries int `json:"max_retries"`

	// Priority determines the order in which tasks are processed.
	Priority TaskPriority `json:"priority"`
}

// Task represents a unit of work to be run by the orchestrator.
// Tasks belong to workflows and are processed by registered executors.
type Task struct {
	// ExecuteAt is when the task is due to be processed.
	ExecuteAt time.Time `json:"execute_at"`

	// CreatedAt is when the task was first created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the task was last changed.
	UpdatedAt time.Time `json:"updated_at"`

	// ScheduledExecuteAt is when the task should be dispatched; zero means now.
	ScheduledExecuteAt time.Time `json:"scheduled_execute_at"`

	// Payload holds task-specific data passed to the executor.
	Payload map[string]any `json:"payload"`

	// Result holds the output data from task execution; nil until the task completes.
	Result map[string]any `json:"result,omitempty"`

	// Executor is the name of the registered handler that processes this task.
	Executor string `json:"executor"`

	// WorkflowID is the unique identifier of the workflow this task belongs to.
	WorkflowID string `json:"workflow_id"`

	// ID is the unique task identifier.
	ID string `json:"id"`

	// Status is the current state of the task.
	Status TaskStatus `json:"status"`

	// LastError contains the error message from the most recent failed
	// attempt; empty when the task succeeds or has not been attempted.
	LastError string `json:"last_error,omitempty"`

	// DeduplicationKey prevents duplicate active tasks with the same key. Only one
	// task with this key can be active (SCHEDULED, PENDING, PROCESSING, RETRYING)
	// at a time; empty string disables deduplication.
	DeduplicationKey string `json:"deduplication_key,omitempty"`

	// BuildTag is an optional tag that scopes the task to a particular
	// build run, allowing the dispatcher to filter results so that only
	// tasks from the current build are reported where empty means
	// untagged.
	BuildTag string `json:"build_tag,omitempty"`

	// Config holds task execution settings such as priority and timeout.
	Config TaskConfig `json:"config"`

	// Attempt is the current execution attempt number, starting from zero.
	Attempt int `json:"attempt"`

	// IsFatal indicates the task failed due to a non-retryable error. When
	// true, the task was not retried regardless of remaining retry budget.
	IsFatal bool `json:"is_fatal,omitempty"`

	// persisted is a transient flag indicating the task has already been saved
	// to the store by the batch inserter. PersistWithDedup checks this to
	// avoid a redundant INSERT when the dispatcher receives a task that was
	// already persisted as part of a batch.
	persisted bool
}

// TaskPool holds reusable Task objects to reduce allocations in high-throughput
// paths. It is exported so other packages (such as adapters) can get and put
// tasks from this central pool.
var TaskPool = sync.Pool{
	New: func() any {
		return &Task{
			ExecuteAt:          time.Time{},
			CreatedAt:          time.Time{},
			UpdatedAt:          time.Time{},
			ScheduledExecuteAt: time.Time{},
			Payload:            make(map[string]any),
			Result:             make(map[string]any),
			ID:                 "",
			WorkflowID:         "",
			Executor:           "",
			Status:             "",
			LastError:          "",
			DeduplicationKey:   "",
			Config:             TaskConfig{},
			Attempt:            0,
		}
	},
}

// NewTask retrieves a Task from the sync.Pool, resets it, and initialises it
// with the provided values. This is the primary factory function for creating
// tasks.
//
// Takes executor (string) which specifies the task executor type.
// Takes payload (map[string]any) which provides the task data.
//
// Returns *Task which is a ready-to-use task with a unique ID and default
// configuration.
func NewTask(executor string, payload map[string]any) *Task {
	task, ok := TaskPool.Get().(*Task)
	if !ok {
		task = &Task{}
	}
	task.Reset()

	now := time.Now().UTC()
	taskID := uuid.NewString()

	task.ID = taskID
	task.WorkflowID = taskID
	task.Executor = executor
	task.Status = StatusPending
	task.ExecuteAt = now
	task.Attempt = 0
	task.CreatedAt = now
	task.UpdatedAt = now
	task.Config = TaskConfig{
		Priority:   PriorityNormal,
		Timeout:    defaultNewTaskTimeout,
		MaxRetries: defaultNewTaskMaxRetries,
	}

	maps.Copy(task.Payload, payload)

	return task
}

// Reset clears all fields so the Task can be reused from sync.Pool.
//
// This stops data from a previous use leaking into the next use. Maps and
// slices are cleared rather than set to nil so their memory can be reused.
func (t *Task) Reset() {
	t.ID = ""
	t.WorkflowID = ""
	t.Executor = ""
	t.Status = ""
	t.LastError = ""
	t.DeduplicationKey = ""
	t.BuildTag = ""
	t.Attempt = 0
	t.persisted = false
	t.ExecuteAt = time.Time{}
	t.CreatedAt = time.Time{}
	t.UpdatedAt = time.Time{}
	t.Config = TaskConfig{}

	for k := range t.Payload {
		delete(t.Payload, k)
	}
	for k := range t.Result {
		delete(t.Result, k)
	}
}
