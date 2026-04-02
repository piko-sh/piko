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

package orchestrator_adapters

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestNewWatermillEventBus(t *testing.T) {
	t.Parallel()

	bus := NewWatermillEventBus(nil, nil, nil)
	require.NotNil(t, bus)

	web, ok := bus.(*watermillEventBus)
	require.True(t, ok)
	assert.NotNil(t, web.subscriptions)
	assert.False(t, web.isClosed)
}

func TestNewWatermillEventBus_WithComponents(t *testing.T) {
	t.Parallel()

	router, err := message.NewRouter(message.RouterConfig{}, nil)
	require.NoError(t, err)

	bus := NewWatermillEventBus(nil, nil, router)
	require.NotNil(t, bus)

	web, ok := bus.(*watermillEventBus)
	if !ok {
		t.Fatal("expected *watermillEventBus")
	}
	assert.Equal(t, router, web.Router())
	assert.Nil(t, web.Publisher())
	assert.Nil(t, web.Subscriber())
}

func TestWatermillEventBus_CheckNotClosed_Open(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	_, span, _ := log.Span(t.Context(), "test")
	defer span.End()

	err := web.checkNotClosed(t.Context(), span)
	assert.NoError(t, err)
}

func TestWatermillEventBus_CheckNotClosed_Closed(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      true,
	}

	_, span, _ := log.Span(t.Context(), "test")
	defer span.End()

	err := web.checkNotClosed(t.Context(), span)
	require.Error(t, err)
	assert.ErrorIs(t, err, orchestrator_domain.ErrServiceClosed)
}

func TestWatermillEventBus_MarkAsClosed(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	assert.True(t, web.markAsClosed(), "first call should succeed")
	assert.False(t, web.markAsClosed(), "second call should indicate already closed")
	assert.True(t, web.isClosed)
}

func TestWatermillEventBus_CloseAlreadyClosed(t *testing.T) {
	t.Parallel()

	router, err := message.NewRouter(message.RouterConfig{}, nil)
	require.NoError(t, err)

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      true,
		router:        router,
	}

	err = web.Close(t.Context())
	assert.NoError(t, err)
}

func TestWatermillEventBus_CreateSubscription(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	eventChannel, ctx := web.createSubscription(t.Context(), "test-topic")
	require.NotNil(t, eventChannel)
	require.NotNil(t, ctx)

	assert.Equal(t, eventBusSubscriptionBufferSize, cap(eventChannel))

	sub, exists := web.subscriptions["test-topic"]
	require.True(t, exists)
	assert.Equal(t, "test-topic", sub.topic)
	assert.NotNil(t, sub.cancelFunc)
	assert.Equal(t, eventChannel, sub.outputChan)
}

func TestWatermillEventBus_Unsubscribe(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	eventChannel, _ := web.createSubscription(t.Context(), "test-topic")
	require.NotNil(t, eventChannel)
	assert.Len(t, web.subscriptions, 1)

	web.unsubscribe(t.Context(), "test-topic")
	assert.Len(t, web.subscriptions, 0)
}

func TestWatermillEventBus_UnsubscribeNonExistent(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	web.unsubscribe(t.Context(), "nonexistent-topic")
	assert.Len(t, web.subscriptions, 0)
}

func TestWatermillEventBus_PublishWhenClosed(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      true,
	}

	ctx := t.Context()
	event := orchestrator_domain.Event{
		Type:    orchestrator_domain.EventType("test"),
		Payload: map[string]any{"key": "value"},
	}

	err := web.Publish(ctx, "topic", event)
	require.Error(t, err)
	assert.ErrorIs(t, err, orchestrator_domain.ErrServiceClosed)
}

func TestWatermillEventBus_SubscribeWhenClosed(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      true,
	}

	ctx := t.Context()
	eventChannel, err := web.Subscribe(ctx, "topic")
	require.Error(t, err)
	assert.Nil(t, eventChannel)
	assert.ErrorIs(t, err, orchestrator_domain.ErrServiceClosed)
}

func TestWatermillEventBus_SubscribeWithHandlerWhenClosed(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      true,
	}

	ctx := t.Context()
	err := web.SubscribeWithHandler(ctx, "topic", func(_ context.Context, _ orchestrator_domain.Event) error {
		return nil
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, orchestrator_domain.ErrServiceClosed)
}

func TestWatermillEventBus_CloseAllSubscriptions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	web.createSubscription(ctx, "topic-1")
	web.createSubscription(ctx, "topic-2")
	web.createSubscription(ctx, "topic-3")

	count := web.closeAllSubscriptions(ctx)

	assert.Equal(t, 3, count)
	assert.Nil(t, web.subscriptions)
}

func TestWatermillEventBus_CloseAllSubscriptions_Empty(t *testing.T) {
	t.Parallel()

	web := &watermillEventBus{
		subscriptions: make(map[string]*watermillSubscription),
		isClosed:      false,
	}

	ctx := t.Context()
	count := web.closeAllSubscriptions(ctx)

	assert.Equal(t, 0, count)
}
