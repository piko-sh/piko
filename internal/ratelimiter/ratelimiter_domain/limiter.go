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

package ratelimiter_domain

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

// Limiter enforces rate limits using strategy-specific methods and
// cache-backed storage. It supports both token bucket and fixed window
// algorithms via separate method families.
//
// All methods are safe for concurrent use.
type Limiter struct {
	// clock provides time functions for rate limiting calculations.
	clock clock.Clock

	// tokenStore provides atomic token bucket state operations.
	tokenStore TokenBucketStorePort

	// counterStore provides atomic counter operations for fixed windows.
	counterStore CounterStorePort

	// keyPrefix is prepended to all rate limit keys.
	keyPrefix string

	// tokenStoreName is the human-readable name of the token bucket store
	// (e.g. "cache", "inmemory", "noop").
	tokenStoreName string

	// counterStoreName is the human-readable name of the counter store
	// (e.g. "cache", "noop").
	counterStoreName string

	// failPolicy determines behaviour when the store is unavailable.
	failPolicy ratelimiter_dto.FailPolicy

	// totalChecks counts the total number of rate limit checks.
	totalChecks atomic.Int64

	// totalAllowed counts the number of allowed requests.
	totalAllowed atomic.Int64

	// totalDenied counts the number of denied requests.
	totalDenied atomic.Int64

	// totalErrors counts the number of store errors encountered.
	totalErrors atomic.Int64
}

var _ RateLimiterInspector = (*Limiter)(nil)

// NewLimiter creates a new rate limiter with the given stores.
//
// Takes tokenStore (TokenBucketStorePort) for token bucket operations. May be
// nil if only fixed window is used.
// Takes counterStore (CounterStorePort) for fixed window operations. May be
// nil if only token bucket is used.
// Takes opts (...Option) which are optional configuration functions.
//
// Returns *Limiter ready for use.
func NewLimiter(tokenStore TokenBucketStorePort, counterStore CounterStorePort, opts ...Option) *Limiter {
	l := &Limiter{
		clock:        clock.RealClock(),
		tokenStore:   tokenStore,
		counterStore: counterStore,
		failPolicy:   ratelimiter_dto.FailOpen,
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// AllowTokenBucket checks whether n tokens can be taken from the bucket
// identified by key. It returns nil if the request is allowed, or
// ErrRateLimited if the rate limit is exceeded.
//
// This is a non-blocking check. For a blocking variant, use WaitTokenBucket.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens to consume.
// Takes config (ratelimiter_dto.TokenBucketConfig) which defines the bucket
// parameters.
//
// Returns error which is nil if allowed, ErrRateLimited if denied, or a
// wrapped error on store failure (subject to FailPolicy).
func (l *Limiter) AllowTokenBucket(ctx context.Context, key string, n float64, config ratelimiter_dto.TokenBucketConfig) error {
	start := l.clock.Now()
	fullKey := l.buildKey(key)

	checksTotal.Add(ctx, 1)
	l.totalChecks.Add(1)

	allowed, err := l.tokenStore.TryTake(ctx, fullKey, n, &config)
	if err != nil {
		return l.handleStoreError(ctx, start, err, "token bucket TryTake", key)
	}

	l.recordLatency(ctx, start)

	if !allowed {
		deniedTotal.Add(ctx, 1)
		l.totalDenied.Add(1)
		return ErrRateLimited
	}

	allowedTotal.Add(ctx, 1)
	l.totalAllowed.Add(1)
	return nil
}

// WaitTokenBucket blocks until n tokens are available in the bucket
// identified by key, or until the context is cancelled.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens to consume.
// Takes config (ratelimiter_dto.TokenBucketConfig) which defines the bucket
// parameters.
//
// Returns error which is nil on success, or a context error if cancelled.
func (l *Limiter) WaitTokenBucket(ctx context.Context, key string, n float64, config ratelimiter_dto.TokenBucketConfig) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := l.AllowTokenBucket(ctx, key, n, config)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrRateLimited) {
			return fmt.Errorf("checking token bucket allowance for key %q: %w", key, err)
		}

		waitDur := l.estimateWaitDuration(ctx, key, n, config)

		if cancelled, waitErr := l.waitOrCancel(ctx, waitDur); cancelled {
			if waitErr != nil {
				return fmt.Errorf("waiting for rate limit on key %q: %w", key, waitErr)
			}
		}
	}
}

// CheckFixedWindow performs a fixed window rate limit check and returns a
// detailed Result. Unlike the token bucket methods, this returns a Result
// with remaining quota and reset information, suitable for HTTP rate limit
// headers.
//
// Takes key (string) which identifies the rate limit counter.
// Takes config (ratelimiter_dto.FixedWindowConfig) which defines the window
// parameters.
//
// Returns ratelimiter_dto.Result which contains the rate limit decision and
// metadata.
// Returns error when the store operation fails (subject to FailPolicy).
func (l *Limiter) CheckFixedWindow(ctx context.Context, key string, config ratelimiter_dto.FixedWindowConfig) (ratelimiter_dto.Result, error) {
	start := l.clock.Now()
	fullKey := l.buildKey(key)

	checksTotal.Add(ctx, 1)
	l.totalChecks.Add(1)

	counterResult, err := l.counterStore.IncrementAndGet(ctx, fullKey, 1, config.Window)
	if err != nil {
		storeErr := l.handleStoreError(ctx, start, err, "fixed window IncrementAndGet", key)
		if l.failPolicy == ratelimiter_dto.FailOpen {
			l.totalAllowed.Add(1)
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     config.Limit,
				Remaining: config.Limit - 1,
				ResetAt:   start.Add(config.Window),
			}, nil
		}
		return ratelimiter_dto.Result{}, storeErr
	}

	l.recordLatency(ctx, start)

	remaining := max(config.Limit-int(counterResult.Count), 0)
	allowed := int(counterResult.Count) <= config.Limit
	resetAt := counterResult.WindowStart.Add(config.Window)

	var retryAfter time.Duration
	if !allowed {
		retryAfter = max(resetAt.Sub(l.clock.Now()), 0)
		deniedTotal.Add(ctx, 1)
		l.totalDenied.Add(1)
	} else {
		allowedTotal.Add(ctx, 1)
		l.totalAllowed.Add(1)
	}

	return ratelimiter_dto.Result{
		Allowed:    allowed,
		Limit:      config.Limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}, nil
}

// DeleteBucket removes stored state for a token bucket key.
//
// Takes key (string) which identifies the rate limit bucket to remove.
//
// Returns error when the deletion fails.
func (l *Limiter) DeleteBucket(ctx context.Context, key string) error {
	fullKey := l.buildKey(key)
	if err := l.tokenStore.DeleteBucket(ctx, fullKey); err != nil {
		return fmt.Errorf("deleting bucket %q: %w", key, err)
	}
	return nil
}

// GetStatus returns the current inspectable state of the rate limiter.
//
// Returns ratelimiter_dto.Status which contains store names, fail policy,
// and aggregate counters.
// Returns error (always nil for the in-process limiter).
func (l *Limiter) GetStatus(_ context.Context) (ratelimiter_dto.Status, error) {
	return ratelimiter_dto.Status{
		TokenBucketStore: l.tokenStoreName,
		CounterStore:     l.counterStoreName,
		FailPolicy:       failPolicyString(l.failPolicy),
		KeyPrefix:        l.keyPrefix,
		TotalChecks:      l.totalChecks.Load(),
		TotalAllowed:     l.totalAllowed.Load(),
		TotalDenied:      l.totalDenied.Load(),
		TotalErrors:      l.totalErrors.Load(),
	}, nil
}

// buildKey constructs the full cache key by prepending the key prefix.
//
// Takes key (string) which is the base key to prefix.
//
// Returns string which is the full key with prefix, or the original key if
// no prefix is set.
func (l *Limiter) buildKey(key string) string {
	if l.keyPrefix == "" {
		return key
	}
	return l.keyPrefix + ":" + key
}

// estimateWaitDuration queries the store for the estimated wait time until
// tokens become available.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which specifies the number of tokens requested.
// Takes config (ratelimiter_dto.TokenBucketConfig) which defines the bucket rules.
//
// Returns time.Duration which is the estimated wait time, or one millisecond
// if unavailable.
func (l *Limiter) estimateWaitDuration(ctx context.Context, key string, n float64, config ratelimiter_dto.TokenBucketConfig) time.Duration {
	fullKey := l.buildKey(key)
	wait, err := l.tokenStore.WaitDuration(ctx, fullKey, n, &config)
	if err != nil || wait == 0 {
		return time.Millisecond
	}
	return wait
}

// waitOrCancel waits for the specified duration or until the context is
// cancelled. Returns (cancelled=true, err) if the context was cancelled, or
// (cancelled=false, nil) when the timer fires.
//
// Takes ctx (context.Context) which may cancel the wait early.
// Takes d (time.Duration) which is how long to wait before proceeding.
//
// Returns cancelled (bool) which is true when the context was cancelled
// before the timer fired.
// Returns err (error) which is the context cancellation error, or nil
// when the timer fired normally.
func (l *Limiter) waitOrCancel(ctx context.Context, d time.Duration) (cancelled bool, err error) {
	timer := l.clock.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-timer.C():
		return false, nil
	}
}

// handleStoreError processes a store error according to the fail policy.
// It records metrics and logs the error.
//
// Takes start (time.Time) which marks when the operation began for latency.
// Takes err (error) which is the store error to handle.
// Takes operation (string) which identifies the operation that failed.
// Takes key (string) which identifies the rate limit key involved.
//
// Returns error when the fail policy is FailClosed, nil when FailOpen.
func (l *Limiter) handleStoreError(ctx context.Context, start time.Time, err error, operation, key string) error {
	ctx, lg := logger_domain.From(ctx, log)
	errorsTotal.Add(ctx, 1)
	l.totalErrors.Add(1)
	l.recordLatency(ctx, start)

	lg.Warn("Rate limiter store error",
		logger_domain.String("operation", operation),
		logger_domain.String("key", key),
		logger_domain.Error(err),
	)

	if l.failPolicy == ratelimiter_dto.FailOpen {
		allowedTotal.Add(ctx, 1)
		l.totalAllowed.Add(1)
		return nil
	}

	deniedTotal.Add(ctx, 1)
	l.totalDenied.Add(1)
	return fmt.Errorf("%w: %s: %w", ErrStoreFailure, operation, err)
}

// recordLatency records the duration of a rate limit check.
//
// Takes start (time.Time) which is the timestamp when the check began.
func (l *Limiter) recordLatency(ctx context.Context, start time.Time) {
	elapsed := float64(l.clock.Now().Sub(start).Milliseconds())
	checkDuration.Record(ctx, elapsed)
}

// failPolicyString returns the human-readable string for a FailPolicy value.
//
// Takes p (ratelimiter_dto.FailPolicy) which is the policy to convert.
//
// Returns string which is "closed" for FailClosed or "open" otherwise.
func failPolicyString(p ratelimiter_dto.FailPolicy) string {
	if p == ratelimiter_dto.FailClosed {
		return "closed"
	}
	return "open"
}
