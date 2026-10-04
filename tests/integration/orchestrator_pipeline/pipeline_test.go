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
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	clockpkg "piko.sh/piko/wdk/clock"
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

func TestPipeline_TasksDispatchedBeforeDispatcherStartsAreProcessed(t *testing.T) {
	h := newPipelineHarness(t,
		withProductionConfig(),
		withClock(clockpkg.RealClock()),
		withMaxRetries(1),
		withDeferredDispatcherStart(),
	)

	exec := newCascadingExecutor(h.registryService)
	h.dispatcher.RegisterExecutor(context.Background(), "artefact.compiler", exec)

	h.seedArtefact("early.pkc", pkcProfiles("early"))

	require.True(t, h.waitForFlush(10*time.Second), "bridge should handle the created event before the dispatcher starts")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched, "first profile dispatched before start")
	assert.False(t, h.dispatcher.IsIdle(), "a task held until start keeps the dispatcher busy")

	h.startDispatcher()

	flushed, idle := h.waitUntilIdle(10*time.Second, 10*time.Second)
	require.True(t, flushed, "flush should complete")
	require.True(t, idle, "held task should be published and the cascade should finish")

	stats = h.dispatcher.Stats()
	assert.Equal(t, int64(4), stats.TasksDispatched, "all 4 profiles dispatched")
	assert.Equal(t, int64(4), stats.TasksCompleted, "all 4 profiles completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "no failures")
}

func TestPipeline_RecoveredStaleTaskIsDispatchedAgain(t *testing.T) {
	exec := newControllableExecutor()
	h := newPipelineHarness(t,
		withMaxRetries(3),
		withExecutor("stale-executor", exec),
		withRecoverySweepInterval(30*time.Second),
	)

	staleSince := time.Now().Add(-time.Hour)
	require.NoError(t, h.taskStore.CreateTask(h.ctx, &orchestrator_domain.Task{
		ID:               "stale-task",
		WorkflowID:       "stale-workflow",
		Executor:         "stale-executor",
		Status:           orchestrator_domain.StatusProcessing,
		DeduplicationKey: "stale-key",
		Payload:          map[string]any{"input": "data"},
		Config:           orchestrator_domain.TaskConfig{Priority: orchestrator_domain.PriorityNormal, MaxRetries: 3},
		CreatedAt:        staleSince,
		UpdatedAt:        staleSince,
		ExecuteAt:        staleSince,
	}))

	require.True(t, h.clock.AwaitTimerSetup(0, 5*time.Second), "the recovery loop should arm its ticker")
	assert.Equal(t, 0, exec.getCallCount(), "nothing runs before a recovery sweep")

	h.clock.Advance(30 * time.Second)

	require.True(t, waitForCondition(10*time.Second, func() bool {
		return exec.getCallCount() == 1 && h.dispatcher.IsIdle()
	}), "the recovered task must be dispatched and run again")

	stored, err := h.taskStore.GetTasksByID(h.ctx, []string{"stale-task"})
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, orchestrator_domain.StatusComplete, stored[0].Status,
		"completing the recovered task frees its deduplication key")
	assert.Equal(t, 1, exec.getCallCount(), "the recovered task runs exactly once")
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
