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
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
)

func createConcurrencyTestCache(t *testing.T, walEnabled bool) *OtterAdapter[string, testArticle] {
	t.Helper()

	var providerSpecific any
	if walEnabled {
		directory := t.TempDir()
		walConfig := wal_domain.DefaultConfig(directory)
		walConfig.SyncMode = wal_domain.SyncModeBatched
		walConfig.SnapshotThreshold = 10000

		providerSpecific = PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		}
	}

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries:   100000,
		ProviderSpecific: providerSpecific,
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = adapter.Close(context.Background())
	})

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	return cache
}

func TestRace_ConcurrentSetGet(t *testing.T) {
	cache := createConcurrencyTestCache(t, true)
	ctx := context.Background()
	var wg sync.WaitGroup

	for w := range 10 {
		wg.Go(func() {
			for i := range 100 {
				key := fmt.Sprintf("key-%d-%d", w, i)
				_ = cache.Set(ctx, key, testArticle{Title: fmt.Sprintf("Title %d-%d", w, i)})
			}
		})
	}

	for r := range 10 {
		wg.Go(func() {
			for i := range 100 {
				key := fmt.Sprintf("key-%d-%d", r, i)
				_, _, _ = cache.GetIfPresent(ctx, key)
			}
		})
	}

	wg.Wait()
}

func TestRace_ConcurrentSetDelete(t *testing.T) {
	cache := createConcurrencyTestCache(t, true)
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := range 1000 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	for w := range 5 {
		wg.Go(func() {
			for i := range 200 {
				key := fmt.Sprintf("key-%d", (w*200+i)%1000)
				_ = cache.Set(ctx, key, testArticle{Title: fmt.Sprintf("Updated %d-%d", w, i)})
			}
		})
	}

	for d := range 5 {
		wg.Go(func() {
			for i := range 200 {
				key := fmt.Sprintf("key-%d", (d*200+i)%1000)
				_ = cache.Invalidate(ctx, key)
			}
		})
	}

	wg.Wait()
}

func TestRace_ConcurrentBulkSet(t *testing.T) {
	cache := createConcurrencyTestCache(t, true)
	ctx := context.Background()
	var wg sync.WaitGroup

	for w := range 10 {
		wg.Go(func() {
			items := make(map[string]testArticle, 50)
			for i := range 50 {
				items[fmt.Sprintf("bulk-%d-%d", w, i)] = testArticle{Title: fmt.Sprintf("Bulk %d-%d", w, i)}
			}
			for range 5 {
				_ = cache.BulkSet(ctx, items)
			}
		})
	}

	wg.Wait()
}

func TestRace_ConcurrentCheckpoint(t *testing.T) {
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeBatched
	walConfig.SnapshotThreshold = 50

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	var wg sync.WaitGroup
	stop := make(chan struct{})

	for w := range 5 {
		wg.Go(func() {
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					_ = cache.Set(ctx, fmt.Sprintf("writer%d-key-%d", w, i), testArticle{Title: fmt.Sprintf("W%d-%d", w, i)})
					i++
				}
			}
		})
	}

	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				cache.checkpointMu.Lock()
				cache.performCheckpointLocked()
				cache.checkpointMu.Unlock()
				time.Sleep(5 * time.Millisecond)
			}
		}
	})

	time.Sleep(100 * time.Millisecond)
	close(stop)
	wg.Wait()
}

func TestRace_ConcurrentRecovery(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 1000

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter1, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	cache1, ok := adapter1.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 100 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}
	_ = cache1.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	var wg sync.WaitGroup

	for w := range 10 {
		wg.Go(func() {
			for i := range 50 {
				_ = cache2.Set(ctx, fmt.Sprintf("new-key-%d-%d", w, i), testArticle{Title: "New"})
				_, _, _ = cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i%100))
			}
		})
	}

	wg.Wait()
}

func TestStress_HighConcurrency_100Writers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	cache := createConcurrencyTestCache(t, true)
	ctx := context.Background()
	var wg sync.WaitGroup
	var writeCount atomic.Int64
	writers := 100
	opsPerWriter := 100

	for w := range writers {
		wg.Go(func() {
			for i := range opsPerWriter {
				key := fmt.Sprintf("writer%d-key-%d", w, i)
				_ = cache.Set(ctx, key, testArticle{Title: fmt.Sprintf("Title %d-%d", w, i), Views: w*opsPerWriter + i})
				writeCount.Add(1)
			}
		})
	}

	wg.Wait()

	assert.Equal(t, int64(writers*opsPerWriter), writeCount.Load())
	t.Logf("Completed %d writes across %d goroutines", writeCount.Load(), writers)
}

func TestStress_HighConcurrency_MixedReadWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	cache := createConcurrencyTestCache(t, true)
	ctx := context.Background()

	for i := range 1000 {
		_ = cache.Set(ctx, fmt.Sprintf("initial-%d", i), testArticle{Title: fmt.Sprintf("Initial %d", i)})
	}

	var wg sync.WaitGroup
	var readCount, writeCount atomic.Int64
	stop := make(chan struct{})
	duration := 500 * time.Millisecond

	for w := range 20 {
		wg.Go(func() {
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					_ = cache.Set(ctx, fmt.Sprintf("writer%d-%d", w, i), testArticle{Title: fmt.Sprintf("W%d-%d", w, i)})
					writeCount.Add(1)
					i++
				}
			}
		})
	}

	for range 30 {
		wg.Go(func() {
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					_, _, _ = cache.GetIfPresent(ctx, fmt.Sprintf("initial-%d", i%1000))
					readCount.Add(1)
					i++
				}
			}
		})
	}

	time.Sleep(duration)
	close(stop)
	wg.Wait()

	t.Logf("Reads: %d, Writes: %d in %v", readCount.Load(), writeCount.Load(), duration)
}

func TestStress_RapidCheckpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeBatched
	walConfig.SnapshotThreshold = 10

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	var wg sync.WaitGroup
	var checkpointCount atomic.Int64

	for w := range 10 {
		wg.Go(func() {
			for i := range 100 {
				_ = cache.Set(ctx, fmt.Sprintf("key-%d-%d", w, i), testArticle{Title: fmt.Sprintf("T%d-%d", w, i)})
			}
		})
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:

				if cache.wal.EntryCount() < 10 {
					checkpointCount.Add(1)
				}
			}
		}
	}()

	wg.Wait()
	close(done)

	t.Logf("Observed ~%d potential checkpoints", checkpointCount.Load())

	successCount := 0
	for w := range 10 {
		for i := range 100 {
			if _, ok, _ := cache.GetIfPresent(ctx, fmt.Sprintf("key-%d-%d", w, i)); ok {
				successCount++
			}
		}
	}

	assert.Greater(t, successCount, 500, "Most entries should be accessible")
}

func TestStress_ContinuousWritesDuringCheckpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeBatched
	walConfig.SnapshotThreshold = 10000

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 100000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	var writeCount atomic.Int64
	var checkpointCount atomic.Int64

	for w := range 10 {
		wg.Go(func() {
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
					_ = cache.Set(ctx, fmt.Sprintf("w%d-k%d", w, i), testArticle{Title: fmt.Sprintf("W%d-%d", w, i)})
					writeCount.Add(1)
					i++
				}
			}
		})
	}

	wg.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				time.Sleep(50 * time.Millisecond)
				cache.checkpointMu.Lock()
				cache.performCheckpointLocked()
				cache.checkpointMu.Unlock()
				checkpointCount.Add(1)
			}
		}
	})

	time.Sleep(500 * time.Millisecond)
	close(stop)
	wg.Wait()

	t.Logf("Completed %d writes and %d checkpoints", writeCount.Load(), checkpointCount.Load())
	assert.Greater(t, writeCount.Load(), int64(1000), "Should complete many writes")
	assert.GreaterOrEqual(t, checkpointCount.Load(), int64(2), "Should complete multiple checkpoints")
}

func TestDeadlock_SetDuringCheckpoint(t *testing.T) {
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeBatched
	walConfig.SnapshotThreshold = 10000

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 100 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("T%d", i)})
	}

	done := make(chan struct{})

	go func() {

		_ = cache.Set(ctx, "test-key", testArticle{Title: "Test"})
		close(done)
	}()

	cache.checkpointMu.Lock()
	cache.performCheckpointLocked()
	cache.checkpointMu.Unlock()

	select {
	case <-done:

	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected: Set blocked for too long")
	}
}

func TestDeadlock_CheckpointDuringBulkSet(t *testing.T) {
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeBatched
	walConfig.SnapshotThreshold = 10000

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	ctx := context.Background()
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Go(func() {
		items := make(map[string]testArticle, 1000)
		for i := range 1000 {
			items[fmt.Sprintf("bulk-%d", i)] = testArticle{Title: fmt.Sprintf("Bulk %d", i)}
		}

		for range 10 {
			select {
			case <-done:
				return
			default:
				_ = cache.BulkSet(ctx, items)
			}
		}
	})

	wg.Go(func() {
		for range 10 {
			select {
			case <-done:
				return
			default:
				cache.checkpointMu.Lock()
				cache.performCheckpointLocked()
				cache.checkpointMu.Unlock()
			}
		}
	})

	completed := make(chan struct{})
	go func() {
		wg.Wait()
		close(completed)
	}()

	select {
	case <-completed:

	case <-time.After(30 * time.Second):
		close(done)
		t.Fatal("Deadlock detected: operations blocked for too long")
	}
}

func TestDeadlock_NestedOperations(t *testing.T) {
	cache := createConcurrencyTestCache(t, true)
	ctx := context.Background()

	for i := range 100 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("T%d", i)})
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	ops := []func(){
		func() {
			_ = cache.Set(ctx, "set-key", testArticle{Title: "Set"})
		},
		func() {
			_, _, _ = cache.GetIfPresent(ctx, "key-50")
		},
		func() {
			_ = cache.Invalidate(ctx, "key-25")
		},
		func() {
			items := map[string]testArticle{"bulk-1": {Title: "B1"}}
			_ = cache.BulkSet(ctx, items)
		},
		func() {
			_ = cache.SetWithTTL(ctx, "ttl-key", testArticle{Title: "TTL"}, time.Hour)
		},
	}

	for range 20 {
		for _, op := range ops {
			wg.Go(func() {
				for {
					select {
					case <-done:
						return
					default:
						op()
					}
				}
			})
		}
	}

	time.Sleep(200 * time.Millisecond)
	close(done)

	completed := make(chan struct{})
	go func() {
		wg.Wait()
		close(completed)
	}()

	select {
	case <-completed:

	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected: nested operations blocked for too long")
	}
}

func TestConcurrency_WALAppendOrder(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 10000

	opts := cache_dto.Options[string, testArticle]{
		MaximumEntries: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err)

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	var wg sync.WaitGroup

	for w := range 10 {
		wg.Go(func() {
			for i := range 100 {
				_ = cache.Set(ctx, "shared-key", testArticle{Title: fmt.Sprintf("Writer%d-Update%d", w, i)})
			}
		})
	}

	wg.Wait()
	_ = cache.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	value, ok, _ := cache2.GetIfPresent(ctx, "shared-key")
	assert.True(t, ok)

	assert.NotEmpty(t, value.Title)
}

func TestConcurrency_NoWAL_Baseline(t *testing.T) {
	cache := createConcurrencyTestCache(t, false)
	ctx := context.Background()
	var wg sync.WaitGroup
	var opCount atomic.Int64

	for w := range 50 {
		wg.Go(func() {
			for i := range 100 {
				_ = cache.Set(ctx, fmt.Sprintf("key-%d-%d", w, i), testArticle{Title: fmt.Sprintf("T%d-%d", w, i)})
				opCount.Add(1)
			}
		})
	}

	wg.Wait()

	assert.Equal(t, int64(5000), opCount.Load())
	t.Logf("NoWAL baseline: %d operations completed", opCount.Load())
}

func newLoggedPerEntryAdapter(t *testing.T, directory string, mockClock *clock.MockClock, snapshotThreshold int) *OtterAdapter[string, testArticle] {
	t.Helper()

	return newPerEntryAdapter(t, mockClock, func(builder *cache_domain.CacheBuilder[string, testArticle]) {
		walConfig := wal_domain.DefaultConfig(directory)
		walConfig.SyncMode = wal_domain.SyncModeNone
		walConfig.EnableCompression = false
		walConfig.SnapshotThreshold = snapshotThreshold
		builder.Options(PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		})
	})
}

func TestStress_MixedWritesAndTransactionsLeaveNoPendingExpiry(t *testing.T) {
	ctx := context.Background()
	adapter := newLoggedPerEntryAdapter(t, t.TempDir(), newTTLTestClock(), 50)

	var group sync.WaitGroup
	for writer := range 16 {
		group.Go(func() {
			for round := range 300 {
				key := fmt.Sprintf("k%d", round%17)
				switch (writer + round) % 6 {
				case 0:
					assert.NoError(t, adapter.SetWithTTL(ctx, key, testArticle{Title: "a"}, time.Minute))
				case 1:
					_, _, err := adapter.ComputeWithTTL(ctx, key, func(old testArticle, _ bool) cache_dto.ComputeResult[testArticle] {
						return cache_dto.ComputeResult[testArticle]{Value: old, Action: cache_dto.ComputeActionSet, TTL: time.Second}
					})
					assert.NoError(t, err)
				case 2:
					assert.NoError(t, adapter.SetExpiresAfter(ctx, key, time.Hour))
				case 3:
					assert.NoError(t, adapter.Set(ctx, key, testArticle{Title: "b"}, "tag"))
				case 4:
					transaction := cache_domain.BeginTransaction(ctx, adapter)
					assert.NoError(t, transaction.SetWithTTL(ctx, key, testArticle{Title: "tx"}, time.Minute))
					if round%2 == 0 {
						assert.NoError(t, transaction.Rollback(ctx))
					} else {
						assert.NoError(t, transaction.Commit(ctx))
					}
				case 5:
					if round%50 == 0 {
						assert.NoError(t, adapter.Checkpoint(ctx))
					}
					_, _, err := adapter.ComputeIfPresent(ctx, key, func(old testArticle) (testArticle, cache_dto.ComputeAction) {
						return old, cache_dto.ComputeActionDelete
					})
					assert.NoError(t, err)
				}
			}
		})
	}
	group.Wait()

	pending := 0
	adapter.expiryOverrides.pending.Range(func(any, any) bool { pending++; return true })
	assert.Zero(t, adapter.expiryOverrides.count.Load(), "every TTL handed to a write is used or discarded")
	assert.Zero(t, pending, "every TTL handed to a write is used or discarded")
}

func TestStress_ReplayMatchesLiveStateAfterConcurrentPlainAndTimedWrites(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	mockClock := newTTLTestClock()
	adapter := newLoggedPerEntryAdapter(t, directory, mockClock, 1<<30)

	keys := []string{"a", "b", "c", "d"}
	var group sync.WaitGroup
	for writer := range 8 {
		group.Go(func() {
			for round := range 400 {
				key := keys[(writer+round)%len(keys)]
				title := fmt.Sprintf("%d-%d", writer, round)
				switch round % 4 {
				case 0:
					assert.NoError(t, adapter.Set(ctx, key, testArticle{Title: title}))
				case 1:
					assert.NoError(t, adapter.SetWithTTL(ctx, key, testArticle{Title: title}, time.Duration(writer+1)*time.Minute))
				case 2:
					assert.NoError(t, adapter.BulkSet(ctx, map[string]testArticle{key: {Title: title}}))
				case 3:
					wait, err := adapter.SetDeferred(ctx, key, testArticle{Title: title})
					assert.NoError(t, err)
					assert.NoError(t, wait())
				}
			}
		})
	}
	group.Wait()

	live := make(map[string]cache_dto.Entry[string, testArticle], len(keys))
	for _, key := range keys {
		entry, found, err := adapter.ProbeEntry(ctx, key)
		require.NoError(t, err)
		require.True(t, found)
		live[key] = entry
	}
	require.NoError(t, adapter.wal.Sync(ctx))

	recovered := newLoggedPerEntryAdapter(t, crashCopy(t, directory), mockClock, 1<<30)
	for _, key := range keys {
		entry, found, err := recovered.ProbeEntry(ctx, key)
		require.NoError(t, err)
		require.True(t, found, "key %s must replay", key)
		assert.Equal(t, live[key].Value, entry.Value, "key %s must replay to the value the cache holds", key)
		assert.Equal(t, live[key].ExpiresAtNano, entry.ExpiresAtNano, "key %s must replay to the expiry the cache holds", key)
	}
}

func TestStress_ConcurrentComputesKeepTheIndexesInStepWithTheCache(t *testing.T) {
	ctx := context.Background()
	provider, err := OtterProviderFactory(cache_dto.Options[string, Product]{
		MaximumEntries: 1000,
		SearchSchema:   cache_dto.NewSearchSchema(cache_dto.SortableNumericField("Stock")),
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, Product])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })

	keys := []string{"p0", "p1", "p2"}
	var group sync.WaitGroup
	for writer := range 8 {
		group.Go(func() {
			for round := range 500 {
				key := keys[(writer+round)%len(keys)]
				stock := writer*1000 + round
				switch round % 3 {
				case 0:
					assert.NoError(t, cache.SetWithTTL(ctx, key, Product{ID: key, Stock: stock}, time.Hour, fmt.Sprintf("writer-%d", writer)))
				case 1:
					_, _, computeErr := cache.Compute(ctx, key, func(oldValue Product, _ bool) (Product, cache_dto.ComputeAction) {
						oldValue.ID = key
						oldValue.Stock = stock
						return oldValue, cache_dto.ComputeActionSet
					})
					assert.NoError(t, computeErr)
				case 2:
					assert.NoError(t, cache.Invalidate(ctx, key))
				}
			}
		})
	}
	group.Wait()

	stockIndex := cache.sortedIndexes["Stock"]
	presentKeys := 0
	for _, key := range keys {
		value, found, getErr := cache.GetIfPresent(ctx, key)
		require.NoError(t, getErr)
		if !found {
			assert.Empty(t, cache.tagIndex.GetTags(key), "a removed key must leave no tag behind: %s", key)
			continue
		}
		presentKeys++
		assert.Contains(t, stockIndex.KeysBetween(value.Stock, value.Stock, true), key,
			"the sorted index must hold the value the cache holds for %s", key)
	}
	assert.Equal(t, presentKeys, stockIndex.Size(), "the sorted index must hold exactly the keys the cache holds")
}

func TestStress_ConcurrentPreparedWritesAndRemovalsKeepEveryIndexInStepWithTheCache(t *testing.T) {
	ctx := context.Background()
	provider, err := OtterProviderFactory(cache_dto.Options[string, testVecDocument]{
		MaximumEntries: 1000,
		SearchSchema: cache_dto.NewSearchSchema(
			cache_dto.TextField("Title"),
			cache_dto.SortableNumericField("Rating"),
			cache_dto.VectorField("Embedding", 2),
		),
	})
	require.NoError(t, err)
	cache, ok := provider.(*OtterAdapter[string, testVecDocument])
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, cache.Close(context.Background())) })

	keys := []string{"d0", "d1", "d2", "d3"}
	document := func(writer, round int) testVecDocument {
		return testVecDocument{
			Title:     fmt.Sprintf("writer%d round%d", writer, round),
			Category:  "",
			Embedding: []float32{float32(writer + 1), float32(round + 1)},
			Rating:    float64(writer*1000 + round),
		}
	}
	var group sync.WaitGroup
	for writer := range 8 {
		group.Go(func() {
			for round := range 400 {
				key := keys[(writer+round)%len(keys)]
				switch round % 5 {
				case 0, 1:
					assert.NoError(t, cache.Set(ctx, key, document(writer, round), fmt.Sprintf("writer-%d", writer), "all"))
				case 2:
					_, _, computeErr := cache.Compute(ctx, key, func(testVecDocument, bool) (testVecDocument, cache_dto.ComputeAction) {
						return document(writer, round), cache_dto.ComputeActionSet
					})
					assert.NoError(t, computeErr)
				case 3:
					assert.NoError(t, cache.Invalidate(ctx, key))
				case 4:
					_, invalidateErr := cache.InvalidateByTags(ctx, fmt.Sprintf("writer-%d", (writer+1)%8))
					assert.NoError(t, invalidateErr)
				}
			}
		})
	}
	group.Wait()

	present := 0
	for _, key := range keys {
		value, found, getErr := cache.GetIfPresent(ctx, key)
		require.NoError(t, getErr)
		cache.invertedIndex.mu.RLock()
		terms, indexed := cache.invertedIndex.keyTerms[key]
		cache.invertedIndex.mu.RUnlock()
		inRating := cache.sortedIndexes["Rating"].KeysBetween(value.Rating, value.Rating, true)
		if !found {
			assert.Empty(t, cache.tagIndex.GetTags(key), "a removed key must leave no tag behind: %s", key)
			assert.False(t, indexed, "a removed key must leave no text terms behind: %s", key)
			continue
		}
		present++
		assert.LessOrEqual(t, len(cache.tagIndex.GetTags(key)), 2, "a rewrite must replace tags, not add to them: %s", key)
		require.True(t, indexed, "a present key must be in the text index: %s", key)
		assert.ElementsMatch(t, strings.Fields(value.Title), slices.Collect(maps.Keys(terms)),
			"the text index must hold the terms of the value the cache holds for %s", key)
		assert.Contains(t, inRating, key, "the sorted index must hold the value the cache holds for %s", key)
	}
	for _, tag := range []string{"all", "writer-0", "writer-7"} {
		for key := range cache.tagIndex.Get(tag) {
			_, found, getErr := cache.GetIfPresent(ctx, key)
			require.NoError(t, getErr)
			assert.True(t, found, "tag %s must not point at the removed key %s", tag, key)
		}
	}
	assert.Equal(t, present, cache.sortedIndexes["Rating"].Size(), "the sorted index must hold exactly the keys the cache holds")
	assert.Equal(t, present, cache.invertedIndex.totalDocuments, "the text index must hold exactly the keys the cache holds")
	assert.Equal(t, present, cache.vectorIndexes["Embedding"].graph.Load().Len(), "the vector index must hold exactly the keys the cache holds")
}
