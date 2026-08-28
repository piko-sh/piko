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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/security/security_adapters"
	"piko.sh/piko/internal/security/security_domain"
)

// setupRealIP adds the RealIP middleware that extracts the client IP and creates a
// request ID, storing both in the request context. This must run before rate limiting and
// other middleware that need the client IP or request ID.
//
// Takes r (chi.Router) which receives the RealIP middleware.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides trusted proxy settings.
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

// setupRateLimiting adds rate limiting middleware to the router if enabled. The rate
// limiter reads the client IP from context, set by RealIP middleware.
//
// Takes r (chi.Router) which receives the rate limiting middleware.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides rate limit settings.
// Takes rateLimitService (security_domain.RateLimitService) which handles rate limit
// checks.
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

// setupAuthGuard installs the authentication guard, unless doing so would lock everyone
// out.
//
// Takes r (chi.Router) which receives the guard middleware.
// Takes deps (daemon_domain.RouterDependencies) which supplies the config and provider.
func (*HTTPRouterBuilder) setupAuthGuard(ctx context.Context, r chi.Router, deps daemon_domain.RouterDependencies) {
	if deps.AuthGuardConfig == nil {
		return
	}

	if deps.AuthProvider == nil {
		_, l := logger_domain.From(ctx, log)
		l.Error("An auth guard is configured but no auth provider is set, so every request " +
			"would resolve as unauthenticated and the whole site would redirect to a login " +
			"page that cannot be passed. The guard has not been installed; add " +
			"WithAuthProvider to enable it.")

		return
	}

	authGuard := security_adapters.NewAuthGuardMiddleware(*deps.AuthGuardConfig)
	r.Use(authGuard.Handler)
}

// setupCORS sets up CORS middleware to handle cross-origin requests.
//
// Takes r (chi.Router) which receives the CORS middleware.
// Takes routerConfig (*daemon_domain.RouterConfig) which provides the allowed origins.
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
