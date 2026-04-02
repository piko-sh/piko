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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
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
		MaximumSize:      100000,
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
		MaximumSize: 10000,
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
		MaximumSize: 10000,
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
		MaximumSize: 10000,
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
		MaximumSize: 100000,
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
		MaximumSize: 10000,
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
		MaximumSize: 10000,
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

	case <-time.After(10 * time.Second):
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
		MaximumSize: 10000,
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
