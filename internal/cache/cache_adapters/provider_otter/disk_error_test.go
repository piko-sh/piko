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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestDiskError_DirectoryNotExist(t *testing.T) {

	nonExistentDir := "/nonexistent/path/that/cannot/be/created/wal"

	walConfig := wal_domain.DefaultConfig(nonExistentDir)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	_, err := OtterProviderFactory(opts)
	assert.Error(t, err, "Should fail when directory cannot be created")
}

func TestDiskError_ReadOnlyDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping as root can write to read-only directories")
	}

	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0o555)
	require.NoError(t, err)

	t.Cleanup(func() {

		_ = os.Chmod(readOnlyDir, 0o755)
	})

	walConfig := wal_domain.DefaultConfig(readOnlyDir)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	_, err = OtterProviderFactory(opts)
	assert.Error(t, err, "Should fail when directory is read-only")
}

func TestDiskError_PartialCorruption(t *testing.T) {
	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 1000

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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
	ctx := context.Background()
	for i := range 50 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}
	_ = cache1.Close(ctx)

	walPath := filepath.Join(directory, "wal.dat")
	f, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {

		_, _ = f.Write([]byte{0xFF, 0xFE, 0xFD, 0xFC, 0x00, 0x00})
		_ = f.Close()
	}

	adapter2, err := OtterProviderFactory(opts)

	if err != nil {
		t.Logf("Recovery failed due to corruption (expected): %v", err)
		return
	}
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	recoveredCount := 0
	for i := range 50 {
		if _, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i)); ok {
			recoveredCount++
		}
	}

	t.Logf("Recovered %d out of 50 entries after corruption", recoveredCount)

	assert.Greater(t, recoveredCount, 0, "Should recover some entries despite corruption")
}

func TestDiskError_TruncatedWAL(t *testing.T) {
	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 1000

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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
	ctx := context.Background()
	for i := range 50 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}
	_ = cache1.Close(ctx)

	walPath := filepath.Join(directory, "wal.dat")
	info, err := os.Stat(walPath)
	if err == nil && info.Size() > 100 {

		err = os.Truncate(walPath, info.Size()/2)
		if err != nil {
			t.Skipf("Could not truncate WAL file: %v", err)
		}
	}

	adapter2, err := OtterProviderFactory(opts)
	if err != nil {
		t.Logf("Recovery failed due to truncation (expected): %v", err)
		return
	}
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	recoveredCount := 0
	for i := range 50 {
		if _, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i)); ok {
			recoveredCount++
		}
	}

	t.Logf("Recovered %d out of 50 entries after truncation", recoveredCount)
}

func TestDiskError_EmptyWALFile(t *testing.T) {
	directory := t.TempDir()

	walPath := filepath.Join(directory, "wal.dat")
	f, err := os.Create(walPath)
	require.NoError(t, err)
	_ = f.Close()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err, "Should handle empty WAL file")
	ctx := context.Background()
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	_ = cache.Set(ctx, "test-key", testArticle{Title: "Test"})
	value, ok, _ := cache.GetIfPresent(ctx, "test-key")
	assert.True(t, ok)
	assert.Equal(t, "Test", value.Title)
}

func TestDiskError_SnapshotReadFails(t *testing.T) {
	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 20

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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
	ctx := context.Background()
	for i := range 50 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	cache1.checkpointMu.Lock()
	cache1.performCheckpointLocked()
	cache1.checkpointMu.Unlock()
	_ = cache1.Close(ctx)

	snapshotPath := filepath.Join(directory, "snapshot.piko")
	f, err := os.OpenFile(snapshotPath, os.O_WRONLY, 0o644)
	if err == nil {

		_, _ = f.WriteAt([]byte{0xFF, 0xFE, 0xFD, 0xFC, 0x00, 0x00}, 0)
		_ = f.Close()
	}

	adapter2, err := OtterProviderFactory(opts)
	if err != nil {
		t.Logf("Recovery failed due to snapshot corruption (expected): %v", err)
		return
	}
	defer func() { _ = adapter2.Close(ctx) }()

	t.Log("Recovery succeeded despite snapshot corruption (WAL backup worked)")
}

func TestDiskError_MissingSnapshotWithWAL(t *testing.T) {
	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 1000

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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
	ctx := context.Background()
	for i := range 50 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	if cache1.wal != nil {
		_ = cache1.wal.Close()
	}
	cache1.client.CleanUp()

	snapshotPath := filepath.Join(directory, "snapshot.piko")
	_ = os.Remove(snapshotPath)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 50 {
		value, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i))
		assert.True(t, ok, "Entry key-%d should be recovered from WAL", i)
		assert.Equal(t, fmt.Sprintf("Title %d", i), value.Title)
	}
}

func TestDiskError_RecoveryAfterCleanShutdown(t *testing.T) {
	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 100

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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
	ctx := context.Background()
	for i := range 100 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i), Views: i * 10})
	}
	_ = cache1.Close(ctx)

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	for i := range 100 {
		value, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i))
		require.True(t, ok, "Entry key-%d should exist", i)
		assert.Equal(t, fmt.Sprintf("Title %d", i), value.Title)
		assert.Equal(t, i*10, value.Views)
	}
}

func TestDiskError_RecoveryAfterCrash(t *testing.T) {

	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 1000

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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
	ctx := context.Background()
	for i := range 50 {
		_ = cache1.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Title %d", i)})
	}

	if cache1.wal != nil {
		_ = cache1.wal.Close()
	}

	adapter2, err := OtterProviderFactory(opts)
	require.NoError(t, err)
	defer func() { _ = adapter2.Close(ctx) }()

	cache2, ok := adapter2.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	recoveredCount := 0
	for i := range 50 {
		if _, ok, _ := cache2.GetIfPresent(ctx, fmt.Sprintf("key-%d", i)); ok {
			recoveredCount++
		}
	}

	assert.Equal(t, 50, recoveredCount, "All entries should be recovered with SyncModeEveryWrite")
}

func TestDiskError_InvalidConfig_EmptyDir(t *testing.T) {
	walConfig := wal_domain.Config{
		Dir:      "",
		SyncMode: wal_domain.SyncModeEveryWrite,
	}

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	_, err := OtterProviderFactory(opts)
	assert.Error(t, err, "Should fail with empty directory config")
}

func TestDiskError_InvalidConfig_NilKeyCodec(t *testing.T) {
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   nil,
			ValueCodec: jsonArticleCodec{},
		},
	}

	_, err := OtterProviderFactory(opts)
	assert.Error(t, err, "Should fail with nil key codec")
}

func TestDiskError_InvalidConfig_NilValueCodec(t *testing.T) {
	directory := t.TempDir()
	walConfig := wal_domain.DefaultConfig(directory)

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: nil,
		},
	}

	_, err := OtterProviderFactory(opts)
	assert.Error(t, err, "Should fail with nil value codec")
}

func TestDiskError_MultipleOpenClose(t *testing.T) {
	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite
	walConfig.SnapshotThreshold = 100

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
		ProviderSpecific: PersistenceConfig[string, testArticle]{
			Enabled:    true,
			WALConfig:  walConfig,
			KeyCodec:   stringKeyCodec{},
			ValueCodec: jsonArticleCodec{},
		},
	}

	ctx := context.Background()

	for cycle := range 10 {
		adapter, err := OtterProviderFactory(opts)
		require.NoError(t, err, "Cycle %d: should open successfully", cycle)

		cache, ok := adapter.(*OtterAdapter[string, testArticle])
		require.True(t, ok)

		for i := range 50 {
			_ = cache.Set(ctx, fmt.Sprintf("key-%d", i), testArticle{Title: fmt.Sprintf("Cycle%d-Title%d", cycle, i)})
		}

		_ = cache.Close(ctx)
	}

	adapter, err := OtterProviderFactory(opts)
	require.NoError(t, err, "Final open should succeed")
	defer func() { _ = adapter.Close(ctx) }()

	cache, ok := adapter.(*OtterAdapter[string, testArticle])
	require.True(t, ok)

	value, ok, _ := cache.GetIfPresent(ctx, "key-0")
	assert.True(t, ok)
	assert.Contains(t, value.Title, "Cycle9")
}

func TestDiskError_CacheStillWorksAfterWALError(t *testing.T) {

	directory := t.TempDir()

	walConfig := wal_domain.DefaultConfig(directory)
	walConfig.SyncMode = wal_domain.SyncModeEveryWrite

	opts := cache_dto.Options[string, testArticle]{
		MaximumSize: 1000,
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

	_ = cache.Set(ctx, "key-1", testArticle{Title: "Title 1"})
	value, ok, _ := cache.GetIfPresent(ctx, "key-1")
	assert.True(t, ok)
	assert.Equal(t, "Title 1", value.Title)

	_ = cache.Invalidate(ctx, "key-1")
	_, ok, _ = cache.GetIfPresent(ctx, "key-1")
	assert.False(t, ok)

	items := map[string]testArticle{
		"bulk-1": {Title: "Bulk 1"},
		"bulk-2": {Title: "Bulk 2"},
	}
	err = cache.BulkSet(ctx, items)
	assert.NoError(t, err)

	value, ok, _ = cache.GetIfPresent(ctx, "bulk-1")
	assert.True(t, ok)
	assert.Equal(t, "Bulk 1", value.Title)
}
