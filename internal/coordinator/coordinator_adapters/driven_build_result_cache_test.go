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

package coordinator_adapters_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/coordinator/coordinator_adapters"
	"piko.sh/piko/internal/coordinator/coordinator_domain"
)

func newTestCacheService() cache_domain.Service {
	return cache_domain.NewService("")
}

func TestMemoryCache_EmptyCacheMiss(t *testing.T) {
	cache, cacheErr := coordinator_adapters.NewBuildResultCache(context.Background(), newTestCacheService())
	require.NoError(t, cacheErr)

	cached, err := cache.Get(context.Background(), "some-key")
	assert.Nil(t, cached)
	assert.ErrorIs(t, err, coordinator_domain.ErrCacheMiss)
}

func TestMemoryCache_SetThenGetHit(t *testing.T) {
	cache, cacheErr := coordinator_adapters.NewBuildResultCache(context.Background(), newTestCacheService())
	require.NoError(t, cacheErr)
	ctx := context.Background()

	key := "hash-123"
	stored := &annotator_dto.ProjectAnnotationResult{}
	require.NoError(t, cache.Set(ctx, key, stored))

	cached, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Same(t, stored, cached)
}

func TestMemoryCache_KeyMismatchIsMiss(t *testing.T) {
	cache, cacheErr := coordinator_adapters.NewBuildResultCache(context.Background(), newTestCacheService())
	require.NoError(t, cacheErr)
	ctx := context.Background()

	stored := &annotator_dto.ProjectAnnotationResult{}
	require.NoError(t, cache.Set(ctx, "key-A", stored))

	cached, err := cache.Get(ctx, "key-B")
	assert.Nil(t, cached)
	assert.ErrorIs(t, err, coordinator_domain.ErrCacheMiss)
}

func TestMemoryCache_ClearResetsCache(t *testing.T) {
	cache, cacheErr := coordinator_adapters.NewBuildResultCache(context.Background(), newTestCacheService())
	require.NoError(t, cacheErr)
	ctx := context.Background()

	key := "hash-abc"
	stored := &annotator_dto.ProjectAnnotationResult{}
	require.NoError(t, cache.Set(ctx, key, stored))

	cached, err := cache.Get(ctx, key)
	require.NoError(t, err)
	assert.Same(t, stored, cached)

	require.NoError(t, cache.Clear(ctx))

	res2, err2 := cache.Get(ctx, key)
	assert.Nil(t, res2)
	assert.ErrorIs(t, err2, coordinator_domain.ErrCacheMiss)
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache, cacheErr := coordinator_adapters.NewBuildResultCache(context.Background(), newTestCacheService())
	require.NoError(t, cacheErr)
	ctx := context.Background()
	key := "concurrent-key"

	const readers = 50
	var wg sync.WaitGroup
	wg.Add(readers)
	missCount := 0
	var mu sync.Mutex
	for range readers {
		go func() {
			defer wg.Done()
			if _, err := cache.Get(ctx, key); err != nil {
				mu.Lock()
				if assert.ErrorIs(t, err, coordinator_domain.ErrCacheMiss) {
					missCount++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, readers, missCount, "all reads before Set should be cache misses")

	stored := &annotator_dto.ProjectAnnotationResult{}
	require.NoError(t, cache.Set(ctx, key, stored))

	wg.Add(readers)
	hitCount := 0
	for range readers {
		go func() {
			defer wg.Done()
			cached, err := cache.Get(ctx, key)
			if assert.NoError(t, err) {
				assert.Same(t, stored, cached)
				mu.Lock()
				hitCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, readers, hitCount, "all reads after Set should be cache hits")
}
