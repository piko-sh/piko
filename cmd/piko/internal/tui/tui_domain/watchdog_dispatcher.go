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

package tui_domain

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"

	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/wdk/clock"
)

const (
	// WatchdogStreamDisconnected indicates the dispatcher is not currently
	// connected to the upstream watchdog event stream.
	WatchdogStreamDisconnected = "disconnected"

	// WatchdogStreamConnecting indicates the dispatcher is attempting to
	// open or reopen the upstream subscription.
	WatchdogStreamConnecting = "connecting"

	// WatchdogStreamConnected indicates the dispatcher is consuming events
	// from a healthy upstream subscription.
	WatchdogStreamConnected = "connected"

	// WatchdogStreamErrored indicates the upstream subscription failed and
	// the dispatcher is waiting before reconnecting.
	WatchdogStreamErrored = "error"
)

const (
	// dispatcherHistoryDefault is the size of the dispatcher's local
	// event ring buffer. Events older than this are evicted; new
	// subscribers backfill from the snapshot.
	dispatcherHistoryDefault = 1024

	// dispatcherSubscriberBuffer is the default per-subscription channel
	// buffer. Slow subscribers see drops rather than blocking the
	// dispatcher.
	dispatcherSubscriberBuffer = 64

	// dispatcherInitialBackoff is the initial reconnect delay applied
	// after the upstream stream closes unexpectedly.
	dispatcherInitialBackoff = 500 * time.Millisecond

	// dispatcherMaxBackoff is the maximum reconnect delay.
	dispatcherMaxBackoff = 30 * time.Second
)

const (
	// dispatcherStateDisconnected is the int32 form of
	// WatchdogStreamDisconnected stored in state.
	dispatcherStateDisconnected int32 = iota

	// dispatcherStateConnecting is the int32 form of
	// WatchdogStreamConnecting stored in state.
	dispatcherStateConnecting

	// dispatcherStateConnected is the int32 form of
	// WatchdogStreamConnected stored in state.
	dispatcherStateConnected

	// dispatcherStateErrored is the int32 form of WatchdogStreamErrored
	// stored in state.
	dispatcherStateErrored
)

// errDispatcherStopped is the cause attached to the dispatcher's
// consumer context when Stop is invoked. Promoted to a package-level
// sentinel so callers can errors.Is against it instead of string-matching.
var errDispatcherStopped = errors.New("dispatcher stopped")

// errUpstreamClosed is the error broadcast to subscribers when the
// upstream subscription closes unexpectedly and the dispatcher is about
// to reconnect.
var errUpstreamClosed = errors.New("upstream stream closed")

// EventDispatcher fans WatchdogProvider events out to multiple panel
// subscribers. The dispatcher owns a single upstream subscription, keeps
// a local ring of recent events for backfill, applies per-subscriber
// filters, and reconnects with exponential backoff when the upstream
// closes unexpectedly.
type EventDispatcher struct {
	// provider supplies the upstream watchdog event subscription.
	provider WatchdogProvider

	// clock yields the current time and supports test injection.
	clock clock.Clock

	// program is the bubbletea program used for centralised wake-ups.
	program *tea.Program

	// rng generates the backoff jitter for reconnect attempts.
	rng *rand.Rand

	// cancel cancels the dispatcher's consumer context.
	cancel context.CancelCauseFunc

	// subs maps subscription IDs to their per-subscriber state.
	subs map[string]*dispatcherSub

	// history is the local ring buffer of recent events used for backfill.
	history []WatchdogEvent

	// wg waits for the consumer goroutine to exit on Stop.
	wg sync.WaitGroup

	// nextSubID generates monotonically increasing subscription IDs.
	nextSubID atomic.Uint64

	// historyCap is the maximum size of the history ring buffer.
	historyCap int

	// lastEventTS tracks the unix-nanos timestamp of the last delivered event.
	lastEventTS atomic.Int64

	// dropped counts the cumulative dispatcher-wide drops since Start.
	dropped atomic.Uint64

	// subscriberBuffer is the per-subscription channel buffer size.
	subscriberBuffer int

	// backoffInitial is the first reconnect delay after a stream failure.
	backoffInitial time.Duration

	// backoffMax is the maximum reconnect delay.
	backoffMax time.Duration

	// historyMu guards history and historyCap.
	historyMu sync.RWMutex

	// subsMu guards subs.
	subsMu sync.Mutex

	// state holds the current connection state as an int32.
	state atomic.Int32
}

// EventFilter narrows the events delivered to a subscriber.
type EventFilter struct {
	// Types restricts delivery to the supplied event types. Empty means
	// no type filter.
	Types map[WatchdogEventType]struct{}

	// MinPriority restricts delivery to events at or above the given
	// priority. Zero matches all priorities.
	MinPriority WatchdogEventPriority
}

// matches reports whether ev passes the filter.
//
// Takes ev (WatchdogEvent) which is the candidate event.
//
// Returns bool which is true when the event should be delivered.
func (f EventFilter) matches(ev WatchdogEvent) bool {
	if f.MinPriority > 0 && ev.Priority < f.MinPriority {
		return false
	}
	if len(f.Types) > 0 {
		if _, ok := f.Types[ev.EventType]; !ok {
			return false
		}
	}
	return true
}

// dispatcherSub is a single panel subscription registered with the
// dispatcher.
type dispatcherSub struct {
	// id is the unique subscription identifier.
	id string

	// ch is the channel used to deliver events to the subscriber.
	ch chan WatchdogEvent

	// filter narrows the events delivered through ch.
	filter EventFilter

	// dropped counts events skipped because the subscriber's channel was full.
	dropped atomic.Uint64

	// closed indicates the subscription has been cancelled and ch is closed.
	closed atomic.Bool
}

// WatchdogSubscription is the subscriber-facing handle returned by
// Subscribe. Panels read events from Events and call Cancel when
// finished.
type WatchdogSubscription struct {
	// Events is the channel from which the subscriber reads events.
	Events <-chan WatchdogEvent

	// Cancel ends the subscription and closes Events.
	Cancel func()

	// Dropped returns the number of events dropped for this subscriber.
	Dropped func() uint64

	// ID is the unique subscription identifier.
	ID string
}

// NewEventDispatcher creates a dispatcher bound to provider. The
// dispatcher does not start consuming events until Start is called.
//
// Takes provider (WatchdogProvider) which supplies events.
// Takes clk (clock.Clock) which yields the current time. Pass nil to use
// the real system clock.
//
// Returns *EventDispatcher ready for Start.
func NewEventDispatcher(provider WatchdogProvider, clk clock.Clock) *EventDispatcher {
	if clk == nil {
		clk = clock.RealClock()
	}
	return &EventDispatcher{
		provider:         provider,
		clock:            clk,
		history:          make([]WatchdogEvent, 0, dispatcherHistoryDefault),
		historyCap:       dispatcherHistoryDefault,
		subs:             make(map[string]*dispatcherSub),
		subscriberBuffer: dispatcherSubscriberBuffer,
		backoffInitial:   dispatcherInitialBackoff,
		backoffMax:       dispatcherMaxBackoff,

		rng: rand.New(rand.NewPCG(uint64(clk.Now().UnixNano()), 0)), //nolint:gosec // jitter, not a security primitive
	}
}

// SetProgram registers the bubbletea program so the dispatcher can
// dispatch wake-up messages on event delivery. Calling this with nil is
// safe; events will still flow to subscribers but Update will not be
// woken centrally.
//
// Takes program (*tea.Program) which is the bubbletea program.
func (d *EventDispatcher) SetProgram(program *tea.Program) {
	d.program = program
}

// SetHistoryCap configures the size of the local event ring buffer.
// Calling after Start truncates as needed.
//
// Takes capacity (int) which is the new size; non-positive values fall
// back to the default.
//
// Concurrency: Safe for concurrent use; guarded by historyMu.
func (d *EventDispatcher) SetHistoryCap(capacity int) {
	if capacity <= 0 {
		capacity = dispatcherHistoryDefault
	}
	d.historyMu.Lock()
	defer d.historyMu.Unlock()
	d.historyCap = capacity
	if len(d.history) > capacity {
		d.history = d.history[len(d.history)-capacity:]
	}
}

// SetBackoff configures the reconnect delays.
//
// Takes initial (time.Duration) which is the first delay.
// Takes ceiling (time.Duration) which is the cap.
func (d *EventDispatcher) SetBackoff(initial, ceiling time.Duration) {
	if initial > 0 {
		d.backoffInitial = initial
	}
	if ceiling > 0 {
		d.backoffMax = ceiling
	}
}

// Start begins consuming events from the provider in a goroutine. Stops
// when Stop is called or ctx is cancelled.
//
// Takes ctx (context.Context) which controls the lifetime.
func (d *EventDispatcher) Start(ctx context.Context) {
	subCtx, cancel := context.WithCancelCause(ctx)
	d.cancel = cancel

	d.wg.Add(1)
	go d.run(subCtx)
}

// Stop ends the dispatcher's consumer goroutine and waits for it to exit.
// Pending subscribers receive a closed channel.
func (d *EventDispatcher) Stop() {
	if d.cancel != nil {
		d.cancel(errDispatcherStopped)
	}
	d.wg.Wait()
	d.closeAllSubscribers()
}

// Subscribe registers a new subscriber. Existing history matching the
// filter is delivered first, then live events flow as they arrive.
//
// Takes filter (EventFilter) which selects events for this subscriber.
// Takes since (time.Time) which is the back-fill cutoff. Zero disables
// back-fill.
//
// Returns WatchdogSubscription describing the subscription.
//
// Concurrency: Safe for concurrent use; guarded by subsMu and historyMu.
func (d *EventDispatcher) Subscribe(filter EventFilter, since time.Time) WatchdogSubscription {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()

	backfill := d.historySnapshot(filter, since)

	bufferSize := max(len(backfill)+8, d.subscriberBuffer)

	sub := &dispatcherSub{
		ch:     make(chan WatchdogEvent, bufferSize),
		filter: filter,
	}

	id := d.nextSubID.Add(1)
	sub.id = fmt.Sprintf("sub-%d", id)

	for _, ev := range backfill {
		sub.ch <- ev
	}

	d.subs[sub.id] = sub

	cancel := func() { d.cancelSub(sub.id) }

	return WatchdogSubscription{
		ID:      sub.id,
		Events:  sub.ch,
		Cancel:  cancel,
		Dropped: func() uint64 { return sub.dropped.Load() },
	}
}

// HistorySnapshot returns a copy of the dispatcher's local event ring,
// useful for panels that need to render the recent past on first paint
// without registering a subscription.
//
// Returns []WatchdogEvent which is a copy of the history.
//
// Concurrency: Safe for concurrent use; guarded by historyMu.
func (d *EventDispatcher) HistorySnapshot() []WatchdogEvent {
	d.historyMu.RLock()
	defer d.historyMu.RUnlock()
	out := make([]WatchdogEvent, len(d.history))
	copy(out, d.history)
	return out
}

// State returns the current connection state. Panels render the live
// indicator using this value.
//
// Returns string which is one of WatchdogStreamDisconnected,
// WatchdogStreamConnecting, WatchdogStreamConnected, or
// WatchdogStreamErrored.
func (d *EventDispatcher) State() string {
	switch d.state.Load() {
	case dispatcherStateConnecting:
		return WatchdogStreamConnecting
	case dispatcherStateConnected:
		return WatchdogStreamConnected
	case dispatcherStateErrored:
		return WatchdogStreamErrored
	default:
		return WatchdogStreamDisconnected
	}
}

// DroppedTotal returns the cumulative dispatcher-wide drop count.
//
// Returns uint64 which is the total drops since Start.
func (d *EventDispatcher) DroppedTotal() uint64 {
	return d.dropped.Load()
}

// run is the consumer loop. It opens an upstream subscription, forwards
// events, and reconnects with exponential backoff when the upstream
// closes.
//
// Takes ctx (context.Context) which controls the loop's lifetime.
func (d *EventDispatcher) run(ctx context.Context) {
	defer d.wg.Done()
	defer goroutine.RecoverPanic(ctx, "watchdog-dispatcher.run")

	backoff := d.backoffInitial
	for {
		if ctx.Err() != nil {
			return
		}

		d.setState(WatchdogStreamConnecting)
		d.broadcastState(WatchdogStreamConnecting, nil)

		since := d.resumeSince()
		ch, cancel, err := d.provider.SubscribeEvents(ctx, since)
		if err != nil {
			d.setState(WatchdogStreamErrored)
			d.broadcastState(WatchdogStreamErrored, err)
			if !d.sleepWithBackoff(ctx, &backoff) {
				return
			}
			continue
		}

		d.setState(WatchdogStreamConnected)
		d.broadcastState(WatchdogStreamConnected, nil)
		backoff = d.backoffInitial

		d.consume(ctx, ch)
		if cancel != nil {
			cancel()
		}

		if ctx.Err() != nil {
			return
		}

		d.setState(WatchdogStreamErrored)
		d.broadcastState(WatchdogStreamErrored, errUpstreamClosed)
		if !d.sleepWithBackoff(ctx, &backoff) {
			return
		}
	}
}

// consume reads from the upstream channel until it closes or ctx is
// cancelled. Each event is recorded in history and fanned out to every
// subscriber whose filter matches.
//
// Takes ctx (context.Context) which controls the consume loop.
// Takes ch (<-chan WatchdogEvent) which delivers upstream events.
func (d *EventDispatcher) consume(ctx context.Context, ch <-chan WatchdogEvent) {
	defer goroutine.RecoverPanic(ctx, "watchdog-dispatcher.consume")

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			d.recordAndFanOut(ev)
			if d.program != nil {
				d.program.Send(WatchdogEventReceivedMsg{Event: ev})
			}
		}
	}
}

// recordAndFanOut appends an event to the history and delivers it to
// every matching subscriber under a single subsMu hold.
//
// Holding subsMu across both steps ensures Subscribe (which also takes
// subsMu) sees a consistent state: any event observed in a backfill
// snapshot has already been fanned out to every existing subscriber,
// and any event arriving after Subscribe registers is delivered live
// rather than appearing in both.
//
// Takes ev (WatchdogEvent) which is the event to record and deliver.
//
// Concurrency: Safe for concurrent use; guarded by subsMu and the
// inner historyMu.
func (d *EventDispatcher) recordAndFanOut(ev WatchdogEvent) {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()

	d.recordHistory(ev)
	d.lastEventTS.Store(ev.EmittedAt.UnixNano())
	d.deliverLocked(ev)
}

// deliverLocked delivers ev to every subscriber whose filter matches.
//
// The caller must hold subsMu. Each send is non-blocking; full
// channels increment per-subscriber and aggregate drop counters.
//
// Takes ev (WatchdogEvent) which is the event to deliver.
func (d *EventDispatcher) deliverLocked(ev WatchdogEvent) {
	for _, sub := range d.subs {
		if sub.closed.Load() {
			continue
		}
		if !sub.filter.matches(ev) {
			continue
		}
		select {
		case sub.ch <- ev:
		default:
			sub.dropped.Add(1)
			d.dropped.Add(1)
		}
	}
}

// recordHistory appends ev to the local ring, evicting the oldest entry
// when at capacity.
//
// Takes ev (WatchdogEvent) which is the event to record.
//
// Concurrency: Safe for concurrent use; guarded by historyMu.
func (d *EventDispatcher) recordHistory(ev WatchdogEvent) {
	d.historyMu.Lock()
	defer d.historyMu.Unlock()
	if len(d.history) == d.historyCap {
		copy(d.history, d.history[1:])
		d.history[len(d.history)-1] = ev
		return
	}
	d.history = append(d.history, ev)
}

// historySnapshot returns the matching subset of the local history,
// filtered by since and the supplied filter.
//
// Takes filter (EventFilter) which selects events by type/priority.
// Takes since (time.Time) which is the back-fill cutoff. Zero disables.
//
// Returns []WatchdogEvent which is the filtered history.
//
// Concurrency: Safe for concurrent use; guarded by historyMu.
func (d *EventDispatcher) historySnapshot(filter EventFilter, since time.Time) []WatchdogEvent {
	d.historyMu.RLock()
	defer d.historyMu.RUnlock()
	out := make([]WatchdogEvent, 0, len(d.history))
	for _, ev := range d.history {
		if !since.IsZero() && ev.EmittedAt.Before(since) {
			continue
		}
		if !filter.matches(ev) {
			continue
		}
		out = append(out, ev)
	}
	return out
}

// resumeSince returns the timestamp the next upstream subscription should
// back-fill from. Returns zero time before any event has been observed.
//
// Returns time.Time which is the resume cutoff.
func (d *EventDispatcher) resumeSince() time.Time {
	ns := d.lastEventTS.Load()
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

// sleepWithBackoff sleeps for the current backoff (with jitter) and
// doubles backoff up to the configured maximum.
//
// Takes ctx (context.Context) which can cut the sleep short.
// Takes backoff (*time.Duration) which is updated in place.
//
// Returns bool which is true when the sleep completed; false when ctx
// was cancelled.
func (d *EventDispatcher) sleepWithBackoff(ctx context.Context, backoff *time.Duration) bool {
	jitter := time.Duration(d.rng.Int64N(int64(*backoff) + 1))
	wait := *backoff + jitter/DispatcherJitterDivisor

	select {
	case <-ctx.Done():
		return false
	case <-d.clock.NewTimer(wait).C():
	}

	*backoff = min(*backoff*2, d.backoffMax)
	return true
}

// setState records the connection state.
//
// Takes name (string) which is the new state name.
func (d *EventDispatcher) setState(name string) {
	switch name {
	case WatchdogStreamConnecting:
		d.state.Store(dispatcherStateConnecting)
	case WatchdogStreamConnected:
		d.state.Store(dispatcherStateConnected)
	case WatchdogStreamErrored:
		d.state.Store(dispatcherStateErrored)
	default:
		d.state.Store(dispatcherStateDisconnected)
	}
}

// broadcastState sends a WatchdogStreamStateMsg via the program.
//
// Takes name (string) which is the state name.
// Takes err (error) which carries the most recent error, may be nil.
func (d *EventDispatcher) broadcastState(name string, err error) {
	if d.program == nil {
		return
	}
	d.program.Send(WatchdogStreamStateMsg{State: name, Err: err})
}

// cancelSub removes a subscription from the registry and closes its
// channel.
//
// Takes id (string) which identifies the subscription.
//
// Concurrency: Safe for concurrent use; guarded by subsMu.
func (d *EventDispatcher) cancelSub(id string) {
	d.subsMu.Lock()
	sub, ok := d.subs[id]
	if !ok {
		d.subsMu.Unlock()
		return
	}
	delete(d.subs, id)
	d.subsMu.Unlock()

	if sub.closed.CompareAndSwap(false, true) {
		close(sub.ch)
	}
}

// closeAllSubscribers closes every registered subscription channel.
// Called from Stop.
//
// Concurrency: Safe for concurrent use; guarded by subsMu.
func (d *EventDispatcher) closeAllSubscribers() {
	d.subsMu.Lock()
	subs := make([]*dispatcherSub, 0, len(d.subs))
	for _, sub := range d.subs {
		subs = append(subs, sub)
	}
	d.subs = make(map[string]*dispatcherSub)
	d.subsMu.Unlock()

	for _, sub := range subs {
		if sub.closed.CompareAndSwap(false, true) {
			close(sub.ch)
		}
	}
}
