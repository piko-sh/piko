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
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// defaultCacheSize is the number of recent build results to cache.
	defaultCacheSize = 5
)

// buildResultCache implements BuildResultCachePort using the cache hexagon. It
// provides fast access to recent builds, useful for development servers where
// it can offer a quick undo by caching file states from moments before.
type buildResultCache struct {
	// cache is the underlying cache instance from the cache hexagon.
	cache cache_domain.Cache[string, *annotator_dto.ProjectAnnotationResult]
}

var _ coordinator_domain.BuildResultCachePort = (*buildResultCache)(nil)

// Get retrieves a build result from the cache by its key.
//
// Takes key (string) which identifies the cached build result.
//
// Returns *annotator_dto.ProjectAnnotationResult which is the cached result.
// Returns error when the key is not found in the cache.
func (c *buildResultCache) Get(ctx context.Context, key string) (*annotator_dto.ProjectAnnotationResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "BuildResultCache.Get",
		logger_domain.String(logKeyKey, key),
	)
	defer span.End()

	result, found, err := c.cache.GetIfPresent(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("retrieving build result cache entry for %q: %w", key, err)
	}

	if found {
		l.Trace("Cache HIT.", logger_domain.String(logKeyKey, key))
		span.SetAttributes(attribute.String("cache.status", "HIT"))
		span.SetStatus(codes.Ok, "Cache hit")
		return result, nil
	}

	l.Trace("Cache MISS.", logger_domain.String(logKeyKey, key))
	span.SetAttributes(attribute.String("cache.status", "MISS"))
	span.SetStatus(codes.Ok, "Cache miss")
	return nil, coordinator_domain.ErrCacheMiss
}

// Set stores a build result in the cache with the given key.
//
// If the key already exists, its value is updated. If the cache is at capacity,
// the least recently used item is evicted before adding the new entry.
//
// Takes key (string) which identifies the cache entry.
// Takes result (*annotator_dto.ProjectAnnotationResult) which is the value
// to store.
//
// Returns error when the operation fails.
func (c *buildResultCache) Set(ctx context.Context, key string, result *annotator_dto.ProjectAnnotationResult) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "BuildResultCache.Set",
		logger_domain.String(logKeyKey, key),
	)
	defer span.End()

	_ = c.cache.Set(ctx, key, result)

	l.Trace("Cache entry stored.",
		logger_domain.String(logKeyKey, key),
		logger_domain.Int("current_size", c.cache.EstimatedSize()))
	span.SetStatus(codes.Ok, "Cache set successfully")
	return nil
}

// Clear removes all entries from the cache.
//
// Returns error when the operation fails.
func (c *buildResultCache) Clear(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "BuildResultCache.Clear")
	defer span.End()

	_ = c.cache.InvalidateAll(ctx)

	l.Internal("Build result cache cleared successfully.")
	span.SetStatus(codes.Ok, "Cache cleared")
	return nil
}

// Close releases resources held by the cache.
func (c *buildResultCache) Close() {
	_ = c.cache.Close(context.Background())
}

// NewBuildResultCache creates a new cache for build results, backed by the
// cache hexagon.
//
// Takes cacheService (cache_domain.Service) which provides the
// cache infrastructure.
//
// Returns coordinator_domain.BuildResultCachePort which provides build result
// caching with LRU eviction.
// Returns error when the cache cannot be created.
func NewBuildResultCache(ctx context.Context, cacheService cache_domain.Service) (coordinator_domain.BuildResultCachePort, error) {
	c, err := cache_domain.NewCacheBuilder[string, *annotator_dto.ProjectAnnotationResult](cacheService).
		FactoryBlueprint(BlueprintBuildResults).
		Namespace("coordinator-build-results").
		MaximumSize(defaultCacheSize).
		Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("building build result cache: %w", err)
	}
	return &buildResultCache{cache: c}, nil
}
