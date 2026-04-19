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

package bootstrap

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

type releaseOverlayFake struct {
	*registry_domain.MockMetadataStore
	layerDeletes atomic.Int64
	leaseDeletes atomic.Int64
}

func newReleaseOverlayFake() *releaseOverlayFake {
	fake := &releaseOverlayFake{MockMetadataStore: &registry_domain.MockMetadataStore{}}
	fake.RunAtomicFunc = func(ctx context.Context, fn func(ctx context.Context, transactionStore registry_domain.MetadataStore) error) error {
		return fn(ctx, fake)
	}
	return fake
}

var (
	_ registry_domain.ReleasePublisher = (*releaseOverlayFake)(nil)
)

func (f *releaseOverlayFake) InsertArtefactLayerIfAbsent(_ context.Context, _ *registry_dto.ArtefactMeta) (bool, error) {
	return false, nil
}

func (f *releaseOverlayFake) DeleteArtefactLayersForRelease(_ context.Context, _ string) error {
	f.layerDeletes.Add(1)
	return nil
}

func (f *releaseOverlayFake) ReclaimArtefactLayersForRelease(_ context.Context, _ string) ([]*registry_dto.ArtefactMeta, error) {
	f.layerDeletes.Add(1)
	return nil, nil
}

func (f *releaseOverlayFake) ClaimRelease(_ context.Context, _, _ string, _, _ int64) (bool, error) {
	return false, nil
}

func (f *releaseOverlayFake) GetRelease(_ context.Context, _ string) (registry_domain.ReleaseLease, bool, error) {
	return registry_domain.ReleaseLease{}, false, nil
}

func (f *releaseOverlayFake) MarkReleasePublished(_ context.Context, _ string, _, _ int64) error {
	return nil
}

func (f *releaseOverlayFake) HeartbeatRelease(_ context.Context, _ string, _ int64) error {
	return nil
}

func (f *releaseOverlayFake) ListExpiredReleases(_ context.Context, _ int64, _ string) ([]string, error) {
	return nil, nil
}

func (f *releaseOverlayFake) DeleteReleaseLease(_ context.Context, _ string) error {
	f.leaseDeletes.Add(1)
	return nil
}

func (f *releaseOverlayFake) DeleteStalePublishingLease(_ context.Context, _ string, _ int64) error {
	return nil
}

func TestRetireRegistryRelease_UsesReleaseOverlay(t *testing.T) {
	t.Parallel()

	overlay := newReleaseOverlayFake()
	container := NewContainer(WithRegistryService(&registry_domain.MockRegistryService{}))
	container.registryMetaStore = &registry_domain.MockMetadataStore{}
	container.registryReleaseOverlay = overlay

	require.NoError(t, container.RetireRegistryRelease(t.Context(), "rel-1"), "retire must succeed")
	assert.Equal(t, int64(1), overlay.layerDeletes.Load(), "the retire must delete layers on the OVERLAY, not the composed meta store")
	assert.Equal(t, int64(1), overlay.leaseDeletes.Load(), "the retire must drop the lease on the overlay")
}

func TestShouldStartReleaseHeartbeat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		outcome registry_domain.PublishOutcome
		want    bool
	}{
		{registry_domain.PublishOutcomePublished, true},
		{registry_domain.PublishOutcomeAlreadyPublished, true},
		{registry_domain.PublishOutcomeInProgress, false},
		{registry_domain.PublishOutcomeUnsupported, false},
	}
	for _, testCase := range cases {
		assert.Equal(t, testCase.want, shouldStartReleaseHeartbeat(testCase.outcome),
			"outcome %s must gate the heartbeat correctly", testCase.outcome)
	}
}
