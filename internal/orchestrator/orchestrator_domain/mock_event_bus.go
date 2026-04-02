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
	"sync/atomic"
)

// MockEventBus is a test double for EventBus where nil function fields
// return zero values and call counts are tracked atomically.
type MockEventBus struct {
	// PublishFunc is the function called by Publish.
	PublishFunc func(ctx context.Context, topic string, event Event) error

	// SubscribeFunc is the function called by Subscribe.
	SubscribeFunc func(ctx context.Context, topic string) (<-chan Event, error)

	// SubscribeWithHandlerFunc is the function called by SubscribeWithHandler.
	SubscribeWithHandlerFunc func(ctx context.Context, topic string, handler EventHandler) error

	// CloseFunc is the function called by Close.
	CloseFunc func(ctx context.Context) error

	// PublishCallCount tracks how many times Publish
	// was called.
	PublishCallCount int64

	// SubscribeCallCount tracks how many times
	// Subscribe was called.
	SubscribeCallCount int64

	// SubscribeWithHandlerCallCount tracks how many
	// times SubscribeWithHandler was called.
	SubscribeWithHandlerCallCount int64

	// CloseCallCount tracks how many times Close was
	// called.
	CloseCallCount int64
}

// Publish sends an event to the given topic.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes topic (string) which identifies the event topic.
// Takes event (Event) which is the event to publish.
//
// Returns error, or nil if PublishFunc is nil.
func (m *MockEventBus) Publish(ctx context.Context, topic string, event Event) error {
	atomic.AddInt64(&m.PublishCallCount, 1)
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, topic, event)
	}
	return nil
}

// Subscribe returns a channel that receives events for the given topic.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes topic (string) which identifies the event topic to subscribe to.
//
// Returns (<-chan Event, error), or (nil, nil) if SubscribeFunc is nil.
func (m *MockEventBus) Subscribe(ctx context.Context, topic string) (<-chan Event, error) {
	atomic.AddInt64(&m.SubscribeCallCount, 1)
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(ctx, topic)
	}
	return nil, nil
}

// SubscribeWithHandler registers a handler for events on a topic.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes topic (string) which identifies the event topic.
// Takes handler (EventHandler) which is the callback to invoke for each event.
//
// Returns error, or nil if SubscribeWithHandlerFunc is nil.
func (m *MockEventBus) SubscribeWithHandler(ctx context.Context, topic string, handler EventHandler) error {
	atomic.AddInt64(&m.SubscribeWithHandlerCallCount, 1)
	if m.SubscribeWithHandlerFunc != nil {
		return m.SubscribeWithHandlerFunc(ctx, topic, handler)
	}
	return nil
}

// Close releases all resources and closes active subscriptions.
//
// Takes ctx (context.Context) which carries logging context for the
// shutdown operation.
//
// Returns error, or nil if CloseFunc is nil.
func (m *MockEventBus) Close(ctx context.Context) error {
	atomic.AddInt64(&m.CloseCallCount, 1)
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}
