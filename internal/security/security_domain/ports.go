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

package security_domain

import (
	"bytes"
	"net/http"

	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/security/security_dto"
)

// CSRFTokenService provides CSRF token generation and validation.
// It uses the Double Submit Cookie pattern where tokens are bound to a
// browser session cookie rather than a timestamp-based expiry.
type CSRFTokenService interface {
	healthprobe_domain.Probe

	// GenerateCSRFPair creates a new CSRF token pair for the given request. The
	// ResponseWriter is required to set the CSRF cookie if not already present.
	//
	// Takes w (http.ResponseWriter) which is used to set the CSRF cookie.
	// Takes r (*http.Request) which provides the request context for token
	// generation.
	// Takes buffer (*bytes.Buffer) which is used to build the action token. The
	// returned CSRFPair.ActionToken is a slice into this buffer, so the buffer
	// must remain valid for the lifetime of the returned pair.
	//
	// Returns security_dto.CSRFPair which contains the generated token pair.
	// Returns error when token generation fails.
	GenerateCSRFPair(w http.ResponseWriter, r *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error)

	// ValidateCSRFPair checks whether the CSRF token pair is valid.
	// Validates that the cookie value embedded in the token matches the actual
	// cookie from the request.
	//
	// Takes r (*http.Request) which provides the request context and cookie.
	// Takes rawEphemeralTokenFromRequest (string) which is the ephemeral token
	// from the request.
	// Takes actionToken ([]byte) which is the action token from the request
	// header. Use mem.Bytes() for zero-copy conversion from string.
	//
	// Returns bool which is true if the token pair is valid.
	// Returns error when validation fails. Returns CSRFValidationError with
	// specific error codes for frontend error recovery.
	ValidateCSRFPair(r *http.Request, rawEphemeralTokenFromRequest string, actionToken []byte) (bool, error)
}

// CSRFCookieSourceAdapter provides the CSRF binding token value for the
// Double Submit Cookie pattern. This abstracts cookie management to allow
// framework users to integrate with their own session systems.
//
// The default implementation uses an HttpOnly cookie. Framework users can
// override this to use their authentication session token (for example, from
// SCS).
type CSRFCookieSourceAdapter interface {
	// GetOrCreateToken returns the CSRF binding token for the request.
	// If no token exists, it creates one and sets it on the response.
	//
	// Takes r (*http.Request) the HTTP request to check for existing token.
	// Takes w (http.ResponseWriter) the response writer for setting cookies.
	//
	// Returns string which is the token value.
	// Returns error when token generation fails.
	GetOrCreateToken(r *http.Request, w http.ResponseWriter) (string, error)

	// GetToken returns the existing CSRF binding token from the request cookie.
	// Used during token validation.
	//
	// Takes r (*http.Request) which is the HTTP request.
	//
	// Returns string which is the token value, or empty if not found.
	GetToken(r *http.Request) string

	// InvalidateToken removes the CSRF token cookie.
	// Call this when a user logs out or when the session ends.
	//
	// Takes w (http.ResponseWriter) which is the response to write the cookie to.
	InvalidateToken(w http.ResponseWriter)
}

// RequestContextBinderAdapter provides a way to bind CSRF tokens to a request.
type RequestContextBinderAdapter interface {
	// GetBindingIdentifier returns the unique identifier for the request binding.
	//
	// Takes r (*http.Request) which is the HTTP request to extract the identifier
	// from.
	//
	// Returns string which is the binding identifier.
	GetBindingIdentifier(r *http.Request) string
}

// SecureCookieWriter provides secure cookie setting with enforced security
// defaults. It applies appropriate security flags to cookies (HttpOnly, Secure,
// SameSite) based on configuration, helping protect against session hijacking
// and CSRF attacks.
type SecureCookieWriter interface {
	// SetCookie writes a cookie to the response with security flags applied.
	// The cookie's security attributes are modified based on configuration:
	// HttpOnly is forced if configured, Secure is forced if configured and
	// serving over HTTPS, and SameSite defaults to the configured value if
	// not set.
	//
	// Takes w (http.ResponseWriter) which is the response to write the cookie to.
	// Takes cookie (*http.Cookie) which is the cookie to set.
	SetCookie(w http.ResponseWriter, cookie *http.Cookie)

	// IsHTTPS returns whether the current request is served over HTTPS.
	// This is used to check if the Secure flag should be set on cookies.
	//
	// Returns bool which is true if the request uses HTTPS, false otherwise.
	IsHTTPS() bool
}

// ClientIPExtractor extracts the real client IP address from HTTP requests. It
// handles trusted proxy headers (X-Forwarded-For, X-Real-IP, CF-Connecting-IP)
// when the request originates from a trusted proxy CIDR range.
type ClientIPExtractor interface {
	// ExtractClientIP returns the real client IP address from the request,
	// trusting forwarding headers when the request originates from a trusted
	// proxy and using the direct remote address otherwise.
	//
	// Takes r (*http.Request) which is the HTTP request.
	//
	// Returns string which is the client IP address.
	ExtractClientIP(r *http.Request) string

	// IsTrustedProxy checks whether the given IP address is within a trusted
	// proxy CIDR range.
	//
	// Takes ip (string) which is the IP address to check.
	//
	// Returns bool which is true if the IP is from a trusted proxy.
	IsTrustedProxy(ip string) bool
}
