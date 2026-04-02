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
	"context"
	"fmt"
	"net/http"

	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
)

// devDaemonBuilder encapsulates the state and logic required to assemble the
// daemon in compiled development mode.
type devDaemonBuilder struct {
	// c is the dependency injection container for resolving services.
	c *Container

	// deps holds shared dependencies for building dev mode components.
	deps *Dependencies

	// orchestratorService manages container orchestration for dev mode.
	orchestratorService orchestrator_domain.OrchestratorService

	// coordinatorService handles build coordination between services.
	coordinatorService coordinator_domain.CoordinatorService

	// i18nService provides translation and localisation for templates.
	i18nService i18n_domain.Service

	// renderer handles template rendering for output generation.
	renderer render_domain.RenderService

	// renderRegistry holds the registry for render templates.
	renderRegistry render_domain.RegistryPort

	// store holds the manifest store view, created during the build process.
	store templater_domain.ManifestStoreView

	// templaterService provides template rendering for the application.
	templaterService templater_domain.TemplaterService

	// compiledRunner runs compiled templates for the templater and email services.
	compiledRunner templater_domain.ManifestRunnerPort

	// finalRouter is the HTTP handler that combines all routes for the dev server.
	finalRouter http.Handler

	// devEventBroadcaster sends build-complete events to connected browsers via
	// SSE.
	devEventBroadcaster *daemon_adapters.DevEventBroadcaster

	// devAPIHandler serves the /_piko/dev/api/* REST endpoints for the
	// dev tools overlay widget.
	devAPIHandler *daemon_adapters.DevAPIHandler
}

// build runs the full assembly process by calling a set of helper methods.
//
// Returns daemon_domain.DaemonService which is the fully assembled daemon
// ready for use.
// Returns error when any build step fails.
func (b *devDaemonBuilder) build(ctx context.Context) (daemon_domain.DaemonService, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Assembling daemon for COMPILED DEVELOPMENT mode...")

	if err := b.resolveServices(); err != nil {
		return nil, fmt.Errorf("resolving dev mode services: %w", err)
	}

	ensureTypeDefinitions(ctx, b.c)

	if err := b.buildTemplater(ctx); err != nil {
		return nil, fmt.Errorf("building dev mode templater: %w", err)
	}

	b.wireMonitoringInspectors()

	if b.c.IsDevWidgetEnabled() || b.c.IsDevHotreloadEnabled() {
		b.devEventBroadcaster = daemon_adapters.NewDevEventBroadcaster()
		shutdown.Register(b.c.GetAppContext(), "DevEventBroadcaster", func(_ context.Context) error {
			b.devEventBroadcaster.Close()
			return nil
		})
	}

	if b.c.IsDevWidgetEnabled() {
		systemCollector := monitoring_domain.NewSystemCollector()
		systemCollector.Start(b.c.GetAppContext())
		if b.devEventBroadcaster != nil {
			b.devEventBroadcaster.SetSystemStatsProvider(systemCollector)
			b.devEventBroadcaster.SetOrchestratorInspector(b.c.GetOrchestratorInspector())
		}
		b.devAPIHandler = daemon_adapters.NewDevAPIHandler(
			systemCollector,
			b.c.GetOrchestratorInspector(),
		)
		b.wireDevAPIOptionalDeps(ctx)
	}

	if err := b.buildRouter(ctx); err != nil {
		return nil, fmt.Errorf("building dev mode router: %w", err)
	}

	daemon, err := b.buildFinalDaemon(ctx)
	if err != nil {
		return nil, fmt.Errorf("building dev mode daemon: %w", err)
	}

	l.Internal("Compiled development daemon bootstrapped successfully.")
	return daemon, nil
}

// resolveServices populates the builder struct with all necessary service
// dependencies by requesting them from the DI container.
//
// Returns error when a required service cannot be fetched from the
// container.
func (b *devDaemonBuilder) resolveServices() (err error) {
	b.orchestratorService, err = b.c.GetOrchestratorService()
	if err != nil {
		return fmt.Errorf("failed to get orchestrator service for dev mode: %w", err)
	}
	b.c.ScheduleGCTasks()

	b.coordinatorService, err = b.c.GetCoordinatorService()
	if err != nil {
		return fmt.Errorf("failed to get coordinator service for dev mode: %w", err)
	}

	b.i18nService, err = b.c.GetI18nService()
	if err != nil {
		return fmt.Errorf("failed to get i18n service for dev mode: %w", err)
	}

	b.renderer = b.c.GetRenderer()
	b.renderRegistry = b.c.GetRenderRegistry()

	return nil
}

// wireMonitoringInspectors connects the orchestrator and registry inspectors
// to the monitoring service if it is enabled, then starts the service.
func (b *devDaemonBuilder) wireMonitoringInspectors() {
	wireMonitoringInspectors(b.c, b.renderRegistry)
}

// wireDevAPIOptionalDeps attaches optional monitoring providers to the dev API
// handler. Each provider is best-effort; failures are logged and the handler
// gracefully degrades to returning {"available": false} for that endpoint.
func (b *devDaemonBuilder) wireDevAPIOptionalDeps(ctx context.Context) {
	wireDevAPIOptionalDeps(ctx, b.c, b.devAPIHandler, b.devEventBroadcaster)
}

// buildTemplater constructs the templating engine. In dev mode, this involves
// loading the manifest from disk and setting up a non-caching compiled runner
// to ensure changes are always reflected.
//
// Returns error when the manifest provider cannot be created or the manifest
// fails to load.
func (b *devDaemonBuilder) buildTemplater(ctx context.Context) error {
	manifestProvider, err := createManifestProvider(ctx, b.c)
	if err != nil {
		return fmt.Errorf("failed to create manifest provider for dev mode: %w", err)
	}

	store, err := templater_adapters.NewManifestStore(
		ctx,
		manifestProvider,
		templater_adapters.WithBaseDir(deref(b.c.config.ServerConfig.Paths.BaseDir, ".")),
	)
	if err != nil {
		return fmt.Errorf("failed to load dev manifest (hint: run 'piko generate all' first): %w", err)
	}
	b.store = store

	defaultLocale := deref(b.c.config.ServerConfig.I18nDefaultLocale, "en")
	b.compiledRunner = templater_adapters.NewCompiledManifestRunner(b.store, b.i18nService, defaultLocale)
	b.templaterService = templater_domain.NewTemplaterService(b.compiledRunner, templater_adapters.NewDrivenRenderer(b.renderer), b.i18nService)
	b.c.SetEmailTemplateService(templater_domain.NewEmailTemplateService(b.compiledRunner, templater_adapters.NewDrivenRenderer(b.renderer)))
	fontEntries := []layouter_dto.FontEntry{
		{Family: fonts.NotoSansFamilyName, Weight: fontWeightNormal, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansRegularTTF},
		{Family: fonts.NotoSansFamilyName, Weight: fontWeightBold, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansBoldTTF},
	}
	fontMetrics, fontMetricsError := layouter_adapters.NewGoTextFontMetrics(fontEntries)
	if fontMetricsError != nil {
		return fmt.Errorf("failed to create font metrics for dev mode: %w", fontMetricsError)
	}
	b.c.SetPdfWriterService(pdfwriter_domain.NewPdfWriterService(
		pdfwriter_adapters.NewTemplateRunnerAdapter(b.compiledRunner),
		pdfwriter_adapters.NewLayouterAdapter(fontMetrics, &layouter_adapters.MockImageResolver{}),
		fontEntries,
		nil,
		fontMetrics,
	))
	return nil
}

// buildRouter builds the final http.Handler for the application.
//
// Returns error when the router cannot be built.
func (b *devDaemonBuilder) buildRouter(ctx context.Context) error {
	var devHandlers *devRouterHandlers
	if b.devEventBroadcaster != nil || b.devAPIHandler != nil {
		devHandlers = &devRouterHandlers{
			eventsBroadcaster: b.devEventBroadcaster,
			apiHandler:        b.devAPIHandler,
		}
		if b.c.IsDevWidgetEnabled() {
			devHandlers.previewHandler = daemon_adapters.NewDevPreviewHandler(
				b.store,
				b.compiledRunner,
				templater_adapters.NewDrivenRenderer(b.renderer),
				b.c.GetEmailTemplateService(),
				b.c.GetPdfWriterService(),
				b.renderRegistry,
			)
		}
	}

	finalRouter, err := buildRouter(ctx, b.deps, b.c, b.store, b.templaterService, true, devHandlers)
	if err != nil {
		return fmt.Errorf("failed to build dev router: %w", err)
	}
	b.finalRouter = finalRouter
	return nil
}

// buildFinalDaemon builds the final daemon service with all dev-specific
// parts, including the file watcher and live-reloading services.
//
// Returns daemon_domain.DaemonService which is the fully set up daemon.
// Returns error when the file system watcher cannot be started or when
// building dependencies fails.
func (b *devDaemonBuilder) buildFinalDaemon(ctx context.Context) (daemon_domain.DaemonService, error) {
	factory, err := b.c.GetSandboxFactory()
	if err != nil {
		return nil, fmt.Errorf("getting sandbox factory for dev mode watcher: %w", err)
	}

	return buildDaemonWithWatcher(ctx, "dev mode", factory, func(fsWatcher lifecycle_domain.FileSystemWatcher) (*daemon_domain.DaemonServiceDeps, error) {
		return b.buildFinalDaemonDeps(ctx, fsWatcher)
	})
}

// buildFinalDaemonDeps creates the daemon service dependencies' struct.
//
// Takes fsWatcher (lifecycle_domain.FileSystemWatcher) which monitors file
// system changes for hot reloading.
//
// Returns *daemon_domain.DaemonServiceDeps which contains all dependencies
// needed by the daemon service.
// Returns error when the health probe server fails to initialise.
//
// Spawns a goroutine to run initial tasks in the background. The goroutine
// runs until the app context is cancelled.
func (b *devDaemonBuilder) buildFinalDaemonDeps(ctx context.Context, fsWatcher lifecycle_domain.FileSystemWatcher) (*daemon_domain.DaemonServiceDeps, error) {
	_, l := logger_domain.From(ctx, log)
	seoService, err := b.c.GetSEOService()
	if err != nil {
		l.Internal("Failed to initialise SEO service, continuing without SEO features", logger_domain.Error(err))
		seoService = nil
	}

	healthServer, healthRouter, drainSignaller, err := setupHealthProbeServer(b.c)
	if err != nil {
		return nil, fmt.Errorf("failed to setup health probe server: %w", err)
	}

	lifecycleService, err := b.c.createLifecycleService(&lifecycleServiceConfig{
		PathsConfig:      NewLifecyclePathsConfig(&b.deps.ConfigProvider.ServerConfig),
		WatcherAdapter:   fsWatcher,
		DevEventNotifier: b.devEventBroadcaster,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create lifecycle service: %w", err)
	}

	appCtx := b.c.GetAppContext()
	if err := lifecycleService.Start(appCtx); err != nil {
		return nil, fmt.Errorf("failed to start lifecycle service: %w", err)
	}

	runInitialTasksInBackground(appCtx, l, lifecycleService)

	daemonConfig := NewDaemonConfig(&b.deps.ConfigProvider.ServerConfig)
	daemonConfig.DevelopmentMode = true
	daemonConfig.NetworkAutoNextPort = true
	daemonConfig.HealthAutoNextPort = true

	serverAdapter, err := b.setupTLSServerAdapter(ctx, daemonConfig)
	if err != nil {
		return nil, fmt.Errorf("setting up TLS server adapter: %w", err)
	}

	var tlsRedirectServer daemon_domain.ServerAdapter
	if daemonConfig.TLSRedirectHTTPPort != "" {
		tlsRedirectServer = daemon_adapters.NewDriverHTTPServerAdapter()
	}

	return &daemon_domain.DaemonServiceDeps{
		DaemonConfig:        daemonConfig,
		WatchMode:           b.deps.ConfigProvider.ServerConfig.Build.WatchMode,
		Server:              serverAdapter,
		OrchestratorService: b.orchestratorService,
		FinalRouter:         b.finalRouter,
		CoordinatorService:  b.coordinatorService,
		SEOService:          seoService,
		HealthServer:        healthServer,
		HealthRouter:        healthRouter,
		DrainSignaller:      drainSignaller,
		TLSRedirectServer:   tlsRedirectServer,
		OnServerBound:       b.c.OnServerBound(),
		OnHealthBound:       b.c.OnHealthBound(),
	}, nil
}

// setupTLSServerAdapter creates the server adapter from TLS configuration and
// registers the certificate loader for shutdown cleanup.
//
// Takes daemonConfig (daemon_domain.DaemonConfig) which provides the TLS
// settings.
//
// Returns daemon_domain.ServerAdapter which is the configured server adapter.
// Returns error when TLS initialisation fails.
func (b *devDaemonBuilder) setupTLSServerAdapter(ctx context.Context, daemonConfig daemon_domain.DaemonConfig) (daemon_domain.ServerAdapter, error) {
	serverAdapter, tlsCleanup, err := daemon_adapters.NewServerAdapterFromTLSConfig(
		ctx,
		daemonConfig.TLS,
		daemon_adapters.ServerPurposeMain,
		b.c.createSandbox,
	)
	if err != nil {
		return nil, fmt.Errorf("initialising server TLS: %w", err)
	}
	shutdown.Register(ctx, "TLSCertificateLoader", func(_ context.Context) error {
		return tlsCleanup()
	})
	return serverAdapter, nil
}

// buildDevDaemon builds a daemon service using the development strategy.
//
// Takes c (*Container) which provides the dependency injection container.
// Takes deps (*Dependencies) which supplies the required service dependencies.
//
// Returns daemon_domain.DaemonService which is the assembled daemon service.
// Returns error when the builder fails to construct the daemon.
func buildDevDaemon(ctx context.Context, c *Container, deps *Dependencies) (daemon_domain.DaemonService, error) {
	builder := &devDaemonBuilder{
		c:    c,
		deps: deps,
	}
	return builder.build(ctx)
}
