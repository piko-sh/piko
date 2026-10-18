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

package otter

import (
	"time"

	"github.com/google/btree"
)

const (
	// timeIndexDegree is the B-tree degree of a timeIndex.
	timeIndexDegree = 32
)

// timeEntry pairs a task ID with its indexed Unix timestamp in nanoseconds.
type timeEntry struct {
	// id identifies the task.
	id string

	// at is the indexed time in Unix nanoseconds.
	at int64
}

// lessTimeEntry orders entries by time, then by ID.
//
// Takes a (timeEntry) which is the left entry.
// Takes b (timeEntry) which is the right entry.
//
// Returns bool which is true when a sorts before b.
func lessTimeEntry(a, b timeEntry) bool {
	if a.at != b.at {
		return a.at < b.at
	}
	return a.id < b.id
}

// timeIndex orders task IDs by timestamp.
//
// The DAL lock protects it; writes require the write lock and reads accept either lock.
type timeIndex struct {
	// tree holds the entries in time and ID order.
	tree *btree.BTreeG[timeEntry]

	// at maps each indexed ID to its time, to find its entry when it is removed.
	at map[string]int64
}

// newTimeIndex returns an empty timeIndex.
//
// Returns *timeIndex which is ready for use.
func newTimeIndex() *timeIndex {
	return &timeIndex{
		tree: btree.NewG(timeIndexDegree, lessTimeEntry),
		at:   make(map[string]int64),
	}
}

// Add indexes id at t, moving it if it is already indexed.
//
// Takes id (string) which identifies the task.
// Takes t (time.Time) which is the time to index it at.
func (x *timeIndex) Add(id string, t time.Time) {
	atNano := t.UnixNano()
	if previous, ok := x.at[id]; ok {
		if previous == atNano {
			return
		}
		x.tree.Delete(timeEntry{id: id, at: previous})
	}
	x.tree.ReplaceOrInsert(timeEntry{id: id, at: atNano})
	x.at[id] = atNano
}

// Remove drops id from the index.
//
// Takes id (string) which identifies the task.
func (x *timeIndex) Remove(id string) {
	at, ok := x.at[id]
	if !ok {
		return
	}
	x.tree.Delete(timeEntry{id: id, at: at})
	delete(x.at, id)
}

// Has reports whether id is indexed at t.
//
// Takes id (string) which identifies the task.
// Takes t (time.Time) which is the time to check.
//
// Returns bool which is true when id is indexed at exactly t.
func (x *timeIndex) Has(id string, t time.Time) bool {
	at, ok := x.at[id]
	return ok && at == t.UnixNano()
}

// Size returns the number of indexed IDs.
//
// Returns int which is the count.
func (x *timeIndex) Size() int {
	return len(x.at)
}

// Keys returns every indexed ID in time order.
//
// Returns []string which holds the IDs.
func (x *timeIndex) Keys() []string {
	keys := make([]string, 0, len(x.at))
	x.tree.Ascend(func(entry timeEntry) bool {
		keys = append(keys, entry.id)
		return true
	})
	return keys
}

// KeysThrough returns up to limit IDs indexed at or before through, in time order.
//
// Takes through (time.Time) which is the inclusive upper bound.
// Takes limit (int) which caps the number of IDs; zero or less returns nil.
//
// Returns []string which holds the IDs.
func (x *timeIndex) KeysThrough(through time.Time, limit int) []string {
	return x.KeysThroughWhere(through, limit, nil)
}

// KeysThroughWhere returns up to limit accepted IDs indexed at or before through, in time
// order.
//
// Takes through (time.Time) which is the inclusive upper bound.
// Takes limit (int) which caps the number of IDs; zero or less returns nil.
// Takes accept (func(string) bool) which selects IDs, or is nil to accept all.
//
// Returns []string which holds the IDs.
func (x *timeIndex) KeysThroughWhere(through time.Time, limit int, accept func(string) bool) []string {
	if limit <= 0 {
		return nil
	}
	boundNano := through.UnixNano()
	var keys []string
	x.tree.Ascend(func(entry timeEntry) bool {
		if entry.at > boundNano {
			return false
		}
		if accept == nil || accept(entry.id) {
			keys = append(keys, entry.id)
		}
		return len(keys) < limit
	})
	return keys
}

// KeysAfterThrough returns up to limit IDs after a (time, ID) cursor and at or before
// through, in time order. The cursor stays valid after its entry is removed.
//
// Takes afterID (string) which is the previous page's last ID.
// Takes after (time.Time) which is the previous page's last time.
// Takes started (bool) which is false to start at the beginning, ignoring the cursor.
// Takes through (time.Time) which is the inclusive upper bound.
// Takes limit (int) which caps the number of IDs; zero or less returns nil.
//
// Returns []string which holds the IDs.
func (x *timeIndex) KeysAfterThrough(afterID string, after time.Time, started bool, through time.Time, limit int) []string {
	if limit <= 0 {
		return nil
	}
	boundNano := through.UnixNano()
	keys := make([]string, 0, min(limit, len(x.at)))
	visit := func(entry timeEntry) bool {
		if entry.at > boundNano {
			return false
		}
		keys = append(keys, entry.id)
		return len(keys) < limit
	}
	if !started {
		x.tree.Ascend(visit)
		return keys
	}
	cursor := timeEntry{id: afterID, at: after.UnixNano()}
	x.tree.AscendGreaterOrEqual(cursor, func(entry timeEntry) bool {
		if entry == cursor {
			return true
		}
		return visit(entry)
	})
	return keys
}

// CountBefore returns how many IDs are indexed strictly before t.
//
// Takes t (time.Time) which is the exclusive upper bound.
//
// Returns int which is the count.
func (x *timeIndex) CountBefore(t time.Time) int {
	boundNano := t.UnixNano()
	count := 0
	x.tree.Ascend(func(entry timeEntry) bool {
		if entry.at >= boundNano {
			return false
		}
		count++
		return true
	})
	return count
}
