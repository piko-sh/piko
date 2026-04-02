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

package registry_adapters

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestNewMetadataCache(t *testing.T) {
	t.Parallel()

	t.Run("Creates valid cache adapter", func(t *testing.T) {
		t.Parallel()

		cache, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
			MaximumSize: 100,
		})
		require.NoError(t, err)
		defer func() { _ = cache.Close(context.Background()) }()

		adapter := NewMetadataCache(cache)

		require.NotNil(t, adapter)
		require.Implements(t, (*registry_domain.MetadataCache)(nil), adapter)
	})
}

func TestMetadataCache_Get(t *testing.T) {
	t.Parallel()

	t.Run("Returns artefact on cache hit", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()
		artefact := createTestArtefact("test-id-1")

		adapter.Set(ctx, artefact)

		result, err := adapter.Get(ctx, "test-id-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test-id-1", result.ID)
		assert.Equal(t, artefact.SourcePath, result.SourcePath)
	})

	t.Run("Returns ErrCacheMiss on cache miss", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		result, err := adapter.Get(ctx, "nonexistent-id")
		assert.ErrorIs(t, err, registry_domain.ErrCacheMiss)
		assert.Nil(t, result)
	})

	t.Run("Retrieves artefact with complex data", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()
		artefact := createComplexArtefact("complex-id")

		adapter.Set(ctx, artefact)

		result, err := adapter.Get(ctx, "complex-id")
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Len(t, result.ActualVariants, 2)
		assert.Len(t, result.DesiredProfiles, 2)
		assert.Equal(t, artefact.ActualVariants[0].VariantID, result.ActualVariants[0].VariantID)
	})
}

func TestMetadataCache_Set(t *testing.T) {
	t.Parallel()

	t.Run("Stores artefact successfully", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()
		artefact := createTestArtefact("set-test-1")

		adapter.Set(ctx, artefact)

		result, err := adapter.Get(ctx, "set-test-1")
		require.NoError(t, err)
		assert.Equal(t, artefact.ID, result.ID)
	})

	t.Run("Overwrites existing artefact", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()
		artefact1 := createTestArtefact("overwrite-test")
		artefact1.SourcePath = "path/v1"

		artefact2 := createTestArtefact("overwrite-test")
		artefact2.SourcePath = "path/v2"

		adapter.Set(ctx, artefact1)
		adapter.Set(ctx, artefact2)

		result, err := adapter.Get(ctx, "overwrite-test")
		require.NoError(t, err)
		assert.Equal(t, "path/v2", result.SourcePath)
	})
}

func TestMetadataCache_GetMultiple(t *testing.T) {
	t.Parallel()

	t.Run("Returns all hits with no misses", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		adapter.Set(ctx, createTestArtefact("art-1"))
		adapter.Set(ctx, createTestArtefact("art-2"))
		adapter.Set(ctx, createTestArtefact("art-3"))

		hits, misses := adapter.GetMultiple(ctx, []string{"art-1", "art-2", "art-3"})

		assert.Len(t, hits, 3)
		assert.Len(t, misses, 0)

		ids := []string{hits[0].ID, hits[1].ID, hits[2].ID}
		assert.ElementsMatch(t, []string{"art-1", "art-2", "art-3"}, ids)
	})

	t.Run("Returns partial hits and misses", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		adapter.Set(ctx, createTestArtefact("hit-1"))
		adapter.Set(ctx, createTestArtefact("hit-2"))

		hits, misses := adapter.GetMultiple(ctx, []string{"hit-1", "miss-1", "hit-2", "miss-2"})

		assert.Len(t, hits, 2)
		assert.Len(t, misses, 2)
		assert.ElementsMatch(t, []string{"miss-1", "miss-2"}, misses)
	})

	t.Run("Returns all misses with no hits", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		hits, misses := adapter.GetMultiple(ctx, []string{"miss-1", "miss-2", "miss-3"})

		assert.Len(t, hits, 0)
		assert.Len(t, misses, 3)
	})

	t.Run("Handles empty input", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		hits, misses := adapter.GetMultiple(ctx, []string{})

		assert.Len(t, hits, 0)
		assert.Len(t, misses, 0)
	})
}

func TestMetadataCache_SetMultiple(t *testing.T) {
	t.Parallel()

	t.Run("Stores multiple artefacts via BulkSet", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()
		artefacts := []*registry_dto.ArtefactMeta{
			createTestArtefact("bulk-1"),
			createTestArtefact("bulk-2"),
			createTestArtefact("bulk-3"),
		}

		adapter.SetMultiple(ctx, artefacts)

		for _, art := range artefacts {
			result, err := adapter.Get(ctx, art.ID)
			require.NoError(t, err)
			assert.Equal(t, art.ID, result.ID)
		}
	})

	t.Run("Handles empty slice", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		adapter.SetMultiple(ctx, []*registry_dto.ArtefactMeta{})
	})

	t.Run("Handles large batch", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		const batchSize = 1000
		artefacts := make([]*registry_dto.ArtefactMeta, batchSize)
		for i := range batchSize {
			artefacts[i] = createTestArtefact(fmt.Sprintf("bulk-large-%d", i))
		}

		adapter.SetMultiple(ctx, artefacts)

		result, err := adapter.Get(ctx, "bulk-large-0")
		require.NoError(t, err)
		assert.Equal(t, "bulk-large-0", result.ID)

		result, err = adapter.Get(ctx, "bulk-large-999")
		require.NoError(t, err)
		assert.Equal(t, "bulk-large-999", result.ID)
	})
}

func TestMetadataCache_Delete(t *testing.T) {
	t.Parallel()

	t.Run("Removes artefact from cache", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()
		artefact := createTestArtefact("delete-test")

		adapter.Set(ctx, artefact)
		adapter.Delete(ctx, "delete-test")

		_, err := adapter.Get(ctx, "delete-test")
		assert.ErrorIs(t, err, registry_domain.ErrCacheMiss)
	})

	t.Run("Handles delete of non-existent key", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		ctx := context.Background()

		adapter.Delete(ctx, "nonexistent-key")
	})
}

func TestMetadataCache_Close(t *testing.T) {
	t.Parallel()

	t.Run("Closes underlying cache", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)

		err := adapter.Close(context.Background())
		assert.NoError(t, err)

		_ = cache.Close(context.Background())
	})

	t.Run("Multiple close calls are safe", func(t *testing.T) {
		t.Parallel()

		cache, adapter := setupTestCache(t)
		defer func() { _ = cache.Close(context.Background()) }()

		err1 := adapter.Close(context.Background())
		err2 := adapter.Close(context.Background())

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

func setupTestCache(t *testing.T) (cache_domain.Cache[string, *registry_dto.ArtefactMeta], registry_domain.MetadataCache) {
	t.Helper()

	otterProvider, err := provider_otter.OtterProviderFactory(cache_dto.Options[string, *registry_dto.ArtefactMeta]{
		MaximumSize: 10000,
	})
	require.NoError(t, err)

	adapter := NewMetadataCache(otterProvider)
	return otterProvider, adapter
}

func createTestArtefact(id string) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID:         id,
		SourcePath: fmt.Sprintf("source/%s", id),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        fmt.Sprintf("%s-v1", id),
				StorageBackendID: "test_disk",
				StorageKey:       fmt.Sprintf("storage/%s", id),
				MimeType:         "text/plain",
			},
		},
		DesiredProfiles: []registry_dto.NamedProfile{},
	}
}

func createComplexArtefact(id string) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID:         id,
		SourcePath: fmt.Sprintf("source/%s", id),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        fmt.Sprintf("%s-original", id),
				StorageBackendID: "disk1",
				StorageKey:       fmt.Sprintf("store/%s/original", id),
				MimeType:         "image/jpeg",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{
					"width":  "1920",
					"height": "1080",
				}),
			},
			{
				VariantID:        fmt.Sprintf("%s-thumb", id),
				StorageBackendID: "disk2",
				StorageKey:       fmt.Sprintf("store/%s/thumb", id),
				MimeType:         "image/webp",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{
					"width":  "200",
					"height": "200",
				}),
			},
		},
		DesiredProfiles: []registry_dto.NamedProfile{
			{
				Name: "thumbnail",
				Profile: registry_dto.DesiredProfile{
					CapabilityName: "image_resize",
				},
			},
			{
				Name: "optimised",
				Profile: registry_dto.DesiredProfile{
					CapabilityName: "image_compress",
				},
			},
		},
	}
}
