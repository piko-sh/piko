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
	"testing"
)

func TestNewHistoryRing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		capacity     int
		wantCapacity int
	}{
		{name: "positive capacity", capacity: 10, wantCapacity: 10},
		{name: "zero defaults", capacity: 0, wantCapacity: historyRingDefaultCapacity},
		{name: "negative defaults", capacity: -5, wantCapacity: historyRingDefaultCapacity},
		{name: "capacity of one", capacity: 1, wantCapacity: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ring := NewHistoryRing(tc.capacity)
			if ring.Capacity() != tc.wantCapacity {
				t.Errorf("Capacity() = %d, want %d", ring.Capacity(), tc.wantCapacity)
			}
			if ring.Len() != 0 {
				t.Errorf("Len() = %d, want 0", ring.Len())
			}
		})
	}
}

func TestHistoryRing_Append(t *testing.T) {
	t.Parallel()

	t.Run("single value", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		ring.Append(42.0)
		if ring.Len() != 1 {
			t.Errorf("Len() = %d, want 1", ring.Len())
		}
		if ring.Latest() != 42.0 {
			t.Errorf("Latest() = %f, want 42.0", ring.Latest())
		}
	})

	t.Run("fill to capacity", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(3)
		ring.Append(1.0)
		ring.Append(2.0)
		ring.Append(3.0)
		if ring.Len() != 3 {
			t.Errorf("Len() = %d, want 3", ring.Len())
		}
		want := []float64{1.0, 2.0, 3.0}
		got := ring.Values()
		assertFloat64Slice(t, got, want)
	})

	t.Run("overflow evicts oldest", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(3)
		ring.Append(1.0)
		ring.Append(2.0)
		ring.Append(3.0)
		ring.Append(4.0)
		ring.Append(5.0)
		if ring.Len() != 3 {
			t.Errorf("Len() = %d, want 3", ring.Len())
		}
		want := []float64{3.0, 4.0, 5.0}
		got := ring.Values()
		assertFloat64Slice(t, got, want)
	})

	t.Run("capacity of one", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(1)
		ring.Append(1.0)
		ring.Append(2.0)
		if ring.Len() != 1 {
			t.Errorf("Len() = %d, want 1", ring.Len())
		}
		if ring.Latest() != 2.0 {
			t.Errorf("Latest() = %f, want 2.0", ring.Latest())
		}
	})
}

func TestHistoryRing_AppendAll(t *testing.T) {
	t.Parallel()

	t.Run("batch within capacity", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		ring.AppendAll([]float64{1.0, 2.0, 3.0})
		if ring.Len() != 3 {
			t.Errorf("Len() = %d, want 3", ring.Len())
		}
		assertFloat64Slice(t, ring.Values(), []float64{1.0, 2.0, 3.0})
	})

	t.Run("batch exceeds capacity", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(3)
		ring.AppendAll([]float64{1.0, 2.0, 3.0, 4.0, 5.0})
		if ring.Len() != 3 {
			t.Errorf("Len() = %d, want 3", ring.Len())
		}
		assertFloat64Slice(t, ring.Values(), []float64{3.0, 4.0, 5.0})
	})

	t.Run("empty slice is no-op", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		ring.AppendAll(nil)
		if ring.Len() != 0 {
			t.Errorf("Len() = %d, want 0", ring.Len())
		}
	})
}

func TestHistoryRing_Values(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when empty", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		if ring.Values() != nil {
			t.Error("Values() should return nil for empty ring")
		}
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		ring.AppendAll([]float64{1.0, 2.0, 3.0})
		values := ring.Values()
		values[0] = 999.0
		if ring.Values()[0] != 1.0 {
			t.Error("Values() should return a copy; modifying it changed the original")
		}
	})
}

func TestHistoryRing_Latest(t *testing.T) {
	t.Parallel()

	t.Run("zero when empty", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		if ring.Latest() != 0 {
			t.Errorf("Latest() = %f, want 0", ring.Latest())
		}
	})

	t.Run("returns most recent", func(t *testing.T) {
		t.Parallel()
		ring := NewHistoryRing(5)
		ring.Append(10.0)
		ring.Append(20.0)
		ring.Append(30.0)
		if ring.Latest() != 30.0 {
			t.Errorf("Latest() = %f, want 30.0", ring.Latest())
		}
	})
}

func TestHistoryRing_Clear(t *testing.T) {
	t.Parallel()

	ring := NewHistoryRing(5)
	ring.AppendAll([]float64{1.0, 2.0, 3.0})
	ring.Clear()

	if ring.Len() != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", ring.Len())
	}
	if ring.Capacity() != 5 {
		t.Errorf("Capacity() after Clear() = %d, want 5", ring.Capacity())
	}
	if ring.Values() != nil {
		t.Error("Values() after Clear() should return nil")
	}
}

func TestHistoryRing_Stats(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		values      []float64
		wantMin     float64
		wantMax     float64
		wantAverage float64
	}{
		{
			name:    "empty buffer",
			values:  nil,
			wantMin: 0, wantMax: 0, wantAverage: 0,
		},
		{
			name:    "single value",
			values:  []float64{5.0},
			wantMin: 5.0, wantMax: 5.0, wantAverage: 5.0,
		},
		{
			name:    "varied values",
			values:  []float64{1.0, 5.0, 3.0, 2.0, 4.0},
			wantMin: 1.0, wantMax: 5.0, wantAverage: 3.0,
		},
		{
			name:    "identical values",
			values:  []float64{7.0, 7.0, 7.0},
			wantMin: 7.0, wantMax: 7.0, wantAverage: 7.0,
		},
		{
			name:    "negative values",
			values:  []float64{-3.0, -1.0, -2.0},
			wantMin: -3.0, wantMax: -1.0, wantAverage: -2.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ring := NewHistoryRing(10)
			ring.AppendAll(tc.values)
			gotMin, gotMax, gotAverage := ring.Stats()
			if gotMin != tc.wantMin {
				t.Errorf("min = %f, want %f", gotMin, tc.wantMin)
			}
			if gotMax != tc.wantMax {
				t.Errorf("max = %f, want %f", gotMax, tc.wantMax)
			}
			if gotAverage != tc.wantAverage {
				t.Errorf("avg = %f, want %f", gotAverage, tc.wantAverage)
			}
		})
	}
}

func assertFloat64Slice(t *testing.T, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d = %f, want %f", i, got[i], want[i])
		}
	}
}
