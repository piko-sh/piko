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
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/deadletter/deadletter_domain"
	"piko.sh/piko/wdk/safedisk"
)

var _ deadletter_domain.DeadLetterPort[testEntry] = (*DiskDeadLetterQueue[testEntry])(nil)

func TestDiskDeadLetterQueue_NewWithSandbox(t *testing.T) {
	t.Parallel()

	dlq, sandbox := newTestDiskDLQ(t)
	assert.NotNil(t, dlq.sandbox)
	assert.Equal(t, sandbox, dlq.sandbox)
	assert.Equal(t, "deadletters.jsonl", dlq.fileName)
}

func TestDiskDeadLetterQueue_NewWithoutSandboxCreatesDefault(t *testing.T) {
	t.Parallel()

	dlq := NewDiskDeadLetterQueue[testEntry]("/tmp/test-dlq.jsonl")
	concrete, ok := dlq.(*DiskDeadLetterQueue[testEntry])
	require.True(t, ok)
	assert.NotNil(t, concrete.sandbox)
	assert.Equal(t, "test-dlq.jsonl", concrete.fileName)
}

func TestDiskDeadLetterQueue_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds single entry to new file", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))

		data, err := sandbox.ReadFile("deadletters.jsonl")
		require.NoError(t, err)
		assert.Contains(t, string(data), `"message":"entry-1"`)
	})

	t.Run("adds multiple entries", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("JSON contains timestamp data and id fields", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))

		data, err := sandbox.ReadFile("deadletters.jsonl")
		require.NoError(t, err)

		var wrapped wrappedEntry[testEntry]
		require.NoError(t, json.Unmarshal(data[:len(data)-1], &wrapped))
		assert.NotEmpty(t, wrapped.ID)
		assert.False(t, wrapped.Timestamp.IsZero())
		assert.Equal(t, sampleEntry(1), wrapped.Data)
	})

	t.Run("error stating existing file", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		sandbox.StatErr = errors.New("disk failure")

		err := dlq.Add(testCtx(), sampleEntry(1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading dead letter file")
	})

	t.Run("error opening file for append", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		sandbox.OpenFileErr = errors.New("disk full")

		err := dlq.Add(testCtx(), sampleEntry(1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "writing dead letter entry")
	})
}

func TestDiskDeadLetterQueue_Get(t *testing.T) {
	t.Parallel()

	t.Run("file not found returns nil nil", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		entries, err := dlq.Get(testCtx(), 0)
		require.NoError(t, err)
		assert.Nil(t, entries)
	})

	t.Run("limit zero returns all", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		entries, err := dlq.Get(testCtx(), 0)
		require.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("limit less than count", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 5 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		entries, err := dlq.Get(testCtx(), 2)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("limit greater than count", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 2 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		entries, err := dlq.Get(testCtx(), 10)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("skips malformed JSON lines", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		validEntry := wrappedEntry[testEntry]{
			ID:        "valid-1",
			Timestamp: time.Now(),
			Data:      sampleEntry(1),
		}
		validJSON, err := json.Marshal(validEntry)
		require.NoError(t, err)

		writeRawLines(t, sandbox, "deadletters.jsonl", []string{
			string(validJSON),
			"this is not valid json",
			"}{also broken",
		})

		entries, err := dlq.Get(testCtx(), 0)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Equal(t, sampleEntry(1), entries[0])
	})

	t.Run("error reading file", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)

		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		sandbox.ReadFileErr = errors.New("permission denied")

		_, err := dlq.Get(testCtx(), 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading dead letter file")
	})
}

func TestDiskDeadLetterQueue_Remove(t *testing.T) {
	t.Parallel()

	t.Run("empty entries slice does nothing", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		err := dlq.Remove(testCtx(), []testEntry{})
		require.NoError(t, err)
	})

	t.Run("rewrites file preserving all entries", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		err := dlq.Remove(testCtx(), []testEntry{sampleEntry(1)})
		require.NoError(t, err)

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("error reading file during getAllUnlocked", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		sandbox.ReadFileErr = errors.New("io error")

		err := dlq.Remove(testCtx(), []testEntry{sampleEntry(1)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading dead letter file")
	})

	t.Run("error writing temp file during rewrite", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		sandbox.WriteFileErr = errors.New("disk full")

		err := dlq.Remove(testCtx(), []testEntry{sampleEntry(1)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "writing temp file")
	})

	t.Run("error renaming during rewrite", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		sandbox.RenameErr = errors.New("rename failed")

		err := dlq.Remove(testCtx(), []testEntry{sampleEntry(1)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "replacing dead letter file")
	})
}

func TestDiskDeadLetterQueue_Count(t *testing.T) {
	t.Parallel()

	t.Run("file not found returns zero", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("counts lines correctly", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("error reading file", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		sandbox.ReadFileErr = errors.New("io error")

		_, err := dlq.Count(testCtx())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading dead letter file")
	})
}

func TestDiskDeadLetterQueue_Clear(t *testing.T) {
	t.Parallel()

	t.Run("removes file successfully", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 3 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		require.NoError(t, dlq.Clear(testCtx()))

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("clears non-existent file without error", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		require.NoError(t, dlq.Clear(testCtx()))
	})

	t.Run("error removing file", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		sandbox.RemoveErr = errors.New("permission denied")

		err := dlq.Clear(testCtx())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "clearing dead letter file")
	})
}

func TestDiskDeadLetterQueue_GetOlderThan(t *testing.T) {
	t.Parallel()

	t.Run("no old entries", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))

		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("filters correctly by timestamp", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		oldTime := time.Now().Add(-2 * time.Hour)
		newTime := time.Now()

		writeRawDiskEntries(t, sandbox, "deadletters.jsonl", []wrappedEntry[testEntry]{
			{ID: "old-1", Timestamp: oldTime, Data: sampleEntry(1)},
			{ID: "new-1", Timestamp: newTime, Data: sampleEntry(2)},
			{ID: "old-2", Timestamp: oldTime, Data: sampleEntry(3)},
		})

		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("error reading file", func(t *testing.T) {
		t.Parallel()

		dlq, sandbox := newTestDiskDLQ(t)
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))
		sandbox.ReadFileErr = errors.New("io error")

		_, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading dead letter file")
	})

	t.Run("empty file returns nil", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		entries, err := dlq.GetOlderThan(testCtx(), time.Hour)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

func TestDiskDeadLetterQueue_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("add then get returns same data", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		expected := sampleEntry(42)
		require.NoError(t, dlq.Add(testCtx(), expected))

		entries, err := dlq.Get(testCtx(), 0)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, expected, entries[0])
	})

	t.Run("add count clear count lifecycle", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)

		for i := range 5 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		count, err := dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 5, count)

		require.NoError(t, dlq.Clear(testCtx()))

		count, err = dlq.Count(testCtx())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("multiple adds preserve order", func(t *testing.T) {
		t.Parallel()

		dlq, _ := newTestDiskDLQ(t)
		for i := range 5 {
			require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		}

		entries, err := dlq.Get(testCtx(), 0)
		require.NoError(t, err)
		require.Len(t, entries, 5)

		for i, e := range entries {
			assert.Equal(t, sampleEntry(i), e)
		}
	})
}

func TestDiskDeadLetterQueue_ConcurrentAdds(t *testing.T) {
	t.Parallel()

	dlq, _ := newTestDiskDLQ(t)
	const goroutines = 20

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

func TestDiskDeadLetterQueue_GetAllUnlockedSkipsMalformed(t *testing.T) {
	t.Parallel()

	dlq, sandbox := newTestDiskDLQ(t)
	validEntry := wrappedEntry[testEntry]{
		ID:        "valid-1",
		Timestamp: time.Now(),
		Data:      sampleEntry(1),
	}
	validJSON, err := json.Marshal(validEntry)
	require.NoError(t, err)

	writeRawLines(t, sandbox, "deadletters.jsonl", []string{
		"not json at all",
		string(validJSON),
		`{"broken": true`,
	})

	entries, err := dlq.GetOlderThan(testCtx(), 0)
	require.NoError(t, err)

	assert.Len(t, entries, 1)
	assert.Equal(t, sampleEntry(1), entries[0])
}

func TestDiskDeadLetterQueue_RewriteEmptyEntries(t *testing.T) {
	t.Parallel()

	dlq, sandbox := newTestDiskDLQ(t)

	require.NoError(t, dlq.Add(testCtx(), sampleEntry(1)))

	require.NoError(t, dlq.Remove(testCtx(), []testEntry{sampleEntry(1)}))

	count, err := dlq.Count(testCtx())
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, readErr := sandbox.ReadFile("deadletters.jsonl.tmp")
	assert.Error(t, readErr, "temp file should have been renamed away")
}

func TestDLQ_AppendIsConstantTime(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sandbox, err := safedisk.NewSandbox(dir, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	raw := NewDiskDeadLetterQueue[testEntry](
		filepath.Join(dir, "deadletters.jsonl"),
		WithDeadLetterSandbox[testEntry](sandbox),
	)
	dlq, ok := raw.(*DiskDeadLetterQueue[testEntry])
	require.True(t, ok)

	const total = 1000
	const sampleWindow = 100

	timings := make([]time.Duration, 0, total)
	for i := range total {
		startTime := time.Now()
		require.NoError(t, dlq.Add(testCtx(), sampleEntry(i)))
		timings = append(timings, time.Since(startTime))
	}

	leadAverage := averageDuration(timings[:sampleWindow])
	tailAverage := averageDuration(timings[total-sampleWindow:])

	if leadAverage <= 0 {
		leadAverage = time.Microsecond
	}
	const allowedScalingFactor = 10.0
	ratio := float64(tailAverage) / float64(leadAverage)
	assert.Lessf(t, ratio, allowedScalingFactor,
		"Add latency degraded from %s (lead) to %s (tail) - possible regression to O(n^2) behaviour",
		leadAverage, tailAverage)
}

func averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func TestDLQ_RejectsAddPastMaxBytes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sandbox, err := safedisk.NewSandbox(dir, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	raw := NewDiskDeadLetterQueue[testEntry](
		filepath.Join(dir, "deadletters.jsonl"),
		WithDeadLetterSandbox[testEntry](sandbox),
		WithMaxDLQBytes[testEntry](512),
	)
	dlq, ok := raw.(*DiskDeadLetterQueue[testEntry])
	require.True(t, ok)

	var fullErr error
	for i := range 200 {
		err := dlq.Add(testCtx(), sampleEntry(i))
		if err != nil {
			fullErr = err
			break
		}
	}

	require.Error(t, fullErr, "expected at least one Add to fail after the cap")
	assert.ErrorIs(t, fullErr, ErrDLQFull)

	count, err := dlq.Count(testCtx())
	require.NoError(t, err)
	assert.Greater(t, count, 0, "some entries should have been written before hitting cap")
}

func TestDLQ_AddSurvivesProcessRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sandbox, err := safedisk.NewSandbox(dir, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sandbox.Close() })

	filePath := filepath.Join(dir, "deadletters.jsonl")
	raw := NewDiskDeadLetterQueue[testEntry](
		filePath,
		WithDeadLetterSandbox[testEntry](sandbox),
	)
	first, ok := raw.(*DiskDeadLetterQueue[testEntry])
	require.True(t, ok)

	for i := range 5 {
		require.NoError(t, first.Add(testCtx(), sampleEntry(i)))
	}

	rawAgain := NewDiskDeadLetterQueue[testEntry](
		filePath,
		WithDeadLetterSandbox[testEntry](sandbox),
	)
	second, ok := rawAgain.(*DiskDeadLetterQueue[testEntry])
	require.True(t, ok)

	entries, err := second.Get(testCtx(), 0)
	require.NoError(t, err)
	assert.Len(t, entries, 5)
}
