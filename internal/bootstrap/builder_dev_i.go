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
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_adapters"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/esbuild/compat"
	esbuildconfig "piko.sh/piko/internal/esbuild/config"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_adapters"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/fonts"
	"piko.sh/piko/internal/layouter/layouter_adapters"
	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/layouter/layouter_dto"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/templater/templater_adapters"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/safedisk"
)

// interpretedDaemonBuilder holds the state and logic needed to build the daemon
// in interpreted development mode.
type interpretedDaemonBuilder struct {
	// variantGenerator creates image variants on demand; nil disables this feature.
	variantGenerator daemon_domain.OnDemandVariantGenerator

	// registryService provides access to the template registry for builds and routing.
	registryService registry_domain.RegistryService

	// orchestratorService manages starting and stopping services for the daemon.
	orchestratorService orchestrator_domain.OrchestratorService

	// i18nService provides translation and locale handling for templates.
	i18nService i18n_domain.Service

	// renderer handles HTML template rendering for the daemon.
	renderer render_domain.RenderService

	// renderRegistry holds the registry used to flush data loaders.
	renderRegistry render_domain.RegistryPort

	// csrfService creates and checks CSRF tokens for forms.
	csrfService security_domain.CSRFTokenService

	// captchaService verifies captcha tokens; nil when captcha is disabled.
	captchaService captcha_domain.CaptchaServicePort

	// generatorService provides content generation capabilities.
	generatorService generator_domain.GeneratorService

	// coordinatorService manages project builds and caching for interpreted mode.
	coordinatorService coordinator_domain.CoordinatorService

	// symbolProvider holds the symbols used for template interpretation.
	symbolProvider any

	// runner runs manifest templates for page and email rendering.
	runner templater_domain.ManifestRunnerPort

	// latestStore holds the current manifest store used for hot-reloading routes.
	latestStore templater_domain.ManifestStoreView

	// templaterService provides template rendering with hot-swappable runners.
	templaterService templater_domain.TemplaterService

	// resolver provides module path lookup and module name retrieval.
	resolver resolver_domain.ResolverPort

	// routerManager sets up HTTP routes and reloads them when files change.
	routerManager daemon_domain.RouterManager

	// buildOrchestrator creates and installs interpreted runners from build results.
	buildOrchestrator *lifecycle_adapters.InterpretedBuildOrchestrator

	// devEventBroadcaster sends build-complete events to connected browsers via
	// SSE. Created at build time and shared between the lifecycle service and the
	// router.
	devEventBroadcaster *daemon_adapters.DevEventBroadcaster

	// devAPIHandler serves the /_piko/dev/api/* REST endpoints for the
	// dev tools overlay widget.
	devAPIHandler *daemon_adapters.DevAPIHandler

	// c is the dependency injection container for retrieving services.
	c *Container

	// deps holds external dependencies from the application layer.
	deps *Dependencies

	// interpreterPool provides a pool of interpreters for JIT compilation.
	// Set from Dependencies.InterpreterPool.
	interpreterPool templater_domain.InterpreterPoolPort

	// entryPoints stores the entry points found for the daemon.
	entryPoints []annotator_dto.EntryPoint

	// mu guards the builder's shared state during concurrent operations.
	mu sync.RWMutex
}

// build assembles the daemon by calling helper methods in sequence.
//
// Returns daemon_domain.DaemonService which is the fully assembled daemon
// ready for use.
// Returns error when service setup, template building, or the initial build
// fails.
func (b *interpretedDaemonBuilder) build(ctx context.Context) (daemon_domain.DaemonService, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Assembling daemon for INTERPRETED DEVELOPMENT mode (Alpha)...")

	if err := b.resolveServices(ctx); err != nil {
		return nil, fmt.Errorf("resolving interpreted mode services: %w", err)
	}

	ensureTypeDefinitions(ctx, b.c)

	b.prepareProviders(ctx)

	bridge := b.c.GetArtefactBridge()
	eventBus := b.c.GetEventBus()

	factory, err := b.c.GetSandboxFactory()
	if err != nil {
		return nil, fmt.Errorf("getting sandbox factory for interpreted build service: %w", err)
	}

	buildPathsConfig := b.buildLifecyclePathsConfig()
	buildService := lifecycle_adapters.NewBuildService(
		b.c.GetConfigProvider(),
		buildPathsConfig,
		b.registryService,
		b.orchestratorService,
		bridge,
		eventBus,
		b.renderer,
		b.resolver,
		b.c.externalComponents,
		factory,
	)

	l.Internal("Performing initial asset seeding and processing...")
	result, err := buildService.RunBuild(ctx)
	if err != nil {
		return nil, fmt.Errorf("initial asset seeding and processing failed: %w", err)
	}
	summary := lifecycle_adapters.FormatBuildSummary(result)
	_, _ = fmt.Fprint(os.Stderr, summary)

	if result.HasFailures() {
		return nil, fmt.Errorf("initial asset seeding failed with %d failed task(s)", result.TotalFailed)
	}
	l.Internal("Initial asset pipeline complete.")

	if err := b.buildTemplaterAndRunner(); err != nil {
		return nil, fmt.Errorf("building templater and runner for interpreted mode: %w", err)
	}

	b.wireMonitoringInspectors()

	b.buildRouter(ctx)

	if err := b.triggerInitialBuild(ctx); err != nil {
		return nil, fmt.Errorf("initial blocking JIT build failed: %w", err)
	}

	return b.buildFinalDaemon(ctx)
}

// resolveServices fills the builder with all needed service dependencies
// by requesting them from the DI container.
//
// Returns error when any service resolution or setup step fails.
func (b *interpretedDaemonBuilder) resolveServices(ctx context.Context) error {
	if err := b.resolveCoreServices(); err != nil {
		return fmt.Errorf("resolving core services: %w", err)
	}
	if err := b.setupCoordinatorOverrides(ctx); err != nil {
		return fmt.Errorf("setting up coordinator overrides: %w", err)
	}
	if err := b.resolveGeneratorAndCoordinator(); err != nil {
		return fmt.Errorf("resolving generator and coordinator: %w", err)
	}
	b.resolveRenderingServices()
	if err := b.resolveResolverService(); err != nil {
		return fmt.Errorf("resolving resolver service: %w", err)
	}

	return nil
}

// wireMonitoringInspectors connects the orchestrator and registry inspectors
// to the monitoring service if monitoring is enabled, then starts the service.
func (b *interpretedDaemonBuilder) wireMonitoringInspectors() {
	wireMonitoringInspectors(b.c, b.renderRegistry)
}

// resolveCoreServices resolves the core services needed for interpreted mode.
//
// Returns error when the registry, orchestrator, or i18n service cannot be
// obtained from the container.
func (b *interpretedDaemonBuilder) resolveCoreServices() error {
	var err error
	b.registryService, err = b.c.GetRegistryService()
	if err != nil {
		return fmt.Errorf("failed to get registry service for interpreted mode: %w", err)
	}
	b.orchestratorService, err = b.c.GetOrchestratorService()
	if err != nil {
		return fmt.Errorf("failed to get orchestrator service for interpreted mode: %w", err)
	}
	b.c.ScheduleGCTasks()
	b.i18nService, err = b.c.GetI18nService()
	if err != nil {
		return fmt.Errorf("failed to get i18n service for interpreted mode: %w", err)
	}
	return nil
}

// setupCoordinatorOverrides configures coordinator overrides before generator
// creation.
//
// CRITICAL: These overrides must be set BEFORE calling GetGeneratorService()
// because generator service creation triggers coordinator creation
// (sync.Once).
//
// Returns error when the interpreted annotator service or code emitter cannot
// be created.
func (b *interpretedDaemonBuilder) setupCoordinatorOverrides(ctx context.Context) error {
	if err := b.createInterpretedAnnotatorService(ctx); err != nil {
		return fmt.Errorf("failed to create interpreted annotator service: %w", err)
	}
	b.c.coordinatorFileHashCacheOverride = coordinator_adapters.NewMemoryFileHashCache()
	codeEmitter, err := b.c.GetCodeEmitter()
	if err != nil {
		return fmt.Errorf("failed to get code emitter for interpreted mode: %w", err)
	}
	b.c.coordinatorCodeEmitterOverride = codeEmitter
	b.c.coordinatorClientScriptEmitterOverride = b.c.createPKJSEmitter()
	return nil
}

// resolveGeneratorAndCoordinator fetches the generator and coordinator
// services from the container.
//
// Returns error when either service cannot be fetched from the container.
func (b *interpretedDaemonBuilder) resolveGeneratorAndCoordinator() error {
	var err error
	b.generatorService, err = b.c.GetGeneratorService()
	if err != nil {
		return fmt.Errorf("failed to get generator service for interpreted mode: %w", err)
	}
	b.coordinatorService, err = b.c.GetCoordinatorService()
	if err != nil {
		return fmt.Errorf("failed to get coordinator service for interpreted mode: %w", err)
	}
	return nil
}

// resolveRenderingServices sets up the renderer, render registry, and CSRF
// services from the container.
func (b *interpretedDaemonBuilder) resolveRenderingServices() {
	b.renderer = b.c.GetRenderer()
	b.renderRegistry = b.c.GetRenderRegistry()
	b.csrfService = b.c.GetCSRFService()
	captchaService, captchaErr := b.c.GetCaptchaService()
	if captchaErr != nil {
		_, l := logger_domain.From(context.Background(), log)
		l.Warn("Captcha service unavailable; captcha-protected actions will be rejected with 403",
			logger_domain.Error(captchaErr))
	}
	b.captchaService = captchaService
}

// resolveResolverService sets up the resolver service for the daemon builder.
//
// Returns error when the resolver cannot be obtained.
func (b *interpretedDaemonBuilder) resolveResolverService() error {
	var err error
	b.resolver, err = b.c.GetResolver()
	if err != nil {
		return fmt.Errorf("failed to get resolver for interpreted mode: %w", err)
	}
	return nil
}

// createInterpretedAnnotatorService creates a custom annotator service
// configured for interpreted mode with WARN-level compilation logging to
// reduce noise from frequent JIT compilations. This override must be set
// before the coordinator service is created since the coordinator depends on
// the annotator.
//
// Returns error when the resolver, reader, processor, or service instance
// cannot be created.
func (b *interpretedDaemonBuilder) createInterpretedAnnotatorService(ctx context.Context) error {
	resolver, err := b.c.GetResolver()
	if err != nil {
		return fmt.Errorf("getting resolver for annotator: %w", err)
	}

	fsReader, cssProcessor, err := b.createAnnotatorReaderAndProcessor(resolver)
	if err != nil {
		return fmt.Errorf("creating annotator reader and processor: %w", err)
	}

	annotatorService, err := b.createAnnotatorServiceInstance(ctx, resolver, fsReader, cssProcessor)
	if err != nil {
		return fmt.Errorf("creating annotator service instance: %w", err)
	}

	b.c.annotatorServiceOverride = annotatorService
	return nil
}

// createAnnotatorReaderAndProcessor creates the filesystem reader and CSS
// processor for the annotator.
//
// Takes resolver (resolver_domain.ResolverPort) which resolves CSS imports.
//
// Returns annotator_domain.FSReaderPort which reads source files from the
// sandboxed filesystem.
// Returns *annotator_domain.CSSProcessor which processes and minifies CSS.
// Returns error when sandbox factory or source sandbox creation fails.
func (b *interpretedDaemonBuilder) createAnnotatorReaderAndProcessor(
	resolver resolver_domain.ResolverPort,
) (annotator_domain.FSReaderPort, *annotator_domain.CSSProcessor, error) {
	serverConfig := b.c.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	sandboxFactory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		Enabled:      deref(serverConfig.Security.Sandbox.Enabled, true),
		AllowedPaths: serverConfig.Security.Sandbox.AllowedPaths,
		CWD:          baseDir,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("creating sandbox factory: %w", err)
	}
	sourceSandbox, err := sandboxFactory.Create("dev-i-source", baseDir, safedisk.ModeReadOnly)
	if err != nil {
		return nil, nil, fmt.Errorf("creating source sandbox: %w", err)
	}
	fsReader := generator_adapters.NewFSReader(sourceSandbox)
	cssProcessor := annotator_domain.NewCSSProcessor(
		esbuildconfig.LoaderLocalCSS,
		&esbuildconfig.Options{
			MinifyWhitespace:       true,
			MinifySyntax:           true,
			UnsupportedCSSFeatures: compat.Nesting,
		},
		resolver,
	)
	return fsReader, cssProcessor, nil
}

// createAnnotatorServiceInstance creates the annotator service with all
// dependencies.
//
// Takes ctx (context.Context) which carries the logging context.
// Takes resolver (resolver_domain.ResolverPort) which resolves references.
// Takes fsReader (annotator_domain.FSReaderPort) which reads files.
// Takes cssProcessor (*annotator_domain.CSSProcessor) which processes CSS.
//
// Returns annotator_domain.AnnotatorPort which is the configured service.
// Returns error when the type inspector cannot be obtained.
func (b *interpretedDaemonBuilder) createAnnotatorServiceInstance(
	ctx context.Context,
	resolver resolver_domain.ResolverPort,
	fsReader annotator_domain.FSReaderPort,
	cssProcessor *annotator_domain.CSSProcessor,
) (annotator_domain.AnnotatorPort, error) {
	typeInspectorManager, err := b.c.GetTypeInspectorManager()
	if err != nil {
		return nil, fmt.Errorf("getting type inspector: %w", err)
	}

	ctx, l := logger_domain.From(ctx, log)
	collectionService, err := b.c.GetCollectionService()
	if err != nil {
		l.Warn("Collection service not available, GetCollection() calls will fail",
			logger_domain.Error(err))
		collectionService = nil
	}

	return annotator_domain.NewAnnotatorService(ctx, &annotator_domain.AnnotatorServiceConfig{
		Resolver:            resolver,
		FSReader:            fsReader,
		TypeInspector:       annotator_domain.NewTypeInspectorBuilderAdapter(typeInspectorManager),
		CSSProcessor:        cssProcessor,
		PathsConfig:         NewAnnotatorPathsConfig(&b.c.config.ServerConfig),
		AssetsConfig:        b.c.GetAssetsConfig(),
		Cache:               annotator_adapters.NewComponentCache(),
		CompilationLogLevel: slog.LevelWarn,
		CollectionService:   collectionService,
		ComponentRegistry:   b.c.GetComponentRegistry(),
	})
}

// prepareProviders sets up the symbol provider and interpreter pool
// for the just-in-time compilation pipeline.
func (b *interpretedDaemonBuilder) prepareProviders(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	b.symbolProvider = b.deps.SymbolProvider
	b.interpreterPool = b.deps.InterpreterPool

	if b.interpreterPool == nil {
		l.Warn("No interpreter pool provided; interpreted mode may not function correctly")
	} else {
		l.Internal("Interpreter pool configured from provider")
	}
}

// buildTemplaterAndRunner creates the core JIT templating engine. It sets up
// the InterpretedBuildOrchestrator and an empty initial runner.
//
// Returns error when the absolute project root path cannot be found.
func (b *interpretedDaemonBuilder) buildTemplaterAndRunner() error {
	projectRoot, err := filepath.Abs(deref(b.deps.ConfigProvider.ServerConfig.Paths.BaseDir, "."))
	if err != nil {
		return fmt.Errorf("failed to get absolute project root: %w", err)
	}

	genPathsConfig := b.buildGeneratorPathsConfig()
	i18nLocale := deref(b.c.config.ServerConfig.I18nDefaultLocale, "en")

	orchFactory, orchFactoryErr := b.c.GetSandboxFactory()
	if orchFactoryErr != nil {
		return fmt.Errorf("getting sandbox factory for interpreted build orchestrator: %w", orchFactoryErr)
	}

	moduleName := b.resolver.GetModuleName()
	b.buildOrchestrator = lifecycle_adapters.NewInterpretedBuildOrchestrator(
		lifecycle_adapters.InterpretedBuildOrchestratorDeps{
			InterpreterPool:   b.interpreterPool,
			RegistryService:   b.registryService,
			I18nService:       b.i18nService,
			PathsConfig:       genPathsConfig,
			I18nDefaultLocale: i18nLocale,
			ModuleName:        moduleName,
			ProjectRoot:       projectRoot,
			SandboxFactory:    orchFactory,
		},
	)

	defaultLocale := deref(b.c.config.ServerConfig.I18nDefaultLocale, "en")
	b.runner = templater_adapters.NewInterpretedManifestRunner(
		b.i18nService,
		make(map[string]*templater_adapters.PageEntry),
		nil,
		defaultLocale,
	)

	b.templaterService = templater_domain.NewTemplaterService(
		b.runner,
		templater_adapters.NewDrivenRenderer(b.renderer),
		b.i18nService,
	)

	b.c.SetEmailTemplateService(templater_domain.NewEmailTemplateService(
		b.runner,
		templater_adapters.NewDrivenRenderer(b.renderer),
	))

	return b.setupPdfWriter()
}

// setupPdfWriter configures the PDF writer service with font entries and
// metrics for interpreted mode.
//
// Returns error when font metrics cannot be created.
func (b *interpretedDaemonBuilder) setupPdfWriter() error {
	fontEntries := []layouter_dto.FontEntry{
		{Family: fonts.NotoSansFamilyName, Weight: fontWeightNormal, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansRegularTTF},
		{Family: fonts.NotoSansFamilyName, Weight: fontWeightBold, Style: int(layouter_domain.FontStyleNormal), Data: fonts.NotoSansBoldTTF},
	}
	fontMetrics, fontMetricsError := layouter_adapters.NewGoTextFontMetrics(fontEntries)
	if fontMetricsError != nil {
		return fmt.Errorf("failed to create font metrics for interpreted mode: %w", fontMetricsError)
	}
	b.c.SetPdfWriterService(pdfwriter_domain.NewPdfWriterService(
		pdfwriter_adapters.NewTemplateRunnerAdapter(b.runner),
		pdfwriter_adapters.NewLayouterAdapter(fontMetrics, &layouter_adapters.MockImageResolver{}),
		fontEntries,
		nil,
		fontMetrics,
	))

	return nil
}

// buildRouter creates the router manager which handles dynamic route loading
// and prepares the final wrapped http.Handler.
func (b *interpretedDaemonBuilder) buildRouter(ctx context.Context) {
	b.setupDevEventBroadcaster()

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

	b.createVariantGenerator(ctx)

	b.routerManager = daemon_adapters.NewRouterManager(&daemon_adapters.RouterManagerConfig{
		RouterConfig:  b.buildRouterConfig(),
		RouteSettings: buildRouteSettings(&b.deps.ConfigProvider.ServerConfig),
		CSPConfig:     buildCSPRuntimeConfig(b.c, b.deps),
		Deps: &daemon_domain.HTTPHandlerDependencies{
			Templater: b.templaterService,
			Validator: b.c.GetValidator(),
		},
		CSRFService:      b.csrfService,
		SiteSettings:     &b.deps.ConfigProvider.WebsiteConfig,
		Actions:          b.c.GetActionRegistry(),
		CacheMiddleware:  nil,
		RegistryService:  b.registryService,
		VariantGenerator: b.variantGenerator,
		RouteProviders:   nil,
		AppRouter:        b.deps.AppRouter,
		AuthGuardConfig:  b.c.authGuardConfig,
		CaptchaService:   b.captchaService,
	})

	if closer, ok := b.routerManager.(interface{ Close() }); ok {
		shutdown.Register(b.c.GetAppContext(), "InterpretedRouterManager", func(_ context.Context) error {
			closer.Close()
			return nil
		})
	}
}

// setupDevEventBroadcaster creates the dev event broadcaster if dev widget or
// hot-reload mode is enabled, and registers it for shutdown cleanup.
func (b *interpretedDaemonBuilder) setupDevEventBroadcaster() {
	if b.c.IsDevWidgetEnabled() || b.c.IsDevHotreloadEnabled() {
		b.devEventBroadcaster = daemon_adapters.NewDevEventBroadcaster()
		shutdown.Register(b.c.GetAppContext(), "DevEventBroadcaster", func(_ context.Context) error {
			b.devEventBroadcaster.Close()
			return nil
		})
	}
}

// wireDevAPIOptionalDeps attaches optional monitoring providers to the dev API
// handler. Each provider is best-effort; failures are logged and the handler
// gracefully degrades to returning {"available": false} for that endpoint.
func (b *interpretedDaemonBuilder) wireDevAPIOptionalDeps(ctx context.Context) {
	wireDevAPIOptionalDeps(ctx, b.c, b.devAPIHandler, b.devEventBroadcaster)
}

// buildRouterConfig constructs the typed RouterConfig for the HTTP router.
//
// Returns *daemon_domain.RouterConfig which contains the assembled router
// settings.
func (b *interpretedDaemonBuilder) buildRouterConfig() *daemon_domain.RouterConfig {
	serverConfig := b.deps.ConfigProvider.ServerConfig

	securityHeaders := serverConfig.Security.Headers
	corpOverride := b.c.GetCrossOriginResourcePolicy()
	if corpOverride != "" {
		securityHeaders.CrossOriginResourcePolicy = &corpOverride
	}
	shValues := NewSecurityHeadersValues(&securityHeaders)

	routerConfig := NewRouterConfig(&serverConfig, shValues, b.c.GetReportingConfig())
	routerConfig.DisableHTTPCache = true
	if b.devEventBroadcaster != nil {
		routerConfig.DevEventsBroadcaster = b.devEventBroadcaster
	}
	if b.devAPIHandler != nil {
		routerConfig.DevAPIHandler = b.devAPIHandler
	}
	if b.c.IsDevWidgetEnabled() {
		if interpRunner, ok := b.runner.(*templater_adapters.InterpretedManifestRunner); ok {
			routerConfig.DevPreviewHandler = daemon_adapters.NewDevPreviewHandler(
				templater_adapters.NewInterpretedManifestStoreView(interpRunner),
				b.runner,
				templater_adapters.NewDrivenRenderer(b.renderer),
				b.c.GetEmailTemplateService(),
				b.c.GetPdfWriterService(),
				b.renderRegistry,
			)
		}
	}
	return routerConfig
}

// createVariantGenerator sets up the variant generator for creating image
// variants on demand. Variants include different formats such as WebP or
// different sizes, and are made when first requested.
func (b *interpretedDaemonBuilder) createVariantGenerator(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	capabilityService, err := b.c.GetCapabilityService()
	if err != nil {
		l.Warn("Failed to get capability service for on-demand variant generation",
			logger_domain.Error(err))
		b.variantGenerator = nil
		return
	}

	b.variantGenerator = daemon_domain.NewOnDemandVariantGenerator(
		b.registryService,
		capabilityService,
		daemon_domain.DefaultOnDemandGeneratorConfig(),
	)
	l.Internal("On-demand variant generator initialised successfully")
}

// buildFinalDaemon builds the final daemon service with all its parts,
// including the file watcher and live-reloading services.
//
// Returns daemon_domain.DaemonService which is the fully configured daemon.
// Returns error when the file system watcher cannot be started or when
// building the daemon parts fails.
func (b *interpretedDaemonBuilder) buildFinalDaemon(ctx context.Context) (daemon_domain.DaemonService, error) {
	factory, err := b.c.GetSandboxFactory()
	if err != nil {
		return nil, fmt.Errorf("getting sandbox factory for interpreted mode watcher: %w", err)
	}

	return buildDaemonWithWatcher(ctx, "interpreted mode", factory, func(fsWatcher lifecycle_domain.FileSystemWatcher) (*daemon_domain.DaemonServiceDeps, error) {
		return b.buildInterpretedDaemonDeps(ctx, fsWatcher)
	})
}

// buildInterpretedDaemonDeps creates the daemon service dependencies' struct.
//
// Takes fsWatcher (lifecycle_domain.FileSystemWatcher) which monitors for file
// system changes.
//
// Returns *daemon_domain.DaemonServiceDeps which contains all configured
// dependencies for the daemon service.
// Returns error when health probe server setup fails.
func (b *interpretedDaemonBuilder) buildInterpretedDaemonDeps(ctx context.Context, fsWatcher lifecycle_domain.FileSystemWatcher) (*daemon_domain.DaemonServiceDeps, error) {
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

	lifecycleService, err := b.buildLifecycleService(fsWatcher)
	if err != nil {
		return nil, err
	}

	appCtx := b.c.GetAppContext()
	if err := lifecycleService.Start(appCtx); err != nil {
		return nil, fmt.Errorf("failed to start lifecycle service: %w", err)
	}

	daemonConfig := NewDaemonConfig(&b.deps.ConfigProvider.ServerConfig)
	daemonConfig.DevelopmentMode = true
	daemonConfig.NetworkAutoNextPort = true
	daemonConfig.HealthAutoNextPort = true

	serverAdapter, err := b.setupTLSServerAdapter(ctx, daemonConfig)
	if err != nil {
		return nil, fmt.Errorf("setting up TLS server adapter: %w", err)
	}

	return &daemon_domain.DaemonServiceDeps{
		DaemonConfig:        daemonConfig,
		WatchMode:           b.deps.ConfigProvider.ServerConfig.Build.WatchMode,
		Server:              serverAdapter,
		OrchestratorService: b.orchestratorService,
		FinalRouter:         b.routerManager,
		CoordinatorService:  b.coordinatorService,
		SEOService:          seoService,
		HealthServer:        healthServer,
		HealthRouter:        healthRouter,
		DrainSignaller:      drainSignaller,
		TLSRedirectServer:   newTLSRedirectServerIfConfigured(daemonConfig),
		OnServerBound:       b.c.OnServerBound(),
		OnHealthBound:       b.c.OnHealthBound(),
	}, nil
}

// setupTLSServerAdapter creates the server adapter from TLS configuration and
// registers the certificate loader for shutdown cleanup.
//
// Takes ctx (context.Context) which carries tracing and cancellation.
// Takes daemonConfig (daemon_domain.DaemonConfig) which provides the TLS
// settings.
//
// Returns daemon_domain.ServerAdapter which is the configured server adapter.
// Returns error when TLS initialisation fails.
func (b *interpretedDaemonBuilder) setupTLSServerAdapter(ctx context.Context, daemonConfig daemon_domain.DaemonConfig) (daemon_domain.ServerAdapter, error) {
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

// triggerInitialBuild performs a blocking initial build to ensure the service
// is ready before accepting HTTP requests.
//
// Returns error when the application context is cancelled, entry point
// discovery fails, the build fails, or route installation fails.
func (b *interpretedDaemonBuilder) triggerInitialBuild(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	appCtx := b.c.GetAppContext()
	l.Internal("Performing BLOCKING initial build via coordinator...")

	if appCtx.Err() != nil {
		l.Internal("Initial build cancelled; application is shutting down.")
		return appCtx.Err()
	}

	entryPoints, err := b.discoverAndStoreEntryPoints(ctx)
	if err != nil {
		return fmt.Errorf("discovering entry points: %w", err)
	}
	if len(entryPoints) == 0 {
		l.Warn("No entry points found. Server will start with no routes.")
		l.Internal("Service is now READY (no pages).")
		return nil
	}

	result, err := b.executeBuild(appCtx, entryPoints)
	if err != nil {
		return fmt.Errorf("executing initial build: %w", err)
	}

	return b.installRunnerAndRoutes(appCtx, result)
}

// discoverAndStoreEntryPoints finds entry points and stores them in the
// builder.
//
// Returns []annotator_dto.EntryPoint which contains the found entry points.
// Returns error when entry point discovery fails.
//
// Not safe for concurrent use; holds the builder's mutex during execution.
func (b *interpretedDaemonBuilder) discoverAndStoreEntryPoints(ctx context.Context) ([]annotator_dto.EntryPoint, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Discovering entry points...")
	b.mu.Lock()
	defer b.mu.Unlock()

	entryPoints, err := b.discoverInitialEntryPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover entry points: %w", err)
	}
	b.entryPoints = entryPoints
	l.Internal("Entry points discovered", logger_domain.Int("count", len(entryPoints)))
	return entryPoints, nil
}

// executeBuild triggers a build via the coordinator.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the entry
// points for the build.
//
// Returns *annotator_dto.ProjectAnnotationResult which contains the build
// output.
// Returns error when the coordinator build fails.
func (b *interpretedDaemonBuilder) executeBuild(ctx context.Context, entryPoints []annotator_dto.EntryPoint) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Requesting build from coordinator...")
	result, err := b.coordinatorService.GetOrBuildProject(ctx, entryPoints)
	if err != nil {
		l.Error("Initial coordinator build failed", logger_domain.Error(err))
		return nil, fmt.Errorf("initial build failed: %w", err)
	}
	l.Internal("Coordinator build completed successfully")
	return result, nil
}

// installRunnerAndRoutes builds the runner, installs it, and loads routes.
//
// Takes result (*annotator_dto.ProjectAnnotationResult) which provides the
// build items used to create the runner.
//
// Returns error when building the runner fails or loading routes fails.
func (b *interpretedDaemonBuilder) installRunnerAndRoutes(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Building interpreted runner from build artefacts...")
	newRunner, err := b.buildOrchestrator.BuildRunner(ctx, result)
	if err != nil {
		l.Error("Failed to build runner from initial build", logger_domain.Error(err))
		return fmt.Errorf("failed to build runner: %w", err)
	}

	b.templaterService.SetRunner(newRunner)
	b.runner = newRunner
	l.Internal("Interpreted runner built and installed")

	interpretedRunner, ok := newRunner.(*templater_adapters.InterpretedManifestRunner)
	if !ok {
		return errors.New("runner is not an InterpretedManifestRunner")
	}
	manifestStore := templater_adapters.NewInterpretedManifestStoreView(interpretedRunner)
	if b.routerManager != nil {
		l.Internal("Loading routes from manifest...")
		if err := b.routerManager.ReloadRoutes(ctx, manifestStore); err != nil {
			l.Error("Failed to load initial routes", logger_domain.Error(err))
			return fmt.Errorf("failed to load initial routes: %w", err)
		}
		l.Internal("Routes loaded successfully")
	}

	b.latestStore = manifestStore
	l.Internal("Initial build completed successfully",
		logger_domain.Int("page_count", len(manifestStore.GetKeys())))
	l.Internal("Service is now READY.")
	return nil
}

// discoverInitialEntryPoints walks all configured source directories to find
// .pk files that should be part of the build. It classifies them and
// produces EntryPoint structs with the Piko Module Path that the coordinator
// expects.
//
// Returns []annotator_dto.EntryPoint which contains the discovered entry
// points sorted for consistent ordering.
// Returns error when no resolver is provided or directory walking fails.
func (b *interpretedDaemonBuilder) discoverInitialEntryPoints(ctx context.Context) ([]annotator_dto.EntryPoint, error) {
	if b.resolver == nil {
		return nil, errors.New("cannot discover entry points: no resolver provided")
	}

	serverConfig := b.deps.ConfigProvider.ServerConfig
	entryPointMap := make(map[string]annotator_dto.EntryPoint)

	if err := b.discoverSourceDir(ctx, deref(serverConfig.Paths.PagesSourceDir, "pages"), true, false, entryPointMap); err != nil {
		return nil, fmt.Errorf("discovering pages source directory: %w", err)
	}
	if err := b.discoverSourceDir(ctx, deref(serverConfig.Paths.PartialsSourceDir, "partials"), false, false, entryPointMap); err != nil {
		return nil, fmt.Errorf("discovering partials source directory: %w", err)
	}
	if err := b.discoverSourceDir(ctx, deref(serverConfig.Paths.EmailsSourceDir, "emails"), false, true, entryPointMap); err != nil {
		return nil, fmt.Errorf("discovering emails source directory: %w", err)
	}

	return b.sortedEntryPoints(entryPointMap), nil
}

// discoverSourceDir walks a single source directory and populates the entry
// point map.
//
// Takes ctx (context.Context) which carries the logging context.
// Takes directory (string) which specifies the directory path relative to base.
// Takes isPage (bool) which indicates whether entries are page templates.
// Takes isEmail (bool) which indicates whether entries are email templates.
// Takes entryPointMap (map[string]annotator_dto.EntryPoint) which collects the
// discovered entry points.
//
// Returns error when walking the directory fails.
func (b *interpretedDaemonBuilder) discoverSourceDir(
	ctx context.Context,
	directory string,
	isPage, isEmail bool,
	entryPointMap map[string]annotator_dto.EntryPoint,
) error {
	if directory == "" {
		return nil
	}

	serverConfig := b.deps.ConfigProvider.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	absDir := filepath.Join(baseDir, directory)

	if !b.sourceDirectoryExists(ctx, directory, baseDir, absDir) {
		return nil
	}

	sfCtx := &sourceFileContext{
		baseDir:       baseDir,
		moduleName:    b.resolver.GetModuleName(),
		isPage:        isPage,
		isEmail:       isEmail,
		entryPointMap: entryPointMap,
	}
	return filepath.WalkDir(absDir, func(absPath string, d os.DirEntry, err error) error {
		return b.processSourceFile(ctx, absPath, d, err, sfCtx)
	})
}

// sourceDirectoryExists checks whether a source directory exists.
//
// Takes ctx (context.Context) which carries the logging context.
// Takes directory (string) which is the relative path to check.
// Takes baseDir (string) which is the base path for creating the sandbox.
// Takes absDir (string) which is the absolute path for logging.
//
// Returns bool which is true if the directory exists, false otherwise.
func (b *interpretedDaemonBuilder) sourceDirectoryExists(ctx context.Context, directory, baseDir, absDir string) bool {
	_, l := logger_domain.From(ctx, log)
	baseSandbox, sandboxErr := b.c.createSandbox("devmode-source-check", baseDir, safedisk.ModeReadOnly)
	if sandboxErr != nil {
		l.Warn("Failed to create sandbox for directory check", logger_domain.Error(sandboxErr))
		return false
	}
	_, statErr := baseSandbox.Stat(directory)
	_ = baseSandbox.Close()
	if errors.Is(statErr, fs.ErrNotExist) {
		l.Internal("Skipping discovery in non-existent directory", logger_domain.String("dir", absDir))
		return false
	}
	return true
}

// sourceFileContext holds context for processing a source file during discovery.
type sourceFileContext struct {
	// entryPointMap stores entry point details keyed by Piko module path.
	entryPointMap map[string]annotator_dto.EntryPoint

	// baseDir is the base directory for computing relative paths.
	baseDir string

	// moduleName is the Go module name used as a prefix for Piko paths.
	moduleName string

	// isPage indicates whether this source file is a page entry point.
	isPage bool

	// isEmail indicates whether this source file is an email template.
	isEmail bool
}

// processSourceFile handles a single file during directory walking.
//
// Takes ctx (context.Context) which carries the logging context.
// Takes absPath (string) which is the full path to the file.
// Takes d (os.DirEntry) which provides file information.
// Takes err (error) which is any error from the walk step.
// Takes sfCtx (*sourceFileContext) which holds the results.
//
// Returns error which is always nil; errors are logged but do not stop the
// walk.
func (*interpretedDaemonBuilder) processSourceFile(
	ctx context.Context,
	absPath string,
	d os.DirEntry,
	err error,
	sfCtx *sourceFileContext,
) error {
	_, l := logger_domain.From(ctx, log)
	if err != nil {
		l.Warn("Error during initial directory walk", logger_domain.Error(err), logger_domain.String("path", absPath))
		return nil
	}
	if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
		return nil
	}

	relPath, err := filepath.Rel(sfCtx.baseDir, absPath)
	if err != nil {
		l.Error("Could not compute relative path for entry point",
			logger_domain.String("baseDir", sfCtx.baseDir),
			logger_domain.String("absPath", absPath),
			logger_domain.Error(err))
		return nil
	}
	relPath = filepath.ToSlash(relPath)

	errResult := isErrorPage(d.Name())
	if !errResult.isErrorPage && strings.HasPrefix(d.Name(), "!") {
		return fmt.Errorf(
			"invalid error page filename %q: files starting with '!' must follow the error page convention. "+
				"Valid formats: !NNN.pk (e.g., !404.pk), !NNN-NNN.pk (e.g., !400-499.pk), or !error.pk (catch-all). "+
				"Only error pages may start with '!'",
			d.Name(),
		)
	}

	pikoPath := filepath.ToSlash(filepath.Join(sfCtx.moduleName, relPath))
	sfCtx.entryPointMap[pikoPath] = annotator_dto.EntryPoint{
		Path:               pikoPath,
		IsPage:             sfCtx.isPage && !errResult.isErrorPage,
		IsEmail:            sfCtx.isEmail,
		IsPublic:           sfCtx.isPage,
		IsErrorPage:        errResult.isErrorPage,
		ErrorStatusCode:    errResult.statusCode,
		ErrorStatusCodeMin: errResult.rangeMin,
		ErrorStatusCodeMax: errResult.rangeMax,
		IsCatchAllError:    errResult.isCatchAll,
	}
	return nil
}

// sortedEntryPoints converts the entry point map to a sorted slice.
//
// Takes entryPointMap (map[string]annotator_dto.EntryPoint) which contains the
// entry points keyed by their paths.
//
// Returns []annotator_dto.EntryPoint which contains the entry points sorted
// alphabetically by path.
func (*interpretedDaemonBuilder) sortedEntryPoints(entryPointMap map[string]annotator_dto.EntryPoint) []annotator_dto.EntryPoint {
	sortedPaths := slices.Sorted(maps.Keys(entryPointMap))

	entryPoints := make([]annotator_dto.EntryPoint, 0, len(entryPointMap))
	for _, path := range sortedPaths {
		entryPoints = append(entryPoints, entryPointMap[path])
	}
	return entryPoints
}

// buildLifecycleService constructs the lifecycle service, attaching
// the dev event broadcaster only when it has been created. Assigning
// a nil *DevEventBroadcaster straight into the DevEventNotifier
// interface would produce a typed-nil that passes `!= nil` checks and
// panics on first access.
//
// Takes fsWatcher (lifecycle_domain.FileSystemWatcher) which feeds
// file-change events into the service.
//
// Returns lifecycle_domain.LifecycleService ready for Start, or an
// error if creation fails.
func (b *interpretedDaemonBuilder) buildLifecycleService(fsWatcher lifecycle_domain.FileSystemWatcher) (lifecycle_domain.LifecycleService, error) {
	config := &lifecycleServiceConfig{
		PathsConfig:             b.buildLifecyclePathsConfig(),
		WatcherAdapter:          fsWatcher,
		RouterManager:           b.routerManager,
		TemplaterService:        b.templaterService,
		InterpretedOrchestrator: b.buildOrchestrator,
	}
	if b.devEventBroadcaster != nil {
		config.DevEventNotifier = b.devEventBroadcaster
	}
	service, err := b.c.createLifecycleService(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create lifecycle service: %w", err)
	}
	return service, nil
}

// buildLifecyclePathsConfig constructs a LifecyclePathsConfig by dereferencing
// the pointer fields from the server configuration.
//
// Returns lifecycle_domain.LifecyclePathsConfig which holds the resolved path
// values for file system operations.
func (b *interpretedDaemonBuilder) buildLifecyclePathsConfig() lifecycle_domain.LifecyclePathsConfig {
	paths := b.deps.ConfigProvider.ServerConfig.Paths
	return lifecycle_domain.LifecyclePathsConfig{
		BaseDir:             deref(paths.BaseDir, "."),
		PagesSourceDir:      deref(paths.PagesSourceDir, "pages"),
		PartialsSourceDir:   deref(paths.PartialsSourceDir, "partials"),
		ComponentsSourceDir: deref(paths.ComponentsSourceDir, "components"),
		EmailsSourceDir:     deref(paths.EmailsSourceDir, "emails"),
		AssetsSourceDir:     deref(paths.AssetsSourceDir, "lib"),
		I18nSourceDir:       deref(paths.I18nSourceDir, "locales"),
	}
}

// buildGeneratorPathsConfig constructs a GeneratorPathsConfig by dereferencing
// the pointer fields from the server configuration.
//
// Returns generator_domain.GeneratorPathsConfig which holds the resolved path
// values for the generator.
func (b *interpretedDaemonBuilder) buildGeneratorPathsConfig() generator_domain.GeneratorPathsConfig {
	paths := b.deps.ConfigProvider.ServerConfig.Paths
	return generator_domain.GeneratorPathsConfig{
		BaseDir:        deref(paths.BaseDir, "."),
		PagesSourceDir: deref(paths.PagesSourceDir, "pages"),
		E2ESourceDir:   deref(paths.E2ESourceDir, "e2e"),
		BaseServePath:  deref(paths.BaseServePath, ""),
	}
}

// buildDevInterpretedDaemon is the entry point for the interpreted development
// strategy. It creates a builder to assemble the daemon.
//
// Takes c (*Container) which provides the dependency injection container.
// Takes deps (*Dependencies) which holds the resolved dependencies.
//
// Returns daemon_domain.DaemonService which is the assembled daemon service.
// Returns error when the builder fails to construct the daemon.
func buildDevInterpretedDaemon(ctx context.Context, c *Container, deps *Dependencies) (daemon_domain.DaemonService, error) {
	builder := &interpretedDaemonBuilder{
		c:    c,
		deps: deps,
	}
	return builder.build(ctx)
}
