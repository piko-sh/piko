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

package coordinator_adapters

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultIntrospectionCacheSize is the number of introspection cache entries
	// to keep.
	defaultIntrospectionCacheSize = 5
)

// introspectionCache implements IntrospectionCachePort using the cache hexagon.
// It is the first tier of a two-tier caching system.
//
// This cache stores Phase 1 annotation pipeline results (buildUnifiedGraph,
// virtualiseModule, initialiseTypeResolver). These operations are costly
// because they call packages.Load() for full Go type introspection.
//
// Phase 1 only depends on <script> blocks from .pk files and all .go files.
// When only <template>, <style>, or <i18n> blocks change, cached data can be
// reused, skipping to Phase 2 for better performance.
type introspectionCache struct {
	// cache is the underlying cache instance from the cache hexagon.
	cache cache_domain.Cache[string, *coordinator_domain.IntrospectionCacheEntry]
}

var _ coordinator_domain.IntrospectionCachePort = (*introspectionCache)(nil)

// Get retrieves an introspection cache entry by key. If found, it validates
// the entry before returning.
//
// Takes key (string) which identifies the cache entry to retrieve.
//
// Returns *coordinator_domain.IntrospectionCacheEntry which contains the cached
// introspection data if valid.
// Returns error when the key is not found or the cached entry is invalid.
func (c *introspectionCache) Get(ctx context.Context, key string) (*coordinator_domain.IntrospectionCacheEntry, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "IntrospectionCache.Get",
		logger_domain.String(logKeyKey, key),
	)
	defer span.End()

	cacheEntry, found, err := c.cache.GetIfPresent(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving introspection cache entry for %q: %w", key, err)
	}

	if found {
		l.Trace("Introspection cache HIT.", logger_domain.String(logKeyKey, key))
		span.SetAttributes(attribute.String("cache.status", "HIT"))
		span.SetStatus(codes.Ok, "Introspection cache hit")

		if !cacheEntry.IsValid() {
			l.Warn("Introspection cache entry is invalid (version mismatch or corrupted), treating as cache miss.",
				logger_domain.String(logKeyKey, key),
				logger_domain.Int("cached_version", cacheEntry.Version),
				logger_domain.Int("current_version", coordinator_domain.CurrentIntrospectionCacheVersion))
			span.SetAttributes(attribute.String("cache.status", "INVALID"))

			if invalidateErr := c.cache.Invalidate(ctx, key); invalidateErr != nil {
				l.Warn("Failed to invalidate stale introspection cache entry",
					logger_domain.String(logKeyKey, key),
					logger_domain.Error(invalidateErr))
			}
			return nil, coordinator_domain.ErrCacheMiss
		}

		return cacheEntry, nil
	}

	l.Trace("Introspection cache MISS.", logger_domain.String(logKeyKey, key))
	span.SetAttributes(attribute.String("cache.status", "MISS"))
	span.SetStatus(codes.Ok, "Introspection cache miss")
	return nil, coordinator_domain.ErrCacheMiss
}

// Set stores an introspection cache entry with the given key.
//
// If the key already exists, its value is updated. If the cache is at capacity,
// the least recently used item is evicted before adding the new entry.
//
// Takes key (string) which identifies the cache entry.
// Takes entry (*coordinator_domain.IntrospectionCacheEntry) which is the
// value to store.
//
// Returns error when the entry is invalid.
func (c *introspectionCache) Set(ctx context.Context, key string, entry *coordinator_domain.IntrospectionCacheEntry) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "IntrospectionCache.Set",
		logger_domain.String(logKeyKey, key),
	)
	defer span.End()

	if !entry.IsValid() {
		l.Error("Attempted to cache invalid introspection entry, rejecting.",
			logger_domain.String(logKeyKey, key))
		span.SetStatus(codes.Error, "Invalid entry rejected")
		return coordinator_domain.ErrInvalidCacheEntry
	}

	if setErr := c.cache.Set(ctx, key, entry); setErr != nil {
		l.Warn("Failed to store introspection cache entry",
			logger_domain.String(logKeyKey, key),
			logger_domain.Error(setErr))
	}

	l.Trace("Introspection cache entry stored.",
		logger_domain.String(logKeyKey, key),
		logger_domain.Int("current_size", c.cache.EstimatedSize()))
	span.SetStatus(codes.Ok, "Introspection cache set successfully")
	return nil
}

// Clear removes all entries from the introspection cache.
//
// Returns error when the cache cannot be cleared.
func (c *introspectionCache) Clear(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "IntrospectionCache.Clear")
	defer span.End()

	if invalidateErr := c.cache.InvalidateAll(ctx); invalidateErr != nil {
		l.Warn("Failed to invalidate all introspection cache entries",
			logger_domain.Error(invalidateErr))
	}

	l.Internal("Introspection cache cleared successfully.")
	span.SetStatus(codes.Ok, "Introspection cache cleared")
	return nil
}

// Close releases resources held by the cache.
func (c *introspectionCache) Close() {
	ctx := context.Background()
	if closeErr := c.cache.Close(ctx); closeErr != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to close introspection cache",
			logger_domain.Error(closeErr))
	}
}

// NewIntrospectionCache creates a new cache for storing introspection results,
// backed by the cache hexagon.
//
// Takes cacheService (cache_domain.Service) which provides the cache
// infrastructure.
//
// Returns coordinator_domain.IntrospectionCachePort which is the cache ready
// for use.
// Returns error when the cache cannot be created.
func NewIntrospectionCache(ctx context.Context, cacheService cache_domain.Service) (coordinator_domain.IntrospectionCachePort, error) {
	c, err := cache_domain.NewCacheBuilder[string, *coordinator_domain.IntrospectionCacheEntry](cacheService).
		FactoryBlueprint(BlueprintIntrospection).
		Namespace("coordinator-introspection").
		MaximumSize(defaultIntrospectionCacheSize).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("building introspection cache: %w", err)
	}
	return &introspectionCache{cache: c}, nil
}
