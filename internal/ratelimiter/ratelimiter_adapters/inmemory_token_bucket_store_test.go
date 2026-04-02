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

	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/clock"
)

func newTestInMemoryStore(t *testing.T) (*InMemoryTokenBucketStore, *clock.MockClock) {
	t.Helper()
	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	store := NewInMemoryTokenBucketStore(WithInMemoryClock(mockClock))
	return store, mockClock
}

func TestNewInMemoryTokenBucketStore(t *testing.T) {
	store := NewInMemoryTokenBucketStore()
	assert.NotNil(t, store)
	assert.NotNil(t, store.buckets)
	assert.NotNil(t, store.clock)
}

func TestInMemoryTokenBucketStore_TryTake_AllowedWithinBurst(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 5}

	for i := range 5 {
		allowed, err := store.TryTake(context.Background(), "key", 1, config)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed within burst", i+1)
	}
}

func TestInMemoryTokenBucketStore_TryTake_DeniedWhenExhausted(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 3}

	for range 3 {
		allowed, err := store.TryTake(context.Background(), "key", 1, config)
		require.NoError(t, err)
		require.True(t, allowed)
	}

	allowed, err := store.TryTake(context.Background(), "key", 1, config)
	require.NoError(t, err)
	assert.False(t, allowed, "should be denied after burst exhausted")
}

func TestInMemoryTokenBucketStore_TryTake_RefillsOverTime(t *testing.T) {
	store, mockClock := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 100, Burst: 1}

	allowed, err := store.TryTake(context.Background(), "key", 1, config)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = store.TryTake(context.Background(), "key", 1, config)
	require.NoError(t, err)
	require.False(t, allowed)

	mockClock.Advance(15 * time.Millisecond)

	allowed, err = store.TryTake(context.Background(), "key", 1, config)
	require.NoError(t, err)
	assert.True(t, allowed, "should be allowed after refill")
}

func TestInMemoryTokenBucketStore_TryTake_IndependentKeys(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 1}

	allowed, err := store.TryTake(context.Background(), "key1", 1, config)
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = store.TryTake(context.Background(), "key1", 1, config)
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = store.TryTake(context.Background(), "key2", 1, config)
	require.NoError(t, err)
	assert.True(t, allowed, "key2 should be independent from key1")
}

func TestInMemoryTokenBucketStore_TryTake_FractionalTokens(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 2}

	allowed, err := store.TryTake(context.Background(), "key", 1.5, config)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = store.TryTake(context.Background(), "key", 1.5, config)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestInMemoryTokenBucketStore_WaitDuration_TokensAvailable(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 5}

	dur, err := store.WaitDuration(context.Background(), "key", 1, config)
	require.NoError(t, err)
	assert.Zero(t, dur, "should not need to wait when tokens are available")
}

func TestInMemoryTokenBucketStore_WaitDuration_TokensExhausted(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 100, Burst: 1}

	_, _ = store.TryTake(context.Background(), "key", 1, config)

	dur, err := store.WaitDuration(context.Background(), "key", 1, config)
	require.NoError(t, err)

	assert.Greater(t, dur, time.Duration(0), "should report a positive wait duration")
	assert.LessOrEqual(t, dur, 15*time.Millisecond, "wait duration should be reasonable")
}

func TestInMemoryTokenBucketStore_DeleteBucket(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 1}

	_, _ = store.TryTake(context.Background(), "key", 1, config)
	allowed, _ := store.TryTake(context.Background(), "key", 1, config)
	require.False(t, allowed)

	err := store.DeleteBucket(context.Background(), "key")
	require.NoError(t, err)

	allowed, err = store.TryTake(context.Background(), "key", 1, config)
	require.NoError(t, err)
	assert.True(t, allowed, "should be allowed after bucket deletion")
}

func TestInMemoryTokenBucketStore_DeleteBucket_NonExistentKey(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	err := store.DeleteBucket(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

func TestInMemoryTokenBucketStore_ConcurrentAccess(t *testing.T) {
	store, _ := newTestInMemoryStore(t)
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1000, Burst: 100}

	const goroutines = 10
	const requestsPerGoroutine = 10

	var wg sync.WaitGroup
	results := make(chan bool, goroutines*requestsPerGoroutine)

	for range goroutines {
		wg.Go(func() {
			for range requestsPerGoroutine {
				allowed, err := store.TryTake(context.Background(), "key", 1, config)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				results <- allowed
			}
		})
	}

	wg.Wait()
	close(results)

	allowedCount := 0
	for allowed := range results {
		if allowed {
			allowedCount++
		}
	}

	assert.LessOrEqual(t, allowedCount, 100, "allowed count should not exceed burst")
	assert.Greater(t, allowedCount, 0, "at least some requests should be allowed")
}

func TestInMemoryTokenBucketStore_BurstDefaultsToRate(t *testing.T) {
	store, _ := newTestInMemoryStore(t)

	config := &ratelimiter_dto.TokenBucketConfig{Rate: 5, Burst: 0}

	for i := range 5 {
		allowed, err := store.TryTake(context.Background(), "key", 1, config)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := store.TryTake(context.Background(), "key", 1, config)
	require.NoError(t, err)
	assert.False(t, allowed, "should be denied after rate-based burst exhausted")
}
