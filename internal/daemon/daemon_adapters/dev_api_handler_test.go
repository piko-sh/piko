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

package daemon_adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

type mockSystemStatsProvider struct {
	stats monitoring_domain.SystemStats
}

func (m *mockSystemStatsProvider) GetStats() monitoring_domain.SystemStats {
	return m.stats
}

type mockOrchestratorInspector struct {
	taskSummaryErr  error
	recentTasksErr  error
	workflowErr     error
	taskSummary     []orchestrator_domain.TaskSummary
	recentTasks     []orchestrator_domain.TaskListItem
	workflowSummary []orchestrator_domain.WorkflowSummary
}

func (m *mockOrchestratorInspector) ListTaskSummary(_ context.Context) ([]orchestrator_domain.TaskSummary, error) {
	return m.taskSummary, m.taskSummaryErr
}

func (m *mockOrchestratorInspector) ListRecentTasks(_ context.Context, _ int) ([]orchestrator_domain.TaskListItem, error) {
	return m.recentTasks, m.recentTasksErr
}

func (m *mockOrchestratorInspector) ListWorkflowSummary(_ context.Context, _ int) ([]orchestrator_domain.WorkflowSummary, error) {
	return m.workflowSummary, m.workflowErr
}

func newTestRouter(handler *DevAPIHandler) *chi.Mux {
	r := chi.NewRouter()
	handler.Mount(r)
	return r
}

func TestDevAPIHandler_Stats_NilProvider(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["available"])
}

func TestDevAPIHandler_Stats_WithProvider(t *testing.T) {
	t.Parallel()

	provider := &mockSystemStatsProvider{
		stats: monitoring_domain.SystemStats{
			Memory: monitoring_domain.MemoryInfo{
				HeapAlloc: 1024 * 1024,
				HeapSys:   2048 * 1024,
			},
			NumGoroutines: 42,
			CPUMillicores: 150.5,
			UptimeMs:      60000,
			Build: monitoring_domain.BuildInfo{
				Version:   "0.1.0",
				GoVersion: "go1.24.0",
			},
		},
	}

	handler := NewDevAPIHandler(provider, nil)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body monitoring_domain.SystemStats
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, uint64(1024*1024), body.Memory.HeapAlloc)
	assert.Equal(t, int32(42), body.NumGoroutines)
	assert.InDelta(t, 150.5, body.CPUMillicores, 0.1)
	assert.Equal(t, "0.1.0", body.Build.Version)
}

func TestDevAPIHandler_Build_NilOrchestrator(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/build", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["available"])
}

func TestDevAPIHandler_Build_WithOrchestrator(t *testing.T) {
	t.Parallel()

	inspector := &mockOrchestratorInspector{
		taskSummary: []orchestrator_domain.TaskSummary{
			{Status: "completed", Count: 10},
			{Status: "failed", Count: 2},
		},
		recentTasks: []orchestrator_domain.TaskListItem{
			{Executor: "compile-component", Status: "completed"},
		},
		workflowSummary: []orchestrator_domain.WorkflowSummary{
			{WorkflowID: "full-build", TaskCount: 5},
		},
	}

	handler := NewDevAPIHandler(nil, inspector)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/build", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.NotNil(t, body["taskSummary"])
	assert.NotNil(t, body["recentTasks"])
	assert.NotNil(t, body["workflowSummary"])
}

func TestDevAPIHandler_Overview_NilDeps(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/overview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	assert.Empty(t, body)
}

func TestDevAPIHandler_Overview_WithDeps(t *testing.T) {
	t.Parallel()

	provider := &mockSystemStatsProvider{
		stats: monitoring_domain.SystemStats{
			Memory:        monitoring_domain.MemoryInfo{HeapAlloc: 512 * 1024},
			NumGoroutines: 20,
			CPUMillicores: 75.0,
			UptimeMs:      120000,
			Build: monitoring_domain.BuildInfo{
				Version:   "0.2.0",
				GoVersion: "go1.24.1",
			},
		},
	}

	inspector := &mockOrchestratorInspector{
		taskSummary: []orchestrator_domain.TaskSummary{
			{Status: "completed", Count: 5},
		},
	}

	handler := NewDevAPIHandler(provider, inspector)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/overview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)

	system, ok := body["system"].(map[string]any)
	require.True(t, ok, "expected system key in response")
	assert.Equal(t, "0.2.0", system["version"])
	assert.Equal(t, "go1.24.1", system["goVersion"])

	build, ok := body["build"].(map[string]any)
	require.True(t, ok, "expected build key in response")
	assert.NotNil(t, build["taskSummary"])
}

func TestDevAPIHandler_Build_Error_SanitisesInProduction(t *testing.T) {
	t.Parallel()

	internal := errors.New("internal database hostname leaked.example.com")
	inspector := &mockOrchestratorInspector{taskSummaryErr: internal}

	handler := NewDevAPIHandler(nil, inspector)
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/build", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["available"])
	assert.NotContains(t, body["error"], "hostname leaked")
}

func TestDevAPIHandler_Build_Error_RevealsInDevelopmentMode(t *testing.T) {
	t.Parallel()

	internal := errors.New("internal database hostname leaked.example.com")
	inspector := &mockOrchestratorInspector{taskSummaryErr: internal}

	handler := NewDevAPIHandler(nil, inspector, WithDevAPIDevelopmentMode(true))
	router := newTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/_piko/dev/api/build", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["available"])
	assert.Contains(t, body["error"], "hostname leaked")
}

func TestDevAPIHandler_Mount_RegistersRoutes(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	router := newTestRouter(handler)

	endpoints := []string{
		"/_piko/dev/api/stats",
		"/_piko/dev/api/build",
		"/_piko/dev/api/overview",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code, "endpoint %s should return 200", endpoint)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"),
			"endpoint %s should return JSON", endpoint)
	}
}

type stubHealthProbeService struct {
	liveness  monitoring_domain.HealthProbeStatus
	readiness monitoring_domain.HealthProbeStatus
}

func (s *stubHealthProbeService) CheckLiveness(context.Context) monitoring_domain.HealthProbeStatus {
	return s.liveness
}

func (s *stubHealthProbeService) CheckReadiness(context.Context) monitoring_domain.HealthProbeStatus {
	return s.readiness
}

type stubProviderInfoInspector struct {
	monitoring_domain.ProviderInfoInspector
	providers     map[string]*monitoring_domain.ProviderListResult
	subResources  map[string]*monitoring_domain.ProviderListResult
	resourceTypes []string
}

func (s *stubProviderInfoInspector) ListResourceTypes(context.Context) []string {
	return s.resourceTypes
}

func (s *stubProviderInfoInspector) ListProviders(_ context.Context, resourceType string) (*monitoring_domain.ProviderListResult, error) {
	list, ok := s.providers[resourceType]
	if !ok {
		return nil, errors.New("resource type not found")
	}
	return list, nil
}

func (*stubProviderInfoInspector) DescribeProvider(_ context.Context, _, name string) (*provider_domain.ProviderDetail, error) {
	if name == "broken" {
		return nil, errors.New("describe failed")
	}
	return &provider_domain.ProviderDetail{Name: name, Sections: nil}, nil
}

func (s *stubProviderInfoInspector) ListSubResources(_ context.Context, _, providerName string) (*monitoring_domain.ProviderListResult, error) {
	sub, ok := s.subResources[providerName]
	if !ok {
		return nil, errors.New("no sub-resources")
	}
	return sub, nil
}

func serveDevAPI(t *testing.T, handler *DevAPIHandler, path string) map[string]any {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	newTestRouter(handler).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.ConfigDefault.Unmarshal(recorder.Body.Bytes(), &body))
	return body
}

func TestDevAPIHandler_OptionalEndpointsReportUnavailableWithoutDependencies(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/_piko/dev/api/health",
		"/_piko/dev/api/resources",
		"/_piko/dev/api/providers",
		"/_piko/dev/api/memory",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			body := serveDevAPI(t, NewDevAPIHandler(nil, nil), path)

			assert.Equal(t, false, body["available"])
		})
	}
}

func TestDevAPIHandler_Health_WithProbeService(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	handler.SetHealthProbeService(&stubHealthProbeService{
		liveness:  monitoring_domain.HealthProbeStatus{Name: "liveness", State: "HEALTHY"},
		readiness: monitoring_domain.HealthProbeStatus{Name: "readiness", State: "DEGRADED"},
	})

	body := serveDevAPI(t, handler, "/_piko/dev/api/health")

	assert.Contains(t, body, "liveness")
	assert.Contains(t, body, "readiness")
	assert.NotContains(t, body, "available")
}

func TestDevAPIHandler_Resources_WithProvider(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	handler.SetResourceProvider(&monitoring_domain.MockResourceProvider{
		GetResourcesFunc: func() monitoring_domain.ResourceData {
			return monitoring_domain.ResourceData{Categories: nil, Total: 7, TimestampMs: 42}
		},
	})

	body := serveDevAPI(t, handler, "/_piko/dev/api/resources")

	assert.NotContains(t, body, "available")
	assert.Len(t, body, 3)
}

func TestDevAPIHandler_Providers_WithInspector(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(nil, nil)
	handler.SetProviderInfoInspector(&stubProviderInfoInspector{
		resourceTypes: []string{"email", "storage", "missing"},
		providers: map[string]*monitoring_domain.ProviderListResult{
			"email":   {Rows: []provider_domain.ProviderListEntry{{Name: "smtp"}, {Name: "broken"}}},
			"storage": {Rows: []provider_domain.ProviderListEntry{{Name: "disk"}}},
		},
		subResources: map[string]*monitoring_domain.ProviderListResult{
			"smtp": {Rows: []provider_domain.ProviderListEntry{{Name: "queue"}}},
			"disk": {Rows: nil},
		},
	})

	body := serveDevAPI(t, handler, "/_piko/dev/api/providers")

	assert.Equal(t, []any{"email", "storage", "missing"}, body["resourceTypes"])
	providers, ok := body["providers"].(map[string]any)
	require.True(t, ok)
	assert.NotContains(t, providers, "missing", "a resource type that cannot be listed is skipped")

	email, ok := providers["email"].(map[string]any)
	require.True(t, ok)
	details, ok := email["details"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, details, "smtp")
	assert.NotContains(t, details, "broken", "a provider that cannot be described is left out")
	subResources, ok := email["subResources"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, subResources, "smtp")

	storage, ok := providers["storage"].(map[string]any)
	require.True(t, ok)
	storageSubResources, ok := storage["subResources"].(map[string]any)
	require.True(t, ok)
	assert.Empty(t, storageSubResources, "an empty sub-resource list is left out")
}

func TestDevAPIHandler_Memory_WithProvider(t *testing.T) {
	t.Parallel()

	handler := NewDevAPIHandler(&mockSystemStatsProvider{}, nil)

	body := serveDevAPI(t, handler, "/_piko/dev/api/memory")

	for _, key := range []string{"memory", "process", "gc", "runtime", "build"} {
		assert.Contains(t, body, key)
	}
}

func TestWriteJSON_ReportsAValueThatCannotBeEncoded(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()

	writeJSON(recorder, http.StatusOK, map[string]any{"channel": make(chan int)})

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "marshal failed")
}
