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

// This file is responsible for assembling the complete HTTP routing and
// middleware stack for the application. It orchestrates the process by
// resolving dependencies from the DI container, mounting application-specific
// routes, and wrapping the final handler in essential system-level middleware.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/internal/storage/storage_adapters/presign_http"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultActionMaxBodyBytes is the fallback limit for action request bodies (1 MiB).
	defaultActionMaxBodyBytes = int64(1_048_576)

	// defaultMaxMultipartFormBytes is the fallback limit for multipart form data (32 MiB).
	defaultMaxMultipartFormBytes = int64(33_554_432)

	// defaultMaxSSEDurationSeconds is the default maximum lifetime for SSE
	// connections (30 minutes), preventing indefinite connection holding while
	// being generous for long-running streams.
	defaultMaxSSEDurationSeconds = 1800

	// defaultActionResponseCacheMaxEntries is the maximum number of entries in
	// the action response cache. This bounds memory from cached action responses.
	defaultActionResponseCacheMaxEntries = 10_000

	// defaultActionResponseCacheMaxTTL is the upper-bound write expiration for
	// the action response cache. Individual actions override this with their
	// own TTL via SetExpiresAfter, but otter requires a non-nil
	// ExpiryCalculator for per-entry expiration to function.
	defaultActionResponseCacheMaxTTL = 1 * time.Hour

	// defaultArtefactMetadataCacheMaxEntries is the maximum number of entries
	// in the artefact metadata cache.
	defaultArtefactMetadataCacheMaxEntries = 500

	// defaultArtefactMetadataCacheTTL is how long artefact metadata stays
	// cached before expiring (5 minutes).
	defaultArtefactMetadataCacheTTL = 5 * time.Minute
)

var (
	// errUnknownManifestFormat is a specific error for invalid configuration.
	errUnknownManifestFormat = errors.New("invalid or unknown manifestFormat in config")

	// manifestProviderBuilders is a dispatch table mapping format strings to
	// builder functions. Each builder receives the manifest file path and an
	// optional sandbox for safe filesystem access.
	manifestProviderBuilders = map[string]manifestProviderBuilder{
		manifestFormatJSON: func(path string, sandbox safedisk.Sandbox) generator_domain.ManifestProviderPort {
			var opts []generator_adapters.JSONManifestProviderOption
			if sandbox != nil {
				opts = append(opts, generator_adapters.WithJSONManifestSandbox(sandbox))
			}
			return generator_adapters.NewJSONManifestProvider(path, opts...)
		},
		manifestFormatFlatbuffers: func(path string, sandbox safedisk.Sandbox) generator_domain.ManifestProviderPort {
			var opts []generator_adapters.FlatBufferManifestProviderOption
			if sandbox != nil {
				opts = append(opts, generator_adapters.WithFlatBufferManifestSandbox(sandbox))
			}
			return generator_adapters.NewFlatBufferManifestProvider(path, opts...)
		},
	}
)

// routerOperation holds state and dependencies for a single router build.
type routerOperation struct {
	// variantGenerator creates image variants on demand for responsive images.
	variantGenerator daemon_domain.OnDemandVariantGenerator

	// store provides read-only access to manifests for cache middleware and
	// routing.
	store templater_domain.ManifestStoreView

	// templaterService renders HTML templates for HTTP responses.
	templaterService templater_domain.TemplaterService

	// registryService provides access to image registry operations.
	registryService registry_domain.RegistryService

	// capabilityService provides feature detection during router operations.
	capabilityService capabilities.Service

	// renderRegistry provides template rendering for HTTP responses.
	renderRegistry render_domain.RegistryPort

	// csrfService creates and validates CSRF tokens.
	csrfService security_domain.CSRFTokenService

	// captchaService verifies captcha tokens; nil when captcha is disabled.
	captchaService captcha_domain.CaptchaServicePort

	// rateLimitService controls request rate limits for router middleware.
	rateLimitService security_domain.RateLimitService

	// devEventsBroadcaster serves the /_piko/dev/events SSE endpoint; nil in
	// production mode.
	devEventsBroadcaster http.Handler

	// devAPIHandler serves the /_piko/dev/api/* REST endpoints; nil in
	// production mode.
	devAPIHandler daemon_domain.DevAPIHandlerPort

	// devPreviewHandler serves the /_piko/dev/preview/* endpoints; nil in
	// production mode.
	devPreviewHandler daemon_domain.DevPreviewHandlerPort

	// container provides access to application services.
	container *Container

	// deps holds the external dependencies needed to build the router.
	deps *Dependencies

	// cspConfig holds the CSP settings defined at startup; these do not change.
	cspConfig security_dto.CSPRuntimeConfig

	// disableHTTPCache disables aggressive HTTP caching on static assets. Set
	// to true in dev mode so that ETags are revalidated on every request.
	disableHTTPCache bool
}

// execute runs the steps to build the router.
//
// Returns http.Handler which is the fully set up router ready to serve
// requests.
// Returns error when service resolution or router building fails.
func (op *routerOperation) execute(ctx context.Context) (http.Handler, error) {
	_, span, l := log.Span(ctx, "bootstrap.routerOperation.execute")
	defer span.End()

	l.Internal("Starting to build the main HTTP router...")

	if err := op.resolveServices(ctx); err != nil {
		return nil, fmt.Errorf("resolving router services: %w", err)
	}

	finalRouter, err := op.buildFinalRouter(ctx)
	if err != nil {
		return nil, fmt.Errorf("building final router: %w", err)
	}

	span.SetStatus(codes.Ok, "Router built successfully")
	l.Internal("HTTP router built")
	return finalRouter, nil
}

// resolveServices fetches all required service dependencies from the DI
// container and sets up the variant generator.
//
// Returns error when any required service cannot be resolved.
func (op *routerOperation) resolveServices(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	var err error
	op.registryService, err = op.container.GetRegistryService()
	if err != nil {
		return fmt.Errorf("failed to get registry service for router: %w", err)
	}
	op.capabilityService, err = op.container.GetCapabilityService()
	if err != nil {
		return fmt.Errorf("failed to get capability service for router: %w", err)
	}

	op.variantGenerator = daemon_domain.NewOnDemandVariantGenerator(
		op.registryService,
		op.capabilityService,
		daemon_domain.DefaultOnDemandGeneratorConfig(),
	)
	l.Internal("On-demand variant generator initialised successfully")

	op.renderRegistry = op.container.GetRenderRegistry()
	op.csrfService = op.container.GetCSRFService()
	captchaService, captchaErr := op.container.GetCaptchaService()
	if captchaErr != nil {
		l.Warn("Captcha service unavailable; captcha-protected actions will be rejected with 403",
			logger_domain.Error(captchaErr))
	}
	op.captchaService = captchaService

	op.rateLimitService, err = op.container.GetRateLimitService()
	if err != nil {
		return fmt.Errorf("failed to get rate limit service for router: %w", err)
	}

	l.Internal("All core services for router resolved successfully.")
	return nil
}

// buildFinalRouter builds the complete HTTP handler with all middleware and
// routes.
//
// Returns http.Handler which is the fully set up router ready for use.
// Returns error when the router fails to build.
func (op *routerOperation) buildFinalRouter(ctx context.Context) (http.Handler, error) {
	ctx, l := logger_domain.From(ctx, log)

	cacheMiddleware := op.createCacheMiddleware(ctx)
	op.mountApplicationRoutes(ctx, cacheMiddleware)

	presignUploadHandler, presignDownloadHandler, publicDownloadHandler := op.getPresignHandlers(ctx)

	var artefactMetaCache cache_domain.Cache[string, *registry_dto.ArtefactMeta]
	if cacheService, err := op.container.GetCacheService(); err == nil {
		artefactMetaCache, err = cache_domain.NewCacheBuilder[string, *registry_dto.ArtefactMeta](cacheService).
			FactoryBlueprint("artefact-metadata").
			Namespace("artefact-metadata").
			MaximumSize(defaultArtefactMetadataCacheMaxEntries).
			WriteExpiration(defaultArtefactMetadataCacheTTL).
			Build(ctx)
		if err != nil {
			l.Warn("Failed to create artefact metadata cache, metadata caching disabled",
				logger_domain.Error(err))
		}
	}

	builder := daemon_adapters.NewHTTPRouterBuilder(artefactMetaCache)
	finalRouter, err := builder.BuildRouter(
		op.buildRouterConfig(ctx),
		daemon_domain.RouterDependencies{
			RegistryService:        op.registryService,
			UserRouter:             op.deps.AppRouter,
			VariantGenerator:       op.variantGenerator,
			CSPConfig:              op.cspConfig,
			PresignUploadHandler:   presignUploadHandler,
			PresignDownloadHandler: presignDownloadHandler,
			PublicDownloadHandler:  publicDownloadHandler,
			RateLimitService:       op.rateLimitService,
			AuthProvider:           op.container.authProvider,
			AuthGuardConfig:        op.container.authGuardConfig,
			AnalyticsService:       op.container.GetAnalyticsService(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build the final top-level router: %w", err)
	}
	shutdown.Register(ctx, "RouterBuilder", func(_ context.Context) error {
		builder.Close()
		return nil
	})
	l.Internal("Top-level router with system middleware and routes has been built.")

	return finalRouter, nil
}

// createCacheMiddleware creates and returns the cache middleware.
//
// Returns *daemon_adapters.CacheMiddleware which handles caching for requests.
func (op *routerOperation) createCacheMiddleware(ctx context.Context) *daemon_adapters.CacheMiddleware {
	_, l := logger_domain.From(ctx, log)

	cacheMiddleware := daemon_adapters.NewCacheMiddleware(
		daemon_adapters.CacheMiddlewareConfig{CacheWriteConcurrency: 5},
		op.store,
		op.registryService,
		op.capabilityService,
		deref(op.deps.ConfigProvider.ServerConfig.Paths.PartialServePath, "/_piko/partials"),
	)
	l.Internal("Cache middleware initialised.")
	return cacheMiddleware
}

// mountApplicationRoutes mounts all application routes from the manifest.
//
// Takes cacheMiddleware (*daemon_adapters.CacheMiddleware) which provides
// caching for route handlers.
func (op *routerOperation) mountApplicationRoutes(ctx context.Context, cacheMiddleware *daemon_adapters.CacheMiddleware) {
	ctx, l := logger_domain.From(ctx, log)

	var actionResponseCache cache_domain.Cache[string, []byte]
	if cacheService, err := op.container.GetCacheService(); err == nil {
		actionResponseCache, err = cache_domain.NewCacheBuilder[string, []byte](cacheService).
			Namespace("action_responses").
			MaximumSize(defaultActionResponseCacheMaxEntries).
			WriteExpiration(defaultActionResponseCacheMaxTTL).
			Build(ctx)
		if err != nil {
			l.Warn("Failed to create action response cache, response caching disabled",
				logger_domain.Error(err))
		}
	}

	daemon_adapters.MountRoutesFromManifest(ctx, &daemon_adapters.MountRoutesConfig{
		Router: op.deps.AppRouter,
		Deps: &daemon_domain.HTTPHandlerDependencies{
			Templater: op.templaterService,
			Validator: op.container.GetValidator(),
		},
		Store:               op.store,
		CSRFService:         op.csrfService,
		RouteSettings:       buildRouteSettings(&op.deps.ConfigProvider.ServerConfig),
		SiteSettings:        &op.deps.ConfigProvider.WebsiteConfig,
		Actions:             op.container.GetActionRegistry(),
		CacheMiddleware:     cacheMiddleware.Handle,
		RateLimitService:    op.rateLimitService,
		RateLimitConfig:     NewRateLimitValues(&op.deps.ConfigProvider.ServerConfig.Security.RateLimit),
		AuthGuardConfig:     op.container.authGuardConfig,
		ActionResponseCache: actionResponseCache,
		CaptchaService:      op.captchaService,
	})
	l.Internal("All dynamic application routes from manifest have been mounted.")
}

// buildRouterConfig constructs the typed RouterConfig from the server
// configuration with any container-level overrides applied.
//
// Returns *daemon_domain.RouterConfig which contains exactly the fields the
// router needs.
func (op *routerOperation) buildRouterConfig(ctx context.Context) *daemon_domain.RouterConfig {
	_, l := logger_domain.From(ctx, log)

	serverConfig := op.deps.ConfigProvider.ServerConfig

	securityHeaders := serverConfig.Security.Headers
	corpOverride := op.container.GetCrossOriginResourcePolicy()
	if corpOverride != "" {
		securityHeaders.CrossOriginResourcePolicy = &corpOverride
		l.Internal("Applying CORP override from option",
			logger_domain.String("policy", corpOverride))
	}
	shValues := NewSecurityHeadersValues(&securityHeaders)

	routerConfig := NewRouterConfig(&serverConfig, shValues, op.container.GetReportingConfig())
	routerConfig.DisableHTTPCache = op.disableHTTPCache
	routerConfig.DevEventsBroadcaster = op.devEventsBroadcaster
	routerConfig.DevAPIHandler = op.devAPIHandler
	routerConfig.DevPreviewHandler = op.devPreviewHandler
	return routerConfig
}

// getPresignHandlers creates and returns presigned upload, download, and
// public download HTTP handlers if the storage service is available.
//
// Returns uploadHandler (http.Handler) which handles authenticated
// uploads, or nil if not available.
// Returns downloadHandler (http.Handler) which handles authenticated
// downloads, or nil if not available.
// Returns publicDownloadHandler (http.Handler) which handles public
// downloads, or nil if not available.
func (op *routerOperation) getPresignHandlers(ctx context.Context) (uploadHandler, downloadHandler, publicDownloadHandler http.Handler) {
	_, l := logger_domain.From(ctx, log)

	storageService, err := op.container.GetStorageService()
	if err != nil {
		l.Internal("Storage service not available, presigned uploads/downloads disabled",
			logger_domain.Error(err))
		return nil, nil, nil
	}

	presignConfig := op.getPresignConfig(storageService)

	publicDownloadHandler = presign_http.NewPublicDownloadHandler(storageService)
	l.Internal("Public download handler configured at /_piko/storage/public")

	if len(presignConfig.Secret) == 0 {
		l.Warn("Presign secret not available, presigned uploads/downloads disabled")
		return nil, nil, publicDownloadHandler
	}

	uploadHandler = presign_http.NewHandler(storageService, presignConfig)
	downloadHandler = presign_http.NewDownloadHandler(storageService, presignConfig)
	l.Internal("Presigned handlers configured at /_piko/storage/upload and /_piko/storage/download")
	return uploadHandler, downloadHandler, publicDownloadHandler
}

// getPresignConfig retrieves the presign configuration from the storage
// service. If the storage service provides a GetConfig method, use it;
// otherwise return defaults.
//
// Takes storageService (storage_domain.Service) which provides the storage
// configuration.
//
// Returns storage_domain.PresignConfig which contains the presign settings.
func (*routerOperation) getPresignConfig(storageService storage_domain.Service) storage_domain.PresignConfig {
	if configProvider, ok := storageService.(interface {
		GetPresignConfig() storage_domain.PresignConfig
	}); ok {
		return configProvider.GetPresignConfig()
	}
	return storage_domain.DefaultPresignConfig()
}

// manifestProviderBuilder defines a function signature for creating a manifest
// provider.
type manifestProviderBuilder func(path string, sandbox safedisk.Sandbox) generator_domain.ManifestProviderPort

// devRouterHandlers holds optional dev-mode handlers passed to buildRouter.
type devRouterHandlers struct {
	// eventsBroadcaster serves the /_piko/dev/events SSE endpoint.
	eventsBroadcaster http.Handler

	// apiHandler serves the /_piko/dev/api/* REST endpoints.
	apiHandler daemon_domain.DevAPIHandlerPort

	// previewHandler serves the /_piko/dev/preview/* endpoints.
	previewHandler daemon_domain.DevPreviewHandlerPort
}

// buildRouter creates the HTTP router for the application.
//
// This is the main entry point for building the router. It creates a
// routerOperation with the given settings and runs it to produce the final
// handler.
//
// Takes deps (*Dependencies) which provides the application dependencies.
// Takes c (*Container) which holds the dependency injection container.
// Takes store (ManifestStoreView) which provides access to manifest data.
// Takes templaterService (TemplaterService) which handles template rendering.
// Takes devHandlers (*devRouterHandlers) which provides optional dev-mode
// handlers; nil in production mode.
//
// Returns http.Handler which is the configured router ready to serve requests.
// Returns error when the router operation fails to run.
func buildRouter(
	ctx context.Context,
	deps *Dependencies,
	c *Container,
	store templater_domain.ManifestStoreView,
	templaterService templater_domain.TemplaterService,
	disableHTTPCache bool,
	devHandlers *devRouterHandlers,
) (http.Handler, error) {
	cspConfig := buildCSPRuntimeConfig(c, deps)

	operation := &routerOperation{
		deps:             deps,
		container:        c,
		store:            store,
		templaterService: templaterService,
		cspConfig:        cspConfig,
		disableHTTPCache: disableHTTPCache,
	}

	if devHandlers != nil {
		operation.devEventsBroadcaster = devHandlers.eventsBroadcaster
		operation.devAPIHandler = devHandlers.apiHandler
		operation.devPreviewHandler = devHandlers.previewHandler
	}

	return operation.execute(ctx)
}

// buildCSPRuntimeConfig builds the Content Security Policy runtime settings
// from the container's CSP builder, falling back to Piko defaults when needed.
// Called once at startup and the result is passed through the router chain, so
// the server config stays unchanged.
//
// Takes c (*Container) which holds the CSP builder settings.
// Takes deps (*Dependencies) which provides access to server config for
// fallback.
//
// Returns security_dto.CSPRuntimeConfig which contains the computed CSP
// settings.
func buildCSPRuntimeConfig(c *Container, deps *Dependencies) security_dto.CSPRuntimeConfig {
	ctx := c.GetAppContext()
	_, l := logger_domain.From(ctx, log)

	if builder := c.GetCSPConfig(); builder != nil {
		mergeCaptchaCSPDomains(ctx, c, builder)
		l.Internal("Building CSP runtime config from builder",
			logger_domain.Bool("report_only", builder.IsReportOnly()),
			logger_domain.Bool("uses_request_tokens", builder.UsesRequestTokens()))
		return builder.RuntimeConfig()
	}

	if policy, wasSet := c.GetCSPPolicyString(); wasSet {
		l.Internal("Using raw CSP policy string")
		return security_dto.CSPRuntimeConfig{
			Policy:            policy,
			ReportOnly:        false,
			UsesRequestTokens: false,
		}
	}

	if policy := deref(deps.ConfigProvider.ServerConfig.Security.Headers.ContentSecurityPolicy, ""); policy != "" {
		l.Internal("Using CSP policy from server config")
		return security_dto.CSPRuntimeConfig{
			Policy:            policy,
			ReportOnly:        false,
			UsesRequestTokens: false,
		}
	}

	builder := security_domain.NewCSPBuilder().WithPikoDefaults()
	mergeCaptchaCSPDomains(ctx, c, builder)
	l.Internal("Using Piko default CSP policy")
	return builder.RuntimeConfig()
}

// mergeCaptchaCSPDomains adds CSP domains from all registered captcha
// providers to the CSP builder. This allows captcha provider SDKs to load
// without manual CSP configuration.
//
// Takes c (*Container) which provides access to the captcha service and
// application context.
// Takes builder (*security_domain.CSPBuilder) which is the CSP builder to
// merge provider domains into.
func mergeCaptchaCSPDomains(ctx context.Context, c *Container, builder *security_domain.CSPBuilder) {
	ctx, l := logger_domain.From(ctx, log)

	captchaService, err := c.GetCaptchaService()
	if err != nil || !captchaService.IsEnabled() {
		return
	}

	for _, providerInfo := range captchaService.ListProviders(ctx) {
		provider, providerErr := captchaService.GetProviderByName(ctx, providerInfo.Name)
		if providerErr != nil {
			continue
		}

		requirements := provider.RenderRequirements()
		if requirements.ServerSideToken {
			continue
		}

		mergeProviderCSPDomains(builder, requirements)

		l.Internal("Merged captcha CSP domains",
			logger_domain.String("provider", providerInfo.Name))
	}
}

// mergeProviderCSPDomains adds the CSP domains from a single captcha provider's
// render requirements to the CSP builder.
//
// Takes builder (*security_domain.CSPBuilder) which is the CSP builder to add
// directive sources to.
// Takes requirements (*captcha_dto.RenderRequirements) which contains the
// script, frame, and connect domains the provider needs.
func mergeProviderCSPDomains(builder *security_domain.CSPBuilder, requirements *captcha_dto.RenderRequirements) {
	if len(requirements.CSPScriptDomains) > 0 {
		scriptSources := make([]security_domain.Source, 0, 1+len(requirements.CSPScriptDomains))
		scriptSources = append(scriptSources, security_domain.Self)
		for _, domain := range requirements.CSPScriptDomains {
			scriptSources = append(scriptSources, security_domain.Source(domain))
		}
		builder.ScriptSrc(scriptSources...)
	}
	if len(requirements.CSPFrameDomains) > 0 {
		frameSources := make([]security_domain.Source, 0, 1+len(requirements.CSPFrameDomains))
		frameSources = append(frameSources, security_domain.Self)
		for _, domain := range requirements.CSPFrameDomains {
			frameSources = append(frameSources, security_domain.Source(domain))
		}
		builder.FrameSrc(frameSources...)
	}
	for _, domain := range requirements.CSPConnectDomains {
		builder.ConnectSrc(security_domain.Source(domain))
	}
}

// createManifestProvider selects the correct manifest loading adapter based
// on configuration.
//
// Takes ctx (context.Context) which carries the application context for logging.
// Takes c (*Container) which provides configuration and sandbox creation.
//
// Returns generator_domain.ManifestProviderPort which is the selected manifest
// loader.
// Returns error when the manifest format is not recognised.
func createManifestProvider(ctx context.Context, c *Container) (generator_domain.ManifestProviderPort, error) {
	_, l := logger_domain.From(ctx, log)

	serverConfig := c.config.ServerConfig

	distDir := filepath.Join(deref(serverConfig.Paths.BaseDir, "."), distDirName)
	manifestPath := filepath.Join(distDir, manifestFilenameBinary)
	l.Internal("Loading component manifest",
		logger_domain.String("format", config.ManifestFormat),
		logger_domain.String("path", manifestPath),
	)

	builder, ok := manifestProviderBuilders[config.ManifestFormat]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errUnknownManifestFormat, config.ManifestFormat)
	}

	sandbox, sandboxErr := c.createSandbox("manifest-read", distDir, safedisk.ModeReadOnly)
	if sandboxErr != nil {
		l.Warn("Failed to create manifest read sandbox, provider will use fallback",
			logger_domain.Error(sandboxErr))
	}

	return builder(manifestPath, sandbox), nil
}

// buildRouteSettings constructs a RouteSettings from the server configuration,
// dereferencing pointer fields with sensible defaults that match the config
// struct tags.
//
// Takes serverConfig (config.ServerConfig) which is the loaded server
// configuration to extract route settings from.
//
// Returns daemon_adapters.RouteSettings which holds the resolved route
// configuration values.
func buildRouteSettings(serverConfig *config.ServerConfig) daemon_adapters.RouteSettings {
	return daemon_adapters.RouteSettings{
		E2EMode:                      deref(serverConfig.Build.E2EMode, false),
		ActionMaxBodyBytes:           deref(serverConfig.Network.ActionMaxBodyBytes, defaultActionMaxBodyBytes),
		DefaultMaxSSEDurationSeconds: deref(serverConfig.Network.DefaultMaxSSEDurationSeconds, defaultMaxSSEDurationSeconds),
		MaxMultipartFormBytes:        deref(serverConfig.Network.MaxMultipartFormBytes, defaultMaxMultipartFormBytes),
		ActionServePath:              deref(serverConfig.Paths.ActionServePath, "/_piko/actions"),
		CSRFSecFetchSiteEnforcement:  deref(serverConfig.Security.CSRF.SecFetchSiteEnforcement, true),
	}
}
