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

package orchestrator_pipeline_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestRestart_RebuildsProfileWhoseTaskWasLost(t *testing.T) {
	metaStore, err := registry_otter.NewOtterDAL(registry_otter.Config{Capacity: 10000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = metaStore.Close() })
	blobDir := t.TempDir()

	firstExecutor := newControllableExecutor()
	first := newPipelineHarness(t,
		withExecutor("artefact.compiler", firstExecutor),
		withRegistryStorage(metaStore, blobDir),
		withDeferredDispatcherStart(),
	)
	first.seedArtefact("lost.css", []registry_dto.NamedProfile{makeProfile("minified", "minify-css")})
	require.True(t, waitForCondition(5*time.Second, func() bool {
		return first.bridge.ArtefactEventsProcessed() >= 1
	}), "the first process handles the artefact's event")
	require.Equal(t, int64(1), first.dispatcher.Stats().TasksDispatched, "and dispatches the profile's task")
	first.stop()
	assert.Zero(t, firstExecutor.getCallCount(),
		"the task never ran: the first process stopped before its workers started")

	executor := newControllableExecutor()
	second := newPipelineHarness(t,
		withExecutor("artefact.compiler", executor),
		withRegistryStorage(metaStore, blobDir),
	)
	assert.Zero(t, second.dispatcher.Stats().TasksDispatched, "the second process starts with an empty task store")

	dispatched, err := lifecycle_domain.ReconcileUnbuiltProfiles(second.ctx, second.registryService, second.bridge)
	require.NoError(t, err)
	assert.Equal(t, 1, dispatched, "startup finds the profile the first process never built")

	require.True(t, waitForCondition(10*time.Second, func() bool { return executor.getCallCount() == 1 }),
		"the lost profile is built")
	executor.mu.Lock()
	payload := executor.calls[0].Payload
	executor.mu.Unlock()
	assert.Equal(t, "lost.css", payload["artefactID"])
	assert.Equal(t, "minified", payload["desiredProfileName"])
	assert.True(t, second.waitForBuildComplete(10*time.Second), "and its task settles")
}

func TestRestart_LeavesBuiltProfilesAlone(t *testing.T) {
	metaStore, err := registry_otter.NewOtterDAL(registry_otter.Config{Capacity: 10000})
	require.NoError(t, err)
	t.Cleanup(func() { _ = metaStore.Close() })
	blobDir := t.TempDir()

	first := newPipelineHarness(t, withRegistryStorage(metaStore, blobDir))
	first.seedArtefact("plain.txt", nil)
	require.True(t, waitForCondition(5*time.Second, func() bool {
		return first.bridge.ArtefactEventsProcessed() >= 1
	}))
	first.stop()

	executor := newControllableExecutor()
	second := newPipelineHarness(t,
		withExecutor("artefact.compiler", executor),
		withRegistryStorage(metaStore, blobDir),
	)
	dispatched, err := lifecycle_domain.ReconcileUnbuiltProfiles(second.ctx, second.registryService, second.bridge)
	require.NoError(t, err)
	assert.Zero(t, dispatched)
	assert.Zero(t, executor.getCallCount())
}
