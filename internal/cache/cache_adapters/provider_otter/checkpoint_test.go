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
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

func createCheckpointTestCache(t *testing.T, directory string, threshold int) *OtterAdapter[string, testArticle] {
	t.Helper()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.EnableCompression = false
	walConfig.SnapshotThreshold = threshold

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

	t.Cleanup(func() {
		_ = adapter.Close(context.Background())
	})

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	return cache
}

func TestCheckpoint_TriggersAtThreshold(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range threshold {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	time.Sleep(50 * time.Millisecond)

	assert.Less(t, cache.wal.EntryCount(), threshold, "WAL should have been truncated after checkpoint")
}

func TestCheckpoint_DoesNotTriggerBelowThreshold(t *testing.T) {
	directory := t.TempDir()
	threshold := 100
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range 50 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	assert.GreaterOrEqual(t, cache.wal.EntryCount(), 50, "WAL should retain entries below threshold")
}

func TestCheckpoint_SnapshotCreated(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range threshold + 5 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	time.Sleep(50 * time.Millisecond)

	snapshotPath := filepath.Join(directory, "snapshot.piko")
	_, err := os.Stat(snapshotPath)
	assert.NoError(t, err, "Snapshot file should exist after checkpoint")
}

func TestCheckpoint_WALTruncatedAfterSnapshot(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range threshold + 5 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	time.Sleep(50 * time.Millisecond)

	entryCount := cache.wal.EntryCount()
	assert.Less(t, entryCount, threshold, "WAL entry count should be less than threshold after checkpoint")
}

func TestCheckpoint_StatePreserved(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	ctx := context.Background()

	cache1 := createCheckpointTestCache(t, directory, threshold)

	for i := range 25 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i), Views: i * 10})
	}

	cache1.checkpointMu.Lock()
	cache1.performCheckpointLocked()
	cache1.checkpointMu.Unlock()

	_ = cache1.Close(ctx)

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = threshold

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 25 {
		value, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i))
		assert.True(t, ok, "Entry key-%d should be present after recovery", i)
		assert.Equal(t, fmt.Sprintf("Title %d", i), value.Title)
		assert.Equal(t, i*10, value.Views)
	}
}

func TestCheckpoint_CrashSafety_WritesDuringCheckpoint(t *testing.T) {
	directory := t.TempDir()
	threshold := 50
	ctx := context.Background()

	cache := createCheckpointTestCache(t, directory, threshold)

	for i := range 30 {
		_ = cache.Set(ctx, fmt.Sprintf("initial-%d", i), testArticle{Title: fmt.Sprintf("Initial %d", i)})
	}

	var wg sync.WaitGroup
	writesDone := make(chan struct{})

	for w := range 5 {
		wg.Go(func() {
			for i := range 50 {
				select {
				case <-writesDone:
					return
				default:
					_ = cache.Set(ctx, fmt.Sprintf("writer%d-key-%d", w, i),
						testArticle{Title: fmt.Sprintf("Writer %d Title %d", w, i)})
					time.Sleep(time.Millisecond)
				}
			}
		})
	}

	for range 3 {
		time.Sleep(20 * time.Millisecond)
		cache.checkpointMu.Lock()
		cache.performCheckpointLocked()
		cache.checkpointMu.Unlock()
	}

	close(writesDone)
	wg.Wait()

	_ = cache.Close(ctx)

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = threshold

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 10000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 30 {
		value, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("initial-%d", i))
		assert.True(t, ok, "Initial entry %d should be present after recovery", i)
		assert.Equal(t, fmt.Sprintf("Initial %d", i), value.Title)
	}
}

func TestCheckpoint_CrashSafety_NoDataLoss(t *testing.T) {
	directory := t.TempDir()
	threshold := 20
	ctx := context.Background()

	func() {
		cache := createCheckpointTestCache(t, directory, threshold)

		for i := range 50 {
			_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i), Views: i})
		}

		cache.checkpointMu.Lock()
		cache.performCheckpointLocked()
		cache.checkpointMu.Unlock()

		for i := 50; i < 75; i++ {
			_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i), Views: i})
		}

		_ = cache.Close(ctx)
	}()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = threshold

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
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 75 {
		value, ok, _ := cache.GetIfPresent(ctx, fmt.Sprintf("key-%d", i))
		assert.True(t, ok, "Entry key-%d should be present", i)
		assert.Equal(t, fmt.Sprintf("Title %d", i), value.Title)
		assert.Equal(t, i, value.Views)
	}
}

func TestCheckpoint_AtomicSnapshot_TempFileCleanup(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range threshold + 5 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	time.Sleep(50 * time.Millisecond)

	entries, err := os.ReadDir(directory)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp", "No temporary files should remain after checkpoint")
	}
}

func TestCheckpoint_ConcurrentWrites(t *testing.T) {
	directory := t.TempDir()
	threshold := 100
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	var wg sync.WaitGroup
	var writeCount atomic.Int64

	for w := range 10 {
		wg.Go(func() {
			for i := range 50 {
				_ = cache.Set(ctx, fmt.Sprintf("writer%d-key-%d", w, i),
					testArticle{Title: fmt.Sprintf("Title %d-%d", w, i)})
				writeCount.Add(1)
			}
		})
	}

	wg.Wait()

	assert.Equal(t, int64(500), writeCount.Load())

	for w := range 10 {
		for i := range 50 {
			value, ok, _ := cache.GetIfPresent(ctx, fmt.Sprintf("writer%d-key-%d", w, i))
			assert.True(t, ok, "Entry writer%d-key-%d should exist", w, i)
			assert.Equal(t, fmt.Sprintf("Title %d-%d", w, i), value.Title)
		}
	}
}

func TestCheckpoint_ConcurrentReads(t *testing.T) {
	directory := t.TempDir()
	threshold := 50
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range 100 {
		_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i), Views: i})
	}

	var wg sync.WaitGroup
	var readCount atomic.Int64
	stopReading := make(chan struct{})

	for r := range 10 {
		wg.Go(func() {
			for {
				select {
				case <-stopReading:
					return
				default:
					key := fmt.Sprintf("key-%d", r*10)
					_, _, _ = cache.GetIfPresent(ctx, key)
					readCount.Add(1)
				}
			}
		})
	}

	time.Sleep(10 * time.Millisecond)
	cache.checkpointMu.Lock()
	cache.performCheckpointLocked()
	cache.checkpointMu.Unlock()
	time.Sleep(10 * time.Millisecond)

	close(stopReading)
	wg.Wait()

	for i := range 100 {
		value, ok, _ := cache.GetIfPresent(ctx, fmt.Sprintf("key-%d", i))
		assert.True(t, ok)
		assert.Equal(t, fmt.Sprintf("Title %d", i), value.Title)
	}

	t.Logf("Completed %d reads during checkpoint", readCount.Load())
}

func TestCheckpoint_ConcurrentMixedOps(t *testing.T) {
	directory := t.TempDir()
	threshold := 100
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range 100 {
		_ = cache.Set(ctx, fmt.Sprintf("initial-%d", i), testArticle{Title: fmt.Sprintf("Initial %d", i)})
	}

	var wg sync.WaitGroup
	stopOps := make(chan struct{})

	for w := range 3 {
		wg.Go(func() {
			i := 0
			for {
				select {
				case <-stopOps:
					return
				default:
					_ = cache.Set(ctx, fmt.Sprintf("writer%d-%d", w, i), testArticle{Title: fmt.Sprintf("W%d-%d", w, i)})
					i++
				}
			}
		})
	}

	for r := range 3 {
		wg.Go(func() {
			for {
				select {
				case <-stopOps:
					return
				default:
					_, _, _ = cache.GetIfPresent(ctx, fmt.Sprintf("initial-%d", r*10))
				}
			}
		})
	}

	wg.Go(func() {
		i := 50
		for {
			select {
			case <-stopOps:
				return
			default:
				_ = cache.Invalidate(ctx, fmt.Sprintf("initial-%d", i%100))
				i++
			}
		}
	})

	for range 5 {
		time.Sleep(10 * time.Millisecond)
		cache.checkpointMu.Lock()
		cache.performCheckpointLocked()
		cache.checkpointMu.Unlock()
	}

	close(stopOps)
	wg.Wait()

	_ = cache.Set(ctx, "final-key", testArticle{Title: "Final"})
	value, ok, _ := cache.GetIfPresent(ctx, "final-key")
	assert.True(t, ok)
	assert.Equal(t, "Final", value.Title)
}

func TestCheckpoint_MultipleCheckpointsSequential(t *testing.T) {
	directory := t.TempDir()
	threshold := 20
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for cycle := range 5 {

		for i := range 25 {
			key := fmt.Sprintf("cycle%d-key-%d", cycle, i)
			_ = cache.Set(ctx, key, testArticle{Title: fmt.Sprintf("Cycle %d Title %d", cycle, i)})
		}

		cache.checkpointMu.Lock()
		cache.performCheckpointLocked()
		cache.checkpointMu.Unlock()
	}

	for cycle := range 5 {
		for i := range 25 {
			key := fmt.Sprintf("cycle%d-key-%d", cycle, i)
			value, ok, _ := cache.GetIfPresent(ctx, key)
			assert.True(t, ok, "Entry %s should exist", key)
			assert.Equal(t, fmt.Sprintf("Cycle %d Title %d", cycle, i), value.Title)
		}
	}
}

func TestCheckpoint_EmptyCache(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	cache.checkpointMu.Lock()
	cache.performCheckpointLocked()
	cache.checkpointMu.Unlock()

	_ = cache.Set(ctx, "key", testArticle{Title: "Test"})
	value, ok, _ := cache.GetIfPresent(ctx, "key")
	assert.True(t, ok)
	assert.Equal(t, "Test", value.Title)
}

func TestCheckpoint_SingleEntry(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	_ = cache.Set(ctx, "only-key", testArticle{Title: "Only Entry", Views: 42})

	cache.checkpointMu.Lock()
	cache.performCheckpointLocked()
	cache.checkpointMu.Unlock()

	_ = cache.Close(ctx)

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = threshold

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
	defer func() { _ = adapter.Close(ctx) }()

	recovered, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)
	value, ok, _ := recovered.GetIfPresent(ctx, "only-key")
	assert.True(t, ok)
	assert.Equal(t, "Only Entry", value.Title)
	assert.Equal(t, 42, value.Views)
}

func TestCheckpoint_WithTTLEntries(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	for i := range 5 {
		_ = cache.SetWithTTL(ctx, fmt.Sprintf("ttl-key-%d", i),
			testArticle{Title: fmt.Sprintf("TTL %d", i)},
			time.Hour)
	}

	for i := range 5 {
		_ = cache.Set(ctx, fmt.Sprintf("no-ttl-key-%d", i),
			testArticle{Title: fmt.Sprintf("No TTL %d", i)})
	}

	cache.checkpointMu.Lock()
	cache.performCheckpointLocked()
	cache.checkpointMu.Unlock()

	for i := range 5 {
		value, ok, _ := cache.GetIfPresent(ctx, fmt.Sprintf("ttl-key-%d", i))
		assert.True(t, ok)
		assert.Equal(t, fmt.Sprintf("TTL %d", i), value.Title)

		value, ok, _ = cache.GetIfPresent(ctx, fmt.Sprintf("no-ttl-key-%d", i))
		assert.True(t, ok)
		assert.Equal(t, fmt.Sprintf("No TTL %d", i), value.Title)
	}
}

func TestCheckpoint_WithTags(t *testing.T) {
	directory := t.TempDir()
	threshold := 10
	cache := createCheckpointTestCache(t, directory, threshold)
	ctx := context.Background()

	_ = cache.Set(ctx, "tagged-1", testArticle{Title: "Tagged 1"}, "category:blog", "status:published")
	_ = cache.Set(ctx, "tagged-2", testArticle{Title: "Tagged 2"}, "category:news", "status:published")
	_ = cache.Set(ctx, "tagged-3", testArticle{Title: "Tagged 3"}, "category:blog", "status:draft")

	cache.checkpointMu.Lock()
	cache.performCheckpointLocked()
	cache.checkpointMu.Unlock()

	_ = cache.Close(ctx)

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = threshold

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
	defer func() { _ = adapter.Close(ctx) }()

	recovered, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	value, ok, _ := recovered.GetIfPresent(ctx, "tagged-1")
	assert.True(t, ok)
	assert.Equal(t, "Tagged 1", value.Title)

	value, ok, _ = recovered.GetIfPresent(ctx, "tagged-2")
	assert.True(t, ok)
	assert.Equal(t, "Tagged 2", value.Title)

	value, ok, _ = recovered.GetIfPresent(ctx, "tagged-3")
	assert.True(t, ok)
	assert.Equal(t, "Tagged 3", value.Title)

	tags := recovered.tagIndex.GetTags("tagged-1")
	assert.Contains(t, tags, "category:blog")
	assert.Contains(t, tags, "status:published")
}
