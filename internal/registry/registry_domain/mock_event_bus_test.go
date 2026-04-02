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

package registry_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestMockEventBus_Publish(t *testing.T) {
	t.Parallel()

	t.Run("nil PublishFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockEventBus{}

		err := m.Publish(context.Background(), "topic-1", orchestrator_domain.Event{})

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PublishCallCount))
	})

	t.Run("delegates to PublishFunc", func(t *testing.T) {
		t.Parallel()
		event := orchestrator_domain.Event{Payload: map[string]any{"key": "value"}}
		m := &MockEventBus{
			PublishFunc: func(_ context.Context, topic string, event orchestrator_domain.Event) error {
				assert.Equal(t, "topic-1", topic)
				assert.Equal(t, event, event)
				return nil
			},
		}

		err := m.Publish(context.Background(), "topic-1", event)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PublishCallCount))
	})

	t.Run("propagates error from PublishFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("publish failed")
		m := &MockEventBus{
			PublishFunc: func(context.Context, string, orchestrator_domain.Event) error {
				return expectedErr
			},
		}

		err := m.Publish(context.Background(), "topic-1", orchestrator_domain.Event{})

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockEventBus_Subscribe(t *testing.T) {
	t.Parallel()

	t.Run("nil SubscribeFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockEventBus{}

		got, err := m.Subscribe(context.Background(), "topic-1")

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeCallCount))
	})

	t.Run("delegates to SubscribeFunc", func(t *testing.T) {
		t.Parallel()
		eventChannel := make(chan orchestrator_domain.Event, 1)
		m := &MockEventBus{
			SubscribeFunc: func(_ context.Context, topic string) (<-chan orchestrator_domain.Event, error) {
				assert.Equal(t, "topic-1", topic)
				return eventChannel, nil
			},
		}

		got, err := m.Subscribe(context.Background(), "topic-1")

		require.NoError(t, err)
		assert.Equal(t, (<-chan orchestrator_domain.Event)(eventChannel), got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeCallCount))
	})

	t.Run("propagates error from SubscribeFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("subscribe failed")
		m := &MockEventBus{
			SubscribeFunc: func(context.Context, string) (<-chan orchestrator_domain.Event, error) {
				return nil, expectedErr
			},
		}

		got, err := m.Subscribe(context.Background(), "topic-1")

		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockEventBus_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &MockEventBus{}

		err := m.Close(context.Background())

		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()
		called := false
		m := &MockEventBus{
			CloseFunc: func(_ context.Context) error {
				called = true
				return nil
			},
		}

		err := m.Close(context.Background())

		assert.NoError(t, err)
		assert.True(t, called)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CloseCallCount))
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("close failed")
		m := &MockEventBus{
			CloseFunc: func(_ context.Context) error {
				return expectedErr
			},
		}

		err := m.Close(context.Background())

		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestMockEventBus_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockEventBus
	ctx := context.Background()

	assert.NoError(t, m.Publish(ctx, "", orchestrator_domain.Event{}))

	got, err := m.Subscribe(ctx, "")
	assert.Nil(t, got)
	assert.NoError(t, err)

	assert.NoError(t, m.Close(ctx))
}

func TestMockEventBus_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockEventBus{
		PublishFunc: func(context.Context, string, orchestrator_domain.Event) error { return nil },
		SubscribeFunc: func(context.Context, string) (<-chan orchestrator_domain.Event, error) {
			return nil, nil
		},
		CloseFunc: func(_ context.Context) error { return nil },
	}

	ctx := context.Background()
	var wg sync.WaitGroup

	for range goroutines {
		wg.Go(func() {
			_ = m.Publish(ctx, "", orchestrator_domain.Event{})
			_, _ = m.Subscribe(ctx, "")
			_ = m.Close(ctx)
		})
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PublishCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SubscribeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CloseCallCount))
}
