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
	"net/http"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

// HTTPHandlerAdapter provides HTTP endpoints for health checks.
type HTTPHandlerAdapter struct {
	// service performs liveness and readiness health checks.
	service healthprobe_domain.Service
}

// NewHTTPHandlerAdapter creates a new HTTP handler adapter for health checks.
//
// Takes service (healthprobe_domain.Service) which provides health check logic.
//
// Returns *HTTPHandlerAdapter which wraps the service for HTTP handling.
func NewHTTPHandlerAdapter(service healthprobe_domain.Service) *HTTPHandlerAdapter {
	return &HTTPHandlerAdapter{
		service: service,
	}
}

// ServeLiveness handles the liveness check endpoint.
// Returns 200 OK if healthy, 503 Service Unavailable if unhealthy.
//
// Takes w (http.ResponseWriter) which receives the health check response.
// Takes r (*http.Request) which provides the request context.
func (h *HTTPHandlerAdapter) ServeLiveness(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, r, new(h.service.CheckLiveness(r.Context())))
}

// ServeReadiness handles the readiness check endpoint.
// Returns 200 OK if healthy, 503 Service Unavailable if unhealthy.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes r (*http.Request) which provides the request context.
func (h *HTTPHandlerAdapter) ServeReadiness(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, r, new(h.service.CheckReadiness(r.Context())))
}

// writeResponse writes the health check status as a JSON response.
//
// Takes w (http.ResponseWriter) which receives the JSON-encoded response.
// Takes r (*http.Request) which provides the context for error tracking.
// Takes status (*healthprobe_dto.Status) which contains the health check result.
func writeResponse(w http.ResponseWriter, r *http.Request, status *healthprobe_dto.Status) {
	httpStatusCode := http.StatusOK
	if status.State == healthprobe_dto.StateUnhealthy {
		httpStatusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatusCode)

	if err := json.ConfigStd.NewEncoder(w).Encode(status); err != nil {
		healthprobe_domain.HealthCheckErrorCount.Add(r.Context(), 1)
	}
}
