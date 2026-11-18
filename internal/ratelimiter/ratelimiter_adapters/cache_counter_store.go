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
	"cmp"
	"context"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/cache/cache_adapters/encoder_json"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// defaultCounterNamespace is the default cache namespace for fixed window counters.
	defaultCounterNamespace = "ratelimiter:fw"

	// defaultCounterMaxSize is the default maximum number of entries for in-memory cache
	// providers.
	defaultCounterMaxSize = 100000

	// defaultClockSkewTolerance is the clock skew allowed between instances that share a
	// remote counter store when the config leaves ClockSkewTolerance at zero.
	defaultClockSkewTolerance = 2 * time.Second
)

// counterEntry holds the internal state of a fixed-window counter, including the count
// and the time the window started.
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

	// Provider names the cache provider that holds counters.
	//
	// Empty uses an in-memory cache local to this process. Instances share limits only when
	// they use the same remote provider.
	Provider string

	// Namespace is the key prefix for all entries. Defaults to "ratelimiter:fw".
	Namespace string

	// MaximumEntries is the maximum entry count for in-memory providers; remote providers
	// ignore this. Defaults to 100000.
	MaximumEntries int

	// ClockSkewTolerance extends a counter's window to absorb clock differences.
	//
	// A counter may keep its current count this long past the nominal window end, while the
	// reported reset time remains at that end. Zero allows no skew for a local cache and two
	// seconds with a remote provider. Negative values are rejected.
	ClockSkewTolerance time.Duration
}

// CacheCounterStore keeps fixed-window counters in a cache.
//
// An increment at or after the window and any allowed clock skew starts a new one. The
// cache can discard inactive counters after their window expires.
type CacheCounterStore struct {
	// clock provides the current time for window start tracking.
	clock clock.Clock

	// cache stores counter entries with string keys.
	cache cache_domain.Cache[string, *counterEntry]

	// clockSkewTolerance is added to the window before the store itself starts a new one.
	clockSkewTolerance time.Duration
}

// cacheStoreParams holds the configuration for creating a typed cache store.
type cacheStoreParams struct {
	// CacheService is the cache service for storing entries.
	CacheService cache_domain.Service

	// Clock provides time operations for window tracking.
	Clock clock.Clock

	// Provider names the cache provider; empty uses the typed in-memory blueprint.
	Provider string

	// Blueprint is the factory blueprint name for cache creation.
	Blueprint string

	// Namespace is the key prefix for all cache entries.
	Namespace string

	// DefaultNamespace is the fallback namespace when Namespace is empty.
	DefaultNamespace string

	// MaximumEntries is the maximum number of entries for in-memory providers.
	MaximumEntries int

	// DefaultMaxSize is the fallback maximum size when MaximumEntries is zero.
	DefaultMaxSize int
}

var (
	_ ratelimiter_domain.CounterStorePort = (*CacheCounterStore)(nil)
)

// NewCacheCounterStore creates a counter store backed by the cache hexagon.
//
// Takes config (CacheCounterStoreConfig) which configures the store.
//
// Returns *CacheCounterStore which is ready for use.
// Returns error when the config is invalid or the cache cannot be created.
func NewCacheCounterStore(ctx context.Context, config CacheCounterStoreConfig) (*CacheCounterStore, error) {
	tolerance, err := resolveClockSkewTolerance(config.ClockSkewTolerance, config.Provider)
	if err != nil {
		return nil, err
	}

	cache, clk, err := newCacheStore[*counterEntry](ctx, cacheStoreParams{
		CacheService:     config.CacheService,
		Clock:            config.Clock,
		Provider:         config.Provider,
		Blueprint:        "ratelimiter-counter",
		Namespace:        config.Namespace,
		DefaultNamespace: defaultCounterNamespace,
		MaximumEntries:   config.MaximumEntries,
		DefaultMaxSize:   defaultCounterMaxSize,
	})
	if err != nil {
		return nil, fmt.Errorf("creating counter cache: %w", err)
	}

	return &CacheCounterStore{
		clock:              clk,
		cache:              cache,
		clockSkewTolerance: tolerance,
	}, nil
}

// IncrementAndGet atomically increments the counter for key by delta.
//
// The first increment for a key, or at or after the window and clock skew tolerance have
// elapsed, starts a new window holding delta.
//
// Takes key (string) which identifies the rate limit counter.
// Takes delta (int64) which is the amount to increment by.
// Takes window (time.Duration) which is the length of each window and the TTL of the
// counter when its window starts.
//
// Returns ratelimiter_dto.CounterResult which contains the counter value after
// incrementing and the window start time.
// Returns error when the cache operation fails.
func (s *CacheCounterStore) IncrementAndGet(ctx context.Context, key string, delta int64, window time.Duration) (ratelimiter_dto.CounterResult, error) {
	now := s.clock.Now()

	result, present, err := s.cache.ComputeWithTTL(ctx, key, func(oldValue *counterEntry, found bool) cache_dto.ComputeResult[*counterEntry] {
		if !found || oldValue == nil {
			return startWindow(now, delta, window)
		}
		if oldValue.windowElapsed(now, window+s.clockSkewTolerance) {
			return startWindow(now, delta, window)
		}
		return continueWindow(oldValue, delta)
	})

	if err != nil {
		return ratelimiter_dto.CounterResult{}, fmt.Errorf("incrementing counter for key %q: %w", key, err)
	}
	if !present || result == nil {
		return ratelimiter_dto.CounterResult{}, fmt.Errorf("failed to increment counter for key %q", key)
	}

	return ratelimiter_dto.CounterResult{
		Count:       result.Count,
		WindowStart: time.Unix(0, result.WindowStartNano).UTC(),
	}, nil
}

// Close releases resources held by the store.
//
// Returns error when the underlying cache fails to close.
func (s *CacheCounterStore) Close(ctx context.Context) error {
	return s.cache.Close(ctx)
}

// windowElapsed reports whether the counter's window has ended by now. The store checks
// this itself because the cache may keep an entry past its TTL, and a counter that
// outlived its window would carry its count into the next one.
//
// Takes now (time.Time) which is the time of the increment.
// Takes window (time.Duration) which is the length of the window, including any clock
// skew tolerance.
//
// Returns bool which is true when now is at or after the end of the window.
func (e *counterEntry) windowElapsed(now time.Time, window time.Duration) bool {
	return now.UnixNano()-e.WindowStartNano >= window.Nanoseconds()
}

// resolveClockSkewTolerance sets the default skew tolerance for remote providers.
//
// Takes tolerance (time.Duration) which is the configured tolerance.
// Takes provider (string) which is the configured cache provider.
//
// Returns time.Duration which is the tolerance to use.
// Returns error when tolerance is negative.
func resolveClockSkewTolerance(tolerance time.Duration, provider string) (time.Duration, error) {
	if tolerance < 0 {
		return 0, fmt.Errorf("%w: clock skew tolerance must not be negative, got %s",
			ratelimiter_domain.ErrInvalidConfig, tolerance)
	}
	if tolerance == 0 && provider != "" {
		return defaultClockSkewTolerance, nil
	}
	return tolerance, nil
}

// startWindow builds the compute result that opens a new window at now holding delta.
//
// Takes now (time.Time) which is when the window starts.
// Takes delta (int64) which is the window's first count.
// Takes window (time.Duration) which is the TTL of the new counter.
//
// Returns cache_dto.ComputeResult[*counterEntry] which stores the new counter.
func startWindow(now time.Time, delta int64, window time.Duration) cache_dto.ComputeResult[*counterEntry] {
	return cache_dto.ComputeResult[*counterEntry]{
		Value: &counterEntry{
			Count:           delta,
			WindowStartNano: now.UnixNano(),
		},
		Action: cache_dto.ComputeActionSet,
		TTL:    window,
	}
}

// continueWindow builds the compute result that adds delta to the current window and
// leaves the counter's expiry unchanged.
//
// Takes entry (*counterEntry) which is the counter for the current window.
// Takes delta (int64) which is the amount to add.
//
// Returns cache_dto.ComputeResult[*counterEntry] which stores the incremented counter.
func continueWindow(entry *counterEntry, delta int64) cache_dto.ComputeResult[*counterEntry] {
	return cache_dto.ComputeResult[*counterEntry]{
		Value: &counterEntry{
			Count:           entry.Count + delta,
			WindowStartNano: entry.WindowStartNano,
		},
		Action: cache_dto.ComputeActionSet,
		TTL:    0,
	}
}

// newCacheStore validates configuration, applies defaults, and creates a typed cache
// instance with per-entry expiry. This is shared by both the counter and token bucket
// store constructors.
//
// With no provider the cache is the typed in-memory blueprint. With a provider the values
// are JSON encoded so remote providers can hold them, and every instance that names the
// same remote provider shares the same state.
//
// Takes params (cacheStoreParams) which holds the cache configuration.
//
// Returns Cache[string, V] which is the created typed cache.
// Returns clock.Clock which is the clock to use (with defaults applied).
// Returns error when the cache service is nil or cache creation fails.
func newCacheStore[V any](ctx context.Context, params cacheStoreParams) (cache_domain.Cache[string, V], clock.Clock, error) {
	if params.CacheService == nil {
		return nil, nil, errors.New("cache service is required")
	}

	namespace := cmp.Or(params.Namespace, params.DefaultNamespace)

	maximumSize := params.MaximumEntries
	if maximumSize <= 0 {
		maximumSize = params.DefaultMaxSize
	}

	clk := params.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	builder := cache_domain.NewCacheBuilder[string, V](params.CacheService).
		Namespace(namespace).
		MaximumEntries(maximumSize).
		Clock(clk).
		PerEntryExpiration()
	if params.Provider == "" {
		builder = builder.FactoryBlueprint(params.Blueprint)
	} else {
		builder = builder.Provider(params.Provider).TypedEncoder(encoder_json.New[V]())
	}

	cache, err := builder.Build(ctx)
	if err != nil {
		return nil, nil, err
	}

	return cache, clk, nil
}
