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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayoutCache_NewCache(t *testing.T) {
	cache := newLayoutCache()

	assert.NotNil(t, cache)
	assert.Empty(t, cache.entries)
}

func TestLayoutCache_LookupMiss(t *testing.T) {
	cache := newLayoutCache()
	box := &LayoutBox{Type: BoxBlock}
	input := layoutInput{AvailableWidth: 100, SizingMode: SizingModeNormal}

	result := cache.Lookup(box, input)

	assert.Nil(t, result)
}

func TestLayoutCache_StoreAndLookup(t *testing.T) {
	cache := newLayoutCache()
	box := &LayoutBox{Type: BoxBlock}
	input := layoutInput{AvailableWidth: 100, SizingMode: SizingModeNormal}
	fragment := &Fragment{ContentWidth: 80, ContentHeight: 40}

	cache.Store(box, input, fragment)
	result := cache.Lookup(box, input)

	assert.Same(t, fragment, result)
}

func TestLayoutCache_DifferentWidthMiss(t *testing.T) {
	cache := newLayoutCache()
	box := &LayoutBox{Type: BoxBlock}
	storeInput := layoutInput{AvailableWidth: 100, SizingMode: SizingModeNormal}
	lookupInput := layoutInput{AvailableWidth: 200, SizingMode: SizingModeNormal}
	fragment := &Fragment{ContentWidth: 80, ContentHeight: 40}

	cache.Store(box, storeInput, fragment)
	result := cache.Lookup(box, lookupInput)

	assert.Nil(t, result)
}

func TestLayoutCache_DifferentModeMiss(t *testing.T) {
	cache := newLayoutCache()
	box := &LayoutBox{Type: BoxBlock}
	storeInput := layoutInput{AvailableWidth: 100, SizingMode: SizingModeNormal}
	lookupInput := layoutInput{AvailableWidth: 100, SizingMode: SizingModeMinContent}
	fragment := &Fragment{ContentWidth: 80, ContentHeight: 40}

	cache.Store(box, storeInput, fragment)
	result := cache.Lookup(box, lookupInput)

	assert.Nil(t, result)
}

func TestLayoutCache_NilCacheSafety(t *testing.T) {
	var cache *layoutCache
	box := &LayoutBox{Type: BoxBlock}
	input := layoutInput{AvailableWidth: 100, SizingMode: SizingModeNormal}
	fragment := &Fragment{ContentWidth: 80, ContentHeight: 40}

	assert.NotPanics(t, func() {
		result := cache.Lookup(box, input)
		assert.Nil(t, result)
	})

	assert.NotPanics(t, func() {
		cache.Store(box, input, fragment)
	})
}
