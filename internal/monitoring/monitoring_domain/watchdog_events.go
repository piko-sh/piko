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

package monitoring_domain

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// watchdogEventSubscriber tracks a single live event-stream consumer. The
// channel is closed by the watchdog when the subscription is cancelled or
// the watchdog stops; subscribers should range over it.
type watchdogEventSubscriber struct {
	// ch is the buffered channel events are delivered on. Buffer is
	// eventSubscriberBuffer; a slow subscriber loses oldest pending
	// events rather than blocking the watchdog.
	ch chan WatchdogEventInfo

	// done is signalled by the subscriber's cancel function so the
	// watchdog can prune the entry from the active list and avoid
	// sending into a channel no one reads.
	done chan struct{}

	// closeOnce serialises ch + done close across the cancel-function and
	// Stop paths. Without it, a consumer cancelling the subscription at
	// the same instant the watchdog is closing all subscribers would
	// double-close the channels and panic.
	closeOnce sync.Once
}

// recordEvent appends an emitted event to the in-memory ring (capped at
// defaultEventRingSize) and fans it out to active subscribers. Slow
// subscribers drop the oldest pending event rather than blocking the
// watchdog loop.
//
// Takes info (WatchdogEventInfo) which is the event to record and dispatch.
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) recordEvent(ctx context.Context, info WatchdogEventInfo) {
	w.mu.Lock()

	if len(w.eventRing) >= defaultEventRingSize {
		copy(w.eventRing, w.eventRing[1:])
		w.eventRing = w.eventRing[:len(w.eventRing)-1]
	}
	w.eventRing = append(w.eventRing, info)

	subscribers := make([]*watchdogEventSubscriber, len(w.eventSubscribers))
	copy(subscribers, w.eventSubscribers)
	w.mu.Unlock()

	watchdogEventEmittedCount.Add(ctx, 1,
		metric.WithAttributes(attribute.String("event_type", string(info.EventType))),
	)

	for _, sub := range subscribers {
		if dropped := deliverEventToSubscriber(sub, info); dropped {
			watchdogEventSubscriberDropCount.Add(ctx, 1)
		}
	}
}

// deliverEventToSubscriber pushes an event onto a subscriber's channel,
// dropping the oldest pending event if the channel is full so emission
// never waits on consumers.
//
// Takes sub (*watchdogEventSubscriber) which is the subscriber being
// delivered to.
// Takes info (WatchdogEventInfo) which is the event to deliver.
//
// Returns bool which is true when an older event was discarded to make
// room, so callers can attribute the drop in metrics.
func deliverEventToSubscriber(sub *watchdogEventSubscriber, info WatchdogEventInfo) (dropped bool) {
	defer func() {
		if r := recover(); r != nil {
			dropped = false
		}
	}()

	select {
	case <-sub.done:
		return false
	default:
	}

	select {
	case sub.ch <- info:
		return false
	default:
	}

	select {
	case <-sub.ch:
		dropped = true
	default:
	}
	select {
	case sub.ch <- info:
	default:
	}
	return dropped
}

// ListEvents returns recent watchdog events from the in-memory ring.
//
// Takes limit (int) which caps the number of returned events
// (0 = no cap, return everything in the ring).
// Takes since (time.Time) which filters events emitted before this instant.
// Takes eventType (string) which filters by event type when non-empty.
//
// Returns []WatchdogEventInfo in chronological order (oldest first).
//
// Safe for concurrent use; acquires the watchdog's mutex.
func (w *Watchdog) ListEvents(_ context.Context, limit int, since time.Time, eventType string) []WatchdogEventInfo {
	w.mu.Lock()
	snapshot := make([]WatchdogEventInfo, len(w.eventRing))
	copy(snapshot, w.eventRing)
	w.mu.Unlock()

	filtered := snapshot[:0]
	for _, info := range snapshot {
		if !since.IsZero() && info.EmittedAt.Before(since) {
			continue
		}
		if eventType != "" && string(info.EventType) != eventType {
			continue
		}
		filtered = append(filtered, info)
	}

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}

// SubscribeEvents registers a streaming subscriber and back-fills events
// emitted at or after since (zero disables back-fill).
//
// The returned channel is closed when the cancel function runs, the
// watchdog stops, or ctx is cancelled. The cancel function is idempotent.
//
// Takes since (time.Time) which back-fills events from the ring at or
// after the given instant before live streaming begins. Pass zero to
// skip back-fill.
//
// Returns <-chan WatchdogEventInfo delivering events in emission order.
// Returns func() that cancels the subscription idempotently.
//
// Safe for concurrent use; spawns a lifecycle goroutine.
func (w *Watchdog) SubscribeEvents(ctx context.Context, since time.Time) (<-chan WatchdogEventInfo, func()) {
	sub := &watchdogEventSubscriber{
		ch:   make(chan WatchdogEventInfo, eventSubscriberBuffer),
		done: make(chan struct{}),
	}

	if !w.registerSubscriber(sub, since) {
		close(sub.ch)
		return sub.ch, func() {}
	}
	watchdogEventSubscriberCount.Add(ctx, 1)

	cancelOnce := make(chan struct{})
	cancel := w.subscriberCanceller(ctx, sub, cancelOnce)

	go w.watchSubscriberLifecycle(ctx, cancel, cancelOnce)

	return sub.ch, cancel
}

// registerSubscriber back-fills the supplied subscriber and adds it to the
// active list.
//
// Takes sub (*watchdogEventSubscriber) which is the freshly constructed
// subscriber to attach.
// Takes since (time.Time) which gates back-fill: events emitted before
// this instant are skipped; pass zero to disable back-fill.
//
// Returns bool which is true on success and false when the watchdog is
// already stopped, in which case the caller must close the channel and
// short-circuit.
//
// Safe for concurrent use; acquires the watchdog mutex.
func (w *Watchdog) registerSubscriber(sub *watchdogEventSubscriber, since time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return false
	}

	if !since.IsZero() {
		for _, info := range w.eventRing {
			if info.EmittedAt.Before(since) {
				continue
			}
			deliverEventToSubscriber(sub, info)
		}
	}
	w.eventSubscribers = append(w.eventSubscribers, sub)
	return true
}

// subscriberCanceller builds the idempotent cancel function returned to
// the caller.
//
// It removes sub from the subscriber list, decrements the live subscriber
// gauge, and closes its channels exactly once. The close itself is
// serialised via sub.closeOnce so it cannot race with the Stop path that
// also closes subscribers. cancelOnce is signalled separately so the
// lifecycle goroutine knows to exit. The ctx supplied at subscribe time is
// used for the gauge decrement so metric attributes match the increment
// side.
//
// Takes sub (*watchdogEventSubscriber) which is the subscriber to cancel.
// Takes cancelOnce (chan struct{}) which is closed exactly once when the
// returned func runs.
//
// Returns func() which is the idempotent cancel handler.
func (w *Watchdog) subscriberCanceller(ctx context.Context, sub *watchdogEventSubscriber, cancelOnce chan struct{}) func() {
	detachedCtx := context.WithoutCancel(ctx)
	return func() {
		select {
		case <-cancelOnce:
			return
		default:
		}
		close(cancelOnce)

		w.removeSubscriber(sub)
		if closeSubscriber(sub) {
			watchdogEventSubscriberCount.Add(detachedCtx, -1)
		}
	}
}

// removeSubscriber unlinks the supplied subscriber from the active list
// under the watchdog mutex. Idempotent: a subscriber not in the list is a
// no-op.
//
// Takes sub (*watchdogEventSubscriber) which is the subscriber to remove.
//
// Safe for concurrent use; acquires the watchdog mutex.
func (w *Watchdog) removeSubscriber(sub *watchdogEventSubscriber) {
	w.mu.Lock()
	defer w.mu.Unlock()
	filtered := w.eventSubscribers[:0]
	for _, existing := range w.eventSubscribers {
		if existing == sub {
			continue
		}
		filtered = append(filtered, existing)
	}
	w.eventSubscribers = filtered
}

// watchSubscriberLifecycle observes the supplied context and the watchdog
// stop channel; either closing triggers the subscriber's cancel function.
// Returns when one of the lifecycle signals fires or the subscriber is
// cancelled directly by the consumer.
//
// Takes cancel (func()) which is the idempotent cancel function.
// Takes cancelOnce (<-chan struct{}) which is closed when cancel has
// already run, allowing the goroutine to exit without a second call.
func (w *Watchdog) watchSubscriberLifecycle(ctx context.Context, cancel func(), cancelOnce <-chan struct{}) {
	select {
	case <-ctx.Done():
		cancel()
	case <-w.stopCh:
		cancel()
	case <-cancelOnce:
	}
}

// closeAllEventSubscribers is invoked from Stop to release any active
// subscribers. Each cancel closes the channel and removes the entry; the
// loop runs until none remain.
func (w *Watchdog) closeAllEventSubscribers(ctx context.Context) {
	for {
		sub := w.popFirstSubscriber()
		if sub == nil {
			return
		}
		if closeSubscriber(sub) {
			watchdogEventSubscriberCount.Add(ctx, -1)
		}
	}
}

// popFirstSubscriber removes and returns the first subscriber from the
// active list under the watchdog mutex.
//
// Returns *watchdogEventSubscriber which is the popped subscriber, or
// nil when the list is empty (signalling the caller's loop to terminate).
func (w *Watchdog) popFirstSubscriber() *watchdogEventSubscriber {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.eventSubscribers) == 0 {
		return nil
	}
	sub := w.eventSubscribers[0]
	w.eventSubscribers = w.eventSubscribers[1:]
	return sub
}

// closeSubscriber finalises a subscriber.
//
// Closes its done and event channels exactly once via sub.closeOnce so the
// cancel and Stop paths cannot race into a double-close.
//
// Takes sub (*watchdogEventSubscriber) which is the subscriber to close.
//
// Returns bool which is true when this call performed the close (and
// therefore the gauge should be decremented), false when another caller
// already finalised the subscriber.
func closeSubscriber(sub *watchdogEventSubscriber) bool {
	closed := false
	sub.closeOnce.Do(func() {
		close(sub.done)
		close(sub.ch)
		closed = true
	})
	return closed
}
