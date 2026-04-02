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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestErrorReporting_FailedTasksReturnsDetails(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compiler service failed: typescript parsing error"

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("test-exec", exec),
	)

	err := h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal)
	require.NoError(t, err)

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle")

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1, "expected exactly one failed task")

	f := failures[0]
	assert.Equal(t, "test-exec", f.Executor)
	assert.Contains(t, f.LastError, "compiler service failed: typescript parsing error")
	assert.Equal(t, 1, f.Attempt)
	assert.NotEmpty(t, f.TaskID)
	assert.NotEmpty(t, f.WorkflowID)
}

func TestErrorReporting_SuccessfulBuildReturnsNoFailures(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("test-exec", exec),
	)

	require.NoError(t, h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal))
	require.NoError(t, h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal))

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle)

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(0), stats.TasksFailed)
	assert.Equal(t, int64(2), stats.TasksCompleted)

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	assert.Empty(t, failures, "no failures expected for successful build")
}

func TestErrorReporting_MixedOutcomesReportsOnlyFailures(t *testing.T) {
	successExec := newControllableExecutor()

	failExec := newControllableExecutor()
	failExec.alwaysFail = true
	failExec.failError = "esbuild: duplicate declaration"

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("success-exec", successExec),
		withExecutor("fail-exec", failExec),
	)

	require.NoError(t, h.dispatchTask("success-exec", orchestrator_domain.PriorityNormal))
	require.NoError(t, h.dispatchTask("fail-exec", orchestrator_domain.PriorityNormal))
	require.NoError(t, h.dispatchTask("success-exec", orchestrator_domain.PriorityNormal))

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle)

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(2), stats.TasksCompleted)
	assert.Equal(t, int64(1), stats.TasksFailed)

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1, "only the failed task should be reported")

	assert.Equal(t, "fail-exec", failures[0].Executor)
	assert.Contains(t, failures[0].LastError, "esbuild: duplicate declaration")
}

func TestErrorReporting_FailedTaskAfterRetriesHasCorrectAttempt(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "compilation error"

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("test-exec", exec),
	)

	err := h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal)
	require.NoError(t, err)

	ok := h.advancePastRetry(1)
	require.True(t, ok, "retry timer should be scheduled")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle)

	failures, err := h.dispatcher.FailedTasks(h.ctx)
	require.NoError(t, err)
	require.Len(t, failures, 1)

	assert.Equal(t, 2, failures[0].Attempt, "should report final attempt count")
	assert.Contains(t, failures[0].LastError, "compilation error")
}
