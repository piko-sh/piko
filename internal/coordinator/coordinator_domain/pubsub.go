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

package coordinator_domain

import (
	"context"

	"piko.sh/piko/internal/logger/logger_domain"
)

// subscriber holds a channel and name for receiving build notifications.
type subscriber struct {
	// notificationChannel is the channel for sending build
	// notifications to this subscriber.
	notificationChannel chan BuildNotification

	// name identifies the subscriber for logging purposes.
	name string
}

// Subscribe registers a subscriber to receive build notifications.
//
// Takes name (string) which identifies the subscriber for debugging.
//
// Returns <-chan BuildNotification which yields notifications when builds
// complete.
// Returns UnsubscribeFunc which must be called to stop receiving
// notifications and release resources.
//
// Safe for concurrent use. The returned channel is closed when the
// unsubscribe function is called.
func (s *coordinatorService) Subscribe(name string) (<-chan BuildNotification, UnsubscribeFunc) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()

	id := s.nextSubID
	s.nextSubID++

	sub := subscriber{
		name:                name,
		notificationChannel: make(chan BuildNotification, 1),
	}
	s.subscribers[id] = sub

	unsubscribe := func() {
		s.subMutex.Lock()
		defer s.subMutex.Unlock()
		if _, ok := s.subscribers[id]; ok {
			close(s.subscribers[id].notificationChannel)
			delete(s.subscribers, id)
		}
	}

	return sub.notificationChannel, unsubscribe
}

// publish sends a build notification to all registered subscribers.
//
// Takes notification (BuildNotification) which contains the build result to
// send.
//
// Safe for concurrent use. Uses a read lock to protect the subscribers map.
// Sends do not block; if a subscriber's channel is full, the subscriber is
// skipped and a warning is logged.
func (s *coordinatorService) publish(ctx context.Context, notification BuildNotification) {
	s.subMutex.RLock()
	defer s.subMutex.RUnlock()

	if len(s.subscribers) == 0 {
		return
	}

	_, pl := logger_domain.From(ctx, log)
	pl.Trace("Publishing new build result to subscribers.", logger_domain.Int("subscriber_count", len(s.subscribers)))
	for id, sub := range s.subscribers {
		select {
		case sub.notificationChannel <- notification:
		default:
			pl.Warn("Could not send build notification to subscriber, channel is full.",
				logger_domain.String("subscriber_name", sub.name),
				logger_domain.Uint64("subscriber_id", id),
			)
		}
	}
}
