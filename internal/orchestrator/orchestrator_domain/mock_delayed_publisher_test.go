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

package orchestrator_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockDelayedPublisher_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockDelayedPublisher
	ctx := context.Background()
	task := &Task{ID: "zero-val"}

	require.NoError(t, m.Schedule(ctx, task))
	m.Start(ctx)
	m.Stop()
	assert.Equal(t, 0, m.PendingCount())
}

func TestMockDelayedPublisher_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	m := &MockDelayedPublisher{}
	ctx := context.Background()
	task := &Task{ID: "concurrent"}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = m.Schedule(ctx, task)
			m.Start(ctx)
			m.Stop()
			_ = m.PendingCount()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ScheduleCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.StartCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.StopCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PendingCountCallCount))
}

func TestMockDelayedPublisher_Schedule(t *testing.T) {
	t.Parallel()

	t.Run("nil ScheduleFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockDelayedPublisher{}
		err := m.Schedule(context.Background(), &Task{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ScheduleCallCount))
	})

	t.Run("delegates to ScheduleFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTask *Task
		m := &MockDelayedPublisher{
			ScheduleFunc: func(_ context.Context, task *Task) error {
				capturedTask = task
				return nil
			},
		}

		task := &Task{ID: "s1"}
		err := m.Schedule(context.Background(), task)

		require.NoError(t, err)
		assert.Same(t, task, capturedTask)
	})

	t.Run("propagates error from ScheduleFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("schedule failed")
		m := &MockDelayedPublisher{
			ScheduleFunc: func(_ context.Context, _ *Task) error {
				return expected
			},
		}

		err := m.Schedule(context.Background(), &Task{})
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockDelayedPublisher_Start(t *testing.T) {
	t.Parallel()

	t.Run("nil StartFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockDelayedPublisher{}
		m.Start(context.Background())

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.StartCallCount))
	})

	t.Run("delegates to StartFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockDelayedPublisher{
			StartFunc: func(_ context.Context) {
				called = true
			},
		}

		m.Start(context.Background())
		assert.True(t, called)
	})
}

func TestMockDelayedPublisher_Stop(t *testing.T) {
	t.Parallel()

	t.Run("nil StopFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockDelayedPublisher{}
		m.Stop()

		assert.Equal(t, int64(1), atomic.LoadInt64(&m.StopCallCount))
	})

	t.Run("delegates to StopFunc", func(t *testing.T) {
		t.Parallel()

		called := false
		m := &MockDelayedPublisher{
			StopFunc: func() {
				called = true
			},
		}

		m.Stop()
		assert.True(t, called)
	})
}

func TestMockDelayedPublisher_PendingCount(t *testing.T) {
	t.Parallel()

	t.Run("nil PendingCountFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockDelayedPublisher{}
		got := m.PendingCount()

		assert.Equal(t, 0, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PendingCountCallCount))
	})

	t.Run("delegates to PendingCountFunc", func(t *testing.T) {
		t.Parallel()

		m := &MockDelayedPublisher{
			PendingCountFunc: func() int {
				return 17
			},
		}

		got := m.PendingCount()
		assert.Equal(t, 17, got)
	})
}
