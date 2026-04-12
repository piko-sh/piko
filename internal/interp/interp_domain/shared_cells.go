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

package interp_domain

import (
	"piko.sh/piko/wdk/stablepool"
)

// A frame's sharedCells deduplicates upvalueCell pointers across the closures the frame
// creates. Typical N is 1-4: a handful of locals captured by one or two closures.
//
// sharedCellMap holds the entries in a fixed inline array plus a slice overflow for the
// rare case where many cells are captured. Linear scan in the inline array is competitive
// with map lookup at N <= 8 and substantially cheaper allocations-wise.
//
// The callFrame field is a single pointer to this struct, matching the 8-byte field size
// required by asm dispatch offsets. The *sharedCellMap is allocated lazily on first
// closure creation in a frame and pooled so a long-running interpreter session amortises
// the allocation across frame churn.

const (
	// inlineSharedCellsCapacity is the number of {key, cell} pairs stored inline in
	// sharedCellMap. Sized to the 95th-percentile closure-capture count observed in typical
	// Go programs; closures capturing more than this fall back to the overflow slice.
	inlineSharedCellsCapacity = 8
)

// sharedCellEntry is one (key, cell) pair held by the overflow slice. The inline portion
// of sharedCellMap stores keys and cells as parallel arrays to keep the inline allocation
// tight.
type sharedCellEntry struct {
	// cell is the pointer to the deduplicated upvalueCell.
	cell *upvalueCell

	// key is the joinWide(kind, regIndex) identifier for the cell.
	key uint16
}

// sharedCellMap deduplicates upvalueCell pointers per frame. Used by handleMakeClosure to
// avoid allocating a fresh cell when a previous closure (from the same frame) has already
// captured the same (kind, register) pair.
//
// Read by syncSharedCellsToNamedResults at frame finalisation. Both accesses are
// linear-scan; for N <= inlineSharedCellsCapacity (the overwhelming common case) every
// lookup hits a single cache line.
type sharedCellMap struct {
	// link is the intrusive header for stablepool. MUST be the first field.
	stablepool.Link

	// inlineCells stores the cell pointers paired by index with inlineKeys for the first
	// inlineSharedCellsCapacity entries.
	inlineCells [inlineSharedCellsCapacity]*upvalueCell

	// overflow holds extra entries when inline storage fills. Nil for the typical small-N
	// case so popFrame can detect a "needs slice-clearing" path cheaply.
	overflow []sharedCellEntry

	// inlineLen counts how many of the inline slots are in use. Always <=
	// inlineSharedCellsCapacity.
	inlineLen int

	// inlineKeys / inlineCells store the first inlineSharedCellsCapacity entries inline.
	// They are kept as parallel arrays rather than a [N]sharedCellEntry array so the key
	// scan can vectorise without touching the cell pointers; the keys are 16-bit ints packed
	// densely.
	inlineKeys [inlineSharedCellsCapacity]uint16
}

var (
	// sharedCellMapPool amortises sharedCellMap allocations across frame churn. Backed by
	// stablepool so the inline-cell arrays and overflow slice capacity survive Go GC cycles.
	//
	// Pre-warm 128 (closure-creation is common enough to justify); growth to 1024 for
	// closure-heavy workloads. The cleaner is nil because releaseSharedCellMap performs the
	// clearing explicitly before Put.
	//
	//nolint:gochecknoglobals // pool by definition is package-level
	sharedCellMapPool, _ = stablepool.New[sharedCellMap](
		func(m *sharedCellMap) { *m = sharedCellMap{} },
		nil,
		128,
		stablepool.WithGrowth[sharedCellMap](1024),
	)
)

// get returns the upvalueCell previously stored under key, or nil and false if not
// present. Inline storage is scanned first; the hot path of a frame creating only a
// handful of closures never touches m.overflow.
//
// Takes key (uint16) which is the joinWide(kind, regIndex) lookup key produced by
// handleMakeClosure.
//
// Returns the stored cell pointer and whether the key was found.
func (m *sharedCellMap) get(key uint16) (*upvalueCell, bool) {
	for i := range m.inlineLen {
		if m.inlineKeys[i] == key {
			return m.inlineCells[i], true
		}
	}
	for i := range m.overflow {
		if m.overflow[i].key == key {
			return m.overflow[i].cell, true
		}
	}
	return nil, false
}

// set installs cell under key.
//
// Overwrites any existing entry for the same key. New entries fill inline storage first,
// then spill into overflow.
//
// Takes key (uint16) which is the joinWide(kind, regIndex) lookup key.
// Takes cell (*upvalueCell) which is the cell to associate with the key.
func (m *sharedCellMap) set(key uint16, cell *upvalueCell) {
	for i := range m.inlineLen {
		if m.inlineKeys[i] == key {
			m.inlineCells[i] = cell
			return
		}
	}
	for i := range m.overflow {
		if m.overflow[i].key == key {
			m.overflow[i].cell = cell
			return
		}
	}
	if m.inlineLen < inlineSharedCellsCapacity {
		m.inlineKeys[m.inlineLen] = key
		m.inlineCells[m.inlineLen] = cell
		m.inlineLen++
		return
	}
	m.overflow = append(m.overflow, sharedCellEntry{key: key, cell: cell})
}

// remove deletes the entry for key, mirroring Go's delete(map, key).
//
// A no-op when key is absent. Maintains the "no gaps" invariant of inline storage by
// swapping the last in-use slot into the removed position.
//
// Takes key (uint16) which is the entry to drop.
func (m *sharedCellMap) remove(key uint16) {
	for i := range m.inlineLen {
		if m.inlineKeys[i] == key {
			last := m.inlineLen - 1
			m.inlineKeys[i] = m.inlineKeys[last]
			m.inlineCells[i] = m.inlineCells[last]
			m.inlineCells[last] = nil
			m.inlineLen = last
			return
		}
	}
	for i := range m.overflow {
		if m.overflow[i].key == key {
			last := len(m.overflow) - 1
			m.overflow[i] = m.overflow[last]
			m.overflow[last] = sharedCellEntry{}
			m.overflow = m.overflow[:last]
			return
		}
	}
}

// has reports whether key is present without returning the cell. Used by closure-return
// synchronisation where the cell pointer isn't needed, only its existence.
//
// Takes key (uint16) which is the lookup key.
//
// Returns true when key has an associated cell.
func (m *sharedCellMap) has(key uint16) bool {
	for i := range m.inlineLen {
		if m.inlineKeys[i] == key {
			return true
		}
	}
	for i := range m.overflow {
		if m.overflow[i].key == key {
			return true
		}
	}
	return false
}

// acquireSharedCellMap rents a cleared *sharedCellMap from the pool. Used by
// handleMakeClosure on first closure creation in a frame.
//
// Returns a pointer to a sharedCellMap with inlineLen=0 and no overflow entries.
func acquireSharedCellMap() *sharedCellMap {
	return sharedCellMapPool.MustGet()
}

// releaseSharedCellMap clears m and returns it to the pool.
//
// Called by popFrame when the frame's sharedCells pointer is being dropped. Idempotent
// against nil for caller convenience. Only the in-use inline slots are zeroed, dropping
// pointer references without touching cold cache lines.
//
// Takes m (*sharedCellMap) which may be nil. When non-nil the inline cell pointers are
// zeroed so the GC can reclaim the pointed-at upvalueCells once no closure references
// them, and the overflow slice is truncated (keeping its capacity).
func releaseSharedCellMap(m *sharedCellMap) {
	if m == nil {
		return
	}
	clear(m.inlineCells[:m.inlineLen])
	m.inlineLen = 0
	if len(m.overflow) > 0 {
		clear(m.overflow)
		m.overflow = m.overflow[:0]
	}
	sharedCellMapPool.Put(m)
}
