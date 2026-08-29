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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
)

type countingRecorder struct {
	hits, misses, evictions, loadSuccesses, loadFailures atomic.Uint64
}

func (r *countingRecorder) RecordHits(count uint64)   { r.hits.Add(count) }
func (r *countingRecorder) RecordMisses(count uint64) { r.misses.Add(count) }
func (r *countingRecorder) RecordEviction()           { r.evictions.Add(1) }

func (r *countingRecorder) RecordLoadSuccess(time.Duration) { r.loadSuccesses.Add(1) }
func (r *countingRecorder) RecordLoadFailure(time.Duration) { r.loadFailures.Add(1) }

type loaderFunc struct {
	load func(ctx context.Context, key string) (string, error)
}

func (l loaderFunc) Load(ctx context.Context, key string) (string, error) { return l.load(ctx, key) }

func (l loaderFunc) Reload(ctx context.Context, key string, _ string) (string, error) {
	return l.load(ctx, key)
}

func driveStatsTraffic(t *testing.T, cache *OtterAdapter[string, string]) {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, cache.Set(ctx, "present", "value"))

	for range 2 {
		_, found, err := cache.GetIfPresent(ctx, "present")
		require.NoError(t, err)
		require.True(t, found)
	}
	for range 2 {
		_, found, err := cache.GetIfPresent(ctx, "absent")
		require.NoError(t, err)
		require.False(t, found)
	}

	_, err := cache.Get(ctx, "loaded", loaderFunc{load: func(context.Context, string) (string, error) {
		return "loaded-value", nil
	}})
	require.NoError(t, err)

	_, err = cache.Get(ctx, "unloadable", loaderFunc{load: func(context.Context, string) (string, error) {
		return "", errors.New("upstream unavailable")
	}})
	require.Error(t, err)
}

func TestStats_ReportsTrafficWithoutAConfiguredRecorder(t *testing.T) {
	t.Parallel()

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 100})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	driveStatsTraffic(t, cache)
	stats := cache.Stats()

	assert.Equal(t, uint64(2), stats.Hits, "an unconfigured cache must still be distinguishable from an idle one")
	assert.Equal(t, uint64(4), stats.Misses)
	assert.Equal(t, uint64(1), stats.LoadSuccessCount)
	assert.Equal(t, uint64(1), stats.LoadFailureCount)

	ratio, hasRequests := stats.HitRatio()
	require.True(t, hasRequests)
	assert.InDelta(t, 2.0/6.0, ratio, 0.0001)
}

func TestStats_ReportsTrafficAndForwardsToAConfiguredRecorder(t *testing.T) {
	t.Parallel()

	recorder := &countingRecorder{}
	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumEntries: 100,
		StatsRecorder:  recorder,
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	driveStatsTraffic(t, cache)
	stats := cache.Stats()

	assert.Equal(t, uint64(2), stats.Hits)
	assert.Equal(t, uint64(4), stats.Misses)
	assert.Equal(t, uint64(1), stats.LoadSuccessCount)
	assert.Equal(t, uint64(1), stats.LoadFailureCount)

	assert.Equal(t, uint64(2), recorder.hits.Load(), "the caller's recorder must still see every event")
	assert.Equal(t, uint64(4), recorder.misses.Load())
	assert.Equal(t, uint64(1), recorder.loadSuccesses.Load())
	assert.Equal(t, uint64(1), recorder.loadFailures.Load())
}

func TestStats_ReportsNoRatioBeforeAnyRequest(t *testing.T) {
	t.Parallel()

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 100})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	_, hasRequests := cache.Stats().HitRatio()
	assert.False(t, hasRequests, "an idle cache has no hit ratio, and must not invent one")
}
