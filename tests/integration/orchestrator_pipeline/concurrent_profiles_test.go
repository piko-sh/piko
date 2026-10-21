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

package orchestrator_pipeline_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/registry/registry_dto"
)

type profileCapabilities struct{}

func (profileCapabilities) Register(string, capabilities_domain.CapabilityFunc) error { return nil }

func (profileCapabilities) Execute(_ context.Context, name string, input io.Reader, _ capabilities_domain.CapabilityParams) (io.Reader, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Duration(rand.IntN(3)) * time.Millisecond)
	return bytes.NewReader(append([]byte(name+":"), data...)), nil
}

var sharedCapabilities = []string{
	string(capabilities_dto.CapabilityMinifyJS),
	string(capabilities_dto.CapabilityCopyJS),
}

func withNormalWorkers(workers int) harnessOption {
	return func(c *harnessConfig) {
		c.dispatcherConfig.WatermillNormalHandlers = workers
	}
}

func TestConcurrentProfilesOfOneArtefactAllPersist(t *testing.T) {
	const artefacts, profiles = 12, 4
	h := newPipelineHarness(t, withNormalWorkers(8))
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler",
		orchestrator_adapters.NewCompilerExecutor(h.registryService, profileCapabilities{}))

	profileList := make([]registry_dto.NamedProfile, profiles)
	for i := range profileList {
		profileList[i] = registry_dto.NamedProfile{
			Name: fmt.Sprintf("profile-%d", i),
			Profile: registry_dto.DesiredProfile{
				CapabilityName: sharedCapabilities[i%len(sharedCapabilities)],
				DependsOn:      registry_dto.DependenciesFromSlice([]string{"source"}),
				ResultingTags: registry_dto.TagsFromMap(map[string]string{
					"storageBackendId": "test_store",
					"fileExtension":    ".out",
					"mimeType":         "text/plain",
				}),
			},
		}
	}
	for i := range artefacts {
		h.seedArtefactWithContent(fmt.Sprintf("assets/concurrent-%d.txt", i), []byte(fmt.Sprintf("source %d", i)), profileList)
	}

	flushed, idle := h.waitUntilIdle(10*time.Second, 30*time.Second)
	require.True(t, flushed)
	require.True(t, idle)
	assert.Zero(t, h.dispatcher.Stats().TasksFailed)

	blobStore, err := h.registryService.GetBlobStore("test_store")
	require.NoError(t, err)
	for i := range artefacts {
		artefact, err := h.registryService.GetArtefact(h.ctx, fmt.Sprintf("assets/concurrent-%d.txt", i))
		require.NoError(t, err)
		variants := make(map[string]registry_dto.Variant, len(artefact.ActualVariants))
		for _, variant := range artefact.ActualVariants {
			variants[variant.VariantID] = variant
		}
		for _, profile := range profileList {
			variant, ok := variants[profile.Name]
			if !assert.True(t, ok, "artefact %d lost its %s variant", i, profile.Name) {
				continue
			}
			exists, err := blobStore.Exists(h.ctx, variant.StorageKey)
			require.NoError(t, err)
			assert.True(t, exists, "artefact %d variant %s has no blob at %s", i, profile.Name, variant.StorageKey)
		}
	}
}
