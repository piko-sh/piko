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

package llm_domain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// SecondsPerMinute is the number of seconds in a minute.
	SecondsPerMinute = 60.0
)

// rateLimitConfig stores rate limiting settings for a scope.
type rateLimitConfig struct {
	// requestsPerMinute is the maximum allowed requests per minute; 0 disables
	// request rate limiting.
	requestsPerMinute int

	// tokensPerMinute is the maximum number of tokens allowed per minute;
	// 0 disables token rate limiting.
	tokensPerMinute int
}

// RateLimiter enforces request and token rate limits using the token bucket
// algorithm. It delegates state management to a TokenBucketStorePort
// implementation, which can be backed by in-memory storage (Otter) or
// distributed storage (Redis).
type RateLimiter struct {
	// clock provides time functions for rate limiting; used for creating timers.
	clock clock.Clock

	// store provides bucket storage for rate limit tracking.
	store ratelimiter_domain.TokenBucketStorePort

	// configs maps scope names to their rate limit settings.
	configs map[string]*rateLimitConfig

	// mu guards access to configs and limiters maps.
	mu sync.RWMutex
}

// RateLimiterOption is a function type that configures a RateLimiter.
type RateLimiterOption func(*RateLimiter)

// NewRateLimiter creates a new RateLimiter with the given store.
//
// Takes store (ratelimiter_domain.TokenBucketStorePort) which provides token
// bucket state storage.
// Takes opts (...RateLimiterOption) which are optional configuration functions.
//
// Returns *RateLimiter ready for configuration.
func NewRateLimiter(store ratelimiter_domain.TokenBucketStorePort, opts ...RateLimiterOption) *RateLimiter {
	l := &RateLimiter{
		clock:   clock.RealClock(),
		store:   store,
		configs: make(map[string]*rateLimitConfig),
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// SetLimits configures rate limits for a scope.
//
// Takes scope (string) which identifies the rate limit scope.
// Takes requestsPerMinute (int) which is the max requests per minute
// (0 = unlimited).
// Takes tokensPerMinute (int) which is the max tokens per minute
// (0 = unlimited).
//
// Safe for concurrent use.
func (l *RateLimiter) SetLimits(scope string, requestsPerMinute, tokensPerMinute int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.configs[scope] = &rateLimitConfig{
		requestsPerMinute: requestsPerMinute,
		tokensPerMinute:   tokensPerMinute,
	}
}

// RemoveLimits removes rate limits for a scope.
//
// Takes scope (string) which identifies the scope to remove limits from.
//
// Safe for concurrent use. Protected by mutex.
func (l *RateLimiter) RemoveLimits(scope string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.configs, scope)

	backgroundCtx := context.Background()
	if requestDeleteError := l.store.DeleteBucket(backgroundCtx, buildBucketKey(scope, BucketTypeRequest)); requestDeleteError != nil {
		_, warningLogger := logger_domain.From(backgroundCtx, nil)
		warningLogger.Warn("failed to delete request rate limit bucket",
			logger_domain.String("scope", scope),
			logger_domain.Error(requestDeleteError))
	}
	if tokenDeleteError := l.store.DeleteBucket(backgroundCtx, buildBucketKey(scope, BucketTypeToken)); tokenDeleteError != nil {
		_, warningLogger := logger_domain.From(backgroundCtx, nil)
		warningLogger.Warn("failed to delete token rate limit bucket",
			logger_domain.String("scope", scope),
			logger_domain.Error(tokenDeleteError))
	}
}

// Allow checks if a single request is allowed for the scope.
// Returns ErrRateLimited if the request would exceed the rate limit.
//
// Takes scope (string) which identifies the rate limit scope.
//
// Returns error which is ErrRateLimited if the rate limit is exceeded.
func (l *RateLimiter) Allow(ctx context.Context, scope string) error {
	return l.AllowN(ctx, scope, 1, 0)
}

// AllowN checks if a request consuming the specified tokens is allowed.
// Returns ErrRateLimited if the request would exceed the rate limit.
//
// Takes scope (string) which identifies the rate limit scope.
// Takes requests (int) which is the number of requests to account for.
// Takes tokens (int) which is the number of tokens to account for.
//
// Returns error when the rate limit is exceeded or the store fails.
//
// Safe for concurrent use.
func (l *RateLimiter) AllowN(ctx context.Context, scope string, requests, tokens int) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	l.mu.RLock()
	config := l.configs[scope]
	l.mu.RUnlock()

	if config == nil {
		return nil
	}

	if config.requestsPerMinute > 0 && requests > 0 {
		key := buildBucketKey(scope, BucketTypeRequest)
		ok, err := l.store.TryTake(ctx, key, float64(requests), new(requestBucketConfig(config)))
		if err != nil {
			return fmt.Errorf("checking request rate limit for scope %q: %w", scope, err)
		}
		if !ok {
			return ErrRateLimited
		}
	}

	if config.tokensPerMinute > 0 && tokens > 0 {
		key := buildBucketKey(scope, BucketTypeToken)
		ok, err := l.store.TryTake(ctx, key, float64(tokens), new(tokenBucketConfig(config)))
		if err != nil {
			return fmt.Errorf("checking token rate limit for scope %q: %w", scope, err)
		}
		if !ok {
			return ErrRateLimited
		}
	}

	return nil
}

// Wait blocks until the request is allowed or the context is cancelled.
//
// Takes scope (string) which identifies the rate limit scope.
//
// Returns error when the context is cancelled before the request is allowed.
func (l *RateLimiter) Wait(ctx context.Context, scope string) error {
	return l.WaitN(ctx, scope, 1, 0)
}

// WaitN blocks until the specified request and token count is allowed or the
// context is cancelled.
//
// Takes scope (string) which identifies the rate limit scope.
// Takes requests (int) which is the number of requests to wait for.
// Takes tokens (int) which is the number of tokens to wait for.
//
// Returns error when the context is cancelled before the request is allowed.
func (l *RateLimiter) WaitN(ctx context.Context, scope string, requests, tokens int) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		maxWait := l.calculateWaitDuration(ctx, scope, requests, tokens)

		if done, err := l.tryAllowOrWait(ctx, scope, requests, tokens, maxWait); done {
			if err != nil {
				return fmt.Errorf("waiting for rate limit on scope %q: %w", scope, err)
			}
			return nil
		}
	}
}

// HasLimits reports whether a scope has rate limits configured.
//
// Takes scope (string) which identifies the rate limit scope.
//
// Returns bool which is true if rate limits are configured for the scope.
//
// Safe for concurrent use.
func (l *RateLimiter) HasLimits(scope string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, exists := l.configs[scope]
	return exists
}

// GetLimits returns the current rate limits for a scope.
//
// Takes scope (string) which identifies the rate limit scope.
//
// Returns requestsPerMinute (int) which is the requests per minute limit,
// or 0 if not configured.
// Returns tokensPerMinute (int) which is the tokens per minute limit,
// or 0 if not configured.
//
// Safe for concurrent use.
func (l *RateLimiter) GetLimits(scope string) (requestsPerMinute, tokensPerMinute int) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	config, exists := l.configs[scope]
	if !exists {
		return 0, 0
	}
	return config.requestsPerMinute, config.tokensPerMinute
}

// calculateWaitDuration calculates the maximum wait duration across all
// buckets for a scope.
//
// Takes scope (string) which identifies the rate limit configuration to use.
// Takes requests (int) which specifies the number of requests to check.
// Takes tokens (int) which specifies the number of tokens to check.
//
// Returns time.Duration which is the longest wait needed across all buckets,
// or zero if the scope has no configuration.
//
// Safe for concurrent use; acquires a read lock to access scope configuration.
func (l *RateLimiter) calculateWaitDuration(ctx context.Context, scope string, requests, tokens int) time.Duration {
	l.mu.RLock()
	config := l.configs[scope]
	l.mu.RUnlock()

	if config == nil {
		return 0
	}

	var maxWait time.Duration

	if config.requestsPerMinute > 0 && requests > 0 {
		key := buildBucketKey(scope, BucketTypeRequest)
		wait, err := l.store.WaitDuration(ctx, key, float64(requests), new(requestBucketConfig(config)))
		if err == nil && wait > maxWait {
			maxWait = wait
		}
	}

	if config.tokensPerMinute > 0 && tokens > 0 {
		key := buildBucketKey(scope, BucketTypeToken)
		wait, err := l.store.WaitDuration(ctx, key, float64(tokens), new(tokenBucketConfig(config)))
		if err == nil && wait > maxWait {
			maxWait = wait
		}
	}

	return maxWait
}

// tryAllowOrWait attempts to take tokens or waits for the specified duration.
//
// Takes scope (string) which identifies the rate limit bucket.
// Takes requests (int) which specifies the number of requests to consume.
// Takes tokens (int) which specifies the number of tokens to consume.
// Takes maxWait (time.Duration) which sets the maximum time to wait for tokens.
//
// Returns bool which indicates whether to stop the wait loop.
// Returns error when the context is cancelled.
func (l *RateLimiter) tryAllowOrWait(ctx context.Context, scope string, requests, tokens int, maxWait time.Duration) (bool, error) {
	if maxWait == 0 {
		if err := l.AllowN(ctx, scope, requests, tokens); err == nil {
			return true, nil
		}
		maxWait = time.Millisecond
	}

	timer := l.clock.NewTimer(maxWait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-timer.C():
		return false, nil
	}
}

// WithRateLimiterClock sets the clock used for time operations.
// If not set, clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns RateLimiterOption to apply to the rate limiter.
func WithRateLimiterClock(c clock.Clock) RateLimiterOption {
	return func(l *RateLimiter) {
		l.clock = c
	}
}

// buildBucketKey constructs the cache key for a scope and bucket type.
//
// Takes scope (string) which identifies the cache namespace.
// Takes bucketType (BucketType) which specifies the type of bucket.
//
// Returns string which is the combined key in "scope:bucketType" format.
func buildBucketKey(scope string, bucketType BucketType) string {
	return scope + ":" + string(bucketType)
}

// requestBucketConfig creates the token bucket config for request rate limiting.
//
// Takes config (*rateLimitConfig) which specifies the rate limit settings.
//
// Returns ratelimiter_dto.TokenBucketConfig which contains the computed rate
// and burst values.
func requestBucketConfig(config *rateLimitConfig) ratelimiter_dto.TokenBucketConfig {
	return ratelimiter_dto.TokenBucketConfig{
		Rate:  float64(config.requestsPerMinute) / SecondsPerMinute,
		Burst: config.requestsPerMinute,
	}
}

// tokenBucketConfig creates the token bucket config for token rate limiting.
//
// Takes config (*rateLimitConfig) which specifies the rate limit settings.
//
// Returns ratelimiter_dto.TokenBucketConfig which contains the calculated rate
// and burst values.
func tokenBucketConfig(config *rateLimitConfig) ratelimiter_dto.TokenBucketConfig {
	return ratelimiter_dto.TokenBucketConfig{
		Rate:  float64(config.tokensPerMinute) / SecondsPerMinute,
		Burst: config.tokensPerMinute,
	}
}
