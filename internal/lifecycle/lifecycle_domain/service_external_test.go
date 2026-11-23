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

package lifecycle_domain

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type seedRecorder struct {
	err     error
	seeded  map[string]string
	mu      sync.Mutex
	attempt int
}

func (r *seedRecorder) registry() *registry_domain.MockRegistryService {
	return &registry_domain.MockRegistryService{
		UpsertArtefactFunc: func(
			_ context.Context,
			artefactID, sourcePath string,
			sourceData io.Reader,
			_ string,
			_ []registry_dto.NamedProfile,
		) (*registry_dto.ArtefactMeta, error) {
			content, readErr := io.ReadAll(sourceData)
			r.mu.Lock()
			defer r.mu.Unlock()
			r.attempt++
			if readErr != nil {
				return nil, readErr
			}
			if r.err != nil {
				return nil, r.err
			}
			r.seeded[artefactID] = sourcePath + "=" + string(content)
			return &registry_dto.ArtefactMeta{}, nil
		},
	}
}

func newExternalSeedService(t *testing.T, recorder *seedRecorder, moduleDirErr error) *lifecycleService {
	t.Helper()
	fileSystem := NewMockFileSystem()
	fileSystem.AddFile("/modules/lib/components/card.pkc", []byte("card"))
	fileSystem.AddFile("/modules/lib/components/nested/badge.PKC", []byte("badge"))
	fileSystem.AddFile("/modules/lib/components/readme.md", []byte("ignored"))
	fileSystem.AddFile("/modules/lib/icons/star.svg", []byte("star"))
	fileSystem.AddFile("/modules/lib/icons/solid/moon.svg", []byte("moon"))

	resolver := &resolver_domain.MockResolver{
		FindModuleBoundaryFunc: func(_ context.Context, importPath string) (string, string, error) {
			if !strings.HasPrefix(importPath, "example.org/lib/") {
				return "", "", errors.New("unknown module")
			}
			return "example.org/lib", strings.TrimPrefix(importPath, "example.org/lib/"), nil
		},
		GetModuleDirFunc: func(_ context.Context, _ string) (string, error) {
			if moduleDirErr != nil {
				return "", moduleDirErr
			}
			return "/modules/lib", nil
		},
	}
	return &lifecycleService{
		fs:              fileSystem,
		resolver:        resolver,
		registryService: recorder.registry(),
		externalComponents: []component_dto.ComponentDefinition{
			{ModulePath: "example.org/lib/components", AssetPaths: []string{"icons", "icons"}},
			{ModulePath: "example.org/lib/components"},
			{ModulePath: "example.org/lib/missing"},
			{ModulePath: "other.org/unknown/components", AssetPaths: []string{"icons"}},
			{ModulePath: ""},
		},
	}
}

func TestSeedExternalComponentFilesSeedsComponentsAndAssets(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}}
	service := newExternalSeedService(t, recorder, nil)

	service.seedExternalComponentFiles(t.Context(), make(chan struct{}, 2))

	assert.Equal(t, map[string]string{
		"example.org/lib/components/card.pkc":         "card.pkc=card",
		"example.org/lib/components/nested/badge.PKC": "nested/badge.PKC=badge",
		"example.org/lib/icons/star.svg":              "star.svg=star",
		"example.org/lib/icons/solid/moon.svg":        "solid/moon.svg=moon",
	}, recorder.seeded)
}

func TestSeedExternalComponentFilesKeepsGoingWhenAnUpsertFails(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}, err: errors.New("registry unavailable")}
	service := newExternalSeedService(t, recorder, nil)

	service.seedExternalComponentFiles(t.Context(), make(chan struct{}, 1))

	assert.Empty(t, recorder.seeded)
	assert.Equal(t, 4, recorder.attempt, "every file is attempted even when the registry refuses them")
}

func TestSeedExternalComponentFilesSkipsUnresolvableModules(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}}
	service := newExternalSeedService(t, recorder, errors.New("module not downloaded"))

	service.seedExternalComponentFiles(t.Context(), make(chan struct{}, 1))

	assert.Empty(t, recorder.seeded)
	assert.Zero(t, recorder.attempt)
}

func TestSeedExternalComponentFilesStopsWhenCancelled(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}}
	service := newExternalSeedService(t, recorder, nil)
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("shutting down"))

	service.seedExternalComponentFiles(ctx, make(chan struct{}, 1))

	assert.Zero(t, recorder.attempt)
}

func TestSeedExternalComponentFilesSkipsReadOnlyRegistryBlobs(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}}
	service := newExternalSeedService(t, recorder, nil)
	service.registryBlobsReadOnly = true

	service.seedExternalComponentFiles(t.Context(), make(chan struct{}, 1))

	assert.Zero(t, recorder.attempt)
}

func TestWalkAndSeedExternalDirsReportMissingDirectories(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}}
	service := newExternalSeedService(t, recorder, nil)

	service.walkAndSeedExternalDir(t.Context(), "/modules/absent", "example.org/lib/absent", make(chan struct{}, 1))
	service.walkAndSeedExternalAssetDir(t.Context(), "/modules/absent", "example.org/lib/absent", make(chan struct{}, 1))

	assert.Zero(t, recorder.attempt)
}

func TestSeedArtefactAsyncReportsAFileThatCannotBeOpened(t *testing.T) {
	t.Parallel()

	recorder := &seedRecorder{seeded: map[string]string{}}
	service := newExternalSeedService(t, recorder, nil)
	limiter := make(chan struct{}, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	limiter <- struct{}{}
	service.seedArtefactAsync(t.Context(), &wg, limiter, "/modules/lib/missing.pkc", "example.org/lib/missing.pkc", "missing.pkc", "component")
	wg.Wait()

	require.Empty(t, limiter, "the limiter slot is released")
	assert.Zero(t, recorder.attempt)
}
