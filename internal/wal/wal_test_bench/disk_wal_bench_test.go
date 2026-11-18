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

package wal_test_bench

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

const (
	benchDirectoryEnv = "PIKO_WAL_BENCH_DIR"
)

type entrySize struct {
	name  string
	bytes int
}

type syncPolicy struct {
	name string
	mode wal_domain.SyncMode
}

var (
	entrySizes = []entrySize{
		{name: "small", bytes: 32},
		{name: "1KiB", bytes: 1 << 10},
		{name: "64KiB", bytes: 64 << 10},
	}
	syncPolicies = []syncPolicy{
		{name: "none", mode: wal_domain.SyncModeNone},
		{name: "batched", mode: wal_domain.SyncModeBatched},
		{name: "every", mode: wal_domain.SyncModeEveryWrite},
	}
)

type slowKeyCodec struct{}

func (slowKeyCodec) EncodeKey(key string) ([]byte, error) { return []byte(key), nil }

func (slowKeyCodec) DecodeKey(data []byte) (string, error) { return string(data), nil }

type slowValueCodec struct{}

func (slowValueCodec) EncodeValue(value testValue) ([]byte, error) {
	return binaryValueCodec{}.EncodeValue(value)
}

func (slowValueCodec) DecodeValue(data []byte) (testValue, error) {
	return binaryValueCodec{}.DecodeValue(data)
}

func benchCodec(fast bool) *driven_disk.BinaryCodec[string, testValue] {
	if fast {
		return newTestCodec()
	}
	return driven_disk.NewBinaryCodec[string, testValue](slowKeyCodec{}, slowValueCodec{})
}

func benchDirectory(b *testing.B) string {
	b.Helper()
	base := os.Getenv(benchDirectoryEnv)
	if base == "" {
		return b.TempDir()
	}
	directory, err := os.MkdirTemp(base, "walbench")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = os.RemoveAll(directory) })
	return directory
}

func benchEntry(index, size int) wal_domain.Entry[string, testValue] {
	return wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       fmt.Sprintf("key-%d", index%4096),
		Value:     testValue{Name: strings.Repeat("v", size), Count: index},
		Tags:      []string{"tag-a", "tag-b"},
		Timestamp: int64(index),
		ExpiresAt: 0,
		Batch:     nil,
	}
}

func openBenchWAL(b *testing.B, directory string, mode wal_domain.SyncMode, fast bool) *driven_disk.DiskWAL[string, testValue] {
	b.Helper()
	config := wal_domain.DefaultConfig(directory)
	config.SyncMode = mode
	wal, err := driven_disk.NewDiskWAL(context.Background(), config, benchCodec(fast))
	if err != nil {
		b.Fatal(err)
	}
	return wal
}

func newBenchWAL(b *testing.B, mode wal_domain.SyncMode, fast bool) *driven_disk.DiskWAL[string, testValue] {
	b.Helper()
	wal := openBenchWAL(b, benchDirectory(b), mode, fast)
	b.Cleanup(func() { _ = wal.Close() })
	return wal
}

func reportLatency(b *testing.B, samples []time.Duration) {
	b.Helper()
	if len(samples) == 0 {
		return
	}
	slices.SortFunc(samples, cmp.Compare[time.Duration])
	b.ReportMetric(float64(samples[len(samples)/2].Nanoseconds()), "p50-ns")
	b.ReportMetric(float64(samples[len(samples)*99/100].Nanoseconds()), "p99-ns")
}

func BenchmarkWAL_EncodeEntry(b *testing.B) {
	for _, fast := range []bool{false, true} {
		for _, size := range entrySizes {
			b.Run(fmt.Sprintf("fast=%t/size=%s", fast, size.name), func(b *testing.B) {
				codec := benchCodec(fast)
				entry := benchEntry(1, size.bytes)
				b.ReportAllocs()
				b.SetBytes(int64(size.bytes))
				for b.Loop() {
					result, err := codec.EncodeWithLengthAndCRC(entry)
					if err != nil {
						b.Fatal(err)
					}
					result.Release()
				}
			})
		}
	}
}

func BenchmarkWAL_Append(b *testing.B) {
	for _, policy := range syncPolicies {
		for _, size := range entrySizes {
			b.Run(fmt.Sprintf("sync=%s/size=%s", policy.name, size.name), func(b *testing.B) {
				wal := newBenchWAL(b, policy.mode, false)
				entry := benchEntry(1, size.bytes)
				ctx := context.Background()
				b.ReportAllocs()
				b.SetBytes(int64(size.bytes))
				for b.Loop() {
					if err := wal.Append(ctx, entry); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkWAL_AppendParallel(b *testing.B) {
	for _, policy := range syncPolicies {
		for _, size := range entrySizes[:2] {
			b.Run(fmt.Sprintf("sync=%s/size=%s", policy.name, size.name), func(b *testing.B) {
				wal := newBenchWAL(b, policy.mode, false)
				entry := benchEntry(1, size.bytes)
				ctx := context.Background()
				var mu sync.Mutex
				var samples []time.Duration
				b.ReportAllocs()
				b.SetBytes(int64(size.bytes))
				b.RunParallel(func(pb *testing.PB) {
					local := make([]time.Duration, 0, 1024)
					count := 0
					for pb.Next() {
						start := time.Now()
						if err := wal.Append(ctx, entry); err != nil {
							b.Error(err)
							return
						}
						if count%16 == 0 {
							local = append(local, time.Since(start))
						}
						count++
					}
					mu.Lock()
					samples = append(samples, local...)
					mu.Unlock()
				})
				reportLatency(b, samples)
			})
		}
	}
}

func BenchmarkWAL_Enqueue(b *testing.B) {
	for _, size := range entrySizes {
		b.Run("size="+size.name, func(b *testing.B) {
			wal := newBenchWAL(b, wal_domain.SyncModeBatched, false)
			entry := benchEntry(1, size.bytes)
			ctx := context.Background()
			b.ReportAllocs()
			b.SetBytes(int64(size.bytes))
			index := 0
			for b.Loop() {
				wait, err := wal.Enqueue(ctx, entry)
				if err != nil {
					b.Fatal(err)
				}
				if index%256 == 255 {
					if err := wait(); err != nil {
						b.Fatal(err)
					}
				}
				index++
			}
		})
	}
}

func BenchmarkWAL_EnqueueParallel(b *testing.B) {
	wal := newBenchWAL(b, wal_domain.SyncModeBatched, false)
	entry := benchEntry(1, entrySizes[0].bytes)
	ctx := context.Background()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		index := 0
		for pb.Next() {
			wait, err := wal.Enqueue(ctx, entry)
			if err != nil {
				b.Error(err)
				return
			}
			if index%64 == 63 {
				if err := wait(); err != nil {
					b.Error(err)
					return
				}
			}
			index++
		}
	})
}

func BenchmarkWAL_AppendBatch(b *testing.B) {
	for _, count := range []int{16, 256} {
		b.Run(fmt.Sprintf("entries=%d", count), func(b *testing.B) {
			wal := newBenchWAL(b, wal_domain.SyncModeBatched, false)
			members := make([]wal_domain.Entry[string, testValue], count)
			for index := range members {
				members[index] = benchEntry(index, 64)
			}
			batch := wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpBatch,
				Batch:     members,
				Timestamp: 1,
			}
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				if err := wal.Append(ctx, batch); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkWAL_RecoverEntries(b *testing.B) {
	for _, count := range []int{10_000, 100_000} {
		b.Run(fmt.Sprintf("entries=%d", count), func(b *testing.B) {
			directory := benchDirectory(b)
			ctx := context.Background()
			writer := openBenchWAL(b, directory, wal_domain.SyncModeNone, false)
			var wait func() error
			for index := range count {
				var err error
				wait, err = writer.Enqueue(ctx, benchEntry(index, 64))
				if err != nil {
					b.Fatal(err)
				}
			}
			if err := wait(); err != nil {
				b.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				reader := openBenchWAL(b, directory, wal_domain.SyncModeNone, false)
				b.StartTimer()
				recovered := 0
				for _, err := range reader.Recover(ctx) {
					if err != nil {
						b.Fatal(err)
					}
					recovered++
				}
				b.StopTimer()
				if recovered != count {
					b.Fatalf("recovered %d entries, want %d", recovered, count)
				}
				_ = reader.Close()
				b.StartTimer()
			}
		})
	}
}

func BenchmarkWAL_SnapshotEntries(b *testing.B) {
	for _, compressed := range []bool{false, true} {
		for _, count := range []int{10_000, 100_000} {
			b.Run(fmt.Sprintf("compressed=%t/entries=%d", compressed, count), func(b *testing.B) {
				config := wal_domain.DefaultConfig(benchDirectory(b))
				config.EnableCompression = compressed
				store, err := driven_disk.NewDiskSnapshot(context.Background(), config, newTestCodec())
				if err != nil {
					b.Fatal(err)
				}
				b.Cleanup(func() { _ = store.Close() })
				entries := make([]wal_domain.Entry[string, testValue], count)
				for index := range entries {
					entries[index] = benchEntry(index, 64)
				}
				ctx := context.Background()

				b.Run("save", func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						if err := store.Save(ctx, entries); err != nil {
							b.Fatal(err)
						}
					}
				})
				b.Run("load", func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						loaded := 0
						for _, err := range store.Load(ctx) {
							if err != nil {
								b.Fatal(err)
							}
							loaded++
						}
						if loaded != count {
							b.Fatalf("loaded %d entries, want %d", loaded, count)
						}
					}
				})
			})
		}
	}
}
