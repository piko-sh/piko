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

package healthprobe_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/healthprobe/healthprobe_adapters"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type alwaysHealthyProbe struct{}

func (p *alwaysHealthyProbe) Name() string {
	return "AlwaysHealthy"
}

func (p *alwaysHealthyProbe) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Always healthy",
		Timestamp: time.Now(),
		Duration:  "1ms",
	}
}

type conditionalProbe struct {
	livenessState  healthprobe_dto.State
	readinessState healthprobe_dto.State
}

func (p *conditionalProbe) Name() string {
	return "ConditionalProbe"
}

func (p *conditionalProbe) Check(_ context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
	state := p.livenessState
	message := "Liveness check"

	if checkType == healthprobe_dto.CheckTypeReadiness {
		state = p.readinessState
		message = "Readiness check"
	}

	return healthprobe_dto.Status{
		Name:      p.Name(),
		State:     state,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  "2ms",
	}
}

type slowProbe struct {
	delay time.Duration
}

func (p *slowProbe) Name() string {
	return "SlowProbe"
}

func (p *slowProbe) Check(ctx context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	select {
	case <-time.After(p.delay):
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateHealthy,
			Message:   "Completed",
			Timestamp: time.Now(),
			Duration:  p.delay.String(),
		}
	case <-ctx.Done():
		return healthprobe_dto.Status{
			Name:      p.Name(),
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Timed out",
			Timestamp: time.Now(),
			Duration:  "0ms",
		}
	}
}

type compositeProbe struct {
	name         string
	dependencies []*healthprobe_dto.Status
}

func (p *compositeProbe) Name() string {
	return p.name
}

func (p *compositeProbe) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	overallState := healthprobe_dto.StateHealthy
	for _, dependency := range p.dependencies {
		if dependency.State == healthprobe_dto.StateUnhealthy {
			overallState = healthprobe_dto.StateUnhealthy
			break
		} else if dependency.State == healthprobe_dto.StateDegraded {
			overallState = healthprobe_dto.StateDegraded
		}
	}

	return healthprobe_dto.Status{
		Name:         p.name,
		State:        overallState,
		Message:      "Composite check",
		Timestamp:    time.Now(),
		Duration:     "5ms",
		Dependencies: p.dependencies,
	}
}

func TestHealthProbeSystem_EndToEnd_AllHealthy(t *testing.T) {

	registry := healthprobe_adapters.NewInMemoryRegistry()
	registry.Register(&alwaysHealthyProbe{})
	registry.Register(&conditionalProbe{
		livenessState:  healthprobe_dto.StateHealthy,
		readinessState: healthprobe_dto.StateHealthy,
	})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "IntegrationTestApp")

	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()
	handler.ServeLiveness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Liveness check failed: status %d", recorder.Code)
	}

	var livenessResponse healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&livenessResponse); err != nil {
		t.Fatalf("Failed to decode liveness response: %v", err)
	}

	if livenessResponse.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY, got %s", livenessResponse.State)
	}

	if len(livenessResponse.Dependencies) != 2 {
		t.Errorf("Expected 2 probes, got %d", len(livenessResponse.Dependencies))
	}

	request = httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder = httptest.NewRecorder()
	handler.ServeReadiness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Readiness check failed: status %d", recorder.Code)
	}
}

func TestHealthProbeSystem_EndToEnd_OneUnhealthy(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()
	registry.Register(&alwaysHealthyProbe{})
	registry.Register(&conditionalProbe{
		livenessState:  healthprobe_dto.StateHealthy,
		readinessState: healthprobe_dto.StateUnhealthy,
	})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "IntegrationTestApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()
	handler.ServeLiveness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Liveness should be healthy, got status %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder = httptest.NewRecorder()
	handler.ServeReadiness(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Readiness should be unhealthy (503), got status %d", recorder.Code)
	}

	var readinessResponse healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&readinessResponse); err != nil {
		t.Fatalf("Failed to decode readiness response: %v", err)
	}

	if readinessResponse.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected UNHEALTHY, got %s", readinessResponse.State)
	}
}

func TestHealthProbeSystem_TimeoutHandling(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()

	registry.Register(&slowProbe{
		delay: 10 * time.Second,
	})

	service := healthprobe_domain.NewService(registry, 100*time.Millisecond, "TimeoutTestApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()

	startTime := time.Now()
	handler.ServeLiveness(recorder, request)
	elapsed := time.Since(startTime)

	if elapsed > 2*time.Second {
		t.Errorf("Health check took too long: %v", elapsed)
	}

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 due to timeout, got %d", recorder.Code)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected UNHEALTHY due to timeout, got %s", response.State)
	}
}

func TestHealthProbeSystem_NestedDependencies(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()

	registry.Register(&compositeProbe{
		name: "ServiceA",
		dependencies: []*healthprobe_dto.Status{
			{
				Name:    "DatabaseA",
				State:   healthprobe_dto.StateHealthy,
				Message: "Connected",
			},
			{
				Name:    "CacheA",
				State:   healthprobe_dto.StateDegraded,
				Message: "Low memory",
			},
		},
	})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "NestedTestApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()
	handler.ServeReadiness(recorder, request)

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.State != healthprobe_dto.StateDegraded {
		t.Errorf("Expected DEGRADED, got %s", response.State)
	}

	if len(response.Dependencies) != 1 {
		t.Fatalf("Expected 1 top-level dependency, got %d", len(response.Dependencies))
	}

	serviceADeps := response.Dependencies[0].Dependencies
	if len(serviceADeps) != 2 {
		t.Errorf("Expected ServiceA to have 2 nested dependencies, got %d", len(serviceADeps))
	}
}

func TestHealthProbeSystem_NoProbes(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()
	service := healthprobe_domain.NewService(registry, 5*time.Second, "EmptyTestApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()
	handler.ServeLiveness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected 200 with no probes, got %d", recorder.Code)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY with no probes, got %s", response.State)
	}

	if len(response.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(response.Dependencies))
	}
}

func TestHealthProbeSystem_ConcurrentChecks(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()

	for range 10 {
		registry.Register(&slowProbe{
			delay: 100 * time.Millisecond,
		})
	}

	service := healthprobe_domain.NewService(registry, 5*time.Second, "ConcurrentTestApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	startTime := time.Now()
	handler.ServeReadiness(recorder, request)
	elapsed := time.Since(startTime)

	if elapsed > 500*time.Millisecond {
		t.Errorf("Probes did not execute concurrently, took %v", elapsed)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Dependencies) != 10 {
		t.Errorf("Expected 10 probe results, got %d", len(response.Dependencies))
	}
}

func TestHealthProbeSystem_MultipleHTTPRequests(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()
	registry.Register(&alwaysHealthyProbe{})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "MultiRequestTestApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	const requestCount = 50
	results := make(chan int, requestCount)

	for range requestCount {
		go func() {
			request := httptest.NewRequest(http.MethodGet, "/live", nil)
			recorder := httptest.NewRecorder()
			handler.ServeLiveness(recorder, request)
			results <- recorder.Code
		}()
	}

	for i := range requestCount {
		code := <-results
		if code != http.StatusOK {
			t.Errorf("Request %d failed with status %d", i, code)
		}
	}
}

func TestHealthProbeSystem_RealWorldScenario(t *testing.T) {

	registry := healthprobe_adapters.NewInMemoryRegistry()

	registry.Register(&compositeProbe{
		name: "DatabaseService",
		dependencies: []*healthprobe_dto.Status{
			{Name: "PostgreSQL", State: healthprobe_dto.StateHealthy, Message: "Connected"},
			{Name: "ConnectionPool", State: healthprobe_dto.StateHealthy, Message: "10/50 connections"},
		},
	})

	registry.Register(&compositeProbe{
		name: "CacheService",
		dependencies: []*healthprobe_dto.Status{
			{Name: "Redis", State: healthprobe_dto.StateDegraded, Message: "Low memory"},
		},
	})

	registry.Register(&compositeProbe{
		name: "StorageService",
		dependencies: []*healthprobe_dto.Status{
			{Name: "DiskStorage", State: healthprobe_dto.StateHealthy, Message: "5TB available"},
			{Name: "S3Storage", State: healthprobe_dto.StateHealthy, Message: "Connected"},
		},
	})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "MyApplication")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()
	handler.ServeReadiness(recorder, request)

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.State != healthprobe_dto.StateDegraded {
		t.Errorf("Expected DEGRADED due to cache, got %s", response.State)
	}

	if len(response.Dependencies) != 3 {
		t.Errorf("Expected 3 services, got %d", len(response.Dependencies))
	}

	for _, dependency := range response.Dependencies {
		if dependency.Name == "DatabaseService" && len(dependency.Dependencies) != 2 {
			t.Errorf("DatabaseService should have 2 dependencies, got %d", len(dependency.Dependencies))
		}
		if dependency.Name == "CacheService" && len(dependency.Dependencies) != 1 {
			t.Errorf("CacheService should have 1 dependency, got %d", len(dependency.Dependencies))
		}
		if dependency.Name == "StorageService" && len(dependency.Dependencies) != 2 {
			t.Errorf("StorageService should have 2 dependencies, got %d", len(dependency.Dependencies))
		}
	}
}

func TestHealthProbeSystem_DynamicProbeRegistration(t *testing.T) {
	registry := healthprobe_adapters.NewInMemoryRegistry()

	registry.Register(&alwaysHealthyProbe{})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "DynamicTestApp")

	ctx := context.Background()
	result1 := service.CheckLiveness(ctx)

	if len(result1.Dependencies) != 1 {
		t.Errorf("Expected 1 probe initially, got %d", len(result1.Dependencies))
	}

	registry.Register(&conditionalProbe{
		livenessState:  healthprobe_dto.StateHealthy,
		readinessState: healthprobe_dto.StateHealthy,
	})

	result2 := service.CheckLiveness(ctx)

	if len(result2.Dependencies) != 2 {
		t.Errorf("Expected 2 probes after registration, got %d", len(result2.Dependencies))
	}
}

func BenchmarkHealthCheck_SmallSystem(b *testing.B) {
	registry := healthprobe_adapters.NewInMemoryRegistry()

	for range 5 {
		registry.Register(&alwaysHealthyProbe{})
	}

	service := healthprobe_domain.NewService(registry, 5*time.Second, "BenchApp")
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_ = service.CheckLiveness(ctx)
	}
}

func BenchmarkHealthCheck_LargeSystem(b *testing.B) {
	registry := healthprobe_adapters.NewInMemoryRegistry()

	for range 50 {
		registry.Register(&alwaysHealthyProbe{})
	}

	service := healthprobe_domain.NewService(registry, 5*time.Second, "BenchApp")
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_ = service.CheckLiveness(ctx)
	}
}

func BenchmarkHTTPHandler_Liveness(b *testing.B) {
	registry := healthprobe_adapters.NewInMemoryRegistry()
	registry.Register(&alwaysHealthyProbe{})

	service := healthprobe_domain.NewService(registry, 5*time.Second, "BenchApp")
	handler := healthprobe_adapters.NewHTTPHandlerAdapter(service)

	b.ResetTimer()
	for b.Loop() {
		request := httptest.NewRequest(http.MethodGet, "/live", nil)
		recorder := httptest.NewRecorder()
		handler.ServeLiveness(recorder, request)
	}
}
