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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	windowTestKey      = "ratelimit:global:203.0.113.7"
	windowTestLimit    = 5
	windowTestWindow   = time.Minute
	sharedProviderName = "shared"
)

func createCounterStore(t *testing.T, mockClock clock.Clock) *CacheCounterStore {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *counterEntry]{
		MaximumEntries: 1000,
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	return &CacheCounterStore{
		clock: mockClock,
		cache: cache,
	}
}

func TestCacheCounterStore_FirstIncrement(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	result, err := store.IncrementAndGet(t.Context(), "test-key", 1, time.Minute)

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Count)
	assert.Equal(t, mockClock.Now(), result.WindowStart)
}

func TestCacheCounterStore_MultipleIncrements(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	first, err := store.IncrementAndGet(t.Context(), "test-key", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), first.Count)
	windowStart := first.WindowStart

	for i := int64(2); i <= 5; i++ {
		mockClock.Advance(time.Second)
		result, err := store.IncrementAndGet(t.Context(), "test-key", 1, time.Minute)
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
		_, _ = store.IncrementAndGet(t.Context(), "key-a", 1, time.Minute)
	}
	for range 5 {
		_, _ = store.IncrementAndGet(t.Context(), "key-b", 1, time.Minute)
	}

	resultA, err := store.IncrementAndGet(t.Context(), "key-a", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(4), resultA.Count)

	resultB, err := store.IncrementAndGet(t.Context(), "key-b", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(6), resultB.Count)
}

func TestCacheCounterStore_CustomDelta(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	result, err := store.IncrementAndGet(t.Context(), "test-key", 5, time.Minute)

	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Count)

	result, err = store.IncrementAndGet(t.Context(), "test-key", 3, time.Minute)
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

	for range numGoroutines {
		wg.Go(func() {
			for range incrementsPerGoroutine {
				_, err := store.IncrementAndGet(t.Context(), "concurrent-key", 1, time.Minute)
				if err != nil {
					t.Errorf("concurrent increment failed: %v", err)
				}
			}
		})
	}

	wg.Wait()

	finalResult, err := store.IncrementAndGet(t.Context(), "concurrent-key", 1, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*incrementsPerGoroutine+1), finalResult.Count)
}

func TestCacheCounterStore_RateLimitScenario(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	const limit = 10

	for i := 1; i <= limit+5; i++ {
		result, err := store.IncrementAndGet(t.Context(), "rate:192.168.1.1", 1, time.Minute)
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

	first, err := store.IncrementAndGet(t.Context(), "test-key", 1, time.Minute)
	require.NoError(t, err)
	windowStart := first.WindowStart

	mockClock.Advance(30 * time.Second)

	second, err := store.IncrementAndGet(t.Context(), "test-key", 1, time.Minute)
	require.NoError(t, err)

	assert.Equal(t, windowStart, second.WindowStart, "window start should be preserved across increments")
	assert.Equal(t, int64(2), second.Count)
}

func TestNewCacheCounterStore_NilCacheService(t *testing.T) {
	t.Parallel()

	store, err := NewCacheCounterStore(t.Context(), CacheCounterStoreConfig{})

	assert.Nil(t, store)
	assert.EqualError(t, err, "creating counter cache: cache service is required")
}

func TestCacheCounterStore_Close(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := createCounterStore(t, mockClock)

	require.NoError(t, store.Close(t.Context()))
}

func TestCacheCounterStore_ClosePropagatesCacheError(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")
	store := &CacheCounterStore{
		clock: clock.RealClock(),
		cache: &closeFailingCache[*counterEntry]{err: closeErr},
	}

	require.ErrorIs(t, store.Close(t.Context()), closeErr)
}

func TestCacheCounterStore_WindowRollover(t *testing.T) {
	t.Parallel()

	stores := []struct {
		newStore counterStoreFactory
		name     string
	}{
		{
			name:     "store from NewCacheCounterStore",
			newStore: newProductionCounterStore,
		},
		{
			name:     "store over an Otter cache without expiry support",
			newStore: createCounterStore,
		},
	}

	testCases := []struct {
		name            string
		advance         time.Duration
		wantCount       int64
		wantFreshWindow bool
	}{
		{
			name:            "increment inside the window keeps counting",
			advance:         windowTestWindow - time.Second,
			wantCount:       windowTestLimit + 1,
			wantFreshWindow: false,
		},
		{
			name:            "increment at the end of the window starts a new one",
			advance:         windowTestWindow,
			wantCount:       1,
			wantFreshWindow: true,
		},
		{
			name:            "increment long after the window starts a new one",
			advance:         24 * time.Hour,
			wantCount:       1,
			wantFreshWindow: true,
		},
	}

	for _, store := range stores {
		for _, testCase := range testCases {
			t.Run(store.name+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()

				ctx := t.Context()
				mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
				counters := store.newStore(t, mockClock)
				firstWindowStart := mockClock.Now()

				for range windowTestLimit {
					_, err := counters.IncrementAndGet(ctx, windowTestKey, 1, windowTestWindow)
					require.NoError(t, err)
				}
				mockClock.Advance(testCase.advance)

				result, err := counters.IncrementAndGet(ctx, windowTestKey, 1, windowTestWindow)
				require.NoError(t, err)

				wantWindowStart := firstWindowStart
				if testCase.wantFreshWindow {
					wantWindowStart = mockClock.Now()
				}
				assert.Equal(t, testCase.wantCount, result.Count)
				assert.Equal(t, wantWindowStart, result.WindowStart)
			})
		}
	}
}

func TestCacheCounterStore_ExpiresIdleCounters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		advance     time.Duration
		wantPresent bool
	}{
		{
			name:        "counter is kept inside its window",
			advance:     windowTestWindow - time.Second,
			wantPresent: true,
		},
		{
			name:        "counter is dropped once its window ends",
			advance:     windowTestWindow,
			wantPresent: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
			counters := newProductionCounterStore(t, mockClock)

			for range windowTestLimit {
				_, err := counters.IncrementAndGet(ctx, windowTestKey, 1, windowTestWindow)
				require.NoError(t, err)
			}
			mockClock.Advance(testCase.advance)

			_, present, err := counters.cache.GetIfPresent(ctx, windowTestKey)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantPresent, present)
		})
	}
}

func TestCacheCounterStore_UsesConfiguredProvider(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider := provider_mock.NewMockProvider()
	service := newServiceWithProvider(t, sharedProviderName, provider)

	store, err := NewCacheCounterStore(ctx, CacheCounterStoreConfig{
		CacheService:       service,
		Clock:              clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
		Provider:           sharedProviderName,
		Namespace:          "",
		MaximumEntries:     0,
		ClockSkewTolerance: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	for range 3 {
		_, err := store.IncrementAndGet(ctx, windowTestKey, 1, windowTestWindow)
		require.NoError(t, err)
	}

	stored := readProviderValue[counterEntry](t, provider, defaultCounterNamespace, windowTestKey)
	assert.Equal(t, int64(3), stored.Count, "the named provider holds the counter")
}

func TestCacheCounterStore_ClockSkewTolerance(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		provider      string
		tolerance     time.Duration
		advance       time.Duration
		wantCount     int64
		wantNewWindow bool
	}{
		{
			name:          "local store starts a new window at the end of the window",
			provider:      "",
			tolerance:     0,
			advance:       windowTestWindow,
			wantCount:     1,
			wantNewWindow: true,
		},
		{
			name:          "remote store keeps counting inside the default tolerance",
			provider:      sharedProviderName,
			tolerance:     0,
			advance:       windowTestWindow + defaultClockSkewTolerance - time.Millisecond,
			wantCount:     windowTestLimit + 1,
			wantNewWindow: false,
		},
		{
			name:          "remote store starts a new window once the tolerance has passed",
			provider:      sharedProviderName,
			tolerance:     0,
			advance:       windowTestWindow + defaultClockSkewTolerance,
			wantCount:     1,
			wantNewWindow: true,
		},
		{
			name:          "explicit tolerance replaces the default",
			provider:      sharedProviderName,
			tolerance:     5 * time.Second,
			advance:       windowTestWindow + 4*time.Second,
			wantCount:     windowTestLimit + 1,
			wantNewWindow: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
			store, err := NewCacheCounterStore(ctx, CacheCounterStoreConfig{
				CacheService:       newServiceWithProvider(t, sharedProviderName, provider_mock.NewMockProvider()),
				Clock:              mockClock,
				Provider:           testCase.provider,
				Namespace:          "",
				MaximumEntries:     0,
				ClockSkewTolerance: testCase.tolerance,
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = store.Close(context.Background()) })
			firstWindowStart := mockClock.Now()

			for range windowTestLimit {
				_, err := store.IncrementAndGet(ctx, windowTestKey, 1, windowTestWindow)
				require.NoError(t, err)
			}
			mockClock.Advance(testCase.advance)

			result, err := store.IncrementAndGet(ctx, windowTestKey, 1, windowTestWindow)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantCount, result.Count)
			if testCase.wantNewWindow {
				assert.Equal(t, mockClock.Now(), result.WindowStart)
			} else {
				assert.Equal(t, firstWindowStart, result.WindowStart)
			}
		})
	}
}

func TestNewCacheCounterStore_RejectsNegativeClockSkewTolerance(t *testing.T) {
	t.Parallel()

	store, err := NewCacheCounterStore(t.Context(), CacheCounterStoreConfig{
		CacheService:       cache_domain.NewService(""),
		Clock:              nil,
		Provider:           "",
		Namespace:          "",
		MaximumEntries:     0,
		ClockSkewTolerance: -time.Second,
	})

	assert.Nil(t, store)
	require.ErrorIs(t, err, ratelimiter_domain.ErrInvalidConfig)
}

func TestNewCacheCounterStore_UnknownProvider(t *testing.T) {
	t.Parallel()

	store, err := NewCacheCounterStore(t.Context(), CacheCounterStoreConfig{
		CacheService:       cache_domain.NewService(""),
		Clock:              nil,
		Provider:           "missing",
		Namespace:          "",
		MaximumEntries:     0,
		ClockSkewTolerance: 0,
	})

	assert.Nil(t, store)
	require.ErrorContains(t, err, "creating counter cache")
}

type counterStoreFactory func(t *testing.T, mockClock clock.Clock) *CacheCounterStore

type closeFailingCache[V any] struct {
	cache_domain.Cache[string, V]
	err error
}

func (c *closeFailingCache[V]) Close(context.Context) error {
	return c.err
}

func newProductionCounterStore(t *testing.T, mockClock clock.Clock) *CacheCounterStore {
	t.Helper()

	store, err := NewCacheCounterStore(t.Context(), CacheCounterStoreConfig{
		CacheService:       cache_domain.NewService(""),
		Clock:              mockClock,
		Provider:           "",
		Namespace:          "",
		MaximumEntries:     0,
		ClockSkewTolerance: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close(context.Background())) })

	return store
}

func newServiceWithProvider(t *testing.T, name string, provider *provider_mock.MockProvider) cache_domain.Service {
	t.Helper()

	service := cache_domain.NewService(name)
	require.NoError(t, service.RegisterProvider(t.Context(), name, provider))
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	return service
}

func readProviderValue[T any](t *testing.T, provider *provider_mock.MockProvider, namespace, key string) T {
	t.Helper()

	cacheAny, err := provider.CreateNamespaceTyped(namespace, cache_dto.Options[string, []byte]{})
	require.NoError(t, err)
	byteCache, ok := cacheAny.(cache_domain.Cache[string, []byte])
	require.True(t, ok, "the provider holds a byte cache for the namespace")

	encoded, found, err := byteCache.GetIfPresent(t.Context(), key)
	require.NoError(t, err)
	require.True(t, found, "the provider holds the key")

	var value T
	require.NoError(t, cache_domain.CacheAPI.Unmarshal(encoded, &value))
	return value
}
