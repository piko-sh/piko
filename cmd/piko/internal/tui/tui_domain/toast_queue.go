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
	"sync"
	"time"

	"piko.sh/piko/wdk/clock"
)

// ToastKind classifies the styling and TTL of a transient message shown in
// the status bar.
type ToastKind int

const (
	// ToastInfo is for plain informational messages.
	ToastInfo ToastKind = iota

	// ToastSuccess marks an action that completed successfully.
	ToastSuccess

	// ToastWarn raises a warning the user should notice but does not need
	// to act on immediately.
	ToastWarn

	// ToastError signals a failure that requires attention.
	ToastError
)

const (
	// defaultToastTTLInfo is the default TTL applied to ToastInfo
	// messages.
	defaultToastTTLInfo = 4 * time.Second

	// defaultToastTTLSuccess is the default TTL applied to ToastSuccess
	// messages.
	defaultToastTTLSuccess = 4 * time.Second

	// defaultToastTTLWarn is the default TTL applied to ToastWarn
	// messages, kept longer so warnings linger on the bar.
	defaultToastTTLWarn = 8 * time.Second

	// defaultToastTTLError is the default TTL applied to ToastError
	// messages, matching the warning TTL so errors remain visible.
	defaultToastTTLError = 8 * time.Second
)

// Toast is a single transient message with an absolute expiry timestamp.
type Toast struct {
	// ExpireAt is the wall-clock instant after which the toast is dropped.
	ExpireAt time.Time

	// Body is the text to render in the status bar.
	Body string

	// Kind selects the toast's styling and is consulted by the status bar
	// to colour the message.
	Kind ToastKind
}

// ToastQueue is a thread-safe FIFO of toasts with TTL-based eviction. The
// status bar reads the front of the queue each render; expired entries are
// pruned lazily.
type ToastQueue struct {
	// clock supplies the current time for TTL bookkeeping.
	clock clock.Clock

	// toasts is the buffered FIFO of pending toasts.
	toasts []Toast

	// mu guards toasts for safe concurrent reads and writes.
	mu sync.RWMutex
}

// NewToastQueue creates an empty queue using the real system clock.
//
// Returns *ToastQueue ready to receive Push calls.
func NewToastQueue() *ToastQueue {
	return &ToastQueue{clock: clock.RealClock()}
}

// NewToastQueueWithClock creates a queue using an injected clock so tests
// can advance time deterministically.
//
// Takes clk (clock.Clock) which yields the current time. A nil clk falls
// back to the real system clock.
//
// Returns *ToastQueue using clk for TTL bookkeeping.
func NewToastQueueWithClock(clk clock.Clock) *ToastQueue {
	if clk == nil {
		clk = clock.RealClock()
	}
	return &ToastQueue{clock: clk}
}

// Push appends a toast with a default TTL chosen by kind.
//
// Takes kind (ToastKind) which selects styling and TTL.
// Takes body (string) which is the message to display.
func (q *ToastQueue) Push(kind ToastKind, body string) {
	q.PushTTL(kind, body, defaultTTLFor(kind))
}

// PushTTL appends a toast with an explicit TTL.
//
// Takes kind (ToastKind) which selects styling.
// Takes body (string) which is the message to display.
// Takes ttl (time.Duration) which is the time-to-live; non-positive values
// fall back to the kind's default TTL.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (q *ToastQueue) PushTTL(kind ToastKind, body string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = defaultTTLFor(kind)
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.toasts = append(q.toasts, Toast{
		Kind:     kind,
		Body:     body,
		ExpireAt: q.clock.Now().Add(ttl),
	})
}

// Current returns the oldest non-expired toast, evicting expired entries
// in the process.
//
// Returns Toast which is the front of the queue (zero value when empty).
// Returns bool which is true when a toast is available.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (q *ToastQueue) Current() (Toast, bool) {
	now := q.clock.Now()
	q.mu.Lock()
	defer q.mu.Unlock()
	q.evictExpired(now)
	if len(q.toasts) == 0 {
		return Toast{}, false
	}
	return q.toasts[0], true
}

// Tick evicts expired toasts. The status bar is repainted on a 1Hz clock
// tick, so calling Tick at that cadence keeps memory bounded even when no
// further toasts are pushed.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (q *ToastQueue) Tick() {
	now := q.clock.Now()
	q.mu.Lock()
	defer q.mu.Unlock()
	q.evictExpired(now)
}

// Len returns the number of toasts currently buffered (including some that
// may have expired but not yet been evicted).
//
// Returns int which is the buffered toast count.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (q *ToastQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.toasts)
}

// evictExpired removes expired toasts from the front of the queue. Caller
// must hold the write lock.
//
// Takes now (time.Time) which is the reference time.
func (q *ToastQueue) evictExpired(now time.Time) {
	cut := 0
	for cut < len(q.toasts) && !q.toasts[cut].ExpireAt.After(now) {
		cut++
	}
	if cut > 0 {
		q.toasts = q.toasts[cut:]
	}
}

// defaultTTLFor returns the canonical TTL for a kind.
//
// Takes kind (ToastKind) which is the toast classification.
//
// Returns time.Duration which is the kind's default TTL.
func defaultTTLFor(kind ToastKind) time.Duration {
	switch kind {
	case ToastSuccess:
		return defaultToastTTLSuccess
	case ToastWarn:
		return defaultToastTTLWarn
	case ToastError:
		return defaultToastTTLError
	default:
		return defaultToastTTLInfo
	}
}
