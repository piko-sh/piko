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
	"fmt"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// OrchestratorInspectorService implements the gRPC service for inspecting
// orchestrator tasks and workflows.
type OrchestratorInspectorService struct {
	pb.UnimplementedOrchestratorInspectorServiceServer

	// inspector provides task and workflow querying operations.
	inspector orchestrator_domain.OrchestratorInspector
}

// NewOrchestratorInspectorService creates a new OrchestratorInspectorService.
//
// Takes inspector (OrchestratorInspector) which provides orchestrator
// inspection capabilities.
//
// Returns *OrchestratorInspectorService which is ready for use as a gRPC
// service handler.
func NewOrchestratorInspectorService(inspector orchestrator_domain.OrchestratorInspector) *OrchestratorInspectorService {
	return &OrchestratorInspectorService{
		UnimplementedOrchestratorInspectorServiceServer: pb.UnimplementedOrchestratorInspectorServiceServer{},
		inspector: inspector,
	}
}

// GetTaskSummary returns task counts grouped by status.
//
// Returns *pb.GetTaskSummaryResponse which contains the task summaries.
// Returns error when the task summary cannot be retrieved.
func (s *OrchestratorInspectorService) GetTaskSummary(ctx context.Context, _ *pb.GetTaskSummaryRequest) (*pb.GetTaskSummaryResponse, error) {
	summaries, err := s.inspector.ListTaskSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing task summary: %w", err)
	}

	pbSummaries := make([]*pb.TaskSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.TaskSummary{
			Status: sum.Status,
			Count:  sum.Count,
		}
	}

	return &pb.GetTaskSummaryResponse{
		Summaries: pbSummaries,
	}, nil
}

// ListRecentTasks returns the most recently updated tasks.
//
// Takes request (*pb.ListRecentTasksRequest) which specifies the
// limit for results.
//
// Returns *pb.ListRecentTasksResponse which contains the list of recent tasks.
// Returns error when the underlying inspector fails to retrieve tasks.
func (s *OrchestratorInspectorService) ListRecentTasks(ctx context.Context, request *pb.ListRecentTasksRequest) (*pb.ListRecentTasksResponse, error) {
	limit := request.GetLimit()
	if limit <= 0 {
		limit = defaultListLimit
	}

	tasks, err := s.inspector.ListRecentTasks(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing recent tasks: %w", err)
	}

	return &pb.ListRecentTasksResponse{
		Tasks: convertTasksToPB(tasks),
	}, nil
}

// ListWorkflowSummary returns workflow-level aggregates.
//
// Takes request (*pb.ListWorkflowSummaryRequest) which specifies the query limit.
//
// Returns *pb.ListWorkflowSummaryResponse which contains the workflow summaries.
// Returns error when the underlying inspector fails to retrieve summaries.
func (s *OrchestratorInspectorService) ListWorkflowSummary(ctx context.Context, request *pb.ListWorkflowSummaryRequest) (*pb.ListWorkflowSummaryResponse, error) {
	limit := request.GetLimit()
	if limit <= 0 {
		limit = defaultListLimit
	}

	summaries, err := s.inspector.ListWorkflowSummary(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing workflow summary: %w", err)
	}

	pbSummaries := make([]*pb.WorkflowSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.WorkflowSummary{
			WorkflowId:    sum.WorkflowID,
			TaskCount:     sum.TaskCount,
			CompleteCount: sum.CompleteCount,
			FailedCount:   sum.FailedCount,
			ActiveCount:   sum.ActiveCount,
			CreatedAt:     sum.CreatedAt,
			UpdatedAt:     sum.UpdatedAt,
		}
	}

	return &pb.ListWorkflowSummaryResponse{
		Summaries: pbSummaries,
	}, nil
}

// WatchTasks streams task updates at the requested interval.
//
// Takes request (*pb.WatchTasksRequest) which specifies the update interval.
// Takes stream (pb.OrchestratorInspectorService_WatchTasksServer) which
// receives the streamed task updates.
//
// Returns error when the context is cancelled or sending an update fails.
func (s *OrchestratorInspectorService) WatchTasks(request *pb.WatchTasksRequest, stream pb.OrchestratorInspectorService_WatchTasksServer) error {
	return runWatchLoop(stream.Context(), request.GetIntervalMs(), func() error {
		return s.sendTasksUpdate(stream.Context(), stream)
	}, "task", nil)
}

// sendTasksUpdate fetches and sends a single task update to the stream.
//
// Takes stream (pb.OrchestratorInspectorService_WatchTasksServer) which
// receives the task update message.
//
// Returns error when the stream send fails.
//
//nolint:dupl // similar structure, different gRPC types
func (s *OrchestratorInspectorService) sendTasksUpdate(ctx context.Context, stream pb.OrchestratorInspectorService_WatchTasksServer) error {
	ctx, l := logger_domain.From(ctx, log)
	summaries, err := s.inspector.ListTaskSummary(ctx)
	if err != nil {
		l.Error("Failed to list task summary in WatchTasks", Error(err))
		return nil
	}

	tasks, err := s.inspector.ListRecentTasks(ctx, defaultListLimit)
	if err != nil {
		l.Error("Failed to list recent tasks in WatchTasks", Error(err))
		return nil
	}

	return stream.Send(&pb.TasksUpdate{
		Summaries:   convertTaskSummariesToPB(summaries),
		RecentTasks: convertTasksToPB(tasks),
		TimestampMs: time.Now().UnixMilli(),
	})
}

// convertTaskSummariesToPB converts domain task summaries to protobuf format.
//
// Takes summaries ([]orchestrator_domain.TaskSummary) which contains the domain
// task summaries to convert.
//
// Returns []*pb.TaskSummary which contains the converted protobuf summaries.
func convertTaskSummariesToPB(summaries []orchestrator_domain.TaskSummary) []*pb.TaskSummary {
	pbSummaries := make([]*pb.TaskSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.TaskSummary{
			Status: sum.Status,
			Count:  sum.Count,
		}
	}
	return pbSummaries
}

// convertTasksToPB converts domain tasks to protobuf format.
//
// Takes tasks ([]orchestrator_domain.TaskListItem) which contains the domain
// task items to convert.
//
// Returns []*pb.TaskListItem which contains the converted protobuf task items.
func convertTasksToPB(tasks []orchestrator_domain.TaskListItem) []*pb.TaskListItem {
	pbTasks := make([]*pb.TaskListItem, len(tasks))
	for i, task := range tasks {
		pbTasks[i] = &pb.TaskListItem{
			Id:         task.ID,
			WorkflowId: task.WorkflowID,
			Executor:   task.Executor,
			Status:     task.Status,
			Priority:   task.Priority,
			Attempt:    task.Attempt,
			LastError:  task.LastError,
			CreatedAt:  task.CreatedAt,
			UpdatedAt:  task.UpdatedAt,
		}
	}
	return pbTasks
}
