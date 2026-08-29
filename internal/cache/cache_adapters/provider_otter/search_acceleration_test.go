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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
)

func newSortableProductCache(t *testing.T, count int) *OtterAdapter[string, Product] {
	t.Helper()

	provider, err := OtterProviderFactory(cache_dto.Options[string, Product]{
		MaximumEntries: count * 2,
		SearchSchema:   cache_dto.NewSearchSchema(cache_dto.SortableNumericField("Stock")),
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, Product])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	ctx := context.Background()
	for i := range count {
		require.NoError(t, cache.Set(ctx, fmt.Sprintf("p%d", i), Product{ID: fmt.Sprintf("p%d", i), Stock: i}))
	}

	return cache
}

func TestQuery_EqualityIsAnsweredFromTheSortedIndex(t *testing.T) {
	t.Parallel()

	cache := newSortableProductCache(t, 50)

	result, err := cache.Query(context.Background(), &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{{Field: "Stock", Operation: cache_dto.FilterOpEq, Value: 17}},
		Limit:   cache_dto.NoLimit,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1, "equality is the commonest lookup and must be answerable")
	assert.Equal(t, "p17", result.Items[0].Key)
}

func TestQuery_InIsAnsweredFromTheSortedIndex(t *testing.T) {
	t.Parallel()

	cache := newSortableProductCache(t, 50)

	result, err := cache.Query(context.Background(), &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{{Field: "Stock", Operation: cache_dto.FilterOpIn, Values: []any{3, 9, 21}}},
		Limit:   cache_dto.NoLimit,
	})

	require.NoError(t, err)
	assert.Len(t, result.Items, 3)
}

func TestQuery_AcceleratesAFilterThatIsNotTheFirst(t *testing.T) {
	t.Parallel()

	cache := newSortableProductCache(t, 60)

	result, err := cache.Query(context.Background(), &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{
			{Field: "ID", Operation: cache_dto.FilterOpEq, Value: "p42"},
			{Field: "Stock", Operation: cache_dto.FilterOpBetween, Values: []any{40, 45}},
		},
		Limit: cache_dto.NoLimit,
	})

	require.NoError(t, err)
	require.Len(t, result.Items, 1,
		"an index-servable filter beyond the first must still narrow the candidate set")
	assert.Equal(t, "p42", result.Items[0].Key)
}

func TestQuery_AccelerationSurvivesExpiredEntries(t *testing.T) {
	t.Parallel()

	cache := newSortableProductCache(t, 40)
	ctx := context.Background()

	require.NoError(t, cache.SetWithTTL(ctx, "doomed", Product{ID: "doomed", Stock: 999}, time.Nanosecond))

	result, err := cache.Query(ctx, &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{{Field: "Stock", Operation: cache_dto.FilterOpBetween, Values: []any{5, 9}}},
		Limit:   cache_dto.NoLimit,
	})

	require.NoError(t, err)
	assert.Len(t, result.Items, 5,
		"an expired entry must not disable index acceleration for everything else")
}

func TestQuery_NoLimitReturnsEveryMatch(t *testing.T) {
	t.Parallel()

	cache := newSortableProductCache(t, 50)

	limited, err := cache.Query(context.Background(), &cache_dto.QueryOptions{})
	require.NoError(t, err)
	assert.Len(t, limited.Items, 10, "an unspecified limit still pages")

	all, err := cache.Query(context.Background(), &cache_dto.QueryOptions{Limit: cache_dto.NoLimit})
	require.NoError(t, err)
	assert.Len(t, all.Items, 50, "NoLimit must return every match rather than a page")
}
