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
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/wdk/clock"
	"piko.sh/safedisk"
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

func newTestWAL(t *testing.T) (*DiskWAL[string, testValue], string) {
	t.Helper()

	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = wal.Close()
	})

	return wal, directory
}

func TestDiskWAL_AppendAndRecover(t *testing.T) {
	wal, _ := newTestWAL(t)
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
		{
			Operation: wal_domain.OpDelete,
			Key:       "key1",
			Timestamp: time.Now().UnixNano(),
		},
	}

	for _, entry := range entries {
		err := wal.Append(ctx, entry)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, wal.EntryCount())
	assert.Greater(t, wal.Size(), int64(0))

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	require.Len(t, recovered, 3)

	assert.Equal(t, entries[0].Key, recovered[0].Key)
	assert.Equal(t, entries[0].Value, recovered[0].Value)
	assert.Equal(t, entries[1].Key, recovered[1].Key)
	assert.Equal(t, entries[2].Operation, recovered[2].Operation)
}

func TestDiskWAL_RecoverAfterReopen(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	entries := []wal_domain.Entry[string, testValue]{
		{
			Operation: wal_domain.OpSet,
			Key:       "persistent-key",
			Value:     testValue{Name: "persistent", Count: 42},
			Timestamp: time.Now().UnixNano(),
		},
	}

	for _, entry := range entries {
		err := wal1.Append(ctx, entry)
		require.NoError(t, err)
	}

	require.NoError(t, wal1.Close())

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal2.Close() }()

	recovered, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)
	require.Len(t, recovered, 1)

	assert.Equal(t, "persistent-key", recovered[0].Key)
	assert.Equal(t, testValue{Name: "persistent", Count: 42}, recovered[0].Value)
}

func TestDiskWAL_Truncate(t *testing.T) {
	wal, _ := newTestWAL(t)
	ctx := context.Background()

	for i := range 5 {
		err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 5, wal.EntryCount())

	err := wal.Truncate(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(0), wal.Size())

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	assert.Empty(t, recovered)
}

func TestDiskWAL_RecoverDetectsCorruption(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	for i := range 5 {
		err := wal1.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	sizeBefore := wal1.Size()
	require.NoError(t, wal1.Close())

	walPath := filepath.Join(directory, "data.wal")
	data, err := os.ReadFile(walPath)
	require.NoError(t, err)

	corruptPos := len(data) * 2 / 3
	data[corruptPos] ^= 0xFF

	err = os.WriteFile(walPath, data, 0600)
	require.NoError(t, err)

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal2.Close() }()

	recovered, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)

	assert.Less(t, len(recovered), 5, "should have truncated some entries")
	assert.Greater(t, len(recovered), 0, "should have recovered some entries")

	assert.Less(t, wal2.Size(), sizeBefore)
}

func TestDiskWAL_RecoverPartialWrite(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	for i := range 3 {
		err := wal1.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	require.NoError(t, wal1.Close())

	walPath := filepath.Join(directory, "data.wal")
	data, err := os.ReadFile(walPath)
	require.NoError(t, err)

	truncatedData := data[:len(data)-10]
	err = os.WriteFile(walPath, truncatedData, 0600)
	require.NoError(t, err)

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal2.Close() }()

	recovered, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)

	assert.Equal(t, 2, len(recovered))
}

func TestDiskWAL_ClosedOperations(t *testing.T) {
	wal, _ := newTestWAL(t)
	ctx := context.Background()

	require.NoError(t, wal.Close())

	err := wal.Append(ctx, wal_domain.Entry[string, testValue]{})
	assert.ErrorIs(t, err, wal_domain.ErrWALClosed)

	_, err = collectEntries(wal.Recover(ctx))
	assert.ErrorIs(t, err, wal_domain.ErrWALClosed)

	err = wal.Truncate(ctx)
	assert.ErrorIs(t, err, wal_domain.ErrWALClosed)

	err = wal.Sync(ctx)
	assert.ErrorIs(t, err, wal_domain.ErrWALClosed)

	err = wal.Close()
	assert.ErrorIs(t, err, wal_domain.ErrWALClosed)
}

func TestDiskWAL_SyncModes(t *testing.T) {
	codec := newTestCodec()
	ctx := context.Background()

	testCases := []struct {
		name     string
		syncMode wal_domain.SyncMode
	}{
		{name: "sync none", syncMode: wal_domain.SyncModeNone},
		{name: "sync every write", syncMode: wal_domain.SyncModeEveryWrite},
		{name: "sync batched", syncMode: wal_domain.SyncModeBatched},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			config := wal_domain.Config{
				Dir:               directory,
				SyncMode:          tc.syncMode,
				BatchSyncInterval: 50 * time.Millisecond,
				BatchSyncCount:    5,
			}

			wal, err := NewDiskWAL(context.Background(), config, codec)
			require.NoError(t, err)
			defer func() { _ = wal.Close() }()

			for i := range 10 {
				err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
					Operation: wal_domain.OpSet,
					Key:       "key",
					Value:     testValue{Name: "value", Count: i},
					Timestamp: time.Now().UnixNano(),
				})
				require.NoError(t, err)
			}

			recovered, err := collectEntries(wal.Recover(ctx))
			require.NoError(t, err)
			assert.Len(t, recovered, 10)
		})
	}
}

func TestDiskWAL_ContextCancellation(t *testing.T) {
	wal, _ := newTestWAL(t)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key",
		Value:     testValue{Name: "value", Count: 1},
		Timestamp: time.Now().UnixNano(),
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDiskWAL_Path(t *testing.T) {
	wal, directory := newTestWAL(t)

	expectedPath := filepath.Join(directory, "data.wal")
	assert.Equal(t, expectedPath, wal.Path())
}

func TestNewDiskWAL_InvalidConfig(t *testing.T) {
	codec := newTestCodec()

	testCases := []struct {
		name   string
		config wal_domain.Config
	}{
		{
			name:   "empty directory",
			config: wal_domain.Config{},
		},
		{
			name:   "invalid compression level",
			config: wal_domain.Config{Dir: t.TempDir(), CompressionLevel: 100},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDiskWAL(context.Background(), tc.config, codec)
			assert.Error(t, err)
		})
	}
}

func TestNewDiskWAL_NilCodec(t *testing.T) {
	config := wal_domain.Config{Dir: t.TempDir()}

	_, err := NewDiskWAL[string, testValue](context.Background(), config, nil)
	assert.ErrorIs(t, err, wal_domain.ErrCodecRequired)
}

func BenchmarkDiskWAL_Append(b *testing.B) {
	directory := b.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(b, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

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

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(b, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "benchmark-key",
		Value:     testValue{Name: "benchmark", Count: 12345},
		Tags:      []string{"tag1", "tag2"},
		Timestamp: time.Now().UnixNano(),
	}

	b.ResetTimer()
	for b.Loop() {
		_ = wal.Append(ctx, entry)
	}
}

func TestDiskWAL_WithClock(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	mockClk := clock.NewMockClock(time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC))

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec, WithWALClock[string, testValue](mockClk))
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	assert.NotNil(t, wal)
}

func TestDiskWAL_ConcurrentAppend(t *testing.T) {
	wal, _ := newTestWAL(t)
	ctx := context.Background()

	const numGoroutines = 10
	const entriesPerGoroutine = 10

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*entriesPerGoroutine)

	for g := range numGoroutines {
		wg.Go(func() {
			goroutineID := g
			for i := range entriesPerGoroutine {
				entry := wal_domain.Entry[string, testValue]{
					Operation: wal_domain.OpSet,
					Key:       fmt.Sprintf("key-%d-%d", goroutineID, i),
					Value:     testValue{Name: "concurrent", Count: goroutineID*100 + i},
					Timestamp: time.Now().UnixNano(),
				}
				if err := wal.Append(ctx, entry); err != nil {
					errChan <- err
				}
			}
		})
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent append error: %v", err)
	}

	assert.Equal(t, numGoroutines*entriesPerGoroutine, wal.EntryCount())

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	assert.Len(t, recovered, numGoroutines*entriesPerGoroutine)
}

func TestDiskWAL_ConcurrentAppendAndClose(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 50 {
			entry := wal_domain.Entry[string, testValue]{
				Operation: wal_domain.OpSet,
				Key:       "key",
				Value:     testValue{Name: "value", Count: 1},
				Timestamp: time.Now().UnixNano(),
			}
			err := wal.Append(ctx, entry)
			if err != nil {

				return
			}
			time.Sleep(time.Millisecond)
		}
	})

	time.Sleep(10 * time.Millisecond)
	err = wal.Close()
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(5 * time.Second):
		t.Fatal("Append blocked forever - DEADLOCK detected!")
	}
}

func TestDiskWAL_Sync(t *testing.T) {
	wal, _ := newTestWAL(t)
	ctx := context.Background()

	err := wal.Sync(ctx)
	require.NoError(t, err)

	for i := range 3 {
		err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	err = wal.Sync(ctx)
	require.NoError(t, err)
}

func TestDiskWAL_BatchedSyncWithTicker(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:               directory,
		SyncMode:          wal_domain.SyncModeBatched,
		BatchSyncInterval: 10 * time.Millisecond,
		BatchSyncCount:    100,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	for i := range 5 {
		err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	time.Sleep(30 * time.Millisecond)

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	assert.Len(t, recovered, 5)
}

func TestDiskWAL_RecoverPartialLengthPrefix(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	for i := range 2 {
		err := wal1.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "key",
			Value:     testValue{Name: "value", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	require.NoError(t, wal1.Close())

	walPath := filepath.Join(directory, "data.wal")
	file, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = file.Write([]byte{0x00, 0x00})
	require.NoError(t, err)
	require.NoError(t, file.Close())

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal2.Close() }()

	recovered, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)

	assert.Equal(t, 2, len(recovered))
}

func TestDiskWAL_RecoverEmptyFile(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	walPath := filepath.Join(directory, "data.wal")
	file, err := os.Create(walPath)
	require.NoError(t, err)
	require.NoError(t, file.Close())

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	assert.Empty(t, recovered)
}

func TestDiskWAL_RecoverReadError(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	err = wal1.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key",
		Value:     testValue{Name: "value", Count: 1},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)
	require.NoError(t, wal1.Close())

	walPath := filepath.Join(directory, "data.wal")
	data, err := os.ReadFile(walPath)
	require.NoError(t, err)

	newData := append(data, 0x00, 0x00, 0xFF, 0xFF)
	err = os.WriteFile(walPath, newData, 0600)
	require.NoError(t, err)

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal2.Close() }()

	recovered, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)

	assert.Equal(t, 1, len(recovered))
}

func TestDiskWAL_AppendCloseRace_NoDeadlock(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	ctx := context.Background()
	errChan := make(chan error, 1)

	go func() {
		entry := wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "race-key",
			Value:     testValue{Name: "race", Count: 1},
			Timestamp: time.Now().UnixNano(),
		}
		errChan <- wal.Append(ctx, entry)
	}()

	time.Sleep(10 * time.Millisecond)

	closeErr := wal.Close()
	require.NoError(t, closeErr)

	select {
	case err := <-errChan:

		if err != nil {
			assert.ErrorIs(t, err, wal_domain.ErrWALClosed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Append blocked forever - DEADLOCK detected!")
	}
}

func TestDiskWAL_ManyAppendersCloseConcurrently(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	ctx := context.Background()

	const numGoroutines = 100
	const entriesPerGoroutine = 100

	var wg sync.WaitGroup
	startSignal := make(chan struct{})

	for g := range numGoroutines {
		wg.Go(func() {
			<-startSignal

			for i := range entriesPerGoroutine {
				entry := wal_domain.Entry[string, testValue]{
					Operation: wal_domain.OpSet,
					Key:       fmt.Sprintf("key-%d-%d", g, i),
					Value:     testValue{Name: "stress", Count: g*1000 + i},
					Timestamp: time.Now().UnixNano(),
				}

				err := wal.Append(ctx, entry)
				if err != nil {

					if !errors.Is(err, wal_domain.ErrWALClosed) {
						t.Errorf("unexpected error: %v", err)
					}
					return
				}
			}
		})
	}

	wg.Go(func() {
		<-startSignal

		time.Sleep(5 * time.Millisecond)

		err := wal.Close()
		if err != nil && !errors.Is(err, wal_domain.ErrWALClosed) {
			t.Errorf("Close returned unexpected error: %v", err)
		}
	})

	close(startSignal)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(30 * time.Second):
		t.Fatal("Test timed out - possible deadlock!")
	}
}

func TestDiskWAL_AppendDuringDrainPending(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	ctx := context.Background()

	for i := range 5 {
		err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       fmt.Sprintf("pre-close-%d", i),
			Value:     testValue{Name: "pre", Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	appendDone := make(chan error, 1)

	go func() {
		entry := wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "racing-entry",
			Value:     testValue{Name: "race", Count: 999},
			Timestamp: time.Now().UnixNano(),
		}
		appendDone <- wal.Append(ctx, entry)
	}()

	time.Sleep(100 * time.Microsecond)

	closeErr := wal.Close()
	require.NoError(t, closeErr)

	select {
	case err := <-appendDone:

		if err != nil {
			assert.ErrorIs(t, err, wal_domain.ErrWALClosed)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Append blocked forever - DEADLOCK!")
	}
}

func TestDiskWAL_RepeatedOpenAppendCloseCycles(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	const cycles = 50
	const appendersPerCycle = 10
	const entriesPerAppender = 20

	for cycle := range cycles {
		wal, err := NewDiskWAL(context.Background(), config, codec)
		require.NoError(t, err, "cycle %d", cycle)

		ctx := context.Background()
		var wg sync.WaitGroup

		for a := range appendersPerCycle {
			wg.Go(func() {
				for i := range entriesPerAppender {
					entry := wal_domain.Entry[string, testValue]{
						Operation: wal_domain.OpSet,
						Key:       fmt.Sprintf("c%d-a%d-e%d", cycle, a, i),
						Value:     testValue{Name: "cycle", Count: i},
						Timestamp: time.Now().UnixNano(),
					}
					err := wal.Append(ctx, entry)
					if errors.Is(err, wal_domain.ErrWALClosed) {
						return
					}
				}
			})
		}

		time.Sleep(2 * time.Millisecond)
		closeErr := wal.Close()
		require.NoError(t, closeErr, "cycle %d close", cycle)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:

		case <-time.After(10 * time.Second):
			t.Fatalf("Cycle %d timed out - deadlock!", cycle)
		}
	}
}

func TestDiskWAL_ContextCancelDuringCloseRace(t *testing.T) {
	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())

	errChan := make(chan error, 1)

	go func() {
		entry := wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       "ctx-cancel",
			Value:     testValue{Name: "cancel", Count: 1},
			Timestamp: time.Now().UnixNano(),
		}
		errChan <- wal.Append(ctx, entry)
	}()

	go cancel(fmt.Errorf("test: simulating cancelled context"))
	go func() { _ = wal.Close() }()

	select {
	case err := <-errChan:
		if err != nil {
			assert.True(t,
				errors.Is(err, context.Canceled) || errors.Is(err, wal_domain.ErrWALClosed),
				"unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Append blocked forever!")
	}
}

func TestDiskWAL_WithMockSandbox_WriteError(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	writeErr := errors.New("disk full")
	mockFile.WriteAtErr = writeErr

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	ctx := context.Background()
	err = wal.Append(ctx, entry)

	assert.Error(t, err)

	assert.Contains(t, err.Error(), "disk full")
}

func TestDiskWAL_WithMockSandbox_SyncError(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	syncErr := errors.New("sync failed")
	mockFile.SyncErr = syncErr

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	ctx := context.Background()
	err = wal.Append(ctx, entry)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fsync failed")
}

func TestDiskWAL_WithMockSandbox_StatError(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	statErr := errors.New("stat failed")
	mockFile.StatErr = statErr

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	_, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "getting WAL file info")
}

func TestDiskWAL_WithMockSandbox_MultipleAppends(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()

	for i := range 10 {
		entry := wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       fmt.Sprintf("key%d", i),
			Value:     testValue{Name: fmt.Sprintf("value%d", i), Count: i},
			Timestamp: time.Now().UnixNano(),
		}
		err := wal.Append(ctx, entry)
		assert.NoError(t, err)
	}

	assert.EqualValues(t, 10, wal.EntryCount())

	assert.Greater(t, len(mockFile.Data()), 0)
}

func TestDiskWAL_WithMockSandbox_TruncateError(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	truncateErr := errors.New("truncate failed")
	mockFile.TruncateErr = truncateErr

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	err = wal.Truncate(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "truncate failed")
}

func TestDiskWAL_WithMockSandbox_CloseError(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	closeErr := errors.New("close failed")
	mockFile.CloseErr = closeErr

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)

	ctx := context.Background()
	err = wal.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)

	err = wal.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing WAL file")
}

func TestDiskWAL_ConcurrentAppend_HighContention_DataIntegrity(t *testing.T) {
	t.Parallel()

	wal, _ := newTestWAL(t)
	ctx := context.Background()

	const numGoroutines = 50
	const entriesPerGoroutine = 100

	type expectedEntry struct {
		key   string
		name  string
		count int
	}

	allExpected := make([]expectedEntry, 0, numGoroutines*entriesPerGoroutine)
	var mu sync.Mutex

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*entriesPerGoroutine)

	for g := range numGoroutines {
		wg.Go(func() {
			for i := range entriesPerGoroutine {
				key := fmt.Sprintf("g%d-e%d", g, i)
				name := fmt.Sprintf("name-%d-%d", g, i)
				count := g*1000 + i

				entry := wal_domain.Entry[string, testValue]{
					Operation: wal_domain.OpSet,
					Key:       key,
					Value:     testValue{Name: name, Count: count},
					Timestamp: time.Now().UnixNano(),
				}
				if err := wal.Append(ctx, entry); err != nil {
					errChan <- err
					return
				}

				mu.Lock()
				allExpected = append(allExpected, expectedEntry{key: key, name: name, count: count})
				mu.Unlock()
			}
		})
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("concurrent append error: %v", err)
	}

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	require.Len(t, recovered, numGoroutines*entriesPerGoroutine)

	recoveredMap := make(map[string]testValue, len(recovered))
	for _, e := range recovered {
		recoveredMap[e.Key] = e.Value
	}

	for _, exp := range allExpected {
		value, ok := recoveredMap[exp.key]
		if !ok {
			t.Errorf("missing key %q in recovered entries", exp.key)
			continue
		}
		assert.Equal(t, exp.name, value.Name, "key %q", exp.key)
		assert.Equal(t, exp.count, value.Count, "key %q", exp.key)
	}
}

func TestDiskWAL_Append_ContextCancelBeforeEnqueue(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	codec := newTestCodec()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	countBefore := wal.EntryCount()

	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "cancelled",
		Value:     testValue{Name: "should-not-persist", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	err = wal.Append(ctx, entry)
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, countBefore, wal.EntryCount())
	}

	require.NoError(t, wal.Append(context.Background(), wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "after-cancel",
		Value:     testValue{Name: "ok", Count: 2},
		Timestamp: time.Now().UnixNano(),
	}))

	recovered, recErr := collectEntries(wal.Recover(context.Background()))
	require.NoError(t, recErr)

	var found bool
	for _, e := range recovered {
		if e.Key == "after-cancel" {
			assert.Equal(t, "ok", e.Value.Name)
			found = true
		}
	}
	assert.True(t, found, "entry appended after cancellation should be recoverable")
}

func TestNotifyBatchWaiters_ReleasesAllBuffers(t *testing.T) {
	t.Parallel()

	const batchSize = 5
	batch := make([]pendingWrite, batchSize)
	channels := make([]chan error, batchSize)
	var released atomic.Int32

	for i := range batchSize {
		resultChannel := make(chan error, 1)
		channels[i] = resultChannel

		ptr, buffer := GetByteBuffer(64)
		buffer = append(buffer, []byte("test-data")...)

		batch[i] = pendingWrite{
			result: resultChannel,
			data:   buffer,
			encoded: wal_domain.NewEncodedEntry(buffer, ptr, func(handle *[]byte, data []byte) {
				released.Add(1)
				PutByteBuffer(handle, data)
			}),
		}
	}

	testErr := errors.New("batch write error")
	notifyBatchWaiters(batch, testErr)
	assert.Equal(t, int32(batchSize), released.Load(), "every buffer should go back to its pool")

	for i, resultChannel := range channels {
		select {
		case err := <-resultChannel:
			assert.Equal(t, testErr, err, "waiter %d should receive the error", i)
		default:
			t.Errorf("waiter %d was not notified", i)
		}
	}
}

func TestDiskWAL_AlignedWriteSize(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	err = wal.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(directory, "data.wal"))
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "file should not be empty")
	assert.Equal(t, int64(0), info.Size()%4096,
		"physical file size should be 4KB-aligned during operation, got %d", info.Size())

	require.NoError(t, wal.Close())

	info, err = os.Stat(filepath.Join(directory, "data.wal"))
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "file should not be empty after close")
	assert.Less(t, info.Size(), int64(4096),
		"file should be smaller than 4096 after close strips padding")
}

func TestDiskWAL_AlignedWriteAfterReopen(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	for i := range 2 {
		err := wal1.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       fmt.Sprintf("key%d", i),
			Value:     testValue{Name: fmt.Sprintf("value%d", i), Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}
	require.NoError(t, wal1.Close())

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	for i := 2; i < 4; i++ {
		err := wal2.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       fmt.Sprintf("key%d", i),
			Value:     testValue{Name: fmt.Sprintf("value%d", i), Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	entries, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)
	assert.Len(t, entries, 4)

	for i, entry := range entries {
		assert.Equal(t, fmt.Sprintf("key%d", i), entry.Key)
	}

	require.NoError(t, wal2.Close())
}

func TestDiskWAL_AlignedWriteMultipleBatches(t *testing.T) {
	t.Parallel()

	wal, _ := newTestWAL(t)
	ctx := context.Background()

	const numEntries = 100
	for i := range numEntries {
		err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet,
			Key:       fmt.Sprintf("key-%03d", i),
			Value:     testValue{Name: fmt.Sprintf("value-%03d", i), Count: i},
			Timestamp: time.Now().UnixNano(),
		})
		require.NoError(t, err)
	}

	entries, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	assert.Len(t, entries, numEntries)
}

func TestDiskWAL_AlignedWriteRecoverWithPadding(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	codec := newTestCodec()
	ctx := context.Background()

	config := wal_domain.Config{
		Dir:      directory,
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	wal1, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	err = wal1.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "padded",
		Value:     testValue{Name: "value", Count: 42},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)

	require.NoError(t, wal1.Sync(ctx))

	data, err := os.ReadFile(filepath.Join(directory, "data.wal"))
	require.NoError(t, err)
	assert.Equal(t, 0, len(data)%4096, "file should be 4KB-aligned during operation")

	hasTrailingZeros := false
	for _, byteValue := range slices.Backward(data) {
		if byteValue == 0 {
			hasTrailingZeros = true
		} else {
			break
		}
	}
	assert.True(t, hasTrailingZeros, "file should have trailing zero padding")

	require.NoError(t, wal1.Close())

	f, err := os.OpenFile(filepath.Join(directory, "data.wal"), os.O_RDWR, 0o600)
	require.NoError(t, err)
	logicalSize, err := f.Seek(0, io.SeekEnd)
	require.NoError(t, err)
	pad := (4096 - int(logicalSize)%4096) % 4096
	if pad > 0 {
		zeros := make([]byte, pad)
		_, err = f.Write(zeros)
		require.NoError(t, err)
	}
	require.NoError(t, f.Close())

	wal2, err := NewDiskWAL(context.Background(), config, codec)
	require.NoError(t, err)

	entries, err := collectEntries(wal2.Recover(ctx))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "padded", entries[0].Key)

	require.NoError(t, wal2.Close())
}

func TestDiskWAL_TruncateResetsAlignment(t *testing.T) {
	t.Parallel()

	wal, _ := newTestWAL(t)
	ctx := context.Background()

	err := wal.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "before-truncate",
		Value:     testValue{Name: "old", Count: 1},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)

	require.NoError(t, wal.Truncate(ctx))

	err = wal.Append(ctx, wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "after-truncate",
		Value:     testValue{Name: "new", Count: 2},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)

	entries, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "after-truncate", entries[0].Key)
}

func TestDiskWAL_FatalError_WriteAtFailure(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	mockFile.WriteAtErr = errors.New("disk full")

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	err = wal.Append(ctx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")

	err = wal.Append(ctx, entry)
	assert.ErrorIs(t, err, wal_domain.ErrWriterInBadState)
}

func TestDiskWAL_FatalError_SyncFailure(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	mockFile.SyncErr = errors.New("sync failed")

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeEveryWrite,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	err = wal.Append(ctx, entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sync failed")

	err = wal.Append(ctx, entry)
	assert.ErrorIs(t, err, wal_domain.ErrWriterInBadState)
}

func TestDiskWAL_FatalError_DoesNotBlock(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	mockFile.WriteAtErr = errors.New("disk error")

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_ = wal.Append(ctx, entry)

	const numWriters = 50
	var wg sync.WaitGroup

	for range numWriters {
		wg.Go(func() {
			_ = wal.Append(ctx, entry)
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(5 * time.Second):
		t.Fatal("writers blocked after fatal error - possible deadlock")
	}
}

func TestDiskWAL_FatalError_CloseStillWorks(t *testing.T) {
	t.Parallel()

	mockSandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
	mockFile := safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)

	mockFile.WriteAtErr = errors.New("disk error")

	config := wal_domain.Config{
		Dir:         "/test",
		WALFileName: "data.wal",
		SyncMode:    wal_domain.SyncModeNone,
	}

	wal, err := NewDiskWAL(
		context.Background(),
		config,
		newTestCodec(),
		WithSandbox[string, testValue](mockSandbox, mockFile),
	)
	require.NoError(t, err)

	ctx := context.Background()
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "key1",
		Value:     testValue{Name: "value1", Count: 1},
		Timestamp: time.Now().UnixNano(),
	}

	_ = wal.Append(ctx, entry)

	err = wal.Close()
	assert.NoError(t, err)
}

func TestDiskWAL_RecoveryAndAppendReportTheCancellationCause(t *testing.T) {
	cause := errors.New("shutdown requested")
	cancelled, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	entry := wal_domain.Entry[string, testValue]{}
	entry.Operation = wal_domain.OpSet
	entry.Key = "key1"

	testCases := []struct {
		run  func(t *testing.T) error
		name string
	}{
		{
			name: "recovering the log",
			run: func(t *testing.T) error {
				wal, _ := newTestWAL(t)
				require.NoError(t, wal.Append(context.Background(), entry))
				_, err := collectEntries(wal.Recover(cancelled))
				return err
			},
		},
		{
			name: "appending to the log",
			run: func(t *testing.T) error {
				wal, _ := newTestWAL(t)
				return wal.Append(cancelled, entry)
			},
		},
		{
			name: "loading a snapshot",
			run: func(t *testing.T) error {
				snapshot, _ := newTestSnapshot(t, false)
				require.NoError(t, snapshot.Save(context.Background(), []wal_domain.Entry[string, testValue]{entry}))
				_, err := collectEntries(snapshot.Load(cancelled))
				return err
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.run(t)

			assert.ErrorIs(t, err, cause)
			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestDiskWAL_EnqueueEncodedRoundTrip(t *testing.T) {
	t.Parallel()

	wal, _ := newTestWAL(t)
	ctx := context.Background()

	encoded, err := wal.Encode(wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "encoded",
		Value:     testValue{Name: "ahead", Count: 3},
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)
	wait, err := wal.EnqueueEncoded(ctx, encoded)
	require.NoError(t, err)
	require.NoError(t, wait())

	recovered, err := collectEntries(wal.Recover(ctx))
	require.NoError(t, err)
	require.Len(t, recovered, 1)
	assert.Equal(t, "encoded", recovered[0].Key)
	assert.Equal(t, "ahead", recovered[0].Value.Name)
}

func TestDiskWAL_EnqueueEncodedOnClosedWALReleasesTheEntry(t *testing.T) {
	t.Parallel()

	wal, _ := newTestWAL(t)
	encoded, err := wal.Encode(wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "late",
		Timestamp: time.Now().UnixNano(),
	})
	require.NoError(t, err)
	require.NoError(t, wal.Close())

	_, err = wal.EnqueueEncoded(context.Background(), encoded)

	require.ErrorIs(t, err, wal_domain.ErrWALClosed)
}

func TestDiskWAL_EnqueueFailsWhenTheQueueStaysFull(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	stalled := &DiskWAL[string, testValue]{
		clock:       mockClock,
		codec:       newTestCodec(),
		pendingChan: make(chan pendingWrite),
		stopChan:    make(chan struct{}),
		config:      wal_domain.Config{QueueTimeout: time.Second},
	}
	entry := wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet,
		Key:       "stalled",
		Timestamp: mockClock.Now().UnixNano(),
	}

	baseline := mockClock.TimerCount()
	result := make(chan error, 1)
	go func() {
		_, err := stalled.Enqueue(context.Background(), entry)
		result <- err
	}()
	require.True(t, mockClock.AwaitTimerSetup(baseline, 5*time.Second), "the write waits on the queue timer")
	mockClock.Advance(time.Second)

	select {
	case err := <-result:
		require.ErrorIs(t, err, wal_domain.ErrQueueStalled)
	case <-time.After(5 * time.Second):
		t.Fatal("a write to a stalled queue must fail once the queue timeout passes")
	}
}

type syncCountingFile struct {
	*safedisk.MockFileHandle
	syncs atomic.Int32
}

func (f *syncCountingFile) Sync() error {
	f.syncs.Add(1)
	return f.MockFileHandle.Sync()
}

func newIntervalSyncWAL(t *testing.T, mockClock *clock.MockClock) (*DiskWAL[string, testValue], *syncCountingFile) {
	t.Helper()
	file := &syncCountingFile{MockFileHandle: safedisk.NewMockFileHandle("data.wal", "/test/data.wal", nil)}
	config := wal_domain.Config{
		Dir:               "/test",
		WALFileName:       "data.wal",
		SyncMode:          wal_domain.SyncModeBatched,
		BatchSyncCount:    1000,
		BatchSyncInterval: 100 * time.Millisecond,
	}
	wal, err := NewDiskWAL(context.Background(), config, newTestCodec(),
		WithSandbox[string, testValue](safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite), file),
		WithWALClock[string, testValue](mockClock))
	require.NoError(t, err)
	t.Cleanup(func() { _ = wal.Close() })
	return wal, file
}

func TestDiskWAL_BatchedModeSyncsAQuietLogWithinTheInterval(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	wal, file := newIntervalSyncWAL(t, mockClock)

	require.NoError(t, wal.Append(context.Background(), wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet, Key: "quiet", Value: testValue{Name: "v"}, Timestamp: 1,
	}))
	assert.Equal(t, int32(0), file.syncs.Load(), "one write is below the batch count, so it is not synced at once")

	require.Eventually(t, func() bool {
		mockClock.Advance(50 * time.Millisecond)
		return file.syncs.Load() == 1
	}, 5*time.Second, time.Millisecond, "a written entry must be synced once the batch interval passes")

	for range 5 {
		mockClock.Advance(time.Second)
	}
	assert.Equal(t, int32(1), file.syncs.Load(), "nothing waits for a sync, so no further sync runs")
}

func TestDiskWAL_BatchedModeSyncsInlineOnceTheIntervalHasPassed(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	wal, file := newIntervalSyncWAL(t, mockClock)
	mockClock.Advance(time.Second)

	require.NoError(t, wal.Append(context.Background(), wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet, Key: "late", Value: testValue{Name: "v"}, Timestamp: 1,
	}))

	assert.Equal(t, int32(1), file.syncs.Load(), "a batch written after the interval is synced before its writers are told")
}

func TestDiskWAL_IntervalSyncFailureIsFatal(t *testing.T) {
	t.Parallel()

	mockClock := clock.NewMockClock(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	wal, file := newIntervalSyncWAL(t, mockClock)
	file.SyncErr = errors.New("sync failed")

	require.NoError(t, wal.Append(context.Background(), wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet, Key: "doomed", Value: testValue{Name: "v"}, Timestamp: 1,
	}))
	require.Eventually(t, func() bool {
		mockClock.Advance(50 * time.Millisecond)
		return wal.fatalErr.Load() != nil
	}, 5*time.Second, time.Millisecond)

	err := wal.Append(context.Background(), wal_domain.Entry[string, testValue]{
		Operation: wal_domain.OpSet, Key: "after", Value: testValue{Name: "v"}, Timestamp: 2,
	})
	require.ErrorIs(t, err, wal_domain.ErrWriterInBadState)
}

func TestDiskWAL_ReadsBackEveryEntryAfterTheLengthOnlyEndSignal(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	config := wal_domain.Config{Dir: directory, SyncMode: wal_domain.SyncModeNone}
	ctx := context.Background()
	wal, err := NewDiskWAL(ctx, config, newSlowTestCodec())
	require.NoError(t, err)
	for index := range 50 {
		require.NoError(t, wal.Append(ctx, wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet, Key: fmt.Sprintf("key-%d", index), Value: testValue{Name: "v", Count: index}, Timestamp: int64(index),
		}))
	}
	require.NoError(t, wal.Close())

	reopened, err := NewDiskWAL(ctx, config, newSlowTestCodec())
	require.NoError(t, err)
	defer func() { _ = reopened.Close() }()
	entries, err := collectEntries(reopened.Recover(ctx))
	require.NoError(t, err)
	require.Len(t, entries, 50)
	for index, entry := range entries {
		assert.Equal(t, fmt.Sprintf("key-%d", index), entry.Key, "recovery keeps the order entries were written in")
		assert.Equal(t, index, entry.Value.Count)
	}
}

func TestDiskWAL_QueueEncodedKeepsQueueOrderAndWritesOnSignal(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	ctx := context.Background()
	config := wal_domain.Config{Dir: directory, SyncMode: wal_domain.SyncModeNone}
	wal, err := NewDiskWAL(ctx, config, newTestCodec())
	require.NoError(t, err)

	waits := make([]func() error, 0, 3)
	for index := range 3 {
		encoded, err := wal.Encode(wal_domain.Entry[string, testValue]{
			Operation: wal_domain.OpSet, Key: fmt.Sprintf("quiet-%d", index), Value: testValue{Name: "v", Count: index}, Timestamp: int64(index),
		})
		require.NoError(t, err)
		wait, err := wal.QueueEncoded(ctx, encoded)
		require.NoError(t, err)
		waits = append(waits, wait)
	}
	wal.Signal()
	require.Eventually(t, func() bool { return wal.EntryCount() == 3 }, 5*time.Second, time.Millisecond,
		"a signal writes every entry queued before it")
	for _, wait := range waits {
		require.NoError(t, wait())
	}

	encoded, err := wal.Encode(wal_domain.Entry[string, testValue]{Operation: wal_domain.OpSet, Key: "waited", Timestamp: 3})
	require.NoError(t, err)
	wait, err := wal.QueueEncoded(ctx, encoded)
	require.NoError(t, err)
	require.NoError(t, wait(), "the wait of a quietly queued entry signals before it waits")
	require.NoError(t, wal.Close())

	reopened, err := NewDiskWAL(ctx, config, newTestCodec())
	require.NoError(t, err)
	defer func() { _ = reopened.Close() }()
	entries, err := collectEntries(reopened.Recover(ctx))
	require.NoError(t, err)
	keys := make([]string, 0, len(entries))
	for _, entry := range entries {
		keys = append(keys, entry.Key)
	}
	assert.Equal(t, []string{"quiet-0", "quiet-1", "quiet-2", "waited"}, keys, "entries reach the file in the order they were queued")
}

func TestDiskWAL_QuietEntryIsWrittenByTheNextSignalledWrite(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wal, err := NewDiskWAL(ctx, wal_domain.Config{Dir: t.TempDir(), SyncMode: wal_domain.SyncModeNone}, newTestCodec())
	require.NoError(t, err)
	defer func() { _ = wal.Close() }()

	encoded, err := wal.Encode(wal_domain.Entry[string, testValue]{Operation: wal_domain.OpSet, Key: "quiet", Timestamp: 1})
	require.NoError(t, err)
	_, err = wal.QueueEncoded(ctx, encoded)
	require.NoError(t, err)

	require.NoError(t, wal.Append(ctx, wal_domain.Entry[string, testValue]{Operation: wal_domain.OpSet, Key: "loud", Timestamp: 2}))
	assert.Equal(t, 2, wal.EntryCount(), "a write that signals also writes the quiet entry queued before it")
}

func TestDiskWAL_QueueEncodedRefusesAClosedLog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wal, err := NewDiskWAL(ctx, wal_domain.Config{Dir: t.TempDir(), SyncMode: wal_domain.SyncModeNone}, newTestCodec())
	require.NoError(t, err)
	encoded, err := wal.Encode(wal_domain.Entry[string, testValue]{Operation: wal_domain.OpSet, Key: "late", Timestamp: 1})
	require.NoError(t, err)
	require.NoError(t, wal.Close())

	_, err = wal.QueueEncoded(ctx, encoded)

	require.ErrorIs(t, err, wal_domain.ErrWALClosed)
	assert.NotPanics(t, wal.Signal, "signalling a closed log does nothing")
}
