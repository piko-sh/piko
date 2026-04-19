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
	"os"
	"path/filepath"
	"testing"
	"time"

	"testing/fstest"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/wal/wal_adapters/driven_disk"
	"piko.sh/piko/internal/wal/wal_domain"
)

func TestLoadRegistryCacheFromFS_MissingSnapshot(t *testing.T) {
	fsys := fstest.MapFS{}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	_, found, err := cache.GetIfPresent(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected empty cache")
	}
}

func TestLoadRegistryCacheFromFS_WithSnapshot(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{
			Data: snapshotData,
		},
	}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, found, err := cache.GetIfPresent(context.Background(), "art-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected to find entry 'art-1'")
	}
	if value.SourcePath != "/pages/index.pk" {
		t.Errorf("SourcePath = %q, want %q", value.SourcePath, "/pages/index.pk")
	}

	value2, found2, err := cache.GetIfPresent(context.Background(), "art-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found2 {
		t.Fatal("expected to find entry 'art-2'")
	}
	if value2.SourcePath != "/pages/about.pk" {
		t.Errorf("SourcePath = %q, want %q", value2.SourcePath, "/pages/about.pk")
	}
}

func TestLoadRegistryCacheFromFS_CompressedSnapshot(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, true)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{
			Data: snapshotData,
		},
	}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, found, err := cache.GetIfPresent(context.Background(), "art-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected to find entry 'art-1' in compressed snapshot")
	}
	if value.SourcePath != "/pages/index.pk" {
		t.Errorf("SourcePath = %q, want %q", value.SourcePath, "/pages/index.pk")
	}
}

func TestLoadRegistryCacheFromFS_DefaultCapacity(t *testing.T) {
	fsys := fstest.MapFS{}

	cache, err := LoadRegistryCacheFromFS(context.Background(), fsys, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache with default capacity")
	}
}

func TestLoadOrchestratorCacheFromFS_MissingSnapshot(t *testing.T) {
	fsys := fstest.MapFS{}

	cache, err := LoadOrchestratorCacheFromFS(context.Background(), fsys, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
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
	if err != nil {
		t.Fatalf("creating snapshot: %v", err)
	}
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

	if err := snapshot.Save(context.Background(), entries); err != nil {
		t.Fatalf("saving snapshot: %v", err)
	}

	snapshotPath := snapshot.Path()
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("reading snapshot file: %v", err)
	}

	return data
}

func TestLoadRegistryCacheFromFS_CorruptMagic(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)
	snapshotData[0] = 0xFF

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: snapshotData},
	}

	_, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	if err == nil {
		t.Fatal("expected error for corrupt magic bytes")
	}
}

func TestLoadRegistryCacheFromFS_TruncatedHeader(t *testing.T) {
	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: []byte("short")},
	}

	_, err := LoadRegistryCacheFromFS(context.Background(), fsys, 1000)
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestLoadRegistryCacheFromFS_CancelledContext(t *testing.T) {
	snapshotData := createTestRegistrySnapshot(t, false)

	fsys := fstest.MapFS{
		registrySnapshotPath: &fstest.MapFile{Data: snapshotData},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cache, err := LoadRegistryCacheFromFS(ctx, fsys, 1000)
	if err != nil {

		return
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
}
