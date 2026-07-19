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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
)

func TestReleaseRetire_RemovesLayersLeaseAndReferences(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		ctx := t.Context()
		mustPublish(t, store, "rel-retire", time.Unix(1_700_000_000, 0),
			seedArtefact("retire-page", "retire-360", "retire/key-360"))
		requireStorageKeyResolves(t, store, "retire/key-360", "retire-page")

		require.NoError(t, registry_domain.RetireRelease(ctx, store, "rel-retire"),
			"retiring the release must succeed")

		requireStorageKeyGone(t, store, "retire/key-360")

		_, exists := releaseLease(t, store, "rel-retire")
		assert.False(t, exists, "the retired release's lease must be gone")
		assert.Equal(t, 0, blobRefCount(t, store, "retire/key-360"),
			"the retired layer's blob reference must have been released")

		hints, err := store.PopGCHints(ctx, 100)
		require.NoError(t, err, "popping garbage collection hints must succeed")
		hintedKeys := make([]string, 0, len(hints))
		for _, hint := range hints {
			hintedKeys = append(hintedKeys, hint.StorageKey)
		}
		assert.Contains(t, hintedKeys, "retire/key-360",
			"the blob that reached zero references must be hinted for collection")
	})
}

func TestReleaseRedeployAfterRetire_Republishes(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		seed := seedArtefact("redeploy-page", "redeploy-360", "redeploy/key-360")

		mustPublish(t, store, "rel-redeploy", time.Unix(1_700_000_000, 0), seed)
		require.NoError(t, registry_domain.RetireRelease(t.Context(), store, "rel-redeploy"),
			"retiring the release must succeed")

		mustPublish(t, store, "rel-redeploy", time.Unix(1_700_000_200, 0), seed)

		requireStorageKeyResolves(t, store, "redeploy/key-360", "redeploy-page")
		assert.Equal(t, 1, blobRefCount(t, store, "redeploy/key-360"),
			"the republished layer must reference the blob exactly once")
	})
}

func TestReapExpiredReleases_SkipsOwnAndLiveReleases(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		ctx := t.Context()
		reapTime := time.Unix(1_700_010_000, 0)
		stalePublishTime := reapTime.Add(-2 * time.Hour)

		mustPublish(t, store, "rel-live", stalePublishTime,
			seedArtefact("reap-live", "reap-live-360", "reap/key-live"))
		mustPublish(t, store, "rel-dead", stalePublishTime,
			seedArtefact("reap-dead", "reap-dead-360", "reap/key-dead"))
		mustPublish(t, store, "rel-own", stalePublishTime,
			seedArtefact("reap-own", "reap-own-360", "reap/key-own"))

		require.NoError(t, registry_domain.HeartbeatRelease(ctx, store, "rel-live", reapTime.Add(-10*time.Minute)),
			"refreshing rel-live's heartbeat must succeed")

		retired, err := registry_domain.ReapExpiredReleases(ctx, store, "rel-own", time.Hour, reapTime)
		require.NoError(t, err, "reaping expired releases must succeed")
		assert.Equal(t, 1, retired, "exactly one stale foreign release must be retired")

		_, deadExists := releaseLease(t, store, "rel-dead")
		assert.False(t, deadExists, "the stale foreign release must have been reaped")
		requireStorageKeyGone(t, store, "reap/key-dead")

		_, liveExists := releaseLease(t, store, "rel-live")
		assert.True(t, liveExists, "the freshly heartbeating release must survive the reap")
		requireStorageKeyResolves(t, store, "reap/key-live", "reap-live")

		_, ownExists := releaseLease(t, store, "rel-own")
		assert.True(t, ownExists, "the reaper's own release must survive however stale")
		requireStorageKeyResolves(t, store, "reap/key-own", "reap-own")
	})
}
