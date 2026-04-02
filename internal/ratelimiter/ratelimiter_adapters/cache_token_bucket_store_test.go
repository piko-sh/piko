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
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

func createTokenBucketStore(t *testing.T, clk clock.Clock) *CacheTokenBucketStore {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *ratelimiter_domain.TokenBucketState]{
		MaximumSize: 1000,
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	return &CacheTokenBucketStore{
		clock: clk,
		cache: cache,
	}
}

func TestCacheTokenBucketStore_TryTake_Allowed(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	allowed, err := store.TryTake(context.Background(), "test", 1.0, config)

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestCacheTokenBucketStore_TryTake_Exhausted(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 2.0, Burst: 2}

	allowed, _ := store.TryTake(context.Background(), "test", 1.0, config)
	assert.True(t, allowed)
	allowed, _ = store.TryTake(context.Background(), "test", 1.0, config)
	assert.True(t, allowed)

	allowed, err := store.TryTake(context.Background(), "test", 1.0, config)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCacheTokenBucketStore_TryTake_RefillsOverTime(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	allowed, _ := store.TryTake(context.Background(), "test", 1.0, config)
	assert.True(t, allowed)

	allowed, _ = store.TryTake(context.Background(), "test", 1.0, config)
	assert.False(t, allowed)

	mockClock.Advance(time.Second)

	allowed, err := store.TryTake(context.Background(), "test", 1.0, config)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestCacheTokenBucketStore_TryTake_NilConfig(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	allowed, err := store.TryTake(context.Background(), "test", 1.0, nil)

	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestCacheTokenBucketStore_WaitDuration_TokensAvailable(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

	wait, err := store.WaitDuration(context.Background(), "test", 1.0, config)

	require.NoError(t, err)
	assert.Zero(t, wait)
}

func TestCacheTokenBucketStore_WaitDuration_TokensNeeded(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	_, _ = store.TryTake(context.Background(), "test", 1.0, config)

	wait, err := store.WaitDuration(context.Background(), "test", 1.0, config)

	require.NoError(t, err)
	assert.Greater(t, wait, time.Duration(0))
}

func TestCacheTokenBucketStore_WaitDuration_NilConfig(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	_, err := store.WaitDuration(context.Background(), "test", 1.0, nil)
	assert.Error(t, err)
}

func TestCacheTokenBucketStore_DeleteBucket(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	_, _ = store.TryTake(context.Background(), "test", 1.0, config)

	err := store.DeleteBucket(context.Background(), "test")
	require.NoError(t, err)

	allowed, _ := store.TryTake(context.Background(), "test", 1.0, config)
	assert.True(t, allowed)
}

func TestCacheTokenBucketStore_Concurrent(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1000.0, Burst: 1000}

	const numGoroutines = 50
	const takesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range takesPerGoroutine {
				_, err := store.TryTake(context.Background(), "concurrent", 1.0, config)
				if err != nil {
					t.Errorf("concurrent TryTake failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()
}

func TestCacheTokenBucketStore_DifferentKeys(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	allowed, _ := store.TryTake(context.Background(), "key-a", 1.0, config)
	assert.True(t, allowed)

	allowed, _ = store.TryTake(context.Background(), "key-b", 1.0, config)
	assert.True(t, allowed)

	allowed, _ = store.TryTake(context.Background(), "key-a", 1.0, config)
	assert.False(t, allowed)
}

func TestNewCacheTokenBucketStore_NilCacheService(t *testing.T) {
	t.Parallel()

	store, err := NewCacheTokenBucketStore(context.Background(), CacheTokenBucketStoreConfig{})

	assert.Nil(t, store)
	assert.EqualError(t, err, "creating token bucket cache: cache service is required")
}

func TestCacheTokenBucketStore_Close(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	store.Close()
}
