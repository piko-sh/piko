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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestNewMockMetadataStore(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil store", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		require.NotNil(t, store)
	})

	t.Run("implements MetadataStore interface", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		require.Implements(t, (*registry_domain.MetadataStore)(nil), store)
	})
}

func TestMockMetadataStore_GetArtefact(t *testing.T) {
	t.Parallel()

	t.Run("returns error for missing artefact", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		_, err := store.GetArtefact(context.Background(), "missing")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "artefact not found")
	})

	t.Run("retrieves stored artefact", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()
		artefact := &registry_dto.ArtefactMeta{
			ID:         "art-1",
			SourcePath: "images/photo.jpg",
		}

		err := store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: artefact},
		})
		require.NoError(t, err)

		result, err := store.GetArtefact(ctx, "art-1")
		require.NoError(t, err)
		assert.Equal(t, "art-1", result.ID)
		assert.Equal(t, "images/photo.jpg", result.SourcePath)
	})

	t.Run("returns a clone not original reference", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()
		artefact := &registry_dto.ArtefactMeta{
			ID:         "clone-test",
			SourcePath: "original/path",
		}

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: artefact},
		})

		result1, _ := store.GetArtefact(ctx, "clone-test")
		result1.SourcePath = "mutated/path"

		result2, _ := store.GetArtefact(ctx, "clone-test")
		assert.Equal(t, "original/path", result2.SourcePath)
	})
}

func TestMockMetadataStore_GetMultipleArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("returns matching artefacts", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "a"}},
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "b"}},
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "c"}},
		})

		results, err := store.GetMultipleArtefacts(ctx, []string{"a", "c", "missing"})

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		results, err := store.GetMultipleArtefacts(context.Background(), []string{"x", "y"})

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestMockMetadataStore_ListAllArtefactIDs(t *testing.T) {
	t.Parallel()

	t.Run("returns empty for empty store", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		ids, err := store.ListAllArtefactIDs(context.Background())

		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("returns all stored IDs", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "alpha"}},
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "bravo"}},
		})

		ids, err := store.ListAllArtefactIDs(ctx)

		require.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.ElementsMatch(t, []string{"alpha", "bravo"}, ids)
	})
}

func TestMockMetadataStore_SearchArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("returns all artefacts regardless of query", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "one"}},
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "two"}},
		})

		results, err := store.SearchArtefacts(ctx, registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{"format": "webp"},
		})

		require.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestMockMetadataStore_SearchArtefactsByTagValues(t *testing.T) {
	t.Parallel()

	t.Run("finds artefacts matching tag values", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		tags := registry_dto.Tags{}
		tags.SetByName("format", "webp")

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "tagged",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "v1", MetadataTags: tags},
				},
			}},
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "untagged",
			}},
		})

		results, err := store.SearchArtefactsByTagValues(ctx, "format", []string{"webp"})

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "tagged", results[0].ID)
	})

	t.Run("returns empty when no tags match", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "art"}},
		})

		results, err := store.SearchArtefactsByTagValues(ctx, "colour", []string{"red"})

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

func TestMockMetadataStore_FindArtefactByVariantStorageKey(t *testing.T) {
	t.Parallel()

	t.Run("finds artefact by variant storage key", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "indexed-art",
				ActualVariants: []registry_dto.Variant{
					{VariantID: "v1", StorageKey: "storage/key-1"},
				},
			}},
		})

		result, err := store.FindArtefactByVariantStorageKey(ctx, "storage/key-1")

		require.NoError(t, err)
		assert.Equal(t, "indexed-art", result.ID)
	})

	t.Run("returns error for missing storage key", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		_, err := store.FindArtefactByVariantStorageKey(context.Background(), "nonexistent")

		assert.Error(t, err)
	})
}

func TestMockMetadataStore_AtomicUpdate(t *testing.T) {
	t.Parallel()

	t.Run("upsert creates new artefact", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		err := store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "new"}},
		})

		require.NoError(t, err)

		result, err := store.GetArtefact(ctx, "new")
		require.NoError(t, err)
		assert.Equal(t, "new", result.ID)
	})

	t.Run("upsert updates existing artefact", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "update-me", SourcePath: "v1",
			}},
		})

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "update-me", SourcePath: "v2",
			}},
		})

		result, _ := store.GetArtefact(ctx, "update-me")
		assert.Equal(t, "v2", result.SourcePath)
	})

	t.Run("delete removes artefact", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "to-delete",
				ActualVariants: []registry_dto.Variant{
					{StorageKey: "sk-1"},
				},
			}},
		})

		err := store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "to-delete"},
		})
		require.NoError(t, err)

		_, err = store.GetArtefact(ctx, "to-delete")
		assert.Error(t, err)
	})

	t.Run("delete cleans up variant index", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{
				ID: "cleanup-test",
				ActualVariants: []registry_dto.Variant{
					{StorageKey: "variant-key"},
				},
			}},
		})

		_ = store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "cleanup-test"},
		})

		_, err := store.FindArtefactByVariantStorageKey(ctx, "variant-key")
		assert.Error(t, err)
	})

	t.Run("delete of nonexistent artefact is silent", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		err := store.AtomicUpdate(context.Background(), []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeDeleteArtefact, ArtefactID: "nonexistent"},
		})

		assert.NoError(t, err)
	})

	t.Run("gc hints action is accepted silently", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		err := store.AtomicUpdate(context.Background(), []registry_dto.AtomicAction{
			{Type: registry_dto.ActionTypeAddGCHints},
		})

		assert.NoError(t, err)
	})

	t.Run("unknown action type returns error", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		err := store.AtomicUpdate(context.Background(), []registry_dto.AtomicAction{
			{Type: "UNKNOWN_ACTION"},
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action type")
	})
}

func TestMockMetadataStore_PopGCHints(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice and no error", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		hints, err := store.PopGCHints(context.Background(), 10)

		require.NoError(t, err)
		assert.Empty(t, hints)
	})
}

func TestMockMetadataStore_BlobRefCounting(t *testing.T) {
	t.Parallel()

	t.Run("increment creates new reference", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		count, err := store.IncrementBlobRefCount(ctx, registry_domain.BlobReference{
			StorageKey: "blob-1",
			CreatedAt:  time.Now(),
		})

		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("multiple increments accumulate", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()
		blob := registry_domain.BlobReference{StorageKey: "blob-2", CreatedAt: time.Now()}

		_, _ = store.IncrementBlobRefCount(ctx, blob)
		_, _ = store.IncrementBlobRefCount(ctx, blob)
		count, err := store.IncrementBlobRefCount(ctx, blob)

		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("decrement reduces count", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()
		blob := registry_domain.BlobReference{StorageKey: "blob-3", CreatedAt: time.Now()}

		_, _ = store.IncrementBlobRefCount(ctx, blob)
		_, _ = store.IncrementBlobRefCount(ctx, blob)

		count, deleted, err := store.DecrementBlobRefCount(ctx, "blob-3")

		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.False(t, deleted)
	})

	t.Run("decrement to zero deletes reference", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()
		blob := registry_domain.BlobReference{StorageKey: "blob-4", CreatedAt: time.Now()}

		_, _ = store.IncrementBlobRefCount(ctx, blob)

		count, deleted, err := store.DecrementBlobRefCount(ctx, "blob-4")

		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.True(t, deleted)
	})

	t.Run("decrement nonexistent returns error", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		_, _, err := store.DecrementBlobRefCount(context.Background(), "nonexistent")

		assert.Error(t, err)
	})

	t.Run("get ref count returns zero for unknown key", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		count, err := store.GetBlobRefCount(context.Background(), "unknown")

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("get ref count returns current count", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()
		blob := registry_domain.BlobReference{StorageKey: "blob-5", CreatedAt: time.Now()}

		_, _ = store.IncrementBlobRefCount(ctx, blob)
		_, _ = store.IncrementBlobRefCount(ctx, blob)

		count, err := store.GetBlobRefCount(ctx, "blob-5")

		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestMockMetadataStore_RunAtomic(t *testing.T) {
	t.Parallel()

	t.Run("executes function with self as transaction store", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()
		ctx := context.Background()

		err := store.RunAtomic(ctx, func(ctx context.Context, transactionStore registry_domain.MetadataStore) error {
			return transactionStore.AtomicUpdate(ctx, []registry_dto.AtomicAction{
				{Type: registry_dto.ActionTypeUpsertArtefact, Artefact: &registry_dto.ArtefactMeta{ID: "tx-art"}},
			})
		})

		require.NoError(t, err)

		result, err := store.GetArtefact(ctx, "tx-art")
		require.NoError(t, err)
		assert.Equal(t, "tx-art", result.ID)
	})
}

func TestMockMetadataStore_Close(t *testing.T) {
	t.Parallel()

	t.Run("returns nil error", func(t *testing.T) {
		t.Parallel()

		store := NewMockMetadataStore()

		assert.NoError(t, store.Close())
	})
}
