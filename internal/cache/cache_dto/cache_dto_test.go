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

package cache_dto

import (
	"testing"
	"time"
)

func TestDeletionEvent_WasEvicted(t *testing.T) {
	tests := []struct {
		cause DeletionCause
		want  bool
	}{
		{cause: CauseInvalidation, want: false},
		{cause: CauseReplacement, want: false},
		{cause: CauseOverflow, want: true},
		{cause: CauseExpiration, want: true},
	}
	for _, tt := range tests {
		de := DeletionEvent[string, int]{Cause: tt.cause}
		if got := de.WasEvicted(); got != tt.want {
			t.Errorf("WasEvicted(cause=%d) = %v, want %v", tt.cause, got, tt.want)
		}
	}
}

func TestStats_HitRatio(t *testing.T) {
	tests := []struct {
		name   string
		hits   uint64
		misses uint64
		want   float64
	}{
		{name: "zero total", hits: 0, misses: 0, want: 0.0},
		{name: "all hits", hits: 100, misses: 0, want: 1.0},
		{name: "half", hits: 50, misses: 50, want: 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{Hits: tt.hits, Misses: tt.misses}
			if got := s.HitRatio(); got != tt.want {
				t.Errorf("HitRatio() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestStats_MissRatio(t *testing.T) {
	s := Stats{Hits: 75, Misses: 25}
	if got := s.MissRatio(); got != 0.25 {
		t.Errorf("MissRatio() = %f, want 0.25", got)
	}

	empty := Stats{}
	if got := empty.MissRatio(); got != 0.0 {
		t.Errorf("MissRatio() on empty = %f, want 0.0", got)
	}
}

func TestStats_AverageLoadPenalty(t *testing.T) {
	s := Stats{
		LoadSuccessCount: 8,
		LoadFailureCount: 2,
		TotalLoadTime:    10 * time.Second,
	}
	got := s.AverageLoadPenalty()
	if got != time.Second {
		t.Errorf("AverageLoadPenalty() = %v, want 1s", got)
	}

	empty := Stats{}
	if got := empty.AverageLoadPenalty(); got != 0 {
		t.Errorf("AverageLoadPenalty() on empty = %v, want 0", got)
	}
}

func TestSearchResult_Keys(t *testing.T) {
	r := SearchResult[string, int]{
		Items: []SearchHit[string, int]{
			{Key: "a", Value: 1},
			{Key: "b", Value: 2},
		},
	}
	keys := r.Keys()
	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Errorf("Keys() = %v, want [a b]", keys)
	}
}

func TestSearchResult_Values(t *testing.T) {
	r := SearchResult[string, int]{
		Items: []SearchHit[string, int]{
			{Key: "a", Value: 1},
			{Key: "b", Value: 2},
		},
	}
	values := r.Values()
	if len(values) != 2 || values[0] != 1 || values[1] != 2 {
		t.Errorf("Values() = %v, want [1 2]", values)
	}
}

func TestSearchResult_IsEmpty(t *testing.T) {
	empty := SearchResult[string, int]{}
	if !empty.IsEmpty() {
		t.Error("empty result should be IsEmpty")
	}

	nonEmpty := SearchResult[string, int]{
		Items: []SearchHit[string, int]{{Key: "a"}},
	}
	if nonEmpty.IsEmpty() {
		t.Error("non-empty result should not be IsEmpty")
	}
}

func TestSearchResult_HasMore(t *testing.T) {
	r := SearchResult[string, int]{
		Items:  []SearchHit[string, int]{{Key: "a"}},
		Total:  10,
		Offset: 0,
	}
	if !r.HasMore() {
		t.Error("HasMore() should be true when Total > Offset+len(Items)")
	}

	r2 := SearchResult[string, int]{
		Items:  []SearchHit[string, int]{{Key: "a"}},
		Total:  1,
		Offset: 0,
	}
	if r2.HasMore() {
		t.Error("HasMore() should be false when all items returned")
	}
}
