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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/clock"
)

func createCounterStore(t *testing.T, clk clock.Clock) *CacheCounterStore {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *counterEntry]{
		MaximumSize: 1000,
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	return &CacheCounterStore{
		clock: clk,
		cache: cache,
	}
}

func TestCacheCounterStore_FirstIncrement(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	result, err := store.IncrementAndGet(context.Background(), "test-key", 1, time.Minute)

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Count)
	assert.Equal(t, mockClock.Now(), result.WindowStart)
}

func TestCacheCounterStore_MultipleIncrements(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	first, err := store.IncrementAndGet(context.Background(), "test-key", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), first.Count)
	windowStart := first.WindowStart

	for i := int64(2); i <= 5; i++ {
		mockClock.Advance(time.Second)
		result, err := store.IncrementAndGet(context.Background(), "test-key", 1, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, i, result.Count)
		assert.Equal(t, windowStart, result.WindowStart, "window start should not change")
	}
}

func TestCacheCounterStore_DifferentKeys(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	for range 3 {
		_, _ = store.IncrementAndGet(context.Background(), "key-a", 1, time.Minute)
	}
	for range 5 {
		_, _ = store.IncrementAndGet(context.Background(), "key-b", 1, time.Minute)
	}

	resultA, err := store.IncrementAndGet(context.Background(), "key-a", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(4), resultA.Count)

	resultB, err := store.IncrementAndGet(context.Background(), "key-b", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(6), resultB.Count)
}

func TestCacheCounterStore_CustomDelta(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	result, err := store.IncrementAndGet(context.Background(), "test-key", 5, time.Minute)

	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Count)

	result, err = store.IncrementAndGet(context.Background(), "test-key", 3, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(8), result.Count)
}

func TestCacheCounterStore_Concurrent(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	const numGoroutines = 50
	const incrementsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range incrementsPerGoroutine {
				_, err := store.IncrementAndGet(context.Background(), "concurrent-key", 1, time.Minute)
				if err != nil {
					t.Errorf("concurrent increment failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	finalResult, err := store.IncrementAndGet(context.Background(), "concurrent-key", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*incrementsPerGoroutine+1), finalResult.Count)
}

func TestCacheCounterStore_RateLimitScenario(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	const limit = 10

	for i := 1; i <= limit+5; i++ {
		result, err := store.IncrementAndGet(context.Background(), "rate:192.168.1.1", 1, time.Minute)
		require.NoError(t, err)

		if i <= limit {
			assert.LessOrEqual(t, result.Count, int64(limit), "request %d should be within limit", i)
		} else {
			assert.Greater(t, result.Count, int64(limit), "request %d should exceed limit", i)
		}
	}
}

func TestCacheCounterStore_WindowStartPreserved(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	first, err := store.IncrementAndGet(context.Background(), "test-key", 1, time.Minute)
	require.NoError(t, err)
	windowStart := first.WindowStart

	mockClock.Advance(30 * time.Second)

	second, err := store.IncrementAndGet(context.Background(), "test-key", 1, time.Minute)
	require.NoError(t, err)

	assert.Equal(t, windowStart, second.WindowStart, "window start should be preserved across increments")
	assert.Equal(t, int64(2), second.Count)
}

func TestNewCacheCounterStore_NilCacheService(t *testing.T) {
	t.Parallel()

	store, err := NewCacheCounterStore(context.Background(), CacheCounterStoreConfig{})

	assert.Nil(t, store)
	assert.EqualError(t, err, "creating counter cache: cache service is required")
}

func TestCacheCounterStore_Close(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	store.Close()
}
