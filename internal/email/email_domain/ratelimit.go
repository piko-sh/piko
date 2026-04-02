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

package email_domain

import (
	"context"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

// ProviderRateLimiter controls how quickly email providers can send messages.
// It wraps the centralised rate limiter with a provider-specific token bucket
// configuration.
type ProviderRateLimiter struct {
	// limiter is the underlying centralised rate limiter; nil disables rate limiting.
	limiter *ratelimiter_domain.Limiter

	// config defines the token bucket parameters for this provider.
	config ratelimiter_dto.TokenBucketConfig
}

// bucketKey is the fixed key used for the single bucket within each
// provider's InMemoryTokenBucketStore instance.
const bucketKey = "default"

// ProviderRateLimitConfig holds rate limiting configuration for an email provider.
type ProviderRateLimitConfig struct {
	// Clock provides time operations for testing determinism; nil uses RealClock().
	Clock clock.Clock

	// CallsPerSecond is the maximum number of API calls allowed per second.
	// A value of 0 or less disables rate limiting.
	CallsPerSecond float64

	// Burst is the maximum number of calls allowed at once; 0 uses CallsPerSecond.
	Burst int
}

// Wait blocks until the rate limiter allows another API call.
// Returns immediately if the rate limiter is disabled.
//
// Returns error when the context is cancelled or the deadline is exceeded.
func (r *ProviderRateLimiter) Wait(ctx context.Context) error {
	if r == nil || r.limiter == nil {
		return nil
	}
	err := r.limiter.WaitTokenBucket(ctx, bucketKey, 1.0, r.config)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return context.DeadlineExceeded
	}
	return nil
}

// ProviderOption is a function type that configures email provider settings.
type ProviderOption func(*providerOptions)

// providerOptions holds settings for email providers.
type providerOptions struct {
	// RateLimitConfig specifies the rate limiting settings for the provider.
	RateLimitConfig ProviderRateLimitConfig
}

// ApplyProviderOptions applies functional options to a default rate limit
// configuration and returns a configured rate limiter. This is used by email
// provider implementations to create their rate limiters.
//
// Takes defaults (ProviderRateLimitConfig) which specifies the base rate limit
// settings.
// Takes opts (...ProviderOption) which provides optional overrides for the
// default configuration.
//
// Returns *ProviderRateLimiter which is ready for use by an email provider.
func ApplyProviderOptions(defaults ProviderRateLimitConfig, opts ...ProviderOption) *ProviderRateLimiter {
	options := &providerOptions{
		RateLimitConfig: defaults,
	}

	for _, opt := range opts {
		opt(options)
	}

	return newProviderRateLimiter(options.RateLimitConfig)
}

// newProviderRateLimiter creates a rate limiter for controlling API calls to a
// provider.
//
// When config has CallsPerSecond of zero or less, returns nil to show that no
// rate limiting should be used.
//
// Takes config (ProviderRateLimitConfig) which gives the rate limit settings
// including calls per second and optional burst size.
//
// Returns *ProviderRateLimiter which is the configured rate limiter, or nil if
// rate limiting is disabled.
func newProviderRateLimiter(config ProviderRateLimitConfig) *ProviderRateLimiter {
	if config.CallsPerSecond <= 0 {
		return nil
	}

	burst := config.Burst
	if burst <= 0 {
		burst = int(config.CallsPerSecond)
	}

	bucketConfig := ratelimiter_dto.TokenBucketConfig{
		Rate:  config.CallsPerSecond,
		Burst: burst,
	}

	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	store := ratelimiter_adapters.NewInMemoryTokenBucketStore(
		ratelimiter_adapters.WithInMemoryClock(clk),
	)
	limiter := ratelimiter_domain.NewLimiter(store, ratelimiter_adapters.NoopCounterStore{},
		ratelimiter_domain.WithClock(clk),
	)

	return &ProviderRateLimiter{
		limiter: limiter,
		config:  bucketConfig,
	}
}

// withRateLimit sets the rate limit settings for a provider.
//
// Takes callsPerSecond (float64) which sets the maximum calls allowed per
// second.
// Takes burst (int) which sets the maximum burst size for short periods.
//
// Returns ProviderOption which applies the rate limit settings to a provider.
func withRateLimit(callsPerSecond float64, burst int) ProviderOption {
	return func(opts *providerOptions) {
		opts.RateLimitConfig = ProviderRateLimitConfig{
			CallsPerSecond: callsPerSecond,
			Burst:          burst,
			Clock:          nil,
		}
	}
}

// withUnlimitedRate turns off rate limiting for a provider.
//
// Returns ProviderOption which sets up a provider to have no rate limit.
func withUnlimitedRate() ProviderOption {
	return func(opts *providerOptions) {
		opts.RateLimitConfig = ProviderRateLimitConfig{
			CallsPerSecond: 0,
			Burst:          0,
			Clock:          nil,
		}
	}
}
