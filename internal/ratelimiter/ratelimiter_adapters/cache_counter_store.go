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
	"time"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultCounterNamespace is the default cache namespace for fixed window
	// counters.
	defaultCounterNamespace = "ratelimiter:fw"

	// defaultCounterMaxSize is the default maximum number of entries for in-memory
	// cache providers.
	defaultCounterMaxSize = 100000
)

// counterEntry holds the internal state of a fixed-window counter, including
// the count and the time the window started.
type counterEntry struct {
	// Count is the number of requests in the current window.
	Count int64

	// WindowStartNano is the Unix timestamp in nanoseconds when the window began.
	WindowStartNano int64
}

// CacheCounterStoreConfig configures a cache-backed counter store.
type CacheCounterStoreConfig struct {
	// CacheService is the cache service for storing counters; required.
	CacheService cache_domain.Service

	// Clock provides time operations. If nil, clock.RealClock() is used.
	Clock clock.Clock

	// Provider specifies the cache provider. If empty, uses the service's
	// default provider.
	Provider string

	// Namespace is the key prefix for all entries. Defaults to "ratelimiter:fw".
	Namespace string

	// MaximumSize is the maximum entry count for in-memory providers; remote
	// providers ignore this. Defaults to 100000.
	MaximumSize int
}

// CacheCounterStore implements CounterStorePort using a cache backend.
// It uses the cache's atomic ComputeWithTTL method to implement fixed-window
// rate limiting where the TTL is set only on the first increment.
type CacheCounterStore struct {
	// clock provides the current time for window start tracking.
	clock clock.Clock

	// cache stores counter entries with string keys.
	cache cache_domain.Cache[string, *counterEntry]
}

var _ ratelimiter_domain.CounterStorePort = (*CacheCounterStore)(nil)

// NewCacheCounterStore creates a counter store backed by the cache hexagon.
//
// Takes config (CacheCounterStoreConfig) which configures the store.
//
// Returns *CacheCounterStore which is ready for use.
// Returns error when the cache cannot be created.
func NewCacheCounterStore(ctx context.Context, config CacheCounterStoreConfig) (*CacheCounterStore, error) {
	cache, clk, err := newCacheStore[*counterEntry](ctx, cacheStoreParams{
		CacheService:     config.CacheService,
		Clock:            config.Clock,
		Blueprint:        "ratelimiter-counter",
		Namespace:        config.Namespace,
		DefaultNamespace: defaultCounterNamespace,
		MaximumSize:      config.MaximumSize,
		DefaultMaxSize:   defaultCounterMaxSize,
	})
	if err != nil {
		return nil, fmt.Errorf("creating counter cache: %w", err)
	}

	return &CacheCounterStore{
		clock: clk,
		cache: cache,
	}, nil
}

// IncrementAndGet atomically increments the counter for key by delta. The
// TTL is set only when the counter is first created, implementing a
// fixed-window strategy where the window starts on the first request.
//
// Takes key (string) which identifies the rate limit counter.
// Takes delta (int64) which is the amount to increment by.
// Takes window (time.Duration) which is the TTL for new counters.
//
// Returns ratelimiter_dto.CounterResult which contains the counter value
// after incrementing and the window start time.
// Returns error when the cache operation fails.
func (s *CacheCounterStore) IncrementAndGet(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error) {
	now := s.clock.Now()

	result, present, err := s.cache.ComputeWithTTL(ctx, key, func(oldValue *counterEntry, found bool) cache_dto.ComputeResult[*counterEntry] {
		if !found || oldValue == nil {
			return cache_dto.ComputeResult[*counterEntry]{
				Value: &counterEntry{
					Count:           delta,
					WindowStartNano: now.UnixNano(),
				},
				Action: cache_dto.ComputeActionSet,
				TTL:    window,
			}
		}

		return cache_dto.ComputeResult[*counterEntry]{
			Value: &counterEntry{
				Count:           oldValue.Count + delta,
				WindowStartNano: oldValue.WindowStartNano,
			},
			Action: cache_dto.ComputeActionSet,
		}
	})

	if err != nil {
		return ratelimiter_dto.CounterResult{}, fmt.Errorf("incrementing counter for key %q: %w", key, err)
	}
	if !present {
		return ratelimiter_dto.CounterResult{}, fmt.Errorf("failed to increment counter for key %q", key)
	}

	return ratelimiter_dto.CounterResult{
		Count:       result.Count,
		WindowStart: time.Unix(0, result.WindowStartNano).UTC(),
	}, nil
}

// Close releases resources held by the store.
func (s *CacheCounterStore) Close() {
	_ = s.cache.Close(context.Background())
}

// cacheStoreParams holds the configuration for creating a typed cache store.
type cacheStoreParams struct {
	// CacheService is the cache service for storing entries.
	CacheService cache_domain.Service

	// Clock provides time operations for window tracking.
	Clock clock.Clock

	// Blueprint is the factory blueprint name for cache creation.
	Blueprint string

	// Namespace is the key prefix for all cache entries.
	Namespace string

	// DefaultNamespace is the fallback namespace when Namespace is empty.
	DefaultNamespace string

	// MaximumSize is the maximum number of entries for in-memory providers.
	MaximumSize int

	// DefaultMaxSize is the fallback maximum size when MaximumSize is zero.
	DefaultMaxSize int
}

// newCacheStore validates configuration, applies defaults, and creates a typed
// cache instance. This is shared by both the counter and token bucket store
// constructors.
//
// Takes params (cacheStoreParams) which holds the cache configuration.
//
// Returns Cache[string, V] which is the created typed cache.
// Returns clock.Clock which is the clock to use (with defaults applied).
// Returns error when the cache service is nil or cache creation fails.
func newCacheStore[V any](ctx context.Context, params cacheStoreParams) (cache_domain.Cache[string, V], clock.Clock, error) {
	if params.CacheService == nil {
		var zero cache_domain.Cache[string, V]
		return zero, nil, errors.New("cache service is required")
	}

	namespace := params.Namespace
	if namespace == "" {
		namespace = params.DefaultNamespace
	}

	maximumSize := params.MaximumSize
	if maximumSize <= 0 {
		maximumSize = params.DefaultMaxSize
	}

	clk := params.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	cache, err := cache_domain.NewCacheBuilder[string, V](params.CacheService).
		FactoryBlueprint(params.Blueprint).
		Namespace(namespace).
		MaximumSize(maximumSize).
		Build(ctx)
	if err != nil {
		var zero cache_domain.Cache[string, V]
		return zero, nil, err
	}

	return cache, clk, nil
}
