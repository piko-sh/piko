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

package registry_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_dal/snapshot"
	"piko.sh/piko/internal/registry/registry_dal/union"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func newUnionEmptyBaseStore(t *testing.T) registry_domain.MetadataStore {
	t.Helper()
	overlay := newOtterStore(t)
	base := snapshot.New(nil)
	return union.New(base, overlay)
}

func TestUnionEmptyBaseConformance(t *testing.T) {
	t.Parallel()
	RunStoreSuite(t, Config{
		NewStore:                           newUnionEmptyBaseStore,
		SupportsArtefactLocker:             false,
		SupportsRegistryInspector:          false,
		SupportsRollback:                   true,
		SupportsNestedTransactionRejection: true,
		SupportsGCHints:                    true,
		SupportsSRIHashPersistence:         true,
	})
}

func TestUnionSeededBaseReadMerge(t *testing.T) {
	overlay := newOtterStore(t)

	baseArtefact := buildArtefact("shared", "source/base")
	baseArtefact.ActualVariants[0].InputFingerprint = "fp-base-source"
	baseOnly := buildArtefact("base-only", "source/base-only")
	baseOnly.ActualVariants[0].InputFingerprint = "fp-base-only"
	base := snapshot.New([]*registry_dto.ArtefactMeta{baseArtefact, baseOnly})

	store := union.New(base, overlay)

	overlayArtefact := buildArtefact("shared", "source/base")
	overlayArtefact.ActualVariants[0].InputFingerprint = "fp-base-source"
	runtimeVariant := buildVariant("image_w360", "gen/360")
	runtimeVariant.InputFingerprint = "fp-runtime-360"
	overlayArtefact.ActualVariants = append(overlayArtefact.ActualVariants, runtimeVariant)
	upsert(t, store, overlayArtefact)

	merged, err := store.GetArtefact(ctx(t), "shared")
	require.NoError(t, err)
	require.NotNil(t, findVariant(merged, "source"), "the base source variant must be present")
	require.NotNil(t, findVariant(merged, "image_w360"), "the overlay runtime variant must be present")

	baseServed, err := store.GetArtefact(ctx(t), "base-only")
	require.NoError(t, err)
	require.Len(t, baseServed.ActualVariants, 1, "a base-only artefact is served from the base alone")

	freshOverlay := newOtterStore(t)
	freshBase := snapshot.New([]*registry_dto.ArtefactMeta{baseArtefact, baseOnly})
	fresh := union.New(freshBase, freshOverlay)
	freshShared, err := fresh.GetArtefact(ctx(t), "shared")
	require.NoError(t, err)
	require.Len(t, freshShared.ActualVariants, 1,
		"the base must not carry the overlay's runtime variant; the write went to the overlay only")
}
