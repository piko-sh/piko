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

func TestMockEventBus_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockEventBus
	ctx := context.Background()
	event := Event{Type: "test.event", Payload: map[string]any{"key": "val"}}

	require.NoError(t, m.Publish(ctx, "topic", event))

	eventChannel, err := m.Subscribe(ctx, "topic")
	require.NoError(t, err)
	assert.Nil(t, eventChannel)

	require.NoError(t, m.Close(ctx))
}

func TestMockEventBus_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	m := &MockEventBus{}
	ctx := context.Background()
	event := Event{Type: "concurrent"}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = m.Publish(ctx, "t", event)
			_, _ = m.Subscribe(ctx, "t")
			_ = m.Close(ctx)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PublishCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SubscribeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CloseCallCount))
}

func TestMockEventBus_Publish(t *testing.T) {
	t.Parallel()

	t.Run("nil PublishFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockEventBus{}
		err := m.Publish(context.Background(), "topic", Event{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.PublishCallCount))
	})

	t.Run("delegates to PublishFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTopic string
		var capturedEvent Event
		m := &MockEventBus{
			PublishFunc: func(_ context.Context, topic string, event Event) error {
				capturedTopic = topic
				capturedEvent = event
				return nil
			},
		}

		event := Event{Type: "my.event", Payload: map[string]any{"k": "v"}}
		err := m.Publish(context.Background(), "my-topic", event)

		require.NoError(t, err)
		assert.Equal(t, "my-topic", capturedTopic)
		assert.Equal(t, EventType("my.event"), capturedEvent.Type)
	})

	t.Run("propagates error from PublishFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("publish failed")
		m := &MockEventBus{
			PublishFunc: func(_ context.Context, _ string, _ Event) error {
				return expected
			},
		}

		err := m.Publish(context.Background(), "topic", Event{})
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockEventBus_Subscribe(t *testing.T) {
	t.Parallel()

	t.Run("nil SubscribeFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockEventBus{}
		eventChannel, err := m.Subscribe(context.Background(), "topic")

		require.NoError(t, err)
		assert.Nil(t, eventChannel)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeCallCount))
	})

	t.Run("delegates to SubscribeFunc", func(t *testing.T) {
		t.Parallel()

		want := make(chan Event, 1)
		var capturedTopic string
		m := &MockEventBus{
			SubscribeFunc: func(_ context.Context, topic string) (<-chan Event, error) {
				capturedTopic = topic
				return want, nil
			},
		}

		got, err := m.Subscribe(context.Background(), "events.*")

		require.NoError(t, err)
		assert.Equal(t, (<-chan Event)(want), got)
		assert.Equal(t, "events.*", capturedTopic)
	})

	t.Run("propagates error from SubscribeFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("subscribe failed")
		m := &MockEventBus{
			SubscribeFunc: func(_ context.Context, _ string) (<-chan Event, error) {
				return nil, expected
			},
		}

		_, err := m.Subscribe(context.Background(), "topic")
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockEventBus_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockEventBus{}
		err := m.Close(context.Background())

		require.NoError(t, err)
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

		require.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("close failed")
		m := &MockEventBus{
			CloseFunc: func(_ context.Context) error {
				return expected
			},
		}

		err := m.Close(context.Background())
		assert.ErrorIs(t, err, expected)
	})
}

func TestMockEventBus_ZeroValueIncludesHandler(t *testing.T) {
	t.Parallel()

	var m MockEventBus
	ctx := context.Background()
	event := Event{Type: "test.event"}
	handler := func(_ context.Context, _ Event) error { return nil }

	require.NoError(t, m.Publish(ctx, "topic", event))

	eventChannel, err := m.Subscribe(ctx, "topic")
	require.NoError(t, err)
	assert.Nil(t, eventChannel)

	require.NoError(t, m.Close(ctx))

	require.NoError(t, m.SubscribeWithHandler(ctx, "topic", handler))
}

func TestMockEventBus_ConcurrentAccessWithHandler(t *testing.T) {
	t.Parallel()

	const goroutines = 50
	m := &MockEventBus{}
	ctx := context.Background()
	handler := func(_ context.Context, _ Event) error { return nil }

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_ = m.Publish(ctx, "t", Event{})
			_, _ = m.Subscribe(ctx, "t")
			_ = m.Close(ctx)
			_ = m.SubscribeWithHandler(ctx, "t", handler)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.PublishCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SubscribeCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CloseCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SubscribeWithHandlerCallCount))
}

func TestMockEventBus_SubscribeWithHandler(t *testing.T) {
	t.Parallel()

	t.Run("nil SubscribeWithHandlerFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockEventBus{}
		handler := func(_ context.Context, _ Event) error { return nil }
		err := m.SubscribeWithHandler(context.Background(), "topic", handler)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SubscribeWithHandlerCallCount))
	})

	t.Run("delegates to SubscribeWithHandlerFunc", func(t *testing.T) {
		t.Parallel()

		var capturedTopic string
		var capturedHandler EventHandler
		m := &MockEventBus{
			SubscribeWithHandlerFunc: func(_ context.Context, topic string, handler EventHandler) error {
				capturedTopic = topic
				capturedHandler = handler
				return nil
			},
		}

		handler := func(_ context.Context, _ Event) error { return nil }
		err := m.SubscribeWithHandler(context.Background(), "events.completed", handler)

		require.NoError(t, err)
		assert.Equal(t, "events.completed", capturedTopic)
		assert.NotNil(t, capturedHandler)
	})

	t.Run("propagates error from SubscribeWithHandlerFunc", func(t *testing.T) {
		t.Parallel()

		expected := errors.New("subscribe with handler failed")
		m := &MockEventBus{
			SubscribeWithHandlerFunc: func(_ context.Context, _ string, _ EventHandler) error {
				return expected
			},
		}

		handler := func(_ context.Context, _ Event) error { return nil }
		err := m.SubscribeWithHandler(context.Background(), "topic", handler)
		assert.ErrorIs(t, err, expected)
	})
}
