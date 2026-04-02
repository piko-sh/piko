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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/registry/registry_dto"
)

func Test_ArtefactMetaWeigher(t *testing.T) {
	t.Parallel()

	t.Run("Weighs minimal artefact correctly", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:              "test",
			SourcePath:      "source/x",
			ActualVariants:  []registry_dto.Variant{},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		weight := ArtefactMetaWeigher("key", artefact)

		assert.Equal(t, uint32(15), weight)
	})

	t.Run("Weighs artefact with variants", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:         "id",
			SourcePath: "source",
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:        "v1",
					StorageBackendID: "backend",
					StorageKey:       "key",
					MimeType:         "text/plain",
					MetadataTags:     registry_dto.Tags{},
				},
			},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		weight := ArtefactMetaWeigher("k", artefact)

		assert.Equal(t, uint32(31), weight)
	})

	t.Run("Weighs artefact with metadata tags", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:         "id",
			SourcePath: "src",
			ActualVariants: []registry_dto.Variant{
				{
					VariantID:        "v",
					StorageBackendID: "b",
					StorageKey:       "k",
					MimeType:         "m",
					MetadataTags: registry_dto.TagsFromMap(map[string]string{
						"width":  "100",
						"height": "200",
					}),
				},
			},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		weight := ArtefactMetaWeigher("k", artefact)

		assert.Equal(t, uint32(27), weight)
	})

	t.Run("Weighs artefact with desired profiles", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:             "id",
			SourcePath:     "src",
			ActualVariants: []registry_dto.Variant{},
			DesiredProfiles: []registry_dto.NamedProfile{
				{
					Name: "thumb",
					Profile: registry_dto.DesiredProfile{
						CapabilityName: "resize",
					},
				},
			},
		}

		weight := ArtefactMetaWeigher("k", artefact)

		assert.Equal(t, uint32(145), weight)
	})

	t.Run("Weighs artefact with multiple profiles", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:             "id",
			SourcePath:     "src",
			ActualVariants: []registry_dto.Variant{},
			DesiredProfiles: []registry_dto.NamedProfile{
				{Name: "p1", Profile: registry_dto.DesiredProfile{CapabilityName: "c1"}},
				{Name: "p2", Profile: registry_dto.DesiredProfile{CapabilityName: "c2"}},
			},
		}

		weight := ArtefactMetaWeigher("k", artefact)

		assert.Equal(t, uint32(270), weight)
	})

	t.Run("Weighs large artefact correctly", func(t *testing.T) {
		t.Parallel()

		artefact := createLargeArtefact()

		weight := ArtefactMetaWeigher("large-key", artefact)

		assert.Greater(t, weight, uint32(1000))
	})

	t.Run("Weight calculation is deterministic", func(t *testing.T) {
		t.Parallel()

		artefact := createTestArtefact("test")

		weight1 := ArtefactMetaWeigher("key", artefact)
		weight2 := ArtefactMetaWeigher("key", artefact)

		assert.Equal(t, weight1, weight2)
	})

	t.Run("Different keys affect weight", func(t *testing.T) {
		t.Parallel()

		artefact := createTestArtefact("test")

		weight1 := ArtefactMetaWeigher("short", artefact)
		weight2 := ArtefactMetaWeigher("very-long-key", artefact)

		assert.Less(t, weight1, weight2, "Longer keys should result in greater weight")
		assert.Equal(t, uint32(len("very-long-key")-len("short")), weight2-weight1)
	})

	t.Run("Different artefacts have different weights", func(t *testing.T) {
		t.Parallel()

		small := createTestArtefact("small")
		large := createLargeArtefact()

		weightSmall := ArtefactMetaWeigher("key", small)
		weightLarge := ArtefactMetaWeigher("key", large)

		assert.Less(t, weightSmall, weightLarge)
	})

	t.Run("Empty artefact has minimal weight", func(t *testing.T) {
		t.Parallel()

		artefact := &registry_dto.ArtefactMeta{
			ID:              "",
			SourcePath:      "",
			ActualVariants:  []registry_dto.Variant{},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		weight := ArtefactMetaWeigher("", artefact)

		assert.Equal(t, uint32(0), weight)
	})

	t.Run("Weight increases with variant count", func(t *testing.T) {
		t.Parallel()

		base := &registry_dto.ArtefactMeta{
			ID:              "id",
			SourcePath:      "src",
			ActualVariants:  []registry_dto.Variant{},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		with1Variant := &registry_dto.ArtefactMeta{
			ID:         "id",
			SourcePath: "src",
			ActualVariants: []registry_dto.Variant{
				{VariantID: "v1", StorageBackendID: "b", StorageKey: "k", MimeType: "m"},
			},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		with2Variants := &registry_dto.ArtefactMeta{
			ID:         "id",
			SourcePath: "src",
			ActualVariants: []registry_dto.Variant{
				{VariantID: "v1", StorageBackendID: "b", StorageKey: "k", MimeType: "m"},
				{VariantID: "v2", StorageBackendID: "b", StorageKey: "k", MimeType: "m"},
			},
			DesiredProfiles: []registry_dto.NamedProfile{},
		}

		weightBase := ArtefactMetaWeigher("k", base)
		weight1 := ArtefactMetaWeigher("k", with1Variant)
		weight2 := ArtefactMetaWeigher("k", with2Variants)

		assert.Less(t, weightBase, weight1)
		assert.Less(t, weight1, weight2)
	})
}

func createLargeArtefact() *registry_dto.ArtefactMeta {
	artefact := &registry_dto.ArtefactMeta{
		ID:              "large-artefact-with-very-long-identifier-for-testing-purposes",
		SourcePath:      "very/long/source/path/to/artefact/file/in/deep/directory/structure.jpg",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ActualVariants:  make([]registry_dto.Variant, 10),
		DesiredProfiles: make([]registry_dto.NamedProfile, 0, 5),
	}

	for i := range 10 {
		artefact.ActualVariants[i] = registry_dto.Variant{
			VariantID:        fmt.Sprintf("variant-%d-with-long-descriptive-identifier", i),
			StorageBackendID: fmt.Sprintf("storage-backend-%d", i),
			StorageKey:       fmt.Sprintf("storage/key/path/level1/level2/level3/%d/file.dat", i),
			MimeType:         "application/octet-stream",
			MetadataTags: registry_dto.TagsFromMap(map[string]string{
				"key1":       "value1",
				"key2":       "value2",
				"key3":       "value3",
				"resolution": "1920x1080",
				"format":     "progressive",
			}),
		}
	}

	for i := range 5 {
		artefact.DesiredProfiles = append(artefact.DesiredProfiles, registry_dto.NamedProfile{
			Name: fmt.Sprintf("profile-%d-with-descriptive-name", i),
			Profile: registry_dto.DesiredProfile{
				CapabilityName: fmt.Sprintf("capability-%d-processing", i),
			},
		})
	}

	return artefact
}
