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

package analytics_domain

import (
	"context"
	"testing"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

func TestService_HealthProbe_Name(t *testing.T) {
	svc := NewService(nil)
	if svc.Name() != "AnalyticsService" {
		t.Errorf("Name() = %q, want AnalyticsService", svc.Name())
	}
}

func TestService_HealthProbe_Liveness_Running(t *testing.T) {
	mc := newMockCollector("test")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("State = %q, want HEALTHY", status.State)
	}
	if status.Message != "analytics service operational" {
		t.Errorf("Message = %q, want analytics service operational", status.Message)
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestService_HealthProbe_Liveness_Stopped(t *testing.T) {
	mc := newMockCollector("test")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())
	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("State = %q, want UNHEALTHY", status.State)
	}
}

func TestService_HealthProbe_Liveness_NoCollectors(t *testing.T) {
	svc := NewService(nil)

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.State != healthprobe_dto.StateDegraded {
		t.Errorf("State = %q, want DEGRADED", status.State)
	}
	if status.Message != "no analytics collectors configured" {
		t.Errorf("Message = %q, want no analytics collectors configured", status.Message)
	}
}

func TestService_HealthProbe_Readiness_Healthy(t *testing.T) {
	mc := newMockCollector("test")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("State = %q, want HEALTHY", status.State)
	}
	if len(status.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(status.Dependencies))
	}
	if status.Dependencies[0].Name != "test" {
		t.Errorf("Dependency name = %q, want test", status.Dependencies[0].Name)
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestService_HealthProbe_Readiness_Stopped(t *testing.T) {
	mc := newMockCollector("test")
	svc := NewService([]Collector{mc})
	svc.Start(context.Background())
	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("State = %q, want UNHEALTHY", status.State)
	}
}

func TestService_HealthProbe_Readiness_DependenciesSorted(t *testing.T) {
	zebra := newMockCollector("zebra")
	alpha := newMockCollector("alpha")
	svc := NewService([]Collector{zebra, alpha})
	svc.Start(context.Background())

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if len(status.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(status.Dependencies))
	}
	if status.Dependencies[0].Name != "alpha" {
		t.Errorf("first dependency = %q, want alpha", status.Dependencies[0].Name)
	}
	if status.Dependencies[1].Name != "zebra" {
		t.Errorf("second dependency = %q, want zebra", status.Dependencies[1].Name)
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestService_HealthProbe_Readiness_SmallChannel(t *testing.T) {
	mc := newMockCollector("test")
	svc := NewService([]Collector{mc}, WithChannelBufferSize(1))
	svc.Start(context.Background())

	status := svc.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("empty channel with capacity 1: State = %q, want HEALTHY", status.State)
	}

	if err := svc.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
