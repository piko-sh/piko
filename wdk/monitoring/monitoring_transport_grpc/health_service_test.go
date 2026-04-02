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
	"testing"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockHealthProbeService struct {
	liveness  monitoring_domain.HealthProbeStatus
	readiness monitoring_domain.HealthProbeStatus
}

func (m *mockHealthProbeService) CheckLiveness(_ context.Context) monitoring_domain.HealthProbeStatus {
	return m.liveness
}

func (m *mockHealthProbeService) CheckReadiness(_ context.Context) monitoring_domain.HealthProbeStatus {
	return m.readiness
}

func TestHealthService_GetHealth_WithProbeService(t *testing.T) {
	t.Parallel()

	mock := &mockHealthProbeService{
		liveness: monitoring_domain.HealthProbeStatus{
			Name:      "liveness",
			State:     "HEALTHY",
			Message:   "",
			Timestamp: 1000,
			Duration:  "1ms",
			Dependencies: []monitoring_domain.HealthProbeStatus{
				{
					Name:      "database",
					State:     "HEALTHY",
					Timestamp: 1000,
					Duration:  "500µs",
				},
			},
		},
		readiness: monitoring_domain.HealthProbeStatus{
			Name:      "readiness",
			State:     "HEALTHY",
			Message:   "",
			Timestamp: 1000,
			Duration:  "2ms",
		},
	}

	service := NewHealthService(mock)

	response, err := service.GetHealth(context.Background(), &pb.GetHealthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Liveness == nil {
		t.Fatal("expected liveness status")
	}
	if response.Liveness.State != "HEALTHY" {
		t.Errorf("expected liveness state HEALTHY, got %s", response.Liveness.State)
	}
	if response.Liveness.Name != "liveness" {
		t.Errorf("expected liveness name 'liveness', got %s", response.Liveness.Name)
	}
	if len(response.Liveness.Dependencies) != 1 {
		t.Errorf("expected 1 liveness dependency, got %d", len(response.Liveness.Dependencies))
	}

	if response.Readiness == nil {
		t.Fatal("expected readiness status")
	}
	if response.Readiness.State != "HEALTHY" {
		t.Errorf("expected readiness state HEALTHY, got %s", response.Readiness.State)
	}

	if response.TimestampMs <= 0 {
		t.Error("expected positive timestamp")
	}
}

func TestHealthService_GetHealth_NilProbeService(t *testing.T) {
	t.Parallel()

	service := NewHealthService(nil)

	response, err := service.GetHealth(context.Background(), &pb.GetHealthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Liveness == nil {
		t.Fatal("expected liveness status even with nil probe service")
	}
	if response.Liveness.State != "HEALTHY" {
		t.Errorf("expected liveness state HEALTHY, got %s", response.Liveness.State)
	}
	if response.Liveness.Name != "piko" {
		t.Errorf("expected liveness name 'piko', got %s", response.Liveness.Name)
	}

	if response.Readiness == nil {
		t.Fatal("expected readiness status even with nil probe service")
	}
	if response.Readiness.State != "HEALTHY" {
		t.Errorf("expected readiness state HEALTHY, got %s", response.Readiness.State)
	}

	if response.TimestampMs <= 0 {
		t.Error("expected positive timestamp")
	}
}
