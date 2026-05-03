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

package provider_mock_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_mock"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/wdk/clock"
)

func TestNewMockAdapter_ImplementsProviderPort(t *testing.T) {
	t.Parallel()

	adapter := provider_mock.NewMockAdapter[string, string]()
	var _ cache_domain.ProviderPort[string, string] = adapter

	require.NotNil(t, adapter)
}

func TestMockAdapter_SetThenGetIfPresent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	require.NoError(t, adapter.Set(ctx, "k", "v"))

	value, found, err := adapter.GetIfPresent(ctx, "k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "v", value)
}

func TestMockAdapter_GetIfPresent_RecordsCalls(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, int]()

	_, _, _ = adapter.GetIfPresent(ctx, "missing")
	_, _, _ = adapter.GetIfPresent(ctx, "another")

	require.Equal(t, 0, adapter.EstimatedSize())
}

func TestMockAdapter_Get_LoadsOnMissAndStores(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	loadCount := 0
	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, key string) (string, error) {
		loadCount++
		return "loaded:" + key, nil
	})

	first, err := adapter.Get(ctx, "k", loader)
	require.NoError(t, err)
	require.Equal(t, "loaded:k", first)
	require.Equal(t, 1, loadCount)

	second, err := adapter.Get(ctx, "k", loader)
	require.NoError(t, err)
	require.Equal(t, "loaded:k", second)
	require.Equal(t, 1, loadCount)

	require.Len(t, adapter.GetGetCalls(), 2)
}

func TestMockAdapter_Set_RecordsTagsAndCallHistory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	require.NoError(t, adapter.Set(ctx, "a", "v1", "alpha", "common"))
	require.NoError(t, adapter.Set(ctx, "b", "v2", "common"))

	calls := adapter.GetSetCalls()
	require.Len(t, calls, 2)
	require.Equal(t, "a", calls[0].Key)
	require.Equal(t, "v1", calls[0].Value)
	require.Equal(t, []string{"alpha", "common"}, calls[0].Tags)
}

func TestMockAdapter_InvalidateByTags_DeletesTaggedEntries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	require.NoError(t, adapter.Set(ctx, "a", "1", "tag1"))
	require.NoError(t, adapter.Set(ctx, "b", "2", "tag1", "tag2"))
	require.NoError(t, adapter.Set(ctx, "c", "3", "tag2"))
	require.Equal(t, 3, adapter.EstimatedSize())

	count, err := adapter.InvalidateByTags(ctx, "tag1")
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, 1, adapter.EstimatedSize())

	_, found, _ := adapter.GetIfPresent(ctx, "a")
	require.False(t, found)
	_, found, _ = adapter.GetIfPresent(ctx, "c")
	require.True(t, found)
}

func TestMockAdapter_InvalidateAll_ClearsState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.Set(ctx, "a", "1"))
	require.NoError(t, adapter.Set(ctx, "b", "2"))

	require.NoError(t, adapter.InvalidateAll(ctx))
	require.Equal(t, 0, adapter.EstimatedSize())
	require.Equal(t, 1, adapter.GetInvalidateAllCount())
}

func TestMockAdapter_Invalidate_RemovesSingleKeyAndUpdatesTagIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.Set(ctx, "a", "1", "tag"))
	require.NoError(t, adapter.Set(ctx, "b", "2", "tag"))

	require.NoError(t, adapter.Invalidate(ctx, "a"))
	require.Equal(t, []string{"a"}, adapter.GetInvalidateCalls())
	require.Equal(t, 1, adapter.EstimatedSize())

	count, err := adapter.InvalidateByTags(ctx, "tag")
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestMockAdapter_SetError_PropagatesError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	sentinel := errors.New("injected failure")

	adapter.SetError(sentinel)

	err := adapter.Set(ctx, "k", "v")
	require.ErrorIs(t, err, sentinel)

	_, _, err = adapter.GetIfPresent(ctx, "k")
	require.ErrorIs(t, err, sentinel)
}

func TestMockAdapter_SetWithTTL_ExpiresAfterDuration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clk := clock.NewMockClock(time.Unix(0, 0))
	adapter := provider_mock.NewMockAdapter(provider_mock.WithMockClock[string, string](clk))

	require.NoError(t, adapter.SetWithTTL(ctx, "k", "v", time.Second))

	value, found, err := adapter.GetIfPresent(ctx, "k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "v", value)

	clk.Advance(2 * time.Second)
	_, found, err = adapter.GetIfPresent(ctx, "k")
	require.NoError(t, err)
	require.False(t, found)
}

func TestMockAdapter_BulkSetAndBulkGet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, int]()

	require.NoError(t, adapter.BulkSet(ctx, map[string]int{"a": 1, "b": 2}))

	loader := cache_dto.BulkLoaderFunc[string, int](func(_ context.Context, _ []string) (map[string]int, error) {
		return map[string]int{"c": 3}, nil
	})

	got, err := adapter.BulkGet(ctx, []string{"a", "b", "c"}, loader)
	require.NoError(t, err)
	require.Equal(t, 1, got["a"])
	require.Equal(t, 2, got["b"])
	require.Equal(t, 3, got["c"])
}

func TestMockAdapter_BulkGet_UsesCustomFunctionWhenSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, int]()

	want := map[string]int{"x": 9}
	adapter.SetBulkGetFunc(func(_ context.Context, _ []string) (map[string]int, error) {
		return want, nil
	})

	got, err := adapter.BulkGet(ctx, []string{"x"}, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestMockAdapter_Compute_ActionSetAndDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	value, present, err := adapter.Compute(ctx, "k", func(_ string, _ bool) (string, cache_dto.ComputeAction) {
		return "new", cache_dto.ComputeActionSet
	})
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "new", value)

	value, present, err = adapter.Compute(ctx, "k", func(old string, found bool) (string, cache_dto.ComputeAction) {
		require.True(t, found)
		require.Equal(t, "new", old)
		return "", cache_dto.ComputeActionDelete
	})
	require.NoError(t, err)
	require.False(t, present)
	require.Empty(t, value)
}

func TestMockAdapter_ComputeIfAbsent_OnlyComputesOnMiss(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, int]()

	called := 0
	value, computed, err := adapter.ComputeIfAbsent(ctx, "k", func() int { called++; return 42 })
	require.NoError(t, err)
	require.True(t, computed)
	require.Equal(t, 42, value)

	value, computed, err = adapter.ComputeIfAbsent(ctx, "k", func() int { called++; return 99 })
	require.NoError(t, err)
	require.False(t, computed)
	require.Equal(t, 42, value)
	require.Equal(t, 1, called)
}

func TestMockAdapter_ComputeIfPresent_DoesNothingOnMiss(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, int]()

	value, present, err := adapter.ComputeIfPresent(ctx, "k", func(_ int) (int, cache_dto.ComputeAction) {
		t.Fatal("compute fn must not be called when absent")
		return 0, cache_dto.ComputeActionSet
	})
	require.NoError(t, err)
	require.False(t, present)
	require.Zero(t, value)
}

func TestMockAdapter_ComputeWithTTL_AppliesPerCallTTL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clk := clock.NewMockClock(time.Unix(0, 0))
	adapter := provider_mock.NewMockAdapter(provider_mock.WithMockClock[string, string](clk))

	value, present, err := adapter.ComputeWithTTL(ctx, "k", func(_ string, _ bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{Value: "v", Action: cache_dto.ComputeActionSet, TTL: 100 * time.Millisecond}
	})
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, "v", value)

	clk.Advance(200 * time.Millisecond)
	_, found, err := adapter.GetIfPresent(ctx, "k")
	require.NoError(t, err)
	require.False(t, found)
}

func TestMockAdapter_AllKeysValues_IteratesContents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, int]()
	require.NoError(t, adapter.Set(ctx, "a", 1))
	require.NoError(t, adapter.Set(ctx, "b", 2))

	var collectedKeys []string
	for k := range adapter.Keys() {
		collectedKeys = append(collectedKeys, k)
	}
	slices.Sort(collectedKeys)
	require.Equal(t, []string{"a", "b"}, collectedKeys)

	totalValues := 0
	for v := range adapter.Values() {
		totalValues += v
	}
	require.Equal(t, 3, totalValues)
}

func TestMockAdapter_Stats_ReturnsRecordedHits(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.Set(ctx, "k", "v"))

	_, _, _ = adapter.GetIfPresent(ctx, "k")
	_, _, _ = adapter.GetIfPresent(ctx, "missing")

	stats := adapter.Stats()
	require.Equal(t, uint64(2), stats.Hits)
}

func TestMockAdapter_SetMaximumGetMaximum(t *testing.T) {
	t.Parallel()

	adapter := provider_mock.NewMockAdapter[string, string]()
	adapter.SetMaximum(100)

	require.Equal(t, uint64(100), adapter.GetMaximum())
}

func TestMockAdapter_WeightedSize_ReportsStorageCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.Set(ctx, "a", "1"))
	require.NoError(t, adapter.Set(ctx, "b", "2"))
	require.NoError(t, adapter.Set(ctx, "c", "3"))

	require.Equal(t, uint64(3), adapter.WeightedSize())
}

func TestMockAdapter_Close_TracksCallCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	require.NoError(t, adapter.Close(ctx))
	require.NoError(t, adapter.Close(ctx))
}

func TestMockAdapter_Reset_ClearsState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.Set(ctx, "a", "1"))
	adapter.SetError(errors.New("err"))

	adapter.Reset()

	require.Equal(t, 0, adapter.EstimatedSize())
	require.NoError(t, adapter.Set(ctx, "b", "2"))
}

func TestMockAdapter_SearchAndQuery_ReturnNotSupported(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	_, err := adapter.Search(ctx, "*", nil)
	require.ErrorIs(t, err, cache_domain.ErrSearchNotSupported)

	_, err = adapter.Query(ctx, nil)
	require.ErrorIs(t, err, cache_domain.ErrSearchNotSupported)

	require.False(t, adapter.SupportsSearch())
	require.Nil(t, adapter.GetSchema())
}

func TestMockProviderFactory_CreatesAdapter(t *testing.T) {
	t.Parallel()

	cache, err := provider_mock.MockProviderFactory(cache_dto.Options[string, string]{})

	require.NoError(t, err)
	require.NotNil(t, cache)
}

func TestMockAdapter_Refresh_DeliversLoadResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	loader := cache_dto.LoaderFunc[string, string](func(_ context.Context, key string) (string, error) {
		return "fresh:" + key, nil
	})

	resultChan := adapter.Refresh(ctx, "k", loader)
	result := <-resultChan
	require.NoError(t, result.Err)
	require.Equal(t, "fresh:k", result.Value)
}

func TestMockAdapter_BulkRefresh_DispatchesToLoader(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()

	called := make(chan struct{}, 1)
	loader := cache_dto.BulkLoaderFunc[string, string](func(_ context.Context, _ []string) (map[string]string, error) {
		called <- struct{}{}
		return map[string]string{"a": "1"}, nil
	})

	adapter.BulkRefresh(ctx, []string{"a"}, loader)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("BulkRefresh loader not called within timeout")
	}
}

func TestMockAdapter_GetEntryProbeEntry_ReturnsEntries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.Set(ctx, "k", "v"))

	entry, found, err := adapter.GetEntry(ctx, "k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "k", entry.Key)
	require.Equal(t, "v", entry.Value)

	probeEntry, found, err := adapter.ProbeEntry(ctx, "k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, entry.Key, probeEntry.Key)
}

func TestMockAdapter_SetExpiresAfter_StoresDurationAndExpiresEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clk := clock.NewMockClock(time.Unix(0, 0))
	adapter := provider_mock.NewMockAdapter(provider_mock.WithMockClock[string, string](clk))
	require.NoError(t, adapter.Set(ctx, "k", "v"))

	require.NoError(t, adapter.SetExpiresAfter(ctx, "k", time.Second))

	clk.Advance(2 * time.Second)
	_, found, err := adapter.GetIfPresent(ctx, "k")
	require.NoError(t, err)
	require.False(t, found)
}

func TestMockAdapter_SetRefreshableAfter_RecordsCalls(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := provider_mock.NewMockAdapter[string, string]()
	require.NoError(t, adapter.SetRefreshableAfter(ctx, "k", time.Minute))
}
