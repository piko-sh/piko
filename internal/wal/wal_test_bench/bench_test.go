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
	"context"
	"encoding/binary"
	"iter"
	"testing"
	"time"

	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

func collectEntries[K comparable, V any](it iter.Seq2[wal_domain.Entry[K, V], error]) ([]wal_domain.Entry[K, V], error) {
	entries := make([]wal_domain.Entry[K, V], 0, 16)
	for entry, err := range it {
		if err != nil {
			return entries, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

type stringKeyCodec struct{}

func (stringKeyCodec) EncodeKey(key string) ([]byte, error) {
	return []byte(key), nil
}

func (stringKeyCodec) DecodeKey(data []byte) (string, error) {
	return string(data), nil
}

func (stringKeyCodec) KeySize(key string) int {
	return len(key)
}

func (stringKeyCodec) EncodeKeyTo(key string, buffer []byte) (int, error) {
	return copy(buffer, key), nil
}

func (stringKeyCodec) DecodeKeyFrom(data []byte) (string, error) {
	return mem.String(data), nil
}

type testValue struct {
	Name  string
	Count int
}

type binaryValueCodec struct{}

func (binaryValueCodec) EncodeValue(value testValue) ([]byte, error) {
	size := 4 + len(value.Name) + 8
	buffer := make([]byte, size)
	offset := 0

	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.Name)))
	offset += 4
	copy(buffer[offset:], value.Name)
	offset += len(value.Name)
	binary.BigEndian.PutUint64(buffer[offset:], uint64(value.Count))

	return buffer, nil
}

func (binaryValueCodec) DecodeValue(data []byte) (testValue, error) {
	offset := 0
	nameLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	name := string(data[offset : offset+int(nameLen)])
	offset += int(nameLen)
	count := int(binary.BigEndian.Uint64(data[offset:]))
	return testValue{Name: name, Count: count}, nil
}

func (binaryValueCodec) ValueSize(value testValue) int {
	return 4 + len(value.Name) + 8
}

func (binaryValueCodec) EncodeValueTo(value testValue, buffer []byte) (int, error) {
	offset := 0
	binary.BigEndian.PutUint32(buffer[offset:], uint32(len(value.Name)))
	offset += 4
	copy(buffer[offset:], value.Name)
	offset += len(value.Name)
	binary.BigEndian.PutUint64(buffer[offset:], uint64(value.Count))
	offset += 8
	return offset, nil
}

func (binaryValueCodec) DecodeValueFrom(data []byte) (testValue, error) {
	offset := 0
	nameLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	name := mem.String(data[offset : offset+int(nameLen)])
	offset += int(nameLen)
	count := int(binary.BigEndian.Uint64(data[offset:]))
	return testValue{Name: name, Count: count}, nil
}

func newTestCodec() *driven_disk.BinaryCodec[string, testValue] {
	return driven_disk.NewBinaryCodec[string, testValue](stringKeyCodec{}, binaryValueCodec{})
}

func BenchmarkCodec_Encode(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = codec.Encode(entry)
	}
}

func BenchmarkCodec_EncodeWithCRC(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result, _ := codec.EncodeWithCRC(entry)
		result.Release()
	}
}

func BenchmarkCodec_Decode(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	encoded, _ := codec.Encode(entry)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result, _ := codec.Decode(encoded)
		result.Release()
	}
}

func BenchmarkCodec_DecodeWithCRC(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, _ := codec.EncodeWithCRC(entry)
	defer encResult.Release()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result, _ := codec.DecodeWithCRC(encResult.Data)
		result.Release()
	}
}

func BenchmarkValidateCRC(b *testing.B) {
	codec := newTestCodec()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2", "tag3"},
		Timestamp: time.Now().UnixNano(),
	}

	encResult, _ := codec.EncodeWithCRC(entry)
	defer encResult.Release()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = driven_disk.ValidateCRC(encResult.Data)
	}
}

func BenchmarkComputeCRC(b *testing.B) {
	data := []byte("the quick brown fox jumps over the lazy dog")

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = driven_disk.ComputeCRC(data)
	}
}

func BenchmarkDiskWAL_Append(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeNone,
	}

	wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = wal.Append(ctx, entry)
	}
}

func BenchmarkDiskWAL_AppendWithSync(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = wal.Append(ctx, entry)
	}
}

func BenchmarkDiskWAL_AppendBatched(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:               directory,
		SyncMode:          wal_domain.SyncModeBatched,
		BatchSyncInterval: 100 * time.Millisecond,
		BatchSyncCount:    100,
	}

	wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = wal.Append(ctx, entry)
	}
}

func BenchmarkDiskWAL_Recover(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeNone,
	}

	wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	for i := range 1000 {
		entry := wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: time.Now().UnixNano(),
		}
		if err := wal.Append(ctx, entry); err != nil {
			b.Fatal(err)
		}
	}

	if err := wal.Close(); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		_, _ = collectEntries(wal.Recover(ctx))

		b.StopTimer()
		_ = wal.Close()
		b.StartTimer()
	}
}

func BenchmarkDiskSnapshot_Save(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: false,
		CompressionLevel:  3,
	}

	snapshot, err := driven_disk.NewDiskSnapshot(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()

	entries := make([]wal_domain.Entry[string, testValue], 100)
	for i := range entries {
		entries[i] = wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: time.Now().UnixNano(),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = snapshot.Save(ctx, entries)
	}
}

func BenchmarkDiskSnapshot_SaveCompressed(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: true,
		CompressionLevel:  3,
	}

	snapshot, err := driven_disk.NewDiskSnapshot(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()

	entries := make([]wal_domain.Entry[string, testValue], 100)
	for i := range entries {
		entries[i] = wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: time.Now().UnixNano(),
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = snapshot.Save(ctx, entries)
	}
}

func BenchmarkDiskSnapshot_Load(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: false,
		CompressionLevel:  3,
	}

	snapshot, err := driven_disk.NewDiskSnapshot(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()

	entries := make([]wal_domain.Entry[string, testValue], 100)
	for i := range entries {
		entries[i] = wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: time.Now().UnixNano(),
		}
	}

	if err := snapshot.Save(ctx, entries); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = collectEntries(snapshot.Load(ctx))
	}
}

func BenchmarkDiskSnapshot_LoadCompressed(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: true,
		CompressionLevel:  3,
	}

	snapshot, err := driven_disk.NewDiskSnapshot(context.Background(), config, codec)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = snapshot.Close() }()

	entries := make([]wal_domain.Entry[string, testValue], 100)
	for i := range entries {
		entries[i] = wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: time.Now().UnixNano(),
		}
	}

	if err := snapshot.Save(ctx, entries); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = collectEntries(snapshot.Load(ctx))
	}
}

func BenchmarkDiskWAL_ConcurrentAppend(b *testing.B) {
	for _, numGoroutines := range []int{1, 2, 4, 8, 16, 32} {
		b.Run("goroutines-"+string(rune('0'+numGoroutines/10))+string(rune('0'+numGoroutines%10)), func(b *testing.B) {
			directory := b.TempDir()
			codec := newTestCodec()

			config := wal_domain.Config{
				Dir:      directory,
				SyncMode: wal_domain.SyncModeNone,
			}

			wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = wal.Close() }()

			ctx := context.Background()
			entry := wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "benchmark-key",
				Value:     testValue{Name: "benchmark", Count: 12345},
				Tags:      []string{"tag1", "tag2"},
				Timestamp: time.Now().UnixNano(),
			}

			b.ReportAllocs()
			b.SetParallelism(numGoroutines)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = wal.Append(ctx, entry)
				}
			})
		})
	}
}

func BenchmarkDiskWAL_ConcurrentAppendWithSync(b *testing.B) {
	for _, numGoroutines := range []int{1, 4, 16} {
		b.Run("goroutines-"+string(rune('0'+numGoroutines/10))+string(rune('0'+numGoroutines%10)), func(b *testing.B) {
			directory := b.TempDir()
			codec := newTestCodec()

			config := wal_domain.Config{
				Dir:      directory,
				SyncMode: wal_domain.SyncModeEveryWrite,
			}

			wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = wal.Close() }()

			ctx := context.Background()
			entry := wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "benchmark-key",
				Value:     testValue{Name: "benchmark", Count: 12345},
				Tags:      []string{"tag1", "tag2"},
				Timestamp: time.Now().UnixNano(),
			}

			b.ReportAllocs()
			b.SetParallelism(numGoroutines)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = wal.Append(ctx, entry)
				}
			})
		})
	}
}

func BenchmarkDiskWAL_ConcurrentAppendBatched(b *testing.B) {
	for _, numGoroutines := range []int{1, 4, 16} {
		b.Run("goroutines-"+string(rune('0'+numGoroutines/10))+string(rune('0'+numGoroutines%10)), func(b *testing.B) {
			directory := b.TempDir()
			codec := newTestCodec()

			config := wal_domain.Config{
				Dir:               directory,
				SyncMode:          wal_domain.SyncModeBatched,
				BatchSyncInterval: 10 * time.Millisecond,
				BatchSyncCount:    100,
			}

			wal, err := driven_disk.NewDiskWAL(context.Background(), config, codec)
			if err != nil {
				b.Fatal(err)
			}
			defer func() { _ = wal.Close() }()

			ctx := context.Background()
			entry := wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "benchmark-key",
				Value:     testValue{Name: "benchmark", Count: 12345},
				Tags:      []string{"tag1", "tag2"},
				Timestamp: time.Now().UnixNano(),
			}

			b.ReportAllocs()
			b.SetParallelism(numGoroutines)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = wal.Append(ctx, entry)
				}
			})
		})
	}
}

func BenchmarkEntry_Sizes(b *testing.B) {
	codec := newTestCodec()

	sizes := []struct {
		name  string
		entry wal_domain.Entry[string, testValue]
	}{
		{
			name: "minimal",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpDelete,
				Key:       "k",
				Timestamp: 1,
			},
		},
		{
			name: "small",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "key",
				Value:     testValue{Name: "v", Count: 1},
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "medium",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "medium-length-key-name",
				Value:     testValue{Name: "medium length value name here", Count: 12345},
				Tags:      []string{"tag1", "tag2", "tag3"},
				Timestamp: time.Now().UnixNano(),
			},
		},
		{
			name: "large",
			entry: wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "this-is-a-very-long-key-name-for-testing-purposes-to-see-how-it-affects-performance",
				Value:     testValue{Name: "this is a very long value name for testing purposes to measure encoding performance with larger payloads", Count: 999999},
				Tags:      []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8", "tag9", "tag10"},
				ExpiresAt: time.Now().Add(time.Hour).UnixNano(),
				Timestamp: time.Now().UnixNano(),
			},
		},
	}

	for _, tc := range sizes {
		b.Run(tc.name+"_encode", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result, _ := codec.EncodeWithCRC(tc.entry)
				result.Release()
			}
		})

		encResult, _ := codec.EncodeWithCRC(tc.entry)
		b.Run(tc.name+"_decode", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result, _ := codec.DecodeWithCRC(encResult.Data)
				result.Release()
			}
		})
		encResult.Release()
	}
}
