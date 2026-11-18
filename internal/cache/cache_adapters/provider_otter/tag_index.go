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

package provider_otter

import (
	"maps"
	"slices"
	"sync"
	"sync/atomic"
)

const (
	// linearStringSearchLimit is the longest list of tags or terms searched by scanning; a
	// longer list is put in a set first so a membership test stays constant time.
	linearStringSearchLimit = 16
)

// TagIndex maps tags to cache keys for the in-memory Otter cache. It keeps a reverse
// index so that all keys with a given tag can be found and removed together.
//
// Tag index reads and writes can run concurrently.
//
// For tokenised full-text search, use InvertedIndex instead.
type TagIndex[K comparable] struct {
	// index maps each tag to the set of keys that have that tag.
	index map[string]map[K]struct{}

	// keyToTags maps keys to their tags for quick reverse lookup.
	keyToTags map[K]map[string]struct{}

	// maxTagsPerKey limits tags per key. Zero means unlimited.
	maxTagsPerKey int

	// taggedKeys mirrors the number of keys with tags, so GetTags can answer for a cache
	// that uses no tags without taking the lock every write would otherwise share.
	taggedKeys atomic.Int64

	// mu guards access to the tag index data.
	mu sync.RWMutex
}

// stringLookup answers whether a list holds a string, scanning a short list and using a
// set built once for a long one, so a membership test stays cheap at either size.
type stringLookup struct {
	// set holds the strings of a long list; nil when the list is scanned.
	set map[string]struct{}

	// list is the list being searched.
	list []string
}

// Add replaces the tags linked to key.
//
// Takes key (K) which identifies the item to tag.
// Takes tags ([]string) which lists the tags to link with the key.
func (ti *TagIndex[K]) Add(key K, tags []string) {
	ti.replace(key, ti.prepareTags(tags))
}

// prepareTags cuts tags to the per-key limit and drops repeats, so replace can compare
// them with the key's current tags. It reads no index state, so a write can call it
// before any lock is taken.
//
// Takes tags ([]string) which are the tags of a write.
//
// Returns []string which holds each tag once; it is tags itself when nothing had to
// change.
func (ti *TagIndex[K]) prepareTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	if ti.maxTagsPerKey > 0 && len(tags) > ti.maxTagsPerKey {
		tags = tags[:ti.maxTagsPerKey]
	}
	return uniqueStrings(tags)
}

// replace gives key exactly the prepared tags. Writing the tags a key already has, the
// common case for a key that is rewritten, only takes the read lock, so writes of
// different keys do not queue behind each other.
//
// Takes key (K) which identifies the item to tag.
// Takes tags ([]string) which are tags from prepareTags; empty keeps the key's tags.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) replace(key K, tags []string) {
	if len(tags) == 0 {
		return
	}
	ti.mu.RLock()
	unchanged := ti.hasExactTagsLocked(key, tags)
	ti.mu.RUnlock()
	if unchanged {
		return
	}

	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.replaceLocked(key, tags)
}

// hasExactTagsLocked reports whether key has exactly the given tags.
//
// Must be called with the lock held for reading or writing.
//
// Takes key (K) which identifies the item.
// Takes tags ([]string) which holds each tag once.
//
// Returns bool which is true when the key's tags and tags are the same set.
func (ti *TagIndex[K]) hasExactTagsLocked(key K, tags []string) bool {
	current, ok := ti.keyToTags[key]
	if !ok || len(current) != len(tags) {
		return false
	}
	for _, tag := range tags {
		if _, linked := current[tag]; !linked {
			return false
		}
	}
	return true
}

// replaceLocked gives key exactly the given tags, unlinking only the tags it no longer
// has and linking only the ones it did not have.
//
// Must be called with the write lock held.
//
// Takes key (K) which identifies the item.
// Takes tags ([]string) which holds each tag once.
func (ti *TagIndex[K]) replaceLocked(key K, tags []string) {
	current, ok := ti.keyToTags[key]
	if ok {
		if ti.hasExactTagsLocked(key, tags) {
			return
		}
		wanted := newStringLookup(tags)
		for tag := range current {
			if !wanted.contains(tag) {
				ti.unlinkLocked(tag, key)
				delete(current, tag)
			}
		}
	} else {
		current = make(map[string]struct{}, len(tags))
		ti.keyToTags[key] = current
	}

	for _, tag := range tags {
		if _, linked := current[tag]; linked {
			continue
		}
		keys, exists := ti.index[tag]
		if !exists {
			keys = make(map[K]struct{})
			ti.index[tag] = keys
		}
		keys[key] = struct{}{}
		current[tag] = struct{}{}
	}
	ti.publishSizeLocked()
}

// unlinkLocked removes key from the keys of tag, dropping the tag once no key has it.
//
// Must be called with the write lock held.
//
// Takes tag (string) which is the tag to unlink.
// Takes key (K) which identifies the item.
func (ti *TagIndex[K]) unlinkLocked(tag string, key K) {
	if keys, ok := ti.index[tag]; ok {
		delete(keys, key)
		if len(keys) == 0 {
			delete(ti.index, tag)
		}
	}
}

// Invalidate finds all keys associated with the given tags, removes them from the index
// entirely, and returns the list of keys to be invalidated from the main cache.
//
// Takes tags ([]string) which contains the tags whose associated keys should be
// invalidated.
//
// Returns []K which contains the keys that were invalidated.
func (ti *TagIndex[K]) Invalidate(tags []string) []K {
	keys, _ := ti.invalidate(tags, false)
	return keys
}

// invalidate removes the keys of the given tags from the index, as Invalidate does.
//
// Takes tags ([]string) which contains the tags whose keys are removed.
// Takes keepTags (bool) which asks for the tags each key had, so they can be put back.
//
// Returns []K which contains the keys that were removed.
// Returns map[K][]string which holds the tags each removed key had; nil unless keepTags.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) invalidate(tags []string, keepTags bool) ([]K, map[K][]string) {
	if len(tags) == 0 {
		return nil, nil
	}
	ti.mu.Lock()
	defer ti.mu.Unlock()

	keysToInvalidate := make(map[K]struct{})
	for _, tag := range tags {
		if keys, ok := ti.index[tag]; ok {
			for key := range keys {
				keysToInvalidate[key] = struct{}{}
			}
			delete(ti.index, tag)
		}
	}

	if len(keysToInvalidate) == 0 {
		return nil, nil
	}

	result := slices.Collect(maps.Keys(keysToInvalidate))
	var previousTags map[K][]string
	if keepTags {
		previousTags = make(map[K][]string, len(result))
	}
	for _, key := range result {
		if keepTags {
			previousTags[key] = slices.Collect(maps.Keys(ti.keyToTags[key]))
		}
		ti.removeKeyUnsafe(key)
	}
	return result, previousTags
}

// restoreIfUntagged gives key its tags back unless a write has tagged it since.
//
// Takes key (K) which is the key to restore.
// Takes tags ([]string) which are the tags it had.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) restoreIfUntagged(key K, tags []string) {
	if len(tags) == 0 {
		return
	}
	ti.mu.Lock()
	defer ti.mu.Unlock()
	if _, tagged := ti.keyToTags[key]; tagged {
		return
	}
	ti.replaceLocked(key, tags)
}

// removeKeyUnsafe removes all tag links for a given key.
//
// Must be called with a write lock held.
//
// Takes key (K) which identifies the entry to remove from the index.
func (ti *TagIndex[K]) removeKeyUnsafe(key K) {
	if tags, ok := ti.keyToTags[key]; ok {
		for tag := range tags {
			ti.unlinkLocked(tag, key)
		}
		delete(ti.keyToTags, key)
		ti.publishSizeLocked()
	}
}

// publishSizeLocked records how many keys have tags. Must be called with the write lock
// held, after every change to keyToTags.
func (ti *TagIndex[K]) publishSizeLocked() {
	ti.taggedKeys.Store(int64(len(ti.keyToTags)))
}

// RemoveKey removes a single key from the tag index.
//
// Takes key (K) which identifies the entry to remove.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) RemoveKey(key K) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.removeKeyUnsafe(key)
}

// Get returns all keys associated with the given tag.
//
// Takes tag (string) which is the tag to look up.
//
// Returns map[K]struct{} which is the set of keys with this tag.
// Returns nil if no keys have this tag.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) Get(tag string) map[K]struct{} {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	if keys, ok := ti.index[tag]; ok {
		result := make(map[K]struct{}, len(keys))
		for k := range keys {
			result[k] = struct{}{}
		}
		return result
	}
	return nil
}

// AddSingle links a single key with a single tag, without changing other tags the key may
// have.
//
// Takes tag (string) which is the tag to link.
// Takes key (K) which identifies the item to tag.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) AddSingle(tag string, key K) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	if _, ok := ti.index[tag]; !ok {
		ti.index[tag] = make(map[K]struct{})
	}
	ti.index[tag][key] = struct{}{}

	if _, ok := ti.keyToTags[key]; !ok {
		ti.keyToTags[key] = make(map[string]struct{})
	}
	ti.keyToTags[key][tag] = struct{}{}
	ti.publishSizeLocked()
}

// RemoveSingle removes a single tag from a key, without affecting other tags the key may
// have.
//
// Takes tag (string) which is the tag to disassociate.
// Takes key (K) which identifies the item to untag.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) RemoveSingle(tag string, key K) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.unlinkLocked(tag, key)

	if tags, ok := ti.keyToTags[key]; ok {
		delete(tags, tag)
		if len(tags) == 0 {
			delete(ti.keyToTags, key)
			ti.publishSizeLocked()
		}
	}
}

// Clear removes all tags and keys from the index.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) Clear() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.index = make(map[string]map[K]struct{})
	ti.keyToTags = make(map[K]map[string]struct{})
	ti.publishSizeLocked()
}

// GetTags returns the tags associated with a key.
//
// Takes key (K) which identifies the item to look up.
//
// Returns []string which contains the tags for the key, or nil if the key has no tags.
//
// Safe for concurrent use.
func (ti *TagIndex[K]) GetTags(key K) []string {
	if ti.taggedKeys.Load() == 0 {
		return nil
	}
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	tagSet, ok := ti.keyToTags[key]
	if !ok || len(tagSet) == 0 {
		return nil
	}

	return slices.Collect(maps.Keys(tagSet))
}

// contains reports whether the list holds value.
//
// Takes value (string) which is the string to look for.
//
// Returns bool which is true when the list holds value.
func (l stringLookup) contains(value string) bool {
	if l.set != nil {
		_, ok := l.set[value]
		return ok
	}
	return slices.Contains(l.list, value)
}

// NewTagIndex creates a new empty tag index.
//
// Returns *TagIndex[K] ready for use.
func NewTagIndex[K comparable]() *TagIndex[K] {
	return &TagIndex[K]{
		index:         make(map[string]map[K]struct{}),
		keyToTags:     make(map[K]map[string]struct{}),
		taggedKeys:    atomic.Int64{},
		mu:            sync.RWMutex{},
		maxTagsPerKey: 0,
	}
}

// newTagIndex is an internal alias for backwards compatibility.
//
// Returns *TagIndex[K] which is a new empty tag index.
func newTagIndex[K comparable]() *TagIndex[K] {
	return NewTagIndex[K]()
}

// newStringLookup builds a lookup over list.
//
// Takes list ([]string) which is the list to search.
//
// Returns stringLookup which scans list when it is short and uses a set otherwise.
func newStringLookup(list []string) stringLookup {
	if len(list) <= linearStringSearchLimit {
		return stringLookup{set: nil, list: list}
	}
	set := make(map[string]struct{}, len(list))
	for _, value := range list {
		set[value] = struct{}{}
	}
	return stringLookup{set: set, list: list}
}

// uniqueStrings returns values with repeats removed, keeping the first of each.
//
// Takes values ([]string) which may hold repeats.
//
// Returns []string which is values itself when it holds no repeats, or else a new slice.
func uniqueStrings(values []string) []string {
	if !hasRepeatedString(values) {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

// hasRepeatedString reports whether values holds any string more than once, without
// allocating for a short list.
//
// Takes values ([]string) which is the list to check.
//
// Returns bool which is true when a string appears more than once.
func hasRepeatedString(values []string) bool {
	if len(values) <= linearStringSearchLimit {
		for position := 1; position < len(values); position++ {
			if slices.Contains(values[:position], values[position]) {
				return true
			}
		}
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
