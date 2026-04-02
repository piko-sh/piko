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

package daemon_dto

import (
	"context"
	"net/http"
)

// AuthContext represents the resolved authentication state for a
// request. Users implement this interface to expose their auth
// system's data to Piko pages and actions.
//
// The Get method is the extension point for app-specific data such as
// account objects, session details, TOTP state, or role lists. Piko
// does not interpret the returned values.
type AuthContext interface {
	// IsAuthenticated reports whether the request has a valid
	// authentication session.
	IsAuthenticated() bool

	// UserID returns the unique identifier of the authenticated user,
	// or empty string if not authenticated.
	UserID() string

	// Get returns an arbitrary auth value by key. Returns nil if the
	// key is not recognised.
	Get(key string) any
}

// AuthProvider resolves authentication state from an HTTP request.
// Piko calls Authenticate on every request when a provider is
// registered via WithAuthProvider.
//
// Return (nil, nil) for unauthenticated requests; this is not an
// error. Return a non-nil error only for unexpected failures (e.g.
// database unreachable); the middleware will log the error and treat
// the request as unauthenticated.
type AuthProvider interface {
	// Authenticate resolves the authentication state from an HTTP request.
	Authenticate(ctx context.Context, r *http.Request) (AuthContext, error)
}

// AuthPolicy declares authentication requirements for a page.
// Returned by the optional AuthPolicy() function in PK files,
// following the same pattern as CachePolicy() and Middlewares().
//
// Example usage in a PK file:
//
//	func AuthPolicy() piko.AuthPolicy {
//	    return piko.AuthPolicy{Required: true, Roles: []string{"admin"}}
//	}
type AuthPolicy struct {
	// Roles lists the roles required to access the page. If non-empty, the
	// user must have at least one of these roles (checked via AuthContext.Get("roles")).
	Roles []string

	// Required indicates the page requires an authenticated user.
	Required bool
}

// AuthGuardConfig controls prefix-level and page-level
// authentication enforcement. When registered via WithAuthGuard,
// Piko installs middleware that protects routes not listed in
// PublicPaths or PublicPrefixes.
type AuthGuardConfig struct {
	// OnUnauthenticated is called when a protected route is accessed
	// without valid authentication. If nil, a default redirect to
	// LoginPath is used.
	//
	// The AuthContext parameter may be non-nil but unauthenticated
	// (e.g. expired session, pending TOTP). This allows
	// app-specific redirect logic such as redirecting to a TOTP
	// verification page.
	OnUnauthenticated func(w http.ResponseWriter, r *http.Request, auth AuthContext)

	// LoginPath is where unauthenticated users are redirected.
	// Defaults to "/login" if empty.
	LoginPath string

	// RedirectParam is the query parameter name used to preserve
	// the original path across the login redirect. Defaults to
	// "redirect" if empty.
	RedirectParam string

	// PublicPaths are exact paths that do not require
	// authentication.
	PublicPaths []string

	// PublicPrefixes are path prefixes that do not require
	// authentication.
	PublicPrefixes []string
}
