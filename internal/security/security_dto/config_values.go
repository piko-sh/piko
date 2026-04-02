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

package security_dto

// SecurityHeadersValues holds the resolved security header configuration.
// All fields are value types; pointer-to-value conversion is performed in the
// bootstrap layer.
type SecurityHeadersValues struct {
	// XFrameOptions controls the X-Frame-Options header.
	XFrameOptions string

	// XContentTypeOptions sets the X-Content-Type-Options header.
	XContentTypeOptions string

	// ReferrerPolicy sets the Referrer-Policy header.
	ReferrerPolicy string

	// ContentSecurityPolicy sets the CSP header value.
	ContentSecurityPolicy string

	// StrictTransportSecurity sets the HSTS header.
	StrictTransportSecurity string

	// CrossOriginOpenerPolicy sets the COOP header.
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy sets the CORP header.
	CrossOriginResourcePolicy string

	// PermissionsPolicy sets the Permissions-Policy header.
	PermissionsPolicy string

	// Enabled controls whether security headers are added to responses.
	Enabled bool

	// StripServerHeader removes the Server header from responses.
	StripServerHeader bool

	// StripPoweredByHeader removes the X-Powered-By header.
	StripPoweredByHeader bool
}

// CookieSecurityValues holds the resolved cookie security configuration.
// All fields are value types; pointer-to-value conversion is performed in the
// bootstrap layer.
type CookieSecurityValues struct {
	// DefaultSameSite is the default SameSite attribute for cookies.
	DefaultSameSite string

	// ForceHTTPOnly forces the HttpOnly flag on all cookies.
	ForceHTTPOnly bool

	// ForceSecureOnHTTPS forces the Secure flag on cookies when using HTTPS.
	ForceSecureOnHTTPS bool
}

// ReportingValues holds the resolved reporting endpoints configuration.
// All fields are value types; pointer-to-value conversion is performed in the
// bootstrap layer.
type ReportingValues struct {
	// HeaderValue is the pre-built Reporting-Endpoints header value.
	// Empty string means the header will not be set.
	HeaderValue string
}

// RateLimitValues holds the resolved rate limit configuration.
// All fields are value types; pointer-to-value conversion is performed in the
// bootstrap layer.
type RateLimitValues struct {
	// Storage specifies the backend for rate limit counters.
	Storage string

	// TrustedProxies is a list of CIDR ranges trusted to set X-Forwarded-For.
	TrustedProxies []string

	// ExemptPaths lists URL path prefixes that bypass rate limiting.
	ExemptPaths []string

	// Global sets the rate limit for all requests across the service.
	Global RateLimitTierValues

	// Actions sets rate limits for server action endpoints.
	Actions RateLimitTierValues

	// CloudflareEnabled enables trust of the CF-Connecting-IP header.
	CloudflareEnabled bool

	// Enabled controls whether rate limiting is active.
	Enabled bool

	// HeadersEnabled controls whether rate limit headers are included in responses.
	HeadersEnabled bool
}

// RateLimitTierValues configures rate limits for a tier (global or actions).
type RateLimitTierValues struct {
	// RequestsPerMinute is the maximum requests allowed per minute per client.
	RequestsPerMinute int

	// BurstSize is the maximum burst size for rate limiting.
	BurstSize int
}
