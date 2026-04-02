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
	"time"

	"github.com/stretchr/testify/mock"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtefactLogic_Integration(t *testing.T) {

	t.Run("Upsert_CascadingInvalidation", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "complex_dependencies.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "scripts/app.js"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()

		originalArt, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)
		require.Len(t, originalArt.ActualVariants, 5, "Fixture should have 5 initial variants")
		for _, v := range originalArt.ActualVariants {
			assert.Equal(t, registry_dto.VariantStatusReady, v.Status, "Initial variants should be READY")
		}

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()

		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()
		updatedArt, err := f.service.UpsertArtefact(ctx, artefactID, "source/app.js", strings.NewReader("new content v2"), "test_disk", []registry_dto.NamedProfile{})
		require.NoError(t, err)
		require.NotNil(t, updatedArt)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		finalArt, err := f.service.GetArtefact(ctx, artefactID)
		require.NoError(t, err)

		var staleCount int
		var readyCount int
		for _, v := range finalArt.ActualVariants {
			if v.Status == registry_dto.VariantStatusStale {
				staleCount++
			}
			if v.Status == registry_dto.VariantStatusReady {
				readyCount++
			}
		}
		assert.Equal(t, 1, readyCount)
		assert.True(t, staleCount >= 4)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("AddVariant_ReplacesExistingAndCreatesGCHint", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "complex_dependencies.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "components/user-profile.pkc"

		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		newVariant := registry_dto.Variant{
			VariantID:        "minified_js",
			StorageKey:       "minified/pkc_hash4_new",
			StorageBackendID: "test_disk",
			MimeType:         "application/javascript",
			SizeBytes:        999,
			CreatedAt:        time.Now(),
		}
		updatedArt, err := f.service.AddVariant(ctx, artefactID, &newVariant)
		require.NoError(t, err)

		var foundNewVariant bool
		for _, v := range updatedArt.ActualVariants {
			if v.VariantID == "minified_js" {
				foundNewVariant = true
				assert.Equal(t, registry_dto.VariantStatusReady, v.Status, "The new variant should be READY")
				assert.Equal(t, "minified/pkc_hash4_new", v.StorageKey, "The new storage key should be present")
				assert.Equal(t, int64(999), v.SizeBytes)
			}
		}
		assert.True(t, foundNewVariant, "The new minified_js variant was not found in the final artefact")
		assert.Len(t, updatedArt.ActualVariants, 3, "Total variant count should remain 3 (source, compiled_js, minified_js)")

		f.spyStore.On("PopGCHints", mock.Anything, mock.Anything).Return().Once()
		hints, err := f.service.PopGCHints(ctx, 10)
		require.NoError(t, err)
		require.Len(t, hints, 1)
		assert.Equal(t, "minified/pkc_hash3_old", hints[0].StorageKey, "A GC hint for the old blob should have been created")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("DeleteArtefact_CreatesGCHintsForAllVariants", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "lib/main.css"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()

		f.spyStore.On("DecrementBlobRefCount", mock.Anything, "source/hash1.css").Return().Once()
		f.spyStore.On("DecrementBlobRefCount", mock.Anything, "minified/hash2.css").Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		err := f.service.DeleteArtefact(ctx, artefactID)
		require.NoError(t, err)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		_, err = f.service.GetArtefact(ctx, artefactID)
		assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)

		f.spyStore.On("PopGCHints", mock.Anything, mock.Anything).Return().Once()
		hints, err := f.service.PopGCHints(ctx, 10)
		require.NoError(t, err)
		require.Len(t, hints, 2)

		foundKeys := map[string]bool{}
		for _, hint := range hints {
			foundKeys[hint.StorageKey] = true
		}
		assert.True(t, foundKeys["source/hash1.css"], "GC hint for source blob should exist")
		assert.True(t, foundKeys["minified/hash2.css"], "GC hint for minified blob should exist")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("Upsert_IdenticalContentIsNoOp", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()
		artefactID := "assets/logo.svg"

		originalContent := "content that hashes to svg_hash1"

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()

		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()

		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		_, err := f.service.UpsertArtefact(ctx, artefactID, "source/assets/logo.svg", strings.NewReader(originalContent), "test_disk", nil)
		require.NoError(t, err)

		f.spyStore.AssertExpectations(t)

		f.spyStore.AssertNotCalled(t, "DecrementBlobRefCount", mock.Anything, mock.Anything)
	})
}
