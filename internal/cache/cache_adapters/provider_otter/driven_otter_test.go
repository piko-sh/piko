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
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/cache/cache_test/conformance"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/testutil/allocbudget"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	ttlTestKey = "key"
	ttlTestValue = "value"
	ttlTestNamespace = "ttl-test"
	ttlWarningMessage = "Cache ignores per-entry TTLs because it was built without expiry support"
	tenYears = 10 * 365 * 24 * time.Hour
)

type ttlTestCache = cache_domain.ProviderPort[string, string]

type ttlWrite func(ctx context.Context, cache ttlTestCache, mockClock *clock.MockClock) error

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
	entries    []wal_domain.Entry[string, string]
	closeCalls int
}

func (w *scriptedWAL) Append(_ context.Context, entry wal_domain.Entry[string, string]) error {
	w.appended = append(w.appended, entry.Operation)
	if w.appendErr == nil {
		w.entries = append(w.entries, entry)
	}
	return w.appendErr
}

func (*scriptedWAL) EntryCount() int { return 0 }

func (*scriptedWAL) Size() int64 { return 0 }

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
		{name: "a clear the log rejects fails and leaves the cache as it was", appendErr: errors.New("disk full")},
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

			clearErr := cache.InvalidateAll(ctx)

			assert.Equal(t, []wal_domain.Operation{wal_domain.OpClear}, wal.appended)
			_, found, err := cache.GetIfPresent(ctx, "page")
			require.NoError(t, err)
			if testCase.appendErr != nil {
				require.ErrorIs(t, clearErr, testCase.appendErr, "the log's refusal is reported")
				assert.True(t, found, "a clear the log never recorded must not empty the cache")
				return
			}
			require.NoError(t, clearErr)
			assert.False(t, found)
		})
	}
}

func TestOtterAdapter_PerEntryTTL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		writes      []ttlWrite
		advance     time.Duration
		wantPresent bool
	}{
		{
			name:        "SetWithTTL keeps the entry until its TTL",
			writes:      []ttlWrite{setWithTTL(time.Minute)},
			advance:     time.Minute - time.Second,
			wantPresent: true,
		},
		{
			name:        "SetWithTTL drops the entry once its TTL passes",
			writes:      []ttlWrite{setWithTTL(time.Minute)},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "SetWithTTL without a TTL keeps the current expiry",
			writes:      []ttlWrite{setWithTTL(time.Minute), setWithTTL(0)},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "SetWithTTL with NoExpiry removes the expiry",
			writes:      []ttlWrite{setWithTTL(time.Minute), setWithTTL(cache_dto.NoExpiry)},
			advance:     tenYears,
			wantPresent: true,
		},
		{
			name:        "SetWithTTL caps a TTL that would overflow the clock",
			writes:      []ttlWrite{setWithTTL(time.Duration(math.MaxInt64 - 1))},
			advance:     365 * 24 * time.Hour,
			wantPresent: true,
		},
		{
			name:        "SetWithTTL keeps an entry with a sub-second TTL until it passes",
			writes:      []ttlWrite{setWithTTL(500 * time.Millisecond)},
			advance:     499 * time.Millisecond,
			wantPresent: true,
		},
		{
			name:        "SetWithTTL drops an entry with a sub-second TTL once it passes",
			writes:      []ttlWrite{setWithTTL(500 * time.Millisecond)},
			advance:     500 * time.Millisecond,
			wantPresent: false,
		},
		{
			name:        "ComputeWithTTL drops a created entry once its TTL passes",
			writes:      []ttlWrite{computeWithTTL(time.Minute)},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "ComputeWithTTL update without a TTL keeps the original expiry",
			writes:      []ttlWrite{computeWithTTL(time.Minute), advanceBy(30 * time.Second), computeWithTTL(0)},
			advance:     30 * time.Second,
			wantPresent: false,
		},
		{
			name:        "ComputeWithTTL update with a TTL replaces the expiry",
			writes:      []ttlWrite{computeWithTTL(time.Minute), advanceBy(30 * time.Second), computeWithTTL(time.Minute)},
			advance:     45 * time.Second,
			wantPresent: true,
		},
		{
			name:        "ComputeWithTTL with NoExpiry removes the expiry",
			writes:      []ttlWrite{computeWithTTL(time.Minute), computeWithTTL(cache_dto.NoExpiry)},
			advance:     tenYears,
			wantPresent: true,
		},
		{
			name:        "a noop compute does not apply its TTL to an entry without one",
			writes:      []ttlWrite{set(), computeAction(cache_dto.ComputeActionNoop, time.Minute)},
			advance:     2 * time.Minute,
			wantPresent: true,
		},
		{
			name:        "a noop compute does not replace an existing expiry",
			writes:      []ttlWrite{setWithTTL(time.Minute), computeAction(cache_dto.ComputeActionNoop, time.Hour)},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "a delete compute removes the entry whatever TTL it returns",
			writes:      []ttlWrite{set(), computeAction(cache_dto.ComputeActionDelete, time.Hour)},
			advance:     0,
			wantPresent: false,
		},
		{
			name:        "SetExpiresAfter drops an existing entry once its TTL passes",
			writes:      []ttlWrite{set(), setExpiresAfter(time.Minute)},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "SetExpiresAfter zero neither expires nor deletes the entry",
			writes:      []ttlWrite{set(), setExpiresAfter(0)},
			advance:     tenYears,
			wantPresent: true,
		},
		{
			name:        "SetExpiresAfter negative keeps the current expiry",
			writes:      []ttlWrite{setWithTTL(time.Minute), setExpiresAfter(-time.Second)},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "SetExpiresAfter with NoExpiry removes the expiry",
			writes:      []ttlWrite{setWithTTL(time.Minute), setExpiresAfter(cache_dto.NoExpiry)},
			advance:     tenYears,
			wantPresent: true,
		},
		{
			name:        "SetExpiresAfter never creates a missing entry",
			writes:      []ttlWrite{setExpiresAfter(time.Minute)},
			advance:     0,
			wantPresent: false,
		},
		{
			name:        "Set without a TTL does not expire",
			writes:      []ttlWrite{set()},
			advance:     tenYears,
			wantPresent: true,
		},
		{
			name:        "Set keeps the expiry of an entry that has a TTL",
			writes:      []ttlWrite{setWithTTL(time.Minute), set()},
			advance:     time.Minute,
			wantPresent: false,
		},
		{
			name:        "Set over an expired entry stores a value that does not expire",
			writes:      []ttlWrite{setWithTTL(time.Minute), advanceBy(2 * time.Minute), set()},
			advance:     24 * time.Hour,
			wantPresent: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := newTTLTestClock()
			cache := newPerEntryAdapter[string](t, mockClock)

			for _, write := range testCase.writes {
				require.NoError(t, write(ctx, cache, mockClock))
			}
			mockClock.Advance(testCase.advance)

			_, present, err := cache.GetIfPresent(ctx, ttlTestKey)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantPresent, present)
		})
	}
}

func TestOtterAdapter_PerEntryExpiryConformance(t *testing.T) {
	t.Parallel()

	conformance.RunPerEntryExpirySuite(t, conformance.PerEntryConfig{
		NewHarness: func(t *testing.T) conformance.PerEntryHarness {
			mockClock := newTTLTestClock()
			return conformance.PerEntryHarness{
				Cache:       newPerEntryAdapter[string](t, mockClock),
				AdvanceTime: mockClock.Advance,
				Clock:       mockClock,
			}
		},
	})
}

func TestOtterAdapter_TTLWithoutExpirySupport(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		write ttlWrite
		name  string
	}{
		{
			name:  "SetWithTTL",
			write: setWithTTL(time.Minute),
		},
		{
			name:  "ComputeWithTTL",
			write: computeWithTTL(time.Minute),
		},
		{
			name: "SetExpiresAfter",
			write: func(ctx context.Context, cache ttlTestCache, mockClock *clock.MockClock) error {
				if err := set()(ctx, cache, mockClock); err != nil {
					return err
				}
				return setExpiresAfter(time.Minute)(ctx, cache, mockClock)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, logOutput := newCapturingContext(t)
			mockClock := newTTLTestClock()
			cache := newTTLTestCache(t, cache_dto.Options[string, string]{
				Namespace:      ttlTestNamespace,
				MaximumEntries: 100,
				Clock:          mockClock,
			})

			require.NoError(t, testCase.write(ctx, cache, mockClock))
			require.NoError(t, testCase.write(ctx, cache, mockClock))

			assert.Equal(t, 1, strings.Count(logOutput.String(), ttlWarningMessage),
				"the warning is logged once per cache, got:\n%s", logOutput.String())
			assert.Contains(t, logOutput.String(), ttlTestNamespace)
		})
	}
}

func TestOtterAdapter_NoExpiryWithoutExpirySupportStoresQuietly(t *testing.T) {
	t.Parallel()

	ctx, logOutput := newCapturingContext(t)
	cache := newTTLTestCache(t, cache_dto.Options[string, string]{
		Namespace:      ttlTestNamespace,
		MaximumEntries: 100,
	})

	require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, ttlTestValue, cache_dto.NoExpiry))
	require.NoError(t, cache.SetExpiresAfter(ctx, ttlTestKey, cache_dto.NoExpiry))

	value, present, err := cache.GetIfPresent(ctx, ttlTestKey)
	require.NoError(t, err)
	assert.True(t, present)
	assert.Equal(t, ttlTestValue, value)
	assert.NotContains(t, logOutput.String(), ttlWarningMessage,
		"asking for no expiry on a cache that never expires entries needs no warning")
}

func TestOtterAdapter_ConcurrentTTLWritesKeepTheirExpiry(t *testing.T) {
	t.Parallel()

	const keyCount = 4096

	testCases := []struct {
		ttlWrite    func(ctx context.Context, cache ttlTestCache, key string) error
		plainWrite  func(ctx context.Context, cache ttlTestCache, key string) error
		name        string
		prepopulate bool
	}{
		{
			name:       "SetWithTTL racing Set",
			ttlWrite:   setKeyWithTTL,
			plainWrite: setKey,
		},
		{
			name:       "SetWithTTL racing ComputeWithTTL without a TTL",
			ttlWrite:   setKeyWithTTL,
			plainWrite: incrementKeyWithoutTTL,
		},
		{
			name:       "fixed-window increments racing to create the entry",
			ttlWrite:   incrementKeyWithTTLOnCreate,
			plainWrite: incrementKeyWithTTLOnCreate,
		},
		{
			name:        "SetExpiresAfter racing Set",
			ttlWrite:    expireKey,
			plainWrite:  setKey,
			prepopulate: true,
		},
		{
			name:        "SetExpiresAfter racing ComputeWithTTL without a TTL",
			ttlWrite:    expireKey,
			plainWrite:  incrementKeyWithoutTTL,
			prepopulate: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cache := newPerEntryAdapter[string](t, newTTLTestClock())
			keys := make([]string, keyCount)
			for index := range keys {
				keys[index] = fmt.Sprintf("key-%d", index)
				if testCase.prepopulate {
					require.NoError(t, cache.Set(ctx, keys[index], ttlTestValue))
				}
			}

			start := make(chan struct{})
			var group sync.WaitGroup
			errs := make(chan error, 2*keyCount)
			for _, key := range keys {
				group.Go(func() {
					<-start
					errs <- testCase.ttlWrite(ctx, cache, key)
				})
				group.Go(func() {
					<-start
					errs <- testCase.plainWrite(ctx, cache, key)
				})
			}
			close(start)
			group.Wait()
			close(errs)
			for err := range errs {
				require.NoError(t, err)
			}

			lost := 0
			for _, key := range keys {
				entry, found, err := cache.ProbeEntry(ctx, key)
				require.NoError(t, err)
				require.True(t, found, "key %s vanished", key)
				if entry.ExpiresAtNano == math.MaxInt64 {
					lost++
				}
			}
			assert.Zero(t, lost, "%d of %d keys lost the TTL a concurrent write set", lost, keyCount)
		})
	}
}

func TestOtterAdapter_ComputeLogsTheWrite(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		write         func(ctx context.Context, cache *OtterAdapter[string, string]) error
		wantExpiresAt func(start time.Time) int64
		name          string
		wantOps       []wal_domain.Operation
		existing      bool
	}{
		{
			name: "ComputeWithTTL logs the value with its absolute expiry",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return computeWithTTL(time.Minute)(ctx, cache, nil)
			},
			wantOps:       []wal_domain.Operation{wal_domain.OpSet},
			wantExpiresAt: expiresAfter(time.Minute),
		},
		{
			name: "SetWithTTL with NoExpiry logs an entry that never expires",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return setWithTTL(cache_dto.NoExpiry)(ctx, cache, nil)
			},
			wantOps:       []wal_domain.Operation{wal_domain.OpSet},
			wantExpiresAt: func(time.Time) int64 { return math.MaxInt64 },
		},
		{
			name: "SetExpiresAfter logs the new expiry",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return setExpiresAfter(2*time.Minute)(ctx, cache, nil)
			},
			existing:      true,
			wantOps:       []wal_domain.Operation{wal_domain.OpSet},
			wantExpiresAt: expiresAfter(2 * time.Minute),
		},
		{
			name: "Set under per-entry expiry logs the expiry the entry keeps",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return set()(ctx, cache, nil)
			},
			existing:      true,
			wantOps:       []wal_domain.Operation{wal_domain.OpSet},
			wantExpiresAt: expiresAfter(time.Minute),
		},
		{
			name: "ComputeIfPresent logs the expiry the entry keeps",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				_, _, err := cache.ComputeIfPresent(ctx, ttlTestKey, func(string) (string, cache_dto.ComputeAction) {
					return "updated", cache_dto.ComputeActionSet
				})
				return err
			},
			existing:      true,
			wantOps:       []wal_domain.Operation{wal_domain.OpSet},
			wantExpiresAt: expiresAfter(time.Minute),
		},
		{
			name: "ComputeIfAbsent logs a new entry for the policy to decide",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				_, _, err := cache.ComputeIfAbsent(ctx, ttlTestKey, func() string { return ttlTestValue })
				return err
			},
			wantOps:       []wal_domain.Operation{wal_domain.OpSet},
			wantExpiresAt: func(time.Time) int64 { return 0 },
		},
		{
			name: "ComputeIfAbsent over an existing entry logs nothing",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				_, _, err := cache.ComputeIfAbsent(ctx, ttlTestKey, func() string { return "unused" })
				return err
			},
			existing: true,
			wantOps:  nil,
		},
		{
			name: "a delete compute of an existing entry logs the removal",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return computeAction(cache_dto.ComputeActionDelete, 0)(ctx, cache, nil)
			},
			existing: true,
			wantOps:  []wal_domain.Operation{wal_domain.OpDelete},
		},
		{
			name: "a delete compute of a missing entry logs nothing",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return computeAction(cache_dto.ComputeActionDelete, 0)(ctx, cache, nil)
			},
			wantOps: nil,
		},
		{
			name: "a noop compute logs nothing",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return computeAction(cache_dto.ComputeActionNoop, time.Minute)(ctx, cache, nil)
			},
			existing: true,
			wantOps:  nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := newTTLTestClock()
			start := mockClock.Now()
			cache := newPerEntryAdapter[string](t, mockClock)
			if testCase.existing {
				require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, ttlTestValue, time.Minute))
			}
			wal := &scriptedWAL{}
			cache.wal = wal
			cache.walEnabled = true

			require.NoError(t, testCase.write(ctx, cache))

			assert.Equal(t, testCase.wantOps, wal.appended)
			if testCase.wantExpiresAt != nil {
				require.Len(t, wal.entries, 1)
				assert.Equal(t, testCase.wantExpiresAt(start), wal.entries[0].ExpiresAt)
				assert.Equal(t, start.UnixNano(), wal.entries[0].Timestamp,
					"the log stamps entries with the cache's clock")
			}
		})
	}
}

func TestOtterAdapter_ComputeLeavesTheCacheUnchangedWhenTheLogFails(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		write func(ctx context.Context, cache *OtterAdapter[string, string]) error
		name  string
	}{
		{
			name: "ComputeWithTTL",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return computeWithTTL(time.Hour)(ctx, cache, nil)
			},
		},
		{
			name: "SetWithTTL",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return setWithTTL(time.Hour)(ctx, cache, nil)
			},
		},
		{
			name: "Compute delete",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return computeAction(cache_dto.ComputeActionDelete, 0)(ctx, cache, nil)
			},
		},
		{
			name: "ComputeIfPresent",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				_, _, err := cache.ComputeIfPresent(ctx, ttlTestKey, func(string) (string, cache_dto.ComputeAction) {
					return "updated", cache_dto.ComputeActionSet
				})
				return err
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := newTTLTestClock()
			cache := newPerEntryAdapter[string](t, mockClock)
			require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, "original", time.Minute))
			diskFull := errors.New("disk full")
			cache.wal = &scriptedWAL{appendErr: diskFull}
			cache.walEnabled = true

			err := testCase.write(ctx, cache)

			require.ErrorIs(t, err, diskFull)
			entry, found, probeErr := cache.ProbeEntry(ctx, ttlTestKey)
			require.NoError(t, probeErr)
			require.True(t, found, "a write the log refused must not remove the entry")
			assert.Equal(t, "original", entry.Value)
			assert.Equal(t, mockClock.Now().Add(time.Minute).UnixNano(), entry.ExpiresAtNano,
				"a write the log refused must not change the expiry")
		})
	}
}

func TestOtterAdapter_ComputeLeavesTheCacheUnchangedWhenComputeIfAbsentCannotLog(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cache := newPerEntryAdapter[string](t, newTTLTestClock())
	diskFull := errors.New("disk full")
	cache.wal = &scriptedWAL{appendErr: diskFull}
	cache.walEnabled = true

	_, _, err := cache.ComputeIfAbsent(ctx, ttlTestKey, func() string { return ttlTestValue })

	require.ErrorIs(t, err, diskFull)
	_, found, probeErr := cache.ProbeEntry(ctx, ttlTestKey)
	require.NoError(t, probeErr)
	assert.False(t, found, "a value the log refused must not be stored")
}

func TestOtterAdapter_ComputeQueuesOnADeferredLog(t *testing.T) {
	t.Parallel()

	waitFailed := errors.New("sync failed")
	queueFailed := errors.New("log closed")

	testCases := []struct {
		enqueueErr error
		waitErr    error
		wantErr    error
		name       string
		wantStored bool
	}{
		{
			name:       "a queued write is stored and waited for",
			wantStored: true,
		},
		{
			name:       "a failed wait is reported but the cache keeps the write",
			waitErr:    waitFailed,
			wantErr:    waitFailed,
			wantStored: true,
		},
		{
			name:       "a write the log cannot queue is not stored",
			enqueueErr: queueFailed,
			wantErr:    queueFailed,
			wantStored: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := newTTLTestClock()
			cache := newPerEntryAdapter[string](t, mockClock)
			wal := &deferredWAL{enqueueErr: testCase.enqueueErr, waitErr: testCase.waitErr}
			cache.wal = wal
			cache.walEnabled = true

			err := computeWithTTL(time.Minute)(ctx, cache, nil)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			_, found, probeErr := cache.ProbeEntry(ctx, ttlTestKey)
			require.NoError(t, probeErr)
			assert.Equal(t, testCase.wantStored, found)
			if testCase.enqueueErr == nil {
				require.Len(t, wal.queued, 1)
				assert.Equal(t, mockClock.Now().Add(time.Minute).UnixNano(), wal.queued[0].ExpiresAt)
				assert.Equal(t, 1, wal.waits, "the write waits for its entry once, outside the lock")
			}
		})
	}
}

func TestOtterAdapter_ComputeJoinsAnOpenLogBatch(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cache := newPerEntryAdapter[string](t, newTTLTestClock())
	wal := &scriptedWAL{}
	cache.wal = wal
	cache.walEnabled = true

	batch := cache.BeginLogBatch()
	_, _, err := batch.ComputeWithTTL(ctx, ttlTestKey, func(string, bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{Value: ttlTestValue, Action: cache_dto.ComputeActionSet, TTL: time.Minute}
	})
	require.NoError(t, err)
	assert.Empty(t, wal.appended, "a write inside a batch waits for the commit")
	require.NoError(t, batch.CommitLogBatch(ctx))

	assert.Equal(t, []wal_domain.Operation{wal_domain.OpSet}, wal.appended)
}

func TestOtterAdapter_ComputeKeepsTheSearchIndexesCurrent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider, err := OtterProviderFactory(cache_dto.Options[string, Product]{
		MaximumEntries: 100,
		SearchSchema:   cache_dto.NewSearchSchema(cache_dto.SortableNumericField("Stock")),
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, Product])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })
	require.NoError(t, cache.Set(ctx, "p1", Product{ID: "p1", Stock: 1}))

	_, _, err = cache.Compute(ctx, "p1", func(oldValue Product, _ bool) (Product, cache_dto.ComputeAction) {
		oldValue.Stock = 7
		return oldValue, cache_dto.ComputeActionSet
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"p1"}, queryStock(t, cache, 7), "a computed value is searchable")
	assert.Empty(t, queryStock(t, cache, 1), "the value a compute replaced is no longer searchable")

	_, _, err = cache.Compute(ctx, "p1", func(oldValue Product, _ bool) (Product, cache_dto.ComputeAction) {
		return oldValue, cache_dto.ComputeActionDelete
	})
	require.NoError(t, err)

	assert.Empty(t, queryStock(t, cache, 7), "a value a compute removed is no longer searchable")
}

func TestOtterAdapter_SnapshotEntriesKeepTheirExpiry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClock := newTTLTestClock()
	cache := newPerEntryAdapter[string](t, mockClock)
	require.NoError(t, cache.SetWithTTL(ctx, "expiring", ttlTestValue, time.Hour))
	require.NoError(t, cache.Set(ctx, "lasting", ttlTestValue))

	expiries := make(map[string]int64)
	for _, entry := range cache.SnapshotEntries() {
		assert.Equal(t, mockClock.Now().UnixNano(), entry.Timestamp)
		expiries[entry.Key] = entry.ExpiresAt
	}

	assert.Equal(t, map[string]int64{
		"expiring": mockClock.Now().Add(time.Hour).UnixNano(),
		"lasting":  math.MaxInt64,
	}, expiries)
}

func TestOtterAdapter_SnapshotEntriesWithoutExpirySupportRecordNoExpiry(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 16})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })
	require.NoError(t, cache.Set(ctx, ttlTestKey, ttlTestValue, "tagged"))

	entries := cache.SnapshotEntries()

	require.Len(t, entries, 1)
	assert.Zero(t, entries[0].ExpiresAt)
	assert.Equal(t, []string{"tagged"}, entries[0].Tags)
}

func TestOtterAdapter_PlainWritesLogTheExpiryTheyKeep(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		write func(ctx context.Context, cache *OtterAdapter[string, string]) error
		name  string
	}{
		{
			name: "SetDeferred",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				wait, err := cache.SetDeferred(ctx, ttlTestKey, "updated")
				if err != nil {
					return err
				}
				return wait()
			},
		},
		{
			name: "BulkSet",
			write: func(ctx context.Context, cache *OtterAdapter[string, string]) error {
				return cache.BulkSet(ctx, map[string]string{ttlTestKey: "updated"})
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			mockClock := newTTLTestClock()
			cache := newPerEntryAdapter[string](t, mockClock)
			require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, ttlTestValue, time.Minute))
			wal := &deferredWAL{}
			cache.wal = wal
			cache.walEnabled = true

			require.NoError(t, testCase.write(ctx, cache))

			entries := append(slices.Clone(wal.queued), wal.appended...)
			require.Len(t, entries, 1)
			assert.Equal(t, mockClock.Now().Add(time.Minute).UnixNano(), entries[0].ExpiresAt,
				"a write without a TTL under per-entry expiry keeps, and logs, the expiry it had")
			entry, found, err := cache.ProbeEntry(ctx, ttlTestKey)
			require.NoError(t, err)
			require.True(t, found)
			assert.Equal(t, "updated", entry.Value)
		})
	}
}

func TestOtterAdapter_ReadsAndSettingsDelegateToOtter(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumWeight: 1 << 20,
		Weigher:       func(_ string, value string) uint32 { return uint32(len(value)) },
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })
	require.NoError(t, cache.Set(ctx, ttlTestKey, ttlTestValue))

	assert.Equal(t, map[string]string{ttlTestKey: ttlTestValue}, maps.Collect(cache.All()))
	assert.Equal(t, []string{ttlTestKey}, slices.Collect(cache.Keys()))
	assert.Equal(t, []string{ttlTestValue}, slices.Collect(cache.Values()))
	entry, found, err := cache.GetEntry(ctx, ttlTestKey)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, ttlTestValue, entry.Value)
	_, found, err = cache.GetEntry(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, found)
	loaded, err := cache.BulkGet(ctx, []string{ttlTestKey}, cache_dto.BulkLoaderFunc[string, string](
		func(context.Context, []string) (map[string]string, error) { return nil, nil }))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{ttlTestKey: ttlTestValue}, loaded)
	assert.True(t, cache.IsWeightBounded())
	assert.Equal(t, uint64(1<<20), cache.GetMaximum())
	cache.SetMaximum(1 << 21)
	assert.Equal(t, uint64(1<<21), cache.GetMaximum())
	require.Eventually(t, func() bool { return cache.WeightedSize() == uint64(len(ttlTestValue)) },
		time.Second, time.Millisecond)
	require.NoError(t, cache.SetRefreshableAfter(ctx, ttlTestKey, time.Minute))
}

type deferredWAL struct {
	wal_domain.WAL[string, string]
	enqueueErr error
	waitErr    error
	queued     []wal_domain.Entry[string, string]
	appended   []wal_domain.Entry[string, string]
	onEncode   func()
	encoded    []wal_domain.Entry[string, string]
	waits      int
	released   int
	signals    int
}

func (w *deferredWAL) Encode(entry wal_domain.Entry[string, string]) (wal_domain.EncodedEntry, error) {
	if w.onEncode != nil {
		w.onEncode()
	}
	w.encoded = append(w.encoded, entry)
	return wal_domain.NewEncodedEntry([]byte{byte(len(w.encoded) - 1)}, nil, func(*[]byte, []byte) { w.released++ }), nil
}

func (w *deferredWAL) EnqueueEncoded(_ context.Context, encoded wal_domain.EncodedEntry) (func() error, error) {
	if w.enqueueErr != nil {
		encoded.Release()
		return nil, w.enqueueErr
	}
	w.queued = append(w.queued, w.encoded[encoded.Data()[0]])
	return func() error {
		w.waits++
		return w.waitErr
	}, nil
}

func (w *deferredWAL) QueueEncoded(ctx context.Context, encoded wal_domain.EncodedEntry) (func() error, error) {
	return w.EnqueueEncoded(ctx, encoded)
}

func (w *deferredWAL) Signal() { w.signals++ }

func (w *deferredWAL) Append(_ context.Context, entry wal_domain.Entry[string, string]) error {
	w.appended = append(w.appended, entry)
	return nil
}

func (w *deferredWAL) Enqueue(_ context.Context, entry wal_domain.Entry[string, string]) (func() error, error) {
	if w.enqueueErr != nil {
		return nil, w.enqueueErr
	}
	w.queued = append(w.queued, entry)
	return func() error {
		w.waits++
		return w.waitErr
	}, nil
}

func (*deferredWAL) EntryCount() int { return 0 }

func (*deferredWAL) Size() int64 { return 0 }

func (*deferredWAL) Close() error { return nil }

func newPerEntryAdapter[V any](
	t *testing.T,
	mockClock *clock.MockClock,
	configure ...func(*cache_domain.CacheBuilder[string, V]),
) *OtterAdapter[string, V] {
	t.Helper()

	withClock := func(builder *cache_domain.CacheBuilder[string, V]) { builder.Clock(mockClock) }
	built := conformance.BuildPerEntryCache(t, OtterProviderFactory[string, V], append([]func(*cache_domain.CacheBuilder[string, V]){withClock}, configure...)...)
	adapter, ok := built.(*OtterAdapter[string, V])
	require.True(t, ok, "the otter factory builds an *OtterAdapter")

	return adapter
}

func newTTLTestCache(t *testing.T, options cache_dto.Options[string, string]) ttlTestCache {
	t.Helper()

	cache, err := OtterProviderFactory(options)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cache.Close(context.Background()) })

	return cache
}

func newTTLTestClock() *clock.MockClock {
	return clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
}

func newCapturingContext(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()

	logOutput := new(bytes.Buffer)
	ctx := logger_domain.WithLogger(t.Context(),
		logger_domain.New(slog.New(slog.NewTextHandler(logOutput, nil)), "otter-ttl-test"))
	return ctx, logOutput
}

func queryStock(t *testing.T, cache *OtterAdapter[string, Product], stock int) []string {
	t.Helper()

	result, err := cache.Query(t.Context(), &cache_dto.QueryOptions{
		Filters: []cache_dto.Filter{{Field: "Stock", Operation: cache_dto.FilterOpEq, Value: stock}},
		Limit:   cache_dto.NoLimit,
	})
	require.NoError(t, err)
	keys := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		keys = append(keys, item.Key)
	}
	return keys
}

func expiresAfter(duration time.Duration) func(time.Time) int64 {
	return func(start time.Time) int64 {
		return start.Add(duration).UnixNano()
	}
}

func set() ttlWrite {
	return func(ctx context.Context, cache ttlTestCache, _ *clock.MockClock) error {
		return cache.Set(ctx, ttlTestKey, ttlTestValue)
	}
}

func setWithTTL(ttl time.Duration) ttlWrite {
	return func(ctx context.Context, cache ttlTestCache, _ *clock.MockClock) error {
		return cache.SetWithTTL(ctx, ttlTestKey, ttlTestValue, ttl)
	}
}

func computeWithTTL(ttl time.Duration) ttlWrite {
	return computeAction(cache_dto.ComputeActionSet, ttl)
}

func computeAction(action cache_dto.ComputeAction, ttl time.Duration) ttlWrite {
	return func(ctx context.Context, cache ttlTestCache, _ *clock.MockClock) error {
		_, _, err := cache.ComputeWithTTL(ctx, ttlTestKey, func(string, bool) cache_dto.ComputeResult[string] {
			return cache_dto.ComputeResult[string]{
				Value:  ttlTestValue,
				Action: action,
				TTL:    ttl,
			}
		})
		return err
	}
}

func setExpiresAfter(ttl time.Duration) ttlWrite {
	return func(ctx context.Context, cache ttlTestCache, _ *clock.MockClock) error {
		return cache.SetExpiresAfter(ctx, ttlTestKey, ttl)
	}
}

func advanceBy(duration time.Duration) ttlWrite {
	return func(_ context.Context, _ ttlTestCache, mockClock *clock.MockClock) error {
		mockClock.Advance(duration)
		return nil
	}
}

func setKey(ctx context.Context, cache ttlTestCache, key string) error {
	return cache.Set(ctx, key, ttlTestValue)
}

func setKeyWithTTL(ctx context.Context, cache ttlTestCache, key string) error {
	return cache.SetWithTTL(ctx, key, ttlTestValue, time.Minute)
}

func expireKey(ctx context.Context, cache ttlTestCache, key string) error {
	return cache.SetExpiresAfter(ctx, key, time.Minute)
}

func incrementKeyWithTTLOnCreate(ctx context.Context, cache ttlTestCache, key string) error {
	_, _, err := cache.ComputeWithTTL(ctx, key, func(oldValue string, found bool) cache_dto.ComputeResult[string] {
		var ttl time.Duration
		if !found {
			ttl = time.Minute
		}
		return cache_dto.ComputeResult[string]{Value: oldValue + "x", Action: cache_dto.ComputeActionSet, TTL: ttl}
	})
	return err
}

func incrementKeyWithoutTTL(ctx context.Context, cache ttlTestCache, key string) error {
	_, _, err := cache.ComputeWithTTL(ctx, key, func(oldValue string, _ bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{Value: oldValue + "x", Action: cache_dto.ComputeActionSet, TTL: 0}
	})
	return err
}

type gatedWAL struct {
	wal_domain.WAL[string, string]
	entered  chan struct{}
	release  chan struct{}
	held     string
	entries  []wal_domain.Entry[string, string]
	mu       sync.Mutex
	heldOnce sync.Once
}

func (w *gatedWAL) Append(_ context.Context, entry wal_domain.Entry[string, string]) error {
	w.mu.Lock()
	w.entries = append(w.entries, entry)
	w.mu.Unlock()
	if entry.Value == w.held {
		w.heldOnce.Do(func() {
			close(w.entered)
			<-w.release
		})
	}
	return nil
}

func (*gatedWAL) EntryCount() int { return 0 }

func (*gatedWAL) Size() int64 { return 0 }

func (*gatedWAL) Close() error { return nil }

func TestOtterAdapter_PlainWriteIsLoggedInTheOrderItIsApplied(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClock := newTTLTestClock()
	cache := newPerEntryAdapter[string](t, mockClock)
	wal := &gatedWAL{entered: make(chan struct{}), release: make(chan struct{}), held: "plain"}
	cache.wal = wal
	cache.walEnabled = true

	plainDone := make(chan error, 1)
	go func() { plainDone <- cache.Set(ctx, ttlTestKey, "plain") }()
	<-wal.entered

	timedDone := make(chan error, 1)
	go func() { timedDone <- cache.SetWithTTL(ctx, ttlTestKey, "timed", time.Minute) }()
	select {
	case err := <-timedDone:
		require.NoError(t, err)
		timedDone <- nil
	case <-time.After(200 * time.Millisecond):
	}
	close(wal.release)
	require.NoError(t, <-plainDone)
	require.NoError(t, <-timedDone)

	entry, found, err := cache.ProbeEntry(ctx, ttlTestKey)
	require.NoError(t, err)
	require.True(t, found)
	wal.mu.Lock()
	defer wal.mu.Unlock()
	require.NotEmpty(t, wal.entries)
	last := wal.entries[len(wal.entries)-1]
	assert.Equal(t, entry.Value, last.Value, "the last logged write must be the value the cache holds")
	assert.Equal(t, entry.ExpiresAtNano, last.ExpiresAt, "the last logged write must carry the expiry the cache holds")
}

func TestOtterAdapter_PlainWriteEncodesItsLogEntryOutsideTheLock(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cache := newPerEntryAdapter[string](t, newTTLTestClock())
	wal := &deferredWAL{}
	wal.onEncode = func() {
		otherWrite := make(chan struct{})
		go func() {
			cache.client.Compute(ttlTestKey, func(oldValue string, _ bool) (string, otter.ComputeOp) {
				return oldValue, otter.CancelOp
			})
			close(otherWrite)
		}()
		select {
		case <-otherWrite:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "the entry was encoded while the key's lock was held")
		}
	}
	cache.wal = wal
	cache.walEnabled = true

	require.NoError(t, cache.Set(ctx, ttlTestKey, ttlTestValue))
	require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, ttlTestValue, time.Minute))

	assert.Len(t, wal.queued, 2, "both writes queue the entries encoded ahead")
	assert.Zero(t, wal.released, "an entry that was queued is not released by the cache")
}

func TestOtterAdapter_PlainWriteReencodesWhenTheStateChangedUnderneath(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClock := newTTLTestClock()
	cache := newPerEntryAdapter[string](t, mockClock)
	wal := &deferredWAL{}
	wal.onEncode = func() {
		wal.onEncode = nil
		require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, "raced", time.Hour))
	}
	cache.wal = wal
	cache.walEnabled = true

	require.NoError(t, cache.Set(ctx, ttlTestKey, ttlTestValue))

	require.Len(t, wal.queued, 2)
	assert.Equal(t, mockClock.Now().Add(time.Hour).UnixNano(), wal.queued[1].ExpiresAt,
		"a write whose expiry changed after it was encoded logs the expiry it keeps")
	assert.Equal(t, ttlTestValue, wal.queued[1].Value)
	assert.Equal(t, 1, wal.released, "the stale encoding is released")
}

func TestOtterAdapter_StalledLogQueueFailsTheWriteWithoutHanging(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cache := newPerEntryAdapter[string](t, newTTLTestClock())
	wal := &deferredWAL{enqueueErr: fmt.Errorf("queue: %w", wal_domain.ErrQueueStalled)}
	cache.wal = wal
	cache.walEnabled = true

	err := cache.Set(ctx, ttlTestKey, ttlTestValue)
	require.ErrorIs(t, err, wal_domain.ErrQueueStalled)
	_, _, err = cache.Compute(ctx, ttlTestKey, func(string, bool) (string, cache_dto.ComputeAction) {
		return ttlTestValue, cache_dto.ComputeActionSet
	})
	require.ErrorIs(t, err, wal_domain.ErrQueueStalled)

	_, found, err := cache.GetIfPresent(ctx, ttlTestKey)
	require.NoError(t, err)
	assert.False(t, found, "a write the log could not queue is not stored")
	assert.Equal(t, 2, wal.released, "each encoded entry, the plain write's and the compute's, is released when it cannot be queued")
	require.NoError(t, cache.Checkpoint(ctx), "a stalled queue holds no lock once the write has failed")
}

func TestOtterAdapter_ComputeIndexesTheValueInTheOrderItIsApplied(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	entered := make(chan struct{})
	release := make(chan struct{})
	var holdOnce sync.Once
	analyser := func(text string) []string {
		if text == "first" {
			holdOnce.Do(func() {
				close(entered)
				<-release
			})
		}
		return strings.Fields(text)
	}
	provider, err := OtterProviderFactory(cache_dto.Options[string, Product]{
		MaximumEntries: 100,
		SearchSchema:   cache_dto.NewSearchSchemaWithAnalyser(analyser, cache_dto.TextField("Name")),
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, Product])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })

	write := func(name string) error {
		_, _, computeErr := cache.Compute(ctx, "p1", func(Product, bool) (Product, cache_dto.ComputeAction) {
			return Product{ID: "p1", Name: name}, cache_dto.ComputeActionSet
		})
		return computeErr
	}
	firstDone := make(chan error, 1)
	go func() { firstDone <- write("first") }()
	<-entered

	secondDone := make(chan error, 1)
	go func() { secondDone <- write("second") }()
	select {
	case secondErr := <-secondDone:
		require.NoError(t, secondErr)
		secondDone <- nil
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)

	value, found, err := cache.GetIfPresent(ctx, "p1")
	require.NoError(t, err)
	require.True(t, found)
	matching, err := cache.Search(ctx, value.Name, &cache_dto.SearchOptions{Limit: cache_dto.NoLimit})
	require.NoError(t, err)
	assert.Len(t, matching.Items, 1, "the index must describe the value the cache holds, %q", value.Name)
}

type lockProbeHandler struct {
	slog.Handler
	probe func()
}

func (h lockProbeHandler) Handle(ctx context.Context, record slog.Record) error {
	h.probe()
	return h.Handler.Handle(ctx, record)
}

func TestOtterAdapter_UnsupportedTTLWarningIsLoggedOutsideTheLock(t *testing.T) {
	t.Parallel()

	provider, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 16})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, string])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })

	logOutput := new(bytes.Buffer)
	handler := lockProbeHandler{Handler: slog.NewTextHandler(logOutput, nil), probe: func() {
		otherWrite := make(chan struct{})
		go func() {
			cache.client.Compute(ttlTestKey, func(oldValue string, _ bool) (string, otter.ComputeOp) {
				return oldValue, otter.CancelOp
			})
			close(otherWrite)
		}()
		select {
		case <-otherWrite:
		case <-time.After(5 * time.Second):
			assert.Fail(t, "the warning was logged while the key's lock was held")
		}
	}}
	ctx := logger_domain.WithLogger(t.Context(), logger_domain.New(slog.New(handler), "otter-ttl-test"))

	_, _, err = cache.ComputeWithTTL(ctx, ttlTestKey, func(string, bool) cache_dto.ComputeResult[string] {
		return cache_dto.ComputeResult[string]{Value: ttlTestValue, Action: cache_dto.ComputeActionSet, TTL: time.Minute}
	})
	require.NoError(t, err)

	assert.Contains(t, logOutput.String(), ttlWarningMessage)
}

func TestOtterAdapter_LongTTLNearTheEndOfRepresentableTimeDoesNotOverflow(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mockClock := clock.NewMockClock(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))
	cache := newPerEntryAdapter[string](t, mockClock)
	wal := &deferredWAL{}
	cache.wal = wal
	cache.walEnabled = true
	const century = 100 * 365 * 24 * time.Hour

	require.NoError(t, cache.SetWithTTL(ctx, ttlTestKey, ttlTestValue, century))

	entry, found, err := cache.ProbeEntry(ctx, ttlTestKey)
	require.NoError(t, err)
	require.True(t, found, "a long TTL must not wrap to a time in the past and expire the entry at once")
	assert.Greater(t, entry.ExpiresAtNano, mockClock.Now().UnixNano())
	require.Len(t, wal.queued, 1)
	assert.Greater(t, wal.queued[0].ExpiresAt, mockClock.Now().UnixNano(), "the logged expiry must not wrap either")
	assert.Less(t, wal.queued[0].ExpiresAt, int64(math.MaxInt64), "a capped TTL is still an expiry, not the never-expires marker")
}

func TestOtterAdapter_LoggedExpiryCapsATTLAgainstItsOwnClockReading(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC))
	cache := newPerEntryAdapter[string](t, mockClock)

	longestTTL := cache_dto.NoExpiry - 1

	expiresAt := cache.loggedExpiry(ttlTestKey, longestTTL, true)

	assert.Greater(t, expiresAt, mockClock.Now().UnixNano(), "the logged expiry must not wrap to a time in the past")
	assert.Less(t, expiresAt, int64(math.MaxInt64), "a capped TTL is still an expiry, not the never-expires marker")
}

func TestCapTTL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		ttl     time.Duration
		nowNano int64
		want    time.Duration
	}{
		{name: "a short TTL is kept", ttl: time.Minute, nowNano: 1_000, want: time.Minute},
		{name: "a TTL that would overflow is capped", ttl: time.Duration(math.MaxInt64), nowNano: 1_000, want: time.Duration(math.MaxInt64 - 1_001)},
		{name: "no current time keeps the TTL", ttl: time.Hour, nowNano: 0, want: time.Hour},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, capTTL(testCase.ttl, testCase.nowNano))
		})
	}
}

func TestOtterAdapter_InvalidateByTagsSurvivesARestart(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	open := func() *OtterAdapter[string, testVecDocument] {
		adapter, err := OtterProviderFactory(cache_dto.Options[string, testVecDocument]{
			MaximumEntries:   100,
			ProviderSpecific: newVecPersistenceConfig(directory),
		})
		require.NoError(t, err)
		cache, ok := adapter.(*OtterAdapter[string, testVecDocument])
		require.True(t, ok)
		return cache
	}
	ctx := context.Background()

	first := open()
	require.NoError(t, first.Set(ctx, "stale-1", testVecDocument{Title: "a"}, "stale"))
	require.NoError(t, first.Set(ctx, "stale-2", testVecDocument{Title: "b"}, "stale"))
	require.NoError(t, first.Set(ctx, "fresh", testVecDocument{Title: "c"}, "fresh"))
	removed, err := first.InvalidateByTags(ctx, "stale")
	require.NoError(t, err)
	assert.Equal(t, 2, removed)
	require.NoError(t, first.Close(ctx))

	second := open()
	t.Cleanup(func() { _ = second.Close(ctx) })
	_, found, err := second.GetIfPresent(ctx, "stale-1")
	require.NoError(t, err)
	assert.False(t, found, "a key removed by tag must stay removed after the log is replayed")
	_, found, err = second.GetIfPresent(ctx, "fresh")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestOtterAdapter_InvalidateByTagsStopsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	adapter, err := OtterProviderFactory(cache_dto.Options[string, testVecDocument]{
		MaximumEntries:   100,
		ProviderSpecific: newVecPersistenceConfig(t.TempDir()),
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = adapter.Close(context.Background()) })
	require.NoError(t, adapter.Set(context.Background(), "tagged", testVecDocument{Title: "a"}, "group"))

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test: caller gave up"))
	removed, err := adapter.InvalidateByTags(ctx, "group")

	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, removed)
}

func TestOtterAdapter_OnDeletionReportsTheRemovedEntry(t *testing.T) {
	t.Parallel()

	events := make(chan cache_dto.DeletionEvent[string, string], 1)
	adapter, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumEntries: 10,
		OnDeletion:     func(event cache_dto.DeletionEvent[string, string]) { events <- event },
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = adapter.Close(context.Background()) })

	require.NoError(t, adapter.Set(context.Background(), "gone", "value"))
	require.NoError(t, adapter.Invalidate(context.Background(), "gone"))

	select {
	case event := <-events:
		assert.Equal(t, "gone", event.Key)
		assert.Equal(t, "value", event.Value)
		assert.Equal(t, cache_dto.CauseInvalidation, event.Cause)
	case <-time.After(5 * time.Second):
		t.Fatal("the deletion handler was never called")
	}
}

func TestOtterAdapter_InvalidateByTagsKeepsTheTagsOfKeysItCouldNotRemove(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cache := newPerEntryAdapter[string](t, newTTLTestClock())
	for _, key := range []string{"a", "b", "c"} {
		require.NoError(t, cache.Set(ctx, key, "value", "group", "other"))
	}
	wal := &deferredWAL{enqueueErr: errors.New("disk gone")}
	cache.wal = wal
	cache.walEnabled = true

	removed, err := cache.InvalidateByTags(ctx, "group")

	require.Error(t, err)
	assert.Zero(t, removed)
	assert.Len(t, cache.tagIndex.Get("group"), 3, "keys that were not removed must keep their tags")
	assert.ElementsMatch(t, []string{"group", "other"}, cache.tagIndex.GetTags("a"))

	wal.enqueueErr = nil
	removed, err = cache.InvalidateByTags(ctx, "group")
	require.NoError(t, err)
	assert.Equal(t, 3, removed, "a later call still finds every key by its tag")
}

func TestAllocBudget_OtterSet(t *testing.T) {
	ctx := context.Background()

	t.Run("log=off", func(t *testing.T) {
		adapter, err := OtterProviderFactory(cache_dto.Options[string, string]{MaximumEntries: 100})
		require.NoError(t, err)
		t.Cleanup(func() { _ = adapter.Close(ctx) })
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 2.5}, func() {
			_ = adapter.Set(ctx, "key", "value")
		})
	})

	t.Run("log=on", func(t *testing.T) {
		walConfig := wal_domain.DefaultConfig(t.TempDir())
		walConfig.SnapshotThreshold = 1 << 30
		adapter, err := OtterProviderFactory(cache_dto.Options[string, string]{
			MaximumEntries: 100,
			ProviderSpecific: PersistenceConfig[string, string]{
				Enabled:    true,
				WALConfig:  walConfig,
				KeyCodec:   stringKeyCodec{},
				ValueCodec: plainStringValueCodec{},
			},
		})
		require.NoError(t, err)
		t.Cleanup(func() { _ = adapter.Close(ctx) })
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 9.5}, func() {
			if err := adapter.Set(ctx, "key", "value"); err != nil {
				t.Fatal(err)
			}
		})
	})
}

func TestOtterAdapter_SetDeferredReachesTheLogWithoutWaiting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	walConfig := wal_domain.DefaultConfig(t.TempDir())
	walConfig.SnapshotThreshold = 1 << 30
	adapter, err := OtterProviderFactory(cache_dto.Options[string, string]{
		MaximumEntries: 10,
		ProviderSpecific: PersistenceConfig[string, string]{
			Enabled: true, WALConfig: walConfig, KeyCodec: stringKeyCodec{}, ValueCodec: plainStringValueCodec{},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = adapter.Close(ctx) })
	cache, ok := adapter.(*OtterAdapter[string, string])
	require.True(t, ok)

	_, err = cache.SetDeferred(ctx, "deferred", "value")
	require.NoError(t, err)

	require.Eventually(t, func() bool { return cache.wal.EntryCount() == 1 }, 5*time.Second, time.Millisecond,
		"an entry queued under the lock must be written even when its wait is never called")
}

func TestOtterAdapter_BulkSetSignalsTheLogWhenAWriteFails(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cache := newPerEntryAdapter[string](t, newTTLTestClock())
	wal := &deferredWAL{enqueueErr: errors.New("queue full")}
	cache.wal = wal
	cache.walEnabled = true

	err := cache.BulkSet(ctx, map[string]string{"a": "1"})

	require.Error(t, err)
	assert.Equal(t, 1, wal.signals, "entries queued before the failure must still be handed to the writer")
}

type refusingWAL[V any] struct {
	wal_domain.WAL[string, V]
	err error
}

func (w *refusingWAL[V]) Append(context.Context, wal_domain.Entry[string, V]) error { return w.err }

func (*refusingWAL[V]) EntryCount() int { return 0 }

func (*refusingWAL[V]) Size() int64 { return 0 }

func (*refusingWAL[V]) Close() error { return nil }

func newIndexedProductCache(t *testing.T, analyser cache_dto.TextAnalyseFunc) *OtterAdapter[string, Product] {
	t.Helper()

	fields := []cache_dto.FieldSchema{cache_dto.TextField("Name"), cache_dto.SortableNumericField("Stock")}
	schema := cache_dto.NewSearchSchema(fields...)
	if analyser != nil {
		schema = cache_dto.NewSearchSchemaWithAnalyser(analyser, fields...)
	}
	provider, err := OtterProviderFactory(cache_dto.Options[string, Product]{MaximumEntries: 100, SearchSchema: schema})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, Product])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })
	return cache
}

func assertProductIndexed(t *testing.T, cache *OtterAdapter[string, Product], key string, product Product, indexed bool) {
	t.Helper()

	matching, err := cache.Search(context.Background(), product.Name, &cache_dto.SearchOptions{Limit: cache_dto.NoLimit})
	require.NoError(t, err)
	inStock := cache.sortedIndexes["Stock"].KeysBetween(product.Stock, product.Stock, true)
	if indexed {
		assert.Len(t, matching.Items, 1, "the text index must hold %q", product.Name)
		assert.Contains(t, inStock, key, "the sorted index must hold stock %d", product.Stock)
		return
	}
	assert.Empty(t, matching.Items, "the text index must not hold %q", product.Name)
	assert.NotContains(t, inStock, key, "the sorted index must not hold stock %d", product.Stock)
}

func TestOtterAdapter_PreparedSearchEntryIsOnlyCommittedForTheValueItWasBuiltFrom(t *testing.T) {
	t.Parallel()

	held := Product{ID: "p1", Name: "held", Stock: 1}
	prepared := Product{ID: "p1", Name: "prepared", Stock: 2}
	applied := Product{ID: "p1", Name: "applied", Stock: 3}

	testCases := []struct {
		remap       func(oldValue Product, found bool) (Product, otter.ComputeOp, time.Duration)
		name        string
		wantIndexed []Product
		wantMissing []Product
		wantPresent bool
	}{
		{
			name:        "a plain write commits the entry built before the lock",
			remap:       nil,
			wantIndexed: []Product{prepared},
			wantMissing: []Product{held},
			wantPresent: true,
		},
		{
			name: "a write of a different value indexes the value it writes",
			remap: func(Product, bool) (Product, otter.ComputeOp, time.Duration) {
				return applied, otter.WriteOp, 0
			},
			wantIndexed: []Product{applied},
			wantMissing: []Product{held, prepared},
			wantPresent: true,
		},
		{
			name: "a cancelled write leaves the index on the value the cache holds",
			remap: func(oldValue Product, _ bool) (Product, otter.ComputeOp, time.Duration) {
				return oldValue, otter.CancelOp, 0
			},
			wantIndexed: []Product{held},
			wantMissing: []Product{prepared},
			wantPresent: true,
		},
		{
			name: "a removal leaves nothing in the index",
			remap: func(oldValue Product, _ bool) (Product, otter.ComputeOp, time.Duration) {
				return oldValue, otter.InvalidateOp, 0
			},
			wantIndexed: nil,
			wantMissing: []Product{held, prepared},
			wantPresent: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cache := newIndexedProductCache(t, nil)
			require.NoError(t, cache.Set(ctx, "p1", held))

			request := cache.plainWrite(nil, "p1", prepared, 0, resolvedTTL{}, nil)
			require.NotNil(t, request.search, "a plain write must build its search entry before the lock")
			request.remap = testCase.remap
			_, present, err := cache.computeWithExpiry(ctx, "p1", request)
			require.NoError(t, err)

			assert.Equal(t, testCase.wantPresent, present)
			for _, product := range testCase.wantIndexed {
				assertProductIndexed(t, cache, "p1", product, true)
			}
			for _, product := range testCase.wantMissing {
				assertProductIndexed(t, cache, "p1", product, false)
			}
		})
	}
}

func TestOtterAdapter_PlainWriteTheLogRefusesChangesNoIndex(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := newIndexedProductCache(t, nil)
	held := Product{ID: "p1", Name: "held", Stock: 1}
	require.NoError(t, cache.Set(ctx, "p1", held, "old-tag"))
	refusal := errors.New("disk full")
	cache.wal = &refusingWAL[Product]{err: refusal}
	cache.walEnabled = true

	err := cache.Set(ctx, "p1", Product{ID: "p1", Name: "refused", Stock: 2}, "new-tag")

	require.ErrorIs(t, err, refusal)
	cache.walEnabled = false
	assertProductIndexed(t, cache, "p1", held, true)
	assertProductIndexed(t, cache, "p1", Product{ID: "p1", Name: "refused", Stock: 2}, false)
	assert.Equal(t, []string{"old-tag"}, cache.tagIndex.GetTags("p1"))
}

func TestOtterAdapter_SetAnalysesTextBeforeTheLockAndIndexesInApplyOrder(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	entered := make(chan struct{})
	release := make(chan struct{})
	var holdOnce sync.Once
	cache := newIndexedProductCache(t, func(text string) []string {
		if text == "first" {
			holdOnce.Do(func() {
				close(entered)
				<-release
			})
		}
		return strings.Fields(text)
	})

	firstDone := make(chan error, 1)
	go func() { firstDone <- cache.Set(ctx, "p1", Product{ID: "p1", Name: "first", Stock: 1}) }()
	<-entered

	secondDone := make(chan error, 1)
	go func() { secondDone <- cache.Set(ctx, "p1", Product{ID: "p1", Name: "second", Stock: 2}) }()
	select {
	case secondErr := <-secondDone:
		require.NoError(t, secondErr, "a write must not wait for another write's text analysis")
	case <-time.After(10 * time.Second):
		close(release)
		require.FailNow(t, "a write waited for another write's text analysis, which must run before the lock")
	}
	close(release)
	require.NoError(t, <-firstDone)

	value, found, err := cache.GetIfPresent(ctx, "p1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "first", value.Name, "the write applied last is the one whose analysis finished last")
	assertProductIndexed(t, cache, "p1", value, true)
	assertProductIndexed(t, cache, "p1", Product{ID: "p1", Name: "second", Stock: 2}, false)
}

func TestOtterAdapter_RewriteRemovesWhatTheNewValueNoLongerHas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider, err := OtterProviderFactory(cache_dto.Options[string, testVecDocument]{
		MaximumEntries: 100,
		SearchSchema:   cache_dto.NewSearchSchema(cache_dto.TextField("Title"), cache_dto.VectorField("Embedding", 2)),
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, testVecDocument])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })

	require.NoError(t, cache.Set(ctx, "d1", testVecDocument{Title: "alpha beta", Embedding: []float32{1, 0}}))
	require.NoError(t, cache.Set(ctx, "d1", testVecDocument{Title: "", Embedding: nil}))

	matching, err := cache.Search(ctx, "alpha", &cache_dto.SearchOptions{Limit: cache_dto.NoLimit})
	require.NoError(t, err)
	assert.Empty(t, matching.Items, "a value without text must not stay under its old terms")
	assert.Equal(t, 0, cache.invertedIndex.totalDocuments)
	assert.Equal(t, 0, cache.vectorIndexes["Embedding"].graph.Load().Len(), "a value without a vector must not keep its old one")
}
