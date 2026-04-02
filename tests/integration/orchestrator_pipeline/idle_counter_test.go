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

func TestIdleCounter_TaskSucceedsOnFirstAttempt(t *testing.T) {
	exec := newControllableExecutor()

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("test-exec", exec),
	)

	err := h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal)
	require.NoError(t, err)

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle after successful task")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(1), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "failed")
	assert.Equal(t, int64(0), stats.TasksRetried, "retried")
	assert.Equal(t, 1, exec.getCallCount(), "executor call count")
}

func TestIdleCounter_TaskFailsAfterAllRetries(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true
	exec.failError = "permanent failure"

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("test-exec", exec),
	)

	err := h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal)
	require.NoError(t, err)

	ok := h.advancePastRetry(1)
	require.True(t, ok, "delayed publisher should schedule retry timer")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle even after task fails with retries")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(2), stats.TasksDispatched, "dispatched (initial + 1 retry re-dispatch)")
	assert.Equal(t, int64(0), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "failed")
	assert.Equal(t, int64(1), stats.TasksRetried, "retried")
	assert.Equal(t, 2, exec.getCallCount(), "executor called twice (initial + retry)")
}

func TestIdleCounter_TaskFailsThenSucceeds(t *testing.T) {
	exec := newControllableExecutor()
	exec.failUntil = 1

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("test-exec", exec),
	)

	err := h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal)
	require.NoError(t, err)

	ok := h.advancePastRetry(1)
	require.True(t, ok, "delayed publisher should schedule retry timer")

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle after retry succeeds")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(2), stats.TasksDispatched, "dispatched (initial + 1 retry re-dispatch)")
	assert.Equal(t, int64(1), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(0), stats.TasksFailed, "failed")
	assert.Equal(t, int64(1), stats.TasksRetried, "retried")
	assert.Equal(t, 2, exec.getCallCount(), "executor called twice")
}

func TestIdleCounter_MultipleTasksMixedOutcomes(t *testing.T) {
	successExec := newControllableExecutor()

	failExec := newControllableExecutor()
	failExec.alwaysFail = true
	failExec.failError = "always fails"

	retryExec := newControllableExecutor()
	retryExec.failUntil = 1

	h := newPipelineHarness(t,
		withMaxRetries(2),
		withExecutor("success-exec", successExec),
		withExecutor("fail-exec", failExec),
		withExecutor("retry-exec", retryExec),
	)

	require.NoError(t, h.dispatchTask("success-exec", orchestrator_domain.PriorityNormal))
	require.NoError(t, h.dispatchTask("fail-exec", orchestrator_domain.PriorityNormal))
	require.NoError(t, h.dispatchTask("retry-exec", orchestrator_domain.PriorityNormal))

	idle := h.advanceUntilIdle(10 * time.Second)
	require.True(t, idle, "dispatcher should become idle with mixed outcomes")

	stats := h.dispatcher.Stats()

	assert.Equal(t, int64(5), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(2), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "failed")
	assert.Equal(t, int64(2), stats.TasksRetried, "retried")
}

func TestIdleCounter_ZeroRetryTaskFailsImmediately(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("test-exec", exec),
	)

	err := h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal)
	require.NoError(t, err)

	idle := h.waitForIdle(5 * time.Second)
	require.True(t, idle, "dispatcher should become idle immediately when task fails with zero retries")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(1), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(0), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(1), stats.TasksFailed, "failed")
	assert.Equal(t, int64(0), stats.TasksRetried, "retried")
	assert.Equal(t, 1, exec.getCallCount(), "executor called once")
}

func TestIdleCounter_ManyTasksAllFail(t *testing.T) {
	exec := newControllableExecutor()
	exec.alwaysFail = true

	h := newPipelineHarness(t,
		withMaxRetries(1),
		withExecutor("test-exec", exec),
	)

	const taskCount = 10
	for range taskCount {
		require.NoError(t, h.dispatchTask("test-exec", orchestrator_domain.PriorityNormal))
	}

	idle := h.waitForIdle(10 * time.Second)
	require.True(t, idle, "dispatcher should become idle after all tasks fail")

	stats := h.dispatcher.Stats()
	assert.Equal(t, int64(taskCount), stats.TasksDispatched, "dispatched")
	assert.Equal(t, int64(0), stats.TasksCompleted, "completed")
	assert.Equal(t, int64(taskCount), stats.TasksFailed, "failed")
	assert.Equal(t, int64(0), stats.TasksRetried, "retried")
	assert.Equal(t, taskCount, exec.getCallCount(), "executor call count")
}
