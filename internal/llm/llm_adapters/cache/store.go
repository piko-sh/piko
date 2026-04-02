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

package cache

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

const (
	// DefaultCacheMaximumSize is the default maximum number of entries for
	// in-memory cache providers.
	DefaultCacheMaximumSize = 10000
)

// Config configures the unified cache store.
type Config struct {
	// CacheService is the cache service used for storing entries; must not be nil.
	CacheService cache_domain.Service

	// Clock provides time operations. If nil, defaults to RealClock.
	Clock clock.Clock

	// Provider specifies the cache provider (e.g., "otter", "redis",
	// "redis-cluster"). If empty, uses the service's default provider.
	Provider string

	// Namespace is the key prefix for all cache entries (e.g. "llm:cache").
	Namespace string

	// MaximumSize is the maximum number of entries for in-memory providers.
	// Remote providers ignore this value.
	MaximumSize int
}

// Store provides an LLM cache using the internal cache service.
// It implements llm_domain.CacheStorePort and io.Closer.
type Store struct {
	// cache stores LLM response entries for fast retrieval.
	cache cache_domain.Cache[string, *llm_dto.CacheEntry]

	// clock provides the current time for expiry checks and timestamps.
	clock clock.Clock

	// service provides domain-level caching operations.
	service cache_domain.Service

	// hits counts the number of successful cache lookups.
	hits atomic.Int64

	// misses counts cache lookups that found no valid entry.
	misses atomic.Int64
}

var _ llm_domain.CacheStorePort = (*Store)(nil)

// StoreOption configures optional Store dependencies.
type StoreOption func(*Store)

// WithCache injects a pre-built cache, bypassing the default cache builder.
// Intended for testing.
//
// Takes cache (cache_domain.Cache[string, *llm_dto.CacheEntry]) which is the
// cache to use.
//
// Returns StoreOption which applies this setting to the store.
func WithCache(cache cache_domain.Cache[string, *llm_dto.CacheEntry]) StoreOption {
	return func(s *Store) {
		s.cache = cache
	}
}

// Get retrieves a cache entry by key.
//
// Takes key (string) which identifies the cache entry to retrieve.
//
// Returns *llm_dto.CacheEntry which is the cached entry, or nil if not found.
// Returns error when the retrieval fails.
func (s *Store) Get(ctx context.Context, key string) (*llm_dto.CacheEntry, error) {
	entry, found, err := s.cache.GetIfPresent(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving cache entry for key %q: %w", key, err)
	}
	if !found {
		s.misses.Add(1)
		return nil, nil
	}

	if entry != nil && entry.IsExpiredAt(s.clock.Now()) {
		_ = s.cache.Invalidate(ctx, key)
		s.misses.Add(1)
		return nil, nil
	}

	s.hits.Add(1)
	return entry, nil
}

// Set stores a cache entry.
//
// Takes key (string) which identifies the cache entry.
// Takes entry (*llm_dto.CacheEntry) which contains the data to store.
//
// Returns error when the underlying cache operation fails.
func (s *Store) Set(ctx context.Context, key string, entry *llm_dto.CacheEntry) error {
	if entry == nil {
		return nil
	}

	ttl := entry.ExpiresAt.Sub(entry.CreatedAt)
	if ttl <= 0 {
		return s.cache.Set(ctx, key, entry)
	}

	return s.cache.SetWithTTL(ctx, key, entry, ttl)
}

// Delete removes a cache entry by key.
//
// Takes key (string) which identifies the entry to remove.
//
// Returns error when the deletion fails.
func (s *Store) Delete(ctx context.Context, key string) error {
	return s.cache.Invalidate(ctx, key)
}

// Clear removes all cache entries.
//
// Returns error when the operation fails.
func (s *Store) Clear(ctx context.Context) error {
	return s.cache.InvalidateAll(ctx)
}

// GetStats returns cache statistics.
//
// Returns *llm_dto.CacheStats which contains the current hit and miss counts,
// cache size, and estimated cost saved.
// Returns error which is always nil.
func (s *Store) GetStats(_ context.Context) (*llm_dto.CacheStats, error) {
	return &llm_dto.CacheStats{
		Hits:               s.hits.Load(),
		Misses:             s.misses.Load(),
		Size:               int64(s.cache.EstimatedSize()),
		EstimatedCostSaved: maths.ZeroMoney(llm_dto.CostCurrency),
	}, nil
}

// Close releases resources held by the store.
func (s *Store) Close() {
	_ = s.cache.Close(context.Background())
}

// New creates a new unified cache store backed by internal/cache.
//
// Takes ctx (context.Context) which controls cancellation during cache setup.
// Takes config (Config) which configures the store.
// Takes options (...StoreOption) which are optional functions to configure the
// store, such as WithCache for injecting a test cache.
//
// Returns *Store which is ready for use.
// Returns error when cache creation fails.
func New(ctx context.Context, config Config, options ...StoreOption) (*Store, error) {
	clk := config.Clock
	if clk == nil {
		clk = clock.RealClock()
	}

	store := &Store{
		clock:   clk,
		service: config.CacheService,
	}

	for _, option := range options {
		option(store)
	}

	if store.cache == nil {
		if config.CacheService == nil {
			return nil, errors.New("cache service is required")
		}

		namespace := cmp.Or(config.Namespace, "llm:cache")

		maximumSize := config.MaximumSize
		if maximumSize <= 0 {
			maximumSize = DefaultCacheMaximumSize
		}

		cache, err := cache_domain.NewCacheBuilder[string, *llm_dto.CacheEntry](config.CacheService).
			FactoryBlueprint(FactoryBlueprintName).
			Namespace(namespace).
			MaximumSize(maximumSize).
			Build(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}

		store.cache = cache
	}

	return store, nil
}
