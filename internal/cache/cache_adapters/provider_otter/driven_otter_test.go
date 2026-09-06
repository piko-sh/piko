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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestTagIndex_Invalidate_UnlinksKeyFromUnrequestedTags(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})

	invalidated := index.Invalidate([]string{"alpha"})

	if len(invalidated) != 1 || invalidated[0] != "k1" {
		t.Fatalf("Invalidate([alpha]) = %v, want [k1]", invalidated)
	}
	if remaining := index.Get("beta"); len(remaining) != 0 {
		t.Errorf("Get(\"beta\") = %v, want empty: the key is stranded in a tag it was invalidated out of", remaining)
	}
	if tags := index.GetTags("k1"); len(tags) != 0 {
		t.Errorf("GetTags(\"k1\") = %v, want empty", tags)
	}
}

func TestTagIndex_Invalidate_DoesNotReturnAKeyRetaggedSinceItsRemoval(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})
	index.Invalidate([]string{"alpha"})

	index.Add("k1", []string{"gamma"})

	if invalidated := index.Invalidate([]string{"beta"}); len(invalidated) != 0 {
		t.Errorf("Invalidate([beta]) = %v, want empty: k1 no longer carries beta", invalidated)
	}
	if remaining := index.Get("gamma"); len(remaining) != 1 {
		t.Errorf("Get(\"gamma\") = %v, want k1 still present", remaining)
	}
}

func TestTagIndex_Invalidate_LeavesUnrelatedKeysUntouched(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "shared"})
	index.Add("k2", []string{"shared"})

	index.Invalidate([]string{"alpha"})

	remaining := index.Get("shared")
	if len(remaining) != 1 {
		t.Fatalf("Get(\"shared\") = %v, want exactly k2", remaining)
	}
	if _, ok := remaining["k2"]; !ok {
		t.Errorf("Get(\"shared\") = %v, want k2", remaining)
	}
}

func TestTagIndex_Invalidate_EmptiesEveryBucketItDrains(t *testing.T) {
	t.Parallel()

	index := NewTagIndex[string]()
	index.Add("k1", []string{"alpha", "beta"})
	index.Add("k2", []string{"beta", "gamma"})

	index.Invalidate([]string{"alpha", "gamma"})

	if remaining := index.Get("beta"); len(remaining) != 0 {
		t.Errorf("Get(\"beta\") = %v, want empty: both keys carried beta and both were invalidated", remaining)
	}
}

type loadContext struct {
	cause   error
	carrier *daemon_dto.PikoRequestCtx
}

func recordLoadContext(ctx context.Context) loadContext {
	return loadContext{cause: context.Cause(ctx), carrier: daemon_dto.PikoRequestCtxFromContext(ctx)}
}

func TestRefresh_LoadsUnderADetachedContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		refresh func(ctx context.Context, cache *OtterAdapter[string, string], loads chan<- loadContext)
		name    string
	}{
		{
			name: "single key",
			refresh: func(ctx context.Context, cache *OtterAdapter[string, string], loads chan<- loadContext) {
				result := cache.Refresh(ctx, "key", loaderFunc{load: func(loadCtx context.Context, _ string) (string, error) {
					loads <- recordLoadContext(loadCtx)
					return "fresh", nil
				}})
				go func() { <-result }()
			},
		},
		{
			name: "bulk",
			refresh: func(ctx context.Context, cache *OtterAdapter[string, string], loads chan<- loadContext) {
				cache.BulkRefresh(ctx, []string{"key"}, cache_dto.BulkLoaderFunc[string, string](
					func(loadCtx context.Context, keys []string) (map[string]string, error) {
						loads <- recordLoadContext(loadCtx)
						return map[string]string{keys[0]: "fresh"}, nil
					}))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cache := newStringCache(t, cache_dto.Options[string, string]{
				MaximumEntries:    100,
				RefreshCalculator: alwaysRefreshCalculator[string, string]{},
			})
			require.NoError(t, cache.Set(context.Background(), "key", "stale"))

			pctx := daemon_dto.AcquirePikoRequestCtx()
			pctx.ClientIP = "10.0.0.1"
			requestCtx, cancelRequest := context.WithCancelCause(
				daemon_dto.WithPikoRequestCtx(context.Background(), pctx))
			cancelRequest(errors.New("request finished"))

			loads := make(chan loadContext, 1)
			tc.refresh(requestCtx, cache, loads)
			var load loadContext
			select {
			case load = <-loads:
			case <-time.After(5 * time.Second):
				require.FailNow(t, "the refresh never ran its loader")
			}
			daemon_dto.ReleasePikoRequestCtx(pctx)

			require.NoError(t, load.cause, "a refresh must not inherit the request's cancellation")
			require.NotNil(t, load.carrier)
			assert.NotSame(t, pctx, load.carrier, "a refresh must not read the pooled carrier")
			assert.Equal(t, "10.0.0.1", load.carrier.ClientIP)
			require.Eventually(t, func() bool {
				value, found, err := cache.GetIfPresent(context.Background(), "key")
				return err == nil && found && value == "fresh"
			}, time.Second, time.Millisecond)
		})
	}
}

type scriptedWAL struct {
	wal_domain.WAL[string, string]
	appendErr  error
	closeErr   error
	appended   []wal_domain.Operation
	closeCalls int
}

func (w *scriptedWAL) Append(_ context.Context, entry wal_domain.Entry[string, string]) error {
	w.appended = append(w.appended, entry.Operation)
	return w.appendErr
}

func (w *scriptedWAL) Close() error {
	w.closeCalls++
	return w.closeErr
}

func TestOtterAdapter_CloseReportsTheFirstOutcomeOnEveryCall(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		closeErr error
		name     string
	}{
		{name: "a failed close is reported again on later calls", closeErr: errors.New("disk detached")},
		{name: "a clean close stays clean on later calls", closeErr: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 16})
			require.NoError(t, err)
			cache, ok := adapter.(*OtterAdapter[string, string])
			require.True(t, ok)
			wal := &scriptedWAL{closeErr: testCase.closeErr}
			cache.wal = wal
			cache.walEnabled = true

			first := cache.Close(context.Background())
			second := cache.Close(context.Background())

			if testCase.closeErr == nil {
				assert.NoError(t, first)
				assert.NoError(t, second)
			} else {
				assert.ErrorIs(t, first, testCase.closeErr)
				assert.ErrorIs(t, second, testCase.closeErr)
			}
			assert.Equal(t, 1, wal.closeCalls, "the persistence close path runs exactly once")
		})
	}
}

func TestOtterAdapter_InvalidateAllLogsTheClear(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		appendErr error
		name      string
	}{
		{name: "a recorded clear empties the cache", appendErr: nil},
		{name: "a clear the log rejects still empties the cache", appendErr: errors.New("disk full")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 16})
			require.NoError(t, err)
			cache, ok := adapter.(*OtterAdapter[string, string])
			require.True(t, ok)
			ctx := context.Background()
			require.NoError(t, cache.Set(ctx, "page", "rendered"))
			wal := &scriptedWAL{appendErr: testCase.appendErr}
			cache.wal = wal
			cache.walEnabled = true

			require.NoError(t, cache.InvalidateAll(ctx))

			assert.Equal(t, []wal_domain.Operation{wal_domain.OpClear}, wal.appended)
			_, found, err := cache.GetIfPresent(ctx, "page")
			require.NoError(t, err)
			assert.False(t, found)
		})
	}
}
