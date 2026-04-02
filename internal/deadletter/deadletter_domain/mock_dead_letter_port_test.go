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

package deadletter_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEntry struct {
	Timestamp time.Time
	ID        string
}

func (e *testEntry) GetID() string            { return e.ID }
func (e *testEntry) SetID(id string)          { e.ID = id }
func (e *testEntry) GetTimestamp() time.Time  { return e.Timestamp }
func (e *testEntry) SetTimestamp(t time.Time) { e.Timestamp = t }

func newTestEntry(id string) *testEntry {
	return &testEntry{
		ID:        id,
		Timestamp: time.Now(),
	}
}

func TestMockDeadLetterPort_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expectedErr   error
		addFunc       func(ctx context.Context, entry *testEntry) error
		name          string
		expectedCalls int64
	}{
		{
			name:          "nil AddFunc returns nil error",
			addFunc:       nil,
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to AddFunc and returns nil error",
			addFunc: func(_ context.Context, _ *testEntry) error {
				return nil
			},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to AddFunc and returns error",
			addFunc: func(_ context.Context, _ *testEntry) error {
				return errors.New("add failed")
			},
			expectedErr:   errors.New("add failed"),
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockDeadLetterPort[*testEntry]{
				AddFunc:               tt.addFunc,
				GetFunc:               nil,
				RemoveFunc:            nil,
				CountFunc:             nil,
				ClearFunc:             nil,
				GetOlderThanFunc:      nil,
				AddCallCount:          0,
				GetCallCount:          0,
				RemoveCallCount:       0,
				CountCallCount:        0,
				ClearCallCount:        0,
				GetOlderThanCallCount: 0,
			}

			ctx := context.Background()
			entry := newTestEntry("test-1")

			err := mock.Add(ctx, entry)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCalls, atomic.LoadInt64(&mock.AddCallCount))
		})
	}
}

func TestMockDeadLetterPort_Add_MultipleCalls(t *testing.T) {
	t.Parallel()

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc:               nil,
		GetFunc:               nil,
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()

	for range 5 {
		err := mock.Add(ctx, newTestEntry("entry"))
		require.NoError(t, err)
	}

	assert.Equal(t, int64(5), atomic.LoadInt64(&mock.AddCallCount))
}

func TestMockDeadLetterPort_Add_PassesArguments(t *testing.T) {
	t.Parallel()

	var capturedCtx context.Context
	var capturedEntry *testEntry

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc: func(ctx context.Context, entry *testEntry) error {
			capturedCtx = ctx
			capturedEntry = entry
			return nil
		},
		GetFunc:               nil,
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.WithValue(context.Background(), testContextKey{}, "test-value")
	entry := newTestEntry("captured-entry")

	err := mock.Add(ctx, entry)

	require.NoError(t, err)
	assert.Equal(t, ctx, capturedCtx)
	assert.Equal(t, entry, capturedEntry)
}

type testContextKey struct{}

func TestMockDeadLetterPort_Get(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		expectedErr    error
		getFunc        func(ctx context.Context, limit int) ([]*testEntry, error)
		name           string
		expectedResult []*testEntry
		limit          int
		expectedCalls  int64
	}{
		{
			name:           "nil GetFunc returns nil slice and nil error",
			getFunc:        nil,
			limit:          10,
			expectedResult: nil,
			expectedErr:    nil,
			expectedCalls:  1,
		},
		{
			name: "delegates to GetFunc and returns entries",
			getFunc: func(_ context.Context, limit int) ([]*testEntry, error) {
				result := make([]*testEntry, 0, limit)
				for range limit {
					result = append(result, &testEntry{
						ID:        "entry",
						Timestamp: now,
					})
				}
				return result, nil
			},
			limit: 3,
			expectedResult: []*testEntry{
				{ID: "entry", Timestamp: now},
				{ID: "entry", Timestamp: now},
				{ID: "entry", Timestamp: now},
			},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to GetFunc and returns error",
			getFunc: func(_ context.Context, _ int) ([]*testEntry, error) {
				return nil, errors.New("get failed")
			},
			limit:          5,
			expectedResult: nil,
			expectedErr:    errors.New("get failed"),
			expectedCalls:  1,
		},
		{
			name: "delegates to GetFunc with empty result",
			getFunc: func(_ context.Context, _ int) ([]*testEntry, error) {
				return []*testEntry{}, nil
			},
			limit:          10,
			expectedResult: []*testEntry{},
			expectedErr:    nil,
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockDeadLetterPort[*testEntry]{
				AddFunc:               nil,
				GetFunc:               tt.getFunc,
				RemoveFunc:            nil,
				CountFunc:             nil,
				ClearFunc:             nil,
				GetOlderThanFunc:      nil,
				AddCallCount:          0,
				GetCallCount:          0,
				RemoveCallCount:       0,
				CountCallCount:        0,
				ClearCallCount:        0,
				GetOlderThanCallCount: 0,
			}

			ctx := context.Background()

			result, err := mock.Get(ctx, tt.limit)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedCalls, atomic.LoadInt64(&mock.GetCallCount))
		})
	}
}

func TestMockDeadLetterPort_Get_PassesLimit(t *testing.T) {
	t.Parallel()

	var capturedLimit int

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc: nil,
		GetFunc: func(_ context.Context, limit int) ([]*testEntry, error) {
			capturedLimit = limit
			return nil, nil
		},
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()

	_, err := mock.Get(ctx, 42)

	require.NoError(t, err)
	assert.Equal(t, 42, capturedLimit)
}

func TestMockDeadLetterPort_Remove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expectedErr   error
		removeFunc    func(ctx context.Context, entries []*testEntry) error
		name          string
		entries       []*testEntry
		expectedCalls int64
	}{
		{
			name:          "nil RemoveFunc returns nil error",
			removeFunc:    nil,
			entries:       []*testEntry{newTestEntry("e1")},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to RemoveFunc and returns nil error",
			removeFunc: func(_ context.Context, _ []*testEntry) error {
				return nil
			},
			entries:       []*testEntry{newTestEntry("e1"), newTestEntry("e2")},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to RemoveFunc and returns error",
			removeFunc: func(_ context.Context, _ []*testEntry) error {
				return errors.New("remove failed")
			},
			entries:       []*testEntry{newTestEntry("e1")},
			expectedErr:   errors.New("remove failed"),
			expectedCalls: 1,
		},
		{
			name:          "nil RemoveFunc with empty entries returns nil error",
			removeFunc:    nil,
			entries:       []*testEntry{},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name:          "nil RemoveFunc with nil entries returns nil error",
			removeFunc:    nil,
			entries:       nil,
			expectedErr:   nil,
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockDeadLetterPort[*testEntry]{
				AddFunc:               nil,
				GetFunc:               nil,
				RemoveFunc:            tt.removeFunc,
				CountFunc:             nil,
				ClearFunc:             nil,
				GetOlderThanFunc:      nil,
				AddCallCount:          0,
				GetCallCount:          0,
				RemoveCallCount:       0,
				CountCallCount:        0,
				ClearCallCount:        0,
				GetOlderThanCallCount: 0,
			}

			ctx := context.Background()

			err := mock.Remove(ctx, tt.entries)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCalls, atomic.LoadInt64(&mock.RemoveCallCount))
		})
	}
}

func TestMockDeadLetterPort_Remove_PassesEntries(t *testing.T) {
	t.Parallel()

	var capturedEntries []*testEntry

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc: nil,
		GetFunc: nil,
		RemoveFunc: func(_ context.Context, entries []*testEntry) error {
			capturedEntries = entries
			return nil
		},
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()
	entries := []*testEntry{newTestEntry("a"), newTestEntry("b"), newTestEntry("c")}

	err := mock.Remove(ctx, entries)

	require.NoError(t, err)
	require.Len(t, capturedEntries, 3)
	assert.Equal(t, "a", capturedEntries[0].GetID())
	assert.Equal(t, "b", capturedEntries[1].GetID())
	assert.Equal(t, "c", capturedEntries[2].GetID())
}

func TestMockDeadLetterPort_Count(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expectedErr   error
		countFunc     func(ctx context.Context) (int, error)
		name          string
		expectedCount int
		expectedCalls int64
	}{
		{
			name:          "nil CountFunc returns zero and nil error",
			countFunc:     nil,
			expectedCount: 0,
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to CountFunc and returns count",
			countFunc: func(_ context.Context) (int, error) {
				return 42, nil
			},
			expectedCount: 42,
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to CountFunc and returns error",
			countFunc: func(_ context.Context) (int, error) {
				return 0, errors.New("count failed")
			},
			expectedCount: 0,
			expectedErr:   errors.New("count failed"),
			expectedCalls: 1,
		},
		{
			name: "delegates to CountFunc with large count",
			countFunc: func(_ context.Context) (int, error) {
				return 999999, nil
			},
			expectedCount: 999999,
			expectedErr:   nil,
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockDeadLetterPort[*testEntry]{
				AddFunc:               nil,
				GetFunc:               nil,
				RemoveFunc:            nil,
				CountFunc:             tt.countFunc,
				ClearFunc:             nil,
				GetOlderThanFunc:      nil,
				AddCallCount:          0,
				GetCallCount:          0,
				RemoveCallCount:       0,
				CountCallCount:        0,
				ClearCallCount:        0,
				GetOlderThanCallCount: 0,
			}

			ctx := context.Background()

			count, err := mock.Count(ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCount, count)
			assert.Equal(t, tt.expectedCalls, atomic.LoadInt64(&mock.CountCallCount))
		})
	}
}

func TestMockDeadLetterPort_Clear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expectedErr   error
		clearFunc     func(ctx context.Context) error
		name          string
		expectedCalls int64
	}{
		{
			name:          "nil ClearFunc returns nil error",
			clearFunc:     nil,
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to ClearFunc and returns nil error",
			clearFunc: func(_ context.Context) error {
				return nil
			},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to ClearFunc and returns error",
			clearFunc: func(_ context.Context) error {
				return errors.New("clear failed")
			},
			expectedErr:   errors.New("clear failed"),
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockDeadLetterPort[*testEntry]{
				AddFunc:               nil,
				GetFunc:               nil,
				RemoveFunc:            nil,
				CountFunc:             nil,
				ClearFunc:             tt.clearFunc,
				GetOlderThanFunc:      nil,
				AddCallCount:          0,
				GetCallCount:          0,
				RemoveCallCount:       0,
				CountCallCount:        0,
				ClearCallCount:        0,
				GetOlderThanCallCount: 0,
			}

			ctx := context.Background()

			err := mock.Clear(ctx)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCalls, atomic.LoadInt64(&mock.ClearCallCount))
		})
	}
}

func TestMockDeadLetterPort_GetOlderThan(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		expectedErr    error
		getOlderFunc   func(ctx context.Context, duration time.Duration) ([]*testEntry, error)
		name           string
		expectedResult []*testEntry
		duration       time.Duration
		expectedCalls  int64
	}{
		{
			name:           "nil GetOlderThanFunc returns nil slice and nil error",
			getOlderFunc:   nil,
			duration:       time.Hour,
			expectedResult: nil,
			expectedErr:    nil,
			expectedCalls:  1,
		},
		{
			name: "delegates to GetOlderThanFunc and returns entries",
			getOlderFunc: func(_ context.Context, _ time.Duration) ([]*testEntry, error) {
				return []*testEntry{
					{ID: "old-1", Timestamp: now},
					{ID: "old-2", Timestamp: now},
				}, nil
			},
			duration: 24 * time.Hour,
			expectedResult: []*testEntry{
				{ID: "old-1", Timestamp: now},
				{ID: "old-2", Timestamp: now},
			},
			expectedErr:   nil,
			expectedCalls: 1,
		},
		{
			name: "delegates to GetOlderThanFunc and returns error",
			getOlderFunc: func(_ context.Context, _ time.Duration) ([]*testEntry, error) {
				return nil, errors.New("get older than failed")
			},
			duration:       time.Minute,
			expectedResult: nil,
			expectedErr:    errors.New("get older than failed"),
			expectedCalls:  1,
		},
		{
			name: "delegates to GetOlderThanFunc with empty result",
			getOlderFunc: func(_ context.Context, _ time.Duration) ([]*testEntry, error) {
				return []*testEntry{}, nil
			},
			duration:       time.Second,
			expectedResult: []*testEntry{},
			expectedErr:    nil,
			expectedCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &MockDeadLetterPort[*testEntry]{
				AddFunc:               nil,
				GetFunc:               nil,
				RemoveFunc:            nil,
				CountFunc:             nil,
				ClearFunc:             nil,
				GetOlderThanFunc:      tt.getOlderFunc,
				AddCallCount:          0,
				GetCallCount:          0,
				RemoveCallCount:       0,
				CountCallCount:        0,
				ClearCallCount:        0,
				GetOlderThanCallCount: 0,
			}

			ctx := context.Background()

			result, err := mock.GetOlderThan(ctx, tt.duration)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)
			assert.Equal(t, tt.expectedCalls, atomic.LoadInt64(&mock.GetOlderThanCallCount))
		})
	}
}

func TestMockDeadLetterPort_GetOlderThan_PassesDuration(t *testing.T) {
	t.Parallel()

	var capturedDuration time.Duration

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc:    nil,
		GetFunc:    nil,
		RemoveFunc: nil,
		CountFunc:  nil,
		ClearFunc:  nil,
		GetOlderThanFunc: func(_ context.Context, duration time.Duration) ([]*testEntry, error) {
			capturedDuration = duration
			return nil, nil
		},
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()

	_, err := mock.GetOlderThan(ctx, 48*time.Hour)

	require.NoError(t, err)
	assert.Equal(t, 48*time.Hour, capturedDuration)
}

func TestMockDeadLetterPort_CallCountsAreIndependent(t *testing.T) {
	t.Parallel()

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc:               nil,
		GetFunc:               nil,
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()

	_ = mock.Add(ctx, newTestEntry("e1"))
	_ = mock.Add(ctx, newTestEntry("e2"))
	_, _ = mock.Get(ctx, 10)
	_ = mock.Remove(ctx, nil)
	_, _ = mock.Count(ctx)
	_, _ = mock.Count(ctx)
	_, _ = mock.Count(ctx)
	_ = mock.Clear(ctx)
	_, _ = mock.GetOlderThan(ctx, time.Hour)
	_, _ = mock.GetOlderThan(ctx, time.Hour)

	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.AddCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveCallCount))
	assert.Equal(t, int64(3), atomic.LoadInt64(&mock.CountCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ClearCallCount))
	assert.Equal(t, int64(2), atomic.LoadInt64(&mock.GetOlderThanCallCount))
}

func TestMockDeadLetterPort_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc:               nil,
		GetFunc:               nil,
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 6)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.Add(ctx, newTestEntry("concurrent"))
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.Get(ctx, 10)
		}()
		go func() {
			defer wg.Done()
			_ = mock.Remove(ctx, nil)
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.Count(ctx)
		}()
		go func() {
			defer wg.Done()
			_ = mock.Clear(ctx)
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.GetOlderThan(ctx, time.Hour)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.AddCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.RemoveCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.CountCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ClearCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GetOlderThanCallCount))
}

func TestMockDeadLetterPort_ImplementsDeadLetterPort(t *testing.T) {
	t.Parallel()

	mock := &MockDeadLetterPort[*testEntry]{
		AddFunc:               nil,
		GetFunc:               nil,
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	var _ DeadLetterPort[*testEntry] = mock
}

func TestMockDeadLetterPort_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockDeadLetterPort[*testEntry]

	ctx := context.Background()
	entry := newTestEntry("zero-value")

	err := mock.Add(ctx, entry)
	require.NoError(t, err)

	result, err := mock.Get(ctx, 10)
	require.NoError(t, err)
	assert.Nil(t, result)

	err = mock.Remove(ctx, []*testEntry{entry})
	require.NoError(t, err)

	count, err := mock.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	err = mock.Clear(ctx)
	require.NoError(t, err)

	older, err := mock.GetOlderThan(ctx, time.Hour)
	require.NoError(t, err)
	assert.Nil(t, older)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.AddCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RemoveCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CountCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ClearCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GetOlderThanCallCount))
}

func TestMockDeadLetterPort_WithStringType(t *testing.T) {
	t.Parallel()

	var captured string

	mock := &MockDeadLetterPort[string]{
		AddFunc: func(_ context.Context, entry string) error {
			captured = entry
			return nil
		},
		GetFunc: func(_ context.Context, _ int) ([]string, error) {
			return []string{"alpha", "beta", "gamma"}, nil
		},
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()

	err := mock.Add(ctx, "hello-deadletter")
	require.NoError(t, err)
	assert.Equal(t, "hello-deadletter", captured)

	results, err := mock.Get(ctx, 5)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta", "gamma"}, results)
}

func TestMockDeadLetterPort_WithIntType(t *testing.T) {
	t.Parallel()

	mock := &MockDeadLetterPort[int]{
		AddFunc: nil,
		GetFunc: func(_ context.Context, limit int) ([]int, error) {
			result := make([]int, 0, limit)
			for i := range limit {
				result = append(result, i*10)
			}
			return result, nil
		},
		RemoveFunc:            nil,
		CountFunc:             nil,
		ClearFunc:             nil,
		GetOlderThanFunc:      nil,
		AddCallCount:          0,
		GetCallCount:          0,
		RemoveCallCount:       0,
		CountCallCount:        0,
		ClearCallCount:        0,
		GetOlderThanCallCount: 0,
	}

	ctx := context.Background()

	results, err := mock.Get(ctx, 4)

	require.NoError(t, err)
	assert.Equal(t, []int{0, 10, 20, 30}, results)
}

func TestTestEntry_ImplementsEntry(t *testing.T) {
	t.Parallel()

	entry := &testEntry{
		ID:        "",
		Timestamp: time.Time{},
	}

	var _ Entry = entry
}

func TestTestEntry_GetID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{
			name:     "returns non-empty ID",
			id:       "abc-123",
			expected: "abc-123",
		},
		{
			name:     "returns empty ID",
			id:       "",
			expected: "",
		},
		{
			name:     "returns UUID-style ID",
			id:       "550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := &testEntry{
				ID:        tt.id,
				Timestamp: time.Time{},
			}

			assert.Equal(t, tt.expected, entry.GetID())
		})
	}
}

func TestTestEntry_SetID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		initial  string
		newID    string
		expected string
	}{
		{
			name:     "sets new ID on empty entry",
			initial:  "",
			newID:    "new-id",
			expected: "new-id",
		},
		{
			name:     "overwrites existing ID",
			initial:  "old-id",
			newID:    "new-id",
			expected: "new-id",
		},
		{
			name:     "sets empty ID",
			initial:  "has-id",
			newID:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := &testEntry{
				ID:        tt.initial,
				Timestamp: time.Time{},
			}

			entry.SetID(tt.newID)

			assert.Equal(t, tt.expected, entry.GetID())
		})
	}
}

func TestTestEntry_GetTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now()
	zero := time.Time{}

	tests := []struct {
		timestamp time.Time
		expected  time.Time
		name      string
	}{
		{
			name:      "returns current time",
			timestamp: now,
			expected:  now,
		},
		{
			name:      "returns zero time",
			timestamp: zero,
			expected:  zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := &testEntry{
				ID:        "",
				Timestamp: tt.timestamp,
			}

			assert.Equal(t, tt.expected, entry.GetTimestamp())
		})
	}
}

func TestTestEntry_SetTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now()
	later := now.Add(time.Hour)

	tests := []struct {
		initial      time.Time
		newTimestamp time.Time
		expected     time.Time
		name         string
	}{
		{
			name:         "sets timestamp on zero entry",
			initial:      time.Time{},
			newTimestamp: now,
			expected:     now,
		},
		{
			name:         "overwrites existing timestamp",
			initial:      now,
			newTimestamp: later,
			expected:     later,
		},
		{
			name:         "sets zero timestamp",
			initial:      now,
			newTimestamp: time.Time{},
			expected:     time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entry := &testEntry{
				ID:        "",
				Timestamp: tt.initial,
			}

			entry.SetTimestamp(tt.newTimestamp)

			assert.Equal(t, tt.expected, entry.GetTimestamp())
		})
	}
}
