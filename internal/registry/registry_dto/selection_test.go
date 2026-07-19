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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sourceVariant(contentHash string) Variant {
	return Variant{
		VariantID:   "source",
		StorageKey:  "source/" + contentHash,
		ContentHash: contentHash,
		Status:      VariantStatusReady,
		Kind:        KindSource,
		Producer:    ProducerBuild,
		CreatedAt:   time.Unix(1000, 0),
	}
}

func derivedVariant(variantID, storageKey, parentHash string) Variant {
	return Variant{
		VariantID:   variantID,
		StorageKey:  storageKey,
		ContentHash: "hash-" + storageKey,
		Status:      VariantStatusReady,
		Kind:        KindDerived,
		Producer:    ProducerBuild,
		CreatedAt:   time.Unix(1000, 0),
		Transform: VariantTransform{
			ParentVariantID:   "source",
			ParentContentHash: parentHash,
			CapabilityName:    "image-transform",
			CapabilityVersion: 1,
		},
	}
}

func TestSelectSourceAlwaysValid(t *testing.T) {
	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1")}}
	got := artefact.SelectVariant("source", "")
	require.NotNil(t, got)
	assert.Equal(t, "source/h1", got.StorageKey)
}

func TestDerivedValidWhenParentMatches(t *testing.T) {
	artefact := &ArtefactMeta{ActualVariants: []Variant{
		sourceVariant("h1"),
		derivedVariant("image_w360", "gen/360", "h1"),
	}}
	got := artefact.SelectVariant("image_w360", "")
	require.NotNil(t, got, "a derived variant whose parent matches must be selectable")
	assert.Equal(t, "gen/360", got.StorageKey)
}

func TestDerivedStaleWhenSourceReplaced(t *testing.T) {
	artefact := &ArtefactMeta{ActualVariants: []Variant{
		sourceVariant("h1"),
		derivedVariant("image_w360", "gen/360", "h1"),
	}}

	upload := sourceVariant("h2")
	upload.Producer = ProducerRuntime
	upload.CreatedAt = time.Unix(2000, 0)
	artefact.ActualVariants = append(artefact.ActualVariants, upload)

	source := artefact.SelectVariant("source", "")
	require.NotNil(t, source)
	assert.Equal(t, "h2", source.ContentHash, "the runtime upload must shadow the build source")

	derived := artefact.SelectVariant("image_w360", "")
	assert.Nil(t, derived,
		"the derived variant was made from the old source, so it must not be served by name after the upload")
}

func TestDerivedStaleWhenCapabilityVersionMoves(t *testing.T) {
	stale := derivedVariant("image_w360", "gen/360", "h1")
	stale.Transform.CapabilityVersion = 999

	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1"), stale}}
	got := artefact.SelectVariant("image_w360", "")
	assert.Nil(t, got, "a variant whose capability version differs from the current one is stale")
}

func TestStaleStatusNotSelected(t *testing.T) {
	stale := derivedVariant("image_w360", "gen/360", "h1")
	stale.Status = VariantStatusStale

	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1"), stale}}
	assert.Nil(t, artefact.SelectVariant("image_w360", ""),
		"a STALE variant must not be served by name")
}

func TestSelectByStorageKeyIgnoresValidity(t *testing.T) {
	stale := derivedVariant("image_w360", "gen/360", "h1")
	stale.Status = VariantStatusStale

	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1"), stale}}
	got := artefact.SelectVariantByStorageKey("gen/360")
	require.NotNil(t, got, "a storage-key lookup is a request for those specific bytes and must resolve")
	assert.Equal(t, "gen/360", got.StorageKey)
}

func TestRuntimeVariantWinsItsID(t *testing.T) {
	build := derivedVariant("image_w360", "gen/build", "h1")
	runtime := derivedVariant("image_w360", "gen/runtime", "h1")
	runtime.Producer = ProducerRuntime

	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1"), build, runtime}}
	got := artefact.SelectVariant("image_w360", "")
	require.NotNil(t, got)
	assert.Equal(t, "gen/runtime", got.StorageKey, "a runtime variant must win its ID over a build one")
}

func TestPreferReleaseWins(t *testing.T) {
	relA := derivedVariant("image_w360", "gen/relA", "h1")
	relA.BuildRelease = "rel-A"
	relB := derivedVariant("image_w360", "gen/relB", "h1")
	relB.BuildRelease = "rel-B"
	relB.CreatedAt = time.Unix(3000, 0)

	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1"), relA, relB}}
	got := artefact.SelectVariant("image_w360", "rel-A")
	require.NotNil(t, got)
	assert.Equal(t, "gen/relA", got.StorageKey, "the instance's own release must win over a newer other release")
}

func TestUnknownKindStatusGatedOnly(t *testing.T) {
	ready := derivedVariant("legacy", "gen/legacy", "h1")
	ready.Kind = KindUnknown
	ready.Transform = VariantTransform{}

	artefact := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h1"), ready}}
	require.NotNil(t, artefact.SelectVariant("legacy", ""),
		"a ready unstamped variant must still be served (backward compatibility)")

	artefact.ActualVariants[1].Status = VariantStatusStale
	assert.Nil(t, artefact.SelectVariant("legacy", ""),
		"a stale unstamped variant must not be served")
}

func TestSelfDependencyDoesNotRecurseForever(t *testing.T) {
	loop := derivedVariant("loop", "gen/loop", "hash-gen/loop")
	loop.Transform.ParentVariantID = "loop"

	artefact := &ArtefactMeta{ActualVariants: []Variant{loop}}
	assert.Nil(t, artefact.SelectVariant("loop", ""),
		"a self-dependent variant must be treated as invalid, not recurse until the stack overflows")
}

func wideDeepArtefact(levels, candidatesPer, winnerIndex int) (*ArtefactMeta, string) {
	meta := &ArtefactMeta{ActualVariants: []Variant{sourceVariant("h0")}}
	parentID := "source"
	parentHash := "h0"
	topKey := ""
	for level := 1; level <= levels; level++ {
		levelID := fmt.Sprintf("L%d", level)
		levelHash := fmt.Sprintf("c%d", level)
		for candidate := range candidatesPer {
			storageKey := fmt.Sprintf("%s/%d", levelID, candidate)
			derived := derivedVariant(levelID, storageKey, parentHash)
			derived.ContentHash = levelHash
			derived.Transform.ParentVariantID = parentID
			derived.CreatedAt = time.Unix(int64(1000+candidate), 0)
			if level == levels && candidate == winnerIndex {
				derived.Producer = ProducerRuntime
				topKey = storageKey
			}
			meta.ActualVariants = append(meta.ActualVariants, derived)
		}
		parentID = levelID
		parentHash = levelHash
	}
	return meta, topKey
}

func TestSelectVariantMemoisesWideDeepGraph(t *testing.T) {
	const (
		levels        = 7
		candidatesPer = 12
		winnerIndex   = 7
	)

	artefact, winnerKey := wideDeepArtefact(levels, candidatesPer, winnerIndex)

	done := make(chan *Variant, 1)
	go func() {
		done <- artefact.SelectVariant("L7", "")
	}()

	select {
	case got := <-done:
		require.NotNil(t, got, "the top level must resolve to a valid variant")
		assert.Equal(t, winnerKey, got.StorageKey,
			"the runtime-produced top candidate must win its ID")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "SelectVariant did not finish; parent resolution is not memoised")
	}
}
