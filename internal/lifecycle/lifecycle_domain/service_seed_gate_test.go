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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/lifecycle/lifecycle_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestInitialSeed_SkipsWritesWhenRegistryBlobsReadOnly(t *testing.T) {
	t.Parallel()

	registry := &mockTrackingRegistryService{}
	ls := &lifecycleService{
		registryBlobsReadOnly: true,
		registryService:       registry,
	}

	ls.upsertAssetArtefact(fileEventContext{
		ctx:        t.Context(),
		relPath:    "assets/icons/search.svg",
		artefactID: "example.com/assets/icons/search.svg",
		event: lifecycle_dto.FileEvent{
			Path: "/test/assets/icons/search.svg",
			Type: lifecycle_dto.FileEventTypeCreate,
		},
	})

	assert.NoError(t, ls.seedThemeArtefact(t.Context()),
		"theme re-seed must be a clean no-op when registry blobs are read-only")
	assert.NoError(t, ls.seedCaptchaInitScripts(t.Context()),
		"captcha re-seed must be a clean no-op when registry blobs are read-only")
	assert.Empty(t, registry.upsertedArtefacts,
		"read-only registry blobs must skip every initial-seed artefact write and serve the baked-in base")
}

func TestSeedExternalComponentFiles_SkippedWhenRegistryBlobsReadOnly(t *testing.T) {
	t.Parallel()

	resolverCalled := false
	resolver := &resolver_domain.MockResolver{
		FindModuleBoundaryFunc: func(_ context.Context, _ string) (string, string, error) {
			resolverCalled = true
			return "", "", errors.New("resolver must not run when registry blobs are read-only")
		},
	}
	ls := &lifecycleService{
		registryBlobsReadOnly: true,
		resolver:              resolver,
		externalComponents: []component_dto.ComponentDefinition{
			{ModulePath: "piko.sh/piko/components", AssetPaths: []string{"icons"}},
		},
	}

	ls.seedExternalComponentFiles(t.Context(), make(chan struct{}, 1))

	assert.False(t, resolverCalled,
		"read-only registry blobs must skip the external component and asset seed before resolving modules")
}
