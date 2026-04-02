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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"testing"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestOrchestratorInspectorService_GetTaskSummary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inspector     *mockOrchestratorInspector
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns summaries from inspector",
			inspector: &mockOrchestratorInspector{
				taskSummaryReturn: []orchestrator_domain.TaskSummary{
					{Status: "PENDING", Count: 5},
					{Status: "COMPLETE", Count: 10},
					{Status: "FAILED", Count: 2},
				},
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "returns empty when no summaries",
			inspector: &mockOrchestratorInspector{
				taskSummaryReturn: []orchestrator_domain.TaskSummary{},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockOrchestratorInspector{
				taskSummaryError: errors.New("database error"),
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewOrchestratorInspectorService(tc.inspector)

			response, err := service.GetTaskSummary(context.Background(), &pb.GetTaskSummaryRequest{})

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(response.Summaries) != tc.expectedCount {
				t.Errorf("expected %d summaries, got %d", tc.expectedCount, len(response.Summaries))
			}
		})
	}
}

func TestOrchestratorInspectorService_ListRecentTasks(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inspector     *mockOrchestratorInspector
		request       *pb.ListRecentTasksRequest
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns tasks from inspector",
			inspector: &mockOrchestratorInspector{
				recentTasksReturn: []orchestrator_domain.TaskListItem{
					{ID: "task-1", Status: "PENDING", Executor: "worker-1"},
					{ID: "task-2", Status: "COMPLETE", Executor: "worker-2"},
				},
			},
			request:       &pb.ListRecentTasksRequest{Limit: 10},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "uses default limit when not specified",
			inspector: &mockOrchestratorInspector{
				recentTasksReturn: []orchestrator_domain.TaskListItem{
					{ID: "task-1"},
				},
			},
			request:       &pb.ListRecentTasksRequest{Limit: 0},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockOrchestratorInspector{
				recentTasksError: errors.New("database error"),
			},
			request:       &pb.ListRecentTasksRequest{},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewOrchestratorInspectorService(tc.inspector)

			response, err := service.ListRecentTasks(context.Background(), tc.request)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(response.Tasks) != tc.expectedCount {
				t.Errorf("expected %d tasks, got %d", tc.expectedCount, len(response.Tasks))
			}
		})
	}
}

func TestOrchestratorInspectorService_ListWorkflowSummary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inspector     *mockOrchestratorInspector
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns workflow summaries",
			inspector: &mockOrchestratorInspector{
				workflowSummaryReturn: []orchestrator_domain.WorkflowSummary{
					{
						WorkflowID:    "workflow-1",
						TaskCount:     10,
						CompleteCount: 8,
						FailedCount:   1,
						ActiveCount:   1,
					},
				},
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockOrchestratorInspector{
				workflowSummaryError: errors.New("database error"),
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewOrchestratorInspectorService(tc.inspector)

			response, err := service.ListWorkflowSummary(context.Background(), &pb.ListWorkflowSummaryRequest{})

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(response.Summaries) != tc.expectedCount {
				t.Errorf("expected %d summaries, got %d", tc.expectedCount, len(response.Summaries))
			}
		})
	}
}

func TestConvertTasksToPB(t *testing.T) {
	t.Parallel()

	tasks := []orchestrator_domain.TaskListItem{
		{
			ID:         "task-123",
			WorkflowID: "workflow-456",
			Executor:   "worker-1",
			Status:     "PENDING",
			Priority:   10,
			Attempt:    1,
			LastError:  nil,
			CreatedAt:  1000,
			UpdatedAt:  2000,
		},
	}

	result := convertTasksToPB(tasks)

	if len(result) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result))
	}

	task := result[0]
	if task.Id != "task-123" {
		t.Errorf("expected ID task-123, got %s", task.Id)
	}
	if task.WorkflowId != "workflow-456" {
		t.Errorf("expected WorkflowID workflow-456, got %s", task.WorkflowId)
	}
	if task.Executor != "worker-1" {
		t.Errorf("expected Executor worker-1, got %s", task.Executor)
	}
	if task.Status != "PENDING" {
		t.Errorf("expected Status PENDING, got %s", task.Status)
	}
	if task.Priority != 10 {
		t.Errorf("expected Priority 10, got %d", task.Priority)
	}
}

func TestConvertTaskSummariesToPB(t *testing.T) {
	t.Parallel()

	summaries := []orchestrator_domain.TaskSummary{
		{Status: "PENDING", Count: 5},
		{Status: "COMPLETE", Count: 10},
	}

	result := convertTaskSummariesToPB(summaries)

	if len(result) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(result))
	}

	if result[0].Status != "PENDING" || result[0].Count != 5 {
		t.Errorf("first summary mismatch: %+v", result[0])
	}
	if result[1].Status != "COMPLETE" || result[1].Count != 10 {
		t.Errorf("second summary mismatch: %+v", result[1])
	}
}
