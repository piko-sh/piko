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

import "context"

// TaskSummary holds the count of tasks grouped by status for dashboard display.
type TaskSummary struct {
	// Status is the task state used for grouping counts.
	Status string

	// Count is the number of tasks with this status.
	Count int64
}

// TaskListItem holds task data for list display in the TUI.
type TaskListItem struct {
	// LastError is the most recent error message if the task failed; nil if no error.
	LastError *string

	// ID is the unique identifier for the task.
	ID string

	// WorkflowID is the identifier of the workflow this task belongs to.
	WorkflowID string

	// Executor identifies the worker or service that runs this task.
	Executor string

	// Status is the current state of the task.
	Status string

	// CreatedAt is the Unix timestamp when the task was created.
	CreatedAt int64

	// UpdatedAt is the Unix timestamp when the task was last changed.
	UpdatedAt int64

	// Priority is the task execution priority; higher values run first.
	Priority int32

	// Attempt is the current retry attempt number for this task.
	Attempt int32
}

// WorkflowSummary holds totals and counts for a workflow, used in dashboard views.
type WorkflowSummary struct {
	// WorkflowID is the unique identifier for the workflow.
	WorkflowID string

	// TaskCount is the total number of tasks in the workflow.
	TaskCount int64

	// CompleteCount is the number of tasks that have finished successfully.
	CompleteCount int64

	// FailedCount is the number of workflow tasks that did not complete.
	FailedCount int64

	// ActiveCount is the number of workflow instances that are currently running.
	ActiveCount int64

	// CreatedAt is the Unix timestamp when the workflow was created.
	CreatedAt int64

	// UpdatedAt is the Unix timestamp when the workflow was last changed.
	UpdatedAt int64
}

// OrchestratorInspector provides read-only access to orchestrator state.
// The monitoring service uses the port to show task and workflow data in the TUI
// without needing direct database access.
type OrchestratorInspector interface {
	// ListTaskSummary returns task counts grouped by status.
	//
	// Returns []TaskSummary which contains the count for each status.
	// Returns error when the query fails.
	ListTaskSummary(ctx context.Context) ([]TaskSummary, error)

	// ListRecentTasks returns the most recently updated tasks.
	//
	// Takes limit (int32) which specifies the maximum number of tasks to return.
	//
	// Returns []TaskListItem which contains the task data for display.
	// Returns error when the query fails.
	ListRecentTasks(ctx context.Context, limit int32) ([]TaskListItem, error)

	// ListWorkflowSummary returns workflow-level aggregates.
	//
	// Takes limit (int32) which specifies the maximum number of workflows to return.
	//
	// Returns []WorkflowSummary which contains aggregated workflow data.
	// Returns error when the query fails.
	ListWorkflowSummary(ctx context.Context, limit int32) ([]WorkflowSummary, error)
}
