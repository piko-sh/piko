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

package worker_domain

import (
	"context"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/clock"
)

// Wake tells a subscribed worker loop that a queue may have new work to claim.
type Wake struct {
	// Queue is the queue the producer provisioned the job on. An empty string is a broadcast
	// wake.
	Queue string
}

// InProcessNotifier is the default in-memory Notifier: it fans wake signals out to
// subscribers running in the same process, used when no external notify transport is
// wired.
type InProcessNotifier struct {
	// clk is the time source the notifier is built with.
	clk clock.Clock

	// subscribers is the set of live subscriptions to fan wakes out to.
	subscribers map[*subscription]struct{}

	// mu guards subscribers and closed against concurrent notify and subscribe.
	mu sync.Mutex

	// closed reports whether the notifier has been shut down.
	closed bool
}

var (
	_ Notifier = (*InProcessNotifier)(nil)
)

// subscription is one worker loop's registration: the queues it cares about and the
// channel wakes are delivered on.
type subscription struct {
	// queues is the set of queue names this subscription wakes for.
	queues map[string]struct{}

	// wake is the buffered channel wakes are delivered on.
	wake chan Wake

	// wantsAll reports whether the subscription wakes for every queue.
	wantsAll bool

	// unsubscribeOnce guards closing wake so it happens exactly once.
	unsubscribeOnce sync.Once
}

// InProcessNotifierOption configures an InProcessNotifier at construction.
type InProcessNotifierOption func(notifier *InProcessNotifier)

// NewInProcessNotifier builds an InProcessNotifier with no subscribers.
//
// Takes opts (...InProcessNotifierOption) which configure the notifier.
//
// Returns *InProcessNotifier which is the ready notifier.
func NewInProcessNotifier(opts ...InProcessNotifierOption) *InProcessNotifier {
	i := &InProcessNotifier{
		clk:         clock.RealClock(),
		subscribers: make(map[*subscription]struct{}),
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Notify wakes every subscriber interested in the given queue. A closed notifier is a
// no-op.
//
// Takes queue (string) which is the queue that gained work; empty broadcasts to all.
//
// Returns error which is always nil for the in-process notifier.
//
// Concurrency: holds mu for the whole fan-out; each wake is a non-blocking channel send.
func (i *InProcessNotifier) Notify(ctx context.Context, queue string) error {
	ctx, l := logger_domain.From(ctx, log)

	i.mu.Lock()
	defer i.mu.Unlock()
	if i.closed {
		return nil
	}

	amountNotified := 0
	wake := Wake{Queue: queue}
	for subscriber := range i.subscribers {
		if !shouldWakeForQueue(subscriber, queue) {
			continue
		}
		select {
		case subscriber.wake <- wake:
		default:
		}
		amountNotified++
	}

	l.Trace("Notified workers of new jobs",
		logger_domain.String(attrQueue, queue),
		logger_domain.Int("subscribers", len(i.subscribers)),
		logger_domain.Int("subscribers_notified", amountNotified),
	)

	return nil
}

// Subscribe registers interest in a set of queues and returns the wake channel along with
// an unsubscribe function. An empty queue list subscribes to every queue.
//
// Takes queues ([]string) which are the queues to receive wakes for; empty means all.
//
// Returns <-chan Wake which delivers wake signals to the caller.
// Returns func(context.Context) which unsubscribes and closes the wake channel.
// Returns error which is ErrNotifierClosed when the notifier is already closed.
//
// Concurrency: acquires mu to register the subscription; the returned unsubscribe
// function re-acquires mu to remove it and closes the wake channel exactly once.
func (i *InProcessNotifier) Subscribe(ctx context.Context, queues []string) (<-chan Wake, func(ctx context.Context), error) {
	ctx, l := logger_domain.From(ctx, log)

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.closed {
		return nil, nil, ErrNotifierClosed
	}

	indexedQueues := make(map[string]struct{}, len(queues))
	for i := range queues {
		indexedQueues[queues[i]] = struct{}{}
	}

	sub := &subscription{
		queues:   indexedQueues,
		wantsAll: len(indexedQueues) == 0,
		wake:     make(chan Wake, 1),
	}
	i.subscribers[sub] = struct{}{}

	l.Trace("worker subscribed to job notifications",
		logger_domain.Int("queues", len(sub.queues)),
		logger_domain.Int("subscribers", len(i.subscribers)),
	)

	return sub.wake, func(_ context.Context) {
		i.mu.Lock()
		delete(i.subscribers, sub)
		i.mu.Unlock()
		sub.unsubscribeOnce.Do(func() {
			close(sub.wake)
		})
	}, nil
}

// WithNotifierClock sets the time source the notifier uses.
//
// Takes clk (clock.Clock) which is the time source to use.
//
// Returns InProcessNotifierOption which records the clock.
func WithNotifierClock(clk clock.Clock) InProcessNotifierOption {
	return func(i *InProcessNotifier) {
		i.clk = clk
	}
}

// shouldWakeForQueue reports whether a subscription should be woken for a given queue.
//
// Takes sub (*subscription) which is the subscription to test.
// Takes queue (string) which is the queue that gained work; empty means broadcast.
//
// Returns bool which is true when the subscription wakes for the queue.
func shouldWakeForQueue(sub *subscription, queue string) bool {
	if sub.wantsAll {
		return true
	}
	if queue == "" {
		return true
	}
	_, ok := sub.queues[queue]
	return ok
}
