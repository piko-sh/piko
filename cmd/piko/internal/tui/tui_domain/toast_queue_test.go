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
	"testing"
	"time"

	"piko.sh/piko/wdk/clock"
)

func TestToastQueuePushAndCurrent(t *testing.T) {
	mockClock := clock.NewMockClock(time.Unix(0, 0))
	q := NewToastQueueWithClock(mockClock)

	q.Push(ToastInfo, "hello")

	got, ok := q.Current()
	if !ok {
		t.Fatalf("Current returned ok=false after Push")
	}
	if got.Body != "hello" || got.Kind != ToastInfo {
		t.Errorf("Current = %+v, want body=hello kind=ToastInfo", got)
	}
}

func TestToastQueueExpiry(t *testing.T) {
	mockClock := clock.NewMockClock(time.Unix(0, 0))
	q := NewToastQueueWithClock(mockClock)

	q.PushTTL(ToastInfo, "first", time.Second)
	q.PushTTL(ToastInfo, "second", 10*time.Second)

	mockClock.Advance(2 * time.Second)
	got, ok := q.Current()
	if !ok || got.Body != "second" {
		t.Errorf("expected first toast to be evicted: %+v", got)
	}

	mockClock.Advance(20 * time.Second)
	q.Tick()
	if q.Len() != 0 {
		t.Errorf("Tick should evict all expired toasts; len = %d", q.Len())
	}
	if _, ok := q.Current(); ok {
		t.Errorf("Current should report empty after all TTLs elapse")
	}
}

func TestToastQueueOrder(t *testing.T) {
	mockClock := clock.NewMockClock(time.Unix(0, 0))
	q := NewToastQueueWithClock(mockClock)

	q.PushTTL(ToastInfo, "first", time.Hour)
	q.PushTTL(ToastInfo, "second", time.Hour)

	got, _ := q.Current()
	if got.Body != "first" {
		t.Errorf("Current = %q, want first", got.Body)
	}
}

func TestToastQueueDefaultTTLByKind(t *testing.T) {
	if defaultTTLFor(ToastError) <= defaultTTLFor(ToastInfo) {
		t.Errorf("error TTL should exceed info TTL")
	}
	if defaultTTLFor(ToastWarn) <= defaultTTLFor(ToastSuccess) {
		t.Errorf("warn TTL should exceed success TTL")
	}
}

func TestToastQueueConcurrent(t *testing.T) {
	q := NewToastQueue()

	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			q.Push(ToastInfo, "x")
		})
		wg.Go(func() {
			_, _ = q.Current()
		})
	}
	wg.Wait()
}
