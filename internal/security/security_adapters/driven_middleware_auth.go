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

package security_adapters

import (
	"net/http"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// AuthMiddleware calls the configured AuthProvider on every request
// and stores the resolved AuthContext on PikoRequestCtx.CachedAuth.
//
// It runs after RealIP (so ClientIP is available to the provider)
// and before rate limiting. On error, the middleware logs the failure
// and treats the request as unauthenticated - it never blocks a
// request.
type AuthMiddleware struct {
	// provider holds the auth provider that resolves authentication state.
	provider daemon_dto.AuthProvider

	// logger holds the logger for recording authentication errors.
	logger logger_domain.Logger
}

// NewAuthMiddleware creates an AuthMiddleware that delegates to the
// given provider.
//
// Takes provider (daemon_dto.AuthProvider) which resolves auth state.
// Takes logger (logger_domain.Logger) which receives error logs.
//
// Returns *AuthMiddleware which is ready for use as HTTP middleware.
func NewAuthMiddleware(provider daemon_dto.AuthProvider, logger logger_domain.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		provider: provider,
		logger:   logger,
	}
}

// Handler returns an http.Handler middleware that resolves
// authentication state and stores it on the per-request context
// carrier.
//
// Takes next (http.Handler) which is the next handler in the chain.
//
// Returns http.Handler which wraps next with auth resolution.
func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		pctx := daemon_dto.PikoRequestCtxFromContext(request.Context())
		if pctx == nil {
			next.ServeHTTP(writer, request)
			return
		}

		auth, err := m.provider.Authenticate(request.Context(), request)
		if err != nil {
			_, logger := logger_domain.From(request.Context(), m.logger)
			logger.Warn("Auth provider returned error",
				logger_domain.Error(err),
			)
			next.ServeHTTP(writer, request)
			return
		}

		pctx.CachedAuth = auth
		next.ServeHTTP(writer, request)
	})
}
