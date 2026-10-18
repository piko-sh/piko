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

package registry_domain_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func namedProfile(name, capability string) registry_dto.NamedProfile {
	return registry_dto.NamedProfile{
		Name:    name,
		Profile: registry_dto.DesiredProfile{CapabilityName: capability},
	}
}

func profileNames(profiles []registry_dto.NamedProfile) []string {
	names := make([]string, len(profiles))
	for i := range profiles {
		names[i] = profiles[i].Name
	}
	return names
}

func TestMergeDesiredProfiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		existing []registry_dto.NamedProfile
		incoming []registry_dto.NamedProfile
		want     []registry_dto.NamedProfile
	}{
		{
			name:     "no existing profiles takes the incoming ones",
			existing: nil,
			incoming: []registry_dto.NamedProfile{namedProfile("gzip", "compress-gzip")},
			want:     []registry_dto.NamedProfile{namedProfile("gzip", "compress-gzip")},
		},
		{
			name:     "no incoming profiles keeps the existing ones",
			existing: []registry_dto.NamedProfile{namedProfile("placeholder", "image")},
			incoming: nil,
			want:     []registry_dto.NamedProfile{namedProfile("placeholder", "image")},
		},
		{
			name:     "both empty stays empty",
			existing: nil,
			incoming: nil,
			want:     nil,
		},
		{
			name:     "disjoint profiles are appended after the existing ones",
			existing: []registry_dto.NamedProfile{namedProfile("placeholder", "image"), namedProfile("image_w640_webp", "image")},
			incoming: []registry_dto.NamedProfile{namedProfile("gzip", "compress-gzip"), namedProfile("br", "compress-br")},
			want: []registry_dto.NamedProfile{
				namedProfile("placeholder", "image"),
				namedProfile("image_w640_webp", "image"),
				namedProfile("gzip", "compress-gzip"),
				namedProfile("br", "compress-br"),
			},
		},
		{
			name:     "an existing profile keeps its definition when the incoming set names it",
			existing: []registry_dto.NamedProfile{namedProfile("minified", "annotation-minify")},
			incoming: []registry_dto.NamedProfile{namedProfile("minified", "walk-minify"), namedProfile("gzip", "compress-gzip")},
			want:     []registry_dto.NamedProfile{namedProfile("minified", "annotation-minify"), namedProfile("gzip", "compress-gzip")},
		},
		{
			name:     "a name repeated in the incoming set is added once",
			existing: []registry_dto.NamedProfile{namedProfile("placeholder", "image")},
			incoming: []registry_dto.NamedProfile{namedProfile("gzip", "first"), namedProfile("gzip", "second")},
			want:     []registry_dto.NamedProfile{namedProfile("placeholder", "image"), namedProfile("gzip", "first")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, registry_domain.MergeDesiredProfiles(tc.existing, tc.incoming))
		})
	}

	t.Run("a merge that adds nothing returns the existing slice without copying", func(t *testing.T) {
		t.Parallel()

		existing := []registry_dto.NamedProfile{namedProfile("placeholder", "image"), namedProfile("gzip", "compress-gzip")}
		merged := registry_domain.MergeDesiredProfiles(existing, []registry_dto.NamedProfile{namedProfile("gzip", "other")})

		require.Len(t, merged, len(existing))
		assert.Same(t, &existing[0], &merged[0])
	})

	t.Run("a merge that adds a profile does not write into the existing slice", func(t *testing.T) {
		t.Parallel()

		backing := make([]registry_dto.NamedProfile, 1, 4)
		backing[0] = namedProfile("placeholder", "image")
		merged := registry_domain.MergeDesiredProfiles(backing, []registry_dto.NamedProfile{namedProfile("gzip", "compress-gzip")})

		assert.Equal(t, []string{"placeholder", "gzip"}, profileNames(merged))
		assert.Empty(t, backing[1:cap(backing)][0].Name, "the existing slice's spare capacity must stay untouched")
	})
}

func TestUpsertMergesProfiles(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts []registry_domain.UpsertOption
		want bool
	}{
		{name: "no options", opts: nil, want: false},
		{name: "nil option is ignored", opts: []registry_domain.UpsertOption{nil}, want: false},
		{name: "profile merge", opts: []registry_domain.UpsertOption{registry_domain.WithProfileMerge()}, want: true},
		{name: "profile merge among nil options", opts: []registry_domain.UpsertOption{nil, registry_domain.WithProfileMerge()}, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, registry_domain.UpsertMergesProfiles(tc.opts...))
		})
	}
}

type statefulUpsertFixture struct {
	testFixture
	stored *registry_dto.ArtefactMeta
	mu     sync.Mutex
}

func newStatefulUpsertFixture() *statefulUpsertFixture {
	f := &statefulUpsertFixture{testFixture: setupTest()}
	wireNewArtefactUpsertMocks(&f.testFixture)
	f.metaStore.GetArtefactFunc = func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.stored == nil {
			return nil, registry_domain.ErrArtefactNotFound
		}
		return f.stored.Clone(), nil
	}
	f.metaStore.AtomicUpdateFunc = func(_ context.Context, actions []registry_dto.AtomicAction) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, action := range actions {
			if action.Type == registry_dto.ActionTypeUpsertArtefact {
				f.stored = action.Artefact.Clone()
			}
		}
		return nil
	}
	return f
}

func (f *statefulUpsertFixture) storedArtefact(t *testing.T) *registry_dto.ArtefactMeta {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotNil(t, f.stored, "an artefact should have been persisted")
	return f.stored.Clone()
}

func TestRegistryService_UpsertArtefact_ProfileMerge(t *testing.T) {
	const (
		artefactID       = "example.com/site/lib/photo.png"
		sourcePath       = "lib/photo.png"
		storageBackendID = "local_disk_cache"
		sourceBytes      = "png bytes"
	)
	annotationProfiles := []registry_dto.NamedProfile{
		namedProfile("image_w640_webp", "image"),
		namedProfile("placeholder", "image"),
	}
	walkProfiles := []registry_dto.NamedProfile{
		namedProfile("gzip", "compress-gzip"),
		namedProfile("br", "compress-br"),
	}

	t.Run("walk upsert after an annotation upsert keeps the annotation's profiles", func(t *testing.T) {
		f := newStatefulUpsertFixture()

		_, err := f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID, annotationProfiles)
		require.NoError(t, err)

		walked, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath,
			strings.NewReader(sourceBytes), storageBackendID, walkProfiles, registry_domain.WithProfileMerge())
		require.NoError(t, err)

		wantNames := []string{"image_w640_webp", "placeholder", "gzip", "br"}
		assert.Equal(t, wantNames, profileNames(walked.DesiredProfiles))

		stored := f.storedArtefact(t)
		assert.Equal(t, wantNames, profileNames(stored.DesiredProfiles))
		require.Len(t, stored.ActualVariants, 1)
		assert.Equal(t, "source", stored.ActualVariants[0].VariantID, "the walk still stores the source bytes")
		assert.Equal(t, sourcePath, stored.SourcePath)
	})

	t.Run("walk upsert without the option replaces the annotation's profiles", func(t *testing.T) {
		f := newStatefulUpsertFixture()

		_, err := f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID, annotationProfiles)
		require.NoError(t, err)

		_, err = f.service.UpsertArtefact(f.testContext, artefactID, sourcePath,
			strings.NewReader(sourceBytes), storageBackendID, walkProfiles)
		require.NoError(t, err)

		assert.Equal(t, []string{"gzip", "br"}, profileNames(f.storedArtefact(t).DesiredProfiles))
	})

	t.Run("annotation upsert after a merged walk upsert still decides the profile set", func(t *testing.T) {
		f := newStatefulUpsertFixture()

		_, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath,
			strings.NewReader(sourceBytes), storageBackendID, walkProfiles, registry_domain.WithProfileMerge())
		require.NoError(t, err)

		_, err = f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID, annotationProfiles)
		require.NoError(t, err)

		stored := f.storedArtefact(t)
		assert.Equal(t, []string{"image_w640_webp", "placeholder"}, profileNames(stored.DesiredProfiles))
		require.Len(t, stored.ActualVariants, 1, "a metadata-only upsert keeps the source variant")
	})

	t.Run("merge on a new artefact takes the given profiles", func(t *testing.T) {
		f := newStatefulUpsertFixture()

		created, err := f.service.UpsertArtefact(f.testContext, artefactID, sourcePath,
			strings.NewReader(sourceBytes), storageBackendID, walkProfiles, registry_domain.WithProfileMerge())
		require.NoError(t, err)

		assert.Equal(t, []string{"gzip", "br"}, profileNames(created.DesiredProfiles))
	})

	t.Run("metadata-only merge that adds nothing is skipped", func(t *testing.T) {
		f := newStatefulUpsertFixture()

		_, err := f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID, annotationProfiles)
		require.NoError(t, err)
		writesBefore := f.metaStore.AtomicUpdateCallCount.Load()
		eventsBefore := f.eventBus.PublishCallCount.Load()

		_, err = f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID,
			[]registry_dto.NamedProfile{namedProfile("placeholder", "image")}, registry_domain.WithProfileMerge())
		require.NoError(t, err)

		assert.Equal(t, writesBefore, f.metaStore.AtomicUpdateCallCount.Load(), "nothing changed, so nothing is written")
		assert.Equal(t, eventsBefore, f.eventBus.PublishCallCount.Load(), "nothing changed, so no event is published")
		assert.Equal(t, []string{"image_w640_webp", "placeholder"}, profileNames(f.storedArtefact(t).DesiredProfiles))
	})

	t.Run("metadata-only merge that adds a profile is persisted", func(t *testing.T) {
		f := newStatefulUpsertFixture()

		_, err := f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID, annotationProfiles)
		require.NoError(t, err)

		_, err = f.service.UpsertArtefact(f.testContext, artefactID, artefactID, nil, storageBackendID,
			walkProfiles[:1], registry_domain.WithProfileMerge())
		require.NoError(t, err)

		assert.Equal(t, []string{"image_w640_webp", "placeholder", "gzip"}, profileNames(f.storedArtefact(t).DesiredProfiles))
	})
}
