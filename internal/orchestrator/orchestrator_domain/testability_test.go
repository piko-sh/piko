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
	"errors"
	"testing"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func Test_calculateRetryBackoff(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		randFunction    randFunc
		name            string
		expectedMinimum time.Duration
		expectedMaximum time.Duration
		attempt         int
	}{
		{
			name:            "attempt 1 with no jitter",
			attempt:         1,
			randFunction:    nil,
			expectedMinimum: 10 * time.Second,
			expectedMaximum: 10 * time.Second,
		},
		{
			name:            "attempt 2 with no jitter",
			attempt:         2,
			randFunction:    nil,
			expectedMinimum: 100 * time.Second,
			expectedMaximum: 100 * time.Second,
		},
		{
			name:            "attempt 3 with no jitter",
			attempt:         3,
			randFunction:    nil,
			expectedMinimum: 1000 * time.Second,
			expectedMaximum: 1000 * time.Second,
		},
		{
			name:    "attempt 1 with zero jitter",
			attempt: 1,
			randFunction: func(n int) int {
				return 0
			},
			expectedMinimum: 10 * time.Second,
			expectedMaximum: 10 * time.Second,
		},
		{
			name:    "attempt 1 with max jitter",
			attempt: 1,
			randFunction: func(n int) int {
				return n - 1
			},
			expectedMinimum: 10*time.Second + 999*time.Millisecond,
			expectedMaximum: 10*time.Second + 999*time.Millisecond,
		},
		{
			name:    "attempt 1 with 500ms jitter",
			attempt: 1,
			randFunction: func(n int) int {
				return 500
			},
			expectedMinimum: 10*time.Second + 500*time.Millisecond,
			expectedMaximum: 10*time.Second + 500*time.Millisecond,
		},
		{
			name:            "attempt 0 with no jitter",
			attempt:         0,
			randFunction:    nil,
			expectedMinimum: 1 * time.Second,
			expectedMaximum: 1 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := calculateRetryBackoff(tc.attempt, tc.randFunction)

			if result < tc.expectedMinimum {
				t.Errorf("expected backoff >= %v, got %v", tc.expectedMinimum, result)
			}
			if result > tc.expectedMaximum {
				t.Errorf("expected backoff <= %v, got %v", tc.expectedMaximum, result)
			}
		})
	}
}

func Test_buildCompletionEventPayload(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	testCases := []struct {
		task             *Task
		taskErr          error
		name             string
		expectedStatus   string
		expectedError    string
		expectedTaskID   string
		expectedWorkflow string
		expectedArtefact string
		duration         time.Duration
	}{
		{
			name: "successful task",
			task: &Task{
				ID:         "task-123",
				WorkflowID: "workflow-456",
				Payload:    map[string]any{"artefactID": "artefact-789"},
			},
			taskErr:          nil,
			duration:         5 * time.Second,
			expectedStatus:   "success",
			expectedError:    "",
			expectedTaskID:   "task-123",
			expectedWorkflow: "workflow-456",
			expectedArtefact: "artefact-789",
		},
		{
			name: "failed task",
			task: &Task{
				ID:         "task-fail",
				WorkflowID: "workflow-fail",
				Payload:    map[string]any{},
			},
			taskErr:          errors.New("execution failed"),
			duration:         2 * time.Second,
			expectedStatus:   "failure",
			expectedError:    "execution failed",
			expectedTaskID:   "task-fail",
			expectedWorkflow: "workflow-fail",
			expectedArtefact: "",
		},
		{
			name: "task without artefact",
			task: &Task{
				ID:         "task-no-artefact",
				WorkflowID: "workflow-no-artefact",
				Payload:    map[string]any{"otherField": "value"},
			},
			taskErr:          nil,
			duration:         1 * time.Second,
			expectedStatus:   "success",
			expectedError:    "",
			expectedTaskID:   "task-no-artefact",
			expectedWorkflow: "workflow-no-artefact",
			expectedArtefact: "",
		},
		{
			name: "task with nil payload",
			task: &Task{
				ID:         "task-nil-payload",
				WorkflowID: "workflow-nil",
				Payload:    nil,
			},
			taskErr:          nil,
			duration:         500 * time.Millisecond,
			expectedStatus:   "success",
			expectedError:    "",
			expectedTaskID:   "task-nil-payload",
			expectedWorkflow: "workflow-nil",
			expectedArtefact: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := buildCompletionEventPayload(tc.task, tc.taskErr, tc.duration, fixedTime)

			if payload["taskId"] != tc.expectedTaskID {
				t.Errorf("taskId: expected %s, got %v", tc.expectedTaskID, payload["taskId"])
			}
			if payload["workflowId"] != tc.expectedWorkflow {
				t.Errorf("workflowId: expected %s, got %v", tc.expectedWorkflow, payload["workflowId"])
			}
			if payload["artefactId"] != tc.expectedArtefact {
				t.Errorf("artefactId: expected %s, got %v", tc.expectedArtefact, payload["artefactId"])
			}
			if payload["status"] != tc.expectedStatus {
				t.Errorf("status: expected %s, got %v", tc.expectedStatus, payload["status"])
			}
			if payload["error"] != tc.expectedError {
				t.Errorf("error: expected %s, got %v", tc.expectedError, payload["error"])
			}
			if payload["durationMs"] != tc.duration.Milliseconds() {
				t.Errorf("durationMs: expected %d, got %v", tc.duration.Milliseconds(), payload["durationMs"])
			}
			if payload["completedAt"] != fixedTime {
				t.Errorf("completedAt: expected %v, got %v", fixedTime, payload["completedAt"])
			}
		})
	}
}

func Test_shouldRetryTask(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		attempt           int
		configMaxRetries  int
		defaultMaxRetries int
		expected          bool
	}{
		{
			name:              "first attempt under config limit",
			attempt:           1,
			configMaxRetries:  3,
			defaultMaxRetries: 5,
			expected:          true,
		},
		{
			name:              "attempt equals config limit",
			attempt:           3,
			configMaxRetries:  3,
			defaultMaxRetries: 5,
			expected:          false,
		},
		{
			name:              "attempt exceeds config limit",
			attempt:           4,
			configMaxRetries:  3,
			defaultMaxRetries: 5,
			expected:          false,
		},
		{
			name:              "uses default when config is zero",
			attempt:           2,
			configMaxRetries:  0,
			defaultMaxRetries: 5,
			expected:          true,
		},
		{
			name:              "uses default when config is negative",
			attempt:           2,
			configMaxRetries:  -1,
			defaultMaxRetries: 5,
			expected:          true,
		},
		{
			name:              "attempt equals default limit",
			attempt:           5,
			configMaxRetries:  0,
			defaultMaxRetries: 5,
			expected:          false,
		},
		{
			name:              "zero attempts always retries",
			attempt:           0,
			configMaxRetries:  3,
			defaultMaxRetries: 5,
			expected:          true,
		},
		{
			name:              "single retry allowed",
			attempt:           1,
			configMaxRetries:  1,
			defaultMaxRetries: 5,
			expected:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := shouldRetryTask(tc.attempt, tc.configMaxRetries, tc.defaultMaxRetries)

			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func Test_determineLivenessState(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		expectedMessage string
		expectedState   healthprobe_dto.State
		input           healthCheckInput
	}{
		{
			name: "healthy when running",
			input: healthCheckInput{
				IsStopped:    false,
				TaskStoreNil: false,
			},
			expectedState:   healthprobe_dto.StateHealthy,
			expectedMessage: "Orchestrator service is running",
		},
		{
			name: "unhealthy when stopped",
			input: healthCheckInput{
				IsStopped:    true,
				TaskStoreNil: false,
			},
			expectedState:   healthprobe_dto.StateUnhealthy,
			expectedMessage: "Orchestrator service has been stopped",
		},
		{
			name: "unhealthy when task store nil",
			input: healthCheckInput{
				IsStopped:    false,
				TaskStoreNil: true,
			},
			expectedState:   healthprobe_dto.StateUnhealthy,
			expectedMessage: "Task store is not initialised",
		},
		{
			name: "stopped takes precedence over nil store",
			input: healthCheckInput{
				IsStopped:    true,
				TaskStoreNil: true,
			},
			expectedState:   healthprobe_dto.StateUnhealthy,
			expectedMessage: "Orchestrator service has been stopped",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state, message := determineLivenessState(tc.input)

			if state != tc.expectedState {
				t.Errorf("state: expected %v, got %v", tc.expectedState, state)
			}
			if message != tc.expectedMessage {
				t.Errorf("message: expected %q, got %q", tc.expectedMessage, message)
			}
		})
	}
}

func Test_determineReadinessState(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		expectedMessage string
		expectedState   healthprobe_dto.State
		input           healthCheckInput
	}{
		{
			name: "healthy with executors",
			input: healthCheckInput{
				IsStopped:     false,
				ExecutorCount: 3,
			},
			expectedState:   healthprobe_dto.StateHealthy,
			expectedMessage: "Orchestrator ready with 3 executor(s)",
		},
		{
			name: "healthy with single executor",
			input: healthCheckInput{
				IsStopped:     false,
				ExecutorCount: 1,
			},
			expectedState:   healthprobe_dto.StateHealthy,
			expectedMessage: "Orchestrator ready with 1 executor(s)",
		},
		{
			name: "degraded with no executors",
			input: healthCheckInput{
				IsStopped:     false,
				ExecutorCount: 0,
			},
			expectedState:   healthprobe_dto.StateDegraded,
			expectedMessage: "Orchestrator running but no executors registered",
		},
		{
			name: "unhealthy when stopped",
			input: healthCheckInput{
				IsStopped:     true,
				ExecutorCount: 5,
			},
			expectedState:   healthprobe_dto.StateUnhealthy,
			expectedMessage: "Orchestrator service has been stopped",
		},
		{
			name: "stopped takes precedence over no executors",
			input: healthCheckInput{
				IsStopped:     true,
				ExecutorCount: 0,
			},
			expectedState:   healthprobe_dto.StateUnhealthy,
			expectedMessage: "Orchestrator service has been stopped",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state, message := determineReadinessState(tc.input)

			if state != tc.expectedState {
				t.Errorf("state: expected %v, got %v", tc.expectedState, state)
			}
			if message != tc.expectedMessage {
				t.Errorf("message: expected %q, got %q", tc.expectedMessage, message)
			}
		})
	}
}

func Test_defaultQueueSelector(t *testing.T) {
	t.Parallel()

	qs := defaultQueueSelector()

	if qs.HighWeight != 10 {
		t.Errorf("HighWeight: expected 10, got %d", qs.HighWeight)
	}
	if qs.NormalWeight != 5 {
		t.Errorf("NormalWeight: expected 5, got %d", qs.NormalWeight)
	}
	if qs.LowWeight != 2 {
		t.Errorf("LowWeight: expected 2, got %d", qs.LowWeight)
	}
}

func Test_queueSelector_PollOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		selector     queueSelector
		expectedLen  int
		expectedHigh int
		expectedNorm int
		expectedLow  int
	}{
		{
			name:         "default weights",
			selector:     defaultQueueSelector(),
			expectedLen:  17,
			expectedHigh: 10,
			expectedNorm: 5,
			expectedLow:  2,
		},
		{
			name: "custom weights",
			selector: queueSelector{
				HighWeight:   3,
				NormalWeight: 2,
				LowWeight:    1,
			},
			expectedLen:  6,
			expectedHigh: 3,
			expectedNorm: 2,
			expectedLow:  1,
		},
		{
			name: "all equal",
			selector: queueSelector{
				HighWeight:   5,
				NormalWeight: 5,
				LowWeight:    5,
			},
			expectedLen:  15,
			expectedHigh: 5,
			expectedNorm: 5,
			expectedLow:  5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			order := tc.selector.PollOrder()

			if len(order) != tc.expectedLen {
				t.Errorf("expected len=%d, got len=%d", tc.expectedLen, len(order))
			}

			counts := make(map[TaskPriority]int)
			for _, p := range order {
				counts[p]++
			}

			if counts[PriorityHigh] != tc.expectedHigh {
				t.Errorf("high count: expected %d, got %d", tc.expectedHigh, counts[PriorityHigh])
			}
			if counts[PriorityNormal] != tc.expectedNorm {
				t.Errorf("normal count: expected %d, got %d", tc.expectedNorm, counts[PriorityNormal])
			}
			if counts[PriorityLow] != tc.expectedLow {
				t.Errorf("low count: expected %d, got %d", tc.expectedLow, counts[PriorityLow])
			}
		})
	}
}

func Test_queueSelector_PollAttempts(t *testing.T) {
	t.Parallel()

	qs := queueSelector{
		HighWeight:   10,
		NormalWeight: 5,
		LowWeight:    2,
	}

	testCases := []struct {
		priority TaskPriority
		expected int
	}{
		{priority: PriorityHigh, expected: 10},
		{priority: PriorityNormal, expected: 5},
		{priority: PriorityLow, expected: 2},
		{priority: TaskPriority(99), expected: 5},
	}

	for _, tc := range testCases {
		result := qs.PollAttempts(tc.priority)
		if result != tc.expected {
			t.Errorf("priority %d: expected %d, got %d", tc.priority, tc.expected, result)
		}
	}
}
