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

import "time"

const (
	// TopicTaskCompleted is the event topic published when a task completes or
	// fails. The task dispatcher uses this topic to signal completion for
	// observability and workflow tracking.
	TopicTaskCompleted = "task.completed.v1"

	// TopicTaskDispatchHigh is the Watermill topic for distributing
	// high-priority tasks using competing-consumer semantics, processed
	// by more handlers (default 10) for higher throughput.
	TopicTaskDispatchHigh = "task.dispatch.high.v1"

	// TopicTaskDispatchNormal is the Watermill topic for distributing
	// normal-priority tasks using competing-consumer semantics, processed
	// by fewer handlers (default 5) than high priority.
	TopicTaskDispatchNormal = "task.dispatch.normal.v1"

	// TopicTaskDispatchLow is the Watermill message topic for distributing
	// low-priority tasks. Tasks published here use competing-consumer semantics
	// and are processed by the fewest handlers (default 2).
	TopicTaskDispatchLow = "task.dispatch.low.v1"
)

// CompletionEvent represents a task completion notification (success or
// failure). It is published to the event bus for observability and workflow
// tracking.
type CompletionEvent struct {
	// CompletedAt is when the task finished.
	CompletedAt time.Time `json:"completedAt"`

	// TaskID identifies the completed task.
	TaskID string `json:"taskId"`

	// WorkflowID identifies the workflow this task belongs to.
	WorkflowID string `json:"workflowId"`

	// ArtefactID is the unique identifier of the artefact that was processed.
	ArtefactID string `json:"artefactId"`

	// Status indicates the task result: "success" or "failure".
	Status string `json:"status"`

	// ErrorMessage holds the error details when Status is "failure".
	ErrorMessage string `json:"error,omitempty"`

	// DurationMs is the time taken to run the task, in milliseconds.
	DurationMs int64 `json:"durationMs"`
}
