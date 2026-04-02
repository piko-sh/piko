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

//go:build integration

package registry_test_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBlobDeduplication_Integration(t *testing.T) {

	t.Run("SameContent_DifferentArtefacts_ShareBlob", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		identicalContent := "const shared = 'identical content';"

		f.spyStore.On("GetArtefact", mock.Anything, "artefact-a.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact1, err := f.service.UpsertArtefact(ctx, "artefact-a.js", "src/a.js",
			strings.NewReader(identicalContent), "test_disk", nil)
		require.NoError(t, err)
		require.NotNil(t, artefact1)
		require.Len(t, artefact1.ActualVariants, 1)

		storageKey1 := artefact1.ActualVariants[0].StorageKey
		t.Logf("First artefact storage key: %s", storageKey1)

		f.spyStore.On("GetArtefact", mock.Anything, "artefact-b.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact2, err := f.service.UpsertArtefact(ctx, "artefact-b.js", "src/b.js",
			strings.NewReader(identicalContent), "test_disk", nil)
		require.NoError(t, err)
		require.NotNil(t, artefact2)
		require.Len(t, artefact2.ActualVariants, 1)

		storageKey2 := artefact2.ActualVariants[0].StorageKey
		t.Logf("Second artefact storage key: %s", storageKey2)

		assert.Equal(t, storageKey1, storageKey2,
			"Identical content should result in the same storage key (blob deduplication)")

		f.spyStore.On("GetBlobRefCount", mock.Anything, storageKey1).Return().Once()
		refCount, err := f.spyStore.GetBlobRefCount(ctx, storageKey1)
		require.NoError(t, err)
		assert.Equal(t, 2, refCount, "Blob should have ref_count=2 (shared by two artefacts)")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("DeleteOneArtefact_SharedBlob_KeepsAlive", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		sharedContent := "const shared = 'this blob is shared';"

		f.spyStore.On("GetArtefact", mock.Anything, "shared-a.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		artefact1, err := f.service.UpsertArtefact(ctx, "shared-a.js", "src/shared-a.js",
			strings.NewReader(sharedContent), "test_disk", nil)
		require.NoError(t, err)

		f.spyStore.On("GetArtefact", mock.Anything, "shared-b.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		artefact2, err := f.service.UpsertArtefact(ctx, "shared-b.js", "src/shared-b.js",
			strings.NewReader(sharedContent), "test_disk", nil)
		require.NoError(t, err)

		sharedStorageKey := artefact1.ActualVariants[0].StorageKey
		assert.Equal(t, sharedStorageKey, artefact2.ActualVariants[0].StorageKey,
			"Both artefacts should share the same blob")

		f.spyStore.On("GetArtefact", mock.Anything, "shared-a.js").Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.MatchedBy(func(actions any) bool {

			return true
		})).Return(nil).Once()

		err = f.service.DeleteArtefact(ctx, "shared-a.js")
		require.NoError(t, err)

		f.spyStore.On("GetBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		refCount, err := f.spyStore.GetBlobRefCount(ctx, sharedStorageKey)
		require.NoError(t, err)
		assert.Equal(t, 1, refCount,
			"Blob should still exist with ref_count=1 after deleting one of two artefacts")

		f.spyStore.On("GetArtefact", mock.Anything, "shared-b.js").Return().Once()
		artefact2AfterDelete, err := f.service.GetArtefact(ctx, "shared-b.js")
		require.NoError(t, err)
		require.NotNil(t, artefact2AfterDelete)
		assert.Equal(t, sharedStorageKey, artefact2AfterDelete.ActualVariants[0].StorageKey)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("DeleteLastReference_BlobMarkedForGC", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		uniqueContent := "const unique = 'only one artefact uses this';"

		f.spyStore.On("GetArtefact", mock.Anything, "unique.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact, err := f.service.UpsertArtefact(ctx, "unique.js", "src/unique.js",
			strings.NewReader(uniqueContent), "test_disk", nil)
		require.NoError(t, err)
		storageKey := artefact.ActualVariants[0].StorageKey

		f.spyStore.On("GetArtefact", mock.Anything, "unique.js").Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, storageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		err = f.service.DeleteArtefact(ctx, "unique.js")
		require.NoError(t, err)

		f.spyStore.On("PopGCHints", mock.Anything, mock.Anything).Return().Once()
		hints, err := f.service.PopGCHints(ctx, 10)
		require.NoError(t, err)
		require.Len(t, hints, 1, "Should have exactly one GC hint for the deleted blob")
		assert.Equal(t, storageKey, hints[0].StorageKey,
			"GC hint should be for the blob that's no longer referenced")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("UpdateContent_OldBlobDecremented_NewBlobIncremented", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		originalContent := "const version = 1;"
		updatedContent := "const version = 2;"

		f.spyStore.On("GetArtefact", mock.Anything, "versioned.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact1, err := f.service.UpsertArtefact(ctx, "versioned.js", "src/versioned.js",
			strings.NewReader(originalContent), "test_disk", nil)
		require.NoError(t, err)
		oldStorageKey := artefact1.ActualVariants[0].StorageKey

		f.spyStore.On("GetArtefact", mock.Anything, "versioned.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, oldStorageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact2, err := f.service.UpsertArtefact(ctx, "versioned.js", "src/versioned.js",
			strings.NewReader(updatedContent), "test_disk", nil)
		require.NoError(t, err)
		newStorageKey := artefact2.ActualVariants[0].StorageKey

		assert.NotEqual(t, oldStorageKey, newStorageKey,
			"Different content should result in different storage keys")

		f.spyStore.On("GetBlobRefCount", mock.Anything, oldStorageKey).Return().Once()
		oldRefCount, err := f.spyStore.GetBlobRefCount(ctx, oldStorageKey)
		require.NoError(t, err)
		assert.Equal(t, 0, oldRefCount,
			"Old blob should have ref_count=0 after being replaced")

		f.spyStore.On("GetBlobRefCount", mock.Anything, newStorageKey).Return().Once()
		newRefCount, err := f.spyStore.GetBlobRefCount(ctx, newStorageKey)
		require.NoError(t, err)
		assert.Equal(t, 1, newRefCount, "New blob should have ref_count=1")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("MetadataOnlyUpdate_NoSourceData_NoRefCountChanges", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		initialContent := "const initial = true;"

		f.spyStore.On("GetArtefact", mock.Anything, "metadata-test.js").Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact1, err := f.service.UpsertArtefact(ctx, "metadata-test.js", "src/metadata-test.js",
			strings.NewReader(initialContent), "test_disk", nil)
		require.NoError(t, err)
		storageKey := artefact1.ActualVariants[0].StorageKey

		f.spyStore.On("GetArtefact", mock.Anything, "metadata-test.js").Return().Once()

		artefact2, err := f.service.UpsertArtefact(ctx, "metadata-test.js", "src/metadata-test.js",
			nil, "test_disk", nil)
		require.NoError(t, err)

		assert.Equal(t, storageKey, artefact2.ActualVariants[0].StorageKey,
			"Metadata-only update should not change storage key")

		f.spyStore.On("GetBlobRefCount", mock.Anything, storageKey).Return().Once()
		refCount, err := f.spyStore.GetBlobRefCount(ctx, storageKey)
		require.NoError(t, err)
		assert.Equal(t, 1, refCount,
			"Metadata-only update should not change blob reference count")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("ThreeArtefacts_SameBlob_DeleteTwo_KeepsOne", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		sharedContent := "export const SHARED_CONSTANT = 42;"

		artefactIDs := []string{"triple-a.js", "triple-b.js", "triple-c.js"}
		var sharedStorageKey string

		for i, artefactID := range artefactIDs {
			f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
			f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
			f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

			artefact, err := f.service.UpsertArtefact(ctx, artefactID, "src/"+artefactID,
				strings.NewReader(sharedContent), "test_disk", nil)
			require.NoError(t, err)

			if i == 0 {
				sharedStorageKey = artefact.ActualVariants[0].StorageKey
			} else {
				assert.Equal(t, sharedStorageKey, artefact.ActualVariants[0].StorageKey,
					"All three artefacts should share the same blob")
			}
		}

		f.spyStore.On("GetBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		refCount, err := f.spyStore.GetBlobRefCount(ctx, sharedStorageKey)
		require.NoError(t, err)
		assert.Equal(t, 3, refCount, "Blob should have ref_count=3")

		f.spyStore.On("GetArtefact", mock.Anything, "triple-a.js").Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		err = f.service.DeleteArtefact(ctx, "triple-a.js")
		require.NoError(t, err)

		f.spyStore.On("GetArtefact", mock.Anything, "triple-b.js").Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		err = f.service.DeleteArtefact(ctx, "triple-b.js")
		require.NoError(t, err)

		f.spyStore.On("GetBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		refCount, err = f.spyStore.GetBlobRefCount(ctx, sharedStorageKey)
		require.NoError(t, err)
		assert.Equal(t, 1, refCount, "Blob should have ref_count=1 after deleting 2 of 3 artefacts")

		f.spyStore.On("PopGCHints", mock.Anything, mock.Anything).Return().Once()
		hints, err := f.service.PopGCHints(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, hints, "No GC hints should exist while blob is still referenced")

		f.spyStore.On("GetArtefact", mock.Anything, "triple-c.js").Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, sharedStorageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		err = f.service.DeleteArtefact(ctx, "triple-c.js")
		require.NoError(t, err)

		f.spyStore.On("PopGCHints", mock.Anything, mock.Anything).Return().Once()
		hints, err = f.service.PopGCHints(ctx, 10)
		require.NoError(t, err)
		require.Len(t, hints, 1, "GC hint should exist after last reference is deleted")
		assert.Equal(t, sharedStorageKey, hints[0].StorageKey)

		f.spyStore.AssertExpectations(t)
	})
}
