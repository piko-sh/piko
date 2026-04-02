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

package retry_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/retry"
)

type testItem struct {
	retryAt time.Time
	label   string
}

func newHeap() *retry.Heap[*testItem] {
	return retry.NewHeap(func(item *testItem) time.Time { return item.retryAt })
}

func TestHeap_EmptyLen(t *testing.T) {
	h := newHeap()
	require.Equal(t, 0, h.Len())
}

func TestHeap_PopEmpty(t *testing.T) {
	h := newHeap()
	_, ok := h.PopItem()
	require.False(t, ok)
}

func TestHeap_PeekEmpty(t *testing.T) {
	h := newHeap()
	_, ok := h.Peek()
	require.False(t, ok)
}

func TestHeap_PushIncrementsLen(t *testing.T) {
	h := newHeap()
	h.PushItem(&testItem{retryAt: time.Now()})
	require.Equal(t, 1, h.Len())

	h.PushItem(&testItem{retryAt: time.Now()})
	require.Equal(t, 2, h.Len())
}

func TestHeap_PopsInTimeOrder(t *testing.T) {
	h := newHeap()
	now := time.Now()

	h.PushItem(&testItem{retryAt: now.Add(30 * time.Second), label: "third"})
	h.PushItem(&testItem{retryAt: now.Add(10 * time.Second), label: "first"})
	h.PushItem(&testItem{retryAt: now.Add(20 * time.Second), label: "second"})

	first, ok := h.PopItem()
	require.True(t, ok)
	require.Equal(t, "first", first.label)

	second, ok := h.PopItem()
	require.True(t, ok)
	require.Equal(t, "second", second.label)

	third, ok := h.PopItem()
	require.True(t, ok)
	require.Equal(t, "third", third.label)

	_, ok = h.PopItem()
	require.False(t, ok)
}

func TestHeap_PeekReturnsEarliest(t *testing.T) {
	h := newHeap()
	now := time.Now()

	h.PushItem(&testItem{retryAt: now.Add(20 * time.Second), label: "later"})
	h.PushItem(&testItem{retryAt: now.Add(5 * time.Second), label: "sooner"})

	item, ok := h.Peek()
	require.True(t, ok)
	require.Equal(t, "sooner", item.label)
	require.Equal(t, 2, h.Len())
}

func TestHeap_PeekDoesNotRemove(t *testing.T) {
	h := newHeap()
	h.PushItem(&testItem{retryAt: time.Now(), label: "only"})

	_, ok := h.Peek()
	require.True(t, ok)
	require.Equal(t, 1, h.Len())
}

func TestHeap_ManyItems(t *testing.T) {
	h := newHeap()
	base := time.Now()
	n := 100

	for i := n; i > 0; i-- {
		h.PushItem(&testItem{retryAt: base.Add(time.Duration(i) * time.Second)})
	}

	require.Equal(t, n, h.Len())

	var previous time.Time
	for h.Len() > 0 {
		item, ok := h.PopItem()
		require.True(t, ok)
		if !previous.IsZero() {
			require.False(t, item.retryAt.Before(previous), "items should pop in non-decreasing time order")
		}
		previous = item.retryAt
	}
}

func TestHeap_DuplicateTimes(t *testing.T) {
	h := newHeap()
	same := time.Now()

	h.PushItem(&testItem{retryAt: same, label: "a"})
	h.PushItem(&testItem{retryAt: same, label: "b"})
	h.PushItem(&testItem{retryAt: same, label: "c"})

	require.Equal(t, 3, h.Len())

	seen := make(map[string]bool)
	for h.Len() > 0 {
		item, ok := h.PopItem()
		require.True(t, ok)
		seen[item.label] = true
	}
	require.Len(t, seen, 3)
}

func TestHeap_PushAfterDrain(t *testing.T) {
	h := newHeap()
	now := time.Now()

	h.PushItem(&testItem{retryAt: now})
	_, _ = h.PopItem()
	require.Equal(t, 0, h.Len())

	h.PushItem(&testItem{retryAt: now.Add(time.Second), label: "reused"})
	item, ok := h.PopItem()
	require.True(t, ok)
	require.Equal(t, "reused", item.label)
}
