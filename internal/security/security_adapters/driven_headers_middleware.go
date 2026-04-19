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
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/security/security_dto"
)

const (
	// tokenByteLength is the number of random bytes used for CSP tokens.
	tokenByteLength = 16

	// requestTokenPlaceholder is the marker text replaced with the actual CSP
	// token.
	requestTokenPlaceholder = "{{REQUEST_TOKEN}}"

	// maxStaticHeaders is the maximum number of static security headers that
	// can be configured. This allows the staticHeaders array to be embedded
	// directly in the struct with no separate backing array allocation.
	maxStaticHeaders = 9
)

// staticHeader is a pre-computed header entry with a reusable []string value.
// By assigning directly to the header map (h[key] = value) instead of calling
// Header.Set(), we avoid both the []string{v} allocation and the
// CanonicalMIMEHeaderKey canonicalisation on every request.
type staticHeader struct {
	// key is the canonical HTTP header name.
	key string

	// value is the pre-allocated header value slice.
	value []string
}

// SecurityHeadersMiddleware adds OWASP-recommended HTTP security headers to
// responses. It protects against common web vulnerabilities like clickjacking,
// XSS, and MIME sniffing.
//
// Static header values are pre-computed once at construction time as
// []string slices and assigned directly to the header map on each request,
// bypassing Header.Set() entirely. This eliminates ~10 allocations per
// request from the security headers alone.
type SecurityHeadersMiddleware struct {
	// tokenGenerator creates secure random per-request tokens for CSP.
	// Defaults to generateSecureToken; injectable for testing.
	tokenGenerator func() string

	// cspHeaderName is the pre-resolved CSP header key. Either
	// "Content-Security-Policy" or "Content-Security-Policy-Report-Only".
	cspHeaderName string

	// staticHeaders holds pre-computed header entries that are assigned
	// directly to the response header map on every request with zero
	// allocations. The fixed-size array is embedded inline in the struct,
	// avoiding a separate slice backing array allocation.
	staticHeaders [maxStaticHeaders]staticHeader

	// config holds the security header settings for the middleware.
	config SecurityHeadersValues

	// cspConfig holds the computed CSP configuration, kept separate from config
	// to maintain immutability of the base config.
	cspConfig security_dto.CSPRuntimeConfig

	// staticCSPValue is the pre-allocated CSP header value for policies that
	// don't use per-request tokens. Nil when tokens are enabled.
	staticCSPValue []string

	// staticHeaderCount tracks how many entries in staticHeaders are populated.
	staticHeaderCount int

	// forceHTTPS enables the HSTS header when set to true.
	forceHTTPS bool
}

// NewSecurityHeadersMiddleware creates a new security headers middleware
// instance.
//
// Takes config (SecurityHeadersValues) which specifies the security header
// settings to apply.
// Takes forceHTTPS (bool) which controls whether the
// Strict-Transport-Security header is included.
// Takes cspConfig (CSPRuntimeConfig) which provides the computed CSP settings.
// Takes reportingValues (ReportingValues) which provides the pre-built
// Reporting-Endpoints header value.
//
// Returns *SecurityHeadersMiddleware which is configured and ready for use.
func NewSecurityHeadersMiddleware(config SecurityHeadersValues, forceHTTPS bool, cspConfig security_dto.CSPRuntimeConfig, reportingValues ReportingValues) *SecurityHeadersMiddleware {
	m := &SecurityHeadersMiddleware{
		config:         config,
		cspConfig:      cspConfig,
		forceHTTPS:     forceHTTPS,
		tokenGenerator: generateSecureToken,
	}

	n := 0
	if config.XFrameOptions != "" {
		m.staticHeaders[n] = staticHeader{key: "X-Frame-Options", value: []string{config.XFrameOptions}}
		n++
	}
	if config.XContentTypeOptions != "" {
		m.staticHeaders[n] = staticHeader{key: "X-Content-Type-Options", value: []string{config.XContentTypeOptions}}
		n++
	}
	if config.ReferrerPolicy != "" {
		m.staticHeaders[n] = staticHeader{key: "Referrer-Policy", value: []string{config.ReferrerPolicy}}
		n++
	}
	if forceHTTPS && config.StrictTransportSecurity != "" {
		m.staticHeaders[n] = staticHeader{key: "Strict-Transport-Security", value: []string{config.StrictTransportSecurity}}
		n++
	}
	if config.CrossOriginOpenerPolicy != "" {
		m.staticHeaders[n] = staticHeader{key: "Cross-Origin-Opener-Policy", value: []string{config.CrossOriginOpenerPolicy}}
		n++
	}
	if config.CrossOriginResourcePolicy != "" {
		m.staticHeaders[n] = staticHeader{key: "Cross-Origin-Resource-Policy", value: []string{config.CrossOriginResourcePolicy}}
		n++
	}
	if config.PermissionsPolicy != "" {
		m.staticHeaders[n] = staticHeader{key: "Permissions-Policy", value: []string{config.PermissionsPolicy}}
		n++
	}
	if reportingValues.HeaderValue != "" {
		m.staticHeaders[n] = staticHeader{key: "Reporting-Endpoints", value: []string{reportingValues.HeaderValue}}
		n++
	}
	m.staticHeaders[n] = staticHeader{key: "X-Xss-Protection", value: []string{"0"}}
	n++
	m.staticHeaderCount = n

	if cspConfig.Policy != "" {
		m.cspHeaderName = "Content-Security-Policy"
		if cspConfig.ReportOnly {
			m.cspHeaderName = "Content-Security-Policy-Report-Only"
		}
		if !cspConfig.UsesRequestTokens {
			m.staticCSPValue = []string{cspConfig.Policy}
		}
	}

	return m
}

// Handler returns the middleware handler function for use with chi or other
// routers.
//
// Takes next (http.Handler) which is the next handler in the chain.
//
// Returns http.Handler which wraps the next handler and adds security headers.
func (m *SecurityHeadersMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.config.StripServerHeader {
			w.Header().Del("Server")
		}
		if m.config.StripPoweredByHeader {
			w.Header().Del("X-Powered-By")
		}

		if m.cspConfig.UsesRequestTokens {
			token := m.tokenGenerator()
			if pctx := daemon_dto.PikoRequestCtxFromContext(r.Context()); pctx != nil {
				pctx.CSPToken = token
			}
		}

		m.setSecurityHeaders(w, r)

		wrapped := &headerStripperWriter{
			ResponseWriter: w,
			stripServer:    m.config.StripServerHeader,
			stripPoweredBy: m.config.StripPoweredByHeader,
			wroteHeader:    false,
		}

		next.ServeHTTP(wrapped, r)
	})
}

// setSecurityHeaders assigns all pre-computed static headers to the response
// and sets the CSP header (which may be dynamic when request tokens are used).
//
// Static headers are assigned directly to the header map, bypassing
// Header.Set() and its per-call []string + CanonicalMIMEHeaderKey allocations.
//
// Takes w (http.ResponseWriter) which receives the security headers.
// Takes r (*http.Request) which provides request context for CSP handling.
func (m *SecurityHeadersMiddleware) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	for i := range m.staticHeaderCount {
		h[m.staticHeaders[i].key] = m.staticHeaders[i].value
	}
	m.setCSPHeader(h, r)
}

// setCSPHeader sets the Content-Security-Policy header, reusing a
// pre-allocated value for static policies and substituting per-request
// tokens for dynamic ones.
//
// Takes h (http.Header) which receives the CSP header.
// Takes r (*http.Request) which provides context for token replacement.
func (m *SecurityHeadersMiddleware) setCSPHeader(h http.Header, r *http.Request) {
	if m.cspHeaderName == "" {
		return
	}

	if m.staticCSPValue != nil {
		h[m.cspHeaderName] = m.staticCSPValue
		return
	}

	token := GetRequestTokenFromContext(r.Context())
	if token != "" {
		policy := strings.ReplaceAll(m.cspConfig.Policy, requestTokenPlaceholder, formatCSPToken(token))
		h[m.cspHeaderName] = []string{policy}
	}
}

// headerStripperWriter wraps http.ResponseWriter to remove sensitive headers
// before they reach the client. It implements io.Writer.
type headerStripperWriter struct {
	http.ResponseWriter

	// stripServer controls whether to remove the Server header from responses.
	stripServer bool

	// stripPoweredBy controls whether to remove the X-Powered-By header.
	stripPoweredBy bool

	// wroteHeader indicates whether the HTTP headers have been written.
	wroteHeader bool
}

// WriteHeader writes the HTTP status code after removing any headers that
// should be stripped. Later calls are ignored.
//
// Takes statusCode (int) which specifies the HTTP status code to write.
func (w *headerStripperWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	if w.stripServer {
		w.Header().Del("Server")
	}
	if w.stripPoweredBy {
		w.Header().Del("X-Powered-By")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write calls WriteHeader before writing the body.
//
// Takes b ([]byte) which contains the data to write to the response.
//
// Returns int which is the number of bytes written.
// Returns error when the underlying writer fails.
func (w *headerStripperWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher by forwarding to the underlying writer.
// This is required for Server-Sent Events and other streaming responses
// that need to flush buffered data to the client immediately.
func (w *headerStripperWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		if !w.wroteHeader {
			w.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter for middleware that needs it.
//
// Returns http.ResponseWriter which is the wrapped response writer.
func (w *headerStripperWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// GetRequestTokenFromContext retrieves the CSP request token from the
// PikoRequestCtx carrier in the context.
//
// Returns string which is the token, or an empty string if no token was set
// for this request.
func GetRequestTokenFromContext(ctx context.Context) string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil {
		return pctx.CSPToken
	}
	return ""
}

// generateSecureToken creates a random token for CSP using secure random bytes.
//
// Returns string which is a base64-encoded token.
//
// Panics if crypto/rand fails, since this indicates a catastrophic system
// problem (no entropy source) that would compromise all security guarantees.
func generateSecureToken() string {
	b := make([]byte, tokenByteLength)
	if _, err := rand.Read(b); err != nil {
		panic("security: crypto/rand failed: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// formatCSPToken formats a per-request token for use in a CSP header.
// The CSP specification requires the format 'nonce-<base64value>'.
//
// Takes token (string) which is the base64-encoded per-request token value.
//
// Returns string which is the formatted CSP token directive.
func formatCSPToken(token string) string {
	return "'nonce-" + token + "'"
}
