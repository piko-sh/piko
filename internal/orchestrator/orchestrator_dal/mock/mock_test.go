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

package mock_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_dal/mock"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestOrchestratorDAL_GetTasksByID(t *testing.T) {
	t.Parallel()

	t.Run("returns copies of the stored tasks and skips missing IDs", func(t *testing.T) {
		t.Parallel()

		dal := mock.NewOrchestratorDAL()
		dal.SetTask(&orchestrator_domain.Task{ID: "t1", Status: orchestrator_domain.StatusRetrying})

		tasks, err := dal.GetTasksByID(t.Context(), []string{"t1", "missing"})
		require.NoError(t, err)
		require.Len(t, tasks, 1)
		assert.Equal(t, "t1", tasks[0].ID)

		tasks[0].Status = orchestrator_domain.StatusFailed
		again, err := dal.GetTasksByID(t.Context(), []string{"t1"})
		require.NoError(t, err)
		assert.Equal(t, orchestrator_domain.StatusRetrying, again[0].Status)
	})

	t.Run("returns the configured error", func(t *testing.T) {
		t.Parallel()

		dal := mock.NewOrchestratorDAL()
		expected := errors.New("read failed")
		dal.SetBehaviour("GetTasksByID", &mock.Behaviour{Error: expected})

		tasks, err := dal.GetTasksByID(t.Context(), []string{"t1"})
		require.ErrorIs(t, err, expected)
		assert.Nil(t, tasks)
	})
}
