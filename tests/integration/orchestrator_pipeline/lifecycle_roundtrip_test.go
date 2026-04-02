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
	"piko.sh/piko/internal/capabilities"
	"piko.sh/piko/internal/compiler/compiler_adapters"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/registry/registry_dto"
	clockpkg "piko.sh/piko/wdk/clock"
)

func TestLifecycle_TwoPhaseDetection_Success(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("lifecycle-ok", []registry_dto.NamedProfile{
		makeProfile("web", "compile-component"),
	})

	flushed, idle := h.waitUntilIdle(5*time.Second, 5*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete")
	require.True(t, idle, "Phase 2 (idle) should complete")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) should be >= published (%d)", processed, published)
}

func TestLifecycle_TwoPhaseDetection_Failure(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compilation error: duplicate declaration"

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("lifecycle-fail", []registry_dto.NamedProfile{
		makeProfile("web", "compile-component"),
	})

	flushed, idle := h.waitUntilIdle(5*time.Second, 10*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete even when tasks fail")

	if !idle {
		idle = h.advanceUntilIdle(10 * time.Second)
	}
	require.True(t, idle, "Phase 2 (idle) should complete after retries exhausted")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one task failed permanently")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no tasks completed")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) should be >= published (%d)", processed, published)
}

func TestLifecycle_TwoPhaseDetection_FailureRealClock(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compilation error: duplicate declaration"

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withClock(clockpkg.RealClock()),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("lifecycle-fail-real", []registry_dto.NamedProfile{
		makeProfile("web", "compile-component"),
	})

	flushed, idle := h.waitUntilIdle(5*time.Second, 30*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete")
	require.True(t, idle, "Phase 2 (idle) should complete with real clock")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one task failed permanently")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no tasks completed")
}

func TestLifecycle_ProductionConfig_FailingExecutor(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compilation error: typescript parsing failed: duplicate declaration"

	h := newPipelineHarness(t,
		withProductionConfig(),
		withClock(clockpkg.RealClock()),
		withMaxRetries(2),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("production-fail", []registry_dto.NamedProfile{
		makeProfile("web", "compile-component"),
	})

	flushed, idle := h.waitUntilIdle(5*time.Second, 30*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete with production config")
	require.True(t, idle,
		"Phase 2 (idle) should complete with production config and real clock")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one task failed permanently")
}

func TestLifecycle_EventCounterTracking(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	const count = 5
	for i := range count {
		h.seedArtefact(
			fmt.Sprintf("counter-track-%d", i),
			[]registry_dto.NamedProfile{
				makeProfile("web", "compile-component"),
			},
		)
	}

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "flush should complete")
	require.True(t, idle, "idle should complete")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()

	assert.Equal(t, int64(count), published, "published event count")
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) >= published (%d)", processed, published)
}

func TestLifecycle_MixedOutcomes_TwoPhaseDetection(t *testing.T) {
	exec := newControllableExecutor()
	exec.failUntil = 2
	exec.failError = "compilation error: syntax error"

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	for i := range 3 {
		h.seedArtefact(
			fmt.Sprintf("mixed-lifecycle-%d", i),
			[]registry_dto.NamedProfile{
				makeProfile("web", "compile-component"),
			},
		)
	}

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "flush should complete with mixed outcomes")
	require.True(t, idle, "idle should complete with mixed outcomes")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(3), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(1), stats.TasksCompleted, "one succeeded")
	assert.Equal(t, int64(2), stats.TasksFailed, "two failed")
}

func TestLifecycle_RealCompiler_InvalidPKC_FullRoundTrip(t *testing.T) {

	inputReader := compiler_adapters.NewMemoryInputReader()
	compiler := compiler_domain.NewCompilerOrchestrator(
		inputReader,
		[]compiler_domain.TransformationPort{},
	)
	capService, err := capabilities.NewServiceWithBuiltins(
		capabilities.WithCompiler(compiler),
	)
	require.NoError(t, err)

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withClock(clockpkg.RealClock()),
	)

	realExecutor := orchestrator_adapters.NewCompilerExecutor(h.registryService, capService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", realExecutor)

	var resultingTags registry_dto.Tags
	resultingTags.SetByName("storageBackendId", "test_store")
	resultingTags.SetByName("fileExtension", ".js")
	resultingTags.SetByName("mimeType", "application/javascript")

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	params := registry_dto.ProfileParams{}
	params.SetByName("sourcePath", "broken-component.pkc")

	h.seedArtefactWithContent("broken-component", minimalInvalidPKC(), []registry_dto.NamedProfile{
		{
			Name: "compiled",
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityNeed,
				CapabilityName: "compile-component",
				DependsOn:      deps,
				Params:         params,
				ResultingTags:  resultingTags,
			},
		},
	})

	flushed, idle := h.waitUntilIdle(5*time.Second, 30*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete")
	require.True(t, idle,
		"Phase 2 (idle) should complete after real compilation (which fails fatally)")

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(0), stats.TasksCompleted, "compilation fails on invalid PKC")
	assert.Equal(t, int64(1), stats.TasksFailed, "one fatal failure from duplicate declaration")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	t.Logf("Published: %d, Processed: %d", published, processed)
	assert.GreaterOrEqual(t, published, int64(1),
		"published should include the initial seed event")
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) >= published (%d)", processed, published)
}

func TestLifecycle_RealCompiler_ValidPKC_SuccessRoundTrip(t *testing.T) {
	inputReader := compiler_adapters.NewMemoryInputReader()
	compiler := compiler_domain.NewCompilerOrchestrator(
		inputReader,
		[]compiler_domain.TransformationPort{},
	)
	capService, err := capabilities.NewServiceWithBuiltins(
		capabilities.WithCompiler(compiler),
	)
	require.NoError(t, err)

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withClock(clockpkg.RealClock()),
	)

	realExecutor := orchestrator_adapters.NewCompilerExecutor(h.registryService, capService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", realExecutor)

	var resultingTags registry_dto.Tags
	resultingTags.SetByName("storageBackendId", "test_store")
	resultingTags.SetByName("fileExtension", ".js")
	resultingTags.SetByName("mimeType", "application/javascript")

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	params := registry_dto.ProfileParams{}
	params.SetByName("sourcePath", "valid-component.pkc")

	h.seedArtefactWithContent("valid-component", minimalValidPKC(), []registry_dto.NamedProfile{
		{
			Name: "compiled",
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityNeed,
				CapabilityName: "compile-component",
				DependsOn:      deps,
				Params:         params,
				ResultingTags:  resultingTags,
			},
		},
	})

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "Phase 1 (flush) should complete with secondary event")
	require.True(t, idle, "Phase 2 (idle) should complete after successful compilation")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()

	t.Logf("Published: %d, Processed: %d", published, processed)
	assert.GreaterOrEqual(t, published, int64(2),
		"published should include secondary event from AddVariant")
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) >= published (%d)", processed, published)
}

func TestLifecycle_MultipleArtefacts_ProductionConfig_RealClock(t *testing.T) {
	exec := newControllableExecutor()
	exec.failUntil = 3
	exec.failError = "compilation error: syntax error in component"

	h := newPipelineHarness(t,
		withProductionConfig(),
		withClock(clockpkg.RealClock()),
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	const artefactCount = 5
	for i := range artefactCount {
		h.seedArtefact(
			fmt.Sprintf("prod-artefact-%d", i),
			[]registry_dto.NamedProfile{
				makeProfile("web", "compile-component"),
			},
		)
	}

	flushed, idle := h.waitUntilIdle(10*time.Second, 15*time.Second)
	require.True(t, flushed, "flush should complete with production config")
	require.True(t, idle, "idle should complete with production config and real clock")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(artefactCount), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(2), stats.TasksCompleted, "two tasks succeeded")
	assert.Equal(t, int64(3), stats.TasksFailed, "three tasks failed")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	assert.GreaterOrEqual(t, processed, published,
		"processed (%d) >= published (%d)", processed, published)
}

func TestLifecycle_FlushBeforeIdle(t *testing.T) {
	exec := newControllableExecutor()
	exec.executionTime = 500 * time.Millisecond

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("flush-order", []registry_dto.NamedProfile{
		makeProfile("web", "compile-component"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "flush should complete while executor is still running")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "idle should complete after executor finishes")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task completed")
}

func minimalValidPKC() []byte {
	return []byte(`<template>
<div>hello</div>
</template>
<script>
const greeting = "hello world";
</script>
`)
}
