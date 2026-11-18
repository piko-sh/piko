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
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	persistedKey = "article"
	persistedTTL = time.Hour
)

func TestOtterAdapter_PersistedTTLSurvivesRestart(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		write            func(ctx context.Context, cache *OtterAdapter[string, testArticle]) error
		name             string
		wantNeverExpires bool
	}{
		{
			name: "SetWithTTL",
			write: func(ctx context.Context, cache *OtterAdapter[string, testArticle]) error {
				return cache.SetWithTTL(ctx, persistedKey, testArticle{Title: "first"}, persistedTTL)
			},
		},
		{
			name: "ComputeWithTTL",
			write: func(ctx context.Context, cache *OtterAdapter[string, testArticle]) error {
				_, _, err := cache.ComputeWithTTL(ctx, persistedKey, func(testArticle, bool) cache_dto.ComputeResult[testArticle] {
					return cache_dto.ComputeResult[testArticle]{
						Value:  testArticle{Title: "computed"},
						Action: cache_dto.ComputeActionSet,
						TTL:    persistedTTL,
					}
				})
				return err
			},
		},
		{
			name: "SetExpiresAfter",
			write: func(ctx context.Context, cache *OtterAdapter[string, testArticle]) error {
				if err := cache.Set(ctx, persistedKey, testArticle{Title: "first"}); err != nil {
					return err
				}
				return cache.SetExpiresAfter(ctx, persistedKey, persistedTTL)
			},
		},
		{
			name: "Set after SetWithTTL keeps the TTL",
			write: func(ctx context.Context, cache *OtterAdapter[string, testArticle]) error {
				if err := cache.SetWithTTL(ctx, persistedKey, testArticle{Title: "first"}, persistedTTL); err != nil {
					return err
				}
				return cache.Set(ctx, persistedKey, testArticle{Title: "second"})
			},
		},
		{
			name: "SetWithTTL with NoExpiry removes the TTL",
			write: func(ctx context.Context, cache *OtterAdapter[string, testArticle]) error {
				if err := cache.SetWithTTL(ctx, persistedKey, testArticle{Title: "first"}, persistedTTL); err != nil {
					return err
				}
				return cache.SetWithTTL(ctx, persistedKey, testArticle{Title: "second"}, cache_dto.NoExpiry)
			},
			wantNeverExpires: true,
		},
	}

	for _, testCase := range testCases {
		for _, checkpoint := range []bool{false, true} {
			name := testCase.name + " recovered from the log"
			if checkpoint {
				name = testCase.name + " recovered from a snapshot"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				ctx := t.Context()
				directory := t.TempDir()
				writerClock := newTTLTestClock()
				writer := newPerEntryAdapter(t, writerClock, withPersistence(directory, false))
				require.NoError(t, testCase.write(ctx, writer))
				if checkpoint {
					require.NoError(t, writer.Checkpoint(ctx))
				}

				readerClock := clock.NewMockClock(writerClock.Now().Add(30 * time.Minute))
				reader := newPerEntryAdapter(t, readerClock, withPersistence(directory, true))

				_, found, err := reader.GetIfPresent(ctx, persistedKey)
				require.NoError(t, err)
				require.True(t, found, "the entry still has 30 minutes left")

				if testCase.wantNeverExpires {
					readerClock.Advance(tenYears)
				} else {
					readerClock.Advance(31 * time.Minute)
				}
				_, found, err = reader.GetIfPresent(ctx, persistedKey)
				require.NoError(t, err)
				assert.Equal(t, testCase.wantNeverExpires, found,
					"the restored entry keeps the expiry it had before the restart")
			})
		}
	}
}

func TestOtterAdapter_RestartWithoutTTLKeepsTheEntry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	directory := t.TempDir()
	writerClock := newTTLTestClock()
	writer := newPerEntryAdapter(t, writerClock, withPersistence(directory, false))
	require.NoError(t, writer.Set(ctx, persistedKey, testArticle{Title: "lasting"}, "news"))
	require.NoError(t, writer.Close(ctx))

	readerClock := clock.NewMockClock(writerClock.Now().Add(tenYears))
	reader := newPerEntryAdapter(t, readerClock, withPersistence(directory, false))

	value, found, err := reader.GetIfPresent(ctx, persistedKey)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "lasting", value.Title)
	removed, err := reader.InvalidateByTags(ctx, "news")
	require.NoError(t, err)
	assert.Equal(t, 1, removed, "the restored entry keeps its tags")
}

func TestPopulateCacheFromState(t *testing.T) {
	t.Parallel()

	mockClock := newTTLTestClock()
	cache := newPerEntryAdapter[string](t, mockClock)
	nowNano := mockClock.Now().UnixNano()

	populateCacheFromState(cache, map[string]entryData[string]{
		"policy":   {Value: "a", ExpiresAt: 0},
		"lasting":  {Value: "b", ExpiresAt: math.MaxInt64},
		"expired":  {Value: "c", ExpiresAt: nowNano - 1, Tags: []string{"gone"}},
		"expiring": {Value: "d", ExpiresAt: nowNano + int64(time.Minute), Tags: []string{"kept"}},
	}, nowNano)

	ctx := t.Context()
	wantExpiries := map[string]int64{
		"policy":   math.MaxInt64,
		"lasting":  math.MaxInt64,
		"expiring": nowNano + int64(time.Minute),
	}
	for key, wantExpiry := range wantExpiries {
		entry, found, err := cache.ProbeEntry(ctx, key)
		require.NoError(t, err)
		require.True(t, found, key)
		assert.Equal(t, wantExpiry, entry.ExpiresAtNano, key)
	}
	_, found, err := cache.ProbeEntry(ctx, "expired")
	require.NoError(t, err)
	assert.False(t, found, "an entry whose expiry passed while the cache was down is not restored")
	assert.Nil(t, cache.tagIndex.Get("gone"), "a skipped entry leaves no tags behind")
	assert.Equal(t, map[string]struct{}{"expiring": {}}, cache.tagIndex.Get("kept"))
}

func TestRestoreRecoveredEntry_KeepsAnEntryNearTheEndOfRepresentableTime(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))
	cache := newPerEntryAdapter[string](t, mockClock)
	recoveryStartNano := mockClock.Now().Add(-time.Second).UnixNano()

	restored := restoreRecoveredEntry(cache, persistedKey, entryData[string]{Value: "lasting", ExpiresAt: math.MaxInt64 - 1}, recoveryStartNano)

	require.True(t, restored)
	entry, found, err := cache.ProbeEntry(t.Context(), persistedKey)
	require.NoError(t, err)
	require.True(t, found, "the clock moves on during recovery, and the remaining TTL must not wrap the expiry into the past")
	assert.Greater(t, entry.ExpiresAtNano, mockClock.Now().UnixNano())
	assert.Equal(t, "lasting", entry.Value)
}

func TestPopulateCacheFromState_WithoutExpirySupportKeepsEveryEntry(t *testing.T) {
	t.Parallel()

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 16})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })
	nowNano := time.Now().UnixNano()

	populateCacheFromState(cache, map[string]entryData[string]{
		"expiring": {Value: "a", ExpiresAt: nowNano + int64(time.Minute)},
		"lasting":  {Value: "b", ExpiresAt: math.MaxInt64},
	}, nowNano)

	assert.Equal(t, 2, cache.EstimatedSize())
}

func TestBuildExpiryCalculator_LeavesACacheWithoutExpiryAlone(t *testing.T) {
	t.Parallel()

	assert.Nil(t, buildExpiryCalculator[string, string](nil, &expiryOverrides[string]{}),
		"otter must not track expiry for a cache built without an expiration option")
}

func withPersistence(directory string, readOnly bool) func(*cache_domain.CacheBuilder[string, testArticle]) {
	return func(builder *cache_domain.CacheBuilder[string, testArticle]) {
		walConfig := wal_domain.DefaultConfig(directory)
		walConfig.SyncMode = wal_domain.SyncModeEveryWrite
		walConfig.EnableCompression = false
		builder.Options(PersistenceConfig[string, testArticle]{
			Enabled:    true,
			ReadOnly:   readOnly,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		})
	}
}
