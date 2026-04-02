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

package healthprobe_domain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type mockRegistry struct {
	probes []Probe
}

func (m *mockRegistry) Register(probe Probe) {
	m.probes = append(m.probes, probe)
}

func (m *mockRegistry) GetAll() []Probe {
	return m.probes
}

type mockProbe struct {
	name       string
	state      healthprobe_dto.State
	message    string
	checkDelay time.Duration
}

func (m *mockProbe) Name() string {
	return m.name
}

func (m *mockProbe) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	if m.checkDelay > 0 {
		select {
		case <-time.After(m.checkDelay):
		case <-ctx.Done():
			return healthprobe_dto.Status{
				Name:      m.name,
				State:     healthprobe_dto.StateUnhealthy,
				Message:   "Context cancelled",
				Timestamp: time.Now(),
				Duration:  "0ms",
			}
		}
	}

	return healthprobe_dto.Status{
		Name:      m.name,
		State:     m.state,
		Message:   m.message,
		Timestamp: time.Now(),
		Duration:  m.checkDelay.String(),
	}
}

func TestNewService(t *testing.T) {
	registry := &mockRegistry{}
	timeout := 5 * time.Second
	appName := "TestApp"

	service := NewService(registry, timeout, appName)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	var _ Service = service
}

func TestService_CheckLiveness_AllHealthy(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "Probe1",
		state:   healthprobe_dto.StateHealthy,
		message: "All good",
	})
	registry.Register(&mockProbe{
		name:    "Probe2",
		state:   healthprobe_dto.StateHealthy,
		message: "Everything OK",
	})

	service := NewService(registry, 5*time.Second, "TestApp")
	ctx := context.Background()

	result := service.CheckLiveness(ctx)

	if result.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY, got %s", result.State)
	}

	if result.Name != "TestApp" {
		t.Errorf("Expected name 'TestApp', got %s", result.Name)
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
	}
}

func TestService_CheckLiveness_OneUnhealthy(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "HealthyProbe",
		state:   healthprobe_dto.StateHealthy,
		message: "All good",
	})
	registry.Register(&mockProbe{
		name:    "UnhealthyProbe",
		state:   healthprobe_dto.StateUnhealthy,
		message: "Database down",
	})

	service := NewService(registry, 5*time.Second, "TestApp")
	ctx := context.Background()

	result := service.CheckLiveness(ctx)

	if result.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected UNHEALTHY, got %s", result.State)
	}

	if len(result.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result.Dependencies))
	}
}

func TestService_CheckReadiness_Degraded(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "HealthyProbe",
		state:   healthprobe_dto.StateHealthy,
		message: "All good",
	})
	registry.Register(&mockProbe{
		name:    "DegradedProbe",
		state:   healthprobe_dto.StateDegraded,
		message: "Low disk space",
	})

	service := NewService(registry, 5*time.Second, "TestApp")
	ctx := context.Background()

	result := service.CheckReadiness(ctx)

	if result.State != healthprobe_dto.StateDegraded {
		t.Errorf("Expected DEGRADED, got %s", result.State)
	}

	if !contains(result.Message, "issues") {
		t.Errorf("Expected message to contain 'issues', got: %s", result.Message)
	}
}

func TestService_StateAggregation(t *testing.T) {
	testCases := []struct {
		name          string
		expectedState healthprobe_dto.State
		probeStates   []healthprobe_dto.State
	}{
		{
			name:          "All healthy",
			probeStates:   []healthprobe_dto.State{healthprobe_dto.StateHealthy, healthprobe_dto.StateHealthy},
			expectedState: healthprobe_dto.StateHealthy,
		},
		{
			name:          "One degraded",
			probeStates:   []healthprobe_dto.State{healthprobe_dto.StateHealthy, healthprobe_dto.StateDegraded},
			expectedState: healthprobe_dto.StateDegraded,
		},
		{
			name:          "One unhealthy overrides degraded",
			probeStates:   []healthprobe_dto.State{healthprobe_dto.StateDegraded, healthprobe_dto.StateUnhealthy},
			expectedState: healthprobe_dto.StateUnhealthy,
		},
		{
			name:          "Multiple unhealthy",
			probeStates:   []healthprobe_dto.State{healthprobe_dto.StateUnhealthy, healthprobe_dto.StateUnhealthy},
			expectedState: healthprobe_dto.StateUnhealthy,
		},
		{
			name:          "Empty probes list",
			probeStates:   []healthprobe_dto.State{},
			expectedState: healthprobe_dto.StateHealthy,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := &mockRegistry{}

			for i, state := range tc.probeStates {
				registry.Register(&mockProbe{
					name:    "Probe" + string(rune('A'+i)),
					state:   state,
					message: "Test probe",
				})
			}

			service := NewService(registry, 5*time.Second, "TestApp")
			result := service.CheckLiveness(context.Background())

			if result.State != tc.expectedState {
				t.Errorf("Expected state %s, got %s", tc.expectedState, result.State)
			}
		})
	}
}

func TestService_ProbeTimeout(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:       "SlowProbe",
		checkDelay: 10 * time.Second,
		state:      healthprobe_dto.StateHealthy,
		message:    "Should timeout",
	})

	service := NewService(registry, 100*time.Millisecond, "TestApp")
	ctx := context.Background()

	result := service.CheckLiveness(ctx)

	if result.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected UNHEALTHY due to timeout, got %s", result.State)
	}

	if len(result.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(result.Dependencies))
	}

	if result.Dependencies[0].State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected probe to be UNHEALTHY, got %s", result.Dependencies[0].State)
	}
}

func TestService_ConcurrentExecution(t *testing.T) {
	registry := &mockRegistry{}

	for i := range 5 {
		registry.Register(&mockProbe{
			name:       "Probe" + string(rune('A'+i)),
			checkDelay: 100 * time.Millisecond,
			state:      healthprobe_dto.StateHealthy,
			message:    "OK",
		})
	}

	service := NewService(registry, 5*time.Second, "TestApp")
	ctx := context.Background()

	result := service.CheckLiveness(ctx)

	if result.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY, got %s", result.State)
	}

	if len(result.Dependencies) != 5 {
		t.Errorf("Expected 5 dependencies, got %d", len(result.Dependencies))
	}
}

func TestService_ContextCancellation(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:       "SlowProbe",
		checkDelay: 5 * time.Second,
		state:      healthprobe_dto.StateHealthy,
		message:    "OK",
	})

	service := NewService(registry, 10*time.Second, "TestApp")

	ctx, cancel := context.WithTimeoutCause(context.Background(), 100*time.Millisecond, fmt.Errorf("test: simulating short context deadline for cancellation"))
	defer cancel()

	result := service.CheckLiveness(ctx)

	if result.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected UNHEALTHY after cancellation, got %s", result.State)
	}
}

func TestService_EmptyRegistry(t *testing.T) {
	registry := &mockRegistry{}
	service := NewService(registry, 5*time.Second, "TestApp")
	ctx := context.Background()

	result := service.CheckLiveness(ctx)

	if result.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY with no probes, got %s", result.State)
	}

	if len(result.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(result.Dependencies))
	}
}

func TestService_LivenessVsReadiness(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "TestProbe",
		state:   healthprobe_dto.StateHealthy,
		message: "OK",
	})

	service := NewService(registry, 5*time.Second, "TestApp")
	ctx := context.Background()

	liveness := service.CheckLiveness(ctx)
	readiness := service.CheckReadiness(ctx)

	if liveness.Name != readiness.Name {
		t.Error("Liveness and readiness should have same application name")
	}

	if liveness.State != healthprobe_dto.StateHealthy {
		t.Errorf("Liveness should be HEALTHY, got %s", liveness.State)
	}

	if readiness.State != healthprobe_dto.StateHealthy {
		t.Errorf("Readiness should be HEALTHY, got %s", readiness.State)
	}
}

func TestService_CheckReadiness_ReturnsUnhealthy_WhenDraining(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "DB",
		state:   healthprobe_dto.StateHealthy,
		message: "OK",
	})

	service := NewService(registry, 5*time.Second, "TestApp")

	drainable, ok := service.(interface{ SignalDrain() })
	if !ok {
		t.Fatal("service does not implement SignalDrain")
	}
	drainable.SignalDrain()

	result := service.CheckReadiness(context.Background())

	if result.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("expected UNHEALTHY after drain, got %s", result.State)
	}
	if result.Message != "Application is shutting down" {
		t.Errorf("expected shutdown message, got %q", result.Message)
	}
	if result.Name != "TestApp" {
		t.Errorf("expected name TestApp, got %s", result.Name)
	}
	if result.Duration != "0s" {
		t.Errorf("expected duration 0s, got %s", result.Duration)
	}
	if len(result.Dependencies) != 0 {
		t.Errorf("expected no dependencies when draining, got %d", len(result.Dependencies))
	}
}

func TestService_CheckLiveness_Unaffected_WhenDraining(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "DB",
		state:   healthprobe_dto.StateHealthy,
		message: "OK",
	})

	service := NewService(registry, 5*time.Second, "TestApp")

	drainable, ok := service.(interface{ SignalDrain() })
	if !ok {
		t.Fatal("service does not implement SignalDrain")
	}
	drainable.SignalDrain()

	result := service.CheckLiveness(context.Background())

	if result.State != healthprobe_dto.StateHealthy {
		t.Errorf("expected liveness to remain HEALTHY after drain, got %s", result.State)
	}
	if len(result.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(result.Dependencies))
	}
}

func TestService_SignalDrain_ConcurrentSafety(t *testing.T) {
	registry := &mockRegistry{}
	registry.Register(&mockProbe{
		name:    "DB",
		state:   healthprobe_dto.StateHealthy,
		message: "OK",
	})

	service := NewService(registry, 5*time.Second, "TestApp")

	drainable, ok := service.(interface{ SignalDrain() })
	if !ok {
		t.Fatal("service does not implement SignalDrain")
	}

	done := make(chan struct{})
	for range 10 {
		go func() {
			drainable.SignalDrain()
			_ = service.CheckReadiness(context.Background())
			_ = service.CheckLiveness(context.Background())
			done <- struct{}{}
		}()
	}

	for range 10 {
		<-done
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
