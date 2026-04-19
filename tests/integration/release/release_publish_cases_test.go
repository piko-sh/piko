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

package release_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func roundTripSeed(t *testing.T, seed *registry_dto.ArtefactMeta) *registry_dto.ArtefactMeta {
	t.Helper()
	encoded, err := json.Marshal(seed)
	require.NoError(t, err, "encoding the seed artefact")
	decoded := &registry_dto.ArtefactMeta{}
	require.NoError(t, json.Unmarshal(encoded, decoded), "decoding the seed artefact")
	return decoded
}

func TestReleasePublish_StampsLayersUnderReleaseKey_FromUnstampedSeed(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		seed := roundTripSeed(t, seedArtefact("stamp-page", "stamp-360", "stamp/key-360"))
		require.Empty(t, seed.ReleaseID, "a seed artefact must arrive with an empty release id")

		mustPublish(t, store, "rel-stamp", time.Unix(1_700_000_000, 0), seed)
		requireStorageKeyResolves(t, store, "stamp/key-360", "stamp-page")

		require.NoError(t, registry_domain.RetireRelease(t.Context(), store, "rel-stamp"),
			"retiring the release must succeed")
		requireStorageKeyGone(t, store, "stamp/key-360")
	})
}

func TestReleasePublish_TwoReleasesCoexist_BothStorageKeysResolve(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		seedA := seedArtefact("coexist-page", "coexist-360-a", "coexist/key-a")
		seedB := seedArtefact("coexist-page", "coexist-360-b", "coexist/key-b")

		mustPublish(t, store, "rel-coexist-a", time.Unix(1_700_000_000, 0), seedA)
		mustPublish(t, store, "rel-coexist-b", time.Unix(1_700_000_100, 0), seedB)

		requireStorageKeyResolves(t, store, "coexist/key-a", "coexist-page")
		requireStorageKeyResolves(t, store, "coexist/key-b", "coexist-page")

		leaseA, existsA := releaseLease(t, store, "rel-coexist-a")
		require.True(t, existsA, "release rel-coexist-a must hold a lease")
		assert.Equal(t, registry_domain.ReleaseStatePublished, leaseA.State,
			"release rel-coexist-a must report published")

		leaseB, existsB := releaseLease(t, store, "rel-coexist-b")
		require.True(t, existsB, "release rel-coexist-b must hold a lease")
		assert.Equal(t, registry_domain.ReleaseStatePublished, leaseB.State,
			"release rel-coexist-b must report published")
	})
}

func TestReleasePublish_ConcurrentIdenticalPublish_IsIdempotent(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		ctx := t.Context()
		seeds := []*registry_dto.ArtefactMeta{seedArtefact("race-page", "race-360", "race/key-360")}
		digest := registry_domain.SeedDigest(seeds)
		publishTime := time.Unix(1_700_000_000, 0)

		publishErrors := make([]error, 2)
		var racers sync.WaitGroup
		for i := range publishErrors {
			racers.Go(func() {
				_, publishErrors[i] = registry_domain.PublishRelease(ctx, store, "rel-race", digest, seeds, publishTime)
			})
		}
		racers.Wait()

		require.NoError(t, publishErrors[0], "the first concurrent publisher must not error")
		require.NoError(t, publishErrors[1], "the second concurrent publisher must not error")

		outcome, err := registry_domain.PublishRelease(ctx, store, "rel-race", digest, seeds, publishTime.Add(time.Second))
		require.NoError(t, err, "a sequential re-publish must not error")
		assert.Equal(t, registry_domain.PublishOutcomeAlreadyPublished, outcome,
			"a re-publish after the race must find the release already published")

		assert.Equal(t, 1, blobRefCount(t, store, "race/key-360"),
			"concurrent identical publishes must reference the blob exactly once")
	})
}

func TestReleasePublish_DigestConflict_IsRejected(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		firstSeed := seedArtefact("conflict-page", "conflict-360-a", "conflict/key-a")
		mustPublish(t, store, "rel-conflict", time.Unix(1_700_000_000, 0), firstSeed)

		conflictingSeeds := []*registry_dto.ArtefactMeta{
			seedArtefact("conflict-page", "conflict-360-b", "conflict/key-b"),
		}
		conflictingDigest := registry_domain.SeedDigest(conflictingSeeds)
		_, err := registry_domain.PublishRelease(t.Context(), store, "rel-conflict",
			conflictingDigest, conflictingSeeds, time.Unix(1_700_000_100, 0))
		require.ErrorIs(t, err, registry_domain.ErrReleaseDigestConflict,
			"a different digest under one release id must be rejected")

		assert.Equal(t, 1, blobRefCount(t, store, "conflict/key-a"),
			"the published payload's blob reference count must be unchanged by the rejected publish")
		assert.Equal(t, 0, blobRefCount(t, store, "conflict/key-b"),
			"the rejected payload must not have referenced any blob")
	})
}

func TestReleasePublish_UnsupportedOverlay_IsNoOp(t *testing.T) {
	store := newOtterStore(t)
	ctx := t.Context()
	seeds := []*registry_dto.ArtefactMeta{
		seedArtefact("unsupported-page", "unsupported-360", "unsupported/key-360"),
	}

	outcome, err := registry_domain.PublishRelease(ctx, store, "rel-unsupported",
		registry_domain.SeedDigest(seeds), seeds, time.Unix(1_700_000_000, 0))
	require.NoError(t, err, "publishing to a backend without ReleasePublisher must not error")
	assert.Equal(t, registry_domain.PublishOutcomeUnsupported, outcome,
		"a backend without ReleasePublisher must report the publish as unsupported")

	assert.NoError(t, registry_domain.RetireRelease(ctx, store, "rel-unsupported"),
		"retiring on a backend without ReleasePublisher must be a no-op")
}
