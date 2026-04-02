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

	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// DispatcherInspectorService implements the gRPC service for inspecting
// dispatcher state and dead letter queues.
type DispatcherInspectorService struct {
	pb.UnimplementedDispatcherInspectorServiceServer

	// inspector provides dispatcher and DLQ querying operations.
	inspector dispatcher_domain.DispatcherInspector
}

// NewDispatcherInspectorService creates a new DispatcherInspectorService.
//
// Takes inspector (DispatcherInspector) which provides dispatcher inspection
// capabilities.
//
// Returns *DispatcherInspectorService which is ready for use as a gRPC
// service handler.
func NewDispatcherInspectorService(inspector dispatcher_domain.DispatcherInspector) *DispatcherInspectorService {
	return &DispatcherInspectorService{
		UnimplementedDispatcherInspectorServiceServer: pb.UnimplementedDispatcherInspectorServiceServer{},
		inspector: inspector,
	}
}

// GetDispatcherSummary returns statistics for all configured dispatchers.
//
// Returns *pb.GetDispatcherSummaryResponse which contains the dispatcher
// summaries.
// Returns error when the summaries cannot be retrieved.
func (s *DispatcherInspectorService) GetDispatcherSummary(ctx context.Context, _ *pb.GetDispatcherSummaryRequest) (*pb.GetDispatcherSummaryResponse, error) {
	summaries, err := s.inspector.GetDispatcherSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting dispatcher summaries: %w", err)
	}

	pbSummaries := make([]*pb.DispatcherSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.DispatcherSummary{
			Type:            sum.Type,
			QueuedItems:     safeconv.IntToInt32(sum.QueuedItems),
			RetryQueueSize:  safeconv.IntToInt32(sum.RetryQueueSize),
			DeadLetterCount: safeconv.IntToInt32(sum.DeadLetterCount),
			TotalProcessed:  sum.TotalProcessed,
			TotalSuccessful: sum.TotalSuccessful,
			TotalFailed:     sum.TotalFailed,
			TotalRetries:    sum.TotalRetries,
			UptimeMs:        sum.Uptime.Milliseconds(),
		}
	}

	return &pb.GetDispatcherSummaryResponse{
		Summaries: pbSummaries,
	}, nil
}

// ListDLQEntries returns dead letter queue entries for a specific dispatcher.
//
// Takes request (*pb.ListDLQEntriesRequest) which specifies the
// dispatcher type and limit.
//
// Returns *pb.ListDLQEntriesResponse which contains the DLQ entries.
// Returns error when the entries cannot be retrieved.
func (s *DispatcherInspectorService) ListDLQEntries(ctx context.Context, request *pb.ListDLQEntriesRequest) (*pb.ListDLQEntriesResponse, error) {
	limit := int(request.GetLimit())
	if limit <= 0 {
		limit = defaultListLimit
	}

	entries, err := s.inspector.GetDLQEntries(ctx, request.GetDispatcherType(), limit)
	if err != nil {
		return nil, fmt.Errorf("listing DLQ entries: %w", err)
	}

	pbEntries := make([]*pb.DLQEntry, len(entries))
	for i := range entries {
		pbEntries[i] = &pb.DLQEntry{
			Id:             entries[i].ID,
			Type:           entries[i].Type,
			OriginalError:  entries[i].OriginalError,
			TotalAttempts:  safeconv.IntToInt32(entries[i].TotalAttempts),
			FirstAttemptMs: entries[i].FirstAttempt.UnixMilli(),
			LastAttemptMs:  entries[i].LastAttempt.UnixMilli(),
			AddedAtMs:      entries[i].AddedAt.UnixMilli(),
		}
	}

	return &pb.ListDLQEntriesResponse{
		Entries: pbEntries,
	}, nil
}

// GetDLQCount returns the number of entries in a dispatcher's dead letter
// queue.
//
// Takes request (*pb.GetDLQCountRequest) which specifies the dispatcher type.
//
// Returns *pb.GetDLQCountResponse which contains the entry count.
// Returns error when the count cannot be retrieved.
func (s *DispatcherInspectorService) GetDLQCount(ctx context.Context, request *pb.GetDLQCountRequest) (*pb.GetDLQCountResponse, error) {
	count, err := s.inspector.GetDLQCount(ctx, request.GetDispatcherType())
	if err != nil {
		return nil, fmt.Errorf("getting DLQ count: %w", err)
	}

	return &pb.GetDLQCountResponse{
		Count:          safeconv.IntToInt32(count),
		DispatcherType: request.GetDispatcherType(),
	}, nil
}

// ClearDLQ removes all entries from a dispatcher's dead letter queue.
//
// Takes request (*pb.ClearDLQRequest) which specifies the
// dispatcher type to clear.
//
// Returns *pb.ClearDLQResponse which indicates success.
// Returns error when the queue cannot be cleared.
func (s *DispatcherInspectorService) ClearDLQ(ctx context.Context, request *pb.ClearDLQRequest) (*pb.ClearDLQResponse, error) {
	if err := s.inspector.ClearDLQ(ctx, request.GetDispatcherType()); err != nil {
		return nil, fmt.Errorf("clearing DLQ: %w", err)
	}

	return &pb.ClearDLQResponse{
		DispatcherType: request.GetDispatcherType(),
		Success:        true,
	}, nil
}
