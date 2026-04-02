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

func TestPipelineFlush_EventCountsMatch(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("artefact-1", []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	})
	h.seedArtefact("artefact-2", []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	})
	h.seedArtefact("artefact-3", []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush within timeout")

	published := h.registryService.ArtefactEventsPublished()
	processed := h.bridge.ArtefactEventsProcessed()
	assert.Equal(t, int64(3), published, "expected 3 events published")
	assert.GreaterOrEqual(t, processed, published,
		"processed events should be at least as many as published")
}

func TestPipelineFlush_FlushThenIdle(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("artefact-flush", []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle after flush")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "failed")
}

func TestPipelineFlush_MultipleArtefactsFlushBeforeIdle(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	const artefactCount = 5
	for i := range artefactCount {
		h.seedArtefact(
			fmt.Sprintf("flush-artefact-%d", i),
			[]registry_dto.NamedProfile{
				makeProfile("web", "image.resize"),
			},
		)
	}

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush all events")

	published := h.registryService.ArtefactEventsPublished()
	assert.Equal(t, int64(artefactCount), published, "all artefacts should publish events")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle after flush")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(artefactCount), stats.TasksCompleted, "all tasks completed")
}
