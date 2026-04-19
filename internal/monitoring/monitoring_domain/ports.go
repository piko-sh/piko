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

package monitoring_domain

import (
	"context"
	"io"
	"net/http"
	"time"

	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/tlscert"
)

// MonitoringService manages the monitoring subsystem lifecycle.
// It encapsulates telemetry collection, OTEL SDK integration, and an
// optional transport layer for remote access to monitoring data.
type MonitoringService interface {
	// Start begins background collection and, when a transport is configured,
	// the transport server. Blocks until the context is cancelled
	// or an error occurs.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns error if the transport fails to start.
	Start(ctx context.Context) error

	// Stop shuts down the monitoring subsystem in a controlled manner.
	//
	// Takes ctx (context.Context) for logging context propagation.
	Stop(ctx context.Context)

	// SpanProcessor returns the span processor for OTEL SDK setup.
	//
	// The concrete type satisfies sdktrace.SpanProcessor when the SDK is loaded
	// and should be registered with the OpenTelemetry trace provider.
	//
	// Returns SpanProcessor which processes and stores spans.
	SpanProcessor() SpanProcessor

	// MetricsReader returns the metrics reader for OTEL SDK integration.
	//
	// The concrete type satisfies sdkmetric.Reader when the SDK is loaded
	// and should be registered with the OTEL meter provider.
	//
	// Returns MetricReader which provides access to collected metrics.
	MetricsReader() MetricReader

	// Address returns the transport server listen address, or the configured
	// address when no transport is running.
	//
	// Returns string which is the address the server is listening on.
	Address() string

	// SetInspectors configures the inspector and health probe dependencies.
	//
	// This must be called before Start() for the inspectors to be available
	// via the transport. It is safe to call multiple times; the last values
	// are used.
	//
	// Takes orchestrator (OrchestratorInspector) which may be nil.
	// Takes registry (RegistryInspector) which may be nil.
	// Takes healthProbe (HealthProbeService) which may be nil.
	// Takes dispatcher (DispatcherInspector) which may be nil.
	// Takes rateLimiter (RateLimiterInspector) which may be nil.
	SetInspectors(
		orchestrator orchestrator_domain.OrchestratorInspector,
		registry registry_domain.RegistryInspector,
		healthProbe HealthProbeService,
		dispatcher dispatcher_domain.DispatcherInspector,
		rateLimiter ratelimiter_domain.RateLimiterInspector,
	)

	// SetProviderInfoInspector sets the provider info inspector for resource
	// discovery. Must be called before Start() for the inspector to be
	// available via the transport.
	//
	// Takes inspector (ProviderInfoInspector) which may be nil.
	SetProviderInfoInspector(inspector ProviderInfoInspector)

	// SetRenderCacheStatsProvider sets the render cache stats provider.
	// Must be called before Start() for cache statistics to be included
	// in the system stats response.
	//
	// Takes provider (RenderCacheStatsProvider) which may be nil.
	SetRenderCacheStatsProvider(provider RenderCacheStatsProvider)
}

// TelemetryProvider provides access to OTEL telemetry data. Implementations
// store and retrieve metrics and traces for the monitoring service.
type TelemetryProvider interface {
	// GetMetrics returns all current metrics as JSON-serialisable data.
	GetMetrics() []MetricData

	// GetSpans returns recent spans with optional filtering.
	//
	// Takes limit (int) which sets the maximum number of spans to return.
	// Takes errorsOnly (bool) which filters to return only error spans when true.
	//
	// Returns []SpanData which contains the matching spans.
	GetSpans(limit int, errorsOnly bool) []SpanData

	// GetSpanByTraceID returns all spans for a given trace ID.
	//
	// Takes traceID (string) which identifies the trace to look up.
	//
	// Returns []SpanData which contains all spans belonging to the trace.
	GetSpanByTraceID(traceID string) []SpanData
}

// SystemStatsProvider provides system statistics. Implementations collect
// runtime and system metrics for the monitoring service.
type SystemStatsProvider interface {
	// GetStats returns the current system statistics.
	//
	// Returns SystemStats which contains the current system metrics.
	GetStats() SystemStats
}

// ResourceProvider provides resource information.
// Implementations collect resource stats for the monitoring service.
type ResourceProvider interface {
	// GetResources returns the current resource data.
	//
	// Returns ResourceData which contains the resource information.
	GetResources() ResourceData
}

// RenderCacheStatsProvider provides render cache statistics for the monitoring
// service. The render registry adapter implements RenderCacheStatsProvider so that
// component and SVG cache sizes can be reported via the system stats endpoint.
type RenderCacheStatsProvider interface {
	// GetComponentCacheSize returns the number of entries in the component
	// metadata cache.
	//
	// Returns int which is the current cache size.
	GetComponentCacheSize() int

	// GetSVGCacheSize returns the number of entries in the SVG asset cache.
	//
	// Returns int which is the current cache size.
	GetSVGCacheSize() int
}

// MetricsExporter provides metrics export capabilities by integrating with the
// OTEL MeterProvider and exposing an HTTP handler. Implementations include
// Prometheus, and potentially other formats in future.
type MetricsExporter interface {
	// Reader returns the metric reader for OTEL MeterProvider registration.
	// Metrics recorded through OTEL will be available via Handler().
	//
	// Returns MetricReader for MeterProvider registration.
	Reader() MetricReader

	// Handler returns an HTTP handler that serves metrics.
	// This handler should be mounted at the configured metrics path
	// (typically /metrics).
	//
	// Returns http.Handler which serves metrics in the exporter's format.
	Handler() http.Handler
}

// MonitoringDeps holds all dependencies required by the monitoring transport
// services.
type MonitoringDeps struct {
	// OrchestratorInspector provides read-only access to orchestrator state.
	// May be nil if orchestrator is not available.
	OrchestratorInspector orchestrator_domain.OrchestratorInspector

	// RegistryInspector provides read-only access to registry state.
	// May be nil if registry is not available.
	RegistryInspector registry_domain.RegistryInspector

	// DispatcherInspector provides read-only access to dispatcher state and DLQs.
	// May be nil if no dispatchers are configured.
	DispatcherInspector dispatcher_domain.DispatcherInspector

	// RateLimiterInspector provides read-only access to rate limiter state.
	// May be nil if rate limiter is not configured.
	RateLimiterInspector ratelimiter_domain.RateLimiterInspector

	// TelemetryProvider provides access to OTEL metrics and traces.
	// May be nil if telemetry is not configured.
	TelemetryProvider TelemetryProvider

	// SystemStatsProvider provides system statistics.
	// May be nil if system stats collection is not enabled.
	SystemStatsProvider SystemStatsProvider

	// ResourceProvider provides resource information.
	// May be nil if FD tracking is not enabled.
	ResourceProvider ResourceProvider

	// HealthProbeService provides health check functions.
	// May be nil if health probing is not set up.
	HealthProbeService HealthProbeService

	// ProviderInfoInspector provides read-only access to provider information
	// across hexagon services. May be nil if no services implement
	// ResourceDescriptor.
	ProviderInfoInspector ProviderInfoInspector

	// RenderCacheStatsProvider provides render cache statistics.
	// May be nil if the render registry is not available.
	RenderCacheStatsProvider RenderCacheStatsProvider

	// ProfilingController manages on-demand pprof profiling.
	// May be nil if remote profiling is not enabled.
	ProfilingController ProfilingController

	// WatchdogInspector provides read-only access to watchdog state and stored
	// profiles. May be nil if the watchdog is not enabled.
	WatchdogInspector WatchdogInspector
}

// HealthProbeService provides health check methods for liveness and readiness.
// It implements monitoring_domain.HealthProbeService.
type HealthProbeService interface {
	// CheckLiveness runs all liveness health probes.
	//
	// Returns HealthProbeStatus which contains the results of all probes.
	CheckLiveness(ctx context.Context) HealthProbeStatus

	// CheckReadiness runs all readiness health checks.
	//
	// Returns HealthProbeStatus which contains the results of all checks.
	CheckReadiness(ctx context.Context) HealthProbeStatus
}

// MetricsCollectorAdapter defines the interface for metrics collection
// adapters. Implementations collect metrics from OTEL and store them in the
// TelemetryStore.
type MetricsCollectorAdapter interface {
	// Start begins periodic metric collection.
	Start(ctx context.Context)

	// Stop halts the periodic collection.
	Stop()

	// Reader returns the metrics reader for OTEL MeterProvider registration.
	Reader() MetricReader
}

// ProviderInfoInspector provides read-only access to provider information
// across all hexagon services. It aggregates data from ResourceDescriptors
// registered at bootstrap time.
type ProviderInfoInspector interface {
	// ListResourceTypes returns the names of all registered resource types
	// (e.g. "email", "storage", "cache").
	//
	// Returns []string which contains the sorted resource type names.
	ListResourceTypes(ctx context.Context) []string

	// ListProviders returns column definitions and provider rows for a
	// specific resource type.
	//
	// Takes resourceType (string) which identifies the resource to query.
	//
	// Returns *ProviderListResult which contains columns and rows.
	// Returns error when the resource type is not found.
	ListProviders(ctx context.Context, resourceType string) (*ProviderListResult, error)

	// DescribeProvider returns detailed information for a single provider
	// within a resource type.
	//
	// Takes resourceType (string) which identifies the resource.
	// Takes name (string) which identifies the provider.
	//
	// Returns *provider_domain.ProviderDetail which contains structured sections.
	// Returns error when the resource type or provider is not found.
	DescribeProvider(ctx context.Context, resourceType, name string) (*provider_domain.ProviderDetail, error)

	// ListSubResources returns sub-resources for a named provider within a
	// resource type. Only works when the service implements
	// SubResourceDescriptor.
	//
	// Takes resourceType (string) which identifies the resource.
	// Takes providerName (string) which identifies the provider.
	//
	// Returns *ProviderListResult which contains sub-resource columns and rows.
	// Returns error when the resource type is not found or the service does
	// not support sub-resources.
	ListSubResources(ctx context.Context, resourceType, providerName string) (*ProviderListResult, error)

	// DescribeResourceType returns a service-level overview for the given
	// resource type. Only works when the service implements
	// ResourceTypeDescriptor.
	//
	// Takes resourceType (string) which identifies the resource.
	//
	// Returns *provider_domain.ProviderDetail which contains the overview.
	// Returns error when the resource type is not found or the service does
	// not support type-level describe.
	DescribeResourceType(ctx context.Context, resourceType string) (*provider_domain.ProviderDetail, error)
}

// TransportServer is the port interface for a monitoring
// transport layer that handles network serving and
// delegates to domain dependencies via MonitoringDeps.
type TransportServer interface {
	// Start begins serving on the configured address. Blocks
	// until the context is cancelled or an error occurs.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns error when the server fails to start or encounters an error.
	Start(ctx context.Context) error

	// Stop shuts down the transport server gracefully.
	//
	// Takes ctx (context.Context) for logging context propagation.
	Stop(ctx context.Context)

	// Address returns the address the transport is listening on.
	//
	// Returns string which is the network address.
	Address() string
}

// TransportFactory creates a TransportServer from the given dependencies and
// configuration.
//
// Transport adapters (e.g. wdk/monitoring/monitoring_transport_grpc) provide
// implementations of the function type.
type TransportFactory func(deps MonitoringDeps, config TransportConfig) (TransportServer, error)

// TransportConfig holds transport-agnostic server settings passed from the
// domain to the transport factory.
type TransportConfig struct {
	// Address is the TCP address to listen on (e.g. ":9091" or "localhost:9091").
	Address string

	// TLS holds the resolved TLS settings for the transport. When enabled,
	// the transport uses TLS for security.
	TLS tlscert.TLSValues

	// AutoNextPort enables automatic port selection when the configured port
	// is already in use.
	AutoNextPort bool
}

// SpanProcessor receives span lifecycle events from the OTEL SDK.
//
// The domain layer passes these opaquely between adapters and the OTEL setup without
// calling methods directly. Concrete implementations (e.g. from otel/sdk/trace)
// satisfy SpanProcessor via structural typing.
type SpanProcessor interface {
	// Shutdown flushes remaining spans and releases resources.
	//
	// Takes ctx (context.Context) which controls the shutdown deadline.
	//
	// Returns error when the shutdown fails or the context expires.
	Shutdown(ctx context.Context) error

	// ForceFlush immediately exports all buffered span data.
	//
	// Takes ctx (context.Context) which controls the flush deadline.
	//
	// Returns error when the flush fails or the context expires.
	ForceFlush(ctx context.Context) error
}

// MetricReader provides collected metrics to the OTEL SDK.
//
// The domain layer passes these opaquely between adapters and the OTEL setup without
// calling methods directly. Concrete implementations (e.g. from otel/sdk/metric)
// satisfy MetricReader via structural typing.
type MetricReader interface {
	// Shutdown flushes remaining metrics and releases resources.
	//
	// Takes ctx (context.Context) which controls the shutdown deadline.
	//
	// Returns error when the shutdown fails or the context expires.
	Shutdown(ctx context.Context) error
}

// SpanProcessorFactory creates a span processor that stores spans in the given
// store, letting adapters provide their span processor implementation.
type SpanProcessorFactory func(store *TelemetryStore) SpanProcessor

// MetricsCollectorFactory creates a metrics collector that stores metrics in
// the given store, letting adapters provide their own metrics collector
// implementation.
type MetricsCollectorFactory func(store *TelemetryStore, interval time.Duration) MetricsCollectorAdapter

// ProfilingController manages on-demand pprof profiling for production
// diagnostics. It controls the lifecycle of the pprof HTTP server and
// Go runtime profiling rates, with mandatory auto-disable after a
// configured duration.
type ProfilingController interface {
	// Enable starts the pprof HTTP server and sets Go runtime profiling
	// rates. If already enabled, it extends the expiry deadline without
	// restarting the server.
	//
	// Takes opts (ProfilingEnableOpts) which configures the profiling
	// session duration, port, and sampling rates.
	//
	// Returns *ProfilingStatus with the current state after enabling.
	// The AlreadyEnabled field is set when profiling was already active.
	// Returns error when the server fails to start or the duration
	// exceeds the maximum allowed.
	Enable(ctx context.Context, opts ProfilingEnableOpts) (*ProfilingStatus, error)

	// Disable stops the pprof HTTP server and resets Go runtime profiling
	// rates to their pre-enable values.
	//
	// Returns bool which is true if profiling was previously enabled.
	// Returns error when the server fails to shut down cleanly.
	Disable(ctx context.Context) (bool, error)

	// Close cancels any pending auto-disable timer and disables profiling.
	// It is safe for use as a shutdown hook.
	//
	// Returns error when the shutdown fails.
	Close(ctx context.Context) error

	// Status returns the current profiling state including whether it is
	// enabled, the port, time remaining, and available profile types.
	//
	// Returns *ProfilingStatus with the current state.
	Status(ctx context.Context) *ProfilingStatus

	// SnapshotFlightRecorder writes the current rolling execution trace
	// buffer to the provided writer. Returns an error when the flight
	// recorder is not enabled or the write fails.
	//
	// Takes w (io.Writer) which receives the trace data.
	//
	// Returns error when the flight recorder is disabled or the snapshot
	// fails.
	SnapshotFlightRecorder(ctx context.Context, w io.Writer) error

	// CaptureProfile captures a Go runtime profile and writes the raw
	// pprof data to the provided writer. For duration-based profiles
	// (cpu, trace), this blocks for the requested duration.
	//
	// Takes profileType (string) which identifies the profile to capture
	// (heap, goroutine, allocs, cpu, trace, block, mutex).
	// Takes durationSeconds (int) which sets the capture window for
	// duration-based profiles; ignored for snapshot profiles.
	// Takes w (io.Writer) which receives the raw profile data.
	//
	// Returns string which contains any warning message (e.g. when
	// runtime rates are not configured for the requested profile type).
	// Returns error when the profile type is unknown or capture fails.
	CaptureProfile(ctx context.Context, profileType string, durationSeconds int, w io.Writer) (string, error)
}

// ProfilingEnableOpts configures a profiling session.
type ProfilingEnableOpts struct {
	// Duration is how long profiling remains enabled before auto-disabling.
	Duration time.Duration

	// Port is the pprof HTTP server port. Zero uses the default (6060).
	Port int

	// BlockProfileRate sets the Go runtime block profile rate. Zero uses
	// the default production-safe rate.
	BlockProfileRate int

	// MutexProfileFraction sets the Go runtime mutex profile fraction.
	// Zero uses the default production-safe fraction.
	MutexProfileFraction int
}

// ProfilingStatus describes the current state of on-demand profiling.
type ProfilingStatus struct {
	// ExpiresAt is when profiling will auto-disable.
	ExpiresAt time.Time

	// PprofBaseURL is the full URL prefix for pprof endpoints.
	PprofBaseURL string

	// AvailableProfiles lists the profile types that can be captured.
	AvailableProfiles []string

	// Port is the port the pprof HTTP server listens on.
	Port int

	// BlockProfileRate is the current Go runtime block profile rate.
	BlockProfileRate int

	// MutexProfileFraction is the current Go runtime mutex profile fraction.
	MutexProfileFraction int

	// MemProfileRate is the current Go runtime memory profile rate.
	MemProfileRate int

	// Enabled is true when the pprof HTTP server is running.
	Enabled bool

	// AlreadyEnabled is true when profiling was already active before the
	// current Enable call. This allows the transport layer to inform the
	// caller that the session was extended rather than freshly started.
	AlreadyEnabled bool
}
