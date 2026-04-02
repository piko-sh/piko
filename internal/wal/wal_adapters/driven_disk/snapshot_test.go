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

package driven_disk

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
)

func newTestSnapshot(t *testing.T, enableCompression bool) (*DiskSnapshot[string, testValue], string) {
	t.Helper()

	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: enableCompression,
		CompressionLevel:  3,
	}

	snapshot, err := NewDiskSnapshot(context.Background(), config, codec)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = snapshot.Close()
	})

	return snapshot, directory
}

func TestDiskSnapshot_SaveAndLoad(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Tags:      []string{"tag1"},
			Timestamp: time.Now().UnixNano(),
		},
		{
			Operation: wal_domain.OpSet,
			Key:       "key2",
			Value:     testValue{Name: "value2", Count: 2},
			Tags:      []string{"tag2", "tag3"},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	assert.True(t, snapshot.Exists())

	loaded, err := collectEntries(snapshot.Load(ctx))
	require.NoError(t, err)
	require.Len(t, loaded, 2)

	assert.Equal(t, entries[0].Key, loaded[0].Key)
	assert.Equal(t, entries[0].Value, loaded[0].Value)
	assert.Equal(t, entries[1].Key, loaded[1].Key)
	assert.Equal(t, entries[1].Value, loaded[1].Value)
}

func TestDiskSnapshot_SaveAndLoadCompressed(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, true)
	ctx := context.Background()

	entries := make([]wal_domain.Entry[string, testValue], 100)
	for i := range entries {
		entries[i] = wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "repeated value for compression", Count: i},
			Tags:      []string{"tag1", "tag2"},
			Timestamp: time.Now().UnixNano(),
		}
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	loaded, err := collectEntries(snapshot.Load(ctx))
	require.NoError(t, err)
	require.Len(t, loaded, 100)

	assert.Equal(t, entries[0].Key, loaded[0].Key)
	assert.Equal(t, entries[99].Value.Count, loaded[99].Value.Count)
}

func TestDiskSnapshot_LoadNotFound(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)
	ctx := context.Background()

	assert.False(t, snapshot.Exists())

	_, err := collectEntries(snapshot.Load(ctx))
	assert.ErrorIs(t, err, wal_domain.ErrSnapshotNotFound)
}

func TestDiskSnapshot_Delete(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)
	assert.True(t, snapshot.Exists())

	err = snapshot.Delete(ctx)
	require.NoError(t, err)
	assert.False(t, snapshot.Exists())

	err = snapshot.Delete(ctx)
	require.NoError(t, err)
}

func TestDiskSnapshot_EmptySnapshot(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)
	ctx := context.Background()

	err := snapshot.Save(ctx, []wal_domain.Entry[string, testValue]{})
	require.NoError(t, err)

	loaded, err := collectEntries(snapshot.Load(ctx))
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestDiskSnapshot_AtomicWrite(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	tempPath := filepath.Join(directory, snapshot.config.SnapshotFileName+".tmp")
	_, err = os.Stat(tempPath)
	assert.True(t, os.IsNotExist(err), "temp file should not exist after save")
}

func TestDiskSnapshot_CorruptedData(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	snapshotPath := filepath.Join(directory, snapshot.config.SnapshotFileName)
	data, err := os.ReadFile(snapshotPath)
	require.NoError(t, err)

	data[snapshotHeaderSize+10] ^= 0xFF

	err = os.WriteFile(snapshotPath, data, 0600)
	require.NoError(t, err)

	_, err = collectEntries(snapshot.Load(ctx))
	assert.ErrorIs(t, err, wal_domain.ErrCorrupted)
}

func TestDiskSnapshot_CorruptedHeader(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	snapshotPath := filepath.Join(directory, snapshot.config.SnapshotFileName)
	data, err := os.ReadFile(snapshotPath)
	require.NoError(t, err)

	data[0] ^= 0xFF

	err = os.WriteFile(snapshotPath, data, 0600)
	require.NoError(t, err)

	_, err = collectEntries(snapshot.Load(ctx))
	assert.Error(t, err)
}

func TestDiskSnapshot_Path(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)

	expectedPath := filepath.Join(directory, "snapshot.piko")
	assert.Equal(t, expectedPath, snapshot.Path())
}

func TestNewDiskSnapshot_NilCodec(t *testing.T) {
	config := wal_domain.Config{Dir: t.TempDir()}

	_, err := NewDiskSnapshot[string, testValue](context.Background(), config, nil)
	assert.ErrorIs(t, err, wal_domain.ErrCodecRequired)
}

func TestDiskSnapshot_CompressionRatio(t *testing.T) {
	ctx := context.Background()

	entries := make([]wal_domain.Entry[string, testValue], 1000)
	for i := range entries {
		entries[i] = wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "the quick brown fox jumps over the lazy dog", Count: i},
			Tags:      []string{"tag1", "tag2", "tag3"},
			Timestamp: time.Now().UnixNano(),
		}
	}

	snapshotNoComp, dirNoComp := newTestSnapshot(t, false)
	err := snapshotNoComp.Save(ctx, entries)
	require.NoError(t, err)

	snapshotComp, dirComp := newTestSnapshot(t, true)
	err = snapshotComp.Save(ctx, entries)
	require.NoError(t, err)

	infoNoComp, err := os.Stat(filepath.Join(dirNoComp, "snapshot.piko"))
	require.NoError(t, err)

	infoComp, err := os.Stat(filepath.Join(dirComp, "snapshot.piko"))
	require.NoError(t, err)

	t.Logf("Uncompressed: %d bytes, Compressed: %d bytes, Ratio: %.2f%%",
		infoNoComp.Size(), infoComp.Size(),
		float64(infoComp.Size())/float64(infoNoComp.Size())*100)

	assert.Less(t, infoComp.Size(), infoNoComp.Size())

	loadedNoComp, err := collectEntries(snapshotNoComp.Load(ctx))
	require.NoError(t, err)
	assert.Len(t, loadedNoComp, 1000)

	loadedComp, err := collectEntries(snapshotComp.Load(ctx))
	require.NoError(t, err)
	assert.Len(t, loadedComp, 1000)
}

func BenchmarkDiskSnapshot_Save(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: true,
		CompressionLevel:  3,
	}

	snapshot, err := NewDiskSnapshot(context.Background(), config, codec)
	require.NoError(b, err)
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
		EnableCompression: true,
		CompressionLevel:  3,
	}

	snapshot, err := NewDiskSnapshot(context.Background(), config, codec)
	require.NoError(b, err)
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

	err = snapshot.Save(ctx, entries)
	require.NoError(b, err)

	b.ResetTimer()
	for b.Loop() {
		_, _ = collectEntries(snapshot.Load(ctx))
	}
}

func TestDiskSnapshot_WithClock(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	mockClk := clock.NewMockClock(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))

	config := wal_domain.Config{
		Dir:               directory,
		EnableCompression: false,
		CompressionLevel:  3,
	}

	snapshot, err := NewDiskSnapshot(context.Background(), config, codec, WithSnapshotClock[string, testValue](mockClk))
	require.NoError(t, err)
	defer func() { _ = snapshot.Close() }()

	assert.NotNil(t, snapshot)
}

func TestDiskSnapshot_DoubleClose(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)

	err := snapshot.Close()
	assert.NoError(t, err)

	err = snapshot.Close()
	assert.NoError(t, err)
}

func TestDiskSnapshot_LoadCompressedWithoutDecoder(t *testing.T) {
	ctx := context.Background()

	directory := t.TempDir()
	codec := newTestCodec()

	configWithComp := wal_domain.Config{
		Dir:               directory,
		EnableCompression: true,
		CompressionLevel:  3,
	}

	snapshotWithComp, err := NewDiskSnapshot(context.Background(), configWithComp, codec)
	require.NoError(t, err)

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err = snapshotWithComp.Save(ctx, entries)
	require.NoError(t, err)
	require.NoError(t, snapshotWithComp.Close())

	configNoComp := wal_domain.Config{
		Dir:               directory,
		EnableCompression: false,
		CompressionLevel:  3,
	}

	snapshotNoComp, err := NewDiskSnapshot(context.Background(), configNoComp, codec)
	require.NoError(t, err)
	defer func() { _ = snapshotNoComp.Close() }()

	_, err = collectEntries(snapshotNoComp.Load(ctx))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "compressed")
}

func TestDiskSnapshot_InvalidVersion(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	snapshotPath := filepath.Join(directory, snapshot.config.SnapshotFileName)
	data, err := os.ReadFile(snapshotPath)
	require.NoError(t, err)

	data[4] = 255

	err = os.WriteFile(snapshotPath, data, 0600)
	require.NoError(t, err)

	_, err = collectEntries(snapshot.Load(ctx))
	assert.Error(t, err)
}

func TestDiskSnapshot_LoadTruncatedEntryData(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
		{
			Operation: wal_domain.OpSet,
			Key:       "key2",
			Value:     testValue{Name: "value2", Count: 2},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	snapshotPath := filepath.Join(directory, snapshot.config.SnapshotFileName)
	data, err := os.ReadFile(snapshotPath)
	require.NoError(t, err)

	truncatedData := data[:40]
	err = os.WriteFile(snapshotPath, truncatedData, 0600)
	require.NoError(t, err)

	_, err = collectEntries(snapshot.Load(ctx))
	assert.Error(t, err)
}

func TestDiskSnapshot_LoadCorruptedEntryLength(t *testing.T) {
	snapshot, directory := newTestSnapshot(t, false)
	ctx := context.Background()

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err := snapshot.Save(ctx, entries)
	require.NoError(t, err)

	snapshotPath := filepath.Join(directory, snapshot.config.SnapshotFileName)
	data, err := os.ReadFile(snapshotPath)
	require.NoError(t, err)

	data[32] = 0xFF
	data[33] = 0xFF
	data[34] = 0xFF
	data[35] = 0xFF

	err = os.WriteFile(snapshotPath, data, 0600)
	require.NoError(t, err)

	_, err = collectEntries(snapshot.Load(ctx))
	assert.Error(t, err)
}

func TestDiskSnapshot_SaveToReadOnlyDir(t *testing.T) {

	if os.Getuid() == 0 {
		t.Skip("Cannot test permission errors as root")
	}

	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir: directory,
	}

	snapshot, err := NewDiskSnapshot(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = snapshot.Close() }()

	ctx := context.Background()
	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "key1",
			Value:     testValue{Name: "value1", Count: 1},
			Timestamp: time.Now().UnixNano(),
		},
	}

	err = os.Chmod(directory, 0o555)
	require.NoError(t, err)
	defer func() { _ = os.Chmod(directory, 0o755) }()

	err = snapshot.Save(ctx, entries)
	assert.Error(t, err)
}

func TestDiskSnapshot_DeleteNonExistent(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)
	ctx := context.Background()

	assert.False(t, snapshot.Exists())
	err := snapshot.Delete(ctx)
	require.NoError(t, err)
}

func TestDiskSnapshot_CloseTwice(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir: directory,
	}

	snapshot, err := NewDiskSnapshot(context.Background(), config, codec)
	require.NoError(t, err)

	err = snapshot.Close()
	require.NoError(t, err)

	err = snapshot.Close()
	require.NoError(t, err)
}

func TestDiskSnapshot_SaveEmptyEntries(t *testing.T) {
	snapshot, _ := newTestSnapshot(t, false)
	ctx := context.Background()

	err := snapshot.Save(ctx, []wal_domain.Entry[string, testValue]{})
	require.NoError(t, err)

	loaded, err := collectEntries(snapshot.Load(ctx))
	require.NoError(t, err)
	assert.Empty(t, loaded)
}
