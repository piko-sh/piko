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
)

func TestInMemoryPanelStateStoreRoundTrip(t *testing.T) {
	store := NewInMemoryPanelStateStore()
	now := time.Now()

	in := PanelSnapshot{
		LastRefresh:  now,
		ExpandedKeys: map[string]bool{"a": true},
		SearchQuery:  "needle",
		DataIdentity: 12345,
		Cursor:       4,
		ScrollOffset: 2,
	}
	store.Save("p1", in)

	out, ok := store.Load("p1")
	if !ok {
		t.Fatalf("Load returned ok=false after Save")
	}
	if out.Cursor != in.Cursor || out.ScrollOffset != in.ScrollOffset {
		t.Errorf("cursor/scroll round-trip mismatch: got %+v, want %+v", out, in)
	}
	if out.DataIdentity != in.DataIdentity {
		t.Errorf("DataIdentity round-trip mismatch: got %d, want %d", out.DataIdentity, in.DataIdentity)
	}
	if out.SearchQuery != "needle" {
		t.Errorf("SearchQuery round-trip mismatch: %q", out.SearchQuery)
	}
	if !out.ExpandedKeys["a"] {
		t.Error("ExpandedKeys round-trip mismatch")
	}
}

func TestInMemoryPanelStateStoreReset(t *testing.T) {
	store := NewInMemoryPanelStateStore()
	store.Save("p1", PanelSnapshot{Cursor: 1})

	store.Reset("p1")
	if _, ok := store.Load("p1"); ok {
		t.Errorf("snapshot present after Reset")
	}
}

func TestInMemoryPanelStateStoreLoadMissing(t *testing.T) {
	store := NewInMemoryPanelStateStore()
	if _, ok := store.Load("missing"); ok {
		t.Errorf("Load returned ok=true for unseen panel ID")
	}
}

func TestInMemoryPanelStateStoreConcurrent(t *testing.T) {
	store := NewInMemoryPanelStateStore()

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Go(func() {
			store.Save("p", PanelSnapshot{Cursor: i})
		})
		wg.Go(func() {
			_, _ = store.Load("p")
		})
	}
	wg.Wait()
}

func TestHashIdentifiersStable(t *testing.T) {
	ids := []string{"alpha", "beta", "gamma"}
	a := HashIdentifiers(ids)
	b := HashIdentifiers(ids)
	if a != b {
		t.Errorf("HashIdentifiers not stable: %d vs %d", a, b)
	}
	if a == 0 {
		t.Errorf("expected non-zero hash for non-empty input")
	}
}

func TestHashIdentifiersOrderSensitive(t *testing.T) {
	a := HashIdentifiers([]string{"alpha", "beta"})
	b := HashIdentifiers([]string{"beta", "alpha"})
	if a == b {
		t.Errorf("expected different hashes for reordered input, got %d", a)
	}
}

func TestHashIdentifiersEmpty(t *testing.T) {
	if got := HashIdentifiers(nil); got != 0 {
		t.Errorf("HashIdentifiers(nil) = %d, want 0", got)
	}
	if got := HashIdentifiers([]string{}); got != 0 {
		t.Errorf("HashIdentifiers([]) = %d, want 0", got)
	}
}

func TestHashIdentifiersDelimiterPrefixCollision(t *testing.T) {
	a := HashIdentifiers([]string{"ab", "c"})
	b := HashIdentifiers([]string{"a", "bc"})
	if a == b {
		t.Errorf("delimiter failed: prefix collision between ['ab','c'] and ['a','bc']: %d", a)
	}
}
