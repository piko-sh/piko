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

package collection_domain

import (
	"context"
	"testing"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/wdk/clock"
)

func mustCastToCollectionServiceHealth(t *testing.T, service CollectionService) *collectionService {
	t.Helper()
	cs, ok := service.(*collectionService)
	if !ok {
		t.Fatal("expected *collectionService")
	}
	return cs
}

func TestCollectionService_Name(t *testing.T) {
	registry := newTestProviderRegistry()
	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry))

	name := service.Name()
	if name != "CollectionService" {
		t.Errorf("Expected name 'CollectionService', got %q", name)
	}
}

func TestCollectionService_Check_Liveness(t *testing.T) {
	registry := newTestProviderRegistry()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.Name != "CollectionService" {
		t.Errorf("Expected name 'CollectionService', got %q", status.Name)
	}

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected StateHealthy, got %v", status.State)
	}

	if status.Message != "Collection service is running" {
		t.Errorf("Expected message 'Collection service is running', got %q", status.Message)
	}

	if status.Timestamp != baseTime {
		t.Errorf("Expected timestamp %v, got %v", baseTime, status.Timestamp)
	}
}

func TestCollectionService_Check_Liveness_NilRegistry(t *testing.T) {

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := &collectionService{
		registry: nil,
		clock:    mockClock,
	}

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeLiveness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy for nil registry, got %v", status.State)
	}

	if status.Message != "Provider registry is not initialised" {
		t.Errorf("Expected 'Provider registry is not initialised', got %q", status.Message)
	}
}

func TestCollectionService_Check_Readiness_NoProviders(t *testing.T) {
	registry := newTestProviderRegistry()

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected StateHealthy for no providers, got %v", status.State)
	}

	if status.Message != "No collection providers configured" {
		t.Errorf("Expected 'No collection providers configured', got %q", status.Message)
	}
}

func TestCollectionService_Check_Readiness_WithProviders(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &MockCollectionProvider{
		NameFunc: func() string { return "test-provider" },
	}
	_ = registry.Register(provider)

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.Name != "CollectionService" {
		t.Errorf("Expected name 'CollectionService', got %q", status.Name)
	}

	if len(status.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(status.Dependencies))
	}

	dependency := status.Dependencies[0]
	if dependency.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected provider to be StateHealthy (no health check, skipped), got %v", dependency.State)
	}
}

func TestCollectionService_Check_Readiness_WithHealthyProvider(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &mockHealthyProvider{
		MockCollectionProvider: MockCollectionProvider{
			NameFunc: func() string { return "healthy-provider" },
		},
	}
	_ = registry.Register(provider)

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected StateHealthy, got %v", status.State)
	}

	if status.Message != "Collection service ready with 1 provider(s)" {
		t.Errorf("Expected ready message, got %q", status.Message)
	}
}

func TestCollectionService_Check_Readiness_WithUnhealthyProvider(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &mockUnhealthyProvider{
		MockCollectionProvider: MockCollectionProvider{
			NameFunc: func() string { return "unhealthy-provider" },
		},
	}
	_ = registry.Register(provider)

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy, got %v", status.State)
	}

	if status.Message != "Collection service has provider issues" {
		t.Errorf("Expected 'Collection service has provider issues', got %q", status.Message)
	}
}

func TestCollectionService_Check_Readiness_MixedProviders(t *testing.T) {
	registry := newTestProviderRegistry()

	healthy := &mockHealthyProvider{
		MockCollectionProvider: MockCollectionProvider{
			NameFunc: func() string { return "healthy" },
		},
	}
	unhealthy := &mockUnhealthyProvider{
		MockCollectionProvider: MockCollectionProvider{
			NameFunc: func() string { return "unhealthy" },
		},
	}
	_ = registry.Register(healthy)
	_ = registry.Register(unhealthy)

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy (one provider unhealthy), got %v", status.State)
	}

	if len(status.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(status.Dependencies))
	}
}

func TestCollectionService_Check_Readiness_DegradedProvider(t *testing.T) {
	registry := newTestProviderRegistry()

	provider := &mockDegradedProvider{
		MockCollectionProvider: MockCollectionProvider{
			NameFunc: func() string { return "degraded-provider" },
		},
	}
	_ = registry.Register(provider)

	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(baseTime)

	service := mustCastToCollectionServiceHealth(t, NewCollectionService(context.Background(), registry, withServiceClock(mockClock)))

	status := service.Check(context.Background(), healthprobe_dto.CheckTypeReadiness)

	if status.State != healthprobe_dto.StateDegraded {
		t.Errorf("Expected StateDegraded, got %v", status.State)
	}
}

func TestAggregateState_HealthyToHealthy(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateHealthy, healthprobe_dto.StateHealthy)
	if result != healthprobe_dto.StateHealthy {
		t.Errorf("Expected StateHealthy, got %v", result)
	}
}

func TestAggregateState_HealthyToDegraded(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateHealthy, healthprobe_dto.StateDegraded)
	if result != healthprobe_dto.StateDegraded {
		t.Errorf("Expected StateDegraded, got %v", result)
	}
}

func TestAggregateState_HealthyToUnhealthy(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateHealthy, healthprobe_dto.StateUnhealthy)
	if result != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy, got %v", result)
	}
}

func TestAggregateState_DegradedToHealthy(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateDegraded, healthprobe_dto.StateHealthy)
	if result != healthprobe_dto.StateDegraded {
		t.Errorf("Expected StateDegraded (preserve worse state), got %v", result)
	}
}

func TestAggregateState_DegradedToDegraded(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateDegraded, healthprobe_dto.StateDegraded)
	if result != healthprobe_dto.StateDegraded {
		t.Errorf("Expected StateDegraded, got %v", result)
	}
}

func TestAggregateState_DegradedToUnhealthy(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateDegraded, healthprobe_dto.StateUnhealthy)
	if result != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy, got %v", result)
	}
}

func TestAggregateState_UnhealthyToHealthy(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateUnhealthy, healthprobe_dto.StateHealthy)
	if result != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy (preserve worse state), got %v", result)
	}
}

func TestAggregateState_UnhealthyToDegraded(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateUnhealthy, healthprobe_dto.StateDegraded)
	if result != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy (preserve worse state), got %v", result)
	}
}

func TestAggregateState_UnhealthyToUnhealthy(t *testing.T) {
	result := aggregateState(healthprobe_dto.StateUnhealthy, healthprobe_dto.StateUnhealthy)
	if result != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected StateUnhealthy, got %v", result)
	}
}

func TestGetProviderStatus_NoHealthCheck(t *testing.T) {
	provider := &MockCollectionProvider{
		NameFunc: func() string { return "no-health" },
	}

	status := getProviderStatus(context.Background(), healthprobe_dto.CheckTypeReadiness, "no-health", provider)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected StateHealthy for provider without health check (skipped), got %v", status.State)
	}

	if status.Message != "Provider does not support health checks (skipped)" {
		t.Errorf("Expected health check not supported message, got %q", status.Message)
	}

	expectedName := "no-health (Provider)"
	if status.Name != expectedName {
		t.Errorf("Expected name %q, got %q", expectedName, status.Name)
	}
}

func TestGetProviderStatus_WithHealthCheck(t *testing.T) {
	provider := &mockHealthyProvider{
		MockCollectionProvider: MockCollectionProvider{
			NameFunc: func() string { return "healthy" },
		},
	}

	status := getProviderStatus(context.Background(), healthprobe_dto.CheckTypeReadiness, "healthy", provider)

	if status.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected StateHealthy, got %v", status.State)
	}

	if status.Name != "healthy" {
		t.Errorf("Expected name 'healthy', got %q", status.Name)
	}
}

type mockHealthyProvider struct {
	MockCollectionProvider
}

func (m *mockHealthyProvider) Name() string {
	return m.MockCollectionProvider.Name()
}

func (m *mockHealthyProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:    m.Name(),
		State:   healthprobe_dto.StateHealthy,
		Message: "Provider is healthy",
	}
}

type mockUnhealthyProvider struct {
	MockCollectionProvider
}

func (m *mockUnhealthyProvider) Name() string {
	return m.MockCollectionProvider.Name()
}

func (m *mockUnhealthyProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:    m.Name(),
		State:   healthprobe_dto.StateUnhealthy,
		Message: "Provider is unhealthy",
	}
}

type mockDegradedProvider struct {
	MockCollectionProvider
}

func (m *mockDegradedProvider) Name() string {
	return m.MockCollectionProvider.Name()
}

func (m *mockDegradedProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:    m.Name(),
		State:   healthprobe_dto.StateDegraded,
		Message: "Provider is degraded",
	}
}
