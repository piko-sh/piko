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

package persistence_sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dal/querier_adapter"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func makeArtefact(id string) *registry_dto.ArtefactMeta {
	now := time.Now().UTC().Truncate(time.Second)
	return &registry_dto.ArtefactMeta{
		ID:         id,
		SourcePath: "test/" + id + ".png",
		Status:     registry_dto.VariantStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
		ActualVariants: []registry_dto.Variant{
			{
				VariantID:        "source",
				StorageKey:       "blob/" + id + "/source",
				StorageBackendID: "local",
				MimeType:         "image/png",
				SizeBytes:        1024,
				Status:           registry_dto.VariantStatusReady,
				CreatedAt:        now,
			},
		},
		DesiredProfiles: []registry_dto.NamedProfile{
			{
				Name: "thumbnail",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: "resize",
				},
			},
		},
	}
}

func TestRegistryGetArtefactNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	_, err := dal.GetArtefact(ctx, "nonexistent-id")
	assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
}

func TestRegistryAtomicUpdateUpsertAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	artefact := makeArtefact("art-upsert-001")

	err := dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
		{
			Type:     registry_dto.ActionTypeUpsertArtefact,
			Artefact: artefact,
		},
	})
	require.NoError(t, err, "upserting artefact")

	got, err := dal.GetArtefact(ctx, "art-upsert-001")
	require.NoError(t, err, "getting artefact")

	assert.Equal(t, artefact.ID, got.ID)
	assert.Equal(t, artefact.SourcePath, got.SourcePath)
	assert.Equal(t, len(artefact.ActualVariants), len(got.ActualVariants))
	assert.Equal(t, len(artefact.DesiredProfiles), len(got.DesiredProfiles))

	if len(got.ActualVariants) > 0 {
		assert.Equal(t, artefact.ActualVariants[0].VariantID, got.ActualVariants[0].VariantID)
		assert.Equal(t, artefact.ActualVariants[0].StorageKey, got.ActualVariants[0].StorageKey)
		assert.Equal(t, artefact.ActualVariants[0].MimeType, got.ActualVariants[0].MimeType)
	}

	if len(got.DesiredProfiles) > 0 {
		assert.Equal(t, artefact.DesiredProfiles[0].Name, got.DesiredProfiles[0].Name)
		assert.Equal(t, artefact.DesiredProfiles[0].Profile.CapabilityName, got.DesiredProfiles[0].Profile.CapabilityName)
	}
}

func TestRegistryAtomicUpdateDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	artefact := makeArtefact("art-delete-001")

	err := dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
		{
			Type:     registry_dto.ActionTypeUpsertArtefact,
			Artefact: artefact,
		},
	})
	require.NoError(t, err, "upserting artefact")

	_, err = dal.GetArtefact(ctx, "art-delete-001")
	require.NoError(t, err, "artefact should exist before deletion")

	err = dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
		{
			Type:       registry_dto.ActionTypeDeleteArtefact,
			ArtefactID: "art-delete-001",
		},
	})
	require.NoError(t, err, "deleting artefact")

	_, err = dal.GetArtefact(ctx, "art-delete-001")
	assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
}

func TestRegistryListAllArtefactIDs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	expected_ids := []string{"art-list-001", "art-list-002", "art-list-003"}
	actions := make([]registry_dto.AtomicAction, len(expected_ids))
	for i, id := range expected_ids {
		actions[i] = registry_dto.AtomicAction{
			Type:     registry_dto.ActionTypeUpsertArtefact,
			Artefact: makeArtefact(id),
		}
	}

	err := dal.AtomicUpdate(ctx, actions)
	require.NoError(t, err, "upserting artefacts")

	ids, err := dal.ListAllArtefactIDs(ctx)
	require.NoError(t, err, "listing artefact IDs")

	assert.ElementsMatch(t, expected_ids, ids)
}

func TestRegistryBlobRefCountIncrement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	blob := registry_domain.BlobReference{
		StorageKey:       "blob/increment-test/source",
		StorageBackendID: "local",
		ContentHash:      "sha256:abc123",
		MimeType:         "image/png",
		SizeBytes:        2048,
		CreatedAt:        time.Now().UTC(),
	}

	ref_count, err := dal.IncrementBlobRefCount(ctx, blob)
	require.NoError(t, err, "first increment")
	assert.Equal(t, 1, ref_count)

	ref_count, err = dal.IncrementBlobRefCount(ctx, blob)
	require.NoError(t, err, "second increment")
	assert.Equal(t, 2, ref_count)

	ref_count, err = dal.GetBlobRefCount(ctx, blob.StorageKey)
	require.NoError(t, err, "getting blob ref count")
	assert.Equal(t, 2, ref_count)
}

func TestRegistryBlobRefCountDecrement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	blob := registry_domain.BlobReference{
		StorageKey:       "blob/decrement-test/source",
		StorageBackendID: "local",
		ContentHash:      "sha256:def456",
		MimeType:         "image/jpeg",
		SizeBytes:        4096,
		CreatedAt:        time.Now().UTC(),
	}

	_, err := dal.IncrementBlobRefCount(ctx, blob)
	require.NoError(t, err)
	_, err = dal.IncrementBlobRefCount(ctx, blob)
	require.NoError(t, err)

	ref_count, should_delete, err := dal.DecrementBlobRefCount(ctx, blob.StorageKey)
	require.NoError(t, err, "first decrement")
	assert.Equal(t, 1, ref_count)
	assert.False(t, should_delete)

	ref_count, should_delete, err = dal.DecrementBlobRefCount(ctx, blob.StorageKey)
	require.NoError(t, err, "second decrement")
	assert.Equal(t, 0, ref_count)
	assert.True(t, should_delete)

	ref_count, err = dal.GetBlobRefCount(ctx, blob.StorageKey)
	require.NoError(t, err)
	assert.Equal(t, 0, ref_count)
}

func TestRegistryGCHints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	gc_hints := []registry_dto.GCHint{
		{BackendID: "local", StorageKey: "blob/gc-001"},
		{BackendID: "local", StorageKey: "blob/gc-002"},
		{BackendID: "s3", StorageKey: "blob/gc-003"},
	}

	err := dal.AtomicUpdate(ctx, []registry_dto.AtomicAction{
		{
			Type:    registry_dto.ActionTypeAddGCHints,
			GCHints: gc_hints,
		},
	})
	require.NoError(t, err, "adding GC hints")

	popped, err := dal.PopGCHints(ctx, 100)
	require.NoError(t, err, "popping GC hints")
	assert.Len(t, popped, 3)

	popped_keys := make([]string, len(popped))
	for i, hint := range popped {
		popped_keys[i] = hint.StorageKey
	}
	assert.ElementsMatch(t, []string{"blob/gc-001", "blob/gc-002", "blob/gc-003"}, popped_keys)

	popped, err = dal.PopGCHints(ctx, 100)
	require.NoError(t, err, "popping GC hints again")
	assert.Empty(t, popped)
}

func TestRegistryRunAtomic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	database := setupRegistryDB(t)
	dal := querier_adapter.NewDAL(database)
	ctx := context.Background()

	deliberate_error := errors.New("deliberate rollback")

	err := dal.RunAtomic(ctx, func(ctx context.Context, store registry_domain.MetadataStore) error {
		artefact := makeArtefact("art-atomic-rollback")
		upsert_err := store.AtomicUpdate(ctx, []registry_dto.AtomicAction{
			{
				Type:     registry_dto.ActionTypeUpsertArtefact,
				Artefact: artefact,
			},
		})
		if upsert_err != nil {
			return upsert_err
		}

		_, get_err := store.GetArtefact(ctx, "art-atomic-rollback")
		if get_err != nil {
			return get_err
		}

		return deliberate_error
	})
	assert.ErrorIs(t, err, deliberate_error)

	_, err = dal.GetArtefact(ctx, "art-atomic-rollback")
	assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
}
