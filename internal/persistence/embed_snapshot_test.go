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

package persistence

import (
	"context"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestLoadRegistryCacheFromFS_MissingSnapshot(t *testing.T) {
	fsys := fstest.MapFS{}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	require.NoError(t, err, "unexpected error")

	require.NotNil(t, cache, "expected non-nil cache")

	_, found, err := cache.GetIfPresent(context.Background(), "nonexistent")
	require.NoError(t, err, "unexpected error")
	assert.False(t, found, "expected empty cache")
}

func TestLoadRegistryCacheFromFS_WithSnapshot(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{
			Data: snapshotData,
		},
	}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	require.NoError(t, err, "unexpected error")

	value, found, err := cache.GetIfPresent(context.Background(), "art-1")
	require.NoError(t, err, "unexpected error")
	require.True(t, found, "expected to find entry 'art-1'")
	assert.Equal(t, "/pages/index.pk", value.SourcePath,
		"SourcePath = %q, want %q", value.SourcePath, "/pages/index.pk")

	value2, found2, err := cache.GetIfPresent(context.Background(), "art-2")
	require.NoError(t, err, "unexpected error")
	require.True(t, found2, "expected to find entry 'art-2'")
	assert.Equal(t, "/pages/about.pk", value2.SourcePath,
		"SourcePath = %q, want %q", value2.SourcePath, "/pages/about.pk")
}

func TestLoadRegistryCacheFromFS_CompressedSnapshot(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, true)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{
			Data: snapshotData,
		},
	}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	require.NoError(t, err, "unexpected error")

	value, found, err := cache.GetIfPresent(context.Background(), "art-1")
	require.NoError(t, err, "unexpected error")
	require.True(t, found, "expected to find entry 'art-1' in compressed snapshot")
	assert.Equal(t, "/pages/index.pk", value.SourcePath,
		"SourcePath = %q, want %q", value.SourcePath, "/pages/index.pk")
}

func TestLoadRegistryCacheFromFS_DefaultCapacity(t *testing.T) {
	fsys := fstest.MapFS{}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 0)
	require.NoError(t, err, "unexpected error")
	require.NotNil(t, cache, "expected non-nil cache with default capacity")
}

func TestLoadRegistryArtefactsFromFS(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)
	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: snapshotData},
	}

	arts, err := LoadRegistryArtefactsFromFS(context.Background(), fsys)
	require.NoError(t, err, "unexpected error")
	require.Len(t, arts, 2,
		"got %d artefacts, want 2 (the seed extractor must return all snapshot entries)", len(arts))
}

func TestLoadRegistryArtefactsFromFS_MissingReturnsEmpty(t *testing.T) {
	arts, err := LoadRegistryArtefactsFromFS(context.Background(), fstest.MapFS{})
	require.NoError(t, err, "unexpected error")
	assert.Empty(t, arts, "expected 0 artefacts for a missing snapshot, got %d", len(arts))
}

func TestLoadOrchestratorCacheFromFS_MissingSnapshot(t *testing.T) {
	fsys := fstest.MapFS{}

	cache, err := LoadOrchestratorCacheFromFS(context.Background(), fsys, 1000)
	require.NoError(t, err, "unexpected error")
	require.NotNil(t, cache, "expected non-nil cache")
}

func createTestRegistrySnapshot(t *testing.T, compressed bool) []byte {
	t.Helper()

	tempDir := t.TempDir()
	snapshotDir := filepath.Join(tempDir, "snapshot")

	codec := driven_disk.NewBinaryCodec[string, *registry_dto.ArtefactMeta](
		StringKeyCodec{},
		ArtefactMetaCodec{},
	)

	config := wal_domain.Config{
		Dir:               snapshotDir,
		EnableCompression: compressed,
	}

	snapshot, err := driven_disk.NewDiskSnapshot[string, *registry_dto.ArtefactMeta](
		context.Background(),
		config,
		codec,
	)
	require.NoError(t, err, "creating snapshot")
	defer func() { _ = snapshot.Close() }()

	now := time.Now()

	entries := []wal_domain.Entry[string, *registry_dto.ArtefactMeta]{
		{
			Key: "art-1",
			Value: &registry_dto.ArtefactMeta{
				ID:         "art-1",
				SourcePath: "/pages/index.pk",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			Operation: wal_domain.OpSet,
			Timestamp: now.UnixNano(),
		},
		{
			Key: "art-2",
			Value: &registry_dto.ArtefactMeta{
				ID:         "art-2",
				SourcePath: "/pages/about.pk",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			Operation: wal_domain.OpSet,
			Timestamp: now.UnixNano(),
		},
	}

	require.NoError(t, snapshot.Save(context.Background(), entries), "saving snapshot")

	snapshotPath := snapshot.Path()
	data, err := os.ReadFile(snapshotPath)
	require.NoError(t, err, "reading snapshot file")

	return data
}

func TestLoadRegistryCacheFromFS_CorruptMagic(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)
	snapshotData[0] = 0xFF

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: snapshotData},
	}

	_, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	require.Error(t, err, "expected error for corrupt magic bytes")
}

func TestLoadRegistryCacheFromFS_TruncatedHeader(t *testing.T) {
	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: []byte("short")},
	}

	_, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	require.Error(t, err, "expected error for truncated header")
}

func TestLoadRegistryCacheFromFS_CancelledContext(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: snapshotData},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := LoadRegistryCacheFromFS(ctx, fsys, 1000)
	require.Error(t, err,
		"expected an error when the context is cancelled before decoding a non-empty snapshot")
}

func inflateHeaderEntryCount(t *testing.T, snapshot []byte) {
	t.Helper()
	const entryCountOffset = 8
	const headerCRCOffset = 28
	declared := binary.BigEndian.Uint64(snapshot[entryCountOffset:])
	binary.BigEndian.PutUint64(snapshot[entryCountOffset:], declared+1)
	binary.BigEndian.PutUint32(snapshot[headerCRCOffset:], crc32.Checksum(snapshot[:headerCRCOffset], crcTable))
}

func TestLoadRegistryCacheFromFS_DeclaredEntryCountMismatch(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)
	inflateHeaderEntryCount(t, snapshotData)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: snapshotData},
	}

	_, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	require.Error(t, err,
		"expected an error when the header declares more entries than the stream holds")
}

func TestLoadOrchestratorCacheFromFS_WithSnapshot(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		snapshotData := createTestOrchestratorSnapshot(t, compressed)

		fsys := fstest.MapFS{
			orchestratorSnapshotPath: &fstest.MapFile{Data: snapshotData},
		}

		cache, err := LoadOrchestratorCacheFromFS(context.Background(), fsys, 1000)
		require.NoError(t, err, "compressed=%v: unexpected error", compressed)

		task, found, err := cache.GetIfPresent(context.Background(), "task-1")
		require.NoError(t, err, "compressed=%v: unexpected error", compressed)
		require.True(t, found, "compressed=%v: expected to find entry 'task-1'", compressed)
		assert.Equal(t, "send-email", task.Executor,
			"compressed=%v: Executor = %q, want %q", compressed, task.Executor, "send-email")
		assert.Equal(t, "wf-1", task.WorkflowID,
			"compressed=%v: WorkflowID = %q, want %q", compressed, task.WorkflowID, "wf-1")
	}
}

func createTestOrchestratorSnapshot(t *testing.T, compressed bool) []byte {
	t.Helper()

	snapshotDir := filepath.Join(t.TempDir(), "snapshot")

	codec := driven_disk.NewBinaryCodec[string, *orchestrator_domain.Task](
		StringKeyCodec{},
		TaskCodec{},
	)

	config := wal_domain.Config{
		Dir:               snapshotDir,
		EnableCompression: compressed,
	}

	snapshot, err := driven_disk.NewDiskSnapshot[string, *orchestrator_domain.Task](
		context.Background(),
		config,
		codec,
	)
	require.NoError(t, err, "creating snapshot")
	defer func() { _ = snapshot.Close() }()

	now := time.Now()

	entries := []wal_domain.Entry[string, *orchestrator_domain.Task]{
		{
			Key: "task-1",
			Value: &orchestrator_domain.Task{
				ID:         "task-1",
				WorkflowID: "wf-1",
				Executor:   "send-email",
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			Operation: wal_domain.OpSet,
			Timestamp: now.UnixNano(),
		},
	}

	require.NoError(t, snapshot.Save(context.Background(), entries), "saving snapshot")

	data, err := os.ReadFile(snapshot.Path())
	require.NoError(t, err, "reading snapshot file")

	return data
}
