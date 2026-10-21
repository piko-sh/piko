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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	clockpkg "piko.sh/piko/wdk/clock"
)

type slowRegistry struct {
	registry_domain.RegistryService
	delay time.Duration
}

func (r slowRegistry) GetArtefact(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	time.Sleep(r.delay)
	return r.RegistryService.GetArtefact(ctx, artefactID)
}

func TestBuildCompletion_BridgeLatencyIsNeverPremature(t *testing.T) {
	for run := range 100 {
		t.Run(fmt.Sprintf("run=%03d", run), func(t *testing.T) {
			h := newPipelineHarness(t,
				withProductionConfig(),
				withClock(clockpkg.RealClock()),
				withBridgeRegistry(func(inner registry_domain.RegistryService) registry_domain.RegistryService {
					return slowRegistry{RegistryService: inner, delay: 2 * time.Millisecond}
				}),
			)
			exec := newCascadingExecutor(h.registryService)
			h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

			h.seedArtefact("chain.pkc", pkcProfiles("chain-component"))
			require.True(t, h.waitForBuildComplete(15*time.Second), "the build completes")

			artefact, err := h.registryService.GetArtefact(h.ctx, "chain.pkc")
			require.NoError(t, err)
			ready := 0
			for _, variant := range artefact.ActualVariants {
				if variant.VariantID != "source" && variant.Status == registry_dto.VariantStatusReady {
					ready++
				}
			}
			assert.Equal(t, 4, ready, "every profile in the chain is built before the build completes")
		})
	}
}

func TestBuildCompletion_PendingRetryKeepsBuildOpen(t *testing.T) {
	exec := newControllableExecutor()
	exec.failUntil = 1
	h := newPipelineHarness(t, withExecutor("flaky", exec))

	baseline := h.clock.TimerCount()
	require.NoError(t, h.dispatchTask("flaky", 1))
	require.Eventually(t, func() bool { return h.dispatcher.Stats().TasksRetried == 1 }, 5*time.Second, 5*time.Millisecond)
	require.True(t, h.clock.AwaitTimerSetup(baseline, 5*time.Second), "the retry is scheduled")

	for range 20 {
		assert.False(t, h.buildComplete(), "a task waiting for its retry keeps the build open")
		time.Sleep(5 * time.Millisecond)
	}

	h.clock.Advance(12 * time.Second)
	require.True(t, h.waitForBuildComplete(10*time.Second), "the build completes once the retry settles")
	assert.Equal(t, 2, exec.getCallCount())
}
