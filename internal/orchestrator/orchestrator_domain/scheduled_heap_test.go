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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/testutil/allocbudget"
)

func scheduledAt(id string, due time.Time) scheduledRef {
	return scheduledRef{due: due, ref: TaskRef{ID: id}}
}

func popIDs(t *testing.T, h *scheduledRefHeap) []string {
	t.Helper()
	var ids []string
	for {
		entry, ok := h.Pop()
		if !ok {
			return ids
		}
		ids = append(ids, entry.ref.ID)
	}
}

func TestScheduledRefHeap_PopsInDueOrder(t *testing.T) {
	t.Parallel()

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	var h scheduledRefHeap
	for i, id := range []string{"c", "a", "e", "b", "d"} {
		offsets := []time.Duration{3, 1, 5, 2, 4}
		h.Push(scheduledAt(id, base.Add(offsets[i]*time.Minute)))
	}
	require.Equal(t, 5, h.Len())

	next, ok := h.Peek()
	require.True(t, ok)
	assert.Equal(t, "a", next.ref.ID, "peek returns the entry due first")
	assert.Equal(t, 5, h.Len(), "peek does not remove it")

	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, popIDs(t, &h))
}

func TestScheduledRefHeap_EmptyHeap(t *testing.T) {
	t.Parallel()

	var h scheduledRefHeap
	_, ok := h.Peek()
	assert.False(t, ok)
	_, ok = h.Pop()
	assert.False(t, ok)
	assert.Zero(t, h.Len())
}

func TestScheduledRefHeap_InterleavedPushPop(t *testing.T) {
	t.Parallel()

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	var h scheduledRefHeap
	h.Push(scheduledAt("late", base.Add(time.Hour)))
	h.Push(scheduledAt("soon", base.Add(time.Minute)))

	first, ok := h.Pop()
	require.True(t, ok)
	assert.Equal(t, "soon", first.ref.ID)

	h.Push(scheduledAt("sooner", base))
	h.Push(scheduledAt("middle", base.Add(30*time.Minute)))
	assert.Equal(t, []string{"sooner", "middle", "late"}, popIDs(t, &h))
}

func TestScheduledRefHeap_ManyEntries(t *testing.T) {
	t.Parallel()

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	var h scheduledRefHeap
	for i := range 1000 {
		h.Push(scheduledRef{due: base.Add(time.Duration((i*7919)%1000) * time.Second)})
	}
	previous := time.Time{}
	for h.Len() > 0 {
		entry, _ := h.Pop()
		require.False(t, entry.due.Before(previous), "entries come out in due order")
		previous = entry.due
	}
}

func TestScheduledRefHeap_PopReleasesTheReference(t *testing.T) {
	t.Parallel()

	var h scheduledRefHeap
	h.Push(scheduledAt("held", time.Now()))
	_, _ = h.Pop()
	assert.Equal(t, scheduledRef{}, h[:1][0], "a popped slot keeps no reference to its task ID")
}

func TestAllocBudget_ScheduledRefHeap(t *testing.T) {
	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	h := make(scheduledRefHeap, 0, 64)
	ref := TaskRef{ID: "0192f0c4-7c1e-7b39-9d4e-3f2a1b0c9d8e", Attempt: 1}
	allocbudget.Assert(t, allocbudget.Budget{Allocs: 0}, func() {
		h.Push(scheduledRef{due: base, ref: ref})
		_, _ = h.Pop()
	})
}
