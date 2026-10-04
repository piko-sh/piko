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

// This file provides a lightweight, manual Dependency Injection (DI) container for the
// Piko application. It centralises the creation and wiring of all major services,
// ensuring they are initialised lazily and used as singletons.
//
// This container is designed for flexibility using the Functional Options pattern. The
// NewContainer constructor accepts a series of Option functions that can override default
// service implementations or configure how they are built. This approach avoids external
// DI frameworks while providing clean, testable, and highly maintainable service
// management.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"piko.sh/piko/internal/component/component_adapters"
	"piko.sh/piko/internal/component/component_domain"
	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/dispatcher/dispatcher_adapters"
	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_adapters"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/profiler"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/seo/seo_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/safedisk"
)

// Option configures a Container during initialisation.
type Option func(*Container)

const (
	// errCreateSandboxFactory is the error message format used when sandbox factory creation
	// fails.
	errCreateSandboxFactory = "failed to create sandbox factory: %w"

	// errCreateSourceSandbox is the error format used when sandbox creation fails.
	errCreateSourceSandbox = "failed to create source sandbox: %w"

	// logMessageAutoRegisteredShutdown is the log message used when a service is registered
	// for shutdown handling.
	logMessageAutoRegisteredShutdown = "Auto-registered shutdown for user-provided service"

	// logKeyService is the log key for the service name in shutdown messages.
	logKeyService = "service"

	// logKeyMethod is the log key for the shutdown method name.
	logKeyMethod = "method"

	// logKeyPath is the structured log key for file system paths.
	logKeyPath = "path"
)

// SandboxFactory is a function type that creates sandboxes. Inject a custom factory to
// use mock sandboxes for testing.
//
// Takes name (string) which identifies the sandbox instance.
// Takes baseDir (string) which specifies the root directory for the sandbox.
// Takes mode (safedisk.Mode) which defines the access permissions.
//
// Returns safedisk.Sandbox which provides controlled filesystem access.
// Returns error when sandbox creation fails.
type SandboxFactory func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error)

// RegistryMetadataCacheConfig configures the metadata cache for the Registry service.
type RegistryMetadataCacheConfig struct {
	// MaxWeight is the maximum cache size in bytes.
	MaxWeight uint64

	// TTL is the time-to-live for cache entries; 0 means entries never expire.
	TTL time.Duration

	// StatsEnabled enables the collection of cache statistics when true.
	StatsEnabled bool
}

// Container holds all singleton services and dependencies for the application. Fields are
// unexported to prevent direct modification; configure via Options passed to
// NewContainer.
//
// The state is grouped by concern into embedded sub-structs whose fields are promoted, so
// methods read and write them directly. All but containerLifecycle start from their zero
// values.
type Container struct {
	// containerOverrides holds the user-supplied service replacements.
	containerOverrides

	// containerServiceErrors holds the cached errors from lazy service creation.
	containerServiceErrors

	// containerFeatureServices holds the cached application feature services.
	containerFeatureServices

	// containerCoreServices holds the cached build and render pipeline services.
	containerCoreServices

	// containerLifecycle holds the application context, callbacks, and factory functions.
	containerLifecycle

	// containerProviders holds the registered providers and their default names.
	containerProviders

	// containerFeatureFlags holds the boolean toggles and state flags.
	containerFeatureFlags

	// containerSettings holds the option-supplied and resolved configuration.
	containerSettings

	// containerOnceGuards holds the sync.Once guards for lazy singleton initialisation.
	containerOnceGuards
}

// NewContainer creates a new dependency injection container.
//
// It sets sensible defaults and then applies any options provided. The returned
// container's serverConfig is empty until ConfigAndContainer resolves the With* option
// overrides into it.
//
// Takes opts (...Option) which are options to change behaviour.
//
// Returns *Container which is the configured dependency injection container.
func NewContainer(opts ...Option) *Container {
	c := &Container{}
	c.containerLifecycle = newContainerLifecycle()
	c.csrfSecretKeyProvider = func() []byte {
		return resolveCSRFSecret(deref(c.serverConfig.CSRFSecret, ""))
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// GetAppContext returns the application-wide context that lives until shutdown.
//
// Returns context.Context which stays active until the application shuts down.
func (c *Container) GetAppContext() context.Context {
	c.appCtxOnce.Do(func() {
		c.appCtx, c.appCancel = context.WithCancelCause(context.Background())
		shutdown.Register(c.appCtx, "AppCancel", func(_ context.Context) error {
			c.appCancel(errors.New("application shutting down"))
			return nil
		})
	})
	return c.appCtx
}

// GetServerConfig returns a pointer to the resolved server configuration held by the
// container.
//
// Returns *ServerConfig which provides access to network, security, paths, and other
// framework settings.
func (c *Container) GetServerConfig() *ServerConfig {
	return &c.serverConfig
}

// GetWebsiteConfig returns a pointer to the website configuration supplied via
// WithWebsiteConfig.
//
// Returns *config.WebsiteConfig which provides theme, fonts, favicons, and i18n metadata.
// Empty when WithWebsiteConfig was not called.
func (c *Container) GetWebsiteConfig() *config.WebsiteConfig {
	return &c.websiteConfig
}

// SetSEOProductionMode records whether SEO artefacts are being generated for a production
// run.
//
// Takes isProduction (bool) which is true for a production run.
func (c *Container) SetSEOProductionMode(isProduction bool) {
	c.seoProductionMode = &isProduction
}

// AddRouteSource registers a build-time route source that enumerates the concrete URLs
// for a page bound to a p-route-source directive.
//
// Takes source (seo_domain.RouteSource) which enumerates the URLs.
func (c *Container) AddRouteSource(source seo_domain.RouteSource) {
	c.routeSources = append(c.routeSources, source)
}

// GetSandboxFactory returns the cached safedisk.Factory built from the server
// configuration. The factory is created lazily on first call and reused for all
// subsequent calls, ensuring consistent path validation and sandbox mode across the
// application.
//
// Returns safedisk.Factory which creates sandboxes with the configured allowed paths and
// enabled/disabled mode.
// Returns error when the factory cannot be created from the server config.
func (c *Container) GetSandboxFactory() (safedisk.Factory, error) {
	c.sandboxFactoryOnce.Do(func() {
		serverConfig := c.serverConfig
		c.sandboxFactoryInstance, c.sandboxFactoryErr = safedisk.NewFactory(safedisk.FactoryConfig{
			Enabled:      deref(serverConfig.Security.Sandbox.Enabled, true),
			AllowedPaths: serverConfig.Security.Sandbox.AllowedPaths,
			CWD:          deref(serverConfig.Paths.BaseDir, "."),
		})
	})
	return c.sandboxFactoryInstance, c.sandboxFactoryErr
}

// IsDevWidgetEnabled reports whether the dev tools overlay widget is enabled.
//
// Returns bool which is true when the widget should be rendered in dev mode.
func (c *Container) IsDevWidgetEnabled() bool { return c.devWidgetEnabled }

// IsDevHotreloadEnabled reports whether the SSE hot-reload JS module is enabled.
//
// Returns bool which is true when automatic page refresh should be active in dev mode.
func (c *Container) IsDevHotreloadEnabled() bool { return c.devHotreloadEnabled }

// IsEmbeddedMode reports whether the container is configured with an embedded .piko
// folder for single-binary deployments.
//
// Returns bool which is true when the container serves data from an embedded filesystem.
func (c *Container) IsEmbeddedMode() bool { return c.embeddedPikoFS != nil }

// SetOnServerBound stores a callback to invoke when the main HTTP server binds to a port.
//
// Takes fn (func(address string)) which is the callback receiving the resolved listen
// address.
func (c *Container) SetOnServerBound(fn func(address string)) { c.onServerBound = fn }

// OnServerBound returns the stored server-bound callback, or nil.
//
// Returns func(address string) which is the callback, or nil if not set.
func (c *Container) OnServerBound() func(address string) { return c.onServerBound }

// SetOnHealthBound stores a callback to invoke when the health server binds to a port.
//
// Takes fn (func(address string)) which is the callback receiving the resolved listen
// address.
func (c *Container) SetOnHealthBound(fn func(address string)) { c.onHealthBound = fn }

// OnHealthBound returns the stored health-bound callback, or nil.
//
// Returns func(address string) which is the callback, or nil if not set.
func (c *Container) OnHealthBound() func(address string) { return c.onHealthBound }

// IsSRIEnabled reports whether Subresource Integrity hashes should be added to script and
// link tags. Returns true by default unless explicitly disabled via WithSRI(false).
//
// Returns bool which is true when SRI integrity attributes should be emitted.
func (c *Container) IsSRIEnabled() bool {
	if c.sriEnabled != nil {
		return *c.sriEnabled
	}
	return true
}

// SetCompilerDebugLogsEnabled overrides the default for compiler debug log files. Use
// this to disable debug log files in contexts like the LSP where they are not needed.
//
// Takes enabled (bool) which controls whether debug log files are written.
func (c *Container) SetCompilerDebugLogsEnabled(enabled bool) {
	c.compilerDebugLogsEnabled = &enabled
}

// IsStartupBannerEnabled returns whether the startup banner should be displayed. Defaults
// to true when not explicitly set.
//
// Returns bool which is true when the banner should be shown.
func (c *Container) IsStartupBannerEnabled() bool {
	if c.startupBannerEnabled != nil {
		return *c.startupBannerEnabled
	}
	return true
}

// IsIAmACatPerson returns whether the large pixel-art mascot should be replaced with the
// small ASCII art version. Defaults to false.
//
// Returns bool which is true when the small mascot should be used.
func (c *Container) IsIAmACatPerson() bool {
	if c.iAmACatPerson != nil {
		return *c.iAmACatPerson
	}
	return false
}

// IsCSSTreeShakingEnabled returns whether CSS tree-shaking is enabled.
//
// Returns bool which is true when CSS tree-shaking is active.
func (c *Container) IsCSSTreeShakingEnabled() bool {
	return c.cssTreeShaking
}

// GetCSSTreeShakingSafelist returns the CSS classes preserved during tree-shaking.
//
// Returns []string which lists CSS class names that are never removed.
func (c *Container) GetCSSTreeShakingSafelist() []string {
	return c.cssTreeShakingSafelist
}

// GetCSSResetCSS returns the resolved CSS reset content for PK files. When empty, no CSS
// reset should be included in theme CSS output.
//
// Returns string which is the CSS reset content, or empty when disabled.
func (c *Container) GetCSSResetCSS() string {
	return c.cssResetCSS
}

// IsExperimentalPrerenderingEnabled returns whether static HTML prerendering is enabled
// at generation time.
//
// Returns bool which is true when prerendering is active.
func (c *Container) IsExperimentalPrerenderingEnabled() bool {
	return c.experimentalPrerendering
}

// IsExperimentalCommentStrippingEnabled returns whether HTML comment stripping is enabled
// for generated output.
//
// Returns bool which is true when comment stripping is active.
func (c *Container) IsExperimentalCommentStrippingEnabled() bool {
	return c.experimentalCommentStripping
}

// IsExperimentalDwarfLineDirectivesEnabled returns whether valid DWARF //line directives
// are emitted in generated Go code.
//
// Returns bool which is true when DWARF line directives are active.
func (c *Container) IsExperimentalDwarfLineDirectivesEnabled() bool {
	return c.experimentalDwarfLineDirectives
}

// GetActionRegistry returns the action registry.
//
// Returns the global registry populated by auto-generated init() functions. Actions are
// discovered automatically during annotation and generate registry.go files that register
// actions via init().
//
// Returns map[string]daemon_adapters.ActionHandlerEntry which maps action names to their
// handler entries.
func (*Container) GetActionRegistry() map[string]daemon_adapters.ActionHandlerEntry {
	return daemon_adapters.GetGlobalActionRegistry()
}

// GetComponentRegistry returns the PKC component registry for deterministic tag lookup
// during template processing.
//
// The registry is lazily initialised on first access. It contains:
//   - External components registered via WithComponents()
//   - Local components discovered from the components/ folder
//
// Returns component_domain.ComponentRegistry which provides tag name lookups.
func (c *Container) GetComponentRegistry() component_domain.ComponentRegistry {
	c.componentRegistryOnce.Do(func() {
		c.componentRegistry = component_adapters.NewInMemoryRegistry()

		_, l := logger_domain.From(c.GetAppContext(), log)
		for _, definition := range c.externalComponents {
			if err := c.componentRegistry.Register(definition); err != nil {
				l.Error("Failed to register external component",
					logger_domain.String("tag_name", definition.TagName),
					logger_domain.Error(err),
				)
			}
		}

		if len(c.externalComponents) > 0 {
			l.Internal("Registered external components",
				logger_domain.Int("count", len(c.externalComponents)),
			)
		}

		c.discoverLocalComponents()
	})
	return c.componentRegistry
}

// GetMetricsExporter returns the metrics exporter, if configured.
// Returns nil if metrics export was not enabled.
//
// Returns monitoring_domain.MetricsExporter which provides the metrics handler.
func (c *Container) GetMetricsExporter() monitoring_domain.MetricsExporter {
	return c.metricsExporter
}

// SetMetricsExporter sets the metrics exporter. This is called during container
// initialisation when metrics are enabled.
//
// Takes exporter (monitoring_domain.MetricsExporter) which is the exporter to use.
func (c *Container) SetMetricsExporter(exporter monitoring_domain.MetricsExporter) {
	c.metricsExporter = exporter
}

// AddSpanProcessor appends an additional OTEL span processor for registration on the
// tracer provider during OTEL setup (called by WithSpanProcessor); nil processors are
// ignored.
//
// Takes p (monitoring_domain.SpanProcessor) which receives every finished span.
func (c *Container) AddSpanProcessor(p monitoring_domain.SpanProcessor) {
	if p == nil {
		return
	}
	c.extraSpanProcessors = append(c.extraSpanProcessors, p)
}

// GetSpanProcessors returns the additional span processors registered via
// WithSpanProcessor. Returns nil when none were registered.
//
// Returns []monitoring_domain.SpanProcessor for tracer provider registration.
func (c *Container) GetSpanProcessors() []monitoring_domain.SpanProcessor {
	return c.extraSpanProcessors
}

// SetQueryObserver sets the query observer the instrumented DBTX wrapper notifies after
// each database statement. Called by WithQueryObserver.
//
// Takes o (monitoring_domain.QueryObserver) which receives each observed database call.
func (c *Container) SetQueryObserver(o monitoring_domain.QueryObserver) {
	c.queryObserver = o
}

// GetQueryObserver returns the registered query observer, or nil when none was set.
//
// Returns monitoring_domain.QueryObserver passed to the instrumented DBTX wrappers.
func (c *Container) GetQueryObserver() monitoring_domain.QueryObserver {
	return c.queryObserver
}

// SetReadinessInfoKeyFilter sets the predicate that decides which provider info keys are
// too sensitive to egress off-box via readiness telemetry. Called by
// WithReadinessInfoKeyFilter; a nil filter restores the built-in default.
//
// Takes fn (func(string) bool) which reports true for a key that must be dropped.
func (c *Container) SetReadinessInfoKeyFilter(fn func(string) bool) {
	c.readinessInfoKeyFilter = fn
}

// GetReadinessInfoKeyFilter returns the registered readiness info key filter, or nil when
// none was set and the built-in default applies.
//
// Returns func(string) bool reporting true for a key that must be dropped off-box.
func (c *Container) GetReadinessInfoKeyFilter() func(string) bool {
	return c.readinessInfoKeyFilter
}

// GetMonitoringService returns the full monitoring service, if configured.
// Returns nil if monitoring was not enabled via WithMonitoring().
//
// Returns monitoring_domain.MonitoringService which provides gRPC access and OTEL
// integration.
func (c *Container) GetMonitoringService() monitoring_domain.MonitoringService {
	return c.monitoringService
}

// SetMonitoringService sets the full monitoring service. This is called by WithMonitoring
// during container initialisation.
//
// Takes service (monitoring_domain.MonitoringService) which is the service to use.
func (c *Container) SetMonitoringService(service monitoring_domain.MonitoringService) {
	c.monitoringService = service
}

// GetProfilingConfig returns the pprof server configuration, if set.
// Returns nil when profiling was not enabled via WithProfiling().
//
// Returns *profiler.Config which holds the server profiling settings.
func (c *Container) GetProfilingConfig() *profiler.Config {
	return c.profilingConfig
}

// GetProfilingAddress returns the address the profiling server bound to.
//
// Returns string which is empty when profiling is disabled or the server failed to start.
func (c *Container) GetProfilingAddress() string {
	return c.profilingServer.Address()
}

// SetProfilingConfig stores the pprof server configuration. This is called by
// WithProfiling during container initialisation.
//
// Takes profilingConfig (*profiler.Config) which provides the profiling settings.
func (c *Container) SetProfilingConfig(profilingConfig *profiler.Config) {
	c.profilingConfig = profilingConfig
}

// GetGeneratorProfilingConfig returns the generator profiling configuration, if set.
// Returns nil when generator profiling was not enabled via WithGeneratorProfiling().
//
// Returns *profiler.Config which holds the capture-to-disk settings.
func (c *Container) GetGeneratorProfilingConfig() *profiler.Config {
	return c.generatorProfilingConfig
}

// SetGeneratorProfilingConfig stores the generator profiling configuration. This is
// called by WithGeneratorProfiling during container initialisation.
//
// Takes profilingConfig (*profiler.Config) which provides the generator profiling
// settings.
func (c *Container) SetGeneratorProfilingConfig(profilingConfig *profiler.Config) {
	c.generatorProfilingConfig = profilingConfig
}

// GetOrchestratorInspector returns the orchestrator inspector for monitoring.
// Returns nil if the orchestrator has not been initialised.
//
// Returns orchestrator_domain.OrchestratorInspector which provides read-only task data.
func (c *Container) GetOrchestratorInspector() orchestrator_domain.OrchestratorInspector {
	return c.orchestratorInspector
}

// GetRegistryInspector returns the registry inspector for monitoring.
// Returns nil if the registry has not been initialised.
//
// Returns registry_domain.RegistryInspector which provides read-only artefact data.
func (c *Container) GetRegistryInspector() registry_domain.RegistryInspector {
	return c.registryInspector
}

// GetMonitoringHealthProbeService returns the health probe service adapted for
// monitoring.
//
// Returns monitoring_domain.HealthProbeService for gRPC health reporting.
// Returns nil when the health probe service is not available.
func (c *Container) GetMonitoringHealthProbeService() monitoring_domain.HealthProbeService {
	healthService, err := c.GetHealthProbeService()
	if err != nil || healthService == nil {
		return nil
	}
	return monitoring_adapters.NewHealthProbeAdapter(healthService)
}

// GetEmailDispatcher returns the email dispatcher, if configured.
// Returns nil if no email dispatcher has been set up.
//
// Returns email_domain.EmailDispatcherPort which provides email dispatch and DLQ access.
func (c *Container) GetEmailDispatcher() email_domain.EmailDispatcherPort {
	return c.emailDispatcher
}

// GetNotificationDispatcher returns the notification dispatcher, if configured.
// Returns nil if no notification dispatcher has been set up.
//
// Returns notification_domain.NotificationDispatcherPort which provides notification
// dispatch and DLQ access.
func (c *Container) GetNotificationDispatcher() notification_domain.NotificationDispatcherPort {
	return c.notificationDispatcher
}

// GetDispatcherInspector returns a dispatcher inspector that provides read-only access to
// email and notification dispatcher state and DLQs.
//
// Returns dispatcher_domain.DispatcherInspector which provides unified DLQ monitoring, or
// nil if neither dispatcher is configured.
func (c *Container) GetDispatcherInspector() dispatcher_domain.DispatcherInspector {
	if c.emailDispatcher == nil && c.notificationDispatcher == nil {
		return nil
	}
	return dispatcher_adapters.NewInspector(c.emailDispatcher, c.notificationDispatcher)
}

// GetRateLimiterInspector returns the rate limiter as a RateLimiterInspector, or nil if
// the rate limiter is not available.
//
// Returns ratelimiter_domain.RateLimiterInspector which provides rate limiter state
// inspection, or nil.
func (c *Container) GetRateLimiterInspector() ratelimiter_domain.RateLimiterInspector {
	limiter, err := c.GetRateLimiter()
	if err != nil || limiter == nil {
		return nil
	}
	return limiter
}

// StartMonitoringService starts the monitoring gRPC server if configured. This should be
// called after SetInspectors() has been called to wire the orchestrator and registry
// inspectors.
//
// If the server fails to start, an error is logged but the application continues
// (monitoring is optional observability).
//
// Spawns a goroutine that runs the monitoring gRPC server until the application context
// is cancelled. The server is registered for graceful shutdown.
func (c *Container) StartMonitoringService() {
	monitoringService := c.GetMonitoringService()
	if monitoringService == nil {
		return
	}

	appCtx := c.GetAppContext()
	appCtx, l := logger_domain.From(appCtx, log)

	go func() {
		if err := monitoringService.Start(appCtx); err != nil {
			if appCtx.Err() == nil {
				l.Error("Monitoring gRPC server failed",
					logger_domain.Error(err))
			}
		}
	}()

	shutdown.Register(appCtx, "MonitoringService", func(ctx context.Context) error {
		monitoringService.Stop(ctx)
		return nil
	})

	l.Internal(
		"Monitoring gRPC service started",
		logger_domain.String("address", monitoringService.Address()),
	)
}

// StartProfilingServer starts the pprof HTTP server if profiling was enabled via
// WithProfiling. It configures runtime block and mutex profiling rates, checks for
// problematic build flags, and starts the server in a background goroutine with graceful
// shutdown.
//
// This is a no-op when profiling is not configured.
func (c *Container) StartProfilingServer() {
	profilingConfig := c.GetProfilingConfig()
	if profilingConfig == nil {
		return
	}

	_, l := logger_domain.From(c.GetAppContext(), log)

	profiler.SetRuntimeRates(*profilingConfig)

	if warning := profiler.CheckBuildFlags(); warning != "" {
		l.Warn(warning)
	}

	server, err := profiler.StartServer(c.GetAppContext(), *profilingConfig)
	if err != nil {
		l.Error("Failed to start profiling server",
			logger_domain.Error(err))
		return
	}
	server.SetErrorHandler(func(err error) {
		l.Error("Profiling server error", logger_domain.Error(err))
	})
	c.profilingServer = server
	addr := server.Address()

	shutdown.Register(c.GetAppContext(), "ProfilingServer", func(ctx context.Context) error {
		return server.Shutdown(ctx)
	})

	base := "http://" + addr + profiler.BasePath

	noticeAttrs := []logger_domain.Attr{
		logger_domain.String("address", base+"/debug/pprof/"),
		logger_domain.Int("block_profile_rate", profilingConfig.BlockProfileRate),
		logger_domain.Int("mutex_profile_fraction", profilingConfig.MutexProfileFraction),
		logger_domain.Int("goroutine_count", profiler.GoroutineCount()),
	}
	if profilingConfig.EnableRollingTrace {
		noticeAttrs = append(noticeAttrs,
			logger_domain.String("rolling_trace_min_age", profilingConfig.RollingTraceMinAge.String()),
			logger_domain.String("rolling_trace_max_bytes", fmt.Sprintf("%.1f MiB", float64(profilingConfig.RollingTraceMaxBytes)/(1024*1024))),
		)
	}

	l.Notice("Profiling server started", noticeAttrs...)
	logProfilingEndpoints(c.GetAppContext(), base, addr, profilingConfig.EnableRollingTrace)
}

// logProfilingEndpoints logs the available pprof and profiler endpoints.
//
// Takes base (string) which is the base URL prefix for pprof endpoints.
// Takes addr (string) which is the listen address for status endpoints.
// Takes rollingTraceEnabled (bool) which controls whether the rolling trace endpoint is
// included.
func logProfilingEndpoints(ctx context.Context, base, addr string, rollingTraceEnabled bool) {
	_, l := logger_domain.From(ctx, log)

	internalAttrs := []logger_domain.Attr{
		logger_domain.String("cpu", base+"/debug/pprof/profile?seconds=30"),
		logger_domain.String("heap", base+"/debug/pprof/heap"),
		logger_domain.String("allocs", base+"/debug/pprof/allocs"),
		logger_domain.String("goroutine", base+"/debug/pprof/goroutine"),
		logger_domain.String("block", base+"/debug/pprof/block"),
		logger_domain.String("mutex", base+"/debug/pprof/mutex"),
		logger_domain.String("trace", base+"/debug/pprof/trace?seconds=5"),
		logger_domain.String("status", "http://"+addr+profiler.ProfilerStatusPath),
	}
	if rollingTraceEnabled {
		internalAttrs = append(internalAttrs,
			logger_domain.String("rolling_trace", "http://"+addr+profiler.RollingTracePath),
		)
	}

	l.Internal("Available pprof endpoints", internalAttrs...)
}

// StartGeneratorProfiling begins capture-to-disk profiling if generator profiling was
// enabled via WithGeneratorProfiling. It configures runtime rates, checks build flags,
// and starts capturing CPU and trace profiles.
//
// Returns func() which must be called (typically via defer) to stop profiling and write
// all profile files.
// Returns nil when generator profiling is not configured.
func (c *Container) StartGeneratorProfiling() func() {
	generatorProfilingConfig := c.GetGeneratorProfilingConfig()
	if generatorProfilingConfig == nil {
		return nil
	}

	_, l := logger_domain.From(c.GetAppContext(), log)

	profiler.SetRuntimeRates(*generatorProfilingConfig)

	if warning := profiler.CheckBuildFlags(); warning != "" {
		l.Warn(warning)
	}

	if generatorProfilingConfig.Sandbox == nil {
		sandbox, sandboxErr := c.createSandbox("profiler-capture", generatorProfilingConfig.OutputDir, safedisk.ModeReadWrite)
		if sandboxErr != nil {
			l.Warn("Failed to create profiler sandbox, using fallback",
				logger_domain.Error(sandboxErr))
		} else {
			generatorProfilingConfig.Sandbox = sandbox
		}
	}

	cleanup, err := profiler.StartCapture(*generatorProfilingConfig)
	if err != nil {
		l.Error("Failed to start generator profiling",
			logger_domain.Error(err))
		return nil
	}

	l.Notice("Generator profiling started",
		logger_domain.String("output_dir", generatorProfilingConfig.OutputDir),
		logger_domain.Int("goroutine_count", profiler.GoroutineCount()),
	)

	return func() {
		cleanup()
		l.Notice("Generator profiling complete",
			logger_domain.String("output_dir", generatorProfilingConfig.OutputDir),
		)
	}
}

// registryBlobsReadOnly reports whether registry blob storage resolves to the read-only
// embedded filesystem.
//
// True for an embedded boot with no writable storage provider registered (embeddedPikoFS
// is set) and no in-memory runtime store enabled. This mirrors getRegistryBlobProvider's
// fall-through to the embedded fs.FS. On-demand variant generation cannot persist in that
// case, so callers skip it and fall back to the baked source asset.
//
// Returns bool which is true when registry blob writes would fail (read-only).
func (c *Container) registryBlobsReadOnly() bool {
	return c.embeddedPikoFS != nil && len(c.storageProviders) == 0 && !c.inMemoryRuntimeStoreEnabled
}

// isProductionMode reports whether the container is running a compiled production build
// (runMode "prod"), as recorded by SetSEOProductionMode. It gates runtime work that is
// redundant in a compiled deployment, such as re-seeding external components from source
// that is not shipped in the production image.
//
// Returns bool which is true for a production run.
func (c *Container) isProductionMode() bool {
	return c.seoProductionMode != nil && *c.seoProductionMode
}

// applyAutoMemoryLimit calls the configured auto memory limit function to set GOMEMLIMIT
// based on the container's cgroup memory limit.
//
// This is a no-op when no auto memory limit provider is configured.
func (c *Container) applyAutoMemoryLimit(ctx context.Context) {
	if c.autoMemoryLimitFunc == nil {
		return
	}

	_, l := logger_domain.From(ctx, log)

	limit, err := c.autoMemoryLimitFunc()
	if err != nil {
		l.Warn("Auto memory limit detection skipped", logger_domain.Error(err))
		return
	}

	l.Info("Auto memory limit applied",
		logger_domain.String("GOMEMLIMIT", fmt.Sprintf("%d MiB", limit/(1024*1024))))
}

// ensureOverrides lazily initialises the configServerOverrides struct so that option
// functions can write individual fields without nil-checking.
//
// Returns *ServerConfig which provides the override settings to modify.
func (c *Container) ensureOverrides() *ServerConfig {
	if c.configServerOverrides == nil {
		c.configServerOverrides = new(ServerConfig)
	}
	return c.configServerOverrides
}

// discoverLocalComponents walks the components folder and registers all .pkc files in the
// component registry for deterministic tag lookup.
//
// Component tag names come from the filename without the .pkc extension. For example,
// "my-button.pkc" registers as tag name "my-button".
func (c *Container) discoverLocalComponents() {
	_, l := logger_domain.From(c.GetAppContext(), log)
	serverConfig := c.serverConfig
	componentsDir := deref(serverConfig.Paths.ComponentsSourceDir, "components")
	if componentsDir == "" {
		l.Internal("No components directory configured, skipping local component discovery")
		return
	}

	absDir := filepath.Join(deref(serverConfig.Paths.BaseDir, "."), componentsDir)

	if !validateComponentsDirectory(c.GetAppContext(), absDir) {
		return
	}

	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	registered, regErrors := c.walkAndRegisterComponents(absDir, baseDir)
	logComponentDiscoveryResults(c.GetAppContext(), registered, regErrors, componentsDir)
}

// walkAndRegisterComponents walks the directory and registers .pkc files.
//
// Takes absDir (string) which is the absolute path to the directory to walk.
// Takes baseDir (string) which is the base path for computing relative paths.
//
// Returns int which is the count of successfully registered components.
// Returns []string which contains error messages for any failed registrations.
func (c *Container) walkAndRegisterComponents(absDir, baseDir string) (int, []string) {
	_, l := logger_domain.From(c.GetAppContext(), log)
	var registered int
	var regErrors []string

	walkErr := filepath.WalkDir(absDir, func(absPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking component directory %q: %w", absPath, err)
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".pkc") {
			return nil
		}

		tagName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		relPath, _ := filepath.Rel(baseDir, absPath)

		definition := component_dto.ComponentDefinition{
			TagName:    tagName,
			SourcePath: relPath,
			IsExternal: false,
		}

		if regErr := c.componentRegistry.Register(definition); regErr != nil {
			regErrors = append(regErrors, fmt.Sprintf("%s: %v", tagName, regErr))
		} else {
			registered++
		}
		return nil
	})

	if walkErr != nil {
		l.Warn("Error walking components directory",
			logger_domain.String(logKeyPath, absDir),
			logger_domain.Error(walkErr))
	}

	return registered, regErrors
}

// contextCloser defines a service that can be closed with a context for timeout control.
// Services set via Set* or Add* methods are registered for shutdown if they implement the
// contract.
type contextCloser interface {
	// Close releases resources held by the service.
	//
	// Returns error when the close operation fails.
	Close(ctx context.Context) error
}

// contextShutdown provides a way to shut down a service with a context.
type contextShutdown interface {
	// Shutdown stops the service in a controlled way.
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Shutdown(ctx context.Context) error
}

// contextStopper defines a service that can be stopped with a context.
type contextStopper interface {
	// Stop signals the component to stop and release resources.
	//
	// Returns error when the shutdown fails.
	Stop(ctx context.Context) error
}

// validateComponentsDirectory checks that the components directory exists and is valid.
//
// Takes absDir (string) which is the absolute path to the components directory.
//
// Returns bool which is true if the directory is valid and discovery should proceed.
func validateComponentsDirectory(ctx context.Context, absDir string) bool {
	_, l := logger_domain.From(ctx, log)
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			l.Internal("Components directory does not exist, skipping discovery",
				logger_domain.String(logKeyPath, absDir))
			return false
		}
		l.Warn("Failed to stat components directory",
			logger_domain.String(logKeyPath, absDir),
			logger_domain.Error(err))
		return false
	}
	if !info.IsDir() {
		l.Warn("Components path is not a directory",
			logger_domain.String(logKeyPath, absDir))
		return false
	}
	return true
}

// logComponentDiscoveryResults logs the outcome of component discovery.
//
// Takes registered (int) which is the number of components registered.
// Takes regErrors ([]string) which lists any errors that occurred.
// Takes componentsDir (string) which is the directory that was scanned.
func logComponentDiscoveryResults(ctx context.Context, registered int, regErrors []string, componentsDir string) {
	_, l := logger_domain.From(ctx, log)
	if len(regErrors) > 0 {
		l.Warn("Some components failed to register",
			logger_domain.Int("failed_count", len(regErrors)),
			logger_domain.Strings("errors", regErrors))
	}

	if registered > 0 {
		l.Internal("Discovered and registered local components",
			logger_domain.Int("count", registered),
			logger_domain.String("dir", componentsDir))
	}
}

// defaultMetadataCacheProvider returns a no-op cache by default. Use
// WithMemoryRegistryCache to enable caching.
//
// Returns registry_domain.MetadataCache which is nil to disable caching.
func defaultMetadataCacheProvider() registry_domain.MetadataCache {
	return nil
}

// registerCloseableForShutdown registers a service for graceful shutdown if it uses a
// known shutdown interface, so user-provided services are cleaned up without the need for
// manual shutdown setup.
//
// The following shutdown patterns are checked (in order of priority):
//   - Close(context.Context) error
//   - Shutdown(context.Context) error
//   - Stop(context.Context) error
//   - io.Closer (Close() error)
//
// Does nothing if the service does not use any shutdown interface.
//
// Takes name (string) which identifies the service in shutdown logs.
// Takes service (any) which is the service to check for shutdown support.
func registerCloseableForShutdown(ctx context.Context, name string, service any) {
	if service == nil {
		return
	}

	_, l := logger_domain.From(ctx, log)
	shutdownName := fmt.Sprintf("%s-Override", name)

	if closer, ok := service.(contextCloser); ok {
		shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
			return closer.Close(ctx)
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Close(ctx)"))
		return
	}

	if shutdowner, ok := service.(contextShutdown); ok {
		shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
			return shutdowner.Shutdown(ctx)
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Shutdown(ctx)"))
		return
	}

	if stopper, ok := service.(contextStopper); ok {
		shutdown.Register(ctx, shutdownName, func(ctx context.Context) error {
			return stopper.Stop(ctx)
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Stop(ctx)"))
		return
	}

	if closer, ok := service.(io.Closer); ok {
		shutdown.Register(ctx, shutdownName, func(_ context.Context) error {
			return closer.Close()
		})
		l.Internal(logMessageAutoRegisteredShutdown,
			logger_domain.String(logKeyService, name),
			logger_domain.String(logKeyMethod, "Close()"))
		return
	}

	l.Trace("Service does not implement shutdown interface",
		logger_domain.String("service", name))
}

// resolveActionResponseCacheMaxBytes returns the operator-supplied byte cap on the
// action-response cache when set.
//
// Returns uint64 which is the resolved byte cap.
func (c *Container) resolveActionResponseCacheMaxBytes() uint64 {
	if c.actionResponseCacheMaxBytesOverride > 0 {
		return c.actionResponseCacheMaxBytesOverride
	}
	return defaultActionResponseCacheMaxBytes
}
