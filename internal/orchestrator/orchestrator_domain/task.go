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
	"time"
)

// TaskPriority defines how important a task is compared to other tasks. Tasks with higher
// priority are processed before those with lower priority.
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
	// StatusScheduled means the task will run at a set later time.
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
)

// TaskConfig holds the settings for running a task.
type TaskConfig struct {
	// Timeout is the maximum duration for task execution. A zero or negative value means the
	// system default timeout is used.
	Timeout time.Duration `json:"timeout"`

	// MaxRetries is the maximum number of retry attempts after failure. A value of 0 or less
	// uses the default from TaskProcessingConfig.
	MaxRetries int `json:"max_retries"`

	// Priority determines the order in which tasks are processed.
	Priority TaskPriority `json:"priority"`
}

// Task represents a unit of work to be run by the orchestrator. Tasks belong to workflows
// and are processed by registered executors.
type Task struct {
	// continuation holds the successor to persist when this task settles successfully.
	continuation *settledContinuation

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

	// LastPublishedAt records the last task-message publication time; zero means unrecorded.
	LastPublishedAt time.Time `json:"last_published_at,omitzero"`

	// Rerun requests another run after settlement; nil means no request is pending.
	Rerun *RerunRequest `json:"rerun,omitempty"`

	// receiptID identifies the receipt pending admission with this task.
	receiptID string

	// Executor is the name of the registered handler that processes this task.
	Executor string `json:"executor"`

	// WorkflowID is the unique identifier of the workflow this task belongs to.
	WorkflowID string `json:"workflow_id"`

	// ID is the unique task identifier.
	ID string `json:"id"`

	// Status is the current state of the task.
	Status TaskStatus `json:"status"`

	// LastError contains the error message from the most recent failed attempt; empty when
	// the task succeeds or has not been attempted.
	LastError string `json:"last_error,omitempty"`

	// DeduplicationKey prevents duplicate active tasks with the same key. Only one task with
	// this key can be active (SCHEDULED, PENDING, PROCESSING, RETRYING) at a time; empty
	// string disables deduplication.
	DeduplicationKey string `json:"deduplication_key,omitempty"`

	// BuildTag is an optional tag that scopes the task to a particular build run, allowing
	// the dispatcher to filter results so that only tasks from the current build are
	// reported where empty means untagged.
	BuildTag string `json:"build_tag,omitempty"`

	// OwnerInstance identifies the node instance executing the task while it is PROCESSING;
	// empty otherwise.
	OwnerInstance string `json:"owner_instance,omitempty"`

	// Config holds task execution settings such as priority and timeout.
	Config TaskConfig `json:"config"`

	// Attempt is the current execution attempt number, starting from zero.
	Attempt int `json:"attempt"`

	// claimed identifies a delivery whose attempt was atomically persisted.
	claimed bool

	// settled marks attempts whose outcomes have already been handled.
	settled bool

	// IsFatal indicates the task failed due to a non-retryable error. When true, the task
	// was not retried regardless of remaining retry budget.
	IsFatal bool `json:"is_fatal,omitempty"`

	// Background marks maintenance tasks excluded from build tags and completion checks.
	Background bool `json:"background,omitempty"`

	// persisted is a transient flag indicating the task has already been saved to the store
	// by the batch inserter. PersistWithDedup checks this to avoid a redundant INSERT when
	// the dispatcher receives a task that was already persisted as part of a batch.
	persisted bool
}

// NewTask creates a task for an executor.
//
// Timestamps, timeout, and retry limit remain unset until admission applies the
// configured clock and defaults.
//
// Takes executor (string) which specifies the task executor type.
// Takes payload (map[string]any) which provides the task data; the task keeps a copy.
//
// Returns *Task which is a ready-to-use task with a unique ID and normal priority.
func NewTask(executor string, payload map[string]any) *Task {
	taskID := NewTaskID()
	task := &Task{
		continuation:       nil,
		ExecuteAt:          time.Time{},
		CreatedAt:          time.Time{},
		UpdatedAt:          time.Time{},
		ScheduledExecuteAt: time.Time{},
		Payload:            make(map[string]any, len(payload)),
		Result:             nil,
		LastPublishedAt:    time.Time{},
		Rerun:              nil,
		receiptID:          "",
		Executor:           executor,
		WorkflowID:         taskID,
		ID:                 taskID,
		Status:             StatusPending,
		LastError:          "",
		DeduplicationKey:   "",
		BuildTag:           "",
		OwnerInstance:      "",
		Config:             TaskConfig{Timeout: 0, MaxRetries: 0, Priority: PriorityNormal},
		Attempt:            0,
		claimed:            false,
		settled:            false,
		IsFatal:            false,
		Background:         false,
		persisted:          false,
	}
	maps.Copy(task.Payload, payload)
	return task
}

// Persisted reports whether the task's record has already been created in the task store,
// so a later dispatch of the same task does not insert it again.
//
// Returns bool which is true once the record exists.
func (t *Task) Persisted() bool {
	return t.persisted
}

// ClearAttemptState removes the claim, settlement, successor, and receipt bookkeeping for
// an attempt.
func (t *Task) ClearAttemptState() {
	t.continuation = nil
	t.receiptID = ""
	t.claimed = false
	t.settled = false
}

// stampAdmission fills any unset creation, update, and execution times with now.
//
// Times already set by the caller, such as a scheduled execution time, are kept.
//
// Takes now (time.Time) which is the admitting component's current time.
func (t *Task) stampAdmission(now time.Time) {
	now = now.UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}
	if t.ExecuteAt.IsZero() {
		t.ExecuteAt = now
	}
}
