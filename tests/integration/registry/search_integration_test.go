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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
)

func TestSearchFunctionality_Integration(t *testing.T) {

	t.Run("FindArtefactByVariantStorageKey_SourceVariant_Found", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		sourceStorageKey := "source/hash1.css"
		expectedArtefactID := "lib/main.css"

		f.spyStore.On("FindArtefactByVariantStorageKey", mock.Anything, sourceStorageKey).Return().Once()

		artefact, err := f.service.FindArtefactByVariantStorageKey(ctx, sourceStorageKey)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, expectedArtefactID, artefact.ID)
		assert.Len(t, artefact.ActualVariants, 2, "lib/main.css should have 2 variants")

		artefact2, err := f.service.GetArtefact(ctx, expectedArtefactID)
		require.NoError(t, err)
		assert.Equal(t, artefact.ID, artefact2.ID, "Should get same artefact from cache")

		f.spyStore.AssertExpectations(t)

		f.spyStore.AssertNotCalled(t, "GetArtefact", mock.Anything, expectedArtefactID)
	})

	t.Run("FindArtefactByVariantStorageKey_GeneratedVariant_Found", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		minifiedStorageKey := "minified/hash2.css"
		expectedArtefactID := "lib/main.css"

		f.spyStore.On("FindArtefactByVariantStorageKey", mock.Anything, minifiedStorageKey).Return().Once()

		artefact, err := f.service.FindArtefactByVariantStorageKey(ctx, minifiedStorageKey)

		require.NoError(t, err)
		require.NotNil(t, artefact)
		assert.Equal(t, expectedArtefactID, artefact.ID)

		var foundMinifiedVariant bool
		for _, v := range artefact.ActualVariants {
			if v.StorageKey == minifiedStorageKey {
				foundMinifiedVariant = true
				assert.Equal(t, "minified", v.VariantID)
			}
		}
		assert.True(t, foundMinifiedVariant, "Should find the minified variant we searched for")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("FindArtefactByVariantStorageKey_NonExistentKey_NotFound", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		nonExistentKey := "nonexistent/blob_key_12345"

		f.spyStore.On("FindArtefactByVariantStorageKey", mock.Anything, nonExistentKey).Return().Once()

		artefact, err := f.service.FindArtefactByVariantStorageKey(ctx, nonExistentKey)

		assert.Error(t, err)
		assert.ErrorIs(t, err, registry_domain.ErrArtefactNotFound)
		assert.Nil(t, artefact)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("SearchArtefactsByTagValues_SingleTag_FindsArtefacts", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		tagKey := "type"
		tagValues := []string{"css"}

		f.spyStore.On("SearchArtefactsByTagValues", mock.Anything, tagKey, tagValues).Return().Once()

		artefacts, err := f.service.SearchArtefactsByTagValues(ctx, tagKey, tagValues)

		require.NoError(t, err)
		require.Len(t, artefacts, 1, "Should find 1 artefact with type=css")

		artefact := artefacts[0]
		assert.Equal(t, "lib/main.css", artefact.ID)

		cachedArtefact, err := f.service.GetArtefact(ctx, artefact.ID)
		require.NoError(t, err)
		assert.Equal(t, artefact.ID, cachedArtefact.ID)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("SearchArtefactsByTagValues_NoResults_ReturnsEmpty", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		tagKey := "type"
		tagValues := []string{"nonexistent-type"}

		f.spyStore.On("SearchArtefactsByTagValues", mock.Anything, tagKey, tagValues).Return().Once()

		artefacts, err := f.service.SearchArtefactsByTagValues(ctx, tagKey, tagValues)

		require.NoError(t, err)
		assert.Empty(t, artefacts, "Should return empty slice when no artefacts match")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("SearchArtefactsByTagValues_EmptyTagValues_ReturnsEmpty", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		tagKey := "type"
		var tagValues []string

		artefacts, err := f.service.SearchArtefactsByTagValues(ctx, tagKey, tagValues)

		require.NoError(t, err)
		assert.Empty(t, artefacts, "Should return empty slice for empty tag values input")

		f.spyStore.AssertNotCalled(t, "SearchArtefactsByTagValues", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("SearchArtefacts_SimpleTagQuery_FindsArtefacts", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		query := registry_domain.SearchQuery{
			SimpleTagQuery: map[string]string{
				"type": "component",
			},
		}

		f.spyStore.On("SearchArtefacts", mock.Anything, query).Return().Once()

		artefacts, err := f.service.SearchArtefacts(ctx, query)

		require.NoError(t, err)
		require.Len(t, artefacts, 1, "Should find 1 artefact with type=component")

		artefact := artefacts[0]
		assert.Equal(t, "components/header.pkc", artefact.ID)

		cachedArtefact, err := f.service.GetArtefact(ctx, artefact.ID)
		require.NoError(t, err)
		assert.Equal(t, artefact.ID, cachedArtefact.ID)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("SearchArtefacts_EmptyQuery_ReturnsError", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		query := registry_domain.SearchQuery{}

		f.spyStore.On("SearchArtefacts", mock.Anything, query).Return().Once()

		artefacts, err := f.service.SearchArtefacts(ctx, query)

		assert.Error(t, err, "Empty query should return an error")
		assert.Nil(t, artefacts)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("FindArtefactByVariantStorageKey_CachePriming_Works", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, true, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		storageKey := "source/hash3.pkc"
		expectedArtefactID := "components/header.pkc"

		f.spyStore.On("FindArtefactByVariantStorageKey", mock.Anything, storageKey).Return().Once()

		artefact1, err := f.service.FindArtefactByVariantStorageKey(ctx, storageKey)
		require.NoError(t, err)
		assert.Equal(t, expectedArtefactID, artefact1.ID)

		artefact2, err := f.service.GetArtefact(ctx, expectedArtefactID)
		require.NoError(t, err)
		assert.Equal(t, artefact1.ID, artefact2.ID)

		f.spyStore.AssertExpectations(t)

		f.spyStore.AssertNotCalled(t, "GetArtefact", mock.Anything, expectedArtefactID)
	})

	t.Run("SearchArtefactsByTagValues_MultipleTags_FindsCorrectArtefacts", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "base_scenario.sql")
		defer f.cleanup()
		ctx := context.Background()

		tagKey := "type"
		tagValues := []string{"css", "component"}

		f.spyStore.On("SearchArtefactsByTagValues", mock.Anything, tagKey, tagValues).Return().Once()

		artefacts, err := f.service.SearchArtefactsByTagValues(ctx, tagKey, tagValues)

		require.NoError(t, err)
		require.Len(t, artefacts, 2, "Should find 2 artefacts matching the tag values")

		artefactIDs := make(map[string]bool)
		for _, a := range artefacts {
			artefactIDs[a.ID] = true
		}

		assert.True(t, artefactIDs["lib/main.css"], "Should find lib/main.css")
		assert.True(t, artefactIDs["components/header.pkc"], "Should find components/header.pkc")

		f.spyStore.AssertExpectations(t)
	})
}
