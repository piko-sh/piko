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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestTransactionalFailures_Integration(t *testing.T) {

	t.Run("UpsertArtefact_AtomicUpdateFails_BlobUploadedButMetadataFails", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "test-artefact.js"
		content := "console.log('test');"

		transactionErr := errors.New("database transaction failed")

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return().Once()

		f.spyStore.SetFailure("AtomicUpdate", transactionErr)

		artefact, err := f.service.UpsertArtefact(ctx, artefactID, "src/test.js",
			strings.NewReader(content), "test_disk", nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, transactionErr)
		assert.Nil(t, artefact)

		f.spyStore.ClearFailures()

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		_, err = f.service.GetArtefact(ctx, artefactID)
		assert.Error(t, err, "Artefact should not exist after failed transaction")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("UpsertArtefact_IncrementBlobRefCountFails_OperationFails", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "test-artefact2.js"
		content := "console.log('test2');"

		refCountErr := errors.New("blob reference table locked")

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()

		f.spyStore.SetFailure("IncrementBlobRefCount", refCountErr)

		artefact, err := f.service.UpsertArtefact(ctx, artefactID, "src/test2.js",
			strings.NewReader(content), "test_disk", nil)

		require.Error(t, err, "UpsertArtefact should fail when ref count increment fails")
		assert.Nil(t, artefact)
		assert.Contains(t, err.Error(), "blob ref count")

		f.spyStore.ClearFailures()

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		_, err = f.service.GetArtefact(ctx, artefactID)
		assert.Error(t, err, "Artefact should not exist after failed operation")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("DeleteArtefact_AtomicUpdateFails_RefCountsDecrementedButMetadataNotDeleted", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "test-artefact3.js"
		content := "console.log('test3');"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact, err := f.service.UpsertArtefact(ctx, artefactID, "src/test3.js",
			strings.NewReader(content), "test_disk", nil)
		require.NoError(t, err)
		storageKey := artefact.ActualVariants[0].StorageKey

		transactionErr := errors.New("database transaction failed during delete")

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, storageKey).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return().Once()

		f.spyStore.SetFailure("AtomicUpdate", transactionErr)

		err = f.service.DeleteArtefact(ctx, artefactID)
		require.Error(t, err)
		assert.ErrorIs(t, err, transactionErr)

		f.spyStore.ClearFailures()

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		fetchedArtefact, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		assert.Equal(t, artefactID, fetchedArtefact.ID, "Artefact should still exist after failed delete")

		f.spyStore.On("GetBlobRefCount", mock.Anything, storageKey).Return().Once()
		refCount, err := f.spyStore.GetBlobRefCount(ctx, storageKey)
		require.NoError(t, err)
		assert.Equal(t, 1, refCount, "Ref count should be preserved after transaction rollback")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("AddVariant_AtomicUpdateFails_NewVariantBlobExistsButUpdateFails", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "lib/main.css"

		newVariant := "gzipped"

		transactionErr := errors.New("failed to update artefact metadata")

		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return().Once()

		f.spyStore.SetFailure("AtomicUpdate", transactionErr)

		variant := registry_dto.Variant{
			VariantID:        newVariant,
			StorageKey:       "gzipped/hash_new.css",
			StorageBackendID: "test_disk",
			MimeType:         "text/css",
			SizeBytes:        256,
			CreatedAt:        time.Now().UTC(),
			Status:           registry_dto.VariantStatusReady,
		}

		_, err := f.service.AddVariant(ctx, artefactID, &variant)
		require.Error(t, err)
		assert.ErrorIs(t, err, transactionErr)

		f.spyStore.ClearFailures()

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		artefact, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		assert.Len(t, artefact.ActualVariants, 2, "New variant should not be added after failed transaction")

		var foundNewVariant bool
		for _, v := range artefact.ActualVariants {
			if v.VariantID == newVariant {
				foundNewVariant = true
			}
		}
		assert.False(t, foundNewVariant, "New variant should not exist after failed transaction")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("UpsertArtefact_PartialFailureRecovery_SecondAttemptSucceeds", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "recovery-test.js"
		content := "console.log('recovery');"

		transactionErr := errors.New("temporary database error")

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return().Once()

		f.spyStore.SetFailure("AtomicUpdate", transactionErr)

		_, err := f.service.UpsertArtefact(ctx, artefactID, "src/recovery.js",
			strings.NewReader(content), "test_disk", nil)
		require.Error(t, err)

		f.spyStore.ClearFailures()

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact, err := f.service.UpsertArtefact(ctx, artefactID, "src/recovery.js",
			strings.NewReader(content), "test_disk", nil)
		require.NoError(t, err)
		require.NotNil(t, artefact)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		fetchedArtefact, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		assert.Equal(t, artefactID, fetchedArtefact.ID)

		f.spyStore.On("GetBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		refCount, err := f.spyStore.GetBlobRefCount(ctx, artefact.ActualVariants[0].StorageKey)
		require.NoError(t, err)
		assert.Equal(t, 1, refCount, "Only the successful attempt's increment should persist")

		f.spyStore.AssertExpectations(t)
	})
}
