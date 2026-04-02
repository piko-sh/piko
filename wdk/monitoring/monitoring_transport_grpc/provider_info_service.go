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
	"fmt"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// errMessageProviderInfoNotAvailable is the error message returned when the
// provider inspector is not configured.
const errMessageProviderInfoNotAvailable = "provider info not available"

// ProviderInfoService implements the gRPC ProviderInfoService interface.
type ProviderInfoService struct {
	pb.UnimplementedProviderInfoServiceServer

	// inspector provides access to provider information across hexagons.
	// May be nil if no resource descriptors are registered.
	inspector monitoring_domain.ProviderInfoInspector
}

// NewProviderInfoService creates a new ProviderInfoService.
//
// Takes inspector (monitoring_domain.ProviderInfoInspector) which provides
// access to provider information. May be nil for graceful degradation.
//
// Returns *ProviderInfoService which is ready for gRPC registration.
func NewProviderInfoService(inspector monitoring_domain.ProviderInfoInspector) *ProviderInfoService {
	return &ProviderInfoService{
		inspector: inspector,
	}
}

// ListResourceTypes returns all registered resource types.
//
// Returns *pb.ListResourceTypesResponse which contains the type names.
// Returns error when the request fails.
func (s *ProviderInfoService) ListResourceTypes(ctx context.Context, _ *pb.ListResourceTypesRequest) (*pb.ListResourceTypesResponse, error) {
	if s.inspector == nil {
		return &pb.ListResourceTypesResponse{}, nil
	}

	types := s.inspector.ListResourceTypes(ctx)

	return &pb.ListResourceTypesResponse{
		ResourceTypes: types,
	}, nil
}

// ListProviders returns providers for a specific resource type with dynamic
// column definitions.
//
// Takes request (*pb.ListProvidersRequest) which specifies the resource type to
// list providers for.
//
// Returns *pb.ListProvidersResponse which contains columns and provider rows.
// Returns error when the resource type is unknown.
func (s *ProviderInfoService) ListProviders(ctx context.Context, request *pb.ListProvidersRequest) (*pb.ListProvidersResponse, error) {
	if s.inspector == nil {
		return nil, errors.New(errMessageProviderInfoNotAvailable)
	}

	result, err := s.inspector.ListProviders(ctx, request.GetResourceType())
	if err != nil {
		return nil, fmt.Errorf("listing providers: %w", err)
	}

	columns := make([]*pb.ProviderColumn, len(result.Columns))
	for i, col := range result.Columns {
		columns[i] = &pb.ProviderColumn{
			Header:   col.Header,
			Key:      col.Key,
			WideOnly: col.WideOnly,
		}
	}

	rows := make([]*pb.ProviderRow, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = &pb.ProviderRow{
			Name:      row.Name,
			IsDefault: row.IsDefault,
			Values:    row.Values,
		}
	}

	return &pb.ListProvidersResponse{
		Columns: columns,
		Rows:    rows,
	}, nil
}

// DescribeProvider returns detailed information for a single provider.
//
// Takes request (*pb.DescribeProviderRequest) which specifies the resource type
// and provider name to describe.
//
// Returns *pb.DescribeProviderResponse which contains structured detail
// sections.
// Returns error when the resource type or provider is not found.
func (s *ProviderInfoService) DescribeProvider(ctx context.Context, request *pb.DescribeProviderRequest) (*pb.DescribeProviderResponse, error) {
	if s.inspector == nil {
		return nil, errors.New(errMessageProviderInfoNotAvailable)
	}

	detail, err := s.inspector.DescribeProvider(ctx, request.GetResourceType(), request.GetName())
	if err != nil {
		return nil, fmt.Errorf("describing provider: %w", err)
	}

	sections := make([]*pb.ProviderInfoSection, len(detail.Sections))
	for i, section := range detail.Sections {
		entries := make([]*pb.ProviderInfoEntry, len(section.Entries))
		for j, entry := range section.Entries {
			entries[j] = &pb.ProviderInfoEntry{
				Key:   entry.Key,
				Value: entry.Value,
			}
		}

		sections[i] = &pb.ProviderInfoSection{
			Title:   section.Title,
			Entries: entries,
		}
	}

	return &pb.DescribeProviderResponse{
		Name:     detail.Name,
		Sections: sections,
	}, nil
}

// ListSubResources returns sub-resources for a named provider.
//
// Takes request (*pb.ListSubResourcesRequest) which specifies the resource type
// and provider name to query.
//
// Returns *pb.ListSubResourcesResponse which contains columns, rows, and
// the sub-resource name.
// Returns error when the resource type or provider does not support
// sub-resources.
func (s *ProviderInfoService) ListSubResources(ctx context.Context, request *pb.ListSubResourcesRequest) (*pb.ListSubResourcesResponse, error) {
	if s.inspector == nil {
		return nil, errors.New(errMessageProviderInfoNotAvailable)
	}

	result, err := s.inspector.ListSubResources(ctx, request.GetResourceType(), request.GetProviderName())
	if err != nil {
		return nil, fmt.Errorf("listing sub-resources: %w", err)
	}

	columns := make([]*pb.ProviderColumn, len(result.Columns))
	for i, col := range result.Columns {
		columns[i] = &pb.ProviderColumn{
			Header:   col.Header,
			Key:      col.Key,
			WideOnly: col.WideOnly,
		}
	}

	rows := make([]*pb.ProviderRow, len(result.Rows))
	for i, row := range result.Rows {
		rows[i] = &pb.ProviderRow{
			Name:      row.Name,
			IsDefault: row.IsDefault,
			Values:    row.Values,
		}
	}

	return &pb.ListSubResourcesResponse{
		Columns:         columns,
		Rows:            rows,
		SubResourceName: result.SubResourceName,
	}, nil
}

// DescribeResourceType returns a service-level overview for a resource type.
//
// Takes request (*pb.DescribeResourceTypeRequest) which specifies the resource
// type to describe.
//
// Returns *pb.DescribeProviderResponse which contains the overview sections.
// Returns error when the resource type does not support type-level describe.
func (s *ProviderInfoService) DescribeResourceType(ctx context.Context, request *pb.DescribeResourceTypeRequest) (*pb.DescribeProviderResponse, error) {
	if s.inspector == nil {
		return nil, errors.New(errMessageProviderInfoNotAvailable)
	}

	detail, err := s.inspector.DescribeResourceType(ctx, request.GetResourceType())
	if err != nil {
		return nil, fmt.Errorf("describing resource type: %w", err)
	}

	sections := make([]*pb.ProviderInfoSection, len(detail.Sections))
	for i, section := range detail.Sections {
		entries := make([]*pb.ProviderInfoEntry, len(section.Entries))
		for j, entry := range section.Entries {
			entries[j] = &pb.ProviderInfoEntry{
				Key:   entry.Key,
				Value: entry.Value,
			}
		}

		sections[i] = &pb.ProviderInfoSection{
			Title:   section.Title,
			Entries: entries,
		}
	}

	return &pb.DescribeProviderResponse{
		Name:     detail.Name,
		Sections: sections,
	}, nil
}
