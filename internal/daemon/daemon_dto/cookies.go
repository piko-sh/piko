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
	"time"
)

// CookieOption is a function that sets options on an HTTP cookie.
type CookieOption func(*http.Cookie)

// SessionCookie creates a secure session cookie with sensible defaults.
//
// The cookie is configured with:
//   - HttpOnly: true (prevents JavaScript access, XSS protection)
//   - Secure: true (HTTPS only - override with SessionCookieInsecure for dev)
//   - SameSite: Lax (CSRF protection while allowing normal navigation)
//   - Path: "/" (valid for entire site)
//
// Takes name (string) which specifies the cookie name.
// Takes value (string) which specifies the cookie value.
// Takes expires (time.Time) which sets when the cookie expires.
//
// Returns *http.Cookie which is the configured session cookie.
func SessionCookie(name, value string, expires time.Time) *http.Cookie {
	maxAge := max(int(time.Until(expires).Seconds()), 0)

	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

// SessionCookieInsecure creates a session cookie that works over HTTP.
//
// Takes name (string) which specifies the cookie name.
// Takes value (string) which specifies the cookie value.
// Takes expires (time.Time) which specifies when the cookie expires.
//
// Returns *http.Cookie which is configured for insecure HTTP transmission.
//
// Use this only for local development where HTTPS is not available.
//
// WARNING: Do not use in production - cookies will be transmitted in plain
// text.
func SessionCookieInsecure(name, value string, expires time.Time) *http.Cookie {
	cookie := SessionCookie(name, value, expires)
	cookie.Secure = false
	return cookie
}

// ClearCookie creates a cookie that instructs the browser to delete an
// existing cookie.
//
// Takes name (string) which specifies the cookie name to clear.
//
// Returns *http.Cookie which can be added to a response to delete the cookie.
func ClearCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearCookieInsecure creates a cookie that tells the browser to delete a
// cookie over plain HTTP. Use this only for local development when HTTPS is
// not available.
//
// Takes name (string) which specifies the cookie to clear.
//
// Returns *http.Cookie which is ready to delete the named cookie without the
// Secure flag.
func ClearCookieInsecure(name string) *http.Cookie {
	cookie := ClearCookie(name)
	cookie.Secure = false
	return cookie
}

// Cookie creates a fully customisable cookie.
// Use this when you need more control than SessionCookie provides.
//
// Takes name (string) which specifies the cookie name.
// Takes value (string) which specifies the cookie value.
// Takes maxAge (time.Duration) which sets when the cookie expires.
// Takes opts (...CookieOption) which provides optional behaviour controls.
//
// Returns *http.Cookie which is configured with secure defaults.
func Cookie(name, value string, maxAge time.Duration, opts ...CookieOption) *http.Cookie {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		Expires:  time.Now().Add(maxAge),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// WithPath sets the URL path scope for a cookie.
//
// Takes path (string) which specifies the path where the cookie is valid.
//
// Returns CookieOption which applies the path setting to a cookie.
func WithPath(path string) CookieOption {
	return func(c *http.Cookie) {
		c.Path = path
	}
}

// WithDomain sets the domain for a cookie.
//
// Takes domain (string) which specifies the domain the cookie applies to.
//
// Returns CookieOption which sets the domain on a cookie.
func WithDomain(domain string) CookieOption {
	return func(c *http.Cookie) {
		c.Domain = domain
	}
}

// WithInsecure sets the cookie to allow sending over plain HTTP.
// WARNING: Only use for local development.
//
// Returns CookieOption which sets the cookie to use insecure transport.
func WithInsecure() CookieOption {
	return func(c *http.Cookie) {
		c.Secure = false
	}
}

// WithJavaScriptAccess allows JavaScript to access the cookie.
// WARNING: Only use if the cookie does not contain sensitive data.
//
// Returns CookieOption which disables the HttpOnly flag on the cookie.
func WithJavaScriptAccess() CookieOption {
	return func(c *http.Cookie) {
		c.HttpOnly = false
	}
}

// WithSameSiteStrict sets the cookie to SameSite=Strict mode.
// This gives the strongest protection against cross-site request forgery but
// may stop some valid actions from working, such as links from other sites.
//
// Returns CookieOption which applies the SameSite=Strict setting to a cookie.
func WithSameSiteStrict() CookieOption {
	return func(c *http.Cookie) {
		c.SameSite = http.SameSiteStrictMode
	}
}

// WithSameSiteNone sets the cookie to SameSite=None mode, permitting
// cross-site requests but requiring Secure=true.
//
// Returns CookieOption which configures a cookie for cross-site use.
func WithSameSiteNone() CookieOption {
	return func(c *http.Cookie) {
		c.SameSite = http.SameSiteNoneMode
		c.Secure = true
	}
}

// SmartSessionCookie creates a session cookie that automatically sets
// the Secure flag based on the DevelopmentMode flag in the request
// context. In production Secure is true; in development Secure is
// false.
//
// Takes ctx (context.Context) which provides the request context for
// mode detection.
// Takes name (string) which specifies the cookie name.
// Takes value (string) which specifies the cookie value.
// Takes expires (time.Time) which sets when the cookie expires.
//
// Returns *http.Cookie which is configured for the current environment.
func SmartSessionCookie(ctx context.Context, name, value string, expires time.Time) *http.Cookie {
	cookie := SessionCookie(name, value, expires)
	if pctx := PikoRequestCtxFromContext(ctx); pctx != nil && pctx.DevelopmentMode {
		cookie.Secure = false
	}
	return cookie
}

// SmartClearCookie creates a clear-cookie that automatically adapts
// the Secure flag to match the runtime environment.
//
// Takes ctx (context.Context) which provides the request context for
// mode detection.
// Takes name (string) which specifies the cookie name to clear.
//
// Returns *http.Cookie which instructs the browser to delete the cookie.
func SmartClearCookie(ctx context.Context, name string) *http.Cookie {
	cookie := ClearCookie(name)
	if pctx := PikoRequestCtxFromContext(ctx); pctx != nil && pctx.DevelopmentMode {
		cookie.Secure = false
	}
	return cookie
}
