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
	"sync"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func newRecordingBus(t *testing.T, subscriber *recordingSubscriber, opts ...WatermillEventBusOption) orchestrator_domain.EventBus {
	t.Helper()
	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	require.NoError(t, err)
	bus := NewWatermillEventBus(discardPublisher{}, subscriber, router, opts...)
	t.Cleanup(func() { _ = bus.Close(context.Background()) })
	return bus
}

type discardPublisher struct{}

func (discardPublisher) Publish(string, ...*message.Message) error { return nil }

func (discardPublisher) Close() error { return nil }

type recordingSubscriber struct {
	mu     sync.Mutex
	topics []string
}

func (s *recordingSubscriber) Subscribe(_ context.Context, topic string) (<-chan *message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.topics = append(s.topics, topic)
	return make(chan *message.Message), nil
}

func (*recordingSubscriber) Close() error { return nil }

func (s *recordingSubscriber) subscribed() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.topics...)
}

func TestSubscribeBroadcastUsesTheBroadcastSubscriber(t *testing.T) {
	competing, broadcast := &recordingSubscriber{}, &recordingSubscriber{}
	bus := newRecordingBus(t, competing, WithBroadcastSubscriber(broadcast))
	noop := func(context.Context, orchestrator_domain.Event) error { return nil }

	require.NoError(t, bus.SubscribeWithHandler(t.Context(), "tasks.work", noop))
	require.NoError(t, orchestrator_domain.SubscribeBroadcast(t.Context(), bus, "artefact.updated", noop))

	assert.Equal(t, []string{"tasks.work"}, competing.subscribed(), "work stays with the competing consumers")
	assert.Equal(t, []string{"artefact.updated"}, broadcast.subscribed(), "every replica receives a broadcast topic")
}

func TestSubscribeBroadcastFallsBackOnAnInProcessBus(t *testing.T) {
	only := &recordingSubscriber{}
	bus := newRecordingBus(t, only)

	require.NoError(t, orchestrator_domain.SubscribeBroadcast(t.Context(), bus, "artefact.updated",
		func(context.Context, orchestrator_domain.Event) error { return nil }))
	assert.Equal(t, []string{"artefact.updated"}, only.subscribed(),
		"an in-process bus already delivers every message to every subscription")
}
