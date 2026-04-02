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

package collection_adapters

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/wdk/safedisk"
)

type mockHybridRegistry struct {
	entries map[string]mockRegistryEntry
}

type mockRegistryEntry struct {
	blob   []byte
	etag   string
	config collection_dto.HybridConfig
}

func newMockHybridRegistry() *mockHybridRegistry {
	return &mockHybridRegistry{
		entries: make(map[string]mockRegistryEntry),
	}
}

func (m *mockHybridRegistry) Register(_ context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig) {
	key := providerName + ":" + collectionName
	m.entries[key] = mockRegistryEntry{
		blob:   blob,
		etag:   etag,
		config: config,
	}
}

func (m *mockHybridRegistry) GetBlob(_ context.Context, providerName, collectionName string) ([]byte, bool) {
	key := providerName + ":" + collectionName
	entry, ok := m.entries[key]
	if !ok {
		return nil, false
	}
	return entry.blob, false
}

func (m *mockHybridRegistry) GetETag(providerName, collectionName string) string {
	key := providerName + ":" + collectionName
	entry, ok := m.entries[key]
	if !ok {
		return ""
	}
	return entry.etag
}

func (m *mockHybridRegistry) List() []string {
	keys := make([]string, 0, len(m.entries))
	for k := range m.entries {
		keys = append(keys, k)
	}
	return keys
}

func TestNewDiskHybridCache(t *testing.T) {
	t.Parallel()

	t.Run("creates cache with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		require.NotNil(t, cache)
		assert.Equal(t, sandbox, cache.sandbox)
		assert.Equal(t, "hybrid.json", cache.cacheFileName)
	})

	t.Run("creates cache with default sandbox when none injected", func(t *testing.T) {
		t.Parallel()

		registry := newMockHybridRegistry()

		cache := newDiskHybridCache(
			"/tmp/test-hybrid-cache/hybrid.json",
			registry,
		)

		require.NotNil(t, cache)

		assert.Equal(t, "hybrid.json", cache.cacheFileName)
	})
}

func TestDiskHybridCache_Load(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when sandbox is nil", func(t *testing.T) {
		t.Parallel()

		registry := newMockHybridRegistry()

		cache := &diskHybridCache{
			sandbox:       nil,
			cacheFileName: "hybrid.json",
			registry:      registry,
		}

		err := cache.Load(context.Background())

		require.NoError(t, err)
	})

	t.Run("returns nil when cache file does not exist (cold start)", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.NoError(t, err)
		assert.Empty(t, registry.entries)
	})

	t.Run("loads entries from valid cache file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		blobData := []byte("test-blob-data")
		entries := []persistedHybridEntry{
			{
				ProviderName:   "test-provider",
				CollectionName: "test-collection",
				CurrentETag:    "etag-123",
				CurrentBlob:    base64.StdEncoding.EncodeToString(blobData),
				Config:         collection_dto.DefaultHybridConfig(),
			},
		}
		data, _ := json.Marshal(entries)
		require.NoError(t, sandbox.WriteFile("hybrid.json", data, 0600))

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.NoError(t, err)
		assert.Len(t, registry.entries, 1)
		assert.Equal(t, blobData, registry.entries["test-provider:test-collection"].blob)
		assert.Equal(t, "etag-123", registry.entries["test-provider:test-collection"].etag)
	})

	t.Run("handles corrupted cache file gracefully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		require.NoError(t, sandbox.WriteFile("hybrid.json", []byte("not valid json{{{"), 0600))

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.NoError(t, err)
		assert.Empty(t, registry.entries)
	})

	t.Run("handles invalid base64 blob gracefully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		entries := []persistedHybridEntry{
			{
				ProviderName:   "test-provider",
				CollectionName: "test-collection",
				CurrentETag:    "etag-123",
				CurrentBlob:    "not-valid-base64!!!",
				Config:         collection_dto.DefaultHybridConfig(),
			},
		}
		data, _ := json.Marshal(entries)
		require.NoError(t, sandbox.WriteFile("hybrid.json", data, 0600))

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.NoError(t, err)
		assert.Empty(t, registry.entries)
	})
}

func TestDiskHybridCache_Load_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when ReadFile fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		require.NoError(t, sandbox.WriteFile("hybrid.json", []byte("{}"), 0600))
		sandbox.ReadFileErr = errors.New("disk read error")

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Load(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading hybrid cache file")
	})
}

func TestDiskHybridCache_Persist(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when sandbox is nil", func(t *testing.T) {
		t.Parallel()

		registry := newMockHybridRegistry()

		cache := &diskHybridCache{
			sandbox:       nil,
			cacheFileName: "hybrid.json",
			registry:      registry,
		}

		err := cache.Persist(context.Background())

		require.NoError(t, err)
	})

	t.Run("returns nil when no entries to persist", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Persist(context.Background())

		require.NoError(t, err)

		_, statErr := sandbox.Stat("hybrid.json")
		assert.Error(t, statErr)
	})

	t.Run("persists entries to disk", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry := newMockHybridRegistry()

		registry.Register(context.Background(), "test-provider", "test-collection", []byte("blob-data"), "etag-456", collection_dto.DefaultHybridConfig())

		cache := newDiskHybridCache(
			"/cache/hybrid.json",
			registry,
			WithHybridCacheSandbox(sandbox),
		)

		err := cache.Persist(context.Background())

		require.NoError(t, err)

		data, readErr := sandbox.ReadFile("hybrid.json")
		require.NoError(t, readErr)

		var entries []persistedHybridEntry
		unmarshalErr := json.Unmarshal(data, &entries)
		require.NoError(t, unmarshalErr)
		assert.Len(t, entries, 1)
		assert.Equal(t, "test-provider", entries[0].ProviderName)
		assert.Equal(t, "test-collection", entries[0].CollectionName)
	})
}

func TestDiskHybridCache_Persist_Errors(t *testing.T) {
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
			wantContains: "creating hybrid cache directory",
		},
		{
			name: "WriteFile error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.WriteFileErr = errors.New("disk full")
			},
			wantContains: "writing hybrid cache to temp file",
		},
		{
			name: "Rename error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.RenameErr = errors.New("rename failed")
			},
			wantContains: "renaming hybrid cache file",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
			defer func() { _ = sandbox.Close() }()
			tc.setupMock(sandbox)
			registry := newMockHybridRegistry()

			registry.Register(context.Background(), "provider", "collection", []byte("data"), "etag", collection_dto.DefaultHybridConfig())

			cache := newDiskHybridCache(
				"/cache/hybrid.json",
				registry,
				WithHybridCacheSandbox(sandbox),
			)

			err := cache.Persist(context.Background())

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantContains)
		})
	}
}

func TestParseHybridKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		key                string
		wantProvider       string
		wantCollectionName string
	}{
		{
			name:               "valid key",
			key:                "provider:collection",
			wantProvider:       "provider",
			wantCollectionName: "collection",
		},
		{
			name:               "key with multiple colons",
			key:                "provider:collection:extra",
			wantProvider:       "provider",
			wantCollectionName: "collection:extra",
		},
		{
			name:               "key without colon",
			key:                "no-colon",
			wantProvider:       "",
			wantCollectionName: "",
		},
		{
			name:               "empty key",
			key:                "",
			wantProvider:       "",
			wantCollectionName: "",
		},
		{
			name:               "colon at start",
			key:                ":collection",
			wantProvider:       "",
			wantCollectionName: "collection",
		},
		{
			name:               "colon at end",
			key:                "provider:",
			wantProvider:       "provider",
			wantCollectionName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			provider, collectionName := parseHybridKey(tc.key)

			assert.Equal(t, tc.wantProvider, provider)
			assert.Equal(t, tc.wantCollectionName, collectionName)
		})
	}
}

func TestDiskHybridCache_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("persisted data can be loaded back", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		registry1 := newMockHybridRegistry()

		registry1.Register(context.Background(), "provider1", "collection1", []byte("blob1"), "etag1", collection_dto.DefaultHybridConfig())
		registry1.Register(context.Background(), "provider2", "collection2", []byte("blob2"), "etag2", collection_dto.DefaultHybridConfig())

		cache1 := newDiskHybridCache(
			"/cache/hybrid.json",
			registry1,
			WithHybridCacheSandbox(sandbox),
		)
		require.NoError(t, cache1.Persist(context.Background()))

		registry2 := newMockHybridRegistry()
		cache2 := newDiskHybridCache(
			"/cache/hybrid.json",
			registry2,
			WithHybridCacheSandbox(sandbox),
		)
		require.NoError(t, cache2.Load(context.Background()))

		assert.Len(t, registry2.entries, 2)
		assert.Equal(t, []byte("blob1"), registry2.entries["provider1:collection1"].blob)
		assert.Equal(t, "etag1", registry2.entries["provider1:collection1"].etag)
		assert.Equal(t, []byte("blob2"), registry2.entries["provider2:collection2"].blob)
		assert.Equal(t, "etag2", registry2.entries["provider2:collection2"].etag)
	})
}
