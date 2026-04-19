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
	"google.golang.org/grpc"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

// defaultServiceRegistrar returns a registrar that registers all standard
// monitoring gRPC services based on available dependencies.
//
// Returns serviceRegistrar which registers health, metrics, and inspector
// services.
func defaultServiceRegistrar() serviceRegistrar {
	return func(server *grpc.Server, deps monitoring_domain.MonitoringDeps) {
		pb.RegisterHealthServiceServer(server, NewHealthService(deps.HealthProbeService))
		registerMetricsService(server, deps)
		registerInspectorServices(server, deps)
	}
}

// registerMetricsService registers the metrics gRPC service when at least one
// telemetry provider is available.
//
// Takes server (*grpc.Server) which is the gRPC server to register the service
// on.
// Takes deps (monitoring_domain.MonitoringDeps) which provides the telemetry,
// system stats, resource, and render cache dependencies.
func registerMetricsService(server *grpc.Server, deps monitoring_domain.MonitoringDeps) {
	if deps.TelemetryProvider != nil || deps.SystemStatsProvider != nil || deps.ResourceProvider != nil {
		pb.RegisterMetricsServiceServer(server, NewMetricsService(
			deps.TelemetryProvider,
			deps.SystemStatsProvider,
			deps.ResourceProvider,
			deps.RenderCacheStatsProvider,
		))
	}
}

// registerInspectorServices registers all optional inspector gRPC services
// that have non-nil dependencies.
//
// Takes server (*grpc.Server) which is the gRPC server to register the
// services on.
// Takes deps (monitoring_domain.MonitoringDeps) which provides the optional
// inspector dependencies checked for nil before registration.
func registerInspectorServices(server *grpc.Server, deps monitoring_domain.MonitoringDeps) {
	if deps.OrchestratorInspector != nil {
		pb.RegisterOrchestratorInspectorServiceServer(server, NewOrchestratorInspectorService(deps.OrchestratorInspector))
	}

	if deps.RegistryInspector != nil {
		pb.RegisterRegistryInspectorServiceServer(server, NewRegistryInspectorService(deps.RegistryInspector))
	}

	if deps.DispatcherInspector != nil {
		pb.RegisterDispatcherInspectorServiceServer(server, NewDispatcherInspectorService(deps.DispatcherInspector))
	}

	if deps.RateLimiterInspector != nil {
		pb.RegisterRateLimiterInspectorServiceServer(server, NewRateLimiterInspectorService(deps.RateLimiterInspector))
	}

	if deps.ProviderInfoInspector != nil {
		pb.RegisterProviderInfoServiceServer(server, NewProviderInfoService(deps.ProviderInfoInspector))
	}

	if deps.ProfilingController != nil {
		pb.RegisterProfilingServiceServer(server, NewProfilingService(deps.ProfilingController))
	}

	if deps.WatchdogInspector != nil {
		pb.RegisterWatchdogInspectorServiceServer(server, NewWatchdogInspectorService(deps.WatchdogInspector))
	}
}
