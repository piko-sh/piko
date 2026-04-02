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
	"fmt"
	"math"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

const (
	// defaultHighQueueWeight is the default weight for high priority queue items.
	defaultHighQueueWeight = 10

	// defaultNormalQueueWeight is the default weight for the normal priority queue.
	defaultNormalQueueWeight = 5

	// defaultLowQueueWeight is the weight for low priority queue items.
	defaultLowQueueWeight = 2
)

// randFunc is a function type that returns a random integer from 0 to n-1.
// Used for dependency injection in backoff calculation.
type randFunc func(n int) int

// completionEventData holds the data for a task completion event.
// This struct makes the completion event schema explicit and testable.
type completionEventData struct {
	// CompletedAt is when the task finished.
	CompletedAt time.Time

	// TaskID is the unique identifier for this task.
	TaskID string

	// WorkflowID is the unique identifier for the workflow.
	WorkflowID string

	// ArtefactID is the unique identifier for the completion artefact.
	ArtefactID string

	// Status indicates the result of the operation: "success" or "failure".
	Status string

	// Error holds the error message if parsing failed; empty on success.
	Error string

	// DurationMs is the completion time in milliseconds.
	DurationMs int64
}

// healthCheckInput holds the state needed to check if a service is healthy.
// This makes health logic testable without starting the service.
type healthCheckInput struct {
	// IsStopped indicates whether the orchestrator service has been stopped.
	IsStopped bool

	// TaskStoreNil bool // TaskStoreNil indicates whether the task store is nil.
	TaskStoreNil bool

	// ExecutorCount is the number of registered executors; 0 means degraded state.
	ExecutorCount int
}

// queueSelector determines the order in which priority queues should be polled.
// This implements weighted fair queuing where higher-weighted priorities
// are checked more frequently.
type queueSelector struct {
	// HighWeight is how many times high priority tasks appear in the poll order.
	HighWeight int

	// NormalWeight is how many times to poll the normal priority queue per cycle.
	NormalWeight int

	// LowWeight is how many times low priority tasks appear in the poll order.
	LowWeight int
}

// PollOrder returns the priorities in the order they should be polled based
// on weights.
//
// Each priority appears in the slice a number of times equal to its weight.
// Use it to test the weighted fair queuing logic.
//
// Returns []TaskPriority which contains priorities repeated by their weights.
func (qs queueSelector) PollOrder() []TaskPriority {
	order := make([]TaskPriority, 0, qs.HighWeight+qs.NormalWeight+qs.LowWeight)
	for range qs.HighWeight {
		order = append(order, PriorityHigh)
	}
	for range qs.NormalWeight {
		order = append(order, PriorityNormal)
	}
	for range qs.LowWeight {
		order = append(order, PriorityLow)
	}
	return order
}

// PollAttempts returns the number of poll attempts for a given priority level.
// This is used by the dispatcher's selectTask method.
//
// Takes priority (TaskPriority) which specifies the task priority level.
//
// Returns int which is the weight value for the given priority.
func (qs queueSelector) PollAttempts(priority TaskPriority) int {
	switch priority {
	case PriorityHigh:
		return qs.HighWeight
	case PriorityLow:
		return qs.LowWeight
	default:
		return qs.NormalWeight
	}
}

// calculateRetryBackoff computes the delay before retrying a failed task.
//
// Uses exponential backoff with configurable jitter. The formula is:
// base^attempt seconds + random jitter milliseconds, where base is 10 seconds.
//
// Takes attempt (int) which is the current attempt number (1-indexed after
// first failure).
// Takes randFunction (randFunc) which returns a random int in [0, n). Pass nil
// for no jitter.
//
// Returns time.Duration which is the calculated backoff delay.
func calculateRetryBackoff(attempt int, randFunction randFunc) time.Duration {
	backoff := time.Second * time.Duration(math.Pow(backoffBase, float64(attempt)))

	var jitter time.Duration
	if randFunction != nil {
		jitter = time.Duration(randFunction(jitterMilliseconds)) * time.Millisecond
	}

	return backoff + jitter
}

// buildCompletionEventPayload constructs the payload map for task completion
// events. This is a pure function that can be unit tested without an EventBus.
//
// Takes task (*Task) which provides the task details for the payload.
// Takes taskErr (error) which indicates the task outcome; nil means success.
// Takes duration (time.Duration) which specifies how long the task took.
// Takes now (time.Time) which provides the completion timestamp.
//
// Returns map[string]any which contains the structured event payload.
func buildCompletionEventPayload(task *Task, taskErr error, duration time.Duration, now time.Time) map[string]any {
	status := "success"
	errorMessage := ""
	if taskErr != nil {
		status = "failure"
		errorMessage = taskErr.Error()
	}

	artefactID := getPayloadString(task.Payload, "artefactID")

	return map[string]any{
		"taskId":      task.ID,
		"workflowId":  task.WorkflowID,
		"artefactId":  artefactID,
		"status":      status,
		"error":       errorMessage,
		"durationMs":  duration.Milliseconds(),
		"completedAt": now,
	}
}

// shouldRetryTask determines whether a task should be retried based on
// its current attempt count and configuration.
//
// Takes attempt (int) which is the current attempt number after failed
// execution.
// Takes configMaxRetries (int) which is the task's configured max retries,
// where 0 means use the default.
// Takes defaultMaxRetries (int) which is the system default max retries.
//
// Returns bool which is true if the task should be retried, false if it has
// exhausted retries.
func shouldRetryTask(attempt, configMaxRetries, defaultMaxRetries int) bool {
	maxRetries := configMaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	return attempt < maxRetries
}

// determineLivenessState works out the liveness health state and message.
// Liveness checks confirm the service is running and not stuck.
//
// Takes input (healthCheckInput) which provides the current service status.
//
// Returns healthprobe_dto.State which shows healthy or unhealthy state.
// Returns string which contains a readable status message.
func determineLivenessState(input healthCheckInput) (healthprobe_dto.State, string) {
	if input.IsStopped {
		return healthprobe_dto.StateUnhealthy, "Orchestrator service has been stopped"
	}
	if input.TaskStoreNil {
		return healthprobe_dto.StateUnhealthy, "Task store is not initialised"
	}
	return healthprobe_dto.StateHealthy, "Orchestrator service is running"
}

// determineReadinessState works out the readiness health state and message.
// Readiness checks confirm the service is ready to accept work.
//
// Takes input (healthCheckInput) which provides the current service status.
//
// Returns healthprobe_dto.State which shows the readiness level.
// Returns string which describes the current readiness condition.
func determineReadinessState(input healthCheckInput) (healthprobe_dto.State, string) {
	if input.IsStopped {
		return healthprobe_dto.StateUnhealthy, "Orchestrator service has been stopped"
	}
	if input.ExecutorCount == 0 {
		return healthprobe_dto.StateDegraded, "Orchestrator running but no executors registered"
	}
	return healthprobe_dto.StateHealthy, fmt.Sprintf("Orchestrator ready with %d executor(s)", input.ExecutorCount)
}

// defaultQueueSelector returns a queueSelector with production defaults.
//
// Returns queueSelector which is set up with the default queue weights.
func defaultQueueSelector() queueSelector {
	return queueSelector{
		HighWeight:   defaultHighQueueWeight,
		NormalWeight: defaultNormalQueueWeight,
		LowWeight:    defaultLowQueueWeight,
	}
}
