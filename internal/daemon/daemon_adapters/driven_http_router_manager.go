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

package daemon_adapters

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/spamdetect/spamdetect_domain"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

// RouterManagerConfig bundles the configuration parameters for creating a
// RouterManager.
type RouterManagerConfig struct {
	// CSRFService creates and validates CSRF tokens.
	CSRFService security_domain.CSRFTokenService

	// CaptchaService verifies captcha tokens. Nil when captcha is disabled.
	CaptchaService captcha_domain.CaptchaServicePort

	// SpamDetectService analyses form content for spam. Nil when disabled.
	SpamDetectService spamdetect_domain.SpamDetectServicePort

	// RegistryService provides access to the service registry.
	RegistryService registry_domain.RegistryService

	// VariantGenerator creates resource variants when they are requested.
	VariantGenerator daemon_domain.OnDemandVariantGenerator

	// PresignDownloadHandler handles presigned URL downloads; may be nil.
	PresignDownloadHandler http.Handler

	// PresignUploadHandler handles presigned URL uploads; nil disables this
	// feature.
	PresignUploadHandler http.Handler

	// RateLimitService provides rate limiting for the rate limiting middleware.
	RateLimitService security_domain.RateLimitService

	// SiteSettings holds the website settings used by the router.
	SiteSettings *config.WebsiteConfig

	// Deps holds shared dependencies used by HTTP handlers.
	Deps *daemon_domain.HTTPHandlerDependencies

	// CacheMiddleware wraps handlers with HTTP caching behaviour.
	CacheMiddleware func(next http.Handler) http.Handler

	// AppRouter is the main HTTP router for the application.
	AppRouter *chi.Mux

	// RouterConfig provides the router-specific settings for BuildRouter.
	RouterConfig *daemon_domain.RouterConfig

	// AuthGuardConfig holds auth guard settings for page-level AuthPolicy
	// enforcement. Nil when no auth guard is configured.
	AuthGuardConfig *daemon_dto.AuthGuardConfig

	// ArtefactCache provides the cache hexagon instance for artefact metadata.
	// May be nil to disable metadata caching.
	ArtefactCache cache_domain.Cache[string, *registry_dto.ArtefactMeta]

	// Actions maps action names to their handler entries.
	Actions map[string]ActionHandlerEntry

	// RouteProviders is the list of route providers that supply HTTP routes.
	RouteProviders []daemon_domain.RouteProvider

	// CSPConfig holds the Content Security Policy settings for security headers.
	CSPConfig security_dto.CSPRuntimeConfig

	// RouteSettings provides typed route settings for route mounting.
	RouteSettings RouteSettings
}

// RouterManager is an http.Handler that reloads its routes while running.
// It implements daemon_domain.RouterManager and
// lifecycle_domain.RouterReloadNotifier to serve requests using the latest
// router settings without dropping any requests that are still in progress.
type RouterManager struct {
	// currentRouter is the active router that handles requests; it is swapped
	// during route reloads.
	currentRouter http.Handler

	// csrfService creates and checks CSRF tokens.
	csrfService security_domain.CSRFTokenService

	// registryService provides access to the artefact registry.
	registryService registry_domain.RegistryService

	// variantGenerator builds image variants when routes are set up.
	variantGenerator daemon_domain.OnDemandVariantGenerator

	// presignDownloadHandler handles presigned URL downloads; may be nil.
	presignDownloadHandler http.Handler

	// presignUploadHandler provides presigned URL uploads; nil disables this
	// feature.
	presignUploadHandler http.Handler

	// rateLimitService controls request rate limits for the rate limiting
	// middleware.
	rateLimitService security_domain.RateLimitService

	// artefactCache provides the cache hexagon instance for artefact metadata.
	artefactCache cache_domain.Cache[string, *registry_dto.ArtefactMeta]

	// currentBuilder holds the builder that created the current router, so its
	// resources (e.g. metadata cache) can be closed on the next reload.
	currentBuilder daemon_domain.RouterBuilder

	// captchaService verifies captcha tokens. Nil when captcha is disabled.
	captchaService captcha_domain.CaptchaServicePort

	// spamdetectService analyses form content for spam. Nil when disabled.
	spamdetectService spamdetect_domain.SpamDetectServicePort

	// siteSettings holds the website settings that route handlers use.
	siteSettings *config.WebsiteConfig

	// appRouter is the main HTTP router for application routes.
	appRouter *chi.Mux

	// routerConfig stores router-specific settings used when building routers.
	routerConfig *daemon_domain.RouterConfig

	// authGuardConfig holds auth guard settings for page-level AuthPolicy
	// enforcement. Nil when no auth guard is configured.
	authGuardConfig *daemon_dto.AuthGuardConfig

	// cacheMiddleware wraps handlers to add response caching.
	cacheMiddleware func(next http.Handler) http.Handler

	// actions maps action names to handlers for route processing.
	actions map[string]ActionHandlerEntry

	// deps holds shared dependencies used when setting up HTTP routes.
	deps *daemon_domain.HTTPHandlerDependencies

	// routeProviders holds the providers that add routes when the router reloads.
	routeProviders []daemon_domain.RouteProvider

	// cspConfig holds the CSP settings from startup; read-only after start.
	cspConfig security_dto.CSPRuntimeConfig

	// routeSettings stores typed route settings used when mounting routes.
	routeSettings RouteSettings

	// mu guards access to currentRouter during hot-reloads.
	mu sync.RWMutex
}

// NewRouterManager creates a new RouterManager that is not yet set up.
//
// Takes routerManagerConfig (*RouterManagerConfig) which
// provides all dependencies and settings for the router
// manager.
//
// Returns *RouterManager which is ready for use after setup.
func NewRouterManager(routerManagerConfig *RouterManagerConfig) *RouterManager {
	return &RouterManager{
		currentRouter:          nil,
		csrfService:            routerManagerConfig.CSRFService,
		registryService:        routerManagerConfig.RegistryService,
		variantGenerator:       routerManagerConfig.VariantGenerator,
		cspConfig:              routerManagerConfig.CSPConfig,
		deps:                   routerManagerConfig.Deps,
		siteSettings:           routerManagerConfig.SiteSettings,
		actions:                routerManagerConfig.Actions,
		cacheMiddleware:        routerManagerConfig.CacheMiddleware,
		appRouter:              routerManagerConfig.AppRouter,
		routerConfig:           routerManagerConfig.RouterConfig,
		routeSettings:          routerManagerConfig.RouteSettings,
		routeProviders:         routerManagerConfig.RouteProviders,
		presignUploadHandler:   routerManagerConfig.PresignUploadHandler,
		presignDownloadHandler: routerManagerConfig.PresignDownloadHandler,
		rateLimitService:       routerManagerConfig.RateLimitService,
		authGuardConfig:        routerManagerConfig.AuthGuardConfig,
		artefactCache:          routerManagerConfig.ArtefactCache,
		captchaService:         routerManagerConfig.CaptchaService,
		spamdetectService:      routerManagerConfig.SpamDetectService,
		mu:                     sync.RWMutex{},
	}
}

// ServeHTTP handles an HTTP request by passing it to the current router.
// This method satisfies the http.Handler interface.
//
// Takes w (http.ResponseWriter) which receives the HTTP response.
// Takes r (*http.Request) which contains the incoming HTTP request.
//
// Safe for concurrent use. Acquires a read lock to access the current router.
func (rm *RouterManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rm.mu.RLock()
	router := rm.currentRouter
	rm.mu.RUnlock()

	if router != nil {
		router.ServeHTTP(w, r)
	} else {
		http.Error(w, "Service is initialising, please try again shortly.", http.StatusServiceUnavailable)
	}
}

// ReloadRoutes performs hot-reloading by building a new router in the
// background and atomically swapping it into place.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes store (ManifestStoreView) which provides the new manifest data for
// route configuration.
//
// Returns error when the router build fails.
//
// Safe for concurrent use. Uses a write lock during the atomic swap of the
// current router.
func (rm *RouterManager) ReloadRoutes(ctx context.Context, store templater_domain.ManifestStoreView) error {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Hot-reloading Chi router with new manifest...")

	newAppRouter := chi.NewRouter()
	newAppRouter.Use(rm.appRouter.Middlewares()...)

	MountRoutesFromManifest(ctx, &MountRoutesConfig{
		Router:            newAppRouter,
		Deps:              rm.deps,
		Store:             store,
		CSRFService:       rm.csrfService,
		RouteSettings:     rm.routeSettings,
		SiteSettings:      rm.siteSettings,
		Actions:           rm.actions,
		CacheMiddleware:   rm.cacheMiddleware,
		AuthGuardConfig:   rm.authGuardConfig,
		CaptchaService:    rm.captchaService,
		SpamDetectService: rm.spamdetectService,
	})

	for _, provider := range rm.routeProviders {
		provider.MountRoutes(newAppRouter)
	}

	builder := NewHTTPRouterBuilder(rm.artefactCache)
	finalRouter, err := builder.BuildRouter(
		rm.routerConfig,
		daemon_domain.RouterDependencies{
			RegistryService:        rm.registryService,
			UserRouter:             newAppRouter,
			VariantGenerator:       rm.variantGenerator,
			CSPConfig:              rm.cspConfig,
			PresignUploadHandler:   rm.presignUploadHandler,
			PresignDownloadHandler: rm.presignDownloadHandler,
			RateLimitService:       rm.rateLimitService,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build new router during reload: %w", err)
	}

	rm.mu.Lock()
	oldBuilder := rm.currentBuilder
	rm.currentRouter = finalRouter
	rm.currentBuilder = builder
	rm.mu.Unlock()

	if oldBuilder != nil {
		oldBuilder.Close()
	}

	l.Internal("Chi router hot-reload complete.")
	return nil
}

// Close stops background goroutines owned by the current builder.
//
// This includes resources such as the artefact metadata cache. Call during
// application shutdown.
//
// Safe for concurrent use. Acquires a read lock to access the current
// builder.
func (rm *RouterManager) Close() {
	rm.mu.RLock()
	b := rm.currentBuilder
	rm.mu.RUnlock()

	if b != nil {
		b.Close()
	}
}
