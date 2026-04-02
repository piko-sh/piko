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

	"piko.sh/piko/internal/registry/registry_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

func TestRegistryInspectorService_GetArtefactSummary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inspector     *mockRegistryInspector
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns summaries from inspector",
			inspector: &mockRegistryInspector{
				artefactSummaryReturn: []registry_domain.ArtefactSummary{
					{Status: "READY", Count: 15},
					{Status: "PENDING", Count: 5},
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "returns empty when no summaries",
			inspector: &mockRegistryInspector{
				artefactSummaryReturn: []registry_domain.ArtefactSummary{},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockRegistryInspector{
				artefactSummaryError: errors.New("database error"),
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRegistryInspectorService(tc.inspector)

			response, err := service.GetArtefactSummary(context.Background(), &pb.GetArtefactSummaryRequest{})

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

func TestRegistryInspectorService_GetVariantSummary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inspector     *mockRegistryInspector
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns variant summaries",
			inspector: &mockRegistryInspector{
				variantSummaryReturn: []registry_domain.VariantSummary{
					{Status: "READY", Count: 100},
					{Status: "STALE", Count: 10},
				},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockRegistryInspector{
				variantSummaryError: errors.New("database error"),
			},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRegistryInspectorService(tc.inspector)

			response, err := service.GetVariantSummary(context.Background(), &pb.GetVariantSummaryRequest{})

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

func TestRegistryInspectorService_ListRecentArtefacts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		inspector     *mockRegistryInspector
		request       *pb.ListRecentArtefactsRequest
		name          string
		expectedCount int
		expectError   bool
	}{
		{
			name: "returns artefacts from inspector",
			inspector: &mockRegistryInspector{
				recentArtefactsReturn: []registry_domain.ArtefactListItem{
					{
						ID:           "art-1",
						SourcePath:   "/images/photo.jpg",
						Status:       "READY",
						VariantCount: 5,
						TotalSize:    1024000,
					},
					{
						ID:           "art-2",
						SourcePath:   "/images/logo.png",
						Status:       "PENDING",
						VariantCount: 0,
						TotalSize:    0,
					},
				},
			},
			request:       &pb.ListRecentArtefactsRequest{Limit: 10},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "uses default limit when not specified",
			inspector: &mockRegistryInspector{
				recentArtefactsReturn: []registry_domain.ArtefactListItem{
					{ID: "art-1"},
				},
			},
			request:       &pb.ListRecentArtefactsRequest{Limit: 0},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "propagates error",
			inspector: &mockRegistryInspector{
				recentArtefactsError: errors.New("database error"),
			},
			request:       &pb.ListRecentArtefactsRequest{},
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewRegistryInspectorService(tc.inspector)

			response, err := service.ListRecentArtefacts(context.Background(), tc.request)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(response.Artefacts) != tc.expectedCount {
				t.Errorf("expected %d artefacts, got %d", tc.expectedCount, len(response.Artefacts))
			}
		})
	}
}

func TestConvertArtefactsToPB(t *testing.T) {
	t.Parallel()

	artefacts := []registry_domain.ArtefactListItem{
		{
			ID:           "art-123",
			SourcePath:   "/images/test.jpg",
			Status:       "READY",
			VariantCount: 3,
			TotalSize:    512000,
			CreatedAt:    1000,
			UpdatedAt:    2000,
		},
	}

	result := convertArtefactsToPB(artefacts)

	if len(result) != 1 {
		t.Fatalf("expected 1 artefact, got %d", len(result))
	}

	art := result[0]
	if art.Id != "art-123" {
		t.Errorf("expected ID art-123, got %s", art.Id)
	}
	if art.SourcePath != "/images/test.jpg" {
		t.Errorf("expected SourcePath /images/test.jpg, got %s", art.SourcePath)
	}
	if art.Status != "READY" {
		t.Errorf("expected Status READY, got %s", art.Status)
	}
	if art.VariantCount != 3 {
		t.Errorf("expected VariantCount 3, got %d", art.VariantCount)
	}
	if art.TotalSize != 512000 {
		t.Errorf("expected TotalSize 512000, got %d", art.TotalSize)
	}
}

func TestConvertArtefactSummariesToPB(t *testing.T) {
	t.Parallel()

	summaries := []registry_domain.ArtefactSummary{
		{Status: "READY", Count: 100},
		{Status: "PENDING", Count: 25},
	}

	result := convertArtefactSummariesToPB(summaries)

	if len(result) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(result))
	}

	if result[0].Status != "READY" || result[0].Count != 100 {
		t.Errorf("first summary mismatch: %+v", result[0])
	}
	if result[1].Status != "PENDING" || result[1].Count != 25 {
		t.Errorf("second summary mismatch: %+v", result[1])
	}
}
