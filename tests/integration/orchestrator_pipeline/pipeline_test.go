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

func TestPipeline_SingleArtefactProcessesEndToEnd(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("test-artefact", []registry_dto.NamedProfile{
		makeProfile("web", "image.resize"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle after processing")

	assert.Equal(t, 1, exec.getCallCount(), "executor should be called once")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(1), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "failed")
}

func TestPipeline_MultipleArtefactsWithMultipleProfiles(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	for i := range 3 {
		h.seedArtefact(fmt.Sprintf("artefact-%d", i), []registry_dto.NamedProfile{
			makeProfile("web", "image.resize"),
			makeProfile("thumb", "image.thumbnail"),
		})
	}

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle")

	assert.Equal(t, 6, exec.getCallCount(),
		"executor should be called 6 times (3 artefacts x 2 profiles)")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(6), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "failed")
}

func TestPipeline_ExecutorFailureReachesIdle(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compilation error: syntax error in component"

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("broken-artefact", []registry_dto.NamedProfile{
		makeProfile("web", "compile.typescript"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.advanceUntilIdle(10 * time.Second)
	require.True(t, idle, "dispatcher should become idle even after executor failures")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksFailed, "one task should fail permanently")
	assert.Equal(t, int64(0), stats.TasksCompleted, "no tasks completed")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)
	assert.Contains(t, failures[0].LastError, "compilation error: syntax error in component")
}

func TestPipeline_ExecutorRecoveryOnRetry(t *testing.T) {
	exec := newControllableExecutor()
	exec.failUntil = 1

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("artefact.compiler", exec),
	)

	h.seedArtefact("retry-artefact", []registry_dto.NamedProfile{
		makeProfile("web", "compile.typescript"),
	})

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.advanceUntilIdle(10 * time.Second)
	require.True(t, idle, "dispatcher should become idle after retry succeeds")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task completed after retry")
	assert.Equal(t, int64(0), stats.TasksFailed, "no permanently failed tasks")
	assert.Equal(t, int64(1), stats.TasksRetried, "one retry occurred")
	assert.Equal(t, 2, exec.getCallCount(), "executor called twice (initial + retry)")
}

func TestPipeline_MultipleArtefactsMixedOutcomes(t *testing.T) {
	exec := newControllableExecutor()

	exec.failUntil = 2

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("artefact.compiler", exec),
	)

	for i := range 3 {
		h.seedArtefact(fmt.Sprintf("mixed-artefact-%d", i), []registry_dto.NamedProfile{
			makeProfile("web", "image.resize"),
		})
	}

	flushed := h.waitForFlush(5 * time.Second)
	require.True(t, flushed, "pipeline should flush")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle with mixed outcomes")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(3), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(1), stats.TasksCompleted, "one task succeeded")
	assert.Equal(t, int64(2), stats.TasksFailed, "two tasks failed")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	assert.Len(t, failures, 2, "two failed tasks reported")
}
