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

//go:build bench

package provider_otter

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/wal/wal_domain"
)

type benchArticle struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Views   int    `json:"views"`
}

type benchArticleCodec struct{}

func (benchArticleCodec) EncodeValue(value benchArticle) ([]byte, error) {
	return json.Marshal(value)
}

func (benchArticleCodec) DecodeValue(data []byte) (benchArticle, error) {
	var v benchArticle
	err := json.Unmarshal(data, &v)
	return v, err
}

type fastStringKeyCodec struct{}

func (fastStringKeyCodec) EncodeKey(key string) ([]byte, error) {
	return []byte(key), nil
}

func (fastStringKeyCodec) DecodeKey(data []byte) (string, error) {
	return string(data), nil
}

func (fastStringKeyCodec) KeySize(key string) int {
	return len(key)
}

func (fastStringKeyCodec) EncodeKeyTo(key string, buffer []byte) (int, error) {
	copy(buffer, key)
	return len(key), nil
}

func (fastStringKeyCodec) DecodeKeyFrom(data []byte) (string, error) {
	return mem.String(data), nil
}

type fastBenchArticleCodec struct{}

func (c fastBenchArticleCodec) EncodeValue(value benchArticle) ([]byte, error) {
	buffer := make([]byte, c.ValueSize(value))
	_, err := c.EncodeValueTo(value, buffer)
	return buffer, err
}

func (c fastBenchArticleCodec) DecodeValue(data []byte) (benchArticle, error) {
	return c.DecodeValueFrom(data)
}

func (fastBenchArticleCodec) ValueSize(value benchArticle) int {

	return 4 + len(value.ID) + 4 + len(value.Title) + 4 + len(value.Content) + 4
}

func (fastBenchArticleCodec) EncodeValueTo(value benchArticle, buffer []byte) (int, error) {
	offset := 0

	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.ID)))
	offset += 4
	copy(buffer[offset:], value.ID)
	offset += len(value.ID)

	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.Title)))
	offset += 4
	copy(buffer[offset:], value.Title)
	offset += len(value.Title)

	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.Content)))
	offset += 4
	copy(buffer[offset:], value.Content)
	offset += len(value.Content)

	binary.BigEndian.PutUint32(buffer[offset:], uint32(value.Views))
	offset += 4

	return offset, nil
}

func (fastBenchArticleCodec) DecodeValueFrom(data []byte) (benchArticle, error) {
	var v benchArticle
	offset := 0

	if len(data) < 16 {
		return v, fmt.Errorf("data too short: %d", len(data))
	}

	idLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	if offset+idLen > len(data) {
		return v, errors.New("truncated ID")
	}
	v.ID = mem.String(data[offset : offset+idLen])
	offset += idLen

	if offset+4 > len(data) {
		return v, errors.New("truncated at title length")
	}
	titleLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	if offset+titleLen > len(data) {
		return v, errors.New("truncated Title")
	}
	v.Title = mem.String(data[offset : offset+titleLen])
	offset += titleLen

	if offset+4 > len(data) {
		return v, errors.New("truncated at content length")
	}
	contentLen := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4
	if offset+contentLen > len(data) {
		return v, errors.New("truncated Content")
	}
	v.Content = mem.String(data[offset : offset+contentLen])
	offset += contentLen

	if offset+4 > len(data) {
		return v, errors.New("truncated at Views")
	}
	v.Views = int(binary.BigEndian.Uint32(data[offset:]))

	return v, nil
}

func createBenchCache(b *testing.B, walEnabled bool, syncMode wal_domain.SyncMode) *OtterAdapter[string, benchArticle] {
	return createBenchCacheWithCodec(b, walEnabled, syncMode, false)
}

func createBenchCacheWithCodec(b *testing.B, walEnabled bool, syncMode wal_domain.SyncMode, useFastCodec bool) *OtterAdapter[string, benchArticle] {
	b.Helper()

	var providerSpecific any
	if walEnabled {
		directory := b.TempDir()
		walConfig := wal_domain.DefaultConfig(directory)
		walConfig.SyncMode = syncMode
		walConfig.EnableCompression = false
		walConfig.SnapshotThreshold = 100_000

		var keyCodec wal_domain.KeyCodec[string]
		var valueCodec wal_domain.ValueCodec[benchArticle]

		if useFastCodec {
			keyCodec = fastStringKeyCodec{}
			valueCodec = fastBenchArticleCodec{}
		} else {
			keyCodec = stringKeyCodec{}
			valueCodec = benchArticleCodec{}
		}

		providerSpecific = PersistenceConfig[string, benchArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   keyCodec,
			ValueCodec: valueCodec,
		}
	}

	opts := cache_dto.Options[string, benchArticle]{
		MaximumSize:      100_000,
		ProviderSpecific: providerSpecific,
	}

	adapter, err := OtterProviderFactory(opts)
	if err != nil {
		b.Fatalf("Failed to create cache: %v", err)
	}

	b.Cleanup(func() {
		adapter.Close(context.Background())
	})

	cache, ok := adapter.(*OtterAdapter[string, benchArticle])
	if !ok {
		b.Fatal("type assertion failed")
	}
	return cache
}

func createPrefilledCache(b *testing.B, walEnabled bool, syncMode wal_domain.SyncMode, n int) *OtterAdapter[string, benchArticle] {
	b.Helper()

	cache := createBenchCache(b, walEnabled, syncMode)

	for i := range n {
		key := fmt.Sprintf("prefill-key-%d", i)
		value := benchArticle{
			ID:      key,
			Title:   fmt.Sprintf("Title %d", i),
			Content: "Pre-filled content for benchmarking",
			Views:   i,
		}
		cache.Set(context.Background(), key, value)
	}

	return cache
}

func generateBenchValue(i int) benchArticle {
	return benchArticle{
		ID:      fmt.Sprintf("bench-id-%d", i),
		Title:   fmt.Sprintf("Benchmark Title %d", i),
		Content: "This is benchmark content that simulates a realistic payload size",
		Views:   i * 10,
	}
}

func BenchmarkSet_NoWAL(b *testing.B) {
	cache := createBenchCache(b, false, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(context.Background(), key, generateBenchValue(i))
	}
}

func BenchmarkGet_NoWAL(b *testing.B) {
	cache := createPrefilledCache(b, false, 0, 10000)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("prefill-key-%d", i%10000)
		cache.GetIfPresent(context.Background(), key)
	}
}

func BenchmarkSetWithTTL_NoWAL(b *testing.B) {
	cache := createBenchCache(b, false, 0)
	ctx := context.Background()
	ttl := 5 * time.Minute

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		_ = cache.SetWithTTL(ctx, key, generateBenchValue(i), ttl)
	}
}

func BenchmarkInvalidate_NoWAL(b *testing.B) {
	cache := createPrefilledCache(b, false, 0, b.N)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("prefill-key-%d", i)
		cache.Invalidate(context.Background(), key)
	}
}

func BenchmarkInvalidateAll_NoWAL(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		cache := createPrefilledCache(b, false, 0, 1000)
		cache.InvalidateAll(context.Background())
	}
}

func BenchmarkBulkSet_NoWAL(b *testing.B) {
	cache := createBenchCache(b, false, 0)
	ctx := context.Background()

	items := make(map[string]benchArticle, 100)
	for i := range 100 {
		key := fmt.Sprintf("bulk-key-%d", i)
		items[key] = generateBenchValue(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = cache.BulkSet(ctx, items)
	}
}

func BenchmarkSet_WAL_SyncEveryWrite(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeEveryWrite)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(context.Background(), key, generateBenchValue(i))
	}
}

func BenchmarkGet_WAL_SyncEveryWrite(b *testing.B) {
	cache := createPrefilledCache(b, true, wal_domain.SyncModeEveryWrite, 10000)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("prefill-key-%d", i%10000)
		cache.GetIfPresent(context.Background(), key)
	}
}

func BenchmarkSetWithTTL_WAL_SyncEveryWrite(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeEveryWrite)
	ctx := context.Background()
	ttl := 5 * time.Minute

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		_ = cache.SetWithTTL(ctx, key, generateBenchValue(i), ttl)
	}
}

func BenchmarkInvalidate_WAL_SyncEveryWrite(b *testing.B) {
	cache := createPrefilledCache(b, true, wal_domain.SyncModeEveryWrite, b.N)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("prefill-key-%d", i)
		cache.Invalidate(context.Background(), key)
	}
}

func BenchmarkBulkSet_WAL_SyncEveryWrite(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeEveryWrite)
	ctx := context.Background()

	items := make(map[string]benchArticle, 100)
	for i := range 100 {
		key := fmt.Sprintf("bulk-key-%d", i)
		items[key] = generateBenchValue(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = cache.BulkSet(ctx, items)
	}
}

func BenchmarkSet_WAL_SyncBatched(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeBatched)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(context.Background(), key, generateBenchValue(i))
	}
}

func BenchmarkGet_WAL_SyncBatched(b *testing.B) {
	cache := createPrefilledCache(b, true, wal_domain.SyncModeBatched, 10000)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("prefill-key-%d", i%10000)
		cache.GetIfPresent(context.Background(), key)
	}
}

func BenchmarkSetWithTTL_WAL_SyncBatched(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeBatched)
	ctx := context.Background()
	ttl := 5 * time.Minute

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		_ = cache.SetWithTTL(ctx, key, generateBenchValue(i), ttl)
	}
}

func BenchmarkInvalidate_WAL_SyncBatched(b *testing.B) {
	cache := createPrefilledCache(b, true, wal_domain.SyncModeBatched, b.N)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("prefill-key-%d", i)
		cache.Invalidate(context.Background(), key)
	}
}

func BenchmarkBulkSet_WAL_SyncBatched(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeBatched)
	ctx := context.Background()

	items := make(map[string]benchArticle, 100)
	for i := range 100 {
		key := fmt.Sprintf("bulk-key-%d", i)
		items[key] = generateBenchValue(i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = cache.BulkSet(ctx, items)
	}
}

func BenchmarkParallelSet_NoWAL(b *testing.B) {
	cache := createBenchCache(b, false, 0)
	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			key := fmt.Sprintf("key-%d", i)
			cache.Set(context.Background(), key, generateBenchValue(int(i)))
		}
	})
}

func BenchmarkParallelSet_WAL_SyncEveryWrite(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeEveryWrite)
	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			key := fmt.Sprintf("key-%d", i)
			cache.Set(context.Background(), key, generateBenchValue(int(i)))
		}
	})
}

func BenchmarkParallelSet_WAL_SyncBatched(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeBatched)
	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			key := fmt.Sprintf("key-%d", i)
			cache.Set(context.Background(), key, generateBenchValue(int(i)))
		}
	})
}

func BenchmarkParallelMixed_NoWAL(b *testing.B) {
	cache := createPrefilledCache(b, false, 0, 10000)
	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			if i%3 == 0 {

				key := fmt.Sprintf("new-key-%d", i)
				cache.Set(context.Background(), key, generateBenchValue(int(i)))
			} else {

				key := fmt.Sprintf("prefill-key-%d", i%10000)
				cache.GetIfPresent(context.Background(), key)
			}
		}
	})
}

func BenchmarkParallelMixed_WAL(b *testing.B) {
	cache := createPrefilledCache(b, true, wal_domain.SyncModeBatched, 10000)
	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			if i%3 == 0 {

				key := fmt.Sprintf("new-key-%d", i)
				cache.Set(context.Background(), key, generateBenchValue(int(i)))
			} else {

				key := fmt.Sprintf("prefill-key-%d", i%10000)
				cache.GetIfPresent(context.Background(), key)
			}
		}
	})
}

func benchmarkCheckpoint(b *testing.B, stateSize int) {
	b.Helper()

	for range b.N {
		b.StopTimer()

		directory := b.TempDir()
		walConfig := wal_domain.DefaultConfig(directory)
		walConfig.SyncMode = wal_domain.SyncModeBatched
		walConfig.SnapshotThreshold = stateSize * 2

		opts := cache_dto.Options[string, benchArticle]{
			MaximumSize: int(stateSize * 2),
			ProviderSpecific: PersistenceConfig[string, benchArticle]{
				Enabled:    true,
				WALConfig:  walConfig,
				KeyCodec:   stringKeyCodec{},
				ValueCodec: benchArticleCodec{},
			},
		}

		adapter, err := OtterProviderFactory(opts)
		if err != nil {
			b.Fatalf("Failed to create cache: %v", err)
		}

		cache, ok := adapter.(*OtterAdapter[string, benchArticle])
		if !ok {
			b.Fatal("type assertion failed")
		}

		for i := range stateSize {
			key := fmt.Sprintf("key-%d", i)
			cache.Set(context.Background(), key, generateBenchValue(i))
		}

		b.StartTimer()

		cache.checkpointMu.Lock()
		cache.performCheckpointLocked()
		cache.checkpointMu.Unlock()

		b.StopTimer()
		cache.Close(context.Background())
	}
}

func BenchmarkCheckpoint_SmallState_1K(b *testing.B) {
	benchmarkCheckpoint(b, 1_000)
}

func BenchmarkCheckpoint_MediumState_10K(b *testing.B) {
	benchmarkCheckpoint(b, 10_000)
}

func BenchmarkCheckpoint_LargeState_100K(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping large checkpoint benchmark in short mode")
	}
	benchmarkCheckpoint(b, 100_000)
}

func benchmarkRecovery(b *testing.B, entryCount int, withSnapshot bool) {
	b.Helper()

	for range b.N {
		b.StopTimer()

		directory := b.TempDir()
		walConfig := wal_domain.DefaultConfig(directory)
		walConfig.SyncMode = wal_domain.SyncModeBatched
		walConfig.SnapshotThreshold = entryCount * 2

		opts := cache_dto.Options[string, benchArticle]{
			MaximumSize: int(entryCount * 2),
			ProviderSpecific: PersistenceConfig[string, benchArticle]{
				Enabled:    true,
				WALConfig:  walConfig,
				KeyCodec:   stringKeyCodec{},
				ValueCodec: benchArticleCodec{},
			},
		}

		adapter1, err := OtterProviderFactory(opts)
		if err != nil {
			b.Fatalf("Failed to create cache: %v", err)
		}

		cache1, ok := adapter1.(*OtterAdapter[string, benchArticle])
		if !ok {
			b.Fatal("type assertion failed")
		}
		for i := range entryCount {
			key := fmt.Sprintf("key-%d", i)
			cache1.Set(context.Background(), key, generateBenchValue(i))
		}

		if withSnapshot {
			cache1.checkpointMu.Lock()
			cache1.performCheckpointLocked()
			cache1.checkpointMu.Unlock()
		}

		cache1.Close(context.Background())

		b.StartTimer()

		adapter2, err := OtterProviderFactory(opts)
		if err != nil {
			b.Fatalf("Failed to create cache for recovery: %v", err)
		}

		b.StopTimer()
		adapter2.Close(context.Background())
	}
}

func BenchmarkRecovery_SmallWAL_1K(b *testing.B) {
	benchmarkRecovery(b, 1_000, false)
}

func BenchmarkRecovery_MediumWAL_10K(b *testing.B) {
	benchmarkRecovery(b, 10_000, false)
}

func BenchmarkRecovery_LargeWAL_100K(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping large recovery benchmark in short mode")
	}
	benchmarkRecovery(b, 100_000, false)
}

func BenchmarkRecovery_WithSnapshot_10K(b *testing.B) {
	benchmarkRecovery(b, 10_000, true)
}

func BenchmarkSetAlloc_NoWAL(b *testing.B) {
	cache := createBenchCache(b, false, 0)
	value := generateBenchValue(0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Set(context.Background(), key, value)
	}
}

func BenchmarkSetAlloc_WAL(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeBatched)
	value := generateBenchValue(0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Set(context.Background(), key, value)
	}
}

func BenchmarkStress_ManyGoroutines_NoWAL(b *testing.B) {
	cache := createBenchCache(b, false, 0)
	goroutines := 100
	var wg sync.WaitGroup

	b.ReportAllocs()
	b.ResetTimer()

	opsPerGoroutine := max(b.N/goroutines, 1)

	for g := range goroutines {
		wg.Go(func() {
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("g%d-key-%d", g, i)
				cache.Set(context.Background(), key, generateBenchValue(i))
			}
		})
	}
	wg.Wait()
}

func BenchmarkStress_ManyGoroutines_WAL(b *testing.B) {
	cache := createBenchCache(b, true, wal_domain.SyncModeBatched)
	goroutines := 100
	var wg sync.WaitGroup

	b.ReportAllocs()
	b.ResetTimer()

	opsPerGoroutine := max(b.N/goroutines, 1)

	for g := range goroutines {
		wg.Go(func() {
			for i := range opsPerGoroutine {
				key := fmt.Sprintf("g%d-key-%d", g, i)
				cache.Set(context.Background(), key, generateBenchValue(i))
			}
		})
	}
	wg.Wait()
}

func BenchmarkSetAlloc_WAL_FastCodec(b *testing.B) {
	cache := createBenchCacheWithCodec(b, true, wal_domain.SyncModeBatched, true)
	value := generateBenchValue(0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i%1000)
		cache.Set(context.Background(), key, value)
	}
}

func BenchmarkSet_WAL_SyncBatched_FastCodec(b *testing.B) {
	cache := createBenchCacheWithCodec(b, true, wal_domain.SyncModeBatched, true)

	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(context.Background(), key, generateBenchValue(i))
	}
}

func BenchmarkParallelSet_WAL_SyncBatched_FastCodec(b *testing.B) {
	cache := createBenchCacheWithCodec(b, true, wal_domain.SyncModeBatched, true)
	var counter atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := counter.Add(1)
			key := fmt.Sprintf("key-%d", i)
			cache.Set(context.Background(), key, generateBenchValue(int(i)))
		}
	})
}
