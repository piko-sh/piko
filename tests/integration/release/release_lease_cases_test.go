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
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestReleaseHeartbeat_AdvancesAndNeverRewinds(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		ctx := t.Context()
		start := time.Unix(1_700_000_000, 0)
		mustPublish(t, store, "rel-heartbeat", start,
			seedArtefact("heartbeat-page", "heartbeat-360", "heartbeat/key-360"))

		lease, exists := releaseLease(t, store, "rel-heartbeat")
		require.True(t, exists, "the published release must hold a lease")
		require.Equal(t, start.Unix(), lease.HeartbeatAt,
			"publishing must stamp the heartbeat with the publish time")

		require.NoError(t, registry_domain.HeartbeatRelease(ctx, store, "rel-heartbeat", start.Add(30*time.Second)),
			"a later heartbeat must succeed")
		lease, exists = releaseLease(t, store, "rel-heartbeat")
		require.True(t, exists, "the lease must survive a heartbeat")
		assert.Equal(t, start.Add(30*time.Second).Unix(), lease.HeartbeatAt,
			"a later heartbeat must advance the lease")

		require.NoError(t, registry_domain.HeartbeatRelease(ctx, store, "rel-heartbeat", start.Add(10*time.Second)),
			"an out-of-order heartbeat must not error")
		lease, exists = releaseLease(t, store, "rel-heartbeat")
		require.True(t, exists, "the lease must survive an out-of-order heartbeat")
		assert.Equal(t, start.Add(30*time.Second).Unix(), lease.HeartbeatAt,
			"an out-of-order heartbeat must not rewind the lease")
	})
}

func TestReleasePublish_StalePublishingLease_IsTakenOver(t *testing.T) {
	forEachPublisherBackend(t, func(t *testing.T, store registry_domain.MetadataStore) {
		ctx := t.Context()
		publisher := requirePublisher(t, store)
		publishTime := time.Unix(1_700_000_000, 0)
		staleHeartbeat := publishTime.Add(-2 * time.Hour).Unix()
		seeds := []*registry_dto.ArtefactMeta{
			seedArtefact("takeover-page", "takeover-360", "takeover/key-360"),
		}
		digest := registry_domain.SeedDigest(seeds)

		won, err := publisher.ClaimRelease(ctx, "rel-takeover", digest, staleHeartbeat, staleHeartbeat)
		require.NoError(t, err, "seeding the stale publishing lease must succeed")
		require.True(t, won, "the freshly seeded lease must be won by this claim")

		outcome, err := registry_domain.PublishRelease(ctx, store, "rel-takeover", digest, seeds, publishTime)
		require.NoError(t, err, "publishing over a stale publishing lease must succeed")
		assert.Equal(t, registry_domain.PublishOutcomePublished, outcome,
			"the stale publishing lease must be taken over and published")

		lease, exists := releaseLease(t, store, "rel-takeover")
		require.True(t, exists, "the taken-over release must hold a lease")
		assert.Equal(t, registry_domain.ReleaseStatePublished, lease.State,
			"the taken-over lease must end in the published state")
	})
}
