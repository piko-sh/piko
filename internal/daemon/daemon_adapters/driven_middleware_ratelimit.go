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
	"net/http"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// headerRateLimitLimit is the canonical MIME form of X-RateLimit-Limit.
	headerRateLimitLimit = "X-Ratelimit-Limit"

	// headerRateLimitRemaining is the canonical MIME form of X-RateLimit-Remaining.
	headerRateLimitRemaining = "X-Ratelimit-Remaining"

	// headerRateLimitReset is the canonical MIME form of X-RateLimit-Reset.
	headerRateLimitReset = "X-Ratelimit-Reset"

	// headerRetryAfter is the HTTP header name that tells clients how long to wait
	// before sending another request after being rate limited.
	headerRetryAfter = "Retry-After"
)

// rateLimitMiddleware limits HTTP requests based on client IP address.
// It reads the client IP from the request context, which should be set by
// the RealIP middleware earlier in the chain.
type rateLimitMiddleware struct {
	// clock provides time functions for rate limit window calculations.
	clock clock.Clock

	// service checks rate limits against the backing store.
	service security_domain.RateLimitService

	// config holds the rate limiting settings for global and action requests.
	config security_dto.RateLimitValues
}

// rateLimitMiddlewareOption sets options for a rate limit middleware.
type rateLimitMiddlewareOption func(*rateLimitMiddleware)

// Handler returns the middleware handler function for use with chi or other
// routers. The client IP is read from the request context, which should be
// set by the RealIP middleware.
//
// Takes next (http.Handler) which is the next handler in the chain to call.
//
// Returns http.Handler which wraps the next handler with rate limiting.
func (m *rateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.isExemptPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := security_dto.ClientIPFromRequest(r)

		result := m.checkRateLimit(r.Context(), clientIP, "global", m.config.Global)

		if m.config.HeadersEnabled {
			m.setRateLimitHeaders(w, result)
		}

		if !result.Allowed {
			w.Header()["Content-Type"] = headerValContentTypeText
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("Rate limit exceeded. Please try again later."))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ActionHandler applies rate limiting for a specific action and writes a 429
// response if the limit is exceeded.
//
// Call this within an action handler to apply per-action limits. The client IP
// is read from request context.
//
// Takes w (http.ResponseWriter) which receives rate limit headers and error
// responses.
// Takes r (*http.Request) which provides the request context containing the
// client IP.
// Takes override (*security_dto.RateLimitOverride) which allows customising the
// rate limit settings for this action, or nil to use defaults.
//
// Returns bool which is true if the request is allowed, or false if rate
// limited and a 429 response was written.
func (m *rateLimitMiddleware) ActionHandler(
	w http.ResponseWriter,
	r *http.Request,
	override *security_dto.RateLimitOverride,
) bool {
	clientIP := security_dto.ClientIPFromRequest(r)

	tierConfig := m.config.Actions
	keySuffix := "action"

	if override != nil {
		if override.RequestsPerMinute > 0 {
			tierConfig.RequestsPerMinute = override.RequestsPerMinute
		}
		if override.BurstSize > 0 {
			tierConfig.BurstSize = override.BurstSize
		}
		if override.KeySuffix != "" {
			keySuffix = override.KeySuffix
		}
	}

	result := m.checkRateLimit(r.Context(), clientIP, keySuffix, tierConfig)

	if m.config.HeadersEnabled {
		m.setRateLimitHeaders(w, result)
	}

	if !result.Allowed {
		w.Header()["Content-Type"] = headerValContentTypeText
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("Rate limit exceeded. Please try again later."))
		return false
	}

	return true
}

// CheckActionAllowed checks whether an action request is within its rate limit
// without writing headers or a response body. Use this for batch action paths
// where per-action HTTP responses are not written to the client directly.
//
// Takes r (*http.Request) which provides the request context containing the
// client IP.
// Takes override (*security_dto.RateLimitOverride) which allows customising the
// rate limit settings for this action, or nil to use defaults.
//
// Returns bool which is true if the request is allowed.
func (m *rateLimitMiddleware) CheckActionAllowed(
	r *http.Request,
	override *security_dto.RateLimitOverride,
) bool {
	clientIP := security_dto.ClientIPFromRequest(r)

	tierConfig := m.config.Actions
	keySuffix := "action"

	if override != nil {
		if override.RequestsPerMinute > 0 {
			tierConfig.RequestsPerMinute = override.RequestsPerMinute
		}
		if override.BurstSize > 0 {
			tierConfig.BurstSize = override.BurstSize
		}
		if override.KeySuffix != "" {
			keySuffix = override.KeySuffix
		}
	}

	result := m.checkRateLimit(r.Context(), clientIP, keySuffix, tierConfig)
	return result.Allowed
}

// checkRateLimit checks if a request is within the rate limit for a given key
// and tier.
//
// Takes ctx (context.Context) which carries cancellation through to the
// underlying counter store call.
// Takes clientIP (string) which identifies the client making the request.
// Takes keySuffix (string) which specifies the rate limit bucket name.
// Takes tier (security_dto.RateLimitTierValues) which defines the limit
// settings.
//
// Returns ratelimiter_dto.Result which contains the limit decision and
// remaining quota. If the service returns an error, the request is denied (fail
// closed) to prevent rate limit bypass during backend outages.
func (m *rateLimitMiddleware) checkRateLimit(
	ctx context.Context,
	clientIP string,
	keySuffix string,
	tier security_dto.RateLimitTierValues,
) ratelimiter_dto.Result {
	window := time.Minute
	key := "ratelimit:" + keySuffix + ":" + clientIP

	result, err := m.service.CheckLimit(ctx, key, tier.RequestsPerMinute, window)
	if err != nil {
		return ratelimiter_dto.Result{
			Allowed:    false,
			Limit:      tier.RequestsPerMinute,
			Remaining:  0,
			ResetAt:    m.clock.Now().Add(window),
			RetryAfter: window,
		}
	}

	return result
}

// setRateLimitHeaders adds rate limit headers to the HTTP response.
//
// Takes w (http.ResponseWriter) which receives the rate limit headers.
// Takes result (ratelimiter_dto.Result) which holds the current rate
// limit state.
func (*rateLimitMiddleware) setRateLimitHeaders(w http.ResponseWriter, result ratelimiter_dto.Result) {
	h := w.Header()
	h[headerRateLimitLimit] = []string{strconv.Itoa(result.Limit)}
	h[headerRateLimitRemaining] = []string{strconv.Itoa(result.Remaining)}
	h[headerRateLimitReset] = []string{strconv.FormatInt(result.ResetAt.Unix(), 10)}

	if !result.Allowed {
		h[headerRetryAfter] = []string{strconv.Itoa(int(result.RetryAfter.Seconds()))}
	}
}

// isExemptPath checks if the given path is exempt from rate limiting.
//
// Takes path (string) which is the request path to check.
//
// Returns bool which is true if the path matches any configured exempt prefix.
func (m *rateLimitMiddleware) isExemptPath(path string) bool {
	for _, exempt := range m.config.ExemptPaths {
		if strings.HasPrefix(path, exempt) {
			return true
		}
	}
	return false
}

// withRateLimitClock sets a custom clock for time operations. This is used
// mainly for testing to make timing predictable.
//
// Takes c (clock.Clock) which provides the time source for rate limiting.
//
// Returns rateLimitMiddlewareOption which sets the middleware clock.
func withRateLimitClock(c clock.Clock) rateLimitMiddlewareOption {
	return func(m *rateLimitMiddleware) {
		m.clock = c
	}
}

// newRateLimitMiddleware creates a new rate limit middleware instance. The
// middleware reads client IPs from request context, set by RealIP middleware.
//
// Takes config (security_dto.RateLimitValues) which specifies the rate limiting
// rules.
// Takes service (security_domain.RateLimitService) which tracks request rates.
// Takes opts (...rateLimitMiddlewareOption) which provides optional behaviour
// controls.
//
// Returns *rateLimitMiddleware which is configured and ready for use.
func newRateLimitMiddleware(
	config security_dto.RateLimitValues,
	service security_domain.RateLimitService,
	opts ...rateLimitMiddlewareOption,
) *rateLimitMiddleware {
	m := &rateLimitMiddleware{
		clock:   clock.RealClock(),
		service: service,
		config:  config,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}
