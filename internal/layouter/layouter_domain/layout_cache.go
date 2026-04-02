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

package layouter_domain

// layoutCache stores previously computed layout results so
// that repeated layouts with identical inputs skip
// computation. The cache is scoped to a single
// LayoutBoxTree call.
type layoutCache struct {
	// entries maps cache keys to their cached fragment results.
	entries map[layoutCacheKey]*Fragment
}

// layoutCacheKey identifies a unique layout computation by
// the box being laid out and the constraints passed to it.
type layoutCacheKey struct {
	// box is the pointer to the LayoutBox, used as an
	// identity check. Safe because box trees are not
	// modified during a single layout pass.
	box *LayoutBox

	// availableWidth is the inline-axis available space.
	availableWidth float64

	// availableBlockSize is the block-axis available space
	// used for percentage height resolution. Included in
	// the key because grid re-layout passes may change
	// this value after row heights are computed.
	availableBlockSize float64

	// sizingMode distinguishes normal, min-content, and
	// max-content measurements.
	sizingMode SizingMode
}

// newLayoutCache creates a new empty layout cache.
//
// Returns *layoutCache which is ready to store results.
func newLayoutCache() *layoutCache {
	return &layoutCache{
		entries: make(map[layoutCacheKey]*Fragment),
	}
}

// Lookup retrieves a cached Fragment for the given box and
// constraints, or nil if no cache entry exists.
//
// Takes box (*LayoutBox) which identifies the box.
// Takes input (layoutInput) which carries the constraints.
//
// Returns *Fragment which is the cached result, or nil.
func (cache *layoutCache) Lookup(box *LayoutBox, input layoutInput) *Fragment {
	if cache == nil {
		return nil
	}
	key := layoutCacheKey{
		box:                box,
		availableWidth:     input.AvailableWidth,
		availableBlockSize: input.AvailableBlockSize,
		sizingMode:         input.SizingMode,
	}
	return cache.entries[key]
}

// Store saves a Fragment result for the given box and
// constraints.
//
// Takes box (*LayoutBox) which identifies the box.
// Takes input (layoutInput) which carries the constraints.
// Takes fragment (*Fragment) which is the result to cache.
func (cache *layoutCache) Store(box *LayoutBox, input layoutInput, fragment *Fragment) {
	if cache == nil {
		return
	}
	key := layoutCacheKey{
		box:                box,
		availableWidth:     input.AvailableWidth,
		availableBlockSize: input.AvailableBlockSize,
		sizingMode:         input.SizingMode,
	}
	cache.entries[key] = fragment
}

// Invalidate removes all cached entries for a given box.
//
// Takes box (*LayoutBox) which identifies the box to invalidate.
func (cache *layoutCache) Invalidate(box *LayoutBox) {
	if cache == nil {
		return
	}
	for key := range cache.entries {
		if key.box == box {
			delete(cache.entries, key)
		}
	}
}
