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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func runReleases(t *testing.T, config Config) {
	if !config.SupportsReleasePublisher {
		t.Skip("backend does not advertise SupportsReleasePublisher")
	}

	t.Run("publish stamps unstamped seed under the release key", func(t *testing.T) {
		store := config.NewStore(t)
		seedArtefact := buildArtefact("rel-art", "release/blob/one")
		seedArtefact.ReleaseID = ""

		outcome, err := registry_domain.PublishRelease(ctx(t), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{seedArtefact}, time.Unix(1_000, 0))
		require.NoError(t, err, "publishing an unstamped seed must succeed")
		require.Equal(t, registry_domain.PublishOutcomePublished, outcome, "the first publisher wins")

		found, err := store.FindArtefactByVariantStorageKey(ctx(t), "release/blob/one")
		require.NoError(t, err, "the published storage key must resolve")
		assert.Equal(t, "rel-art", found.ID, "the layer must serve under the artefact id")

		require.NoError(t, registry_domain.RetireRelease(ctx(t), store, "rel-1"), "retire must succeed")
		_, err = store.FindArtefactByVariantStorageKey(ctx(t), "release/blob/one")
		require.Error(t, err, "the key must stop resolving once the release retires, proving the layer was keyed by the release and not the runtime layer")
	})

	t.Run("heartbeat advances monotonically", func(t *testing.T) {
		store := config.NewStore(t)
		publisher := releasePublisher(t, store)
		seedArtefact := buildArtefact("rel-art", "release/blob/hb")

		_, err := registry_domain.PublishRelease(ctx(t), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{seedArtefact}, time.Unix(1_000, 0))
		require.NoError(t, err, "publish must succeed")

		require.NoError(t, registry_domain.HeartbeatRelease(ctx(t), store, "rel-1", time.Unix(1_090, 0)), "a later heartbeat must succeed")
		lease, exists, err := publisher.GetRelease(ctx(t), "rel-1")
		require.NoError(t, err, "the lease must read back")
		require.True(t, exists, "the lease must exist")
		assert.Equal(t, int64(1_090), lease.HeartbeatAt, "the heartbeat must advance")

		require.NoError(t, registry_domain.HeartbeatRelease(ctx(t), store, "rel-1", time.Unix(1_050, 0)), "an older heartbeat must not error")
		lease, _, err = publisher.GetRelease(ctx(t), "rel-1")
		require.NoError(t, err, "the lease must read back after the stale heartbeat")
		assert.Equal(t, int64(1_090), lease.HeartbeatAt, "an out-of-order heartbeat must not rewind")
	})

	t.Run("stale publishing lease is deletable for takeover", func(t *testing.T) {
		store := config.NewStore(t)
		publisher := releasePublisher(t, store)

		won, err := publisher.ClaimRelease(ctx(t), "rel-1", "digest-1", 1_000, 1_000)
		require.NoError(t, err, "the claim must succeed")
		require.True(t, won, "the first claim wins")

		require.NoError(t, publisher.DeleteStalePublishingLease(ctx(t), "rel-1", 2_000), "the stale delete must succeed")
		_, exists, err := publisher.GetRelease(ctx(t), "rel-1")
		require.NoError(t, err, "the lease lookup must succeed")
		assert.False(t, exists, "a publishing lease older than the cutoff must be deleted")

		won, err = publisher.ClaimRelease(ctx(t), "rel-1", "digest-1", 3_000, 3_000)
		require.NoError(t, err, "the re-claim must succeed")
		assert.True(t, won, "the release must be claimable after the stale delete")

		require.NoError(t, publisher.MarkReleasePublished(ctx(t), "rel-1", 3_100, 3_100), "publishing must complete")
		require.NoError(t, publisher.DeleteStalePublishingLease(ctx(t), "rel-1", 9_000), "the delete must not error on a published lease")
		_, exists, err = publisher.GetRelease(ctx(t), "rel-1")
		require.NoError(t, err, "the lease lookup must succeed")
		assert.True(t, exists, "a published lease must never be deleted by the stale-publishing takeover")
	})

	t.Run("retire reclaims layers, references, and lease", func(t *testing.T) {
		store := config.NewStore(t)
		publisher := releasePublisher(t, store)
		seedArtefact := buildArtefact("rel-art", "release/blob/reclaim")

		_, err := registry_domain.PublishRelease(ctx(t), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{seedArtefact}, time.Unix(1_000, 0))
		require.NoError(t, err, "publish must succeed")
		count, err := store.GetBlobRefCount(ctx(t), "release/blob/reclaim")
		require.NoError(t, err, "the reference count must read back")
		require.Equal(t, 1, count, "publish must reference the blob once")

		require.NoError(t, registry_domain.RetireRelease(ctx(t), store, "rel-1"), "retire must succeed")

		count, err = store.GetBlobRefCount(ctx(t), "release/blob/reclaim")
		require.NoError(t, err, "the reference count must read back after retire")
		assert.Equal(t, 0, count, "retire must release the blob reference")

		_, exists, err := publisher.GetRelease(ctx(t), "rel-1")
		require.NoError(t, err, "the lease lookup must succeed")
		assert.False(t, exists, "retire must drop the lease")

		hints, err := store.PopGCHints(ctx(t), 10)
		require.NoError(t, err, "hints must pop")
		assert.Contains(t, hintKeys(hints), "release/blob/reclaim", "the zero-reference blob must be hinted")
	})
}

func releasePublisher(t *testing.T, store registry_domain.MetadataStore) registry_domain.ReleasePublisher {
	t.Helper()
	publisher, ok := store.(registry_domain.ReleasePublisher)
	require.True(t, ok, "a backend advertising SupportsReleasePublisher must implement ReleasePublisher")
	return publisher
}
