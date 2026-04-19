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

package registry_domain

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
)

type fakeReleaseStore struct {
	*MockMetadataStore
	mu             sync.Mutex
	leases         map[string]ReleaseLease
	layers         map[string]*registry_dto.ArtefactMeta
	refCounts      map[string]int
	gcHints        []registry_dto.GCHint
	heartbeatCalls int
}

func newFakeReleaseStore() *fakeReleaseStore {
	fake := &fakeReleaseStore{
		MockMetadataStore: &MockMetadataStore{},
		leases:            map[string]ReleaseLease{},
		layers:            map[string]*registry_dto.ArtefactMeta{},
		refCounts:         map[string]int{},
	}
	fake.IncrementBlobRefCountFunc = func(_ context.Context, blob BlobReference) (int, error) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		fake.refCounts[blob.StorageKey]++
		return fake.refCounts[blob.StorageKey], nil
	}
	fake.DecrementBlobRefCountFunc = func(_ context.Context, storageKey string) (int, bool, error) {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		if fake.refCounts[storageKey] > 0 {
			fake.refCounts[storageKey]--
		}
		remaining := fake.refCounts[storageKey]
		return remaining, remaining <= 0, nil
	}
	fake.RunAtomicFunc = func(ctx context.Context, fn func(ctx context.Context, transactionStore MetadataStore) error) error {
		return fn(ctx, fake)
	}
	fake.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		for _, action := range actions {
			if action.Type == registry_dto.ActionTypeAddGCHints {
				fake.gcHints = append(fake.gcHints, action.GCHints...)
			}
		}
		return nil
	}
	return fake
}

func layerKey(artefactID, releaseID string) string { return artefactID + "|" + releaseID }

func (f *fakeReleaseStore) InsertArtefactLayerIfAbsent(_ context.Context, artefact *registry_dto.ArtefactMeta) (bool, error) {
	if artefact.ReleaseID == "" {
		return false, fmt.Errorf("refusing to publish artefact '%s' as a layer with an empty release id", artefact.ID)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	key := layerKey(artefact.ID, artefact.ReleaseID)
	if f.layers[key] != nil {
		return false, nil
	}
	f.layers[key] = artefact.Clone()
	return true, nil
}

func (f *fakeReleaseStore) DeleteArtefactLayersForRelease(_ context.Context, releaseID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key := range f.layers {
		if strings.HasSuffix(key, "|"+releaseID) {
			delete(f.layers, key)
		}
	}
	return nil
}

func (f *fakeReleaseStore) ReclaimArtefactLayersForRelease(_ context.Context, releaseID string) ([]*registry_dto.ArtefactMeta, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	reclaimed := make([]*registry_dto.ArtefactMeta, 0)
	for key, layer := range f.layers {
		if strings.HasSuffix(key, "|"+releaseID) {
			reclaimed = append(reclaimed, layer)
			delete(f.layers, key)
		}
	}
	return reclaimed, nil
}

func (f *fakeReleaseStore) DeleteReleaseLease(_ context.Context, releaseID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.leases, releaseID)
	return nil
}

func (f *fakeReleaseStore) DeleteStalePublishingLease(_ context.Context, releaseID string, staleBefore int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	lease, ok := f.leases[releaseID]
	if ok && lease.State == ReleaseStatePublishing && lease.HeartbeatAt < staleBefore {
		delete(f.leases, releaseID)
	}
	return nil
}

func (f *fakeReleaseStore) ClaimRelease(_ context.Context, releaseID, publishDigest string, firstSeenAt, heartbeatAt int64) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.leases[releaseID]; exists {
		return false, nil
	}
	f.leases[releaseID] = ReleaseLease{
		ReleaseID:     releaseID,
		PublishDigest: publishDigest,
		State:         ReleaseStatePublishing,
		FirstSeenAt:   firstSeenAt,
		HeartbeatAt:   heartbeatAt,
	}
	return true, nil
}

func (f *fakeReleaseStore) GetRelease(_ context.Context, releaseID string) (ReleaseLease, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	lease, ok := f.leases[releaseID]
	return lease, ok, nil
}

func (f *fakeReleaseStore) MarkReleasePublished(_ context.Context, releaseID string, publishedAt, heartbeatAt int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	lease := f.leases[releaseID]
	lease.ReleaseID = releaseID
	lease.State = ReleaseStatePublished
	lease.PublishedAt = publishedAt
	lease.HeartbeatAt = max(lease.HeartbeatAt, heartbeatAt)
	f.leases[releaseID] = lease
	return nil
}

func (f *fakeReleaseStore) HeartbeatRelease(_ context.Context, releaseID string, heartbeatAt int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heartbeatCalls++
	lease, ok := f.leases[releaseID]
	if !ok {
		return nil
	}
	if heartbeatAt > lease.HeartbeatAt {
		lease.HeartbeatAt = heartbeatAt
		f.leases[releaseID] = lease
	}
	return nil
}

func (f *fakeReleaseStore) ListExpiredReleases(_ context.Context, cutoff int64, ownRelease string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	expired := make([]string, 0)
	for id, lease := range f.leases {
		if id == ownRelease {
			continue
		}
		if lease.State == ReleaseStatePublished && lease.HeartbeatAt < cutoff {
			expired = append(expired, id)
		}
	}
	return expired, nil
}

func (f *fakeReleaseStore) leaseState(releaseID string) (ReleaseLease, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	lease, ok := f.leases[releaseID]
	return lease, ok
}

func (f *fakeReleaseStore) hasLayer(artefactID, releaseID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.layers[layerKey(artefactID, releaseID)] != nil
}

func makeReleaseArtefact(id, release string) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID:        id,
		ReleaseID: release,
		ActualVariants: []registry_dto.Variant{{
			VariantID:        id + "-variant",
			StorageKey:       id + "-key",
			StorageBackendID: "overlay",
			SizeBytes:        10,
			MimeType:         "image/png",
			ContentHash:      "hash-" + id,
		}},
	}
}

func TestPublishRelease_UnsupportedOverlayIsNoOp(t *testing.T) {
	t.Parallel()

	plain := &MockMetadataStore{}
	outcome, err := PublishRelease(t.Context(), plain, "rel-1", "digest", []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}, time.Unix(100, 0))

	require.NoError(t, err, "an overlay without ReleasePublisher must publish nothing without error")
	assert.Equal(t, PublishOutcomeUnsupported, outcome, "a plain overlay is unsupported")
}

func TestPublishRelease_EmptyReleaseIDErrors(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	_, err := PublishRelease(t.Context(), store, "", "digest", nil, time.Unix(100, 0))

	require.Error(t, err, "publishing without a release id must fail")
	assert.Contains(t, err.Error(), "empty release id", "the error must name the missing release id")
}

func TestPublishRelease_PublishesLayersAndReferencesBlobs(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	artefacts := []*registry_dto.ArtefactMeta{
		makeReleaseArtefact("a", "rel-1"),
		makeReleaseArtefact("b", "rel-1"),
	}

	outcome, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", artefacts, time.Unix(100, 0))

	require.NoError(t, err, "publishing a fresh release must succeed")
	assert.Equal(t, PublishOutcomePublished, outcome, "the claiming node publishes")
	assert.True(t, store.hasLayer("a", "rel-1"), "artefact a's layer must be inserted")
	assert.True(t, store.hasLayer("b", "rel-1"), "artefact b's layer must be inserted")
	assert.Equal(t, 1, store.refCounts["a-key"], "blob a must be referenced exactly once")
	assert.Equal(t, 1, store.refCounts["b-key"], "blob b must be referenced exactly once")

	lease, ok := store.leaseState("rel-1")
	require.True(t, ok, "a lease must exist after publish")
	assert.Equal(t, ReleaseStatePublished, lease.State, "the lease must be flipped to published")
}

func TestPublishRelease_SecondNodeIsIdempotent(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	artefacts := []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}

	first, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", artefacts, time.Unix(100, 0))
	require.NoError(t, err, "the first publish must succeed")
	require.Equal(t, PublishOutcomePublished, first, "the first node publishes")

	second, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", artefacts, time.Unix(200, 0))
	require.NoError(t, err, "a second node with the same digest must not error")
	assert.Equal(t, PublishOutcomeAlreadyPublished, second, "the second node finds the release already published")
	assert.Equal(t, 1, store.refCounts["a-key"], "reference counts must not double when a second node publishes")
}

func TestPublishRelease_DigestConflictIsAnError(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	_, err := PublishRelease(t.Context(), store, "rel-1", "digest-A", []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}, time.Unix(100, 0))
	require.NoError(t, err, "the first build publishes cleanly")

	_, err = PublishRelease(t.Context(), store, "rel-1", "digest-B", []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}, time.Unix(200, 0))
	require.Error(t, err, "a different payload under the same release id must be rejected")
	assert.Contains(t, err.Error(), "different payload", "the error must explain the digest conflict")
}

func TestRetireRelease_DeletesLayers(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	_, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}, time.Unix(100, 0))
	require.NoError(t, err, "publish must succeed before retire")
	require.True(t, store.hasLayer("a", "rel-1"), "the layer must be present before retire")

	require.NoError(t, RetireRelease(t.Context(), store, "rel-1"), "retire must succeed")
	assert.False(t, store.hasLayer("a", "rel-1"), "the layer must be gone after retire")
}

func TestRetireRelease_DropsLeaseAndAllowsReclaim(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	seed := []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}
	_, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", seed, time.Unix(100, 0))
	require.NoError(t, err, "publish must succeed before retire")

	require.NoError(t, RetireRelease(t.Context(), store, "rel-1"), "retire must succeed")
	_, exists := store.leaseState("rel-1")
	assert.False(t, exists, "the lease must be dropped by retire")

	outcome, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", seed, time.Unix(200, 0))
	require.NoError(t, err, "a redeploy after retire must publish cleanly")
	assert.Equal(t, PublishOutcomePublished, outcome, "the redeploy must re-claim and republish, not short-circuit")
	assert.True(t, store.hasLayer("a", "rel-1"), "the republished layer must be present")
}

func TestHeartbeatRelease_AdvancesMonotonically(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	_, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "rel-1")}, time.Unix(100, 0))
	require.NoError(t, err, "publish must succeed before heartbeating")

	require.NoError(t, HeartbeatRelease(t.Context(), store, "rel-1", time.Unix(160, 0)), "a later heartbeat must succeed")
	lease, exists := store.leaseState("rel-1")
	require.True(t, exists, "the lease must exist")
	assert.Equal(t, int64(160), lease.HeartbeatAt, "the heartbeat must advance to the newer value")

	require.NoError(t, HeartbeatRelease(t.Context(), store, "rel-1", time.Unix(130, 0)), "an older heartbeat must not error")
	lease, _ = store.leaseState("rel-1")
	assert.Equal(t, int64(160), lease.HeartbeatAt, "an out-of-order heartbeat must not rewind a fresher one")
}

func TestReapExpiredReleases_Converges(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	base := time.Unix(10_000, 0)
	_, err := PublishRelease(t.Context(), store, "stale", "digest-stale", []*registry_dto.ArtefactMeta{makeReleaseArtefact("b", "stale")}, base.Add(-2*time.Hour))
	require.NoError(t, err, "the stale release publishes")
	store.mu.Lock()
	staleLease := store.leases["stale"]
	staleLease.HeartbeatAt = base.Add(-2 * time.Hour).Unix()
	store.leases["stale"] = staleLease
	store.mu.Unlock()

	retired, err := ReapExpiredReleases(t.Context(), store, "own", 30*time.Minute, base)
	require.NoError(t, err, "the first reap must succeed")
	assert.Equal(t, 1, retired, "the first reap retires the stale release")

	retired, err = ReapExpiredReleases(t.Context(), store, "own", 30*time.Minute, base)
	require.NoError(t, err, "the second reap must succeed")
	assert.Equal(t, 0, retired, "the second reap must retire nothing because the lease is gone")
}

func TestReapExpiredReleases_RetiresStaleNotOwn(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	base := time.Unix(10_000, 0)

	_, err := PublishRelease(t.Context(), store, "own", "digest-own", []*registry_dto.ArtefactMeta{makeReleaseArtefact("a", "own")}, base)
	require.NoError(t, err, "own release publishes")
	_, err = PublishRelease(t.Context(), store, "stale", "digest-stale", []*registry_dto.ArtefactMeta{makeReleaseArtefact("b", "stale")}, base)
	require.NoError(t, err, "stale release publishes")

	store.mu.Lock()
	staleLease := store.leases["stale"]
	staleLease.HeartbeatAt = base.Add(-2 * time.Hour).Unix()
	store.leases["stale"] = staleLease
	store.mu.Unlock()

	retired, err := ReapExpiredReleases(t.Context(), store, "own", 30*time.Minute, base)
	require.NoError(t, err, "reaping must succeed")
	assert.Equal(t, 1, retired, "exactly the stale release is retired")
	assert.False(t, store.hasLayer("b", "stale"), "the stale release's layer must be gone")
	assert.True(t, store.hasLayer("a", "own"), "the caller's own release must never be reaped")
}

func TestSeedDigest_OrderIndependentAndContentSensitive(t *testing.T) {
	t.Parallel()

	a := makeReleaseArtefact("a", "rel-1")
	b := makeReleaseArtefact("b", "rel-1")

	forward := SeedDigest([]*registry_dto.ArtefactMeta{a, b})
	reversed := SeedDigest([]*registry_dto.ArtefactMeta{b, a})
	assert.Equal(t, forward, reversed, "the digest must not depend on artefact order")

	changed := makeReleaseArtefact("a", "rel-1")
	changed.ActualVariants[0].ContentHash = "different"
	divergent := SeedDigest([]*registry_dto.ArtefactMeta{changed, b})
	assert.NotEqual(t, forward, divergent, "a content change must change the digest")
}

func makeUnstampedArtefact(id string) *registry_dto.ArtefactMeta {
	artefact := makeReleaseArtefact(id, "")
	return artefact
}

func TestPublishRelease_StampsReleaseIDOnLayers(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	seed := []*registry_dto.ArtefactMeta{makeUnstampedArtefact("a")}

	outcome, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", seed, time.Unix(100, 0))

	require.NoError(t, err, "publishing an unstamped seed must succeed")
	assert.Equal(t, PublishOutcomePublished, outcome, "the claiming node publishes")
	assert.True(t, store.hasLayer("a", "rel-1"), "the layer must land under the release key")
	assert.False(t, store.hasLayer("a", ""), "no layer may land on the runtime key")
	assert.Equal(t, "", seed[0].ReleaseID, "the caller's seed artefact must not be mutated")
}

func TestPublishRelease_TakesOverStalePublishingLease(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	base := time.Unix(100_000, 0)
	staleClaim := base.Add(-10 * time.Minute)
	won, err := store.ClaimRelease(t.Context(), "rel-1", "digest-1", staleClaim.Unix(), staleClaim.Unix())
	require.NoError(t, err, "seeding the stale publishing lease must succeed")
	require.True(t, won, "the seeded claim must win")

	outcome, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{makeUnstampedArtefact("a")}, base)

	require.NoError(t, err, "the takeover publish must succeed")
	assert.Equal(t, PublishOutcomePublished, outcome, "the taker publishes after deleting the stale lease")
	assert.True(t, store.hasLayer("a", "rel-1"), "the taken-over publish must write the layers")
	lease, exists := store.leaseState("rel-1")
	require.True(t, exists, "a lease must exist after the takeover")
	assert.Equal(t, ReleaseStatePublished, lease.State, "the taken-over lease must end published")
}

func TestPublishRelease_FreshPublishingLeaseIsLeftAlone(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	base := time.Unix(100_000, 0)
	freshClaim := base.Add(-1 * time.Minute)
	won, err := store.ClaimRelease(t.Context(), "rel-1", "digest-1", freshClaim.Unix(), freshClaim.Unix())
	require.NoError(t, err, "seeding the fresh publishing lease must succeed")
	require.True(t, won, "the seeded claim must win")

	outcome, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", []*registry_dto.ArtefactMeta{makeUnstampedArtefact("a")}, base)

	require.NoError(t, err, "deferring to a live publisher must not error")
	assert.Equal(t, PublishOutcomeInProgress, outcome, "a freshly heartbeated publishing lease is left to its owner")
	assert.False(t, store.hasLayer("a", "rel-1"), "no layers may be written while the owner is publishing")
}

func TestPublishLayers_HeartbeatsBetweenLayers(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	won, err := store.ClaimRelease(t.Context(), "rel-1", "digest-1", 100, 100)
	require.NoError(t, err, "the claim must succeed")
	require.True(t, won, "the claim must win")

	seed := []*registry_dto.ArtefactMeta{
		makeUnstampedArtefact("a"),
		makeUnstampedArtefact("b"),
		makeUnstampedArtefact("c"),
	}
	published, err := publishLayers(t.Context(), store, "rel-1", seed, 0, nil, time.Now)

	require.NoError(t, err, "publishing the layers must succeed")
	assert.Equal(t, 3, published, "every layer must insert")
	store.mu.Lock()
	beats := store.heartbeatCalls
	store.mu.Unlock()
	assert.GreaterOrEqual(t, beats, 3, "a zero interval must heartbeat after every layer")
}

func TestRetireRelease_DecrementsBlobReferencesAndEmitsHints(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	seed := []*registry_dto.ArtefactMeta{makeUnstampedArtefact("a")}
	_, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", seed, time.Unix(100, 0))
	require.NoError(t, err, "publish must succeed before retire")
	store.mu.Lock()
	require.Equal(t, 1, store.refCounts["a-key"], "publish must reference the blob once")
	store.mu.Unlock()

	require.NoError(t, RetireRelease(t.Context(), store, "rel-1"), "retire must succeed")

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Equal(t, 0, store.refCounts["a-key"], "retire must release the blob reference")
	require.Len(t, store.gcHints, 1, "a blob that reached zero must be hinted for collection")
	assert.Equal(t, "a-key", store.gcHints[0].StorageKey, "the hint must name the retired blob")
}

func TestRetireRelease_SharedBlobSurvivesWithPositiveCount(t *testing.T) {
	t.Parallel()

	store := newFakeReleaseStore()
	sharedSeed := func() []*registry_dto.ArtefactMeta {
		artefact := makeUnstampedArtefact("a")
		artefact.ActualVariants[0].StorageKey = "shared-key"
		return []*registry_dto.ArtefactMeta{artefact}
	}
	_, err := PublishRelease(t.Context(), store, "rel-1", "digest-1", sharedSeed(), time.Unix(100, 0))
	require.NoError(t, err, "release one publishes")
	_, err = PublishRelease(t.Context(), store, "rel-2", "digest-2", sharedSeed(), time.Unix(200, 0))
	require.NoError(t, err, "release two publishes the same content-addressed blob")
	store.mu.Lock()
	require.Equal(t, 2, store.refCounts["shared-key"], "both releases must reference the shared blob")
	store.mu.Unlock()

	require.NoError(t, RetireRelease(t.Context(), store, "rel-1"), "retiring one release must succeed")

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Equal(t, 1, store.refCounts["shared-key"], "the surviving release must keep its reference")
	assert.Empty(t, store.gcHints, "a blob another release still references must not be hinted")
}
