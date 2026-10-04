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

package provider_multilevel

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
)

func newTwoLevelService(t *testing.T) cache_domain.Service {
	t.Helper()

	service := cache_domain.NewService("l1")
	require.NoError(t, service.RegisterProvider(context.Background(), "l1", provider_otter.NewOtterProvider()))
	require.NoError(t, service.RegisterProvider(context.Background(), "l2", provider_otter.NewOtterProvider()))
	t.Cleanup(func() { _ = service.Close(context.Background()) })

	return service
}

func buildMultiLevel[V any](t *testing.T, service cache_domain.Service, namespace string) cache_domain.Cache[string, V] {
	t.Helper()

	cache, err := cache_domain.NewCacheBuilder[string, V](service).
		Namespace(namespace).
		MultiLevel("l1", "l2").
		MaximumEntries(100).
		Build(context.Background())
	require.NoError(t, err)

	return cache
}

func TestBuilderMultiLevelCachesKeepTheirNamespacesApart(t *testing.T) {
	ctx := context.Background()
	service := newTwoLevelService(t)

	users := buildMultiLevel[string](t, service, "users")
	sessions := buildMultiLevel[string](t, service, "sessions")
	counters := buildMultiLevel[[]byte](t, service, "counters")

	require.NoError(t, users.Set(ctx, "id-1", "alice-profile"))
	require.NoError(t, sessions.Set(ctx, "id-2", "session-token"))
	require.NoError(t, counters.Set(ctx, "id-3", []byte{1}))

	_, found, err := sessions.GetIfPresent(ctx, "id-1")
	require.NoError(t, err)
	assert.False(t, found, "one namespace read another namespace's key")

	require.NoError(t, sessions.InvalidateAll(ctx))

	value, found, err := users.GetIfPresent(ctx, "id-1")
	require.NoError(t, err)
	assert.True(t, found, "invalidating one namespace wiped another")
	assert.Equal(t, "alice-profile", value)

	counter, found, err := counters.GetIfPresent(ctx, "id-3")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, []byte{1}, counter)
}

func TestBuilderMultiLevelRefusesEntriesAboveTheCeiling(t *testing.T) {
	ctx := context.Background()
	service := newTwoLevelService(t)

	cache, err := cache_domain.NewCacheBuilder[string, string](service).
		Namespace("weighted").
		MaximumWeight(1<<20).
		Weigher(func(_ string, value string) uint32 { return uint32(len(value)) }).
		MaxEntryWeight(10).
		MultiLevel("l1", "l2").
		Build(ctx)
	require.NoError(t, err)

	require.ErrorIs(t, cache.Set(ctx, "big", strings.Repeat("x", 1000)), cache_domain.ErrEntryTooLarge)
	_, found, err := cache.GetIfPresent(ctx, "big")
	require.NoError(t, err)
	assert.False(t, found, "a refused entry must not be stored at either level")

	require.NoError(t, cache.Set(ctx, "small", "fits"))
	value, found, err := cache.GetIfPresent(ctx, "small")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fits", value)
}
