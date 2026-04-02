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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
)

func newTestStore(t *testing.T) (*Store, *clock.MockClock) {
	t.Helper()
	mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	mockCache := provider_mock.NewMockAdapter[string, *llm_dto.CacheEntry](
		provider_mock.WithMockClock[string, *llm_dto.CacheEntry](mockClock),
	)

	store, err := New(context.Background(), Config{
		Clock: mockClock,
	}, WithCache(mockCache))
	require.NoError(t, err)

	return store, mockClock
}

func newTestEntry(createdAt, expiresAt time.Time) *llm_dto.CacheEntry {
	return &llm_dto.CacheEntry{
		RequestHash: "abc123",
		Provider:    "openai",
		Model:       "gpt-4o",
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
	}
}

func TestStore_Get_CacheHit(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(time.Hour))
	require.NoError(t, store.Set(ctx, "key1", entry))

	result, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "abc123", result.RequestHash)

	stats, _ := store.GetStats(ctx)
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
}

func TestStore_Get_CacheMiss(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t)
	ctx := context.Background()

	result, err := store.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, result)

	stats, _ := store.GetStats(ctx)
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestStore_Get_Expired(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(30*time.Minute))
	require.NoError(t, store.Set(ctx, "expiring", entry))

	mockClock.Advance(time.Hour)

	result, err := store.Get(ctx, "expiring")
	require.NoError(t, err)
	assert.Nil(t, result)

	stats, _ := store.GetStats(ctx)
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestStore_Set(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(time.Hour))
	require.NoError(t, store.Set(ctx, "key1", entry))

	result, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "openai", result.Provider)
}

func TestStore_Set_NilEntry(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t)
	ctx := context.Background()

	err := store.Set(ctx, "key1", nil)
	require.NoError(t, err)

	result, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestStore_Set_ZeroTTL(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(-time.Hour))
	require.NoError(t, store.Set(ctx, "notTTL", entry))

	result, err := store.Get(ctx, "notTTL")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(time.Hour))
	require.NoError(t, store.Set(ctx, "toDelete", entry))

	require.NoError(t, store.Delete(ctx, "toDelete"))

	result, err := store.Get(ctx, "toDelete")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestStore_Clear(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	require.NoError(t, store.Set(ctx, "a", newTestEntry(now, now.Add(time.Hour))))
	require.NoError(t, store.Set(ctx, "b", newTestEntry(now, now.Add(time.Hour))))

	require.NoError(t, store.Clear(ctx))

	resultA, err := store.Get(ctx, "a")
	require.NoError(t, err)
	assert.Nil(t, resultA)

	resultB, err := store.Get(ctx, "b")
	require.NoError(t, err)
	assert.Nil(t, resultB)
}

func TestStore_GetStats(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := context.Background()

	now := mockClock.Now()
	require.NoError(t, store.Set(ctx, "k", newTestEntry(now, now.Add(time.Hour))))

	_, _ = store.Get(ctx, "k")
	_, _ = store.Get(ctx, "k")
	_, _ = store.Get(ctx, "missing")

	stats, err := store.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
}

func TestStore_Close(t *testing.T) {
	t.Parallel()
	store, _ := newTestStore(t)
	store.Close()
}
