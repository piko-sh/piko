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

	"piko.sh/piko/internal/security/security_domain"
)

// SecureCookieWriter implements security_domain.SecureCookieWriter with
// configurable security defaults for HTTP cookies. It enforces security flags
// to protect against session hijacking and CSRF attacks.
type SecureCookieWriter struct {
	// config holds the security settings for cookies.
	config CookieSecurityValues

	// isHTTPS indicates whether the request uses HTTPS.
	isHTTPS bool
}

// SetCookie writes a cookie to the response with security flags applied.
//
// The original cookie is modified in place with the following rules:
// HttpOnly is forced if ForceHTTPOnly is enabled; Secure is forced if
// ForceSecureOnHTTPS is enabled and serving over HTTPS; SameSite defaults
// to DefaultSameSite if not already set.
//
// Takes rw (http.ResponseWriter) which receives the cookie header.
// Takes cookie (*http.Cookie) which is modified in place with security flags.
func (w *SecureCookieWriter) SetCookie(rw http.ResponseWriter, cookie *http.Cookie) {
	if w.config.ForceHTTPOnly {
		cookie.HttpOnly = true
	}

	if w.config.ForceSecureOnHTTPS && w.isHTTPS {
		cookie.Secure = true
	}

	if cookie.SameSite == 0 {
		cookie.SameSite = parseSameSite(w.config.DefaultSameSite)
	}

	http.SetCookie(rw, cookie)
}

// IsHTTPS returns whether the current context is served over HTTPS.
//
// Returns bool which is true when the connection uses HTTPS.
func (w *SecureCookieWriter) IsHTTPS() bool {
	return w.isHTTPS
}

// NewSecureCookieWriter creates a new secure cookie writer with the given
// configuration.
//
// Takes config (CookieSecurityValues) which specifies the cookie security
// settings.
// Takes isHTTPS (bool) which indicates whether the connection uses HTTPS.
//
// Returns security_domain.SecureCookieWriter which is the configured writer
// ready for use.
func NewSecureCookieWriter(config CookieSecurityValues, isHTTPS bool) security_domain.SecureCookieWriter {
	return &SecureCookieWriter{
		config:  config,
		isHTTPS: isHTTPS,
	}
}

// parseSameSite converts a SameSite mode name to its http.SameSite value.
//
// Takes s (string) which is the mode name: "Strict", "None", or "Lax".
//
// Returns http.SameSite which is the matching constant. Defaults to
// SameSiteLaxMode for unknown values.
func parseSameSite(s string) http.SameSite {
	switch s {
	case "Strict":
		return http.SameSiteStrictMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
