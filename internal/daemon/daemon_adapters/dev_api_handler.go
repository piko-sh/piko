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
	"net/http"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// DevAPIHandler serves JSON REST endpoints for the dev tools overlay widget.
// These endpoints provide system stats, build pipeline info, health probes,
// file descriptor data, and provider info at /_piko/dev/api/*.
type DevAPIHandler struct {
	// systemStats provides system metrics for the stats and overview endpoints.
	systemStats monitoring_domain.SystemStatsProvider

	// orchestrator provides build pipeline state for the build and overview endpoints.
	orchestrator orchestrator_domain.OrchestratorInspector

	// healthProbe provides liveness and readiness probe results.
	healthProbe monitoring_domain.HealthProbeService

	// resources provides file descriptor and resource data.
	resources monitoring_domain.ResourceProvider

	// providerInfo provides resource type and provider discovery data.
	providerInfo monitoring_domain.ProviderInfoInspector
}

// NewDevAPIHandler creates a new handler for dev tool REST endpoints.
// All parameters may be nil; handlers degrade gracefully when their
// data source is unavailable.
//
// Takes systemStats (monitoring_domain.SystemStatsProvider)
// which supplies system metrics.
// Takes orchestrator
// (orchestrator_domain.OrchestratorInspector) which supplies
// build pipeline state.
//
// Returns *DevAPIHandler which is the initialised handler.
func NewDevAPIHandler(
	systemStats monitoring_domain.SystemStatsProvider,
	orchestrator orchestrator_domain.OrchestratorInspector,
) *DevAPIHandler {
	return &DevAPIHandler{
		systemStats:  systemStats,
		orchestrator: orchestrator,
	}
}

// SetHealthProbeService sets the health probe provider. Must be called before
// the handler starts serving.
//
// Takes p (monitoring_domain.HealthProbeService) which
// supplies liveness and readiness probes.
func (h *DevAPIHandler) SetHealthProbeService(p monitoring_domain.HealthProbeService) {
	h.healthProbe = p
}

// SetResourceProvider sets the file descriptor / resource provider.
//
// Takes p (monitoring_domain.ResourceProvider) which supplies
// resource and file descriptor data.
func (h *DevAPIHandler) SetResourceProvider(p monitoring_domain.ResourceProvider) {
	h.resources = p
}

// SetProviderInfoInspector sets the provider info inspector for resource
// discovery.
//
// Takes p (monitoring_domain.ProviderInfoInspector) which
// supplies resource type and provider discovery.
func (h *DevAPIHandler) SetProviderInfoInspector(p monitoring_domain.ProviderInfoInspector) {
	h.providerInfo = p
}

// Mount registers the dev API routes on the given router.
//
// Takes router (chi.Router) which is the router to register endpoints on.
func (h *DevAPIHandler) Mount(router chi.Router) {
	router.Get("/_piko/dev/api/stats", h.handleStats)
	router.Get("/_piko/dev/api/build", h.handleBuild)
	router.Get("/_piko/dev/api/overview", h.handleOverview)
	router.Get("/_piko/dev/api/health", h.handleHealth)
	router.Get("/_piko/dev/api/resources", h.handleResources)
	router.Get("/_piko/dev/api/providers", h.handleProviders)
	router.Get("/_piko/dev/api/memory", h.handleMemory)
}

// handleStats returns current system statistics (memory, CPU, goroutines, etc).
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
func (h *DevAPIHandler) handleStats(w http.ResponseWriter, _ *http.Request) {
	if h.systemStats == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}
	writeJSON(w, http.StatusOK, h.systemStats.GetStats())
}

// handleBuild returns build pipeline information from the orchestrator.
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
// Takes r (*http.Request) which provides the request context.
func (h *DevAPIHandler) handleBuild(w http.ResponseWriter, r *http.Request) {
	if h.orchestrator == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	ctx := r.Context()
	summary, err := h.orchestrator.ListTaskSummary(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "error": err.Error()})
		return
	}

	recentTasks, _ := h.orchestrator.ListRecentTasks(ctx, devSSERecentTaskLimit)
	workflows, _ := h.orchestrator.ListWorkflowSummary(ctx, devSSEWorkflowSummaryLimit)

	writeJSON(w, http.StatusOK, map[string]any{
		"taskSummary":     summary,
		"recentTasks":     recentTasks,
		"workflowSummary": workflows,
	})
}

// handleOverview returns a combined summary of system stats and build info.
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
// Takes r (*http.Request) which provides the request context.
func (h *DevAPIHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	result := map[string]any{}

	if h.systemStats != nil {
		stats := h.systemStats.GetStats()
		result["system"] = map[string]any{
			"version":       stats.Build.Version,
			"goVersion":     stats.Build.GoVersion,
			"uptimeMs":      stats.UptimeMs,
			"heapAlloc":     stats.Memory.HeapAlloc,
			"goroutines":    stats.NumGoroutines,
			"cpuMillicores": stats.CPUMillicores,
		}
	}

	if h.orchestrator != nil {
		summary, err := h.orchestrator.ListTaskSummary(r.Context())
		if err == nil {
			result["build"] = map[string]any{
				"taskSummary": summary,
			}
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// handleHealth returns liveness and readiness probe results.
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
// Takes r (*http.Request) which provides the request context.
func (h *DevAPIHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if h.healthProbe == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	ctx := r.Context()
	writeJSON(w, http.StatusOK, map[string]any{
		"liveness":  h.healthProbe.CheckLiveness(ctx),
		"readiness": h.healthProbe.CheckReadiness(ctx),
	})
}

// handleResources returns open file descriptors grouped by category.
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
func (h *DevAPIHandler) handleResources(w http.ResponseWriter, _ *http.Request) {
	if h.resources == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	writeJSON(w, http.StatusOK, h.resources.GetResources())
}

// handleProviders returns registered resource types and their providers,
// including per-provider detail sections and sub-resources.
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
// Takes r (*http.Request) which provides the request context.
func (h *DevAPIHandler) handleProviders(w http.ResponseWriter, r *http.Request) {
	if h.providerInfo == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	ctx := r.Context()
	types := h.providerInfo.ListResourceTypes(ctx)

	providers := make(map[string]any, len(types))
	for _, rt := range types {
		list, err := h.providerInfo.ListProviders(ctx, rt)
		if err != nil {
			continue
		}

		details := make(map[string]any, len(list.Rows))
		subResources := make(map[string]any)
		for _, row := range list.Rows {
			if detail, dErr := h.providerInfo.DescribeProvider(ctx, rt, row.Name); dErr == nil && detail != nil {
				details[row.Name] = detail
			}
			if sub, sErr := h.providerInfo.ListSubResources(ctx, rt, row.Name); sErr == nil && sub != nil && len(sub.Rows) > 0 {
				subResources[row.Name] = sub
			}
		}

		providers[rt] = map[string]any{
			"Columns":      list.Columns,
			"Rows":         list.Rows,
			"details":      details,
			"subResources": subResources,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resourceTypes": types,
		"providers":     providers,
	})
}

// handleMemory returns detailed memory, process, GC, runtime, and build info.
//
// Takes w (http.ResponseWriter) which is the response writer for the JSON response.
func (h *DevAPIHandler) handleMemory(w http.ResponseWriter, _ *http.Request) {
	if h.systemStats == nil {
		writeJSON(w, http.StatusOK, map[string]any{"available": false})
		return
	}

	stats := h.systemStats.GetStats()
	writeJSON(w, http.StatusOK, map[string]any{
		"memory":  stats.Memory,
		"process": stats.Process,
		"gc":      stats.GC,
		"runtime": stats.Runtime,
		"build":   stats.Build,
	})
}

// writeJSON marshals v to JSON and writes it to w with the given status code.
//
// Takes w (http.ResponseWriter) which is the response writer.
// Takes status (int) which is the HTTP status code.
// Takes v (any) which is the value to marshal and write.
func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := devJSON.Marshal(v)
	if err != nil {
		http.Error(w, `{"error":"marshal failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
