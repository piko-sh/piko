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
	"net/url"
	"strings"

	"piko.sh/piko/internal/daemon/daemon_dto"
)

const (
	// defaultLoginPath holds the default URL path for the login page.
	defaultLoginPath = "/login"

	// defaultRedirectParam holds the default query parameter name for the post-login
	// redirect target.
	defaultRedirectParam = "redirect"

	// captchaChallengePath is the endpoint a captcha challenge is fetched from. Proving you
	// are a person cannot itself require being signed in.
	captchaChallengePath = "/_piko/captcha/challenge"
)

// AuthGuardMiddleware enforces authentication on routes not listed in the public paths or
// prefixes. When a protected route is accessed without valid authentication, the
// middleware either calls a custom OnUnauthenticated handler or redirects to the login
// page.
type AuthGuardMiddleware struct {
	// publicPaths holds every path the guard never challenges, normalised so a trailing
	// slash does not change the answer. It always contains the login and captcha paths.
	publicPaths map[string]struct{}

	// config holds the authentication guard configuration.
	config daemon_dto.AuthGuardConfig
}

// NewAuthGuardMiddleware creates an AuthGuardMiddleware with the given configuration.
//
// Takes config (daemon_dto.AuthGuardConfig) which specifies public paths, login redirect,
// and optional custom handler.
//
// Returns *AuthGuardMiddleware which is ready for use as HTTP middleware.
func NewAuthGuardMiddleware(config daemon_dto.AuthGuardConfig) *AuthGuardMiddleware {
	loginPath := config.LoginPath
	if loginPath == "" {
		loginPath = defaultLoginPath
	}

	return &AuthGuardMiddleware{
		config:      config,
		publicPaths: buildGuardPublicPaths(config, loginPath),
	}
}

// buildGuardPublicPaths collects the paths the guard must never challenge.
//
// Takes config (daemon_dto.AuthGuardConfig) which supplies the application's public
// paths.
// Takes loginPath (string) which is the resolved login path, possibly carrying a query.
//
// Returns map[string]struct{} which holds every exempt path, normalised.
func buildGuardPublicPaths(config daemon_dto.AuthGuardConfig, loginPath string) map[string]struct{} {
	paths := make(map[string]struct{}, len(config.PublicPaths)+2)
	for _, path := range config.PublicPaths {
		paths[normalisePublicPath(path)] = struct{}{}
	}

	paths[normalisePublicPath(pathComponentOf(loginPath))] = struct{}{}
	paths[normalisePublicPath(captchaChallengePath)] = struct{}{}

	return paths
}

// pathComponentOf strips any query string from a configured path.
//
// A login path may legitimately carry a query, such as "/login?next=1", but the exemption
// is matched against a request path, which never does.
//
// Takes path (string) which is the configured path.
//
// Returns string which is the path without its query.
func pathComponentOf(path string) string {
	component, _, _ := strings.Cut(path, "?")

	return component
}

// normalisePublicPath puts a path in the one form the exemption set is keyed by, so that
// "/login" and "/login/" are the same entry.
//
// Takes path (string) which is the raw configured or requested path.
//
// Returns string which is the normalised form.
func normalisePublicPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "/" {
		return trimmed
	}

	return strings.TrimSuffix(trimmed, "/")
}

// Handler returns an http.Handler middleware that enforces authentication on non-public
// routes.
//
// Takes next (http.Handler) which is the next handler in the chain.
//
// Returns http.Handler which wraps next with auth enforcement.
func (m *AuthGuardMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if m.isPublicPath(request.URL.Path) {
			next.ServeHTTP(writer, request)
			return
		}

		var auth daemon_dto.AuthContext
		if pctx := daemon_dto.PikoRequestCtxFromContext(request.Context()); pctx != nil {
			if cachedAuth, ok := pctx.CachedAuth.(daemon_dto.AuthContext); ok {
				auth = cachedAuth
			}
		}

		if auth == nil || !auth.IsAuthenticated() {
			if m.config.OnUnauthenticated != nil {
				m.config.OnUnauthenticated(writer, request, auth)
				return
			}
			m.redirectToLogin(writer, request)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

// isPublicPath checks whether the given path is in the public allow list.
//
// Takes path (string) which is the URL path to check.
//
// Returns bool which is true when the path matches a public path or prefix.
func (m *AuthGuardMiddleware) isPublicPath(path string) bool {
	if _, public := m.publicPaths[normalisePublicPath(path)]; public {
		return true
	}
	for _, prefix := range m.config.PublicPrefixes {
		if strings.HasSuffix(prefix, "/") {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// redirectToLogin answers an unauthenticated request, preserving the original path so the
// caller returns to it after authenticating.
//
// Takes writer (http.ResponseWriter) which receives the response.
// Takes request (*http.Request) which provides the original path.
func (m *AuthGuardMiddleware) redirectToLogin(writer http.ResponseWriter, request *http.Request) {
	RespondUnauthenticated(writer, request, nil, m.config)
}

// buildLoginRedirect constructs a redirect URL that preserves the full request URI (path
// + query) and safely appends the redirect parameter even when loginPath already contains
// a query string.
//
// Takes loginPath (string) which is the base login page URL.
// Takes redirectParam (string) which is the query parameter name for the post-login
// redirect target.
// Takes request (*http.Request) which provides the original request URI.
//
// Returns string containing the fully constructed redirect URL.
func buildLoginRedirect(loginPath, redirectParam string, request *http.Request) string {
	parsed, err := url.Parse(loginPath)
	if err != nil {
		parsed = &url.URL{Path: loginPath}
	}
	query := parsed.Query()
	query.Set(redirectParam, request.URL.RequestURI())
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
