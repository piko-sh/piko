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

package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestScheduleGCTasks(t *testing.T) {
	testCases := []struct {
		scheduleErr error
		name        string
	}{
		{name: "a running orchestrator receives the hints task", scheduleErr: nil},
		{name: "a shutting down orchestrator refusal is tolerated", scheduleErr: orchestrator_domain.ErrOrchestratorShuttingDown},
		{name: "any other scheduling failure is tolerated", scheduleErr: errors.New("task insertion queue is full")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var scheduled *orchestrator_domain.Task
			orchestrator := &orchestrator_domain.MockOrchestratorService{}
			orchestrator.ScheduleFunc = func(_ context.Context, task *orchestrator_domain.Task, _ time.Time) (*orchestrator_domain.WorkflowReceipt, error) {
				scheduled = task
				return nil, testCase.scheduleErr
			}
			container := NewContainer()
			container.orchestratorService = orchestrator

			container.ScheduleGCTasks()

			assert.EqualValues(t, 1, orchestrator.ScheduleCallCount.Load())
			require.NotNil(t, scheduled)
			assert.Equal(t, orchestrator_adapters.ExecutorNameBlobGC, scheduled.Executor)
			assert.Equal(t, "blob.gc.hints", scheduled.DeduplicationKey)
		})
	}
}

func TestScheduleGCTasks_WithoutOrchestrator(t *testing.T) {
	container := NewContainer()

	assert.NotPanics(t, container.ScheduleGCTasks)
}
