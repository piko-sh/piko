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

package coordinator_adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewDiskFileHashCache(t *testing.T) {
	t.Parallel()

	t.Run("creates cache with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		require.NotNil(t, cache)
		diskCache, ok := cache.(*diskFileHashCache)
		require.True(t, ok, "expected *diskFileHashCache")
		assert.Equal(t, sandbox, diskCache.sandbox)
		assert.Equal(t, "hashes.json", diskCache.cacheFileName)
	})

	t.Run("creates cache with default sandbox when none injected", func(t *testing.T) {
		t.Parallel()

		cache := NewDiskFileHashCache("/tmp/test-file-hash-cache/hashes.json")

		require.NotNil(t, cache)
		diskCache, ok := cache.(*diskFileHashCache)
		require.True(t, ok, "expected *diskFileHashCache")
		assert.Equal(t, "hashes.json", diskCache.cacheFileName)
	})
}

func TestDiskFileHashCache_Load(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when sandbox is nil", func(t *testing.T) {
		t.Parallel()

		cache := &diskFileHashCache{
			cache:         make(map[string]cacheEntry),
			sandbox:       nil,
			cacheFileName: "hashes.json",
		}

		err := cache.Load(context.Background())

		require.NoError(t, err)
	})

	t.Run("returns nil when cache file does not exist (cold start)", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.NoError(t, err)
	})

	t.Run("loads entries from valid cache file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		modTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
		entries := map[string]cacheEntry{
			"/project/src/main.go": {
				ModTime: modTime,
				Hash:    "abc123def456",
			},
		}
		data, _ := json.ConfigStd.MarshalIndent(entries, "", "  ")
		require.NoError(t, sandbox.WriteFile("hashes.json", data, 0600))

		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.NoError(t, err)

		hash, found := cache.Get(context.Background(), "/project/src/main.go", modTime)
		assert.True(t, found)
		assert.Equal(t, "abc123def456", hash)
	})

	t.Run("returns error on corrupted JSON", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("hashes.json", []byte("not valid json{{{"), 0600))

		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing cache file JSON")
	})
}

func TestDiskFileHashCache_Load_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when ReadFile fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("hashes.json", []byte("{}"), 0600))
		sandbox.ReadFileErr = errors.New("disk read error")

		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading cache file")
	})
}

func TestDiskFileHashCache_Persist(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when sandbox is nil", func(t *testing.T) {
		t.Parallel()

		cache := &diskFileHashCache{
			cache:         make(map[string]cacheEntry),
			sandbox:       nil,
			cacheFileName: "hashes.json",
		}

		err := cache.Persist(context.Background())

		require.NoError(t, err)
	})

	t.Run("persists entries to disk", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		modTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
		cache.Set(context.Background(), "/project/src/main.go", modTime, "abc123")

		err := cache.Persist(context.Background())

		require.NoError(t, err)

		data, readErr := sandbox.ReadFile("hashes.json")
		require.NoError(t, readErr)

		var entries map[string]cacheEntry
		unmarshalErr := json.Unmarshal(data, &entries)
		require.NoError(t, unmarshalErr)
		assert.Len(t, entries, 1)
		assert.Equal(t, "abc123", entries["/project/src/main.go"].Hash)
	})
}

func TestDiskFileHashCache_Persist_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setupMock    func(*safedisk.MockSandbox)
		wantContains string
	}{
		{
			name: "MkdirAll error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.MkdirAllErr = errors.New("cannot create directory")
			},
			wantContains: "creating cache directory",
		},
		{
			name: "WriteFile error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.WriteFileErr = errors.New("disk full")
			},
			wantContains: "writing cache to temp file",
		},
		{
			name: "Rename error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.RenameErr = errors.New("rename failed")
			},
			wantContains: "renaming cache file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
			defer func() { _ = sandbox.Close() }()
			tc.setupMock(sandbox)

			cache := NewDiskFileHashCache(
				"/cache/hashes.json",
				WithCacheSandbox(sandbox),
			)

			modTime := time.Now()
			cache.Set(context.Background(), "/project/file.go", modTime, "hash123")

			err := cache.Persist(context.Background())

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantContains)
		})
	}
}

func TestDiskFileHashCache_GetSet(t *testing.T) {
	t.Parallel()

	t.Run("returns cache miss for unknown file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		hash, found := cache.Get(context.Background(), "/unknown/file.go", time.Now())

		assert.False(t, found)
		assert.Empty(t, hash)
	})

	t.Run("returns cache miss when mod time differs", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		cachedTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		newTime := time.Date(2026, 1, 15, 11, 0, 0, 0, time.UTC)

		cache.Set(context.Background(), "/project/file.go", cachedTime, "oldhash")
		hash, found := cache.Get(context.Background(), "/project/file.go", newTime)

		assert.False(t, found)
		assert.Empty(t, hash)
	})

	t.Run("returns cache hit when mod time matches", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		modTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		cache.Set(context.Background(), "/project/file.go", modTime, "myhash")

		hash, found := cache.Get(context.Background(), "/project/file.go", modTime)

		assert.True(t, found)
		assert.Equal(t, "myhash", hash)
	})

	t.Run("normalises file paths", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		cache := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		modTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		cache.Set(context.Background(), "/project/../project/file.go", modTime, "normalised")

		hash, found := cache.Get(context.Background(), "/project/file.go", modTime)

		assert.True(t, found)
		assert.Equal(t, "normalised", hash)
	})
}

func TestDiskFileHashCache_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("persisted data can be loaded back", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		cache1 := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)

		modTime1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
		modTime2 := time.Date(2026, 1, 16, 12, 0, 0, 0, time.UTC)
		cache1.Set(context.Background(), "/project/file1.go", modTime1, "hash1")
		cache1.Set(context.Background(), "/project/file2.go", modTime2, "hash2")

		require.NoError(t, cache1.Persist(context.Background()))

		cache2 := NewDiskFileHashCache(
			"/cache/hashes.json",
			WithCacheSandbox(sandbox),
		)
		require.NoError(t, cache2.Load(context.Background()))

		hash1, found1 := cache2.Get(context.Background(), "/project/file1.go", modTime1)
		assert.True(t, found1)
		assert.Equal(t, "hash1", hash1)

		hash2, found2 := cache2.Get(context.Background(), "/project/file2.go", modTime2)
		assert.True(t, found2)
		assert.Equal(t, "hash2", hash2)
	})
}
