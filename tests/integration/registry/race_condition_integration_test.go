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
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestRaceCondition_Integration(t *testing.T) {

	t.Run("AssetPipelineFirst_ThenFileWatcher_EnrichesPlaceholder", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "icons/caret-right-thin.svg"
		sourcePath := "icons/caret-right-thin.svg"
		svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><path d="M10 10"/></svg>`

		desiredProfiles := []registry_dto.NamedProfile{
			{
				Name: "image_w640_webp",
				Profile: registry_dto.DesiredProfile{
					CapabilityName: "image-transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"width":  "640",
						"format": "webp",
					}),
					DependsOn: registry_dto.DependenciesFromSlice([]string{"source"}),
					Priority:  registry_dto.PriorityWant,
				},
			},
		}

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		placeholder, err := f.service.UpsertArtefact(ctx, artefactID, sourcePath,
			nil, "test_disk", desiredProfiles)

		require.NoError(t, err, "Metadata-only update should succeed and create placeholder")
		require.NotNil(t, placeholder)
		assert.Equal(t, artefactID, placeholder.ID)
		assert.Len(t, placeholder.ActualVariants, 0, "Placeholder should have no variants yet")
		assert.Equal(t, desiredProfiles, placeholder.DesiredProfiles)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		enrichedArtefact, err := f.service.UpsertArtefact(ctx, artefactID, sourcePath,
			strings.NewReader(svgContent), "test_disk", nil)

		require.NoError(t, err, "Source data update should enrich the placeholder")
		require.NotNil(t, enrichedArtefact)
		assert.Equal(t, artefactID, enrichedArtefact.ID)
		assert.Len(t, enrichedArtefact.ActualVariants, 1, "Should now have source variant")
		assert.Equal(t, "source", enrichedArtefact.ActualVariants[0].VariantID)
		assert.Equal(t, registry_dto.VariantStatusReady, enrichedArtefact.ActualVariants[0].Status)

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		finalArtefact, err := f.service.GetArtefact(ctx, artefactID)

		require.NoError(t, err)
		assert.Equal(t, artefactID, finalArtefact.ID)
		assert.Len(t, finalArtefact.ActualVariants, 1, "Should have source variant")
		assert.Equal(t, "source", finalArtefact.ActualVariants[0].VariantID)

		f.spyStore.AssertExpectations(t)
	})

	t.Run("FileWatcherFirst_ThenAssetPipeline_UpdatesProfiles", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		artefactID := "icons/search.svg"
		sourcePath := "icons/search.svg"
		svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><circle r="10"/></svg>`

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact1, err := f.service.UpsertArtefact(ctx, artefactID, sourcePath,
			strings.NewReader(svgContent), "test_disk", nil)

		require.NoError(t, err)
		assert.Len(t, artefact1.ActualVariants, 1)
		assert.Equal(t, "source", artefact1.ActualVariants[0].VariantID)

		desiredProfiles := []registry_dto.NamedProfile{
			{
				Name: "image_w320_webp",
				Profile: registry_dto.DesiredProfile{
					CapabilityName: "image-transform",
					Params: registry_dto.ProfileParamsFromMap(map[string]string{
						"width":  "320",
						"format": "webp",
					}),
				},
			},
		}

		f.spyStore.On("GetArtefact", mock.Anything, artefactID).Return().Once()
		f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

		artefact2, err := f.service.UpsertArtefact(ctx, artefactID, sourcePath,
			nil, "test_disk", desiredProfiles)

		require.NoError(t, err)
		assert.Len(t, artefact2.ActualVariants, 1, "Should preserve source variant")
		assert.Equal(t, "source", artefact2.ActualVariants[0].VariantID)
		assert.Equal(t, desiredProfiles, artefact2.DesiredProfiles, "Should update profiles")

		f.spyStore.AssertExpectations(t)
	})

	t.Run("MultipleAssetsRaceCondition_AllSucceed", func(t *testing.T) {
		t.Parallel()
		f := setupIntegrationTest(t, false, "")
		defer f.cleanup()
		ctx := context.Background()

		icons := []string{
			"icons/caret-right-thin.svg",
			"icons/chevron-down.svg",
			"icons/copy.svg",
			"icons/email.svg",
			"icons/pencil.svg",
		}

		for _, iconPath := range icons {
			f.spyStore.On("GetArtefact", mock.Anything, iconPath).Return().Once()
			f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

			artefact, err := f.service.UpsertArtefact(ctx, iconPath, iconPath,
				nil, "test_disk", []registry_dto.NamedProfile{
					{Name: "image_w128_webp", Profile: registry_dto.DesiredProfile{CapabilityName: "image-transform"}},
				})

			require.NoError(t, err, "Metadata-only update should succeed for %s", iconPath)
			assert.Len(t, artefact.ActualVariants, 0, "Placeholder should have no variants")
		}

		for _, iconPath := range icons {
			f.spyStore.On("GetArtefact", mock.Anything, iconPath).Return().Once()
			f.spyStore.On("IncrementBlobRefCount", mock.Anything, mock.Anything).Return().Once()
			f.spyStore.On("AtomicUpdate", mock.Anything, mock.Anything).Return(nil).Once()

			svgContent := `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg>`
			artefact, err := f.service.UpsertArtefact(ctx, iconPath, iconPath,
				strings.NewReader(svgContent), "test_disk", nil)

			require.NoError(t, err, "Source data update should succeed for %s", iconPath)
			assert.Len(t, artefact.ActualVariants, 1, "Should have source variant")
		}

		for _, iconPath := range icons {
			f.spyStore.On("GetArtefact", mock.Anything, iconPath).Return().Once()
			artefact, err := f.service.GetArtefact(ctx, iconPath)

			require.NoError(t, err)
			assert.Equal(t, iconPath, artefact.ID)
			assert.Len(t, artefact.ActualVariants, 1)
			assert.Equal(t, "source", artefact.ActualVariants[0].VariantID)
		}

		f.spyStore.AssertExpectations(t)
	})
}
