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

package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type stubTaskStore struct{ orchestrator_domain.TaskStore }

type stubEventBus struct{ orchestrator_domain.EventBus }

func TestConfig_Validate_NilTaskStore(t *testing.T) {
	t.Parallel()

	config := &Config{}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TaskStore")
}

func TestConfig_Validate_NilEventBus(t *testing.T) {
	t.Parallel()

	config := &Config{TaskStore: &stubTaskStore{}}
	err := config.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EventBus")
}

func TestConfig_Validate_Valid(t *testing.T) {
	t.Parallel()

	config := &Config{
		TaskStore: &stubTaskStore{},
		EventBus:  &stubEventBus{},
	}
	assert.NoError(t, config.Validate())
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("both nil returns TaskStore error first", func(t *testing.T) {
		t.Parallel()

		config := &Config{}
		err := config.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, errTaskStoreNil)
	})

	t.Run("TaskStore set but EventBus nil returns EventBus error", func(t *testing.T) {
		t.Parallel()

		config := &Config{TaskStore: &stubTaskStore{}}
		err := config.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, errEventBusNil)
	})

	t.Run("both set returns no error", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			TaskStore: &stubTaskStore{},
			EventBus:  &stubEventBus{},
		}
		assert.NoError(t, config.Validate())
	})

	t.Run("optional fields do not affect validation", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			TaskStore:          &stubTaskStore{},
			EventBus:           &stubEventBus{},
			WorkerCount:        0,
			SchedulerInterval:  0,
			DispatcherInterval: 0,
		}
		assert.NoError(t, config.Validate())
	})

	t.Run("validation with all fields populated", func(t *testing.T) {
		t.Parallel()

		config := &Config{
			TaskStore:          &stubTaskStore{},
			EventBus:           &stubEventBus{},
			WorkerCount:        16,
			SchedulerInterval:  30 * time.Second,
			DispatcherInterval: 5 * time.Second,
		}
		assert.NoError(t, config.Validate())
	})
}

func TestNewTask(t *testing.T) {
	t.Parallel()

	payload := map[string]any{"key": "value"}
	task := NewTask("my-executor", payload)

	require.NotNil(t, task)
	assert.Equal(t, "my-executor", task.Executor)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, orchestrator_domain.StatusPending, task.Status)
}

func TestNewTask_UniqueIDs(t *testing.T) {
	t.Parallel()

	first := NewTask("executor-a", nil)
	second := NewTask("executor-b", nil)

	require.NotNil(t, first)
	require.NotNil(t, second)
	assert.NotEqual(t, first.ID, second.ID)
}

func TestNewTask_NilPayload(t *testing.T) {
	t.Parallel()

	task := NewTask("executor", nil)
	require.NotNil(t, task)
	assert.Equal(t, "executor", task.Executor)
}

func TestNewService(t *testing.T) {
	t.Parallel()

	t.Run("valid config returns service", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			TaskStore: &stubTaskStore{},
			EventBus:  &stubEventBus{},
		})
		require.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("nil TaskStore returns error", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			EventBus: &stubEventBus{},
		})
		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "TaskStore")
	})

	t.Run("nil EventBus returns error", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			TaskStore: &stubTaskStore{},
		})
		require.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "EventBus")
	})

	t.Run("empty config returns validation error", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{})
		require.Error(t, err)
		assert.Nil(t, service)
	})

	t.Run("custom worker count is accepted", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			TaskStore:   &stubTaskStore{},
			EventBus:    &stubEventBus{},
			WorkerCount: 16,
		})
		require.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("zero worker count defaults without error", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			TaskStore:   &stubTaskStore{},
			EventBus:    &stubEventBus{},
			WorkerCount: 0,
		})
		require.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("negative worker count defaults without error", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			TaskStore:   &stubTaskStore{},
			EventBus:    &stubEventBus{},
			WorkerCount: -1,
		})
		require.NoError(t, err)
		assert.NotNil(t, service)
	})

	t.Run("custom intervals are accepted", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(context.Background(), Config{
			TaskStore:          &stubTaskStore{},
			EventBus:           &stubEventBus{},
			SchedulerInterval:  30 * time.Second,
			DispatcherInterval: 5 * time.Second,
		})
		require.NoError(t, err)
		assert.NotNil(t, service)
	})
}

func TestPriorityConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, orchestrator_domain.PriorityLow, PriorityLow)
	assert.Equal(t, orchestrator_domain.PriorityNormal, PriorityNormal)
	assert.Equal(t, orchestrator_domain.PriorityHigh, PriorityHigh)
}
