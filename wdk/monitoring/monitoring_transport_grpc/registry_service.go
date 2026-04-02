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
	"piko.sh/piko/internal/registry/registry_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// RegistryInspectorService implements the gRPC service for inspecting registry
// artefacts and variants.
type RegistryInspectorService struct {
	pb.UnimplementedRegistryInspectorServiceServer

	// inspector provides access to registry data for summaries and listings.
	inspector registry_domain.RegistryInspector
}

// NewRegistryInspectorService creates a new RegistryInspectorService.
//
// Takes inspector (RegistryInspector) which provides registry inspection
// capabilities.
//
// Returns *RegistryInspectorService which is ready for use as a gRPC service.
func NewRegistryInspectorService(inspector registry_domain.RegistryInspector) *RegistryInspectorService {
	return &RegistryInspectorService{
		UnimplementedRegistryInspectorServiceServer: pb.UnimplementedRegistryInspectorServiceServer{},
		inspector: inspector,
	}
}

// GetArtefactSummary returns artefact counts grouped by status.
//
// Returns *pb.GetArtefactSummaryResponse which contains the summary counts.
// Returns error when the summary cannot be retrieved from the inspector.
func (s *RegistryInspectorService) GetArtefactSummary(ctx context.Context, _ *pb.GetArtefactSummaryRequest) (*pb.GetArtefactSummaryResponse, error) {
	summaries, err := s.inspector.ListArtefactSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing artefact summary: %w", err)
	}

	pbSummaries := make([]*pb.ArtefactSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.ArtefactSummary{
			Status: sum.Status,
			Count:  sum.Count,
		}
	}

	return &pb.GetArtefactSummaryResponse{
		Summaries: pbSummaries,
	}, nil
}

// GetVariantSummary returns variant counts grouped by status.
//
// Returns *pb.GetVariantSummaryResponse which contains the variant summaries.
// Returns error when the variant summary cannot be retrieved.
func (s *RegistryInspectorService) GetVariantSummary(ctx context.Context, _ *pb.GetVariantSummaryRequest) (*pb.GetVariantSummaryResponse, error) {
	summaries, err := s.inspector.ListVariantSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing variant summary: %w", err)
	}

	pbSummaries := make([]*pb.VariantSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.VariantSummary{
			Status: sum.Status,
			Count:  sum.Count,
		}
	}

	return &pb.GetVariantSummaryResponse{
		Summaries: pbSummaries,
	}, nil
}

// ListRecentArtefacts returns the most recently updated artefacts.
//
// Takes request (*pb.ListRecentArtefactsRequest) which specifies the query limit.
//
// Returns *pb.ListRecentArtefactsResponse which contains the recent artefacts.
// Returns error when the underlying inspector fails to retrieve artefacts.
func (s *RegistryInspectorService) ListRecentArtefacts(ctx context.Context, request *pb.ListRecentArtefactsRequest) (*pb.ListRecentArtefactsResponse, error) {
	limit := request.GetLimit()
	if limit <= 0 {
		limit = defaultListLimit
	}

	artefacts, err := s.inspector.ListRecentArtefacts(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing recent artefacts: %w", err)
	}

	return &pb.ListRecentArtefactsResponse{
		Artefacts: convertArtefactsToPB(artefacts),
	}, nil
}

// WatchArtefacts streams artefact updates at the requested interval.
//
// Takes request (*pb.WatchArtefactsRequest) which specifies the polling interval.
// Takes stream (pb.RegistryInspectorService_WatchArtefactsServer) which
// receives the artefact updates.
//
// Returns error when the stream context is cancelled or sending fails.
func (s *RegistryInspectorService) WatchArtefacts(request *pb.WatchArtefactsRequest, stream pb.RegistryInspectorService_WatchArtefactsServer) error {
	return runWatchLoop(stream.Context(), request.GetIntervalMs(), func() error {
		return s.sendArtefactsUpdate(stream.Context(), stream)
	}, "artefact", nil)
}

// sendArtefactsUpdate fetches and sends a single artefact update to the
// stream.
//
// Takes stream (pb.RegistryInspectorService_WatchArtefactsServer) which
// receives the artefact update.
//
// Returns error when sending the update to the stream fails.
//
//nolint:dupl // similar structure, different gRPC types
func (s *RegistryInspectorService) sendArtefactsUpdate(ctx context.Context, stream pb.RegistryInspectorService_WatchArtefactsServer) error {
	ctx, l := logger_domain.From(ctx, log)
	summaries, err := s.inspector.ListArtefactSummary(ctx)
	if err != nil {
		l.Error("Failed to list artefact summary in WatchArtefacts", Error(err))
		return nil
	}

	artefacts, err := s.inspector.ListRecentArtefacts(ctx, defaultListLimit)
	if err != nil {
		l.Error("Failed to list recent artefacts in WatchArtefacts", Error(err))
		return nil
	}

	return stream.Send(&pb.ArtefactsUpdate{
		Summaries:       convertArtefactSummariesToPB(summaries),
		RecentArtefacts: convertArtefactsToPB(artefacts),
		TimestampMs:     time.Now().UnixMilli(),
	})
}

// convertArtefactSummariesToPB converts domain artefact summaries to protobuf
// format.
//
// Takes summaries ([]registry_domain.ArtefactSummary) which contains the domain
// artefact summaries to convert.
//
// Returns []*pb.ArtefactSummary which contains the converted protobuf summaries.
func convertArtefactSummariesToPB(summaries []registry_domain.ArtefactSummary) []*pb.ArtefactSummary {
	pbSummaries := make([]*pb.ArtefactSummary, len(summaries))
	for i, sum := range summaries {
		pbSummaries[i] = &pb.ArtefactSummary{
			Status: sum.Status,
			Count:  sum.Count,
		}
	}
	return pbSummaries
}

// convertArtefactsToPB converts domain artefacts to protobuf format.
//
// Takes artefacts ([]registry_domain.ArtefactListItem) which contains the
// domain artefacts to convert.
//
// Returns []*pb.ArtefactListItem which contains the converted protobuf items.
func convertArtefactsToPB(artefacts []registry_domain.ArtefactListItem) []*pb.ArtefactListItem {
	pbArtefacts := make([]*pb.ArtefactListItem, len(artefacts))
	for i, art := range artefacts {
		pbArtefacts[i] = &pb.ArtefactListItem{
			Id:           art.ID,
			SourcePath:   art.SourcePath,
			Status:       art.Status,
			VariantCount: art.VariantCount,
			TotalSize:    art.TotalSize,
			CreatedAt:    art.CreatedAt,
			UpdatedAt:    art.UpdatedAt,
		}
	}
	return pbArtefacts
}
