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

package daemon_domain

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"piko.sh/piko/internal/analytics/analytics_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/seo/seo_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

// DaemonService defines the interface for the main daemon that manages the
// HTTP server lifecycle, file watching, and asset processing.
type DaemonService interface {
	// RunDev runs the application in development mode.
	//
	// Returns error when the development server fails to start or stops with an
	// error.
	RunDev(parentCtx context.Context) error

	// RunProd runs the application in production mode.
	//
	// Returns error when the application fails to start or encounters a fatal
	// error.
	RunProd(parentCtx context.Context) error

	// Stop halts the running process and releases its resources.
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Stop(ctx context.Context) error

	// GetHandler returns the HTTP handler for serving requests.
	// Use it to test without starting the full server.
	//
	// Returns http.Handler which processes incoming HTTP requests.
	GetHandler() http.Handler
}

// ServerAdapter defines an interface for HTTP server operations.
// It allows different server types to be used for testing.
type ServerAdapter interface {
	// ListenAndServe starts the HTTP server on the given address.
	//
	// Takes address (string) which specifies the TCP address to listen on.
	// Takes handler (http.Handler) which processes incoming HTTP requests.
	//
	// Returns error when the server fails to start or encounters a fatal error.
	ListenAndServe(address string, handler http.Handler) error

	// Shutdown stops the service in a controlled way.
	//
	// Returns error when the shutdown fails or the context is cancelled.
	Shutdown(ctx context.Context) error

	// SetOnBound registers an optional callback that is invoked after the
	// server successfully binds to a port, before it starts serving requests.
	// The callback receives the resolved listen address.
	SetOnBound(func(address string))
}

// RouteProvider defines the contract for services that can mount routes to a
// Chi router.
type RouteProvider interface {
	// MountRoutes registers HTTP routes on the given router.
	//
	// Takes router (*chi.Mux) which is the router to add routes to.
	MountRoutes(router *chi.Mux)
}

// RouterDependencies groups all dependencies needed to build the HTTP router.
// This reduces parameter count and makes the BuildRouter signature more
// maintainable.
type RouterDependencies struct {
	// RegistryService provides access to the asset registry.
	RegistryService registry_domain.RegistryService

	// UserRouter handles user-defined routes.
	UserRouter http.Handler

	// VariantGenerator creates image variants on demand.
	VariantGenerator OnDemandVariantGenerator

	// PresignUploadHandler handles presigned URL uploads; may be nil if not
	// enabled.
	PresignUploadHandler http.Handler

	// PresignDownloadHandler handles presigned URL downloads.
	// May be nil if not enabled.
	PresignDownloadHandler http.Handler

	// PublicDownloadHandler handles public file downloads without authentication;
	// may be nil if not enabled.
	PublicDownloadHandler http.Handler

	// RateLimitService provides rate limiting for the rate limiting middleware.
	RateLimitService security_domain.RateLimitService

	// AuthProvider resolves authentication state from requests. Nil
	// means no auth middleware is installed.
	AuthProvider daemon_dto.AuthProvider

	// AuthGuardConfig controls route-level authentication enforcement.
	// Nil means no route protection middleware is installed.
	AuthGuardConfig *daemon_dto.AuthGuardConfig

	// CSPConfig provides the computed CSP settings for security headers
	// middleware.
	CSPConfig security_dto.CSPRuntimeConfig

	// AnalyticsService distributes analytics events to registered
	// collectors. Nil means no analytics middleware is installed.
	AnalyticsService *analytics_domain.Service
}

// RouterConfig holds the exact fields the HTTP router needs from the server
// configuration. This replaces passing the full ServerConfig to BuildRouter,
// making the router's requirements explicit and decoupled.
type RouterConfig struct {
	// DevAPIHandler serves the /_piko/dev/api/* REST endpoints for the
	// dev tools overlay widget. Nil in production mode.
	DevAPIHandler DevAPIHandlerPort

	// DevEventsBroadcaster serves the /_piko/dev/events SSE endpoint for
	// dev-mode build notifications. Nil in production mode.
	DevEventsBroadcaster http.Handler

	// DevPreviewHandler serves the /_piko/dev/preview/* endpoints for
	// dev-mode template previewing. Nil in production mode.
	DevPreviewHandler DevPreviewHandlerPort

	// PublicDomain is the public-facing domain for CORS origins.
	PublicDomain string

	// DistServePath is the URL path prefix for serving the embedded frontend.
	DistServePath string

	// ArtefactServePath is the URL path prefix for serving artefacts.
	ArtefactServePath string

	// Reporting holds the Reporting-Endpoints header configuration.
	Reporting security_dto.ReportingValues

	// Port is the server port, used in log messages.
	Port string

	// SecurityHeaders holds the security headers middleware configuration.
	SecurityHeaders security_dto.SecurityHeadersValues

	// RateLimit holds the rate limiting configuration.
	RateLimit security_dto.RateLimitValues

	// RequestTimeoutSeconds sets the per-request timeout; 0 means no timeout.
	RequestTimeoutSeconds int

	// MaxConcurrentRequests is the maximum concurrent in-flight requests
	// (default: 10000). Set to 0 to disable.
	MaxConcurrentRequests int

	// ForceHTTPS enables HTTPS-only CORS origins and HSTS.
	ForceHTTPS bool

	// WatchMode indicates whether the server is running in dev watch mode.
	WatchMode bool

	// DisableHTTPCache forces all static asset responses to use no-cache,
	// requiring ETag revalidation on every request. This is derived from
	// WatchMode by default but expressed as a separate flag to clarify intent.
	DisableHTTPCache bool
}

// DevAPIHandlerPort defines the interface for mounting dev API routes.
type DevAPIHandlerPort interface {
	// Mount registers the dev API routes on the given router.
	Mount(router chi.Router)
}

// DevPreviewHandlerPort defines the interface for mounting dev preview routes.
type DevPreviewHandlerPort interface {
	// Mount registers the preview API and render routes on the given router.
	Mount(router chi.Router)
}

// RouterBuilder defines how to build the final HTTP router.
// It is implemented by daemon_adapters.HTTPRouterBuilder.
type RouterBuilder interface {
	// BuildRouter constructs an HTTP router with the given configuration and
	// dependencies.
	//
	// Takes config (*RouterConfig) which provides the router-specific settings.
	// Takes deps (RouterDependencies) which groups all router dependencies.
	//
	// Returns http.Handler which is the configured router ready to serve requests.
	// Returns error when router construction fails.
	BuildRouter(
		config *RouterConfig,
		deps RouterDependencies,
	) (http.Handler, error)

	// Close stops background goroutines owned by the builder.
	// Call when the router is no longer needed.
	Close()
}

// StructValidator defines the minimal interface for struct validation.
// It is satisfied by the playground validator from the
// validation_provider_playground WDK module.
type StructValidator interface {
	// Struct validates a struct's exposed fields based on validation tags.
	//
	// Takes s (any) which is the struct to validate.
	//
	// Returns error when any field fails its validation constraint.
	Struct(s any) error
}

// HTTPHandlerDependencies contains shared dependencies required by HTTP
// handlers, including the templating service and request validator.
type HTTPHandlerDependencies struct {
	// Templater probes and renders pages and partials for HTTP requests.
	Templater templater_domain.TemplaterService

	// Validator checks action request data before processing. May be nil
	// when no validator is configured.
	Validator StructValidator
}

// BuildCacheInvalidator defines the contract for a service that holds a build
// cache which can be invalidated, typically in response to file system changes.
type BuildCacheInvalidator interface {
	// InvalidateBuildCache clears any stored build results.
	InvalidateBuildCache()
}

// SEOServicePort defines the contract for generating SEO artefacts such as
// sitemap.xml and robots.txt from a project view. This is an optional
// dependency; if nil, SEO artefact generation is skipped during hot-reload.
type SEOServicePort interface {
	// GenerateArtefacts creates SEO artefacts for the given project view.
	//
	// Takes view (*seo_dto.ProjectView) which contains the project data to
	// process.
	//
	// Returns error when artefact generation fails.
	GenerateArtefacts(ctx context.Context, view *seo_dto.ProjectView) error
}

// RouterManager handles route loading and reloading for hot-reload support.
// It implements http.Handler and is only used in development modes.
type RouterManager interface {
	// ReloadRoutes refreshes the routing configuration from the manifest store.
	//
	// Takes ctx (context.Context) which carries logging context for trace/request
	// ID propagation.
	// Takes store (ManifestStoreView) which provides access to route manifests.
	//
	// Returns error when the routes cannot be reloaded.
	ReloadRoutes(ctx context.Context, store templater_domain.ManifestStoreView) error

	http.Handler
}

// AssetPipelinePort processes build results to create transformation profiles.
// It allows tests to inject mock implementations.
type AssetPipelinePort interface {
	// ProcessBuildResult handles the annotation result from a build operation.
	//
	// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
	// build annotations to process.
	//
	// Returns error when processing the result fails.
	ProcessBuildResult(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error
}

// DrainSignaller marks a service as draining so that readiness probes return
// unhealthy before the HTTP servers shut down. This allows load balancers to
// deregister the instance during the drain period.
type DrainSignaller interface {
	// SignalDrain marks the service as draining. After this call, readiness
	// checks should return unhealthy.
	SignalDrain()
}

// SignalNotifier provides an interface for receiving OS shutdown signals.
// In production, it uses signal.NotifyContext to listen for SIGINT/SIGTERM;
// in tests, it can be mocked to control shutdown timing.
type SignalNotifier interface {
	// NotifyContext returns a context that is cancelled when a shutdown signal
	// is received. The returned cancel function should be called to release
	// resources.
	//
	// Takes parent (context.Context) which is the parent context to derive from.
	//
	// Returns ctx (context.Context) which is cancelled on shutdown.
	// Returns cancel (context.CancelFunc) which releases associated resources.
	NotifyContext(parent context.Context) (ctx context.Context, cancel context.CancelFunc)
}
