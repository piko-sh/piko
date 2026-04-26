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
	"hash/fnv"
	"sync"
	"time"
)

// PanelSnapshot captures the per-panel state that should survive panel
// switches and refreshes. Panels opt into snapshotting by implementing the
// Snapshotter interface; the model saves/restores snapshots through the
// configured PanelStateStore.
type PanelSnapshot struct {
	// LastRefresh is when the panel last received a data update. Used by
	// the store for diagnostics; not used for cursor/scroll restoration.
	LastRefresh time.Time

	// ExpandedKeys is the set of currently-expanded item identifiers in a
	// tree-style panel. Stable across refreshes when the same identifiers
	// reappear.
	ExpandedKeys map[string]bool

	// SearchQuery is the currently-applied filter or search query for the
	// panel.
	SearchQuery string

	// DataIdentity is a fingerprint of the items the panel is showing.
	// Restoring a snapshot whose DataIdentity matches the panel's current
	// items preserves the cursor; differing identities cause the cursor to
	// be clamped to the new range to avoid pointing at a removed item.
	DataIdentity uint64

	// Cursor is the saved cursor position in the panel's list.
	Cursor int

	// ScrollOffset is the saved scroll offset.
	ScrollOffset int
}

// Snapshotter is implemented by panels that participate in persistent
// per-panel state. The model calls Snapshot when focus leaves a panel and
// Restore when focus returns; DataIdentity is consulted to decide whether to
// keep the cursor or clamp it.
type Snapshotter interface {
	// Snapshot returns the current panel state.
	Snapshot() PanelSnapshot

	// Restore applies a previously-saved snapshot to the panel.
	Restore(snap PanelSnapshot)

	// DataIdentity returns a stable fingerprint of the currently-displayed
	// items so the restorer can detect data-identity changes.
	DataIdentity() uint64
}

// PanelStateStore persists per-panel snapshots keyed by panel ID. The store
// is owned by the service and shared across panel switches.
type PanelStateStore interface {
	// Save records a snapshot for the given panel ID.
	Save(panelID string, snap PanelSnapshot)

	// Load returns a previously-saved snapshot for the given panel ID, plus
	// a flag indicating whether one was found.
	Load(panelID string) (PanelSnapshot, bool)

	// Reset removes any saved snapshot for the given panel ID.
	Reset(panelID string)
}

// InMemoryPanelStateStore is the default PanelStateStore used by the model.
// Snapshots live for the lifetime of the service and are discarded when the
// process exits.
type InMemoryPanelStateStore struct {
	// snapshots holds the saved snapshots keyed by panel ID.
	snapshots map[string]PanelSnapshot

	// mu guards snapshots for safe concurrent access.
	mu sync.RWMutex
}

// NewInMemoryPanelStateStore creates a thread-safe in-memory store suitable
// for the duration of a TUI session.
//
// Returns *InMemoryPanelStateStore which is empty and ready to receive
// snapshots.
func NewInMemoryPanelStateStore() *InMemoryPanelStateStore {
	return &InMemoryPanelStateStore{
		snapshots: make(map[string]PanelSnapshot),
	}
}

// Save records snap for panelID.
//
// Takes panelID (string) which identifies the panel.
// Takes snap (PanelSnapshot) which is the state to record.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (s *InMemoryPanelStateStore) Save(panelID string, snap PanelSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[panelID] = snap
}

// Load returns the snapshot for panelID, if any.
//
// Takes panelID (string) which identifies the panel.
//
// Returns PanelSnapshot which is the saved state (zero value when missing).
// Returns bool which is true when a snapshot was found.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (s *InMemoryPanelStateStore) Load(panelID string) (PanelSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.snapshots[panelID]
	return snap, ok
}

// Reset clears the snapshot for panelID.
//
// Takes panelID (string) which identifies the panel.
//
// Concurrency: Safe for concurrent use; guarded by mu.
func (s *InMemoryPanelStateStore) Reset(panelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.snapshots, panelID)
}

// HashIdentifiers returns a stable FNV-1a 64 hash of the supplied
// identifiers. Panels use this to compute a DataIdentity for their current
// item set; matching identities indicate the cursor can be restored
// verbatim.
//
// Takes ids ([]string) which are the stable identifiers of the displayed
// items.
//
// Returns uint64 which is the combined hash; 0 for an empty slice.
func HashIdentifiers(ids []string) uint64 {
	if len(ids) == 0 {
		return 0
	}
	h := fnv.New64a()
	for _, id := range ids {
		_, _ = h.Write([]byte(id))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}
