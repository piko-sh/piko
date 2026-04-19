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

package registry_dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func layerVariant(variantID, fingerprint, storageKey string) Variant {
	return Variant{
		VariantID:        variantID,
		InputFingerprint: fingerprint,
		StorageKey:       storageKey,
		ContentHash:      "hash-" + storageKey,
		Status:           VariantStatusReady,
		Kind:             KindSource,
		Producer:         ProducerBuild,
		CreatedAt:        time.Unix(1000, 0),
	}
}

func TestMergeNilBaseReturnsOverlay(t *testing.T) {
	overlay := &ArtefactMeta{
		ID:             "art-1",
		SourcePath:     "src/art-1",
		ActualVariants: []Variant{layerVariant("source", "fp-1", "source/1")},
	}
	merged := MergeLayers(nil, []*ArtefactMeta{overlay})
	require.NotNil(t, merged)
	assert.Equal(t, "art-1", merged.ID)
	require.Len(t, merged.ActualVariants, 1)
	assert.Equal(t, "source/1", merged.ActualVariants[0].StorageKey)
}

func TestMergeAllNilReturnsNil(t *testing.T) {
	assert.Nil(t, MergeLayers(nil, nil))
	assert.Nil(t, MergeLayers(nil, []*ArtefactMeta{nil, nil}))
}

func TestMergeUnionsDistinctVariants(t *testing.T) {
	base := &ArtefactMeta{
		ID:             "art-2",
		SourcePath:     "src/art-2",
		ActualVariants: []Variant{layerVariant("source", "fp-src", "source/2")},
	}
	overlay := &ArtefactMeta{
		ID:             "art-2",
		ActualVariants: []Variant{layerVariant("image_w360", "fp-360", "gen/360")},
	}
	merged := MergeLayers(base, []*ArtefactMeta{overlay})
	require.Len(t, merged.ActualVariants, 2, "distinct records from both layers must survive")

	keys := map[string]bool{}
	for _, v := range merged.ActualVariants {
		keys[v.StorageKey] = true
	}
	assert.True(t, keys["source/2"], "the base variant must be present")
	assert.True(t, keys["gen/360"], "the overlay variant must be present")
}

func TestMergeDedupesIdenticalRecord(t *testing.T) {
	base := &ArtefactMeta{
		ID:             "art-3",
		SourcePath:     "src/art-3",
		ActualVariants: []Variant{layerVariant("source", "fp-same", "source/base")},
	}
	overlay := &ArtefactMeta{
		ID:             "art-3",
		ActualVariants: []Variant{layerVariant("source", "fp-same", "source/overlay")},
	}
	merged := MergeLayers(base, []*ArtefactMeta{overlay})
	require.Len(t, merged.ActualVariants, 1, "the same record in both layers must appear once")
	assert.Equal(t, "source/base", merged.ActualVariants[0].StorageKey,
		"the base copy wins because its bytes live in the binary")
}

func TestMergeCoexistsDifferingFingerprints(t *testing.T) {
	base := &ArtefactMeta{
		ID:             "art-4",
		SourcePath:     "src/art-4",
		ActualVariants: []Variant{layerVariant("image_w360", "fp-relA", "gen/relA")},
	}
	overlay := &ArtefactMeta{
		ID:             "art-4",
		ActualVariants: []Variant{layerVariant("image_w360", "fp-relB", "gen/relB")},
	}
	merged := MergeLayers(base, []*ArtefactMeta{overlay})
	require.Len(t, merged.ActualVariants, 2,
		"the same variant ID with different fingerprints must coexist across releases")
}

func TestMergeDoesNotMutateInputs(t *testing.T) {
	base := &ArtefactMeta{
		ID:             "art-5",
		SourcePath:     "src/art-5",
		ActualVariants: []Variant{layerVariant("source", "fp-src", "source/5")},
	}
	overlay := &ArtefactMeta{
		ID:             "art-5",
		ActualVariants: []Variant{layerVariant("image_w360", "fp-360", "gen/360")},
	}
	merged := MergeLayers(base, []*ArtefactMeta{overlay})
	merged.ActualVariants[0].StorageKey = "mutated"

	assert.Equal(t, "source/5", base.ActualVariants[0].StorageKey, "the base layer must not be mutated by a merge")
	assert.Len(t, base.ActualVariants, 1, "the base variant slice must be unchanged")
}

func TestMergeTakesNewestUpdatedAt(t *testing.T) {
	base := &ArtefactMeta{ID: "art-6", UpdatedAt: time.Unix(1000, 0)}
	overlay := &ArtefactMeta{ID: "art-6", UpdatedAt: time.Unix(2000, 0)}
	merged := MergeLayers(base, []*ArtefactMeta{overlay})
	assert.Equal(t, time.Unix(2000, 0), merged.UpdatedAt, "the newest UpdatedAt across layers must win")
}
