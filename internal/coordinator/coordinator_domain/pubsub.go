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
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxQueuedNotifications caps each subscriber's queue, which drops its oldest
	// notification when full.
	maxQueuedNotifications = 256
)

// subscriber holds a channel and name for receiving build notifications.
type subscriber struct {
	// notificationChannel is the channel for sending build notifications to this subscriber.
	notificationChannel chan BuildNotification

	// pump delivers notifications to notificationChannel in publish order.
	pump *notificationPump

	// name identifies the subscriber for logging purposes.
	name string
}

// notificationPump delivers queued build notifications in order using a goroutine that
// exits when the queue empties.
type notificationPump struct {
	// out is the subscriber's channel.
	out chan BuildNotification

	// done releases a pending send to out when stop closes it.
	done chan struct{}

	// queue holds notifications waiting to be delivered, oldest first.
	queue []BuildNotification

	// wg tracks the pump goroutine.
	wg sync.WaitGroup

	// mu guards queue, active, and stopped.
	mu sync.Mutex

	// active is true while a pump goroutine is delivering the queue.
	active bool

	// stopped is true once stop has been called; later notifications are dropped.
	stopped bool
}

// newNotificationPump creates a pump that delivers to out.
//
// Takes out (chan BuildNotification) which is the subscriber's channel.
//
// Returns *notificationPump which is idle until a notification is pushed.
func newNotificationPump(out chan BuildNotification) *notificationPump {
	return &notificationPump{
		out:     out,
		done:    make(chan struct{}),
		queue:   nil,
		wg:      sync.WaitGroup{},
		mu:      sync.Mutex{},
		active:  false,
		stopped: false,
	}
}

// push queues a notification for delivery after every notification pushed before it.
//
// When nothing is queued and the subscriber's channel has room, the notification is
// delivered straight away without starting a goroutine.
//
// Takes notification (BuildNotification) which is the notification to deliver.
//
// Returns bool which is true when the queue was full and the oldest queued notification
// was dropped to make room.
//
// Concurrent calls are serialised by mu; queued deliveries run in a pump goroutine.
func (p *notificationPump) push(notification BuildNotification) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return false
	}

	if !p.active && len(p.queue) == 0 {
		select {
		case p.out <- notification:
			return false
		default:
		}
	}

	dropped := false
	if len(p.queue) >= maxQueuedNotifications {
		p.queue = p.queue[1:]
		dropped = true
	}
	p.queue = append(p.queue, notification)

	if !p.active {
		p.active = true
		p.wg.Go(p.deliver)
	}
	return dropped
}

// deliver sends queued notifications in order until the queue is empty or the pump is
// stopped.
//
// Concurrent access is coordinated through mu; channel sends occur after releasing the
// lock.
func (p *notificationPump) deliver() {
	for {
		p.mu.Lock()
		if p.stopped || len(p.queue) == 0 {
			p.active = false
			p.mu.Unlock()
			return
		}
		notification := p.queue[0]
		p.queue = p.queue[1:]
		p.mu.Unlock()

		select {
		case p.out <- notification:
		case <-p.done:
			p.mu.Lock()
			p.active = false
			p.mu.Unlock()
			return
		}
	}
}

// stop discards queued notifications and waits for the pump goroutine to exit. It is safe
// to call more than once.
func (p *notificationPump) stop() {
	p.mu.Lock()
	if !p.stopped {
		p.stopped = true
		p.queue = nil
		close(p.done)
	}
	p.mu.Unlock()
	p.wg.Wait()
}

// Subscribe registers a subscriber to receive build notifications.
//
// Takes name (string) which identifies the subscriber for debugging.
//
// Returns <-chan BuildNotification which yields notifications when builds complete, in
// the order they were published.
// Returns UnsubscribeFunc which must be called to stop receiving notifications and
// release resources.
//
// Safe for concurrent use. The returned channel is closed when the unsubscribe function
// is called.
func (s *coordinatorService) Subscribe(name string) (<-chan BuildNotification, UnsubscribeFunc) {
	s.subMutex.Lock()
	defer s.subMutex.Unlock()

	id := s.nextSubID
	s.nextSubID++

	notificationChannel := make(chan BuildNotification, 1)
	sub := subscriber{
		name:                name,
		notificationChannel: notificationChannel,
		pump:                newNotificationPump(notificationChannel),
	}
	s.subscribers[id] = sub

	unsubscribe := func() {
		s.subMutex.Lock()
		defer s.subMutex.Unlock()
		if registered, ok := s.subscribers[id]; ok {
			registered.pump.stop()
			close(registered.notificationChannel)
			delete(s.subscribers, id)
		}
	}

	return sub.notificationChannel, unsubscribe
}

// publish queues a build notification for every registered subscriber.
//
// Takes notification (BuildNotification) which contains the build result to send.
//
// Safe for concurrent use. Uses a read lock to protect the subscribers map. Never blocks:
// each subscriber's pump delivers its notifications in publish order, and a subscriber
// that falls maxQueuedNotifications behind loses its oldest pending notification.
func (s *coordinatorService) publish(ctx context.Context, notification BuildNotification) {
	s.subMutex.RLock()
	defer s.subMutex.RUnlock()

	if len(s.subscribers) == 0 {
		return
	}

	_, pl := logger_domain.From(ctx, log)
	pl.Trace("Publishing new build result to subscribers.", logger_domain.Int("subscriber_count", len(s.subscribers)))
	for id, sub := range s.subscribers {
		if sub.pump.push(notification) {
			pl.Warn("Subscriber is not keeping up with build notifications; dropped the oldest queued one.",
				logger_domain.String("subscriber_name", sub.name),
				logger_domain.Uint64("subscriber_id", id),
			)
		}
	}
}

// stopNotificationPumps stops every subscriber's pump so no delivery goroutine outlives
// the coordinator. Subscribers' channels stay open until they unsubscribe.
//
// Safe for concurrent use; read-locks the subscribers map.
func (s *coordinatorService) stopNotificationPumps() {
	s.subMutex.RLock()
	defer s.subMutex.RUnlock()
	for _, sub := range s.subscribers {
		sub.pump.stop()
	}
}
