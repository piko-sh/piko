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
	"fmt"
)

// BroadcastSubscriber is implemented by event buses that can deliver every message on a
// topic to every process, as well as sharing a topic's messages between competing
// consumers.
type BroadcastSubscriber interface {
	// SubscribeBroadcastWithHandler subscribes a handler that receives every message on the
	// topic, in every process that subscribes.
	//
	// Takes topic (string) which names the topic.
	// Takes handler (EventHandler) which processes each message.
	//
	// Returns error when the subscription cannot be made.
	SubscribeBroadcastWithHandler(ctx context.Context, topic string, handler EventHandler) error
}

// SubscribeBroadcast registers a handler that receives every topic message on each
// replica.
//
// Buses without broadcast support use their ordinary subscription behaviour.
//
// Takes bus (EventBus) which carries the topic.
// Takes topic (string) which names the topic.
// Takes handler (EventHandler) which processes each message.
//
// Returns error when the subscription cannot be made.
func SubscribeBroadcast(ctx context.Context, bus EventBus, topic string, handler EventHandler) error {
	if broadcaster, ok := bus.(BroadcastSubscriber); ok {
		if err := broadcaster.SubscribeBroadcastWithHandler(ctx, topic, handler); err != nil {
			return fmt.Errorf("subscribing to every message on %q: %w", topic, err)
		}
		return nil
	}
	return bus.SubscribeWithHandler(ctx, topic, handler)
}
