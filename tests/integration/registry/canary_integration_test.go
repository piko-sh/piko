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

func canaryArtefact(artefactID, releaseID, variantID, storageKey string) *registry_dto.ArtefactMeta {
	now := time.Unix(1_000, 0).UTC()
	return &registry_dto.ArtefactMeta{
		ID:        artefactID,
		ReleaseID: releaseID,
		CreatedAt: now,
		UpdatedAt: now,
		ActualVariants: []registry_dto.Variant{{
			VariantID:        variantID,
			StorageKey:       storageKey,
			StorageBackendID: "overlay",
			MimeType:         "image/png",
			SizeBytes:        128,
			ContentHash:      "hash-" + storageKey,
			Status:           registry_dto.VariantStatusReady,
		}},
	}
}

func TestCanary_BothReleaseStorageKeysResolve(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	releaseA := canaryArtefact("cow", "release-a", "cow-360-a", "key-a")
	releaseB := canaryArtefact("cow", "release-b", "cow-360-b", "key-b")

	outcomeA, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", []*registry_dto.ArtefactMeta{releaseA}, time.Unix(1_000, 0))
	require.NoError(t, err, "publishing release A must succeed")
	require.Equal(t, registry_domain.PublishOutcomePublished, outcomeA, "release A is the first publisher")

	outcomeB, err := registry_domain.PublishRelease(ctx, store, "release-b", "digest-b", []*registry_dto.ArtefactMeta{releaseB}, time.Unix(1_001, 0))
	require.NoError(t, err, "publishing release B must succeed alongside release A")
	require.Equal(t, registry_domain.PublishOutcomePublished, outcomeB, "release B publishes independently")

	merged, err := store.GetArtefact(ctx, "cow")
	require.NoError(t, err, "the shared artefact must read back")
	variantIDs := make(map[string]bool)
	for _, v := range merged.ActualVariants {
		variantIDs[v.VariantID] = true
	}
	assert.True(t, variantIDs["cow-360-a"], "release A's variant must survive the merge")
	assert.True(t, variantIDs["cow-360-b"], "release B's variant must survive the merge")

	fromA, err := store.FindArtefactByVariantStorageKey(ctx, "key-a")
	require.NoError(t, err, "release A's storage key must resolve")
	assert.Equal(t, "cow", fromA.ID, "key-a belongs to the cow artefact")

	fromB, err := store.FindArtefactByVariantStorageKey(ctx, "key-b")
	require.NoError(t, err, "release B's storage key must resolve")
	assert.Equal(t, "cow", fromB.ID, "key-b belongs to the cow artefact")
}

func TestCanary_ConcurrentIdenticalPublishIsIdempotent(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	artefact := canaryArtefact("cow", "release-a", "cow-360-a", "key-a")

	first, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", []*registry_dto.ArtefactMeta{artefact}, time.Unix(1_000, 0))
	require.NoError(t, err, "the first node publishes")
	require.Equal(t, registry_domain.PublishOutcomePublished, first, "the first node wins the claim")

	second, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", []*registry_dto.ArtefactMeta{artefact}, time.Unix(1_002, 0))
	require.NoError(t, err, "the second node must not error")
	assert.Equal(t, registry_domain.PublishOutcomeAlreadyPublished, second, "the second node finds it already published")

	refCount, err := store.GetBlobRefCount(ctx, "key-a")
	require.NoError(t, err, "the blob reference count must read back")
	assert.Equal(t, 1, refCount, "an idempotent re-publish must not double the reference count")
}

func TestCanary_RetireRemovesOnlyThatRelease(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	releaseA := canaryArtefact("cow", "release-a", "cow-360-a", "key-a")
	releaseB := canaryArtefact("cow", "release-b", "cow-360-b", "key-b")
	_, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", []*registry_dto.ArtefactMeta{releaseA}, time.Unix(1_000, 0))
	require.NoError(t, err, "release A publishes")
	_, err = registry_domain.PublishRelease(ctx, store, "release-b", "digest-b", []*registry_dto.ArtefactMeta{releaseB}, time.Unix(1_001, 0))
	require.NoError(t, err, "release B publishes")

	require.NoError(t, registry_domain.RetireRelease(ctx, store, "release-b"), "retiring release B must succeed")

	merged, err := store.GetArtefact(ctx, "cow")
	require.NoError(t, err, "the artefact must still read back from release A alone")
	for _, v := range merged.ActualVariants {
		assert.NotEqual(t, "cow-360-b", v.VariantID, "release B's variant must be gone after retire")
	}

	fromA, err := store.FindArtefactByVariantStorageKey(ctx, "key-a")
	require.NoError(t, err, "release A's storage key must still resolve after B is retired")
	assert.Equal(t, "cow", fromA.ID, "release A remains serveable")
}

func TestCanary_HeartbeatAdvancesLease(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	artefact := canaryArtefact("cow", "release-a", "cow-360-a", "key-a")
	_, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", []*registry_dto.ArtefactMeta{artefact}, time.Unix(1_000, 0))
	require.NoError(t, err, "publish must succeed before heartbeating")

	publisher, ok := store.(registry_domain.ReleasePublisher)
	require.True(t, ok, "the SQLite store must implement ReleasePublisher")

	require.NoError(t, registry_domain.HeartbeatRelease(ctx, store, "release-a", time.Unix(1_090, 0)), "a later heartbeat must succeed")
	lease, exists, err := publisher.GetRelease(ctx, "release-a")
	require.NoError(t, err, "the lease must read back")
	require.True(t, exists, "the lease must exist")
	assert.Equal(t, int64(1_090), lease.HeartbeatAt, "the stored heartbeat must advance to the newer value")

	require.NoError(t, registry_domain.HeartbeatRelease(ctx, store, "release-a", time.Unix(1_050, 0)), "an older heartbeat must not error")
	lease, _, err = publisher.GetRelease(ctx, "release-a")
	require.NoError(t, err, "the lease must read back after the stale heartbeat")
	assert.Equal(t, int64(1_090), lease.HeartbeatAt, "an out-of-order heartbeat must not rewind the stored value")
}

func TestCanary_RetireDropsLeaseAndAllowsReclaim(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	seed := []*registry_dto.ArtefactMeta{canaryArtefact("cow", "release-a", "cow-360-a", "key-a")}
	_, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", seed, time.Unix(1_000, 0))
	require.NoError(t, err, "publish must succeed before retire")

	require.NoError(t, registry_domain.RetireRelease(ctx, store, "release-a"), "retire must succeed")

	publisher, ok := store.(registry_domain.ReleasePublisher)
	require.True(t, ok, "the SQLite store must implement ReleasePublisher")
	_, exists, err := publisher.GetRelease(ctx, "release-a")
	require.NoError(t, err, "the lease lookup must succeed")
	assert.False(t, exists, "the lease row must be gone after retire")

	outcome, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", seed, time.Unix(2_000, 0))
	require.NoError(t, err, "a redeploy after retire must publish cleanly")
	assert.Equal(t, registry_domain.PublishOutcomePublished, outcome, "the redeploy must re-claim and republish")

	fromA, err := store.FindArtefactByVariantStorageKey(ctx, "key-a")
	require.NoError(t, err, "the republished storage key must resolve")
	assert.Equal(t, "cow", fromA.ID, "the republished layer must serve again")
}

func TestCanary_StalePublishingLeaseIsTakenOver(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	publisher, ok := store.(registry_domain.ReleasePublisher)
	require.True(t, ok, "the SQLite store must implement ReleasePublisher")

	staleClaim := time.Unix(1_000, 0)
	won, err := publisher.ClaimRelease(ctx, "release-a", "digest-a", staleClaim.Unix(), staleClaim.Unix())
	require.NoError(t, err, "seeding the stale publishing lease must succeed")
	require.True(t, won, "the seeded claim must win")

	takeoverTime := staleClaim.Add(time.Hour)
	seed := []*registry_dto.ArtefactMeta{canaryArtefact("cow", "release-a", "cow-360-a", "key-a")}
	outcome, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", seed, takeoverTime)

	require.NoError(t, err, "the takeover publish must succeed")
	assert.Equal(t, registry_domain.PublishOutcomePublished, outcome, "the taker publishes after deleting the stale lease")

	lease, exists, err := publisher.GetRelease(ctx, "release-a")
	require.NoError(t, err, "the lease must read back")
	require.True(t, exists, "a lease must exist after takeover")
	assert.Equal(t, registry_domain.ReleaseStatePublished, lease.State, "the taken-over lease must end published")

	fromA, err := store.FindArtefactByVariantStorageKey(ctx, "key-a")
	require.NoError(t, err, "the published storage key must resolve")
	assert.Equal(t, "cow", fromA.ID, "the taken-over publish must serve")
}

func TestCanary_RetireReclaimsBlobReferences(t *testing.T) {
	store := newSQLiteConformanceStore(t)
	ctx := t.Context()

	seed := []*registry_dto.ArtefactMeta{canaryArtefact("cow", "release-a", "cow-360-a", "key-a")}
	_, err := registry_domain.PublishRelease(ctx, store, "release-a", "digest-a", seed, time.Unix(1_000, 0))
	require.NoError(t, err, "publish must succeed before retire")
	refCount, err := store.GetBlobRefCount(ctx, "key-a")
	require.NoError(t, err, "the reference count must read back after publish")
	require.Equal(t, 1, refCount, "publish must reference the blob once")

	require.NoError(t, registry_domain.RetireRelease(ctx, store, "release-a"), "retire must succeed")

	refCount, err = store.GetBlobRefCount(ctx, "key-a")
	require.NoError(t, err, "the reference count must read back after retire")
	assert.Equal(t, 0, refCount, "retire must release the blob reference")

	hints, err := store.PopGCHints(ctx, 10)
	require.NoError(t, err, "hints must pop")
	hintKeys := make(map[string]bool)
	for _, hint := range hints {
		hintKeys[hint.StorageKey] = true
	}
	assert.True(t, hintKeys["key-a"], "the zero-reference blob must be hinted for collection")

	require.NoError(t, registry_domain.RetireRelease(ctx, store, "release-a"), "a second retire must be a no-op")
	refCount, err = store.GetBlobRefCount(ctx, "key-a")
	require.NoError(t, err, "the reference count must read back after the second retire")
	assert.Equal(t, 0, refCount, "a second retire must not double-decrement")
}
