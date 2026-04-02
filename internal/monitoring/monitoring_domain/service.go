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
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/tlscert"
)

// ServiceConfig holds settings for the monitoring service.
type ServiceConfig struct {
	// TransportFactory creates the transport server for remote monitoring access.
	// When nil, the service runs in local-only mode (collectors run but no
	// transport serves remote clients).
	TransportFactory TransportFactory

	// Factories overrides the default noop service factories with real
	// implementations (e.g. from logger_otel_sdk.OtelServiceFactories()).
	//
	// When nil, the bootstrap layer uses its default noop factories.
	Factories *ServiceFactories

	// Address specifies the transport server listen address. If it starts with
	// a colon, BindAddress is prepended to form the full address.
	Address string

	// BindAddress is the network address to bind to when the Address field starts
	// with a colon.
	BindAddress string

	// TLS holds the resolved TLS settings for the monitoring transport.
	// When enabled, the transport uses TLS for security.
	TLS tlscert.TLSValues

	// MaxSpans is the maximum number of spans to keep.
	MaxSpans int

	// MaxMetrics is the maximum number of unique metrics to store.
	MaxMetrics int

	// MaxMetricAge is the maximum age for metric data points; older data is discarded.
	MaxMetricAge time.Duration

	// MetricsCollectionInterval is how often to collect metrics from OTEL.
	MetricsCollectionInterval time.Duration

	// AutoNextPort enables automatic port selection when the configured port
	// is already in use.
	AutoNextPort bool
}

// DefaultMetricsCollectionInterval is the default interval for collecting metrics.
const DefaultMetricsCollectionInterval = 5 * time.Second

var _ MonitoringService = (*Service)(nil)

// ServiceOption configures the monitoring service.
type ServiceOption func(*ServiceConfig)

// Service implements MonitoringService.
// It manages all monitoring parts and controls their lifecycle.
type Service struct {
	// factories holds the service factory functions for creating OTEL adapters.
	factories ServiceFactories

	// spanProcessor handles span processing and storage for traces.
	spanProcessor SpanProcessor

	// metricsCollector collects and stores metrics; started and stopped with the
	// service.
	metricsCollector MetricsCollectorAdapter

	// orchestratorInspector queries orchestrator state for monitoring.
	orchestratorInspector orchestrator_domain.OrchestratorInspector

	// registryInspector inspects the container registry for image information.
	registryInspector registry_domain.RegistryInspector

	// healthProbeService provides health probe functionality for status reporting.
	healthProbeService HealthProbeService

	// dispatcherInspector queries dispatcher state and DLQs for monitoring.
	dispatcherInspector dispatcher_domain.DispatcherInspector

	// rateLimiterInspector queries rate limiter state for monitoring.
	rateLimiterInspector ratelimiter_domain.RateLimiterInspector

	// providerInfoInspector provides provider information across hexagons.
	providerInfoInspector ProviderInfoInspector

	// renderCacheStats provides render cache statistics; nil when not available.
	renderCacheStats RenderCacheStatsProvider

	// store provides telemetry data for the server dashboard.
	store *TelemetryStore

	// systemCollector gathers system-level metrics and is started and stopped
	// with the service lifecycle.
	systemCollector *SystemCollector

	// resourceCollector provides file descriptor metrics for telemetry.
	resourceCollector *ResourceCollector

	// transport holds the transport server instance; nil when no transport is
	// configured or Start has not been called yet.
	transport TransportServer

	// config holds the service configuration for address binding.
	config ServiceConfig

	// mu protects the inspector fields which may be set after construction.
	mu sync.RWMutex
}

// ServiceFactories holds the factory functions required by the Service.
// These are provided by adapters to create the OTEL adapter components.
type ServiceFactories struct {
	// SpanProcessorFactory creates span processors for the monitoring service.
	SpanProcessorFactory SpanProcessorFactory

	// MetricsCollectorFactory creates the metrics collector.
	MetricsCollectorFactory MetricsCollectorFactory
}

// NewService creates a new monitoring service.
//
// Takes deps (MonitoringDeps) which provides orchestrator and registry inspectors.
// Takes factories (ServiceFactories) which provides adapter factories.
// Takes opts (...ServiceOption) for configuration.
//
// Returns *Service ready to be started.
func NewService(deps MonitoringDeps, factories ServiceFactories, opts ...ServiceOption) *Service {
	config := ServiceConfig{
		Address:                   ":9091",
		BindAddress:               "127.0.0.1",
		MaxSpans:                  DefaultMaxSpans,
		MaxMetrics:                DefaultMaxMetrics,
		MaxMetricAge:              DefaultMaxMetricAge,
		MetricsCollectionInterval: DefaultMetricsCollectionInterval,
	}

	for _, opt := range opts {
		opt(&config)
	}

	store := NewTelemetryStore(
		WithMaxSpans(config.MaxSpans),
		WithMaxMetrics(config.MaxMetrics),
		WithMaxMetricAge(config.MaxMetricAge),
	)

	spanProcessor := factories.SpanProcessorFactory(store)
	metricsCollector := factories.MetricsCollectorFactory(store, config.MetricsCollectionInterval)
	listenAddr := config.Address
	if config.Address[0] == ':' {
		listenAddr = config.BindAddress + config.Address
	}

	systemCollector := NewSystemCollector(WithListenAddress(listenAddr))
	resourceCollector := NewResourceCollector()

	return &Service{
		store:                 store,
		spanProcessor:         spanProcessor,
		metricsCollector:      metricsCollector,
		systemCollector:       systemCollector,
		resourceCollector:     resourceCollector,
		config:                config,
		factories:             factories,
		orchestratorInspector: deps.OrchestratorInspector,
		registryInspector:     deps.RegistryInspector,
	}
}

// SetInspectors updates the orchestrator, registry, dispatcher, rate limiter
// inspectors and health probe service. This must be called before Start() for
// the inspectors to be available via the transport.
//
// Takes orchestrator (OrchestratorInspector) which may be nil.
// Takes registry (RegistryInspector) which may be nil.
// Takes healthProbe (HealthProbeService) which may be nil.
// Takes dispatcher (DispatcherInspector) which may be nil.
// Takes rateLimiter (RateLimiterInspector) which may be nil.
//
// Safe for concurrent use. The last values set are used when Start() is called.
func (s *Service) SetInspectors(
	orchestrator orchestrator_domain.OrchestratorInspector,
	registry registry_domain.RegistryInspector,
	healthProbe HealthProbeService,
	dispatcher dispatcher_domain.DispatcherInspector,
	rateLimiter ratelimiter_domain.RateLimiterInspector,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orchestratorInspector = orchestrator
	s.registryInspector = registry
	s.healthProbeService = healthProbe
	s.dispatcherInspector = dispatcher
	s.rateLimiterInspector = rateLimiter
}

// SetProviderInfoInspector sets the provider info inspector for resource
// discovery. This must be called before Start() for the inspector to be
// available via the transport.
//
// Takes inspector (ProviderInfoInspector) which may be nil.
//
// Safe for concurrent use.
func (s *Service) SetProviderInfoInspector(inspector ProviderInfoInspector) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providerInfoInspector = inspector
}

// SetRenderCacheStatsProvider sets the render cache stats provider for
// reporting cache sizes via the system stats endpoint. Must be called before
// Start() for the stats to be available.
//
// Takes provider (RenderCacheStatsProvider) which may be nil.
//
// Safe for concurrent use.
func (s *Service) SetRenderCacheStatsProvider(provider RenderCacheStatsProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.renderCacheStats = provider
}

// Start begins background collection and, when a transport factory is
// configured, the transport server. This method blocks until the context is
// cancelled or an error occurs.
//
// Returns error when the transport fails to start.
//
// Safe for concurrent use. Starts metric and system collectors in the
// background, then blocks on the transport (or context) until cancelled.
func (s *Service) Start(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Starting monitoring service")

	address := s.config.Address
	if s.config.Address[0] == ':' {
		address = s.config.BindAddress + s.config.Address
	}

	s.mu.RLock()
	deps := MonitoringDeps{
		OrchestratorInspector:    s.orchestratorInspector,
		RegistryInspector:        s.registryInspector,
		DispatcherInspector:      s.dispatcherInspector,
		RateLimiterInspector:     s.rateLimiterInspector,
		TelemetryProvider:        s.store,
		SystemStatsProvider:      s.systemCollector,
		ResourceProvider:         s.resourceCollector,
		HealthProbeService:       s.healthProbeService,
		ProviderInfoInspector:    s.providerInfoInspector,
		RenderCacheStatsProvider: s.renderCacheStats,
	}
	s.mu.RUnlock()

	s.metricsCollector.Start(ctx)
	s.systemCollector.Start(ctx)
	s.resourceCollector.Start(ctx)

	if s.config.TransportFactory != nil {
		transportConfig := TransportConfig{
			Address:      address,
			AutoNextPort: s.config.AutoNextPort,
			TLS:          s.config.TLS,
		}

		transport, err := s.config.TransportFactory(deps, transportConfig)
		if err != nil {
			return fmt.Errorf("creating monitoring transport: %w", err)
		}
		s.mu.Lock()
		s.transport = transport
		s.mu.Unlock()

		return transport.Start(ctx)
	}

	<-ctx.Done()
	return ctx.Err()
}

// Stop shuts down the monitoring service and releases its resources.
//
// Takes ctx (context.Context) for logging context propagation.
//
// Safe for concurrent use. Protected by a read lock on the service mutex.
func (s *Service) Stop(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	l.Notice("Stopping monitoring service")

	s.mu.RLock()
	transport := s.transport
	s.mu.RUnlock()

	if transport != nil {
		transport.Stop(ctx)
	}
	s.systemCollector.Stop()
	s.resourceCollector.Stop()
	s.metricsCollector.Stop()
}

// SpanProcessor returns the OTEL span processor for SDK integration.
// This should be registered with the OTEL trace provider.
//
// Returns SpanProcessor which processes and stores spans.
func (s *Service) SpanProcessor() SpanProcessor {
	return s.spanProcessor
}

// MetricsReader returns the OTEL metrics reader for SDK integration.
// This should be registered with the OTEL meter provider.
//
// Returns MetricReader which reads and stores metrics.
func (s *Service) MetricsReader() MetricReader {
	return s.metricsCollector.Reader()
}

// Address returns the address the transport server is listening on, or the
// configured address when no transport is running.
//
// Returns string which is the server address.
//
// Safe for concurrent use. Protected by a read lock on the service mutex.
func (s *Service) Address() string {
	s.mu.RLock()
	transport := s.transport
	s.mu.RUnlock()

	if transport != nil {
		return transport.Address()
	}
	if s.config.Address[0] == ':' {
		return s.config.BindAddress + s.config.Address
	}
	return s.config.Address
}

// WithServiceAddress sets the gRPC server listen address.
//
// Takes addr (string) which specifies the address to listen on.
//
// Returns ServiceOption which configures the service address.
func WithServiceAddress(addr string) ServiceOption {
	return func(c *ServiceConfig) {
		c.Address = addr
	}
}

// WithServiceBindAddress sets the network address to bind to.
//
// Takes addr (string) which specifies the address and port to bind to.
//
// Returns ServiceOption which configures the bind address on a ServiceConfig.
func WithServiceBindAddress(addr string) ServiceOption {
	return func(c *ServiceConfig) {
		c.BindAddress = addr
	}
}

// WithServiceMaxSpans sets the maximum number of spans to retain.
//
// Takes n (int) which specifies the maximum span count.
//
// Returns ServiceOption which configures the span limit on a ServiceConfig.
func WithServiceMaxSpans(n int) ServiceOption {
	return func(c *ServiceConfig) {
		c.MaxSpans = n
	}
}

// WithServiceMaxMetrics sets the maximum number of metrics to retain.
//
// Takes n (int) which specifies the maximum number of metrics to keep.
//
// Returns ServiceOption which configures the metric retention limit.
func WithServiceMaxMetrics(n int) ServiceOption {
	return func(c *ServiceConfig) {
		c.MaxMetrics = n
	}
}

// WithServiceMaxMetricAge sets the maximum age for metric data points.
//
// Takes d (time.Duration) which specifies the maximum age before metrics
// expire.
//
// Returns ServiceOption which configures the maximum metric age on a service.
func WithServiceMaxMetricAge(d time.Duration) ServiceOption {
	return func(c *ServiceConfig) {
		c.MaxMetricAge = d
	}
}

// WithServiceMetricsInterval sets the metrics collection interval.
//
// Takes d (time.Duration) which specifies how often metrics are collected.
//
// Returns ServiceOption which configures the metrics interval on a service.
func WithServiceMetricsInterval(d time.Duration) ServiceOption {
	return func(c *ServiceConfig) {
		c.MetricsCollectionInterval = d
	}
}

// WithServiceAutoNextPort enables automatic port selection for the monitoring
// server. When enabled, if the configured port is already in use, the server
// tries the next port, up to 100 attempts.
//
// Takes enabled (bool) which controls whether auto-port selection is active.
//
// Returns ServiceOption which configures auto-port selection on the service.
func WithServiceAutoNextPort(enabled bool) ServiceOption {
	return func(c *ServiceConfig) {
		c.AutoNextPort = enabled
	}
}

// WithServiceTLS sets the TLS configuration for the monitoring transport.
//
// Takes tls (tlscert.TLSValues) which specifies the TLS settings.
//
// Returns ServiceOption which configures TLS on a service.
func WithServiceTLS(tls tlscert.TLSValues) ServiceOption {
	return func(c *ServiceConfig) {
		c.TLS = tls
	}
}

// WithServiceTransportFactory sets the transport factory for remote monitoring
// access. When nil (default), the service runs in local-only mode: collectors
// are active but no transport serves remote clients.
//
// Takes factory (TransportFactory) which creates the transport server.
//
// Returns ServiceOption which configures the transport factory on a service.
func WithServiceTransportFactory(factory TransportFactory) ServiceOption {
	return func(c *ServiceConfig) {
		c.TransportFactory = factory
	}
}
