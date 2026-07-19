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

package snapshot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func makeSnapshotArtefact(id, storageKey, tagKey, tagValue string) *registry_dto.ArtefactMeta {
	variant := registry_dto.Variant{
		VariantID:        id + "-v1",
		StorageBackendID: "local",
		StorageKey:       storageKey,
		Status:           registry_dto.VariantStatusReady,
		CreatedAt:        time.Unix(1000, 0),
	}
	if tagKey != "" {
		variant.MetadataTags.SetByName(tagKey, tagValue)
	}
	return &registry_dto.ArtefactMeta{
		ID:             id,
		SourcePath:     id + ".png",
		ActualVariants: []registry_dto.Variant{variant},
		CreatedAt:      time.Unix(1000, 0),
		UpdatedAt:      time.Unix(2000, 0),
	}
}

func TestNewIndexesAndClonesArtefacts(t *testing.T) {
	t.Parallel()

	original := makeSnapshotArtefact("art-1", "blob/a", "kind", "image")
	store := New([]*registry_dto.ArtefactMeta{original, nil})

	assert.Equal(t, []string{"art-1"}, store.ArtefactIDs(), "the nil artefact must be skipped and the id indexed")

	got, err := store.GetArtefact(context.Background(), "art-1")
	require.NoError(t, err, "the indexed artefact must be found")
	require.NotNil(t, got, "a found artefact must not be nil")

	got.ID = "mutated"
	stored, err := store.GetArtefact(context.Background(), "art-1")
	require.NoError(t, err, "the second read must still find the artefact")
	assert.Equal(t, "art-1", stored.ID, "each read must return an independent clone")
}

func TestGetArtefactMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	store := New(nil)
	_, err := store.GetArtefact(context.Background(), "absent")
	require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound, "an absent artefact must report not found")
}

func TestGetMultipleArtefactsSkipsAbsent(t *testing.T) {
	t.Parallel()

	store := New([]*registry_dto.ArtefactMeta{
		makeSnapshotArtefact("art-1", "blob/a", "", ""),
		makeSnapshotArtefact("art-2", "blob/b", "", ""),
	})

	got, err := store.GetMultipleArtefacts(context.Background(), []string{"art-1", "absent", "art-2"})
	require.NoError(t, err, "fetching multiple artefacts must not error")
	assert.Len(t, got, 2, "only the present artefacts must be returned")
}

func TestFindArtefactByVariantStorageKey(t *testing.T) {
	t.Parallel()

	store := New([]*registry_dto.ArtefactMeta{makeSnapshotArtefact("art-1", "blob/a", "", "")})

	owner, err := store.FindArtefactByVariantStorageKey(context.Background(), "blob/a")
	require.NoError(t, err, "an owned storage key must resolve")
	assert.Equal(t, "art-1", owner.ID, "the resolved owner must be the indexing artefact")

	_, err = store.FindArtefactByVariantStorageKey(context.Background(), "blob/missing")
	require.ErrorIs(t, err, registry_domain.ErrArtefactNotFound, "an unowned key must report not found")
}

func TestOwnershipAndBlobRefCount(t *testing.T) {
	t.Parallel()

	store := New([]*registry_dto.ArtefactMeta{makeSnapshotArtefact("art-1", "blob/a", "", "")})

	assert.True(t, store.OwnsArtefact("art-1"), "the base owns its indexed artefact")
	assert.False(t, store.OwnsArtefact("art-2"), "the base does not own an absent artefact")
	assert.True(t, store.OwnsStorageKey("blob/a"), "the base owns its indexed storage key")
	assert.False(t, store.OwnsStorageKey("blob/x"), "the base does not own an absent key")

	count, err := store.GetBlobRefCount(context.Background(), "blob/a")
	require.NoError(t, err, "a base blob ref count must not error")
	assert.Equal(t, 1, count, "a base-owned blob is always referenced once")

	count, err = store.GetBlobRefCount(context.Background(), "blob/x")
	require.NoError(t, err, "an absent blob ref count must not error")
	assert.Zero(t, count, "an unowned blob has a zero reference count")
}

func TestSearchArtefacts(t *testing.T) {
	t.Parallel()

	store := New([]*registry_dto.ArtefactMeta{
		makeSnapshotArtefact("art-1", "blob/a", "kind", "image"),
		makeSnapshotArtefact("art-2", "blob/b", "kind", "video"),
	})

	matched, err := store.SearchArtefacts(context.Background(), registry_domain.SearchQuery{
		SimpleTagQuery: map[string]string{"kind": "image"},
	})
	require.NoError(t, err, "a simple tag query must not error")
	require.Len(t, matched, 1, "only the image artefact matches")
	assert.Equal(t, "art-1", matched[0].ID, "the matched artefact must be the image one")

	_, err = store.SearchArtefacts(context.Background(), registry_domain.SearchQuery{
		RawRediSearchQuery: "@kind:image",
	})
	require.ErrorIs(t, err, registry_domain.ErrSearchUnsupported, "a raw full-text query is unsupported")
}

func TestSearchArtefactsByTagValues(t *testing.T) {
	t.Parallel()

	store := New([]*registry_dto.ArtefactMeta{
		makeSnapshotArtefact("art-1", "blob/a", "kind", "image"),
		makeSnapshotArtefact("art-2", "blob/b", "kind", "video"),
	})

	matched, err := store.SearchArtefactsByTagValues(context.Background(), "kind", []string{"image", "video"})
	require.NoError(t, err, "a tag-values search must not error")
	assert.Len(t, matched, 2, "both values match one artefact each")
}

func TestInspectorSummaries(t *testing.T) {
	t.Parallel()

	store := New([]*registry_dto.ArtefactMeta{
		makeSnapshotArtefact("art-1", "blob/a", "", ""),
		makeSnapshotArtefact("art-2", "blob/b", "", ""),
	})

	artefactSummary, err := store.ListArtefactSummary(context.Background())
	require.NoError(t, err, "the artefact summary must not error")
	assert.NotEmpty(t, artefactSummary, "two artefacts produce at least one status bucket")

	variantSummary, err := store.ListVariantSummary(context.Background())
	require.NoError(t, err, "the variant summary must not error")
	assert.NotEmpty(t, variantSummary, "two variants produce at least one status bucket")

	recent, err := store.ListRecentArtefacts(context.Background(), 1)
	require.NoError(t, err, "listing recent artefacts must not error")
	assert.Len(t, recent, 1, "the limit must cap the recent list")
}

func TestWritesAreReadOnly(t *testing.T) {
	t.Parallel()

	store := New(nil)

	require.ErrorIs(t, store.AtomicUpdate(context.Background(), nil), registry_domain.ErrReadOnlyStore,
		"an atomic update must be rejected as read-only")

	_, err := store.IncrementBlobRefCount(context.Background(), registry_domain.BlobReference{StorageKey: "blob/a"})
	require.ErrorIs(t, err, registry_domain.ErrReadOnlyStore, "incrementing a ref count must be rejected as read-only")

	_, _, err = store.DecrementBlobRefCount(context.Background(), "blob/a")
	require.ErrorIs(t, err, registry_domain.ErrReadOnlyStore, "decrementing a ref count must be rejected as read-only")

	require.ErrorIs(t, store.RunAtomic(context.Background(), func(context.Context, registry_domain.MetadataStore) error {
		return nil
	}), registry_domain.ErrReadOnlyStore, "running a transaction must be rejected as read-only")
}
