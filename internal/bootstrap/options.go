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

package bootstrap

import (
	"strconv"
	"time"

	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/crypto/crypto_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/image/image_domain"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/markdown/markdown_domain"
	"piko.sh/piko/internal/monitoring/monitoring_adapters"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// kilobyte is 2^10 bytes.
	kilobyte = 1024

	// megabyte is the number of bytes in a megabyte.
	megabyte = 1024 * kilobyte

	// defaultRegistryCacheSizeMB is the default size for the registry cache
	// in megabytes.
	defaultRegistryCacheSizeMB = 256

	// defaultRegistryCacheTTLMinutes is the default time-to-live in minutes for
	// registry cache entries.
	defaultRegistryCacheTTLMinutes = 30
)

// MonitoringOption configures the monitoring service.
type MonitoringOption func(*monitoring_domain.ServiceConfig)

// ProfilingOption configures the pprof HTTP debug server.
type ProfilingOption func(*profiler.Config)

// GeneratorProfilingOption configures profiling for short-lived generator
// builds that capture profiles to disk.
type GeneratorProfilingOption func(*profiler.Config)

// cssResetConfig holds the intermediate configuration for the CSS reset
// feature before it is resolved into the final CSS string.
type cssResetConfig struct {
	// css is the custom CSS override provided by the user. When non-empty, it
	// takes precedence over the useComplete flag.
	css string

	// useComplete selects the comprehensive legacy reset instead of the simple
	// default.
	useComplete bool
}

// CSSResetOption is a functional option for configuring the CSS reset feature.
type CSSResetOption func(*cssResetConfig)

// WithPort sets the TCP port for the main HTTP server.
//
// Takes port (int) which specifies the port number (e.g. 8080, 443).
//
// Returns Option which sets the server port.
func WithPort(port int) Option {
	return func(c *Container) {
		overrides := c.ensureOverrides()
		s := strconv.Itoa(port)
		overrides.Network.Port = &s
	}
}

// WithEventBus sets a custom EventBus implementation for the container.
//
// If the bus implements a shutdown interface (Close, Shutdown, or Stop),
// it will be registered for graceful shutdown.
//
// Takes bus (EventBus) which specifies the event bus to use.
//
// Returns Option which configures the container to use the given event bus.
func WithEventBus(bus orchestrator_domain.EventBus) Option {
	return func(c *Container) {
		c.eventBusOverride = bus
		registerCloseableForShutdown(c.GetAppContext(), "EventBus", bus)
	}
}

// WithRegistryService sets a custom RegistryService for the container.
//
// If the service has a shutdown method (Close, Shutdown, or Stop), it will be
// registered for graceful shutdown.
//
// Takes service (RegistryService) which is the registry service to use.
//
// Returns Option which sets up the container to use the provided service.
func WithRegistryService(service registry_domain.RegistryService) Option {
	return func(c *Container) {
		c.registryServiceOverride = service
		registerCloseableForShutdown(c.GetAppContext(), "RegistryService", service)
	}
}

// WithCapabilityService sets a custom capability service for the container.
//
// If the service has a shutdown method (Close, Shutdown, or Stop), it will be
// registered for graceful shutdown.
//
// Takes service (capabilities.Service) which replaces the default capability
// service.
//
// Returns Option which sets up the container with the custom service.
func WithCapabilityService(service capabilities.Service) Option {
	return func(c *Container) {
		c.capabilityServiceOverride = service
		registerCloseableForShutdown(c.GetAppContext(), "CapabilityService", service)
	}
}

// WithOrchestratorService sets a custom OrchestratorService for the container.
//
// If the service has a shutdown method (Close, Shutdown, or Stop), it will be
// added to the list of services that are closed when the application stops.
//
// Takes service (OrchestratorService) which is the custom service to use.
//
// Returns Option which sets the container to use the given service.
func WithOrchestratorService(service orchestrator_domain.OrchestratorService) Option {
	return func(c *Container) {
		c.orchestratorServiceOverride = service
		registerCloseableForShutdown(c.GetAppContext(), "OrchestratorService", service)
	}
}

// WithI18nService provides a custom internationalisation service.
//
// If the service has a shutdown method (Close, Shutdown, or Stop), it will be
// registered for graceful shutdown.
//
// Takes service (i18n_domain.Service) which is the internationalisation
// service to use.
//
// Returns Option which sets up the container to use the provided service.
func WithI18nService(service i18n_domain.Service) Option {
	return func(c *Container) {
		c.i18nServiceOverride = service
		registerCloseableForShutdown(c.GetAppContext(), "I18nService", service)
	}
}

// WithEmailService sets a custom email service for the container.
//
// If the service has a shutdown method (Close, Shutdown, or Stop), it will be
// registered for graceful shutdown when the container stops.
//
// Takes service (email_domain.Service) which is the email service to use.
//
// Returns Option which configures the container with the given service.
func WithEmailService(service email_domain.Service) Option {
	return func(c *Container) {
		c.emailServiceOverride = service
		registerCloseableForShutdown(c.GetAppContext(), "EmailService", service)
	}
}

// WithImageService sets a custom image service for the container.
//
// Takes service (image_domain.Service) which handles image operations.
//
// Returns Option which configures the container to use the given service.
func WithImageService(service image_domain.Service) Option {
	return func(c *Container) { c.SetImageService(service) }
}

// WithCacheService sets a custom cache service for the container.
//
// Takes service (cache_domain.Service) which is the cache service to use.
//
// Returns Option which configures the container with the given cache service.
func WithCacheService(service cache_domain.Service) Option {
	return func(c *Container) { c.SetCacheService(service) }
}

// WithCryptoService sets a custom crypto service implementation.
//
// Takes service (CryptoServicePort) which specifies the crypto service to use.
//
// Returns Option which configures the container with the given crypto service.
func WithCryptoService(service crypto_domain.CryptoServicePort) Option {
	return func(c *Container) { c.SetCryptoService(service) }
}

// WithSearchService sets a custom search service implementation.
// This is typically used for testing with mock implementations.
//
// Takes service (SearchServicePort) which specifies the search service to use.
//
// Returns Option which configures the container with the given search service.
func WithSearchService(service collection_domain.SearchServicePort) Option {
	return func(c *Container) { c.SetSearchService(service) }
}

// WithMarkdownParser sets the markdown parser implementation used by the
// collection service for processing markdown content.
//
// When a parser is configured, the collection service will register a markdown
// content provider that uses it to parse .md files. If no parser is set, the
// markdown collection provider is not registered.
//
// Takes parser (markdown_domain.MarkdownParserPort) which provides the
// markdown parsing implementation.
//
// Returns Option which configures the container with the given parser.
func WithMarkdownParser(parser markdown_domain.MarkdownParserPort) Option {
	return func(c *Container) {
		c.SetMarkdownParser(parser)
	}
}

// WithMemoryRegistryCache configures the default RegistryService to use an
// in-memory cache via the cache hexagon. For more control over cache settings,
// use WithRegistryMetadataCacheConfig instead.
//
// Returns Option which applies the in-memory cache settings to the container.
func WithMemoryRegistryCache() Option {
	return func(c *Container) {
		c.registryMetadataCacheConfig = &RegistryMetadataCacheConfig{
			MaxWeight:    defaultRegistryCacheSizeMB * megabyte,
			TTL:          defaultRegistryCacheTTLMinutes * time.Minute,
			StatsEnabled: true,
		}
	}
}

// WithJSONTypeInspectorCache configures the TypeInspectorManager to use a
// JSON file-based cache for storing type inspection data.
//
// Returns Option which applies the JSON cache configuration to a Container.
func WithJSONTypeInspectorCache() Option {
	return func(c *Container) {
		c.typeDataProvider = func(sandbox safedisk.Sandbox) inspector_domain.TypeDataProvider {
			_, l := logger_domain.From(c.GetAppContext(), log)
			l.Internal("Using JSONTypeInspectorCache provider (configured by option)")
			return inspector_adapters.NewJSONCache(sandbox)
		}
	}
}

// WithCSRFSecret sets the secret key used for CSRF token generation.
//
// Takes key ([]byte) which is the secret key for generating CSRF tokens.
//
// Returns Option which applies the CSRF secret setting to a Container.
func WithCSRFSecret(key []byte) Option {
	return func(c *Container) {
		c.csrfSecretKeyProvider = func() []byte {
			return key
		}
	}
}

// WithCSRFTokenMaxAge sets the safety-net maximum age for CSRF tokens.
//
// The primary expiry mechanism is cookie rotation; this is a fallback for
// tokens backed by cookies that persist beyond their expected lifetime.
//
// When d is 0 or negative, the default of 30 days is used.
//
// Takes d (time.Duration) which specifies the maximum token age.
//
// Returns Option which applies the CSRF token max age to a Container.
func WithCSRFTokenMaxAge(d time.Duration) Option {
	return func(c *Container) {
		c.csrfTokenMaxAge = d
	}
}

// WithCSRFSecFetchSiteEnforcement controls whether browser requests (identified
// by the Sec-Fetch-Site header) are required to include CSRF tokens, enabled by
// default for defence-in-depth security.
//
// Takes enabled (bool) which controls enforcement.
//
// Returns Option which applies the CSRF Sec-Fetch-Site enforcement to a Container.
func WithCSRFSecFetchSiteEnforcement(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.CSRF.SecFetchSiteEnforcement = &enabled
	}
}

// WithTrustedProxies configures the CIDR ranges of reverse proxies trusted to
// set X-Forwarded-For headers. When a request arrives from one of these ranges,
// the real client IP is extracted from forwarding headers rather than using the
// connection IP directly.
//
// Common values include RFC 1918 private ranges: "10.0.0.0/8",
// "172.16.0.0/12", "192.168.0.0/16".
//
// Takes cidrs (...string) which are the CIDR ranges to trust.
//
// Returns Option which configures the trusted proxy list.
func WithTrustedProxies(cidrs ...string) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.RateLimit.TrustedProxies = cidrs
	}
}

// WithCloudflareEnabled controls whether to trust the CF-Connecting-IP header
// from trusted proxies.
//
// When false (default), the header is ignored even from trusted proxies.
// Enable this only when Cloudflare is your edge proxy and its IP ranges are
// listed in TrustedProxies.
//
// Takes enabled (bool) which controls whether CF-Connecting-IP is trusted.
//
// Returns Option which configures the Cloudflare setting.
func WithCloudflareEnabled(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.RateLimit.CloudflareEnabled = &enabled
	}
}

// WithRateLimitEnabled enables or disables HTTP request rate limiting.
// Rate limiting is disabled by default to prevent accidental self-limiting
// when deployed behind a reverse proxy without TrustedProxies configured.
//
// Takes enabled (bool) which controls whether rate limiting is active.
//
// Returns Option which configures rate limiting.
func WithRateLimitEnabled(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Security.RateLimit.Enabled = &enabled
	}
}

// WithConfigResolvers adds custom configuration resolvers to the container.
//
// Takes resolvers (...config_domain.Resolver) which are the resolvers to add.
//
// Returns Option which sets up the container with the given resolvers.
func WithConfigResolvers(resolvers ...config_domain.Resolver) Option {
	return func(c *Container) {
		c.configResolvers = append(c.configResolvers, resolvers...)
	}
}

// WithShutdownDrainDelay sets the duration to wait after marking the instance
// as not ready (readiness returns 503) before shutting down the HTTP server.
// This gives load balancers time to deregister the instance during rolling
// deploys.
//
// Default: 0s in dev mode, 3s in production. Override with this option or via
// healthProbe.shutdownDrainDelay in piko.yaml.
//
// Takes delay (time.Duration) which specifies the drain delay duration.
//
// Returns Option which configures the shutdown drain delay.
func WithShutdownDrainDelay(delay time.Duration) Option {
	return func(c *Container) {
		c.ensureOverrides().HealthProbe.ShutdownDrainDelay = new(int(delay.Seconds()))
	}
}

// WithServerConfigDefaults sets the default server settings for a Container.
//
// Takes serverConfigDefaults (*config.ServerConfig) which provides the default
// values for server settings.
//
// Returns Option which applies the server defaults to a Container.
func WithServerConfigDefaults(serverConfigDefaults *config.ServerConfig) Option {
	return func(c *Container) {
		c.configServerDefaults = serverConfigDefaults
	}
}

// WithFrontendModule enables a built-in frontend module across the whole site.
// Built-in modules include Analytics, Modals, and Toasts.
//
// Takes module (FrontendModule) which specifies the built-in module to enable.
// Takes moduleConfig (...any) which provides an optional config struct:
//   - AnalyticsConfig for ModuleAnalytics
//   - ModalsConfig for ModuleModals
//   - ToastsConfig for ModuleToasts
//
// Returns Option which sets up the container to load the specified module.
func WithFrontendModule(module daemon_frontend.FrontendModule, moduleConfig ...any) Option {
	return func(c *Container) {
		var resolvedConfig any
		if len(moduleConfig) > 0 {
			resolvedConfig = moduleConfig[0]
		}
		c.AddFrontendModule(module, resolvedConfig)
	}
}

// WithCustomFrontendModule registers a custom frontend JavaScript module.
// The module will be served at /_piko/dist/ppframework.{name}.min.js and
// included in all pages.
//
// Takes name (string) which specifies the module name used in the URL.
// Takes content ([]byte) which contains the JavaScript module source code.
// Takes moduleConfig (...map[string]any) which passes settings to the module.
//
// Returns Option which sets up the container to include the custom module.
func WithCustomFrontendModule(name string, content []byte, moduleConfig ...map[string]any) Option {
	return func(c *Container) {
		var resolvedConfig map[string]any
		if len(moduleConfig) > 0 {
			resolvedConfig = moduleConfig[0]
		}
		c.AddCustomFrontendModule(name, content, resolvedConfig)
	}
}

// WithMetricsExporter configures a metrics exporter for the health probe
// server. When enabled, OTEL metrics are exposed at the configured MetricsPath
// (default: /metrics) on the health probe server (default port: 9090).
//
// The exporter integrates with the OTEL MeterProvider, so all metrics recorded
// through OTEL instrumentation will be available for scraping.
//
// This option should be used in conjunction with the health probe server. If
// the health probe is disabled, the metrics endpoint will not be available.
//
// Takes exporter (monitoring_domain.MetricsExporter) which is the metrics
// exporter to use.
//
// Returns Option which configures the container with the metrics exporter.
func WithMetricsExporter(exporter monitoring_domain.MetricsExporter) Option {
	return func(c *Container) {
		_, l := logger_domain.From(c.GetAppContext(), log)
		if exporter == nil {
			l.Error("WithMetricsExporter called with nil exporter")
			return
		}
		c.SetMetricsExporter(exporter)
		l.Internal("Metrics exporter enabled")
	}
}

// WithMonitoringAddress sets the address for the gRPC monitoring server.
//
// Takes addr (string) which specifies the address to listen on.
//
// Returns MonitoringOption which configures the monitoring service address.
func WithMonitoringAddress(addr string) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.Address = addr
	}
}

// WithMonitoringBindAddress sets the network address for the monitoring server
// to bind to.
//
// Takes addr (string) which specifies the address and port to listen on.
//
// Returns MonitoringOption which applies the bind address setting.
func WithMonitoringBindAddress(addr string) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.BindAddress = addr
	}
}

// WithMonitoringAutoNextPort enables automatic port selection for the
// monitoring server. When the configured port is already in use, the server
// tries consecutive ports up to 100 attempts.
//
// Takes enabled (bool) which controls whether auto-port selection is active.
//
// Returns MonitoringOption which configures auto-port selection.
func WithMonitoringAutoNextPort(enabled bool) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.AutoNextPort = enabled
	}
}

// WithMonitoring enables the monitoring subsystem for telemetry collection.
// When a transport is configured via WithMonitoringTransport, monitoring data
// (metrics, traces, system stats) is also available to remote clients.
//
// The monitoring service provides span processors and metric readers for
// use with the OpenTelemetry SDK, so traces and metrics can be captured
// without needing external backends like Jaeger or Prometheus.
//
// The service starts during application bootstrap and stops during graceful
// shutdown.
//
// Takes opts (...MonitoringOption) which provides optional settings:
//   - WithMonitoringAddress(":9091"): sets the transport listen port.
//   - WithMonitoringBindAddress("127.0.0.1"): sets the bind address.
//   - WithMonitoringTransport(factory): sets the transport (e.g. gRPC).
//
// Returns Option which configures the container with the monitoring service.
func WithMonitoring(opts ...MonitoringOption) Option {
	return func(c *Container) {
		monitoringConfig := monitoring_domain.ServiceConfig{
			Address:                   ":9091",
			BindAddress:               "127.0.0.1",
			MaxSpans:                  monitoring_domain.DefaultMaxSpans,
			MaxMetrics:                monitoring_domain.DefaultMaxMetrics,
			MaxMetricAge:              monitoring_domain.DefaultMaxMetricAge,
			MetricsCollectionInterval: monitoring_domain.DefaultMetricsCollectionInterval,
		}
		for _, opt := range opts {
			opt(&monitoringConfig)
		}

		serviceOpts := buildMonitoringServiceOpts(&monitoringConfig)
		factories := resolveMonitoringFactories(&monitoringConfig)
		service := monitoring_domain.NewService(monitoring_domain.MonitoringDeps{}, factories, serviceOpts...)

		c.SetMonitoringService(service)

		if monitoringConfig.ProfilingEnabled {
			controller := profiler.NewController()
			service.SetProfilingController(controller)
		}

		_, l := logger_domain.From(c.GetAppContext(), log)
		l.Internal("Monitoring service enabled",
			logger_domain.String("address", monitoringConfig.BindAddress+monitoringConfig.Address))
	}
}

// buildMonitoringServiceOpts constructs the service options slice from the
// monitoring configuration.
//
// Takes serviceConfig (*monitoring_domain.ServiceConfig) which provides the
// monitoring settings to convert into service options.
//
// Returns []monitoring_domain.ServiceOption which contains the resolved
// options.
func buildMonitoringServiceOpts(serviceConfig *monitoring_domain.ServiceConfig) []monitoring_domain.ServiceOption {
	serviceOpts := []monitoring_domain.ServiceOption{
		monitoring_domain.WithServiceAddress(serviceConfig.Address),
		monitoring_domain.WithServiceBindAddress(serviceConfig.BindAddress),
		monitoring_domain.WithServiceMaxSpans(serviceConfig.MaxSpans),
		monitoring_domain.WithServiceMaxMetrics(serviceConfig.MaxMetrics),
		monitoring_domain.WithServiceMaxMetricAge(serviceConfig.MaxMetricAge),
		monitoring_domain.WithServiceMetricsInterval(serviceConfig.MetricsCollectionInterval),
	}

	if serviceConfig.AutoNextPort {
		serviceOpts = append(serviceOpts, monitoring_domain.WithServiceAutoNextPort(serviceConfig.AutoNextPort))
	}

	if serviceConfig.TLS.Enabled() {
		serviceOpts = append(serviceOpts, monitoring_domain.WithServiceTLS(serviceConfig.TLS))
	}

	if serviceConfig.TransportFactory != nil {
		serviceOpts = append(serviceOpts, monitoring_domain.WithServiceTransportFactory(serviceConfig.TransportFactory))
	}

	return appendWatchdogServiceOpts(serviceOpts, serviceConfig)
}

// resolveMonitoringFactories returns the service factories from configuration
// or the default factories if none are configured.
//
// Takes serviceConfig (*monitoring_domain.ServiceConfig) which may contain
// custom factory overrides.
//
// Returns monitoring_domain.ServiceFactories which contains the resolved
// factories.
func resolveMonitoringFactories(serviceConfig *monitoring_domain.ServiceConfig) monitoring_domain.ServiceFactories {
	if serviceConfig.Factories != nil {
		return *serviceConfig.Factories
	}

	return monitoring_adapters.DefaultServiceFactories()
}

// appendWatchdogServiceOpts appends watchdog-related service options when
// watchdog configuration is present.
//
// Takes serviceOpts ([]monitoring_domain.ServiceOption) which is the existing
// options slice to extend.
// Takes serviceConfig (*monitoring_domain.ServiceConfig) which provides the
// watchdog settings.
//
// Returns []monitoring_domain.ServiceOption which is the extended options
// slice.
func appendWatchdogServiceOpts(
	serviceOpts []monitoring_domain.ServiceOption,
	serviceConfig *monitoring_domain.ServiceConfig,
) []monitoring_domain.ServiceOption {
	if serviceConfig.WatchdogConfig != nil {
		serviceOpts = append(serviceOpts, monitoring_domain.WithServiceWatchdogConfig(serviceConfig.WatchdogConfig))
	}

	if serviceConfig.WatchdogNotifier != nil {
		serviceOpts = append(serviceOpts, monitoring_domain.WithServiceWatchdogNotifier(serviceConfig.WatchdogNotifier))
	}

	if serviceConfig.WatchdogProfileUploader != nil {
		serviceOpts = append(serviceOpts, monitoring_domain.WithServiceWatchdogProfileUploader(serviceConfig.WatchdogProfileUploader))
	}

	return serviceOpts
}

// WithMonitoringTransport sets the transport factory for the monitoring
// service, enabling remote clients to connect via the given transport
// (e.g. gRPC).
//
// Example:
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringTransport(monitoring_transport_grpc.Transport()),
//	)
//
// Takes factory (monitoring_domain.TransportFactory) which creates the
// transport server.
//
// Returns MonitoringOption which configures the transport on the service.
func WithMonitoringTransport(factory monitoring_domain.TransportFactory) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.TransportFactory = factory
	}
}

// WithMonitoringOtelFactories overrides the default noop monitoring service
// factories with real OTEL SDK implementations. Use this with the factories
// from logger_otel_sdk.OtelServiceFactories() to enable SDK-backed span
// processing and metrics collection.
//
// Example:
//
//	piko.WithMonitoring(
//	    piko.WithMonitoringOtelFactories(logger_otel_sdk.OtelServiceFactories()),
//	)
//
// Takes factories (monitoring_domain.ServiceFactories) which provides the
// span processor and metrics collector factories.
//
// Returns MonitoringOption which configures the factories on the service.
func WithMonitoringOtelFactories(factories monitoring_domain.ServiceFactories) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.Factories = &factories
	}
}

// WithMonitoringProfiling enables the remote profiling gRPC service, allowing
// operators to toggle pprof on and off at runtime via the monitoring endpoint.
// Without this option, the ProfilingService is not registered and profiling
// cannot be controlled remotely.
//
// Returns MonitoringOption which enables the profiling service.
func WithMonitoringProfiling() MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.ProfilingEnabled = true
	}
}

// WatchdogOption configures the runtime watchdog.
type WatchdogOption func(*monitoring_domain.WatchdogConfig)

// WithMonitoringWatchdog enables the runtime watchdog that monitors heap
// memory, goroutine counts, and GC pressure, automatically capturing
// diagnostic profiles when anomalies are detected.
//
// Takes opts (...WatchdogOption) which configure thresholds and behaviour.
//
// Returns MonitoringOption which enables the watchdog on the service.
func WithMonitoringWatchdog(opts ...WatchdogOption) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		watchdogConfig := monitoring_domain.DefaultWatchdogConfig()
		for _, opt := range opts {
			opt(&watchdogConfig)
		}
		c.WatchdogConfig = &watchdogConfig
	}
}

// WithWatchdogHeapThresholdPercent sets the heap threshold as a fraction of
// GOMEMLIMIT (0.0-1.0). Default: 0.85.
//
// Takes percent (float64) which is the threshold fraction.
//
// Returns WatchdogOption which configures the heap threshold.
func WithWatchdogHeapThresholdPercent(percent float64) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.HeapThresholdPercent = percent
	}
}

// WithWatchdogHeapThresholdBytes sets the absolute heap threshold in bytes,
// used when GOMEMLIMIT is not configured. Default: 512 MiB.
//
// Takes thresholdBytes (uint64) which is the threshold in bytes.
//
// Returns WatchdogOption which configures the heap threshold.
func WithWatchdogHeapThresholdBytes(thresholdBytes uint64) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.HeapThresholdBytes = thresholdBytes
	}
}

// WithWatchdogGoroutineThreshold sets the goroutine count that triggers a
// goroutine profile capture. Default: 10,000.
//
// Takes threshold (int) which is the goroutine count threshold.
//
// Returns WatchdogOption which configures the goroutine threshold.
func WithWatchdogGoroutineThreshold(threshold int) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.GoroutineThreshold = threshold
	}
}

// WithWatchdogCheckInterval sets how often the watchdog evaluates runtime
// metrics.
//
// Shorter intervals detect anomalies faster at negligible CPU cost.
// Default: 500ms.
//
// Takes interval (time.Duration) which is the check period.
//
// Returns WatchdogOption which configures the check interval.
func WithWatchdogCheckInterval(interval time.Duration) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.CheckInterval = interval
	}
}

// WithWatchdogCooldown sets the minimum duration between consecutive profile
// captures for the same metric type. Default: 2 minutes.
//
// Takes duration (time.Duration) which is the cooldown period.
//
// Returns WatchdogOption which configures the cooldown.
func WithWatchdogCooldown(duration time.Duration) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.Cooldown = duration
	}
}

// WithWatchdogMaxProfilesPerType sets the maximum number of stored profiles
// per type (heap, goroutine).
//
// Oldest profiles are rotated out. Default: 5.
//
// Takes count (int) which is the maximum profile count per type.
//
// Returns WatchdogOption which configures profile rotation.
func WithWatchdogMaxProfilesPerType(count int) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.MaxProfilesPerType = count
	}
}

// WithWatchdogProfileDirectory sets the local directory for profile storage.
// Default: os.TempDir()/piko-watchdog.
//
// Takes directory (string) which is the directory path.
//
// Returns WatchdogOption which configures the profile directory.
func WithWatchdogProfileDirectory(directory string) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.ProfileDirectory = directory
	}
}

// WithWatchdogDeltaProfiling enables storing a baseline heap profile alongside
// each capture so the user can compute a diff between consecutive captures.
//
// Returns WatchdogOption which enables delta profiling.
func WithWatchdogDeltaProfiling() WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.DeltaProfilingEnabled = true
	}
}

// WithWatchdogRSSThresholdPercent sets the fraction of the cgroup memory limit
// above which RSS triggers a profile capture. Default: 0.85.
//
// Takes percent (float64) which is the threshold fraction (0.0-1.0).
//
// Returns WatchdogOption which configures the RSS threshold.
func WithWatchdogRSSThresholdPercent(percent float64) WatchdogOption {
	return func(c *monitoring_domain.WatchdogConfig) {
		c.RSSThresholdPercent = percent
	}
}

// WithWatchdogNotifier sets the notification delivery mechanism for watchdog
// events. When set, the watchdog sends notifications to external systems when
// thresholds are breached or errors occur.
//
// Takes notifier (monitoring_domain.WatchdogNotifier) which delivers event
// notifications.
//
// Returns MonitoringOption which configures the notifier on the service.
func WithWatchdogNotifier(notifier monitoring_domain.WatchdogNotifier) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.WatchdogNotifier = notifier
	}
}

// WithWatchdogProfileUploader sets the remote storage backend for watchdog
// profile uploads. When set, captured profiles are uploaded to the configured
// storage provider after being written to local disk.
//
// Takes uploader (monitoring_domain.WatchdogProfileUploader) which handles
// remote storage.
//
// Returns MonitoringOption which configures the uploader on the service.
func WithWatchdogProfileUploader(uploader monitoring_domain.WatchdogProfileUploader) MonitoringOption {
	return func(c *monitoring_domain.ServiceConfig) {
		c.WatchdogProfileUploader = uploader
	}
}

// WithProfilingPort sets the port for the pprof HTTP server.
//
// Takes port (int) which specifies the port number to listen on.
//
// Returns ProfilingOption which configures the pprof server port.
func WithProfilingPort(port int) ProfilingOption {
	return func(c *profiler.Config) {
		c.Port = port
	}
}

// WithProfilingBindAddress sets the network address for the pprof server to
// bind to.
//
// Takes addr (string) which specifies the bind address.
//
// Returns ProfilingOption which configures the pprof server bind address.
func WithProfilingBindAddress(addr string) ProfilingOption {
	return func(c *profiler.Config) {
		c.BindAddress = addr
	}
}

// WithProfilingBlockRate sets the block profiling rate. After calling
// runtime.SetBlockProfileRate, the profiler samples one blocking event per
// this many nanoseconds of blocking.
//
// Takes rate (int) which specifies the sampling rate in nanoseconds.
//
// Returns ProfilingOption which configures the block profile rate.
func WithProfilingBlockRate(rate int) ProfilingOption {
	return func(c *profiler.Config) {
		c.BlockProfileRate = rate
	}
}

// WithProfilingMutexFraction sets the mutex profiling fraction. On average
// 1/n mutex contention events are reported.
//
// Takes fraction (int) which specifies the sampling fraction.
//
// Returns ProfilingOption which configures the mutex profile fraction.
func WithProfilingMutexFraction(fraction int) ProfilingOption {
	return func(c *profiler.Config) {
		c.MutexProfileFraction = fraction
	}
}

// WithProfilingMemProfileRate sets the memory profiling sample rate.
//
// When rate is 0 (the default), the Go runtime default of 512 KiB sampling is
// used. Lower values capture smaller allocations but add overhead.
//
// Takes rate (int) which specifies the sampling rate in bytes.
//
// Returns ProfilingOption which configures the memory profile rate.
func WithProfilingMemProfileRate(rate int) ProfilingOption {
	return func(c *profiler.Config) {
		c.MemProfileRate = rate
	}
}

// WithProfilingRollingTrace enables a bounded in-memory rolling execution trace
// buffer for the profiling server. The most recent trace window can later be
// downloaded from /_piko/profiler/trace/recent.
//
// Returns ProfilingOption which enables rolling trace capture with safe
// defaults.
func WithProfilingRollingTrace() ProfilingOption {
	return func(c *profiler.Config) {
		c.EnableRollingTrace = true
		if c.RollingTraceMinAge == 0 {
			c.RollingTraceMinAge = profiler.DefaultRollingTraceMinAge
		}
		if c.RollingTraceMaxBytes == 0 {
			c.RollingTraceMaxBytes = profiler.DefaultRollingTraceMaxBytes
		}
	}
}

// WithProfilingRollingTraceMinAge sets the retention target for the rolling
// trace recorder. Implicitly enables rolling trace capture if not already
// enabled.
//
// Takes minAge (time.Duration) which specifies how much recent trace history to
// keep when rolling trace capture is enabled.
//
// Returns ProfilingOption which configures the rolling trace minimum age.
func WithProfilingRollingTraceMinAge(minAge time.Duration) ProfilingOption {
	return func(c *profiler.Config) {
		c.EnableRollingTrace = true
		c.RollingTraceMinAge = minAge
	}
}

// WithProfilingRollingTraceMaxBytes sets the memory budget hint for the rolling
// trace recorder. Implicitly enables rolling trace capture if not already
// enabled.
//
// Takes maxBytes (uint64) which specifies the approximate in-memory budget for
// the rolling trace buffer.
//
// Returns ProfilingOption which configures the rolling trace buffer size.
func WithProfilingRollingTraceMaxBytes(maxBytes uint64) ProfilingOption {
	return func(c *profiler.Config) {
		c.EnableRollingTrace = true
		c.RollingTraceMaxBytes = maxBytes
	}
}

// WithAutoMemoryLimit configures the Go runtime to set GOMEMLIMIT based on
// the container's cgroup memory limit.
//
// This prevents OOM kills in containerised deployments by making the
// garbage collector aware of the memory ceiling.
//
// The provider function is called during bootstrap and should return the
// limit that was applied (in bytes), or an error if detection failed.
//
// Takes provider (func() (int64, error)) which detects and applies the
// memory limit.
//
// Returns Option which configures automatic memory limit detection.
func WithAutoMemoryLimit(provider func() (int64, error)) Option {
	return func(c *Container) {
		c.autoMemoryLimitFunc = provider
	}
}

// WithProfiling enables the pprof HTTP debug server. The server exposes
// profiling endpoints at /_piko/debug/pprof/ on a dedicated port (default 6060).
//
// When enabled, block and mutex profiling rates are configured so that
// /_piko/debug/pprof/block and /_piko/debug/pprof/mutex return meaningful data.
//
// The server starts during application bootstrap and stops during graceful
// shutdown.
//
// Takes opts (...ProfilingOption) which provides optional settings:
//   - WithProfilingPort(6060): sets the HTTP listen port.
//   - WithProfilingBindAddress("localhost"): sets the bind address.
//   - WithProfilingBlockRate(1000): sets the block profile rate.
//   - WithProfilingMutexFraction(10): sets the mutex profile fraction.
//   - WithProfilingMemProfileRate(0): sets the memory profile sample rate.
//   - WithProfilingRollingTrace(): enables bounded rolling trace capture.
//   - WithProfilingRollingTraceMinAge(15 * time.Second): adjusts retained trace age.
//   - WithProfilingRollingTraceMaxBytes(16 * 1024 * 1024): adjusts trace buffer size.
//
// Returns Option which configures the container with profiling settings.
func WithProfiling(opts ...ProfilingOption) Option {
	return func(c *Container) {
		profilingConfig := profiler.Config{
			Port:                 profiler.DefaultPort,
			BindAddress:          profiler.DefaultBindAddress,
			BlockProfileRate:     profiler.DefaultBlockProfileRate,
			MutexProfileFraction: profiler.DefaultMutexProfileFraction,
			MemProfileRate:       profiler.DefaultMemProfileRate,
		}
		for _, opt := range opts {
			opt(&profilingConfig)
		}
		c.SetProfilingConfig(&profilingConfig)
	}
}

// WithGeneratorProfilingOutputDir sets the directory for captured profile
// files.
//
// Takes directory (string) which specifies the output directory path.
//
// Returns GeneratorProfilingOption which configures the output directory.
func WithGeneratorProfilingOutputDir(directory string) GeneratorProfilingOption {
	return func(c *profiler.Config) {
		c.OutputDir = directory
	}
}

// WithGeneratorProfilingBlockRate sets the block profiling rate for generator
// profiling.
//
// Takes rate (int) which specifies the sampling rate in nanoseconds.
//
// Returns GeneratorProfilingOption which configures the block profile rate.
func WithGeneratorProfilingBlockRate(rate int) GeneratorProfilingOption {
	return func(c *profiler.Config) {
		c.BlockProfileRate = rate
	}
}

// WithGeneratorProfilingMutexFraction sets the mutex profiling fraction for
// generator profiling.
//
// Takes fraction (int) which specifies the sampling fraction.
//
// Returns GeneratorProfilingOption which configures the mutex profile fraction.
func WithGeneratorProfilingMutexFraction(fraction int) GeneratorProfilingOption {
	return func(c *profiler.Config) {
		c.MutexProfileFraction = fraction
	}
}

// WithGeneratorProfilingMemProfileRate sets the memory profiling sample rate
// for generator profiling. 0 uses the Go runtime default.
//
// Takes rate (int) which specifies the sampling rate in bytes.
//
// Returns GeneratorProfilingOption which configures the memory profile rate.
func WithGeneratorProfilingMemProfileRate(rate int) GeneratorProfilingOption {
	return func(c *profiler.Config) {
		c.MemProfileRate = rate
	}
}

// WithGeneratorProfiling enables profiling for short-lived generator builds.
// CPU, trace, heap, block, mutex, goroutine, and allocs profiles are captured
// to disk in the specified output directory (default "./profiles").
//
// Unlike WithProfiling, this does not start an HTTP server. Instead, it wraps
// the build execution: profiling starts before the build and profiles are
// written when the build completes.
//
// Takes opts (...GeneratorProfilingOption) which provides optional settings:
//   - WithGeneratorProfilingOutputDir("./profiles"): sets the output directory.
//   - WithGeneratorProfilingBlockRate(1): sets the block profile rate.
//   - WithGeneratorProfilingMutexFraction(1): sets the mutex profile fraction.
//   - WithGeneratorProfilingMemProfileRate(4096): sets the memory profile rate.
//
// Returns Option which configures the container with generator profiling
// settings.
func WithGeneratorProfiling(opts ...GeneratorProfilingOption) Option {
	return func(c *Container) {
		profilingConfig := profiler.Config{
			OutputDir:            profiler.DefaultOutputDir,
			BlockProfileRate:     profiler.CaptureBlockProfileRate,
			MutexProfileFraction: profiler.CaptureMutexProfileFraction,
			MemProfileRate:       profiler.CaptureMemProfileRate,
		}
		for _, opt := range opts {
			opt(&profilingConfig)
		}
		c.SetGeneratorProfilingConfig(&profilingConfig)
	}
}

// WithDatabase registers a named database connection for use by the querier
// system and persistence adapters. When a database named DatabaseNameRegistry
// or DatabaseNameOrchestrator is registered, the bootstrap container uses the
// querier-based DAL adapters instead of the default otter in-memory backend
// for that subsystem.
//
// Takes name (string) which identifies the database for later retrieval.
// Takes registration (*DatabaseRegistration) which provides connection and
// migration configuration.
//
// Returns Option which registers the database with the container.
func WithDatabase(name string, registration *DatabaseRegistration) Option {
	return func(c *Container) {
		c.AddDatabase(name, registration)
	}
}

// WithComponents registers external component definitions from libraries.
//
// Use this to make UI component libraries from external Go modules available
// in your Piko templates. Components are checked at registration time:
//   - Tag names must contain a hyphen (Web Components specification)
//   - Tag names must not shadow standard HTML elements
//   - Duplicate registrations are rejected
//
// Takes components (...component_dto.ComponentDefinition) which are the
// component definitions to register.
//
// Returns Option which sets up the container with the external components.
func WithComponents(components ...component_dto.ComponentDefinition) Option {
	return func(c *Container) {
		c.externalComponents = append(c.externalComponents, components...)
	}
}

// WithResolver sets a custom module resolver for the container. This must be
// called before any services that depend on the resolver.
//
// Takes resolver (resolver_domain.ResolverPort) which is the resolver to use.
//
// Returns Option which configures the container with the given resolver.
func WithResolver(resolver resolver_domain.ResolverPort) Option {
	return func(c *Container) {
		c.SetResolverOverride(resolver)
	}
}

// WithSandboxFactory sets a custom sandbox factory for the container. This
// allows injection of mock sandboxes for testing filesystem operations.
//
// Takes factory (SandboxFactory) which creates sandboxes for filesystem access.
//
// Returns Option which configures the container with the given factory.
func WithSandboxFactory(factory SandboxFactory) Option {
	return func(c *Container) {
		c.sandboxFactory = factory
	}
}

// WithExperimentalPrerendering configures whether static HTML prerendering is
// enabled for template generation. This is an experimental feature.
//
// When enabled, fully-static template subtrees are rendered to HTML bytes at
// generation time, avoiding AST traversal at runtime.
// Email templates are never prerendered regardless of this setting.
//
// Takes enabled (bool) which specifies whether prerendering is active.
//
// Returns Option which configures the container's prerendering behaviour.
func WithExperimentalPrerendering(enabled bool) Option {
	return func(c *Container) {
		c.experimentalPrerendering = enabled
	}
}

// WithExperimentalCommentStripping configures whether HTML comments are
// stripped from the generated output. This is an experimental feature.
//
// When enabled, HTML comments (<!-- ... -->) are omitted from output.
//
// Takes enabled (bool) which specifies whether comment stripping is active.
//
// Returns Option which configures the container's comment stripping behaviour.
func WithExperimentalCommentStripping(enabled bool) Option {
	return func(c *Container) {
		c.experimentalCommentStripping = enabled
	}
}

// WithExperimentalDwarfLineDirectives configures whether generated Go code
// emits valid DWARF //line directives. This is an experimental feature.
//
// When enabled, the code generator emits "//line file:line" (no space) which
// the Go compiler processes and embeds into DWARF debug info, enabling
// debuggers like Delve to map breakpoints back to .pk source files.
// When disabled (default), the generator emits "// line file:line" (with a
// space) which is treated as a plain comment and has no DWARF effect.
//
// Takes enabled (bool) which specifies whether DWARF line directives are active.
//
// Returns Option which configures the container's DWARF line directive behaviour.
func WithExperimentalDwarfLineDirectives(enabled bool) Option {
	return func(c *Container) {
		c.experimentalDwarfLineDirectives = enabled
	}
}

// WithStandardLoader causes the type inspector to use the standard
// golang.org/x/tools/go/packages.Load instead of the faster
// quickpackages loader.
//
// This is slower but always stable, as it is maintained by the
// Go team. Useful as a fallback when quickpackages encounters
// issues with specific dependency configurations (e.g. complex
// CGo setups).
//
// Takes enabled (bool) which controls whether the standard
// loader is used.
//
// Returns Option which configures the container's package loader
// behaviour.
func WithStandardLoader(enabled bool) Option {
	return func(c *Container) {
		c.useStandardLoader = enabled
	}
}

// WithCSSTreeShaking enables CSS tree-shaking during scaffold generation.
//
// Takes enabled (bool) which controls whether CSS tree-shaking is active.
//
// Returns Option which configures the container's CSS tree-shaking behaviour.
func WithCSSTreeShaking(enabled bool) Option {
	return func(c *Container) {
		c.cssTreeShaking = enabled
	}
}

// WithCSSTreeShakingSafelist sets CSS class names to preserve during
// tree-shaking.
//
// Takes classes ([]string) which lists the CSS class names to preserve.
//
// Returns Option which configures the safelist on the container.
func WithCSSTreeShakingSafelist(classes []string) Option {
	return func(c *Container) {
		c.cssTreeShakingSafelist = append(c.cssTreeShakingSafelist, classes...)
	}
}

// WithCSSResetComplete selects the comprehensive legacy CSS reset instead of
// the simple default. The comprehensive reset includes element-level resets,
// typography defaults, heading sizes via theme variables, and focus-ring
// styles.
//
// Returns CSSResetOption which switches to the complete reset preset.
func WithCSSResetComplete() CSSResetOption {
	return func(resetConfig *cssResetConfig) {
		resetConfig.useComplete = true
	}
}

// WithCSSResetPKOverride replaces the default CSS reset with custom CSS
// content for PK files (pages, partials, emails).
//
// Takes css (string) which is the custom CSS reset content.
//
// Returns CSSResetOption which sets the custom CSS override.
func WithCSSResetPKOverride(css string) CSSResetOption {
	return func(resetConfig *cssResetConfig) {
		resetConfig.css = css
	}
}

// WithCSSReset enables the CSS reset for PK files (pages, partials, emails),
// defaulting to the simple reset (box-sizing, margin, and padding zeroing)
// unless overridden via WithCSSResetComplete or WithCSSResetPKOverride.
//
// Takes opts (...CSSResetOption) which provides optional settings:
//   - WithCSSResetComplete(): selects the comprehensive legacy reset.
//   - WithCSSResetPKOverride(css): overrides with custom CSS content.
//
// Returns Option which configures the container's CSS reset behaviour.
func WithCSSReset(opts ...CSSResetOption) Option {
	return func(c *Container) {
		resetConfig := cssResetConfig{}
		for _, opt := range opts {
			opt(&resetConfig)
		}

		switch {
		case resetConfig.css != "":
			c.cssResetCSS = resetConfig.css
		case resetConfig.useComplete:
			c.cssResetCSS = render_domain.DefaultCSSResetComplete
		default:
			c.cssResetCSS = render_domain.DefaultCSSResetSimple
		}
	}
}

// WithStartupBanner controls whether the startup information banner is
// displayed when the server starts. Defaults to true.
//
// Takes enabled (bool) which specifies whether the banner is shown.
//
// Returns Option which configures the startup banner setting.
func WithStartupBanner(enabled bool) Option {
	return func(c *Container) {
		c.startupBannerEnabled = &enabled
	}
}

// WithIAmACatPerson swaps the large pixel-art mascot in the startup banner
// for the small ASCII art version. Defaults to false.
//
// Returns Option which configures the mascot preference.
func WithIAmACatPerson() Option {
	return func(c *Container) {
		c.iAmACatPerson = new(true)
	}
}

// WithWatchMode controls whether file system watching for hot-reloading is
// enabled. This is typically derived from the run mode (dev -> true,
// prod -> false) and does not need to be set manually.
//
// Takes enabled (bool) which controls whether watch mode is active.
//
// Returns Option which configures the watch mode setting.
func WithWatchMode(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Build.WatchMode = &enabled
	}
}

// WithE2EMode controls whether E2E test pages and partials are included in
// the build. WARNING: Never enable in production.
//
// Takes enabled (bool) which controls whether E2E mode is active.
//
// Returns Option which configures the E2E mode setting.
func WithE2EMode(enabled bool) Option {
	return func(c *Container) {
		c.ensureOverrides().Build.E2EMode = &enabled
	}
}

// WithSEO provides the SEO configuration for sitemap and robots.txt
// generation. SEO is only active when this option is provided with an enabled
// configuration and a non-empty sitemap hostname.
//
// Takes seoConfig (config.SEOConfig) which specifies the SEO settings.
//
// Returns Option which configures the SEO service on the container.
func WithSEO(seoConfig config.SEOConfig) Option {
	return func(c *Container) {
		c.SetSEOConfig(seoConfig)
	}
}

// WithAssets provides the asset configuration including image/video profiles,
// screen breakpoints, and default densities for responsive images. These
// settings are used at compile time by the annotator for static asset analysis.
//
// Takes assetsConfig (config.AssetsConfig) which specifies the asset settings.
//
// Returns Option which configures asset profiles on the container.
func WithAssets(assetsConfig config.AssetsConfig) Option {
	return func(c *Container) {
		c.SetAssetsConfig(assetsConfig)
	}
}

// WithWebsiteConfig provides the website configuration programmatically,
// replacing the file-based config.json loading entirely. When set, the
// config.json file is not read.
//
// Takes websiteConfig (config.WebsiteConfig) which specifies the website
// settings including theme, favicons, fonts, and i18n.
//
// Returns Option which configures the container with the given website config.
func WithWebsiteConfig(websiteConfig config.WebsiteConfig) Option {
	return func(c *Container) {
		c.websiteConfigOverride = &websiteConfig
	}
}

// WithDevWidget enables the dev tools overlay widget in dev mode, providing
// at-a-glance system stats, build pipeline status, and provider information
// with no effect in production mode.
//
// Returns Option which enables the dev widget.
func WithDevWidget() Option {
	return func(c *Container) {
		c.devWidgetEnabled = true
	}
}

// WithDevHotreload enables automatic page refresh in dev mode, where the
// browser receives an SSE event that triggers a navigation reload when a
// rebuild completes, with no effect in production mode.
//
// Returns Option which enables dev hot-reload.
func WithDevHotreload() Option {
	return func(c *Container) {
		c.devHotreloadEnabled = true
	}
}

// WithSRI controls whether Subresource Integrity (SRI) hashes are added to
// script and link tags in rendered HTML, enabled by default but can be
// disabled for development environments where assets change frequently.
//
// Takes enabled (bool) which controls whether integrity attributes are emitted.
//
// Returns Option which configures the SRI setting.
func WithSRI(enabled bool) Option {
	return func(c *Container) {
		c.sriEnabled = &enabled
	}
}

// WithAuthProvider registers an authentication provider that Piko
// calls on every request to resolve the auth state. The resolved
// AuthContext is available via RequestData.Auth() in pages and
// ActionMetadata.Auth() in actions.
//
// Takes provider (daemon_dto.AuthProvider) which resolves auth state
// from HTTP requests.
//
// Returns Option which configures the auth provider.
func WithAuthProvider(provider daemon_dto.AuthProvider) Option {
	return func(c *Container) {
		c.authProvider = provider
	}
}

// WithBackendAnalytics registers one or more backend analytics
// collectors.
//
// Events are fired automatically for page views via middleware.
// Multiple collectors can be registered; each receives every event.
//
// Takes collectors (...analytics_domain.Collector) which handle event
// delivery.
//
// Returns Option which registers the collectors.
func WithBackendAnalytics(collectors ...analytics_domain.Collector) Option {
	return func(c *Container) {
		for _, collector := range collectors {
			c.AddAnalyticsCollector(collector)
		}
	}
}

// WithAuthGuard enables route-level authentication enforcement, where routes
// not listed in the public paths or prefixes require authentication and
// unauthenticated requests are redirected to the login path or handled by a
// custom callback.
//
// Requires WithAuthProvider to be set; ignored without it.
//
// Takes authGuardConfig (daemon_dto.AuthGuardConfig) which specifies public
// paths, login redirect, and optional custom handler.
//
// Returns Option which configures the auth guard.
func WithAuthGuard(authGuardConfig daemon_dto.AuthGuardConfig) Option {
	return func(c *Container) {
		c.authGuardConfig = &authGuardConfig
	}
}
