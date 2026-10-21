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

package orchestrator_adapters

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
)

func emptyOutputArtefact(sourceHash string) *registry_dto.ArtefactMeta {
	return &registry_dto.ArtefactMeta{
		ID: "theme.css",
		ActualVariants: []registry_dto.Variant{{
			VariantID:   "source",
			Status:      registry_dto.VariantStatusReady,
			ContentHash: sourceHash,
		}},
		DesiredProfiles: []registry_dto.NamedProfile{{
			Name: "minified",
			Profile: registry_dto.DesiredProfile{
				CapabilityName: "minify-css",
				DependsOn:      registry_dto.DependenciesFromSlice([]string{"source"}),
			},
		}},
	}
}

func TestEmptyOutputs_BuiltEmptyOnlyForTheSourceItWasBuiltFrom(t *testing.T) {
	t.Parallel()

	record := NewEmptyOutputs()
	assert.False(t, record.BuiltEmpty(emptyOutputArtefact("hash-1"), "minified", "source"), "nothing recorded yet")

	record.Record("theme.css", "minified", "hash-1")
	assert.True(t, record.BuiltEmpty(emptyOutputArtefact("hash-1"), "minified", "source"))
	assert.False(t, record.BuiltEmpty(emptyOutputArtefact("hash-2"), "minified", "source"),
		"a changed source must be built again")
	assert.False(t, record.BuiltEmpty(emptyOutputArtefact("hash-1"), "gzip", "source"), "only the recorded profile")

	var unset *EmptyOutputs
	unset.Record("theme.css", "minified", "hash-1")
	assert.False(t, unset.BuiltEmpty(emptyOutputArtefact("hash-1"), "minified", "source"), "a nil record is inert")
}

func TestBridge_ProfileBuiltEmptyIsNeitherUnbuiltNorDispatchedAgain(t *testing.T) {
	t.Parallel()

	artefact := emptyOutputArtefact("hash-1")
	registry := &registry_domain.MockRegistryService{
		GetArtefactFunc: func(context.Context, string) (*registry_dto.ArtefactMeta, error) {
			return artefact, nil
		},
		GetMultipleArtefactsFunc: func(context.Context, []string) ([]*registry_dto.ArtefactMeta, error) {
			return []*registry_dto.ArtefactMeta{artefact}, nil
		},
	}
	dispatcher := &mockTaskDispatcher{}
	record := NewEmptyOutputs()
	bridge := NewArtefactWorkflowBridge(registry, nil, dispatcher, nil, WithBridgeEmptyOutputs(record))

	unbuilt, err := bridge.UnbuiltProfiles(t.Context(), []string{"theme.css"})
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{"theme.css": {"minified"}}, unbuilt, "before it is built the profile is missing")

	record.Record("theme.css", "minified", "hash-1")
	unbuilt, err = bridge.UnbuiltProfiles(t.Context(), []string{"theme.css"})
	require.NoError(t, err)
	assert.Empty(t, unbuilt, "a profile built with no output is built")

	dispatched, err := bridge.ReconcileArtefacts(t.Context(), []string{"theme.css"})
	require.NoError(t, err)
	assert.Zero(t, dispatched, "reconciliation does not build it again")
	assert.Empty(t, dispatcher.dispatched)

	needed, err := bridge.profileDispatchRequirement("theme.css", "minified")(t.Context())
	require.NoError(t, err)
	assert.False(t, needed, "a dispatch already under way is released as not required")
}
