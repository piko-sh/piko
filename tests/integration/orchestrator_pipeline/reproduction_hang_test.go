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

func TestReproduction_FailingExecutorRealClock(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compilation error: typescript parsing failed: duplicate declaration"

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("broken-component", []registry_dto.NamedProfile{
		makeProfile("web", "compile-component"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.advanceUntilIdle(30 * time.Second)
	require.True(t, idle,
		"dispatcher should become idle after executor failures")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one task should fail permanently")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no tasks completed")
}

func TestReproduction_RealCompilerWithInvalidPKC(t *testing.T) {

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

	deps := registry_dto.DependenciesFromSlice([]string{"source"})
	params := registry_dto.ProfileParams{}
	params.SetByName("sourcePath", "broken-component.pkc")

	h.seedArtefactWithContent("broken-component", minimalInvalidPKC(), []registry_dto.NamedProfile{
		{
			Name: "compiled",
			Profile: registry_dto.DesiredProfile{
				Priority:       registry_dto.PriorityNeed,
				CapabilityName: "compile-component",
				Params:         params,
				DependsOn:      deps,
			},
		},
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.waitForIdle(30 * time.Second)
	require.True(t, idle,
		"dispatcher should become idle after real compilation failure")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one task should fail permanently")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no tasks completed")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)
}

func TestReproduction_MultipleProfilesMixedOutcomesRealClock(t *testing.T) {
	exec := newControllableExecutor()

	exec.failUntil = 2
	exec.failError = "compilation error: typescript parsing failed: duplicate declaration"

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withClock(clockpkg.RealClock()),
		withExecutor("artefact.compiler", exec),
	)

	for i := range 3 {
		h.seedArtefact(
			fmt.Sprintf("component-%d", i),
			[]registry_dto.NamedProfile{
				makeProfile("web", "compile-component"),
			},
		)
	}

	flushed := h.waitForFlush(10 * time.Second)
	require.True(t, flushed, "pipeline should flush all events")

	idle := h.waitForIdle(10 * time.Second)
	require.True(t, idle,
		"dispatcher should become idle with mixed outcomes and real clock")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(3), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task succeeded")
	assert.Equal(t, int64(2), stats.TasksFailed, "two tasks failed")
}

func minimalInvalidPKC() []byte {
	return []byte(`<template>
<div>test</div>
</template>
<script>
const v = 1;
const v = 2;
</script>
`)
}
