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
	"path/filepath"
	"time"

	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
)

// defaultShutdownDrainDelay is the fallback drain delay used when no
// explicit shutdown drain delay is configured.
const defaultShutdownDrainDelay = 3 * time.Second

// prodDaemonBuilder holds the state and logic needed to build the daemon for
// production mode.
type prodDaemonBuilder struct {
	// c is the dependency injection container that provides services and settings.
	c *Container

	// deps holds shared dependencies for building daemon components.
	deps *Dependencies

	// i18nService provides translation and localisation for templates.
	i18nService i18n_domain.Service

	// renderer provides template rendering for templater services.
	renderer render_domain.RenderService

	// renderRegistry holds the set of available renderers.
	renderRegistry render_domain.RegistryPort

	// store holds the manifest store view that is built during setup.
	store templater_domain.ManifestStoreView

	// templaterService renders templates for HTTP handlers.
	templaterService templater_domain.TemplaterService

	// cachingRunner runs manifests with AST caching enabled.
	cachingRunner templater_domain.ManifestRunnerPort

	// finalRouter is the HTTP handler that routes requests to the service.
	finalRouter http.Handler
}

// build assembles the production daemon by calling a series of helper methods.
// Follows the Extract Method pattern, with each step handled by a focused helper.
//
// Returns daemon_domain.DaemonService which is the fully assembled daemon.
// Returns error when any build step fails.
func (b *prodDaemonBuilder) build(ctx context.Context) (daemon_domain.DaemonService, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Assembling daemon for PRODUCTION mode...")

	if err := b.resolveServices(); err != nil {
		return nil, fmt.Errorf("resolving production services: %w", err)
	}

	if err := b.buildTemplater(ctx); err != nil {
		return nil, fmt.Errorf("building production templater: %w", err)
	}

	b.wireMonitoringInspectors()

	if err := b.buildRouter(ctx); err != nil {
		return nil, fmt.Errorf("building production router: %w", err)
	}

	daemon, err := b.buildFinalDaemon(ctx)
	if err != nil {
		return nil, fmt.Errorf("building production daemon: %w", err)
	}

	b.c.ScheduleGCTasks()

	l.Notice("Production daemon bootstrapped successfully.")
	return daemon, nil
}

// resolveServices populates the builder struct with all necessary service
// dependencies by requesting them from the DI container.
//
// Returns error when a required service cannot be fetched from the
// container.
func (b *prodDaemonBuilder) resolveServices() (err error) {
	b.i18nService, err = b.c.GetI18nService()
	if err != nil {
		return fmt.Errorf("failed to get i18n service for production: %w", err)
	}

	b.renderer = b.c.GetRenderer()
	b.renderRegistry = b.c.GetRenderRegistry()

	return nil
}

// wireMonitoringInspectors connects the orchestrator and registry inspectors
// to the monitoring service if it is enabled, then starts the service.
func (b *prodDaemonBuilder) wireMonitoringInspectors() {
	wireMonitoringInspectors(b.c, b.renderRegistry)
}

// buildTemplater creates the caching template engine for production.
//
// Returns error when the manifest provider, store, or AST cache service cannot
// be created.
func (b *prodDaemonBuilder) buildTemplater(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	manifestProvider, err := createManifestProvider(ctx, b.c)
	if err != nil {
		return fmt.Errorf("failed to create manifest provider for production: %w", err)
	}
	store, err := templater_adapters.NewManifestStore(
		ctx,
		manifestProvider,
		templater_adapters.WithBaseDir(deref(b.c.serverConfig.Paths.BaseDir, ".")),
	)
	if err != nil {
		return fmt.Errorf("failed to load and link production manifest: %w", err)
	}
	b.store = store

	defaultLocale := deref(b.c.serverConfig.I18nDefaultLocale, "en")
	compiledRunner := templater_adapters.NewCompiledManifestRunner(b.store, b.i18nService, defaultLocale)

	astCacheService, err := b.bootstrapASTCacheService(ctx)
	if err != nil {
		return fmt.Errorf("failed to bootstrap AST cache service for production: %w", err)
	}
	shutdown.Register(b.c.GetAppContext(), "ASTCacheService", func(ctx context.Context) error {
		astCacheService.Shutdown(ctx)
		return nil
	})
	b.cachingRunner = templater_adapters.NewCachingManifestRunner(compiledRunner, astCacheService)
	l.Internal("AST Caching decorator enabled for the manifest runner.")

	b.templaterService = templater_domain.NewTemplaterService(b.cachingRunner, templater_adapters.NewDrivenRenderer(b.renderer), b.i18nService)
	b.c.SetEmailTemplateService(templater_domain.NewEmailTemplateService(b.cachingRunner, templater_adapters.NewDrivenRenderer(b.renderer)))
	fontEntries := []layouter_dto.FontEntry{
		{Family: fonts.NotoSansFamilyName, Weight: fontWeightNormal, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansRegularTTF},
		{Family: fonts.NotoSansFamilyName, Weight: fontWeightBold, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansBoldTTF},
	}
	fontMetrics, fontMetricsError := layouter_adapters.NewGoTextFontMetrics(fontEntries)
	if fontMetricsError != nil {
		return fmt.Errorf("failed to create font metrics for production: %w", fontMetricsError)
	}
	b.c.SetPdfWriterService(pdfwriter_domain.NewPdfWriterService(
		pdfwriter_adapters.NewTemplateRunnerAdapter(b.cachingRunner),
		pdfwriter_adapters.NewLayouterAdapter(fontMetrics, &layouter_adapters.MockImageResolver{}),
		fontEntries,
		nil,
		fontMetrics,
	))
	return nil
}

// buildRouter creates the HTTP handler for the application.
//
// Returns error when the router cannot be built.
func (b *prodDaemonBuilder) buildRouter(ctx context.Context) error {
	finalRouter, err := buildRouter(ctx, b.deps, b.c, b.store, b.templaterService, false, nil)
	if err != nil {
		return fmt.Errorf("failed to build production router: %w", err)
	}
	b.finalRouter = finalRouter
	return nil
}

// buildFinalDaemon creates the final daemon service for production use.
//
// Returns daemon_domain.DaemonService which is the configured production daemon.
// Returns error when health probe setup or lifecycle service creation fails.
func (b *prodDaemonBuilder) buildFinalDaemon(ctx context.Context) (daemon_domain.DaemonService, error) {
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
		PathsConfig: NewLifecyclePathsConfig(&b.c.serverConfig),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create lifecycle service: %w", err)
	}

	appCtx := b.c.GetAppContext()
	if err := lifecycleService.Start(appCtx); err != nil {
		return nil, fmt.Errorf("failed to start lifecycle service: %w", err)
	}

	runInitialTasksInBackground(appCtx, l, lifecycleService)

	daemonConfig := NewDaemonConfig(&b.c.serverConfig)
	if b.c.serverConfig.HealthProbe.ShutdownDrainDelay == nil {
		daemonConfig.ShutdownDrainDelay = defaultShutdownDrainDelay
	}
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

	return daemon_domain.NewService(ctx, &daemon_domain.DaemonServiceDeps{
		DaemonConfig:      daemonConfig,
		WatchMode:         b.c.serverConfig.Build.WatchMode,
		Server:            serverAdapter,
		FinalRouter:       b.finalRouter,
		SEOService:        seoService,
		HealthServer:      healthServer,
		HealthRouter:      healthRouter,
		DrainSignaller:    drainSignaller,
		TLSRedirectServer: newTLSRedirectServerIfConfigured(daemonConfig),
		OnServerBound:     b.c.OnServerBound(),
		OnHealthBound:     b.c.OnHealthBound(),
	}), nil
}

// bootstrapASTCacheService creates the AST cache service based on config.
// The AST cache improves performance by storing the parsed component tree.
//
// Takes ctx (context.Context) which is the parent context for background
// worker goroutines.
//
// Returns ast_domain.ASTCacheService which is the configured cache service.
// Returns error when the cache service cannot be created.
func (b *prodDaemonBuilder) bootstrapASTCacheService(ctx context.Context) (ast_domain.ASTCacheService, error) {
	serverConfig := b.c.serverConfig
	astCacheConfig := ast_adapters.ASTCacheConfig{
		L1CacheCapacity: config.DefaultL1CacheCapacity,
		L1CacheTTL:      time.Duration(config.DefaultL1CacheTTLMinutes) * time.Minute,
		L2CacheBaseDir: filepath.Join(
			deref(serverConfig.Paths.BaseDir, "."),
			config.PikoInternalPath,
			config.L2CacheDirName,
		),
	}
	return ast_adapters.NewASTCacheService(ctx, astCacheConfig)
}

// buildProdDaemon is the entry point for the production strategy.
// It creates and runs a builder to assemble the daemon.
//
// Takes c (*Container) which provides the dependency injection container.
// Takes deps (*Dependencies) which holds the resolved dependencies.
//
// Returns daemon_domain.DaemonService which is the assembled daemon service.
// Returns error when the builder fails to assemble the daemon.
func buildProdDaemon(ctx context.Context, c *Container, deps *Dependencies) (daemon_domain.DaemonService, error) {
	builder := &prodDaemonBuilder{
		c:    c,
		deps: deps,
	}
	return builder.build(ctx)
}
