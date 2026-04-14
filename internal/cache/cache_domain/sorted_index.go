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

package cache_domain

import (
	"cmp"
	"slices"
	"sync"

	"github.com/google/btree"
)

const (
	// btreeDegree is the branching factor for B-tree nodes.
	// Degree 32 fits optimally in CPU cache lines (L1/L2).
	btreeDegree = 32

	// smallFilterThreshold is the size below which we
	// convert slice->map for filtering.
	// For small sets, map creation overhead is acceptable for O(1) lookup benefit.
	smallFilterThreshold = 100
)

// treeItem stores a key and its associated value for B-tree storage.
type treeItem[K comparable] struct {
	// key is the map key stored at this node.
	key K

	// value holds the data stored in this tree item.
	value any
}

// Less implements the btree.Item interface for B-tree ordering.
// Items are sorted by value first, then by key for stable sorting.
//
// Takes b (btree.Item) which is the item to compare against.
//
// Returns bool which is true if this item should come before b.
func (a *treeItem[K]) Less(b btree.Item) bool {
	bItem, ok := b.(*treeItem[K])
	if !ok {
		return false
	}

	valueCmp := CompareValues(a.value, bItem.value)
	if valueCmp != 0 {
		return valueCmp < 0
	}

	aKey := any(a.key)
	bKey := any(bItem.key)

	switch ak := aKey.(type) {
	case string:
		if bk, ok := bKey.(string); ok {
			return ak < bk
		}
	case int:
		if bk, ok := bKey.(int); ok {
			return ak < bk
		}
	case int64:
		if bk, ok := bKey.(int64); ok {
			return ak < bk
		}
	}

	return keyTiebreak(&a.key, &bItem.key)
}

// SortedIndex maintains keys in sorted order by a field value using a B-tree.
// Supports numeric, string, and any comparable types.
//
// Thread-safe for concurrent read/write access.
// Uses Google's B-tree implementation for O(log n) operations.
type SortedIndex[K comparable] struct {
	// tree stores items in sorted order with O(log n) lookup time.
	tree *btree.BTree

	// keyToItem maps keys to their tree items for O(1) lookup.
	keyToItem map[K]*treeItem[K]

	// mu guards the index for safe concurrent access.
	mu sync.RWMutex
}

// Add inserts or updates a key with its sortable value. If the key exists, it
// is removed and re-inserted at the correct position.
//
// Takes key (K) which identifies the item.
// Takes value (any) which is the sortable field value.
//
// Safe for concurrent use.
func (idx *SortedIndex[K]) Add(key K, value any) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.AddUnsafe(key, value)
}

// AddUnsafe inserts or updates without acquiring the lock. Caller must hold
// the write lock.
//
// Takes key (K) which identifies the entry to insert or update.
// Takes value (any) which is the data to store for the key.
func (idx *SortedIndex[K]) AddUnsafe(key K, value any) {
	if existingItem, exists := idx.keyToItem[key]; exists {
		idx.tree.Delete(existingItem)
	}

	item := &treeItem[K]{
		key:   key,
		value: value,
	}
	idx.tree.ReplaceOrInsert(item)
	idx.keyToItem[key] = item
}

// Remove deletes a key from the index.
//
// Takes key (K) which identifies the item to remove.
//
// Safe for concurrent use.
func (idx *SortedIndex[K]) Remove(key K) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if item, exists := idx.keyToItem[key]; exists {
		idx.tree.Delete(item)
		delete(idx.keyToItem, key)
	}
}

// Lock acquires the write lock on the sorted index.
//
// Concurrency: acquires the write lock.
func (idx *SortedIndex[K]) Lock() {
	idx.mu.Lock()
}

// Unlock releases the write lock on the sorted index.
//
// Concurrency: releases the write lock.
func (idx *SortedIndex[K]) Unlock() {
	idx.mu.Unlock()
}

// Keys returns all keys in sorted order.
//
// Takes ascending (bool) which specifies sort direction.
//
// Returns []K containing keys in the requested order.
//
// Safe for concurrent use; acquires a read lock for the duration of the call.
func (idx *SortedIndex[K]) Keys(ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make([]K, 0, idx.tree.Len())

	if ascending {
		idx.tree.Ascend(func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			result = append(result, item.key)
			return true
		})
	} else {
		idx.tree.Descend(func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			result = append(result, item.key)
			return true
		})
	}

	return result
}

// KeysFiltered returns keys from a subset, in sorted order.
// Only keys present in the filter set are returned.
//
// Takes filter (map[K]struct{}) which contains keys to include.
// Takes ascending (bool) which specifies sort direction.
//
// Returns []K containing filtered keys in the requested order.
//
// Safe for concurrent use. Uses a read lock during iteration.
func (idx *SortedIndex[K]) KeysFiltered(filter map[K]struct{}, ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make([]K, 0, len(filter))

	if ascending {
		idx.tree.Ascend(func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if _, ok := filter[item.key]; ok {
				result = append(result, item.key)
			}
			return true
		})
	} else {
		idx.tree.Descend(func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if _, ok := filter[item.key]; ok {
				result = append(result, item.key)
			}
			return true
		})
	}

	return result
}

// KeysFilteredSlice returns keys from a subset in sorted order, including only
// keys present in the filter slice. More efficient than KeysFiltered when you
// have a slice of keys.
//
// Takes filter ([]K) which contains keys to include.
// Takes ascending (bool) which specifies sort direction.
//
// Returns []K containing filtered keys in the requested order.
//
// Safe for concurrent use; acquires a read lock during execution.
func (idx *SortedIndex[K]) KeysFilteredSlice(filter []K, ascending bool) []K {
	if len(filter) < smallFilterThreshold {
		filterMap := make(map[K]struct{}, len(filter))
		for _, k := range filter {
			filterMap[k] = struct{}{}
		}
		return idx.KeysFiltered(filterMap, ascending)
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make([]K, 0, len(filter))
	for _, key := range filter {
		if item, exists := idx.keyToItem[key]; exists {
			result = append(result, item.key)
		}
	}

	slices.SortFunc(result, func(a, b K) int {
		aItem := idx.keyToItem[a]
		bItem := idx.keyToItem[b]
		comparison := CompareValues(aItem.value, bItem.value)
		if !ascending {
			comparison = -comparison
		}
		return comparison
	})

	return result
}

// Clear removes all entries from the index.
//
// Safe for concurrent use.
func (idx *SortedIndex[K]) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.tree = btree.New(btreeDegree)
	idx.keyToItem = make(map[K]*treeItem[K])
}

// Size returns the number of entries in the index.
//
// Returns int which is the count of entries.
//
// Safe for concurrent use.
func (idx *SortedIndex[K]) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.tree.Len()
}

// KeysGreaterThan returns keys with values greater than the threshold.
//
// Takes threshold (any) which is the minimum value (exclusive).
// Takes ascending (bool) which specifies sort direction.
//
// Returns []K containing keys in the requested order.
//
// Safe for concurrent use; guarded by a read lock.
//
//nolint:revive // B-tree callback dispatch
func (idx *SortedIndex[K]) KeysGreaterThan(threshold any, ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	pivot := &treeItem[K]{value: threshold}
	result := make([]K, 0)

	if ascending {
		idx.tree.AscendGreaterOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if CompareValues(item.value, threshold) > 0 {
				result = append(result, item.key)
			}
			return true
		})
	} else {
		var temp []K
		idx.tree.AscendGreaterOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if CompareValues(item.value, threshold) > 0 {
				temp = append(temp, item.key)
			}
			return true
		})
		result = make([]K, len(temp))
		for i := range temp {
			result[i] = temp[len(temp)-1-i]
		}
	}

	return result
}

// KeysGreaterThanOrEqual returns keys with values >= the threshold.
//
// Takes threshold (any) which specifies the minimum value for comparison.
// Takes ascending (bool) which determines the sort order of results.
//
// Returns []K which contains the matching keys in the specified order.
//
// Safe for concurrent use; holds a read lock for the duration of the call.
func (idx *SortedIndex[K]) KeysGreaterThanOrEqual(threshold any, ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	pivot := &treeItem[K]{value: threshold}
	result := make([]K, 0)

	if ascending {
		idx.tree.AscendGreaterOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			result = append(result, item.key)
			return true
		})
	} else {
		var temp []K
		idx.tree.AscendGreaterOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			temp = append(temp, item.key)
			return true
		})
		result = make([]K, len(temp))
		for i := range temp {
			result[i] = temp[len(temp)-1-i]
		}
	}

	return result
}

// KeysLessThan returns keys with values less than the threshold.
//
// Takes threshold (any) which specifies the upper bound for comparison.
// Takes ascending (bool) which determines the order of returned keys.
//
// Returns []K which contains the matching keys in the specified order.
//
// Safe for concurrent use; acquires a read lock for the duration of the call.
func (idx *SortedIndex[K]) KeysLessThan(threshold any, ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	pivot := &treeItem[K]{value: threshold}
	result := make([]K, 0)

	if ascending {
		idx.tree.AscendLessThan(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if CompareValues(item.value, threshold) < 0 {
				result = append(result, item.key)
			}
			return true
		})
	} else {
		idx.tree.DescendLessOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if CompareValues(item.value, threshold) < 0 {
				result = append(result, item.key)
			}
			return true
		})
	}

	return result
}

// KeysLessThanOrEqual returns keys with values less than or equal to the
// threshold.
//
// Takes threshold (any) which specifies the upper bound for comparison.
// Takes ascending (bool) which determines the order of returned keys.
//
// Returns []K which contains the matching keys in the specified order.
//
// Safe for concurrent use; holds a read lock for the duration of the call.
//
//nolint:revive // B-tree callback dispatch
func (idx *SortedIndex[K]) KeysLessThanOrEqual(threshold any, ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	pivot := &treeItem[K]{value: threshold}
	result := make([]K, 0)

	if ascending {
		idx.tree.AscendLessThan(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			result = append(result, item.key)
			return true
		})
		idx.tree.AscendGreaterOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			if CompareValues(item.value, threshold) == 0 {
				result = append(result, item.key)
			}
			return CompareValues(item.value, threshold) == 0
		})
	} else {
		idx.tree.DescendLessOrEqual(pivot, func(i btree.Item) bool {
			item, ok := i.(*treeItem[K])
			if !ok {
				return true
			}
			result = append(result, item.key)
			return true
		})
	}

	return result
}

// KeysBetween returns keys with values in the range [minValue, maxValue].
//
// Takes minValue (any) which specifies the lower bound of the range.
// Takes maxValue (any) which specifies the upper bound of the range.
// Takes ascending (bool) which determines the sort order of results.
//
// Returns []K which contains the keys within the specified value range.
//
// Safe for concurrent use; protects access with a read lock.
func (idx *SortedIndex[K]) KeysBetween(minValue, maxValue any, ascending bool) []K {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	if ascending {
		return idx.collectRangeAscending(minValue, maxValue)
	}
	return idx.collectRangeDescending(minValue, maxValue)
}

// collectRangeAscending gathers keys within a range in ascending order.
//
// Takes minValue (any) which sets the lower bound of the range.
// Takes maxValue (any) which sets the upper bound of the range.
//
// Returns []K which holds the keys found within the given range.
//
// The caller must hold a read lock.
func (idx *SortedIndex[K]) collectRangeAscending(minValue, maxValue any) []K {
	result := make([]K, 0)
	idx.tree.Ascend(func(i btree.Item) bool {
		item, ok := i.(*treeItem[K])
		if !ok {
			return true
		}
		return idx.appendIfInRange(item, minValue, maxValue, &result)
	})
	return result
}

// collectRangeDescending gathers keys within a range in descending order.
//
// Takes minValue (any) which sets the lower bound of the range.
// Takes maxValue (any) which sets the upper bound of the range.
//
// Returns []K which contains the keys in descending order.
//
// The caller must hold the read lock.
func (idx *SortedIndex[K]) collectRangeDescending(minValue, maxValue any) []K {
	temp := idx.collectRangeAscending(minValue, maxValue)
	return ReverseSlice(temp)
}

// appendIfInRange checks if an item falls within a range and adds its key to
// the result.
//
// Takes item (*treeItem[K]) which is the tree item to check.
// Takes minValue (any) which is the lower bound of the range (inclusive).
// Takes maxValue (any) which is the upper bound of the range (inclusive).
// Takes result (*[]K) which collects keys of items within the range.
//
// Returns bool which is true to continue iteration, false to stop.
func (*SortedIndex[K]) appendIfInRange(item *treeItem[K], minValue, maxValue any, result *[]K) bool {
	cmpMin := CompareValues(item.value, minValue)
	cmpMax := CompareValues(item.value, maxValue)

	if cmpMin < 0 {
		return true
	}
	if cmpMax > 0 {
		return false
	}

	*result = append(*result, item.key)
	return true
}

// NewSortedIndex creates a new empty sorted index backed by a B-tree.
// The B-tree uses degree 32 for optimal cache performance.
//
// Returns *SortedIndex[K] which is the initialised empty sorted index.
func NewSortedIndex[K comparable]() *SortedIndex[K] {
	return &SortedIndex[K]{
		tree:      btree.New(btreeDegree),
		keyToItem: make(map[K]*treeItem[K]),
	}
}

// ReverseSlice returns a new slice with elements in reverse order.
//
// Takes s ([]K) which is the slice to reverse.
//
// Returns []K which contains the elements in reverse order.
func ReverseSlice[K any](s []K) []K {
	result := make([]K, len(s))
	for i := range s {
		result[i] = s[len(s)-1-i]
	}
	return result
}

// CompareValues compares two values for ordering.
// Supports int, int64, float64, string, and falls back to string comparison.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns int which is negative if a < b, zero if a == b, or positive if
// a > b.
func CompareValues(a, b any) int {
	switch av := a.(type) {
	case int:
		if bv, ok := b.(int); ok {
			return cmp.Compare(av, bv)
		}
	case int64:
		if bv, ok := b.(int64); ok {
			return cmp.Compare(av, bv)
		}
	case float64:
		if bv, ok := b.(float64); ok {
			return cmp.Compare(av, bv)
		}
	case string:
		if bv, ok := b.(string); ok {
			return cmp.Compare(av, bv)
		}
	}

	return cmp.Compare(ToString(a), ToString(b))
}
