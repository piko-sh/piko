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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestEdgeCase_NoArtefactsImmediateIdle(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	idle := h.waitForIdle(2 * time.Second)
	require.True(t, idle, "dispatcher should be idle with no artefacts")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(0), stats.TasksDispatched, "no tasks dispatched")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no tasks completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no tasks failed")
	assert.Equal(t, 0, exec.getCallCount(), "executor should not be called")
}

func TestEdgeCase_ContextCancellationShutsDownCleanly(t *testing.T) {
	exec := newControllableExecutor()
	exec.executionTime = 2 * time.Second

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("cancel-artefact", []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	})

	started := waitForCondition(5*time.Second, func() bool {
		return exec.getCallCount() > 0
	})
	require.True(t, started, "executor should start processing")

	h.cancel()
}

func TestEdgeCase_DeduplicationBlocksDuplicateTask(t *testing.T) {
	exec := newControllableExecutor()
	exec.executionTime = 1 * time.Second

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	profile := []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	}

	h.seedArtefact("dedup-artefact", profile)

	started := waitForCondition(5*time.Second, func() bool {
		return exec.getCallCount() > 0
	})
	require.True(t, started, "first task should start processing")

	h.seedArtefact("dedup-artefact", profile)

	idle := h.waitForIdle(10 * time.Second)
	require.True(t, idle, "dispatcher should become idle")

	assert.Equal(t, 1, exec.getCallCount(),
		"executor should be called once due to deduplication")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")
}

func TestEdgeCase_RapidSequentialSeeding(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	const count = 20
	for i := range count {
		h.seedArtefact(
			fmt.Sprintf("rapid-%d", i),
			[]registry_dto.NamedProfile{
				makeProfile("web", "image.resize"),
			},
		)
	}

	flushed := h.waitForFlush(10 * time.Second)
	require.True(t, flushed, "pipeline should flush all events")

	idle := h.waitForIdle(10 * time.Second)
	require.True(t, idle, "dispatcher should become idle")

	published := h.registryService.ArtefactEventsPublished()
	assert.Equal(t, int64(count), published, "all events published")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(count), stats.TasksCompleted, "all tasks completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")
	assert.Equal(t, count, exec.getCallCount(), "executor called for each artefact")
}
