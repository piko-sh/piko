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
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/route_pattern"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// buildHeadersMaxParts is the maximum number of parts in a Link header (URL +
// rel + as + type + crossorigin).
const buildHeadersMaxParts = 5

// contextKey is a custom type for context keys to avoid conflicts.
type contextKey string

// RouteSettings holds the server configuration fields that route setup needs.
type RouteSettings struct {
	// ActionServePath is the URL base path for action endpoints.
	ActionServePath string

	// ActionMaxBodyBytes is the maximum request body size for actions.
	ActionMaxBodyBytes int64

	// MaxMultipartFormBytes limits multipart form data size.
	MaxMultipartFormBytes int64

	// DefaultMaxSSEDurationSeconds caps server-sent event streams.
	DefaultMaxSSEDurationSeconds int

	// E2EMode enables end-to-end test routes in the server.
	E2EMode bool

	// CSRFSecFetchSiteEnforcement requires CSRF tokens on browser requests
	// identified by the Sec-Fetch-Site header.
	CSRFSecFetchSiteEnforcement bool
}

// MountRoutesConfig bundles configuration parameters for
// MountRoutesFromManifest.
type MountRoutesConfig struct {
	// Router is the HTTP router where route handlers are registered.
	Router chi.Router

	// Store provides read access to page and partial templates for route setup.
	Store templater_domain.ManifestStoreView

	// CSRFService provides CSRF token checks for route handlers.
	CSRFService security_domain.CSRFTokenService

	// RateLimitService provides rate limit checking for per-action limits.
	RateLimitService security_domain.RateLimitService

	// ActionResponseCache stores cached action responses. Nil disables action
	// response caching.
	ActionResponseCache cache_domain.Cache[string, []byte]

	// CaptchaService verifies captcha tokens. Nil when captcha is disabled.
	CaptchaService captcha_domain.CaptchaServicePort

	// SiteSettings holds the website settings used by route handlers.
	SiteSettings *config.WebsiteConfig

	// AuthGuardConfig holds auth guard settings for page-level AuthPolicy
	// enforcement. Nil when no auth guard is configured.
	AuthGuardConfig *daemon_dto.AuthGuardConfig

	// CacheMiddleware is the middleware function that controls HTTP caching.
	CacheMiddleware func(next http.Handler) http.Handler

	// Actions maps action names to their handler entries for route registration.
	Actions map[string]ActionHandlerEntry

	// Deps holds shared dependencies used by HTTP route handlers.
	Deps *daemon_domain.HTTPHandlerDependencies

	// RouteSettings holds the server settings used when setting up routes.
	RouteSettings RouteSettings

	// RateLimitConfig holds the rate limiting settings.
	RateLimitConfig security_dto.RateLimitValues
}

// routeRegistrationDeps holds the dependencies needed to register routes.
type routeRegistrationDeps struct {
	// router registers GET and POST handlers for page and partial routes.
	router chi.Router

	// deps holds shared dependencies for page and partial request handlers.
	deps *daemon_domain.HTTPHandlerDependencies

	// store provides access to manifest data for rendering partial pages.
	store templater_domain.ManifestStoreView

	// siteSettings holds the website settings used when rendering pages.
	siteSettings *config.WebsiteConfig

	// cacheMiddleware adds HTTP caching headers to handlers.
	cacheMiddleware func(next http.Handler) http.Handler

	// authGuardConfig holds the auth guard config for page-level AuthPolicy
	// enforcement. Nil when no auth guard is configured.
	authGuardConfig *daemon_dto.AuthGuardConfig

	// e2eModeEnabled indicates whether E2E test mode is active; when false,
	// E2E-only pages and partials return a 404 error.
	e2eModeEnabled bool
}

// renderRespondParams groups the parameters for renderAndRespond.
type renderRespondParams struct {
	// Deps holds shared dependencies for templating and rendering.
	Deps *daemon_domain.HTTPHandlerDependencies

	// PageDef is the page definition describing the page to render.
	PageDef *templater_dto.PageDefinition

	// Entry is the page entry view specifying the page configuration.
	Entry templater_domain.PageEntryView

	// Store provides access to manifest data for error page rendering.
	Store templater_domain.ManifestStoreView

	// Config holds the website configuration settings for rendering.
	Config *config.WebsiteConfig

	// PageProbe is the probe result containing page data and link headers.
	PageProbe *templater_dto.PageProbeResult

	// Span is the tracing span for the current request.
	Span trace.Span
}

// pageErrorContext groups the shared dependencies needed when handling a page
// error response.
type pageErrorContext struct {
	// Deps holds shared dependencies for templating and error page rendering.
	Deps *daemon_domain.HTTPHandlerDependencies

	// Store provides manifest data for error page lookup.
	Store templater_domain.ManifestStoreView

	// WebsiteConfig holds the site settings used during error page rendering.
	WebsiteConfig *config.WebsiteConfig

	// Entry identifies the page that triggered the error.
	Entry templater_domain.PageEntryView

	// Span is the tracing span for the current request.
	Span trace.Span
}

// errorPageRequest groups the parameters that describe what error page to
// render.
type errorPageRequest struct {
	// Message is the user-visible error message.
	Message string

	// InternalMessage is the full internal error detail, shown only in
	// development mode.
	InternalMessage string

	// OriginalPath is the request path that triggered the error.
	OriginalPath string

	// StatusCode is the HTTP status code to return with the error page.
	StatusCode int
}

// responseTracker wraps an http.ResponseWriter to track whether a response body
// or status code has been written. Used to determine whether it is safe to render
// an error page after a rendering failure.
type responseTracker struct {
	http.ResponseWriter

	// started is true once Write or WriteHeader has been called.
	started bool
}

// Write delegates to the underlying ResponseWriter and marks the response as
// started.
//
// Takes b ([]byte) which is the data to write to the response body.
//
// Returns int which is the number of bytes written.
// Returns error when the underlying writer fails.
func (rt *responseTracker) Write(b []byte) (int, error) {
	rt.started = true
	return rt.ResponseWriter.Write(b)
}

// WriteHeader delegates to the underlying ResponseWriter and marks the
// response as started.
//
// Takes code (int) which is the HTTP status code to send.
func (rt *responseTracker) WriteHeader(code int) {
	rt.started = true
	rt.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying ResponseWriter if it implements
// http.Flusher.
func (rt *responseTracker) Flush() {
	if f, ok := rt.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter for middleware that needs to
// inspect it.
//
// Returns http.ResponseWriter which is the wrapped response writer.
func (rt *responseTracker) Unwrap() http.ResponseWriter {
	return rt.ResponseWriter
}

// MountRoutesFromManifest registers all page and partial routes from the
// manifest store onto the given Chi router. It applies caching and middleware
// based on the provided settings.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes mountConfig (*MountRoutesConfig) which provides the router, dependencies,
// manifest store, and middleware settings.
func MountRoutesFromManifest(ctx context.Context, mountConfig *MountRoutesConfig) {
	ctx, span, l := log.Span(ctx, "MountRoutesFromManifest")
	defer span.End()

	l.Internal("Starting route registration from manifest")
	span.SetAttributes(attribute.Int("actionCount", len(mountConfig.Actions)))

	e2eModeEnabled := mountConfig.RouteSettings.E2EMode
	regDeps := &routeRegistrationDeps{
		router:          mountConfig.Router,
		deps:            mountConfig.Deps,
		store:           mountConfig.Store,
		siteSettings:    mountConfig.SiteSettings,
		cacheMiddleware: mountConfig.CacheMiddleware,
		authGuardConfig: mountConfig.AuthGuardConfig,
		e2eModeEnabled:  e2eModeEnabled,
	}

	pageCount, partialCount := registerRoutesFromStore(ctx, regDeps, mountConfig.Store)

	span.SetAttributes(
		attribute.Int("pageCount", pageCount),
		attribute.Int("partialCount", partialCount),
	)
	l.Internal("Routes registered from manifest",
		logger_domain.Int("pageCount", pageCount),
		logger_domain.Int("partialCount", partialCount),
		logger_domain.Int(logFieldActionCount, len(mountConfig.Actions)),
	)

	mountActionRoutes(mountConfig)

	mountConfig.Router.NotFound(func(w http.ResponseWriter, request *http.Request) {
		if renderErrorPage(request.Context(), w, request, pageErrorContext{
			Deps:          mountConfig.Deps,
			Store:         mountConfig.Store,
			WebsiteConfig: mountConfig.SiteSettings,
		}, errorPageRequest{
			StatusCode:   http.StatusNotFound,
			OriginalPath: request.URL.Path,
		}) {
			return
		}
		if writeDevErrorFallback(request.Context(), w, http.StatusNotFound, "no route matches "+request.URL.Path) {
			return
		}
		http.NotFound(w, request)
	})

	span.SetStatus(codes.Ok, "Routes successfully mounted")
}

// registerPageRoute registers routes for a page entry with all its locale
// variants.
//
// For each locale the route pattern is translated for chi (see
// translateCatchAllForChi), wrapped in middleware, and registered for both
// GET and POST.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes regDeps (*routeRegistrationDeps) which provides the router and
// middleware dependencies.
// Takes entry (templater_domain.PageEntryView) which is the page to register.
func registerPageRoute(ctx context.Context, regDeps *routeRegistrationDeps, entry templater_domain.PageEntryView) {
	_, l := logger_domain.From(ctx, log)
	routePatterns := entry.GetRoutePatterns()
	if len(routePatterns) == 0 {
		l.Warn("Skipping page entry (no route patterns configured)",
			logger_domain.String(logFieldOriginalPath, entry.GetOriginalPath()))
		return
	}

	i18nStrategy := entry.GetI18nStrategy()
	for locale, routePattern := range routePatterns {
		registerPageRouteForLocale(ctx, regDeps, entry, locale, routePattern, i18nStrategy)
	}
}

// registerPageRouteForLocale registers a single locale variant of a page
// route.
//
// The chi pattern is derived from routePattern by translating any trailing
// named regex catch-all into chi's native wildcard form, with the original
// parameter name aliased back at request time.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes regDeps (*routeRegistrationDeps) which provides the router and
// middleware dependencies.
// Takes entry (templater_domain.PageEntryView) which is the page being
// registered.
// Takes locale (string) which is the locale variant being registered.
// Takes routePattern (string) which is the pre-translation route pattern.
// Takes i18nStrategy (string) which is the strategy name used for logging.
func registerPageRouteForLocale(
	ctx context.Context,
	regDeps *routeRegistrationDeps,
	entry templater_domain.PageEntryView,
	locale, routePattern, i18nStrategy string,
) {
	_, l := logger_domain.From(ctx, log)
	chiPattern, aliasedParamName := translateCatchAllForChi(routePattern)

	l.Internal("Registering page route",
		logger_domain.String(logFieldRoutePattern, routePattern),
		logger_domain.String(logFieldChiPattern, chiPattern),
		logger_domain.String(logFieldLocale, locale),
		logger_domain.String(logFieldI18nStrategy, i18nStrategy),
		logger_domain.String(logFieldOriginalPath, entry.GetOriginalPath()),
		logger_domain.Bool(logFieldIsE2EOnly, entry.GetIsE2EOnly()))

	baseHandler := buildPageRouteHandler(regDeps, entry, locale, routePattern, aliasedParamName)
	finalHandler := applyMiddlewares(baseHandler, entry, regDeps.cacheMiddleware, regDeps.authGuardConfig)
	if entry.GetIsE2EOnly() && !regDeps.e2eModeEnabled {
		finalHandler = e2eGuardMiddleware(finalHandler)
	}

	regDeps.router.Method(methodGET, chiPattern, finalHandler)
	regDeps.router.Method(methodPOST, chiPattern, finalHandler)
}

// buildPageRouteHandler builds the base http.Handler for a page route.
//
// It captures the locale and route pattern on the request context and, when
// the route had a named catch-all, copies chi's wildcard capture back under
// the original parameter name so generated code can read it via r.PathParam.
//
// Takes regDeps (*routeRegistrationDeps) which provides the dependencies the
// page handler needs.
// Takes entry (templater_domain.PageEntryView) which is the page being
// served.
// Takes locale (string) which is set on the request context.
// Takes routePattern (string) which is set as the matched pattern on the
// request context.
// Takes aliasedParamName (string) which is the named parameter that should
// receive the chi wildcard alias, or "" to skip aliasing.
//
// Returns http.Handler which is the assembled base handler.
func buildPageRouteHandler(
	regDeps *routeRegistrationDeps,
	entry templater_domain.PageEntryView,
	locale, routePattern, aliasedParamName string,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if aliasedParamName != "" {
			aliasCatchAllParam(request, aliasedParamName)
		}
		pctx := daemon_dto.PikoRequestCtxFromContext(request.Context())
		pctx.Locale = locale
		pctx.MatchedPattern = routePattern
		handlePageRequest(w, request, regDeps.deps, entry, regDeps.siteSettings, regDeps.store)
	})
}

// aliasCatchAllParam copies chi's "*" URL param onto a named alias.
//
// Lets generated code read the catch-all value under its original parameter
// name. When piko's mainRouter mounts the user router under its own "/*"
// route both routers capture catch-all values into the same URLParams stack.
// The parent's "*" (e.g. "docs/get-started/introduction") sits first and the
// child's "*" (e.g. "get-started/introduction") sits second. We iterate
// backwards so the alias reflects the most specific capture, which is the
// user-router's value.
//
// Takes request (*http.Request) whose chi route context is mutated in place.
// Takes aliasedParamName (string) which is the alias to receive the chi "*"
// value.
func aliasCatchAllParam(request *http.Request, aliasedParamName string) {
	rctx := chi.RouteContext(request.Context())
	if rctx == nil {
		return
	}
	for i := len(rctx.URLParams.Keys) - 1; i >= 0; i-- {
		if rctx.URLParams.Keys[i] == "*" {
			rctx.URLParams.Keys = append(rctx.URLParams.Keys, aliasedParamName)
			rctx.URLParams.Values = append(rctx.URLParams.Values, rctx.URLParams.Values[i])
			return
		}
	}
}

// translateCatchAllForChi rewrites a piko-style named regex catch-all pattern.
//
// Patterns such as `/docs/{slug:.+}`, `/docs/{slug:.*}`, or `/docs/{slug:.+?}`
// become chi's native wildcard form (`/docs/*`); the original named parameter
// is returned so the runtime can alias chi's `*` URL param back under that
// name.
//
// Any trailing `{name:<regex>}` segment is treated as a catch-all because
// chi's `{name:regex}` form is segment-bounded; if the user wrote a regex
// they almost certainly want multi-segment matching, otherwise the bare
// `{name}` form would have sufficed. Patterns without a trailing
// `{name:<regex>}` segment are returned unchanged with an empty alias.
//
// Takes pattern (string) which is the piko-style route pattern.
//
// Returns chiPattern (string) which is the chi-translated route pattern.
// Returns aliasedName (string) which is the original named parameter, or "".
func translateCatchAllForChi(pattern string) (chiPattern string, aliasedName string) {
	parsed := route_pattern.ParseTrailing(pattern)
	if !parsed.Found || !parsed.HasRegex || parsed.Regex == "" {
		return pattern, ""
	}
	return parsed.Prefix + "*", parsed.Name
}

// registerPartialRoute registers a route for a partial page entry.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes regDeps (*routeRegistrationDeps) which provides the router and
// middleware needed for route setup.
// Takes entry (templater_domain.PageEntryView) which specifies the partial
// page entry to register.
func registerPartialRoute(ctx context.Context, regDeps *routeRegistrationDeps, entry templater_domain.PageEntryView) {
	_, l := logger_domain.From(ctx, log)
	routePattern := entry.GetRoutePattern()
	if routePattern == "" {
		l.Warn("Skipping partial entry (empty route pattern)",
			logger_domain.String(logFieldOriginalPath, entry.GetOriginalPath()))
		return
	}

	l.Internal("Mounting route for partial",
		logger_domain.String(logFieldRoutePattern, routePattern),
		logger_domain.String(logFieldOriginalPath, entry.GetOriginalPath()),
		logger_domain.Bool("isE2EOnly", entry.GetIsE2EOnly()))

	var baseHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		handlePartialRequest(w, request, regDeps.deps, regDeps.store, entry, regDeps.siteSettings)
	})

	finalHandler := applyMiddlewares(baseHandler, entry, regDeps.cacheMiddleware, regDeps.authGuardConfig)

	if entry.GetIsE2EOnly() && !regDeps.e2eModeEnabled {
		finalHandler = e2eGuardMiddleware(finalHandler)
	}

	regDeps.router.Method(methodGET, routePattern, finalHandler)
	regDeps.router.Method(methodPOST, routePattern, finalHandler)
}

// applyMiddlewares wraps a handler with cache and page middlewares.
//
// Takes baseHandler (http.Handler) which is the handler to wrap.
// Takes entry (templater_domain.PageEntryView) which holds the page middleware
// settings.
// Takes cacheMiddleware (func(...)) which adds caching to the handler. Pass nil
// to skip caching.
//
// Returns http.Handler which is the handler with all middlewares applied.
func applyMiddlewares(
	baseHandler http.Handler,
	entry templater_domain.PageEntryView,
	cacheMiddleware func(next http.Handler) http.Handler,
	authGuardConfig *daemon_dto.AuthGuardConfig,
) http.Handler {
	finalHandler := baseHandler
	if cacheMiddleware != nil {
		finalHandler = cacheMiddleware(baseHandler)
	}
	if entry.GetHasAuthPolicy() && authGuardConfig != nil {
		finalHandler = authPolicyMiddleware(finalHandler, entry, authGuardConfig)
	}
	if entry.GetHasMiddleware() {
		middlewares := entry.GetMiddlewares()
		for i := len(middlewares) - 1; i >= 0; i-- {
			finalHandler = middlewares[i](finalHandler)
		}
	}
	return finalHandler
}

// authPolicyMiddleware enforces page-level AuthPolicy requirements.
// It checks whether the request's AuthContext satisfies the Required
// and Roles constraints declared by the page's AuthPolicy() function.
//
// Takes next (http.Handler) which is the downstream handler to invoke when
// authorisation succeeds.
// Takes entry (templater_domain.PageEntryView) which provides the page's
// declared auth policy and role requirements.
// Takes guardConfig (*daemon_dto.AuthGuardConfig) which supplies the
// unauthenticated handler and login redirect settings.
//
// Returns http.Handler which wraps the downstream handler with auth checks.
func authPolicyMiddleware(
	next http.Handler,
	entry templater_domain.PageEntryView,
	guardConfig *daemon_dto.AuthGuardConfig,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		policy := entry.GetAuthPolicy(nil)

		if !policy.Required && len(policy.Roles) == 0 {
			next.ServeHTTP(writer, request)
			return
		}

		auth := resolveAuthContext(request)

		if (policy.Required || len(policy.Roles) > 0) && (auth == nil || !auth.IsAuthenticated()) {
			handleUnauthenticated(writer, request, auth, guardConfig)
			return
		}

		if len(policy.Roles) > 0 && auth != nil {
			if !authHasRequiredRole(auth, policy.Roles) {
				http.Error(writer, "Forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(writer, request)
	})
}

// resolveAuthContext extracts the AuthContext from the request's
// PikoRequestCtx, returning nil when no authentication data is present.
//
// Takes request (*http.Request) which carries the PikoRequestCtx in its
// context.
//
// Returns daemon_dto.AuthContext which holds the caller's authentication
// state, or nil if no auth data is available.
func resolveAuthContext(request *http.Request) daemon_dto.AuthContext {
	pctx := daemon_dto.PikoRequestCtxFromContext(request.Context())
	if pctx == nil {
		return nil
	}
	auth, ok := pctx.CachedAuth.(daemon_dto.AuthContext)
	if !ok {
		return nil
	}
	return auth
}

// handleUnauthenticated responds to an unauthenticated request by
// delegating to the guard's custom handler or falling back to a login
// redirect.
//
// Takes writer (http.ResponseWriter) which receives the redirect or custom
// response.
// Takes request (*http.Request) which provides the original URI for the
// redirect parameter.
// Takes auth (daemon_dto.AuthContext) which is passed to the custom handler
// if one is configured.
// Takes guardConfig (*daemon_dto.AuthGuardConfig) which supplies the login
// path, redirect parameter name, and optional custom handler.
func handleUnauthenticated(
	writer http.ResponseWriter,
	request *http.Request,
	auth daemon_dto.AuthContext,
	guardConfig *daemon_dto.AuthGuardConfig,
) {
	if guardConfig.OnUnauthenticated != nil {
		guardConfig.OnUnauthenticated(writer, request, auth)
		return
	}
	loginPath := guardConfig.LoginPath
	if loginPath == "" {
		loginPath = "/login"
	}
	redirectParam := guardConfig.RedirectParam
	if redirectParam == "" {
		redirectParam = "redirect"
	}
	parsed, err := url.Parse(loginPath)
	if err != nil {
		parsed = &url.URL{Path: loginPath}
	}
	query := parsed.Query()
	query.Set(redirectParam, request.URL.RequestURI())
	parsed.RawQuery = query.Encode()
	http.Redirect(writer, request, parsed.String(), http.StatusSeeOther)
}

// authHasRequiredRole reports whether the given AuthContext holds at
// least one of the required roles.
//
// Takes auth (daemon_dto.AuthContext) which provides the user's roles via
// the "roles" key.
// Takes requiredRoles ([]string) which lists the roles to check against.
//
// Returns bool which is true when the user holds at least one required role.
func authHasRequiredRole(auth daemon_dto.AuthContext, requiredRoles []string) bool {
	userRoles, ok := auth.Get("roles").([]string)
	if !ok {
		return false
	}
	return hasAnyRole(userRoles, requiredRoles)
}

// hasAnyRole checks whether userRoles contains at least one of the
// required roles.
//
// Takes userRoles ([]string) which lists the roles the user holds.
// Takes requiredRoles ([]string) which lists the roles to match against.
//
// Returns bool which is true when at least one role appears in both lists.
func hasAnyRole(userRoles, requiredRoles []string) bool {
	for _, required := range requiredRoles {
		if slices.Contains(userRoles, required) {
			return true
		}
	}
	return false
}

// e2eGuardMiddleware wraps a handler to return 404 Not Found. This middleware
// is applied to E2E-only routes when E2E mode is disabled at runtime, providing
// defence in depth - even if E2E pages are compiled into the binary, they won't
// be accessible in production.
//
// Takes next (http.Handler) which is the handler to wrap.
//
// Returns http.Handler which always returns 404 Not Found.
func e2eGuardMiddleware(next http.Handler) http.Handler {
	_ = next
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	})
}

// registerRoutesFromStore iterates through store entries and registers page
// and partial routes.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes regDeps (*routeRegistrationDeps) which provides dependencies for route
// registration.
// Takes store (ManifestStoreView) which contains the manifest entries to
// process.
//
// Returns pageCount (int) which is the number of page routes registered.
// Returns partialCount (int) which is the number of partial routes registered.
func registerRoutesFromStore(
	ctx context.Context,
	regDeps *routeRegistrationDeps,
	store templater_domain.ManifestStoreView,
) (pageCount, partialCount int) {
	keys := store.GetKeys()

	for _, key := range keys {
		entry, ok := store.GetPageEntry(key)
		if !ok {
			continue
		}

		if !entry.GetIsPage() {
			if entry.GetRoutePattern() == "" {
				continue
			}
			registerPartialRoute(ctx, regDeps, entry)
			partialCount++
			continue
		}

		registerPageRoute(ctx, regDeps, entry)
		pageCount++
	}

	return pageCount, partialCount
}

// parseFragmentParam checks if the request is for a page fragment.
//
// Uses fast string matching to avoid creating a url.Values map when the
// parameter value is clear (e.g. "_f=true" or "_f=1"). Falls back to full
// query parsing for other values.
//
// Takes request (*http.Request) which provides the URL to check.
//
// Returns bool which is true when the _f query parameter indicates a fragment
// request.
func parseFragmentParam(request *http.Request) bool {
	rawQuery := request.URL.RawQuery
	if rawQuery == "" {
		return false
	}

	if strings.Contains(rawQuery, "_f=true") || strings.Contains(rawQuery, "_f=1") {
		return true
	}
	if strings.Contains(rawQuery, "_f=false") || strings.Contains(rawQuery, "_f=0") {
		return false
	}

	if fParam := request.URL.Query().Get("_f"); fParam != "" {
		if value, err := strconv.ParseBool(fParam); err == nil {
			return value
		}
	}
	return false
}

// getMatchedPattern returns the matched route pattern from the request
// context. It first checks for a custom pattern stored in the context, then
// falls back to Chi's route pattern if none is found.
//
// Takes request (*http.Request) which holds the context with route data.
//
// Returns string which is the matched route pattern, or empty if none is
// found.
func getMatchedPattern(request *http.Request) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(request.Context()); pctx != nil && pctx.MatchedPattern != "" {
		return pctx.MatchedPattern
	}
	if routeCtx := chi.RouteContext(request.Context()); routeCtx != nil {
		return routeCtx.RoutePattern()
	}
	return ""
}

// handlePageRenderError deals with errors from page rendering, treating
// redirect errors as a special case and attempting to render a custom
// error page when the response has not yet started.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes w (http.ResponseWriter) which receives the error response.
// Takes request (*http.Request) which holds the original client request.
// Takes err (error) which is the rendering error to handle.
// Takes pageCtx (pageErrorContext) which provides the page entry, span, and
// error page dependencies.
func handlePageRenderError(
	ctx context.Context, w http.ResponseWriter, request *http.Request,
	err error, pageCtx pageErrorContext, responseStarted bool,
) {
	ctx, l := logger_domain.From(ctx, log)

	if redirectErr, isRedirect := templater_dto.IsRedirect(err); isRedirect {
		handleRedirect(ctx, w, request, &redirectErr.Metadata, pageCtx.Span)
		return
	}

	switch {
	case errors.Is(err, context.Canceled):
		cause := context.Cause(ctx)
		if cause == nil {
			cause = err
		}
		l.Trace("Page rendering cancelled (client disconnected)",
			logger_domain.String(logFieldOriginalPath, pageCtx.Entry.GetOriginalPath()),
			logger_domain.String("cause", cause.Error()))
		return
	case errors.Is(err, context.DeadlineExceeded):
		cause := context.Cause(ctx)
		if cause == nil {
			cause = err
		}
		l.Warn("Page rendering timed out",
			logger_domain.String(logFieldOriginalPath, pageCtx.Entry.GetOriginalPath()),
			logger_domain.String("cause", cause.Error()))
		return
	}

	l.Error("Error during page stream rendering",
		logger_domain.String(logFieldError, err.Error()),
		logger_domain.String(logFieldOriginalPath, pageCtx.Entry.GetOriginalPath()))
	pageCtx.Span.RecordError(err)
	pageCtx.Span.SetStatus(codes.Error, "Error during page stream")
	requestErrorCount.Add(ctx, 1)

	if responseStarted {
		return
	}

	developmentMode := isDevelopmentModeFromContext(ctx)
	statusCode := extractErrorStatusCode(err)
	safeMessage := extractErrorMessage(err, developmentMode)
	errPageReq := errorPageRequest{
		StatusCode:      statusCode,
		Message:         safeMessage,
		InternalMessage: err.Error(),
		OriginalPath:    request.URL.Path,
	}
	if renderErrorPage(ctx, w, request, pageCtx, errPageReq) {
		return
	}
	if writeDevErrorFallback(ctx, w, statusCode, err.Error()) {
		return
	}
	http.Error(w, errMessageInternalServer, statusCode)
}

// handlePageRequest handles an HTTP request for a page route.
//
// Takes w (http.ResponseWriter) which receives the rendered page response.
// Takes request (*http.Request) which contains the incoming HTTP request.
// Takes deps (*daemon_domain.HTTPHandlerDependencies) which provides shared
// services for templating and other tasks.
// Takes entry (templater_domain.PageEntryView) which specifies which page to
// render.
// Takes websiteConfig (*config.WebsiteConfig) which provides site settings.
// Takes store (templater_domain.ManifestStoreView) which provides error page
// lookup for rendering custom error pages.
func handlePageRequest(
	w http.ResponseWriter, request *http.Request, deps *daemon_domain.HTTPHandlerDependencies,
	entry templater_domain.PageEntryView, websiteConfig *config.WebsiteConfig,
	store templater_domain.ManifestStoreView,
) {
	ctx, span := tracer.Start(request.Context(), "handlePageRequest")
	span.SetAttributes(
		attribute.String(logFieldPath, entry.GetOriginalPath()),
		attribute.String(logFieldMethod, request.Method),
		attribute.String("url.path", request.URL.Path),
		attribute.String("url.query", request.URL.RawQuery),
	)
	defer span.End()
	l := log.WithSpanContext(ctx)

	routePath := getMatchedPattern(request)
	pageRequestCount.Add(ctx, 1, cachedMetricOption(routePath, request.Method))
	l.Trace("Handling page request")

	pageDef := acquirePageDef(entry.GetOriginalPath(), routePath)
	defer releasePageDef(pageDef)

	var probeDurMs, renderDurMs int64
	defer func() {
		span.SetAttributes(
			attribute.Int64("probeDuration", probeDurMs),
			attribute.Int64("renderDuration", renderDurMs),
		)
	}()

	probeStartTime := time.Now()
	pageProbe, err := deps.Templater.ProbePage(ctx, *pageDef, request, websiteConfig)
	probeDurMs = time.Since(probeStartTime).Milliseconds()
	if err != nil {
		handlePageProbeError(ctx, w, request, err, pageErrorContext{
			Deps:          deps,
			Store:         store,
			WebsiteConfig: websiteConfig,
			Entry:         entry,
			Span:          span,
		})
		return
	}

	renderDurMs = renderAndRespond(ctx, w, request, renderRespondParams{
		Deps:      deps,
		PageDef:   pageDef,
		Entry:     entry,
		Store:     store,
		Config:    websiteConfig,
		PageProbe: pageProbe,
		Span:      span,
	})
}

// acquirePageDef gets a PageDefinition from the pool and populates it.
//
// Takes originalPath (string) which is the page's original path.
// Takes routePath (string) which is the normalised route pattern.
//
// Returns *templater_dto.PageDefinition which is the populated definition.
func acquirePageDef(originalPath, routePath string) *templater_dto.PageDefinition {
	pageDef, ok := pageDefPool.Get().(*templater_dto.PageDefinition)
	if !ok {
		pageDef = &templater_dto.PageDefinition{}
	}
	pageDef.OriginalPath = originalPath
	pageDef.NormalisedPath = routePath
	pageDef.TemplateHTML = ""
	return pageDef
}

// releasePageDef returns a PageDefinition to the pool after clearing it.
//
// Takes pageDef (*templater_dto.PageDefinition) which is the definition to
// release.
func releasePageDef(pageDef *templater_dto.PageDefinition) {
	*pageDef = templater_dto.PageDefinition{}
	pageDefPool.Put(pageDef)
}

// renderAndRespond renders the page and writes the HTTP response, returning
// the render duration in milliseconds.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes w (http.ResponseWriter) which receives the rendered output.
// Takes request (*http.Request) which provides the original request.
// Takes p (renderRespondParams) which groups the page definition, entry,
// store, config, probe result, and span.
//
// Returns int64 which is the render duration in milliseconds.
func renderAndRespond(
	ctx context.Context, w http.ResponseWriter, request *http.Request,
	p renderRespondParams,
) int64 {
	h := w.Header()
	h[headerContentType] = headerValContentTypeHTML
	h[headerXPPResponseSupport] = headerValFragmentPatch

	tracker := &responseTracker{ResponseWriter: w}
	renderStartTime := time.Now()
	err := p.Deps.Templater.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          *p.PageDef,
		Writer:        tracker,
		Response:      tracker,
		Request:       request,
		IsFragment:    parseFragmentParam(request),
		WebsiteConfig: p.Config,
		ProbeData:     p.PageProbe.ProbeData,
	})
	renderDurMs := time.Since(renderStartTime).Milliseconds()
	if err != nil {
		handlePageRenderError(ctx, w, request, err, pageErrorContext{
			Deps:          p.Deps,
			Store:         p.Store,
			WebsiteConfig: p.Config,
			Entry:         p.Entry,
			Span:          p.Span,
		}, tracker.started)
		return renderDurMs
	}

	if len(p.PageProbe.LinkHeaders) > 0 && request.ProtoMajor >= 2 && sendSpecificEarlyHints(w, request, p.PageProbe.LinkHeaders) {
		p.Span.SetStatus(codes.Ok, "Page rendered successfully")
		return renderDurMs
	}
	w.WriteHeader(http.StatusOK)
	p.Span.SetStatus(codes.Ok, "Page rendered successfully")
	return renderDurMs
}

// handlePageProbeError handles errors that occur when checking a page.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes w (http.ResponseWriter) which receives the error response.
// Takes request (*http.Request) which holds the original client request.
// Takes err (error) which is the probe error to handle.
func handlePageProbeError(
	ctx context.Context, w http.ResponseWriter, request *http.Request,
	err error, pageCtx pageErrorContext,
) {
	ctx, l := logger_domain.From(ctx, log)

	l.Error("Error probing page", logger_domain.String(logFieldError, err.Error()),
		logger_domain.String(logFieldOriginalPath, pageCtx.Entry.GetOriginalPath()))
	pageCtx.Span.RecordError(err)
	pageCtx.Span.SetStatus(codes.Error, "Error probing page")
	requestErrorCount.Add(ctx, 1)

	developmentMode := isDevelopmentModeFromContext(ctx)
	statusCode := extractErrorStatusCode(err)
	safeMessage := extractErrorMessage(err, developmentMode)
	errPageReq := errorPageRequest{
		StatusCode:      statusCode,
		Message:         safeMessage,
		InternalMessage: err.Error(),
		OriginalPath:    request.URL.Path,
	}
	if renderErrorPage(ctx, w, request, pageCtx, errPageReq) {
		return
	}
	if writeDevErrorFallback(ctx, w, statusCode, err.Error()) {
		return
	}
	http.Error(w, errMessageInternalServer, statusCode)
}

// validateRedirectStatusCode checks that the status code is a valid HTTP
// redirect code and returns 302 if it is not.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes statusCode (int) which is the HTTP status code to check.
//
// Returns int which is the status code if valid, or 302 if not.
func validateRedirectStatusCode(ctx context.Context, statusCode int) int {
	ctx, l := logger_domain.From(ctx, log)

	validCodes := map[int]bool{
		http.StatusMovedPermanently:  true,
		http.StatusFound:             true,
		http.StatusSeeOther:          true,
		http.StatusTemporaryRedirect: true,
	}
	if !validCodes[statusCode] {
		l.Warn("Invalid redirect status code, using 302", logger_domain.Int("providedStatus", statusCode))
		return http.StatusFound
	}
	return statusCode
}

// handleRedirect processes redirect metadata and sends an appropriate HTTP
// redirect response.
//
// Takes ctx (context.Context) which carries the logger context.
// Takes w (http.ResponseWriter) which receives the redirect response.
// Takes meta (*templater_dto.InternalMetadata) which contains the
// redirect target URL and status code.
// Takes span (trace.Span) which records tracing data for the
// redirect.
func handleRedirect(
	ctx context.Context, w http.ResponseWriter, _ *http.Request,
	meta *templater_dto.InternalMetadata, span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	if meta.ServerRedirect != "" {
		l.Error("ServerRedirect present in final metadata - this indicates a redirect loop was detected")
		http.Error(w, errMessageInternalServer, http.StatusInternalServerError)
		span.SetStatus(codes.Error, "Server redirect loop detected")
		return
	}

	var targetURL, redirectType string
	var statusCode int

	switch {
	case meta.ClientRedirect != "":
		targetURL, redirectType = meta.ClientRedirect, "ClientRedirect"
		statusCode = meta.RedirectStatus
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	default:
		l.Error("RedirectRequired error without valid redirect target")
		http.Error(w, errMessageInternalServer, http.StatusInternalServerError)
		span.SetStatus(codes.Error, "Invalid redirect metadata")
		return
	}

	statusCode = validateRedirectStatusCode(ctx, statusCode)
	l.Trace("Issuing HTTP redirect", logger_domain.String("type", redirectType),
		logger_domain.String("targetURL", targetURL), logger_domain.Int("statusCode", statusCode))

	w.Header().Del("Content-Type")
	w.Header().Del("X-PP-Response-Support")
	w.Header()["Location"] = []string{targetURL}
	w.WriteHeader(statusCode)

	span.SetAttributes(attribute.String("redirect.type", redirectType),
		attribute.String("redirect.target", targetURL), attribute.Int("redirect.status", statusCode))
	span.SetStatus(codes.Ok, "Redirect issued successfully")
}

// handlePartialRequest handles an HTTP request for a partial route by
// probing and rendering the partial template with tracing and metrics.
//
// Takes w (http.ResponseWriter) which receives the rendered response.
// Takes request (*http.Request) which is the incoming HTTP request.
// Takes deps (*daemon_domain.HTTPHandlerDependencies) which provides
// shared handler dependencies.
// Takes entry (templater_domain.PageEntryView) which describes the
// partial route being served.
// Takes websiteConfig (*config.WebsiteConfig) which provides website
// configuration.
func handlePartialRequest(
	w http.ResponseWriter, request *http.Request, deps *daemon_domain.HTTPHandlerDependencies,
	_ templater_domain.ManifestStoreView, entry templater_domain.PageEntryView, websiteConfig *config.WebsiteConfig,
) {
	ctx, span := tracer.Start(request.Context(), "handlePartialRequest")
	span.SetAttributes(
		attribute.String(logFieldPath, entry.GetOriginalPath()),
		attribute.String(logFieldMethod, request.Method),
		attribute.String("url.path", request.URL.Path),
		attribute.String("url.query", request.URL.RawQuery),
	)
	defer span.End()
	l := log.WithSpanContext(ctx)

	routePath := entry.GetRoutePattern()
	partialRequestCount.Add(ctx, 1, cachedMetricOption(routePath, request.Method))
	l.Trace("Handling partial request")

	partialDef := acquirePartialDef(entry.GetOriginalPath(), routePath)
	defer releasePartialDef(partialDef)

	var probeDurMs, renderDurMs int64
	defer func() {
		span.SetAttributes(
			attribute.Int64("probeDuration", probeDurMs),
			attribute.Int64("renderDuration", renderDurMs),
		)
	}()

	probeStartTime := time.Now()
	partialProbe, err := deps.Templater.ProbePartial(ctx, *partialDef, request, websiteConfig)
	probeDurMs = time.Since(probeStartTime).Milliseconds()
	if err != nil {
		handlePartialProbeError(ctx, w, err, entry, span)
		return
	}

	writePartialResponseHeaders(w, request, partialProbe.LinkHeaders)

	renderStartTime := time.Now()
	err = deps.Templater.RenderPartial(ctx, templater_domain.RenderRequest{
		Page:          *partialDef,
		Writer:        w,
		Response:      w,
		Request:       request,
		IsFragment:    parseFragmentParam(request),
		WebsiteConfig: websiteConfig,
		ProbeData:     partialProbe.ProbeData,
	})
	renderDurMs = time.Since(renderStartTime).Milliseconds()
	if err != nil {
		handlePartialRenderError(ctx, err, entry, span)
		return
	}

	span.SetStatus(codes.Ok, "Partial rendered successfully")
}

// acquirePartialDef gets a PageDefinition from the pool and initialises it for
// the given partial route.
//
// Takes originalPath (string) which is the raw request path.
// Takes normalisedPath (string) which is the cleaned path for template lookup.
//
// Returns *templater_dto.PageDefinition which is the initialised definition
// ready for partial rendering.
func acquirePartialDef(originalPath, normalisedPath string) *templater_dto.PageDefinition {
	partialDef, ok := pageDefPool.Get().(*templater_dto.PageDefinition)
	if !ok {
		partialDef = &templater_dto.PageDefinition{}
	}
	partialDef.OriginalPath = originalPath
	partialDef.NormalisedPath = normalisedPath
	partialDef.TemplateHTML = ""
	return partialDef
}

// releasePartialDef resets the PageDefinition and returns it to the pool.
//
// Takes partialDef (*templater_dto.PageDefinition) which is the definition to
// reset and return.
func releasePartialDef(partialDef *templater_dto.PageDefinition) {
	*partialDef = templater_dto.PageDefinition{}
	pageDefPool.Put(partialDef)
}

// writePartialResponseHeaders sets the content-type and response-support
// headers, then either sends early hints or writes a 200 status immediately.
//
// Takes w (http.ResponseWriter) which receives the response headers.
// Takes request (*http.Request) which provides the protocol version for early
// hints support detection.
// Takes linkHeaders ([]render_dto.LinkHeader) which contains the link headers
// to send as early hints when HTTP/2+ is available.
func writePartialResponseHeaders(w http.ResponseWriter, request *http.Request, linkHeaders []render_dto.LinkHeader) {
	h := w.Header()
	h[headerContentType] = headerValContentTypeHTML
	h[headerXPPResponseSupport] = headerValFragmentPatch
	if len(linkHeaders) == 0 || request.ProtoMajor < 2 || !sendSpecificEarlyHints(w, request, linkHeaders) {
		w.WriteHeader(http.StatusOK)
	}
}

// handlePartialProbeError handles errors that occur when probing a partial.
// It logs the error, records it in the trace span, and sends an internal
// server error response to the client.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes w (http.ResponseWriter) which receives the error response.
// Takes err (error) which is the error to handle.
// Takes entry (templater_domain.PageEntryView) which provides the path for
// logging.
// Takes span (trace.Span) which records the error for tracing.
func handlePartialProbeError(
	ctx context.Context, w http.ResponseWriter, err error,
	entry templater_domain.PageEntryView, span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	l.Error("Error probing partial", logger_domain.String(logFieldError, err.Error()),
		logger_domain.String(logFieldOriginalPath, entry.GetOriginalPath()))
	span.RecordError(err)
	span.SetStatus(codes.Error, "Error probing partial")
	requestErrorCount.Add(ctx, 1)
	http.Error(w, errMessageInternalServer, http.StatusInternalServerError)
}

// handlePartialRenderError logs and records an error from partial rendering.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes err (error) which is the error that occurred during rendering.
// Takes entry (templater_domain.PageEntryView) which provides page context for
// logging.
// Takes span (trace.Span) which records the error for tracing.
func handlePartialRenderError(
	ctx context.Context, err error, entry templater_domain.PageEntryView, span trace.Span,
) {
	ctx, l := logger_domain.From(ctx, log)

	l.Error("Error during partial stream rendering", logger_domain.String(logFieldError, err.Error()),
		logger_domain.String(logFieldOriginalPath, entry.GetOriginalPath()))
	span.RecordError(err)
	span.SetStatus(codes.Error, "Error during partial stream")
	requestErrorCount.Add(ctx, 1)
}

// mountActionRoutes mounts actions onto the router using ActionHandler.
//
// Takes cfg (*MountRoutesConfig) which provides the router, services, and
// settings needed to register action routes.
func mountActionRoutes(cfg *MountRoutesConfig) {
	_, span, l := log.Span(context.Background(), "mountActionRoutes",
		logger_domain.Int(logFieldActionCount, len(cfg.Actions)),
	)
	defer span.End()

	if len(cfg.Actions) == 0 {
		l.Internal("No actions to register")
		span.SetStatus(codes.Ok, "No actions to register")
		return
	}

	l.Internal("Starting action registration", logger_domain.Int(logFieldActionCount, len(cfg.Actions)))

	maxBodyBytes := int64(10 * 1024 * 1024)
	if cfg.RouteSettings.ActionMaxBodyBytes > 0 {
		maxBodyBytes = cfg.RouteSettings.ActionMaxBodyBytes
	}

	handler := NewActionHandler(cfg.CSRFService, maxBodyBytes, cfg.RateLimitService, cfg.RateLimitConfig, cfg.RouteSettings.CSRFSecFetchSiteEnforcement, cfg.ActionResponseCache, cfg.CaptchaService)
	if cfg.RouteSettings.DefaultMaxSSEDurationSeconds > 0 {
		handler.defaultMaxSSEDuration = time.Duration(cfg.RouteSettings.DefaultMaxSSEDurationSeconds) * time.Second
	}
	if cfg.RouteSettings.MaxMultipartFormBytes > 0 {
		handler.maxMultipartFormBytes = cfg.RouteSettings.MaxMultipartFormBytes
	}
	handler.RegisterAll(cfg.Actions)

	basePath := "/_piko/actions"
	if cfg.RouteSettings.ActionServePath != "" {
		basePath = cfg.RouteSettings.ActionServePath
	}

	handler.Mount(cfg.Router, basePath)

	if challengeProvider, ok := cfg.CaptchaService.(interface {
		ChallengeHandler() http.Handler
	}); ok {
		if challengeHandler := challengeProvider.ChallengeHandler(); challengeHandler != nil {
			cfg.Router.Get("/_piko/captcha/challenge", challengeHandler.ServeHTTP)
			l.Internal("HMAC captcha challenge endpoint mounted at /_piko/captcha/challenge")
		}
	}

	span.SetStatus(codes.Ok, "Actions registered successfully")
	l.Internal("Action registration complete", logger_domain.Int(logFieldActionCount, len(cfg.Actions)))
}

// extractOTelContext extracts OpenTelemetry context from request headers.
//
// Takes request (*http.Request) which contains the headers with trace context.
//
// Returns context.Context which includes any trace and span data from the
// request.
func extractOTelContext(request *http.Request) context.Context {
	ctx := request.Context()
	carrier := propagation.MapCarrier{}
	for k, v := range request.Header {
		if len(v) > 0 {
			carrier.Set(k, v[0])
		}
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// buildHeaders builds an HTTP Link header value from link header data.
//
// Takes lh (render_dto.LinkHeader) which contains the link URL, relation type,
// and optional attributes such as resource type and cross-origin settings.
//
// Returns string which is the formatted header value with parts joined by
// semicolons.
func buildHeaders(lh render_dto.LinkHeader) string {
	parts := make([]string, 0, buildHeadersMaxParts)
	parts = append(parts,
		fmt.Sprintf("<%s>", lh.URL),
		fmt.Sprintf("rel=%s", lh.Rel),
	)

	if lh.As != "" {
		parts = append(parts, fmt.Sprintf("as=%s", lh.As))
	}
	if lh.Type != "" {
		parts = append(parts, fmt.Sprintf("type=%q", lh.Type))
	}
	if lh.CrossOrigin == "anonymous" || lh.CrossOrigin == "use-credentials" {
		parts = append(parts, fmt.Sprintf("crossorigin=%s", lh.CrossOrigin))
	} else if lh.CrossOrigin != "" {
		parts = append(parts, "crossorigin")
	}
	return strings.Join(parts, "; ")
}

// sendSpecificEarlyHints sends HTTP 103 Early Hints with the specified link
// headers to enable browser preloading.
//
// Takes w (http.ResponseWriter) which receives the early hints response.
// Takes headersToSend ([]render_dto.LinkHeader) which specifies the link
// headers to send as early hints.
//
// Returns bool which indicates whether the early hints were sent and flushed
// successfully.
func sendSpecificEarlyHints(w http.ResponseWriter, r *http.Request, headersToSend []render_dto.LinkHeader) bool {
	if len(headersToSend) == 0 {
		return false
	}

	ctx := r.Context()
	_, l := logger_domain.From(ctx, log)

	for _, lh := range headersToSend {
		w.Header().Add("Link", buildHeaders(lh))
	}

	l.Trace("Attempting to send 103 Early Hints", logger_domain.Int("count", len(headersToSend)))
	w.WriteHeader(http.StatusEarlyHints)

	flusher, ok := w.(http.Flusher)
	if !ok {
		l.Warn("ResponseWriter does not support flushing, 103 Early Hints might not be sent effectively.")
		return false
	}
	flusher.Flush()
	l.Trace("Flushed 103 Early Hints")
	return true
}

// isDevelopmentModeFromContext reads the DevelopmentMode flag from the
// PikoRequestCtx on the context.
//
// Takes ctx (context.Context) which carries the request context.
//
// Returns bool which is true when the context carries a PikoRequestCtx
// with DevelopmentMode enabled, or false if no carrier is present.
func isDevelopmentModeFromContext(ctx context.Context) bool {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.DevelopmentMode
	}
	return false
}

// extractErrorStatusCode returns the HTTP status code from an error if it
// implements the ActionError interface. Returns 500 for plain errors.
//
// Takes err (error) which is the error to inspect.
//
// Returns int which is the HTTP status code extracted from the error.
func extractErrorStatusCode(err error) int {
	if actionErr, ok := errors.AsType[daemon_dto.ActionError](err); ok {
		return actionErr.StatusCode()
	}
	return http.StatusInternalServerError
}

// extractErrorMessage returns a user-safe error message.
//
// Takes err (error) which is the error to inspect.
// Takes developmentMode (bool) which controls whether internal details are
// returned.
//
// Returns string which is the user-facing error message.
func extractErrorMessage(err error, developmentMode bool) string {
	return safeerror.ExtractSafeMessage(err, developmentMode)
}

// writeDevErrorFallback writes a plain HTML error page in development mode
// when no custom error page is available. In production mode it returns false
// and the caller falls through to the standard http.Error or http.NotFound
// response.
//
// Takes ctx (context.Context) which carries the development mode flag.
// Takes w (http.ResponseWriter) which receives the HTML response.
// Takes statusCode (int) which is the HTTP status code.
// Takes message (string) which is the error detail to display.
//
// Returns bool which is true if the response was written (dev mode), false
// if the caller should use its own fallback (prod mode).
func writeDevErrorFallback(ctx context.Context, w http.ResponseWriter, statusCode int, message string) bool {
	if !isDevelopmentModeFromContext(ctx) {
		return false
	}

	escapedMessage := html.EscapeString(message)
	statusText := html.EscapeString(http.StatusText(statusCode))

	w.Header().Set(headerContentType, contentTypeHTML)
	w.WriteHeader(statusCode)
	fmt.Fprintf(w,
		`<!doctype html><html><head><title>%d %s</title>`+
			`<style>*{margin:0;padding:0;box-sizing:border-box}`+
			`body{font-family:system-ui,sans-serif;padding:2rem;color:#1a1a1a}`+
			`h1{font-size:1.5rem;margin-bottom:1rem}`+
			`pre{background:#f5f5f5;padding:1rem;border-radius:4px;`+
			`white-space:pre-wrap;word-break:break-word;font-size:0.875rem;`+
			`line-height:1.5;overflow-x:auto}</style></head>`+
			`<body><h1>%d %s</h1><pre>%s</pre></body></html>`,
		statusCode, statusText, statusCode, statusText, escapedMessage)

	return true
}

// renderErrorPage attempts to render a custom error page for the given status
// code and request path. It looks up the error page in the manifest store and
// renders it with the correct status code and error context.
//
// Takes ctx (context.Context) which carries the logger and trace context.
// Takes w (http.ResponseWriter) which receives the rendered error page.
// Takes request (*http.Request) which holds the original client request.
// Takes pageCtx (pageErrorContext) which provides the manifest store, templater,
// and trace span.
//
// Returns true if an error page was rendered, false if no error page exists or
// rendering failed.
func renderErrorPage(
	ctx context.Context, w http.ResponseWriter, request *http.Request,
	pageCtx pageErrorContext, errPageReq errorPageRequest,
) bool {
	ctx, l := logger_domain.From(ctx, log)
	errEntry, ok := pageCtx.Store.FindErrorPage(errPageReq.StatusCode, errPageReq.OriginalPath)
	if !ok {
		return false
	}

	epc := daemon_dto.ErrorPageContext{
		StatusCode:   errPageReq.StatusCode,
		Message:      errPageReq.Message,
		OriginalPath: errPageReq.OriginalPath,
	}
	if isDevelopmentModeFromContext(ctx) {
		epc.InternalMessage = errPageReq.InternalMessage
	}
	errCtx := daemon_dto.WithErrorPageContext(ctx, epc)
	errReq := request.WithContext(errCtx)

	errPageDef := templater_dto.PageDefinition{
		OriginalPath:   errEntry.GetOriginalPath(),
		NormalisedPath: errEntry.GetOriginalPath(),
	}

	probe, err := pageCtx.Deps.Templater.ProbePage(errCtx, errPageDef, errReq, pageCtx.WebsiteConfig)
	if err != nil {
		l.Warn("Failed to probe error page",
			logger_domain.String("errorPage", errEntry.GetOriginalPath()),
			logger_domain.Error(err))
		return false
	}

	w.Header()[headerContentType] = headerValContentTypeHTML
	w.WriteHeader(errPageReq.StatusCode)

	err = pageCtx.Deps.Templater.RenderPage(errCtx, templater_domain.RenderRequest{
		Page:          errPageDef,
		Writer:        w,
		Response:      w,
		Request:       errReq,
		IsFragment:    false,
		WebsiteConfig: pageCtx.WebsiteConfig,
		ProbeData:     probe.ProbeData,
	})
	if err != nil {
		l.Warn("Failed to render error page",
			logger_domain.String("errorPage", errEntry.GetOriginalPath()),
			logger_domain.Error(err))
		return false
	}

	return true
}
