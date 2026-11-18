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

package ratelimiter_adapters

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultTokenBucketNamespace is the default cache namespace for token bucket state.
	defaultTokenBucketNamespace = "ratelimiter:tb"

	// defaultTokenBucketMaxSize is the default maximum number of entries for in-memory cache
	// providers.
	defaultTokenBucketMaxSize = 10000

	// tokenBucketExpiryMargin is added to the time a bucket needs to refill completely
	// before its key expires, so a bucket is never dropped while it could still be short of
	// tokens.
	tokenBucketExpiryMargin = time.Second
)

// CacheTokenBucketStoreConfig configures a cache-backed token bucket store.
type CacheTokenBucketStoreConfig struct {
	// CacheService is the cache service for storing bucket state; required.
	CacheService cache_domain.Service

	// Clock provides time operations. If nil, clock.RealClock() is used.
	Clock clock.Clock

	// Provider names the cache provider that holds buckets.
	//
	// Empty uses an in-memory cache local to this process. Instances share buckets only when
	// they use the same remote provider.
	Provider string

	// Namespace is the key prefix for all entries. Defaults to "ratelimiter:tb".
	Namespace string

	// MaximumEntries is the maximum number of entries for in-memory providers, ignored by
	// remote providers and defaulting to 10000.
	MaximumEntries int
}

// CacheTokenBucketStore keeps token buckets in a cache.
//
// Updates are atomic. A bucket expires after it could have refilled completely because an
// expired bucket is recreated full.
type CacheTokenBucketStore struct {
	// clock provides the current time for refill calculations.
	clock clock.Clock

	// cache stores token bucket states.
	cache cache_domain.Cache[string, *ratelimiter_domain.TokenBucketState]
}

var (
	_ ratelimiter_domain.TokenBucketStorePort = (*CacheTokenBucketStore)(nil)
)

// NewCacheTokenBucketStore creates a token bucket store backed by the cache hexagon.
//
// Takes config (CacheTokenBucketStoreConfig) which configures the store.
//
// Returns *CacheTokenBucketStore which is ready for use.
// Returns error when the cache cannot be created.
func NewCacheTokenBucketStore(ctx context.Context, config CacheTokenBucketStoreConfig) (*CacheTokenBucketStore, error) {
	cache, clk, err := newCacheStore[*ratelimiter_domain.TokenBucketState](ctx, cacheStoreParams{
		CacheService:     config.CacheService,
		Clock:            config.Clock,
		Provider:         config.Provider,
		Blueprint:        "ratelimiter-token-bucket",
		Namespace:        config.Namespace,
		DefaultNamespace: defaultTokenBucketNamespace,
		MaximumEntries:   config.MaximumEntries,
		DefaultMaxSize:   defaultTokenBucketMaxSize,
	})
	if err != nil {
		return nil, fmt.Errorf("creating token bucket cache: %w", err)
	}

	return &CacheTokenBucketStore{
		clock: clk,
		cache: cache,
	}, nil
}

// TryTake atomically attempts to take n tokens from the bucket. It first refills based on
// elapsed time, then attempts to deduct tokens.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens to take.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket parameters.
//
// Returns bool which is true if tokens were successfully taken.
// Returns error when config is nil or the cache operation fails.
func (s *CacheTokenBucketStore) TryTake(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (bool, error) {
	if config == nil {
		return false, errors.New("token bucket config is required")
	}

	var success bool

	_, _, err := s.cache.ComputeWithTTL(ctx, key, func(state *ratelimiter_domain.TokenBucketState, found bool) cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState] {
		success = false
		now := s.clock.Now().UnixNano()

		if !found || state == nil {
			state = ratelimiter_domain.NewBucketState(config, now)
		}

		state = ratelimiter_domain.RefillBucket(state, now)

		if state.Tokens >= n {
			state = &ratelimiter_domain.TokenBucketState{
				Tokens:         state.Tokens - n,
				MaxTokens:      state.MaxTokens,
				RefillRate:     state.RefillRate,
				LastRefillNano: state.LastRefillNano,
			}
			success = true
		}

		return storeBucket(state)
	})
	if err != nil {
		return false, fmt.Errorf("computing token bucket state: %w", err)
	}

	return success, nil
}

// WaitDuration returns the estimated time until n tokens become available.
//
// Takes key (string) which identifies the rate limit bucket.
// Takes n (float64) which is the number of tokens needed.
// Takes config (*ratelimiter_dto.TokenBucketConfig) which defines bucket parameters.
//
// Returns time.Duration which is zero if tokens are available, otherwise the estimated
// wait time.
// Returns error when config is nil or the cache operation fails.
func (s *CacheTokenBucketStore) WaitDuration(ctx context.Context, key string, n float64, config *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
	if config == nil {
		return 0, errors.New("token bucket config is required")
	}

	var waitDuration time.Duration

	_, _, err := s.cache.ComputeWithTTL(ctx, key, func(state *ratelimiter_domain.TokenBucketState, found bool) cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState] {
		waitDuration = 0
		now := s.clock.Now().UnixNano()

		if !found || state == nil {
			return storeBucket(ratelimiter_domain.NewBucketState(config, now))
		}

		state = ratelimiter_domain.RefillBucket(state, now)

		if state.Tokens < n {
			waitDuration = ratelimiter_domain.TokenWaitDuration(n-state.Tokens, state.RefillRate)
		}

		return cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState]{
			Value:  state,
			Action: cache_dto.ComputeActionNoop,
			TTL:    0,
		}
	})
	if err != nil {
		return 0, fmt.Errorf("computing wait duration: %w", err)
	}

	return waitDuration, nil
}

// DeleteBucket removes a bucket's state from storage.
//
// Takes key (string) which identifies the rate limit bucket.
//
// Returns error when the cache fails to remove the bucket.
func (s *CacheTokenBucketStore) DeleteBucket(ctx context.Context, key string) error {
	return s.cache.Invalidate(ctx, key)
}

// Close releases resources held by the store.
//
// Returns error when the underlying cache fails to close.
func (s *CacheTokenBucketStore) Close(ctx context.Context) error {
	return s.cache.Close(ctx)
}

// storeBucket builds the compute result that stores state until it would have refilled
// completely.
//
// Takes state (*ratelimiter_domain.TokenBucketState) which is the bucket to store.
//
// Returns cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState] which stores the
// bucket with its expiry.
func storeBucket(state *ratelimiter_domain.TokenBucketState) cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState] {
	return cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState]{
		Value:  state,
		Action: cache_dto.ComputeActionSet,
		TTL:    bucketTTL(state),
	}
}

// bucketTTL returns how long a bucket must be kept. Dropping a bucket after it has
// refilled loses nothing, because a missing bucket is recreated full.
//
// Takes state (*ratelimiter_domain.TokenBucketState) which is the bucket after the take.
//
// Returns time.Duration which is the time to refill completely plus a margin, or
// cache_dto.NoExpiry when the bucket never refills or the time does not fit a Duration.
func bucketTTL(state *ratelimiter_domain.TokenBucketState) time.Duration {
	if state.RefillRate <= 0 || math.IsNaN(state.RefillRate) {
		return cache_dto.NoExpiry
	}

	missing := max(state.MaxTokens-state.Tokens, 0)
	refillNanos := math.Ceil(missing / state.RefillRate)
	if refillNanos >= float64(math.MaxInt64-tokenBucketExpiryMargin) {
		return cache_dto.NoExpiry
	}

	return time.Duration(refillNanos) + tokenBucketExpiryMargin
}
