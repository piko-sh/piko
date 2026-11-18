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
	"math"
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
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	bucketTestKey = "email:provider"
)

func createTokenBucketStore(t *testing.T, mockClock clock.Clock) *CacheTokenBucketStore {
	t.Helper()

	cache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *ratelimiter_domain.TokenBucketState]{
		MaximumEntries: 1000,
	})
	require.NoError(t, err)

	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	return &CacheTokenBucketStore{
		clock: mockClock,
		cache: cache,
	}
}

func TestCacheTokenBucketStore_TryTake_Allowed(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}
	allowed, err := store.TryTake(t.Context(), "test", 1.0, config)

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestCacheTokenBucketStore_TryTake_Exhausted(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 2.0, Burst: 2}

	allowed, _ := store.TryTake(t.Context(), "test", 1.0, config)
	assert.True(t, allowed)
	allowed, _ = store.TryTake(t.Context(), "test", 1.0, config)
	assert.True(t, allowed)

	allowed, err := store.TryTake(t.Context(), "test", 1.0, config)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestCacheTokenBucketStore_TryTake_RefillsOverTime(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	allowed, _ := store.TryTake(t.Context(), "test", 1.0, config)
	assert.True(t, allowed)

	allowed, _ = store.TryTake(t.Context(), "test", 1.0, config)
	assert.False(t, allowed)

	mockClock.Advance(time.Second)

	allowed, err := store.TryTake(t.Context(), "test", 1.0, config)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestCacheTokenBucketStore_TryTake_NilConfig(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	allowed, err := store.TryTake(t.Context(), "test", 1.0, nil)

	assert.Error(t, err)
	assert.False(t, allowed)
}

func TestCacheTokenBucketStore_WaitDuration_TokensAvailable(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

	wait, err := store.WaitDuration(t.Context(), "test", 1.0, config)

	require.NoError(t, err)
	assert.Zero(t, wait)
}

func TestCacheTokenBucketStore_WaitDuration_TokensNeeded(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	_, _ = store.TryTake(t.Context(), "test", 1.0, config)

	wait, err := store.WaitDuration(t.Context(), "test", 1.0, config)

	require.NoError(t, err)
	assert.Greater(t, wait, time.Duration(0))
}

func TestCacheTokenBucketStore_WaitDuration_SlowRefillIsCapped(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1e-300, Burst: 1}

	_, _ = store.TryTake(t.Context(), "test", 1.0, config)

	wait, err := store.WaitDuration(t.Context(), "test", 1.0, config)

	require.NoError(t, err)
	assert.Equal(t, time.Hour, wait, "a wait too long for time.Duration is capped, never negative")
}

func TestCacheTokenBucketStore_WaitDuration_NilConfig(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	_, err := store.WaitDuration(t.Context(), "test", 1.0, nil)
	assert.Error(t, err)
}

func TestCacheTokenBucketStore_DeleteBucket(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	_, _ = store.TryTake(t.Context(), "test", 1.0, config)

	err := store.DeleteBucket(t.Context(), "test")
	require.NoError(t, err)

	allowed, _ := store.TryTake(t.Context(), "test", 1.0, config)
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

	for range numGoroutines {
		wg.Go(func() {
			for range takesPerGoroutine {
				_, err := store.TryTake(t.Context(), "concurrent", 1.0, config)
				if err != nil {
					t.Errorf("concurrent TryTake failed: %v", err)
				}
			}
		})
	}

	wg.Wait()
}

func TestCacheTokenBucketStore_DifferentKeys(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

	allowed, _ := store.TryTake(t.Context(), "key-a", 1.0, config)
	assert.True(t, allowed)

	allowed, _ = store.TryTake(t.Context(), "key-b", 1.0, config)
	assert.True(t, allowed)

	allowed, _ = store.TryTake(t.Context(), "key-a", 1.0, config)
	assert.False(t, allowed)
}

func TestNewCacheTokenBucketStore_NilCacheService(t *testing.T) {
	t.Parallel()

	store, err := NewCacheTokenBucketStore(t.Context(), CacheTokenBucketStoreConfig{})

	assert.Nil(t, store)
	assert.EqualError(t, err, "creating token bucket cache: cache service is required")
}

func TestCacheTokenBucketStore_Close(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Now())
	store := createTokenBucketStore(t, mockClock)

	require.NoError(t, store.Close(t.Context()))
}

func TestCacheTokenBucketStore_ClosePropagatesCacheError(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")
	store := &CacheTokenBucketStore{
		clock: clock.RealClock(),
		cache: &closeFailingCache[*ratelimiter_domain.TokenBucketState]{err: closeErr},
	}

	require.ErrorIs(t, store.Close(t.Context()), closeErr)
}

func TestCacheTokenBucketStore_UsesConfiguredProvider(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider := provider_mock.NewMockProvider()

	store, err := NewCacheTokenBucketStore(ctx, CacheTokenBucketStoreConfig{
		CacheService:   newServiceWithProvider(t, sharedProviderName, provider),
		Clock:          clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)),
		Provider:       sharedProviderName,
		Namespace:      "",
		MaximumEntries: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 5}
	for range 2 {
		allowed, err := store.TryTake(ctx, bucketTestKey, 1, config)
		require.NoError(t, err)
		require.True(t, allowed)
	}

	stored := readProviderValue[ratelimiter_domain.TokenBucketState](t, provider, defaultTokenBucketNamespace, bucketTestKey)
	assert.InDelta(t, 3.0, stored.Tokens, 0.001, "the named provider holds the bucket")
}

func TestCacheTokenBucketStore_TryTake_RetriedCallbackReportsLastAttempt(t *testing.T) {
	t.Parallel()

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 1}
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	full := ratelimiter_domain.NewBucketState(config, now.UnixNano())
	empty := &ratelimiter_domain.TokenBucketState{
		Tokens:         0,
		MaxTokens:      full.MaxTokens,
		RefillRate:     full.RefillRate,
		LastRefillNano: now.UnixNano(),
	}
	store := &CacheTokenBucketStore{
		clock: clock.NewMockClock(now),
		cache: &retryingBucketCache{attempts: []*ratelimiter_domain.TokenBucketState{full, empty}},
	}

	allowed, err := store.TryTake(t.Context(), bucketTestKey, 1, config)

	require.NoError(t, err)
	assert.False(t, allowed, "the result must come from the attempt the cache kept, not an earlier one")
}

func TestCacheTokenBucketStore_WaitDuration_RetriedCallbackReportsLastAttempt(t *testing.T) {
	t.Parallel()

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1, Burst: 1}
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	full := ratelimiter_domain.NewBucketState(config, now.UnixNano())
	empty := &ratelimiter_domain.TokenBucketState{
		Tokens:         0,
		MaxTokens:      full.MaxTokens,
		RefillRate:     full.RefillRate,
		LastRefillNano: now.UnixNano(),
	}
	store := &CacheTokenBucketStore{
		clock: clock.NewMockClock(now),
		cache: &retryingBucketCache{attempts: []*ratelimiter_domain.TokenBucketState{empty, full}},
	}

	wait, err := store.WaitDuration(t.Context(), bucketTestKey, 1, config)

	require.NoError(t, err)
	assert.Zero(t, wait, "the wait must come from the attempt the cache kept, not an earlier one")
}

func TestCacheTokenBucketStore_ExpiresRefilledBuckets(t *testing.T) {
	t.Parallel()

	config := ratelimiter_dto.TokenBucketConfig{Rate: 2, Burst: 4}
	refillTime := 2 * time.Second

	testCases := []struct {
		name        string
		advance     time.Duration
		wantPresent bool
	}{
		{
			name:        "bucket is kept while it refills",
			advance:     refillTime + tokenBucketExpiryMargin - time.Millisecond,
			wantPresent: true,
		},
		{
			name:        "bucket is dropped once it has refilled",
			advance:     refillTime + tokenBucketExpiryMargin,
			wantPresent: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
			store := newProductionTokenBucketStore(t, mockClock)

			for range config.Burst {
				allowed, err := store.TryTake(ctx, bucketTestKey, 1, &config)
				require.NoError(t, err)
				require.True(t, allowed)
			}
			mockClock.Advance(testCase.advance)

			_, present, err := store.cache.GetIfPresent(ctx, bucketTestKey)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantPresent, present)
		})
	}
}

func TestBucketTTL(t *testing.T) {
	t.Parallel()

	perSecond := 1 / float64(time.Second)

	testCases := []struct {
		state *ratelimiter_domain.TokenBucketState
		name  string
		want  time.Duration
	}{
		{
			name:  "full bucket only needs the margin",
			state: &ratelimiter_domain.TokenBucketState{Tokens: 5, MaxTokens: 5, RefillRate: perSecond},
			want:  tokenBucketExpiryMargin,
		},
		{
			name:  "empty bucket waits for a full refill",
			state: &ratelimiter_domain.TokenBucketState{Tokens: 0, MaxTokens: 5, RefillRate: perSecond},
			want:  5*time.Second + tokenBucketExpiryMargin,
		},
		{
			name:  "bucket that never refills never expires",
			state: &ratelimiter_domain.TokenBucketState{Tokens: 0, MaxTokens: 5, RefillRate: 0},
			want:  cache_dto.NoExpiry,
		},
		{
			name:  "refill time beyond a duration never expires",
			state: &ratelimiter_domain.TokenBucketState{Tokens: 0, MaxTokens: math.MaxFloat64, RefillRate: perSecond},
			want:  cache_dto.NoExpiry,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, bucketTTL(testCase.state))
		})
	}
}

type retryingBucketCache struct {
	cache_domain.Cache[string, *ratelimiter_domain.TokenBucketState]
	attempts []*ratelimiter_domain.TokenBucketState
}

func (c *retryingBucketCache) ComputeWithTTL(
	_ context.Context,
	_ string,
	computeFunction func(oldValue *ratelimiter_domain.TokenBucketState, found bool) cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState],
) (*ratelimiter_domain.TokenBucketState, bool, error) {
	var result cache_dto.ComputeResult[*ratelimiter_domain.TokenBucketState]
	for _, state := range c.attempts {
		result = computeFunction(state, true)
	}
	return result.Value, true, nil
}

func (c *retryingBucketCache) Compute(
	_ context.Context,
	_ string,
	computeFunction func(oldValue *ratelimiter_domain.TokenBucketState, found bool) (*ratelimiter_domain.TokenBucketState, cache_dto.ComputeAction),
) (*ratelimiter_domain.TokenBucketState, bool, error) {
	var value *ratelimiter_domain.TokenBucketState
	for _, state := range c.attempts {
		value, _ = computeFunction(state, true)
	}
	return value, true, nil
}

func newProductionTokenBucketStore(t *testing.T, mockClock clock.Clock) *CacheTokenBucketStore {
	t.Helper()

	store, err := NewCacheTokenBucketStore(t.Context(), CacheTokenBucketStoreConfig{
		CacheService:   cache_domain.NewService(""),
		Clock:          mockClock,
		Provider:       "",
		Namespace:      "",
		MaximumEntries: 0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close(context.Background())) })

	return store
}
