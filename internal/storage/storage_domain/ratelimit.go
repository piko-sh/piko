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

package storage_domain

import (
	"context"
	"fmt"

	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

// ProviderRateLimiter wraps rate limiting functionality for storage providers.
// It uses the centralised token bucket algorithm to control the rate of
// operations.
type ProviderRateLimiter struct {
	// limiter is the centralised rate limiter; nil disables rate limiting.
	limiter *ratelimiter_domain.Limiter

	// config defines the token bucket parameters for this provider.
	config ratelimiter_dto.TokenBucketConfig
}

// bucketKey is the fixed key used for the single bucket within each
// provider's InMemoryTokenBucketStore instance.
const bucketKey = "default"

// ProviderRateLimitConfig holds rate limiting configuration for a storage
// provider.
type ProviderRateLimitConfig struct {
	// Clock provides time operations for testing determinism; nil uses
	// RealClock().
	Clock clock.Clock

	// CallsPerSecond is the maximum operations per second.
	// A value of 0 or less disables rate limiting.
	CallsPerSecond float64

	// Burst is the maximum number of calls allowed in a short burst, letting
	// brief spikes go above the rate limit. If 0, defaults to CallsPerSecond.
	Burst int
}

// NewProviderRateLimiter creates a new rate limiter for a storage provider.
//
// When config has CallsPerSecond <= 0, returns nil (no rate limiting).
//
// Takes config (ProviderRateLimitConfig) which specifies the rate limit
// settings including calls per second and burst size.
//
// Returns *ProviderRateLimiter which is the configured limiter, or nil if
// rate limiting is disabled.
func NewProviderRateLimiter(config ProviderRateLimitConfig) *ProviderRateLimiter {
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

// Wait blocks until the rate limiter allows another operation.
//
// Returns immediately if the rate limiter is disabled (nil). Respects context
// cancellation.
//
// Returns error when the rate limiter fails or the context is cancelled.
func (r *ProviderRateLimiter) Wait(ctx context.Context) error {
	if r == nil || r.limiter == nil {
		return nil
	}
	if err := r.limiter.WaitTokenBucket(ctx, bucketKey, 1.0, r.config); err != nil {
		return fmt.Errorf("waiting for rate limiter: %w", err)
	}
	return nil
}

// Allow checks if an operation is currently allowed without blocking.
//
// Returns bool which is true if allowed, false if the rate limit would be
// exceeded.
func (r *ProviderRateLimiter) Allow() bool {
	if r == nil || r.limiter == nil {
		return true
	}
	err := r.limiter.AllowTokenBucket(context.Background(), bucketKey, 1.0, r.config)
	return err == nil
}

// ProviderOption is a function that configures a storage provider.
type ProviderOption func(*ProviderOptions)

// ProviderOptions holds settings for storage providers.
type ProviderOptions struct {
	// RateLimitConfig specifies the rate limiting settings for the provider.
	RateLimitConfig ProviderRateLimitConfig
}

// WithRateLimit sets the rate limiting configuration for a provider.
//
// Takes callsPerSecond (float64) which specifies the maximum operations per
// second.
// Takes burst (int) which specifies the maximum burst size allowed.
//
// Returns ProviderOption which configures rate limiting on a provider.
func WithRateLimit(callsPerSecond float64, burst int) ProviderOption {
	return func(opts *ProviderOptions) {
		opts.RateLimitConfig = ProviderRateLimitConfig{
			CallsPerSecond: callsPerSecond,
			Burst:          burst,
			Clock:          nil,
		}
	}
}

// WithUnlimitedRate disables rate limiting for a provider.
//
// Returns ProviderOption which sets a provider to have no rate limits.
func WithUnlimitedRate() ProviderOption {
	return func(opts *ProviderOptions) {
		opts.RateLimitConfig = ProviderRateLimitConfig{
			CallsPerSecond: 0,
			Burst:          0,
			Clock:          nil,
		}
	}
}

// ApplyProviderOptions applies functional options to create a rate limiter.
// This is used by provider constructors to initialise rate limiting.
//
// Takes defaults (ProviderRateLimitConfig) which specifies the base rate limit
// settings.
// Takes opts (...ProviderOption) which provides optional overrides to the
// defaults.
//
// Returns *ProviderRateLimiter which is configured with the merged options.
func ApplyProviderOptions(defaults ProviderRateLimitConfig, opts ...ProviderOption) *ProviderRateLimiter {
	options := &ProviderOptions{
		RateLimitConfig: defaults,
	}

	for _, opt := range opts {
		opt(options)
	}

	return NewProviderRateLimiter(options.RateLimitConfig)
}
