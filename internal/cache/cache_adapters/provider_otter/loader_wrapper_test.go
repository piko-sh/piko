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

package provider_otter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
)

func newStringCache(t *testing.T, options cache_dto.Options[string, string]) *OtterAdapter[string, string] {
	t.Helper()

	provider, err := OtterProviderFactory(options)
	require.NoError(t, err)
	adapter, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { _ = adapter.Close(context.Background()) })

	return adapter
}

func TestGet_ErrDoNotStore_ReturnsTheValueWithoutStoringIt(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{MaximumEntries: 100})
	ctx := context.Background()

	value, err := cache.Get(ctx, "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "fresh-value", cache_dto.ErrDoNotStore
	}})

	require.NoError(t, err, "declining to cache is not a failure for the caller, who asked for the value")
	assert.Equal(t, "fresh-value", value)

	_, found, err := cache.GetIfPresent(ctx, "key")
	require.NoError(t, err)
	assert.False(t, found, "the value must not have been admitted")
}

func TestRefresh_ErrDoNotStore_RemovesTheSupersededEntry(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{
		MaximumEntries:    100,
		RefreshCalculator: alwaysRefreshCalculator[string, string]{},
	})
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "key", "stale-value"))

	result := <-cache.Refresh(ctx, "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "fresh-value", cache_dto.ErrDoNotStore
	}})

	require.NoError(t, result.Err, "declining to cache is not a failure")
	assert.Equal(t, "fresh-value", result.Value)

	_, found, err := cache.GetIfPresent(ctx, "key")
	require.NoError(t, err)
	assert.False(t, found, "a declined reload must not leave the superseded value behind")
}

func TestGet_ErrDoNotStore_AgreesAcrossSingleflightWaiters(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{MaximumEntries: 100})
	ctx := context.Background()

	release := make(chan struct{})
	entered := make(chan struct{}, 1)
	var loadCount, arrived atomic.Int64

	const waiters = 16
	values := make([]string, waiters)
	errs := make([]error, waiters)

	var finished sync.WaitGroup
	finished.Add(waiters)

	for i := range waiters {
		go func() {
			defer finished.Done()
			arrived.Add(1)
			values[i], errs[i] = cache.Get(ctx, "shared", loaderFunc{
				load: func(context.Context, string) (string, error) {
					loadCount.Add(1)
					select {
					case entered <- struct{}{}:
					default:
					}
					<-release

					return "one-answer", cache_dto.ErrDoNotStore
				},
			})
		}()
	}

	<-entered
	require.Eventually(t, func() bool { return arrived.Load() == waiters }, time.Second, time.Millisecond,
		"every caller must reach Get before the in-flight load is allowed to finish")
	close(release)
	finished.Wait()

	for i := range waiters {
		require.NoError(t, errs[i], "waiter %d", i)
		assert.Equal(t, "one-answer", values[i], "waiter %d", i)
	}

	assert.Less(t, loadCount.Load(), int64(waiters),
		"a wrapper that defeated stampede protection would run one load per caller")

	_, found, err := cache.GetIfPresent(ctx, "shared")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestGet_ErrDoNotStore_CountsAsALoadSuccess(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{MaximumEntries: 100})

	_, err := cache.Get(context.Background(), "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "value", cache_dto.ErrDoNotStore
	}})
	require.NoError(t, err)

	stats := cache.Stats()
	assert.Zero(t, stats.LoadFailureCount, "declining to cache is not a load failure")
	assert.Equal(t, uint64(1), stats.LoadSuccessCount)
}

func TestGet_ErrNotFound_SurfacesTheSentinelAndIsNotAFailure(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{MaximumEntries: 100})

	_, err := cache.Get(context.Background(), "missing", loaderFunc{load: func(context.Context, string) (string, error) {
		return "", cache_dto.ErrNotFound
	}})

	require.ErrorIs(t, err, cache_dto.ErrNotFound, "the caller's own sentinel must survive the round trip")
	assert.Zero(t, cache.Stats().LoadFailureCount, "a documented absence is not a load failure")
}

func TestRefresh_ErrNotFound_RemovesTheStaleEntry(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{
		MaximumEntries:    100,
		RefreshCalculator: alwaysRefreshCalculator[string, string]{},
	})
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "key", "stale-value"))

	result := <-cache.Refresh(ctx, "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "", cache_dto.ErrNotFound
	}})
	require.ErrorIs(t, result.Err, cache_dto.ErrNotFound)

	_, found, err := cache.GetIfPresent(ctx, "key")
	require.NoError(t, err)
	assert.False(t, found, "an upstream deletion must evict the stale entry")
}

func TestGet_LoaderFailurePropagates(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{MaximumEntries: 100})
	upstream := errors.New("upstream unavailable")

	_, err := cache.Get(context.Background(), "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "", upstream
	}})

	require.ErrorIs(t, err, upstream)
	assert.Equal(t, uint64(1), cache.Stats().LoadFailureCount)
}

func TestGet_LoadedEntryIsSearchable(t *testing.T) {
	t.Parallel()

	schema := cache_dto.NewSearchSchema(cache_dto.TextField("Name"))

	provider, err := OtterProviderFactory(cache_dto.Options[string, Product]{
		MaximumEntries: 100,
		SearchSchema:   schema,
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, Product])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	ctx := context.Background()
	loaded := Product{Name: "widget"}

	_, err = cache.Get(ctx, "p1", productLoader{value: loaded})
	require.NoError(t, err)

	result, err := cache.Search(ctx, "widget", &cache_dto.SearchOptions{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1,
		"an entry admitted through the loader must be searchable, or Search silently omits it")
	assert.Equal(t, "p1", result.Items[0].Key)
}

type alwaysRefreshCalculator[K comparable, V any] struct{}

func (alwaysRefreshCalculator[K, V]) RefreshAfterCreate(cache_dto.Entry[K, V]) time.Duration {
	return time.Nanosecond
}

func (alwaysRefreshCalculator[K, V]) RefreshAfterUpdate(cache_dto.Entry[K, V], V) time.Duration {
	return time.Nanosecond
}

func (alwaysRefreshCalculator[K, V]) RefreshAfterRead(cache_dto.Entry[K, V]) time.Duration {
	return time.Nanosecond
}

func (alwaysRefreshCalculator[K, V]) RefreshAfterReload(cache_dto.Entry[K, V], V) time.Duration {
	return time.Nanosecond
}

func (alwaysRefreshCalculator[K, V]) RefreshAfterReloadFailure(cache_dto.Entry[K, V], error) time.Duration {
	return time.Nanosecond
}

type productLoader struct {
	value Product
}

func (p productLoader) Load(context.Context, string) (Product, error) { return p.value, nil }

func (p productLoader) Reload(context.Context, string, Product) (Product, error) {
	return p.value, nil
}

func TestGet_OversizedLoadedValueIsReturnedButNotStored(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{
		MaximumWeight:  1 << 20,
		MaxEntryWeight: 8,
		Weigher:        func(_ string, value string) uint32 { return uint32(len(value)) },
	})
	ctx := context.Background()

	value, err := cache.Get(ctx, "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "far-too-long-a-value", nil
	}})

	require.NoError(t, err, "the caller asked for the value and the loader produced it")
	assert.Equal(t, "far-too-long-a-value", value)

	_, found, err := cache.GetIfPresent(ctx, "key")
	require.NoError(t, err)
	assert.False(t, found,
		"the read-through path must honour the ceiling too, or it binds on writes and silently not on loads")
}

func TestGet_LoadedValueAtTheCeilingIsStored(t *testing.T) {
	t.Parallel()

	cache := newStringCache(t, cache_dto.Options[string, string]{
		MaximumWeight:  1 << 20,
		MaxEntryWeight: 8,
		Weigher:        func(_ string, value string) uint32 { return uint32(len(value)) },
	})
	ctx := context.Background()

	_, err := cache.Get(ctx, "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "12345678", nil
	}})
	require.NoError(t, err)

	value, found, err := cache.GetIfPresent(ctx, "key")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "12345678", value)
}

func TestRefresh_OversizedReloadKeepsTheCachedEntry(t *testing.T) {
	t.Parallel()

	var rejections []cache_dto.DeletionEvent[string, string]

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumWeight:     1 << 20,
		MaxEntryWeight:    8,
		Weigher:           func(_ string, value string) uint32 { return uint32(len(value)) },
		RefreshCalculator: alwaysRefreshCalculator[string, string]{},
		OnDeletion: func(event cache_dto.DeletionEvent[string, string]) {
			if event.Cause == cache_dto.CauseRejected {
				rejections = append(rejections, event)
			}
		},
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	ctx := context.Background()
	require.NoError(t, cache.Set(ctx, "key", "good"))

	<-cache.Refresh(ctx, "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "far-too-long-a-value", nil
	}})

	value, found, getErr := cache.GetIfPresent(ctx, "key")
	require.NoError(t, getErr)
	require.True(t, found,
		"an oversized refresh must not cost the caller the good entry it was refreshing")
	assert.Equal(t, "good", value)

	require.Len(t, rejections, 1, "the read path must report a refusal like the write path does")
	assert.Equal(t, "key", rejections[0].Key)
	assert.False(t, rejections[0].WasEvicted())
}

func TestGet_OversizedLoadReportsTheRejection(t *testing.T) {
	t.Parallel()

	var rejections []cache_dto.DeletionEvent[string, string]

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumWeight:  1 << 20,
		MaxEntryWeight: 8,
		Weigher:        func(_ string, value string) uint32 { return uint32(len(value)) },
		OnDeletion: func(event cache_dto.DeletionEvent[string, string]) {
			if event.Cause == cache_dto.CauseRejected {
				rejections = append(rejections, event)
			}
		},
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	_, err = cache.Get(context.Background(), "key", loaderFunc{load: func(context.Context, string) (string, error) {
		return "far-too-long-a-value", nil
	}})
	require.NoError(t, err)

	require.Len(t, rejections, 1)
	assert.Equal(t, cache_dto.CauseRejected, rejections[0].Cause)
}

func TestWrapLoader_PassesThroughTheAbsenceOfALoader(t *testing.T) {
	t.Parallel()

	adapter := newStringCache(t, cache_dto.Options[string, string]{MaximumEntries: 10})

	assert.Nil(t, adapter.wrapLoader(nil),
		"otter reads a nil loader as no loader, and a wrapper around nothing would look like one")
	assert.Nil(t, adapter.wrapBulkLoader(nil))

	assert.NotNil(t, adapter.wrapLoader(loaderFunc{
		load: func(context.Context, string) (string, error) { return "", nil },
	}))
}
