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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/analytics/analytics_adapters"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/security/security_adapters"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

// HTTPRouterBuilder builds the main HTTP router with all system middleware
// and route handlers. It implements the RouterBuilder interface.
type HTTPRouterBuilder struct {
	// artefactCache is the cache hexagon instance for artefact metadata. Set
	// before BuildRouter is called; used by serveArtefact to create the
	// metadataCache wrapper.
	artefactCache cache_domain.Cache[string, *registry_dto.ArtefactMeta]

	// metadataCache holds the artefact metadata cache created during route setup.
	// Stored here so Close() can release its resources.
	metadataCache *artefactMetadataCache

	// variantGenerationGroup deduplicates concurrent background variant
	// generation for the same artefact. Without this, N concurrent requests
	// for a pending artefact would each spawn a goroutine that loops over
	// missing profiles - only the first does real work while the rest
	// short-circuit, but the goroutine burst is unnecessary.
	variantGenerationGroup singleflight.Group
}

// BuildRouter constructs the final HTTP handler with all middleware and routes
// configured. It sets up CORS, logging, recovery, timeouts, and mounts asset
// and page handlers.
//
// Takes routerConfig (*daemon_domain.RouterConfig) which provides the
// router-specific settings.
// Takes deps (daemon_domain.RouterDependencies) which groups all router
// dependencies.
//
// Returns http.Handler which is the fully configured router ready to serve.
// Returns error when router construction fails.
func (builder *HTTPRouterBuilder) BuildRouter(
	routerConfig *daemon_domain.RouterConfig,
	deps daemon_domain.RouterDependencies,
) (http.Handler, error) {
	_, span, l := log.Span(context.Background(), "BuildRouter",
		logger_domain.String("port", routerConfig.Port),
	)
	defer span.End()

	l.Internal("Building router")
	mainRouter := chi.NewRouter()

	builder.setupBaseMiddleware(mainRouter, routerConfig, deps.CSPConfig)
	builder.setupStaticRoutes(mainRouter, routerConfig, deps.RegistryService, deps.VariantGenerator, deps.PresignUploadHandler, deps.PresignDownloadHandler, deps.PublicDownloadHandler)
	if err := builder.setupDynamicRoutes(mainRouter, routerConfig, deps); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("setting up dynamic routes: %w", err)
	}

	mainRouter.NotFound(http.NotFound)

	span.SetStatus(codes.Ok, "Router built successfully")
	return mainRouter, nil
}

// Close releases resources held by the builder, such as the artefact metadata
// cache. Call when the router is no longer needed.
func (builder *HTTPRouterBuilder) Close() {
	if builder.metadataCache != nil {
		builder.metadataCache.Close(context.Background())
	}
}

// setupBaseMiddleware adds middleware that applies to all routes.
//
// Takes router (*chi.Mux) which is the router to add middleware to.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides security and
// network settings.
// Takes cspConfig (security_dto.CSPRuntimeConfig) which provides the
// computed CSP settings.
func (*HTTPRouterBuilder) setupBaseMiddleware(
	router *chi.Mux,
	routerConfig *daemon_domain.RouterConfig,
	cspConfig security_dto.CSPRuntimeConfig,
) {
	if routerConfig.SecurityHeaders.Enabled {
		securityMiddleware := security_adapters.NewSecurityHeadersMiddleware(
			routerConfig.SecurityHeaders,
			routerConfig.ForceHTTPS,
			cspConfig,
			routerConfig.Reporting,
		)
		router.Use(securityMiddleware.Handler)
	}

	router.Use(middleware.Recoverer)

	router.Use(middleware.Heartbeat("/ping"))
}

// setupStaticRoutes adds routes for static files without CORS, logging, or
// timeouts.
//
// Takes router (*chi.Mux) which is the router to add routes to.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides path settings.
// Takes registryService (registry_domain.RegistryService) which provides access to
// registry data.
// Takes variantGenerator (daemon_domain.OnDemandVariantGenerator) which creates
// image variants on demand.
// Takes presignUploadHandler (http.Handler) which handles presigned URL
// uploads;
// may be nil if presigned uploads are not enabled.
// Takes presignDownloadHandler (http.Handler) which handles presigned URL
// downloads; may be nil if presigned downloads are not enabled.
// Takes publicDownloadHandler (http.Handler) which handles public file
// downloads
// without authentication; may be nil if not enabled.
func (builder *HTTPRouterBuilder) setupStaticRoutes(
	router *chi.Mux,
	routerConfig *daemon_domain.RouterConfig,
	registryService registry_domain.RegistryService,
	variantGenerator daemon_domain.OnDemandVariantGenerator,
	presignUploadHandler http.Handler,
	presignDownloadHandler http.Handler,
	publicDownloadHandler http.Handler,
) {
	noCache := routerConfig.DisableHTTPCache
	router.Get(routerConfig.DistServePath+"/*", serveEmbeddedFrontend(routerConfig.WatchMode, noCache))
	router.Get("/theme.css", builder.serveTheme(registryService, noCache))
	router.Get("/sitemap.xml", builder.serveSitemapArtefact(registryService, "sitemap.xml", noCache))
	router.Get("/sitemap-{number}.xml", builder.serveSitemapChunk(registryService, noCache))
	router.Get("/robots.txt", builder.serveRobotsTxt(registryService, noCache))
	router.Get(fmt.Sprintf("%s/*", routerConfig.ArtefactServePath), builder.serveArtefact(registryService, variantGenerator, noCache))

	router.Get("/_piko/video/{artefactID}/master.m3u8", builder.serveVideoMasterPlaylist(registryService, noCache))
	router.Get("/_piko/video/{artefactID}/{quality}/manifest.m3u8", builder.serveVideoVariantPlaylist(registryService, noCache))
	router.Get("/_piko/video/{artefactID}/{quality}/{chunk}", builder.serveVideoChunk(registryService))

	if presignUploadHandler != nil {
		router.Put("/_piko/storage/upload", presignUploadHandler.ServeHTTP)
		router.Post("/_piko/storage/upload", presignUploadHandler.ServeHTTP)
	}

	if presignDownloadHandler != nil {
		router.Get("/_piko/storage/download", presignDownloadHandler.ServeHTTP)
		router.Head("/_piko/storage/download", presignDownloadHandler.ServeHTTP)
	}

	if publicDownloadHandler != nil {
		router.Get("/_piko/storage/public/*", publicDownloadHandler.ServeHTTP)
		router.Head("/_piko/storage/public/*", publicDownloadHandler.ServeHTTP)
	}

	if routerConfig.DevEventsBroadcaster != nil {
		router.Get("/_piko/dev/events", routerConfig.DevEventsBroadcaster.ServeHTTP)
	}

	if routerConfig.DevAPIHandler != nil {
		routerConfig.DevAPIHandler.Mount(router)
	}

	if routerConfig.DevPreviewHandler != nil {
		routerConfig.DevPreviewHandler.Mount(router)
	}
}

// setupDynamicRoutes adds routes with the full middleware stack including CORS,
// logging, and timeouts.
//
// Takes router (*chi.Mux) which is the router to configure.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides timeout and
// CORS settings.
// Takes deps (daemon_domain.RouterDependencies) which groups all router
// dependencies including auth, rate limiting, and user routes.
//
// Returns error when the trusted proxy configuration is invalid.
func (builder *HTTPRouterBuilder) setupDynamicRoutes(
	router *chi.Mux,
	routerConfig *daemon_domain.RouterConfig,
	deps daemon_domain.RouterDependencies,
) error {
	var setupErr error
	router.Group(func(r chi.Router) {
		if err := builder.setupRealIP(r, routerConfig); err != nil {
			setupErr = err
			return
		}

		if deps.AuthProvider != nil {
			authMiddleware := security_adapters.NewAuthMiddleware(deps.AuthProvider, log)
			r.Use(authMiddleware.Handler)
		}

		if deps.AnalyticsService != nil {
			analyticsMw := analytics_adapters.NewAnalyticsMiddleware(deps.AnalyticsService)
			r.Use(analyticsMw.Handler)
		}

		builder.setupRateLimiting(r, routerConfig, deps.RateLimitService)
		builder.setupCORS(r, routerConfig)

		if routerConfig.MaxConcurrentRequests > 0 {
			r.Use(middleware.Throttle(routerConfig.MaxConcurrentRequests))
		}

		if routerConfig.RequestTimeoutSeconds > 0 {
			r.Use(middleware.Timeout(time.Duration(routerConfig.RequestTimeoutSeconds) * time.Second))
		}

		if deps.AuthGuardConfig != nil {
			authGuard := security_adapters.NewAuthGuardMiddleware(*deps.AuthGuardConfig)
			r.Use(authGuard.Handler)
		}

		if deps.UserRouter != nil {
			r.Handle("/*", deps.UserRouter)
		}
	})
	return setupErr
}

// setupRealIP adds the RealIP middleware that extracts the client IP and
// creates a request ID, storing both in the request context. This must run
// before rate limiting and other middleware that need the client IP or
// request ID.
//
// Takes r (chi.Router) which receives the RealIP middleware.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides trusted proxy
// settings.
//
// Returns error when the trusted proxy configuration is invalid.
func (*HTTPRouterBuilder) setupRealIP(r chi.Router, routerConfig *daemon_domain.RouterConfig) error {
	ipExtractor, err := security_adapters.NewTrustedProxyIPExtractor(
		routerConfig.RateLimit.TrustedProxies,
		routerConfig.RateLimit.CloudflareEnabled,
	)
	if err != nil {
		return fmt.Errorf("creating IP extractor: %w", err)
	}
	realIPMiddleware := security_adapters.NewRealIPMiddleware(ipExtractor)
	r.Use(realIPMiddleware.Handler)
	return nil
}

// setupRateLimiting adds rate limiting middleware to the router if enabled.
// The rate limiter reads the client IP from context, set by RealIP middleware.
//
// Takes r (chi.Router) which receives the rate limiting middleware.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides rate limit
// settings.
// Takes rateLimitService (security_domain.RateLimitService) which handles rate
// limit checks.
func (*HTTPRouterBuilder) setupRateLimiting(
	r chi.Router,
	routerConfig *daemon_domain.RouterConfig,
	rateLimitService security_domain.RateLimitService,
) {
	if !routerConfig.RateLimit.Enabled {
		return
	}
	rateLimitMiddleware := newRateLimitMiddleware(
		routerConfig.RateLimit,
		rateLimitService,
	)
	r.Use(rateLimitMiddleware.Handler)
}

// setupCORS sets up CORS middleware to handle cross-origin requests.
//
// Takes r (chi.Router) which receives the CORS middleware.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides the allowed
// origins.
func (*HTTPRouterBuilder) setupCORS(r chi.Router, routerConfig *daemon_domain.RouterConfig) {
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   buildAllowedOrigins(routerConfig),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Authorization", headerContentType, "X-CSRF-Token", "X-CSRF-Action-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
}

// staticArtefactConfig holds settings for serving static files.
type staticArtefactConfig struct {
	// artefactID is the unique identifier for the static artefact to serve.
	artefactID string

	// defaultMimeType is the MIME type to use when the variant does not specify
	// one.
	defaultMimeType string

	// cacheMaxAge is the Cache-Control header value for static responses.
	cacheMaxAge string

	// preferredType specifies which variant type to prefer, such as "minified" or
	// "source"; an empty string allows compressed variants with fallback to
	// source.
	preferredType string

	// useCompression enables the selection of compressed variants when set to
	// true.
	useCompression bool
}

// serveTheme returns a handler that serves the theme CSS file.
//
// Takes registryService (registry_domain.RegistryService) which provides access
// to static assets.
//
// Returns http.HandlerFunc which serves the minified, compressed theme CSS
// with a one-hour cache time.
func (*HTTPRouterBuilder) serveTheme(registryService registry_domain.RegistryService, disableHTTPCache bool) http.HandlerFunc {
	return serveStaticArtefactHandler(registryService, staticArtefactConfig{
		artefactID:      "theme.css",
		defaultMimeType: contentTypeCSS,
		cacheMaxAge:     cacheControlForMode(disableHTTPCache, cacheControlMutableAsset),
		preferredType:   "minified",
		useCompression:  true,
	}, "serveTheme")
}

// serveSitemapArtefact returns a handler that serves a sitemap file by its ID
// (e.g., "sitemap.xml", "sitemap-1.xml").
//
// Takes registryService (registry_domain.RegistryService) which provides access to
// stored files.
// Takes artefactID (string) which is the name of the sitemap file to serve.
//
// Returns http.HandlerFunc which serves the sitemap with XML content type and
// a one-hour cache duration.
func (*HTTPRouterBuilder) serveSitemapArtefact(registryService registry_domain.RegistryService, artefactID string, disableHTTPCache bool) http.HandlerFunc {
	return serveStaticArtefactHandler(registryService, staticArtefactConfig{
		artefactID:      artefactID,
		defaultMimeType: contentTypeXML,
		cacheMaxAge:     cacheControlForMode(disableHTTPCache, cacheControlMutableAsset),
		preferredType:   variantSource,
		useCompression:  true,
	}, "serveSitemapArtefact")
}

// serveSitemapChunk serves numbered sitemap chunk files such as
// /sitemap-1.xml or /sitemap-2.xml.
//
// Takes registryService (registry_domain.RegistryService) which provides access
// to stored sitemap files.
//
// Returns http.HandlerFunc which handles requests for sitemap chunks.
func (builder *HTTPRouterBuilder) serveSitemapChunk(registryService registry_domain.RegistryService, disableHTTPCache bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		number := chi.URLParam(r, "number")
		artefactID := fmt.Sprintf("sitemap-%s.xml", number)
		builder.serveSitemapArtefact(registryService, artefactID, disableHTTPCache)(w, r)
	}
}

// serveRobotsTxt returns a handler that serves the robots.txt file.
//
// Takes registryService (RegistryService) which provides access to static files.
//
// Returns http.HandlerFunc which serves the robots.txt file with a one-hour
// cache.
func (*HTTPRouterBuilder) serveRobotsTxt(registryService registry_domain.RegistryService, disableHTTPCache bool) http.HandlerFunc {
	return serveStaticArtefactHandler(registryService, staticArtefactConfig{
		artefactID:      "robots.txt",
		defaultMimeType: "text/plain; charset=utf-8",
		cacheMaxAge:     cacheControlForMode(disableHTTPCache, cacheControlMutableAsset),
		preferredType:   variantSource,
		useCompression:  false,
	}, "serveRobotsTxt")
}

// artefactLookupResult holds the result of finding an artefact by its key.
type artefactLookupResult struct {
	// err holds any error that occurred during artefact lookup.
	err error

	// artefact holds the metadata for the resolved artefact.
	artefact *registry_dto.ArtefactMeta

	// httpStatus is the HTTP status code to return; 0 means success.
	httpStatus int

	// foundByStorageKey indicates whether the artefact was found using its storage
	// key rather than its main identifier.
	foundByStorageKey bool
}

// variantResolutionContext bundles the dependencies and state needed for
// variant resolution.
type variantResolutionContext struct {
	// span is the trace span for reporting errors and status changes.
	span trace.Span

	// w is the HTTP response writer for sending responses to the client.
	w http.ResponseWriter

	// r is the HTTP request used to check client content encoding preferences.
	r *http.Request

	// registryService provides access to artefact and variant data.
	registryService registry_domain.RegistryService

	// variantGenerator creates image variants when they are needed.
	variantGenerator daemon_domain.OnDemandVariantGenerator

	// cacheControl is the Cache-Control header value for the response.
	cacheControl string
}

// serveArtefact returns an HTTP handler that serves artefact content.
//
// Takes registryService (registry_domain.RegistryService) which provides access
// to artefact metadata and storage.
// Takes variantGenerator (daemon_domain.OnDemandVariantGenerator) which
// creates image variants when needed.
//
// Returns http.HandlerFunc which handles artefact requests with optional
// variant selection.
func (builder *HTTPRouterBuilder) serveArtefact(
	registryService registry_domain.RegistryService,
	variantGenerator daemon_domain.OnDemandVariantGenerator,
	disableHTTPCache bool,
) http.HandlerFunc {
	artefactCacheControl := cacheControlForMode(disableHTTPCache, cacheControlLongLived)
	builder.metadataCache = newArtefactMetadataCache(builder.artefactCache, registryService)
	metadataCache := builder.metadataCache

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		artefactID := chi.URLParam(r, "*")
		variantParam := r.URL.Query().Get("v")

		ctx, span, l := log.Span(ctx, "serveArtefact",
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.String("variantParam", variantParam),
			logger_domain.String(logFieldMethod, r.Method),
			logger_domain.String(logFieldPath, r.URL.Path),
		)
		defer span.End()

		if artefactID == "" {
			l.ReportError(span, errors.New("empty artefact ID"), "Empty artefact ID")
			http.NotFound(w, r)
			return
		}

		lookup := lookupArtefact(ctx, span, metadataCache, registryService, artefactID)
		if lookup.httpStatus != 0 {
			if lookup.httpStatus == http.StatusNotFound {
				http.NotFound(w, r)
			} else {
				http.Error(w, errMessageInternalServer, lookup.httpStatus)
			}
			return
		}

		vrc := variantResolutionContext{
			span:             span,
			w:                w,
			r:                r,
			registryService:  registryService,
			variantGenerator: variantGenerator,
			cacheControl:     artefactCacheControl,
		}

		variant := builder.resolveVariant(ctx, vrc, lookup, artefactID, variantParam)
		if variant == nil {
			return
		}

		serveVariantResponse(ctx, vrc, lookup.artefact, variant)
	}
}

// resolveVariant finds which variant to serve based on the artefact state
// and request settings.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes vrc (variantResolutionContext) which provides the request context.
// Takes lookup (artefactLookupResult) which holds the artefact lookup data.
// Takes artefactID (string) which is the artefact or storage key.
// Takes variantParam (string) which is the requested variant name.
//
// Returns *registry_dto.Variant which is the matched variant, or nil if an
// error occurred.
func (builder *HTTPRouterBuilder) resolveVariant(
	ctx context.Context,
	vrc variantResolutionContext,
	lookup artefactLookupResult,
	artefactID, variantParam string,
) *registry_dto.Variant {
	_, l := logger_domain.From(ctx, log)

	artefact := lookup.artefact

	if artefact.Status == registry_dto.VariantStatusPending && !lookup.foundByStorageKey {
		return builder.resolveVariantForPendingArtefact(ctx, vrc, artefact, variantParam)
	}

	if lookup.foundByStorageKey {
		variant := findVariantByStorageKey(artefact.ActualVariants, artefactID)
		if variant == nil {
			err := fmt.Errorf("variant with storage key %q not found for artefact %s", artefactID, artefact.ID)
			l.ReportError(vrc.span, err, "Consistency error: Artefact found by storage key but variant missing")
			http.Error(vrc.w, errMessageInternalServer, http.StatusInternalServerError)
			return nil
		}
		return variant
	}

	return builder.resolveVariantByArtefactID(ctx, vrc, artefact, variantParam)
}

// resolveVariantForPendingArtefact creates a variant for an artefact that is
// still pending.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes vrc (variantResolutionContext) which provides the request context and
// dependencies.
// Takes artefact (*registry_dto.ArtefactMeta) which specifies the pending
// artefact to create a variant for.
// Takes variantParam (string) which specifies the requested variant, or empty
// for the default source variant.
//
// Returns *registry_dto.Variant which is the created variant, or nil if
// creation failed.
func (builder *HTTPRouterBuilder) resolveVariantForPendingArtefact(
	ctx context.Context,
	vrc variantResolutionContext,
	artefact *registry_dto.ArtefactMeta,
	variantParam string,
) *registry_dto.Variant {
	ctx, l := logger_domain.From(ctx, log)

	lazyArtefactServeCount.Add(ctx, 1)

	l.Trace("Artefact in PENDING state, generating first variant on-demand",
		logger_domain.String(logFieldArtefactID, artefact.ID))

	baseVariantID := variantSource
	if variantParam != "" {
		baseVariantID = variantParam
	}

	startTime := time.Now()
	variant := builder.tryGenerateVariantOnDemand(ctx, vrc.variantGenerator, artefact, baseVariantID)
	lazyVariantGenerationDuration.Record(ctx, float64(time.Since(startTime).Milliseconds()))

	if variant == nil {
		err := fmt.Errorf("failed to generate variant %q for pending artefact %s", baseVariantID, artefact.ID)
		l.ReportError(vrc.span, err, "Lazy variant generation failed")
		http.Error(vrc.w, "Service Unavailable", http.StatusServiceUnavailable)
		return nil
	}

	if remainingCount := len(artefact.DesiredProfiles) - 1; remainingCount > 0 {
		backgroundVariantQueueCount.Add(ctx, int64(remainingCount))
	}
	builder.queueRemainingVariants(ctx, vrc.registryService, vrc.variantGenerator, artefact, baseVariantID)

	return variant
}

// resolveVariantByArtefactID finds a variant using the artefact ID and an
// optional variant parameter.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes vrc (variantResolutionContext) which provides the resolution context
// including the HTTP writer and span.
// Takes artefact (*registry_dto.ArtefactMeta) which contains the artefact
// metadata with the available variants.
// Takes variantParam (string) which specifies the requested variant ID, or an
// empty string to use the default source variant.
//
// Returns *registry_dto.Variant which is the found variant, or nil if not
// found and the source fallback also fails.
func (builder *HTTPRouterBuilder) resolveVariantByArtefactID(
	ctx context.Context,
	vrc variantResolutionContext,
	artefact *registry_dto.ArtefactMeta,
	variantParam string,
) *registry_dto.Variant {
	ctx, l := logger_domain.From(ctx, log)

	baseVariantID := variantSource
	if variantParam != "" {
		baseVariantID = variantParam
	}

	variant := findVariantByID(artefact.ActualVariants, baseVariantID)

	if variant == nil && variantParam != "" {
		variant = builder.tryGenerateVariantOnDemand(ctx, vrc.variantGenerator, artefact, variantParam)
	}

	if variant == nil {
		if variantParam != "" {
			l.Trace("Variant not found or generation failed, falling back to source",
				logger_domain.String("requestedVariant", variantParam))
		}
		variant = findVariantByID(artefact.ActualVariants, variantSource)
		if variant == nil {
			l.Internal("Variant not found",
				logger_domain.String(logFieldVariantID, baseVariantID),
				logger_domain.String(logFieldArtefactID, artefact.ID))
			vrc.span.SetStatus(codes.Error, "Variant not found")
			http.NotFound(vrc.w, nil)
			return nil
		}
	}

	return variant
}

// tryGenerateVariantOnDemand tries to create an image variant on demand.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes generator (daemon_domain.OnDemandVariantGenerator) which creates the
// variant.
// Takes artefact (*registry_dto.ArtefactMeta) which identifies the source
// image.
// Takes profileName (string) which specifies which variant profile to create.
//
// Returns *registry_dto.Variant which is the created variant if successful,
// or nil if the generator is not available, the profile name is not valid, or
// creation fails.
func (*HTTPRouterBuilder) tryGenerateVariantOnDemand(
	ctx context.Context,
	generator daemon_domain.OnDemandVariantGenerator,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
) *registry_dto.Variant {
	ctx, l := logger_domain.From(ctx, log)

	if generator == nil {
		l.Trace("On-demand variant generator not available, skipping generation")
		return nil
	}

	if generator.ParseProfileName(profileName) == nil {
		l.Trace("Profile name is not a valid on-demand generation profile",
			logger_domain.String("profileName", profileName))
		return nil
	}

	l.Trace("Attempting on-demand variant generation",
		logger_domain.String(logFieldArtefactID, artefact.ID),
		logger_domain.String("profileName", profileName),
	)

	variant, err := generator.GenerateVariant(ctx, artefact, profileName)
	if err != nil {
		l.Warn("On-demand variant generation failed",
			logger_domain.String(logFieldArtefactID, artefact.ID),
			logger_domain.String("profileName", profileName),
			logger_domain.String(logFieldError, err.Error()),
		)
		return nil
	}

	l.Trace("Successfully generated variant on-demand",
		logger_domain.String(logFieldArtefactID, artefact.ID),
		logger_domain.String("variantID", variant.VariantID),
		logger_domain.Int64("sizeBytes", variant.SizeBytes),
	)

	return variant
}

// queueRemainingVariants queues background generation for all variants in
// DesiredProfiles except the one already generated. This enables lazy loading
// where the first HTTP request generates one variant, then the remaining
// variants are generated in the background.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes registryService (registry_domain.RegistryService) which fetches fresh
// artefact state.
// Takes variantGenerator (daemon_domain.OnDemandVariantGenerator) which
// creates the image variants.
// Takes artefact (*registry_dto.ArtefactMeta) which identifies the source
// image.
// Takes alreadyGenerated (string) which names the variant already created.
//
// Concurrency: uses singleflight to deduplicate concurrent calls for the same
// artefact - only the first caller spawns a background goroutine; subsequent
// callers share the in-flight result. The goroutine detaches cancellation from
// the parent context but preserves tracing values, and checks for existing
// variants before generating to avoid doing the same work twice.
func (builder *HTTPRouterBuilder) queueRemainingVariants(
	ctx context.Context,
	registryService registry_domain.RegistryService,
	variantGenerator daemon_domain.OnDemandVariantGenerator,
	artefact *registry_dto.ArtefactMeta,
	alreadyGenerated string,
) {
	ctx, l := logger_domain.From(ctx, log)

	profileNames := collectMissingVariantProfiles(artefact, alreadyGenerated)
	if len(profileNames) == 0 {
		l.Trace("No additional variants to queue for background generation")
		return
	}

	l.Trace("Queueing background variant generation",
		logger_domain.Int("variantCount", len(profileNames)),
		logger_domain.String(logFieldArtefactID, artefact.ID))

	ch := builder.variantGenerationGroup.DoChan(artefact.ID, func() (any, error) {
		defer goroutine.RecoverPanic(context.WithoutCancel(ctx), "daemon.variantGeneration")
		bgCtx := context.WithoutCancel(ctx)
		for _, profileName := range profileNames {
			freshArtefact, err := registryService.GetArtefact(bgCtx, artefact.ID)
			if err == nil && variantExistsInArtefact(freshArtefact, profileName) {
				l.Trace("Variant already exists, skipping background generation",
					logger_domain.String(logFieldProfileName, profileName))
				continue
			}
			generateBackgroundVariant(bgCtx, variantGenerator, artefact, profileName)
		}
		return nil, nil
	})

	go func() { <-ch }()
}

// videoQualityInfo holds resolution and bandwidth details for HLS manifests.
type videoQualityInfo struct {
	// resolution is the video size in WIDTHxHEIGHT format (e.g. 1920x1080).
	resolution string

	// bandwidth is the bit rate in bits per second for this quality level.
	bandwidth int
}

// hlsQualityConfigs maps quality names to their resolution and bandwidth.
var hlsQualityConfigs = map[string]videoQualityInfo{
	"2160p": {resolution: "3840x2160", bandwidth: 15000000},
	"1440p": {resolution: "2560x1440", bandwidth: 8000000},
	"1080p": {resolution: "1920x1080", bandwidth: 5000000},
	"720p":  {resolution: "1280x720", bandwidth: 2500000},
	"480p":  {resolution: "854x480", bandwidth: 1000000},
	"360p":  {resolution: "640x360", bandwidth: 500000},
}

// serveVideoMasterPlaylist returns an HTTP handler that serves the HLS master
// playlist for a video artefact.
//
// Takes registryService (registry_domain.RegistryService) which provides access to
// video artefact metadata.
//
// Returns http.HandlerFunc which builds and serves the master M3U8 playlist.
func (*HTTPRouterBuilder) serveVideoMasterPlaylist(
	registryService registry_domain.RegistryService,
	disableHTTPCache bool,
) http.HandlerFunc {
	playlistCacheControl := []string{cacheControlForMode(disableHTTPCache, cacheControlMutableAsset)}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		artefactID := chi.URLParam(r, "artefactID")

		ctx, span, l := log.Span(ctx, "serveVideoMasterPlaylist",
			logger_domain.String(logFieldArtefactID, artefactID),
		)
		defer span.End()

		if artefactID == "" {
			l.ReportError(span, errors.New("empty artefact ID"), "Empty video artefact ID")
			http.NotFound(w, r)
			return
		}

		artefact, err := registryService.GetArtefact(ctx, artefactID)
		if err != nil {
			l.ReportError(span, err, "Failed to find video artefact")
			http.NotFound(w, r)
			return
		}

		playlist := buildMasterPlaylist(artefact)

		h := w.Header()
		h[headerContentType] = headerValContentTypeMPEGURL
		h[headerCacheControl] = playlistCacheControl
		_, _ = w.Write([]byte(playlist))
	}
}

// serveVideoVariantPlaylist returns an HTTP handler that serves an HLS variant
// playlist for a specific quality level.
//
// Takes registryService (registry_domain.RegistryService) which provides access to
// video variant chunks.
//
// Returns http.HandlerFunc which builds and serves the variant M3U8 playlist.
func (*HTTPRouterBuilder) serveVideoVariantPlaylist(
	registryService registry_domain.RegistryService,
	disableHTTPCache bool,
) http.HandlerFunc {
	playlistCacheControl := []string{cacheControlForMode(disableHTTPCache, cacheControlMutableAsset)}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		artefactID := chi.URLParam(r, "artefactID")
		quality := chi.URLParam(r, "quality")

		ctx, span, l := log.Span(ctx, "serveVideoVariantPlaylist",
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.String(logFieldQuality, quality),
		)
		defer span.End()

		if artefactID == "" || quality == "" {
			l.ReportError(span, errors.New("missing parameters"), "Missing artefact ID or quality")
			http.NotFound(w, r)
			return
		}

		variantID := hlsVariantPrefix + quality
		artefact, err := registryService.GetArtefact(ctx, artefactID)
		if err != nil {
			l.ReportError(span, err, "Failed to find video artefact")
			http.NotFound(w, r)
			return
		}

		variant := findVariantByID(artefact.ActualVariants, variantID)
		if variant == nil {
			l.Warn(msgVariantNotFound, logger_domain.String("variantID", variantID))
			http.NotFound(w, r)
			return
		}

		playlist := buildVariantPlaylist(variant)

		h := w.Header()
		h[headerContentType] = headerValContentTypeMPEGURL
		h[headerCacheControl] = playlistCacheControl
		_, _ = w.Write([]byte(playlist))
	}
}

// serveVideoChunk returns an HTTP handler that serves a single HLS video
// chunk.
//
// Takes registryService (registry_domain.RegistryService) which provides access to
// the chunk data.
//
// Returns http.HandlerFunc which sends the chunk data to the client.
func (*HTTPRouterBuilder) serveVideoChunk(
	registryService registry_domain.RegistryService,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		artefactID := chi.URLParam(r, "artefactID")
		quality := chi.URLParam(r, "quality")
		chunkID := chi.URLParam(r, "chunk")

		ctx, span, l := log.Span(ctx, "serveVideoChunk",
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.String(logFieldQuality, quality),
			logger_domain.String(logFieldChunkID, chunkID),
		)
		defer span.End()

		if artefactID == "" || quality == "" || chunkID == "" {
			l.ReportError(span, errors.New("missing parameters"), "Missing chunk parameters")
			http.NotFound(w, r)
			return
		}

		variant, chunk, err := lookupArtefactChunk(ctx, registryService, artefactID, quality, chunkID)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		chunkData, err := registryService.GetVariantChunk(ctx, variant, chunkID)
		if err != nil {
			l.ReportError(span, err, "Failed to get chunk data")
			http.Error(w, "Failed to load video chunk", http.StatusInternalServerError)
			return
		}

		streamChunkToResponse(ctx, w, chunkData, chunk)
	}
}

// NewHTTPRouterBuilder creates a new HTTP router builder.
//
// Takes artefactCache (cache_domain.Cache) which provides the backing store
// for artefact metadata caching. May be nil to disable caching.
//
// Returns daemon_domain.RouterBuilder which is the configured builder ready
// for use.
func NewHTTPRouterBuilder(artefactCache cache_domain.Cache[string, *registry_dto.ArtefactMeta]) daemon_domain.RouterBuilder {
	return &HTTPRouterBuilder{
		artefactCache: artefactCache,
	}
}

// buildAllowedOrigins builds the list of allowed CORS origins from the router
// settings.
//
// When no public domain is set, returns an empty slice so that cross-origin
// requests are rejected by default. This is the safe default - deployments
// that need CORS must configure PublicDomain explicitly.
//
// Takes routerConfig (*daemon_domain.RouterConfig) which provides the public
// domain and HTTPS setting.
//
// Returns []string which contains the allowed origins.
func buildAllowedOrigins(routerConfig *daemon_domain.RouterConfig) []string {
	if routerConfig.PublicDomain == "" {
		log.Warn("PublicDomain is not configured - cross-origin requests will be rejected. Set PublicDomain to enable CORS.")
		return nil
	}
	origins := []string{
		fmt.Sprintf("https://%s", routerConfig.PublicDomain),
	}
	if !routerConfig.ForceHTTPS {
		origins = append(origins, fmt.Sprintf("http://%s", routerConfig.PublicDomain))
	}
	return origins
}

// cacheControlForMode returns the appropriate Cache-Control header value,
// using no-cache to force ETag revalidation when HTTP caching is disabled
// (dev mode) and the provided production value when enabled (prod mode).
//
// Takes disableHTTPCache (bool) which indicates whether HTTP caching is
// disabled.
// Takes prodValue (string) which is the Cache-Control value to use in
// production mode.
//
// Returns string which is the resolved Cache-Control header value.
func cacheControlForMode(disableHTTPCache bool, prodValue string) string {
	if disableHTTPCache {
		return cacheControlNoCache
	}
	return prodValue
}

// fetchStaticArtefact gets an artefact from the registry and handles
// not-found errors.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes registryService (registry_domain.RegistryService) which provides access to
// the artefact registry.
// Takes artefactID (string) which is the unique identifier of the artefact.
// Takes span (trace.Span) which records error status when fetching fails.
//
// Returns *registry_dto.ArtefactMeta which holds the artefact metadata.
// Returns bool which is true if the artefact was found, false otherwise.
func fetchStaticArtefact(
	ctx context.Context,
	registryService registry_domain.RegistryService,
	artefactID string,
	span trace.Span,
) (*registry_dto.ArtefactMeta, bool) {
	ctx, l := logger_domain.From(ctx, log)

	artefact, err := registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		if errors.Is(err, registry_domain.ErrArtefactNotFound) {
			l.Internal("Artefact not found in registry",
				logger_domain.String(logFieldArtefactID, artefactID))
			span.SetStatus(codes.Error, "Artefact not found")
			return nil, false
		}
		l.ReportError(span, err, "Failed to get artefact from registry",
			logger_domain.String(logFieldArtefactID, artefactID))
		return nil, false
	}
	return artefact, true
}

// selectStaticVariant picks the best variant from an artefact based on the
// given settings.
//
// Takes r (*http.Request) which provides Accept-Encoding headers for choosing
// a compressed variant.
// Takes artefact (*registry_dto.ArtefactMeta) which holds the available
// variants.
// Takes config (staticArtefactConfig) which sets compression and type options.
//
// Returns *registry_dto.Variant which is the best matching variant, or the
// source variant if no better match is found.
func selectStaticVariant(
	r *http.Request,
	artefact *registry_dto.ArtefactMeta,
	config staticArtefactConfig,
) *registry_dto.Variant {
	if config.useCompression {
		if v := findBestCompressedVariant(r, artefact, config.preferredType); v != nil {
			return v
		}
	}
	if config.preferredType != "" && config.preferredType != variantSource {
		if v := findVariantByTag(artefact, "type", config.preferredType); v != nil {
			return v
		}
	}
	return findVariantByID(artefact.ActualVariants, variantSource)
}

// writeStaticVariantResponse writes a variant to the response with the correct
// headers.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes w (http.ResponseWriter) which receives the response data and headers.
// Takes registryService (registry_domain.RegistryService) which provides access to
// variant data.
// Takes variant (*registry_dto.Variant) which is the variant to serve.
// Takes config (staticArtefactConfig) which holds the default MIME type and cache
// settings.
// Takes span (trace.Span) which receives status updates for tracing.
func writeStaticVariantResponse(
	ctx context.Context,
	w http.ResponseWriter,
	registryService registry_domain.RegistryService,
	variant *registry_dto.Variant,
	config staticArtefactConfig,
	span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	blobStream, err := registryService.GetVariantData(ctx, variant)
	if err != nil {
		l.ReportError(span, err, "Failed to get blob stream for variant",
			logger_domain.String(logFieldVariantID, variant.VariantID))
		http.Error(w, errMessageInternalServer, http.StatusInternalServerError)
		return
	}
	defer func() { _ = blobStream.Close() }()

	httpResponseSize.Record(ctx, variant.SizeBytes)

	contentType := config.defaultMimeType
	if mt := variant.MetadataTags.Get(registry_dto.TagMimeType); mt != "" {
		contentType = mt
	}

	etag := variant.MetadataTags.Get(registry_dto.TagEtag)
	h := w.Header()
	h[headerContentType] = []string{contentType}
	h[headerETag] = []string{etag}
	h[headerCacheControl] = []string{config.cacheMaxAge}

	if encoding := variant.MetadataTags.Get(registry_dto.TagContentEncoding); encoding != "" {
		h[headerContentEncoding] = []string{encoding}
	}
	if variant.SizeBytes > 0 {
		h[headerContentLength] = []string{strconv.FormatInt(variant.SizeBytes, 10)}
	}

	w.WriteHeader(http.StatusOK)
	span.SetStatus(codes.Ok, "Artefact served successfully")

	if _, err := io.Copy(w, blobStream); err != nil {
		l.Warn("Error while streaming artefact to client",
			logger_domain.String(logFieldError, err.Error()))
	}
}

// serveStaticArtefactHandler creates a handler for serving static artefacts
// with the given settings.
//
// Takes registryService (registry_domain.RegistryService) which provides access
// to the artefact registry.
// Takes config (staticArtefactConfig) which sets the artefact options.
// Takes spanName (string) which names the tracing span.
//
// Returns http.HandlerFunc which serves the static artefact with content
// type selection and caching.
func serveStaticArtefactHandler(
	registryService registry_domain.RegistryService,
	config staticArtefactConfig,
	spanName string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span, l := log.Span(r.Context(), spanName,
			logger_domain.String(logFieldArtefactID, config.artefactID),
			logger_domain.String("method", r.Method),
			logger_domain.String(logFieldPath, r.URL.Path),
		)
		defer span.End()

		artefact, ok := fetchStaticArtefact(ctx, registryService, config.artefactID, span)
		if !ok {
			http.NotFound(w, r)
			return
		}

		artefactServeCount.Add(ctx, 1)
		bestVariant := selectStaticVariant(r, artefact, config)
		if bestVariant == nil {
			err := fmt.Errorf("no suitable variant found for artefact %s", artefact.ID)
			l.ReportError(span, err, "Consistency error: artefact exists but has no suitable variant")
			http.Error(w, errMessageInternalServer, http.StatusInternalServerError)
			return
		}

		etag := bestVariant.MetadataTags.Get(registry_dto.TagEtag)
		if etag != "" && r.Header.Get(headerIfNoneMatch) == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		writeStaticVariantResponse(ctx, w, registryService, bestVariant, config, span)
	}
}

// lookupArtefact finds an artefact by its ID, checking the cache first, then
// loading from storage. If the artefact is not found by ID, it tries to find
// it using the storage key as a fallback.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes span (trace.Span) which provides tracing context for error reports.
// Takes cache (*artefactMetadataCache) which holds cached artefact metadata.
// Takes registryService (registry_domain.RegistryService) which provides access
// to the registry for the storage key fallback.
// Takes artefactID (string) which is the artefact ID or storage key to find.
//
// Returns artefactLookupResult which contains the artefact, any error, the
// HTTP status code, and whether it was found by storage key.
func lookupArtefact(
	ctx context.Context,
	span trace.Span,
	cache *artefactMetadataCache,
	registryService registry_domain.RegistryService,
	artefactID string,
) artefactLookupResult {
	ctx, l := logger_domain.From(ctx, log)

	if cached, hit := cache.Get(ctx, artefactID); hit {
		artefactMetadataCacheHitCount.Add(ctx, 1)
		return artefactLookupResult{err: nil, artefact: cached, httpStatus: 0, foundByStorageKey: false}
	}

	artefactMetadataCacheMissCount.Add(ctx, 1)

	artefact, err := cache.GetOrLoad(ctx, artefactID)
	if err == nil {
		return artefactLookupResult{err: nil, artefact: artefact, httpStatus: 0, foundByStorageKey: false}
	}

	if !errors.Is(err, registry_domain.ErrArtefactNotFound) {
		l.ReportError(span, err, "Failed to find artefact by ID")
		return artefactLookupResult{err: err, artefact: nil, httpStatus: http.StatusInternalServerError, foundByStorageKey: false}
	}

	return lookupArtefactByStorageKey(ctx, span, registryService, artefactID)
}

// lookupArtefactByStorageKey attempts to find an artefact by its storage key
// as a fallback.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes span (trace.Span) which records the operation status.
// Takes registryService (registry_domain.RegistryService) which provides artefact
// lookup.
// Takes artefactID (string) which is the storage key to search for.
//
// Returns artefactLookupResult which contains the artefact if found, or error
// details with the appropriate HTTP status code.
func lookupArtefactByStorageKey(
	ctx context.Context,
	span trace.Span,
	registryService registry_domain.RegistryService,
	artefactID string,
) artefactLookupResult {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("Artefact not found by ID, trying storage key lookup", logger_domain.String(logFieldPath, artefactID))

	artefact, err := registryService.FindArtefactByVariantStorageKey(ctx, artefactID)
	if err != nil {
		if errors.Is(err, registry_domain.ErrArtefactNotFound) {
			l.Internal("Asset not found in registry by artefact ID or storage key")
			span.SetStatus(codes.Error, "Asset not found")
			return artefactLookupResult{err: err, artefact: nil, httpStatus: http.StatusNotFound, foundByStorageKey: false}
		}
		l.ReportError(span, err, "Failed to find artefact by storage key")
		return artefactLookupResult{err: err, artefact: nil, httpStatus: http.StatusInternalServerError, foundByStorageKey: false}
	}

	return artefactLookupResult{err: nil, artefact: artefact, httpStatus: 0, foundByStorageKey: true}
}

// serveVariantResponse writes the variant data to the HTTP response.
//
// It finds the best compressed variant for the request, checks for ETag
// matches to return 304 Not Modified when possible, and streams the variant
// data to the client with proper cache headers.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes vrc (variantResolutionContext) which holds the response writer and
// registry service.
// Takes artefact (*registry_dto.ArtefactMeta) which holds the artefact
// details.
// Takes requestedVariant (*registry_dto.Variant) which is the variant to
// serve.
func serveVariantResponse(
	ctx context.Context,
	vrc variantResolutionContext,
	artefact *registry_dto.ArtefactMeta,
	requestedVariant *registry_dto.Variant,
) {
	ctx, l := logger_domain.From(ctx, log)

	bestVariant := findBestCompressedVariant(vrc.r, artefact, requestedVariant.VariantID)
	if bestVariant == nil {
		bestVariant = requestedVariant
	}

	etag := bestVariant.MetadataTags.Get(registry_dto.TagEtag)
	if etag != "" && vrc.r.Header.Get(headerIfNoneMatch) == etag {
		vrc.w.WriteHeader(http.StatusNotModified)
		return
	}

	blobStream, err := vrc.registryService.GetVariantData(ctx, bestVariant)
	if err != nil {
		l.ReportError(vrc.span, err, "Failed to get blob stream for variant",
			logger_domain.String(logFieldArtefactID, artefact.ID),
			logger_domain.String(logFieldVariantID, bestVariant.VariantID),
		)
		http.Error(vrc.w, errMessageInternalServer, http.StatusInternalServerError)
		return
	}
	defer func() { _ = blobStream.Close() }()

	httpResponseSize.Record(ctx, bestVariant.SizeBytes)

	h := vrc.w.Header()
	h[headerContentType] = []string{bestVariant.MimeType}
	h[headerETag] = []string{etag}
	h[headerCacheControl] = []string{vrc.cacheControl}

	if encoding := bestVariant.MetadataTags.Get(registry_dto.TagContentEncoding); encoding != "" {
		h[headerContentEncoding] = []string{encoding}
	}
	if bestVariant.SizeBytes > 0 {
		h[headerContentLength] = []string{strconv.FormatInt(bestVariant.SizeBytes, 10)}
	}

	vrc.w.WriteHeader(http.StatusOK)
	vrc.span.SetStatus(codes.Ok, "Artefact served successfully")
	artefactServeCount.Add(ctx, 1)

	if _, err := io.Copy(vrc.w, blobStream); err != nil {
		l.Warn("Error while streaming artefact to client", logger_domain.String(logFieldError, err.Error()))
	}
}

// findBestCompressedVariant selects the best compressed variant based on the
// client's Accept-Encoding header.
//
// Takes r (*http.Request) which provides the Accept-Encoding header.
// Takes artefact (*registry_dto.ArtefactMeta) which contains available
// variants.
// Takes baseVariantID (string) which identifies the base variant to match.
//
// Returns *registry_dto.Variant which is the best matching compressed variant,
// or the base variant if no compressed version is available.
func findBestCompressedVariant(r *http.Request, artefact *registry_dto.ArtefactMeta, baseVariantID string) *registry_dto.Variant {
	acceptEncoding := r.Header.Get("Accept-Encoding")

	if strings.Contains(acceptEncoding, "br") {
		if v := findVariantByID(artefact.ActualVariants, baseVariantID+"_br"); v != nil {
			return v
		}
		if v := findVariantByTag(artefact, "contentEncoding", "br"); v != nil {
			return v
		}
	}

	if strings.Contains(acceptEncoding, "gz") || strings.Contains(acceptEncoding, "gzip") {
		if v := findVariantByID(artefact.ActualVariants, baseVariantID+"_gzip"); v != nil {
			return v
		}
		if v := findVariantByTag(artefact, "contentEncoding", "gzip"); v != nil {
			return v
		}
	}

	return findVariantByID(artefact.ActualVariants, baseVariantID)
}

// serveEmbeddedFrontend creates a handler that serves static files from the
// embedded frontend.
//
// In production mode (when watchMode is false), requests for .es.js files are
// automatically redirected to the minified .min.es.js versions for optimal
// performance.
//
// When the requested file is not found, sends a 404 Not Found response. When
// the client's If-None-Match header matches the file's ETag, sends a 304 Not
// Modified response.
//
// Takes watchMode (bool) which indicates whether the server is in dev mode.
//
// Returns http.HandlerFunc which serves the embedded frontend assets.
func serveEmbeddedFrontend(watchMode bool, disableHTTPCache bool) http.HandlerFunc {
	frontendCacheControl := cacheControlForMode(disableHTTPCache, cacheControlMutableAsset)
	return func(w http.ResponseWriter, r *http.Request) {
		fileParam := chi.URLParam(r, "*")

		const forceUnminified = true
		if !forceUnminified && !watchMode && strings.HasSuffix(fileParam, ".es.js") && !strings.Contains(fileParam, ".min.") {
			fileParam = strings.Replace(fileParam, ".es.js", ".min.es.js", 1)
		}

		basePath := path.Join("built", fileParam)

		finalPath := daemon_frontend.DetermineBestAssetPath(r.Context(), basePath, r.Header.Get("Accept-Encoding"))

		asset, found := daemon_frontend.GetAsset(r.Context(), finalPath)
		if !found {
			asset, found = daemon_frontend.GetAsset(r.Context(), basePath)
			if !found {
				http.NotFound(w, r)
				return
			}
		}

		h := w.Header()
		h[headerETag] = []string{asset.ETag}
		h[headerCacheControl] = []string{frontendCacheControl}

		if match := r.Header.Get(headerIfNoneMatch); match != "" && strings.Contains(match, asset.ETag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		h[headerContentType] = []string{asset.MimeType}
		if asset.Encoding != "" {
			h[headerContentEncoding] = []string{asset.Encoding}
		}
		w.Header().Add("Vary", "Accept-Encoding")

		http.ServeContent(w, r, filepath.Base(basePath), time.Time{}, bytes.NewReader(asset.Content))
	}
}

// findVariantByID finds a variant with the given ID in a slice.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes id (string) which is the ID to find.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if not
// found.
func findVariantByID(variants []registry_dto.Variant, id string) *registry_dto.Variant {
	for i := range variants {
		if variants[i].VariantID == id {
			return &variants[i]
		}
	}
	return nil
}

// findVariantByStorageKey searches for a variant with the given storage key.
//
// Takes variants ([]registry_dto.Variant) which is the slice to search.
// Takes storageKey (string) which is the key to match against.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if not
// found.
func findVariantByStorageKey(variants []registry_dto.Variant, storageKey string) *registry_dto.Variant {
	for i := range variants {
		if variants[i].StorageKey == storageKey {
			return &variants[i]
		}
	}
	return nil
}

// collectMissingVariantProfiles returns variant profile names that have not
// been generated yet.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the desired
// profiles to check.
// Takes alreadyGenerated (string) which specifies a profile name to skip.
//
// Returns []string which contains profile names that still need to be
// generated.
func collectMissingVariantProfiles(artefact *registry_dto.ArtefactMeta, alreadyGenerated string) []string {
	profiles := make([]string, 0, len(artefact.DesiredProfiles))
	for i := range artefact.DesiredProfiles {
		profileName := artefact.DesiredProfiles[i].Name
		if profileName != alreadyGenerated && profileName != variantSource {
			profiles = append(profiles, profileName)
		}
	}
	return profiles
}

// variantExistsInArtefact checks whether a variant with the given ID exists in
// the artefact.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the variants to
// search.
// Takes variantID (string) which is the ID to look for.
//
// Returns bool which is true if the variant exists, false otherwise.
func variantExistsInArtefact(artefact *registry_dto.ArtefactMeta, variantID string) bool {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == variantID {
			return true
		}
	}
	return false
}

// generateBackgroundVariant creates a single variant and logs the result.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes variantGenerator (daemon_domain.OnDemandVariantGenerator) which
// creates the variant.
// Takes artefact (*registry_dto.ArtefactMeta) which identifies the source
// artefact.
// Takes profileName (string) which specifies the variant profile to create.
func generateBackgroundVariant(
	ctx context.Context,
	variantGenerator daemon_domain.OnDemandVariantGenerator,
	artefact *registry_dto.ArtefactMeta,
	profileName string,
) {
	ctx, l := logger_domain.From(ctx, log)

	startTime := time.Now()
	_, err := variantGenerator.GenerateVariant(ctx, artefact, profileName)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		l.Warn("Background variant generation failed",
			logger_domain.String(logFieldArtefactID, artefact.ID),
			logger_domain.String(logFieldProfileName, profileName),
			logger_domain.Error(err))
		return
	}
	l.Trace("Background variant generated successfully",
		logger_domain.String(logFieldArtefactID, artefact.ID),
		logger_domain.String(logFieldProfileName, profileName),
		logger_domain.Int64("durationMs", duration))
}

// buildMasterPlaylist creates an HLS master playlist from video profiles.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the video
// profiles to include in the playlist.
//
// Returns string which is the complete m3u8 master playlist content.
func buildMasterPlaylist(artefact *registry_dto.ArtefactMeta) string {
	var buffer bytes.Buffer
	_, _ = buffer.WriteString("#EXTM3U\n")

	for i := range artefact.DesiredProfiles {
		np := &artefact.DesiredProfiles[i]
		if !strings.HasPrefix(np.Name, hlsVariantPrefix) {
			continue
		}

		quality := strings.TrimPrefix(np.Name, hlsVariantPrefix)
		qualityInfo, ok := hlsQualityConfigs[quality]
		if !ok {
			continue
		}

		_, _ = fmt.Fprintf(&buffer, "#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s\n",
			qualityInfo.bandwidth, qualityInfo.resolution)
		_, _ = fmt.Fprintf(&buffer, "%s/manifest.m3u8\n", quality)
	}

	return buffer.String()
}

// buildVariantPlaylist creates an HLS variant playlist from video chunks.
//
// Takes variant (*registry_dto.Variant) which holds the video chunks to
// include.
//
// Returns string which is the complete M3U8 playlist content.
func buildVariantPlaylist(variant *registry_dto.Variant) string {
	var buffer bytes.Buffer
	_, _ = buffer.WriteString("#EXTM3U\n")
	_, _ = buffer.WriteString("#EXT-X-VERSION:3\n")

	maxDuration := 0.0
	for i := range variant.Chunks {
		chunk := &variant.Chunks[i]
		if chunk.DurationSeconds != nil && *chunk.DurationSeconds > maxDuration {
			maxDuration = *chunk.DurationSeconds
		}
	}
	if maxDuration == 0 {
		maxDuration = hlsDefaultSegmentDuration
	}

	_, _ = fmt.Fprintf(&buffer, "#EXT-X-TARGETDURATION:%d\n", int(maxDuration+0.5))
	_, _ = buffer.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

	for i := range variant.Chunks {
		chunk := &variant.Chunks[i]
		if chunk.MimeType != "video/mp2t" && chunk.MimeType != "video/MP2T" {
			continue
		}

		duration := hlsDefaultSegmentDuration
		if chunk.DurationSeconds != nil {
			duration = *chunk.DurationSeconds
		}

		_, _ = fmt.Fprintf(&buffer, "#EXTINF:%.6f,\n", duration)
		_, _ = fmt.Fprintf(&buffer, "%s\n", chunk.ChunkID)
	}

	_, _ = buffer.WriteString("#EXT-X-ENDLIST\n")
	return buffer.String()
}

// lookupArtefactChunk finds the variant and chunk for a video artefact.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes registryService (registry_domain.RegistryService) which provides access
// to stored artefacts.
// Takes artefactID (string) which identifies the video artefact.
// Takes quality (string) which identifies the quality variant to find.
// Takes chunkID (string) which identifies the chunk to find.
//
// Returns *registry_dto.Variant which is the matched quality variant.
// Returns *registry_dto.VariantChunk which is the requested chunk.
// Returns error when the artefact, variant, or chunk cannot be found.
func lookupArtefactChunk(
	ctx context.Context,
	registryService registry_domain.RegistryService,
	artefactID, quality, chunkID string,
) (*registry_dto.Variant, *registry_dto.VariantChunk, error) {
	ctx, l := logger_domain.From(ctx, log)

	variantID := hlsVariantPrefix + quality
	artefact, err := registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		l.Warn("Failed to find video artefact", logger_domain.Error(err))
		return nil, nil, fmt.Errorf("getting video artefact %q: %w", artefactID, err)
	}

	variant := findVariantByID(artefact.ActualVariants, variantID)
	if variant == nil {
		l.Warn(msgVariantNotFound, logger_domain.String("variantID", variantID))
		return nil, nil, fmt.Errorf("variant not found: %s", variantID)
	}

	chunk := findChunkByID(variant, chunkID)
	if chunk == nil {
		l.Warn("Chunk not found", logger_domain.String(logFieldChunkID, chunkID))
		return nil, nil, fmt.Errorf("chunk not found: %s", chunkID)
	}

	return variant, chunk, nil
}

// streamChunkToResponse writes chunk data to an HTTP response.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes w (http.ResponseWriter) which receives the streamed data.
// Takes chunkData (io.ReadCloser) which provides the content to stream.
// Takes chunk (*registry_dto.VariantChunk) which provides metadata for headers.
func streamChunkToResponse(
	ctx context.Context,
	w http.ResponseWriter,
	chunkData io.ReadCloser,
	chunk *registry_dto.VariantChunk,
) {
	ctx, l := logger_domain.From(ctx, log)

	defer func() {
		if closeErr := chunkData.Close(); closeErr != nil {
			l.Warn("Failed to close chunk data", logger_domain.Error(closeErr))
		}
	}()

	h := w.Header()
	h[headerContentType] = []string{chunk.MimeType}
	h[headerCacheControl] = headerValCacheImmutable
	h[headerContentLength] = []string{strconv.FormatInt(chunk.SizeBytes, 10)}

	if _, err := io.Copy(w, chunkData); err != nil {
		l.Warn("Error streaming chunk", logger_domain.Error(err))
	}
}

// findChunkByID finds a chunk with the given ID within a variant.
//
// Takes variant (*registry_dto.Variant) which contains the chunks to search.
// Takes chunkID (string) which is the ID of the chunk to find.
//
// Returns *registry_dto.VariantChunk which is the matching chunk, or nil if
// not found.
func findChunkByID(variant *registry_dto.Variant, chunkID string) *registry_dto.VariantChunk {
	for i := range variant.Chunks {
		if variant.Chunks[i].ChunkID == chunkID {
			return &variant.Chunks[i]
		}
	}
	return nil
}
