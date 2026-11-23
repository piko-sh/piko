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
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
	"piko.sh/sharded"
)

func newTestStore(t *testing.T) (*Store, *clock.MockClock) {
	t.Helper()
	mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	mockCache := provider_mock.NewMockAdapter[string, *storedEntry](
		provider_mock.WithMockClock[string, *storedEntry](mockClock),
	)

	store, err := New(t.Context(), Config{
		Clock: mockClock,
	}, withCache(mockCache))
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
	ctx := t.Context()

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
	ctx := t.Context()

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
	ctx := t.Context()

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
	ctx := t.Context()

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
	ctx := t.Context()

	err := store.Set(ctx, "key1", nil)
	require.NoError(t, err)

	result, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestStore_Set_ZeroTTL(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := t.Context()

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
	ctx := t.Context()

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
	ctx := t.Context()

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
	ctx := t.Context()

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
	require.NoError(t, store.Close(t.Context()))
}

func TestStore_UseAfterCloseDoesNotPanic(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	store, err := New(ctx, Config{CacheService: cache_domain.NewService("otter")})
	require.NoError(t, err)
	require.NoError(t, store.Close(ctx))

	now := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	assert.NotPanics(t, func() {
		_ = store.Set(ctx, "key", newTestEntry(now, now.Add(time.Hour)))
		_, _ = store.Get(ctx, "key")
		_ = store.Delete(ctx, "key")
		_ = store.Clear(ctx)
		_, _ = store.GetStats(ctx)
		_ = store.Close(ctx)
	}, "a request still running when the store closes must not crash the process")
}

func TestStore_CacheDropsEntryOnceItExpires(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		advance     time.Duration
		wantPresent bool
	}{
		{
			name:        "entry is kept until it expires",
			advance:     time.Hour - time.Second,
			wantPresent: true,
		},
		{
			name:        "entry is dropped once it expires",
			advance:     time.Hour,
			wantPresent: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
			store, err := New(ctx, Config{
				CacheService:   cache_domain.NewService(""),
				Clock:          mockClock,
				Provider:       "",
				Namespace:      "",
				MaximumEntries: 0,
			})
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, store.Close(context.Background())) })

			now := mockClock.Now()
			require.NoError(t, store.Set(ctx, "key1", newTestEntry(now, now.Add(time.Hour))))
			mockClock.Advance(testCase.advance)

			_, present, err := store.cache.GetIfPresent(ctx, "key1")
			require.NoError(t, err)
			assert.Equal(t, testCase.wantPresent, present, "the cache itself must drop the entry, not only Get")
		})
	}
}

func TestStore_Get_CountsHits(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := t.Context()

	now := mockClock.Now()
	require.NoError(t, store.Set(ctx, "key1", newTestEntry(now, now.Add(time.Hour))))

	for want := int64(1); want <= 3; want++ {
		result, err := store.Get(ctx, "key1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, want, result.HitCount)
	}
}

func TestStore_Get_ReturnsCopyOwnedByCaller(t *testing.T) {
	t.Parallel()
	store, mockClock := newTestStore(t)
	ctx := t.Context()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(time.Hour))
	entry.Response = &llm_dto.CompletionResponse{
		ID:      "response-1",
		Choices: []llm_dto.Choice{{Message: llm_dto.NewAssistantMessage("cached")}},
	}
	require.NoError(t, store.Set(ctx, "key1", entry))
	entry.Response.Choices[0].Message.Content = "changed after Set"

	first, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	first.Response.Choices[0].Message.Content = "changed by a reader"
	first.Model = "changed by a reader"

	second, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "cached", second.Response.Content())
	assert.Equal(t, "gpt-4o", second.Model)
}

func TestStore_Get_ConcurrentReadersDoNotShareEntries(t *testing.T) {
	t.Parallel()

	const readers = 8

	store, mockClock := newTestStore(t)
	ctx := t.Context()

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(time.Hour))
	entry.Response = &llm_dto.CompletionResponse{ID: "shared"}
	require.NoError(t, store.Set(ctx, "key1", entry))

	var wg sync.WaitGroup
	for reader := range readers {
		wg.Go(func() {
			result, err := store.Get(ctx, "key1")
			if !assert.NoError(t, err) || !assert.NotNil(t, result) {
				return
			}
			result.HitCount = int64(reader)
			result.Response.ID = "reader"
		})
	}
	wg.Wait()

	result, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "shared", result.Response.ID)
	assert.Equal(t, int64(readers+1), result.HitCount)
}

func TestStore_Set_SkipsOversizedResponse(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	store, err := New(ctx, Config{
		CacheService:       cache_domain.NewService(""),
		Clock:              mockClock,
		Provider:           "",
		Namespace:          "",
		MaximumEntries:     0,
		MaximumWeightBytes: 0,
		MaximumEntryBytes:  1024,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, store.Close(context.Background())) })

	now := mockClock.Now()
	large := newTestEntry(now, now.Add(time.Hour))
	large.Response = &llm_dto.CompletionResponse{
		Choices: []llm_dto.Choice{{Message: llm_dto.NewAssistantMessage(strings.Repeat("x", 2048))}},
	}
	small := newTestEntry(now, now.Add(time.Hour))
	small.Response = &llm_dto.CompletionResponse{
		Choices: []llm_dto.Choice{{Message: llm_dto.NewAssistantMessage("short")}},
	}

	require.NoError(t, store.Set(ctx, "large", large))
	require.NoError(t, store.Set(ctx, "small", small))

	largeResult, err := store.Get(ctx, "large")
	require.NoError(t, err)
	assert.Nil(t, largeResult, "a response above the entry limit is not cached")

	smallResult, err := store.Get(ctx, "small")
	require.NoError(t, err)
	assert.NotNil(t, smallResult)
}

func TestStore_Close_PropagatesCacheError(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")
	store, err := New(t.Context(), Config{}, withCache(&failingCache{closeErr: closeErr}))
	require.NoError(t, err)

	require.ErrorIs(t, store.Close(t.Context()), closeErr)
}

func TestStore_Get_ExpiredEntryInvalidateFailureIsAMiss(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	now := mockClock.Now()
	cache := &failingCache{
		stored:        &storedEntry{Entry: newTestEntry(now.Add(-2*time.Hour), now.Add(-time.Hour))},
		invalidateErr: errors.New("invalidate failed"),
	}
	store, err := New(t.Context(), Config{Clock: mockClock}, withCache(cache))
	require.NoError(t, err)

	result, err := store.Get(t.Context(), "expired")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.True(t, cache.invalidated, "the expired entry is removed")
	assert.Equal(t, int64(1), store.misses.Value())
}

func TestStore_UsesConfiguredProvider(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClock := clock.NewMockClock(time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC))
	provider := provider_mock.NewMockProvider()
	service := cache_domain.NewService(sharedProviderName)
	require.NoError(t, service.RegisterProvider(ctx, sharedProviderName, provider))
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	store, err := New(ctx, Config{
		CacheService:       service,
		Clock:              mockClock,
		Provider:           sharedProviderName,
		Namespace:          "",
		MaximumEntries:     0,
		MaximumWeightBytes: 0,
		MaximumEntryBytes:  0,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	now := mockClock.Now()
	entry := newTestEntry(now, now.Add(time.Hour))
	entry.Response = &llm_dto.CompletionResponse{
		ID: "remote",
		FallbackInfo: &llm_dto.FallbackResult{
			Errors:             map[string]error{"openai": errors.New("timeout")},
			UsedProvider:       "anthropic",
			AttemptedProviders: []string{"openai", "anthropic"},
		},
	}
	require.NoError(t, store.Set(ctx, "key1", entry))

	cacheAny, err := provider.CreateNamespaceTyped(defaultNamespace, cache_dto.Options[string, []byte]{})
	require.NoError(t, err)
	byteCache, ok := cacheAny.(cache_domain.Cache[string, []byte])
	require.True(t, ok)
	_, found, err := byteCache.GetIfPresent(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, found, "the named provider holds the entry")

	result, err := store.Get(ctx, "key1")
	require.NoError(t, err, "an entry with fallback errors must still decode")
	require.NotNil(t, result)
	assert.Equal(t, "remote", result.Response.ID)
	assert.Equal(t, "anthropic", result.Response.FallbackInfo.UsedProvider)
	assert.Nil(t, result.Response.FallbackInfo.Errors)
}

func TestNew_Validation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		config  Config
	}{
		{
			name:    "missing cache service",
			config:  Config{},
			wantErr: nil,
		},
		{
			name: "entry limit above the cache weight",
			config: Config{
				CacheService:       cache_domain.NewService(""),
				MaximumWeightBytes: 1024,
				MaximumEntryBytes:  2048,
			},
			wantErr: errEntryLimitExceedsWeight,
		},
		{
			name: "unknown provider",
			config: Config{
				CacheService: cache_domain.NewService(""),
				Provider:     "missing",
			},
			wantErr: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			store, err := New(t.Context(), testCase.config)

			assert.Nil(t, store)
			require.Error(t, err)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			}
		})
	}
}

func TestNewWeigher(t *testing.T) {
	t.Parallel()

	counterBytes := uint32(sharded.InflatedCounterBytes())
	weigher := newWeigher(uint64(counterBytes)*40, 10)

	assert.Equal(t, 4*counterBytes, weigher("key", &storedEntry{weight: 1}), "small entries weigh the per-entry minimum")
	assert.Equal(t, 9*counterBytes, weigher("key", &storedEntry{weight: 8 * counterBytes}),
		"large entries weigh their size plus the hit counter once it has inflated")
	assert.Equal(t, counterBytes+1, newWeigher(uint64(counterBytes)*10+1, 10)("key", &storedEntry{weight: 0}),
		"the minimum rounds up")
	assert.Equal(t, uint32(math.MaxUint32), weigher("key", &storedEntry{weight: math.MaxUint32 - 1}),
		"the weight saturates instead of wrapping")
}

func TestApproximateValueBytes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		value any
		name  string
		want  int
	}{
		{name: "nil", value: nil, want: 0},
		{name: "number", value: 1.5, want: metadataScalarBytes},
		{name: "boolean", value: true, want: metadataScalarBytes},
		{name: "string counts its length", value: strings.Repeat("x", 10000), want: metadataScalarBytes + 10000},
		{name: "byte slice counts its length", value: []byte("abcd"), want: metadataScalarBytes + 4},
		{name: "string slice", value: []string{"ab", "c"}, want: metadataScalarBytes + 2*metadataScalarBytes + 3},
		{name: "string map", value: map[string]string{"k": "vv"}, want: metadataScalarBytes + 1 + metadataScalarBytes + 2},
		{
			name:  "nested values",
			value: map[string]any{"a": []any{"xy", 1}},
			want:  metadataScalarBytes + 1 + metadataScalarBytes + metadataScalarBytes + 2 + metadataScalarBytes,
		},
		{name: "unknown type", value: struct{}{}, want: metadataValueBytes},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, approximateValueBytes(testCase.value))
		})
	}
}

func TestApproximateValueBytesIsBounded(t *testing.T) {
	t.Parallel()

	var deep any = "leaf"
	for range 100000 {
		deep = []any{deep}
	}
	deepBytes := approximateValueBytes(deep)
	assert.Positive(t, deepBytes)
	assert.Less(t, deepBytes, 100000*metadataScalarBytes, "values below the depth limit are estimated, not walked")

	wide := make([]any, 1<<20)
	for index := range wide {
		wide[index] = index
	}
	assert.GreaterOrEqual(t, approximateValueBytes(wide), len(wide)*metadataScalarBytes,
		"a wide value is still counted once the walk limit is reached")
}

func TestApproximateEntryBytes(t *testing.T) {
	t.Parallel()

	text := "part"
	entry := &llm_dto.CacheEntry{
		RequestHash: "hash",
		Provider:    "openai",
		Model:       "gpt-4o",
		Response: &llm_dto.CompletionResponse{
			ID:    "id",
			Model: "gpt-4o",
			Choices: []llm_dto.Choice{{Message: llm_dto.Message{
				Content: "hello",
				ContentParts: []llm_dto.ContentPart{{
					Text:      &text,
					ImageURL:  &llm_dto.ImageURL{URL: "https://a"},
					ImageData: &llm_dto.ImageData{Data: "AAAA"},
				}},
				ToolCalls: []llm_dto.ToolCall{{ID: "c", Type: "function", Function: llm_dto.FunctionCall{Name: "f", Arguments: "{}"}}},
			}}},
			Sources: []llm_dto.VectorSearchResult{{
				Metadata: map[string]any{"k": 1},
				ID:       "s",
				Content:  "body",
				Vector:   []float32{1, 2},
			}},
		},
	}

	assert.Equal(t, entryOverheadBytes+len("hash")+len("openai")+len("gpt-4o"), approximateEntryBytes(&llm_dto.CacheEntry{
		RequestHash: "hash", Provider: "openai", Model: "gpt-4o",
	}))

	want := entryOverheadBytes + 4 + 6 + 6 +
		2 + 6 +
		5 + 4 + 9 + 4 + 1 + 8 + 1 + 2 +
		1 + 4 + 2*float32Bytes + metadataScalarBytes + 1 + metadataScalarBytes
	assert.Equal(t, want, approximateEntryBytes(entry))
}

const (
	sharedProviderName = "shared"
)

type failingCache struct {
	cache_domain.Cache[string, *storedEntry]
	stored        *storedEntry
	closeErr      error
	invalidateErr error
	invalidated   bool
}

func (c *failingCache) GetIfPresent(context.Context, string) (*storedEntry, bool, error) {
	return c.stored, c.stored != nil, nil
}

func (c *failingCache) Invalidate(context.Context, string) error {
	c.invalidated = true
	return c.invalidateErr
}

func (c *failingCache) Close(context.Context) error {
	return c.closeErr
}
