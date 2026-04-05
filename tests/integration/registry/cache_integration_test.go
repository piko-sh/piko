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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
)

func TestRegistryCache_Integration(t *testing.T) {

	t.Run("GetArtefact_MissThenHit", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "lib/main.css"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()

		art1, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		require.NotNil(t, art1)
		assert.Equal(t, artefactID, art1.ID)

		art2, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		require.NotNil(t, art2)
		assert.Equal(t, art1, art2, "Should receive the exact same artefact data from the cache")

		art3, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		require.NotNil(t, art3)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("GetArtefact_StampedeProtection", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "components/header.pkc"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return()

		var wg sync.WaitGroup
		const numConcurrentGets = 50

		wg.Add(numConcurrentGets)
		for range numConcurrentGets {
			go func() {
				defer wg.Done()
				art, err := f.service.GetArtefact(ctx, artefactID)
				require.NoError(t, err)
				require.NotNil(t, art)
				assert.Equal(t, artefactID, art.ID)
			}()
		}
		wg.Wait()

		storeCalls := len(f.spyStore.Calls)
		assert.LessOrEqual(t, storeCalls, 5,
			"Expected stampede protection to coalesce calls, but store was called %d times", storeCalls)
		f.spyStore.AssertExpectations(t)

		_, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		f.spyStore.AssertExpectations(t)
	})

	t.Run("GetMultiple_PartialMissThenFullHit", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		idsToGet := []string{"lib/main.css", "components/header.pkc", "non-existent-id"}
		expectedMisses := []string{"components/header.pkc", "non-existent-id"}

		f.spyStore.On("GetArtefact", mock.Anything, "lib/main.css").Return().Once()
		_, err := f.service.GetArtefact(ctx, "lib/main.css")
		require.NoError(t, err)
		f.spyStore.AssertExpectations(t)

		f.spyStore.On("GetMultipleArtefacts", mock.Anything, mock.Anything).Run(func(arguments mock.Arguments) {
			missedIDs, ok := arguments.Get(1).([]string)
			require.True(t, ok, "expected []string argument at index 1")
			assert.ElementsMatch(t, expectedMisses, missedIDs)
		}).Return().Once()

		artefacts1, err := f.service.GetMultipleArtefacts(ctx, idsToGet)
		require.NoError(t, err)
		assert.Len(t, artefacts1, 2, "Should return the 2 existing artefacts")

		f.spyStore.AssertExpectations(t)

		f.spyStore.On("GetMultipleArtefacts", mock.Anything, []string{"non-existent-id"}).Return().Once()
		artefacts2, err := f.service.GetMultipleArtefacts(ctx, idsToGet)
		require.NoError(t, err)
		assert.Len(t, artefacts2, 2)
		f.spyStore.AssertExpectations(t)
	})

	t.Run("WritesAndDeletes_InvalidateCache", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "lib/main.css"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		artV1, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		assert.Equal(t, "source/lib/main.css", artV1.SourcePath)
		f.spyStore.AssertExpectations(t)

		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		_, err = f.service.UpsertArtefact(ctx, artefactID, "path/v2", strings.NewReader("v2"), "test_disk", nil)
		require.NoError(t, err)

		artV2, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		assert.Equal(t, "path/v2", artV2.SourcePath, "Cache should have been updated with the new version")
		f.spyStore.AssertExpectations(t)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, mock.Anything).Return().Times(2)
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		err = f.service.DeleteArtefact(ctx, artefactID)
		require.NoError(t, err)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		_, err = f.service.GetArtefact(ctx, artefactID)
		assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound, "Cache should have been invalidated, leading to a DB miss")
		f.spyStore.AssertExpectations(t)
	})

	t.Run("FindByVariantStorageKey_PrimesCache", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()
		storageKey := "minified/hash2.css"
		artefactID := "lib/main.css"

		f.spyStore.On("FindArtefactByVariantStorageKey", mock.Anything, storageKey).Return().Once()

		art1, err := f.service.FindArtefactByVariantStorageKey(ctx, storageKey)
		require.NoError(t, err)
		require.NotNil(t, art1)
		assert.Equal(t, artefactID, art1.ID)
		f.spyStore.AssertExpectations(t)

		art2, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		require.NotNil(t, art2)
		assert.Equal(t, artefactID, art2.ID)
		f.spyStore.AssertExpectations(t)
	})
}
