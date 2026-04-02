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

package deadletter_adapters

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/deadletter/deadletter_domain"
)

var _ deadletter_domain.DeadLetterPort[testEntry] = (*MemoryDeadLetterQueue[testEntry])(nil)

func TestMemoryDeadLetterQueue_NewCreatesEmptyQueue(t *testing.T) {
	t.Parallel()

	dlq := newTestMemoryDLQ()

	count, err := dlq.Count(testCtx())
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	entries, err := dlq.Get(testCtx(), 0)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMemoryDeadLetterQueue_Add(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		entries       []testEntry
		expectedCount int
	}{
		{
			name:          "adds single entry",
			entries:       []testEntry{sampleEntry(1)},
			expectedCount: 1,
		},
		{
			name:          "adds multiple entries",
			entries:       []testEntry{sampleEntry(1), sampleEntry(2), sampleEntry(3)},
			expectedCount: 3,
		},
		{
			name: "identical values stored separately",
			entries: []testEntry{
				{Message: "duplicate", Code: 1},
				{Message: "duplicate", Code: 1},
			},
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dlq := newTestMemoryDLQ()

			for _, e := range tc.entries {
				require.NoError(t, dlq.Add(testCtx(), e))
			}

			count, err := dlq.Count(testCtx())
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCount, count)
		})
	}
}

func TestMemoryDeadLetterQueue_Get(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		addCount      int
		limit         int
		expectedCount int
	}{
		{
			name:          "empty queue returns empty slice",
			addCount:      0,
			limit:         0,
			expectedCount: 0,
		},
		{
			name:          "limit zero returns all",
			addCount:      3,
			limit:         0,
			expectedCount: 3,
		},
		{
			name:          "limit negative returns all",
			addCount:      3,
			limit:         -1,
			expectedCount: 3,
		},
		{
			name:          "limit less than count",
			addCount:      5,
			limit:         2,
			expectedCount: 2,
		},
		{
			name:          "limit greater than count",
			addCount:      2,
			limit:         10,
			expectedCount: 2,
		},
		{
			name:          "limit equals count",
			addCount:      3,
			limit:         3,
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dlq := newTestMemoryDLQ()
			for i := range tc.addCount {
				require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
			}

			entries, err := dlq.Get(testCtx(), tc.limit)
			require.NoError(t, err)
			assert.Len(t, entries, tc.expectedCount)
		})
	}
}

func TestMemoryDeadLetterQueue_Remove(t *testing.T) {
	t.Parallel()

	t.Run("returns nil error", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))

		err := dlq.Remove(testCtx(), []testEntry{sampleEntry(1)})
		require.NoError(t, err)
	})

	t.Run("does not remove entries", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(2)))

		_ = dlq.Remove(testCtx(), []testEntry{sampleEntry(1)})

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 2, count, "entries should not be removed")
	})
}

func TestMemoryDeadLetterQueue_Count(t *testing.T) {
	t.Parallel()

	t.Run("empty queue", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("after adds", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

func TestMemoryDeadLetterQueue_Clear(t *testing.T) {
	t.Parallel()

	t.Run("clears all entries", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		require.NoError(t, dlq.Clear(testCtx()))

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("clear empty queue succeeds", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		require.NoError(t, dlq.Clear(testCtx()))
	})
}

func TestMemoryDeadLetterQueue_GetOlderThan(t *testing.T) {
	t.Parallel()

	t.Run("no entries older than duration", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))

		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("all entries older than duration", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		oldTime := time.Now().Add(-2 * time.Hour)
		injectMemoryEntry(dlq, "old-1", oldTime, sampleEntry(1))
		injectMemoryEntry(dlq, "old-2", oldTime, sampleEntry(2))

		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("mixed old and new entries", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		oldTime := time.Now().Add(-2 * time.Hour)
		injectMemoryEntry(dlq, "old-1", oldTime, sampleEntry(1))

		require.NoError(t, dlq.Add(testCtx(), sampleEntry(2)))

		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, sampleEntry(1), entries[0])
	})

	t.Run("empty queue returns nil", func(t *testing.T) {
		t.Parallel()

		dlq := newTestMemoryDLQ()
		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestMemoryDeadLetterQueue_ConcurrentAdds(t *testing.T) {
	t.Parallel()

	dlq := newTestMemoryDLQ()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			_ = dlq.Add(testCtx(), sampleEntry(id))
		}(i)
	}

	wg.Wait()

	count, err := dlq.Count(testCtx())
	require.NoError(t, err)
	assert.Equal(t, goroutines, count)
}

func TestMemoryDeadLetterQueue_AddThenGet(t *testing.T) {
	t.Parallel()

	dlq := newTestMemoryDLQ()
	expected := sampleEntry(42)
	require.NoError(t, dlq.Add(testCtx(), expected))

	entries, err := dlq.Get(testCtx(), 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, expected, entries[0])
}
