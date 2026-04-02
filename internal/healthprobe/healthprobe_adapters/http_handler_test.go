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

package healthprobe_adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type mockService struct {
	livenessStatus  healthprobe_dto.Status
	readinessStatus healthprobe_dto.Status
}

func (m *mockService) CheckLiveness(_ context.Context) healthprobe_dto.Status {
	return m.livenessStatus
}

func (m *mockService) CheckReadiness(_ context.Context) healthprobe_dto.Status {
	return m.readinessStatus
}

func TestHTTPHandlerAdapter_ServeLiveness_Healthy(t *testing.T) {
	mockService := &mockService{
		livenessStatus: healthprobe_dto.Status{
			Name:      "TestApp",
			State:     healthprobe_dto.StateHealthy,
			Message:   "All systems operational",
			Timestamp: time.Now(),
			Duration:  "5ms",
		},
	}

	handler := NewHTTPHandlerAdapter(mockService)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()

	handler.ServeLiveness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY state, got %s", response.State)
	}

	if response.Name != "TestApp" {
		t.Errorf("Expected name 'TestApp', got %s", response.Name)
	}
}

func TestHTTPHandlerAdapter_ServeLiveness_Unhealthy(t *testing.T) {
	mockService := &mockService{
		livenessStatus: healthprobe_dto.Status{
			Name:      "TestApp",
			State:     healthprobe_dto.StateUnhealthy,
			Message:   "Database connection failed",
			Timestamp: time.Now(),
			Duration:  "100ms",
		},
	}

	handler := NewHTTPHandlerAdapter(mockService)

	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()

	handler.ServeLiveness(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", recorder.Code)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response.State != healthprobe_dto.StateUnhealthy {
		t.Errorf("Expected UNHEALTHY state, got %s", response.State)
	}
}

func TestHTTPHandlerAdapter_ServeReadiness_Healthy(t *testing.T) {
	mockService := &mockService{
		readinessStatus: healthprobe_dto.Status{
			Name:      "TestApp",
			State:     healthprobe_dto.StateHealthy,
			Message:   "Ready to serve traffic",
			Timestamp: time.Now(),
			Duration:  "15ms",
			Dependencies: []*healthprobe_dto.Status{
				{
					Name:    "Database",
					State:   healthprobe_dto.StateHealthy,
					Message: "Connected",
				},
			},
		},
	}

	handler := NewHTTPHandlerAdapter(mockService)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	handler.ServeReadiness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response.State != healthprobe_dto.StateHealthy {
		t.Errorf("Expected HEALTHY state, got %s", response.State)
	}

	if len(response.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(response.Dependencies))
	}
}

func TestHTTPHandlerAdapter_ServeReadiness_Degraded(t *testing.T) {
	mockService := &mockService{
		readinessStatus: healthprobe_dto.Status{
			Name:      "TestApp",
			State:     healthprobe_dto.StateDegraded,
			Message:   "Running with reduced capacity",
			Timestamp: time.Now(),
			Duration:  "20ms",
		},
	}

	handler := NewHTTPHandlerAdapter(mockService)

	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	handler.ServeReadiness(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200 for degraded, got %d", recorder.Code)
	}

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if response.State != healthprobe_dto.StateDegraded {
		t.Errorf("Expected DEGRADED state, got %s", response.State)
	}
}

func TestHTTPHandlerAdapter_StatusCodeMapping(t *testing.T) {
	testCases := []struct {
		name               string
		state              healthprobe_dto.State
		expectedStatusCode int
	}{
		{
			name:               "Healthy returns 200",
			state:              healthprobe_dto.StateHealthy,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Degraded returns 200",
			state:              healthprobe_dto.StateDegraded,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Unhealthy returns 503",
			state:              healthprobe_dto.StateUnhealthy,
			expectedStatusCode: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := &mockService{
				livenessStatus: healthprobe_dto.Status{
					Name:      "TestApp",
					State:     tc.state,
					Message:   "Test",
					Timestamp: time.Now(),
					Duration:  "1ms",
				},
			}

			handler := NewHTTPHandlerAdapter(mockService)
			request := httptest.NewRequest(http.MethodGet, "/live", nil)
			recorder := httptest.NewRecorder()

			handler.ServeLiveness(recorder, request)

			if recorder.Code != tc.expectedStatusCode {
				t.Errorf("Expected status %d, got %d", tc.expectedStatusCode, recorder.Code)
			}
		})
	}
}

func TestHTTPHandlerAdapter_JSONEncoding(t *testing.T) {
	mockService := &mockService{
		readinessStatus: healthprobe_dto.Status{
			Name:      "TestApp",
			State:     healthprobe_dto.StateHealthy,
			Message:   "All systems operational",
			Timestamp: time.Date(2025, 11, 14, 10, 30, 0, 0, time.UTC),
			Duration:  "25ms",
			Dependencies: []*healthprobe_dto.Status{
				{
					Name:    "Database",
					State:   healthprobe_dto.StateHealthy,
					Message: "Connected",
				},
				{
					Name:    "Cache",
					State:   healthprobe_dto.StateDegraded,
					Message: "Low hit rate",
				},
			},
		},
	}

	handler := NewHTTPHandlerAdapter(mockService)
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()

	handler.ServeReadiness(recorder, request)

	var response healthprobe_dto.Status
	if err := json.ConfigStd.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if len(response.Dependencies) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(response.Dependencies))
	}

	if response.Dependencies[0].Name != "Database" {
		t.Errorf("Expected first dependency to be 'Database', got %s", response.Dependencies[0].Name)
	}

	if response.Dependencies[1].State != healthprobe_dto.StateDegraded {
		t.Errorf("Expected second dependency to be DEGRADED, got %s", response.Dependencies[1].State)
	}
}

func TestHTTPHandlerAdapter_ContextPropagation(t *testing.T) {
	contextChecked := false

	mockService := &mockService{
		livenessStatus: healthprobe_dto.Status{
			Name:      "TestApp",
			State:     healthprobe_dto.StateHealthy,
			Message:   "OK",
			Timestamp: time.Now(),
			Duration:  "1ms",
		},
	}

	wrapper := &contextCheckingService{
		inner: mockService,
		onCheck: func(ctx context.Context) {
			contextChecked = true
			if ctx == nil {
				t.Error("Context should not be nil")
			}
		},
	}

	handler := NewHTTPHandlerAdapter(wrapper)
	request := httptest.NewRequest(http.MethodGet, "/live", nil)
	recorder := httptest.NewRecorder()

	handler.ServeLiveness(recorder, request)

	if !contextChecked {
		t.Error("Context was not properly propagated to service")
	}
}

type contextCheckingService struct {
	inner   *mockService
	onCheck func(context.Context)
}

func (c *contextCheckingService) CheckLiveness(ctx context.Context) healthprobe_dto.Status {
	c.onCheck(ctx)
	return c.inner.CheckLiveness(ctx)
}

func (c *contextCheckingService) CheckReadiness(ctx context.Context) healthprobe_dto.Status {
	c.onCheck(ctx)
	return c.inner.CheckReadiness(ctx)
}
