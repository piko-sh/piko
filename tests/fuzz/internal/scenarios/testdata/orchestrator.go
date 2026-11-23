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
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	registry_otter "piko.sh/piko/internal/registry/registry_dal/otter"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestGeneratedPikoScenario(t *testing.T) {
	token := os.Getenv("PIKO_FUZZ_CASE_TOKEN")
	draws, err := hex.DecodeString(token)
	require.NoError(t, err)
	require.Len(t, draws, 8)
	harnessExecutor := newControllableExecutor()
	harness := newPipelineHarness(t, withMaxRetries(2), withExecutor("artefact.compiler", harnessExecutor))
	completed, failed, retries, calls := 0, 0, 0, 0
	actions := []byte{0, 1, 2}
	actions = append(actions, draws[:1+int(draws[0]%3)]...)
	for i, b := range actions {
		failures := int(b % 3)
		harnessExecutor.mu.Lock()
		harnessExecutor.failUntil = calls + failures
		harnessExecutor.mu.Unlock()
		name := fmt.Sprintf("item-%s-%d", token, i)
		harness.seedArtefact(name, []registry_dto.NamedProfile{makeProfile(name, "compile-component")})
		require.True(t, harness.waitForFlush(5*time.Second))
		require.True(t, harness.advanceUntilIdle(7*time.Second))
		attempts := min(failures+1, 2)
		calls += attempts
		retries += attempts - 1
		if failures == 2 {
			failed++
		} else {
			completed++
		}
		stats := harness.dispatcher.Stats()
		require.EqualValues(t, completed, stats.TasksCompleted, "PIKO_CHECK orchestration-completed: step %d", i)
		require.EqualValues(t, failed, stats.TasksFailed, "PIKO_CHECK orchestration-failed: step %d", i)
		require.EqualValues(t, retries, stats.TasksRetried, "PIKO_CHECK orchestration-retries: step %d", i)
		require.Equal(t, calls, harnessExecutor.getCallCount(), "PIKO_CHECK orchestration-executed: step %d", i)
		require.GreaterOrEqual(t, harness.bridge.ArtefactEventsProcessed(), harness.registryService.ArtefactEventsIssued())
	}
	t.Run("restart", func(t *testing.T) {
		store, err := registry_otter.NewOtterDAL(registry_otter.Config{Capacity: 128})
		require.NoError(t, err)
		t.Cleanup(func() { _ = store.Close() })
		dir := t.TempDir()
		old := newControllableExecutor()
		first := newPipelineHarness(t, withRegistryStorage(store, dir), withDeferredDispatcherStart(), withExecutor("artefact.compiler", old))
		expected := map[string]string{}
		for i := range 1 + int(draws[1]%3) {
			name, profile := fmt.Sprintf("restart-%d-%s.css", i, token), fmt.Sprintf("style-%d", i)
			expected[name] = profile
			first.seedArtefact(name, []registry_dto.NamedProfile{makeProfile(profile, "minify-css")})
		}
		first.seedArtefact("static-"+token+".txt", nil)
		require.True(t, waitForCondition(5*time.Second, func() bool { return first.bridge.ArtefactEventsProcessed() >= int64(len(expected)+1) }))
		require.EqualValues(t, len(expected), first.dispatcher.Stats().TasksDispatched, "PIKO_CHECK dispatch-before-stop")
		first.stop()
		require.Zero(t, old.getCallCount(), "PIKO_CHECK unstarted-dispatch")
		live := newControllableExecutor()
		second := newPipelineHarness(t, withRegistryStorage(store, dir), withExecutor("artefact.compiler", live))
		dispatched, err := lifecycle_domain.ReconcileUnbuiltProfiles(second.ctx, second.registryService, second.bridge)
		require.NoError(t, err)
		require.Equal(t, len(expected), dispatched, "PIKO_CHECK reconcile-count")
		require.True(t, second.waitForBuildComplete(10*time.Second))
		live.mu.Lock()
		defer live.mu.Unlock()
		actual := map[string]string{}
		for _, task := range live.calls {
			id, _ := task.Payload["artefactID"].(string)
			profile, _ := task.Payload["desiredProfileName"].(string)
			actual[id] = profile
		}
		require.Len(t, live.calls, len(expected), "PIKO_CHECK recovered-once")
		require.Equal(t, expected, actual, "PIKO_CHECK recovered-profile-identity")
	})
}
